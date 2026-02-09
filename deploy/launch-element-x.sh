#!/bin/bash
# ArmorClaw Quick Launch - Element X Ready
# One-command deployment with auto-provisioning

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     ArmorClaw - Element X Quick Launch               ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}No .env file found. Creating from template...${NC}"
    
    # Get domain from user
    read -p "Enter your domain (or 'localhost' for testing): " DOMAIN
    DOMAIN=${DOMAIN:-localhost}
    
    # Generate passwords
    ADMIN_PASS=$(openssl rand -base64 16 | tr -d '/+=')
    BRIDGE_PASS=$(openssl rand -base64 16 | tr -d '/+=')
    
    cat > .env <<EOF
MATRIX_DOMAIN=$DOMAIN
MATRIX_ADMIN_USER=admin
MATRIX_ADMIN_PASSWORD=$ADMIN_PASS
MATRIX_BRIDGE_USER=bridge
MATRIX_BRIDGE_PASSWORD=$BRIDGE_PASS
ROOM_NAME=ArmorClaw Agents
ROOM_ALIAS=agents
EOF
    
    echo -e "${GREEN}.env file created with secure passwords${NC}"
fi

# Load environment
export $(cat .env | grep -v '^#' | xargs)

echo ""
echo "Configuration:"
echo "  Domain: ${MATRIX_DOMAIN}"
echo "  Admin User: ${MATRIX_ADMIN_USER}"
echo "  Room: ${ROOM_NAME}"
echo ""

read -p "Continue with deployment? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 0
fi

echo -e "${BLUE}Starting services...${NC}"

# Stop existing services if running
docker-compose -f docker-compose-stack.yml down 2>/dev/null || true

# Start services
echo "1. Starting Matrix, Caddy, and Bridge..."
docker-compose -f docker-compose-stack.yml up -d

echo ""
echo "2. Waiting for services to be ready..."
sleep 10

echo ""
echo "3. Running provisioning (creates users and room)..."
docker-compose -f docker-compose-stack.yml run --rm provision

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║           Deployment Complete!                         ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Element X Connection Details:${NC}"
echo ""
echo "  Homeserver URL:"
echo "    ${MATRIX_DOMAIN}"
echo "    https://${MATRIX_DOMAIN}"
echo ""
echo "  Username:"
echo "    ${MATRIX_ADMIN_USER}"
echo ""
echo "  Password:"
echo "    ${MATRIX_ADMIN_PASSWORD}"
echo ""
echo "  Room to Join:"
echo "    #${ROOM_ALIAS}:${MATRIX_DOMAIN}"
echo "    ${ROOM_NAME}"
echo ""
echo -e "${BLUE}Steps to Connect:${NC}"
echo "  1. Open Element X app"
echo "  2. Tap 'Login'"
echo "  3. Enter homeserver: https://${MATRIX_DOMAIN}"
echo "  4. Enter username: ${MATRIX_ADMIN_USER}"
echo "  5. Enter password: (see above)"
echo "  6. Join room: #${ROOM_ALIAS}:${MATRIX_DOMAIN}"
echo ""
echo -e "${YELLOW}Note: SSL certificates may take 1-2 minutes to provision.${NC}"
echo ""
echo "View logs:"
echo "  docker-compose -f docker-compose-stack.yml logs -f"
echo ""
