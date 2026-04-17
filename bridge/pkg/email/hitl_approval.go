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
		m.log.Warn("approval_timeout", "approval_id", approvalID, "email_id", req.EmailID)
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
