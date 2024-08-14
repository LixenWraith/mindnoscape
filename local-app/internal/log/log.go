package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	commandFile *os.File
	errorFile   *os.File
	mu          sync.Mutex
}

func NewLogger(logFolder, commandLogName, errorLogName string) (*Logger, error) {
	if err := os.MkdirAll(logFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	commandFilePath := filepath.Join(logFolder, commandLogName)
	errorFilePath := filepath.Join(logFolder, errorLogName)

	commandFile, err := os.OpenFile(commandFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open command log file: %w", err)
	}

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

func (l *Logger) LogCommand(command string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, command)

	_, err := l.commandFile.WriteString(logEntry)
	if err != nil {
		return fmt.Errorf("failed to write command log: %w", err)
	}

	return nil
}

func (l *Logger) LogError(err error) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, err.Error())

	_, writeErr := l.errorFile.WriteString(logEntry)
	if writeErr != nil {
		return fmt.Errorf("failed to write error log: %w", writeErr)
	}

	return nil
}

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
