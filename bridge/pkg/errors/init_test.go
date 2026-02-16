package errors

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// testStorePath creates a temp path for database files
// On Windows, uses a path that won't conflict with cleanup
func testStorePath(t *testing.T) string {
	if runtime.GOOS == "windows" {
		// Use os.TempDir directly to avoid cleanup issues
		tmpDir := os.TempDir()
		return filepath.Join(tmpDir, "armorclaw_errors_test_"+t.Name()+".db")
	}
	return filepath.Join(t.TempDir(), "errors.db")
}

// cleanupStore removes the test database file
func cleanupStore(t *testing.T, path string) {
	if runtime.GOOS == "windows" {
		// Give time for file handles to release
		time.Sleep(100 * time.Millisecond)
		os.Remove(path)
	}
}

func TestInitialize(t *testing.T) {
	storePath := testStorePath(t)
	defer cleanupStore(t, storePath)

	cfg := Config{
		StorePath:       storePath,
		RetentionDays:   30,
		RateLimitWindow: "5m",
		RetentionPeriod: "24h",
		SetupUserMXID:   "@admin:example.com",
		Enabled:         true,
		StoreEnabled:    true,
		NotifyEnabled:   true,
	}

	system, err := Initialize(cfg)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if system.registry == nil {
		t.Error("Registry should be initialized")
	}
	if system.resolver == nil {
		t.Error("Resolver should be initialized")
	}
	if system.store == nil {
		t.Error("Store should be initialized")
	}
	if system.notifier == nil {
		t.Error("Notifier should be initialized")
	}

	// Verify globals are set
	if GetGlobalRegistry() == nil {
		t.Error("Global registry should be set")
	}
	if GetGlobalAdminResolver() == nil {
		t.Error("Global resolver should be set")
	}
	if GetGlobalStore() == nil {
		t.Error("Global store should be set")
	}
	if GetGlobalNotifier() == nil {
		t.Error("Global notifier should be set")
	}

	// Close before cleanup
	system.Stop()
}

func TestInitialize_StoreDisabled(t *testing.T) {
	cfg := Config{
		StoreEnabled:  false,
		SetupUserMXID: "@admin:example.com",
		Enabled:       true,
	}

	system, err := Initialize(cfg)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if system.store != nil {
		t.Error("Store should be nil when disabled")
	}

	system.Stop()
}

func TestSystem_StartStop(t *testing.T) {
	storePath := testStorePath(t)
	defer cleanupStore(t, storePath)

	system, _ := Initialize(Config{
		StorePath:    storePath,
		StoreEnabled: true,
	})

	// Start
	if err := system.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !system.started {
		t.Error("System should be started")
	}

	// Second start should be no-op
	if err := system.Start(context.Background()); err != nil {
		t.Fatalf("Second Start() error = %v", err)
	}

	// Stop
	if err := system.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Second stop should be no-op
	if err := system.Stop(); err != nil {
		t.Fatalf("Second Stop() error = %v", err)
	}
}

func TestSystem_Notify(t *testing.T) {
	mockSender := &mockMatrixSender{}

	system, _ := Initialize(Config{
		MatrixSender:  mockSender,
		SetupUserMXID: "@admin:example.com",
		Enabled:       true,
		NotifyEnabled: true,
		StoreEnabled:  false,
	})

	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test error",
		TraceID:   "tr_test",
		Timestamp: time.Now(),
	}

	notifyErr := system.Notify(context.Background(), err)
	if notifyErr != nil {
		t.Fatalf("Notify() error = %v", notifyErr)
	}

	if mockSender.callCount != 1 {
		t.Errorf("SendMessage called %d times, want 1", mockSender.callCount)
	}

	system.Stop()
}

func TestSystem_Store(t *testing.T) {
	storePath := testStorePath(t)
	defer cleanupStore(t, storePath)

	system, _ := Initialize(Config{
		StorePath:    storePath,
		StoreEnabled: true,
	})

	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test error",
		TraceID:   "tr_test",
		Timestamp: time.Now(),
	}

	storeErr := system.Store(context.Background(), err)
	if storeErr != nil {
		t.Fatalf("Store() error = %v", storeErr)
	}

	// Query to verify
	results, _ := system.Query(context.Background(), ErrorQuery{})
	if len(results) != 1 {
		t.Errorf("Query() returned %d results, want 1", len(results))
	}

	system.Stop()
}

func TestSystem_Resolve(t *testing.T) {
	storePath := testStorePath(t)
	defer cleanupStore(t, storePath)

	system, _ := Initialize(Config{
		StorePath:    storePath,
		StoreEnabled: true,
	})

	// Store an error
	system.Store(context.Background(), &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_test",
		Timestamp: time.Now(),
	})

	// Resolve it
	resolveErr := system.Resolve(context.Background(), "tr_test", "@admin:example.com")
	if resolveErr != nil {
		t.Fatalf("Resolve() error = %v", resolveErr)
	}

	// Verify resolved
	resolved := true
	results, _ := system.Query(context.Background(), ErrorQuery{Resolved: &resolved})
	if len(results) != 1 {
		t.Error("Error should be resolved")
	}

	system.Stop()
}

func TestSystem_Stats(t *testing.T) {
	storePath := testStorePath(t)
	defer cleanupStore(t, storePath)

	system, _ := Initialize(Config{
		StorePath:     storePath,
		StoreEnabled:  true,
		SetupUserMXID: "@admin:example.com",
	})

	// Store some errors
	for i := 0; i < 3; i++ {
		system.Store(context.Background(), &TracedError{
			Code:      "CTX-001",
			Category:  "container",
			Severity:  SeverityError,
			Message:   "test",
			TraceID:   "tr_test",
			Timestamp: time.Now(),
		})
	}

	stats := system.Stats(context.Background())

	if stats.Store == nil {
		t.Error("Store stats should be populated")
	}

	if stats.AdminSource == "" {
		t.Error("AdminSource should be populated")
	}

	system.Stop()
}

func TestSystem_Setters(t *testing.T) {
	system, _ := Initialize(Config{
		StoreEnabled: false,
	})

	// Test SetMatrixSender
	system.SetMatrixSender(&mockMatrixSender{})

	// Test SetMatrixAdapter
	system.SetMatrixAdapter(&mockMatrixAdapter{})

	// Test SetAdminMXID
	system.SetAdminMXID("@newadmin:example.com")
	if system.resolver.GetConfigAdmin() != "@newadmin:example.com" {
		t.Error("SetAdminMXID failed")
	}

	// Test SetSetupUser
	system.SetSetupUser("@setup:example.com")
	if system.resolver.GetSetupUser() != "@setup:example.com" {
		t.Error("SetSetupUser failed")
	}

	// Test SetAdminRoom
	system.SetAdminRoom("!room:example.com")
	if system.resolver.GetAdminRoom() != "!room:example.com" {
		t.Error("SetAdminRoom failed")
	}

	// Test SetEnabled
	system.SetEnabled(false)
	if system.IsEnabled() {
		t.Error("SetEnabled(false) failed")
	}

	system.Stop()
}

func TestSystem_Cleanup(t *testing.T) {
	storePath := testStorePath(t)
	defer cleanupStore(t, storePath)

	system, _ := Initialize(Config{
		StorePath:     storePath,
		StoreEnabled:  true,
		RetentionDays: 30,
	})

	// Cleanup with no old errors should succeed
	deleted, err := system.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if deleted != 0 {
		t.Errorf("Cleanup() deleted %d, want 0", deleted)
	}

	system.Stop()
}

func TestSystem_Getters(t *testing.T) {
	system, _ := Initialize(Config{StoreEnabled: false})

	if system.GetRegistry() == nil {
		t.Error("GetRegistry should return registry")
	}
	if system.GetResolver() == nil {
		t.Error("GetResolver should return resolver")
	}
	if system.GetNotifier() == nil {
		t.Error("GetNotifier should return notifier")
	}
	// Store is nil when disabled
	if system.GetStore() != nil {
		t.Error("GetStore should return nil when disabled")
	}

	system.Stop()
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.StorePath != "/var/lib/armorclaw/errors.db" {
		t.Errorf("Default StorePath = %q, want /var/lib/armorclaw/errors.db", cfg.StorePath)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("Default RetentionDays = %d, want 30", cfg.RetentionDays)
	}
	if !cfg.Enabled {
		t.Error("Default Enabled should be true")
	}
}

func TestGlobalSystem(t *testing.T) {
	storePath := testStorePath(t)
	defer cleanupStore(t, storePath)

	system, _ := Initialize(Config{
		StorePath:     storePath,
		StoreEnabled:  true,
		SetupUserMXID: "@admin:example.com",
	})

	SetGlobalSystem(system)

	if GetGlobalSystem() != system {
		t.Error("GetGlobalSystem should return the set system")
	}

	system.Stop()
}

func TestReport(t *testing.T) {
	mockSender := &mockMatrixSender{}

	_, err := Initialize(Config{
		MatrixSender:  mockSender,
		SetupUserMXID: "@admin:example.com",
		Enabled:       true,
		NotifyEnabled: true,
		StoreEnabled:  false,
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	reportErr := Report(context.Background(), "CTX-001", os.ErrNotExist)
	if reportErr != nil {
		t.Fatalf("Report() error = %v", reportErr)
	}

	if mockSender.callCount != 1 {
		t.Errorf("SendMessage called %d times, want 1", mockSender.callCount)
	}
}

func TestReportf(t *testing.T) {
	mockSender := &mockMatrixSender{}

	_, err := Initialize(Config{
		MatrixSender:  mockSender,
		SetupUserMXID: "@admin:example.com",
		Enabled:       true,
		NotifyEnabled: true,
		StoreEnabled:  false,
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	reportErr := Reportf(context.Background(), "CTX-001", "container %s failed", "abc123")
	if reportErr != nil {
		t.Fatalf("Reportf() error = %v", reportErr)
	}

	if mockSender.callCount != 1 {
		t.Errorf("SendMessage called %d times, want 1", mockSender.callCount)
	}
}

func TestTrack(t *testing.T) {
	ClearAllComponents()

	Track("test_event", map[string]any{"key": "value"})

	events := GetRecentEvents("errors", 1)
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input     string
		defaultMs int64
		expected  time.Duration
	}{
		{"5m", 0, 5 * time.Minute},
		{"24h", 0, 24 * time.Hour},
		{"", 60000, 60 * time.Second}, // default
		{"invalid", 30000, 30 * time.Second}, // fallback to default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input, tt.defaultMs)
			if result != tt.expected {
				t.Errorf("parseDuration(%q, %d) = %v, want %v", tt.input, tt.defaultMs, result, tt.expected)
			}
		})
	}
}
