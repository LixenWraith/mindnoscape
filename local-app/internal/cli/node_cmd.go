package cli

import (
	"fmt"
	"strings"
)

func (c *CLI) NodeInfo(args []string) error {
	if c.Data.MindmapManager.CurrentMindmap == nil {
		c.UI.Info("No mindmap is currently selected.")
		return nil
	}
	root := c.Data.MindmapManager.CurrentMindmap.Root
	c.UI.Printf("Root node: %s\n", root.Content)
	if len(root.Extra) > 0 {
		c.UI.Println("Extra fields:")
		for key, value := range root.Extra {
			c.UI.Printf("  %s: %s\n", key, value)
		}
	}
	return nil
}

// NodeAdd handles the 'node add' command
func (c *CLI) NodeAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: node add <parent> <content> [<extra field label>:<extra field value>]... [--index]")
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

	err := c.Data.NodeManager.NodeAdd(parentIdentifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node added successfully")
	return nil
}

// NodeMod handles the 'node mod' command
func (c *CLI) NodeModify(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: node mod <node> <content> [<extra field label>:<extra field value>]... [--index]")
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

	err := c.Data.NodeManager.NodeModify(identifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node modified successfully")
	return nil
}

// NodeMove handles the 'node move' command
func (c *CLI) NodeMove(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: node move <source> <target> [--index]")
	}

	sourceIdentifier := args[0]
	targetIdentifier := args[1]
	useIndex := false

	// Check for --index flag
	if len(args) > 2 && args[2] == "--index" {
		useIndex = true
	}

	err := c.Data.NodeManager.NodeMove(sourceIdentifier, targetIdentifier, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node moved successfully")
	return nil
}

// NodeDel handles the 'node del' command
func (c *CLI) NodeDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: node del <node> [--index]")
	}

	identifier := args[0]
	useIndex := false

	// Check for --index flag
	if len(args) > 1 && args[1] == "--index" {
		useIndex = true
	}

	err := c.Data.NodeManager.NodeDelete(identifier, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node deleted successfully")
	return nil
}

// NodeFind handles the 'node find' command
func (c *CLI) NodeFind(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: node find <query> [--index]")
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

	output, err := c.Data.NodeManager.NodeFind(query, showIndex)
	if err != nil {
		return fmt.Errorf("failed to find nodes: %v", err)
	}

	for _, line := range output {
		c.UI.Println(line)
	}

	return nil
}

// NodeView handles the 'node view' command (placeholder)
func (c *CLI) NodeView(args []string) error {
	c.UI.Info("Node view functionality is not implemented yet.")
	return nil
}

// NodeList handles the 'node list' command (placeholder)
func (c *CLI) NodeList(args []string) error {
	c.UI.Info("Node list functionality is not implemented yet.")
	return nil
}

// NodeSort handles the 'node sort' command
func (c *CLI) NodeSort(args []string) error {
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
	err := c.Data.NodeManager.NodeSort(identifier, field, reverse, useIndex)
	if err != nil {
		return err
	}
	c.UI.Success("Sorted successfully")
	return nil
}

// NodeLink handles the 'node link' command (placeholder)
func (c *CLI) NodeConnect(args []string) error {
	c.UI.Info("Node connection functionality is not implemented yet.")
	return nil
}

// ExecuteNodeCommand routes the node command to the appropriate handler
func (c *CLI) ExecuteNodeCommand(args []string) error {
	if len(args) == 0 {
		return c.NodeInfo(args)
	}

	operation := args[0]
	switch operation {
	case "add":
		return c.NodeAdd(args[1:])
	case "mod":
		return c.NodeModify(args[1:])
	case "move":
		return c.NodeMove(args[1:])
	case "del":
		return c.NodeDelete(args[1:])
	case "find":
		return c.NodeFind(args[1:])
	case "view":
		return c.NodeView(args[1:])
	case "list":
		return c.NodeList(args[1:])
	case "sort":
		return c.NodeSort(args[1:])
	case "link":
		return c.NodeConnect(args[1:])
	default:
		return fmt.Errorf("unknown node operation: %s", operation)
	}
}
