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
	c.UI.Message("System Information:")

	userCount, err := c.Data.UserManager.UserCount()
	if err != nil {
		return fmt.Errorf("failed to get user count: %w", err)
	}
	c.UI.Info(fmt.Sprintf("Number of users: %d", userCount))

	mindmapCount, err := c.Data.MindmapManager.MindmapCount()
	if err != nil {
		return fmt.Errorf("failed to get mindmap count: %w", err)
	}
	c.UI.Info(fmt.Sprintf("Number of mindmaps: %d", mindmapCount))

	if c.CurrentUser != nil {
		c.UI.Info(fmt.Sprintf("Current User: %s", c.CurrentUser.Username))
	} else {
		c.UI.Info("No user selected.")
	}

	if c.CurrentMindmap != nil {
		c.UI.Info(fmt.Sprintf("Current Mindmap: %s", c.CurrentMindmap.Name))
		c.UI.Info(fmt.Sprintf("Number of nodes: %d", c.CurrentMindmap.NodeCount))
		c.UI.Info(fmt.Sprintf("Depth: %d", c.CurrentMindmap.Depth))
	} else {
		c.UI.Info("No mindmap selected.")
	}

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
