package data

import (
	"fmt"
	"mindnoscape/local-app/internal/storage"
)

const (
	ColorYellow    = "{{yellow}}"
	ColorOrange    = "{{orange}}"
	ColorDarkBrown = "{{darkbrown}}"
	ColorDefault   = "{{default}}"
)

// Manager is the main struct that coordinates all data operations
type Manager struct {
	Store          storage.Store
	UserManager    *UserManager
	MindmapManager *MindmapManager
	NodeManager    *NodeManager
	HistoryManager *HistoryManager
	CurrentUser    string
}

// NewManager creates a new Manager instance
func NewManager(store storage.Store) (*Manager, error) {
	m := &Manager{
		Store:       store,
		CurrentUser: "guest",
	}

	var err error
	m.MindmapManager, err = NewMindmapManager(store, m.CurrentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create MindmapManager: %w", err)
	}

	m.HistoryManager = NewHistoryManager(m.MindmapManager)
	m.UserManager = NewUserManager(m.MindmapManager)
	m.NodeManager = NewNodeManager(m.MindmapManager)

	m.MindmapManager.HistoryManager = m.HistoryManager

	return m, nil
}

// ChangeUser changes the current user and updates all managers
func (m *Manager) ChangeUser(username string) error {
	err := m.UserManager.ChangeUser(username)
	if err != nil {
		return fmt.Errorf("failed to change user: %w", err)
	}

	m.CurrentUser = username
	m.MindmapManager.CurrentUser = username
	return nil
}

// CreateMindmap creates a new data
func (m *Manager) CreateMindmap(name string, isPublic bool) error {
	return m.MindmapManager.AddMindmap(name, isPublic)
}

// DeleteMindmap deletes an existing data
func (m *Manager) DeleteMindmap(name string) error {
	return m.MindmapManager.DeleteMindmap(name)
}

// ChangeMindmap switches to a different data
func (m *Manager) ChangeMindmap(name string) error {
	return m.MindmapManager.ChangeMindmap(name)
}

// AddNode adds a new node to the current data
func (m *Manager) AddNode(parentIdentifier string, content string, extra map[string]string, useIndex bool) error {
	return m.NodeManager.AddNode(parentIdentifier, content, extra, useIndex)
}

// DeleteNode deletes a node from the current data
func (m *Manager) DeleteNode(identifier string, useIndex bool) error {
	return m.NodeManager.DeleteNode(identifier, useIndex)
}

// ModifyNode modifies an existing node in the current data
func (m *Manager) ModifyNode(identifier string, content string, extra map[string]string, useIndex bool) error {
	return m.NodeManager.ModifyNode(identifier, content, extra, useIndex)
}

// MoveNode moves a node to a new parent in the current data
func (m *Manager) MoveNode(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	return m.NodeManager.MoveNode(sourceIdentifier, targetIdentifier, useIndex)
}

// Undo undoes the last operation
func (m *Manager) Undo() error {
	return m.HistoryManager.Undo()
}

// Redo redoes the last undone operation
func (m *Manager) Redo() error {
	return m.HistoryManager.Redo()
}

// SaveMindmap saves the current data to a file
func (m *Manager) SaveMindmap(filename, format string) error {
	return m.MindmapManager.SaveMindmap(filename, format)
}

// LoadMindmap loads a data from a file
func (m *Manager) LoadMindmap(filename, format string) error {
	return m.MindmapManager.LoadMindmap(filename, format)
}

// ListMindmaps returns a list of all mindmaps for the current user
func (m *Manager) ListMindmaps() ([]storage.MindmapInfo, error) {
	return m.MindmapManager.ListMindmap()
}

// ShowMindmap returns a visual representation of the current data
func (m *Manager) ShowMindmap(logicalIndex string, showIndex bool) ([]string, error) {
	return m.MindmapManager.ShowMindmap(logicalIndex, showIndex)
}

// ModifyMindmapAccess changes the access level of a data
func (m *Manager) ModifyMindmapAccess(name string, isPublic bool) error {
	return m.MindmapManager.ModifyMindmapAccess(name, isPublic)
}
