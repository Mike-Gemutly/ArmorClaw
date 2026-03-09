package providers

import (
	"fmt"
	"os"
)

// LoadFromEnvironment loads the provider registry with ENV override priority
// Priority: ARMORCLAW_PROVIDERS_URL > DefaultRegistryPath > EmbeddedRegistry
func LoadFromEnvironment() (*Registry, error) {
	// Priority 1: ENV override
	if envURL := os.Getenv("ARMORCLAW_PROVIDERS_URL"); envURL != "" {
		// Note: For full ENV URL support, we'd need HTTP client code
		// For now, just return the embedded registry
		// TODO: Implement remote registry download
		fmt.Fprintf(os.Stderr, "Warning: ARMORCLAW_PROVIDERS_URL not yet implemented, using embedded registry\n")
		return &EmbeddedRegistry, nil
	}

	// Priority 2: Local registry file
	return LoadRegistry(DefaultRegistryPath)
}

// LoadDefaultRegistry loads the default registry
func LoadDefaultRegistry() (*Registry, error) {
	return LoadRegistry(DefaultRegistryPath)
}

// LoadEmbeddedRegistry loads the embedded registry (fallback)
func LoadEmbeddedRegistry() *Registry {
	return &EmbeddedRegistry
}
