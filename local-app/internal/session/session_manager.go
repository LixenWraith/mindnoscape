package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"mindnoscape/local-app/internal/data"
	"mindnoscape/local-app/internal/model"
)

const (
	sessionIDLength        = 32
	defaultCleanupInterval = 5 * time.Minute
	defaultSessionTimeout  = 30 * time.Minute
)

// SessionManager manages multiple concurrent sessions
type SessionManager struct {
	sessions      map[string]*Session
	dataManager   *data.Manager
	cleanupTicker *time.Ticker
	done          chan bool
	commandQueue  chan commandExecution
}

// commandExecution represents a command to be executed in a session, its result and error
type commandExecution struct {
	session *Session
	command model.Command
	result  chan interface{}
	err     chan error
}

// NewSessionManager starts the command execution goroutine
func NewSessionManager(dataManager *data.Manager) *SessionManager {
	sm := &SessionManager{
		sessions:     make(map[string]*Session),
		dataManager:  dataManager,
		done:         make(chan bool),
		commandQueue: make(chan commandExecution),
	}
	sm.startCleanupRoutine()
	go sm.commandExecutor()
	return sm
}

// SessionAdd creates a new session and returns its ID
func (sm *SessionManager) SessionAdd() (string, error) {
	sessionID, _ := generateSessionID()
	sm.sessions[sessionID] = NewSession(sessionID, sm.dataManager)
	return sessionID, nil
}

// SessionGet retrieves a session by its ID
func (sm *SessionManager) SessionGet(sessionID string) (*Session, bool) {
	session, exists := sm.sessions[sessionID]
	return session, exists
}

// SessionDelete removes a session
func (sm *SessionManager) SessionDelete(sessionID string) {
	delete(sm.sessions, sessionID)
}

// SessionRun executes a command for a specific session
func (sm *SessionManager) SessionRun(sessionID string, cmd model.Command) (interface{}, error) {
	session, exists := sm.SessionGet(sessionID)
	if !exists {
		return nil, errors.New("session not found")
	}

	// Print current user and mindmap information
	currentUser, _ := session.UserGet()
	currentMindmap, _ := session.MindmapGet()

	userName := "None"
	if currentUser != nil {
		userName = currentUser.Username
	}

	mindmapName := "None"
	if currentMindmap != nil {
		mindmapName = currentMindmap.Name
	}

	fmt.Printf("DEBUG: Current Session State - User: %s, Mindmap: %s\n", userName, mindmapName)

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
		return res, nil
	case e := <-err:
		return nil, e
	}
}

// commandExecutor processes commands from the queue
func (sm *SessionManager) commandExecutor() {
	for cmd := range sm.commandQueue {
		result, err := cmd.session.CommandRun(cmd.command)
		if err != nil {
			cmd.err <- err
		} else {
			cmd.result <- result
		}
	}
}

// startCleanupRoutine starts a goroutine that periodically cleans up inactive sessions
func (sm *SessionManager) startCleanupRoutine() {
	sm.cleanupTicker = time.NewTicker(defaultCleanupInterval)
	go func() {
		for {
			select {
			case <-sm.cleanupTicker.C:
				sm.cleanupInactiveSessions()
			case <-sm.done:
				sm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// StopCleanupRoutine stops the cleanup routine
func (sm *SessionManager) StopCleanupRoutine() {
	sm.done <- true
}

// cleanupInactiveSessions removes inactive sessions
func (sm *SessionManager) cleanupInactiveSessions() {
	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.LastActivity) > defaultSessionTimeout {
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
