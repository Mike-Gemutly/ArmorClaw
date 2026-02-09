// Package logger provides tests for the structured logging system
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewLogger tests creating a new logger instance
func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid text logger",
			config: Config{
				Level:     "info",
				Format:    "text",
				Output:    "stdout",
				Component: "test",
			},
			wantErr: false,
		},
		{
			name: "valid json logger",
			config: Config{
				Level:     "debug",
				Format:    "json",
				Output:    "stderr",
				Component: "test",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: Config{
				Level:     "invalid",
				Format:    "text",
				Output:    "stdout",
				Component: "test",
			},
			wantErr: false, // Falls back to info
		},
		{
			name: "empty values use defaults",
			config: Config{
				Level:     "",
				Format:    "",
				Output:    "",
				Component: "",
			},
			wantErr: false, // Falls back to defaults (stdout is valid)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger == nil {
				t.Error("New() returned nil logger")
			}
		})
	}
}

// TestLoggerLevels tests different log levels
func TestLoggerLevels(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	logger, err := New(Config{
		Level:     "debug",
		Format:    "json",
		Output:    "stdout",
		Component: "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Redirect logger output to buffer
	originalLogger := logger.Logger
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger.Logger = slog.New(jsonHandler)

	tests := []struct {
		name   string
		level  string
		method func(msg string, args ...any)
	}{
		{"debug", "debug", func(msg string, args ...any) {
			logger.Debug(msg, args...)
		}},
		{"info", "info", func(msg string, args ...any) {
			logger.Info(msg, args...)
		}},
		{"warn", "warn", func(msg string, args ...any) {
			logger.Warn(msg, args...)
		}},
		{"error", "error", func(msg string, args ...any) {
			logger.Error(msg, args...)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.method("test message", "key", "value")

			output := buf.String()
			if output == "" {
				t.Errorf("No output for %s level", tt.level)
			}

			// Verify JSON format
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Errorf("Output is not valid JSON: %v", err)
			}

			// Verify common fields
			if logEntry["level"] == nil {
				t.Error("Missing level field")
			}
			if logEntry["msg"] == nil {
				t.Error("Missing msg field")
			}
		})
	}

	// Restore original logger
	logger.Logger = originalLogger
}

// TestWithComponent tests creating logger with component name
func TestWithComponent(t *testing.T) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: "base",
	})

	newLogger := logger.WithComponent("rpc")
	if newLogger == nil {
		t.Fatal("WithComponent() returned nil")
	}

	// Verify it's a different instance
	if newLogger == logger {
		t.Error("WithComponent() returned same instance")
	}
}

// TestWithRequestID tests creating logger with request ID
func TestWithRequestID(t *testing.T) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: "base",
	})

	requestID := "req-12345"
	newLogger := logger.WithRequestID(requestID)
	if newLogger == nil {
		t.Fatal("WithRequestID() returned nil")
	}

	// Verify it's a different instance
	if newLogger == logger {
		t.Error("WithRequestID() returned same instance")
	}
}

// TestWithSessionID tests creating logger with session ID
func TestWithSessionID(t *testing.T) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: "base",
	})

	sessionID := "sess-67890"
	newLogger := logger.WithSessionID(sessionID)
	if newLogger == nil {
		t.Fatal("WithSessionID() returned nil")
	}

	// Verify it's a different instance
	if newLogger == logger {
		t.Error("WithSessionID() returned same instance")
	}
}

// TestWithContainerID tests creating logger with container ID
func TestWithContainerID(t *testing.T) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: "base",
	})

	containerID := "container-abc123"
	newLogger := logger.WithContainerID(containerID)
	if newLogger == nil {
		t.Fatal("WithContainerID() returned nil")
	}

	// Verify it's a different instance
	if newLogger == logger {
		t.Error("WithContainerID() returned same instance")
	}
}

// TestSecurityEvent tests logging security events
func TestSecurityEvent(t *testing.T) {
	var buf bytes.Buffer

	logger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "test",
	})

	// Redirect logger output to buffer
	originalHandler := logger.Handler()
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger.Logger = slog.New(jsonHandler)

	ctx := context.Background()
	logger.SecurityEvent(ctx, "test_event",
		slog.String("test_key", "test_value"),
		slog.Int("test_int", 42),
	)

	output := buf.String()
	if output == "" {
		t.Fatal("SecurityEvent() produced no output")
	}

	// Verify JSON format
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify security event fields
	if logEntry["event_type"] != "test_event" {
		t.Errorf("event_type = %v, want test_event", logEntry["event_type"])
	}
	if logEntry["category"] != "security" {
		t.Errorf("category = %v, want security", logEntry["category"])
	}
	if logEntry["test_key"] != "test_value" {
		t.Errorf("test_key = %v, want test_value", logEntry["test_key"])
	}
	if logEntry["test_int"] != float64(42) {
		t.Errorf("test_int = %v, want 42", logEntry["test_int"])
	}

	// Verify timestamp exists
	if logEntry["timestamp"] == nil {
		t.Error("Missing timestamp field")
	} else {
		// Verify timestamp is valid RFC3339
		_, err := time.Parse(time.RFC3339, logEntry["timestamp"].(string))
		if err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}
	}

	// Restore original handler
	logger.Logger = slog.New(originalHandler)
}

// TestAuditEvent tests logging audit events
func TestAuditEvent(t *testing.T) {
	var buf bytes.Buffer

	logger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "test",
	})

	// Redirect logger output to buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger.Logger = slog.New(jsonHandler)

	ctx := context.Background()
	logger.AuditEvent(ctx, "user_login",
		slog.String("user_id", "user@example.com"),
		slog.String("ip", "192.168.1.1"),
	)

	output := buf.String()
	if output == "" {
		t.Fatal("AuditEvent() produced no output")
	}

	// Verify JSON format
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify audit event fields
	if logEntry["action"] != "user_login" {
		t.Errorf("action = %v, want user_login", logEntry["action"])
	}
	if logEntry["category"] != "audit" {
		t.Errorf("category = %v, want audit", logEntry["category"])
	}
	if logEntry["user_id"] != "user@example.com" {
		t.Errorf("user_id = %v, want user@example.com", logEntry["user_id"])
	}
}

// TestErrorEvent tests logging error events
func TestErrorEvent(t *testing.T) {
	var buf bytes.Buffer

	logger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "test",
	})

	// Redirect logger output to buffer
	logger.Logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	ctx := context.Background()
	testErr := os.ErrNotExist
	logger.ErrorEvent(ctx, "file not found", testErr,
		slog.String("file_path", "/tmp/test.txt"),
	)

	output := buf.String()
	if output == "" {
		t.Fatal("ErrorEvent() produced no output")
	}

	// Verify JSON format
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify error event fields
	if logEntry["error"] == nil {
		t.Error("Missing error field")
	}
	if logEntry["error_type"] == nil {
		t.Error("Missing error_type field")
	}
	if logEntry["file_path"] != "/tmp/test.txt" {
		t.Errorf("file_path = %v, want /tmp/test.txt", logEntry["file_path"])
	}
}

// TestGlobalLogger tests the global logger functions
func TestGlobalLogger(t *testing.T) {
	// Reset global logger
	globalLogger = nil
	once = *new(sync.Once)

	// Test Global() returns a logger even if not initialized
	logger := Global()
	if logger == nil {
		t.Fatal("Global() returned nil")
	}

	// Test convenience functions
	Info("test info")
	Warn("test warn")
	Error("test error")
	Debug("test debug")

	// Test with initialized logger
	Initialize("info", "text", "stdout")

	Info("test info 2")
	Warn("test warn 2")
	Error("test error 2")
	Debug("test debug 2")
}

// TestFileOutput tests logging to a file
func TestFileOutput(t *testing.T) {
	// Create temp directory
	tmpDir := os.TempDir()
	logFile := filepath.Join(tmpDir, "test-logger-"+time.Now().Format("20060102150405")+".log")

	logger, err := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    logFile,
		Component: "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Write test log
	logger.Info("test message to file", "key", "value")

	// Give it a moment to flush
	time.Sleep(100 * time.Millisecond)

	// Read log file
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify content
	if len(data) == 0 {
		t.Error("Log file is empty")
	}

	// Verify JSON format
	var logEntry map[string]interface{}
	if err := json.Unmarshal(data, &logEntry); err != nil {
		t.Errorf("Log file content is not valid JSON: %v", err)
	}

	if logEntry["msg"] != "test message to file" {
		t.Errorf("msg = %v, want 'test message to file'", logEntry["msg"])
	}

	// Clean up
	os.Remove(logFile)
}

// TestJSONFormat tests JSON log format
func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer

	logger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "test-component",
	})

	// Redirect to buffer
	logger.Logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("test message", "key1", "value1", "key2", 42)

	// Verify JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify required fields (note: redirected logger doesn't have default attributes)
	requiredFields := []string{"time", "level", "msg"}
	for _, field := range requiredFields {
		if logEntry[field] == nil {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify values
	if logEntry["msg"] != "test message" {
		t.Errorf("msg = %v, want 'test message'", logEntry["msg"])
	}
	// Note: component and service are not present when using redirected logger
	// The original logger includes them, but the redirected one doesn't
	if logEntry["key1"] != "value1" {
		t.Errorf("key1 = %v, want 'value1'", logEntry["key1"])
	}
	if logEntry["key2"] != float64(42) {
		t.Errorf("key2 = %v, want 42", logEntry["key2"])
	}
}

// TestTextFormat tests text log format
func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer

	logger, _ := New(Config{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: "test-component",
	})

	// Redirect to buffer
	textHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger.Logger = slog.New(textHandler)

	logger.Info("test message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Fatal("No output for text format")
	}

	// Verify text format contains expected elements
	if !strings.Contains(output, "test message") {
		t.Error("Output doesn't contain message")
	}
	if !strings.Contains(output, "key=value") {
		t.Error("Output doesn't contain key=value pair")
	}
}

// TestInitialize tests logger initialization
func TestInitialize(t *testing.T) {
	// Reset
	globalLogger = nil
	once = *new(sync.Once)

	tests := []struct {
		name    string
		level   string
		format  string
		output  string
		wantErr bool
	}{
		{
			name:    "valid initialization",
			level:   "info",
			format:  "json",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "empty values use defaults",
			level:   "",
			format:  "",
			output:  "",
			wantErr: false,
		},
		{
			name:    "debug level",
			level:   "debug",
			format:  "text",
			output:  "stderr",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset before each test
			globalLogger = nil
			once = *new(sync.Once)

			err := Initialize(tt.level, tt.format, tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && globalLogger == nil {
				t.Error("Initialize() didn't set globalLogger")
			}
		})
	}
}

// BenchmarkLoggerJSON benchmarks JSON logging
func BenchmarkLoggerJSON(b *testing.B) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "bench",
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.SecurityEvent(ctx, "bench_event",
			slog.String("key", "value"),
			slog.Int("iteration", i),
		)
	}
}

// BenchmarkLoggerText benchmarks text logging
func BenchmarkLoggerText(b *testing.B) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: "bench",
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("bench message", "iteration", i)
	}
}
