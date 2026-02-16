# Error Handling System Design

> Created: 2026-02-15
> Version: 1.0.0
> Status: Approved

## Overview

A robust error handling system for ArmorClaw that captures detailed traces, assigns structured error codes, and delivers LLM-friendly error reports to administrators via Matrix.

## Requirements

- **Comprehensive Coverage**: All bridge operations, container runtime, and Matrix adapter errors
- **Structured Codes**: Category + Number format (e.g., `CTX-042`) with function/parameter metadata
- **Detailed Traces**: Call stacks, state snapshots, and component-scoped recent events
- **Smart Notification**: All critical errors, first occurrence per code, rate-limited repeats
- **LLM-Friendly Format**: Hybrid plain-text summary with JSON detail block
- **Flexible Admin Resolution**: Setup user → config override → room membership fallback
- **Dual Persistence**: Audit log summaries + dedicated error store with full traces

## Architecture

### Package Structure

```
bridge/pkg/errors/
├── codes.go         # Error code registry (RPC-001, CTX-042, etc.)
├── error.go         # TracedError type with code, trace, state snapshot
├── context.go       # Component-scoped ring buffer for recent events
├── registry.go      # Error code registry with sampling/deduplication
├── notifier.go      # Admin notification with hybrid format
├── store.go         # Dedicated error.db (SQLite) for full traces
├── admin.go         # Admin resolution (setup user → config → room fallback)
└── doc.go           # Package documentation
```

### TracedError Structure

```go
type TracedError struct {
    Code        string                 // "RPC-042"
    Category    string                 // "rpc", "container", "matrix"
    Severity    Severity               // Warning, Error, Critical
    Message     string                 // Human-readable summary
    Function    string                 // "StartContainer"
    File        string                 // "bridge/pkg/docker/client.go"
    Line        int                    // 142
    Stack       []StackFrame           // Full call stack
    Inputs      map[string]any         // Function parameters
    State       map[string]any         // Key variable values at crash
    RecentLogs  []ComponentLogEntry    // Component-scoped recent events
    Timestamp   time.Time
    TraceID     string                 // Unique ID for this occurrence
    RepeatCount int                    // Times seen since last notification
}
```

### Error Code Categories

| Prefix | Category | Range |
|--------|----------|-------|
| `RPC-` | RPC layer | 001-999 |
| `CTX-` | Container/runtime | 001-999 |
| `MAT-` | Matrix adapter | 001-999 |
| `SYS-` | System/infrastructure | 001-999 |

### Severity Levels

- **Warning**: Non-critical issues that don't break functionality
- **Error**: Operation failed but system continues
- **Critical**: System-level failure requiring immediate attention

## Component-Scoped Event Tracking

Each package maintains its own ring buffer of recent events:

```go
type ComponentTracker struct {
    name      string           // "docker", "matrix", "rpc", etc.
    buffer    *RingBuffer      // Last 5-10 events
    mu        sync.RWMutex
}

var components = map[string]*ComponentTracker{
    "docker":    NewComponentTracker("docker", 10),
    "matrix":    NewComponentTracker("matrix", 10),
    "rpc":       NewComponentTracker("rpc", 10),
    "keystore":  NewComponentTracker("keystore", 5),
    "voice":     NewComponentTracker("voice", 10),
    "webrtc":    NewComponentTracker("webrtc", 10),
}
```

When an error occurs, the trace includes:
- Last 5-10 events from the failing component
- Last 3 events from related components

## Admin Resolution

Three-tier fallback chain:

1. **Config Override**: `admin_mxid` in `armorclaw.toml`
2. **Setup User**: Matrix user ID captured during first-run wizard
3. **Room Membership**: First admin/moderator in configured admin room

```go
func (r *AdminResolver) Resolve() ([]AdminTarget, error) {
    // 1. Check explicit config
    if mxid := r.getConfigAdmin(); mxid != "" {
        return []AdminTarget{{MXID: mxid, Source: "config"}}, nil
    }

    // 2. Fallback to setup user
    if r.setupUser != "" {
        return []AdminTarget{{MXID: r.setupUser, Source: "setup"}}, nil
    }

    // 3. Fallback to admin room
    members, _ := r.matrixAdapter.GetRoomMembers(r.adminRoomID)
    for _, m := range members {
        if m.PowerLevel >= 50 {
            return []AdminTarget{{MXID: m.UserID, Source: "room"}}, nil
        }
    }

    return nil, errors.New("no admin target resolved")
}
```

## Message Format

Hybrid format for LLM consumption:

```
🔴 CRITICAL: CTX-042

Container failed to start: permission denied on socket

📍 Location: StartContainer @ docker/client.go:142
🏷️ Trace ID: tr_8f3a2b1c
⏰ 2026-02-15 18:32:05 UTC

```json
{
  "code": "CTX-042",
  "category": "container",
  "severity": "critical",
  "message": "permission denied on socket",
  "function": "StartContainer",
  "file": "bridge/pkg/docker/client.go",
  "line": 142,
  "inputs": {"container_id": "abc123", "image": "armorclaw/agent:v1"},
  "state": {"docker_connected": true, "socket_path": "/var/run/docker.sock"},
  "stack": [
    {"function": "StartContainer", "file": "client.go", "line": 142},
    {"function": "(*RPCServer).ContainerStart", "file": "server.go", "line": 287},
    {"function": "main.main", "file": "main.go", "line": 95}
  ],
  "recent_logs": [...]
}
```

📋 Copy the JSON block above to analyze with an LLM.
```

## Smart Sampling

Notification strategy to prevent spam:

- **Critical**: Always notify immediately
- **First Occurrence**: Notify on first instance of each error code
- **Rate-Limited Repeats**: Subsequent occurrences within 5-minute window are counted but not notified
- **Window Expired**: After rate limit window, notify with accumulated count

```go
func (r *SamplingRegistry) ShouldNotify(err *TracedError) bool {
    if err.Severity == SeverityCritical {
        return true
    }

    record, exists := r.seen[err.Code]
    if !exists {
        r.seen[err.Code] = NewRecord(err)
        return true
    }

    if err.Timestamp.Sub(record.LastSeen) < r.rateLimitWindow {
        record.Count++
        return false
    }

    err.RepeatCount = record.Count
    record.Reset(err)
    return true
}
```

## Persistence

### Audit Log (Summary)

Existing audit log extended with error event type:

```go
type AuditErrorEntry struct {
    Timestamp   time.Time
    EventType   string  // "error_occurred"
    Code        string  // "CTX-042"
    Severity    string  // "critical"
    Message     string  // Short summary
    TraceID     string  // Link to full trace
    Count       int     // Occurrence count
}
```

### Dedicated Error Store

SQLite database at `/var/lib/armorclaw/errors.db`:

```sql
CREATE TABLE errors (
    trace_id     TEXT PRIMARY KEY,
    code         TEXT NOT NULL,
    category     TEXT NOT NULL,
    severity     TEXT NOT NULL,
    message      TEXT NOT NULL,
    trace_json   TEXT NOT NULL,
    first_seen   TIMESTAMP NOT NULL,
    resolved     BOOLEAN DEFAULT FALSE,
    resolved_by  TEXT,
    resolved_at  TIMESTAMP
);

CREATE INDEX idx_errors_code ON errors(code);
CREATE INDEX idx_errors_unresolved ON errors(resolved) WHERE resolved = FALSE;
```

**Retention Policy:**
- Unresolved errors: Kept indefinitely
- Resolved errors: Purged after 30 days

## Error Code Registry

### Container Errors (CTX-)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| CTX-001 | Error | container start failed | Check Docker daemon status, image availability, and resource limits |
| CTX-002 | Error | container exec failed | Verify container is running and command is valid |
| CTX-003 | Critical | container health check timeout | Container may be hung; check logs and consider restart |
| CTX-010 | Critical | permission denied on docker socket | Bridge needs docker group membership or sudo |

### Matrix Errors (MAT-)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| MAT-001 | Error | matrix connection failed | Check homeserver URL and network connectivity |
| MAT-002 | Error | matrix authentication failed | Verify access token or device credentials |
| MAT-010 | Error | E2EE decryption failed | Device keys may be missing or rotated |

### RPC Errors (RPC-)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| RPC-001 | Warning | invalid JSON-RPC request | Check request format matches JSON-RPC 2.0 spec |
| RPC-002 | Error | method not found | Verify method name against RPC API docs |

### System Errors (SYS-)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| SYS-001 | Critical | keystore decryption failed | Master key may be wrong or keystore corrupted |
| SYS-002 | Error | audit log write failed | Check disk space and permissions on /var/lib/armorclaw |

## RPC Integration

### New Methods

**GetErrors** - Retrieve stored errors
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "GetErrors",
  "params": {
    "code": "CTX-001",
    "resolved": false,
    "limit": 20
  }
}
```

**ResolveError** - Mark error as resolved
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ResolveError",
  "params": {
    "trace_id": "tr_8f3a2b1c",
    "resolved_by": "@admin:example.com"
  }
}
```

## Package Integration Pattern

Example integration in docker client:

```go
package docker

import "github.com/armorclaw/bridge/pkg/errors"

var (
    tracker = errors.GetComponentTracker("docker")

    ErrContainerStart    = errors.Register("CTX-001")
    ErrContainerExec     = errors.Register("CTX-002")
    ErrHealthTimeout     = errors.Register("CTX-003")
    ErrDockerPermission  = errors.Register("CTX-010")
)

func (c *Client) StartContainer(ctx context.Context, id string, config ContainerConfig) error {
    tracker.Event("start_container_start", map[string]any{
        "container_id": id,
        "image":        config.Image,
    })

    err := c.client.ContainerStart(ctx, id, types.ContainerStartOptions{})
    if err != nil {
        return ErrContainerStart.Wrap(err).
            WithFunction("StartContainer").
            WithInputs(map[string]any{"id": id, "config": config}).
            WithState(map[string]any{
                "connected":      c.connected,
                "socket_path":    c.socketPath,
                "active_count":   c.activeContainers,
            }).
            Build()
    }

    tracker.Event("start_container_success", map[string]any{"container_id": id})
    return nil
}
```

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `bridge/pkg/errors/codes.go` | Create | Error code definitions |
| `bridge/pkg/errors/error.go` | Create | TracedError type and builder |
| `bridge/pkg/errors/context.go` | Create | Component event trackers |
| `bridge/pkg/errors/registry.go` | Create | Sampling/deduplication |
| `bridge/pkg/errors/notifier.go` | Create | Admin notification sender |
| `bridge/pkg/errors/store.go` | Create | SQLite error persistence |
| `bridge/pkg/errors/admin.go` | Create | Admin resolution chain |
| `bridge/pkg/errors/doc.go` | Create | Package documentation |
| `bridge/pkg/rpc/server.go` | Modify | Add GetErrors, ResolveError methods |
| `bridge/cmd/bridge/main.go` | Modify | Initialize error system |
| `bridge/pkg/docker/client.go` | Modify | Wrap errors with traces |
| `bridge/internal/adapter/matrix.go` | Modify | Wrap errors with traces |

## Configuration

Add to `armorclaw.toml`:

```toml
[errors]
enabled = true
store_path = "/var/lib/armorclaw/errors.db"
rate_limit_window = "5m"
retention_days = 30

[errors.admin]
# Optional override for admin notifications
mxid = "@admin:example.com"
```

## Testing Strategy

1. **Unit Tests**: Error builder, sampling logic, admin resolution
2. **Integration Tests**: Full trace capture, notification delivery
3. **Manual Tests**: Copy JSON to LLM, verify helpful analysis

---

*Design approved: 2026-02-15*
