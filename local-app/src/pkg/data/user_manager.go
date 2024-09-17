// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to user management.
package data

import (
	"context"
	"crypto/subtle"
	"fmt"

	"mindnoscape/local-app/src/pkg/event"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
	"mindnoscape/local-app/src/pkg/storage"
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
	ctx := context.Background()
	logger.Info(ctx, "Creating new UserManager", nil)

	if userStore == nil {
		logger.Error(ctx, "UserStore not initialized", nil)
		return nil, fmt.Errorf("userStore not initialized")
	}
	if eventManager == nil {
		logger.Error(ctx, "EventManager not initialized", nil)
		return nil, fmt.Errorf("eventManager not initialized")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger not initialized")
	}

	um := &UserManager{
		userStore:    userStore,
		eventManager: eventManager,
		logger:       logger,
	}

	logger.Info(ctx, "UserManager created successfully", nil)
	return um, nil
}

// UserAdd creates a new user with the given username, password, and active state.
func (um *UserManager) UserAdd(newUserInfo model.UserInfo) (int, error) {
	ctx := context.Background()
	um.logger.Info(ctx, "Adding new user", log.Fields{"username": newUserInfo.Username})

	// Check if the user already exists
	existingUsers, err := um.UserGet(model.UserInfo{Username: newUserInfo.Username}, model.UserFilter{Username: true})
	if err != nil {
		um.logger.Error(ctx, "Error checking user existence", log.Fields{"error": err, "username": newUserInfo.Username})
		return 0, fmt.Errorf("error checking user existence: %w", err)
	}
	if len(existingUsers) > 0 {
		um.logger.Warn(ctx, "User already exists", log.Fields{"username": newUserInfo.Username})
		return 0, fmt.Errorf("user '%s' already exists", newUserInfo.Username)
	}

	// Add the new user using the storage layer
	userID, err := um.userStore.UserAdd(newUserInfo)
	if err != nil {
		um.logger.Error(ctx, "Failed to create user", log.Fields{"error": err, "username": newUserInfo.Username})
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	um.logger.Info(ctx, "User added successfully", log.Fields{"userID": userID, "username": newUserInfo.Username})
	return userID, nil
}

// UserAuthenticate verifies a user's credentials.
func (um *UserManager) UserAuthenticate(userInfo model.UserInfo) (bool, error) {
	ctx := context.Background()
	um.logger.Info(ctx, "Authenticating user", log.Fields{"username": userInfo.Username})

	// Get the user by username
	users, err := um.UserGet(model.UserInfo{Username: userInfo.Username}, model.UserFilter{Username: true, PasswordHash: true})
	if err != nil {
		um.logger.Error(ctx, "Error retrieving user", log.Fields{"error": err, "username": userInfo.Username})
		return false, fmt.Errorf("error retrieving user: %w", err)
	}
	if len(users) == 0 {
		um.logger.Warn(ctx, "User doesn't exist", log.Fields{"username": userInfo.Username})
		return false, fmt.Errorf("user '%s' doesn't exist", userInfo.Username)
	}

	// Compare the password hashes
	storedUser := users[0]
	if subtle.ConstantTimeCompare(storedUser.PasswordHash, userInfo.PasswordHash) == 1 {
		um.logger.Info(ctx, "User authenticated successfully", log.Fields{"username": userInfo.Username})
		return true, nil
	}

	um.logger.Warn(ctx, "Authentication failed", log.Fields{"username": userInfo.Username})
	return false, nil
}

// UserGet retrieves users based on the provided info and filter.
func (um *UserManager) UserGet(userInfo model.UserInfo, userFilter model.UserFilter) ([]*model.User, error) {
	ctx := context.Background()
	um.logger.Info(ctx, "Retrieving users", log.Fields{"filter": userFilter})

	users, err := um.userStore.UserGet(userInfo, userFilter)
	if err != nil {
		um.logger.Error(ctx, "Failed to get users", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	um.logger.Info(ctx, "Users retrieved successfully", log.Fields{"count": len(users)})
	return users, nil
}

// UserUpdate updates an existing user's information.
func (um *UserManager) UserUpdate(user *model.User, userUpdateInfo model.UserInfo, userFilter model.UserFilter) error {
	ctx := context.Background()
	um.logger.Info(ctx, "Updating user", log.Fields{"userID": user.ID, "username": user.Username})

	err := um.userStore.UserUpdate(user, userUpdateInfo, userFilter)
	if err != nil {
		um.logger.Error(ctx, "Failed to update user", log.Fields{"error": err, "userID": user.ID})
		return fmt.Errorf("failed to update user: %w", err)
	}

	um.logger.Info(ctx, "User updated successfully", log.Fields{"userID": user.ID, "username": user.Username})
	return nil
}

// UserDelete removes a user and all associated data.
func (um *UserManager) UserDelete(user *model.User) error {
	ctx := context.Background()
	um.logger.Info(ctx, "Deleting user", log.Fields{"userID": user.ID, "username": user.Username})

	err := um.userStore.UserDelete(user)
	if err != nil {
		um.logger.Error(ctx, "Failed to delete user", log.Fields{"error": err, "userID": user.ID})
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Publish UserDeleted event
	um.eventManager.Publish(event.Event{
		Type: event.UserDeleted,
		Data: user,
	})

	um.logger.Info(ctx, "User deleted successfully", log.Fields{"userID": user.ID, "username": user.Username})
	return nil
}

// UserToInfo extracts UserInfo from a User instance
func (um *UserManager) UserToInfo(user *model.User) model.UserInfo {
	// Calculate the mindmap count if Mindmaps is not nil
	var mindmapCount *int
	if user.Mindmaps != nil {
		count := len(user.Mindmaps)
		mindmapCount = &count
	}

	// Create and return the UserInfo struct
	return model.UserInfo{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Active:       user.Active,
		MindmapCount: mindmapCount,
	}
}
