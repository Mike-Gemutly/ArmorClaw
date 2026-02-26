package wizard

import (
	"fmt"
	"net"
	"strings"
)

// ValidateNonEmpty returns an error if the input is empty or whitespace-only.
func ValidateNonEmpty(fieldName string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
		return nil
	}
}

// ValidateAPIKey validates API key length and basic format.
func ValidateAPIKey(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("API key is required")
	}
	if len(s) < 20 {
		return fmt.Errorf("API key too short (minimum 20 characters)")
	}
	return nil
}

// ValidatePassword validates password minimum length.
// An empty password is allowed (auto-generate).
func ValidatePassword(s string) error {
	if s == "" {
		return nil // empty means auto-generate
	}
	if len(s) < 8 {
		return fmt.Errorf("password must be at least 8 characters (or leave empty to auto-generate)")
	}
	return nil
}

// ValidateURL checks for a basic URL format.
func ValidateURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("URL is required")
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	return nil
}

// ValidatePortAvailable checks if a TCP port is available on localhost.
func ValidatePortAvailable(port int) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is already in use", port)
	}
	ln.Close()
	return nil
}
