package session

import (
	"context"
	"errors"
	"fmt"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// handleUserAdd handles the user add command
func handleUserAdd(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling user add command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		s.logger.Error(ctx, "Invalid number of arguments for user add", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user add")
	}

	username := cmd.Args[0]
	var password string
	if len(cmd.Args) == 2 {
		password = cmd.Args[1]
	}

	s.logger.Debug(ctx, "Creating user info", log.Fields{"username": username})
	userInfo := model.UserInfo{
		Username:     username,
		PasswordHash: []byte(password), // Password is already hashed
	}

	userID, err := s.DataManager.UserManager.UserAdd(userInfo)
	if err != nil {
		s.logger.Error(ctx, "Failed to add user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to add user: %w", err)
	}

	s.logger.Info(ctx, "User added successfully", log.Fields{"userID": userID})
	return userID, nil
}

// handleUserUpdate handles the user update command
func handleUserUpdate(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling user update command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 3 {
		s.logger.Error(ctx, "Invalid number of arguments for user update", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user update")
	}

	currentUser, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	username := cmd.Args[0]
	if username != currentUser.Username {
		s.logger.Error(ctx, "Can only update the current user", log.Fields{"requestedUser": username, "currentUser": currentUser.Username})
		return nil, fmt.Errorf("can only update the current user")
	}

	updateInfo := model.UserInfo{}
	updateFilter := model.UserFilter{}

	if len(cmd.Args) > 1 {
		updateInfo.Username = cmd.Args[1]
		updateFilter.Username = true
		s.logger.Debug(ctx, "Updating username", log.Fields{"newUsername": updateInfo.Username})
	}
	if len(cmd.Args) > 2 {
		updateInfo.PasswordHash = []byte(cmd.Args[2]) // Password is already hashed
		updateFilter.PasswordHash = true
		s.logger.Debug(ctx, "Updating password", nil)
	}

	err = s.DataManager.UserManager.UserUpdate(currentUser, updateInfo, updateFilter)
	if err != nil {
		s.logger.Error(ctx, "Failed to update user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Update the session's User object if the username was changed
	if updateFilter.Username {
		currentUser.Username = updateInfo.Username
		s.logger.Debug(ctx, "Updated session user", log.Fields{"newUsername": currentUser.Username})
	}

	s.logger.Info(ctx, "User updated successfully", nil)
	return nil, nil
}

// handleUserDelete handles the user delete command
func handleUserDelete(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling user delete command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) != 1 {
		s.logger.Error(ctx, "Invalid number of arguments for user delete", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user delete")
	}

	currentUser, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	username := cmd.Args[0]
	if username != currentUser.Username {
		s.logger.Error(ctx, "Can only delete the current user", log.Fields{"requestedUser": username, "currentUser": currentUser.Username})
		return nil, fmt.Errorf("can only delete the current user")
	}

	err = s.DataManager.UserManager.UserDelete(currentUser)
	if err != nil {
		s.logger.Error(ctx, "Failed to delete user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	// Clear the session's User and Mindmap
	s.UserSet(nil)
	s.MindmapSet(nil)
	s.logger.Debug(ctx, "Cleared session user and mindmap", nil)

	s.logger.Info(ctx, "User deleted successfully", nil)
	return nil, nil
}

// handleUserSelect handles the user select command
func handleUserSelect(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling user select command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) != 1 {
		s.logger.Error(ctx, "Invalid number of arguments for user select", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("invalid number of arguments for user select")
	}

	username := cmd.Args[0]
	s.logger.Debug(ctx, "Attempting to select user", log.Fields{"username": username})

	users, err := s.DataManager.UserManager.UserGet(model.UserInfo{Username: username}, model.UserFilter{Username: true})
	if err != nil {
		s.logger.Error(ctx, "Failed to get user", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if len(users) == 0 {
		s.logger.Warn(ctx, "User not found", log.Fields{"username": username})
		return nil, fmt.Errorf("user not found: %s", username)
	}
	user := users[0]

	s.UserSet(user)
	s.logger.Debug(ctx, "User selected and set in session", log.Fields{"username": user.Username})

	s.logger.Info(ctx, "User selected successfully", log.Fields{"username": username})
	return fmt.Sprintf("User '%s' selected successfully", username), nil
}
