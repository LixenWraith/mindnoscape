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
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Created new session with ID: %s\n", sessionID))
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
	a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Entering AdapterStart method\n"))
	go func() {
		a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Started command processing goroutine\n"))
		for {
			a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Waiting for command in AdapterStart\n"))
			select {
			case cmd := <-a.cmdChan:
				a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Received command in AdapterStart: %+v\n", cmd))
				if a.sessionID == "" {
					a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Error: sessionID is empty\n"))
					a.errChan <- fmt.Errorf("no active session")
					continue
				}
				result, err := a.sessionManager.SessionRun(a.sessionID, cmd)
				if err != nil {
					a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Error processing command: %v\n", err))
					a.errChan <- err
				} else {
					a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Command processed successfully\n"))
					a.resultChan <- result
				}
			case <-a.stopChan:
				a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Stopping command processing goroutine\n"))
				return
			}
		}
	}()
	a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Exiting AdapterStart method\n"))
	return nil
}

// AdapterStop Stop signals the CLI adapter to stop
func (a *CLIAdapter) AdapterStop() error {
	if a.stopChan != nil {
		close(a.stopChan)
	}
	return nil
}

// GetType returns the type of the adapter
func (a *CLIAdapter) GetType() string {
	return "CLI"
}

// CommandProcess processes a command and returns the result
func (a *CLIAdapter) CommandProcess(cmd model.Command) (interface{}, error) {
	a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Entering CommandProcess method"))
	// Expand short commands to full commands
	cmd.Scope, cmd.Operation = a.expandCommand(cmd.Scope, cmd.Operation)

	// Validate the command
	sessionCmd := session.NewSessionCommand(cmd)
	if err := sessionCmd.Validate(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Send command to the channel
	a.cmdChan <- cmd

	a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: CommandProcess method sent the command to the channel\n"))

	// Wait for result or error
	select {
	case result := <-a.resultChan:
		a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Received result: %v\n", result))
		return result, nil
	case err := <-a.errChan:
		a.logger.LogInfo(context.Background(), fmt.Sprintf("DEBUG: Received error: %v\n", err))
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
