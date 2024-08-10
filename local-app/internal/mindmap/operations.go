package mindmap

import (
	"fmt"
	"mindnoscape/local-app/internal/models"
	"sort"
	"strconv"
	"strings"
)

func (mm *MindMapManager) ListMindMaps() []string {
	var names []string
	for name := range mm.MindMaps {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (mm *MindMapManager) ensureCurrentMindMap() error {
	if mm.CurrentMindMap == nil {
		return fmt.Errorf("no mindmap selected, use 'switch' command to select a mindmap")
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

func (mm *MindMapManager) AddNode(parentIdentifier string, content string, extra map[string]string, useIndex bool) error {
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

	newIndex := mm.CurrentMindMap.MaxIndex + 1
	mm.CurrentMindMap.MaxIndex = newIndex

	newNode := models.NewNode(newIndex, content)
	newNode.Extra = extra
	newNode.ParentID = parentNode.Index

	// Assign logical index
	if parentNode == mm.CurrentMindMap.Root {
		newNode.LogicalIndex = fmt.Sprintf("%d", len(parentNode.Children)+1)
	} else {
		newNode.LogicalIndex = fmt.Sprintf("%s.%d", parentNode.LogicalIndex, len(parentNode.Children)+1)
	}

	// Add to storage
	if err := mm.Store.AddNode(mm.CurrentMindMap.Root.Content, parentNode.Index, newNode.Content, newNode.Extra, newNode.LogicalIndex); err != nil {
		return fmt.Errorf("failed to add node to storage: %v", err)
	}

	// Update in-memory structure
	parentNode.Children = append(parentNode.Children, newNode)
	mm.CurrentMindMap.Nodes[newIndex] = newNode

	fmt.Printf("Added new node: Index= '%v' Content='%s', LogicalIndex='%s', ParentIndex=%d\n", newNode.Index, newNode.Content, newNode.LogicalIndex, parentNode.Index)

	return nil
}

func (mm *MindMapManager) DeleteNode(identifier string, useIndex bool) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	nodeToDelete, err := mm.findNode(identifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find node to delete: %v", err)
	}

	if nodeToDelete == mm.CurrentMindMap.Root {
		return fmt.Errorf("cannot delete root node")
	}

	// Remove from storage
	if err := mm.Store.DeleteNode(mm.CurrentMindMap.Root.Content, nodeToDelete.Index); err != nil {
		return fmt.Errorf("failed to delete node from storage: %v", err)
	}

	// Find the parent node
	parentNode := mm.CurrentMindMap.Nodes[nodeToDelete.ParentID]
	if parentNode == nil {
		return fmt.Errorf("parent node not found")
	}

	// Update in-memory structure
	for i, child := range parentNode.Children {
		if child == nodeToDelete {
			parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
			break
		}
	}

	// Delete the node and its descendants recursively
	mm.deleteNodeRecursive(nodeToDelete)

	// Update logical indexes
	err = mm.updateLogicalIndexes(mm.CurrentMindMap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after deletion: %v", err)
	}

	return nil
}

func (mm *MindMapManager) updateLogicalIndexes(node *models.Node) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	var assign func(*models.Node, string) error
	assign = func(n *models.Node, prefix string) error {
		for i, child := range n.Children {
			if n == mm.CurrentMindMap.Root {
				child.LogicalIndex = fmt.Sprintf("%d", i+1)
			} else {
				child.LogicalIndex = fmt.Sprintf("%s.%d", prefix, i+1)
			}
			err := mm.Store.UpdateNodeOrder(mm.CurrentMindMap.Root.Content, child.Index, child.LogicalIndex)
			if err != nil {
				return fmt.Errorf("failed to update logical index for node %d: %v", child.Index, err)
			}
			if err := assign(child, child.LogicalIndex); err != nil {
				return err
			}
		}
		return nil
	}

	return assign(node, "")
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
	err := mm.Store.ClearAllNodes(mindmapName)
	if err != nil {
		return fmt.Errorf("failed to clear nodes: %v", err)
	}

	// Remove the mindmap from the in-memory map
	delete(mm.MindMaps, mindmapName)

	// Set the current mindmap to nil
	mm.CurrentMindMap = nil

	fmt.Printf("Mind map '%s' cleared and removed\n", mindmapName)
	return nil
}

func (mm *MindMapManager) ModifyNode(identifier string, content string, extra map[string]string, useIndex bool) error {
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
	if err := mm.Store.UpdateNode(mm.CurrentMindMap.Root.Content, node.Index, node.Content, node.Extra, node.LogicalIndex); err != nil {
		return fmt.Errorf("failed to update node in storage: %v", err)
	}

	return nil
}

func (mm *MindMapManager) MoveNode(sourceIdentifier, targetIdentifier string, useIndex bool) error {
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
	if err := mm.Store.MoveNode(mm.CurrentMindMap.Root.Content, sourceNode.Index, targetNode.Index); err != nil {
		return fmt.Errorf("failed to move node in storage: %v", err)
	}

	// Update logical indexes starting from the root
	err = mm.updateLogicalIndexes(mm.CurrentMindMap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after move: %v", err)
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
	if err := mm.Store.MoveNode(mm.CurrentMindMap.Root.Content, sourceNode.Index, targetParent.Index); err != nil {
		return fmt.Errorf("failed to move node in storage: %v", err)
	}

	// Update logical indexes starting from the root
	err = mm.updateLogicalIndexes(mm.CurrentMindMap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after insertion: %v", err)
	}

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
	mm.assignLogicalIndexForSubtree(node)

	// Update the database to reflect the new order
	err = mm.updateNodeOrderInDB(mm.CurrentMindMap.Root.Content, node)
	if err != nil {
		return fmt.Errorf("failed to update node order in database: %v", err)
	}

	return nil
}

func (mm *MindMapManager) assignLogicalIndexForSubtree(node *models.Node) {
	var assign func(*models.Node, string)
	assign = func(n *models.Node, prefix string) {
		for i, child := range n.Children {
			if n == mm.CurrentMindMap.Root {
				child.LogicalIndex = fmt.Sprintf("%d", i+1)
			} else {
				child.LogicalIndex = fmt.Sprintf("%s.%d", prefix, i+1)
			}
			assign(child, child.LogicalIndex)
		}
	}

	if node == mm.CurrentMindMap.Root {
		assign(node, "")
	} else {
		assign(node, node.LogicalIndex)
	}
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
			err := mm.Store.UpdateNodeOrder(mindmapName, child.Index, child.LogicalIndex)
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
