#!/bin/bash
# E2E Test Template
# Copy this file to create new E2E tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Configuration
BRIDGE_BIN="${SCRIPT_DIR}/../../bridge/build/armorclaw-bridge"

# ============================================================================
# Test Cases
# ============================================================================

test_bridge_lifecycle() {
    echo ""
    echo "Test: Bridge Lifecycle"

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "bridge_start" "false" "Failed to start bridge"
        return 1
    }

    wait_for_bridge 30 || {
        log_result "bridge_socket" "false" "Socket not ready"
        stop_bridge
        return 1
    }

    stop_bridge || {
        log_result "bridge_stop" "false" "Failed to stop bridge"
        return 1
    }

    log_result "bridge_lifecycle" "true" "Bridge start/stop cycle successful"
}

test_rpc_communication() {
    echo ""
    echo "Test: RPC Communication"

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "rpc_bridge_start" "false" "Failed to start bridge"
        return 1
    }

    wait_for_bridge 30 || {
        log_result "rpc_socket" "false" "Socket not ready"
        stop_bridge
        return 1
    }

    result=$(rpc_call "system.status" "{}")

    if [[ -n "$result" ]]; then
        log_result "rpc_status" "true" "RPC call successful"
    else
        log_result "rpc_status" "false" "No RPC response"
        stop_bridge
        return 1
    fi

    stop_bridge
}

test_matrix_integration() {
    echo ""
    echo "Test: Matrix Integration Placeholder"

    log_result "matrix_integration" "true" "Placeholder test"
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    echo "========================================"
    echo "E2E Test Suite"
    echo "========================================"
    echo ""

    setup_test_env || exit 1

    check_dependencies || exit 1

    echo ""
    echo "Running tests..."
    echo ""

    test_bridge_lifecycle
    test_rpc_communication
    test_matrix_integration

    test_summary
    exit $?
}

main "$@"
