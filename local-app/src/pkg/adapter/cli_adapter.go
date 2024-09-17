// Package adapter provides an adapter for the CLI interface to interact with the session package.
package adapter

import (
	"context"
	"fmt"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
	"mindnoscape/local-app/src/pkg/session"
)

// CLIAdapter provides command-line interface support for managing sessions and executing commands
type CLIAdapter struct {
	sessionID      string
	sessionManager *session.SessionManager
	cmdChan        chan model.Command
	resultChan     chan interface{}
	stopChan       chan struct{}
	errChan        chan error
	logger         *log.Logger
}

// NewCLIAdapter creates a new instance of CLIAdapter using the provided SessionManager
func NewCLIAdapter(sm *session.SessionManager, logger *log.Logger) (*CLIAdapter, error) {
	sessionID, err := sm.SessionAdd()
	if err != nil {
		logger.Error(context.Background(), "Failed to create session", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	logger.Info(context.Background(), "Created new session", log.Fields{"sessionID": sessionID})

	return &CLIAdapter{
		sessionManager: sm,
		sessionID:      sessionID,
		cmdChan:        make(chan model.Command),
		resultChan:     make(chan interface{}),
		errChan:        make(chan error),
		stopChan:       make(chan struct{}),
		logger:         logger,
	}, nil
}

// AdapterStart AdapterRun Run starts the CLI adapter's main loop
func (a *CLIAdapter) AdapterStart() error {
	a.logger.Info(context.Background(), "Starting CLI adapter", nil)
	go func() {
		a.logger.Info(context.Background(), "Started command processing goroutine", nil)
		for {
			select {
			case cmd := <-a.cmdChan:
				a.logger.Command(context.Background(), "Received command", log.Fields{"command": cmd})
				if a.sessionID == "" {
					a.logger.Error(context.Background(), "No active session", nil)
					a.errChan <- fmt.Errorf("no active session")
					continue
				}
				result, err := a.sessionManager.SessionRun(a.sessionID, cmd)
				if err != nil {
					a.logger.Error(context.Background(), "Error processing command", log.Fields{"error": err, "command": cmd})
					a.errChan <- err
				} else {
					a.logger.Info(context.Background(), "Command processed successfully", log.Fields{"command": cmd})
					a.resultChan <- result
				}
			case <-a.stopChan:
				a.logger.Info(context.Background(), "Stopping command processing goroutine", nil)
				return
			}
		}
	}()
	a.logger.Info(context.Background(), "CLI adapter started", nil)
	return nil
}

// AdapterStop Stop signals the CLI adapter to stop
func (a *CLIAdapter) AdapterStop() error {
	if a.stopChan != nil {
		close(a.stopChan)
	}
	a.logger.Info(context.Background(), "CLI adapter stopped", nil)
	return nil
}

// GetType returns the type of the adapter
func (a *CLIAdapter) GetType() string {
	return "CLI"
}

// CommandProcess processes a command and returns the result
func (a *CLIAdapter) CommandProcess(cmd model.Command) (interface{}, error) {
	a.logger.Info(context.Background(), "Processing command", log.Fields{"command": cmd})
	// Expand short commands to full commands
	cmd.Scope, cmd.Operation = a.expandCommand(cmd.Scope, cmd.Operation)

	// Validate the command
	sessionCmd := session.NewSessionCommand(cmd, a.logger)
	if err := sessionCmd.Validate(); err != nil {
		a.logger.Error(context.Background(), "Command validation failed", log.Fields{"error": err, "command": cmd})
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Send command to the channel
	a.cmdChan <- cmd

	a.logger.Info(context.Background(), "Command sent to processing channel", log.Fields{"command": cmd})

	// Wait for result or error
	select {
	case result := <-a.resultChan:
		a.logger.Info(context.Background(), "Command processed successfully", log.Fields{"command": cmd, "result": result})
		return result, nil
	case err := <-a.errChan:
		a.logger.Error(context.Background(), "Command processing failed", log.Fields{"error": err, "command": cmd})
		return nil, err
	}
}

// expandCommand converts concise (one letter) commands and operations to the long (complete string) format
func (a *CLIAdapter) expandCommand(scope, operation string) (string, string) {
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
