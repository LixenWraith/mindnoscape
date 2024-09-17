// Package log provides functionality for logging commands and errors
package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"mindnoscape/local-app/src/pkg/model"
)

// Fields represents a map of additional information for log entries
type Fields map[string]interface{}

// LogMessage represents a message to be logged
type LogMessage struct {
	Level   LogLevel
	Content string
	Fields  Fields
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
	level         LogLevel
}

// NewLogger creates a new Logger instance with specified log folder and file names
func NewLogger(cfg *model.Config, level LogLevel) (*Logger, error) {
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
	commandLogger := slog.New(slog.NewJSONHandler(commandFile, &slog.HandlerOptions{Level: LevelCommand.toSlogLevel()}))
	errorLogger := slog.New(slog.NewJSONHandler(errorFile, &slog.HandlerOptions{Level: LevelWarn.toSlogLevel()}))
	infoLogger := slog.New(slog.NewJSONHandler(infoFile, &slog.HandlerOptions{Level: LevelDebug.toSlogLevel()}))

	logger := &Logger{
		commandLogger: commandLogger,
		errorLogger:   errorLogger,
		infoLogger:    infoLogger,
		commandFile:   commandFile,
		errorFile:     errorFile,
		infoFile:      infoFile,
		logChan:       make(chan LogMessage, 100), // Buffered channel with capacity of 100
		done:          make(chan struct{}),
		level:         level,
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
			attrs := make([]slog.Attr, 0, len(msg.Fields))
			for k, v := range msg.Fields {
				attrs = append(attrs, slog.Any(k, v))
			}
			switch msg.Level {
			case LevelCommand:
				l.commandLogger.LogAttrs(msg.Context, msg.Level.toSlogLevel(), msg.Content, attrs...)
			case LevelError, LevelWarn:
				l.errorLogger.LogAttrs(msg.Context, msg.Level.toSlogLevel(), msg.Content, attrs...)
			case LevelInfo, LevelDebug:
				l.infoLogger.LogAttrs(msg.Context, msg.Level.toSlogLevel(), msg.Content, attrs...)
			}
		case <-l.done:
			return
		}
	}
}

// Command logs a command message
func (l *Logger) Command(ctx context.Context, message string, fields Fields) {
	l.logChan <- LogMessage{Level: LevelCommand, Content: message, Fields: fields, Context: ctx}
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, message string, fields Fields) {
	l.logChan <- LogMessage{Level: LevelError, Content: message, Fields: fields, Context: ctx}
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, message string, fields Fields) {
	l.logChan <- LogMessage{Level: LevelWarn, Content: message, Fields: fields, Context: ctx}
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, message string, fields Fields) {
	l.logChan <- LogMessage{Level: LevelInfo, Content: message, Fields: fields, Context: ctx}
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, message string, fields Fields) {
	l.logChan <- LogMessage{Level: LevelDebug, Content: message, Fields: fields, Context: ctx}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
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
