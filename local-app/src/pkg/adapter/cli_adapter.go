// Package adapter provides an adapter for the CLI interface to interact with the session package.
package adapter

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// CLIAdapter provides command-line interface support for managing multiple CLI connections
type CLIAdapter struct {
	connections     map[string]*Connection
	connectionMutex sync.RWMutex
	adapterManager  *AdapterManager
	logger          *log.Logger
}

// Connection represents a single CLI connection
type Connection struct {
	ID     string
	CmdIn  chan model.Command
	ResOut chan interface{}
}

// NewCLIAdapter creates a new instance of CLIAdapter using the provided SessionManager
func NewCLIAdapter(am *AdapterManager, logger *log.Logger) (*CLIAdapter, error) {
	logger.Info(context.Background(), "Creating new CLI adapter", nil)
	return &CLIAdapter{
		connections:    make(map[string]*Connection),
		adapterManager: am,
		logger:         logger,
	}, nil
}

// AdapterStart AdapterRun Run starts the CLI adapter's main loop
func (a *CLIAdapter) AdapterStart() error {
	a.logger.Info(context.Background(), "CLI adapter started", nil)
	return nil
}

// AdapterStop Stop signals the CLI adapter to stop
func (a *CLIAdapter) AdapterStop() error {
	a.logger.Info(context.Background(), "CLI adapter stopping", nil)
	a.connectionMutex.Lock()
	defer a.connectionMutex.Unlock()
	for _, conn := range a.connections {
		close(conn.CmdIn)
		close(conn.ResOut)
	}
	a.connections = make(map[string]*Connection)
	a.logger.Info(context.Background(), "CLI adapter stopped", nil)
	return nil
}

// GetType returns the type of the adapter
func (a *CLIAdapter) GetType() string {
	return AdapterTypeCLI
}

// ConnectionAdd creates a new connection and adds it to the list of connections
func (a *CLIAdapter) ConnectionAdd() *Connection {
	id := a.generateUniqueID()
	conn := &Connection{
		ID:     id,
		CmdIn:  make(chan model.Command),
		ResOut: make(chan interface{}),
	}
	a.connectionMutex.Lock()
	a.connections[id] = conn
	a.connectionMutex.Unlock()
	a.logger.Info(context.Background(), "New CLI connection created", log.Fields{"connectionID": id})
	return conn
}

// ConnectionDelete removes a CLI connection
func (a *CLIAdapter) ConnectionDelete(id string) {
	a.connectionMutex.Lock()
	defer a.connectionMutex.Unlock()
	if conn, exists := a.connections[id]; exists {
		close(conn.CmdIn)
		close(conn.ResOut)
		delete(a.connections, id)
		a.logger.Info(context.Background(), "CLI connection removed", log.Fields{"connectionID": id})
	}
}

// ProcessInput converts the input string into command and runs it
func (a *CLIAdapter) ProcessInput(connID string, input string) (interface{}, error) {
	cmd, err := a.parseCommand(input)
	if err != nil {
		return nil, err
	}
	return a.adapterManager.CommandRun(connID, cmd)
}

func (a *CLIAdapter) parseCommand(input string) (model.Command, error) {
	args := strings.Fields(input)
	if len(args) == 0 {
		a.logger.Info(context.Background(), "Empty command", nil)
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

	a.logger.Info(context.Background(), "Command parsed", log.Fields{"command": cmd})
	return cmd, nil
}

// CommandRun runs a command through adapter manager and returns the result
func (a *CLIAdapter) CommandRun(connID string, cmd model.Command) (interface{}, error) {
	a.logger.Info(context.Background(), "Processing command through CLI adapter", log.Fields{"command": cmd})
	return a.adapterManager.CommandRun(connID, cmd)
}

// PromptGet gets the current prompt of the session
func (a *CLIAdapter) PromptGet() string {
	return "> "
}

// generateUniqueID generates a unique ID for a new connection
// This is a placeholder implementation and should be replaced with a proper unique ID generator
func (a *CLIAdapter) generateUniqueID() string {
	return fmt.Sprintf("cli-%d", len(a.connections)+1)
}
