package session

import (
	"errors"
	"time"

	"mindnoscape/local-app/pkg/data"
	"mindnoscape/local-app/pkg/model"
)

// CommandHandler is a function type for command handlers
type CommandHandler func(*Session, model.Command) (interface{}, error)

// Session represents an individual user session
type Session struct {
	ID              string
	DataManager     *data.Manager
	User            *model.User
	Mindmap         *model.Mindmap
	LastActivity    time.Time
	commandHandlers map[string]map[string]CommandHandler
}

// NewSession creates a new Session instance
func NewSession(id string, dataManager *data.Manager) *Session {
	s := &Session{
		ID:           id,
		DataManager:  dataManager,
		LastActivity: time.Now(),
	}
	s.initCommandHandlers()
	return s
}

// initCommandHandlers initializes the command handlers map
func (s *Session) initCommandHandlers() {
	s.commandHandlers = map[string]map[string]CommandHandler{
		"user":    initUserCommandHandlers(),
		"mindmap": initMindmapCommandHandlers(),
		"node":    initNodeCommandHandlers(),
		"system":  initSystemCommandHandlers(),
	}
}

// CommandRun executes a command within the session context
func (s *Session) CommandRun(cmd model.Command) (interface{}, error) {
	s.updateLastActivity()

	scopeHandlers, ok := s.commandHandlers[cmd.Scope]
	if !ok {
		return nil, errors.New("invalid command scope")
	}

	handler, ok := scopeHandlers[cmd.Operation]
	if !ok {
		return nil, errors.New("invalid command operation")
	}

	return handler(s, cmd)
}

// updateLastActivity updates the LastActivity timestamp
func (s *Session) updateLastActivity() {
	s.LastActivity = time.Now()
}

// UserGet retrieves the current user
func (s *Session) UserGet() (*model.User, error) {
	if s.User == nil {
		return nil, errors.New("no user selected")
	}
	return s.User, nil
}

// UserSet sets the current user
func (s *Session) UserSet(user *model.User) {
	s.User = user
}

// MindmapGet retrieves the current mindmap
func (s *Session) MindmapGet() (*model.Mindmap, error) {
	if s.Mindmap == nil {
		return nil, errors.New("no mindmap selected")
	}
	return s.Mindmap, nil
}

// MindmapSet sets the current mindmap
func (s *Session) MindmapSet(mindmap *model.Mindmap) {
	s.Mindmap = mindmap
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
		"exit": handleSystemExit,
		"quit": handleSystemExit,
	}
}
