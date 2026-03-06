# ArmorClaw Matrix Integration Fixes

> **Status:** Required for reliable deployment
> **Priority:** High
> **Created:** 2026-03-06

This document tracks required code changes to make ArmorClaw + Conduit deploy reliably without manual intervention.

---

## Problem Summary

The default ArmorClaw container cannot successfully deploy a Matrix stack automatically due to:
1. Synapse-specific API usage (not compatible with Conduit)
2. Username format incompatibility (`@bridge:host` vs `bridge`)
3. Auto-spawning Matrix stack conflicts with external deployments
4. Hardcoded `localhost` instead of Docker DNS
5. Configuration overwrite issues
6. Weak health detection / race conditions

---

## Required Code Changes

### 1. Replace Synapse API with Standard Matrix API

**File:** `deploy/container-setup.sh`
**Function:** `register_matrix_user()`

**Current (broken):**
```bash
# Uses Synapse-specific endpoint
GET/POST /_synapse/admin/v1/register
```

**Required fix:**
Use standard Matrix v3 registration API:
```bash
POST /_matrix/client/v3/register
{
  "username": "bridge",
  "password": "bridgepass",
  "auth": {"type": "m.login.dummy"}
}
```

**Location:** Lines ~1700-1750 in `container-setup.sh`

---

### 2. Fix Username Format

**File:** `deploy/container-setup.sh`
**Function:** Config generation

**Current (broken):**
```bash
username = "@bridge"
```

**Required fix:**
```bash
username = "bridge"  # No @ prefix
```

The Matrix login API expects just the localpart, not the full Matrix ID.

**Also check:** Username parsing in Go bridge code if it adds @ prefix.

---

### 3. Add Matrix Wait Logic

**File:** `deploy/container-setup.sh`
**Function:** Before bridge starts

**Add:**
```bash
wait_for_matrix() {
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf --connect-timeout 2 "${MATRIX_URL:-http://localhost:6167}/_matrix/client/versions" >/dev/null 2>&1; then
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    return 1
}
```

Call this before attempting user registration or bridge login.

---

### 4. Respect Existing Config

**File:** `deploy/container-setup.sh`
**Function:** Config generation

**Current (broken):**
```bash
# Always writes new config
write_config
```

**Required fix:**
```bash
if [ ! -f "$CONFIG_FILE" ]; then
    write_config
else
    log_info "Config file exists - preserving user settings"
fi
```

---

### 5. Docker DNS vs Localhost

**File:** `deploy/container-setup.sh`
**Function:** Matrix URL configuration

**Current (broken for Docker):**
```bash
MATRIX_URL="http://localhost:6167"
```

**Required fix - detect environment:**
```bash
# If running in Docker with shared network, use container DNS
if [ -f /.dockerenv ] && [ "${ARMORCLAW_EXTERNAL_MATRIX:-false}" != "true" ]; then
    # Check if we're in docker-compose with matrix service
    if getent hosts matrix 2>/dev/null; then
        MATRIX_URL="http://matrix:6167"
    elif getent hosts conduit 2>/dev/null; then
        MATRIX_URL="http://conduit:6167"
    fi
fi
```

---

### 6. Improve Error Messages

**File:** `deploy/container-setup.sh`
**Function:** Matrix connection failures

**Current:**
```
Matrix stack unavailable
```

**Required fix:**
```bash
print_error "Matrix server unreachable at $MATRIX_URL"
print_error ""
print_error "Possible causes:"
print_error "  1. Matrix container not started"
print_error "  2. Wrong Docker network (containers must share a network)"
print_error "  3. Port mismatch (expected 6167)"
print_error ""
print_error "Quick fix:"
print_error "  docker compose -f docker-compose-full.yml up -d"
```

---

### 7. Username Parsing in Go

**File:** `bridge/internal/adapter/matrix.go` (or config parsing)
**Check:** If Go code adds @ prefix to username

**Required fix:**
```go
// Strip @ prefix and :server suffix if present
func normalizeUsername(username string) string {
    // Remove @ prefix
    username = strings.TrimPrefix(username, "@")
    // Remove :server suffix
    if idx := strings.Index(username, ":"); idx > 0 {
        username = username[:idx]
    }
    return username
}
```

---

## Files to Modify

| File | Changes |
|------|---------|
| `deploy/container-setup.sh` | All items 1-6 above |
| `bridge/internal/adapter/matrix.go` | Item 7 - username normalization |
| `docker-compose-full.yml` | Ensure correct service names and networking |

---

## Verification Steps

After fixes, verify:

1. **Fresh install works:**
   ```bash
   docker compose down -v
   docker compose up -d
   # Should work without manual intervention
   ```

2. **Config preserved on restart:**
   ```bash
   # Edit config manually
   docker compose restart
   # Config should still have manual edits
   ```

3. **External Matrix mode:**
   ```bash
   ARMORCLAW_EXTERNAL_MATRIX=true docker compose up -d
   # Should skip internal Matrix management
   ```

---

## Related Documentation

- [Production Deployment Guide](production-deployment.md)
- [README Deployment Options](../../README.md#installation-options)

---

### 8. AI Provider Base URL Support

**File:** `bridge/cmd/bridge/main.go`, `bridge/pkg/keystore/keystore.go`
**Function:** `add-key` command, credential storage

**Issue:**
The `add-key` command did not support `--base-url` flag for OpenAI-compatible providers.

**Fix (COMPLETED):**
```bash
# Now supports custom base URLs
armorclaw-bridge add-key --provider openai --base-url https://open.bigmodel.cn/api/paas/v4 --id zhipu --token your-api-key
```

**Supported OpenAI-Compatible Providers:**
| Provider | Base URL |
|----------|----------|
| Zhipu AI | `https://open.bigmodel.cn/api/paas/v4` |
| DeepSeek | `https://api.deepseek.com/v1` |
| Moonshot | `https://api.moonshot.cn/v1` |
| NVIDIA NIM | `https://integrate.api.nvidia.com/v1` |
| OpenRouter | `https://openrouter.ai/api/v1` |
| Groq | `https://api.groq.com/openai/v1` |
| Cloudflare AI Gateway | `https://gateway.ai.cloudflare.com/v1` |
| Custom | `https://your-api.com/v1` |

---

## Files to Modify

| File | Changes |
|------|---------|
| `deploy/container-setup.sh` | All items 1-6 above |
| `bridge/internal/adapter/matrix.go` | Item 7 - username normalization |
| `bridge/cmd/bridge/main.go` | Item 8 - base URL support |
| `bridge/pkg/keystore/keystore.go` | Item 8 - base URL storage |
| `docker-compose-full.yml` | Ensure correct service names and networking |

---

## Verification Steps

After fixes, verify:

1. **Fresh install works:**
   ```bash
   docker compose down -v
   docker compose up -d
   # Should work without manual intervention
   ```

2. **Config preserved on restart:**
   ```bash
   # Edit config manually
   docker compose restart
   # Config should still have manual edits
   ```

3. **External Matrix mode:**
   ```bash
   ARMORCLAW_EXTERNAL_MATRIX=true docker compose up -d
   # Should skip internal Matrix management
   ```

4. **Custom AI provider:**
   ```bash
   armorclaw-bridge add-key --provider openai --base-url https://api.deepseek.com/v1 --id deepseek --token your-key
   armorclaw-bridge list-keys
   # Should show the key with base URL
   ```

---

## Related Documentation

- [Production Deployment Guide](production-deployment.md)
- [README Deployment Options](../../README.md#installation-options)

---

## Progress

- [x] Added `ARMORCLAW_EXTERNAL_MATRIX` support
- [x] Replace Synapse API with standard Matrix API
- [x] Fix username format in config generation
- [x] Add Matrix wait/retry logic
- [x] Preserve existing config on restart
- [x] Docker DNS auto-detection
- [x] Improved error messages
- [ ] Username normalization in Go
- [x] AI Provider base URL support (`--base-url` flag)
