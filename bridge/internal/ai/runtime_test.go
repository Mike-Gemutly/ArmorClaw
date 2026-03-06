package ai

import (
	"testing"

	"github.com/armorclaw/bridge/pkg/keystore"
)

type mockKeystore struct {
	keys map[keystore.Provider][]keystore.KeyInfo
}

func newMockKeystore() *mockKeystore {
	return &mockKeystore{
		keys: make(map[keystore.Provider][]keystore.KeyInfo),
	}
}

func (m *mockKeystore) AddKey(provider keystore.Provider, id string) {
	m.keys[provider] = append(m.keys[provider], keystore.KeyInfo{
		ID:          id,
		Provider:    provider,
		DisplayName: id + " key",
	})
}

func (m *mockKeystore) List(provider keystore.Provider) ([]keystore.KeyInfo, error) {
	if provider == "" {
		var all []keystore.KeyInfo
		for _, keys := range m.keys {
			all = append(all, keys...)
		}
		return all, nil
	}
	return m.keys[provider], nil
}

func TestAIRuntime_ListProviders(t *testing.T) {
	ks := newMockKeystore()
	ks.AddKey(keystore.ProviderOpenAI, "gpt-4o-key")
	ks.AddKey(keystore.ProviderAnthropic, "claude-key")
	ks.AddKey(keystore.ProviderOpenAI, "gpt-3.5-key")

	runtime := &AIRuntime{
		keystore: ks,
		provider: "openai",
		model:    "gpt-4o",
	}

	providers, err := runtime.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d: %v", len(providers), providers)
	}

	found := make(map[string]bool)
	for _, p := range providers {
		found[p] = true
	}

	if !found["openai"] {
		t.Error("Expected openai in providers")
	}
	if !found["anthropic"] {
		t.Error("Expected anthropic in providers")
	}
}

func TestAIRuntime_ListModels(t *testing.T) {
	ks := newMockKeystore()
	ks.AddKey(keystore.ProviderOpenAI, "gpt-4o")
	ks.AddKey(keystore.ProviderOpenAI, "gpt-4o-mini")
	ks.AddKey(keystore.ProviderAnthropic, "claude-3-5-sonnet")

	runtime := &AIRuntime{
		keystore: ks,
		provider: "openai",
		model:    "gpt-4o",
	}

	models, err := runtime.ListModels("openai")
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d: %v", len(models), models)
	}

	models, err = runtime.ListModels("anthropic")
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d: %v", len(models), models)
	}
}

func TestAIRuntime_ListModels_Defaults(t *testing.T) {
	ks := newMockKeystore()

	runtime := &AIRuntime{
		keystore: ks,
		provider: "openai",
		model:    "gpt-4o",
	}

	models, err := runtime.ListModels("openai")
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected default models, got none")
	}

	if models[0] != "gpt-4o" {
		t.Errorf("Expected first model to be gpt-4o, got %s", models[0])
	}
}

func TestAIRuntime_GetCurrent(t *testing.T) {
	ks := newMockKeystore()
	runtime := &AIRuntime{
		keystore: ks,
		provider: "anthropic",
		model:    "claude-3-opus",
	}

	provider, model := runtime.GetCurrent()
	if provider != "anthropic" {
		t.Errorf("Expected provider anthropic, got %s", provider)
	}
	if model != "claude-3-opus" {
		t.Errorf("Expected model claude-3-opus, got %s", model)
	}
}

func TestAIRuntime_GetStatus(t *testing.T) {
	ks := newMockKeystore()
	runtime := &AIRuntime{
		keystore: ks,
		provider: "openai",
		model:    "gpt-4o",
	}

	status := runtime.GetStatus()
	expected := "Provider: openai\nModel: gpt-4o"
	if status != expected {
		t.Errorf("Expected status %q, got %q", expected, status)
	}
}
