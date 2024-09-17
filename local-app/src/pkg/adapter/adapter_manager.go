package adapter

import (
	"context"
	"fmt"
	"sync"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
	"mindnoscape/local-app/src/pkg/session"
)

// DataOperations defines the interface for mindmap-related operations
type DataOperations interface {
	// TODO: add user/subtree export functionalities and generalize
	MindmapExport(filename, format string) error
	MindmapImport(filename, format string) (*model.Mindmap, error)
}

// AdapterInstance represents an instance of an adapter
type AdapterInstance interface {
	// CommandProcess Command HandleCommand processes a command and returns the result
	CommandProcess(cmd model.Command) (interface{}, error)

	// AdapterStart AdapterRun Run starts the adapter instance
	AdapterStart() error

	// AdapterStop Stop terminates the adapter instance
	AdapterStop() error

	// GetType returns the type of the adapter
	GetType() string
}

// AdapterFactory creates new instances of adapters
type AdapterFactory func() (AdapterInstance, error)

// AdapterManager manages all adapter instances
type AdapterManager struct {
	factories      map[string]AdapterFactory
	instances      sync.Map // map[string]AdapterInstance
	sessionManager *session.SessionManager
	cmdChan        chan commandRequest
	stopChan       chan struct{}
	logger         *log.Logger
}

// commandRequest represents a request to execute a command within a specific session and carries a result channel
type commandRequest struct {
	SessionID  string
	Command    model.Command
	ResultChan chan interface{}
}

// NewAdapterManager creates a new AdapterManager
func NewAdapterManager(sm *session.SessionManager, logger *log.Logger) *AdapterManager {
	am := &AdapterManager{
		factories:      make(map[string]AdapterFactory),
		sessionManager: sm,
		cmdChan:        make(chan commandRequest),
		stopChan:       make(chan struct{}),
		logger:         logger,
	}
	go am.commandHandler()
	am.logger.Info(context.Background(), "AdapterManager initialized", nil)
	return am
}

// AdapterAdd creates a new adapter instance
func (am *AdapterManager) AdapterAdd(adapterType string) (string, error) {
	// Check if a factory for the specified adapter type exists
	factory, ok := am.factories[adapterType]
	if !ok {
		am.logger.Error(context.Background(), "Unknown adapter type", log.Fields{"adapterType": adapterType})
		return "", fmt.Errorf("unknown adapter type: %s", adapterType)
	}

	// Create a new instance of the adapter using the factory
	instance, err := factory()
	if err != nil {
		am.logger.Error(context.Background(), "Failed to create adapter instance", log.Fields{"adapterType": adapterType, "error": err})
		return "", err
	}

	// Create a new session for this adapter instance
	sessionID, err := am.sessionManager.SessionAdd()
	if err != nil {
		am.logger.Error(context.Background(), "Failed to add session", log.Fields{"error": err})
		return "", fmt.Errorf("failed to add session: %w", err)
	}

	// Store the adapter instance with its associated session ID
	am.instances.Store(sessionID, instance)

	// Start a goroutine to handle the lifecycle of this adapter instance
	go am.instanceHandler(sessionID)

	// Return the session ID associated with the new adapter instance
	am.logger.Info(context.Background(), "Adapter instance added", log.Fields{"adapterType": adapterType, "sessionID": sessionID})
	return sessionID, nil
}

// CommandRun runs a command on a specific adapter instance
func (am *AdapterManager) CommandRun(sessionID string, cmd model.Command) (interface{}, error) {
	resultChan := make(chan interface{})
	am.cmdChan <- commandRequest{SessionID: sessionID, Command: cmd, ResultChan: resultChan}
	result := <-resultChan
	if err, ok := result.(error); ok {
		am.logger.Error(context.Background(), "Command execution failed", log.Fields{"sessionID": sessionID, "command": cmd, "error": err})
		return nil, err
	}
	am.logger.Info(context.Background(), "Command executed successfully", log.Fields{"sessionID": sessionID, "command": cmd})
	return result, nil
}

// Shutdown Stop stops all adapter instances and the command handler
func (am *AdapterManager) Shutdown() {
	close(am.stopChan)
	am.instances.Range(func(key, value interface{}) bool {
		instance := value.(AdapterInstance)
		instance.AdapterStop()
		return true
	})
	am.logger.Info(context.Background(), "AdapterManager shut down", nil)
}

func (am *AdapterManager) commandHandler() {
	for {
		select {
		case req := <-am.cmdChan:
			instance, ok := am.instances.Load(req.SessionID)
			if !ok {
				am.logger.Error(context.Background(), "No adapter instance found for session", log.Fields{"sessionID": req.SessionID})
				req.ResultChan <- fmt.Errorf("no adapter instance found for session: %s", req.SessionID)
				continue
			}
			// Use the CommandProcess method of the AdapterInstance
			result, err := instance.(AdapterInstance).CommandProcess(req.Command)
			if err != nil {
				am.logger.Error(context.Background(), "Command processing failed", log.Fields{"sessionID": req.SessionID, "command": req.Command, "error": err})
				req.ResultChan <- err
			} else {
				am.logger.Info(context.Background(), "Command processed successfully", log.Fields{"sessionID": req.SessionID, "command": req.Command})
				req.ResultChan <- result
			}
		case <-am.stopChan:
			return
		}
	}
}

func (am *AdapterManager) instanceHandler(sessionID string) {
	instance, _ := am.instances.Load(sessionID)
	adapterInstance := instance.(AdapterInstance)

	defer func() {
		adapterInstance.AdapterStop()
		am.instances.Delete(sessionID)
		am.sessionManager.SessionDelete(sessionID)
		am.logger.Info(context.Background(), "Adapter instance handler stopped", log.Fields{"sessionID": sessionID})
	}()

	am.logger.Info(context.Background(), "Adapter instance handler started", log.Fields{"sessionID": sessionID})
}
