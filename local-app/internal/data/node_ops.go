package data

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mindnoscape/local-app/internal/models"
	"mindnoscape/local-app/internal/storage"
)

type NodeOperations interface {
	NodeAdd(parentIdentifier string, content string, extra map[string]string, useID bool) error
	NodeDelete(identifier string, useID bool) error
	NodeUpdate(identifier string, content string, extra map[string]string, useID bool) error
	NodeMove(sourceIdentifier, targetIdentifier string, useID bool) error
	NodeFind(query string, showIndex bool) ([]string, error)
	NodeSort(identifier string, field string, reverse bool, useID bool) error
	NodeGet(ID int) (*models.Node, error)
	NodeGetAll() ([]*models.Node, error)
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

func (nm *NodeManager) NodeAdd(parentIdentifier string, content string, extra map[string]string, useID bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	if err := nm.checkOwnership(); err != nil {
		return err
	}

	parentNode, err := nm.findNodeByID(parentIdentifier, useID)
	if err != nil {
		return fmt.Errorf("failed to find parent node: %w", err)
	}

	if parentNode == nil {
		return fmt.Errorf("parent node not found")
	}

	newNode := models.NewNode(0, content, nm.mm.CurrentMindmap.ID)
	newNode.Extra = extra
	newNode.ParentID = parentNode.ID

	// Assign index
	if parentNode.ID == 0 {
		// For root's children, use the number of existing children + 1
		newNode.Index = fmt.Sprintf("%d", len(parentNode.Children)+1)
	} else {
		// Find the highest index among siblings
		highestIndex := 0
		for _, sibling := range parentNode.Children {
			parts := strings.Split(sibling.Index, ".")
			if len(parts) > 0 {
				if lastPart, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					if lastPart > highestIndex {
						highestIndex = lastPart
					}
				}
			}
		}
		// Increment the highest index for the new node
		newNode.Index = fmt.Sprintf("%s.%d", parentNode.Index, highestIndex+1)
	}

	// Add to storage
	newID, err := nm.nodeStore.NodeAdd(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, parentNode.ID, newNode.Content, newNode.Extra, newNode.Index)
	if err != nil {
		return fmt.Errorf("failed to add node to storage: %w", err)
	}

	newNode.ID = newID

	// Update in-memory structure
	parentNode.Children = append(parentNode.Children, newNode)
	nm.mm.CurrentMindmap.Nodes[newID] = newNode

	// Add to operation history
	op := Operation{
		Type: OpAdd,
		AffectedNode: models.NodeInfo{
			ID:       newNode.ID,
			ParentID: parentNode.ID,
		},
		NewContent: content,
		NewExtra:   extra,
	}
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeDelete(identifier string, useID bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	if err := nm.checkOwnership(); err != nil {
		return err
	}

	node, err := nm.findNodeByID(identifier, useID)
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
		if err := nm.nodeStore.NodeDelete(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, n.ID); err != nil {
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

	// Update indexes
	err = nm.updateSubtreeIndex(nm.mm.CurrentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update indexes after deletion: %w", err)
	}

	// Add to operation history
	op := Operation{
		Type: OpDelete,
		AffectedNode: models.NodeInfo{
			ID:       node.ID,
			ParentID: node.ParentID,
		},
		DeletedTree: deletedTree,
	}
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeUpdate(identifier string, content string, extra map[string]string, useID bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	if err := nm.checkOwnership(); err != nil {
		return err
	}

	node, err := nm.findNodeByID(identifier, useID)
	if err != nil {
		return fmt.Errorf("failed to find node to update: %w", err)
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
	if err := nm.nodeStore.NodeUpdate(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, node.ID, node.Content, node.Extra, node.Index); err != nil {
		return fmt.Errorf("failed to update node in storage: %w", err)
	}

	// Add to operation history
	op := Operation{
		Type: OpUpdate,
		AffectedNode: models.NodeInfo{
			ID: node.ID,
		},
		OldContent: oldContent,
		NewContent: content,
		OldExtra:   oldExtra,
		NewExtra:   extra,
	}
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeMove(sourceIdentifier, targetIdentifier string, useID bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	if err := nm.checkOwnership(); err != nil {
		return err
	}

	sourceNode, err := nm.findNodeByID(sourceIdentifier, useID)
	if err != nil {
		return fmt.Errorf("failed to find source node: %w", err)
	}

	targetNode, err := nm.findNodeByID(targetIdentifier, useID)
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
	sourceNode.ParentID = targetNode.ID

	// Update in storage
	if err := nm.nodeStore.NodeMove(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, sourceNode.ID, targetNode.ID); err != nil {
		return fmt.Errorf("failed to move node in storage: %w", err)
	}

	// Update indexes starting from the root
	err = nm.updateSubtreeIndex(nm.mm.CurrentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update indexes after move: %w", err)
	}

	// Add to operation history
	op := Operation{
		Type: OpMove,
		AffectedNode: models.NodeInfo{
			ID: sourceNode.ID,
		},
		OldParentID: oldParentID,
		NewParentID: targetNode.ID,
	}
	nm.historyManager.HistoryAdd(op)

	return nil
}

func (nm *NodeManager) NodeFind(query string) ([]*models.Node, error) {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return nil, fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	if !nm.mm.CurrentMindmap.IsPublic && nm.mm.CurrentMindmap.Owner != nm.mm.CurrentUser {
		return nil, fmt.Errorf("you don't have permission to view this mindmap")
	}

	var matches []*models.Node
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

	return matches, nil
}

func (nm *NodeManager) NodeSort(identifier string, field string, reverse bool, useID bool) error {
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}

	if err := nm.checkOwnership(); err != nil {
		return err
	}

	var node *models.Node
	var err error

	if identifier == "" {
		node = nm.mm.CurrentMindmap.Root
	} else {
		node, err = nm.findNodeByID(identifier, useID)
		if err != nil {
			return fmt.Errorf("failed to find node to sort: %w", err)
		}
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	// Sort the entire subtree
	sortedChildren := nm.sortNodeSubtreeRecursively(node, field, reverse)

	// Update the node's children with the sorted children
	node.Children = sortedChildren

	// Update indices in memory and database
	err = nm.updateSubtreeIndex(nm.mm.CurrentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update index after sorting: %w", err)
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

func (nm *NodeManager) sortNodeSubtreeRecursively(node *models.Node, field string, reverse bool) []*models.Node {
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
		child.Children = nm.sortNodeSubtreeRecursively(child, field, reverse)
	}

	return node.Children
}

func (nm *NodeManager) NodeGet(ID int) ([]*models.Node, error) {
	if nm.mm.CurrentMindmap == nil {
		return nil, fmt.Errorf("no mindmap selected")
	}

	node, err := nm.nodeStore.NodeGet(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}

func (nm *NodeManager) NodeGetAll() ([]*models.Node, error) {
	if nm.mm.CurrentMindmap == nil {
		return nil, fmt.Errorf("no mindmap selected")
	}

	node, err := nm.nodeStore.NodeGetAll(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}

// Helper functions that were previously in the MindmapManager but are now part of NodeManager

func (nm *NodeManager) ensureCurrentMindmap() error {
	if nm.mm.CurrentMindmap == nil {
		return fmt.Errorf("no data selected")
	}
	return nil
}

func (nm *NodeManager) checkOwnership() error {
	if nm.mm.CurrentMindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}
	if nm.mm.CurrentMindmap.Owner != nm.mm.CurrentUser {
		return fmt.Errorf("you don't have permission to modify this mindmap")
	}
	return nil
}

func (nm *NodeManager) findNodeByID(identifier string, useID bool) (*models.Node, error) {
	if useID {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return nil, fmt.Errorf("invalid index: %w", err)
		}
		return nm.mm.CurrentMindmap.Nodes[index], nil
	}
	return nm.findNodeByIndex(identifier) // TODO: this doesn't make sense anymore should does it itself
}

func (nm *NodeManager) findNodeByIndex(Index string) (*models.Node, error) {
	if Index == "0" {
		return nm.mm.CurrentMindmap.Root, nil
	}

	parts := strings.Split(Index, ".")
	currentNode := nm.mm.CurrentMindmap.Root

	for _, part := range parts {
		ID, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid index part '%s': %w", part, err)
		}
		if ID < 1 || ID > len(currentNode.Children) {
			return nil, fmt.Errorf("invalid index: part %s is out of range", part)
		}
		currentNode = currentNode.Children[ID-1]
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
	delete(nm.mm.CurrentMindmap.Nodes, node.ID)
}

func (nm *NodeManager) updateSubtreeIndex(node *models.Node) error {
	var recalculate func(*models.Node, string) error
	recalculate = func(n *models.Node, parentIndex string) error {
		for i, child := range n.Children {
			var newIndex string
			if parentIndex == "0" {
				newIndex = fmt.Sprintf("%d", i+1)
			} else {
				newIndex = fmt.Sprintf("%s.%d", parentIndex, i+1)
			}

			if child.Index != newIndex {
				child.Index = newIndex
				err := nm.nodeStore.NodeOrderUpdate(nm.mm.CurrentMindmap.Name, nm.mm.CurrentUser, child.ID, child.Index)
				if err != nil {
					return fmt.Errorf("failed to update index for node %s: %w", child.Index, err)
				}
			}

			if err := recalculate(child, child.Index); err != nil {
				return err
			}
		}
		return nil
	}

	return recalculate(node, node.Index)
}
