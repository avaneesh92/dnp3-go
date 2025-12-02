package logger

import (
	"fmt"
	"log"
	"os"
)

// Level represents logging level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns string representation of Level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is the interface for logging
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	SetLevel(level Level)
}

// DefaultLogger is a simple logger implementation
type DefaultLogger struct {
	level  Level
	logger *log.Logger
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(level Level) *DefaultLogger {
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Debug logs debug message
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	if l.level <= LevelDebug {
		l.logger.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs info message
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	if l.level <= LevelInfo {
		l.logger.Printf("[INFO] "+format, args...)
	}
}

// Warn logs warning message
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		l.logger.Printf("[WARN] "+format, args...)
	}
}

// Error logs error message
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	if l.level <= LevelError {
		l.logger.Printf("[ERROR] "+format, args...)
	}
}

// SetLevel sets the logging level
func (l *DefaultLogger) SetLevel(level Level) {
	l.level = level
}

// NoOpLogger is a logger that doesn't log anything
type NoOpLogger struct{}

// NewNoOpLogger creates a logger that doesn't log
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// Debug does nothing
func (l *NoOpLogger) Debug(format string, args ...interface{}) {}

// Info does nothing
func (l *NoOpLogger) Info(format string, args ...interface{}) {}

// Warn does nothing
func (l *NoOpLogger) Warn(format string, args ...interface{}) {}

// Error does nothing
func (l *NoOpLogger) Error(format string, args ...interface{}) {}

// SetLevel does nothing
func (l *NoOpLogger) SetLevel(level Level) {}

// Global default logger
var defaultLogger Logger = NewDefaultLogger(LevelInfo)

// SetDefault sets the default logger
func SetDefault(logger Logger) {
	defaultLogger = logger
}

// GetDefault returns the default logger
func GetDefault() Logger {
	return defaultLogger
}

// Helper functions using default logger

// Debug logs debug message using default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs info message using default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs warning message using default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs error message using default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Logf is a generic logging function
func Logf(level Level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	switch level {
	case LevelDebug:
		defaultLogger.Debug(msg)
	case LevelInfo:
		defaultLogger.Info(msg)
	case LevelWarn:
		defaultLogger.Warn(msg)
	case LevelError:
		defaultLogger.Error(msg)
	}
}
