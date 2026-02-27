// Package agent provides integration between agent state management and
// Mobile Secretary components like HITL consent and sealed keystore.
package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/pii"
)

// ErrIntegrationNotInitialized is returned when the integration is not initialized
var ErrIntegrationNotInitialized = errors.New("agent integration not initialized")

// IntegrationConfig configures the agent integration
type IntegrationConfig struct {
	// AgentID is the unique identifier for this agent
	AgentID string

	// StateMachine is the agent's state machine
	StateMachine *StateMachine

	// HITLManager is the HITL consent manager (optional)
	HITLManager *pii.HITLConsentManager
}

// Integration provides integration between agent state and Mobile Secretary components
type Integration struct {
	mu           sync.RWMutex
	agentID      string
	stateMachine *StateMachine
	hitlManager  *pii.HITLConsentManager

	// Callbacks for state changes
	onStatusChange func(ctx context.Context, event StatusEvent) error

	// Active consent request tracking
	activeRequestID string
}

// NewIntegration creates a new agent integration
func NewIntegration(cfg IntegrationConfig) (*Integration, error) {
	if cfg.AgentID == "" {
		return nil, errors.New("agent_id is required")
	}
	if cfg.StateMachine == nil {
		return nil, errors.New("state_machine is required")
	}

	integration := &Integration{
		agentID:      cfg.AgentID,
		stateMachine: cfg.StateMachine,
		hitlManager:  cfg.HITLManager,
	}

	// Subscribe to state machine events
	go integration.processEvents()

	return integration, nil
}

// OnStatusChange sets a callback for status changes
func (i *Integration) OnStatusChange(callback func(ctx context.Context, event StatusEvent) error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.onStatusChange = callback
}

// RequestPIIAccess requests PII access and transitions to AWAITING_APPROVAL
func (i *Integration) RequestPIIAccess(ctx context.Context, profileID string, roomID string, manifest *pii.SkillManifest) (*pii.AccessRequest, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.hitlManager == nil {
		return nil, ErrIntegrationNotInitialized
	}

	// Extract field names from manifest
	fields := make([]string, len(manifest.Variables))
	for idx, field := range manifest.Variables {
		fields[idx] = field.Key
	}

	// Transition to AWAITING_APPROVAL
	err := i.stateMachine.RequestApproval(fields)
	if err != nil {
		return nil, fmt.Errorf("failed to transition to awaiting approval: %w", err)
	}

	// Create HITL access request
	request, err := i.hitlManager.RequestAccess(ctx, manifest, profileID, roomID)
	if err != nil {
		// Revert state on error
		i.stateMachine.Fail(fmt.Errorf("PII access request failed: %w", err))
		return nil, err
	}

	// Track active request
	i.activeRequestID = request.ID

	return request, nil
}

// WaitForPIIApproval waits for PII access approval
func (i *Integration) WaitForPIIApproval(ctx context.Context, requestID string) (*pii.AccessRequest, error) {
	if i.hitlManager == nil {
		return nil, ErrIntegrationNotInitialized
	}

	// Wait for approval
	request, err := i.hitlManager.WaitForApproval(ctx, requestID)
	if err != nil {
		// Transition to error state
		i.stateMachine.Fail(fmt.Errorf("PII approval failed: %w", err))
		return nil, err
	}

	i.mu.Lock()
	i.activeRequestID = ""
	i.mu.Unlock()

	// Check approval status
	if request.Status == pii.StatusApproved {
		// Transition back to FORM_FILLING
		if err := i.stateMachine.Transition(StatusFormFilling, StatusMetadata{
			Step:     "approved",
			Progress: 50,
		}); err != nil {
			return nil, err
		}
	} else if request.Status == pii.StatusRejected {
		// Transition to ERROR
		i.stateMachine.FailWithString("PII access rejected: " + request.RejectionReason)
	}

	return request, nil
}

// StartBrowsing transitions to BROWSING with URL tracking
func (i *Integration) StartBrowsing(url string) error {
	// If already in BROWSING state, just update metadata
	if i.stateMachine.Current() == StatusBrowsing {
		i.stateMachine.ForceTransition(StatusBrowsing, StatusMetadata{
			URL: url,
		})
		return nil
	}
	return i.stateMachine.Transition(StatusBrowsing, StatusMetadata{
		URL: url,
	})
}

// UpdateProgress updates the current task progress
func (i *Integration) UpdateProgress(step string, progress int) error {
	currentStatus := i.stateMachine.Current()

	// Only update progress for active states
	if currentStatus == StatusBrowsing || currentStatus == StatusFormFilling {
		// Use ForceTransition to allow same-state metadata updates
		i.stateMachine.ForceTransition(currentStatus, StatusMetadata{
			Step:     step,
			Progress: progress,
		})
	}

	return nil
}

// WaitForCaptcha transitions to AWAITING_CAPTCHA and returns a channel to wait on
func (i *Integration) WaitForCaptcha(ctx context.Context) error {
	if err := i.stateMachine.Transition(StatusAwaitingCaptcha); err != nil {
		return err
	}

	// In a real implementation, this would integrate with a captcha solving service
	// or notify the mobile app to prompt the user
	return nil
}

// ResolveCaptcha transitions out of AWAITING_CAPTCHA (called when captcha is solved)
func (i *Integration) ResolveCaptcha() error {
	return i.stateMachine.Transition(StatusBrowsing)
}

// WaitFor2FA transitions to AWAITING_2FA
func (i *Integration) WaitFor2FA() error {
	return i.stateMachine.Transition(StatusAwaiting2FA)
}

// Resolve2FA transitions out of AWAITING_2FA with the provided code
func (i *Integration) Resolve2FA(code string) error {
	// Code would be used by the calling code
	_ = code
	return i.stateMachine.Transition(StatusFormFilling)
}

// StartPayment transitions to PROCESSING_PAYMENT
func (i *Integration) StartPayment() error {
	return i.stateMachine.Transition(StatusProcessingPayment)
}

// CompleteTask transitions to COMPLETE
func (i *Integration) CompleteTask() error {
	return i.stateMachine.Transition(StatusComplete)
}

// FailTask transitions to ERROR with an error message
func (i *Integration) FailTask(err error) error {
	return i.stateMachine.Fail(err)
}

// GetStatus returns the current agent status
func (i *Integration) GetStatus() StatusEvent {
	i.mu.RLock()
	defer i.mu.RUnlock()

	event := i.stateMachine.LastEvent()
	if event == nil {
		return StatusEvent{
			AgentID:   i.agentID,
			Status:    StatusOffline,
			Timestamp: time.Now().UnixMilli(),
		}
	}
	return *event
}

// processEvents processes state machine events and triggers callbacks
func (i *Integration) processEvents() {
	events := i.stateMachine.Events()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				// Channel closed, exit
				return
			}

			i.mu.RLock()
			callback := i.onStatusChange
			i.mu.RUnlock()

			if callback != nil {
				// Call callback in goroutine to avoid blocking
				go func(e StatusEvent) {
					_ = callback(context.Background(), e)
				}(event)
			}

		case <-i.stateMachine.Done():
			// State machine shut down
			return
		}
	}
}

// AgentCoordinator coordinates multiple agents and their status
type AgentCoordinator struct {
	mu          sync.RWMutex
	integrations map[string]*Integration
}

// NewAgentCoordinator creates a new agent coordinator
func NewAgentCoordinator() *AgentCoordinator {
	return &AgentCoordinator{
		integrations: make(map[string]*Integration),
	}
}

// RegisterAgent registers a new agent with the coordinator
func (c *AgentCoordinator) RegisterAgent(agentID string, sm *StateMachine) (*Integration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.integrations[agentID]; exists {
		return nil, fmt.Errorf("agent %s already registered", agentID)
	}

	integration, err := NewIntegration(IntegrationConfig{
		AgentID:      agentID,
		StateMachine: sm,
	})
	if err != nil {
		return nil, err
	}

	c.integrations[agentID] = integration
	return integration, nil
}

// UnregisterAgent removes an agent from the coordinator
func (c *AgentCoordinator) UnregisterAgent(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if integration, exists := c.integrations[agentID]; exists {
		integration.stateMachine.Shutdown()
		delete(c.integrations, agentID)
	}
}

// GetAgent returns an agent integration by ID
func (c *AgentCoordinator) GetAgent(agentID string) (*Integration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	integration, exists := c.integrations[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}
	return integration, nil
}

// GetAllStatuses returns the status of all registered agents
func (c *AgentCoordinator) GetAllStatuses() []StatusEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statuses := make([]StatusEvent, 0, len(c.integrations))
	for _, integration := range c.integrations {
		statuses = append(statuses, integration.GetStatus())
	}
	return statuses
}

// BroadcastStatus sends status updates to all listeners
func (c *AgentCoordinator) BroadcastStatus(ctx context.Context, event StatusEvent) error {
	// In a real implementation, this would:
	// 1. Send to Matrix room
	// 2. Update mobile app via push notification
	// 3. Update web dashboard via WebSocket

	// For now, we just log the event
	// This would be replaced with actual Matrix/Matrix SDK integration
	return nil
}
