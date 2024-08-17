package data

import (
	"fmt"

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
	UserGet() string
	UserExists(username string) (bool, error)
}

type UserManager struct {
	store          storage.UserStore
	currentUser    string
	config         *config.Config
	mindmapManager *MindmapManager
}

func NewUserManager(store storage.UserStore, cfg *config.Config, mindmapManager *MindmapManager) (*UserManager, error) {
	if store == nil {
		return nil, fmt.Errorf("failed to initialize user storage")
	}

	return &UserManager{
		store:          store,
		currentUser:    "",
		config:         cfg,
		mindmapManager: mindmapManager,
	}, nil
}

func (um *UserManager) UserExists(username string) (bool, error) {
	return um.store.UserExists(username)
}

func (um *UserManager) UserAdd(username, password string) error {
	exists, err := um.store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if exists {
		return fmt.Errorf("user '%s' already exists", username)
	}

	err = um.store.UserAdd(username, password)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (um *UserManager) UserDelete(username string) error {
	// Check if the user exists
	exists, err := um.store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	// Delete user
	err = um.store.UserDelete(username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// If the deleted user was the current user, switch to guest
	if um.currentUser == username {
		um.currentUser = ""
	}

	return nil
}

func (um *UserManager) UserUpdate(oldUsername, newUsername, newPassword string) error {
	// Check if the old username exists
	exists, err := um.store.UserExists(oldUsername)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", oldUsername)
	}

	// If newUsername is provided, check if it already exists
	if newUsername != "" && newUsername != oldUsername {
		exists, err = um.store.UserExists(newUsername)
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

	err = um.store.UserUpdate(oldUsername, newUsername, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Update current user if it was modified
	if um.currentUser == oldUsername && newUsername != "" {
		um.currentUser = newUsername
	}

	return nil
}

func (um *UserManager) UserAuthenticate(username, password string) (bool, error) {
	authenticated, err := um.store.UserAuthenticate(username, password)
	if err != nil {
		return false, fmt.Errorf("authentication error: %w", err)
	}
	return authenticated, nil
}

func (um *UserManager) UserSelect(username string) error {
	if username == "" {
		um.UserReset()
		return nil
	}

	exists, err := um.store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	um.currentUser = username
	if um.mindmapManager != nil {
		um.mindmapManager.MindmapReset()
	}
	um.mindmapManager.CurrentUser = username
	return nil
}

func (um *UserManager) UserReset() {
	um.currentUser = ""
	if um.mindmapManager != nil {
		um.mindmapManager.MindmapReset()
	}
}

func (um *UserManager) UserGet() string {
	return um.currentUser
}
