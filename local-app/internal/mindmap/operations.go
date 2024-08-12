package mindmap

import (
	"fmt"
	"mindnoscape/local-app/internal/models"
	"sort"
	"strconv"
	"strings"
)

func (mm *MindMapManager) ensureCurrentMindMap() error {
	if mm.CurrentMindMap == nil {
		return fmt.Errorf("no mindmap selected, use 'switch' command to select a mindmap or 'new' to create a new one")
	}
	return nil
}

// findNode is a helper function that finds a node by either index or logical index.
func (mm *MindMapManager) findNode(identifier string, useIndex bool) (*models.Node, error) {
	if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return nil, fmt.Errorf("invalid index: %v", err)
		}
		node := mm.CurrentMindMap.Nodes[index]
		if node == nil {
			return nil, fmt.Errorf("node not found with index: %d", index)
		}
		return node, nil
	} else {
		return mm.findNodeByLogicalIndex(identifier)
	}
}

func (mm *MindMapManager) AddNode(parentIdentifier string, content string, extra map[string]string, useIndex bool, options ...interface{}) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	parentNode, err := mm.findNode(parentIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find parent node: %v", err)
	}

	if parentNode == nil {
		return fmt.Errorf("parent node not found")
	}

	// Default addToHistory to true
	addToHistory := true
	var specificIndex int
	hasSpecificIndex := false

	// Process options
	for _, option := range options {
		switch v := option.(type) {
		case bool:
			addToHistory = v
		case int:
			specificIndex = v
			hasSpecificIndex = true
		}
	}

	var newIndex int
	if hasSpecificIndex {
		newIndex = specificIndex
	} else {
		newIndex = mm.CurrentMindMap.MaxIndex + 1
		mm.CurrentMindMap.MaxIndex = newIndex
	}

	newNode := models.NewNode(mm.CurrentMindMap.MaxIndex+1, content, mm.CurrentMindMap.Root.MindMapID)
	newNode.Extra = extra
	newNode.ParentID = parentNode.Index

	// Assign logical index
	if parentNode == mm.CurrentMindMap.Root {
		newNode.LogicalIndex = fmt.Sprintf("%d", len(parentNode.Children)+1)
	} else {
		newNode.LogicalIndex = fmt.Sprintf("%s.%d", parentNode.LogicalIndex, len(parentNode.Children)+1)
	}

	// Add to storage
	if err := mm.Store.AddNode(mm.CurrentMindMap.Root.Content, mm.CurrentUser, parentNode.Index, newNode.Content, newNode.Extra, newNode.LogicalIndex); err != nil {
		return fmt.Errorf("failed to add node to storage: %v", err)
	}

	// Update in-memory structure
	parentNode.Children = append(parentNode.Children, newNode)
	mm.CurrentMindMap.Nodes[newIndex] = newNode

	// Add to operation history
	if addToHistory {
		op := Operation{
			Type: OpAdd,
			AffectedNode: NodeInfo{
				Index:    newNode.Index,
				ParentID: parentNode.Index,
			},
			NewContent: content,
			NewExtra:   extra,
		}
		mm.addToHistory(op)
	}

	fmt.Printf("Added new node: Index= '%v' Content='%s', LogicalIndex='%s', ParentIndex=%d\n", newNode.Index, newNode.Content, newNode.LogicalIndex, parentNode.Index)

	return nil
}

func (mm *MindMapManager) DeleteNode(identifier string, useIndex bool, options ...bool) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	node, err := mm.findNode(identifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find node to delete: %v", err)
	}

	if node == mm.CurrentMindMap.Root {
		return fmt.Errorf("cannot delete root node")
	}

	// Default addToHistory to true
	addToHistory := true
	// If an option is provided, use it
	if len(options) > 0 {
		addToHistory = options[0]
	}

	deletedTree := []*models.Node{node}
	mm.getSubtreeNodes(node, &deletedTree)

	// Remove from storage
	if err := mm.Store.DeleteNode(mm.CurrentMindMap.Root.Content, mm.CurrentUser, node.Index); err != nil {
		return fmt.Errorf("failed to delete node from storage: %v", err)
	}

	// Find the parent node
	parentNode := mm.CurrentMindMap.Nodes[node.ParentID]
	if parentNode == nil {
		return fmt.Errorf("parent node not found")
	}

	// Update in-memory structure
	for i, child := range parentNode.Children {
		if child == node {
			parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
			break
		}
	}

	// Delete the node and its descendants recursively
	mm.deleteNodeRecursive(node)

	// Update logical indexes
	err = mm.recalculateLogicalIndices(mm.CurrentMindMap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after deletion: %v", err)
	}

	// Add to operation history
	if addToHistory {
		op := Operation{
			Type: OpDelete,
			AffectedNode: NodeInfo{
				Index:    node.Index,
				ParentID: node.ParentID,
			},
			DeletedTree: deletedTree,
		}
		mm.addToHistory(op)
	}

	return nil
}

func (mm *MindMapManager) getSubtreeNodes(node *models.Node, nodes *[]*models.Node) {
	for _, child := range node.Children {
		*nodes = append(*nodes, child)
		mm.getSubtreeNodes(child, nodes)
	}
}

func (mm *MindMapManager) recalculateLogicalIndices(node *models.Node) error {
	var recalculate func(*models.Node, string) error
	recalculate = func(n *models.Node, parentIndex string) error {
		for i, child := range n.Children {
			var newLogicalIndex string
			if parentIndex == "" {
				newLogicalIndex = fmt.Sprintf("%d", i+1)
			} else {
				newLogicalIndex = fmt.Sprintf("%s.%d", parentIndex, i+1)
			}

			if child.LogicalIndex != newLogicalIndex {
				child.LogicalIndex = newLogicalIndex
				err := mm.Store.UpdateNodeOrder(mm.CurrentMindMap.Root.Content, mm.CurrentUser, child.Index, child.LogicalIndex)
				if err != nil {
					return fmt.Errorf("failed to update logical index for node %d: %v", child.Index, err)
				}
			}

			if err := recalculate(child, child.LogicalIndex); err != nil {
				return err
			}
		}
		return nil
	}

	return recalculate(node, "")
}

func (mm *MindMapManager) deleteNodeRecursive(node *models.Node) {
	for _, child := range node.Children {
		mm.deleteNodeRecursive(child)
	}
	delete(mm.CurrentMindMap.Nodes, node.Index)
}

func (mm *MindMapManager) Clear() error {
	if mm.CurrentMindMap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	mindmapName := mm.CurrentMindMap.Root.Content
	err := mm.Store.ClearAllNodes(mindmapName, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to clear nodes: %v", err)
	}

	// Remove the mindmap from the in-memory map
	delete(mm.MindMaps, mindmapName)

	// Set the current mindmap to nil
	mm.CurrentMindMap = nil

	mm.ClearOperationHistory()

	fmt.Printf("Mind map '%s' cleared and removed\n", mindmapName)
	return nil
}

func (mm *MindMapManager) ModifyNode(identifier string, content string, extra map[string]string, useIndex bool, options ...bool) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	node, err := mm.findNode(identifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find node to modify: %v", err)
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	// Default addToHistory to true
	addToHistory := true
	// If an option is provided, use it
	if len(options) > 0 {
		addToHistory = options[0]
	}

	oldContent := node.Content
	oldExtra := make(map[string]string)
	for k, v := range node.Extra {
		oldExtra[k] = v
	}

	if content != "" {
		node.Content = content
	}
	for k, v := range extra {
		if v == "" {
			delete(node.Extra, k)
		} else {
			node.Extra[k] = v
		}
	}

	// Update in storage
	if err := mm.Store.UpdateNode(mm.CurrentMindMap.Root.Content, mm.CurrentUser, node.Index, node.Content, node.Extra, node.LogicalIndex); err != nil {
		return fmt.Errorf("failed to update node in storage: %v", err)
	}

	// Add to operation history
	if addToHistory {
		op := Operation{
			Type: OpModify,
			AffectedNode: NodeInfo{
				Index: node.Index,
			},
			OldContent: oldContent,
			NewContent: content,
			OldExtra:   oldExtra,
			NewExtra:   extra,
		}
		mm.addToHistory(op)
	}

	return nil
}

func (mm *MindMapManager) MoveNode(sourceIdentifier, targetIdentifier string, useIndex bool, options ...bool) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	sourceNode, err := mm.findNode(sourceIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find source node: %v", err)
	}

	targetNode, err := mm.findNode(targetIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find target node: %v", err)
	}

	if sourceNode == mm.CurrentMindMap.Root {
		return fmt.Errorf("cannot move root node")
	}

	oldParentNode := mm.CurrentMindMap.Nodes[sourceNode.ParentID]
	if oldParentNode == nil {
		return fmt.Errorf("old parent node not found")
	}

	// Default addToHistory to true
	addToHistory := true
	// If an option is provided, use it
	if len(options) > 0 {
		addToHistory = options[0]
	}

	oldParentID := sourceNode.ParentID

	// Remove from old parent
	for i, child := range oldParentNode.Children {
		if child == sourceNode {
			oldParentNode.Children = append(oldParentNode.Children[:i], oldParentNode.Children[i+1:]...)
			break
		}
	}

	// Add to new parent
	targetNode.Children = append(targetNode.Children, sourceNode)
	sourceNode.ParentID = targetNode.Index

	// Update in storage
	if err := mm.Store.MoveNode(mm.CurrentMindMap.Root.Content, mm.CurrentUser, sourceNode.Index, targetNode.Index); err != nil {
		return fmt.Errorf("failed to move node in storage: %v", err)
	}

	// Update logical indexes starting from the root
	err = mm.recalculateLogicalIndices(mm.CurrentMindMap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after move: %v", err)
	}

	// Add to operation history
	if addToHistory {
		op := Operation{
			Type: OpMove,
			AffectedNode: NodeInfo{
				Index: sourceNode.Index,
			},
			OldParentID: oldParentID,
			NewParentID: targetNode.Index,
		}
		mm.addToHistory(op)
	}

	return nil
}

func (mm *MindMapManager) InsertNode(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	sourceNode, err := mm.findNode(sourceIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find source node: %v", err)
	}

	targetNode, err := mm.findNode(targetIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find target node: %v", err)
	}

	if sourceNode == mm.CurrentMindMap.Root {
		return fmt.Errorf("cannot insert root node")
	}

	if targetNode == mm.CurrentMindMap.Root {
		return fmt.Errorf("cannot insert before root node")
	}

	oldParentID := sourceNode.ParentID

	// Find the parent of the target node
	targetParent := mm.CurrentMindMap.Nodes[targetNode.ParentID]
	if targetParent == nil {
		return fmt.Errorf("target parent node not found")
	}

	// Remove source node from its current parent
	sourceParent := mm.CurrentMindMap.Nodes[sourceNode.ParentID]
	if sourceParent == nil {
		return fmt.Errorf("source parent node not found")
	}
	for i, child := range sourceParent.Children {
		if child == sourceNode {
			sourceParent.Children = append(sourceParent.Children[:i], sourceParent.Children[i+1:]...)
			break
		}
	}

	// Insert source node before target node
	for i, child := range targetParent.Children {
		if child == targetNode {
			targetParent.Children = append(targetParent.Children[:i], append([]*models.Node{sourceNode}, targetParent.Children[i:]...)...)
			break
		}
	}

	// Update parent ID of source node
	sourceNode.ParentID = targetParent.Index

	// Update in storage
	if err := mm.Store.MoveNode(mm.CurrentMindMap.Root.Content, mm.CurrentUser, sourceNode.Index, targetParent.Index); err != nil {
		return fmt.Errorf("failed to move node in storage: %v", err)
	}

	// Update logical indexes starting from the root
	err = mm.recalculateLogicalIndices(mm.CurrentMindMap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after insertion: %v", err)
	}

	// Add to operation history
	op := Operation{
		Type: OpInsert,
		AffectedNode: NodeInfo{
			Index: sourceNode.Index,
		},
		OldParentID: oldParentID,
		NewParentID: targetNode.ParentID,
	}
	mm.addToHistory(op)

	return nil
}

// Sort sorts the children of the specified node based on content or an extra field.
// If no node is specified, it sorts all nodes in the mindmap.
// The sorting can be done in reverse order if the reverse flag is set.
// If an extra field is specified, it sorts based on that field; otherwise, it sorts by content.
func (mm *MindMapManager) Sort(identifier string, field string, reverse bool, useIndex bool) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	var node *models.Node
	var err error

	if identifier == "" {
		node = mm.CurrentMindMap.Root
	} else {
		node, err = mm.findNode(identifier, useIndex)
		if err != nil {
			return fmt.Errorf("failed to find node to sort: %v", err)
		}
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	fmt.Printf("Sorting children of node: %s (LogicalIndex: %s)\n", node.Content, node.LogicalIndex)

	mm.sortNodeChildrenRecursively(node, field, reverse)
	err = mm.recalculateLogicalIndices(node)
	if err != nil {
		return fmt.Errorf("failed to recalculate logical indices after sorting: %v", err)
	}

	// Update the database to reflect the new order
	err = mm.updateNodeOrderInDB(mm.CurrentMindMap.Root.Content, node)
	if err != nil {
		return fmt.Errorf("failed to update node order in database: %v", err)
	}

	return nil
}

func (mm *MindMapManager) sortNodeChildrenRecursively(node *models.Node, field string, reverse bool) {
	sort.Slice(node.Children, func(i, j int) bool {
		var vi, vj string
		if field == "" {
			vi, vj = node.Children[i].Content, node.Children[j].Content
		} else {
			vi = node.Children[i].Extra[field]
			vj = node.Children[j].Extra[field]
		}

		// If the field doesn't exist, fall back to Content
		if vi == "" && vj == "" {
			vi, vj = node.Children[i].Content, node.Children[j].Content
		}

		// Try to compare as numbers if possible
		ni, errI := strconv.ParseFloat(vi, 64)
		nj, errJ := strconv.ParseFloat(vj, 64)
		if errI == nil && errJ == nil {
			if reverse {
				return ni > nj
			}
			return ni < nj
		}

		// Fall back to string comparison
		if reverse {
			return vi > vj
		}
		return vi < vj
	})

	// Recursively sort children of children
	for _, child := range node.Children {
		mm.sortNodeChildrenRecursively(child, field, reverse)
	}
}

func (mm *MindMapManager) updateNodeOrderInDB(mindmapName string, node *models.Node) error {
	var updateNode func(*models.Node) error
	updateNode = func(n *models.Node) error {
		for _, child := range n.Children {
			err := mm.Store.UpdateNodeOrder(mindmapName, mm.CurrentUser, child.Index, child.LogicalIndex)
			if err != nil {
				return err
			}

			// Recursively update the order of grandchildren
			err = updateNode(child)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return updateNode(node)
}

func (mm *MindMapManager) FindNodes(query string) []*models.Node {
	if mm.CurrentMindMap == nil {
		return nil
	}

	var matches []*models.Node
	for _, node := range mm.CurrentMindMap.Nodes {
		if strings.Contains(strings.ToLower(node.Content), strings.ToLower(query)) {
			matches = append(matches, node)
			continue
		}
		for _, value := range node.Extra {
			if strings.Contains(strings.ToLower(value), strings.ToLower(query)) {
				matches = append(matches, node)
				break
			}
		}
	}
	return matches
}
