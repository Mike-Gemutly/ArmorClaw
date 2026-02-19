#!/bin/bash
# ArmorClaw Security Verification Script
# Verifies all security measures are in place
# Version: 1.0.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
RUN_DIR="/run/armorclaw"
LOG_DIR="/var/log/armorclaw"

# Counters
PASS=0
FAIL=0
WARN=0

#=============================================================================
# Helper Functions
#=============================================================================

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASS++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAIL++))
}

warn() {
    echo -e "${YELLOW}⚠ WARN${NC}: $1"
    ((WARN++))
}

info() {
    echo -e "${CYAN}ℹ INFO${NC}: $1"
}

section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

#=============================================================================
# Firewall Checks
#=============================================================================

check_firewall() {
    section "Firewall Checks"

    # Check UFW installed
    if command -v ufw &> /dev/null; then
        pass "UFW is installed"
    else
        fail "UFW is not installed"
        return
    fi

    # Check UFW active
    if ufw status | grep -q "Status: active"; then
        pass "UFW is active"
    else
        fail "UFW is not active"
    fi

    # Check default deny
    if ufw status | grep -q "Default: deny"; then
        pass "Default deny policy is set"
    else
        warn "Default deny policy may not be fully set"
    fi

    # Check SSH allowed
    if ufw status | grep -q "22/tcp"; then
        pass "SSH is allowed"
    else
        warn "SSH may not be allowed (remote access at risk)"
    fi

    # Check rate limiting
    if ufw status | grep -q "LIMIT"; then
        pass "SSH rate limiting is enabled"
    else
        warn "SSH rate limiting is not enabled"
    fi
}

#=============================================================================
# Container Security Checks
#=============================================================================

check_container_security() {
    section "Container Security Checks"

    # Check Docker installed
    if command -v docker &> /dev/null; then
        pass "Docker is installed"
    else
        fail "Docker is not installed"
        return
    fi

    # Check Docker running
    if docker info &> /dev/null; then
        pass "Docker daemon is running"
    else
        fail "Docker daemon is not running"
    fi

    # Check seccomp profile
    if [ -f "/etc/docker/seccomp/armorclaw.json" ]; then
        pass "Seccomp profile exists"
    else
        warn "Seccomp profile not found"
    fi

    # Check Docker daemon config
    if [ -f "/etc/docker/daemon.json" ]; then
        pass "Docker daemon.json exists"
        if grep -q "no-new-privileges" /etc/docker/daemon.json; then
            pass "no-new-privileges is set"
        else
            warn "no-new-privileges may not be set"
        fi
    else
        warn "Docker daemon.json not found"
    fi

    # Check for privileged containers
    PRIVILEGED=$(docker ps --format '{{.Names}}' --filter status=running 2>/dev/null | while read container; do
        docker inspect --format '{{.HostConfig.Privileged}}' "$container" 2>/dev/null
    done | grep -c "true" || echo "0")

    if [ "$PRIVILEGED" -eq 0 ]; then
        pass "No privileged containers running"
    else
        fail "$PRIVILEGED privileged container(s) detected"
    fi
}

#=============================================================================
# File Permission Checks
#=============================================================================

check_permissions() {
    section "File Permission Checks"

    # Check config directory
    if [ -d "$CONFIG_DIR" ]; then
        pass "Config directory exists"
        PERMS=$(stat -c '%a' "$CONFIG_DIR" 2>/dev/null || stat -f '%Lp' "$CONFIG_DIR")
        if [ "$PERMS" -le "750" ]; then
            pass "Config directory permissions are secure ($PERMS)"
        else
            fail "Config directory permissions are too permissive ($PERMS)"
        fi
    else
        fail "Config directory does not exist"
    fi

    # Check data directory
    if [ -d "$DATA_DIR" ]; then
        pass "Data directory exists"
    else
        fail "Data directory does not exist"
    fi

    # Check run directory
    if [ -d "$RUN_DIR" ]; then
        pass "Run directory exists"
    else
        fail "Run directory does not exist"
    fi

    # Check keystore
    if [ -f "$DATA_DIR/keystore.db" ]; then
        pass "Keystore database exists"
        PERMS=$(stat -c '%a' "$DATA_DIR/keystore.db" 2>/dev/null || stat -f '%Lp' "$DATA_DIR/keystore.db")
        if [ "$PERMS" = "600" ]; then
            pass "Keystore permissions are correct ($PERMS)"
        else
            fail "Keystore permissions should be 600 (got $PERMS)"
        fi
    else
        warn "Keystore not yet initialized"
    fi

    # Check config file
    if [ -f "$CONFIG_DIR/config.toml" ]; then
        pass "Config file exists"
        PERMS=$(stat -c '%a' "$CONFIG_DIR/config.toml" 2>/dev/null || stat -f '%Lp' "$CONFIG_DIR/config.toml")
        if [ "$PERMS" -le "640" ]; then
            pass "Config file permissions are secure ($PERMS)"
        else
            warn "Config file permissions may be too permissive ($PERMS)"
        fi
    else
        fail "Config file does not exist"
    fi
}

#=============================================================================
# User/Group Checks
#=============================================================================

check_users() {
    section "User and Group Checks"

    # Check armorclaw user
    if id "armorclaw" &>/dev/null; then
        pass "ArmorClaw user exists"
        SHELL=$(getent passwd armorclaw | cut -d: -f7)
        if [ "$SHELL" = "/bin/false" ] || [ "$SHELL" = "/usr/sbin/nologin" ]; then
            pass "ArmorClaw user has no login shell"
        else
            warn "ArmorClaw user has a login shell ($SHELL)"
        fi
    else
        warn "ArmorClaw user does not exist"
    fi

    # Check for passwordless sudo
    if sudo -l -U armorclaw 2>/dev/null | grep -q "NOPASSWD"; then
        warn "ArmorClaw user has passwordless sudo"
    else
        pass "ArmorClaw user does not have passwordless sudo"
    fi
}

#=============================================================================
# Network Security Checks
#=============================================================================

check_network() {
    section "Network Security Checks"

    # Check SYN cookies
    if [ "$(cat /proc/sys/net/ipv4/tcp_syncookies 2>/dev/null)" = "1" ]; then
        pass "TCP SYN cookies enabled"
    else
        warn "TCP SYN cookies not enabled"
    fi

    # Check IP forwarding
    if [ "$(cat /proc/sys/net/ipv4/ip_forward 2>/dev/null)" = "0" ]; then
        pass "IP forwarding disabled"
    else
        warn "IP forwarding is enabled"
    fi

    # Check reverse path filtering
    if [ "$(cat /proc/sys/net/ipv4/conf/all/rp_filter 2>/dev/null)" = "1" ]; then
        pass "Reverse path filtering enabled"
    else
        warn "Reverse path filtering not enabled"
    fi

    # Check for unexpected listening ports
    info "Checking listening ports..."
    LISTENING=$(ss -tuln | grep LISTEN | wc -l)
    info "Found $LISTENING listening ports"

    # Check for exposed Docker socket
    if ss -xl | grep -q "docker.sock"; then
        warn "Docker socket is exposed"
    else
        pass "Docker socket not exposed on network"
    fi
}

#=============================================================================
# Audit Checks
#=============================================================================

check_audit() {
    section "Audit Checks"

    # Check auditd installed
    if command -v auditd &> /dev/null || [ -f "/sbin/auditd" ]; then
        pass "Auditd is installed"
    else
        warn "Auditd is not installed"
        return
    fi

    # Check auditd running
    if service auditd status > /dev/null 2>&1 || systemctl is-active auditd > /dev/null 2>&1; then
        pass "Auditd is running"
    else
        warn "Auditd is not running"
    fi

    # Check ArmorClaw audit rules
    if [ -f "/etc/audit/rules.d/armorclaw.rules" ]; then
        pass "ArmorClaw audit rules exist"
    else
        warn "ArmorClaw audit rules not found"
    fi
}

#=============================================================================
# Bridge Checks
#=============================================================================

check_bridge() {
    section "Bridge Checks"

    # Check bridge binary
    if [ -f "/opt/armorclaw/armorclaw-bridge" ] || [ -f "/usr/local/bin/armorclaw-bridge" ]; then
        pass "Bridge binary exists"
    else
        fail "Bridge binary not found"
        return
    fi

    # Check systemd service
    if [ -f "/etc/systemd/system/armorclaw-bridge.service" ]; then
        pass "Systemd service file exists"
    else
        warn "Systemd service file not found"
    fi

    # Check socket
    if [ -S "$RUN_DIR/bridge.sock" ]; then
        pass "Bridge socket exists"
    else
        info "Bridge socket not found (bridge may not be running)"
    fi

    # Check lockdown state
    if [ -f "$DATA_DIR/lockdown.json" ]; then
        pass "Lockdown state file exists"
        if grep -q '"setup_complete":true' "$DATA_DIR/lockdown.json" 2>/dev/null; then
            pass "Setup is complete"
        else
            info "Setup may not be complete"
        fi
    else
        info "Lockdown state file not found"
    fi
}

#=============================================================================
# Security Configuration Checks
#=============================================================================

check_security_config() {
    section "Security Configuration Checks"

    # Check security config directory
    if [ -d "$CONFIG_DIR/security.d" ]; then
        pass "Security config directory exists"
    else
        info "Security config directory not found"
    fi

    # Check for category configuration
    if [ -f "$CONFIG_DIR/security.d/categories.json" ]; then
        pass "Category configuration exists"
    else
        info "Category configuration not found"
    fi

    # Check for adapter configuration
    if [ -f "$CONFIG_DIR/security.d/adapters.json" ]; then
        pass "Adapter configuration exists"
    else
        info "Adapter configuration not found"
    fi

    # Check Matrix config
    if [ -f "$CONFIG_DIR/config.toml" ]; then
        if grep -q "matrix.*enabled.*=.*true" "$CONFIG_DIR/config.toml" 2>/dev/null; then
            pass "Matrix is enabled"
        else
            info "Matrix is not enabled"
        fi
    fi
}

#=============================================================================
# Summary
#=============================================================================

print_summary() {
    section "Security Verification Summary"

    TOTAL=$((PASS + FAIL + WARN))
    SCORE=$((PASS * 100 / TOTAL))

    echo ""
    echo -e "${BOLD}Results:${NC}"
    echo -e "  ${GREEN}Passed:${NC}   $PASS"
    echo -e "  ${RED}Failed:${NC}   $FAIL"
    echo -e "  ${YELLOW}Warnings:${NC} $WARN"
    echo ""
    echo -e "${BOLD}Security Score:${NC} $SCORE%"
    echo ""

    if [ "$FAIL" -eq 0 ]; then
        if [ "$WARN" -eq 0 ]; then
            echo -e "${GREEN}✓ All security checks passed!${NC}"
        else
            echo -e "${YELLOW}⚠ Security is good but has $WARN warning(s)${NC}"
        fi
    else
        echo -e "${RED}✗ $FAIL security check(s) failed${NC}"
    fi

    # Return exit code
    if [ "$FAIL" -gt 0 ]; then
        return 1
    fi
    return 0
}

#=============================================================================
# Main
#=============================================================================

main() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}        ${BOLD}ArmorClaw Security Verification${NC}             ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}              $(date '+%Y-%m-%d %H:%M:%S')                  ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════╝${NC}"

    check_firewall
    check_container_security
    check_permissions
    check_users
    check_network
    check_audit
    check_bridge
    check_security_config

    print_summary
}

# Run main
main "$@"
