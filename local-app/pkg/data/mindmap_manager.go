// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to mindmap management.
package data

import (
	"fmt"

	"mindnoscape/local-app/pkg/event"
	"mindnoscape/local-app/pkg/log"
	"mindnoscape/local-app/pkg/model"
	"mindnoscape/local-app/pkg/storage"
)

// MindmapOperations defines the interface for mindmap-related operations
type MindmapOperations interface {
	MindmapAdd(user *model.User, newMindmapInfo model.MindmapInfo) (int, error)
	MindmapPermission(user *model.User, mindmapInfo model.MindmapInfo) (int, error)
	MindmapGet(user *model.User, mindmapInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) ([]*model.Mindmap, error)
	MindmapToInfo(mindmap *model.Mindmap) model.MindmapInfo
	MindmapUpdate(user *model.User, mindmap *model.Mindmap, mindmapUpdateInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) error
	MindmapDelete(user *model.User, mindmap *model.Mindmap) error
}

// MindmapManager handles all mindmap-related operations and maintains the current mindmap state.
type MindmapManager struct {
	mindmapStore storage.MindmapStore
	eventManager *event.EventManager
	logger       *log.Logger
}

func NewMindmapManager(mindmapStore storage.MindmapStore, eventManager *event.EventManager, logger *log.Logger) (*MindmapManager, error) {
	if mindmapStore == nil {
		return nil, fmt.Errorf("mindmapStore not initialized")
	}
	if eventManager == nil {
		return nil, fmt.Errorf("eventManager not initialized")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger not initialized")
	}
	mm := &MindmapManager{
		mindmapStore: mindmapStore,
		eventManager: eventManager,
		logger:       logger,
	}
	eventManager.Subscribe(event.RootNodeRenamed, mm.handleRootNodeRenamed)
	return mm, nil
}

// MindmapAdd creates a new mindmap with the given name and visibility.
func (mm *MindmapManager) MindmapAdd(user *model.User, newMindmapInfo model.MindmapInfo) (int, error) {
	// Check if the user already has a mindmap with the same name
	existingMindmaps, err := mm.MindmapGet(user, newMindmapInfo, model.MindmapFilter{Name: true})
	if err != nil {
		return 0, fmt.Errorf("failed to check for existing mindmap: %w", err)
	}
	if len(existingMindmaps) > 0 {
		return 0, fmt.Errorf("mindmap with name '%s' already exists for this user", newMindmapInfo.Name)
	}

	// Add the new mindmap to storage
	newMindmapInfo.Owner = user.Username
	id, err := mm.mindmapStore.MindmapAdd(user, newMindmapInfo)
	if err != nil {
		return 0, fmt.Errorf("failed to add mindmap: %w", err)
	}
	newMindmapInfo.ID = id

	mindmaps, err := mm.MindmapGet(user, newMindmapInfo, model.MindmapFilter{ID: true, Owner: true})
	if err != nil {
		return 0, fmt.Errorf("failed to get the new mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		return 0, fmt.Errorf("could not find the new mindmap")
	}
	newMindmap := mindmaps[0]

	fmt.Println("DEBUG: Added mindmap with ID", newMindmap.ID, "and name", newMindmap.Name)
	fmt.Println("DEBUG: newMindmap: ", newMindmap)

	// Initialize the Nodes map
	newMindmap.Nodes = make(map[int]*model.Node)

	// Publish MindmapCreated event
	mm.eventManager.Publish(event.Event{
		Type: event.MindmapAdded,
		Data: newMindmap,
	})

	fmt.Printf("DEBUG: Published MindmapAdded event for mindmap %d\n", id)

	return id, nil
}

// MindmapPermission checks the permission of a user for a specific mindmap
func (mm *MindmapManager) MindmapPermission(user *model.User, mindmapInfo model.MindmapInfo) (int, error) {
	// Get the mindmap
	mindmaps, err := mm.MindmapGet(user, mindmapInfo, model.MindmapFilter{ID: true, Owner: true, IsPublic: true})
	if err != nil {
		return 0, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		return 0, fmt.Errorf("mindmap not found")
	}
	mindmap := mindmaps[0]

	// Check ownership
	if mindmap.Owner == user.Username {
		return 2, nil // Owner has full permission
	}

	// Check if the mindmap is public
	if mindmap.IsPublic {
		return 1, nil // Public mindmap, read-only permission
	}

	// User is not the owner and the mindmap is not public
	return 0, nil // No permission
}

// MindmapGet retrieves the information of mindmaps based on provided filters and user permissions.
func (mm *MindmapManager) MindmapGet(user *model.User, mindmapInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) ([]*model.Mindmap, error) {
	// Get all mindmaps that match the filter
	mindmaps, err := mm.mindmapStore.MindmapGet(user, mindmapInfo, mindmapFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmaps: %w", err)
	}

	// Filter mindmaps based on user permissions
	var allowedMindmaps []*model.Mindmap
	for _, mindmap := range mindmaps {
		if mindmap.Owner == user.Username || mindmap.IsPublic {
			allowedMindmaps = append(allowedMindmaps, mindmap)
		}
	}

	return allowedMindmaps, nil
}

// MindmapUpdate updates an existing mindmap's information
func (mm *MindmapManager) MindmapUpdate(user *model.User, mindmap *model.Mindmap, mindmapUpdateInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) error {
	// Check if the user has permission to update the mindmap
	permission, err := mm.MindmapPermission(user, model.MindmapInfo{ID: mindmap.ID})
	if err != nil {
		return fmt.Errorf("failed to check mindmap permission: %w", err)
	}
	if permission < 2 { // Only owner (permission level 2) can update
		return fmt.Errorf("user %s does not have permission to update mindmap %s", user.Username, mindmap.Name)
	}

	// Store old values for potential rollback and event
	oldName := mindmap.Name
	oldIsPublic := mindmap.IsPublic

	// Update mindmap fields based on the filter
	if mindmapFilter.Name && mindmapUpdateInfo.Name != "" {
		mindmap.Name = mindmapUpdateInfo.Name
	}
	if mindmapFilter.IsPublic {
		mindmap.IsPublic = mindmapUpdateInfo.IsPublic
	}

	// Update in storage
	err = mm.mindmapStore.MindmapUpdate(mindmap, mindmapUpdateInfo, mindmapFilter)
	if err != nil {
		// Rollback changes if storage update fails
		mindmap.Name = oldName
		mindmap.IsPublic = oldIsPublic
		return fmt.Errorf("failed to update mindmap in storage: %w", err)
	}

	// Publish MindmapUpdated event
	mm.eventManager.Publish(event.Event{
		Type: event.MindmapUpdated,
		Data: map[string]interface{}{
			"mindmap":     mindmap,
			"oldName":     oldName,
			"oldIsPublic": oldIsPublic,
		},
	})

	return nil
}

// MindmapDelete removes a mindmap and all its associated data.
func (mm *MindmapManager) MindmapDelete(user *model.User, mindmap *model.Mindmap) error {
	if mindmap.Owner != user.Username {
		return fmt.Errorf("user %s does not have permission to delete %s mindmap", user.Username, mindmap.Name)
	}

	// Delete the mindmap from storage
	err := mm.mindmapStore.MindmapDelete(mindmap)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap: %w", err)
	}
	return nil
}

// MindmapToInfo extracts MindmapInfo from a Mindmap instance
func (mm *MindmapManager) MindmapToInfo(mindmap *model.Mindmap) model.MindmapInfo {
	var nodeCount *int
	var depth *int

	if mindmap.Nodes != nil {
		count := len(mindmap.Nodes)
		nodeCount = &count

		// Calculate depth
		depthValue := mm.calculateMindmapDepth(mindmap.Root)
		depth = &depthValue
	}

	return model.MindmapInfo{
		ID:        mindmap.ID,
		Name:      mindmap.Name,
		Owner:     mindmap.Owner,
		IsPublic:  mindmap.IsPublic,
		NodeCount: nodeCount,
		Depth:     depth,
	}
}

// calculateMindmapDepth computes the maximum depth of the mindmap tree structure
func (mm *MindmapManager) calculateMindmapDepth(root *model.Node) int {
	if root == nil {
		return 0
	}
	maxDepth := 0
	for _, child := range root.Children {
		depth := mm.calculateMindmapDepth(child)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth + 1
}

// handleUserDeleted deletes all mindmaps associated with the deleted user
func (mm *MindmapManager) handleUserDeleted(e event.Event) {
	user, ok := e.Data.(*model.User)
	if !ok {
		mm.logger.LogError(fmt.Errorf("Invalid event data for user delete event"))
		return
	}

	// Get all mindmaps owned by the user
	mindmaps, err := mm.MindmapGet(user, model.MindmapInfo{Owner: user.Username}, model.MindmapFilter{Owner: true})
	if err != nil {
		mm.logger.LogError(fmt.Errorf("Failed to get mindmaps for deleted user %s: %v", user.Username, err))
		return
	}

	// Delete each mindmap
	for _, mindmap := range mindmaps {
		err := mm.MindmapDelete(user, mindmap)
		if err != nil {
			mm.logger.LogError(fmt.Errorf("Failed to delete mindmap %d for deleted user %s: %v", mindmap.ID, user.Username, err))
		}
	}
}

// handleRootNodeRenamed updates the mindmap name when the root node is renamed
func (mm *MindmapManager) handleRootNodeRenamed(e event.Event) {
	data, ok := e.Data.(map[string]interface{})
	if !ok {
		mm.logger.LogError(fmt.Errorf("Invalid event data for root node rename event"))
		return
	}

	mindmapID, ok := data["mindmapID"].(int)
	if !ok {
		mm.logger.LogError(fmt.Errorf("Invalid mindmap ID in root node rename event"))
		return
	}

	newName, ok := data["newName"].(string)
	if !ok {
		mm.logger.LogError(fmt.Errorf("Invalid new name in root node rename event"))
		return
	}

	// Prevent setting an empty name for the mindmap
	if newName == "" {
		mm.logger.LogError(fmt.Errorf("Attempted to set empty name for mindmap %d", mindmapID))
		return
	}

	oldName, ok := data["oldName"].(string)
	if !ok {
		mm.logger.LogError(fmt.Errorf("Invalid old name in root node rename event"))
		return
	}

	// Get the mindmap
	mindmaps, err := mm.MindmapGet(nil, model.MindmapInfo{ID: mindmapID}, model.MindmapFilter{ID: true})
	if err != nil || len(mindmaps) == 0 {
		mm.logger.LogError(fmt.Errorf("Failed to get mindmap %d: %v", mindmapID, err))
		return
	}
	mindmap := mindmaps[0]

	// Update the mindmap name
	err = mm.MindmapUpdate(nil, mindmap, model.MindmapInfo{Name: newName}, model.MindmapFilter{Name: true})
	if err != nil {
		mm.logger.LogError(fmt.Errorf("Failed to update mindmap name for mindmap %d: %v", mindmapID, err))
		return
	}

	// Publish MindmapUpdated event
	mm.eventManager.Publish(event.Event{
		Type: event.MindmapUpdated,
		Data: map[string]interface{}{
			"mindmap": mindmap,
			"oldName": oldName,
		},
	})
}
