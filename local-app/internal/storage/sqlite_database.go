package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDatabase implements the Database interface for SQLite
type SQLiteDatabase struct {
	BaseDatabase
}

// Open opens a connection to the SQLite database
func (s *SQLiteDatabase) Open(dataSourceName string) error {
	// Ensure the directory for the database file exists
	dbDir := filepath.Dir(dataSourceName)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory '%s': %w", dbDir, err)
	}

	// Open the database connection with additional parameters
	db, err := sql.Open("sqlite3", dataSourceName+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}

	// Set pragmas for better performance and reliability
	if _, err := db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		db.Close()
		return fmt.Errorf("failed to set SQLite synchronous pragma: %w", err)
	}
	if _, err := db.Exec("PRAGMA cache_size = 5000"); err != nil {
		db.Close()
		return fmt.Errorf("failed to set SQLite cache pragma: %w", err)
	}

	// Verify the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to verify database connection: %v", err)
	}

	s.db = db
	return nil
}

// Close closes the connection to the SQLite database
func (s *SQLiteDatabase) Close() error {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("failed to close SQLite database: %w", err)
		}
	}
	return nil
}
