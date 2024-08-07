package storage

import (
	"database/sql"
	"fmt"
	"mindnoscape/local-app/internal/models"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) (Store, error) {
	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}
	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	_, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS nodes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            parent_id INTEGER,
            content TEXT,
            logical_index TEXT,
            FOREIGN KEY (parent_id) REFERENCES nodes(id)
        );
        CREATE TABLE IF NOT EXISTS node_attributes (
            node_id INTEGER,
            key TEXT,
            value TEXT,
            FOREIGN KEY (node_id) REFERENCES nodes(id)
        );
    `)
	return err
}

func (s *SQLiteStore) AddNode(parentID int, content string, extra map[string]string, logicalIndex string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if parentID == -1 {
		// This is the root node
		result, err = tx.Exec("INSERT INTO nodes (id, parent_id, content, logical_index) VALUES (0, NULL, ?, ?)", content, logicalIndex)
	} else {
		result, err = tx.Exec("INSERT INTO nodes (parent_id, content, logical_index) VALUES (?, ?, ?)", parentID, content, logicalIndex)
	}
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	for key, value := range extra {
		_, err = tx.Exec("INSERT INTO node_attributes (node_id, key, value) VALUES (?, ?, ?)", id, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetNode(id int) (*models.Node, error) {
	node := &models.Node{Index: id}

	var parentID sql.NullInt64
	err := s.db.QueryRow("SELECT COALESCE(parent_id, -1), content, logical_index FROM nodes WHERE id = ?", id).Scan(&parentID, &node.Content, &node.LogicalIndex)
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		node.ParentID = int(parentID.Int64)
	} else {
		node.ParentID = -1 // or 0, depending on how you want to represent the root
	}

	node.Extra = make(map[string]string)
	rows, err := s.db.Query("SELECT key, value FROM node_attributes WHERE node_id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		node.Extra[key] = value
	}

	return node, nil
}

func (s *SQLiteStore) GetParentNode(id int) (*models.Node, error) {
	var parentID int
	err := s.db.QueryRow("SELECT parent_id FROM nodes WHERE id = ?", id).Scan(&parentID)
	if err != nil {
		return nil, err
	}
	return s.GetNode(parentID)
}

func (s *SQLiteStore) GetAllNodes() ([]*models.Node, error) {
	rows, err := s.db.Query("SELECT id, IFNULL(parent_id, -1) as parent_id, content, logical_index FROM nodes ORDER BY logical_index")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]*models.Node, 0)
	for rows.Next() {
		node := &models.Node{Extra: make(map[string]string), Children: []*models.Node{}}
		var parentID int
		if err := rows.Scan(&node.Index, &parentID, &node.Content, &node.LogicalIndex); err != nil {
			return nil, err
		}
		if parentID == -1 {
			node.ParentID = 0 // Set to 0 for root node
		} else {
			node.ParentID = parentID
		}
		nodes = append(nodes, node)
	}

	// Fetch extra attributes for all nodes
	for _, node := range nodes {
		attrRows, err := s.db.Query("SELECT key, value FROM node_attributes WHERE node_id = ?", node.Index)
		if err != nil {
			return nil, err
		}
		for attrRows.Next() {
			var key, value string
			if err := attrRows.Scan(&key, &value); err != nil {
				attrRows.Close()
				return nil, err
			}
			node.Extra[key] = value
		}
		attrRows.Close()
	}

	return nodes, nil
}

func (s *SQLiteStore) UpdateNode(id int, content string, extra map[string]string, logicalIndex string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE nodes SET content = ?, logical_index = ? WHERE id = ?", content, logicalIndex, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM node_attributes WHERE node_id = ?", id)
	if err != nil {
		return err
	}

	for key, value := range extra {
		_, err = tx.Exec("INSERT INTO node_attributes (node_id, key, value) VALUES (?, ?, ?)", id, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) ClearAllNodes() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all nodes except the root
	_, err = tx.Exec("DELETE FROM nodes WHERE id != 0")
	if err != nil {
		return err
	}

	// Delete all attributes
	_, err = tx.Exec("DELETE FROM node_attributes")
	if err != nil {
		return err
	}

	// Reset root node
	_, err = tx.Exec("UPDATE nodes SET content = 'Root', logical_index = '0' WHERE id = 0")
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) DeleteNode(id int) error {
	_, err := s.db.Exec("DELETE FROM nodes WHERE id = ?", id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM node_attributes WHERE node_id = ?", id)
	return err
}

func (s *SQLiteStore) UpdateNodeOrder(nodeID int, logicalIndex string) error {
	_, err := s.db.Exec("UPDATE nodes SET logical_index = ? WHERE id = ?", logicalIndex, nodeID)
	return err
}

func (s *SQLiteStore) MoveNode(sourceID, targetID int) error {
	_, err := s.db.Exec("UPDATE nodes SET parent_id = ? WHERE id = ?", targetID, sourceID)
	return err
}
