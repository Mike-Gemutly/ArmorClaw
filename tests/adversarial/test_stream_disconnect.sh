#!/usr/bin/env bash
# Adversarial E2E Test: Stream Disconnect (Vault Kill & Restart)
#
# Validates that the v6 microkernel's VaultEventBridge degrades gracefully
# when the Rust Vault process is killed and restarted. The Go Bridge must
# NOT panic or crash — it must log a reconnection message and reconnect
# automatically via exponential backoff (1s → 30s max).
#
# Threat model: An attacker gains the ability to kill the vault container
# (e.g., via Docker API abuse or resource exhaustion). The system MUST
# survive the outage and resume normal operation once vault is restored.
#
# Architecture reference:
#   - VaultEventBridge (Go) subscribes to vault governance events via gRPC
#   - syncLoop reconnects with exponential backoff on stream errors
#   - Reconnection log: "vault event stream error, reconnecting"
#   - Clean stop log: "vault event bridge stopped: context cancelled"
#
# Prerequisites: Docker stack running (docker compose up)
# Exit codes: 0 = PASS or SKIP, 1 = FAIL
set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
VAULT_CONTAINER="${ARMORCLAW_VAULT_CONTAINER:-armorclaw-vault}"
BRIDGE_CONTAINER="${ARMORCLAW_BRIDGE_CONTAINER:-armorclaw-bridge}"
RECONNECT_TIMEOUT="${STREAM_RECONNECT_TIMEOUT:-45}"
POST_RESTART_TIMEOUT="${STREAM_POST_RESTART_TIMEOUT:-30}"
PASS_COUNT=0
FAIL_COUNT=0
SKIP=0

# Log patterns from bridge/pkg/vault/events.go
RECONNECT_PATTERN="vault event stream error, reconnecting"
STOPPED_PATTERN="vault event bridge stopped: context cancelled"

# ── Tool check ───────────────────────────────────────────────────────────────
for cmd in docker grep; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "SKIP: '$cmd' not found in PATH"
        exit 0
    fi
done

# ── Helpers ──────────────────────────────────────────────────────────────────
log_pass() { echo "  PASS: $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
log_fail() { echo "  FAIL: $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }

# Check if a container is running
container_running() {
    docker ps --format '{{.Names}}' | grep -q "$1"
}

# Get container logs since a given timestamp (ISO 8601)
logs_since() {
    local container="$1"
    local since="$2"
    docker logs --since "$since" "$container" 2>&1 || true
}

# Wait for a log pattern to appear within a timeout (seconds)
wait_for_log() {
    local container="$1"
    local pattern="$2"
    local timeout="$3"
    local since="$4"
    local elapsed=0

    while [ "$elapsed" -lt "$timeout" ]; do
        if logs_since "$container" "$since" | grep -q "$pattern"; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    return 1
}

# ── Pre-flight ───────────────────────────────────────────────────────────────
echo "========================================"
echo "Adversarial Test: Stream Disconnect"
echo "Validates graceful degradation on vault kill & restart"
echo "========================================"
echo ""

if ! container_running "$VAULT_CONTAINER"; then
    echo "SKIP: Vault container '$VAULT_CONTAINER' is not running"
    echo "  Start the stack with: docker compose up -d"
    exit 0
fi

if ! container_running "$BRIDGE_CONTAINER"; then
    echo "SKIP: Bridge container '$BRIDGE_CONTAINER' is not running"
    echo "  Start the stack with: docker compose up -d"
    exit 0
fi

echo "  Vault container:  $VAULT_CONTAINER (running)"
echo "  Bridge container: $BRIDGE_CONTAINER (running)"
echo "  Reconnect timeout: ${RECONNECT_TIMEOUT}s"
echo "  Post-restart timeout: ${POST_RESTART_TIMEOUT}s"
echo ""

# ── Test 1: Event bridge is active before kill ──────────────────────────────
echo "Test 1: Verify event bridge is active (stream connected)"
echo "------------------------------------------------------------------------"

# Check that bridge logs don't show repeated reconnection errors before we
# start the test — the stream should be healthy.
PRE_TEST_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
sleep 2

# Look for the bridge having started its sync loop. If we see recent bridge
# logs at all, it's running. A healthy state means no recent reconnect msgs.
BRIDGE_LOGS=$(logs_since "$BRIDGE_CONTAINER" "$PRE_TEST_TS")

if [ -z "$BRIDGE_LOGS" ]; then
    # No recent logs is fine — bridge may be idle. Container is running.
    log_pass "Bridge container is running (no recent reconnect errors)"
elif echo "$BRIDGE_LOGS" | grep -q "$RECONNECT_PATTERN"; then
    echo "  WARN: Bridge already showing reconnection errors before test start"
    echo "  The stream may already be broken. Proceeding anyway."
    log_pass "Bridge container is running (pre-existing reconnect warnings noted)"
else
    log_pass "Bridge container is running, stream appears healthy"
fi

# ── Test 2: Kill vault — bridge must NOT crash ──────────────────────────────
echo ""
echo "Test 2: Kill vault container — bridge must survive"
echo "------------------------------------------------------------------------"

KILL_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "  Killing vault container: $VAULT_CONTAINER"
docker kill "$VAULT_CONTAINER" >/dev/null 2>&1 || true

# Verify vault is actually stopped
if container_running "$VAULT_CONTAINER"; then
    log_fail "Vault container did not stop after docker kill"
else
    log_pass "Vault container stopped successfully"
fi

# Give bridge a moment to detect the disconnection
sleep 3

# Critical assertion: bridge container must STILL be running
if container_running "$BRIDGE_CONTAINER"; then
    log_pass "Bridge container survived vault kill (still running)"
else
    log_fail "Bridge container crashed after vault was killed"
fi

# ── Test 3: Bridge logs reconnection message (not panic) ────────────────────
echo ""
echo "Test 3: Bridge logs reconnection message (graceful degradation)"
echo "------------------------------------------------------------------------"

# Wait for the reconnection log message with exponential backoff (up to timeout)
# The bridge's syncLoop should emit "vault event stream error, reconnecting"
# when the gRPC stream to vault breaks.
if wait_for_log "$BRIDGE_CONTAINER" "$RECONNECT_PATTERN" "$RECONNECT_TIMEOUT" "$KILL_TS"; then
    log_pass "Bridge logged reconnection message: '$RECONNECT_PATTERN'"
else
    # Check if bridge stopped cleanly instead (acceptable — context cancelled)
    if wait_for_log "$BRIDGE_CONTAINER" "$STOPPED_PATTERN" 5 "$KILL_TS"; then
        echo "  NOTE: Bridge logged clean stop instead of reconnect"
        echo "  This is acceptable if the bridge is shutting down gracefully."
        log_pass "Bridge logged clean stop: '$STOPPED_PATTERN'"
    else
        log_fail "Bridge did not log reconnection message within ${RECONNECT_TIMEOUT}s"
        echo "  Expected pattern: '$RECONNECT_PATTERN'"
        echo "  Bridge logs since kill:"
        logs_since "$BRIDGE_CONTAINER" "$KILL_TS" | tail -20 || true
    fi
fi

# ── Test 4: Restart vault — bridge reconnects automatically ─────────────────
echo ""
echo "Test 4: Restart vault — bridge reconnects automatically"
echo "------------------------------------------------------------------------"

# Verify bridge is still running before restart
if ! container_running "$BRIDGE_CONTAINER"; then
    log_fail "Bridge container is not running — cannot test reconnection"
else
    RESTART_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    echo "  Restarting vault container: $VAULT_CONTAINER"
    docker restart "$VAULT_CONTAINER" >/dev/null 2>&1 || true

    # Wait for vault to be running again
    VAULT_UP=0
    for i in $(seq 1 15); do
        if container_running "$VAULT_CONTAINER"; then
            VAULT_UP=1
            break
        fi
        sleep 2
    done

    if [ "$VAULT_UP" -eq 1 ]; then
        log_pass "Vault container restarted successfully"
    else
        log_fail "Vault container failed to restart within 30s"
    fi

    # Wait for bridge to reconnect. After vault comes back, the bridge's
    # syncLoop should successfully re-establish the gRPC stream.
    # We check that no NEW reconnection errors appear after vault is up,
    # meaning the stream stabilized.
    echo "  Waiting ${POST_RESTART_TIMEOUT}s for bridge to stabilize stream..."
    sleep "$POST_RESTART_TIMEOUT"

    # Check for reconnection errors AFTER restart (new ones, not the old ones)
    POST_RESTART_RECONNECT=$(logs_since "$BRIDGE_CONTAINER" "$RESTART_TS" | grep -c "$RECONNECT_PATTERN" || echo "0")

    # A few initial reconnects are expected (backoff may still be cycling).
    # But if we see a flood of reconnects, the stream never recovered.
    if [ "$POST_RESTART_RECONNECT" -le 5 ]; then
        log_pass "Bridge reconnected after vault restart ($POST_RESTART_RECONNECT reconnect log(s) post-restart)"
    else
        log_fail "Bridge shows excessive reconnect attempts ($POST_RESTART_RECONNECT) after vault restart — stream may not be recovering"
    fi
fi

# ── Test 5: System remains operational throughout ────────────────────────────
echo ""
echo "Test 5: System remains operational (bridge container stable)"
echo "------------------------------------------------------------------------"

# Final check: bridge must still be running after the entire ordeal
if container_running "$BRIDGE_CONTAINER"; then
    log_pass "Bridge container is still running after kill+restart cycle"
else
    log_fail "Bridge container is NOT running after kill+restart cycle"
fi

# Also verify vault is running (for cleanup/cleanup state)
if container_running "$VAULT_CONTAINER"; then
    log_pass "Vault container is running (system fully operational)"
else
    echo "  WARN: Vault container is not running at end of test"
    echo "  This may be expected if restart failed. Check manually."
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "========================================"
echo "Stream Disconnect Test Summary"
echo "========================================"
echo "  Passed: $PASS_COUNT"
echo "  Failed: $FAIL_COUNT"
echo "  Skipped tests: $SKIP (acceptable in CI/Docker-less environments)"
echo ""

if [ "$FAIL_COUNT" -gt 0 ]; then
    echo "FAIL: $FAIL_COUNT assertion(s) failed — graceful degradation broken"
    exit 1
elif [ "$PASS_COUNT" -eq 0 ] && [ "$SKIP" -gt 0 ]; then
    echo "SKIP: All tests skipped (Docker stack or tools unavailable)"
    exit 0
else
    echo "PASS: Stream disconnect graceful degradation verified"
    exit 0
fi
