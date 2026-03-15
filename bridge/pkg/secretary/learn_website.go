package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/google/uuid"
)

//=============================================================================
// Learn Website Service
//=============================================================================

// LearnWebsiteService provides website form field discovery and mapping capabilities.
// It uses existing browser integration to inspect pages and discover form fields,
// then generates structured mapping drafts for user confirmation.
type LearnWebsiteService struct {
	browser *BrowserIntegration
	store   Store
	log     *logger.Logger
}

// LearnWebsiteConfig holds configuration for the learn website service
type LearnWebsiteConfig struct {
	Browser *BrowserIntegration
	Store   Store
	Logger  *logger.Logger
}

// NewLearnWebsiteService creates a new learn website service instance
func NewLearnWebsiteService(cfg LearnWebsiteConfig) (*LearnWebsiteService, error) {
	if cfg.Browser == nil {
		return nil, fmt.Errorf("browser integration is required")
	}
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("learn_website")
	}

	return &LearnWebsiteService{
		browser: cfg.Browser,
		store:   cfg.Store,
		log:     log,
	}, nil
}

//=============================================================================
// Request/Response Types
//=============================================================================

// LearnWebsiteRequest is the input for website learning
type LearnWebsiteRequest struct {
	// TargetURL is the URL to inspect for form fields
	TargetURL string `json:"target_url"`

	// Initiator is the Matrix user ID who initiated the request
	Initiator string `json:"initiator"`

	// WaitUntil specifies when to consider navigation complete
	WaitUntil string `json:"wait_until,omitempty"` // "load", "domcontentloaded", "networkidle"

	// Timeout in milliseconds for page load
	Timeout int `json:"timeout,omitempty"`

	// FormSelector optionally targets specific forms (CSS selector)
	FormSelector string `json:"form_selector,omitempty"`

	// IncludeHidden includes hidden fields in discovery
	IncludeHidden bool `json:"include_hidden,omitempty"`
}

// LearnWebsiteResult is the output from website learning
type LearnWebsiteResult struct {
	// RequestID is a unique identifier for this learning session
	RequestID string `json:"request_id"`

	// TargetURL is the URL that was inspected
	TargetURL string `json:"target_url"`

	// DiscoveredAt is when the discovery was performed
	DiscoveredAt time.Time `json:"discovered_at"`

	// Fields are the discovered form fields
	Fields []DiscoveredField `json:"fields"`

	// MappingDraft is the generated mapping draft for confirmation
	MappingDraft *MappingDraft `json:"mapping_draft"`

	// PageTitle is the title of the discovered page
	PageTitle string `json:"page_title,omitempty"`

	// Warnings contains any non-fatal issues encountered
	Warnings []string `json:"warnings,omitempty"`
}

// DiscoveredField represents a single discovered form field
type DiscoveredField struct {
	// ID is a unique identifier for this field within the discovery session
	ID string `json:"id"`

	// Selector is the CSS selector for the field
	Selector string `json:"selector"`

	// TagName is the HTML tag name (input, select, textarea, button)
	TagName string `json:"tag_name"`

	// InputType is the input type for input elements (text, email, password, etc.)
	InputType string `json:"input_type,omitempty"`

	// Name is the name attribute value
	Name string `json:"name,omitempty"`

	// ElementID is the id attribute value
	ElementID string `json:"element_id,omitempty"`

	// LabelText is the associated label text
	LabelText string `json:"label_text,omitempty"`

	// Placeholder is the placeholder attribute value
	Placeholder string `json:"placeholder,omitempty"`

	// Required indicates if the field is required
	Required bool `json:"required"`

	// Visible indicates if the field is visible
	Visible bool `json:"visible"`

	// Enabled indicates if the field is enabled
	Enabled bool `json:"enabled"`

	// Options contains options for select/radio/checkbox elements
	Options []FieldOption `json:"options,omitempty"`

	// AutofillHint is the autocomplete attribute value
	AutofillHint string `json:"autofill_hint,omitempty"`

	// AriaLabel is the aria-label attribute value
	AriaLabel string `json:"aria_label,omitempty"`

	// Confidence is the confidence score for selector reliability (0-100)
	Confidence int `json:"confidence"`
}

// FieldOption represents an option for select/radio/checkbox elements
type FieldOption struct {
	Value    string `json:"value"`
	Label    string `json:"label,omitempty"`
	Selected bool   `json:"selected,omitempty"`
}

// MappingDraft represents a structured mapping draft for user confirmation
type MappingDraft struct {
	// ID is a unique identifier for this draft
	ID string `json:"id"`

	// TargetURL is the URL this mapping applies to
	TargetURL string `json:"target_url"`

	// CreatedAt is when this draft was created
	CreatedAt time.Time `json:"created_at"`

	// CreatedBy is the Matrix user ID who created this draft
	CreatedBy string `json:"created_by"`

	// FieldMappings maps discovered fields to suggested PII references
	FieldMappings []FieldMapping `json:"field_mappings"`

	// Status is the current status of the draft
	Status MappingStatus `json:"status"`

	// TemplateName is the suggested template name
	TemplateName string `json:"template_name,omitempty"`

	// TemplateDescription is the suggested template description
	TemplateDescription string `json:"template_description,omitempty"`
}

// FieldMapping maps a discovered field to a PII reference
type FieldMapping struct {
	// FieldID references the discovered field
	FieldID string `json:"field_id"`

	// Selector is the CSS selector
	Selector string `json:"selector"`

	// FieldName is a human-readable name for the field
	FieldName string `json:"field_name"`

	// SuggestedPIIRef is the suggested PII reference (e.g., "user.email", "payment.card_number")
	SuggestedPIIRef string `json:"suggested_pii_ref,omitempty"`

	// ConfirmedPIIRef is the user-confirmed PII reference
	ConfirmedPIIRef string `json:"confirmed_pii_ref,omitempty"`

	// ActionType is the type of action (fill, select, click)
	ActionType string `json:"action_type"`

	// Required indicates if this field must be filled
	Required bool `json:"required"`

	// UserConfirmed indicates if the user has confirmed this mapping
	UserConfirmed bool `json:"user_confirmed"`
}

// MappingStatus represents the status of a mapping draft
type MappingStatus string

const (
	MappingStatusDraft     MappingStatus = "draft"
	MappingStatusPending   MappingStatus = "pending_confirmation"
	MappingStatusConfirmed MappingStatus = "confirmed"
	MappingStatusRejected  MappingStatus = "rejected"
)

// ConfirmedMapping represents a confirmed field mapping for persistence
type ConfirmedMapping struct {
	// DraftID references the original mapping draft
	DraftID string `json:"draft_id"`

	// TargetURL is the URL this mapping applies to
	TargetURL string `json:"target_url"`

	// TemplateName is the name for the created template
	TemplateName string `json:"template_name"`

	// TemplateDescription is the description for the created template
	TemplateDescription string `json:"template_description,omitempty"`

	// ConfirmedBy is the Matrix user ID who confirmed the mapping
	ConfirmedBy string `json:"confirmed_by"`

	// ConfirmedAt is when the mapping was confirmed
	ConfirmedAt time.Time `json:"confirmed_at"`

	// FieldMappings are the confirmed field mappings
	FieldMappings []ConfirmedFieldMapping `json:"field_mappings"`
}

// ConfirmedFieldMapping is a user-confirmed field mapping
type ConfirmedFieldMapping struct {
	// Selector is the CSS selector
	Selector string `json:"selector"`

	// FieldName is a human-readable name
	FieldName string `json:"field_name"`

	// PIIRef is the PII reference to use (e.g., "user.email")
	PIIRef string `json:"pii_ref,omitempty"`

	// ActionType is the type of action (fill, select, click)
	ActionType string `json:"action_type"`

	// StaticValue is a static value to use (for non-PII fields)
	StaticValue string `json:"static_value,omitempty"`

	// Order is the execution order
	Order int `json:"order"`
}

//=============================================================================
// Learn Website Errors
//=============================================================================

// LearnWebsiteError represents an error from the learn website service
type LearnWebsiteError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *LearnWebsiteError) Error() string {
	return e.Message
}

var (
	ErrInvalidURL         = &LearnWebsiteError{Code: "INVALID_URL", Message: "invalid target URL"}
	ErrNavigationFailed   = &LearnWebsiteError{Code: "NAVIGATION_FAILED", Message: "failed to navigate to target URL"}
	ErrExtractionFailed   = &LearnWebsiteError{Code: "EXTRACTION_FAILED", Message: "failed to extract form fields"}
	ErrNoFieldsDiscovered = &LearnWebsiteError{Code: "NO_FIELDS", Message: "no form fields discovered"}
	ErrDraftNotFound      = &LearnWebsiteError{Code: "DRAFT_NOT_FOUND", Message: "mapping draft not found"}
	ErrDraftAlreadyUsed   = &LearnWebsiteError{Code: "DRAFT_ALREADY_USED", Message: "mapping draft already confirmed or rejected"}
)

//=============================================================================
// Learn Method
//=============================================================================

// Learn navigates to a target URL and discovers form fields
func (s *LearnWebsiteService) Learn(ctx context.Context, req *LearnWebsiteRequest) (*LearnWebsiteResult, error) {
	// Validate request
	if err := s.validateLearnRequest(req); err != nil {
		return nil, err
	}

	requestID := uuid.New().String()
	s.log.Info("starting website learning",
		"request_id", requestID,
		"url", req.TargetURL,
		"initiator", req.Initiator)

	// Navigate to the target URL
	navCmd := s.buildNavigateCommand(req)
	navResult, err := s.browser.Navigate(ctx, navCmd)
	if err != nil {
		s.log.Error("navigation failed", "error", err, "url", req.TargetURL)
		return nil, fmt.Errorf("%w: %v", ErrNavigationFailed, err)
	}

	// Wait for page to be ready
	if req.WaitUntil != "" || req.Timeout > 0 {
		waitCmd := s.buildWaitCommand(req)
		if waitCmd != nil {
			_, err = s.browser.Wait(ctx, waitCmd)
			if err != nil {
				s.log.Warn("wait command failed, continuing anyway", "error", err)
			}
		}
	}

	// Extract form fields
	extractCmd := s.buildExtractCommand(req)
	extractResult, err := s.browser.Extract(ctx, extractCmd)
	if err != nil {
		s.log.Error("extraction failed", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrExtractionFailed, err)
	}

	// Parse extracted data into discovered fields
	fields, warnings := s.parseExtractedFields(extractResult, req.IncludeHidden)

	if len(fields) == 0 {
		return nil, ErrNoFieldsDiscovered
	}

	// Normalize fields and calculate confidence
	for i := range fields {
		s.normalizeField(&fields[i])
		s.calculateConfidence(&fields[i])
	}

	// Build mapping draft
	mappingDraft := s.buildMappingDraft(fields, req)

	// Extract page title if available
	var pageTitle string
	if navResult != nil {
		if title, ok := navResult["title"].(string); ok {
			pageTitle = title
		}
	}

	result := &LearnWebsiteResult{
		RequestID:    requestID,
		TargetURL:    req.TargetURL,
		DiscoveredAt: time.Now(),
		Fields:       fields,
		MappingDraft: mappingDraft,
		PageTitle:    pageTitle,
		Warnings:     warnings,
	}

	s.log.Info("website learning completed",
		"request_id", requestID,
		"fields_count", len(fields),
		"warnings_count", len(warnings))

	return result, nil
}

//=============================================================================
// ConfirmMapping Method
//=============================================================================

// ConfirmMapping persists a confirmed field mapping as a task template
func (s *LearnWebsiteService) ConfirmMapping(ctx context.Context, mapping *ConfirmedMapping) error {
	if err := s.validateConfirmedMapping(mapping); err != nil {
		return err
	}

	s.log.Info("confirming mapping",
		"draft_id", mapping.DraftID,
		"template_name", mapping.TemplateName,
		"confirmed_by", mapping.ConfirmedBy)

	// Build a TaskTemplate from the confirmed mapping
	template := s.buildTemplateFromMapping(mapping)

	// Create the template in the store
	if err := s.store.CreateTemplate(ctx, template); err != nil {
		s.log.Error("failed to persist template", "error", err)
		return fmt.Errorf("failed to persist confirmed mapping: %w", err)
	}

	s.log.Info("mapping confirmed and template created",
		"template_id", template.ID,
		"template_name", template.Name)

	return nil
}

//=============================================================================
// Helper Methods
//=============================================================================

func (s *LearnWebsiteService) validateLearnRequest(req *LearnWebsiteRequest) error {
	if req.TargetURL == "" {
		return ErrInvalidURL
	}

	// Basic URL validation
	if !strings.HasPrefix(req.TargetURL, "http://") && !strings.HasPrefix(req.TargetURL, "https://") {
		return ErrInvalidURL
	}

	if req.Initiator == "" {
		return fmt.Errorf("initiator is required")
	}

	return nil
}

func (s *LearnWebsiteService) validateConfirmedMapping(mapping *ConfirmedMapping) error {
	if mapping.TargetURL == "" {
		return fmt.Errorf("target URL is required")
	}
	if mapping.TemplateName == "" {
		return fmt.Errorf("template name is required")
	}
	if mapping.ConfirmedBy == "" {
		return fmt.Errorf("confirmed_by is required")
	}
	if len(mapping.FieldMappings) == 0 {
		return fmt.Errorf("at least one field mapping is required")
	}
	return nil
}

func (s *LearnWebsiteService) buildNavigateCommand(req *LearnWebsiteRequest) map[string]interface{} {
	cmd := map[string]interface{}{
		"url": req.TargetURL,
	}

	if req.WaitUntil != "" {
		cmd["waitUntil"] = req.WaitUntil
	}
	if req.Timeout > 0 {
		cmd["timeout"] = req.Timeout
	} else {
		cmd["timeout"] = 30000 // default 30 seconds
	}

	return cmd
}

func (s *LearnWebsiteService) buildWaitCommand(req *LearnWebsiteRequest) map[string]interface{} {
	if req.WaitUntil == "" {
		return nil
	}

	cmd := map[string]interface{}{
		"type": req.WaitUntil,
	}

	if req.Timeout > 0 {
		cmd["timeout"] = req.Timeout
	}

	return cmd
}

func (s *LearnWebsiteService) buildExtractCommand(req *LearnWebsiteRequest) map[string]interface{} {
	// Build extraction command to find form fields
	// The browser extract command should return field metadata
	fields := []map[string]interface{}{
		{
			"name":     "form_fields",
			"selector": s.getFieldSelector(req.FormSelector),
			"extract":  "form_fields",
		},
		{
			"name":      "page_title",
			"selector":  "title",
			"attribute": "text",
		},
	}

	return map[string]interface{}{
		"fields": fields,
	}
}

func (s *LearnWebsiteService) getFieldSelector(formSelector string) string {
	if formSelector != "" {
		return formSelector + " input, " + formSelector + " select, " + formSelector + " textarea, " + formSelector + " button"
	}
	return "input, select, textarea, button"
}

func (s *LearnWebsiteService) parseExtractedFields(result map[string]interface{}, includeHidden bool) ([]DiscoveredField, []string) {
	var fields []DiscoveredField
	var warnings []string

	if result == nil {
		return fields, warnings
	}

	// Extract form fields from result
	formFieldsData, ok := result["form_fields"]
	if !ok {
		warnings = append(warnings, "no form_fields key in extraction result")
		return fields, warnings
	}

	// Handle different result formats
	var formFields []map[string]interface{}

	switch v := formFieldsData.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				formFields = append(formFields, m)
			}
		}
	case map[string]interface{}:
		if items, ok := v["items"].([]interface{}); ok {
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					formFields = append(formFields, m)
				}
			}
		}
	}

	for i, ff := range formFields {
		field := DiscoveredField{
			ID: fmt.Sprintf("field_%d", i),
		}

		// Extract selector
		if sel, ok := ff["selector"].(string); ok {
			field.Selector = sel
		} else {
			// Generate selector from other attributes
			field.Selector = s.generateSelector(ff)
		}

		// Extract tag name
		if tag, ok := ff["tagName"].(string); ok {
			field.TagName = strings.ToLower(tag)
		} else if tag, ok := ff["tag"].(string); ok {
			field.TagName = strings.ToLower(tag)
		}

		// Extract input type
		if inputType, ok := ff["type"].(string); ok {
			field.InputType = strings.ToLower(inputType)
		}

		// Extract name attribute
		if name, ok := ff["name"].(string); ok {
			field.Name = name
		}

		// Extract id attribute
		if id, ok := ff["id"].(string); ok {
			field.ElementID = id
		}

		// Extract label text
		if label, ok := ff["label"].(string); ok {
			field.LabelText = label
		} else if label, ok := ff["labelText"].(string); ok {
			field.LabelText = label
		}

		// Extract placeholder
		if placeholder, ok := ff["placeholder"].(string); ok {
			field.Placeholder = placeholder
		}

		// Extract required flag
		if required, ok := ff["required"].(bool); ok {
			field.Required = required
		}

		// Extract visible flag
		if visible, ok := ff["visible"].(bool); ok {
			field.Visible = visible
		} else {
			field.Visible = true // assume visible by default
		}

		// Skip hidden fields if not included
		if !field.Visible && !includeHidden {
			continue
		}

		// Extract enabled flag
		if enabled, ok := ff["enabled"].(bool); ok {
			field.Enabled = enabled
		} else {
			field.Enabled = true
		}

		// Extract autocomplete hint
		if autofill, ok := ff["autocomplete"].(string); ok {
			field.AutofillHint = autofill
		}

		// Extract aria-label
		if ariaLabel, ok := ff["ariaLabel"].(string); ok {
			field.AriaLabel = ariaLabel
		}

		// Extract options for select/radio/checkbox
		if options, ok := ff["options"].([]interface{}); ok {
			for _, opt := range options {
				if optMap, ok := opt.(map[string]interface{}); ok {
					option := FieldOption{}
					if val, ok := optMap["value"].(string); ok {
						option.Value = val
					}
					if lbl, ok := optMap["label"].(string); ok {
						option.Label = lbl
					}
					if sel, ok := optMap["selected"].(bool); ok {
						option.Selected = sel
					}
					field.Options = append(field.Options, option)
				}
			}
		}

		fields = append(fields, field)
	}

	return fields, warnings
}

func (s *LearnWebsiteService) generateSelector(ff map[string]interface{}) string {
	// Try to generate a unique selector
	if id, ok := ff["id"].(string); ok && id != "" {
		return "#" + id
	}

	if name, ok := ff["name"].(string); ok && name != "" {
		tag := "input"
		if t, ok := ff["tagName"].(string); ok {
			tag = strings.ToLower(t)
		}
		return fmt.Sprintf("%s[name=\"%s\"]", tag, name)
	}

	// Fallback to tag with type
	tag := "input"
	if t, ok := ff["tagName"].(string); ok {
		tag = strings.ToLower(t)
	}

	inputType := ""
	if t, ok := ff["type"].(string); ok {
		inputType = strings.ToLower(t)
	}

	if inputType != "" {
		return fmt.Sprintf("%s[type=\"%s\"]", tag, inputType)
	}

	return tag
}

func (s *LearnWebsiteService) normalizeField(field *DiscoveredField) {
	// Clean up selector
	field.Selector = strings.TrimSpace(field.Selector)

	// Normalize tag name
	field.TagName = strings.ToLower(field.TagName)

	// Normalize input type
	field.InputType = strings.ToLower(field.InputType)

	// Clean up label text
	field.LabelText = strings.TrimSpace(field.LabelText)

	// Clean up placeholder
	field.Placeholder = strings.TrimSpace(field.Placeholder)

	// Ensure tag name is set
	if field.TagName == "" {
		field.TagName = "input"
	}

	// Set default input type for input elements
	if field.TagName == "input" && field.InputType == "" {
		field.InputType = "text"
	}
}

func (s *LearnWebsiteService) calculateConfidence(field *DiscoveredField) {
	confidence := 50 // base confidence

	// ID selector is most reliable
	if field.ElementID != "" {
		confidence += 30
	}

	// Name attribute is fairly reliable
	if field.Name != "" {
		confidence += 20
	}

	// Unique selector pattern
	if strings.HasPrefix(field.Selector, "#") {
		confidence += 10
	}

	// Has label text
	if field.LabelText != "" {
		confidence += 5
	}

	// Has autocomplete hint (good for matching)
	if field.AutofillHint != "" {
		confidence += 10
	}

	// Visible and enabled
	if field.Visible && field.Enabled {
		confidence += 5
	}

	// Cap at 100
	if confidence > 100 {
		confidence = 100
	}

	field.Confidence = confidence
}

func (s *LearnWebsiteService) buildMappingDraft(fields []DiscoveredField, req *LearnWebsiteRequest) *MappingDraft {
	draftID := uuid.New().String()

	var mappings []FieldMapping
	for _, field := range fields {
		mapping := FieldMapping{
			FieldID:       field.ID,
			Selector:      field.Selector,
			FieldName:     s.generateFieldName(field),
			ActionType:    s.determineActionType(field),
			Required:      field.Required,
			UserConfirmed: false,
		}

		// Suggest PII reference based on field attributes
		mapping.SuggestedPIIRef = s.suggestPIIRef(field)

		mappings = append(mappings, mapping)
	}

	// Generate template name from URL
	templateName := s.generateTemplateName(req.TargetURL)

	return &MappingDraft{
		ID:                  draftID,
		TargetURL:           req.TargetURL,
		CreatedAt:           time.Now(),
		CreatedBy:           req.Initiator,
		FieldMappings:       mappings,
		Status:              MappingStatusDraft,
		TemplateName:        templateName,
		TemplateDescription: fmt.Sprintf("Auto-generated template for %s", req.TargetURL),
	}
}

func (s *LearnWebsiteService) generateFieldName(field DiscoveredField) string {
	// Prefer label text
	if field.LabelText != "" {
		return field.LabelText
	}

	// Use placeholder
	if field.Placeholder != "" {
		return field.Placeholder
	}

	// Use name attribute
	if field.Name != "" {
		return strings.ReplaceAll(field.Name, "_", " ")
	}

	// Use id
	if field.ElementID != "" {
		return strings.ReplaceAll(field.ElementID, "_", " ")
	}

	// Fallback
	return field.TagName + " field"
}

func (s *LearnWebsiteService) determineActionType(field DiscoveredField) string {
	switch field.TagName {
	case "select":
		return "select"
	case "button":
		return "click"
	case "input":
		switch field.InputType {
		case "checkbox", "radio":
			return "click"
		case "submit", "button":
			return "click"
		default:
			return "fill"
		}
	case "textarea":
		return "fill"
	default:
		return "fill"
	}
}

func (s *LearnWebsiteService) suggestPIIRef(field DiscoveredField) string {
	// Use autocomplete hint if available
	autocompleteMap := map[string]string{
		"email":            "user.email",
		"username":         "user.username",
		"current-password": "user.password",
		"new-password":     "user.new_password",
		"given-name":       "user.first_name",
		"family-name":      "user.last_name",
		"name":             "user.full_name",
		"tel":              "user.phone",
		"street-address":   "user.address.street",
		"address-level2":   "user.address.city",
		"address-level1":   "user.address.state",
		"postal-code":      "user.address.zip",
		"country":          "user.address.country",
		"cc-number":        "payment.card_number",
		"cc-exp":           "payment.card_expiry",
		"cc-csc":           "payment.card_cvv",
		"cc-name":          "payment.card_name",
	}

	if ref, ok := autocompleteMap[field.AutofillHint]; ok {
		return ref
	}

	// Infer from field name/id
	nameLower := strings.ToLower(field.Name + " " + field.ElementID + " " + field.LabelText + " " + field.Placeholder)

	patterns := []struct {
		pattern string
		ref     string
	}{
		{`email|e-mail`, "user.email"},
		{`password|passwd|pwd`, "user.password"},
		{`username|user_name|login`, "user.username"},
		{`first.*name|fname`, "user.first_name"},
		{`last.*name|lname`, "user.last_name"},
		{`phone|tel|mobile`, "user.phone"},
		{`address|street`, "user.address.street"},
		{`city`, "user.address.city"},
		{`state|province`, "user.address.state"},
		{`zip|postal`, "user.address.zip"},
		{`country`, "user.address.country"},
		{`card.*number|cc.*num|credit.*card`, "payment.card_number"},
		{`cvv|cvc|security.*code`, "payment.card_cvv"},
		{`expir|exp.*date`, "payment.card_expiry"},
		{`card.*name|name.*card`, "payment.card_name"},
	}

	for _, p := range patterns {
		matched, _ := regexp.MatchString(p.pattern, nameLower)
		if matched {
			return p.ref
		}
	}

	// No suggestion available
	return ""
}

func (s *LearnWebsiteService) generateTemplateName(targetURL string) string {
	// Extract domain from URL
	parts := strings.Split(targetURL, "/")
	if len(parts) >= 3 {
		domain := parts[2]
		// Remove www prefix
		domain = strings.TrimPrefix(domain, "www.")
		return fmt.Sprintf("%s_form_template", domain)
	}
	return fmt.Sprintf("form_template_%s", uuid.New().String()[:8])
}

func (s *LearnWebsiteService) buildTemplateFromMapping(mapping *ConfirmedMapping) *TaskTemplate {
	templateID := uuid.New().String()
	now := time.Now()

	// Build workflow steps from field mappings
	var steps []WorkflowStep
	var piiRefs []string

	for i, fm := range mapping.FieldMappings {
		stepID := fmt.Sprintf("step_%d", i)

		config := s.buildStepConfig(fm)

		step := WorkflowStep{
			StepID:      stepID,
			Order:       i,
			Type:        StepAction,
			Name:        fm.FieldName,
			Description: fmt.Sprintf("%s for %s", fm.ActionType, fm.FieldName),
			Config:      config,
		}

		// Set next step ID
		if i < len(mapping.FieldMappings)-1 {
			step.NextStepID = fmt.Sprintf("step_%d", i+1)
		}

		steps = append(steps, step)

		// Collect PII refs
		if fm.PIIRef != "" {
			piiRefs = append(piiRefs, fm.PIIRef)
		}
	}

	// Build variables schema
	variables := s.buildVariablesSchema(mapping.FieldMappings)

	return &TaskTemplate{
		ID:          templateID,
		Name:        mapping.TemplateName,
		Description: mapping.TemplateDescription,
		Steps:       steps,
		Variables:   variables,
		PIIRefs:     piiRefs,
		CreatedBy:   mapping.ConfirmedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsActive:    true,
	}
}

func (s *LearnWebsiteService) buildStepConfig(fm ConfirmedFieldMapping) json.RawMessage {
	switch fm.ActionType {
	case "fill":
		config := map[string]interface{}{
			"action": "fill",
			"params": map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"selector": fm.Selector,
					},
				},
			},
		}

		// Use PIIRef or static value
		if fm.PIIRef != "" {
			config["params"].(map[string]interface{})["fields"].([]map[string]interface{})[0]["value_ref"] = fm.PIIRef
		} else if fm.StaticValue != "" {
			config["params"].(map[string]interface{})["fields"].([]map[string]interface{})[0]["value"] = fm.StaticValue
		}

		data, _ := json.Marshal(config)
		return data

	case "click":
		config := map[string]interface{}{
			"action": "click",
			"params": map[string]interface{}{
				"selector": fm.Selector,
			},
		}
		data, _ := json.Marshal(config)
		return data

	case "select":
		config := map[string]interface{}{
			"action": "fill",
			"params": map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"selector": fm.Selector,
					},
				},
			},
		}

		if fm.PIIRef != "" {
			config["params"].(map[string]interface{})["fields"].([]map[string]interface{})[0]["value_ref"] = fm.PIIRef
		} else if fm.StaticValue != "" {
			config["params"].(map[string]interface{})["fields"].([]map[string]interface{})[0]["value"] = fm.StaticValue
		}

		data, _ := json.Marshal(config)
		return data

	default:
		config := map[string]interface{}{
			"action": fm.ActionType,
			"params": map[string]interface{}{
				"selector": fm.Selector,
			},
		}
		data, _ := json.Marshal(config)
		return data
	}
}

func (s *LearnWebsiteService) buildVariablesSchema(mappings []ConfirmedFieldMapping) json.RawMessage {
	properties := make(map[string]interface{})

	for _, fm := range mappings {
		if fm.PIIRef != "" {
			// Create variable name from PII ref
			varName := strings.ReplaceAll(fm.PIIRef, ".", "_")
			properties[varName] = map[string]interface{}{
				"type":        "string",
				"description": fm.FieldName,
			}
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	data, _ := json.Marshal(schema)
	return data
}
