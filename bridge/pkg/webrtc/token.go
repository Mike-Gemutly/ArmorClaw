package webrtc

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Token represents a short-lived call session token used for:
// - WebRTC signaling authentication
// - TURN credential derivation
type Token struct {
	SessionID  string    `json:"session_id"`
	RoomID     string    `json:"room_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	Signature  string    `json:"signature"`
}

// TokenClaims represents the claims within a token
type TokenClaims struct {
	SessionID string `json:"session_id"`
	RoomID    string `json:"room_id"`
	ExpiresAt int64  `json:"expires_at"`
	IssuedAt int64  `json:"issued_at"`
}

// TokenManager generates and validates call session tokens
type TokenManager struct {
	secret []byte // HMAC secret key
	ttl    time.Duration // Token TTL
}

// NewTokenManager creates a new token manager with the given secret and TTL
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Generate creates a new token for the given session and room
func (tm *TokenManager) Generate(sessionID, roomID string) (*Token, error) {
	now := time.Now()
	expiresAt := now.Add(tm.ttl)

	claims := TokenClaims{
		SessionID: sessionID,
		RoomID:    roomID,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
	}

	// Marshal claims to JSON
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Create signature
	signature := tm.sign(claimsJSON)

	token := &Token{
		SessionID: sessionID,
		RoomID:    roomID,
		ExpiresAt: expiresAt,
		Signature: signature,
	}

	return token, nil
}

// Validate checks if a token is valid and returns the associated claims
func (tm *TokenManager) Validate(token *Token) (*TokenClaims, error) {
	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Reconstruct claims
	claims := TokenClaims{
		SessionID: token.SessionID,
		RoomID:    token.RoomID,
		ExpiresAt: token.ExpiresAt.Unix(),
		IssuedAt:  token.ExpiresAt.Unix() - int64(tm.ttl.Seconds()),
	}

	// Marshal claims to JSON
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Verify signature
	expectedSignature := tm.sign(claimsJSON)
	if token.Signature != expectedSignature {
		return nil, ErrTokenInvalid
	}

	return &claims, nil
}

// GenerateTURNCredentials creates ephemeral TURN credentials from a token
// Format: username = <expiry>:<session_id>, password = HMAC(secret, username)
func (tm *TokenManager) GenerateTURNCredentials(token *Token, turnServer, stunServer string) *TURNCredentials {
	// Create username in format: <expiry>:<session_id>
	username := fmt.Sprintf("%d:%s", token.ExpiresAt.Unix(), token.SessionID)

	// Generate password as HMAC of username
	password := tm.hmac(username)

	return &TURNCredentials{
		Username:  username,
		Password:  password,
		Expires:   token.ExpiresAt,
		TURNServer: turnServer,
		STUNServer: stunServer,
	}
}

// ValidateTURNCredentials validates TURN credentials against a token
func (tm *TokenManager) ValidateTURNCredentials(username, password string) (*TokenClaims, error) {
	// Parse username to get expiry and session ID
	var expiry int64
	var sessionID string

	_, err := fmt.Sscanf(username, "%d:%s", &expiry, &sessionID)
	if err != nil {
		return nil, ErrTURNInvalidFormat
	}

	// Check expiration
	if time.Now().Unix() > expiry {
		return nil, ErrTURNExpired
	}

	// Verify password
	expectedPassword := tm.hmac(username)
	if password != expectedPassword {
		return nil, ErrTURNInvalidPassword
	}

	// Return claims (note: we don't have room_id here, but session_id is enough)
	claims := &TokenClaims{
		SessionID: sessionID,
		ExpiresAt: expiry,
	}

	return claims, nil
}

// sign creates an HMAC signature of the data
func (tm *TokenManager) sign(data []byte) string {
	h := hmac.New(sha256.New, tm.secret)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmac creates an HMAC hash of the string
func (tm *TokenManager) hmac(s string) string {
	h := hmac.New(sha256.New, tm.secret)
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// ToJSON converts the token to a JSON string for transport
func (t *Token) ToJSON() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TokenFromJSON parses a token from a JSON string
func TokenFromJSON(jsonStr string) (*Token, error) {
	var token Token
	err := json.Unmarshal([]byte(jsonStr), &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// Errors
var (
	// ErrTokenExpired is returned when a token has passed its expiration time
	ErrTokenExpired = &TokenError{Code: "token_expired", Message: "token has expired"}

	// ErrTokenInvalid is returned when a token's signature is invalid
	ErrTokenInvalid = &TokenError{Code: "token_invalid", Message: "token signature is invalid"}

	// ErrTURNInvalidFormat is returned when TURN credentials have invalid format
	ErrTURNInvalidFormat = &TokenError{Code: "turn_invalid_format", Message: "TURN credentials format is invalid"}

	// ErrTURNExpired is returned when TURN credentials have expired
	ErrTURNExpired = &TokenError{Code: "turn_expired", Message: "TURN credentials have expired"}

	// ErrTURNInvalidPassword is returned when TURN password is invalid
	ErrTURNInvalidPassword = &TokenError{Code: "turn_invalid_password", Message: "TURN password is invalid"}
)

// TokenError represents an error related to token management
type TokenError struct {
	Code    string
	Message string
}

func (e *TokenError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
