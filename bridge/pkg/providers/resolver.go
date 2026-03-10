package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// LoadFromEnvironment loads the provider registry with ENV override priority
// Priority: ARMORCLAW_PROVIDERS_URL > DefaultRegistryPath > EmbeddedRegistry
func LoadFromEnvironment() (*Registry, error) {
	// Priority 1: ENV override
	if envURL := os.Getenv("ARMORCLAW_PROVIDERS_URL"); envURL != "" {
		registry, err := fetchRemoteRegistry(envURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to fetch remote registry from %s: %v. Falling back to local/embedded.\n", envURL, err)
		} else {
			return registry, nil
		}
	}

	// Priority 2: Local registry file
	return LoadRegistry(DefaultRegistryPath)
}

// fetchRemoteRegistry downloads and parses a registry from a URL
func fetchRemoteRegistry(url string) (*Registry, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var registry Registry
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return &registry, nil
}

// LoadDefaultRegistry loads the default registry
func LoadDefaultRegistry() (*Registry, error) {
	return LoadRegistry(DefaultRegistryPath)
}

// LoadEmbeddedRegistry loads the embedded registry (fallback)
func LoadEmbeddedRegistry() *Registry {
	return &EmbeddedRegistry
}
