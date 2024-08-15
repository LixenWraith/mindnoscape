package storage

import (
	"fmt"

	"database/sql"
)

type SQLiteMindmapStorage struct {
	db *sql.DB
}

func NewSQLiteMindmapStorage(db *sql.DB) *SQLiteMindmapStorage {
	return &SQLiteMindmapStorage{db: db}
}

func (s *SQLiteMindmapStorage) MindmapAdd(mindmapName string, owner string, isPublic bool) (int, error) {
	result, err := s.db.Exec("INSERT INTO mindmaps (name, owner, is_public) VALUES (?, ?, ?)", mindmapName, owner, isPublic)
	if err != nil {
		return 0, fmt.Errorf("failed to add data: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return int(id), nil
}

func (s *SQLiteMindmapStorage) MindmapDelete(mindmapName string, username string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the data ID
	var mindmapID int
	err = tx.QueryRow("SELECT id FROM mindmaps WHERE name = ? AND owner = ?", mindmapName, username).Scan(&mindmapID)
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

func (s *SQLiteMindmapStorage) MindmapGetAll(username string) ([]MindmapInfo, error) {
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

func (s *SQLiteMindmapStorage) MindmapExists(mindmapName string, username string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM mindmaps WHERE name = ? AND (owner = ? OR is_public = 1)", mindmapName, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check data existence: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteMindmapStorage) MindmapPermission(mindmapName string, username string, setPublic ...bool) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var isOwner bool
	var isPublic bool
	err = tx.QueryRow("SELECT owner = ?, is_public FROM mindmaps WHERE name = ?", username, mindmapName).Scan(&isOwner, &isPublic)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("mindmap '%s' does not exist", mindmapName)
		}
		return false, fmt.Errorf("failed to check mindmap permission: %w", err)
	}

	if len(setPublic) > 0 {
		if !isOwner {
			return false, fmt.Errorf("user '%s' does not have permission to modify mindmap '%s'", username, mindmapName)
		}

		newPublicStatus := setPublic[0]
		if isPublic != newPublicStatus {
			_, err = tx.Exec("UPDATE mindmaps SET is_public = ? WHERE name = ?", newPublicStatus, mindmapName)
			if err != nil {
				return false, fmt.Errorf("failed to update mindmap access: %w", err)
			}
			isPublic = newPublicStatus
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return isOwner || isPublic, nil
}
