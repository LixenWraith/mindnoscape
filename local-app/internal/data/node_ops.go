package data

import (
	"fmt"
	"mindnoscape/local-app/internal/storage"
	"sort"
	"strconv"
	"strings"

	"mindnoscape/local-app/internal/models"
)

type NodeOperations interface {
	NodeAdd(parentIdentifier string, content string, extra map[string]string, useIndex bool) error
	NodeDelete(identifier string, useIndex bool) error
	NodeModify(identifier string, content string, extra map[string]string, useIndex bool) error
	NodeMove(sourceIdentifier, targetIdentifier string, useIndex bool) error
	NodeFind(query string, showIndex bool) ([]string, error)
	NodeSort(identifier string, field string, reverse bool, useIndex bool) error
}

type NodeManager struct {
	nodeStore      storage.NodeStore
	mm             *MindmapManager
	historyManager *HistoryManager
}

func NewNodeManager(mm *MindmapManager) *NodeManager {
	nm := &NodeManager{
		nodeStore: mm.NodeStore,
		mm:        mm,
	}
	nm.historyManager = NewHistoryManager(nm)
	return nm
}

func (nm *NodeManager) NodeAdd(parentIdentifier string, content string, extra map[string]string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	parentNode, err := nm.findNodeByIndex(parentIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find parent node: %w", err)
	}

	if parentNode == nil {
		return fmt.Errorf("parent node not found")
	}

	newIndex := nm.mm.CurrentMindmap.MaxIndex + 1
	nm.mm.CurrentMindmap.MaxIndex = newIndex

	newNode := models.NewNode(newIndex, content, nm.mm.CurrentMindmap.ID)
	newNode.Extra = extra
	newNode.ParentID = parentNode.Index

	// Assign logical index
	if parentNode.Index == 0 {
		newNode.LogicalIndex = fmt.Sprintf("%d", len(parentNode.Children)+1)
	} else {
		newNode.LogicalIndex = fmt.Sprintf("%s.%d", parentNode.LogicalIndex, len(parentNode.Children)+1)
	}

	// Add to storage
	if err := nm.nodeStore.NodeAdd(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, parentNode.Index, newNode.Content, newNode.Extra, newNode.LogicalIndex); err != nil {
		return fmt.Errorf("failed to add node to storage: %w", err)
	}

	// Update in-memory structure
	parentNode.Children = append(parentNode.Children, newNode)
	nm.mm.CurrentMindmap.Nodes[newIndex] = newNode

	// Add to operation history
	op := Operation{
		Type: OpAdd,
		AffectedNode: NodeInfo{
			Index:    newNode.Index,
			ParentID: parentNode.Index,
		},
		NewContent: content,
		NewExtra:   extra,
	}
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeDelete(identifier string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	node, err := nm.findNodeByIndex(identifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find node to delete: %w", err)
	}

	if node == nm.mm.CurrentMindmap.Root {
		return fmt.Errorf("cannot delete root node")
	}

	oldParentNode := nm.mm.CurrentMindmap.Nodes[node.ParentID]
	if oldParentNode == nil {
		return fmt.Errorf("old parent node not found")
	}

	deletedTree := []*models.Node{node}
	nm.getSubtreeNodes(node, &deletedTree)

	// Remove from storage
	for _, n := range deletedTree {
		if err := nm.nodeStore.NodeDelete(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, n.Index); err != nil {
			return fmt.Errorf("failed to delete node from storage: %w", err)
		}
	}

	// Update in-memory structure
	for i, child := range oldParentNode.Children {
		if child == node {
			oldParentNode.Children = append(oldParentNode.Children[:i], oldParentNode.Children[i+1:]...)
			break
		}
	}

	// Delete the node and its descendants recursively
	nm.deleteNodeRecursive(node)

	// Update logical indexes
	err = nm.updateSubtreeLogicalIndices(nm.mm.CurrentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after deletion: %w", err)
	}

	// Add to operation history
	op := Operation{
		Type: OpDelete,
		AffectedNode: NodeInfo{
			Index:    node.Index,
			ParentID: node.ParentID,
		},
		DeletedTree: deletedTree,
	}
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeModify(identifier string, content string, extra map[string]string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	node, err := nm.findNodeByIndex(identifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find node to modify: %w", err)
	}

	if node == nil {
		return fmt.Errorf("node not found")
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
	if err := nm.nodeStore.NodeModify(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, node.Index, node.Content, node.Extra, node.LogicalIndex); err != nil {
		return fmt.Errorf("failed to update node in storage: %w", err)
	}

	// Add to operation history
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
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeMove(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	sourceNode, err := nm.findNodeByIndex(sourceIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find source node: %w", err)
	}

	targetNode, err := nm.findNodeByIndex(targetIdentifier, useIndex)
	if err != nil {
		return fmt.Errorf("failed to find target node: %w", err)
	}

	if sourceNode == nm.mm.CurrentMindmap.Root {
		return fmt.Errorf("cannot move root node")
	}

	oldParentNode := nm.mm.CurrentMindmap.Nodes[sourceNode.ParentID]
	if oldParentNode == nil {
		return fmt.Errorf("old parent node not found")
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
	if err := nm.nodeStore.NodeMove(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, sourceNode.Index, targetNode.Index); err != nil {
		return fmt.Errorf("failed to move node in storage: %w", err)
	}

	// Update logical indexes starting from the root
	err = nm.updateSubtreeLogicalIndices(nm.mm.CurrentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update logical indexes after move: %w", err)
	}

	// Add to operation history
	op := Operation{
		Type: OpMove,
		AffectedNode: NodeInfo{
			Index: sourceNode.Index,
		},
		OldParentID: oldParentID,
		NewParentID: targetNode.Index,
	}
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeFind(query string, showIndex bool) ([]string, error) {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return nil, fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	matches := []*models.Node{}
	for _, node := range nm.mm.CurrentMindmap.Nodes {
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

	if len(matches) == 0 {
		return []string{"No matches found."}, nil
	}

	output := []string{fmt.Sprintf("Found %d matches:", len(matches))}
	visualOutput, err := nm.visualizeNode(matches, showIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to visualize nodes: %w", err)
	}
	output = append(output, visualOutput...)

	return output, nil
}

func (nm *NodeManager) NodeSort(identifier string, field string, reverse bool, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	var node *models.Node
	var err error

	if identifier == "" {
		node = nm.mm.CurrentMindmap.Root
	} else {
		node, err = nm.findNodeByIndex(identifier, useIndex)
		if err != nil {
			return fmt.Errorf("failed to find node to sort: %w", err)
		}
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	// Sort the entire subtree
	nm.sortNodeSubtreeRecursively(node, field, reverse)

	// Update logical indices in memory and database
	err = nm.updateSubtreeLogicalIndices(node)
	if err != nil {
		return fmt.Errorf("failed to update logical indices and persist changes after sorting: %w", err)
	}

	return nil
}

func (nm *NodeManager) NodeUndo() error {
	return nm.historyManager.Undo()
}

func (nm *NodeManager) NodeRedo() error {
	return nm.historyManager.Redo()
}

func (nm *NodeManager) NodeReset() {
	nm.historyManager.HistoryReset()
}

func (nm *NodeManager) sortNodeSubtreeRecursively(node *models.Node, field string, reverse bool) {
	// Sort the children of this node
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
		nm.sortNodeSubtreeRecursively(child, field, reverse)
	}
}

func (nm *NodeManager) visualizeNode(nodes []*models.Node, showIndex bool) ([]string, error) {
	var output []string

	for _, node := range nodes {
		var line strings.Builder

		line.WriteString(fmt.Sprintf("%s%s%s ", string(ColorYellow), node.LogicalIndex, string(ColorDefault)))
		line.WriteString(node.Content)
		if showIndex {
			line.WriteString(fmt.Sprintf(" %s[%d]%s", string(ColorOrange), node.Index, string(ColorDefault)))
		}

		// Add extra fields
		if len(node.Extra) > 0 {
			var extraFields []string
			for k, v := range node.Extra {
				extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
			}
			sort.Strings(extraFields) // Sort extra fields for consistent output
			line.WriteString(" " + strings.Join(extraFields, ", "))
		}

		output = append(output, line.String())
	}

	return output, nil
}

// Helper functions that were previously in the MindmapManager but are now part of NodeManager

func (nm *NodeManager) ensureCurrentMindmap() error {
	if nm.mm.CurrentMindmap == nil {
		return fmt.Errorf("no data selected")
	}
	return nil
}

func (nm *NodeManager) findNodeByIndex(identifier string, useIndex bool) (*models.Node, error) {
	if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return nil, fmt.Errorf("invalid index: %w", err)
		}
		return nm.mm.CurrentMindmap.Nodes[index], nil
	}
	return nm.findNodeByLogicalIndex(identifier)
}

func (nm *NodeManager) findNodeByLogicalIndex(logicalIndex string) (*models.Node, error) {
	if logicalIndex == "0" {
		return nm.mm.CurrentMindmap.Root, nil
	}

	parts := strings.Split(logicalIndex, ".")
	currentNode := nm.mm.CurrentMindmap.Root

	for _, part := range parts {
		index, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid logical index part '%s': %w", part, err)
		}
		if index < 1 || index > len(currentNode.Children) {
			return nil, fmt.Errorf("invalid logical index: part %s is out of range", part)
		}
		currentNode = currentNode.Children[index-1]
	}

	return currentNode, nil
}

func (nm *NodeManager) getSubtreeNodes(node *models.Node, nodes *[]*models.Node) {
	for _, child := range node.Children {
		*nodes = append(*nodes, child)
		nm.getSubtreeNodes(child, nodes)
	}
}

func (nm *NodeManager) deleteNodeRecursive(node *models.Node) {
	for _, child := range node.Children {
		nm.deleteNodeRecursive(child)
	}
	delete(nm.mm.CurrentMindmap.Nodes, node.Index)
}

func (nm *NodeManager) updateSubtreeLogicalIndices(node *models.Node) error {
	fmt.Println(node)
	var recalculate func(*models.Node, string) error
	recalculate = func(n *models.Node, parentIndex string) error {
		for i, child := range n.Children {
			var newLogicalIndex string
			if parentIndex == "0" {
				newLogicalIndex = fmt.Sprintf("%d", i+1)
			} else {
				newLogicalIndex = fmt.Sprintf("%s.%d", parentIndex, i+1)
			}

			if child.LogicalIndex != newLogicalIndex {
				child.LogicalIndex = newLogicalIndex
				err := nm.updateNodeLogicalIndex(child.Index, child.LogicalIndex)
				if err != nil {
					return fmt.Errorf("failed to update logical index for node %d: %w", child.Index, err)
				}
			}

			if err := recalculate(child, child.LogicalIndex); err != nil {
				return err
			}
		}
		return nil
	}

	return recalculate(node, node.LogicalIndex)
}

func (nm *NodeManager) updateNodeLogicalIndex(nodeID int, logicalIndex string) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return err
	}

	mindmapName := nm.mm.CurrentMindmap.Name
	err := nm.nodeStore.NodeOrderUpdate(mindmapName, nm.mm.CurrentUser, nodeID, logicalIndex)
	if err != nil {
		return fmt.Errorf("failed to update node order in database: %w", err)
	}

	return nil
}
