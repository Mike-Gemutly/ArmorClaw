#!/bin/bash
# ArmorClaw Setup Wizard
# Interactive guided installation and configuration
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

# Global variables
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
RUN_DIR="/run/armorclaw"
SOCKET_PATH="$RUN_DIR/bridge.sock"
LOG_FILE="/var/log/armorclaw-setup.log"

# Required commands
REQUIRED_COMMANDS=("docker" "docker compose" "go" "tar" "systemctl")

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    echo -e "${CYAN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}        ${BOLD}ArmorClaw Setup Wizard${NC}                      ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}        ${BOLD}Version 1.0.0${NC}                                  ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    local step="$1"
    local total="$2"
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Step $step of $total: ${BOLD}$3${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
    echo -e "\n${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "\n${RED}✗ ERROR: $1${NC}" >&2
}

print_warning() {
    echo -e "\n${YELLOW}⚠ WARNING: $1${NC}"
}

print_info() {
    echo -e "\n${CYAN}ℹ $1${NC}"
}

prompt_yes_no() {
    local prompt="$1"
    local default="${2:-n}"

    while true; do
        if [ "$default" = "y" ]; then
            echo -ne "${CYAN}$prompt [Y/n]: ${NC}"
        else
            echo -ne "${CYAN}$prompt [y/N]: ${NC}"
        fi

        read -r response
        response=${response:-$default}

        case "$response" in
            [Yy]|[Yy][Ee][Ss]) return 0 ;;
            [Nn]|[Nn][Oo]) return 1 ;;
        esac

        echo -e "${YELLOW}Please answer yes or no.${NC}"
    done
}

prompt_input() {
    local prompt="$1"
    local default="$2"
    local result

    if [ -n "$default" ]; then
        echo -ne "${CYAN}$prompt [$default]: ${NC}"
    else
        echo -ne "${CYAN}$prompt: ${NC}"
    fi

    read -r result
    echo "${result:-$default}"
}

prompt_password() {
    local prompt="$1"
    local password
    local confirm

    while true; do
        read -s -p "$prompt: " password
        echo
        read -s -p "Confirm: " confirm
        echo

        if [ "$password" = "$confirm" ]; then
            if [ ${#password} -ge 8 ]; then
                echo "$password"
                return 0
            else
                print_error "Password must be at least 8 characters"
            fi
        else
            print_error "Passwords do not match"
        fi
    done
}

check_command() {
    local cmd="$1"
    if ! command -v "$cmd" &> /dev/null; then
        return 1
    fi
    return 0
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
# Step 1: Welcome and Overview
#=============================================================================

step_welcome() {
    print_step 1 12 "Welcome and Overview"

    cat <<'EOF'
${BOLD}Welcome to ArmorClaw!${NC}

ArmorClaw is a secure containerization system for AI agents. This wizard will guide you through:

  1. ✓ System requirements check
  2. ✓ Docker installation/verification
  3. ✓ Container image setup
  4. ✓ Bridge installation
  5. ✓ Budget confirmation (FINANCIAL RESPONSIBILITY)
  6. ✓ Configuration file creation
  7. ✓ Keystore initialization
  8. ✓ First API key setup
  9. ✓ Systemd service setup
  10. ✓ Post-installation verification
  11. ✓ Start first agent (optional)

${BOLD}Estimated time:${NC} 10-15 minutes
${BOLD}Configuration directory:${NC} /etc/armorclaw
${BOLD}Data directory:${NC} /var/lib/armorclaw

${YELLOW}Note:${NC} You can cancel at any time by pressing Ctrl+C
EOF

    echo ""
    if ! prompt_yes_no "Continue with setup?"; then
        print_info "Setup cancelled by user"
        exit 0
    fi
}

#=============================================================================
# Step 2: System Requirements Check
#=============================================================================

step_prerequisites() {
    print_step 2 11 "System Requirements Check"

    local all_ok=true

    # Check OS
    print_info "Checking operating system..."
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        print_success "OS: $PRETTY_NAME"

        case "$ID" in
            ubuntu|debian)
                print_success "Supported OS detected"
                ;;
            *)
                print_warning "You are using $PRETTY_NAME. Official support is for Ubuntu 22.04+ and Debian 12+."
                if ! prompt_yes_no "Continue anyway?"; then
                    exit 1
                fi
                ;;
        esac
    else
        print_error "Cannot detect OS. /etc/os-release not found."
        exit 1
    fi

    # Check RAM
    print_info "Checking memory..."
    local total_mem=$(free -m | awk '/^Mem:/{print $2}')
    if [ "$total_mem" -ge 2048 ]; then
        print_success "Memory: ${total_mem}MB (recommended: 2048MB+)"
    elif [ "$total_mem" -ge 1024 ]; then
        print_warning "Memory: ${total_mem}MB (below 2048MB recommendation, but should work)"
    else
        print_error "Memory: ${total_mem}MB (minimum 1024MB required)"
        all_ok=false
    fi

    # Check disk space
    print_info "Checking disk space..."
    local avail_space=$(df -m / | awk 'NR==2 {print $4}')
    if [ "$avail_space" -ge 5120 ]; then
        print_success "Disk space: ${avail_space}MB available"
    else
        print_warning "Disk space: ${avail_space}MB available (5GB+ recommended)"
    fi

    # Check CPU
    print_info "Checking CPU..."
    local cpu_cores=$(nproc)
    if [ "$cpu_cores" -ge 2 ]; then
        print_success "CPU: $cpu_cores cores"
    else
        print_warning "CPU: $cpu_cores core (2+ cores recommended)"
    fi

    # Check for sudo/root
    print_info "Checking permissions..."
    if [ "$EUID" -eq 0 ]; then
        print_success "Running as root"
    elif sudo -n true 2>/dev/null; then
        print_success "Sudo access available"
    else
        print_error "This script requires root or sudo access"
        all_ok=false
    fi

    if [ "$all_ok" = false ]; then
        print_error "Prerequisites check failed. Please fix the issues above and run again."
        exit 1
    fi

    print_success "All prerequisites met!"
}

#=============================================================================
# Step 3: Docker Verification
#=============================================================================

step_docker() {
    print_step 3 11 "Docker Verification"

    if check_command docker; then
        local docker_version=$(docker --version | awk '{print $3}' | sed 's/,//')
        print_success "Docker installed: $docker_version"
    else
        print_info "Docker not found. Would you like to install it now?"
        if prompt_yes_no "Install Docker?" "y"; then
            print_info "Installing Docker..."
            curl -fsSL https://get.docker.com -o get-docker.sh
            sudo sh get-docker.sh
            rm get-docker.sh

            # Add user to docker group
            local user="$SUDO_USER"
            if [ -z "$user" ]; then
                user="$USER"
            fi
            if [ -n "$user" ]; then
                sudo usermod -aG docker "$user"
                print_warning "You will need to log out and log back in for group changes to take effect"
            fi

            print_success "Docker installed"
        else
            print_error "Docker is required for ArmorClaw"
            exit 1
        fi
    fi

    # Check if Docker is running
    print_info "Checking if Docker daemon is running..."
    if sudo docker info &> /dev/null; then
        print_success "Docker daemon is running"
    else
        print_error "Docker daemon is not running"
        print_info "Start it with: sudo systemctl start docker"
        exit 1
    fi

    # Check Docker Compose
    if check_command "docker compose" || check_command docker-compose; then
        print_success "Docker Compose available"
    else
        print_warning "Docker Compose not found (optional, recommended for full stack deployment)"
    fi

    # Test Docker with a simple container
    print_info "Testing Docker with hello-world container..."
    if sudo docker run --rm hello-world &> /dev/null; then
        print_success "Docker is working correctly"
    else
        print_error "Docker test failed"
        exit 1
    fi
}

#=============================================================================
# Step 4: Container Image Setup
#=============================================================================

step_container() {
    print_step 4 11 "Container Image Setup"

    # Check if image exists locally
    print_info "Checking for ArmorClaw agent container image..."
    if sudo docker images | grep -q "armorclaw/agent"; then
        print_success "Container image found locally"
        sudo docker images | grep "armorclaw/agent"
    else
        print_info "Container image not found locally"

        if [ -f "Dockerfile" ]; then
            print_info "Dockerfile found in current directory"
            if prompt_yes_no "Build container image now? (takes ~5 minutes)"; then
                print_info "Building container image..."
                print_info "This may take several minutes on first build..."

                if sudo docker build -t armorclaw/agent:v1 .; then
                    print_success "Container image built successfully"
                else
                    print_error "Container build failed"
                    exit 1
                fi
            fi
        else
            print_info "No Dockerfile found. You can build the image later:"
            echo "  cd armorclaw"
            echo "  docker build -t armorclaw/agent:v1 ."
        fi
    fi

    # Show container info
    if sudo docker images | grep -q "armorclaw/agent"; then
        echo ""
        echo -e "${BOLD}Container Image Details:${NC}"
        sudo docker images armorclaw/agent:v1
        echo ""
    fi
}

#=============================================================================
# Step 5: Bridge Installation
#=============================================================================

step_bridge_install() {
    print_step 5 11 "Bridge Installation"

    # Check if already installed
    if [ -f "/usr/local/bin/armorclaw-bridge" ] || [ -f "/opt/armorclaw/armorclaw-bridge" ]; then
        print_info "Bridge binary found"
        if prompt_yes_no "Reinstall bridge?"; then
            sudo rm -f /usr/local/bin/armorclaw-bridge /opt/armorclaw/armorclaw-bridge
        else
            print_success "Using existing bridge installation"
            return
        fi
    fi

    # Check if we need to build or use pre-built binary
    print_info "Bridge binary installation options:"
    echo "  1. Build from source (requires Go 1.21+)"
    echo "  2. Use pre-built binary (if available)"
    echo ""

    local choice=$(prompt_input "Choose option [1/2]" "1")

    case "$choice" in
        1)
            if check_command go; then
                local go_version=$(go version | awk '{print $3}')
                print_success "Go installed: $go_version"
            else
                print_error "Go is not installed. Please install Go 1.21+ or use option 2."
                exit 1
            fi

            print_info "Building bridge from source..."
            cd bridge
            if go build -o armorclaw-bridge ./cmd/bridge; then
                print_success "Bridge built successfully"

                # Install to system location
                print_info "Installing to /opt/armorclaw/..."
                sudo mkdir -p /opt/armorclaw
                sudo mv armorclaw-bridge /opt/armorclaw/
                sudo chmod +x /opt/armorclaw/armorclaw-bridge
                sudo ln -sf /opt/armorclaw/armorclaw-bridge /usr/local/bin/armorclaw-bridge
                print_success "Bridge installed"
            else
                print_error "Bridge build failed"
                exit 1
            fi
            cd ..
            ;;
        2)
            print_info "Pre-built binary option not yet implemented"
            print_info "Please build from source (option 1)"
            exit 1
            ;;
        *)
            print_error "Invalid option"
            exit 1
            ;;
    esac

    # Create system user
    print_info "Creating system user..."
    if id "armorclaw" &>/dev/null; then
        print_success "User 'armorclaw' already exists"
    else
        sudo useradd -r -s /bin/false -d /var/lib/armorclaw armorclaw
        print_success "User 'armorclaw' created"
    fi
}

#=============================================================================
# Step 6: Budget Confirmation
#=============================================================================

step_budget_confirmation() {
    print_step 6 11 "Budget Confirmation"

    cat <<EOF
${BOLD}CRITICAL: Financial Responsibility${NC}

ArmorClaw provides token budget tracking, but you MUST set hard limits
in your AI provider dashboard to prevent unexpected charges.

${YELLOW}Please verify you have set (or will set) the following:${NC}

EOF

    echo "  1. ${CYAN}OpenAI${NC}: https://platform.openai.com/settings/limits"
    echo "  2. ${CYAN}Anthropic${NC}: https://console.anthropic.com/settings/limits"
    echo ""

    print_info "Set your hard monthly limit to a reasonable amount (e.g., \$100)"
    echo ""

    if ! prompt_yes_no "I have set (or will set) hard limits in my provider dashboard"; then
        print_error "Budget confirmation required"
        print_info "Please set hard limits before continuing"
        exit 1
    fi

    print_success "Budget confirmation complete"
}

#=============================================================================
# Step 7: Configuration File Creation
#=============================================================================

step_configuration() {
    print_step 7 11 "Configuration File Creation"

    print_info "Creating configuration directory..."
    sudo mkdir -p "$CONFIG_DIR"
    sudo mkdir -p "$DATA_DIR"
    sudo mkdir -p "$RUN_DIR"
    sudo chown armorclaw:armorclaw "$RUN_DIR" "$DATA_DIR"
    sudo chmod 770 "$RUN_DIR"

    local config_file="$CONFIG_DIR/config.toml"

    # Get configuration values
    print_info "${BOLD}Configuration Options:${NC}"

    local socket_path=$(prompt_input "Socket path" "$SOCKET_PATH")
    local log_level=$(prompt_input "Log level [debug/info/warn/error]" "info")
    local daemonize=$(prompt_input "Run as daemon [true/false]" "false")

    # Matrix configuration (optional)
    echo ""
    print_info "Matrix Configuration (optional - for remote communication)"
    local matrix_enabled="false"
    local matrix_url=""
    local matrix_user=""
    local matrix_pass=""

    if prompt_yes_no "Enable Matrix communication?"; then
        matrix_enabled="true"
        local matrix_url=$(prompt_input "Matrix homeserver URL" "https://matrix.example.com")
        local matrix_user=$(prompt_input "Matrix bridge username" "bridge")
        local matrix_pass=""
        echo ""
        print_info "Enter Matrix password (leave empty to set later):"
        matrix_pass=$(prompt_password "  Password")

        # Zero-trust configuration
        echo ""
        print_info "${BOLD}Zero-Trust Security Configuration${NC}"
        print_info "ArmorClaw can restrict Matrix communication to trusted senders and rooms."
        print_info "This provides an additional layer of security for remote agent control."
        echo ""

        local enable_trust="false"
        local trusted_senders=""
        local trusted_rooms=""
        local reject_untrusted="false"

        if prompt_yes_no "Enable zero-trust sender/room filtering?"; then
            enable_trust="true"
            echo ""
            print_info "Enter trusted Matrix user IDs (one per line, empty line to finish):"
            print_info "Format: @user:domain.com, *@trusted.domain.com, or *:domain.com for wildcards"
            echo ""
            while IFS= read -r line; do
                [ -z "$line" ] && break
                trusted_senders="$trusted_senders  \"$line\","
            done

            echo ""
            print_info "Enter trusted room IDs (one per line, empty line to finish):"
            print_info "Format: !roomid:domain.com"
            echo ""
            while IFS= read -r line; do
                [ -z "$line" ] && break
                trusted_rooms="$trusted_rooms  \"$line\","
            done

            echo ""
            if prompt_yes_no "Send rejection message to untrusted senders?"; then
                reject_untrusted="true"
            fi
        fi
    fi

    # Budget configuration
    echo ""
    print_info "${BOLD}Budget Guardrails Configuration${NC}"
    print_info "ArmorClaw provides token budget tracking to prevent unexpected API costs."
    echo ""
    print_info "IMPORTANT: You MUST ALSO set hard limits in your AI provider dashboard!"
    print_info "  • OpenAI: https://platform.openai.com/settings/limits"
    print_info "  • Anthropic: https://console.anthropic.com/settings/limits"
    echo ""

    local budget_daily=$(prompt_input "Daily budget limit in USD (0 = no limit)" "5.00")
    local budget_monthly=$(prompt_input "Monthly budget limit in USD (0 = no limit)" "100.00")
    local budget_hardstop="true"

    if ! prompt_yes_no "Enable hard-stop when budget exceeded? (recommended)"; then
        budget_hardstop="false"
    fi

    # Create configuration file
    print_info "Creating configuration file: $config_file"
    sudo tee "$config_file" > /dev/null <<EOF
# ArmorClaw Bridge Configuration
# Generated by setup wizard on $(date)

[server]
socket_path = "$socket_path"
daemonize = $daemonize

[keystore]
db_path = "$DATA_DIR/keystore.db"

[matrix]
enabled = $matrix_enabled
EOF

    if [ "$matrix_enabled" = "true" ]; then
        sudo tee -a "$config_file" > /dev/null <<EOF
homeserver_url = "$matrix_url"
username = "$matrix_user"
password = "$matrix_pass"
EOF
    fi

    # Zero-trust configuration
    if [ "$matrix_enabled" = "true" ] && [ "$enable_trust" = "true" ]; then
        sudo tee -a "$config_file" > /dev/null <<EOF

[matrix.zero_trust]
trusted_senders = [
$trusted_senders
]
trusted_rooms = [
$trusted_rooms
]
reject_untrusted = $reject_untrusted
EOF
    fi

    # Budget configuration
    sudo tee -a "$config_file" > /dev/null <<EOF

[budget]
daily_limit_usd = $budget_daily
monthly_limit_usd = $budget_monthly
alert_threshold = 80.0
hard_stop = $budget_hardstop
EOF

    sudo tee -a "$config_file" > /dev/null <<EOF

[logging]
level = "$log_level"
output = "stdout"
EOF

    # Set permissions
    sudo chown armorclaw:armorclaw "$config_file"
    sudo chmod 640 "$config_file"

    print_success "Configuration file created"
    echo ""
    cat "$config_file"
}

#=============================================================================
# Step 7: Keystore Initialization
#=============================================================================

step_keystore() {
    print_step 8 11 "Keystore Initialization"

    local keystore_db="$DATA_DIR/keystore.db"

    if [ -f "$keystore_db" ]; then
        print_info "Keystore already exists at $keystore_db"
        if prompt_yes_no "Reinitialize keystore? (WARNING: This will delete all stored credentials)"; then
            sudo rm -f "$keystore_db" "$keystore_db.salt"
        else
            print_success "Using existing keystore"
            return
        fi
    fi

    print_info "${BOLD}Keystore Security Features:${NC}"
    cat <<'EOF'
  • Double-layer encryption (SQLCipher + XChaCha20-Poly1305)
  • Hardware-derived master key (machine-id + DMI UUID + MAC)
  • Zero-touch reboot (salt persistence)
  • Theft protection (database unusable on different hardware)
EOF

    echo ""
    print_info "Initializing keystore..."

    # Initialize keystore by running bridge with init command
    if sudo -u armorclaw /opt/armorclaw/armorclaw-bridge --init; then
        print_success "Keystore initialized"
        print_info "Keystore database: $keystore_db"
        print_info "Salt file: $keystore_db.salt"
    else
        print_warning "Init command not available, keystore will be created on first start"
    fi
}

#=============================================================================
# Step 8: First API Key Setup
#=============================================================================

step_api_key() {
    print_step 9 11 "First API Key Setup"

    echo ""
    print_info "${BOLD}You can now add your first API key for AI agents.${NC}"
    echo ""
    print_info "Supported providers: openai, anthropic, openrouter, google, xai"
    echo ""

    if ! prompt_yes_no "Add an API key now?"; then
        print_info "You can add API keys later using the bridge CLI"
        return
    fi

    local provider=$(prompt_input "Provider (e.g., openai)" "")
    if [ -z "$provider" ]; then
        print_warning "No provider specified, skipping API key setup"
        return
    fi

    local key_id=$(prompt_input "Key ID (e.g., $provider-main)" "$provider-main")
    local key_token=""
    echo ""
    print_info "Enter your API key:"
    read -s -p "  " key_token
    echo

    if [ -z "$key_token" ]; then
        print_error "API key cannot be empty"
        return
    fi

    local display_name=$(prompt_input "Display name (e.g., 'Production Key')" "$provider API Key")

    print_info "Adding API key to keystore..."
    print_info "  Provider: $provider"
    print_info "  Key ID: $key_id"
    print_info "  Display name: $display_name"

    # Use bridge CLI to add key (if available) or show manual command
    echo ""
    echo -e "${CYAN}To add this key later, use:${NC}"
    echo "  echo '{\"jsonrpc\":\"2.0\",\"method\":\"add_key\",\"params\":{"
    echo "    \"id\":\"$key_id\","
    echo "    \"provider\":\"$provider\","
    echo "    \"token\":\"YOUR_TOKEN_HERE\","
    echo "    \"display_name\":\"$display_name\""
    echo "  },\"id\":1}' | sudo socat - UNIX-CONNECT:$socket_path"
}

#=============================================================================
# Step 9: Systemd Service Setup
#=============================================================================

step_systemd() {
    print_step 10 11 "Systemd Service Setup"

    local service_file="/etc/systemd/system/armorclaw-bridge.service"

    if [ -f "$service_file" ]; then
        print_info "Service file already exists"
        if prompt_yes_no "Recreate service file?"; then
            sudo rm -f "$service_file"
        else
            print_success "Using existing service file"
            return
        fi
    fi

    print_info "Creating systemd service..."

    sudo tee "$service_file" > /dev/null <<EOF
[Unit]
Description=ArmorClaw Bridge Service
After=network.target docker.service
Wants=docker.service

[Service]
Type=notify
NotifyAccess=all
User=armorclaw
Group=armorclaw
ExecStart=/opt/armorclaw/armorclaw-bridge -config $CONFIG_DIR/config.toml
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

    print_success "Service file created"

    # Reload systemd
    print_info "Reloading systemd daemon..."
    sudo systemctl daemon-reload

    print_success "Systemd service configured"
    echo ""
    echo -e "${CYAN}Commands to control the service:${NC}"
    echo "  Start:   sudo systemctl start armorclaw-bridge"
    echo "  Stop:    sudo systemctl stop armorclaw-bridge"
    echo "  Status:  sudo systemctl status armorclaw-bridge"
    echo "  Logs:    sudo journalctl -u armorclaw-bridge -f"
    echo "  Enable:  sudo systemctl enable armorclaw-bridge"
}

#=============================================================================
# Final Verification
#=============================================================================

step_verify() {
    print_step 11 11 "Final Verification"

    print_info "Running verification checks..."

    local all_ok=true

    # Check directories
    print_info "Checking directories..."
    if [ -d "$CONFIG_DIR" ]; then
        print_success "Config directory: $CONFIG_DIR"
    else
        print_error "Config directory missing: $CONFIG_DIR"
        all_ok=false
    fi

    if [ -d "$DATA_DIR" ]; then
        print_success "Data directory: $DATA_DIR"
    else
        print_error "Data directory missing: $DATA_DIR"
        all_ok=false
    fi

    if [ -d "$RUN_DIR" ]; then
        print_success "Run directory: $RUN_DIR"
    else
        print_error "Run directory missing: $RUN_DIR"
        all_ok=false
    fi

    # Check binary
    print_info "Checking bridge binary..."
    if [ -x "/opt/armorclaw/armorclaw-bridge" ]; then
        print_success "Bridge binary: /opt/armorclaw/armorclaw-bridge"
    else
        print_error "Bridge binary not found"
        all_ok=false
    fi

    # Check symlink
    if [ -L "/usr/local/bin/armorclaw-bridge" ]; then
        print_success "Symlink: /usr/local/bin/armorclaw-bridge"
    else
        print_warning "Symlink missing (not critical)"
    fi

    # Check service file
    print_info "Checking systemd service..."
    if [ -f "/etc/systemd/system/armorclaw-bridge.service" ]; then
        print_success "Service file installed"
    else
        print_warning "Service file not found"
    fi

    # Check config
    print_info "Checking configuration..."
    if [ -f "$CONFIG_DIR/config.toml" ]; then
        print_success "Configuration file exists"
    else
        print_error "Configuration file missing"
        all_ok=false
    fi

    if [ "$all_ok" = true ]; then
        print_success "All verification checks passed!"
    else
        print_error "Some verification checks failed"
    fi
}

#=============================================================================
# Completion Message
#=============================================================================

print_completion() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}          ${BOLD}Setup Complete!${NC}                           ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${BOLD}Next Steps:${NC}"
    echo ""
    echo "1. Start the bridge:"
    echo "   ${CYAN}sudo systemctl start armorclaw-bridge${NC}"
    echo ""
    echo "2. Enable auto-start on boot:"
    echo "   ${CYAN}sudo systemctl enable armorclaw-bridge${NC}"
    echo ""
    echo "3. Check status:"
    echo "   ${CYAN}sudo systemctl status armorclaw-bridge${NC}"
    echo ""
    echo "4. View logs:"
    echo "   ${CYAN}sudo journalctl -u armorclaw-bridge -f${NC}"
    echo ""
    echo "5. Test the bridge:"
    echo "   ${CYAN}echo '{\"jsonrpc\":\"2.0\",\"method\":\"health\",\"id\":1}' | sudo socat - UNIX-CONNECT:$SOCKET_PATH${NC}"
    echo ""

    echo -e "${BOLD}Configuration:${NC}"
    echo "  Config:  $CONFIG_DIR/config.toml"
    echo "  Data:    $DATA_DIR"
    echo "  Socket:  $SOCKET_PATH"
    echo "  Log:     $LOG_FILE"
    echo ""

    echo -e "${BOLD}Documentation:${NC}"
    echo "  Element X Quick Start: docs/guides/element-x-quickstart.md ⭐"
    echo "  Setup Guide:           docs/guides/setup-guide.md"
    echo "  Full Docs:              docs/index.md"
    echo "  GitHub:                 https://github.com/armorclaw/armorclaw"
    echo ""

    if [ -f "$LOG_FILE" ]; then
        echo -e "${BOLD}Setup log saved to:${NC} $LOG_FILE"
    fi
}

#=============================================================================
# Step 10: Start First Agent
#=============================================================================

step_start_agent() {
    print_step 12 12 "Start First Agent (Optional)"

    echo ""
    echo "Would you like to start your first agent now?"
    echo ""
    echo "This will:"
    echo "  - Start the ArmorClaw Bridge"
    echo "  - Launch an agent container with your API key"
    echo "  - Verify everything is working"
    echo ""

    if ! prompt_yes_no "Start first agent now?"; then
        echo ""
        echo -e "${YELLOW}⊘ Skipping agent startup${NC}"
        echo "You can start an agent later with:"
        echo "  ${CYAN}sudo $INSTALL_DIR/armorclaw-bridge start --key <key-id>${NC}"
        return
    fi

    echo ""
    echo -e "${BLUE}Starting ArmorClaw Bridge...${NC}"

    # Start the bridge
    if ! sudo systemctl start armorclaw-bridge 2>/dev/null; then
        echo ""
        echo -e "${RED}✗ Failed to start bridge${NC}"
        echo "Start it manually with:"
        echo "  ${CYAN}sudo systemctl start armorclaw-bridge${NC}"
        echo ""
        echo "Then start an agent with:"
        echo "  ${CYAN}sudo $INSTALL_DIR/armorclaw-bridge start --key <key-id>${NC}"
        return
    fi

    echo -e "${GREEN}✓ Bridge started${NC}"

    # Wait for bridge to be ready
    echo ""
    echo -e "${BLUE}Waiting for bridge to be ready...${NC}"
    sleep 3

    # Get the first key ID
    local key_id
    key_id=$(sudo "$INSTALL_DIR/armorclaw-bridge" list-keys 2>/dev/null | grep -m1 '•' | awk '{print $2}' || echo "")

    if [ -z "$key_id" ]; then
        echo ""
        echo -e "${YELLOW}⚠ No API keys found in keystore${NC}"
        echo "Add a key with:"
        echo "  ${CYAN}sudo $INSTALL_DIR/armorclaw-bridge add-key --provider <provider> --token <token>${NC}"
        return
    fi

    echo ""
    echo -e "${BLUE}Starting agent with key: ${key_id}${NC}"

    # Start the agent
    local output
    output=$(sudo "$INSTALL_DIR/armorclaw-bridge" start --key "$key_id" 2>&1)
    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo ""
        echo -e "${RED}✗ Failed to start agent${NC}"
        echo "$output"
        echo ""
        echo "Check bridge logs with:"
        echo "  ${CYAN}sudo journalctl -u armorclaw-bridge -n 50${NC}"
        return
    fi

    # Parse container ID from output
    local container_id
    container_id=$(echo "$output" | grep -oP 'Container ID: \K[a-f0-9]+' || echo "")

    echo -e "${GREEN}✓ Agent started successfully${NC}"

    if [ -n "$container_id" ]; then
        echo ""
        echo "  Container ID: ${CYAN}$container_id${NC}"
    fi

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}     ${BOLD}🎉 First Agent Running Successfully!${NC}            ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${BOLD}Check agent status:${NC}"
    echo "  ${CYAN}sudo $INSTALL_DIR/armorclaw-bridge status${NC}"
    echo ""

    echo -e "${BOLD}View agent logs:${NC}"
    if [ -n "$container_id" ]; then
        echo "  ${CYAN}docker logs -f $container_id${NC}"
    else
        echo "  ${CYAN}docker logs -f <container-id>${NC}"
    fi
    echo ""

    echo -e "${BOLD}Stop the agent:${NC}"
    echo "  ${CYAN}sudo $INSTALL_DIR/armorclaw-bridge stop --name my-agent${NC}"
    echo ""

    echo -e "${BOLD}Next Steps:${NC}"
    echo "  1. Connect via Element X (see docs/guides/element-x-quickstart.md)"
    echo "  2. Send commands to your agent"
    echo "  3. Configure agent with /attach_config"
    echo ""
}

#=============================================================================
# Main Setup Flow
#=============================================================================

main() {
    # Initialize logging
    init_logging

    # Print welcome
    print_header
    step_welcome

    # Run all steps
    step_prerequisites
    step_docker
    step_container
    step_bridge_install
    step_budget_confirmation
    step_configuration
    step_keystore
    step_api_key
    step_systemd
    step_verify
    step_start_agent

    # Print completion message
    print_completion
}

# Run main function
main "$@"
