// Package main provides the ArmorClaw License Server
// This server handles license validation for premium features
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// Configuration
type Config struct {
	Port          string
	DatabaseURL   string
	AdminToken    string
	GracePeriodDays int
}

// Server represents the license server
type Server struct {
	config Config
	db     *sql.DB
	logger *slog.Logger
	limiter *RateLimiter
}

// RateLimiter manages rate limiting per license
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*RateLimitEntry
}

type RateLimitEntry struct {
	Count     int
	ResetAt   time.Time
}

// License represents a license in the database
type License struct {
	ID           int
	LicenseKey   string
	Tier         string
	CustomerEmail string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Status       string
	MaxInstances int // Maximum allowed instances (0 = unlimited)
}

// Feature represents an available feature
type Feature struct {
	ID          int
	FeatureKey  string
	Name        string
	Description string
	Tier        string
}

// Instance represents a registered bridge instance
type Instance struct {
	ID         int
	InstanceID string
	LicenseID  int
	Hostname   string
	FirstSeen  time.Time
	LastSeen   time.Time
	Version    string
}

// ValidationRequest is the request body for license validation
type ValidationRequest struct {
	LicenseKey string `json:"license_key"`
	InstanceID string `json:"instance_id"`
	Feature    string `json:"feature"`
	Version    string `json:"version"`
}

// ValidationResponse is the response for license validation
type ValidationResponse struct {
	Valid            bool     `json:"valid"`
	Tier             string   `json:"tier,omitempty"`
	Features         []string `json:"features,omitempty"`
	ExpiresAt        string   `json:"expires_at,omitempty"`
	InstanceID       string   `json:"instance_id,omitempty"`
	GracePeriodDays  int      `json:"grace_period_days"`
	FeatureValid     bool     `json:"feature_valid,omitempty"`
	ErrorCode        string   `json:"error_code,omitempty"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	AvailableFeatures []string `json:"available_features,omitempty"`
}

// ActivationRequest is the request body for license activation
type ActivationRequest struct {
	LicenseKey string `json:"license_key"`
	Email      string `json:"email,omitempty"`
	InstanceID string `json:"instance_id"`
	Hostname   string `json:"hostname,omitempty"`
	Version    string `json:"version,omitempty"`
}

// ActivationResponse is the response for license activation
type ActivationResponse struct {
	Activated bool     `json:"activated"`
	Tier      string   `json:"tier,omitempty"`
	Features  []string `json:"features,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"`
}

// ErrorResponse represents an API error with structured codes
type ErrorResponse struct {
	Error        string `json:"error"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func main() {
	// Load configuration
	config := Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		AdminToken:    getEnv("ADMIN_TOKEN", ""),
		GracePeriodDays: parseInt(getEnv("GRACE_PERIOD_DAYS", "3"), 3),
	}

	if config.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Connect to database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create server
	server := &Server{
		config:  config,
		db:      db,
		logger:  logger,
		limiter: &RateLimiter{requests: make(map[string]*RateLimitEntry)},
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/licenses/validate", server.handleValidate)
	mux.HandleFunc("GET /v1/licenses/status", server.withAdminAuth(server.handleStatus))
	mux.HandleFunc("POST /v1/licenses/activate", server.handleActivate)
	mux.HandleFunc("POST /admin/v1/licenses", server.withAdminAuth(server.handleAdminCreate))
	mux.HandleFunc("DELETE /admin/v1/licenses/{key}", server.withAdminAuth(server.handleAdminRevoke))
	mux.HandleFunc("GET /health", server.handleHealth)

	// Start server
	addr := ":" + config.Port
	logger.Info("Starting license server", "addr", addr)

	if err := http.ListenAndServe(addr, server.loggingMiddleware(mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// initSchema initializes the database schema
func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS licenses (
		id SERIAL PRIMARY KEY,
		license_key VARCHAR(255) UNIQUE NOT NULL,
		tier VARCHAR(50) NOT NULL,
		customer_email VARCHAR(255),
		created_at TIMESTAMP DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL,
		status VARCHAR(50) DEFAULT 'active',
		max_instances INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS features (
		id SERIAL PRIMARY KEY,
		feature_key VARCHAR(100) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		tier VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS license_features (
		license_id INTEGER REFERENCES licenses(id) ON DELETE CASCADE,
		feature_id INTEGER REFERENCES features(id) ON DELETE CASCADE,
		granted_at TIMESTAMP DEFAULT NOW(),
		PRIMARY KEY (license_id, feature_id)
	);

	CREATE TABLE IF NOT EXISTS instances (
		id SERIAL PRIMARY KEY,
		instance_id UUID UNIQUE NOT NULL,
		license_id INTEGER REFERENCES licenses(id) ON DELETE CASCADE,
		hostname VARCHAR(255),
		first_seen TIMESTAMP DEFAULT NOW(),
		last_seen TIMESTAMP DEFAULT NOW(),
		version VARCHAR(50),
		metadata JSONB
	);

	CREATE TABLE IF NOT EXISTS validations (
		id SERIAL PRIMARY KEY,
		instance_id UUID,
		feature_key VARCHAR(100),
		validated_at TIMESTAMP DEFAULT NOW(),
		was_valid BOOLEAN,
		error_code VARCHAR(100)
	);

	CREATE INDEX IF NOT EXISTS idx_licenses_key ON licenses(license_key);
	CREATE INDEX IF NOT EXISTS idx_licenses_email ON licenses(customer_email);
	CREATE INDEX IF NOT EXISTS idx_validations_instance ON validations(instance_id);
	CREATE INDEX IF NOT EXISTS idx_validations_feature ON validations(feature_key);
	CREATE INDEX IF NOT EXISTS idx_instances_license ON instances(license_id);

	-- Insert default features if not exists
	INSERT INTO features (feature_key, name, description, tier) VALUES
		('slack-adapter', 'Slack Adapter', 'Slack Enterprise integration', 'pro'),
		('discord-adapter', 'Discord Adapter', 'Discord bot integration', 'pro'),
		('pii-scrubber', 'PII Scrubber', 'Basic PII redaction', 'pro'),
		('audit-log', 'Audit Log', 'Basic audit logging (30-day retention)', 'pro'),
		('whatsapp-adapter', 'WhatsApp Adapter', 'WhatsApp Business API', 'ent'),
		('teams-adapter', 'Teams Adapter', 'Microsoft Teams integration', 'ent'),
		('pii-scrubber-hipaa', 'HIPAA PII Scrubber', 'HIPAA-compliant PII scrubbing', 'ent'),
		('audit-log-compliance', 'Compliance Audit Log', 'Full compliance logging (90+ day)', 'ent'),
		('sso-integration', 'SSO Integration', 'SAML/OIDC single sign-on', 'ent'),
		('custom-adapter', 'Custom Adapter', 'Custom protocol adapter support', 'ent')
	ON CONFLICT (feature_key) DO NOTHING;

	-- Add max_instances column if it doesn't exist (for migrations)
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns
			WHERE table_name = 'licenses' AND column_name = 'max_instances') THEN
			ALTER TABLE licenses ADD COLUMN max_instances INTEGER DEFAULT 1;
		END IF;
	END $$;
	`

	_, err := db.Exec(schema)
	return err
}

// handleValidate handles POST /v1/licenses/validate
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Also check header for license key
	if req.LicenseKey == "" {
		req.LicenseKey = r.Header.Get("X-License-Key")
	}

	if req.LicenseKey == "" {
		s.writeError(w, http.StatusUnauthorized, ErrorResponse{Error: "License key required"})
		return
	}

	// Validate license key format
	if !isValidLicenseKey(req.LicenseKey) {
		s.writeError(w, http.StatusUnauthorized, ErrorResponse{Error: "Invalid license key format"})
		return
	}

	// Rate limiting
	tier := s.getLicenseTier(req.LicenseKey)
	if !s.checkRateLimit(req.LicenseKey, tier) {
		w.Header().Set("Retry-After", "60")
		s.writeError(w, http.StatusTooManyRequests, ErrorResponse{
			Error: "Rate limit exceeded",
		})
		return
	}

	// Validate license
	resp, err := s.validateLicense(r.Context(), req)
	if err != nil {
		s.logger.Error("License validation failed", "error", err, "license_key", maskLicenseKey(req.LicenseKey))
		resp = &ValidationResponse{
			Valid:           false,
			ErrorCode:       "VALIDATION_ERROR",
			ErrorMessage:    err.Error(),
			GracePeriodDays: s.config.GracePeriodDays,
		}
	}

	// Log validation
	s.logValidation(r.Context(), req, resp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// validateLicense performs the actual license validation
func (s *Server) validateLicense(ctx context.Context, req ValidationRequest) (*ValidationResponse, error) {
	// Query license
	var license License
	var features []string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, license_key, tier, customer_email, created_at, expires_at, status
		FROM licenses WHERE license_key = $1
	`, req.LicenseKey).Scan(&license.ID, &license.LicenseKey, &license.Tier,
		&license.CustomerEmail, &license.CreatedAt, &license.ExpiresAt, &license.Status)

	if err == sql.ErrNoRows {
		return &ValidationResponse{
			Valid:           false,
			ErrorCode:       "LICENSE_NOT_FOUND",
			ErrorMessage:    "License key not found",
			GracePeriodDays: s.config.GracePeriodDays,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check license status
	if license.Status == "revoked" {
		return &ValidationResponse{
			Valid:           false,
			ErrorCode:       "LICENSE_REVOKED",
			ErrorMessage:    "License has been revoked",
			GracePeriodDays: s.config.GracePeriodDays,
		}, nil
	}

	if license.Status == "suspended" {
		return &ValidationResponse{
			Valid:           false,
			ErrorCode:       "LICENSE_SUSPENDED",
			ErrorMessage:    "License is suspended",
			GracePeriodDays: s.config.GracePeriodDays,
		}, nil
	}

	// Check expiration
	if time.Now().After(license.ExpiresAt) {
		return &ValidationResponse{
			Valid:           false,
			ErrorCode:       "LICENSE_EXPIRED",
			ErrorMessage:    fmt.Sprintf("License expired on %s", license.ExpiresAt.Format("2006-01-02")),
			GracePeriodDays: s.config.GracePeriodDays,
		}, nil
	}

	// Get features for this license
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.feature_key
		FROM features f
		JOIN license_features lf ON f.id = lf.feature_id
		WHERE lf.license_id = $1
	`, license.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var feature string
		if err := rows.Scan(&feature); err != nil {
			return nil, err
		}
		features = append(features, feature)
	}

	// Register or update instance
	if req.InstanceID != "" {
		s.registerInstance(ctx, license.ID, req.InstanceID, req.Version, "")
	}

	// Check specific feature if requested
	featureValid := true
	var availableFeatures []string
	if req.Feature != "" && req.Feature != "license-info" {
		featureValid = containsFeature(features, req.Feature)
		availableFeatures = features
	}

	return &ValidationResponse{
		Valid:             true,
		Tier:              license.Tier,
		Features:          features,
		ExpiresAt:         license.ExpiresAt.Format(time.RFC3339),
		InstanceID:        req.InstanceID,
		GracePeriodDays:   s.config.GracePeriodDays,
		FeatureValid:      featureValid,
		AvailableFeatures: availableFeatures,
	}, nil
}

// handleActivate handles POST /v1/licenses/activate
// Uses database transaction with row-level locking to prevent race conditions
func (s *Server) handleActivate(w http.ResponseWriter, r *http.Request) {
	var req ActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.LicenseKey == "" || req.InstanceID == "" {
		s.writeError(w, http.StatusBadRequest, "license_key and instance_id are required")
		return
	}

	// Use transaction with row-level locking to prevent race conditions
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", "error", err)
		s.writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer tx.Rollback()

	// Lock the license row and get current state atomically
	// SELECT FOR UPDATE prevents other transactions from modifying this row
	var license License
	err = tx.QueryRowContext(r.Context(), `
		SELECT id, license_key, tier, expires_at, status, COALESCE(max_instances, 1)
		FROM licenses WHERE license_key = $1 AND status = 'active'
		FOR UPDATE
	`, req.LicenseKey).Scan(&license.ID, &license.LicenseKey, &license.Tier,
		&license.ExpiresAt, &license.Status, &license.MaxInstances)

	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, ErrorResponse{
			Error:       "License not found or inactive",
			ErrorCode:   "LICENSE_NOT_FOUND",
			ErrorMessage: "The provided license key is invalid or has been deactivated",
		})
		return
	}
	if err != nil {
		s.logger.Error("Failed to query license", "error", err, "license_key", maskLicenseKey(req.LicenseKey))
		s.writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Check if this instance is already registered
	var existingCount int
	err = tx.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM instances WHERE instance_id = $1 AND license_id = $2
	`, req.InstanceID, license.ID).Scan(&existingCount)
	if err != nil {
		s.logger.Error("Failed to check existing instance", "error", err)
		s.writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// If instance is not already registered, check instance limit
	if existingCount == 0 {
		// Count current instances atomically (within the locked transaction)
		var currentInstances int
		err = tx.QueryRowContext(r.Context(), `
			SELECT COUNT(*) FROM instances WHERE license_id = $1
		`, license.ID).Scan(&currentInstances)
		if err != nil {
			s.logger.Error("Failed to count instances", "error", err)
			s.writeError(w, http.StatusInternalServerError, "Database error")
			return
		}

		// Check against max_instances limit
		// max_instances = 0 means unlimited
		if license.MaxInstances > 0 && currentInstances >= license.MaxInstances {
			s.logger.Warn("Instance limit exceeded",
				"license_key", maskLicenseKey(req.LicenseKey),
				"current_instances", currentInstances,
				"max_instances", license.MaxInstances,
			)
			s.writeError(w, http.StatusForbidden, ErrorResponse{
				Error:       "Instance limit exceeded",
				ErrorCode:   "INSTANCE_LIMIT_EXCEEDED",
				ErrorMessage: fmt.Sprintf("This license has reached its maximum of %d instance(s). Current: %d", license.MaxInstances, currentInstances),
			})
			return
		}
	}

	// Register instance within the same transaction
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO instances (instance_id, license_id, hostname, version, first_seen, last_seen)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (instance_id) DO UPDATE SET
			last_seen = NOW(),
			version = EXCLUDED.version,
			hostname = COALESCE(EXCLUDED.hostname, instances.hostname)
	`, req.InstanceID, license.ID, req.Hostname, req.Version)
	if err != nil {
		s.logger.Error("Failed to register instance", "error", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to register instance")
		return
	}

	// Get features
	features, err := s.getLicenseFeaturesWithContext(r.Context(), tx, license.ID)
	if err != nil {
		s.logger.Error("Failed to get features", "error", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get features")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", "error", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to complete activation")
		return
	}

	s.logger.Info("Instance activated",
		"license_key", maskLicenseKey(req.LicenseKey),
		"instance_id", req.InstanceID[:8]+"...",
		"tier", license.Tier,
	)

	resp := &ActivationResponse{
		Activated: true,
		Tier:      license.Tier,
		Features:  features,
		ExpiresAt: license.ExpiresAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStatus handles GET /v1/licenses/status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("license_key")
	if licenseKey == "" {
		s.writeError(w, http.StatusBadRequest, "license_key parameter required")
		return
	}

	var license License
	err := s.db.QueryRowContext(r.Context(), `
		SELECT id, license_key, tier, customer_email, created_at, expires_at, status
		FROM licenses WHERE license_key = $1
	`, licenseKey).Scan(&license.ID, &license.LicenseKey, &license.Tier,
		&license.CustomerEmail, &license.CreatedAt, &license.ExpiresAt, &license.Status)

	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "License not found")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	features, _ := s.getLicenseFeatures(r.Context(), license.ID)
	instances := s.getLicenseInstances(r.Context(), license.ID)

	// Get validation stats
	var validationsThisMonth int
	s.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM validations
		WHERE instance_id IN (SELECT instance_id FROM instances WHERE license_id = $1)
		AND validated_at >= DATE_TRUNC('month', CURRENT_DATE)
	`, license.ID).Scan(&validationsThisMonth)

	var lastValidation sql.NullTime
	s.db.QueryRowContext(r.Context(), `
		SELECT MAX(validated_at) FROM validations
		WHERE instance_id IN (SELECT instance_id FROM instances WHERE license_id = $1)
	`, license.ID).Scan(&lastValidation)

	response := map[string]interface{}{
		"license_key":            maskLicenseKey(license.LicenseKey),
		"tier":                   license.Tier,
		"status":                 license.Status,
		"created_at":             license.CreatedAt.Format(time.RFC3339),
		"expires_at":             license.ExpiresAt.Format(time.RFC3339),
		"features":               features,
		"usage": map[string]interface{}{
			"validations_this_month": validationsThisMonth,
			"last_validation":        formatNullTime(lastValidation),
			"instances":              instances,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAdminCreate handles POST /admin/v1/licenses
func (s *Server) handleAdminCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tier         string   `json:"tier"`
		Email        string   `json:"email"`
		DurationDays int      `json:"duration_days"`
		MaxInstances int      `json:"max_instances"` // 0 = unlimited
		Features     []string `json:"features"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Tier == "" || req.DurationDays == 0 {
		s.writeError(w, http.StatusBadRequest, "tier and duration_days are required")
		return
	}

	// Set default max_instances based on tier if not specified
	maxInstances := req.MaxInstances
	if maxInstances == 0 {
		maxInstances = getDefaultMaxInstances(req.Tier)
	}

	// Generate license key
	licenseKey, err := generateLicenseKey(req.Tier)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to generate license key")
		return
	}

	expiresAt := time.Now().AddDate(0, 0, req.DurationDays)

	// Insert license with max_instances
	var licenseID int
	err = s.db.QueryRowContext(r.Context(), `
		INSERT INTO licenses (license_key, tier, customer_email, expires_at, status, max_instances)
		VALUES ($1, $2, $3, $4, 'active', $5)
		RETURNING id
	`, licenseKey, req.Tier, req.Email, expiresAt, maxInstances).Scan(&licenseID)

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create license")
		return
	}

	// Assign features
	for _, featureKey := range req.Features {
		s.db.ExecContext(r.Context(), `
			INSERT INTO license_features (license_id, feature_id)
			SELECT $1, id FROM features WHERE feature_key = $2
			ON CONFLICT DO NOTHING
		`, licenseID, featureKey)
	}

	// If no specific features, assign all features for tier
	if len(req.Features) == 0 {
		s.db.ExecContext(r.Context(), `
			INSERT INTO license_features (license_id, feature_id)
			SELECT $1, id FROM features WHERE tier = $2 OR tier = 'pro'
		`, licenseID, req.Tier)
	}

	s.logger.Info("License created", "license_key", maskLicenseKey(licenseKey), "tier", req.Tier, "expires_at", expiresAt, "max_instances", maxInstances)

	response := map[string]interface{}{
		"license_key":    licenseKey,
		"tier":           req.Tier,
		"expires_at":     expiresAt.Format(time.RFC3339),
		"max_instances":  maxInstances,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getDefaultMaxInstances returns the default instance limit for a tier
func getDefaultMaxInstances(tier string) int {
	switch tier {
	case "ent", "enterprise":
		return 10 // Enterprise: 10 instances
	case "pro", "professional":
		return 3  // Professional: 3 instances
	default:
		return 1  // Free/Essential: 1 instance
	}
}

// handleAdminRevoke handles DELETE /admin/v1/licenses/{key}
func (s *Server) handleAdminRevoke(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.PathValue("key")
	if licenseKey == "" {
		s.writeError(w, http.StatusBadRequest, "License key required")
		return
	}

	result, err := s.db.ExecContext(r.Context(), `
		UPDATE licenses SET status = 'revoked' WHERE license_key = $1
	`, licenseKey)

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to revoke license")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		s.writeError(w, http.StatusNotFound, "License not found")
		return
	}

	s.logger.Info("License revoked", "license_key", maskLicenseKey(licenseKey))

	response := map[string]interface{}{
		"revoked":     true,
		"revoked_at":  time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Handle nil database gracefully (for testing or initialization)
	if s.db == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "note": "database not configured"})
		return
	}

	if err := s.db.PingContext(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// Helper functions

func (s *Server) writeError(w http.ResponseWriter, code int, message interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: fmt.Sprintf("%v", message),
	})
}

func (s *Server) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("admin_token")
		} else {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		if token == "" || token != s.config.AdminToken {
			s.writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next(w, r)
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
		)
	})
}

func (s *Server) checkRateLimit(licenseKey, tier string) bool {
	s.limiter.mu.Lock()
	defer s.limiter.mu.Unlock()

	var limit int
	switch tier {
	case "ent":
		limit = 10000
	case "pro":
		limit = 1000
	default:
		limit = 100
	}

	entry, exists := s.limiter.requests[licenseKey]
	now := time.Now()

	if !exists || now.After(entry.ResetAt) {
		s.limiter.requests[licenseKey] = &RateLimitEntry{
			Count:   1,
			ResetAt: now.Add(time.Hour),
		}
		return true
	}

	if entry.Count >= limit {
		return false
	}

	entry.Count++
	return true
}

func (s *Server) getLicenseTier(licenseKey string) string {
	parts := strings.Split(licenseKey, "-")
	if len(parts) >= 2 {
		return strings.ToLower(parts[1])
	}
	return "free"
}

func (s *Server) registerInstance(ctx context.Context, licenseID int, instanceID, version, hostname string) {
	s.db.ExecContext(ctx, `
		INSERT INTO instances (instance_id, license_id, hostname, version, first_seen, last_seen)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (instance_id) DO UPDATE SET
			last_seen = NOW(),
			version = EXCLUDED.version,
			hostname = COALESCE(EXCLUDED.hostname, instances.hostname)
	`, instanceID, licenseID, hostname, version)
}

func (s *Server) getLicenseFeatures(ctx context.Context, licenseID int) ([]string, error) {
	return s.getLicenseFeaturesWithContext(ctx, s.db, licenseID)
}

// getLicenseFeaturesWithContext gets features using a transaction or db connection
func (s *Server) getLicenseFeaturesWithContext(ctx context.Context, db Querier, licenseID int) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT f.feature_key
		FROM features f
		JOIN license_features lf ON f.id = lf.feature_id
		WHERE lf.license_id = $1
	`, licenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		features = append(features, f)
	}
	return features, nil
}

// Querier is an interface for database operations (supports both *sql.DB and *sql.Tx)
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (s *Server) getLicenseInstances(ctx context.Context, licenseID int) []map[string]interface{} {
	rows, err := s.db.QueryContext(ctx, `
		SELECT instance_id, hostname, first_seen, last_seen
		FROM instances WHERE license_id = $1
	`, licenseID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var instances []map[string]interface{}
	for rows.Next() {
		var instanceID, hostname string
		var firstSeen, lastSeen time.Time
		if err := rows.Scan(&instanceID, &hostname, &firstSeen, &lastSeen); err != nil {
			continue
		}
		instances = append(instances, map[string]interface{}{
			"instance_id": instanceID,
			"hostname":    hostname,
			"first_seen":  firstSeen.Format(time.RFC3339),
			"last_seen":   lastSeen.Format(time.RFC3339),
		})
	}
	return instances
}

func (s *Server) logValidation(ctx context.Context, req ValidationRequest, resp *ValidationResponse) {
	var errorCode string
	if !resp.Valid {
		errorCode = resp.ErrorCode
	}

	s.db.ExecContext(ctx, `
		INSERT INTO validations (instance_id, feature_key, validated_at, was_valid, error_code)
		VALUES ($1, $2, NOW(), $3, $4)
	`, req.InstanceID, req.Feature, resp.Valid, errorCode)
}

func isValidLicenseKey(key string) bool {
	parts := strings.Split(key, "-")
	if len(parts) != 3 {
		return false
	}
	if parts[0] != "SCLW" {
		return false
	}
	tier := strings.ToLower(parts[1])
	if tier != "free" && tier != "pro" && tier != "ent" {
		return false
	}
	if len(parts[2]) != 16 {
		return false
	}
	_, err := hex.DecodeString(parts[2])
	return err == nil
}

func generateLicenseKey(tier string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("SCLW-%s-%s", strings.ToUpper(tier), hex.EncodeToString(b)), nil
}

func maskLicenseKey(key string) string {
	if len(key) < 10 {
		return key
	}
	return key[:10] + "****"
}

func containsFeature(features []string, feature string) bool {
	for _, f := range features {
		if f == feature {
			return true
		}
	}
	return false
}

func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseInt(s string, defaultVal int) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultVal
}
