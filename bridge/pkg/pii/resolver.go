// Package pii provides the BlindFillEngine for secure PII resolution.
// The engine resolves skill variable requests from encrypted profiles
// without exposing all PII - only approved fields are returned.
package pii

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
)

var (
	// ErrProfileNotFound is returned when the requested profile doesn't exist
	ErrProfileNotFound = errors.New("profile not found")

	// ErrFieldNotFound is returned when a requested field doesn't exist in the profile
	ErrFieldNotFound = errors.New("field not found in profile")

	// ErrRequiredFieldMissing is returned when a required field is not approved
	ErrRequiredFieldMissing = errors.New("required field not approved")

	// ErrAccessDenied is returned when access to a field is denied
	ErrAccessDenied = errors.New("access to field denied")

	// ErrResolutionExpired is returned when trying to use an expired resolution
	ErrResolutionExpired = errors.New("resolution has expired")
)

// KeystoreInterface defines the interface for keystore operations
type KeystoreInterface interface {
	RetrieveProfile(id string) (*UserProfileData, error)
}

// UserProfileData is the minimal interface for profile data from keystore
type UserProfileData struct {
	ID           string
	ProfileName  string
	ProfileType  string
	Data         []byte // JSON-serialized
	FieldSchema  string
	CreatedAt    int64
	UpdatedAt    int64
	LastAccessed int64
	IsDefault    bool
}

// BlindFillEngine resolves skill variable requests from encrypted profiles.
// It ensures that only approved fields are returned and all access is logged.
type BlindFillEngine struct {
	keystore     KeystoreInterface
	auditLogger  *audit.CriticalOperationLogger
	securityLog  *logger.SecurityLogger
	log          *logger.Logger
}

// NewBlindFillEngine creates a new blind fill engine
func NewBlindFillEngine(keystore KeystoreInterface, securityLog *logger.SecurityLogger) *BlindFillEngine {
	return &BlindFillEngine{
		keystore:    keystore,
		securityLog: securityLog,
		log:         logger.Global().WithComponent("pii_resolver"),
	}
}

// SetAuditLogger sets the audit logger for the engine
func (e *BlindFillEngine) SetAuditLogger(auditLogger *audit.CriticalOperationLogger) {
	e.auditLogger = auditLogger
}

// ResolveVariables resolves approved fields from a profile for a skill manifest.
// This is the core "blind fill" operation:
// 1. Validate the manifest
// 2. Retrieve the encrypted profile
// 3. Extract only the approved fields
// 4. Log the access (field names only, never values)
// 5. Return resolved variables
func (e *BlindFillEngine) ResolveVariables(
	ctx context.Context,
	manifest *SkillManifest,
	profileID string,
	approvedFields []string,
	requestID string,
	grantedBy string,
) (*ResolvedVariables, error) {
	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	// Validate approved fields contain all required fields
	if err := e.validateRequiredFields(manifest, approvedFields); err != nil {
		return nil, err
	}

	// Retrieve the encrypted profile
	profileData, err := e.keystore.RetrieveProfile(profileID)
	if err != nil {
		e.log.Error("profile_retrieve_failed",
			"profile_id", profileID,
			"error", err.Error(),
		)
		return nil, ErrProfileNotFound
	}

	// Decrypt and parse the profile data
	var profileDataParsed ProfileData
	if err := json.Unmarshal(profileData.Data, &profileDataParsed); err != nil {
		e.log.Error("profile_parse_failed",
			"profile_id", profileID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to parse profile data: %w", err)
	}

	// Create resolved variables
	resolved := NewResolvedVariables(manifest.SkillID, requestID, profileID, grantedBy)

	// Get all available fields from profile
	profileFields := profileDataParsed.ToMap()

	// Build approved fields set for fast lookup
	approvedSet := make(map[string]bool)
	for _, f := range approvedFields {
		approvedSet[f] = true
	}

	// Process each requested variable
	for _, req := range manifest.Variables {
		if approvedSet[req.Key] {
			// Field is approved - get the value
			value, exists := profileFields[req.Key]
			if exists && value != "" {
				resolved.SetVariable(req.Key, value)
			} else if req.DefaultValue != "" {
				// Use default value if field is empty but has default
				resolved.SetVariable(req.Key, req.DefaultValue)
			} else if req.Required {
				// Required field is missing
				e.log.Warn("required_field_missing",
					"skill_id", manifest.SkillID,
					"field", req.Key,
				)
				resolved.DenyField(req.Key)
			}
		} else {
			// Field not approved
			resolved.DenyField(req.Key)
		}
	}

	// Log the resolution (field names only, never values)
	e.log.Info("variables_resolved",
		"skill_id", manifest.SkillID,
		"request_id", requestID,
		"profile_id", profileID,
		"granted_fields_count", len(resolved.GrantedFields),
		"denied_fields_count", len(resolved.DeniedFields),
	)

	// Audit logging
	if e.auditLogger != nil {
		_ = e.auditLogger.LogPIIAccessGranted(ctx, requestID, manifest.SkillID, grantedBy, resolved.GrantedFields)
	}

	// Security logging
	if e.securityLog != nil {
		e.securityLog.LogPIIAccessGranted(ctx, requestID, manifest.SkillName, grantedBy, resolved.GrantedFields,
			slog.String("profile_id", profileID),
		)
	}

	return resolved, nil
}

// validateRequiredFields ensures all required fields are in the approved list
func (e *BlindFillEngine) validateRequiredFields(manifest *SkillManifest, approvedFields []string) error {
	approvedSet := make(map[string]bool)
	for _, f := range approvedFields {
		approvedSet[f] = true
	}

	for _, req := range manifest.Variables {
		if req.Required && !approvedSet[req.Key] {
			return fmt.Errorf("%w: %s", ErrRequiredFieldMissing, req.Key)
		}
	}

	return nil
}

// ValidateResolution checks if a resolved variables instance is still valid
func (e *BlindFillEngine) ValidateResolution(resolved *ResolvedVariables) error {
	if resolved == nil {
		return errors.New("resolution is nil")
	}

	if resolved.IsExpired() {
		return ErrResolutionExpired
	}

	return nil
}

// HashValue creates a one-way hash of a PII value for logging/comparison
// This allows detecting duplicate values without exposing the actual value
func HashValue(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:8]) // First 8 bytes for brevity
}

// ProfileResolver provides a simplified interface for resolving profile fields
type ProfileResolver struct {
	engine    *BlindFillEngine
	profileID string
}

// NewProfileResolver creates a resolver for a specific profile
func (e *BlindFillEngine) NewProfileResolver(profileID string) *ProfileResolver {
	return &ProfileResolver{
		engine:    e,
		profileID: profileID,
	}
}

// Resolve resolves a manifest using this profile
func (pr *ProfileResolver) Resolve(
	ctx context.Context,
	manifest *SkillManifest,
	approvedFields []string,
	requestID string,
	grantedBy string,
) (*ResolvedVariables, error) {
	return pr.engine.ResolveVariables(ctx, manifest, pr.profileID, approvedFields, requestID, grantedBy)
}

// ProfileAccessRequest represents a pending access request for audit tracking
type ProfileAccessRequest struct {
	ID              string
	SkillID         string
	SkillName       string
	ProfileID       string
	RequestedFields []string
	RequiredFields  []string
	Status          string // "pending", "approved", "rejected", "expired"
	CreatedAt       time.Time
	ExpiresAt       time.Time
	ApprovedAt      *time.Time
	ApprovedBy      string
	ApprovedFields  []string
	RejectedAt      *time.Time
	RejectedBy      string
	RejectionReason string
}

// IsExpired checks if the request has expired
func (r *ProfileAccessRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsApproved checks if the request is approved
func (r *ProfileAccessRequest) IsApproved() bool {
	return r.Status == "approved"
}

// IsRejected checks if the request is rejected
func (r *ProfileAccessRequest) IsRejected() bool {
	return r.Status == "rejected"
}
