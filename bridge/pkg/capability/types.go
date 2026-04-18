// Package capability defines typed artifact contracts for the multi-agent
// capability broker. Every struct carries JSON tags for wire serialization
// and a Validate method for input checking.
package capability

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Risk taxonomy
// ---------------------------------------------------------------------------

// RiskLevel classifies the risk decision for an action.
type RiskLevel string

const (
	RiskAllow RiskLevel = "ALLOW"
	RiskDeny  RiskLevel = "DENY"
	RiskDefer RiskLevel = "DEFER"
)

// RiskClass categorises the domain of risk.
type RiskClass string

const (
	RiskPayment              RiskClass = "payment"
	RiskIdentityPII          RiskClass = "identity_pii"
	RiskCredentialUse        RiskClass = "credential_use"
	RiskExternalCommunication RiskClass = "external_communication"
	RiskFileExfiltration     RiskClass = "file_exfiltration"
	RiskIrreversibleAction   RiskClass = "irreversible_action"
)

// ---------------------------------------------------------------------------
// Action types
// ---------------------------------------------------------------------------

// ActionRequest represents an agent requesting permission to perform an action.
type ActionRequest struct {
	AgentID string         `json:"agent_id"`
	TeamID  string         `json:"team_id,omitempty"`
	Action  string         `json:"action"`
	Params  map[string]any `json:"params,omitempty"`
}

// Validate ensures AgentID and Action are non-empty.
func (v *ActionRequest) Validate() error {
	if v.AgentID == "" {
		return fmt.Errorf("capability: %T: field agent_id is required", v)
	}
	if v.Action == "" {
		return fmt.Errorf("capability: %T: field action is required", v)
	}
	return nil
}

// ActionResponse is the broker's ruling on an action request.
type ActionResponse struct {
	Allowed        bool      `json:"allowed"`
	Classification RiskLevel `json:"classification"`
	Reason         string    `json:"reason,omitempty"`
	SessionID      string    `json:"session_id,omitempty"`
	RiskClass      RiskClass `json:"risk_class,omitempty"`
}

// Validate ensures Classification is one of the known RiskLevel values.
func (v *ActionResponse) Validate() error {
	switch v.Classification {
	case RiskAllow, RiskDeny, RiskDefer:
		return nil
	default:
		return fmt.Errorf("capability: %T: field classification has invalid value %q", v, v.Classification)
	}
}

// ---------------------------------------------------------------------------
// Capability set
// ---------------------------------------------------------------------------

// CapabilitySet is a set of named capabilities enabled for an agent.
type CapabilitySet map[string]bool

// Validate ensures at least one entry exists when the set is non-nil.
func (v CapabilitySet) Validate() error {
	if v != nil && len(v) == 0 {
		return fmt.Errorf("capability: CapabilitySet: at least one entry is required when non-nil")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Secret references
// ---------------------------------------------------------------------------

// SecretRef points to a stored secret without revealing its value.
type SecretRef struct {
	Field   string `json:"field"`
	Hash    string `json:"hash,omitempty"`
	Version int    `json:"version,omitempty"`
}

// Validate ensures Field is non-empty.
func (v *SecretRef) Validate() error {
	if v.Field == "" {
		return fmt.Errorf("capability: %T: field field is required", v)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Browser artifacts
// ---------------------------------------------------------------------------

// BrowserIntent describes an upcoming browser operation.
type BrowserIntent struct {
	URL        string   `json:"url"`
	Action     string   `json:"action"`
	FormFields []string `json:"form_fields,omitempty"`
}

// Validate ensures URL and Action are non-empty.
func (v *BrowserIntent) Validate() error {
	if v.URL == "" {
		return fmt.Errorf("capability: %T: field url is required", v)
	}
	if v.Action == "" {
		return fmt.Errorf("capability: %T: field action is required", v)
	}
	return nil
}

// BrowserResult captures the outcome of a browser operation.
type BrowserResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title,omitempty"`
	ExtractedData []string `json:"extracted_data,omitempty"`
	Screenshots   []string `json:"screenshots,omitempty"`
}

// Validate ensures URL is non-empty.
func (v *BrowserResult) Validate() error {
	if v.URL == "" {
		return fmt.Errorf("capability: %T: field url is required", v)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Document artifacts
// ---------------------------------------------------------------------------

// DocumentRef references a document collection and optional chunk subset.
type DocumentRef struct {
	CollectionID string   `json:"collection_id"`
	ChunkIDs     []string `json:"chunk_ids,omitempty"`
}

// Validate ensures CollectionID is non-empty.
func (v *DocumentRef) Validate() error {
	if v.CollectionID == "" {
		return fmt.Errorf("capability: %T: field collection_id is required", v)
	}
	return nil
}

// ExtractedChunkSet holds text chunks extracted from a document.
type ExtractedChunkSet struct {
	Chunks  []string `json:"chunks"`
	Summary string   `json:"summary,omitempty"`
}

// Validate ensures at least one chunk is present.
func (v *ExtractedChunkSet) Validate() error {
	if len(v.Chunks) == 0 {
		return fmt.Errorf("capability: %T: at least one chunk is required", v)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Email artifacts
// ---------------------------------------------------------------------------

// EmailDraft represents a composed email awaiting approval.
type EmailDraft struct {
	To          string   `json:"to"`
	Subject     string   `json:"subject"`
	BodyMasked  string   `json:"body_masked"`
	Attachments []string `json:"attachments,omitempty"`
}

// Validate ensures To and Subject are non-empty.
func (v *EmailDraft) Validate() error {
	if v.To == "" {
		return fmt.Errorf("capability: %T: field to is required", v)
	}
	if v.Subject == "" {
		return fmt.Errorf("capability: %T: field subject is required", v)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Blocker types
// ---------------------------------------------------------------------------

// Well-known blocker type constants for WorkflowBlocker.Type.
const (
	BlockerTypeCAPTCHA = "captcha"
)

// ---------------------------------------------------------------------------
// Approval & workflow
// ---------------------------------------------------------------------------

// ApprovalDecision records which fields a human approved or denied.
type ApprovalDecision struct {
	Approved     bool     `json:"approved"`
	Fields       []string `json:"fields,omitempty"`
	DeniedFields []string `json:"denied_fields,omitempty"`
}

// Validate always returns nil — Approved=false with no denied fields is valid.
func (v *ApprovalDecision) Validate() error {
	return nil
}

// WorkflowBlocker describes a condition preventing workflow progress.
type WorkflowBlocker struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Validate ensures Type and Message are non-empty.
func (v *WorkflowBlocker) Validate() error {
	if v.Type == "" {
		return fmt.Errorf("capability: %T: field type is required", v)
	}
	if v.Message == "" {
		return fmt.Errorf("capability: %T: field message is required", v)
	}
	return nil
}

// ---------------------------------------------------------------------------
// CAPTCHA detection
// ---------------------------------------------------------------------------

var captchaIndicators = []string{
	"captcha",
	"recaptcha",
	"hcaptcha",
	"g-recaptcha",
	"h-captcha",
	"cf-turnstile",
	"arkose",
	"funcaptcha",
}

func NewCAPTABlocker(screenshotURL string) WorkflowBlocker {
	msg := "CAPTCHA detected during browser automation"
	if screenshotURL != "" {
		msg += " (screenshot available)"
	}
	return WorkflowBlocker{
		Type:       BlockerTypeCAPTCHA,
		Message:    msg,
		Suggestion: "Delegate to team lead or request human intervention via Matrix",
	}
}

func DetectCAPTCHA(pageContent string) bool {
	lower := strings.ToLower(pageContent)
	for _, indicator := range captchaIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}
