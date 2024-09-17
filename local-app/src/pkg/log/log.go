// Package log provides functionality for logging commands and errors
package log

import (
	"context"
	"fmt"
	"log/slog"
	"mindnoscape/local-app/src/pkg/model"
	"os"
	"path/filepath"
	"sync"
)

// LogMessage represents a message to be logged
type LogMessage struct {
	Type    string // "command", "error", or "info"
	Content string
	Context context.Context
}

// Logger represents a logging instance that can write to command, error, and info log files
type Logger struct {
	commandLogger *slog.Logger
	errorLogger   *slog.Logger
	infoLogger    *slog.Logger
	commandFile   *os.File
	errorFile     *os.File
	infoFile      *os.File
	logChan       chan LogMessage
	done          chan struct{}
	wg            sync.WaitGroup
	infoEnabled   bool // Flag to enable/disable info logging
}

// NewLogger creates a new Logger instance with specified log folder and file names
func NewLogger(cfg *model.Config, infoEnabled bool) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(cfg.LogFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open command log file
	commandFilePath := filepath.Join(cfg.LogFolder, cfg.CommandLog)
	commandFile, err := os.OpenFile(commandFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open command log file: %w", err)
	}

	// Open error log file
	errorFilePath := filepath.Join(cfg.LogFolder, cfg.ErrorLog)
	errorFile, err := os.OpenFile(errorFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		commandFile.Close()
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}

	// Open info log file
	infoFilePath := filepath.Join(cfg.LogFolder, cfg.InfoLog)
	infoFile, err := os.OpenFile(infoFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		commandFile.Close()
		errorFile.Close()
		return nil, fmt.Errorf("failed to open info log file: %w", err)
	}

	// Create slog loggers
	commandLogger := slog.New(slog.NewJSONHandler(commandFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	errorLogger := slog.New(slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	infoLogger := slog.New(slog.NewJSONHandler(infoFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger := &Logger{
		commandLogger: commandLogger,
		errorLogger:   errorLogger,
		infoLogger:    infoLogger,
		commandFile:   commandFile,
		errorFile:     errorFile,
		infoFile:      infoFile,
		logChan:       make(chan LogMessage, 100), // Buffered channel with capacity of 100
		done:          make(chan struct{}),
		infoEnabled:   infoEnabled,
	}

	// Start the logging goroutine
	logger.wg.Add(1)
	go logger.processLogs()

	return logger, nil
}

// processLogs handles incoming log messages
func (l *Logger) processLogs() {
	defer l.wg.Done()
	for {
		select {
		case msg := <-l.logChan:
			switch msg.Type {
			case "command":
				l.commandLogger.InfoContext(msg.Context, msg.Content)
			case "error":
				l.errorLogger.ErrorContext(msg.Context, msg.Content)
			case "info":
				if l.infoEnabled {
					l.infoLogger.DebugContext(msg.Context, msg.Content)
				}
			}
		case <-l.done:
			return
		}
	}
}

func (l *Logger) LogCommand(ctx context.Context, command string) {
	l.logChan <- LogMessage{Type: "command", Content: command, Context: ctx}
}

func (l *Logger) LogError(ctx context.Context, err error) {
	l.logChan <- LogMessage{Type: "error", Content: err.Error(), Context: ctx}
}

func (l *Logger) LogInfo(ctx context.Context, message string) {
	if l.infoEnabled {
		l.logChan <- LogMessage{Type: "info", Content: message, Context: ctx}
	}
}

// SetInfoEnabled enables or disables info logging
func (l *Logger) SetInfoEnabled(enabled bool) {
	l.infoEnabled = enabled
}

// Close stops the logging goroutine and closes all log files
func (l *Logger) Close() error {
	close(l.done)
	l.wg.Wait() // Wait for the logging goroutine to finish

	if err := l.commandFile.Close(); err != nil {
		return fmt.Errorf("failed to close command log file: %w", err)
	}

	if err := l.errorFile.Close(); err != nil {
		return fmt.Errorf("failed to close error log file: %w", err)
	}

	if err := l.infoFile.Close(); err != nil {
		return fmt.Errorf("failed to close info log file: %w", err)
	}

	return nil
}

// GetLogChannels returns the log channel for external use
func (l *Logger) GetLogChannels() chan<- LogMessage {
	return l.logChan
}
