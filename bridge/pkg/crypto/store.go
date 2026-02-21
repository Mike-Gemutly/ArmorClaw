// Package crypto provides cryptographic interfaces for E2EE support
package crypto

import (
	"context"
	"errors"
)

// Store defines the interface for cryptographic key storage
// Used by the Bridge AppService to store ingested room keys
type Store interface {
	// AddInboundGroupSession stores an inbound Megolm session
	AddInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string, sessionKey []byte) error

	// GetInboundGroupSession retrieves an inbound Megolm session
	GetInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string) ([]byte, error)

	// HasInboundGroupSession checks if a session exists
	HasInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string) bool

	// Clear removes all stored sessions
	Clear(ctx context.Context) error
}

// MemoryStore is an in-memory implementation of Store for testing
type MemoryStore struct {
	sessions map[string][]byte // key: roomID:senderKey:sessionID
}

// NewMemoryStore creates a new in-memory crypto store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string][]byte),
	}
}

func (s *MemoryStore) sessionKey(roomID, senderKey, sessionID string) string {
	return roomID + ":" + senderKey + ":" + sessionID
}

// AddInboundGroupSession stores an inbound Megolm session
func (s *MemoryStore) AddInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string, sessionKey []byte) error {
	key := s.sessionKey(roomID, senderKey, sessionID)
	s.sessions[key] = sessionKey
	return nil
}

// GetInboundGroupSession retrieves an inbound Megolm session
func (s *MemoryStore) GetInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string) ([]byte, error) {
	key := s.sessionKey(roomID, senderKey, sessionID)
	session, exists := s.sessions[key]
	if !exists {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// HasInboundGroupSession checks if a session exists
func (s *MemoryStore) HasInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string) bool {
	key := s.sessionKey(roomID, senderKey, sessionID)
	_, exists := s.sessions[key]
	return exists
}

// Clear removes all stored sessions
func (s *MemoryStore) Clear(ctx context.Context) error {
	s.sessions = make(map[string][]byte)
	return nil
}

// ErrSessionNotFound is returned when a session is not found
var ErrSessionNotFound = errors.New("session not found")
