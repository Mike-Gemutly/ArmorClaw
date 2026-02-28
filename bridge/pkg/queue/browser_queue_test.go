package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// Mock Implementations
//=============================================================================

type mockProcessor struct {
	mu         sync.Mutex
	processed  []*BrowserJob
	errors     map[string]error
	delays     map[string]time.Duration
	cancelled  []string
}

func newMockProcessor() *mockProcessor {
	return &mockProcessor{
		errors:    make(map[string]error),
		delays:    make(map[string]time.Duration),
	}
}

func (m *mockProcessor) ProcessJob(ctx context.Context, job *BrowserJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for simulated error
	if err, ok := m.errors[job.ID]; ok {
		return err
	}

	// Simulate delay
	if delay, ok := m.delays[job.ID]; ok {
		select {
		case <-ctx.Done():
			m.cancelled = append(m.cancelled, job.ID)
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	// Simulate processing
	job.Result = map[string]interface{}{
		"processed": true,
		"steps":     len(job.Commands),
	}

	m.processed = append(m.processed, job)
	return nil
}

func (m *mockProcessor) CancelJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelled = append(m.cancelled, jobID)
	return nil
}

func (m *mockProcessor) setJobError(jobID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[jobID] = err
}

func (m *mockProcessor) setJobDelay(jobID string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delays[jobID] = delay
}

func (m *mockProcessor) getProcessedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.processed)
}

type mockEmitter struct {
	mu      sync.Mutex
	emitted []*BrowserJob
}

func newMockEmitter() *mockEmitter {
	return &mockEmitter{}
}

func (m *mockEmitter) EmitStatus(ctx context.Context, job *BrowserJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emitted = append(m.emitted, job)
	return nil
}

func (m *mockEmitter) getEmittedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.emitted)
}

//=============================================================================
// Test Helpers
//=============================================================================

func createTestJob(id string, commandCount int) *BrowserJob {
	commands := make([]BrowserCommand, commandCount)
	for i := 0; i < commandCount; i++ {
		navCmd := studio.NavigateCommand{
			URL:       fmt.Sprintf("https://example.com/%s/%d", id, i),
			WaitUntil: studio.WaitUntilLoad,
			Timeout:   5000,
		}
		content, _ := json.Marshal(navCmd)
		commands[i] = BrowserCommand{
			Type:    "navigate",
			Content: content,
		}
	}

	return &BrowserJob{
		ID:       id,
		AgentID:  "agent_test",
		RoomID:   "room_test",
		UserID:   "user_test",
		Commands: commands,
		Priority: 5,
	}
}

//=============================================================================
// BrowserJob Tests
//=============================================================================

func TestBrowserJob_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		job      *BrowserJob
		expected bool
	}{
		{
			name:     "no expiry",
			job:      &BrowserJob{ID: "test"},
			expected: false,
		},
		{
			name: "expired",
			job: &BrowserJob{
				ID:        "test",
				ExpiresAt: ptrTime(time.Now().Add(-1 * time.Hour)),
			},
			expected: true,
		},
		{
			name: "not expired",
			job: &BrowserJob{
				ID:        "test",
				ExpiresAt: ptrTime(time.Now().Add(1 * time.Hour)),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.job.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBrowserJob_IsTerminal(t *testing.T) {
	tests := []struct {
		status   JobStatus
		expected bool
	}{
		{JobStatusPending, false},
		{JobStatusRunning, false},
		{JobStatusPaused, false},
		{JobStatusAwaitingPII, false},
		{JobStatusCompleted, true},
		{JobStatusFailed, true},
		{JobStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			job := &BrowserJob{Status: tt.status}
			if got := job.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBrowserJob_CanRetry(t *testing.T) {
	tests := []struct {
		name     string
		job      *BrowserJob
		expected bool
	}{
		{
			name: "failed with retries available",
			job: &BrowserJob{
				Status:     JobStatusFailed,
				Attempts:   1,
				MaxRetries: 3,
			},
			expected: true,
		},
		{
			name: "failed but max retries reached",
			job: &BrowserJob{
				Status:     JobStatusFailed,
				Attempts:   3,
				MaxRetries: 3,
			},
			expected: false,
		},
		{
			name: "not failed",
			job: &BrowserJob{
				Status:     JobStatusPending,
				Attempts:   0,
				MaxRetries: 3,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.job.CanRetry(); got != tt.expected {
				t.Errorf("CanRetry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

//=============================================================================
// BrowserQueue Tests
//=============================================================================

func TestBrowserQueue_Enqueue(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	tests := []struct {
		name    string
		job     *BrowserJob
		wantErr bool
	}{
		{
			name:    "valid job",
			job:     createTestJob("job1", 3),
			wantErr: false,
		},
		{
			name: "missing ID",
			job: &BrowserJob{
				AgentID:  "agent1",
				Commands: []BrowserCommand{{Type: "navigate"}},
			},
			wantErr: true,
		},
		{
			name: "missing agent ID",
			job: &BrowserJob{
				ID:       "job2",
				Commands: []BrowserCommand{{Type: "navigate"}},
			},
			wantErr: true,
		},
		{
			name: "no commands",
			job: &BrowserJob{
				ID:       "job3",
				AgentID:  "agent1",
				Commands: []BrowserCommand{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := queue.Enqueue(tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("Enqueue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrowserQueue_EnqueueDuplicate(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	job := createTestJob("job1", 2)

	// First enqueue should succeed
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}

	// Second enqueue should fail
	if err := queue.Enqueue(job); err == nil {
		t.Error("second Enqueue() should have failed for duplicate job")
	}
}

func TestBrowserQueue_Get(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	job := createTestJob("job1", 2)
	queue.Enqueue(job)

	// Get existing job
	got, err := queue.Get("job1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "job1" {
		t.Errorf("Get() returned wrong job: %s", got.ID)
	}

	// Get non-existent job
	_, err = queue.Get("nonexistent")
	if err == nil {
		t.Error("Get() should have failed for non-existent job")
	}
}

func TestBrowserQueue_Cancel(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	job := createTestJob("job1", 2)
	queue.Enqueue(job)

	// Cancel pending job
	if err := queue.Cancel("job1"); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	got, _ := queue.Get("job1")
	if got.Status != JobStatusCancelled {
		t.Errorf("Cancel() status = %v, want %v", got.Status, JobStatusCancelled)
	}

	// Cancel non-existent job
	if err := queue.Cancel("nonexistent"); err == nil {
		t.Error("Cancel() should have failed for non-existent job")
	}
}

func TestBrowserQueue_Retry(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	// Create a failed job
	job := createTestJob("job1", 2)
	job.Status = JobStatusFailed
	job.Attempts = 1
	job.MaxRetries = 3
	queue.Enqueue(job)

	// Manually set status back to failed after enqueue
	queue.mu.Lock()
	queue.jobs["job1"].Status = JobStatusFailed
	queue.mu.Unlock()

	// Retry the job
	if err := queue.Retry("job1"); err != nil {
		t.Fatalf("Retry() error = %v", err)
	}

	got, _ := queue.Get("job1")
	if got.Status != JobStatusPending {
		t.Errorf("Retry() status = %v, want %v", got.Status, JobStatusPending)
	}
	if got.Attempts != 2 {
		t.Errorf("Retry() attempts = %v, want 2", got.Attempts)
	}
}

func TestBrowserQueue_Processing(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	config := DefaultBrowserQueueConfig()
	config.WorkerCount = 1
	queue := NewBrowserQueue(config, processor, emitter)

	// Start queue
	queue.Start()
	defer queue.Stop()

	// Enqueue job
	job := createTestJob("job1", 2)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check job was processed
	got, _ := queue.Get("job1")
	if got.Status != JobStatusCompleted {
		t.Errorf("job status = %v, want %v", got.Status, JobStatusCompleted)
	}

	if processor.getProcessedCount() != 1 {
		t.Errorf("processed count = %d, want 1", processor.getProcessedCount())
	}
}

func TestBrowserQueue_PriorityJobs(t *testing.T) {
	processor := newMockProcessor()
	processor.setJobDelay("low_priority", 200*time.Millisecond)
	processor.setJobDelay("high_priority", 200*time.Millisecond)

	emitter := newMockEmitter()
	config := DefaultBrowserQueueConfig()
	config.WorkerCount = 1 // Single worker to ensure ordering
	queue := NewBrowserQueue(config, processor, emitter)

	queue.Start()
	defer queue.Stop()

	// Enqueue low priority job first
	lowJob := createTestJob("low_priority", 1)
	lowJob.Priority = 1
	queue.Enqueue(lowJob)

	// Enqueue high priority job
	highJob := createTestJob("high_priority", 1)
	highJob.Priority = 10
	queue.Enqueue(highJob)

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Both should be completed
	stats := queue.Stats()
	if stats.Completed != 2 {
		t.Errorf("completed jobs = %d, want 2", stats.Completed)
	}
}

func TestBrowserQueue_FailedJob(t *testing.T) {
	processor := newMockProcessor()
	processor.setJobError("fail_job", fmt.Errorf("simulated error"))

	emitter := newMockEmitter()
	config := DefaultBrowserQueueConfig()
	config.WorkerCount = 1
	queue := NewBrowserQueue(config, processor, emitter)

	queue.Start()
	defer queue.Stop()

	job := createTestJob("fail_job", 1)
	queue.Enqueue(job)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	got, _ := queue.Get("fail_job")
	if got.Status != JobStatusFailed {
		t.Errorf("job status = %v, want %v", got.Status, JobStatusFailed)
	}
	if got.Error == "" {
		t.Error("job should have error message")
	}
}

func TestBrowserQueue_Stats(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	// Create jobs in different states
	job1 := createTestJob("pending1", 1)
	job2 := createTestJob("pending2", 1)
	job3 := createTestJob("completed1", 1)
	job3.Status = JobStatusCompleted
	job4 := createTestJob("failed1", 1)
	job4.Status = JobStatusFailed

	queue.Enqueue(job1)
	queue.Enqueue(job2)
	queue.Enqueue(job3)
	queue.Enqueue(job4)

	stats := queue.Stats()

	if stats.Total != 4 {
		t.Errorf("Total = %d, want 4", stats.Total)
	}
	if stats.Pending != 2 {
		t.Errorf("Pending = %d, want 2", stats.Pending)
	}
	if stats.Completed != 1 {
		t.Errorf("Completed = %d, want 1", stats.Completed)
	}
	if stats.Failed != 1 {
		t.Errorf("Failed = %d, want 1", stats.Failed)
	}
}

func TestBrowserQueue_List(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	job1 := createTestJob("job1", 1)
	job2 := createTestJob("job2", 1)
	job2.Status = JobStatusCompleted
	job3 := createTestJob("job3", 1)
	job3.Status = JobStatusFailed

	queue.Enqueue(job1)
	queue.Enqueue(job2)
	queue.Enqueue(job3)

	// List all jobs
	all := queue.List()
	if len(all) != 3 {
		t.Errorf("List() returned %d jobs, want 3", len(all))
	}

	// List only completed
	completed := queue.List(JobStatusCompleted)
	if len(completed) != 1 {
		t.Errorf("List(completed) returned %d jobs, want 1", len(completed))
	}

	// List pending and failed
	mixed := queue.List(JobStatusPending, JobStatusFailed)
	if len(mixed) != 2 {
		t.Errorf("List(pending, failed) returned %d jobs, want 2", len(mixed))
	}
}

func TestBrowserQueue_ListByAgent(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)

	job1 := createTestJob("job1", 1)
	job1.AgentID = "agent1"
	job2 := createTestJob("job2", 1)
	job2.AgentID = "agent2"
	job3 := createTestJob("job3", 1)
	job3.AgentID = "agent1"

	queue.Enqueue(job1)
	queue.Enqueue(job2)
	queue.Enqueue(job3)

	agent1Jobs := queue.ListByAgent("agent1")
	if len(agent1Jobs) != 2 {
		t.Errorf("ListByAgent(agent1) returned %d jobs, want 2", len(agent1Jobs))
	}

	agent2Jobs := queue.ListByAgent("agent2")
	if len(agent2Jobs) != 1 {
		t.Errorf("ListByAgent(agent2) returned %d jobs, want 1", len(agent2Jobs))
	}
}

func TestBrowserQueue_Cleanup(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	config := DefaultBrowserQueueConfig()
	config.JobTTL = 1 * time.Hour
	queue := NewBrowserQueue(config, processor, emitter)

	// Create old completed job
	oldJob := createTestJob("old_job", 1)
	oldJob.Status = JobStatusCompleted
	completedTime := time.Now().Add(-2 * time.Hour)
	oldJob.CompletedAt = &completedTime
	queue.Enqueue(oldJob)

	// Create new completed job
	newJob := createTestJob("new_job", 1)
	newJob.Status = JobStatusCompleted
	now := time.Now()
	newJob.CompletedAt = &now
	queue.Enqueue(newJob)

	// Create pending job (should not be cleaned)
	pendingJob := createTestJob("pending_job", 1)
	queue.Enqueue(pendingJob)

	// Manually set the old job's completed time after enqueue
	queue.mu.Lock()
	queue.jobs["old_job"].CompletedAt = &completedTime
	queue.mu.Unlock()

	removed := queue.Cleanup()
	if removed != 1 {
		t.Errorf("Cleanup() removed %d jobs, want 1", removed)
	}

	// Old job should be removed
	if _, err := queue.Get("old_job"); err == nil {
		t.Error("old_job should have been removed")
	}

	// Other jobs should remain
	if _, err := queue.Get("new_job"); err != nil {
		t.Error("new_job should still exist")
	}
	if _, err := queue.Get("pending_job"); err != nil {
		t.Error("pending_job should still exist")
	}
}

//=============================================================================
// Helper Functions
//=============================================================================

func ptrTime(t time.Time) *time.Time {
	return &t
}
