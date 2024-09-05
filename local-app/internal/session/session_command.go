package session

import (
	"errors"
	"fmt"
	"mindnoscape/local-app/internal/model"
)

// SessionCommand wraps the model.Command and adds session-specific functionality
type SessionCommand struct {
	model.Command
}

// NewSessionCommand creates a new SessionCommand from a model.Command
func NewSessionCommand(cmd model.Command) SessionCommand {
	return SessionCommand{Command: cmd}
}

// Validate checks if the command is valid
func (c *SessionCommand) Validate() error {
	// Implement validation logic here
	if c.Scope == "" {
		return errors.New("command scope is required")
	}
	if c.Operation == "" {
		return errors.New("command operation is required")
	}
	return c.validateScopeAndOperation()
}

// validateScopeAndOperation checks if the scope and operation are valid
func (c *SessionCommand) validateScopeAndOperation() error {
	// Implement validation logic for each scope and operation
	switch c.Scope {
	case "user":
		return c.validateUserCommand()
	case "mindmap":
		return c.validateMindmapCommand()
	case "node":
		return c.validateNodeCommand()
	case "system":
		return c.validateSystemCommand()
	default:
		return fmt.Errorf("invalid command scope: %s", c.Scope)
	}
}

func (c *SessionCommand) validateUserCommand() error {
	switch c.Operation {
	case "add":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			return errors.New("user add command requires 1 or 2 arguments: <username> [password]")
		}
	case "update":
		if len(c.Args) < 1 || len(c.Args) > 3 {
			return errors.New("user update command requires 1 to 3 arguments: <username> [new_username] [new_password]")
		}
	case "delete", "select":
		if len(c.Args) != 1 {
			return fmt.Errorf("user %s command requires 1 argument: <username>", c.Operation)
		}
	default:
		return fmt.Errorf("invalid user operation: %s", c.Operation)
	}
	return nil
}

func (c *SessionCommand) validateMindmapCommand() error {
	switch c.Operation {
	case "add":
		if len(c.Args) != 1 {
			return errors.New("mindmap add command requires 1 argument: <mindmap_name>")
		}
	case "delete", "select":
		if len(c.Args) > 1 {
			return fmt.Errorf("mindmap %s command requires 0 or 1 argument: [mindmap_name]", c.Operation)
		}
	case "permission":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			return errors.New("mindmap permission command requires 1 or 2 arguments: <mindmap_name> [public|private]")
		}
	case "import", "export":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			return fmt.Errorf("mindmap %s command requires 1 or 2 arguments: <filename> [json|xml]", c.Operation)
		}
	case "list":
		if len(c.Args) != 0 {
			return errors.New("mindmap list command does not accept any arguments")
		}
	case "view":
		if len(c.Args) > 2 {
			return errors.New("mindmap view command accepts at most 2 arguments: [index] [--id]")
		}
	default:
		return fmt.Errorf("invalid mindmap operation: %s", c.Operation)
	}
	return nil
}

func (c *SessionCommand) validateNodeCommand() error {
	switch c.Operation {
	case "add":
		if len(c.Args) < 2 {
			return errors.New("node add command requires at least 2 arguments: <parent> <content> [<extra field label>:<extra field value>]... [--id]")
		}
	case "update":
		if len(c.Args) < 2 {
			return errors.New("node update command requires at least 2 arguments: <node> <content> [<extra field label>:<extra field value>]... [--id]")
		}
	case "move":
		if len(c.Args) < 2 || len(c.Args) > 3 {
			return errors.New("node move command requires 2 or 3 arguments: <source> <target> [--id]")
		}
	case "delete":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			return errors.New("node delete command requires 1 or 2 arguments: <node> [--id]")
		}
	case "find":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			return errors.New("node find command requires 1 or 2 arguments: <query> [--id]")
		}
	case "sort":
		if len(c.Args) > 4 {
			return errors.New("node sort command accepts at most 4 arguments: [identifier] [field] [--reverse] [--id]")
		}
	default:
		return fmt.Errorf("invalid node operation: %s", c.Operation)
	}
	return nil
}

func (c *SessionCommand) validateSystemCommand() error {
	switch c.Operation {
	case "exit", "quit":
		if len(c.Args) != 0 {
			return fmt.Errorf("system %s command does not accept any arguments", c.Operation)
		}
	default:
		return fmt.Errorf("invalid system operation: %s", c.Operation)
	}
	return nil
}
