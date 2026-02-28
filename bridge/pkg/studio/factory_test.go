package studio

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

//=============================================================================
// Mock Docker Client
//=============================================================================

type mockDockerClient struct {
	createdContainers []mockContainer
	startedContainers []string
	stoppedContainers []string
	removedContainers []string
	inspectError      error
	containerState    *types.ContainerState
}

type mockContainer struct {
	id     string
	config *container.Config
	name   string
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig any, platform any, name string) (container.CreateResponse, error) {
	id := "container-" + name
	m.createdContainers = append(m.createdContainers, mockContainer{
		id:     id,
		config: config,
		name:   name,
	})
	return container.CreateResponse{ID: id}, nil
}

func (m *mockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	m.startedContainers = append(m.startedContainers, containerID)
	return nil
}

func (m *mockDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	m.stoppedContainers = append(m.stoppedContainers, containerID)
	return nil
}

func (m *mockDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	m.removedContainers = append(m.removedContainers, containerID)
	return nil
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if m.inspectError != nil {
		return types.ContainerJSON{}, m.inspectError
	}
	state := m.containerState
	if state == nil {
		state = &types.ContainerState{Running: true, ExitCode: 0}
	}
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:    containerID,
			State: state,
		},
	}, nil
}

func (m *mockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return []types.Container{}, nil
}

//=============================================================================
// Mock Keystore Provider
//=============================================================================

type mockKeystore struct {
	unsealed bool
	secrets  map[string]string
}

func (m *mockKeystore) IsUnsealed() bool {
	return m.unsealed
}

func (m *mockKeystore) Get(key string) (string, error) {
	if val, ok := m.secrets[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}

//=============================================================================
// Factory Tests (CGO-free)
//=============================================================================

func TestAgentFactory_Spawn(t *testing.T) {
	// Create in-memory store
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create a test agent definition
	def := &AgentDefinition{
		ID:           "test-agent-001",
		Name:         "Test Agent",
		Skills:       []string{"browser_navigate", "pdf_generator"},
		PIIAccess:    []string{"client_name"},
		ResourceTier: "medium",
		CreatedBy:    "@test:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	if err := store.CreateDefinition(def); err != nil {
		t.Fatalf("failed to create definition: %v", err)
	}

	// Create mock docker client
	mockDocker := &mockDockerClient{}

	// Create factory
	factory := NewAgentFactory(FactoryConfig{
		DockerClient: mockDocker,
		Store:        store,
	})

	// Spawn agent
	ctx := context.Background()
	result, err := factory.Spawn(ctx, &SpawnRequest{
		DefinitionID:    "test-agent-001",
		TaskDescription: "Test task",
		UserID:          "@test:example.com",
	})
	if err != nil {
		t.Fatalf("failed to spawn agent: %v", err)
	}

	// Verify result
	if result.Instance == nil {
		t.Fatal("expected instance in result")
	}

	if result.Instance.Status != StatusRunning {
		t.Errorf("expected status running, got: %s", result.Instance.Status)
	}

	if result.Instance.DefinitionID != "test-agent-001" {
		t.Errorf("expected definition ID test-agent-001, got: %s", result.Instance.DefinitionID)
	}

	// Verify docker calls
	if len(mockDocker.createdContainers) != 1 {
		t.Errorf("expected 1 container created, got: %d", len(mockDocker.createdContainers))
	}

	if len(mockDocker.startedContainers) != 1 {
		t.Errorf("expected 1 container started, got: %d", len(mockDocker.startedContainers))
	}

	// Verify container config
	created := mockDocker.createdContainers[0]
	if created.config.User != "10001:10001" {
		t.Errorf("expected non-root user, got: %s", created.config.User)
	}

	if created.config.Labels["armorclaw.agent_id"] != "test-agent-001" {
		t.Errorf("expected agent_id label, got: %s", created.config.Labels["armorclaw.agent_id"])
	}
}

func TestAgentFactory_Spawn_InactiveDefinition(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create inactive definition
	def := &AgentDefinition{
		ID:           "inactive-agent",
		Name:         "Inactive Agent",
		Skills:       []string{"browser_navigate"},
		ResourceTier: "medium",
		CreatedBy:    "@test:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     false,
	}
	if err := store.CreateDefinition(def); err != nil {
		t.Fatalf("failed to create definition: %v", err)
	}

	factory := NewAgentFactory(FactoryConfig{
		DockerClient: &mockDockerClient{},
		Store:        store,
	})

	ctx := context.Background()
	_, err = factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: "inactive-agent",
		UserID:       "@test:example.com",
	})

	if err == nil {
		t.Error("expected error for inactive definition")
	}
}

func TestAgentFactory_Spawn_NonexistentDefinition(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	factory := NewAgentFactory(FactoryConfig{
		DockerClient: &mockDockerClient{},
		Store:        store,
	})

	ctx := context.Background()
	_, err = factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: "nonexistent",
		UserID:       "@test:example.com",
	})

	if err == nil {
		t.Error("expected error for nonexistent definition")
	}
}

func TestAgentFactory_Spawn_WithKeystore(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "pii-agent",
		Name:         "PII Agent",
		Skills:       []string{"browser_navigate"},
		PIIAccess:    []string{"client_name", "client_email"},
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
	mockKeystore := &mockKeystore{
		unsealed: true,
		secrets: map[string]string{
			"client_name":  "John Doe",
			"client_email": "john@example.com",
		},
	}

	factory := NewAgentFactory(FactoryConfig{
		DockerClient: mockDocker,
		Store:        store,
		Keystore:     mockKeystore,
	})

	ctx := context.Background()
	result, err := factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: "pii-agent",
		UserID:       "@test:example.com",
	})
	if err != nil {
		t.Fatalf("failed to spawn agent: %v", err)
	}

	// Check that PII was injected
	created := mockDocker.createdContainers[0]
	foundName := false
	foundEmail := false
	for _, env := range created.config.Env {
		if env == "PII_CLIENT_NAME=John Doe" {
			foundName = true
		}
		if env == "PII_CLIENT_EMAIL=john@example.com" {
			foundEmail = true
		}
	}

	if !foundName {
		t.Error("expected PII_CLIENT_NAME to be injected")
	}
	if !foundEmail {
		t.Error("expected PII_CLIENT_EMAIL to be injected")
	}

	// Should have no warnings since all PII fields were found
	if len(result.Warnings) > 0 {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
}

func TestAgentFactory_Stop(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create definition
	def := &AgentDefinition{
		ID:           "stop-test-agent",
		Name:         "Stop Test Agent",
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
	})

	// Spawn first
	ctx := context.Background()
	result, err := factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: "stop-test-agent",
		UserID:       "@test:example.com",
	})
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	// Stop
	err = factory.Stop(ctx, result.Instance.ID, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	// Verify docker stop was called
	if len(mockDocker.stoppedContainers) != 1 {
		t.Errorf("expected 1 container stopped, got: %d", len(mockDocker.stoppedContainers))
	}

	// Verify instance status updated
	instance, _ := store.GetInstance(result.Instance.ID)
	if instance.Status != StatusCompleted {
		t.Errorf("expected status completed, got: %s", instance.Status)
	}
}

func TestAgentFactory_Remove(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "remove-test-agent",
		Name:         "Remove Test Agent",
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
	})

	ctx := context.Background()
	result, err := factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: "remove-test-agent",
		UserID:       "@test:example.com",
	})
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	// Remove
	err = factory.Remove(ctx, result.Instance.ID)
	if err != nil {
		t.Fatalf("failed to remove: %v", err)
	}

	// Verify docker remove was called
	if len(mockDocker.removedContainers) != 1 {
		t.Errorf("expected 1 container removed, got: %d", len(mockDocker.removedContainers))
	}

	// Verify instance is gone from store
	_, err = store.GetInstance(result.Instance.ID)
	if err == nil {
		t.Error("expected error getting deleted instance")
	}
}

func TestAgentFactory_GetStatus(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "status-test-agent",
		Name:         "Status Test Agent",
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

	mockDocker := &mockDockerClient{
		containerState: &types.ContainerState{Running: true, ExitCode: 0},
	}
	factory := NewAgentFactory(FactoryConfig{
		DockerClient: mockDocker,
		Store:        store,
	})

	ctx := context.Background()
	result, _ := factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: "status-test-agent",
		UserID:       "@test:example.com",
	})

	// Get status - should be running
	instance, err := factory.GetStatus(ctx, result.Instance.ID)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if instance.Status != StatusRunning {
		t.Errorf("expected status running, got: %s", instance.Status)
	}
}

func TestTruncateLabel(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a very long string that needs truncation", 20, "this is a very lo..."},
		{"", 10, ""},
	}

	for _, test := range tests {
		result := truncateLabel(test.input, test.maxLen)
		if result != test.expected {
			t.Errorf("truncateLabel(%q, %d) = %q, expected %q", test.input, test.maxLen, result, test.expected)
		}
		if len(result) > test.maxLen {
			t.Errorf("result too long: %q (max %d)", result, test.maxLen)
		}
	}
}
