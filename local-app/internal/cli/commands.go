package cli

import (
	"fmt"
	"io"
	"mindnoscape/local-app/internal/mindmap"
	"mindnoscape/local-app/internal/models"
	"sort"
	"strings"

	"mindnoscape/local-app/internal/storage"
)

func (c *CLI) handleNew(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: new <mindmap name>")
	}

	name := args[0]
	err := c.MindMap.CreateNewMindMap(name)
	if err != nil {
		return err
	}

	fmt.Printf("New mindmap '%s' created and switched to\n", name)

	// Update the prompt
	c.Prompt = fmt.Sprintf("%s > ", name)
	c.RL.SetPrompt(c.Prompt)

	return nil
}

func (c *CLI) handleSwitch(args []string) error {
	// First, check if there are any mindmaps available
	mindmaps := c.MindMap.ListMindMaps()
	if len(mindmaps) == 0 {
		fmt.Println("No mindmaps available, use 'new' to create a new mindmap or 'load' to load one from a file")
		return nil
	}

	if len(args) == 0 {
		// Check if we're currently in a mindmap
		if c.MindMap.CurrentMindMap == nil {
			fmt.Println("Not currently in any mindmap, use 'switch <mindmap name>' to switch to a mindmap")
			return nil
		}
		// Switch out of the current mindmap
		c.MindMap.CurrentMindMap = nil
		c.Prompt = "> "
		fmt.Println("Switched out of the current mindmap")
		return nil
	}

	name := args[0]
	err := c.MindMap.SwitchMindMap(name)
	if err != nil {
		return err
	}

	c.updatePrompt()
	fmt.Printf("Switched to mindmap '%s'.\n", name)
	return nil
}

func (c *CLI) handleList(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: list")
	}

	mindmaps := c.MindMap.ListMindMaps()

	if len(mindmaps) == 0 {
		fmt.Println("No mindmaps available")
	} else {
		fmt.Println("Available mindmaps:")
		for i, name := range mindmaps {
			fmt.Printf("%d. %s\n", i+1, name)
		}
	}

	return nil
}

func (c *CLI) handleAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: add <parent> <content> [<extra field label>:<extra field value>]... [--index]")
	}

	parentIdentifier := args[0]
	content := args[1]
	extra := make(map[string]string)
	useIndex := false

	// Process extra fields and check for --index flag
	for _, arg := range args[2:] {
		if arg == "--index" {
			useIndex = true
		} else {
			parts := strings.SplitN(arg, ":", 2)
			if len(parts) == 2 {
				extra[parts[0]] = parts[1]
			}
		}
	}

	err := c.MindMap.AddNode(parentIdentifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node added successfully")
	return nil
}

func (c *CLI) handleDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: del <node> [--index]")
	}

	identifier := args[0]
	useIndex := false

	// Check for --index flag
	if len(args) > 1 && args[1] == "--index" {
		useIndex = true
	}

	err := c.MindMap.DeleteNode(identifier, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node deleted successfully")
	return nil
}

func (c *CLI) handleClear(args []string) error {
	if len(args) == 0 {
		// No argument given
		if c.MindMap.CurrentMindMap != nil {
			// Clear current mindmap and switch out
			mindmapName := c.MindMap.CurrentMindMap.Root.Content
			err := c.MindMap.Clear()
			if err != nil {
				return fmt.Errorf("failed to clear mindmap '%s': %v", mindmapName, err)
			}
			c.Prompt = "> "
			fmt.Printf("Mind map '%s' cleared and removed. Switched out of the mindmap\n", mindmapName)
		} else {
			// Clear all mindmaps
			mindmaps := c.MindMap.ListMindMaps()
			for _, name := range mindmaps {
				err := c.MindMap.Store.ClearAllNodes(name)
				if err != nil {
					return fmt.Errorf("failed to clear mindmap '%s': %v", name, err)
				}
				delete(c.MindMap.MindMaps, name)
			}
			fmt.Println("All mind maps cleared")
		}
	} else {
		// Mindmap name given as argument
		mindmapName := args[0]
		exists, err := c.MindMap.Store.MindMapExists(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to check if mindmap '%s' exists: %v", mindmapName, err)
		}
		if !exists {
			return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
		}

		if c.MindMap.CurrentMindMap != nil && c.MindMap.CurrentMindMap.Root.Content == mindmapName {
			// Clear current mindmap and switch out
			err := c.MindMap.Clear()
			if err != nil {
				return fmt.Errorf("failed to clear mindmap '%s': %v", mindmapName, err)
			}
			c.Prompt = "> "
			fmt.Printf("Mind map '%s' cleared and removed\nSwitched out of the mindmap\n", mindmapName)
		} else {
			// Clear specified mindmap without switching
			err := c.MindMap.Store.ClearAllNodes(mindmapName)
			if err != nil {
				return fmt.Errorf("failed to clear mindmap '%s': %v", mindmapName, err)
			}
			delete(c.MindMap.MindMaps, mindmapName)
			fmt.Printf("Mind map '%s' cleared and removed\n", mindmapName)
		}
	}

	return nil
}

func (c *CLI) handleModify(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mod <node> <content> [<extra field label>:<extra field value>]... [--index]")
	}

	identifier := args[0]
	content := args[1]
	extra := make(map[string]string)
	useIndex := false

	// Process extra fields and check for --index flag
	for i := 2; i < len(args); i++ {
		if args[i] == "--index" {
			useIndex = true
		} else {
			parts := strings.SplitN(args[i], ":", 2)
			if len(parts) == 2 {
				extra[parts[0]] = parts[1]
			}
		}
	}

	err := c.MindMap.ModifyNode(identifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node modified successfully")
	return nil
}

func (c *CLI) handleMove(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: move <source> <target> [--index]")
	}

	sourceIdentifier := args[0]
	targetIdentifier := args[1]
	useIndex := false

	// Check for --index flag
	if len(args) > 2 && args[2] == "--index" {
		useIndex = true
	}

	err := c.MindMap.MoveNode(sourceIdentifier, targetIdentifier, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node moved successfully")
	return nil
}

func (c *CLI) handleInsert(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: insert <source> <target> [--index]")
	}

	sourceIdentifier := args[0]
	targetIdentifier := args[1]
	useIndex := false

	// Check for --index flag
	if len(args) > 2 && args[2] == "--index" {
		useIndex = true
	}

	err := c.MindMap.InsertNode(sourceIdentifier, targetIdentifier, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node inserted successfully")
	return nil
}

func (c *CLI) handleShow(args []string) error {
	logicalIndex := ""
	showIndex := false

	for _, arg := range args {
		if arg == "--index" {
			showIndex = true
		} else {
			logicalIndex = arg
		}
	}

	return c.MindMap.Show(logicalIndex, showIndex)
}

func (c *CLI) handleSort(args []string) error {
	identifier := ""
	field := ""
	reverse := false
	useIndex := false
	for i := 0; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		switch arg {
		case "--reverse":
			reverse = true
		case "--index":
			useIndex = true
		default:
			if identifier == "" {
				identifier = args[i]
			} else if field == "" {
				field = args[i]
			}
		}
	}
	err := c.MindMap.Sort(identifier, field, reverse, useIndex)
	if err != nil {
		return err
	}
	fmt.Println("Sorted successfully")
	return nil
}

func (c *CLI) handleSave(args []string) error {
	filename := "mindmap.json"
	format := "json"

	if len(args) >= 1 {
		filename = args[0]
	}
	if len(args) >= 2 {
		format = args[1]
	}

	if c.MindMap.CurrentMindMap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	err := storage.SaveToFile(c.MindMap.Store, c.MindMap.CurrentMindMap.Root.Content, filename, format)
	if err != nil {
		return err
	}

	fmt.Printf("Mind map saved to %s in %s format\n", filename, format)
	return nil
}

func (c *CLI) handleLoad(args []string) error {
	filename := "mindmap.json"
	format := "json"

	if len(args) >= 1 {
		filename = args[0]
	}
	if len(args) >= 2 {
		format = args[1]
	}

	// Import the file into a temporary root node
	tempRoot, err := storage.ImportFromFile(filename, format)
	if err != nil {
		return err
	}

	mindmapName := tempRoot.Content
	exists, err := c.MindMap.Store.MindMapExists(mindmapName)
	if err != nil {
		return fmt.Errorf("failed to check if mindmap exists: %v", err)
	}

	isCurrentMindmap := c.MindMap.CurrentMindMap != nil && c.MindMap.CurrentMindMap.Root.Content == mindmapName

	// If we're currently in the mindmap we're loading, switch out first
	if isCurrentMindmap {
		c.MindMap.CurrentMindMap = nil
		c.Prompt = "> "
		fmt.Printf("Switched out of mindmap '%s' before reloading\n", mindmapName)
	}

	if exists {
		fmt.Printf("Replacing content of existing mindmap '%s'\n", mindmapName)
		err = c.MindMap.Store.ClearAllNodes(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to clear existing nodes for mindmap '%s': %v", mindmapName, err)
		}
		// Remove from in-memory map as well
		delete(c.MindMap.MindMaps, mindmapName)
	} else {
		fmt.Printf("Creating new mindmap '%s' from file\n", mindmapName)
	}

	// Load the content into the mindmap
	err = storage.LoadFromFile(c.MindMap.Store, mindmapName, filename, format)
	if err != nil {
		return fmt.Errorf("failed to load nodes for mindmap '%s': %v", mindmapName, err)
	}

	// Create a new MindMap struct
	newMindMap := &mindmap.MindMap{
		Nodes: make(map[int]*models.Node),
	}
	c.MindMap.MindMaps[mindmapName] = newMindMap

	// Reload the mind map from storage
	err = c.MindMap.LoadNodes(mindmapName)
	if err != nil {
		return fmt.Errorf("failed to reload mind map after load: %v", err)
	}

	fmt.Printf("Mind map '%s' loaded from %s\n", mindmapName, filename)

	// If we were in the mindmap before, switch back to it
	if isCurrentMindmap {
		err = c.MindMap.SwitchMindMap(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to switch back to mindmap '%s': %v", mindmapName, err)
		}
		c.Prompt = fmt.Sprintf("%s > ", mindmapName)
		fmt.Printf("Switched back to mindmap '%s'\n", mindmapName)
	}

	return nil
}

func (c *CLI) handleFind(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: find <query> [--index]")
	}

	query := args[0]
	showIndex := false

	// Check for --index flag
	if len(args) > 1 && args[1] == "--index" {
		showIndex = true
	}

	// If the query is enclosed in quotes, remove them
	if strings.HasPrefix(query, "\"") && strings.HasSuffix(query, "\"") {
		query = query[1 : len(query)-1]
	}

	if c.MindMap.CurrentMindMap == nil {
		return fmt.Errorf("no mindmap selected, use 'switch' command to select a mindmap")
	}

	matches := c.MindMap.FindNodes(query)

	if len(matches) == 0 {
		fmt.Println("No matches found.")
		return nil
	}

	fmt.Printf("Found %d matches:\n", len(matches))
	for _, node := range matches {
		c.printNode(node, showIndex)
	}

	return nil
}

func (c *CLI) printNode(node *models.Node, showIndex bool) {
	fmt.Printf("LogicalIndex: %s, Content: %s", node.LogicalIndex, node.Content)
	if showIndex {
		fmt.Printf(" [%d]", node.Index)
	}

	// Add extra fields
	if len(node.Extra) > 0 {
		var extraFields []string
		for k, v := range node.Extra {
			extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
		}
		sort.Strings(extraFields) // Sort extra fields for consistent output
		fmt.Printf(" %s", strings.Join(extraFields, " "))
	}

	fmt.Println() // End the line
}

func (c *CLI) handleHelp(args []string) error {
	if len(args) > 0 {
		c.printHelp(args[0])
	} else {
		c.printHelp("")
	}
	return nil
}

func (c *CLI) handleExit() error {
	fmt.Println("Exiting...")
	err := c.RL.Close()
	if err != nil {
		fmt.Printf("Error closing readline: %v\n", err)
	}
	return fmt.Errorf("exit requested: %w", io.EOF)
}
