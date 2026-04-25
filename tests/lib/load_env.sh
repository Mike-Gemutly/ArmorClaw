#!/usr/bin/env bash
# load_env.sh — Shared environment loader for ArmorClaw test scripts
#
# Sources .env for VPS connection details, exports key variables with
# sensible defaults, sources tests/e2e/common.sh (so callers get
# rpc_call, log_result, etc.), and provides ssh_vps() and
# check_bridge_running() helpers.
#
# Usage:
#   source "$(dirname "$0")/../lib/load_env.sh"
#   # or from tests/ root:
#   source "tests/lib/load_env.sh"

# ── Source .env (matching test-vps-smoke.sh pattern) ──────────────────────────
set -a
# Try repo-root .env first, then caller-directory relative
source "$(dirname "$0")/../../.env" 2>/dev/null || true
source .env 2>/dev/null || true
set +a

# ── Environment variables with defaults ───────────────────────────────────────
: "${VPS_IP:?VPS_IP is required — set in .env or export manually}"
: "${VPS_USER:=root}"
: "${BRIDGE_PORT:=8080}"
: "${MATRIX_PORT:=6167}"
: "${SSH_KEY_PATH:=~/.ssh/openclaw_win}"
# ADMIN_TOKEN may be empty — tests that need it should check and skip gracefully
export ADMIN_TOKEN="${ADMIN_TOKEN:-}"

export VPS_IP VPS_USER BRIDGE_PORT MATRIX_PORT SSH_KEY_PATH

# ── Source common.sh AFTER .env so its defaults don't override ────────────────
COMMON_SH="$(dirname "$0")/../e2e/common.sh"
if [[ -f "$COMMON_SH" ]]; then
  source "$COMMON_SH"
else
  echo "[WARN] tests/e2e/common.sh not found at $COMMON_SH — skipping"
fi

# ── ssh_vps helper ────────────────────────────────────────────────────────────
# Runs a command on the VPS via SSH. Mirrors the pattern from test-persistence.sh.
#
# Usage:
#   ssh_vps "systemctl status armorclaw-bridge"
#   ssh_vps "ls /run/armorclaw/"
ssh_vps() {
  ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "${VPS_USER}@${VPS_IP}" "$@"
}

# ── check_bridge_running helper ───────────────────────────────────────────────
# Returns 0 if the bridge systemd service is active on the VPS, 1 otherwise.
#
# Usage:
#   if check_bridge_running; then echo "Bridge is up"; fi
check_bridge_running() {
  local status
  status=$(ssh_vps "systemctl is-active armorclaw-bridge.service" 2>/dev/null || echo "")
  if [[ "$status" == "active" ]]; then
    return 0
  fi
  return 1
}
