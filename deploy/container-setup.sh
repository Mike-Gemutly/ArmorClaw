#!/bin/bash
# ArmorClaw Container Setup Wizard
# Simplified setup for Docker container deployment
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

# Paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
RUN_DIR="/run/armorclaw"
CONFIG_FILE="$CONFIG_DIR/config.toml"
SETUP_FLAG="$CONFIG_DIR/.setup_complete"

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║        ${BOLD}ArmorClaw Container Setup${NC}${CYAN}                     ║"
    echo "║        ${BOLD}Version 1.0.0${NC}${CYAN}                                  ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_step() {
    local step="$1"
    local total="$2"
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Step $step of $total: ${BOLD}$3${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ ERROR: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
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

#=============================================================================
# Environment Variable Support
#=============================================================================

# Check for environment variables (non-interactive mode)
check_env_vars() {
    local has_all=true

    if [ -z "$ARMORCLAW_MATRIX_SERVER" ]; then
        has_all=false
    fi
    if [ -z "$ARMORCLAW_API_KEY" ]; then
        has_all=false
    fi

    if [ "$has_all" = true ]; then
        print_info "Environment variables detected - using non-interactive mode"
        NON_INTERACTIVE=true
        MATRIX_SERVER="${ARMORCLAW_MATRIX_SERVER}"
        MATRIX_URL="${ARMORCLAW_MATRIX_URL:-http://localhost:6167}"
        API_KEY="${ARMORCLAW_API_KEY}"
        API_BASE_URL="${ARMORCLAW_API_BASE_URL:-https://api.openai.com/v1}"
        BRIDGE_PASSWORD="${ARMORCLAW_BRIDGE_PASSWORD:-bridge123}"
        LOG_LEVEL="${ARMORCLAW_LOG_LEVEL:-info}"
    else
        NON_INTERACTIVE=false
    fi
}

#=============================================================================
# Setup Functions
#=============================================================================

create_directories() {
    print_info "Creating directory structure..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$RUN_DIR"
    mkdir -p "/var/log/armorclaw"

    print_success "Directories created"
}

check_docker_socket() {
    print_info "Checking Docker socket..."

    if [ ! -S /var/run/docker.sock ]; then
        print_error "Docker socket not found at /var/run/docker.sock"
        print_error "This container requires Docker socket access."
        echo ""
        echo "Run with: docker run -v /var/run/docker.sock:/var/run/docker.sock ..."
        exit 1
    fi

    # Test Docker access
    if ! docker info >/dev/null 2>&1; then
        print_error "Cannot connect to Docker daemon"
        print_error "Ensure your user has Docker permissions"
        exit 1
    fi

    print_success "Docker socket accessible"
}

configure_matrix() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for Matrix configuration"
    else
        print_step 1 5 "Matrix Homeserver Configuration"

        echo "Enter your Matrix server details:"
        echo ""

        MATRIX_SERVER=$(prompt_input "Matrix server domain (e.g., matrix.example.com)" "")
        if [ -z "$MATRIX_SERVER" ]; then
            print_error "Matrix server domain is required"
            exit 1
        fi

        MATRIX_URL=$(prompt_input "Matrix homeserver URL" "http://localhost:6167")

        echo ""
        if prompt_yes_no "Create bridge user on Matrix?" "y"; then
            BRIDGE_USER=$(prompt_input "Bridge username" "bridge")
            BRIDGE_PASSWORD=$(prompt_input "Bridge password" "")
            if [ -z "$BRIDGE_PASSWORD" ]; then
                # Generate random password
                BRIDGE_PASSWORD=$(openssl rand -base64 16 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 16 | head -n 1)
                print_info "Generated password: $BRIDGE_PASSWORD"
            fi
        else
            BRIDGE_USER="bridge"
            BRIDGE_PASSWORD="bridge123"
        fi
    fi

    print_success "Matrix configuration complete"
}

configure_api() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for API configuration"
    else
        print_step 2 5 "AI Provider Configuration"

        echo "Select your AI provider:"
        echo "  1) OpenAI"
        echo "  2) Anthropic (Claude)"
        echo "  3) GLM-5 (Zhipu AI)"
        echo "  4) Custom (OpenAI-compatible)"
        echo ""

        local choice=$(prompt_input "Provider" "1")

        case "$choice" in
            1)
                API_BASE_URL="https://api.openai.com/v1"
                ;;
            2)
                API_BASE_URL="https://api.anthropic.com/v1"
                ;;
            3)
                API_BASE_URL="https://api.z.ai/api/coding/paas/v4"
                ;;
            4)
                API_BASE_URL=$(prompt_input "API base URL" "")
                ;;
            *)
                print_warning "Invalid choice, using OpenAI"
                API_BASE_URL="https://api.openai.com/v1"
                ;;
        esac

        API_KEY=$(prompt_input "API key" "")
        if [ -z "$API_KEY" ]; then
            print_error "API key is required"
            exit 1
        fi
    fi

    print_success "API configuration complete"
}

configure_bridge() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for bridge configuration"
    else
        print_step 3 5 "Bridge Configuration"

        LOG_LEVEL=$(prompt_input "Log level (debug/info/warn)" "info")
        SOCKET_PATH=$(prompt_input "Bridge socket path" "/run/armorclaw/bridge.sock")
    fi

    print_success "Bridge configuration complete"
}

write_config() {
    print_step 4 5 "Writing Configuration"

    print_info "Creating config.toml..."

    cat > "$CONFIG_FILE" << EOF
# ArmorClaw Bridge Configuration
# Generated by container-setup.sh on $(date -Iseconds)

[server]
socket_path = "${SOCKET_PATH:-/run/armorclaw/bridge.sock}"
pid_file = "/run/armorclaw/bridge.pid"
daemonize = false

[keystore]
db_path = "${DATA_DIR}/keystore.db"
master_key = ""
providers = []

[matrix]
enabled = true
homeserver_url = "${MATRIX_URL}"
username = "${BRIDGE_USER:-bridge}"
password = "${BRIDGE_PASSWORD}"
device_id = "armorclaw-bridge"
sync_interval = 5
auto_rooms = []

[matrix.retry]
max_retries = 3
retry_delay = 5
backoff_multiplier = 2.0

[matrix.zero_trust]
trusted_senders = []
trusted_rooms = []
reject_untrusted = false

[budget]
daily_limit_usd = 5.0
monthly_limit_usd = 100.0
alert_threshold = 80.0
hard_stop = true

[logging]
level = "${LOG_LEVEL:-info}"
format = "json"
output = "stdout"

[discovery]
enabled = true
port = 8080
tls = false
api_path = "/api"
ws_path = "/ws"

[notifications]
enabled = false
alert_threshold = 0.8

[eventbus]
websocket_enabled = false
websocket_addr = "0.0.0.0:8444"
websocket_path = "/events"
max_subscribers = 100
inactivity_timeout = "30m"

[errors]
enabled = true
store_enabled = true
notify_enabled = true
store_path = "${DATA_DIR}/errors.db"
retention_days = 30
rate_limit_window = "5m"
retention_period = "24h"

[compliance]
enabled = false
streaming_mode = true
quarantine_enabled = false
notify_on_quarantine = false
audit_enabled = false
audit_retention_days = 30
tier = "basic"
EOF

    chmod 600 "$CONFIG_FILE"
    print_success "Configuration written to $CONFIG_FILE"
}

initialize_keystore() {
    print_step 5 5 "Initializing Keystore"

    print_info "Keystore will be initialized by bridge on first run..."

    # Create keystore directory
    mkdir -p "$DATA_DIR"

    # Store API key temporarily for later injection
    # The bridge will initialize its own SQLCipher keystore
    print_info "Storing API key for bridge injection..."

    # Create a temp file with API key for the bridge to read on startup
    cat > "$DATA_DIR/.api_key_temp" << EOF
api_key=$API_KEY
base_url=$API_BASE_URL
EOF
    chmod 600 "$DATA_DIR/.api_key_temp"

    print_success "Keystore prepared (bridge will initialize on startup)"
}

add_api_key_to_bridge() {
    print_info "Adding API key to bridge keystore..."

    # Wait for bridge socket
    local max_attempts=30
    local attempt=0
    while [ ! -S /run/armorclaw/bridge.sock ] && [ $attempt -lt $max_attempts ]; do
        sleep 1
        attempt=$((attempt + 1))
    done

    if [ -S /run/armorclaw/bridge.sock ]; then
        # Add API key via RPC
        echo "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"keystore.add_provider\",\"params\":{\"provider\":\"openai\",\"token\":\"$API_KEY\",\"display_name\":\"Default API\"}}" | \
            socat - UNIX-CONNECT:/run/armorclaw/bridge.sock 2>/dev/null || true

        # Clean up temp file
        rm -f "$DATA_DIR/.api_key_temp"

        print_success "API key added to keystore"
    else
        print_warning "Bridge socket not available - API key stored in $DATA_DIR/.api_key_temp"
        print_warning "Add manually with: keystore.add_provider RPC method"
    fi
}

start_matrix_stack() {
    print_info "Starting Matrix stack..."

    cd /opt/armorclaw

    if [ -f "docker-compose.matrix.yml" ]; then
        # Check if Matrix is already running
        if docker ps | grep -q "armorclaw-matrix\|matrix-conduit"; then
            print_success "Matrix stack already running"
        else
            print_info "Starting Matrix containers..."
            docker compose -f docker-compose.matrix.yml up -d 2>/dev/null || true
            sleep 5
        fi
    else
        print_warning "docker-compose.matrix.yml not found - skipping Matrix stack"
    fi
}

final_summary() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}        ${BOLD}Setup Complete!${NC}                                  ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Configuration:"
    echo "  - Matrix server: $MATRIX_SERVER"
    echo "  - Matrix URL: $MATRIX_URL"
    echo "  - API provider: $API_BASE_URL"
    echo "  - Log level: ${LOG_LEVEL:-info}"
    echo ""
    echo "Files:"
    echo "  - Config: $CONFIG_FILE"
    echo "  - Keystore: $DATA_DIR/keystore.db"
    echo ""
    echo "Next steps:"
    echo "  1. Connect ArmorChat to: https://$MATRIX_SERVER"
    echo "  2. Start DM with: @bridge:$MATRIX_SERVER"
    echo "  3. Send '!status' to verify connection"
    echo ""

    # Mark setup complete
    touch "$SETUP_FLAG"
}

#=============================================================================
# Main
#=============================================================================

main() {
    print_header

    # Check for environment variables
    check_env_vars

    # Create directories
    create_directories

    # Check Docker socket
    check_docker_socket

    # Run configuration steps
    configure_matrix
    configure_api
    configure_bridge
    write_config
    initialize_keystore

    # Start Matrix if available
    start_matrix_stack

    # Show summary
    final_summary
}

# Run main
main "$@"
