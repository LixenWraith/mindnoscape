package storage

import (
	"mindnoscape/local-app/internal/models"
)

type MindmapInfo struct {
	Name     string
	IsPublic bool
	Owner    string
}

type Store interface {
	// User operations
	EnsureGuestUser() error
	UserAdd(username, hashedPassword string) error
	UserDelete(username string) error
	UserExists(username string) (bool, error)
	UserGet(username string) (*models.User, error)
	UserModify(oldUsername, newUsername, newHashedPassword string) error
	UserAuthenticate(username, password string) (bool, error)

	// Mindmap operations
	MindmapAdd(mindmapName string, owner string, isPublic bool) (int, error)
	MindmapDelete(mindmapName string, username string) error
	MindmapGetAll(mindmapName string) ([]MindmapInfo, error)
	MindmapExists(mindmapName string, username string) (bool, error)
	MindmapPermission(mindmapName string, username string, setPublic ...bool) (bool, error)

	// Node operations
	NodeAdd(mindmapName string, username string, parentID int, content string, extra map[string]string, logicalIndex string) error
	NodeDelete(mindmapName string, username string, id int) error
	NodeGet(mindmapName string, username string, id int) ([]*models.Node, error)
	NodeGetParent(mindmapName string, username string, id int) ([]*models.Node, error)
	NodeGetAll(mindmapName string, username string) ([]*models.Node, error)
	NodeModify(mindmapName string, username string, id int, content string, extra map[string]string, logicalIndex string) error
	NodeMove(mindmapName string, username string, sourceID, targetID int) error
	NodeOrderUpdate(mindmapName string, username string, nodeID int, logicalIndex string) error
}
