package email

import (
	"encoding/json"
	"time"

	"github.com/armorclaw/bridge/pkg/eventbus"
)

type EmailReceivedEvent struct {
	eventbus.BaseEvent
	From        string            `json:"from"`
	To          string            `json:"to"`
	Subject     string            `json:"subject"`
	BodyMasked  string            `json:"body_masked"`
	FileIDs     []string          `json:"file_ids"`
	PIIFields   []string          `json:"pii_fields"`
	EmailID     string            `json:"email_id"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
}

func NewEmailReceivedEvent(from, to, subject, bodyMasked, emailID string, fileIDs, piiFields []string) *EmailReceivedEvent {
	return &EmailReceivedEvent{
		BaseEvent: eventbus.BaseEvent{
			Type: eventbus.EventTypeEmailReceived,
			Ts:   time.Now(),
		},
		From:       from,
		To:         to,
		Subject:    subject,
		BodyMasked: bodyMasked,
		EmailID:    emailID,
		FileIDs:    fileIDs,
		PIIFields:  piiFields,
	}
}

func (e *EmailReceivedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

var _ eventbus.BridgeEvent = (*EmailReceivedEvent)(nil)

type TeamRoutedEmailEvent struct {
	*EmailReceivedEvent
	TeamID   string   `json:"team_id"`
	AgentIDs []string `json:"agent_ids"`
}
