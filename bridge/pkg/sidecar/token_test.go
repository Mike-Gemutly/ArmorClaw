package sidecar

import (
	"fmt"
	"testing"
	"time"
)

func TestNewTokenGenerator(t *testing.T) {
	tests := []struct {
		name        string
		secret      []byte
		expectError bool
	}{
		{
			name:        "valid secret",
			secret:      []byte("test-secret-key-32-bytes-long!"),
			expectError: false,
		},
		{
			name:        "empty secret",
			secret:      []byte(""),
			expectError: true,
		},
		{
			name:        "nil secret",
			secret:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTokenGenerator(tt.secret)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!")
	tg, err := NewTokenGenerator(secret)
	if err != nil {
		t.Fatalf("failed to create token generator: %v", err)
	}

	tests := []struct {
		name        string
		requestID   string
		operation   string
		expectError bool
	}{
		{
			name:        "valid request",
			requestID:   "test-request-id",
			operation:   "test-operation",
			expectError: false,
		},
		{
			name:        "empty request ID",
			requestID:   "",
			operation:   "test-operation",
			expectError: true,
		},
		{
			name:        "empty operation",
			requestID:   "test-request-id",
			operation:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, timestamp, err := tg.GenerateToken(tt.requestID, tt.operation)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if token == "" {
				t.Errorf("expected non-empty token")
			}

			if timestamp == 0 {
				t.Errorf("expected non-zero timestamp")
			}

			// Verify token format
			info, err := ParseToken(token)
			if err != nil {
				t.Errorf("failed to parse generated token: %v", err)
			}

			if info.RequestID != tt.requestID {
				t.Errorf("request_id mismatch: got %s, want %s", info.RequestID, tt.requestID)
			}

			if info.Timestamp != timestamp {
				t.Errorf("timestamp mismatch: got %d, want %d", info.Timestamp, timestamp)
			}

			if info.Operation != tt.operation {
				t.Errorf("operation mismatch: got %s, want %s", info.Operation, tt.operation)
			}

			if info.Signature == "" {
				t.Errorf("expected non-empty signature")
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
		requestID   string
		timestamp   int64
		operation   string
		signature   string
	}{
		{
			name:        "valid token",
			token:       "test-request-id:1234567890:test-operation:abcdef123456",
			expectError: false,
			requestID:   "test-request-id",
			timestamp:   1234567890,
			operation:   "test-operation",
			signature:   "abcdef123456",
		},
		{
			name:        "invalid format - too few parts",
			token:       "test-request-id:1234567890",
			expectError: true,
		},
		{
			name:        "invalid format - too many parts",
			token:       "a:b:c:d:e",
			expectError: true,
		},
		{
			name:        "invalid timestamp",
			token:       "test-request-id:not-a-number:test-operation:abcdef",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseToken(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if info.RequestID != tt.requestID {
				t.Errorf("request_id mismatch: got %s, want %s", info.RequestID, tt.requestID)
			}

			if info.Timestamp != tt.timestamp {
				t.Errorf("timestamp mismatch: got %d, want %d", info.Timestamp, tt.timestamp)
			}

			if info.Operation != tt.operation {
				t.Errorf("operation mismatch: got %s, want %s", info.Operation, tt.operation)
			}

			if info.Signature != tt.signature {
				t.Errorf("signature mismatch: got %s, want %s", info.Signature, tt.signature)
			}

			expectedExpiration := tt.timestamp + int64(TokenTTL.Seconds())
			if info.Expiration != expectedExpiration {
				t.Errorf("expiration mismatch: got %d, want %d", info.Expiration, expectedExpiration)
			}
		})
	}
}

func TestValidateTokenSignature(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!")
	tg, err := NewTokenGenerator(secret)
	if err != nil {
		t.Fatalf("failed to create token generator: %v", err)
	}

	// Generate a valid token
	validToken, _, err := tg.GenerateToken("test-request-id", "test-operation")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name        string
		token       string
		expectValid bool
		expectError bool
	}{
		{
			name:        "valid token signature",
			token:       validToken,
			expectValid: true,
			expectError: false,
		},
		{
			name:        "invalid signature",
			token:       "test-request-id:1234567890:test-operation:wrong-signature",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "invalid token format",
			token:       "invalid-token",
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := tg.ValidateTokenSignature(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if valid != tt.expectValid {
				t.Errorf("validity mismatch: got %v, want %v", valid, tt.expectValid)
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name     string
		info     *TokenInfo
		expected bool
	}{
		{
			name: "expired token",
			info: &TokenInfo{
				RequestID:  "test",
				Timestamp:  now - int64(TokenTTL.Seconds()) - 100,
				Operation:  "test",
				Signature:  "test",
				Expiration: now - 100,
			},
			expected: true,
		},
		{
			name: "valid token",
			info: &TokenInfo{
				RequestID:  "test",
				Timestamp:  now,
				Operation:  "test",
				Signature:  "test",
				Expiration: now + int64(TokenTTL.Seconds()),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expired := IsTokenExpired(tt.info)
			if expired != tt.expected {
				t.Errorf("expired mismatch: got %v, want %v", expired, tt.expected)
			}
		})
	}
}

func TestIsTokenTooOld(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name     string
		info     *TokenInfo
		expected bool
	}{
		{
			name: "token too old",
			info: &TokenInfo{
				RequestID:  "test",
				Timestamp:  now - int64(MaxTimestampAge.Seconds()) - 100,
				Operation:  "test",
				Signature:  "test",
				Expiration: now + 1000,
			},
			expected: true,
		},
		{
			name: "token not too old",
			info: &TokenInfo{
				RequestID:  "test",
				Timestamp:  now,
				Operation:  "test",
				Signature:  "test",
				Expiration: now + 1000,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tooOld := IsTokenTooOld(tt.info)
			if tooOld != tt.expected {
				t.Errorf("tooOld mismatch: got %v, want %v", tooOld, tt.expected)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!")
	tg, err := NewTokenGenerator(secret)
	if err != nil {
		t.Fatalf("failed to create token generator: %v", err)
	}

	// Generate a valid token
	validToken, _, err := tg.GenerateToken("test-request-id", "test-operation")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Parse the valid token to get its info
	info, err := ParseToken(validToken)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	// Create a token with old timestamp (older than MaxTimestampAge)
	oldTimestamp := time.Now().Unix() - int64(MaxTimestampAge.Seconds()) - 100
	oldToken, _, err := tg.GenerateTokenWithTimestamp("test-request-id", "test-operation", oldTimestamp)
	if err != nil {
		t.Fatalf("failed to generate old token: %v", err)
	}

	// Create an expired token (also too old since TTL > MaxTimestampAge)
	expiredTimestamp := time.Now().Unix() - int64(TokenTTL.Seconds()) - 100
	expiredToken, _, err := tg.GenerateTokenWithTimestamp("test-request-id", "test-operation", expiredTimestamp)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	// Create a token with invalid signature (use current timestamp to avoid "too old" check)
	currentTimestamp := time.Now().Unix()
	invalidToken := info.RequestID + ":" + fmt.Sprintf("%d", currentTimestamp) + ":" + info.Operation + ":invalid-signature"

	tests := []struct {
		name          string
		token         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid token",
			token:       validToken,
			expectError: false,
		},
		{
			name:          "token too old",
			token:         oldToken,
			expectError:   true,
			errorContains: "too old",
		},
		{
			name:          "expired token",
			token:         expiredToken,
			expectError:   true,
			errorContains: "too old",
		},
		{
			name:          "invalid signature",
			token:         invalidToken,
			expectError:   true,
			errorContains: "signature",
		},
		{
			name:          "invalid format",
			token:         "invalid",
			expectError:   true,
			errorContains: "format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tg.ValidateToken(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorContains != "" && err != nil {
					if !containsString(err.Error(), tt.errorContains) {
						t.Errorf("error does not contain expected substring: got %s, want %s", err.Error(), tt.errorContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	id, err := GenerateRequestID()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if id == "" {
		t.Errorf("expected non-empty request ID")
	}

	if len(id) != 32 { // 16 bytes = 32 hex characters
		t.Errorf("request ID length mismatch: got %d, want 32", len(id))
	}

	// Generate another ID to ensure uniqueness
	id2, err := GenerateRequestID()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if id == id2 {
		t.Errorf("request IDs should be unique")
	}
}

func TestGenerateSharedSecret(t *testing.T) {
	secret, err := GenerateSharedSecret()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(secret) != 32 { // 256 bits
		t.Errorf("shared secret length mismatch: got %d, want 32", len(secret))
	}

	// Generate another secret to ensure uniqueness
	secret2, err := GenerateSharedSecret()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// With 256 bits, probability of collision is negligible
	// But we can at least check they're not identical in memory
	for i := range secret {
		if secret[i] != secret2[i] {
			return // Different bytes found, secrets are unique
		}
	}
	t.Errorf("shared secrets should be unique (extremely unlikely to collide)")
}

func TestEncodeDecodeSharedSecret(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!")

	encoded := EncodeSharedSecret(secret)
	if encoded == "" {
		t.Errorf("expected non-empty encoded secret")
	}

	decoded, err := DecodeSharedSecret(encoded)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(decoded) != len(secret) {
		t.Errorf("decoded secret length mismatch: got %d, want %d", len(decoded), len(secret))
	}

	for i := range secret {
		if decoded[i] != secret[i] {
			t.Errorf("decoded secret byte mismatch at index %d: got %d, want %d", i, decoded[i], secret[i])
		}
	}
}

// Helper method for testing with custom timestamps
func (tg *TokenGenerator) GenerateTokenWithTimestamp(requestID string, operation string, timestamp int64) (string, int64, error) {
	if requestID == "" {
		return "", 0, fmt.Errorf("request_id cannot be empty")
	}
	if operation == "" {
		return "", 0, fmt.Errorf("operation cannot be empty")
	}

	// Create the data to sign: request_id + timestamp + operation
	dataToSign := fmt.Sprintf("%s%d%s", requestID, timestamp, operation)

	// Calculate HMAC-SHA256 signature
	signature := tg.calculateHMAC(dataToSign)

	// Format token as: request_id:timestamp:operation:signature
	token := fmt.Sprintf("%s:%d:%s:%s", requestID, timestamp, operation, signature)

	return token, timestamp, nil
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsStringHelper(s, substr)))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
