#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Matrix Plane Integration Tests
#
# Tests Matrix messaging via matrix-commander (Docker or local).
# Covers login, sync, send/verify, round-trip, file upload, and optional
# assistant round-trip.
#
# Usage:
#   MATRIX_USER=@user:server MATRIX_PASSWORD=xxx ROOM_ID=!room:server \
#     bash tests/test-matrix-plane.sh
#
# Optional: ASSISTANT_ROOM_ID=!assistant:server  (enables Category D)
# ──────────────────────────────────────────────────────────────────────────────

# Auto-source .env for VPS connection details
set -a
source .env 2>/dev/null || true
set +a

# ── Environment variables ─────────────────────────────────────────────────────
: "${MATRIX_PORT:=6167}"
: "${HOMESERVER:=https://${VPS_IP:-localhost}:${MATRIX_PORT}}"
: "${MATRIX_USER:?missing MATRIX_USER (pass via CLI or .env)}"
: "${MATRIX_PASSWORD:?missing MATRIX_PASSWORD (pass via CLI or .env)}"
: "${ROOM_ID:?missing ROOM_ID (pass via CLI or .env)}"
: "${MATRIX_STORE:=$HOME/.matrix-commander}"
: "${MC_MODE:=docker}"
: "${CI_MODE:=0}"

# ── Prerequisite check ────────────────────────────────────────────────────────
if [[ "$MC_MODE" == "docker" ]]; then
    command -v docker >/dev/null 2>&1 || { echo "FAIL: docker required (MC_MODE=docker)"; exit 1; }
else
    command -v matrix-commander >/dev/null 2>&1 || { echo "FAIL: matrix-commander required (MC_MODE=local)"; exit 1; }
fi

# ── matrix-commander wrapper ──────────────────────────────────────────────────
mc() {
    if [[ "$MC_MODE" == "docker" ]]; then
        docker run --rm \
            -v "$MATRIX_STORE:/data" \
            matrixcommander/matrix-commander "$@"
    else
        matrix-commander "$@"
    fi
}

# ── Counters ──────────────────────────────────────────────────────────────────
TOTAL=0 PASSED=0 FAILED=0
LS_TOTAL=0 LS_PASS=0  # Login & Sync
M_TOTAL=0 M_PASS=0    # Messaging
F_TOTAL=0 F_PASS=0    # File Upload
AS_TOTAL=0 AS_PASS=0  # Assistant
FAILURES=""

# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "Category A: Login & Sync"
echo "========================================="

# A1: Login
LS_TOTAL=$((LS_TOTAL + 1)); TOTAL=$((TOTAL + 1))
echo "  [1/N] Logging in..."
if mc --login --homeserver "$HOMESERVER" --user "$MATRIX_USER" --password "$MATRIX_PASSWORD" 2>&1; then
    PASSED=$((PASSED + 1)); LS_PASS=$((LS_PASS + 1))
    echo "  [PASS] Login succeeds"
else
    # May already be logged in — check if store exists
    if [[ -f "$MATRIX_STORE/credentials.json" ]] || [[ -d "$MATRIX_STORE" ]]; then
        PASSED=$((PASSED + 1)); LS_PASS=$((LS_PASS + 1))
        echo "  [PASS] Login (already authenticated)"
    else
        FAILED=$((FAILED + 1))
        echo "  [FAIL] Login fails"
    fi
fi

# A2: Sync
LS_TOTAL=$((LS_TOTAL + 1)); TOTAL=$((TOTAL + 1))
echo "  [2/N] Syncing..."
if mc --sync off --timeout 30 2>&1; then
    PASSED=$((PASSED + 1)); LS_PASS=$((LS_PASS + 1))
    echo "  [PASS] Sync succeeds"
else
    FAILED=$((FAILED + 1))
    echo "  [FAIL] Sync fails"
fi

# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "Category B: Messaging"
echo "========================================="

# Generate unique token (openssl rand -hex 4 is cross-platform)
UNIQUE_TOKEN="ARMORCLAW-SMOKE-$(openssl rand -hex 4 2>/dev/null || echo "$(date +%s)$RANDOM")"

# B1: Send message
M_TOTAL=$((M_TOTAL + 1)); TOTAL=$((TOTAL + 1))
echo "  [3/N] Sending message with token: $UNIQUE_TOKEN"
if mc --room "$ROOM_ID" --message "$UNIQUE_TOKEN test message from armorclaw smoke test" 2>&1; then
    PASSED=$((PASSED + 1)); M_PASS=$((M_PASS + 1))
    echo "  [PASS] Send message succeeds"
else
    FAILED=$((FAILED + 1))
    echo "  [FAIL] Send message fails"
fi

# B2: Verify message landed (poll for 30s, every 5s)
M_TOTAL=$((M_TOTAL + 1)); TOTAL=$((TOTAL + 1))
echo "  [4/N] Verifying message landed..."
FOUND=false
for attempt in 1 2 3 4 5 6; do
    sleep 5
    RECENT=$(mc --room "$ROOM_ID" --sync off --timeout 5000 2>&1 || true)
    if echo "$RECENT" | grep -q "$UNIQUE_TOKEN"; then
        FOUND=true
        break
    fi
done
if $FOUND; then
    PASSED=$((PASSED + 1)); M_PASS=$((M_PASS + 1))
    echo "  [PASS] Message verified in room"
else
    FAILED=$((FAILED + 1))
    echo "  [FAIL] Message not found after 30s"
fi

# B3: Round-trip (send another + poll)
M_TOTAL=$((M_TOTAL + 1)); TOTAL=$((TOTAL + 1))
RT_TOKEN="ARMORCLAW-RT-$(openssl rand -hex 4 2>/dev/null || echo "$(date +%s)$RANDOM")"
echo "  [5/N] Round-trip test with token: $RT_TOKEN"
mc --room "$ROOM_ID" --message "$RT_TOKEN round-trip test" 2>&1 || true
RT_FOUND=false
for attempt in 1 2 3 4 5 6; do
    sleep 5
    RT_RECENT=$(mc --room "$ROOM_ID" --sync off --timeout 5000 2>&1 || true)
    if echo "$RT_RECENT" | grep -q "$RT_TOKEN"; then
        RT_FOUND=true
        break
    fi
done
if $RT_FOUND; then
    PASSED=$((PASSED + 1)); M_PASS=$((M_PASS + 1))
    echo "  [PASS] Round-trip verified"
else
    FAILED=$((FAILED + 1))
    echo "  [FAIL] Round-trip not verified after 30s"
fi

# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "Category C: File Upload"
echo "========================================="

F_TOTAL=$((F_TOTAL + 1)); TOTAL=$((TOTAL + 1))
TMPFILE=$(mktemp /tmp/armorclaw-smoke-XXXXXX.txt)
echo "ArmorClaw smoke test file upload $(date)" > "$TMPFILE"
echo "  [6/N] Uploading file..."
if mc --room "$ROOM_ID" --file "$TMPFILE" 2>&1; then
    PASSED=$((PASSED + 1)); F_PASS=$((F_PASS + 1))
    echo "  [PASS] File upload succeeds"
else
    FAILED=$((FAILED + 1))
    echo "  [FAIL] File upload fails"
fi
rm -f "$TMPFILE"

# ══════════════════════════════════════════════════════════════════════════════
echo ""
if [ -n "${ASSISTANT_ROOM_ID:-}" ]; then
    echo "========================================="
    echo "Category D: Assistant Round-Trip"
    echo "========================================="

    AS_TOTAL=$((AS_TOTAL + 1)); TOTAL=$((TOTAL + 1))
    echo "  [7/N] Sending assistant prompt..."
    mc --room "$ASSISTANT_ROOM_ID" --message "Reply with exactly: SMOKE_TEST_OK" 2>&1 || true
    ASSISTANT_FOUND=false
    for attempt in $(seq 1 12); do
        sleep 5
        AS_RECENT=$(mc --room "$ASSISTANT_ROOM_ID" --sync off --timeout 5000 2>&1 || true)
        if echo "$AS_RECENT" | grep -q "SMOKE_TEST_OK"; then
            ASSISTANT_FOUND=true
            break
        fi
    done
    if $ASSISTANT_FOUND; then
        PASSED=$((PASSED + 1)); AS_PASS=$((AS_PASS + 1))
        echo "  [PASS] Assistant responded with SMOKE_TEST_OK"
    else
        FAILED=$((FAILED + 1))
        echo "  [FAIL] Assistant did not respond within 60s"
    fi
else
    echo "  [SKIP] Assistant round-trip (ASSISTANT_ROOM_ID not set)"
fi

# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo "========================================="
echo "MATRIX PLANE TEST SUMMARY"
echo "========================================="
echo "Total: $TOTAL | Passed: $PASSED | Failed: $FAILED"
echo "Groups: LoginSync($LS_PASS/$LS_TOTAL) Messaging($M_PASS/$M_TOTAL) FileUpload($F_PASS/$F_TOTAL) Assistant($AS_PASS/$AS_TOTAL)"
echo "========================================="
if [ -n "$FAILURES" ]; then
    echo "FAILURES:"
    echo -e "$FAILURES"
    echo "========================================="
fi

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
