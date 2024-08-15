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
	err := m.UserManager.UserSelect(username)
	if err != nil {
		return fmt.Errorf("failed to change user: %w", err)
	}

	m.CurrentUser = username
	m.MindmapManager.CurrentUser = username
	return nil
}

func (m *Manager) MindmapAdd(name string, isPublic bool) error {
	return m.MindmapManager.MindmapAdd(name, isPublic)
}

func (m *Manager) MindmapDelete(name string) error {
	return m.MindmapManager.MindmapDelete(name)
}

func (m *Manager) MindmapSelect(name string) error {
	return m.MindmapManager.MindmapSelect(name)
}

func (m *Manager) NodeAdd(parentIdentifier string, content string, extra map[string]string, useIndex bool) error {
	return m.NodeManager.NodeAdd(parentIdentifier, content, extra, useIndex)
}

func (m *Manager) NodeDelete(identifier string, useIndex bool) error {
	return m.NodeManager.NodeDelete(identifier, useIndex)
}

func (m *Manager) NodeModify(identifier string, content string, extra map[string]string, useIndex bool) error {
	return m.NodeManager.NodeModify(identifier, content, extra, useIndex)
}

func (m *Manager) NodeMove(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	return m.NodeManager.NodeMove(sourceIdentifier, targetIdentifier, useIndex)
}

func (m *Manager) SystemUndo() error {
	return m.HistoryManager.Undo()
}

func (m *Manager) SystemRedo() error {
	return m.HistoryManager.Redo()
}

func (m *Manager) MindmapExport(filename, format string) error {
	return m.MindmapManager.MindmapExport(filename, format)
}

func (m *Manager) MindmapImport(filename, format string) error {
	return m.MindmapManager.MindmapImport(filename, format)
}

func (m *Manager) MindmapList() ([]storage.MindmapInfo, error) {
	return m.MindmapManager.MindmapList()
}

func (m *Manager) ShowMindmap(logicalIndex string, showIndex bool) ([]string, error) {
	return m.MindmapManager.MindmapView(logicalIndex, showIndex)
}

func (m *Manager) MindmapPermission(name string, username string, setPublic ...bool) (bool, error) {
	return m.MindmapManager.MindmapPermission(name, username, setPublic...)
}
