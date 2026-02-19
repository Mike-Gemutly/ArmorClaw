// Package dashboard provides integration tests for the web dashboard
package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewDashboardServer tests dashboard server creation
func TestNewDashboardServer(t *testing.T) {
	config := DefaultDashboardConfig()
	config.Addr = ":0" // Random port

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	if server.config.Addr != ":0" {
		t.Errorf("Expected addr :0, got %s", server.config.Addr)
	}
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", resp["status"])
	}

	if resp["version"] == nil {
		t.Error("Response should include version")
	}
}

// TestAPIStatusEndpoint tests the status API endpoint
func TestAPIStatusEndpoint(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	// Update stats
	server.UpdateStats(DashboardStats{
		Uptime:          time.Now(),
		Version:         "1.0.0",
		ContainersActive: 5,
		LicenseStatus:   "active",
		HealthStatus:    "healthy",
	})

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	// Use handler directly (no auth middleware)
	server.handleAPIStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats DashboardStats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if stats.ContainersActive != 5 {
		t.Errorf("Expected 5 active containers, got %d", stats.ContainersActive)
	}
}

// TestAPIContainersEndpoint tests the containers API endpoint
func TestAPIContainersEndpoint(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false
	config.ShowContainers = true

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/containers", nil)
	w := httptest.NewRecorder()

	server.handleAPIContainers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var containers []ContainerInfo
	if err := json.Unmarshal(w.Body.Bytes(), &containers); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should return at least mock data
	if len(containers) == 0 {
		t.Log("Warning: No containers returned")
	}
}

// TestAPIAuditEndpoint tests the audit API endpoint
func TestAPIAuditEndpoint(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false
	config.ShowAuditLogs = true

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/audit", nil)
	w := httptest.NewRecorder()

	server.handleAPIAudit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var entries []AuditLogEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
}

// TestAPILicenseEndpoint tests the license API endpoint
func TestAPILicenseEndpoint(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false
	config.ShowLicense = true

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/license", nil)
	w := httptest.NewRecorder()

	server.handleAPILicense(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var license LicenseInfo
	if err := json.Unmarshal(w.Body.Bytes(), &license); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if license.LicenseID == "" {
		t.Error("License ID should not be empty")
	}

	if license.Tier == "" {
		t.Error("License tier should not be empty")
	}
}

// TestAPISystemEndpoint tests the system info API endpoint
func TestAPISystemEndpoint(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/system", nil)
	w := httptest.NewRecorder()

	server.handleAPISystem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var info SystemInfo
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if info.Hostname == "" {
		t.Error("Hostname should not be empty")
	}
}

// TestAuthMiddleware tests authentication middleware
func TestAuthMiddleware(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = true
	config.AdminToken = "secret-token-123"

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	tests := []struct {
		name   string
		token  string
		want   int
	}{
		{
			name:  "valid token",
			token: "secret-token-123",
			want:  http.StatusOK,
		},
		{
			name:  "invalid token",
			token: "wrong-token",
			want:  http.StatusUnauthorized,
		},
		{
			name:  "no token",
			token: "",
			want:  http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/status", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			w := httptest.NewRecorder()

			handler := server.authMiddleware(server.handleAPIStatus)
			handler(w, req)

			if w.Code != tt.want {
				t.Errorf("Expected status %d, got %d", tt.want, w.Code)
			}
		})
	}
}

// TestAuthWithCookie tests authentication via session cookie
func TestAuthWithCookie(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = true
	config.AdminToken = "cookie-token"

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "cookie-token",
	})
	w := httptest.NewRecorder()

	handler := server.authMiddleware(server.handleAPIStatus)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with valid cookie, got %d", w.Code)
	}
}

// TestDisabledFeatures tests that disabled features return 403
func TestDisabledFeatures(t *testing.T) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false
	config.ShowContainers = false
	config.ShowAuditLogs = false
	config.ShowLicense = false

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	// Test containers disabled
	req := httptest.NewRequest("GET", "/containers", nil)
	w := httptest.NewRecorder()
	server.handleContainers(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for disabled containers, got %d", w.Code)
	}

	// Test audit disabled
	req = httptest.NewRequest("GET", "/audit", nil)
	w = httptest.NewRecorder()
	server.handleAudit(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for disabled audit, got %d", w.Code)
	}

	// Test license disabled
	req = httptest.NewRequest("GET", "/license", nil)
	w = httptest.NewRecorder()
	server.handleLicense(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for disabled license, got %d", w.Code)
	}
}

// TestDashboardRedirect tests index redirect
func TestDashboardRedirect(t *testing.T) {
	config := DefaultDashboardConfig()
	config.BasePath = "/"

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if location != "/dashboard" {
		t.Errorf("Expected redirect to /dashboard, got %s", location)
	}
}

// TestCustomBasePath tests custom base path
func TestCustomBasePath(t *testing.T) {
	config := DefaultDashboardConfig()
	config.BasePath = "/admin"

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	if server.config.BasePath != "/admin" {
		t.Errorf("Expected base path /admin, got %s", server.config.BasePath)
	}
}

// TestUpdateStats tests stats update functionality
func TestUpdateStats(t *testing.T) {
	config := DefaultDashboardConfig()

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	newStats := DashboardStats{
		Uptime:           time.Now(),
		Version:          "2.0.0",
		ContainersTotal:  10,
		ContainersActive: 8,
		AuditEntries:     1000,
		LicenseStatus:    "active",
		HealthStatus:     "healthy",
	}

	server.UpdateStats(newStats)

	retrieved := server.GetStats()

	if retrieved.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", retrieved.Version)
	}

	if retrieved.ContainersTotal != 10 {
		t.Errorf("Expected 10 total containers, got %d", retrieved.ContainersTotal)
	}

	if retrieved.ContainersActive != 8 {
		t.Errorf("Expected 8 active containers, got %d", retrieved.ContainersActive)
	}
}

// TestJSONResponse tests JSON response helper
func TestJSONResponse(t *testing.T) {
	config := DefaultDashboardConfig()

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}

	server.jsonResponse(w, data)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type application/json")
	}

	if !strings.Contains(w.Body.String(), "test") {
		t.Error("Response body should contain test data")
	}
}

// TestConcurrentStatsUpdate tests concurrent stats updates
func TestConcurrentStatsUpdate(t *testing.T) {
	config := DefaultDashboardConfig()

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	done := make(chan bool)

	// Concurrent updates
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				server.UpdateStats(DashboardStats{
					ContainersActive: id*100 + j,
				})
			}
			done <- true
		}(i)
	}

	// Wait for completion
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no race condition occurred
	_ = server.GetStats()
}

// TestServerStartStop tests server lifecycle
func TestServerStartStop(t *testing.T) {
	config := DefaultDashboardConfig()
	config.Addr = ":0" // Random port
	config.Enabled = false // Disabled mode

	server, err := NewDashboardServer(config)
	if err != nil {
		t.Fatalf("Failed to create dashboard server: %v", err)
	}

	// Start in goroutine
	go server.Start()

	// Stop with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

// TestDefaultConfig tests default configuration values
func TestDefaultConfig(t *testing.T) {
	config := DefaultDashboardConfig()

	if !config.Enabled {
		t.Error("Dashboard should be enabled by default")
	}

	if config.Addr == "" {
		t.Error("Address should not be empty")
	}

	if !config.ShowAuditLogs {
		t.Error("Audit logs should be shown by default")
	}

	if !config.ShowLicense {
		t.Error("License should be shown by default")
	}

	if !config.ShowContainers {
		t.Error("Containers should be shown by default")
	}

	if !config.ShowHealth {
		t.Error("Health should be shown by default")
	}
}

// BenchmarkHealthEndpoint benchmarks health endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	config := DefaultDashboardConfig()
	server, _ := NewDashboardServer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		server.handleHealth(w, req)
	}
}

// BenchmarkStatusEndpoint benchmarks status endpoint
func BenchmarkStatusEndpoint(b *testing.B) {
	config := DefaultDashboardConfig()
	config.AuthEnabled = false
	server, _ := NewDashboardServer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/status", nil)
		w := httptest.NewRecorder()
		server.handleAPIStatus(w, req)
	}
}

// BenchmarkJSONResponse benchmarks JSON response generation
func BenchmarkJSONResponse(b *testing.B) {
	config := DefaultDashboardConfig()
	server, _ := NewDashboardServer(config)

	data := DashboardStats{
		Version:         "1.0.0",
		ContainersActive: 10,
		HealthStatus:    "healthy",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		server.jsonResponse(w, data)
	}
}
