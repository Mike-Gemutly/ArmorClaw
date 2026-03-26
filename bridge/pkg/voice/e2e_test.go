package voice

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// E2E tests require running Docker sidecars. Set ARMORCLAW_E2E=1 to enable.
// These tests verify connectivity to external voice services.

// TestE2EServiceHealth verifies all voice services are reachable
func TestE2EServiceHealth(t *testing.T) {
	if os.Getenv("ARMORCLAW_E2E") == "" {
		t.Skip("Skipping E2E test. Set ARMORCLAW_E2E=1 to enable.")
	}

	services := map[string]string{
		"VAD": getEnvOrDefault("VAD_URL", "http://localhost:8001") + "/health",
		"STT": getEnvOrDefault("STT_URL", "http://localhost:8002") + "/health",
		"TTS": getEnvOrDefault("TTS_URL", "http://localhost:8003") + "/health",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for name, url := range services {
		healthCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		req, err := http.NewRequestWithContext(healthCtx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			t.Errorf("%s: failed to create request: %v", name, err)
			continue
		}

		resp, err := client.Do(req)
		cancel()

		if err != nil {
			t.Errorf("%s service unreachable at %s: %v", name, url, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 500 {
			t.Errorf("%s service unhealthy (status %d)", name, resp.StatusCode)
		} else {
			t.Logf("%s service healthy at %s", name, url)
		}
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
