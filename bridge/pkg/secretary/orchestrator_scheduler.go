package secretary

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

//=============================================================================
// Scheduler Errors
//=============================================================================

var (
	ErrScheduledJobNotFound = fmt.Errorf("scheduled job not found")
	ErrScheduledJobInactive = fmt.Errorf("scheduled job is inactive")
	ErrInvalidScheduleTime  = fmt.Errorf("invalid schedule time")
	ErrSchedulerNotRunning  = fmt.Errorf("scheduler not running")
)

//=============================================================================
// Scheduled Job Types
//=============================================================================

type ScheduleType string

const (
	ScheduleOnce      ScheduleType = "once"
	ScheduleRecurring ScheduleType = "recurring"
)

type ScheduledJob struct {
	ID           string                 `json:"id"`
	TemplateID   string                 `json:"template_id"`
	ScheduleType ScheduleType           `json:"schedule_type"`
	ScheduledAt  time.Time              `json:"scheduled_at"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	CreatedBy    string                 `json:"created_by"`
	CreatedAt    time.Time              `json:"created_at"`
	Cancelled    bool                   `json:"cancelled"`
	WorkflowID   string                 `json:"workflow_id,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
}

type ScheduledJobConfig struct {
	ID           string
	TemplateID   string
	ScheduleType ScheduleType
	ScheduledAt  time.Time
	Variables    map[string]interface{}
	CreatedBy    string
}

//=============================================================================
// Scheduler Event Types
//=============================================================================

const (
	SchedulerEventJobScheduled = "scheduler.job_scheduled"
	SchedulerEventJobStarted   = "scheduler.job_started"
	SchedulerEventJobCompleted = "scheduler.job_completed"
	SchedulerEventJobCancelled = "scheduler.job_cancelled"
	SchedulerEventJobFailed    = "scheduler.job_failed"
)

//=============================================================================
// Scheduler
//=============================================================================

type Scheduler struct {
	mu            sync.RWMutex
	store         Store
	orchestrator  *WorkflowOrchestratorImpl
	eventEmitter  EventEmitter
	scheduledJobs map[string]*ScheduledJob
	ticker        *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
	running       bool
	log           *logger.Logger
	tickInterval  time.Duration
	location      *time.Location // Parsed timezone for time conversion
	timezone      string         // Configured IANA timezone string
}

type SchedulerConfig struct {
	Store        Store
	Orchestrator *WorkflowOrchestratorImpl
	EventEmitter EventEmitter
	TickInterval time.Duration
	Logger       *logger.Logger
	Timezone     string // IANA timezone (e.g., "America/New_York")
}

func NewScheduler(cfg SchedulerConfig) (*Scheduler, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if cfg.Orchestrator == nil {
		return nil, fmt.Errorf("orchestrator is required")
	}
	if cfg.EventEmitter == nil {
		return nil, fmt.Errorf("event emitter is required")
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("scheduler")
	}

	tickInterval := cfg.TickInterval
	if tickInterval == 0 {
		tickInterval = 1 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		store:         cfg.Store,
		orchestrator:  cfg.Orchestrator,
		eventEmitter:  cfg.EventEmitter,
		scheduledJobs: make(map[string]*ScheduledJob),
		ctx:           ctx,
		cancel:        cancel,
		log:           log,
		tickInterval:  tickInterval,
	}, nil
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	if s.timezone != "" {
		loc, err := time.LoadLocation(s.timezone)
		if err == nil {
			s.location = loc
			s.log.Info("scheduler_timezone_loaded", "timezone", s.timezone)
		} else {
			s.log.Warn("scheduler_timezone_load_failed", "timezone", s.timezone, "error", err.Error())
		}
	}

	s.ticker = time.NewTicker(s.tickInterval)
	s.running = true

	go s.run()

	s.log.Info("scheduler_started", "tick_interval", s.tickInterval.String(), "timezone", s.timezone)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.cancel()

	s.log.Info("scheduler_stopped")
}

func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Scheduler) run() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			s.processScheduledJobs()
		}
	}
}

func (s *Scheduler) processScheduledJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().In(s.location)

	for id, job := range s.scheduledJobs {
		if job.Cancelled {
			delete(s.scheduledJobs, id)
			continue
		}

		if job.StartedAt != nil {
			continue
		}

		if now.After(job.ScheduledAt) || now.Equal(job.ScheduledAt) {
			go s.executeJob(job)
		}
	}
}

func (s *Scheduler) executeJob(job *ScheduledJob) {
	s.mu.Lock()
	now := time.Now()
	job.StartedAt = &now
	s.mu.Unlock()

	workflow, err := s.createWorkflowFromJob(job)
	if err != nil {
		s.mu.Lock()
		job.Cancelled = true
		s.mu.Unlock()

		s.log.Error("scheduled_job_failed", "job_id", job.ID, "error", err.Error())
		s.emitSchedulerEvent(SchedulerEventJobFailed, job, err.Error())
		return
	}

	job.WorkflowID = workflow.ID

	if err := s.orchestrator.StartWorkflow(workflow.ID); err != nil {
		s.mu.Lock()
		job.Cancelled = true
		s.mu.Unlock()

		s.log.Error("scheduled_job_start_failed", "job_id", job.ID, "workflow_id", workflow.ID, "error", err.Error())
		s.emitSchedulerEvent(SchedulerEventJobFailed, job, err.Error())
		return
	}

	s.log.Info("scheduled_job_started", "job_id", job.ID, "workflow_id", workflow.ID)
	s.emitSchedulerEvent(SchedulerEventJobStarted, job, "")

	if job.ScheduleType == ScheduleOnce {
		s.mu.Lock()
		delete(s.scheduledJobs, job.ID)
		s.mu.Unlock()
	}
}

func (s *Scheduler) createWorkflowFromJob(job *ScheduledJob) (*Workflow, error) {
	template, err := s.store.GetTemplate(s.ctx, job.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	workflow := &Workflow{
		ID:          generateSchedulerID("wf"),
		TemplateID:  job.TemplateID,
		Name:        template.Name + " (scheduled)",
		Description: template.Description,
		Status:      StatusPending,
		Variables:   job.Variables,
		CreatedBy:   job.CreatedBy,
		StartedAt:   time.Now(),
	}

	if err := s.store.CreateWorkflow(s.ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow, nil
}

func (s *Scheduler) ScheduleJob(cfg ScheduledJobConfig) (*ScheduledJob, error) {
	if cfg.TemplateID == "" {
		return nil, fmt.Errorf("template_id is required")
	}

	if cfg.ScheduledAt.IsZero() {
		return nil, ErrInvalidScheduleTime
	}

	if cfg.ScheduledAt.Before(time.Now()) {
		return nil, fmt.Errorf("scheduled time cannot be in the past")
	}

	template, err := s.store.GetTemplate(s.ctx, cfg.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return nil, fmt.Errorf("template not found: %s", cfg.TemplateID)
	}

	job := &ScheduledJob{
		ID:           cfg.ID,
		TemplateID:   cfg.TemplateID,
		ScheduleType: cfg.ScheduleType,
		ScheduledAt:  cfg.ScheduledAt,
		Variables:    cfg.Variables,
		CreatedBy:    cfg.CreatedBy,
		CreatedAt:    time.Now(),
		Cancelled:    false,
	}

	if job.ID == "" {
		job.ID = generateSchedulerID("job")
	}

	if job.ScheduleType == "" {
		job.ScheduleType = ScheduleOnce
	}

	s.mu.Lock()
	s.scheduledJobs[job.ID] = job
	s.mu.Unlock()

	s.log.Info("job_scheduled", "job_id", job.ID, "template_id", job.TemplateID, "scheduled_at", job.ScheduledAt.Format(time.RFC3339))
	s.emitSchedulerEvent(SchedulerEventJobScheduled, job, "")

	return job, nil
}

func (s *Scheduler) CancelJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.scheduledJobs[jobID]
	if !exists {
		return ErrScheduledJobNotFound
	}

	if job.StartedAt != nil {
		return fmt.Errorf("cannot cancel job that has already started")
	}

	job.Cancelled = true
	delete(s.scheduledJobs, jobID)

	s.log.Info("job_cancelled", "job_id", jobID)
	s.emitSchedulerEvent(SchedulerEventJobCancelled, job, "")

	return nil
}

func (s *Scheduler) GetJob(jobID string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.scheduledJobs[jobID]
	if !exists {
		return nil, ErrScheduledJobNotFound
	}

	copy := *job
	return &copy, nil
}

func (s *Scheduler) ListJobs() []*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ScheduledJob, 0, len(s.scheduledJobs))
	for _, job := range s.scheduledJobs {
		if !job.Cancelled {
			copy := *job
			result = append(result, &copy)
		}
	}
	return result
}

func (s *Scheduler) GetJobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, job := range s.scheduledJobs {
		if !job.Cancelled {
			count++
		}
	}
	return count
}

func (s *Scheduler) GetPendingJobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	now := time.Now()
	for _, job := range s.scheduledJobs {
		if !job.Cancelled && job.StartedAt == nil && job.ScheduledAt.After(now) {
			count++
		}
	}
	return count
}

func (s *Scheduler) GetDueJobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	now := time.Now()
	for _, job := range s.scheduledJobs {
		if !job.Cancelled && job.StartedAt == nil && (now.After(job.ScheduledAt) || now.Equal(job.ScheduledAt)) {
			count++
		}
	}
	return count
}

func (s *Scheduler) emitSchedulerEvent(eventType string, job *ScheduledJob, errMsg string) {
	event := map[string]interface{}{
		"event_type":    eventType,
		"job_id":        job.ID,
		"template_id":   job.TemplateID,
		"schedule_type": job.ScheduleType,
		"scheduled_at":  job.ScheduledAt.UnixMilli(),
		"created_by":    job.CreatedBy,
		"timestamp":     time.Now().UnixMilli(),
	}

	if job.WorkflowID != "" {
		event["workflow_id"] = job.WorkflowID
	}

	if job.StartedAt != nil {
		event["started_at"] = job.StartedAt.UnixMilli()
	}

	if errMsg != "" {
		event["error"] = errMsg
	}

	workflow := &Workflow{
		ID:         job.ID,
		TemplateID: job.TemplateID,
		Status:     StatusPending,
		CreatedBy:  job.CreatedBy,
		StartedAt:  job.ScheduledAt,
	}

	switch eventType {
	case SchedulerEventJobScheduled:
		s.eventEmitter.EmitProgress(workflow, job.ID, "scheduled", 0.0)
	case SchedulerEventJobStarted:
		s.eventEmitter.EmitStarted(workflow)
	case SchedulerEventJobCancelled:
		s.eventEmitter.EmitCancelled(workflow, "cancelled by user")
	case SchedulerEventJobFailed:
		s.eventEmitter.EmitFailed(workflow, job.ID, fmt.Errorf("%s", errMsg), false)
	case SchedulerEventJobCompleted:
		s.eventEmitter.EmitCompleted(workflow, "job completed")
	}
}

func (s *Scheduler) Shutdown() {
	s.Stop()

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, job := range s.scheduledJobs {
		if job.StartedAt == nil {
			job.Cancelled = true
			s.eventEmitter.EmitCancelled(&Workflow{
				ID:         job.ID,
				TemplateID: job.TemplateID,
				CreatedBy:  job.CreatedBy,
			}, "scheduler shutdown")
		}
		delete(s.scheduledJobs, id)
	}

	s.log.Info("scheduler_shutdown_complete")
}

//=============================================================================
// ID Generation
//=============================================================================

var schedulerIDCounter int64

func generateSchedulerID(prefix string) string {
	schedulerIDCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), schedulerIDCounter)
}
