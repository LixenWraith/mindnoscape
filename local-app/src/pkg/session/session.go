package session

import (
	"context"
	"errors"
	"time"

	"mindnoscape/local-app/src/pkg/data"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// CommandHandler is a function type for command handlers
type CommandHandler func(*Session, model.Command) (interface{}, error)

// Session represents an individual user session
type Session struct {
	ID              string
	DataManager     *data.DataManager
	User            *model.User
	Mindmap         *model.Mindmap
	LastActivity    time.Time
	commandHandlers map[string]map[string]CommandHandler
	logger          *log.Logger
}

// NewSession creates a new Session instance
func NewSession(id string, dataManager *data.DataManager, logger *log.Logger) *Session {
	ctx := context.Background()
	logger.Info(ctx, "Creating new Session", log.Fields{"sessionID": id})

	s := &Session{
		ID:           id,
		DataManager:  dataManager,
		LastActivity: time.Now(),
		logger:       logger,
	}
	s.initCommandHandlers()

	logger.Info(ctx, "New Session created successfully", log.Fields{"sessionID": id})
	return s
}

// initCommandHandlers initializes the command handlers map
func (s *Session) initCommandHandlers() {
	ctx := context.Background()
	s.logger.Debug(ctx, "Initializing command handlers", nil)

	s.commandHandlers = map[string]map[string]CommandHandler{
		"user":    initUserCommandHandlers(),
		"mindmap": initMindmapCommandHandlers(),
		"node":    initNodeCommandHandlers(),
		"system":  initSystemCommandHandlers(),
	}

	s.logger.Debug(ctx, "Command handlers initialized", nil)
}

// CommandRun executes a command within the session context
func (s *Session) CommandRun(cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Running command", log.Fields{"command": cmd})

	// Update last activity
	s.logger.Debug(ctx, "Updating last activity timestamp", nil)
	s.LastActivity = time.Now()

	scopeHandlers, ok := s.commandHandlers[cmd.Scope]
	if !ok {
		s.logger.Error(ctx, "Invalid command scope", log.Fields{"scope": cmd.Scope})
		return nil, errors.New("invalid command scope")
	}

	handler, ok := scopeHandlers[cmd.Operation]
	if !ok {
		s.logger.Error(ctx, "Invalid command operation", log.Fields{"operation": cmd.Operation})
		return nil, errors.New("invalid command operation")
	}

	result, err := handler(s, cmd)
	if err != nil {
		s.logger.Error(ctx, "Command execution failed", log.Fields{"error": err})
	} else {
		s.logger.Info(ctx, "Command executed successfully", nil)
	}

	return result, err
}

// UserGet retrieves the current user
func (s *Session) UserGet() (*model.User, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Retrieving current user", nil)

	if s.User == nil {
		s.logger.Warn(ctx, "No user selected", nil)
		return nil, errors.New("no user selected")
	}

	s.logger.Debug(ctx, "Current user retrieved", log.Fields{"username": s.User.Username})
	return s.User, nil
}

// UserSet sets the current user
func (s *Session) UserSet(user *model.User) {
	ctx := context.Background()
	s.logger.Info(ctx, "Setting current user", log.Fields{"username": user.Username})
	s.User = user
}

// MindmapGet retrieves the current mindmap
func (s *Session) MindmapGet() (*model.Mindmap, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Retrieving current mindmap", nil)

	if s.Mindmap == nil {
		s.logger.Warn(ctx, "No mindmap selected", nil)
		return nil, errors.New("no mindmap selected")
	}

	s.logger.Debug(ctx, "Current mindmap retrieved", log.Fields{"mindmapID": s.Mindmap.ID})
	return s.Mindmap, nil
}

// MindmapSet sets the current mindmap
func (s *Session) MindmapSet(mindmap *model.Mindmap) {
	ctx := context.Background()
	if mindmap != nil {
		s.logger.Info(ctx, "Setting current mindmap", log.Fields{"mindmapID": mindmap.ID})
	} else {
		s.logger.Info(ctx, "Clearing current mindmap", nil)
	}
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
