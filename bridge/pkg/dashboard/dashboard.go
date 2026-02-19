// Package dashboard provides a web-based management dashboard for ArmorClaw
package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

// DashboardConfig configures the web dashboard
type DashboardConfig struct {
	// Server settings
	Enabled  bool
	Addr     string // Address to listen on (e.g., ":8080")
	BasePath string // Base path for the dashboard (e.g., "/admin")

	// Security
	AuthEnabled bool
	AdminToken  string // Bearer token for API auth

	// TLS settings
	TLSEnabled bool
	CertFile   string
	KeyFile    string

	// Features
	ShowAuditLogs    bool
	ShowLicense      bool
	ShowContainers   bool
	ShowHealth       bool

	// Logger
	Logger *slog.Logger
}

// DashboardServer provides the web dashboard
type DashboardServer struct {
	config   DashboardConfig
	server   *http.Server
	templates *template.Template
	mux      *http.ServeMux
	stats    *DashboardStats
	statsMu  sync.RWMutex
	logger   *slog.Logger
}

// DashboardStats contains dashboard statistics
type DashboardStats struct {
	Uptime         time.Time     `json:"uptime"`
	Version        string        `json:"version"`
	ContainersTotal int          `json:"containers_total"`
	ContainersActive int         `json:"containers_active"`
	AuditEntries   int          `json:"audit_entries"`
	LicenseStatus  string       `json:"license_status"`
	LicenseExpiry  time.Time    `json:"license_expiry"`
	LastUpdated    time.Time    `json:"last_updated"`
	HealthStatus   string       `json:"health_status"`
}

// SystemInfo contains system information
type SystemInfo struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
}

// ContainerInfo represents container information for dashboard
type ContainerInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	Image    string            `json:"image"`
	Created  time.Time         `json:"created"`
	Ports    []string          `json:"ports"`
	Labels   map[string]string `json:"labels"`
	Health   string            `json:"health"`
}

// AuditLogEntry represents an audit log entry for dashboard
type AuditLogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Status    string    `json:"status"`
}

// LicenseInfo represents license information for dashboard
type LicenseInfo struct {
	LicenseID   string    `json:"license_id"`
	Tier        string    `json:"tier"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	Instances   int       `json:"instances"`
	MaxInstances int      `json:"max_instances"`
	Features    []string  `json:"features"`
}

// DefaultDashboardConfig returns default dashboard configuration
func DefaultDashboardConfig() DashboardConfig {
	return DashboardConfig{
		Enabled:        true,
		Addr:           ":8080",
		BasePath:       "/",
		AuthEnabled:    true,
		ShowAuditLogs:  true,
		ShowLicense:    true,
		ShowContainers: true,
		ShowHealth:     true,
	}
}

// NewDashboardServer creates a new dashboard server
func NewDashboardServer(config DashboardConfig) (*DashboardServer, error) {
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	// Parse templates
	tmpl, err := template.ParseFS(staticFS, "static/*.html")
	if err != nil {
		logger.Warn("Failed to parse templates, using embedded defaults", "error", err)
		tmpl = template.New("default")
	}

	// Create mux
	mux := http.NewServeMux()

	// Create server
	server := &http.Server{
		Addr:         config.Addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	dashboard := &DashboardServer{
		config:    config,
		server:    server,
		templates: tmpl,
		mux:       mux,
		logger:    logger,
		stats: &DashboardStats{
			Uptime:        time.Now(),
			Version:       "1.0.0",
			LicenseStatus: "unknown",
			HealthStatus:  "unknown",
		},
	}

	// Setup routes
	dashboard.setupRoutes()

	return dashboard, nil
}

// setupRoutes configures the HTTP routes
func (d *DashboardServer) setupRoutes() {
	basePath := strings.TrimSuffix(d.config.BasePath, "/")

	// Static files
	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		d.logger.Error("Failed to get static FS", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(staticContent))
		d.mux.Handle(basePath+"/static/", http.StripPrefix(basePath+"/static/", fileServer))
	}

	// Dashboard pages
	d.mux.HandleFunc(basePath+"/", d.handleIndex)
	d.mux.HandleFunc(basePath+"/dashboard", d.handleDashboard)
	d.mux.HandleFunc(basePath+"/containers", d.handleContainers)
	d.mux.HandleFunc(basePath+"/audit", d.handleAudit)
	d.mux.HandleFunc(basePath+"/license", d.handleLicense)
	d.mux.HandleFunc(basePath+"/settings", d.handleSettings)

	// API endpoints
	d.mux.HandleFunc(basePath+"/api/status", d.authMiddleware(d.handleAPIStatus))
	d.mux.HandleFunc(basePath+"/api/containers", d.authMiddleware(d.handleAPIContainers))
	d.mux.HandleFunc(basePath+"/api/audit", d.authMiddleware(d.handleAPIAudit))
	d.mux.HandleFunc(basePath+"/api/license", d.authMiddleware(d.handleAPILicense))
	d.mux.HandleFunc(basePath+"/api/health", d.handleHealth)
	d.mux.HandleFunc(basePath+"/api/system", d.authMiddleware(d.handleAPISystem))
}

// authMiddleware provides authentication for API endpoints
func (d *DashboardServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.config.AuthEnabled {
			next(w, r)
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			// Check for session cookie (for web UI)
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value == "" {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}
			// Validate session cookie
			if cookie.Value != d.config.AdminToken {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}
		} else {
			// Validate Bearer token
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != d.config.AdminToken {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}
		}

		next(w, r)
	}
}

// Start starts the dashboard server
func (d *DashboardServer) Start() error {
	if !d.config.Enabled {
		d.logger.Info("Dashboard is disabled")
		return nil
	}

	d.logger.Info("Starting dashboard server",
		"addr", d.config.Addr,
		"base_path", d.config.BasePath,
		"tls", d.config.TLSEnabled,
	)

	var err error
	if d.config.TLSEnabled {
		err = d.server.ListenAndServeTLS(d.config.CertFile, d.config.KeyFile)
	} else {
		err = d.server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("dashboard server error: %w", err)
	}

	return nil
}

// Stop stops the dashboard server
func (d *DashboardServer) Stop(ctx context.Context) error {
	d.logger.Info("Stopping dashboard server")
	return d.server.Shutdown(ctx)
}

// UpdateStats updates the dashboard statistics
func (d *DashboardServer) UpdateStats(stats DashboardStats) {
	d.statsMu.Lock()
	defer d.statsMu.Unlock()

	stats.LastUpdated = time.Now()
	d.stats = &stats
}

// GetStats returns current dashboard statistics
func (d *DashboardServer) GetStats() DashboardStats {
	d.statsMu.RLock()
	defer d.statsMu.RUnlock()

	return *d.stats
}

// Page handlers

func (d *DashboardServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	basePath := strings.TrimSuffix(d.config.BasePath, "/")
	http.Redirect(w, r, basePath+"/dashboard", http.StatusFound)
}

func (d *DashboardServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "Dashboard",
		"Stats":  d.GetStats(),
		"Config": d.config,
	}

	d.renderTemplate(w, "dashboard.html", data)
}

func (d *DashboardServer) handleContainers(w http.ResponseWriter, r *http.Request) {
	if !d.config.ShowContainers {
		http.Error(w, "Containers view disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":  "Containers",
		"Config": d.config,
	}

	d.renderTemplate(w, "containers.html", data)
}

func (d *DashboardServer) handleAudit(w http.ResponseWriter, r *http.Request) {
	if !d.config.ShowAuditLogs {
		http.Error(w, "Audit view disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":  "Audit Logs",
		"Config": d.config,
	}

	d.renderTemplate(w, "audit.html", data)
}

func (d *DashboardServer) handleLicense(w http.ResponseWriter, r *http.Request) {
	if !d.config.ShowLicense {
		http.Error(w, "License view disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":  "License",
		"Config": d.config,
	}

	d.renderTemplate(w, "license.html", data)
}

func (d *DashboardServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "Settings",
		"Config": d.config,
	}

	d.renderTemplate(w, "settings.html", data)
}

// API handlers

func (d *DashboardServer) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	stats := d.GetStats()
	d.jsonResponse(w, stats)
}

func (d *DashboardServer) handleAPIContainers(w http.ResponseWriter, r *http.Request) {
	// Mock container data for MVP
	containers := []ContainerInfo{
		{
			ID:      "abc123",
			Name:    "armorclaw-agent-1",
			Status:  "running",
			Image:   "armorclaw/agent:v1",
			Created: time.Now().Add(-24 * time.Hour),
			Health:  "healthy",
			Labels: map[string]string{
				"app": "armorclaw",
			},
		},
	}

	d.jsonResponse(w, containers)
}

func (d *DashboardServer) handleAPIAudit(w http.ResponseWriter, r *http.Request) {
	// Mock audit data for MVP
	entries := []AuditLogEntry{
		{
			ID:        "audit-001",
			Timestamp: time.Now().Add(-1 * time.Hour),
			EventType: "container.create",
			User:      "admin",
			Action:    "create",
			Resource:  "container/armorclaw-agent-1",
			Status:    "success",
		},
	}

	d.jsonResponse(w, entries)
}

func (d *DashboardServer) handleAPILicense(w http.ResponseWriter, r *http.Request) {
	// Mock license data for MVP
	license := LicenseInfo{
		LicenseID:    "LIC-XXXXX-XXXXX",
		Tier:         "enterprise",
		Status:       "active",
		ExpiresAt:    time.Now().AddDate(1, 0, 0),
		Instances:    1,
		MaxInstances: 10,
		Features: []string{
			"containers",
			"audit",
			"sso",
			"hipaa",
		},
	}

	d.jsonResponse(w, license)
}

func (d *DashboardServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	d.jsonResponse(w, health)
}

func (d *DashboardServer) handleAPISystem(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	info := SystemInfo{
		Hostname:  hostname,
		OS:        "linux",
		Arch:      "amd64",
		GoVersion: "go1.24",
	}

	d.jsonResponse(w, info)
}

// Helper methods

func (d *DashboardServer) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := d.templates.ExecuteTemplate(w, name, data); err != nil {
		d.logger.Error("Failed to render template", "name", name, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (d *DashboardServer) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		d.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
