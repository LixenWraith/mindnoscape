package storage

import (
	"mindnoscape/local-app/internal/models"
)

type MindMapInfo struct {
	Name     string
	IsPublic bool
	Owner    string
}

type Store interface {
	CreateUser(username, hashedPassword string) error
	DeleteUser(username string) error
	UserExists(username string) (bool, error)
	GetUser(username string) (*models.User, error)
	UpdateUser(oldUsername, newUsername, newHashedPassword string) error
	AuthenticateUser(username, password string) (bool, error)

	AddMindMap(name string, owner string, isPublic bool) (int, error)
	GetAllMindMaps(username string) ([]MindMapInfo, error)
	MindMapExists(name string, username string) (bool, error)
	UpdateMindMapAccess(name string, username string, isPublic bool) error
	HasMindMapPermission(mindmapName string, username string) (bool, error)

	AddNode(mindmapName string, username string, parentID int, content string, extra map[string]string, logicalIndex string) error
	GetNode(mindmapName string, username string, id int) (*models.Node, error)
	GetAllNodesForMindMap(mindmapName string, username string) ([]*models.Node, error)
	UpdateNode(mindmapName string, username string, id int, content string, extra map[string]string, logicalIndex string) error
	DeleteNode(mindmapName string, username string, id int) error
	GetParentNode(mindmapName string, username string, id int) (*models.Node, error)
	MoveNode(mindmapName string, username string, sourceID, targetID int) error
	UpdateNodeOrder(mindmapName string, username string, nodeID int, logicalIndex string) error
	ClearAllNodes(mindmapName string, username string) error
}
