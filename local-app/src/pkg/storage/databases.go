// Package storage provides functionality for persisting and retrieving Mindnoscape data.
// This file handles the general SQL database interfaces and schemas.
package storage

import (
	"context"
	"database/sql"
	"fmt"

	"mindnoscape/local-app/src/pkg/log"
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
func NewDatabase(driver DBDriver, logger *log.Logger) (Database, error) {
	switch driver {
	case SQLite:
		return &SQLiteDatabase{BaseDatabase: BaseDatabase{logger: logger}}, nil
	// case PostgreSQL:
	//     return &PostgreSQLDatabase{}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

// BaseDatabase provides a base implementation of some Database methods
type BaseDatabase struct {
	db     *sql.DB
	tx     *sql.Tx
	logger *log.Logger
}

// Begin starts a new transaction
func (b *BaseDatabase) Begin() error {
	tx, err := b.db.Begin()
	if err != nil {
		b.logger.Error(context.Background(), "Failed to begin transaction", log.Fields{"error": err})
		return err
	}
	b.tx = tx
	b.logger.Info(context.Background(), "Transaction started", nil)
	return nil
}

// Commit commits the current transaction
func (b *BaseDatabase) Commit() error {
	if b.tx == nil {
		return fmt.Errorf("no active transaction")
		b.logger.Error(context.Background(), "No active transaction to commit", nil)
	}
	err := b.tx.Commit()
	if err != nil {
		b.logger.Error(context.Background(), "Failed to commit transaction", log.Fields{"error": err})
		return err
	}
	b.tx = nil
	b.logger.Info(context.Background(), "Transaction committed", nil)
	return nil
}

// Rollback rolls back the current transaction
func (b *BaseDatabase) Rollback() error {
	if b.tx == nil {
		b.logger.Error(context.Background(), "No active transaction to rollback", nil)
		return fmt.Errorf("no active transaction")
	}
	err := b.tx.Rollback()
	if err != nil {
		b.logger.Error(context.Background(), "Failed to rollback transaction", log.Fields{"error": err})
		return err
	}
	b.tx = nil
	b.logger.Info(context.Background(), "Transaction rolled back", nil)
	return nil
}

// Exec executes a query without returning any rows
func (b *BaseDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	b.logger.Debug(context.Background(), "Executing query", log.Fields{"query": query, "args": args})
	if b.tx != nil {
		return b.tx.Exec(query, args...)
	}
	return b.db.Exec(query, args...)
}

// Query executes a query that returns rows
func (b *BaseDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	b.logger.Debug(context.Background(), "Querying", log.Fields{"query": query, "args": args})
	return b.db.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row
func (b *BaseDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return b.db.QueryRow(query, args...)
}

// InitSchema initializes the database schema
func (b *BaseDatabase) InitSchema() error {
	b.logger.Info(context.Background(), "Initializing database schema", nil)

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
		b.logger.Error(context.Background(), "Failed to create tables", log.Fields{"error": err})
		return fmt.Errorf("failed to create tables: %w", err)
	}
	b.logger.Info(context.Background(), "Database schema initialized successfully", nil)
	return nil
}

// CreateMindmapTables creates tables for a specific mindmap
func (b *BaseDatabase) CreateMindmapTables(mindmapID int) error {
	b.logger.Info(context.Background(), "Creating mindmap tables", log.Fields{"mindmapID": mindmapID})

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
		b.logger.Error(context.Background(), "Failed to create mindmap tables", log.Fields{"error": err, "mindmapID": mindmapID})
		return fmt.Errorf("failed to create mindmap tables: mindmap_id=%d, error=%v", mindmapID, err)
	}
	b.logger.Info(context.Background(), "Mindmap tables created successfully", log.Fields{"mindmapID": mindmapID})
	return nil
}

// DropMindmapTables drops tables for a specific mindmap
func (b *BaseDatabase) DropMindmapTables(mindmapID int) error {
	b.logger.Info(context.Background(), "Dropping mindmap tables", log.Fields{"mindmapID": mindmapID})

	_, err := b.Exec(fmt.Sprintf(`
		DROP TABLE IF EXISTS nodes_%d;
		DROP TABLE IF EXISTS node_content_%d;
	`, mindmapID, mindmapID))

	if err != nil {
		b.logger.Error(context.Background(), "Failed to drop mindmap tables", log.Fields{"error": err, "mindmapID": mindmapID})
		return fmt.Errorf("failed to drop mindmap tables: mindmap_id=%d, error=%v", mindmapID, err)
	}
	b.logger.Info(context.Background(), "Mindmap tables dropped successfully", log.Fields{"mindmapID": mindmapID})
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
