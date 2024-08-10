package storage

import (
	"mindnoscape/local-app/internal/models"
)

type Store interface {
	// New methods for managing multiple mindmaps
	AddMindMap(name string) error
	GetAllMindMaps() ([]string, error)
	MindMapExists(name string) (bool, error)

	// Updated methods to include mindmap name
	AddNode(mindmapName string, parentID int, content string, extra map[string]string, logicalIndex string) error
	GetNode(mindmapName string, id int) (*models.Node, error)
	UpdateNode(mindmapName string, id int, content string, extra map[string]string, logicalIndex string) error
	DeleteNode(mindmapName string, id int) error
	GetAllNodesForMindMap(mindmapName string) ([]*models.Node, error)
	GetParentNode(mindmapName string, id int) (*models.Node, error)
	MoveNode(mindmapName string, sourceID, targetID int) error
	UpdateNodeOrder(mindmapName string, nodeID int, logicalIndex string) error
	ClearAllNodes(mindmapName string) error
	DebugPrintDBStructure(mindmapName string) error
}
