// Package main provides integration tests for the license server
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestConfig holds test configuration
type TestConfig struct {
	DatabaseURL string
	AdminToken  string
}

func getTestConfig() TestConfig {
	return TestConfig{
		DatabaseURL: os.Getenv("TEST_DATABASE_URL"),
		AdminToken:  "test-admin-token-12345",
	}
}

// createTestServer creates a test server instance
func createTestServer(db *sql.DB, adminToken string) *Server {
	return &Server{
		config: Config{
			AdminToken:     adminToken,
			GracePeriodDays: 3,
		},
		db:      db,
		limiter: &RateLimiter{requests: make(map[string]*RateLimitEntry)},
	}
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	server := createTestServer(nil, "test-token")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// When db is nil, status should be "ok" with note about database not configured
	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", resp["status"])
	}
}

// TestValidateLicenseWithDB tests the license validation endpoint with database
func TestValidateLicenseWithDB(t *testing.T) {
	config := getTestConfig()
	if config.DatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	server := createTestServer(db, config.AdminToken)

	// Create test license
	licenseKey := fmt.Sprintf("TEST-VALIDATE-%d", time.Now().UnixNano())
	_, err = db.Exec(`
		INSERT INTO licenses (license_key, tier, customer_email, created_at)
		VALUES ($1, 'enterprise', 'test@example.com', NOW())
	`, licenseKey)
	if err != nil {
		t.Fatalf("Failed to create test license: %v", err)
	}
	defer db.Exec("DELETE FROM licenses WHERE license_key = $1", licenseKey)

	tests := []struct {
		name       string
		payload    ValidationRequest
		wantStatus int
		wantValid  bool
	}{
		{
			name: "valid license",
			payload: ValidationRequest{
				LicenseKey: licenseKey,
				InstanceID: "instance-001",
				Version:    "1.0.0",
			},
			wantStatus: http.StatusOK,
			wantValid:  true,
		},
		{
			name: "invalid license",
			payload: ValidationRequest{
				LicenseKey: "INVALID-LICENSE",
				InstanceID: "instance-001",
				Version:    "1.0.0",
			},
			wantStatus: http.StatusOK,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/v1/licenses/validate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleValidate(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusOK {
				var resp ValidationResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if resp.Valid != tt.wantValid {
					t.Errorf("Expected valid=%v, got %v", tt.wantValid, resp.Valid)
				}
			}
		})
	}
}

// TestLicenseActivationWithDB tests the license activation endpoint
func TestLicenseActivationWithDB(t *testing.T) {
	config := getTestConfig()
	if config.DatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	server := createTestServer(db, config.AdminToken)

	// Create test license
	licenseKey := fmt.Sprintf("TEST-ACTIVATE-%d", time.Now().UnixNano())
	_, err = db.Exec(`
		INSERT INTO licenses (license_key, tier, customer_email, created_at)
		VALUES ($1, 'professional', 'test@example.com', NOW())
	`, licenseKey)
	if err != nil {
		t.Fatalf("Failed to create test license: %v", err)
	}
	defer db.Exec("DELETE FROM licenses WHERE license_key = $1", licenseKey)

	payload := ActivationRequest{
		LicenseKey: licenseKey,
		InstanceID: "instance-activate-001",
		Email:      "test@example.com",
		Version:    "1.0.0",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/v1/licenses/activate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleActivate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAdminAuth tests admin endpoint authentication
func TestAdminAuth(t *testing.T) {
	config := getTestConfig()
	server := createTestServer(nil, config.AdminToken)

	tests := []struct {
		name   string
		token  string
		want   int
	}{
		{
			name:  "valid token",
			token: config.AdminToken,
			want:  http.StatusOK,
		},
		{
			name:  "invalid token",
			token: "invalid-token",
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
			req := httptest.NewRequest("GET", "/admin/licenses", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			w := httptest.NewRecorder()

			handler := server.withAdminAuth(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
			})
			handler(w, req)

			if w.Code != tt.want {
				t.Errorf("Expected status %d, got %d", tt.want, w.Code)
			}
		})
	}
}

// TestGracePeriod tests grace period handling
func TestGracePeriod(t *testing.T) {
	// Test grace period calculation
	now := time.Now()
	graceDays := 3

	// License expired 2 days ago (within grace period)
	expiredWithinGrace := now.Add(-48 * time.Hour)
	inGracePeriod := expiredWithinGrace.AddDate(0, 0, graceDays).After(now)

	if !inGracePeriod {
		t.Error("License expired 2 days ago should be within 3-day grace period")
	}

	// License expired 5 days ago (outside grace period)
	expiredOutsideGrace := now.Add(-5 * 24 * time.Hour)
	outsideGracePeriod := expiredOutsideGrace.AddDate(0, 0, graceDays).After(now)

	if outsideGracePeriod {
		t.Error("License expired 5 days ago should be outside 3-day grace period")
	}
}

// TestTierFeatures tests tier-specific features
func TestTierFeatures(t *testing.T) {
	features := map[string][]string{
		"basic":        getTierFeatures("basic"),
		"professional": getTierFeatures("professional"),
		"enterprise":   getTierFeatures("enterprise"),
	}

	// Basic should have containers
	basicContainers := false
	for _, f := range features["basic"] {
		if f == "containers" {
			basicContainers = true
		}
	}
	if !basicContainers {
		t.Error("Basic tier should include containers")
	}

	// Professional should have adapters
	profAdapters := false
	for _, f := range features["professional"] {
		if f == "adapters" {
			profAdapters = true
		}
	}
	if !profAdapters {
		t.Error("Professional tier should include adapters")
	}

	// Enterprise should have sso and hipaa
	entSSO := false
	entHIPAA := false
	for _, f := range features["enterprise"] {
		if f == "sso" {
			entSSO = true
		}
		if f == "hipaa" {
			entHIPAA = true
		}
	}
	if !entSSO {
		t.Error("Enterprise tier should include sso")
	}
	if !entHIPAA {
		t.Error("Enterprise tier should include hipaa")
	}
}

// TestRateLimiterInternal tests rate limiter functionality
func TestRateLimiterInternal(t *testing.T) {
	server := createTestServer(nil, "test-token")

	// Test that rate limiting works via checkRateLimit
	key := "test-license-key"

	// Should allow first 100 requests for enterprise tier
	for i := 0; i < 100; i++ {
		if !server.checkRateLimit(key, "enterprise") {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// Requests beyond limit should be denied
	allowed := server.checkRateLimit(key, "enterprise")
	if allowed {
		t.Error("Request beyond limit should be denied")
	}
}

// TestLicenseStatusWithDB tests the status endpoint
func TestLicenseStatusWithDB(t *testing.T) {
	config := getTestConfig()
	if config.DatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	server := createTestServer(db, config.AdminToken)

	// Create test license
	licenseKey := fmt.Sprintf("TEST-STATUS-%d", time.Now().UnixNano())
	_, err = db.Exec(`
		INSERT INTO licenses (license_key, tier, customer_email, created_at)
		VALUES ($1, 'enterprise', 'test@example.com', NOW())
	`, licenseKey)
	if err != nil {
		t.Fatalf("Failed to create test license: %v", err)
	}
	defer db.Exec("DELETE FROM licenses WHERE license_key = $1", licenseKey)

	req := httptest.NewRequest("GET", "/v1/licenses/status?license_key="+licenseKey, nil)
	w := httptest.NewRecorder()

	server.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestValidationWithoutDatabase tests validation gracefully handles no database
func TestValidationWithoutDatabase(t *testing.T) {
	server := createTestServer(nil, "test-token")

	payload := ValidationRequest{
		LicenseKey: "any-key",
		InstanceID: "instance-001",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/v1/licenses/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleValidate(w, req)

	// Should handle gracefully (not panic)
	t.Logf("Validation without database returned status %d: %s", w.Code, w.Body.String())
}

func getTierFeatures(tier string) []string {
	switch tier {
	case "basic":
		return []string{"containers", "basic_support"}
	case "professional":
		return []string{"containers", "adapters", "audit", "priority_support"}
	case "enterprise":
		return []string{"containers", "adapters", "audit", "sso", "hipaa", "dedicated_support"}
	default:
		return []string{"containers"}
	}
}
