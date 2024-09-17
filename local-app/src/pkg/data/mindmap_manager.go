// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to mindmap management.
package data

import (
	"context"
	"fmt"

	"mindnoscape/local-app/src/pkg/event"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
	"mindnoscape/local-app/src/pkg/storage"
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
	ctx := context.Background()
	logger.Info(ctx, "Creating new MindmapManager", nil)

	if mindmapStore == nil {
		logger.Error(ctx, "MindmapStore not initialized", nil)
		return nil, fmt.Errorf("mindmapStore not initialized")
	}
	if eventManager == nil {
		logger.Error(ctx, "EventManager not initialized", nil)
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

	logger.Info(ctx, "MindmapManager created successfully", nil)
	return mm, nil
}

// MindmapAdd creates a new mindmap with the given name and visibility.
func (mm *MindmapManager) MindmapAdd(user *model.User, newMindmapInfo model.MindmapInfo) (int, error) {
	ctx := context.Background()
	mm.logger.Info(ctx, "Adding new mindmap", log.Fields{"username": user.Username, "mindmapName": newMindmapInfo.Name})

	// Check if the user already has a mindmap with the same name
	existingMindmaps, err := mm.MindmapGet(user, newMindmapInfo, model.MindmapFilter{Name: true})
	if err != nil {
		mm.logger.Error(ctx, "Failed to check for existing mindmap", log.Fields{"error": err, "mindmapName": newMindmapInfo.Name})
		return 0, fmt.Errorf("failed to check for existing mindmap: %w", err)
	}
	if len(existingMindmaps) > 0 {
		mm.logger.Warn(ctx, "Mindmap with the same name already exists", log.Fields{"mindmapName": newMindmapInfo.Name})
		return 0, fmt.Errorf("mindmap with name '%s' already exists for this user", newMindmapInfo.Name)
	}

	// Add the new mindmap to storage
	newMindmapInfo.Owner = user.Username
	id, err := mm.mindmapStore.MindmapAdd(user, newMindmapInfo)
	if err != nil {
		mm.logger.Error(ctx, "Failed to add mindmap", log.Fields{"error": err, "mindmapName": newMindmapInfo.Name})
		return 0, fmt.Errorf("failed to add mindmap: %w", err)
	}
	newMindmapInfo.ID = id

	mindmaps, err := mm.MindmapGet(user, newMindmapInfo, model.MindmapFilter{ID: true, Owner: true})
	if err != nil {
		mm.logger.Error(ctx, "Failed to get the new mindmap", log.Fields{"error": err, "mindmapID": id})
		return 0, fmt.Errorf("failed to get the new mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		mm.logger.Error(ctx, "Could not find the new mindmap", log.Fields{"mindmapID": id})
		return 0, fmt.Errorf("could not find the new mindmap")
	}
	newMindmap := mindmaps[0]

	mm.logger.Debug(ctx, "Added mindmap", log.Fields{"mindmapID": newMindmap.ID, "mindmapName": newMindmap.Name})
	mm.logger.Debug(ctx, "New mindmap details", log.Fields{"mindmap": newMindmap})

	// Initialize the Nodes map
	newMindmap.Nodes = make(map[int]*model.Node)

	// Publish MindmapCreated event
	mm.eventManager.Publish(event.Event{
		Type: event.MindmapAdded,
		Data: newMindmap,
	})
	mm.logger.Debug(ctx, "Published MindmapAdded event", log.Fields{"mindmapID": id})

	mm.logger.Info(ctx, "Mindmap added successfully", log.Fields{"mindmapID": id, "mindmapName": newMindmapInfo.Name})
	return id, nil
}

// MindmapPermission checks the permission of a user for a specific mindmap
func (mm *MindmapManager) MindmapPermission(user *model.User, mindmapInfo model.MindmapInfo) (int, error) {
	ctx := context.Background()
	mm.logger.Info(ctx, "Checking mindmap permission", log.Fields{"username": user.Username, "mindmapID": mindmapInfo.ID})

	// Get the mindmap
	mindmaps, err := mm.MindmapGet(user, mindmapInfo, model.MindmapFilter{ID: true, Owner: true, IsPublic: true})
	if err != nil {
		mm.logger.Error(ctx, "Failed to get mindmap", log.Fields{"error": err, "mindmapID": mindmapInfo.ID})
		return 0, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		mm.logger.Warn(ctx, "Mindmap not found", log.Fields{"mindmapID": mindmapInfo.ID})
		return 0, fmt.Errorf("mindmap not found")
	}
	mindmap := mindmaps[0]

	// Check ownership
	if mindmap.Owner == user.Username {
		mm.logger.Debug(ctx, "User is the owner of the mindmap", log.Fields{"username": user.Username, "mindmapID": mindmap.ID})
		return 2, nil // Owner has full permission
	}

	// Check if the mindmap is public
	if mindmap.IsPublic {
		mm.logger.Debug(ctx, "Mindmap is public", log.Fields{"mindmapID": mindmap.ID})
		return 1, nil // Public mindmap, read-only permission
	}

	// User is not the owner and the mindmap is not public
	mm.logger.Debug(ctx, "User has no permission for the mindmap", log.Fields{"username": user.Username, "mindmapID": mindmap.ID})
	return 0, nil // No permission
}

// MindmapGet retrieves the information of mindmaps based on provided filters and user permissions.
func (mm *MindmapManager) MindmapGet(user *model.User, mindmapInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) ([]*model.Mindmap, error) {
	ctx := context.Background()
	mm.logger.Info(ctx, "Retrieving mindmaps", log.Fields{"username": user.Username, "filter": mindmapFilter})

	// Get all mindmaps that match the filter
	mindmaps, err := mm.mindmapStore.MindmapGet(user, mindmapInfo, mindmapFilter)
	if err != nil {
		mm.logger.Error(ctx, "Failed to get mindmaps", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to get mindmaps: %w", err)
	}

	// Filter mindmaps based on user permissions
	var allowedMindmaps []*model.Mindmap
	for _, mindmap := range mindmaps {
		if mindmap.Owner == user.Username || mindmap.IsPublic {
			allowedMindmaps = append(allowedMindmaps, mindmap)
		}
	}

	mm.logger.Info(ctx, "Mindmaps retrieved successfully", log.Fields{"count": len(allowedMindmaps)})
	return allowedMindmaps, nil
}

// MindmapUpdate updates an existing mindmap's information
func (mm *MindmapManager) MindmapUpdate(user *model.User, mindmap *model.Mindmap, mindmapUpdateInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) error {
	ctx := context.Background()
	mm.logger.Info(ctx, "Updating mindmap", log.Fields{"username": user.Username, "mindmapID": mindmap.ID})

	// Check if the user has permission to update the mindmap
	permission, err := mm.MindmapPermission(user, model.MindmapInfo{ID: mindmap.ID})
	if err != nil {
		mm.logger.Error(ctx, "Failed to check mindmap permission", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return fmt.Errorf("failed to check mindmap permission: %w", err)
	}
	if permission < 2 { // Only owner (permission level 2) can update
		mm.logger.Warn(ctx, "User does not have permission to update mindmap", log.Fields{"username": user.Username, "mindmapID": mindmap.ID})
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
		mm.logger.Error(ctx, "Failed to update mindmap in storage", log.Fields{"error": err, "mindmapID": mindmap.ID})
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

	mm.logger.Info(ctx, "Mindmap updated successfully", log.Fields{"mindmapID": mindmap.ID})
	return nil
}

// MindmapDelete removes a mindmap and all its associated data.
func (mm *MindmapManager) MindmapDelete(user *model.User, mindmap *model.Mindmap) error {
	ctx := context.Background()
	mm.logger.Info(ctx, "Deleting mindmap", log.Fields{"username": user.Username, "mindmapID": mindmap.ID})

	if mindmap.Owner != user.Username {
		mm.logger.Warn(ctx, "User does not have permission to delete mindmap", log.Fields{"username": user.Username, "mindmapID": mindmap.ID})
		return fmt.Errorf("user %s does not have permission to delete %s mindmap", user.Username, mindmap.Name)
	}

	// Delete the mindmap from storage
	err := mm.mindmapStore.MindmapDelete(mindmap)
	if err != nil {
		mm.logger.Error(ctx, "Failed to delete mindmap", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return fmt.Errorf("failed to delete mindmap: %w", err)
	}

	mm.logger.Info(ctx, "Mindmap deleted successfully", log.Fields{"mindmapID": mindmap.ID})
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
	ctx := context.Background()
	mm.logger.Info(ctx, "Handling UserDeleted event", nil)

	user, ok := e.Data.(*model.User)
	if !ok {
		mm.logger.Error(ctx, "Invalid event data for user delete event", nil)
		return
	}

	// Get all mindmaps owned by the user
	mindmaps, err := mm.MindmapGet(user, model.MindmapInfo{Owner: user.Username}, model.MindmapFilter{Owner: true})
	if err != nil {
		mm.logger.Error(ctx, "Failed to get mindmaps for deleted user", log.Fields{"error": err, "username": user.Username})
		return
	}

	// Delete each mindmap
	for _, mindmap := range mindmaps {
		err := mm.MindmapDelete(user, mindmap)
		if err != nil {
			mm.logger.Error(ctx, "Failed to delete mindmap for deleted user", log.Fields{"error": err, "username": user.Username, "mindmapID": mindmap.ID})
		}
	}

	mm.logger.Info(ctx, "Finished handling UserDeleted event", log.Fields{"username": user.Username, "deletedMindmaps": len(mindmaps)})
}

// handleRootNodeRenamed updates the mindmap name when the root node is renamed
func (mm *MindmapManager) handleRootNodeRenamed(e event.Event) {
	ctx := context.Background()
	mm.logger.Info(ctx, "Handling RootNodeRenamed event", nil)

	data, ok := e.Data.(map[string]interface{})
	if !ok {
		mm.logger.Error(ctx, "Invalid event data for root node rename event", nil)
		return
	}

	mindmapID, ok := data["mindmapID"].(int)
	if !ok {
		mm.logger.Error(ctx, "Invalid mindmap ID in root node rename event", nil)
		return
	}

	newName, ok := data["newName"].(string)
	if !ok {
		mm.logger.Error(ctx, "Invalid new name in root node rename event", nil)
		return
	}

	// Prevent setting an empty name for the mindmap
	if newName == "" {
		mm.logger.Warn(ctx, "Attempted to set empty name for mindmap", log.Fields{"mindmapID": mindmapID})
		return
	}

	oldName, ok := data["oldName"].(string)
	if !ok {
		mm.logger.Error(ctx, "Invalid old name in root node rename event", nil)
		return
	}

	// Get the mindmap
	mindmaps, err := mm.MindmapGet(nil, model.MindmapInfo{ID: mindmapID}, model.MindmapFilter{ID: true})
	if err != nil || len(mindmaps) == 0 {
		mm.logger.Error(ctx, "Failed to get mindmap", log.Fields{"error": err, "mindmapID": mindmapID})
		return
	}
	mindmap := mindmaps[0]

	// Update the mindmap name
	err = mm.MindmapUpdate(nil, mindmap, model.MindmapInfo{Name: newName}, model.MindmapFilter{Name: true})
	if err != nil {
		mm.logger.Error(ctx, "Failed to update mindmap name", log.Fields{"error": err, "mindmapID": mindmapID})
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

	mm.logger.Info(ctx, "Finished handling RootNodeRenamed event", log.Fields{"mindmapID": mindmapID, "newName": newName})
}
