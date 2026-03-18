// Package sdtw provides tests for WhatsApp adapter
package sdtw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWhatsAppAdapter_New(t *testing.T) {
	adapter := NewWhatsAppAdapter()

	if adapter.Platform() != "whatsapp" {
		t.Errorf("expected platform 'whatsapp', got '%s'", adapter.Platform())
	}

	if adapter.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", adapter.Version())
	}

	caps := adapter.Capabilities()
	if !caps.Read || !caps.Write {
		t.Error("expected Read and Write to be true")
	}

	if caps.Reactions {
		t.Error("expected Reactions to be false for WhatsApp")
	}

	if caps.Edit {
		t.Error("expected Edit to be false for WhatsApp")
	}

	if !caps.Delete {
		t.Error("expected Delete to be true for WhatsApp")
	}

	if !caps.ReadReceipts {
		t.Error("expected ReadReceipts to be true for WhatsApp")
	}
}

func TestWhatsAppAdapter_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		config  AdapterConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AdapterConfig{
				Enabled: true,
				Credentials: map[string]string{
					"access_token": "test_token_123",
				},
				Settings: map[string]string{
					"phone_number_id": "123456789",
				},
			},
			wantErr: false,
		},
		{
			name: "missing access token",
			config: AdapterConfig{
				Enabled:     true,
				Credentials: map[string]string{},
				Settings: map[string]string{
					"phone_number_id": "123456789",
				},
			},
			wantErr: true,
		},
		{
			name: "missing phone number id",
			config: AdapterConfig{
				Enabled: true,
				Credentials: map[string]string{
					"access_token": "test_token_123",
				},
				Settings: map[string]string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewWhatsAppAdapter()
			ctx := context.Background()
			err := adapter.Initialize(ctx, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWhatsAppAdapter_Start(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*WhatsAppAdapter)
		wantErr bool
	}{
		{
			name: "start initialized adapter",
			setup: func(w *WhatsAppAdapter) {
				ctx := context.Background()
				config := AdapterConfig{
					Enabled: true,
					Credentials: map[string]string{
						"access_token": "test_token",
					},
					Settings: map[string]string{
						"phone_number_id": "123456789",
					},
				}
				w.Initialize(ctx, config)
			},
			wantErr: true, // Will fail connection test without mock server
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewWhatsAppAdapter()
			tt.setup(adapter)

			ctx := context.Background()
			err := adapter.Start(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWhatsAppAdapter_SendMessage_Text(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("expected Bearer token in Authorization header")
		}

		if !strings.Contains(r.URL.Path, "/123456789/messages") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Parse request body
		var reqBody WhatsAppMessage
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		if reqBody.MessagingProduct != "whatsapp" {
			t.Errorf("expected messaging_product 'whatsapp', got '%s'", reqBody.MessagingProduct)
		}

		if reqBody.Type != "text" {
			t.Errorf("expected type 'text', got '%s'", reqBody.Type)
		}

		// Send success response
		resp := WhatsAppAPIResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID string `json:"id"`
			}{{ID: "wamid.HbgLMHztRdF"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Initialize adapter with mock server URL
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token_123",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	if err := adapter.Initialize(ctx, config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Replace client with mock server client
	adapter.client = server.Client()

	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Send message
	target := Target{
		Platform: "whatsapp",
		Channel:  "1234567890", // Phone number
	}

	msg := Message{
		ID:        "msg-123",
		Content:   "Hello, World!",
		Type:      MessageTypeText,
		Timestamp: time.Now(),
	}

	result, err := adapter.SendMessage(ctx, target, msg)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if !result.Delivered {
		t.Error("expected Delivered to be true")
	}

	if result.MessageID == "" {
		t.Error("expected MessageID to be set")
	}

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
}

func TestWhatsAppAdapter_SendMessage_Template(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var reqBody WhatsAppMessage
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody.Type != "template" {
			t.Errorf("expected type 'template', got '%s'", reqBody.Type)
		}

		if reqBody.Template == nil {
			t.Error("expected Template to be set")
		}

		if reqBody.Template.Name != "hello_world" {
			t.Errorf("expected template name 'hello_world', got '%s'", reqBody.Template.Name)
		}

		// Send success response
		resp := WhatsAppAPIResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID string `json:"id"`
			}{{ID: "wamid.TemplateMsg123"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Initialize adapter
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.client = server.Client()
	adapter.Start(ctx)

	// Send template message
	target := Target{
		Platform: "whatsapp",
		Channel:  "1234567890",
	}

	msg := Message{
		ID:        "msg-124",
		Content:   "",
		Type:      MessageTypeText,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"template_name":     "hello_world",
			"template_language": "en_US",
		},
	}

	result, err := adapter.SendMessage(ctx, target, msg)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if !result.Delivered {
		t.Error("expected Delivered to be true")
	}
}

func TestWhatsAppAdapter_SendMessage_Image(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody WhatsAppMessage
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody.Type != "image" {
			t.Errorf("expected type 'image', got '%s'", reqBody.Type)
		}

		if reqBody.Image == nil {
			t.Error("expected Image to be set")
		}

		// Send success response
		resp := WhatsAppAPIResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID string `json:"id"`
			}{{ID: "wamid.ImageMsg123"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Initialize adapter
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.client = server.Client()
	adapter.Start(ctx)

	// Send image message
	target := Target{
		Platform: "whatsapp",
		Channel:  "1234567890",
	}

	msg := Message{
		ID:        "msg-125",
		Content:   "Check this image!",
		Type:      MessageTypeImage,
		Timestamp: time.Now(),
		Attachments: []Attachment{
			{
				ID:       "media_123",
				URL:      "https://example.com/image.jpg",
				MimeType: "image/jpeg",
			},
		},
	}

	result, err := adapter.SendMessage(ctx, target, msg)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if !result.Delivered {
		t.Error("expected Delivered to be true")
	}
}

func TestWhatsAppAdapter_ReceiveEvent(t *testing.T) {
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token":   "test_token",
			"webhook_secret": "webhook_secret_123",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.Start(ctx)

	// Test message event
	webhookPayload := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "123456789",
			"changes": [{
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {
						"display_phone_number": "+1234567890",
						"phone_number_id": "123456789"
					},
					"messages": [{
						"from": "1234567890",
						"id": "wamid.Incoming123",
						"timestamp": "1710000000",
						"type": "text",
						"text": {
							"body": "Hello from user"
						}
					}]
				},
				"field": "messages"
			}]
		}]
	}`

	event := ExternalEvent{
		Platform:  "whatsapp",
		EventType: "message",
		Timestamp: time.Now(),
		Source:    "1234567890",
		Content:   webhookPayload,
		Signature: SignMessage(webhookPayload, "webhook_secret_123"),
	}

	err := adapter.ReceiveEvent(event)
	if err != nil {
		t.Errorf("ReceiveEvent() error = %v", err)
	}

	// Verify event was updated
	if event.Content != "Hello from user" {
		t.Errorf("expected content 'Hello from user', got '%s'", event.Content)
	}

	if event.EventType != "text" {
		t.Errorf("expected event type 'text', got '%s'", event.EventType)
	}

	if event.Metadata["message_id"] != "wamid.Incoming123" {
		t.Errorf("expected message_id 'wamid.Incoming123', got '%s'", event.Metadata["message_id"])
	}
}

func TestWhatsAppAdapter_ReceiveEvent_InvalidSignature(t *testing.T) {
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token":   "test_token",
			"webhook_secret": "webhook_secret_123",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.Start(ctx)

	// Test invalid signature
	webhookPayload := `{"object": "whatsapp_business_account"}`
	event := ExternalEvent{
		Platform:  "whatsapp",
		Content:   webhookPayload,
		Signature: "invalid_signature",
	}

	err := adapter.ReceiveEvent(event)
	if err == nil {
		t.Error("expected error for invalid signature")
	}

	if adapterErr, ok := err.(*AdapterError); ok {
		if adapterErr.Code != ErrAuthFailed {
			t.Errorf("expected ErrAuthFailed, got %v", adapterErr.Code)
		}
	} else {
		t.Errorf("expected AdapterError type, got %T", err)
	}
}

func TestWhatsAppAdapter_EditMessage(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// Parse request body
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["messaging_product"] != "whatsapp" {
			t.Errorf("expected messaging_product 'whatsapp'")
		}

		if reqBody["status"] != "read" {
			t.Errorf("expected status 'read'")
		}

		if reqBody["message_id"] != "wamid.Msg123" {
			t.Errorf("expected message_id 'wamid.Msg123'")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Initialize adapter
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.client = server.Client()
	adapter.Start(ctx)

	// Mark message as read
	target := Target{Platform: "whatsapp"}
	err := adapter.EditMessage(ctx, target, "wamid.Msg123", "")

	if err != nil {
		t.Errorf("EditMessage() error = %v", err)
	}
}

func TestWhatsAppAdapter_DeleteMessage(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/wamid.MsgToDelete") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("expected Bearer token in Authorization header")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Initialize adapter
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.client = server.Client()
	adapter.Start(ctx)

	// Delete message
	target := Target{Platform: "whatsapp"}
	err := adapter.DeleteMessage(ctx, target, "wamid.MsgToDelete")

	if err != nil {
		t.Errorf("DeleteMessage() error = %v", err)
	}
}

func TestWhatsAppAdapter_DeleteMessage_NotFound(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Initialize adapter
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.client = server.Client()
	adapter.Start(ctx)

	// Try to delete non-existent message
	target := Target{Platform: "whatsapp"}
	err := adapter.DeleteMessage(ctx, target, "wamid.NotFound")

	if err == nil {
		t.Error("expected error for not found message")
	}

	if adapterErr, ok := err.(*AdapterError); ok {
		if adapterErr.Code != ErrInvalidTarget {
			t.Errorf("expected ErrInvalidTarget, got %v", adapterErr.Code)
		}
	} else {
		t.Errorf("expected AdapterError type, got %T", err)
	}
}

func TestWhatsAppAdapter_DeleteMessage_Forbidden(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	// Initialize adapter
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.client = server.Client()
	adapter.Start(ctx)

	// Try to delete expired message
	target := Target{Platform: "whatsapp"}
	err := adapter.DeleteMessage(ctx, target, "wamid.Expired")

	if err == nil {
		t.Error("expected error for expired message")
	}

	if adapterErr, ok := err.(*AdapterError); ok {
		if adapterErr.Code != ErrAuthFailed {
			t.Errorf("expected ErrAuthFailed, got %v", adapterErr.Code)
		}
	} else {
		t.Errorf("expected AdapterError type, got %T", err)
	}
}

func TestWhatsAppAdapter_HealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*WhatsAppAdapter)
		wantStatus HealthStatus
	}{
		{
			name: "running adapter",
			setup: func(w *WhatsAppAdapter) {
				ctx := context.Background()
				config := AdapterConfig{
					Enabled: true,
					Credentials: map[string]string{
						"access_token": "test_token",
					},
					Settings: map[string]string{
						"phone_number_id": "123456789",
					},
				}
				w.Initialize(ctx, config)
				w.running = true
			},
			wantStatus: HealthStatus{
				Connected:  true,
				ErrorRate:  0.0,
				QueueDepth: 0,
			},
		},
		{
			name: "not running adapter",
			setup: func(w *WhatsAppAdapter) {
				w.running = false
			},
			wantStatus: HealthStatus{
				Connected: false,
				Error:     "adapter not running",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewWhatsAppAdapter()
			tt.setup(adapter)

			status, err := adapter.HealthCheck()
			if err != nil {
				t.Errorf("HealthCheck() error = %v", err)
			}

			if status.Connected != tt.wantStatus.Connected {
				t.Errorf("expected Connected %v, got %v", tt.wantStatus.Connected, status.Connected)
			}

			if tt.wantStatus.Error != "" && status.Error != tt.wantStatus.Error {
				t.Errorf("expected Error '%s', got '%s'", tt.wantStatus.Error, status.Error)
			}
		})
	}
}

func TestWhatsAppAdapter_Metrics(t *testing.T) {
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)

	metrics, err := adapter.Metrics()
	if err != nil {
		t.Errorf("Metrics() error = %v", err)
	}

	if metrics.Uptime <= 0 {
		t.Error("expected Uptime to be greater than 0")
	}
}

func TestWhatsAppAdapter_SendTemplateMessage(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody WhatsAppMessage
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody.Type != "template" {
			t.Errorf("expected type 'template', got '%s'", reqBody.Type)
		}

		resp := WhatsAppAPIResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID string `json:"id"`
			}{{ID: "wamid.TemplateTest123"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Initialize adapter
	adapter := NewWhatsAppAdapter()
	ctx := context.Background()
	config := AdapterConfig{
		Enabled: true,
		Credentials: map[string]string{
			"access_token": "test_token",
		},
		Settings: map[string]string{
			"phone_number_id": "123456789",
		},
	}

	adapter.Initialize(ctx, config)
	adapter.client = server.Client()
	adapter.Start(ctx)

	// Send template message
	components := []WhatsAppTemplateComponent{
		{
			Type: "body",
			Parameters: []WhatsAppParameter{
				{Type: "text", Text: "John Doe"},
			},
		},
	}

	result, err := adapter.SendTemplateMessage(ctx, "1234567890", "hello_world", "en_US", components)
	if err != nil {
		t.Fatalf("SendTemplateMessage() error = %v", err)
	}

	if !result.Delivered {
		t.Error("expected Delivered to be true")
	}

	if result.MessageID != "wamid.TemplateTest123" {
		t.Errorf("expected MessageID 'wamid.TemplateTest123', got '%s'", result.MessageID)
	}
}

func TestWhatsAppAdapter_VerifyWebhook(t *testing.T) {
	adapter := NewWhatsAppAdapter()

	payload := []byte(`{"test": "payload"}`)
	secret := "test_secret"
	signature := SignMessage(string(payload), secret)

	if !adapter.VerifyWebhook(payload, signature, secret) {
		t.Error("expected webhook verification to succeed")
	}

	if adapter.VerifyWebhook(payload, "invalid_signature", secret) {
		t.Error("expected webhook verification to fail with invalid signature")
	}
}

func TestRateLimiter(t *testing.T) {
	ctx := context.Background()
	limiter := newRateLimiter(5, 100*time.Millisecond)

	// First 5 requests should succeed immediately
	for i := 0; i < 5; i++ {
		start := time.Now()
		err := limiter.Wait(ctx)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("request %d failed: %v", i, err)
		}

		if elapsed > 10*time.Millisecond {
			t.Errorf("request %d took too long: %v", i, elapsed)
		}
	}

	// 6th request should wait for refill
	start := time.Now()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("6th request failed: %v", err)
	}

	if elapsed < 90*time.Millisecond {
		t.Errorf("6th request should have waited for refill, but took %v", elapsed)
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	limiter := newRateLimiter(1, 1*time.Second)

	// Consume all tokens
	limiter.Wait(ctx)

	// Cancel context
	cancel()

	// Should return context canceled error
	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("expected error after context cancellation")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestMapWhatsAppError(t *testing.T) {
	tests := []struct {
		errorCode   int
		expectError ErrorCode
	}{
		{131047, ErrAuthFailed},
		{131021, ErrAuthFailed},
		{131026, ErrAuthFailed},
		{131052, ErrRateLimited},
		{131014, ErrInvalidTarget},
		{131015, ErrInvalidTarget},
		{131001, ErrNetworkError},
		{999, ErrPlatformError},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.errorCode), func(t *testing.T) {
			result := mapWhatsAppError(tt.errorCode)
			if result != tt.expectError {
				t.Errorf("expected %v, got %v", tt.expectError, result)
			}
		})
	}
}

func TestIsRetryableWhatsAppError(t *testing.T) {
	tests := []struct {
		errorCode int
		retryable bool
	}{
		{131052, true},  // Rate limit
		{131001, true},  // Network error
		{131047, false}, // Auth failed
		{999, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.errorCode), func(t *testing.T) {
			result := isRetryableWhatsAppError(tt.errorCode)
			if result != tt.retryable {
				t.Errorf("expected %v, got %v", tt.retryable, result)
			}
		})
	}
}

func TestGetDefaultWhatsAppCapabilities(t *testing.T) {
	caps := GetDefaultWhatsAppCapabilities()

	if !caps.Read || !caps.Write {
		t.Error("expected Read and Write to be true")
	}

	if !caps.Media {
		t.Error("expected Media to be true")
	}

	if caps.Reactions {
		t.Error("expected Reactions to be false")
	}

	if caps.Threads {
		t.Error("expected Threads to be false")
	}

	if caps.Edit {
		t.Error("expected Edit to be false")
	}

	if !caps.Delete {
		t.Error("expected Delete to be true")
	}

	if caps.Typing {
		t.Error("expected Typing to be false")
	}

	if !caps.ReadReceipts {
		t.Error("expected ReadReceipts to be true")
	}
}
