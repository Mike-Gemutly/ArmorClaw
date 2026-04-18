package interfaces

import "context"

// ConsentResult captures the outcome of a consent request.
type ConsentResult struct {
	Approved       bool     `json:"approved"`
	ApprovedFields []string `json:"approved_fields,omitempty"`
	DeniedFields   []string `json:"denied_fields,omitempty"`
	Error          error    `json:"-"`
}

// ConsentProvider requests human approval for deferred actions.
// Implementations may use Matrix HITL, email approval, or other channels.
type ConsentProvider interface {
	// RequestConsent asks for human approval. Returns a channel that will
	// receive exactly one ConsentResult when the human responds or timeout occurs.
	// The caller MUST select on the channel with a 300s timeout.
	RequestConsent(ctx context.Context, requestID, reason string, fields []string) (<-chan ConsentResult, error)
}
