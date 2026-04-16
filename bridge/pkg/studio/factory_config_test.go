package studio

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func setupFactoryWithDef(t *testing.T, defID string) (*AgentFactory, *mockDockerClient) {
	t.Helper()

	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	def := &AgentDefinition{
		ID:           defID,
		Name:         "Config Test Agent",
		Skills:       []string{"browser_navigate"},
		ResourceTier: "medium",
		CreatedBy:    "@test:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	if err := store.CreateDefinition(def); err != nil {
		t.Fatalf("failed to create definition: %v", err)
	}

	mockDocker := &mockDockerClient{}
	factory := NewAgentFactory(FactoryConfig{
		DockerClient: mockDocker,
		Store:        store,
		StateDir:     os.TempDir(),
	})

	return factory, mockDocker
}

func findEnv(t *testing.T, env []string, prefix string) (string, bool) {
	t.Helper()
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return e, true
		}
	}
	return "", false
}

func TestSpawnConfigPassthrough(t *testing.T) {
	factory, mockDocker := setupFactoryWithDef(t, "cfg-passthrough")

	_, err := factory.Spawn(context.Background(), &SpawnRequest{
		DefinitionID: "cfg-passthrough",
		UserID:       "@test:example.com",
		Config:       json.RawMessage(`{"model":"gpt-4","temperature":0.7}`),
	})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	if len(mockDocker.createdContainers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(mockDocker.createdContainers))
	}

	val, found := findEnv(t, mockDocker.createdContainers[0].config.Env, "STEP_CONFIG=")
	if !found {
		t.Fatal("STEP_CONFIG env var not found in container config")
	}

	expected := `STEP_CONFIG={"model":"gpt-4","temperature":0.7}`
	if val != expected {
		t.Errorf("expected %q, got %q", expected, val)
	}
}

func TestSpawnConfigNil(t *testing.T) {
	factory, mockDocker := setupFactoryWithDef(t, "cfg-nil")

	_, err := factory.Spawn(context.Background(), &SpawnRequest{
		DefinitionID: "cfg-nil",
		UserID:       "@test:example.com",
		Config:       nil,
	})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	if len(mockDocker.createdContainers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(mockDocker.createdContainers))
	}

	_, found := findEnv(t, mockDocker.createdContainers[0].config.Env, "STEP_CONFIG=")
	if found {
		t.Error("STEP_CONFIG should NOT be present when Config is nil")
	}
}

func TestSpawnConfigEmpty(t *testing.T) {
	factory, mockDocker := setupFactoryWithDef(t, "cfg-empty")

	_, err := factory.Spawn(context.Background(), &SpawnRequest{
		DefinitionID: "cfg-empty",
		UserID:       "@test:example.com",
		Config:       json.RawMessage{},
	})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	if len(mockDocker.createdContainers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(mockDocker.createdContainers))
	}

	_, found := findEnv(t, mockDocker.createdContainers[0].config.Env, "STEP_CONFIG=")
	if found {
		t.Error("STEP_CONFIG should NOT be present when Config is empty")
	}
}

func TestSpawnConfigRawJSON(t *testing.T) {
	factory, mockDocker := setupFactoryWithDef(t, "cfg-raw")

	_, err := factory.Spawn(context.Background(), &SpawnRequest{
		DefinitionID: "cfg-raw",
		UserID:       "@test:example.com",
		Config:       json.RawMessage("not-json"),
	})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	if len(mockDocker.createdContainers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(mockDocker.createdContainers))
	}

	val, found := findEnv(t, mockDocker.createdContainers[0].config.Env, "STEP_CONFIG=")
	if !found {
		t.Fatal("STEP_CONFIG env var not found in container config")
	}

	expected := "STEP_CONFIG=not-json"
	if val != expected {
		t.Errorf("expected %q, got %q", expected, val)
	}
}
