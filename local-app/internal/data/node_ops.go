// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to node management within mindmaps.
package data

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mindnoscape/local-app/internal/models"
	"mindnoscape/local-app/internal/storage"
)

// NodeOperations defines the interface for node-related operations
type NodeOperations interface {
	NodeAdd(parentIdentifier string, content string, extra map[string]string, useID bool, skipHistory bool, id ...int) (int, error)
	NodeDelete(identifier string, useID bool, skipHistory bool) error
	NodeUpdate(identifier string, content string, extra map[string]string, useID bool, skipHistory bool) error
	NodeMove(sourceIdentifier, targetIdentifier string, useID bool, skipHistory bool) error
	NodeFind(query string) ([]*models.Node, error)
	NodeSort(identifier string, field string, reverse bool, useID bool) error
	NodeGet(ID int) (*models.Node, error)
	NodeGetAll() ([]*models.Node, error)
	NodeUndo() error
	NodeRedo() error
}

// NodeManager handles all node-related operations within a mindmap.
type NodeManager struct {
	nodeStore      storage.NodeStore
	mm             *MindmapManager
	historyManager *HistoryManager
}

// NewNodeManager creates a new NodeManager instance.
func NewNodeManager(mm *MindmapManager) *NodeManager {
	nm := &NodeManager{
		nodeStore: mm.nodeStore,
		mm:        mm,
	}
	nm.historyManager = NewHistoryManager(nm)
	return nm
}

// NodeAdd adds a new node to the current mindmap.
func (nm *NodeManager) NodeAdd(parentIdentifier string, content string, extra map[string]string, useID bool, skipHistory bool, id ...int) (int, error) {
	// Ensure a mindmap is selected and the user has ownership
	if err := nm.ensureCurrentMindmap(); err != nil {
		return 0, fmt.Errorf("failed to ensure current mindmap: %w", err)
	}
	if err := nm.checkOwnership(); err != nil {
		return 0, err
	}

	// Find the parent node
	parentNode, err := nm.findNodeByID(parentIdentifier, useID)
	if err != nil {
		return 0, fmt.Errorf("failed to find parent node: %w", err)
	}
	if parentNode == nil {
		return 0, fmt.Errorf("parent node not found")
	}

	// Count nodes with the same content
	copies := 0
	for _, node := range nm.mm.currentMindmap.Nodes {
		if node.Content == content {
			copies += 1
		}
	}

	// Create the new node
	newNode := models.NewNode(0, content, nm.mm.currentMindmap.ID)
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
	var newID int
	if len(id) > 0 {
		newID, err = nm.nodeStore.NodeAdd(nm.mm.currentMindmap.Name, nm.mm.currentUser, parentNode.ID, newNode.Content, newNode.Extra, newNode.Index, id[0])
	} else {
		newID, err = nm.nodeStore.NodeAdd(nm.mm.currentMindmap.Name, nm.mm.currentUser, parentNode.ID, newNode.Content, newNode.Extra, newNode.Index)
	}
	if err != nil {
		return copies, fmt.Errorf("failed to add node to storage: %w", err)
	}
	copies += 1

	// Use the auto-incremented id assigned by db as id
	newNode.ID = newID

	// Update in-memory structure
	parentNode.Children = append(parentNode.Children, newNode)
	nm.mm.currentMindmap.Nodes[newID] = newNode

	// Add to operation history if not skipping due to a call from undo/redo
	if !skipHistory {
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
	}

	return copies, nil
}

// NodeDelete removes a node and its subtree from the current mindmap.
func (nm *NodeManager) NodeDelete(identifier string, useID bool, skipHistory bool) error {
	// Ensure a mindmap is selected and the user has ownership
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}
	if err := nm.checkOwnership(); err != nil {
		return err
	}

	// Find the node to delete
	node, err := nm.findNodeByID(identifier, useID)
	if err != nil {
		return fmt.Errorf("failed to find node to delete: %w", err)
	}

	// Prevent deleting the root node
	if node == nm.mm.currentMindmap.Root {
		return fmt.Errorf("cannot delete root node")
	}

	// Find the parent node
	oldParentNode := nm.mm.currentMindmap.Nodes[node.ParentID]
	if oldParentNode == nil {
		return fmt.Errorf("old parent node not found")
	}

	// Collect all nodes in the subtree to be deleted
	nodesToDelete := []*models.Node{node}
	nm.getSubtreeNodes(node, &nodesToDelete)

	// Remove from storage
	for _, n := range nodesToDelete {
		err = nm.nodeStore.NodeDelete(nm.mm.currentMindmap.Name, nm.mm.currentUser, n.ID)
		if err != nil {
			return fmt.Errorf("failed to delete node %d from storage: %w", n.ID, err)
		}
	}

	// Update in-memory structure
	for i, child := range oldParentNode.Children {
		if child == node {
			oldParentNode.Children = append(oldParentNode.Children[:i], oldParentNode.Children[i+1:]...)
			break
		}
	}

	// Delete the node and its descendants recursively from the in-memory structure
	nm.deleteNodeRecursive(node)

	// Update indexes
	err = nm.updateSubtreeIndex(nm.mm.currentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update indexes after deletion: %w", err)
	}

	// Add to operation history if not skipping due to a call from undo/redo
	if !skipHistory {
		op := Operation{
			Type: OpDelete,
			AffectedNode: models.NodeInfo{
				ID:       node.ID,
				ParentID: node.ParentID,
			},
			DeletedTree: nodesToDelete,
		}
		nm.historyManager.HistoryAdd(op)
	}

	return nil
}

// NodeUpdate modifies the content or extra fields of an existing node.
func (nm *NodeManager) NodeUpdate(identifier string, content string, extra map[string]string, useID bool, skipHistory bool) error {
	// Ensure a mindmap is selected and the user has ownership
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}
	if err := nm.checkOwnership(); err != nil {
		return err
	}

	// Find the node to update
	node, err := nm.findNodeByID(identifier, useID)
	if err != nil {
		return fmt.Errorf("failed to find node to update: %w", err)
	}
	if node == nil {
		return fmt.Errorf("node not found")
	}

	// Store old content and extra for undo
	oldContent := node.Content
	oldExtra := make(map[string]string)
	for k, v := range node.Extra {
		oldExtra[k] = v
	}

	// Update node content and extra fields
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
	if err := nm.nodeStore.NodeUpdate(nm.mm.currentMindmap.Name, nm.mm.currentUser, node.ID, node.Content, node.Extra, node.Index); err != nil {
		return fmt.Errorf("failed to update node in storage: %w", err)
	}

	// Add to operation history if not skipping due to a call from undo/redo
	if !skipHistory {
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
	}

	return nil
}

func (nm *NodeManager) NodeMove(sourceIdentifier, targetIdentifier string, useID bool, skipHistory bool) error {
	// Ensure a mindmap is selected and the user has ownership
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}
	if err := nm.checkOwnership(); err != nil {
		return err
	}

	// Find the source and target nodes
	sourceNode, err := nm.findNodeByID(sourceIdentifier, useID)
	if err != nil {
		return fmt.Errorf("failed to find source node: %w", err)
	}
	targetNode, err := nm.findNodeByID(targetIdentifier, useID)
	if err != nil {
		return fmt.Errorf("failed to find target node: %w", err)
	}

	// Prevent moving the root node
	if sourceNode == nm.mm.currentMindmap.Root {
		return fmt.Errorf("cannot move root node")
	}

	// Find the old parent node
	oldParentNode := nm.mm.currentMindmap.Nodes[sourceNode.ParentID]
	if oldParentNode == nil {
		return fmt.Errorf("old parent node not found")
	}

	// Store old parent ID for undo
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
	if err := nm.nodeStore.NodeMove(nm.mm.currentMindmap.Name, nm.mm.currentUser, sourceNode.ID, targetNode.ID); err != nil {
		return fmt.Errorf("failed to move node in storage: %w", err)
	}

	// Update indexes starting from the root
	err = nm.updateSubtreeIndex(nm.mm.currentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update indexes after move: %w", err)
	}

	// Add to operation history if not skipping due to a call from undo/redo
	if !skipHistory {
		op := Operation{
			Type: OpMove,
			AffectedNode: models.NodeInfo{
				ID: sourceNode.ID,
			},
			OldParentID: oldParentID,
			NewParentID: targetNode.ID,
		}
		nm.historyManager.HistoryAdd(op)
	}

	return nil
}

// NodeFind searches for nodes in the current mindmap based on a query string.
func (nm *NodeManager) NodeFind(query string) ([]*models.Node, error) {
	// Ensure a mindmap is selected
	if err := nm.ensureCurrentMindmap(); err != nil {
		return nil, fmt.Errorf("failed to ensure current mindmap: %w", err)
	}
	// Check if the user has permission to view the mindmap
	if !nm.mm.currentMindmap.IsPublic && nm.mm.currentMindmap.Owner != nm.mm.currentUser {
		return nil, fmt.Errorf("you don't have permission to view this mindmap")
	}

	// Search for nodes that their content or extra field contains the query string
	var matches []*models.Node
	for _, node := range nm.mm.currentMindmap.Nodes {
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

// NodeSort sorts the children of a specified node based on a given field.
func (nm *NodeManager) NodeSort(identifier string, field string, reverse bool, useID bool) error {
	// Ensure a mindmap is selected and the user has ownership
	if err := nm.ensureCurrentMindmap(); err != nil {
		return fmt.Errorf("failed to ensure current mindmap: %w", err)
	}
	if err := nm.checkOwnership(); err != nil {
		return err
	}

	// Find the node to sort
	var node *models.Node
	var err error
	if identifier == "" {
		node = nm.mm.currentMindmap.Root
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
	err = nm.updateSubtreeIndex(nm.mm.currentMindmap.Root)
	if err != nil {
		return fmt.Errorf("failed to update index after sorting: %w", err)
	}

	return nil
}

// NodeUndo reverts the last node operation.
func (nm *NodeManager) NodeUndo() error {
	op, err := nm.historyManager.GetLastOperation()
	if err != nil {
		return err
	}

	switch op.Type {
	case OpAdd:
		err = nm.NodeDelete(strconv.Itoa(op.AffectedNode.ID), true, true)
	case OpDelete:
		err = nm.restoreSubtree(op.DeletedTree)
	case OpMove:
		err = nm.NodeMove(strconv.Itoa(op.AffectedNode.ID), strconv.Itoa(op.OldParentID), true, true)
	case OpUpdate:
		err = nm.NodeUpdate(strconv.Itoa(op.AffectedNode.ID), op.OldContent, op.OldExtra, true, true)
	}

	if err != nil {
		return fmt.Errorf("failed to undo %s: %w", op.Type, err)
	}

	nm.historyManager.RemoveLastOperation()
	return nil
}

// NodeRedo reapplies the last undone node operation.
func (nm *NodeManager) NodeRedo() error {
	op, err := nm.historyManager.GetNextOperation()
	if err != nil {
		return err
	}

	switch op.Type {
	case OpAdd:
		_, err = nm.NodeAdd(strconv.Itoa(op.AffectedNode.ParentID), op.NewContent, op.NewExtra, true, true, op.AffectedNode.ID)
	case OpDelete:
		err = nm.NodeDelete(strconv.Itoa(op.AffectedNode.ID), true, true)
	case OpMove:
		err = nm.NodeMove(strconv.Itoa(op.AffectedNode.ID), strconv.Itoa(op.NewParentID), true, true)
	case OpUpdate:
		err = nm.NodeUpdate(strconv.Itoa(op.AffectedNode.ID), op.NewContent, op.NewExtra, true, true)
	}

	if err != nil {
		return fmt.Errorf("failed to redo %s: %w", op.Type, err)
	}

	nm.historyManager.MoveToNextOperation()
	return nil
}

// NodeReset clears the node operation history.
func (nm *NodeManager) NodeReset() {
	nm.historyManager.HistoryReset()
}

// NodeGet retrieves a specific node by its ID.
func (nm *NodeManager) NodeGet(ID int) ([]*models.Node, error) {
	if nm.mm.currentMindmap == nil {
		return nil, fmt.Errorf("no mindmap selected")
	}

	node, err := nm.nodeStore.NodeGet(nm.mm.currentMindmap.Name, nm.mm.currentUser, ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}

// NodeGetAll retrieves all nodes in the current mindmap.
func (nm *NodeManager) NodeGetAll() ([]*models.Node, error) {
	if nm.mm.currentMindmap == nil {
		return nil, fmt.Errorf("no mindmap selected")
	}

	node, err := nm.nodeStore.NodeGetAll(nm.mm.currentMindmap.Name, nm.mm.currentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}

// Helper functions that were previously in the MindmapManager but are now part of NodeManager

func (nm *NodeManager) ensureCurrentMindmap() error {
	if nm.mm.currentMindmap == nil {
		return fmt.Errorf("no data selected")
	}
	return nil
}

// checkOwnership verifies if the current user owns the selected mindmap.
func (nm *NodeManager) checkOwnership() error {
	if nm.mm.currentMindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}
	if nm.mm.currentMindmap.Owner != nm.mm.currentUser {
		return fmt.Errorf("you don't have permission to modify this mindmap")
	}
	return nil
}

// findNodeByID locates a node by its ID or index.
func (nm *NodeManager) findNodeByID(identifier string, useID bool) (*models.Node, error) {
	if useID {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return nil, fmt.Errorf("invalid index: %w", err)
		}
		return nm.mm.currentMindmap.Nodes[index], nil
	}
	return nm.findNodeByIndex(identifier) // TODO: this doesn't make sense anymore should does it itself
}

// findNodeByIndex locates a node by its logical index.
func (nm *NodeManager) findNodeByIndex(Index string) (*models.Node, error) {
	if Index == "0" {
		return nm.mm.currentMindmap.Root, nil
	}

	parts := strings.Split(Index, ".")
	currentNode := nm.mm.currentMindmap.Root

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

// getSubtreeNodes collects all nodes in a subtree.
func (nm *NodeManager) getSubtreeNodes(node *models.Node, nodes *[]*models.Node) {
	for _, child := range node.Children {
		*nodes = append(*nodes, child)
		nm.getSubtreeNodes(child, nodes)
	}
}

// sortNodeSubtreeRecursively sorts a node's subtree based on the given field and order.
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

// deleteNodeRecursive removes a node and its descendants from the in-memory structure.
func (nm *NodeManager) deleteNodeRecursive(node *models.Node) {
	for _, child := range node.Children {
		nm.deleteNodeRecursive(child)
	}
	delete(nm.mm.currentMindmap.Nodes, node.ID)
}

// updateSubtreeIndex updates the indices of all nodes in a subtree.
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
				err := nm.nodeStore.NodeOrderUpdate(nm.mm.currentMindmap.Name, nm.mm.currentUser, child.ID, child.Index)
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

// restoreSubtree recreates a deleted subtree of nodes.
func (nm *NodeManager) restoreSubtree(nodes []*models.Node) error {
	for _, node := range nodes {
		_, err := nm.NodeAdd(strconv.Itoa(node.ParentID), node.Content, node.Extra, true, true, node.ID)
		if err != nil {
			return fmt.Errorf("failed to restore node %d: %w", node.ID, err)
		}
	}
	return nil
}
