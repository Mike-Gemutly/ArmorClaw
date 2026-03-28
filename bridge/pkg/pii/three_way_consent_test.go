package pii

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type MockMatrixAdapter struct {
	mu              sync.RWMutex
	CreatedRooms    map[string]*MockRoom
	Invites         map[string][]string
	SentMessages    map[string][]string
	SentEvents      map[string][]MockEvent
	UserID          string
	CreateRoomError error
}

type MockRoom struct {
	RoomID     string
	RoomAlias  string
	Name       string
	Topic      string
	InviteList []string
	CreatedAt  time.Time
}

type MockEvent struct {
	RoomID    string
	EventType string
	Content   []byte
}

func NewMockMatrixAdapter() *MockMatrixAdapter {
	return &MockMatrixAdapter{
		UserID:       "@bridge:server.example.com",
		CreatedRooms: make(map[string]*MockRoom),
		Invites:      make(map[string][]string),
		SentMessages: make(map[string][]string),
		SentEvents:   make(map[string][]MockEvent),
	}
}

func (m *MockMatrixAdapter) CreateRoom(ctx context.Context, params map[string]interface{}) (roomID, roomAlias string, err error) {
	if m.CreateRoomError != nil {
		return "", "", m.CreateRoomError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	roomID = "!mock_room_" + time.Now().Format("20060102150405.000000") + ":server.example.com"
	if alias, ok := params["room_alias_name"].(string); ok {
		roomAlias = "#" + alias + ":server.example.com"
	}

	room := &MockRoom{
		RoomID:    roomID,
		RoomAlias: roomAlias,
		CreatedAt: time.Now(),
	}

	if name, ok := params["name"].(string); ok {
		room.Name = name
	}
	if topic, ok := params["topic"].(string); ok {
		room.Topic = topic
	}
	if invites, ok := params["invite"].([]string); ok {
		room.InviteList = invites
		for _, userID := range invites {
			m.Invites[roomID] = append(m.Invites[roomID], userID)
		}
	}

	m.CreatedRooms[roomID] = room
	return roomID, roomAlias, nil
}

func (m *MockMatrixAdapter) InviteUser(ctx context.Context, roomID, userID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Invites[roomID] = append(m.Invites[roomID], userID)
	return nil
}

func (m *MockMatrixAdapter) SendMessage(roomID, message, msgType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentMessages[roomID] = append(m.SentMessages[roomID], message)
	return "$event_" + time.Now().Format("20060102150405"), nil
}

func (m *MockMatrixAdapter) SendEvent(roomID, eventType string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentEvents[roomID] = append(m.SentEvents[roomID], MockEvent{
		RoomID:    roomID,
		EventType: eventType,
		Content:   content,
	})
	return nil
}

func (m *MockMatrixAdapter) GetUserID() string {
	return m.UserID
}

func (m *MockMatrixAdapter) GetInvites(roomID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Invites[roomID]
}

func (m *MockMatrixAdapter) GetCreatedRoom(roomID string) *MockRoom {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CreatedRooms[roomID]
}

func TestThreeWayConsentManager_RequestConsent(t *testing.T) {
	mockMatrix := NewMockMatrixAdapter()
	hitlManager := NewHITLConsentManager(HITLConfig{})

	manager, err := NewThreeWayConsentManager(ThreeWayConfig{
		HITLManager: hitlManager,
		Matrix:      mockMatrix,
		Timeout:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to create ThreeWayConsentManager: %v", err)
	}

	request := &AccessRequest{
		ID:              "request-123",
		SkillID:         "skill-456",
		SkillName:       "Test Skill",
		ProfileID:       "profile-789",
		RequestedFields: []string{"full_name", "email"},
		RequiredFields:  []string{"full_name"},
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}

	userMatrixID := "@patient:server.example.com"
	agentMatrixID := "@doctor:server.example.com"

	consentRoom, err := manager.CreateConsentRoom(context.Background(), request, userMatrixID, agentMatrixID)
	if err != nil {
		t.Fatalf("CreateConsentRoom failed: %v", err)
	}

	if consentRoom == nil {
		t.Fatal("ConsentRoom should not be nil")
	}
	if consentRoom.MatrixRoomID == "" {
		t.Error("MatrixRoomID should not be empty")
	}
	if consentRoom.State != ConsentStatePending {
		t.Errorf("Expected state 'pending', got '%s'", consentRoom.State)
	}
	if consentRoom.UserMatrixID != userMatrixID {
		t.Errorf("Expected UserMatrixID '%s', got '%s'", userMatrixID, consentRoom.UserMatrixID)
	}
	if consentRoom.AgentMatrixID != agentMatrixID {
		t.Errorf("Expected AgentMatrixID '%s', got '%s'", agentMatrixID, consentRoom.AgentMatrixID)
	}
	if consentRoom.BridgeMatrixID != mockMatrix.UserID {
		t.Errorf("Expected BridgeMatrixID '%s', got '%s'", mockMatrix.UserID, consentRoom.BridgeMatrixID)
	}
	if consentRoom.ApprovalToken == "" {
		t.Error("ApprovalToken should not be empty")
	}
}

func TestThreeWayConsentManager_CorrectUsersInvited(t *testing.T) {
	mockMatrix := NewMockMatrixAdapter()
	hitlManager := NewHITLConsentManager(HITLConfig{})

	manager, err := NewThreeWayConsentManager(ThreeWayConfig{
		HITLManager: hitlManager,
		Matrix:      mockMatrix,
		Timeout:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to create ThreeWayConsentManager: %v", err)
	}

	request := &AccessRequest{
		ID:              "request-invite-test",
		SkillID:         "skill-123",
		SkillName:       "Invite Test Skill",
		ProfileID:       "profile-123",
		RequestedFields: []string{"ssn"},
		RequiredFields:  []string{"ssn"},
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}

	userMatrixID := "@subject:server.example.com"
	agentMatrixID := "@requester:server.example.com"

	consentRoom, err := manager.CreateConsentRoom(context.Background(), request, userMatrixID, agentMatrixID)
	if err != nil {
		t.Fatalf("CreateConsentRoom failed: %v", err)
	}

	invites := mockMatrix.GetInvites(consentRoom.MatrixRoomID)

	expectedInvites := map[string]bool{
		userMatrixID:  false,
		agentMatrixID: false,
	}

	for _, invite := range invites {
		if _, exists := expectedInvites[invite]; exists {
			expectedInvites[invite] = true
		}
	}

	for userID, invited := range expectedInvites {
		if !invited {
			t.Errorf("Expected user '%s' to be invited, but they were not", userID)
		}
	}

	if len(invites) < 2 {
		t.Errorf("Expected at least 2 invites (user and agent), got %d", len(invites))
	}
}

func TestThreeWayConsentManager_RoomNotRecreated(t *testing.T) {
	mockMatrix := NewMockMatrixAdapter()
	hitlManager := NewHITLConsentManager(HITLConfig{})

	manager, err := NewThreeWayConsentManager(ThreeWayConfig{
		HITLManager: hitlManager,
		Matrix:      mockMatrix,
		Timeout:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to create ThreeWayConsentManager: %v", err)
	}

	request := &AccessRequest{
		ID:              "request-duplicate",
		SkillID:         "skill-dup",
		SkillName:       "Duplicate Test",
		ProfileID:       "profile-dup",
		RequestedFields: []string{"name"},
		RequiredFields:  []string{"name"},
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}

	userMatrixID := "@user:server.example.com"
	agentMatrixID := "@agent:server.example.com"

	consentRoom1, err := manager.CreateConsentRoom(context.Background(), request, userMatrixID, agentMatrixID)
	if err != nil {
		t.Fatalf("First CreateConsentRoom failed: %v", err)
	}

	roomsBefore := len(mockMatrix.CreatedRooms)

	consentRoom2, err := manager.CreateConsentRoom(context.Background(), request, userMatrixID, agentMatrixID)
	if err == nil {
		t.Error("Expected error when creating duplicate room, got nil")
	}
	if consentRoom2 != nil {
		t.Error("Expected nil room on duplicate creation")
	}
	if !errors.Is(err, ErrConsentRoomExists) {
		t.Errorf("Expected ErrConsentRoomExists, got: %v", err)
	}

	roomsAfter := len(mockMatrix.CreatedRooms)
	if roomsAfter != roomsBefore {
		t.Errorf("No new room should be created: had %d rooms, now have %d", roomsBefore, roomsAfter)
	}

	_ = consentRoom1
}

func TestThreeWayConsentManager_InvalidApprovalToken(t *testing.T) {
	mockMatrix := NewMockMatrixAdapter()
	hitlManager := NewHITLConsentManager(HITLConfig{})

	manager, err := NewThreeWayConsentManager(ThreeWayConfig{
		HITLManager: hitlManager,
		Matrix:      mockMatrix,
		Timeout:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to create ThreeWayConsentManager: %v", err)
	}

	request := &AccessRequest{
		ID:              "request-invalid-token",
		SkillID:         "skill-invalid",
		SkillName:       "Invalid Token Test",
		ProfileID:       "profile-invalid",
		RequestedFields: []string{"phone"},
		RequiredFields:  []string{"phone"},
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}

	userMatrixID := "@user:server.example.com"
	agentMatrixID := "@agent:server.example.com"

	consentRoom, err := manager.CreateConsentRoom(context.Background(), request, userMatrixID, agentMatrixID)
	if err != nil {
		t.Fatalf("CreateConsentRoom failed: %v", err)
	}

	err = manager.HandleApprovalCommand(context.Background(), consentRoom.MatrixRoomID, userMatrixID, "invalid-token-12345", []string{"phone"})
	if err == nil {
		t.Error("Expected error with invalid token, got nil")
	}
	if !errors.Is(err, ErrInvalidApprovalToken) {
		t.Errorf("Expected ErrInvalidApprovalToken, got: %v", err)
	}
}

func TestThreeWayConsentManager_NotAuthorized(t *testing.T) {
	mockMatrix := NewMockMatrixAdapter()
	hitlManager := NewHITLConsentManager(HITLConfig{})

	manager, err := NewThreeWayConsentManager(ThreeWayConfig{
		HITLManager: hitlManager,
		Matrix:      mockMatrix,
		Timeout:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to create ThreeWayConsentManager: %v", err)
	}

	request := &AccessRequest{
		ID:              "request-auth-test",
		SkillID:         "skill-auth",
		SkillName:       "Auth Test",
		ProfileID:       "profile-auth",
		RequestedFields: []string{"ssn"},
		RequiredFields:  []string{"ssn"},
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}

	userMatrixID := "@authorized:server.example.com"
	agentMatrixID := "@agent:server.example.com"

	consentRoom, err := manager.CreateConsentRoom(context.Background(), request, userMatrixID, agentMatrixID)
	if err != nil {
		t.Fatalf("CreateConsentRoom failed: %v", err)
	}

	unauthorizedUser := "@unauthorized:server.example.com"
	err = manager.HandleApprovalCommand(context.Background(), consentRoom.MatrixRoomID, unauthorizedUser, consentRoom.ApprovalToken, []string{"ssn"})
	if err == nil {
		t.Error("Expected error from unauthorized user, got nil")
	}
	if !errors.Is(err, ErrNotAuthorized) {
		t.Errorf("Expected ErrNotAuthorized, got: %v", err)
	}
}

func TestThreeWayConsentManager_RoomNotFound(t *testing.T) {
	mockMatrix := NewMockMatrixAdapter()
	hitlManager := NewHITLConsentManager(HITLConfig{})

	manager, err := NewThreeWayConsentManager(ThreeWayConfig{
		HITLManager: hitlManager,
		Matrix:      mockMatrix,
		Timeout:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to create ThreeWayConsentManager: %v", err)
	}

	_, err = manager.GetConsentRoomByMatrixID("!nonexistent:server.example.com")
	if err == nil {
		t.Error("Expected error for nonexistent room, got nil")
	}
	if !errors.Is(err, ErrRoomNotFound) {
		t.Errorf("Expected ErrRoomNotFound, got: %v", err)
	}
}

func TestThreeWayConsentManager_ListPendingRooms(t *testing.T) {
	mockMatrix := NewMockMatrixAdapter()
	hitlManager := NewHITLConsentManager(HITLConfig{})

	manager, err := NewThreeWayConsentManager(ThreeWayConfig{
		HITLManager: hitlManager,
		Matrix:      mockMatrix,
		Timeout:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to create ThreeWayConsentManager: %v", err)
	}

	pending := manager.ListPendingRooms()
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending rooms, got %d", len(pending))
	}

	request := &AccessRequest{
		ID:              "request-pending",
		SkillID:         "skill-pending",
		SkillName:       "Pending Test",
		ProfileID:       "profile-pending",
		RequestedFields: []string{"name"},
		RequiredFields:  []string{"name"},
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}

	_, err = manager.CreateConsentRoom(context.Background(), request, "@user:server.example.com", "@agent:server.example.com")
	if err != nil {
		t.Fatalf("CreateConsentRoom failed: %v", err)
	}

	pending = manager.ListPendingRooms()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending room, got %d", len(pending))
	}
}

func TestThreeWayConsentManager_ConsentRoomIsExpired(t *testing.T) {
	expiredRoom := &ConsentRoom{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if !expiredRoom.IsExpired() {
		t.Error("Room with past expiry should be expired")
	}

	validRoom := &ConsentRoom{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if validRoom.IsExpired() {
		t.Error("Room with future expiry should not be expired")
	}
}

func TestThreeWayConsentManager_ConsentRoomIsPending(t *testing.T) {
	pendingRoom := &ConsentRoom{
		State: ConsentStatePending,
	}
	if !pendingRoom.IsPending() {
		t.Error("Room with pending state should be pending")
	}

	approvedRoom := &ConsentRoom{
		State: ConsentStateApproved,
	}
	if approvedRoom.IsPending() {
		t.Error("Room with approved state should not be pending")
	}
}
