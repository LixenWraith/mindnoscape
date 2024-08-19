// Package cli provides the command-line interface functionality for Mindnoscape.
// This file contains handlers for node-related commands.
package cli

import (
	"fmt"
	"strings"
)

// NodeInfo displays information about the root node of the current mindmap. Placeholder for future implementation.
func (c *CLI) NodeInfo(args []string) error {
	return nil
}

// NodeAdd handles the 'node add' command, adding a new node to the mindmap.
func (c *CLI) NodeAdd(args []string) error {
	// Check for correct usage
	if len(args) < 2 {
		return fmt.Errorf("usage: node add <parent> <content> [<extra field label>:<extra field value>]... [--id]")
	}

	// Parse arguments
	parentIdentifier := args[0]
	content := args[1]
	extra := make(map[string]string)
	useID := false

	for _, arg := range args[2:] {
		if arg == "--id" || arg == "-i" {
			useID = true
		} else {
			parts := strings.SplitN(arg, ":", 2)
			if len(parts) == 2 {
				extra[parts[0]] = parts[1]
			}
		}
	}

	// Add the new node
	// Add the new node
	copies, err := c.Data.NodeManager.NodeAdd(parentIdentifier, content, extra, useID, false)
	if err != nil {
		return err
	}

	if copies > 1 {
		c.UI.Warning(fmt.Sprintf("There are now %d nodes with the content '%s'", copies, content))
	}

	c.UI.Success("Node added successfully.")
	return nil
}

// NodeUpdate handles the 'node update' command, modifying an existing node.
func (c *CLI) NodeUpdate(args []string) error {
	// Check for correct usage
	if len(args) < 2 {
		return fmt.Errorf("usage: node update <node> <content> [<extra field label>:<extra field value>]... [-i]")
	}

	// Parse arguments
	identifier := args[0]
	content := args[1]
	extra := make(map[string]string)
	useID := false

	for i := 2; i < len(args); i++ {
		if args[i] == "-i" {
			useID = true
		} else {
			parts := strings.SplitN(args[i], ":", 2)
			if len(parts) == 2 {
				extra[parts[0]] = parts[1]
			}
		}
	}

	// Update the node
	err := c.Data.NodeManager.NodeUpdate(identifier, content, extra, useID, false)
	if err != nil {
		return err
	}

	c.UI.Success("Node modified successfully.")
	return nil
}

// NodeMove handles the 'node move' command, moving a node to a new parent.
func (c *CLI) NodeMove(args []string) error {
	// Check for correct usage
	if len(args) < 2 {
		return fmt.Errorf("usage: node move <source> <target> [--id]")
	}

	// Parse arguments
	sourceIdentifier := args[0]
	targetIdentifier := args[1]
	useID := false

	if len(args) > 2 && (args[2] == "--id" || args[2] == "-i") {
		useID = true
	}

	// Move the node
	err := c.Data.NodeManager.NodeMove(sourceIdentifier, targetIdentifier, useID, false)
	if err != nil {
		return err
	}

	c.UI.Success("Node moved successfully.")
	return nil
}

// NodeDelete handles the 'node delete' command, deleting a node from the mindmap.
func (c *CLI) NodeDelete(args []string) error {
	// Check for correct usage
	if len(args) < 1 {
		return fmt.Errorf("usage: node delete <node> [-index]")
	}

	// Parse arguments
	identifier := args[0]
	useID := false

	if len(args) > 1 && (args[1] == "--id" || args[1] == "-i") {
		useID = true
	}

	// Delete the node
	err := c.Data.NodeManager.NodeDelete(identifier, useID, false)
	if err != nil {
		return err
	}

	c.UI.Success("Node deleted successfully.")
	return nil
}

// NodeFind handles the 'node find' command, searching for nodes in the mindmap with content or extra fields containing the queried string.
func (c *CLI) NodeFind(args []string) error {
	// Check for correct usage
	if len(args) < 1 {
		return fmt.Errorf("usage: node find <query> [--id]")
	}

	// Parse arguments
	query := args[0]
	showIndex := false

	if len(args) > 1 && (args[1] == "--id" || args[1] == "-i") {
		showIndex = true
	}

	// Removing queried string if enclosed by quotes
	if strings.HasPrefix(query, "\"") && strings.HasSuffix(query, "\"") {
		query = query[1 : len(query)-1]
	}

	// Find matching nodes
	matches, err := c.Data.NodeManager.NodeFind(query)
	if err != nil {
		return fmt.Errorf("failed to find nodes: %v", err)
	}

	// Display the results
	c.UI.NodeUI.NodeFind(matches, showIndex)

	return nil
}

// NodeView handles the 'node view' command (placeholder for future implementation).
func (c *CLI) NodeView(args []string) error {
	c.UI.Info("Node view functionality is not implemented yet.")
	return nil
}

// NodeList handles the 'node list' command (placeholder for future implementation).
func (c *CLI) NodeList(args []string) error {
	c.UI.Info("Node list functionality is not implemented yet.")
	return nil
}

// NodeSort handles the 'node sort' command, sorting the subtree under a specified node.
func (c *CLI) NodeSort(args []string) error {
	// Parse arguments
	identifier := ""
	field := ""
	reverse := false
	useID := false
	for i := 0; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		switch arg {
		case "--reverse", "-r":
			reverse = true
		case "--id", "-i":
			useID = true
		default:
			if identifier == "" {
				identifier = args[i]
			} else if field == "" {
				field = args[i]
			}
		}
	}

	// Sort the nodes
	err := c.Data.NodeManager.NodeSort(identifier, field, reverse, useID)
	if err != nil {
		return err
	}

	c.UI.Success("Sorted successfully.")
	return nil
}

// NodeConnect handles the 'node connect' command (placeholder for future implementation).
func (c *CLI) NodeConnect(args []string) error {
	c.UI.Info("Node connection functionality is not implemented yet.")
	return nil
}

// NodeUndo handles the 'node undo' command, undoing the last node operation.
func (c *CLI) NodeUndo(args []string) error {
	// Check for correct usage
	if len(args) != 0 {
		return fmt.Errorf("usage: node undo")
	}

	// Perform the undo operation
	err := c.Data.NodeManager.NodeUndo()
	if err != nil {
		return err
	}

	c.UI.Success("Undo successful.")
	return nil
}

// NodeRedo handles the 'node redo' command, redoing the last undone node operation.
func (c *CLI) NodeRedo(args []string) error {
	// Check for correct usage
	if len(args) != 0 {
		return fmt.Errorf("usage: node redo")
	}

	// Perform the redo operation
	err := c.Data.NodeManager.NodeRedo()
	if err != nil {
		return err
	}

	c.UI.Success("Redo successful.")
	return nil
}

// ExecuteNodeCommand routes the node command to the appropriate handler.
func (c *CLI) ExecuteNodeCommand(args []string) error {
	// If no arguments, show node info
	if len(args) == 0 {
		return c.NodeInfo(args)
	}

	// Route to specific node command handlers
	operation := args[0]
	switch operation {
	case "add":
		return c.NodeAdd(args[1:])
	case "update":
		return c.NodeUpdate(args[1:])
	case "move":
		return c.NodeMove(args[1:])
	case "delete":
		return c.NodeDelete(args[1:])
	case "find":
		return c.NodeFind(args[1:])
	case "view":
		return c.NodeView(args[1:])
	case "list":
		return c.NodeList(args[1:])
	case "sort":
		return c.NodeSort(args[1:])
	case "connect":
		return c.NodeConnect(args[1:])
	case "undo":
		return c.NodeUndo(args[1:])
	case "redo":
		return c.NodeRedo(args[1:])
	default:
		return fmt.Errorf("unknown node operation: %s", operation)
	}
}
