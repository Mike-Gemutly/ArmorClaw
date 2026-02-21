// Package sdtw provides tests for SDTW adapters
package sdtw

import (
	"context"
	"testing"
	"time"
)

func TestBaseAdapter(t *testing.T) {
	caps := CapabilitySet{
		Read:  true,
		Write: true,
	}

	adapter := NewBaseAdapter("test", "1.0.0", caps)

	if adapter.Platform() != "test" {
		t.Errorf("expected platform 'test', got '%s'", adapter.Platform())
	}

	if adapter.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", adapter.Version())
	}

	if !adapter.Capabilities().Read {
		t.Error("expected Read capability to be true")
	}

	// Test initialization
	ctx := context.Background()
	config := AdapterConfig{
		Platform: "test",
		Enabled:  true,
	}

	if err := adapter.Initialize(ctx, config); err != nil {
		t.Errorf("initialization failed: %v", err)
	}

	// Test health check
	health, err := adapter.HealthCheck()
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}

	if !health.Connected {
		t.Error("expected connected to be true after initialization")
	}

	// Test metrics
	metrics, err := adapter.Metrics()
	if err != nil {
		t.Errorf("metrics failed: %v", err)
	}

	// Uptime should be >= 0 (not strictly greater than 0 due to timing)
	if metrics.Uptime < 0 {
		t.Errorf("expected uptime to be >= 0, got %v", metrics.Uptime)
	}
}

func TestAdapterError(t *testing.T) {
	err := NewAdapterError(ErrRateLimited, "rate limit exceeded", true)

	if err.Code != ErrRateLimited {
		t.Errorf("expected code '%s', got '%s'", ErrRateLimited, err.Code)
	}

	if !err.Retryable {
		t.Error("expected retryable to be true")
	}

	if err.Permanent {
		t.Error("expected permanent to be false")
	}

	if err.Error() == "" {
		t.Error("expected error string to not be empty")
	}
}

func TestSignMessage(t *testing.T) {
	secret := "test-secret"
	content := "test message"

	signature := SignMessage(content, secret)
	if signature == "" {
		t.Error("expected signature to not be empty")
	}

	// Test verification
	if !VerifySignature(content, signature, secret) {
		t.Error("signature verification failed")
	}

	// Test wrong signature
	if VerifySignature(content, "wrong", secret) {
		t.Error("expected verification to fail with wrong signature")
	}
}

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr bool
	}{
		{
			name: "valid message",
			msg: Message{
				ID:        "msg-123",
				Content:   "test content",
				Type:      MessageTypeText,
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			msg: Message{
				Content:   "test",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing content and attachments",
			msg: Message{
				ID:        "msg-123",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "message with attachments",
			msg: Message{
				ID:      "msg-123",
				Type:    MessageTypeImage,
				Attachments: []Attachment{
					{ID: "att-1", URL: "http://example.com/file.jpg"},
				},
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessage(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlackAdapter(t *testing.T) {
	adapter := NewSlackAdapter()

	if adapter.Platform() != "slack" {
		t.Errorf("expected platform 'slack', got '%s'", adapter.Platform())
	}

	caps := adapter.Capabilities()
	if !caps.Read || !caps.Write {
		t.Error("expected Read and Write to be true")
	}

	if !caps.Threads {
		t.Error("expected Threads to be true for Slack")
	}
}

func TestDiscordAdapter(t *testing.T) {
	adapter := NewDiscordAdapter()

	if adapter.Platform() != "discord" {
		t.Errorf("expected platform 'discord', got '%s'", adapter.Platform())
	}

	caps := adapter.Capabilities()
	if !caps.Read || !caps.Write {
		t.Error("expected Read and Write to be true")
	}

	if !caps.Edit || !caps.Delete {
		t.Error("expected Edit and Delete to be true for Discord")
	}
}

func TestTeamsAdapter(t *testing.T) {
	adapter := NewTeamsAdapter(TeamsConfig{})

	if adapter.Platform() != "teams" {
		t.Errorf("expected platform 'teams', got '%s'", adapter.Platform())
	}

	if adapter.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", adapter.Version())
	}

	// Test that operations return not implemented errors
	ctx := context.Background()
	target := Target{Channel: "test-channel"}
	msg := Message{
		ID:        "msg-123",
		Content:   "test",
		Timestamp: time.Now(),
	}

	result, err := adapter.SendMessage(ctx, target, msg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Delivered {
		t.Error("expected Delivered to be false for stub")
	}

	if result.Error == nil {
		t.Error("expected Error to be set for stub")
	}

	// Test health check
	health, err := adapter.HealthCheck()
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}

	if health.Connected {
		t.Error("expected Connected to be false for stub")
	}
}

func TestWhatsAppAdapter(t *testing.T) {
	adapter := NewWhatsAppAdapter()

	if adapter.Platform() != "whatsapp" {
		t.Errorf("expected platform 'whatsapp', got '%s'", adapter.Platform())
	}

	// Test that operations return not implemented errors
	ctx := context.Background()
	target := Target{Channel: "1234567890"}
	msg := Message{
		ID:        "msg-123",
		Content:   "test",
		Timestamp: time.Now(),
	}

	result, err := adapter.SendMessage(ctx, target, msg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Delivered {
		t.Error("expected Delivered to be false for stub")
	}

	if result.Error == nil {
		t.Error("expected Error to be set for stub")
	}
}

func TestTarget(t *testing.T) {
	target := Target{
		Platform: "slack",
		RoomID:   "!room:matrix.org",
		Channel:  "C12345678",
		Metadata: map[string]string{
			"team": "example",
		},
	}

	if target.Platform != "slack" {
		t.Errorf("expected platform 'slack', got '%s'", target.Platform)
	}

	if len(target.Metadata) == 0 {
		t.Error("expected metadata to not be empty")
	}
}

func TestAttachment(t *testing.T) {
	att := Attachment{
		ID:       "att-123",
		URL:      "http://example.com/file.pdf",
		MimeType: "application/pdf",
		Size:     1024,
		Filename: "document.pdf",
	}

	if att.ID != "att-123" {
		t.Errorf("expected ID 'att-123', got '%s'", att.ID)
	}

	if att.Size != 1024 {
		t.Errorf("expected Size 1024, got %d", att.Size)
	}
}

func TestCapabilitySetDefaults(t *testing.T) {
	// Test Slack defaults
	slackCaps := GetDefaultSlackCapabilities()
	if !slackCaps.Read || !slackCaps.Write {
		t.Error("expected Slack to have Read and Write capabilities")
	}

	// Test Discord defaults
	discordCaps := GetDefaultDiscordCapabilities()
	if !discordCaps.Edit || !discordCaps.Delete {
		t.Error("expected Discord to have Edit and Delete capabilities")
	}

	// Test Teams defaults
	teamsCaps := GetDefaultTeamsCapabilities()
	if !teamsCaps.ReadReceipts {
		t.Error("expected Teams to have ReadReceipts capability")
	}

	// Test WhatsApp defaults
	whatsappCaps := GetDefaultWhatsAppCapabilities()
	if whatsappCaps.Edit {
		t.Error("expected WhatsApp to not have Edit capability")
	}

	if !whatsappCaps.ReadReceipts {
		t.Error("expected WhatsApp to have ReadReceipts capability")
	}
}

func TestErrorCode(t *testing.T) {
	tests := []struct {
		code   ErrorCode
		expect string
	}{
		{ErrRateLimited, "rate_limited"},
		{ErrAuthFailed, "auth_failed"},
		{ErrInvalidTarget, "invalid_target"},
		{ErrNetworkError, "network_error"},
		{ErrTimeout, "timeout"},
		{ErrCircuitOpen, "circuit_open"},
		{ErrValidation, "validation_error"},
		{ErrPlatformError, "platform_error"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if string(tt.code) != tt.expect {
				t.Errorf("expected '%s', got '%s'", tt.expect, tt.code)
			}
		})
	}
}

func TestExternalEvent(t *testing.T) {
	event := ExternalEvent{
		Platform:  "slack",
		EventType: "message",
		Timestamp: time.Now(),
		Source:    "C12345678",
		Content:   "test message",
		Metadata: map[string]string{
			"user": "U12345678",
		},
	}

	if event.Platform != "slack" {
		t.Errorf("expected platform 'slack', got '%s'", event.Platform)
	}

	if len(event.Metadata) == 0 {
		t.Error("expected metadata to not be empty")
	}
}

func TestSendResult(t *testing.T) {
	result := SendResult{
		MessageID: "msg-456",
		Delivered: true,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"channel": "C12345678",
		},
	}

	if !result.Delivered {
		t.Error("expected Delivered to be true")
	}

	if result.MessageID != "msg-456" {
		t.Errorf("expected MessageID 'msg-456', got '%s'", result.MessageID)
	}
}

func TestAdapterConfig(t *testing.T) {
	config := AdapterConfig{
		Platform: "slack",
		Enabled:  true,
		Credentials: map[string]string{
			"bot_token": "xoxb-test-token",
		},
		Settings: map[string]string{
			"team_id": "T12345678",
		},
		RateLimits: RateLimitConfig{
			RequestsPerSecond: 10,
			BurstSize:         50,
		},
	}

	if config.Platform != "slack" {
		t.Errorf("expected platform 'slack', got '%s'", config.Platform)
	}

	if config.Credentials["bot_token"] != "xoxb-test-token" {
		t.Error("expected bot_token to be set")
	}

	if config.RateLimits.RequestsPerSecond != 10 {
		t.Errorf("expected RequestsPerSecond 10, got %d", config.RateLimits.RequestsPerSecond)
	}
}

func TestHealthStatus(t *testing.T) {
	status := HealthStatus{
		Connected:    true,
		LastPing:     time.Now(),
		LastMessage:  time.Now(),
		ErrorRate:    0.5,
		Latency:      100 * time.Millisecond,
		QueueDepth:   10,
	}

	if !status.Connected {
		t.Error("expected Connected to be true")
	}

	if status.ErrorRate != 0.5 {
		t.Errorf("expected ErrorRate 0.5, got %f", status.ErrorRate)
	}

	if status.QueueDepth != 10 {
		t.Errorf("expected QueueDepth 10, got %d", status.QueueDepth)
	}
}

func TestAdapterMetrics(t *testing.T) {
	metrics := AdapterMetrics{
		MessagesSent:     100,
		MessagesReceived: 200,
		MessagesFailed:   5,
		LastError:        "connection timeout",
		LastErrorTime:    time.Now(),
	}

	if metrics.MessagesSent != 100 {
		t.Errorf("expected MessagesSent 100, got %d", metrics.MessagesSent)
	}

	if metrics.MessagesFailed != 5 {
		t.Errorf("expected MessagesFailed 5, got %d", metrics.MessagesFailed)
	}
}
