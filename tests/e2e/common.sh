#!/bin/bash
# E2E Test Common Functions
# Shared utilities for end-to-end tests

set -euo pipefail

# Color codes
export GREEN='\033[0;32m'
export RED='\033[0;31m'
export YELLOW='\033[1;33m'
export NC='\033[0m' # No Color

# Test tracking
export TESTS_RUN=0
export TESTS_PASSED=0
export TESTS_FAILED=0
export TEST_NS="test-e2e-$(date +%s)"
export TEST_DIR="/tmp/armorclaw-$TEST_NS"

# Bridge state
export BRIDGE_PID=""
export BRIDGE_BIN=""
export BRIDGE_SOCKET="/tmp/bridge-$TEST_NS.sock"
export BRIDGE_CONFIG="/tmp/bridge-config-$TEST_NS.toml"
export KEYSTORE_DIR="/tmp/armorclaw-keystore-$TEST_NS"

# Matrix state
export MATRIX_CONTAINER=""
export MATRIX_PID=""

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up test artifacts...${NC}"

    if [[ -n "$BRIDGE_PID" ]]; then
        kill "$BRIDGE_PID" 2>/dev/null || true
        wait "$BRIDGE_PID" 2>/dev/null || true
        BRIDGE_PID=""
    fi

    if [[ -n "$BRIDGE_BIN" && -f "$BRIDGE_BIN" ]]; then
        pkill -f "$BRIDGE_BIN" 2>/dev/null || true
    fi

    if [[ -n "$MATRIX_CONTAINER" ]]; then
        docker stop "$MATRIX_CONTAINER" 2>/dev/null || true
        docker rm "$MATRIX_CONTAINER" 2>/dev/null || true
        MATRIX_CONTAINER=""
    fi

    rm -f "$BRIDGE_SOCKET" "$BRIDGE_CONFIG" 2>/dev/null || true
    rm -rf "$KEYSTORE_DIR" "$TEST_DIR" 2>/dev/null || true

    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# ============================================================================
# Bridge Management
# ============================================================================

start_bridge() {
    local config_file="${1:-$BRIDGE_CONFIG}"

    echo -e "${YELLOW}Starting bridge with config: $config_file${NC}"

    mkdir -p "$KEYSTORE_DIR"

    if [[ ! -f "$BRIDGE_BIN" ]]; then
        echo -e "${RED}✗ Bridge binary not found: $BRIDGE_BIN${NC}"
        return 1
    fi

    ARMORCLAW_ERRORS_STORE_PATH="$KEYSTORE_DIR/errors.db" \
        ARMORCLAW_SKIP_DOCKER_CHECK=1 \
        "$BRIDGE_BIN" --config "$config_file" &
    BRIDGE_PID=$!

    echo -e "${GREEN}✓ Bridge started (PID: $BRIDGE_PID)${NC}"
    return 0
}

stop_bridge() {
    echo -e "${YELLOW}Stopping bridge...${NC}"

    if [[ -n "$BRIDGE_PID" ]]; then
        kill "$BRIDGE_PID" 2>/dev/null || true
        wait "$BRIDGE_PID" 2>/dev/null || true
        BRIDGE_PID=""
        echo -e "${GREEN}✓ Bridge stopped${NC}"
        return 0
    fi

    if [[ -n "$BRIDGE_BIN" && -f "$BRIDGE_BIN" ]]; then
        pkill -f "$BRIDGE_BIN" 2>/dev/null || true
        echo -e "${GREEN}✓ Bridge processes killed${NC}"
        return 0
    fi

    echo -e "${YELLOW}ℹ No bridge process found${NC}"
    return 0
}

wait_for_bridge() {
    local timeout="${1:-30}"
    local socket_path="${2:-$BRIDGE_SOCKET}"

    echo -e "${YELLOW}Waiting for bridge socket: $socket_path${NC}"

    local count=0
    while [[ $count -lt $timeout ]]; do
        if [[ -S "$socket_path" ]]; then
            echo -e "${GREEN}✓ Bridge socket ready${NC}"
            return 0
        fi
        sleep 1
        ((count++)) || true
    done

    echo -e "${RED}✗ Bridge socket not ready after ${timeout}s${NC}"
    return 1
}

# ============================================================================
# Matrix Management
# ============================================================================

wait_for_matrix() {
    local timeout="${1:-30}"
    local matrix_url="${2:-http://localhost:6167}"

    echo -e "${YELLOW}Waiting for Matrix: $matrix_url${NC}"

    local count=0
    while [[ $count -lt $timeout ]]; do
        if curl -s -o /dev/null -w "%{http_code}" "$matrix_url/_matrix/client/versions" 2>/dev/null | grep -q "200"; then
            echo -e "${GREEN}✓ Matrix is ready${NC}"
            return 0
        fi
        sleep 1
        ((count++)) || true
    done

    echo -e "${RED}✗ Matrix not ready after ${timeout}s${NC}"
    return 1
}

start_matrix_container() {
    local image="${1:-matrixconduit/matrix-conduit:latest}"
    local name="e2e-matrix-$TEST_NS"

    echo -e "${YELLOW}Starting Matrix container: $name${NC}"

    MATRIX_CONTAINER=$(
        docker run -d --rm \
            --name "$name" \
            -p 6167:6167 \
            "$image"
    )

    if [[ -n "$MATRIX_CONTAINER" ]]; then
        echo -e "${GREEN}✓ Matrix container started: $MATRIX_CONTAINER${NC}"
        return 0
    fi

    echo -e "${RED}✗ Failed to start Matrix container${NC}"
    return 1
}

# ============================================================================
# Test Utilities
# ============================================================================

log_result() {
    local test_name="$1"
    local passed="$2"
    local message="${3:-}"

    ((TESTS_RUN++)) || true

    if [[ "$passed" == "true" ]]; then
        ((TESTS_PASSED++)) || true
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        if [[ -n "$message" ]]; then
            echo -e "  ${GREEN}  →${NC} $message"
        fi
    else
        ((TESTS_FAILED++)) || true
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        if [[ -n "$message" ]]; then
            echo -e "  ${RED}  →${NC} $message"
        fi
    fi
}

test_summary() {
    echo ""
    echo "========================================"
    echo "Test Summary"
    echo "========================================"
    echo "Total:  $TESTS_RUN"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    echo "========================================"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
        return 0
    else
        echo -e "${RED}✗ SOME TESTS FAILED${NC}"
        return 1
    fi
}

# ============================================================================
# RPC Utilities
# ============================================================================

rpc_call() {
    local method="$1"
    local params="${2:-{\}}"
    local socket="${3:-$BRIDGE_SOCKET}"
    local timeout="${4:-5}"

    if command -v socat &>/dev/null; then
        timeout "$timeout" bash -c \
            "echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$method\",\"params\":$params}' | \
             socat - UNIX-CONNECT:$socket" 2>/dev/null || echo ""
    else
        echo -e "${RED}✗ socat not installed${NC}"
        return 1
    fi
}

# ============================================================================
# Setup
# ============================================================================

setup_test_env() {
    echo -e "${YELLOW}Setting up test environment...${NC}"

    # Create test directory
    mkdir -p "$TEST_DIR"

    # Set trap for cleanup
    trap cleanup EXIT

    echo -e "${GREEN}✓ Test environment ready${NC}"
    echo "  Test namespace: $TEST_NS"
    echo "  Test directory: $TEST_DIR"
}

# ============================================================================
# Validation
# ============================================================================

check_dependencies() {
    local missing=0

    if ! command -v docker &>/dev/null; then
        echo -e "${RED}✗ docker not found${NC}"
        ((missing++)) || true
    fi

    if ! command -v socat &>/dev/null; then
        echo -e "${RED}✗ socat not found${NC}"
        ((missing++)) || true
    fi

    if ! command -v curl &>/dev/null; then
        echo -e "${RED}✗ curl not found${NC}"
        ((missing++)) || true
    fi

    if [[ $missing -gt 0 ]]; then
        echo -e "${RED}✗ Missing $missing required dependencies${NC}"
        return 1
    fi

    echo -e "${GREEN}✓ All dependencies found${NC}"
    return 0
}
