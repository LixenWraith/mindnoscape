// Package storage provides functionality for persisting and retrieving Mindnoscape data.
// This file handles the general SQL database interfaces and schemas.
package storage

import (
	"database/sql"
	"fmt"
)

// DBDriver represents the type of database driver
type DBDriver string

const (
	SQLite DBDriver = "sqlite"
	// PostgreSQL DBDriver = "postgres" // Uncomment when adding PostgreSQL support
)

// Database interface defines common database operations
type Database interface {
	Open(dataSourceName string) error
	Close() error
	Begin() error
	Commit() error
	Rollback() error
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	InitSchema() error
	CreateMindmapTables(mindmapID int) error
	DropMindmapTables(mindmapID int) error
}

// NewDatabase creates a new Database instance based on the specified driver
func NewDatabase(driver DBDriver) (Database, error) {
	switch driver {
	case SQLite:
		return &SQLiteDatabase{}, nil
	// case PostgreSQL:
	//     return &PostgreSQLDatabase{}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

// BaseDatabase provides a base implementation of some Database methods
type BaseDatabase struct {
	db *sql.DB
	tx *sql.Tx
}

func (b *BaseDatabase) Begin() error {
	tx, err := b.db.Begin()
	if err != nil {
		return err
	}
	b.tx = tx
	return nil
}

func (b *BaseDatabase) Commit() error {
	if b.tx == nil {
		return fmt.Errorf("no active transaction")
	}
	err := b.tx.Commit()
	b.tx = nil
	return err
}

func (b *BaseDatabase) Rollback() error {
	if b.tx == nil {
		return fmt.Errorf("no active transaction")
	}
	err := b.tx.Rollback()
	b.tx = nil
	return err
}

func (b *BaseDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	if b.tx != nil {
		return b.tx.Exec(query, args...)
	}
	return b.db.Exec(query, args...)
}

func (b *BaseDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return b.db.Query(query, args...)
}

func (b *BaseDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return b.db.QueryRow(query, args...)
}

// InitSchema initializes the database schema
func (b *BaseDatabase) InitSchema() error {
	_, err := b.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash BLOB NOT NULL,
			active BOOLEAN NOT NULL DEFAULT 1,
			created DATETIME NOT NULL,
			updated DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS mindmaps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mindmap_name TEXT NOT NULL,
			owner TEXT NOT NULL,
			is_public BOOLEAN NOT NULL DEFAULT 0,
			created DATETIME NOT NULL,
			updated DATETIME NOT NULL,
			FOREIGN KEY (owner) REFERENCES users(username),
			UNIQUE (mindmap_name, owner)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	return nil
}

// CreateMindmapTables creates tables for a specific mindmap
func (b *BaseDatabase) CreateMindmapTables(mindmapID int) error {
	query := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS nodes_%d (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            mindmap_id INTEGER NOT NULL,
            parent_id INTEGER,
            node_name TEXT NOT NULL,
            index_value TEXT NOT NULL,
            created DATETIME NOT NULL,
            updated DATETIME NOT NULL,
            FOREIGN KEY (mindmap_id) REFERENCES mindmaps(id)
        );
        CREATE TABLE IF NOT EXISTS node_content_%d (
            node_id INTEGER,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            FOREIGN KEY (node_id) REFERENCES nodes_%d(id)
        );
    `, mindmapID, mindmapID, mindmapID)

	_, err := b.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create mindmap tables: mindmap_id=%d, error=%v", mindmapID, err)
	}
	return nil
}

// DropMindmapTables drops tables for a specific mindmap
func (b *BaseDatabase) DropMindmapTables(mindmapID int) error {
	_, err := b.Exec(fmt.Sprintf(`
		DROP TABLE IF EXISTS nodes_%d;
		DROP TABLE IF EXISTS node_content_%d;
	`, mindmapID, mindmapID))
	if err != nil {
		return fmt.Errorf("failed to drop mindmap tables: mindmap_id=%d, error=%v", mindmapID, err)
	}
	return nil
}

// validateDBDriver checks if the provided driver is supported
func validateDBDriver(driver string) (DBDriver, error) {
	switch DBDriver(driver) {
	case SQLite:
		return SQLite, nil
	// case PostgreSQL:
	//     return PostgreSQL, nil
	default:
		return "", fmt.Errorf("unsupported database driver: %s", driver)
	}
}
