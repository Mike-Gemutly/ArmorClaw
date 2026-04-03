// Package sidecar provides token generation for secure sidecar communication
package sidecar

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// TokenTTL is the token time-to-live (30 minutes per Oracle recommendation)
	TokenTTL = 30 * time.Minute

	// MaxTimestampAge is the maximum age of a token timestamp (5 minutes)
	// Tokens with timestamps older than this will be rejected
	MaxTimestampAge = 5 * time.Minute
)

// TokenGenerator generates ephemeral tokens for sidecar authentication
type TokenGenerator struct {
	sharedSecret []byte
}

// NewTokenGenerator creates a new TokenGenerator with the given shared secret
func NewTokenGenerator(sharedSecret []byte) (*TokenGenerator, error) {
	if len(sharedSecret) == 0 {
		return nil, fmt.Errorf("shared secret cannot be empty")
	}
	return &TokenGenerator{
		sharedSecret: sharedSecret,
	}, nil
}

// GenerateToken generates an ephemeral token for the given request
// Token format: {request_id}:{timestamp}:{operation}:{signature}
// The signature is HMAC-SHA256 of (request_id + timestamp + operation)
func (tg *TokenGenerator) GenerateToken(requestID string, operation string) (string, int64, error) {
	if requestID == "" {
		return "", 0, fmt.Errorf("request_id cannot be empty")
	}
	if operation == "" {
		return "", 0, fmt.Errorf("operation cannot be empty")
	}

	timestamp := time.Now().Unix()

	// Create the data to sign: request_id + timestamp + operation
	dataToSign := fmt.Sprintf("%s%d%s", requestID, timestamp, operation)

	// Calculate HMAC-SHA256 signature
	signature := tg.calculateHMAC(dataToSign)

	// Format token as: request_id:timestamp:operation:signature
	token := fmt.Sprintf("%s:%d:%s:%s", requestID, timestamp, operation, signature)

	return token, timestamp, nil
}

// calculateHMAC calculates the HMAC-SHA256 of the given data
func (tg *TokenGenerator) calculateHMAC(data string) string {
	h := hmac.New(sha256.New, tg.sharedSecret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// TokenInfo represents the parsed components of a token
type TokenInfo struct {
	RequestID  string
	Timestamp  int64
	Operation  string
	Signature  string
	Expiration int64 // Unix timestamp when token expires
}

// ParseToken parses a token into its components
func ParseToken(token string) (*TokenInfo, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid token format: expected 4 parts, got %d", len(parts))
	}

	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	return &TokenInfo{
		RequestID:  parts[0],
		Timestamp:  timestamp,
		Operation:  parts[2],
		Signature:  parts[3],
		Expiration: timestamp + int64(TokenTTL.Seconds()),
	}, nil
}

// GenerateRequestID generates a unique request ID using crypto/rand
func GenerateRequestID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate request ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ValidateTokenSignature validates the signature of a token
func (tg *TokenGenerator) ValidateTokenSignature(token string) (bool, error) {
	info, err := ParseToken(token)
	if err != nil {
		return false, err
	}

	// Recreate the data that was signed
	dataToSign := fmt.Sprintf("%s%d%s", info.RequestID, info.Timestamp, info.Operation)

	// Calculate the expected signature
	expectedSignature := tg.calculateHMAC(dataToSign)

	// Use constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(info.Signature), []byte(expectedSignature)), nil
}

// IsTokenExpired checks if a token has expired based on its TTL
func IsTokenExpired(info *TokenInfo) bool {
	now := time.Now().Unix()
	return now > info.Expiration
}

// IsTokenTooOld checks if a token's timestamp is too old (beyond MaxTimestampAge)
func IsTokenTooOld(info *TokenInfo) bool {
	now := time.Now().Unix()
	maxAge := int64(MaxTimestampAge.Seconds())
	return (now - info.Timestamp) > maxAge
}

// ValidateToken performs full validation of a token including signature, expiration, and age
func (tg *TokenGenerator) ValidateToken(token string) (*TokenInfo, error) {
	info, err := ParseToken(token)
	if err != nil {
		return nil, err
	}

	// Check if token is too old (timestamp > 5 minutes ago)
	if IsTokenTooOld(info) {
		return nil, fmt.Errorf("token timestamp is too old (> %s)", MaxTimestampAge)
	}

	// Check if token has expired (TTL exceeded)
	if IsTokenExpired(info) {
		return nil, fmt.Errorf("token has expired (TTL: %s)", TokenTTL)
	}

	// Verify signature
	valid, err := tg.ValidateTokenSignature(token)
	if err != nil {
		return nil, fmt.Errorf("signature validation failed: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid token signature")
	}

	return info, nil
}

// GenerateSharedSecret generates a random shared secret for development/testing
// In production, this should be loaded from the keystore
func GenerateSharedSecret() ([]byte, error) {
	secret := make([]byte, 32) // 256 bits
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate shared secret: %w", err)
	}
	return secret, nil
}

// EncodeSharedSecret encodes a shared secret to base64 for storage/transmission
func EncodeSharedSecret(secret []byte) string {
	return base64.StdEncoding.EncodeToString(secret)
}

// DecodeSharedSecret decodes a base64-encoded shared secret
func DecodeSharedSecret(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
