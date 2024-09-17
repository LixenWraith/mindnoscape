// Package data provides data management functionality for the Mindnoscape application.
// It coordinates operations between user, mindmap, and node managers.
package data

import (
	"context"
	"fmt"

	"mindnoscape/local-app/src/pkg/event"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
	"mindnoscape/local-app/src/pkg/storage"
)

// DataManager is the main struct that coordinates all data operations
type DataManager struct {
	UserManager    *UserManager
	MindmapManager *MindmapManager
	NodeManager    *NodeManager
	EventManager   *event.EventManager
	Config         *model.Config
	Logger         *log.Logger
}

// NewDataManager creates a new Manager instance
func NewDataManager(store *storage.Storage, cfg *model.Config, logger *log.Logger) (*DataManager, error) {
	ctx := context.Background()
	logger.Info(ctx, "Creating new DataManager", nil)

	eventManager := event.NewEventManager(logger)
	m := &DataManager{
		EventManager: eventManager,
		Config:       cfg,
		Logger:       logger,
	}

	// Initialize UserManager
	var err error
	m.UserManager, err = NewUserManager(store.UserStore, eventManager, logger)
	if err != nil {
		logger.Error(ctx, "Failed to create UserManager", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to create UserManager: %w", err)
	}

	// Initialize MindmapManager
	m.MindmapManager, err = NewMindmapManager(store.MindmapStore, eventManager, logger)
	if err != nil {
		logger.Error(ctx, "Failed to create MindmapManager", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to create MindmapManager: %w", err)
	}

	// Initialize NodeManager
	m.NodeManager, err = NewNodeManager(store.NodeStore, eventManager, logger)
	if err != nil {
		logger.Error(ctx, "Failed to create NodeManager", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to create NodeManager: %w", err)
	}

	// Handle default user logic
	if cfg.DefaultUserActive {
		logger.Debug(ctx, "Handling default user logic", nil)
		defaultUserInfo := model.UserInfo{Username: cfg.DefaultUser}
		exists, err := m.UserManager.UserGet(defaultUserInfo, model.UserFilter{Username: true})
		if err != nil {
			logger.Error(ctx, "Failed to check default user existence", log.Fields{"error": err})
			return nil, fmt.Errorf("failed to check default user existence: %w", err)
		}
		if len(exists) == 0 {
			defaultUserInfo.PasswordHash = []byte(cfg.DefaultUserPassword)
			_, err = m.UserManager.UserAdd(defaultUserInfo)
			if err != nil {
				logger.Error(ctx, "Failed to create default user", log.Fields{"error": err})
				return nil, fmt.Errorf("failed to create default user: %w", err)
			}
			logger.Info(ctx, "Default user created", nil)
		} else {
			logger.Info(ctx, "Default user already exists", nil)
		}
	}

	// Subscribe MindmapManager to UserDeleted events
	eventManager.Subscribe(event.UserDeleted, m.MindmapManager.handleUserDeleted)

	// Subscribe NodeManager to MindmapCreated events
	eventManager.Subscribe(event.MindmapAdded, m.NodeManager.handleMindmapAdded)

	// Subscribe to MindmapDeleted events
	eventManager.Subscribe(event.MindmapDeleted, m.NodeManager.handleMindmapDeleted)

	// Subscribe to MindmapUpdated events
	eventManager.Subscribe(event.MindmapUpdated, m.NodeManager.handleMindmapUpdated)

	// Subscribe to MindmapSelected events
	eventManager.Subscribe(event.MindmapSelected, m.NodeManager.handleMindmapSelected)

	return m, nil
}

// MindmapExport exports a mindmap to a file in the specified format.
func (m *DataManager) MindmapExport(user *model.User, mindmap *model.Mindmap, filename, format string) error {
	ctx := context.Background()
	m.Logger.Info(ctx, "Exporting mindmap", log.Fields{"user": user.Username, "mindmapID": mindmap.ID, "filename": filename, "format": format})

	err := storage.FileExport(mindmap, filename, format, m.Logger)
	if err != nil {
		m.Logger.Error(ctx, "Failed to export mindmap", log.Fields{"error": err, "mindmapID": mindmap.ID})
		return fmt.Errorf("failed to export mindmap: %w", err)
	}

	m.Logger.Info(ctx, "Mindmap exported successfully", log.Fields{"mindmapID": mindmap.ID})
	return nil
}

// MindmapImport imports a mindmap from a file in the specified format.
func (m *DataManager) MindmapImport(user *model.User, filename, format string) (*model.Mindmap, error) {
	ctx := context.Background()
	m.Logger.Info(ctx, "Importing mindmap", log.Fields{"user": user.Username, "filename": filename, "format": format})

	// Import the mindmap
	importedMindmap, err := storage.FileImport(filename, format, m.Logger)
	if err != nil {
		m.Logger.Error(ctx, "Failed to import mindmap", log.Fields{"error": err, "filename": filename})
		return nil, fmt.Errorf("failed to import mindmap: %w", err)
	}

	// Validate the imported mindmap structure
	if err := m.validateMindmap(importedMindmap); err != nil {
		m.Logger.Error(ctx, "Invalid mindmap structure", log.Fields{"error": err})
		return nil, fmt.Errorf("invalid mindmap structure: %w", err)
	}

	// Check if a mindmap with the same name exists for the user
	existingMindmaps, err := m.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: importedMindmap.Name}, model.MindmapFilter{Name: true})
	if err != nil {
		m.Logger.Error(ctx, "Failed to check for existing mindmap", log.Fields{"error": err, "mindmapName": importedMindmap.Name})
		return nil, fmt.Errorf("failed to check for existing mindmap: %w", err)
	}

	if len(existingMindmaps) > 0 {
		m.Logger.Debug(ctx, "Existing mindmap found, deleting", log.Fields{"mindmapName": importedMindmap.Name})
		// Delete existing mindmap
		err = m.MindmapManager.MindmapDelete(user, existingMindmaps[0])
		if err != nil {
			m.Logger.Error(ctx, "Failed to delete existing mindmap", log.Fields{"error": err, "mindmapName": importedMindmap.Name})
			return nil, fmt.Errorf("failed to delete existing mindmap: %w", err)
		}
	}

	// Add the new mindmap
	importedMindmap.Owner = user.Username
	newMindmapID, err := m.MindmapManager.MindmapAdd(user, model.MindmapInfo{
		Name:     importedMindmap.Name,
		IsPublic: importedMindmap.IsPublic,
	})
	if err != nil {
		m.Logger.Error(ctx, "Failed to add imported mindmap", log.Fields{"error": err, "mindmapName": importedMindmap.Name})
		return nil, fmt.Errorf("failed to add imported mindmap: %w", err)
	}
	importedMindmap.ID = newMindmapID

	// Add nodes
	for _, node := range importedMindmap.Nodes {
		m.Logger.Debug(ctx, "Adding node to imported mindmap", log.Fields{"nodeID": node.ID, "nodeName": node.Name})
		_, _, err := m.NodeManager.NodeAdd(importedMindmap, m.NodeManager.NodeToInfo(node), true)
		if err != nil {
			// Rollback: delete the newly added mindmap
			m.Logger.Error(ctx, "Failed to add node, rolling back", log.Fields{"error": err, "nodeID": node.ID})
			m.MindmapManager.MindmapDelete(user, importedMindmap)
			return nil, fmt.Errorf("failed to add node: %w", err)
		}
	}

	m.Logger.Info(ctx, "Mindmap imported successfully", log.Fields{"mindmapID": importedMindmap.ID, "mindmapName": importedMindmap.Name})
	return importedMindmap, nil
}

// validateMindmap checks the imported mindmap structure for validity.
func (mm *DataManager) validateMindmap(mindmap *model.Mindmap) error {
	ctx := context.Background()
	mm.Logger.Debug(ctx, "Validating mindmap structure", log.Fields{"mindmapID": mindmap.ID})

	// Check root node
	if mindmap.Root == nil || mindmap.Root.ID != 0 || mindmap.Root.ParentID != -1 || mindmap.Root.Index != "0" {
		mm.Logger.Warn(ctx, "Invalid root node structure", log.Fields{"rootNode": mindmap.Root})
		return fmt.Errorf("invalid root node structure")
	}

	nodeIDs := make(map[int]bool)
	nodeIDs[0] = true // Root node

	// First pass: Check for duplicate IDs
	var checkDuplicateIDs func(*model.Node) error
	checkDuplicateIDs = func(node *model.Node) error {
		if node.ID != 0 {
			if nodeIDs[node.ID] {
				mm.Logger.Warn(ctx, "Duplicate node ID found", log.Fields{"nodeID": node.ID})
				return fmt.Errorf("duplicate node ID: %d", node.ID)
			}
			nodeIDs[node.ID] = true
		}
		for _, child := range node.Children {
			if err := checkDuplicateIDs(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := checkDuplicateIDs(mindmap.Root); err != nil {
		return err
	}

	// Second pass: Validate parent IDs
	var validateParentIDs func(*model.Node) error
	validateParentIDs = func(node *model.Node) error {
		if node.ID != 0 {
			if !nodeIDs[node.ParentID] {
				mm.Logger.Warn(ctx, "Invalid parent ID", log.Fields{"nodeID": node.ID, "parentID": node.ParentID})
				return fmt.Errorf("invalid parent ID for node %d: parent %d not found", node.ID, node.ParentID)
			}
		}
		for _, child := range node.Children {
			if err := validateParentIDs(child); err != nil {
				return err
			}
		}
		return nil
	}
	err := validateParentIDs(mindmap.Root)
	if err != nil {
		return err
	}

	mm.Logger.Debug(ctx, "Mindmap structure validated successfully", nil)
	return nil
}

func buildTreeFromNodes(nodes []*model.Node) *model.Node {
	nodeMap := make(map[int]*model.Node)
	var root *model.Node

	// First pass: build the map
	for _, node := range nodes {
		nodeMap[node.ID] = node
		if node.ParentID == -1 {
			root = node
		}
	}

	// Second pass: build the tree
	for _, node := range nodes {
		if node.ParentID != -1 {
			parent := nodeMap[node.ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	return root
}
