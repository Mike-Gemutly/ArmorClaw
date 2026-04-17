// Package proto provides generated message types for the email gRPC service.
// These types mirror the email.proto definitions.
package proto

// IngestEmailRequest is the raw email from Postfix pipe(8) transport
type IngestEmailRequest struct {
	RawEmail     []byte `json:"raw_email"`
	EnvelopeFrom string `json:"envelope_from"`
	EnvelopeTo   string `json:"envelope_to"`
	QueueID      string `json:"queue_id"`
}

// IngestEmailResponse confirms email was accepted or rejected
type IngestEmailResponse struct {
	Accepted        bool     `json:"accepted"`
	FileIDs         []string `json:"file_ids"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
	EmailID         string   `json:"email_id"`
}

// EmailSendRequest is an outbound email to be sent
type EmailSendRequest struct {
	To          string        `json:"to"`
	Subject     string        `json:"subject"`
	BodyText    string        `json:"body_text"`
	BodyHTML    string        `json:"body_html,omitempty"`
	From        string        `json:"from"`
	ReplyTo     string        `json:"reply_to,omitempty"`
	Provider    string        `json:"provider"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

// EmailSendResponse confirms the email was sent
type EmailSendResponse struct {
	Sent         bool   `json:"sent"`
	MessageID    string `json:"message_id,omitempty"`
	ProviderUsed string `json:"provider_used,omitempty"`
	Error        string `json:"error,omitempty"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id,omitempty"`
}
