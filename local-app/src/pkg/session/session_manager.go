package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"mindnoscape/local-app/src/pkg/data"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

const (
	sessionIDLength        = 32
	defaultCleanupInterval = 5 * time.Minute
	defaultSessionTimeout  = 30 * time.Minute
)

// CommandHandler is a function type for command handlers
type CommandHandler func(*SessionManager, *model.Session, model.Command) (interface{}, error)

// SessionManager manages multiple concurrent sessions
type SessionManager struct {
	sessions        map[string]*model.Session
	dataManager     *data.DataManager
	cleanupTicker   *time.Ticker
	done            chan bool
	commandQueue    chan commandExecution
	logger          *log.Logger
	commandHandlers map[string]map[string]CommandHandler
}

// commandExecution represents a command to be executed in a session, its result and error
type commandExecution struct {
	session *model.Session
	command model.Command
	result  chan interface{}
	err     chan error
}

// NewSessionManager starts the command execution goroutine
func NewSessionManager(dataManager *data.DataManager, logger *log.Logger) *SessionManager {
	ctx := context.Background()
	logger.Info(ctx, "Creating new SessionManager", nil)

	sm := &SessionManager{
		sessions:     make(map[string]*model.Session),
		dataManager:  dataManager,
		done:         make(chan bool),
		commandQueue: make(chan commandExecution),
		logger:       logger,
	}
	sm.startCleanupRoutine()
	sm.initCommandHandlers()
	go sm.commandExecutor()

	logger.Info(ctx, "SessionManager created successfully", nil)
	return sm
}

// SessionAdd creates a new session and returns its ID
func (sm *SessionManager) SessionAdd() (*model.Session, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Adding new session", nil)

	sessionID, err := generateSessionID()
	if err != nil {
		sm.logger.Error(ctx, "Failed to generate session ID", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := &model.Session{
		ID:           sessionID,
		LastActivity: time.Now(),
	}
	sm.sessions[sessionID] = session
	sm.logger.Info(ctx, "New session added", log.Fields{"sessionID": sessionID})
	return session, nil
}

// SessionGet retrieves a session by its ID
func (sm *SessionManager) SessionGet(sessionID string) (*model.Session, bool) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Retrieving session", log.Fields{"sessionID": sessionID})

	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.logger.Warn(ctx, "Session not found", log.Fields{"sessionID": sessionID})
		return nil, false
	}

	sm.logger.Debug(ctx, "Session retrieved", log.Fields{"sessionID": sessionID})
	return session, true
}

// SessionDelete removes a session
func (sm *SessionManager) SessionDelete(sessionID string) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Deleting session", log.Fields{"sessionID": sessionID})

	if _, exists := sm.sessions[sessionID]; !exists {
		sm.logger.Warn(ctx, "Attempted to delete non-existent session", log.Fields{"sessionID": sessionID})
		return
	}

	delete(sm.sessions, sessionID)
	sm.logger.Info(ctx, "Session deleted", log.Fields{"sessionID": sessionID})
}

// SessionRun executes a command for a specific session
func (sm *SessionManager) SessionRun(sessionID string, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Running command in session", log.Fields{"sessionID": sessionID, "command": cmd})

	// Validate the session
	// TODO: use SessionGet
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.logger.Error(ctx, "Session not found", log.Fields{"sessionID": sessionID})
		return nil, errors.New("session not found")
	}

	// Expand the command
	cmd.Scope, cmd.Operation = sm.expandCommand(cmd.Scope, cmd.Operation)

	// Validate the command
	if err := sm.validateCommand(cmd); err != nil {
		sm.logger.Error(ctx, "Command validation failed", log.Fields{"sessionID": sessionID, "error": err})
		return nil, err
	}

	result := make(chan interface{})
	err := make(chan error)

	sm.commandQueue <- commandExecution{
		session: session,
		command: cmd,
		result:  result,
		err:     err,
	}

	select {
	case res := <-result:
		sm.logger.Info(ctx, "Command executed successfully", log.Fields{"sessionID": sessionID})
		return res, nil
	case e := <-err:
		sm.logger.Error(ctx, "Command execution failed", log.Fields{"sessionID": sessionID, "error": e})
		return nil, e
	}
}

// startCleanupRoutine starts a goroutine that periodically cleans up inactive sessions
func (sm *SessionManager) startCleanupRoutine() {
	ctx := context.Background()
	sm.logger.Info(ctx, "Starting cleanup routine", nil)

	sm.cleanupTicker = time.NewTicker(defaultCleanupInterval)
	go func() {
		for {
			select {
			case <-sm.cleanupTicker.C:
				sm.cleanupInactiveSessions()
			case <-sm.done:
				sm.logger.Info(ctx, "Stopped cleanup routine", nil)
				sm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// expandCommand converts concise (one letter) commands and operations to the long (complete string) format
func (sm *SessionManager) expandCommand(scope, operation string) (string, string) {
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
			case "h":
				expandedOperation = "help"
			}
		}
	}

	return expandedScope, expandedOperation
}

// initCommandHandlers initializes the command handlers
func (sm *SessionManager) initCommandHandlers() {
	sm.commandHandlers = map[string]map[string]CommandHandler{
		"user":    initUserCommandHandlers(),
		"mindmap": initMindmapCommandHandlers(),
		"node":    initNodeCommandHandlers(),
		"system":  initSystemCommandHandlers(),
	}
}

// initUserCommandHandlers initializes user command handlers
func initUserCommandHandlers() map[string]CommandHandler {
	return map[string]CommandHandler{
		"add":    handleUserAdd,
		"update": handleUserUpdate,
		"delete": handleUserDelete,
		"select": handleUserSelect,
	}
}

// initMindmapCommandHandlers initializes mindmap command handlers
func initMindmapCommandHandlers() map[string]CommandHandler {
	return map[string]CommandHandler{
		"add":        handleMindmapAdd,
		"delete":     handleMindmapDelete,
		"permission": handleMindmapPermission,
		"import":     handleMindmapImport,
		"export":     handleMindmapExport,
		"select":     handleMindmapSelect,
		"list":       handleMindmapList,
		"view":       handleMindmapView,
	}
}

// initNodeCommandHandlers initializes node command handlers
func initNodeCommandHandlers() map[string]CommandHandler {
	return map[string]CommandHandler{
		"add":    handleNodeAdd,
		"update": handleNodeUpdate,
		"move":   handleNodeMove,
		"delete": handleNodeDelete,
		"find":   handleNodeFind,
		"sort":   handleNodeSort,
	}
}

// initSystemCommandHandlers initializes system command handlers
func initSystemCommandHandlers() map[string]CommandHandler {
	return map[string]CommandHandler{
		"help": handleSystemHelp,
		"exit": handleSystemExit,
		"quit": handleSystemExit,
	}
}

// commandExecutor processes commands from the queue
func (sm *SessionManager) commandExecutor() {
	ctx := context.Background()
	sm.logger.Info(ctx, "Starting command executor", nil)

	for cmd := range sm.commandQueue {
		sm.logger.Debug(ctx, "Processing command", log.Fields{"sessionID": cmd.session.ID, "command": cmd.command})

		scopeHandlers, ok := sm.commandHandlers[cmd.command.Scope]
		if !ok {
			cmd.err <- fmt.Errorf("invalid command scope: %s", cmd.command.Scope)
			continue
		}

		handler, ok := scopeHandlers[cmd.command.Operation]
		if !ok {
			cmd.err <- fmt.Errorf("invalid command operation: %s", cmd.command.Operation)
			continue
		}

		result, err := handler(sm, cmd.session, cmd.command)
		if err != nil {
			sm.logger.Error(ctx, "Command execution failed", log.Fields{"sessionID": cmd.session.ID, "error": err})
			cmd.err <- err
		} else {
			sm.logger.Debug(ctx, "Command executed successfully", log.Fields{"sessionID": cmd.session.ID})
			cmd.result <- result
		}
	}
}

// StopCleanupRoutine stops the cleanup routine
func (sm *SessionManager) StopCleanupRoutine() {
	ctx := context.Background()
	sm.logger.Info(ctx, "Stopping cleanup routine", nil)
	sm.done <- true
}

// cleanupInactiveSessions removes inactive sessions
func (sm *SessionManager) cleanupInactiveSessions() {
	ctx := context.Background()
	sm.logger.Debug(ctx, "Running cleanup for inactive sessions", nil)

	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.LastActivity) > defaultSessionTimeout {
			sm.logger.Info(ctx, "Removing inactive session", log.Fields{"sessionID": id})
			sm.SessionDelete(id)
		}
	}
}

// generateSessionID creates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	b := make([]byte, sessionIDLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (sm *SessionManager) validateCommand(cmd model.Command) error {
	ctx := context.Background()
	sm.logger.Info(ctx, "Validating command", log.Fields{"scope": cmd.Scope, "operation": cmd.Operation})

	if cmd.Scope == "" {
		sm.logger.Error(ctx, "Command scope is empty", nil)
		return errors.New("command scope is required")
	}
	return sm.validateScopeAndOperation(cmd)
}

func (sm *SessionManager) validateScopeAndOperation(cmd model.Command) error {
	ctx := context.Background()
	sm.logger.Debug(ctx, "Validating scope and operation", log.Fields{"scope": cmd.Scope, "operation": cmd.Operation})

	switch cmd.Scope {
	case "user":
		return sm.validateUserCommand(cmd)
	case "mindmap":
		return sm.validateMindmapCommand(cmd)
	case "node":
		return sm.validateNodeCommand(cmd)
	case "system":
		return sm.validateSystemCommand(cmd)
	default:
		sm.logger.Error(ctx, "Invalid command scope", log.Fields{"scope": cmd.Scope})
		return fmt.Errorf("invalid command scope: %s", cmd.Scope)
	}
}

func (sm *SessionManager) validateUserCommand(cmd model.Command) error {
	ctx := context.Background()
	sm.logger.Debug(ctx, "Validating user command", log.Fields{"operation": cmd.Operation})

	switch cmd.Operation {
	case "add":
		if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for user add command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("user add command requires 1 or 2 arguments: <username> [password]")
		}
	case "update":
		if len(cmd.Args) < 1 || len(cmd.Args) > 3 {
			sm.logger.Error(ctx, "Invalid number of arguments for user update command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("user update command requires 1 to 3 arguments: <username> [new_username] [new_password]")
		}
	case "delete", "select":
		if len(cmd.Args) != 1 {
			sm.logger.Error(ctx, "Invalid number of arguments for user command", log.Fields{"operation": cmd.Operation, "argCount": len(cmd.Args)})
			return fmt.Errorf("user %s command requires 1 argument: <username>", cmd.Operation)
		}
	default:
		sm.logger.Error(ctx, "Invalid user operation", log.Fields{"operation": cmd.Operation})
		return fmt.Errorf("invalid user operation: %s", cmd.Operation)
	}
	return nil
}

func (sm *SessionManager) validateMindmapCommand(cmd model.Command) error {
	ctx := context.Background()
	sm.logger.Debug(ctx, "Validating mindmap command", log.Fields{"operation": cmd.Operation})

	switch cmd.Operation {
	case "add":
		if len(cmd.Args) != 1 {
			sm.logger.Error(ctx, "Invalid number of arguments for mindmap add command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("mindmap add command requires 1 argument: <mindmap_name>")
		}
	case "delete", "select":
		if len(cmd.Args) > 1 {
			sm.logger.Error(ctx, "Invalid number of arguments for mindmap command", log.Fields{"operation": cmd.Operation, "argCount": len(cmd.Args)})
			return fmt.Errorf("mindmap %s command requires 0 or 1 argument: [mindmap_name]", cmd.Operation)
		}
	case "permission":
		if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for mindmap permission command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("mindmap permission command requires 1 or 2 arguments: <mindmap_name> [public|private]")
		}
	case "import", "export":
		if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for mindmap import/export command", log.Fields{"operation": cmd.Operation, "argCount": len(cmd.Args)})
			return fmt.Errorf("mindmap %s command requires 1 or 2 arguments: <filename> [json|xml]", cmd.Operation)
		}
	case "list":
		if len(cmd.Args) != 0 {
			sm.logger.Error(ctx, "Invalid number of arguments for mindmap list command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("mindmap list command does not accept any arguments")
		}
	case "view":
		if len(cmd.Args) > 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for mindmap view command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("mindmap view command accepts at most 2 arguments: [index] [--id]")
		}
	default:
		sm.logger.Error(ctx, "Invalid mindmap operation", log.Fields{"operation": cmd.Operation})
		return fmt.Errorf("invalid mindmap operation: %s", cmd.Operation)
	}
	return nil
}

func (sm *SessionManager) validateNodeCommand(cmd model.Command) error {
	ctx := context.Background()
	sm.logger.Debug(ctx, "Validating node command", log.Fields{"operation": cmd.Operation})

	switch cmd.Operation {
	case "add":
		if len(cmd.Args) < 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for node add command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("node add command requires at least 2 arguments: <parent> <content> [<extra field label>:<extra field value>]... [--id]")
		}
	case "update":
		if len(cmd.Args) < 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for node update command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("node update command requires at least 2 arguments: <node> <content> [<extra field label>:<extra field value>]... [--id]")
		}
	case "move":
		if len(cmd.Args) < 2 || len(cmd.Args) > 3 {
			sm.logger.Error(ctx, "Invalid number of arguments for node move command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("node move command requires 2 or 3 arguments: <source> <target> [--id]")
		}
	case "delete":
		if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for node delete command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("node delete command requires 1 or 2 arguments: <node> [--id]")
		}
	case "find":
		if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for node find command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("node find command requires 1 or 2 arguments: <query> [--id]")
		}
	case "sort":
		if len(cmd.Args) > 4 {
			sm.logger.Error(ctx, "Invalid number of arguments for node sort command", log.Fields{"argCount": len(cmd.Args)})
			return errors.New("node sort command accepts at most 4 arguments: [identifier] [field] [--reverse] [--id]")
		}
	default:
		sm.logger.Error(ctx, "Invalid node operation", log.Fields{"operation": cmd.Operation})
		return fmt.Errorf("invalid node operation: %s", cmd.Operation)
	}
	return nil
}

func (sm *SessionManager) validateSystemCommand(cmd model.Command) error {
	ctx := context.Background()
	sm.logger.Debug(ctx, "Validating system command", log.Fields{"operation": cmd.Operation})

	switch cmd.Operation {
	case "exit", "quit":
		if len(cmd.Args) != 0 {
			sm.logger.Error(ctx, "Invalid number of arguments for system command", log.Fields{"operation": cmd.Operation, "argCount": len(cmd.Args)})
			return fmt.Errorf("system %s command does not accept any arguments", cmd.Operation)
		}
	case "help":
		if len(cmd.Args) > 2 {
			sm.logger.Error(ctx, "Invalid number of arguments for system command", log.Fields{"operation": cmd.Operation, "argCount": len(cmd.Args)})
			return fmt.Errorf("system %s command does not accept any arguments", cmd.Operation)
		}
	default:
		sm.logger.Error(ctx, "Invalid system operation", log.Fields{"operation": cmd.Operation})
		return fmt.Errorf("invalid system operation: %s", cmd.Operation)
	}
	return nil
}
