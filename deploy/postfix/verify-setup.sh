#!/usr/bin/env bash
# ArmorClaw Postfix Verification
# Comprehensive health checks for the Postfix + ArmorClaw email pipeline
#
# Usage: sudo ./deploy/postfix/verify-setup.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Counters
PASSED=0
FAILED=0
SKIPPED=0
WARNINGS=0

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Postfix Verification${NC}                     ${CYAN}║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "\n${BLUE}▶${NC} ${BOLD}$1${NC}"
}

print_success() {
    echo -e "  ${GREEN}✓${NC} $1"
    ((PASSED++))
}

print_error() {
    echo -e "  ${RED}✗${NC} $1"
    ((FAILED++))
}

print_warning() {
    echo -e "  ${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

print_skip() {
    echo -e "  ${CYAN}⊘${NC} $1"
    ((SKIPPED++))
}

print_info() {
    echo -e "  ${CYAN}ℹ${NC} $1"
}

#=============================================================================
# 1. Binary Check
#=============================================================================

check_binary() {
    print_step "Checking binary..."

    if test -x /usr/local/bin/armorclaw-mta-recv; then
        print_success "armorclaw-mta-recv binary found and executable"
    else
        if test -f /usr/local/bin/armorclaw-mta-recv; then
            print_error "armorclaw-mta-recv exists but is NOT executable"
        else
            print_error "armorclaw-mta-recv binary not found at /usr/local/bin/armorclaw-mta-recv"
        fi
    fi
}

#=============================================================================
# 2. Directory Checks
#=============================================================================

check_directories() {
    print_step "Checking directories..."

    # /run/armorclaw/ — exists, owned by armorclaw:armorclaw-mail
    if test -d /run/armorclaw; then
        local owner permissions
        owner=$(stat -c '%U:%G' /run/armorclaw 2>/dev/null || echo "unknown")
        permissions=$(stat -c '%a' /run/armorclaw 2>/dev/null || echo "unknown")
        if [[ "$owner" == "armorclaw:armorclaw-mail" ]]; then
            print_success "/run/armorclaw exists (${permissions}, ${owner})"
        else
            print_warning "/run/armorclaw exists but owner is ${owner} (expected armorclaw:armorclaw-mail)"
        fi
    else
        print_error "/run/armorclaw does not exist"
    fi

    # /var/lib/armorclaw/email-files/
    if test -d /var/lib/armorclaw/email-files; then
        print_success "/var/lib/armorclaw/email-files exists"
    else
        print_error "/var/lib/armorclaw/email-files does not exist"
    fi

    # /var/log/armorclaw/email/
    if test -d /var/log/armorclaw/email; then
        print_success "/var/log/armorclaw/email exists"
    else
        print_error "/var/log/armorclaw/email does not exist"
    fi

    # /etc/armorclaw/certs/
    if test -d /etc/armorclaw/certs; then
        print_success "/etc/armorclaw/certs exists"
    else
        print_error "/etc/armorclaw/certs does not exist"
    fi
}

#=============================================================================
# 3. Socket Check
#=============================================================================

SOCKET_EXISTS=false

check_socket() {
    print_step "Checking socket..."

    if test -S /run/armorclaw/email-ingest.sock; then
        print_success "/run/armorclaw/email-ingest.sock exists and is a Unix socket"
        SOCKET_EXISTS=true
    else
        print_warning "Socket not found — bridge may not be running"
        print_info "The socket is created by the bridge at startup (not by this script)"
    fi
}

#=============================================================================
# 4. Postfix Check
#=============================================================================

check_postfix() {
    print_step "Checking Postfix..."

    if command -v postfix &>/dev/null; then
        if postfix status &>/dev/null; then
            print_success "Postfix is running"
        else
            print_error "Postfix is installed but not running"
            print_info "Start with: sudo postfix start"
        fi
    else
        print_error "Postfix is not installed"
    fi
}

#=============================================================================
# 5. Transport Check
#=============================================================================

check_transport() {
    print_step "Checking transport routing..."

    if ! test -f /etc/postfix/transport_maps; then
        print_error "/etc/postfix/transport_maps not found"
        return
    fi

    if ! test -f /etc/postfix/transport_maps.db; then
        print_error "/etc/postfix/transport_maps.db not found — run: sudo postmap /etc/postfix/transport_maps"
        return
    fi

    if command -v postmap &>/dev/null; then
        local result
        result=$(postmap -q "test@localhost" /etc/postfix/transport_maps 2>/dev/null || true)
        if [[ "$result" == "armorclaw:" ]]; then
            print_success "transport_maps routes test@localhost → armorclaw:"
        else
            print_error "transport_maps query returned '${result}' (expected 'armorclaw:')"
            print_info "Check /etc/postfix/transport_maps has a catch-all entry"
        fi
    else
        print_error "postmap command not found — Postfix tools may not be installed"
    fi
}

#=============================================================================
# 6. YARA Rules Check
#=============================================================================

check_yara() {
    print_step "Checking YARA rules..."

    if test -f configs/yara_rules.yar; then
        print_success "YARA rules file found (source tree: configs/yara_rules.yar)"
    elif test -f /etc/armorclaw/yara_rules.yar; then
        print_success "YARA rules file found (installed: /etc/armorclaw/yara_rules.yar)"
    else
        # Try relative to project root
        if test -f "$PROJECT_ROOT/configs/yara_rules.yar"; then
            print_success "YARA rules file found ($PROJECT_ROOT/configs/yara_rules.yar)"
        else
            print_error "YARA rules file not found (checked configs/yara_rules.yar and /etc/armorclaw/yara_rules.yar)"
        fi
    fi
}

#=============================================================================
# 7. End-to-End Email Test
#=============================================================================

check_end_to_end() {
    print_step "End-to-end email test..."

    # Skip if bridge not running (socket missing)
    if ! $SOCKET_EXISTS; then
        print_skip "Skipped (bridge not running — socket not found)"
        return
    fi

    # Check for swaks or sendmail
    local email_cmd=""
    if command -v swaks &>/dev/null; then
        email_cmd="swaks"
    elif command -v sendmail &>/dev/null; then
        email_cmd="sendmail"
    fi

    if [[ -z "$email_cmd" ]]; then
        print_skip "Skipped (neither swaks nor sendmail available)"
        return
    fi

    # Send test email
    if [[ "$email_cmd" == "swaks" ]]; then
        if swaks --to verify-test@localhost --from verify@localhost --body "ArmorClaw verify test" --server localhost &>/dev/null; then
            print_success "Test email sent via swaks to verify-test@localhost"
        else
            print_error "swaks failed to send test email"
        fi
    else
        if echo "ArmorClaw verify test" | sendmail verify-test@localhost 2>/dev/null; then
            print_success "Test email sent via sendmail to verify-test@localhost"
        else
            print_error "sendmail failed to send test email"
        fi
    fi
}

#=============================================================================
# 8. Log Writability Check
#=============================================================================

check_log_writable() {
    print_step "Checking log directory..."

    if test -d /var/log/armorclaw/email; then
        if test -w /var/log/armorclaw/email; then
            print_success "/var/log/armorclaw/email is writable"
        else
            print_error "/var/log/armorclaw/email is NOT writable"
            print_info "Fix with: sudo chown armorclaw:armorclaw /var/log/armorclaw/email"
        fi
    else
        # Already caught in directory checks, but be explicit
        print_error "/var/log/armorclaw/email does not exist"
    fi
}

#=============================================================================
# Summary
#=============================================================================

print_summary() {
    echo ""
    echo -e "${BLUE}───────────────────────────────────────${NC}"
    echo -e "Results: ${GREEN}${PASSED} passed${NC}, ${RED}${FAILED} failed${NC}, ${CYAN}${SKIPPED} skipped${NC}, ${YELLOW}${WARNINGS} warning${NC}"

    echo ""
    if [[ $FAILED -gt 0 ]]; then
        echo -e "${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║${NC}  ${BOLD}✗ ${FAILED} check failed — review output above${NC}                    ${RED}║${NC}"
        echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}"
        exit 1
    else
        echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║${NC}  ${BOLD}✓ All checks passed${NC}                                         ${GREEN}║${NC}"
        echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
        exit 0
    fi
}

#=============================================================================
# Main
#=============================================================================

main() {
    print_header
    check_binary
    check_directories
    check_socket
    check_postfix
    check_transport
    check_yara
    check_end_to_end
    check_log_writable
    print_summary
}

main "$@"
