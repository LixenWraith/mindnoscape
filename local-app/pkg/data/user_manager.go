// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to user management.
package data

import (
	"crypto/subtle"
	"fmt"

	"mindnoscape/local-app/pkg/event"
	"mindnoscape/local-app/pkg/log"
	"mindnoscape/local-app/pkg/model"
	"mindnoscape/local-app/pkg/storage"
)

// UserOperations defines the interface for user-related operations
type UserOperations interface {
	UserAdd(newUserInfo model.UserInfo) (int, error)
	UserAuthenticate(userInfo model.UserInfo) (bool, error)
	UserGet(userInfo model.UserInfo, userFilter model.UserFilter) ([]*model.User, error)
	UserToInfo(user *model.User) model.UserInfo
	UserUpdate(user *model.User, userUpdateInfo model.UserInfo, userFilter model.UserFilter) error
	UserDelete(user *model.User) error
}

// todo: add input checks and sanitization

// UserManager handles all user-related operations and maintains the current user state.
type UserManager struct {
	userStore    storage.UserStore
	eventManager *event.EventManager
	logger       *log.Logger
}

// NewUserManager creates a new UserManager instance.
func NewUserManager(userStore storage.UserStore, eventManager *event.EventManager, logger *log.Logger) (*UserManager, error) {
	if userStore == nil {
		return nil, fmt.Errorf("userStore not initialized")
	}
	if eventManager == nil {
		return nil, fmt.Errorf("eventManager not initialized")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger not initialized")
	}
	return &UserManager{
		userStore:    userStore,
		eventManager: eventManager,
		logger:       logger,
	}, nil
}

// UserAdd creates a new user with the given username, password, and active state.
func (um *UserManager) UserAdd(newUserInfo model.UserInfo) (int, error) {
	// Check if the user already exists
	existingUsers, err := um.UserGet(model.UserInfo{Username: newUserInfo.Username}, model.UserFilter{Username: true})
	if err != nil {
		return 0, fmt.Errorf("error checking user existence: %w", err)
	}
	if len(existingUsers) > 0 {
		return 0, fmt.Errorf("user '%s' already exists", newUserInfo.Username)
	}

	// Add the new user using the storage layer
	userID, err := um.userStore.UserAdd(newUserInfo)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// UserAuthenticate verifies a user's credentials.
func (um *UserManager) UserAuthenticate(userInfo model.UserInfo) (bool, error) {
	// Get the user by username
	users, err := um.UserGet(model.UserInfo{Username: userInfo.Username}, model.UserFilter{Username: true, PasswordHash: true})
	if err != nil {
		return false, fmt.Errorf("error retrieving user: %w", err)
	}
	if len(users) == 0 {
		return false, fmt.Errorf("user '%s' doesn't exist", userInfo.Username)
	}

	// Compare the password hashes
	storedUser := users[0]
	if subtle.ConstantTimeCompare(storedUser.PasswordHash, userInfo.PasswordHash) == 1 {
		return true, nil
	}

	return false, nil
}

// UserGet retrieves users based on the provided info and filter.
func (um *UserManager) UserGet(userInfo model.UserInfo, userFilter model.UserFilter) ([]*model.User, error) {
	users, err := um.userStore.UserGet(userInfo, userFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	return users, nil
}

// UserUpdate updates an existing user's information.
func (um *UserManager) UserUpdate(user *model.User, userUpdateInfo model.UserInfo, userFilter model.UserFilter) error {
	err := um.userStore.UserUpdate(user, userUpdateInfo, userFilter)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// UserDelete removes a user and all associated data.
func (um *UserManager) UserDelete(user *model.User) error {
	err := um.userStore.UserDelete(user)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Publish UserDeleted event
	um.eventManager.Publish(event.Event{
		Type: event.UserDeleted,
		Data: user,
	})

	return nil
}

// UserToInfo extracts UserInfo from a User instance
func (um *UserManager) UserToInfo(user *model.User) model.UserInfo {
	var mindmapCount *int

	if user.Mindmaps != nil {
		count := len(user.Mindmaps)
		mindmapCount = &count
	}

	return model.UserInfo{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Active:       user.Active,
		MindmapCount: mindmapCount,
	}
}
