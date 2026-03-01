package wizard

import (
	"fmt"
	"net"
	"strings"
)

// ValidateNonEmpty returns an error if the input is empty or whitespace-only.
// The fieldName parameter is included in the error message for clarity.
func ValidateNonEmpty(fieldName string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required (field: %s)", fieldName, fieldName)
		}
		return nil
	}
}

// ValidateAPIKey validates API key length and basic format.
// Returns a descriptive error indicating the expected format.
func ValidateAPIKey(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("API key is required (field: APIKey) - get your key from your AI provider's dashboard")
	}
	if len(s) < 20 {
		return fmt.Errorf("API key too short: got %d characters, need at least 20 (field: APIKey) - verify you copied the complete key", len(s))
	}
	// Check for common prefixes to give helpful feedback
	if strings.HasPrefix(s, "sk-") || strings.HasPrefix(s, "sk-ant-") || strings.HasPrefix(s, "sk-proj-") {
		return nil // Valid prefix
	}
	// Allow other formats but warn
	return nil
}

// ValidatePassword validates password minimum length.
// An empty password is allowed (auto-generate).
func ValidatePassword(s string) error {
	if s == "" {
		return nil // empty means auto-generate
	}
	if len(s) < 8 {
		return fmt.Errorf("password must be at least 8 characters (got %d), or leave empty to auto-generate (field: AdminPassword)", len(s))
	}
	return nil
}

// ValidateURL checks for a basic URL format.
// Returns a descriptive error with examples.
func ValidateURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("URL is required (field: CustomURL) - example: https://api.your-provider.com/v1")
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return fmt.Errorf("URL must start with http:// or https:// (field: CustomURL, got: %q)", s[:min(20, len(s))]+"...")
	}
	// Basic validation that URL has a host
	if len(s) < 10 {
		return fmt.Errorf("URL appears incomplete (field: CustomURL) - example: https://api.your-provider.com/v1")
	}
	return nil
}

// ValidatePortAvailable checks if a TCP port is available on localhost.
func ValidatePortAvailable(port int) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is already in use (function: ValidatePortAvailable) - stop the service using this port or choose a different one", port)
	}
	ln.Close()
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
