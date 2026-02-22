#!/bin/bash
# ArmorClaw Quick Setup - Express Installation
# Streamlined 3-minute setup with secure defaults
# Version: 1.0.0
#
# Usage: sudo ./deploy/setup-quick.sh [--non-interactive]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Global variables
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
RUN_DIR="/run/armorclaw"
SOCKET_PATH="$RUN_DIR/bridge.sock"
LOG_FILE="/var/log/armorclaw-setup.log"
INSTALL_DIR="/opt/armorclaw"

# Non-interactive mode
NON_INTERACTIVE=false
if [[ "$1" == "--non-interactive" || "$1" == "-y" ]]; then
    NON_INTERACTIVE=true
fi

# Smart defaults for quick setup
DEFAULT_LOG_LEVEL="info"
DEFAULT_LOG_FORMAT="text"
DEFAULT_DAILY_BUDGET="5.00"
DEFAULT_MONTHLY_BUDGET="100.00"
DEFAULT_HARD_STOP="true"
DEFAULT_MATRIX_ENABLED="false"

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    clear 2>/dev/null || true
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Quick Setup${NC}                           ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}Express Installation (2-3 min)${NC}                   ${CYAN}║${NC}"
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

print_done() {
    echo -e "  ${GREEN}✓${NC} $1"
}

fail() {
    print_error "$1"
    exit 1
}

prompt_yes_no() {
    if $NON_INTERACTIVE; then
        return 0  # Default to yes in non-interactive mode
    fi

    local prompt="$1"
    local default="${2:-n}"

    echo ""
    echo -ne "  ${CYAN}$prompt [${default^^}/$(echo $default | tr 'yn' 'ny')]${NC}: "
    read -r response
    response=${response:-$default}

    case "$response" in
        [Yy]|[Yy][Ee][Ss]) return 0 ;;
        [Nn]|[Nn][Oo]) return 1 ;;
    esac

    # Default behavior
    [[ "$default" == "y" ]]
}

#=============================================================================
# Logging
#=============================================================================

init_logging() {
    mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
    exec > >(tee -a "$LOG_FILE")
    exec 2>&1
}

#=============================================================================
# Step 1: Prerequisites Check (Auto)
#=============================================================================

check_prerequisites() {
    print_step "Checking prerequisites..."

    local errors=0

    # Check for sudo/root
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
    print_done "Running as root"

    # Check OS
    if [[ -f /etc/os-release ]]; then
        source /etc/os-release
        print_done "OS: $PRETTY_NAME"
    else
        print_error "Cannot detect OS"
        ((errors++))
    fi

    # Check Docker
    if command -v docker &>/dev/null; then
        if docker info &>/dev/null; then
            print_done "Docker: $(docker --version | awk '{print $3}' | tr -d ',')"
        else
            print_error "Docker is installed but not running"
            print_info "Start with: systemctl start docker"
            ((errors++))
        fi
    else
        print_error "Docker not installed"
        print_info "Install with: curl -fsSL https://get.docker.com | sh"
        ((errors++))
    fi

    # Check memory (minimum 1GB)
    local total_mem=$(free -m | awk '/^Mem:/{print $2}')
    if [[ $total_mem -ge 1024 ]]; then
        print_done "Memory: ${total_mem}MB"
    else
        print_error "Memory: ${total_mem}MB (minimum 1GB required)"
        ((errors++))
    fi

    # Check disk space (minimum 2GB)
    local avail_space=$(df -m / | awk 'NR==2 {print $4}')
    if [[ $avail_space -ge 2048 ]]; then
        print_done "Disk: ${avail_space}MB available"
    else
        print_warning "Disk: ${avail_space}MB available (2GB+ recommended)"
    fi

    if [[ $errors -gt 0 ]]; then
        fail "Prerequisites check failed. Fix the issues above and run again."
    fi

    print_success "All prerequisites met!"
}

#=============================================================================
# Step 2: Build/Install Bridge
#=============================================================================

install_bridge() {
    print_step "Installing ArmorClaw Bridge..."

    # Check if already installed
    if [[ -x "$INSTALL_DIR/armorclaw-bridge" ]]; then
        print_done "Bridge already installed at $INSTALL_DIR/armorclaw-bridge"

        if ! $NON_INTERACTIVE && prompt_yes_no "Reinstall bridge?" "n"; then
            rm -f "$INSTALL_DIR/armorclaw-bridge"
        else
            return 0
        fi
    fi

    # Check for Go
    if ! command -v go &>/dev/null; then
        print_info "Installing Go..."
        apt-get update -qq
        apt-get install -y -qq golang-go
    fi

    local go_version=$(go version | awk '{print $3}')
    print_done "Go: $go_version"

    # Build bridge
    print_info "Building bridge from source..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local project_root="$(dirname "$script_dir")"

    cd "$project_root/bridge"
    if go build -o armorclaw-bridge ./cmd/bridge; then
        print_done "Bridge built successfully"
    else
        fail "Bridge build failed"
    fi

    # Install to system location
    mkdir -p "$INSTALL_DIR"
    mv armorclaw-bridge "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/armorclaw-bridge"
    ln -sf "$INSTALL_DIR/armorclaw-bridge" /usr/local/bin/armorclaw-bridge

    print_success "Bridge installed to $INSTALL_DIR/armorclaw-bridge"
    cd "$project_root"
}

#=============================================================================
# Step 3: Create System User
#=============================================================================

create_user() {
    print_step "Creating system user..."

    if id "armorclaw" &>/dev/null; then
        print_done "User 'armorclaw' already exists"
    else
        useradd -r -s /bin/false -d "$DATA_DIR" armorclaw
        print_done "User 'armorclaw' created"
    fi
}

#=============================================================================
# Step 4: Generate Configuration (Smart Defaults)
#=============================================================================

generate_config() {
    print_step "Generating configuration..."

    # Create directories
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$RUN_DIR"
    chown armorclaw:armorclaw "$RUN_DIR" "$DATA_DIR" 2>/dev/null || true
    chmod 770 "$RUN_DIR"

    local config_file="$CONFIG_DIR/config.toml"

    # Generate provisioning secret for QR codes
    local provisioning_secret=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p -c 32)

    # Create config with smart defaults
    cat > "$config_file" <<EOF
# ArmorClaw Bridge Configuration
# Generated by quick setup on $(date)
#
# Quick setup uses secure defaults. Customize later in this file.
# See: docs/guides/configuration.md for all options.

[server]
socket_path = "$SOCKET_PATH"
daemonize = false

[keystore]
db_path = "$DATA_DIR/keystore.db"

# Matrix disabled by default - enable with: ./deploy/setup-matrix.sh
[matrix]
enabled = false
homeserver_url = ""
username = ""
password = ""
device_id = "armorclaw-bridge"

# Budget tracking with sensible defaults
[budget]
daily_limit_usd = $DEFAULT_DAILY_BUDGET
monthly_limit_usd = $DEFAULT_MONTHLY_BUDGET
alert_threshold = 80.0
hard_stop = true

# Logging configuration
[logging]
level = "$DEFAULT_LOG_LEVEL"
format = "$DEFAULT_LOG_FORMAT"
output = "stdout"

# Provisioning for secure device setup (QR codes)
[provisioning]
signing_secret = "$provisioning_secret"
default_expiry_seconds = 60
max_expiry_seconds = 300
one_time_use = true

# Voice disabled by default
[voice]
enabled = false

# WebRTC signaling disabled by default
[webrtc]
signaling_enabled = false

# Notifications disabled by default
[notifications]
enabled = false

# Event bus disabled by default
[eventbus]
websocket_enabled = false

# Discovery enabled for local network
[discovery]
enabled = true
port = 8080
tls = false
EOF

    chown armorclaw:armorclaw "$config_file"
    chmod 640 "$config_file"

    print_success "Configuration generated: $config_file"
    print_info "Provisioning secret generated for secure QR code setup"
}

#=============================================================================
# Step 5: Initialize Keystore
#=============================================================================

init_keystore() {
    print_step "Initializing keystore..."

    local keystore_db="$DATA_DIR/keystore.db"

    if [[ -f "$keystore_db" ]]; then
        print_done "Keystore already exists"
        return 0
    fi

    # Initialize by running bridge with --init flag
    if sudo -u armorclaw "$INSTALL_DIR/armorclaw-bridge" --init 2>/dev/null; then
        print_done "Keystore initialized"
    else
        # Fallback: keystore will be created on first start
        print_info "Keystore will be created on first bridge start"
    fi
}

#=============================================================================
# Step 6: Setup Systemd Service
#=============================================================================

setup_systemd() {
    print_step "Setting up systemd service..."

    local service_file="/etc/systemd/system/armorclaw-bridge.service"

    cat > "$service_file" <<EOF
[Unit]
Description=ArmorClaw Bridge Service
After=network.target docker.service
Wants=docker.service

[Service]
Type=notify
NotifyAccess=all
User=armorclaw
Group=armorclaw
ExecStart=$INSTALL_DIR/armorclaw-bridge -config $CONFIG_DIR/config.toml
Restart=on-failure
RestartSec=10s

# Resource limits
MemoryMax=512M
CPUQuota=50%

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$CONFIG_DIR $DATA_DIR $RUN_DIR

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    print_done "Systemd service configured"
}

#=============================================================================
# Step 7: Start Bridge
#=============================================================================

start_bridge() {
    print_step "Starting bridge..."

    # Start service
    if systemctl start armorclaw-bridge; then
        print_done "Bridge service started"
    else
        fail "Failed to start bridge service"
    fi

    # Wait for socket
    print_info "Waiting for bridge to be ready..."
    local wait_count=0
    while [[ ! -S "$SOCKET_PATH" ]] && [[ $wait_count -lt 30 ]]; do
        sleep 0.5
        ((wait_count++))
    done

    if [[ -S "$SOCKET_PATH" ]]; then
        print_done "Bridge socket ready: $SOCKET_PATH"
    else
        print_warning "Bridge socket not ready after 15s"
        print_info "Check logs: journalctl -u armorclaw-bridge -n 50"
    fi
}

#=============================================================================
# Step 8: Verify Health
#=============================================================================

verify_health() {
    print_step "Verifying installation..."

    local all_ok=true

    # Check directories
    [[ -d "$CONFIG_DIR" ]] && print_done "Config directory" || { print_error "Config directory missing"; all_ok=false; }
    [[ -d "$DATA_DIR" ]] && print_done "Data directory" || { print_error "Data directory missing"; all_ok=false; }
    [[ -d "$RUN_DIR" ]] && print_done "Run directory" || { print_error "Run directory missing"; all_ok=false; }

    # Check binary
    [[ -x "$INSTALL_DIR/armorclaw-bridge" ]] && print_done "Bridge binary" || { print_error "Bridge binary missing"; all_ok=false; }

    # Check config
    [[ -f "$CONFIG_DIR/config.toml" ]] && print_done "Configuration file" || { print_error "Configuration file missing"; all_ok=false; }

    # Check service
    systemctl is-active armorclaw-bridge &>/dev/null && print_done "Service running" || { print_warning "Service not running"; }

    # Check socket
    [[ -S "$SOCKET_PATH" ]] && print_done "Bridge socket" || { print_warning "Bridge socket not available"; }

    if $all_ok; then
        print_success "Installation verified!"
    else
        print_warning "Some checks failed - review above"
    fi
}

#=============================================================================
# Step 9: Optional API Key
#=============================================================================

prompt_api_key() {
    if $NON_INTERACTIVE; then
        print_info "Skipping API key setup (non-interactive mode)"
        print_info "Add keys later with: armorclaw-bridge add-key"
        return 0
    fi

    print_step "API Key Setup (Optional)"

    echo ""
    echo "  Would you like to add an API key now?"
    echo "  You can skip this and add keys later."
    echo ""

    if ! prompt_yes_no "Add an API key now?" "n"; then
        print_info "You can add API keys later using:"
        echo "    sudo armorclaw-bridge add-key --provider <provider> --token <token>"
        return 0
    fi

    echo ""
    echo -ne "  ${CYAN}Provider (openai/anthropic/openrouter/google/xai):${NC} "
    read -r provider

    if [[ -z "$provider" ]]; then
        print_info "No provider specified, skipping API key setup"
        return 0
    fi

    echo -ne "  ${CYAN}API Key:${NC} "
    read -s key_token
    echo ""

    if [[ -z "$key_token" ]]; then
        print_warning "No key provided, skipping"
        return 0
    fi

    # Add key via RPC (if bridge is running)
    if [[ -S "$SOCKET_PATH" ]]; then
        local key_id="${provider}-main"
        local rpc_cmd='{"jsonrpc":"2.0","method":"add_key","params":{"id":"'"$key_id"'","provider":"'"$provider"'","token":"'"$key_token"'","display_name":"'"$provider"' API Key"},"id":1}'

        if echo "$rpc_cmd" | socat - UNIX-CONNECT:"$SOCKET_PATH" &>/dev/null; then
            print_done "API key added: $key_id"
        else
            print_warning "Could not add key via RPC - add manually"
            print_info "Command: echo '$rpc_cmd' | socat - UNIX-CONNECT:$SOCKET_PATH"
        fi
    else
        print_warning "Bridge not ready - add key manually after setup"
    fi
}

#=============================================================================
# Step 10: Generate QR Code for Provisioning
#=============================================================================

generate_qr_code() {
    print_step "Device Provisioning"

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local provision_script="$script_dir/armorclaw-provision.sh"

    if [[ -x "$provision_script" ]]; then
        # Use dedicated provisioning script
        "$provision_script" --expiry 300
    else
        # Fallback inline generation
        print_info "Generating provisioning QR code..."

        local hostname=$(hostname)
        local ip_address=$(hostname -I | awk '{print $1}')
        local provisioning_secret=$(grep 'signing_secret' "$CONFIG_DIR/config.toml" 2>/dev/null | sed 's/.*= *"\([^"]*\)".*/\1' || echo "")

        if [[ -z "$provisioning_secret" ]]; then
            print_warning "Could not read provisioning secret"
            return 0
        fi

        # Generate JWT-like token (simplified)
        local expiry=$(($(date +%s) + 300))  # 5 minutes
        local token_data="${hostname}:${provisioning_secret}:${expiry}"
        local token=$(echo -n "$token_data" | sha256sum | awk '{print $1}')

        # Create provisioning URL
        local provision_url="armorclaw://provision?host=${ip_address}&port=8080&token=${token}&expires=${expiry}"

        echo ""
        echo -e "  ${BOLD}Scan this QR code with ArmorChat to connect:${NC}"
        echo ""

        # Try to generate QR code with qrencode
        if command -v qrencode &>/dev/null; then
            qrencode -t UTF8 "$provision_url" 2>/dev/null || \
            qrencode -t ASCII "$provision_url" 2>/dev/null
        else
            # ASCII fallback - show URL
            echo -e "  ${CYAN}──────────────────────────────────────${NC}"
            echo -e "  ${CYAN}│${NC} Install qrencode for QR display:   ${CYAN}│${NC}"
            echo -e "  ${CYAN}│${NC}   sudo apt install qrencode         ${CYAN}│${NC}"
            echo -e "  ${CYAN}──────────────────────────────────────${NC}"
            echo ""
            echo -e "  ${BOLD}Manual connection:${NC}"
            echo -e "  ${CYAN}$provision_url${NC}"
        fi

        echo ""
        echo -e "  ${YELLOW}Note:${NC} QR code expires in 5 minutes"
    fi

    echo ""
    print_info "Generate new codes anytime with: sudo ./deploy/armorclaw-provision.sh"
}

#=============================================================================
# Completion Message
#=============================================================================

print_completion() {
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}                 ${BOLD}Setup Complete!${NC}                              ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}             ${BOLD}ArmorClaw is ready to use.${NC}                       ${GREEN}║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${BOLD}Quick Start:${NC}"
    echo ""
    echo "  1. ${CYAN}Connect ArmorChat${NC} (scan QR code above)"
    echo "  2. ${CYAN}Add API key${NC}:"
    echo "     sudo armorclaw-bridge add-key --provider openai --token sk-..."
    echo "  3. ${CYAN}Start an agent${NC}:"
    echo "     sudo armorclaw-bridge start --key openai-main"
    echo ""

    echo -e "${BOLD}Service Commands:${NC}"
    echo ""
    echo "  Status:  ${CYAN}sudo systemctl status armorclaw-bridge${NC}"
    echo "  Logs:    ${CYAN}sudo journalctl -u armorclaw-bridge -f${NC}"
    echo "  Restart: ${CYAN}sudo systemctl restart armorclaw-bridge${NC}"
    echo ""

    echo -e "${BOLD}Next Steps:${NC}"
    echo ""
    echo "  ${CYAN}• Enable Matrix:${NC}       ./deploy/setup-matrix.sh"
    echo "  ${CYAN}• Harden security:${NC}     ./deploy/armorclaw-harden.sh"
    echo "  ${CYAN}• New device QR:${NC}       ./deploy/armorclaw-provision.sh"
    echo "  ${CYAN}• Full configuration:${NC}  nano $CONFIG_DIR/config.toml"
    echo ""

    echo -e "${BOLD}Documentation:${NC}"
    echo ""
    echo "  Quick Start:    docs/guides/setup-guide.md"
    echo "  Configuration:  docs/guides/configuration.md"
    echo "  Full Docs:      docs/index.md"
    echo ""

    if [[ -f "$LOG_FILE" ]]; then
        echo -e "${BOLD}Setup log:${NC} $LOG_FILE"
    fi

    echo ""
}

#=============================================================================
# Main Flow
#=============================================================================

main() {
    # Initialize logging
    init_logging

    # Print header
    print_header

    # Run setup steps
    check_prerequisites
    install_bridge
    create_user
    generate_config
    init_keystore
    setup_systemd
    start_bridge
    verify_health
    prompt_api_key
    generate_qr_code

    # Print completion
    print_completion
}

# Run main function
main "$@"
