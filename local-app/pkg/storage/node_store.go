package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"mindnoscape/local-app/pkg/model"
)

// NodeStore defines the interface for node-related storage operations.
type NodeStore interface {
	NodeAdd(mindmap *model.Mindmap, newNodeInfo model.NodeInfo, forceID ...bool) (int, error)
	NodeGet(mindmap *model.Mindmap, nodeInfo model.NodeInfo, nodeFilter model.NodeFilter) ([]*model.Node, error)
	NodeUpdate(mindmap *model.Mindmap, node *model.Node, nodeUpdateInfo model.NodeInfo, nodeUpdateFilter model.NodeFilter) error
	NodeDelete(mindmap *model.Mindmap, node *model.Node) error
}

// NodeStorage implements the NodeStore interface.
type NodeStorage struct {
	storage *Storage
}

// NewNodeStorage creates a new NodeStorage instance.
func NewNodeStorage(storage *Storage) *NodeStorage {
	return &NodeStorage{storage: storage}
}

// NodeAdd adds a new node to the database.
func (s *NodeStorage) NodeAdd(mindmap *model.Mindmap, newNodeInfo model.NodeInfo, forceID ...bool) (int, error) {
	db := s.storage.GetDatabase()
	now := time.Now()

	// Start a transaction
	err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer db.Rollback()

	// Insert the node into nodes_{mindmap_id} table
	var result sql.Result
	var id int64
	if len(forceID) > 0 && forceID[0] {
		// Use the provided ID when forceID is true
		query := "INSERT INTO nodes_? (id, mindmap_id, parent_id, node_name, index_value, created, updated) VALUES (?, ?, ?, ?, ?, ?, ?)"
		result, err = db.Exec(query, mindmap.ID, newNodeInfo.ID, mindmap.ID, newNodeInfo.ParentID, newNodeInfo.Name, newNodeInfo.Index, now, now)
		if err != nil {
			return 0, fmt.Errorf("failed to add node with forced ID: %w", err)
		}
	} else {
		// Use auto-incrementing ID
		query := "INSERT INTO nodes_? (mindmap_id, parent_id, node_name, index_value, created, updated) VALUES (?, ?, ?, ?, ?, ?)"
		result, err = db.Exec(query, mindmap.ID, mindmap.ID, newNodeInfo.ParentID, newNodeInfo.Name, newNodeInfo.Index, now, now)
		if err != nil {
			return 0, fmt.Errorf("failed to add node: %w", err)
		}
	}
	id, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Insert content into node_content_{mindmap_id} table
	if len(newNodeInfo.Content) > 0 {
		contentQuery := "INSERT INTO node_content_? (node_id, key, value) VALUES (?, ?, ?)"
		for key, value := range newNodeInfo.Content {
			_, err = db.Exec(contentQuery, mindmap.ID, id, key, value)
			if err != nil {
				db.Rollback()
				return 0, fmt.Errorf("failed to add node content: %w", err)
			}
		}
	}

	// Commit the transaction
	if err := db.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(id), nil
}

// NodeGet retrieves nodes based on the provided info and filter.
func (s *NodeStorage) NodeGet(mindmap *model.Mindmap, nodeInfo model.NodeInfo, nodeFilter model.NodeFilter) ([]*model.Node, error) {
	db := s.storage.GetDatabase()
	query := "SELECT id, parent_id, node_name, index_value, created, updated FROM nodes_? WHERE 1=1"
	var args []interface{}

	args = append(args, mindmap.ID)
	if nodeFilter.ID {
		query += " AND id = ?"
		args = append(args, nodeInfo.ID)
	}
	if nodeFilter.ParentID {
		query += " AND parent_id = ?"
		args = append(args, nodeInfo.ParentID)
	}
	if nodeFilter.Name {
		query += " AND node_name = ?"
		args = append(args, nodeInfo.Name)
	}
	if nodeFilter.Index {
		query += " AND index_value = ?"
		args = append(args, nodeInfo.Index)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		var n model.Node
		err := rows.Scan(&n.ID, &n.ParentID, &n.Name, &n.Index, &n.Created, &n.Updated)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node row: %w", err)
		}
		n.MindmapID = mindmap.ID
		n.Content = make(map[string]string)
		nodes = append(nodes, &n)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating node rows: %w", err)
	}

	// Fetch content for each node
	for _, node := range nodes {
		contentQuery := "SELECT key, value FROM node_content_? WHERE node_id = ?"
		contentRows, err := db.Query(contentQuery, mindmap.ID, node.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to query node content: %w", err)
		}
		defer contentRows.Close()

		for contentRows.Next() {
			var key, value string
			if err := contentRows.Scan(&key, &value); err != nil {
				return nil, fmt.Errorf("failed to scan content row: %w", err)
			}
			node.Content[key] = value
		}

		if err := contentRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating content rows: %w", err)
		}
	}

	return nodes, nil
}

// NodeUpdate updates an existing node in the database.
func (s *NodeStorage) NodeUpdate(mindmap *model.Mindmap, node *model.Node, nodeUpdateInfo model.NodeInfo, nodeUpdateFilter model.NodeFilter) error {
	db := s.storage.GetDatabase()

	var err error
	if err = db.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer db.Rollback()

	var updates []string
	var args []interface{}

	if nodeUpdateFilter.Name {
		updates = append(updates, "node_name = ?")
		args = append(args, nodeUpdateInfo.Name)
	}
	if nodeUpdateFilter.ParentID {
		updates = append(updates, "parent_id = ?")
		args = append(args, nodeUpdateInfo.ParentID)
	}
	if nodeUpdateFilter.Index {
		updates = append(updates, "index_value = ?")
		args = append(args, nodeUpdateInfo.Index)
	}

	if len(updates) > 0 {
		updates = append(updates, "updated = ?")
		args = append(args, time.Now())

		// Use a prepared statement with a placeholder for the table name
		query := "UPDATE nodes_? SET " + strings.Join(updates, ", ") + " WHERE id = ?"
		args = append([]interface{}{mindmap.ID}, args...)
		args = append(args, node.ID)

		_, err = db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to update node: %w", err)
		}
	}

	if nodeUpdateFilter.Content {
		// Delete existing content
		deleteQuery := "DELETE FROM node_content_? WHERE node_id = ?"
		_, err = db.Exec(deleteQuery, mindmap.ID, node.ID)
		if err != nil {
			return fmt.Errorf("failed to delete existing node content: %w", err)
		}

		// Insert new content
		if len(nodeUpdateInfo.Content) > 0 {
			insertQuery := "INSERT INTO node_content_? (node_id, key, value) VALUES (?, ?, ?)"
			for key, value := range nodeUpdateInfo.Content {
				_, err = db.Exec(insertQuery, mindmap.ID, node.ID, key, value)
				if err != nil {
					return fmt.Errorf("failed to insert new node content: %w", err)
				}
			}
		}
	}

	return db.Commit()
}

// NodeDelete removes a node from the database.
func (s *NodeStorage) NodeDelete(mindmap *model.Mindmap, node *model.Node) error {
	db := s.storage.GetDatabase()

	if err := db.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer db.Rollback()

	// Delete node content
	contentQuery := "DELETE FROM node_content_? WHERE node_id = ?"
	_, err := db.Exec(contentQuery, mindmap.ID, node.ID)
	if err != nil {
		return fmt.Errorf("failed to delete node content: %w", err)
	}

	// Delete node
	nodeQuery := "DELETE FROM nodes_? WHERE id = ?"
	_, err = db.Exec(nodeQuery, mindmap.ID, node.ID)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	if err := db.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
