#!/bin/bash
# Deployment Mode Tests
# Tests deployment mode detection: Native, Sentinel, Cloudflare Tunnel, Cloudflare Proxy

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

EVIDENCE_DIR="$PROJECT_DIR/.sisyphus/evidence"

if [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    echo -e "${RED}Error: .env file not found${NC}"
    exit 2
fi

# Validate required environment variables
if [ -z "\$VPS_IP" ]; then
    echo -e "\${RED}Error: VPS_IP not set\${NC}"
    exit 2
fi

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

echo "========================================="
echo "Deployment Mode Tests"
echo "========================================="
echo "VPS IP: \$VPS_IP"
echo "VPS User: \$VPS_USER"
echo "========================================="

# Test 1: Docker Compose files exist
echo -n "Test 1: Docker Compose files exist... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
if [ -f "\$PROJECT_DIR/docker-compose.yml" ] || [ -f "\$PROJECT_DIR/docker-compose-full.yml" ]; then
    echo -e "\${GREEN}[PASS]\${NC} Docker Compose Files: Found"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} Docker Compose Files: Not found (may be ok if using manual setup)"
fi

# Test 2: Deployment mode environment variable
echo -n "Test 2: Deployment mode environment variable... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
if [ -n "\$ARMORCLAW_SERVER_MODE" ]; then
    echo -e "\${GREEN}[PASS]\${NC} Deployment Mode: Set to \$ARMORCLAW_SERVER_MODE"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} Deployment Mode: Not set (will auto-detect)"
fi

# Test 3: Docker daemon is running
echo -n "Test 3: Docker daemon is running... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
if docker info >/dev/null 2>&1; then
    echo -e "\${GREEN}[PASS]\${NC} Docker Daemon: Running"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${RED}[FAIL]\${NC} Docker Daemon: Not running"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

# Test 4: Detect current deployment mode via SSH
echo -n "Test 4: Current deployment mode detection... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
CURRENT_MODE="\$(ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "\$VPS_USER@\$VPS_IP" "
if [ -n "\$CURRENT_MODE" ]; then
    echo -e "\${CYAN}Current Mode\${NC}: \$CURRENT_MODE"
    echo -e "\${GREEN}[PASS]\${NC} Deployment Mode: Detected"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} Deployment Mode: Could not detect"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

# Test 5: Port bindings check (8080, 6167, 8448)
echo -n "Test 5: Port bindings check... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
BRIDGE_PORT="\${BRIDGE_PORT:-8080}"
MATRIX_PORT="\${MATRIX_PORT:-6167}"
PUBLIC_PORT="\${PUBLIC_PORT:-8448}"

# Check Bridge port
BRIDGE_BOUND=false
if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "\$VPS_USER@\$VPS_IP" "
netstat -tuln 2>/dev/null | grep -q ":$BRIDGE_PORT " && BRIDGE_BOUND=true

if \$BRIDGE_BOUND; then
    echo -e "\${GREEN}[PASS]\${NC} Bridge Port ($BRIDGE_PORT): Bound"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} Bridge Port ($BRIDGE_PORT): Not bound (may not be running)"
fi

# Check Matrix port
MATRIX_BOUND=false
if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "\$VPS_USER@\$VPS_IP" "
netstat -tuln 2>/dev/null | grep -q ":$MATRIX_PORT " && MATRIX_BOUND=true

if \$MATRIX_BOUND; then
    echo -e "\${GREEN}[PASS]\${NC} Matrix Port ($MATRIX_PORT): Bound"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} Matrix Port ($MATRIX_PORT): Not bound (may not be running)"
fi

# Check public access port
PUBLIC_BOUND=false
if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "\$VPS_USER@\$VPS_IP" "
netstat -tuln 2>/dev/null | grep -q ":$PUBLIC_PORT " && PUBLIC_BOUND=true

if \$PUBLIC_BOUND; then
    echo -e "\${GREEN}[PASS]\${NC} Public Port ($PUBLIC_PORT): Bound"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} Public Port ($PUBLIC_PORT): Not bound (Sentinel/Proxy mode not active)"
fi

# Test 6: Configuration validation
echo -n "Test 6: Configuration validation... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))

# Check for Native mode indicators
NATIVE_MODE=false
if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "\$VPS_USER@\$VPS_IP" "
[ -f "/run/armorclaw/bridge.sock" ] && NATIVE_MODE=true

if \$NATIVE_MODE; then
    echo -e "\${GREEN}[PASS]\${NC} Native Mode: Detected (Unix socket exists)"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
fi

# Check for Sentinel mode indicators (Caddy reverse proxy)
SENTINEL_MODE=false
if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "\$VPS_USER@\$VPS_IP" "
docker ps --filter name=caddy --format "{{.Names}}" | grep -q caddy && SENTINEL_MODE=true

if \$SENTINEL_MODE; then
    echo -e "\${GREEN}[PASS]\${NC} Sentinel Mode: Detected (Caddy running)"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
fi

# Check for Cloudflare Tunnel mode indicators (cloudflared)
CF_TUNNEL_MODE=false
if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "\$VPS_USER@\$VPS_IP" "
docker ps --filter name=cloudflared --format "{{.Names}}" | grep -q cloudflared && CF_TUNNEL_MODE=true

if \$CF_TUNNEL_MODE; then
    echo -e "\${GREEN}[PASS]\${NC} Cloudflare Tunnel Mode: Detected (cloudflared running)"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
fi

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Total Tests: \$TESTS_TOTAL"
echo -e "\${GREEN}Passed: \$TESTS_PASSED\${NC}"
echo -e "\${RED}Failed: \$TESTS_FAILED\${NC}"
echo ""

# Save evidence
mkdir -p "\$EVIDENCE_DIR"
cat > "\$EVIDENCE_DIR/task-8-deployment-results.json" << JSONEOF
{
  "test_suite": "Deployment Modes",
  "timestamp": "\$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "vps_ip": "\$VPS_IP",
  "vps_user": "\$VPS_USER",
  "total_tests": \$TESTS_TOTAL,
  "passed": \$TESTS_PASSED,
  "failed": \$TESTS_FAILED,
  "deployment_mode": "\${ARMORCLAW_SERVER_MODE:-auto}",
  "ports_checked": {
    "bridge": "\$BRIDGE_PORT",
    "matrix": "\$MATRIX_PORT",
    "public": "\$PUBLIC_PORT"
  }
}
JSONEOF

cat > "\$EVIDENCE_DIR/task-8-deployment-success.txt" << CONSOLEEOF
=========================================
Deployment Mode Tests Complete
=========================================

Total Tests: \$TESTS_TOTAL
Passed: \$TESTS_PASSED
Failed: \$TESTS_FAILED

Mode Detected: \${CURRENT_MODE:-auto}
Bridge Port: \$BRIDGE_PORT
Matrix Port: \$MATRIX_PORT
Public Port: \$PUBLIC_PORT
=========================================
CONSOLEEOF

echo -e "\${CYAN}Evidence saved to:\${NC} \$EVIDENCE_DIR/task-8-deployment-*.txt"
echo ""
echo "========================================="
echo "Deployment Mode Tests Complete"
echo "========================================="
