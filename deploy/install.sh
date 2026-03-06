#!/bin/bash
# ArmorClaw Quick Install Script
# Version: 0.4.0
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
#
# Options:
#   --bootstrap      Generate docker-compose.yml for deployment
#   --bridge-only    Run bridge only, no Matrix (for testing)
#   --no-start       Install but don't start the container
#   --ports          Show detected ports
#   --help           Show this help

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# Defaults
BOOTSTRAP_MODE=false
BRIDGE_ONLY=false
NO_START=false
SHOW_PORTS=false
CONTAINER_NAME="armorclaw"
IMAGE="mikegemut/armorclaw:latest"
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
COMPOSE_FILE="/opt/armorclaw/docker-compose.yml"

# Default ports
BRIDGE_PORT=8443
MATRIX_PORT=6167
PUSH_PORT=5000

#=============================================================================
# Environment Variable Pass-through
#=============================================================================

# Build environment variable args for docker run (for non-interactive mode)
# Only pass env vars when ARMORCLAW_API_KEY is set (triggers non-interactive mode)
build_env_args() {
    local args=""

    # Only pass environment variables when non-interactive mode is requested
    # If ARMORCLAW_API_KEY is set, pass all config vars for unattended setup
    if [ -n "${ARMORCLAW_API_KEY:-}" ]; then
        args="-e ARMORCLAW_SERVER_NAME=${DETECTED_SERVER_IP}"
        args="$args -e ARMORCLAW_API_KEY=${ARMORCLAW_API_KEY}"
        [ -n "${ARMORCLAW_API_BASE_URL:-}" ] && args="$args -e ARMORCLAW_API_BASE_URL=${ARMORCLAW_API_BASE_URL}"
        [ -n "${ARMORCLAW_PROFILE:-}" ] && args="$args -e ARMORCLAW_PROFILE=${ARMORCLAW_PROFILE}"
        [ -n "${ARMORCLAW_ADMIN_PASSWORD:-}" ] && args="$args -e ARMORCLAW_ADMIN_PASSWORD=${ARMORCLAW_ADMIN_PASSWORD}"
    fi

    echo "$args"
}

#=============================================================================
# Port Detection
#=============================================================================

check_port() {
    local port="$1"
    if ss -ltn 2>/dev/null | grep -qE ":${port}\s"; then
        return 1  # In use
    fi
    return 0  # Available
}

find_port() {
    local start="$1"
    local port="$start"

    if check_port "$port"; then
        echo "$port"
        return
    fi

    # Try random high ports
    for i in {1..100}; do
        port=$((RANDOM % 10000 + 30000))
        if check_port "$port"; then
            echo "$port"
            return
        fi
    done

    # Sequential fallback
    port=$((start + 1))
    while [ $((port - start)) -lt 1000 ]; do
        if check_port "$port"; then
            echo "$port"
            return
        fi
        port=$((port + 1))
    done
}

auto_detect_ports() {
    echo -e "${CYAN}Detecting available ports...${NC}"

    if ! check_port "$BRIDGE_PORT"; then
        BRIDGE_PORT=$(find_port "$BRIDGE_PORT")
        [ -z "$BRIDGE_PORT" ] && { echo -e "${RED}No available port for Bridge${NC}"; exit 1; }
    fi

    if ! check_port "$MATRIX_PORT"; then
        MATRIX_PORT=$(find_port "$MATRIX_PORT")
        [ -z "$MATRIX_PORT" ] && { echo -e "${RED}No available port for Matrix${NC}"; exit 1; }
    fi

    if ! check_port "$PUSH_PORT"; then
        PUSH_PORT=$(find_port "$PUSH_PORT")
        [ -z "$PUSH_PORT" ] && { echo -e "${RED}No available port for Push${NC}"; exit 1; }
    fi

    echo -e "${GREEN}Ports detected:${NC}"
    echo -e "  ${DIM}Bridge:${NC}  $BRIDGE_PORT"
    echo -e "  ${DIM}Matrix:${NC}  $MATRIX_PORT"
    echo -e "  ${DIM}Push:${NC}    $PUSH_PORT"
    echo ""
}

#=============================================================================
# Bootstrap Mode - Generate Compose File
#=============================================================================

generate_compose() {
    local server_ip
    server_ip=$(ip route get 1 2>/dev/null | awk '{print $7; exit}' || hostname -I 2>/dev/null | awk '{print $1}')
    [ -z "$server_ip" ] && server_ip="your-server-ip"

    mkdir -p /opt/armorclaw

    cat > "$COMPOSE_FILE" << EOF
# ArmorClaw Docker Compose
# Generated: $(date -Iseconds)
# Server: $server_ip

services:
  armorclaw:
    image: $IMAGE
    container_name: $CONTAINER_NAME
    restart: unless-stopped
    user: root
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - armorclaw-config:/etc/armorclaw
      - armorclaw-keystore:/var/lib/armorclaw
    ports:
      - "${BRIDGE_PORT}:8443"
      - "${MATRIX_PORT}:6167"
      - "${PUSH_PORT}:5000"
    environment:
      - ARMORCLAW_SERVER_NAME=${ARMORCLAW_SERVER_NAME:-$server_ip}
EOF

    # Add optional env vars
    [ -n "${ARMORCLAW_API_KEY:-}" ] && echo "      - ARMORCLAW_API_KEY=${ARMORCLAW_API_KEY}" >> "$COMPOSE_FILE"
    [ -n "${ARMORCLAW_API_BASE_URL:-}" ] && echo "      - ARMORCLAW_API_BASE_URL=${ARMORCLAW_API_BASE_URL}" >> "$COMPOSE_FILE"
    [ -n "${ARMORCLAW_PROFILE:-}" ] && echo "      - ARMORCLAW_PROFILE=${ARMORCLAW_PROFILE}" >> "$COMPOSE_FILE"
    [ -n "${ARMORCLAW_ADMIN_PASSWORD:-}" ] && echo "      - ARMORCLAW_ADMIN_PASSWORD=${ARMORCLAW_ADMIN_PASSWORD}" >> "$COMPOSE_FILE"

    cat >> "$COMPOSE_FILE" << EOF
    healthcheck:
      test: ["CMD-SHELL", "socat -t 1 STDIN UNIX-CONNECT:/run/armorclaw/bridge.sock || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

volumes:
  armorclaw-config:
  armorclaw-keystore:
EOF

    echo -e "${GREEN}Generated: $COMPOSE_FILE${NC}"
    echo ""
    echo -e "${CYAN}Deploy with:${NC}"
    echo "  docker compose -f $COMPOSE_FILE up -d"
    echo ""
    echo -e "${CYAN}Customize with environment variables:${NC}"
    echo "  export ARMORCLAW_API_KEY=sk-your-key"
    echo "  export ARMORCLAW_ADMIN_PASSWORD=\$(openssl rand -base64 24)"
    echo "  $0 --bootstrap"
    echo ""
}

#=============================================================================
# Parse Arguments
#=============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        --bootstrap)
            BOOTSTRAP_MODE=true
            shift
            ;;
        --bridge-only)
            BRIDGE_ONLY=true
            shift
            ;;
        --no-start)
            NO_START=true
            shift
            ;;
        --ports)
            SHOW_PORTS=true
            shift
            ;;
        --help|-h)
            echo "ArmorClaw Quick Install v0.4.0"
            echo ""
            echo "Usage:"
            echo "  # Interactive (requires TTY):"
            echo "  bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh)\""
            echo ""
            echo "  # Non-interactive (CI/CD, no TTY required):"
            echo "  export ARMORCLAW_API_KEY=sk-your-key"
            echo "  curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash"
            echo ""
            echo "Options:"
            echo "  --bootstrap      Generate docker-compose.yml (no container start)"
            echo "  --bridge-only    Run bridge only, no Matrix (for testing)"
            echo "  --no-start       Pull image but don't start container"
            echo "  --ports          Show auto-detected ports"
            echo "  --help           Show this help"
            echo ""
            echo "Environment Variables (for non-interactive mode):"
            echo "  ARMORCLAW_API_KEY        AI provider API key (required for non-interactive)"
            echo "  ARMORCLAW_API_BASE_URL   Custom API endpoint (optional)"
            echo "  ARMORCLAW_PROFILE        Deployment profile: quick (default) or enterprise"
            echo "  ARMORCLAW_ADMIN_PASSWORD Admin password (auto-generated if not set)"
            echo "  ARMORCLAW_SERVER_NAME    Server hostname or IP (auto-detected if not set)"
            echo ""
            echo "Examples:"
            echo "  # Interactive with TTY"
            echo "  bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh)\""
            echo ""
            echo "  # Non-interactive (CI/CD)"
            echo "  export ARMORCLAW_API_KEY=sk-your-key"
            echo "  curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash"
            echo ""
            echo "  # Generate compose file for customization"
            echo "  curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash -s -- --bootstrap"
            echo ""
            echo "  # Bridge-only for testing"
            echo "  curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash -s -- --bridge-only"
            echo ""
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

#=============================================================================
# Main
#=============================================================================

echo -e "${CYAN}"
echo "╔══════════════════════════════════════════════════════╗"
echo "║        ${BOLD}ArmorClaw Quick Install${NC}${CYAN}  v0.4.0              ║"
echo "╚══════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check root/sudo
SUDO=""
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: Not running as root. Using sudo for system directories.${NC}"
    SUDO="sudo"
fi

# Check Docker
echo -e "${CYAN}Checking prerequisites...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}ERROR: Docker is not installed.${NC}"
    echo ""
    echo "Install Docker:"
    echo "  curl -fsSL https://get.docker.com | sh"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo -e "${RED}ERROR: Docker daemon is not running.${NC}"
    echo ""
    echo "Start Docker:"
    echo "  sudo systemctl start docker"
    exit 1
fi

echo -e "${GREEN}Docker is running${NC}"

# Port detection
auto_detect_ports

# Handle --ports only
if [ "$SHOW_PORTS" = true ]; then
    exit 0
fi

# Bootstrap mode
if [ "$BOOTSTRAP_MODE" = true ]; then
    generate_compose
    exit 0
fi

# Stop existing container
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${YELLOW}Removing existing container...${NC}"
    docker rm -f ${CONTAINER_NAME} 2>/dev/null || true
fi

# Clean up previous failed installs (fresh start)
echo -e "${CYAN}Cleaning up previous installation state...${NC}"
$SUDO rm -f ${CONFIG_DIR}/.setup_complete 2>/dev/null || true
$SUDO rm -f ${DATA_DIR}/keystore.db 2>/dev/null || true
$SUDO rm -f ${DATA_DIR}/keystore.db-shm 2>/dev/null || true
$SUDO rm -f ${DATA_DIR}/keystore.db-wal 2>/dev/null || true
$SUDO rm -f ${DATA_DIR}/.api_key_temp 2>/dev/null || true
$SUDO rm -f ${DATA_DIR}/.admin_password 2>/dev/null || true
$SUDO rm -f ${DATA_DIR}/.admin_user 2>/dev/null || true
# Also clean up Matrix config from previous runs
$SUDO rm -rf /tmp/armorclaw-configs 2>/dev/null || true

# Pull image
echo -e "${CYAN}Pulling image: ${IMAGE}${NC}"
docker pull ${IMAGE}
echo -e "${GREEN}Image pulled${NC}"

# Create directories
echo -e "${CYAN}Creating directories...${NC}"
$SUDO mkdir -p ${CONFIG_DIR}
$SUDO mkdir -p ${DATA_DIR}

# Detect server IP for bridge-only mode too
DETECTED_SERVER_IP=$(ip route get 1 2>/dev/null | awk '{print $7; exit}' || hostname -I 2>/dev/null | awk '{print $1}')
if [ -z "$DETECTED_SERVER_IP" ]; then
    DETECTED_SERVER_IP="localhost"
fi

# Bridge-only mode
if [ "$BRIDGE_ONLY" = true ]; then
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║        ${BOLD}Bridge-Only Mode (No Matrix)${NC}${CYAN}                 ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo -e "${DIM}Server IP: ${DETECTED_SERVER_IP}${NC}"
    echo ""

    $SUDO tee ${CONFIG_DIR}/config.toml > /dev/null << EOF
# ArmorClaw Bridge Configuration
# Generated by install.sh (Matrix disabled)

socket_path = "/run/armorclaw/bridge.sock"
db_path = "/var/lib/armorclaw/keystore.db"
log_level = "info"

[server]
socket_path = "/run/armorclaw/bridge.sock"

[keystore]
db_path = "/var/lib/armorclaw/keystore.db"

[matrix]
enabled = false
homeserver_url = ""

[logging]
level = "info"
format = "text"
output = "stdout"
EOF

    $SUDO touch ${CONFIG_DIR}/.setup_complete
    echo -e "${GREEN}Configuration created${NC}"

    if [ "$NO_START" = true ]; then
        echo -e "${YELLOW}--no-start specified. Container not started.${NC}"
        exit 0
    fi

    echo -e "${CYAN}Starting ArmorClaw Bridge...${NC}"

    # Build environment args (bridge-only still needs server name)
    # Only pass env vars when ARMORCLAW_API_KEY is set (non-interactive mode)
    ENV_ARGS=""
    if [ -n "${ARMORCLAW_API_KEY:-}" ]; then
        ENV_ARGS="-e ARMORCLAW_SERVER_NAME=${DETECTED_SERVER_IP}"
        ENV_ARGS="$ENV_ARGS -e ARMORCLAW_API_KEY=${ARMORCLAW_API_KEY}"
        [ -n "${ARMORCLAW_ADMIN_PASSWORD:-}" ] && ENV_ARGS="$ENV_ARGS -e ARMORCLAW_ADMIN_PASSWORD=${ARMORCLAW_ADMIN_PASSWORD}"
    fi

    docker run -d --name ${CONTAINER_NAME} \
        --restart unless-stopped \
        --user root \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v ${CONFIG_DIR}:/etc/armorclaw \
        -v ${DATA_DIR}:/var/lib/armorclaw \
        ${ENV_ARGS} \
        -p ${BRIDGE_PORT}:8443 \
        ${IMAGE}

    sleep 3

    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo ""
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}ArmorClaw Bridge is running!${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        echo "Container: ${CONTAINER_NAME}"
        echo "Config:    ${CONFIG_DIR}/config.toml"
        echo "RPC:       http://localhost:${BRIDGE_PORT}"
        echo ""
        echo -e "${CYAN}Commands:${NC}"
        echo "  Logs:    docker logs -f ${CONTAINER_NAME}"
        echo "  Stop:    docker stop ${CONTAINER_NAME}"
        echo "  Remove:  docker rm -f ${CONTAINER_NAME}"
        echo ""
    else
        echo -e "${RED}Container failed to start. Check logs:${NC}"
        echo "  docker logs ${CONTAINER_NAME}"
        exit 1
    fi
    exit 0
fi

# Full stack mode (default)
echo -e "${CYAN}"
echo "╔══════════════════════════════════════════════════════╗"
echo "║        ${BOLD}Full Stack Mode (With Matrix)${NC}${CYAN}               ║"
echo "╚══════════════════════════════════════════════════════╝"
echo -e "${NC}"
echo "Starting ArmorClaw with Matrix integration..."
echo "The setup wizard will guide you through configuration."
echo ""

# Detect server IP on host (more reliable than container-side detection)
DETECTED_SERVER_IP=$(ip route get 1 2>/dev/null | awk '{print $7; exit}' || hostname -I 2>/dev/null | awk '{print $1}')
if [ -z "$DETECTED_SERVER_IP" ]; then
    DETECTED_SERVER_IP="localhost"
fi
echo -e "${DIM}Detected server IP: ${DETECTED_SERVER_IP}${NC}"
echo ""

if [ "$NO_START" = true ]; then
    echo -e "${YELLOW}--no-start specified. Container not started.${NC}"
    echo ""
    # Build env args for display
    ENV_ARGS="-e ARMORCLAW_SERVER_NAME=${DETECTED_SERVER_IP}"
    [ -n "${ARMORCLAW_API_KEY:-}" ] && ENV_ARGS="$ENV_ARGS -e ARMORCLAW_API_KEY=***"
    [ -n "${ARMORCLAW_API_BASE_URL:-}" ] && ENV_ARGS="$ENV_ARGS -e ARMORCLAW_API_BASE_URL=${ARMORCLAW_API_BASE_URL}"
    [ -n "${ARMORCLAW_PROFILE:-}" ] && ENV_ARGS="$ENV_ARGS -e ARMORCLAW_PROFILE=${ARMORCLAW_PROFILE}"
    [ -n "${ARMORCLAW_ADMIN_PASSWORD:-}" ] && ENV_ARGS="$ENV_ARGS -e ARMORCLAW_ADMIN_PASSWORD=***"
    echo "To start:"
    echo "  docker run -it --name ${CONTAINER_NAME} \\"
    echo "    --restart unless-stopped \\"
    echo "    --user root \\"
    echo "    -v /var/run/docker.sock:/var/run/docker.sock \\"
    echo "    -v ${CONFIG_DIR}:/etc/armorclaw \\"
    echo "    -v ${DATA_DIR}:/var/lib/armorclaw \\"
    echo "    ${ENV_ARGS} \\"
    echo "    -p ${BRIDGE_PORT}:8443 -p ${MATRIX_PORT}:6167 -p ${PUSH_PORT}:5000 \\"
    echo "    ${IMAGE}"
    exit 0
fi

# Build environment args
ENV_ARGS=$(build_env_args)

# Determine run mode based on TTY availability and environment
DOCKER_FLAGS=""
if [ -n "${ARMORCLAW_API_KEY:-}" ]; then
    # Non-interactive mode (API key provided) - run detached
    DOCKER_FLAGS="-d"
elif [ -t 0 ]; then
    # Interactive mode with TTY - run with interactive terminal
    DOCKER_FLAGS="-it"
else
    # No TTY and no API key - show error with instructions
    echo -e "${RED}ERROR: No TTY detected and no API key provided.${NC}"
    echo ""
    echo "To run in non-interactive mode, set environment variables:"
    echo ""
    echo "  export ARMORCLAW_API_KEY=sk-your-key"
    echo "  curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash"
    echo ""
    echo "Or run with a TTY:"
    echo ""
    echo "  bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh)\""
    echo ""
    exit 1
fi

docker run ${DOCKER_FLAGS} --name ${CONTAINER_NAME} \
    --restart unless-stopped \
    --user root \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v ${CONFIG_DIR}:/etc/armorclaw \
    -v ${DATA_DIR}:/var/lib/armorclaw \
    ${ENV_ARGS} \
    -p ${BRIDGE_PORT}:8443 \
    -p ${MATRIX_PORT}:6167 \
    -p ${PUSH_PORT}:5000 \
    ${IMAGE}
