# ArmorClaw Bridge RPC API Reference

> **Protocol:** JSON-RPC 2.0
> **Transport:** Unix Domain Socket
> **Socket:** `/run/armorclaw/bridge.sock`
> **Version:** 0.2.0
> **Last Updated:** 2026-02-21

---

## Overview

The ArmorClaw Bridge exposes a JSON-RPC 2.0 API over a Unix domain socket. All requests follow the JSON-RPC 2.0 specification.

### Connection

```bash
# Connect via netcat
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | nc -U /run/armorclaw/bridge.sock

# Connect via Python
import socket
import json

sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
sock.connect("/run/armorclaw/bridge.sock")
request = {"jsonrpc": "2.0", "id": 1, "method": "status"}
sock.sendall(json.dumps(request).encode())
response = json.loads(sock.recv(4096).decode())
```

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": <integer|string>,
  "method": "<method_name>",
  "params": <object|array|null>
}
```

### Response Format

**Success:**
```json
{
  "jsonrpc": "2.0",
  "id": <same_as_request>,
  "result": <any>
}
```

**Error:**
```json
{
  "jsonrpc": "2.0",
  "id": <same_as_request>,
  "error": {
    "code": <integer>,
    "message": <string>,
    "data": <any|null>
  }
}
```

---

## Core Methods

### status

Get bridge status and container information.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "status"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "version": "1.0.0",
    "state": "running",
    "socket": "/run/armorclaw/bridge.sock",
    "containers": 0,
    "container_ids": []
  }
}
```

**Fields:**
- `version` (string) - Bridge version
- `state` (string) - Bridge state: "running" or "stopped"
- `socket` (string) - Socket path
- `containers` (integer) - Number of running containers
- `container_ids` (array) - List of container IDs

---

### health

Health check endpoint.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "health"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "status": "healthy"
  }
}
```

---

### start

Start a new container with injected credentials.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "start",
  "params": {
    "key_id": "openai-key-1",
    "agent_type": "openclaw",
    "image": "armorclaw/agent:v1"
  }
}
```

**Parameters:**
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| key_id | string | ✅ Yes | - | ID of stored credential to inject |
| agent_type | string | ❌ No | "openclaw" | Type of agent to run |
| image | string | ❌ No | "armorclaw/agent:v1" | Container image to use |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "container_id": "abc123def456",
    "container_name": "armorclaw-openclaw-1738864000",
    "status": "running",
    "endpoint": "/run/armorclaw/containers/armorclaw-openclaw-1738864000.sock"
  }
}
```

**Fields:**
- `container_id` (string) - Docker container ID
- `container_name` (string) - Generated container name
- `status` (string) - "running"
- `endpoint` (string) - Container-specific socket path

**Error Codes:**
- `-32602` (InvalidParams) - key_id is required or credential not found
- `-32603` (InternalError) - Container creation failed

---

### stop

Stop a running container.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "stop",
  "params": {
    "container_id": "abc123def456"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| container_id | string | ✅ Yes | Docker container ID to stop |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "status": "stopped",
    "container_id": "abc123def456",
    "container_name": "armorclaw-openclaw-1738864000"
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - container_id is required
- `-2` (ContainerStopped) - Container not found

---

## Keystore Methods

### list_keys

List stored credentials (optionally filtered by provider).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "list_keys",
  "params": {
    "provider": "openai"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| provider | string | ❌ No | Filter by provider (openai, anthropic, etc.) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": [
    {
      "id": "openai-key-1",
      "provider": "openai",
      "display_name": "Production OpenAI Key",
      "created_at": 1738864000,
      "expires_at": 1741459200,
      "tags": ["production", "gpt-4"]
    }
  ]
}
```

**Fields:**
- `id` (string) - Credential ID
- `provider` (string) - Provider name
- `display_name` (string) - Human-readable name
- `created_at` (integer) - Unix timestamp
- `expires_at` (integer|null) - Unix timestamp or null
- `tags` (array) - Associated tags

---

### get_key

Retrieve a specific credential (decrypts the token).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "get_key",
  "params": {
    "id": "openai-key-1"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | ✅ Yes | Credential ID to retrieve |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "id": "openai-key-1",
    "provider": "openai",
    "token": "sk-proj-abc123...",
    "display_name": "Production OpenAI Key",
    "created_at": 1738864000,
    "expires_at": 1741459200,
    "tags": ["production", "gpt-4"]
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - id parameter required
- `-3` (KeyNotFound) - Key not found
- `-32603` (InternalError) - Decryption failed or key expired

---

### store_key

Store a new API key in the encrypted keystore.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "store_key",
  "params": {
    "id": "openai-key-1",
    "provider": "openai",
    "token": "sk-proj-abc123...",
    "display_name": "Production OpenAI Key",
    "tags": ["production", "gpt-4"]
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | ✅ Yes | Unique key identifier |
| provider | string | ✅ Yes | Provider (openai, anthropic, openrouter, google, xai) |
| token | string | ✅ Yes | API token to store |
| display_name | string | ❌ No | Human-readable name |
| expires_at | integer | ❌ No | Unix timestamp for expiration |
| tags | array | ❌ No | Tags for organization |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "result": {
    "id": "openai-key-1",
    "provider": "openai",
    "created_at": 1738864000
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - id, provider, and token are required
- `-32603` (InternalError) - Storage or encryption failed

---

## Matrix Methods

### matrix.login

Login to Matrix homeserver (optional if credentials in config).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "matrix.login",
  "params": {
    "username": "bridge-bot",
    "password": "secret"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| username | string | ✅ Yes | Matrix username |
| password | string | ✅ Yes | Matrix password |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "result": {
    "status": "logged_in",
    "user_id": "@bridge-bot:matrix.armorclaw.com"
  }
}
```
---
### matrix.refresh_token

Manually refresh the Matrix access token using a stored refresh token. This is useful when a token is nearing expiry or has already expired.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "method": "matrix.refresh_token"
}
```

**Parameters:**
None required. Uses the stored refresh token from the Matrix adapter.

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "result": {
    "status": "refreshed"
  }
}
```

**Error Codes:**
- `-32601` (InvalidParams) - No refresh token available
- `-32603` (InternalError) - Token refresh failed

**Notes:**
- P1-HIGH-1: This method enables manual token refresh without requiring re-login
- Tokens are automatically refreshed before API calls when nearing 7-day expiry
- Refresh tokens are encrypted and stored in the keystore

---

### matrix.send

Send a message to a Matrix room.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "method": "matrix.send",
  "params": {
    "room_id": "!room:matrix.armorclaw.com",
    "message": "Hello from ArmorClaw!",
    "msgtype": "m.text"
  }
}
```

**Parameters:**
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| room_id | string | ✅ Yes | - | Matrix room ID |
| message | string | ✅ Yes | - | Message content |
| msgtype | string | ❌ No | "m.text" | Message type (m.text, m.notice) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "result": {
    "event_id": "$event_id",
    "room_id": "!room:matrix.armorclaw.com"
  }
}
```

**Error Codes:**
- `-32603` (InternalError) - Not logged in or send failed

---

### matrix.receive

Receive pending Matrix events (up to 10).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "method": "matrix.receive"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "result": {
    "events": [
      {
        "type": "m.room.message",
        "room_id": "!room:matrix.armorclaw.com",
        "sender": "@user:matrix.armorclaw.com",
        "content": {"msgtype": "m.text", "body": "Hello!"},
        "origin": "1738864000000",
        "event_id": "$event_id"
      }
    ],
    "count": 1
  }
}
```

---

### matrix.status

Get Matrix connection status.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "matrix.status"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "result": {
    "enabled": true,
    "status": "connected",
    "user_id": "@bridge-bot:matrix.armorclaw.com",
    "logged_in": true
  }
}
```

**Fields:**
- `enabled` (boolean) - Matrix communication enabled
- `status` (string) - "connected" or "disconnected"
- `user_id` (string) - Matrix user ID
- `logged_in` (boolean) - Authentication status

---

## Config Methods

### attach_config

Attach a configuration file for use in containers.

This allows sending configs via Element X that can be injected into containers as environment variables or mounted files.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "attach_config",
  "params": {
    "name": "agent.env",
    "content": "MODEL=gpt-4\nTEMPERATURE=0.7",
    "encoding": "raw",
    "type": "env",
    "metadata": {
      "source": "element-x",
      "user": "admin"
    }
  }
}
```

**Parameters:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| name | string | ✅ Yes | - | Config filename (e.g., "agent.env", "config.toml") |
| content | string | ✅ Yes | - | File content (base64 or raw string) |
| encoding | string | ❌ No | "raw" | Content encoding ("base64" or "raw") |
| type | string | ❌ No | - | Config type hint ("env", "toml", "yaml", "json") |
| metadata | object | ❌ No | - | Additional metadata key-value pairs |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "result": {
    "config_id": "config-agent.env-1736294400",
    "name": "agent.env",
    "path": "/run/armorclaw/configs/agent.env",
    "size": 25,
    "type": "env",
    "encoding": "raw",
    "metadata": {
      "source": "element-x",
      "user": "admin"
    }
  }
}
```

**Fields:**
- `config_id` (string) - Unique config ID for reference
- `name` (string) - Config filename
- `path` (string) - Full path to stored config
- `size` (integer) - Content size in bytes
- `type` (string) - Config type (if provided)
- `encoding` (string) - Content encoding used
- `metadata` (object) - Additional metadata (if provided)

**Validation:**
- Path traversal protection (no `../` or absolute paths)
- Size limit: 1 MB maximum
- Invalid characters in filename rejected

**Error Codes:**
- `-32602` (InvalidParams) - Invalid parameters
- `-32603` (InternalError) - Failed to write config file

**Examples:**

Environment file:
```bash
echo '{
  "jsonrpc":"2.0",
  "id":1,
  "method":"attach_config",
  "params":{
    "name":"agent.env",
    "content":"MODEL=gpt-4\nMAX_TOKENS=4096",
    "encoding":"raw",
    "type":"env"
  }
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

Base64-encoded:
```bash
SECRET=$(echo -n "API_KEY=secret123" | base64)
echo "{
  \"jsonrpc\":\"2.0\",
  \"id\":2,
  \"method\":\"attach_config\",
  \"params\":{
    \"name\":\"secret.env\",
    \"content\":\"$SECRET\",
    \"encoding\":\"base64\"
  }
}" | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### list_configs

List all attached configuration files.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "list_configs"
}
```

**Parameters:** None

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    {
      "name": "agent.env",
      "path": "/run/armorclaw/configs/agent.env",
      "size": 256,
      "modified": "2026-02-11T12:00:00Z"
    },
    {
      "name": "model.toml",
      "path": "/run/armorclaw/configs/model.toml",
      "size": 512,
      "modified": "2026-02-11T12:30:00Z"
    }
  ]
}
```

**Fields:**
- `name` (string) - Config filename
- `path` (string) - Full path to config file
- `size` (integer) - File size in bytes
- `modified` (string) - Last modification time (ISO 8601)

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"list_configs"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Error Codes

### JSON-RPC 2.0 Standard Errors
| Code | Name | Description |
|------|------|-------------|
| -32700 | Parse error | Invalid JSON |
| -32600 | Invalid request | Invalid JSON-RPC request |
| -32601 | Method not found | Method does not exist |
| -32602 | Invalid params | Invalid method parameters |
| -32603 | Internal error | Internal error |

### Custom Errors
| Code | Name | Description |
|------|------|-------------|
| -1 | ContainerRunning | Container operation failed |
| -2 | ContainerStopped | Container not found |
| -3 | KeyNotFound | Credential not found |

---

## Security Considerations

### Authentication
- Unix socket file permissions: 0660 (owner + group read/write)
- No authentication required (socket access control via filesystem)

### Authorization
- All Docker operations scoped to create, exec, remove
- Seccomp profiles applied to all containers
- Read-only root filesystem enforced

### Rate Limiting
- Not implemented (v1) - filesystem-based access control

---

## Examples

### Complete Workflow: Start Container with Matrix

```bash
# 1. Start the bridge
sudo ./build/armorclaw-bridge -matrix-enabled \
  -matrix-homeserver https://matrix.armorclaw.com \
  -matrix-username bridge-bot \
  -matrix-password secret

# 2. Login to Matrix
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.login","params":{"username":"bridge-bot","password":"secret"}}' | \
  nc -U /run/armorclaw/bridge.sock

# 3. Start container with OpenAI key
echo '{"jsonrpc":"2.0","id":2,"method":"start","params":{"key_id":"openai-key-1"}}' | \
  nc -U /run/armorclaw/bridge.sock

# 4. Send message to Matrix room
echo '{"jsonrpc":"2.0","id":3,"method":"matrix.send","params":{"room_id":"!room:matrix.armorclaw.com","message":"Agent started"}}' | \
  nc -U /run/armorclaw/bridge.sock

# 5. Check container status
echo '{"jsonrpc":"2.0","id":4,"method":"status"}' | \
  nc -U /run/armorclaw/bridge.sock

# 6. Stop container
echo '{"jsonrpc":"2.0","id":5,"method":"stop","params":{"container_id":"abc123"}}' | \
  nc -U /run/armorclaw/bridge.sock
```

---

### Complete Workflow: Secret Passing and Config Injection

This workflow demonstrates the complete flow of storing credentials via CLI, injecting them into a container, and attaching configuration files.

```bash
# 1. Store an API key in the keystore (use CLI, not RPC)
./build/armorclaw-bridge add-key --provider openai --token sk-proj-abc123... --id openai-key-1

# 2. Retrieve key to verify storage (RPC)
echo '{"jsonrpc":"2.0","id":2,"method":"get_key","params":{"id":"openai-key-1"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response:
# {"jsonrpc":"2.0","id":2,"result":{"id":"openai-key-1","provider":"openai","token":"sk-proj-abc123...",...}}

# 3. Attach configuration file (optional, before starting container)
echo '{"jsonrpc":"2.0","id":3,"method":"attach_config","params":{"name":"agent.env","content":"MODEL=gpt-4\nMAX_TOKENS=4096","encoding":"raw","type":"env"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response:
# {"jsonrpc":"2.0","id":3,"result":{"config_id":"config-agent.env-1736294400","name":"agent.env","path":"/run/armorclaw/configs/agent.env","size":25}}

# 4. Start container with injected credentials
echo '{"jsonrpc":"2.0","id":4,"method":"start","params":{"key_id":"openai-key-1","agent_type":"openclaw","image":"armorclaw/agent:v1"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response:
# {"jsonrpc":"2.0","id":4,"result":{"container_id":"abc123def456","container_name":"armorclaw-openclaw-1738864000","status":"running","endpoint":"/run/armorclaw/containers/armorclaw-openclaw-1738864000.sock"}}

# Secret Injection Flow:
# - Bridge retrieves credentials from keystore
# - Bridge creates secrets file at /run/armorclaw/secrets/<container>.json
# - Bridge mounts file to container at /run/secrets:ro
# - Container entrypoint reads secrets from /run/secrets
# - Container sets environment variables from secrets
# - Agent verifies credentials are present
# - Bridge cleans up secrets file after 10 seconds

# 5. Check container status
echo '{"jsonrpc":"2.0","id":5,"method":"status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response:
# {"jsonrpc":"2.0","id":5,"result":{"version":"1.0.0","state":"running","containers":1,"container_ids":["abc123def456"]}}

# 6. Stop container
echo '{"jsonrpc":"2.0","id":7,"method":"stop","params":{"container_id":"abc123def456"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response:
# {"jsonrpc":"2.0","id":7,"result":{"status":"stopped","container_id":"abc123def456","container_name":"armorclaw-openclaw-1738864000"}}
```

**Security Notes:**
- Credentials are injected via file mount (not environment variables)
- Secrets file exists for 10 seconds only (ephemeral)
- Container reads secrets and verifies before starting agent
- Secrets are not visible in `docker inspect` output
- File-based injection is cross-platform compatible

---

## WebRTC Voice Methods

The WebRTC Voice methods provide secure voice call functionality with Matrix authorization and WebRTC transport. All voice calls require proper session management and security validation.

### webrtc.start

Initiate a WebRTC voice session.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "webrtc.start",
  "params": {
    "room_id": "!abc123:matrix.example.com",
    "ttl": "30m"
  }
}
```

**Parameters:**
- `room_id` (string, required) - Matrix room ID for the call
- `ttl` (string, optional) - Session time-to-live (default: "30m", format: "30m", "1h", etc.)

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "session_id": "session-abc123",
    "sdp_answer": "v=0\r\no=- 456 2 IN IP4 127.0.0.1\r\n...",
    "turn_credentials": {
      "username": "1234567890",
      "password": "turn-password",
      "ttl": 86400
    },
    "signaling_url": "wss://example.com/webrtc",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**Fields:**
- `session_id` (string) - Unique session identifier
- `sdp_answer` (string) - SDP answer for WebRTC connection
- `turn_credentials` (object) - TURN server credentials for NAT traversal
  - `username` (string) - TURN username
  - `password` (string) - TURN password
  - `ttl` (number) - Credential lifetime in seconds
- `signaling_url` (string) - WebSocket signaling server URL
- `token` (string) - Session authentication token

**Errors:**
- `-32602` (Invalid params) - Missing or invalid parameters
- `-32000` (Internal error) - Matrix adapter not configured
- `-32001` (Room access denied) - Room not in allowed list
- `-32002` (Rate limit exceeded) - Too many calls per time window
- `-32003` (Max concurrent calls) - Concurrent call limit reached

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"webrtc.start","params":{"room_id":"!abc123:matrix.example.com","ttl":"30m"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### webrtc.end

Terminate an active WebRTC voice session.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "webrtc.end",
  "params": {
    "session_id": "session-abc123",
    "reason": "user_hangup"
  }
}
```

**Parameters:**
- `session_id` (string, required) - Session ID to terminate
- `reason` (string, optional) - Reason for termination (logged for audit)

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "session_id": "session-abc123",
    "status": "terminated",
    "duration": "5m23s"
  }
}
```

**Fields:**
- `session_id` (string) - Terminated session ID
- `status` (string) - Termination status
- `duration` (string) - Session duration

**Errors:**
- `-32602` (Invalid params) - Missing session_id
- `-32004` (Session not found) - Invalid session ID

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":2,"method":"webrtc.end","params":{"session_id":"session-abc123","reason":"call_complete"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### webrtc.ice_candidate

Submit ICE candidates for a WebRTC session.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "webrtc.ice_candidate",
  "params": {
    "session_id": "session-abc123",
    "candidate": {
      "candidate": "candidate:1 1 UDP 2130706431 192.168.1.100 54321 typ host",
      "sdpMid": "0",
      "sdpMLineIndex": 0
    }
  }
}
```

**Parameters:**
- `session_id` (string, required) - Target session ID
- `candidate` (object, required) - ICE candidate
  - `candidate` (string) - ICE candidate string
  - `sdpMid` (string) - SDP mid identifier
  - `sdpMLineIndex` (number) - SDP media line index

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "session_id": "session-abc123",
    "status": "candidate_accepted"
  }
}
```

**Fields:**
- `session_id` (string) - Session ID
- `status` (string) - Candidate acceptance status

**Errors:**
- `-32602` (Invalid params) - Missing parameters
- `-32004` (Session not found) - Invalid session ID

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":3,"method":"webrtc.ice_candidate","params":{"session_id":"session-abc123","candidate":{"candidate":"candidate:1 1 UDP 2130706431 192.168.1.100 54321 typ host","sdpMid":"0","sdpMLineIndex":0}}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### webrtc.list

List all active WebRTC voice sessions.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "webrtc.list",
  "params": {}
}
```

**Parameters:** None

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "active_sessions": 2,
    "sessions": [
      {
        "session_id": "session-abc123",
        "room_id": "!abc123:matrix.example.com",
        "state": "connected",
        "duration": "5m23s",
        "created_at": "2026-02-08T12:00:00Z"
      },
      {
        "session_id": "session-def456",
        "room_id": "!def456:matrix.example.com",
        "state": "connecting",
        "duration": "0m45s",
        "created_at": "2026-02-08T12:05:00Z"
      }
    ]
  }
}
```

**Fields:**
- `active_sessions` (number) - Count of active sessions
- `sessions` (array) - Array of active session objects
  - `session_id` (string) - Session identifier
  - `room_id` (string) - Matrix room ID
  - `state` (string) - Session state: "connecting", "connected", "failed"
  - `duration` (string) - Session duration
  - `created_at` (string) - Session creation timestamp (ISO 8601)

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":4,"method":"webrtc.list"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### webrtc.get_audit_log

Retrieve the security audit log for voice calls.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "webrtc.get_audit_log",
  "params": {
    "limit": 100
  }
}
```

**Parameters:**
- `limit` (number, optional) - Maximum number of entries to return (default: 100)

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "entries": [
      {
        "timestamp": "2026-02-08T12:00:00Z",
        "event_type": "call_created",
        "session_id": "session-abc123",
        "room_id": "!abc123:matrix.example.com",
        "user_id": "@user:matrix.example.com",
        "details": {}
      },
      {
        "timestamp": "2026-02-08T12:05:23Z",
        "event_type": "call_ended",
        "session_id": "session-abc123",
        "room_id": "!abc123:matrix.example.com",
        "user_id": "@user:matrix.example.com",
        "details": {
          "reason": "user_hangup",
          "duration": "5m23s"
        }
      }
    ]
  }
}
```

**Fields:**
- `entries` (array) - Array of audit log entries
  - `timestamp` (string) - Event timestamp (ISO 8601)
  - `event_type` (string) - Event type: "call_created", "call_ended", "call_rejected", "budget_warning", "security_violation"
  - `session_id` (string) - Associated session ID
  - `room_id` (string) - Matrix room ID
  - `user_id` (string) - Matrix user ID
  - `details` (object) - Additional event details

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":5,"method":"webrtc.get_audit_log","params":{"limit":50}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### WebRTC Voice Error Codes

| Code | Message | Description |
|------|---------|-------------|
| -32000 | Internal error | General internal error |
| -32001 | Room access denied | Room not in allowed list or in blocked list |
| -32002 | Rate limit exceeded | Too many calls per time window |
| -32003 | Max concurrent calls | Concurrent call limit reached |
| -32004 | Session not found | Invalid session ID |
| -32005 | Security policy violation | Security policy check failed |
| -32006 | Budget exceeded | Budget limit exceeded |
| -32007 | TTL exceeded | Session TTL expired |

---

### WebRTC Voice Security Notes

- **Zero-Trust Authorization**: All calls require room-scoped authorization
- **E2EE Required**: End-to-end encryption must be enabled for Matrix rooms
- **Rate Limiting**: Calls are rate-limited per user to prevent abuse
- **Concurrent Limits**: Maximum concurrent calls enforced (configurable)
- **Budget Enforcement**: Token and duration limits tracked and enforced
- **TTL Management**: Sessions expire after configured TTL
- **Audit Logging**: All security events logged for compliance

---

## Recovery Methods (v1.6.0)

Account recovery methods for GAP #6 - allows users to recover access when all devices are lost.

### recovery.generate_phrase

Generate a new 12-word recovery phrase (BIP39-style).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "recovery.generate_phrase"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "phrase": "abandon ability able about above absent absorb abstract absurd abuse access accident",
    "word_count": 12,
    "warning": "Store this phrase securely. It will never be shown again.",
    "recovery_window_hours": 48
  }
}
```

---

### recovery.store_phrase

Store an encrypted recovery phrase in the keystore.

**Parameters:**
- `phrase` (string, required) - The 12-word recovery phrase

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "recovery.store_phrase",
  "params": {
    "phrase": "abandon ability able about above absent absorb abstract absurd abuse access accident"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "message": "Recovery phrase stored successfully"
  }
}
```

---

### recovery.verify

Verify a recovery phrase and start the recovery process.

**Parameters:**
- `phrase` (string, required) - The 12-word recovery phrase
- `new_device_id` (string, required) - ID of the new device being recovered

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "recovery.verify",
  "params": {
    "phrase": "abandon ability able about above absent absorb abstract absurd abuse access accident",
    "new_device_id": "device-abc123"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "recovery_id": "recovery-xyz789",
    "status": "active",
    "started_at": "2026-02-14T15:30:00Z",
    "expires_at": "2026-02-16T15:30:00Z",
    "read_only_mode": true,
    "message": "Recovery started. Full access will be restored after the recovery window."
  }
}
```

---

### recovery.status

Check the status of a recovery attempt.

**Parameters:**
- `recovery_id` (string, required) - The recovery session ID

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "recovery.status",
  "params": {
    "recovery_id": "recovery-xyz789"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "recovery_id": "recovery-xyz789",
    "status": "active",
    "started_at": "2026-02-14T15:30:00Z",
    "expires_at": "2026-02-16T15:30:00Z",
    "attempts": 1,
    "read_only_mode": true
  }
}
```

---

### recovery.complete

Complete the recovery process and invalidate old devices.

**Parameters:**
- `recovery_id` (string, required) - The recovery session ID
- `old_devices` (array of strings, optional) - List of old device IDs to invalidate

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "recovery.complete",
  "params": {
    "recovery_id": "recovery-xyz789",
    "old_devices": ["device-old1", "device-old2"]
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "message": "Recovery completed. Full access restored.",
    "invalidated_count": 2
  }
}
```

---

### recovery.is_device_valid

Check if a device is valid (not invalidated by recovery).

**Parameters:**
- `device_id` (string, required) - The device ID to check

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "recovery.is_device_valid",
  "params": {
    "device_id": "device-abc123"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "device_id": "device-abc123",
    "valid": true
  }
}
```

---

## Platform Methods (v1.6.0)

Platform connection methods for GAP #8 - SDTW (Slack, Discord, Teams, WhatsApp) integration.

### platform.connect

Connect an external platform.

**Parameters:**
- `platform` (string, required) - Platform type: "slack", "discord", "teams", "whatsapp"
- `matrix_room` (string, required) - Matrix room ID for this connection
- `workspace_id` (string) - Workspace/team ID (Slack, Teams)
- `access_token` (string) - OAuth access token (Slack, WhatsApp)
- `bot_token` (string) - Bot token (Discord)
- `client_id` (string) - OAuth client ID (Teams)
- `client_secret` (string) - OAuth client secret (Teams)
- `tenant_id` (string) - Azure tenant ID (Teams)
- `phone_number_id` (string) - WhatsApp Business phone number ID
- `verify_token` (string) - Webhook verify token (WhatsApp)
- `channels` (array of strings) - Channels to connect

**Request (Slack):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "platform.connect",
  "params": {
    "platform": "slack",
    "workspace_id": "T0XXXXXXXX",
    "access_token": "xoxb-xxxxxxxxxxxx",
    "matrix_room": "!abc123:matrix.example.com",
    "channels": ["C0XXXXXXXX"]
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "platform_id": "slack-abc12345",
    "platform": "slack",
    "status": "connected",
    "matrix_room": "!abc123:matrix.example.com",
    "channels": ["C0XXXXXXXX"],
    "message": "slack connected successfully"
  }
}
```

---

### platform.disconnect

Disconnect a platform.

**Parameters:**
- `platform_id` (string, required) - The platform connection ID

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "platform.disconnect",
  "params": {
    "platform_id": "slack-abc12345"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "message": "slack disconnected successfully"
  }
}
```

---

### platform.list

List all connected platforms.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "platform.list"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "connections": [
      {
        "platform_id": "slack-abc12345",
        "platform": "slack",
        "name": "slack (T0XXXXXXXX)",
        "workspace_id": "T0XXXXXXXX",
        "matrix_room": "!abc123:matrix.example.com",
        "channels": ["C0XXXXXXXX"],
        "status": "connected",
        "connected_at": "2026-02-14T15:30:00Z"
      }
    ],
    "count": 1
  }
}
```

---

### platform.status

Get status of a platform connection.

**Parameters:**
- `platform_id` (string, required) - The platform connection ID

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "platform.status",
  "params": {
    "platform_id": "slack-abc12345"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "platform_id": "slack-abc12345",
    "platform": "slack",
    "name": "slack (T0XXXXXXXX)",
    "workspace_id": "T0XXXXXXXX",
    "matrix_room": "!abc123:matrix.example.com",
    "channels": ["C0XXXXXXXX"],
    "status": "connected",
    "connected_at": "2026-02-14T15:30:00Z",
    "uptime_seconds": 3600
  }
}
```

---

### platform.test

Test a platform connection.

**Parameters:**
- `platform_id` (string, required) - The platform connection ID

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "platform.test",
  "params": {
    "platform_id": "slack-abc12345"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "platform_id": "slack-abc12345",
    "platform": "slack",
    "test_passed": true,
    "latency_ms": 150,
    "api_status": "ok",
    "auth_test": "ok",
    "workspace": "T0XXXXXXXX",
    "tested_at": "2026-02-14T15:35:00Z"
  }
}
```

---

## Plugin Methods (v1.8.0)

Plugin methods for managing external adapter plugins. These methods enable dynamic loading of platform adapters without modifying the bridge core.

### plugin.discover

Discover available plugins in the plugin directory.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.discover"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "plugins": [
      {
        "name": "telegram-adapter",
        "version": "1.0.0",
        "api_version": "1.0.0",
        "type": "adapter",
        "description": "Telegram Bot API adapter for ArmorClaw",
        "platform": "telegram",
        "capabilities": {
          "read": true,
          "write": true,
          "media": true,
          "reactions": true
        }
      }
    ],
    "count": 1
  }
}
```

---

### plugin.load

Load a plugin from a shared library file.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| library_path | string | ✅ Yes | Path to the .so plugin file |
| metadata_path | string | ❌ No | Path to metadata.json file |
| enabled | boolean | ❌ No | Enable plugin after loading (default: false) |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.load",
  "params": {
    "library_path": "/var/lib/armorclaw/plugins/telegram-adapter/telegram.so",
    "metadata_path": "/var/lib/armorclaw/plugins/telegram-adapter/metadata.json",
    "enabled": true
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "loaded",
    "library_path": "/var/lib/armorclaw/plugins/telegram-adapter/telegram.so"
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - library_path is required
- `-32603` (InternalError) - Plugin load failed (missing symbol, API mismatch, etc.)

---

### plugin.initialize

Initialize a loaded plugin with configuration and credentials.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | ✅ Yes | Plugin name |
| config | object | ❌ No | Plugin-specific configuration |
| credentials | object | ❌ No | Credentials (supports `@keystore:` prefix) |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.initialize",
  "params": {
    "name": "telegram-adapter",
    "config": {
      "webhook_url": "https://example.com/webhook"
    },
    "credentials": {
      "bot_token": "@keystore:telegram-bot-token"
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "initialized",
    "name": "telegram-adapter"
  }
}
```

**Notes:**
- Credentials with `@keystore:` prefix are resolved from the encrypted keystore
- Plugin must be in "loaded" state before initialization

---

### plugin.start

Start an initialized plugin.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | ✅ Yes | Plugin name |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.start",
  "params": {
    "name": "telegram-adapter"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "running",
    "name": "telegram-adapter"
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - name is required
- `-32603` (InternalError) - Plugin not initialized or start failed

---

### plugin.stop

Stop a running plugin.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | ✅ Yes | Plugin name |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.stop",
  "params": {
    "name": "telegram-adapter"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "stopped",
    "name": "telegram-adapter"
  }
}
```

---

### plugin.unload

Unload a plugin completely.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | ✅ Yes | Plugin name |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.unload",
  "params": {
    "name": "telegram-adapter"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "unloaded",
    "name": "telegram-adapter"
  }
}
```

---

### plugin.list

List all loaded plugins with their status.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.list"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "plugins": [
      {
        "metadata": {
          "name": "telegram-adapter",
          "version": "1.0.0",
          "api_version": "1.0.0",
          "type": "adapter",
          "description": "Telegram Bot API adapter"
        },
        "state": "running",
        "load_time": "2026-02-17T12:00:00Z",
        "start_time": "2026-02-17T12:01:00Z"
      }
    ],
    "count": 1
  }
}
```

**Plugin States:**
- `unloaded` - Plugin not loaded
- `loaded` - Plugin loaded but not initialized
- `initialized` - Plugin initialized with config
- `running` - Plugin is running
- `error` - Plugin encountered error
- `disabled` - Plugin is disabled

---

### plugin.status

Get detailed status of a specific plugin.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | ✅ Yes | Plugin name |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.status",
  "params": {
    "name": "telegram-adapter"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "metadata": {
      "name": "telegram-adapter",
      "version": "1.0.0",
      "api_version": "1.0.0",
      "type": "adapter",
      "platform": "telegram"
    },
    "state": "running",
    "load_time": "2026-02-17T12:00:00Z",
    "start_time": "2026-02-17T12:01:00Z"
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - name is required
- `-32603` (InternalError) - Plugin not found

---

### plugin.health

Check health of all running plugins.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "plugin.health"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "plugins": {
      "telegram-adapter": {
        "healthy": true
      },
      "discord-adapter": {
        "healthy": false,
        "error": "connection timeout"
      }
    },
    "count": 2
  }
}
```

---

## License Methods (v1.9.0)

License validation methods for ArmorClaw premium features. These methods enable feature-gating based on license tiers.

### license.validate

Validate license and check feature access.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| feature | string | ❌ No | Feature to validate (default: "default") |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "license.validate",
  "params": {
    "feature": "slack-adapter"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "valid": true,
    "feature": "slack-adapter",
    "tier": "pro",
    "instance_id": "550e8400e29b41d4a716446655440000",
    "expires_at": "2026-03-05T00:00:00Z",
    "grace_until": "2026-03-08T00:00:00Z",
    "features": ["slack-adapter", "discord-adapter", "pii-scrubber"]
  }
}
```

**Fields:**
- `valid` (boolean) - Whether the license is valid for the feature
- `feature` (string) - Feature that was validated
- `tier` (string) - License tier: "free", "pro", "ent"
- `instance_id` (string) - Unique instance identifier
- `expires_at` (string) - License expiration timestamp
- `grace_until` (string) - End of grace period for offline use
- `features` (array) - List of available features

---

### license.status

Get current license status and tier.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "license.status"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tier": "pro",
    "features": ["slack-adapter", "discord-adapter", "pii-scrubber"],
    "instance_id": "550e8400e29b41d4a716446655440000"
  }
}
```

---

### license.features

Get list of all available features for the current license.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "license.features"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "features": [
      "slack-adapter",
      "discord-adapter",
      "pii-scrubber"
    ],
    "count": 3
  }
}
```

---

### license.set_key

Set or update the license key.

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| license_key | string | ✅ Yes | License key in format SCLW-TIER-XXXXXXXXXXXXXXXX |

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "license.set_key",
  "params": {
    "license_key": "SCLW-PRO-A1B2C3D4E5F67890"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "valid": true,
    "tier": "pro",
    "features": ["slack-adapter", "discord-adapter", "pii-scrubber"],
    "instance_id": "550e8400e29b41d4a716446655440000"
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - license_key is required
- `-32603` (InternalError) - License validation failed

---

### License Tiers

| Tier | Features |
|------|----------|
| `free` | matrix-adapter, keystore, basic-validation, offline-queue |
| `pro` | slack-adapter, discord-adapter, pii-scrubber, audit-log, priority-support |
| `ent` | All Pro features + whatsapp-adapter, teams-adapter, pii-scrubber-hipaa, sso-integration |

---

## Bridge Discovery Methods (v1.10.0)

Bridge discovery methods enable ArmorChat and ArmorTerminal to discover available features and adapt their UI accordingly.

### bridge.capabilities

Returns detailed bridge capabilities for feature discovery. This method is used by clients to determine which features are available and adapt their behavior accordingly.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "bridge.capabilities"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "version": "1.6.2",
    "features": {
      "e2ee": true,
      "key_backup": true,
      "key_recovery": true,
      "cross_signing": true,
      "verification": true,
      "push": true,
      "agents": true,
      "workflows": true,
      "hitl": true,
      "budget": true,
      "containers": true,
      "matrix": true,
      "pii_profiles": true,
      "platform_bridges": true
    },
    "methods": ["status", "health", "agent.start", "workflow.start", ...],
    "websocket_events": ["agent.started", "workflow.completed", "hitl.pending", ...],
    "platforms": {
      "slack": true,
      "discord": true,
      "telegram": true,
      "whatsapp": true
    },
    "limits": {
      "max_containers": 10,
      "max_agents": 5,
      "max_workflow_steps": 50,
      "hitl_timeout_seconds": 60,
      "max_subscribers": 100
    }
  }
}
```

**Response Fields:**
| Field | Type | Description |
|-------|------|-------------|
| version | string | Bridge version |
| features | object | Feature flags for UI adaptation |
| features.e2ee | boolean | End-to-end encryption support |
| features.key_backup | boolean | Key backup support (SSSS) |
| features.key_recovery | boolean | Key recovery support |
| features.agents | boolean | Agent lifecycle support |
| features.workflows | boolean | Workflow execution support |
| features.hitl | boolean | Human-in-the-Loop approval support |
| features.budget | boolean | Budget tracking support |
| methods | string[] | List of available RPC methods |
| websocket_events | string[] | List of WebSocket event types |
| platforms | object | Available platform bridges |
| limits | object | Resource limits and timeouts |

**Usage Example (Kotlin):**
```kotlin
// Query bridge capabilities
val response = bridgeApi.call("bridge.capabilities")
val capabilities = response.result

// Adapt UI based on capabilities
if (capabilities.features["agents"] == true) {
    // Show agent management UI
}

if (capabilities.features["hitl"] == true) {
    // Enable HITL approval workflows
}

// Check if specific method is available
if ("workflow.start" in capabilities.methods) {
    // Enable workflow controls
}
```

---

## Error Management Methods (v1.7.0)

Error management methods for querying and resolving tracked errors. These methods enable LLMs and admins to diagnose issues through structured error codes and traces.

### get_errors

Query tracked errors with optional filters.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "get_errors",
  "params": {
    "code": "CTX-001",
    "category": "container",
    "severity": "error",
    "resolved": false,
    "limit": 50,
    "offset": 0
  }
}
```

**Parameters:**
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| code | string | ❌ No | - | Filter by error code (e.g., "CTX-001") |
| category | string | ❌ No | - | Filter by category: container, matrix, rpc, system, budget, voice |
| severity | string | ❌ No | - | Filter by severity: debug, info, warning, error, critical |
| resolved | boolean | ❌ No | false | Include resolved errors |
| limit | number | ❌ No | 50 | Maximum results to return |
| offset | number | ❌ No | 0 | Pagination offset |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "errors": [
      {
        "trace_id": "tr_abc123",
        "code": "CTX-001",
        "category": "container",
        "severity": "error",
        "message": "container start failed",
        "function": "StartContainer",
        "inputs": {"container_id": "abc123"},
        "state": {"status": "exited", "exit_code": 1},
        "cause": "OCI runtime error",
        "component_events": [
          {"component": "docker", "event": "start", "success": false, "timestamp": "2026-02-15T12:00:00Z"}
        ],
        "timestamp": "2026-02-15T12:00:00Z",
        "resolved": false
      }
    ],
    "stats": {
      "sampling": {
        "total_codes": 25,
        "total_errors": 147
      }
    },
    "query": {
      "code": "CTX-001",
      "category": "container",
      "severity": "error",
      "resolved": false
    }
  }
}
```

**Error Fields:**
- `trace_id` (string) - Unique trace identifier for this error
- `code` (string) - Error code (e.g., CTX-001, MAT-002)
- `category` (string) - Error category
- `severity` (string) - Severity level
- `message` (string) - Human-readable message
- `function` (string) - Function where error occurred
- `inputs` (object) - Input parameters (sanitized)
- `state` (object) - System state at error time
- `cause` (string) - Root cause message
- `component_events` (array) - Related component events
- `timestamp` (string) - ISO 8601 timestamp
- `resolved` (boolean) - Resolution status

**Example:**
```bash
# Get all unresolved container errors
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors","params":{"category":"container","resolved":false}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### resolve_error

Mark an error as resolved.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "resolve_error",
  "params": {
    "trace_id": "tr_abc123",
    "resolved_by": "@admin:matrix.example.com"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| trace_id | string | ✅ Yes | Trace ID of the error to resolve |
| resolved_by | string | ❌ No | Matrix user ID or identifier of resolver |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "success": true,
    "trace_id": "tr_abc123",
    "timestamp": "2026-02-15T14:30:00Z"
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - trace_id is required
- `-32603` (InternalError) - Error not found or update failed

**Example:**
```bash
# Resolve an error
echo '{"jsonrpc":"2.0","id":2,"method":"resolve_error","params":{"trace_id":"tr_abc123","resolved_by":"@admin:example.com"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Appendix A: Provider Values

Valid providers for `list_keys` and credential storage:
- `openai` - OpenAI API
- `anthropic` - Anthropic Claude API
- `openrouter` - OpenRouter API
- `google` - Google Generative AI
- `xai` - xAI (Grok) API

---

## Appendix B: Message Types

Valid `msgtype` values for `matrix.send`:
- `m.text` - Plain text message
- `m.notice` - Notice message (highlighted, bot messages)

---

## Appendix C: Error Codes Reference

Structured error codes for programmatic error handling. Each code follows the format `CAT-NNN` where CAT is the category prefix.

### Container Errors (CTX-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| CTX-001 | Error | container start failed | Check Docker daemon status, image availability, and resource limits |
| CTX-002 | Error | container exec failed | Verify container is running and command is valid |
| CTX-003 | Critical | container health check timeout | Container may be hung; check logs and consider restart |
| CTX-010 | Critical | permission denied on docker socket | Bridge needs docker group membership or sudo |
| CTX-011 | Error | container not found | Container may have been removed or ID is incorrect |
| CTX-012 | Error | container already running | Stop the container first or use a different ID |
| CTX-020 | Error | image pull failed | Check image name, registry access, and network connectivity |
| CTX-021 | Error | image not found | Verify image exists in registry and name is correct |

### Matrix Errors (MAT-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| MAT-001 | Error | matrix connection failed | Check homeserver URL and network connectivity |
| MAT-002 | Error | matrix authentication failed | Verify access token or device credentials |
| MAT-003 | Warning | matrix sync timeout | Homeserver may be slow; will retry automatically |
| MAT-010 | Error | E2EE decryption failed | Device keys may be missing or rotated |
| MAT-011 | Error | E2EE encryption failed | Device keys may be missing; try re-verifying |
| MAT-020 | Error | room join failed | Check room ID and user permissions |
| MAT-021 | Error | message send failed | Check room membership and message content |
| MAT-030 | Critical | voice call failed | Check TURN server configuration and network |

### RPC Errors (RPC-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| RPC-001 | Warning | invalid JSON-RPC request | Check request format matches JSON-RPC 2.0 spec |
| RPC-002 | Error | method not found | Verify method name against RPC API docs |
| RPC-003 | Error | invalid params | Check parameter types and required fields |
| RPC-010 | Error | socket connection failed | Check bridge is running and socket permissions |
| RPC-011 | Warning | request timeout | Operation took too long; may need retry |
| RPC-020 | Error | unauthorized | Check authentication credentials and permissions |

### System Errors (SYS-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| SYS-001 | Critical | keystore decryption failed | Master key may be wrong or keystore corrupted |
| SYS-002 | Error | audit log write failed | Check disk space and permissions on /var/lib/armorclaw |
| SYS-003 | Error | configuration load failed | Check config file syntax and file permissions |
| SYS-010 | Critical | secret injection failed | Check secrets file format and permissions |
| SYS-011 | Error | secret cleanup failed | Secrets may persist; manual cleanup may be needed |
| SYS-020 | Critical | out of memory | Increase system memory or reduce concurrent operations |
| SYS-021 | Critical | disk full | Free up disk space or increase storage |

### Budget Errors (BGT-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| BGT-001 | Warning | budget warning threshold reached | Monitor usage; consider adjusting limits |
| BGT-002 | Critical | budget exceeded | Operation blocked; increase budget or wait for reset |

### Voice/WebRTC Errors (VOX-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| VOX-001 | Error | WebRTC connection failed | Check ICE/TURN configuration and network connectivity |
| VOX-002 | Error | audio capture failed | Check microphone permissions and device availability |
| VOX-003 | Error | audio playback failed | Check speaker configuration and permissions |

---

**API Reference Last Updated:** 2026-02-17
**Compatible with Bridge Version:** 1.9.0
