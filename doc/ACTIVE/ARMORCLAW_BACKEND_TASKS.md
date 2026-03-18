# ArmorClaw Bridge Server — Mobile Secretary Implementation Tasks

> **Target:** ArmorClaw Bridge Server (Go backend)  
> **Purpose:** Step-by-step backend changes needed for Mobile Secretary  
> **Date:** 2026-02-27

---

## 📋 Overview

This document outlines **all backend work** required on the ArmorClaw Bridge Server to support the Mobile Secretary system. Each phase corresponds to the Mobile Secretary implementation plan, but focuses only on server-side changes.

**Repository Location:** ❓ **Unknown** — Must be located first

---

## 🔍 Phase 0: Backend Discovery & Setup

### Step 0.1: Locate ArmorClaw Repository
**Task:** Find the Go backend codebase  
**Acceptance Criteria:**
- Repository cloned locally
- Can build and run server
- Can run existing tests
- Have access to commit/deploy

**Commands to verify:**
```bash
# Check project structure
ls -la
cat go.mod  # Verify module name
go build ./cmd/bridge  # Ensure it compiles

# Run tests
go test ./...

# Check existing RPC methods
grep -r "rpc\." pkg/ internal/
```

---

### Step 0.2: Document Existing RPC Methods
**Task:** Audit current RPC implementation  
**Expected Files:**
```
pkg/rpc/
├── server.go           # JSON-RPC 2.0 server
├── methods.go          # Method registration
├── bridge_methods.go   # bridge.* methods
├── push_methods.go     # push.* methods
└── health_methods.go   # health check
```

**Create Inventory:**
```markdown
| Method | Status | Purpose |
|--------|--------|---------|
| bridge.status | ✅ Exists | Server health |
| bridge.health | ✅ Exists | Detailed health |
| push.register_token | ✅ Exists | FCM registration |
| push.unregister_token | ✅ Exists | FCM cleanup |
| agent.* | ❓ Unknown | Agent operations |
| hitl.* | ❓ Unknown | HITL approvals |
| workflow.* | ❓ Unknown | Workflow engine |
| keystore.* | ❓ Unknown | PII vault |
| browser.* | ❓ Unknown | Browser automation |
| pii.* | ❓ Unknown | PII access control |
```

**Deliverable:** `EXISTING_RPC_METHODS.md` in Bridge repo

---

### Step 0.3: Set Up Development Environment
**Task:** Prepare for Mobile Secretary development  
**Requirements:**
- Go 1.21+ installed
- Docker Desktop running (for Playwright container)
- PostgreSQL/SQLite running (for keystore)
- Matrix homeserver access (test server)

**Environment Variables:**
```bash
export BRIDGE_PORT=8080
export MATRIX_HOMESERVER_URL=https://matrix.armorclaw.local
export RPC_SECRET_KEY=<your-secret>
export DATABASE_URL=postgres://localhost/armorclaw_dev
export BROWSER_SERVICE_URL=http://localhost:3000  # Future
```

---

## 🌐 Phase 1: Browser Automation Backend (Days 2-5)

### Step 1.1: Create Browser Skill Queue (Day 2)
**Task:** Add job queue for browser operations  
**Files to CREATE:**
```
pkg/queue/
├── browser_queue.go        # Job queue implementation
├── browser_queue_test.go   # Unit tests
└── job.go                  # Job struct definition
```

**Implementation:**
```go
// pkg/queue/job.go
package queue

import "time"

type JobType string

const (
    JobTypeNavigate    JobType = "navigate"
    JobTypeFill        JobType = "fill"
    JobTypeClick       JobType = "click"
    JobTypeScreenshot  JobType = "screenshot"
    JobTypeWait        JobType = "wait"
)

type BrowserJob struct {
    ID          string                 `json:"id"`
    Type        JobType                `json:"type"`
    AgentID     string                 `json:"agent_id"`
    Payload     map[string]interface{} `json:"payload"`
    Status      string                 `json:"status"` // pending, running, success, failed
    Result      interface{}            `json:"result,omitempty"`
    Error       string                 `json:"error,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    RetryCount  int                    `json:"retry_count"`
    MaxRetries  int                    `json:"max_retries"`
}
```

**Job Queue Manager:**
```go
// pkg/queue/browser_queue.go
package queue

import (
    "context"
    "sync"
)

type BrowserQueue struct {
    jobs     map[string]*BrowserJob
    mu       sync.RWMutex
    workers  int
    jobChan  chan *BrowserJob
}

func NewBrowserQueue(workers int) *BrowserQueue {
    return &BrowserQueue{
        jobs:    make(map[string]*BrowserJob),
        workers: workers,
        jobChan: make(chan *BrowserJob, 100),
    }
}

func (q *BrowserQueue) Enqueue(job *BrowserJob) error {
    q.mu.Lock()
    q.jobs[job.ID] = job
    q.mu.Unlock()
    
    q.jobChan <- job
    return nil
}

func (q *BrowserQueue) GetJob(id string) (*BrowserJob, bool) {
    q.mu.RLock()
    defer q.mu.RUnlock()
    job, exists := q.jobs[id]
    return job, exists
}
```

**Acceptance Criteria:**
- ✅ Can enqueue browser jobs
- ✅ Can query job status
- ✅ Handles concurrent access safely
- ✅ Unit tests pass

---

### Step 1.2: Browser Skill HTTP Client (Day 3)
**Task:** Add HTTP client to communicate with browser service  
**Files to CREATE:**
```
pkg/skills/
├── browser.go           # HTTP client for browser service
├── browser_test.go      # Unit tests
└── types.go             # Request/response types
```

**Implementation:**
```go
// pkg/skills/browser.go
package skills

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type BrowserClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewBrowserClient(baseURL string) *BrowserClient {
    return &BrowserClient{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

// Navigate to a URL
func (c *BrowserClient) Navigate(ctx context.Context, url string) (*NavigateResponse, error) {
    req := NavigateRequest{URL: url}
    var resp NavigateResponse
    
    if err := c.post(ctx, "/navigate", req, &resp); err != nil {
        return nil, err
    }
    
    return &resp, nil
}

// Fill a form field
func (c *BrowserClient) Fill(ctx context.Context, selector, value string) error {
    req := FillRequest{Selector: selector, Value: value}
    var resp SuccessResponse
    
    return c.post(ctx, "/fill", req, &resp)
}

// Click an element
func (c *BrowserClient) Click(ctx context.Context, selector string) error {
    req := ClickRequest{Selector: selector}
    var resp SuccessResponse
    
    return c.post(ctx, "/click", req, &resp)
}

// Take screenshot
func (c *BrowserClient) Screenshot(ctx context.Context) ([]byte, error) {
    var resp ScreenshotResponse
    
    if err := c.post(ctx, "/screenshot", nil, &resp); err != nil {
        return nil, err
    }
    
    // Decode base64 image
    return base64.StdEncoding.DecodeString(resp.Image)
}

// Get browser status
func (c *BrowserClient) Status(ctx context.Context) (*StatusResponse, error) {
    resp, err := c.httpClient.Get(c.baseURL + "/status")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var status StatusResponse
    if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
        return nil, err
    }
    
    return &status, nil
}

func (c *BrowserClient) post(ctx context.Context, path string, req, resp interface{}) error {
    body, err := json.Marshal(req)
    if err != nil {
        return err
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    httpResp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return err
    }
    defer httpResp.Body.Close()
    
    if httpResp.StatusCode != http.StatusOK {
        return fmt.Errorf("browser service error: %d", httpResp.StatusCode)
    }
    
    return json.NewDecoder(httpResp.Body).Decode(resp)
}
```

**Types:**
```go
// pkg/skills/types.go
package skills

type NavigateRequest struct {
    URL string `json:"url"`
}

type NavigateResponse struct {
    Success bool   `json:"success"`
    Title   string `json:"title"`
    URL     string `json:"url"`
}

type FillRequest struct {
    Selector string `json:"selector"`
    Value    string `json:"value"`
}

type ClickRequest struct {
    Selector string `json:"selector"`
}

type SuccessResponse struct {
    Success bool `json:"success"`
}

type ScreenshotResponse struct {
    Image string `json:"image"` // base64-encoded PNG
}

type StatusResponse struct {
    Status string `json:"status"`
    URL    string `json:"url,omitempty"`
    Title  string `json:"title,omitempty"`
}
```

**Acceptance Criteria:**
- ✅ Can call browser service endpoints
- ✅ Handles timeouts gracefully
- ✅ Proper error handling
- ✅ Unit tests with mocked HTTP

---

### Step 1.3: Browser RPC Methods (Day 4)
**Task:** Expose browser operations via JSON-RPC  
**Files to CREATE:**
```
pkg/rpc/
├── browser_methods.go       # browser.* RPC methods
└── browser_methods_test.go  # RPC tests
```

**Implementation:**
```go
// pkg/rpc/browser_methods.go
package rpc

import (
    "context"
    "github.com/armorclaw/bridge/pkg/queue"
    "github.com/armorclaw/bridge/pkg/skills"
)

type BrowserMethods struct {
    browserClient *skills.BrowserClient
    queue         *queue.BrowserQueue
}

func NewBrowserMethods(client *skills.BrowserClient, q *queue.BrowserQueue) *BrowserMethods {
    return &BrowserMethods{
        browserClient: client,
        queue:         q,
    }
}

// browser.navigate
type BrowserNavigateRequest struct {
    URL string `json:"url"`
}

type BrowserNavigateResponse struct {
    Success bool   `json:"success"`
    Title   string `json:"title"`
    URL     string `json:"url"`
}

func (m *BrowserMethods) Navigate(ctx context.Context, req *BrowserNavigateRequest) (*BrowserNavigateResponse, error) {
    resp, err := m.browserClient.Navigate(ctx, req.URL)
    if err != nil {
        return nil, err
    }
    
    return &BrowserNavigateResponse{
        Success: resp.Success,
        Title:   resp.Title,
        URL:     resp.URL,
    }, nil
}

// browser.fill
type BrowserFillRequest struct {
    Selector string `json:"selector"`
    Value    string `json:"value"`
}

type BrowserFillResponse struct {
    Success bool `json:"success"`
}

func (m *BrowserMethods) Fill(ctx context.Context, req *BrowserFillRequest) (*BrowserFillResponse, error) {
    if err := m.browserClient.Fill(ctx, req.Selector, req.Value); err != nil {
        return nil, err
    }
    
    return &BrowserFillResponse{Success: true}, nil
}

// browser.click
type BrowserClickRequest struct {
    Selector string `json:"selector"`
}

type BrowserClickResponse struct {
    Success bool `json:"success"`
}

func (m *BrowserMethods) Click(ctx context.Context, req *BrowserClickRequest) (*BrowserClickResponse, error) {
    if err := m.browserClient.Click(ctx, req.Selector); err != nil {
        return nil, err
    }
    
    return &BrowserClickResponse{Success: true}, nil
}

// browser.screenshot
type BrowserScreenshotResponse struct {
    Image string `json:"image"` // base64-encoded PNG
}

func (m *BrowserMethods) Screenshot(ctx context.Context) (*BrowserScreenshotResponse, error) {
    imageBytes, err := m.browserClient.Screenshot(ctx)
    if err != nil {
        return nil, err
    }
    
    return &BrowserScreenshotResponse{
        Image: base64.StdEncoding.EncodeToString(imageBytes),
    }, nil
}

// browser.status
type BrowserStatusResponse struct {
    Status string `json:"status"`
    URL    string `json:"url,omitempty"`
    Title  string `json:"title,omitempty"`
}

func (m *BrowserMethods) Status(ctx context.Context) (*BrowserStatusResponse, error) {
    status, err := m.browserClient.Status(ctx)
    if err != nil {
        return nil, err
    }
    
    return &BrowserStatusResponse{
        Status: status.Status,
        URL:    status.URL,
        Title:  status.Title,
    }, nil
}
```

**Register Methods:**
```go
// pkg/rpc/server.go (modify)
func (s *Server) RegisterMethods() {
    // Existing methods
    s.Register("bridge.status", s.bridgeMethods.Status)
    s.Register("bridge.health", s.bridgeMethods.Health)
    
    // NEW: Browser methods
    s.Register("browser.navigate", s.browserMethods.Navigate)
    s.Register("browser.fill", s.browserMethods.Fill)
    s.Register("browser.click", s.browserMethods.Click)
    s.Register("browser.screenshot", s.browserMethods.Screenshot)
    s.Register("browser.status", s.browserMethods.Status)
}
```

**Acceptance Criteria:**
- ✅ RPC methods callable from ArmorChat
- ✅ Proper error responses
- ✅ JSON-RPC 2.0 compliant
- ✅ Integration tests pass

---

### Step 1.4: Integration Testing (Day 5)
**Task:** End-to-end test with browser service  
**Test Scenarios:**
1. Navigate to httpbin.org/forms/post
2. Fill form fields
3. Click submit button
4. Verify success
5. Take screenshot

**Test File:**
```go
// pkg/rpc/browser_methods_integration_test.go
// +build integration

package rpc_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestBrowserNavigateIntegration(t *testing.T) {
    client := setupRPCClient(t)
    
    req := BrowserNavigateRequest{URL: "https://httpbin.org/forms/post"}
    var resp BrowserNavigateResponse
    
    err := client.Call(context.Background(), "browser.navigate", req, &resp)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
    assert.Contains(t, resp.Title, "httpbin")
}
```

---

## 🔄 Phase 2: Agent State Machine (Days 6-7)

### Step 2.1: Agent Status Domain Model (Day 6)
**Task:** Create agent status enum and state machine  
**Files to CREATE:**
```
pkg/agent/
├── status.go              # AgentStatus enum
├── state_machine.go       # State transitions
├── state_machine_test.go  # Unit tests
└── events.go              # Matrix event emission
```

**Implementation:**
```go
// pkg/agent/status.go
package agent

type Status string

const (
    StatusIdle              Status = "IDLE"
    StatusBrowsing          Status = "BROWSING"
    StatusFormFilling       Status = "FORM_FILLING"
    StatusProcessingPayment Status = "PROCESSING_PAYMENT"
    StatusAwaitingCaptcha   Status = "AWAITING_CAPTCHA"
    StatusAwaiting2FA       Status = "AWAITING_2FA"
    StatusAwaitingApproval  Status = "AWAITING_APPROVAL"
    StatusError             Status = "ERROR"
    StatusComplete          Status = "COMPLETE"
)

type StatusEvent struct {
    AgentID   string                 `json:"agent_id"`
    Status    Status                 `json:"status"`
    Metadata  map[string]interface{} `json:"metadata"`
    Timestamp int64                  `json:"timestamp"`
}
```

**State Machine:**
```go
// pkg/agent/state_machine.go
package agent

import (
    "fmt"
    "sync"
    "time"
)

type StateMachine struct {
    agentID       string
    currentStatus Status
    metadata      map[string]interface{}
    mu            sync.RWMutex
    eventEmitter  EventEmitter
}

type EventEmitter interface {
    EmitAgentStatus(event *StatusEvent) error
}

func NewStateMachine(agentID string, emitter EventEmitter) *StateMachine {
    return &StateMachine{
        agentID:       agentID,
        currentStatus: StatusIdle,
        metadata:      make(map[string]interface{}),
        eventEmitter:  emitter,
    }
}

func (sm *StateMachine) TransitionTo(newStatus Status, metadata map[string]interface{}) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    // Validate transition
    if !sm.isValidTransition(sm.currentStatus, newStatus) {
        return fmt.Errorf("invalid transition from %s to %s", sm.currentStatus, newStatus)
    }
    
    sm.currentStatus = newStatus
    sm.metadata = metadata
    
    // Emit Matrix event
    event := &StatusEvent{
        AgentID:   sm.agentID,
        Status:    newStatus,
        Metadata:  metadata,
        Timestamp: time.Now().Unix(),
    }
    
    return sm.eventEmitter.EmitAgentStatus(event)
}

func (sm *StateMachine) isValidTransition(from, to Status) bool {
    validTransitions := map[Status][]Status{
        StatusIdle: {StatusBrowsing, StatusError},
        StatusBrowsing: {StatusFormFilling, StatusAwaitingCaptcha, StatusError, StatusComplete},
        StatusFormFilling: {StatusProcessingPayment, StatusAwaitingApproval, StatusError, StatusComplete},
        StatusProcessingPayment: {StatusAwaiting2FA, StatusAwaitingApproval, StatusComplete, StatusError},
        StatusAwaitingCaptcha: {StatusBrowsing, StatusError},
        StatusAwaiting2FA: {StatusProcessingPayment, StatusError},
        StatusAwaitingApproval: {StatusFormFilling, StatusError, StatusIdle},
        StatusError: {StatusIdle},
        StatusComplete: {StatusIdle},
    }
    
    allowed, exists := validTransitions[from]
    if !exists {
        return false
    }
    
    for _, s := range allowed {
        if s == to {
            return true
        }
    }
    
    return false
}

func (sm *StateMachine) GetStatus() (Status, map[string]interface{}) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    // Deep copy metadata
    metaCopy := make(map[string]interface{})
    for k, v := range sm.metadata {
        metaCopy[k] = v
    }
    
    return sm.currentStatus, metaCopy
}
```

**Acceptance Criteria:**
- ✅ State transitions validated
- ✅ Illegal transitions rejected
- ✅ Thread-safe
- ✅ Unit tests for all transitions

---

### Step 2.2: Matrix Event Emitter (Day 7)
**Task:** Emit agent status events to Matrix rooms  
**Files to CREATE:**
```
pkg/matrix/
├── event_emitter.go       # Matrix event emission
└── event_emitter_test.go  # Tests
```

**Implementation:**
```go
// pkg/matrix/event_emitter.go
package matrix

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/armorclaw/bridge/pkg/agent"
)

type EventEmitter struct {
    homeserverURL string
    accessToken   string
    httpClient    *http.Client
}

func NewEventEmitter(homeserverURL, accessToken string) *EventEmitter {
    return &EventEmitter{
        homeserverURL: homeserverURL,
        accessToken:   accessToken,
        httpClient:    &http.Client{},
    }
}

func (e *EventEmitter) EmitAgentStatus(event *agent.StatusEvent) error {
    // Construct Matrix event
    matrixEvent := map[string]interface{}{
        "type": "com.armorclaw.agent.status",
        "content": map[string]interface{}{
            "agent_id": event.AgentID,
            "status":   event.Status,
            "metadata": event.Metadata,
        },
    }
    
    // Send to control room (hardcoded for now, should be configurable)
    roomID := "!controlplane:armorclaw.local"
    
    return e.sendEvent(roomID, matrixEvent)
}

func (e *EventEmitter) sendEvent(roomID string, event map[string]interface{}) error {
    url := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/com.armorclaw.agent.status/%d",
        e.homeserverURL, roomID, time.Now().UnixNano())
    
    body, err := json.Marshal(event["content"])
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", "Bearer "+e.accessToken)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := e.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed to send event: %d", resp.StatusCode)
    }
    
    return nil
}
```

**Acceptance Criteria:**
- ✅ Events sent to Matrix homeserver
- ✅ Proper authentication
- ✅ Error handling
- ✅ Integration test with test homeserver

---

### Step 2.3: Agent RPC Methods (Day 7)
**Task:** Add agent management RPC methods  
**Files to CREATE:**
```
pkg/rpc/
├── agent_methods.go       # agent.* RPC methods
└── agent_methods_test.go  # Tests
```

**Methods to Implement:**
```go
// pkg/rpc/agent_methods.go
package rpc

// agent.list - List all agents
func (m *AgentMethods) List(ctx context.Context) (*AgentListResponse, error)

// agent.status - Get agent status
func (m *AgentMethods) Status(ctx context.Context, req *AgentStatusRequest) (*AgentStatusResponse, error)

// agent.stop - Stop an agent
func (m *AgentMethods) Stop(ctx context.Context, req *AgentStopRequest) (*AgentStopResponse, error)
```

**Example Implementation:**
```go
type AgentStatusRequest struct {
    AgentID string `json:"agent_id"`
}

type AgentStatusResponse struct {
    AgentID  string                 `json:"agent_id"`
    Status   string                 `json:"status"`
    Metadata map[string]interface{} `json:"metadata"`
}

func (m *AgentMethods) Status(ctx context.Context, req *AgentStatusRequest) (*AgentStatusResponse, error) {
    sm, exists := m.agents[req.AgentID]
    if !exists {
        return nil, fmt.Errorf("agent not found: %s", req.AgentID)
    }
    
    status, metadata := sm.GetStatus()
    
    return &AgentStatusResponse{
        AgentID:  req.AgentID,
        Status:   string(status),
        Metadata: metadata,
    }, nil
}
```

---

## 🔐 Phase 3: PII Access Control (Days 8-10)

### Step 3.1: PII Request Model (Day 8)
**Task:** Create PII field sensitivity model  
**Files to CREATE:**
```
pkg/keystore/
├── pii.go              # PII types and sensitivity
├── pii_request.go      # Request generation
└── pii_request_test.go # Tests
```

**Implementation:**
```go
// pkg/keystore/pii.go
package keystore

type FieldSensitivity string

const (
    SensitivityLow      FieldSensitivity = "LOW"      // Name, email
    SensitivityMedium   FieldSensitivity = "MEDIUM"   // Phone, address
    SensitivityHigh     FieldSensitivity = "HIGH"     // SSN, passport
    SensitivityCritical FieldSensitivity = "CRITICAL" // Credit card, bank account
)

type PiiField struct {
    Name        string           `json:"name"`
    Sensitivity FieldSensitivity `json:"sensitivity"`
    Description string           `json:"description"`
    MaskedValue string           `json:"masked_value,omitempty"`
}

type PiiAccessRequest struct {
    RequestID string      `json:"request_id"`
    AgentID   string      `json:"agent_id"`
    Fields    []PiiField  `json:"fields"`
    Reason    string      `json:"reason"`
    ExpiresAt int64       `json:"expires_at"`
    Status    string      `json:"status"` // pending, approved, denied
}
```

**Request Generator:**
```go
// pkg/keystore/pii_request.go
package keystore

import (
    "fmt"
    "time"
    "github.com/google/uuid"
)

type PiiRequestBuilder struct {
    request *PiiAccessRequest
}

func NewPiiRequest(agentID, reason string) *PiiRequestBuilder {
    return &PiiRequestBuilder{
        request: &PiiAccessRequest{
            RequestID: uuid.New().String(),
            AgentID:   agentID,
            Reason:    reason,
            ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
            Status:    "pending",
            Fields:    []PiiField{},
        },
    }
}

func (b *PiiRequestBuilder) AddField(name string, sensitivity FieldSensitivity, description string) *PiiRequestBuilder {
    b.request.Fields = append(b.request.Fields, PiiField{
        Name:        name,
        Sensitivity: sensitivity,
        Description: description,
    })
    return b
}

func (b *PiiRequestBuilder) WithExpiry(duration time.Duration) *PiiRequestBuilder {
    b.request.ExpiresAt = time.Now().Add(duration).Unix()
    return b
}

func (b *PiiRequestBuilder) Build() *PiiAccessRequest {
    return b.request
}
```

---

### Step 3.2: PII RPC Methods (Day 9)
**Task:** Add PII access control RPC methods  
**Files to CREATE:**
```
pkg/rpc/
├── pii_methods.go       # pii.* RPC methods
└── pii_methods_test.go  # Tests
```

**Methods:**
```go
// pii.request - Request PII access
type PiiRequestRequest struct {
    AgentID string                       `json:"agent_id"`
    Fields  []string                     `json:"fields"` // field names
    Reason  string                       `json:"reason"`
}

type PiiRequestResponse struct {
    RequestID string `json:"request_id"`
    ExpiresAt int64  `json:"expires_at"`
}

func (m *PiiMethods) Request(ctx context.Context, req *PiiRequestRequest) (*PiiRequestResponse, error)

// pii.approve - Approve PII request
type PiiApproveRequest struct {
    RequestID      string   `json:"request_id"`
    ApprovedFields []string `json:"approved_fields"`
}

func (m *PiiMethods) Approve(ctx context.Context, req *PiiApproveRequest) (*SuccessResponse, error)

// pii.deny - Deny PII request
type PiiDenyRequest struct {
    RequestID string  `json:"request_id"`
    Reason    *string `json:"reason,omitempty"`
}

func (m *PiiMethods) Deny(ctx context.Context, req *PiiDenyRequest) (*SuccessResponse, error)

// pii.pending - Get pending requests
func (m *PiiMethods) Pending(ctx context.Context) (*PiiPendingResponse, error)
```

---

### Step 3.3: Matrix PII_REQUEST Event (Day 10)
**Task:** Emit PII request events to Matrix  
**Implementation:**
```go
// pkg/matrix/event_emitter.go (add method)
func (e *EventEmitter) EmitPiiRequest(request *keystore.PiiAccessRequest) error {
    matrixEvent := map[string]interface{}{
        "type": "com.armorclaw.pii.request",
        "content": map[string]interface{}{
            "request_id": request.RequestID,
            "agent_id":   request.AgentID,
            "fields":     request.Fields,
            "reason":     request.Reason,
            "expires_at": request.ExpiresAt,
        },
    }
    
    return e.sendEvent(e.controlRoomID, matrixEvent)
}
```

---

## 🔐 Phase 4: User-Held Key Encryption (Days 11-15)

### Step 4.1: KEK Storage (Day 11-12)
**Task:** Memory-only KEK storage with auto-seal  
**Files to CREATE:**
```
pkg/keystore/
├── sealed_keystore.go       # Sealed keystore implementation
├── memory_kek.go            # In-memory KEK storage
└── sealed_keystore_test.go  # Tests
```

**Implementation:**
```go
// pkg/keystore/sealed_keystore.go
package keystore

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "sync"
    "time"
)

type SealedKeystore struct {
    kek           []byte // Key Encryption Key (in memory only)
    sealed        bool
    sealTime      *time.Time
    sessionExpiry time.Time
    mu            sync.RWMutex
    vault         map[string][]byte // Encrypted PII values
}

func NewSealedKeystore() *SealedKeystore {
    return &SealedKeystore{
        sealed: true,
        vault:  make(map[string][]byte),
    }
}

// Challenge generates a nonce for the unseal protocol
func (sk *SealedKeystore) Challenge() (string, error) {
    nonce := make([]byte, 32)
    if _, err := rand.Read(nonce); err != nil {
        return "", err
    }
    
    return base64.StdEncoding.EncodeToString(nonce), nil
}

// Unseal accepts a wrapped KEK and unseals the keystore
func (sk *SealedKeystore) Unseal(wrappedKEK []byte, nonce string) error {
    sk.mu.Lock()
    defer sk.mu.Unlock()
    
    // Verify nonce (implement challenge-response)
    // Unwrap KEK
    kek, err := unwrapKEK(wrappedKEK)
    if err != nil {
        return err
    }
    
    sk.kek = kek
    sk.sealed = false
    sk.sessionExpiry = time.Now().Add(4 * time.Hour)
    
    // Start auto-seal timer
    go sk.autoSealTimer()
    
    return nil
}

// IsSealed returns whether the keystore is sealed
func (sk *SealedKeystore) IsSealed() bool {
    sk.mu.RLock()
    defer sk.mu.RUnlock()
    return sk.sealed
}

// Get retrieves a decrypted PII value
func (sk *SealedKeystore) Get(field string) (string, error) {
    sk.mu.RLock()
    defer sk.mu.RUnlock()
    
    if sk.sealed {
        return "", fmt.Errorf("keystore is sealed")
    }
    
    encryptedValue, exists := sk.vault[field]
    if !exists {
        return "", fmt.Errorf("field not found: %s", field)
    }
    
    // Decrypt with KEK
    return decryptWithKEK(encryptedValue, sk.kek)
}

// Seal seals the keystore and wipes KEK from memory
func (sk *SealedKeystore) Seal() {
    sk.mu.Lock()
    defer sk.mu.Unlock()
    
    // Wipe KEK from memory
    for i := range sk.kek {
        sk.kek[i] = 0
    }
    sk.kek = nil
    sk.sealed = true
    now := time.Now()
    sk.sealTime = &now
}

func (sk *SealedKeystore) autoSealTimer() {
    time.Sleep(4 * time.Hour)
    sk.Seal()
}
```

---

### Step 4.2: Keystore RPC Methods (Day 13)
**Task:** Add keystore unseal RPC methods  
**Files to CREATE:**
```
pkg/rpc/
├── keystore_methods.go       # keystore.* RPC methods
└── keystore_methods_test.go  # Tests
```

**Methods:**
```go
// keystore.challenge - Get unseal challenge
func (m *KeystoreMethods) Challenge(ctx context.Context) (*KeystoreChallengeResponse, error)

// keystore.unseal - Unseal keystore
type KeystoreUnsealRequest struct {
    WrappedKEK string `json:"wrapped_kek"`
    Nonce      string `json:"nonce"`
}

func (m *KeystoreMethods) Unseal(ctx context.Context, req *KeystoreUnsealRequest) (*SuccessResponse, error)

// keystore.sealed - Check if sealed
func (m *KeystoreMethods) Sealed(ctx context.Context) (*KeystoreSealedResponse, error)

// keystore.extend_session - Extend session
func (m *KeystoreMethods) ExtendSession(ctx context.Context) (*KeystoreExtendResponse, error)
```

---

### Step 4.3: Auto-Seal Event Emission (Day 14)
**Task:** Emit seal/unseal events to Matrix  
**Events:**
```go
// com.armorclaw.keystore.sealed
{
  "type": "com.armorclaw.keystore.sealed",
  "content": {
    "reason": "timeout|reboot|manual"
  }
}

// com.armorclaw.keystore.unsealed
{
  "type": "com.armorclaw.keystore.unsealed",
  "content": {
    "expires_at": 1234567890
  }
}
```

---

### Step 4.4: Integration Testing (Day 15)
**Test Scenarios:**
1. Unseal keystore with client-derived KEK
2. Retrieve PII fields
3. Auto-seal after timeout
4. VPS reboot → verify sealed state
5. Session extension

---

## 📊 Summary: Backend Implementation Checklist

### Phase 1: Browser Automation
- [ ] Browser job queue (`pkg/queue/browser_queue.go`)
- [ ] Browser HTTP client (`pkg/skills/browser.go`)
- [ ] Browser RPC methods (`pkg/rpc/browser_methods.go`)
- [ ] Integration tests with Playwright container

### Phase 2: Agent State Machine
- [ ] Agent status enum (`pkg/agent/status.go`)
- [ ] State machine with transitions (`pkg/agent/state_machine.go`)
- [ ] Matrix event emitter (`pkg/matrix/event_emitter.go`)
- [ ] Agent RPC methods (`pkg/rpc/agent_methods.go`)

### Phase 3: PII Access Control
- [ ] PII field sensitivity model (`pkg/keystore/pii.go`)
- [ ] PII request generation (`pkg/keystore/pii_request.go`)
- [ ] PII RPC methods (`pkg/rpc/pii_methods.go`)
- [ ] Matrix PII_REQUEST event emission

### Phase 4: User-Held Keys
- [ ] Sealed keystore (`pkg/keystore/sealed_keystore.go`)
- [ ] Memory KEK storage (`pkg/keystore/memory_kek.go`)
- [ ] Keystore RPC methods (`pkg/rpc/keystore_methods.go`)
- [ ] Auto-seal timer and event emission

---

## 🚀 Deployment Considerations

### Environment Variables
```bash
# Browser service
BROWSER_SERVICE_URL=http://localhost:3000

# Matrix homeserver
MATRIX_HOMESERVER_URL=https://matrix.armorclaw.local
MATRIX_BOT_ACCESS_TOKEN=<bot-token>
CONTROL_ROOM_ID=!controlplane:armorclaw.local

# Keystore
KEYSTORE_AUTO_SEAL_DURATION=4h
KEYSTORE_SESSION_WARNING=30m
```

### Docker Compose
```yaml
services:
  bridge:
    build: .
    environment:
      - BROWSER_SERVICE_URL=http://browser-service:3000
    depends_on:
      - browser-service
      
  browser-service:
    build: ./container/openclaw
    ports:
      - "3000:3000"
```

---

## 📝 Testing Strategy

### Unit Tests
- All `*_test.go` files with ≥80% coverage
- Table-driven tests for state transitions
- Mock HTTP clients for browser service

### Integration Tests
- Real Playwright container
- Test Matrix homeserver
- End-to-end RPC call tests

### E2E Tests
- Full Mobile Secretary workflow:
  1. Unseal keystore from mobile
  2. Start agent task
  3. Agent navigates to form
  4. BlindFill approval request
  5. User approves on mobile
  6. Agent completes form
  7. Auto-seal after timeout

---

**Total Estimated Backend Effort:** 15-20 days (assumes 1 developer)

**Critical Dependencies:**
- ArmorClaw repository access
- Matrix homeserver test environment
- Docker/Playwright setup
- PostgreSQL/SQLite for keystore persistence
