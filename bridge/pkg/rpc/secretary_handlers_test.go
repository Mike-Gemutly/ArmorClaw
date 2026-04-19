package rpc

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSecretaryMethodRegistration(t *testing.T) {
	methods := []string{
		"secretary.start_workflow",
		"secretary.get_workflow",
		"secretary.cancel_workflow",
		"secretary.advance_workflow",
		"secretary.list_templates",
		"secretary.create_template",
		"secretary.get_template",
		"secretary.delete_template",
		"secretary.update_template",
		"task.create",
		"task.list",
		"task.cancel",
		"task.get",
	}

	server := &Server{}
	server.registerHandlers()

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			if _, exists := server.handlers[method]; !exists {
				t.Errorf("secretary/task method %q not registered in handlers map", method)
			}
		})
	}
}

func TestSecretaryHandlerNotConfigured(t *testing.T) {
	server := &Server{}
	server.registerHandlers()

	methods := []string{
		"secretary.start_workflow",
		"secretary.get_workflow",
		"secretary.cancel_workflow",
		"secretary.advance_workflow",
		"secretary.list_templates",
		"secretary.create_template",
		"secretary.get_template",
		"secretary.delete_template",
		"secretary.update_template",
		"task.create",
		"task.list",
		"task.cancel",
		"task.get",
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := server.handlers[method]
			if handler == nil {
				t.Fatalf("handler for %q not registered", method)
			}

			req := &Request{
				Method: method,
				Params: json.RawMessage(`{}`),
			}

			_, errObj := handler(context.Background(), req)
			if errObj == nil {
				t.Errorf("expected error when secretary handler not configured, got nil for %s", method)
			}
			if errObj.Code != InternalError {
				t.Errorf("expected InternalError code %d, got %d for %s", InternalError, errObj.Code, method)
			}
		})
	}
}

func TestSecretaryHandlerDelegatesToRPC(t *testing.T) {
	mock := &mockSecretaryRPCHandler{
		responses: map[string]interface{}{
			"secretary.start_workflow": map[string]interface{}{"status": "started", "workflow_id": "wf_123"},
			"secretary.get_workflow":   map[string]interface{}{"workflow_id": "wf_123", "status": "running"},
			"secretary.cancel_workflow": map[string]interface{}{"status": "cancelled"},
			"secretary.advance_workflow": map[string]interface{}{"status": "advanced"},
			"secretary.list_templates":  map[string]interface{}{"templates": []interface{}{}, "count": 0},
			"secretary.create_template": map[string]interface{}{"template_id": "tpl_123"},
			"secretary.get_template":    map[string]interface{}{"template_id": "tpl_123", "name": "test"},
			"secretary.delete_template": map[string]interface{}{"deleted": true},
			"secretary.update_template": map[string]interface{}{"template_id": "tpl_123", "name": "updated"},
			"task.create":              map[string]interface{}{"task_id": "task_123"},
			"task.list":                map[string]interface{}{"tasks": []interface{}{}},
			"task.cancel":              map[string]interface{}{"success": true},
			"task.get":                 map[string]interface{}{"task_id": "task_123"},
		},
	}

	server := &Server{
		secretaryHandler: mock,
	}
	server.registerHandlers()

	for method, expectedResult := range mock.responses {
		t.Run(method, func(t *testing.T) {
			handler := server.handlers[method]
			if handler == nil {
				t.Fatalf("handler for %q not registered", method)
			}

			req := &Request{
				Method: method,
				Params: json.RawMessage(`{}`),
			}

			result, errObj := handler(context.Background(), req)
			if errObj != nil {
				t.Fatalf("unexpected error for %s: %v", method, errObj)
			}
			if result == nil {
				t.Fatalf("expected non-nil result for %s", method)
			}

			// Verify the mock was called with correct method
			if mock.lastMethod != method {
				t.Errorf("expected delegate called with %q, got %q", method, mock.lastMethod)
			}

			// Verify result content matches
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("expected map result for %s, got %T", method, result)
			}
			expectedMap := expectedResult.(map[string]interface{})
			for k := range expectedMap {
				if _, exists := resultMap[k]; !exists {
					t.Errorf("expected key %q in result for %s", k, method)
				}
			}
		})
	}
}

func TestSecretaryHandlerDelegatesErrors(t *testing.T) {
	mock := &mockSecretaryRPCHandler{
		errResponse: true,
		errCode:     -32001,
		errMessage:  "something went wrong",
	}

	server := &Server{
		secretaryHandler: mock,
	}
	server.registerHandlers()

	handler := server.handlers["secretary.get_workflow"]
	if handler == nil {
		t.Fatalf("handler not registered")
	}

	req := &Request{
		Method: "secretary.get_workflow",
		Params: json.RawMessage(`{"workflow_id": "missing"}`),
	}

	result, errObj := handler(context.Background(), req)
	if result != nil {
		t.Errorf("expected nil result on error, got %v", result)
	}
	if errObj == nil {
		t.Fatal("expected error, got nil")
	}
	if errObj.Code != -32001 {
		t.Errorf("expected error code -32001, got %d", errObj.Code)
	}
	if errObj.Message != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", errObj.Message)
	}
}

// mockSecretaryRPCHandler satisfies the secretaryRPCHandler interface for testing
type mockSecretaryRPCHandler struct {
	responses  map[string]interface{}
	errResponse bool
	errCode    int
	errMessage string
	lastMethod string
	lastParams json.RawMessage
}

func (m *mockSecretaryRPCHandler) Handle(method string, params json.RawMessage) (interface{}, error) {
	m.lastMethod = method
	m.lastParams = params

	if m.errResponse {
		return nil, &secretaryHandlerError{code: m.errCode, message: m.errMessage}
	}

	if resp, ok := m.responses[method]; ok {
		return resp, nil
	}

	return nil, &secretaryHandlerError{code: -32001, message: "unknown method"}
}
