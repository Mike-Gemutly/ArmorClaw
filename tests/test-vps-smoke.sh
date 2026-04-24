#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# VPS Integration Smoke Test
#
# Tests a running ArmorClaw deployment via SSH + curl.
# Validates health, auth enforcement, governance RPCs, and discovery endpoint.
#
# Usage:  VPS_IP=1.2.3.4 ADMIN_TOKEN=xxx bash tests/test-vps-smoke.sh
# Requires: ssh, curl, jq
# ──────────────────────────────────────────────────────────────────────────────

# Auto-source .env for VPS connection details
set -a
source .env 2>/dev/null || true
set +a

# ── Environment variables ─────────────────────────────────────────────────────
: "${VPS_IP:?missing VPS_IP (set in .env or CLI)}"
: "${VPS_USER:=root}"
: "${BRIDGE_PORT:=8080}"
: "${ADMIN_TOKEN:?missing ADMIN_TOKEN (pass via CLI)}"
: "${CI_MODE:=0}"

# ── Dependency check ──────────────────────────────────────────────────────────
command -v jq >/dev/null 2>&1 || { echo "FAIL: jq is required"; exit 1; }

# ── Counters ──────────────────────────────────────────────────────────────────
TOTAL=0 PASSED=0 FAILED=0
H_TOTAL=0 H_PASS=0  # Health
A_TOTAL=0 A_PASS=0  # Auth
G_TOTAL=0 G_PASS=0  # Governance
D_TOTAL=0 D_PASS=0  # Discovery
FAILURES=""

# ── Helpers ───────────────────────────────────────────────────────────────────
ssh_vps() { ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no "${VPS_USER}@${VPS_IP}" "$@"; }

rpc_vps() {
  local method="$1" params="${2:-\{\}}"
  # shellcheck disable=SC2086
  if [ "$params" = "\{\}" ]; then
    params='{}'
  fi
  ssh_vps "curl -kfsS -H 'Authorization: Bearer ${ADMIN_TOKEN}' -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"${method}\",\"params\":${params}}' http://localhost:${BRIDGE_PORT}/api"
}

run_test() {
    local name="$1" expected="$2" actual="$3"
    TOTAL=$((TOTAL + 1))
    if echo "$actual" | grep -q "$expected"; then
        PASSED=$((PASSED + 1))
        echo "  [PASS] $name"
    else
        FAILED=$((FAILED + 1))
        FAILURES="$FAILURES\n  - $name: expected '$expected' got: $(echo "$actual" | head -c 200)"
        echo "  [FAIL] $name"
        echo "    Expected: $expected"
        echo "    Got: $(echo "$actual" | head -c 200)"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# Category A: Bridge Health (3 tests)
# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "Category A: Bridge Health"
echo "========================================="

# A1: SSH connectivity
H_TOTAL=$((H_TOTAL + 1)); TOTAL=$((TOTAL + 1))
SSH_RESULT=$(ssh_vps "echo ok" 2>&1) && SSH_OK=true || SSH_OK=false
if $SSH_OK && [ "$SSH_RESULT" = "ok" ]; then
    PASSED=$((PASSED + 1)); H_PASS=$((H_PASS + 1))
    echo "  [PASS] [1/11] SSH connectivity"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [1/11] SSH connectivity: got '$SSH_RESULT'"
    echo "  [FAIL] [1/11] SSH connectivity"
    echo "    Got: $(echo "$SSH_RESULT" | head -c 200)"
fi

# A2: Health endpoint
H_TOTAL=$((H_TOTAL + 1)); TOTAL=$((TOTAL + 1))
HEALTH=$(ssh_vps "curl -kfsS http://localhost:${BRIDGE_PORT}/health" 2>&1) && HEALTH_OK=true || HEALTH_OK=false
if $HEALTH_OK && echo "$HEALTH" | jq -e '.status == "ok"' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); H_PASS=$((H_PASS + 1))
    echo "  [PASS] [2/11] Health endpoint returns ok"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [2/11] Health endpoint: got '$(echo "$HEALTH" | head -c 200)'"
    echo "  [FAIL] [2/11] Health endpoint"
    echo "    Got: $(echo "$HEALTH" | head -c 200)"
fi

# A3: Bridge container running
H_TOTAL=$((H_TOTAL + 1)); TOTAL=$((TOTAL + 1))
CONTAINERS=$(ssh_vps "docker ps --filter name=armorclaw --format '{{.Names}}'" 2>&1) && C_OK=true || C_OK=false
if $C_OK && [ -n "$CONTAINERS" ]; then
    PASSED=$((PASSED + 1)); H_PASS=$((H_PASS + 1))
    echo "  [PASS] [3/11] Bridge container running"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [3/11] Bridge container running: got '$(echo "$CONTAINERS" | head -c 200)'"
    echo "  [FAIL] [3/11] Bridge container running"
    echo "    Got: $(echo "$CONTAINERS" | head -c 200)"
fi

# ══════════════════════════════════════════════════════════════════════════════
# Category B: Auth Enforcement (3 tests)
# CRITICAL: Bridge returns HTTP 200 even for auth errors — check JSON-RPC body.
# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "Category B: Auth Enforcement"
echo "========================================="

# B1: No auth → JSON-RPC error -32001 "unauthorized"
A_TOTAL=$((A_TOTAL + 1)); TOTAL=$((TOTAL + 1))
NOAUTH_RESP=$(ssh_vps "curl -ks -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"device.list\"}' http://localhost:${BRIDGE_PORT}/api" 2>&1)
if echo "$NOAUTH_RESP" | jq -e '.error.code == -32001' >/dev/null 2>&1 && echo "$NOAUTH_RESP" | jq -e '.error.message == "unauthorized"' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); A_PASS=$((A_PASS + 1))
    echo "  [PASS] [4/11] No auth returns JSON-RPC -32001 unauthorized"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [4/11] No auth -32001: got '$(echo "$NOAUTH_RESP" | head -c 300)'"
    echo "  [FAIL] [4/11] No auth returns JSON-RPC -32001 unauthorized"
    echo "    Got: $(echo "$NOAUTH_RESP" | head -c 300)"
fi

# B2: Valid auth → succeeds
A_TOTAL=$((A_TOTAL + 1)); TOTAL=$((TOTAL + 1))
AUTH_RESP=$(rpc_vps "device.list") && AUTH_OK=true || AUTH_OK=false
if $AUTH_OK && echo "$AUTH_RESP" | jq -e '.result' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); A_PASS=$((A_PASS + 1))
    echo "  [PASS] [5/11] Valid auth returns result"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [5/11] Valid auth: got '$(echo "$AUTH_RESP" | head -c 200)'"
    echo "  [FAIL] [5/11] Valid auth returns result"
    echo "    Got: $(echo "$AUTH_RESP" | head -c 200)"
fi

# B3: Invalid token → same JSON-RPC error
A_TOTAL=$((A_TOTAL + 1)); TOTAL=$((TOTAL + 1))
BAD_RESP=$(ssh_vps "curl -ks -H 'Authorization: Bearer invalid-token-12345' -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"device.list\"}' http://localhost:${BRIDGE_PORT}/api" 2>&1)
if echo "$BAD_RESP" | jq -e '.error.code == -32001' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); A_PASS=$((A_PASS + 1))
    echo "  [PASS] [6/11] Invalid token returns -32001"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [6/11] Invalid token -32001: got '$(echo "$BAD_RESP" | head -c 200)'"
    echo "  [FAIL] [6/11] Invalid token returns -32001"
    echo "    Got: $(echo "$BAD_RESP" | head -c 200)"
fi

# ══════════════════════════════════════════════════════════════════════════════
# Category C: Governance Methods via HTTP (4 tests)
# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "Category C: Governance Methods"
echo "========================================="

# C1: device.list
G_TOTAL=$((G_TOTAL + 1)); TOTAL=$((TOTAL + 1))
DL=$(rpc_vps "device.list")
if echo "$DL" | jq -e '.result' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); G_PASS=$((G_PASS + 1))
    echo "  [PASS] [7/11] device.list returns result"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [7/11] device.list: got '$(echo "$DL" | head -c 200)'"
    echo "  [FAIL] [7/11] device.list"
    echo "    Got: $(echo "$DL" | head -c 200)"
fi

# C2: invite.list
G_TOTAL=$((G_TOTAL + 1)); TOTAL=$((TOTAL + 1))
IL=$(rpc_vps "invite.list")
if echo "$IL" | jq -e '.result' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); G_PASS=$((G_PASS + 1))
    echo "  [PASS] [8/11] invite.list returns result"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [8/11] invite.list: got '$(echo "$IL" | head -c 200)'"
    echo "  [FAIL] [8/11] invite.list"
    echo "    Got: $(echo "$IL" | head -c 200)"
fi

# C3: invite.create
G_TOTAL=$((G_TOTAL + 1)); TOTAL=$((TOTAL + 1))
IC=$(rpc_vps "invite.create" '{"role":"admin","expiration":"1h","max_uses":1,"created_by":"vps-smoke-test"}')
if echo "$IC" | jq -e '.result.code' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); G_PASS=$((G_PASS + 1))
    echo "  [PASS] [9/11] invite.create returns invite with code"
    INVITE_ID=$(echo "$IC" | jq -r '.result.id')
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [9/11] invite.create: got '$(echo "$IC" | head -c 200)'"
    echo "  [FAIL] [9/11] invite.create"
    echo "    Got: $(echo "$IC" | head -c 200)"
fi

# C4: invite.revoke (use invite from C3 if available)
G_TOTAL=$((G_TOTAL + 1)); TOTAL=$((TOTAL + 1))
if [ -n "${INVITE_ID:-}" ]; then
    IR=$(rpc_vps "invite.revoke" "{\"invite_id\":\"$INVITE_ID\",\"revoked_by\":\"vps-smoke-test\"}")
    if echo "$IR" | jq -e '.result.success' >/dev/null 2>&1 || echo "$IR" | jq -e '.result' >/dev/null 2>&1; then
        PASSED=$((PASSED + 1)); G_PASS=$((G_PASS + 1))
        echo "  [PASS] [10/11] invite.revoke returns success"
    else
        FAILED=$((FAILED + 1))
        FAILURES="$FAILURES\n  - [10/11] invite.revoke: got '$(echo "$IR" | head -c 200)'"
        echo "  [FAIL] [10/11] invite.revoke"
        echo "    Got: $(echo "$IR" | head -c 200)"
    fi
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [10/11] invite.revoke: no invite_id from C3"
    echo "  [FAIL] [10/11] invite.revoke (no invite_id from C3)"
fi

# ── Device approve/reject (optional — only if PENDING_DEVICE_ID is set) ───────
if [ -n "${PENDING_DEVICE_ID:-}" ]; then
    echo ""
    echo "Device Lifecycle Tests (PENDING_DEVICE_ID=$PENDING_DEVICE_ID)"
    # Add device.approve and device.reject tests here
else
    echo ""
    echo "  [SKIP] Device approve/reject happy-path (PENDING_DEVICE_ID not set)"
fi

# ══════════════════════════════════════════════════════════════════════════════
# Category D: Discovery Endpoint (1 test)
# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "Category D: Discovery Endpoint"
echo "========================================="

D_TOTAL=$((D_TOTAL + 1)); TOTAL=$((TOTAL + 1))
DISC=$(ssh_vps "curl -kfsS http://localhost:${BRIDGE_PORT}/api/discovery" 2>&1) && DISC_OK=true || DISC_OK=false
if $DISC_OK && echo "$DISC" | jq -e '.version' >/dev/null 2>&1; then
    PASSED=$((PASSED + 1)); D_PASS=$((D_PASS + 1))
    echo "  [PASS] [11/11] Discovery endpoint returns version"
else
    FAILED=$((FAILED + 1))
    FAILURES="$FAILURES\n  - [11/11] Discovery endpoint: got '$(echo "$DISC" | head -c 200)'"
    echo "  [FAIL] [11/11] Discovery endpoint"
    echo "    Got: $(echo "$DISC" | head -c 200)"
fi

# ══════════════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "VPS SMOKE TEST SUMMARY"
echo "========================================="
echo "Total: $TOTAL | Passed: $PASSED | Failed: $FAILED"
echo "Groups: Health($H_PASS/$H_TOTAL) Auth($A_PASS/$A_TOTAL) Governance($G_PASS/$G_TOTAL) Discovery($D_PASS/$D_TOTAL)"
echo "========================================="
if [ -n "$FAILURES" ]; then
    echo "FAILURES:"
    echo -e "$FAILURES"
    echo "========================================="
fi

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
