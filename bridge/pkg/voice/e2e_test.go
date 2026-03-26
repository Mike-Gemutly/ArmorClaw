package voice

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

 
// E2E tests require running Docker sidecars. Set ARMORCLAW_E2E=1 to enable.
// These tests use select/time.After patterns to avoid deadlocks.
 
func TestE2EPipelineWithRealServices(t *testing.T)
 {
	if os.Getenv("ARMORCLAW_E2E") == "" {
		t.Skip("Skipping E2E test. Set ARMORCLAW_E2E=1 to enable.")
	 }
 
	// Get service URLs from environment (set by Docker sidecars)
	vadURL := getEnvOrDefault("VAD_URL", "http://localhost:8001")
	sttURL := getEnvOrDefault("STT_URL", "http://localhost:8002")
	ttsURL := getEnvOrDefault("TTS_URL", "http://localhost:8003")
 
	config := PipelineConfig{
		STTConfig: STTClientConfig{
			WhisperURL: sttURL,
			Timeout:    10 * time.Second,
			MaxRetries: 3,
		},
		TTSConfig: TTSClientConfig{
			PiperURL:    ttsURL,
			Timeout:     10 * time.Second,
			MaxRetries:  3,
			MaxTextSize: 5000,
		},
		VADConfig: VADClientConfig{
			SileroVADURL: vadURL,
			Timeout:      10 * time.Second,
			MaxRetries:   3,
			Threshold:    0.5,
		},
		HitlTimeout:  30 * time.Second,
		MaxChunkSize: 4096,
	}
 
	pipeline, err := NewVoicePipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
 
	// Start pipeline with timeout protection
	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startCancel()
 
	startDone := make(chan error, 1)
	go func() {
		startDone <- pipeline.Start()
	}()
 
	select {
	case err := <-startDone:
		if err != nil {
			t.Fatalf("Failed to start pipeline: %v", err)
		}
	case <-startCtx.Done():
		t.Fatal("Pipeline start timed out")
	}
	defer pipeline.Stop()
 
	// Send audio chunk with timeout protection
	audioData := make([]byte, 1024) // Simulated audio chunk
	chunk := AudioChunk{
		Data:      audioData,
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}
 
	processCtx, processCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer processCancel()
 
	processDone := make(chan error, 1)
	go func() {
		processDone <- pipeline.ProcessAudio(chunk)
	}()
 
	select {
	case err := <-processDone:
		if err != nil {
		t.Logf("ProcessAudio returned error (may be expected for mock audio): %v", err)
		}
	case <-processCtx.Done():
		t.Fatal("ProcessAudio timed out - possible deadlock")
	}
 
	// Wait for pipeline to process with time.After gate (non-blocking)
	select {
	case <-time.After(2 * time.Second):
		// Expected path - processing complete or idle
	case <-waitForStateChange(pipeline, PipelineStateIdle, 5*time.Second):
		// Pipeline returned to idle
	}
 
	state := pipeline.GetState()
	t.Logf("Pipeline state after E2E processing: %s", state.String())
}
 
// TestE2EPipelinePauseResume tests pause/resume with select/time.After pattern
func TestE2EPipelinePauseResume(t *testing.T)
 {
	if os.Getenv("ARMORCLAW_E2E") == "" {
		t.Skip("Skipping E2E test. Set ARMORCLAW_E2E=1 to enable.")
	 }
 
	vadURL := getEnvOrDefault("VAD_URL", "http://localhost:8001")
	sttURL := getEnvOrDefault("STT_URL", "http://localhost:8002")
	ttsURL := getEnvOrDefault("TTS_URL", "http://localhost:8003")
 
	config := PipelineConfig{
		STTConfig: STTClientConfig{
			WhisperURL: sttURL,
			Timeout:    5 * time.Second,
			MaxRetries: 2,
		},
		TTSConfig: TTSClientConfig{
			PiperURL:    ttsURL,
			Timeout:     5 * time.Second,
			MaxRetries:  2,
			MaxTextSize: 5000,
		},
		VADConfig: VADClientConfig{
			SileroVADURL: vadURL,
			Timeout:      5 * time.Second,
			MaxRetries:   2,
			Threshold:    1.5,
		},
		HitlTimeout:  30 * time.Second,
		MaxChunkSize: 4096,
	}
 
	pipeline, err := NewVoicePipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
 
	if err := pipeline.Start(); err != nil {
		t.Fatalf("Failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()
 
	// Pause with timeout
	pauseDone := make(chan error, 1)
	go func() {
		pauseDone <- pipeline.Pause()
	}()
 
	select {
	case err := <-pauseDone:
		if err != nil {
		t.Fatalf("Pause failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Pause timed out - possible deadlock")
	}
 
	if state := pipeline.GetState(); state != PipelineStatePaused {
		t.Errorf("Expected paused state, got %s", state.String())
	}
 
	// Resume with timeout
	resumeDone := make(chan error, 1)
	go func() {
		resumeDone <- pipeline.Resume()
	}()
 
	select {
	case err := <-resumeDone:
		if err != nil {
		t.Fatalf("Resume failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Resume timed out - possible deadlock")
	}
 
	if state := pipeline.GetState(); state != PipelineStateIdle {
		t.Errorf("Expected idle state after resume, got %s", state.String())
	}
}
 
// TestE2EPipelineStopRecovery tests graceful stop and restart
func TestE2EPipelineStopRecovery(t *testing.T)
 {
	if os.Getenv("ARMORCLAW_E2E") == "" {
		t.Skip("Skipping E2E test. Set ARMORCLAW_E2E=1 to enable.")
	 }
 
	config := PipelineConfig{
		STTConfig: STTClientConfig{
			WhisperURL: getEnvOrDefault("STT_URL", "http://localhost:8002"),
			Timeout:    5 * time.Second,
			MaxRetries: 2,
		},
		TTSConfig: TTSClientConfig{
			PiperURL:    getEnvOrDefault("TTS_URL", "http://localhost:8003"),
			Timeout:     5 * time.Second,
			MaxRetries:  2,
			MaxTextSize: 5000,
		},
		VADConfig: VADClientConfig{
			SileroVADURL: getEnvOrDefault("VAD_URL", "http://localhost:8001"),
			Timeout:      5 * time.Second,
			MaxRetries:   2,
			Threshold:    1.5,
		},
		HitlTimeout:  30 * time.Second,
		MaxChunkSize: 4096,
	}
 
	pipeline, err := NewVoicePipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
 
	// Start
	if err := pipeline.Start(); err != nil {
		t.Fatalf("Failed to start pipeline: %v", err)
	}
 
	// Stop with timeout protection
	stopDone := make(chan error, 1)
	go func() {
		stopDone <- pipeline.Stop()
	}()
 
	select {
	case err := <-stopDone:
		if err != nil {
		t.Fatalf("Stop failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Stop timed out - possible deadlock in processLoop")
	}
 
	// Verify we can restart
	if err := pipeline.Start(); err != nil {
		t.Fatalf("Failed to restart pipeline: %v", err)
	}
 
	// Clean stop
	pipeline.Stop()
}
 
// TestE2EServiceHealth verifies all services are reachable
func TestE2EServiceHealth(t *testing.T)
 {
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
 
// TestE2ETTSService tests TTS service wrapper with real Piper
func TestE2ETTSService(t *testing.T)
 {
	if os.Getenv("ARMORCLAW_E2E") == "" {
		t.Skip("Skipping E2E test. Set ARMORCLAW_E2E=1 to enable.")
	 }
 
	ttsURL := getEnvOrDefault("TTS_URL", "http://localhost:8003")
 
	client := NewTTSClient(TTSClientConfig{
		PiperURL:    ttsURL,
		Timeout:     10 * time.Second,
		MaxRetries:  2,
		MaxTextSize: 5000,
	})
 
	service := NewTTSService(client)
 
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
 
	result, err := service.Synthesize(ctx, "Hello, this is a test.")
	if err != nil {
		t.Fatalf("TTS synthesis failed: %v", err)
	 }
 
	if len(result.AudioData) == 0 {
		t.Error("Expected audio data from TTS synthesis")
    }
 
	t.Logf("TTS synthesis complete: %d bytes audio, duration %v", len(result.AudioData), result.Duration)
}
 
// TestE2EVADService tests VAD service wrapper with real Silero
func TestE2EVADService(t *testing.T)
 {
	if os.Getenv("ARMORCLAW_E2E") == "" {
		t.Skip("Skipping E2E test. Set ARMORCLAW_E2E=1 to enable.")
    }
 
	vadURL := getEnvOrDefault("VAD_URL", "http://localhost:8001")
 
	client := NewVADClient(VADClientConfig{
		SileroVADURL: vadURL,
		Timeout:      10 * time.Second,
		MaxRetries:   2,
		Threshold:    1.5,
	 })
 
	service := NewVADService(client)
 
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
 
	// Create test audio data (silence - should not detect speech)
	silenceData := make([]byte, 1024)
 
	result, err := service.DetectSpeech(ctx, silenceData)
	if err != nil {
		t.Fatalf("VAD detection failed: %v", err)
    }
 
	t.Logf("VAD result: speech=%v confidence=%.2f latency=%v",
        result.SpeechDetected, result.Confidence, result.Latency)
}
 
// waitForStateChange returns a channel that closes when state matches or timeout
func waitForStateChange(pipeline *VoicePipeline, target PipelineState, timeout time.Duration) <-chan struct{} {
    done := make(chan struct{})
 
    go func() {
        defer close(done)
        ticker := time.NewTicker(50 * time.Millisecond)
        defer ticker.Stop()
 
        timeoutCh := time.After(timeout)
 
        for {
            select {
            case <-ticker.C:
                if pipeline.GetState() == target {
                    return
                }
            case <-timeoutCh:
                return
            }
        }
    }()
 
    return done
}()
 
func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
