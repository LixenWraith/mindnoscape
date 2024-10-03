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
	sessions       map[string]*model.Session
	sessionMutex   sync.RWMutex
	adapterManager *AdapterManager
	logger         *log.Logger
}

// NewCLIAdapter creates a new instance of CLIAdapter using the provided SessionManager
func NewCLIAdapter(am *AdapterManager, logger *log.Logger) (*CLIAdapter, error) {
	logger.Info(context.Background(), "Creating new CLI adapter", nil)
	return &CLIAdapter{
		sessions:       make(map[string]*model.Session),
		adapterManager: am,
		logger:         logger,
	}, nil
}

// AdapterStop Stop signals the CLI adapter to stop
func (a *CLIAdapter) AdapterStop() error {
	ctx := context.Background()
	a.logger.Info(ctx, "CLI adapter stopping", nil)

	a.sessionMutex.Lock()
	for sessionID := range a.sessions {
		delete(a.sessions, sessionID)
		a.logger.Debug(ctx, "Removed session during adapter stop", log.Fields{"sessionID": sessionID})
	}
	a.sessionMutex.Unlock()

	a.logger.Info(ctx, "CLI adapter stopped", nil)
	return nil
}

// SessionAdd adds a new cli session
func (a *CLIAdapter) SessionAdd() (string, error) {
	sessionID, err := a.adapterManager.SessionAdd()
	if err != nil {
		return "", err
	}

	session, exists := a.adapterManager.SessionGet(sessionID)
	if !exists {
		a.logger.Error(context.Background(), "Session does not exist", log.Fields{"sessionID": sessionID})
		return "", fmt.Errorf("session %s does not exist after addition by cli adapter", sessionID)
	}

	a.sessionMutex.Lock()
	a.sessions[sessionID] = session
	a.sessionMutex.Unlock()
	a.logger.Info(context.Background(), "New CLI session added", log.Fields{"sessionID": sessionID})

	return sessionID, nil
}

// SessionDelete deletes a cli session
func (a *CLIAdapter) SessionDelete(sessionID string) {
	a.sessionMutex.Lock()
	delete(a.sessions, sessionID)
	a.sessionMutex.Unlock()
	a.logger.Info(context.Background(), "CLI session removed", log.Fields{"sessionID": sessionID})
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

// PromptGet gets the current prompt of the session
func (a *CLIAdapter) PromptGet(sessionID string) string {
	a.sessionMutex.RLock()
	defer a.sessionMutex.RUnlock()

	session, exists := a.sessions[sessionID]
	if !exists {
		a.logger.Warn(context.Background(), "Session not found", log.Fields{"sessionID": sessionID})
		return "> "
	}

	if session.User == nil {
		a.logger.Warn(context.Background(), "No user selected", log.Fields{"sessionID": sessionID})
		return "> "
	}

	if session.Mindmap == nil {
		a.logger.Warn(context.Background(), "No mindmap selected", log.Fields{"sessionID": sessionID})
		return fmt.Sprintf("%s > ", session.User.Username)
	}

	return fmt.Sprintf("%s @ %s > ", session.User.Username, session.Mindmap.Name)
}
