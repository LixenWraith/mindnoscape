package cli

import (
	"fmt"
)

func (c *CLI) UserInfo(args []string) error {
	c.UI.Printf("Current user: %s\n", c.CurrentUser)
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

	err = c.Data.UserManager.CreateUser(username, password)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("User '%s' created successfully", username))
	return nil
}

// UserMod handles the 'user mod' command
func (c *CLI) UserMod(args []string) error {
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

	err = c.Data.UserManager.ModifyUser(username, newUsername, newPassword)
	if err != nil {
		return err
	}

	c.UI.Success("User updated successfully")
	return nil
}

// UserDel handles the 'user del' command
func (c *CLI) UserDel(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: user del <username>")
	}

	username := args[0]
	password, err := c.promptForPassword("Enter password to confirm deletion: ")
	if err != nil {
		return err
	}

	authenticated, err := c.Data.UserManager.AuthenticateUser(username, password)
	if err != nil {
		return fmt.Errorf("authentication error: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("invalid username or password")
	}

	err = c.Data.UserManager.DeleteUser(username)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("User '%s' deleted successfully", username))
	return nil
}

// UserSelect handles the 'user select' command
func (c *CLI) UserSelect(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: user select <username>")
	}

	username := args[0]
	var password string
	var err error

	if username == "guest" {
		password = ""
	} else {
		password, err = c.promptForPassword("Enter password: ")
		if err != nil {
			return err
		}
	}

	authenticated, err := c.Data.UserManager.AuthenticateUser(username, password)
	if err != nil {
		return fmt.Errorf("authentication error: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("invalid username or password")
	}

	err = c.Data.ChangeUser(username)
	if err != nil {
		return fmt.Errorf("failed to switch user: %v", err)
	}

	c.CurrentUser = username
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
		return c.MindmapInfo(args)
	}

	operation := args[0]
	switch operation {
	case "add":
		return c.UserAdd(args[1:])
	case "mod":
		return c.UserMod(args[1:])
	case "del":
		return c.UserDel(args[1:])
	case "select":
		return c.UserSelect(args[1:])
	case "list":
		return c.UserList(args[1:])
	default:
		return fmt.Errorf("unknown user operation: %s", operation)
	}
}
