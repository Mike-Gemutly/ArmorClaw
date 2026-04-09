package subprocess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type LightpandaVersionResponse struct {
	Browser              string `json:"Browser"`
	ProtocolVersion      string `json:"Protocol-Version"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func TestIntegration_RestartCycle(t *testing.T) {
	versionResponse := LightpandaVersionResponse{
		Browser:              "Lightpanda/0.2.6",
		ProtocolVersion:      "1.3",
		WebSocketDebuggerURL: "ws://localhost:9222",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionResponse)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	restarter := NewRestarter()
	restarter.baseDelay = 10 * time.Millisecond

	failureCount := 0
	checker := &mockHealthChecker{
		serverURL: server.URL,
		failCount: &failureCount,
	}

	healthPassed := make(chan bool, 1)
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(50 * time.Millisecond)
			healthPassed <- checker.Check()
		}
		close(healthPassed)
	}()

	passCount := 0
	for passed := range healthPassed {
		if passed {
			passCount++
		}
	}

	if passCount == 0 {
		t.Errorf("Expected at least one successful health check, got %d", passCount)
	}

	t.Logf("Integration test completed: %d passes, %d failures", passCount, failureCount)
}

func TestIntegration_RestartAfterMaxFailures(t *testing.T) {
	versionResponse := LightpandaVersionResponse{
		Browser:         "Lightpanda/0.2.6",
		ProtocolVersion: "1.3",
	}

	failures := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" {
			failures++
			if failures <= 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionResponse)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	restarter := NewRestarter()
	restarter.baseDelay = 5 * time.Millisecond
	restarter.maxRestarts = 3

	restartAttempts := 0
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	restartFunc := func(ctx context.Context) error {
		restartAttempts++
		t.Logf("Restart attempt #%d", restartAttempts)
		return nil
	}

	for i := 0; i < 2; i++ {
		_ = restarter.restartEngine(ctx, restartFunc)
	}

	if restartAttempts != 2 {
		t.Errorf("Expected 2 restart attempts, got %d", restartAttempts)
	}

	if restarter.GetRestartCount() != 2 {
		t.Errorf("Expected restart count 2, got %d", restarter.GetRestartCount())
	}
}

func TestIntegration_HealthCheckTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" {
			time.Sleep(3 * time.Second)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &mockHealthChecker{
		serverURL: server.URL,
	}

	start := time.Now()
	result := checker.Check()
	elapsed := time.Since(start)

	if result {
		t.Error("Expected health check to fail due to timeout")
	}

	if elapsed < 2*time.Second {
		t.Errorf("Expected timeout around 2s, got %v", elapsed)
	}

	if elapsed > 3*time.Second {
		t.Errorf("Expected timeout around 2s, but took too long: %v", elapsed)
	}
}

func TestIntegration_ConcurrentHealthChecks(t *testing.T) {
	versionResponse := LightpandaVersionResponse{
		Browser:         "Lightpanda/0.2.6",
		ProtocolVersion: "1.3",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" {
			time.Sleep(50 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionResponse)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &mockHealthChecker{
		serverURL: server.URL,
	}

	results := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			results <- checker.Check()
		}()
	}

	passCount := 0
	for i := 0; i < 10; i++ {
		if <-results {
			passCount++
		}
	}

	if passCount != 10 {
		t.Errorf("Expected all 10 health checks to pass, got %d", passCount)
	}
}

type mockHealthChecker struct {
	serverURL string
	failCount *int
}

func (m *mockHealthChecker) Check() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", m.serverURL+"/json/version", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if m.failCount != nil {
			*m.failCount++
		}
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
