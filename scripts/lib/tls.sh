#!/usr/bin/env bash
# tls.sh — TLS operations wrapper for ArmorClaw
#
# Thin wrapper around deploy/scripts/generate-certs.sh. Provides mode detection,
# fingerprint computation, expiry tracking, and cert generation with public IP SANs.
#
# Usage:
#   source "$(dirname "$0")/lib/tls.sh"
#   tls_get_mode
#   tls_generate_certs --public-ip 203.0.113.10

set -uo pipefail

_TLS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_REPO_ROOT="$(cd "${_TLS_DIR}/../.." && pwd)"

source "${_REPO_ROOT}/scripts/lib/contract.sh"

# ── tls_get_mode ────────────────────────────────────────────────────────────
# Queries bridge.status RPC and reads .tls.mode.
# Falls back to ARMORCLAW_TLS_MODE env var when bridge is unreachable (bootstrap-only).
tls_get_mode() {
  local status
  status=$(_contract_bridge_rpc "bridge.status" "{}" 1 2>/dev/null) || true

  if echo "$status" | jq -e '.result.tls.mode' >/dev/null 2>&1; then
    echo "$status" | jq -r '.result.tls.mode'
    return 0
  fi

  if [[ -n "${ARMORCLAW_TLS_MODE:-}" ]]; then
    echo "$ARMORCLAW_TLS_MODE"
    return 0
  fi

  echo "none"
}

# ── tls_generate_certs [--public-ip IP] [--hostname NAME] [--output DIR] ─────
# Calls deploy/scripts/generate-certs.sh with forwarded arguments.
tls_generate_certs() {
  local cert_script="${_REPO_ROOT}/deploy/scripts/generate-certs.sh"
  if [[ ! -x "$cert_script" ]]; then
    echo "ERROR: cert generator not found at $cert_script" >&2
    return 1
  fi
  bash "$cert_script" "$@"
}

# ── tls_get_fingerprint [cert_path] ────────────────────────────────────────
# Reads cert from disk, computes SHA-256 fingerprint via openssl.
# Default cert path: /etc/armorclaw/certs/server.crt
tls_get_fingerprint() {
  local cert_path="${1:-/etc/armorclaw/certs/server.crt}"
  if [[ ! -f "$cert_path" ]]; then
    echo "" >&2
    return 1
  fi
  openssl x509 -in "$cert_path" -fingerprint -sha256 -noout 2>/dev/null \
    | cut -d= -f2 | tr -d ':' | tr 'A-F' 'a-f'
}

# ── tls_get_expiry [cert_path] ─────────────────────────────────────────────
# Returns cert expiry as Unix timestamp.
tls_get_expiry() {
  local cert_path="${1:-/etc/armorclaw/certs/server.crt}"
  if [[ ! -f "$cert_path" ]]; then
    echo "0"
    return 1
  fi
  local date_str
  date_str=$(openssl x509 -in "$cert_path" -enddate -noout 2>/dev/null | cut -d= -f2)
  if [[ -z "$date_str" ]]; then
    echo "0"
    return 1
  fi
  date -d "$date_str" +%s 2>/dev/null || echo "0"
}

# ── tls_metadata [cert_path] ───────────────────────────────────────────────
# Outputs JSON with mode, fingerprint, expiry, trust_type.
tls_metadata() {
  local cert_path="${1:-/etc/armorclaw/certs/server.crt}"
  local mode
  mode=$(tls_get_mode)
  local fp
  fp=$(tls_get_fingerprint "$cert_path" 2>/dev/null) || fp=""
  local expiry
  expiry=$(tls_get_expiry "$cert_path" 2>/dev/null) || expiry="0"

  local trust_type=""
  if [[ "$mode" == "private" ]]; then
    trust_type="self_signed"
  elif [[ "$mode" == "public" ]]; then
    trust_type="public_ca"
  fi

  jq -n \
    --arg mode "$mode" \
    --arg fingerprint "$fp" \
    --arg trust_type "$trust_type" \
    --argjson expires_at "$expiry" \
    '{mode: $mode, fingerprint_sha256: $fingerprint, trust_type: $trust_type, expires_at: $expires_at}'
}
