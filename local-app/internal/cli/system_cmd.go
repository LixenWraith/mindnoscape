// Package cli provides the command-line interface functionality for Mindnoscape.
// This file contains handlers for system-related commands.
package cli

import (
	"fmt"
	"io"
	//	"mindnoscape/local-app/internal/ui"
)

// SystemExit handles the 'system exit' command, which terminates the program.
func (c *CLI) SystemExit() error {
	c.UI.Message("Exiting...")
	return io.EOF
}

// SystemQuit handles the 'system quit' command, which is equivalent to 'system exit'.
func (c *CLI) SystemQuit() error {
	return c.SystemExit()
}

// SystemInfo handles the 'system' command, displaying general system information.
func (c *CLI) SystemInfo(args []string) error {
	// Print system information header
	c.UI.Message("System Information:")

	// Display current user information
	currentUser := c.Data.UserManager.UserGet()
	if currentUser.Username == "" {
		c.UI.Info("No user selected.")
	} else {
		c.UI.Info(fmt.Sprintf("Current User: %s", currentUser.Username))
	}

	// Display current mindmap information
	if c.Data.MindmapManager.MindmapGet() != nil {
		c.UI.Info(fmt.Sprintf("Current Mindmap: %s", c.Data.MindmapManager.MindmapGet().Name))
	} else {
		c.UI.Info("No mindmap selected.")
	}

	// TODO: Add more system information in future implementations
	// Some ideas for future additions:
	// - Number of users in the system
	// - Total number of mindmaps
	// - Database size
	// - Application version
	// - Last backup time
	// - Current configuration settings

	return nil
}

// ExecuteSystemCommand routes the system command to the appropriate handler.
func (c *CLI) ExecuteSystemCommand(args []string) error {
	// If no arguments, show system info
	if len(args) == 0 {
		return c.SystemInfo(args)
	}

	// Route to specific system command handlers
	operation := args[0]
	switch operation {
	case "exit":
		return c.SystemExit()
	case "quit":
		return c.SystemQuit()
	default:
		return fmt.Errorf("unknown system operation: %s", operation)
	}
}
