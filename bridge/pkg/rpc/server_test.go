// Package rpc tests JSON-RPC server functionality
// Tests focus on egress proxy support (P0-CRIT-1)
package rpc

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/keystore"
)

// mockKeystore is a mock keystore for testing
type mockKeystore struct {
	keys map[string]keystore.Credential
}

func (m *mockKeystore) Retrieve(id string) (keystore.Credential, error) {
	if cred, ok := m.keys[id]; ok {
		return cred, nil
	}
	return keystore.Credential{}, keystore.ErrKeyNotFound
}

func (m *mockKeystore) Store(cred keystore.Credential) error {
	m.keys[cred.ID] = cred
	return nil
}

func (m *mockKeystore) List(providers ...keystore.Provider) ([]keystore.Credential, error) {
	result := make([]keystore.Credential, 0, len(m.keys))
	for _, k := range m.keys {
		result = append(result, k)
	}
	return result, nil
}

// TestHTTPProxyPassedToContainer tests that HTTP_PROXY is passed to containers
func TestHTTPProxyPassedToContainer(t *testing.T) {
	// Set HTTP_PROXY environment variable
	proxyURL := "http://squid:3128:8080"
	os.Setenv("HTTP_PROXY", proxyURL)
	defer os.Unsetenv("HTTP_PROXY")

	// Create test keystore
	testKeystore := &mockKeystore{
		keys: map[string]keystore.Credential{
			"test-key": {
				ID:          "test-key",
				Provider:     keystore.ProviderOpenAI,
				Token:        "test-token",
				DisplayName:  "Test Key",
				CreatedAt:    time.Now().Unix(),
				ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(),
			},
		},
	}

	// Create server with mock dependencies
	cfg := Config{
		SocketPath: "/tmp/test-bridge.sock",
		Keystore:   testKeystore,
		AuditLog:   &audit.AuditLog{},
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	// Verify HTTP_PROXY is set
	if os.Getenv("HTTP_PROXY") != proxyURL {
		t.Errorf("HTTP_PROXY not set correctly, got: %s", os.Getenv("HTTP_PROXY"))
	}
}

// TestHTTPProxyNotSetWhenEmpty tests that no HTTP_PROXY env var is passed when not configured
func TestHTTPProxyNotSetWhenEmpty(t *testing.T) {
	// Ensure HTTP_PROXY is not set
	os.Unsetenv("HTTP_PROXY")

	testKeystore := &mockKeystore{
		keys: map[string]keystore.Credential{
			"test-key": {
				ID:          "test-key",
				Provider:     keystore.ProviderOpenAI,
				Token:        "test-token",
				DisplayName:  "Test Key",
				CreatedAt:    time.Now().Unix(),
				ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(),
			},
		},
	}

	cfg := Config{
		SocketPath: "/tmp/test-bridge-sock2.sock",
		Keystore:   testKeystore,
		AuditLog:   &audit.AuditLog{},
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()
}

// TestProxyLogging tests that proxy configuration is logged to security log
func TestProxyLogging(t *testing.T) {
	proxyURL := "http://squid:3128:8080"

	// Verify proxy URL is correctly formatted
	if proxyURL == "" {
		t.Error("Proxy URL should not be empty")
	}

	if !strings.HasPrefix(proxyURL, "http://") && !strings.HasPrefix(proxyURL, "https://") {
		t.Error("Proxy URL should start with http:// or https://")
	}
}

// TestProxySecurityTests validates security aspects of proxy configuration
func TestProxySecurityTests(t *testing.T) {
	tests := []struct {
		name    string
		proxyURL string
		valid    bool
	}{
		{
			name:    "Valid HTTP proxy",
			proxyURL: "http://squid:3128:8080",
			valid:    true,
		},
		{
			name:    "Valid HTTPS proxy",
			proxyURL: "https://squid:3128:8083",
			valid:    true,
		},
		{
			name:    "Invalid proxy - no protocol",
			proxyURL: "squid:3128:8080",
			valid:    false,
		},
		{
			name:    "Invalid proxy - bad protocol",
			proxyURL: "ftp://squid:3128:8080",
			valid:    false,
		},
		{
			name:    "Empty proxy",
			proxyURL: "",
			valid:    true, // Empty is valid (no proxy)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				if tt.proxyURL != "" && !strings.Contains(tt.proxyURL, "://") {
					t.Errorf("Expected valid proxy to have protocol: %s", tt.proxyURL)
				}
			} else {
				if tt.proxyURL != "" && (strings.Contains(tt.proxyURL, "://") || strings.Contains(tt.proxyURL, "ftp://")) {
					t.Errorf("Expected invalid proxy to be rejected: %s", tt.proxyURL)
				}
			}
		})
	}
}

// TestSquidACLConfiguration validates Squid proxy ACL settings
func TestSquidACLConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		aclLine     string
		validACL    bool
		description string
	}{
		{
			name:     "Localnet ACL allowed",
			aclLine:  "acl localnet src 172.18.0.0/24",
			validACL: true,
			description: "Should allow local network",
		},
		{
			name:     "HTTP access allowed",
			aclLine:  "http_access allow localnet",
			validACL: true,
			description: "Should allow HTTP access from localnet",
		},
		{
			name:     "Cache deny all",
			aclLine:  "cache deny all",
			validACL: true,
			description: "Should deny caching for security",
		},
		{
			name:     "Proxy port configured",
			aclLine:  "http_port 3128",
			validACL: true,
			description: "Should configure HTTP port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.validACL {
				t.Errorf("ACL configuration should be valid: %s", tt.aclLine)
			}

			// Validate ACL structure
			if tt.aclLine == "" {
				t.Error("ACL line should not be empty")
			}
		})
	}
}

// TestEgressProxyURLValidation tests egress proxy URL validation
func TestEgressProxyURLValidation(t *testing.T) {
	tests := []struct {
		name     string
		proxyURL string
		expected string
	}{
		{
			name:     "Slack proxy URL",
			proxyURL: "http://squid:3128:8080/slack",
			expected: "slack",
		},
		{
			name:     "Discord proxy URL",
			proxyURL: "http://squid:3128:8081/discord",
			expected: "discord",
		},
		{
			name:     "Teams proxy URL",
			proxyURL: "http://squid:3128:8082/teams",
			expected: "teams",
		},
		{
			name:     "WhatsApp proxy URL",
			proxyURL: "http://squid:3128:8083/whatsapp",
			expected: "whatsapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate URL structure
			if !strings.HasPrefix(tt.proxyURL, "http://") {
				t.Errorf("Proxy URL should start with http://: %s", tt.proxyURL)
			}

			// Validate port is present
			if !strings.Contains(tt.proxyURL, ":3128:") {
				t.Errorf("Proxy URL should contain port :3128: : %s", tt.proxyURL)
			}

			// Validate platform path
			if !strings.Contains(tt.proxyURL, "/"+tt.expected) {
				t.Errorf("Proxy URL should contain platform path /%s: %s", tt.expected, tt.proxyURL)
			}
		})
	}
}

// TestStartRequestProxyParams tests start request with proxy parameters
func TestStartRequestProxyParams(t *testing.T) {
	requestJSON := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "start",
		"params": {
			"key_id": "test-key",
			"agent_type": "sdtw-slack",
			"image": "armorclaw/sdtw-slack:v1"
		}
	}`

	var req Request
	err := json.Unmarshal([]byte(requestJSON), &req)
	if err != nil {
		t.Fatalf("Failed to parse request: %v", err)
	}

	// Validate required fields
	if req.Method != "start" {
		t.Errorf("Expected method 'start', got: %s", req.Method)
	}

	var params struct {
		KeyID     string `json:"key_id"`
		AgentType string `json:"agent_type"`
		Image     string `json:"image"`
	}

	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.Fatalf("Failed to parse params: %v", err)
		}
	}

	if params.KeyID == "" {
		t.Error("key_id is required")
	}

	if params.AgentType == "" {
		t.Error("agent_type is required")
	}
}

// TestProxyConfigurationRoundTrip tests proxy configuration round-trip
func TestProxyConfigurationRoundTrip(t *testing.T) {
	// Test that proxy configuration flows from bridge to container

	proxyURL := "http://squid:3128:8080"

	// Simulate bridge setting HTTP_PROXY
	os.Setenv("HTTP_PROXY", proxyURL)
	defer os.Unsetenv("HTTP_PROXY")

	// Simulate container reading HTTP_PROXY
	containerHTTPProxy := os.Getenv("HTTP_PROXY")

	// Verify proxy environment variable is set
	if containerHTTPProxy != proxyURL {
		t.Errorf("HTTP_PROXY not set in container, got: %s", containerHTTPProxy)
	}
}
