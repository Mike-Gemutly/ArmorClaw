#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BRIDGE_BIN="${SCRIPT_DIR}/../bridge/build/armorclaw-bridge"
SOCKET_PATH="/tmp/bridge-test-$$.sock"
BRIDGE_PID=""
FAILED=0

cleanup() {
    if [[ -n "$BRIDGE_PID" ]]; then
        kill "$BRIDGE_PID" 2>/dev/null || true
        wait "$BRIDGE_PID" 2>/dev/null || true
    fi
    rm -f "$SOCKET_PATH" 2>/dev/null || true
}
trap cleanup EXIT

if ! command -v socat &>/dev/null; then
    echo "❌ socat not installed. Install with: apt-get install socat"
    exit 1
fi

if [[ ! -x "$BRIDGE_BIN" ]]; then
    echo "Building bridge binary..."
    cd "${SCRIPT_DIR}/../bridge"
    go build -o build/armorclaw-bridge ./cmd/bridge
fi

echo "Starting bridge..."
ARMORCLAW_SOCKET_PATH="$SOCKET_PATH" \
    "$BRIDGE_BIN" &
BRIDGE_PID=$!

echo "Waiting for socket..."
for i in {1..30}; do
    if [[ -S "$SOCKET_PATH" ]]; then
        break
    fi
    sleep 0.5
done

if [[ ! -S "$SOCKET_PATH" ]]; then
    echo "❌ FAILED: Socket not created after 15 seconds"
    exit 1
fi
echo "✅ Socket created at $SOCKET_PATH"

rpc_call() {
    local method="$1"
    local params="${2:-{}}"
    local timeout="${3:-5}"
    
    timeout "$timeout" bash -c \
        "echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$method\",\"params\":$params}' | \
         socat - UNIX-CONNECT:$SOCKET_PATH" 2>/dev/null || echo ""
}

# Test function
test_method() {
    local method="$1"
    local expected="$2"
    local params="${3:-{}}"
    
    result=$(rpc_call "$method" "$params")
    
    if [[ -z "$result" ]]; then
        echo "❌ FAILED: $method - no response"
        return 1
    fi
    
    if echo "$result" | grep -q "$expected"; then
        echo "✅ PASSED: $method"
        return 0
    else
        echo "❌ FAILED: $method - expected '$expected' in response"
        echo "   Got: $result"
        return 1
    fi
}

echo ""
echo "Testing critical RPC methods..."
echo ""

# Critical RPC methods to test
# Format: "method" "expected_in_response" "params"
test_method "matrix.status" '"enabled"' || ((FAILED++)) || true

# Test error handling
echo ""
echo "Testing error handling..."

# Invalid method should return error code -32601
test_method "invalid.method" '"code":-32601' || ((FAILED++)) || true

# Summary
echo ""
echo "========================================"
if [[ $FAILED -eq 0 ]]; then
    echo "🎉 All integration tests passed"
    exit 0
else
    echo "❌ $FAILED test(s) failed"
    exit 1
fi
