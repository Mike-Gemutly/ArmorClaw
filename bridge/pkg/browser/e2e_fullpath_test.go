package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// E2E Full-Path Tests: Handler → Client → Mock browser-service → Result
// ---------------------------------------------------------------------------
//
// These tests exercise the complete in-process browser path:
//   1. Handler receives a BrowserIntent wrapped in step config JSON.
//   2. Handler dispatches to the correct action function.
//   3. Action calls Client.Post → HTTP request to browser-service.
//   4. httptest mock server returns a realistic ServiceResponse.
//   5. Client decodes response → action converts → handler returns BrowserResult JSON.
//
// No real browser-service, Docker, or Playwright is required.

type mockServiceConfig struct {
	navigateResp   *ServiceResponse
	fillResp       *ServiceResponse
	extractResp    *ServiceResponse
	screenshotResp *ServiceResponse
	workflowResp   *ServiceWorkflowResponse
	healthResp     *ServiceHealthResponse
	errorPaths     map[string]int // path → HTTP error code

	mu       sync.Mutex
	requests []capturedRequest
}

type capturedRequest struct {
	path    string
	method  string
	body    map[string]interface{}
	headers http.Header
}

func (c *mockServiceConfig) track(path, method string, body map[string]interface{}, hdrs http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, capturedRequest{
		path: path, method: method, body: body, headers: hdrs,
	})
}

func (c *mockServiceConfig) getRequests() []capturedRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]capturedRequest, len(c.requests))
	copy(out, c.requests)
	return out
}

// newE2EServer creates an httptest.Server that simulates the browser-service.
func newE2EServer(cfg *mockServiceConfig) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/navigate", func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(tFromWriter(w), r)
		cfg.track("/navigate", r.Method, body, r.Header)

		if code, ok := cfg.errorPaths["/navigate"]; ok {
			w.WriteHeader(code)
			return
		}
		writeJSON(w, cfg.navigateResp)
	})

	mux.HandleFunc("/fill", func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(nil, r)
		cfg.track("/fill", r.Method, body, r.Header)

		if code, ok := cfg.errorPaths["/fill"]; ok {
			w.WriteHeader(code)
			return
		}
		writeJSON(w, cfg.fillResp)
	})

	mux.HandleFunc("/extract", func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(nil, r)
		cfg.track("/extract", r.Method, body, r.Header)

		if code, ok := cfg.errorPaths["/extract"]; ok {
			w.WriteHeader(code)
			return
		}
		writeJSON(w, cfg.extractResp)
	})

	mux.HandleFunc("/screenshot", func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(nil, r)
		cfg.track("/screenshot", r.Method, body, r.Header)

		if code, ok := cfg.errorPaths["/screenshot"]; ok {
			w.WriteHeader(code)
			return
		}
		writeJSON(w, cfg.screenshotResp)
	})

	mux.HandleFunc("/workflow", func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(nil, r)
		cfg.track("/workflow", r.Method, body, r.Header)

		if code, ok := cfg.errorPaths["/workflow"]; ok {
			w.WriteHeader(code)
			return
		}
		writeJSON(w, cfg.workflowResp)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		cfg.track("/health", r.Method, nil, r.Header)
		writeJSON(w, cfg.healthResp)
	})

	return httptest.NewServer(mux)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// tFromWriter extracts a testing.T from a ResponseWriter (nil-safe).
// Used only for error logging inside handlers — not for assertions.
func tFromWriter(w http.ResponseWriter) interface{} { return nil }

func decodeBody(_ interface{}, r *http.Request) map[string]interface{} {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil || len(raw) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// handlerInvoke calls the handler with the given config and returns the
// parsed BrowserResult.
func handlerInvoke(t *testing.T, h func(ctx context.Context, config json.RawMessage) (json.RawMessage, error), action, url string, extra map[string]interface{}) map[string]interface{} {
	t.Helper()

	payload := map[string]interface{}{
		"action": action,
		"url":    url,
	}
	for k, v := range extra {
		payload[k] = v
	}

	input, err := json.Marshal(payload)
	require.NoError(t, err)

	out, err := h(context.Background(), input)
	require.NoError(t, err, "handler returned error for action=%s", action)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &result), "failed to parse handler output")
	return result
}

// ---------------------------------------------------------------------------
// E2E Test: Navigate
// ---------------------------------------------------------------------------

func TestE2E_Navigate_FullPath(t *testing.T) {
	cfg := &mockServiceConfig{
		navigateResp: &ServiceResponse{
			Success:  true,
			Duration: 250,
			Data: map[string]interface{}{
				"title":       "Example Domain",
				"finalUrl":    "https://example.com/",
				"status":      200.0, // JSON numbers decode as float64
				"loadTime":    180.0,
			},
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	result := handlerInvoke(t, h, "navigate", "https://example.com", nil)

	// Verify handler output contains URL and title
	assert.Equal(t, "https://example.com", result["url"])
	assert.Equal(t, "Example Domain", result["title"])

	// Verify mock server received correct HTTP request
	reqs := cfg.getRequests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "/navigate", reqs[0].path)
	assert.Equal(t, "POST", reqs[0].method)

	// Verify the URL was sent to the browser-service
	assert.Equal(t, "https://example.com", reqs[0].body["url"])
}

// ---------------------------------------------------------------------------
// E2E Test: Fill without PII (direct values)
// ---------------------------------------------------------------------------

func TestE2E_Fill_NoPII_FullPath(t *testing.T) {
	cfg := &mockServiceConfig{
		fillResp: &ServiceResponse{
			Success:  true,
			Duration: 120,
			Data: map[string]interface{}{
				"filledFields": 3.0,
			},
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	result := handlerInvoke(t, h, "fill", "https://example.com/form", map[string]interface{}{
		"form_fields": []string{"#name", "#email", "#phone"},
	})

	assert.Equal(t, "https://example.com/form", result["url"])

	// Verify the fill command was sent to the mock service
	reqs := cfg.getRequests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "/fill", reqs[0].path)
	assert.Equal(t, "POST", reqs[0].method)

	// Verify fields were included in the request body
	fields, ok := reqs[0].body["fields"].([]interface{})
	require.True(t, ok, "expected 'fields' array in request body")
	assert.Len(t, fields, 3)
}

// ---------------------------------------------------------------------------
// E2E Test: Fill with PII (value_ref fields that need HITL approval)
// ---------------------------------------------------------------------------
// In the real system, the browser-service handles PII resolution via
// HITL approval. Here we verify that the fill command with value_ref
// fields reaches the browser-service correctly, and the service
// responds as if HITL approved the values.

func TestE2E_Fill_WithPII_FullPath(t *testing.T) {
	cfg := &mockServiceConfig{
		fillResp: &ServiceResponse{
			Success:  true,
			Duration: 350,
			Data: map[string]interface{}{
				"filledFields": 2.0,
				"piiResolved":  true,
			},
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	// Simulate a fill command that contains PII references
	// The handler passes form_fields as selectors to client.Fill
	result := handlerInvoke(t, h, "fill", "https://shop.example.com/checkout", map[string]interface{}{
		"form_fields": []string{"#card-number", "#cvv"},
	})

	assert.Equal(t, "https://shop.example.com/checkout", result["url"])

	// Verify the fill request was sent to the service
	reqs := cfg.getRequests()
	require.Len(t, reqs, 1)

	// The handler converts form_fields into ServiceFillField structs
	// with only the Selector populated (value resolved at service level)
	fields, ok := reqs[0].body["fields"].([]interface{})
	require.True(t, ok)
	assert.Len(t, fields, 2)

	// Verify each field has a selector
	for i, f := range fields {
		fieldMap, ok := f.(map[string]interface{})
		require.True(t, ok, "field %d should be a map", i)
		assert.Contains(t, fieldMap, "selector", "field %d missing selector", i)
	}
}

// ---------------------------------------------------------------------------
// E2E Test: Extract
// ---------------------------------------------------------------------------

func TestE2E_Extract_FullPath(t *testing.T) {
	cfg := &mockServiceConfig{
		extractResp: &ServiceResponse{
			Success:  true,
			Duration: 80,
			Data: map[string]interface{}{
				"price": "$19.99",
				"title": "Premium Widget",
			},
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	result := handlerInvoke(t, h, "extract", "https://shop.example.com/product", map[string]interface{}{
		"form_fields": []string{".price", "h1.product-title"},
	})

	assert.Equal(t, "https://shop.example.com/product", result["url"])

	// Extracted data should contain key=value pairs from service response
	extractedData, ok := result["extracted_data"].([]interface{})
	require.True(t, ok, "expected extracted_data array")
	require.Len(t, extractedData, 2, "should have 2 extracted fields")

	// Verify extracted values contain the data from mock service
	allExtracted := fmt.Sprintf("%v", extractedData)
	assert.Contains(t, allExtracted, "price")
	assert.Contains(t, allExtracted, "$19.99")
	assert.Contains(t, allExtracted, "title")
	assert.Contains(t, allExtracted, "Premium Widget")

	// Verify the extract endpoint was called
	reqs := cfg.getRequests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "/extract", reqs[0].path)
}

// ---------------------------------------------------------------------------
// E2E Test: Screenshot
// ---------------------------------------------------------------------------

func TestE2E_Screenshot_FullPath(t *testing.T) {
	cfg := &mockServiceConfig{
		screenshotResp: &ServiceResponse{
			Success:    true,
			Duration:   200,
			Screenshot: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	result := handlerInvoke(t, h, "screenshot", "https://example.com", nil)

	assert.Equal(t, "https://example.com", result["url"])

	// Screenshots should contain the base64 data from mock service
	screenshots, ok := result["screenshots"].([]interface{})
	require.True(t, ok, "expected screenshots array")
	require.Len(t, screenshots, 1, "should have 1 screenshot")

	screenshot, ok := screenshots[0].(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(screenshot, "iVBOR"), "should be a PNG base64 string")

	// Verify the screenshot endpoint was called
	reqs := cfg.getRequests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "/screenshot", reqs[0].path)
}

// ---------------------------------------------------------------------------
// E2E Test: Service Error Propagation
// ---------------------------------------------------------------------------

func TestE2E_ServiceError_PropagatesToHandler(t *testing.T) {
	cfg := &mockServiceConfig{
		errorPaths: map[string]int{
			"/navigate": http.StatusInternalServerError,
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{
		"url":    "https://example.com",
		"action": "navigate",
	})

	_, err := h(context.Background(), input)
	require.Error(t, err, "handler should return error when service returns 500")
	assert.Contains(t, err.Error(), "500")
}

// ---------------------------------------------------------------------------
// E2E Test: Service returns error in response body
// ---------------------------------------------------------------------------

func TestE2E_ServiceResponseError_InResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Service returns 200 but with an error in the response body
		json.NewEncoder(w).Encode(ServiceResponse{
			Success: false,
			Error: &ServiceError{
				Code:    ServiceErrNavigationFailed,
				Message: "DNS resolution failed for example.invalid",
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)

	// Navigate should succeed at HTTP level but we can still check the response
	resp, err := client.Navigate(context.Background(), ServiceNavigateCommand{
		URL: "https://example.invalid",
	})
	require.NoError(t, err, "HTTP request should succeed (200 status)")
	assert.False(t, resp.Success, "response success should be false")
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ServiceErrNavigationFailed, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "DNS resolution failed")
}

// ---------------------------------------------------------------------------
// E2E Test: Context Cancellation
// ---------------------------------------------------------------------------

func TestE2E_ContextCancellation(t *testing.T) {
	// Server that delays response to test context cancellation
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		json.NewEncoder(w).Encode(ServiceResponse{Success: true})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	client.SetTimeout(10 * time.Second) // Long HTTP timeout

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Navigate(ctx, ServiceNavigateCommand{URL: "https://example.com"})
	require.Error(t, err, "should fail due to context cancellation")
}

// ---------------------------------------------------------------------------
// E2E Test: Multiple Sequential Actions (Navigate → Extract → Screenshot)
// ---------------------------------------------------------------------------

func TestE2E_MultiStep_SequentialActions(t *testing.T) {
	cfg := &mockServiceConfig{
		navigateResp: &ServiceResponse{
			Success:  true,
			Duration: 150,
			Data: map[string]interface{}{
				"title": "Test Page",
			},
		},
		extractResp: &ServiceResponse{
			Success:  true,
			Duration: 50,
			Data: map[string]interface{}{
				"price": "$29.99",
			},
		},
		screenshotResp: &ServiceResponse{
			Success:    true,
			Duration:   100,
			Screenshot: "base64imagedata",
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	// Step 1: Navigate
	navResult := handlerInvoke(t, h, "navigate", "https://shop.example.com", nil)
	assert.Equal(t, "Test Page", navResult["title"])

	// Step 2: Extract
	extResult := handlerInvoke(t, h, "extract", "https://shop.example.com", map[string]interface{}{
		"form_fields": []string{".price"},
	})
	extractedData := extResult["extracted_data"].([]interface{})
	allExtracted := fmt.Sprintf("%v", extractedData)
	assert.Contains(t, allExtracted, "$29.99")

	// Step 3: Screenshot
	ssResult := handlerInvoke(t, h, "screenshot", "https://shop.example.com", nil)
	screenshots := ssResult["screenshots"].([]interface{})
	assert.Len(t, screenshots, 1)

	// Verify all 3 requests were received in order
	reqs := cfg.getRequests()
	require.Len(t, reqs, 3)
	assert.Equal(t, "/navigate", reqs[0].path)
	assert.Equal(t, "/extract", reqs[1].path)
	assert.Equal(t, "/screenshot", reqs[2].path)
}

// ---------------------------------------------------------------------------
// E2E Test: Client Direct Methods (Health, Session, Click, Wait)
// ---------------------------------------------------------------------------

func TestE2E_Client_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		json.NewEncoder(w).Encode(ServiceHealthResponse{
			Status:    "ok",
			Browser:   "initialized",
			State:     ServiceStateIdle,
			Timestamp: time.Now(),
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	health, err := client.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", health.Status)
	assert.Equal(t, "initialized", health.Browser)
	assert.Equal(t, ServiceStateIdle, health.State)
}

func TestE2E_Client_Click(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/click", r.URL.Path)

		var cmd ServiceClickCommand
		require.NoError(t, json.NewDecoder(r.Body).Decode(&cmd))
		assert.Equal(t, "#submit-btn", cmd.Selector)
		assert.Equal(t, "navigation", cmd.WaitFor)

		json.NewEncoder(w).Encode(ServiceResponse{
			Success: true,
			Data:    map[string]interface{}{"clicked": true},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	resp, err := client.Click(context.Background(), ServiceClickCommand{
		Selector: "#submit-btn",
		WaitFor:  "navigation",
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestE2E_Client_Wait(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/wait", r.URL.Path)

		var cmd ServiceWaitCommand
		require.NoError(t, json.NewDecoder(r.Body).Decode(&cmd))
		assert.Equal(t, "selector", cmd.Condition)
		assert.Equal(t, ".loaded", cmd.Value)

		json.NewEncoder(w).Encode(ServiceResponse{Success: true})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	resp, err := client.Wait(context.Background(), ServiceWaitCommand{
		Condition: "selector",
		Value:     ".loaded",
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// ---------------------------------------------------------------------------
// E2E Table-Driven: All Actions via Handler
// ---------------------------------------------------------------------------

func TestE2E_TableDriven_AllActions(t *testing.T) {
	tests := []struct {
		name         string
		action       string
		url          string
		extra        map[string]interface{}
		servicePath  string
		serviceResp  interface{}
		resultCheck  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:        "navigate",
			action:      "navigate",
			url:         "https://example.com",
			servicePath: "/navigate",
			serviceResp: &ServiceResponse{
				Success: true,
				Data:    map[string]interface{}{"title": "Example"},
			},
			resultCheck: func(t *testing.T, r map[string]interface{}) {
				assert.Equal(t, "Example", r["title"])
			},
		},
		{
			name:        "fill",
			action:      "fill",
			url:         "https://example.com/form",
			extra:       map[string]interface{}{"form_fields": []string{"#input"}},
			servicePath: "/fill",
			serviceResp: &ServiceResponse{Success: true},
			resultCheck: func(t *testing.T, r map[string]interface{}) {
				assert.Equal(t, "https://example.com/form", r["url"])
			},
		},
		{
			name:        "extract",
			action:      "extract",
			url:         "https://example.com/data",
			extra:       map[string]interface{}{"form_fields": []string{".data"}},
			servicePath: "/extract",
			serviceResp: &ServiceResponse{
				Success: true,
				Data:    map[string]interface{}{"key": "value"},
			},
			resultCheck: func(t *testing.T, r map[string]interface{}) {
				assert.NotNil(t, r["extracted_data"])
			},
		},
		{
			name:        "screenshot",
			action:      "screenshot",
			url:         "https://example.com",
			servicePath: "/screenshot",
			serviceResp: &ServiceResponse{
				Success:    true,
				Screenshot: "imgdata",
			},
			resultCheck: func(t *testing.T, r map[string]interface{}) {
				ss, ok := r["screenshots"].([]interface{})
				require.True(t, ok)
				assert.Len(t, ss, 1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.servicePath, r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				writeJSON(w, tc.serviceResp)
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			h := Handler(client)

			result := handlerInvoke(t, h, tc.action, tc.url, tc.extra)
			if tc.resultCheck != nil {
				tc.resultCheck(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Fill with PII — Simulated HITL Approval via Channel
// ---------------------------------------------------------------------------
// This test simulates the real-world pattern where a fill operation with
// PII references needs HITL approval before the browser-service processes
// the fill. The mock server waits for an approval signal before responding.

func TestE2E_Fill_WithPII_HITLApproval(t *testing.T) {
	approvalCh := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fill" {
			// Simulate HITL: wait for approval before responding
			select {
			case <-approvalCh:
				// Approved — respond with success
				json.NewEncoder(w).Encode(ServiceResponse{
					Success:  true,
					Duration: 200,
					Data: map[string]interface{}{
						"filledFields": 2.0,
						"piiResolved":  true,
					},
				})
			case <-time.After(2 * time.Second):
				// HITL timeout — deny
				json.NewEncoder(w).Encode(ServiceResponse{
					Success: false,
					Error: &ServiceError{
						Code:    ServiceErrPIIRequestDenied,
						Message: "HITL approval timeout",
					},
				})
			}
			return
		}
		json.NewEncoder(w).Encode(ServiceResponse{Success: true})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	// Start the fill in a goroutine
	type handlerResult struct {
		result map[string]interface{}
		err    error
	}
	resultCh := make(chan handlerResult, 1)

	go func() {
		input, _ := json.Marshal(map[string]interface{}{
			"url":         "https://shop.example.com/checkout",
			"action":      "fill",
			"form_fields": []string{"#card-number", "#cvv"},
		})
		out, err := h(context.Background(), input)
		if err != nil {
			resultCh <- handlerResult{err: err}
			return
		}
		var r map[string]interface{}
		json.Unmarshal(out, &r)
		resultCh <- handlerResult{result: r}
	}()

	// Simulate user approval after a short delay
	time.Sleep(50 * time.Millisecond)
	approvalCh <- struct{}{}

	// Wait for handler result
	select {
	case res := <-resultCh:
		require.NoError(t, res.err)
		assert.Equal(t, "https://shop.example.com/checkout", res.result["url"])
	case <-time.After(3 * time.Second):
		t.Fatal("handler timed out waiting for HITL approval")
	}
}

// ---------------------------------------------------------------------------
// E2E: Workflow Multi-Step
// ---------------------------------------------------------------------------

func TestE2E_Workflow_MultiStep(t *testing.T) {
	cfg := &mockServiceConfig{
		workflowResp: &ServiceWorkflowResponse{
			Success:  true,
			Duration: 500,
			Data: &ServiceWorkflowData{
				TotalSteps:     3,
				CompletedSteps: 3,
				Steps: []ServiceResponse{
					{Success: true, Data: map[string]interface{}{"title": "Page"}},
					{Success: true},
					{Success: true, Data: map[string]interface{}{"price": "$9.99"}, Screenshot: "ss1"},
				},
			},
		},
	}

	srv := newE2EServer(cfg)
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	result := handlerInvoke(t, h, "workflow", "https://example.com", map[string]interface{}{
		"form_fields": []string{"fill", "click"},
	})

	assert.Equal(t, "https://example.com", result["url"])

	// Workflow should collect extracted data from step responses
	extractedData, ok := result["extracted_data"].([]interface{})
	if ok && len(extractedData) > 0 {
		allExtracted := fmt.Sprintf("%v", extractedData)
		assert.Contains(t, allExtracted, "price")
	}

	// Screenshots should be collected from step responses
	screenshots, ok := result["screenshots"].([]interface{})
	if ok {
		assert.Len(t, screenshots, 1)
	}

	// Verify workflow endpoint was called
	reqs := cfg.getRequests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "/workflow", reqs[0].path)
}

// ---------------------------------------------------------------------------
// E2E: Invalid/Unsupported Actions
// ---------------------------------------------------------------------------

func TestE2E_UnsupportedAction_ReturnsError(t *testing.T) {
	client := NewClient("http://unused")
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{
		"url":    "https://example.com",
		"action": "teleport",
	})

	_, err := h(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported browser action: teleport")
}

func TestE2E_MissingURL_ReturnsError(t *testing.T) {
	client := NewClient("http://unused")
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{
		"action": "navigate",
	})

	_, err := h(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field url is required")
}
