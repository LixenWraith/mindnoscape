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

// SessionManager manages multiple concurrent sessions
type SessionManager struct {
	sessions      map[string]*Session
	dataManager   *data.DataManager
	cleanupTicker *time.Ticker
	done          chan bool
	commandQueue  chan commandExecution
	logger        *log.Logger
}

// commandExecution represents a command to be executed in a session, its result and error
type commandExecution struct {
	session *Session
	command model.Command
	result  chan interface{}
	err     chan error
}

// NewSessionManager starts the command execution goroutine
func NewSessionManager(dataManager *data.DataManager, logger *log.Logger) *SessionManager {
	ctx := context.Background()
	logger.Info(ctx, "Creating new SessionManager", nil)

	sm := &SessionManager{
		sessions:     make(map[string]*Session),
		dataManager:  dataManager,
		done:         make(chan bool),
		commandQueue: make(chan commandExecution),
		logger:       logger,
	}
	sm.startCleanupRoutine()
	go sm.commandExecutor()

	logger.Info(ctx, "SessionManager created successfully", nil)
	return sm
}

// SessionAdd creates a new session and returns its ID
func (sm *SessionManager) SessionAdd() (string, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Adding new session", nil)

	sessionID, err := generateSessionID()
	if err != nil {
		sm.logger.Error(ctx, "Failed to generate session ID", log.Fields{"error": err})
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	sm.sessions[sessionID] = NewSession(sessionID, sm.dataManager, sm.logger)
	sm.logger.Info(ctx, "New session added", log.Fields{"sessionID": sessionID})
	return sessionID, nil
}

// SessionGet retrieves a session by its ID
func (sm *SessionManager) SessionGet(sessionID string) (*Session, bool) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Retrieving session", log.Fields{"sessionID": sessionID})

	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.logger.Warn(ctx, "Session not found", log.Fields{"sessionID": sessionID})
	} else {
		sm.logger.Debug(ctx, "Session retrieved", log.Fields{"sessionID": sessionID})
	}
	return session, exists
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

	session, exists := sm.SessionGet(sessionID)
	if !exists {
		sm.logger.Error(ctx, "Session not found", log.Fields{"sessionID": sessionID})
		return nil, errors.New("session not found")
	}

	// Log command in command log
	sm.logger.Command(ctx, "Command received", log.Fields{
		"sessionID": sessionID,
		"scope":     cmd.Scope,
		"operation": cmd.Operation,
		"args":      cmd.Args,
	})

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

// commandExecutor processes commands from the queue
func (sm *SessionManager) commandExecutor() {
	ctx := context.Background()
	sm.logger.Info(ctx, "Starting command executor", nil)

	for cmd := range sm.commandQueue {
		sm.logger.Debug(ctx, "Processing command", log.Fields{"sessionID": cmd.session.ID, "command": cmd.command})
		result, err := cmd.session.CommandRun(cmd.command)
		if err != nil {
			sm.logger.Error(ctx, "Command execution failed", log.Fields{"sessionID": cmd.session.ID, "error": err})
			cmd.err <- err
		} else {
			sm.logger.Debug(ctx, "Command executed successfully", log.Fields{"sessionID": cmd.session.ID})
			cmd.result <- result
		}
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
				sm.logger.Info(ctx, "Stopping cleanup routine", nil)
				sm.cleanupTicker.Stop()
				return
			}
		}
	}()
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
