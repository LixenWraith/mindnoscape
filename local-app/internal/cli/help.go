package cli

import "fmt"

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

func (c *CLI) showGeneralHelp() error {
	c.UI.Println("Command syntax: <scope> [operation] [arguments] [options]")
	c.UI.Println("\nAvailable commands:")

	currentScope := ""
	for _, cmd := range commandHelps {
		if cmd.Scope != currentScope {
			c.UI.Printf("\n%s:\n", cmd.Scope)
			currentScope = cmd.Scope
		}
		c.UI.Printf("  %-15s %s\n", cmd.Operation, cmd.ShortDesc)
	}
	return nil
}

func (c *CLI) showScopeHelp(scope string) error {
	c.UI.Printf("Commands for %s:\n\n", scope)
	for _, cmd := range commandHelps {
		if cmd.Scope == scope {
			c.UI.Printf("%-15s %s\n", cmd.Operation, cmd.ShortDesc)
		}
	}
	return nil
}

func (c *CLI) showOperationHelp(scope, operation string) error {
	for _, cmd := range commandHelps {
		if cmd.Scope == scope && cmd.Operation == operation {
			c.UI.Printf("Command: %s %s\n", scope, operation)
			c.UI.Printf("Description: %s\n", cmd.LongDesc)
			c.UI.Printf("Syntax: %s\n", cmd.Syntax)
			if len(cmd.Arguments) > 0 {
				c.UI.Println("Arguments:")
				for _, arg := range cmd.Arguments {
					c.UI.Printf("  %s\n", arg)
				}
			}
			if len(cmd.Options) > 0 {
				c.UI.Println("Options:")
				for _, opt := range cmd.Options {
					c.UI.Printf("  %s\n", opt)
				}
			}
			if len(cmd.Examples) > 0 {
				c.UI.Println("Examples:")
				for _, ex := range cmd.Examples {
					c.UI.Printf("  %s\n", ex)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("no help found for %s %s", scope, operation)
}

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
		Operation: "mod",
		ShortDesc: "Modify an existing user",
		LongDesc:  "Modifies the username or password of an existing user account.",
		Syntax:    "user mod <username> [new_username] [new_password]",
		Arguments: []string{"username: The name of the user to modify", "new_username: (Optional) The new username", "new_password: (Optional) The new password"},
		Examples:  []string{"user mod john", "user mod john johnny", "user mod john johnny new_password"},
	},
	{
		Scope:     "user",
		Operation: "del",
		ShortDesc: "Delete a user",
		LongDesc:  "Deletes an existing user account and all associated mindmaps.",
		Syntax:    "user del <username>",
		Arguments: []string{"username: The name of the user to delete"},
		Examples:  []string{"user del john"},
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
		Operation: "del",
		ShortDesc: "Delete a mindmap",
		LongDesc:  "Deletes the specified mindmap or all mindmaps owned by the current user if no name is provided.",
		Syntax:    "mindmap del [mindmap_name]",
		Arguments: []string{"mindmap_name: (Optional) The name of the mindmap to delete"},
		Examples:  []string{"mindmap del", "mindmap del my_ideas"},
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
		Syntax:    "mindmap view [logical_index] [--index]",
		Arguments: []string{"logical_index: (Optional) The logical index of the node to view", "--index: (Optional) Show node indices"},
		Examples:  []string{"mindmap view", "mindmap view 1.2", "mindmap view --index"},
	},
	{
		Scope:     "node",
		Operation: "add",
		ShortDesc: "Add a new node",
		LongDesc:  "Adds a new node to the current mindmap.",
		Syntax:    "node add <parent> <content> [<extra field label>:<extra field value>]... [--index]",
		Arguments: []string{"parent: The parent node identifier", "content: The content of the new node", "extra: (Optional) Extra fields in the format label:value", "--index: (Optional) Use numeric index instead of logical index"},
		Examples:  []string{"node add 1 \"New idea\"", "node add 2.1 \"Sub-idea\" priority:high --index"},
	},
	{
		Scope:     "node",
		Operation: "mod",
		ShortDesc: "Modify a node",
		LongDesc:  "Modifies the content or extra fields of an existing node.",
		Syntax:    "node mod <node> <content> [<extra field label>:<extra field value>]... [--index]",
		Arguments: []string{"node: The node identifier to modify", "content: The new content for the node", "extra: (Optional) Extra fields to modify in the format label:value", "--index: (Optional) Use numeric index instead of logical index"},
		Examples:  []string{"node mod 1.1 \"Updated idea\"", "node mod 2 \"Changed content\" priority:low --index"},
	},
	{
		Scope:     "node",
		Operation: "move",
		ShortDesc: "Move a node",
		LongDesc:  "Moves a node to a new parent in the current mindmap.",
		Syntax:    "node move <source> <target> [--index]",
		Arguments: []string{"source: The identifier of the node to move", "target: The identifier of the new parent node", "--index: (Optional) Use numeric index instead of logical index"},
		Examples:  []string{"node move 1.2 2.1", "node move 3 1 --index"},
	},
	{
		Scope:     "node",
		Operation: "del",
		ShortDesc: "Delete a node",
		LongDesc:  "Deletes a node and its subtree from the current mindmap.",
		Syntax:    "node del <node> [--index]",
		Arguments: []string{"node: The identifier of the node to delete", "--index: (Optional) Use numeric index instead of logical index"},
		Examples:  []string{"node del 1.2", "node del 3 --index"},
	},
	{
		Scope:     "node",
		Operation: "find",
		ShortDesc: "Find nodes",
		LongDesc:  "Searches for nodes in the current mindmap based on a query string.",
		Syntax:    "node find <query> [--index]",
		Arguments: []string{"query: The search query string", "--index: (Optional) Show node indices in the results"},
		Examples:  []string{"node find \"important idea\"", "node find project --index"},
	},
	{
		Scope:     "node",
		Operation: "sort",
		ShortDesc: "Sort child nodes",
		LongDesc:  "Sorts the child nodes of a specified node based on content or an extra field.",
		Syntax:    "node sort [identifier] [field] [--reverse] [--index]",
		Arguments: []string{"identifier: (Optional) The node whose children to sort. Defaults to root", "field: (Optional) The field to sort by. Defaults to node content", "--reverse: (Optional) Sort in descending order", "--index: (Optional) Use numeric index instead of logical index"},
		Examples:  []string{"node sort", "node sort 1.2 priority --reverse", "node sort 2 --index"},
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
