package browser

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/agent"
)

// mockStatusEmitter captures emitted status events for testing
type mockStatusEmitter struct {
	mu     sync.Mutex
	events []agent.StatusEvent
}

func (m *mockStatusEmitter) EmitStatus(ctx context.Context, event agent.StatusEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockStatusEmitter) getEvents() []agent.StatusEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]agent.StatusEvent{}, m.events...)
}

// TestNewBrowserSkill tests browser skill creation
func TestNewBrowserSkill(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("test-agent", mock)

	if skill.agentID != "test-agent" {
		t.Errorf("expected agent_id test-agent, got %s", skill.agentID)
	}

	session := skill.GetSession()
	if session == nil {
		t.Error("expected session to be initialized")
	}
	if session.Status != BrowserStatusIdle {
		t.Errorf("expected initial status %s, got %s", BrowserStatusIdle, session.Status)
	}
}

// TestNavigate tests navigation with status emission
func TestNavigate(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("nav-agent", mock)

	ctx := context.Background()
	err := skill.Navigate(ctx, "https://example.com/login")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}

	events := mock.getEvents()
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}

	// First event should be browsing/navigating
	found := false
	for _, e := range events {
		if e.Status == agent.StatusBrowsing && e.Metadata.Step == "navigating" {
			found = true
			if e.Metadata.URL != "https://example.com/login" {
				t.Errorf("expected URL in metadata, got %s", e.Metadata.URL)
			}
			break
		}
	}
	if !found {
		t.Error("expected navigating event to be emitted")
	}

	// Check session was updated
	session := skill.GetSession()
	if session.URL != "https://example.com/login" {
		t.Errorf("expected session URL, got %s", session.URL)
	}
}

// TestNavigateInvalidURL tests navigation with invalid URL
func TestNavigateInvalidURL(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("invalid-agent", mock)

	ctx := context.Background()
	err := skill.Navigate(ctx, "not a valid url ://")
	if err == nil {
		t.Error("expected error for invalid URL")
	}

	// Should have emitted error status
	events := mock.getEvents()
	found := false
	for _, e := range events {
		if e.Status == agent.StatusError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error event for invalid URL")
	}
}

// TestWaitForElement tests element waiting with status emission
func TestWaitForElement(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("wait-agent", mock)

	ctx := context.Background()
	err := skill.WaitForElement(ctx, "#submit-button", 5*time.Second)
	if err != nil {
		t.Fatalf("failed to wait for element: %v", err)
	}

	events := mock.getEvents()
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}

	// Should have waiting_element event
	found := false
	for _, e := range events {
		if e.Metadata.Step == "waiting_element" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected waiting_element event")
	}
}

// TestFillForm tests form filling with status emission
func TestFillForm(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("form-agent", mock)

	ctx := context.Background()
	err := skill.FillForm(ctx, "#email", "test@example.com")
	if err != nil {
		t.Fatalf("failed to fill form: %v", err)
	}

	events := mock.getEvents()
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}

	// Should have filling_field and field_filled events
	foundFilling := false
	foundFilled := false
	for _, e := range events {
		if e.Metadata.Step == "filling_field" {
			foundFilling = true
		}
		if e.Metadata.Step == "field_filled" {
			foundFilled = true
		}
	}
	if !foundFilling {
		t.Error("expected filling_field event")
	}
	if !foundFilled {
		t.Error("expected field_filled event")
	}
}

// TestClick tests clicking with status emission
func TestClick(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("click-agent", mock)

	ctx := context.Background()
	err := skill.Click(ctx, "#submit-button")
	if err != nil {
		t.Fatalf("failed to click: %v", err)
	}

	events := mock.getEvents()
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}
}

// TestWaitForCaptcha tests CAPTCHA waiting
func TestWaitForCaptcha(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("captcha-agent", mock)

	// Set a URL first
	skill.session.URL = "https://example.com/protected"

	ctx := context.Background()
	err := skill.WaitForCaptcha(ctx)
	if err != nil {
		t.Fatalf("failed to wait for captcha: %v", err)
	}

	events := mock.getEvents()
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}

	// Should have awaiting_captcha status
	found := false
	for _, e := range events {
		if e.Status == agent.StatusAwaitingCaptcha {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected awaiting_captcha event")
	}
}

// TestWaitFor2FA tests 2FA waiting
func TestWaitFor2FA(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("2fa-agent", mock)

	skill.session.URL = "https://bank.example.com/verify"

	ctx := context.Background()
	err := skill.WaitFor2FA(ctx)
	if err != nil {
		t.Fatalf("failed to wait for 2FA: %v", err)
	}

	events := mock.getEvents()
	found := false
	for _, e := range events {
		if e.Status == agent.StatusAwaiting2FA {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected awaiting_2fa event")
	}
}

// TestComplete tests task completion
func TestComplete(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("complete-agent", mock)

	skill.session.URL = "https://example.com/success"

	ctx := context.Background()
	skill.Complete(ctx)

	events := mock.getEvents()
	found := false
	for _, e := range events {
		if e.Status == agent.StatusComplete {
			found = true
			if e.Metadata.Progress != 100 {
				t.Errorf("expected progress 100 on complete, got %d", e.Metadata.Progress)
			}
			break
		}
	}
	if !found {
		t.Error("expected complete event")
	}
}

// TestFail tests task failure
func TestFail(t *testing.T) {
	mock := &mockStatusEmitter{}
	skill := NewBrowserSkill("fail-agent", mock)

	ctx := context.Background()
	skill.Fail(ctx, fmt.Errorf("test failure message"))

	// Session should be in error state
	session := skill.GetSession()
	if session.Status != BrowserStatusError {
		t.Errorf("expected error status, got %s", session.Status)
	}

	// Should have emitted error status
	events := mock.getEvents()
	found := false
	for _, e := range events {
		if e.Status == agent.StatusError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error event")
	}
}

// TestNilEmitter tests that nil emitter doesn't panic
func TestNilEmitter(t *testing.T) {
	skill := NewBrowserSkill("nil-agent", nil)

	ctx := context.Background()

	// These should not panic
	_ = skill.Navigate(ctx, "https://example.com")
	_ = skill.FillForm(ctx, "#field", "value")
	_ = skill.Click(ctx, "#button")
	skill.Complete(ctx)
}

// TestGetSession tests session retrieval
func TestGetSession(t *testing.T) {
	skill := NewBrowserSkill("session-agent", nil)

	// Modify session
	skill.session.URL = "https://test.example.com"
	skill.session.Status = BrowserStatusReady

	// Get session
	session := skill.GetSession()

	if session.URL != "https://test.example.com" {
		t.Errorf("expected URL, got %s", session.URL)
	}
	if session.Status != BrowserStatusReady {
		t.Errorf("expected status %s, got %s", BrowserStatusReady, session.Status)
	}
}
