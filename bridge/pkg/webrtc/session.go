// Package webrtc provides WebRTC voice call support for ArmorClaw
// All WebRTC, audio I/O, and signaling live in the Bridge, not in containers
package webrtc

import (
	"sync"
	"time"
)

// SessionState represents the current state of a voice call session
type SessionState int

const (
	// SessionPending is the initial state when session is created but not yet connected
	SessionPending SessionState = iota
	// SessionActive means the WebRTC peer connection is established and media is flowing
	SessionActive
	// SessionEnded means the session was terminated normally
	SessionEnded
	// SessionFailed means the session terminated due to an error
	SessionFailed
	// SessionExpired means the session was terminated due to TTL expiry
	SessionExpired
)

// String returns the string representation of the session state
func (s SessionState) String() string {
	switch s {
	case SessionPending:
		return "pending"
	case SessionActive:
		return "active"
	case SessionEnded:
		return "ended"
	case SessionFailed:
		return "failed"
	case SessionExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// Session represents a single WebRTC voice call session
// It binds together: Matrix room, Agent container, WebRTC peer, TURN allocation, Budget session
type Session struct {
	ID            string        // Unique session identifier (UUID)
	ContainerID   string        // Associated agent container ID
	RoomID        string        // Matrix room for authorization
	State         SessionState  // Current session state
	CreatedAt     time.Time     // When the session was created
	ExpiresAt     time.Time     // When the session will expire (TTL)
	LastActivity  time.Time     // Last time there was activity in the session
	BudgetSession string        // Associated budget tracking session (empty if not started)

	// WebRTC peer connection management
	PeerConnectionID string     // Identifier for the WebRTC peer connection
	SDPOffer         string     // SDP offer from client
	SDPAnswer        string     // SDP answer from bridge
	RemoteSDP        string     // Remote SDP (offer or answer)

	// TURN allocation
	TURNCredentials *TURNCredentials // TURN credentials for this session

	// Close channel for graceful shutdown
	closeOnce sync.Once
	closeChan chan struct{}
}

// TURNCredentials represents ephemeral TURN credentials for a session
type TURNCredentials struct {
	Username string    // Format: <expiry>:<session_id>
	Password string    // HMAC(secret, username)
	Expires  time.Time // When these credentials expire
	TURNServer string  // TURN server address (host:port)
	STUNServer string  // STUN server address (host:port)
}

// IsActive returns true if the session is in an active state
func (s *Session) IsActive() bool {
	return s.State == SessionActive
}

// IsExpired returns true if the session has passed its TTL
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// MarkActivity updates the LastActivity timestamp to current time
func (s *Session) MarkActivity() {
	s.LastActivity = time.Now()
}

// RemainingTTL returns the remaining time until the session expires
func (s *Session) RemainingTTL() time.Duration {
	return time.Until(s.ExpiresAt)
}

// Close signals the session to close gracefully
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		close(s.closeChan)
	})
}

// Closed returns a channel that's closed when the session is closed
func (s *Session) Closed() <-chan struct{} {
	return s.closeChan
}

// SessionConfig holds configuration for session management
type SessionConfig struct {
	DefaultTTL    time.Duration // Default time-to-live for sessions
	MaxTTL        time.Duration // Maximum allowed TTL
	CleanupInterval time.Duration // How often to check for expired sessions
}

// DefaultSessionConfig returns the default session configuration
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		DefaultTTL:     10 * time.Minute,
		MaxTTL:         1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}
}

// SessionManager manages the lifecycle of all WebRTC voice call sessions
// It is the single source of truth for session state
type SessionManager struct {
	sessions sync.Map // map[string]*Session
	config   SessionConfig
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewSessionManager creates a new session manager with the given configuration
func NewSessionManager(config SessionConfig) *SessionManager {
	if config.DefaultTTL == 0 {
		config = DefaultSessionConfig()
	}

	sm := &SessionManager{
		sessions: sync.Map{},
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	sm.wg.Add(1)
	go sm.cleanupLoop()

	return sm
}

// Create creates a new session and adds it to the manager
func (sm *SessionManager) Create(containerID, roomID string, ttl time.Duration) (*Session, error) {
	// Validate TTL
	if ttl <= 0 {
		ttl = sm.config.DefaultTTL
	}
	if ttl > sm.config.MaxTTL {
		ttl = sm.config.MaxTTL
	}

	// Generate session ID
	sessionID := generateSessionID()

	// Calculate expiry
	now := time.Now()
	expiresAt := now.Add(ttl)

	// Create session
	session := &Session{
		ID:           sessionID,
		ContainerID:  containerID,
		RoomID:       roomID,
		State:        SessionPending,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
		LastActivity: now,
		closeChan:    make(chan struct{}),
	}

	// Store session
	sm.sessions.Store(sessionID, session)

	// Emit session created event (will be logged by caller)
	// logger.LogSecurityEvent("session_created", map[string]interface{}{
	// 	"session_id": sessionID,
	// 	"container_id": containerID,
	// 	"room_id": roomID,
	// 	"ttl": ttl.String(),
	// })

	return session, nil
}

// Get retrieves a session by ID, returning (nil, false) if not found
func (sm *SessionManager) Get(sessionID string) (*Session, bool) {
	if session, ok := sm.sessions.Load(sessionID); ok {
		return session.(*Session), true
	}
	return nil, false
}

// UpdateState updates the state of a session
func (sm *SessionManager) UpdateState(sessionID string, state SessionState) error {
	session, ok := sm.Get(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session.State = state
	session.MarkActivity()

	// Emit state change event
	// logger.LogSecurityEvent("session_state_changed", map[string]interface{}{
	// 	"session_id": sessionID,
	// 	"old_state": session.State.String(),
	// 	"new_state": state.String(),
	// })

	return nil
}

// End terminates a session gracefully
func (sm *SessionManager) End(sessionID string) error {
	session, ok := sm.Get(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	// Update state
	session.State = SessionEnded

	// Close the session
	session.Close()

	// Remove from sessions map
	sm.sessions.Delete(sessionID)

	// Emit session ended event
	// logger.LogSecurityEvent("session_ended", map[string]interface{}{
	// 	"session_id": sessionID,
	// 	"container_id": session.ContainerID,
	// 	"duration": time.Since(session.CreatedAt).String(),
	// })

	return nil
}

// Fail marks a session as failed and removes it
func (sm *SessionManager) Fail(sessionID string, reason string) error {
	session, ok := sm.Get(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	// Update state
	session.State = SessionFailed

	// Close the session
	session.Close()

	// Remove from sessions map
	sm.sessions.Delete(sessionID)

	// Emit session failed event
	// logger.LogSecurityEvent("session_failed", map[string]interface{}{
	// 	"session_id": sessionID,
	// 	"reason": reason,
	// })

	return nil
}

// cleanupLoop runs periodically to clean up expired sessions
func (sm *SessionManager) cleanupLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupExpired()
		case <-sm.stopChan:
			return
		}
	}
}

// cleanupExpired removes all sessions that have expired
func (sm *SessionManager) cleanupExpired() {
	now := time.Now()

	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)

		if now.After(session.ExpiresAt) {
			// Session has expired
			session.State = SessionExpired
			session.Close()
			sm.sessions.Delete(key)

			// Emit session expired event
			// logger.LogSecurityEvent("session_expired", map[string]interface{}{
			// 	"session_id": session.ID,
			// 	"container_id": session.ContainerID,
			// })
		}

		return true
	})
}

// Stop stops the session manager and cleans up all sessions
func (sm *SessionManager) Stop() {
	close(sm.stopChan)
	sm.wg.Wait()

	// Close all remaining sessions
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.Close()
		sm.sessions.Delete(key)
		return true
	})
}

// List returns all active sessions
func (sm *SessionManager) List() []*Session {
	var sessions []*Session

	sm.sessions.Range(func(key, value interface{}) bool {
		sessions = append(sessions, value.(*Session))
		return true
	})

	return sessions
}

// Count returns the number of active sessions
func (sm *SessionManager) Count() int {
	count := 0
	sm.sessions.Range(func(_, _) interface{} bool {
		count++
		return true
	})
	return count
}

// generateSessionID generates a unique session ID using a simple approach
// In production, this should use a cryptographically secure random generator
func generateSessionID() string {
	return "sess_" + randomString(16)
}

// randomString generates a random hex string
func randomString(length int) string {
	// Simple implementation - in production use crypto/rand
	const charset = "0123456789abcdef"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// Errors
var (
	// ErrSessionNotFound is returned when a session ID is not found
	ErrSessionNotFound = &SessionError{Code: "not_found", Message: "session not found"}

	// ErrSessionExpired is returned when attempting to operate on an expired session
	ErrSessionExpired = &SessionError{Code: "expired", Message: "session has expired"}

	// ErrSessionEnded is returned when attempting to operate on an ended session
	ErrSessionEnded = &SessionError{Code: "ended", Message: "session has ended"}
)

// SessionError represents an error related to session management
type SessionError struct {
	Code    string
	Message string
}

func (e *SessionError) Error() string {
	return e.Message
}
