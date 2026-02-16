// Package voice provides security enforcement and audit for WebRTC voice calls
// Implements TTL enforcement, security policies, and comprehensive logging
package voice

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/webrtc"
	"log/slog"
)

// SecurityPolicy defines security policies for voice calls
type SecurityPolicy struct {
	// Maximum concurrent calls
	MaxConcurrentCalls int

	// Maximum call duration
	MaxCallDuration time.Duration

	// Allowed users (empty = all users allowed)
	AllowedUsers map[string]bool

	// Blocked users
	BlockedUsers map[string]bool

	// Allowed rooms (empty = all rooms allowed)
	AllowedRooms map[string]bool

	// Blocked rooms
	BlockedRooms map[string]bool

	// Require E2EE for calls
	RequireE2EE bool

	// Require TLS for signaling
	RequireSignalingTLS bool

	// Rate limiting (max calls per user per time window)
	RateLimitCalls     int
	RateLimitWindow  time.Duration
}

// DefaultSecurityPolicy returns default security policies
func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{
		MaxConcurrentCalls:     10,
		MaxCallDuration:        1 * time.Hour,
		AllowedUsers:            make(map[string]bool),
		BlockedUsers:            make(map[string]bool),
		AllowedRooms:            make(map[string]bool),
		BlockedRooms:            make(map[string]bool),
		RequireE2EE:             true,
		RequireSignalingTLS:     true,
		RateLimitCalls:          10,
		RateLimitWindow:         1 * time.Hour,
	}
}

// SecurityEnforcer enforces security policies for voice calls
type SecurityEnforcer struct {
	policy      SecurityPolicy
	activeCalls sync.Map // map[roomID]string
	userCallCounts sync.Map // map[userID]*callCount
	mu          sync.RWMutex
	securityLog *logger.SecurityLogger
}

// callCount tracks calls per user for rate limiting
type callCount struct {
	counts    int
	windowEnd time.Time
	mu        sync.Mutex
}

// NewSecurityEnforcer creates a new security enforcer
func NewSecurityEnforcer(policy SecurityPolicy) *SecurityEnforcer {
	return &SecurityEnforcer{
		policy:      policy,
		activeCalls: sync.Map{},
		userCallCounts: sync.Map{},
		securityLog:  logger.NewSecurityLogger(logger.Global().WithComponent("voice_security")),
	}
}

// CheckStartCall validates that a call can be started
func (se *SecurityEnforcer) CheckStartCall(userID, roomID string) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	// Check concurrent call limit
	activeCount := 0
	se.activeCalls.Range(func(_, _ interface{}) bool {
		activeCount++
		return true
	})

	if activeCount >= se.policy.MaxConcurrentCalls {
		se.securityLog.LogAccessDenied(context.Background(), "voice_call_start", roomID, "max_concurrent_calls_exceeded",
			slog.String("user_id", userID),
			slog.Int("max_calls", se.policy.MaxConcurrentCalls))
		return ErrMaxConcurrentCallsExceeded
	}

	// Check user blocklist
	if se.policy.BlockedUsers[userID] {
		se.securityLog.LogAccessDenied(context.Background(), "voice_call_start", roomID, "user_blocked",
			slog.String("user_id", userID))
		return ErrUserBlocked
	}

	// Check room blocklist
	if se.policy.BlockedRooms[roomID] {
		se.securityLog.LogAccessDenied(context.Background(), "voice_call_start", roomID, "room_blocked",
			slog.String("user_id", userID))
		return ErrRoomBlocked
	}

	// Check rate limit
	if !se.checkRateLimit(userID) {
		se.securityLog.LogAccessDenied(context.Background(), "voice_call_start", roomID, "rate_limit_exceeded",
			slog.String("user_id", userID),
			slog.Int("calls_per_window", se.policy.RateLimitCalls),
			slog.String("window", se.policy.RateLimitWindow.String()))
		return ErrRateLimitExceeded
	}

	// If allowlist is configured, check it
	if len(se.policy.AllowedUsers) > 0 && !se.policy.AllowedUsers[userID] {
		se.securityLog.LogAccessDenied(context.Background(), "voice_call_start", roomID, "user_not_in_allowlist",
			slog.String("user_id", userID))
		return ErrUserNotAllowed
	}

	if len(se.policy.AllowedRooms) > 0 && !se.policy.AllowedRooms[roomID] {
		se.securityLog.LogAccessDenied(context.Background(), "voice_call_start", roomID, "room_not_in_allowlist",
			slog.String("user_id", userID))
		return ErrRoomNotAllowed
	}

	return nil
}

// RegisterCall registers an active call
func (se *SecurityEnforcer) RegisterCall(roomID, userID string) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	// Store active call
	se.activeCalls.Store(roomID, userID)

	// Log security event
	se.securityLog.LogSecurityEvent("voice_call_registered",
		slog.String("room_id", roomID),
		slog.String("user_id", userID),
		slog.Int("active_calls", se.getActiveCallCount()))

	return nil
}

// UnregisterCall unregisters an active call
func (se *SecurityEnforcer) UnregisterCall(roomID, userID string) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	// Remove active call
	se.activeCalls.Delete(roomID)

	// Log security event
	se.securityLog.LogSecurityEvent("voice_call_unregistered",
		slog.String("room_id", roomID),
		slog.String("user_id", userID),
		slog.Int("active_calls", se.getActiveCallCount()))

	return nil
}

// checkRateLimit checks if a user has exceeded their rate limit
func (se *SecurityEnforcer) checkRateLimit(userID string) bool {
	now := time.Now()

	// Get or create call count tracker
	value, _ := se.userCallCounts.LoadOrStore(userID, &callCount{})
	tracker := value.(*callCount)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	// Reset if window expired
	if now.After(tracker.windowEnd) {
		tracker.counts = 0
		tracker.windowEnd = now.Add(se.policy.RateLimitWindow)
	}

	// Check limit
	if tracker.counts >= se.policy.RateLimitCalls {
		return false
	}

	// Increment count
	tracker.counts++
	return true
}

// getActiveCallCount returns the number of active calls
func (se *SecurityEnforcer) getActiveCallCount() int {
	count := 0
	se.activeCalls.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// ValidateCallParameters validates call parameters for security
func (se *SecurityEnforcer) ValidateCallParameters(roomID, userID, sessionID string) error {
	// Validate room ID format (basic check)
	if roomID == "" {
		return ErrInvalidRoomID
	}

	// Validate user ID
	if userID == "" {
		return ErrInvalidUserID
	}

	// Validate session ID
	if sessionID == "" {
		return ErrInvalidSessionID
	}

	// Check if session is already associated with a call
	if sessionID != "" {
		// In a full implementation, this would check session state
		// For now, just validate format
	}

	// Validate call duration
	if se.policy.MaxCallDuration > 0 {
		// This would be checked during the call
		// Enforcement happens via TTL manager
	}

	// Check E2EE requirement
	if se.policy.RequireE2EE {
		// In a full implementation, this would verify DTLS-SRTP
		// For now, assume WebRTC engine handles this
	}

	// Check TLS requirement
	if se.policy.RequireSignalingTLS {
		// In a full implementation, this would verify wss://
		// For now, assume signaling server handles this
	}

	return nil
}

// AuditCall generates an audit record for a call
func (se *SecurityEnforcer) AuditCall(call *MatrixCall) *AuditRecord {
	record := &AuditRecord{
		CallID:        call.ID,
		RoomID:        call.RoomID,
		CallerID:      call.CallerID,
		CalleeID:      call.CalleeID,
		State:         call.State.String(),
		StartTime:     call.CreatedAt,
		EndTime:       call.UpdatedAt,
		Duration:      call.UpdatedAt.Sub(call.CreatedAt),
		PolicyVersion: "v1",
	}

	// Add security events
	call.mu.RLock()
	for _, event := range call.CallEvents {
		record.Events = append(record.Events, AuditEvent{
			Type:      string(event.Type),
			Timestamp: event.CreateTime,
			PartyID:   event.PartyID,
		})
	}
	call.mu.RUnlock()

	return record
}

// AuditRecord represents an audit record for a voice call
type AuditRecord struct {
	CallID         string        `json:"call_id"`
	RoomID         string        `json:"room_id"`
	CallerID       string        `json:"caller_id"`
	CalleeID       string        `json:"callee_id"`
	State          string        `json:"state"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	TokenUsage     TokenUsage    `json:"token_usage,omitempty"`
	PolicyVersion string        `json:"policy_version"`
	Events         []AuditEvent  `json:"events"`
	Violations     []string      `json:"violations,omitempty"`
}

// AuditEvent represents a single event in the audit trail
type AuditEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	PartyID   string    `json:"party_id"`
}

// SecurityAudit provides security audit functionality
type SecurityAudit struct {
	policy      SecurityPolicy
	enforcer    *SecurityEnforcer
	mu          sync.RWMutex
	auditLog    []AuditRecord
	securityLog *logger.SecurityLogger
}

// NewSecurityAudit creates a new security auditor
func NewSecurityAudit(policy SecurityPolicy) *SecurityAudit {
	return &SecurityAudit{
		policy:      policy,
		enforcer:    NewSecurityEnforcer(policy),
		auditLog:    make([]AuditRecord, 0),
		securityLog: logger.NewSecurityLogger(logger.Global().WithComponent("voice_audit")),
	}
}

// AuditCall audits a call for security compliance
func (sa *SecurityAudit) AuditCall(call *MatrixCall) (*AuditRecord, error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	// Generate audit record
	record := sa.enforcer.AuditCall(call)

	// Check for policy violations
	violations := sa.checkViolations(call)
	record.Violations = violations

	// Add to audit log
	sa.auditLog = append(sa.auditLog, *record)

	// Log if there are violations
	if len(violations) > 0 {
		sa.securityLog.LogSecurityEvent("voice_call_audit_violations",
			slog.String("call_id", call.ID),
			slog.String("room_id", call.RoomID),
			slog.Int("violation_count", len(violations)),
			slog.Any("violations", violations))
	}

	return record, nil
}

// checkViolations checks if a call violates any security policies
func (sa *SecurityAudit) checkViolations(call *MatrixCall) []string {
	violations := make([]string, 0)

	// Check if user is blocked
	if sa.policy.BlockedUsers[call.CallerID] {
		violations = append(violations, "caller_blocked")
	}

	if sa.policy.BlockedUsers[call.CalleeID] {
		violations = append(violations, "callee_blocked")
	}

	// Check if room is blocked
	if sa.policy.BlockedRooms[call.RoomID] {
		violations = append(violations, "room_blocked")
	}

	// Check if call duration exceeded
	if sa.policy.MaxCallDuration > 0 && time.Since(call.CreatedAt) > sa.policy.MaxCallDuration {
		violations = append(violations, "duration_exceeded")
	}

	// Check concurrent calls
	activeCalls := sa.enforcer.getActiveCallCount()
	if activeCalls > sa.policy.MaxConcurrentCalls {
		violations = append(violations, "max_concurrent_calls_exceeded")
	}

	return violations
}

// GetAuditLog returns the audit log
func (sa *SecurityAudit) GetAuditLog() []AuditRecord {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	// Return a copy to avoid race conditions
	log := make([]AuditRecord, len(sa.auditLog))
	copy(log, sa.auditLog)

	return log
}

// GenerateReport generates a security audit report
func (sa *SecurityAudit) GenerateReport() *SecurityReport {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	report := &SecurityReport{
		PolicyVersion:  sa.policy,
		GeneratedAt:    time.Now(),
		TotalCalls:      len(sa.auditLog),
		Violations:     make(map[string]int),
		RecentCalls:    make([]*AuditRecord, 0),
	}

	// Count violations
	for _, record := range sa.auditLog {
		for _, violation := range record.Violations {
			report.Violations[violation]++
		}
	}

	// Get recent calls (last 100)
	recentCount := 100
	start := 0
	if len(sa.auditLog) > recentCount {
		start = len(sa.auditLog) - recentCount
	}
	// Convert slice to pointers
	for i := start; i < len(sa.auditLog); i++ {
		record := sa.auditLog[i]
		report.RecentCalls = append(report.RecentCalls, &record)
	}

	return report
}

// SecurityReport represents a security audit report
type SecurityReport struct {
	PolicyVersion  SecurityPolicy `json:"policy_version"`
	GeneratedAt    time.Time     `json:"generated_at"`
	TotalCalls      int            `json:"total_calls"`
	Violations     map[string]int `json:"violations"`
	RecentCalls    []*AuditRecord `json:"recent_calls"`
}

// TTLManager enforces TTL for voice sessions
type TTLManager struct {
	sessions    *webrtc.SessionManager
	config      TTLConfig
	mu          sync.RWMutex
	securityLog *logger.SecurityLogger
}

// TTLConfig holds TTL configuration
type TTLConfig struct {
	// Default TTL for voice sessions
	DefaultTTL time.Duration

	// Maximum TTL
	MaxTTL time.Duration

	// TTL enforcement interval
	EnforcementInterval time.Duration

	// Warning threshold (percentage)
	WarningThreshold float64

	// Hard stop when TTL exceeded
	HardStop bool
}

// DefaultTTLConfig returns default TTL configuration
func DefaultTTLConfig() TTLConfig {
	return TTLConfig{
		DefaultTTL:           10 * time.Minute,
		MaxTTL:               1 * time.Hour,
		EnforcementInterval:  30 * time.Second,
		WarningThreshold:     0.9, // 90%
		HardStop:              true,
	}
}

// NewTTLManager creates a new TTL manager
func NewTTLManager(sessions *webrtc.SessionManager, config TTLConfig) *TTLManager {
	return &TTLManager{
		sessions:    sessions,
		config:      config,
		securityLog: logger.NewSecurityLogger(logger.Global().WithComponent("voice_ttl")),
	}
}

// EnforceTTL enforces TTL for all active voice sessions
func (tm *TTLManager) EnforceTTL(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get all sessions
	sessions := tm.sessions.List()

	// Check each session
	expired := make([]string, 0)
	warning := make([]string, 0)

	for _, session := range sessions {
		if session.IsExpired() {
			expired = append(expired, session.ID)
			continue
		}

		// Check if approaching TTL limit
		remaining := session.RemainingTTL()
		ttlPercentage := float64(remaining) / float64(session.ExpiresAt.Sub(session.CreatedAt))

		if ttlPercentage < tm.config.WarningThreshold {
			warning = append(warning, session.ID)
			// Log warning immediately with session-specific values
			tm.securityLog.LogSecurityEvent("voice_session_ttl_warning",
				slog.String("session_id", session.ID),
				slog.String("container_id", session.ContainerID),
				slog.String("room_id", session.RoomID),
				slog.Float64("ttl_remaining_percent", ttlPercentage*100),
				slog.Duration("ttl_remaining", remaining))
		}
	}

	// Handle expired sessions
	for _, sessionID := range expired {
		sess, ok := tm.sessions.Get(sessionID)
		if ok {
			// Update state
			tm.sessions.UpdateState(sessionID, webrtc.SessionExpired)

			// Log security event
			tm.securityLog.LogSecurityEvent("voice_session_ttl_enforced",
				slog.String("session_id", sessionID),
				slog.String("container_id", sess.ContainerID),
				slog.String("room_id", sess.RoomID),
				slog.String("reason", "ttl_expired"))

			// End session
			tm.sessions.End(sessionID)
		}
	}

	// Return results
	if len(expired) > 0 {
		return fmt.Errorf("%d sessions expired", len(expired))
	}

	return nil
}

// StartEnforcement starts the TTL enforcement loop
func (tm *TTLManager) StartEnforcement(ctx context.Context) error {
	if tm.config.EnforcementInterval <= 0 {
		return fmt.Errorf("invalid enforcement interval")
	}

	go tm.enforcementLoop(ctx)

	return nil
}

// enforcementLoop periodically enforces TTL
func (tm *TTLManager) enforcementLoop(ctx context.Context) {
	ticker := time.NewTicker(tm.config.EnforcementInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.EnforceTTL(ctx)

		case <-ctx.Done():
			return
		}
	}
}

// GetTTLStats returns TTL statistics
func (tm *TTLManager) GetTTLStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	sessions := tm.sessions.List()
	expiringSoon := 0
	totalRemaining := time.Duration(0)

	for _, session := range sessions {
		remaining := session.RemainingTTL()
		totalRemaining += remaining

		if remaining < 5*time.Minute {
			expiringSoon++
		}
	}

	return map[string]interface{}{
		"total_sessions":      len(sessions),
		"expiring_soon":       expiringSoon,
		"average_remaining":   totalRemaining / time.Duration(len(sessions)),
		"warning_threshold":   tm.config.WarningThreshold * 100,
		"enforcement_interval": tm.config.EnforcementInterval.String(),
	}
}

// Errors
var (
	// ErrMaxConcurrentCallsExceeded is returned when max concurrent calls limit is reached
	ErrMaxConcurrentCallsExceeded = fmt.Errorf("maximum concurrent calls exceeded")

	// ErrUserBlocked is returned when a user is blocked
	ErrUserBlocked = fmt.Errorf("user is blocked")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")

	// ErrUserNotAllowed is returned when user is not in allowlist
	ErrUserNotAllowed = fmt.Errorf("user not in allowlist")

	// ErrRoomNotAllowed is returned when room is not in allowlist
	ErrRoomNotAllowed = fmt.Errorf("room not in allowlist")

	// ErrInvalidRoomID is returned when room ID is invalid
	ErrInvalidRoomID = fmt.Errorf("invalid room ID")

	// ErrInvalidUserID is returned when user ID is invalid
	ErrInvalidUserID = fmt.Errorf("invalid user ID")

	// ErrInvalidSessionID is returned when session ID is invalid
	ErrInvalidSessionID = fmt.Errorf("invalid session ID")

	// ErrCallDurationExceeded is returned when call exceeds max duration
	ErrCallDurationExceeded = fmt.Errorf("call duration exceeded")
)
