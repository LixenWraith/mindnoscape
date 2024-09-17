package storage

import (
	"context"
	"fmt"
	"time"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// UserStore defines the interface for user-related storage operations.
type UserStore interface {
	UserAdd(newUser model.UserInfo) (int, error)
	UserGet(userInfo model.UserInfo, userFilter model.UserFilter) ([]*model.User, error)
	UserUpdate(user *model.User, userUpdateInfo model.UserInfo, userFilter model.UserFilter) error
	UserDelete(user *model.User) error
}

// UserStorage implements the UserStore interface.
type UserStorage struct {
	storage *Storage
	logger  *log.Logger
}

// NewUserStorage creates a new UserStorage instance.
func NewUserStorage(storage *Storage) *UserStorage {
	return &UserStorage{
		storage: storage,
		logger:  storage.logger,
	}
}

// UserAdd adds a new user to the database.
func (s *UserStorage) UserAdd(newUser model.UserInfo) (int, error) {
	s.logger.Info(context.Background(), "Adding new user", log.Fields{"user": newUser})

	db := s.storage.GetDatabase()
	now := time.Now()

	err := db.Begin()
	if err != nil {
		s.logger.Error(context.Background(), "Failed to begin transaction", log.Fields{"error": err})
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer db.Rollback()

	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, active, created, updated) VALUES (?, ?, ?, ?, ?)",
		newUser.Username, newUser.PasswordHash, newUser.Active, now, now,
	)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to add user", log.Fields{"error": err, "user": newUser})
		return 0, fmt.Errorf("failed to add user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		s.logger.Error(context.Background(), "Failed to get last insert ID", log.Fields{"error": err})
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	if err := db.Commit(); err != nil {
		s.logger.Error(context.Background(), "Failed to commit transaction", log.Fields{"error": err})
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info(context.Background(), "User added successfully", log.Fields{"userID": id, "username": newUser.Username})
	return int(id), nil
}

// UserGet retrieves users based on the provided info and filter.
func (s *UserStorage) UserGet(userInfo model.UserInfo, userFilter model.UserFilter) ([]*model.User, error) {
	s.logger.Info(context.Background(), "Retrieving users", log.Fields{"filter": userFilter})

	db := s.storage.GetDatabase()
	query := "SELECT id, username, password_hash, active, created, updated FROM users WHERE 1=1"
	var args []interface{}

	if userFilter.ID {
		query += " AND id = ?"
		args = append(args, userInfo.ID)
	}
	if userFilter.Username {
		query += " AND username = ?"
		args = append(args, userInfo.Username)
	}
	if userFilter.Active {
		query += " AND active = ?"
		args = append(args, userInfo.Active)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to query users", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Active, &u.Created, &u.Updated)
		if err != nil {
			s.logger.Error(context.Background(), "Failed to scan user row", log.Fields{"error": err})
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, &u)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error(context.Background(), "Error iterating user rows", log.Fields{"error": err})
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	s.logger.Info(context.Background(), "Users retrieved successfully", log.Fields{"count": len(users)})
	return users, nil
}

// UserUpdate updates an existing user in the database.
func (s *UserStorage) UserUpdate(user *model.User, userUpdateInfo model.UserInfo, userFilter model.UserFilter) error {
	s.logger.Info(context.Background(), "Deleting user", log.Fields{"user": user})

	db := s.storage.GetDatabase()
	query := "UPDATE users SET updated = ? WHERE id = ?"
	args := []interface{}{time.Now(), user.ID}

	if userFilter.Username {
		query += ", username = ?"
		args = append(args, userUpdateInfo.Username)
	}
	if userFilter.PasswordHash {
		query += ", password_hash = ?"
		args = append(args, userUpdateInfo.PasswordHash)
	}
	if userFilter.Active {
		query += ", active = ?"
		args = append(args, userUpdateInfo.Active)
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to update user", log.Fields{"error": err, "user": user})
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info(context.Background(), "User updated successfully", log.Fields{"user": user})
	return nil
}

// UserDelete removes a user from the database.
func (s *UserStorage) UserDelete(user *model.User) error {
	s.logger.Info(context.Background(), "Deleting user", log.Fields{"user": user})

	db := s.storage.GetDatabase()
	_, err := db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to delete user", log.Fields{"error": err, "userID": user.ID, "username": user.Username})
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.Info(context.Background(), "User deleted successfully", log.Fields{"user": user})
	return nil
}
