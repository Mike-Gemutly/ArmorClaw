package email

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type mockSender struct {
	lastTo       string
	lastSubject  string
	lastBody     string
	lastHTML     string
	lastAttCount int
	sendErr      error
	messageID    string
}

func (m *mockSender) Send(_ context.Context, to, subject, bodyText, bodyHTML string, attachments ...*EmailAttachment) (string, error) {
	m.lastTo = to
	m.lastSubject = subject
	m.lastBody = bodyText
	m.lastHTML = bodyHTML
	m.lastAttCount = len(attachments)
	return m.messageID, m.sendErr
}

func (m *mockSender) Provider() string { return "mock" }

func TestDraftManager_SaveAndRetrieve(t *testing.T) {
	sender := &mockSender{messageID: "msg-001"}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	draft := EmailDraft{
		TeamID:   "team-1",
		To:       "alice@example.com",
		Subject:  "Hello",
		BodyText: "World",
	}

	id, err := dm.SaveDraft(context.Background(), draft)
	if err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty draft ID")
	}

	drafts, err := dm.ListDrafts(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected 1 draft, got %d", len(drafts))
	}
	if drafts[0].ID != id {
		t.Errorf("draft ID mismatch: got %s, want %s", drafts[0].ID, id)
	}
	if drafts[0].To != "alice@example.com" {
		t.Errorf("To mismatch: got %s", drafts[0].To)
	}
	if drafts[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestDraftManager_UpdateDraft(t *testing.T) {
	sender := &mockSender{messageID: "msg-002"}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	draft := EmailDraft{
		TeamID:   "team-1",
		To:       "bob@example.com",
		Subject:  "Old Subject",
		BodyText: "Old Body",
	}
	id, _ := dm.SaveDraft(context.Background(), draft)

	updated := EmailDraft{
		TeamID:   "team-1",
		To:       "bob@example.com",
		Subject:  "New Subject",
		BodyText: "New Body",
	}
	err := dm.UpdateDraft(context.Background(), id, updated)
	if err != nil {
		t.Fatalf("UpdateDraft: %v", err)
	}

	drafts, _ := dm.ListDrafts(context.Background(), "team-1")
	if len(drafts) != 1 {
		t.Fatalf("expected 1 draft, got %d", len(drafts))
	}
	if drafts[0].Subject != "New Subject" {
		t.Errorf("subject not updated: got %s", drafts[0].Subject)
	}
	if drafts[0].BodyText != "New Body" {
		t.Errorf("body not updated: got %s", drafts[0].BodyText)
	}
}

func TestDraftManager_UpdateNonExistent(t *testing.T) {
	sender := &mockSender{}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	err := dm.UpdateDraft(context.Background(), "nonexistent", EmailDraft{})
	if err == nil {
		t.Error("expected error updating nonexistent draft")
	}
}

func TestDraftManager_SendDraft(t *testing.T) {
	sender := &mockSender{messageID: "msg-sent-1"}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	draft := EmailDraft{
		TeamID:   "team-1",
		To:       "carol@example.com",
		Subject:  "Send me",
		BodyText: "Payload",
		BodyHTML: "<p>Payload</p>",
		Attachments: []*EmailAttachment{
			{Filename: "doc.pdf", FileID: "f1"},
		},
	}
	id, _ := dm.SaveDraft(context.Background(), draft)

	msgID, err := dm.SendDraft(context.Background(), id)
	if err != nil {
		t.Fatalf("SendDraft: %v", err)
	}
	if msgID != "msg-sent-1" {
		t.Errorf("message ID: got %s, want msg-sent-1", msgID)
	}
	if sender.lastTo != "carol@example.com" {
		t.Errorf("sender To: got %s", sender.lastTo)
	}
	if sender.lastSubject != "Send me" {
		t.Errorf("sender Subject: got %s", sender.lastSubject)
	}
	if sender.lastAttCount != 1 {
		t.Errorf("attachments: got %d, want 1", sender.lastAttCount)
	}

	drafts, _ := dm.ListDrafts(context.Background(), "team-1")
	if len(drafts) != 0 {
		t.Errorf("draft should be removed after send, got %d", len(drafts))
	}
}

func TestDraftManager_SendDraftNonExistent(t *testing.T) {
	sender := &mockSender{}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	_, err := dm.SendDraft(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error sending nonexistent draft")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDraftManager_SendDraftFailureReinserts(t *testing.T) {
	sender := &mockSender{sendErr: fmt.Errorf("smtp down")}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	draft := EmailDraft{
		TeamID:  "team-1",
		To:      "dave@example.com",
		Subject: "Retry",
	}
	id, _ := dm.SaveDraft(context.Background(), draft)

	_, err := dm.SendDraft(context.Background(), id)
	if err == nil {
		t.Error("expected send error")
	}

	drafts, _ := dm.ListDrafts(context.Background(), "team-1")
	if len(drafts) != 1 {
		t.Errorf("draft should be re-inserted on send failure, got %d", len(drafts))
	}
}

func TestDraftManager_DeleteDraft(t *testing.T) {
	sender := &mockSender{}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	draft := EmailDraft{TeamID: "team-1", To: "eve@example.com"}
	id, _ := dm.SaveDraft(context.Background(), draft)

	err := dm.DeleteDraft(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}

	drafts, _ := dm.ListDrafts(context.Background(), "team-1")
	if len(drafts) != 0 {
		t.Errorf("expected 0 drafts after delete, got %d", len(drafts))
	}
}

func TestDraftManager_DeleteNonExistent(t *testing.T) {
	sender := &mockSender{}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	err := dm.DeleteDraft(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent draft")
	}
}

func TestDraftManager_ListDraftsFilterByTeam(t *testing.T) {
	sender := &mockSender{}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	for _, d := range []EmailDraft{
		{TeamID: "team-A", To: "a@example.com", Subject: "A1"},
		{TeamID: "team-B", To: "b@example.com", Subject: "B1"},
		{TeamID: "team-A", To: "a2@example.com", Subject: "A2"},
	} {
		dm.SaveDraft(context.Background(), d)
	}

	teamA, _ := dm.ListDrafts(context.Background(), "team-A")
	if len(teamA) != 2 {
		t.Errorf("team-A: expected 2 drafts, got %d", len(teamA))
	}

	teamB, _ := dm.ListDrafts(context.Background(), "team-B")
	if len(teamB) != 1 {
		t.Errorf("team-B: expected 1 draft, got %d", len(teamB))
	}

	teamC, _ := dm.ListDrafts(context.Background(), "team-C")
	if len(teamC) != 0 {
		t.Errorf("team-C: expected 0 drafts, got %d", len(teamC))
	}
}

func TestDraftManager_FullLifecycle(t *testing.T) {
	sender := &mockSender{messageID: "msg-lifecycle"}
	dm := NewDraftManager(DraftManagerConfig{Sender: sender})

	draft := EmailDraft{
		TeamID:   "team-lc",
		To:       "lc@example.com",
		Subject:  "Draft",
		BodyText: "Body",
	}

	// Save
	id, err := dm.SaveDraft(context.Background(), draft)
	if err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	// Update
	err = dm.UpdateDraft(context.Background(), id, EmailDraft{
		TeamID:   "team-lc",
		To:       "lc@example.com",
		Subject:  "Updated Draft",
		BodyText: "Updated Body",
	})
	if err != nil {
		t.Fatalf("UpdateDraft: %v", err)
	}

	// List
	drafts, _ := dm.ListDrafts(context.Background(), "team-lc")
	if len(drafts) != 1 || drafts[0].Subject != "Updated Draft" {
		t.Fatalf("list after update: %+v", drafts)
	}

	// Send
	msgID, err := dm.SendDraft(context.Background(), id)
	if err != nil {
		t.Fatalf("SendDraft: %v", err)
	}
	if msgID != "msg-lifecycle" {
		t.Errorf("msgID: got %s", msgID)
	}

	// Verify removed
	drafts, _ = dm.ListDrafts(context.Background(), "team-lc")
	if len(drafts) != 0 {
		t.Errorf("expected 0 after send, got %d", len(drafts))
	}
}
