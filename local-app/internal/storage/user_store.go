// Package storage provides functionality for persisting and retrieving Mindnoscape data.
// This file implements the storage operations for users using SQLite.
package storage

import (
	"database/sql"
	"fmt"
	"golang.org/x/crypto/bcrypt"

	"mindnoscape/local-app/internal/models"
)

// UserStore defines the interface for user-related storage operations.
type UserStore interface {
	UserAdd(username, hashedPassword string) error
	UserDelete(username string) error
	UserExists(username string) (bool, error)
	UserGet(username string) (*models.User, error)
	UserUpdate(oldUsername, newUsername, newHashedPassword string) error
	UserAuthenticate(username, password string) (bool, error)
}

// SQLiteUserStorage implements the UserStore interface using SQLite.
type SQLiteUserStorage struct {
	db *sql.DB
}

// NewSQLiteUserStorage creates a new SQLiteUserStorage instance.
func NewSQLiteUserStorage(db *sql.DB) *SQLiteUserStorage {
	return &SQLiteUserStorage{db: db}
}

// UserAdd adds a new user to the database.
func (s *SQLiteUserStorage) UserAdd(username, password string) error {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert the new user with hashed password
	_, err = s.db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}

// UserDelete removes a user and all associated data from the database.
func (s *SQLiteUserStorage) UserDelete(username string) error {
	// Start a transaction
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

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UserExists checks if a user with the given username exists.
func (s *SQLiteUserStorage) UserExists(username string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

// UserGet retrieves a user by username.
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

// UserUpdate updates a user's username or password.
func (s *SQLiteUserStorage) UserUpdate(oldUsername, newUsername, newPassword string) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update username if provided
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

	// Update password if provided
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

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UserAuthenticate verifies a user's credentials.
func (s *SQLiteUserStorage) UserAuthenticate(username, password string) (bool, error) {
	// Retrieve the stored hash
	var hashedPassword []byte
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to get user for authentication: %w", err)
	}

	// Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("failed to compare passwords: %w", err)
	}

	return true, nil
}
