# Sending Configs via Element X

This guide explains how to send configuration files to ArmorClaw agents using the Element X app.

> **Last Updated:** 2026-02-17
> **Phase:** Priority 3 - End-to-End Flow Documentation

## Overview

ArmorClaw supports receiving configuration files through Matrix messages. This allows you to:

- Send environment files (`.env`)
- Update agent settings (`.toml`, `.yaml`)
- Inject secrets without rebuilding containers
- Configure agents on-the-fly

---

## Architecture Overview

```
┌─────────────────┐     Matrix      ┌─────────────────┐
│   Element X     │ ◄─────────────► │  Matrix Server  │
│   (Mobile/App)  │                 │   (Conduit)     │
└─────────────────┘                 └────────┬────────┘
                                             │
                                             │ Sync
                                             ▼
                                    ┌─────────────────┐
                                    │ ArmorClaw Bridge│
                                    │  (JSON-RPC)     │
                                    └────────┬────────┘
                                             │
                                             │ attach_config
                                             ▼
                                    ┌─────────────────┐
                                    │ /run/armorclaw/ │
                                    │   configs/      │
                                    └────────┬────────┘
                                             │
                                             │ Mount
                                             ▼
                                    ┌─────────────────┐
                                    │ Agent Container │
                                    │  (Python)       │
                                    └─────────────────┘
```

---

## Complete End-to-End Flow

### Step 1: User Sends Message in Element X

User types in Element X:

```
/attach_config agent.env MODEL=gpt-4 TEMPERATURE=0.7
```

### Step 2: Matrix Server Receives Message

The message is sent to the Matrix homeserver (Conduit) as an `m.room.message` event:

```json
{
  "type": "m.room.message",
  "room_id": "!agents:matrix.armorclaw.com",
  "sender": "@admin:matrix.armorclaw.com",
  "content": {
    "msgtype": "m.text",
    "body": "/attach_config agent.env MODEL=gpt-4 TEMPERATURE=0.7"
  }
}
```

### Step 3: Bridge Receives via Matrix Sync

The ArmorClaw Bridge polls the Matrix server via `/sync` and receives the message:

```
Bridge → Matrix Server: GET /_matrix/client/v3/sync?since=<token>
Matrix Server → Bridge: { "rooms": { "join": { "!agents:...": { ... } } } }
```

### Step 4: Bridge Parses Command

The Bridge parses the `/attach_config` command:

1. Validates the config name (`agent.env`) - no path traversal
2. Extracts the content (`MODEL=gpt-4 TEMPERATURE=0.7`)
3. Determines encoding (default: `raw`)

### Step 5: Bridge Writes Config File

The Bridge writes the config to the tmpfs filesystem:

```go
// Pseudocode
configPath := "/run/armorclaw/configs/agent.env"
configContent := "MODEL=gpt-4\nTEMPERATURE=0.7\n"
os.WriteFile(configPath, []byte(configContent), 0644)
```

### Step 6: Bridge Responds via Matrix

The Bridge sends a response message to the Matrix room:

```json
{
  "type": "m.room.message",
  "content": {
    "msgtype": "m.notice",
    "body": "✅ Config attached: agent.env\nConfig ID: config-agent.env-1736294400\nPath: /run/armorclaw/configs/agent.env"
  }
}
```

### Step 7: Agent Receives Config

If the agent container is running, it can access the config:

```python
# Agent reads config
config_path = "/run/armorclaw/configs/agent.env"
with open(config_path) as f:
    for line in f:
        key, value = line.strip().split('=')
        os.environ[key] = value
```

### Sequence Diagram

```
User          Element X      Matrix Server    Bridge         Agent
 │                │               │            │              │
 │  Type command  │               │            │              │
 │───────────────►│               │            │              │
 │                │  Send message │            │              │
 │                │──────────────►│            │              │
 │                │               │  Sync poll │              │
 │                │               │◄───────────│              │
 │                │               │  Events    │              │
 │                │               │───────────►│              │
 │                │               │            │ Parse cmd    │
 │                │               │            │ Write file   │
 │                │               │            │─────────┐    │
 │                │               │            │         │    │
 │                │               │            │◄────────┘    │
 │                │               │            │ Send notice  │
 │                │               │◄───────────│              │
 │                │  Show notice  │            │              │
 │                │◄──────────────│            │              │
 │◄───────────────│               │            │              │
 │  See ✅ Config attached        │            │              │
 │                │               │            │   Read config│
 │                │               │            │              │◄──
```

---

## Quick Start

### Prerequisites

1. ArmorClaw Bridge running with Matrix enabled
2. Element X app connected to your Matrix homeserver
3. Agent running in a container with bridge connection

### Method 1: Using the /attach_config Command

The simplest way to attach configs is via the `/attach_config` command in Element X:

```
/attach_config agent.env MODEL=gpt-4 TEMPERATURE=0.7
```

The agent will respond with:

```json
{
  "config_id": "config-agent.env-1234567890",
  "name": "agent.env",
  "path": "/run/armorclaw/configs/agent.env",
  "size": 25,
  "type": "env"
}
```

### Method 2: Sending File Attachments

For larger configurations, send a file attachment:

1. **Compose a message** in Element X
2. **Attach the file** (e.g., `config.toml`, `.env`)
3. **Send** to the agent's room

The agent will:
- Receive the file via Matrix
- Store it in `/run/armorclaw/configs/`
- Return a config_id for reference

## Configuration Types

### Environment Files (`.env`)

Simple key-value pairs:

```
/attach_config agent.env MODEL=gpt-4
/attach_config agent.env MODEL=gpt-4 TEMPERATURE=0.7 MAX_TOKENS=4096
```

Result: Stored as `/run/armorclaw/configs/agent.env`

### TOML Configuration

For complex settings, use TOML:

```
/attach_config agent.toml [agent]
model=gpt-4
temperature=0.7

[limits]
max_tokens=4096
timeout=30
```

### JSON Configuration

For structured data:

```
/attach_config settings.json {"model":"gpt-4","temperature":0.7}
```

## Advanced Usage

### Base64 Encoding

For binary or special character content, use base64:

```bash
# Encode your config
echo "SECRET=value" | base64

# Send via Element X
/attach_config secret.env U0VDUkVUPXZhbHVlCg== base64
```

### Config Metadata

Attach metadata for tracking:

```
/attach_config config.env MODEL=gpt-4
# Metadata is auto-generated with:
# - Timestamp
# - Sender
# - Config ID
```

## Config Lifecycle

### Storage Location

All configs are stored in: `/run/armorclaw/configs/`

This is a tmpfs-mounted directory, so configs are:
- ✅ Ephemeral (wiped on reboot)
- ✅ In-memory only (never written to disk)
- ✅ Isolated per container

### Config IDs

Each attachment gets a unique ID:

```
config-<name>-<timestamp>
```

Example: `config-agent.env-1736294400`

### Retrieving Configs

To list attached configs, use the bridge RPC:

```bash
echo '{"jsonrpc":"2.0","method":"list_configs","id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

## Security Considerations

### Path Traversal Protection

The bridge validates all config names:

```
❌ /attach_config ../../../etc/passwd malicious
# Error: invalid config name (path traversal not allowed)

❌ /attach_config /absolute/path.env content
# Error: invalid config name (absolute paths not allowed)

✅ /attach_config relative/path.env content
# OK: stored as /run/armorclaw/configs/relative/path.env
```

### Size Limits

Configs are limited to 1MB:

```
❌ <very large config>
# Error: config content too large (max 1 MB)
```

### Encoding Options

| Encoding | Description | Use Case |
|----------|-------------|----------|
| `raw` | Plain text (default) | Most configs |
| `base64` | Base64 encoded | Binary content, special chars |

## Examples

### Example 1: Configure GPT-4 Model

```
/attach_config model.env OPENAI_API_KEY=sk-xxx
/attach_config model.env MODEL=gpt-4
```

### Example 2: Set Temperature

```
/attach_config settings.env TEMPERATURE=0.7 MAX_TOKENS=4096
```

### Example 3: Complex TOML Config

```
/attach_config agent.toml [agent]
model=gpt-4
temperature=0.7

[logging]
level=info
format=json

[matrix]
enabled=true
room_id=!abc123:matrix.example.com
```

### Example 4: Multi-line YAML

```
/attach_config config.yaml agent:
  model: gpt-4
  temperature: 0.7

logging:
  level: info
```

## Troubleshooting

### Diagnostic Checklist

Before troubleshooting, verify each component:

```bash
# 1. Check bridge is running
sudo ./build/armorclaw-bridge status

# 2. Check bridge socket exists
ls -la /run/armorclaw/bridge.sock

# 3. Check Matrix connection
curl http://localhost:8008/_matrix/client/versions

# 4. Check configs directory
ls -la /run/armorclaw/configs/

# 5. Test RPC directly
echo '{"jsonrpc":"2.0","method":"list_configs","id":1}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Common Issues

#### Issue: Config Not Found

```
❌ Error: config file not found at /run/armorclaw/configs/test.env
```

**Possible Causes:**
1. Config was never attached
2. Config was overwritten by subsequent attachment
3. Container restarted (configs are ephemeral)

**Solutions:**
```bash
# Check if config exists
ls -la /run/armorclaw/configs/

# Re-attach the config
/attach_config test.env KEY=VALUE

# Verify with list_configs RPC
echo '{"jsonrpc":"2.0","method":"list_configs","id":1}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

#### Issue: Invalid Encoding

```
❌ Error: failed to decode base64 content
```

**Solutions:**
```bash
# Verify base64 encoding is valid
echo "U0VDUkVUPXZhbHVlCg==" | base64 -d
# Should output: SECRET=value

# Re-encode if needed
echo "SECRET=value" | base64
# Outputs: U0VDUkVUPXZhbHVlCg==
```

#### Issue: Bridge Not Connected

```
❌ Error: Bridge not connected. Cannot attach config.
```

**Solutions:**
```bash
# Check bridge process
ps aux | grep armorclaw-bridge

# Check bridge status
sudo ./build/armorclaw-bridge status

# Start bridge with Matrix enabled
sudo ./build/armorclaw-bridge start --matrix-enabled

# Check bridge logs
sudo journalctl -u armorclaw-bridge -f
```

#### Issue: Matrix Sync Not Working

```
❌ No response in Element X after sending command
```

**Diagnostic Steps:**
```bash
# Check Matrix server is running
curl http://localhost:8008/_matrix/client/versions

# Check bridge Matrix status
echo '{"jsonrpc":"2.0","method":"matrix.status","id":1}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Expected response:
# {"result":{"connected":true,"user_id":"@bridge:matrix.armorclaw.com",...}}
```

**Solutions:**
- Verify Matrix credentials in bridge config
- Check network connectivity to Matrix server
- Restart bridge with fresh Matrix login

#### Issue: Permission Denied

```
❌ Error: permission denied writing config
```

**Solutions:**
```bash
# Check directory permissions
ls -la /run/armorclaw/configs/

# Should be owned by root or bridge user
# Fix if needed:
sudo chown root:root /run/armorclaw/configs/
sudo chmod 755 /run/armorclaw/configs/
```

#### Issue: Path Traversal Blocked

```
❌ Error: invalid config name (path traversal not allowed)
```

**Invalid Examples:**
```
/attach_config ../../../etc/passwd malicious    # ❌ Blocked
/attach_config /absolute/path.env content       # ❌ Blocked
/attach_config ..\windows\path.env content      # ❌ Blocked
```

**Valid Examples:**
```
/attach_config agent.env content                # ✅ OK
/attach_config subdir/config.yaml content       # ✅ OK
/attach_config ./local.env content              # ✅ OK
```

#### Issue: Config Too Large

```
❌ Error: config content too large (max 1 MB)
```

**Solution:**
- Split config into multiple smaller files
- Use base64 encoding for binary content
- Remove comments/whitespace to reduce size

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
# Start bridge with verbose logging
sudo ./build/armorclaw-bridge start --verbose

# Or set in config file:
# [logging]
# level = "debug"
```

Debug logs show:
- Matrix sync events received
- Command parsing details
- Config file operations
- Error stack traces

---

## Example Test Messages

### RPC Test Messages

These can be sent directly to the bridge socket for testing:

```bash
# Connect to bridge
socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

#### Test 1: Simple Env File

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "attach_config",
  "params": {
    "name": "test.env",
    "content": "KEY1=value1\nKEY2=value2\n",
    "encoding": "raw",
    "type": "env"
  }
}
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "config_id": "config-test.env-1736294400",
    "name": "test.env",
    "path": "/run/armorclaw/configs/test.env",
    "size": 22,
    "type": "env"
  }
}
```

#### Test 2: Base64 Encoded Content

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "attach_config",
  "params": {
    "name": "secret.env",
    "content": "U0VDUkVUPXN1cGVyc2VjcmV0dmFsdWUK",
    "encoding": "base64",
    "type": "env"
  }
}
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "config_id": "config-secret.env-1736294401",
    "name": "secret.env",
    "path": "/run/armorclaw/configs/secret.env",
    "size": 24,
    "type": "env"
  }
}
```

#### Test 3: TOML Configuration

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "attach_config",
  "params": {
    "name": "agent.toml",
    "content": "[agent]\nmodel = \"gpt-4\"\ntemperature = 0.7\n\n[logging]\nlevel = \"info\"\n",
    "encoding": "raw",
    "type": "toml"
  }
}
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "config_id": "config-agent.toml-1736294402",
    "name": "agent.toml",
    "path": "/run/armorclaw/configs/agent.toml",
    "size": 68,
    "type": "toml"
  }
}
```

#### Test 4: List Configs

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "list_configs"
}
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "configs": [
      {
        "config_id": "config-test.env-1736294400",
        "name": "test.env",
        "path": "/run/armorclaw/configs/test.env",
        "size": 22,
        "type": "env"
      },
      {
        "config_id": "config-secret.env-1736294401",
        "name": "secret.env",
        "path": "/run/armorclaw/configs/secret.env",
        "size": 24,
        "type": "env"
      }
    ],
    "count": 2
  }
}
```

#### Test 5: Error Cases

**Invalid Path Traversal:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "attach_config",
  "params": {
    "name": "../../../etc/passwd",
    "content": "malicious",
    "encoding": "raw"
  }
}
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "error": {
    "code": -32602,
    "message": "invalid config name: path traversal not allowed"
  }
}
```

**Missing Required Field:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "attach_config",
  "params": {
    "name": "test.env"
  }
}
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "error": {
    "code": -32602,
    "message": "missing required parameter: content"
  }
}
```

### Matrix Test Messages

These messages can be sent in Element X to test the flow:

```
# Test 1: Simple config
/attach_config test.env KEY=VALUE

# Test 2: Multiple values
/attach_config app.env DEBUG=true LOG_LEVEL=info MAX_RETRIES=3

# Test 3: TOML config
/attach_config settings.toml [server]
port=8080
host=0.0.0.0

# Test 4: Base64 encoded
/attach_config secret.env U0VDUkVUPXZhbHVlCg== base64

# Test 5: Status check
/status
```

---

## Testing

### Automated Test Suite

```bash
./tests/test-attach-config.sh
```

This tests:
- ✅ Raw content attachment
- ✅ Base64-encoded content
- ✅ TOML configuration
- ✅ Path traversal protection
- ✅ Parameter validation
- ✅ File system verification

### Manual Testing Checklist

1. **Bridge Running**
   ```bash
   sudo ./build/armorclaw-bridge status
   # Expected: "Bridge is running"
   ```

2. **Matrix Connected**
   ```bash
   echo '{"jsonrpc":"2.0","method":"matrix.status","id":1}' | \
     socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   # Expected: {"result":{"connected":true,...}}
   ```

3. **Send Test Config**
   - Open Element X
   - Navigate to agent room
   - Send: `/attach_config test.env KEY=VALUE`

4. **Verify Response**
   - Should see: `✅ Config attached: test.env`
   - Note the config_id for reference

5. **Verify File Created**
   ```bash
   ls -la /run/armorclaw/configs/test.env
   cat /run/armorclaw/configs/test.env
   # Expected: KEY=VALUE
   ```

6. **Verify Agent Can Read**
   ```bash
   docker exec <container> cat /run/armorclaw/configs/test.env
   # Expected: KEY=VALUE
   ```

---

## Next Steps

After attaching configs:

1. **Inject into containers** - Configs are available at `/run/armorclaw/configs/`
2. **Use envsubst** - Template environment variables into configs
3. **Monitor agent** - Check `/status` for loaded configs

See also:
- [Configuration Guide](configuration.md)
- [Bridge RPC API](../reference/rpc-api.md)
- [Error Catalog](error-catalog.md) for complete error reference
