package storage

import (
	"database/sql"
	"fmt"
	"mindnoscape/local-app/internal/models"
)

type SQLiteStore struct {
	db            *sql.DB
	mindmapTables map[string]string // Maps mindmap names to table names
}

func NewSQLiteStore(db *sql.DB) (Store, error) {
	store := &SQLiteStore{
		db:            db,
		mindmapTables: make(map[string]string),
	}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}
	if err := store.loadExistingMindmaps(); err != nil {
		return nil, fmt.Errorf("failed to load existing mindmaps: %v", err)
	}
	return store, nil
}

func (s *SQLiteStore) loadExistingMindmaps() error {
	rows, err := s.db.Query("SELECT name FROM mindmaps")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		s.mindmapTables[name] = fmt.Sprintf("mindmap_%s", name)
	}

	return nil
}
func (s *SQLiteStore) initSchema() error {
	_, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS mindmaps (
            name TEXT PRIMARY KEY
        );
        CREATE TABLE IF NOT EXISTS nodes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            parent_id INTEGER,
            content TEXT,
            logical_index TEXT
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

func (s *SQLiteStore) AddMindMap(name string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert into mindmaps table
	_, err = tx.Exec("INSERT INTO mindmaps (name) VALUES (?)", name)
	if err != nil {
		return err
	}

	// Create new table for this mindmap
	tableName := fmt.Sprintf("mindmap_%s", name)
	_, err = tx.Exec(fmt.Sprintf(`
        CREATE TABLE %s (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            parent_id INTEGER,
            content TEXT,
            logical_index TEXT
        )
    `, tableName))
	if err != nil {
		return err
	}

	// Create table for node attributes
	_, err = tx.Exec(fmt.Sprintf(`
        CREATE TABLE %s_attributes (
            node_id INTEGER,
            key TEXT,
            value TEXT,
            FOREIGN KEY (node_id) REFERENCES %s(id)
        )
    `, tableName, tableName))
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.mindmapTables[name] = tableName
	return nil
}

func (s *SQLiteStore) GetAllMindMaps() ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM mindmaps")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mindmaps []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		mindmaps = append(mindmaps, name)
	}
	return mindmaps, nil
}

func (s *SQLiteStore) MindMapExists(name string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStore) DebugPrintDBStructure(mindmapName string) error {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	rows, err := s.db.Query(fmt.Sprintf(`
        SELECT id, parent_id, content, logical_index 
        FROM %s 
        ORDER BY id
    `, tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("Raw Database Structure:")
	fmt.Println("ID | ParentID | LogicalIndex | Content")
	fmt.Println("-------------------------------------------")
	for rows.Next() {
		var id, parentID int
		var content, logicalIndex string
		if err := rows.Scan(&id, &parentID, &content, &logicalIndex); err != nil {
			return err
		}
		fmt.Printf("%2d | %8d | %12s | %s\n", id, parentID, logicalIndex, content)
	}
	fmt.Println("-------------------------------------------")
	return nil
}

func (s *SQLiteStore) AddNode(mindmapName string, parentID int, content string, extra map[string]string, logicalIndex string) error {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if parentID == -1 {
		// This is the root node
		result, err = tx.Exec(fmt.Sprintf("INSERT INTO %s (id, parent_id, content, logical_index) VALUES (0, -1, ?, '0')", tableName), content)
	} else {
		result, err = tx.Exec(fmt.Sprintf("INSERT INTO %s (parent_id, content, logical_index) VALUES (?, ?, ?)", tableName), parentID, content, logicalIndex)
	}
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	for key, value := range extra {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s_attributes (node_id, key, value) VALUES (?, ?, ?)", tableName), id, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetAllNodesForMindMap(mindmapName string) ([]*models.Node, error) {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return nil, fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	rows, err := s.db.Query(fmt.Sprintf(`
        SELECT id, parent_id, content, logical_index 
        FROM %s 
        ORDER BY CASE WHEN id = 0 THEN 0 ELSE 1 END, logical_index
    `, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]*models.Node, 0)
	for rows.Next() {
		node := &models.Node{Extra: make(map[string]string), Children: []*models.Node{}}
		if err := rows.Scan(&node.Index, &node.ParentID, &node.Content, &node.LogicalIndex); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
		fmt.Printf("Loaded node from DB: Index=%d, ParentID=%d, Content=%s, LogicalIndex=%s\n",
			node.Index, node.ParentID, node.Content, node.LogicalIndex)
	}

	// Fetch extra attributes for all nodes
	for _, node := range nodes {
		attrRows, err := s.db.Query(fmt.Sprintf("SELECT key, value FROM %s_attributes WHERE node_id = ?", tableName), node.Index)
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

func (s *SQLiteStore) GetNode(mindmapName string, id int) (*models.Node, error) {
	node := &models.Node{Index: id}

	var parentID sql.NullInt64
	err := s.db.QueryRow("SELECT COALESCE(parent_id, -1), content, logical_index FROM nodes WHERE mindmap_name = ? AND id = ?", mindmapName, id).Scan(&parentID, &node.Content, &node.LogicalIndex)
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

func (s *SQLiteStore) GetParentNode(mindmapName string, id int) (*models.Node, error) {
	var parentID int
	err := s.db.QueryRow("SELECT parent_id FROM nodes WHERE mindmap_name = ? AND id = ?", mindmapName, id).Scan(&parentID)
	if err != nil {
		return nil, err
	}
	return s.GetNode(mindmapName, parentID)
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

func (s *SQLiteStore) UpdateNode(mindmapName string, id int, content string, extra map[string]string, logicalIndex string) error {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(fmt.Sprintf("UPDATE %s SET content = ?, logical_index = ? WHERE id = ?", tableName), content, logicalIndex, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s_attributes WHERE node_id = ?", tableName), id)
	if err != nil {
		return err
	}

	for key, value := range extra {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s_attributes (node_id, key, value) VALUES (?, ?, ?)", tableName), id, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) ClearAllNodes(mindmapName string) error {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Drop the mindmap table
	_, err = tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	if err != nil {
		return err
	}

	// Drop the attributes table
	_, err = tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s_attributes", tableName))
	if err != nil {
		return err
	}

	// Remove the mindmap from the mindmaps table
	_, err = tx.Exec("DELETE FROM mindmaps WHERE name = ?", mindmapName)
	if err != nil {
		return err
	}

	// Remove the mindmap from the in-memory map
	delete(s.mindmapTables, mindmapName)

	return tx.Commit()
}

func (s *SQLiteStore) DeleteNode(mindmapName string, id int) error {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName), id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s_attributes WHERE node_id = ?", tableName), id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) UpdateNodeOrder(mindmapName string, nodeID int, logicalIndex string) error {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	_, err := s.db.Exec(fmt.Sprintf("UPDATE %s SET logical_index = ? WHERE id = ?", tableName), logicalIndex, nodeID)
	return err
}

func (s *SQLiteStore) MoveNode(mindmapName string, sourceID, targetID int) error {
	tableName, ok := s.mindmapTables[mindmapName]
	if !ok {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	_, err := s.db.Exec(fmt.Sprintf("UPDATE %s SET parent_id = ? WHERE id = ?", tableName), targetID, sourceID)
	return err
}
