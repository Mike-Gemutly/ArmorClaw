// Package browser provides browser skill integration with agent status.
// This module tracks browser operations and emits status events.
package browser

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/agent"
)

// BrowserStatus represents the current browser state
type BrowserStatus string

const (
	BrowserStatusIdle     BrowserStatus = "idle"
	BrowserStatusNavigating BrowserStatus = "navigating"
	BrowserStatusLoading    BrowserStatus = "loading"
	BrowserStatusReady      BrowserStatus = "ready"
	BrowserStatusError      BrowserStatus = "error"
)

// BrowserSession represents an active browser session
type BrowserSession struct {
	ID           string
	URL          string
	Status       BrowserStatus
	Title        string
	LastActivity time.Time
	Error        string
}

// StatusEmitter defines the interface for emitting status events
type StatusEmitter interface {
	EmitStatus(ctx context.Context, event agent.StatusEvent) error
}

// BrowserSkill provides browser operations with status tracking
type BrowserSkill struct {
	mu           sync.RWMutex
	agentID      string
	session      *BrowserSession
	statusEmitter StatusEmitter
}

// NewBrowserSkill creates a new browser skill
func NewBrowserSkill(agentID string, emitter StatusEmitter) *BrowserSkill {
	return &BrowserSkill{
		agentID:      agentID,
		statusEmitter: emitter,
		session: &BrowserSession{
			Status: BrowserStatusIdle,
		},
	}
}

// Navigate navigates to a URL and emits status events
func (b *BrowserSkill) Navigate(ctx context.Context, targetURL string) error {
	b.mu.Lock()
	b.session.URL = targetURL
	b.session.Status = BrowserStatusNavigating
	b.session.LastActivity = time.Now()
	b.mu.Unlock()

	// Validate URL
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return b.emitError(ctx, "invalid_url", fmt.Sprintf("Invalid URL: %v", err))
	}

	// Emit browsing started status
	b.emitStatus(ctx, agent.StatusBrowsing, "navigating", 10, map[string]interface{}{
		"url":  targetURL,
		"host": parsed.Host,
	})

	// In a real implementation, this would:
	// 1. Send navigation command to browser automation
	// 2. Wait for page load
	// 3. Handle redirects, auth prompts, etc.

	// Simulate navigation completion
	b.mu.Lock()
	b.session.Status = BrowserStatusReady
	b.mu.Unlock()

	b.emitStatus(ctx, agent.StatusBrowsing, "page_loaded", 100, map[string]interface{}{
		"url":   targetURL,
		"title": b.session.Title,
	})

	return nil
}

// WaitForElement waits for an element to appear and emits status
func (b *BrowserSkill) WaitForElement(ctx context.Context, selector string, timeout time.Duration) error {
	b.emitStatus(ctx, agent.StatusBrowsing, "waiting_element", 20, map[string]interface{}{
		"selector": selector,
		"timeout":  timeout.String(),
	})

	// In a real implementation, this would poll for the element

	return nil
}

// FillForm fills a form field and emits status
func (b *BrowserSkill) FillForm(ctx context.Context, fieldSelector string, value string) error {
	b.emitStatus(ctx, agent.StatusFormFilling, "filling_field", 30, map[string]interface{}{
		"field": fieldSelector,
	})

	// In a real implementation, this would:
	// 1. Find the element
	// 2. Clear existing value
	// 3. Type new value
	// 4. Verify the value was set

	b.emitStatus(ctx, agent.StatusFormFilling, "field_filled", 50, map[string]interface{}{
		"field": fieldSelector,
	})

	return nil
}

// Click clicks an element and emits status
func (b *BrowserSkill) Click(ctx context.Context, selector string) error {
	b.emitStatus(ctx, agent.StatusFormFilling, "clicking", 60, map[string]interface{}{
		"selector": selector,
	})

	// In a real implementation, this would click the element

	return nil
}

// WaitForNavigation waits for page navigation after an action
func (b *BrowserSkill) WaitForNavigation(ctx context.Context, timeout time.Duration) error {
	b.emitStatus(ctx, agent.StatusBrowsing, "waiting_navigation", 70, map[string]interface{}{
		"timeout": timeout.String(),
	})

	// In a real implementation, this would wait for URL change

	return nil
}

// SubmitForm submits a form
func (b *BrowserSkill) SubmitForm(ctx context.Context, formSelector string) error {
	b.emitStatus(ctx, agent.StatusFormFilling, "submitting_form", 80, map[string]interface{}{
		"form": formSelector,
	})

	// In a real implementation, this would submit the form

	return nil
}

// WaitForCaptcha waits for captcha solving
func (b *BrowserSkill) WaitForCaptcha(ctx context.Context) error {
	b.emitStatus(ctx, agent.StatusAwaitingCaptcha, "waiting_captcha", 0, map[string]interface{}{
		"url": b.session.URL,
	})

	// In a real implementation, this would:
	// 1. Detect captcha type
	// 2. Either solve automatically or request user intervention
	// 3. Wait for resolution

	return nil
}

// WaitFor2FA waits for 2FA code
func (b *BrowserSkill) WaitFor2FA(ctx context.Context) error {
	b.emitStatus(ctx, agent.StatusAwaiting2FA, "waiting_2fa", 0, map[string]interface{}{
		"url": b.session.URL,
	})

	// In a real implementation, this would:
	// 1. Detect 2FA field
	// 2. Request code from mobile app
	// 3. Wait for code input

	return nil
}

// Complete marks the task as complete
func (b *BrowserSkill) Complete(ctx context.Context) {
	b.emitStatus(ctx, agent.StatusComplete, "task_complete", 100, map[string]interface{}{
		"url": b.session.URL,
	})
}

// Fail marks the task as failed
func (b *BrowserSkill) Fail(ctx context.Context, err error) {
	b.emitError(ctx, "task_failed", err.Error())
}

// GetSession returns the current browser session
func (b *BrowserSkill) GetSession() *BrowserSession {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.session
}

// emitStatus emits a status event
func (b *BrowserSkill) emitStatus(ctx context.Context, status agent.AgentStatus, step string, progress int, details map[string]interface{}) {
	if b.statusEmitter == nil {
		return
	}

	urlStr := ""
	if u, ok := details["url"].(string); ok {
		urlStr = u
	}

	taskID := ""
	if tid, ok := details["task_id"].(string); ok {
		taskID = tid
	}

	event := agent.StatusEvent{
		AgentID:   b.agentID,
		Status:    status,
		Timestamp: time.Now().UnixMilli(),
		Metadata: agent.StatusMetadata{
			URL:      urlStr,
			Step:     step,
			Progress: progress,
			TaskID:   taskID,
		},
	}

	_ = b.statusEmitter.EmitStatus(ctx, event)
}

// emitError emits an error status event
func (b *BrowserSkill) emitError(ctx context.Context, step string, errMsg string) error {
	b.mu.Lock()
	b.session.Status = BrowserStatusError
	b.session.Error = errMsg
	b.mu.Unlock()

	if b.statusEmitter != nil {
		_ = b.statusEmitter.EmitStatus(ctx, agent.StatusEvent{
			AgentID:   b.agentID,
			Status:    agent.StatusError,
			Timestamp: time.Now().UnixMilli(),
			Metadata: agent.StatusMetadata{
				Step:  step,
				Error: errMsg,
			},
		})
	}

	return fmt.Errorf("browser error: %s", errMsg)
}

// CallbackAdapter adapts agent.Integration to StatusEmitter
type CallbackAdapter struct {
	integration *agent.Integration
}

// NewCallbackAdapter creates an adapter from agent.Integration
func NewCallbackAdapter(integration *agent.Integration) *CallbackAdapter {
	return &CallbackAdapter{
		integration: integration,
	}
}

// EmitStatus implements StatusEmitter
func (a *CallbackAdapter) EmitStatus(ctx context.Context, event agent.StatusEvent) error {
	if a.integration == nil {
		return nil
	}

	// Handle different statuses
	switch event.Status {
	case agent.StatusBrowsing:
		return a.integration.StartBrowsing(event.Metadata.URL)
	case agent.StatusFormFilling:
		return a.integration.UpdateProgress(event.Metadata.Step, event.Metadata.Progress)
	case agent.StatusAwaitingCaptcha:
		return a.integration.WaitForCaptcha(ctx)
	case agent.StatusAwaiting2FA:
		return a.integration.WaitFor2FA()
	case agent.StatusComplete:
		return a.integration.CompleteTask()
	case agent.StatusError:
		return a.integration.FailTask(fmt.Errorf("%s", event.Metadata.Error))
	}

	return nil
}

// WrapWithAgentIntegration creates a BrowserSkill with agent integration
func WrapWithAgentIntegration(agentID string, integration *agent.Integration) *BrowserSkill {
	adapter := NewCallbackAdapter(integration)
	return NewBrowserSkill(agentID, adapter)
}
