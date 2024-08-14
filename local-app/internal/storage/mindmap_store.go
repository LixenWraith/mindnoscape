package storage

import (
	"database/sql"
	"fmt"
)

type SQLiteMindmapStorage struct {
	db *sql.DB
}

func NewSQLiteMindmapStorage(db *sql.DB) *SQLiteMindmapStorage {
	return &SQLiteMindmapStorage{db: db}
}

func (s *SQLiteMindmapStorage) AddMindmap(name string, owner string, isPublic bool) (int, error) {
	result, err := s.db.Exec("INSERT INTO mindmaps (name, owner, is_public) VALUES (?, ?, ?)", name, owner, isPublic)
	if err != nil {
		return 0, fmt.Errorf("failed to add data: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return int(id), nil
}

func (s *SQLiteMindmapStorage) DeleteMindmap(name string, username string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the data ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", name, username).Scan(&mindmapID)
	if err != nil {
		return fmt.Errorf("failed to get data ID: %w", err)
	}

	// Delete associated nodes
	_, err = tx.Exec("DELETE FROM nodes WHERE mindmap_id = ?", mindmapID)
	if err != nil {
		return fmt.Errorf("failed to delete nodes: %w", err)
	}

	// Delete the data
	_, err = tx.Exec("DELETE FROM mindmaps WHERE id = ?", mindmapID)
	if err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *SQLiteMindmapStorage) GetAllMindmaps(username string) ([]MindmapInfo, error) {
	rows, err := s.db.Query(`
		SELECT m.name, m.is_public, m.owner 
		FROM mindmaps m
		WHERE m.owner = ? OR m.is_public = 1
	`, username)
	if err != nil {
		return nil, fmt.Errorf("failed to query mindmaps: %w", err)
	}
	defer rows.Close()

	var mindmaps []MindmapInfo
	for rows.Next() {
		var m MindmapInfo
		if err := rows.Scan(&m.Name, &m.IsPublic, &m.Owner); err != nil {
			return nil, fmt.Errorf("failed to scan data row: %w", err)
		}
		mindmaps = append(mindmaps, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating data rows: %w", err)
	}

	return mindmaps, nil
}

func (s *SQLiteMindmapStorage) MindmapExists(name string, username string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", name, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check data existence: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteMindmapStorage) ModifyMindmapAccess(name string, username string, isPublic bool) error {
	result, err := s.db.Exec("UPDATE mindmaps SET is_public = ? WHERE name = ? AND owner = ?", isPublic, name, username)
	if err != nil {
		return fmt.Errorf("failed to modify data access: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no data found with name '%s' owned by '%s'", name, username)
	}

	return nil
}

func (s *SQLiteMindmapStorage) HasMindmapPermission(mindmapName string, username string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM mindmaps 
		WHERE name = ? AND (owner = ? OR is_public = 1)
	`, mindmapName, username).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check data permission: %w", err)
	}

	return count > 0, nil
}

func (s *SQLiteMindmapStorage) GetMindmapID(name string, username string) (int, error) {
	var id int
	err := s.db.QueryRow(`
		SELECT id 
		FROM mindmaps 
		WHERE name = ? AND (owner = ? OR is_public = 1)
	`, name, username).Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("data not found or user doesn't have permission")
		}
		return 0, fmt.Errorf("failed to get data ID: %w", err)
	}

	return id, nil
}
