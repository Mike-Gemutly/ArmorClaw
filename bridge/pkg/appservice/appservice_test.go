// Package appservice provides Matrix Application Service functionality for ArmorClaw.
package appservice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewAppService(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				HomeserverURL: "https://matrix.example.com",
				ASToken:       "test_as_token",
				HSToken:       "test_hs_token",
				ID:            "armorclaw-bridge",
				ServerName:    "example.com",
			},
			wantErr: false,
		},
		{
			name: "missing homeserver URL",
			config: Config{
				ASToken: "test_as_token",
				HSToken: "test_hs_token",
			},
			wantErr: true,
		},
		{
			name: "missing AS token",
			config: Config{
				HomeserverURL: "https://matrix.example.com",
				HSToken:       "test_hs_token",
			},
			wantErr: true,
		},
		{
			name: "missing HS token",
			config: Config{
				HomeserverURL: "https://matrix.example.com",
				ASToken:       "test_as_token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && as == nil {
				t.Error("New() returned nil without error")
			}
		})
	}
}

func TestAppServiceGenerateGhostUserID(t *testing.T) {
	as, err := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create AppService: %v", err)
	}

	tests := []struct {
		platform   string
		externalID string
		expected   string
	}{
		{"slack", "U12345", "@slack_U12345:example.com"},
		{"discord", "123456789", "@discord_123456789:example.com"},
		{"teams", "user@domain.com", "@teams_user_domain_com:example.com"},
		{"whatsapp", "+1234567890", "@whatsapp__1234567890:example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			result := as.GenerateGhostUserID(tt.platform, tt.externalID)
			if result != tt.expected {
				t.Errorf("GenerateGhostUserID(%s, %s) = %s, want %s",
					tt.platform, tt.externalID, result, tt.expected)
			}
		})
	}
}

func TestAppServiceGhostUserManagement(t *testing.T) {
	as, err := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create AppService: %v", err)
	}

	ghostUser := &GhostUser{
		UserID:      "@slack_U12345:example.com",
		Platform:    "slack",
		ExternalID:  "U12345",
		DisplayName: "Test User",
	}

	// Test registration
	err = as.RegisterGhostUser(ghostUser)
	if err != nil {
		t.Errorf("RegisterGhostUser() error = %v", err)
	}

	// Test retrieval
	retrieved, exists := as.GetGhostUser(ghostUser.UserID)
	if !exists {
		t.Error("GetGhostUser() returned exists = false")
	}
	if retrieved.UserID != ghostUser.UserID {
		t.Errorf("GetGhostUser() UserID = %s, want %s", retrieved.UserID, ghostUser.UserID)
	}
	if retrieved.DisplayName != ghostUser.DisplayName {
		t.Errorf("GetGhostUser() DisplayName = %s, want %s", retrieved.DisplayName, ghostUser.DisplayName)
	}

	// Test isGhostUser
	if !as.isGhostUser(ghostUser.UserID) {
		t.Error("isGhostUser() returned false for registered user")
	}

	// Test duplicate registration
	err = as.RegisterGhostUser(ghostUser)
	if err == nil {
		t.Error("RegisterGhostUser() should fail for duplicate user")
	}
}

func TestAppServiceTransactionHandling(t *testing.T) {
	as, err := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create AppService: %v", err)
	}

	// Start the AppService
	go as.Start()
	defer as.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create test transaction
	txn := Transaction{
		Events: []Event{
			{
				Type:           "m.room.message",
				RoomID:         "!room:example.com",
				Sender:         "@user:example.com",
				Content:        map[string]interface{}{"body": "Hello"},
				EventID:        "$event1",
				OriginServerTS: time.Now().UnixMilli(),
			},
		},
	}

	txnBody, _ := json.Marshal(txn)

	// Create test request
	req := httptest.NewRequest(http.MethodPut, "/transactions/1", strings.NewReader(string(txnBody)))
	req.Header.Set("Authorization", "Bearer test_hs_token")

	rr := httptest.NewRecorder()
	as.handleTransaction(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleTransaction() status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Check event was received
	select {
	case event := <-as.Events():
		if event.EventID != "$event1" {
			t.Errorf("Received event ID = %s, want $event1", event.EventID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Did not receive event from channel")
	}
}

func TestAppServiceTokenVerification(t *testing.T) {
	as, err := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create AppService: %v", err)
	}

	tests := []struct {
		name       string
		setupReq   func(*http.Request)
		wantStatus int
	}{
		{
			name: "valid token in header",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer test_hs_token")
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid token in query",
			setupReq: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("access_token", "test_hs_token")
				r.URL.RawQuery = q.Encode()
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid token",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid_token")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "missing token",
			setupReq: func(r *http.Request) {
				// No token
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := Transaction{Events: []Event{}}
			txnBody, _ := json.Marshal(txn)

			req := httptest.NewRequest(http.MethodPut, "/transactions/1", strings.NewReader(string(txnBody)))
			tt.setupReq(req)

			rr := httptest.NewRecorder()
			as.handleTransaction(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("handleTransaction() status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestAppServiceGetBridgeUserID(t *testing.T) {
	as, err := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
		SenderLocalpart: "_bridge",
	})
	if err != nil {
		t.Fatalf("Failed to create AppService: %v", err)
	}

	expected := "@_bridge:example.com"
	result := as.GetBridgeUserID()

	if result != expected {
		t.Errorf("GetBridgeUserID() = %s, want %s", result, expected)
	}
}

func TestAppServiceGetStats(t *testing.T) {
	as, err := New(Config{
		HomeserverURL: "https://matrix.example.com",
		ASToken:       "test_as_token",
		HSToken:       "test_hs_token",
		ID:            "armorclaw-bridge",
		ServerName:    "example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create AppService: %v", err)
	}

	// Register some ghost users
	as.RegisterGhostUser(&GhostUser{UserID: "@slack_U1:example.com", Platform: "slack"})
	as.RegisterGhostUser(&GhostUser{UserID: "@discord_U2:example.com", Platform: "discord"})

	stats := as.GetStats()

	if stats["id"] != "armorclaw-bridge" {
		t.Errorf("GetStats() id = %v, want armorclaw-bridge", stats["id"])
	}

	ghostCount, ok := stats["ghost_users"].(int)
	if !ok {
		t.Error("GetStats() ghost_users is not int")
	}
	if ghostCount != 2 {
		t.Errorf("GetStats() ghost_users = %d, want 2", ghostCount)
	}
}
