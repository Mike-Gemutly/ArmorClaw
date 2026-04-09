#!/usr/bin/env bash
set -euo pipefail

# test_transport_guard.sh — Static analysis: TrustedProxyGuard wiring at all 3 entry points
# No Docker, no network, no Go source modifications required.

BRIDGE_DIR="${BRIDGE_DIR:-bridge}"
PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  [PASS] $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  [FAIL] $1"; }

echo "========================================="
echo " Transport Guard Wiring Tests"
echo "========================================="

# --- 1. guard.Check present in all 3 handleConnection() entry points ---
echo ""
echo "[CHECK 1] guard.Check presence in handleConnection files"

for f in \
  "$BRIDGE_DIR/pkg/rpc/server.go" \
  "$BRIDGE_DIR/pkg/agent/injection.go" \
  "$BRIDGE_DIR/pkg/socket/server.go"
do
  if grep -q 'guard\.Check(' "$f" 2>/dev/null; then
    pass "guard.Check() found in $(basename "$f")"
  else
    fail "guard.Check() NOT found in $(basename "$f")"
  fi
done

# --- 2. guard field exists on all 3 structs ---
echo ""
echo "[CHECK 2] guard field on structs"

# rpc.Server
if grep -qP 'guard\s+\*trust\.TrustedProxyGuard' "$BRIDGE_DIR/pkg/rpc/server.go" 2>/dev/null; then
  pass "rpc.Server has guard *trust.TrustedProxyGuard"
else
  fail "rpc.Server missing guard *trust.TrustedProxyGuard"
fi

# socket.Server
if grep -qP 'guard\s+\*trust\.TrustedProxyGuard' "$BRIDGE_DIR/pkg/socket/server.go" 2>/dev/null; then
  pass "socket.Server has guard *trust.TrustedProxyGuard"
else
  fail "socket.Server missing guard *trust.TrustedProxyGuard"
fi

# agent.PIIInjector
if grep -qP 'guard\s+\*trust\.TrustedProxyGuard' "$BRIDGE_DIR/pkg/agent/injection.go" 2>/dev/null; then
  pass "agent.PIIInjector has guard *trust.TrustedProxyGuard"
else
  fail "agent.PIIInjector missing guard *trust.TrustedProxyGuard"
fi

# --- 3. Guard in Config struct ---
echo ""
echo "[CHECK 3] Config.Guard field"

if grep -qP 'Guard\s+\*trust\.TrustedProxyGuard' "$BRIDGE_DIR/pkg/rpc/server.go" 2>/dev/null; then
  pass "Config.Guard *trust.TrustedProxyGuard found"
else
  fail "Config.Guard *trust.TrustedProxyGuard NOT found"
fi

# --- 4. Nil-guard pattern (guard != nil before Check) ---
echo ""
echo "[CHECK 4] nil-guard pattern (guard != nil before Check)"

if grep -q 'if s.guard != nil' "$BRIDGE_DIR/pkg/rpc/server.go" 2>/dev/null; then
  pass "rpc/server.go uses if s.guard != nil"
else
  fail "rpc/server.go missing nil-guard pattern"
fi

if grep -q 'if s.guard != nil' "$BRIDGE_DIR/pkg/socket/server.go" 2>/dev/null; then
  pass "socket/server.go uses if s.guard != nil"
else
  fail "socket/server.go missing nil-guard pattern"
fi

if grep -q 'if i.guard != nil' "$BRIDGE_DIR/pkg/agent/injection.go" 2>/dev/null; then
  pass "injection.go uses if i.guard != nil"
else
  fail "injection.go missing nil-guard pattern"
fi

# --- 5. Go build ---
echo ""
echo "[CHECK 5] Go build (pkg/rpc, pkg/socket, pkg/agent, pkg/trust)"

if command -v go &>/dev/null; then
  if (cd "$BRIDGE_DIR" && go build ./pkg/rpc/ ./pkg/socket/ ./pkg/agent/ ./pkg/trust/) 2>/dev/null; then
    pass "go build succeeded"
  else
    fail "go build failed"
  fi
else
  echo "  [SKIP] go not found in PATH"
fi

# --- Summary ---
echo ""
echo "========================================="
echo " Results: $PASS passed, $FAIL failed"
echo "========================================="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
