package storage

import (
	"database/sql"
	"fmt"
	"mindnoscape/local-app/internal/models"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db             *sql.DB
	NodeStorage    *SQLiteNodeStorage
	MindmapStorage *SQLiteMindmapStorage
	UserStorage    *SQLiteUserStorage
}

func NewSQLiteStore(dbDir, dbFile string) (*SQLiteStore, error) {
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, dbFile)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{
		db: db,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	store.NodeStorage = NewSQLiteNodeStorage(db)
	store.MindmapStorage = NewSQLiteMindmapStorage(db)
	store.UserStorage = NewSQLiteUserStorage(db)

	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
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

	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func (s *SQLiteStore) EnsureGuestUser() error {
	exists, err := s.UserStorage.UserExists("guest")
	if err != nil {
		return fmt.Errorf("failed to check if guest user exists: %w", err)
	}
	if !exists {
		err = s.UserStorage.AddUser("guest", "") // Empty password for guest
		if err != nil {
			return fmt.Errorf("failed to create guest user: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) AddUser(username, hashedPassword string) error {
	return s.UserStorage.AddUser(username, hashedPassword)
}

func (s *SQLiteStore) DeleteUser(username string) error {
	return s.UserStorage.DeleteUser(username)
}

func (s *SQLiteStore) UserExists(username string) (bool, error) {
	return s.UserStorage.UserExists(username)
}

func (s *SQLiteStore) GetUser(username string) (*models.User, error) {
	return s.UserStorage.GetUser(username)
}

func (s *SQLiteStore) ModifyUser(oldUsername, newUsername, newHashedPassword string) error {
	return s.UserStorage.ModifyUser(oldUsername, newUsername, newHashedPassword)
}

func (s *SQLiteStore) AuthenticateUser(username, password string) (bool, error) {
	return s.UserStorage.AuthenticateUser(username, password)
}

func (s *SQLiteStore) AddMindmap(name string, owner string, isPublic bool) (int, error) {
	return s.MindmapStorage.AddMindmap(name, owner, isPublic)
}

func (s *SQLiteStore) DeleteMindmap(name string, username string) error {
	return s.MindmapStorage.DeleteMindmap(name, username)
}

func (s *SQLiteStore) GetAllMindmaps(username string) ([]MindmapInfo, error) {
	return s.MindmapStorage.GetAllMindmaps(username)
}

func (s *SQLiteStore) MindmapExists(name string, username string) (bool, error) {
	return s.MindmapStorage.MindmapExists(name, username)
}

func (s *SQLiteStore) ModifyMindmapAccess(name string, username string, isPublic bool) error {
	return s.MindmapStorage.ModifyMindmapAccess(name, username, isPublic)
}

func (s *SQLiteStore) HasMindmapPermission(mindmapName string, username string) (bool, error) {
	return s.MindmapStorage.HasMindmapPermission(mindmapName, username)
}

func (s *SQLiteStore) AddNode(mindmapName string, username string, parentID int, content string, extra map[string]string, logicalIndex string) error {
	return s.NodeStorage.AddNode(mindmapName, username, parentID, content, extra, logicalIndex)
}

func (s *SQLiteStore) DeleteNode(mindmapName string, username string, id int) error {
	return s.NodeStorage.DeleteNode(mindmapName, username, id)
}

func (s *SQLiteStore) GetNode(mindmapName string, username string, id int) ([]*models.Node, error) {
	return s.NodeStorage.GetNode(mindmapName, username, id)
}

func (s *SQLiteStore) GetParentNode(mindmapName string, username string, id int) ([]*models.Node, error) {
	return s.NodeStorage.GetParentNode(mindmapName, username, id)
}

func (s *SQLiteStore) GetAllNodesForMindmap(mindmapName string, username string) ([]*models.Node, error) {
	return s.NodeStorage.GetAllNodesForMindmap(mindmapName, username)
}

func (s *SQLiteStore) ModifyNode(mindmapName string, username string, id int, content string, extra map[string]string, logicalIndex string) error {
	return s.NodeStorage.ModifyNode(mindmapName, username, id, content, extra, logicalIndex)
}

func (s *SQLiteStore) MoveNode(mindmapName string, username string, sourceID, targetID int) error {
	return s.NodeStorage.MoveNode(mindmapName, username, sourceID, targetID)
}

func (s *SQLiteStore) UpdateNodeOrder(mindmapName string, username string, nodeID int, logicalIndex string) error {
	return s.NodeStorage.UpdateNodeOrder(mindmapName, username, nodeID, logicalIndex)
}
