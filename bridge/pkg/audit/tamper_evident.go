// Package audit provides tamper-evident audit logging with hash chain verification
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// TamperEvidentConfig configures the tamper-evident audit log
type TamperEvidentConfig struct {
	// Enable hash chain verification
	Enabled bool

	// Export interval for compliance
	ExportInterval time.Duration

	// Maximum entries before rotation
	MaxEntries int

	// Retention period for audit entries
	RetentionPeriod time.Duration

	// Logger
	Logger *slog.Logger
}

// TamperEvidentLog provides tamper-evident audit logging
type TamperEvidentLog struct {
	config    TamperEvidentConfig
	entries   []*TamperEvidentEntry
	lastHash  string
	mu        sync.RWMutex
	logger    *slog.Logger
	exportCh  chan struct{}
}

// TamperEvidentEntry represents a tamper-evident audit entry
type TamperEvidentEntry struct {
	// Sequence number (monotonically increasing)
	Sequence int64 `json:"sequence"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`

	// Event type
	EventType string `json:"event_type"`

	// Actor (who performed the action)
	Actor Actor `json:"actor"`

	// Action performed
	Action string `json:"action"`

	// Resource affected
	Resource Resource `json:"resource"`

	// Event details
	Details map[string]interface{} `json:"details,omitempty"`

	// Hash of this entry (includes previous hash)
	Hash string `json:"hash"`

	// Hash of previous entry (forms chain)
	PreviousHash string `json:"previous_hash"`

	// Signature (optional, for high-security mode)
	Signature string `json:"signature,omitempty"`

	// Compliance flags
	Compliance ComplianceFlags `json:"compliance"`
}

// Actor represents who performed an action
type Actor struct {
	Type       string `json:"type"`        // "user", "system", "service"
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	IPAddress  string `json:"ip_address,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
}

// Resource represents what was affected
type Resource struct {
	Type string `json:"type"` // "room", "message", "user", "key", "container"
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// ComplianceFlags contains compliance-related flags
type ComplianceFlags struct {
	PHIInvolved   bool   `json:"phi_involved"`
	AuditRequired bool   `json:"audit_required"`
	Severity      string `json:"severity"` // "low", "medium", "high", "critical"
	Category      string `json:"category"` // "access", "modification", "deletion", "export"
}

// VerificationResult represents the result of chain verification
type VerificationResult struct {
	Valid          bool     `json:"valid"`
	TotalEntries   int64    `json:"total_entries"`
	InvalidEntries []int64  `json:"invalid_entries,omitempty"`
	TamperedAt     *int64   `json:"tampered_at,omitempty"`
	VerifiedAt     time.Time `json:"verified_at"`
	Error          string   `json:"error,omitempty"`
}

// NewTamperEvidentLog creates a new tamper-evident audit log
func NewTamperEvidentLog(config TamperEvidentConfig) *TamperEvidentLog {
	if config.Logger == nil {
		config.Logger = slog.Default().With("component", "audit_tamper")
	}

	if config.MaxEntries == 0 {
		config.MaxEntries = 100000
	}

	return &TamperEvidentLog{
		config:   config,
		entries:  make([]*TamperEvidentEntry, 0),
		lastHash: "0000000000000000000000000000000000000000000000000000000000000000",
		exportCh: make(chan struct{}, 1),
		logger:   config.Logger,
	}
}

// LogEntry logs a new tamper-evident entry
func (l *TamperEvidentLog) LogEntry(eventType string, actor Actor, action string, resource Resource, details map[string]interface{}, compliance ComplianceFlags) (*TamperEvidentEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	sequence := int64(len(l.entries) + 1)

	entry := &TamperEvidentEntry{
		Sequence:     sequence,
		Timestamp:    time.Now().UTC(),
		EventType:    eventType,
		Actor:        actor,
		Action:       action,
		Resource:     resource,
		Details:      details,
		PreviousHash: l.lastHash,
		Compliance:   compliance,
	}

	// Calculate hash
	entry.Hash = l.calculateHash(entry)

	// Update chain
	l.entries = append(l.entries, entry)
	l.lastHash = entry.Hash

	// Trim if over max
	if len(l.entries) > l.config.MaxEntries {
		l.entries = l.entries[len(l.entries)-l.config.MaxEntries:]
	}

	l.logger.Debug("audit_entry_logged",
		"sequence", sequence,
		"event_type", eventType,
		"action", action,
		"resource_type", resource.Type,
		"resource_id", resource.ID,
	)

	return entry, nil
}

// calculateHash calculates the hash for an entry
func (l *TamperEvidentLog) calculateHash(entry *TamperEvidentEntry) string {
	// Create a copy without the hash field for hashing
	hashData := struct {
		Sequence     int64                  `json:"sequence"`
		Timestamp    time.Time              `json:"timestamp"`
		EventType    string                 `json:"event_type"`
		Actor        Actor                  `json:"actor"`
		Action       string                 `json:"action"`
		Resource     Resource               `json:"resource"`
		Details      map[string]interface{} `json:"details,omitempty"`
		PreviousHash string                 `json:"previous_hash"`
	}{
		Sequence:     entry.Sequence,
		Timestamp:    entry.Timestamp,
		EventType:    entry.EventType,
		Actor:        entry.Actor,
		Action:       entry.Action,
		Resource:     entry.Resource,
		Details:      entry.Details,
		PreviousHash: entry.PreviousHash,
	}

	data, _ := json.Marshal(hashData)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// VerifyChain verifies the integrity of the audit chain
func (l *TamperEvidentLog) VerifyChain() *VerificationResult {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := &VerificationResult{
		Valid:        true,
		TotalEntries: int64(len(l.entries)),
		VerifiedAt:   time.Now().UTC(),
	}

	if len(l.entries) == 0 {
		return result
	}

	// Verify each entry's hash
	for i, entry := range l.entries {
		expectedHash := l.calculateHash(entry)
		if entry.Hash != expectedHash {
			result.Valid = false
			result.InvalidEntries = append(result.InvalidEntries, entry.Sequence)
			if result.TamperedAt == nil {
				seq := entry.Sequence
				result.TamperedAt = &seq
			}
			l.logger.Warn("audit_tampering_detected",
				"sequence", entry.Sequence,
				"expected_hash", expectedHash,
				"actual_hash", entry.Hash,
			)
		}

		// Verify chain linkage
		if i > 0 {
			if entry.PreviousHash != l.entries[i-1].Hash {
				result.Valid = false
				result.InvalidEntries = append(result.InvalidEntries, entry.Sequence)
			}
		}
	}

	return result
}

// GetEntries retrieves entries with optional filtering
func (l *TamperEvidentLog) GetEntries(filter EntryFilter) []*TamperEvidentEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []*TamperEvidentEntry

	for _, entry := range l.entries {
		if filter.Matches(entry) {
			result = append(result, entry)
		}
	}

	// Apply limit
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[len(result)-filter.Limit:]
	}

	return result
}

// EntryFilter filters audit entries
type EntryFilter struct {
	EventType  string
	ActorID    string
	ResourceID string
	StartTime  time.Time
	EndTime    time.Time
	Severity   string
	PHIOnly    bool
	Limit      int
}

// Matches checks if an entry matches the filter
func (f EntryFilter) Matches(entry *TamperEvidentEntry) bool {
	if f.EventType != "" && entry.EventType != f.EventType {
		return false
	}
	if f.ActorID != "" && entry.Actor.ID != f.ActorID {
		return false
	}
	if f.ResourceID != "" && entry.Resource.ID != f.ResourceID {
		return false
	}
	if !f.StartTime.IsZero() && entry.Timestamp.Before(f.StartTime) {
		return false
	}
	if !f.EndTime.IsZero() && entry.Timestamp.After(f.EndTime) {
		return false
	}
	if f.Severity != "" && entry.Compliance.Severity != f.Severity {
		return false
	}
	if f.PHIOnly && !entry.Compliance.PHIInvolved {
		return false
	}
	return true
}

// Export exports the audit log in a compliance-ready format
func (l *TamperEvidentLog) Export(format string) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	export := struct {
		ExportedAt    time.Time              `json:"exported_at"`
		TotalEntries  int                    `json:"total_entries"`
		ChainVerified bool                   `json:"chain_verified"`
		Entries       []*TamperEvidentEntry  `json:"entries"`
	}{
		ExportedAt:    time.Now().UTC(),
		TotalEntries:  len(l.entries),
		ChainVerified: l.VerifyChain().Valid,
		Entries:       l.entries,
	}

	switch format {
	case "json":
		return json.MarshalIndent(export, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// GetStats returns audit log statistics
func (l *TamperEvidentLog) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	eventCounts := make(map[string]int)
	severityCounts := make(map[string]int)
	phiCount := 0

	for _, entry := range l.entries {
		eventCounts[entry.EventType]++
		severityCounts[entry.Compliance.Severity]++
		if entry.Compliance.PHIInvolved {
			phiCount++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(l.entries),
		"event_counts":    eventCounts,
		"severity_counts": severityCounts,
		"phi_involved":    phiCount,
		"chain_valid":     l.VerifyChain().Valid,
		"first_entry":     l.getFirstEntryTime(),
		"last_entry":      l.getLastEntryTime(),
	}
}

func (l *TamperEvidentLog) getFirstEntryTime() time.Time {
	if len(l.entries) == 0 {
		return time.Time{}
	}
	return l.entries[0].Timestamp
}

func (l *TamperEvidentLog) getLastEntryTime() time.Time {
	if len(l.entries) == 0 {
		return time.Time{}
	}
	return l.entries[len(l.entries)-1].Timestamp
}

// Convenience methods for common audit events

// LogUserAccess logs a user access event
func (l *TamperEvidentLog) LogUserAccess(userID, userName, action, resourceType, resourceID, ipAddress string) (*TamperEvidentEntry, error) {
	return l.LogEntry(
		"user_access",
		Actor{Type: "user", ID: userID, Name: userName, IPAddress: ipAddress},
		action,
		Resource{Type: resourceType, ID: resourceID},
		nil,
		ComplianceFlags{Category: "access", Severity: "low"},
	)
}

// LogSecurityEvent logs a security-related event
func (l *TamperEvidentLog) LogSecurityEvent(severity, action string, details map[string]interface{}) (*TamperEvidentEntry, error) {
	return l.LogEntry(
		"security_event",
		Actor{Type: "system", ID: "security"},
		action,
		Resource{Type: "system", ID: "security"},
		details,
		ComplianceFlags{Category: "access", Severity: severity, AuditRequired: true},
	)
}

// LogPHIEvent logs a PHI-related event
func (l *TamperEvidentLog) LogPHIEvent(userID, action, resourceType, resourceID string, details map[string]interface{}) (*TamperEvidentEntry, error) {
	return l.LogEntry(
		"phi_event",
		Actor{Type: "user", ID: userID},
		action,
		Resource{Type: resourceType, ID: resourceID},
		details,
		ComplianceFlags{Category: "access", Severity: "high", PHIInvolved: true, AuditRequired: true},
	)
}

// LogConfigurationChange logs a configuration change
func (l *TamperEvidentLog) LogConfigurationChange(userID, action, resourceType, resourceID string, changes map[string]interface{}) (*TamperEvidentEntry, error) {
	return l.LogEntry(
		"config_change",
		Actor{Type: "user", ID: userID},
		action,
		Resource{Type: resourceType, ID: resourceID},
		changes,
		ComplianceFlags{Category: "modification", Severity: "medium", AuditRequired: true},
	)
}
