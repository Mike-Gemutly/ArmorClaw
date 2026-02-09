// Package logger provides structured logging for ArmorClaw with security event tracking
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	globalLogger *Logger
	once         sync.Once
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Logger wraps slog.Logger with ArmorClaw-specific functionality
type Logger struct {
	*slog.Logger
	component string
}

// Config holds logger configuration
type Config struct {
	Level       string
	Format      string // "json" or "text"
	Output      string // "stdout", "stderr", or file path
	Component   string // Component name for logs
}

// New creates a new logger instance
func New(cfg Config) (*Logger, error) {
	// Parse log level
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Determine output writer
	var writer io.Writer
	output := cfg.Output
	if output == "" {
		output = "stdout" // Default to stdout if empty
	}

	switch output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// File output
		if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writer = file
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create handler based on format
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	// Create logger with default attributes
	logger := slog.New(handler)
	logger = logger.With(
		"service", "armorclaw",
		"component", cfg.Component,
		"version", "1.1.0",
	)

	return &Logger{
		Logger:    logger,
		component: cfg.Component,
	}, nil
}

// Initialize sets up the global logger with configuration
func Initialize(level, format, output string) error {
	var onceErr error
	once.Do(func() {
		// Use defaults if not specified
		if output == "" {
			output = "stdout"
		}
		if format == "" {
			format = "text"
		}
		if level == "" {
			level = "info"
		}

		loggerCfg := Config{
			Level:     level,
			Format:    format,
			Output:    output,
			Component: "bridge",
		}

		var err error
		globalLogger, err = New(loggerCfg)
		if err != nil {
			onceErr = fmt.Errorf("failed to initialize logger: %w", err)
			return
		}

		globalLogger.Info("logger initialized",
			"level", level,
			"format", format,
			"output", output,
		)
	})

	return onceErr
}

// Global returns the global logger instance
func Global() *Logger {
	if globalLogger == nil {
		// Fallback to default logger if not initialized
		logger, _ := New(Config{
			Level:     "info",
			Format:    "text",
			Output:    "stdout",
			Component: "bridge",
		})
		return logger
	}
	return globalLogger
}

// WithComponent returns a new logger with the component name set
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger:    l.Logger.With("component", component),
		component: component,
	}
}

// WithRequestID returns a new logger with a request ID for tracing
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger:    l.Logger.With("request_id", requestID),
		component: l.component,
	}
}

// WithSessionID returns a new logger with a session ID for container tracking
func (l *Logger) WithSessionID(sessionID string) *Logger {
	return &Logger{
		Logger:    l.Logger.With("session_id", sessionID),
		component: l.component,
	}
}

// WithContainerID returns a new logger with a container ID for tracking
func (l *Logger) WithContainerID(containerID string) *Logger {
	return &Logger{
		Logger:    l.Logger.With("container_id", containerID),
		component: l.component,
	}
}

// SecurityEvent logs a security-relevant event with standard fields
func (l *Logger) SecurityEvent(ctx context.Context, eventType string, attrs ...slog.Attr) {
	// Build base attributes
	baseAttrs := []slog.Attr{
		slog.String("event_type", eventType),
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
		slog.String("category", "security"),
	}

	// Add caller information if available
	if _, file, line, ok := runtimeCaller(3); ok {
		baseAttrs = append(baseAttrs,
			slog.String("source_file", filepath.Base(file)),
			slog.Int("source_line", line),
		)
	}

	// Merge with provided attributes
	allAttrs := append(baseAttrs, attrs...)

	l.LogAttrs(ctx, slog.LevelInfo, "security event", allAttrs...)
}

// AuditEvent logs an audit trail event for compliance
func (l *Logger) AuditEvent(ctx context.Context, action string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("action", action),
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
		slog.String("category", "audit"),
	}

	allAttrs := append(baseAttrs, attrs...)

	l.LogAttrs(ctx, slog.LevelInfo, "audit event", allAttrs...)
}

// ErrorEvent logs an error with context
func (l *Logger) ErrorEvent(ctx context.Context, message string, err error, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("error", err.Error()),
		slog.String("error_type", fmt.Sprintf("%T", err)),
	}

	allAttrs := append(baseAttrs, attrs...)

	l.LogAttrs(ctx, slog.LevelError, message, allAttrs...)
}

// runtimeCaller captures caller information for stack traces
func runtimeCaller(skip int) (pc uintptr, file string, line int, ok bool) {
	pc, file, line, ok = runtime.Caller(skip + 1)
	// Trim file path to basename
	if ok {
		file = filepath.Base(file)
	}
	return
}

// Convenience methods that use global logger

// Info logs an info message
func Info(msg string, args ...any) {
	Global().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Global().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Global().Error(msg, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Global().Debug(msg, args...)
}

// SecurityEvent logs a security event using the global logger
func SecurityEvent(eventType string, attrs ...slog.Attr) {
	Global().SecurityEvent(context.Background(), eventType, attrs...)
}

// AuditEvent logs an audit event using the global logger
func AuditEvent(action string, attrs ...slog.Attr) {
	Global().AuditEvent(context.Background(), action, attrs...)
}

// LogAttr creates a slog.Attr from a key and value (convenience helper)
func LogAttr(key string, value interface{}) slog.Attr {
	return slog.Any(key, value)
}
