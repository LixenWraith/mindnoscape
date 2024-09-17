package storage

import (
	"context"
	"fmt"
	"path/filepath"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// Storage represents the main storage implementation.
type Storage struct {
	db Database
	UserStore
	MindmapStore
	NodeStore
	logger *log.Logger
}

// NewStorage creates a new Storage instance and initializes the database.
func NewStorage(config *model.Config, logger *log.Logger) (*Storage, error) {
	logger.Info(context.Background(), "Initializing storage", log.Fields{
		"databaseType": config.DatabaseType,
		"databaseDir":  config.DatabaseDir,
	})

	dbDriver, err := validateDBDriver(config.DatabaseType)
	if err != nil {
		logger.Error(context.Background(), "Invalid database driver", log.Fields{"error": err, "databaseType": config.DatabaseType})
		return nil, fmt.Errorf("invalid database driver '%s': %w", config.DatabaseType, err)
	}

	db, err := NewDatabase(dbDriver, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to create database instance", log.Fields{"error": err, "databaseType": config.DatabaseType})
		return nil, fmt.Errorf("failed to create database instance: %w", err)
	}

	// Construct the full path for the database file
	dataSourceName := filepath.Join(config.DatabaseDir, config.DatabaseFile)

	// Open the database connection
	if err := db.Open(dataSourceName); err != nil {
		logger.Error(context.Background(), "Failed to open database connection", log.Fields{"error": err, "dataSourceName": dataSourceName})
		return nil, fmt.Errorf("failed to open database connection '%s': %s", dataSourceName, err)
	}

	storage := &Storage{
		db:     db,
		logger: logger,
	}

	// Create user and mindmap tables
	if err := storage.initSchema(); err != nil {
		db.Close()
		logger.Error(context.Background(), "Failed to initialize schema", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to initialize schema: %s", err)
	}

	// Create storages
	storage.UserStore = NewUserStorage(storage)
	storage.MindmapStore = NewMindmapStorage(storage)
	storage.NodeStore = NewNodeStorage(storage)

	logger.Info(context.Background(), "Storage initialized successfully", nil)
	return storage, nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	s.logger.Info(context.Background(), "Closing storage", nil)
	if err := s.db.Close(); err != nil {
		s.logger.Error(context.Background(), "Failed to close database connection", log.Fields{"error": err})
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}

// initSchema initializes the database schema.
func (s *Storage) initSchema() error {
	s.logger.Info(context.Background(), "Initializing database schema", nil)
	if err := s.db.InitSchema(); err != nil {
		s.logger.Error(context.Background(), "Failed to initialize schema", log.Fields{"error": err})
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return nil
}

// GetDatabase returns the database instance
func (s *Storage) GetDatabase() Database {
	return s.db
}
