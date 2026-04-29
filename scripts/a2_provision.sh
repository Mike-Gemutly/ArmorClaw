#!/usr/bin/env bash
# a2_provision.sh — Phase A2: Admin bootstrap for ArmorClaw E2E
#
# Uses discovered RPC methods from contract_manifest.json to provision
# admin access, verify configuration, create Matrix test session.
# Handles blocked Matrix registration gracefully (writes SKIP).

set -uo pipefail

_SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${_SCRIPT_DIR}/lib/contract.sh"

log_info "========================================="
log_info " Phase A2: Provisioning and Admin Bootstrap"
log_info "========================================="

MANIFEST=$(_contract_load_manifest)

# ── A2.1: Determine provisioning RPC method ──────────────────────────────────
log_info "A2.1: Finding provisioning RPC from manifest..."
PROV_METHOD=$(echo "$MANIFEST" | jq -r '.live_discovered.rpc_methods[] | select(.name | test("provision")) | .name' 2>/dev/null | head -1)

if [[ -z "$PROV_METHOD" ]]; then
  PROV_METHOD="provisioning.claim"
  log_info "A2.1: No provisioning method in manifest, using default: $PROV_METHOD"
else
  log_pass "A2.1: Found provisioning method: $PROV_METHOD"
fi

HEALTH_METHOD=$(echo "$MANIFEST" | jq -r '.live_discovered.rpc_methods[] | select(.name | test("health|status")) | select(.status=="responds") | .name' 2>/dev/null | head -1)
[[ -z "$HEALTH_METHOD" ]] && HEALTH_METHOD="bridge.status"

# ── A2.2: Claim provisioning ────────────────────────────────────────────────
log_info "A2.2: Attempting provisioning claim..."
CLAIM_RESULT=$(_contract_bridge_rpc "$PROV_METHOD" '{}' 2 2>/dev/null || echo "")
CLAIM_STATUS="unknown"
if [[ -n "$CLAIM_RESULT" ]]; then
  if echo "$CLAIM_RESULT" | jq -e '.result' >/dev/null 2>&1; then
    CLAIM_STATUS="success"
    log_pass "A2.2: Provisioning claimed successfully"
  elif echo "$CLAIM_RESULT" | jq -e '.error' >/dev/null 2>&1; then
    CLAIM_STATUS="already_claimed"
    log_info "A2.2: Already provisioned or error: $(echo "$CLAIM_RESULT" | jq -r '.error.message' 2>/dev/null)"
  fi
else
  log_info "A2.2: Provisioning claim returned no response"
fi

# ── A2.3: Retrieve effective config ──────────────────────────────────────────
log_info "A2.3: Retrieving configuration..."
CONFIG_RESULT=$(_contract_bridge_rpc "bridge.status" '{}' 1 2>/dev/null || echo "")
if [[ -n "$CONFIG_RESULT" ]] && echo "$CONFIG_RESULT" | jq -e '.result' >/dev/null 2>&1; then
  _contract_save "a2_bridge_status.json" "$(echo "$CONFIG_RESULT" | jq '.result')"
  log_pass "A2.3: Bridge status retrieved"
else
  log_info "A2.3: Bridge status not available"
fi

# ── A2.4: Verify bridge health/status ────────────────────────────────────────
log_info "A2.4: Verifying bridge health..."
HEALTH_RESULT=$(_contract_bridge_rpc "$HEALTH_METHOD" '{}' 2 2>/dev/null || echo "")
if [[ -n "$HEALTH_RESULT" ]] && echo "$HEALTH_RESULT" | jq -e '.result' >/dev/null 2>&1; then
  log_pass "A2.4: Bridge health check responded"
else
  log_info "A2.4: Health check via RPC not available, trying HTTP..."
  HTTP_HEALTH=$(ssh_vps "curl -sf 'http://localhost:${BRIDGE_PORT}/health'" 2>/dev/null || echo "")
  if [[ -n "$HTTP_HEALTH" ]]; then
    log_pass "A2.4: HTTP /health responded"
  else
    log_fail "A2.4: Bridge not healthy"
  fi
fi

# ── A2.5: Retrieve /qr/config ────────────────────────────────────────────────
log_info "A2.5: Checking /qr/config..."
QR_CONFIG=$(ssh_vps "curl -sf -m 5 'http://localhost:${BRIDGE_PORT}/qr/config'" 2>/dev/null || echo "")
if [[ -n "$QR_CONFIG" ]] && echo "$QR_CONFIG" | jq -e . >/dev/null 2>&1; then
  _contract_save "a2_qr_config.json" "$QR_CONFIG"
  log_pass "A2.5: /qr/config available"
else
  log_info "A2.5: /qr/config not available (non-fatal)"
fi

# ── A2.6: Verify /.well-known/matrix/client ──────────────────────────────────
log_info "A2.6: Verifying Matrix client discovery..."
WELL_KNOWN=$(ssh_vps "curl -sf -m 5 'http://localhost:${BRIDGE_PORT}/.well-known/matrix/client'" 2>/dev/null || echo "")
HOMESERVER_URL=""

if [[ -n "$WELL_KNOWN" ]] && echo "$WELL_KNOWN" | jq -e '.["m.homeserver"].base_url' >/dev/null 2>&1; then
  HOMESERVER_URL=$(echo "$WELL_KNOWN" | jq -r '.["m.homeserver"].base_url')
  log_pass "A2.6: Matrix client discovery: $HOMESERVER_URL"
else
  HOMESERVER_URL="http://${VPS_IP}:${MATRIX_PORT}"
  log_info "A2.6: Using default homeserver URL: $HOMESERVER_URL"
fi

# ── A2.7: Create test Matrix user ────────────────────────────────────────────
log_info "A2.7: Creating test Matrix user..."
MATRIX_SESSION_STATUS="PENDING"
TEST_USER="planatest_$(date +%s)"
TEST_PASS="PlanATest_$(openssl rand -hex 8)"
MATRIX_SESSION_JSON="{}"

REG_RESULT=$(ssh_vps "curl -sf -m 10 -X POST 'http://localhost:${MATRIX_PORT}/_matrix/client/v3/register' \
  -H 'Content-Type: application/json' \
  -d '{\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PASS}\",\"auth\":{\"type\":\"m.login.dummy\"}}'" 2>/dev/null || echo "")

if [[ -n "$REG_RESULT" ]] && echo "$REG_RESULT" | jq -e '.access_token' >/dev/null 2>&1; then
  MATRIX_SESSION_STATUS="SUCCESS"
  ACCESS_TOKEN=$(echo "$REG_RESULT" | jq -r '.access_token')
  DEVICE_ID=$(echo "$REG_RESULT" | jq -r '.device_id // "unknown"')
  USER_ID=$(echo "$REG_RESULT" | jq -r '.user_id')
  log_pass "A2.7: Matrix user created: $USER_ID"

  MATRIX_SESSION_JSON=$(jq -nc \
    --arg user "$USER_ID" --arg token "$ACCESS_TOKEN" --arg device "$DEVICE_ID" \
    --arg homeserver "$HOMESERVER_URL" \
    '{user_id: $user, access_token: $token, device_id: $device, homeserver_url: $homeserver}')
  _contract_save "a2_matrix_session.json" "$MATRIX_SESSION_JSON"
elif [[ -n "$REG_RESULT" ]] && echo "$REG_RESULT" | jq -e '.error' >/dev/null 2>&1; then
  MATRIX_SESSION_STATUS="BLOCKED"
  BLOCKED_REASON=$(echo "$REG_RESULT" | jq -r '.error // "registration blocked"' 2>/dev/null)
  log_info "A2.7: Matrix registration blocked: $BLOCKED_REASON"

  # Try with registration token from A1 (Conduit REGISTRATION_TOKEN)
  if [[ "$BLOCKED_REASON" == *"registration"* ]] || [[ "$BLOCKED_REASON" == *"forbidden"* ]]; then
    log_info "A2.7: Trying with shared-secret registration..."
    REG_RESULT2=$(ssh_vps "curl -sf -m 10 -X POST 'http://localhost:${MATRIX_PORT}/_matrix/client/v3/register' \
      -H 'Content-Type: application/json' \
      -d '{\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PASS}\",\"auth\":{\"type\":\"m.login.registration_token\",\"token\":\"planatest\"}}'" 2>/dev/null || echo "")

    if [[ -n "$REG_RESULT2" ]] && echo "$REG_RESULT2" | jq -e '.access_token' >/dev/null 2>&1; then
      MATRIX_SESSION_STATUS="SUCCESS"
      ACCESS_TOKEN=$(echo "$REG_RESULT2" | jq -r '.access_token')
      USER_ID=$(echo "$REG_RESULT2" | jq -r '.user_id')
      log_pass "A2.7: Matrix user created with token: $USER_ID"
      MATRIX_SESSION_JSON=$(jq -nc \
        --arg user "$USER_ID" --arg token "$ACCESS_TOKEN" \
        --arg homeserver "$HOMESERVER_URL" \
        '{user_id: $user, access_token: $token, homeserver_url: $homeserver}')
      _contract_save "a2_matrix_session.json" "$MATRIX_SESSION_JSON"
    fi
  fi
else
  MATRIX_SESSION_STATUS="FAILED"
  log_info "A2.7: Matrix registration returned empty or unexpected response"
fi

if [[ "$MATRIX_SESSION_STATUS" != "SUCCESS" ]]; then
  log_info "A2.7: Matrix session SKIPPED (reason: $MATRIX_SESSION_STATUS)"
fi

# ── A2.8: Create test room ──────────────────────────────────────────────────
ROOM_ID=""
if [[ "$MATRIX_SESSION_STATUS" == "SUCCESS" ]]; then
  log_info "A2.8: Creating test Matrix room..."
  ROOM_RESULT=$(ssh_vps "curl -sf -m 10 -X POST 'http://localhost:${MATRIX_PORT}/_matrix/client/v3/createRoom' \
    -H 'Content-Type: application/json' \
    -H 'Authorization: Bearer ${ACCESS_TOKEN}' \
    -d '{\"name\":\"Plan A Test Room\",\"visibility\":\"private\"}'" 2>/dev/null || echo "")

  if [[ -n "$ROOM_RESULT" ]] && echo "$ROOM_RESULT" | jq -e '.room_id' >/dev/null 2>&1; then
    ROOM_ID=$(echo "$ROOM_RESULT" | jq -r '.room_id')
    log_pass "A2.8: Test room created: $ROOM_ID"
  else
    log_info "A2.8: Room creation failed or returned unexpected response"
  fi
else
  log_skip "A2.8: Room creation skipped (no Matrix session)"
fi

# ── A2.9: Write provisioning outputs ─────────────────────────────────────────
log_info "A2.9: Writing provisioning outputs..."

PROV_OUTPUTS=$(jq -nc \
  --arg homeserver_url "$HOMESERVER_URL" \
  --arg bridge_port "$BRIDGE_PORT" \
  --arg vps_ip "$VPS_IP" \
  --arg matrix_session "$MATRIX_SESSION_STATUS" \
  --arg room_id "$ROOM_ID" \
  --arg user_id "${USER_ID:-}" \
  --arg claim_status "$CLAIM_STATUS" \
  '{
    homeserver_url: $homeserver_url,
    bridge_url: ("http://" + $vps_ip + ":" + $bridge_port),
    vps_ip: $vps_ip,
    bridge_port: ($bridge_port | tonumber),
    matrix_session: (if $matrix_session == "SUCCESS" then "active" else "SKIPPED" end),
    matrix_session_reason: (if $matrix_session == "SUCCESS" then "registration succeeded" else $matrix_session end),
    test_room_id: $room_id,
    test_user_id: $user_id,
    provisioning_claim_status: $claim_status,
    timestamp: (now | todate)
  }')

_contract_save "a2_provisioning_outputs.json" "$PROV_OUTPUTS"

_contract_update_manifest_merge '.provisioning = $PROV' --argjson PROV "$PROV_OUTPUTS"

_contract_save "a2_summary.json" "$(jq -nc \
  --arg phase "A2" --arg matrix "$MATRIX_SESSION_STATUS" --arg claim "$CLAIM_STATUS" \
  '{
    phase: $phase,
    status: "complete",
    provisioning_claim: $claim,
    matrix_session: $matrix,
    timestamp: (now | todate)
  }')"

log_info "========================================="
log_info " Phase A2: Provisioning Complete"
log_info "  Matrix session: $MATRIX_SESSION_STATUS"
log_info "  Claim status: $CLAIM_STATUS"
log_info "========================================="
harness_summary
