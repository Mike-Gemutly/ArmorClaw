// Package sdtw provides adapters for Slack, Discord, Teams, and WhatsApp platforms
package sdtw

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// SDTWAdapter defines the contract for platform adapters
type SDTWAdapter interface {
	// Metadata
	Platform() string          // e.g., "slack", "discord", "teams", "whatsapp"
	Capabilities() CapabilitySet // Feature detection
	Version() string            // Adapter version for compatibility

	// Lifecycle
	Initialize(ctx context.Context, config AdapterConfig) error
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error

	// Core Operations
	SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error)
	ReceiveEvent(event ExternalEvent) error // Inbound handler

	// Health & Monitoring
	HealthCheck() (HealthStatus, error)
	Metrics() (AdapterMetrics, error)
}

// CapabilitySet defines adapter feature support
type CapabilitySet struct {
	Read         bool // Can receive messages
	Write        bool // Can send messages
	Media        bool // Supports media attachments
	Reactions    bool // Supports message reactions
	Threads      bool // Supports threaded replies
	Edit         bool // Can edit messages
	Delete       bool // Can delete messages
	Typing       bool // Typing indicators
	ReadReceipts bool // Read receipt support
}

// Target identifies a message destination
type Target struct {
	Platform string            // "slack", "discord", "teams", "whatsapp"
	RoomID   string            // Matrix room ID (maps to SDTW target)
	Channel  string            // Platform channel ID
	UserID   string            // Platform user ID (for DMs)
	ThreadID string            // For threaded replies
	Metadata map[string]string // Platform-specific data
}

// Message represents sanitized message content
type Message struct {
	ID          string            // Unique message identifier
	Content     string            // Text content (PII scrubbed)
	Type        MessageType       // Text, Image, File, Media
	Attachments []Attachment
	ReplyTo     string            // Parent message ID (threads)
	Metadata    map[string]string // Platform-specific metadata
	Timestamp   time.Time         // Message creation time
	Signature   string            // HMAC-SHA256 integrity
}

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
	MessageTypeMedia MessageType = "media"
)

// Attachment represents a file attachment
type Attachment struct {
	ID       string
	URL      string
	MimeType string
	Size     int64
	Filename string
}

// SendResult represents the result of a send operation
type SendResult struct {
	MessageID string            // Platform message ID
	Delivered bool              // Delivery confirmation
	Timestamp time.Time         // Delivery time
	Error     *AdapterError     // Error details if failed
	Metadata  map[string]string // Platform response data
}

// ExternalEvent represents an incoming event from a platform
type ExternalEvent struct {
	Platform   string            // Source platform
	EventType  string            // Event type (message, reaction, etc.)
	Timestamp  time.Time         // Event timestamp
	Source     string            // Source channel/user
	Content    string            // Event content
	Attachments []Attachment     // File attachments
	Metadata   map[string]string // Platform-specific data
	Signature  string            // HMAC for verification
}

// AdapterConfig holds configuration for an adapter
type AdapterConfig struct {
	Platform     string
	Enabled      bool
	Credentials  map[string]string // Injected from keystore
	Settings     map[string]string // Platform-specific settings
	RateLimits   RateLimitConfig
	WebhookURL   string
	DefaultTarget string
}

// RateLimitConfig defines rate limit settings
type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
	BackoffOnLimit    bool
}

// HealthStatus represents the health of an adapter
type HealthStatus struct {
	Connected    bool          // Connection state
	LastPing     time.Time     // Last successful ping
	LastMessage  time.Time     // Last message processed
	ErrorRate    float64       // Error percentage (1h window)
	Latency      time.Duration // Average latency (5m window)
	QueueDepth   int           // Pending message count
	Error        string        // Current error if any
}

// AdapterMetrics holds metrics for an adapter
type AdapterMetrics struct {
	MessagesSent     int64
	MessagesReceived int64
	MessagesFailed   int64
	LastError        string
	LastErrorTime    time.Time
	Uptime           time.Duration
}

// AdapterError represents an error from an adapter
type AdapterError struct {
	Code       ErrorCode      // Error classification
	Message    string         // Human-readable error
	Retryable  bool           // Can be retried
	RetryAfter time.Duration  // Suggested backoff
	Permanent  bool           // Non-recoverable
}

// Error implements the error interface
func (e *AdapterError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ErrorCode represents an error classification
type ErrorCode string

const (
	ErrRateLimited    ErrorCode = "rate_limited"
	ErrAuthFailed     ErrorCode = "auth_failed"
	ErrInvalidTarget  ErrorCode = "invalid_target"
	ErrNetworkError   ErrorCode = "network_error"
	ErrTimeout        ErrorCode = "timeout"
	ErrCircuitOpen    ErrorCode = "circuit_open"
	ErrValidation     ErrorCode = "validation_error"
	ErrPlatformError  ErrorCode = "platform_error"
)

// NewAdapterError creates a new AdapterError
func NewAdapterError(code ErrorCode, message string, retryable bool) *AdapterError {
	return &AdapterError{
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Permanent: !retryable,
	}
}

// SignMessage creates an HMAC-SHA256 signature for a message
func SignMessage(content string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature verifies an HMAC-SHA256 signature
func VerifySignature(content, signature, secret string) bool {
	expected := SignMessage(content, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// ValidateMessage performs basic validation on a message
func ValidateMessage(msg Message) error {
	if msg.ID == "" {
		return NewAdapterError(ErrValidation, "message ID is required", false)
	}
	if msg.Content == "" && len(msg.Attachments) == 0 {
		return NewAdapterError(ErrValidation, "message must have content or attachments", false)
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	if msg.Type == "" {
		msg.Type = MessageTypeText
	}
	return nil
}

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	config      AdapterConfig
	capabilities CapabilitySet
	version     string
	metrics     AdapterMetrics
	startTime   time.Time
	initialized bool
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(platform, version string, caps CapabilitySet) *BaseAdapter {
	return &BaseAdapter{
		config:      AdapterConfig{Platform: platform},
		capabilities: caps,
		version:     version,
		startTime:   time.Now(),
	}
}

// Platform returns the platform name
func (b *BaseAdapter) Platform() string {
	return b.config.Platform
}

// Capabilities returns the adapter's capabilities
func (b *BaseAdapter) Capabilities() CapabilitySet {
	return b.capabilities
}

// Version returns the adapter version
func (b *BaseAdapter) Version() string {
	return b.version
}

// Initialize sets up the adapter with configuration
func (b *BaseAdapter) Initialize(ctx context.Context, config AdapterConfig) error {
	b.config = config
	b.initialized = true
	return nil
}

// HealthCheck returns the current health status
func (b *BaseAdapter) HealthCheck() (HealthStatus, error) {
	status := HealthStatus{
		Connected:   b.initialized,
		LastPing:    time.Now(),
		LastMessage: b.metrics.LastErrorTime,
		ErrorRate:   0.0,
		QueueDepth:  0,
	}

	if b.metrics.MessagesFailed > 0 {
		total := b.metrics.MessagesSent + b.metrics.MessagesReceived
		status.ErrorRate = float64(b.metrics.MessagesFailed) / float64(total) * 100
	}

	if b.metrics.LastError != "" {
		status.Error = b.metrics.LastError
	}

	return status, nil
}

// Metrics returns the current metrics
func (b *BaseAdapter) Metrics() (AdapterMetrics, error) {
	b.metrics.Uptime = time.Since(b.startTime)
	return b.metrics, nil
}

// RecordSent records a sent message
func (b *BaseAdapter) RecordSent() {
	b.metrics.MessagesSent++
}

// RecordReceived records a received message
func (b *BaseAdapter) RecordReceived() {
	b.metrics.MessagesReceived++
}

// RecordError records an error
func (b *BaseAdapter) RecordError(err error) {
	b.metrics.MessagesFailed++
	b.metrics.LastError = err.Error()
	b.metrics.LastErrorTime = time.Now()
}
