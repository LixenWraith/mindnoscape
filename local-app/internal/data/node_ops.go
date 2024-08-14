package data

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mindnoscape/local-app/internal/models"
)

type NodeOperations interface {
	AddNode(parentIdentifier string, content string, extra map[string]string, useIndex bool) error
	DeleteNode(identifier string, useIndex bool) error
	ModifyNode(identifier string, content string, extra map[string]string, useIndex bool) error
	MoveNode(sourceIdentifier, targetIdentifier string, useIndex bool) error
	FindNode(query string, showIndex bool) ([]string, error)
	Sort(identifier string, field string, reverse bool, useIndex bool) error
}

type NodeManager struct {
	mm *MindmapManager
}

func NewNodeManager(mm *MindmapManager) *NodeManager {
	return &NodeManager{mm: mm}
}

func (nm *NodeManager) AddNode(parentIdentifier string, content string, extra map[string]string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current data: %w", err)
	}

	if nm.mm.CurrentMindmap == nil {
		return fmt.Errorf("no data selected")
	}

	if nm.mm.CurrentMindmap.Nodes == nil {
		nm.mm.CurrentMindmap.Nodes = make(map[int]*models.Node)
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
	if err := nm.mm.Store.AddNode(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, parentNode.Index, newNode.Content, newNode.Extra, newNode.LogicalIndex); err != nil {
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
	nm.mm.HistoryManager.AddToHistory(op)

	return nil
}

func (nm *NodeManager) DeleteNode(identifier string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current data: %w", err)
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
		if err := nm.mm.Store.DeleteNode(nm.mm.CurrentMindmap.Root.Content, nm.mm.CurrentUser, n.Index); err != nil {
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
	err = nm.recalculateLogicalIndices(nm.mm.CurrentMindmap.Root)
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
	nm.mm.HistoryManager.AddToHistory(op)

	return nil
}

func (nm *NodeManager) ModifyNode(identifier string, content string, extra map[string]string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current data: %w", err)
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
	if err := nm.mm.Store.ModifyNode(nm.mm.CurrentMindmap.Root.Content, nm.mm.CurrentUser, node.Index, node.Content, node.Extra, node.LogicalIndex); err != nil {
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
	nm.mm.HistoryManager.AddToHistory(op)

	return nil
}

func (nm *NodeManager) MoveNode(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current data: %w", err)
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
	if err := nm.mm.Store.MoveNode(nm.mm.CurrentMindmap.Root.Content, nm.mm.CurrentUser, sourceNode.Index, targetNode.Index); err != nil {
		return fmt.Errorf("failed to move node in storage: %w", err)
	}

	// Update logical indexes starting from the root
	err = nm.recalculateLogicalIndices(nm.mm.CurrentMindmap.Root)
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
	nm.mm.HistoryManager.AddToHistory(op)

	return nil
}

func (nm *NodeManager) FindNode(query string, showIndex bool) ([]string, error) {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return nil, fmt.Errorf("failed to ensure current data: %w", err)
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

func (nm *NodeManager) SortNode(identifier string, field string, reverse bool, useIndex bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current data: %w", err)
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

	// Sort children of the node in memory
	nm.sortNodeChildrenRecursively(node, field, reverse)
	err = nm.recalculateLogicalIndices(node)
	if err != nil {
		return fmt.Errorf("failed to recalculate logical indices after sorting: %w", err)
	}

	// Update the database to reflect the new order
	err = nm.updateNodeOrderRecursive(node)
	if err != nil {
		return fmt.Errorf("failed to update node order in database: %w", err)
	}

	return nil
}

func (nm *NodeManager) sortNodeChildrenRecursively(node *models.Node, field string, reverse bool) {
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
		nm.sortNodeChildrenRecursively(child, field, reverse)
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

func (nm *NodeManager) recalculateLogicalIndices(node *models.Node) error {
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
				err := nm.updateNodeOrder(child.Index, child.LogicalIndex)
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

	return recalculate(node, "")
}

func (nm *NodeManager) updateNodeOrder(nodeID int, logicalIndex string) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return err
	}

	mindmapName := nm.mm.CurrentMindmap.Name
	err := nm.mm.Store.UpdateNodeOrder(mindmapName, nm.mm.CurrentUser, nodeID, logicalIndex)
	if err != nil {
		return fmt.Errorf("failed to update node order in database: %w", err)
	}

	return nil
}

// This function replaces the previous updateNodeOrderInDB
func (nm *NodeManager) updateNodeOrderRecursive(node *models.Node) error {
	if err := nm.updateNodeOrder(node.Index, node.LogicalIndex); err != nil {
		return fmt.Errorf("failed to update node order for node %d: %w", node.Index, err)
	}

	for _, child := range node.Children {
		if err := nm.updateNodeOrderRecursive(child); err != nil {
			return err
		}
	}

	return nil
}
