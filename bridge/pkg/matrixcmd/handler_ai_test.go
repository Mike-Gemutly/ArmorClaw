package matrixcmd

import (
	"context"
	"testing"
)

type mockAIRuntime struct {
	providers []string
	models    map[string][]string
	current   struct {
		provider string
		model    string
	}
	switchErr error
}

func (m *mockAIRuntime) ListProviders() ([]string, error) {
	return m.providers, nil
}

func (m *mockAIRuntime) ListModels(provider string) ([]string, error) {
	if models, ok := m.models[provider]; ok {
		return models, nil
	}
	return []string{"default"}, nil
}

func (m *mockAIRuntime) SwitchProvider(provider, model string) error {
	if m.switchErr != nil {
		return m.switchErr
	}
	m.current.provider = provider
	m.current.model = model
	return nil
}

func (m *mockAIRuntime) GetStatus() string {
	return "Provider: " + m.current.provider + "\nModel: " + m.current.model
}

func (m *mockAIRuntime) GetCurrent() (provider, model string) {
	return m.current.provider, m.current.model
}

func TestHandleAI_Providers(t *testing.T) {
	ai := &mockAIRuntime{
		providers: []string{"openai", "anthropic", "xai"},
	}

	h := &CommandHandler{
		aiRuntime: ai,
		commands:  make(map[string]*command),
		sendMessage: func(roomID, userID, message string) error {
			return nil
		},
	}

	h.registerCommand("ai", `^/ai\s*(.*)$`, "AI management", h.handleAI)

	ctx := context.Background()
	_, err := h.HandleMessage(ctx, "room1", "user1", "/ai providers")
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestHandleAI_Models(t *testing.T) {
	ai := &mockAIRuntime{
		models: map[string][]string{
			"openai": {"gpt-4o", "gpt-4o-mini"},
		},
	}

	h := &CommandHandler{
		aiRuntime: ai,
		commands:  make(map[string]*command),
		sendMessage: func(roomID, userID, message string) error {
			return nil
		},
	}

	h.registerCommand("ai", `^/ai\s*(.*)$`, "AI management", h.handleAI)

	ctx := context.Background()
	_, err := h.HandleMessage(ctx, "room1", "user1", "/ai models openai")
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestHandleAI_Switch(t *testing.T) {
	ai := &mockAIRuntime{
		providers: []string{"openai", "anthropic"},
		models: map[string][]string{
			"openai": {"gpt-4o", "gpt-4o-mini"},
		},
	}

	h := &CommandHandler{
		aiRuntime: ai,
		commands:  make(map[string]*command),
		sendMessage: func(roomID, userID, message string) error {
			return nil
		},
	}

	h.registerCommand("ai", `^/ai\s*(.*)$`, "AI management", h.handleAI)

	ctx := context.Background()
	_, err := h.HandleMessage(ctx, "room1", "user1", "/ai switch openai gpt-4o")
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	if ai.current.provider != "openai" {
		t.Errorf("Expected provider openai, got %s", ai.current.provider)
	}
	if ai.current.model != "gpt-4o" {
		t.Errorf("Expected model gpt-4o, got %s", ai.current.model)
	}
}

func TestHandleAI_Status(t *testing.T) {
	ai := &mockAIRuntime{}
	ai.current.provider = "anthropic"
	ai.current.model = "claude-3-opus"

	h := &CommandHandler{
		aiRuntime: ai,
		commands:  make(map[string]*command),
		sendMessage: func(roomID, userID, message string) error {
			return nil
		},
	}

	h.registerCommand("ai", `^/ai\s*(.*)$`, "AI management", h.handleAI)

	ctx := context.Background()
	_, err := h.HandleMessage(ctx, "room1", "user1", "/ai status")
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestHandleAI_Help(t *testing.T) {
	ai := &mockAIRuntime{}

	h := &CommandHandler{
		aiRuntime: ai,
		commands:  make(map[string]*command),
		sendMessage: func(roomID, userID, message string) error {
			return nil
		},
	}

	h.registerCommand("ai", `^/ai\s*(.*)$`, "AI management", h.handleAI)

	ctx := context.Background()
	handled, err := h.HandleMessage(ctx, "room1", "user1", "/ai")
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
	if !handled {
		t.Error("Expected /ai to be handled")
	}
}

func TestHandleAI_NoRuntime(t *testing.T) {
	h := &CommandHandler{
		aiRuntime: nil,
		commands:  make(map[string]*command),
		sendMessage: func(roomID, userID, message string) error {
			return nil
		},
	}

	h.registerCommand("ai", `^/ai\s*(.*)$`, "AI management", h.handleAI)

	ctx := context.Background()
	handled, err := h.HandleMessage(ctx, "room1", "user1", "/ai providers")
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
	if !handled {
		t.Error("Expected /ai to be handled (returns error response)")
	}
}
