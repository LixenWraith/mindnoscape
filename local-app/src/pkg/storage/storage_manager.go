package storage

import (
	"fmt"
	"path/filepath"

	"mindnoscape/local-app/src/pkg/model"
)

// Storage represents the main storage implementation.
type Storage struct {
	db Database
	UserStore
	MindmapStore
	NodeStore
}

// NewStorage creates a new Storage instance and initializes the database.
func NewStorage(config *model.Config) (*Storage, error) {
	dbDriver, err := validateDBDriver(config.DatabaseType)
	if err != nil {
		return nil, fmt.Errorf("invalid database driver '%s': %w", config.DatabaseType, err)
	}

	db, err := NewDatabase(dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to create database instance: %w", err)
	}

	// Construct the full path for the database file
	dataSourceName := filepath.Join(config.DatabaseDir, config.DatabaseFile)

	// Open the database connection
	if err := db.Open(dataSourceName); err != nil {
		return nil, fmt.Errorf("failed to open database connection '%s': %s", dataSourceName, err)
	}

	storage := &Storage{
		db: db,
	}

	// Create user and mindmap tables
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %s", err)
	}

	// Create storages
	storage.UserStore = NewUserStorage(storage)
	storage.MindmapStore = NewMindmapStorage(storage)
	storage.NodeStore = NewNodeStorage(storage)

	return storage, nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}

// initSchema initializes the database schema.
func (s *Storage) initSchema() error {
	if err := s.db.InitSchema(); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return nil
}

// GetDatabase returns the database instance
func (s *Storage) GetDatabase() Database {
	return s.db
}
