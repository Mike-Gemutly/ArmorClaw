<claude-mem-context>
# Recent Activity

### Feb 28, 2026

| ID | Time | T | Title | Read |
|----|------|---|-------|------|
| #102 | 3:40 AM | 🟣 | BrowserCommandHandler created for Matrix event processing | ~450 |
</claude-mem-context>

# Browser Platform Package

## Overview
The `platform/browser` package contains the browser command handling infrastructure for the ArmorClaw Android app.

## BrowserCommandHandler

### Purpose
Processes browser automation events from Matrix and coordinates with the Bridge via JSON-RPC.

### Architecture
```
MatrixSyncManager.events
       │
       ▼
BrowserCommandHandler
       │
       ├── BrowserCommandEvent → Enqueue job via RPC
       ├── BrowserResponseEvent → Update UI state
       ├── BrowserStatusEvent → Update browser status
       ├── AgentStatusEvent → Update ControlPlaneStore
       └── PiiResponseEvent → Handle PII approval/denial
```

### Key Methods
- `start(syncManager)` - Begin listening to Matrix events
- `cancelActiveJob()` - Cancel running browser job
- `retryJob(jobId)` - Retry failed job
- `getQueueStats()` - Get queue statistics

### State Flows
- `browserStatus: StateFlow<BrowserStatus?>` - Current browser state
- `activeJob: StateFlow<BrowserJob?>` - Currently running job
- `jobHistory: StateFlow<List<BrowserJob>>` - Recent job history

### Dependencies
- `BridgeRpcClient` - For JSON-RPC calls to Bridge
- `ControlPlaneStore` - For updating agent status
- `MatrixSyncManager` - For receiving Matrix events

## Event Processing

### BrowserCommandEvent
When an agent sends a browser command:
1. Parse command from Matrix event content
2. Build `BrowserCommand` with type and params
3. Enqueue job via `browserEnqueue()` RPC
4. Fetch and track job details

### AgentStatusEvent
When Bridge sends agent status:
1. Parse status string to `AgentTaskStatus` enum
2. Extract metadata (url, step, progress, etc.)
3. Create `AgentTaskStatusEvent`
4. Update `ControlPlaneStore.processStatusEvent()`

### PII Response Event
Confirmation event for PII approval/denial:
- Logged for audit purposes
- Actual handling done via ControlPlaneStore
