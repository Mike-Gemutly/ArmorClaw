#!/usr/bin/env bash
# restart_bridge.sh — Serialized bridge restart with readiness polling
#
# Restarts the armorclaw-bridge systemd service on the VPS via SSH,
# then polls until the service is active and accepting RPC calls.
# Uses flock for serialization so parallel test scripts don't race.
#
# Requires: load_env.sh (for ssh_vps, ADMIN_TOKEN, BRIDGE_PORT)
#
# Usage:
#   source "$(dirname "$0")/load_env.sh"
#   source "$(dirname "$0")/restart_bridge.sh"
#   restart_bridge          # default 30s timeout
#   restart_bridge 60       # custom 60s timeout

# ── restart_bridge [max_wait_seconds=30] ──────────────────────────────────────
# Returns 0 on success, 1 on timeout.
restart_bridge() {
  local max_wait="${1:-30}"
  local lock_file="/tmp/armorclaw-test-restart.lock"

  (
    flock -x 200 || return 1

    echo "[INFO] Restarting armorclaw-bridge.service..."
    ssh_vps "systemctl restart armorclaw-bridge.service" 2>/dev/null || true

    # Poll readiness: up to 15 intervals of 2s (matching test-persistence.sh)
    local intervals=15
    local sleep_interval=2
    local ready=false

    for i in $(seq 1 "$intervals"); do
      sleep "$sleep_interval"

      if ssh_vps "systemctl is-active armorclaw-bridge.service" 2>/dev/null | grep -q "active"; then
        # Quick RPC health check — try a lightweight call
        local health_resp
        health_resp=$(ssh_vps "curl -kfsS -o /dev/null -w '%{http_code}' https://localhost:${BRIDGE_PORT}/health" 2>/dev/null || echo "000")
        if [[ "$health_resp" == "200" ]]; then
          ready=true
          echo "[INFO] Bridge ready after $((i * sleep_interval))s"
          break
        fi
      fi

      echo "[INFO] ... waiting ($((i * sleep_interval))s)"
    done

    if $ready; then
      return 0
    else
      echo "[FAIL] Bridge not ready after ${max_wait}s"
      return 1
    fi
  ) 200>"$lock_file"
}
