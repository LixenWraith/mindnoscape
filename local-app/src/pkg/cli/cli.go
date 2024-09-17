package cli

import (
	"context"
	"fmt"
	"io"
	"mindnoscape/local-app/src/pkg/log"
	"os"
	"strings"

	"mindnoscape/local-app/src/pkg/adapter"
	"mindnoscape/local-app/src/pkg/model"
)

// CLI represents the command-line interface
type CLI struct {
	adapter adapter.AdapterInstance
	stopCh  chan struct{}
	reader  io.Reader
	writer  io.Writer
	logger  *log.Logger
}

// NewCLI creates a new CLI instance
func NewCLI(adapter adapter.AdapterInstance, logger *log.Logger) (*CLI, error) {
	return &CLI{
		adapter: adapter,
		stopCh:  make(chan struct{}),
		reader:  os.Stdin,
		writer:  os.Stdout,
		logger:  logger,
	}, nil
}

// Run starts the CLI and handles user input
func (c *CLI) Run() error {
	fmt.Println("Welcome to Mindnoscape CLI!")
	fmt.Println("Type 'help' for a list of commands or 'exit' to quit.")

	if err := c.adapter.AdapterStart(); err != nil {
		return fmt.Errorf("failed to start CLI adapter: %w", err)
	}
	defer func() {
		if err := c.adapter.AdapterStop(); err != nil {
			fmt.Printf("Error stopping CLI adapter: %v\n", err)
		}
	}()

	c.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: CLI adapter started"))

	// Main loop
	for {
		fmt.Print("> ")
		input, err := c.readLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			c.logger.LogInfo(context.Background(), fmt.Sprintf("Error reading input: %v\n", err))
			continue
		}

		if input == "exit" || input == "quit" {
			break
		}

		// Parse input into model.Command
		cmd, err := c.parseCommand(input)
		if err != nil {
			fmt.Printf("Error parsing command: %v\n", err)
			c.logger.LogInfo(context.Background(), fmt.Sprintf("Error parsing command: %v\n", err))
			continue
		}

		// Check for help command
		if cmd.Scope == "help" {
			// Split input into words
			args := strings.Fields(input)
			if len(args) == 0 { // General help
				c.printHelp(nil)
			} else {
				c.printHelp(args[1:])
			}
			continue
		}

		// Pass command to the adapter
		result, err := c.adapter.CommandProcess(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else if result != nil {
			fmt.Printf("Result: %v\n", result)
		}
	}

	return nil
}

// readLine reads a line of input from the reader
func (c *CLI) readLine() (string, error) {
	var line strings.Builder
	for {
		var b [1]byte
		n, err := c.reader.Read(b[:])
		if err != nil {
			if err == io.EOF && line.Len() > 0 {
				return line.String(), nil
			}
			return "", err
		}
		if n == 0 {
			continue
		}
		if b[0] == '\n' {
			return line.String(), nil
		}
		line.WriteByte(b[0])
	}
}

// Stop signals the CLI to stop its main loop
func (c *CLI) Stop() {
	close(c.stopCh)
}

// parseCommand parses user input into a model.Command
func (c *CLI) parseCommand(input string) (model.Command, error) {
	args := strings.Fields(input)
	if len(args) == 0 {
		return model.Command{}, fmt.Errorf("empty command")
	}

	cmd := model.Command{
		Scope:     strings.ToLower(args[0]),
		Operation: "",
		Args:      []string{},
	}

	if len(args) > 1 {
		cmd.Operation = strings.ToLower(args[1])
		cmd.Args = args[2:]
	}

	return cmd, nil
}

// printHelp prints the help message based on the provided arguments
func (c *CLI) printHelp(args []string) {
	switch len(args) {
	case 0:
		c.showGeneralHelp()
	case 1:
		c.showScopeHelp(args[0])
	case 2:
		c.showOperationHelp(args[0], args[1])
	default:
		fmt.Println("Invalid help command. Use 'help [scope] [operation]'")
	}
}

// showGeneralHelp displays an overview of all available commands grouped by scope
func (c *CLI) showGeneralHelp() {
	fmt.Println("Command syntax: <scope> [operation] [arguments] [options]")
	fmt.Println("\nAvailable commands:")
	currentScope := ""
	for _, cmd := range commandHelps {
		if cmd.Scope != currentScope {
			fmt.Printf("\n%s:\n", cmd.Scope)
			currentScope = cmd.Scope
		}
		fmt.Printf("  %-15s %s\n", cmd.Operation, cmd.ShortDesc)
	}
}

// showScopeHelp displays help information for all commands within a specific scope
func (c *CLI) showScopeHelp(scope string) {
	fmt.Printf("Commands for %s:\n\n", scope)
	for _, cmd := range commandHelps {
		if cmd.Scope == scope {
			fmt.Printf("%-15s %s\n", cmd.Operation, cmd.ShortDesc)
		}
	}
}

// showOperationHelp displays detailed help information for a specific operation within a scope
func (c *CLI) showOperationHelp(scope, operation string) {
	for _, cmd := range commandHelps {
		if cmd.Scope == scope && cmd.Operation == operation {
			fmt.Printf("Command: %s %s\n", scope, operation)
			fmt.Printf("Description: %s\n", cmd.LongDesc)
			fmt.Printf("Syntax: %s\n", cmd.Syntax)
			if len(cmd.Arguments) > 0 {
				fmt.Println("Arguments:")
				for _, arg := range cmd.Arguments {
					fmt.Printf("  %s\n", arg)
				}
			}
			if len(cmd.Options) > 0 {
				fmt.Println("Options:")
				for _, opt := range cmd.Options {
					fmt.Printf("  %s\n", opt)
				}
			}
			if len(cmd.Examples) > 0 {
				fmt.Println("Examples:")
				for _, ex := range cmd.Examples {
					fmt.Printf("  %s\n", ex)
				}
			}
			return
		}
	}
	fmt.Printf("No help found for %s %s\n", scope, operation)
}

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
		LongDesc:  "Selects the specified user account. If no username is provided, deselects the current user.",
		Syntax:    "user select [username]",
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
		LongDesc:  "Deletes the specified mindmap. If no mindmap name is provided deletes the current mindmap and if no mindmap is selected, deletes all mindmaps owned by the current user.",
		Syntax:    "mindmap delete [mindmap_name]",
		Arguments: []string{"mindmap_name: (Optional) The name of the mindmap to delete"},
		Examples:  []string{"mindmap delete", "mindmap delete my_ideas"},
	},
	{
		Scope:     "mindmap",
		Operation: "permission",
		ShortDesc: "Modify mindmap permission",
		LongDesc:  "Changes or displays the permission of a mindmap to public or private.",
		Syntax:    "mindmap permission <mindmap_name> [public|private]",
		Arguments: []string{"mindmap_name: The name of the mindmap", "permission: 'public' or 'private'"},
		Examples:  []string{"mindmap permission my_mindmap", "mindmap permission my_ideas public", "mindmap permission project_x private"},
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
		Arguments: []string{"query: The search query string", "--id: (Optional) Show node id in the results"},
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
