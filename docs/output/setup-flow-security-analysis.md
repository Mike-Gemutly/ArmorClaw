# Setup Flow Security Analysis

> **Date:** 2026-02-07
> **Status:** CRITICAL ISSUES FOUND
> **Priority:** URGENT - Blocks production deployment

---

## Executive Summary

The current setup flow has **CRITICAL SECURITY GAPS** that prevent API keys from being loaded correctly before container hardening. If unaddressed, containers will be **UNUSABLE** in production.

**Root Cause:** Secrets injection mechanism is incompletely implemented.

---

## Current Setup Flow (BROKEN)

```
┌─────────────────────────────────────────────────────────────┐
│  STEP 1: User starts container                              │
│  docker run -e OPENAI_API_KEY=sk-xxx armorclaw/agent:v1   │
└──────────────────┬────────────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────────────┐
│  STEP 2: Container entrypoint runs (as root)              │
│  - Verifies API keys present in environment ✓              │
│  - Switches to UID 10001 (claw user)                       │
│  - **UNSETS all environment variables** ✗                  │
│  - Execs agent process                                     │
└──────────────────┬────────────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────────────┐
│  STEP 3: Agent starts (as UID 10001)                      │
│  - NO API keys available (they were unset!)               │
│  - Container is hardened (no shell, no tools)             │
│  - NO WAY to inject keys now                              │
└─────────────────────────────────────────────────────────────┘
```

**Result:** Container starts but cannot make API calls.

---

## Critical Issues Found

### Issue #1: Entrypoint Unsets Secrets Before Agent Can Use Them

**Location:** `container/opt/openclaw/entrypoint.py:64-66`

```python
for key in ['OPENAI_API_KEY', 'ANTHROPIC_API_KEY', ...]:
    os.environ[key] = ''  # <-- CLEARS THE KEYS!
```

**Problem:** The entrypoint clears environment variables AFTER verifying presence but BEFORE the agent starts. The agent exec happens AFTER this clearing, so it never receives the keys.

**Impact:** Containers start successfully but cannot make API calls.

---

### Issue #2: File Descriptor Passing Is Not Implemented

**Location:** `bridge/pkg/rpc/server.go:487-489`

```go
// Mount secrets as file descriptor 3 (passed via extra_hosts)
// In production, this would use proper FD passing via the container API
```

**Problem:** This is a TODO comment. No actual FD passing occurs.

**Missing Implementation:**
1. No `ExtraHosts` configuration in hostConfig
2. No named pipe or socket mount in container
3. No container code to read from `/proc/self/fd/3`
4. No bridge-to-container FD passing mechanism

**Impact:** The bridge cannot inject secrets after container start.

---

### Issue #3: Container Cannot Recover From Missing Secrets

**Location:** `container/openclaw/agent.py`

**Problem:** If secrets aren't available at agent start:
- No retry mechanism
- No fallback to fetch from bridge
- No error handling for missing credentials
- Container just runs in "standalone mode" indefinitely

**Impact:** Silent failure - container appears healthy but is non-functional.

---

### Issue #4: No Pre-Hardening Validation

**Problem:** API keys are NOT validated before:
1. Building the container image
2. Hardening the container (removing tools)
3. Starting the container

**Missing Validation:**
- No check that keys actually work with the API
- No verification of key format
- No test API call before container start

**Impact:** Invalid keys result in unusable hardened containers.

---

### Issue #5: Environment Variables Are Exposed in Docker Inspect

**Location:** Current implementation uses `-e OPENAI_API_KEY=...`

**Problem:** Even before unsetting, keys are visible in:
```bash
docker inspect <container_id> | grep -i env
```

**Impact:** Defeats the purpose of ephemeral secrets injection.

---

## Correct Setup Flow (REQUIRED)

```
┌─────────────────────────────────────────────────────────────┐
│  PHASE 0: Pre-Hardening (Setup Wizard)                     │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 1. Collect API keys from user                       │    │
│  │ 2. VALIDATE keys work (test API call)               │    │
│  │ 3. Store keys in encrypted keystore                 │    │
│  │ 4. Generate key_id for retrieval                    │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────┬────────────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────────────┐
│  PHASE 1: Bridge Start                                     │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 1. Start ArmorClaw bridge                          │    │
│  │ 2. Load encrypted keystore                           │    │
│  │ 3. Listen for container start requests              │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────┬────────────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────────────┐
│  PHASE 2: Container Start (via Bridge)                   │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 1. Bridge receives start request with key_id        │    │
│  │ 2. Retrieve token from keystore                      │    │
│  │ 3. Create named pipe: /run/armorclaw/secrets-<id>  │    │
│  │ 4. Write secrets JSON to pipe                        │    │
│  │ 5. Mount pipe as container volume/secret             │    │
│  │ 6. Start container with SECRETS_FD=3 env var        │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────┬────────────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────────────┐
│  PHASE 3: Container Agent Startup                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 1. Read SECRETS_FD env var                          │    │
│  │ 2. Open /proc/self/fd/3                             │    │
│  │ 3. Parse secrets JSON                               │    │
│  │ 4. Set environment variables for Python             │    │
│  │ 5. Close FD 3                                       │    │
│  │ 6. Start agent with credentials                     │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────┬────────────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────────────┐
│  PHASE 4: Post-Start Validation                           │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 1. Agent verifies it has credentials                │    │
│  │ 2. Test API call (if validation enabled)            │    │
│  │ 3. Report status to bridge                          │    │
│  │ 4. If failed: cleanup and exit                      │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## Required Fixes

### Fix #1: Implement Named Pipe Secret Passing

**Bridge Changes (`bridge/pkg/rpc/server.go`):**

```go
// Instead of writing to a pipe that's never mounted:
// 1. Create a named pipe in /run/armorclaw/
secretPipe := fmt.Sprintf("/run/armorclaw/secrets-%s", containerID)
err := syscall.Mkfifo(secretPipe, 0660)

// 2. Write secrets to pipe in background
go func() {
    f, _ := os.OpenFile(secretPipe, os.O_WRONLY, 0660)
    defer f.Close()
    f.Write(secretsData)
}()

// 3. Mount pipe as volume in container
hostConfig.Binds = append(hostConfig.Binds,
    fmt.Sprintf("%s:/run/secrets:ro", secretPipe))
```

**Container Changes (`container/opt/openclaw/entrypoint.py`):**

```python
# Read from SECRETS_FD or /run/secrets
secrets_fd = os.getenv('SECRETS_FD', '3')
secrets_path = os.getenv('SECRETS_PATH', '/run/secrets')

try:
    if os.path.exists(secrets_path):
        with open(secrets_path, 'r') as f:
            secrets = json.load(f)
            # Set environment variables from secrets
            os.environ['OPENAI_API_KEY'] = secrets['token']
    else:
        # Try file descriptor
        fd = int(secrets_fd)
        with os.fdopen(fd, 'r') as f:
            secrets = json.load(f)
except Exception as e:
    print(f"Failed to read secrets: {e}", file=sys.stderr)
    sys.exit(1)
```

### Fix #2: Remove Premature Environment Variable Unsetting

**Change (`container/opt/openclaw/entrypoint.py:64-66`):**

```python
# REMOVE THESE LINES:
# for key in ['OPENAI_API_KEY', ...]:
#     os.environ[key] = ''
```

**Rationale:** Let the agent process inherit the environment variables. If using FD passing, the entrypoint will read from FD and set them.

### Fix #3: Add Pre-Hardening Key Validation

**New Script (`scripts/validate-keys.sh`):**

```bash
#!/bin/bash
validate_openai_key() {
    local key="$1"
    response=$(curl -s https://api.openai.com/v1/models \
        -H "Authorization: Bearer $key")

    if echo "$response" | jq -e '.object == "list"' > /dev/null; then
        echo "✓ OpenAI key is valid"
        return 0
    else
        echo "✗ OpenAI key is invalid"
        return 1
    fi
}

# Validate all keys before allowing container start
```

### Fix #4: Add Post-Start Credentials Check

**Change (`container/openclaw/agent.py`):**

```python
def verify_credentials() -> bool:
    """Verify that credentials are available."""
    # Check for API keys
    has_keys = any(
        os.getenv(key) for key in [
            "OPENAI_API_KEY",
            "ANTHROPIC_API_KEY",
            # ...
        ]
    )

    if not has_keys:
        logger.error("No API credentials available!")
        logger.error("Container cannot function without credentials.")
        logger.error("Please check bridge secret injection.")
        return False

    return True

# In main():
if not verify_credentials():
    sys.exit(1)
```

---

## Implementation Plan

### Phase 1: Fix Secret Passing (URGENT)

| Task | File | Effort |
|------|------|--------|
| Implement named pipe creation | `bridge/pkg/rpc/server.go` | 2 hours |
| Mount pipe as container volume | `bridge/pkg/rpc/server.go` | 1 hour |
| Update entrypoint to read secrets | `container/opt/openclaw/entrypoint.py` | 2 hours |
| Add SECRETS_PATH environment variable | `Dockerfile` | 30 minutes |
| Test secret passing end-to-end | New test script | 2 hours |

**Total:** ~7-8 hours

### Phase 2: Add Validation (IMPORTANT)

| Task | File | Effort |
|------|------|--------|
| Create validate-keys.sh script | `scripts/validate-keys.sh` | 2 hours |
| Add pre-flight check to start RPC | `bridge/pkg/rpc/server.go` | 1 hour |
| Add post-start check to agent | `container/openclaw/agent.py` | 1 hour |
| Update setup wizard | New setup wizard | 3 hours |

**Total:** ~7 hours

---

## Testing Required

### Test 1: Secret Passing Verification

```bash
# 1. Store key in keystore
echo '{"jsonrpc":"2.0","method":"store_key","params":{"provider":"openai","token":"sk-..."},"id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# 2. Start container
echo '{"jsonrpc":"2.0","method":"start","params":{"key_id":"..."},"id":2}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# 3. Verify container has secrets
docker exec <container_id> env | grep API_KEY
```

### Test 2: Missing Secret Recovery

```bash
# 1. Start container WITHOUT valid key_id
# 2. Verify container exits with error
# 3. Verify cleanup (no orphaned containers)
```

### Test 3: Pre-Hardening Validation

```bash
# 1. Run validate-keys.sh with invalid key
# 2. Verify it FAILS before container build
# 3. Run with valid key
# 4. Verify it PASSES
```

---

## Security Considerations

### DO NOT Use Environment Variables for Secrets

Environment variables are visible in:
```bash
docker inspect <container>
/proc/<pid>/environ
docker history <image>
```

### USE File Descriptor Passing Instead

FD passing ensures:
- ✅ Secrets never touch disk
- ✅ Secrets not visible in docker inspect
- ✅ Secrets isolated to single process
- ✅ Secrets vanish when process exits

---

## Timeline

**IMMEDIATE (This Session):**
- ✅ Document issues
- ⏳ Fix named pipe implementation
- ⏳ Update entrypoint to read secrets

**Today:**
- ⏳ Add validation scripts
- ⏳ Test end-to-end secret passing
- ⏳ Update documentation

**Before Production:**
- ⏳ Setup wizard with pre-validation
- ⏳ Recovery mechanisms
- ⏳ Complete test suite

---

## Conclusion

The current setup flow has **CRITICAL GAPS** that prevent API keys from being loaded correctly. The secret injection mechanism is incompletely implemented, and the entrypoint clears environment variables before the agent can use them.

**URGENT ACTION REQUIRED:** Implement the fixes outlined above before any production deployment.

**Risk if Unaddressed:** Containers will start successfully but cannot make API calls, rendering the entire system non-functional.
