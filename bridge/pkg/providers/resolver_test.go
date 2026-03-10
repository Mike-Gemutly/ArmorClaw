package providers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchRemoteRegistry(t *testing.T) {
	// Create a test registry
	testRegistry := Registry{
		Providers: []Provider{
			{
				ID:      "test-remote",
				Name:    "Test Remote Provider",
				BaseURL: "https://api.test.com",
			},
		},
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testRegistry)
	}))
	defer server.Close()

	// Test fetchRemoteRegistry
	registry, err := fetchRemoteRegistry(server.URL)
	if err != nil {
		t.Fatalf("fetchRemoteRegistry failed: %v", err)
	}

	if registry.GetProviderCount() != 1 {
		t.Errorf("Expected 1 provider, got %d", registry.GetProviderCount())
	}

	p, found := registry.ResolveProvider("test-remote")
	if !found {
		t.Error("Should find remote provider")
	}
	if p.Name != "Test Remote Provider" {
		t.Errorf("Expected 'Test Remote Provider', got '%s'", p.Name)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	// Create a test registry
	testRegistry := Registry{
		Providers: []Provider{
			{
				ID:      "env-remote",
				Name:    "Env Remote Provider",
				BaseURL: "https://api.env.com",
			},
		},
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testRegistry)
	}))
	defer server.Close()

	// Set environment variable
	os.Setenv("ARMORCLAW_PROVIDERS_URL", server.URL)
	defer os.Unsetenv("ARMORCLAW_PROVIDERS_URL")

	// Test LoadFromEnvironment
	registry, err := LoadFromEnvironment()
	if err != nil {
		t.Fatalf("LoadFromEnvironment failed: %v", err)
	}

	if registry.GetProviderCount() != 1 {
		t.Errorf("Expected 1 provider, got %d", registry.GetProviderCount())
	}

	p, found := registry.ResolveProvider("env-remote")
	if !found {
		t.Error("Should find remote provider from env")
	}
	_ = p
}
