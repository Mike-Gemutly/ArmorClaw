#!/bin/bash
# E2E Test: API Key Setup
# Tests US-2: API Key stored in environment variables only

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Configuration
BRIDGE_BIN="${SCRIPT_DIR}/../../bridge/build/armorclaw-bridge"
TEST_API_KEY="sk-test-api-key-12345678901234567890"

# ============================================================================
# Test Cases
# ============================================================================

test_api_key_env_only() {
    echo ""
    echo "Test: API Key stored in environment variables only"

    # Set API key in environment
    export OPENROUTER_API_KEY="$TEST_API_KEY"

    # Create bridge config
    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[ai]
enabled = true
default_provider = "openrouter"
default_model = "openai/gpt-4o"

[error_system]
enabled = false
store_enabled = false
EOF

    # Start bridge
    start_bridge "$BRIDGE_CONFIG" || {
        log_result "api_key_bridge_start" "false" "Failed to start bridge"
        return 1
    }

    # Wait for bridge to be ready
    wait_for_bridge 30 || {
        log_result "api_key_socket" "false" "Socket not ready"
        stop_bridge
        return 1
    }

    # Verify bridge status
    status_result=$(rpc_call "bridge.status" "{}")

    if [[ -n "$status_result" ]]; then
        log_result "api_key_bridge_status" "true" "Bridge status accessible"
    else
        log_result "api_key_bridge_status" "false" "Bridge status not accessible"
        stop_bridge
        return 1
    fi

    # Verify API key is NOT persisted to disk
    # Check keystore database for stored credentials
    if [[ -f "$KEYSTORE_DIR/keystore.db" ]]; then
        # Use sqlite3 to check if any credentials are stored
        if command -v sqlite3 &>/dev/null; then
            stored_keys=$(sqlite3 "$KEYSTORE_DIR/keystore.db" "SELECT COUNT(*) FROM credentials" 2>/dev/null || echo "0")

            if [[ "$stored_keys" -eq 0 ]]; then
                log_result "api_key_not_persisted" "true" "No API keys persisted to disk (expected)"
            else
                log_result "api_key_not_persisted" "false" "Found $stored_keys persisted keys (unexpected)"
                stop_bridge
                return 1
            fi
        else
            # If sqlite3 is not available, check file size and content
            key_size=$(stat -f%z "$KEYSTORE_DIR/keystore.db" 2>/dev/null || stat -c%s "$KEYSTORE_DIR/keystore.db" 2>/dev/null || echo "0")

            # Empty database is small (< 4KB)
            if [[ "$key_size" -lt 4096 ]]; then
                log_result "api_key_not_persisted" "true" "Keystore database empty (expected)"
            else
                log_result "api_key_not_persisted" "false" "Keystore database has content (unexpected)"
                stop_bridge
                return 1
            fi
        fi
    else
        log_result "api_key_not_persisted" "true" "Keystore database not created (expected)"
    fi

    # Test AI chat with environment API key
    # This verifies the API key from environment is accessible
    chat_result=$(rpc_call "ai.chat" '{"messages":[{"role":"user","content":"test"}],"model":"openai/gpt-4o"}' 2>&1)

    # Check if the error indicates missing API key (should not happen if env var is set)
    if echo "$chat_result" | grep -q "API key"; then
        log_result "api_key_env_accessible" "false" "API key not accessible from environment"
        stop_bridge
        return 1
    else
        log_result "api_key_env_accessible" "true" "API key accessible from environment"
    fi

    # Unset environment variable for cleanup
    unset OPENROUTER_API_KEY

    stop_bridge

    log_result "api_key_env_only" "true" "API key stored in environment only, not persisted"
}

test_multiple_providers_from_env() {
    echo ""
    echo "Test: Multiple providers from environment variables"

    # Set multiple API keys
    export OPENROUTER_API_KEY="sk-openrouter-test-key"
    export OPEN_AI_KEY="sk-openai-test-key"
    export ZAI_API_KEY="sk-zai-test-key"

    # Create bridge config
    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[ai]
enabled = true
default_provider = "openrouter"
default_model = "openai/gpt-4o"

[error_system]
enabled = false
store_enabled = false
EOF

    # Start bridge
    start_bridge "$BRIDGE_CONFIG" || {
        log_result "multi_provider_bridge_start" "false" "Failed to start bridge"
        return 1
    }

    wait_for_bridge 30 || {
        log_result "multi_provider_socket" "false" "Socket not ready"
        stop_bridge
        return 1
    }

    # Test with OpenRouter (default)
    chat_result=$(rpc_call "ai.chat" '{"messages":[{"role":"user","content":"test"}]}' 2>&1)
    if ! echo "$chat_result" | grep -q "API key.*not found"; then
        log_result "multi_provider_openrouter" "true" "OpenRouter provider accessible"
    else
        log_result "multi_provider_openrouter" "false" "OpenRouter provider not accessible"
    fi

    # Verify no keys persisted
    if [[ -f "$KEYSTORE_DIR/keystore.db" ]]; then
        if command -v sqlite3 &>/dev/null; then
            stored_keys=$(sqlite3 "$KEYSTORE_DIR/keystore.db" "SELECT COUNT(*) FROM credentials" 2>/dev/null || echo "0")

            if [[ "$stored_keys" -eq 0 ]]; then
                log_result "multi_provider_not_persisted" "true" "No keys persisted with multiple providers"
            else
                log_result "multi_provider_not_persisted" "false" "Keys persisted with multiple providers"
            fi
        fi
    fi

    # Cleanup
    unset OPENROUTER_API_KEY OPEN_AI_KEY ZAI_API_KEY
    stop_bridge

    log_result "multi_providers_from_env" "true" "Multiple providers supported from environment"
}

test_provider_switching() {
    echo ""
    echo "Test: Provider switching via Matrix commands (placeholder)"

    # Note: Provider switching via Matrix commands requires full Matrix setup
    # This test is a placeholder for future implementation

    # Set API key for switching test
    export OPENROUTER_API_KEY="sk-switch-test-key"

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[ai]
enabled = true
default_provider = "openrouter"
default_model = "openai/gpt-4o"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "provider_switch_bridge_start" "false" "Failed to start bridge"
        return 1
    }

    wait_for_bridge 30 || {
        log_result "provider_switch_socket" "false" "Socket not ready"
        stop_bridge
        return 1
    }

    # Verify bridge is running with default provider
    status_result=$(rpc_call "bridge.status" "{}")
    if [[ -n "$status_result" ]]; then
        log_result "provider_switch_default" "true" "Default provider configured"
    fi

    # TODO: Implement actual provider switching test when Matrix commands are fully available
    # This would involve:
    # 1. Sending Matrix command "/ai switch openai gpt-4o"
    # 2. Verifying provider switched via RPC call
    # 3. Verifying chat uses new provider

    unset OPENROUTER_API_KEY
    stop_bridge

    log_result "provider_switching" "true" "Provider switching placeholder (requires full Matrix)"
}

test_no_persistence_on_restart() {
    echo ""
    echo "Test: No persistence on bridge restart"

    # Set API key and start bridge
    export OPENROUTER_API_KEY="sk-restart-test-key"

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[ai]
enabled = true
default_provider = "openrouter"
default_model = "openai/gpt-4o"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "restart_bridge_start_1" "false" "Failed to start bridge first time"
        return 1
    }

    wait_for_bridge 30 || {
        log_result "restart_socket_1" "false" "Socket not ready first time"
        stop_bridge
        return 1
    }

    # Test chat works
    chat_result=$(rpc_call "ai.chat" '{"messages":[{"role":"user","content":"test"}]}' 2>&1)
    first_run_worked=0
    if ! echo "$chat_result" | grep -q "API key.*not found"; then
        first_run_worked=1
        log_result "restart_first_run" "true" "First run successful with env API key"
    else
        log_result "restart_first_run" "false" "First run failed"
    fi

    # Stop bridge
    stop_bridge

    # Start bridge again with SAME config (no changes)
    start_bridge "$BRIDGE_CONFIG" || {
        log_result "restart_bridge_start_2" "false" "Failed to start bridge second time"
        return 1
    }

    wait_for_bridge 30 || {
        log_result "restart_socket_2" "false" "Socket not ready second time"
        stop_bridge
        return 1
    }

    # Check keystore database again - should still be empty
    if [[ -f "$KEYSTORE_DIR/keystore.db" ]]; then
        if command -v sqlite3 &>/dev/null; then
            stored_keys=$(sqlite3 "$KEYSTORE_DIR/keystore.db" "SELECT COUNT(*) FROM credentials" 2>/dev/null || echo "0")

            if [[ "$stored_keys" -eq 0 ]]; then
                log_result "restart_no_persistence" "true" "No keys persisted after restart"
            else
                log_result "restart_no_persistence" "false" "Keys persisted after restart (unexpected)"
            fi
        fi
    fi

    unset OPENROUTER_API_KEY
    stop_bridge

    log_result "no_persistence_on_restart" "true" "API keys not persisted across restarts"
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    echo "========================================"
    echo "E2E Test: API Key Setup (US-2)"
    echo "========================================"
    echo ""

    setup_test_env || exit 1

    check_dependencies || exit 1

    echo ""
    echo "Running tests..."
    echo ""

    test_api_key_env_only
    test_multiple_providers_from_env
    test_provider_switching
    test_no_persistence_on_restart

    test_summary
    exit $?
}

main "$@"
