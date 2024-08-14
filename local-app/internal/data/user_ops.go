package data

import (
	"fmt"

	"mindnoscape/local-app/internal/storage"
)

// UserOperations defines the interface for user-related operations
type UserOperations interface {
	CreateUser(username, password string) error
	DeleteUser(username string) error
	ModifyUser(oldUsername, newUsername, newPassword string) error
	AuthenticateUser(username, password string) (bool, error)
	ChangeUser(username string) error
	GetCurrentUser() string
	GetUserMindmaps(username string) ([]storage.MindmapInfo, error)
}

type UserManager struct {
	mm *MindmapManager
}

func NewUserManager(mm *MindmapManager) *UserManager {
	return &UserManager{mm: mm}
}

// Ensure UserManager implements UserOperations
var _ UserOperations = (*UserManager)(nil)

func (um *UserManager) CreateUser(username, password string) error {
	exists, err := um.mm.Store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if exists {
		return fmt.Errorf("user '%s' already exists", username)
	}

	err = um.mm.Store.AddUser(username, password)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (um *UserManager) DeleteUser(username string) error {
	// Check if the user exists
	exists, err := um.mm.Store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	// Delete user and their mindmaps
	err = um.mm.Store.DeleteUser(username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// If the deleted user was the current user, switch to guest
	if um.mm.CurrentUser == username {
		um.mm.CurrentUser = "guest"
		_ = um.ChangeUser("guest")
	}

	return nil
}

func (um *UserManager) ModifyUser(oldUsername, newUsername, newPassword string) error {
	// Check if the old username exists
	exists, err := um.mm.Store.UserExists(oldUsername)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", oldUsername)
	}

	// If newUsername is provided, check if it already exists
	if newUsername != "" && newUsername != oldUsername {
		exists, err = um.mm.Store.UserExists(newUsername)
		if err != nil {
			return fmt.Errorf("error checking new username existence: %w", err)
		}
		if exists {
			return fmt.Errorf("new username '%s' already exists", newUsername)
		}
	}

	err = um.mm.Store.ModifyUser(oldUsername, newUsername, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Update current user if it was modified
	if um.mm.CurrentUser == oldUsername && newUsername != "" {
		um.mm.CurrentUser = newUsername
	}

	return nil
}

func (um *UserManager) AuthenticateUser(username, password string) (bool, error) {
	authenticated, err := um.mm.Store.AuthenticateUser(username, password)
	if err != nil {
		return false, fmt.Errorf("authentication error: %w", err)
	}
	return authenticated, nil
}

func (um *UserManager) ChangeUser(username string) error {
	// Check if the user exists
	exists, err := um.mm.Store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists && username != "guest" {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	// Change user in MindmapManager
	err = um.mm.ChangeUser(username)
	if err != nil {
		return fmt.Errorf("failed to change user: %w", err)
	}

	return nil
}

func (um *UserManager) GetCurrentUser() string {
	return um.mm.CurrentUser
}

func (um *UserManager) GetUserMindmaps(username string) ([]storage.MindmapInfo, error) {
	mindmaps, err := um.mm.Store.GetAllMindmaps(username)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user mindmaps: %w", err)
	}
	return mindmaps, nil
}
