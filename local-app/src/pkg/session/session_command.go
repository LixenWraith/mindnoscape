package session

import (
	"context"
	"errors"
	"fmt"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// SessionCommand wraps the model.Command and adds session-specific functionality
type SessionCommand struct {
	model.Command
	logger *log.Logger
}

// NewSessionCommand creates a new SessionCommand from a model.Command
func NewSessionCommand(cmd model.Command, logger *log.Logger) SessionCommand {
	return SessionCommand{Command: cmd, logger: logger}
}

// Validate checks if the command is valid
func (c *SessionCommand) Validate() error {
	ctx := context.Background()
	c.logger.Info(ctx, "Validating command", log.Fields{"scope": c.Scope, "operation": c.Operation})

	if c.Scope == "" {
		c.logger.Error(ctx, "Command scope is empty", nil)
		return errors.New("command scope is required")
	}
	if c.Operation == "" {
		c.logger.Error(ctx, "Command operation is empty", nil)
		return errors.New("command operation is required")
	}
	return c.validateScopeAndOperation()
}

// validateScopeAndOperation checks if the scope and operation are valid
func (c *SessionCommand) validateScopeAndOperation() error {
	ctx := context.Background()
	c.logger.Debug(ctx, "Validating scope and operation", log.Fields{"scope": c.Scope, "operation": c.Operation})

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
		c.logger.Error(ctx, "Invalid command scope", log.Fields{"scope": c.Scope})
		return fmt.Errorf("invalid command scope: %s", c.Scope)
	}
}

func (c *SessionCommand) validateUserCommand() error {
	ctx := context.Background()
	c.logger.Debug(ctx, "Validating user command", log.Fields{"operation": c.Operation})

	switch c.Operation {
	case "add":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			c.logger.Error(ctx, "Invalid number of arguments for user add command", log.Fields{"argCount": len(c.Args)})
			return errors.New("user add command requires 1 or 2 arguments: <username> [password]")
		}
	case "update":
		if len(c.Args) < 1 || len(c.Args) > 3 {
			c.logger.Error(ctx, "Invalid number of arguments for user update command", log.Fields{"argCount": len(c.Args)})
			return errors.New("user update command requires 1 to 3 arguments: <username> [new_username] [new_password]")
		}
	case "delete", "select":
		if len(c.Args) != 1 {
			c.logger.Error(ctx, "Invalid number of arguments for user command", log.Fields{"operation": c.Operation, "argCount": len(c.Args)})
			return fmt.Errorf("user %s command requires 1 argument: <username>", c.Operation)
		}
	default:
		c.logger.Error(ctx, "Invalid user operation", log.Fields{"operation": c.Operation})
		return fmt.Errorf("invalid user operation: %s", c.Operation)
	}
	return nil
}

func (c *SessionCommand) validateMindmapCommand() error {
	ctx := context.Background()
	c.logger.Debug(ctx, "Validating mindmap command", log.Fields{"operation": c.Operation})

	switch c.Operation {
	case "add":
		if len(c.Args) != 1 {
			c.logger.Error(ctx, "Invalid number of arguments for mindmap add command", log.Fields{"argCount": len(c.Args)})
			return errors.New("mindmap add command requires 1 argument: <mindmap_name>")
		}
	case "delete", "select":
		if len(c.Args) > 1 {
			c.logger.Error(ctx, "Invalid number of arguments for mindmap command", log.Fields{"operation": c.Operation, "argCount": len(c.Args)})
			return fmt.Errorf("mindmap %s command requires 0 or 1 argument: [mindmap_name]", c.Operation)
		}
	case "permission":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			c.logger.Error(ctx, "Invalid number of arguments for mindmap permission command", log.Fields{"argCount": len(c.Args)})
			return errors.New("mindmap permission command requires 1 or 2 arguments: <mindmap_name> [public|private]")
		}
	case "import", "export":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			c.logger.Error(ctx, "Invalid number of arguments for mindmap import/export command", log.Fields{"operation": c.Operation, "argCount": len(c.Args)})
			return fmt.Errorf("mindmap %s command requires 1 or 2 arguments: <filename> [json|xml]", c.Operation)
		}
	case "list":
		if len(c.Args) != 0 {
			c.logger.Error(ctx, "Invalid number of arguments for mindmap list command", log.Fields{"argCount": len(c.Args)})
			return errors.New("mindmap list command does not accept any arguments")
		}
	case "view":
		if len(c.Args) > 2 {
			c.logger.Error(ctx, "Invalid number of arguments for mindmap view command", log.Fields{"argCount": len(c.Args)})
			return errors.New("mindmap view command accepts at most 2 arguments: [index] [--id]")
		}
	default:
		c.logger.Error(ctx, "Invalid mindmap operation", log.Fields{"operation": c.Operation})
		return fmt.Errorf("invalid mindmap operation: %s", c.Operation)
	}
	return nil
}

func (c *SessionCommand) validateNodeCommand() error {
	ctx := context.Background()
	c.logger.Debug(ctx, "Validating node command", log.Fields{"operation": c.Operation})

	switch c.Operation {
	case "add":
		if len(c.Args) < 2 {
			c.logger.Error(ctx, "Invalid number of arguments for node add command", log.Fields{"argCount": len(c.Args)})
			return errors.New("node add command requires at least 2 arguments: <parent> <content> [<extra field label>:<extra field value>]... [--id]")
		}
	case "update":
		if len(c.Args) < 2 {
			c.logger.Error(ctx, "Invalid number of arguments for node update command", log.Fields{"argCount": len(c.Args)})
			return errors.New("node update command requires at least 2 arguments: <node> <content> [<extra field label>:<extra field value>]... [--id]")
		}
	case "move":
		if len(c.Args) < 2 || len(c.Args) > 3 {
			c.logger.Error(ctx, "Invalid number of arguments for node move command", log.Fields{"argCount": len(c.Args)})
			return errors.New("node move command requires 2 or 3 arguments: <source> <target> [--id]")
		}
	case "delete":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			c.logger.Error(ctx, "Invalid number of arguments for node delete command", log.Fields{"argCount": len(c.Args)})
			return errors.New("node delete command requires 1 or 2 arguments: <node> [--id]")
		}
	case "find":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			c.logger.Error(ctx, "Invalid number of arguments for node find command", log.Fields{"argCount": len(c.Args)})
			return errors.New("node find command requires 1 or 2 arguments: <query> [--id]")
		}
	case "sort":
		if len(c.Args) > 4 {
			c.logger.Error(ctx, "Invalid number of arguments for node sort command", log.Fields{"argCount": len(c.Args)})
			return errors.New("node sort command accepts at most 4 arguments: [identifier] [field] [--reverse] [--id]")
		}
	default:
		c.logger.Error(ctx, "Invalid node operation", log.Fields{"operation": c.Operation})
		return fmt.Errorf("invalid node operation: %s", c.Operation)
	}
	return nil
}

func (c *SessionCommand) validateSystemCommand() error {
	ctx := context.Background()
	c.logger.Debug(ctx, "Validating system command", log.Fields{"operation": c.Operation})

	switch c.Operation {
	case "exit", "quit":
		if len(c.Args) != 0 {
			c.logger.Error(ctx, "Invalid number of arguments for system command", log.Fields{"operation": c.Operation, "argCount": len(c.Args)})
			return fmt.Errorf("system %s command does not accept any arguments", c.Operation)
		}
	default:
		c.logger.Error(ctx, "Invalid system operation", log.Fields{"operation": c.Operation})
		return fmt.Errorf("invalid system operation: %s", c.Operation)
	}
	return nil
}
