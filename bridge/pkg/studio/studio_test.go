package studio

import (
	"fmt"
	"os"
	"testing"
	"time"
)

//=============================================================================
// Store Tests
//=============================================================================

func TestNewStore(t *testing.T) {
	// Use in-memory database
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected store to be created")
	}
}

func TestStoreWithFile(t *testing.T) {
	// Create temp file
	tmpFile := "/tmp/studio_test_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	store, err := NewStore(StoreConfig{Path: tmpFile})
	if err != nil {
		t.Fatalf("failed to create store with file: %v", err)
	}
	defer store.Close()

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("expected database file to be created")
	}
}

//=============================================================================
// Skill Registry Tests
//=============================================================================

func TestDefaultSkills(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	skills, err := store.ListSkills("")
	if err != nil {
		t.Fatalf("failed to list skills: %v", err)
	}

	// Should have default skills
	if len(skills) < 5 {
		t.Errorf("expected at least 5 default skills, got %d", len(skills))
	}

	// Check for specific default skills
	expectedSkills := []string{"browser_navigate", "form_filler", "pdf_generator", "email_sender"}
	for _, expected := range expectedSkills {
		found := false
		for _, skill := range skills {
			if skill.ID == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected default skill %s not found", expected)
		}
	}
}

func TestSkillRegistry_Validate(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	registry := NewSkillRegistry(store)

	// Test valid skills
	result := registry.Validate([]string{"browser_navigate", "pdf_generator"})
	if !result.Valid {
		t.Errorf("expected valid result for existing skills, got: %s", result.Message)
	}

	// Test invalid skills
	result = registry.Validate([]string{"browser_navigate", "nonexistent_skill"})
	if result.Valid {
		t.Error("expected invalid result for nonexistent skill")
	}
	if len(result.InvalidIDs) != 1 || result.InvalidIDs[0] != "nonexistent_skill" {
		t.Errorf("expected nonexistent_skill in InvalidIDs, got: %v", result.InvalidIDs)
	}
}

func TestSkillRegistry_Exists(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	registry := NewSkillRegistry(store)

	if !registry.Exists("browser_navigate") {
		t.Error("expected browser_navigate to exist")
	}

	if registry.Exists("nonexistent") {
		t.Error("expected nonexistent skill to not exist")
	}
}

func TestSkillRegistry_Register(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	registry := NewSkillRegistry(store)

	// Register new skill
	skill := &Skill{
		ID:          "custom_skill",
		Name:        "Custom Skill",
		Description: "A custom skill for testing",
		Category:    "custom",
		CreatedAt:   time.Now(),
	}

	if err := registry.Register(skill); err != nil {
		t.Fatalf("failed to register skill: %v", err)
	}

	// Verify it was registered
	if !registry.Exists("custom_skill") {
		t.Error("expected custom_skill to exist after registration")
	}

	// Get the skill
	retrieved, err := registry.Get("custom_skill")
	if err != nil {
		t.Fatalf("failed to get registered skill: %v", err)
	}

	if retrieved.Name != "Custom Skill" {
		t.Errorf("expected name 'Custom Skill', got: %s", retrieved.Name)
	}
}

//=============================================================================
// PII Registry Tests
//=============================================================================

func TestDefaultPIIFields(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	fields, err := store.ListPIIFields("")
	if err != nil {
		t.Fatalf("failed to list PII fields: %v", err)
	}

	// Should have default PII fields
	if len(fields) < 5 {
		t.Errorf("expected at least 5 default PII fields, got %d", len(fields))
	}

	// Check for specific default fields
	expectedFields := []string{"client_name", "client_email", "client_ssn"}
	for _, expected := range expectedFields {
		found := false
		for _, field := range fields {
			if field.ID == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected default PII field %s not found", expected)
		}
	}
}

func TestPIIRegistry_Validate(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	registry := NewPIIRegistry(store)

	// Test valid fields
	result := registry.Validate([]string{"client_name", "client_email"})
	if !result.Valid {
		t.Errorf("expected valid result for existing fields, got: %s", result.Message)
	}

	// Check requires_approval
	if len(result.RequiresApproval) < 1 {
		t.Error("expected client_email to require approval")
	}

	// Test invalid fields
	result = registry.Validate([]string{"client_name", "nonexistent_field"})
	if result.Valid {
		t.Error("expected invalid result for nonexistent field")
	}
	if len(result.InvalidIDs) != 1 || result.InvalidIDs[0] != "nonexistent_field" {
		t.Errorf("expected nonexistent_field in InvalidIDs, got: %v", result.InvalidIDs)
	}
}

func TestPIIRegistry_Sensitivity(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	registry := NewPIIRegistry(store)

	// Test low sensitivity (no approval required)
	nameField, err := registry.Get("client_name")
	if err != nil {
		t.Fatalf("failed to get client_name: %v", err)
	}
	if nameField.RequiresApproval {
		t.Error("expected client_name (low sensitivity) to not require approval")
	}

	// Test critical sensitivity (approval required)
	ssnField, err := registry.Get("client_ssn")
	if err != nil {
		t.Fatalf("failed to get client_ssn: %v", err)
	}
	if !ssnField.RequiresApproval {
		t.Error("expected client_ssn (critical sensitivity) to require approval")
	}
}

//=============================================================================
// Agent Definition Tests
//=============================================================================

func TestCreateDefinition(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	def := &AgentDefinition{
		ID:           "test-agent-001",
		Name:         "Test Agent",
		Description:  "A test agent",
		Skills:       []string{"browser_navigate", "pdf_generator"},
		PIIAccess:    []string{"client_name", "client_email"},
		ResourceTier: "medium",
		CreatedBy:    "@admin:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}

	if err := store.CreateDefinition(def); err != nil {
		t.Fatalf("failed to create definition: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetDefinition("test-agent-001")
	if err != nil {
		t.Fatalf("failed to get definition: %v", err)
	}

	if retrieved.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got: %s", retrieved.Name)
	}

	if len(retrieved.Skills) != 2 {
		t.Errorf("expected 2 skills, got: %d", len(retrieved.Skills))
	}

	if !retrieved.IsActive {
		t.Error("expected IsActive to be true")
	}
}

func TestListDefinitions(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create multiple definitions
	for i := 1; i <= 3; i++ {
		def := &AgentDefinition{
			ID:           fmt.Sprintf("agent-%d", i),
			Name:         fmt.Sprintf("Agent %d", i),
			Skills:       []string{"browser_navigate"},
			PIIAccess:    []string{"client_name"},
			ResourceTier: "medium",
			CreatedBy:    "@admin:example.com",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     i != 3, // Third one is inactive
		}
		if err := store.CreateDefinition(def); err != nil {
			t.Fatalf("failed to create definition %d: %v", i, err)
		}
	}

	// List all
	all, err := store.ListDefinitions(false)
	if err != nil {
		t.Fatalf("failed to list definitions: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 definitions, got: %d", len(all))
	}

	// List active only
	active, err := store.ListDefinitions(true)
	if err != nil {
		t.Fatalf("failed to list active definitions: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active definitions, got: %d", len(active))
	}
}

func TestUpdateDefinition(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create definition
	def := &AgentDefinition{
		ID:           "update-test",
		Name:         "Original Name",
		Skills:       []string{"browser_navigate"},
		PIIAccess:    []string{"client_name"},
		ResourceTier: "medium",
		CreatedBy:    "@admin:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	if err := store.CreateDefinition(def); err != nil {
		t.Fatalf("failed to create definition: %v", err)
	}

	// Update definition
	def.Name = "Updated Name"
	def.Skills = []string{"browser_navigate", "pdf_generator"}
	def.ResourceTier = "high"

	if err := store.UpdateDefinition(def); err != nil {
		t.Fatalf("failed to update definition: %v", err)
	}

	// Verify update
	retrieved, err := store.GetDefinition("update-test")
	if err != nil {
		t.Fatalf("failed to get updated definition: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got: %s", retrieved.Name)
	}

	if len(retrieved.Skills) != 2 {
		t.Errorf("expected 2 skills after update, got: %d", len(retrieved.Skills))
	}

	if retrieved.ResourceTier != "high" {
		t.Errorf("expected resource tier 'high', got: %s", retrieved.ResourceTier)
	}
}

func TestDeleteDefinition(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create definition
	def := &AgentDefinition{
		ID:           "delete-test",
		Name:         "To Delete",
		Skills:       []string{"browser_navigate"},
		PIIAccess:    []string{"client_name"},
		ResourceTier: "medium",
		CreatedBy:    "@admin:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	if err := store.CreateDefinition(def); err != nil {
		t.Fatalf("failed to create definition: %v", err)
	}

	// Delete
	if err := store.DeleteDefinition("delete-test"); err != nil {
		t.Fatalf("failed to delete definition: %v", err)
	}

	// Verify deleted
	_, err = store.GetDefinition("delete-test")
	if err == nil {
		t.Error("expected error when getting deleted definition")
	}
}

//=============================================================================
// Agent Instance Tests
//=============================================================================

func TestCreateInstance(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create definition first
	def := &AgentDefinition{
		ID:           "inst-test-def",
		Name:         "Instance Test",
		Skills:       []string{"browser_navigate"},
		PIIAccess:    []string{"client_name"},
		ResourceTier: "medium",
		CreatedBy:    "@admin:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	if err := store.CreateDefinition(def); err != nil {
		t.Fatalf("failed to create definition: %v", err)
	}

	// Create instance
	now := time.Now()
	instance := &AgentInstance{
		ID:              "inst-001",
		DefinitionID:    "inst-test-def",
		ContainerID:     "container-abc123",
		Status:          StatusRunning,
		TaskDescription: "Test task",
		SpawnedBy:       "@admin:example.com",
		StartedAt:       &now,
	}

	if err := store.CreateInstance(instance); err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetInstance("inst-001")
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}

	if retrieved.Status != StatusRunning {
		t.Errorf("expected status running, got: %s", retrieved.Status)
	}

	if retrieved.ContainerID != "container-abc123" {
		t.Errorf("expected container ID 'container-abc123', got: %s", retrieved.ContainerID)
	}
}

//=============================================================================
// Statistics Tests
//=============================================================================

func TestGetStats(t *testing.T) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create some definitions
	for i := 1; i <= 2; i++ {
		tier := "medium"
		if i == 2 {
			tier = "high"
		}
		def := &AgentDefinition{
			ID:           fmt.Sprintf("stats-agent-%d", i),
			Name:         fmt.Sprintf("Stats Agent %d", i),
			Skills:       []string{"browser_navigate"},
			PIIAccess:    []string{"client_name"},
			ResourceTier: tier,
			CreatedBy:    "@admin:example.com",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}
		if err := store.CreateDefinition(def); err != nil {
			t.Fatalf("failed to create definition: %v", err)
		}
	}

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalDefinitions != 2 {
		t.Errorf("expected 2 total definitions, got: %d", stats.TotalDefinitions)
	}

	if stats.ActiveDefinitions != 2 {
		t.Errorf("expected 2 active definitions, got: %d", stats.ActiveDefinitions)
	}

	if stats.SkillsAvailable < 5 {
		t.Errorf("expected at least 5 skills available, got: %d", stats.SkillsAvailable)
	}

	if stats.ByTier["medium"] != 1 {
		t.Errorf("expected 1 medium tier definition, got: %d", stats.ByTier["medium"])
	}

	if stats.ByTier["high"] != 1 {
		t.Errorf("expected 1 high tier definition, got: %d", stats.ByTier["high"])
	}
}

//=============================================================================
// Resource Profile Tests
//=============================================================================

func TestDefaultResourceProfiles(t *testing.T) {
	profiles := DefaultResourceProfiles

	if len(profiles) != 3 {
		t.Errorf("expected 3 default profiles, got: %d", len(profiles))
	}

	// Check low profile
	low := profiles["low"]
	if low.MemoryMB != 256 {
		t.Errorf("expected low profile memory 256MB, got: %d", low.MemoryMB)
	}

	// Check high profile
	high := profiles["high"]
	if high.MemoryMB != 2048 {
		t.Errorf("expected high profile memory 2048MB, got: %d", high.MemoryMB)
	}
}

func TestGetProfile(t *testing.T) {
	// Test valid tier
	profile := GetProfile("medium")
	if profile.Tier != "medium" {
		t.Errorf("expected tier 'medium', got: %s", profile.Tier)
	}

	// Test invalid tier (should return medium default)
	profile = GetProfile("invalid")
	if profile.Tier != "medium" {
		t.Errorf("expected fallback to medium, got: %s", profile.Tier)
	}
}

//=============================================================================
// Wizard State Tests
//=============================================================================

func TestWizardState_IsExpired(t *testing.T) {
	// Not expired
	state := &WizardState{
		UserID:    "@user:example.com",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if state.IsExpired() {
		t.Error("expected wizard state to not be expired")
	}

	// Expired
	state.ExpiresAt = time.Now().Add(-1 * time.Minute)
	if !state.IsExpired() {
		t.Error("expected wizard state to be expired")
	}
}

//=============================================================================
// Sensitivity Tests
//=============================================================================

func TestRequiresApprovalForSensitivity(t *testing.T) {
	tests := []struct {
		sensitivity string
		expected    bool
	}{
		{"low", false},
		{"medium", false},
		{"high", true},
		{"critical", true},
	}

	for _, test := range tests {
		result := RequiresApprovalForSensitivity(test.sensitivity)
		if result != test.expected {
			t.Errorf("RequiresApprovalForSensitivity(%s) = %v, expected %v", test.sensitivity, result, test.expected)
		}
	}
}
