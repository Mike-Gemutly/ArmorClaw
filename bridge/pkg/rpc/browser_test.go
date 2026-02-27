package rpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestBrowserJobManager_CreateJob tests job creation
func TestBrowserJobManager_CreateJob(t *testing.T) {
	mgr := NewBrowserJobManager()

	job := mgr.CreateJob("test-job-1", "agent-001")

	if job.ID != "test-job-1" {
		t.Errorf("expected job ID test-job-1, got %s", job.ID)
	}
	if job.AgentID != "agent-001" {
		t.Errorf("expected agent ID agent-001, got %s", job.AgentID)
	}
	if job.Status != "idle" {
		t.Errorf("expected initial status idle, got %s", job.Status)
	}
	if job.skill == nil {
		t.Error("expected skill to be initialized")
	}
}

// TestBrowserJobManager_GetJob tests job retrieval
func TestBrowserJobManager_GetJob(t *testing.T) {
	mgr := NewBrowserJobManager()
	mgr.CreateJob("test-job-2", "agent-002")

	job, exists := mgr.GetJob("test-job-2")
	if !exists {
		t.Fatal("expected job to exist")
	}
	if job.AgentID != "agent-002" {
		t.Errorf("expected agent ID agent-002, got %s", job.AgentID)
	}

	// Test non-existent job
	_, exists = mgr.GetJob("non-existent")
	if exists {
		t.Error("expected non-existent job to not exist")
	}
}

// TestBrowserJobManager_RemoveJob tests job removal
func TestBrowserJobManager_RemoveJob(t *testing.T) {
	mgr := NewBrowserJobManager()
	mgr.CreateJob("test-job-3", "agent-003")

	mgr.RemoveJob("test-job-3")

	_, exists := mgr.GetJob("test-job-3")
	if exists {
		t.Error("expected job to be removed")
	}
}

// TestBrowserJobManager_ListJobs tests job listing
func TestBrowserJobManager_ListJobs(t *testing.T) {
	mgr := NewBrowserJobManager()
	mgr.CreateJob("job-1", "agent-1")
	mgr.CreateJob("job-2", "agent-2")
	mgr.CreateJob("job-3", "agent-3")

	jobs := mgr.ListJobs()
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

// TestBrowserJobManager_CleanupOldJobs tests old job cleanup
func TestBrowserJobManager_CleanupOldJobs(t *testing.T) {
	mgr := NewBrowserJobManager()

	// Create and complete an old job
	oldJob := mgr.CreateJob("old-job", "agent-old")
	oldJob.Status = "completed"
	oldJob.CreatedAt = time.Now().Add(-2 * time.Hour)

	// Create a new job
	newJob := mgr.CreateJob("new-job", "agent-new")
	newJob.Status = "running"

	// Cleanup jobs older than 1 hour
	removed := mgr.CleanupOldJobs(1 * time.Hour)
	if removed != 1 {
		t.Errorf("expected 1 job removed, got %d", removed)
	}

	// Verify old job is gone
	_, exists := mgr.GetJob("old-job")
	if exists {
		t.Error("expected old job to be removed")
	}

	// Verify new job still exists
	_, exists = mgr.GetJob("new-job")
	if !exists {
		t.Error("expected new job to still exist")
	}
}

// TestHandleBrowserNavigate tests browser.navigate RPC
func TestHandleBrowserNavigate(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	params := map[string]interface{}{
		"url":      "https://example.com",
		"agent_id": "test-agent",
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.navigate",
		Params:  paramsJSON,
	}

	resp := s.handleBrowserNavigate(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if result["status"] != "running" {
		t.Errorf("expected status running, got %v", result["status"])
	}
	if result["url"] != "https://example.com" {
		t.Errorf("expected url, got %v", result["url"])
	}
	if result["job_id"] == "" {
		t.Error("expected job_id to be set")
	}
}

// TestHandleBrowserNavigate_MissingURL tests browser.navigate without URL
func TestHandleBrowserNavigate_MissingURL(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	params := map[string]interface{}{
		"agent_id": "test-agent",
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.navigate",
		Params:  paramsJSON,
	}

	resp := s.handleBrowserNavigate(req)

	if resp.Error == nil {
		t.Fatal("expected error for missing URL")
	}
	if resp.Error.Code != InvalidParams {
		t.Errorf("expected InvalidParams error code, got %d", resp.Error.Code)
	}
}

// TestHandleBrowserStatus tests browser.status RPC
func TestHandleBrowserStatus(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	// First create a job
	createParams := map[string]interface{}{
		"url":    "https://example.com",
		"job_id": "test-status-job",
	}
	createParamsJSON, _ := json.Marshal(createParams)
	createReq := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.navigate",
		Params:  createParamsJSON,
	}
	s.handleBrowserNavigate(createReq)

	// Now check status
	statusParams := map[string]interface{}{
		"job_id": "test-status-job",
	}
	statusParamsJSON, _ := json.Marshal(statusParams)
	statusReq := &Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "browser.status",
		Params:  statusParamsJSON,
	}

	resp := s.handleBrowserStatus(statusReq)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if result["job_id"] != "test-status-job" {
		t.Errorf("expected job_id test-status-job, got %v", result["job_id"])
	}
}

// TestHandleBrowserStatus_NonExistent tests browser.status for non-existent job
func TestHandleBrowserStatus_NonExistent(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	params := map[string]interface{}{
		"job_id": "non-existent-job",
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.status",
		Params:  paramsJSON,
	}

	resp := s.handleBrowserStatus(req)

	if resp.Error == nil {
		t.Fatal("expected error for non-existent job")
	}
}

// TestHandleBrowserFill tests browser.fill RPC
func TestHandleBrowserFill(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	// Create a job first
	createParams := map[string]interface{}{
		"url":    "https://example.com",
		"job_id": "fill-test-job",
	}
	createParamsJSON, _ := json.Marshal(createParams)
	createReq := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.navigate",
		Params:  createParamsJSON,
	}
	s.handleBrowserNavigate(createReq)

	// Fill form
	fillParams := map[string]interface{}{
		"job_id":   "fill-test-job",
		"selector": "#email",
		"value":    "test@example.com",
	}
	fillParamsJSON, _ := json.Marshal(fillParams)
	fillReq := &Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "browser.fill",
		Params:  fillParamsJSON,
	}

	resp := s.handleBrowserFill(fillReq)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if result["status"] != "filled" {
		t.Errorf("expected status filled, got %v", result["status"])
	}
}

// TestHandleBrowserClick tests browser.click RPC
func TestHandleBrowserClick(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	// Create a job first
	createParams := map[string]interface{}{
		"url":    "https://example.com",
		"job_id": "click-test-job",
	}
	createParamsJSON, _ := json.Marshal(createParams)
	createReq := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.navigate",
		Params:  createParamsJSON,
	}
	s.handleBrowserNavigate(createReq)

	// Click element
	clickParams := map[string]interface{}{
		"job_id":   "click-test-job",
		"selector": "#submit-button",
	}
	clickParamsJSON, _ := json.Marshal(clickParams)
	clickReq := &Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "browser.click",
		Params:  clickParamsJSON,
	}

	resp := s.handleBrowserClick(clickReq)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if result["status"] != "clicked" {
		t.Errorf("expected status clicked, got %v", result["status"])
	}
}

// TestHandleBrowserList tests browser.list RPC
func TestHandleBrowserList(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		createParams := map[string]interface{}{
			"url":    "https://example.com",
			"job_id": "list-test-job-" + string(rune('1'+i)),
		}
		createParamsJSON, _ := json.Marshal(createParams)
		createReq := &Request{
			JSONRPC: "2.0",
			ID:      i + 1,
			Method:  "browser.navigate",
			Params:  createParamsJSON,
		}
		s.handleBrowserNavigate(createReq)
	}

	// List jobs
	listReq := &Request{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "browser.list",
		Params:  json.RawMessage("{}"),
	}

	resp := s.handleBrowserList(listReq)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	count, ok := result["count"].(int)
	if !ok {
		t.Fatal("expected count to be an int")
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

// TestHandleBrowserCancel tests browser.cancel RPC
func TestHandleBrowserCancel(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	// Create a job
	createParams := map[string]interface{}{
		"url":    "https://example.com",
		"job_id": "cancel-test-job",
	}
	createParamsJSON, _ := json.Marshal(createParams)
	createReq := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.navigate",
		Params:  createParamsJSON,
	}
	s.handleBrowserNavigate(createReq)

	// Cancel job
	cancelParams := map[string]interface{}{
		"job_id": "cancel-test-job",
	}
	cancelParamsJSON, _ := json.Marshal(cancelParams)
	cancelReq := &Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "browser.cancel",
		Params:  cancelParamsJSON,
	}

	resp := s.handleBrowserCancel(cancelReq)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if result["status"] != "cancelled" {
		t.Errorf("expected status cancelled, got %v", result["status"])
	}
}

// TestHandleBrowserComplete tests browser.complete RPC
func TestHandleBrowserComplete(t *testing.T) {
	s := &Server{
		ctx:         context.Background(),
		browserJobs: NewBrowserJobManager(),
	}

	// Create a job
	createParams := map[string]interface{}{
		"url":    "https://example.com",
		"job_id": "complete-test-job",
	}
	createParamsJSON, _ := json.Marshal(createParams)
	createReq := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "browser.navigate",
		Params:  createParamsJSON,
	}
	s.handleBrowserNavigate(createReq)

	// Complete job
	completeParams := map[string]interface{}{
		"job_id": "complete-test-job",
	}
	completeParamsJSON, _ := json.Marshal(completeParams)
	completeReq := &Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "browser.complete",
		Params:  completeParamsJSON,
	}

	resp := s.handleBrowserComplete(completeReq)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if result["status"] != "completed" {
		t.Errorf("expected status completed, got %v", result["status"])
	}
}
