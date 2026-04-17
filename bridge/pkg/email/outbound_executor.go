package email

import (
	"context"
	"fmt"
	"net/mail"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
)

type OutboundExecutor struct {
	senders  map[string]EmailSender
	masker   *pii.Masker
	approval *EmailApprovalManager
	log      *logger.Logger
}

type OutboundExecutorConfig struct {
	Senders  map[string]EmailSender
	Approval *EmailApprovalManager
	Log      *logger.Logger
}

func NewOutboundExecutor(cfg OutboundExecutorConfig) *OutboundExecutor {
	return &OutboundExecutor{
		senders:  cfg.Senders,
		masker:   pii.NewMasker(),
		approval: cfg.Approval,
		log:      cfg.Log,
	}
}

func (e *OutboundExecutor) Execute(ctx context.Context, req *OutboundRequest) (*OutboundResult, error) {
	if _, err := mail.ParseAddress(req.To); err != nil {
		return &OutboundResult{Status: ApprovalRejected, Error: "invalid recipient"}, fmt.Errorf("invalid recipient: %w", err)
	}
	if _, err := mail.ParseAddress(req.From); err != nil {
		return &OutboundResult{Status: ApprovalRejected, Error: "invalid sender"}, fmt.Errorf("invalid sender: %w", err)
	}

	if e.approval != nil && len(req.PIIFields) > 0 {
		decision, err := e.approval.RequestApproval(ctx, req)
		if err != nil {
			return &OutboundResult{Status: ApprovalRejected, Error: err.Error()}, err
		}
		if !decision.Approved {
			return &OutboundResult{Status: ApprovalRejected, Error: "approval denied"}, nil
		}
	}

	resolvedBody := e.masker.ResolvePlaceholders(req.BodyText, req.PIIResolutions)
	resolvedSubject := e.masker.ResolvePlaceholders(req.Subject, req.PIIResolutions)

	provider := req.Provider
	if provider == "" {
		provider = "gmail"
	}

	sender, ok := e.senders[provider]
	if !ok {
		fallback, ok2 := e.senders["smtp"]
		if !ok2 {
			return &OutboundResult{Status: ApprovalRejected, Error: "no sender available"}, fmt.Errorf("no sender for provider %s", provider)
		}
		sender = fallback
		provider = "smtp"
	}

	messageID, err := sender.Send(ctx, req.To, resolvedSubject, resolvedBody, "")
	if err != nil {
		return &OutboundResult{
			Status:       ApprovalRejected,
			ProviderUsed: provider,
			Error:        err.Error(),
		}, err
	}

	now := time.Now()
	return &OutboundResult{
		Success:      true,
		MessageID:    messageID,
		ProviderUsed: provider,
		ApprovedAt:   &now,
		Status:       ApprovalApproved,
	}, nil
}

func (e *OutboundExecutor) AvailableProviders() []string {
	var providers []string
	for name := range e.senders {
		providers = append(providers, name)
	}
	return providers
}

type OutboundRequest struct {
	To             string            `json:"to"`
	From           string            `json:"from"`
	Subject        string            `json:"subject"`
	BodyText       string            `json:"body_text"`
	BodyHTML       string            `json:"body_html,omitempty"`
	Provider       string            `json:"provider"`
	PIIFields      []string          `json:"pii_fields,omitempty"`
	PIIResolutions map[string]string `json:"pii_resolutions,omitempty"`
	EmailID        string            `json:"email_id"`
}

var _ = fmt.Sprintf
