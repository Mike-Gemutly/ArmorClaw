package email

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/eventbus"
)

func TestNewEmailReceivedEvent(t *testing.T) {
	evt := NewEmailReceivedEvent(
		"sender@test.com",
		"recipient@test.com",
		"Test Subject",
		"Hello [PII_MASKED]",
		"email-001",
		[]string{"file-1", "file-2"},
		[]string{"ssn", "email"},
	)

	if evt.From != "sender@test.com" {
		t.Errorf("From = %q, want sender@test.com", evt.From)
	}
	if evt.To != "recipient@test.com" {
		t.Errorf("To = %q, want recipient@test.com", evt.To)
	}
	if evt.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want Test Subject", evt.Subject)
	}
	if evt.BodyMasked != "Hello [PII_MASKED]" {
		t.Errorf("BodyMasked = %q", evt.BodyMasked)
	}
	if evt.EmailID != "email-001" {
		t.Errorf("EmailID = %q, want email-001", evt.EmailID)
	}
	if len(evt.FileIDs) != 2 {
		t.Errorf("FileIDs len = %d, want 2", len(evt.FileIDs))
	}
	if len(evt.PIIFields) != 2 {
		t.Errorf("PIIFields len = %d, want 2", len(evt.PIIFields))
	}
}

func TestNewEmailReceivedEvent_EmptyFields(t *testing.T) {
	evt := NewEmailReceivedEvent("", "", "", "", "", nil, nil)

	if evt.From != "" {
		t.Errorf("From = %q, want empty", evt.From)
	}
	if evt.FileIDs != nil {
		t.Error("FileIDs should be nil")
	}
	if evt.PIIFields != nil {
		t.Error("PIIFields should be nil")
	}
}

func TestNewEmailReceivedEvent_SetsType(t *testing.T) {
	evt := NewEmailReceivedEvent("a@b.com", "c@d.com", "sub", "body", "id", nil, nil)
	if evt.Type != eventbus.EventTypeEmailReceived {
		t.Errorf("Type = %q, want %q", evt.Type, eventbus.EventTypeEmailReceived)
	}
}

func TestNewEmailReceivedEvent_SetsTimestamp(t *testing.T) {
	before := time.Now()
	evt := NewEmailReceivedEvent("a@b.com", "c@d.com", "sub", "body", "id", nil, nil)
	after := time.Now()

	if evt.Ts.Before(before) || evt.Ts.After(after) {
		t.Errorf("Ts = %v, expected between %v and %v", evt.Ts, before, after)
	}
}

func TestEmailReceivedEvent_ToJSON(t *testing.T) {
	evt := NewEmailReceivedEvent(
		"sender@test.com",
		"recipient@test.com",
		"Test",
		"masked body",
		"email-002",
		[]string{"f1"},
		[]string{"ssn"},
	)

	data, err := evt.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["from"] != "sender@test.com" {
		t.Errorf("from = %v", parsed["from"])
	}
	if parsed["to"] != "recipient@test.com" {
		t.Errorf("to = %v", parsed["to"])
	}
	if parsed["email_id"] != "email-002" {
		t.Errorf("email_id = %v", parsed["email_id"])
	}
	if parsed["type"] != "email.received" {
		t.Errorf("type = %v", parsed["type"])
	}
}

func TestEmailReceivedEvent_ToJSON_RoundTrip(t *testing.T) {
	original := NewEmailReceivedEvent(
		"a@b.com",
		"c@d.com",
		"Subject Line",
		"Body text",
		"eid-123",
		[]string{"f1", "f2"},
		[]string{"cc"},
	)

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var restored EmailReceivedEvent
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if restored.From != original.From {
		t.Errorf("From mismatch: %q vs %q", restored.From, original.From)
	}
	if restored.To != original.To {
		t.Errorf("To mismatch: %q vs %q", restored.To, original.To)
	}
	if restored.Subject != original.Subject {
		t.Errorf("Subject mismatch")
	}
	if restored.EmailID != original.EmailID {
		t.Errorf("EmailID mismatch")
	}
	if len(restored.FileIDs) != 2 {
		t.Errorf("FileIDs len = %d", len(restored.FileIDs))
	}
}

func TestEmailReceivedEvent_ImplementsBridgeEvent(t *testing.T) {
	var _ eventbus.BridgeEvent = (*EmailReceivedEvent)(nil)
}

func TestEmailReceivedEvent_EventTypeMethod(t *testing.T) {
	evt := NewEmailReceivedEvent("a@b.com", "c@d.com", "s", "b", "id", nil, nil)
	if evt.EventType() != eventbus.EventTypeEmailReceived {
		t.Errorf("EventType() = %q, want %q", evt.EventType(), eventbus.EventTypeEmailReceived)
	}
}

func TestEmailReceivedEvent_TimestampMethod(t *testing.T) {
	evt := NewEmailReceivedEvent("a@b.com", "c@d.com", "s", "b", "id", nil, nil)
	ts := evt.Timestamp()
	if ts.IsZero() {
		t.Error("Timestamp() should not be zero")
	}
}

func TestEmailReceivedEvent_JSONContainsAllFields(t *testing.T) {
	evt := NewEmailReceivedEvent("a@b.com", "c@d.com", "sub", "body", "eid", []string{"f1"}, []string{"ssn"})
	data, _ := evt.ToJSON()
	s := string(data)

	for _, field := range []string{`"from"`, `"to"`, `"subject"`, `"body_masked"`, `"email_id"`, `"file_ids"`, `"pii_fields"`, `"type"`, `"timestamp"`} {
		if !strings.Contains(s, field) {
			t.Errorf("JSON missing field %q", field)
		}
	}
}
