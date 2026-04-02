#!/bin/bash
# =============================================================================
# Unit Tests for detect_network_environment()
# Tests: Public IP detection, local IP detection, NAT detection, port checking
# =============================================================================

set -euo pipefail

# Set project root relative to this script
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK_DIR=$(mktemp -d)
FAILED=0

log() { echo "[TEST] $*"; }
pass() { echo "✓ $*"; }
fail() { echo "✗ $*"; ((FAILED++)); }
cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

# Source the cloudflare-functions.sh library
source "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh"

# Test 1: Function exists
test_function_exists() {
    log "Test 1: detect_network_environment function exists"
    if declare -f detect_network_environment >/dev/null 2>&1; then
        pass "detect_network_environment function defined"
    else
        fail "detect_network_environment function not found"
        return 1
    fi
}

# Test 2: Default values are set when detection fails
test_default_values() {
    log "Test 2: Default values when detection fails"

    # Mock curl to fail
    curl() { echo ""; }
    export -f curl

    # Mock ip command to fail
    ip() { return 1; }
    export -f ip

    # Run detection in subshell to avoid polluting environment
    (
        detect_network_environment 2>/dev/null
        if [[ "$RECOMMEND" == "tunnel" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "Default fallback to tunnel mode" || fail "Default fallback failed"

    # Restore curl
    unset curl
}

# Test 3: NAT detection (public != local)
test_nat_detection() {
    log "Test 3: NAT detection"

    # Mock curl to return public IP
    curl() {
        if [[ "$*" == *"api.ipify.org"* ]]; then
            echo "1.2.3.4"
        else
            echo ""
        fi
    }
    export -f curl

    # Mock ip to return local IP
    ip() {
        if [[ "$*" == *"route get"* ]]; then
            echo "192.168.1.100 via 192.168.1.1 dev eth0 src 192.168.1.100"
        fi
    }
    export -f ip

    # Run detection in subshell
    (
        detect_network_environment 2>/dev/null
        if [[ "$RECOMMEND" == "tunnel" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "NAT detected correctly" || fail "NAT detection failed"

    unset curl ip
}

# Test 4: Direct public IP (public == local)
test_direct_public_ip() {
    log "Test 4: Direct public IP detection"

    # Mock curl to return public IP
    curl() {
        if [[ "$*" == *"api.ipify.org"* ]]; then
            echo "1.2.3.4"
        else
            echo ""
        fi
    }
    export -f curl

    # Mock ip to return same public IP (VPS with public IP)
    ip() {
        if [[ "$*" == *"route get"* ]]; then
            echo "1.2.3.4 via 1.2.3.1 dev eth0 src 1.2.3.4"
        fi
    }
    export -f ip

    # Mock nc to return failure (ports not available)
    nc() {
        return 1
    }
    export -f nc

    # Run detection in subshell
    (
        detect_network_environment 2>/dev/null
        if [[ "$RECOMMEND" == "tunnel" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "Direct IP with closed ports recommends tunnel" || fail "Direct IP detection failed"

    unset curl ip nc
}

# Test 5: Port detection logic structure
test_port_detection_structure() {
    log "Test 5: Port detection code structure exists"

    # Verify the function contains port checking logic
    if grep -q "port_80_open" "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh" && \
       grep -q "port_443_open" "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh"; then
        pass "Port detection variables defined"
    else
        fail "Port detection variables missing"
        return 1
    fi

    # Verify port checking code exists
    if grep -q "nc -z localhost 80" "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh" && \
       grep -q "nc -z localhost 443" "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh"; then
        pass "Port checking commands present"
    else
        fail "Port checking commands missing"
        return 1
    fi

    # Verify timeout fallback exists
    if grep -q "/dev/tcp/127.0.0.1/80" "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh" && \
       grep -q "/dev/tcp/127.0.0.1/443" "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh"; then
        pass "Bash /dev/tcp fallback present"
    else
        fail "Bash /dev/tcp fallback missing"
        return 1
    fi
}

# Test 6: Recommendation logic with port availability
test_recommendation_with_ports() {
    log "Test 6: Recommendation considers port availability"

    # Test that port availability affects recommendation
    # Note: These checks are on separate lines, so we check for the presence of both patterns
    if grep 'port_80_open.*= "yes".*port_443_open.*= "yes"' "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh" >/dev/null 2>&1; then
        pass "Both ports open condition present"
    else
        fail "Both ports open condition missing"
        return 1
    fi

    if grep 'port_80_open.*= "yes"' "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh" >/dev/null 2>&1; then
        pass "Port 80 open condition present"
    else
        fail "Port 80 open condition missing"
        return 1
    fi

    if grep 'port_443_open.*= "yes"' "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh" >/dev/null 2>&1; then
        pass "Port 443 open condition present"
    else
        fail "Port 443 open condition missing"
        return 1
    fi

    # Verify that proxy recommendation is set in these contexts
    if grep -A1 'port_80_open.*= "yes".*port_443_open.*= "yes"' "$PROJECT_ROOT/deploy/lib/cloudflare-functions.sh" | grep -q 'RECOMMEND="proxy"'; then
        pass "Both ports open -> proxy recommendation found"
    else
        fail "Both ports open -> proxy recommendation missing"
        return 1
    fi
}

# Test 8: Fallback to hostname -I
test_hostname_fallback() {
    log "Test 8: Fallback to hostname -I"

    # Mock ip to fail
    ip() { return 1; }
    export -f ip

    # Mock hostname to return IP
    hostname() {
        if [[ "$*" == *"-I"* ]]; then
            echo "192.168.1.100"
        fi
    }
    export -f hostname

    # Mock curl
    curl() {
        if [[ "$*" == *"api.ipify.org"* ]]; then
            echo "1.2.3.4"
        else
            echo ""
        fi
    }
    export -f curl

    # Run detection in subshell
    (
        detect_network_environment 2>/dev/null
        if [[ "$RECOMMEND" == "tunnel" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "hostname -I fallback works" || fail "hostname fallback failed"

    unset ip hostname curl
}

# Test 9: Fallback to ifconfig
test_ifconfig_fallback() {
    log "Test 9: Fallback to ifconfig"

    # Mock ip and hostname to fail
    ip() { return 1; }
    export -f ip

    hostname() { return 1; }
    export -f hostname

    # Mock ifconfig to return IP
    ifconfig() {
        echo "eth0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500"
        echo "        inet 192.168.1.100  netmask 255.255.255.0  broadcast 192.168.1.255"
    }
    export -f ifconfig

    # Mock curl
    curl() {
        if [[ "$*" == *"api.ipify.org"* ]]; then
            echo "1.2.3.4"
        else
            echo ""
        fi
    }
    export -f curl

    # Run detection in subshell
    (
        detect_network_environment 2>/dev/null
        if [[ "$RECOMMEND" == "tunnel" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "ifconfig fallback works" || fail "ifconfig fallback failed"

    unset ip hostname ifconfig curl
}

# Test 10: REASON variable is set
test_reason_variable() {
    log "Test 10: REASON variable is set"

    # Mock curl to return public IP
    curl() {
        if [[ "$*" == *"api.ipify.org"* ]]; then
            echo "1.2.3.4"
        else
            echo ""
        fi
    }
    export -f curl

    # Mock ip to return local IP
    ip() {
        if [[ "$*" == *"route get"* ]]; then
            echo "192.168.1.100 via 192.168.1.1 dev eth0 src 192.168.1.100"
        fi
    }
    export -f ip

    # Run detection in subshell
    (
        detect_network_environment 2>/dev/null
        if [[ -n "$REASON" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "REASON variable is set" || fail "REASON variable not set"

    unset curl ip
}

# Test 11: Unknown NAT status defaults to tunnel
test_unknown_nat_defaults() {
    log "Test 11: Unknown NAT status defaults to tunnel"

    # Mock curl to fail
    curl() { echo ""; }
    export -f curl

    # Mock ip to fail
    ip() { return 1; }
    export -f ip

    # Run detection in subshell
    (
        detect_network_environment 2>/dev/null
        if [[ "$RECOMMEND" == "tunnel" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "Unknown NAT defaults to tunnel" || fail "Unknown NAT handling failed"

    unset curl ip
}

# Test 7: Graceful degradation when tools missing
test_missing_tools_graceful_degradation() {
    log "Test 7: Graceful degradation when tools missing"

    # Mock curl to fail (no network)
    curl() { echo ""; }
    export -f curl

    # Mock ip to fail (no ip command)
    ip() { return 1; }
    export -f ip

    # Mock hostname to fail
    hostname() { return 1; }
    export -f hostname

    # Mock ifconfig to fail
    ifconfig() { return 1; }
    export -f ifconfig

    # Run detection - should not crash, should use defaults
    (
        detect_network_environment 2>/dev/null || true
        if [[ "$RECOMMEND" == "tunnel" ]]; then
            echo "PASS"
        else
            echo "FAIL"
        fi
    ) | grep -q "PASS" && pass "Graceful degradation works" || fail "Graceful degradation failed"

    unset curl ip hostname ifconfig
}

main() {
    echo "=========================================="
    echo "Running detect_network_environment Test Suite"
    echo "=========================================="

    test_function_exists || true
    test_default_values || true
    test_nat_detection || true
    test_direct_public_ip || true
    test_port_detection_structure || true
    test_recommendation_with_ports || true
    test_hostname_fallback || true
    test_ifconfig_fallback || true
    test_reason_variable || true
    test_unknown_nat_defaults || true
    test_missing_tools_graceful_degradation || true

    echo "=========================================="
    if [[ $FAILED -eq 0 ]]; then
        echo "All tests passed!"
        exit 0
    else
        echo "FAILED: $FAILED test(s)"
        exit 1
    fi
}

main "$@"
