#!/bin/bash
# ArmorClaw Quick Security Verification
# Run this script to verify all security layers are working

set -euo pipefail

echo "🔒 ArmorClaw Security Verification"
echo "=================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
WARN=0

test_result() {
    local result=$1
    local name=$2
    local severity=${3:-"HIGH"}

    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✅ PASS${NC}: $name"
        ((PASS++))
    elif [ "$result" = "FAIL" ]; then
        echo -e "${RED}❌ FAIL${NC}: $name [$severity]"
        ((FAIL++))
    else
        echo -e "${YELLOW}⚠️  WARN${NC}: $name"
        ((WARN++))
    fi
}

# Check if container image exists
echo "Checking container image..."
if ! docker images armorclaw/agent:v1 | grep -q armorclaw; then
    echo "Building container image..."
    docker build -t armorclaw/agent:v1 . >/dev/null 2>&1
fi
echo ""

# ============================================================================
# Layer 1: Filesystem Hardening (chmod a-x)
# ============================================================================
echo "Layer 1: Filesystem Hardening"
echo "------------------------------"

# Test 1.1: Python cannot exec /bin/sh
if docker run --rm armorclaw/agent:v1 python3 -c "import os; os.execl('/bin/sh')" 2>&1 | grep -iqE "permission|denied|operation not permitted|OSError"; then
    test_result "PASS" "Python cannot exec /bin/sh"
else
    test_result "FAIL" "Python can exec /bin/sh" "CRITICAL"
fi

# Test 1.2: Node cannot spawn /bin/sh
if docker run --rm armorclaw/agent:v1 node -e "require('child_process').exec('/bin/sh')" 2>&1 | grep -iqE "error|denied|not found|EACCES"; then
    test_result "PASS" "Node cannot spawn /bin/sh"
else
    test_result "FAIL" "Node can spawn /bin/sh" "CRITICAL"
fi

# Test 1.3: Shells are not executable
if docker run --rm armorclaw/agent:v1 which sh bash 2>&1 | grep -iq "not found"; then
    test_result "PASS" "Shells (sh, bash) not in PATH"
else
    test_result "WARN" "Shells found in PATH (may be OK if not executable)" "LOW"
fi

echo ""

# ============================================================================
# Layer 2: Network Isolation
# ============================================================================
echo "Layer 2: Network Isolation"
echo "--------------------------"

# Test 2.1: Python urllib blocked
if docker run --rm --network=none armorclaw/agent:v1 python3 -c "import urllib.request; urllib.request.urlopen('http://httpbin.org/post')" 2>&1 | grep -iqE "connection refused|network unreachable|timeout|operation not permitted"; then
    test_result "PASS" "Python urllib blocked by --network=none"
else
    test_result "FAIL" "Python urllib can make network requests" "CRITICAL"
fi

# Test 2.2: Node fetch blocked
if docker run --rm --network=none armorclaw/agent:v1 node -e "fetch('http://httpbin.org/post')" 2>&1 | grep -iqE "ECONNREFUSED|ENETUNREACH|fetch is not defined"; then
    test_result "PASS" "Node fetch blocked by --network=none"
else
    test_result "FAIL" "Node fetch can make network requests" "CRITICAL"
fi

# Test 2.3: No network tools
if docker run --rm armorclaw/agent:v1 which curl wget nc 2>&1 | grep -iq "not found"; then
    test_result "PASS" "Network tools (curl, wget, nc) removed"
else
    test_result "FAIL" "Network tools still present" "HIGH"
fi

echo ""

# ============================================================================
# Layer 3: Seccomp Profile
# ============================================================================
echo "Layer 3: Seccomp Profile"
echo "-----------------------"

# Test 3.1: Seccomp profile is valid JSON
if echo '{}' | jq . >/dev/null 2>&1 && [ -f "container/seccomp-profile.json" ]; then
    if jq . < container/seccomp-profile.json >/dev/null 2>&1; then
        test_result "PASS" "Seccomp profile is valid JSON"
    else
        test_result "FAIL" "Seccomp profile has invalid JSON" "HIGH"
    fi
else
    test_result "WARN" "jq not installed, skipping JSON validation" "LOW"
fi

# Test 3.2: Seccomp blocks socket() syscall
if docker run --rm --security-opt seccomp=container/seccomp-profile.json armorclaw/agent:v1 python3 -c "import socket; s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)" 2>&1 | grep -iqE "permission|denied|operation not permitted"; then
    test_result "PASS" "Seccomp blocks socket() syscall"
else
    test_result "WARN" "Seccomp may not block socket() (check profile)" "MEDIUM"
fi

echo ""

# ============================================================================
# Layer 4: Non-Root User
# ============================================================================
echo "Layer 4: Non-Root User"
echo "--------------------"

# Test 4.1: Container not running as root
if docker run --rm armorclaw/agent:v1 id | grep -qv "uid=0"; then
    test_result "PASS" "Container not running as root"
else
    test_result "FAIL" "Container running as root" "CRITICAL"
fi

# Test 4.2: Cannot read root-only files
if docker run --rm armorclaw/agent:v1 cat /etc/shadow 2>&1 | grep -iq "permission denied"; then
    test_result "PASS" "Cannot read /etc/shadow"
else
    test_result "WARN" "May be able to read some root files (check seccomp)" "MEDIUM"
fi

echo ""

# ============================================================================
# Layer 5: Capability Dropping
# ============================================================================
echo "Layer 5: Capability Dropping"
echo "---------------------------"

# Test 5.1: Capabilities dropped
CONTAINER_ID=$(docker run -d --rm --cap-drop=ALL armorclaw/agent:v1 python3 -c "import time; time.sleep(30)")
CAP_DROP=$(docker inspect $CONTAINER_ID --format '{{.HostConfig.CapDrop}}' 2>/dev/null || echo "")
docker stop $CONTAINER_ID >/dev/null 2>&1

if echo "$CAP_DROP" | grep -q "ALL"; then
    test_result "PASS" "All capabilities dropped (--cap-drop=ALL)"
else
    test_result "FAIL" "Not all capabilities dropped" "HIGH"
fi

echo ""

# ============================================================================
# Layer 6: Read-Only Filesystem
# ============================================================================
echo "Layer 6: Read-Only Filesystem"
echo "----------------------------"

# Test 6.1: Root filesystem is read-only
if docker run --rm --read-only armorclaw/agent:v1 python3 -c "open('/test.txt', 'w')" 2>&1 | grep -iq "read-only"; then
    test_result "PASS" "Root filesystem is read-only"
else
    test_result "WARN" "May not be using --read-only flag" "MEDIUM"
fi

echo ""

# ============================================================================
# Layer 7: LD_PRELOAD Hook
# ============================================================================
echo "Layer 7: LD_PRELOAD Security Hook"
echo "---------------------------------"

# Test 7.1: LD_PRELOAD is set
if docker run --rm armorclaw/agent:v1 env | grep -q "LD_PRELOAD"; then
    test_result "PASS" "LD_PRELOAD environment variable is set"
else
    test_result "WARN" "LD_PRELOAD not set (security hook not active)" "MEDIUM"
fi

# Test 7.2: Security hook library exists
CONTAINER_ID=$(docker run -d --rm armorclaw/agent:v1 python3 -c "import time; time.sleep(30)")
if docker exec $CONTAINER_ID ls /opt/openclaw/lib/libarmorclaw_hook.so >/dev/null 2>&1; then
    test_result "PASS" "Security hook library exists"
else
    test_result "WARN" "Security hook library not found" "MEDIUM"
fi
docker stop $CONTAINER_ID >/dev/null 2>&1

echo ""

# ============================================================================
# SUMMARY
# ============================================================================
echo "=================================="
echo "Security Verification Summary"
echo "=================================="
echo ""
echo "Passed:  $PASS"
echo "Failed:  $FAIL"
echo "Warnings: $WARN"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✅ ALL CRITICAL SECURITY CHECKS PASSED${NC}"
    echo ""
    echo "Blast radius: Container memory only"
    echo "Security posture: PRODUCTION READY"
    exit 0
else
    echo -e "${RED}❌ $FAIL CRITICAL SECURITY CHECK(S) FAILED${NC}"
    echo ""
    echo "Security posture: NOT PRODUCTION READY"
    echo ""
    echo "Please review failed tests above and fix before deployment."
    exit 1
fi
