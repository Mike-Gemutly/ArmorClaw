package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// JetskiBroker implements BrowserBroker by delegating to a Jetski sidecar.
// It manages browser sessions via the Jetski RPC API and sends CDP commands
// through a WebSocket connection to Jetski's CDP proxy (:9222).
type JetskiBroker struct {
	cdpURL  string       // ws://<host>:9222 — CDP proxy endpoint
	rpcURL  string       // http://<host>:9223 — RPC management endpoint
	logger  *slog.Logger

	// sessionMap maps JobID → Jetski session ID.
	sessionMap sync.Map

	// wsConns maps JobID → active WebSocket connection to CDP proxy.
	wsConns sync.Map

	// cdpMsgID is an atomic counter for CDP message IDs.
	cdpMsgID atomic.Int64
}

// NewJetskiBroker creates a new JetskiBroker.
// cdpURL is the WebSocket URL for the CDP proxy (e.g. "ws://localhost:9222").
// rpcURL is the HTTP URL for the Jetski RPC API (e.g. "http://localhost:9223").
func NewJetskiBroker(cdpURL string, rpcURL string, logger *slog.Logger) *JetskiBroker {
	return &JetskiBroker{
		cdpURL: cdpURL,
		rpcURL: rpcURL,
		logger: logger,
	}
}

// cdpMessage mirrors Jetski's CDPMessage wire format.
type cdpMessage struct {
	ID     int             `json:"id"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *cdpError       `json:"error,omitempty"`
}

// cdpError mirrors Jetski's CDPError.
type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

//-----------------------------------------------------------------------------
// Session lifecycle
//-----------------------------------------------------------------------------

// StartJob creates a new browser session via Jetski RPC and returns a JobID.
func (b *JetskiBroker) StartJob(ctx context.Context, req StartJobRequest) (JobID, error) {
	b.logger.Info("jetski: creating session", "agent_id", req.AgentID)

	url := b.rpcURL + "/rpc/session/create"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("jetski: build create-session request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("jetski: create-session rpc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("jetski: create-session returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("jetski: decode session response: %w", err)
	}

	if result.ID == "" {
		return "", fmt.Errorf("jetski: create-session returned empty session id")
	}

	jobID := JobID(result.ID)
	b.sessionMap.Store(jobID, result.ID)

	// Establish WebSocket connection to CDP proxy.
	wsConn, _, err := websocket.DefaultDialer.DialContext(ctx, b.cdpURL, nil)
	if err != nil {
		b.sessionMap.Delete(jobID)
		return "", fmt.Errorf("jetski: connect cdp websocket: %w", err)
	}
	b.wsConns.Store(jobID, wsConn)

	b.logger.Info("jetski: session created", "job_id", jobID, "session_id", result.ID)
	return jobID, nil
}

// Status queries the Jetski RPC for overall session status.
func (b *JetskiBroker) Status(ctx context.Context, id JobID) (*JobSummary, error) {
	b.logger.Debug("jetski: status query", "job_id", id)

	url := b.rpcURL + "/rpc/status"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("jetski: build status request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("jetski: status rpc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jetski: status returned %d: %s", resp.StatusCode, string(body))
	}

	var statusResp struct {
		ActiveSessions int    `json:"active_sessions"`
		EngineHealth   string `json:"engine_health"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("jetski: decode status response: %w", err)
	}

	// Verify the session still exists in our mapping.
	_, ok := b.sessionMap.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: unknown job %s", id)
	}

	jobStatus := JobStatusRunning
	if statusResp.EngineHealth != "ok" {
		jobStatus = JobStatusFailed
	}

	now := time.Now()
	return &JobSummary{
		ID:        id,
		Status:    jobStatus,
		CreatedAt: now,
		StartedAt: &now,
	}, nil
}

// Complete closes the Jetski session and cleans up resources.
func (b *JetskiBroker) Complete(ctx context.Context, id JobID) (*BrokerResult, error) {
	b.logger.Info("jetski: completing session", "job_id", id)

	if err := b.closeSession(ctx, id); err != nil {
		b.logger.Error("jetski: close session on complete", "job_id", id, "error", err)
	}

	return &BrokerResult{
		JobID:   id,
		Success: true,
	}, nil
}

// Fail terminates a session due to an error.
func (b *JetskiBroker) Fail(ctx context.Context, id JobID, reason string) error {
	b.logger.Info("jetski: failing session", "job_id", id, "reason", reason)

	if err := b.closeSession(ctx, id); err != nil {
		b.logger.Error("jetski: close session on fail", "job_id", id, "error", err)
	}
	return nil
}

// List returns all active sessions for an agent.
func (b *JetskiBroker) List(ctx context.Context, agentID string) ([]JobSummary, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Cancel aborts a running session.
func (b *JetskiBroker) Cancel(ctx context.Context, id JobID) error {
	b.logger.Info("jetski: cancelling session", "job_id", id)

	if err := b.closeSession(ctx, id); err != nil {
		b.logger.Error("jetski: close session on cancel", "job_id", id, "error", err)
	}
	return nil
}

//-----------------------------------------------------------------------------
// Browser operations
//-----------------------------------------------------------------------------

// Navigate sends a CDP Page.navigate command via WebSocket to Jetski's CDP proxy.
func (b *JetskiBroker) Navigate(ctx context.Context, id JobID, url string) (*BrokerResult, error) {
	b.logger.Info("jetski: navigate", "job_id", id, "url", url)

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	msgID := int(b.cdpMsgID.Add(1))

	params, _ := json.Marshal(map[string]string{
		"url": url,
	})

	msg := cdpMessage{
		ID:     msgID,
		Method: "Page.navigate",
		Params: params,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("jetski: write cdp Page.navigate: %w", err)
	}

	// Read the response. Set a reasonable deadline.
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("jetski: set read deadline: %w", err)
	}

	var resp cdpMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return nil, fmt.Errorf("jetski: read cdp response: %w", err)
	}

	// Clear the deadline.
	_ = conn.SetReadDeadline(time.Time{})

	if resp.Error != nil {
		return &BrokerResult{
			JobID:   id,
			Success: false,
			URL:     url,
			Error: &BrokerError{
				Code:    fmt.Sprintf("%d", resp.Error.Code),
				Message: resp.Error.Message,
			},
		}, fmt.Errorf("jetski: cdp error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return &BrokerResult{
		JobID:   id,
		Success: true,
		URL:     url,
	}, nil
}

// Fill injects values into form fields (stub).
func (b *JetskiBroker) Fill(ctx context.Context, id JobID, fields []FillRequest) (*BrokerResult, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Click clicks an element (stub).
func (b *JetskiBroker) Click(ctx context.Context, id JobID, selector string) (*BrokerResult, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// WaitForElement waits for an element to appear (stub).
func (b *JetskiBroker) WaitForElement(ctx context.Context, id JobID, selector string, timeoutMs int) (*BrokerResult, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// WaitForCaptcha waits for CAPTCHA resolution (stub).
func (b *JetskiBroker) WaitForCaptcha(ctx context.Context, id JobID, timeoutMs int) (*BrokerResult, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// WaitFor2FA waits for 2FA input (stub).
func (b *JetskiBroker) WaitFor2FA(ctx context.Context, id JobID, timeoutMs int) (*BrokerResult, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Extract extracts data from the page (stub).
func (b *JetskiBroker) Extract(ctx context.Context, id JobID, spec ExtractSpec) (*ExtractResult, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Screenshot captures a page screenshot (stub).
func (b *JetskiBroker) Screenshot(ctx context.Context, id JobID, fullPage bool) (*BrokerResult, error) {
	return nil, fmt.Errorf("not yet implemented")
}

//-----------------------------------------------------------------------------
// Internal helpers
//-----------------------------------------------------------------------------

// closeSession calls the Jetski RPC to close a session and cleans up local state.
func (b *JetskiBroker) closeSession(ctx context.Context, id JobID) error {
	sessionID, ok := b.sessionMap.Load(id)
	if !ok {
		return nil // already cleaned up
	}

	// Close WebSocket connection.
	if wsConn, ok := b.wsConns.LoadAndDelete(id); ok {
		conn := wsConn.(*websocket.Conn)
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "session complete"))
		_ = conn.Close()
	}

	// Call Jetski RPC to close the session.
	url := b.rpcURL + "/rpc/session/close"
	body := fmt.Sprintf(`{"id":"%s"}`, sessionID.(string))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err == nil {
		httpReq.Body = io.NopCloser(stringReader(body))
		httpReq.ContentLength = int64(len(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err == nil {
			resp.Body.Close()
		}
	}

	b.sessionMap.Delete(id)
	b.logger.Info("jetski: session closed", "job_id", id)
	return nil
}

// stringReader is a minimal io.Reader wrapper for a string.
type stringReader string

func (s stringReader) Read(p []byte) (int, error) {
	if len(s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s)
	return n, nil
}
