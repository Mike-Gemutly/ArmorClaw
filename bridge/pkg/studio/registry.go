package studio

import (
	"fmt"
	"strings"
)

//=============================================================================
// Skill Registry
//=============================================================================

// SkillRegistry manages available skills and provides validation
type SkillRegistry struct {
	store Store
}

// NewSkillRegistry creates a new skill registry
func NewSkillRegistry(store Store) *SkillRegistry {
	return &SkillRegistry{store: store}
}

// List returns all skills, optionally filtered by category
func (r *SkillRegistry) List(category string) ([]*Skill, error) {
	return r.store.ListSkills(category)
}

// Get retrieves a skill by ID
func (r *SkillRegistry) Get(id string) (*Skill, error) {
	return r.store.GetSkill(id)
}

// Register adds a new skill to the registry
func (r *SkillRegistry) Register(skill *Skill) error {
	// Validate required fields
	if skill.ID == "" {
		return fmt.Errorf("skill ID is required")
	}
	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	// Set defaults
	if skill.Category == "" {
		skill.Category = "general"
	}

	return r.store.RegisterSkill(skill)
}

// Validate checks if all skill IDs are valid
func (r *SkillRegistry) Validate(skillIDs []string) *SkillValidationResult {
	result := &SkillValidationResult{
		Valid:      true,
		InvalidIDs: []string{},
	}

	for _, id := range skillIDs {
		skill, err := r.store.GetSkill(id)
		if err != nil || skill == nil {
			result.Valid = false
			result.InvalidIDs = append(result.InvalidIDs, id)
		}
	}

	if !result.Valid {
		result.Message = fmt.Sprintf("Invalid skill IDs: %s", strings.Join(result.InvalidIDs, ", "))
	}

	return result
}

// Exists checks if a skill exists
func (r *SkillRegistry) Exists(id string) bool {
	skill, err := r.store.GetSkill(id)
	return err == nil && skill != nil
}

// ByCategory returns skills grouped by category
func (r *SkillRegistry) ByCategory() (map[string][]*Skill, error) {
	all, err := r.store.ListSkills("")
	if err != nil {
		return nil, err
	}

	byCategory := make(map[string][]*Skill)
	for _, skill := range all {
		byCategory[skill.Category] = append(byCategory[skill.Category], skill)
	}

	return byCategory, nil
}

//=============================================================================
// PII Registry
//=============================================================================

// PIIRegistry manages available PII fields and provides validation
type PIIRegistry struct {
	store Store
}

// NewPIIRegistry creates a new PII registry
func (r *PIIRegistry) NewPIIRegistry(store Store) *PIIRegistry {
	return &PIIRegistry{store: store}
}

// NewPIIRegistry creates a new PII registry
func NewPIIRegistry(store Store) *PIIRegistry {
	return &PIIRegistry{store: store}
}

// List returns all PII fields, optionally filtered by sensitivity
func (r *PIIRegistry) List(sensitivity string) ([]*PIIField, error) {
	return r.store.ListPIIFields(sensitivity)
}

// Get retrieves a PII field by ID
func (r *PIIRegistry) Get(id string) (*PIIField, error) {
	return r.store.GetPIIField(id)
}

// Register adds a new PII field to the registry
func (r *PIIRegistry) Register(field *PIIField) error {
	// Validate required fields
	if field.ID == "" {
		return fmt.Errorf("PII field ID is required")
	}
	if field.Name == "" {
		return fmt.Errorf("PII field name is required")
	}

	// Validate sensitivity level
	validSensitivity := map[string]bool{
		"low":      true,
		"medium":   true,
		"high":     true,
		"critical": true,
	}
	if field.Sensitivity == "" {
		field.Sensitivity = "medium"
	} else if !validSensitivity[field.Sensitivity] {
		return fmt.Errorf("invalid sensitivity level: %s (must be low, medium, high, or critical)", field.Sensitivity)
	}

	// Auto-set requires_approval for high/critical
	if field.Sensitivity == "high" || field.Sensitivity == "critical" {
		field.RequiresApproval = true
	}

	return r.store.RegisterPIIField(field)
}

// Validate checks if all PII field IDs are valid
func (r *PIIRegistry) Validate(fieldIDs []string) *PIIValidationResult {
	result := &PIIValidationResult{
		Valid:            true,
		InvalidIDs:       []string{},
		RequiresApproval: []string{},
	}

	for _, id := range fieldIDs {
		field, err := r.store.GetPIIField(id)
		if err != nil || field == nil {
			result.Valid = false
			result.InvalidIDs = append(result.InvalidIDs, id)
		} else if field.RequiresApproval {
			result.RequiresApproval = append(result.RequiresApproval, id)
		}
	}

	if !result.Valid {
		result.Message = fmt.Sprintf("Invalid PII field IDs: %s", strings.Join(result.InvalidIDs, ", "))
	}

	return result
}

// Exists checks if a PII field exists
func (r *PIIRegistry) Exists(id string) bool {
	field, err := r.store.GetPIIField(id)
	return err == nil && field != nil
}

// BySensitivity returns PII fields grouped by sensitivity level
func (r *PIIRegistry) BySensitivity() (map[string][]*PIIField, error) {
	all, err := r.store.ListPIIFields("")
	if err != nil {
		return nil, err
	}

	bySensitivity := make(map[string][]*PIIField)
	for _, field := range all {
		bySensitivity[field.Sensitivity] = append(bySensitivity[field.Sensitivity], field)
	}

	return bySensitivity, nil
}

// GetFieldsRequiringApproval returns PII fields that require explicit approval
func (r *PIIRegistry) GetFieldsRequiringApproval(fieldIDs []string) []string {
	var requiresApproval []string
	for _, id := range fieldIDs {
		field, err := r.store.GetPIIField(id)
		if err == nil && field != nil && field.RequiresApproval {
			requiresApproval = append(requiresApproval, id)
		}
	}
	return requiresApproval
}

//=============================================================================
// Resource Profile Manager
//=============================================================================

// ProfileManager manages resource profiles
type ProfileManager struct {
	store Store
}

// NewProfileManager creates a new profile manager
func NewProfileManager(store Store) *ProfileManager {
	return &ProfileManager{store: store}
}

// Get returns a resource profile by tier
func (m *ProfileManager) Get(tier string) (ResourceProfile, error) {
	// First check defaults
	if profile, ok := DefaultResourceProfiles[tier]; ok {
		return profile, nil
	}
	return ResourceProfile{}, fmt.Errorf("resource profile not found: %s", tier)
}

// Validate checks if a tier is valid
func (m *ProfileManager) Validate(tier string) error {
	if tier == "" {
		return nil // Empty tier defaults to medium
	}

	validTiers := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
	}

	if !validTiers[tier] {
		return fmt.Errorf("invalid resource tier: %s (must be low, medium, or high)", tier)
	}

	return nil
}

// List returns all available profiles
func (m *ProfileManager) List() map[string]ResourceProfile {
	return DefaultResourceProfiles
}
