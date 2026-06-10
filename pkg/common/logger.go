package common

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l LogLevel) String() string {
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

// Logger is a simple structured logger.
type Logger struct {
	mu      sync.Mutex
	level   LogLevel
	output  io.Writer
	prefix  string
}

// NewLogger creates a new logger.
func NewLogger(prefix string, level LogLevel) *Logger {
	return &Logger{level: level, output: os.Stderr, prefix: prefix}
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.output, "%s [%s] %s: %s\n", ts, level, l.prefix, msg)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) { l.log(LevelDebug, format, args...) }

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) { l.log(LevelInfo, format, args...) }

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) { l.log(LevelWarn, format, args...) }

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) { l.log(LevelError, format, args...) }

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Default logger.
var defaultLogger = NewLogger("agent-os", LevelInfo)

// DefaultLogger returns the default logger.
func DefaultLogger() *Logger { return defaultLogger }

// SetDefaultLogger sets the default logger.
func SetDefaultLogger(l *Logger) { defaultLogger = l }
