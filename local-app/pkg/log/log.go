// Package log provides functionality for logging commands and errors
package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger represents a logging instance that can write to command and error log files.
type Logger struct {
	commandFile *os.File
	errorFile   *os.File
	mu          sync.Mutex
}

// NewLogger creates a new Logger instance with specified log folder and file names.
func NewLogger(logFolder, commandLogName, errorLogName string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open command log file
	commandFilePath := filepath.Join(logFolder, commandLogName)
	errorFilePath := filepath.Join(logFolder, errorLogName)

	commandFile, err := os.OpenFile(commandFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open command log file: %w", err)
	}

	// Open error log file
	errorFile, err := os.OpenFile(errorFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		commandFile.Close()
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}

	return &Logger{
		commandFile: commandFile,
		errorFile:   errorFile,
	}, nil
}

// LogCommand logs a command to the command log file.
func (l *Logger) LogCommand(command string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Format the log entry with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, command)

	// Write the log entry to the command log file
	_, err := l.commandFile.WriteString(logEntry)
	if err != nil {
		return fmt.Errorf("failed to write command log: %w", err)
	}

	return nil
}

// LogError logs an error to the error log file.
func (l *Logger) LogError(err error) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Format the log entry with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, err.Error())

	// Write the log entry to the error log file
	_, writeErr := l.errorFile.WriteString(logEntry)
	if writeErr != nil {
		return fmt.Errorf("failed to write error log: %w", writeErr)
	}

	return nil
}

// Close closes both the command and error log files.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close the command log file
	if err := l.commandFile.Close(); err != nil {
		return fmt.Errorf("failed to close command log file: %w", err)
	}

	// Close the error log file
	if err := l.errorFile.Close(); err != nil {
		return fmt.Errorf("failed to close error log file: %w", err)
	}

	return nil
}
