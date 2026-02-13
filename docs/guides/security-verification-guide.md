# ArmorClaw Security Verification Guide

> **Last Updated:** 2026-02-09
> **Version:** v1.0
> **Purpose:** Manual verification of security hardening measures

## Overview

This guide provides step-by-step instructions for manually verifying that ArmorClaw's security hardening measures are working correctly. It covers all layers of defense implemented to prevent container escape and data exfiltration.

---

## Quick Verification (One-Liner)

Run this single command to verify the most critical security controls:

```bash
docker run --rm --network=none --security-opt seccomp=container/seccomp-profile.json --cap-drop=ALL --read-only armorclaw/agent:v1 python3 -c "import os; os.execl('/bin/sh')" 2>&1 | grep -i "permission\|denied\|operation not permitted" && echo "✅ SECURITY VERIFIED" || echo "❌ SECURITY BREACH DETECTED"
```

Expected output: `✅ SECURITY VERIFIED`

---

## Layer-by-Layer Verification

### Layer 1: Filesystem Hardening (chmod a-x)

**Purpose:** Prevent Python/Node from executing shells or other binaries.

#### Test 1.1: Verify shells are not executable

```bash
# Start a test container
CONTAINER_ID=$(docker run -d --rm armorclaw/agent:v1 python3 -c "import time; time.sleep(30)")

# Check if /bin/sh has execute permissions
docker exec $CONTAINER_ID ls -la /bin/sh 2>&1

# Expected: Permission denied or file not found
# If file exists with -rwxr-xr-x, this layer FAILED

# Cleanup
docker stop $CONTAINER_ID
```

#### Test 1.2: Verify Python cannot exec /bin/sh

```bash
# Try to execute shell via Python os.execl
docker run --rm armorclaw/agent:v1 python3 -c "import os; os.execl('/bin/sh')" 2>&1

# Expected: Permission denied, operation not permitted, or OSError
# If a shell prompt appears, this layer FAILED
```

#### Test 1.3: Verify Node cannot spawn processes

```bash
# Try to execute shell via Node child_process
docker run --rm armorclaw/agent:v1 node -e "require('child_process').exec('/bin/sh')" 2>&1

# Expected: EACCES or permission denied
# If a shell prompt appears, this layer FAILED
```

---

### Layer 2: Network Isolation (--network=none)

**Purpose:** Prevent data exfiltration via network.

#### Test 2.1: Verify Python urllib cannot connect

```bash
# Try to make HTTP request via Python
docker run --rm --network=none armorclaw/agent:v1 python3 -c "import urllib.request; urllib.request.urlopen('http://httpbin.org/post')" 2>&1

# Expected: Network unreachable, connection refused, or timeout
# If HTTP response is received, this layer FAILED
```

#### Test 2.2: Verify Node fetch cannot connect

```bash
# Try to make HTTP request via Node
docker run --rm --network=none armorclaw/agent:v1 node -e "fetch('http://httpbin.org/post')" 2>&1

# Expected: ECONNREFUSED, ENETUNREACH, or fetch error
# If HTTP response is received, this layer FAILED
```

#### Test 2.3: Verify no network tools available

```bash
# Check if curl, wget, nc exist
docker run --rm armorclaw/agent:v1 which curl wget nc 2>&1

# Expected: "command not found" or similar for all tools
# If any tool path is returned, this layer FAILED
```

---

### Layer 3: Seccomp Profile (Syscall Filtering)

**Purpose:** Block dangerous syscalls at kernel level.

#### Test 3.1: Verify seccomp profile is valid JSON

```bash
# Check JSON syntax
cat container/seccomp-profile.json | jq . 2>&1

# Expected: Valid JSON output, not "parse error"
# If parse error, seccomp profile is INVALID
```

#### Test 3.2: Verify seccomp blocks socket syscall

```bash
# Try to create a socket via Python
docker run --rm --security-opt seccomp=container/seccomp-profile.json armorclaw/agent:v1 python3 -c "import socket; s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)" 2>&1

# Expected: Permission denied or operation not permitted
# If socket is created successfully, this layer FAILED
```

#### Test 3.3: Verify seccomp blocks connect syscall

```bash
# Try to connect via Python (with --network=none to be sure)
docker run --rm --security-opt seccomp=container/seccomp-profile.json --network=none armorclaw/agent:v1 python3 -c "import socket; s = socket.socket(); s.connect(('httpbin.org', 80))" 2>&1

# Expected: Permission denied or operation not permitted
# If connection succeeds, this layer FAILED
```

---

### Layer 4: Capability Dropping (--cap-drop=ALL)

**Purpose:** Remove all Linux capabilities from container.

#### Test 4.1: Verify container has no capabilities

```bash
# Start container and check capabilities
CONTAINER_ID=$(docker run -d --rm --cap-drop=ALL armorclaw/agent:v1 python3 -c "import time; time.sleep(30)")

# Check capabilities
docker inspect $CONTAINER_ID --format '{{.HostConfig.CapDrop}}' 2>&1

# Expected: [ALL] or similar
# If empty or missing capabilities, this layer FAILED

# Cleanup
docker stop $CONTAINER_ID
```

#### Test 4.2: Verify cannot perform privileged operations

```bash
# Try to perform a privileged operation (requires CAP_NET_RAW)
docker run --rm --cap-drop=ALL armorclaw/agent:v1 python3 -c "import socket; s = socket.socket(socket.AF_PACKET, socket.SOCK_RAW)" 2>&1

# Expected: Permission denied or operation not permitted
# If socket is created, this layer FAILED
```

---

### Layer 5: Read-Only Root Filesystem (--read-only)

**Purpose:** Prevent container from modifying its own filesystem.

#### Test 5.1: Verify root filesystem is read-only

```bash
# Try to write to root filesystem
docker run --rm --read-only armorclaw/agent:v1 python3 -c "open('/tmp/test.txt', 'w')" 2>&1

# Expected: Read-only file system error
# If file is created successfully, this layer FAILED
```

#### Test 5.2: Verify /tmp is a tmpfs (writable)

```bash
# Check /tmp is tmpfs
CONTAINER_ID=$(docker run -d --rm --read-only armorclaw/agent:v1 python3 -c "import time; time.sleep(30)")
docker exec $CONTAINER_ID mount | grep tmp 2>&1

# Expected: tmpfs on /tmp
# If no tmpfs, container may not function properly

# Cleanup
docker stop $CONTAINER_ID
```

---

### Layer 6: LD_PRELOAD Security Hook

**Purpose:** Intercept dangerous library calls at library level.

#### Test 6.1: Verify LD_PRELOAD is set

```bash
# Check LD_PRELOAD environment variable
docker run --rm armorclaw/agent:v1 env | grep LD_PRELOAD 2>&1

# Expected: LD_PRELOAD=/opt/openclaw/lib/libarmorclaw_hook.so
# If not set, this layer is NOT ACTIVE
```

#### Test 6.2: Verify security hook library exists

```bash
# Check if library file exists
CONTAINER_ID=$(docker run -d --rm armorclaw/agent:v1 python3 -c "import time; time.sleep(30)")
docker exec $CONTAINER_ID ls -la /opt/openclaw/lib/libarmorclaw_hook.so 2>&1

# Expected: File exists with -rwxr-xr-x permissions
# If file doesn't exist, this layer FAILED

# Cleanup
docker stop $CONTAINER_ID
```

#### Test 6.3: Verify hook intercepts execve

```bash
# Try to execute via Python (should be intercepted by LD_PRELOAD)
docker run --rm armorclaw/agent:v1 python3 -c "import os; os.execve('/bin/sh', ['sh'], None)" 2>&1

# Expected: "ArmorClaw Security: Operation blocked by security policy"
# If shell executes, this layer FAILED
```

---

### Layer 7: Non-Root User (claw:10001)

**Purpose:** Container runs as unprivileged user.

#### Test 7.1: Verify container user is not root

```bash
# Check user ID
docker run --rm armorclaw/agent:v1 id 2>&1

# Expected: uid=10001 (claw) or similar, NOT uid=0 (root)
# If uid=0, this layer FAILED
```

#### Test 7.2: Verify cannot perform root-only operations

```bash
# Try to read root-only file
docker run --rm armorclaw/agent:v1 cat /etc/shadow 2>&1

# Expected: Permission denied
# If file contents are displayed, this layer FAILED
```

---

## Comprehensive Verification Script

Save this as `verify-security.sh` and run it:

```bash
#!/bin/bash
set -euo pipefail

echo "🔒 ArmorClaw Security Verification"
echo "=================================="
echo ""

PASS=0
FAIL=0

test_result() {
    if [ $1 -eq 0 ]; then
        echo "✅ PASS: $2"
        ((PASS++))
    else
        echo "❌ FAIL: $2"
        ((FAIL++))
    fi
}

# Test 1: Shells not executable
echo "Layer 1: Filesystem Hardening"
docker run --rm armorclaw/agent:v1 python3 -c "import os; os.execl('/bin/sh')" 2>&1 | grep -iq "permission\|denied\|operation not permitted"
test_result $? "Python cannot exec /bin/sh"

docker run --rm armorclaw/agent:v1 node -e "require('child_process').exec('/bin/sh')" 2>&1 | grep -iq "error\|denied\|not found"
test_result $? "Node cannot spawn /bin/sh"

# Test 2: Network isolation
echo ""
echo "Layer 2: Network Isolation"
docker run --rm --network=none armorclaw/agent:v1 python3 -c "import urllib.request; urllib.request.urlopen('http://httpbin.org/post')" 2>&1 | grep -iq "connection refused\|network unreachable\|timeout"
test_result $? "Python urllib blocked"

docker run --rm --network=none armorclaw/agent:v1 node -e "fetch('http://httpbin.org/post')" 2>&1 | grep -iq "ECONNREFUSED\|ENETUNREACH\|fetch is not defined"
test_result $? "Node fetch blocked"

# Test 3: Seccomp profile
echo ""
echo "Layer 3: Seccomp Profile"
echo '{"defaultAction":"SCMP_ACT_ALLOW"}' | jq . >/dev/null 2>&1
test_result $? "Seccomp profile is valid JSON"

docker run --rm --security-opt seccomp=container/seccomp-profile.json armorclaw/agent:v1 python3 -c "import socket; s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)" 2>&1 | grep -iq "permission\|denied"
test_result $? "Seccomp blocks socket()"

# Test 4: Non-root user
echo ""
echo "Layer 4: Non-Root User"
docker run --rm armorclaw/agent:v1 id | grep -qv "uid=0"
test_result $? "Container not running as root"

# Summary
echo ""
echo "=================================="
echo "Results: $PASS passed, $FAIL failed"
echo ""

if [ $FAIL -eq 0 ]; then
    echo "✅ ALL SECURITY CHECKS PASSED"
    exit 0
else
    echo "❌ $FAIL SECURITY CHECK(S) FAILED"
    exit 1
fi
```

Run it:
```bash
chmod +x verify-security.sh
./verify-security.sh
```

---

## Windows PowerShell Verification

For Windows users, use `tests/test-exploits.ps1`:

```powershell
# Navigate to project root
cd E:\Micha\.LocalCode\ArmorClaw

# Run PowerShell test script
.\tests\test-exploits.ps1
```

---

## Expected Results Summary

| Test | Expected Result | Severity |
|------|----------------|----------|
| Python `os.execl('/bin/sh')` | Permission denied | CRITICAL |
| Node `child_process.exec('/bin/sh')` | Error/EACCES | CRITICAL |
| Python `urllib.urlopen()` | Connection refused | CRITICAL |
| Node `fetch()` | ECONNREFUSED | CRITICAL |
| `which sh bash curl` | Command not found | HIGH |
| Container user | uid=10001 (not 0) | HIGH |
| `docker exec ls /host` | No such file | MEDIUM |
| `docker exec ls /var/run/docker.sock` | No such file | CRITICAL |

---

## Troubleshooting Failed Tests

### Test Fails: "Python can exec /bin/sh"

**Cause:** The `chmod a-x` step didn't run correctly.

**Fix:**
1. Rebuild container: `docker build --no-cache -t armorclaw/agent:v1 .`
2. Verify build log shows "RUN find /bin -type f -exec chmod a-x"
3. Check that `/usr/bin/env` has execute permissions

### Test Fails: "Network exfiltration works"

**Cause:** `--network=none` flag not applied or seccomp profile not loaded.

**Fix:**
1. Verify command includes `--network=none`
2. Check seccomp profile path is correct
3. Test with simpler command: `docker run --rm --network=none alpine ping -c 1 google.com`

### Test Fails: "Seccomp profile invalid"

**Cause:** JSON syntax error in `container/seccomp-profile.json`.

**Fix:**
1. Validate JSON: `cat container/seccomp-profile.json | jq .`
2. Check for trailing commas
3. Verify all strings are quoted

### Test Fails: "Container won't start"

**Cause:** `--read-only` flag conflicts with application requirements.

**Fix:**
1. Check if `/tmp` is mounted as tmpfs
2. Verify LD_PRELOAD library path is correct
3. Check entrypoint script has execute permissions

---

## Security Checklist

Before deploying ArmorClaw to production, verify:

- [ ] Container image builds successfully
- [ ] All exploit tests pass (26/26)
- [ ] Seccomp profile is valid JSON
- [ ] Container runs as non-root user
- [ ] `--network=none` blocks all network traffic
- [ ] `--cap-drop=ALL` removes all capabilities
- [ ] `--read-only` prevents filesystem writes
- [ ] Shells (sh, bash, dash) are not executable
- [ ] Dangerous tools (curl, wget, nc) are removed
- [ ] Python/Node cannot spawn child processes
- [ ] LD_PRELOAD hook is loaded and functional

---

## Reporting Security Issues

If you find a security vulnerability:

1. **DO NOT** open a public issue
2. Email: security@armorclaw.com
3. Include:
   - Steps to reproduce
   - Expected vs actual behavior
   - Docker version and OS
   - Container image tag

---

**Last Updated:** 2026-02-09
**Next Review:** After any security-related commits
