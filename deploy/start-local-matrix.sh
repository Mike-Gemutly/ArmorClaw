#!/bin/bash
# ArmorClaw Local Development - Matrix Stack
# Starts local Matrix, Caddy, and Bridge for development/testing

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     ArmorClaw - Local Development Stack             ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Install Docker from https://docs.docker.com/get-docker/"
    exit 1
fi

# Check docker-compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}Error: docker-compose is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker found${NC}"

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo -e "${RED}Error: Docker is not running${NC}"
    echo "Start Docker Desktop or the Docker daemon"
    exit 1
fi

echo -e "${GREEN}✓ Docker is running${NC}"

# Check ports
echo ""
echo "Checking ports..."
for port in 80 443 6167 8448; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠ Port $port is already in use${NC}"
        echo "  Stop the service using port $port or modify docker-compose-stack.yml"
    else
        echo -e "${GREEN}✓ Port $port is available${NC}"
    fi
done

# Check /etc/hosts
echo ""
echo "Checking /etc/hosts configuration..."
if grep -q "matrix.local" /etc/hosts 2>/dev/null; then
    echo -e "${GREEN}✓ matrix.local found in /etc/hosts${NC}"
else
    echo -e "${YELLOW}⚠ matrix.local not found in /etc/hosts${NC}"
    echo ""
    echo "Add this line to /etc/hosts:"
    echo "  127.0.0.1 matrix.local"
    echo ""
    echo "On macOS/Linux:"
    echo "  sudo nano /etc/hosts"
    echo ""
    echo "On Windows:"
    echo "  notepad C:\\Windows\\System32\\drivers\\etc\\hosts"
    echo ""
    read -p "Press Enter to continue (you'll need to add this later)..."
fi

# Create .env if it doesn't exist
if [ ! -f .env ]; then
    echo ""
    echo -e "${YELLOW}Creating .env file for local development...${NC}"

    # Generate passwords
    ADMIN_PASS=$(openssl rand -base64 16 | tr -d '/+=')
    BRIDGE_PASS=$(openssl rand -base64 16 | tr -d '/+=')

    cat > .env <<EOF
# Local Development Configuration
MATRIX_DOMAIN=matrix.local
MATRIX_ADMIN_USER=admin
MATRIX_ADMIN_PASSWORD=${ADMIN_PASS}
MATRIX_BRIDGE_USER=bridge
MATRIX_BRIDGE_PASSWORD=${BRIDGE_PASS}
ROOM_NAME=ArmorClaw Agents
ROOM_ALIAS=agents
EOF

    echo -e "${GREEN}✓ .env file created${NC}"
else
    echo -e "${GREEN}✓ .env file found${NC}"
fi

echo ""
echo "Configuration:"
echo "  Domain: matrix.local"
echo "  Admin User: admin"
echo "  Room: #agents:matrix.local"
echo ""

read -p "Start local development stack? (Y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Nn]$ ]]; then
    echo "Cancelled"
    exit 0
fi

echo ""
echo -e "${BLUE}Starting services...${NC}"

# Stop existing services
echo "Stopping any existing services..."
docker-compose -f docker-compose-stack.yml down 2>/dev/null || true

# Start services
echo ""
echo "1. Starting Matrix Conduit..."
docker-compose -f docker-compose-stack.yml up -d matrix

echo "2. Waiting for Matrix to be ready..."
sleep 10

echo "3. Starting Caddy (reverse proxy)..."
docker-compose -f docker-compose-stack.yml up -d caddy

echo "4. Waiting for Caddy to start..."
sleep 5

echo "5. Starting ArmorClaw Bridge..."
docker-compose -f docker-compose-stack.yml up -d bridge

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Local Development Stack Started!                  ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Load credentials
export $(cat .env | grep -v '^#' | xargs)

echo -e "${BLUE}Services Running:${NC}"
echo "  Matrix Conduit: http://localhost:6167"
echo "  Caddy Proxy: http://localhost"
echo "  ArmorClaw Bridge: Running in container"
echo ""

echo -e "${BLUE}Element X Connection Details:${NC}"
echo ""
echo "  Homeserver URL:"
echo "    http://matrix.local"
echo "    (or https://matrix.local if SSL is configured)"
echo ""
echo "  Username:"
echo "    ${MATRIX_ADMIN_USER}"
echo ""
echo "  Password:"
echo "    ${MATRIX_ADMIN_PASSWORD}"
echo ""
echo "  Room to Join:"
echo "    #${ROOM_ALIAS}:${MATRIX_DOMAIN}"
echo ""

echo -e "${BLUE}Steps to Connect:${NC}"
echo "  1. Open Element X app"
echo "  2. Tap 'Login'"
echo "  3. Enter homeserver: http://matrix.local"
echo "  4. Enter username: ${MATRIX_ADMIN_USER}"
echo "  5. Enter password: (see above)"
echo "  6. Tap 'Log in'"
echo "  7. Join room: #agents:matrix.local"
echo ""

echo -e "${BLUE}Testing Commands:${NC}"
echo ""
echo "Check services:"
echo "  docker-compose -f docker-compose-stack.yml ps"
echo ""
echo "View logs:"
echo "  docker-compose -f docker-compose-stack.yml logs -f"
echo ""
echo "View specific service logs:"
echo "  docker-compose -f docker-compose-stack.yml logs -f matrix"
echo "  docker-compose -f docker-compose-stack.yml logs -f bridge"
echo ""

echo -e "${YELLOW}Note: First startup may take 1-2 minutes for all services to be ready.${NC}"
echo ""

echo -e "${BLUE}To stop services:${NC}"
echo "  docker-compose -f docker-compose-stack.yml down"
echo ""

echo -e "${BLUE}To restart services:${NC}"
echo "  ./deploy/start-local-matrix.sh"
echo ""

echo -e "${GREEN}Happy developing! 🚀${NC}"
echo ""
