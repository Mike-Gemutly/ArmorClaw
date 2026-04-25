#!/usr/bin/env bash
# assert_json.sh — JSON assertion helpers for ArmorClaw test scripts
#
# Provides functions for validating JSON-RPC responses and structured
# data.  Uses jq for extraction and bash for comparison.  All functions
# echo [PASS]/[FAIL] with descriptive messages and return 0/1.
#
# Requires: jq
# Requires: GREEN, RED, NC color variables (from tests/e2e/common.sh)

# ── assert_json_has_key <json_string> <key> ───────────────────────────────────
# Returns 0 if key exists at top level, 1 otherwise.
assert_json_has_key() {
  local json="$1" key="$2"
  if echo "$json" | jq -e --arg k "$key" 'has($k)' >/dev/null 2>&1; then
    echo -e "${GREEN}[PASS]${NC} key '$key' found in JSON"
    return 0
  else
    echo -e "${RED}[FAIL]${NC} key '$key' not found in JSON"
    return 1
  fi
}

# ── assert_json_equals <json_string> <key> <expected_value> ───────────────────
# Extracts key's value via jq and compares to expected_value.
assert_json_equals() {
  local json="$1" key="$2" expected="$3"
  local actual
  actual=$(echo "$json" | jq -r --arg k "$key" '.[$k]' 2>/dev/null)
  if [[ "$actual" == "$expected" ]]; then
    echo -e "${GREEN}[PASS]${NC} .$key == '$expected'"
    return 0
  else
    echo -e "${RED}[FAIL]${NC} .$key expected '$expected', got '$actual'"
    return 1
  fi
}

# ── assert_json_contains <json_string> <key> <substring> ──────────────────────
# Extracts key's value and checks if it contains the substring.
assert_json_contains() {
  local json="$1" key="$2" substring="$3"
  local actual
  actual=$(echo "$json" | jq -r --arg k "$key" '.[$k]' 2>/dev/null)
  if [[ "$actual" == *"$substring"* ]]; then
    echo -e "${GREEN}[PASS]${NC} .$key contains '$substring'"
    return 0
  else
    echo -e "${RED}[FAIL]${NC} .$key does not contain '$substring' (got: '$actual')"
    return 1
  fi
}

# ── assert_json_not_contains <json_string> <substring> ────────────────────────
# Checks that the raw JSON string does NOT contain the substring.
assert_json_not_contains() {
  local json="$1" substring="$2"
  if echo "$json" | grep -q "$substring"; then
    echo -e "${RED}[FAIL]${NC} JSON unexpectedly contains '$substring'"
    return 1
  else
    echo -e "${GREEN}[PASS]${NC} JSON does not contain '$substring'"
    return 0
  fi
}

# ── assert_rpc_success <rpc_response> ─────────────────────────────────────────
# Checks that the response has no top-level "error" key.
assert_rpc_success() {
  local response="$1"
  if echo "$response" | jq -e 'has("error")' >/dev/null 2>&1; then
    local err_msg
    err_msg=$(echo "$response" | jq -r '.error.message // "unknown"' 2>/dev/null)
    echo -e "${RED}[FAIL]${NC} RPC returned error: $err_msg"
    return 1
  else
    echo -e "${GREEN}[PASS]${NC} RPC succeeded (no error key)"
    return 0
  fi
}

# ── assert_rpc_error <rpc_response> [expected_code] ───────────────────────────
# Checks that the response has an "error" key. Optionally validates error code.
assert_rpc_error() {
  local response="$1" expected_code="${2:-}"
  if ! echo "$response" | jq -e 'has("error")' >/dev/null 2>&1; then
    echo -e "${RED}[FAIL]${NC} RPC did not return an error"
    return 1
  fi

  if [[ -n "$expected_code" ]]; then
    local actual_code
    actual_code=$(echo "$response" | jq -r '.error.code' 2>/dev/null)
    if [[ "$actual_code" == "$expected_code" ]]; then
      echo -e "${GREEN}[PASS]${NC} RPC error code $actual_code matches expected $expected_code"
      return 0
    else
      echo -e "${RED}[FAIL]${NC} RPC error code $actual_code != expected $expected_code"
      return 1
    fi
  fi

  echo -e "${GREEN}[PASS]${NC} RPC returned error as expected"
  return 0
}
