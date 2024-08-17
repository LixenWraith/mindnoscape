package storage

import (
	"fmt"

	"mindnoscape/local-app/internal/models"

	"database/sql"
	"golang.org/x/crypto/bcrypt"
)

type UserStore interface {
	UserAdd(username, hashedPassword string) error
	UserDelete(username string) error
	UserExists(username string) (bool, error)
	UserGet(username string) (*models.User, error)
	UserUpdate(oldUsername, newUsername, newHashedPassword string) error
	UserAuthenticate(username, password string) (bool, error)
}

type SQLiteUserStorage struct {
	db *sql.DB
}

func NewSQLiteUserStorage(db *sql.DB) *SQLiteUserStorage {
	return &SQLiteUserStorage{db: db}
}

func (s *SQLiteUserStorage) UserAdd(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = s.db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}

func (s *SQLiteUserStorage) UserDelete(username string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete user's mindmaps
	_, err = tx.Exec("DELETE FROM mindmaps WHERE owner = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user's mindmaps: %w", err)
	}

	// Delete user
	_, err = tx.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *SQLiteUserStorage) UserExists(username string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteUserStorage) UserGet(username string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow("SELECT username, password_hash FROM users WHERE username = ?", username).Scan(&user.Username, &user.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *SQLiteUserStorage) UserUpdate(oldUsername, newUsername, newPassword string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if newUsername != "" && newUsername != oldUsername {
		_, err = tx.Exec("UPDATE users SET username = ? WHERE username = ?", newUsername, oldUsername)
		if err != nil {
			return fmt.Errorf("failed to update username: %w", err)
		}

		_, err = tx.Exec("UPDATE mindmaps SET owner = ? WHERE owner = ?", newUsername, oldUsername)
		if err != nil {
			return fmt.Errorf("failed to update data ownership: %w", err)
		}
	}

	if newPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash new password: %w", err)
		}

		_, err = tx.Exec("UPDATE users SET password_hash = ? WHERE username = ?", hashedPassword, oldUsername)
		if err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *SQLiteUserStorage) UserAuthenticate(username, password string) (bool, error) {
	var hashedPassword []byte
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to get user for authentication: %w", err)
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("failed to compare passwords: %w", err)
	}

	return true, nil
}
