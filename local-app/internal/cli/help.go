// Package cli provides the command-line interface functionality for Mindnoscape.
// This file specifically handles the help system, providing detailed information
// about commands and their usage to users.
package cli

import "fmt"

// CommandHelp represents the structure of help information for a specific command.
type CommandHelp struct {
	Scope     string
	Operation string
	ShortDesc string
	LongDesc  string
	Syntax    string
	Arguments []string
	Options   []string
	Examples  []string
}

// HandleHelp processes the help command and displays appropriate help information.
// It can show general help, scope-specific help, or operation-specific help.
func (c *CLI) HandleHelp(args []string) error {
	switch len(args) {
	case 0:
		return c.showGeneralHelp()
	case 1:
		return c.showScopeHelp(args[0])
	case 2:
		return c.showOperationHelp(args[0], args[1])
	default:
		return fmt.Errorf("invalid help command. Use 'help [scope] [operation]'")
	}
}

// showGeneralHelp displays an overview of all available commands grouped by scope.
func (c *CLI) showGeneralHelp() error {
	// Print the general command syntax
	c.UI.Message("Command syntax: <scope> [operation] [arguments] [options]")
	c.UI.Message("\nAvailable commands:")

	// Iterate through command helps and print them grouped by scope
	currentScope := ""
	for _, cmd := range commandHelps {
		if cmd.Scope != currentScope {
			c.UI.Message("\n%s:\n", cmd.Scope)
			currentScope = cmd.Scope
		}
		c.UI.Message("  %-15s %s\n", cmd.Operation, cmd.ShortDesc)
	}
	return nil
}

// showScopeHelp displays help information for all commands within a specific scope.
func (c *CLI) showScopeHelp(scope string) error {
	// Print the header for the scope
	c.UI.Message("Commands for %s:\n\n", scope)

	// Iterate through commands and print those matching the scope
	for _, cmd := range commandHelps {
		if cmd.Scope == scope {
			c.UI.Message("%-15s %s\n", cmd.Operation, cmd.ShortDesc)
		}
	}
	return nil
}

// showOperationHelp displays detailed help information for a specific operation within a scope.
func (c *CLI) showOperationHelp(scope, operation string) error {
	// Search for the matching command help
	for _, cmd := range commandHelps {
		if cmd.Scope == scope && cmd.Operation == operation {
			// Print detailed information about the command
			c.UI.Message("Command: %s %s\n", scope, operation)
			c.UI.Message("Description: %s\n", cmd.LongDesc)
			c.UI.Message("Syntax: %s\n", cmd.Syntax)
			if len(cmd.Arguments) > 0 {
				c.UI.Message("Arguments:")
				for _, arg := range cmd.Arguments {
					c.UI.Message("  %s\n", arg)
				}
			}
			if len(cmd.Options) > 0 {
				c.UI.Message("Options:")
				for _, opt := range cmd.Options {
					c.UI.Message("  %s\n", opt)
				}
			}
			if len(cmd.Examples) > 0 {
				c.UI.Message("Examples:")
				for _, ex := range cmd.Examples {
					c.UI.Message("  %s\n", ex)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("no help found for %s %s", scope, operation)
}

// commandHelps is a slice of CommandHelp structs containing help information for all commands.
var commandHelps = []CommandHelp{
	{
		Scope:     "user",
		Operation: "add",
		ShortDesc: "Add a new user",
		LongDesc:  "Creates a new user account with the specified username and password.",
		Syntax:    "user add <username> [password]",
		Arguments: []string{"username: The name of the new user", "password: (Optional) The password for the new user"},
		Examples:  []string{"user add john", "user add jane secret_password"},
	},
	{
		Scope:     "user",
		Operation: "update",
		ShortDesc: "Update an existing user",
		LongDesc:  "Updates the username or password of an existing user account.",
		Syntax:    "user update <username> [new_username] [new_password]",
		Arguments: []string{"username: The name of the user to update", "new_username: (Optional) The new username", "new_password: (Optional) The new password"},
		Examples:  []string{"user update john", "user update john johnny", "user update john johnny new_password"},
	},
	{
		Scope:     "user",
		Operation: "delete",
		ShortDesc: "Delete a user",
		LongDesc:  "Deletes an existing user account and all associated mindmaps.",
		Syntax:    "user delete <username>",
		Arguments: []string{"username: The name of the user to delete"},
		Examples:  []string{"user delete john"},
	},
	{
		Scope:     "user",
		Operation: "select",
		ShortDesc: "Select a user",
		LongDesc:  "Selects the specified user account.",
		Syntax:    "user select <username>",
		Arguments: []string{"username: The name of the user to select"},
		Examples:  []string{"user select john"},
	},
	{
		Scope:     "mindmap",
		Operation: "add",
		ShortDesc: "Create a new mindmap",
		LongDesc:  "Creates a new mindmap with the specified name.",
		Syntax:    "mindmap add <mindmap_name>",
		Arguments: []string{"mindmap_name: The name of the new mindmap"},
		Examples:  []string{"mindmap add my_ideas"},
	},
	{
		Scope:     "mindmap",
		Operation: "delete",
		ShortDesc: "Delete a mindmap",
		LongDesc:  "Deletes the specified mindmap or all mindmaps owned by the current user if no name is provided.",
		Syntax:    "mindmap delete [mindmap_name]",
		Arguments: []string{"mindmap_name: (Optional) The name of the mindmap to delete"},
		Examples:  []string{"mindmap delete", "mindmap delete my_ideas"},
	},
	{
		Scope:     "mindmap",
		Operation: "permission",
		ShortDesc: "Modify mindmap permission",
		LongDesc:  "Changes the permission of a mindmap to public or private.",
		Syntax:    "mindmap permission <mindmap_name> <public|private>",
		Arguments: []string{"mindmap_name: The name of the mindmap", "permission: 'public' or 'private'"},
		Examples:  []string{"mindmap permission my_ideas public", "mindmap permission project_x private"},
	},
	{
		Scope:     "mindmap",
		Operation: "import",
		ShortDesc: "Import a mindmap from a file",
		LongDesc:  "Imports a mindmap from a file in JSON or XML format.",
		Syntax:    "mindmap import <filename> [json|xml]",
		Arguments: []string{"filename: The name of the file to import from", "format: (Optional) The file format, either 'json' or 'xml'. Defaults to 'json'"},
		Examples:  []string{"mindmap import my_ideas.json", "mindmap import project_x.xml xml"},
	},
	{
		Scope:     "mindmap",
		Operation: "export",
		ShortDesc: "Export a mindmap to a file",
		LongDesc:  "Exports the current mindmap to a file in JSON or XML format.",
		Syntax:    "mindmap export <filename> [json|xml]",
		Arguments: []string{"filename: The name of the file to save to", "format: (Optional) The file format, either 'json' or 'xml'. Defaults to 'json'"},
		Examples:  []string{"mindmap export my_ideas.json", "mindmap export project_x.xml xml"},
	},
	{
		Scope:     "mindmap",
		Operation: "select",
		ShortDesc: "Select a mindmap",
		LongDesc:  "Selects the specified mindmap or deselects the current mindmap if no name is provided.",
		Syntax:    "mindmap select [mindmap_name]",
		Arguments: []string{"mindmap_name: (Optional) The name of the mindmap to select"},
		Examples:  []string{"mindmap select", "mindmap select my_ideas"},
	},
	{
		Scope:     "mindmap",
		Operation: "list",
		ShortDesc: "List available mindmaps",
		LongDesc:  "Displays a list of all mindmaps accessible to the current user.",
		Syntax:    "mindmap list",
		Examples:  []string{"mindmap list"},
	},
	{
		Scope:     "mindmap",
		Operation: "view",
		ShortDesc: "View mindmap structure",
		LongDesc:  "Displays the structure of the current mindmap or a specific node.",
		Syntax:    "mindmap view [index] [--id]",
		Arguments: []string{"index: (Optional) The index of the node to view", "--id: (Optional) Show node id"},
		Examples:  []string{"mindmap view", "mindmap view 1.2", "mindmap view --id"},
	},
	{
		Scope:     "node",
		Operation: "add",
		ShortDesc: "Add a new node",
		LongDesc:  "Adds a new node to the current mindmap.",
		Syntax:    "node add <parent> <content> [<extra field label>:<extra field value>]... [--id]",
		Arguments: []string{"parent: The parent node identifier", "content: The content of the new node", "extra: (Optional) Extra fields in the format label:value", "--id: (Optional) Use id instead of index"},
		Examples:  []string{"node add 1 \"New idea\"", "node add 2.1 \"Sub-idea\" priority:high --id"},
	},
	{
		Scope:     "node",
		Operation: "update",
		ShortDesc: "Update a node",
		LongDesc:  "Updates the content or extra fields of an existing node.",
		Syntax:    "node update <node> <content> [<extra field label>:<extra field value>]... [--id]",
		Arguments: []string{"node: The node identifier to modify", "content: The new content for the node", "extra: (Optional) Extra fields to modify in the format label:value", "--id: (Optional) Use id instead of index"},
		Examples:  []string{"node update 1.1 \"Updated idea\"", "node update 2 \"Changed content\" priority:low --id"},
	},
	{
		Scope:     "node",
		Operation: "move",
		ShortDesc: "Move a node",
		LongDesc:  "Moves a node to a new parent in the current mindmap.",
		Syntax:    "node move <source> <target> [--id]",
		Arguments: []string{"source: The identifier of the node to move", "target: The identifier of the new parent node", "--id: (Optional) Use id instead of index"},
		Examples:  []string{"node move 1.2 2.1", "node move 3 1 --id"},
	},
	{
		Scope:     "node",
		Operation: "delete",
		ShortDesc: "Delete a node",
		LongDesc:  "Deletes a node and its subtree from the current mindmap.",
		Syntax:    "node delete <node> [--id]",
		Arguments: []string{"node: The identifier of the node to delete", "--id: (Optional) Use id instead of index"},
		Examples:  []string{"node delete 1.2", "node delete 3 --id"},
	},
	{
		Scope:     "node",
		Operation: "find",
		ShortDesc: "Find nodes",
		LongDesc:  "Searches for nodes in the current mindmap based on a query string.",
		Syntax:    "node find <query> [--id]",
		Arguments: []string{"query: The search query string", "--id: (Optional) Show node indices in the results"},
		Examples:  []string{"node find \"important idea\"", "node find project --id"},
	},
	{
		Scope:     "node",
		Operation: "sort",
		ShortDesc: "Sort child nodes",
		LongDesc:  "Sorts the child nodes of a specified node based on content or an extra field.",
		Syntax:    "node sort [identifier] [field] [--reverse] [--id]",
		Arguments: []string{"identifier: (Optional) The node whose children to sort. Defaults to root", "field: (Optional) The field to sort by. Defaults to node content", "--reverse: (Optional) Sort in descending order", "--id: (Optional) Use id instead of index"},
		Examples:  []string{"node sort", "node sort 1.2 priority --reverse", "node sort 2 --id"},
	},
	{
		Scope:     "node",
		Operation: "undo",
		ShortDesc: "Undo the last node operation",
		LongDesc:  "Undoes the last node operation performed in the current mindmap.",
		Syntax:    "node undo",
		Examples:  []string{"node undo"},
	},
	{
		Scope:     "node",
		Operation: "redo",
		ShortDesc: "Redo the last undone node operation",
		LongDesc:  "Redoes the last node operation that was undone in the current mindmap.",
		Syntax:    "node redo",
		Examples:  []string{"node redo"},
	},
	{
		Scope:     "system",
		Operation: "exit",
		ShortDesc: "Exit the program",
		LongDesc:  "Exits the Mindnoscape program, saving all changes.",
		Syntax:    "system exit",
		Examples:  []string{"system exit"},
	},
	{
		Scope:     "system",
		Operation: "quit",
		ShortDesc: "Quit the program",
		LongDesc:  "Quits the Mindnoscape program, saving all changes. Equivalent to 'system exit'.",
		Syntax:    "system quit",
		Examples:  []string{"system quit"},
	},
}
