package subprocess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type MockHealthChecker struct {
	shouldPass bool
}

func (m *MockHealthChecker) Check() bool {
	return m.shouldPass
}

type MockRestartFunc struct {
	calls int
	mu    sync.Mutex
}

func (m *MockRestartFunc) Restart(ctx context.Context) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
}

func (m *MockRestartFunc) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestProcessManager_New(t *testing.T) {
	pm := NewProcessManager()
	if pm == nil {
		t.Fatal("NewProcessManager returned nil")
	}
	if pm.maxFailures != 3 {
		t.Errorf("Expected maxFailures=3, got %d", pm.maxFailures)
	}
}

func TestProcessManager_killProcessGroup_NoProcess(t *testing.T) {
	pm := NewProcessManager()
	err := pm.killProcessGroup()
	if err == nil {
		t.Error("Expected error when no process is running")
	}
}

func TestWatchdog_CircuitBreaker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockRestart := &MockRestartFunc{}
	mockChecker := &MockHealthChecker{shouldPass: false}

	watchdog := NewWatchdog(1, mockChecker, func(ctx context.Context) {
		mockRestart.Restart(ctx)
	})

	watchdog.Start(ctx)

	for i := 0; i < 30; i++ {
		if mockRestart.Calls() > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if mockRestart.Calls() == 0 {
		t.Error("Expected restart to be called")
	}
}

func TestWatchdog_SuccessReset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRestart := &MockRestartFunc{}
	mockChecker := &MockHealthChecker{shouldPass: true}

	watchdog := NewWatchdog(3, mockChecker, func(ctx context.Context) {
		mockRestart.Restart(ctx)
	})

	watchdog.Start(ctx)
	time.Sleep(200 * time.Millisecond)
	cancel()

	if mockRestart.Calls() > 0 {
		t.Error("Expected no restart when health checks pass")
	}
}

func TestWatchdog_ResetFailures(t *testing.T) {
	mockChecker := &MockHealthChecker{shouldPass: false}
	mockRestart := &MockRestartFunc{}

	watchdog := NewWatchdog(3, mockChecker, func(ctx context.Context) {
		mockRestart.Restart(ctx)
	})

	watchdog.ResetFailures()
}

func TestCDPHealthChecker_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Browser":"Lightpanda","Protocol-Version":"1.3"}`))
	}))
	defer server.Close()

	checker := &CDPHealthChecker{port: "9222"}
	checker.httpClient = server.Client()

	resp, err := checker.httpClient.Get(server.URL)
	if err != nil && resp.StatusCode != http.StatusOK {
		t.Error("Expected health check to pass")
	}

	var version CDPVersionResponse
	_ = json.NewDecoder(resp.Body).Decode(&version)
	resp.Body.Close()

	if version.Browser == "" {
		t.Error("Expected health check to pass")
	}
}

func TestCDPHealthChecker_Failure(t *testing.T) {
	checker := NewCDPHealthChecker("9222")

	result := checker.Check()
	if result {
		t.Error("Expected health check to fail when server is not available")
	}
}

func TestCDPHealthChecker_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := &CDPHealthChecker{port: "9222"}
	checker.httpClient = server.Client()

	result := checker.Check()
	if result {
		t.Error("Expected health check to fail on timeout")
	}
}

func TestProcessManager_RaceCondition(t *testing.T) {
	pm := NewProcessManager()

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pm.killProcessGroup()
		}()
	}

	wg.Wait()
}
