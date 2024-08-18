package storage

// Package storage provides functionality for persisting and retrieving Mindnoscape data.
// This file implements the storage operations for mindmaps using SQLite.
import (
	"database/sql"
	"fmt"

	"mindnoscape/local-app/internal/models"
)

// MindmapStore defines the interface for mindmap-related storage operations.
type MindmapStore interface {
	MindmapAdd(mindmapName string, owner string, isPublic bool) (int, error)
	MindmapDelete(mindmapName string, username string) error
	MindmapGetAll(username string) ([]models.MindmapInfo, error)
	MindmapExists(mindmapName string, username string) (bool, error)
	MindmapPermission(mindmapName string, username string, setPublic ...bool) (bool, error)
	MindmapGet(mindmapName string) (*models.MindmapInfo, error)
}

// SQLiteMindmapStorage implements the MindmapStore interface using SQLite.
type SQLiteMindmapStorage struct {
	db *sql.DB
}

// NewSQLiteMindmapStorage creates a new SQLiteMindmapStorage instance.
func NewSQLiteMindmapStorage(db *sql.DB) *SQLiteMindmapStorage {
	return &SQLiteMindmapStorage{db: db}
}

// MindmapAdd adds a new mindmap to the database.
func (s *SQLiteMindmapStorage) MindmapAdd(mindmapName string, owner string, isPublic bool) (int, error) {
	// Validate input
	if owner == "" {
		return 0, fmt.Errorf("owner cannot be empty")
	}

	// Insert the new mindmap
	result, err := s.db.Exec("INSERT INTO mindmaps (name, owner, is_public) VALUES (?, ?, ?)", mindmapName, owner, isPublic)
	if err != nil {
		return 0, fmt.Errorf("failed to add mindmap: %w", err)
	}

	// Get and return the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return int(id), nil
}

// MindmapDelete removes a mindmap and all its associated data from the database.
func (s *SQLiteMindmapStorage) MindmapDelete(name string, username string) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the mindmap ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", name, username).Scan(&mindmapID)
	if err != nil {
		return fmt.Errorf("failed to get mindmap ID: %w", err)
	}

	// Delete associated node attributes
	_, err = tx.Exec("DELETE FROM node_attributes WHERE node_id IN (SELECT id FROM nodes WHERE mindmap_id = ?)", mindmapID)
	if err != nil {
		return fmt.Errorf("failed to delete node attributes: %w", err)
	}

	// Delete associated nodes
	_, err = tx.Exec("DELETE FROM nodes WHERE mindmap_id = ?", mindmapID)
	if err != nil {
		return fmt.Errorf("failed to delete nodes: %w", err)
	}

	// Delete the mindmap
	_, err = tx.Exec("DELETE FROM mindmaps WHERE id = ?", mindmapID)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap: %w", err)
	}

	// Reset auto-increment counter of the mindmap
	_, err = tx.Exec("DELETE FROM sqlite_sequence WHERE name IN ('mindmaps', 'nodes', 'node_attributes')")
	if err != nil {
		return fmt.Errorf("failed to reset auto-increment counters: %w", err)
	}

	// Commit the transaction
	return tx.Commit()
}

// MindmapGetAll retrieves all mindmaps accessible to the given user.
func (s *SQLiteMindmapStorage) MindmapGetAll(username string) ([]models.MindmapInfo, error) {
	// Query for mindmaps owned by the user or public mindmaps
	rows, err := s.db.Query(`
        SELECT m.id, m.name, m.is_public, m.owner 
        FROM mindmaps m
        WHERE m.owner = ? OR m.is_public = 1
    `, username)
	if err != nil {
		return nil, fmt.Errorf("failed to query mindmaps: %w", err)
	}
	defer rows.Close()

	// Scan the results into MindmapInfo structs
	var mindmaps []models.MindmapInfo
	for rows.Next() {
		var m models.MindmapInfo
		if err := rows.Scan(&m.ID, &m.Name, &m.IsPublic, &m.Owner); err != nil {
			return nil, fmt.Errorf("failed to scan mindmap row: %w", err)
		}
		mindmaps = append(mindmaps, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mindmap rows: %w", err)
	}

	return mindmaps, nil
}

// MindmapExists checks if a mindmap with the given name exists and is accessible to the user.
func (s *SQLiteMindmapStorage) MindmapExists(mindmapName string, username string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check data existence: %w", err)
	}
	return count > 0, nil
}

// MindmapPermission checks or sets the permission of a mindmap.
func (s *SQLiteMindmapStorage) MindmapPermission(mindmapName string, username string, setPublic ...bool) (bool, error) {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check ownership and public status
	var isOwner bool
	var isPublic bool
	err = tx.QueryRow("SELECT owner = ?, is_public FROM mindmaps WHERE name = ?", username, mindmapName).Scan(&isOwner, &isPublic)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("mindmap '%s' does not exist", mindmapName)
		}
		return false, fmt.Errorf("failed to check mindmap permission: %w", err)
	}

	// If setPublic is provided, update the permission
	if len(setPublic) > 0 {
		if !isOwner {
			return false, fmt.Errorf("user '%s' does not have permission to modify mindmap '%s'", username, mindmapName)
		}

		newPublicStatus := setPublic[0]
		if isPublic != newPublicStatus {
			_, err = tx.Exec("UPDATE mindmaps SET is_public = ? WHERE name = ?", newPublicStatus, mindmapName)
			if err != nil {
				return false, fmt.Errorf("failed to update mindmap permission: %w", err)
			}
			isPublic = newPublicStatus
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return isOwner || isPublic, nil
}

// MindmapGet retrieves information about a specific mindmap.
func (s *SQLiteMindmapStorage) MindmapGet(name string) (*models.MindmapInfo, error) {
	var info models.MindmapInfo
	err := s.db.QueryRow("SELECT id, name, owner, is_public FROM mindmaps WHERE name = ?", name).Scan(&info.ID, &info.Name, &info.Owner, &info.IsPublic)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mindmap '%s' does not exist", name)
		}
		return nil, fmt.Errorf("failed to get mindmap info: %w", err)
	}
	return &info, nil
}
