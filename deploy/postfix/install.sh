#!/usr/bin/env bash
# ArmorClaw Postfix Email Setup
# Installs and configures Postfix for the email ingestion pipeline
#
# Usage: sudo ./deploy/postfix/install.sh

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

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Postfix Email Setup${NC}                      ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}Email Ingestion Pipeline${NC}                         ${CYAN}║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "\n${BLUE}▶${NC} ${BOLD}$1${NC}"
    echo -e "${BLUE}  ─────────────────────────────────────${NC}"
}

print_success() {
    echo -e "  ${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "  ${RED}✗${NC} ${BOLD}ERROR:${NC} $1" >&2
}

print_warning() {
    echo -e "  ${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "  ${CYAN}ℹ${NC} $1"
}

fail() {
    print_error "$1"
    exit 1
}

#=============================================================================
# 1. Prerequisites Check
#=============================================================================

check_prerequisites() {
    print_step "Checking prerequisites..."

    # Check root
    if [[ $EUID -ne 0 ]]; then
        fail "This script must be run as root (use sudo)"
    fi
    print_success "Running as root"

    # Check postfix
    if ! command -v postfix &>/dev/null; then
        fail "Postfix is not installed. Install it first: apt install postfix"
    fi
    print_success "Postfix found: $(postconf -h mail_version 2>/dev/null || echo 'installed')"

    # Check Go compiler
    if ! command -v go &>/dev/null; then
        fail "Go compiler not found. Install Go 1.21+ to build mta-recv"
    fi
    print_success "Go compiler found: $(go version | awk '{print $3}')"

    # Check source config files
    local missing=false
    for f in main.cf master.cf transport_maps; do
        if [[ ! -f "$SCRIPT_DIR/$f" ]]; then
            print_error "Missing config file: $SCRIPT_DIR/$f"
            missing=true
        fi
    done
    if $missing; then
        fail "Required config files not found in $SCRIPT_DIR/"
    fi
    print_success "Config files present (main.cf, master.cf, transport_maps)"

    # Check mta-recv source
    if [[ ! -d "$PROJECT_ROOT/bridge/cmd/mta-recv" ]]; then
        fail "mta-recv source not found at $PROJECT_ROOT/bridge/cmd/mta-recv/"
    fi
    print_success "mta-recv source found"
}

#=============================================================================
# 2. Create System User
#=============================================================================

create_user() {
    print_step "Creating system user..."

    if id armorclaw &>/dev/null; then
        print_info "User 'armorclaw' already exists"
    else
        useradd --system --no-create-home --shell /usr/sbin/nologin armorclaw
        print_success "User 'armorclaw' created"
    fi
}

#=============================================================================
# 3. Create Directories
#=============================================================================

create_directories() {
    print_step "Creating directories..."

    mkdir -p /run/armorclaw
    mkdir -p /var/lib/armorclaw/email-files
    mkdir -p /var/log/armorclaw/email
    mkdir -p /etc/armorclaw/certs

    chown armorclaw:armorclaw /run/armorclaw
    chown armorclaw:armorclaw /var/lib/armorclaw/email-files
    chown armorclaw:armorclaw /var/log/armorclaw/email

    print_success "Directory structure created"
    print_info "/run/armorclaw           - runtime socket"
    print_info "/var/lib/armorclaw/email-files - email storage"
    print_info "/var/log/armorclaw/email - email logs"
    print_info "/etc/armorclaw/certs     - TLS certificates"
}

#=============================================================================
# 4. Socket Permission Setup
#=============================================================================

setup_socket_permissions() {
    print_step "Setting up socket permissions..."

    # Create shared mail group
    if getent group armorclaw-mail &>/dev/null; then
        print_info "Group 'armorclaw-mail' already exists"
    else
        groupadd armorclaw-mail
        print_success "Group 'armorclaw-mail' created"
    fi

    # Add armorclaw user to mail group
    usermod -aG armorclaw-mail armorclaw
    print_success "armorclaw added to armorclaw-mail group"

    # Add postfix to mail group for socket access (non-fatal)
    if id postfix &>/dev/null; then
        usermod -aG armorclaw-mail postfix
        print_success "postfix added to armorclaw-mail group"
    else
        print_warning "postfix user not found — skipping group add"
    fi

    # Set socket directory permissions
    chmod 0775 /run/armorclaw
    chown armorclaw:armorclaw-mail /run/armorclaw
    print_success "Socket directory permissions set (0775, armorclaw:armorclaw-mail)"
}

#=============================================================================
# 5. Backup Existing Postfix Config
#=============================================================================

backup_config() {
    print_step "Backing up existing Postfix config..."

    if [[ -f /etc/postfix/main.cf ]]; then
        local backup="/etc/postfix/main.cf.backup.$(date +%Y%m%d%H%M%S)"
        cp /etc/postfix/main.cf "$backup"
        print_success "Backed up main.cf → $(basename "$backup")"
    else
        print_info "No existing main.cf to back up"
    fi

    if [[ -f /etc/postfix/master.cf ]]; then
        local backup="/etc/postfix/master.cf.backup.$(date +%Y%m%d%H%M%S)"
        cp /etc/postfix/master.cf "$backup"
        print_success "Backed up master.cf → $(basename "$backup")"
    else
        print_info "No existing master.cf to back up"
    fi
}

#=============================================================================
# 6. Copy Postfix Config Files
#=============================================================================

copy_configs() {
    print_step "Installing Postfix config files..."

    cp "$SCRIPT_DIR/main.cf" /etc/postfix/main.cf
    print_success "main.cf installed"

    cp "$SCRIPT_DIR/master.cf" /etc/postfix/master.cf
    print_success "master.cf installed"

    cp "$SCRIPT_DIR/transport_maps" /etc/postfix/transport_maps
    print_success "transport_maps installed"
}

#=============================================================================
# 7. Generate transport_maps.db
#=============================================================================

generate_transport_db() {
    print_step "Generating transport_maps.db..."

    postmap /etc/postfix/transport_maps
    print_success "transport_maps.db generated"
}

#=============================================================================
# 8. Build mta-recv Binary
#=============================================================================

build_mta_recv() {
    print_step "Building mta-recv binary..."

    print_info "Source: $PROJECT_ROOT/bridge/cmd/mta-recv/"
    print_info "Output: /usr/local/bin/armorclaw-mta-recv"

    go build -o /usr/local/bin/armorclaw-mta-recv "$PROJECT_ROOT/bridge/cmd/mta-recv/"
    chmod 755 /usr/local/bin/armorclaw-mta-recv

    print_success "armorclaw-mta-recv built and installed"
}

#=============================================================================
# 9. Reload Postfix
#=============================================================================

reload_postfix() {
    print_step "Reloading Postfix..."

    if postfix status &>/dev/null; then
        if postfix reload; then
            print_success "Postfix reloaded"
        else
            print_warning "Postfix reload failed — check config with: postfix check"
        fi
    else
        print_warning "Postfix is not running"
        if postfix start; then
            print_success "Postfix started"
        else
            print_warning "Postfix start failed — start manually after verifying config"
        fi
    fi
}

#=============================================================================
# Summary
#=============================================================================

print_summary() {
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}                 ${BOLD}Postfix Email Setup Complete!${NC}                    ${GREEN}║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${BOLD}Installed Components:${NC}"
    echo ""
    echo -e "  ${GREEN}✓${NC} System user: armorclaw"
    echo -e "  ${GREEN}✓${NC} Socket group: armorclaw-mail"
    echo -e "  ${GREEN}✓${NC} Binary: /usr/local/bin/armorclaw-mta-recv"
    echo -e "  ${GREEN}✓${NC} Config: /etc/postfix/{main,master,transport_maps}.cf"
    echo -e "  ${GREEN}✓${NC} Socket dir: /run/armorclaw (0775)"
    echo ""

    echo -e "${BOLD}Next Steps:${NC}"
    echo ""
    echo -e "  1. Place TLS certs at ${CYAN}/etc/armorclaw/certs/server.{crt,key}${NC}"
    echo -e "  2. Start the ArmorClaw bridge (creates ${CYAN}/run/armorclaw/email-ingest.sock${NC})"
    echo -e "  3. Run verification: ${CYAN}sudo ./deploy/postfix/verify-setup.sh${NC}"
    echo -e "  4. Test with: ${CYAN}echo 'Test' | sendmail user@localhost${NC}"
    echo ""

    echo -e "${BOLD}Important Notes:${NC}"
    echo ""
    echo -e "  • Set ${CYAN}\$ARMORCLAW_HOSTNAME${NC} and ${CYAN}\$ARMORCLAW_DOMAIN${NC} before starting bridge"
    echo -e "  • Socket ${CYAN}/run/armorclaw/email-ingest.sock${NC} is created by IngestServer at runtime"
    echo -e "  • mta-recv accepts positional args: ${CYAN}<sender> <recipient> [queue_id]${NC}"
    echo ""
}

#=============================================================================
# Main
#=============================================================================

main() {
    print_header
    check_prerequisites
    create_user
    create_directories
    setup_socket_permissions
    backup_config
    copy_configs
    generate_transport_db
    build_mta_recv
    reload_postfix
    print_summary
}

main "$@"
