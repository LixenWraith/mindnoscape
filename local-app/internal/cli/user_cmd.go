package cli

import (
	"fmt"
	// "mindnoscape/local-app/internal/ui"
)

func (c *CLI) UserInfo(args []string) error {
	currentUser := c.Data.UserManager.UserGet()
	if currentUser == "" {
		c.UI.Info("No user is currently selected.")
	} else {
		c.UI.Printf("Current user: %s\n", currentUser)
	}
	return nil
}

// UserAdd handles the 'user add' command
func (c *CLI) UserAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: user add <username> [password]")
	}

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

	err = c.Data.UserManager.UserAdd(username, password)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("User '%s' created successfully", username))
	return nil
}

// UserUpdate handles the 'user mod' command
func (c *CLI) UserUpdate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: user mod <username> [new_username] [new_password]")
	}

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

	err = c.Data.UserManager.UserUpdate(username, newUsername, newPassword)
	if err != nil {
		return err
	}

	c.UI.Success("User updated successfully")
	return nil
}

// UserDelete handles the 'user del' command
func (c *CLI) UserDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: user del <username>")
	}

	username := args[0]
	password, err := c.promptForPassword("Enter password to confirm deletion: ")
	if err != nil {
		return err
	}

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

	c.UI.Success(fmt.Sprintf("User '%s' deleted successfully", username))
	return nil
}

// UserSelect handles the 'user select' command
func (c *CLI) UserSelect(args []string) error {
	if len(args) == 0 {
		err := c.Data.UserManager.UserSelect("")
		if err != nil {
			return err
		}
		c.UI.Success("Deselected the current user")
		return nil
	}

	username := args[0]
	password, err := c.promptForPassword(fmt.Sprintf("Enter password for %s (press Enter for empty password): ", username))
	if err != nil {
		return err
	}

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

	c.UI.Success(fmt.Sprintf("Switched to user: %s", username))
	c.UpdatePrompt()
	return nil
}

// UserList handles the 'user list' command (placeholder)
func (c *CLI) UserList(args []string) error {
	c.UI.Info("User list functionality is not implemented yet.")
	return nil
}

// ExecuteUserCommand routes the user command to the appropriate handler
func (c *CLI) ExecuteUserCommand(args []string) error {
	if len(args) == 0 {
		return c.UserInfo(args)
	}

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
