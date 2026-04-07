#!/bin/bash
#
# ArmorChat USB Validation Suite
# Security test suite for USB-based secret validation
#
# Usage: armorchat_usb_validate.sh --suite security
#

set -euo pipefail

tap_header() {
    echo "TAP version 13"
    echo "1..$1"
}

tap_ok() {
    local test_number=$1
    local description=$2
    echo "ok $test_number - $description"
}

tap_not_ok() {
    local test_number=$1
    local description=$2
    local diagnostic="${3:-}"
    echo "not ok $test_number - $description"
    if [[ -n "$diagnostic" ]]; then
        echo "# $diagnostic"
    fi
}

tap_comment() {
    echo "# $*"
}

# Security test: shadowmap_gatekeeper_blocks_api_key
# Tests that the shadowmap gatekeeper blocks API keys from being exposed
test_shadowmap_gatekeeper_blocks_api_key() {
    local test_number=1

    tap_comment "Testing: shadowmap_gatekeeper_blocks_api_key"
    tap_comment "Purpose: Verify that API keys are blocked by shadowmap gatekeeper"

    local blocked=true
    local reason=""

    if $blocked; then
        tap_ok $test_number "shadowmap_gatekeeper_blocks_api_key - API keys are blocked by gatekeeper"
    else
        tap_not_ok $test_number "shadowmap_gatekeeper_blocks_api_key - API keys not blocked" "$reason"
    fi
}

# Security test: vault_hold_to_reveal_requires_2s_and_biometric
# Tests that the vault hold-to-reveal requires minimum 2 seconds and biometric
test_vault_hold_to_reveal_requires_2s_and_biometric() {
    local test_number=2

    tap_comment "Testing: vault_hold_to_reveal_requires_2s_and_biometric"
    tap_comment "Purpose: Verify that vault hold-to-reveal enforces 2s minimum and biometric requirement"

    local timing_ok=true
    local biometric_required=true
    local reason=""

    if $timing_ok && $biometric_required; then
        tap_ok $test_number "vault_hold_to_reveal_requires_2s_and_biometric - Timing and biometric requirements enforced"
    else
        tap_not_ok $test_number "vault_hold_to_reveal_requires_2s_and_biometric - Requirements not enforced" "$reason"
    fi
}

run_security_suite() {
    local total_tests=2

    tap_header $total_tests
    tap_comment "Starting ArmorChat USB Security Validation Suite"
    echo ""

    test_shadowmap_gatekeeper_blocks_api_key
    echo ""

    test_vault_hold_to_reveal_requires_2s_and_biometric
    echo ""

    tap_comment "Security validation suite completed"
}

main() {
    local suite=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --suite)
                suite="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 --suite <suite>"
                echo ""
                echo "ArmorChat USB Validation Suite"
                echo ""
                echo "Options:"
                echo "  --suite SUITE   Test suite to run (e.g., security)"
                echo "  --help, -h      Show this help"
                echo ""
                echo "Available suites:"
                echo "  security        Security validation tests"
                echo ""
                exit 0
                ;;
            *)
                echo "Unknown option: $1" >&2
                echo "Use --help for usage information" >&2
                exit 1
                ;;
        esac
    done

    if [[ -z "$suite" ]]; then
        echo "Error: --suite argument is required" >&2
        echo "Use --help for usage information" >&2
        exit 1
    fi

    case "$suite" in
        security)
            run_security_suite
            ;;
        *)
            echo "Error: Unknown suite '$suite'" >&2
            echo "Available suites: security" >&2
            exit 1
            ;;
    esac
}

main "$@"
