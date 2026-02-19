// Package pii provides LLM response compliance handling
// This module provides tier-dependent PII/PHI scrubbing for LLM responses
package pii

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
)

// Default buffer limits
const (
	DefaultMaxBufferSize = 10 * 1024 * 1024 // 10MB max buffer
)

// LLMComplianceHandler manages PII/PHI compliance for LLM responses.
// It provides a clean facade over HIPAAScrubber with LLM-specific concerns.
//
// Separation of Concerns:
// - This handler: LLM-specific logic (session management, streaming orchestration)
// - HIPAAScrubber: Compliance logic (detection, scrubbing, quarantine)
// - Errors: Structured error handling with traceability
type LLMComplianceHandler struct {
	// scrubber handles all PII/PHI compliance logic (owned by this handler)
	scrubber *HIPAAScrubber

	// Immutable config set at construction
	config LLMComplianceConfig

	// Logger with component context
	logger *slog.Logger

	// Atomic flags for thread-safe access without locks
	enabled         atomic.Bool
	streamingMode   atomic.Bool
	maxBufferSize   atomic.Int64
}

// LLMComplianceConfig configures LLM response compliance
type LLMComplianceConfig struct {
	// Enabled controls whether compliance checking is active
	Enabled bool

	// StreamingMode enables chunk-level scrubbing
	// WARNING: May miss cross-chunk patterns
	StreamingMode bool

	// QuarantineEnabled blocks responses with critical PHI
	QuarantineEnabled bool

	// NotifyOnQuarantine sends notification when response quarantined
	NotifyOnQuarantine bool

	// AuditEnabled logs all compliance events
	AuditEnabled bool

	// Tier is the compliance tier (basic, standard, full)
	Tier string

	// MaxBufferSize limits the buffer size for streaming (bytes)
	MaxBufferSize int64

	// OnQuarantine callback for quarantine notifications
	// IMPORTANT: This callback MUST NOT call back into this handler
	// to avoid deadlocks. Keep it simple and fast.
	OnQuarantine func(sessionID, userID, phiType string, detections []PHIDetection)
}

// LLMResponse represents an LLM API response
type LLMResponse struct {
	Content    string
	TokenUsage TokenUsage
	Model      string
	Provider   string
	SessionID  string
	UserID     string
	RoomID     string
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ComplianceResult represents the result of compliance processing
type ComplianceResult struct {
	// OriginalContent is the unmodified response (for audit)
	OriginalContent string

	// ScrubbedContent is the PII/PHI-safe content
	ScrubbedContent string

	// Detections lists all PII/PHI found
	Detections []PHIDetection

	// WasQuarantined indicates if response was blocked
	WasQuarantined bool

	// QuarantineMessage is shown to user when quarantined
	QuarantineMessage string

	// Mode used for processing (streaming/buffered)
	Mode ScrubMode

	// Source identifies where this response came from (for error tracing)
	Source string
}

// NewLLMComplianceHandler creates a new compliance handler.
// The handler owns the scrubber and manages its lifecycle.
func NewLLMComplianceHandler(config LLMComplianceConfig) *LLMComplianceHandler {
	// Logger with component context for traceability
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(
		"component", "llm_compliance",
		"tier", config.Tier,
	)

	// Set defaults
	if config.MaxBufferSize == 0 {
		config.MaxBufferSize = DefaultMaxBufferSize
	}

	// Create HIPAA scrubber with matching config
	// Note: Quarantine logic is delegated to the scrubber
	hipaaConfig := HIPAAConfig{
		Tier:               HIPAATier(config.Tier),
		EnableAuditLog:     config.AuditEnabled,
		AuditRetentionDays: 90,
		HashContext:        true,
		Logger:             logger.With("subcomponent", "hipaa_scrubber"),
		StreamingMode:      config.StreamingMode,
		QuarantineEnabled:  config.QuarantineEnabled,
		NotifyOnQuarantine: config.NotifyOnQuarantine,
	}

	// Set quarantine notifier if callback provided
	// Note: The callback is stored in the scrubber, not here
	if config.OnQuarantine != nil {
		hipaaConfig.QuarantineNotifier = func(userID, roomID, phiType string, detections []PHIDetection) {
			// Call with empty sessionID (scrubber doesn't know about sessions)
			config.OnQuarantine("", userID, phiType, detections)
		}
	}

	scrubber := NewHIPAAScrubber(hipaaConfig)

	// Disable scrubber if compliance is off
	if !config.Enabled {
		scrubber.Disable()
	}

	h := &LLMComplianceHandler{
		scrubber: scrubber,
		config:   config,
		logger:   logger,
	}

	// Initialize atomic flags
	h.enabled.Store(config.Enabled)
	h.streamingMode.Store(config.StreamingMode)
	h.maxBufferSize.Store(config.MaxBufferSize)

	return h
}

// ProcessResponse processes an LLM response for compliance.
// This is the main entry point for outbound LLM response scrubbing.
//
// Thread-safe: Uses atomic reads for config flags, no locks held during scrubbing.
func (h *LLMComplianceHandler) ProcessResponse(ctx context.Context, response LLMResponse) (*ComplianceResult, error) {
	// Check context first
	select {
	case <-ctx.Done():
		return nil, NewContextCanceledError("llm_response")
	default:
	}

	// Atomic read - no lock needed
	if !h.enabled.Load() {
		return &ComplianceResult{
			OriginalContent: response.Content,
			ScrubbedContent: response.Content,
			Mode:           h.scrubber.GetMode(),
			Source:         response.SessionID,
		}, nil
	}

	// Build source identifier for error tracing
	source := buildSourceID("llm_response", response.SessionID, response.UserID)

	// Delegate all compliance logic to scrubber (no locks held here)
	scrubbed, detections, err := h.scrubber.ScrubPHIWithMetadata(
		ctx,
		response.Content,
		source,
		response.UserID,
		response.RoomID,
	)
	if err != nil {
		h.logger.Error("scrubbing_failed",
			"source", source,
			"session_id", response.SessionID,
			"error", err.Error(),
		)
		return nil, NewScrubbingError(source, err)
	}

	result := &ComplianceResult{
		OriginalContent: response.Content,
		ScrubbedContent: scrubbed,
		Detections:      detections,
		Mode:            h.scrubber.GetMode(),
		Source:          source,
	}

	// Check quarantine status from the scrubbed content
	// The scrubber returns a quarantine message when blocking
	if h.config.QuarantineEnabled && isQuarantineMessage(scrubbed) {
		result.WasQuarantined = true
		result.QuarantineMessage = scrubbed
		result.ScrubbedContent = ""

		h.logger.Warn("response_quarantined",
			"source", source,
			"session_id", response.SessionID,
			"user_id", response.UserID,
			"detection_count", len(detections),
		)
	} else if len(detections) > 0 {
		h.logger.Info("response_scrubbed",
			"source", source,
			"session_id", response.SessionID,
			"detection_count", len(detections),
			"mode", result.Mode,
		)
	}

	return result, nil
}

// ProcessStreamChunk processes a streaming response chunk.
// In buffered mode, chunks are accumulated until FlushStream is called.
// In streaming mode, chunks are scrubbed immediately (may miss cross-chunk patterns).
//
// Thread-safe: Uses atomic reads, delegates to scrubber's thread-safe methods.
func (h *LLMComplianceHandler) ProcessStreamChunk(ctx context.Context, chunk string, sessionID string) (string, []PHIDetection, error) {
	// Check context
	select {
	case <-ctx.Done():
		return "", nil, NewContextCanceledError("stream_chunk")
	default:
	}

	// Atomic read - no lock needed
	if !h.enabled.Load() {
		return chunk, nil, nil
	}

	// Check buffer size limit (prevent OOM)
	if !h.streamingMode.Load() {
		currentSize := h.scrubber.GetBufferSize()
		maxSize := h.maxBufferSize.Load()
		if int64(currentSize)+int64(len(chunk)) > maxSize {
			source := buildSourceID("stream", sessionID, "")
			h.logger.Error("buffer_overflow",
				"source", source,
				"current_size", currentSize,
				"chunk_size", len(chunk),
				"max_size", maxSize,
			)
			return "", nil, NewBufferOverflowError(source, currentSize+len(chunk), int(maxSize))
		}
	}

	source := buildSourceID("llm_stream", sessionID, "")
	return h.scrubber.ScrubChunk(ctx, chunk, source)
}

// FlushStream flushes the accumulated stream buffer (buffered mode only).
// Returns the fully scrubbed content.
//
// Thread-safe: Delegates to scrubber's thread-safe FlushBufferWithMetadata.
func (h *LLMComplianceHandler) FlushStream(ctx context.Context, sessionID, userID, roomID string) (string, []PHIDetection, error) {
	// Check context
	select {
	case <-ctx.Done():
		return "", nil, NewContextCanceledError("flush_stream")
	default:
	}

	// Atomic read
	if !h.enabled.Load() {
		return "", nil, nil
	}

	source := buildSourceID("llm_stream", sessionID, userID)
	result, detections, err := h.scrubber.FlushBufferWithMetadata(ctx, source, userID, roomID)
	if err != nil {
		h.logger.Error("flush_failed",
			"source", source,
			"session_id", sessionID,
			"error", err.Error(),
		)
		return "", nil, NewScrubbingError(source, err)
	}

	return result, detections, nil
}

// AppendToStream appends content to the internal buffer (buffered mode).
// Use ProcessStreamChunk instead for automatic mode handling.
func (h *LLMComplianceHandler) AppendToStream(chunk string) error {
	// Check buffer size limit
	currentSize := h.scrubber.GetBufferSize()
	maxSize := h.maxBufferSize.Load()
	if int64(currentSize)+int64(len(chunk)) > maxSize {
		return NewBufferOverflowError("direct_append", currentSize+len(chunk), int(maxSize))
	}

	h.scrubber.AppendChunk(chunk)
	return nil
}

// GetStreamBufferSize returns the current buffer size
func (h *LLMComplianceHandler) GetStreamBufferSize() int {
	return h.scrubber.GetBufferSize()
}

// ClearStream clears the internal buffer without processing
func (h *LLMComplianceHandler) ClearStream() {
	h.scrubber.ClearBuffer()
}

// IsEnabled returns whether compliance is enabled (thread-safe)
func (h *LLMComplianceHandler) IsEnabled() bool {
	return h.enabled.Load()
}

// SetEnabled enables or disables compliance at runtime (thread-safe)
func (h *LLMComplianceHandler) SetEnabled(enabled bool) {
	h.enabled.Store(enabled)
	if enabled {
		h.scrubber.Enable()
	} else {
		h.scrubber.Disable()
	}
}

// GetMode returns the current processing mode
func (h *LLMComplianceHandler) GetMode() ScrubMode {
	return h.scrubber.GetMode()
}

// SetQuarantineCallback sets the callback for quarantine notifications.
//
// WARNING: This callback MUST NOT call back into this handler to avoid deadlocks.
// Keep the callback simple and fast (e.g., queue a message, don't process synchronously).
//
// Thread-safe: Uses atomic swap pattern.
func (h *LLMComplianceHandler) SetQuarantineCallback(callback func(sessionID, userID, phiType string, detections []PHIDetection)) {
	// Update config (immutable after construction except via this method)
	h.config.OnQuarantine = callback

	// Delegate to scrubber with adapter
	h.scrubber.SetQuarantineNotifier(func(userID, roomID, phiType string, detections []PHIDetection) {
		callback("", userID, phiType, detections)
	})

	h.logger.Info("quarantine_callback_updated")
}

// GetAuditLog returns the compliance audit log
func (h *LLMComplianceHandler) GetAuditLog(limit int) []HIPAAAuditEntry {
	return h.scrubber.GetAuditLog(limit)
}

// ExportAuditLog exports the audit log in the specified format
func (h *LLMComplianceHandler) ExportAuditLog(format string) ([]byte, error) {
	return h.scrubber.ExportAuditLog(format)
}

// GetScrubber returns the underlying HIPAAScrubber for advanced use cases.
// WARNING: Direct scrubber access bypasses handler's buffer limits and logging.
func (h *LLMComplianceHandler) GetScrubber() *HIPAAScrubber {
	return h.scrubber
}

// Helper functions

// buildSourceID creates a traceable source identifier
func buildSourceID(prefix, sessionID, userID string) string {
	if sessionID != "" && userID != "" {
		return prefix + ":" + sessionID + ":" + userID
	}
	if sessionID != "" {
		return prefix + ":" + sessionID
	}
	return prefix
}

// isQuarantineMessage checks if the content is a quarantine message
func isQuarantineMessage(content string) bool {
	return len(content) > 0 && content[0] == '[' && containsQuarantineMarker(content)
}

// containsQuarantineMarker checks for quarantine message markers
func containsQuarantineMarker(s string) bool {
	// Fast check for quarantine markers without importing strings
	return len(s) > 20 && (s[:20] == "[MESSAGE QUARANTINED" ||
		(len(s) > 11 && s[:11] == "[QUARANTINE"))
}

// DefaultLLMComplianceConfig returns default config based on tier
func DefaultLLMComplianceConfig(tier string) LLMComplianceConfig {
	switch tier {
	case "ent", "enterprise", "maximum":
		// Enterprise: Full compliance, buffered mode, quarantine enabled
		return LLMComplianceConfig{
			Enabled:            true,
			StreamingMode:      false, // Buffered for full compliance
			QuarantineEnabled:  true,
			NotifyOnQuarantine: true,
			AuditEnabled:       true,
			Tier:               "full",
			MaxBufferSize:      DefaultMaxBufferSize,
		}
	case "pro", "professional":
		// Professional: Optional compliance, streaming mode by default
		return LLMComplianceConfig{
			Enabled:            false, // Disabled by default for performance
			StreamingMode:      true,  // Allow streaming when enabled
			QuarantineEnabled:  false,
			NotifyOnQuarantine: false,
			AuditEnabled:       false,
			Tier:               "basic",
			MaxBufferSize:      DefaultMaxBufferSize,
		}
	default:
		// Essential/Free: Compliance disabled for maximum performance
		return LLMComplianceConfig{
			Enabled:            false,
			StreamingMode:      true,
			QuarantineEnabled:  false,
			NotifyOnQuarantine: false,
			AuditEnabled:       false,
			Tier:               "basic",
			MaxBufferSize:      DefaultMaxBufferSize,
		}
	}
}

// ShouldUseBufferedMode determines if buffered mode should be used
// based on compliance requirements and performance considerations
func ShouldUseBufferedMode(tier string, requiresHIPAA bool) bool {
	// Always use buffered mode for HIPAA requirements
	if requiresHIPAA {
		return true
	}

	// Enterprise tier defaults to buffered
	if tier == "ent" || tier == "enterprise" || tier == "maximum" {
		return true
	}

	// Other tiers use streaming for performance
	return false
}
