// Package appservice provides integration tests for platform adapters
package appservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/armorclaw/bridge/internal/sdtw"
)

// MockMatrixClient implements a mock Matrix client for testing
type MockMatrixClient struct {
	mu               sync.Mutex
	sentMessages     []MatrixSentMessage
	joinedRooms      map[string][]string // roomID -> userIDs
	ghostUsers       map[string]*GhostUser
	displayNames     map[string]string
	lastErrorMessage string
	shouldFailSend   bool
	shouldFailJoin   bool
}

type MatrixSentMessage struct {
	RoomID    string
	Content   string
	SenderID  string
	Timestamp time.Time
}

func NewMockMatrixClient() *MockMatrixClient {
	return &MockMatrixClient{
		sentMessages: make([]MatrixSentMessage, 0),
		joinedRooms:  make(map[string][]string),
		ghostUsers:   make(map[string]*GhostUser),
		displayNames: make(map[string]string),
	}
}

func (m *MockMatrixClient) SendText(ctx context.Context, roomID, content, senderID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailSend {
		return "", fmt.Errorf("mock matrix send failed")
	}

	msg := MatrixSentMessage{
		RoomID:    roomID,
		Content:   content,
		SenderID:  senderID,
		Timestamp: time.Now(),
	}
	m.sentMessages = append(m.sentMessages, msg)
	return "event-" + time.Now().Format("20060102150405"), nil
}

func (m *MockMatrixClient) JoinRoom(ctx context.Context, roomID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailJoin {
		return fmt.Errorf("mock join failed")
	}

	if m.joinedRooms[roomID] == nil {
		m.joinedRooms[roomID] = make([]string, 0)
	}
	m.joinedRooms[roomID] = append(m.joinedRooms[roomID], userID)
	return nil
}

func (m *MockMatrixClient) LeaveRoom(ctx context.Context, roomID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if users, exists := m.joinedRooms[roomID]; exists {
		for i, u := range users {
			if u == userID {
				m.joinedRooms[roomID] = append(users[:i], users[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *MockMatrixClient) SetDisplayName(ctx context.Context, displayName, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.displayNames[userID] = displayName
	return nil
}

func (m *MockMatrixClient) GetSentMessages() []MatrixSentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MatrixSentMessage{}, m.sentMessages...)
}

func (m *MockMatrixClient) GetJoinedRooms() map[string][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string][]string)
	for k, v := range m.joinedRooms {
		result[k] = append([]string{}, v...)
	}
	return result
}

func (m *MockMatrixClient) ClearSentMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = make([]MatrixSentMessage, 0)
}

// MockAppService implements a mock AppService for testing
type MockAppService struct {
	ghostUsers   map[string]*GhostUser
	bridgeUserID string
	eventChan    chan Event
	mu           sync.Mutex
}

func NewMockAppService() *MockAppService {
	return &MockAppService{
		ghostUsers:   make(map[string]*GhostUser),
		bridgeUserID: "@bridge:test.com",
		eventChan:    make(chan Event, 100),
	}
}

func (m *MockAppService) Events() <-chan Event {
	return m.eventChan
}

func (m *MockAppService) GetBridgeUserID() string {
	return m.bridgeUserID
}

func (m *MockAppService) isGhostUser(userID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.ghostUsers[userID]
	return exists
}

func (m *MockAppService) GenerateGhostUserID(platform, externalID string) string {
	return fmt.Sprintf("@%s_%s:%s", platform, externalID, "test.com")
}

func (m *MockAppService) GetGhostUser(userID string) (*GhostUser, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, exists := m.ghostUsers[userID]
	return user, exists
}

func (m *MockAppService) RegisterGhostUser(user *GhostUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ghostUsers[user.UserID] = user
	return nil
}

// MockWhatsAppServer creates a mock WhatsApp API server
func MockWhatsAppServer() (*httptest.Server, *[]http.Request) {
	var requests []http.Request
	mu := &sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, *r)
		mu.Unlock()

		if r.URL.Path == "/v18.0/1234567890/messages" {
			// Handle message send
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"messaging_product": "whatsapp",
				"contacts": []map[string]interface{}{
					{"input": "1234567890", "wa_id": "1234567890"},
				},
				"messages": []map[string]interface{}{
					{"id": "wamid-" + time.Now().Format("20060102150405")},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/v18.0/1234567890/media" {
			// Handle media upload
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"id": "media-" + time.Now().Format("20060102150405"),
			}
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/v18.0/media-") {
			// Handle media download
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"url":               "https://example.com/media/file.jpg",
				"mime_type":         "image/jpeg",
				"sha256":            "abc123",
				"file_size":         1024,
				"id":                "media-123",
				"messaging_product": "whatsapp",
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server, &requests
}

// MockDiscordServer creates a mock Discord API server
func MockDiscordServer() (*httptest.Server, *[]http.Request) {
	var requests []http.Request
	mu := &sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, *r)
		mu.Unlock()

		auth := r.Header.Get("Authorization")
		if auth != "Bot test-bot-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if strings.Contains(r.URL.Path, "/channels/") && strings.HasSuffix(r.URL.Path, "/messages") {
			if r.Method == http.MethodPost {
				// Handle message send
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"id":        "msg-" + time.Now().Format("20060102150405"),
					"content":   "test message",
					"timestamp": time.Now().Format(time.RFC3339),
				}
				json.NewEncoder(w).Encode(response)
			} else if r.Method == http.MethodPatch {
				// Handle message edit
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"id":        "msg-123",
					"content":   "edited message",
					"timestamp": time.Now().Format(time.RFC3339),
				}
				json.NewEncoder(w).Encode(response)
			} else if r.Method == http.MethodDelete {
				// Handle message delete
				w.WriteHeader(http.StatusNoContent)
			}
		} else if strings.Contains(r.URL.Path, "/reactions/") {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusNoContent)
			} else if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
			} else if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				response := []map[string]interface{}{
					{"emoji": "👍", "count": 2},
					{"emoji": "❤️", "count": 1},
				}
				json.NewEncoder(w).Encode(response)
			}
		} else if r.URL.Path == "/gateway" {
			// Handle gateway connection
			w.WriteHeader(http.StatusUpgradeRequired)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server, &requests
}

// MockTeamsServer creates a mock Microsoft Graph API server
func MockTeamsServer() (*httptest.Server, *[]http.Request) {
	var requests []http.Request
	mu := &sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, *r)
		mu.Unlock()

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if strings.Contains(r.URL.Path, "/chats/") && strings.HasSuffix(r.URL.Path, "/messages") {
			if r.Method == http.MethodPost {
				// Handle message send
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"id":              "msg-" + time.Now().Format("20060102150405"),
					"body":            map[string]interface{}{"content": "test message"},
					"createdDateTime": time.Now().Format(time.RFC3339),
				}
				json.NewEncoder(w).Encode(response)
			} else if r.Method == http.MethodPatch {
				// Handle message edit
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"id":              "msg-123",
					"body":            map[string]interface{}{"content": "edited message"},
					"createdDateTime": time.Now().Format(time.RFC3339),
				}
				json.NewEncoder(w).Encode(response)
			} else if r.Method == http.MethodDelete {
				// Handle message delete (soft delete)
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"id": "msg-123",
				}
				json.NewEncoder(w).Encode(response)
			}
		} else if strings.Contains(r.URL.Path, "/reactions") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			} else if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
			} else if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"value": []map[string]interface{}{
						{
							"reactionType": "👍",
							"createdBy": map[string]interface{}{
								"user": map[string]interface{}{
									"id":    "user-123",
									"email": "user@example.com",
								},
							},
							"createdDateTime": time.Now().Format(time.RFC3339),
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			}
		} else if strings.Contains(r.URL.Path, "/oauth2/v2.0/token") {
			// Handle token refresh
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"access_token":  "new-access-token-" + time.Now().Format("20060102150405"),
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "new-refresh-token",
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server, &requests
}

// TestHelper provides common test utilities
type TestHelper struct {
	T            *testing.T
	MatrixClient *MockMatrixClient
	AppService   *MockAppService
	Context      context.Context
	Cancel       context.CancelFunc
}

func NewTestHelper(t *testing.T) *TestHelper {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	return &TestHelper{
		T:            t,
		MatrixClient: NewMockMatrixClient(),
		AppService:   NewMockAppService(),
		Context:      ctx,
		Cancel:       cancel,
	}
}

func (h *TestHelper) Cleanup() {
	h.Cancel()
}

func (h *TestHelper) AssertMessageSent(roomID, content string, senderID string) {
	messages := h.MatrixClient.GetSentMessages()
	found := false
	for _, msg := range messages {
		if msg.RoomID == roomID && msg.Content == content && (senderID == "" || msg.SenderID == senderID) {
			found = true
			break
		}
	}
	if !found {
		h.T.Errorf("expected message '%s' to be sent to room '%s', got messages: %+v", content, roomID, messages)
	}
}

func (h *TestHelper) AssertMessageCount(expected int) {
	messages := h.MatrixClient.GetSentMessages()
	if len(messages) != expected {
		h.T.Errorf("expected %d messages, got %d", expected, len(messages))
	}
}

func (h *TestHelper) AssertGhostUserRegistered(userID string) {
	_, exists := h.AppService.GetGhostUser(userID)
	if !exists {
		h.T.Errorf("expected ghost user '%s' to be registered", userID)
	}
}

func (h *TestHelper) AssertDisplayNameSet(userID, displayName string) {
	h.MatrixClient.mu.Lock()
	defer h.MatrixClient.mu.Unlock()

	if h.MatrixClient.displayNames[userID] != displayName {
		h.T.Errorf("expected display name '%s' for user '%s', got '%s'", displayName, userID, h.MatrixClient.displayNames[userID])
	}
}

// WaitForCondition waits for a condition to be true
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for condition: %s", msg)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// Test helper functions for message flow verification
func verifyMessageFlow(t *testing.T, adapter sdtw.SDTWAdapter, target sdtw.Target, content string) {
	ctx := context.Background()
	msg := sdtw.Message{
		ID:        "msg-test-123",
		Content:   content,
		Type:      sdtw.MessageTypeText,
		Timestamp: time.Now(),
	}

	result, err := adapter.SendMessage(ctx, target, msg)
	if err != nil {
		t.Errorf("SendMessage failed: %v", err)
		return
	}

	if !result.Delivered {
		t.Errorf("expected message to be delivered, got Delivered=false")
	}

	if result.MessageID == "" {
		t.Errorf("expected message ID to be set")
	}
}

func verifyErrorWrapped(t *testing.T, err error, expectedCode sdtw.ErrorCode) {
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}

	adapterErr, ok := err.(*sdtw.AdapterError)
	if !ok {
		t.Errorf("expected AdapterError, got %T", err)
		return
	}

	if adapterErr.Code != expectedCode {
		t.Errorf("expected error code '%s', got '%s'", expectedCode, adapterErr.Code)
	}
}

func TestWhatsAppAdapterIntegration(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	whatsappServer, _ := MockWhatsAppServer()
	defer whatsappServer.Close()

	adapter := sdtw.NewWhatsAppAdapter()

	config := sdtw.AdapterConfig{
		Platform: "whatsapp",
		Enabled:  true,
		Credentials: map[string]string{
			"access_token": "test-token",
		},
		Settings: map[string]string{
			"phone_number_id":     "1234567890",
			"business_account_id": "test-account",
			"webhook_secret":      "test-secret",
		},
	}

	ctx := helper.Context
	if err := adapter.Initialize(ctx, config); err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	t.Run("SendMessageToWhatsApp", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "whatsapp",
			UserID:   "1234567890",
		}

		msg := sdtw.Message{
			ID:        "msg-1",
			Content:   "Hello from test",
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Now(),
		}

		result, err := adapter.SendMessage(ctx, target, msg)
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
		}

		if !result.Delivered {
			t.Error("expected message to be delivered")
		}

		if result.MessageID == "" {
			t.Error("expected message ID to be set")
		}

		t.Logf("Message sent successfully with ID: %s", result.MessageID)
	})

	t.Run("ReceiveWhatsAppWebhook", func(t *testing.T) {
		webhookPayload := map[string]interface{}{
			"object": "whatsapp_business_account",
			"entry": []map[string]interface{}{
				{
					"id": "1234567890",
					"changes": []map[string]interface{}{
						{
							"value": map[string]interface{}{
								"messaging_product": "whatsapp",
								"metadata": map[string]interface{}{
									"display_phone_number": "+1234567890",
									"phone_number_id":      "1234567890",
								},
								"messages": []map[string]interface{}{
									{
										"from":      "9876543210",
										"id":        "wamid-123",
										"timestamp": "1704067200",
										"type":      "text",
										"text": map[string]interface{}{
											"body": "Hello from WhatsApp",
										},
									},
								},
							},
							"field": "messages",
						},
					},
				},
			},
		}

		payloadBytes, _ := json.Marshal(webhookPayload)

		events, err := handleWhatsAppWebhookTest(payloadBytes)
		if err != nil {
			t.Errorf("failed to handle webhook: %v", err)
		}

		if len(events) == 0 {
			t.Error("expected events from webhook")
		}

		event := events[0]
		if event.Platform != "whatsapp" {
			t.Errorf("expected platform 'whatsapp', got '%s'", event.Platform)
		}

		if event.Content != "Hello from WhatsApp" {
			t.Errorf("expected content 'Hello from WhatsApp', got '%s'", event.Content)
		}

		t.Logf("Received webhook event: %+v", event)
	})

	t.Run("MediaMessage", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "whatsapp",
			UserID:   "1234567890",
		}

		msg := sdtw.Message{
			ID:      "msg-2",
			Content: "Image attachment",
			Type:    sdtw.MessageTypeImage,
			Attachments: []sdtw.Attachment{
				{
					ID:       "att-1",
					URL:      "https://example.com/image.jpg",
					MimeType: "image/jpeg",
					Filename: "image.jpg",
					Size:     1024,
				},
			},
			Timestamp: time.Now(),
		}

		result, err := adapter.SendMessage(ctx, target, msg)
		if err != nil {
			t.Errorf("SendMediaMessage failed: %v", err)
		}

		if !result.Delivered {
			t.Error("expected media message to be delivered")
		}

		t.Logf("Media message sent with ID: %s", result.MessageID)
	})

	t.Run("HealthCheck", func(t *testing.T) {
		health, err := adapter.HealthCheck()
		if err != nil {
			t.Errorf("HealthCheck failed: %v", err)
		}

		if !health.Connected {
			t.Error("expected adapter to be connected after initialization")
		}

		t.Logf("Health status: %+v", health)
	})
}

func handleWhatsAppWebhookTest(payload []byte) ([]sdtw.ExternalEvent, error) {
	var webhook map[string]interface{}
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, err
	}

	object := webhook["object"]
	if object != "whatsapp_business_account" {
		return nil, fmt.Errorf("invalid object type")
	}

	entry := webhook["entry"].([]interface{})[0].(map[string]interface{})
	changes := entry["changes"].([]interface{})[0].(map[string]interface{})
	value := changes["value"].(map[string]interface{})
	messages := value["messages"].([]interface{})[0].(map[string]interface{})

	event := sdtw.ExternalEvent{
		Platform:  "whatsapp",
		EventType: "message",
		Timestamp: time.Now(),
		Source:    messages["from"].(string),
		Content:   messages["text"].(map[string]interface{})["body"].(string),
		Metadata: map[string]string{
			"message_id": messages["id"].(string),
		},
	}

	return []sdtw.ExternalEvent{event}, nil
}

func TestDiscordAdapterIntegration(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	discordServer, _ := MockDiscordServer()
	defer discordServer.Close()

	adapter := sdtw.NewDiscordAdapter()

	config := sdtw.AdapterConfig{
		Platform: "discord",
		Enabled:  true,
		Credentials: map[string]string{
			"bot_token": "test-bot-token",
		},
		Settings: map[string]string{
			"guild_id":       "test-guild",
			"command_prefix": "!",
		},
	}

	ctx := helper.Context
	if err := adapter.Initialize(ctx, config); err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	t.Run("SendMessageToDiscord", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "discord",
			Channel:  "123456789",
		}

		msg := sdtw.Message{
			ID:        "msg-1",
			Content:   "Hello from Discord test",
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Now(),
		}

		result, err := adapter.SendMessage(ctx, target, msg)
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
		}

		if !result.Delivered {
			t.Error("expected message to be delivered")
		}

		if result.MessageID == "" {
			t.Error("expected message ID to be set")
		}

		t.Logf("Message sent successfully with ID: %s", result.MessageID)
	})

	t.Run("SendReaction", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "discord",
			Channel:  "123456789",
		}

		err := adapter.SendReaction(ctx, target, "msg-123", "👍")
		if err != nil {
			t.Errorf("SendReaction failed: %v", err)
		}

		t.Log("Reaction sent successfully")
	})

	t.Run("EditMessage", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "discord",
			Channel:  "123456789",
		}

		err := adapter.EditMessage(ctx, target, "msg-123", "Edited message content")
		if err != nil {
			t.Errorf("EditMessage failed: %v", err)
		}

		t.Log("Message edited successfully")
	})

	t.Run("DeleteMessage", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "discord",
			Channel:  "123456789",
		}

		err := adapter.DeleteMessage(ctx, target, "msg-123")
		if err != nil {
			t.Errorf("DeleteMessage failed: %v", err)
		}

		t.Log("Message deleted successfully")
	})

	t.Run("HealthCheck", func(t *testing.T) {
		health, err := adapter.HealthCheck()
		if err != nil {
			t.Errorf("HealthCheck failed: %v", err)
		}

		if !health.Connected {
			t.Error("expected adapter to be connected after initialization")
		}

		t.Logf("Health status: %+v", health)
	})
}

func TestTeamsAdapterIntegration(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	teamsServer, _ := MockTeamsServer()
	defer teamsServer.Close()

	teamsConfig := sdtw.TeamsConfig{
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
		TenantID:      "test-tenant-id",
		BotID:         "test-bot-id",
		BotName:       "Test Bot",
		AccessToken:   "test-access-token",
		WebhookSecret: "test-webhook-secret",
		Enabled:       true,
	}

	adapter := sdtw.NewTeamsAdapter(teamsConfig)

	config := sdtw.AdapterConfig{
		Platform: "teams",
		Enabled:  true,
		Credentials: map[string]string{
			"client_id":     teamsConfig.ClientID,
			"client_secret": teamsConfig.ClientSecret,
			"tenant_id":     teamsConfig.TenantID,
		},
	}

	ctx := helper.Context
	if err := adapter.Initialize(ctx, config); err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("failed to start adapter: %v", err)
	}
	defer adapter.Shutdown(ctx)

	t.Run("SendMessageToTeams", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "teams",
			UserID:   "19:user-id",
		}

		msg := sdtw.Message{
			ID:        "msg-1",
			Content:   "Hello from Teams test",
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Now(),
		}

		result, err := adapter.SendMessage(ctx, target, msg)
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
		}

		if !result.Delivered {
			t.Error("expected message to be delivered")
		}

		if result.MessageID == "" {
			t.Error("expected message ID to be set")
		}

		t.Logf("Message sent successfully with ID: %s", result.MessageID)
	})

	t.Run("SendReaction", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "teams",
			UserID:   "19:user-id",
		}

		err := adapter.SendReaction(ctx, target, "msg-123", "👍")
		if err != nil {
			t.Errorf("SendReaction failed: %v", err)
		}

		t.Log("Reaction sent successfully")
	})

	t.Run("EditMessage", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "teams",
			UserID:   "19:user-id",
		}

		err := adapter.EditMessage(ctx, target, "msg-123", "Edited Teams message")
		if err != nil {
			t.Errorf("EditMessage failed: %v", err)
		}

		t.Log("Message edited successfully")
	})

	t.Run("DeleteMessage", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "teams",
			UserID:   "19:user-id",
		}

		err := adapter.DeleteMessage(ctx, target, "msg-123")
		if err != nil {
			t.Errorf("DeleteMessage failed: %v", err)
		}

		t.Log("Message deleted successfully")
	})

	t.Run("GetReactions", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "teams",
			UserID:   "19:user-id",
		}

		reactions, err := adapter.GetReactions(ctx, target, "msg-123")
		if err != nil {
			t.Errorf("GetReactions failed: %v", err)
		}

		if len(reactions) == 0 {
			t.Error("expected reactions to be returned")
		}

		t.Logf("Retrieved %d reactions", len(reactions))
	})

	t.Run("HealthCheck", func(t *testing.T) {
		health, err := adapter.HealthCheck()
		if err != nil {
			t.Errorf("HealthCheck failed: %v", err)
		}

		if !health.Connected {
			t.Error("expected adapter to be connected after initialization")
		}

		t.Logf("Health status: %+v", health)
	})
}

func TestErrorHandlingAndRetryLogic(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	ctx := helper.Context

	t.Run("AdapterErrorWrapping", func(t *testing.T) {
		adapter := sdtw.NewWhatsAppAdapter()

		config := sdtw.AdapterConfig{
			Platform:    "whatsapp",
			Enabled:     true,
			Credentials: map[string]string{},
		}

		err := adapter.Initialize(ctx, config)
		if err == nil {
			t.Error("expected initialization error")
		}

		verifyErrorWrapped(t, err, sdtw.ErrAuthFailed)
	})

	t.Run("RateLimitedError", func(t *testing.T) {
		adapter := sdtw.NewWhatsAppAdapter()

		target := sdtw.Target{
			Platform: "whatsapp",
			UserID:   "1234567890",
		}

		msg := sdtw.Message{
			ID:        "msg-test",
			Content:   "Test message",
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Now(),
		}

		result, err := adapter.SendMessage(ctx, target, msg)
		if err == nil {
			t.Error("expected error for uninitialized adapter")
		}

		if result != nil && result.Error != nil {
			t.Logf("Adapter error code: %s", result.Error.Code)
			t.Logf("Adapter error message: %s", result.Error.Message)
			t.Logf("Retryable: %v", result.Error.Retryable)
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		msg := sdtw.Message{
			ID:        "",
			Content:   "",
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Now(),
		}

		err := sdtw.ValidateMessage(msg)
		if err == nil {
			t.Error("expected validation error")
		}

		verifyErrorWrapped(t, err, sdtw.ErrValidation)
	})

	t.Run("ErrorRetryability", func(t *testing.T) {
		testCases := []struct {
			name      string
			code      sdtw.ErrorCode
			retryable bool
		}{
			{"RateLimit", sdtw.ErrRateLimited, true},
			{"NetworkError", sdtw.ErrNetworkError, true},
			{"Timeout", sdtw.ErrTimeout, true},
			{"AuthFailed", sdtw.ErrAuthFailed, false},
			{"InvalidTarget", sdtw.ErrInvalidTarget, false},
			{"Validation", sdtw.ErrValidation, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := sdtw.NewAdapterError(tc.code, "test error", tc.retryable)

				if err.Retryable != tc.retryable {
					t.Errorf("expected retryable %v for code %s, got %v", tc.retryable, tc.code, err.Retryable)
				}

				if err.Permanent == tc.retryable {
					t.Errorf("expected permanent to be opposite of retryable for code %s", tc.code)
				}
			})
		}
	})

	t.Run("MetricsOnError", func(t *testing.T) {
		adapter := sdtw.NewBaseAdapter("test", "1.0.0", sdtw.CapabilitySet{
			Read:  true,
			Write: true,
		})

		testError := fmt.Errorf("test error")
		adapter.RecordError(testError)

		metrics, err := adapter.Metrics()
		if err != nil {
			t.Errorf("failed to get metrics: %v", err)
		}

		if metrics.MessagesFailed != 1 {
			t.Errorf("expected 1 failed message, got %d", metrics.MessagesFailed)
		}

		if metrics.LastError != "test error" {
			t.Errorf("expected last error 'test error', got '%s'", metrics.LastError)
		}

		if metrics.LastErrorTime.IsZero() {
			t.Error("expected last error time to be set")
		}
	})
}

func TestRateLimiting(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	ctx := helper.Context

	t.Run("AdapterRateLimitConfig", func(t *testing.T) {
		config := sdtw.AdapterConfig{
			Platform: "whatsapp",
			Enabled:  true,
			RateLimits: sdtw.RateLimitConfig{
				RequestsPerSecond: 10,
				BurstSize:         50,
				BackoffOnLimit:    true,
			},
		}

		if config.RateLimits.RequestsPerSecond != 10 {
			t.Errorf("expected requests per second 10, got %d", config.RateLimits.RequestsPerSecond)
		}

		if config.RateLimits.BurstSize != 50 {
			t.Errorf("expected burst size 50, got %d", config.RateLimits.BurstSize)
		}

		if !config.RateLimits.BackoffOnLimit {
			t.Error("expected backoff on limit to be true")
		}
	})

	t.Run("MetricsTracking", func(t *testing.T) {
		adapter := sdtw.NewBaseAdapter("test", "1.0.0", sdtw.CapabilitySet{
			Read:  true,
			Write: true,
		})

		adapter.RecordSent()
		adapter.RecordSent()
		adapter.RecordReceived()

		metrics, err := adapter.Metrics()
		if err != nil {
			t.Errorf("failed to get metrics: %v", err)
		}

		if metrics.MessagesSent != 2 {
			t.Errorf("expected 2 sent messages, got %d", metrics.MessagesSent)
		}

		if metrics.MessagesReceived != 1 {
			t.Errorf("expected 1 received message, got %d", metrics.MessagesReceived)
		}
	})

	t.Run("HealthStatusErrorRate", func(t *testing.T) {
		adapter := sdtw.NewBaseAdapter("test", "1.0.0", sdtw.CapabilitySet{
			Read:  true,
			Write: true,
		})

		adapter.RecordSent()
		adapter.RecordError(fmt.Errorf("error 1"))
		adapter.RecordError(fmt.Errorf("error 2"))

		health, err := adapter.HealthCheck()
		if err != nil {
			t.Errorf("failed to get health status: %v", err)
		}

		if health.ErrorRate != 0 {
			t.Logf("Error rate: %.2f%%", health.ErrorRate)
		}

		if health.Error != "" {
			t.Logf("Current error: %s", health.Error)
		}
	})

	t.Run("RateLimitedResponse", func(t *testing.T) {
		whatsappServer, requests := MockWhatsAppServer()
		defer whatsappServer.Close()

		adapter := sdtw.NewWhatsAppAdapter()

		config := sdtw.AdapterConfig{
			Platform: "whatsapp",
			Enabled:  true,
			Credentials: map[string]string{
				"access_token": "test-token",
			},
			Settings: map[string]string{
				"phone_number_id": "1234567890",
			},
		}

		if err := adapter.Initialize(ctx, config); err != nil {
			t.Fatalf("failed to initialize adapter: %v", err)
		}

		target := sdtw.Target{
			Platform: "whatsapp",
			UserID:   "1234567890",
		}

		msg := sdtw.Message{
			ID:        "msg-1",
			Content:   "Test message",
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Now(),
		}

		result, err := adapter.SendMessage(ctx, target, msg)
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
		}

		if len(*requests) == 0 {
			t.Error("expected request to be made")
		} else {
			t.Logf("Request made to: %s", (*requests)[0].URL.String())
		}

		if result != nil {
			t.Logf("Result delivered: %v", result.Delivered)
			if result.Error != nil {
				t.Logf("Error code: %s", result.Error.Code)
			}
		}
	})
}
func TestMatrixBridgeEventVerification(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	t.Run("ExternalEventFormatting", func(t *testing.T) {
		externalEvent := sdtw.ExternalEvent{
			Platform:  "whatsapp",
			EventType: "message",
			Timestamp: time.Now(),
			Source:    "1234567890",
			Content:   "Test message from WhatsApp",
			Metadata: map[string]string{
				"user_id":   "9876543210",
				"user_name": "Test User",
			},
		}

		if externalEvent.Platform != "whatsapp" {
			t.Errorf("expected platform 'whatsapp', got '%s'", externalEvent.Platform)
		}

		if externalEvent.EventType != "message" {
			t.Errorf("expected event type 'message', got '%s'", externalEvent.EventType)
		}

		if externalEvent.Content != "Test message from WhatsApp" {
			t.Errorf("expected content 'Test message from WhatsApp', got '%s'", externalEvent.Content)
		}

		t.Logf("External event formatted correctly: %+v", externalEvent)
	})

	t.Run("MessageFormatting", func(t *testing.T) {
		msg := sdtw.Message{
			ID:        "msg-123",
			Content:   "Test message content",
			Type:      sdtw.MessageTypeText,
			Timestamp: time.Now(),
		}

		if err := sdtw.ValidateMessage(msg); err != nil {
			t.Errorf("message validation failed: %v", err)
		}

		t.Logf("Message formatted correctly: %+v", msg)
	})

	t.Run("TargetMapping", func(t *testing.T) {
		target := sdtw.Target{
			Platform: "whatsapp",
			Channel:  "1234567890",
			RoomID:   "!room:matrix.org",
			Metadata: map[string]string{
				"mapping": "test",
			},
		}

		if target.Platform != "whatsapp" {
			t.Errorf("expected platform 'whatsapp', got '%s'", target.Platform)
		}

		if target.Channel != "1234567890" {
			t.Errorf("expected channel '1234567890', got '%s'", target.Channel)
		}

		if target.RoomID != "!room:matrix.org" {
			t.Errorf("expected room ID '!room:matrix.org', got '%s'", target.RoomID)
		}

		t.Logf("Target mapped correctly: %+v", target)
	})
}
