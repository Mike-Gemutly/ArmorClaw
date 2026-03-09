package providers

import (
	"testing"
)

func TestLoadRegistry(t *testing.T) {
	// Test with embedded registry (should work without file)
	registry := LoadEmbeddedRegistry()

	if registry == nil {
		t.Fatal("Embedded registry should not be nil")
	}

	if registry.GetProviderCount() == 0 {
		t.Error("Embedded registry should have providers")
	}

	// Test provider resolution
	zhipu, found := registry.ResolveProvider("zhipu")
	if !found {
		t.Error("Should find zhipu provider by ID")
	}
	if zhipu.Name != "Zhipu AI (Z AI)" {
		t.Errorf("Expected 'Zhipu AI (Z AI)', got '%s'", zhipu.Name)
	}

	// Test alias resolution
	zai, found := registry.ResolveProvider("zai")
	if !found {
		t.Error("Should find zhipu provider by alias 'zai'")
	}
	if zai.ID != "zhipu" {
		t.Errorf("Expected 'zhipu', got '%s'", zai.ID)
	}

	// Test moonshot by ID
	moonshot, found := registry.ResolveProvider("moonshot")
	if !found {
		t.Error("Should find moonshot provider by ID")
	}
	if moonshot.BaseURL != "https://api.moonshot.ai/v1" {
		t.Errorf("Expected 'https://api.moonshot.ai/v1', got '%s'", moonshot.BaseURL)
	}

	// Test non-existent provider
	_, found = registry.ResolveProvider("nonexistent")
	if found {
		t.Error("Should not find nonexistent provider")
	}
}

func TestGetAllProviders(t *testing.T) {
	registry := LoadEmbeddedRegistry()
	providers := registry.GetAllProviders()

	if len(providers) == 0 {
		t.Error("GetAllProviders should return non-empty map")
	}

	// Check that zhipu is in the map
	if _, found := providers["zhipu"]; !found {
		t.Error("zhipu should be in providers map")
	}
}

func TestGetProviderCount(t *testing.T) {
	registry := LoadEmbeddedRegistry()
	count := registry.GetProviderCount()

	if count == 0 {
		t.Error("Provider count should not be zero")
	}

	expectedCount := 12
	if count != expectedCount {
		t.Errorf("Expected %d providers, got %d", expectedCount, count)
	}
}
