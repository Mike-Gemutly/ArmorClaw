// Package browser provides HTTP client for browser-service communication
package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

//=============================================================================
// HTTP Client Types
//=============================================================================

// ServiceWaitUntil specifies when to consider navigation complete
type ServiceWaitUntil string

const (
	ServiceWaitUntilLoad            ServiceWaitUntil = "load"
	ServiceWaitUntilDOMContentLoaded ServiceWaitUntil = "domcontentloaded"
	ServiceWaitUntilNetworkIdle     ServiceWaitUntil = "networkidle"
)

// ServiceState represents the current state of the browser session
type ServiceState string

const (
	ServiceStateIdle       ServiceState = "IDLE"
	ServiceStateLoading    ServiceState = "LOADING"
	ServiceStateFilling    ServiceState = "FILLING"
	ServiceStateWaiting    ServiceState = "WAITING"
	ServiceStateProcessing ServiceState = "PROCESSING"
	ServiceStateError      ServiceState = "ERROR"
)

// ServiceErrorCode represents error codes returned by the browser service
type ServiceErrorCode string

const (
	ServiceErrElementNotFound   ServiceErrorCode = "ELEMENT_NOT_FOUND"
	ServiceErrNavigationFailed  ServiceErrorCode = "NAVIGATION_FAILED"
	ServiceErrTimeout           ServiceErrorCode = "TIMEOUT"
	ServiceErrPIIRequestDenied  ServiceErrorCode = "PII_REQUEST_DENIED"
	ServiceErrInvalidSelector   ServiceErrorCode = "INVALID_SELECTOR"
	ServiceErrBrowserNotReady   ServiceErrorCode = "BROWSER_NOT_READY"
	ServiceErrExtractionFailed  ServiceErrorCode = "EXTRACTION_FAILED"
	ServiceErrScreenshotFailed  ServiceErrorCode = "SCREENSHOT_FAILED"
	ServiceErrSessionExpired    ServiceErrorCode = "SESSION_EXPIRED"
	ServiceErrCaptchaRequired   ServiceErrorCode = "CAPTCHA_REQUIRED"
	ServiceErrTwoFARequired     ServiceErrorCode = "TWO_FA_REQUIRED"
)

// ServiceNavigateCommand represents a navigation request
type ServiceNavigateCommand struct {
	URL       string           `json:"url"`
	WaitUntil ServiceWaitUntil `json:"waitUntil,omitempty"`
	Timeout   int              `json:"timeout,omitempty"` // ms, default: 30000
}

// ServiceFillField represents a single form field to fill
type ServiceFillField struct {
	Selector string `json:"selector"`
	Value    string `json:"value,omitempty"`
	ValueRef string `json:"value_ref,omitempty"` // PII reference
}

// ServiceFillCommand represents a form fill request
type ServiceFillCommand struct {
	Fields      []ServiceFillField `json:"fields"`
	AutoSubmit  bool               `json:"auto_submit,omitempty"`
	SubmitDelay int                `json:"submit_delay,omitempty"` // ms
}

// ServiceClickCommand represents a click request
type ServiceClickCommand struct {
	Selector string `json:"selector"`
	WaitFor  string `json:"waitFor,omitempty"` // "none", "navigation", "selector"
	Timeout  int    `json:"timeout,omitempty"` // ms
}

// ServiceWaitCommand represents a wait request
type ServiceWaitCommand struct {
	Condition string `json:"condition"` // "selector", "timeout", "url"
	Value     string `json:"value"`
	Timeout   int    `json:"timeout,omitempty"` // ms
}

// ServiceExtractField represents a field to extract from the page
type ServiceExtractField struct {
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Attribute string `json:"attribute,omitempty"` // default: "textContent"
}

// ServiceExtractCommand represents a data extraction request
type ServiceExtractCommand struct {
	Fields []ServiceExtractField `json:"fields"`
}

// ServiceScreenshotCommand represents a screenshot request
type ServiceScreenshotCommand struct {
	FullPage bool   `json:"fullPage,omitempty"`
	Selector string `json:"selector,omitempty"`
	Format   string `json:"format,omitempty"` // "png" or "jpeg"
}

// ServiceWorkflowStep represents a single step in a workflow
type ServiceWorkflowStep struct {
	Action string `json:"action"` // navigate, fill, click, wait, extract, screenshot

	// Embedded command fields (only one used per action)
	URL       string             `json:"url,omitempty"`
	WaitUntil ServiceWaitUntil   `json:"waitUntil,omitempty"`
	Timeout   int                `json:"timeout,omitempty"`

	Fields      []ServiceFillField `json:"fields,omitempty"`
	AutoSubmit  bool               `json:"auto_submit,omitempty"`
	SubmitDelay int                `json:"submit_delay,omitempty"`

	Selector string `json:"selector,omitempty"`
	WaitFor  string `json:"waitFor,omitempty"`

	Condition string `json:"condition,omitempty"`
	Value     string `json:"value,omitempty"`

	// Extract fields
	Name      string `json:"name,omitempty"`
	Attribute string `json:"attribute,omitempty"`

	FullPage bool   `json:"fullPage,omitempty"`
	Format   string `json:"format,omitempty"`
}

// ServiceWorkflowCommand represents a multi-step workflow request
type ServiceWorkflowCommand struct {
	Steps []ServiceWorkflowStep `json:"steps"`
}

// ServiceError represents an error from the browser service
type ServiceError struct {
	Code       ServiceErrorCode `json:"code"`
	Message    string           `json:"message"`
	Selector   string           `json:"selector,omitempty"`
	Screenshot string           `json:"screenshot,omitempty"` // base64
}

// Error implements the error interface
func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ServiceResponse represents a response from the browser service
type ServiceResponse struct {
	Success    bool                   `json:"success"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Error      *ServiceError          `json:"error,omitempty"`
	Duration   int                    `json:"duration"` // ms
	Screenshot string                 `json:"screenshot,omitempty"` // base64 when error
}

// ServiceHealthResponse represents the health check response
type ServiceHealthResponse struct {
	Status    string       `json:"status"`
	Browser   string       `json:"browser"`
	State     ServiceState `json:"state"`
	Timestamp time.Time    `json:"timestamp"`
}

// ServiceSessionInfo represents session info from the service
type ServiceSessionInfo struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActivity time.Time `json:"lastActivity"`
	State        string    `json:"state"`
	CurrentURL   string    `json:"currentUrl,omitempty"`
}

// ServiceSessionResponse represents the session info response
type ServiceSessionResponse struct {
	Success bool               `json:"success"`
	Data    *ServiceSessionInfo `json:"data,omitempty"`
	Error   string             `json:"error,omitempty"`
}

// ServiceWorkflowResponse represents the workflow execution response
type ServiceWorkflowResponse struct {
	Success  bool              `json:"success"`
	Data     *ServiceWorkflowData `json:"data,omitempty"`
	Duration int               `json:"duration"`
}

// ServiceWorkflowData contains workflow execution results
type ServiceWorkflowData struct {
	Steps          []ServiceResponse `json:"steps"`
	TotalSteps     int               `json:"totalSteps"`
	CompletedSteps int               `json:"completedSteps"`
}

// ServiceInterventionType represents the type of intervention detected
type ServiceInterventionType string

const (
	ServiceInterventionCaptcha    ServiceInterventionType = "captcha"
	ServiceInterventionTwoFA      ServiceInterventionType = "twofa"
	ServiceInterventionUnexpected ServiceInterventionType = "unexpected"
	ServiceInterventionBlocked    ServiceInterventionType = "blocked"
)

//=============================================================================
// HTTP Client
//=============================================================================

// Client is the HTTP client for browser-service communication
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new browser service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for browser operations
		},
	}
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

//=============================================================================
// API Methods
//=============================================================================

// Health checks the browser service health
func (c *Client) Health(ctx context.Context) (*ServiceHealthResponse, error) {
	var resp ServiceHealthResponse
	if err := c.get(ctx, "/health", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Initialize initializes the browser
func (c *Client) Initialize(ctx context.Context) error {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	return c.post(ctx, "/initialize", nil, &resp)
}

// Close closes the browser
func (c *Client) Close(ctx context.Context) error {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	return c.post(ctx, "/close", nil, &resp)
}

// Session gets the current browser session info
func (c *Client) Session(ctx context.Context) (*ServiceSessionResponse, error) {
	var resp ServiceSessionResponse
	if err := c.get(ctx, "/session", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Navigate navigates to a URL
func (c *Client) Navigate(ctx context.Context, cmd ServiceNavigateCommand) (*ServiceResponse, error) {
	var resp ServiceResponse
	if err := c.post(ctx, "/navigate", cmd, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Fill fills form fields
func (c *Client) Fill(ctx context.Context, cmd ServiceFillCommand) (*ServiceResponse, error) {
	var resp ServiceResponse
	if err := c.post(ctx, "/fill", cmd, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Click clicks an element
func (c *Client) Click(ctx context.Context, cmd ServiceClickCommand) (*ServiceResponse, error) {
	var resp ServiceResponse
	if err := c.post(ctx, "/click", cmd, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Wait waits for a condition
func (c *Client) Wait(ctx context.Context, cmd ServiceWaitCommand) (*ServiceResponse, error) {
	var resp ServiceResponse
	if err := c.post(ctx, "/wait", cmd, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Extract extracts data from the page
func (c *Client) Extract(ctx context.Context, cmd ServiceExtractCommand) (*ServiceResponse, error) {
	var resp ServiceResponse
	if err := c.post(ctx, "/extract", cmd, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Screenshot takes a screenshot
func (c *Client) Screenshot(ctx context.Context, cmd ServiceScreenshotCommand) (*ServiceResponse, error) {
	var resp ServiceResponse
	if err := c.post(ctx, "/screenshot", cmd, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Workflow executes a multi-step workflow
func (c *Client) Workflow(ctx context.Context, cmd ServiceWorkflowCommand) (*ServiceWorkflowResponse, error) {
	var resp ServiceWorkflowResponse
	if err := c.post(ctx, "/workflow", cmd, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

//=============================================================================
// HTTP Helpers
//=============================================================================

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.doRequest(req, out)
}

func (c *Client) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.doRequest(req, out)
}

func (c *Client) doRequest(req *http.Request, out interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
