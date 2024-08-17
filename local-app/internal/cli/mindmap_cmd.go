package cli

import (
	"fmt"
)

func (c *CLI) MindmapInfo(args []string) error {
	if c.Data.MindmapManager.CurrentMindmap == nil {
		c.UI.Info("No mindmap is currently selected.")
		return nil
	}
	c.UI.Printf("Current mindmap: %s\n", c.Data.MindmapManager.CurrentMindmap.Name)
	return nil
}

// MindmapAdd handles the 'mindmap add' command
func (c *CLI) MindmapAdd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: mindmap add <mindmap name>")
	}

	name := args[0]
	var err = c.Data.MindmapManager.MindmapAdd(name, false) // Default to private mindmaps
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("New mindmap '%s' created", name))
	return nil
}

// MindmapUpdate handles the 'mindmap update' command (placeholder)
func (c *CLI) MindmapUpdate(args []string) error {
	c.UI.Info("Mindmap update functionality is not implemented yet.")
	return nil
}

// MindmapDelete handles the 'mindmap del' command
func (c *CLI) MindmapDelete(args []string) error {
	if len(args) == 0 {
		// Clear all mindmaps owned by the current user
		mindmaps, err := c.Data.MindmapManager.MindmapList()
		if err != nil {
			return fmt.Errorf("failed to get mindmaps: %v", err)
		}

		clearedCount := 0
		for _, mm := range mindmaps {
			if mm.Owner == c.CurrentUser {
				err := c.Data.MindmapManager.MindmapDelete(mm.Name)
				if err != nil {
					return fmt.Errorf("failed to delete mindmap '%s': %v", mm.Name, err)
				}
				clearedCount++
			}
		}

		c.UI.Success(fmt.Sprintf("%d mind map(s) deleted", clearedCount))
	} else {
		// Delete a specific mindmap
		mindmapName := args[0]
		err := c.Data.MindmapManager.MindmapDelete(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to delete mindmap '%s': %v", mindmapName, err)
		}

		c.UI.Success(fmt.Sprintf("Mind map '%s' deleted", mindmapName))
	}

	return nil
}

// MindmapPermission handles the 'mindmap permission' command
func (c *CLI) MindmapPermission(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap permission <mindmap name> [public|private]")
	}

	mindmapName := args[0]

	// Check permission
	if len(args) == 1 {
		hasPermission, err := c.Data.MindmapManager.MindmapPermission(mindmapName, nil)
		if err != nil {
			return fmt.Errorf("failed to check mindmap permission: %v", err)
		}

		if hasPermission {
			c.UI.Info(fmt.Sprintf("You have permission to access mindmap '%s'", mindmapName))
		} else {
			c.UI.Info(fmt.Sprintf("You don't have permission to access mindmap '%s'", mindmapName))
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

// MindmapImport handles the 'mindmap import' command
func (c *CLI) MindmapImport(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap import <filename> [json|xml]")
	}

	filename := args[0]
	format := "json"
	if len(args) == 2 {
		format = args[1]
	}

	err := c.Data.MindmapManager.MindmapImport(filename, format)
	if err != nil {
		return fmt.Errorf("failed to load mindmap: %v", err)
	}

	c.UI.Success(fmt.Sprintf("Mind map loaded from %s", filename))
	return nil
}

// MindmapExport handles the 'mindmap export' command
func (c *CLI) MindmapExport(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap export <filename> [json|xml]")
	}

	filename := args[0]
	format := "json"
	if len(args) == 2 {
		format = args[1]
	}

	err := c.Data.MindmapManager.MindmapExport(filename, format)
	if err != nil {
		return fmt.Errorf("failed to save mindmap: %v", err)
	}

	c.UI.Success(fmt.Sprintf("Mind map saved to %s in %s format", filename, format))
	return nil
}

// MindmapSelect handles the 'mindmap select' command
func (c *CLI) MindmapSelect(args []string) error {
	if len(args) == 0 {
		err := c.Data.MindmapManager.MindmapSelect("")
		if err != nil {
			return err
		}
		c.UI.Success("Deselected the current mindmap")
		return nil
	}

	name := args[0]
	err := c.Data.MindmapManager.MindmapSelect(name)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("Switched to mindmap '%s'", name))
	return nil
}

// MindmapList handles the 'mindmap list' command
func (c *CLI) MindmapList(args []string) error {
	mindmaps, err := c.Data.MindmapManager.MindmapList()
	if err != nil {
		return fmt.Errorf("failed to retrieve mindmaps: %v", err)
	}

	c.UI.MindmapUI.MindmapList(mindmaps, c.CurrentUser)

	return nil
}

// MindmapView handles the 'mindmap view' command
func (c *CLI) MindmapView(args []string) error {
	showIndex := false

	for _, arg := range args {
		if arg == "--id" || arg == "-i" {
			showIndex = true
		}
	}

	// Get the current mindmap
	mindmap := c.Data.MindmapManager.CurrentMindmap
	if mindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	// Get all nodes of the current mindmap
	nodes, err := c.Data.NodeManager.NodeGetAll()
	if err != nil {
		return fmt.Errorf("failed to get mindmap nodes: %v", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found for the current mindmap")
	}

	// Use the MindmapUI to visualize the mindmap
	c.UI.MindmapUI.MindmapView(nodes, showIndex)

	return nil
}

// MindmapConnect handles the 'mindmap connect' command (placeholder)
func (c *CLI) MindmapConnect(args []string) error {
	c.UI.Info("Mindmap connection functionality is not implemented yet.")
	return nil
}

// ExecuteMindmapCommand routes the mindmap command to the appropriate handler
func (c *CLI) ExecuteMindmapCommand(args []string) error {
	if len(args) == 0 {
		return c.MindmapInfo(args)
	}

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
