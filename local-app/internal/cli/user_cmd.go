// Package cli provides the command-line interface functionality for Mindnoscape.
// This file contains handlers for user-related commands.
package cli

import (
	"fmt"
	// "mindnoscape/local-app/internal/ui"
)

// UserInfo displays information about the currently selected user.
func (c *CLI) UserInfo(args []string) error {
	currentUser := c.Data.UserManager.UserGet()
	if currentUser.Username == "" {
		c.UI.Info("No user is currently selected.")
		return nil
	}

	c.UI.Message(fmt.Sprintf("Current user: %s", currentUser.Username))

	ownedMindmaps, err := c.Data.MindmapManager.MindmapCountOwned(currentUser.Username)
	if err != nil {
		return fmt.Errorf("failed to count owned mindmaps: %w", err)
	}
	c.UI.Info(fmt.Sprintf("Owned mindmaps: %d", ownedMindmaps))

	accessibleMindmaps, err := c.Data.MindmapManager.MindmapsCountPermitted(currentUser.Username)
	if err != nil {
		return fmt.Errorf("failed to count accessible mindmaps: %w", err)
	}
	c.UI.Info(fmt.Sprintf("Accessible mindmaps: %d", accessibleMindmaps))

	return nil
}

// UserAdd handles the 'user add' command, creating a new user account.
func (c *CLI) UserAdd(args []string) error {
	// Check for correct usage
	if len(args) < 1 {
		return fmt.Errorf("usage: user add <username> [password]")
	}

	// Get username and password
	username := args[0]
	var password string
	var err error

	if len(args) > 1 {
		password = args[1]
	} else {
		password, err = c.promptForPassword("Enter password for new user: ")
		if err != nil {
			return err
		}
	}

	// Create the new user
	err = c.Data.UserManager.UserAdd(username, password)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("User '%s' created successfully.", username))
	return nil
}

// UserUpdate handles the 'user mod' command, modifying an existing user account.
func (c *CLI) UserUpdate(args []string) error {
	// Check for correct usage
	if len(args) < 1 {
		return fmt.Errorf("usage: user update <username> [new_username] [new_password]")
	}

	// Get username and new details
	username := args[0]
	var newUsername, newPassword string
	var err error

	if len(args) > 1 {
		newUsername = args[1]
	} else {
		newUsername, err = c.promptForInput("Enter new username (leave empty to keep current): ")
		if err != nil {
			return err
		}
	}

	if len(args) > 2 {
		newPassword = args[2]
	} else {
		newPassword, err = c.promptForPassword("Enter new password (leave empty to keep current): ")
		if err != nil {
			return err
		}
	}

	// Update the user
	err = c.Data.UserManager.UserUpdate(username, newUsername, newPassword)
	if err != nil {
		return err
	}

	c.UI.Success("User updated successfully.")
	return nil
}

// UserDelete handles the 'user del' command, deleting an existing user account.
func (c *CLI) UserDelete(args []string) error {
	// Check for correct usage
	if len(args) < 1 {
		return fmt.Errorf("usage: user delete <username>")
	}

	// Get username and confirm with password
	username := args[0]
	password, err := c.promptForPassword("Enter password to confirm deletion: ")
	if err != nil {
		return err
	}

	// Authenticate and delete the user
	authenticated, err := c.Data.UserManager.UserAuthenticate(username, password)
	if err != nil {
		return fmt.Errorf("authentication error: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("invalid username or password")
	}

	err = c.Data.UserManager.UserDelete(username)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("User '%s' deleted successfully.", username))
	return nil
}

// UserSelect handles the 'user select' command, selecting a user as current user with authentication.
func (c *CLI) UserSelect(args []string) error {
	// If no arguments, deselect current user
	if len(args) == 0 {
		err := c.Data.UserManager.UserSelect("")
		if err != nil {
			return err
		}
		c.CurrentUser = nil
		c.CurrentMindmap = nil
		c.UI.Success("Deselected the current user.")
		return nil
	}

	// Get username and password
	username := args[0]
	password, err := c.promptForPassword(fmt.Sprintf("Enter password for %s (press Enter for empty password): ", username))
	if err != nil {
		return err
	}

	// Authenticate and select the user
	authenticated, err := c.Data.UserManager.UserAuthenticate(username, password)
	if err != nil {
		return fmt.Errorf("authentication error: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("invalid username or password")
	}

	err = c.Data.UserManager.UserSelect(username)
	if err != nil {
		return fmt.Errorf("failed to switch user: %v", err)
	}

	c.CurrentMindmap = nil
	c.CurrentUser = nil
	c.CurrentUser = c.Data.UserManager.UserGet()

	c.UI.Success(fmt.Sprintf("Selected user: %s", username))
	return nil
}

// UserList handles the 'user list' command (placeholder for future implementation).
func (c *CLI) UserList(args []string) error {
	c.UI.Info("User list functionality is not implemented yet.")
	return nil
}

// ExecuteUserCommand routes the user command to the appropriate handler.
func (c *CLI) ExecuteUserCommand(args []string) error {
	// If no arguments, show user info
	if len(args) == 0 {
		return c.UserInfo(args)
	}

	// Route to specific user command handlers
	operation := args[0]
	switch operation {
	case "add":
		return c.UserAdd(args[1:])
	case "update":
		return c.UserUpdate(args[1:])
	case "delete":
		return c.UserDelete(args[1:])
	case "select":
		return c.UserSelect(args[1:])
	case "list":
		return c.UserList(args[1:])
	default:
		return fmt.Errorf("unknown user operation: %s", operation)
	}
}
