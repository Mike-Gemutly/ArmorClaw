// Package sidecar provides PII interception tests
package sidecar

import (
	"context"
	"testing"
)

func TestPIIInterceptor_NewPIIInterceptor(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	interceptor := NewPIIInterceptor(config)

	if interceptor == nil {
		t.Fatal("Expected interceptor to be created")
	}

	if interceptor.config != config {
		t.Error("Expected interceptor config to match provided config")
	}
}

func TestPIIInterceptor_NewPIIInterceptor_NilConfig(t *testing.T) {
	interceptor := NewPIIInterceptor(nil)

	if interceptor == nil {
		t.Fatal("Expected interceptor to be created with default config")
	}

	if interceptor.config == nil {
		t.Error("Expected default config to be set")
	}

	if !interceptor.isEnabled() {
		t.Error("Expected PII interception to be enabled by default")
	}
}

func TestPIIInterceptor_InterceptRequest_Disabled(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Enabled = false
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Content: []byte("My email is test@example.com"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if intercepted != req {
		t.Error("Expected request to be unchanged when PII interception is disabled")
	}
}

func TestPIIInterceptor_InterceptRequest_LogOnly(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.LogOnly = true
	interceptor := NewPIIInterceptor(config)

	req := &ExtractTextRequest{
		DocumentContent: []byte("Call me at 555-123-4567"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "ExtractText", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if intercepted != req {
		t.Error("Expected request to be unchanged in log-only mode")
	}
}

func TestPIIInterceptor_InterceptRequest_RedactEmail(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/file.txt",
		ContentType:    "text/plain",
		Content:        []byte("Contact: john.doe@example.com"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*UploadBlobRequest)
	contentStr := string(interceptedReq.Content)

	if contentStr == "Contact: john.doe@example.com" {
		t.Error("Expected email to be redacted")
	}

	if contentStr != "Contact: [REDACTED_EMAIL]" {
		t.Errorf("Expected redacted email, got: %s", contentStr)
	}
}

func TestPIIInterceptor_InterceptRequest_RedactPhone(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/file.txt",
		ContentType:    "text/plain",
		Content:        []byte("Phone: 555-123-4567"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*UploadBlobRequest)
	contentStr := string(interceptedReq.Content)

	if contentStr != "Phone: [REDACTED_PHONE]" {
		t.Errorf("Expected redacted phone, got: %s", contentStr)
	}
}

func TestPIIInterceptor_InterceptRequest_RejectAction(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionReject
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/file.txt",
		ContentType:    "text/plain",
		Content:        []byte("Email: test@example.com"),
	}

	_, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err == nil {
		t.Error("Expected error when PII is detected and action is reject")
	}

	if err == nil || err.Error() != "PII detected in UploadBlob request, rejecting: 1 PII instances found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestPIIInterceptor_InterceptRequest_NoPII(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/file.txt",
		ContentType:    "text/plain",
		Content:        []byte("This is just a normal document"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*UploadBlobRequest)
	contentStr := string(interceptedReq.Content)

	if contentStr != "This is just a normal document" {
		t.Errorf("Expected content to be unchanged, got: %s", contentStr)
	}
}

func TestPIIInterceptor_InterceptRequest_MultiplePII(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/file.txt",
		ContentType:    "text/plain",
		Content:        []byte("Email: test@example.com, Phone: 555-123-4567"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*UploadBlobRequest)
	contentStr := string(interceptedReq.Content)

	if contentStr != "Email: [REDACTED_EMAIL], Phone: [REDACTED_PHONE]" {
		t.Errorf("Expected both PII types to be redacted, got: %s", contentStr)
	}
}

func TestPIIInterceptor_InterceptRequest_CreditCard(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &ExtractTextRequest{
		DocumentFormat:  "txt",
		DocumentContent: []byte("Card: 4111-1111-1111-1111"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "ExtractText", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*ExtractTextRequest)
	contentStr := string(interceptedReq.DocumentContent)

	if contentStr != "Card: [REDACTED_CREDIT_CARD]" {
		t.Errorf("Expected redacted credit card, got: %s", contentStr)
	}
}

func TestPIIInterceptor_InterceptRequest_SSN(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &ProcessDocumentRequest{
		InputFormat:  "txt",
		InputContent: []byte("SSN: 123-45-6789"),
		Operation:    "process",
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "ProcessDocument", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*ProcessDocumentRequest)
	contentStr := string(interceptedReq.InputContent)

	if contentStr != "SSN: [REDACTED_SSN]" {
		t.Errorf("Expected redacted SSN, got: %s", contentStr)
	}
}

func TestPIIInterceptor_InterceptRequest_IPAddress(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/file.txt",
		ContentType:    "text/plain",
		Content:        []byte("Server IP: 192.168.1.1"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*UploadBlobRequest)
	contentStr := string(interceptedReq.Content)

	if contentStr != "Server IP: [REDACTED_IP]" {
		t.Errorf("Expected redacted IP address, got: %s", contentStr)
	}
}

func TestPIIInterceptor_InterceptRequest_APIToken(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Action = ActionRedact
	interceptor := NewPIIInterceptor(config)

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/file.txt",
		ContentType:    "text/plain",
		Content:        []byte("Token: sk_test_1234567890abcdefghijklmnopqrstuvwxyz"),
	}

	intercepted, err := interceptor.InterceptRequest(context.Background(), "UploadBlob", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	interceptedReq := intercepted.(*UploadBlobRequest)
	contentStr := string(interceptedReq.Content)

	if contentStr == "Token: sk_test_1234567890abcdefghijklmnopqrstuvwxyz" {
		t.Error("Expected API token to be redacted")
	}

	if contentStr != "Token: [REDACTED_API_KEY]" {
		t.Logf("Content: %s", contentStr)
	}
}

func TestPIIInterceptor_SetEnabled(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.Enabled = true
	interceptor := NewPIIInterceptor(config)

	if !interceptor.isEnabled() {
		t.Error("Expected PII interception to be enabled")
	}

	interceptor.SetEnabled(false)

	if interceptor.isEnabled() {
		t.Error("Expected PII interception to be disabled after SetEnabled(false)")
	}
}

func TestPIIInterceptor_SetAction(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	interceptor := NewPIIInterceptor(config)

	interceptor.SetAction(ActionReject)

	if interceptor.config.Action != ActionReject {
		t.Errorf("Expected action to be Reject, got: %s", interceptor.config.Action)
	}

	interceptor.SetAction(ActionRedact)

	if interceptor.config.Action != ActionRedact {
		t.Errorf("Expected action to be Redact, got: %s", interceptor.config.Action)
	}
}

func TestPIIInterceptor_SetLogOnly(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	config.LogOnly = false
	interceptor := NewPIIInterceptor(config)

	if interceptor.config.LogOnly {
		t.Error("Expected log-only to be false")
	}

	interceptor.SetLogOnly(true)

	if !interceptor.config.LogOnly {
		t.Error("Expected log-only to be true after SetLogOnly(true)")
	}
}

func TestPIIInterceptor_GetStatistics(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	interceptor := NewPIIInterceptor(config)

	stats := interceptor.GetStatistics()

	if stats["enabled"] != true {
		t.Error("Expected enabled to be true in statistics")
	}

	if stats["action"] != ActionRedact {
		t.Errorf("Expected action to be Redact, got: %v", stats["action"])
	}

	if stats["log_only"] != false {
		t.Error("Expected log_only to be false in statistics")
	}

	if stats["pattern_count"] == nil {
		t.Error("Expected pattern_count to be set in statistics")
	}
}

func TestPIIInterceptor_ScrubRequestText(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	interceptor := NewPIIInterceptor(config)

	text := "Contact test@example.com at 555-123-4567"

	scrubbed, redactions := interceptor.ScrubRequestText(text)

	if scrubbed == text {
		t.Error("Expected text to be scrubbed")
	}

	if len(redactions) == 0 {
		t.Error("Expected redactions to be detected")
	}
}

func TestPIIInterceptor_ContainsPII(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	interceptor := NewPIIInterceptor(config)

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Has Email", "test@example.com", true},
		{"Has Phone", "555-123-4567", true},
		{"Has SSN", "123-45-6789", true},
		{"No PII", "Hello world", false},
		{"Has Credit Card", "4111111111111111", true},
		{"Has IP", "192.168.1.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interceptor.ContainsPII(tt.text)
			if result != tt.expected {
				t.Errorf("ContainsPII(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestPIIInterceptor_UpdateConfig(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	interceptor := NewPIIInterceptor(config)

	newConfig := DefaultPIIInterceptorConfig()
	newConfig.Enabled = false
	newConfig.Action = ActionReject

	interceptor.UpdateConfig(newConfig)

	if interceptor.config.Enabled {
		t.Error("Expected enabled to be false after update")
	}

	if interceptor.config.Action != ActionReject {
		t.Errorf("Expected action to be Reject, got: %s", interceptor.config.Action)
	}
}

func TestPIIInterceptor_isLikelyText(t *testing.T) {
	config := DefaultPIIInterceptorConfig()
	interceptor := NewPIIInterceptor(config)

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Plain text", "Hello world", true},
		{"Email", "test@example.com", true},
		{"Binary-like", "\x00\x01\x02\x03", false},
		{"Mixed text - mostly binary", "\x00\x01\x02\x03\x04\x05\x06\x07H", false},
		{"Mixed text - mostly text", "Hello world\x00", true},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interceptor.isLikelyText(tt.text)
			if result != tt.expected {
				t.Errorf("isLikelyText(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}
