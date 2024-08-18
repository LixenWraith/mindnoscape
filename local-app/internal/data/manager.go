// Package data provides data management functionality for the Mindnoscape application.
// It coordinates operations between user, mindmap, and node managers.
package data

import (
	"fmt"

	"mindnoscape/local-app/internal/config"
	"mindnoscape/local-app/internal/storage"
)

// Manager is the main struct that coordinates all data operations
type Manager struct {
	UserManager    *UserManager
	MindmapManager *MindmapManager
	NodeManager    *NodeManager
	HistoryManager *HistoryManager

	Config *config.Config
}

// NewManager creates a new Manager instance
func NewManager(userStore storage.UserStore, mindmapStore storage.MindmapStore, nodeStore storage.NodeStore, cfg *config.Config) (*Manager, error) {
	m := &Manager{
		Config: cfg,
	}

	// Initialize MindmapManager
	var err error
	m.MindmapManager, err = NewMindmapManager(mindmapStore, nodeStore, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create MindmapManager: %w", err)
	}

	// Initialize UserManager
	// TODO: UserManger dependence on MindmapManager is due to the old program structure, interface and logic to be updated
	m.UserManager, err = NewUserManager(userStore, cfg, m.MindmapManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create UserManager: %w", err)
	}

	// Initialize NodeManager and HistoryManager
	m.NodeManager = NewNodeManager(m.MindmapManager)
	m.HistoryManager = NewHistoryManager(m.NodeManager)

	// Handle default user logic
	if cfg.DefaultUserActive {
		exists, err := m.UserManager.UserExists(cfg.DefaultUser)
		if err != nil {
			return nil, fmt.Errorf("failed to check default user existence: %w", err)
		}
		if !exists {
			err = m.UserManager.UserAdd(cfg.DefaultUser, cfg.DefaultUserPassword)
			if err != nil {
				return nil, fmt.Errorf("failed to create default user: %w", err)
			}
		}
	}

	return m, nil
}
