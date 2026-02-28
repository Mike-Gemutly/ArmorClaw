package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

//=============================================================================
// RPC Error Codes
//=============================================================================

const (
	RPCErrInvalidParams  = -32602
	RPCErrInternal       = -32603
	RPCErrNotFound       = -32001
	RPCErrValidation     = -32002
	RPCErrConflict       = -32004
	RPCErrQueueFull      = -32010
)

//=============================================================================
// RPC Request/Response Types
//=============================================================================

// RPCRequest represents a JSON-RPC 2.0 request
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	UserID  string          `json:"-"` // Set by auth middleware
}

// RPCResponse represents a JSON-RPC 2.0 response
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SuccessResponse creates a successful RPC response
func SuccessResponse(result interface{}) *RPCResponse {
	return &RPCResponse{
		JSONRPC: "2.0",
		Result:  result,
	}
}

// ErrorResponse creates an error RPC response
func ErrorResponse(code int, message string, data ...interface{}) *RPCResponse {
	resp := &RPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	if len(data) > 0 {
		resp.Error.Data = data[0]
	}
	return resp
}

//=============================================================================
// Queue RPC Handler
//=============================================================================

// QueueRPCHandler provides JSON-RPC handlers for browser queue operations
type QueueRPCHandler struct {
	queue *BrowserQueue
	log   *logger.Logger
}

// NewQueueRPCHandler creates a new queue RPC handler
func NewQueueRPCHandler(queue *BrowserQueue, log *logger.Logger) *QueueRPCHandler {
	if log == nil {
		log = logger.Global().WithComponent("queue_rpc")
	}
	return &QueueRPCHandler{
		queue: queue,
		log:   log,
	}
}

// Handle routes requests to appropriate handlers
func (h *QueueRPCHandler) Handle(req *RPCRequest) *RPCResponse {
	switch req.Method {
	// Job Management
	case "browser.enqueue":
		return h.handleEnqueue(req)
	case "browser.get_job":
		return h.handleGetJob(req)
	case "browser.list_jobs":
		return h.handleListJobs(req)
	case "browser.cancel_job":
		return h.handleCancelJob(req)
	case "browser.retry_job":
		return h.handleRetryJob(req)

	// Queue Management
	case "browser.queue_stats":
		return h.handleQueueStats(req)
	case "browser.queue_cleanup":
		return h.handleQueueCleanup(req)

	// Queue Control
	case "browser.queue_start":
		return h.handleQueueStart(req)
	case "browser.queue_stop":
		return h.handleQueueStop(req)

	default:
		return ErrorResponse(RPCErrNotFound, fmt.Sprintf("Unknown method: %s", req.Method))
	}
}

//=============================================================================
// Job Management Handlers
//=============================================================================

// EnqueueParams for browser.enqueue
type EnqueueParams struct {
	ID           string            `json:"id"`
	AgentID      string            `json:"agent_id"`
	RoomID       string            `json:"room_id"`
	DefinitionID string            `json:"definition_id,omitempty"`
	Commands     []BrowserCommand  `json:"commands"`
	Priority     int               `json:"priority,omitempty"`
	Timeout      int               `json:"timeout,omitempty"`    // seconds
	MaxRetries   int               `json:"max_retries,omitempty"`
	ExpiresIn    int               `json:"expires_in,omitempty"` // seconds from now
}

func (h *QueueRPCHandler) handleEnqueue(req *RPCRequest) *RPCResponse {
	var params EnqueueParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(RPCErrInvalidParams, "Invalid params: "+err.Error())
	}

	// Validate required fields
	if params.AgentID == "" {
		return ErrorResponse(RPCErrInvalidParams, "agent_id is required")
	}
	if len(params.Commands) == 0 {
		return ErrorResponse(RPCErrInvalidParams, "commands are required")
	}

	// Create job
	job := &BrowserJob{
		ID:           params.ID,
		AgentID:      params.AgentID,
		RoomID:       params.RoomID,
		UserID:       req.UserID,
		DefinitionID: params.DefinitionID,
		Commands:     params.Commands,
		Priority:     params.Priority,
		MaxRetries:   params.MaxRetries,
		CreatedAt:    time.Now(),
	}

	// Set timeout
	if params.Timeout > 0 {
		job.Timeout = time.Duration(params.Timeout) * time.Second
	}

	// Set expiration
	if params.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(params.ExpiresIn) * time.Second)
		job.ExpiresAt = &expiresAt
	}

	// Enqueue
	if err := h.queue.Enqueue(job); err != nil {
		if err.Error() == fmt.Sprintf("queue is full (%d jobs)", h.queue.config.QueueSize) {
			return ErrorResponse(RPCErrQueueFull, err.Error())
		}
		return ErrorResponse(RPCErrInternal, "Failed to enqueue job: "+err.Error())
	}

	h.log.Info("job_enqueued_via_rpc",
		"job_id", job.ID,
		"agent_id", job.AgentID,
		"user_id", req.UserID,
		"priority", job.Priority)

	return SuccessResponse(map[string]interface{}{
		"job_id":     job.ID,
		"status":     job.Status,
		"created_at": job.CreatedAt.UnixMilli(),
		"message":    "Job enqueued successfully",
	})
}

// GetJobParams for browser.get_job
type GetJobParams struct {
	ID string `json:"id"`
}

func (h *QueueRPCHandler) handleGetJob(req *RPCRequest) *RPCResponse {
	var params GetJobParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(RPCErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(RPCErrInvalidParams, "Job ID is required")
	}

	job, err := h.queue.Get(params.ID)
	if err != nil {
		return ErrorResponse(RPCErrNotFound, "Job not found: "+params.ID)
	}

	return SuccessResponse(h.jobToResponse(job))
}

// ListJobsParams for browser.list_jobs
type ListJobsParams struct {
	AgentID string     `json:"agent_id,omitempty"`
	Status  []JobStatus `json:"status,omitempty"`
}

func (h *QueueRPCHandler) handleListJobs(req *RPCRequest) *RPCResponse {
	var params ListJobsParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(RPCErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	var jobs []*BrowserJob
	if params.AgentID != "" {
		jobs = h.queue.ListByAgent(params.AgentID)
	} else {
		jobs = h.queue.List(params.Status...)
	}

	// Convert to response format
	result := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		result[i] = h.jobToResponse(job)
	}

	return SuccessResponse(map[string]interface{}{
		"jobs":  result,
		"count": len(result),
	})
}

// CancelJobParams for browser.cancel_job
type CancelJobParams struct {
	ID string `json:"id"`
}

func (h *QueueRPCHandler) handleCancelJob(req *RPCRequest) *RPCResponse {
	var params CancelJobParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(RPCErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(RPCErrInvalidParams, "Job ID is required")
	}

	if err := h.queue.Cancel(params.ID); err != nil {
		return ErrorResponse(RPCErrInternal, "Failed to cancel job: "+err.Error())
	}

	h.log.Info("job_cancelled_via_rpc", "job_id", params.ID, "user_id", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"job_id":  params.ID,
		"status":  JobStatusCancelled,
		"message": "Job cancelled successfully",
	})
}

// RetryJobParams for browser.retry_job
type RetryJobParams struct {
	ID string `json:"id"`
}

func (h *QueueRPCHandler) handleRetryJob(req *RPCRequest) *RPCResponse {
	var params RetryJobParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(RPCErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(RPCErrInvalidParams, "Job ID is required")
	}

	if err := h.queue.Retry(params.ID); err != nil {
		return ErrorResponse(RPCErrInternal, "Failed to retry job: "+err.Error())
	}

	h.log.Info("job_retried_via_rpc", "job_id", params.ID, "user_id", req.UserID)

	job, _ := h.queue.Get(params.ID)
	return SuccessResponse(map[string]interface{}{
		"job_id":   params.ID,
		"status":   job.Status,
		"attempts": job.Attempts,
		"message":  "Job requeued successfully",
	})
}

//=============================================================================
// Queue Management Handlers
//=============================================================================

func (h *QueueRPCHandler) handleQueueStats(req *RPCRequest) *RPCResponse {
	stats := h.queue.Stats()
	return SuccessResponse(stats)
}

// CleanupParams for browser.queue_cleanup
type CleanupParams struct {
	TTLHours int `json:"ttl_hours,omitempty"`
}

func (h *QueueRPCHandler) handleQueueCleanup(req *RPCRequest) *RPCResponse {
	var params CleanupParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(RPCErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	// Note: TTL override not implemented, uses config default
	removed := h.queue.Cleanup()

	h.log.Info("queue_cleanup_via_rpc", "removed", removed, "user_id", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"removed":  removed,
		"message":  fmt.Sprintf("Cleaned up %d old jobs", removed),
	})
}

//=============================================================================
// Queue Control Handlers
//=============================================================================

func (h *QueueRPCHandler) handleQueueStart(req *RPCRequest) *RPCResponse {
	h.queue.Start()
	return SuccessResponse(map[string]interface{}{
		"status":  "started",
		"workers": h.queue.workers,
		"message": "Queue started successfully",
	})
}

func (h *QueueRPCHandler) handleQueueStop(req *RPCRequest) *RPCResponse {
	h.queue.Stop()
	return SuccessResponse(map[string]interface{}{
		"status":  "stopped",
		"message": "Queue stopped successfully",
	})
}

//=============================================================================
// Helper Functions
//=============================================================================

// jobToResponse converts a job to a response-friendly format
func (h *QueueRPCHandler) jobToResponse(job *BrowserJob) map[string]interface{} {
	result := map[string]interface{}{
		"id":            job.ID,
		"agent_id":      job.AgentID,
		"room_id":       job.RoomID,
		"user_id":       job.UserID,
		"definition_id": job.DefinitionID,
		"status":        job.Status,
		"priority":      job.Priority,
		"attempts":      job.Attempts,
		"current_step":  job.CurrentStep,
		"total_steps":   len(job.Commands),
		"created_at":    job.CreatedAt.UnixMilli(),
	}

	if job.StartedAt != nil {
		result["started_at"] = job.StartedAt.UnixMilli()
	}
	if job.CompletedAt != nil {
		result["completed_at"] = job.CompletedAt.UnixMilli()
	}
	if job.ExpiresAt != nil {
		result["expires_at"] = job.ExpiresAt.UnixMilli()
	}
	if job.Error != "" {
		result["error"] = job.Error
	}
	if job.Result != nil {
		result["result"] = job.Result
	}
	if len(job.Screenshots) > 0 {
		result["screenshots"] = len(job.Screenshots)
	}

	return result
}

//=============================================================================
// Integration Helper
//=============================================================================

// Integration provides helpers for integrating the queue with other systems
type Integration struct {
	queue   *BrowserQueue
	handler *QueueRPCHandler
}

// NewIntegration creates a new integration helper
func NewIntegration(queue *BrowserQueue, log *logger.Logger) *Integration {
	return &Integration{
		queue:   queue,
		handler: NewQueueRPCHandler(queue, log),
	}
}

// GetQueue returns the underlying queue
func (i *Integration) GetQueue() *BrowserQueue {
	return i.queue
}

// GetHandler returns the RPC handler
func (i *Integration) GetHandler() *QueueRPCHandler {
	return i.handler
}

// ProcessMatrixCommand processes a Matrix command to create a browser job
func (i *Integration) ProcessMatrixCommand(ctx context.Context, userID, roomID, command string) (string, error) {
	// Parse command: !browser <url> [options]
	// This is a simplified implementation
	return "", fmt.Errorf("not implemented")
}

// EnqueueFromAgent creates a job from an agent instance
func (i *Integration) EnqueueFromAgent(ctx context.Context, agentID, definitionID, roomID string, commands []BrowserCommand) (*BrowserJob, error) {
	job := &BrowserJob{
		ID:           fmt.Sprintf("job_%d", time.Now().UnixMilli()),
		AgentID:      agentID,
		DefinitionID: definitionID,
		RoomID:       roomID,
		Commands:     commands,
		Priority:     5,
		CreatedAt:    time.Now(),
	}

	if err := i.queue.Enqueue(job); err != nil {
		return nil, err
	}

	return job, nil
}
