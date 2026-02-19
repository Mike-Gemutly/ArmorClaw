// Package appservice provides Matrix Application Service functionality for ArmorClaw.
// The Bridge operates as an AppService to bridge external platforms (SDTW) to Matrix.
//
// This replaces the previous "user proxy" model with a proper AppService model:
// - Clients connect directly to the Matrix Homeserver (Zero-Trust)
// - Bridge acts as an AppService for SDTW platform bridging
// - No user crypto is handled by the server
package appservice

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	errsys "github.com/armorclaw/bridge/pkg/errors"
)

// Component tracker for error handling
var appserviceTracker = errsys.GetComponentTracker("appservice")

// Event represents a Matrix event received from the homeserver
type Event struct {
	Type           string                 `json:"type"`
	RoomID         string                 `json:"room_id"`
	Sender         string                 `json:"sender"`
	Content        map[string]interface{} `json:"content"`
	EventID        string                 `json:"event_id"`
	OriginServerTS int64                  `json:"origin_server_ts"`
	StateKey       *string                `json:"state_key,omitempty"`
	Unsigned       map[string]interface{} `json:"unsigned,omitempty"`
}

// Transaction represents a batch of events from the homeserver
type Transaction struct {
	Events []Event `json:"events"`
}

// UserProfile is used for ghost user creation
type UserProfile struct {
	DisplayName string `json:"displayname,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

// Config holds AppService configuration
type Config struct {
	// Homeserver configuration
	HomeserverURL string `json:"homeserver_url"`
	ServerName    string `json:"server_name"`

	// AppService identity
	ID              string `json:"id"`
	URL             string `json:"url"`              // Where homeserver sends events
	ASToken         string `json:"as_token"`         // AppService token (from registration)
	HSToken         string `json:"hs_token"`         // Homeserver token (for verification)
	SenderLocalpart string `json:"sender_localpart"` // e.g., "_bridge"

	// Network configuration
	ListenAddress string `json:"listen_address"`
	ListenPort    int    `json:"listen_port"`

	// Rate limiting
	MaxTransactionsPerSecond int `json:"max_transactions_per_second"`

	// Event handlers
	OnEvent     func(Event) error     `json:"-"`
	OnUserQuery func(userID string) (*UserProfile, error) `json:"-"`
	OnRoomQuery func(roomAlias string) (roomID string, err error) `json:"-"`
}

// AppService implements the Matrix Application Service protocol
type AppService struct {
	config Config
	server *http.Server
	logger *slog.Logger

	// Event processing
	eventChan   chan Event
	eventBuffer []Event
	bufferMu    sync.Mutex

	// Ghost user tracking
	ghostUsers map[string]*GhostUser
	ghostMu    sync.RWMutex

	// Rate limiting
	transactionCount int
	rateLimitReset   time.Time
	rateLimitMu      sync.Mutex

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// GhostUser represents a bridged user from an external platform
type GhostUser struct {
	UserID      string    `json:"user_id"`
	Platform    string    `json:"platform"`    // slack, discord, teams, whatsapp
	ExternalID  string    `json:"external_id"` // Original ID on platform
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	LastActive  time.Time `json:"last_active"`
}

// New creates a new AppService instance
func New(config Config) (*AppService, error) {
	if config.HomeserverURL == "" {
		return nil, fmt.Errorf("homeserver URL is required")
	}
	if config.ASToken == "" || config.HSToken == "" {
		return nil, fmt.Errorf("AS token and HS token are required")
	}
	if config.ListenAddress == "" {
		config.ListenAddress = "0.0.0.0"
	}
	if config.ListenPort == 0 {
		config.ListenPort = 9999
	}
	if config.MaxTransactionsPerSecond == 0 {
		config.MaxTransactionsPerSecond = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	as := &AppService{
		config:      config,
		eventChan:   make(chan Event, 1000),
		eventBuffer: make([]Event, 0),
		ghostUsers:  make(map[string]*GhostUser),
		ctx:         ctx,
		cancel:      cancel,
		logger: slog.Default().With(
			"component", "appservice",
			"id", config.ID,
		),
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/transactions/", as.handleTransaction)
	mux.HandleFunc("/users/", as.handleUserQuery)
	mux.HandleFunc("/rooms/", as.handleRoomQuery)
	mux.HandleFunc("/health", as.handleHealth)

	as.server = &http.Server{
		Addr:         net.JoinHostPort(config.ListenAddress, strconv.Itoa(config.ListenPort)),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return as, nil
}

// Start begins listening for events from the homeserver
func (as *AppService) Start() error {
	as.logger.Info("starting_appservice",
		"address", as.server.Addr,
		"homeserver", as.config.HomeserverURL,
	)

	as.wg.Add(1)
	go func() {
		defer as.wg.Done()
		if err := as.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			as.logger.Error("server_error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the AppService
func (as *AppService) Stop() error {
	as.logger.Info("stopping_appservice")
	as.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := as.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	as.wg.Wait()
	close(as.eventChan)

	return nil
}

// Events returns a channel for receiving Matrix events
func (as *AppService) Events() <-chan Event {
	return as.eventChan
}

// handleTransaction handles PUT /transactions/{txnId}
func (as *AppService) handleTransaction(w http.ResponseWriter, r *http.Request) {
	// Verify method
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify HS token
	if !as.verifyToken(r) {
		appserviceTracker.Event("auth_failed", nil)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Rate limiting
	if !as.checkRateLimit() {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}

	// Parse transaction
	var txn Transaction
	if err := json.NewDecoder(r.Body).Decode(&txn); err != nil {
		appserviceTracker.Event("parse_error", nil)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Process events
	for _, event := range txn.Events {
		// Filter: Only process events not from our ghost users
		if as.isGhostUser(event.Sender) {
			continue // Skip our own events
		}

		// Record for tracking
		appserviceTracker.Event("event_received", event.EventID)

		// Send to event channel (non-blocking)
		select {
		case as.eventChan <- event:
		default:
			// Buffer full, log warning
			as.logger.Warn("event_buffer_full", "event_id", event.EventID)
			as.bufferEvent(event)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleUserQuery handles GET /users/{userId}
func (as *AppService) handleUserQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !as.verifyToken(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID := strings.TrimPrefix(r.URL.Path, "/users/")

	// Check if handler is registered
	if as.config.OnUserQuery != nil {
		profile, err := as.config.OnUserQuery(userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(profile)
		return
	}

	// Default: Return empty profile (user exists)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(UserProfile{})
}

// handleRoomQuery handles GET /rooms/{roomAlias}
func (as *AppService) handleRoomQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !as.verifyToken(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomAlias := strings.TrimPrefix(r.URL.Path, "/rooms/")

	if as.config.OnRoomQuery != nil {
		roomID, err := as.config.OnRoomQuery(roomAlias)
		if err != nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"room_id": roomID})
		return
	}

	http.Error(w, "room not found", http.StatusNotFound)
}

// handleHealth handles GET /health
func (as *AppService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"id":     as.config.ID,
	})
}

// verifyToken validates the homeserver token
func (as *AppService) verifyToken(r *http.Request) bool {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" {
		token := strings.TrimPrefix(auth, "Bearer ")
		return hmac.Equal([]byte(token), []byte(as.config.HSToken))
	}

	// Check query parameter
	token := r.URL.Query().Get("access_token")
	return hmac.Equal([]byte(token), []byte(as.config.HSToken))
}

// checkRateLimit implements simple rate limiting
func (as *AppService) checkRateLimit() bool {
	as.rateLimitMu.Lock()
	defer as.rateLimitMu.Unlock()

	now := time.Now()
	if now.After(as.rateLimitReset) {
		as.transactionCount = 0
		as.rateLimitReset = now.Add(time.Second)
	}

	if as.transactionCount >= as.config.MaxTransactionsPerSecond {
		return false
	}

	as.transactionCount++
	return true
}

// bufferEvent stores event when channel is full
func (as *AppService) bufferEvent(event Event) {
	as.bufferMu.Lock()
	defer as.bufferMu.Unlock()

	// Keep last 100 events
	if len(as.eventBuffer) >= 100 {
		as.eventBuffer = as.eventBuffer[1:]
	}
	as.eventBuffer = append(as.eventBuffer, event)
}

// isGhostUser checks if a user ID belongs to this AppService
func (as *AppService) isGhostUser(userID string) bool {
	// Check against registered ghost users
	as.ghostMu.RLock()
	_, exists := as.ghostUsers[userID]
	as.ghostMu.RUnlock()
	return exists
}

// RegisterGhostUser registers a new ghost user
func (as *AppService) RegisterGhostUser(gu *GhostUser) error {
	if gu.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	as.ghostMu.Lock()
	defer as.ghostMu.Unlock()

	// Check if user already exists
	if _, exists := as.ghostUsers[gu.UserID]; exists {
		return fmt.Errorf("ghost user %s already registered", gu.UserID)
	}

	gu.CreatedAt = time.Now()
	gu.LastActive = time.Now()
	as.ghostUsers[gu.UserID] = gu

	as.logger.Info("ghost_user_registered",
		"user_id", gu.UserID,
		"platform", gu.Platform,
		"external_id", gu.ExternalID,
	)

	return nil
}

// GetGhostUser retrieves a ghost user by Matrix ID
func (as *AppService) GetGhostUser(userID string) (*GhostUser, bool) {
	as.ghostMu.RLock()
	defer as.ghostMu.RUnlock()

	gu, exists := as.ghostUsers[userID]
	if exists {
		// Update last active
		gu.LastActive = time.Now()
	}
	return gu, exists
}

// GetGhostUserByExternal retrieves a ghost user by platform and external ID
func (as *AppService) GetGhostUserByExternal(platform, externalID string) (*GhostUser, bool) {
	as.ghostMu.RLock()
	defer as.ghostMu.RUnlock()

	for _, gu := range as.ghostUsers {
		if gu.Platform == platform && gu.ExternalID == externalID {
			gu.LastActive = time.Now()
			return gu, true
		}
	}
	return nil, false
}

// GetBridgeUserID returns the Matrix user ID of the bridge bot
func (as *AppService) GetBridgeUserID() string {
	return fmt.Sprintf("@%s:%s", as.config.SenderLocalpart, as.config.ServerName)
}

// GenerateGhostUserID generates a Matrix user ID for a platform user
func (as *AppService) GenerateGhostUserID(platform, externalID string) string {
	// Sanitize external ID for Matrix compatibility
	safeID := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, externalID)

	return fmt.Sprintf("@%s_%s:%s", platform, safeID, as.config.ServerName)
}

// GetConfig returns the AppService configuration
func (as *AppService) GetConfig() Config {
	return as.config
}

// GetStats returns runtime statistics
func (as *AppService) GetStats() map[string]interface{} {
	as.ghostMu.RLock()
	ghostCount := len(as.ghostUsers)
	as.ghostMu.RUnlock()

	as.bufferMu.Lock()
	bufferLen := len(as.eventBuffer)
	as.bufferMu.Unlock()

	return map[string]interface{}{
		"id":               as.config.ID,
		"homeserver":       as.config.HomeserverURL,
		"ghost_users":      ghostCount,
		"event_buffer":     bufferLen,
		"events_processed": appserviceTracker.Count(),
	}
}
