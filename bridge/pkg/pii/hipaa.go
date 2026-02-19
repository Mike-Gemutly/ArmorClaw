// Package pii provides HIPAA-compliant PII detection and scrubbing extensions
package pii

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sync"
	"time"
)

// HIPAATier represents the HIPAA compliance tier
type HIPAATier string

const (
	HIPAATierBasic    HIPAATier = "basic"    // Standard PII scrubbing
	HIPAATierStandard HIPAATier = "standard" // Enhanced medical patterns
	HIPAATierFull     HIPAATier = "full"     // Full HIPAA compliance
)

// PHIType represents Protected Health Information types
type PHIType string

const (
	PHITypeMRN          PHIType = "mrn"           // Medical Record Number
	PHITypeHPBN         PHIType = "hpbn"          // Health Plan Beneficiary Number
	PHITypeDeviceID     PHIType = "device_id"     // Medical Device Identifier
	PHITypeBiometric    PHIType = "biometric"     // Biometric identifiers
	PHITypeLabResult    PHIType = "lab_result"    // Lab results with identifiers
	PHITypeDiagnosis    PHIType = "diagnosis"     // Diagnosis codes (ICD)
	PHITypePrescription PHIType = "prescription"  // Prescription info
	PHITypeTreatment    PHIType = "treatment"     // Treatment information
)

// HIPAAConfig configures HIPAA-compliant scrubbing
type HIPAAConfig struct {
	// Compliance tier
	Tier HIPAATier

	// Enable PHI audit logging
	EnableAuditLog bool

	// Audit log retention (days)
	AuditRetentionDays int

	// Hash context for logging (never log original)
	HashContext bool

	// Custom patterns
	CustomPatterns []*PIIPattern

	// Logger
	Logger *slog.Logger

	// StreamingMode enables chunk-level scrubbing (may miss cross-chunk patterns)
	// When false, uses buffered mode for full compliance
	StreamingMode bool

	// QuarantineEnabled blocks messages with critical PHI for admin review
	QuarantineEnabled bool

	// NotifyOnQuarantine sends notification to sender when message quarantined
	NotifyOnQuarantine bool

	// QuarantineNotifier is called when a message is quarantined
	// Parameters: userID, roomID, phiType, message
	QuarantineNotifier func(userID, roomID, phiType string, detections []PHIDetection)
}

// ScrubMode represents the processing mode for scrubbing
type ScrubMode string

const (
	// ScrubModeBuffered collects full response before scrubbing (recommended for HIPAA)
	ScrubModeBuffered ScrubMode = "buffered"
	// ScrubModeStreaming processes chunks as they arrive (may miss cross-chunk patterns)
	ScrubModeStreaming ScrubMode = "streaming"
)

// PHIDetection represents a detected PHI item
type PHIDetection struct {
	Type        PHIType   `json:"type"`
	Severity    string    `json:"severity"`
	Start       int       `json:"start"`
	End         int       `json:"end"`
	ContextHash string    `json:"context_hash,omitempty"`
	Confidence  float64   `json:"confidence"`
	Timestamp   time.Time `json:"timestamp"`
}

// HIPAAAuditEntry represents an audit log entry for HIPAA compliance
type HIPAAAuditEntry struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Action       string    `json:"action"` // detect, redact, allow
	PHIType      PHIType   `json:"phi_type"`
	Severity     string    `json:"severity"`
	Confidence   float64   `json:"confidence"`
	ContextHash  string    `json:"context_hash,omitempty"`
	Source       string    `json:"source,omitempty"`
	UserID       string    `json:"user_id,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	Result       string    `json:"result"` // redacted, blocked, logged
}

// HIPAAScrubber extends the standard scrubber with HIPAA compliance
type HIPAAScrubber struct {
	*Scrubber
	config          HIPAAConfig
	phiPatterns     map[PHIType]*regexp.Regexp
	auditLog        []HIPAAAuditEntry
	auditMu         sync.Mutex
	logger          *slog.Logger
	enabled         bool
	streamingMode   bool
	chunkBuffer     string
	chunkMu         sync.Mutex
}

// GetMode returns the current scrubbing mode
func (hs *HIPAAScrubber) GetMode() ScrubMode {
	if hs.streamingMode {
		return ScrubModeStreaming
	}
	return ScrubModeBuffered
}

// NewHIPAAScrubber creates a HIPAA-compliant PII scrubber
func NewHIPAAScrubber(config HIPAAConfig) *HIPAAScrubber {
	// Add HIPAA-specific patterns
	patterns := DefaultPatterns()
	patterns = append(patterns, getHIPAAPatterns(config.Tier)...)

	if len(config.CustomPatterns) > 0 {
		patterns = append(patterns, config.CustomPatterns...)
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	if config.AuditRetentionDays == 0 {
		config.AuditRetentionDays = 90 // HIPAA minimum
	}

	hs := &HIPAAScrubber{
		Scrubber:      NewWithPatterns(patterns),
		config:        config,
		phiPatterns:   make(map[PHIType]*regexp.Regexp),
		auditLog:      make([]HIPAAAuditEntry, 0),
		logger:        logger,
		enabled:       true,
		streamingMode: config.StreamingMode,
		chunkBuffer:   "",
	}

	// Compile PHI patterns
	hs.compilePHIPatterns()

	return hs
}

// compilePHIPatterns compiles PHI-specific detection patterns
func (hs *HIPAAScrubber) compilePHIPatterns() {
	patterns := map[PHIType]string{
		// Medical Record Number (various formats)
		PHITypeMRN: `\b(?:MRN|Medical\s*Record)[\s:#]*([A-Z0-9]{6,12})\b`,

		// Health Plan Beneficiary Number (Medicare/Medicaid)
		PHITypeHPBN: `\b(?:[A-Z]{1,3}\d{6}[A-Z]{0,2}|\d{4}-\d{4}-\d{4})\b`,

		// Medical Device Identifier (UDI format)
		PHITypeDeviceID: `\b(?:\d{4,5}/\d{5,6}/\d{2}|\d{2}[A-Z]\d{5}[A-Z]{2}\d)\b`,

		// Biometric patterns (simplified)
		PHITypeBiometric: `\b(?:fingerprint|retinal|biometric|facial\s*recognition)[\s:]+[A-Z0-9]{10,}\b`,

		// Lab result identifiers
		PHITypeLabResult: `\b(?:Lab|Laboratory|Test)[\s:#]*([A-Z]{2,4}-\d{4,8})\b`,

		// ICD-10 Diagnosis codes
		PHITypeDiagnosis: `\b[A-Z]\d{2}(?:\.\d{1,4})?\b`,

		// Prescription numbers
		PHITypePrescription: `\b(?:Rx|Prescription|R)[\s:#]*\d{6,12}\b`,
	}

	for phiType, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			hs.logger.Error("Failed to compile PHI pattern", "type", phiType, "error", err)
			continue
		}
		hs.phiPatterns[phiType] = compiled
	}
}

// ScrubPHI scrubs text for both PII and PHI
func (hs *HIPAAScrubber) ScrubPHI(ctx context.Context, text string, source string) (string, []PHIDetection, error) {
	return hs.ScrubPHIWithMetadata(ctx, text, source, "", "")
}

// ScrubPHIWithMetadata scrubs text with additional metadata for quarantine notifications
func (hs *HIPAAScrubber) ScrubPHIWithMetadata(ctx context.Context, text string, source string, userID string, roomID string) (string, []PHIDetection, error) {
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	if !hs.enabled {
		return text, nil, nil
	}

	// First, scrub standard PII
	result, _ := hs.Scrubber.Scrub(text)

	// Then detect and scrub PHI
	var detections []PHIDetection
	var criticalDetections []PHIDetection

	for phiType, pattern := range hs.phiPatterns {
		matches := pattern.FindAllStringIndex(result, -1)
		for _, match := range matches {
			start, end := match[0], match[1]
			original := result[start:end]

			severity := getPHISeverity(phiType)
			detection := PHIDetection{
				Type:       phiType,
				Severity:   severity,
				Start:      start,
				End:        end,
				Confidence: 0.85, // Base confidence for pattern matches
				Timestamp:  time.Now(),
			}

			if hs.config.HashContext {
				detection.ContextHash = hashContext(original)
			}

			detections = append(detections, detection)

			// Track critical detections for quarantine
			if severity == "critical" {
				criticalDetections = append(criticalDetections, detection)
			}

			// Handle quarantine mode for critical PHI
			if hs.config.QuarantineEnabled && severity == "critical" {
				// Log quarantine action
				if hs.config.EnableAuditLog {
					hs.logHIPAAAudit(HIPAAAuditEntry{
						Timestamp:   time.Now(),
						Action:      "quarantine",
						PHIType:     phiType,
						Severity:    severity,
						Confidence:  detection.Confidence,
						ContextHash: detection.ContextHash,
						Source:      source,
						UserID:      userID,
						SessionID:   roomID,
						Result:      "quarantined",
					})
				}

				// Send notification if configured
				if hs.config.NotifyOnQuarantine && hs.config.QuarantineNotifier != nil {
					hs.config.QuarantineNotifier(userID, roomID, string(phiType), criticalDetections)
				}

				// Return quarantined message
				return "[MESSAGE QUARANTINED - Contains critical PHI. An administrator will review.]", detections, nil
			}

			// Replace with PHI redaction marker
			replacement := hs.getPHIReplacement(phiType)
			result = result[:start] + replacement + result[end:]

			// Log audit entry
			if hs.config.EnableAuditLog {
				hs.logHIPAAAudit(HIPAAAuditEntry{
					Timestamp:   time.Now(),
					Action:      "redact",
					PHIType:     phiType,
					Severity:    severity,
					Confidence:  detection.Confidence,
					ContextHash: detection.ContextHash,
					Source:      source,
					UserID:      userID,
					SessionID:   roomID,
					Result:      "redacted",
				})
			}
		}
	}

	return result, detections, nil
}

// DetectPHI detects PHI without scrubbing
func (hs *HIPAAScrubber) DetectPHI(text string) []PHIDetection {
	if !hs.enabled {
		return nil
	}

	var detections []PHIDetection

	for phiType, pattern := range hs.phiPatterns {
		matches := pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			start, end := match[0], match[1]
			original := text[start:end]

			detection := PHIDetection{
				Type:       phiType,
				Severity:   getPHISeverity(phiType),
				Start:      start,
				End:        end,
				Confidence: 0.85,
				Timestamp:  time.Now(),
			}

			if hs.config.HashContext {
				detection.ContextHash = hashContext(original)
			}

			detections = append(detections, detection)
		}
	}

	return detections
}

// HasPHI checks if text contains any PHI
func (hs *HIPAAScrubber) HasPHI(text string) bool {
	return len(hs.DetectPHI(text)) > 0
}

// GetAuditLog returns HIPAA audit entries
func (hs *HIPAAScrubber) GetAuditLog(limit int) []HIPAAAuditEntry {
	hs.auditMu.Lock()
	defer hs.auditMu.Unlock()

	if limit <= 0 || limit > len(hs.auditLog) {
		limit = len(hs.auditLog)
	}

	start := len(hs.auditLog) - limit
	if start < 0 {
		start = 0
	}

	result := make([]HIPAAAuditEntry, limit)
	copy(result, hs.auditLog[start:])
	return result
}

// ExportAuditLog exports audit log in compliance format
func (hs *HIPAAScrubber) ExportAuditLog(format string) ([]byte, error) {
	hs.auditMu.Lock()
	defer hs.auditMu.Unlock()

	switch format {
	case "json":
		return json.MarshalIndent(hs.auditLog, "", "  ")
	default:
		return json.Marshal(hs.auditLog)
	}
}

// ClearAuditLog clears old audit entries (respecting retention)
func (hs *HIPAAScrubber) ClearAuditLog() {
	hs.auditMu.Lock()
	defer hs.auditMu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -hs.config.AuditRetentionDays)
	var filtered []HIPAAAuditEntry

	for _, entry := range hs.auditLog {
		if entry.Timestamp.After(cutoff) {
			filtered = append(filtered, entry)
		}
	}

	hs.auditLog = filtered
}

// Enable enables the scrubber
func (hs *HIPAAScrubber) Enable() {
	hs.enabled = true
}

// Disable disables the scrubber (for testing)
func (hs *HIPAAScrubber) Disable() {
	hs.enabled = false
}

// logHIPAAAudit adds an entry to the HIPAA audit log
func (hs *HIPAAScrubber) logHIPAAAudit(entry HIPAAAuditEntry) {
	hs.auditMu.Lock()
	defer hs.auditMu.Unlock()

	// Generate unique ID
	entry.ID = generateAuditID()

	// Keep audit log bounded (last 10000 entries)
	if len(hs.auditLog) >= 10000 {
		hs.auditLog = hs.auditLog[1:]
	}

	hs.auditLog = append(hs.auditLog, entry)

	// Also log to structured logger
	hs.logger.Info("PHI audit",
		"id", entry.ID,
		"action", entry.Action,
		"phi_type", entry.PHIType,
		"severity", entry.Severity,
		"result", entry.Result,
		"source", entry.Source,
	)
}

// getPHIReplacement returns the replacement string for PHI type
func (hs *HIPAAScrubber) getPHIReplacement(phiType PHIType) string {
	replacements := map[PHIType]string{
		PHITypeMRN:          "[MRN REDACTED]",
		PHITypeHPBN:         "[HEALTH_ID REDACTED]",
		PHITypeDeviceID:     "[DEVICE_ID REDACTED]",
		PHITypeBiometric:    "[BIOMETRIC REDACTED]",
		PHITypeLabResult:    "[LAB_RESULT REDACTED]",
		PHITypeDiagnosis:    "[DIAGNOSIS REDACTED]",
		PHITypePrescription: "[PRESCRIPTION REDACTED]",
		PHITypeTreatment:    "[TREATMENT REDACTED]",
	}

	if r, ok := replacements[phiType]; ok {
		return r
	}
	return "[PHI REDACTED]"
}

// Helper functions

func getHIPAAPatterns(tier HIPAATier) []*PIIPattern {
	patterns := []*PIIPattern{
		// Medicare Number (MBI format)
		{
			Name:        "medicare_number",
			Pattern:     regexp.MustCompile(`\b[1-9][A-Z0-9]\d[A-Z]\d[A-Z]\d[A-Z]\d[A-Z]\d\b`),
			Replacement: "[MEDICARE_REDRACTED]",
			Description: "Medicare Beneficiary Identifier (MBI)",
		},
		// NPI (National Provider Identifier)
		{
			Name:        "npi",
			Pattern:     regexp.MustCompile(`\b[1-9]\d{9}\b`),
			Replacement: "[NPI_REDACTED]",
			Description: "National Provider Identifier (10 digits, starts with 1-9)",
		},
	}

	if tier == HIPAATierFull {
		// Additional patterns for full compliance
		patterns = append(patterns,
			// CPT codes
			&PIIPattern{
				Name:        "cpt_code",
				Pattern:     regexp.MustCompile(`\b\d{4,5}[A-Z]?\b`),
				Replacement: "[CPT_REDACTED]",
				Description: "CPT procedure codes",
			},
		)
	}

	return patterns
}

func getPHISeverity(phiType PHIType) string {
	switch phiType {
	case PHITypeMRN, PHITypeHPBN, PHITypeBiometric:
		return "critical"
	case PHITypeDeviceID, PHITypeLabResult, PHITypeDiagnosis:
		return "high"
	default:
		return "medium"
	}
}

func hashContext(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:8]) // First 8 bytes for brevity
}

func generateAuditID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Buffered Processing Methods for LLM Response Stream

// AppendChunk adds a chunk to the internal buffer for buffered processing
// Use this when in buffered mode to collect the full response before scrubbing
func (hs *HIPAAScrubber) AppendChunk(chunk string) {
	hs.chunkMu.Lock()
	defer hs.chunkMu.Unlock()
	hs.chunkBuffer += chunk
}

// FlushBuffer processes the accumulated buffer and returns scrubbed content
// After calling, the buffer is cleared
func (hs *HIPAAScrubber) FlushBuffer(ctx context.Context, source string) (string, []PHIDetection, error) {
	hs.chunkMu.Lock()
	defer hs.chunkMu.Unlock()

	if hs.chunkBuffer == "" {
		return "", nil, nil
	}

	result, detections, err := hs.ScrubPHI(ctx, hs.chunkBuffer, source)
	hs.chunkBuffer = "" // Clear buffer after processing
	return result, detections, err
}

// FlushBufferWithMetadata processes buffer with metadata for quarantine notifications
func (hs *HIPAAScrubber) FlushBufferWithMetadata(ctx context.Context, source, userID, roomID string) (string, []PHIDetection, error) {
	hs.chunkMu.Lock()
	defer hs.chunkMu.Unlock()

	if hs.chunkBuffer == "" {
		return "", nil, nil
	}

	result, detections, err := hs.ScrubPHIWithMetadata(ctx, hs.chunkBuffer, source, userID, roomID)
	hs.chunkBuffer = ""
	return result, detections, err
}

// GetBufferSize returns the current size of the chunk buffer
func (hs *HIPAAScrubber) GetBufferSize() int {
	hs.chunkMu.Lock()
	defer hs.chunkMu.Unlock()
	return len(hs.chunkBuffer)
}

// ClearBuffer clears the internal buffer without processing
func (hs *HIPAAScrubber) ClearBuffer() {
	hs.chunkMu.Lock()
	defer hs.chunkMu.Unlock()
	hs.chunkBuffer = ""
}

// ScrubChunk processes a single chunk in streaming mode
// WARNING: May miss cross-chunk PHI patterns. Use buffered mode for full compliance.
func (hs *HIPAAScrubber) ScrubChunk(ctx context.Context, chunk string, source string) (string, []PHIDetection, error) {
	if !hs.streamingMode {
		// In buffered mode, just append to buffer
		hs.AppendChunk(chunk)
		return chunk, nil, nil // Return chunk as-is, will be scrubbed on flush
	}

	// Streaming mode: scrub immediately (may miss cross-chunk patterns)
	return hs.ScrubPHI(ctx, chunk, source)
}

// SetQuarantineNotifier sets the callback function for quarantine notifications
func (hs *HIPAAScrubber) SetQuarantineNotifier(notifier func(userID, roomID, phiType string, detections []PHIDetection)) {
	hs.config.QuarantineNotifier = notifier
}

// DefaultQuarantineNotifier creates a default quarantine notification message
func DefaultQuarantineNotifier(userID, roomID, phiType string, detections []PHIDetection) string {
	return fmt.Sprintf(
		"⚠️ Your message was flagged for review due to security policy. "+
			"Detected: %s. An administrator will review and release it if appropriate.",
		phiType,
	)
}
