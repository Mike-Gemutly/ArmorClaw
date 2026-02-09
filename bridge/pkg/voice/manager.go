// Package voice provides the voice call manager
// Integrates WebRTC, Matrix, budget, and security for voice calls
package voice

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/budget"
	"github.com/armorclaw/bridge/pkg/webrtc"
	"github.com/armorclaw/bridge/pkg/logger"
	"log/slog"
)

// Manager integrates all voice call components
type Manager struct {
	// Core components
	config      Config
	sessionMgr  *webrtc.SessionManager
	tokenMgr    *webrtc.TokenManager
	webrtcEngine *webrtc.Engine
	turnMgr     *webrtc.TURNManager

	// Voice-specific components
	voiceMgr    *MatrixManager
	budgetMgr   *budget.Manager
	budgetTracker *BudgetTracker
	securityEnforcer *SecurityEnforcer
	ttlManager  *TTLManager
	securityAudit *SecurityAudit

	// State
	calls       sync.Map // map[callID]*MatrixCall
	activeCount int
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	securityLog *logger.SecurityLogger
}

// ManagerConfig holds configuration for the voice manager
type ManagerConfig struct {
	// WebRTC configuration
	WebRTCConfig webrtc.EngineConfig

	// Voice configuration
	VoiceConfig Config

	// Security policy
	SecurityPolicy SecurityPolicy

	// Budget configuration
	BudgetConfig budget.Config
	BudgetManager *budget.Manager

	// Session management
	DefaultLifetime time.Duration
	MaxLifetime     time.Duration

	// TURN configuration
	TURNSharedSecret string
	TURNServerURL    string
}

// NewManager creates a new voice manager
func NewManager(
	sessionMgr *webrtc.SessionManager,
	tokenMgr *webrtc.TokenManager,
	webrtcEngine *webrtc.Engine,
	turnMgr *webrtc.TURNManager,
	config Config,
) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	// Create budget manager if not provided
	budgetMgr := budget.NewManager(budget.Config{
		GlobalTokenLimit:     config.DefaultTokenLimit,
		GlobalDurationLimit:  config.DefaultDurationLimit,
		WarningThreshold:     config.WarningThreshold,
		HardStop:             config.HardStop,
	})

	// Create security enforcer
	securityEnforcer := NewSecurityEnforcer(config.SecurityPolicy)

	// Create budget tracker
	budgetTracker := NewBudgetTracker(config.BudgetConfig, budgetMgr)

	// Create TTL manager
	ttlConfig := DefaultTTLConfig()
	if config.TTLConfig.DefaultTTL > 0 {
		ttlConfig = config.TTLConfig
	}
	ttlManager := NewTTLManager(sessionMgr, ttlConfig)

	// Create security auditor
	securityAudit := NewSecurityAudit(config.SecurityPolicy)

	// Create voice manager
	voiceMgr := NewMatrixManager(config, sessionMgr)

	return &Manager{
		config:         config,
		sessionMgr:     sessionMgr,
		tokenMgr:       tokenMgr,
		webrtcEngine:   webrtcEngine,
		turnMgr:        turnMgr,
		voiceMgr:       voiceMgr,
		budgetMgr:      budgetMgr,
		budgetTracker:  budgetTracker,
		securityEnforcer: securityEnforcer,
		ttlManager:     ttlManager,
		securityAudit:  securityAudit,
		ctx:            ctx,
		cancel:         cancel,
		securityLog:    logger.NewSecurityLogger(logger.Global().WithComponent("voice_manager")),
	}
}

// Start starts the voice manager
func (m *Manager) Start() error {
	// Start TTL enforcement
	if err := m.ttlManager.StartEnforcement(m.ctx); err != nil {
		return fmt.Errorf("failed to start TTL enforcement: %w", err)
	}

	// Start budget enforcement
	if err := m.budgetTracker.StartEnforcement(m.ctx); err != nil {
		return fmt.Errorf("failed to start budget enforcement: %w", err)
	}

	// Start voice manager
	m.voiceMgr.Start()

	m.securityLog.LogSecurityEvent("voice_manager_started",
		slog.Int("max_concurrent_calls", m.config.MaxConcurrentCalls),
		slog.Duration("max_call_duration", m.config.MaxCallDuration),
	)

	return nil
}

// Stop stops the voice manager
func (m *Manager) Stop() {
	// Signal all goroutines to stop
	m.cancel()

	// Stop voice manager
	m.voiceMgr.Stop()

	// Wait for all goroutines
	m.wg.Wait()

	// Clean up all active calls
	m.calls.Range(func(key, value interface{}) bool {
		call := value.(*MatrixCall)
		m.EndCall(call.ID, "manager_shutdown")
		return true
	})

	m.securityLog.LogSecurityEvent("voice_manager_stopped")
}

// HandleMatrixCallEvent handles a Matrix call event
func (m *Manager) HandleMatrixCallEvent(roomID, eventID, senderID string, event *CallEvent) error {
	// Forward to voice manager
	return m.voiceMgr.HandleCallEvent(roomID, eventID, senderID, event)
}

// CreateCall creates a new voice call
func (m *Manager) CreateCall(roomID, offerSDP string, userID string) (*MatrixCall, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check concurrent call limit
	activeCount := m.getActiveCallCount()
	if activeCount >= m.config.MaxConcurrentCalls {
		m.securityLog.LogAccessDenied(m.ctx, "voice_call_create", roomID,
			slog.String("user_id", userID),
			slog.String("reason", "max_concurrent_calls_exceeded"),
			slog.Int("max_calls", m.config.MaxConcurrentCalls))
		return nil, ErrMaxConcurrentCallsExceeded
	}

	// Check security policy
	if err := m.securityEnforcer.CheckStartCall(userID, roomID); err != nil {
		return nil, err
	}

	// Create the call via voice manager
	call, err := m.voiceMgr.StartCall(roomID, offerSDP)
	if err != nil {
		return nil, err
	}

	// Store in active calls
	m.calls.Store(call.ID, call)
	m.activeCount++

	// Register with security enforcer
	m.securityEnforcer.RegisterCall(roomID, userID)

	// Create budget session
	budgetSession, err := m.budgetTracker.StartSession(
		call.ID,
		call.ID,
		roomID,
		m.config.DefaultTokenLimit,
		m.config.DefaultDurationLimit,
	)
	if err != nil {
		// Log but don't fail - budget is optional
		m.securityLog.LogSecurityEvent("voice_budget_session_failed",
			slog.String("call_id", call.ID),
			slog.String("error", err.Error()))
	} else {
		call.mu.Lock()
		call.BudgetSession = budgetSession
		call.mu.Unlock()
	}

	m.securityLog.LogSecurityEvent("voice_call_created",
		slog.String("call_id", call.ID),
		slog.String("room_id", roomID),
		slog.String("user_id", userID),
		slog.Int("active_calls", m.activeCount))

	return call, nil
}

// AnswerCall answers an incoming call
func (m *Manager) AnswerCall(callID, answerSDP string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the call
	call, ok := m.calls.Load(callID)
	if !ok {
		return ErrCallNotFound
	}

	// Answer via voice manager
	if err := m.voiceMgr.AnswerCall(callID, answerSDP); err != nil {
		return err
	}

	matrixCall := call.(*MatrixCall)
	matrixCall.mu.Lock()
	matrixCall.State = CallStateConnected
	matrixCall.AnsweredAt = time.Now()
	matrixCall.mu.Unlock()

	m.securityLog.LogSecurityEvent("voice_call_answered",
		slog.String("call_id", callID))

	return nil
}

// RejectCall rejects an incoming call
func (m *Manager) RejectCall(callID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the call
	call, ok := m.calls.Load(callID)
	if !ok {
		return ErrCallNotFound
	}

	// Reject via voice manager
	if err := m.voiceMgr.RejectCall(callID, reason); err != nil {
		return err
	}

	// Clean up
	m.cleanupCall(callID)

	m.securityLog.LogSecurityEvent("voice_call_rejected",
		slog.String("call_id", callID),
		slog.String("reason", reason))

	return nil
}

// EndCall ends an active call
func (m *Manager) EndCall(callID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the call
	value, ok := m.calls.Load(callID)
	if !ok {
		return ErrCallNotFound
	}

	call := value.(*MatrixCall)

	// End via voice manager
	if err := m.voiceMgr.EndCall(callID, reason); err != nil {
		return err
	}

	// End budget session
	m.budgetTracker.EndSession(callID)

	// Unregister from security
	m.securityEnforcer.UnregisterCall(call.RoomID, call.CallerID)

	// Audit the call
	_, err := m.securityAudit.AuditCall(call)
	if err != nil {
		m.securityLog.LogSecurityEvent("voice_call_audit_failed",
			slog.String("call_id", callID),
			slog.String("error", err.Error()))
	}

	// Clean up
	m.cleanupCall(callID)

	m.securityLog.LogSecurityEvent("voice_call_ended",
		slog.String("call_id", callID),
		slog.String("reason", reason))

	return nil
}

// SendCandidates sends ICE candidates for a call
func (m *Manager) SendCandidates(callID string, candidates []Candidate) error {
	return m.voiceMgr.SendCandidates(callID, candidates)
}

// GetCall retrieves an active call
func (m *Manager) GetCall(callID string) (*MatrixCall, bool) {
	value, ok := m.calls.Load(callID)
	if !ok {
		return nil, false
	}
	return value.(*MatrixCall), true
}

// ListCalls returns all active calls
func (m *Manager) ListCalls() []*MatrixCall {
	calls := make([]*MatrixCall, 0)

	m.calls.Range(func(key, value interface{}) bool {
		call := value.(*MatrixCall)
		calls = append(calls, call)
		return true
	})

	return calls
}

// GetStats returns voice manager statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get base stats from voice manager
	stats := m.voiceMgr.GetStats()

	// Add additional stats
	stats["active_calls"] = m.activeCount
	stats["max_concurrent_calls"] = m.config.MaxConcurrentCalls

	// Add budget stats
	budgetStats := m.budgetTracker.GetStats()
	for k, v := range budgetStats {
		stats["budget_"+k] = v
	}

	// Add TTL stats
	ttlStats := m.ttlManager.GetTTLStats()
	for k, v := range ttlStats {
		stats["ttl_"+k] = v
	}

	return stats
}

// GetAuditLog returns the security audit log
func (m *Manager) GetAuditLog() []AuditRecord {
	return m.securityAudit.GetAuditLog()
}

// GenerateReport generates a security report
func (m *Manager) GenerateReport() *SecurityReport {
	return m.securityAudit.GenerateReport()
}

// cleanupCall removes a call from tracking
func (m *Manager) cleanupCall(callID string) {
	m.calls.Delete(callID)
	m.activeCount--
}

// getActiveCallCount returns the number of active calls
func (m *Manager) getActiveCallCount() int {
	count := 0
	m.calls.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// CreateWebRTCSSession creates a WebRTC session for a call
func (m *Manager) CreateWebRTCSSession(callID, roomID string, ttl time.Duration) (*webrtc.Session, string, error) {
	// Create session
	session := m.sessionMgr.Create(
		fmt.Sprintf("container-%s", callID),
		roomID,
		fmt.Sprintf("user-%s", callID),
		ttl,
	)

	// Generate call token
	token, err := m.tokenMgr.GenerateToken(CallToken{
		SessionID: session.ID,
		RoomID:    roomID,
		ExpiresAt: time.Now().Add(ttl),
	})
	if err != nil {
		m.sessionMgr.End(session.ID)
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return session, token, nil
}

// GetWebRTCSession retrieves a WebRTC session by call ID
func (m *Manager) GetWebRTCSession(callID string) (*webrtc.Session, bool) {
	call, ok := m.GetCall(callID)
	if !ok {
		return nil, false
	}

	sessionID := call.GetSession()
	if sessionID == "" {
		return nil, false
	}

	return m.sessionMgr.Get(sessionID)
}

// ValidateCallToken validates a call session token
func (m *Manager) ValidateCallToken(tokenString string) (*CallToken, error) {
	return m.tokenMgr.ValidateToken(tokenString)
}

// SetTURNManager sets the TURN credential manager
func (m *Manager) SetTURNManager(turnMgr *webrtc.TURNManager) {
	m.turnMgr = turnMgr
	m.webrtcEngine.SetTURNManager(turnMgr)
}
