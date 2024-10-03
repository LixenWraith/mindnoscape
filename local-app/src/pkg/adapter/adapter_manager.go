package adapter

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
	"mindnoscape/local-app/src/pkg/session"
)

// Adapter type constants
const (
	AdapterTypeCLI = "CLI"
	AdapterTypeWeb = "WEB"
	AdapterTypeAPI = "API"
)

// AdapterInstance represents an instance of an adapter
type AdapterInstance interface {
	// CommandRun Command HandleCommand processes a command and returns the result
	CommandRun(connID string, cmd model.Command) (interface{}, error)

	// AdapterStart AdapterRun Run starts the adapter instance
	AdapterStart() error

	// AdapterStop Stop terminates the adapter instance
	AdapterStop() error

	// GetType returns the type of the adapter
	GetType() string
}

// AdapterManager manages all adapter instances
type AdapterManager struct {
	adapters        map[string]AdapterInstance
	adapterMutex    sync.RWMutex
	sessionManager  *session.SessionManager
	adapterSessions sync.Map
	cmdChan         chan commandRequest
	stopChan        chan struct{}
	logger          *log.Logger
}

// commandRequest represents a request to execute a command within a specific session and carries a result channel
type commandRequest struct {
	AdapterType string
	ConnID      string
	SessionID   string
	Command     model.Command
	ResultChan  chan commandResult
}

type commandResult struct {
	Result interface{}
	Error  error
}

// NewAdapterManager creates a new AdapterManager
func NewAdapterManager(sm *session.SessionManager, logger *log.Logger) (*AdapterManager, error) {
	am := &AdapterManager{
		adapters:       make(map[string]AdapterInstance),
		sessionManager: sm,
		cmdChan:        make(chan commandRequest),
		stopChan:       make(chan struct{}),
		logger:         logger,
	}
	go am.commandHandler()
	am.logger.Info(context.Background(), "AdapterManager initialized", nil)

	// Initialize and add CLIAdapter
	cliAdapter, err := NewCLIAdapter(am, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CLI adapter: %v", err)
	}
	err = am.AdapterAdd(cliAdapter)
	if err != nil {
		return nil, fmt.Errorf("failed to add CLI adapter: %v", err)
	}

	// Initialize other adapters here as needed
	// webAdapter, err := NewWebAdapter(am, logger)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to initialize Web adapter: %v", err)
	// }
	// err = am.AdapterAdd(webAdapter)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to add Web adapter: %v", err)
	// }

	return am, nil
}

// AdapterAdd creates a new adapter instance
func (am *AdapterManager) AdapterAdd(adapter AdapterInstance) error {
	adapterType := adapter.GetType()
	am.adapterMutex.Lock()
	defer am.adapterMutex.Unlock()

	if _, exists := am.adapters[adapterType]; exists {
		return fmt.Errorf("adapter of type %s already registered", adapterType)
	}

	am.adapters[adapterType] = adapter
	am.logger.Info(context.Background(), "Adapter registered", log.Fields{"adapterType": adapterType})
	return adapter.AdapterStart()
}

// AdapterGet returns the adapter instance for the given adapter type
func (am *AdapterManager) AdapterGet(adapterType string) (AdapterInstance, error) {
	am.adapterMutex.RLock()
	defer am.adapterMutex.RUnlock()

	adapter, exists := am.adapters[adapterType]
	if !exists {
		return nil, fmt.Errorf("adapter of type %s not found", adapterType)
	}

	// Placeholder check (replace with actual implementation if needed)
	if adapter == nil {
		return nil, fmt.Errorf("adapter of type %s is nil (placeholder)", adapterType)
	}

	return adapter, nil
}

// AdapterDelete stops and deletes an adapter instance
func (am *AdapterManager) AdapterDelete(adapterType string) error {
	am.adapterMutex.Lock()
	defer am.adapterMutex.Unlock()

	adapter, exists := am.adapters[adapterType]
	if !exists {
		return fmt.Errorf("adapter of type %s not found", adapterType)
	}

	err := adapter.AdapterStop()
	if err != nil {
		am.logger.Error(context.Background(), "Error stopping adapter", log.Fields{"adapterType": adapterType, "error": err})
	}

	delete(am.adapters, adapterType)
	am.logger.Info(context.Background(), "Adapter unregistered", log.Fields{"adapterType": adapterType})
	return nil
}

// CommandRun runs a command on a specific adapter instance
func (am *AdapterManager) CommandRun(connID string, cmd model.Command) (interface{}, error) {
	am.logger.Info(context.Background(), "Processing command through adapter manager", log.Fields{"connID": connID, "command": cmd})

	cmd.Scope, cmd.Operation = am.expandCommand(cmd.Scope, cmd.Operation)

	sessionID, err := am.getOrCreateSession(connID)
	if err != nil {
		am.logger.Error(context.Background(), "Failed to get or create session", log.Fields{"error": err, "connID": connID})
		return nil, fmt.Errorf("failed to get or create session: %w", err)
	}

	// Log command in command log
	am.logger.Command(context.Background(), "Command received", log.Fields{
		"sessionID": sessionID,
		"scope":     cmd.Scope,
		"operation": cmd.Operation,
		"args":      cmd.Args,
	})

	resultChan := make(chan commandResult)
	am.cmdChan <- commandRequest{
		ConnID:     connID,
		SessionID:  sessionID,
		Command:    cmd,
		ResultChan: resultChan,
	}

	result := <-resultChan
	if result.Error != nil {
		am.logger.Error(context.Background(), "Command execution failed", log.Fields{"connID": connID, "sessionID": sessionID, "command": cmd, "error": result.Error})
		return nil, result.Error
	}

	am.logger.Info(context.Background(), "Command executed successfully", log.Fields{"connID": connID, "sessionID": sessionID, "command": cmd})
	return result.Result, nil
}

// Shutdown stops all adapter instances and the command handler
func (am *AdapterManager) Shutdown() {
	close(am.stopChan)
	am.adapterMutex.Lock()
	defer am.adapterMutex.Unlock()

	for adapterType, adapter := range am.adapters {
		err := adapter.AdapterStop()
		if err != nil {
			am.logger.Error(context.Background(), "Error stopping adapter", log.Fields{"adapterType": adapterType, "error": err})
		}
	}
	am.adapters = make(map[string]AdapterInstance)
	am.logger.Info(context.Background(), "AdapterManager shut down", nil)
}

func (am *AdapterManager) commandHandler() {
	for {
		select {
		case req := <-am.cmdChan:
			sessionInstance, exists := am.sessionManager.SessionGet(req.SessionID)
			if !exists {
				am.logger.Error(context.Background(), "Session not found", log.Fields{"connID": req.ConnID, "sessionID": req.SessionID})
				req.ResultChan <- commandResult{Error: fmt.Errorf("session not found")}
				continue
			}

			// Create and validate the SessionCommand
			sessionCmd := session.NewCommand(req.Command, am.logger)
			if err := sessionCmd.Validate(); err != nil {
				am.logger.Error(context.Background(), "Command validation failed", log.Fields{"error": err, "command": req.Command})
				req.ResultChan <- commandResult{Error: err}
				continue
			}

			// Execute the command
			result, err := sessionInstance.CommandRun(req.Command)
			if err != nil {
				am.logger.Error(context.Background(), "Command execution failed", log.Fields{"error": err, "command": req.Command, "connID": req.ConnID, "sessionID": req.SessionID})
				req.ResultChan <- commandResult{Error: err}
			} else {
				am.logger.Info(context.Background(), "Command executed successfully", log.Fields{"command": req.Command, "connID": req.ConnID, "sessionID": req.SessionID})
				req.ResultChan <- commandResult{Result: result}
			}
		case <-am.stopChan:
			return
		}
	}
}

func (am *AdapterManager) getOrCreateSession(connID string) (string, error) {
	if sessionID, ok := am.adapterSessions.Load(connID); ok {
		return sessionID.(string), nil
	}

	sessionID, err := am.sessionManager.SessionAdd()
	if err != nil {
		return "", fmt.Errorf("failed to create new session: %w", err)
	}

	am.adapterSessions.Store(connID, sessionID)
	return sessionID, nil
}

// expandCommand converts concise (one letter) commands and operations to the long (complete string) format
func (am *AdapterManager) expandCommand(scope, operation string) (string, string) {
	expandedScope := scope
	expandedOperation := operation

	// Expand scope if it's a single letter
	if len(scope) == 1 {
		switch scope {
		case "s":
			expandedScope = "system"
		case "u":
			expandedScope = "user"
		case "m":
			expandedScope = "mindmap"
		case "n":
			expandedScope = "node"
		case "h":
			expandedScope = "help"
		}
	}

	// Expand operation if it's a single letter
	if len(operation) == 1 {
		switch expandedScope {
		case "user":
			switch operation {
			case "a":
				expandedOperation = "add"
			case "u":
				expandedOperation = "update"
			case "d":
				expandedOperation = "delete"
			case "s":
				expandedOperation = "select"
			case "l":
				expandedOperation = "list"
			}
		case "mindmap":
			switch operation {
			case "a":
				expandedOperation = "add"
			case "u":
				expandedOperation = "update"
			case "d":
				expandedOperation = "delete"
			case "p":
				expandedOperation = "permission"
			case "i":
				expandedOperation = "import"
			case "e":
				expandedOperation = "export"
			case "s":
				expandedOperation = "select"
			case "l":
				expandedOperation = "list"
			case "v":
				expandedOperation = "view"
			}
		case "node":
			switch operation {
			case "a":
				expandedOperation = "add"
			case "u":
				expandedOperation = "update"
			case "m":
				expandedOperation = "move"
			case "d":
				expandedOperation = "delete"
			case "f":
				expandedOperation = "find"
			case "s":
				expandedOperation = "sort"
			}
		case "system":
			switch operation {
			case "e":
				expandedOperation = "exit"
			case "q":
				expandedOperation = "quit"
			}
		}
	}

	return expandedScope, expandedOperation
}

// GetHelp returns help information based on the provided arguments
func (am *AdapterManager) GetHelp(args []string) string {
	switch len(args) {
	case 0:
		return am.getGeneralHelp()
	case 1:
		return am.getScopeHelp(args[0])
	case 2:
		return am.getOperationHelp(args[0], args[1])
	default:
		return "Invalid help command. Use 'help [scope] [operation]'"
	}
}

func (am *AdapterManager) getGeneralHelp() string {
	var help strings.Builder
	help.WriteString("Command syntax: <scope> [operation] [arguments] [options]\n\n")
	help.WriteString("Available commands:\n")
	currentScope := ""
	for _, cmd := range commandHelps {
		if cmd.Scope != currentScope {
			help.WriteString(fmt.Sprintf("\n%s:\n", cmd.Scope))
			currentScope = cmd.Scope
		}
		help.WriteString(fmt.Sprintf("  %-15s %s\n", cmd.Operation, cmd.ShortDesc))
	}
	return help.String()
}

func (am *AdapterManager) getScopeHelp(scope string) string {
	var help strings.Builder
	help.WriteString(fmt.Sprintf("Commands for %s:\n\n", scope))
	for _, cmd := range commandHelps {
		if cmd.Scope == scope {
			help.WriteString(fmt.Sprintf("%-15s %s\n", cmd.Operation, cmd.ShortDesc))
		}
	}
	return help.String()
}

func (am *AdapterManager) getOperationHelp(scope, operation string) string {
	for _, cmd := range commandHelps {
		if cmd.Scope == scope && cmd.Operation == operation {
			var help strings.Builder
			help.WriteString(fmt.Sprintf("Command: %s %s\n", scope, operation))
			help.WriteString(fmt.Sprintf("Description: %s\n", cmd.LongDesc))
			help.WriteString(fmt.Sprintf("Syntax: %s\n", cmd.Syntax))
			if len(cmd.Arguments) > 0 {
				help.WriteString("Arguments:\n")
				for _, arg := range cmd.Arguments {
					help.WriteString(fmt.Sprintf("  %s\n", arg))
				}
			}
			if len(cmd.Options) > 0 {
				help.WriteString("Options:\n")
				for _, opt := range cmd.Options {
					help.WriteString(fmt.Sprintf("  %s\n", opt))
				}
			}
			if len(cmd.Examples) > 0 {
				help.WriteString("Examples:\n")
				for _, ex := range cmd.Examples {
					help.WriteString(fmt.Sprintf("  %s\n", ex))
				}
			}
			return help.String()
		}
	}
	return fmt.Sprintf("No help found for %s %s\n", scope, operation)
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
