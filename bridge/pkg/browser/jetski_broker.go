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
