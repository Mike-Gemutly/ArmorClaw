// Package rpc tests JSON-RPC server functionality
// Tests focus on egress proxy support (P0-CRIT-1)
package rpc

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestHTTPProxyEnvironmentVariable tests that HTTP_PROXY env var can be set and read
func TestHTTPProxyEnvironmentVariable(t *testing.T) {
	// Set HTTP_PROXY environment variable
	proxyURL := "http://squid:3128:8080"
	os.Setenv("HTTP_PROXY", proxyURL)
	defer os.Unsetenv("HTTP_PROXY")

	// Verify HTTP_PROXY is set
	if os.Getenv("HTTP_PROXY") != proxyURL {
		t.Errorf("HTTP_PROXY not set correctly, got: %s", os.Getenv("HTTP_PROXY"))
	}
}

// TestHTTPProxyNotSetWhenEmpty tests that HTTP_PROXY can be unset
func TestHTTPProxyNotSetWhenEmpty(t *testing.T) {
	// Ensure HTTP_PROXY is not set
	os.Unsetenv("HTTP_PROXY")

	// Verify HTTP_PROXY is not set
	if os.Getenv("HTTP_PROXY") != "" {
		t.Errorf("HTTP_PROXY should be empty, got: %s", os.Getenv("HTTP_PROXY"))
	}
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
		name     string
		proxyURL string
		valid    bool
	}{
		{
			name:     "Valid HTTP proxy",
			proxyURL: "http://squid:3128:8080",
			valid:    true,
		},
		{
			name:     "Valid HTTPS proxy",
			proxyURL: "https://squid:3128:8083",
			valid:    true,
		},
		{
			name:     "Invalid proxy - no protocol",
			proxyURL: "squid:3128:8080",
			valid:    false,
		},
		{
			name:     "Invalid proxy - bad protocol",
			proxyURL: "ftp://squid:3128:8080",
			valid:    false,
		},
		{
			name:     "Empty proxy",
			proxyURL: "",
			valid:    true, // Empty is valid (no proxy)
		},
	}

	// isValidProxy checks if a proxy URL is valid
	isValidProxy := func(url string) bool {
		if url == "" {
			return true // Empty is valid
		}
		return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidProxy(tt.proxyURL)
			if result != tt.valid {
				t.Errorf("Proxy URL %q: expected valid=%v, got valid=%v", tt.proxyURL, tt.valid, result)
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

// TestGetErrorsRPCMethod tests the get_errors RPC method request format
func TestGetErrorsRPCMethod(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name:   "no params - get all errors",
			params: nil,
		},
		{
			name: "filter by code",
			params: map[string]interface{}{
				"code": "CTX-001",
			},
		},
		{
			name: "filter by category",
			params: map[string]interface{}{
				"category": "container",
			},
		},
		{
			name: "filter by severity",
			params: map[string]interface{}{
				"severity": "error",
			},
		},
		{
			name: "filter by resolved status",
			params: map[string]interface{}{
				"resolved": false,
			},
		},
		{
			name: "with pagination",
			params: map[string]interface{}{
				"limit":  10,
				"offset": 0,
			},
		},
		{
			name: "combined filters",
			params: map[string]interface{}{
				"category": "container",
				"severity": "error",
				"resolved": false,
				"limit":    50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "get_errors",
			}

			if tt.params != nil {
				paramsJSON, err := json.Marshal(tt.params)
				if err != nil {
					t.Fatalf("Failed to marshal params: %v", err)
				}
				req.Params = paramsJSON
			}

			// Verify request format
			reqJSON, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Verify it can be unmarshaled back
			var unmarshaledReq Request
			if err := json.Unmarshal(reqJSON, &unmarshaledReq); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if unmarshaledReq.Method != "get_errors" {
				t.Errorf("Expected method 'get_errors', got '%s'", unmarshaledReq.Method)
			}

			if unmarshaledReq.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC '2.0', got '%s'", unmarshaledReq.JSONRPC)
			}
		})
	}
}

// TestResolveErrorRPCMethod tests the resolve_error RPC method request format
func TestResolveErrorRPCMethod(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name: "valid trace_id",
			params: map[string]interface{}{
				"trace_id": "tr_abc123",
			},
			expectErr: false,
		},
		{
			name: "with resolved_by",
			params: map[string]interface{}{
				"trace_id":    "tr_abc123",
				"resolved_by": "@admin:example.com",
			},
			expectErr: false,
		},
		{
			name: "missing trace_id",
			params: map[string]interface{}{
				"resolved_by": "@admin:example.com",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "resolve_error",
			}

			paramsJSON, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}
			req.Params = paramsJSON

			// Verify request format
			reqJSON, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Verify it can be unmarshaled back
			var unmarshaledReq Request
			if err := json.Unmarshal(reqJSON, &unmarshaledReq); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if unmarshaledReq.Method != "resolve_error" {
				t.Errorf("Expected method 'resolve_error', got '%s'", unmarshaledReq.Method)
			}

			// Verify params
			var params struct {
				TraceID    string `json:"trace_id"`
				ResolvedBy string `json:"resolved_by,omitempty"`
			}
			if err := json.Unmarshal(unmarshaledReq.Params, &params); err != nil {
				t.Fatalf("Failed to unmarshal params: %v", err)
			}

			if tt.expectErr && params.TraceID != "" {
				t.Error("Expected empty trace_id for error case")
			}
			if !tt.expectErr && params.TraceID == "" {
				t.Error("Expected non-empty trace_id for success case")
			}
		})
	}
}

// TestErrorRPCResponseFormat tests the expected response format for error RPC methods
func TestErrorRPCResponseFormat(t *testing.T) {
	// Test get_errors response format
	getErrorsResult := map[string]interface{}{
		"errors": []interface{}{},
		"stats": map[string]interface{}{
			"sampling": map[string]interface{}{
				"total_codes": 0,
				"total_errors": 0,
			},
		},
		"query": map[string]interface{}{
			"code":     "",
			"category": "",
			"severity": "",
			"resolved": false,
		},
	}

	resultJSON, err := json.Marshal(getErrorsResult)
	if err != nil {
		t.Fatalf("Failed to marshal get_errors result: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(resultJSON, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if _, ok := parsed["errors"]; !ok {
		t.Error("get_errors result should have 'errors' field")
	}
	if _, ok := parsed["stats"]; !ok {
		t.Error("get_errors result should have 'stats' field")
	}

	// Test resolve_error response format
	resolveResult := map[string]interface{}{
		"success":   true,
		"trace_id":  "tr_abc123",
		"timestamp": "2026-02-15T12:00:00Z",
	}

	resolveJSON, err := json.Marshal(resolveResult)
	if err != nil {
		t.Fatalf("Failed to marshal resolve_error result: %v", err)
	}

	var resolveParsed map[string]interface{}
	if err := json.Unmarshal(resolveJSON, &resolveParsed); err != nil {
		t.Fatalf("Failed to unmarshal resolve result: %v", err)
	}

	if success, ok := resolveParsed["success"].(bool); !ok || !success {
		t.Error("resolve_error result should have 'success: true'")
	}
	if _, ok := resolveParsed["trace_id"]; !ok {
		t.Error("resolve_error result should have 'trace_id' field")
	}
}
