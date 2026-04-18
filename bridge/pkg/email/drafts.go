package email

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

// EmailDraft represents an unsent email stored for later editing or sending.
type EmailDraft struct {
	ID          string             `json:"id"`
	TeamID      string             `json:"team_id"`
	To          string             `json:"to"`
	CC          string             `json:"cc,omitempty"`
	Subject     string             `json:"subject"`
	BodyText    string             `json:"body_text"`
	BodyHTML    string             `json:"body_html,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Attachments []*EmailAttachment `json:"attachments,omitempty"`
}

// DraftManager stores and manages email drafts in memory.
type DraftManager struct {
	drafts map[string]*EmailDraft // keyed by draft ID
	sender EmailSender
	log    *logger.Logger
	mu     sync.RWMutex
}

// DraftManagerConfig holds the dependencies for constructing a DraftManager.
type DraftManagerConfig struct {
	Sender EmailSender
	Log    *logger.Logger
}

// NewDraftManager creates a DraftManager ready to use.
func NewDraftManager(cfg DraftManagerConfig) *DraftManager {
	return &DraftManager{
		drafts: make(map[string]*EmailDraft),
		sender: cfg.Sender,
		log:    cfg.Log,
	}
}

// SaveDraft assigns a new UUID to the draft, stores it, and returns the ID.
func (dm *DraftManager) SaveDraft(_ context.Context, draft EmailDraft) (string, error) {
	id, err := generateID()
	if err != nil {
		return "", fmt.Errorf("draft id generation: %w", err)
	}

	now := time.Now()
	draft.ID = id
	draft.CreatedAt = now
	draft.UpdatedAt = now

	dm.mu.Lock()
	dm.drafts[id] = &draft
	dm.mu.Unlock()

	if dm.log != nil {
		dm.log.Info("draft_saved", "draft_id", id, "team_id", draft.TeamID, "to", draft.To)
	}
	return id, nil
}

// UpdateDraft overwrites an existing draft identified by draftID.
func (dm *DraftManager) UpdateDraft(_ context.Context, draftID string, draft EmailDraft) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	existing, ok := dm.drafts[draftID]
	if !ok {
		return fmt.Errorf("draft not found: %s", draftID)
	}

	draft.ID = draftID
	draft.CreatedAt = existing.CreatedAt
	draft.UpdatedAt = time.Now()
	dm.drafts[draftID] = &draft

	if dm.log != nil {
		dm.log.Info("draft_updated", "draft_id", draftID)
	}
	return nil
}

// SendDraft sends the draft through the injected EmailSender and removes it
// from the store. Returns the message ID from the sender.
func (dm *DraftManager) SendDraft(ctx context.Context, draftID string) (string, error) {
	dm.mu.Lock()
	draft, ok := dm.drafts[draftID]
	if !ok {
		dm.mu.Unlock()
		return "", fmt.Errorf("draft not found: %s", draftID)
	}
	delete(dm.drafts, draftID)
	dm.mu.Unlock()

	attachments := draft.Attachments
	messageID, err := dm.sender.Send(ctx, draft.To, draft.Subject, draft.BodyText, draft.BodyHTML, attachments...)
	if err != nil {
		// Re-insert on send failure so the user can retry.
		dm.mu.Lock()
		dm.drafts[draftID] = draft
		dm.mu.Unlock()
		return "", fmt.Errorf("send draft %s: %w", draftID, err)
	}

	if dm.log != nil {
		dm.log.Info("draft_sent", "draft_id", draftID, "message_id", messageID, "to", draft.To)
	}
	return messageID, nil
}

// ListDrafts returns all drafts belonging to the given teamID.
func (dm *DraftManager) ListDrafts(_ context.Context, teamID string) ([]EmailDraft, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var result []EmailDraft
	for _, d := range dm.drafts {
		if strings.EqualFold(d.TeamID, teamID) {
			result = append(result, *d)
		}
	}
	return result, nil
}

// DeleteDraft removes a draft by ID.
func (dm *DraftManager) DeleteDraft(_ context.Context, draftID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, ok := dm.drafts[draftID]; !ok {
		return fmt.Errorf("draft not found: %s", draftID)
	}
	delete(dm.drafts, draftID)

	if dm.log != nil {
		dm.log.Info("draft_deleted", "draft_id", draftID)
	}
	return nil
}

// generateID creates a random 16-byte hex string using crypto/rand.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
