// Package appservice provides bridge management for SDTW adapters
package appservice

import (
	"testing"
)

func TestNewBridgeManager(t *testing.T) {
	as, _ := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})

	client := NewClient("https://matrix.example.com", "test_as_token")

	tests := []struct {
		name    string
		config  BridgeConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: BridgeConfig{
				AppService: as,
				Client:     client,
			},
			wantErr: false,
		},
		{
			name: "missing AppService",
			config: BridgeConfig{
				Client: client,
			},
			wantErr: true,
		},
		{
			name: "missing Client",
			config: BridgeConfig{
				AppService: as,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bm, err := NewBridgeManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBridgeManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && bm == nil {
				t.Error("NewBridgeManager() returned nil without error")
			}
		})
	}
}

func TestBridgeManagerChannelManagement(t *testing.T) {
	as, _ := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})

	client := NewClient("https://matrix.example.com", "test_as_token")

	bm, err := NewBridgeManager(BridgeConfig{
		AppService: as,
		Client:     client,
	})
	if err != nil {
		t.Fatalf("Failed to create BridgeManager: %v", err)
	}

	// Test BridgeChannel
	err = bm.BridgeChannel("!room:example.com", PlatformSlack, "C12345")
	if err != nil {
		t.Errorf("BridgeChannel() error = %v", err)
	}

	// Test duplicate bridge
	err = bm.BridgeChannel("!room:example.com", PlatformSlack, "C12345")
	if err == nil {
		t.Error("BridgeChannel() should fail for duplicate channel")
	}

	// Test GetBridgedChannels
	channels := bm.GetBridgedChannels()
	if len(channels) != 1 {
		t.Errorf("GetBridgedChannels() count = %d, want 1", len(channels))
	}

	// Verify channel details
	if channels[0].MatrixRoomID != "!room:example.com" {
		t.Errorf("Channel MatrixRoomID = %s, want !room:example.com", channels[0].MatrixRoomID)
	}
	if channels[0].Platform != PlatformSlack {
		t.Errorf("Channel Platform = %s, want slack", channels[0].Platform)
	}
	if channels[0].ChannelID != "C12345" {
		t.Errorf("Channel ChannelID = %s, want C12345", channels[0].ChannelID)
	}
}

func TestBridgeManagerStats(t *testing.T) {
	as, _ := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})

	client := NewClient("https://matrix.example.com", "test_as_token")

	bm, _ := NewBridgeManager(BridgeConfig{
		AppService: as,
		Client:     client,
	})

	// Add some channels
	bm.BridgeChannel("!room1:example.com", PlatformSlack, "C1")
	bm.BridgeChannel("!room2:example.com", PlatformDiscord, "D1")

	stats := bm.GetStats()

	adapterCount, ok := stats["adapters"].(int)
	if !ok {
		t.Error("Stats adapters is not int")
	}
	if adapterCount != 0 {
		t.Errorf("Stats adapters = %d, want 0 (no adapters registered)", adapterCount)
	}

	channelCount, ok := stats["bridged_channels"].(int)
	if !ok {
		t.Error("Stats bridged_channels is not int")
	}
	if channelCount != 2 {
		t.Errorf("Stats bridged_channels = %d, want 2", channelCount)
	}

	phiScrubbing, ok := stats["phi_scrubbing"].(bool)
	if !ok {
		t.Error("Stats phi_scrubbing is not bool")
	}
	if phiScrubbing {
		t.Error("Stats phi_scrubbing = true, want false (not configured)")
	}
}

func TestBridgeManagerPHIScrubbing(t *testing.T) {
	as, _ := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})

	client := NewClient("https://matrix.example.com", "test_as_token")

	bm, _ := NewBridgeManager(BridgeConfig{
		AppService:          as,
		Client:              client,
		EnablePHIScrubbing:  true,
		ComplianceTier:      "standard",
	})

	stats := bm.GetStats()

	phiScrubbing, ok := stats["phi_scrubbing"].(bool)
	if !ok {
		t.Error("Stats phi_scrubbing is not bool")
	}
	if !phiScrubbing {
		t.Error("Stats phi_scrubbing = false, want true")
	}

	if stats["compliance_tier"] != "standard" {
		t.Errorf("Stats compliance_tier = %v, want standard", stats["compliance_tier"])
	}
}

func TestPlatformConstants(t *testing.T) {
	platforms := map[Platform]string{
		PlatformSlack:    "slack",
		PlatformDiscord:  "discord",
		PlatformTeams:    "teams",
		PlatformWhatsApp: "whatsapp",
	}

	for platform, expected := range platforms {
		if string(platform) != expected {
			t.Errorf("Platform %s = %s, want %s", platform, string(platform), expected)
		}
	}
}
