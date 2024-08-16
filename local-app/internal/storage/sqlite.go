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
	db           *sql.DB
	NodeStore    NodeStore
	MindmapStore MindmapStore
	UserStore    UserStore
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

	store.NodeStore = NewSQLiteNodeStorage(db)
	store.MindmapStore = NewSQLiteMindmapStorage(db)
	store.UserStore = NewSQLiteUserStorage(db)

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

func (s *SQLiteStore) UserAdd(username, hashedPassword string) error {
	return s.UserStore.UserAdd(username, hashedPassword)
}

func (s *SQLiteStore) UserDelete(username string) error {
	return s.UserStore.UserDelete(username)
}

func (s *SQLiteStore) UserExists(username string) (bool, error) {
	return s.UserStore.UserExists(username)
}

func (s *SQLiteStore) UserGet(username string) (*models.User, error) {
	return s.UserStore.UserGet(username)
}

func (s *SQLiteStore) UserModify(oldUsername, newUsername, newHashedPassword string) error {
	return s.UserStore.UserModify(oldUsername, newUsername, newHashedPassword)
}

func (s *SQLiteStore) UserAuthenticate(username, password string) (bool, error) {
	return s.UserStore.UserAuthenticate(username, password)
}

func (s *SQLiteStore) MindmapAdd(name string, owner string, isPublic bool) (int, error) {
	return s.MindmapStore.MindmapAdd(name, owner, isPublic)
}

func (s *SQLiteStore) MindmapDelete(name string, username string) error {
	return s.MindmapStore.MindmapDelete(name, username)
}

func (s *SQLiteStore) MindmapGetAll(username string) ([]MindmapInfo, error) {
	return s.MindmapStore.MindmapGetAll(username)
}

func (s *SQLiteStore) MindmapExists(name string, username string) (bool, error) {
	return s.MindmapStore.MindmapExists(name, username)
}

func (s *SQLiteStore) MindmapPermission(name string, username string, setPublic ...bool) (bool, error) {
	return s.MindmapStore.MindmapPermission(name, username, setPublic...)
}

func (s *SQLiteStore) NodeAdd(mindmapName string, username string, parentID int, content string, extra map[string]string, logicalIndex string) error {
	return s.NodeStore.NodeAdd(mindmapName, username, parentID, content, extra, logicalIndex)
}

func (s *SQLiteStore) NodeDelete(mindmapName string, username string, id int) error {
	return s.NodeStore.NodeDelete(mindmapName, username, id)
}

func (s *SQLiteStore) NodeGet(mindmapName string, username string, id int) ([]*models.Node, error) {
	return s.NodeStore.NodeGet(mindmapName, username, id)
}

func (s *SQLiteStore) NodeGetParent(mindmapName string, username string, id int) ([]*models.Node, error) {
	return s.NodeStore.NodeGetParent(mindmapName, username, id)
}

func (s *SQLiteStore) NodeGetAll(mindmapName string, username string) ([]*models.Node, error) {
	return s.NodeStore.NodeGetAll(mindmapName, username)
}

func (s *SQLiteStore) NodeModify(mindmapName string, username string, id int, content string, extra map[string]string, logicalIndex string) error {
	return s.NodeStore.NodeModify(mindmapName, username, id, content, extra, logicalIndex)
}

func (s *SQLiteStore) NodeMove(mindmapName string, username string, sourceID, targetID int) error {
	return s.NodeStore.NodeMove(mindmapName, username, sourceID, targetID)
}

func (s *SQLiteStore) NodeOrderUpdate(mindmapName string, username string, nodeID int, logicalIndex string) error {
	return s.NodeStore.NodeOrderUpdate(mindmapName, username, nodeID, logicalIndex)
}
