# ArmorClaw Troubleshooting Guide

> **Last Updated:** 2026-02-07
> **Version:** 1.2.0
> **Component:** All ArmorClaw Components

---

## 🔍 Quick Error Lookup

**Got an error?** Search for the exact error text in our [Error Message Catalog](error-catalog.md).

The catalog contains every error message from ArmorClaw with step-by-step solutions.

---

## How to Use This Guide

### For Immediate Help:
1. **Copy the exact error message** you're seeing
2. **Search the Error Catalog** - it's indexed by error text
3. **Follow the solution** for your specific error

### For Systematic Debugging:
1. Run the [Quick Diagnostics](#quick-diagnostics) below
2. Check [Common Issues](#common-issues) for your symptom
3. Follow [Debug Procedures](#debug-procedures) for detailed investigation
4. Use [Recovery Procedures](#recovery-procedures) if needed

---

## Quick Diagnostics

### Health Check

```bash
# Check bridge status
echo '{"jsonrpc":"2.0","id":1,"method":"health"}' | nc -U /run/armorclaw/bridge.sock

# Check running containers
docker ps --filter "name=armorclaw"

# Check Matrix connection
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.status"}' | nc -U /run/armorclaw/bridge.sock
```

### Log Locations

- **Bridge Logs:** stdout/stderr or configured log file
- **Container Logs:** `docker logs armorclaw-openclaw-<timestamp>`
- **System Logs:** `journalctl -u armorclaw-bridge` (if running as service)

---

## Common Issues

### Bridge Issues

#### Issue: Bridge Won't Start

**Symptoms:**
- `Error: failed to start bridge`
- `Error: cannot bind to socket`

**Possible Causes:**
1. Socket file already exists
2. Socket directory doesn't exist
3. Insufficient permissions
4. Port already in use

**Solutions:**

```bash
# 1. Remove stale socket file
sudo rm -f /run/armorclaw/bridge.sock

# 2. Create socket directory with correct permissions
sudo mkdir -p /run/armorclaw
sudo chown $USER:$USER /run/armorclaw

# 3. Check what's using the socket/port
sudo lsof /run/armorclaw/bridge.sock
sudo netstat -tulpn | grep 8080

# 4. Run with verbose logging
sudo ./build/armorclaw-bridge -log-level=debug
```

---

#### Issue: Permission Denied on Socket

**Symptoms:**
- `permission denied: /run/armorclaw/bridge.sock`
- `cannot connect to bridge`

**Solutions:**

```bash
# Solution 1: Run with sudo
sudo ./build/armorclaw-bridge

# Solution 2: Fix socket permissions
sudo mkdir -p /run/armorclaw
sudo chown $USER:$USER /run/armorclaw
chmod 770 /run/armorclaw

# Solution 3: Use alternative socket location
./build/armorclaw-bridge -socket-path /tmp/armorclaw.sock
```

---

#### Issue: Configuration File Not Found

**Symptoms:**
- `config file not found: ~/.armorclaw/config.toml`

**Solutions:**

```bash
# Solution 1: Initialize config
./build/armorclaw-bridge init

# Solution 2: Specify custom config
./build/armorclaw-bridge -config /path/to/config.toml

# Solution 3: Use environment variables
export ARMORCLAW_MATRIX_ENABLED=true
export ARMORCLAW_LOG_LEVEL=info
sudo ./build/armorclaw-bridge
```
#### Issue: Keystore Not Found

**Symptoms:**
- `keystore not found: ~/.armorclaw/keystore.db`
- `failed to open keystore`

**Solutions:**

```bash
# Solution 1: Initialize keystore
./build/armorclaw-bridge init

# Solution 2: Specify custom keystore path
./build/armorclaw-bridge -keystore-path /custom/path/keystore.db

# Solution 3: Create keystore manually
mkdir -p ~/.armorclaw
touch ~/.armorclaw/keystore.db
chmod 600 ~/.armorclaw/keystore.db
```

---

### Container Issues

#### Issue: Container Won't Start

**Symptoms:**
- `Error: container creation failed`
- `Error: failed to create container`

**Possible Causes:**
1. Invalid image
2. Missing credentials
3. Invalid seccomp profile
4. Insufficient resources

**Solutions:**

```bash
# 1. Verify image exists
docker images | grep armorclaw

# 2. Pull image if missing
docker pull armorclaw/agent:v1

# 3. Check if key exists
echo '{"jsonrpc":"2.0","id":1,"method":"list_keys"}' | nc -U /run/armorclaw/bridge.sock

# 4. Check bridge logs for detailed error
sudo ./build/armorclaw-bridge -log-level=debug

# 5. Verify Docker daemon is running
sudo systemctl status docker
```

---

#### Issue: Secrets Not Injected

**Symptoms:**
- Container starts but has no access to credentials
- `FD 3 not found` in container

**Solutions:**

```bash
# 1. Verify key exists
echo '{"jsonrpc":"2.0","id":1,"method":"get_key","params":{"id":"openai-key-1"}}' | nc -U /run/armorclaw/bridge.sock

# 2. Check container FDs
docker exec <container-id> ls -la /proc/self/fd/

# 3. Check entrypoint logs
docker logs <container-id>

# 4. Verify container has proper entrypoint
docker inspect <container-id> | grep -A 5 "Entrypoint"
```

---

### Matrix Issues

#### Issue: Matrix Login Failed

**Symptoms:**
- `matrix login failed: invalid credentials`
- `Error: authentication failed`

**Solutions:**

```bash
# 1. Verify homeserver is accessible
curl -I https://matrix.armorclaw.com

# 2. Check Matrix status
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.status"}' | nc -U /run/armorclaw/bridge.sock

# 3. Try RPC login
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.login","params":{"username":"bridge-bot","password":"secret"}}' | nc -U /run/armorclaw/bridge.sock

# 4. Enable debug logging
export ARMORCLAW_LOG_LEVEL=debug
sudo ./build/armorclaw-bridge -matrix-enabled
```

---

## Debug Procedures

### Enable Debug Logging

```bash
# Method 1: Environment variable
export ARMORCLAW_LOG_LEVEL=debug
sudo ./build/armorclaw-bridge

# Method 2: CLI flag
sudo ./build/armorclaw-bridge -log-level=debug

# Method 3: Config file
[logging]
level = "debug"
```

### Inspect Container State

```bash
# Container details
docker inspect <container-id>

# Container processes
docker top <container-id>

# Container filesystem changes
docker diff <container-id>

# Resource usage
docker stats <container-id>
```

---

## Recovery Procedures

### Reset Bridge State

```bash
# 1. Stop bridge
sudo pkill -f armorclaw-bridge

# 2. Clean up sockets and PIDs
sudo rm -f /run/armorclaw/bridge.sock
sudo rm -f /run/armorclaw/bridge.pid

# 3. Remove stale containers
docker ps -aq --filter "name=armorclaw" | xargs -r docker rm -f

# 4. Restart bridge
sudo ./build/armorclaw-bridge
```

### Reinitialize Keystore

```bash
# WARNING: This will delete all stored credentials!

# 1. Backup existing keystore
cp ~/.armorclaw/keystore.db ~/.armorclaw/keystore.db.backup

# 2. Remove keystore
rm ~/.armorclaw/keystore.db

# 3. Reinitialize
./build/armorclaw-bridge init
```

---

## Getting Help

### Before Asking for Help

1. **Check logs:** Enable debug logging and capture output
2. **Run diagnostics:** Use health check commands above
3. **Search issues:** Check https://github.com/armorclaw/armorclaw/issues
4. **Read docs:** Review relevant documentation sections

### Resources

- **Documentation:** https://docs.armorclaw.com
- **GitHub Issues:** https://github.com/armorclaw/armorclaw/issues
- **Matrix Room:** #armorclaw:matrix.org
- **Email:** support@armorclaw.com

---

**Troubleshooting Guide Last Updated:** 2026-02-06
**Compatible with Bridge Version:** 1.0.0
