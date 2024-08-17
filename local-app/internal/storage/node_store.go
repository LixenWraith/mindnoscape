package storage

import (
	"fmt"
	"mindnoscape/local-app/internal/models"

	"database/sql"
)

type NodeStore interface {
	NodeAdd(mindmapName string, username string, parentID int, content string, extra map[string]string, index string) (int, error)
	NodeDelete(mindmapName string, username string, id int) error
	NodeGet(mindmapName string, username string, id int) ([]*models.Node, error)
	NodeGetParent(mindmapName string, username string, id int) ([]*models.Node, error)
	NodeGetAll(mindmapName string, username string) ([]*models.Node, error)
	NodeUpdate(mindmapName string, username string, id int, content string, extra map[string]string, index string) error
	NodeMove(mindmapName string, username string, sourceID, targetID int) error
	NodeOrderUpdate(mindmapName string, username string, nodeID int, index string) error
}

type SQLiteNodeStorage struct {
	db *sql.DB
}

func NewSQLiteNodeStorage(db *sql.DB) *SQLiteNodeStorage {
	return &SQLiteNodeStorage{db: db}
}

func (ns *SQLiteNodeStorage) NodeAdd(mindmapName string, username string, parentID int, content string, extra map[string]string, index string) (int, error) {
	tx, err := ns.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	// First, get the data ID and check permissions
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return -1, fmt.Errorf("failed to get data ID: %w", err)
	}

	// Insert the node
	var nodeID int64
	if parentID == -1 {
		// This is the root node, explicitly set its ID to 0 and parentID to -1
		_, err = tx.Exec("INSERT INTO nodes (id, mindmap_id, parent_id, content, node_index) VALUES (0, ?, -1, ?, ?)", mindmapID, content, index)
		nodeID = 0
	} else {
		// For non-root nodes, let the database auto-increment the ID
		result, err := tx.Exec("INSERT INTO nodes (mindmap_id, parent_id, content, node_index) VALUES (?, ?, ?, ?)", mindmapID, parentID, content, index)
		if err != nil {
			return -1, fmt.Errorf("failed to insert node: %w", err)
		}
		nodeID, err = result.LastInsertId()
		if err != nil {
			return -1, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	}

	if err != nil {
		return -1, fmt.Errorf("failed to insert node: %w", err)
	}

	// Insert extra attributes
	for key, value := range extra {
		_, err = tx.Exec("INSERT INTO node_attributes (node_id, key, value) VALUES (?, ?, ?)", nodeID, key, value)
		if err != nil {
			return -1, fmt.Errorf("failed to insert node attribute: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return -1, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(nodeID), nil
}

func (ns *SQLiteNodeStorage) NodeDelete(mindmapName string, username string, id int) error {
	tx, err := ns.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check permissions (ensure the user is the owner)
	var owner string
	err = tx.QueryRow("SELECT owner FROM mindmaps WHERE name = ?", mindmapName).Scan(&owner)
	if err != nil {
		return fmt.Errorf("failed to get mindmap owner: %w", err)
	}
	if owner != username {
		return fmt.Errorf("only the owner can modify the mindmap")
	}

	// Delete node attributes
	_, err = tx.Exec("DELETE FROM node_attributes WHERE node_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete node attributes: %w", err)
	}

	// Delete the node
	_, err = tx.Exec("DELETE FROM nodes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return tx.Commit()
}

func (ns *SQLiteNodeStorage) NodeGet(mindmapName string, username string, id int) ([]*models.Node, error) {
	// Check permissions
	var mindmapID int
	err := ns.db.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Get the node
	var node models.Node
	err = ns.db.QueryRow("SELECT id, COALESCE(parent_id, -1), content, node_index FROM nodes WHERE id = ? AND mindmap_id = ?", id, mindmapID).Scan(&node.ID, &node.ParentID, &node.Content, &node.Index)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Get node attributes
	rows, err := ns.db.Query("SELECT key, value FROM node_attributes WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get node attributes: %w", err)
	}
	defer rows.Close()

	node.Extra = make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan node attribute: %w", err)
		}
		node.Extra[key] = value
	}

	return []*models.Node{&node}, nil
}

func (ns *SQLiteNodeStorage) NodeGetParent(mindmapName string, username string, id int) ([]*models.Node, error) {
	// First, get the parent ID
	var parentID int
	err := ns.db.QueryRow(`
        SELECT COALESCE(n.parent_id, -1)
        FROM nodes n
        JOIN mindmaps m ON n.mindmap_id = m.id
        WHERE n.id = ? AND m.name = ? AND (m.owner = ? OR m.is_public = 1)
    `, id, mindmapName, username).Scan(&parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent ID: %w", err)
	}

	// Now get the parent node
	return ns.NodeGet(mindmapName, username, parentID)
}

func (ns *SQLiteNodeStorage) NodeGetAll(mindmapName string, username string) ([]*models.Node, error) {
	// Check permissions and get mindmap ID
	var mindmapID int
	err := ns.db.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Get all nodes for the mindmap
	rows, err := ns.db.Query("SELECT id, COALESCE(parent_id, -1), content, node_index FROM nodes WHERE mindmap_id = ?", mindmapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*models.Node
	for rows.Next() {
		var node models.Node
		if err := rows.Scan(&node.ID, &node.ParentID, &node.Content, &node.Index); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		node.Extra = make(map[string]string)
		nodes = append(nodes, &node)
	}

	// Get attributes for all nodes
	for _, node := range nodes {
		attrRows, err := ns.db.Query("SELECT key, value FROM node_attributes WHERE node_id = ?", node.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get node attributes: %w", err)
		}
		for attrRows.Next() {
			var key, value string
			if err := attrRows.Scan(&key, &value); err != nil {
				attrRows.Close()
				return nil, fmt.Errorf("failed to scan node attribute: %w", err)
			}
			node.Extra[key] = value
		}
		attrRows.Close()
	}

	return nodes, nil
}

func (ns *SQLiteNodeStorage) NodeUpdate(mindmapName string, username string, id int, content string, extra map[string]string, index string) error {
	tx, err := ns.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check permissions (ensure the user is the owner)
	var owner string
	err = tx.QueryRow("SELECT owner FROM mindmaps WHERE name = ?", mindmapName).Scan(&owner)
	if err != nil {
		return fmt.Errorf("failed to get mindmap owner: %w", err)
	}
	if owner != username {
		return fmt.Errorf("only the owner can modify the mindmap")
	}

	// Update node
	_, err = tx.Exec("UPDATE nodes SET content = ?, node_index = ? WHERE id = ?", content, index, id)
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	// Delete existing attributes
	_, err = tx.Exec("DELETE FROM node_attributes WHERE node_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete existing node attributes: %w", err)
	}

	// Insert new attributes
	for key, value := range extra {
		_, err = tx.Exec("INSERT INTO node_attributes (node_id, key, value) VALUES (?, ?, ?)", id, key, value)
		if err != nil {
			return fmt.Errorf("failed to insert node attribute: %w", err)
		}
	}

	return tx.Commit()
}

func (ns *SQLiteNodeStorage) NodeMove(mindmapName string, username string, sourceID, targetID int) error {
	tx, err := ns.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check permissions (ensure the user is the owner)
	var owner string
	err = tx.QueryRow("SELECT owner FROM mindmaps WHERE name = ?", mindmapName).Scan(&owner)
	if err != nil {
		return fmt.Errorf("failed to get mindmap owner: %w", err)
	}
	if owner != username {
		return fmt.Errorf("only the owner can modify the mindmap")
	}

	// Check if source and target nodes exist
	var sourceCount, targetCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM nodes WHERE id = ? AND mindmap_id = (SELECT id FROM mindmaps WHERE name = ?)", sourceID, mindmapName).Scan(&sourceCount)
	if err != nil {
		return fmt.Errorf("failed to check source node: %w", err)
	}
	if sourceCount == 0 {
		return fmt.Errorf("source node not found")
	}

	err = tx.QueryRow("SELECT COUNT(*) FROM nodes WHERE id = ? AND mindmap_id = (SELECT id FROM mindmaps WHERE name = ?)", targetID, mindmapName).Scan(&targetCount)
	if err != nil {
		return fmt.Errorf("failed to check target node: %w", err)
	}
	if targetCount == 0 {
		return fmt.Errorf("target node not found")
	}

	// Update the parent ID of the source node
	_, err = tx.Exec("UPDATE nodes SET parent_id = ? WHERE id = ?", targetID, sourceID)
	if err != nil {
		return fmt.Errorf("failed to update node parent: %w", err)
	}

	return tx.Commit()
}

func (ns *SQLiteNodeStorage) NodeOrderUpdate(mindmapName string, username string, id int, index string) error {
	// Check permissions
	var count int
	err := ns.db.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("no permission to update node order in data '%s'", mindmapName)
	}

	// Update node index
	_, err = ns.db.Exec("UPDATE nodes SET node_index = ? WHERE id = ?", index, id)
	if err != nil {
		return fmt.Errorf("failed to update node order: %w", err)
	}

	return nil
}
