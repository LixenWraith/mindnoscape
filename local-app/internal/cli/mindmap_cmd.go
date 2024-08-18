// Package cli provides the command-line interface functionality for Mindnoscape.
// This file contains handlers for mindmap-related commands.
package cli

import (
	"fmt"
)

// MindmapInfo displays information about the currently selected mindmap.
func (c *CLI) MindmapInfo(args []string) error {
	if c.Data.MindmapManager.MindmapGet() == nil {
		c.UI.Info("No mindmap selected.")
		return nil
	}
	c.UI.Info(fmt.Sprintf("Current mindmap: %s", c.Data.MindmapManager.MindmapGet().Name))
	return nil
}

// MindmapAdd handles the 'mindmap add' command, creating a new mindmap.
func (c *CLI) MindmapAdd(args []string) error {
	// Check for correct usage
	if len(args) != 1 {
		return fmt.Errorf("usage: mindmap add <mindmap name>")
	}

	// Create the new mindmap
	name := args[0]
	var err = c.Data.MindmapManager.MindmapAdd(name, false) // Default to private mindmaps
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("New mindmap '%s' created.", name))
	return nil
}

// MindmapUpdate handles the 'mindmap update' command (placeholder for future implementation).
func (c *CLI) MindmapUpdate(args []string) error {
	c.UI.Info("Mindmap update functionality is not implemented yet.")
	return nil
}

// MindmapDelete handles the 'mindmap delete' command, deleting an existing mindmap.
func (c *CLI) MindmapDelete(args []string) error {
	if len(args) == 0 {
		// Clear all mindmaps owned by the current user
		mindmaps, err := c.Data.MindmapManager.MindmapList()
		if err != nil {
			return fmt.Errorf("failed to get mindmaps: %v", err)
		}

		var clearedCount = 0
		for _, mm := range mindmaps {
			err := c.Data.MindmapManager.MindmapDelete(mm.Name)
			if err != nil {
				return fmt.Errorf("failed to delete mindmap '%s': %v", mm.Name, err)
			}
			clearedCount++
		}

		c.UI.Success(fmt.Sprintf("%d mind map(s) deleted.", clearedCount))
	} else {
		// Delete a specific mindmap
		mindmapName := args[0]
		err := c.Data.MindmapManager.MindmapDelete(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to delete mindmap '%s': %v", mindmapName, err)
		}

		// If the deleted mindmap was the current one, update CLI's CurrentMindmap
		if c.CurrentMindmap != nil && c.CurrentMindmap.Name == mindmapName {
			c.CurrentMindmap = nil
		}

		c.UI.Success(fmt.Sprintf("Mind map '%s' deleted.", mindmapName))
	}

	return nil
}

// MindmapPermission handles the 'mindmap permission' command, changing a mindmap's visibility.
func (c *CLI) MindmapPermission(args []string) error {
	// Check for correct usage
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap permission <mindmap name> [public|private]")
	}

	mindmapName := args[0]

	// Check permission if only mindmap name is provided
	if len(args) == 1 {
		hasPermission, err := c.Data.MindmapManager.MindmapPermission(mindmapName, nil)
		if err != nil {
			return fmt.Errorf("failed to check mindmap permission: %v", err)
		}

		// Set permission if both mindmap name and permission are provided
		if hasPermission {
			c.UI.Message(fmt.Sprintf("You have permission to access mindmap '%s'", mindmapName))
		} else {
			c.UI.Message(fmt.Sprintf("You don't have permission to access mindmap '%s'", mindmapName))
		}
		return nil
	}

	// Set permission
	permission := args[1]
	var isPublic bool
	switch permission {
	case "public":
		isPublic = true
	case "private":
		isPublic = false
	default:
		return fmt.Errorf("invalid permission option: use 'public' or 'private'")
	}

	success, err := c.Data.MindmapManager.MindmapPermission(mindmapName, &isPublic)
	if err != nil {
		return fmt.Errorf("failed to update mindmap permission: %v", err)
	}

	if success {
		c.UI.Success(fmt.Sprintf("Mindmap '%s' permission set to %s", mindmapName, permission))
	} else {
		c.UI.Warning(fmt.Sprintf("Failed to set mindmap '%s' permission to %s", mindmapName, permission))
	}
	return err
}

// MindmapImport handles the 'mindmap import' command, importing a mindmap from a file.
func (c *CLI) MindmapImport(args []string) error {
	// Check for correct usage
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap import <filename> [json|xml]")
	}

	// Get filename and format
	filename := args[0]
	format := "json"
	if len(args) == 2 {
		format = args[1]
	}

	// Import the mindmap
	updatedMindmap, err := c.Data.MindmapManager.MindmapImport(filename, format)
	if err != nil {
		return fmt.Errorf("failed to load mindmap: %v", err)
	}

	// Update the CLI's current mindmap state
	c.CurrentMindmap = updatedMindmap

	c.UI.Success(fmt.Sprintf("Mind map loaded from %s", filename))
	return nil
}

// MindmapExport handles the 'mindmap export' command, exporting a mindmap to a file.
func (c *CLI) MindmapExport(args []string) error {
	// Check for correct usage
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap export <filename> [json|xml]")
	}

	// Get filename and format
	filename := args[0]
	format := "json"
	if len(args) == 2 {
		format = args[1]
	}

	// Export the mindmap
	err := c.Data.MindmapManager.MindmapExport(filename, format)
	if err != nil {
		return fmt.Errorf("failed to save mindmap: %v", err)
	}

	c.UI.Success(fmt.Sprintf("Mind map saved to %s in %s format", filename, format))
	return nil
}

// MindmapSelect handles the 'mindmap select' command, selecting a mindmap as current mindmap.
func (c *CLI) MindmapSelect(args []string) error {
	// If no arguments, deselect current mindmap
	if len(args) == 0 {
		err := c.Data.MindmapManager.MindmapSelect("")
		if err != nil {
			return err
		}
		c.CurrentMindmap = nil // Ensure this is set to nil
		c.UI.Success("Deselected the current mindmap.")
		return nil
	}

	// Select the specified mindmap
	name := args[0]
	err := c.Data.MindmapManager.MindmapSelect(name)
	if err != nil {
		return err
	}

	c.CurrentMindmap = c.Data.MindmapManager.MindmapGet()
	c.UI.Success(fmt.Sprintf("Selected mindmap '%s'", name))
	return nil
}

// MindmapList handles the 'mindmap list' command, displaying all available mindmaps accessible to the current user.
func (c *CLI) MindmapList(args []string) error {
	// Get the list of mindmaps
	mindmaps, err := c.Data.MindmapManager.MindmapList()
	if err != nil {
		return fmt.Errorf("failed to retrieve mindmaps: %v", err)
	}

	// Display the list of mindmaps
	c.UI.MindmapUI.MindmapList(mindmaps, c.CurrentUser.Username)

	return nil
}

// MindmapView handles the 'mindmap view' command, displaying the structure of a mindmap.
func (c *CLI) MindmapView(args []string) error {
	// Check if ID should be shown
	showIndex := false

	for _, arg := range args {
		if arg == "--id" || arg == "-i" {
			showIndex = true
		}
	}

	// Get the current mindmap
	mindmap := c.Data.MindmapManager.MindmapGet()
	if mindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	// Get all nodes of the current mindmap
	nodes, err := c.Data.NodeManager.NodeGetAll()
	if err != nil {
		return fmt.Errorf("failed to get mindmap nodes: %v", err)
	}

	// Use the MindmapUI to visualize the mindmap
	c.UI.MindmapUI.MindmapView(nodes, showIndex)

	return nil
}

// MindmapConnect handles the 'mindmap connect' command (placeholder for future implementation).
func (c *CLI) MindmapConnect(args []string) error {
	c.UI.Info("Mindmap connection functionality is not implemented yet.")
	return nil
}

// ExecuteMindmapCommand routes the mindmap command to the appropriate handler.
func (c *CLI) ExecuteMindmapCommand(args []string) error {
	// If no arguments, show mindmap info
	if len(args) == 0 {
		return c.MindmapInfo(args)
	}

	// Route to specific mindmap command handlers
	operation := args[0]
	switch operation {
	case "add":
		return c.MindmapAdd(args[1:])
	case "update":
		return c.MindmapUpdate(args[1:])
	case "delete":
		return c.MindmapDelete(args[1:])
	case "permission":
		return c.MindmapPermission(args[1:])
	case "import":
		return c.MindmapImport(args[1:])
	case "export":
		return c.MindmapExport(args[1:])
	case "select":
		return c.MindmapSelect(args[1:])
	case "list":
		return c.MindmapList(args[1:])
	case "view":
		return c.MindmapView(args[1:])
	case "connect":
		return c.MindmapConnect(args[1:])
	default:
		return fmt.Errorf("unknown mindmap operation: %s", operation)
	}
}
