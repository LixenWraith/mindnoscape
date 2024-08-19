// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to user management.
package data

import (
	"fmt"
	"mindnoscape/local-app/internal/models"

	"mindnoscape/local-app/internal/config"
	"mindnoscape/local-app/internal/storage"
)

// UserOperations defines the interface for user-related operations
type UserOperations interface {
	UserAdd(username, password string) error
	UserDelete(username string) error
	UserUpdate(oldUsername, newUsername, newPassword string) error
	UserAuthenticate(username, password string) (bool, error)
	UserSelect(username string) error
	UserReset()
	UserGet() *models.UserInfo
	UserExists(username string) (bool, error)
}

// UserManager handles all user-related operations and maintains the current user state.
type UserManager struct {
	userStore      storage.UserStore
	currentUser    *models.UserInfo
	config         *config.Config
	mindmapManager *MindmapManager
}

// NewUserManager creates a new UserManager instance.
func NewUserManager(userStore storage.UserStore, cfg *config.Config, mindmapManager *MindmapManager) (*UserManager, error) {
	if userStore == nil {
		return nil, fmt.Errorf("failed to initialize user storage")
	}

	return &UserManager{
		userStore: userStore,
		currentUser: &models.UserInfo{
			Username: "",
		},
		config:         cfg,
		mindmapManager: mindmapManager,
	}, nil
}

// UserExists checks if a user with the given username exists.
func (um *UserManager) UserExists(username string) (bool, error) {
	return um.userStore.UserExists(username)
}

func (um *UserManager) UserCount() (int, error) {
	count, err := um.userStore.UserCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}
	return count, nil
}

// UserAdd creates a new user with the given username and password.
func (um *UserManager) UserAdd(username, password string) error {
	// Check if the user already exists
	exists, err := um.userStore.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if exists {
		return fmt.Errorf("user '%s' already exists", username)
	}

	// Add the new user
	err = um.userStore.UserAdd(username, password)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UserDelete removes a user and all associated data.
func (um *UserManager) UserDelete(username string) error {
	// Check if the user exists
	exists, err := um.userStore.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	// Delete user
	err = um.userStore.UserDelete(username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// If the deleted user was the current user, switch to guest
	if um.currentUser.Username == username {
		um.currentUser = &models.UserInfo{
			Username: ""}
	}

	return nil
}

// UserUpdate updates an existing user's username or password.
func (um *UserManager) UserUpdate(oldUsername, newUsername, newPassword string) error {
	// Check if the old username exists
	exists, err := um.userStore.UserExists(oldUsername)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", oldUsername)
	}

	// If newUsername is provided, check if it already exists
	if newUsername != "" && newUsername != oldUsername {
		exists, err = um.userStore.UserExists(newUsername)
		if err != nil {
			return fmt.Errorf("error checking new username existence: %w", err)
		}
		if exists {
			return fmt.Errorf("new username '%s' already exists", newUsername)
		}
	}

	// Prevent changing password for default user
	if oldUsername == um.config.DefaultUser && newPassword != "" {
		return fmt.Errorf("cannot change password for default user")
	}

	// Update the user
	err = um.userStore.UserUpdate(oldUsername, newUsername, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Update current user if it was modified
	if um.currentUser.Username == oldUsername && newUsername != "" {
		um.currentUser.Username = newUsername
	}

	return nil
}

// UserAuthenticate verifies a user's credentials.
func (um *UserManager) UserAuthenticate(username, password string) (bool, error) {
	authenticated, err := um.userStore.UserAuthenticate(username, password)
	if err != nil {
		return false, fmt.Errorf("authentication error: %w", err)
	}
	return authenticated, nil
}

// UserSelect sets the current user.
func (um *UserManager) UserSelect(username string) error {
	// If username is empty, deselect the current user
	if username == "" {
		um.UserReset()
		return nil
	}

	// Check if the user exists
	exists, err := um.userStore.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	// Set the current user and reset the mindmap
	um.currentUser.Username = username
	if um.mindmapManager != nil {
		um.mindmapManager.MindmapReset()
	}
	um.mindmapManager.currentUser = username
	return nil
}

// UserReset clears the current user selection.
func (um *UserManager) UserReset() {
	um.currentUser = &models.UserInfo{}
	if um.mindmapManager != nil {
		um.mindmapManager.MindmapReset()
	}
}

// UserGet returns the currently selected user.
func (um *UserManager) UserGet() *models.UserInfo {
	return um.currentUser
}
