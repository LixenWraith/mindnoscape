package session

import (
	"errors"
	"fmt"

	"mindnoscape/local-app/pkg/model"
)

// handleUserAdd handles the user add command
func handleUserAdd(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return nil, errors.New("invalid number of arguments for user add")
	}

	username := cmd.Args[0]
	var password string
	if len(cmd.Args) == 2 {
		password = cmd.Args[1]
	}

	userInfo := model.UserInfo{
		Username:     username,
		PasswordHash: []byte(password), // Password is already hashed
	}

	userID, err := s.DataManager.UserManager.UserAdd(userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to add user: %w", err)
	}

	return userID, nil
}

// handleUserUpdate handles the user update command
func handleUserUpdate(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 3 {
		return nil, errors.New("invalid number of arguments for user update")
	}

	currentUser, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	username := cmd.Args[0]
	if username != currentUser.Username {
		return nil, fmt.Errorf("can only update the current user")
	}

	updateInfo := model.UserInfo{}
	updateFilter := model.UserFilter{}

	if len(cmd.Args) > 1 {
		updateInfo.Username = cmd.Args[1]
		updateFilter.Username = true
	}
	if len(cmd.Args) > 2 {
		updateInfo.PasswordHash = []byte(cmd.Args[2]) // Password is already hashed
		updateFilter.PasswordHash = true
	}

	err = s.DataManager.UserManager.UserUpdate(currentUser, updateInfo, updateFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Update the session's User object if the username was changed
	// todo: get the user instance again instead of setting it directly to confirm the change
	if updateFilter.Username {
		currentUser.Username = updateInfo.Username
	}

	return nil, nil
}

// handleUserDelete handles the user delete command
func handleUserDelete(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) != 1 {
		return nil, errors.New("invalid number of arguments for user delete")
	}

	currentUser, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	username := cmd.Args[0]
	if username != currentUser.Username {
		return nil, fmt.Errorf("can only delete the current user")
	}

	err = s.DataManager.UserManager.UserDelete(currentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	// Clear the session's User and Mindmap
	s.UserSet(nil)
	s.MindmapSet(nil)

	return nil, nil
}

// handleUserSelect handles the user select command
func handleUserSelect(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) != 1 {
		return nil, errors.New("invalid number of arguments for user select")
	}

	username := cmd.Args[0]
	users, err := s.DataManager.UserManager.UserGet(model.UserInfo{Username: username}, model.UserFilter{Username: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	user := users[0]

	s.UserSet(user)

	return fmt.Sprintf("User '%s' selected successfully", username), nil
}
