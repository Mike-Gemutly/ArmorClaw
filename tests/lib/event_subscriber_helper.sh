#!/usr/bin/env bash
# event_subscriber_helper.sh — WebSocket event subscription for ArmorClaw tests
#
# Provides helpers for subscribing to the bridge WebSocket event stream,
# capturing events, and parsing event types.  Handles self-signed TLS.
#
# Requires: jq, websocat (optional — functions skip gracefully if missing)
# Requires: load_env.sh (for VPS_IP, BRIDGE_PORT)

# ── websocat availability check ───────────────────────────────────────────────
WEBSOCAT_AVAILABLE=true
if ! command -v websocat >/dev/null 2>&1; then
  WEBSOCAT_AVAILABLE=false
  echo "[WARN] websocat not found — WebSocket event tests will be skipped"
fi

# ── subscribe_events <duration_seconds> <output_file> ─────────────────────────
# Connects to the bridge WebSocket endpoint and captures events for the
# specified duration into output_file.  Uses -k for self-signed TLS.
#
# Returns 0 on success, 1 if websocat unavailable or connection fails.
subscribe_events() {
  local duration="$1" output_file="$2"

  if [[ "$WEBSOCAT_AVAILABLE" != "true" ]]; then
    echo "[SKIP] subscribe_events: websocat not available"
    return 1
  fi

  local ws_url="wss://${VPS_IP}:${BRIDGE_PORT}/ws"
  echo "[INFO] Subscribing to $ws_url for ${duration}s..."

  timeout "$duration" websocat -k "$ws_url" > "$output_file" 2>/dev/null
  local rc=$?

  if [[ $rc -eq 0 || $rc -eq 124 ]]; then
    # 124 = timeout expired (expected for duration-based capture)
    local count
    count=$(wc -l < "$output_file" 2>/dev/null || echo 0)
    echo "[INFO] Captured $count event lines"
    return 0
  else
    echo "[WARN] websocat exited with code $rc"
    return 1
  fi
}

# ── capture_events <count> <timeout_seconds> ──────────────────────────────────
# Captures exactly N events with a timeout.  Writes JSON array to stdout.
#
# Returns 0 if count reached, 1 if websocat unavailable or timeout.
capture_events() {
  local count="$1" timeout_seconds="$2"

  if [[ "$WEBSOCAT_AVAILABLE" != "true" ]]; then
    echo "[SKIP] capture_events: websocat not available"
    return 1
  fi

  local ws_url="wss://${VPS_IP}:${BRIDGE_PORT}/ws"
  local captured=0
  local tmp_file
  tmp_file=$(mktemp)

  # Use websocat with timeout, collect lines
  timeout "$timeout_seconds" websocat -k "$ws_url" > "$tmp_file" 2>/dev/null || true

  # Extract first N valid JSON lines
  local result="["
  local first=true
  while IFS= read -r line && [[ $captured -lt $count ]]; do
    if echo "$line" | jq -e '.' >/dev/null 2>&1; then
      if $first; then
        first=false
      else
        result+=","
      fi
      result+="$line"
      captured=$((captured + 1))
    fi
  done < "$tmp_file"
  result+="]"

  rm -f "$tmp_file"

  echo "$result"
  if [[ $captured -ge $count ]]; then
    return 0
  else
    echo "[WARN] capture_events: only got $captured/$count events"
    return 1
  fi
}

# ── parse_event_type <event_json> ─────────────────────────────────────────────
# Extracts the "type" field from an event JSON object.
parse_event_type() {
  local event_json="$1"
  echo "$event_json" | jq -r '.type // "unknown"' 2>/dev/null
}
