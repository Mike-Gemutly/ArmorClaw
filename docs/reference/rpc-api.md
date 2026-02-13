# ArmorClaw Bridge RPC API Reference

> **Protocol:** JSON-RPC 2.0
> **Transport:** Unix Domain Socket
> **Socket:** `/run/armorclaw/bridge.sock`
> **Version:** 1.2.0

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

**API Reference Last Updated:** 2026-02-07
**Compatible with Bridge Version:** 1.0.0
