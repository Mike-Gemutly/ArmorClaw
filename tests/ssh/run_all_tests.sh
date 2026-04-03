#!/bin/bash
# SSH VPS Test Runner - Comprehensive ArmorClaw VPS Testing
# Supports all test categories with CLI interface

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Source environment
PROJECT_DIR="/home/mink/src/armorclaw-omo"
EVIDENCE_DIR="$PROJECT_DIR/.sisyphus/evidence"

if [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    echo -e "${RED}Error: .env file not found${NC}"
    exit 2
fi

# Create evidence directory
mkdir -p "$EVIDENCE_DIR"

# Default values
OUTPUT_FORMAT="console"
RUN_ALL=false
RUN_SPECIFIC=""
VERBOSE=false

# Display help
show_help() {
    cat << HELP
ArmorClaw SSH VPS Test Runner
===========================

Usage: $0 [OPTIONS] [TEST]

Options:
  -a, --all           Run all test categories
  -c, --connectivity  Run SSH connectivity tests only
  -x, --command       Run command execution tests only
  -h, --health        Run container health tests only
  -p, --api           Run API endpoint tests only
  -i, --integration    Run integration tests only
  -s, --security       Run security tests only
  -d, --deployment    Run deployment mode tests only
  -l, --ssl           Run SSL/TLS tests only
  -f, --performance    Run performance tests only
  -o, --output FORMAT Set output format (console|json) [default: console]
  -v, --verbose        Enable verbose output
  --help               Show this help message

Available Tests:
  connectivity       SSH key validation, connection, timeout, retry logic
  command           Remote command execution, output capture, exit codes
  health            Container status, logs, restart, isolation, resources
  api              Bridge RPC, Matrix client, health endpoints
  integration       Bridge↔Matrix, Bridge→Agent, Agent→Browser messaging
  security          Firewall, SSH hardening, container isolation, secrets
  deployment        Native/Sentinel/Cloudflare modes detection
  ssl               Certificate presence, expiry, chain, HTTPS
  performance        SSH speed, API times, container resources

Examples:
  # Run all tests
  $0 --all

  # Run specific test category
  $0 --connectivity

  # Run with JSON output
  $0 --all --output json

  # Run with verbose output
  $0 --all --verbose
HELP
}

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -a|--all)
            RUN_ALL=true
            shift
            ;;
        -c|--connectivity)
            RUN_SPECIFIC="connectivity"
            shift
            ;;
        -x|--command)
            RUN_SPECIFIC="command"
            shift
            ;;
        -h|--health)
            RUN_SPECIFIC="health"
            shift
            ;;
        -p|--api)
            RUN_SPECIFIC="api"
            shift
            ;;
        -i|--integration)
            RUN_SPECIFIC="integration"
            shift
            ;;
        -s|--security)
            RUN_SPECIFIC="security"
            shift
            ;;
        -d|--deployment)
            RUN_SPECIFIC="deployment"
            shift
            ;;
        -l|--ssl)
            RUN_SPECIFIC="ssl"
            shift
            ;;
        -f|--performance)
            RUN_SPECIFIC="performance"
            shift
            ;;
        -o|--output)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

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

# Verbose mode
if $VERBOSE; then
    set -x
fi

# Output format check
if [[ "$OUTPUT_FORMAT" != "console" ]] && [[ "$OUTPUT_FORMAT" != "json" ]]; then
    echo -e "${RED}Error: Invalid output format. Use 'console' or 'json'.${NC}"
    exit 1
fi

# Test counters
TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0

echo -e "${CYAN}=========================================${NC}"
echo -e "${CYAN}ArmorClaw VPS Test Runner${NC}"
echo -e "${CYAN}=========================================${NC}"
echo ""
echo "Configuration:"
echo "  VPS IP: $VPS_IP"
echo "  VPS User: $VPS_USER"
echo "  SSH Key: $SSH_KEY_PATH"
echo "  Output Format: $OUTPUT_FORMAT"
echo "  Verbose: $VERBOSE"
echo ""
echo -e "${CYAN}=========================================${NC}"
echo ""

# Function to run test and track results
run_test() {
    local test_name="$1"
    local test_file="$2"
    
    if $VERBOSE; then
        echo -e "${CYAN}Running: $test_name${NC}"
    fi
    
    if bash "$test_file" > "$EVIDENCE_DIR/${test_name}_output.txt" 2>&1; then
TOTAL_PASSED=$((TOTAL_PASSED + 1))
        if $VERBOSE; then
            echo -e "${GREEN}[PASS]${NC} $test_name"
        else
            echo -e "${GREEN}[PASS]${NC}"
        fi
    else
TOTAL_FAILED=$((TOTAL_FAILED + 1))
        if $VERBOSE; then
            echo -e "${RED}[FAIL]${NC} $test_name"
        else
            echo -e "${RED}[FAIL]${NC}"
        fi
    fi
    
TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# Run tests based on selection
if $RUN_ALL; then
    echo "Running all test categories..."
    run_test "Connectivity" "tests/ssh/test_connectivity.sh"
    run_test "Command" "tests/ssh/test_command_execution.sh"
    run_test "Container Health" "tests/ssh/test_container_health.sh"
    run_test "API Endpoints" "tests/ssh/test_api_endpoints.sh"
    run_test "Integration" "tests/ssh/test_integration.sh"
    run_test "Security" "tests/ssh/test_security.sh"
    run_test "Deployment Modes" "tests/ssh/test_deployment_modes.sh"
    run_test "SSL/TLS" "tests/ssh/test_ssl_tls.sh"
    run_test "Performance" "tests/ssh/test_performance.sh"
elif [ -n "$RUN_SPECIFIC" ]; then
    case "$RUN_SPECIFIC" in
        connectivity)
            run_test "Connectivity" "tests/ssh/test_connectivity.sh"
            ;;
        command)
            run_test "Command" "tests/ssh/test_command_execution.sh"
            ;;
        health)
            run_test "Container Health" "tests/ssh/test_container_health.sh"
            ;;
        api)
            run_test "API Endpoints" "tests/ssh/test_api_endpoints.sh"
            ;;
        integration)
            run_test "Integration" "tests/ssh/test_integration.sh"
            ;;
        security)
            run_test "Security" "tests/ssh/test_security.sh"
            ;;
        deployment)
            run_test "Deployment Modes" "tests/ssh/test_deployment_modes.sh"
            ;;
        ssl)
            run_test "SSL/TLS" "tests/ssh/test_ssl_tls.sh"
            ;;
        performance)
            run_test "Performance" "tests/ssh/test_performance.sh"
            ;;
    esac
else
    echo "No test specified. Use --help for usage."
    exit 1
fi

# Display summary
echo ""
echo -e "${CYAN}=========================================${NC}"
echo -e "${CYAN}Test Summary${NC}"
echo -e "${CYAN}=========================================${NC}"
echo "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $TOTAL_PASSED${NC}"
echo -e "${RED}Failed: $TOTAL_FAILED${NC}"

# Save results to evidence directory
cat > "$EVIDENCE_DIR/task-15-final-results.json" << JSONEOF
{
  "test_suite": "All Tests",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "vps_ip": "$VPS_IP",
  "vps_user": "$VPS_USER",
  "total_tests": $TOTAL_TESTS,
  "passed": $TOTAL_PASSED,
  "failed": $TOTAL_FAILED,
  "output_format": "$OUTPUT_FORMAT",
  "verbose": $VERBOSE
}
JSONEOF

# Exit with proper code
if [ "$TOTAL_FAILED" -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=========================================${NC}"
    echo -e "${GREEN}All tests PASSED!${NC}"
    echo -e "${GREEN}=========================================${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}=========================================${NC}"
    echo -e "${RED}Some tests FAILED${NC}"
    echo -e "${RED}=========================================${NC}"
    exit 1
fi
