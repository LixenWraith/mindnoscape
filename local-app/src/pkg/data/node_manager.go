// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to node management within mindmaps.
package data

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mindnoscape/local-app/src/pkg/event"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
	"mindnoscape/local-app/src/pkg/storage"
)

// NodeOperations defines the interface for node-related operations
type NodeOperations interface {
	NodeAdd(mindmap *model.Mindmap, nodeInfo model.NodeInfo, nodeFilter model.NodeFilter) (int, int, error)
	NodeGet(mindmap *model.Mindmap, nodeInfo model.NodeInfo, nodeFilter model.NodeFilter) ([]*model.Node, error)
	NodeToInfo(node *model.Node) model.NodeInfo
	NodeFind(mindmap *model.Mindmap, nodeFilter model.NodeFilter, query string) ([]*model.Node, error)
	NodeSort(node *model.NodeInfo, nodeUpdateInfo model.NodeInfo, nodeUpdateFilter model.NodeFilter, reverse bool) error
	NodeUpdate(node *model.NodeInfo, nodeUpdateInfo model.NodeInfo, nodeUpdateFilter model.NodeFilter) error
	NodeDelete(node *model.NodeInfo, nodeFilter model.NodeFilter) error
}

// NodeManager handles all node-related operations within a mindmap.
type NodeManager struct {
	nodeStore    storage.NodeStore
	eventManager *event.EventManager
	logger       *log.Logger
}

// NewNodeManager creates a new NodeManager instance.
func NewNodeManager(nodeStore storage.NodeStore, eventManager *event.EventManager, logger *log.Logger) (*NodeManager, error) {
	ctx := context.Background()
	logger.Info(ctx, "Creating new NodeManager", nil)

	if nodeStore == nil {
		logger.Error(ctx, "NodeStore not initialized", nil)
		return nil, fmt.Errorf("nodeStore not initialized")
	}
	if eventManager == nil {
		logger.Error(ctx, "EventManager not initialized", nil)
		return nil, fmt.Errorf("eventManager not initialized")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger not initialized")
	}

	nm := &NodeManager{
		nodeStore:    nodeStore,
		eventManager: eventManager,
		logger:       logger,
	}

	logger.Info(ctx, "NodeManager created successfully", nil)
	return nm, nil
}

// handleMindmapAdded adds the root node after a mindmap is added
func (nm *NodeManager) handleMindmapAdded(e event.Event) {
	ctx := context.Background()
	nm.logger.Info(ctx, "Handling MindmapAdded event", nil)

	mindmap, ok := e.Data.(*model.Mindmap)
	if !ok {
		nm.logger.Error(ctx, "Invalid event data for mindmap add event", nil)
		return
	}

	nm.logger.Debug(ctx, "Handling MindmapAdded event", log.Fields{"mindmapID": mindmap.ID})

	rootNodeInfo := model.NodeInfo{
		ID:        0,
		MindmapID: mindmap.ID,
		ParentID:  -1,
		Name:      mindmap.Name,
		Index:     "0",
	}

	_, _, err := nm.NodeAdd(mindmap, rootNodeInfo, true)
	if err != nil {
		nm.logger.Error(ctx, "Failed to add root node for mindmap", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return
	}

	nm.logger.Info(ctx, "Root node added successfully", log.Fields{"mindmapID": mindmap.ID})
}

// handleMindmapSelected handles the event where a mindmap is selected. It fetches the associated nodes and populates the mindmap structure.
func (nm *NodeManager) handleMindmapSelected(e event.Event) {
	ctx := context.Background()
	nm.logger.Info(ctx, "Handling MindmapSelected event", nil)

	mindmap, ok := e.Data.(*model.Mindmap)
	if !ok {
		nm.logger.Error(ctx, "Invalid event data for mindmap selected event", nil)
		return
	}

	// Fetch all nodes for the mindmap
	nodes, err := nm.NodeGet(mindmap, model.NodeInfo{}, model.NodeFilter{})
	if err != nil {
		nm.logger.Error(ctx, "Failed to fetch nodes for mindmap", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return
	}

	// Populate the Nodes map
	mindmap.Nodes = make(map[int]*model.Node)
	for _, node := range nodes {
		mindmap.Nodes[node.ID] = node
		if node.ID == 0 {
			mindmap.Root = node
		}
	}

	nm.logger.Info(ctx, "Nodes loaded for mindmap", log.Fields{"mindmapID": mindmap.ID, "nodeCount": len(nodes)})
}

// handleMindmapDeleted deletes all the nodes before a mindmap is added
func (nm *NodeManager) handleMindmapDeleted(e event.Event) {
	ctx := context.Background()
	nm.logger.Info(ctx, "Handling MindmapDeleted event", nil)

	mindmap, ok := e.Data.(*model.Mindmap)
	if !ok {
		nm.logger.Error(ctx, "Invalid event data for mindmap delete event", nil)
		return
	}

	// Get the root node
	rootNodes, err := nm.NodeGet(mindmap, model.NodeInfo{ID: 0}, model.NodeFilter{ID: true})
	if err != nil || len(rootNodes) == 0 {
		nm.logger.Error(ctx, "Failed to get root node for mindmap", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return
	}
	rootNode := rootNodes[0]

	// Delete all children of the root node
	for _, child := range rootNode.Children {
		err := nm.NodeDelete(mindmap, child)
		if err != nil {
			nm.logger.Error(ctx, "Failed to delete child node", log.Fields{"error": err, "mindmapID": mindmap.ID, "nodeID": child.ID})
		}
	}

	// Delete the root node with direct storage call since NodeDelete prevents deleting the root node
	err = nm.nodeStore.NodeDelete(mindmap, rootNode)
	if err != nil {
		nm.logger.Error(ctx, "Failed to delete root node", log.Fields{"error": err, "mindmapID": mindmap.ID})
	}

	// Clear the nodes map in the mindmap
	mindmap.Nodes = make(map[int]*model.Node)

	nm.logger.Info(ctx, "All nodes deleted for mindmap", log.Fields{"mindmapID": mindmap.ID})
}

// handleMindmapUpdated updates the root node name when a mindmap is renamed
func (nm *NodeManager) handleMindmapUpdated(e event.Event) {
	ctx := context.Background()
	nm.logger.Info(ctx, "Handling MindmapUpdated event", nil)

	data, ok := e.Data.(map[string]interface{})
	if !ok {
		nm.logger.Error(ctx, "Invalid event data for mindmap update event", nil)
		return
	}

	mindmap, ok := data["mindmap"].(*model.Mindmap)
	if !ok {
		nm.logger.Error(ctx, "Invalid mindmap data in mindmap update event", nil)
		return
	}

	oldName, ok := data["oldName"].(string)
	if !ok {
		nm.logger.Error(ctx, "Invalid old name in mindmap update event", nil)
		return
	}

	// Check if the name has changed
	if mindmap.Name != oldName {
		// Get the root node
		rootNodes, err := nm.NodeGet(mindmap, model.NodeInfo{ParentID: -1}, model.NodeFilter{ParentID: true})
		if err != nil || len(rootNodes) == 0 {
			nm.logger.Error(ctx, "Failed to get root node for mindmap", log.Fields{"error": err, "mindmapID": mindmap.ID})
			return
		}
		rootNode := rootNodes[0]

		// Update the root node name
		err = nm.NodeUpdate(mindmap, rootNode, model.NodeInfo{Name: mindmap.Name}, model.NodeFilter{Name: true})
		if err != nil {
			nm.logger.Error(ctx, "Failed to update root node name", log.Fields{"error": err, "mindmapID": mindmap.ID})
		} else {
			nm.logger.Info(ctx, "Root node name updated", log.Fields{"mindmapID": mindmap.ID, "newName": mindmap.Name})
		}
	}
}

// NodeAdd adds a new node to the current mindmap.
func (nm *NodeManager) NodeAdd(mindmap *model.Mindmap, nodeInfo model.NodeInfo, forceID ...bool) (int, int, error) {
	ctx := context.Background()
	nm.logger.Info(ctx, "Adding new node", log.Fields{"mindmapID": mindmap.ID, "parentID": nodeInfo.ParentID})

	// Validate node info
	if nodeInfo.ParentID != -1 { // If not root
		parentNodes, err := nm.nodeStore.NodeGet(mindmap, model.NodeInfo{ID: nodeInfo.ParentID}, model.NodeFilter{ID: true})
		if err != nil {
			nm.logger.Error(ctx, "Failed to query parent node", log.Fields{"error": err, "parentID": nodeInfo.ParentID})
			return 0, 0, fmt.Errorf("failed to query parent node: %w", err)
		}
		if len(parentNodes) == 0 {
			nm.logger.Warn(ctx, "Parent node not found", log.Fields{"parentID": nodeInfo.ParentID})
			return 0, 0, fmt.Errorf("parent node not found: ID %d", nodeInfo.ParentID)
		}
		nm.logger.Debug(ctx, "Parent node found", log.Fields{"parentNode": parentNodes[0]})
	}

	nm.logger.Debug(ctx, "Node validation complete", nil)

	// Count nodes with the same name
	existingNodes, err := nm.nodeStore.NodeGet(mindmap, nodeInfo, model.NodeFilter{Name: true})
	if err != nil {
		nm.logger.Error(ctx, "Failed to get existing nodes", log.Fields{"error": err})
		return 0, 0, fmt.Errorf("failed to get existing nodes: %w", err)
	}
	copies := 0
	for _, node := range existingNodes {
		if node.Name == nodeInfo.Name {
			copies++
		}
	}
	nm.logger.Debug(ctx, "Node count complete", log.Fields{"copies": copies})

	// Assign index
	if nodeInfo.ParentID == -1 {
		// For root node, use "0" as index
		nodeInfo.Index = "0"
	} else {
		// Get the parent node first
		parentNodes, err := nm.nodeStore.NodeGet(mindmap, model.NodeInfo{ID: nodeInfo.ParentID}, model.NodeFilter{ID: true})
		if err != nil || len(parentNodes) == 0 {
			nm.logger.Error(ctx, "Failed to get parent node for index calculation", log.Fields{"error": err, "parentID": nodeInfo.ParentID})
			return 0, 0, fmt.Errorf("failed to get parent node for index calculation: %w", err)
		}
		parentNode := parentNodes[0]

		// Find the highest index among siblings
		siblings, err := nm.nodeStore.NodeGet(mindmap, model.NodeInfo{ParentID: nodeInfo.ParentID}, model.NodeFilter{ParentID: true})
		if err != nil {
			nm.logger.Error(ctx, "Failed to get sibling nodes", log.Fields{"error": err, "parentID": nodeInfo.ParentID})
			return 0, 0, fmt.Errorf("failed to get sibling nodes: %w", err)
		}

		highestIndex := 0
		for _, sibling := range siblings {
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
		if parentNode.ParentID == -1 { // If parent is root
			nodeInfo.Index = fmt.Sprintf("%d", highestIndex+1)
		} else {
			nodeInfo.Index = fmt.Sprintf("%s.%d", parentNode.Index, highestIndex+1)
		}
	}
	nm.logger.Debug(ctx, "Index calculation complete", log.Fields{"index": nodeInfo.Index})

	// Add to storage
	var newID int
	if len(forceID) > 0 && forceID[0] {
		// Use the provided ID when forceID is true
		newID, err = nm.nodeStore.NodeAdd(mindmap, nodeInfo, true)
	} else {
		newID, err = nm.nodeStore.NodeAdd(mindmap, nodeInfo)
	}
	if err != nil {
		nm.logger.Error(ctx, "Failed to add node to storage", log.Fields{"error": err})
		return newID, copies, fmt.Errorf("failed to add node to storage: %w", err)
	}
	copies++
	nm.logger.Debug(ctx, "Node added to storage", log.Fields{"newID": newID})

	// Get the newly created node
	newNodes, err := nm.nodeStore.NodeGet(mindmap, model.NodeInfo{ID: newID}, model.NodeFilter{ID: true})
	if err != nil || len(newNodes) == 0 {
		return newID, copies, fmt.Errorf("failed to retrieve newly created node: %w", err)
	}
	newNode := newNodes[0]

	// Update in-memory structure
	if nodeInfo.ParentID != -1 {
		parentNode, exists := mindmap.Nodes[nodeInfo.ParentID]
		if !exists {
			nm.logger.Error(ctx, "Parent node not found in memory", log.Fields{"parentID": nodeInfo.ParentID})
			return newID, copies, fmt.Errorf("parent node not found in memory: %d", nodeInfo.ParentID)
		}
		parentNode.Children = append(parentNode.Children, newNode)
	}

	nm.logger.Debug(ctx, "In-memory structure updated", log.Fields{"newNode": newNode})

	// Initialize the Nodes map if it's nil, sure case root node for a new mindmap
	if mindmap.Nodes == nil {
		mindmap.Nodes = make(map[int]*model.Node)
	}
	mindmap.Nodes[newID] = newNode

	nm.logger.Debug(ctx, "Node map initialized with root node", nil)

	// Set mindmap Root node if the new node is root
	if newID == 0 {
		mindmap.Root = newNode
	}

	nm.logger.Info(ctx, "Node added successfully", log.Fields{"nodeID": newID, "mindmapID": mindmap.ID})
	return newID, copies, nil
}

// NodeGet retrieves nodes based on the provided info and filter
func (nm *NodeManager) NodeGet(mindmap *model.Mindmap, nodeInfo model.NodeInfo, nodeFilter model.NodeFilter) ([]*model.Node, error) {
	ctx := context.Background()
	nm.logger.Info(ctx, "Retrieving nodes", log.Fields{"mindmapID": mindmap.ID, "filter": nodeFilter})

	if mindmap == nil {
		nm.logger.Error(ctx, "Mindmap is nil", nil)
		return nil, fmt.Errorf("mindmap is nil")
	}

	nodes, err := nm.nodeStore.NodeGet(mindmap, nodeInfo, nodeFilter)
	if err != nil {
		nm.logger.Error(ctx, "Failed to get nodes", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	if len(nodes) == 0 {
		nm.logger.Debug(ctx, "No nodes found", log.Fields{"filter": nodeFilter})
	}

	nm.logger.Info(ctx, "Nodes retrieved successfully", log.Fields{"count": len(nodes)})
	return nodes, nil
}

// NodeToInfo converts a Node instance to NodeInfo
func (nm *NodeManager) NodeToInfo(node *model.Node) model.NodeInfo {
	return model.NodeInfo{
		ID:        node.ID,
		MindmapID: node.MindmapID,
		ParentID:  node.ParentID,
		Name:      node.Name,
		Index:     node.Index,
		Content:   node.Content,
	}
}

// NodeFind searches for nodes in the mindmap based on a query string
func (nm *NodeManager) NodeFind(mindmap *model.Mindmap, nodeFilter model.NodeFilter, query string) ([]*model.Node, error) {
	ctx := context.Background()
	nm.logger.Info(ctx, "Searching for nodes", log.Fields{"mindmapID": mindmap.ID, "query": query})

	// Check if the mindmap exists
	if mindmap == nil {
		nm.logger.Error(ctx, "Mindmap not specified", nil)
		return nil, fmt.Errorf("mindmap not specified")
	}

	// Fetch all nodes for the mindmap
	allNodes, err := nm.NodeGet(mindmap, model.NodeInfo{}, model.NodeFilter{})
	if err != nil {
		nm.logger.Error(ctx, "Failed to get nodes", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Search for matches based on the filter
	var matches []*model.Node
	lowerQuery := strings.ToLower(query)

	for _, node := range allNodes {
		if nodeFilter.Name && strings.Contains(strings.ToLower(node.Name), lowerQuery) {
			matches = append(matches, node)
			continue
		}
		if nodeFilter.Content {
			contentMatch := false
			for key, value := range node.Content {
				if strings.Contains(strings.ToLower(key), lowerQuery) || strings.Contains(strings.ToLower(value), lowerQuery) {
					contentMatch = true
					break
				}
			}
			if contentMatch {
				matches = append(matches, node)
				continue
			}
		}
		if nodeFilter.Index && strings.Contains(node.Index, query) {
			matches = append(matches, node)
			continue
		}
	}

	nm.logger.Info(ctx, "Node search completed", log.Fields{"matchCount": len(matches)})
	return matches, nil
}

// NodeSort sorts the children of a node based on a given field
func (nm *NodeManager) NodeSort(mindmap *model.Mindmap, nodeInfo model.NodeInfo, field string, reverse bool) error {
	ctx := context.Background()
	nm.logger.Info(ctx, "Sorting nodes", log.Fields{"mindmapID": mindmap.ID, "nodeID": nodeInfo.ID, "field": field, "reverse": reverse})

	// Find the node to sort
	var node *model.Node
	var err error

	if nodeInfo.ID == 0 || nodeInfo.Index == "0" {
		node = mindmap.Root
	} else {
		nodes, err := nm.NodeGet(mindmap, nodeInfo, model.NodeFilter{ID: true, Index: true})
		if err != nil || len(nodes) == 0 {
			nm.logger.Error(ctx, "Failed to find node to sort", log.Fields{"error": err, "nodeID": nodeInfo.ID})
			return fmt.Errorf("failed to find node to sort: %w", err)
		}
		node = nodes[0]
	}

	// Sort the entire subtree
	sortedChildren := nm.sortNodeSubtreeRecursively(node, field, reverse)

	// Update the node's children with the sorted children
	node.Children = sortedChildren

	// Update indices in memory and database
	err = nm.updateSubtreeIndex(mindmap, node)
	if err != nil {
		nm.logger.Error(ctx, "Failed to update index after sorting", log.Fields{"error": err, "nodeID": node.ID})
		return fmt.Errorf("failed to update index after sorting: %w", err)
	}

	// Update the sorted nodes in storage
	err = nm.updateSortedNodesInStorage(mindmap, node)
	if err != nil {
		nm.logger.Error(ctx, "Failed to update sorted nodes in storage", log.Fields{"error": err, "nodeID": node.ID})
		return fmt.Errorf("failed to update sorted nodes in storage: %w", err)
	}

	// Publish NodeSorted event   // todo: placeholder
	nm.eventManager.Publish(event.Event{
		Type: event.NodeSorted,
		Data: map[string]interface{}{
			"mindmap": mindmap,
			"node":    node,
			"field":   field,
			"reverse": reverse,
		},
	})

	nm.logger.Info(ctx, "Nodes sorted successfully", log.Fields{"nodeID": node.ID})
	return nil
}

func (nm *NodeManager) sortNodeSubtreeRecursively(node *model.Node, field string, reverse bool) []*model.Node {
	sort.Slice(node.Children, func(i, j int) bool {
		var vi, vj string
		if field == "" {
			vi, vj = node.Children[i].Name, node.Children[j].Name
		} else {
			vi = node.Children[i].Content[field]
			vj = node.Children[j].Content[field]
		}
		// If the field doesn't exist, fall back to Name
		if vi == "" && vj == "" {
			vi, vj = node.Children[i].Name, node.Children[j].Name
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

func (nm *NodeManager) updateSortedNodesInStorage(mindmap *model.Mindmap, node *model.Node) error {
	ctx := context.Background()
	nm.logger.Debug(ctx, "Updating sorted nodes in storage", log.Fields{"nodeID": node.ID})

	var updateNode func(*model.Node) error
	updateNode = func(n *model.Node) error {
		// Update the current node
		err := nm.nodeStore.NodeUpdate(mindmap, n, model.NodeInfo{
			Index:    n.Index,
			ParentID: n.ParentID,
		}, model.NodeFilter{Index: true, ParentID: true})
		if err != nil {
			nm.logger.Error(ctx, "Failed to update node in storage", log.Fields{"error": err, "nodeID": n.ID})
			return fmt.Errorf("failed to update node %d in storage: %w", n.ID, err)
		}

		// Recursively update children
		for _, child := range n.Children {
			err := updateNode(child)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return updateNode(node)
}

// NodeUpdate updates an existing node's information
func (nm *NodeManager) NodeUpdate(mindmap *model.Mindmap, node *model.Node, nodeUpdateInfo model.NodeInfo, nodeUpdateFilter model.NodeFilter) error {
	ctx := context.Background()
	nm.logger.Info(ctx, "Updating node", log.Fields{"mindmapID": mindmap.ID, "nodeID": node.ID})

	// Check if the mindmap exists
	if mindmap == nil {
		nm.logger.Error(ctx, "Mindmap not specified", nil)
		return fmt.Errorf("mindmap not specified")
	}

	// Check if the node exists
	if node == nil {
		nm.logger.Error(ctx, "Node not found", nil)
		return fmt.Errorf("node not found")
	}

	// Store old values for potential rollback and event
	oldName := node.Name
	oldContent := make(map[string]string)
	for k, v := range node.Content {
		oldContent[k] = v
	}
	oldParentID := node.ParentID

	// Special handling for root node (ID 0)
	if node.ID == 0 {
		if nodeUpdateFilter.Name && nodeUpdateInfo.Name != "" {
			oldName := node.Name
			node.Name = nodeUpdateInfo.Name

			// Publish RootNodeRenamed event instead of directly updating the mindmap
			nm.eventManager.Publish(event.Event{
				Type: event.RootNodeRenamed,
				Data: map[string]interface{}{
					"mindmapID": mindmap.ID,
					"newName":   nodeUpdateInfo.Name,
					"oldName":   oldName,
				},
			})
		}

		// Ensure root node always has correct index and parentID
		node.Index = "0"
		node.ParentID = -1

		// Prevent changing root node's ID, Index, or ParentID
		if nodeUpdateFilter.ID || nodeUpdateFilter.Index || nodeUpdateFilter.ParentID {
			nm.logger.Warn(ctx, "Attempt to change root node's ID, Index, or ParentID", nil)
			return fmt.Errorf("cannot change ID, Index, or ParentID of root node")
		}
	} else {
		// Update non-root node fields based on the filter
		if nodeUpdateFilter.Name && nodeUpdateInfo.Name != "" {
			node.Name = nodeUpdateInfo.Name
		}
		if nodeUpdateFilter.ParentID && nodeUpdateInfo.ParentID != node.ParentID {
			// Check if the new parent exists
			newParent, exists := mindmap.Nodes[nodeUpdateInfo.ParentID]
			if !exists {
				nm.logger.Error(ctx, "New parent node not found", log.Fields{"newParentID": nodeUpdateInfo.ParentID})
				return fmt.Errorf("new parent node not found %v", nodeUpdateInfo.ParentID)
			}

			// Remove node from old parent's children
			oldParent, exists := mindmap.Nodes[node.ParentID]
			if exists {
				for i, child := range oldParent.Children {
					if child.ID == node.ID {
						oldParent.Children = append(oldParent.Children[:i], oldParent.Children[i+1:]...)
						break
					}
				}
			}

			// Add node to new parent's children
			newParent.Children = append(newParent.Children, node)
			node.ParentID = nodeUpdateInfo.ParentID
		}
	}

	if nodeUpdateFilter.Content {
		for k, v := range nodeUpdateInfo.Content {
			if v == "" {
				delete(node.Content, k)
			} else {
				node.Content[k] = v
			}
		}
	}

	// Update in storage
	err := nm.nodeStore.NodeUpdate(mindmap, node, nodeUpdateInfo, nodeUpdateFilter)
	if err != nil {
		// Rollback changes if storage update fails
		node.Name = oldName
		node.Content = oldContent
		node.ParentID = oldParentID
		nm.logger.Error(ctx, "Failed to update node in storage", log.Fields{"error": err, "nodeID": node.ID})
		return fmt.Errorf("failed to update node in storage: %w", err)
	}

	// Update indices if parent changed
	if nodeUpdateFilter.ParentID && oldParentID != node.ParentID {
		err = nm.updateSubtreeIndex(mindmap, mindmap.Root)
		if err != nil {
			nm.logger.Error(ctx, "Failed to update indices after parent change", log.Fields{"error": err, "nodeID": node.ID})
			return fmt.Errorf("failed to update indices after parent change: %w", err)
		}
	}

	// Publish NodeUpdated event
	nm.eventManager.Publish(event.Event{
		Type: event.NodeUpdated,
		Data: map[string]interface{}{
			"mindmap":     mindmap,
			"node":        node,
			"oldName":     oldName,
			"oldContent":  oldContent,
			"oldParentID": oldParentID,
		},
	})

	nm.logger.Info(ctx, "Node updated successfully", log.Fields{"nodeID": node.ID})
	return nil
}

// NodeDelete removes a node and its subtree
func (nm *NodeManager) NodeDelete(mindmap *model.Mindmap, node *model.Node) error {
	ctx := context.Background()
	nm.logger.Info(ctx, "Deleting node", log.Fields{"mindmapID": mindmap.ID, "nodeID": node.ID})

	// Prevent deleting the root node
	if node.ID == 0 {
		nm.logger.Warn(ctx, "Attempt to delete root node", nil)
		return fmt.Errorf("cannot delete root node")
	}

	// Check if the node still exists in the mindmap
	if _, exists := mindmap.Nodes[node.ID]; !exists {
		// Node has already been deleted, possibly as part of its parent's deletion
		nm.logger.Debug(ctx, "Node already deleted", log.Fields{"nodeID": node.ID})
		return nil
	}

	// Find the parent node
	parentNode, exists := mindmap.Nodes[node.ParentID]
	if !exists {
		nm.logger.Error(ctx, "Parent node not found", log.Fields{"parentID": node.ParentID})
		return fmt.Errorf("parent node not found")
	}

	// Collect all nodes in the subtree to be deleted
	nodesToDelete := []*model.Node{node}
	nm.getSubtreeNodes(mindmap, node, &nodesToDelete)

	// Remove from storage and in-memory structure
	for _, n := range nodesToDelete {
		err := nm.nodeStore.NodeDelete(mindmap, n)
		if err != nil {
			nm.logger.Error(ctx, "Failed to delete node from storage", log.Fields{"error": err, "nodeID": n.ID})
			return fmt.Errorf("failed to delete node %d from storage: %w", n.ID, err)
		}
		delete(mindmap.Nodes, n.ID)
	}

	// Update parent's children list
	for i, child := range parentNode.Children {
		if child.ID == node.ID {
			parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
			break
		}
	}

	// Delete the node and its descendants recursively from the in-memory structure
	nm.deleteNodeRecursive(mindmap, node)

	// Update indexes
	err := nm.updateSubtreeIndex(mindmap, mindmap.Root)
	if err != nil {
		nm.logger.Error(ctx, "Failed to update indexes after deletion", log.Fields{"error": err})
		return fmt.Errorf("failed to update indexes after deletion: %w", err)
	}

	// Publish NodeDeleted event  // todo: placeholder
	nm.eventManager.Publish(event.Event{
		Type: event.NodeDeleted,
		Data: map[string]interface{}{
			"mindmap": mindmap,
			"node":    node,
		},
	})

	return nil
}

// deleteNodeRecursive removes a node and its descendants from the in-memory structure. Traverse the whole mindmap, assuming children slice is unreliable.
func (nm *NodeManager) deleteNodeRecursive(mindmap *model.Mindmap, node *model.Node) {
	for _, childNode := range mindmap.Nodes {
		if childNode.ParentID == node.ID { // extra safety check
			nm.deleteNodeRecursive(mindmap, childNode)
		}
	}
	delete(mindmap.Nodes, node.ID)
}

// getSubtreeNodes collects all nodes in a subtree. Traverse the whole mindmap, assuming children slice is unreliable.
func (nm *NodeManager) getSubtreeNodes(mindmap *model.Mindmap, node *model.Node, nodes *[]*model.Node) {
	for _, childNode := range mindmap.Nodes {
		if childNode.ParentID == node.ID { // extra safety check
			*nodes = append(*nodes, childNode)
			nm.getSubtreeNodes(mindmap, childNode, nodes)
		}
	}
}

// updateSubtreeIndex updates the indices of all nodes in a subtree.
func (nm *NodeManager) updateSubtreeIndex(mindmap *model.Mindmap, node *model.Node) error {
	ctx := context.Background()
	nm.logger.Debug(ctx, "Updating subtree index", log.Fields{"nodeID": node.ID})

	var recalculate func(*model.Node, string) error
	recalculate = func(n *model.Node, parentIndex string) error {
		for i, child := range n.Children {
			var newIndex string
			if parentIndex == "0" {
				newIndex = fmt.Sprintf("%d", i+1)
			} else {
				newIndex = fmt.Sprintf("%s.%d", parentIndex, i+1)
			}
			if child.Index != newIndex {
				child.Index = newIndex
				err := nm.NodeUpdate(mindmap, child, model.NodeInfo{Index: newIndex}, model.NodeFilter{Index: true})
				if err != nil {
					nm.logger.Error(ctx, "Failed to update index for node", log.Fields{"error": err, "nodeID": child.ID})
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
