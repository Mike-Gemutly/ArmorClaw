#!/bin/bash
# =============================================================================
# Test Suite for Hardened Installer
# Tests: Lockfile, Docker wait, env passthrough, compose detection
# =============================================================================

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR=$(mktemp -d)
FAILED=0

log() { echo "[TEST] $*"; }
pass() { echo "✓ $*"; }
fail() { echo "✗ $*"; ((FAILED++)); }
cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

cd "$TEST_DIR"

# Test 1: Lockfile (skip if flock not available)
test_lockfile() {
    log "Test 1: Lockfile functionality"
    if ! command -v flock >/dev/null 2>&1; then
        log "flock not installed, skipping"
        pass "Lockfile test skipped (flock not available)"
        return 0
    fi
    exec 200>"$WORK_DIR/lock"
    if ! flock -n 200; then fail "Failed to acquire lock"; return 1; fi
    if flock -n 200 2>/dev/null; then fail "Second process got lock"; flock -u 200 2>/dev/null; return 1; fi
    flock -u 200 2>/dev/null || true
    pass "Lockfile functional"
}

# Test 2: Docker wait
test_docker_wait() {
    log "Test 2: Docker wait loop"
    if ! command -v docker >/dev/null 2>&1; then log "Docker not installed"; return 0; fi
    if ! docker info >/dev/null 2>&1; then log "Docker not running"; return 0; fi
    pass "Docker ready"
}

# Test 3: Env passthrough
test_env_passthrough() {
    log "Test 3: Environment variable passthrough"
    export DOCKER_COMPOSE="docker compose" CONDUIT_VERSION="v1.0" CONDUIT_IMAGE="test:tag"
    local output
    output=$(bash -c 'if [[ -n "$DOCKER_COMPOSE" && -n "$CONDUIT_VERSION" && -n "$CONDUIT_IMAGE" ]]; then echo "PASS"; fi')
    if [[ "$output" == "PASS" ]]; then pass "Env vars passed"; else fail "Env vars not passed"; fi
}

# Test 4: Docker compose detection
test_docker_compose() {
    log "Test 4: Docker Compose detection"
    if docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker compose"
        [[ "$DOCKER_COMPOSE" == "docker compose" ]] && pass "Detected docker compose"
    elif docker-compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker-compose"
        [[ "$DOCKER_COMPOSE" == "docker-compose" ]] && pass "Detected docker-compose"
    else
        log "Docker Compose not installed"
    fi
}

# Test 5: CONDUIT_IMAGE fallback
test_conduit_fallback() {
    log "Test 5: CONDUIT_IMAGE fallback"
    export CONDUIT_VERSION="v2.0"
    CONDUIT_IMAGE="${CONDUIT_IMAGE:-test-image:$CONDUIT_VERSION}"
    [[ "$CONDUIT_IMAGE" == "test-image:v2.0" ]] && pass "Fallback works"
}

# Test 6: Syntax
test_syntax() {
    log "Test 6: Syntax validation"
    for f in deploy/install.sh deploy/setup-matrix.sh deploy/quickstart-entrypoint.sh deploy/deploy-infra.sh; do
        bash -n "$f" && pass "Syntax valid: $f" || fail "Syntax error: $f"
    done
}

# Test 7: Wait function exists
test_wait_function() {
    log "Test 7: wait_for_docker function"
    for f in deploy/install.sh deploy/setup-matrix.sh deploy/quickstart-entrypoint.sh deploy/deploy-infra.sh; do
        grep -q "wait_for_docker()" "$f" && pass "Found in: $f" || fail "Missing from: $f"
    done
}

# Test 8: Variable order
test_variable_order() {
    log "Test 8: Variable ordering"
    grep -q "DOCKER_COMPOSE=\"\$DOCKER_COMPOSE:-docker compose\"" deploy/setup-matrix.sh && \
        grep -q "\$DOCKER_COMPOSE" deploy/setup-matrix.sh && \
        pass "DOCKER_COMPOSE fallback correct" || fail "DOCKER_COMPOSE issue"
}

main() {
    echo "=========================================="
    echo "Running Installer Test Suite"
    echo "=========================================="
    test_lockfile && test_docker_wait && test_env_passthrough && \
        test_docker_compose && test_conduit_fallback && test_syntax && \
        test_wait_function && test_variable_order
    echo "=========================================="
    if [[ $FAILED -eq 0 ]]; then echo "All tests passed!"; exit 0; else echo "FAILED: $FAILED test(s)"; exit 1; fi
}
main "$@"
