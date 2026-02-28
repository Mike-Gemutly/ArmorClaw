package queue

import (
	"encoding/json"
	"testing"

	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// RPC Handler Tests
//=============================================================================

func TestQueueRPCHandler_Enqueue(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	// Create valid command
	navCmd := studio.NavigateCommand{
		URL:       "https://example.com",
		WaitUntil: studio.WaitUntilLoad,
	}
	content, _ := json.Marshal(navCmd)

	params := map[string]interface{}{
		"id":       "test_job_1",
		"agent_id": "agent1",
		"room_id":  "room1",
		"commands": []map[string]interface{}{
			{"type": "navigate", "content": json.RawMessage(content)},
		},
		"priority": 5,
	}
	paramsJSON, _ := json.Marshal(params)

	req := &RPCRequest{
		Method: "browser.enqueue",
		Params: paramsJSON,
		UserID: "user1",
	}

	resp := handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("Enqueue failed: %s", resp.Error.Message)
	}

	result := resp.Result.(map[string]interface{})
	status, ok := result["status"].(JobStatus)
	if !ok {
		t.Fatalf("status is not a JobStatus: %T", result["status"])
	}
	if status != JobStatusPending {
		t.Errorf("status = %v, want %v", status, JobStatusPending)
	}
}

func TestQueueRPCHandler_EnqueueMissingAgentID(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	params := map[string]interface{}{
		"room_id": "room1",
	}
	paramsJSON, _ := json.Marshal(params)

	req := &RPCRequest{
		Method: "browser.enqueue",
		Params: paramsJSON,
		UserID: "user1",
	}

	resp := handler.Handle(req)
	if resp.Error == nil {
		t.Fatal("Expected error for missing agent_id")
	}
	if resp.Error.Code != RPCErrInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, RPCErrInvalidParams)
	}
}

func TestQueueRPCHandler_GetJob(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	// First enqueue a job
	job := createTestJob("job1", 2)
	queue.Enqueue(job)

	// Get the job
	params := map[string]interface{}{"id": "job1"}
	paramsJSON, _ := json.Marshal(params)

	req := &RPCRequest{
		Method: "browser.get_job",
		Params: paramsJSON,
	}

	resp := handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("GetJob failed: %s", resp.Error.Message)
	}

	result := resp.Result.(map[string]interface{})
	if result["id"] != "job1" {
		t.Errorf("job id = %v, want job1", result["id"])
	}
}

func TestQueueRPCHandler_GetJobNotFound(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	params := map[string]interface{}{"id": "nonexistent"}
	paramsJSON, _ := json.Marshal(params)

	req := &RPCRequest{
		Method: "browser.get_job",
		Params: paramsJSON,
	}

	resp := handler.Handle(req)
	if resp.Error == nil {
		t.Fatal("Expected error for non-existent job")
	}
}

func TestQueueRPCHandler_ListJobs(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	// Enqueue jobs
	queue.Enqueue(createTestJob("job1", 1))
	queue.Enqueue(createTestJob("job2", 1))

	params := map[string]interface{}{}
	paramsJSON, _ := json.Marshal(params)

	req := &RPCRequest{
		Method: "browser.list_jobs",
		Params: paramsJSON,
	}

	resp := handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("ListJobs failed: %s", resp.Error.Message)
	}

	result := resp.Result.(map[string]interface{})
	count := result["count"].(int)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestQueueRPCHandler_CancelJob(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	// Enqueue job
	queue.Enqueue(createTestJob("job1", 1))

	// Cancel job
	params := map[string]interface{}{"id": "job1"}
	paramsJSON, _ := json.Marshal(params)

	req := &RPCRequest{
		Method: "browser.cancel_job",
		Params: paramsJSON,
		UserID: "user1",
	}

	resp := handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("CancelJob failed: %s", resp.Error.Message)
	}

	// Verify job is cancelled
	job, _ := queue.Get("job1")
	if job.Status != JobStatusCancelled {
		t.Errorf("job status = %v, want %v", job.Status, JobStatusCancelled)
	}
}

func TestQueueRPCHandler_RetryJob(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	// Create and enqueue a job
	job := createTestJob("job1", 1)
	job.MaxRetries = 3
	queue.Enqueue(job)

	// Manually set status to failed after enqueue
	queue.mu.Lock()
	queue.jobs["job1"].Status = JobStatusFailed
	queue.jobs["job1"].Attempts = 1
	queue.mu.Unlock()

	// Retry job
	params := map[string]interface{}{"id": "job1"}
	paramsJSON, _ := json.Marshal(params)

	req := &RPCRequest{
		Method: "browser.retry_job",
		Params: paramsJSON,
		UserID: "user1",
	}

	resp := handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("RetryJob failed: %s", resp.Error.Message)
	}

	// Verify job is pending
	got, _ := queue.Get("job1")
	if got.Status != JobStatusPending {
		t.Errorf("job status = %v, want %v", got.Status, JobStatusPending)
	}
}

func TestQueueRPCHandler_QueueStats(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	// Enqueue jobs in different states
	job1 := createTestJob("job1", 1)
	job2 := createTestJob("job2", 1)
	job3 := createTestJob("job3", 1)

	queue.Enqueue(job1)
	queue.Enqueue(job2)
	queue.Enqueue(job3)

	// Manually update statuses after enqueue
	queue.mu.Lock()
	queue.jobs["job2"].Status = JobStatusCompleted
	queue.jobs["job3"].Status = JobStatusFailed
	queue.mu.Unlock()

	req := &RPCRequest{
		Method: "browser.queue_stats",
	}

	resp := handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("QueueStats failed: %s", resp.Error.Message)
	}

	stats := resp.Result.(*QueueStats)
	if stats.Total != 3 {
		t.Errorf("Total = %d, want 3", stats.Total)
	}
	if stats.Pending != 1 {
		t.Errorf("Pending = %d, want 1", stats.Pending)
	}
	if stats.Completed != 1 {
		t.Errorf("Completed = %d, want 1", stats.Completed)
	}
	if stats.Failed != 1 {
		t.Errorf("Failed = %d, want 1", stats.Failed)
	}
}

func TestQueueRPCHandler_QueueStartStop(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	// Start queue
	req := &RPCRequest{
		Method: "browser.queue_start",
	}
	resp := handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("QueueStart failed: %s", resp.Error.Message)
	}

	result := resp.Result.(map[string]interface{})
	if result["status"] != "started" {
		t.Errorf("status = %v, want started", result["status"])
	}

	// Stop queue
	req = &RPCRequest{
		Method: "browser.queue_stop",
	}
	resp = handler.Handle(req)
	if resp.Error != nil {
		t.Fatalf("QueueStop failed: %s", resp.Error.Message)
	}

	result = resp.Result.(map[string]interface{})
	if result["status"] != "stopped" {
		t.Errorf("status = %v, want stopped", result["status"])
	}
}

func TestQueueRPCHandler_UnknownMethod(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	handler := NewQueueRPCHandler(queue, nil)

	req := &RPCRequest{
		Method: "browser.unknown",
	}

	resp := handler.Handle(req)
	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}
	if resp.Error.Code != RPCErrNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, RPCErrNotFound)
	}
}

//=============================================================================
// Integration Tests
//=============================================================================

func TestIntegration_EnqueueFromAgent(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	integration := NewIntegration(queue, nil)

	navCmd := studio.NavigateCommand{
		URL:       "https://example.com",
		WaitUntil: studio.WaitUntilLoad,
	}
	content, _ := json.Marshal(navCmd)
	commands := []BrowserCommand{
		{Type: "navigate", Content: content},
	}

	job, err := integration.EnqueueFromAgent(nil, "agent1", "def1", "room1", commands)
	if err != nil {
		t.Fatalf("EnqueueFromAgent failed: %v", err)
	}

	if job.AgentID != "agent1" {
		t.Errorf("agent_id = %s, want agent1", job.AgentID)
	}
	if job.DefinitionID != "def1" {
		t.Errorf("definition_id = %s, want def1", job.DefinitionID)
	}
	if job.RoomID != "room1" {
		t.Errorf("room_id = %s, want room1", job.RoomID)
	}

	// Verify job is in queue
	got, err := queue.Get(job.ID)
	if err != nil {
		t.Fatalf("job not found in queue: %v", err)
	}
	if got.ID != job.ID {
		t.Errorf("retrieved job ID = %s, want %s", got.ID, job.ID)
	}
}

func TestIntegration_GetQueueAndGetHandler(t *testing.T) {
	processor := newMockProcessor()
	emitter := newMockEmitter()
	queue := NewBrowserQueue(DefaultBrowserQueueConfig(), processor, emitter)
	integration := NewIntegration(queue, nil)

	if integration.GetQueue() != queue {
		t.Error("GetQueue() returned wrong queue")
	}

	if integration.GetHandler() == nil {
		t.Error("GetHandler() returned nil")
	}
}
