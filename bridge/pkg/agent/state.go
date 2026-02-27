// Package agent provides agent management functionality for ArmorClaw,
// including state machine management for tracking agent operational status.
package agent

import (
	"fmt"
	"time"
)

// AgentStatus represents the operational state of a secretary agent.
type AgentStatus string

const (
	// StatusIdle indicates the agent is not performing any task.
	StatusIdle AgentStatus = "IDLE"
	// StatusInitializing indicates the agent is starting up.
	StatusInitializing AgentStatus = "INITIALIZING"
	// StatusBrowsing indicates the agent is navigating to a URL.
	StatusBrowsing AgentStatus = "BROWSING"
	// StatusFormFilling indicates the agent is filling form fields.
	StatusFormFilling AgentStatus = "FORM_FILLING"
	// StatusAwaitingCaptcha indicates the agent needs human CAPTCHA solving.
	StatusAwaitingCaptcha AgentStatus = "AWAITING_CAPTCHA"
	// StatusAwaiting2FA indicates the agent needs a 2FA code.
	StatusAwaiting2FA AgentStatus = "AWAITING_2FA"
	// StatusAwaitingApproval indicates the agent is waiting for BlindFill approval.
	StatusAwaitingApproval AgentStatus = "AWAITING_APPROVAL"
	// StatusProcessingPayment indicates the agent is submitting a payment.
	StatusProcessingPayment AgentStatus = "PROCESSING_PAYMENT"
	// StatusError indicates the agent encountered a recoverable error.
	StatusError AgentStatus = "ERROR"
	// StatusComplete indicates the task finished successfully.
	StatusComplete AgentStatus = "COMPLETE"
	// StatusOffline indicates the agent is not reachable.
	StatusOffline AgentStatus = "OFFLINE"
)

// ValidTransitions defines allowed state transitions.
// Each state maps to a slice of states it can transition to.
var ValidTransitions = map[AgentStatus][]AgentStatus{
	StatusIdle: {
		StatusInitializing,
		StatusError,
	},
	StatusInitializing: {
		StatusBrowsing,
		StatusFormFilling,
		StatusError,
		StatusIdle,
	},
	StatusBrowsing: {
		StatusFormFilling,
		StatusAwaitingCaptcha,
		StatusAwaiting2FA,
		StatusAwaitingApproval,
		StatusError,
		StatusComplete,
		StatusIdle,
	},
	StatusFormFilling: {
		StatusAwaitingApproval,
		StatusProcessingPayment,
		StatusAwaitingCaptcha,
		StatusAwaiting2FA,
		StatusError,
		StatusComplete,
		StatusBrowsing,
		StatusIdle,
	},
	StatusAwaitingCaptcha: {
		StatusBrowsing,
		StatusFormFilling,
		StatusError,
		StatusIdle,
	},
	StatusAwaiting2FA: {
		StatusBrowsing,
		StatusFormFilling,
		StatusProcessingPayment,
		StatusError,
		StatusIdle,
	},
	StatusAwaitingApproval: {
		StatusFormFilling,
		StatusProcessingPayment,
		StatusError,
		StatusIdle,
	},
	StatusProcessingPayment: {
		StatusComplete,
		StatusError,
		StatusAwaiting2FA,
		StatusIdle,
	},
	StatusError: {
		StatusIdle,
		StatusInitializing,
	},
	StatusComplete: {
		StatusIdle,
	},
	StatusOffline: {
		StatusInitializing,
	},
}

// IsTerminal returns true if this is a terminal state (requires external action to leave).
func (s AgentStatus) IsTerminal() bool {
	return s == StatusAwaitingCaptcha ||
		s == StatusAwaiting2FA ||
		s == StatusAwaitingApproval ||
		s == StatusOffline
}

// IsActive returns true if the agent is actively working.
func (s AgentStatus) IsActive() bool {
	return s == StatusBrowsing ||
		s == StatusFormFilling ||
		s == StatusInitializing ||
		s == StatusProcessingPayment
}

// NeedsUserAction returns true if the state requires user intervention.
func (s AgentStatus) NeedsUserAction() bool {
	return s == StatusAwaitingCaptcha ||
		s == StatusAwaiting2FA ||
		s == StatusAwaitingApproval
}

// StatusMetadata contains additional context about the current status.
type StatusMetadata struct {
	// URL is the current page being browsed.
	URL string `json:"url,omitempty"`
	// Step describes the current step (e.g., "2/5").
	Step string `json:"step,omitempty"`
	// Progress is a percentage (0-100).
	Progress int `json:"progress,omitempty"`
	// Error contains error details if status is ERROR.
	Error string `json:"error,omitempty"`
	// TaskID identifies the current task.
	TaskID string `json:"task_id,omitempty"`
	// TaskType describes the task (e.g., "flight_booking").
	TaskType string `json:"task_type,omitempty"`
	// FieldsRequested lists PII fields being requested (for AWAITING_APPROVAL).
	FieldsRequested []string `json:"fields_requested,omitempty"`
}

// StatusEvent represents a status change event for Matrix.
type StatusEvent struct {
	// AgentID identifies the agent.
	AgentID string `json:"agent_id"`
	// Status is the new status.
	Status AgentStatus `json:"status"`
	// Previous is the previous status (optional).
	Previous AgentStatus `json:"previous,omitempty"`
	// Metadata contains additional context.
	Metadata StatusMetadata `json:"metadata,omitempty"`
	// Timestamp is when the event occurred (Unix milliseconds).
	Timestamp int64 `json:"timestamp"`
}

// EventType returns the Matrix event type for agent status.
func (e StatusEvent) EventType() string {
	return "com.armorclaw.agent.status"
}

// ToMatrixContent converts the event to Matrix event content.
func (e StatusEvent) ToMatrixContent() map[string]interface{} {
	return map[string]interface{}{
		"agent_id":  e.AgentID,
		"status":    string(e.Status),
		"previous":  string(e.Previous),
		"metadata":  e.Metadata,
		"timestamp": e.Timestamp,
	}
}

// StatusEventFromMatrix parses a Matrix event into a StatusEvent.
func StatusEventFromMatrix(content map[string]interface{}) (*StatusEvent, error) {
	event := &StatusEvent{}

	if agentID, ok := content["agent_id"].(string); ok {
		event.AgentID = agentID
	}
	if status, ok := content["status"].(string); ok {
		event.Status = AgentStatus(status)
	}
	if previous, ok := content["previous"].(string); ok {
		event.Previous = AgentStatus(previous)
	}
	if timestamp, ok := content["timestamp"].(float64); ok {
		event.Timestamp = int64(timestamp)
	}

	if metadata, ok := content["metadata"].(map[string]interface{}); ok {
		if url, ok := metadata["url"].(string); ok {
			event.Metadata.URL = url
		}
		if step, ok := metadata["step"].(string); ok {
			event.Metadata.Step = step
		}
		if progress, ok := metadata["progress"].(float64); ok {
			event.Metadata.Progress = int(progress)
		}
		if errMsg, ok := metadata["error"].(string); ok {
			event.Metadata.Error = errMsg
		}
		if taskID, ok := metadata["task_id"].(string); ok {
			event.Metadata.TaskID = taskID
		}
		if taskType, ok := metadata["task_type"].(string); ok {
			event.Metadata.TaskType = taskType
		}
		if fields, ok := metadata["fields_requested"].([]interface{}); ok {
			for _, f := range fields {
				if s, ok := f.(string); ok {
					event.Metadata.FieldsRequested = append(event.Metadata.FieldsRequested, s)
				}
			}
		}
	}

	return event, nil
}

// ValidateTransition checks if a transition from current to next is valid.
func ValidateTransition(current, next AgentStatus) error {
	allowed, exists := ValidTransitions[current]
	if !exists {
		return fmt.Errorf("invalid current state: %s", current)
	}

	for _, s := range allowed {
		if s == next {
			return nil
		}
	}

	return fmt.Errorf("invalid transition: %s -> %s (allowed: %v)", current, next, allowed)
}

// NewStatusEvent creates a new status event with the current timestamp.
func NewStatusEvent(agentID string, status, previous AgentStatus, metadata StatusMetadata) *StatusEvent {
	return &StatusEvent{
		AgentID:   agentID,
		Status:    status,
		Previous:  previous,
		Metadata:  metadata,
		Timestamp: time.Now().UnixMilli(),
	}
}
