package cli

import (
	"fmt"
	"io"
	//	"mindnoscape/local-app/internal/ui"
)

// SystemExit handles the 'system exit' command
func (c *CLI) SystemExit() error {
	c.UI.Println("Exiting...")
	return io.EOF
}

// SystemQuit handles the 'system quit' command
func (c *CLI) SystemQuit() error {
	return c.SystemExit()
}

// SystemInfo handles the 'system' command
func (c *CLI) SystemInfo(args []string) error {
	c.UI.Println("System Information:")

	currentUser := c.Data.UserManager.UserGet()
	if currentUser == "" {
		c.UI.Println("Current User: No user selected")
	} else {
		c.UI.Printf("Current User: %s\n", currentUser)
	}

	if c.Data.MindmapManager.CurrentMindmap != nil {
		c.UI.Printf("Current Mindmap: %s\n", c.Data.MindmapManager.CurrentMindmap.Name)
	} else {
		c.UI.Println("Current Mindmap: None selected")
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

// ExecuteSystemCommand routes the system command to the appropriate handler
func (c *CLI) ExecuteSystemCommand(args []string) error {
	if len(args) == 0 {
		return c.SystemInfo(args)
	}

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
