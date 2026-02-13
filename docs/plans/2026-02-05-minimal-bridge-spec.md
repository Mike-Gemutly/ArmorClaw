# Minimal Local Bridge Specification

**Date:** 2026-02-05
**Status:** Ready for Implementation
**Version:** 1.0.0-minimal

---

## Overview

A stripped-down Local Bridge implementation focused **solely on credential isolation**. The bridge holds the Matrix access token and provides simple send/receive methods to the agent, ensuring credentials never enter the container.

### Design Philosophy

> **Do one thing well:** Keep Matrix credentials out of the ArmorClaw container.

### What This IS

- ✅ Credential isolation (Matrix token stays on host)
- ✅ Simple send/receive interface for agent
- ✅ Unix socket communication (JSON-RPC)
- ✅ Persistent connection to Matrix (survives agent restarts)

### What This is NOT

- ❌ Complex message validation
- ❌ Rate limiting (Conduit handles this)
- ❌ Content filtering
- ❌ Room whitelisting
- ❌ Audit logging

Those features can be added later if needed.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Host Machine                                                │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Minimal Local Bridge (Go binary)                    │    │
│  │                                                      │    │
│  │  ┌────────────────────────────────────────────────┐ │    │
│  │  │ Matrix Client (mautrix-go or simple HTTP)      │ │    │
│  │  │ • Access token stored HERE                     │ │    │
│  │  │ • Maintains sync connection                    │ │    │
│  │  │ • Queues messages for agent                    │ │    │
│  │  └────────────────────────────────────────────────┘ │    │
│  │                                                      │    │
│  │  ┌────────────────────────────────────────────────┐ │    │
│  │  │ JSON-RPC Server (Unix socket)                  │ │    │
│  │  │ • matrix.send(room, text)                      │ │    │
│  │  │ • matrix.receive(timeout)                      │ │    │
│  │  │ • matrix.status()                              │ │    │
│  │  └────────────────────────────────────────────────┘ │    │
│  └───────────────────────────────────▲──────────────────┘    │
│                                   │ Unix socket               │
│                          /run/armorclaw/bridge.sock          │
│                                   │                          │
│  ┌───────────────────────────────┴─────────────────────────┐ │
│  │ ArmorClaw Container                                    │ │
│  │ ┌────────────────────────────────────────────────────┐ │ │
│  │ │ OpenClaw Agent + Minimal Matrix Skill              │ │ │
│  │ │                                                      │ │ │
│  │ │ # Agent code (NEVER sees Matrix token):             │ │ │
│  │ │ bridge = MatrixBridge("/run/armorclaw/bridge.sock")│ │ │
│  │ │ bridge.send("!room:id", "Hello from agent")        │ │ │
│  │ │ messages = bridge.receive()                        │ │ │
│  │ └────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ✅ Agent CANNOT leak Matrix token (it's not there)          │
└─────────────────────────────────────────────────────────────┘
```

---

## File Structure

```
bridge/
├── cmd/
│   └── armorclaw-bridge/
│       └── main.go                 # Entry point (config + start)
├── internal/
│   ├── bridge/
│   │   ├── server.go               # JSON-RPC server
│   │   ├── socket.go               # Unix socket handler
│   │   └── methods.go              # RPC method handlers
│   ├── matrix/
│   │   ├── client.go               # Matrix HTTP client
│   │   ├── sync.go                 # Long-poll sync loop
│   │   ├── credentials.go          # Token storage (file-based)
│   │   └── events.go               # Event queue for agent
│   └── config/
│       ├── config.go               # Configuration loader
│       └── defaults.go             # Default values
├── agent-skill/
│   └── matrix_bridge.py            # Python skill for agent
├── configs/
│   ├── bridge.toml                 # Default config
│   └── bridge.service              # systemd service file
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## JSON-RPC API

### Methods

Only 4 methods needed:

#### 1. `matrix.send`

Send a message to a Matrix room.

```json
// Request
{
  "jsonrpc": "2.0",
  "method": "matrix.send",
  "params": {
    "room_id": "!aBcDeFgH:matrix.example.com",
    "message": "Hello from agent"
  },
  "id": 1
}

// Response (Success)
{
  "jsonrpc": "2.0",
  "result": {
    "event_id": "$event_id",
    "success": true
  },
  "id": 1
}

// Response (Error)
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params: room_id required"
  },
  "id": 1
}
```

#### 2. `matrix.receive`

Receive new messages (blocking with timeout).

```json
// Request
{
  "jsonrpc": "2.0",
  "method": "matrix.receive",
  "params": {
    "timeout": 30
  },
  "id": 2
}

// Response
{
  "jsonrpc": "2.0",
  "result": {
    "events": [
      {
        "room_id": "!aBcDeFgH:matrix.example.com",
        "sender": "@user:matrix.example.com",
        "content": {
          "msgtype": "m.text",
          "body": "Hello agent"
        },
        "timestamp": "2026-02-05T10:30:00Z"
      }
    ]
  },
  "id": 2
}
```

#### 3. `matrix.status`

Get connection status.

```json
// Request
{
  "jsonrpc": "2.0",
  "method": "matrix.status",
  "params": {},
  "id": 3
}

// Response
{
  "jsonrpc": "2.0",
  "result": {
    "connected": true,
    "homeserver": "http://localhost:6167",
    "user_id": "@agent:matrix.example.com",
    "sync_token": "s12345"
  },
  "id": 3
}
```

#### 4. `bridge.ping`

Health check.

```json
// Request
{
  "jsonrpc": "2.0",
  "method": "bridge.ping",
  "params": {},
  "id": 4
}

// Response
{
  "jsonrpc": "2.0",
  "result": {
    "status": "ok",
    "version": "1.0.0-minimal"
  },
  "id": 4
}
```

---

## Configuration

### bridge.toml

```toml
# /etc/armorclaw/bridge.toml

[bridge]
# Unix socket for agent communication
socket_path = "/run/armorclaw/bridge.sock"

# Bridge runs as this user (create during install)
user = "armorclaw"
group = "armorclaw"

# Logging
log_level = "info"           # debug, info, warn, error
log_file = "/var/log/armorclaw/bridge.log"

[matrix]
# Matrix homeserver URL
homeserver_url = "http://localhost:6167"

# Agent's Matrix credentials
# Stored here, NOT in container
username = "agent"
password = "CHANGE_ME_SECURE_PASSWORD"
device_id = "ARMORCLAW_BRIDGE"

# Storage for access token (obtained on first login)
token_file = "/var/lib/armorclaw/matrix_token.json"
sync_token_file = "/var/lib/armorclaw/sync_token"

[server]
# JSON-RPC server settings
max_message_size = 1048576    # 1 MB
read_timeout = 30             # seconds
write_timeout = 30            # seconds
```

---

## Go Implementation

### main.go

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "armorclaw/bridge/internal/config"
    "armorclaw/bridge/internal/bridge"
    "armorclaw/bridge/internal/matrix"
)

func main() {
    // Load configuration
    cfg, err := config.Load("/etc/armorclaw/bridge.toml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize Matrix client
    client, err := matrix.NewClient(cfg)
    if err != nil {
        log.Fatalf("Failed to create Matrix client: %v", err)
    }

    // Login if needed (load or obtain token)
    if err := client.Authenticate(context.Background()); err != nil {
        log.Fatalf("Matrix authentication failed: %v", err)
    }

    // Start sync loop in background
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go client.SyncLoop(ctx)

    // Create and start bridge server
    server := bridge.NewServer(cfg, client)

    if err := server.Start(); err != nil {
        log.Fatalf("Failed to start bridge: %v", err)
    }
    defer server.Stop()

    log.Printf("ArmorClaw Minimal Bridge started on %s", cfg.Bridge.SocketPath)

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
}
```

### internal/matrix/client.go

```go
package matrix

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "sync"
    "time"

    "armorclaw/bridge/internal/config"
)

type Client struct {
    config     *config.Config
    httpClient *http.Client
    accessToken string
    deviceID    string
    userID      string
    syncToken   string

    // Event queue for agent
    eventQueue  chan Event
    queueMutex  sync.RWMutex

    // Credentials storage
    tokenFile   string
    syncFile    string
}

type Event struct {
    RoomID    string                 `json:"room_id"`
    Sender    string                 `json:"sender"`
    Content   map[string]interface{} `json:"content"`
    Timestamp time.Time              `json:"timestamp"`
}

func NewClient(cfg *config.Config) (*Client, error) {
    return &Client{
        config:     cfg,
        httpClient: &http.Client{Timeout: 30 * time.Second},
        eventQueue: make(chan Event, 100),
        tokenFile:  cfg.Matrix.TokenFile,
        syncFile:   cfg.Matrix.SyncTokenFile,
    }, nil
}

func (c *Client) Authenticate(ctx context.Context) error {
    // Try to load existing token
    if err := c.loadToken(); err == nil {
        // Validate token works
        if _, err := c.Whoami(ctx); err == nil {
            log.Println("Using existing Matrix token")
            return nil
        }
        log.Println("Existing token invalid, re-authenticating")
    }

    // Login with username/password
    return c.login(ctx)
}

func (c *Client) login(ctx context.Context) error {
    payload := map[string]string{
        "type":     "m.login.password",
        "user":     c.Matrix.Username,
        "password": c.Matrix.Password,
        "device_id": c.Matrix.DeviceID,
    }

    resp, err := c.httpClient.PostContext(
        ctx,
        c.config.Matrix.HomeserverURL+"/_matrix/client/v3/login",
        "application/json",
        jsonReader(payload),
    )
    if err != nil {
        return fmt.Errorf("login request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("login failed: status %d", resp.StatusCode)
    }

    var result struct {
        AccessToken string `json:"access_token"`
        DeviceID    string `json:"device_id"`
        UserID      string `json:"user_id"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode login response: %w", err)
    }

    c.accessToken = result.AccessToken
    c.deviceID = result.DeviceID
    c.userID = result.UserID

    // Save token for next restart
    if err := c.saveToken(); err != nil {
        log.Printf("Warning: failed to save token: %v", err)
    }

    log.Printf("Logged in as %s", c.userID)
    return nil
}

func (c *Client) SendMessage(ctx context.Context, roomID, message string) (string, error) {
    payload := map[string]interface{}{
        "msgtype": "m.text",
        "body":    message,
    }

    txnID := generateTxnID()

    url := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
        c.config.Matrix.HomeserverURL, roomID, txnID)

    req, err := http.NewRequestWithContext(ctx, "PUT", url, jsonReader(payload))
    if err != nil {
        return "", err
    }

    req.Header.Set("Authorization", "Bearer "+c.accessToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("send failed: status %d", resp.StatusCode)
    }

    var result struct {
        EventID string `json:"event_id"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }

    return result.EventID, nil
}

func (c *Client) GetEvents(timeout int) []Event {
    // Non-blocking read from event queue
    var events []Event

    deadline := time.After(time.Duration(timeout) * time.Second)

    for {
        select {
        case event := <-c.eventQueue:
            events = append(events, event)
        case <-deadline:
            return events
        default:
            if len(events) > 0 {
                return events
            }
            time.Sleep(100 * time.Millisecond)
        }
    }
}

func (c *Client) SyncLoop(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := c.sync(ctx); err != nil {
                log.Printf("Sync error: %v", err)
            }
        }
    }
}

func (c *Client) sync(ctx context.Context) error {
    since := c.syncToken
    if since == "" {
        since = c.loadSyncToken()
    }

    url := fmt.Sprintf("%s/_matrix/client/v3/sync?timeout=30000",
        c.config.Matrix.HomeserverURL)
    if since != "" {
        url += "&since=" + since
    }

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer "+c.accessToken)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var syncResp struct {
        NextBatch string                 `json:"next_batch"`
        Rooms     map[string]struct {
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

    // Process events
    for roomID, room := range syncResp.Rooms.Join {
        for _, rawEvent := range room.Timeline.Events {
            var event struct {
                Type    string                 `json:"type"`
                Sender  string                 `json:"sender"`
                Content map[string]interface{} `json:"content"`
            }

            if err := json.Unmarshal(rawEvent, &event); err != nil {
                continue
            }

            // Only queue m.room.message events
            if event.Type == "m.room.message" {
                select {
                case c.eventQueue <- Event{
                    RoomID:    roomID,
                    Sender:    event.Sender,
                    Content:   event.Content,
                    Timestamp: time.Now(),
                }:
                default:
                    log.Println("Event queue full, dropping event")
                }
            }
        }
    }

    // Save sync token
    c.syncToken = syncResp.NextBatch
    c.saveSyncToken()

    return nil
}

// Token storage (simple file-based)
func (c *Client) saveToken() error {
    data := map[string]string{
        "access_token": c.accessToken,
        "device_id":    c.deviceID,
        "user_id":      c.userID,
    }

    return writeFileJSON(c.tokenFile, data, 0600)
}

func (c *Client) loadToken() error {
    var data map[string]string
    if err := readFileJSON(c.tokenFile, &data); err != nil {
        return err
    }

    c.accessToken = data["access_token"]
    c.deviceID = data["device_id"]
    c.userID = data["user_id"]
    return nil
}

func (c *Client) saveSyncToken() error {
    return os.WriteFile(c.syncFile, []byte(c.syncToken), 0600)
}

func (c *Client) loadSyncToken() string {
    data, err := os.ReadFile(c.syncFile)
    if err != nil {
        return ""
    }
    return string(data)
}

// Helper functions
func jsonReader(v interface{}) io.Reader {
    b, _ := json.Marshal(v)
    return bytes.NewReader(b)
}

func generateTxnID() string {
    return fmt.Sprintf("m%d", time.Now().UnixNano())
}

func writeFileJSON(path string, data interface{}, perm os.FileMode) error {
    b, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, b, perm)
}

func readFileJSON(path string, dest interface{}) error {
    b, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return json.Unmarshal(b, dest)
}
```

---

## Agent-Side Matrix Skill

```python
# agent-skill/matrix_bridge.py
"""
Minimal Matrix Bridge skill for OpenClaw agents.
Communicates with ArmorClaw Local Bridge via Unix socket.
"""

import json
import socket
import time
from typing import Optional, List, Dict


class MatrixBridge:
    """Simple client for ArmorClaw Minimal Bridge"""

    def __init__(self, socket_path: str = "/run/armorclaw/bridge.sock"):
        self.socket_path = socket_path
        self._request_id = 0

    def _rpc(self, method: str, params: dict) -> dict:
        """Send JSON-RPC request and return response"""
        self._request_id += 1

        request = {
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": self._request_id,
        }

        # Connect to bridge socket
        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        try:
            sock.connect(self.socket_path)

            # Send request
            sock.sendall(json.dumps(request).encode() + b"\n")

            # Read response
            response_data = b""
            while True:
                chunk = sock.recv(4096)
                if not chunk:
                    break
                response_data += chunk
                # Try to parse complete JSON
                try:
                    response = json.loads(response_data.decode())
                    break
                except json.JSONDecodeError:
                    continue

        finally:
            sock.close()

        # Check for errors
        if "error" in response:
            raise RuntimeError(f"Bridge error: {response['error']}")

        return response.get("result", {})

    def send(self, room_id: str, message: str) -> str:
        """Send a message to a Matrix room

        Args:
            room_id: Matrix room ID (e.g., "!abc123:example.com")
            message: Plain text message

        Returns:
            Event ID of sent message
        """
        result = self._rpc("matrix.send", {
            "room_id": room_id,
            "message": message,
        })
        return result.get("event_id")

    def receive(self, timeout: int = 30) -> List[Dict]:
        """Receive new messages from Matrix

        Args:
            timeout: Seconds to wait for messages (blocking)

        Returns:
            List of message events
        """
        result = self._rpc("matrix.receive", {
            "timeout": timeout,
        })
        return result.get("events", [])

    def status(self) -> Dict:
        """Get bridge connection status

        Returns:
            Dict with connection status info
        """
        return self._rpc("matrix.status", {})

    def ping(self) -> bool:
        """Health check

        Returns:
            True if bridge is responding
        """
        result = self._rpc("bridge.ping", {})
        return result.get("status") == "ok"


# Example usage in agent
class Agent:
    def __init__(self):
        self.matrix = MatrixBridge()

    def run(self):
        # Check bridge is available
        if not self.matrix.ping():
            print("Bridge not available!")
            return

        # Get status
        status = self.matrix.status()
        print(f"Connected as {status.get('user_id')}")

        # Listen for messages
        while True:
            events = self.matrix.receive(timeout=60)

            for event in events:
                # Handle incoming message
                self.handle_message(event)

    def handle_message(self, event: Dict):
        room_id = event["room_id"]
        sender = event["sender"]
        content = event["content"]

        if content.get("msgtype") == "m.text":
            text = content["body"]

            # Process message
            response = self.process(text)

            # Send response
            self.matrix.send(room_id, response)

    def process(self, text: str) -> str:
        # Agent's actual processing logic
        return f"I received: {text}"


if __name__ == "__main__":
    agent = Agent()
    agent.run()
```

---

## Installation & Deployment

### 1. Build Bridge Binary

```bash
# From bridge/ directory
cd cmd/armorclaw-bridge
go build -o armorclaw-bridge .

# Or use Makefile
make build
```

### 2. Install on System

```bash
# Create user
sudo useradd --system --user-group --home-dir /var/lib/armorclaw armorclaw

# Create directories
sudo mkdir -p /etc/armorclaw
sudo mkdir -p /var/lib/armorclaw
sudo mkdir -p /var/log/armorclaw
sudo mkdir -p /run/armorclaw

# Set permissions
sudo chown -R armorclaw:armorclaw /var/lib/armorclaw
sudo chown -R armorclaw:armorclaw /var/log/armorclaw
sudo chown -R armorclaw:armorclaw /run/armorclaw

# Install binary
sudo cp armorclaw-bridge /usr/local/bin/
sudo chmod +x /usr/local/bin/armorclaw-bridge

# Install config
sudo cp configs/bridge.toml /etc/armorclaw/
sudo chown armorclaw:armorclaw /etc/armorclaw/bridge.toml
sudo chmod 600 /etc/armorclaw/bridge.toml  # Protect credentials

# Install systemd service
sudo cp configs/armorclaw-bridge.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 3. Configure

Edit `/etc/armorclaw/bridge.toml`:
```bash
sudo nano /etc/armorclaw/bridge.toml

# Change the password
password = "your-secure-password-here"
```

### 4. Start Bridge

```bash
# Enable at boot
sudo systemctl enable armorclaw-bridge

# Start now
sudo systemctl start armorclaw-bridge

# Check status
sudo systemctl status armorclaw-bridge

# View logs
sudo journalctl -u armorclaw-bridge -f
```

---

## Testing

### Manual Testing

```bash
# Test bridge is running
sudo ls -la /run/armorclaw/bridge.sock

# Test with Python client
python3 -c "
from matrix_bridge import MatrixBridge
bridge = MatrixBridge()
print(bridge.ping())
print(bridge.status())
"
```

### Integration Test

```bash
# In one terminal: start bridge
./armorclaw-bridge

# In another: run agent
python3 -m agent_skill
```

---

## Memory Usage

| Component | Estimated RAM |
|-----------|---------------|
| Go binary | 10-20 MB |
| Matrix client state | 5-10 MB |
| Event queue | 1-5 MB |
| **Total Bridge** | **~20-40 MB** |

This is negligible compared to the 2 GB budget.

---

## Next Steps After Minimal Bridge

Once Minimal Bridge is working, you can optionally add:

1. **Message validation** - Sanitize outgoing messages
2. **Rate limiting** - Per-agent throttling
3. **Room filtering** - Whitelist specific rooms
4. **Audit logging** - Log all Matrix interactions
5. **Multiple agent support** - One bridge, many containers

But these are **optional enhancements**. Minimal Bridge as specified here provides the core security benefit.

---

**Status:** Ready for implementation
**Estimated effort:** 8-12 hours
**Dependencies:** Go 1.21+, Matrix Conduit running
