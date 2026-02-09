#!/bin/bash
# =============================================================================
# ArmorClaw Deployment Verification Script
# Purpose: Verify that the bridge installation is working correctly
# Usage: sudo ./verify-bridge.sh
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BRIDGE_USER="armorclaw"
INSTALL_DIR="/opt/armorclaw"
CONFIG_DIR="/etc/armorclaw"
RUN_DIR="/run/armorclaw"
BRIDGE_BIN="$INSTALL_DIR/armorclaw-bridge"

# Test counters
PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASS_COUNT++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAIL_COUNT++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARN_COUNT++))
}

check_test() {
    local description="$1"
    local command="$2"

    echo -n "  $description ... "

    if eval "$command" &>/dev/null; then
        echo -e "${GREEN}PASS${NC}"
        ((PASS_COUNT++))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        ((FAIL_COUNT++))
        return 1
    fi
}

# =============================================================================
# Verification Checks
# =============================================================================

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_fail "This script must be run as root"
        exit 1
    fi
}

check_user() {
    echo ""
    echo "=== User and Group Checks ==="

    if id "$BRIDGE_USER" &>/dev/null; then
        log_success "User '$BRIDGE_USER' exists"

        # Check user properties
        local uid=$(id -u "$BRIDGE_USER")
        local shell=$(getent passwd "$BRIDGE_USER" | cut -d: -f7)

        if [[ "$uid" -ge 1000 ]]; then
            log_success "User UID is system user ($uid)"
        else
            log_warn "User UID is $uid (expected >= 1000)"
        fi

        if [[ "$shell" == "/bin/false" ]] || [[ "$shell" == "/usr/sbin/nologin" ]]; then
            log_success "User has no shell login ($shell)"
        else
            log_warn "User has shell $shell (expected /bin/false)"
        fi
    else
        log_fail "User '$BRIDGE_USER' does not exist"
    fi
}

check_directories() {
    echo ""
    echo "=== Directory Checks ==="

    local dirs=("$INSTALL_DIR" "$CONFIG_DIR" "$RUN_DIR")
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            log_success "Directory exists: $dir"

            # Check permissions
            local perms=$(stat -c %a "$dir")
            local owner=$(stat -c %U "$dir")

            case "$dir" in
                "$INSTALL_DIR")
                    if [[ "$owner" == "$BRIDGE_USER" ]]; then
                        log_success "Owner correct: $owner"
                    else
                        log_fail "Owner is $owner (expected $BRIDGE_USER)"
                    fi
                    ;;
                "$CONFIG_DIR")
                    if [[ "$perms" == "755" ]]; then
                        log_success "Permissions correct: $perms"
                    else
                        log_warn "Permissions are $perms (expected 755)"
                    fi
                    ;;
                "$RUN_DIR")
                    if [[ "$perms" == "770" ]]; then
                        log_success "Permissions correct: $perms"
                    else
                        log_warn "Permissions are $perms (expected 770)"
                    fi
                    ;;
            esac
        else
            log_fail "Directory missing: $dir"
        fi
    done
}

check_binary() {
    echo ""
    echo "=== Binary Checks ==="

    if [[ -f "$BRIDGE_BIN" ]]; then
        log_success "Binary exists: $BRIDGE_BIN"

        # Check if executable
        if [[ -x "$BRIDGE_BIN" ]]; then
            log_success "Binary is executable"
        else
            log_fail "Binary is not executable"
        fi

        # Check owner
        local owner=$(stat -c %U "$BRIDGE_BIN")
        if [[ "$owner" == "$BRIDGE_USER" ]]; then
            log_success "Binary owner correct: $owner"
        else
            log_fail "Binary owner is $owner (expected $BRIDGE_USER)"
        fi

        # Check symlink
        if [[ -L "/usr/local/bin/armorclaw-bridge" ]]; then
            local target=$(readlink -f "/usr/local/bin/armorclaw-bridge")
            if [[ "$target" == "$BRIDGE_BIN" ]]; then
                log_success "Symlink correct: /usr/local/bin/armorclaw-bridge -> $BRIDGE_BIN"
            else
                log_warn "Symlink points to $target"
            fi
        else
            log_warn "Symlink missing: /usr/local/bin/armorclaw-bridge"
        fi

        # Try to get version
        local version=$("$BRIDGE_BIN" --version 2>&1 || echo "unknown")
        log_info "Version: $version"

    else
        log_fail "Binary not found: $BRIDGE_BIN"
    fi
}

check_config() {
    echo ""
    echo "=== Configuration Checks ==="

    local config="$CONFIG_DIR/config.toml"

    if [[ -f "$config" ]]; then
        log_success "Config file exists: $config"

        # Check owner
        local owner=$(stat -c %U "$config")
        if [[ "$owner" == "$BRIDGE_USER" ]]; then
            log_success "Config owner correct: $owner"
        else
            log_fail "Config owner is $owner (expected $BRIDGE_USER)"
        fi

        # Check permissions (should be 640)
        local perms=$(stat -c %a "$config")
        if [[ "$perms" == "640" ]]; then
            log_success "Config permissions correct: $perms"
        else
            log_warn "Config permissions are $perms (expected 640)"
        fi

        # Validate TOML syntax
        if command -v python3 &>/dev/null; then
            if python3 -c "import tomllib; tomllib.load(open('$config', 'rb'))" 2>/dev/null; then
                log_success "Config TOML syntax is valid"
            else
                log_fail "Config TOML syntax is invalid"
            fi
        fi

    else
        log_fail "Config file not found: $config"
    fi
}

check_docker() {
    echo ""
    echo "=== Docker Checks ==="

    if command -v docker &>/dev/null; then
        log_success "Docker is installed"

        # Check if Docker daemon is running
        if docker info &>/dev/null; then
            log_success "Docker daemon is running"

            # Check if user can access Docker
            if sudo -u "$BRIDGE_USER" docker info &>/dev/null; then
                log_success "Bridge user can access Docker"
            else
                log_warn "Bridge user cannot access Docker (may need group membership)"
            fi

            # Check for ArmorClaw container image
            if docker images | grep -q "armorclaw/agent"; then
                log_success "Container image found: armorclaw/agent"
                docker images armorclaw/agent --format "  - Tag: {{.Tag}}, Size: {{.Size}}"
            else
                log_warn "Container image not found: armorclaw/agent"
            fi
        else
            log_fail "Docker daemon is not running"
        fi
    else
        log_warn "Docker is not installed"
    fi
}

check_systemd() {
    echo ""
    echo "=== Systemd Service Checks ==="

    local service="/etc/systemd/system/armorclaw-bridge.service"

    if [[ -f "$service" ]]; then
        log_success "Service file exists: $service"

        # Check if service is enabled
        if systemctl is-enabled armorclaw-bridge &>/dev/null; then
            log_success "Service is enabled"
        else
            log_warn "Service is not enabled"
        fi

        # Check if service is running
        if systemctl is-active armorclaw-bridge &>/dev/null; then
            log_success "Service is running"

            # Show status
            local status=$(systemctl is-active armorclaw-bridge)
            log_info "Service status: $status"
        else
            log_warn "Service is not running"
        fi

        # Check for socket
        if [[ -S "$RUN_DIR/bridge.sock" ]]; then
            log_success "Socket exists: $RUN_DIR/bridge.sock"

            # Check socket permissions
            local perms=$(stat -c %a "$RUN_DIR/bridge.sock")
            log_info "Socket permissions: $perms"
        else
            log_warn "Socket not found: $RUN_DIR/bridge.sock"
        fi

    else
        log_fail "Service file not found: $service"
    fi
}

check_keystore() {
    echo ""
    echo "=== Keystore Checks ==="

    local keystore="$CONFIG_DIR/keystore.db"
    local salt="$CONFIG_DIR/keystore.db.salt"

    if [[ -f "$keystore" ]]; then
        log_success "Keystore database exists"

        # Check owner
        local owner=$(stat -c %U "$keystore")
        if [[ "$owner" == "$BRIDGE_USER" ]]; then
            log_success "Keystore owner correct: $owner"
        else
            log_fail "Keystore owner is $owner (expected $BRIDGE_USER)"
        fi

        # Check for salt
        if [[ -f "$salt" ]]; then
            log_success "Salt file exists (zero-touch reboot enabled)"
        else
            log_warn "Salt file missing (keystore will require re-initialization after reboot)"
        fi
    else
        log_info "Keystore not initialized (will be created on first start)"
    fi
}

print_summary() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                    Verification Summary                        ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    echo -e "  ${GREEN}Passed:${NC}   $PASS_COUNT"
    echo -e "  ${YELLOW}Warnings:${NC} $WARN_COUNT"
    echo -e "  ${RED}Failed:${NC}   $FAIL_COUNT"
    echo ""

    if [[ $FAIL_COUNT -eq 0 ]]; then
        echo -e "${GREEN}✓ All critical checks passed!${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Review warnings above"
        echo "  2. Start the service: sudo systemctl start armorclaw-bridge"
        echo "  3. Check logs: sudo journalctl -u armorclaw-bridge -f"
        exit 0
    else
        echo -e "${RED}✗ Some checks failed. Please review and fix the issues above.${NC}"
        exit 1
    fi
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║           ArmorClaw Bridge Deployment Verification            ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""

    check_root

    # Run all verification checks
    check_user
    check_directories
    check_binary
    check_config
    check_docker
    check_systemd
    check_keystore

    # Print summary
    print_summary
}

main "$@"
