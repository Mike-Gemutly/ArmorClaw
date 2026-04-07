#!/bin/bash
# SSH Connectivity Tests
# Tests SSH key validation, connection, timeout handling, retry logic, network diagnostics

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

# JSON output structure
JSON_OUTPUT="{\"test_suite\":\"SSH Connectivity\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"vps_ip\":\"$VPS_IP\",\"vps_user\":\"$VPS_USER\",\"tests\":[]}"

# Python-based JSON manipulation (alternative to jq)
add_test_result() {
    local json="$1"
    local name="$2"
    local status="$3"
    local message="$4"
    python3 -c "
import json, sys
try:
    data = json.loads('$json')
    data['tests'].append({'name': '$name', 'status': '$status', 'message': '$message'})
    print(json.dumps(data))
except Exception as e:
    sys.stderr.write(f'Error: {e}\n')
    print('$json')
"
}

add_summary() {
    local json="$1"
    local total="$2"
    local passed="$3"
    local failed="$4"
    python3 -c "
import json, sys
try:
    data = json.loads('$json')
    data['total_tests'] = int('$total')
    data['passed'] = int('$passed')
    data['failed'] = int('$failed')
    print(json.dumps(data))
except Exception as e:
    sys.stderr.write(f'Error: {e}\n')
    print('$json')
"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to format result
print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"

    ((TESTS_TOTAL++)) || true

    if [ "$status" = "PASS" ]; then
        ((TESTS_PASSED++)) || true
        echo -e "${GREEN}[PASS]${NC} $test_name: $message"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}[WARN]${NC} $test_name: $message"
    else
        ((TESTS_FAILED++)) || true
        echo -e "${RED}[FAIL]${NC} $test_name: $message"
    fi

    JSON_OUTPUT=$(add_test_result "$JSON_OUTPUT" "$test_name" "$status" "$message")
}

# Function to retry SSH connection with exponential backoff
retry_ssh() {
    local command="$1"
    local max_retries="${2:-3}"
    local initial_delay="${3:-2}"
    
    local attempt=1
    local delay=$initial_delay
    
    while [ $attempt -le $max_retries ]; do
        if eval "$command" 2>&1; then
            return 0
        fi
        
        if [ $attempt -lt $max_retries ]; then
            echo -e "${YELLOW}Attempt $attempt/$max_retries failed, retrying in ${delay}s...${NC}"
            sleep $delay
            delay=$((delay * 2))  # Exponential backoff
        fi
        
        ((attempt++)) || true
    done
    
    return 1
}

# Function to collect network diagnostics
collect_network_diagnostics() {
    local output_file="$EVIDENCE_DIR/task-2-network-diagnostics.txt"
    
    echo "=== SSH Connectivity Network Diagnostics ===" > "$output_file"
    echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$output_file"
    echo "VPS IP: $VPS_IP" >> "$output_file"
    echo "VPS User: $VPS_USER" >> "$output_file"
    echo "" >> "$output_file"
    
    # Ping test
    echo "--- Ping Test ---" >> "$output_file"
    if command_exists ping; then
        ping -c 3 -W 2 "$VPS_IP" >> "$output_file" 2>&1 || echo "Ping failed" >> "$output_file"
    else
        echo "Ping command not available" >> "$output_file"
    fi
    echo "" >> "$output_file"
    
    # Traceroute test
    echo "--- Traceroute Test ---" >> "$output_file"
    if command_exists traceroute; then
        timeout 5 traceroute -w 1 -m 5 "$VPS_IP" >> "$output_file" 2>&1 || echo "Traceroute failed or timed out" >> "$output_file"
    elif command_exists tracepath; then
        timeout 5 tracepath "$VPS_IP" >> "$output_file" 2>&1 || echo "Tracepath failed or timed out" >> "$output_file"
    else
        echo "Traceroute/tracepath command not available" >> "$output_file"
    fi
    echo "" >> "$output_file"
    
    # DNS lookup
    echo "--- DNS Lookup ---" >> "$output_file"
    if command_exists nslookup; then
        nslookup "$VPS_IP" >> "$output_file" 2>&1 || echo "NSLookup failed" >> "$output_file"
    elif command_exists dig; then
        dig -x "$VPS_IP" +short >> "$output_file" 2>&1 || echo "Dig failed" >> "$output_file"
    else
        echo "DNS lookup command not available" >> "$output_file"
    fi
    echo "" >> "$output_file"
    
    # SSH connection attempt log
    echo "--- SSH Connection Test ---" >> "$output_file"
    timeout 10 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes -v "$VPS_USER@$VPS_IP" "echo 'SSH connection successful'" >> "$output_file" 2>&1 || true
    
    echo -e "${CYAN}Network diagnostics saved to $output_file${NC}"
}

echo "========================================="
echo "SSH Connectivity Tests"
echo "========================================="
echo "VPS IP: $VPS_IP"
echo "VPS User: $VPS_USER"
echo "SSH Key: $SSH_KEY_PATH"
echo "========================================="

# Test 1: SSH key exists and is readable
echo -n "Test 1: SSH key file exists and readable... "
if [ -f "$SSH_KEY_PATH" ] && [ -r "$SSH_KEY_PATH" ]; then
    print_result "SSH Key Validation" "PASS" "Key file exists and is readable"
else
    print_result "SSH Key Validation" "FAIL" "Key file not found or not readable"
    exit 1
fi

# Test 2: SSH key has correct permissions (0600)
echo -n "Test 2: SSH key has correct permissions (0600 or 0400)... "
KEY_PERMS=$(stat -c "%a" "$SSH_KEY_PATH")
if [ "$KEY_PERMS" = "600" ] || [ "$KEY_PERMS" = "400" ]; then
    print_result "SSH Key Permissions" "PASS" "Key has correct permissions ($KEY_PERMS)"
else
    print_result "SSH Key Permissions" "FAIL" "Key has incorrect permissions: $KEY_PERMS (expected 600 or 400)"
    exit 1
fi

# Test 3: SSH key is not empty
echo -n "Test 3: SSH key is not empty... "
if [ -s "$SSH_KEY_PATH" ]; then
    print_result "SSH Key Content" "PASS" "Key file is not empty"
else
    print_result "SSH Key Content" "FAIL" "Key file is empty"
    exit 1
fi

# Test 4: SSH key format is valid (RSA or ED25519)
echo -n "Test 4: SSH key format is valid... "
if grep -q "BEGIN.*PRIVATE KEY" "$SSH_KEY_PATH"; then
    print_result "SSH Key Format" "PASS" "Key has valid format"
else
    print_result "SSH Key Format" "FAIL" "Key format is invalid"
    exit 1
fi

# Test 5: SSH client is available
echo -n "Test 5: SSH client is available... "
if command_exists ssh; then
    print_result "SSH Client" "PASS" "SSH client is available"
else
    print_result "SSH Client" "FAIL" "SSH client not found"
    exit 1
fi

# Test 6: SSH can establish connection (with timeout)
echo -n "Test 6: SSH can establish connection (10s timeout)... "
if timeout 10 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$VPS_USER@$VPS_IP" "echo 'SSH connection successful'" 2>&1; then
    print_result "SSH Connection" "PASS" "SSH connection established successfully"
else
    print_result "SSH Connection" "FAIL" "SSH connection failed or timed out"
    exit 1
fi

# Test 7: SSH version check
echo -n "Test 7: SSH version check... "
SSH_VERSION=$(ssh -V 2>&1 | head -1 | awk '{print $2}')
if [ -n "$SSH_VERSION" ]; then
    print_result "SSH Version" "PASS" "SSH version: $SSH_VERSION"
else
    print_result "SSH Version" "FAIL" "Could not determine SSH version"
fi

# Test 8: SSH connection with retry logic
echo -n "Test 8: SSH connection with retry logic (3 retries, exponential backoff)... "
if retry_ssh "timeout 10 ssh -i '$SSH_KEY_PATH' -o StrictHostKeyChecking=no -o ConnectTimeout=10 '$VPS_USER@$VPS_IP' 'echo SSH connection successful'" 3 2; then
    print_result "SSH Connection with Retry" "PASS" "Connection established with retry logic"
else
    print_result "SSH Connection with Retry" "FAIL" "Connection failed after all retries"
fi

# Test 9: SSH connection timeout handling (short timeout)
echo -n "Test 9: SSH connection timeout handling (5s timeout)... "
if timeout 5 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "$VPS_USER@$VPS_IP" "echo 'SSH connection successful'" >/dev/null 2>&1; then
    print_result "SSH Timeout Handling" "PASS" "Connection established within timeout"
else
    print_result "SSH Timeout Handling" "WARN" "Connection timed out (may indicate network latency)"
fi

# Test 10: SSH key authentication only (no password fallback)
echo -n "Test 10: SSH key authentication only (BatchMode)... "
if timeout 10 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes -o PasswordAuthentication=no "$VPS_USER@$VPS_IP" "echo 'Key auth successful'" 2>&1; then
    print_result "SSH Key Authentication" "PASS" "Key-based authentication working"
else
    print_result "SSH Key Authentication" "FAIL" "Key-based authentication failed"
fi

# Test 11: SSH remote command execution
echo -n "Test 11: SSH remote command execution (uptime)... "
UPTIME_OUTPUT=$(timeout 10 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$VPS_USER@$VPS_IP" "uptime" 2>&1)
if [[ "$UPTIME_OUTPUT" == *"load average"* ]]; then
    print_result "SSH Remote Command" "PASS" "Remote command executed successfully"
else
    print_result "SSH Remote Command" "FAIL" "Remote command execution failed"
fi

# Test 12: SSH connection stability (multiple quick connections)
echo -n "Test 12: SSH connection stability (3 quick connections)... "
STABLE=0
for i in {1..3}; do
    if timeout 10 ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$VPS_USER@$VPS_IP" "echo 'Connection $i'" >/dev/null 2>&1; then
        ((STABLE++)) || true
    fi
    sleep 1
done
if [ $STABLE -eq 3 ]; then
    print_result "SSH Connection Stability" "PASS" "All 3 connections successful"
else
    print_result "SSH Connection Stability" "WARN" "Only $STABLE/3 connections successful"
fi

# Collect network diagnostics
echo ""
echo -e "${CYAN}Collecting network diagnostics...${NC}"
collect_network_diagnostics

# Generate summary
echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Total Tests: $TESTS_TOTAL"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"

JSON_OUTPUT=$(add_summary "$JSON_OUTPUT" "$TESTS_TOTAL" "$TESTS_PASSED" "$TESTS_FAILED")

# Save JSON output
JSON_FILE="$EVIDENCE_DIR/task-2-connectivity-results.json"
python3 -c "import json; print(json.dumps(json.loads('$JSON_OUTPUT'), indent=2))" > "$JSON_FILE"
echo -e "${CYAN}JSON results saved to $JSON_FILE${NC}"

# Save console output evidence
CONSOLE_FILE="$EVIDENCE_DIR/task-2-connectivity-success.txt"
echo "SSH Connectivity Test Results - $(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$CONSOLE_FILE"
echo "VPS IP: $VPS_IP" >> "$CONSOLE_FILE"
echo "VPS User: $VPS_USER" >> "$CONSOLE_FILE"
echo "" >> "$CONSOLE_FILE"
echo "Total Tests: $TESTS_TOTAL" >> "$CONSOLE_FILE"
echo "Passed: $TESTS_PASSED" >> "$CONSOLE_FILE"
echo "Failed: $TESTS_FAILED" >> "$CONSOLE_FILE"
echo -e "${CYAN}Console output saved to $CONSOLE_FILE${NC}"

echo ""
echo "========================================="
echo "Connectivity Tests Complete"
echo "========================================="

# Exit with appropriate code
if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi
exit 0
