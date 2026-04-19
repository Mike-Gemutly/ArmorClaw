package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

const defaultApprovalTimeout = 300 * time.Second

type ApprovalDecision struct {
	Approved     bool
	ApprovedBy   string
	ApprovedAt   time.Time
	DeniedFields []string
	ApprovalID   string
}

type pendingApproval struct {
	request  *OutboundRequest
	resultCh chan *ApprovalDecision
	deadline time.Time
}

type EmailApprovalManager struct {
	mu            sync.RWMutex
	pending       map[string]*pendingApproval
	timeout       time.Duration
	log           *logger.Logger
	sendMatrixMsg func(roomID, eventType, body string) error
}

type EmailApprovalConfig struct {
	Timeout       time.Duration
	Log           *logger.Logger
	SendMatrixMsg func(roomID, eventType, body string) error
}

func NewEmailApprovalManager(cfg EmailApprovalConfig) *EmailApprovalManager {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultApprovalTimeout
	}
	return &EmailApprovalManager{
		pending:       make(map[string]*pendingApproval),
		timeout:       cfg.Timeout,
		log:           cfg.Log,
		sendMatrixMsg: cfg.SendMatrixMsg,
	}
}

func (m *EmailApprovalManager) RequestApproval(ctx context.Context, req *OutboundRequest) (*ApprovalDecision, error) {
	approvalID := fmt.Sprintf("approval_%d", time.Now().UnixMilli())

	resultCh := make(chan *ApprovalDecision, 1)
	pa := &pendingApproval{
		request:  req,
		resultCh: resultCh,
		deadline: time.Now().Add(m.timeout),
	}

	m.mu.Lock()
	m.pending[approvalID] = pa
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.pending, approvalID)
		m.mu.Unlock()
	}()

	if m.sendMatrixMsg != nil {
		body := fmt.Sprintf(`{"approval_id":"%s","email_id":"%s","to":"%s","subject":"[masked]","pii_fields":%d,"timeout_s":%d}`,
			approvalID, req.EmailID, req.To, len(req.PIIFields), int(m.timeout.Seconds()))
		_ = m.sendMatrixMsg("", "app.armorclaw.email_approval_request", body)
	}

	select {
	case decision := <-resultCh:
		return decision, nil
	case <-time.After(m.timeout):
		if m.log != nil {
			m.log.Warn("approval_timeout", "approval_id", approvalID, "email_id", req.EmailID)
		}
		return &ApprovalDecision{
			Approved:   false,
			ApprovalID: approvalID,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *EmailApprovalManager) HandleApprovalResponse(approvalID string, approved bool, approvedBy string) error {
	m.mu.RLock()
	pa, ok := m.pending[approvalID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("approval %s not found or expired", approvalID)
	}

	pa.resultCh <- &ApprovalDecision{
		Approved:   approved,
		ApprovedBy: approvedBy,
		ApprovedAt: time.Now(),
		ApprovalID: approvalID,
	}
	return nil
}

func (m *EmailApprovalManager) PendingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pending)
}

// PendingItem is a summary of a pending email approval for listing.
type PendingItem struct {
	ApprovalID  string `json:"approval_id"`
	EmailID     string `json:"email_id"`
	Sender      string `json:"sender"`
	To          string `json:"to"`
	Subject     string `json:"subject"`
	BodyPreview string `json:"body_preview"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// ListPending returns a slice of all pending email approval requests with detail.
func (m *EmailApprovalManager) ListPending() []PendingItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]PendingItem, 0, len(m.pending))
	for id, pa := range m.pending {
		preview := pa.request.BodyText
		if len(preview) > 200 {
			preview = preview[:200]
		}
		items = append(items, PendingItem{
			ApprovalID:  id,
			EmailID:     pa.request.EmailID,
			Sender:      pa.request.From,
			To:          pa.request.To,
			Subject:     pa.request.Subject,
			BodyPreview: preview,
			Status:      "pending",
			CreatedAt:   pa.deadline.Add(-m.timeout).Format(time.RFC3339),
		})
	}
	return items
}
