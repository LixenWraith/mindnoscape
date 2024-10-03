package session

import (
	"context"
	"errors"
	"fmt"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// handleUserAdd handles the user add command
func handleUserAdd(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling user add command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		sm.logger.Error(ctx, "Invalid number of arguments for user add", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user add")
	}

	username := cmd.Args[0]
	var password string
	if len(cmd.Args) == 2 {
		password = cmd.Args[1]
	}

	sm.logger.Debug(ctx, "Creating user info", log.Fields{"username": username})
	userInfo := model.UserInfo{
		Username:     username,
		PasswordHash: []byte(password), // Password is already hashed
	}

	userID, err := sm.dataManager.UserManager.UserAdd(userInfo)
	if err != nil {
		sm.logger.Error(ctx, "Failed to add user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to add user: %w", err)
	}

	sm.logger.Info(ctx, "User added successfully", log.Fields{"userID": userID})
	return userID, nil
}

// handleUserUpdate handles the user update command
func handleUserUpdate(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling user update command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 3 {
		sm.logger.Error(ctx, "Invalid number of arguments for user update", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user update")
	}

	if session.User == nil {
		sm.logger.Error(ctx, "No user selected", nil)
		return nil, fmt.Errorf("no user selected")
	}

	username := cmd.Args[0]
	if username != session.User.Username {
		sm.logger.Error(ctx, "Can only update the current user", log.Fields{"requestedUser": username, "currentUser": session.User.Username})
		return nil, fmt.Errorf("can only update the current user")
	}

	updateInfo := model.UserInfo{}
	updateFilter := model.UserFilter{}

	if len(cmd.Args) > 1 {
		updateInfo.Username = cmd.Args[1]
		updateFilter.Username = true
		sm.logger.Debug(ctx, "Updating username", log.Fields{"newUsername": updateInfo.Username})
	}
	if len(cmd.Args) > 2 {
		updateInfo.PasswordHash = []byte(cmd.Args[2]) // Password is already hashed
		updateFilter.PasswordHash = true
		sm.logger.Debug(ctx, "Updating password", nil)
	}

	err := sm.dataManager.UserManager.UserUpdate(session.User, updateInfo, updateFilter)
	if err != nil {
		sm.logger.Error(ctx, "Failed to update user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Update the session's User object if the username was changed
	if updateFilter.Username {
		session.User.Username = updateInfo.Username
		sm.logger.Debug(ctx, "Updated session user", log.Fields{"newUsername": session.User.Username})
	}

	sm.logger.Info(ctx, "User updated successfully", nil)
	return nil, nil
}

// handleUserDelete handles the user delete command
func handleUserDelete(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling user delete command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) != 1 {
		sm.logger.Error(ctx, "Invalid number of arguments for user delete", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user delete")
	}

	if session.User == nil {
		sm.logger.Error(ctx, "No user selected", nil)
		return nil, fmt.Errorf("no user selected")
	}

	username := cmd.Args[0]
	if username != session.User.Username {
		sm.logger.Error(ctx, "Can only delete the current user", log.Fields{"requestedUser": username, "currentUser": session.User.Username})
		return nil, fmt.Errorf("can only delete the current user")
	}

	err := sm.dataManager.UserManager.UserDelete(session.User)
	if err != nil {
		sm.logger.Error(ctx, "Failed to delete user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	// Clear the session's User and Mindmap
	session.User = nil
	session.Mindmap = nil
	sm.logger.Debug(ctx, "Cleared session user and mindmap", nil)

	sm.logger.Info(ctx, "User deleted successfully", nil)
	return nil, nil
}

// handleUserSelect handles the user select command
func handleUserSelect(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling user select command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) != 1 {
		sm.logger.Error(ctx, "Invalid number of arguments for user select", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user select")
	}

	username := cmd.Args[0]
	sm.logger.Debug(ctx, "Attempting to select user", log.Fields{"username": username})

	users, err := sm.dataManager.UserManager.UserGet(model.UserInfo{Username: username}, model.UserFilter{Username: true})
	if err != nil {
		sm.logger.Error(ctx, "Failed to get user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if len(users) == 0 {
		sm.logger.Warn(ctx, "User not found", log.Fields{"username": username})
		return nil, fmt.Errorf("user not found: %s", username)
	}
	user := users[0]

	session.User = user
	sm.logger.Debug(ctx, "User selected and set in session", log.Fields{"username": user.Username})

	sm.logger.Info(ctx, "User selected successfully", log.Fields{"username": username})
	return fmt.Sprintf("User '%s' selected successfully", username), nil
}
