package studio

import (
	"context"
	"strings"
	"testing"
	"time"
)

//=============================================================================
// Mock Matrix Adapter
//=============================================================================

type mockMatrixAdapter struct {
	sentMessages []string
	lastRoomID   string
}

func (m *mockMatrixAdapter) SendMessage(ctx context.Context, roomID, message string) error {
	m.sentMessages = append(m.sentMessages, message)
	m.lastRoomID = roomID
	return nil
}

func (m *mockMatrixAdapter) SendFormattedMessage(ctx context.Context, roomID, plainBody, formattedBody string) error {
	m.sentMessages = append(m.sentMessages, plainBody)
	m.lastRoomID = roomID
	return nil
}

func (m *mockMatrixAdapter) ReplyToEvent(ctx context.Context, roomID, eventID, message string) error {
	m.sentMessages = append(m.sentMessages, message)
	m.lastRoomID = roomID
	return nil
}

//=============================================================================
// Command Parsing Tests
//=============================================================================

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input       string
		prefix      string
		shouldParse bool
		expected    *Command
	}{
		{
			input:       "!agent list-skills",
			prefix:      "!",
			shouldParse: true,
			expected: &Command{
				Prefix:  "!",
				Name:    "agent",
				Args:    []string{"list-skills"},
				RawText: "!agent list-skills",
			},
		},
		{
			input:       "!agent create MyAgent",
			prefix:      "!",
			shouldParse: true,
			expected: &Command{
				Prefix:  "!",
				Name:    "agent",
				Args:    []string{"create", "MyAgent"},
				RawText: "!agent create MyAgent",
			},
		},
		{
			input:       "!agent spawn agent-123 Process contracts",
			prefix:      "!",
			shouldParse: true,
			expected: &Command{
				Prefix:  "!",
				Name:    "agent",
				Args:    []string{"spawn", "agent-123", "Process", "contracts"},
				RawText: "!agent spawn agent-123 Process contracts",
			},
		},
		{
			input:       "not a command",
			prefix:      "!",
			shouldParse: false,
			expected:    nil,
		},
		{
			input:       "!other command",
			prefix:      "!",
			shouldParse: true,
			expected: &Command{
				Prefix:  "!",
				Name:    "other",
				Args:    []string{"command"},
				RawText: "!other command",
			},
		},
	}

	for _, test := range tests {
		result, ok := ParseCommand(test.input, test.prefix)
		if !test.shouldParse {
			if ok {
				t.Errorf("expected no parse for input %q, got %+v", test.input, result)
			}
		} else {
			if !ok {
				t.Errorf("expected command for input %q, got no parse", test.input)
				continue
			}
			if result.Name != test.expected.Name {
				t.Errorf("expected name %q, got %q", test.expected.Name, result.Name)
			}
			if result.Prefix != test.expected.Prefix {
				t.Errorf("expected prefix %q, got %q", test.expected.Prefix, result.Prefix)
			}
			if len(result.Args) != len(test.expected.Args) {
				t.Errorf("expected %d args, got %d", len(test.expected.Args), len(result.Args))
			}
		}
	}
}

//=============================================================================
// Command Handler Tests (CGO-free)
//=============================================================================

func newTestCommandHandler(t *testing.T) (*SQLiteStore, *CommandHandler, *mockMatrixAdapter) {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	mockMatrix := &mockMatrixAdapter{}
	handler := NewCommandHandler(CommandHandlerConfig{
		Store:         store,
		Matrix:        mockMatrix,
		CommandPrefix: "!",
		WizardTimeout: 5 * time.Minute,
	})

	return store, handler, mockMatrix
}

func TestCommandHandler_ListSkills(t *testing.T) {
	store, handler, mockMatrix := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent list-skills")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if !handled {
		t.Error("expected command to be handled")
	}

	if len(mockMatrix.sentMessages) == 0 {
		t.Error("expected message to be sent")
	}

	// Check that the message contains skill information
	if !strings.Contains(mockMatrix.sentMessages[0], "browser_navigate") {
		t.Error("expected message to contain browser_navigate skill")
	}
}

func TestCommandHandler_ListPII(t *testing.T) {
	store, handler, mockMatrix := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent list-pii")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if !handled {
		t.Error("expected command to be handled")
	}

	if len(mockMatrix.sentMessages) == 0 {
		t.Error("expected message to be sent")
	}

	// Check that the message contains PII field information
	if !strings.Contains(mockMatrix.sentMessages[0], "client_name") {
		t.Error("expected message to contain client_name PII field")
	}
}

func TestCommandHandler_ListProfiles(t *testing.T) {
	store, handler, mockMatrix := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent list-profiles")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if !handled {
		t.Error("expected command to be handled")
	}

	if len(mockMatrix.sentMessages) == 0 {
		t.Error("expected message to be sent")
	}

	// Check that the message contains profile information
	if !strings.Contains(mockMatrix.sentMessages[0], "low") ||
		!strings.Contains(mockMatrix.sentMessages[0], "medium") ||
		!strings.Contains(mockMatrix.sentMessages[0], "high") {
		t.Error("expected message to contain all profile tiers")
	}
}

func TestCommandHandler_UnknownSubcommand(t *testing.T) {
	store, handler, _ := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent unknown-command")
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
	_ = handled // may be true or false depending on implementation
}

func TestCommandHandler_Create_InsufficientArgs(t *testing.T) {
	store, handler, _ := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent create")
	if err == nil {
		t.Error("expected error for missing agent name")
	}
	_ = handled
}

func TestCommandHandler_Spawn_InsufficientArgs(t *testing.T) {
	store, handler, _ := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent spawn")
	if err == nil {
		t.Error("expected error for missing agent ID")
	}
	_ = handled
}

func TestCommandHandler_Delete_InsufficientArgs(t *testing.T) {
	store, handler, _ := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent delete")
	if err == nil {
		t.Error("expected error for missing agent ID")
	}
	_ = handled
}

//=============================================================================
// Help Command Tests
//=============================================================================

func TestCommandHandler_Help(t *testing.T) {
	store, handler, mockMatrix := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent help")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if !handled {
		t.Error("expected command to be handled")
	}

	if len(mockMatrix.sentMessages) == 0 {
		t.Error("expected help message to be sent")
	}

	// Check that help message contains command information
	helpMsg := mockMatrix.sentMessages[0]
	expectedCommands := []string{"list-skills", "create", "spawn", "delete", "help"}
	for _, expected := range expectedCommands {
		if !strings.Contains(helpMsg, expected) {
			t.Errorf("expected help to contain command %q", expected)
		}
	}
}

//=============================================================================
// Stats Command Tests
//=============================================================================

func TestCommandHandler_Stats(t *testing.T) {
	store, handler, mockMatrix := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "!agent stats")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if !handled {
		t.Error("expected command to be handled")
	}

	if len(mockMatrix.sentMessages) == 0 {
		t.Error("expected stats message to be sent")
	}

	// Check that stats contains expected information
	statsMsg := mockMatrix.sentMessages[0]
	if !strings.Contains(statsMsg, "Definitions") && !strings.Contains(statsMsg, "definitions") {
		t.Error("expected stats to contain 'Definitions'")
	}
}

//=============================================================================
// Non-command Tests
//=============================================================================

func TestCommandHandler_NonCommand(t *testing.T) {
	store, handler, mockMatrix := newTestCommandHandler(t)
	defer store.Close()

	ctx := context.Background()

	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@test:example.com", "$event123", "hello world")
	if err != nil {
		t.Errorf("expected no error for non-command, got: %v", err)
	}
	if handled {
		t.Error("expected non-command to not be handled")
	}

	if len(mockMatrix.sentMessages) > 0 {
		t.Error("expected no message to be sent for non-command")
	}
}
