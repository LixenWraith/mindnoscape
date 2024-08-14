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
	AddUser(username, hashedPassword string) error
	DeleteUser(username string) error
	UserExists(username string) (bool, error)
	GetUser(username string) (*models.User, error)
	ModifyUser(oldUsername, newUsername, newHashedPassword string) error
	AuthenticateUser(username, password string) (bool, error)

	// Mindmap operations
	AddMindmap(name string, owner string, isPublic bool) (int, error)
	DeleteMindmap(name string, username string) error
	GetAllMindmaps(username string) ([]MindmapInfo, error)
	MindmapExists(name string, username string) (bool, error)
	ModifyMindmapAccess(name string, username string, isPublic bool) error
	HasMindmapPermission(mindmapName string, username string) (bool, error)

	// Node operations
	AddNode(mindmapName string, username string, parentID int, content string, extra map[string]string, logicalIndex string) error
	DeleteNode(mindmapName string, username string, id int) error
	GetNode(mindmapName string, username string, id int) ([]*models.Node, error)
	GetParentNode(mindmapName string, username string, id int) ([]*models.Node, error)
	GetAllNodesForMindmap(mindmapName string, username string) ([]*models.Node, error)
	ModifyNode(mindmapName string, username string, id int, content string, extra map[string]string, logicalIndex string) error
	MoveNode(mindmapName string, username string, sourceID, targetID int) error
	UpdateNodeOrder(mindmapName string, username string, nodeID int, logicalIndex string) error
}
