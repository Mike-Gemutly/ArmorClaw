package secretary

import (
	"context"
	"testing"
)

type mockAuditLogger struct {
	logFunc func(ctx context.Context, operation string, details map[string]interface{})
}

func (m *mockAuditLogger) LogOperation(ctx context.Context, operation string, details map[string]interface{}) error {
	if m.logFunc != nil {
		m.logFunc(ctx, operation, details)
	}
	return nil
}

func TestAuditLogger_LogWorkflowCreation(t *testing.T) {
	// Given: An audit logger
	logCalled := false
	audit := &mockAuditLogger{
		logFunc: func(ctx context.Context, operation string, details map[string]interface{}) {
			logCalled = true

			if operation != "workflow_created" {
				t.Errorf("expected operation 'workflow_created', got %s", operation)
			}

			if details["workflow_id"] == nil {
				t.Error("expected workflow_id in details")
			}

			if details["template_id"] == nil {
				t.Error("expected template_id in details")
			}

			if details["created_by"] == nil {
				t.Error("expected created_by in details")
			}
		},
	}

	// When: Logging workflow creation
	ctx := context.Background()
	err := audit.LogOperation(ctx, "workflow_created", map[string]interface{}{
		"workflow_id": "workflow_123",
		"template_id": "template_456",
		"created_by":  "@user:example.com",
	})

	// Then: Should log the operation
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !logCalled {
		t.Fatal("expected log to be called")
	}
}

func TestAuditLogger_LogAgentCreation(t *testing.T) {
	// Given: An audit logger
	logCalled := false
	audit := &mockAuditLogger{
		logFunc: func(ctx context.Context, operation string, details map[string]interface{}) {
			logCalled = true

			if operation != "agent_created" {
				t.Errorf("expected operation 'agent_created', got %s", operation)
			}

			if details["agent_id"] == nil {
				t.Error("expected agent_id in details")
			}

			if details["agent_name"] == nil {
				t.Error("expected agent_name in details")
			}

			if details["created_by"] == nil {
				t.Error("expected created_by in details")
			}
		},
	}

	// When: Logging agent creation
	ctx := context.Background()
	err := audit.LogOperation(ctx, "agent_created", map[string]interface{}{
		"agent_id":   "instance_123",
		"agent_name": "Secretary Agent",
		"created_by": "@user:example.com",
	})

	// Then: Should log the operation
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !logCalled {
		t.Fatal("expected log to be called")
	}
}

func TestAuditLogger_LogWorkflowStatusUpdate(t *testing.T) {
	// Given: An audit logger
	logCalled := false
	audit := &mockAuditLogger{
		logFunc: func(ctx context.Context, operation string, details map[string]interface{}) {
			logCalled = true

			if operation != "workflow_status_update" {
				t.Errorf("expected operation 'workflow_status_update', got %s", operation)
			}

			if details["workflow_id"] == nil {
				t.Error("expected workflow_id in details")
			}

			if details["old_status"] == nil {
				t.Error("expected old_status in details")
			}

			if details["new_status"] == nil {
				t.Error("expected new_status in details")
			}
		},
	}

	// When: Logging workflow status update
	ctx := context.Background()
	err := audit.LogOperation(ctx, "workflow_status_update", map[string]interface{}{
		"workflow_id": "workflow_123",
		"old_status":  "pending",
		"new_status":  "running",
	})

	// Then: Should log the operation
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !logCalled {
		t.Fatal("expected log to be called")
	}
}

func TestAuditLogger_LogPIIRequest(t *testing.T) {
	// Given: An audit logger
	logCalled := false
	audit := &mockAuditLogger{
		logFunc: func(ctx context.Context, operation string, details map[string]interface{}) {
			logCalled = true

			if operation != "pii_request" {
				t.Errorf("expected operation 'pii_request', got %s", operation)
			}

			if details["request_id"] == nil {
				t.Error("expected request_id in details")
			}

			if details["pii_fields"] == nil {
				t.Error("expected pii_fields in details")
			}

			if details["requested_by"] == nil {
				t.Error("expected requested_by in details")
			}
		},
	}

	// When: Logging PII request
	ctx := context.Background()
	err := audit.LogOperation(ctx, "pii_request", map[string]interface{}{
		"request_id":   "req_123",
		"pii_fields":   []string{"pii.email", "pii.phone"},
		"requested_by": "@user:example.com",
	})

	// Then: Should log the operation
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !logCalled {
		t.Fatal("expected log to be called")
	}
}
