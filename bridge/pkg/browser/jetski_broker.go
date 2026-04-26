package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
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

	// jobContexts maps JobID → context.CancelFunc for cancelling running jobs.
	jobContexts sync.Map

	// jobMeta maps JobID → *jobMetadata for tracking job lifecycle info.
	jobMeta sync.Map

	// cdpMsgID is an atomic counter for CDP message IDs.
	cdpMsgID atomic.Int64
}

// jobMetadata tracks per-job info for List and lifecycle reporting.
type jobMetadata struct {
	AgentID   string
	CreatedAt time.Time
	StartedAt time.Time
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

	jobCtx, cancel := context.WithCancel(ctx)

	url := b.rpcURL + "/rpc/session/create"

	httpReq, err := http.NewRequestWithContext(jobCtx, http.MethodPost, url, nil)
	if err != nil {
		cancel()
		return "", fmt.Errorf("jetski: build create-session request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		cancel()
		return "", fmt.Errorf("jetski: create-session rpc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		cancel()
		return "", fmt.Errorf("jetski: create-session returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		cancel()
		return "", fmt.Errorf("jetski: decode session response: %w", err)
	}

	if result.ID == "" {
		cancel()
		return "", fmt.Errorf("jetski: create-session returned empty session id")
	}

	jobID := JobID(result.ID)
	b.sessionMap.Store(jobID, result.ID)
	b.jobContexts.Store(jobID, cancel)

	now := time.Now()
	b.jobMeta.Store(jobID, &jobMetadata{
		AgentID:   req.AgentID,
		CreatedAt: now,
		StartedAt: now,
	})

	wsConn, _, err := websocket.DefaultDialer.DialContext(jobCtx, b.cdpURL, nil)
	if err != nil {
		b.sessionMap.Delete(jobID)
		b.jobContexts.Delete(jobID)
		b.jobMeta.Delete(jobID)
		cancel()
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
	var jobs []JobSummary

	b.jobMeta.Range(func(key, value any) bool {
		id := key.(JobID)
		meta := value.(*jobMetadata)

		if agentID != "" && meta.AgentID != agentID {
			return true
		}

		startedAt := meta.StartedAt
		jobs = append(jobs, JobSummary{
			ID:        id,
			AgentID:   meta.AgentID,
			Status:    JobStatusRunning,
			CreatedAt: meta.CreatedAt,
			StartedAt: &startedAt,
		})
		return true
	})

	return jobs, nil
}

// Cancel aborts a running session.
func (b *JetskiBroker) Cancel(ctx context.Context, id JobID) error {
	b.logger.Info("jetski: cancelling session", "job_id", id)

	if cancelFn, ok := b.jobContexts.LoadAndDelete(id); ok {
		cancelFn.(context.CancelFunc)()
	}

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

// Fill injects values into form fields via CDP Input.insertText.
//
// When Sensitive=false the literal Value is inserted directly.
// When Sensitive=true the Value is treated as a PII placeholder; the broker
// sends an approval request through the Bridge PII flow, waits for the human
// to approve, receives the actual value, and then inserts it — the real value
// is never logged.
func (b *JetskiBroker) Fill(ctx context.Context, id JobID, fields []FillRequest) (*BrokerResult, error) {
	start := time.Now()

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	for i, field := range fields {
		fillValue := field.Value

		if field.Sensitive {
			b.logger.Info("jetski: filling sensitive field [REDACTED]",
				"job_id", id, "selector", field.Selector, "field_index", i)

			approved, err := b.resolveSensitiveValue(ctx, id, field)
			if err != nil {
				return &BrokerResult{
					JobID:   id,
					Success: false,
					Duration: elapsedMs(start),
					Error: &BrokerError{
						Code:     "PII_APPROVAL_FAILED",
						Message:  fmt.Sprintf("sensitive field %q: %v", field.Selector, err),
						Selector: field.Selector,
					},
				}, fmt.Errorf("jetski: sensitive fill approval for %q: %w", field.Selector, err)
			}
			fillValue = approved
		} else {
			b.logger.Info("jetski: filling field",
				"job_id", id, "selector", field.Selector, "field_index", i)
		}

		if err := b.cdpFocus(conn, field.Selector); err != nil {
			return &BrokerResult{
				JobID:   id,
				Success: false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:     "FOCUS_FAILED",
					Message:  fmt.Sprintf("focus %q: %v", field.Selector, err),
					Selector: field.Selector,
				},
			}, fmt.Errorf("jetski: focus %q: %w", field.Selector, err)
		}

		if err := b.cdpInsertText(conn, fillValue); err != nil {
			return &BrokerResult{
				JobID:   id,
				Success: false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:     "INSERT_FAILED",
					Message:  fmt.Sprintf("insert text into %q: %v", field.Selector, err),
					Selector: field.Selector,
				},
			}, fmt.Errorf("jetski: insert text into %q: %w", field.Selector, err)
		}
	}

	return &BrokerResult{
		JobID:    id,
		Success:  true,
		Duration: elapsedMs(start),
	}, nil
}

// resolveSensitiveValue routes a sensitive fill request through the Bridge PII
// approval flow. It creates an approval request referencing the ValueRef
// (keystore PII reference), waits for the human-in-the-loop to grant access,
// and returns the actual value for injection. The returned value must never be
// logged.
func (b *JetskiBroker) resolveSensitiveValue(ctx context.Context, id JobID, field FillRequest) (string, error) {
	reqBody := struct {
		JobID     string `json:"job_id"`
		Selector  string `json:"selector"`
		ValueRef  string `json:"value_ref"`
		Requester string `json:"requester"`
	}{
		JobID:     string(id),
		Selector:  field.Selector,
		ValueRef:  field.ValueRef,
		Requester: "jetski-broker",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal approval request: %w", err)
	}

	url := b.rpcURL + "/rpc/approval/request"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, stringReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("build approval request: %w", err)
	}
	httpReq.ContentLength = int64(len(bodyBytes))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("approval request rpc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("approval returned %d: %s", resp.StatusCode, string(respBody))
	}

	var approvalResp struct {
		Approved bool   `json:"approved"`
		Value    string `json:"value"`
		Reason   string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&approvalResp); err != nil {
		return "", fmt.Errorf("decode approval response: %w", err)
	}

	if !approvalResp.Approved {
		return "", fmt.Errorf("approval denied: %s", approvalResp.Reason)
	}

	return approvalResp.Value, nil
}

// cdpFocus sends a CDP command to focus the element matching the given selector.
// It uses Runtime.evaluate to call document.querySelector().focus().
func (b *JetskiBroker) cdpFocus(conn *websocket.Conn, selector string) error {
	msgID := int(b.cdpMsgID.Add(1))

	expr := fmt.Sprintf(`document.querySelector(%q).focus()`, selector)
	params, _ := json.Marshal(map[string]interface{}{
		"expression":            expr,
		"includeCommandLineAPI": true,
	})

	msg := cdpMessage{
		ID:     msgID,
		Method: "Runtime.evaluate",
		Params: params,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("write cdp Runtime.evaluate focus: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	var resp cdpMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("read cdp focus response: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})

	if resp.Error != nil {
		return fmt.Errorf("cdp focus error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return nil
}

// cdpInsertText sends a CDP Input.insertText command with the given value.
func (b *JetskiBroker) cdpInsertText(conn *websocket.Conn, value string) error {
	msgID := int(b.cdpMsgID.Add(1))

	params, _ := json.Marshal(map[string]string{
		"text": value,
	})

	msg := cdpMessage{
		ID:     msgID,
		Method: "Input.insertText",
		Params: params,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("write cdp Input.insertText: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	var resp cdpMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("read cdp insertText response: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})

	if resp.Error != nil {
		return fmt.Errorf("cdp insertText error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return nil
}

// fillWithHumanLikeDelay sends text character-by-character with random delays
// to simulate human typing, used when BrowserStealthConfig.HumanLikeTyping is true.
//
// TODO: Wire BrowserStealthConfig.HumanLikeTyping into the broker and call this
// method instead of cdpInsertText when the flag is enabled. Expected behavior:
// send each character via CDP Input.dispatchKeyEvent with type="char" and a
// random inter-key delay of 30–150ms drawn from a uniform distribution.
func (b *JetskiBroker) fillWithHumanLikeDelay(conn *websocket.Conn, value string) error {
	return fmt.Errorf("fillWithHumanLikeDelay: not yet implemented (awaiting BrowserStealthConfig)")
}

func elapsedMs(start time.Time) int {
	return int(time.Since(start).Milliseconds())
}

func joinSelectors(selectors []string) string {
	result := selectors[0]
	for _, s := range selectors[1:] {
		result += "," + s
	}
	return result
}

// Click clicks an element via CDP Runtime.evaluate using document.querySelector().click().
func (b *JetskiBroker) Click(ctx context.Context, id JobID, selector string) (*BrokerResult, error) {
	start := time.Now()
	b.logger.Info("jetski: click", "job_id", id, "selector", selector)

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	msgID := int(b.cdpMsgID.Add(1))

	expr := fmt.Sprintf(
		`(function(){var el=document.querySelector(%q);if(!el)throw new Error("element not found: %s");el.click();return true})()`,
		selector, selector,
	)
	params, _ := json.Marshal(map[string]interface{}{
		"expression":            expr,
		"includeCommandLineAPI": true,
		"awaitPromise":          true,
		"returnByValue":         true,
		"userGesture":           true,
	})

	msg := cdpMessage{
		ID:     msgID,
		Method: "Runtime.evaluate",
		Params: params,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:     "CLICK_WRITE_FAILED",
				Message:  fmt.Sprintf("write cdp click: %v", err),
				Selector: selector,
			},
		}, fmt.Errorf("jetski: write cdp click: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return nil, fmt.Errorf("jetski: set read deadline: %w", err)
	}

	var resp cdpMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:     "CLICK_READ_FAILED",
				Message:  fmt.Sprintf("read cdp click response: %v", err),
				Selector: selector,
			},
		}, fmt.Errorf("jetski: read cdp click response: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})

	if resp.Error != nil {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:     fmt.Sprintf("CDP_%d", resp.Error.Code),
				Message:  resp.Error.Message,
				Selector: selector,
			},
		}, fmt.Errorf("jetski: cdp click error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return &BrokerResult{
		JobID:    id,
		Success:  true,
		Duration: elapsedMs(start),
	}, nil
}

// WaitForElement polls the DOM for an element matching selector until found or timeout.
func (b *JetskiBroker) WaitForElement(ctx context.Context, id JobID, selector string, timeoutMs int) (*BrokerResult, error) {
	start := time.Now()
	b.logger.Info("jetski: wait for element", "job_id", id, "selector", selector, "timeout_ms", timeoutMs)

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	pollInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:     "WAIT_CANCELLED",
					Message:  ctx.Err().Error(),
					Selector: selector,
				},
			}, ctx.Err()
		default:
		}

		msgID := int(b.cdpMsgID.Add(1))

		expr := fmt.Sprintf(`document.querySelector(%q)!==null`, selector)
		params, _ := json.Marshal(map[string]interface{}{
			"expression":    expr,
			"returnByValue": true,
		})

		msg := cdpMessage{
			ID:     msgID,
			Method: "Runtime.evaluate",
			Params: params,
		}

		if err := conn.WriteJSON(msg); err != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:     "WAIT_WRITE_FAILED",
					Message:  fmt.Sprintf("write cdp query: %v", err),
					Selector: selector,
				},
			}, fmt.Errorf("jetski: write cdp wait query: %w", err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return nil, fmt.Errorf("jetski: set read deadline: %w", err)
		}

		var resp cdpMessage
		if err := conn.ReadJSON(&resp); err != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:     "WAIT_READ_FAILED",
					Message:  fmt.Sprintf("read cdp query response: %v", err),
					Selector: selector,
				},
			}, fmt.Errorf("jetski: read cdp wait response: %w", err)
		}
		_ = conn.SetReadDeadline(time.Time{})

		if resp.Error != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:     fmt.Sprintf("CDP_%d", resp.Error.Code),
					Message:  resp.Error.Message,
					Selector: selector,
				},
			}, fmt.Errorf("jetski: cdp wait query error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		var result struct {
			Value string `json:"value"`
		}
		if resp.Result != nil {
			_ = json.Unmarshal(resp.Result, &result)
		}

		if result.Value == "true" {
			return &BrokerResult{
				JobID:    id,
				Success:  true,
				Duration: elapsedMs(start),
			}, nil
		}

		time.Sleep(pollInterval)
	}

	return &BrokerResult{
		JobID:    id,
		Success:  false,
		Duration: elapsedMs(start),
		Error: &BrokerError{
			Code:     "WAIT_TIMEOUT",
			Message:  fmt.Sprintf("element %q not found within %dms", selector, timeoutMs),
			Selector: selector,
		},
	}, fmt.Errorf("jetski: wait for element %q timed out after %dms", selector, timeoutMs)
}

// WaitForCaptcha waits for CAPTCHA resolution by monitoring common captcha selectors for disappearance.
func (b *JetskiBroker) WaitForCaptcha(ctx context.Context, id JobID, timeoutMs int) (*BrokerResult, error) {
	start := time.Now()
	b.logger.Info("jetski: wait for captcha resolution", "job_id", id, "timeout_ms", timeoutMs)

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	captchaSelectors := []string{
		`iframe[src*="captcha"]`,
		`.g-recaptcha`,
		`#captcha`,
		`iframe[title*="recaptcha"]`,
		`.h-captcha`,
		`iframe[src*="hcaptcha"]`,
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	pollInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    "WAIT_CAPTCHA_CANCELLED",
					Message: ctx.Err().Error(),
				},
			}, ctx.Err()
		default:
		}

		msgID := int(b.cdpMsgID.Add(1))

		expr := fmt.Sprintf(
			`(function(){var s=%q;var sel=s.split(',');for(var i=0;i<sel.length;i++){if(document.querySelector(sel[i]))return false}return true})()`,
			joinSelectors(captchaSelectors),
		)
		params, _ := json.Marshal(map[string]interface{}{
			"expression":    expr,
			"returnByValue": true,
		})

		msg := cdpMessage{
			ID:     msgID,
			Method: "Runtime.evaluate",
			Params: params,
		}

		if err := conn.WriteJSON(msg); err != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    "WAIT_CAPTCHA_WRITE_FAILED",
					Message: fmt.Sprintf("write cdp captcha query: %v", err),
				},
			}, fmt.Errorf("jetski: write cdp captcha query: %w", err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return nil, fmt.Errorf("jetski: set read deadline: %w", err)
		}

		var resp cdpMessage
		if err := conn.ReadJSON(&resp); err != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    "WAIT_CAPTCHA_READ_FAILED",
					Message: fmt.Sprintf("read cdp captcha response: %v", err),
				},
			}, fmt.Errorf("jetski: read cdp captcha response: %w", err)
		}
		_ = conn.SetReadDeadline(time.Time{})

		if resp.Error != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    fmt.Sprintf("CDP_%d", resp.Error.Code),
					Message: resp.Error.Message,
				},
			}, fmt.Errorf("jetski: cdp captcha query error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		var result struct {
			Value string `json:"value"`
		}
		if resp.Result != nil {
			_ = json.Unmarshal(resp.Result, &result)
		}

		if result.Value == "true" {
			return &BrokerResult{
				JobID:    id,
				Success:  true,
				Duration: elapsedMs(start),
			}, nil
		}

		time.Sleep(pollInterval)
	}

	return &BrokerResult{
		JobID:    id,
		Success:  false,
		Duration: elapsedMs(start),
		Error: &BrokerError{
			Code:    "WAIT_CAPTCHA_TIMEOUT",
			Message: fmt.Sprintf("captcha not resolved within %dms", timeoutMs),
		},
	}, fmt.Errorf("jetski: captcha not resolved within %dms", timeoutMs)
}

// WaitFor2FA waits for 2FA input fields to appear in the DOM.
func (b *JetskiBroker) WaitFor2FA(ctx context.Context, id JobID, timeoutMs int) (*BrokerResult, error) {
	start := time.Now()
	b.logger.Info("jetski: wait for 2fa input", "job_id", id, "timeout_ms", timeoutMs)

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	twoFASelectors := []string{
		`input[type="tel"]`,
		`input[name*="otp"]`,
		`input[name*="code"]`,
		`input[name*="token"]`,
		`input[autocomplete="one-time-code"]`,
		`input[inputmode="numeric"]`,
		`input[placeholder*="verification"]`,
		`input[placeholder*="code"]`,
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	pollInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    "WAIT_2FA_CANCELLED",
					Message: ctx.Err().Error(),
				},
			}, ctx.Err()
		default:
		}

		msgID := int(b.cdpMsgID.Add(1))

		expr := fmt.Sprintf(
			`(function(){var s=%q;var sel=s.split(',');for(var i=0;i<sel.length;i++){if(document.querySelector(sel[i]))return true}return false})()`,
			joinSelectors(twoFASelectors),
		)
		params, _ := json.Marshal(map[string]interface{}{
			"expression":    expr,
			"returnByValue": true,
		})

		msg := cdpMessage{
			ID:     msgID,
			Method: "Runtime.evaluate",
			Params: params,
		}

		if err := conn.WriteJSON(msg); err != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    "WAIT_2FA_WRITE_FAILED",
					Message: fmt.Sprintf("write cdp 2fa query: %v", err),
				},
			}, fmt.Errorf("jetski: write cdp 2fa query: %w", err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return nil, fmt.Errorf("jetski: set read deadline: %w", err)
		}

		var resp cdpMessage
		if err := conn.ReadJSON(&resp); err != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    "WAIT_2FA_READ_FAILED",
					Message: fmt.Sprintf("read cdp 2fa response: %v", err),
				},
			}, fmt.Errorf("jetski: read cdp 2fa response: %w", err)
		}
		_ = conn.SetReadDeadline(time.Time{})

		if resp.Error != nil {
			return &BrokerResult{
				JobID:    id,
				Success:  false,
				Duration: elapsedMs(start),
				Error: &BrokerError{
					Code:    fmt.Sprintf("CDP_%d", resp.Error.Code),
					Message: resp.Error.Message,
				},
			}, fmt.Errorf("jetski: cdp 2fa query error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		var result struct {
			Value string `json:"value"`
		}
		if resp.Result != nil {
			_ = json.Unmarshal(resp.Result, &result)
		}

		if result.Value == "true" {
			return &BrokerResult{
				JobID:    id,
				Success:  true,
				Duration: elapsedMs(start),
			}, nil
		}

		time.Sleep(pollInterval)
	}

	return &BrokerResult{
		JobID:    id,
		Success:  false,
		Duration: elapsedMs(start),
		Error: &BrokerError{
			Code:    "WAIT_2FA_TIMEOUT",
			Message: fmt.Sprintf("2fa input not found within %dms", timeoutMs),
		},
	}, fmt.Errorf("jetski: 2fa input not found within %dms", timeoutMs)
}

// Extract queries the page DOM for each field in spec and returns a map of
// field name → extracted value. It builds a single JS expression that
// evaluates all selectors at once via Runtime.evaluate and parses the JSON
// result.
func (b *JetskiBroker) Extract(ctx context.Context, id JobID, spec ExtractSpec) (*ExtractResult, error) {
	start := time.Now()
	b.logger.Info("jetski: extract", "job_id", id, "fields", len(spec.Fields))

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	var jsParts []string
	for _, f := range spec.Fields {
		attr := f.Attribute
		if attr == "" {
			attr = "textContent"
		}
		jsParts = append(jsParts, fmt.Sprintf(
			`%q: document.querySelector(%q)?.%s?.trim() ?? ""`,
			f.Name, f.Selector, attr,
		))
	}
	expr := fmt.Sprintf(`(function(){return JSON.stringify({%s})})()`, strings.Join(jsParts, ","))

	msgID := int(b.cdpMsgID.Add(1))
	params, _ := json.Marshal(map[string]interface{}{
		"expression":    expr,
		"returnByValue": true,
	})

	msg := cdpMessage{
		ID:     msgID,
		Method: "Runtime.evaluate",
		Params: params,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("jetski: write cdp extract: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return nil, fmt.Errorf("jetski: set read deadline: %w", err)
	}

	var resp cdpMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return nil, fmt.Errorf("jetski: read cdp extract response: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})

	if resp.Error != nil {
		return nil, fmt.Errorf("jetski: cdp extract error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	var cdpResult struct {
		Value string `json:"value"`
	}
	if resp.Result != nil {
		_ = json.Unmarshal(resp.Result, &cdpResult)
	}

	fields := make(map[string]string)
	if err := json.Unmarshal([]byte(cdpResult.Value), &fields); err != nil {
		return nil, fmt.Errorf("jetski: parse extract result: %w (raw: %q)", err, cdpResult.Value)
	}

	b.logger.Info("jetski: extract complete", "job_id", id, "fields_extracted", len(fields), "duration_ms", elapsedMs(start))

	return &ExtractResult{
		Fields: fields,
	}, nil
}

// Screenshot captures a viewport PNG screenshot via CDP Page.captureScreenshot
// and returns the decoded PNG bytes in BrokerResult.Screenshots.
func (b *JetskiBroker) Screenshot(ctx context.Context, id JobID, fullPage bool) (*BrokerResult, error) {
	start := time.Now()
	b.logger.Info("jetski: screenshot", "job_id", id, "fullPage", fullPage)

	wsConn, ok := b.wsConns.Load(id)
	if !ok {
		return nil, fmt.Errorf("jetski: no cdp connection for job %s", id)
	}
	conn := wsConn.(*websocket.Conn)

	msgID := int(b.cdpMsgID.Add(1))
	params, _ := json.Marshal(map[string]interface{}{
		"format": "png",
	})

	msg := cdpMessage{
		ID:     msgID,
		Method: "Page.captureScreenshot",
		Params: params,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:    "SCREENSHOT_WRITE_FAILED",
				Message: fmt.Sprintf("write cdp screenshot: %v", err),
			},
		}, fmt.Errorf("jetski: write cdp screenshot: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("jetski: set read deadline: %w", err)
	}

	var resp cdpMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:    "SCREENSHOT_READ_FAILED",
				Message: fmt.Sprintf("read cdp screenshot response: %v", err),
			},
		}, fmt.Errorf("jetski: read cdp screenshot response: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})

	if resp.Error != nil {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:    fmt.Sprintf("CDP_%d", resp.Error.Code),
				Message: resp.Error.Message,
			},
		}, fmt.Errorf("jetski: cdp screenshot error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	var cdpResult struct {
		Data string `json:"data"`
	}
	if resp.Result != nil {
		_ = json.Unmarshal(resp.Result, &cdpResult)
	}

	if cdpResult.Data == "" {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:    "SCREENSHOT_EMPTY",
				Message: "cdp returned empty screenshot data",
			},
		}, fmt.Errorf("jetski: cdp screenshot returned empty data")
	}

	pngBytes, err := base64.StdEncoding.DecodeString(cdpResult.Data)
	if err != nil {
		return &BrokerResult{
			JobID:    id,
			Success:  false,
			Duration: elapsedMs(start),
			Error: &BrokerError{
				Code:    "SCREENSHOT_DECODE_FAILED",
				Message: fmt.Sprintf("base64 decode: %v", err),
			},
		}, fmt.Errorf("jetski: decode screenshot base64: %w", err)
	}

	b.logger.Info("jetski: screenshot complete", "job_id", id, "size_bytes", len(pngBytes), "duration_ms", elapsedMs(start))

	return &BrokerResult{
		JobID:       id,
		Success:     true,
		Duration:    elapsedMs(start),
		Screenshots: []string{base64.StdEncoding.EncodeToString(pngBytes)},
	}, nil
}

//-----------------------------------------------------------------------------
// Chart replay
//-----------------------------------------------------------------------------

// ReplayChart replays all actions from a NavChart, enforcing the PII approval
// flow for any input action whose value is a PII placeholder ({{field_name}}).
//
// Every sensitive fill is routed through the same Bridge PII approval path as
// Fill(Sensitive=true). If any approval is denied the replay is aborted
// immediately and a detailed error is returned describing how far execution
// progressed. Approvals are never cached between actions.
func (b *JetskiBroker) ReplayChart(ctx context.Context, jobID JobID, chart NavChart, piiValues map[string]string) error {
	b.logger.Info("jetski: replay chart start",
		"job_id", jobID,
		"target_domain", chart.TargetDomain,
		"action_count", len(chart.ActionMap),
	)

	keys := make([]string, 0, len(chart.ActionMap))
	for k := range chart.ActionMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, key := range keys {
		action := chart.ActionMap[key]

		select {
		case <-ctx.Done():
			return fmt.Errorf("replay: cancelled before action %q (step %d/%d): %w",
				key, i+1, len(keys), ctx.Err())
		default:
		}

		b.logger.Info("jetski: replay action",
			"job_id", jobID,
			"step", i+1,
			"total", len(keys),
			"action_key", key,
			"action_type", string(action.ActionType),
		)

		switch action.ActionType {
		case ActionNavigate:
			if action.URL == "" {
				return fmt.Errorf("replay: navigate action %q has empty URL at step %d",
					key, i+1)
			}
			result, err := b.Navigate(ctx, jobID, action.URL)
			if err != nil {
				return fmt.Errorf("replay: navigate %q failed at step %d (completed %d/%d): %w",
					key, i+1, i, len(keys), err)
			}
			if result != nil && !result.Success {
				return fmt.Errorf("replay: navigate %q returned failure at step %d (completed %d/%d)",
					key, i+1, i, len(keys))
			}
			b.logger.Info("jetski: replay navigate done",
				"job_id", jobID, "action_key", key, "url", action.URL)

		case ActionClick:
			tier := b.tryClickWithFallback(ctx, jobID, action.Selector)
			b.logger.Info("jetski: replay click done",
				"job_id", jobID, "action_key", key, "selector_tier", string(tier))

		case ActionInput:
			selector, tier := b.resolveSelectorWithFallback(action.Selector)

			value := action.Value

			sensitive := false
			valueRef := ""

			if isPIIPlaceholder(value) {
				placeholderKey := extractPlaceholderKey(value)
				resolved, ok := piiValues[placeholderKey]
				if !ok {
					return fmt.Errorf(
						"replay: unresolved PII placeholder {{%s}} in action %q at step %d",
						placeholderKey, key, i+1)
				}
				sensitive = true
				valueRef = resolved
				b.logger.Info("jetski: replay input [PII — approval required]",
					"job_id", jobID,
					"action_key", key,
					"selector", selector,
					"placeholder", placeholderKey,
				)
			} else {
				value = resolveInlinePlaceholders(value, piiValues)
				b.logger.Info("jetski: replay input",
					"job_id", jobID,
					"action_key", key,
					"selector", selector,
				)
			}

			fields := []FillRequest{{
				Selector:  selector,
				Value:     value,
				ValueRef:  valueRef,
				Sensitive: sensitive,
			}}
			result, err := b.Fill(ctx, jobID, fields)
			if err != nil {
				if sensitive {
					return fmt.Errorf(
						"replay: PII approval denied for action %q at step %d (completed %d/%d): %w",
						key, i+1, i, len(keys), err)
				}
				return fmt.Errorf("replay: fill action %q failed at step %d (completed %d/%d): %w",
					key, i+1, i, len(keys), err)
			}
			if result != nil && !result.Success {
				return fmt.Errorf("replay: fill action %q returned failure at step %d (completed %d/%d)",
					key, i+1, i, len(keys))
			}

			approvalStatus := "not_needed"
			if sensitive {
				approvalStatus = "granted"
			}
			b.logger.Info("jetski: replay input done",
				"job_id", jobID,
				"action_key", key,
				"selector", selector,
				"approval", approvalStatus,
				"selector_tier", string(tier),
			)

		case ActionWait:
			timeout := 5000
			if action.PostActionWait != nil && action.PostActionWait.Timeout > 0 {
				timeout = action.PostActionWait.Timeout
			}

			if action.PostActionWait != nil &&
				action.PostActionWait.Selector != nil &&
				action.PostActionWait.Selector.PrimaryCSS != "" {
				selector := action.PostActionWait.Selector.PrimaryCSS
				b.logger.Info("jetski: replay wait for element",
					"job_id", jobID, "action_key", key, "selector", selector, "timeout_ms", timeout)
				result, err := b.WaitForElement(ctx, jobID, selector, timeout)
				if err != nil {
					return fmt.Errorf(
						"replay: wait for element %q failed at step %d (completed %d/%d): %w",
						selector, i+1, i, len(keys), err)
				}
				if result != nil && !result.Success {
					return fmt.Errorf(
						"replay: wait for element %q timed out at step %d (completed %d/%d)",
						selector, i+1, i, len(keys))
				}
			} else {
				b.logger.Info("jetski: replay wait delay",
					"job_id", jobID, "action_key", key, "timeout_ms", timeout)
				select {
				case <-time.After(time.Duration(timeout) * time.Millisecond):
				case <-ctx.Done():
					return fmt.Errorf(
						"replay: wait action %q cancelled at step %d (completed %d/%d): %w",
						key, i+1, i, len(keys), ctx.Err())
				}
			}

		case ActionAssert:
			b.logger.Info("jetski: replay assert (verification-only)",
				"job_id", jobID,
				"action_key", key,
			)

		default:
			b.logger.Warn("jetski: replay unknown action type — skipping",
				"job_id", jobID,
				"action_key", key,
				"action_type", string(action.ActionType),
			)
		}

		if action.ActionType != ActionWait && action.PostActionWait != nil {
			waitTimeout := 5000
			if action.PostActionWait.Timeout > 0 {
				waitTimeout = action.PostActionWait.Timeout
			}
			if action.PostActionWait.Selector != nil &&
				action.PostActionWait.Selector.PrimaryCSS != "" {
				waitSelector := action.PostActionWait.Selector.PrimaryCSS
				b.logger.Info("jetski: replay post-action wait for element",
					"job_id", jobID, "selector", waitSelector, "timeout_ms", waitTimeout)
				_, _ = b.WaitForElement(ctx, jobID, waitSelector, waitTimeout)
			} else {
				b.logger.Info("jetski: replay post-action delay",
					"job_id", jobID, "timeout_ms", waitTimeout)
				select {
				case <-time.After(time.Duration(waitTimeout) * time.Millisecond):
				case <-ctx.Done():
					return fmt.Errorf("replay: post-action wait cancelled after step %d: %w",
						i+1, ctx.Err())
				}
			}
		}
	}

	b.logger.Info("jetski: replay chart complete",
		"job_id", jobID,
		"actions_executed", len(keys),
	)
	return nil
}

// isPIIPlaceholder returns true if value is a full PII placeholder like
// {{field_name}}.
func isPIIPlaceholder(value string) bool {
	return strings.HasPrefix(value, "{{") &&
		strings.HasSuffix(value, "}}") &&
		len(value) > 4
}

// extractPlaceholderKey returns the trimmed key inside {{…}}.
func extractPlaceholderKey(value string) string {
	return strings.TrimSpace(value[2 : len(value)-2])
}

// resolveInlinePlaceholders replaces all {{key}} occurrences with values from
// the map. Unknown placeholders are left as-is.
func resolveInlinePlaceholders(value string, values map[string]string) string {
	result := value
	for k, v := range values {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

// tryClickWithFallback attempts click with primary_css, then secondary_xpath,
// then fallback_js. Returns the tier that succeeded, or TierFailed if all failed.
func (b *JetskiBroker) tryClickWithFallback(ctx context.Context, jobID JobID, sel *ChartSelector) SelectorTier {
	if sel == nil {
		return TierFailed
	}

	if sel.PrimaryCSS != "" {
		result, err := b.Click(ctx, jobID, sel.PrimaryCSS)
		if err == nil && (result == nil || result.Success) {
			return TierPrimary
		}
	}

	if sel.SecondaryXPath != "" {
		b.logger.Warn("jetski: primary selector failed, trying secondary_xpath",
			"job_id", jobID, "selector", sel.SecondaryXPath)
		result, err := b.Click(ctx, jobID, sel.SecondaryXPath)
		if err == nil && (result == nil || result.Success) {
			return TierSecondary
		}
	}

	if sel.FallbackJS != "" {
		b.logger.Warn("jetski: secondary selector failed, trying fallback_js",
			"job_id", jobID, "selector", sel.FallbackJS)
		result, err := b.Click(ctx, jobID, sel.FallbackJS)
		if err == nil && (result == nil || result.Success) {
			return TierFallback
		}
	}

	return TierFailed
}

// resolveSelectorWithFallback returns the first non-empty selector and its tier.
// For input actions, the fill itself may still fail — but this resolves which
// selector to try first.
func (b *JetskiBroker) resolveSelectorWithFallback(sel *ChartSelector) (string, SelectorTier) {
	if sel == nil {
		return "", TierFailed
	}
	if sel.PrimaryCSS != "" {
		return sel.PrimaryCSS, TierPrimary
	}
	if sel.SecondaryXPath != "" {
		return sel.SecondaryXPath, TierSecondary
	}
	if sel.FallbackJS != "" {
		return sel.FallbackJS, TierFallback
	}
	return "", TierFailed
}

func primarySelector(sel *ChartSelector, actionKey string, step int) string {
	if sel != nil && sel.PrimaryCSS != "" {
		return sel.PrimaryCSS
	}
	return ""
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

	if wsConn, ok := b.wsConns.LoadAndDelete(id); ok {
		conn := wsConn.(*websocket.Conn)
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "session complete"))
		_ = conn.Close()
	}

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

	if cancelFn, loaded := b.jobContexts.LoadAndDelete(id); loaded {
		cancelFn.(context.CancelFunc)()
	}
	b.jobMeta.Delete(id)
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
