package storage

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"mindnoscape/local-app/internal/models"
)

type SQLiteStore struct {
	db *sql.DB
}

func (s *SQLiteStore) initSchema() error {
	_, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            username TEXT PRIMARY KEY,
            password_hash TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS mindmaps (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            owner TEXT NOT NULL,
            is_public BOOLEAN NOT NULL DEFAULT 0,
            FOREIGN KEY (owner) REFERENCES users(username),
            UNIQUE (name, owner)
        );

        CREATE TABLE IF NOT EXISTS nodes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            mindmap_id INTEGER NOT NULL,
            parent_id INTEGER,
            content TEXT,
            logical_index TEXT,
            FOREIGN KEY (mindmap_id) REFERENCES mindmaps(id)
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

func NewSQLiteStore(db *sql.DB) (Store, error) {
	store := &SQLiteStore{
		db: db,
	}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}
	return store, nil
}

// User-related methods

func (s *SQLiteStore) UserExists(username string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStore) AddUser(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hashedPassword)
	return err
}

func (s *SQLiteStore) DeleteUser(username string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all nodes and node attributes for mindmaps owned by this user
	_, err = tx.Exec(`
        DELETE FROM nodes WHERE mindmap_id IN (
            SELECT id FROM mindmaps WHERE owner = ?
        )
    `, username)
	if err != nil {
		return fmt.Errorf("failed to delete nodes: %v", err)
	}

	_, err = tx.Exec(`
        DELETE FROM node_attributes WHERE node_id NOT IN (
            SELECT id FROM nodes
        )
    `)
	if err != nil {
		return fmt.Errorf("failed to delete node attributes: %v", err)
	}

	// Delete all mindmaps owned by this user
	_, err = tx.Exec("DELETE FROM mindmaps WHERE owner = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete mindmaps: %v", err)
	}

	// Delete the user
	_, err = tx.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetUser(username string) (*models.User, error) {
	var user models.User
	var passwordHash string
	err := s.db.QueryRow("SELECT username, password_hash FROM users WHERE username = ?", username).Scan(&user.Username, &passwordHash)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = []byte(passwordHash)
	return &user, nil
}

func (s *SQLiteStore) ModifyUser(oldUsername, newUsername, newPassword string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if newUsername != "" && newUsername != oldUsername {
		_, err = tx.Exec("UPDATE users SET username = ? WHERE username = ?", newUsername, oldUsername)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE mindmaps SET owner = ? WHERE owner = ?", newUsername, oldUsername)
		if err != nil {
			return err
		}
	}

	if newPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE users SET password_hash = ? WHERE username = ?", hashedPassword, oldUsername)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) AuthenticateUser(username, password string) (bool, error) {
	if username == "guest" {
		return true, nil // Guest user is always authenticated
	}

	var passwordHash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	return err == nil, nil
}

// Mindmap-related methods

func (s *SQLiteStore) AddMindmap(name string, owner string, isPublic bool) (int, error) {
	result, err := s.db.Exec("INSERT INTO mindmaps (name, owner, is_public) VALUES (?, ?, ?)", name, owner, isPublic)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *SQLiteStore) GetAllMindmaps(username string) ([]MindmapInfo, error) {
	rows, err := s.db.Query(`
        SELECT name, is_public, owner
        FROM mindmaps
        WHERE owner = ? OR is_public = 1
        ORDER BY name
    `, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mindmaps []MindmapInfo
	for rows.Next() {
		var mm MindmapInfo
		if err := rows.Scan(&mm.Name, &mm.IsPublic, &mm.Owner); err != nil {
			return nil, err
		}
		mindmaps = append(mindmaps, mm)
	}
	return mindmaps, nil
}

func (s *SQLiteStore) MindmapExists(name string, username string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", name, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStore) DeleteMindmap(name string, username string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check if the mindmap exists and the user has permission to delete it
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND owner = ?", name, username).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check mindmap ownership: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("mindmap '%s' does not exist or user does not have permission to delete it", name)
	}

	// Get the mindmap ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", name, username).Scan(&mindmapID)
	if err != nil {
		return fmt.Errorf("failed to get mindmap ID: %v", err)
	}

	// Delete all nodes for this mindmap
	_, err = tx.Exec("DELETE FROM nodes WHERE mindmap_id = ?", mindmapID)
	if err != nil {
		return fmt.Errorf("failed to delete nodes: %v", err)
	}

	// Delete all node attributes for this mindmap's nodes
	_, err = tx.Exec("DELETE FROM node_attributes WHERE node_id NOT IN (SELECT id FROM nodes)")
	if err != nil {
		return fmt.Errorf("failed to delete node attributes: %v", err)
	}

	// Delete the mindmap entry
	_, err = tx.Exec("DELETE FROM mindmaps WHERE id = ?", mindmapID)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap: %v", err)
	}

	return tx.Commit()
}

func (s *SQLiteStore) ModifyMindmapAccess(name string, username string, isPublic bool) error {
	_, err := s.db.Exec("UPDATE mindmaps SET is_public = ? WHERE name = ? AND owner = ?", isPublic, name, username)
	return err
}

// Node-related methods

func (s *SQLiteStore) AddNode(mindmapName string, username string, parentID int, content string, extra map[string]string, logicalIndex string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// First, get the mindmap ID and check permissions
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&mindmapID)
	if err != nil {
		return err
	}

	var result sql.Result
	if parentID == -1 {
		// This is the root node, explicitly set its ID to 0
		_, err = tx.Exec("INSERT INTO nodes (id, mindmap_id, parent_id, content, logical_index) VALUES (0, ?, ?, ?, ?)", mindmapID, nil, content, logicalIndex)
	} else {
		// For non-root nodes, let the database auto-increment the ID
		result, err = tx.Exec("INSERT INTO nodes (mindmap_id, parent_id, content, logical_index) VALUES (?, ?, ?, ?)", mindmapID, parentID, content, logicalIndex)
	}

	if err != nil {
		return err
	}

	var nodeID int64
	if parentID != -1 {
		nodeID, err = result.LastInsertId()
		if err != nil {
			return err
		}
	}

	// Insert extra attributes
	for key, value := range extra {
		_, err = tx.Exec("INSERT INTO node_attributes (node_id, key, value) VALUES (?, ?, ?)", nodeID, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetNode(mindmapName string, username string, id int) ([]*models.Node, error) {
	if id == -1 {
		// Retrieve all nodes for the mindmap
		var mindmapID int
		err := s.db.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&mindmapID)
		if err != nil {
			return nil, fmt.Errorf("failed to get mindmap info: %v", err)
		}

		rows, err := s.db.Query(`
            SELECT id, COALESCE(parent_id, -1) as parent_id, content, logical_index 
            FROM nodes 
            WHERE mindmap_id = ?
            ORDER BY CASE WHEN parent_id IS NULL THEN 0 ELSE 1 END, logical_index
        `, mindmapID)
		if err != nil {
			return nil, fmt.Errorf("failed to query nodes: %v", err)
		}
		defer rows.Close()

		nodes := make([]*models.Node, 0)
		for rows.Next() {
			node := &models.Node{Extra: make(map[string]string), Children: []*models.Node{}}
			if err := rows.Scan(&node.Index, &node.ParentID, &node.Content, &node.LogicalIndex); err != nil {
				return nil, fmt.Errorf("failed to scan node: %v", err)
			}
			node.MindmapID = mindmapID
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
	} else {
		// Retrieve a single node
		node := &models.Node{Index: id, Extra: make(map[string]string)}

		var parentID sql.NullInt64
		err := s.db.QueryRow(`
            SELECT COALESCE(n.parent_id, -1), n.content, n.logical_index, m.id
            FROM nodes n
            JOIN mindmaps m ON n.mindmap_id = m.id
            WHERE m.name = ? AND (m.owner = ? OR m.is_public = 1) AND n.id = ?`,
			mindmapName, username, id).Scan(&parentID, &node.Content, &node.LogicalIndex, &node.MindmapID)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			node.ParentID = int(parentID.Int64)
		} else {
			node.ParentID = -1
		}

		// Fetch extra attributes
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

		return []*models.Node{node}, nil
	}
}

func (s *SQLiteStore) GetAllNodesForMindmap(mindmapName string, username string) ([]*models.Node, error) {
	// First, get the mindmap ID and check permissions
	var mindmapID int
	var owner string
	var isPublic bool
	err := s.db.QueryRow("SELECT id, owner, is_public FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&mindmapID, &owner, &isPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap info: %v", err)
	}

	// Then get all nodes for this mindmap
	rows, err := s.db.Query(`
        SELECT id, COALESCE(parent_id, -1) as parent_id, content, logical_index 
        FROM nodes 
        WHERE mindmap_id = ?
        ORDER BY CASE WHEN parent_id IS NULL THEN 0 ELSE 1 END, logical_index
    `, mindmapID)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %v", err)
	}
	defer rows.Close()

	nodes := make([]*models.Node, 0)
	for rows.Next() {
		node := &models.Node{Extra: make(map[string]string), Children: []*models.Node{}}
		if err := rows.Scan(&node.Index, &node.ParentID, &node.Content, &node.LogicalIndex); err != nil {
			return nil, fmt.Errorf("failed to scan node: %v", err)
		}
		node.MindmapID = mindmapID
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

func (s *SQLiteStore) ModifyNode(mindmapName string, username string, id int, content string, extra map[string]string, logicalIndex string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
        UPDATE nodes 
        SET content = ?, logical_index = ? 
        WHERE id = ? AND mindmap_id = (SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1))
    `, content, logicalIndex, id, mindmapName, username)
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

func (s *SQLiteStore) DeleteNode(mindmapName string, username string, id int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
        DELETE FROM nodes 
        WHERE id = ? AND mindmap_id = (SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1))
    `, id, mindmapName, username)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM node_attributes WHERE node_id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetParentNode(mindmapName string, username string, id int) ([]*models.Node, error) {
	var parentID int
	err := s.db.QueryRow(`
        SELECT parent_id 
        FROM nodes n
        JOIN mindmaps m ON n.mindmap_id = m.id
        WHERE m.name = ? AND (m.owner = ? OR m.is_public = 1) AND n.id = ?`, mindmapName, username, id).Scan(&parentID)
	if err != nil {
		return nil, err
	}
	return s.GetNode(mindmapName, username, parentID)
}

func (s *SQLiteStore) MoveNode(mindmapName string, username string, sourceID, targetID int) error {
	_, err := s.db.Exec(`
        UPDATE nodes 
        SET parent_id = ? 
        WHERE id = ? AND mindmap_id = (SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1))
    `, targetID, sourceID, mindmapName, username)
	return err
}

func (s *SQLiteStore) UpdateNodeOrder(mindmapName string, username string, nodeID int, logicalIndex string) error {
	_, err := s.db.Exec(`
        UPDATE nodes 
        SET logical_index = ? 
        WHERE id = ? AND mindmap_id = (SELECT id FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1))
    `, logicalIndex, nodeID, mindmapName, username)
	return err
}

func (s *SQLiteStore) DeleteAllNodes(mindmapName string, username string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if the user owns the mindmap
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check mindmap ownership: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("user does not have permission to clear this mindmap")
	}

	// Delete all nodes for this mindmap
	_, err = tx.Exec("DELETE FROM nodes WHERE mindmap_id = (SELECT id FROM mindmaps WHERE name = ? AND owner = ?)", mindmapName, username)
	if err != nil {
		return fmt.Errorf("failed to delete nodes: %v", err)
	}

	// Delete all node attributes for this mindmap's nodes
	_, err = tx.Exec("DELETE FROM node_attributes WHERE node_id NOT IN (SELECT id FROM nodes)")
	if err != nil {
		return fmt.Errorf("failed to delete node attributes: %v", err)
	}

	// Delete the mindmap entry
	_, err = tx.Exec("DELETE FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap: %v", err)
	}

	return tx.Commit()
}

func (s *SQLiteStore) HasMindmapPermission(mindmapName string, username string) (bool, error) {
	if username == "guest" {
		// Guest can only access public mindmaps
		var isPublic bool
		err := s.db.QueryRow("SELECT is_public FROM mindmaps WHERE name = ?", mindmapName).Scan(&isPublic)
		if err != nil {
			return false, err
		}
		return isPublic, nil
	}

	// For regular users, check ownership or public status
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
