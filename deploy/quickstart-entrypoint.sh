#!/bin/bash
# ArmorClaw Quick Start Entrypoint - Enhanced UX
# Version: 0.4.0
#
# Improvements:
#   - Auto port detection with fallback
#   - Bootstrap mode (generates docker-compose.yml)
#   - Simplified Quick mode wizard (4 questions only)
#
# Usage:
#   docker run -it armorclaw/quickstart              # Interactive
#   docker run -d armorclaw/quickstart --bootstrap   # Generate compose file
#   docker run -d armorclaw/quickstart --show-config # Print config and exit

set -o pipefail

CONFIG_FILE="/etc/armorclaw/config.toml"
SETUP_FLAG="/etc/armorclaw/.setup_complete"
API_KEY_TEMP="/var/lib/armorclaw/.api_key_temp"
COMPOSE_FILE="/opt/armorclaw/docker-compose.generated.yml"
LOG_FILE="/var/log/armorclaw/setup.log"
VERSION="0.4.0"

# Default ports
DEFAULT_BRIDGE_PORT=8443
DEFAULT_MATRIX_PORT=6167
DEFAULT_PUSH_PORT=5000

# Detected ports (will be set by auto_detect_ports)
BRIDGE_PORT=""
MATRIX_PORT=""
PUSH_PORT=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

#=============================================================================
# Logging
#=============================================================================

init_logging() {
    mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
    touch "$LOG_FILE" 2>/dev/null || true
}

log() {
    local level="$1"; shift
    local msg="$*"
    local ts=$(date -Iseconds 2>/dev/null || date '+%Y-%m-%dT%H:%M:%S%z')
    echo "[$ts] [$level] $msg" >> "$LOG_FILE" 2>/dev/null || true
}
log_info()  { log "INFO" "$@"; }
log_error() { log "ERROR" "$@"; }
log_debug() { [ "${ARMORCLAW_DEBUG:-}" = true ] && log "DEBUG" "$@"; }

init_logging
exec > >(tee -a "$LOG_FILE") 2>&1
log_info "ArmorClaw Quick Start v${VERSION} starting"

#=============================================================================
# Auto Port Detection
#=============================================================================

check_port_available() {
    local port="$1"
    # Check if port is in use using ss (more reliable than netstat)
    if command -v ss >/dev/null 2>&1; then
        if ss -ltn 2>/dev/null | grep -qE ":${port}\s"; then
            return 1  # Port in use
        fi
    # Fallback to netstat
    elif command -v netstat >/dev/null 2>&1; then
        if netstat -ltn 2>/dev/null | grep -qE ":${port}\s"; then
            return 1
        fi
    # Fallback to /proc/net/tcp
    elif [ -f /proc/net/tcp ]; then
        local hex_port=$(printf '%04X' "$port")
        if grep -q ":${hex_port}" /proc/net/tcp 2>/dev/null; then
            return 1
        fi
    fi
    return 0  # Port available
}

find_available_port() {
    local start_port="$1"
    local max_attempts=1000
    local port="$start_port"

    # Try starting port first
    if check_port_available "$port"; then
        echo "$port"
        return 0
    fi

    # Find random available port in range
    while [ $((port - start_port)) -lt $max_attempts ]; do
        # Try high ports first (30000-40000 range)
        port=$((RANDOM % 10000 + 30000))
        if check_port_available "$port"; then
            echo "$port"
            return 0
        fi
    done

    # Fallback to sequential search
    port=$((start_port + 1))
    while [ $((port - start_port)) -lt $max_attempts ]; do
        if check_port_available "$port"; then
            echo "$port"
            return 0
        fi
        port=$((port + 1))
    done

    echo ""
    return 1
}

auto_detect_ports() {
    echo -e "${CYAN}Detecting available ports...${NC}"
    log_info "Starting port detection"

    # Bridge RPC port
    if check_port_available "$DEFAULT_BRIDGE_PORT"; then
        BRIDGE_PORT="$DEFAULT_BRIDGE_PORT"
        log_info "Bridge port: $BRIDGE_PORT (default)"
    else
        BRIDGE_PORT=$(find_available_port "$DEFAULT_BRIDGE_PORT")
        if [ -z "$BRIDGE_PORT" ]; then
            echo -e "${RED}ERROR: Could not find available port for Bridge${NC}"
            log_error "No available port for Bridge"
            return 1
        fi
        log_info "Bridge port: $BRIDGE_PORT (auto-selected)"
    fi

    # Matrix port
    if check_port_available "$DEFAULT_MATRIX_PORT"; then
        MATRIX_PORT="$DEFAULT_MATRIX_PORT"
        log_info "Matrix port: $MATRIX_PORT (default)"
    else
        MATRIX_PORT=$(find_available_port "$DEFAULT_MATRIX_PORT")
        if [ -z "$MATRIX_PORT" ]; then
            echo -e "${RED}ERROR: Could not find available port for Matrix${NC}"
            log_error "No available port for Matrix"
            return 1
        fi
        log_info "Matrix port: $MATRIX_PORT (auto-selected)"
    fi

    # Push gateway port
    if check_port_available "$DEFAULT_PUSH_PORT"; then
        PUSH_PORT="$DEFAULT_PUSH_PORT"
        log_info "Push port: $PUSH_PORT (default)"
    else
        PUSH_PORT=$(find_available_port "$DEFAULT_PUSH_PORT")
        if [ -z "$PUSH_PORT" ]; then
            echo -e "${RED}ERROR: Could not find available port for Push gateway${NC}"
            log_error "No available port for Push gateway"
            return 1
        fi
        log_info "Push port: $PUSH_PORT (auto-selected)"
    fi

    echo -e "${GREEN}Port configuration:${NC}"
    echo -e "  ${DIM}Bridge RPC:${NC}  $BRIDGE_PORT"
    echo -e "  ${DIM}Matrix:${NC}      $MATRIX_PORT"
    echo -e "  ${DIM}Push Gateway:${NC} $PUSH_PORT"
    echo ""

    return 0
}

#=============================================================================
# Bootstrap Mode - Generate Docker Compose
#=============================================================================

generate_docker_compose() {
    local output_file="${1:-$COMPOSE_FILE}"

    # Detect server IP
    local server_ip
    server_ip=$(ip route get 1 2>/dev/null | awk '{print $7; exit}' || hostname -I 2>/dev/null | awk '{print $1}')
    [ -z "$server_ip" ] && server_ip="your-server-ip"

    log_info "Generating docker-compose.yml for server: $server_ip"

    cat > "$output_file" << EOF
# ArmorClaw Docker Compose Configuration
# Generated: $(date -Iseconds)
# Server IP: $server_ip

services:
  bridge:
    image: mikegemut/armorclaw:latest
    container_name: armorclaw
    restart: unless-stopped
    user: root
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - armorclaw-config:/etc/armorclaw
      - armorclaw-keystore:/var/lib/armorclaw
    ports:
      - "${BRIDGE_PORT:-8443}:8443"
      - "${MATRIX_PORT:-6167}:6167"
      - "${PUSH_PORT:-5000}:5000"
    environment:
      - ARMORCLAW_SERVER_NAME=${ARMORCLAW_SERVER_NAME:-$server_ip}
      ${ARMORCLAW_API_KEY:+      - ARMORCLAW_API_KEY=${ARMORCLAW_API_KEY}}
      ${ARMORCLAW_ADMIN_USER:+      - ARMORCLAW_ADMIN_USER=${ARMORCLAW_ADMIN_USER}}
      ${ARMORCLAW_ADMIN_PASSWORD:+      - ARMORCLAW_ADMIN_PASSWORD=${ARMORCLAW_ADMIN_PASSWORD}}
      ${ARMORCLAW_PROVIDER:+      - ARMORCLAW_PROVIDER=${ARMORCLAW_PROVIDER}}
    healthcheck:
      test: ["CMD", "socat", "-t", "1", "STDIN", "UNIX-CONNECT:/run/armorclaw/bridge.sock"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

volumes:
  armorclaw-config:
    name: armorclaw-config
  armorclaw-keystore:
    name: armorclaw-keystore
EOF

    echo -e "${GREEN}Generated: $output_file${NC}"
    echo ""
    echo -e "${CYAN}To deploy:${NC}"
    echo "  docker compose -f $output_file up -d"
    echo ""
    echo -e "${CYAN}To customize:${NC}"
    echo "  export ARMORCLAW_SERVER_NAME=your-domain.com"
    echo "  export ARMORCLAW_API_KEY=sk-your-key"
    echo "  export ARMORCLAW_ADMIN_USER=admin"
    echo "  export ARMORCLAW_ADMIN_PASSWORD=\$(openssl rand -base64 24)"
    echo ""
}

#=============================================================================
# Simplified Quick Mode Wizard
#=============================================================================

run_quick_wizard() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║        ${BOLD}ArmorClaw Quick Setup${NC}${CYAN}                        ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""

    # Step 1: AI Provider Selection
    echo -e "${BOLD}Step 1: Choose AI Provider${NC}"
    echo ""
    echo "  1) OpenAI (GPT-4, GPT-4o)"
    echo "  2) Anthropic (Claude)"
    echo "  3) Google (Gemini)"
    echo "  4) OpenRouter (Multi-provider)"
    echo "  5) xAI (Grok)"
    echo "  6) Skip for now"
    echo ""

    local provider_choice=""
    while [ -z "$provider_choice" ]; do
        read -rp "Select provider [1-6]: " provider_choice
        case "$provider_choice" in
            1) PROVIDER="openai";;
            2) PROVIDER="anthropic";;
            3) PROVIDER="google";;
            4) PROVIDER="openrouter";;
            5) PROVIDER="xai";;
            6) PROVIDER="skip";;
            *) provider_choice=""; echo -e "${RED}Invalid choice. Enter 1-6.${NC}";;
        esac
    done

    if [ "$PROVIDER" = "skip" ]; then
        echo -e "${YELLOW}Skipping API key. Add later via ArmorChat or RPC.${NC}"
        API_KEY=""
    else
        # Step 2: API Key
        echo ""
        echo -e "${BOLD}Step 2: Enter API Key${NC}"
        echo -e "${DIM}Your key is stored encrypted and never logged.${NC}"
        echo ""

        local key_prompt="Enter ${PROVIDER} API key"
        [ "$PROVIDER" = "openai" ] && key_prompt="Enter OpenAI API key (sk-...)"
        [ "$PROVIDER" = "anthropic" ] && key_prompt="Enter Anthropic API key (sk-ant-...)"

        while [ -z "$API_KEY" ]; do
            read -rsp "$key_prompt: " API_KEY
            echo ""
            if [ -z "$API_KEY" ]; then
                echo -e "${YELLOW}Press Enter to skip, or enter your key.${NC}"
                read -rsp "Skip? [y/N]: " skip_choice
                [ "${skip_choice,,}" = "y" ] && break
            fi
        done
    fi

    # Step 3: Admin Username
    echo ""
    echo -e "${BOLD}Step 3: Admin Username${NC}"
    echo ""

    ADMIN_USER="${ARMORCLAW_ADMIN_USER:-}"
    if [ -z "$ADMIN_USER" ]; then
        read -rp "Admin username [admin]: " ADMIN_USER
        [ -z "$ADMIN_USER" ] && ADMIN_USER="admin"
    fi

    # Step 4: Admin Password
    echo ""
    echo -e "${BOLD}Step 4: Admin Password${NC}"
    echo -e "${DIM}Leave empty to auto-generate a secure password.${NC}"
    echo ""

    ADMIN_PASSWORD="${ARMORCLAW_ADMIN_PASSWORD:-}"
    if [ -z "$ADMIN_PASSWORD" ]; then
        read -rsp "Admin password [auto-generate]: " ADMIN_PASSWORD
        echo ""
        if [ -z "$ADMIN_PASSWORD" ]; then
            ADMIN_PASSWORD=$(openssl rand -base64 24 2>/dev/null || tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
            echo -e "${GREEN}Generated password: ${ADMIN_PASSWORD}${NC}"
            echo -e "${YELLOW}Save this password!${NC}"
        fi
    fi

    # Summary
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}Configuration Summary${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${DIM}AI Provider:${NC}  ${PROVIDER:-skipped}"
    echo -e "  ${DIM}Admin User:${NC}   ${ADMIN_USER}"
    echo -e "  ${DIM}Bridge Port:${NC}  ${BRIDGE_PORT}"
    echo -e "  ${DIM}Matrix Port:${NC}  ${MATRIX_PORT}"
    echo -e "  ${DIM}Push Port:${NC}    ${PUSH_PORT}"
    echo ""

    local confirm=""
    read -rp "Deploy with these settings? [Y/n]: " confirm
    [ -z "$confirm" ] && confirm="y"

    if [ "${confirm,,}" != "y" ]; then
        echo -e "${YELLOW}Cancelled.${NC}"
        return 1
    fi

    return 0
}

#=============================================================================
# Environment Variable Mode (Non-Interactive)
#=============================================================================

run_env_mode() {
    echo -e "${CYAN}Running in non-interactive mode (environment variables)${NC}"
    log_info "Using environment variable mode"

    # Extract from env vars
    PROVIDER="${ARMORCLAW_PROVIDER:-openai}"
    API_KEY="${ARMORCLAW_API_KEY:-}"
    ADMIN_USER="${ARMORCLAW_ADMIN_USER:-admin}"
    ADMIN_PASSWORD="${ARMORCLAW_ADMIN_PASSWORD:-}"

    # Generate password if not provided
    if [ -z "$ADMIN_PASSWORD" ]; then
        ADMIN_PASSWORD=$(openssl rand -base64 24 2>/dev/null || tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
        echo -e "${GREEN}Generated admin password: ${ADMIN_PASSWORD}${NC}"
    fi

    log_info "Provider: $PROVIDER, Admin: $ADMIN_USER"
}

#=============================================================================
# Configuration Generation
#=============================================================================

generate_config() {
    # Detect server IP/name
    local server_name="${ARMORCLAW_SERVER_NAME:-}"
    if [ -z "$server_name" ]; then
        server_name=$(ip route get 1 2>/dev/null | awk '{print $7; exit}' || hostname -I 2>/dev/null | awk '{print $1}')
        [ -z "$server_name" ] && server_name="localhost"
    fi

    log_info "Generating config for server: $server_name"

    mkdir -p /etc/armorclaw /var/lib/armorclaw /run/armorclaw

    cat > "$CONFIG_FILE" << EOF
# ArmorClaw Configuration
# Generated: $(date -Iseconds)

socket_path = "/run/armorclaw/bridge.sock"
db_path = "/var/lib/armorclaw/keystore.db"
log_level = "info"

[server]
socket_path = "/run/armorclaw/bridge.sock"
http_port = ${BRIDGE_PORT:-8443}

[keystore]
db_path = "/var/lib/armorclaw/keystore.db"

[matrix]
enabled = true
homeserver_url = "http://localhost:${MATRIX_PORT:-6167}"
server_name = "${server_name}"

[push]
enabled = true
gateway_url = "http://localhost:${PUSH_PORT:-5000}"

[logging]
level = "info"
format = "text"
output = "stdout"
EOF

    # Save admin info for later role assignment
    echo "$ADMIN_USER" > /var/lib/armorclaw/.admin_user
    echo "$server_name" >> /var/lib/armorclaw/.admin_user

    # Save API key for injection
    if [ -n "$API_KEY" ]; then
        cat > "$API_KEY_TEMP" << EOF
provider="${PROVIDER}"
api_key="${API_KEY}"
EOF
        chmod 600 "$API_KEY_TEMP"
    fi

    # Save password
    echo "$ADMIN_PASSWORD" > /var/lib/armorclaw/.admin_password
    chmod 600 /var/lib/armorclaw/.admin_password

    log_info "Configuration generated"
}

#=============================================================================
# Connection Info Display
#=============================================================================

show_connection_info() {
    local server_name="${ARMORCLAW_SERVER_NAME:-}"
    [ -z "$server_name" ] && server_name=$(ip route get 1 2>/dev/null | awk '{print $7; exit}' || echo "localhost")

    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}${BOLD}ArmorClaw is Ready!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${BOLD}Connection Info:${NC}"
    echo ""
    echo -e "  ${DIM}Matrix Server:${NC}  http://${server_name}:${MATRIX_PORT}"
    echo -e "  ${DIM}Bridge RPC:${NC}     http://${server_name}:${BRIDGE_PORT}"
    echo -e "  ${DIM}Push Gateway:${NC}   http://${server_name}:${PUSH_PORT}"
    echo ""
    echo -e "${BOLD}Admin Credentials:${NC}"
    echo ""
    echo -e "  ${DIM}Username:${NC}  ${ADMIN_USER}"
    echo -e "  ${DIM}Password:${NC}  ${ADMIN_PASSWORD}"
    echo ""
    echo -e "${CYAN}Connect via Element X or ArmorChat${NC}"
    echo ""
}

#=============================================================================
# Main Entry Point
#=============================================================================

# Parse command line arguments
BOOTSTRAP_MODE=false
SHOW_CONFIG=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --bootstrap)
            BOOTSTRAP_MODE=true
            shift
            ;;
        --show-config)
            SHOW_CONFIG=true
            shift
            ;;
        --help|-h)
            echo "ArmorClaw Quick Start v${VERSION}"
            echo ""
            echo "Usage: docker run [OPTIONS] armorclaw/quickstart [COMMAND]"
            echo ""
            echo "Commands:"
            echo "  (none)          Start interactive setup"
            echo "  --bootstrap     Generate docker-compose.yml and exit"
            echo "  --show-config   Show current configuration"
            echo "  --help          Show this help"
            echo ""
            echo "Environment Variables (non-interactive mode):"
            echo "  ARMORCLAW_PROVIDER       AI provider (openai, anthropic, google, openrouter, xai)"
            echo "  ARMORCLAW_API_KEY        API key for AI provider"
            echo "  ARMORCLAW_ADMIN_USER     Admin username (default: admin)"
            echo "  ARMORCLAW_ADMIN_PASSWORD Admin password (auto-generated if not set)"
            echo "  ARMORCLAW_SERVER_NAME    Server hostname or IP"
            echo ""
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Bootstrap mode: generate compose file and exit
if [ "$BOOTSTRAP_MODE" = true ]; then
    auto_detect_ports || exit 1
    generate_docker_compose "/opt/armorclaw/docker-compose.yml"
    exit 0
fi

# Show config mode
if [ "$SHOW_CONFIG" = true ]; then
    if [ -f "$CONFIG_FILE" ]; then
        cat "$CONFIG_FILE"
    else
        echo "No configuration found. Run setup first."
        exit 1
    fi
    exit 0
fi

# Check if setup already completed
if [ -f "$SETUP_FLAG" ]; then
    echo -e "${GREEN}Setup already completed. Starting bridge...${NC}"
    log_info "Setup already completed, skipping wizard"
else
    # Auto-detect ports first
    auto_detect_ports || exit 1

    # Run setup wizard (interactive or env-based)
    if [ -n "${ARMORCLAW_API_KEY:-}" ] || [ -n "${ARMORCLAW_ADMIN_PASSWORD:-}" ]; then
        run_env_mode
    else
        run_quick_wizard || exit 1
    fi

    # Generate configuration
    generate_config

    # Mark setup complete
    touch "$SETUP_FLAG"
    log_info "Setup completed"
fi

# Show connection info
show_connection_info

# Start the bridge (delegate to original entrypoint)
echo -e "${CYAN}Starting ArmorClaw Bridge...${NC}"
log_info "Starting bridge"

if [ -x /opt/armorclaw/armorclaw-bridge ]; then
    exec /opt/armorclaw/armorclaw-bridge
else
    echo -e "${RED}Bridge binary not found at /opt/armorclaw/armorclaw-bridge${NC}"
    exit 1
fi
