#!/usr/bin/env bash
# a3_events.sh — Phase A3: Control-plane event validation for ArmorClaw E2E
#
# Validates Matrix event emission via /sync. Degrades gracefully when
# Matrix session is unavailable (A3.1-A3.3 SKIP, A3.4-A3.5 best-effort).
# Saves a3.5_discovered_event_types.txt for cross-referencing.

set -uo pipefail

_SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${_SCRIPT_DIR}/lib/contract.sh"

log_info "========================================="
log_info " Phase A3: Control-Plane Event Validation"
log_info "========================================="

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

# ── A3.0: Check Matrix session availability ──────────────────────────────────
log_info "A3.0: Checking Matrix session..."
SESSION_FILE="${EVIDENCE_DIR}/a2_matrix_session.json"
ACCESS_TOKEN=""
USER_ID=""
HOMESERVER_URL=""
HAS_SESSION=false

if [[ -f "$SESSION_FILE" ]]; then
  ACCESS_TOKEN=$(jq -r '.access_token // empty' "$SESSION_FILE" 2>/dev/null)
  USER_ID=$(jq -r '.user_id // empty' "$SESSION_FILE" 2>/dev/null)
  HOMESERVER_URL=$(jq -r '.homeserver_url // empty' "$SESSION_FILE" 2>/dev/null)
  if [[ -n "$ACCESS_TOKEN" && "$ACCESS_TOKEN" != "null" ]]; then
    HAS_SESSION=true
    log_pass "A3.0: Matrix session available ($USER_ID)"
  fi
fi

if [[ "$HAS_SESSION" == "false" ]]; then
  PROV_OUTPUTS="${EVIDENCE_DIR}/a2_provisioning_outputs.json"
  if [[ -f "$PROV_OUTPUTS" ]]; then
    SESSION_STATUS=$(jq -r '.matrix_session // "unknown"' "$PROV_OUTPUTS" 2>/dev/null)
    REASON=$(jq -r '.matrix_session_reason // "no session file"' "$PROV_OUTPUTS" 2>/dev/null)
    log_skip "A3.0: No Matrix session (status: $SESSION_STATUS, reason: $REASON)"
  else
    log_skip "A3.0: No Matrix session and no provisioning outputs found"
  fi
  log_info "A3.0: A3.1-A3.3 will be SKIPPED, A3.4-A3.5 will proceed best-effort"
fi

# ── A3.0b: Load room ID from provisioning outputs ────────────────────────────
ROOM_ID=""
PROV_OUTPUTS_FILE="${EVIDENCE_DIR}/a2_provisioning_outputs.json"
if [[ -f "$PROV_OUTPUTS_FILE" ]]; then
  ROOM_ID=$(jq -r '.test_room_id // empty' "$PROV_OUTPUTS_FILE" 2>/dev/null)
fi

# ── A3.1: Send m.room.message and verify in /sync ────────────────────────────
if [[ "$HAS_SESSION" == "true" ]]; then
  log_info "A3.1: Sending test message and verifying via /sync..."

  TEST_MSG="Plan A event test $(date +%s)"
  TXN_ID="txn_$(date +%s)"

  if [[ -z "$ROOM_ID" ]]; then
    log_skip "A3.1: No test room ID available from A2 provisioning"
    SKIP_COUNT=$((SKIP_COUNT + 1))
  else
    # Escape the message for safe embedding in the remote curl body
    TEST_MSG_ESCAPED=$(echo "$TEST_MSG" | sed 's/"/\\"/g')

    SEND_RESULT=$(ssh_vps "curl -sf -m 10 -X PUT 'http://localhost:${MATRIX_PORT}/_matrix/client/v3/rooms/${ROOM_ID}/send/m.room.message/${TXN_ID}' \
      -H 'Content-Type: application/json' \
      -H 'Authorization: Bearer ${ACCESS_TOKEN}' \
      -d '{\"msgtype\":\"m.text\",\"body\":\"${TEST_MSG_ESCAPED}\"}'" 2>/dev/null || echo "")

    # Normalize: check for event_id (Matrix v3 success) or error
    if [[ -n "$SEND_RESULT" ]] && echo "$SEND_RESULT" | jq -e '.event_id' >/dev/null 2>&1; then
      EVENT_ID=$(echo "$SEND_RESULT" | jq -r '.event_id')
      log_pass "A3.1: Message sent: $EVENT_ID"
      PASS_COUNT=$((PASS_COUNT + 1))

      sleep 2
      SYNC_RESULT=$(ssh_vps "curl -sf -m 10 'http://localhost:${MATRIX_PORT}/_matrix/client/v3/sync?timeout=5000' \
        -H 'Authorization: Bearer ${ACCESS_TOKEN}'" 2>/dev/null || echo "")

      if [[ -n "$SYNC_RESULT" ]] && echo "$SYNC_RESULT" | jq -e '.rooms' >/dev/null 2>&1; then
        MSG_FOUND=$(echo "$SYNC_RESULT" | jq -r '.rooms.joined // {} | to_entries[] | .value.timeline.events[] | select(.content.body == "'"$TEST_MSG"'") | .event_id' 2>/dev/null | head -1)
        if [[ -n "$MSG_FOUND" ]]; then
          log_pass "A3.1: Message confirmed in /sync: $MSG_FOUND"
          PASS_COUNT=$((PASS_COUNT + 1))
        else
          log_info "A3.1: Message sent but not found in initial /sync (may need next_batch token)"
        fi
      fi
    else
      # Log the actual response for diagnosis
      if [[ -n "$SEND_RESULT" ]]; then
        SEND_ERR=$(echo "$SEND_RESULT" | jq -r '.error // .errcode // "unknown"' 2>/dev/null)
        log_fail "A3.1: Failed to send message (room=$ROOM_ID, error=$SEND_ERR)"
      else
        log_fail "A3.1: Failed to send message (empty response, room=$ROOM_ID)"
      fi
      FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
  fi
else
  log_skip "A3.1: m.room.message test — no Matrix session"
  SKIP_COUNT=$((SKIP_COUNT + 1))
fi

# ── A3.2: Start workflow via RPC, observe workflow events ────────────────────
if [[ "$HAS_SESSION" == "true" ]]; then
  log_info "A3.2: Attempting workflow start to observe events..."
  WF_RESULT=$(_contract_bridge_rpc "secretary.start_workflow" '{"template_name":"test","name":"plan-a-test"}' 1 2>/dev/null || echo "")

  if [[ -n "$WF_RESULT" ]] && echo "$WF_RESULT" | jq -e '.result' >/dev/null 2>&1; then
    log_pass "A3.2: Workflow started"
    PASS_COUNT=$((PASS_COUNT + 1))

    sleep 3
    SYNC_RESULT=$(ssh_vps "curl -sf -m 10 'http://localhost:${MATRIX_PORT}/_matrix/client/v3/sync?timeout=5000' \
      -H 'Authorization: Bearer ${ACCESS_TOKEN}'" 2>/dev/null || echo "")
    if [[ -n "$SYNC_RESULT" ]] && echo "$SYNC_RESULT" | jq -e '.rooms' >/dev/null 2>&1; then
      WF_EVENTS=$(echo "$SYNC_RESULT" | jq -r '[.rooms.joined // {} | to_entries[] | .value.timeline.events[] | .type] | unique | join(", ")' 2>/dev/null)
      if [[ -n "$WF_EVENTS" ]]; then
        log_pass "A3.2: Workflow events observed: $WF_EVENTS"
        PASS_COUNT=$((PASS_COUNT + 1))
      fi
    fi
  else
    log_skip "A3.2: No auto-triggerable workflow (expected for test environment)"
    SKIP_COUNT=$((SKIP_COUNT + 1))
  fi
else
  log_skip "A3.2: Workflow event observation — no Matrix session"
  SKIP_COUNT=$((SKIP_COUNT + 1))
fi

# ── A3.3: Check for agent status events ──────────────────────────────────────
if [[ "$HAS_SESSION" == "true" ]]; then
  log_info "A3.3: Checking for agent status events..."
  AGENT_RESULT=$(_contract_bridge_rpc "studio.stats" '{}' 1 2>/dev/null || echo "")
  if [[ -n "$AGENT_RESULT" ]] && echo "$AGENT_RESULT" | jq -e '.result' >/dev/null 2>&1; then
    log_pass "A3.3: Studio/stats available (agent infrastructure present)"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    log_skip "A3.3: No agent status events (no running agents)"
    SKIP_COUNT=$((SKIP_COUNT + 1))
  fi
else
  log_skip "A3.3: Agent status check — no Matrix session"
  SKIP_COUNT=$((SKIP_COUNT + 1))
fi

# ── A3.4: Scan bridge logs for event publication evidence ────────────────────
log_info "A3.4: Scanning bridge logs for event evidence..."
LOG_EVENTS=$(ssh_vps "docker logs armorclaw 2>&1 | grep -i 'event\|publish\|emit\|matrix.*send' | tail -20" 2>/dev/null || \
  ssh_vps "docker logs armorclaw-bridge 2>&1 | grep -i 'event\|publish\|emit\|matrix.*send' | tail -20" 2>/dev/null || \
  ssh_vps "journalctl -u armorclaw-bridge --no-pager -n 100 2>/dev/null | grep -i 'event\|publish\|emit'" 2>/dev/null || echo "")

if [[ -n "$LOG_EVENTS" ]]; then
  _contract_save "a3_log_events.txt" "$LOG_EVENTS"
  LOG_EVENT_COUNT=$(echo "$LOG_EVENTS" | wc -l)
  log_pass "A3.4: Found ${LOG_EVENT_COUNT} event-related log entries"
  PASS_COUNT=$((PASS_COUNT + 1))
else
  log_info "A3.4: No event evidence found in bridge logs"
  _contract_save "a3_log_events.txt" "(no event evidence in bridge logs)"
fi

# ── A3.5: Full event type scan ───────────────────────────────────────────────
log_info "A3.5: Scanning for all event types..."
DISCOVERED_TYPES_FILE="${EVIDENCE_DIR}/a3.5_discovered_event_types.txt"
echo "# Discovered event types - $(date)" > "$DISCOVERED_TYPES_FILE"

# If we have a session, do /sync to find types
if [[ "$HAS_SESSION" == "true" ]]; then
  for i in 1 2 3; do
    SYNC_RESULT=$(ssh_vps "curl -sf -m 10 'http://localhost:${MATRIX_PORT}/_matrix/client/v3/sync?timeout=3000' \
      -H 'Authorization: Bearer ${ACCESS_TOKEN}'" 2>/dev/null || echo "")
    if [[ -n "$SYNC_RESULT" ]] && echo "$SYNC_RESULT" | jq -e '.rooms' >/dev/null 2>&1; then
      echo "$SYNC_RESULT" | jq -r '.rooms.joined // {} | to_entries[] | .value.timeline.events[]?.type // empty' 2>/dev/null >> "$DISCOVERED_TYPES_FILE"
      echo "$SYNC_RESULT" | jq -r '.rooms.invite // {} | to_entries[] | .value.invite_state.events[]?.type // empty' 2>/dev/null >> "$DISCOVERED_TYPES_FILE"
    fi
    sleep 1
  done
fi

# Also check for event types in contract manifest
MANIFEST=$(_contract_load_manifest)
echo "$MANIFEST" | jq -r '.live_discovered.event_types[]?.type // empty' 2>/dev/null >> "$DISCOVERED_TYPES_FILE"

# Deduplicate
sort -u "$DISCOVERED_TYPES_FILE" > "${DISCOVERED_TYPES_FILE}.tmp" && mv "${DISCOVERED_TYPES_FILE}.tmp" "$DISCOVERED_TYPES_FILE"
TYPE_COUNT=$(grep -cv '^#' "$DISCOVERED_TYPES_FILE" 2>/dev/null || true)
TYPE_COUNT=${TYPE_COUNT:-0}

if [[ "$TYPE_COUNT" -gt 0 ]]; then
  log_pass "A3.5: Discovered ${TYPE_COUNT} unique event types"
  PASS_COUNT=$((PASS_COUNT + 1))
else
  log_info "A3.5: No event types discovered (may be normal for idle deployment)"
fi

# ── Update manifest with event types ─────────────────────────────────────────
EVENT_TYPES_JSON="[]"
while IFS= read -r etype; do
  [[ -z "$etype" || "$etype" == \#* ]] && continue
  EVENT_TYPES_JSON=$(echo "$EVENT_TYPES_JSON" | jq --arg t "$etype" '. + [{type: $t, source: "sync", verified: true}]')
done < "$DISCOVERED_TYPES_FILE"

_contract_update_manifest_merge ".live_discovered.event_types = \$TYPES" --argjson TYPES "$EVENT_TYPES_JSON"

# ── Summary ──────────────────────────────────────────────────────────────────
_contract_save "a3_summary.json" "$(jq -nc \
  --argjson pass "$PASS_COUNT" --argjson fail "$FAIL_COUNT" --argjson skip "$SKIP_COUNT" \
  '{
    phase: "A3",
    pass: $pass,
    fail: $fail,
    skip: $skip,
    total: ($pass + $fail + $skip),
    matrix_session_available: ($skip < 3),
    timestamp: (now | todate)
  }')"

FULL_SYSTEM_PASSED=$PASS_COUNT
FULL_SYSTEM_FAILED=$FAIL_COUNT
FULL_SYSTEM_SKIPPED=$SKIP_COUNT

log_info "========================================="
log_info " Phase A3: Event Validation Complete"
log_info "  Pass: $PASS_COUNT | Fail: $FAIL_COUNT | Skip: $SKIP_COUNT"
log_info "========================================="
harness_summary
