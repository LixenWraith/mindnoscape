package data

import (
	"fmt"
)

// UserOperations defines the interface for user-related operations
type UserOperations interface {
	UserAdd(username, password string) error
	UserDelete(username string) error
	UserModify(oldUsername, newUsername, newPassword string) error
	UserAuthenticate(username, password string) (bool, error)
	UserSelect(username string) error
	UserGet() string
}

type UserManager struct {
	mm *MindmapManager
}

func NewUserManager(mm *MindmapManager) *UserManager {
	return &UserManager{mm: mm}
}

// Ensure UserManager implements UserOperations
var _ UserOperations = (*UserManager)(nil)

func (um *UserManager) UserAdd(username, password string) error {
	exists, err := um.mm.Store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if exists {
		return fmt.Errorf("user '%s' already exists", username)
	}

	err = um.mm.Store.UserAdd(username, password)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (um *UserManager) UserDelete(username string) error {
	// Check if the user exists
	exists, err := um.mm.Store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	// Delete user and their mindmaps
	err = um.mm.Store.UserDelete(username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// If the deleted user was the current user, switch to guest
	if um.mm.CurrentUser == username {
		um.mm.CurrentUser = "guest"
		_ = um.UserSelect("guest")
	}

	return nil
}

func (um *UserManager) UserModify(oldUsername, newUsername, newPassword string) error {
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

	err = um.mm.Store.UserModify(oldUsername, newUsername, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Update current user if it was modified
	if um.mm.CurrentUser == oldUsername && newUsername != "" {
		um.mm.CurrentUser = newUsername
	}

	return nil
}

func (um *UserManager) UserAuthenticate(username, password string) (bool, error) {
	authenticated, err := um.mm.Store.UserAuthenticate(username, password)
	if err != nil {
		return false, fmt.Errorf("authentication error: %w", err)
	}
	return authenticated, nil
}

func (um *UserManager) UserSelect(username string) error {
	// Check if the user exists
	exists, err := um.mm.Store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists && username != "guest" {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	// Change user in MindmapManager
	err = um.mm.UserSelect(username)
	if err != nil {
		return fmt.Errorf("failed to change user: %w", err)
	}

	return nil
}

func (um *UserManager) UserGet() string {
	return um.mm.CurrentUser
}
