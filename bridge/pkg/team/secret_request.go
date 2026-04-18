package team

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

const defaultSecretRequestTimeout = 300 * time.Second

// SecretRequest represents a pending secret request from an agent.
type SecretRequest struct {
	RequestID      string
	AgentID        string
	TeamID         string
	CredentialName string
	TargetDomain   string
	Reason         string
	RiskClass      string
}

// SecretResponse represents the user's response to a secret request.
type SecretResponse struct {
	RequestID   string
	Approved    bool
	RespondedBy string
	SecretValue string // only set if approved
}

// pendingSecretRequest tracks an in-flight secret request and its response channel.
type pendingSecretRequest struct {
	request  *SecretRequest
	resultCh chan *SecretResponse
	deadline time.Time
}

// SecretRequestConfig holds constructor configuration for SecretRequestManager.
type SecretRequestConfig struct {
	Timeout       time.Duration
	Log           *logger.Logger
	SendMatrixMsg func(roomID, eventType, body string) error
}

// SecretRequestManager manages pending secret requests, publishing Matrix events
// and blocking until the user responds or the request times out.
type SecretRequestManager struct {
	mu            sync.RWMutex
	pending       map[string]*pendingSecretRequest
	timeout       time.Duration
	log           *logger.Logger
	sendMatrixMsg func(roomID, eventType, body string) error
}

// NewSecretRequestManager creates a new SecretRequestManager with the given config.
func NewSecretRequestManager(cfg SecretRequestConfig) *SecretRequestManager {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultSecretRequestTimeout
	}
	return &SecretRequestManager{
		pending:       make(map[string]*pendingSecretRequest),
		timeout:       cfg.Timeout,
		log:           cfg.Log,
		sendMatrixMsg: cfg.SendMatrixMsg,
	}
}

// RequestSecret publishes a Matrix event and blocks until the user responds,
// the timeout expires, or the context is cancelled.
func (m *SecretRequestManager) RequestSecret(ctx context.Context, req *SecretRequest) (*SecretResponse, error) {
	requestID := fmt.Sprintf("secret_%d", time.Now().UnixMilli())
	req.RequestID = requestID

	resultCh := make(chan *SecretResponse, 1)
	psr := &pendingSecretRequest{
		request:  req,
		resultCh: resultCh,
		deadline: time.Now().Add(m.timeout),
	}

	m.mu.Lock()
	m.pending[requestID] = psr
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.pending, requestID)
		m.mu.Unlock()
	}()

	// Publish Matrix event.
	if m.sendMatrixMsg != nil {
		body := fmt.Sprintf(
			`{"request_id":"%s","agent_id":"%s","team_id":"%s","credential_name":"%s","target_domain":"%s","reason":"%s","risk_class":"%s","timeout_s":%d}`,
			requestID, req.AgentID, req.TeamID, req.CredentialName, req.TargetDomain, req.Reason, req.RiskClass, int(m.timeout.Seconds()),
		)
		_ = m.sendMatrixMsg("", "app.armorclaw.secret_request", body)
	}

	select {
	case resp := <-resultCh:
		return resp, nil
	case <-time.After(m.timeout):
		if m.log != nil {
			m.log.Warn("secret_request_timeout", "request_id", requestID, "agent_id", req.AgentID, "credential_name", req.CredentialName)
		}
		return &SecretResponse{
			RequestID: requestID,
			Approved:  false,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// HandleResponse delivers a user response to a pending secret request.
func (m *SecretRequestManager) HandleResponse(requestID string, approved bool, respondedBy string, secretValue string) error {
	m.mu.RLock()
	psr, ok := m.pending[requestID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("secret request %s not found or expired", requestID)
	}

	psr.resultCh <- &SecretResponse{
		RequestID:   requestID,
		Approved:    approved,
		RespondedBy: respondedBy,
		SecretValue: secretValue,
	}
	return nil
}

// PendingCount returns the number of currently pending secret requests.
func (m *SecretRequestManager) PendingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pending)
}
