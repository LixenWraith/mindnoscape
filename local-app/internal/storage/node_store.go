// Package storage provides functionality for persisting and retrieving Mindnoscape data.
// This file implements the storage operations for nodes using SQLite.
package storage

import (
	"database/sql"
	"fmt"

	"mindnoscape/local-app/internal/models"
)

// NodeStore defines the interface for node-related storage operations.
type NodeStore interface {
	NodeAdd(mindmapName string, username string, parentID int, content string, extra map[string]string, index string, id ...int) (int, error)
	NodeDelete(mindmapName string, username string, id int) error
	NodeGet(mindmapName string, username string, id int) ([]*models.Node, error)
	NodeGetAll(mindmapName string, username string) ([]*models.Node, error)
	NodeUpdate(mindmapName string, username string, id int, content string, extra map[string]string, index string) error
	NodeMove(mindmapName string, username string, sourceID, targetID int) error
	NodeOrderUpdate(mindmapName string, username string, nodeID int, index string) error
}

// SQLiteNodeStorage implements the NodeStore interface using SQLite.
type SQLiteNodeStorage struct {
	db *sql.DB
}

// NewSQLiteNodeStorage creates a new SQLiteNodeStorage instance.
func NewSQLiteNodeStorage(db *sql.DB) *SQLiteNodeStorage {
	return &SQLiteNodeStorage{db: db}
}

// NodeAdd adds a new node to the database.
func (ns *SQLiteNodeStorage) NodeAdd(mindmapName string, username string, parentID int, content string, extra map[string]string, index string, id ...int) (int, error) {
	// Start a transaction
	tx, err := ns.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Get the mindmap ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return 0, fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Insert the node
	var nodeID int64
	if len(id) > 0 && id[0] != 0 {
		// Use the provided ID
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO nodes_%d (id, parent_id, content, node_index) VALUES (?, ?, ?, ?)", mindmapID), id[0], parentID, content, index)
		nodeID = int64(id[0])
	} else if parentID == -1 {
		// This is the root node, explicitly set its ID to 0 and parentID to -1
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO nodes_%d (id, parent_id, content, node_index) VALUES (0, -1, ?, ?)", mindmapID), content, index)
		nodeID = 0
	} else {
		// For non-root nodes, let the database auto-increment the ID
		result, err := tx.Exec(fmt.Sprintf("INSERT INTO nodes_%d (parent_id, content, node_index) VALUES (?, ?, ?)", mindmapID), parentID, content, index)
		if err != nil {
			return 0, fmt.Errorf("failed to insert node: %w", err)
		}
		nodeID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	}

	if err != nil {
		return 0, fmt.Errorf("failed to insert node: %w", err)
	}

	// Insert extra attributes
	for key, value := range extra {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO node_attributes_%d (node_id, key, value) VALUES (?, ?, ?)", mindmapID), nodeID, key, value)
		if err != nil {
			return 0, fmt.Errorf("failed to insert node attribute: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(nodeID), nil
}

// NodeDelete removes a node from the database.
func (ns *SQLiteNodeStorage) NodeDelete(mindmapName string, username string, id int) error {
	// Start a transaction
	tx, err := ns.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get mindmap ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Check if the node exists
	var count int
	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM nodes_%d WHERE id = ?", mindmapID), id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check node existence: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("node with id %d does not exist", id)
	}

	// Delete node attributes
	_, err = tx.Exec(fmt.Sprintf("DELETE FROM node_attributes_%d WHERE node_id = ?", mindmapID), id)
	if err != nil {
		return fmt.Errorf("failed to delete node attributes: %w", err)
	}

	// Delete the node
	_, err = tx.Exec(fmt.Sprintf("DELETE FROM nodes_%d WHERE id = ?", mindmapID), id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	// Commit the transaction
	return tx.Commit()
}

// NodeGet retrieves a specific node from the database.
func (ns *SQLiteNodeStorage) NodeGet(mindmapName string, username string, id int) ([]*models.Node, error) {
	// Get mindmap ID
	var mindmapID int
	err := ns.db.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Get the node
	var node models.Node
	err = ns.db.QueryRow(fmt.Sprintf("SELECT id, COALESCE(parent_id, -1), content, node_index FROM nodes_%d WHERE id = ?", mindmapID), id).Scan(&node.ID, &node.ParentID, &node.Content, &node.Index)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Get node attributes
	rows, err := ns.db.Query(fmt.Sprintf("SELECT key, value FROM node_attributes_%d WHERE node_id = ?", mindmapID), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get node attributes: %w", err)
	}
	defer rows.Close()

	// Scan attributes into the node's Extra map
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

// NodeGetAll retrieves all nodes for a given mindmap.
func (ns *SQLiteNodeStorage) NodeGetAll(mindmapName string, username string) ([]*models.Node, error) {
	// Get mindmap ID and check permissions
	var mindmapID int
	var isPublic bool
	var owner string
	err := ns.db.QueryRow("SELECT id, is_public, owner FROM mindmaps WHERE name = ?", mindmapName).Scan(&mindmapID, &isPublic, &owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap info: %w", err)
	}

	// Check if the user has permission to access the mindmap
	if !isPublic && owner != username {
		return nil, fmt.Errorf("user %s does not have permission to access mindmap '%s'", username, mindmapName)
	}

	// Get all nodes for the mindmap
	rows, err := ns.db.Query(fmt.Sprintf("SELECT id, COALESCE(parent_id, -1), content, node_index FROM nodes_%d", mindmapID))
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	defer rows.Close()

	// Scan nodes into a slice
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
		attrRows, err := ns.db.Query(fmt.Sprintf("SELECT key, value FROM node_attributes_%d WHERE node_id = ?", mindmapID), node.ID)
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

// NodeUpdate updates an existing node in the database.
func (ns *SQLiteNodeStorage) NodeUpdate(mindmapName string, username string, id int, content string, extra map[string]string, index string) error {
	// Start a transaction
	tx, err := ns.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get mindmap ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Check if the node exists
	var count int
	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM nodes_%d WHERE id = ?", mindmapID), id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check node existence: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("node with id %d does not exist", id)
	}

	// Update node
	_, err = tx.Exec(fmt.Sprintf("UPDATE nodes_%d SET content = ?, node_index = ? WHERE id = ?", mindmapID), content, index, id)
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	// Delete existing attributes
	_, err = tx.Exec(fmt.Sprintf("DELETE FROM node_attributes_%d WHERE node_id = ?", mindmapID), id)
	if err != nil {
		return fmt.Errorf("failed to delete existing node attributes: %w", err)
	}

	// Insert new attributes
	for key, value := range extra {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO node_attributes_%d (node_id, key, value) VALUES (?, ?, ?)", mindmapID), id, key, value)
		if err != nil {
			return fmt.Errorf("failed to insert node attribute: %w", err)
		}
	}

	// Commit the transaction
	return tx.Commit()
}

// NodeMove changes the parent of a node in the database.
func (ns *SQLiteNodeStorage) NodeMove(mindmapName string, username string, sourceID, targetID int) error {
	// Start a transaction
	tx, err := ns.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get mindmap ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Check if source and target nodes exist
	var sourceCount, targetCount int
	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM nodes_%d WHERE id = ?", mindmapID), sourceID).Scan(&sourceCount)
	if err != nil {
		return fmt.Errorf("failed to check source node: %w", err)
	}
	if sourceCount == 0 {
		return fmt.Errorf("source node not found")
	}
	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM nodes_%d WHERE id = ?", mindmapID), targetID).Scan(&targetCount)
	if err != nil {
		return fmt.Errorf("failed to check target node: %w", err)
	}
	if targetCount == 0 {
		return fmt.Errorf("target node not found")
	}

	// Update the parent ID of the source node
	_, err = tx.Exec(fmt.Sprintf("UPDATE nodes_%d SET parent_id = ? WHERE id = ?", mindmapID), targetID, sourceID)
	if err != nil {
		return fmt.Errorf("failed to update node parent: %w", err)
	}

	// Commit the transaction
	return tx.Commit()
}

func (ns *SQLiteNodeStorage) NodeOrderUpdate(mindmapName string, username string, nodeID int, index string) error {
	// Start a transaction
	tx, err := ns.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get mindmap ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Check if the node exists
	var count int
	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM nodes_%d WHERE id = ?", mindmapID), nodeID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check node existence: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("node with id %d does not exist", nodeID)
	}

	// Update node index
	_, err = tx.Exec(fmt.Sprintf("UPDATE nodes_%d SET node_index = ? WHERE id = ?", mindmapID), index, nodeID)
	if err != nil {
		return fmt.Errorf("failed to update node order: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
