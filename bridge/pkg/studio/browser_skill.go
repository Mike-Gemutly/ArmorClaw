package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

//=============================================================================
// Browser Skill - Event-Based Protocol
//=============================================================================

// Event namespace prefix
const BrowserEventPrefix = "com.armorclaw.browser."

// Event types
const (
	// Commands (Client → Bridge)
	BrowserNavigate  = BrowserEventPrefix + "navigate"
	BrowserFill      = BrowserEventPrefix + "fill"
	BrowserClick     = BrowserEventPrefix + "click"
	BrowserWait      = BrowserEventPrefix + "wait"
	BrowserExtract   = BrowserEventPrefix + "extract"
	BrowserScreenshot = BrowserEventPrefix + "screenshot"

	// Responses (Bridge → Client)
	BrowserResponse = BrowserEventPrefix + "response"
	BrowserStatus   = BrowserEventPrefix + "status"
)

// WaitUntil conditions
type WaitUntil string

const (
	WaitUntilLoad             WaitUntil = "load"
	WaitUntilDOMContentLoaded WaitUntil = "domcontentloaded"
	WaitUntilNetworkIdle      WaitUntil = "networkidle"
)

// BrowserState represents the current browser state
type BrowserState string

const (
	BrowserStateLoading    BrowserState = "LOADING"
	BrowserStateFilling    BrowserState = "FILLING"
	BrowserStateWaiting    BrowserState = "WAITING"
	BrowserStateProcessing BrowserState = "PROCESSING"
	BrowserStateIdle       BrowserState = "IDLE"
	BrowserStateError      BrowserState = "ERROR"
)

// Error codes
type BrowserErrorCode string

const (
	ErrElementNotFound    BrowserErrorCode = "ELEMENT_NOT_FOUND"
	ErrNavigationFailed   BrowserErrorCode = "NAVIGATION_FAILED"
	ErrTimeout            BrowserErrorCode = "TIMEOUT"
	ErrPIIRequestDenied   BrowserErrorCode = "PII_REQUEST_DENIED"
	ErrInvalidSelector    BrowserErrorCode = "INVALID_SELECTOR"
	ErrBrowserNotReady    BrowserErrorCode = "BROWSER_NOT_READY"
	ErrExtractionFailed   BrowserErrorCode = "EXTRACTION_FAILED"
	ErrScreenshotFailed   BrowserErrorCode = "SCREENSHOT_FAILED"
)

//=============================================================================
// Command Events
//=============================================================================

// NavigateCommand loads a URL
type NavigateCommand struct {
	URL        string     `json:"url"`
	WaitUntil  WaitUntil  `json:"waitUntil,omitempty"`  // default: "load"
	Timeout    int        `json:"timeout,omitempty"`     // ms, default: 30000
}

// FillField represents a single form field
type FillField struct {
	Selector  string `json:"selector"`
	Value     string `json:"value,omitempty"`      // Static value
	ValueRef  string `json:"value_ref,omitempty"`  // BlindFill reference (e.g., "payment.card_number")
}

// FillCommand fills form fields
type FillCommand struct {
	Fields       []FillField `json:"fields"`
	AutoSubmit   bool        `json:"auto_submit,omitempty"`  // default: false
	SubmitDelay  int         `json:"submit_delay,omitempty"` // ms delay before submit
}

// ClickCommand clicks an element
type ClickCommand struct {
	Selector   string `json:"selector"`
	WaitFor    string `json:"waitFor,omitempty"`    // "none" | "navigation" | "selector"
	Timeout    int    `json:"timeout,omitempty"`    // ms
}

// WaitCondition types
type WaitCondition string

const (
	WaitConditionSelector WaitCondition = "selector"
	WaitConditionTimeout  WaitCondition = "timeout"
	WaitConditionURL      WaitCondition = "url"
)

// WaitCommand pauses until condition is met
type WaitCommand struct {
	Condition WaitCondition `json:"condition"` // "selector" | "timeout" | "url"
	Value     string        `json:"value"`
	Timeout   int           `json:"timeout,omitempty"` // ms, default: 5000
}

// ExtractField defines a field to extract
type ExtractField struct {
	Name     string `json:"name"`
	Selector string `json:"selector"`
	Attribute string `json:"attribute,omitempty"` // default: "textContent"
}

// ExtractCommand retrieves data from the page
type ExtractCommand struct {
	Fields []ExtractField `json:"fields"`
}

// ScreenshotCommand captures the page
type ScreenshotCommand struct {
	FullPage bool   `json:"fullPage,omitempty"`  // default: false
	Selector string `json:"selector,omitempty"`  // optional crop target
	Format   string `json:"format,omitempty"`    // "png" | "jpeg", default: "png"
}

//=============================================================================
// Response Events
//=============================================================================

// ResponseStatus indicates success or error
type BrowserResponseStatus string

const (
	BrowserResponseSuccess BrowserResponseStatus = "success"
	BrowserResponseError   BrowserResponseStatus = "error"
)

// BrowserCmdResponse is sent for every command
type BrowserCmdResponse struct {
	Status  BrowserResponseStatus `json:"status"`
	Command string                `json:"command"` // The command this responds to
	Data    interface{}           `json:"data,omitempty"`
	Error   *BrowserError         `json:"error,omitempty"`
}

// BrowserError contains error details
type BrowserError struct {
	Code       BrowserErrorCode `json:"code"`
	Message    string           `json:"message"`
	Screenshot string           `json:"screenshot,omitempty"` // base64 for visual context
	Selector   string           `json:"selector,omitempty"`  // if applicable
}

// NavigateResponseData contains navigation results
type NavigateResponseData struct {
	URL           string `json:"url"`
	Title         string `json:"title"`
	FinalURL      string `json:"finalUrl,omitempty"`  // after redirects
	LoadTime      int    `json:"loadTime,omitempty"`  // ms
}

// ExtractResponseData contains extracted values
type ExtractResponseData struct {
	Fields map[string]string `json:"fields"`
}

// ScreenshotResponseData contains the captured image
type ScreenshotResponseData struct {
	Image     string `json:"image"`     // base64 encoded
	Format    string `json:"format"`    // "png" | "jpeg"
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

//=============================================================================
// Status Events
//=============================================================================

// BrowserStatusEvent provides real-time feedback
type BrowserStatusEvent struct {
	State    BrowserState `json:"state"`
	Progress int          `json:"progress"` // 0-100
	Message  string       `json:"message"`
}

//=============================================================================
// PII Request Event
//=============================================================================

// PIIRequestEvent is sent when a command needs PII
type PIIRequestEvent struct {
	RequestID   string   `json:"request_id"`
	FieldRefs   []string `json:"field_refs"`  // e.g., ["payment.card_number", "payment.cvv"]
	Context     string   `json:"context"`     // Human-readable context
	Timeout     int      `json:"timeout"`     // seconds to respond
	Screenshot  string   `json:"screenshot,omitempty"` // Current page state
}

// PIIResponseEvent is the user's response
type PIIResponseEvent struct {
	RequestID string            `json:"request_id"`
	Approved  bool              `json:"approved"`
	Values    map[string]string `json:"values,omitempty"` // Only if approved
}

//=============================================================================
// PII Redaction
//=============================================================================

// PII patterns for automatic redaction
var piiPatterns = []*regexp.Regexp{
	// Credit Card (various formats)
	regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
	regexp.MustCompile(`\b\d{13,16}\b`),
	// SSN
	regexp.MustCompile(`\b\d{3}[-\s]?\d{2}[-\s]?\d{4}\b`),
	// Email (partial)
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
	// Phone numbers
	regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}\b`),
	// API Keys (common patterns)
	regexp.MustCompile(`\b(?:sk-|api-|key-)[a-zA-Z0-9]{20,}\b`),
}

// RedactPII replaces sensitive data with asterisks
func RedactPII(text string) string {
	for _, pattern := range piiPatterns {
		text = pattern.ReplaceAllStringFunc(text, func(match string) string {
			if len(match) <= 4 {
				return "****"
			}
			return match[:2] + strings.Repeat("*", len(match)-4) + match[len(match)-2:]
		})
	}
	return text
}

// ContainsPII checks if text likely contains PII
func ContainsPII(text string) bool {
	for _, pattern := range piiPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

//=============================================================================
// Browser Skill Handler
//=============================================================================

// BrowserEventHandler processes browser events
type BrowserEventHandler interface {
	// SendEvent sends an event to the client
	SendEvent(ctx context.Context, eventType string, content interface{}) error

	// RequestPII requests PII from the user
	RequestPII(ctx context.Context, req *PIIRequestEvent) (*PIIResponseEvent, error)

	// ExecuteBrowserCommand runs a browser automation command
	ExecuteBrowserCommand(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error)
}

// BrowserSkill implements the browser automation skill
type BrowserSkill struct {
	handler BrowserEventHandler
	store   Store
}

// NewBrowserSkill creates a new browser skill
func NewBrowserSkill(handler BrowserEventHandler, store Store) *BrowserSkill {
	return &BrowserSkill{
		handler: handler,
		store:   store,
	}
}

// HandleEvent processes an incoming browser event
func (s *BrowserSkill) HandleEvent(ctx context.Context, eventType string, content json.RawMessage) (*BrowserCmdResponse, error) {
	// Send status update
	s.sendStatus(ctx, BrowserStateProcessing, 0, "Processing "+eventType)

	switch eventType {
	case BrowserNavigate:
		return s.handleNavigate(ctx, content)
	case BrowserFill:
		return s.handleFill(ctx, content)
	case BrowserClick:
		return s.handleClick(ctx, content)
	case BrowserWait:
		return s.handleWait(ctx, content)
	case BrowserExtract:
		return s.handleExtract(ctx, content)
	case BrowserScreenshot:
		return s.handleScreenshot(ctx, content)
	default:
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: eventType,
			Error: &BrowserError{
				Code:    ErrBrowserNotReady,
				Message: "Unknown command: " + eventType,
			},
		}, nil
	}
}

func (s *BrowserSkill) handleNavigate(ctx context.Context, content json.RawMessage) (*BrowserCmdResponse, error) {
	var cmd NavigateCommand
	if err := json.Unmarshal(content, &cmd); err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "navigate",
			Error:   &BrowserError{Code: ErrElementNotFound, Message: "Invalid command: " + err.Error()},
		}, nil
	}

	// Validate URL
	if cmd.URL == "" {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "navigate",
			Error:   &BrowserError{Code: ErrNavigationFailed, Message: "URL is required"},
		}, nil
	}

	// Set defaults
	if cmd.WaitUntil == "" {
		cmd.WaitUntil = WaitUntilLoad
	}
	if cmd.Timeout == 0 {
		cmd.Timeout = 30000
	}

	// Send status
	s.sendStatus(ctx, BrowserStateLoading, 10, "Navigating to "+cmd.URL)

	// Execute navigation
	result, err := s.handler.ExecuteBrowserCommand(ctx, BrowserNavigate, content)
	if err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "navigate",
			Error:   &BrowserError{Code: ErrNavigationFailed, Message: err.Error()},
		}, nil
	}

	return &BrowserCmdResponse{
		Status:  BrowserResponseSuccess,
		Command: "navigate",
		Data:    result,
	}, nil
}

func (s *BrowserSkill) handleFill(ctx context.Context, content json.RawMessage) (*BrowserCmdResponse, error) {
	var cmd FillCommand
	if err := json.Unmarshal(content, &cmd); err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "fill",
			Error:   &BrowserError{Code: ErrElementNotFound, Message: "Invalid command: " + err.Error()},
		}, nil
	}

	// Check for PII references that need approval
	var piiRefs []string
	for _, field := range cmd.Fields {
		if field.ValueRef != "" {
			piiRefs = append(piiRefs, field.ValueRef)
		}
	}

	if len(piiRefs) > 0 {
		// Request PII approval
		s.sendStatus(ctx, BrowserStateWaiting, 30, "Requesting PII approval...")

		// Take screenshot for context
		screenshotResult, _ := s.handler.ExecuteBrowserCommand(ctx, BrowserScreenshot, json.RawMessage(`{}`))
		var screenshotB64 string
		if sr, ok := screenshotResult.(*ScreenshotResponseData); ok {
			screenshotB64 = sr.Image
		}

		piiResp, err := s.handler.RequestPII(ctx, &PIIRequestEvent{
			RequestID:  generateID("pii_req"),
			FieldRefs:  piiRefs,
			Context:    fmt.Sprintf("Form fill requires %d sensitive field(s)", len(piiRefs)),
			Timeout:    300, // 5 minutes
			Screenshot: screenshotB64,
		})

		if err != nil || !piiResp.Approved {
			return &BrowserCmdResponse{
				Status:  BrowserResponseError,
				Command: "fill",
				Error:   &BrowserError{Code: ErrPIIRequestDenied, Message: "PII request denied or timed out"},
			}, nil
		}

		// Inject PII values into fields
		for i, field := range cmd.Fields {
			if field.ValueRef != "" {
				if val, ok := piiResp.Values[field.ValueRef]; ok {
					cmd.Fields[i].Value = val
					cmd.Fields[i].ValueRef = "" // Clear reference
				}
			}
		}
	}

	// Execute fill
	s.sendStatus(ctx, BrowserStateFilling, 50, fmt.Sprintf("Filling %d field(s)...", len(cmd.Fields)))

	updatedContent, _ := json.Marshal(cmd)
	result, err := s.handler.ExecuteBrowserCommand(ctx, BrowserFill, updatedContent)
	if err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "fill",
			Error:   &BrowserError{Code: ErrElementNotFound, Message: err.Error()},
		}, nil
	}

	return &BrowserCmdResponse{
		Status:  BrowserResponseSuccess,
		Command: "fill",
		Data:    result,
	}, nil
}

func (s *BrowserSkill) handleClick(ctx context.Context, content json.RawMessage) (*BrowserCmdResponse, error) {
	var cmd ClickCommand
	if err := json.Unmarshal(content, &cmd); err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "click",
			Error:   &BrowserError{Code: ErrElementNotFound, Message: "Invalid command: " + err.Error()},
		}, nil
	}

	if cmd.Selector == "" {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "click",
			Error:   &BrowserError{Code: ErrInvalidSelector, Message: "Selector is required"},
		}, nil
	}

	s.sendStatus(ctx, BrowserStateProcessing, 50, "Clicking "+cmd.Selector)

	result, err := s.handler.ExecuteBrowserCommand(ctx, BrowserClick, content)
	if err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "click",
			Error:   &BrowserError{Code: ErrElementNotFound, Message: err.Error(), Selector: cmd.Selector},
		}, nil
	}

	return &BrowserCmdResponse{
		Status:  BrowserResponseSuccess,
		Command: "click",
		Data:    result,
	}, nil
}

func (s *BrowserSkill) handleWait(ctx context.Context, content json.RawMessage) (*BrowserCmdResponse, error) {
	var cmd WaitCommand
	if err := json.Unmarshal(content, &cmd); err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "wait",
			Error:   &BrowserError{Code: ErrElementNotFound, Message: "Invalid command: " + err.Error()},
		}, nil
	}

	if cmd.Timeout == 0 {
		cmd.Timeout = 5000
	}

	s.sendStatus(ctx, BrowserStateWaiting, 25, fmt.Sprintf("Waiting for %s...", cmd.Condition))

	result, err := s.handler.ExecuteBrowserCommand(ctx, BrowserWait, content)
	if err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "wait",
			Error:   &BrowserError{Code: ErrTimeout, Message: err.Error()},
		}, nil
	}

	return &BrowserCmdResponse{
		Status:  BrowserResponseSuccess,
		Command: "wait",
		Data:    result,
	}, nil
}

func (s *BrowserSkill) handleExtract(ctx context.Context, content json.RawMessage) (*BrowserCmdResponse, error) {
	var cmd ExtractCommand
	if err := json.Unmarshal(content, &cmd); err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "extract",
			Error:   &BrowserError{Code: ErrElementNotFound, Message: "Invalid command: " + err.Error()},
		}, nil
	}

	s.sendStatus(ctx, BrowserStateProcessing, 50, fmt.Sprintf("Extracting %d field(s)...", len(cmd.Fields)))

	result, err := s.handler.ExecuteBrowserCommand(ctx, BrowserExtract, content)
	if err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "extract",
			Error:   &BrowserError{Code: ErrExtractionFailed, Message: err.Error()},
		}, nil
	}

	// Redact PII from extracted values
	if extractData, ok := result.(*ExtractResponseData); ok {
		for k, v := range extractData.Fields {
			if ContainsPII(v) {
				extractData.Fields[k] = RedactPII(v)
			}
		}
	}

	return &BrowserCmdResponse{
		Status:  BrowserResponseSuccess,
		Command: "extract",
		Data:    result,
	}, nil
}

func (s *BrowserSkill) handleScreenshot(ctx context.Context, content json.RawMessage) (*BrowserCmdResponse, error) {
	s.sendStatus(ctx, BrowserStateProcessing, 50, "Capturing screenshot...")

	result, err := s.handler.ExecuteBrowserCommand(ctx, BrowserScreenshot, content)
	if err != nil {
		return &BrowserCmdResponse{
			Status:  BrowserResponseError,
			Command: "screenshot",
			Error:   &BrowserError{Code: ErrScreenshotFailed, Message: err.Error()},
		}, nil
	}

	return &BrowserCmdResponse{
		Status:  BrowserResponseSuccess,
		Command: "screenshot",
		Data:    result,
	}, nil
}

func (s *BrowserSkill) sendStatus(ctx context.Context, state BrowserState, progress int, message string) {
	if s.handler != nil {
		_ = s.handler.SendEvent(ctx, BrowserStatus, &BrowserStatusEvent{
			State:    state,
			Progress: progress,
			Message:  message,
		})
	}
}

//=============================================================================
// Audit Logging Helper
//=============================================================================

// BrowserAuditEntry represents an audit log entry for browser operations
type BrowserAuditEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Command     string    `json:"command"`
	Selector    string    `json:"selector,omitempty"`
	URL         string    `json:"url,omitempty"`
	Success     bool      `json:"success"`
	Duration    int       `json:"duration_ms"`
	PIIRequested bool     `json:"pii_requested,omitempty"`
}

// CreateAuditEntry creates an audit entry for a browser command
func CreateAuditEntry(command string, content json.RawMessage, success bool, duration time.Duration) *BrowserAuditEntry {
	entry := &BrowserAuditEntry{
		Timestamp: time.Now(),
		Command:   command,
		Success:   success,
		Duration:  int(duration.Milliseconds()),
	}

	// Extract selector/URL based on command type
	switch command {
	case "navigate":
		var cmd NavigateCommand
		if json.Unmarshal(content, &cmd) == nil {
			entry.URL = cmd.URL
		}
	case "click":
		var cmd ClickCommand
		if json.Unmarshal(content, &cmd) == nil {
			entry.Selector = cmd.Selector
		}
	case "wait":
		var cmd WaitCommand
		if json.Unmarshal(content, &cmd) == nil {
			entry.Selector = cmd.Value
		}
	}

	// Note: Never log field values, only selectors

	return entry
}
