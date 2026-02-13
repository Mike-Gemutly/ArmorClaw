# Missing RPC Methods Implementation Plan

> **Created:** 2026-02-11
> **Priority:** Medium
> **Estimated Effort:** 4-6 hours
> **Status:** Planning Complete

---

## Executive Summary

Documentation references 4 RPC methods that are not implemented in `bridge/pkg/rpc/server.go`. This plan outlines the implementation strategy for each.

| Method | Priority | Effort | Dependencies |
|--------|----------|--------|--------------|
| `store_key` | High | 30 min | ✅ keystore.Store() exists |
| `webrtc.list` | Medium | 30 min | ✅ sessionManager.List() exists |
| `webrtc.get_audit_log` | Medium | 2-3 hrs | ⚠️ Need audit log storage |
| `list_configs` | Low | 1 hr | ⚠️ Need config tracking |

---

## Current State Analysis

### Already Implemented (Leverage These)

```
keystore.Store(cred Credential) error          # Line 407 in keystore.go
sessionManager.List() []*Session                # Line 333 in session.go
sessionManager.Count() int                      # Line 345 in session.go
```

### RPC Server Route Table (server.go lines 406-444)

```go
case "status":           return s.handleStatus(req)
case "health":           return s.handleHealth(req)
case "start":            return s.handleStart(req)
case "stop":             return s.handleStop(req)
case "list_keys":        return s.handleListKeys(req)
case "get_key":          return s.handleGetKey(req)
case "matrix.send":      return s.handleMatrixSend(req)
case "matrix.receive":   return s.handleMatrixReceive(req)
case "matrix.status":    return s.handleMatrixStatus(req)
case "matrix.login":     return s.handleMatrixLogin(req)
case "attach_config":    return s.handleAttachConfig(req)
case "webrtc.start":     return s.handleWebRTCStart(req)
case "webrtc.ice_candidate": return s.handleWebRTCIceCandidate(req)
case "webrtc.end":       return s.handleWebRTCEnd(req)
```

---

## Implementation Plan

### Phase 1: Quick Wins (1 hour)

#### 1.1 Implement `store_key` RPC Method

**Location:** `bridge/pkg/rpc/server.go`

**Why:** CLI `add-key` command works, but direct RPC access is documented and useful for automation.

**Implementation:**

```go
// Add to handleRequest switch statement
case "store_key":
    return s.handleStoreKey(req)

// New handler function
func (s *Server) handleStoreKey(req *Request) *Response {
    var params struct {
        ID          string   `json:"id"`
        Provider    string   `json:"provider"`
        Token       string   `json:"token"`
        DisplayName string   `json:"display_name,omitempty"`
        ExpiresAt   int64    `json:"expires_at,omitempty"`
        Tags        []string `json:"tags,omitempty"`
    }

    if err := json.Unmarshal(req.Params, &params); err != nil {
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error:   &ErrorObj{Code: InvalidParams, Message: err.Error()},
        }
    }

    // Validate required fields
    if params.ID == "" || params.Provider == "" || params.Token == "" {
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error:   &ErrorObj{Code: InvalidParams, Message: "id, provider, and token are required"},
        }
    }

    // Create credential
    cred := keystore.Credential{
        ID:          params.ID,
        Provider:    keystore.Provider(params.Provider),
        Token:       params.Token,
        DisplayName: params.DisplayName,
        CreatedAt:   time.Now().Unix(),
        ExpiresAt:   params.ExpiresAt,
        Tags:        params.Tags,
    }

    // Store in keystore
    if err := s.keystore.Store(cred); err != nil {
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error:   &ErrorObj{Code: InternalError, Message: err.Error()},
        }
    }

    // Log key storage
    s.securityLog.LogSecretAccess(s.ctx, params.ID, params.Provider, slog.String("status", "stored"))

    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]interface{}{
            "id":         params.ID,
            "provider":   params.Provider,
            "created_at": cred.CreatedAt,
        },
    }
}
```

**Testing:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"store_key","params":{"id":"test-key","provider":"openai","token":"sk-test"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

#### 1.2 Implement `webrtc.list` RPC Method

**Location:** `bridge/pkg/rpc/server.go`

**Why:** SessionManager.List() already exists, just need RPC exposure.

**Implementation:**

```go
// Add to handleRequest switch statement
case "webrtc.list":
    return s.handleWebRTCList(req)

// New handler function
func (s *Server) handleWebRTCList(req *Request) *Response {
    // Check WebRTC is configured
    if s.sessionMgr == nil {
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error:   &ErrorObj{Code: InternalError, Message: "WebRTC not configured"},
        }
    }

    // Get all sessions
    sessions := s.sessionMgr.List()

    // Format response
    result := make([]map[string]interface{}, len(sessions))
    for i, sess := range sessions {
        result[i] = map[string]interface{}{
            "session_id":  sess.ID,
            "room_id":     sess.RoomID,
            "state":       sess.State.String(),
            "created_at":  sess.CreatedAt.Format(time.RFC3339),
            "duration":    time.Since(sess.CreatedAt).String(),
            "container_id": sess.ContainerID,
        }
    }

    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]interface{}{
            "active_sessions": s.sessionMgr.Count(),
            "sessions":        result,
        },
    }
}
```

**Testing:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"webrtc.list"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### Phase 2: Audit Logging (2-3 hours)

#### 2.1 Implement `webrtc.get_audit_log` RPC Method

**Why:** Security compliance requires audit trail of all voice calls.

**Architecture:**

```
┌─────────────────────────────────────────────────────────────┐
│  AuditLog Storage                                           │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ SQLite table: webrtc_audit_log                          ││
│  │ - timestamp (ISO 8601)                                  ││
│  │ - event_type (call_created|call_ended|call_rejected|...)││
│  │ - session_id                                            ││
│  │ - room_id                                               ││
│  │ - user_id (from Matrix)                                 ││
│  │ - details (JSON blob)                                   ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

**Implementation Steps:**

1. **Create audit log storage** (`bridge/pkg/audit/audit.go`)

```go
package audit

import (
    "database/sql"
    "encoding/json"
    "sync"
    "time"
    
    _ "github.com/mutecomm/go-sqlcipher/v4"
)

type EventType string

const (
    EventCallCreated   EventType = "call_created"
    EventCallEnded     EventType = "call_ended"
    EventCallRejected  EventType = "call_rejected"
    EventBudgetWarning EventType = "budget_warning"
    EventSecurityViolation EventType = "security_violation"
)

type Entry struct {
    Timestamp time.Time   `json:"timestamp"`
    EventType EventType   `json:"event_type"`
    SessionID string      `json:"session_id"`
    RoomID    string      `json:"room_id"`
    UserID    string      `json:"user_id"`
    Details   interface{} `json:"details"`
}

type AuditLog struct {
    db   *sql.DB
    path string
    mu   sync.RWMutex
}

func NewAuditLog(dbPath string) (*AuditLog, error) {
    // Open/create database with same encryption as keystore
    // Create table if not exists
    // Return instance
}

func (al *AuditLog) Log(entry Entry) error {
    // Insert into webrtc_audit_log table
}

func (al *AuditLog) Query(limit int, eventType EventType) ([]Entry, error) {
    // SELECT with optional filter
}
```

2. **Integrate with SessionManager** (`bridge/pkg/webrtc/session.go`)

```go
// Add audit log to SessionManager
type SessionManager struct {
    sessions sync.Map
    config   SessionConfig
    stopChan chan struct{}
    wg       sync.WaitGroup
    auditLog *audit.AuditLog  // NEW
}

// Log events on state changes
func (sm *SessionManager) Create(...) (*Session, error) {
    // ... existing code ...
    
    // Log creation
    sm.auditLog.Log(audit.Entry{
        Timestamp: time.Now(),
        EventType: audit.EventCallCreated,
        SessionID: sessionID,
        RoomID:    roomID,
        Details:   map[string]interface{}{"ttl": ttl.String()},
    })
    
    return session, nil
}
```

3. **Add RPC handler** (`bridge/pkg/rpc/server.go`)

```go
case "webrtc.get_audit_log":
    return s.handleWebRTCGetAuditLog(req)

func (s *Server) handleWebRTCGetAuditLog(req *Request) *Response {
    var params struct {
        Limit     int    `json:"limit,omitempty"`
        EventType string `json:"event_type,omitempty"`
    }
    
    // Default limit
    if params.Limit == 0 {
        params.Limit = 100
    }
    
    // Query audit log
    entries, err := s.auditLog.Query(params.Limit, audit.EventType(params.EventType))
    if err != nil {
        return &Response{...}
    }
    
    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]interface{}{
            "entries": entries,
            "count":   len(entries),
        },
    }
}
```

---

### Phase 3: Config Management (1 hour)

#### 3.1 Implement `list_configs` RPC Method

**Why:** Parity with `list_keys`, useful for debugging config attachment.

**Implementation:**

```go
case "list_configs":
    return s.handleListConfigs(req)

func (s *Server) handleListConfigs(req *Request) *Response {
    // Read configs directory
    configDir := DefaultConfigsDir
    
    entries, err := os.ReadDir(configDir)
    if err != nil {
        if os.IsNotExist(err) {
            return &Response{
                JSONRPC: "2.0",
                ID:      req.ID,
                Result:  []interface{}{},
            }
        }
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error:   &ErrorObj{Code: InternalError, Message: err.Error()},
        }
    }
    
    // Build result
    configs := make([]map[string]interface{}, 0)
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        configs = append(configs, map[string]interface{}{
            "name":      entry.Name(),
            "path":      filepath.Join(configDir, entry.Name()),
            "size":      info.Size(),
            "modified":  info.ModTime().Format(time.RFC3339),
        })
    }
    
    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  configs,
    }
}
```

---

## Testing Strategy

### Unit Tests

```go
// bridge/pkg/rpc/server_test.go

func TestHandleStoreKey(t *testing.T) {
    // Test valid store
    // Test missing required fields
    // Test invalid provider
}

func TestHandleWebRTCList(t *testing.T) {
    // Test empty list
    // Test with active sessions
    // Test WebRTC not configured
}

func TestHandleWebRTCGetAuditLog(t *testing.T) {
    // Test empty log
    // Test with entries
    // Test filtering by event type
}

func TestHandleListConfigs(t *testing.T) {
    // Test empty directory
    // Test with configs
}
```

### Integration Tests

```bash
# tests/test-rpc-methods.sh

# Test store_key
echo '{"jsonrpc":"2.0","id":1,"method":"store_key","params":{"id":"test","provider":"openai","token":"sk-test"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Test webrtc.list
echo '{"jsonrpc":"2.0","id":1,"method":"webrtc.list"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Test webrtc.get_audit_log
echo '{"jsonrpc":"2.0","id":1,"method":"webrtc.get_audit_log","params":{"limit":10}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Test list_configs
echo '{"jsonrpc":"2.0","id":1,"method":"list_configs"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Documentation Updates

After implementation:

1. **Update rpc-api.md:**
   - Remove "Planned feature" warnings
   - Mark all methods as "✅ Implemented"

2. **Update progress.md:**
   - Change "14 methods" → "18 methods"

3. **Update index.md:**
   - Update method count in status section

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Audit log database corruption | Low | Medium | Use SQLite WAL mode, regular backups |
| Performance impact of audit logging | Low | Low | Async writes, batch inserts |
| Breaking existing clients | Very Low | High | No breaking changes to existing methods |

---

## Acceptance Criteria

- [ ] `store_key` RPC method returns success for valid input
- [ ] `webrtc.list` RPC method returns session list
- [ ] `webrtc.get_audit_log` RPC method returns audit entries
- [ ] `list_configs` RPC method returns config file list
- [ ] All methods have unit tests with >90% coverage
- [ ] Documentation updated to remove "Planned" warnings
- [ ] No regression in existing RPC methods

---

## Timeline

| Phase | Duration | Target Date |
|-------|----------|-------------|
| Phase 1: Quick Wins | 1 hour | Day 1 |
| Phase 2: Audit Logging | 2-3 hours | Day 1-2 |
| Phase 3: Config Management | 1 hour | Day 2 |
| Testing & Documentation | 1 hour | Day 2 |
| **Total** | **5-6 hours** | **2 days** |

---

**Plan Last Updated:** 2026-02-11
