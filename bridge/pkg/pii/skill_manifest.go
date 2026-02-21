// Package pii provides skill manifest structures for blind fill capability.
// Skills declare their PII requirements via manifests, and users approve
// specific fields through Human-in-the-Loop consent.
package pii

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// SensitivityLevel defines how sensitive a field request is
type SensitivityLevel string

const (
	SensitivityLow      SensitivityLevel = "low"      // Name, city
	SensitivityMedium   SensitivityLevel = "medium"   // Email, phone
	SensitivityHigh     SensitivityLevel = "high"     // DOB, address
	SensitivityCritical SensitivityLevel = "critical" // SSN, financial
)

// VariableRequest represents a skill's request for a specific PII field
type VariableRequest struct {
	// Key is the field identifier (e.g., "full_name", "email")
	Key string `json:"key"`

	// Description explains why the skill needs this field
	Description string `json:"description"`

	// Required indicates if this field is mandatory for the skill
	Required bool `json:"required"`

	// Sensitivity indicates the sensitivity level of this field
	Sensitivity SensitivityLevel `json:"sensitivity"`

	// ProfileHints suggests which profile types might have this field
	ProfileHints []ProfileType `json:"profile_hints,omitempty"`

	// DefaultValue provides a fallback if user doesn't have the field
	DefaultValue string `json:"default_value,omitempty"`
}

// Validate checks if the variable request is valid
func (vr *VariableRequest) Validate() error {
	if vr.Key == "" {
		return errors.New("variable key is required")
	}
	if vr.Description == "" {
		return errors.New("variable description is required")
	}
	return nil
}

// SkillManifest represents a skill's declaration of required PII variables
type SkillManifest struct {
	// SkillID is the unique identifier for the skill
	SkillID string `json:"skill_id"`

	// SkillName is the human-readable name
	SkillName string `json:"skill_name"`

	// SkillDescription explains what the skill does
	SkillDescription string `json:"skill_description,omitempty"`

	// Variables is the list of PII fields the skill requests
	Variables []VariableRequest `json:"variables"`

	// CreatedAt is when this manifest was created
	CreatedAt int64 `json:"created_at"`

	// ExpiresAt is when this manifest expires (optional)
	ExpiresAt int64 `json:"expires_at,omitempty"`

	// Version is the manifest version for compatibility
	Version string `json:"version"`
}

// NewSkillManifest creates a new skill manifest
func NewSkillManifest(skillID, skillName string, variables []VariableRequest) *SkillManifest {
	return &SkillManifest{
		SkillID:    skillID,
		SkillName:  skillName,
		Variables:  variables,
		CreatedAt:  time.Now().Unix(),
		Version:    "1.0",
	}
}

// Validate checks if the manifest is valid
func (sm *SkillManifest) Validate() error {
	if sm.SkillID == "" {
		return errors.New("skill_id is required")
	}
	if sm.SkillName == "" {
		return errors.New("skill_name is required")
	}
	if len(sm.Variables) == 0 {
		return errors.New("at least one variable is required")
	}

	// Validate each variable
	seenKeys := make(map[string]bool)
	for _, v := range sm.Variables {
		if err := v.Validate(); err != nil {
			return err
		}
		if seenKeys[v.Key] {
			return errors.New("duplicate variable key: " + v.Key)
		}
		seenKeys[v.Key] = true
	}

	return nil
}

// GetRequiredFields returns the list of required field keys
func (sm *SkillManifest) GetRequiredFields() []string {
	var required []string
	for _, v := range sm.Variables {
		if v.Required {
			required = append(required, v.Key)
		}
	}
	return required
}

// GetOptionalFields returns the list of optional field keys
func (sm *SkillManifest) GetOptionalFields() []string {
	var optional []string
	for _, v := range sm.Variables {
		if !v.Required {
			optional = append(optional, v.Key)
		}
	}
	return optional
}

// GetAllFieldKeys returns all requested field keys
func (sm *SkillManifest) GetAllFieldKeys() []string {
	keys := make([]string, len(sm.Variables))
	for i, v := range sm.Variables {
		keys[i] = v.Key
	}
	return keys
}

// GetVariable returns the variable request for a specific key
func (sm *SkillManifest) GetVariable(key string) *VariableRequest {
	for _, v := range sm.Variables {
		if v.Key == key {
			return &v
		}
	}
	return nil
}

// ToJSON serializes the manifest to JSON
func (sm *SkillManifest) ToJSON() ([]byte, error) {
	return json.Marshal(sm)
}

// FromJSON deserializes a manifest from JSON
func FromJSON(data []byte) (*SkillManifest, error) {
	var manifest SkillManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// ResolvedVariables represents the result of a successful blind fill
type ResolvedVariables struct {
	// SkillID is the skill that requested these variables
	SkillID string `json:"skill_id"`

	// RequestID is the unique identifier for this resolution
	RequestID string `json:"request_id"`

	// Variables is the map of field keys to resolved values
	// CRITICAL: These values should never be logged
	Variables map[string]string `json:"variables"`

	// GrantedBy is the user ID who approved the access
	GrantedBy string `json:"granted_by"`

	// ProfileID is the profile used for resolution
	ProfileID string `json:"profile_id"`

	// GrantedFields is the list of fields that were approved
	GrantedFields []string `json:"granted_fields"`

	// DeniedFields is the list of fields that were denied
	DeniedFields []string `json:"denied_fields,omitempty"`

	// ResolvedAt is when the resolution occurred
	ResolvedAt int64 `json:"resolved_at"`

	// ExpiresAt is when this resolution expires
	ExpiresAt int64 `json:"expires_at"`
}

// NewResolvedVariables creates a new resolved variables instance
func NewResolvedVariables(skillID, requestID, profileID, grantedBy string) *ResolvedVariables {
	return &ResolvedVariables{
		SkillID:      skillID,
		RequestID:    requestID,
		ProfileID:    profileID,
		GrantedBy:    grantedBy,
		Variables:    make(map[string]string),
		ResolvedAt:   time.Now().Unix(),
		ExpiresAt:    time.Now().Add(5 * time.Minute).Unix(), // 5 minute default expiry
	}
}

// SetVariable sets a resolved variable value
func (rv *ResolvedVariables) SetVariable(key, value string) {
	if rv.Variables == nil {
		rv.Variables = make(map[string]string)
	}
	rv.Variables[key] = value
	rv.GrantedFields = append(rv.GrantedFields, key)
}

// DenyField records a denied field
func (rv *ResolvedVariables) DenyField(key string) {
	rv.DeniedFields = append(rv.DeniedFields, key)
}

// GetVariable retrieves a resolved variable value
func (rv *ResolvedVariables) GetVariable(key string) (string, bool) {
	val, exists := rv.Variables[key]
	return val, exists
}

// HasVariable checks if a variable was resolved
func (rv *ResolvedVariables) HasVariable(key string) bool {
	_, exists := rv.Variables[key]
	return exists
}

// IsExpired checks if the resolution has expired
func (rv *ResolvedVariables) IsExpired() bool {
	return time.Now().Unix() > rv.ExpiresAt
}

// ToJSON serializes the resolved variables to JSON
// WARNING: This includes actual PII values - handle with care
func (rv *ResolvedVariables) ToJSON() ([]byte, error) {
	return json.Marshal(rv)
}

// ToSafeJSON serializes the resolved variables without actual values
// This is safe for logging and debugging
func (rv *ResolvedVariables) ToSafeJSON() ([]byte, error) {
	safe := map[string]interface{}{
		"skill_id":       rv.SkillID,
		"request_id":     rv.RequestID,
		"profile_id":     rv.ProfileID,
		"granted_by":     rv.GrantedBy,
		"granted_fields": rv.GrantedFields,
		"denied_fields":  rv.DeniedFields,
		"resolved_at":    rv.ResolvedAt,
		"expires_at":     rv.ExpiresAt,
		"field_count":    len(rv.Variables),
		// Note: Variables map is intentionally excluded
	}
	return json.Marshal(safe)
}

// GenerateRequestID generates a unique request ID
func GenerateRequestID() string {
	return "req_" + securerandom.MustID(16)
}
