// Package data provides data management functionality for the Mindnoscape application.
// It coordinates operations between user, mindmap, and node managers.
package data

import (
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
func NewDataManager(userStore storage.UserStore, mindmapStore storage.MindmapStore, nodeStore storage.NodeStore, cfg *model.Config, logger *log.Logger) (*DataManager, error) {
	eventManager := event.NewEventManager()
	m := &DataManager{
		EventManager: eventManager,
		Config:       cfg,
		Logger:       logger,
	}

	// Initialize UserManager
	var err error
	m.UserManager, err = NewUserManager(userStore, eventManager, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create UserManager: %w", err)
	}

	// Initialize MindmapManager
	m.MindmapManager, err = NewMindmapManager(mindmapStore, eventManager, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MindmapManager: %w", err)
	}

	// Initialize NodeManager
	m.NodeManager, err = NewNodeManager(nodeStore, eventManager, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create NodeManager: %w", err)
	}

	// Handle default user logic
	if cfg.DefaultUserActive {
		defaultUserInfo := model.UserInfo{Username: cfg.DefaultUser}
		exists, err := m.UserManager.UserGet(defaultUserInfo, model.UserFilter{Username: true})
		if err != nil {
			return nil, fmt.Errorf("failed to check default user existence: %w", err)
		}
		if len(exists) == 0 {
			defaultUserInfo.PasswordHash = []byte(cfg.DefaultUserPassword)
			_, err = m.UserManager.UserAdd(defaultUserInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to create default user: %w", err)
			}
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
	err := storage.FileExport(mindmap, filename, format)
	if err != nil {
		return fmt.Errorf("failed to export mindmap: %w", err)
	}

	return nil
}

// MindmapImport imports a mindmap from a file in the specified format.
func (m *DataManager) MindmapImport(user *model.User, filename, format string) (*model.Mindmap, error) {
	// Import the mindmap
	importedMindmap, err := storage.FileImport(filename, format)
	if err != nil {
		return nil, fmt.Errorf("failed to import mindmap: %w", err)
	}

	// Validate the imported mindmap structure
	if err := m.validateMindmap(importedMindmap); err != nil {
		return nil, fmt.Errorf("invalid mindmap structure: %w", err)
	}

	// Check if a mindmap with the same name exists for the user
	existingMindmaps, err := m.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: importedMindmap.Name}, model.MindmapFilter{Name: true})
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing mindmap: %w", err)
	}

	if len(existingMindmaps) > 0 {
		// Delete existing mindmap
		err = m.MindmapManager.MindmapDelete(user, existingMindmaps[0])
		if err != nil {
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
		return nil, fmt.Errorf("failed to add imported mindmap: %w", err)
	}
	importedMindmap.ID = newMindmapID

	// Add nodes
	for _, node := range importedMindmap.Nodes {
		_, _, err := m.NodeManager.NodeAdd(importedMindmap, m.NodeManager.NodeToInfo(node), true)
		if err != nil {
			// Rollback: delete the newly added mindmap
			m.MindmapManager.MindmapDelete(user, importedMindmap)
			return nil, fmt.Errorf("failed to add node: %w", err)
		}
	}

	return importedMindmap, nil
}

// validateMindmap checks the imported mindmap structure for validity.
func (mm *DataManager) validateMindmap(mindmap *model.Mindmap) error {
	// Check root node
	if mindmap.Root == nil || mindmap.Root.ID != 0 || mindmap.Root.ParentID != -1 || mindmap.Root.Index != "0" {
		return fmt.Errorf("invalid root node structure")
	}

	nodeIDs := make(map[int]bool)
	nodeIDs[0] = true // Root node

	// First pass: Check for duplicate IDs
	var checkDuplicateIDs func(*model.Node) error
	checkDuplicateIDs = func(node *model.Node) error {
		if node.ID != 0 {
			if nodeIDs[node.ID] {
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
	return validateParentIDs(mindmap.Root)
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
