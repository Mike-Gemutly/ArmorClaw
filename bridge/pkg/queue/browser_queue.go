// Package queue provides job queue implementations for browser automation tasks.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/agent"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// Browser Job Types
//=============================================================================

// JobStatus represents the current state of a browser job
type JobStatus string

const (
	JobStatusPending     JobStatus = "pending"
	JobStatusRunning     JobStatus = "running"
	JobStatusPaused      JobStatus = "paused"
	JobStatusCompleted   JobStatus = "completed"
	JobStatusFailed      JobStatus = "failed"
	JobStatusCancelled   JobStatus = "cancelled"
	JobStatusAwaitingPII JobStatus = "awaiting_pii"
)


// BrowserCommand represents a single browser automation command
type BrowserCommand struct {
	Type    string          `json:"type"`    // navigate, fill, click, wait, extract, screenshot
	Content json.RawMessage `json:"content"` // Command-specific payload
}

// BrowserJob represents a queued browser automation job
type BrowserJob struct {
	// Identification
	ID          string `json:"id"`
	AgentID     string `json:"agent_id"`
	RoomID      string `json:"room_id"`
	UserID      string `json:"user_id"`
	DefinitionID string `json:"definition_id,omitempty"`

	// Job configuration
	Commands    []BrowserCommand `json:"commands"`
	Priority    int              `json:"priority"` // Higher = more important
	Timeout     time.Duration    `json:"timeout"`
	MaxRetries  int              `json:"max_retries"`

	// State tracking
	Status      JobStatus  `json:"status"`
	Attempts    int        `json:"attempts"`
	CurrentStep int        `json:"current_step"`
	Error       string     `json:"error,omitempty"`

	// Timing
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`

	// Result
	Result      map[string]interface{} `json:"result,omitempty"`
	Screenshots []string               `json:"screenshots,omitempty"` // Base64 encoded

	// Internal state
	cancelFunc  context.CancelFunc `json:"-"`
	stateMachine *agent.StateMachine `json:"-"`
}

// IsExpired checks if the job has expired
func (j *BrowserJob) IsExpired() bool {
	return j.ExpiresAt != nil && time.Now().After(*j.ExpiresAt)
}

// IsTerminal returns true if the job is in a terminal state
func (j *BrowserJob) IsTerminal() bool {
	switch j.Status {
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

// CanRetry checks if the job can be retried
func (j *BrowserJob) CanRetry() bool {
	return j.Attempts < j.MaxRetries && j.Status == JobStatusFailed
}

//=============================================================================
// Browser Queue
//=============================================================================

// JobProcessor handles browser job execution
type JobProcessor interface {
	// ProcessJob executes a browser job
	ProcessJob(ctx context.Context, job *BrowserJob) error

	// CancelJob cancels a running job
	CancelJob(jobID string) error
}

// JobStatusEmitter emits job status updates
type JobStatusEmitter interface {
	// EmitStatus sends a status update for a job
	EmitStatus(ctx context.Context, job *BrowserJob) error
}

// BrowserQueueConfig holds configuration for the browser queue
type BrowserQueueConfig struct {
	// WorkerCount is the number of concurrent workers
	WorkerCount int

	// QueueSize is the maximum number of pending jobs
	QueueSize int

	// DefaultTimeout is the default job timeout
	DefaultTimeout time.Duration

	// DefaultMaxRetries is the default max retry count
	DefaultMaxRetries int

	// JobTTL is how long completed jobs are kept
	JobTTL time.Duration

	// Logger for queue operations
	Logger *logger.Logger
}

// DefaultBrowserQueueConfig returns sensible defaults
func DefaultBrowserQueueConfig() BrowserQueueConfig {
	return BrowserQueueConfig{
		WorkerCount:       3,
		QueueSize:         100,
		DefaultTimeout:    5 * time.Minute,
		DefaultMaxRetries: 2,
		JobTTL:           24 * time.Hour,
	}
}

// BrowserQueue manages browser automation job queue
type BrowserQueue struct {
	config     BrowserQueueConfig
	log        *logger.Logger

	// Job storage
	mu         sync.RWMutex
	jobs       map[string]*BrowserJob
	pending    chan *BrowserJob
	priority   []*BrowserJob // Priority queue for high-priority jobs

	// Worker management
	workers    int
	active     map[string]bool // jobID -> active
	processor  JobProcessor
	emitter    JobStatusEmitter

	// Lifecycle
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
}

// NewBrowserQueue creates a new browser job queue
func NewBrowserQueue(config BrowserQueueConfig, processor JobProcessor, emitter JobStatusEmitter) *BrowserQueue {
	if config.WorkerCount == 0 {
		config.WorkerCount = 3
	}
	if config.QueueSize == 0 {
		config.QueueSize = 100
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 5 * time.Minute
	}
	if config.DefaultMaxRetries == 0 {
		config.DefaultMaxRetries = 2
	}

	log := config.Logger
	if log == nil {
		log = logger.Global().WithComponent("browser_queue")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BrowserQueue{
		config:    config,
		log:       log,
		jobs:      make(map[string]*BrowserJob),
		pending:   make(chan *BrowserJob, config.QueueSize),
		priority:  make([]*BrowserJob, 0),
		active:    make(map[string]bool),
		processor: processor,
		emitter:   emitter,
		workers:   config.WorkerCount,
		ctx:       ctx,
		cancel:    cancel,
		running:   false,
	}
}

// Start begins processing jobs
func (q *BrowserQueue) Start() {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.mu.Unlock()

	// Start worker goroutines
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	q.log.Info("browser_queue_started", "workers", q.workers, "queue_size", q.config.QueueSize)
}

// Stop gracefully shuts down the queue
func (q *BrowserQueue) Stop() {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return
	}
	q.running = false
	q.mu.Unlock()

	// Cancel all active jobs
	for jobID := range q.active {
		if job, ok := q.jobs[jobID]; ok && job.cancelFunc != nil {
			job.cancelFunc()
		}
	}

	// Stop accepting new jobs and wait for workers
	q.cancel()
	q.wg.Wait()

	q.log.Info("browser_queue_stopped")
}

// Enqueue adds a new job to the queue
func (q *BrowserQueue) Enqueue(job *BrowserJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Validate job
	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}
	if job.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if len(job.Commands) == 0 {
		return fmt.Errorf("job must have at least one command")
	}

	// Check for duplicate
	if _, exists := q.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}

	// Set defaults
	if job.Status == "" {
		job.Status = JobStatusPending
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.Timeout == 0 {
		job.Timeout = q.config.DefaultTimeout
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = q.config.DefaultMaxRetries
	}

	// Store job
	q.jobs[job.ID] = job

	// Add to appropriate queue
	if job.Priority >= 10 {
		// High priority - add to priority queue
		q.priority = append(q.priority, job)
	} else {
		// Normal priority - add to channel
		select {
		case q.pending <- job:
			// Successfully enqueued
		default:
			// Queue full, remove job and return error
			delete(q.jobs, job.ID)
			return fmt.Errorf("queue is full (%d jobs)", q.config.QueueSize)
		}
	}

	q.log.Info("job_enqueued",
		"job_id", job.ID,
		"agent_id", job.AgentID,
		"priority", job.Priority,
		"commands", len(job.Commands))

	return nil
}

// Get retrieves a job by ID
func (q *BrowserQueue) Get(jobID string) (*BrowserJob, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

// Cancel cancels a job
func (q *BrowserQueue) Cancel(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}

	if job.IsTerminal() {
		return fmt.Errorf("job %s is already %s", jobID, job.Status)
	}

	// Cancel running job
	if q.active[jobID] && job.cancelFunc != nil {
		job.cancelFunc()
	}

	now := time.Now()
	job.Status = JobStatusCancelled
	job.CompletedAt = &now

	q.log.Info("job_cancelled", "job_id", jobID)

	// Emit status
	if q.emitter != nil {
		go q.emitter.EmitStatus(context.Background(), job)
	}

	return nil
}

// Retry retries a failed job
func (q *BrowserQueue) Retry(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}

	if !job.CanRetry() {
		return fmt.Errorf("job %s cannot be retried (status: %s, attempts: %d/%d)",
			jobID, job.Status, job.Attempts, job.MaxRetries)
	}

	// Reset job state
	job.Status = JobStatusPending
	job.Attempts++
	job.Error = ""
	job.StartedAt = nil
	job.CompletedAt = nil
	job.CurrentStep = 0
	job.Result = nil

	// Re-queue
	if job.Priority >= 10 {
		q.priority = append(q.priority, job)
	} else {
		select {
		case q.pending <- job:
		default:
			return fmt.Errorf("queue is full")
		}
	}

	q.log.Info("job_retried", "job_id", jobID, "attempt", job.Attempts)

	return nil
}

// Stats returns queue statistics
func (q *BrowserQueue) Stats() *QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := &QueueStats{
		Total: len(q.jobs),
	}

	for _, job := range q.jobs {
		switch job.Status {
		case JobStatusPending:
			stats.Pending++
		case JobStatusRunning:
			stats.Running++
		case JobStatusCompleted:
			stats.Completed++
		case JobStatusFailed:
			stats.Failed++
		case JobStatusCancelled:
			stats.Cancelled++
		case JobStatusAwaitingPII:
			stats.AwaitingPII++
		}
	}

	stats.ActiveWorkers = len(q.active)
	stats.QueueDepth = len(q.pending) + len(q.priority)

	return stats
}

// List returns jobs filtered by status
func (q *BrowserQueue) List(status ...JobStatus) []*BrowserJob {
	q.mu.RLock()
	defer q.mu.RUnlock()

	statusFilter := make(map[JobStatus]bool)
	for _, s := range status {
		statusFilter[s] = true
	}

	var result []*BrowserJob
	for _, job := range q.jobs {
		if len(statusFilter) == 0 || statusFilter[job.Status] {
			result = append(result, job)
		}
	}
	return result
}

// ListByAgent returns jobs for a specific agent
func (q *BrowserQueue) ListByAgent(agentID string) []*BrowserJob {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*BrowserJob
	for _, job := range q.jobs {
		if job.AgentID == agentID {
			result = append(result, job)
		}
	}
	return result
}

// Cleanup removes old completed/failed jobs
func (q *BrowserQueue) Cleanup() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	cutoff := time.Now().Add(-q.config.JobTTL)
	removed := 0

	for id, job := range q.jobs {
		if job.IsTerminal() && job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
			delete(q.jobs, id)
			removed++
		}
	}

	if removed > 0 {
		q.log.Info("queue_cleanup", "removed", removed, "remaining", len(q.jobs))
	}

	return removed
}

//=============================================================================
// Worker
//=============================================================================

// worker processes jobs from the queue
func (q *BrowserQueue) worker(id int) {
	defer q.wg.Done()

	log := q.log.With("worker_id", id)
	log.Debug("worker_started")

	for {
		select {
		case <-q.ctx.Done():
			log.Debug("worker_stopped")
			return

		case job := <-q.pending:
			// Check priority queue first
			q.mu.Lock()
			if len(q.priority) > 0 {
				// Take from priority queue instead
				priorityJob := q.priority[0]
				q.priority = q.priority[1:]
				// Put the regular job back
				select {
				case q.pending <- job:
				default:
					// Queue full, will be picked up next iteration
				}
				job = priorityJob
			}
			q.mu.Unlock()

			q.processJob(job)

		default:
			// Check priority queue
			q.mu.Lock()
			if len(q.priority) > 0 {
				job := q.priority[0]
				q.priority = q.priority[1:]
				q.mu.Unlock()
				q.processJob(job)
				continue
			}
			q.mu.Unlock()

			// Brief sleep to avoid busy waiting
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// processJob handles a single job
func (q *BrowserQueue) processJob(job *BrowserJob) {
	// Create job context with timeout
	ctx, cancel := context.WithTimeout(q.ctx, job.Timeout)
	defer cancel()

	q.mu.Lock()
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	job.cancelFunc = cancel
	q.active[job.ID] = true
	q.mu.Unlock()

	// Emit status
	if q.emitter != nil {
		q.emitter.EmitStatus(ctx, job)
	}

	q.log.Info("job_started",
		"job_id", job.ID,
		"agent_id", job.AgentID,
		"attempt", job.Attempts+1)

	// Execute job
	var err error
	if q.processor != nil {
		err = q.processor.ProcessJob(ctx, job)
	}

	// Update final status
	q.mu.Lock()
	delete(q.active, job.ID)
	now = time.Now()
	job.CompletedAt = &now

	if err != nil {
		if ctx.Err() == context.Canceled {
			job.Status = JobStatusCancelled
			job.Error = "job cancelled"
		} else {
			job.Status = JobStatusFailed
			job.Error = err.Error()
		}
		q.log.Error("job_failed",
			"job_id", job.ID,
			"error", job.Error,
			"status", job.Status)
	} else {
		job.Status = JobStatusCompleted
		q.log.Info("job_completed",
			"job_id", job.ID,
			"steps", len(job.Commands),
			"duration", job.CompletedAt.Sub(*job.StartedAt))
	}
	q.mu.Unlock()

	// Emit final status
	if q.emitter != nil {
		q.emitter.EmitStatus(context.Background(), job)
	}
}

//=============================================================================
// Statistics
//=============================================================================

// QueueStats holds queue statistics
type QueueStats struct {
	Total         int `json:"total"`
	Pending       int `json:"pending"`
	Running       int `json:"running"`
	Completed     int `json:"completed"`
	Failed        int `json:"failed"`
	Cancelled     int `json:"cancelled"`
	AwaitingPII   int `json:"awaiting_pii"`
	ActiveWorkers int `json:"active_workers"`
	QueueDepth    int `json:"queue_depth"`
}

//=============================================================================
// Browser Skill Processor
//=============================================================================

// BrowserSkillProcessor implements JobProcessor using studio.BrowserSkill
type BrowserSkillProcessor struct {
	skill    *studio.BrowserSkill
	stateMap map[string]*agent.StateMachine
	mu       sync.RWMutex
}

// NewBrowserSkillProcessor creates a processor using the studio browser skill
func NewBrowserSkillProcessor(skill *studio.BrowserSkill) *BrowserSkillProcessor {
	return &BrowserSkillProcessor{
		skill:    skill,
		stateMap: make(map[string]*agent.StateMachine),
	}
}

// ProcessJob executes a browser job using the browser skill
func (p *BrowserSkillProcessor) ProcessJob(ctx context.Context, job *BrowserJob) error {
	// Create or get state machine for this agent
	sm := p.getOrCreateStateMachine(job.AgentID)

	// Process each command
	for i, cmd := range job.Commands {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Update current step
		job.CurrentStep = i

		// Update state machine based on command type
		switch cmd.Type {
		case "navigate":
			var navCmd studio.NavigateCommand
			if err := json.Unmarshal(cmd.Content, &navCmd); err != nil {
				return fmt.Errorf("parse navigate command: %w", err)
			}
			if err := sm.StartBrowsing(navCmd.URL); err != nil {
				// Log but continue - state machine might already be in browsing state
			}

		case "fill":
			if err := sm.StartFormFilling(fmt.Sprintf("step_%d", i), i*100/len(job.Commands)); err != nil {
				// Log but continue
			}

		case "click", "wait", "extract", "screenshot":
			// These don't change state significantly
		}

		// Execute command via browser skill
		response, err := p.skill.HandleEvent(ctx, studio.BrowserEventPrefix+cmd.Type, cmd.Content)
		if err != nil {
			sm.FailWithString(err.Error())
			return fmt.Errorf("command %d (%s) failed: %w", i, cmd.Type, err)
		}

		if response.Status == studio.BrowserResponseError {
			sm.FailWithString(response.Error.Message)
			return fmt.Errorf("command %d (%s) error: %s", i, cmd.Type, response.Error.Message)
		}

		// Store result
		if response.Data != nil {
			if job.Result == nil {
				job.Result = make(map[string]interface{})
			}
			job.Result[fmt.Sprintf("step_%d", i)] = response.Data

			// Capture screenshots
			if screenshot, ok := response.Data.(*studio.ScreenshotResponseData); ok {
				job.Screenshots = append(job.Screenshots, screenshot.Image)
			}
		}
	}

	// Mark complete
	sm.Complete()

	return nil
}

// CancelJob cancels a running job
func (p *BrowserSkillProcessor) CancelJob(jobID string) error {
	// The cancel is handled by the context cancellation in processJob
	return nil
}

// getOrCreateStateMachine gets or creates a state machine for an agent
func (p *BrowserSkillProcessor) getOrCreateStateMachine(agentID string) *agent.StateMachine {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sm, ok := p.stateMap[agentID]; ok {
		return sm
	}

	sm := agent.NewStateMachine(agent.StateMachineConfig{
		AgentID:     agentID,
		HistorySize: 100,
	})
	p.stateMap[agentID] = sm

	return sm
}
