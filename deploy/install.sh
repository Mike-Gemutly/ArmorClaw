#!/bin/bash
# ArmorClaw Quick Install Script
# Run this on your VPS to install and start ArmorClaw
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
#
# Or with options:
#   curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash -s -- --with-matrix
#
# Options:
#   --with-matrix    Enable Matrix integration (for ArmorChat)
#   --no-start       Install but don't start the container
#   --help           Show this help

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# Defaults
WITH_MATRIX=false
NO_START=false
CONTAINER_NAME="armorclaw"
IMAGE="mikegemut/armorclaw:latest"
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --with-matrix)
            WITH_MATRIX=true
            shift
            ;;
        --no-start)
            NO_START=true
            shift
            ;;
        --help|-h)
            echo "ArmorClaw Quick Install Script"
            echo ""
            echo "Usage:"
            echo "  curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash"
            echo ""
            echo "Options:"
            echo "  --with-matrix    Enable Matrix integration (for ArmorChat)"
            echo "  --no-start       Install but don't start the container"
            echo "  --help           Show this help"
            echo ""
            echo "Examples:"
            echo "  # Bridge-only mode (default, fastest):"
            echo "  curl -fsSL ... | bash"
            echo ""
            echo "  # With Matrix (for ArmorChat):"
            echo "  curl -fsSL ... | bash -s -- --with-matrix"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${CYAN}"
echo "╔══════════════════════════════════════════════════════╗"
echo "║        ${BOLD}ArmorClaw Quick Install${NC}${CYAN}                        ║"
echo "╚══════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: Not running as root. Some operations may require sudo.${NC}"
    SUDO="sudo"
else
    SUDO=""
fi

# Check for Docker
echo -e "${CYAN}Checking prerequisites...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}ERROR: Docker is not installed.${NC}"
    echo ""
    echo "Install Docker first:"
    echo "  curl -fsSL https://get.docker.com | sh"
    echo ""
    echo "Then re-run this script."
    exit 1
fi

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo -e "${RED}ERROR: Docker daemon is not running.${NC}"
    echo ""
    echo "Start Docker:"
    echo "  sudo systemctl start docker"
    exit 1
fi

echo -e "${GREEN}✓ Docker is running${NC}"

# Stop and remove existing container if present
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${YELLOW}Stopping existing container...${NC}"
    docker rm -f ${CONTAINER_NAME} 2>/dev/null || true
fi

# Pull the latest image
echo -e "${CYAN}Pulling latest image: ${IMAGE}${NC}"
docker pull ${IMAGE}
echo -e "${GREEN}✓ Image pulled${NC}"

# Create directories
echo -e "${CYAN}Creating directories...${NC}"
$SUDO mkdir -p ${CONFIG_DIR}
$SUDO mkdir -p ${DATA_DIR}

if [ "$WITH_MATRIX" = true ]; then
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║        ${BOLD}Full Stack Mode (With Matrix)${NC}${CYAN}               ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo "Starting ArmorClaw with Matrix integration..."
    echo "The setup wizard will guide you through configuration."
    echo ""

    if [ "$NO_START" = true ]; then
        echo -e "${YELLOW}--no-start specified. Container not started.${NC}"
        echo ""
        echo "To start manually:"
        echo "  docker run -it --name ${CONTAINER_NAME} \\"
        echo "    --restart unless-stopped \\"
        echo "    --user root \\"
        echo "    -v /var/run/docker.sock:/var/run/docker.sock \\"
        echo "    -v ${CONFIG_DIR}:/etc/armorclaw \\"
        echo "    -v ${DATA_DIR}:/var/lib/armorclaw \\"
        echo "    -p 8443:8443 -p 6167:6167 -p 5000:5000 \\"
        echo "    ${IMAGE}"
        exit 0
    fi

    docker run -it --name ${CONTAINER_NAME} \
        --restart unless-stopped \
        --user root \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v ${CONFIG_DIR}:/etc/armorclaw \
        -v ${DATA_DIR}:/var/lib/armorclaw \
        -p 8443:8443 \
        -p 6167:6167 \
        -p 5000:5000 \
        ${IMAGE}
else
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║        ${BOLD}Bridge-Only Mode (No Matrix)${NC}${CYAN}                 ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    # Create minimal config with Matrix disabled
    echo -e "${CYAN}Creating configuration...${NC}"
    $SUDO tee ${CONFIG_DIR}/config.toml > /dev/null << 'EOF'
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

    # Create setup flag to skip wizard
    $SUDO touch ${CONFIG_DIR}/.setup_complete

    echo -e "${GREEN}✓ Configuration created${NC}"

    if [ "$NO_START" = true ]; then
        echo -e "${YELLOW}--no-start specified. Container not started.${NC}"
        echo ""
        echo "To start manually:"
        echo "  docker run -d --name ${CONTAINER_NAME} \\"
        echo "    --restart unless-stopped \\"
        echo "    --user root \\"
        echo "    -v /var/run/docker.sock:/var/run/docker.sock \\"
        echo "    -v ${CONFIG_DIR}:/etc/armorclaw \\"
        echo "    -v ${DATA_DIR}:/var/lib/armorclaw \\"
        echo "    -p 8443:8443 \\"
        echo "    ${IMAGE}"
        exit 0
    fi

    echo -e "${CYAN}Starting ArmorClaw Bridge...${NC}"
    docker run -d --name ${CONTAINER_NAME} \
        --restart unless-stopped \
        --user root \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v ${CONFIG_DIR}:/etc/armorclaw \
        -v ${DATA_DIR}:/var/lib/armorclaw \
        -p 8443:8443 \
        ${IMAGE}

    # Wait for bridge to start
    echo -e "${CYAN}Waiting for bridge to initialize...${NC}"
    sleep 3

    # Check if container is running
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo ""
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}✓ ArmorClaw Bridge is running!${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        echo "Container: ${CONTAINER_NAME}"
        echo "Config:    ${CONFIG_DIR}/config.toml"
        echo "Data:      ${DATA_DIR}"
        echo ""
        echo -e "${CYAN}Useful commands:${NC}"
        echo "  View logs:    docker logs -f ${CONTAINER_NAME}"
        echo "  Stop:         docker stop ${CONTAINER_NAME}"
        echo "  Restart:      docker restart ${CONTAINER_NAME}"
        echo "  Remove:       docker rm -f ${CONTAINER_NAME}"
        echo ""
        echo -e "${CYAN}Test RPC:${NC}"
        echo "  docker exec ${CONTAINER_NAME} bash -c \"echo '{\\\"jsonrpc\\\":\\\"2.0\\\",\\\"id\\\":1,\\\"method\\\":\\\"status\\\"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock\""
        echo ""
    else
        echo -e "${RED}ERROR: Container failed to start.${NC}"
        echo ""
        echo "Check logs:"
        echo "  docker logs ${CONTAINER_NAME}"
        exit 1
    fi
fi
