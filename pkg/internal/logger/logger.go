package logger

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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

// Global frame debugging flag
var enableFrameDebug bool

func init() {
	// Check environment variable DNP3_FRAME_DEBUG
	if val := os.Getenv("DNP3_FRAME_DEBUG"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			enableFrameDebug = enabled
		}
	}
}

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

// Frame debugging functions

// SetFrameDebug enables or disables frame debugging
func SetFrameDebug(enable bool) {
	enableFrameDebug = enable
}

// IsFrameDebugEnabled returns whether frame debugging is enabled
func IsFrameDebugEnabled() bool {
	return enableFrameDebug
}

// LogFrameReceived logs a received frame with detailed hex dump if frame debugging is enabled
func LogFrameReceived(channelID string, data []byte) {
	if !enableFrameDebug {
		return
	}

	defaultLogger.Info("<<< FRAME RECEIVED [Channel: %s] (%d bytes)", channelID, len(data))
	defaultLogger.Info("%s", formatFrameHex(data))
}

// LogFrameSent logs a sent frame with detailed hex dump if frame debugging is enabled
func LogFrameSent(channelID string, data []byte) {
	if !enableFrameDebug {
		return
	}

	defaultLogger.Info(">>> FRAME SENT [Channel: %s] (%d bytes)", channelID, len(data))
	defaultLogger.Info("%s", formatFrameHex(data))
}

// formatFrameHex formats a byte array as a detailed hex dump
// Format: offset | hex bytes (16 per line) | ASCII representation
func formatFrameHex(data []byte) string {
	if len(data) == 0 {
		return "    <empty>"
	}

	var sb strings.Builder
	const bytesPerLine = 16

	for i := 0; i < len(data); i += bytesPerLine {
		// Offset
		sb.WriteString(fmt.Sprintf("    %04X | ", i))

		// Hex bytes
		end := i + bytesPerLine
		if end > len(data) {
			end = len(data)
		}

		for j := i; j < end; j++ {
			sb.WriteString(fmt.Sprintf("%02X ", data[j]))
		}

		// Padding for incomplete lines
		for j := end; j < i+bytesPerLine; j++ {
			sb.WriteString("   ")
		}

		// ASCII representation
		sb.WriteString("| ")
		for j := i; j < end; j++ {
			if data[j] >= 32 && data[j] <= 126 {
				sb.WriteByte(data[j])
			} else {
				sb.WriteByte('.')
			}
		}

		if i+bytesPerLine < len(data) {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
