// Package storage provides functionality for persisting and retrieving Mindnoscape data.
// This file implements the SQLite storage backend.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore represents the SQLite storage implementation.
type SQLiteStore struct {
	db           *sql.DB
	UserStore    UserStore
	MindmapStore MindmapStore
	NodeStore    NodeStore
}

// NewSQLiteStore creates a new SQLiteStore instance and initializes the database.
func NewSQLiteStore(dbDir, dbFile string) (*SQLiteStore, error) {
	// Ensure the database directory exists
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the SQLite database or create and open it
	dbPath := filepath.Join(dbDir, dbFile)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create the store instance
	store := &SQLiteStore{
		db: db,
	}

	// Initialize the database schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	store.NodeStore = NewSQLiteNodeStorage(db)
	store.MindmapStore = NewSQLiteMindmapStorage(db)
	store.UserStore = NewSQLiteUserStorage(db)

	return store, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// initSchema initializes the database schema.
func (s *SQLiteStore) initSchema() error {
	// Create tables if they don't exist
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
