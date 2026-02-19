// Package appservice provides bridge management for SDTW adapters
// This file contains the BridgeManager that coordinates SDTW adapters with Matrix
package appservice

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/sdtw"
	"github.com/armorclaw/bridge/pkg/pii"
)

// Platform identifies an external messaging platform
type Platform string

const (
	PlatformSlack    Platform = "slack"
	PlatformDiscord  Platform = "discord"
	PlatformTeams    Platform = "teams"
	PlatformWhatsApp Platform = "whatsapp"
)

// BridgeConfig configures the bridge manager
type BridgeConfig struct {
	// AppService configuration
	AppService *AppService
	Client     *Client

	// Compliance settings
	EnablePHIScrubbing bool
	ComplianceTier     string // basic, standard, full

	// Platform adapters
	Adapters map[Platform]sdtw.SDTWAdapter
}

// BridgedChannel represents a bridged channel between Matrix and external platform
type BridgedChannel struct {
	MatrixRoomID string    `json:"matrix_room_id"`
	Platform     Platform  `json:"platform"`
	ChannelID    string    `json:"channel_id"`
	ChannelName  string    `json:"channel_name"`
	GhostUsers   []string  `json:"ghost_users"` // Matrix IDs of ghost users in this channel
	Enabled      bool      `json:"enabled"`
}

// PlatformEventHandler handles events from external platforms
type PlatformEventHandler func(platform Platform, event sdtw.ExternalEvent)

// BridgeManager coordinates SDTW adapters with the Matrix AppService
type BridgeManager struct {
	config    BridgeConfig
	as        *AppService
	client    *Client
	logger    *slog.Logger
	scrubber  *pii.HIPAAScrubber

	// Platform adapters
	adapters  map[Platform]sdtw.SDTWAdapter
	adapterMu sync.RWMutex

	// Bridged channels
	channels  map[string]*BridgedChannel // key: platform:channelID
	channelMu sync.RWMutex

	// Event handling
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewBridgeManager creates a new bridge manager
func NewBridgeManager(config BridgeConfig) (*BridgeManager, error) {
	if config.AppService == nil {
		return nil, fmt.Errorf("AppService is required")
	}
	if config.Client == nil {
		return nil, fmt.Errorf("Client is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	bm := &BridgeManager{
		config:   config,
		as:       config.AppService,
		client:   config.Client,
		adapters: config.Adapters,
		channels: make(map[string]*BridgedChannel),
		ctx:      ctx,
		cancel:   cancel,
		logger: slog.Default().With(
			"component", "bridge_manager",
		),
	}

	// Initialize PHI scrubber if enabled
	if config.EnablePHIScrubbing {
		bm.scrubber = pii.NewHIPAAScrubber(pii.HIPAAConfig{
			Tier:           pii.HIPAATier(config.ComplianceTier),
			EnableAuditLog: true,
		})
	}

	// Initialize adapters map if not provided
	if bm.adapters == nil {
		bm.adapters = make(map[Platform]sdtw.SDTWAdapter)
	}

	return bm, nil
}

// Start begins processing events from Matrix and platforms
func (bm *BridgeManager) Start() error {
	bm.logger.Info("starting_bridge_manager")

	// Start listening for Matrix events
	bm.wg.Add(1)
	go bm.processMatrixEvents()

	// Start platform adapters
	for platform, adapter := range bm.adapters {
		if err := adapter.Start(bm.ctx); err != nil {
			bm.logger.Error("adapter_start_failed",
				"platform", platform,
				"error", err,
			)
			continue
		}
		bm.logger.Info("adapter_started", "platform", platform)
	}

	return nil
}

// Stop gracefully shuts down the bridge manager
func (bm *BridgeManager) Stop() error {
	bm.logger.Info("stopping_bridge_manager")
	bm.cancel()

	// Stop all adapters
	for platform, adapter := range bm.adapters {
		if err := adapter.Shutdown(bm.ctx); err != nil {
			bm.logger.Error("adapter_stop_failed",
				"platform", platform,
				"error", err,
			)
		}
	}

	bm.wg.Wait()
	return nil
}

// RegisterAdapter registers a platform adapter
func (bm *BridgeManager) RegisterAdapter(platform Platform, adapter sdtw.SDTWAdapter) error {
	bm.adapterMu.Lock()
	defer bm.adapterMu.Unlock()

	if _, exists := bm.adapters[platform]; exists {
		return fmt.Errorf("adapter for %s already registered", platform)
	}

	bm.adapters[platform] = adapter
	bm.logger.Info("adapter_registered", "platform", platform)
	return nil
}

// BridgeChannel creates a bridge between a Matrix room and platform channel
func (bm *BridgeManager) BridgeChannel(matrixRoomID string, platform Platform, channelID string) error {
	key := fmt.Sprintf("%s:%s", platform, channelID)

	bm.channelMu.Lock()
	defer bm.channelMu.Unlock()

	if _, exists := bm.channels[key]; exists {
		return fmt.Errorf("channel %s already bridged", key)
	}

	channel := &BridgedChannel{
		MatrixRoomID: matrixRoomID,
		Platform:     platform,
		ChannelID:    channelID,
		Enabled:      true,
		GhostUsers:   make([]string, 0),
	}

	bm.channels[key] = channel

	bm.logger.Info("channel_bridged",
		"matrix_room", matrixRoomID,
		"platform", platform,
		"channel", channelID,
	)

	return nil
}

// UnbridgeChannel removes a bridge
func (bm *BridgeManager) UnbridgeChannel(platform Platform, channelID string) error {
	key := fmt.Sprintf("%s:%s", platform, channelID)

	bm.channelMu.Lock()
	defer bm.channelMu.Unlock()

	channel, exists := bm.channels[key]
	if !exists {
		return fmt.Errorf("channel %s not bridged", key)
	}

	// Leave Matrix room (as bridge bot)
	bridgeUser := bm.as.GetBridgeUserID()
	bm.client.LeaveRoom(bm.ctx, channel.MatrixRoomID, bridgeUser)

	delete(bm.channels, key)

	bm.logger.Info("channel_unbridged",
		"platform", platform,
		"channel", channelID,
	)

	return nil
}

// processMatrixEvents handles events from Matrix to external platforms
func (bm *BridgeManager) processMatrixEvents() {
	defer bm.wg.Done()

	for {
		select {
		case <-bm.ctx.Done():
			return
		case event, ok := <-bm.as.Events():
			if !ok {
				return
			}
			bm.handleMatrixEvent(event)
		}
	}
}

// handleMatrixEvent processes a Matrix event and forwards to platforms
func (bm *BridgeManager) handleMatrixEvent(event Event) {
	// Only handle message events
	if event.Type != "m.room.message" {
		return
	}

	// Skip events from our own ghost users
	if bm.as.isGhostUser(event.Sender) {
		return
	}

	// Find bridged channels for this room
	bm.channelMu.RLock()
	var targetChannels []*BridgedChannel
	for _, ch := range bm.channels {
		if ch.MatrixRoomID == event.RoomID && ch.Enabled {
			targetChannels = append(targetChannels, ch)
		}
	}
	bm.channelMu.RUnlock()

	if len(targetChannels) == 0 {
		return // No bridges for this room
	}

	// Extract message content
	content, ok := event.Content["body"].(string)
	if !ok {
		return
	}

	// Scrub PHI if enabled
	if bm.scrubber != nil {
		var err error
		content, _, err = bm.scrubber.ScrubPHI(bm.ctx, content, event.RoomID)
		if err != nil {
			bm.logger.Error("phi_scrub_failed",
				"event_id", event.EventID,
				"error", err,
			)
			return
		}
	}

	// Forward to each platform
	for _, ch := range targetChannels {
		adapter, ok := bm.adapters[ch.Platform]
		if !ok {
			bm.logger.Warn("no_adapter_for_platform",
				"platform", ch.Platform,
			)
			continue
		}

		msg := sdtw.Message{
			ID:        event.EventID,
			Content:   content,
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Unix(event.OriginServerTS/1000, 0),
		}

		target := sdtw.Target{
			Platform: string(ch.Platform),
			Channel:  ch.ChannelID,
		}

		if _, err := adapter.SendMessage(bm.ctx, target, msg); err != nil {
			bm.logger.Error("send_to_platform_failed",
				"platform", ch.Platform,
				"channel", ch.ChannelID,
				"error", err,
			)
		}
	}
}

// HandlePlatformEvent processes an event from an external platform to Matrix
func (bm *BridgeManager) HandlePlatformEvent(evt sdtw.ExternalEvent) {
	platform := Platform(evt.Platform)

	// Find bridged channel
	key := fmt.Sprintf("%s:%s", platform, evt.Source)

	bm.channelMu.RLock()
	channel, exists := bm.channels[key]
	bm.channelMu.RUnlock()

	if !exists || !channel.Enabled {
		return
	}

	// Get or create ghost user for sender
	senderID := evt.Metadata["user_id"]
	if senderID == "" {
		senderID = evt.Source
	}
	ghostUserID := bm.as.GenerateGhostUserID(string(platform), senderID)

	// Register ghost user if not exists
	ghostUser, exists := bm.as.GetGhostUser(ghostUserID)
	if !exists {
		senderName := evt.Metadata["user_name"]
		ghostUser = &GhostUser{
			UserID:      ghostUserID,
			Platform:    string(platform),
			ExternalID:  senderID,
			DisplayName: senderName,
		}
		if err := bm.as.RegisterGhostUser(ghostUser); err != nil {
			bm.logger.Error("register_ghost_user_failed",
				"user_id", ghostUserID,
				"error", err,
			)
			return
		}

		// Ensure ghost user is in the Matrix room
		if err := bm.client.JoinRoom(bm.ctx, channel.MatrixRoomID, ghostUserID); err != nil {
			bm.logger.Error("ghost_join_room_failed",
				"user_id", ghostUserID,
				"room_id", channel.MatrixRoomID,
				"error", err,
			)
		}

		// Update display name if provided
		if senderName != "" {
			bm.client.SetDisplayName(bm.ctx, senderName, ghostUserID)
		}
	}

	// Send message to Matrix
	content := evt.Content
	_, err := bm.client.SendText(bm.ctx, channel.MatrixRoomID, content, ghostUserID)
	if err != nil {
		bm.logger.Error("send_to_matrix_failed",
			"room_id", channel.MatrixRoomID,
			"user_id", ghostUserID,
			"error", err,
		)
	}
}

// GetBridgedChannels returns all bridged channels
func (bm *BridgeManager) GetBridgedChannels() []*BridgedChannel {
	bm.channelMu.RLock()
	defer bm.channelMu.RUnlock()

	channels := make([]*BridgedChannel, 0, len(bm.channels))
	for _, ch := range bm.channels {
		channels = append(channels, ch)
	}
	return channels
}

// GetStats returns bridge manager statistics
func (bm *BridgeManager) GetStats() map[string]interface{} {
	bm.channelMu.RLock()
	channelCount := len(bm.channels)
	bm.channelMu.RUnlock()

	bm.adapterMu.RLock()
	adapterCount := len(bm.adapters)
	bm.adapterMu.RUnlock()

	return map[string]interface{}{
		"adapters":         adapterCount,
		"bridged_channels": channelCount,
		"phi_scrubbing":    bm.scrubber != nil,
		"compliance_tier":  bm.config.ComplianceTier,
	}
}
