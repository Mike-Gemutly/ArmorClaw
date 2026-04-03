#!/bin/bash
# SSH Command Execution Tests
# Tests remote command execution with output capture, timeout handling, exit code handling, and stderr handling

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
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

# Expand SSH key path
SSH_KEY_PATH="${SSH_KEY_PATH/#\~/$HOME}"

# Create evidence directory if it doesn't exist
mkdir -p "$EVIDENCE_DIR"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# JSON output structure
JSON_OUTPUT="{\"test_suite\":\"SSH Command Execution\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"vps_ip\":\"$VPS_IP\",\"vps_user\":\"$VPS_USER\",\"tests\":[]}"

# Function to format result
print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"

    ((TESTS_TOTAL++)) || true

    if [ "$status" = "PASS" ]; then
        ((TESTS_PASSED++)) || true
        echo -e "${GREEN}[PASS]${NC} $test_name: $message"
        JSON_OUTPUT=$(echo "$JSON_OUTPUT" | jq --arg name "$test_name" --arg st "PASS" --arg msg "$message" '.tests += [{"name": $name, "status": $st, "message": $msg}]')
    else
        ((TESTS_FAILED++)) || true
        echo -e "${RED}[FAIL]${NC} $test_name: $message"
        JSON_OUTPUT=$(echo "$JSON_OUTPUT" | jq --arg name "$test_name" --arg st "FAIL" --arg msg "$message" '.tests += [{"name": $name, "status": $st, "message": $msg}]')
    fi
}

# Function to execute remote command and capture output
execute_remote_command() {
    local command="$1"
    local timeout="${2:-10}"

    OUTPUT=$(ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes "$VPS_USER@$VPS_IP" "$command" 2>&1)
    EXIT_CODE=$?

    echo "$OUTPUT"
    return $EXIT_CODE
}

# Function to execute remote command with timeout wrapper
execute_remote_command_with_timeout() {
    local command="$1"
    local timeout="${2:-10}"

    OUTPUT=$(timeout "$timeout" ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes "$VPS_USER@$VPS_IP" "$command" 2>&1)
    EXIT_CODE=$?

    echo "$OUTPUT"
    return $EXIT_CODE
}

# Function to test stdout capture
test_stdout_capture() {
    local command="$1"
    local expected_pattern="$2"

    echo -n "Test 1: Basic remote command execution (echo)... "

    OUTPUT=$(execute_remote_command "echo 'Hello from VPS'")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && echo "$OUTPUT" | grep -q "Hello from VPS"; then
        print_result "Basic Remote Command" "PASS" "Command executed successfully with output: $OUTPUT"
    else
        print_result "Basic Remote Command" "FAIL" "Command failed or output incorrect (exit code: $EXIT_CODE, output: $OUTPUT)"
        return 1
    fi
}

# Function to test command with arguments
test_command_with_arguments() {
    echo -n "Test 2: Command with arguments (ls -la /tmp)... "

    OUTPUT=$(execute_remote_command "ls -la /tmp")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && echo "$OUTPUT" | grep -q "total"; then
        # Verify we got directory listing
        FILE_COUNT=$(echo "$OUTPUT" | wc -l)
        print_result "Command with Arguments" "PASS" "Command executed successfully ($FILE_COUNT lines of output)"
    else
        print_result "Command with Arguments" "FAIL" "Command failed (exit code: $EXIT_CODE)"
        return 1
    fi
}

# Function to test command with pipes
test_command_with_pipes() {
    echo -n "Test 3: Command with pipes (ps aux | head)... "

    OUTPUT=$(execute_remote_command "ps aux | head -5")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && echo "$OUTPUT" | grep -q "USER.*PID.*%CPU.*%MEM.*VSZ.*RSS.*TTY.*STAT.*START.*TIME.*COMMAND"; then
        print_result "Command with Pipes" "PASS" "Pipe command executed successfully"
    else
        print_result "Command with Pipes" "FAIL" "Pipe command failed (exit code: $EXIT_CODE)"
        return 1
    fi
}

# Function to test timeout handling
test_timeout_handling() {
    echo -n "Test 4: Command timeout handling (sleep command with 3s timeout)... "

    # This command should timeout
    OUTPUT=$(execute_remote_command_with_timeout "sleep 10" 3)
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 124 ] || echo "$OUTPUT" | grep -qi "timeout\|timed out"; then
        print_result "Timeout Handling" "PASS" "Command timed out as expected"
    else
        print_result "Timeout Handling" "WARN" "Timeout may not have triggered correctly (exit code: $EXIT_CODE)"
        return 0
    fi
}

# Function to test stdout capture
test_stdout_capture_detailed() {
    echo -n "Test 5: Stdout capture (date command)... "

    OUTPUT=$(execute_remote_command "date")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && [ -n "$OUTPUT" ]; then
        print_result "Stdout Capture" "PASS" "Stdout captured successfully: $OUTPUT"
    else
        print_result "Stdout Capture" "FAIL" "Stdout capture failed"
        return 1
    fi
}

# Function to test stderr capture
test_stderr_capture() {
    echo -n "Test 6: Stderr capture (non-existent command)... "

    OUTPUT=$(execute_remote_command "ls /nonexistent_directory_12345" 2>&1)
    EXIT_CODE=$?

    if [ $EXIT_CODE -ne 0 ] && echo "$OUTPUT" | grep -qi "no such file\|cannot access\|not found"; then
        print_result "Stderr Capture" "PASS" "Stderr captured successfully (exit code: $EXIT_CODE)"
    else
        print_result "Stderr Capture" "WARN" "Stderr capture unclear (exit code: $EXIT_CODE, output: $OUTPUT)"
        return 0
    fi
}

# Function to test exit code handling
test_exit_code_handling() {
    echo -n "Test 7: Exit code handling (successful command)... "

    OUTPUT=$(execute_remote_command "true")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ]; then
        print_result "Exit Code (Success)" "PASS" "Success exit code (0) captured correctly"
    else
        print_result "Exit Code (Success)" "FAIL" "Expected exit code 0, got $EXIT_CODE"
        return 1
    fi
}

# Function to test failure exit code
test_failure_exit_code() {
    echo -n "Test 8: Exit code handling (failed command)... "

    OUTPUT=$(execute_remote_command "false")
    EXIT_CODE=$?

    if [ $EXIT_CODE -ne 0 ]; then
        print_result "Exit Code (Failure)" "PASS" "Failure exit code ($EXIT_CODE) captured correctly"
    else
        print_result "Exit Code (Failure)" "FAIL" "Expected non-zero exit code, got $EXIT_CODE"
        return 1
    fi
}

# Function to test command with environment variables
test_command_with_env() {
    echo -n "Test 9: Command with environment variables... "

    OUTPUT=$(execute_remote_command "echo \$HOME")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && echo "$OUTPUT" | grep -q "root"; then
        print_result "Command with Env Vars" "PASS" "Environment variables accessible"
    else
        print_result "Command with Env Vars" "WARN" "Env vars unclear: $OUTPUT"
        return 0
    fi
}

# Function to test command chaining with &&
test_command_chaining() {
    echo -n "Test 10: Command chaining with &&... "

    OUTPUT=$(execute_remote_command "echo 'first' && echo 'second' && echo 'third'")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && echo "$OUTPUT" | grep -q "first" && echo "$OUTPUT" | grep -q "second" && echo "$OUTPUT" | grep -q "third"; then
        print_result "Command Chaining" "PASS" "Command chaining executed successfully"
    else
        print_result "Command Chaining" "FAIL" "Command chaining failed"
        return 1
    fi
}

# Function to test command output redirection
test_output_redirection() {
    echo -n "Test 11: Command output redirection... "

    OUTPUT=$(execute_remote_command "echo 'test content' > /tmp/test_ssh_output.txt && cat /tmp/test_ssh_output.txt && rm -f /tmp/test_ssh_output.txt")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && echo "$OUTPUT" | grep -q "test content"; then
        print_result "Output Redirection" "PASS" "Output redirection worked correctly"
    else
        print_result "Output Redirection" "FAIL" "Output redirection failed"
        return 1
    fi
}

# Function to test complex command with multiple pipes
test_complex_pipes() {
    echo -n "Test 12: Complex command with multiple pipes... "

    OUTPUT=$(execute_remote_command "ps aux | grep -v grep | head -3")
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ] && [ -n "$OUTPUT" ]; then
        LINE_COUNT=$(echo "$OUTPUT" | wc -l)
        print_result "Complex Pipes" "PASS" "Complex pipe command executed ($LINE_COUNT lines)"
    else
        print_result "Complex Pipes" "WARN" "Complex pipes unclear (exit code: $EXIT_CODE)"
        return 0
    fi
}

echo "========================================="
echo "SSH Command Execution Tests"
echo "========================================="
echo "VPS IP: $VPS_IP"
echo "VPS User: $VPS_USER"
echo "SSH Key: $SSH_KEY_PATH"
echo "========================================="
echo ""

# Run all tests
test_stdout_capture
test_command_with_arguments
test_command_with_pipes
test_timeout_handling
test_stdout_capture_detailed
test_stderr_capture
test_exit_code_handling
test_failure_exit_code
test_command_with_env
test_command_chaining
test_output_redirection
test_complex_pipes

# Generate summary
echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Total Tests: $TESTS_TOTAL"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"

JSON_OUTPUT=$(echo "$JSON_OUTPUT" | jq --arg total "$TESTS_TOTAL" --arg passed "$TESTS_PASSED" --arg failed "$TESTS_FAILED" '. + {"total_tests": $total | tonumber, "passed": $passed | tonumber, "failed": $failed | tonumber}')

# Save JSON output
JSON_FILE="$EVIDENCE_DIR/task-3-command-execution-results.json"
echo "$JSON_OUTPUT" | jq '.' > "$JSON_FILE"
echo -e "${CYAN}JSON results saved to $JSON_FILE${NC}"

# Save console output evidence
CONSOLE_FILE="$EVIDENCE_DIR/task-3-command-success.txt"
echo "SSH Command Execution Test Results - $(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$CONSOLE_FILE"
echo "VPS IP: $VPS_IP" >> "$CONSOLE_FILE"
echo "VPS User: $VPS_USER" >> "$CONSOLE_FILE"
echo "" >> "$CONSOLE_FILE"
echo "Total Tests: $TESTS_TOTAL" >> "$CONSOLE_FILE"
echo "Passed: $TESTS_PASSED" >> "$CONSOLE_FILE"
echo "Failed: $TESTS_FAILED" >> "$CONSOLE_FILE"
echo -e "${CYAN}Console output saved to $CONSOLE_FILE${NC}"

# Save detailed test evidence
DETAILED_FILE="$EVIDENCE_DIR/task-3-command-details.txt"
echo "=========================================" > "$DETAILED_FILE"
echo "SSH Command Execution Test Details" >> "$DETAILED_FILE"
echo "=========================================" >> "$DETAILED_FILE"
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$DETAILED_FILE"
echo "VPS IP: $VPS_IP" >> "$DETAILED_FILE"
echo "VPS User: $VPS_USER" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

# Re-run tests and capture detailed output
echo "--- Test 1: Basic Remote Command ---" >> "$DETAILED_FILE"
OUTPUT1=$(execute_remote_command "echo 'Hello from VPS'")
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT1" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo "--- Test 2: Command with Arguments ---" >> "$DETAILED_FILE"
OUTPUT2=$(execute_remote_command "ls -la /tmp | head -5")
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT2" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo "--- Test 3: Command with Pipes ---" >> "$DETAILED_FILE"
OUTPUT3=$(execute_remote_command "ps aux | head -5")
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT3" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo "--- Test 4: Timeout Handling ---" >> "$DETAILED_FILE"
OUTPUT4=$(execute_remote_command_with_timeout "sleep 10" 3)
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT4" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo "--- Test 5: Stdout Capture ---" >> "$DETAILED_FILE"
OUTPUT5=$(execute_remote_command "date")
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT5" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo "--- Test 6: Stderr Capture ---" >> "$DETAILED_FILE"
OUTPUT6=$(execute_remote_command "ls /nonexistent_directory_12345" 2>&1)
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT6" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo "--- Test 7: Exit Code (Success) ---" >> "$DETAILED_FILE"
OUTPUT7=$(execute_remote_command "true")
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT7" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo "--- Test 8: Exit Code (Failure) ---" >> "$DETAILED_FILE"
OUTPUT8=$(execute_remote_command "false")
echo "Exit Code: $?" >> "$DETAILED_FILE"
echo "Output: $OUTPUT8" >> "$DETAILED_FILE"
echo "" >> "$DETAILED_FILE"

echo -e "${CYAN}Detailed test output saved to $DETAILED_FILE${NC}"

echo ""
echo "========================================="
echo "Command Execution Tests Complete"
echo "========================================="

# Exit with appropriate code
if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi
exit 0
