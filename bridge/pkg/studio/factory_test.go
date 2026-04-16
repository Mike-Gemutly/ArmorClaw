package studio

import (
	"context"
	"fmt"
	"os"
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
	id         string
	config     *container.Config
	hostConfig *container.HostConfig
	name       string
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig any, platform any, name string) (container.CreateResponse, error) {
	id := "container-" + name
	m.createdContainers = append(m.createdContainers, mockContainer{
		id:         id,
		config:     config,
		hostConfig: hostConfig,
		name:       name,
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
	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store: store})

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

	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: &mockDockerClient{},
		Store: store})

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

	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: &mockDockerClient{},
		Store: store})

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

	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store:    store,
		Keystore: mockKeystore})

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
	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store: store})

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
	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store: store})

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
	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store: store})

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

//=============================================================================
// Layer 1 Feature Tests
//=============================================================================

func TestGetRunningInstance_ReturnsRunningInstance(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "running-inst-def",
		Name:         "Running Instance Test",
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

	now := time.Now()
	instance := &AgentInstance{
		ID:           "running-inst-001",
		DefinitionID: def.ID,
		ContainerID:  "container-running-inst-001",
		Status:       StatusRunning,
		RoomID:       "!room:test.example.com",
		SpawnedBy:    "@test:example.com",
		StartedAt:    &now,
	}
	if err := store.CreateInstance(instance); err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: &mockDockerClient{},
		Store: store})

	result, err := factory.GetRunningInstance(def.ID)
	if err != nil {
		t.Fatalf("GetRunningInstance returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil running instance, got nil")
	}
	if result.ID != "running-inst-001" {
		t.Errorf("expected instance ID running-inst-001, got: %s", result.ID)
	}
	if result.RoomID != "!room:test.example.com" {
		t.Errorf("expected RoomID !room:test.example.com, got: %s", result.RoomID)
	}
}

func TestGetRunningInstance_ReturnsNilWhenNoRunningInstance(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "no-running-def",
		Name:         "No Running Test",
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

	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: &mockDockerClient{},
		Store: store})

	result, err := factory.GetRunningInstance(def.ID)
	if err != nil {
		t.Fatalf("GetRunningInstance returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil instance, got: %+v", result)
	}
}

func TestGetRunningInstance_ReturnsNilWhenInstanceStopped(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "stopped-inst-def",
		Name:         "Stopped Instance Test",
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

	now := time.Now()
	instance := &AgentInstance{
		ID:           "stopped-inst-001",
		DefinitionID: def.ID,
		ContainerID:  "container-stopped-inst-001",
		Status:       StatusCompleted,
		SpawnedBy:    "@test:example.com",
		StartedAt:    &now,
		CompletedAt:  &now,
	}
	if err := store.CreateInstance(instance); err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: &mockDockerClient{},
		Store: store})

	result, err := factory.GetRunningInstance(def.ID)
	if err != nil {
		t.Fatalf("GetRunningInstance returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for stopped instance, got: %+v", result)
	}
}

func TestSpawn_RoomIDPersisted(t *testing.T) {

	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "roomid-persist-def",
		Name:         "RoomID Persist Test",
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
	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store: store})

	ctx := context.Background()
	result, err := factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: def.ID,
		RoomID:       "!room:test.example.com",
		UserID:       "@test:example.com",
	})
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	if result.Instance.RoomID != "!room:test.example.com" {
		t.Errorf("expected RoomID !room:test.example.com in result, got: %q", result.Instance.RoomID)
	}

	stored, err := store.GetInstance(result.Instance.ID)
	if err != nil {
		t.Fatalf("failed to get instance from store: %v", err)
	}
	if stored.RoomID != "!room:test.example.com" {
		t.Errorf("expected RoomID !room:test.example.com in store, got: %q", stored.RoomID)
	}
}

func TestSpawn_StateDirBindMount(t *testing.T) {

	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "bindmount-def",
		Name:         "BindMount Test",
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
	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store: store})

	ctx := context.Background()
	_, err = factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: def.ID,
		UserID:       "@test:example.com",
	})
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	if len(mockDocker.createdContainers) != 1 {
		t.Fatalf("expected 1 container created, got: %d", len(mockDocker.createdContainers))
	}

	created := mockDocker.createdContainers[0]
	expectedBind := fmt.Sprintf("%s/agent-state/%s:/home/claw/.openclaw", os.TempDir(), def.ID)
	found := false
	for _, bind := range created.hostConfig.Binds {
		if bind == expectedBind {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected bind mount %q, got: %v", expectedBind, created.hostConfig.Binds)
	}
}

func TestSpawn_EmptyRoomID(t *testing.T) {

	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "empty-roomid-def",
		Name:         "Empty RoomID Test",
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
	factory := NewAgentFactory(FactoryConfig{StateDir: os.TempDir(), DockerClient: mockDocker,
		Store: store})

	ctx := context.Background()
	result, err := factory.Spawn(ctx, &SpawnRequest{
		DefinitionID: def.ID,
		RoomID:       "",
		UserID:       "@test:example.com",
	})
	if err != nil {
		t.Fatalf("failed to spawn with empty RoomID: %v", err)
	}

	if result.Instance.RoomID != "" {
		t.Errorf("expected empty RoomID, got: %q", result.Instance.RoomID)
	}

	stored, err := store.GetInstance(result.Instance.ID)
	if err != nil {
		t.Fatalf("failed to get instance from store: %v", err)
	}
	if stored.RoomID != "" {
		t.Errorf("expected empty RoomID in store, got: %q", stored.RoomID)
	}
}

func TestStore_RoomIDRoundTrip(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "roomid-store-test",
		Name:         "RoomID Store Test",
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

	now := time.Now()
	inst := &AgentInstance{
		ID:           "inst-roomid-test",
		DefinitionID: def.ID,
		Status:       StatusRunning,
		RoomID:       "!room:store.test",
		SpawnedBy:    "@test:example.com",
		StartedAt:    &now,
	}
	if err := store.CreateInstance(inst); err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	retrieved, err := store.GetInstance(inst.ID)
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}
	if retrieved.RoomID != "!room:store.test" {
		t.Errorf("expected RoomID !room:store.test, got: %q", retrieved.RoomID)
	}

	retrieved.RoomID = "!new-room:test"
	if err := store.UpdateInstance(retrieved); err != nil {
		t.Fatalf("failed to update instance: %v", err)
	}

	updated, err := store.GetInstance(inst.ID)
	if err != nil {
		t.Fatalf("failed to get updated instance: %v", err)
	}
	if updated.RoomID != "!new-room:test" {
		t.Errorf("expected updated RoomID !new-room:test, got: %q", updated.RoomID)
	}
}

func TestStore_EmptyRoomID(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "empty-roomid-store-test",
		Name:         "Empty RoomID Store Test",
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

	now := time.Now()
	inst := &AgentInstance{
		ID:           "inst-empty-roomid-test",
		DefinitionID: def.ID,
		Status:       StatusRunning,
		RoomID:       "",
		SpawnedBy:    "@test:example.com",
		StartedAt:    &now,
	}
	if err := store.CreateInstance(inst); err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	retrieved, err := store.GetInstance(inst.ID)
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}
	if retrieved.RoomID != "" {
		t.Errorf("expected empty RoomID, got: %q", retrieved.RoomID)
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
