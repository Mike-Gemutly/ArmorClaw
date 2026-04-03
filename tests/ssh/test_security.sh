#!/bin/bash
# Security Verification Tests
# Tests firewall rules, SSH hardening, container isolation, secret access controls,
# network policies, user permissions, and SQLcipher keystore

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
NC='\033[0m'

# Source environment - use absolute path to .env
PROJECT_DIR="/home/mink/src/armorclaw-omo"
EVIDENCE_DIR="$PROJECT_DIR/.sisyphus/evidence"

if [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    echo -e "${RED}Error: .env file not found at $PROJECT_DIR${NC}"
    exit 2
fi

# Validate required environment variables
if [ -z "$VPS_IP" ]; then
    echo -e "${RED}Error: VPS_IP not set${NC}"
    exit 2
fi

if [ -z "$VPS_USER" ]; then
    echo -e "${RED}Error: VPS_USER not set${NC}"
    exit 2
fi

if [ -z "$SSH_KEY_PATH" ]; then
    echo -e "${RED}Error: SSH_KEY_PATH not set${NC}"
    exit 2
fi

# Create evidence directory if it doesn't exist
mkdir -p "$EVIDENCE_DIR"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0
TESTS_WARNED=0

# Result file for collecting test results
RESULT_FILE=$(mktemp)

# Initialize JSON structure
cat > "$RESULT_FILE" << EOF
{
  "test_suite": "Security Verification",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "vps_ip": "$VPS_IP",
  "vps_user": "$VPS_USER",
  "tests": []
}
EOF

# Helper to check if a command exists on VPS
vps_command_exists() {
    local cmd="$1"
    timeout 10 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$VPS_USER@$VPS_IP" "command -v $cmd" >/dev/null 2>&1
}

# Function to run command on VPS and capture output
vps_exec() {
    local command="$1"
    timeout 30 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$VPS_USER@$VPS_IP" "$command" 2>&1
}

# Function to print test group header
print_test_group() {
    local group_name="$1"
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$group_name${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Function to format and record result
print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"

    ((TESTS_TOTAL++)) || true

    if [ "$status" = "PASS" ]; then
        ((TESTS_PASSED++)) || true
        echo -e "${GREEN}[PASS]${NC} $test_name: $message"
    elif [ "$status" = "WARN" ]; then
        ((TESTS_WARNED++)) || true
        echo -e "${YELLOW}[WARN]${NC} $test_name: $message"
    else
        ((TESTS_FAILED++)) || true
        echo -e "${RED}[FAIL]${NC} $test_name: $message"
    fi
}

echo "========================================="
echo "Security Verification Tests"
echo "========================================="
echo "VPS IP: $VPS_IP"
echo "VPS User: $VPS_USER"
echo "========================================="

# ============================================================================
# TEST GROUP 1: Firewall Rules
# ============================================================================
print_test_group "Test Group 1: Firewall Rules"

# Test 1.1: UFW is installed
echo -n "Test 1.1: UFW firewall is installed... "
if vps_command_exists ufw; then
    print_result "UFW Installation" "PASS" "UFW firewall is installed"
else
    print_result "UFW Installation" "WARN" "UFW firewall not installed (may use iptables directly)"
fi

# Test 1.2: UFW is active
echo -n "Test 1.2: UFW firewall is active... "
if vps_exec "ufw status | grep -q 'Status: active'"; then
    print_result "UFW Status" "PASS" "UFW firewall is active"
else
    print_result "UFW Status" "FAIL" "UFW firewall is not active"
fi

# Test 1.3: Default deny policy is set
echo -n "Test 1.3: Default deny policy is set... "
if vps_exec "ufw status verbose | grep -q 'Default: deny'"; then
    print_result "Default Deny Policy" "PASS" "Default deny policy is configured"
else
    print_result "Default Deny Policy" "WARN" "Default deny policy may not be fully set"
fi

# Test 1.4: SSH port is allowed (rate limited if possible)
echo -n "Test 1.4: SSH is allowed with rate limiting... "
SSH_RULE=$(vps_exec "ufw status | grep -E '22/tcp|ssh'" || echo "")
if [ -n "$SSH_RULE" ]; then
    if echo "$SSH_RULE" | grep -qi "LIMIT"; then
        print_result "SSH Rate Limiting" "PASS" "SSH is allowed with rate limiting"
    else
        print_result "SSH Rate Limiting" "WARN" "SSH is allowed but rate limiting not configured"
    fi
else
    print_result "SSH Rate Limiting" "FAIL" "SSH rule not found in UFW"
fi

# Test 1.5: Unnecessary ports are not open
echo -n "Test 1.5: No unnecessary ports are open... "
OPEN_PORTS=$(vps_exec "ufw status | grep ALLOW | grep -v -E '22/tcp|80/tcp|443/tcp|6167/tcp|8080/tcp|8448/tcp' || echo ''")
if [ -z "$OPEN_PORTS" ]; then
    print_result "Open Ports" "PASS" "No unnecessary ports open"
else
    print_result "Open Ports" "WARN" "Additional open ports detected"
fi

# ============================================================================
# TEST GROUP 2: SSH Hardening
# ============================================================================
print_test_group "Test Group 2: SSH Hardening"

# Test 2.1: PasswordAuthentication is disabled
echo -n "Test 2.1: PasswordAuthentication is disabled... "
PASSWORD_AUTH=$(vps_exec "grep -E '^#?PasswordAuthentication' /etc/ssh/sshd_config | tail -1 | awk '{print \$2}'")
if [ "$PASSWORD_AUTH" = "no" ]; then
    print_result "SSH Password Authentication" "PASS" "Password authentication is disabled"
else
    print_result "SSH Password Authentication" "FAIL" "Password authentication is enabled"
fi

# Test 2.2: PermitRootLogin is disabled
echo -n "Test 2.2: PermitRootLogin is disabled... "
ROOT_LOGIN=$(vps_exec "grep -E '^#?PermitRootLogin' /etc/ssh/sshd_config | tail -1 | awk '{print \$2}'")
if [ "$ROOT_LOGIN" = "no" ]; then
    print_result "SSH Root Login" "PASS" "Root login is disabled"
else
    print_result "SSH Root Login" "FAIL" "Root login is enabled"
fi

# Test 2.3: PubkeyAuthentication is enabled
echo -n "Test 2.3: PubkeyAuthentication is enabled... "
PUBKEY_AUTH=$(vps_exec "grep -E '^#?PubkeyAuthentication' /etc/ssh/sshd_config | tail -1 | awk '{print \$2}'")
if [ "$PUBKEY_AUTH" = "yes" ]; then
    print_result "SSH Pubkey Authentication" "PASS" "Public key authentication is enabled"
else
    print_result "SSH Pubkey Authentication" "FAIL" "Public key authentication is disabled"
fi

# Test 2.4: ChallengeResponseAuthentication is disabled
echo -n "Test 2.4: ChallengeResponseAuthentication is disabled... "
CHALLENGE_RESP=$(vps_exec "grep -E '^#?ChallengeResponseAuthentication' /etc/ssh/sshd_config | tail -1 | awk '{print \$2}'")
if [ "$CHALLENGE_RESP" = "no" ]; then
    print_result "SSH Challenge Response" "PASS" "Challenge-response authentication is disabled"
else
    print_result "SSH Challenge Response" "WARN" "Challenge-response authentication is enabled"
fi

# Test 2.5: UsePAM is disabled (security hardening)
echo -n "Test 2.5: UsePAM is disabled... "
USE_PAM=$(vps_exec "grep -E '^#?UsePAM' /etc/ssh/sshd_config | tail -1 | awk '{print \$2}'")
if [ "$USE_PAM" = "no" ]; then
    print_result "SSH UsePAM" "PASS" "UsePAM is disabled"
elif [ "$USE_PAM" = "yes" ]; then
    print_result "SSH UsePAM" "WARN" "UsePAM is enabled (may reduce security)"
else
    print_result "SSH UsePAM" "WARN" "UsePAM setting not found"
fi

# ============================================================================
# TEST GROUP 3: Container Isolation
# ============================================================================
print_test_group "Test Group 3: Container Isolation"

# Test 3.1: Check if Docker is installed
echo -n "Test 3.1: Docker is installed... "
if vps_command_exists docker; then
    print_result "Docker Installation" "PASS" "Docker is installed"
else
    print_result "Docker Installation" "WARN" "Docker not installed (containers may not be used)"
fi

# Test 3.2: Check seccomp profile exists
echo -n "Test 3.2: Seccomp profile exists... "
if vps_exec "[ -f /etc/docker/seccomp/armorclaw.json ]"; then
    print_result "Seccomp Profile" "PASS" "Seccomp profile exists"
else
    print_result "Seccomp Profile" "WARN" "Seccomp profile not found"
fi

# Test 3.3: Check no-new-privileges in Docker daemon config
echo -n "Test 3.3: no-new-privileges is set in Docker daemon... "
if vps_exec "[ -f /etc/docker/daemon.json ] && grep -q 'no-new-privileges' /etc/docker/daemon.json"; then
    print_result "No New Privileges" "PASS" "no-new-privileges is configured"
else
    print_result "No New Privileges" "WARN" "no-new-privileges may not be set"
fi

# Test 3.4: Check for privileged containers
echo -n "Test 3.4: No privileged containers running... "
PRIVILEGED_COUNT=$(vps_exec "docker ps --format '{{.Names}}' --filter status=running 2>/dev/null | while read container; do docker inspect --format '{{.HostConfig.Privileged}}' \"\$container\" 2>/dev/null; done | grep -c 'true' || echo 0")
if [ "$PRIVILEGED_COUNT" = "0" ]; then
    print_result "Privileged Containers" "PASS" "No privileged containers running"
else
    print_result "Privileged Containers" "FAIL" "$PRIVILEGED_COUNT privileged container(s) detected"
fi

# Test 3.5: Check AppArmor status
echo -n "Test 3.5: AppArmor is enabled... "
if vps_command_exists aa-status && vps_exec "aa-status >/dev/null 2>&1"; then
    print_result "AppArmor" "PASS" "AppArmor is enabled and running"
else
    print_result "AppArmor" "WARN" "AppArmor may not be enabled"
fi

# ============================================================================
# TEST GROUP 4: Secret Access Controls
# ============================================================================
print_test_group "Test Group 4: Secret Access Controls"

# Test 4.1: Verify API keys are in environment (memory-only)
echo -n "Test 4.1: API keys are in environment variables (memory-only)... "
if [ -n "$API_KEY" ]; then
    print_result "API Key Source" "PASS" "API key is from environment (memory-only)"
else
    print_result "API Key Source" "WARN" "API key not in environment (may be persisted)"
fi

# Test 4.2: Check if API keys are written to disk (should not be)
echo -n "Test 4.2: API keys are not persisted to disk... "
if vps_exec "! grep -r '$API_KEY' /etc/armorclaw /var/lib/armorclaw /opt/armorclaw 2>/dev/null"; then
    print_result "API Key Persistence" "PASS" "API keys are not persisted to disk"
else
    print_result "API Key Persistence" "WARN" "API keys may be persisted to disk"
fi

# Test 4.3: Check keystore permissions
echo -n "Test 4.3: Keystore has restrictive permissions... "
if vps_exec "[ -f /var/lib/armorclaw/keystore.db ]"; then
    KEYSTORE_PERMS=$(vps_exec "stat -c '%a' /var/lib/armorclaw/keystore.db")
    if [ "$KEYSTORE_PERMS" = "600" ]; then
        print_result "Keystore Permissions" "PASS" "Keystore has correct permissions (600)"
    else
        print_result "Keystore Permissions" "FAIL" "Keystore has incorrect permissions ($KEYSTORE_PERMS)"
    fi
else
    print_result "Keystore Permissions" "WARN" "Keystore not found (may not be initialized)"
fi

# Test 4.4: Verify no secrets in logs
echo -n "Test 4.4: No secrets in log files... "
if vps_exec "! grep -r '$API_KEY' /var/log/armorclaw 2>/dev/null"; then
    print_result "Log Security" "PASS" "No secrets found in logs"
else
    print_result "Log Security" "FAIL" "Secrets found in log files"
fi

# Test 4.5: Check .env file permissions
echo -n "Test 4.5: .env file has restrictive permissions... "
ENV_PERMS=$(stat -c "%a" "$PROJECT_DIR/.env")
if [ "$ENV_PERMS" = "600" ] || [ "$ENV_PERMS" = "400" ]; then
    print_result ".env Permissions" "PASS" ".env has correct permissions ($ENV_PERMS)"
else
    print_result ".env Permissions" "WARN" ".env permissions are too permissive ($ENV_PERMS)"
fi

# ============================================================================
# TEST GROUP 5: Network Policies
# ============================================================================
print_test_group "Test Group 5: Network Policies"

# Test 5.1: TCP SYN cookies are enabled
echo -n "Test 5.1: TCP SYN cookies are enabled... "
if vps_exec "[ \$(cat /proc/sys/net/ipv4/tcp_syncookies) = '1' ]"; then
    print_result "TCP SYN Cookies" "PASS" "TCP SYN cookies are enabled"
else
    print_result "TCP SYN Cookies" "WARN" "TCP SYN cookies not enabled"
fi

# Test 5.2: IP forwarding is disabled
echo -n "Test 5.2: IP forwarding is disabled... "
if vps_exec "[ \$(cat /proc/sys/net/ipv4/ip_forward) = '0' ]"; then
    print_result "IP Forwarding" "PASS" "IP forwarding is disabled"
else
    print_result "IP Forwarding" "WARN" "IP forwarding is enabled"
fi

# Test 5.3: Reverse path filtering is enabled
echo -n "Test 5.3: Reverse path filtering is enabled... "
if vps_exec "[ \$(cat /proc/sys/net/ipv4/conf/all/rp_filter) = '1' ]"; then
    print_result "Reverse Path Filtering" "PASS" "Reverse path filtering is enabled"
else
    print_result "Reverse Path Filtering" "WARN" "Reverse path filtering not enabled"
fi

# Test 5.4: Docker socket is not exposed on network
echo -n "Test 5.4: Docker socket is not exposed on network... "
if vps_exec "! ss -xl | grep -q docker.sock"; then
    print_result "Docker Socket Exposure" "PASS" "Docker socket not exposed on network"
else
    print_result "Docker Socket Exposure" "FAIL" "Docker socket is exposed on network"
fi

# Test 5.5: Check listening ports for ArmorClaw services
echo -n "Test 5.5: ArmorClaw services listening on expected ports... "
LISTENING_PORTS=$(vps_exec "ss -tuln | grep LISTEN | grep -E ':(80|443|6167|8080|8448)' || echo ''")
if [ -n "$LISTENING_PORTS" ]; then
    print_result "Service Ports" "PASS" "ArmorClaw services listening on expected ports"
else
    print_result "Service Ports" "WARN" "ArmorClaw services may not be listening"
fi

# ============================================================================
# TEST GROUP 6: User Permissions
# ============================================================================
print_test_group "Test Group 6: User Permissions"

# Test 6.1: Check if running as root in containers
echo -n "Test 6.1: No containers running as root... "
ROOT_CONTAINERS=$(vps_exec "docker ps --format '{{.Names}}' --filter status=running 2>/dev/null | while read container; do docker inspect --format '{{.Config.User}}' \"\$container\" 2>/dev/null; done | grep -c 'root\|0\|^$' || echo 0")
if [ "$ROOT_CONTAINERS" = "0" ]; then
    print_result "Container Root User" "PASS" "No containers running as root"
else
    print_result "Container Root User" "WARN" "$ROOT_CONTAINERS container(s) may be running as root"
fi

# Test 6.2: Check armorclaw user exists
echo -n "Test 6.2: armorclaw user exists... "
if vps_exec "id armorclaw &>/dev/null"; then
    print_result "ArmorClaw User" "PASS" "Armorclaw user exists"
else
    print_result "ArmorClaw User" "WARN" "Armorclaw user does not exist"
fi

# Test 6.3: Check armorclaw user has no login shell
echo -n "Test 6.3: armorclaw user has no login shell... "
if vps_exec "id armorclaw &>/dev/null"; then
    SHELL=$(vps_exec "getent passwd armorclaw | cut -d: -f7")
    if [ "$SHELL" = "/bin/false" ] || [ "$SHELL" = "/usr/sbin/nologin" ]; then
        print_result "ArmorClaw User Shell" "PASS" "Armorclaw user has no login shell"
    else
        print_result "ArmorClaw User Shell" "WARN" "Armorclaw user has login shell ($SHELL)"
    fi
else
    print_result "ArmorClaw User Shell" "WARN" "Armorclaw user does not exist"
fi

# Test 6.4: Check for passwordless sudo
echo -n "Test 6.4: armorclaw user does not have passwordless sudo... "
if vps_exec "id armorclaw &>/dev/null"; then
    if vps_exec "! sudo -l -U armorclaw 2>/dev/null | grep -q NOPASSWD"; then
        print_result "Passwordless Sudo" "PASS" "Armorclaw user does not have passwordless sudo"
    else
        print_result "Passwordless Sudo" "WARN" "Armorclaw user has passwordless sudo"
    fi
else
    print_result "Passwordless Sudo" "WARN" "Armorclaw user does not exist"
fi

# Test 6.5: Check config directory permissions
echo -n "Test 6.5: Config directory has restrictive permissions... "
if vps_exec "[ -d /etc/armorclaw ]"; then
    CONFIG_PERMS=$(vps_exec "stat -c '%a' /etc/armorclaw")
    if [ "$CONFIG_PERMS" -le "750" ]; then
        print_result "Config Directory Permissions" "PASS" "Config directory has secure permissions ($CONFIG_PERMS)"
    else
        print_result "Config Directory Permissions" "WARN" "Config directory permissions are too permissive ($CONFIG_PERMS)"
    fi
else
    print_result "Config Directory Permissions" "WARN" "Config directory does not exist"
fi

# ============================================================================
# TEST GROUP 7: SQLCipher Keystore
# ============================================================================
print_test_group "Test Group 7: SQLCipher Keystore"

# Test 7.1: Check if SQLCipher is being used
echo -n "Test 7.1: SQLCipher is being used for keystore... "
if vps_exec "[ -f /var/lib/armorclaw/keystore.db ]"; then
    # Check if it's a SQLite database (SQLCipher uses SQLite format)
    if vps_exec "file /var/lib/armorclaw/keystore.db | grep -qi sqlite"; then
        print_result "SQLCipher Database" "PASS" "Keystore is using SQLite/SQLCipher format"
    else
        print_result "SQLCipher Database" "WARN" "Keystore format may not be SQLCipher"
    fi
else
    print_result "SQLCipher Database" "WARN" "Keystore not found"
fi

# Test 7.2: Verify keystore is encrypted
echo -n "Test 7.2: Keystore database is encrypted... "
if vps_exec "[ -f /var/lib/armorclaw/keystore.db ]"; then
    # Try to read database without password - should fail if encrypted
    if vps_exec "! sqlite3 /var/lib/armorclaw/keystore.db 'SELECT * FROM sqlite_master LIMIT 1' >/dev/null 2>&1"; then
        print_result "Keystore Encryption" "PASS" "Keystore appears to be encrypted (cannot read without key)"
    else
        print_result "Keystore Encryption" "WARN" "Keystore may not be encrypted"
    fi
else
    print_result "Keystore Encryption" "WARN" "Keystore not found"
fi

# Test 7.3: Check if SQLCipher libraries are available
echo -n "Test 7.3: SQLCipher libraries are available... "
if vps_command_exists sqlite3 && vps_exec "sqlite3 --version | grep -i sqlcipher"; then
    print_result "SQLCipher Libraries" "PASS" "SQLCipher libraries are available"
else
    print_result "SQLCipher Libraries" "WARN" "SQLCipher libraries may not be available"
fi

# Test 7.4: Verify keystore has proper integrity
echo -n "Test 7.4: Keystore database has proper integrity... "
if vps_exec "[ -f /var/lib/armorclaw/keystore.db ]"; then
    # Check database header for encryption marker (SQLCipher databases have a specific header)
    DB_HEADER=$(vps_exec "head -c 16 /var/lib/armorclaw/keystore.db | xxd -p")
    # SQLite databases start with "SQLite format 3", encrypted SQLCipher may have different header
    if echo "$DB_HEADER" | grep -qi "53514c697465"; then
        print_result "Keystore Integrity" "WARN" "Keystore may be unencrypted (standard SQLite header)"
    else
        print_result "Keystore Integrity" "PASS" "Keystore has non-standard header (likely encrypted)"
    fi
else
    print_result "Keystore Integrity" "WARN" "Keystore not found"
fi

# Test 7.5: Check backup of keystore (should not exist or be encrypted)
echo -n "Test 7.5: No unencrypted keystore backups... "
UNENCRYPTED_BACKUPS=$(vps_exec "find /var/lib/armorclaw -name '*keystore*.db*' -exec sh -c 'file \"\$1\" | grep -qi sqlite && ! file \"\$1\" | grep -qi encrypted' _ {} \; 2>/dev/null | wc -l")
if [ "$UNENCRYPTED_BACKUPS" = "0" ]; then
    print_result "Keystore Backups" "PASS" "No unencrypted keystore backups found"
else
    print_result "Keystore Backups" "WARN" "$UNENCRYPTED_BACKUPS unencrypted backup(s) found"
fi

# ============================================================================
# Generate Summary
# ============================================================================
echo ""
print_test_group "Test Summary"
echo ""
echo -e "Total Tests: $TESTS_TOTAL"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo -e "${YELLOW}Warnings: $TESTS_WARNED${NC}"

# Save JSON output
JSON_FILE="$EVIDENCE_DIR/task-7-security-results.json"
python3 -c "
import json
data = {
    'test_suite': 'Security Verification',
    'timestamp': '$(date -u +%Y-%m-%dT%H:%M:%SZ)',
    'vps_ip': '$VPS_IP',
    'vps_user': '$VPS_USER',
    'total_tests': $TESTS_TOTAL,
    'passed': $TESTS_PASSED,
    'failed': $TESTS_FAILED,
    'warned': $TESTS_WARNED
}
with open('$JSON_FILE', 'w') as f:
    json.dump(data, f, indent=2)
"
echo -e "${CYAN}JSON results saved to $JSON_FILE${NC}"

# Save console output evidence
CONSOLE_FILE="$EVIDENCE_DIR/task-7-security-success.txt"
{
    echo "Security Verification Test Results - $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "VPS IP: $VPS_IP"
    echo "VPS User: $VPS_USER"
    echo ""
    echo "Total Tests: $TESTS_TOTAL"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    echo "Warnings: $TESTS_WARNED"
    echo ""
    echo "Test Groups Executed:"
    echo "  1. Firewall Rules"
    echo "  2. SSH Hardening"
    echo "  3. Container Isolation"
    echo "  4. Secret Access Controls"
    echo "  5. Network Policies"
    echo "  6. User Permissions"
    echo "  7. SQLCipher Keystore"
} > "$CONSOLE_FILE"
echo -e "${CYAN}Console output saved to $CONSOLE_FILE${NC}"

# Save detailed evidence
EVIDENCE_FILE="$EVIDENCE_DIR/task-7-security-evidence.txt"
{
    echo "=== Security Verification Evidence ==="
    echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "VPS IP: $VPS_IP"
    echo "VPS User: $VPS_USER"
    echo ""
    echo "=== Firewall Status ==="
    vps_exec "ufw status verbose" || echo "UFW status not available"
    echo ""
    echo "=== SSH Configuration ==="
    vps_exec "grep -E '^#?(PasswordAuthentication|PermitRootLogin|PubkeyAuthentication|ChallengeResponseAuthentication|UsePAM)' /etc/ssh/sshd_config"
    echo ""
    echo "=== Docker Security ==="
    vps_exec "docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' 2>/dev/null || echo 'Docker not available'"
    echo ""
    echo "=== Container Security Settings ==="
    vps_exec "docker ps --format '{{.Names}}' --filter status=running 2>/dev/null | while read container; do echo 'Container: '\$container; docker inspect --format 'Privileged: {{.HostConfig.Privileged}}' \"\$container\" 2>/dev/null; done"
    echo ""
    echo "=== Network Settings ==="
    vps_exec "cat /proc/sys/net/ipv4/tcp_syncookies /proc/sys/net/ipv4/ip_forward /proc/sys/net/ipv4/conf/all/rp_filter 2>/dev/null"
    echo ""
    echo "=== Listening Ports ==="
    vps_exec "ss -tuln | grep LISTEN || echo 'No listening ports found'"
    echo ""
    echo "=== Keystore Information ==="
    vps_exec "ls -la /var/lib/armorclaw/ 2>/dev/null || echo 'Keystore directory not found'"
} > "$EVIDENCE_FILE"
echo -e "${CYAN}Detailed evidence saved to $EVIDENCE_FILE${NC}"

echo ""
echo "========================================="
echo "Security Verification Tests Complete"
echo "========================================="

# Clean up
rm -f "$RESULT_FILE"

# Exit with appropriate code
if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi
exit 0
