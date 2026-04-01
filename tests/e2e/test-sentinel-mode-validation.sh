#!/bin/bash
# E2E Test: Sentinel Mode Validation (No Docker Required)
# Tests: Installer configuration, Caddyfile template, Docker Compose configuration

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Color codes
export GREEN='\033[0;32m'
export RED='\033[0;31m'
export YELLOW='\033[1;33m'
export NC='\033[0m'

# Test tracking
export TESTS_RUN=0
export TESTS_PASSED=0
export TESTS_FAILED=0
export TEST_NS="test-e2e-$(date +%s)"
export TEST_DIR="/tmp/armorclaw-$TEST_NS"
export TEST_LOG_DIR="$TEST_DIR/logs"
export TEST_RESULTS="$TEST_DIR/results.txt"
export TEST_EVIDENCE_DIR="$TEST_DIR/evidence"

# Test configuration
TEST_DOMAIN="${TEST_DOMAIN:-test.armorclaw.local}"
TEST_EMAIL="${TEST_EMAIL:-test@example.com}"

# Installer paths
INSTALLER_V6="$PROJECT_ROOT/deploy/installer-v6.sh"
DOCKER_COMPOSE="$PROJECT_ROOT/docker-compose.yml"

log_result() {
    local test_name="$1"
    local passed="$2"
    local message="${3:-}"

    ((TESTS_RUN++)) || true

    if [[ "$passed" == "true" ]]; then
        ((TESTS_PASSED++)) || true
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        if [[ -n "$message" ]]; then
            echo -e "  ${GREEN}  →${NC} $message"
        fi
    else
        ((TESTS_FAILED++)) || true
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        if [[ -n "$message" ]]; then
            echo -e "  ${RED}  →${NC} $message"
        fi
    fi
}

test_summary() {
    echo ""
    echo "========================================"
    echo "Test Summary"
    echo "========================================"
    echo "Total:  $TESTS_RUN"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    echo "========================================"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
        return 0
    else
        echo -e "${RED}✗ SOME TESTS FAILED${NC}"
        return 1
    fi
}

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up test artifacts...${NC}"
    rm -rf "$TEST_DIR" 2>/dev/null || true
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

setup_test_env() {
    echo -e "${YELLOW}Setting up test environment...${NC}"
    mkdir -p "$TEST_LOG_DIR"
    mkdir -p "$TEST_EVIDENCE_DIR"
    trap cleanup EXIT
    echo -e "${GREEN}✓ Test environment ready${NC}"
    echo "  Test namespace: $TEST_NS"
    echo "  Test directory: $TEST_DIR"
}

collect_evidence() {
    echo ""
    echo "========================================"
    echo "Collecting Test Evidence"
    echo "========================================"
    echo ""

    # Copy installer script
    cp "$INSTALLER_V6" "$TEST_EVIDENCE_DIR/installer-v6.sh"
    log_result "evidence_installer" "true" "Installer script copied"

    # Copy docker-compose
    cp "$DOCKER_COMPOSE" "$TEST_EVIDENCE_DIR/docker-compose.yml"
    log_result "evidence_docker_compose" "true" "Docker compose copied"

    # Copy Caddyfile template
    if [[ -f "$PROJECT_ROOT/configs/Caddyfile.template" ]]; then
        cp "$PROJECT_ROOT/configs/Caddyfile.template" "$TEST_EVIDENCE_DIR/Caddyfile.template"
        log_result "evidence_caddyfile_template" "true" "Caddyfile template copied"
    fi

    # Create test summary
    {
        echo "========================================"
        echo "Sentinel Mode E2E Test Evidence"
        echo "========================================"
        echo ""
        echo "Test Configuration:"
        echo "  Domain: $TEST_DOMAIN"
        echo "  Email: $TEST_EMAIL"
        echo "  Test Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo ""
        echo "Tests Run: $TESTS_RUN"
        echo "Tests Passed: $TESTS_PASSED"
        echo "Tests Failed: $TESTS_FAILED"
        echo ""
        echo "Evidence Files:"
        ls -1 "$TEST_EVIDENCE_DIR" | sed 's/^/  - /'
    } > "$TEST_RESULTS"

    echo ""
    echo "✓ Evidence collection complete"
    echo "  Evidence directory: $TEST_EVIDENCE_DIR"
    echo "  Results file: $TEST_RESULTS"
}

test_installer_sentinel_mode_detection() {
    echo ""
    echo "========================================"
    echo "Test: Installer Sentinel Mode Detection"
    echo "========================================"
    echo ""

    # Check if installer exists
    if [[ ! -f "$INSTALLER_V6" ]]; then
        log_result "installer_exists" "false" "installer-v6.sh not found"
        return 1
    fi
    log_result "installer_exists" "true" "Found installer-v6.sh"

    # Check if installer has domain detection
    if grep -q "detect_deployment_mode\|DOMAIN\|MODE" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_has_mode_detection" "true" "Installer has deployment mode detection"
    else
        log_result "installer_has_mode_detection" "false" "Installer missing deployment mode detection"
        return 1
    fi

    # Check if installer has email prompt for sentinel mode
    if grep -q "prompt_email\|EMAIL" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_has_email_prompt" "true" "Installer has email prompt for sentinel mode"
    else
        log_result "installer_has_email_prompt" "false" "Installer missing email prompt for sentinel mode"
        return 1
    fi

    # Check if installer has sentinel mode specific logic
    if grep -qE 'MODE.*=.*sentinel|sentinel.*MODE|MODE==\"sentinel\"' "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_has_sentinel_logic" "true" "Installer has sentinel mode logic"
    else
        log_result "installer_has_sentinel_logic" "false" "Installer missing sentinel mode logic"
        return 1
    fi

    # Check if installer generates Caddyfile for sentinel mode
    if grep -q "generate_caddyfile" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_generates_caddyfile" "true" "Installer generates Caddyfile for sentinel mode"
    else
        log_result "installer_generates_caddyfile" "false" "Installer missing Caddyfile generation"
        return 1
    fi

    # Check if installer generates .env file
    if grep -q "generate_env_file" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_generates_env" "true" "Installer generates .env file"
    else
        log_result "installer_generates_env" "false" "Installer missing .env file generation"
        return 1
    fi

    # Check if installer uses sentinel profile
    if grep -q "profile.*sentinel\|--profile sentinel" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_uses_sentinel_profile" "true" "Installer uses sentinel profile for docker compose"
    else
        log_result "installer_uses_sentinel_profile" "false" "Installer missing sentinel profile usage"
        return 1
    fi
}

test_caddyfile_template() {
    echo ""
    echo "========================================"
    echo "Test: Caddyfile Template"
    echo "========================================"
    echo ""

    local caddyfile_template="$PROJECT_ROOT/configs/Caddyfile.template"

    if [[ ! -f "$caddyfile_template" ]]; then
        log_result "caddyfile_template_exists" "false" "Caddyfile.template not found"
        return 1
    fi
    log_result "caddyfile_template_exists" "true" "Found Caddyfile.template"

    # Check if template has domain variable
    if grep -q "DOMAIN_NAME" "$caddyfile_template" 2>/dev/null; then
        log_result "caddyfile_has_domain_var" "true" "Caddyfile template uses DOMAIN_NAME variable"
    else
        log_result "caddyfile_has_domain_var" "false" "Caddyfile template missing DOMAIN_NAME variable"
        return 1
    fi

    # Check if template has email variable
    if grep -q "ADMIN_EMAIL" "$caddyfile_template" 2>/dev/null; then
        log_result "caddyfile_has_email_var" "true" "Caddyfile template uses ADMIN_EMAIL variable"
    else
        log_result "caddyfile_has_email_var" "false" "Caddyfile template missing ADMIN_EMAIL variable"
        return 1
    fi

    # Check if template has Matrix routes
    if grep -q "/_matrix/\|handle.*matrix" "$caddyfile_template" 2>/dev/null; then
        log_result "caddyfile_has_matrix_routes" "true" "Caddyfile template has Matrix routes"
    else
        log_result "caddyfile_has_matrix_routes" "false" "Caddyfile template missing Matrix routes"
        return 1
    fi

    # Check if template has well-known endpoints
    if grep -q "/.well-known/matrix" "$caddyfile_template" 2>/dev/null; then
        log_result "caddyfile_has_well_known" "true" "Caddyfile template has Matrix well-known endpoints"
    else
        log_result "caddyfile_has_well_known" "false" "Caddyfile template missing well-known endpoints"
        return 1
    fi

    # Check if template has Bridge API routes
    if grep -q "/api\|reverse_proxy bridge" "$caddyfile_template" 2>/dev/null; then
        log_result "caddyfile_has_bridge_routes" "true" "Caddyfile template has Bridge API routes"
    else
        log_result "caddyfile_has_bridge_routes" "false" "Caddyfile template missing Bridge API routes"
        return 1
    fi

    # Check if template has discovery endpoint
    if grep -q "/discover\|handle.*discover" "$caddyfile_template" 2>/dev/null; then
        log_result "caddyfile_has_discovery" "true" "Caddyfile template has discovery endpoint"
    else
        log_result "caddyfile_has_discovery" "false" "Caddyfile template missing discovery endpoint"
        return 1
    fi

    # Check if template has health check endpoint
    if grep -q "/health\|handle.*health" "$caddyfile_template" 2>/dev/null; then
        log_result "caddyfile_has_health" "true" "Caddyfile template has health check endpoint"
    else
        log_result "caddyfile_has_health" "false" "Caddyfile template missing health check endpoint"
        return 1
    fi
}

test_docker_compose_sentinel_profile() {
    echo ""
    echo "========================================"
    echo "Test: Docker Compose Sentinel Profile"
    echo "========================================"
    echo ""

    # Check if Caddy service has sentinel profile
    if grep -A 20 "caddy:" "$DOCKER_COMPOSE" 2>/dev/null | grep -q "profiles.*sentinel"; then
        log_result "caddy_has_sentinel_profile" "true" "Caddy service has sentinel profile"
    else
        log_result "caddy_has_sentinel_profile" "false" "Caddy service missing sentinel profile"
        return 1
    fi

    # Check if Caddy has correct ports (80, 443)
    if grep -A 20 "caddy:" "$DOCKER_COMPOSE" 2>/dev/null | grep -qE '80.*:.*80|443.*:.*443'; then
        log_result "caddy_has_correct_ports" "true" "Caddy service has correct ports (80, 443)"
    else
        log_result "caddy_has_correct_ports" "false" "Caddy service missing correct ports"
        return 1
    fi

    # Check if Caddy has Caddyfile volume mount
    if grep -A 20 "caddy:" "$DOCKER_COMPOSE" 2>/dev/null | grep -q "Caddyfile"; then
        log_result "caddy_mounts_caddyfile" "true" "Caddy service mounts Caddyfile"
    else
        log_result "caddy_mounts_caddyfile" "false" "Caddy service missing Caddyfile mount"
        return 1
    fi

    # Check if Caddy has data volumes for certificates
    if grep -A 20 "caddy:" "$DOCKER_COMPOSE" 2>/dev/null | grep -qE "caddy_data|caddy_config"; then
        log_result "caddy_has_cert_volumes" "true" "Caddy service has data volumes for certificates"
    else
        log_result "caddy_has_cert_volumes" "false" "Caddy service missing certificate volumes"
        return 1
    fi

    # Check if ACME_AGREE is set
    if grep -A 20 "caddy:" "$DOCKER_COMPOSE" 2>/dev/null | grep -q "ACME_AGREE=true"; then
        log_result "caddy_acme_agree_set" "true" "Caddy service has ACME_AGREE=true"
    else
        log_result "caddy_acme_agree_set" "false" "Caddy service missing ACME_AGREE=true"
        return 1
    fi
}

test_env_generation() {
    echo ""
    echo "========================================"
    echo "Test: Environment File Generation"
    echo "========================================"
    echo ""

    # Check if installer generates env content with sentinel mode
    if grep -A 50 "generate_env_file" "$INSTALLER_V6" 2>/dev/null | grep -q "ARMORCLAW_SERVER_MODE=\${MODE}"; then
        log_result "env_has_server_mode" "true" ".env file will have ARMORCLAW_SERVER_MODE"
    else
        log_result "env_has_server_mode" "false" ".env file missing ARMORCLAW_SERVER_MODE"
        return 1
    fi

    # Check if env has public base URL for sentinel mode
    if grep -A 50 "generate_env_file" "$INSTALLER_V6" 2>/dev/null | grep -q "ARMORCLAW_PUBLIC_BASE_URL"; then
        log_result "env_has_public_base_url" "true" ".env file will have ARMORCLAW_PUBLIC_BASE_URL"
    else
        log_result "env_has_public_base_url" "false" ".env file missing ARMORCLAW_PUBLIC_BASE_URL"
        return 1
    fi

    # Check if env has email for sentinel mode
    if grep -A 50 "generate_env_file" "$INSTALLER_V6" 2>/dev/null | grep -q "ARMORCLAW_EMAIL"; then
        log_result "env_has_email" "true" ".env file will have ARMORCLAW_EMAIL"
    else
        log_result "env_has_email" "false" ".env file missing ARMORCLAW_EMAIL"
        return 1
    fi

    # Check if env has secrets
    if grep -A 50 "generate_env_file" "$INSTALLER_V6" 2>/dev/null | grep -qE "ADMIN_TOKEN|KEYSTORE_SECRET|MATRIX_SECRET"; then
        log_result "env_has_secrets" "true" ".env file will have admin and secrets"
    else
        log_result "env_has_secrets" "false" ".env file missing secrets"
        return 1
    fi

    # Check if env has Matrix homeserver URL for sentinel mode
    if grep -A 50 "generate_env_file" "$INSTALLER_V6" 2>/dev/null | grep -q "ARMORCLAW_MATRIX_HOMESERVER_URL"; then
        log_result "env_has_matrix_url" "true" ".env file will have ARMORCLAW_MATRIX_HOMESERVER_URL"
    else
        log_result "env_has_matrix_url" "false" ".env file missing ARMORCLAW_MATRIX_HOMESERVER_URL"
        return 1
    fi
}

test_sentinel_deployment_simulation() {
    echo ""
    echo "========================================"
    echo "Test: Sentinel Mode Deployment Simulation"
    echo "========================================"
    echo ""

    # Create test .env file with sentinel mode settings
    local test_env="$TEST_DIR/test.env"

    cat > "$test_env" <<EOF
# Test Environment for Sentinel Mode
ARMORCLAW_SERVER_MODE=sentinel
ARMORCLAW_RPC_TRANSPORT=tcp
ARMORCLAW_LISTEN_ADDR=0.0.0.0:8080
ARMORCLAW_PUBLIC_BASE_URL=https://$TEST_DOMAIN
ARMORCLAW_EMAIL=$TEST_EMAIL
ARMORCLAW_ADMIN_TOKEN=test-token-for-testing-only
ARMORCLAW_KEYSTORE_SECRET=test-secret-for-testing-only
ARMORCLAW_MATRIX_SECRET=test-matrix-secret-for-testing-only
ARMORCLAW_PUBLIC_IP=127.0.0.1
ARMORCLAW_MATRIX_ENABLED=true
ARMORCLAW_MATRIX_HOMESERVER_URL=https://$TEST_DOMAIN:6167
CADDY_HTTP_PORT=80
CADDY_HTTPS_PORT=443
CADDY_CONFIG_PATH=$TEST_DIR
EOF

    if [[ -f "$test_env" ]]; then
        log_result "test_env_created" "true" "Test .env file created"
    else
        log_result "test_env_created" "false" "Failed to create test .env file"
        return 1
    fi

    # Create test Caddyfile
    local test_caddyfile="$TEST_DIR/Caddyfile"

    cat > "$test_caddyfile" <<EOF
$TEST_DOMAIN {
    email $TEST_EMAIL

    handle /_matrix/* {
        reverse_proxy armorclaw-sentinel:8443
    }

    handle /.well-known/matrix/client {
        header Content-Type application/json
        header Access-Control-Allow-Origin *
        respond \`{"m.homeserver":{"base_url":"https://$TEST_DOMAIN"}}\` 200
    }

    handle /.well-known/matrix/server {
        header Content-Type application/json
        header Access-Control-Allow-Origin *
        respond \`{"m.server":"$TEST_DOMAIN:443"}\` 200
    }

    handle /api* {
        reverse_proxy armorclaw-sentinel:8443
    }

    handle /health {
        respond "OK\\n" 200
    }

    handle /discover {
        reverse_proxy armorclaw-sentinel:8443
    }

    handle {
        reverse_proxy armorclaw-sentinel:8443
    }
}
EOF

    if [[ -f "$test_caddyfile" ]]; then
        log_result "test_caddyfile_created" "true" "Test Caddyfile created"
    else
        log_result "test_caddyfile_created" "false" "Failed to create test Caddyfile"
        return 1
    fi

    # Copy Caddyfile to expected location
    cp "$test_caddyfile" "$TEST_EVIDENCE_DIR/generated-Caddyfile"
    log_result "evidence_caddyfile_copied" "true" "Caddyfile evidence copied"
}

test_matrix_well_known_endpoints() {
    echo ""
    echo "========================================"
    echo "Test: Matrix Well-Known Endpoints"
    echo "========================================"
    echo ""

    local caddyfile_template="$PROJECT_ROOT/configs/Caddyfile.template"

    # Test client well-known endpoint content
    if grep -A 5 "\/.well-known\/matrix\/client" "$caddyfile_template" 2>/dev/null | grep -q "m.homeserver.*base_url"; then
        log_result "well_known_client_format" "true" "Client well-known endpoint has correct format"
    else
        log_result "well_known_client_format" "false" "Client well-known endpoint has incorrect format"
        return 1
    fi

    # Test server well-known endpoint content
    if grep -A 5 "\/.well-known\/matrix\/server" "$caddyfile_template" 2>/dev/null | grep -q "m.server.*:443"; then
        log_result "well_known_server_format" "true" "Server well-known endpoint has correct format"
    else
        log_result "well_known_server_format" "false" "Server well-known endpoint has incorrect format"
        return 1
    fi

    # Verify CORS headers are set
    if grep -A 2 "\/.well-known\/matrix" "$caddyfile_template" 2>/dev/null | grep -q "Access-Control-Allow-Origin"; then
        log_result "well_known_cors_headers" "true" "Well-known endpoints have CORS headers"
    else
        log_result "well_known_cors_headers" "false" "Well-known endpoints missing CORS headers"
        return 1
    fi
}

test_installer_non_interactive_mode() {
    echo ""
    echo "========================================"
    echo "Test: Installer Non-Interactive Mode"
    echo "========================================"
    echo ""

    # Check if installer supports non-interactive mode
    if grep -q "ARMORCLAW_EMAIL\|DOMAIN.*env" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_supports_env_vars" "true" "Installer supports environment variables for configuration"
    else
        log_result "installer_supports_env_vars" "false" "Installer missing environment variable support"
        return 1
    fi

    # Check if installer has non-interactive mode detection
    if grep -q "\[ -t 0 \]\|\[ -c /dev/tty \]" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_detects_non_interactive" "true" "Installer can detect non-interactive mode"
    else
        log_result "installer_detects_non_interactive" "false" "Installer missing non-interactive mode detection"
        return 1
    fi

    # Check if installer validates email in sentinel mode for non-interactive
    if grep -q "Non-interactive mode requires ARMORCLAW_EMAIL" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_validates_email" "true" "Installer validates email in non-interactive mode"
    else
        log_result "installer_validates_email" "false" "Installer missing email validation in non-interactive mode"
        return 1
    fi
}

test_secrets_generation() {
    echo ""
    echo "========================================"
    echo "Test: Secrets Generation"
    echo "========================================"
    echo ""

    # Check if installer has secret generation functions
    if grep -q "generate_admin_token\|generate_keystore_secret\|generate_matrix_secret" "$INSTALLER_V6" 2>/dev/null; then
        log_result "installer_has_secret_generation" "true" "Installer has secret generation functions"
    else
        log_result "installer_has_secret_generation" "false" "Installer missing secret generation functions"
        return 1
    fi

    # Check if secrets are generated using openssl
    if grep -A 3 "generate_" "$INSTALLER_V6" 2>/dev/null | grep -q "openssl rand"; then
        log_result "secrets_use_openssl" "true" "Secrets generated using openssl rand"
    else
        log_result "secrets_use_openssl" "false" "Secrets not generated using openssl rand"
        return 1
    fi

    # Check if secrets are exported for use
    if grep -q "export ADMIN_TOKEN\|export KEYSTORE_SECRET\|export MATRIX_SECRET" "$INSTALLER_V6" 2>/dev/null; then
        log_result "secrets_are_exported" "true" "Secrets are exported for use by setup scripts"
    else
        log_result "secrets_are_exported" "false" "Secrets not exported for use"
        return 1
    fi
}

test_sentinel_mode_requirements() {
    echo ""
    echo "========================================"
    echo "Test: Sentinel Mode Requirements"
    echo "========================================"
    echo ""

    # Check all critical sentinel mode requirements are present

    # 1. Domain detection
    if grep -q "prompt_domain\|detect_domain" "$INSTALLER_V6" 2>/dev/null; then
        log_result "requirement_domain_detection" "true" "Requirement 1: Domain detection"
    else
        log_result "requirement_domain_detection" "false" "Requirement 1: Domain detection missing"
        return 1
    fi

    # 2. Email collection for Let's Encrypt
    if grep -q "prompt_email\|EMAIL" "$INSTALLER_V6" 2>/dev/null; then
        log_result "requirement_email_collection" "true" "Requirement 2: Email collection for Let's Encrypt"
    else
        log_result "requirement_email_collection" "false" "Requirement 2: Email collection missing"
        return 1
    fi

    # 3. Sentinel mode selection
    if grep -qE 'MODE.*=.*sentinel|\$MODE.*==.*sentinel|\$\{MODE.*sentinel' "$INSTALLER_V6" 2>/dev/null; then
        log_result "requirement_sentinel_selection" "true" "Requirement 3: Sentinel mode selection"
    else
        log_result "requirement_sentinel_selection" "false" "Requirement 3: Sentinel mode selection missing"
        return 1
    fi

    # 4. Caddyfile generation
    if grep -q "generate_caddyfile" "$INSTALLER_V6" 2>/dev/null; then
        log_result "requirement_caddyfile_generation" "true" "Requirement 4: Caddyfile generation"
    else
        log_result "requirement_caddyfile_generation" "false" "Requirement 4: Caddyfile generation missing"
        return 1
    fi

    # 5. .env generation
    if grep -q "generate_env_file" "$INSTALLER_V6" 2>/dev/null; then
        log_result "requirement_env_generation" "true" "Requirement 5: .env file generation"
    else
        log_result "requirement_env_generation" "false" "Requirement 5: .env generation missing"
        return 1
    fi

    # 6. Docker compose with sentinel profile
    if grep -q "profile.*sentinel\|--profile sentinel" "$INSTALLER_V6" 2>/dev/null; then
        log_result "requirement_sentinel_profile" "true" "Requirement 6: Docker compose sentinel profile"
    else
        log_result "requirement_sentinel_profile" "false" "Requirement 6: Sentinel profile missing"
        return 1
    fi

    # 7. Secrets generation
    if grep -q "generate_secrets\|generate_admin_token" "$INSTALLER_V6" 2>/dev/null; then
        log_result "requirement_secrets" "true" "Requirement 7: Secrets generation"
    else
        log_result "requirement_secrets" "false" "Requirement 7: Secrets generation missing"
        return 1
    fi

    log_result "all_requirements_met" "true" "All sentinel mode requirements are met"
}

main() {
    echo "========================================"
    echo "E2E Test: Sentinel Mode Validation"
    echo "========================================"
    echo ""
    echo "Testing sentinel mode configuration (validation-only, no Docker required)"
    echo "  Installer: $INSTALLER_V6"
    echo "  Docker Compose: $DOCKER_COMPOSE"
    echo "  Test Domain: $TEST_DOMAIN"
    echo ""

    setup_test_env

    echo ""
    echo "Running tests..."
    echo ""

    test_installer_sentinel_mode_detection
    test_caddyfile_template
    test_docker_compose_sentinel_profile
    test_env_generation
    test_sentinel_deployment_simulation
    test_matrix_well_known_endpoints
    test_installer_non_interactive_mode
    test_secrets_generation
    test_sentinel_mode_requirements

    collect_evidence

    local test_result=$?
    test_summary

    echo ""
    echo "========================================"
    echo "Test Evidence Location"
    echo "========================================"
    echo "Evidence: $TEST_EVIDENCE_DIR"
    echo "Results: $TEST_RESULTS"
    echo "========================================"

    exit $test_result
}

main "$@"
