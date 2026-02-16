# ArmorClaw Error Message Catalog

> **Last Updated:** 2026-02-15
> **Purpose:** Enable LLMs and users to work backwards from any error to a solution
> **How to Use:** Search this page for the exact error message text or error code

---

## Structured Error Codes (v1.7.0)

ArmorClaw uses structured error codes for programmatic error handling. Each error has:
- **Code**: `CAT-NNN` format (e.g., `CTX-001`)
- **Category**: container, matrix, rpc, system, budget, voice
- **Severity**: debug, info, warning, error, critical
- **Trace ID**: Unique identifier for the error instance

### Querying Errors via RPC

```bash
# Get all unresolved errors
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Get specific error code
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors","params":{"code":"CTX-001"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Resolve an error
echo '{"jsonrpc":"2.0","id":1,"method":"resolve_error","params":{"trace_id":"tr_abc123"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Error Code Quick Reference

| Prefix | Category | Examples |
|--------|----------|----------|
| CTX-XXX | Container | CTX-001 (start failed), CTX-010 (permission denied) |
| MAT-XXX | Matrix | MAT-001 (connection), MAT-002 (auth), MAT-021 (send failed) |
| RPC-XXX | RPC/API | RPC-001 (invalid request), RPC-010 (socket failed) |
| SYS-XXX | System | SYS-001 (keystore), SYS-010 (secret injection) |
| BGT-XXX | Budget | BGT-001 (warning), BGT-002 (exceeded) |
| VOX-XXX | Voice | VOX-001 (WebRTC), VOX-002 (audio capture) |

### LLM-Friendly Error Trace Format

When an error occurs, the system generates a trace that can be copied to an LLM:

```
[ArmorClaw Error Trace]
Code: CTX-001
Category: container
Severity: error
Trace ID: tr_abc123def456
Function: StartContainer
Timestamp: 2026-02-15T12:00:00Z

Message: container start failed
Help: Check Docker daemon status, image availability, and resource limits

Inputs:
  container_id: abc123
  image: armorclaw/agent:v1

State:
  status: exited
  exit_code: 1

Cause: OCI runtime error: container_linux.go:380: starting container process caused: exec: "python": executable file not found in $PATH

Component Events:
  [docker] start - FAILED at 2026-02-15T12:00:00Z
[/ArmorClaw Error Trace]
```

---

## Quick Index

| Error Category | Quick Link |
|---------------|------------|
| **Configuration Errors** | [Jump to Config](#configuration-errors) |
| **Docker Errors** | [Jump to Docker](#docker-errors) |
| **Keystore Errors** | [Jump to Keystore](#keystore-errors) |
| **Container Errors** | [Jump to Container](#container-errors) |
| **RPC/Bridge Errors** | [Jump to RPC](#rpcbridge-errors) |
| **Matrix Errors** | [Jump to Matrix](#matrix-errors) |
| **CLI Errors** | [Jump to CLI](#cli-errors) |

---

## Configuration Errors

### `Error: --provider is required (openai, anthropic, openrouter, google, xai)`

**When:** Running `armorclaw-bridge add-key` without specifying a provider

**Solution:**
```bash
# Specify the provider flag
./build/armorclaw-bridge add-key --provider openai --token sk-xxx
```

---

### `Error: --token is required or set ARMORCLAW_API_KEY environment variable`

**When:** Running `armorclaw-bridge add-key` without providing an API token

**Solution:**
```bash
# Option 1: Provide token via flag
./build/armorclaw-bridge add-key --provider openai --token sk-xxx

# Option 2: Use environment variable
export ARMORCLAW_API_KEY="sk-xxx"
./build/armorclaw-bridge add-key --provider openai
```

---

### `Error: --key is required. Use 'list-keys' to see available keys.`

**When:** Running `armorclaw-bridge start` without specifying which key to use

**Solution:**
```bash
# First, list your available keys
./build/armorclaw-bridge list-keys

# Then start with a specific key
./build/armorclaw-bridge start --key openai-default
```

---

### `Configuration validation failed: server.socket_path is required`

**When:** Config file is missing the `socket_path` field

**Solution:**
```bash
# Reinitialize config with defaults
./build/armorclaw-bridge init

# Or manually add to config.toml:
[server]
  socket_path = "/run/armorclaw/bridge.sock"
```

---

### `Configuration validation failed: keystore.db_path is required`

**When:** Config file is missing the `db_path` field

**Solution:**
```bash
# Reinitialize config with defaults
./build/armorclaw-bridge init

# Or manually add to config.toml:
[keystore]
  db_path = "~/.armorclaw/keystore.db"
```

---

### `Configuration validation failed: matrix.homeserver_url is required when matrix is enabled`

**When:** Matrix is enabled but homeserver URL is not configured

**Solution:**
```bash
# Either disable Matrix in config.toml:
[matrix]
  enabled = false

# Or provide homeserver URL:
[matrix]
  enabled = true
  homeserver_url = "https://matrix.example.com"
```

---

## Docker Errors

### `Docker is not available or not running. Please install and start Docker first.`

**When:** Docker daemon is not running or not installed

**Solution:**
```bash
# Start Docker Desktop (macOS/Windows)
open -a Docker

# Or start Docker daemon (Linux)
sudo systemctl start docker

# Verify Docker is running
docker ps
```

---

### `Error: container creation failed`

**When:** Docker container creation fails for various reasons

**Solution:**
```bash
# 1. Verify image exists
docker images | grep armorclaw

# 2. Pull image if missing
docker pull armorclaw/agent:v1

# 3. Check Docker daemon is running
docker ps

# 4. Run with debug logging
./build/armorclaw-bridge -log-level=debug
```

---

### `container create failed: ...`

**When:** Specific container creation error from Docker API

**Common causes:**
- Invalid image name
- Insufficient disk space
- Network issues
- Resource constraints

**Solution:**
```bash
# Check Docker disk space
docker system df

# Clean up if needed
docker system prune -a

# Verify image
docker inspect armorclaw/agent:v1

# Check resource availability
docker info
```

---

## Keystore Errors

### `Failed to initialize keystore: ...`

**When:** Keystore database cannot be created or opened

**Solution:**
```bash
# 1. Ensure directory exists and is writable
mkdir -p ~/.armorclaw
chmod 700 ~/.armorclaw

# 2. Check disk space
df -h ~/.armorclaw

# 3. Remove corrupted keystore (WARNING: deletes all stored keys)
rm ~/.armorclaw/keystore.db

# 4. Reinitialize
./build/armorclaw-bridge init
```

---

### `Failed to open keystore: ...`

**When:** Keystore database exists but cannot be opened

**Common causes:**
- Database is locked by another process
- File permissions issue
- Database corruption

**Solution:**
```bash
# 1. Check if another bridge instance is running
ps aux | grep armorclaw-bridge

# 2. Kill existing instance if needed
pkill -f armorclaw-bridge

# 3. Check file permissions
ls -la ~/.armorclaw/keystore.db
chmod 600 ~/.armorclaw/keystore.db

# 4. If corrupted, reinitialize
rm ~/.armorclaw/keystore.db
./build/armorclaw-bridge init
```

---

### `key not found`

**When:** Attempting to use a key ID that doesn't exist in the keystore

**Solution:**
```bash
# 1. List available keys
./build/armorclaw-bridge list-keys

# 2. Use a valid key ID
./build/armorclaw-bridge start --key <actual-key-id>

# 3. Or add the key if missing
./build/armorclaw-bridge add-key --provider openai --token sk-xxx
```

---

### `Key 'xxx' not found`

**When:** Specific key not found in keystore (from RPC call)

**Solution:**
```bash
# List all stored keys via RPC
echo '{"jsonrpc":"2.0","id":1,"method":"list_keys"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Or via CLI
./build/armorclaw-bridge list-keys

# Add the missing key if needed
./build/armorclaw-bridge add-key --provider openai --token sk-xxx
```

---

### `Failed to store credential: ...`

**When:** Cannot save a new API key to the keystore

**Solution:**
```bash
# 1. Check keystore directory is writable
ls -la ~/.armorclaw

# 2. Ensure directory exists
mkdir -p ~/.armorclaw
chmod 700 ~/.armorclaw

# 3. Check disk space
df -h ~

# 4. Try again
./build/armorclaw-bridge add-key --provider openai --token sk-xxx
```

---

## Container Errors

### `[ArmorClaw] ✗ ERROR: No API keys detected`

**When:** Container starts but no API keys are present in environment

**Solution:**
```bash
# 1. Start via bridge with proper key
echo '{"jsonrpc":"2.0","id":1,"method":"start","params":{"key_id":"openai-default"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# 2. For testing only, use environment variable
docker run -e OPENAI_API_KEY=sk-xxx armorclaw/agent:v1

# 3. Verify key exists in keystore
./build/armorclaw-bridge list-keys
```

---

### `[ArmorClaw] ✗ ERROR: Command not found: python`

**When:** Agent command is not available in container PATH

**Solution:**
```bash
# 1. Verify container image is complete
docker images | grep armorclaw

# 2. Rebuild container if needed
docker build -t armorclaw/agent:v1 .

# 3. Check container PATH
docker run --rm armorclaw/agent:v1 which python

# 4. Override command for testing
docker run --rm -e OPENAI_API_KEY=sk-xxx armorclaw/agent:v1 python3 -c "print('test')"
```

---

### `[ArmorClaw] ✗ ERROR: Agent validation failed: ...`

**When:** Agent command validation fails before container starts

**Solution:**
```bash
# 1. Check detailed error message in logs
docker logs <container-id>

# 2. Common issues:
#    - Agent module not installed
#    - Incomplete container image
#    - Build failure

# 3. Rebuild container
docker build -t armorclaw/agent:v1 .

# 4. Test with simple command
docker run --rm armorclaw/agent:v1 python3 --version
```

---

### `[ArmorClaw] ✗ ERROR: Invalid secrets structure`

**When:** Secrets JSON from bridge is malformed or missing required fields

**Solution:**
```bash
# 1. Check bridge logs for details
./build/armorclaw-bridge -log-level=debug

# 2. Verify key in keystore has valid data
./build/armorclaw-bridge list-keys

# 3. Remove and re-add the problematic key
# First, list keys to find the ID
./build/armorclaw-bridge list-keys

# Then remove and re-add (manual keystore operation)
# Or reinitialize if needed
```

---

### `[ArmorClaw] ⚠ WARNING: Not running as UID 10001`

**When:** Container is running as wrong user (security issue)

**Solution:**
```bash
# 1. Check Dockerfile USER directive
grep "^USER" Dockerfile

# 2. Ensure container uses UID 10001
# Dockerfile should have: USER 10001

# 3. Rebuild container
docker build -t armorclaw/agent:v1 .

# 4. Verify user in running container
docker run --rm armorclaw/agent:v1 id
```

---

### `[ArmorClaw] ⚠ WARNING: Low memory available (XXX MB)`

**When:** Container has less than 128MB available memory

**Solution:**
```bash
# 1. Check host memory
free -h

# 2. Check Docker memory limits (Docker Desktop)
# Settings > Resources > Memory

# 3. Increase memory allocation if needed
# Or use a machine with more RAM

# 4. Close unnecessary containers
docker ps
docker stop <unnecessary-containers>
```

---

## RPC/Bridge Errors

### `Failed to start server: ...`

**When:** Bridge cannot start the RPC server

**Common causes:**
- Socket file already exists
- Socket directory permissions
- Port already in use

**Solution:**
```bash
# 1. Remove stale socket file
sudo rm -f /run/armorclaw/bridge.sock

# 2. Create socket directory with proper permissions
sudo mkdir -p /run/armorclaw
sudo chown $USER:$USER /run/armorclaw

# 3. Check for running instances
ps aux | grep armorclaw-bridge

# 4. Try again
./build/armorclaw-bridge
```

---

### `Failed to create socket directory: ...`

**When:** Bridge cannot create the runtime directory for sockets

**Solution:**
```bash
# 1. Create directory manually
sudo mkdir -p /run/armorclaw

# 2. Set ownership
sudo chown $USER:$USER /run/armorclaw

# 3. Set permissions
chmod 770 /run/armorclaw

# 4. Try again
./build/armorclaw-bridge
```

---

### `Failed to set socket permissions: ...`

**When:** Bridge cannot set proper permissions on socket file

**Solution:**
```bash
# 1. Ensure you own the socket directory
ls -la /run/armorclaw

# 2. Fix ownership if needed
sudo chown $USER:$USER /run/armorclaw

# 3. Remove stale socket
sudo rm -f /run/armorclaw/bridge.sock

# 4. Start bridge again
./build/armorclaw-bridge
```

---

### `permission denied: /run/armorclaw/bridge.sock`

**When:** Client cannot connect to bridge socket

**Solution:**
```bash
# 1. Check socket permissions
ls -la /run/armorclaw/bridge.sock

# 2. Ensure user is in correct group
groups

# 3. Fix socket directory permissions
sudo chmod 770 /run/armorclaw

# 4. Use alternative socket location
./build/armorclaw-bridge -socket-path /tmp/armorclaw.sock
```

---

### `method 'xxx' not found`

**When:** Calling an invalid RPC method

**Solution:**
```bash
# 1. List valid methods
# See RPC API reference

# 2. Common valid methods:
# - status
# - health
# - start
# - stop
# - list_keys
# - get_key
# - matrix.send
# - matrix.receive
# - matrix.status
# - matrix.login
# - attach_config

# 3. Example correct usage:
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### `invalid JSON-RPC version`

**When:** Request doesn't specify JSON-RPC 2.0

**Solution:**
```bash
# Correct format:
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Must include "jsonrpc": "2.0" in request
```

---

### `key_id is required`

**When:** Start method called without key_id parameter

**Solution:**
```bash
# Include key_id in params
echo '{"jsonrpc":"2.0","id":1,"method":"start","params":{"key_id":"openai-default"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Or use CLI:
./build/armorclaw-bridge start --key openai-default
```

---

### `container_id is required`

**When:** Stop method called without container_id parameter

**Solution:**
```bash
# Include container_id in params
echo '{"jsonrpc":"2.0","id":1,"method":"stop","params":{"container_id":"abc123"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# First, get container_id from status
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### `id parameter required`

**When:** get_key method called without id parameter

**Solution:**
```bash
# Include id in params
echo '{"jsonrpc":"2.0","id":1,"method":"get_key","params":{"id":"openai-default"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Or use CLI:
./build/armorclaw-bridge list-keys
```

---

## Matrix Errors

### `Matrix adapter not configured`

**When:** Trying to use Matrix features without enabling Matrix

**Solution:**
```bash
# 1. Enable Matrix in config.toml:
[matrix]
  enabled = true
  homeserver_url = "https://matrix.example.com"
  username = "bridge-bot"
  password = "your-password"

# 2. Or use CLI flags:
./build/armorclaw-bridge -matrix-enabled -matrix-homeserver https://matrix.example.com
```

---

### `username and password are required`

**When:** Matrix login missing credentials

**Solution:**
```bash
# 1. Add credentials to config.toml:
[matrix]
  username = "bridge-bot"
  password = "your-password"

# 2. Or use CLI flags:
./build/armorclaw-bridge -matrix-username bridge-bot -matrix-password your-password

# 3. Or use RPC login:
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.login","params":{"username":"bridge-bot","password":"your-password"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### `room_id and message are required`

**When:** Matrix send missing parameters

**Solution:**
```bash
# Include both room_id and message
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.send","params":{"room_id":"!xxx:example.com","message":"Hello"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### `name and content are required`

**When:** attach_config method missing parameters

**Solution:**
```bash
# Include both name and content
echo '{"jsonrpc":"2.0","id":1,"method":"attach_config","params":{"name":"agent.env","content":"MODEL=gpt-4"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## CLI Errors

### `Failed to determine home directory: ...`

**When:** System cannot determine user's home directory

**Solution:**
```bash
# 1. Set HOME environment variable
export HOME=/home/username

# 2. Or use explicit paths
./build/armorclaw-bridge init -config-output /path/to/config.toml

# 3. Verify home directory
echo $HOME
ls -la ~/
```

---

### `Failed to create config directory: ...`

**When:** Cannot create configuration directory

**Solution:**
```bash
# 1. Create parent directory
sudo mkdir -p ~/.armorclaw

# 2. Set ownership
sudo chown $USER:$USER ~/.armorclaw

# 3. Set permissions
chmod 700 ~/.armorclaw

# 4. Try again
./build/armorclaw-bridge init
```

---

## General Troubleshooting

### For any error not listed:

1. **Enable debug logging:**
   ```bash
   ./build/armorclaw-bridge -log-level=debug
   ```

2. **Check logs:**
   ```bash
   # Bridge logs (when running in foreground)
   ./build/armorclaw-bridge -log-level=debug

   # Container logs
   docker logs <container-id>

   # System logs
   journalctl -u armorclaw-bridge
   ```

3. **Verify configuration:**
   ```bash
   ./build/armorclaw-bridge validate
   ```

4. **Check Docker status:**
   ```bash
   docker ps
   docker info
   ```

5. **Reset and retry:**
   ```bash
   # Stop all ArmorClaw processes
   pkill -f armorclaw-bridge

   # Clean up sockets
   sudo rm -rf /run/armorclaw

   # Restart
   ./build/armorclaw-bridge
   ```

---

**Error Catalog Last Updated:** 2026-02-15
**For additional help:** https://github.com/armorclaw/armorclaw/issues
