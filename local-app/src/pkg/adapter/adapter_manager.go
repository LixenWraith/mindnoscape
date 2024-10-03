package adapter

import (
	"context"
	"fmt"
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

// AdapterManager manages all adapter instances
type AdapterManager struct {
	CLIAdapter *CLIAdapter
	//	WebAdapter     *WebAdapter // Placeholder for future implementation
	//	APIAdapter     *APIAdapter // Placeholder for future implementation
	adapterMutex    sync.RWMutex
	sessionManager  *session.SessionManager
	adapterSessions sync.Map
	cmdChan         chan commandRequest
	stopChan        chan struct{}
	logger          *log.Logger
}

// commandRequest represents a request to execute a command within a specific session and carries a result channel
type commandRequest struct {
	//	AdapterType string
	SessionID  string
	Command    model.Command
	ResultChan chan commandResult
}

type commandResult struct {
	Result interface{}
	Error  error
}

// NewAdapterManager creates a new AdapterManager
func NewAdapterManager(sm *session.SessionManager, logger *log.Logger) (*AdapterManager, error) {
	am := &AdapterManager{
		sessionManager: sm,
		cmdChan:        make(chan commandRequest),
		stopChan:       make(chan struct{}),
		logger:         logger,
	}

	// Initialize CLI adapter
	cliAdapter, err := NewCLIAdapter(am, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CLI adapter: %v", err)
	}
	am.CLIAdapter = cliAdapter

	// Initialize other adapters here as needed
	// am.WebAdapter = NewWebAdapter(am, logger)
	// am.APIAdapter = NewAPIAdapter(am, logger)

	go am.commandHandler()

	am.logger.Info(context.Background(), "AdapterManager initialized", nil)
	return am, nil
}

// GetCLIAdapter returns the CLIAdapter
func (am *AdapterManager) GetCLIAdapter() *CLIAdapter {
	return am.CLIAdapter
}

// SessionAdd adds a new session based on typed adapter request
func (am *AdapterManager) SessionAdd() (string, error) {
	ctx := context.Background()
	am.logger.Debug(ctx, "Creating new session through AdapterManager", nil)

	sess, err := am.sessionManager.SessionAdd()
	if err != nil {
		am.logger.Error(ctx, "Failed to create new session", log.Fields{"error": err})
		return "", fmt.Errorf("failed to create new session: %w", err)
	}

	sessionID := sess.ID
	am.adapterSessions.Store(sessionID, sess)

	am.logger.Info(ctx, "New session created", log.Fields{"sessionID": sessionID})
	return sessionID, nil
}

// SessionGet retrieves a session by its ID
func (am *AdapterManager) SessionGet(sessionID string) (*model.Session, bool) {
	return am.sessionManager.SessionGet(sessionID)
}

// CommandRun runs a command on a specific adapter instance
func (am *AdapterManager) CommandRun(sessionID string, cmd model.Command) (interface{}, error) {
	am.logger.Info(context.Background(), "Processing command through adapter manager", log.Fields{"sessionID": sessionID, "command": cmd})

	// Log command in command log
	am.logger.Command(context.Background(), "Command received", log.Fields{
		"sessionID": sessionID,
		"scope":     cmd.Scope,
		"operation": cmd.Operation,
		"args":      cmd.Args,
	})

	resultChan := make(chan commandResult)
	am.cmdChan <- commandRequest{
		SessionID:  sessionID,
		Command:    cmd,
		ResultChan: resultChan,
	}

	result := <-resultChan
	if result.Error != nil {
		am.logger.Error(context.Background(), "Command execution failed", log.Fields{"sessionID": sessionID, "command": cmd, "error": result.Error})
		return nil, result.Error
	}

	am.logger.Info(context.Background(), "Command executed successfully", log.Fields{"sessionID": sessionID, "command": cmd})
	return result.Result, nil
}

// Shutdown stops all adapter instances and the command handler
func (am *AdapterManager) Shutdown() {
	close(am.stopChan)

	if am.CLIAdapter != nil {
		am.CLIAdapter.AdapterStop()
	}
	// Stop other adapters when implemented
	// if am.WebAdapter != nil {
	//     am.WebAdapter.AdapterStop()
	// }
	// if am.APIAdapter != nil {
	//     am.APIAdapter.AdapterStop()
	// }

	am.logger.Info(context.Background(), "AdapterManager shut down", nil)
}

func (am *AdapterManager) commandHandler() {
	for {
		select {
		case req := <-am.cmdChan:
			result, err := am.sessionManager.SessionRun(req.SessionID, req.Command)
			req.ResultChan <- commandResult{Result: result, Error: err}
		case <-am.stopChan:
			return
		}
	}
}
