<claude-mem-context>
# Recent Activity

### Feb 28, 2026

| ID | Time | T | Title | Read |
|----|------|---|-------|------|
| #100 | 3:30 AM | 🟣 | BrowserEvents.kt created with comprehensive browser automation event types | ~550 |
| #101 | 3:35 AM | 🟣 | BridgeRpcClient extended with 6 browser queue methods | ~400 |
| #102 | 3:40 AM | 🟣 | BrowserCommandHandler created for Matrix event processing | ~450 |
| #103 | 3:45 AM | 🟣 | MatrixSyncEvent extended with 5 browser event types | ~300 |

### Feb 27, 2026

| ID | Time | T | Title | Read |
|----|------|---|-------|------|
| #76 | 4:45 AM | 🔵 | PiiAccessRequest domain model implements 4-tier sensitivity security model | ~428 |
| #70 | 4:42 AM | 🔵 | PiiAccessRequest.kt located in shared domain model directory | ~204 |
</claude-mem-context>

# Browser Events Domain Model

## Overview
The `BrowserEvents.kt` file contains all domain models for browser automation events exchanged between the Android app and the ArmorClaw Bridge.

## Key Components

### Event Type Constants (`BrowserEventTypes`)
- `NAVIGATE`, `FILL`, `CLICK`, `WAIT`, `EXTRACT`, `SCREENSHOT` - Command events
- `RESPONSE`, `STATUS` - Response events from Bridge
- `AGENT_STATUS` - Agent status updates (`com.armorclaw.agent.status`)
- `PII_RESPONSE` - PII approval/denial responses

### Command Models
- `NavigateCommand` - URL navigation with wait conditions
- `FillCommand` / `FillField` - Form filling with PII references
- `ClickCommand` - Element clicking with wait options
- `WaitCommand` - Conditional waiting
- `ExtractCommand` / `ExtractField` - Data extraction
- `ScreenshotCommand` - Page capture

### Response Models
- `BrowserCommandResponse` - Command execution result
- `BrowserStatusEvent` - Real-time browser state
- `BridgeAgentStatusEvent` - Agent task status with conversion to `AgentTaskStatus`

### Queue Models (JSON-RPC)
- `BrowserJob` - Complete job definition
- `BrowserJobStatus` - PENDING, RUNNING, PAUSED, COMPLETED, FAILED, CANCELLED
- `BrowserJobPriority` - LOW, NORMAL, HIGH, URGENT
- `BrowserEnqueueResponse`, `BrowserJobResponse`, `BrowserJobListResponse`
- `BrowserCancelResponse`, `BrowserRetryResponse`, `BrowserQueueStatsResponse`

### PII Field Mapping
- `PiiFieldRef` - Constants for all PII field references (payment.*, personal.*)
- `mapPiiFieldRef()` - Convert Bridge reference to domain `PiiField`
- `fieldNameToRef()` - Convert domain name back to Bridge reference

### Intervention Detection
- `InterventionType` - CAPTCHA, TWO_FA, ERROR
- `InterventionSubtype` - RECAPTCHA, HCAPTCHA, CLOUDFLARE, SMS, EMAIL, TOTP, FORM_ERROR
- `InterventionInfo` - Detection result with selectors and hints
