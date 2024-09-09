// Package log provides functionality for logging commands and errors
package log

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Logger represents a logging instance that can write to command and error log files.
type Logger struct {
	commandLogger *slog.Logger
	errorLogger   *slog.Logger
	commandFile   *os.File
	errorFile     *os.File
	mu            sync.Mutex
}

// NewLogger creates a new Logger instance with specified log folder and file names.
func NewLogger(logFolder, commandLogName, errorLogName string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open command log file
	commandFilePath := filepath.Join(logFolder, commandLogName)
	commandFile, err := os.OpenFile(commandFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open command log file: %w", err)
	}

	// Open error log file
	errorFilePath := filepath.Join(logFolder, errorLogName)
	errorFile, err := os.OpenFile(errorFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		commandFile.Close()
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}

	// Create slog loggers
	commandLogger := slog.New(slog.NewJSONHandler(commandFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	errorLogger := slog.New(slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	return &Logger{
		commandLogger: commandLogger,
		errorLogger:   errorLogger,
		commandFile:   commandFile,
		errorFile:     errorFile,
	}, nil
}

// LogCommand logs a command to the command log file.
func (l *Logger) LogCommand(command string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.commandLogger.Info(command)
	return nil
}

// LogError logs an error to the error log file.
func (l *Logger) LogError(err error) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.errorLogger.Error(err.Error())
	return nil
}

// Close closes both the command and error log files.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.commandFile.Close(); err != nil {
		return fmt.Errorf("failed to close command log file: %w", err)
	}

	if err := l.errorFile.Close(); err != nil {
		return fmt.Errorf("failed to close error log file: %w", err)
	}

	return nil
}
