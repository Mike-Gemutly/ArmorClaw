# Headless Browser Infrastructure Design

> **Created:** 2026-02-28
> **Status:** Design Complete
> **Scope:** Bridge Browser Skill API + Android Integration

---

## Executive Summary

This document specifies the headless browser infrastructure that enables ArmorChat's AI agents to perform web automation tasks (form-filling, browsing, payments) while maintaining user control over sensitive credentials through the BlindFill protocol.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              VPS (User-Controlled)                               │
│                                                                                  │
│  ┌─────────────────────┐         HTTP         ┌─────────────────────────────┐   │
│  │   Bridge (Go)       │ ◄─────────────────► │   Browser Service (Node)    │   │
│  │                     │      :3000           │                             │   │
│  │  • Browser Skill    │                      │  • Playwright + Stealth     │   │
│  │  • Agent State      │                      │  • Session persistence      │   │
│  │  • Job Queue        │                      │  • Human-like behavior      │   │
│  │  • Sealed Keystore  │                      │                             │   │
│  └─────────────────────┘                      └─────────────────────────────┘   │
│           │                                                                   │
│           │ Matrix Events                                                     │
│           ▼                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                        ArmorChat (Android)                               │   │
│  │                                                                          │   │
│  │  • BlindFill UI (credential approval)                                    │   │
│  │  • Agent Status Banner (real-time status)                                │   │
│  │  • JSON-RPC Client (queue management)                                    │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Section 1: Bridge Server Implementation (Go)

### What Exists - Ready to Use

| Component | Location | Status |
|-----------|----------|--------|
| Browser Skill Module | `bridge/pkg/studio/browser_skill.go` | ✅ Implemented |
| Agent State Machine | `bridge/pkg/agent/state_machine.go` | ✅ Implemented |
| Job Queue + JSON-RPC | `bridge/pkg/queue/` | ✅ Implemented |
| Sealed Keystore | `bridge/pkg/keystore/sealed_keystore.go` | ✅ Implemented |
| Device Trust System | `bridge/pkg/trust/` | ✅ Implemented |
| Browser Status Tracking | `bridge/pkg/browser/browser.go` | ✅ Implemented |
| Matrix Adapter | `bridge/internal/adapter/matrix.go` | ✅ Implemented |
| EventBus | `bridge/pkg/eventbus/` | ✅ Implemented |

### Browser Skill Event Types

```go
// Command events (ArmorChat → Bridge)
const (
    BrowserNavigate   = "com.armorclaw.browser.navigate"
    BrowserFill       = "com.armorclaw.browser.fill"
    BrowserClick      = "com.armorclaw.browser.click"
    BrowserWait       = "com.armorclaw.browser.wait"
    BrowserExtract    = "com.armorclaw.browser.extract"
    BrowserScreenshot = "com.armorclaw.browser.screenshot"
)

// Response events (Bridge → ArmorChat)
const (
    BrowserResponse = "com.armorclaw.browser.response"
    BrowserStatus   = "com.armorclaw.browser.status"
    AgentStatus     = "com.armorclaw.agent.status"
    PIIResponse     = "com.armorclaw.pii.response"
)
```

### Command Types

```go
// NavigateCommand - Load a URL
type NavigateCommand struct {
    URL       string    `json:"url"`
    WaitUntil WaitUntil `json:"waitUntil,omitempty"` // "load" | "domcontentloaded" | "networkidle"
    Timeout   int       `json:"timeout,omitempty"`    // ms, default: 30000
}

// FillCommand - Fill form fields
type FillCommand struct {
    Fields      []FillField `json:"fields"`
    AutoSubmit  bool `json:"auto_submit,omitempty"`
    SubmitDelay int  `json:"submit_delay,omitempty"` // ms
}

type FillField struct {
    Selector string `json:"selector"`
    Value    string `json:"value,omitempty"`      // Static value
    ValueRef string `json:"value_ref,omitempty"` // PII reference: "payment.card_number"
}

// ClickCommand - Click an element
type ClickCommand struct {
    Selector string `json:"selector"`
    WaitFor  string `json:"waitFor,omitempty"` // "none" | "navigation" | "selector"
    Timeout  int    `json:"timeout,omitempty"`
}

// WaitCommand - Wait for condition
type WaitCommand struct {
    Condition string `json:"condition"` // "selector" | "timeout" | "url"
    Value     string `json:"value"`
    Timeout   int    `json:"timeout,omitempty"`
}

// ExtractCommand - Retrieve page data
type ExtractCommand struct {
    Fields []ExtractField `json:"fields"`
}

type ExtractField struct {
    Name      string `json:"name"`
    Selector  string `json:"selector"`
    Attribute string `json:"attribute,omitempty"` // default: "textContent"
}

// ScreenshotCommand - Capture page
type ScreenshotCommand struct {
    FullPage bool   `json:"fullPage,omitempty"`
    Selector string `json:"selector,omitempty"`
    Format   string `json:"format,omitempty"` // "png" | "jpeg"
}
```

### Response Format

```go
type BrowserCmdResponse struct {
    Status   BrowserResponseStatus `json:"status"` // "success" | "error"
    Command  string                `json:"command"`
    Data     interface{}           `json:"data,omitempty"`
    Error    *BrowserError         `json:"error,omitempty"`
}

type BrowserError struct {
    Code       BrowserErrorCode `json:"code"`
    Message    string           `json:"message"`
    Screenshot string           `json:"screenshot,omitempty"`
    Selector   string           `json:"selector,omitempty"`
}

// Error codes
const (
    ErrElementNotFound    BrowserErrorCode = "ELEMENT_NOT_FOUND"
    ErrNavigationFailed   BrowserErrorCode = "NAVIGATION_FAILED"
    ErrTimeout            BrowserErrorCode = "TIMEOUT"
    ErrPIIRequestDenied   BrowserErrorCode = "PII_REQUEST_DENIED"
    ErrInvalidSelector    BrowserErrorCode = "INVALID_SELECTOR"
    ErrBrowserNotReady    BrowserErrorCode = "BROWSER_NOT_READY"
)
```

### Agent State Machine

```go
type AgentStatus string

const (
    StatusOffline           AgentStatus = "offline"
    StatusInitializing      AgentStatus = "initializing"
    StatusIdle              AgentStatus = "idle"
    StatusBrowsing          AgentStatus = "browsing"
    StatusFormFilling       AgentStatus = "form_filling"
    StatusProcessingPayment AgentStatus = "processing_payment"
    StatusAwaitingCaptcha   AgentStatus = "awaiting_captcha"
    StatusAwaiting2FA       AgentStatus = "awaiting_2fa"
    StatusAwaitingApproval  AgentStatus = "awaiting_approval"
    StatusError             AgentStatus = "error"
    StatusComplete          AgentStatus = "complete"
)
```

### Agent Status Event Structure

```json
{
  "agent_id": "agent_001",
  "status": "form_filling",
  "previous": "browsing",
  "timestamp": 1709250000000,
  "metadata": {
    "url": "https://shop.example.com/checkout",
    "step": "Filling shipping address",
    "progress": 45,
    "fields_requested": ["payment.card_number", "payment.cvv"],
    "error": null
  }
}
```

---

## Section 2: BlindFill Protocol

### PII Request Event

When a browser command needs sensitive data, the Bridge pauses and requests PII:

```go
type PIIRequestEvent struct {
    RequestID  string   `json:"request_id"`
    FieldRefs  []string `json:"field_refs"`  // e.g., ["payment.card_number", "payment.cvv"]
    Context    string   `json:"context"`     // Human-readable: "Checkout requires payment info"
    Timeout    int      `json:"timeout"`     // Seconds to respond
    Screenshot string   `json:"screenshot,omitempty"` // Current page preview
}
```

### PII Response Event

User responds via ArmorChat:

```go
type PIIResponseEvent struct {
    RequestID string            `json:"request_id"`
    Approved  bool              `json:"approved"`
    Values    map[string]string `json:"values,omitempty"` // Only if approved
}
```

### PII Field References

| Reference | Description | Sensitivity |
|-----------|-------------|-------------|
| `payment.card_number` | Credit/debit card number | HIGH |
| `payment.card_expiry` | Card expiry MM/YY | MEDIUM |
| `payment.cvv` | Card verification code | CRITICAL |
| `payment.card_name` | Cardholder name | LOW |
| `personal.name` | Full name | LOW |
| `personal.address` | Street address | MEDIUM |
| `personal.email` | Email address | LOW |
| `personal.phone` | Phone number | LOW |

### Credential Flow

```
┌─────────────┐    Decrypt     ┌─────────────┐    Fill     ┌────────────┐
│ Android     │ ────────────→ │  Sealed     │ ─────────→ │ Browser    │
│ Keystore    │   (BlindFill)  │  Keystore   │  (once)    │ Session    │
│ (encrypted) │                │  (memory)   │            │ (no store) │
└─────────────┘                └─────────────┘            └────────────┘
      ↑                              ↑
      │    User approves via         │
      │    Matrix event              │
      └──────────────────────────────┘

RULES (enforced by SealedKeystore):
1. Keystore starts in SEALED state
2. Unseal requires: mobile approval OR challenge-response OR time-limit
3. Sessions have TTL (default: 5 minutes)
4. Operations are tracked per session
5. No credential persistence after session expires
```

---

## Section 3: Job Queue (JSON-RPC 2.0)

### Job Structure

```go
type BrowserJob struct {
    ID           string           `json:"id"`
    AgentID      string           `json:"agent_id"`
    RoomID       string           `json:"room_id"`
    UserID       string           `json:"user_id"`
    DefinitionID string           `json:"definition_id"`
    Commands     []BrowserCommand `json:"commands"`
    Priority     int              `json:"priority"`      // Higher = first
    Timeout      time.Duration    `json:"timeout"`
    MaxRetries   int              `json:"max_retries"`
    Status       JobStatus        `json:"status"`
    Attempts     int              `json:"attempts"`
    CurrentStep  int              `json:"current_step"`
    Error        string           `json:"error,omitempty"`
    CreatedAt    time.Time        `json:"created_at"`
    StartedAt    *time.Time       `json:"started_at,omitempty"`
    CompletedAt  *time.Time       `json:"completed_at,omitempty"`
    Result       map[string]interface{} `json:"result,omitempty"`
    Screenshots  []string         `json:"screenshots,omitempty"`
}

type JobStatus string

const (
    JobStatusPending     JobStatus = "pending"
    JobStatusRunning     JobStatus = "running"
    JobStatusCompleted   JobStatus = "completed"
    JobStatusFailed      JobStatus = "failed"
    JobStatusCancelled   JobStatus = "cancelled"
    JobStatusAwaitingPII JobStatus = "awaiting_pii"
)
```

### JSON-RPC Methods

| Method | Description |
|--------|-------------|
| `browser.enqueue` | Create browser job |
| `browser.get_job` | Get job status |
| `browser.list_jobs` | List jobs (filterable) |
| `browser.cancel_job` | Cancel running job |
| `browser.retry_job` | Retry failed job |
| `browser.queue_stats` | Get queue statistics |
| `browser.queue_start` | Start queue workers |
| `browser.queue_stop` | Stop queue workers |

### Example: Enqueue Job

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "browser.enqueue",
  "params": {
    "agent_id": "agent_001",
    "room_id": "!room:matrix.org",
    "definition_id": "checkout_agent",
    "commands": [
      {
        "type": "navigate",
        "content": {"url": "https://shop.example.com/checkout", "waitUntil": "load"}
      },
      {
        "type": "fill",
        "content": {
          "fields": [
            {"selector": "#card-number", "value_ref": "payment.card_number"},
            {"selector": "#cvv", "value_ref": "payment.cvv"}
          ]
        }
      },
      {
        "type": "click",
        "content": {"selector": "#submit-btn", "waitFor": "navigation"}
      }
    ],
    "priority": 5,
    "timeout": 300,
    "max_retries": 2
  }
}
```

---

## Section 4: Browser Service (External)

### What's Needed

| Component | Description | Effort |
|-----------|-------------|--------|
| Browser Service (Docker) | Node.js + Playwright + Stealth | 2-3 days |
| Browser Service API | HTTP endpoints for Bridge | 1 day |
| Anti-Detection Config | Fingerprint, behavior, evasion | 1 day |

### Service Architecture

```
browser-service/
├── src/
│   ├── server.ts           # HTTP server
│   ├── session.ts          # Browser session management
│   ├── commands.ts         # Command handlers
│   ├── stealth.ts          # Anti-detection configuration
│   ├── intervention.ts     # Captcha/2FA detection
│   └── humanizer.ts        # Human-like behavior
├── Dockerfile
└── package.json
```

### HTTP API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/session/create` | POST | Create browser session |
| `/session/:id/navigate` | POST | Navigate to URL |
| `/session/:id/fill` | POST | Fill form fields |
| `/session/:id/click` | POST | Click element |
| `/session/:id/wait` | POST | Wait for condition |
| `/session/:id/extract` | POST | Extract page data |
| `/session/:id/screenshot` | POST | Capture page |
| `/session/:id/detect-intervention` | GET | Check for CAPTCHA/2FA |
| `/session/:id/close` | POST | Close session |

### Anti-Detection Configuration

```typescript
export const enhancedStealthConfig = {
  // Fingerprint consistency (same per session)
  fingerprint: {
    seed: "user-session-id",
    webgl: {
      vendor: "Google Inc. (NVIDIA)",
      renderer: "ANGLE (NVIDIA GeForce RTX 3070)"
    },
    audioContext: { noise: 0.0001 },
    canvas: { noise: true }
  },

  // Human-like interactions
  behavior: {
    typing: {
      minDelay: 50,
      maxDelay: 180,
      variance: 0.3,
      mistakeRate: 0.015,
      burstTyping: true
    },
    mouse: {
      movement: "bezier-curve",
      speedVariation: [200, 800],
      overshoot: 0.05
    },
    scrolling: {
      smooth: true,
      speed: "variable",
      pauses: true
    },
    pageLoad: {
      waitBeforeAction: [500, 2000],
      scrollBeforeFill: true
    }
  },

  // Detection evasion
  evasion: {
    webdriver: { hidden: true },
    chrome: { app: true, csi: true, loadTimes: true },
    permissions: { query: true },
    plugins: { array: true },
    languages: ["en-US", "en"],
    hardwareConcurrency: { value: 8 },
    deviceMemory: { value: 8 },
    platform: "Win32"
  }
}
```

### Intervention Detection

```typescript
const INTERVENTION_SELECTORS = {
  captcha: [
    'iframe[src*="recaptcha"]',
    'iframe[src*="hcaptcha"]',
    '.g-recaptcha',
    '.h-captcha',
    '#captcha',
    '[class*="captcha"]'
  ],
  twofa: [
    'input[placeholder*="code"]',
    'input[placeholder*="OTP"]',
    'input[placeholder*="verification"]',
    'input[maxlength="6"][type="text"]',
    'input[maxlength="6"][type="number"]'
  ],
  unexpected: [
    '.error-message',
    '.alert-danger',
    '[role="alert"]',
    '.form-error'
  ]
};

async function detectIntervention(page: Page): Promise<InterventionInfo | null> {
  for (const [type, selectors] of Object.entries(INTERVENTION_SELECTORS)) {
    for (const selector of selectors) {
      const element = await page.$(selector);
      if (element) {
        const screenshot = await page.screenshot({ encoding: 'base64' });
        return { type, selector, screenshot, timestamp: Date.now() };
      }
    }
  }
  return null;
}
```

---

## Section 5: Android Integration Requirements

### What Exists (To Be Verified)

| Component | Expected Location | Status |
|-----------|-------------------|--------|
| BlindFill UI | `shared/ui/components/BlindFillCard.kt` | ❓ Verify |
| Status Banner | `shared/ui/components/AgentStatusBanner.kt` | ❓ Verify |
| PII Domain Models | `shared/domain/model/PiiAccessRequest.kt` | ❓ Verify |
| Matrix Event Listeners | `shared/matrix/` | ❓ Verify |
| JSON-RPC Client | `shared/bridge/` | ❓ Verify |

### What's Needed

| Component | Description | Effort |
|-----------|-------------|--------|
| Matrix Event Listeners | Subscribe to browser events | 1 day |
| JSON-RPC Client | Queue management | 0.5 day |
| PII Field Mapping | Map Bridge refs to domain model | 0.5 day |
| Integration Testing | End-to-end browser automation | 1-2 days |

### Matrix Event Listeners

```kotlin
class BrowserCommandHandler(
    private val matrixClient: MatrixClient,
    private val controlPlaneStore: ControlPlaneStore
) {
    fun subscribe() {
        // Listen for browser responses (command results)
        matrixClient.onEvent("com.armorclaw.browser.response") { event ->
            handleBrowserResponse(event)
        }

        // Listen for browser status updates
        matrixClient.onEvent("com.armorclaw.browser.status") { event ->
            handleBrowserStatus(event)
        }

        // Listen for agent status changes
        matrixClient.onEvent("com.armorclaw.agent.status") { event ->
            handleAgentStatus(event)
        }
    }

    private fun handleAgentStatus(event: MatrixEvent) {
        val status = AgentStatusEvent.fromJson(event.content)

        when (status.status) {
            "idle" -> showIdleState()
            "browsing" -> showBrowsingState(status.metadata.url)
            "form_filling" -> showFormFillingState(status.metadata.progress)
            "awaiting_approval" -> showBlindFillDialog(status.metadata.fieldsRequested)
            "awaiting_captcha" -> showCaptchaUI()
            "awaiting_2fa" -> show2FAUI()
            "processing_payment" -> showProcessingPayment()
            "complete" -> showComplete(status.metadata)
            "error" -> showError(status.metadata.error)
        }
    }
}
```

### PII Field Mapping

```kotlin
fun mapPIIFieldRef(ref: String): PiiField {
    return when (ref) {
        "payment.card_number" -> PiiField(
            name = "Card Number",
            sensitivity = SensitivityLevel.HIGH,
            description = "Credit or debit card number",
            currentValue = maskCardNumber(storedCard?.last4)
        )
        "payment.cvv" -> PiiField(
            name = "CVV",
            sensitivity = SensitivityLevel.CRITICAL,
            description = "Card verification code"
        )
        "payment.card_expiry" -> PiiField(
            name = "Expiry Date",
            sensitivity = SensitivityLevel.MEDIUM,
            description = "MM/YY format"
        )
        "payment.card_name" -> PiiField(
            name = "Cardholder Name",
            sensitivity = SensitivityLevel.LOW
        )
        "personal.name" -> PiiField(
            name = "Full Name",
            sensitivity = SensitivityLevel.LOW
        )
        "personal.address" -> PiiField(
            name = "Address",
            sensitivity = SensitivityLevel.MEDIUM
        )
        "personal.email" -> PiiField(
            name = "Email",
            sensitivity = SensitivityLevel.LOW
        )
        "personal.phone" -> PiiField(
            name = "Phone",
            sensitivity = SensitivityLevel.LOW
        )
        else -> PiiField(
            name = ref.substringAfterLast("."),
            sensitivity = SensitivityLevel.HIGH,
            description = "Requested: $ref"
        )
    }
}
```

### JSON-RPC Client

```kotlin
class BrowserQueueClient(
    private val bridgeUrl: String,
    private val authToken: String
) {
    suspend fun enqueueJob(request: EnqueueJobRequest): Result<BrowserJob>
    suspend fun getJob(jobId: String): Result<BrowserJob>
    suspend fun cancelJob(jobId: String): Result<Unit>
    suspend fun retryJob(jobId: String): Result<Unit>
    suspend fun getQueueStats(): Result<QueueStats>
    suspend fun listJobs(agentId: String? = null, status: List<String>? = null): Result<List<BrowserJob>>
}
```

### PII Response Handling

```kotlin
fun respondToPIIRequest(requestId: String, approved: Boolean, values: Map<String, String>?) {
    matrixClient.sendEvent(roomId, mapOf(
        "type" to "com.armorclaw.pii.response",
        "content" to mapOf(
            "request_id" to requestId,
            "approved" to approved,
            "values" to (values ?: emptyMap<String, String>())
        )
    ))

    controlPlaneStore.removePiiRequest(requestId)
}

fun approvePiiRequest(approvedFields: Set<String>) {
    val request = _pendingPiiRequest.value ?: return

    val values = mutableMapOf<String, String>()
    approvedFields.forEach { fieldName ->
        when (fieldName) {
            "Card Number" -> values["payment.card_number"] = secureStorage.getCardNumber()
            "CVV" -> values["payment.cvv"] = secureStorage.getCVV()
            "Expiry Date" -> values["payment.card_expiry"] = secureStorage.getCardExpiry()
            "Cardholder Name" -> values["payment.card_name"] = userPrefs.getCardName()
            "Full Name" -> values["personal.name"] = userPrefs.getFullName()
            "Address" -> values["personal.address"] = userPrefs.getAddress()
            "Email" -> values["personal.email"] = userPrefs.getEmail()
            "Phone" -> values["personal.phone"] = userPrefs.getPhone()
        }
    }

    respondToPIIRequest(request.requestId, approved = true, values = values)
}
```

---

## Section 6: Event Types Reference

### Complete Event Reference

| Event Type | Direction | Purpose |
|------------|-----------|---------|
| `com.armorclaw.browser.navigate` | Client → Bridge | Navigate to URL |
| `com.armorclaw.browser.fill` | Client → Bridge | Fill form fields |
| `com.armorclaw.browser.click` | Client → Bridge | Click element |
| `com.armorclaw.browser.wait` | Client → Bridge | Wait for condition |
| `com.armorclaw.browser.extract` | Client → Bridge | Extract page data |
| `com.armorclaw.browser.screenshot` | Client → Bridge | Capture page |
| `com.armorclaw.browser.response` | Bridge → Client | Command result |
| `com.armorclaw.browser.status` | Bridge → Client | Browser state change |
| `com.armorclaw.agent.status` | Bridge → Client | Agent state transition |
| `com.armorclaw.pii.response` | Client → Bridge | User PII approval/denial |

### Agent Status States

| State | Description |
|-------|-------------|
| `offline` | Agent not connected |
| `initializing` | Agent starting up |
| `idle` | Agent available |
| `browsing` | Navigating to URL |
| `form_filling` | Filling form fields |
| `processing_payment` | Submitting payment |
| `awaiting_captcha` | Needs CAPTCHA resolution |
| `awaiting_2fa` | Needs 2FA code |
| `awaiting_approval` | Needs PII approval (BlindFill) |
| `error` | Task failed |
| `complete` | Task succeeded |

---

## Section 7: Implementation Checklist

### Bridge (Already Complete)

- [x] Browser Skill Module (`bridge/pkg/studio/browser_skill.go`)
- [x] Agent State Machine (`bridge/pkg/agent/state_machine.go`)
- [x] Job Queue + JSON-RPC (`bridge/pkg/queue/`)
- [x] Sealed Keystore (`bridge/pkg/keystore/sealed_keystore.go`)
- [x] Device Trust System (`bridge/pkg/trust/`)
- [x] Browser Status Tracking (`bridge/pkg/browser/browser.go`)
- [x] Matrix Adapter (`bridge/internal/adapter/matrix.go`)
- [x] EventBus (`bridge/pkg/eventbus/`)

### Browser Service (To Build)

- [ ] Docker container with Playwright
- [ ] HTTP API server
- [ ] Session management
- [ ] Anti-detection configuration
- [ ] Intervention detection
- [ ] Human-like behavior module

### Android Integration (To Build/Verify)

- [ ] Verify BlindFillCard component
- [ ] Verify AgentStatusBanner component
- [ ] Verify PiiAccessRequest domain model
- [ ] Implement Matrix event listeners
- [ ] Implement JSON-RPC client
- [ ] Implement PII field mapping
- [ ] Integration testing

---

## Appendix A: Complete Flow Example

```kotlin
// User initiates checkout from ArmorChat
fun startCheckout(url: String, agent: AgentDefinition) = scope.launch {
    // 1. Create browser job via JSON-RPC
    val job = queueClient.enqueueJob(
        EnqueueJobRequest(
            agent_id = agent.id,
            room_id = currentRoomId,
            definition_id = agent.definitionId,
            commands = listOf(
                BrowserCommandJson("navigate", mapOf("url" to url, "waitUntil" to "load")),
                BrowserCommandJson("fill", mapOf(
                    "fields" to listOf(
                        mapOf("selector" to "#email", "value" to userPrefs.email),
                        mapOf("selector" to "#address", "value_ref" to "personal.address")
                    )
                )),
                BrowserCommandJson("click", mapOf("selector" to "#continue-btn", "waitFor" to "navigation"))
            ),
            priority = 5,
            timeout = 300
        )
    ).getOrThrow()

    // 2. Subscribe to status updates
    matrixService.onEvent("com.armorclaw.agent.status")
        .filter { it.content.agent_id == agent.id }
        .collect { event ->
            val status = AgentStatusEvent.fromJson(event.content)
            when (status.status) {
                "awaiting_approval" -> {
                    // Show BlindFill for PII fields
                    showBlindFillDialog(status.metadata?.fields_requested ?: emptyList())
                }
                "awaiting_captcha" -> {
                    // Show CAPTCHA resolution UI
                    showCaptchaUI(status.metadata?.screenshot)
                }
                "complete" -> {
                    // Job finished successfully
                    showCheckoutComplete(status.metadata)
                }
                "error" -> {
                    showError(status.metadata?.error ?: "Unknown error")
                }
            }
        }
}
```

---

## Appendix B: Security Architecture

### Sealed Keystore

```go
type SealedKeystore struct {
    base            *Keystore
    sessions        map[string]*SealedSession     // session_id -> session
    agentSession    map[string]string             // agent_id -> session_id
    pending         map[string]*PendingUnsealRequest
    defaultTTL      time.Duration
    policy          UnsealPolicy
}

type UnsealPolicy string

const (
    PolicyMobileApproval UnsealPolicy = "mobile_approval" // Requires mobile device approval
    PolicyChallenge      UnsealPolicy = "challenge"       // Ed25519 challenge-response
    PolicyTimeLimited    UnsealPolicy = "time_limited"    // Limited time without approval
    PolicyAuto           UnsealPolicy = "auto"            // Development only
)
```

### Zero-Trust Device Verification

```go
type Device struct {
    ID             string        `json:"id"`
    UserID         string        `json:"user_id"`
    Name           string        `json:"name"`
    Fingerprint    string        `json:"fingerprint"`
    TrustLevel     TrustLevel    `json:"trust_level"`     // "untrusted" | "limited" | "trusted" | "verified"
    VerificationStatus string    `json:"verification_status"`
    LastSeen       time.Time     `json:"last_seen"`
    RiskScore      int           `json:"risk_score"`      // 0-100
}
```

---

*Document generated from brainstorming session on 2026-02-28*
