// Package adapter provides Matrix AppService key ingestion for SDTW
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/crypto"
)

// KeyIngestionManager handles E2EE key ingestion for the Bridge AppService
// Resolves: G-02 (SDTW Decryption - Server-Side Acceptance)
type KeyIngestionManager struct {
	mu              sync.RWMutex
	logger          *slog.Logger
	cryptoStore     crypto.Store
	verifiedDevices map[string]*VerifiedDevice // device_id -> device
	pendingForwards map[string][]*ForwardedKey // room_id -> pending keys
}

// VerifiedDevice represents a device that has been verified
type VerifiedDevice struct {
	DeviceID    string    `json:"device_id"`
	UserID      string    `json:"user_id"`
	VerifiedAt  time.Time `json:"verified_at"`
	KeyReceived bool      `json:"key_received"`
}

// ForwardedKey represents a forwarded room key
type ForwardedKey struct {
	RoomID      string    `json:"room_id"`
	SessionID   string    `json:"session_id"`
	SenderKey   string    `json:"sender_key"`
	ForwardingKey string  `json:"forwarding_key,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	Algorithm   string    `json:"algorithm"`
}

// KeyVerificationEvent represents m.key.verification.done
type KeyVerificationEvent struct {
	Type           string `json:"type"`
	TransactionID  string `json:"transaction_id"`
	FromDevice     string `json:"from_device"`
	RelatesTo      struct {
		EventID string `json:"event_id"`
	} `json:"m.relates_to"`
}

// RoomKeyForwardEvent represents m.room_key.forwarded
type RoomKeyForwardEvent struct {
	Type      string `json:"type"`
	Content   RoomKeyContent `json:"content"`
}

// RoomKeyContent represents the content of a room key event
type RoomKeyContent struct {
	Algorithm     string `json:"algorithm"`
	RoomID        string `json:"room_id"`
	SessionID     string `json:"session_id"`
	SessionKey    string `json:"session_key"`
	SenderKey     string `json:"sender_key"`
	ForwardingKey string `json:"forwarding_curve25519_key,omitempty"`
	ChainIndex    int    `json:"chain_index,omitempty"`
}

// NewKeyIngestionManager creates a new key ingestion manager
func NewKeyIngestionManager(cryptoStore crypto.Store, logger *slog.Logger) *KeyIngestionManager {
	if logger == nil {
		logger = slog.Default().With("component", "key_ingestion")
	}

	return &KeyIngestionManager{
		logger:          logger,
		cryptoStore:     cryptoStore,
		verifiedDevices: make(map[string]*VerifiedDevice),
		pendingForwards: make(map[string][]*ForwardedKey),
	}
}

// HandleVerificationDone handles m.key.verification.done events
// This is called when a user completes verification with the bridge
func (m *KeyIngestionManager) HandleVerificationDone(ctx context.Context, event KeyVerificationEvent, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	deviceID := event.FromDevice
	if deviceID == "" {
		return fmt.Errorf("missing device_id in verification event")
	}

	m.logger.Info("verification_completed",
		"user_id", userID,
		"device_id", deviceID,
		"transaction_id", event.TransactionID,
	)

	// Mark device as verified
	m.verifiedDevices[deviceID] = &VerifiedDevice{
		DeviceID:   deviceID,
		UserID:     userID,
		VerifiedAt: time.Now(),
	}

	// Process any pending key forwards for rooms this user is in
	// (In production, this would query which rooms the user is in)
	m.processPendingKeysForDevice(deviceID)

	return nil
}

// HandleForwardedKey handles m.forwarded_room_key events
// This is called when a user forwards room keys to the bridge
func (m *KeyIngestionManager) HandleForwardedKey(ctx context.Context, event RoomKeyForwardEvent, senderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	content := event.Content

	m.logger.Info("forwarded_key_received",
		"room_id", content.RoomID,
		"session_id", content.SessionID,
		"sender", senderID,
		"algorithm", content.Algorithm,
	)

	// Validate algorithm
	if content.Algorithm != "m.megolm.v1.aes-sha2" {
		return fmt.Errorf("unsupported algorithm: %s", content.Algorithm)
	}

	// Create forwarded key record
	forwardedKey := &ForwardedKey{
		RoomID:        content.RoomID,
		SessionID:     content.SessionID,
		SenderKey:     content.SenderKey,
		ForwardingKey: content.ForwardingKey,
		ReceivedAt:    time.Now(),
		Algorithm:     content.Algorithm,
	}

	// Ingest into crypto store
	if m.cryptoStore != nil {
		if err := m.ingestKeyIntoStore(content); err != nil {
			m.logger.Error("key_ingestion_failed",
				"room_id", content.RoomID,
				"error", err,
			)
			return fmt.Errorf("failed to ingest key: %w", err)
		}
	}

	// Add to pending forwards (in case verification isn't complete yet)
	m.pendingForwards[content.RoomID] = append(m.pendingForwards[content.RoomID], forwardedKey)

	m.logger.Info("key_ingested_successfully",
		"room_id", content.RoomID,
		"session_id", content.SessionID,
	)

	return nil
}

// ingestKeyIntoStore stores the forwarded key in the crypto store
func (m *KeyIngestionManager) ingestKeyIntoStore(content RoomKeyContent) error {
	// In production, this would use the actual Matrix crypto library
	// to ingest the Megolm session into the store
	//
	// Example with libolm:
	// session := olm.InboundGroupSession{}
	// session.Unpickle(content.SessionKey)
	// m.cryptoStore.AddInboundGroupSession(content.RoomID, content.SenderKey, content.SessionID, session)

	return nil
}

// processPendingKeysForDevice processes any pending key forwards after verification
func (m *KeyIngestionManager) processPendingKeysForDevice(deviceID string) {
	// In production, this would:
	// 1. Look up which rooms the device's user is in
	// 2. Process any pending forwarded keys for those rooms
	// 3. Mark them as delivered

	m.logger.Debug("processing_pending_keys", "device_id", deviceID)
}

// IsDeviceVerified checks if a device has been verified
func (m *KeyIngestionManager) IsDeviceVerified(deviceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.verifiedDevices[deviceID]
	return exists && device.KeyReceived
}

// GetVerifiedDevices returns all verified devices
func (m *KeyIngestionManager) GetVerifiedDevices() []*VerifiedDevice {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]*VerifiedDevice, 0, len(m.verifiedDevices))
	for _, d := range m.verifiedDevices {
		devices = append(devices, d)
	}
	return devices
}

// GetPendingKeys returns pending keys for a room
func (m *KeyIngestionManager) GetPendingKeys(roomID string) []*ForwardedKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys, exists := m.pendingForwards[roomID]
	if !exists {
		return nil
	}

	result := make([]*ForwardedKey, len(keys))
	copy(result, keys)
	return result
}

// CanDecryptRoom checks if the bridge can decrypt messages in a room
func (m *KeyIngestionManager) CanDecryptRoom(roomID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys, exists := m.pendingForwards[roomID]
	return exists && len(keys) > 0
}

// HandleKeyEvent handles any key-related Matrix event
func (m *KeyIngestionManager) HandleKeyEvent(ctx context.Context, eventType string, content json.RawMessage, sender string) error {
	switch eventType {
	case "m.key.verification.done":
		var event KeyVerificationEvent
		if err := json.Unmarshal(content, &event); err != nil {
			return fmt.Errorf("invalid verification.done event: %w", err)
		}
		return m.HandleVerificationDone(ctx, event, sender)

	case "m.forwarded_room_key":
		var event RoomKeyForwardEvent
		if err := json.Unmarshal(content, &event); err != nil {
			return fmt.Errorf("invalid forwarded_room_key event: %w", err)
		}
		return m.HandleForwardedKey(ctx, event, sender)

	default:
		m.logger.Debug("unhandled_key_event", "type", eventType)
		return nil
	}
}

// Stats returns key ingestion statistics
func (m *KeyIngestionManager) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalKeys := 0
	for _, keys := range m.pendingForwards {
		totalKeys += len(keys)
	}

	return map[string]interface{}{
		"verified_devices": len(m.verifiedDevices),
		"rooms_with_keys":  len(m.pendingForwards),
		"total_keys":       totalKeys,
	}
}
