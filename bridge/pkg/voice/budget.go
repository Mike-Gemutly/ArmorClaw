// Package voice provides budget enforcement for WebRTC voice calls
// Tracks token usage, duration, and enforces limits mid-call
package voice

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/budget"
	"github.com/armorclaw/bridge/pkg/logger"
	"log/slog"
)

// SessionType represents the type of session (text or voice)
type SessionType string

const (
	// SessionTypeText is a text/chat session
	SessionTypeText SessionType = "text"
	// SessionTypeVoice is a voice call session
	SessionTypeVoice SessionType = "voice"
)

// VoiceSessionTracker tracks a single voice call session's resource usage
type VoiceSessionTracker struct {
	SessionID      string
	CallID         string
	RoomID         string
	StartTime      time.Time
	EndTime        time.Time
	Type           SessionType
	TokenUsage     TokenUsage
	mu             sync.RWMutex
	closed         bool
	budgetSession  *budget.Session
	securityLog    *logger.SecurityLogger
}

// TokenUsage tracks token consumption during a call
type TokenUsage struct {
	InputTokens  uint64 // Tokens from user speech
	OutputTokens uint64 // Tokens for TTS responses
	Model         string // AI model used
	Requests      uint32 // Number of API requests
}

// BudgetTracker manages budget tracking for voice calls
type BudgetTracker struct {
	sessions     sync.Map // map[sessionID]*VoiceSessionTracker
	config       Config
	mu           sync.RWMutex
	budgetMgr    *budget.Manager
	securityLog  *logger.SecurityLogger
}

// Config holds configuration for voice budget tracking
type Config struct {
	// Default token limit per call
	DefaultTokenLimit uint64

	// Default duration limit per call
	DefaultDurationLimit time.Duration

	// Warning threshold (percentage)
	WarningThreshold float64

	// Hard stop when limits exceeded
	HardStop bool
}

// DefaultConfig returns default voice budget configuration
func DefaultConfig() Config {
	return Config{
		DefaultTokenLimit:     100000,  // 100k tokens per call
		DefaultDurationLimit:  30 * time.Minute,
		WarningThreshold:    0.8,     // 80%
		HardStop:             true,    // Enforce hard limit
	}
}

// NewBudgetTracker creates a new voice budget tracker
func NewBudgetTracker(config Config, budgetMgr *budget.Manager) *BudgetTracker {
	return &BudgetTracker{
		config:     config,
		sessions:   sync.Map{},
		budgetMgr:  budgetMgr,
		securityLog: logger.NewSecurityLogger(logger.Global().WithComponent("voice_budget")),
	}
}

// StartSession starts tracking a voice session
func (bt *BudgetTracker) StartSession(sessionID, callID, roomID string, tokenLimit uint64, durationLimit time.Duration) (*VoiceSessionTracker, error) {
	// Set defaults if not specified
	if tokenLimit == 0 {
		tokenLimit = bt.config.DefaultTokenLimit
	}
	if durationLimit == 0 {
		durationLimit = bt.config.DefaultDurationLimit
	}

	// Create budget session
	budgetSess, err := bt.budgetMgr.CreateSession(sessionID, tokenLimit, durationLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget session: %w", err)
	}

	tracker := &VoiceSessionTracker{
		SessionID:     sessionID,
		CallID:        callID,
		RoomID:        roomID,
		StartTime:     time.Now(),
		Type:          SessionTypeVoice,
		budgetSession: budgetSess,
		securityLog:   bt.securityLog,
	}

	bt.sessions.Store(sessionID, tracker)

	// Log security event
	bt.securityLog.LogSecurityEvent("voice_session_started",
		slog.String("session_id", sessionID),
		slog.String("call_id", callID),
		slog.String("room_id", roomID),
		slog.Uint64("token_limit", tokenLimit),
		slog.String("duration_limit", durationLimit.String()))

	return tracker, nil
}

// GetSession retrieves a tracker session
func (bt *BudgetTracker) GetSession(sessionID string) (*VoiceSessionTracker, bool) {
	if sess, ok := bt.sessions.Load(sessionID); ok {
		return sess.(*VoiceSessionTracker), true
	}
	return nil, false
}

// EndSession ends tracking and finalizes budget
func (bt *BudgetTracker) EndSession(sessionID string) error {
	sess, exists := bt.sessions.Load(sessionID)
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	tracker := sess.(*VoiceSessionTracker)
	tracker.mu.Lock()
	tracker.closed = true
	tracker.EndTime = time.Now()
	tracker.mu.Unlock()

	// Finalize budget session
	duration := tracker.EndTime.Sub(tracker.StartTime)
	bt.budgetMgr.FinalizeSession(sessionID, duration, tracker.TokenUsage)

	// Log security event
	bt.securityLog.LogSecurityEvent("voice_session_ended",
		slog.String("session_id", sessionID),
		slog.String("call_id", tracker.CallID),
		slog.String("duration", duration.String()),
		slog.Uint64("input_tokens", tracker.TokenUsage.InputTokens),
		slog.Uint64("output_tokens", tracker.TokenUsage.OutputTokens))

	// Remove from active sessions
	bt.sessions.Delete(sessionID)

	return nil
}

// RecordTokenUsage records token consumption for a session
func (bt *BudgetTracker) RecordTokenUsage(sessionID string, inputTokens, outputTokens uint64, model string) error {
	sess, exists := bt.sessions.Load(sessionID)
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	tracker := sess.(*VoiceSessionTracker)
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if tracker.closed {
		return fmt.Errorf("session is closed")
	}

	// Update token usage
	tracker.TokenUsage.InputTokens += inputTokens
	tracker.TokenUsage.OutputTokens += outputTokens
	tracker.TokenUsage.Model = model
	tracker.TokenUsage.Requests++

	// Update budget session
	totalTokens := inputTokens + outputTokens
	bt.budgetMgr.RecordUsage(sessionID, totalTokens)

	// Check if limits exceeded
	tracker.budgetSession.mu.RLock()
	defer tracker.budgetSession.mu.RUnlock()

	// Check token limit
	if tracker.budgetSession.TokenLimit > 0 {
		percentage := float64(totalTokens) / float64(tracker.budgetSession.TokenLimit)

		if percentage >= 1.0 {
			if bt.config.HardStop {
				tracker.budgetSession.mu.RUnlock()
				bt.EndSession(sessionID)
				tracker.budgetSession.mu.RLock()
				return ErrBudgetExceeded
			}
		} else if percentage >= bt.config.WarningThreshold {
			// Emit warning
			bt.securityLog.LogSecurityEvent("voice_budget_warning",
				slog.String("session_id", sessionID),
				slog.String("call_id", tracker.CallID),
				slog.Float64("usage_percent", percentage*100),
				slog.Uint64("tokens_used", totalTokens),
				slog.Uint64("tokens_remaining", tracker.budgetSession.TokenLimit-totalTokens))
		}
	}

	return nil
}

// CheckDuration checks if the session has exceeded its duration limit
func (bt *BudgetTracker) CheckDuration(sessionID string) error {
	sess, exists := bt.sessions.Load(sessionID)
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	tracker := sess.(*VoiceSessionTracker)
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	if tracker.closed {
		return fmt.Errorf("session is closed")
	}

	// Check duration
	duration := time.Since(tracker.StartTime)
	tracker.budgetSession.mu.RLock()
	defer tracker.budgetSession.mu.RUnlock()

	if tracker.budgetSession.DurationLimit > 0 && duration > tracker.budgetSession.DurationLimit {
		bt.EndSession(sessionID)
		return ErrDurationExceeded
	}

	return nil
}

// GetUsage returns current usage statistics for a session
func (bt *BudgetTracker) GetUsage(sessionID string) (*TokenUsage, time.Duration, error) {
	sess, exists := bt.sessions.Load(sessionID)
	if !exists {
		return nil, 0, fmt.Errorf("session %s not found", sessionID)
	}

	tracker := sess.(*VoiceSessionTracker)
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	var duration time.Duration
	if tracker.closed {
		duration = tracker.EndTime.Sub(tracker.StartTime)
	} else {
		duration = time.Since(tracker.StartTime)
	}

	return &tracker.TokenUsage, duration, nil
}

// GetSessionState returns the current state of a session
func (bt *BudgetTracker) GetSessionState(sessionID string) (SessionState, error) {
	sess, exists := bt.sessions.Load(sessionID)
	if !exists {
		return SessionStateUnknown, fmt.Errorf("session %s not found", sessionID)
	}

	tracker := sess.(*VoiceSessionTracker)
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	if tracker.closed {
		return SessionStateEnded, nil
	}

	return SessionStateActive, nil
}

// GetAllSessions returns all active sessions
func (bt *BudgetTracker) GetAllSessions() []*VoiceSessionTracker {
	sessions := make([]*VoiceSessionTracker, 0)

	bt.sessions.Range(func(key, value interface{}) bool {
		tracker := value.(*VoiceSessionTracker)
		tracker.mu.RLock()
		if !tracker.closed {
			sessions = append(sessions, tracker)
		}
		tracker.mu.RUnlock()
		return true
	})

	return sessions
}

// GetStats returns budget tracker statistics
func (bt *BudgetTracker) GetStats() map[string]interface{} {
	var totalTokensUsed uint64
	var totalDuration time.Duration
	activeSessions := 0

	bt.sessions.Range(func(key, value interface{}) bool {
		tracker := value.(*VoiceSessionTracker)
		tracker.mu.RLock()
		defer tracker.mu.RUnlock()

		if !tracker.closed {
			activeSessions++
			totalTokensUsed += tracker.TokenUsage.InputTokens + tracker.TokenUsage.OutputTokens
			if !tracker.EndTime.IsZero() {
				totalDuration += tracker.EndTime.Sub(tracker.StartTime)
			} else {
				totalDuration += time.Since(tracker.StartTime)
			}
		}
		return true
	})

	return map[string]interface{}{
		"active_sessions":     activeSessions,
		"total_tokens_used":    totalTokensUsed,
		"total_duration":       totalDuration.String(),
		"warning_threshold":    bt.config.WarningThreshold * 100,
		"hard_stop_enabled":    bt.config.HardStop,
	}
}

// SessionState represents the state of a voice session
type SessionState int

const (
	SessionStateUnknown SessionState = iota
	SessionStateActive
	SessionStateEnded
)

// String returns the string representation of the session state
func (s SessionState) String() string {
	switch s {
	case SessionStateUnknown:
		return "unknown"
	case SessionStateActive:
		return "active"
	case SessionStateEnded:
		return "ended"
	default:
		return "unknown"
	}
}

// Stop stops the budget tracker
func (bt *BudgetTracker) Stop() {
	bt.sessions.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		bt.EndSession(sessionID)
		return true
	})
}

// EnforceLimits enforces budget limits across all active sessions
func (bt *BudgetTracker) EnforceLimits(ctx context.Context) error {
	// Check all active sessions for limit violations
	bt.sessions.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		tracker := value.(*VoiceSessionTracker)

		tracker.mu.RLock()
		if tracker.closed {
			tracker.mu.RUnlock()
			return true
		}
		tracker.mu.RUnlock()

		// Check token limit
		tracker.budgetSession.mu.RLock()
		totalTokens := tracker.TokenUsage.InputTokens + tracker.TokenUsage.OutputTokens
		tokenLimit := tracker.budgetSession.TokenLimit

		if tokenLimit > 0 && totalTokens >= tokenLimit {
			tracker.budgetSession.mu.RUnlock()
			bt.EndSession(sessionID)
			bt.securityLog.LogSecurityEvent("voice_budget_enforced",
				slog.String("session_id", sessionID),
				slog.String("call_id", tracker.CallID),
				slog.String("reason", "token_limit_exceeded"),
				slog.Uint64("tokens_used", totalTokens),
				slog.Uint64("token_limit", tokenLimit))
			return true
		}
		tracker.budgetSession.mu.RUnlock()

		// Check duration limit
		duration := time.Since(tracker.StartTime)
		durationLimit := tracker.budgetSession.DurationLimit

		if durationLimit > 0 && duration > durationLimit {
			bt.EndSession(sessionID)
			bt.securityLog.LogSecurityEvent("voice_duration_enforced",
				slog.String("session_id", sessionID),
				slog.String("call_id", tracker.CallID),
				slog.String("reason", "duration_limit_exceeded"),
				slog.String("duration", duration.String()),
				slog.String("duration_limit", durationLimit.String()))
			return true
		}

		return true
	})

	return nil
}

// StartEnforcement starts the background enforcement goroutine
func (bt *BudgetTracker) StartEnforcement(ctx context.Context) error {
	// Check if budget manager is available
	if bt.budgetMgr == nil {
		return fmt.Errorf("budget manager not configured")
	}

	// Start enforcement loop
	go bt.enforcementLoop(ctx)

	return nil
}

// enforcementLoop periodically enforces budget limits
func (bt *BudgetTracker) enforcementLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bt.EnforceLimits(ctx)

		case <-ctx.Done():
			return
		}
	}
}

// Errors
var (
	// ErrBudgetExceeded is returned when token budget is exceeded
	ErrBudgetExceeded = fmt.Errorf("voice budget exceeded")

	// ErrDurationExceeded is returned when duration limit is exceeded
	ErrDurationExceeded = fmt.Errorf("voice call duration exceeded")

	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = fmt.Errorf("voice session not found")
)
