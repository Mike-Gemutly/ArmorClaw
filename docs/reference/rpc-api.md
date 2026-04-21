# ArmorClaw Bridge RPC API Reference

> **Protocol:** JSON-RPC 2.0
> **Transport:** Unix Domain Socket
> **Socket:** `/run/armorclaw/bridge.sock`
> **Version:** 1.12.0
> **Last Updated:** 2026-02-26

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

## Profile Methods (v1.11.0)

Profile methods for managing encrypted PII (Personally Identifiable Information) profiles. These profiles are used by the Blind Fill capability to securely inject user data into skills with explicit consent.

### profile.create

Create a new encrypted PII profile.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "profile.create",
  "params": {
    "profile_name": "Personal",
    "profile_type": "personal",
    "data": {
      "full_name": "John Doe",
      "email": "john@example.com",
      "phone": "555-1234"
    },
    "is_default": true
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| profile_name | string | ✅ Yes | Human-readable profile name |
| profile_type | string | ❌ No | Profile type: "personal", "business", "payment", "medical", "custom" (default: "personal") |
| data | object | ✅ Yes | PII data fields to store |
| is_default | boolean | ❌ No | Set as default for this profile type (default: false) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "profile_abc123def456",
    "profile_name": "Personal",
    "profile_type": "personal",
    "field_count": 3,
    "is_default": true,
    "created_at": 1738864000
  }
}
```

**Standard Field Keys:**
- `full_name`, `first_name`, `last_name` - Identity
- `email`, `phone` - Contact
- `date_of_birth`, `ssn` - Sensitive identity (requires explicit consent)
- `address`, `city`, `state`, `postal_code`, `country` - Location
- `company`, `job_title` - Business
- Custom fields allowed via `custom` object

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"profile.create","params":{"profile_name":"Personal","data":{"full_name":"John Doe","email":"john@example.com"}}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### profile.list

List all stored profiles (without decrypting PII values).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "profile.list",
  "params": {
    "profile_type": "personal"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| profile_type | string | ❌ No | Filter by profile type |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "profiles": [
      {
        "id": "profile_abc123",
        "profile_name": "Personal",
        "profile_type": "personal",
        "field_count": 5,
        "is_default": true,
        "created_at": 1738864000,
        "updated_at": 1738864000
      }
    ],
    "count": 1
  }
}
```

---

### profile.get

Retrieve a specific profile with decrypted PII values.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "profile.get",
  "params": {
    "id": "profile_abc123def456"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | ✅ Yes | Profile ID to retrieve |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "profile_abc123def456",
    "profile_name": "Personal",
    "profile_type": "personal",
    "data": {
      "full_name": "John Doe",
      "email": "john@example.com",
      "phone": "555-1234"
    },
    "field_schema": {
      "profile_type": "personal",
      "fields": [
        {"key": "full_name", "label": "Full Name", "type": "text", "sensitive": false},
        {"key": "email", "label": "Email Address", "type": "email", "sensitive": false}
      ]
    },
    "is_default": true,
    "created_at": 1738864000,
    "updated_at": 1738864000
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - id parameter required
- `-4` (ProfileNotFound) - Profile not found

---

### profile.update

Update an existing profile.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "profile.update",
  "params": {
    "id": "profile_abc123def456",
    "profile_name": "Personal Updated",
    "data": {
      "full_name": "John Doe",
      "email": "john.doe@example.com",
      "phone": "555-5678"
    },
    "is_default": true
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | ✅ Yes | Profile ID to update |
| profile_name | string | ❌ No | New profile name |
| data | object | ❌ No | Updated PII data (merges with existing) |
| is_default | boolean | ❌ No | Set as default for this profile type |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "profile_abc123def456",
    "profile_name": "Personal Updated",
    "field_count": 3,
    "is_default": true,
    "updated_at": 1738865000
  }
}
```

---

### profile.delete

Delete a profile permanently.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "profile.delete",
  "params": {
    "id": "profile_abc123def456"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | ✅ Yes | Profile ID to delete |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "message": "Profile deleted"
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - id parameter required
- `-4` (ProfileNotFound) - Profile not found

---

## PII Access Methods (v1.11.0)

PII Access methods implement Human-in-the-Loop (HITL) consent for skill access to user PII. Skills must request access and users must approve before PII is injected.

### pii.request_access

Request access to PII fields from a profile. This triggers a Matrix notification for user approval.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "pii.request_access",
  "params": {
    "skill_id": "form-filler",
    "skill_name": "Form Filler",
    "profile_id": "profile_abc123def456",
    "variables": [
      {"key": "full_name", "description": "Your name for the form", "required": true, "sensitivity": "low"},
      {"key": "email", "description": "Your email for notifications", "required": true, "sensitivity": "medium"},
      {"key": "phone", "description": "Optional phone number", "required": false, "sensitivity": "medium"}
    ],
    "room_id": "!abc123:matrix.example.com"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| skill_id | string | ✅ Yes | Unique skill identifier |
| skill_name | string | ✅ Yes | Human-readable skill name |
| profile_id | string | ✅ Yes | Profile to access |
| variables | array | ✅ Yes | Requested PII fields |
| variables[].key | string | ✅ Yes | Field key (e.g., "full_name") |
| variables[].description | string | ✅ Yes | Why this field is needed |
| variables[].required | boolean | ❌ No | Is this field required (default: false) |
| variables[].sensitivity | string | ❌ No | "low", "medium", "high", "critical" |
| room_id | string | ❌ No | Matrix room for consent notification |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "request_id": "req_xyz789abc",
    "skill_id": "form-filler",
    "profile_id": "profile_abc123def456",
    "status": "pending",
    "requested_fields": ["full_name", "email", "phone"],
    "required_fields": ["full_name", "email"],
    "expires_at": 1738864600,
    "message": "Access request created. Use pii.approve_access or pii.reject_access to respond."
  }
}
```

**Sensitivity Levels:**
- `low` - Name, city (minimal risk)
- `medium` - Email, phone (contact info)
- `high` - DOB, address (identity)
- `critical` - SSN, financial (maximum protection)

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"pii.request_access","params":{"skill_id":"form-filler","skill_name":"Form Filler","profile_id":"profile_abc123","variables":[{"key":"full_name","description":"Your name","required":true}]}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### pii.approve_access

Approve a PII access request with specific fields.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "pii.approve_access",
  "params": {
    "request_id": "req_xyz789abc",
    "approved_fields": ["full_name", "email"]
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| request_id | string | ✅ Yes | ID of the access request |
| approved_fields | array | ✅ Yes | List of field keys to approve |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "request_id": "req_xyz789abc",
    "status": "approved",
    "approved_fields": ["full_name", "email"],
    "denied_fields": ["phone"],
    "approved_by": "@user:matrix.example.com",
    "approved_at": 1738864050,
    "resolved_variables": {
      "full_name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

**Note:** All required fields from the original request must be approved, or the approval will fail.

**Error Codes:**
- `-32602` (InvalidParams) - request_id or approved_fields required
- `-5` (RequestNotFound) - Access request not found
- `-6` (RequestExpired) - Request has expired
- `-7` (RequestAlreadyProcessed) - Request already approved/rejected
- `-8` (RequiredFieldMissing) - Not all required fields approved

---

### pii.reject_access

Reject a PII access request.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "pii.reject_access",
  "params": {
    "request_id": "req_xyz789abc",
    "reason": "I don't want to share this information"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| request_id | string | ✅ Yes | ID of the access request |
| reason | string | ❌ No | Reason for rejection |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "request_id": "req_xyz789abc",
    "status": "rejected",
    "rejected_by": "@user:matrix.example.com",
    "rejected_at": 1738864050,
    "reason": "I don't want to share this information"
  }
}
```

---

### pii.list_requests

List pending PII access requests.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "pii.list_requests",
  "params": {
    "status": "pending"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| status | string | ❌ No | Filter by status: "pending", "approved", "rejected", "expired" |
| profile_id | string | ❌ No | Filter by profile ID |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "requests": [
      {
        "id": "req_xyz789abc",
        "skill_id": "form-filler",
        "skill_name": "Form Filler",
        "profile_id": "profile_abc123",
        "status": "pending",
        "requested_fields": ["full_name", "email", "phone"],
        "required_fields": ["full_name", "email"],
        "created_at": "2026-02-21T12:00:00Z",
        "expires_at": "2026-02-21T12:01:00Z"
      }
    ],
    "count": 1
  }
}
```

---

### PII Consent Flow (Matrix Integration)

When a skill requests PII access, users receive a Matrix notification:

```
## 🔐 PII Access Request

**Skill:** Form Filler (`form-filler`)
**Request ID:** `req_xyz789abc`
**Profile:** `profile_abc123`

### Requested Fields

**Required:**
- full_name
- email

**Optional:**
- phone

⏱️ Expires in: 60s

### Actions

To approve all fields:
```
!armorclaw approve req_xyz789abc
```

To approve specific fields:
```
!armorclaw approve req_xyz789abc full_name,email
```

To reject:
```
!armorclaw reject req_xyz789abc [optional reason]
```

---

### PII Access Error Codes

| Code | Message | Description |
|------|---------|-------------|
| -4 | ProfileNotFound | Requested profile does not exist |
| -5 | RequestNotFound | Access request does not exist |
| -6 | RequestExpired | Request has expired (60s timeout) |
| -7 | RequestAlreadyProcessed | Request already approved or rejected |
| -8 | RequiredFieldMissing | Not all required fields were approved |

---

### PII Security Notes

1. **Memory-Only Injection:** PII values are injected via Unix sockets, never written to disk
2. **Never Logged:** Actual PII values are never logged - only field names
3. **Consent Required:** Skills cannot access PII without explicit user approval
4. **Time-Limited:** Access requests expire after 60 seconds
5. **Least Privilege:** Users approve specific fields, not entire profiles
6. **Audit Trail:** All PII access is logged in the audit system

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

## Authentication

The Bridge supports two authentication modes depending on deployment configuration.

### Token Types

| Token | Prefix | Validated By | Use Case |
|-------|--------|--------------|----------|
| Admin token | `aat_` | AdminTokenValidator | Provisioning, setup scripts |
| Matrix access token | (variable) | Matrix homeserver whoami | Mobile app, Element, runtime |

### Admin Token Authentication

Admin tokens (prefix `aat_`) are provisioned during initial setup. They grant elevated access to admin-gated methods. The token is validated locally without contacting the Matrix homeserver.

```bash
# Example: admin token in Authorization header
curl -X POST http://localhost:8080/rpc \
  -H "Authorization: Bearer aat_xxxxxxxxxxxxxxxx" \
  -d '{"jsonrpc":"2.0","id":1,"method":"device.list"}'
```

### Matrix Token Authentication

Standard Matrix access tokens are validated against the homeserver's `/_matrix/client/v3/account/whoami` endpoint. Tokens are cached for 5 minutes to reduce homeserver load.

For admin-gated methods, the Bridge checks the user's power level in the configured admin room. The default admin threshold is power level 50 (Matrix moderator).

### Public Methods

The following methods require no authentication:

| Method | Description |
|--------|-------------|
| `system.health` | Server health status |
| `system.config` | Public configuration for client init |
| `system.info` | Server info and capabilities |
| `system.time` | Server time for clock sync |
| `device.validate` | Basic device ID validation |

Public methods are rate-limited to 10 requests per minute per client.

### Admin-Gated Methods

The following methods require admin token or Matrix admin power level:

| Method | Category |
|--------|----------|
| `device.list` | Device Governance |
| `device.get` | Device Governance |
| `device.approve` | Device Governance |
| `device.reject` | Device Governance |
| `invite.create` | Invite Governance |
| `invite.list` | Invite Governance |
| `invite.revoke` | Invite Governance |
| `invite.validate` | Invite Governance |
| `license.activate` | Licensing |
| `license.deactivate` | Licensing |
| `license.update` | Licensing |
| `sso.configure` | SSO |
| `sso.enable` | SSO |
| `sso.disable` | SSO |
| `admin.users` | Administration |
| `admin.rooms` | Administration |
| `admin.settings` | Administration |
| `security.upgrade_tier` | Security |
| `audit.export` | Audit |
| `config.update` | Configuration |

### Authentication Error

When authentication fails, the Bridge returns a JSON-RPC error:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32001,
    "message": "authentication required"
  }
}
```

| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | No token provided for protected method |
| -32001 | `invalid admin token` | Admin token validation failed |
| -32001 | `invalid token` | Matrix token validation failed |
| -32001 | `admin access required` | Non-admin user called admin-gated method |

### Socket-Level Security

In Native mode, the Unix socket (`/run/armorclaw/bridge.sock`) uses filesystem permissions (0660) for access control. In Sentinel/Cloudflare modes, TLS and network-level controls apply in addition to token authentication.

## Method Summary

The following table lists all registered RPC methods. Methods marked **Admin** require admin token or Matrix admin power level. Methods marked **Public** require no authentication. All other methods require a valid Matrix token.

### System & Health

| Method | Auth | Description |
|--------|------|-------------|
| `health.check` | Any | Bridge health check |
| `system.health` | Public | Public health status |
| `system.config` | Public | Public configuration |
| `system.info` | Public | Server info and capabilities |
| `system.time` | Public | Server time for clock sync |
| `mobile.heartbeat` | Any | Mobile device heartbeat |

### AI & Chat

| Method | Auth | Description |
|--------|------|-------------|
| `ai.chat` | Any | Send message to AI agent |

### Browser Automation

| Method | Auth | Description |
|--------|------|-------------|
| `browser.navigate` | Any | Navigate browser to URL |
| `browser.fill` | Any | Fill form field |
| `browser.click` | Any | Click element |
| `browser.status` | Any | Get browser task status |
| `browser.wait_for_element` | Any | Wait for element to appear |
| `browser.wait_for_captcha` | Any | Wait for CAPTCHA resolution |
| `browser.wait_for_2fa` | Any | Wait for 2FA code entry |
| `browser.complete` | Any | Mark browser task complete |
| `browser.fail` | Any | Mark browser task failed |
| `browser.list` | Any | List browser tasks |
| `browser.cancel` | Any | Cancel browser task |

### Bridge Control

| Method | Auth | Description |
|--------|------|-------------|
| `bridge.start` | Any | Start bridge connection |
| `bridge.stop` | Any | Stop bridge connection |
| `bridge.status` | Any | Get bridge status |
| `bridge.channel` | Any | Bridge a Matrix channel |
| `bridge.unchannel` | Any | Remove a bridged channel |
| `bridge.list` | Any | List bridged channels |
| `bridge.ghost_list` | Any | List ghost users |
| `bridge.appservice_status` | Any | Application service status |

### PII / Human-in-the-Loop

| Method | Auth | Description |
|--------|------|-------------|
| `pii.request` | Any | Request PII access |
| `pii.approve` | Any | Approve PII request |
| `pii.deny` | Any | Deny PII request |
| `pii.status` | Any | Check PII request status |
| `pii.list_pending` | Any | List pending PII requests |
| `pii.stats` | Any | PII system statistics |
| `pii.cancel` | Any | Cancel PII request |
| `pii.fulfill` | Any | Fulfill approved PII request |
| `pii.wait_for_approval` | Any | Wait for PII approval |

### Email Approval

| Method | Auth | Description |
|--------|------|-------------|
| `approve_email` | Any | Approve email sending |
| `deny_email` | Any | Deny email sending |
| `email_approval_status` | Any | Check email approval status |
| `email.list_pending` | Any | List pending email approvals |

### Skills

| Method | Auth | Description |
|--------|------|-------------|
| `skills.execute` | Any | Execute a skill |
| `skills.list` | Any | List available skills |
| `skills.get_schema` | Any | Get skill parameter schema |
| `skills.allow` | Any | Allow a skill |
| `skills.block` | Any | Block a skill |
| `skills.allowlist_add` | Any | Add skill to allowlist |
| `skills.allowlist_remove` | Any | Remove skill from allowlist |
| `skills.allowlist_list` | Any | List allowlist entries |
| `skills.web_search` | Any | Web search skill |
| `skills.web_extract` | Any | Web content extraction |
| `skills.email_send` | Any | Send email skill |
| `skills.slack_message` | Any | Send Slack message skill |
| `skills.file_read` | Any | Read file skill |
| `skills.data_analyze` | Any | Data analysis skill |

### Matrix

| Method | Auth | Description |
|--------|------|-------------|
| `matrix.status` | Any | Matrix connection status |
| `matrix.login` | Any | Login to Matrix |
| `matrix.send` | Any | Send Matrix message |
| `matrix.receive` | Any | Receive Matrix events |
| `matrix.join_room` | Any | Join a Matrix room |

### Events

| Method | Auth | Description |
|--------|------|-------------|
| `events.replay` | Any | Replay events |
| `events.stream` | Any | Stream events |

### Studio

| Method | Auth | Description |
|--------|------|-------------|
| `studio.deploy` | Any | Deploy agent via Studio |
| `studio.stats` | Any | Studio statistics |

### Containers

| Method | Auth | Description |
|--------|------|-------------|
| `container.terminate` | Any | Terminate a container |
| `container.list` | Any | List running containers |

### Provisioning

| Method | Auth | Description |
|--------|------|-------------|
| `provisioning.start` | Any | Start device provisioning |
| `provisioning.claim` | Any | Claim provisioned device |

### Hardening

| Method | Auth | Description |
|--------|------|-------------|
| `hardening.status` | Any | Security hardening status |
| `hardening.ack` | Any | Acknowledge hardening notice |
| `hardening.rotate_password` | Any | Rotate admin password |

### Secretary / Tasks

| Method | Auth | Description |
|--------|------|-------------|
| `secretary.start_workflow` | Any | Start a workflow |
| `secretary.get_workflow` | Any | Get workflow status |
| `secretary.cancel_workflow` | Any | Cancel running workflow |
| `secretary.advance_workflow` | Any | Advance workflow step |
| `secretary.list_templates` | Any | List workflow templates |
| `secretary.create_template` | Any | Create workflow template |
| `secretary.get_template` | Any | Get workflow template |
| `secretary.delete_template` | Any | Delete workflow template |
| `secretary.update_template` | Any | Update workflow template |
| `task.create` | Any | Create a task |
| `task.list` | Any | List tasks |
| `task.cancel` | Any | Cancel a task |
| `task.get` | Any | Get task status |

### Account

| Method | Auth | Description |
|--------|------|-------------|
| `account.delete` | Any | Delete user account |

### Keystore

| Method | Auth | Description |
|--------|------|-------------|
| `store_key` | Any | Store API key in encrypted keystore |

### Other

| Method | Auth | Description |
|--------|------|-------------|
| `resolve_blocker` | Any | Resolve a task blocker |

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

## Agent Status Methods (Mobile Secretary)

These methods manage agent state machines for Mobile Secretary workflows.

### Agent Status Values

The agent status follows a state machine with validated transitions:

| Status | Description | Terminal | Needs Action |
|--------|-------------|----------|--------------|
| `OFFLINE` | Agent not reachable | ✅ Yes | ❌ No |
| `INITIALIZING` | Agent starting up | ❌ No | ❌ No |
| `IDLE` | Agent ready, no task | ❌ No | ❌ No |
| `BROWSING` | Navigating to URL | ❌ No | ❌ No |
| `FORM_FILLING` | Filling form fields | ❌ No | ❌ No |
| `AWAITING_CAPTCHA` | Needs human CAPTCHA solving | ✅ Yes | ✅ Yes |
| `AWAITING_2FA` | Needs 2FA code | ✅ Yes | ✅ Yes |
| `AWAITING_APPROVAL` | Waiting for BlindFill approval | ✅ Yes | ✅ Yes |
| `PROCESSING_PAYMENT` | Submitting payment | ❌ No | ❌ No |
| `ERROR` | Recoverable error | ❌ No | ❌ No |
| `COMPLETE` | Task finished | ❌ No | ❌ No |

### agent.set_status

Update agent status with state machine validation.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "agent.set_status",
  "params": {
    "agent_id": "secretary-001",
    "status": "BROWSING",
    "metadata": {
      "url": "https://example.com/checkout",
      "step": "1/5",
      "progress": 20,
      "task_id": "task-abc123",
      "task_type": "flight_booking"
    }
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |
| status | string | ✅ Yes | New status (must be valid transition from current) |
| metadata | object | ❌ No | Additional context (url, step, progress, etc.) |
| force | boolean | ❌ No | Skip transition validation (for recovery) |

**Metadata Fields:**
| Field | Type | Description |
|-------|------|-------------|
| url | string | Current page URL (for BROWSING) |
| step | string | Current step description (e.g., "2/5") |
| progress | number | Progress percentage (0-100) |
| error | string | Error message (for ERROR status) |
| task_id | string | Current task identifier |
| task_type | string | Task type (e.g., "flight_booking") |
| fields_requested | array | PII fields being requested (for AWAITING_APPROVAL) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "agent_id": "secretary-001",
    "status": "BROWSING",
    "previous": "INITIALIZING",
    "timestamp": 1739491200000,
    "is_terminal": false,
    "needs_action": false
  }
}
```

**Error Codes:**
- `-32602` (InvalidParams) - agent_id or status is required
- `-32602` (InvalidParams) - invalid state transition

### agent.get_status

Get current agent status and metadata.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "agent.get_status",
  "params": {
    "agent_id": "secretary-001"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "agent_id": "secretary-001",
    "status": "BROWSING",
    "metadata": {
      "url": "https://example.com/checkout",
      "step": "1/5",
      "progress": 20
    },
    "is_terminal": false,
    "is_active": true,
    "needs_action": false,
    "string": "BROWSING (https://example.com/checkout)"
  }
}
```

### agent.status_history

Get status change history for an agent.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "agent.status_history",
  "params": {
    "agent_id": "secretary-001",
    "limit": 10
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |
| limit | number | ❌ No | Max history entries (default: 50) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "agent_id": "secretary-001",
    "history": [
      {
        "agent_id": "secretary-001",
        "status": "BROWSING",
        "previous": "INITIALIZING",
        "metadata": {"url": "https://example.com"},
        "timestamp": 1739491200000
      }
    ],
    "count": 1
  }
}
```

### agent.subscribe_status

Long-polling subscription for status changes. Returns immediately if status differs from `current_status`, or waits up to `timeout_ms` for a change.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "agent.subscribe_status",
  "params": {
    "agent_id": "secretary-001",
    "current_status": "BROWSING",
    "timeout_ms": 30000
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |
| current_status | string | ❌ No | Only return if status differs |
| timeout_ms | number | ❌ No | Max wait time (default: 30000, max: 60000) |

**Response (Status Changed):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "agent_id": "secretary-001",
    "status": "FORM_FILLING",
    "previous": "BROWSING",
    "metadata": {"step": "1/5"},
    "timestamp": 1739491210000,
    "is_terminal": false,
    "needs_action": false
  }
}
```

**Response (Timeout):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "agent_id": "secretary-001",
    "status": "BROWSING",
    "timeout": true,
    "changed": false,
    "timestamp": 1739491240000
  }
}
```

---

## Sealed Keystore Methods (Mobile Secretary)

These methods manage the sealed keystore for Mobile Secretary workflows. The sealed keystore provides an additional security layer requiring explicit unsealing before sensitive PII operations.

### keystore.unseal_request

Request unsealing of the keystore for an agent. Creates a pending request that must be approved for `mobile_approval` policy.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.unseal_request",
  "params": {
    "agent_id": "secretary-001",
    "reason": "Flight booking for John Doe",
    "fields": ["name", "email", "phone", "credit_card"],
    "task_id": "task-abc123"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |
| reason | string | ✅ Yes | Human-readable reason for unseal request |
| fields | array | ❌ No | PII fields being requested |
| task_id | string | ❌ No | Associated task ID |

**Response (Pending Approval):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "request_id": "req_abc123",
    "agent_id": "secretary-001",
    "reason": "Flight booking for John Doe",
    "fields": ["name", "email", "phone", "credit_card"],
    "requested_at": 1739491200000,
    "expires_at": 1739491260000,
    "policy": "mobile_approval"
  }
}
```

**Response (Already Unsealed):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "already_unsealed": true,
    "session_id": "sess_xyz789",
    "expires_at": 1739491500000
  }
}
```

### keystore.unseal_approve

Approve a pending unseal request (called from mobile device).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.unseal_approve",
  "params": {
    "request_id": "req_abc123",
    "approved_by": "@user:matrix.example.com",
    "device_id": "DEVICEABC123"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| request_id | string | ✅ Yes | Pending request ID to approve |
| approved_by | string | ✅ Yes | Matrix user ID of approver |
| device_id | string | ❌ No | Mobile device ID that approved |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "session_id": "sess_xyz789",
    "agent_id": "secretary-001",
    "unsealed_at": 1739491200,
    "expires_at": 1739491500,
    "policy": "mobile_approval"
  }
}
```

### keystore.unseal_reject

Reject a pending unseal request.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.unseal_reject",
  "params": {
    "request_id": "req_abc123",
    "rejected_by": "@user:matrix.example.com",
    "reason": "Unknown request"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| request_id | string | ✅ Yes | Pending request ID to reject |
| rejected_by | string | ❌ No | Matrix user ID of rejecter |
| reason | string | ❌ No | Reason for rejection |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "request_id": "req_abc123",
    "rejected": true
  }
}
```

### keystore.unseal_status

Get the current sealed/unsealed status for an agent.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.unseal_status",
  "params": {
    "agent_id": "secretary-001"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |

**Response (Sealed):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "is_sealed": true
  }
}
```

**Response (Unsealed):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "is_sealed": false,
    "session_id": "sess_xyz789",
    "agent_id": "secretary-001",
    "expires_at": 1739491500,
    "unseal_policy": "mobile_approval"
  }
}
```

### keystore.unseal_pending

Get all pending unseal requests (for mobile app notification list).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.unseal_pending",
  "params": {
    "agent_id": "secretary-001"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ❌ No | Filter by agent (omit for all) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "requests": [
      {
        "request_id": "req_abc123",
        "agent_id": "secretary-001",
        "reason": "Flight booking",
        "fields": ["name", "email"],
        "task_id": "task-abc123",
        "requested_at": 1739491200000,
        "expires_at": 1739491260000
      }
    ],
    "count": 1
  }
}
```

### keystore.seal

Explicitly seal the keystore for an agent (revoke access).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.seal",
  "params": {
    "agent_id": "secretary-001"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "agent_id": "secretary-001",
    "sealed": true
  }
}
```

---

## Challenge-Response Protocol Methods (Zero-Trust Keystore)

These methods implement the challenge-response protocol for zero-trust keystore unsealing. The mobile device must prove possession of the private key before the keystore can be unsealed.

### challenge.generate

Generate a new challenge for an unseal request. The challenge contains a nonce that must be signed by the mobile device's Ed25519 private key.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.generate",
  "params": {
    "agent_id": "secretary-001",
    "reason": "Flight booking requires PII access",
    "fields": ["name", "email", "phone"]
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier requesting unseal |
| reason | string | ✅ Yes | Human-readable reason for the unseal request |
| fields | array | ❌ No | List of PII fields being requested (for user display) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "type": "com.armorclaw.keystore.unseal_challenge",
    "challenge_id": "chal_abc123def456",
    "nonce": "base64-encoded-32-byte-nonce",
    "server_public_key": "base64-encoded-ed25519-public-key",
    "agent_id": "secretary-001",
    "reason": "Flight booking requires PII access",
    "fields": ["name", "email", "phone"],
    "expires_at": 1739491260000
  }
}
```

### challenge.verify

Verify a challenge response with Ed25519 signature. The mobile device signs the SHA-256 hash of (nonce || timestamp || device_id).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.verify",
  "params": {
    "challenge_id": "chal_abc123def456",
    "signature": "base64-encoded-ed25519-signature",
    "public_key": "base64-encoded-ed25519-public-key",
    "timestamp": 1739491200,
    "device_id": "device-001"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| challenge_id | string | ✅ Yes | ID of the challenge to verify |
| signature | string | ✅ Yes | Base64-encoded Ed25519 signature |
| public_key | string | ✅ Yes | Base64-encoded Ed25519 public key |
| timestamp | integer | ✅ Yes | Unix timestamp when response was created |
| device_id | string | ❌ No | Device identifier for audit logging |
| wrapped_kek | string | ❌ No | Optional wrapped key encryption key |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "challenge_id": "chal_abc123def456",
    "agent_id": "secretary-001",
    "reason": "Flight booking requires PII access",
    "fields": ["name", "email", "phone"],
    "device_id": "device-001"
  }
}
```

### challenge.get

Get a specific challenge by ID.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.get",
  "params": {
    "challenge_id": "chal_abc123def456"
  }
}
```

**Response:** Same format as `challenge.generate` result.

### challenge.list_pending

List all pending challenges, optionally filtered by agent.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.list_pending",
  "params": {
    "agent_id": "secretary-001"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ❌ No | Filter by agent ID (omit for all agents) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "challenges": [
      {
        "type": "com.armorclaw.keystore.unseal_challenge",
        "challenge_id": "chal_abc123",
        "nonce": "...",
        "server_public_key": "...",
        "agent_id": "secretary-001",
        "reason": "...",
        "fields": ["name", "email"],
        "expires_at": 1739491260000
      }
    ],
    "count": 1
  }
}
```

### challenge.register_device

Register a device's Ed25519 public key for pre-verification.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.register_device",
  "params": {
    "device_id": "device-001",
    "public_key": "base64-encoded-ed25519-public-key"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| device_id | string | ✅ Yes | Unique device identifier |
| public_key | string | ✅ Yes | Base64-encoded Ed25519 public key (32 bytes) |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "device_id": "device-001"
  }
}
```

### challenge.unregister_device

Unregister a device's public key.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.unregister_device",
  "params": {
    "device_id": "device-001"
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
    "device_id": "device-001"
  }
}
```

### challenge.server_public_key

Get the server's Ed25519 public key for signature verification on the client side.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.server_public_key"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "public_key": "base64-encoded-ed25519-public-key"
  }
}
```

### challenge.stats

Get challenge manager statistics.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "challenge.stats"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "pending_challenges": 5,
    "registered_devices": 3,
    "ttl_seconds": 60
  }
}
```

---

## Challenge-Based Unseal Methods (Zero-Trust)

These methods combine the challenge-response protocol with sealed keystore unsealing. They are used when the unseal policy is set to `challenge`, requiring Ed25519 signature verification instead of simple approval.

### keystore.unseal_challenge

Request an unseal with a cryptographic challenge. Returns a challenge that must be signed by the mobile device's Ed25519 private key.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.unseal_challenge",
  "params": {
    "agent_id": "secretary-001",
    "reason": "Flight booking requires PII access",
    "fields": ["name", "email", "phone"],
    "task_id": "task-abc123"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agent_id | string | ✅ Yes | Unique agent identifier |
| reason | string | ✅ Yes | Human-readable reason for unseal |
| fields | array | ❌ No | PII fields being requested |
| task_id | string | ❌ No | Associated task ID |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "request_id": "req_xyz789",
    "challenge_id": "chal_abc123",
    "nonce": "base64-encoded-32-byte-nonce",
    "server_public_key": "base64-encoded-ed25519-public-key",
    "agent_id": "secretary-001",
    "reason": "Flight booking requires PII access",
    "fields": ["name", "email", "phone"],
    "expires_at": 1739491260000
  }
}
```

### keystore.unseal_respond

Verify a challenge response and complete the unseal operation.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "keystore.unseal_respond",
  "params": {
    "request_id": "req_xyz789",
    "signature": "base64-encoded-ed25519-signature",
    "public_key": "base64-encoded-ed25519-public-key",
    "timestamp": 1739491200,
    "device_id": "device-001"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| request_id | string | ✅ Yes | ID from unseal_challenge response |
| signature | string | ✅ Yes | Base64-encoded Ed25519 signature of SHA256(nonce \|\| timestamp \|\| device_id) |
| public_key | string | ✅ Yes | Base64-encoded Ed25519 public key (32 bytes) |
| timestamp | integer | ✅ Yes | Unix timestamp when signature was created |
| device_id | string | ❌ No | Device identifier for audit logging |
| wrapped_kek | string | ❌ No | Optional wrapped key encryption key |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true,
    "session_id": "sess_def456",
    "agent_id": "secretary-001",
    "unsealed_at": 1739491200,
    "expires_at": 1739491500,
    "policy": "challenge",
    "device_id": "device-001"
  }
}
```

**Signature Generation:**
The mobile device must sign `SHA256(nonce || timestamp || device_id)` where:
- `nonce` is the raw 32-byte nonce from the challenge
- `timestamp` is the Unix timestamp as a string
- `device_id` is the device identifier string

---

## Device Governance

Device governance methods manage registered devices and their trust states. All device governance methods require admin authentication (admin token or Matrix admin power level).

### Trust States

| State | Description |
|-------|-------------|
| `unverified` | Device connected but not verified |
| `pending_approval` | Awaiting admin approval |
| `awaiting_second_factor` | Waiting for existing device confirmation |
| `verified` | Device is trusted |
| `rejected` | Device was rejected by admin |
| `expired` | Verification request expired |

### Governance Events

Device mutations emit Matrix custom events to the governance room. These are outbound only and never added to the sync filter.

| Event Type | Trigger |
|------------|---------|
| `app.armorclaw.device.approved` | Device trust state set to `verified` |
| `app.armorclaw.device.rejected` | Device trust state set to `rejected` |

Event delivery is best effort. Failures are logged but do not fail the RPC handler.

---

### device.list

List all registered devices.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "device.list"
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
      "id": "dev_abc123",
      "name": "Pixel 8 Pro",
      "type": "mobile",
      "platform": "android",
      "trust_state": "verified",
      "last_seen": "2026-04-19T15:30:00Z",
      "first_seen": "2026-03-10T09:00:00Z",
      "ip_address": "192.168.1.42",
      "user_agent": "ArmorChat/1.12.0",
      "is_current": false,
      "verified_at": "2026-03-10T09:05:00Z",
      "created_at": "2026-03-10T09:00:00Z",
      "updated_at": "2026-04-19T15:30:00Z"
    }
  ]
}
```

Returns an empty array (`[]`) when no devices are registered.

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32603 | `device store not configured` | Device store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"device.list"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### device.get

Get a single device by ID.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "device.get",
  "params": {
    "device_id": "dev_abc123"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| device_id | string | Yes | Device ID to retrieve |

**Response:** Single `DeviceRecord` object (same shape as items in `device.list`).

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32602 | `device_id is required` | Missing device_id parameter |
| -32602 | `invalid parameters` | Malformed JSON params |
| -32000 | `device not found` | No device with given ID |
| -32603 | `device store not configured` | Device store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"device.get","params":{"device_id":"dev_abc123"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### device.approve

Approve a device, setting its trust state to `verified`. Idempotent: approving an already verified device returns success.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "device.approve",
  "params": {
    "device_id": "dev_abc123",
    "approved_by": "@admin:matrix.example.com"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| device_id | string | Yes | Device ID to approve |
| approved_by | string | Yes | Matrix user ID or admin identifier performing the approval |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true
  }
}
```

**Side effects:**
- Audit log entry (`device.approved`) written
- Matrix event `app.armorclaw.device.approved` emitted to governance room

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32602 | `device_id is required` | Missing device_id parameter |
| -32602 | `approved_by is required` | Missing approved_by parameter |
| -32602 | `invalid parameters` | Malformed JSON params |
| -32000 | `device not found` | No device with given ID |
| -32603 | `device store not configured` | Device store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"device.approve","params":{"device_id":"dev_abc123","approved_by":"@admin:example.com"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### device.reject

Reject a device, setting its trust state to `rejected`. Idempotent: rejecting an already rejected device returns success.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "device.reject",
  "params": {
    "device_id": "dev_abc123",
    "rejected_by": "@admin:matrix.example.com",
    "reason": "Unknown device"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| device_id | string | Yes | Device ID to reject |
| rejected_by | string | Yes | Matrix user ID or admin identifier performing the rejection |
| reason | string | No | Human-readable reason for rejection |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true
  }
}
```

**Side effects:**
- Audit log entry (`device.rejected`) written with reason
- Matrix event `app.armorclaw.device.rejected` emitted to governance room

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32602 | `device_id is required` | Missing device_id parameter |
| -32602 | `invalid parameters` | Malformed JSON params |
| -32000 | `device not found` | No device with given ID |
| -32603 | `device store not configured` | Device store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"device.reject","params":{"device_id":"dev_abc123","rejected_by":"@admin:example.com","reason":"Unknown device"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Invite Governance

Invite governance methods manage role-based invitations for onboarding new users. All invite governance methods require admin authentication.

### Roles

| Role | Matrix Power Level | Permissions |
|------|-------------------|-------------|
| `admin` | 100 | Full administrative access including security configuration |
| `moderator` | 50 | Agent management, no security changes |
| `user` | 0 | Standard user access to interact with agents |

### Invite Statuses

| Status | Description |
|--------|-------------|
| `active` | Invite is valid and can be used |
| `used` | Invite has been redeemed |
| `expired` | Invite passed its expiration time |
| `revoked` | Invite was manually revoked by admin |
| `exhausted` | Invite reached its max use count |

### Expiration Values

Invite creation accepts these human-friendly expiration strings:

| Value | Meaning |
|-------|---------|
| `1h` | Expires in 1 hour |
| `6h` | Expires in 6 hours |
| `1d` | Expires in 24 hours |
| `7d` | Expires in 7 days |
| `30d` | Expires in 30 days |
| `never` | No expiration |

### Governance Events

Invite mutations emit Matrix custom events to the governance room.

| Event Type | Trigger |
|------------|---------|
| `app.armorclaw.invite.created` | New invite created |
| `app.armorclaw.invite.revoked` | Invite revoked by admin |

---

### invite.create

Create a new invite with a cryptographically random code.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "invite.create",
  "params": {
    "role": "user",
    "expiration": "7d",
    "max_uses": 5,
    "welcome_message": "Welcome to the team!",
    "created_by": "@admin:matrix.example.com"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| role | string | Yes | One of: `admin`, `moderator`, `user` |
| expiration | string | Yes | Expiration duration: `1h`, `6h`, `1d`, `7d`, `30d`, `never` |
| max_uses | integer | No | Maximum times invite can be used (0 = unlimited) |
| welcome_message | string | No | Message shown to user on invite redemption |
| created_by | string | Yes | Matrix user ID or admin identifier creating the invite |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "inv_abc123def456",
    "code": "Xk9mP2qR",
    "role": "user",
    "created_by": "@admin:matrix.example.com",
    "created_at": "2026-04-20T10:00:00Z",
    "expires_at": "2026-04-27T10:00:00Z",
    "max_uses": 5,
    "use_count": 0,
    "status": "active",
    "welcome_message": "Welcome to the team!"
  }
}
```

**Side effects:**
- Audit log entry (`invite.created`) written
- Matrix event `app.armorclaw.invite.created` emitted to governance room

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32602 | `role is required` | Missing role parameter |
| -32602 | `invalid role: <role>` | Role not one of admin/moderator/user |
| -32602 | `expiration is required` | Missing expiration parameter |
| -32602 | `invalid expiration: <val>` | Unsupported expiration string |
| -32602 | `created_by is required` | Missing created_by parameter |
| -32603 | `invite store not configured` | Invite store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"invite.create","params":{"role":"user","expiration":"7d","max_uses":3,"created_by":"@admin:example.com"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### invite.list

List all invites ordered by creation date descending.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "invite.list"
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
      "id": "inv_abc123def456",
      "code": "Xk9mP2qR",
      "role": "user",
      "created_by": "@admin:matrix.example.com",
      "created_at": "2026-04-20T10:00:00Z",
      "expires_at": "2026-04-27T10:00:00Z",
      "max_uses": 5,
      "use_count": 2,
      "status": "active",
      "welcome_message": "Welcome to the team!"
    }
  ]
}
```

Returns an empty array (`[]`) when no invites exist.

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32603 | `invite store not configured` | Invite store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"invite.list"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### invite.revoke

Revoke an active invite. Idempotent: revoking an already revoked invite returns success.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "invite.revoke",
  "params": {
    "invite_id": "inv_abc123def456",
    "revoked_by": "@admin:matrix.example.com"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| invite_id | string | Yes | Invite ID to revoke |
| revoked_by | string | Yes | Matrix user ID or admin identifier performing the revocation |

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "success": true
  }
}
```

**Side effects:**
- Audit log entry (`invite.revoked`) written
- Matrix event `app.armorclaw.invite.revoked` emitted to governance room

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32602 | `invite_id is required` | Missing invite_id parameter |
| -32602 | `invalid parameters` | Malformed JSON params |
| -32000 | `invite not found` | No invite with given ID |
| -32603 | `invite store not configured` | Invite store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"invite.revoke","params":{"invite_id":"inv_abc123","revoked_by":"@admin:example.com"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### invite.validate

Validate an invite code and return the full invite record. Checks that the invite is active, not expired, and not exhausted. This method is used during onboarding before accepting an invite.

**Authentication:** Admin required

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "invite.validate",
  "params": {
    "code": "Xk9mP2qR"
  }
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| code | string | Yes | Invite code to validate |

**Response:** Full `InviteRecord` object (same shape as items in `invite.list`).

**Error Codes:**
| Code | Message | Cause |
|------|---------|-------|
| -32001 | `authentication required` | Missing or invalid admin credentials |
| -32602 | `code is required` | Missing code parameter |
| -32602 | `invalid parameters` | Malformed JSON params |
| -32000 | `invite not found` | No invite with given code |
| -32000 | `invite is revoked` | Invite has been revoked |
| -32000 | `invite has expired` | Invite past its expiration time |
| -32000 | `invite usage limit reached` | Invite use_count >= max_uses |
| -32603 | `invite store not configured` | Invite store not initialized |

**Example:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"invite.validate","params":{"code":"Xk9mP2qR"}}' | \
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

### PII Errors (PII-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| PII-001 | Error | profile not found | Verify profile ID exists with profile.list |
| PII-002 | Error | profile creation failed | Check profile data format and required fields |
| PII-003 | Warning | field not found in profile | Requested field does not exist in profile |
| PII-010 | Error | access request not found | Request may have expired or never existed |
| PII-011 | Warning | access request expired | Request timed out (60s); create new request |
| PII-012 | Error | access request already processed | Request was already approved or rejected |
| PII-013 | Error | required field not approved | All required fields must be approved |
| PII-020 | Critical | PII injection failed | Check container status and socket availability |
| PII-021 | Warning | PII resolution expired | Resolved variables expired; create new request |
| PII-030 | Critical | keystore decryption failed | Profile data cannot be decrypted |

### Voice/WebRTC Errors (VOX-XXX)

| Code | Severity | Message | Help |
|------|----------|---------|------|
| VOX-001 | Error | WebRTC connection failed | Check ICE/TURN configuration and network connectivity |
| VOX-002 | Error | audio capture failed | Check microphone permissions and device availability |
| VOX-003 | Error | audio playback failed | Check speaker configuration and permissions |

---

**API Reference Last Updated:** 2026-02-26
**Compatible with Bridge Version:** 1.11.0
