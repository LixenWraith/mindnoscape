package cli

import (
	"fmt"
	"mindnoscape/local-app/internal/ui"
	"strings"
)

func (c *CLI) SystemInfo(args []string) error {
	c.UI.Println("System Information:")
	c.UI.Printf("Current User: %s\n", c.CurrentUser)

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
	isPublic := c.CurrentUser == "guest"
	err := c.Data.MindmapManager.AddMindmap(name, isPublic)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("New mindmap '%s' created and switched to", name))
	return nil
}

// MindmapMod handles the 'mindmap mod' command (placeholder)
func (c *CLI) MindmapMod(args []string) error {
	c.UI.Info("Mindmap modification functionality is not implemented yet.")
	return nil
}

// MindmapDel handles the 'mindmap del' command
func (c *CLI) MindmapDel(args []string) error {
	if len(args) == 0 {
		// Clear all mindmaps owned by the current user
		mindmaps, err := c.Data.MindmapManager.ListMindmap()
		if err != nil {
			return fmt.Errorf("failed to get mindmaps: %v", err)
		}

		clearedCount := 0
		for _, mm := range mindmaps {
			if mm.Owner == c.CurrentUser {
				err := c.Data.MindmapManager.DeleteMindmap(mm.Name)
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
		err := c.Data.MindmapManager.DeleteMindmap(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to delete mindmap '%s': %v", mindmapName, err)
		}

		c.UI.Success(fmt.Sprintf("Mind map '%s' deleted", mindmapName))
	}

	c.UpdatePrompt()
	return nil
}

// MindmapAccess handles the 'mindmap access' command
func (c *CLI) MindmapAccess(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: mindmap access <mindmap name> <public|private>")
	}

	mindmapName := args[0]
	access := args[1]

	isPublic := false
	if access == "public" {
		isPublic = true
	} else if access != "private" {
		return fmt.Errorf("invalid access option: use 'public' or 'private'")
	}

	err := c.Data.MindmapManager.ModifyMindmapAccess(mindmapName, isPublic)
	if err != nil {
		return fmt.Errorf("failed to update mindmap access: %v", err)
	}

	c.UI.Success(fmt.Sprintf("Mindmap '%s' access set to %s", mindmapName, access))
	return nil
}

// MindmapLoad handles the 'mindmap load' command
func (c *CLI) MindmapLoad(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap load <filename> [json|xml]")
	}

	filename := args[0]
	format := "json"
	if len(args) == 2 {
		format = args[1]
	}

	err := c.Data.MindmapManager.LoadMindmap(filename, format)
	if err != nil {
		return fmt.Errorf("failed to load mindmap: %v", err)
	}

	c.UI.Success(fmt.Sprintf("Mind map loaded from %s", filename))
	return nil
}

// MindmapSave handles the 'mindmap save' command
func (c *CLI) MindmapSave(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: mindmap save <filename> [json|xml]")
	}

	filename := args[0]
	format := "json"
	if len(args) == 2 {
		format = args[1]
	}

	err := c.Data.MindmapManager.SaveMindmap(filename, format)
	if err != nil {
		return fmt.Errorf("failed to save mindmap: %v", err)
	}

	c.UI.Success(fmt.Sprintf("Mind map saved to %s in %s format", filename, format))
	return nil
}

// MindmapSelect handles the 'mindmap select' command
func (c *CLI) MindmapSelect(args []string) error {
	if len(args) == 0 {
		if c.Data.MindmapManager.CurrentMindmap == nil {
			c.UI.Info("Not currently in any mindmap, use 'mindmap select <mindmap name>' to select a mindmap")
			return nil
		}
		c.Data.MindmapManager.CurrentMindmap = nil
		c.UI.Info("Switched out of the current mindmap")
		return nil
	}

	name := args[0]
	err := c.Data.MindmapManager.ChangeMindmap(name)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("Switched to mindmap '%s'", name))
	c.UpdatePrompt()
	return nil
}

// MindmapList handles the 'mindmap list' command
func (c *CLI) MindmapList(args []string) error {
	mindmaps, err := c.Data.MindmapManager.ListMindmap()
	if err != nil {
		return fmt.Errorf("failed to retrieve mindmaps: %v", err)
	}

	if len(mindmaps) == 0 {
		c.UI.Println("No mindmaps available")
	} else {
		c.UI.Println("Available mindmaps:")
		for _, mm := range mindmaps {
			accessSymbol := "+"
			accessColor := ui.ColorGreen
			if !mm.IsPublic {
				accessSymbol = "-"
				accessColor = ui.ColorRed
			}
			c.UI.Print(mm.Name + " ")
			c.UI.PrintColored(accessSymbol, accessColor)
			if mm.Owner != c.CurrentUser {
				c.UI.Printf(" (owner: %s)", mm.Owner)
			}
			c.UI.Println("")
		}
	}

	return nil
}

// MindmapView handles the 'mindmap view' command
func (c *CLI) MindmapView(args []string) error {
	logicalIndex := ""
	showIndex := false

	for _, arg := range args {
		if arg == "--index" {
			showIndex = true
		} else {
			logicalIndex = arg
		}
	}

	output, err := c.Data.MindmapManager.ShowMindmap(logicalIndex, showIndex)
	if err != nil {
		return err
	}

	for _, line := range output {
		c.printColoredLine(line)
	}

	return nil
}

// To be moved to UI
func (c *CLI) printColoredLine(line string) {
	colorMap := map[string]ui.Color{
		"{{yellow}}":    ui.ColorYellow,
		"{{orange}}":    ui.ColorOrange,
		"{{darkbrown}}": ui.ColorDarkBrown,
		"{{default}}":   ui.ColorDefault,
	}

	for len(line) > 0 {
		startIndex := strings.Index(line, "{{")
		if startIndex == -1 {
			c.UI.Print(line)
			break
		}

		endIndex := strings.Index(line, "}}")
		if endIndex == -1 {
			c.UI.Print(line)
			break
		}

		// Print the part before the color code
		if startIndex > 0 {
			c.UI.Print(line[:startIndex])
		}

		colorCode := line[startIndex : endIndex+2]
		color, exists := colorMap[colorCode]
		if !exists {
			color = ui.ColorDefault
		}

		// Find the next color code or the end of the string
		nextStartIndex := strings.Index(line[endIndex+2:], "{{")
		if nextStartIndex == -1 {
			// No more color codes, print the rest of the line
			c.UI.PrintColored(line[endIndex+2:], color)
			break
		} else {
			// Print the part until the next color code
			c.UI.PrintColored(line[endIndex+2:endIndex+2+nextStartIndex], color)
			line = line[endIndex+2+nextStartIndex:]
		}
	}
	c.UI.Println("") // New line at the end
}

// MindmapLink handles the 'mindmap link' command (placeholder)
func (c *CLI) MindmapLink(args []string) error {
	c.UI.Info("Mindmap linking functionality is not implemented yet.")
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
	case "mod":
		return c.MindmapMod(args[1:])
	case "del":
		return c.MindmapDel(args[1:])
	case "access":
		return c.MindmapAccess(args[1:])
	case "load":
		return c.MindmapLoad(args[1:])
	case "save":
		return c.MindmapSave(args[1:])
	case "select":
		return c.MindmapSelect(args[1:])
	case "list":
		return c.MindmapList(args[1:])
	case "view":
		return c.MindmapView(args[1:])
	case "link":
		return c.MindmapLink(args[1:])
	default:
		return fmt.Errorf("unknown mindmap operation: %s", operation)
	}
}
