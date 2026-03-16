package permissions

import (
	"context"
	"fmt"

	"github.com/armorclaw/bridge/pkg/secretary"
)

// SensitivityLevel defines the sensitivity classification for data fields
type SensitivityLevel int

const (
	SensitivityLow    SensitivityLevel = iota
	SensitivityMedium
	SensitivityHigh
	SensitivityCritical
)

// PredictedPermissions contains the prediction result for a form/subject
type PredictedPermissions struct {
	RequiredFields []string         `json:"required_fields"`
	HasConsent     map[string]bool          `json:"has_consent"`
	NeedConsent    []string               `json:"need_consent"`
	Sensitivity    map[string]SensitivityLevel `json:"sensitivity"`
}

// PermissionPredictor analyzes forms and subjects to predict required permissions
type PermissionPredictor struct {
	formsDB secretary.Store
}

// NewPermissionPredictor creates a new PermissionPredictor instance
func NewPermissionPredictor(formsDB secretary.Store) *PermissionPredictor {
	return &PermissionPredictor{
		formsDB: formsDB,
	}
}

// Predict analyzes a task and returns required permissions
func (p *PermissionPredictor) Predict(ctx context.Context, formID string, subjectID string) (*PredictedPermissions, error) {
	// Validate inputs
	if formID == "" && subjectID == "" {
		return nil, fmt.Errorf("formID and subjectID cannot both be empty")
	}

	// Get form definition from Forms DB
	form, err := p.formsDB.GetTemplate(ctx, formID)
	if err != nil {
		return nil, fmt.Errorf("failed to get form: %w", err)
	}

	if form == nil {
		return nil, fmt.Errorf("form not found: %s", formID)
	}

	result := &PredictedPermissions{
		RequiredFields: []string{},
		HasConsent:     make(map[string]bool),
		NeedConsent:    []string{},
		Sensitivity:    make(map[string]SensitivityLevel),
	}

	// Check template-level PIIRefs
	for _, piiRef := range form.PIIRefs {
		sensitivity := p.getPIISensitivity(piiRef)
		result.Sensitivity[piiRef] = sensitivity
		result.RequiredFields = append(result.RequiredFields, piiRef)
	}

	return result, nil
}

// getPIISensitivity returns sensitivity for a PII reference
func (p *PermissionPredictor) getPIISensitivity(piiRef string) SensitivityLevel {
	// Default to high sensitivity for PII refs
	return SensitivityHigh
}
