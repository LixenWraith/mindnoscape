package storage

import (
	"fmt"
	"mindnoscape/local-app/src/pkg/model"
	"time"
)

// MindmapStore defines the interface for mindmap-related storage operations.
type MindmapStore interface {
	MindmapAdd(user *model.User, newMindmapInfo model.MindmapInfo) (int, error)
	MindmapGet(user *model.User, mindmapInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) ([]*model.Mindmap, error)
	MindmapUpdate(mindmap *model.Mindmap, mindmapUpdateInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) error
	MindmapDelete(mindmap *model.Mindmap) error
}

// MindmapStorage implements the MindmapStore interface.
type MindmapStorage struct {
	storage *Storage
}

// NewMindmapStorage creates a new MindmapStorage instance.
func NewMindmapStorage(storage *Storage) *MindmapStorage {
	return &MindmapStorage{storage: storage}
}

// MindmapAdd adds a new mindmap to the database.
func (s *MindmapStorage) MindmapAdd(user *model.User, newMindmap model.MindmapInfo) (int, error) {
	// Check if the user already has a mindmap with the same name
	existingMindmaps, err := s.MindmapGet(user, newMindmap, model.MindmapFilter{Name: true, Owner: true})
	if err != nil {
		return 0, fmt.Errorf("failed to check for existing mindmap: %w", err)
	}
	if len(existingMindmaps) > 0 {
		return 0, fmt.Errorf("mindmap with name '%s' already exists for this user", newMindmap.Name)
	}

	db := s.storage.GetDatabase()

	// Start a transaction
	err = db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = db.Rollback()
		}
	}()

	// Insert the new mindmap
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO mindmaps (mindmap_name, owner, is_public, created, updated) VALUES (?, ?, ?, ?, ?)",
		newMindmap.Name, user.Username, newMindmap.IsPublic, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to add mindmap: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Create tables for the new mindmap within the transaction
	err = db.CreateMindmapTables(int(id))
	if err != nil {
		return 0, fmt.Errorf("failed to create tables for mindmap %d: %w", id, err)
	}

	// Commit the transaction
	if err := db.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(id), nil
}

// MindmapGet retrieves mindmaps based on the provided info and filter.
func (s *MindmapStorage) MindmapGet(user *model.User, mindmapInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) ([]*model.Mindmap, error) {
	db := s.storage.GetDatabase()
	query := "SELECT id, mindmap_name, owner, is_public, created, updated FROM mindmaps WHERE 1=1"
	var args []interface{}

	if mindmapFilter.ID {
		query += " AND id = ?"
		args = append(args, mindmapInfo.ID)
	}
	if mindmapFilter.Name {
		query += " AND mindmap_name = ?"
		args = append(args, mindmapInfo.Name)
	}
	if mindmapFilter.Owner {
		query += " AND owner = ?"
		args = append(args, mindmapInfo.Owner)
	}
	if mindmapFilter.IsPublic {
		query += " AND is_public = ?"
		args = append(args, mindmapInfo.IsPublic)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query mindmaps: %w", err)
	}
	defer rows.Close()

	var mindmaps []*model.Mindmap
	for rows.Next() {
		var m model.Mindmap
		err := rows.Scan(&m.ID, &m.Name, &m.Owner, &m.IsPublic, &m.Created, &m.Updated)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mindmap row: %w", err)
		}
		mindmaps = append(mindmaps, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mindmap rows: %w", err)
	}

	return mindmaps, nil
}

// MindmapUpdate updates an existing mindmap in the database.
func (s *MindmapStorage) MindmapUpdate(mindmap *model.Mindmap, mindmapUpdateInfo model.MindmapInfo, mindmapFilter model.MindmapFilter) error {
	db := s.storage.GetDatabase()
	query := "UPDATE mindmaps SET updated = ? WHERE id = ?"
	args := []interface{}{time.Now(), mindmap.ID}

	if mindmapFilter.Name {
		query += ", mindmap_name = ?"
		args = append(args, mindmapUpdateInfo.Name)
	}
	if mindmapFilter.Owner {
		query += ", owner = ?"
		args = append(args, mindmapUpdateInfo.Owner)
	}
	if mindmapFilter.IsPublic {
		query += ", is_public = ?"
		args = append(args, mindmapUpdateInfo.IsPublic)
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update mindmap: %w", err)
	}

	return nil
}

// MindmapDelete removes a mindmap from the database.
func (s *MindmapStorage) MindmapDelete(mindmap *model.Mindmap) error {
	db := s.storage.GetDatabase()

	// Start a transaction
	err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func(db Database) {
		_ = db.Rollback()
	}(db)

	// Drop the mindmap tables
	err = db.DropMindmapTables(mindmap.ID)
	if err != nil {
		return fmt.Errorf("failed to drop mindmap tables: %w", err)
	}

	// Delete the mindmap from the mindmaps table
	_, err = db.Exec("DELETE FROM mindmaps WHERE id = ?", mindmap.ID)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap: %w", err)
	}

	// Commit the transaction
	return db.Commit()
}
