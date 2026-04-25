# tests/lib — Shared Test Fixtures and Helper Library

This directory contains shared helpers sourced by all full-system test scripts
(T1–T11, X1–X4, F1–F4).  They extend — but do not duplicate — the functions
in `tests/e2e/common.sh` (rpc_call, log_result, test_summary, check_dependencies).

## Sourcing Example

```bash
#!/usr/bin/env bash
source "$(dirname "$0")/../lib/load_env.sh"
source "$(dirname "$0")/../lib/common_output.sh"
source "$(dirname "$0")/../lib/assert_json.sh"
source "$(dirname "$0")/../lib/restart_bridge.sh"
source "$(dirname "$0")/../lib/event_subscriber_helper.sh"
```

## Helpers

### load_env.sh

Environment loader that sources `.env` for VPS connection details (VPS_IP,
VPS_USER, ADMIN_TOKEN, BRIDGE_PORT, MATRIX_PORT, SSH_KEY_PATH), applies
sensible defaults, then sources `tests/e2e/common.sh` so callers inherit
rpc_call(), log_result(), and color variables.  Provides `ssh_vps()` for
running commands on the VPS and `check_bridge_running()` for checking the
systemd service status.

### assert_json.sh

JSON assertion helpers for validating JSON-RPC responses and structured data.
All functions echo `[PASS]`/`[FAIL]` with descriptive messages and return 0/1.
Available assertions: `assert_json_has_key`, `assert_json_equals`,
`assert_json_contains`, `assert_json_not_contains`, `assert_rpc_success`,
`assert_rpc_error`.

### restart_bridge.sh

Serialized bridge restart with readiness polling. Uses `flock` on
`/tmp/armorclaw-test-restart.lock` to prevent parallel test scripts from
racing during restarts.  Polls `systemctl is-active` plus an HTTP health
check at 2-second intervals (up to 15 attempts, matching the
test-persistence.sh pattern).  Call `restart_bridge [max_wait_seconds]`.

### event_subscriber_helper.sh

WebSocket event subscription via `websocat`.  Checks for websocat availability
at source time and sets `WEBSOCAT_AVAILABLE=false` if not found (all
functions skip gracefully).  Provides `subscribe_events` (duration-based
capture to file), `capture_events` (capture N events with timeout), and
`parse_event_type` (jq-based type extraction).  All functions handle
self-signed TLS via websocat's `-k` flag.

### common_output.sh

Test output and counter helpers independent of the TESTS_RUN/PASSED/FAILED
counters in `tests/e2e/common.sh`.  Tracks FULL_SYSTEM_PASSED/FAILED/SKIPPED
and provides `log_pass`, `log_fail`, `log_skip`, `log_info`, and
`harness_summary` (prints final counts, returns 0 if no failures).
