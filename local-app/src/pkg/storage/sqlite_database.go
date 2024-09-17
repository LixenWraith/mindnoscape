package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"mindnoscape/local-app/src/pkg/log"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDatabase implements the Database interface for SQLite
type SQLiteDatabase struct {
	BaseDatabase
}

// Open opens a connection to the SQLite database
func (s *SQLiteDatabase) Open(dataSourceName string) error {
	s.logger.Info(context.Background(), "Opening SQLite database", log.Fields{"dbPath": filepath.Base(dataSourceName)})

	// Ensure the directory for the database file exists
	dbDir := filepath.Dir(dataSourceName)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		s.logger.Error(context.Background(), "Failed to create database directory", log.Fields{"error": err, "directory": dbDir})
		return fmt.Errorf("failed to create database directory '%s': %w", dbDir, err)
	}

	// Open the database connection with additional parameters
	db, err := sql.Open("sqlite3", dataSourceName+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		s.logger.Error(context.Background(), "Failed to open SQLite database", log.Fields{"error": err})
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}

	// Set pragmas for better performance and reliability
	if _, err := db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		db.Close()
		s.logger.Error(context.Background(), "Failed to set SQLite synchronous pragma", log.Fields{"error": err})
		return fmt.Errorf("failed to set SQLite synchronous pragma: %w", err)
	}
	if _, err := db.Exec("PRAGMA cache_size = 5000"); err != nil {
		db.Close()
		s.logger.Error(context.Background(), "Failed to set SQLite cache pragma", log.Fields{"error": err})
		return fmt.Errorf("failed to set SQLite cache pragma: %w", err)
	}

	// Verify the connection
	if err := db.Ping(); err != nil {
		db.Close()
		s.logger.Error(context.Background(), "Failed to verify database connection", log.Fields{"error": err})
		return fmt.Errorf("failed to verify database connection: %v", err)
	}

	s.db = db
	s.logger.Info(context.Background(), "SQLite database opened successfully", nil)
	return nil
}

// Close closes the connection to the SQLite database
func (s *SQLiteDatabase) Close() error {
	s.logger.Info(context.Background(), "Closing SQLite database", nil)
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error(context.Background(), "Failed to close SQLite database", log.Fields{"error": err})
			return fmt.Errorf("failed to close SQLite database: %w", err)
		}
	}
	s.logger.Info(context.Background(), "SQLite database closed successfully", nil)
	return nil
}
