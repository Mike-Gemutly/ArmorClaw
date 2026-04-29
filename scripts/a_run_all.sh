#!/usr/bin/env bash
# a_run_all.sh — Master runner for ArmorClaw E2E phases (A0-A4)

set -uo pipefail

_SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${_SCRIPT_DIR}/lib/contract.sh"

PHASE="${1:-all}"

log_info "========================================="
log_info " ArmorClaw E2E Master Runner"
log_info " Phase: $PHASE"
log_info "========================================="

run_phase() {
  local phase_name="$1"
  local script="${_SCRIPT_DIR}/${phase_name}.sh"

  if [[ ! -f "$script" ]]; then
    log_fail "Script not found: $script"
    return 1
  fi

  log_info "--- Running ${phase_name} ---"
  bash "$script"
  local exit_code=$?

  if [[ $exit_code -ne 0 ]]; then
    log_fail "${phase_name} failed with exit code ${exit_code}"
    return $exit_code
  fi

  log_pass "${phase_name} completed"
  return 0
}

case "$PHASE" in
  A0)
    run_phase "a0_discover"
    ;;
  A1)
    run_phase "a1_deploy"
    ;;
  A2)
    run_phase "a2_provision"
    ;;
  A3)
    run_phase "a3_events"
    ;;
  A4)
    run_phase "a4_prepare" && run_phase "a4_harness"
    ;;
  all)
    run_phase "a0_discover" || {
      DEPLOY_REQ=$(jq -r '.runtime_flags.deployment_required // false' "${EVIDENCE_DIR}/contract_manifest.json" 2>/dev/null)
      if [[ "$DEPLOY_REQ" == "true" ]]; then
        log_info "Bridge not running — running A1 then re-running A0..."
        run_phase "a1_deploy" && run_phase "a0_discover"
      fi
    }
    run_phase "a2_provision"
    run_phase "a3_events"
    run_phase "a4_prepare"
    run_phase "a4_harness"
    ;;
  *)
    log_fail "Unknown phase: $PHASE. Use: A0, A1, A2, A3, A4, or all"
    exit 1
    ;;
esac

FINAL_EXIT=$?

_contract_save "final_summary.json" "$(jq -nc \
  --arg phase "$PHASE" \
  --argjson exit "$FINAL_EXIT" \
  '{
    phase: $phase,
    exit_code: $exit,
    status: (if $exit == 0 then "success" else "failed" end),
    timestamp: (now | todate)
  }')"

if [[ -f "${EVIDENCE_DIR}/a2_provisioning_outputs.json" ]]; then
  log_info ""
  log_info "=== Provisioning Outputs ==="
  jq '.' "${EVIDENCE_DIR}/a2_provisioning_outputs.json" 2>/dev/null
fi

log_info "========================================="
log_info " Master Runner Complete: Phase $PHASE"
log_info " Exit code: $FINAL_EXIT"
log_info "========================================="

exit $FINAL_EXIT
