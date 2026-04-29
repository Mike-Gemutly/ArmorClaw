#!/usr/bin/env bash
# a4_prepare.sh — Phase A4-prep: Copy test harness to VPS

set -uo pipefail

_SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${_SCRIPT_DIR}/lib/contract.sh"

log_info "========================================="
log_info " Phase A4-Prep: Copying Harness to VPS"
log_info "========================================="

REPO_ROOT="$(cd "${_SCRIPT_DIR}/.." && pwd)"
REMOTE_DIR="/opt/armorclaw/tests"

if [[ ! -d "${REPO_ROOT}/tests" ]]; then
  log_fail "tests/ directory not found at ${REPO_ROOT}/tests"
  exit 1
fi

log_info "Creating remote directory..."
ssh_vps "mkdir -p ${REMOTE_DIR}/lib"

log_info "Copying test files to VPS..."
scp -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -r \
  "${REPO_ROOT}/tests/test-"*.sh "${VPS_USER}@${VPS_IP}:${REMOTE_DIR}/" 2>/dev/null && \
  log_pass "Test scripts copied" || log_fail "Failed to copy test scripts"

scp -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -r \
  "${REPO_ROOT}/tests/lib/"*.sh "${VPS_USER}@${VPS_IP}:${REMOTE_DIR}/lib/" 2>/dev/null && \
  log_pass "Test libraries copied" || log_fail "Failed to copy test libraries"

if [[ -d "${REPO_ROOT}/tests/e2e" ]]; then
  ssh_vps "mkdir -p ${REMOTE_DIR}/e2e"
  scp -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -r \
    "${REPO_ROOT}/tests/e2e/"*.sh "${VPS_USER}@${VPS_IP}:${REMOTE_DIR}/e2e/" 2>/dev/null && \
    log_pass "E2E scripts copied" || log_info "E2E scripts copy skipped"
fi

log_info "Setting execute permissions on VPS..."
ssh_vps "chmod +x ${REMOTE_DIR}/*.sh ${REMOTE_DIR}/lib/*.sh ${REMOTE_DIR}/e2e/*.sh 2>/dev/null"

_contract_save "a4_prepare_status.json" "$(jq -nc '{
  phase: "A4-prepare",
  status: "complete",
  remote_dir: "/opt/armorclaw/tests",
  timestamp: (now | todate)
}')"

log_pass "A4-Prep: Harness copied to VPS"
harness_summary
