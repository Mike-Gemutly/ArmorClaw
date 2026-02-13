# Local Bridge Matrix Gateway Design

**Date:** 2026-02-05
**Status:** Planning
**Related:** `docs/plans/2026-02-05-armorclaw-v1-design.md`

---

## Overview

This document specifies the **Matrix Gateway** component for the Local Bridge, which enables secure communication between OpenClaw agents (running inside ArmorClaw containers) and a Matrix Conduit homeserver running on the host.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Host Machine                                                    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Matrix Conduit (Docker)                                 │    │
│  │ • Homeserver on port 6167                              │    │
│  │ • Database: ./conduit_data/                            │    │
│  │ • Credentials stored separately                        │    │
│  └────────────────────────▲────────────────────────────────┘    │
│                           │                                      │
│                           │ HTTP/WebSocket                      │
│                           │                                      │
│  ┌────────────────────────┴────────────────────────────────┐    │
│  │ Local Bridge (Go Native Binary)                         │    │
│  │ ┌────────────────────────────────────────────────────┐  │    │
│  │ │              Core Bridge Components                │  │    │
│  │ │  • status, health, start, stop methods             │  │    │
│  │ │  • File descriptor passing for secrets             │  │    │
│  │ │  • Container lifecycle management                  │  │    │
│  │ └────────────────────────────────────────────────────┘  │    │
│  │ ┌────────────────────────────────────────────────────┐  │    │
│  │ │           Matrix Gateway Component (NEW)           │  │    │
│  │ │  • Matrix client implementation                     │  │    │
│  │ │  • Holds agent's Matrix credentials                 │  │    │
│  │ │  • Message validation & sanitization                │  │    │
│  │ │  • Rate limiting & throttling                       │  │    │
│  │ │  • Event filtering (agent-only rooms)               │  │    │
│  │ └────────────────────────────────────────────────────┘  │    │
│  └───────────────────────────────────▲────────────────────┘    │
│                                  │ Unix socket                  │
│                          /run/armorclaw/bridge.sock           │
│                                  │                              │
│  ┌───────────────────────────────┴──────────────────────────┐  │
│  │ ArmorClaw Container (Docker)                           │  │
│  │ ┌────────────────────────────────────────────────────┐  │  │
│  │ │ OpenClaw Agent                                     │  │  │
│  │ │                                                    │  │  │
│  │ │ Communications flow:                              │  │  │
│  │ │  1. Agent → Bridge: "Send message to room"        │  │  │
│  │ │  2. Bridge → Matrix: POST /_matrix/client/...     │  │  │
│  │ │  3. Matrix → Bridge: New event via /sync          │  │  │
│  │ │  4. Bridge → Agent: "New message received"        │  │  │
│  │ └────────────────────────────────────────────────────┘  │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                  │
│  External Element Clients → Nginx → Conduit                     │
└──────────────────────────────────────────────────────────────────┘
```

---

## Local Bridge Extension: Matrix Gateway

### New JSON-RPC Methods

```go
// Gateway methods for Matrix communication
const (
    MethodMatrixSend     = "matrix.send"
    MethodMatrixReceive  = "matrix.receive"
    MethodMatrixRooms    = "matrix.rooms"
    MethodMatrixJoin     = "matrix.join"
    MethodMatrixLeave    = "matrix.leave"
    MethodMatrixStatus   = "matrix.status"
)

// MatrixSendRequest sends a message to a Matrix room
type MatrixSendRequest struct {
    RoomID    string `json:"room_id"`              // Target room
    EventType string `json:"event_type"`          // Usually "m.room.message"
    Content   map[string]interface{} `json:"content"` // Message body
}

// MatrixSendResponse confirms message delivery
type MatrixSendResponse struct {
    EventID string `json:"event_id"`  // Matrix event ID
    Success bool   `json:"success"`
}

// MatrixReceiveRequest polls for new messages
type MatrixReceiveRequest struct {
    Timeout int `json:"timeout"`  // Long-poll timeout (seconds)
    Since   string `json:"since"` // Sync token
}

// MatrixReceiveResponse returns new events
type MatrixReceiveResponse struct {
    Events []MatrixEvent `json:"events"`
    NextBatchToken string `json:"next_batch"`
}
```

---

## Configuration

### Bridge Configuration File

```toml
# /etc/armorclaw/bridge.toml

[bridge]
socket_path = "/run/armorclaw/bridge.sock"
user_id = 10002  # armorclaw user
log_level = "info"

[matrix]
# Matrix homeserver connection
homeserver_url = "http://localhost:6167"
# Agent's Matrix credentials (stored in bridge, NOT in container)
agent_username = "agent"
agent_password = "CHANGE_ME_SECURE_PASSWORD"
agent_device_id = "ARMORCLAW_BRIDGE"

# Access token (obtained on first login, stored securely)
access_token = ""
device_id = ""
sync_token = ""

[security]
# Message validation
max_message_size = 65536  # 64 KB
rate_limit_per_minute = 60
rate_limit_burst = 10

# Room restrictions
allowed_rooms = [
    "!aBcDeFgHiJkLmNoPqR:matrix.armorclaw.com",
    "!XyZ123456789:matrix.armorclaw.com"
]
# Or use wildcard for domain
allowed_room_domains = [
    "matrix.armorclaw.com"
]

# Content filtering
block_file_types = [".exe", ".sh", ".bat", ".cmd"]
block_commands = ["/exec", "/eval", "/system"]

[agent]
# Container defaults
default_image = "armorclaw/agent:v1"
memory_limit = "1g"
cpu_quota = "0.5"  # 50% of one CPU
```

---

## Go Implementation Structure

```
bridge/
├── cmd/
│   └── bridge/
│       └── main.go              # Entry point
├── internal/
│   ├── bridge/
│   │   ├── server.go            # JSON-RPC server
│   │   ├── methods.go           # Core bridge methods
│   │   └── socket.go            # Unix socket handler
│   ├── matrix/
│   │   ├── gateway.go           # Matrix gateway component
│   │   ├── client.go            # Matrix client implementation
│   │   ├── sync.go              # Long-poll sync handler
│   │   ├── validator.go         # Message validation
│   │   └── credentials.go       # Credential storage (keyring)
│   ├── container/
│   │   ├── docker.go            # Docker client wrapper
│   │   └── secrets.go           # FD passing for secrets
│   └── config/
│       └── config.go            # Configuration loader
├── pkg/
│   └── api/
│       └── v1/
│           └── bridge.proto     # API definitions (if using gRPC)
├── go.mod
├── go.sum
└── README.md
```

---

## Matrix Gateway Implementation (Key Functions)

### 1. Gateway Initialization

```go
package matrix

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "sync"
)

type Gateway struct {
    config       *Config
    httpClient   *http.Client
    accessToken  string
    deviceID     string
    syncToken    string
    eventQueue   chan Event
    mu           sync.RWMutex
    ctx          context.Context
    cancel       context.CancelFunc
}

func NewGateway(cfg *Config) (*Gateway, error) {
    ctx, cancel := context.WithCancel(context.Background())

    gw := &Gateway{
        config:     cfg,
        httpClient: &http.Client{},
        eventQueue: make(chan Event, 100),
        ctx:        ctx,
        cancel:     cancel,
    }

    // Load stored credentials if available
    if err := gw.loadCredentials(); err != nil {
        // First-time login required
        if err := gw.login(); err != nil {
            return nil, fmt.Errorf("matrix login failed: %w", err)
        }
    }

    // Start background sync
    go gw.syncLoop()

    return gw, nil
}
```

### 2. Matrix Client (Simplified)

```go
func (g *Gateway) login() error {
    payload := map[string]string{
        "type": "m.login.password",
        "user": g.config.AgentUsername,
        "password": g.config.AgentPassword,
    }

    resp, err := g.httpClient.Post(
        g.config.HomeserverURL + "/_matrix/client/r0/login",
        "application/json",
        jsonReader(payload),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result struct {
        AccessToken string `json:"access_token"`
        DeviceID    string `json:"device_id"`
        UserID      string `json:"user_id"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    g.mu.Lock()
    g.accessToken = result.AccessToken
    g.deviceID = result.DeviceID
    g.mu.Unlock()

    // Store credentials securely
    return g.saveCredentials()
}

func (g *Gateway) syncLoop() {
    for {
        select {
        case <-g.ctx.Done():
            return
        default:
            if err := g.sync(); err != nil {
                log.Printf("sync error: %v", err)
                time.Sleep(5 * time.Second)
            }
        }
    }
}

func (g *Gateway) sync() error {
    g.mu.RLock()
    token := g.syncToken
    g.mu.RUnlock()

    url := fmt.Sprintf("%s/_matrix/client/v3/sync?since=%s&timeout=30000",
        g.config.HomeserverURL, token)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+g.accessToken)

    resp, err := g.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var syncResp struct {
        NextBatch string               `json:"next_batch"`
        Rooms     struct {
            Join map[string]struct {
                Timeline struct {
                    Events []json.RawMessage `json:"events"`
                } `json:"timeline"`
            } `json:"join"`
        } `json:"rooms"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
        return err
    }

    // Process events and queue for agent
    for roomID, room := range syncResp.Rooms.Join {
        for _, rawEvent := range room.Timeline.Events {
            if g.isAllowedRoom(roomID) {
                g.eventQueue <- Event{
                    RoomID: roomID,
                    Raw:    rawEvent,
                }
            }
        }
    }

    g.mu.Lock()
    g.syncToken = syncResp.NextBatch
    g.mu.Unlock()

    return nil
}
```

### 3. Message Validation

```go
func (g *Gateway) ValidateMessage(content map[string]interface{}) error {
    // Size check
    if g.config.MaxMessageSize > 0 {
        if size := jsonSize(content); size > g.config.MaxMessageSize {
            return fmt.Errorf("message too large: %d bytes", size)
        }
    }

    // Content type check
    msgType, _ := content["msgtype"].(string)
    if msgType != "m.text" {
        return fmt.Errorf("unsupported message type: %s", msgType)
    }

    // Body validation
    body, _ := content["body"].(string)
    for _, blocked := range g.config.BlockCommands {
        if strings.HasPrefix(body, blocked) {
            return fmt.Errorf("blocked command prefix: %s", blocked)
        }
    }

    // File attachment check
    if url, ok := content["url"].(string); ok {
        if g.isBlockedFileType(url) {
            return fmt.Errorf("blocked file type")
        }
    }

    return nil
}
```

---

## Agent-Side Integration

### Python Matrix Skill (Minimal Client)

```python
# container/opt/openclaw/skills/matrix_skill.py

import json
import socket
from typing import Optional

class MatrixSkill:
    """
    Matrix communication skill for OpenClaw agents.
    Communicates via Local Bridge, never connects directly to Matrix server.
    """

    def __init__(self, bridge_socket: str = "/run/armorclaw/bridge.sock"):
        self.bridge_socket = bridge_socket
        self._request_id = 0

    def _call_bridge(self, method: str, params: dict) -> dict:
        """Send JSON-RPC request to Local Bridge"""
        self._request_id += 1
        request = {
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": self._request_id,
        }

        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        sock.connect(self.bridge_socket)
        sock.sendall(json.dumps(request).encode() + b"\n")

        response = json.loads(sock.recv(65536).decode())
        sock.close()

        if "error" in response:
            raise Exception(response["error"])
        return response["result"]

    def send_message(self, room_id: str, text: str) -> str:
        """Send a message to a Matrix room"""
        result = self._call_bridge("matrix.send", {
            "room_id": room_id,
            "event_type": "m.room.message",
            "content": {
                "msgtype": "m.text",
                "body": text,
            },
        })
        return result["event_id"]

    def receive_messages(self, timeout: int = 30) -> list:
        """Receive new messages from Matrix"""
        result = self._call_bridge("matrix.receive", {
            "timeout": timeout,
        })
        return result.get("events", [])

    def list_rooms(self) -> list:
        """List joined Matrix rooms"""
        result = self._call_bridge("matrix.rooms", {})
        return result.get("rooms", [])
```

---

## Security Considerations

### 1. Credential Isolation

- ✅ Matrix access token stored in Local Bridge only
- ✅ Agent container never sees credentials
- ✅ Credentials stored in encrypted keyring (system keyring)

### 2. Message Sanitization

- ✅ All outgoing messages validated before sending to Matrix
- ✅ All incoming messages sanitized before delivery to agent
- ✅ File attachments blocked by default
- ✅ Command prefixes blocked (/exec, /eval, etc.)

### 3. Rate Limiting

- ✅ Bridge enforces rate limits per agent
- ✅ Prevents spam/abuse by compromised agent
- ✅ Configurable burst and sustained rates

### 4. Room Restrictions

- ✅ Agent can only interact whitelisted rooms
- ✅ Wildcard domain matching possible
- ✅ Prevents agent from joining arbitrary rooms

### 5. Audit Logging

```go
// All Matrix interactions logged
type AuditLog struct {
    Timestamp   time.Time
    Direction   string  // "send" or "receive"
    RoomID      string
    EventType   string
    ContentSize int
    Success     bool
}
```

---

## Deployment

### Docker Compose (Complete Stack)

```yaml
# docker-compose.yml
version: "3.8"

services:
  matrix-conduit:
    image: matrixconduit/matrix-conduit:latest
    restart: unless-stopped
    environment:
      CONDUIT_SERVER_NAME: "${MATRIX_SERVER_NAME}"
      CONDUIT_ALLOW_REGISTRATION: "false"
    volumes:
      - ./conduit_data:/var/lib/matrix-conduit/
    ports:
      - "6167:6167"
    deploy:
      resources:
        limits:
          memory: 400M
        reservations:
          memory: 50M
    networks:
      - armorclaw-net

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - matrix-conduit
    networks:
      - armorclaw-net

networks:
  armorclaw-net:
    driver: bridge

# Local Bridge runs on host (not in Docker)
# ArmorClaw containers started on-demand by bridge
```

---

## Testing Plan

### Unit Tests
- [ ] Gateway initialization and login
- [ ] Message validation
- [ ] Rate limiting enforcement
- [ ] Room whitelist enforcement

### Integration Tests
- [ ] End-to-end message flow
- [ ] Sync long-polling behavior
- [ ] Error handling and recovery

### Security Tests
- [ ] Credential leak prevention
- [ ] Message sanitization
- [ ] Blocked content filtering
- [ ] Rate limit bypass attempts

---

## Open Questions

1. **Multiple Agents:** Should bridge support multiple simultaneous agent containers?
2. **Federation:** Should agent Matrix ID be able to federate with other homeservers?
3. **E2EE:** Should bridge support Olm/Megolm encryption keys?
4. **Voice/Video:** How to handle WebRTC calls (if at all)?

---

**Next Steps:**
1. Confirm architectural approach
2. Implement Local Bridge Matrix Gateway component
3. Create agent-side Matrix skill
4. Write integration tests
5. Deploy and validate memory usage
