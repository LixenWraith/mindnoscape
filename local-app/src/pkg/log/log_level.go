// Package log provides functionality for logging commands and errors
package log

import "log/slog"

// LogLevel represents the type and severity of a log message
type LogLevel int

const (
	LevelCommand LogLevel = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
)

// String returns the string representation of the LogLevel
func (l LogLevel) String() string {
	switch l {
	case LevelCommand:
		return "COMMAND"
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// toSlogLevel converts our custom LogLevel to slog.Level
func (l LogLevel) toSlogLevel() slog.Level {
	switch l {
	case LevelCommand:
		return slog.LevelInfo
	case LevelError:
		return slog.LevelError
	case LevelWarn:
		return slog.LevelWarn
	case LevelInfo:
		return slog.LevelInfo
	case LevelDebug:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
