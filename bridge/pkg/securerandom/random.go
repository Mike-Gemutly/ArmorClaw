// Package securerandom provides cryptographically secure random generation
package securerandom

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
)

// ID generates a cryptographically secure random ID of the specified byte length
// Returns a hex-encoded string (2x the byte length)
func ID(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := crand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// MustID generates a random ID or panics
// Use only in initialization or when failure is unrecoverable
func MustID(byteLen int) string {
	id, err := ID(byteLen)
	if err != nil {
		panic(fmt.Sprintf("securerandom.ID failed: %v", err))
	}
	return id
}

// Bytes generates cryptographically secure random bytes
func Bytes(byteLen int) ([]byte, error) {
	b := make([]byte, byteLen)
	if _, err := crand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// MustBytes generates random bytes or panics
func MustBytes(byteLen int) []byte {
	b, err := Bytes(byteLen)
	if err != nil {
		panic(fmt.Sprintf("securerandom.Bytes failed: %v", err))
	}
	return b
}

// Token generates a URL-safe random token
func Token(byteLen int) (string, error) {
	b, err := Bytes(byteLen)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// MustToken generates a token or panics
func MustToken(byteLen int) string {
	token, err := Token(byteLen)
	if err != nil {
		panic(fmt.Sprintf("securerandom.Token failed: %v", err))
	}
	return token
}

// Fill fills a byte slice with cryptographically secure random bytes
func Fill(b []byte) error {
	if _, err := crand.Read(b); err != nil {
		return fmt.Errorf("failed to fill random bytes: %w", err)
	}
	return nil
}

// MustFill fills a byte slice with random bytes or panics
func MustFill(b []byte) {
	if err := Fill(b); err != nil {
		panic(fmt.Sprintf("securerandom.Fill failed: %v", err))
	}
}

// Challenge generates a random challenge string for authentication
func Challenge() (string, error) {
	return ID(32) // 64 character hex string
}

// MustChallenge generates a challenge or panics
func MustChallenge() string {
	return MustID(32)
}

// Nonce generates a random nonce
func Nonce(byteLen int) ([]byte, error) {
	return Bytes(byteLen)
}

// MustNonce generates a nonce or panics
func MustNonce(byteLen int) []byte {
	return MustBytes(byteLen)
}
