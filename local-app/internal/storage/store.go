package storage

import (
	"mindnoscape/local-app/internal/models"
)

type Store interface {
    AddNode(parentID int, content string, extra map[string]string, logicalIndex string) error
    GetNode(id int) (*models.Node, error)
    UpdateNode(id int, content string, extra map[string]string, logicalIndex string) error
    DeleteNode(id int) error
    GetAllNodes() ([]*models.Node, error)
    GetParentNode(id int) (*models.Node, error)
    MoveNode(sourceID, targetID int) error
	UpdateNodeOrder(nodeID int, logicalIndex string) error
    ClearAllNodes() error
}
