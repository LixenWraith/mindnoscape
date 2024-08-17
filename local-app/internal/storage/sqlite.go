package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db           *sql.DB
	UserStore    UserStore
	MindmapStore MindmapStore
	NodeStore    NodeStore
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
            node_index TEXT,
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
