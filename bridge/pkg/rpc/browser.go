// Package rpc provides browser automation RPC methods.
// These methods expose the BrowserSkill to external callers via JSON-RPC.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/browser"
)

// BrowserJob represents an active browser automation job
type BrowserJob struct {
	ID           string
	AgentID      string
	URL          string
	Status       string // "running", "completed", "failed"
	CreatedAt    time.Time
	CompletedAt  *time.Time
	Error        string
	Session      *browser.BrowserSession
	skill        *browser.BrowserSkill
	cancelFunc   context.CancelFunc
}

// BrowserJobManager manages active browser jobs
type BrowserJobManager struct {
	mu      sync.RWMutex
	jobs    map[string]*BrowserJob
	emitter browser.StatusEmitter
}

// NewBrowserJobManager creates a new job manager
func NewBrowserJobManager() *BrowserJobManager {
	return &BrowserJobManager{
		jobs: make(map[string]*BrowserJob),
	}
}

// SetEmitter sets the status emitter for new jobs
func (m *BrowserJobManager) SetEmitter(emitter browser.StatusEmitter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emitter = emitter
}

// CreateJob creates a new browser job
func (m *BrowserJobManager) CreateJob(id, agentID string) *BrowserJob {
	m.mu.Lock()
	defer m.mu.Unlock()

	job := &BrowserJob{
		ID:        id,
		AgentID:   agentID,
		Status:    "idle",
		CreatedAt: time.Now(),
		skill:     browser.NewBrowserSkill(agentID, m.emitter),
	}
	m.jobs[id] = job
	return job
}

// GetJob retrieves a job by ID
func (m *BrowserJobManager) GetJob(id string) (*BrowserJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, exists := m.jobs[id]
	return job, exists
}

// RemoveJob removes a job
func (m *BrowserJobManager) RemoveJob(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, exists := m.jobs[id]; exists {
		if job.cancelFunc != nil {
			job.cancelFunc()
		}
	}
	delete(m.jobs, id)
}

// ListJobs returns all active jobs
func (m *BrowserJobManager) ListJobs() []*BrowserJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]*BrowserJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// CleanupOldJobs removes jobs older than the specified duration
func (m *BrowserJobManager) CleanupOldJobs(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, job := range m.jobs {
		if job.CreatedAt.Before(cutoff) && (job.Status == "completed" || job.Status == "failed") {
			delete(m.jobs, id)
			removed++
		}
	}
	return removed
}

// generateID creates a simple unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// handleBrowserNavigate handles browser.navigate RPC method
// Starts a new browser navigation job and returns the job ID
func (s *Server) handleBrowserNavigate(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		URL     string `json:"url"`
		AgentID string `json:"agent_id"`
		JobID   string `json:"job_id,omitempty"` // Optional, will be generated if not provided
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.URL == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "url is required",
		}
	}

	if params.AgentID == "" {
		params.AgentID = "default"
	}

	// Generate or use provided job ID
	jobID := params.JobID
	if jobID == "" {
		jobID = "browser_" + generateID()
	}

	// Create job
	job := s.browserJobs.CreateJob(jobID, params.AgentID)
	job.URL = params.URL
	job.Status = "running"

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	job.cancelFunc = cancel

	// Start navigation in background
	go func() {
		defer cancel()

		err := job.skill.Navigate(ctx, params.URL)
		if err != nil {
			job.Status = "failed"
			job.Error = err.Error()
		} else {
			job.Status = "completed"
		}

		now := time.Now()
		job.CompletedAt = &now
		job.Session = job.skill.GetSession()
	}()

	return map[string]interface{}{
		"job_id":    jobID,
		"status":    "running",
		"url":       params.URL,
		"agent_id":  params.AgentID,
		"message":   "Navigation started. Use browser.status to poll for completion.",
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}, nil
}

// handleBrowserFill handles browser.fill RPC method
// Fills a form field in an active browser job
func (s *Server) handleBrowserFill(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID    string `json:"job_id"`
		Selector string `json:"selector"`
		Value    string `json:"value"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	if params.Selector == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "selector is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	if job.Status == "failed" {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "job has failed: " + job.Error,
		}
	}

	// Fill form
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := job.skill.FillForm(ctx, params.Selector, params.Value)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to fill form: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"job_id":   params.JobID,
		"status":   "filled",
		"selector": params.Selector,
		"success":  true,
	}, nil
}

// handleBrowserClick handles browser.click RPC method
// Clicks an element in an active browser job
func (s *Server) handleBrowserClick(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID    string `json:"job_id"`
		Selector string `json:"selector"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	if params.Selector == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "selector is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	if job.Status == "failed" {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "job has failed: " + job.Error,
		}
	}

	// Click element
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := job.skill.Click(ctx, params.Selector)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to click: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"job_id":   params.JobID,
		"status":   "clicked",
		"selector": params.Selector,
		"success":  true,
	}, nil
}

// handleBrowserStatus handles browser.status RPC method
// Polls for job completion and returns current state
func (s *Server) handleBrowserStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	result := map[string]interface{}{
		"job_id":     job.ID,
		"agent_id":   job.AgentID,
		"status":     job.Status,
		"url":        job.URL,
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}

	if job.CompletedAt != nil {
		result["completed_at"] = job.CompletedAt.Format(time.RFC3339)
	}

	if job.Error != "" {
		result["error"] = job.Error
	}

	if job.Session != nil {
		result["session"] = map[string]interface{}{
			"id":            job.Session.ID,
			"url":           job.Session.URL,
			"status":        string(job.Session.Status),
			"title":         job.Session.Title,
			"last_activity": job.Session.LastActivity.Format(time.RFC3339),
		}
	}

	// Include screenshot placeholder (would be base64 in real implementation)
	// In a full implementation, this would capture actual screenshots
	result["screenshot_available"] = false

	return result, nil
}

// handleBrowserWaitForElement handles browser.wait_for_element RPC method
// Waits for an element to appear on the page
func (s *Server) handleBrowserWaitForElement(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID    string `json:"job_id"`
		Selector string `json:"selector"`
		Timeout  int    `json:"timeout"` // seconds
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	if params.Selector == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "selector is required",
		}
	}

	if params.Timeout == 0 {
		params.Timeout = 30
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Wait for element
	ctx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	err := job.skill.WaitForElement(ctx, params.Selector, time.Duration(params.Timeout)*time.Second)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "element not found: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"job_id":   params.JobID,
		"status":   "element_found",
		"selector": params.Selector,
		"success":  true,
	}, nil
}

// handleBrowserWaitForCaptcha handles browser.wait_for_captcha RPC method
// Signals that the agent is waiting for captcha resolution
func (s *Server) handleBrowserWaitForCaptcha(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Emit captcha status
	ctx = ctx
	job.skill.WaitForCaptcha(ctx)

	return map[string]interface{}{
		"job_id":  params.JobID,
		"status":  "awaiting_captcha",
		"message": "Captcha detection started. Mobile app will be notified.",
	}, nil
}

// handleBrowserWaitFor2FA handles browser.wait_for_2fa RPC method
// Signals that the agent is waiting for 2FA code
func (s *Server) handleBrowserWaitFor2FA(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Emit 2FA status
	ctx = ctx
	job.skill.WaitFor2FA(ctx)

	return map[string]interface{}{
		"job_id":  params.JobID,
		"status":  "awaiting_2fa",
		"message": "2FA detection started. Mobile app will be notified.",
	}, nil
}

// handleBrowserComplete handles browser.complete RPC method
// Marks a browser job as complete
func (s *Server) handleBrowserComplete(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Mark complete
	ctx = ctx
	job.skill.Complete(ctx)
	job.Status = "completed"
	now := time.Now()
	job.CompletedAt = &now
	job.Session = job.skill.GetSession()

	return map[string]interface{}{
		"job_id":       params.JobID,
		"status":       "completed",
		"completed_at": job.CompletedAt.Format(time.RFC3339),
		"success":      true,
	}, nil
}

// handleBrowserFail handles browser.fail RPC method
// Marks a browser job as failed
func (s *Server) handleBrowserFail(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID  string `json:"job_id"`
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Mark failed
	ctx = ctx
	job.skill.Fail(ctx, fmt.Errorf("%s", params.Reason))
	job.Status = "failed"
	job.Error = params.Reason
	now := time.Now()
	job.CompletedAt = &now

	return map[string]interface{}{
		"job_id":  params.JobID,
		"status":  "failed",
		"error":   params.Reason,
		"success": true,
	}, nil
}

// handleBrowserList handles browser.list RPC method
// Lists all active browser jobs
func (s *Server) handleBrowserList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	jobs := s.browserJobs.ListJobs()

	result := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		jobInfo := map[string]interface{}{
			"job_id":     job.ID,
			"agent_id":   job.AgentID,
			"url":        job.URL,
			"status":     job.Status,
			"created_at": job.CreatedAt.Format(time.RFC3339),
		}
		if job.CompletedAt != nil {
			jobInfo["completed_at"] = job.CompletedAt.Format(time.RFC3339)
		}
		if job.Error != "" {
			jobInfo["error"] = job.Error
		}
		result = append(result, jobInfo)
	}

	return map[string]interface{}{
		"jobs":  result,
		"count": len(result),
	}, nil
}

// handleBrowserCancel handles browser.cancel RPC method
// Cancels an active browser job
func (s *Server) handleBrowserCancel(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Cancel job
	if job.cancelFunc != nil {
		job.cancelFunc()
	}
	job.Status = "cancelled"
	now := time.Now()
	job.CompletedAt = &now

	return map[string]interface{}{
		"job_id":       params.JobID,
		"status":       "cancelled",
		"cancelled_at": job.CompletedAt.Format(time.RFC3339),
		"success":      true,
	}, nil
}
