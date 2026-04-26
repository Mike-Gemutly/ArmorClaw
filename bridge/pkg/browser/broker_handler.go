package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/armorclaw/bridge/pkg/capability"
)

// BrokerHandler wraps a BrowserBroker into the same
// func(ctx context.Context, config json.RawMessage) (json.RawMessage, error)
// signature used by Handler(client). It dispatches browser actions the same
// way as handler.go:72-87 but calls broker methods instead of client methods.
//
// Each broker call is logged with method, jobID, duration, and error.
func BrokerHandler(broker BrowserBroker) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		var req browserRequest
		if err := json.Unmarshal(config, &req); err != nil {
			return nil, fmt.Errorf("browser_execute: parse config: %w", err)
		}

		intent := req.Intent
		if intent == nil {
			intent = &capability.BrowserIntent{
				URL:        req.URL,
				Action:     req.Action,
				FormFields: req.FormFields,
			}
		}

		if err := intent.Validate(); err != nil {
			return nil, fmt.Errorf("browser_execute: %w", err)
		}

		result, err := brokerDispatchAction(ctx, broker, intent)
		if err != nil {
			return nil, fmt.Errorf("browser_execute: %w", err)
		}

		raw, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("browser_execute: marshal result: %w", err)
		}
		return raw, nil
	}
}

func logBrokerCall(method string, jobID JobID, start time.Time, err error) {
	duration := time.Since(start)
	if err != nil {
		log.Printf("browser broker: %s job_id=%s duration=%v error=%v", method, jobID, duration, err)
	} else {
		log.Printf("browser broker: %s job_id=%s duration=%v ok", method, jobID, duration)
	}
}

// brokerDispatchAction dispatches a browser action to the broker.
// It produces the same BrowserResultCompat output as handler.go:72-87.
func brokerDispatchAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	switch intent.Action {
	case "navigate":
		return brokerNavigateAction(ctx, broker, intent)
	case "fill":
		return brokerFillAction(ctx, broker, intent)
	case "extract":
		return brokerExtractAction(ctx, broker, intent)
	case "screenshot":
		return brokerScreenshotAction(ctx, broker, intent)
	case "workflow":
		return brokerWorkflowAction(ctx, broker, intent)
	default:
		return nil, fmt.Errorf("unsupported browser action: %s", intent.Action)
	}
}

func brokerNavigateAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	// Start a job for this navigation.
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker navigate: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	br, err := broker.Navigate(ctx, jobID, intent.URL)
	logBrokerCall("navigate", jobID, start, err)
	if err != nil {
		return nil, err
	}

	return br.ToBrowserResult(), nil
}

func brokerFillAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker fill: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	var fields []FillRequest
	for _, ff := range intent.FormFields {
		fields = append(fields, FillRequest{Selector: ff})
	}

	br, err := broker.Fill(ctx, jobID, fields)
	logBrokerCall("fill", jobID, start, err)
	if err != nil {
		return nil, err
	}

	return br.ToBrowserResult(), nil
}

func brokerExtractAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker extract: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	spec := ExtractSpec{}
	for _, ff := range intent.FormFields {
		spec.Fields = append(spec.Fields, ExtractField{Name: ff, Selector: ff})
	}

	extractResult, err := broker.Extract(ctx, jobID, spec)
	logBrokerCall("extract", jobID, start, err)
	if err != nil {
		return nil, err
	}

	result := &BrowserResultCompat{URL: intent.URL}
	if extractResult != nil {
		for k, v := range extractResult.Fields {
			result.ExtractedData = append(result.ExtractedData, fmt.Sprintf("%s=%v", k, v))
		}
		if extractResult.Screenshot != "" {
			result.Screenshots = append(result.Screenshots, extractResult.Screenshot)
		}
	}
	return result, nil
}

func brokerScreenshotAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker screenshot: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	br, err := broker.Screenshot(ctx, jobID, false)
	logBrokerCall("screenshot", jobID, start, err)
	if err != nil {
		return nil, err
	}

	return br.ToBrowserResult(), nil
}

func brokerWorkflowAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()

	steps := make([]string, len(intent.FormFields))
	copy(steps, intent.FormFields)

	jobID, err := broker.StartJob(ctx, StartJobRequest{
		URL:   intent.URL,
		Steps: steps,
	})
	if err != nil {
		return nil, fmt.Errorf("broker workflow: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	// Execute navigate as the first step.
	br, err := broker.Navigate(ctx, jobID, intent.URL)
	if err != nil {
		logBrokerCall("workflow", jobID, start, err)
		return nil, err
	}

	result := br.ToBrowserResult()

	// Execute remaining steps via fill (matches handler.go workflow behavior).
	for _, step := range intent.FormFields {
		_, err := broker.Fill(ctx, jobID, []FillRequest{{Selector: step}})
		if err != nil {
			logBrokerCall("workflow", jobID, start, err)
			return nil, err
		}
	}

	logBrokerCall("workflow", jobID, start, nil)
	return result, nil
}

// ---------------------------------------------------------------------------
// TEMPORARY: Legacy Fallback Mechanism (Phase 4 removal target)
//
// The entire fallback mechanism below exists only during the Jetski migration.
// Once Jetski is the sole browser backend (Phase 4), this code and all
// references to ARMORCLAW_BROWSER_FALLBACK should be deleted.
// ---------------------------------------------------------------------------

// FallbackLog records a single fallback event for monitoring and auditing.
type FallbackLog struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"`
	JobID     string    `json:"job_id"`
	OrigError string    `json:"orig_error"`
	LegacyURL string    `json:"legacy_url"`
	Success   bool      `json:"success"`
}

// fallbackState holds the atomic counters for fallback monitoring.
var fallbackState struct {
	totalAttempts int64
	totalSuccess  int64
	totalFailures int64
	mu            sync.Mutex
	logs          []FallbackLog
}

// FallbackCount returns (attempts, successes, failures) for monitoring.
func FallbackCount() (attempts, successes, failures int64) {
	return atomic.LoadInt64(&fallbackState.totalAttempts),
		atomic.LoadInt64(&fallbackState.totalSuccess),
		atomic.LoadInt64(&fallbackState.totalFailures)
}

// FallbackLogs returns a snapshot of recent fallback event logs.
func FallbackLogs() []FallbackLog {
	fallbackState.mu.Lock()
	defer fallbackState.mu.Unlock()
	out := make([]FallbackLog, len(fallbackState.logs))
	copy(out, fallbackState.logs)
	return out
}

// fallbackMaxRetries returns the maximum number of consecutive fallback
// attempts allowed before the mechanism stops retrying.
// Controlled by ARMORCLAW_BROWSER_FALLBACK_MAX_RETRIES (default: 3).
func fallbackMaxRetries() int64 {
	if v := os.Getenv("ARMORCLAW_BROWSER_FALLBACK_MAX_RETRIES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

// fallbackEnabled returns true when the ARMORCLAW_BROWSER_FALLBACK env var
// is set to "true", "1", or "yes".
func fallbackEnabled() bool {
	v := os.Getenv("ARMORCLAW_BROWSER_FALLBACK")
	return v == "true" || v == "1" || v == "yes"
}

// isConnectionError detects network/connection-level failures.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	connSignals := []string{
		"connection refused",
		"timeout",
		"no such host",
		"network is unreachable",
		"connection reset",
		"EOF",
		"dial tcp",
		"i/o timeout",
	}
	for _, sig := range connSignals {
		if containsInsensitive(msg, sig) {
			return true
		}
	}
	return false
}

// containsInsensitive reports whether s contains substr (case-insensitive).
func containsInsensitive(s, substr string) bool {
	sl, subl := len(s), len(substr)
	if subl > sl {
		return false
	}
	for i := 0; i <= sl-subl; i++ {
		match := true
		for j := 0; j < subl; j++ {
			sc, subc := s[i+j], substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// extractOperation parses the browser action from the raw JSON config.
func extractOperation(config json.RawMessage) string {
	var partial struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(config, &partial); err != nil {
		return "unknown"
	}
	if partial.Action == "" {
		return "unknown"
	}
	return partial.Action
}

// extractJobID best-effort extracts a job_id from the error message.
func extractJobID(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	const prefix = "job_id="
	idx := len(msg) - len(prefix)
	for i := 0; i <= idx; i++ {
		if msg[i:i+len(prefix)] == prefix {
			j := i + len(prefix)
			for j < len(msg) && msg[j] != ' ' && msg[j] != ')' {
				j++
			}
			return msg[i+len(prefix) : j]
		}
	}
	return ""
}

// FallbackHandler wraps a primary (Jetski) handler and falls back to a
// secondary (legacy) handler when the primary fails with a connection error.
//
// TEMPORARY: This entire function is a migration escape hatch.
// Remove in Phase 4 when Jetski is the sole backend.
//
// Fallback is gated by ARMORCLAW_BROWSER_FALLBACK (must be "true"/"1"/"yes").
// Infinite loops are prevented by ARMORCLAW_BROWSER_FALLBACK_MAX_RETRIES
// (default 3).
func FallbackHandler(primary, fallback func(ctx context.Context, config json.RawMessage) (json.RawMessage, error)) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		result, err := primary(ctx, config)
		if err == nil {
			return result, nil
		}

		if !fallbackEnabled() || !isConnectionError(err) {
			return nil, err
		}

		attempts := atomic.LoadInt64(&fallbackState.totalAttempts)
		maxRetries := fallbackMaxRetries()
		if attempts >= maxRetries {
			log.Printf("WARNING: browser fallback: max retries (%d) reached, not attempting fallback for error: %v", maxRetries, err)
			return nil, err
		}

		operation := extractOperation(config)
		jobID := extractJobID(err)
		legacyURL := os.Getenv("ARMORCLAW_BROWSER_LEGACY_URL")

		atomic.AddInt64(&fallbackState.totalAttempts, 1)

		log.Printf("WARNING: browser fallback: jetski connection failed, retrying via legacy (op=%s jobID=%s legacyURL=%s): %v",
			operation, jobID, legacyURL, err)

		legacyResult, legacyErr := fallback(ctx, config)

		fbLog := FallbackLog{
			Timestamp: time.Now().UTC(),
			Operation: operation,
			JobID:     jobID,
			OrigError: err.Error(),
			LegacyURL: legacyURL,
		}

		if legacyErr != nil {
			fbLog.Success = false
			atomic.AddInt64(&fallbackState.totalFailures, 1)
			fallbackState.mu.Lock()
			fallbackState.logs = append(fallbackState.logs, fbLog)
			fallbackState.mu.Unlock()
			log.Printf("WARNING: browser fallback: legacy retry also failed (op=%s): %v", operation, legacyErr)
			return nil, fmt.Errorf("jetski error: %w; legacy fallback also failed: %v", err, legacyErr)
		}

		fbLog.Success = true
		atomic.AddInt64(&fallbackState.totalSuccess, 1)
		fallbackState.mu.Lock()
		fallbackState.logs = append(fallbackState.logs, fbLog)
		fallbackState.mu.Unlock()
		log.Printf("WARNING: browser fallback: legacy retry succeeded (op=%s jobID=%s)", operation, jobID)
		return legacyResult, nil
	}
}
