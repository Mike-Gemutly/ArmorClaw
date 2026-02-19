// Package audit provides enhanced compliance audit logging for ArmorClaw Enterprise
package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ComplianceLevel represents the compliance level
type ComplianceLevel string

const (
	ComplianceStandard  ComplianceLevel = "standard"  // 30-day retention
	ComplianceExtended  ComplianceLevel = "extended"  // 90-day retention
	ComplianceFull      ComplianceLevel = "full"      // 1-year retention
	ComplianceHIPAA     ComplianceLevel = "hipaa"     // 6-year retention
)

// ComplianceConfig configures the compliance audit system
type ComplianceConfig struct {
	Level           ComplianceLevel
	StoragePath     string
	MaxEntries      int
	EnableHashChain bool   // Tamper-evident logging
	ExportFormats   []string // Supported: json, csv
}

// ComplianceEntry represents an enhanced audit entry with compliance metadata
type ComplianceEntry struct {
	// Standard fields
	ID        string      `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	EventType EventType   `json:"event_type"`
	SessionID string      `json:"session_id,omitempty"`
	RoomID    string      `json:"room_id,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`

	// Compliance fields
	Source      string `json:"source,omitempty"`       // Component that generated event
	IPAddress   string `json:"ip_address,omitempty"`   // Source IP if applicable
	UserAgent   string `json:"user_agent,omitempty"`   // Client user agent
	Action      string `json:"action"`                 // create, read, update, delete, login, logout
	Resource    string `json:"resource,omitempty"`     // Resource affected
	Status      string `json:"status"`                 // success, failure, denied
	Reason      string `json:"reason,omitempty"`       // Reason for status

	// Integrity fields
	PreviousHash string `json:"previous_hash,omitempty"` // Hash chain
	EntryHash    string `json:"entry_hash,omitempty"`    // Hash of this entry
}

// ComplianceReport represents a generated compliance report
type ComplianceReport struct {
	GeneratedAt     time.Time          `json:"generated_at"`
	PeriodStart     time.Time          `json:"period_start"`
	PeriodEnd       time.Time          `json:"period_end"`
	TotalEvents     int                `json:"total_events"`
	EventsByType    map[EventType]int  `json:"events_by_type"`
	EventsByAction  map[string]int     `json:"events_by_action"`
	EventsByStatus  map[string]int     `json:"events_by_status"`
	UniqueUsers     int                `json:"unique_users"`
	UniqueSessions  int                `json:"unique_sessions"`
	FailedActions   int                `json:"failed_actions"`
	SecurityEvents  int                `json:"security_events"`
	IntegrityStatus string             `json:"integrity_status"`
	Entries         []ComplianceEntry  `json:"entries,omitempty"`
}

// ComplianceAuditLog provides enterprise-grade audit logging
type ComplianceAuditLog struct {
	mu            sync.RWMutex
	config        ComplianceConfig
	entries       []ComplianceEntry
	lastHash      string
	hashKey       []byte
	retentionDays int
}

// DefaultComplianceConfig returns default compliance configuration
func DefaultComplianceConfig() ComplianceConfig {
	return ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     "/var/lib/armorclaw/compliance-audit.db",
		MaxEntries:      100000,
		EnableHashChain: true,
		ExportFormats:   []string{"json", "csv"},
	}
}

// NewComplianceAuditLog creates a new compliance audit log
func NewComplianceAuditLog(config ComplianceConfig) (*ComplianceAuditLog, error) {
	// Set retention based on level
	retentionDays := 90 // Default extended
	switch config.Level {
	case ComplianceStandard:
		retentionDays = 30
	case ComplianceFull:
		retentionDays = 365
	case ComplianceHIPAA:
		retentionDays = 2190 // 6 years
	}

	if config.MaxEntries == 0 {
		config.MaxEntries = 100000
	}

	cal := &ComplianceAuditLog{
		config:        config,
		entries:       make([]ComplianceEntry, 0, 10000),
		retentionDays: retentionDays,
		hashKey:       generateHashKey(),
	}

	// Load existing entries
	if err := cal.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load compliance log: %w", err)
	}

	return cal, nil
}

// Log logs a compliance entry
func (cal *ComplianceAuditLog) Log(entry ComplianceEntry) error {
	cal.mu.Lock()
	defer cal.mu.Unlock()

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Generate entry ID
	entry.ID = generateEntryID(entry.Timestamp)

	// Hash chain for tamper detection
	if cal.config.EnableHashChain {
		entry.PreviousHash = cal.lastHash
		entry.EntryHash = cal.computeHash(entry)
		cal.lastHash = entry.EntryHash
	}

	// Append entry
	cal.entries = append(cal.entries, entry)

	// Prune old entries based on max size
	if len(cal.entries) > cal.config.MaxEntries {
		cal.entries = cal.entries[len(cal.entries)-cal.config.MaxEntries:]
	}

	// Save to file
	return cal.saveToFile()
}

// LogEvent logs a compliance event with standard fields
func (cal *ComplianceAuditLog) LogEvent(eventType EventType, action, resource, status string, details interface{}) error {
	return cal.Log(ComplianceEntry{
		Timestamp: time.Now(),
		EventType: eventType,
		Action:    action,
		Resource:  resource,
		Status:    status,
		Details:   details,
	})
}

// LogUserAction logs a user action for compliance
func (cal *ComplianceAuditLog) LogUserAction(userID, action, resource, status, reason string) error {
	return cal.Log(ComplianceEntry{
		Timestamp: time.Now(),
		EventType: EventType("user_action"),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Status:    status,
		Reason:    reason,
	})
}

// Query queries compliance entries
func (cal *ComplianceAuditLog) Query(params QueryParams) ([]ComplianceEntry, error) {
	cal.mu.RLock()
	defer cal.mu.RUnlock()

	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 10000 {
		params.Limit = 10000
	}

	var result []ComplianceEntry
	for i := len(cal.entries) - 1; i >= 0 && len(result) < params.Limit; i-- {
		entry := cal.entries[i]

		// Apply filters
		if params.EventType != "" && entry.EventType != params.EventType {
			continue
		}
		if params.SessionID != "" && entry.SessionID != params.SessionID {
			continue
		}
		if params.RoomID != "" && entry.RoomID != params.RoomID {
			continue
		}
		if !params.Since.IsZero() && entry.Timestamp.Before(params.Since) {
			continue
		}

		result = append(result, entry)
	}

	return result, nil
}

// Export exports audit log in the specified format
func (cal *ComplianceAuditLog) Export(format string, start, end time.Time) ([]byte, error) {
	cal.mu.RLock()
	defer cal.mu.RUnlock()

	// Filter entries by time range
	var filtered []ComplianceEntry
	for _, entry := range cal.entries {
		if (start.IsZero() || !entry.Timestamp.Before(start)) &&
			(end.IsZero() || !entry.Timestamp.After(end)) {
			filtered = append(filtered, entry)
		}
	}

	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(filtered, "", "  ")
	case "csv":
		return cal.exportCSV(filtered)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportCSV exports entries as CSV
func (cal *ComplianceAuditLog) exportCSV(entries []ComplianceEntry) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"id", "timestamp", "event_type", "session_id", "room_id", "user_id",
		"action", "resource", "status", "reason", "entry_hash"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write entries
	for _, entry := range entries {
		record := []string{
			entry.ID,
			entry.Timestamp.Format(time.RFC3339),
			string(entry.EventType),
			entry.SessionID,
			entry.RoomID,
			entry.UserID,
			entry.Action,
			entry.Resource,
			entry.Status,
			entry.Reason,
			entry.EntryHash,
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return []byte(buf.String()), writer.Error()
}

// GenerateReport generates a compliance report for a time period
func (cal *ComplianceAuditLog) GenerateReport(start, end time.Time) (*ComplianceReport, error) {
	cal.mu.RLock()
	defer cal.mu.RUnlock()

	report := &ComplianceReport{
		GeneratedAt:   time.Now(),
		PeriodStart:   start,
		PeriodEnd:     end,
		EventsByType:  make(map[EventType]int),
		EventsByAction: make(map[string]int),
		EventsByStatus: make(map[string]int),
	}

	users := make(map[string]bool)
	sessions := make(map[string]bool)

	// Verify hash chain integrity
	integrityValid := true
	var prevHash string

	for _, entry := range cal.entries {
		// Time filter
		if !start.IsZero() && entry.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && entry.Timestamp.After(end) {
			continue
		}

		report.TotalEvents++

		// Count by type
		report.EventsByType[entry.EventType]++

		// Count by action
		report.EventsByAction[entry.Action]++

		// Count by status
		report.EventsByStatus[entry.Status]++

		// Track unique users
		if entry.UserID != "" {
			users[entry.UserID] = true
		}

		// Track unique sessions
		if entry.SessionID != "" {
			sessions[entry.SessionID] = true
		}

		// Count failures
		if entry.Status == "failure" || entry.Status == "denied" {
			report.FailedActions++
		}

		// Count security events
		if entry.EventType == EventSecurityViolation ||
			strings.Contains(string(entry.EventType), "security") {
			report.SecurityEvents++
		}

		// Verify hash chain
		if cal.config.EnableHashChain && integrityValid {
			if prevHash != "" && entry.PreviousHash != prevHash {
				integrityValid = false
			}
			prevHash = entry.EntryHash
		}
	}

	report.UniqueUsers = len(users)
	report.UniqueSessions = len(sessions)

	if integrityValid {
		report.IntegrityStatus = "valid"
	} else {
		report.IntegrityStatus = "compromised"
	}

	return report, nil
}

// VerifyIntegrity verifies the hash chain integrity
func (cal *ComplianceAuditLog) VerifyIntegrity() (bool, []int) {
	cal.mu.RLock()
	defer cal.mu.RUnlock()

	var corruptedIndices []int

	for i, entry := range cal.entries {
		if !cal.config.EnableHashChain {
			continue
		}

		// Verify hash
		expectedHash := cal.computeHash(entry)
		if entry.EntryHash != expectedHash {
			corruptedIndices = append(corruptedIndices, i)
			continue
		}

		// Verify chain
		if i > 0 {
			prevEntry := cal.entries[i-1]
			if entry.PreviousHash != prevEntry.EntryHash {
				corruptedIndices = append(corruptedIndices, i)
			}
		}
	}

	return len(corruptedIndices) == 0, corruptedIndices
}

// PurgeOldEntries removes entries older than retention period
func (cal *ComplianceAuditLog) PurgeOldEntries() (int, error) {
	cal.mu.Lock()
	defer cal.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -cal.retentionDays)
	var filtered []ComplianceEntry
	purged := 0

	for _, entry := range cal.entries {
		if entry.Timestamp.After(cutoff) {
			filtered = append(filtered, entry)
		} else {
			purged++
		}
	}

	cal.entries = filtered
	return purged, cal.saveToFile()
}

// Count returns total entry count
func (cal *ComplianceAuditLog) Count() int {
	cal.mu.RLock()
	defer cal.mu.RUnlock()
	return len(cal.entries)
}

// computeHash computes the hash for an entry
func (cal *ComplianceAuditLog) computeHash(entry ComplianceEntry) string {
	h := hmac.New(sha256.New, cal.hashKey)

	// Include all relevant fields in hash
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%v",
		entry.ID,
		entry.Timestamp.Format(time.RFC3339Nano),
		entry.EventType,
		entry.SessionID,
		entry.RoomID,
		entry.UserID,
		entry.Action,
		entry.Status,
		entry.Details,
	)

	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// loadFromFile loads entries from the storage file
func (cal *ComplianceAuditLog) loadFromFile() error {
	file, err := os.Open(cal.config.StoragePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var entry ComplianceEntry
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		cal.entries = append(cal.entries, entry)

		// Track last hash for chain continuation
		if entry.EntryHash != "" {
			cal.lastHash = entry.EntryHash
		}
	}

	return nil
}

// saveToFile saves entries to the storage file
func (cal *ComplianceAuditLog) saveToFile() error {
	// Ensure directory exists
	dir := filepath.Dir(cal.config.StoragePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(cal.config.StoragePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range cal.entries {
		if err := encoder.Encode(entry); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions

func generateEntryID(t time.Time) string {
	return fmt.Sprintf("audit-%d", t.UnixNano())
}

func generateHashKey() []byte {
	key := make([]byte, 32)
	// In production, this should be loaded from secure storage
	// For now, use a deterministic key based on timestamp
	for i := range key {
		key[i] = byte(i * 7 % 256)
	}
	return key
}

// GetRetentionDays returns the configured retention period
func (cal *ComplianceAuditLog) GetRetentionDays() int {
	return cal.retentionDays
}

// GetComplianceLevel returns the configured compliance level
func (cal *ComplianceAuditLog) GetComplianceLevel() ComplianceLevel {
	return cal.config.Level
}

// ParseComplianceLevel parses a string to ComplianceLevel
func ParseComplianceLevel(s string) ComplianceLevel {
	switch strings.ToLower(s) {
	case "standard", "basic":
		return ComplianceStandard
	case "extended", "enhanced":
		return ComplianceExtended
	case "full", "complete":
		return ComplianceFull
	case "hipaa", "medical":
		return ComplianceHIPAA
	default:
		return ComplianceExtended
	}
}

// FormatDuration formats a duration for reports
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	days := hours / 24
	hours = hours % 24

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh %dm", hours, int(d.Minutes())%60)
}

// ParseEntryID extracts timestamp from entry ID
func ParseEntryID(id string) (time.Time, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 || parts[0] != "audit" {
		return time.Time{}, fmt.Errorf("invalid entry ID format")
	}

	nanos, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, nanos), nil
}
