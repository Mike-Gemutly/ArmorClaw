package sidecar

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"google.golang.org/grpc/metadata"
)

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name          string
		md            metadata.MD
		wantRequestID string
		wantUserID    string
		wantAgentID   string
		wantSessionID string
	}{
		{
			name: "all metadata present",
			md: metadata.Pairs(
				"x-request-id", "req-123",
				"x-user-id", "user-456",
				"x-agent-id", "agent-789",
				"x-session-id", "sess-abc",
			),
			wantRequestID: "req-123",
			wantUserID:    "user-456",
			wantAgentID:   "agent-789",
			wantSessionID: "sess-abc",
		},
		{
			name: "partial metadata",
			md: metadata.Pairs(
				"x-request-id", "req-123",
				"x-user-id", "user-456",
			),
			wantRequestID: "req-123",
			wantUserID:    "user-456",
			wantAgentID:   "",
			wantSessionID: "",
		},
		{
			name:          "no metadata",
			md:            metadata.MD{},
			wantRequestID: "",
			wantUserID:    "",
			wantAgentID:   "",
			wantSessionID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRequestID, gotUserID, gotAgentID, gotSessionID := extractMetadata(tt.md)

			if gotRequestID != tt.wantRequestID {
				t.Errorf("extractMetadata() requestID = %v, want %v", gotRequestID, tt.wantRequestID)
			}
			if gotUserID != tt.wantUserID {
				t.Errorf("extractMetadata() userID = %v, want %v", gotUserID, tt.wantUserID)
			}
			if gotAgentID != tt.wantAgentID {
				t.Errorf("extractMetadata() agentID = %v, want %v", gotAgentID, tt.wantAgentID)
			}
			if gotSessionID != tt.wantSessionID {
				t.Errorf("extractMetadata() sessionID = %v, want %v", gotSessionID, tt.wantSessionID)
			}
		})
	}
}

func TestNewAuditClient(t *testing.T) {
	client := NewClient(nil)

	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := audit.Config{
		Path:   filepath.Join(tmpDir, "audit.db"),
		MaxLen: 100,
	}
	auditLog, err := audit.NewAuditLog(cfg)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	ac := NewAuditClient(client, auditLog)

	if ac == nil {
		t.Fatal("NewAuditClient() returned nil")
	}
	if ac.GetClient() != client {
		t.Error("NewAuditClient() client not set correctly")
	}
}

func TestAuditClient_LogQueueEvent(t *testing.T) {
	client := NewClient(nil)

	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := audit.Config{
		Path:   filepath.Join(tmpDir, "audit.db"),
		MaxLen: 100,
	}
	auditLog, err := audit.NewAuditLog(cfg)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	ac := NewAuditClient(client, auditLog)

	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(
		"x-request-id", "req-123",
		"x-user-id", "user-456",
	))

	err = ac.LogQueueEvent(ctx, "TestOperation", 5)
	if err != nil {
		t.Errorf("LogQueueEvent() error = %v", err)
	}

	count := auditLog.Count()
	if count != 1 {
		t.Errorf("LogQueueEvent() count = %v, want 1", count)
	}
}

func TestAuditClient_LogRetryEvent(t *testing.T) {
	client := NewClient(nil)

	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := audit.Config{
		Path:   filepath.Join(tmpDir, "audit.db"),
		MaxLen: 100,
	}
	auditLog, err := audit.NewAuditLog(cfg)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	ac := NewAuditClient(client, auditLog)

	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(
		"x-request-id", "req-123",
		"x-agent-id", "agent-789",
	))

	err = ac.LogRetryEvent(ctx, "TestOperation", 2)
	if err != nil {
		t.Errorf("LogRetryEvent() error = %v", err)
	}

	count := auditLog.Count()
	if count != 1 {
		t.Errorf("LogRetryEvent() count = %v, want 1", count)
	}
}

func TestAuditClient_SetMetadataExtractor(t *testing.T) {
	client := NewClient(nil)

	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := audit.Config{
		Path:   filepath.Join(tmpDir, "audit.db"),
		MaxLen: 100,
	}
	auditLog, err := audit.NewAuditLog(cfg)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	ac := NewAuditClient(client, auditLog)

	customCalled := false
	customExtractor := func(md metadata.MD) (string, string, string, string) {
		customCalled = true
		return "custom-req-id", "custom-user-id", "custom-agent-id", "custom-session-id"
	}

	ac.SetMetadataExtractor(customExtractor)

	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs())

	err = ac.LogQueueEvent(ctx, "TestOperation", 1)
	if err != nil {
		t.Errorf("LogQueueEvent() error = %v", err)
	}

	if !customCalled {
		t.Error("SetMetadataExtractor() custom extractor not called")
	}
}

func TestAuditLogEntry(t *testing.T) {
	entry := &auditLogEntry{
		Operation: "TestOperation",
		Success:   true,
		Duration:  100 * time.Millisecond,
		RequestID: "req-123",
		UserID:    "user-456",
		AgentID:   "agent-789",
		SessionID: "sess-abc",
		FileSize:  1024,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	if entry.Operation != "TestOperation" {
		t.Errorf("Operation = %v, want TestOperation", entry.Operation)
	}
	if !entry.Success {
		t.Error("Success = false, want true")
	}
	if entry.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", entry.Duration)
	}
	if entry.FileSize != 1024 {
		t.Errorf("FileSize = %v, want 1024", entry.FileSize)
	}
	if entry.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %v, want value1", entry.Metadata["key1"])
	}
}
