#!/bin/bash
# ArmorClaw Hostinger VPS Deployment Script (Docker Hub)
# Automated deployment for Hostinger VPS

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     ArmorClaw Hostinger VPS Deployment               ║${NC}"
echo -e "${BLUE}╚═════════════════════════════════════════════════════════╝${NC}"
echo ""

# Configuration
DEPLOY_DIR="/opt/armorclaw"
DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
MATRIX_DOMAIN="${MATRIX_DOMAIN:-}"

# Function to prompt for input
prompt_input() {
    local prompt_text=$1
    local var_name=$2
    local default_value=$3
    
    if [[ -n "$default_value" ]]; then
        read -p "$prompt_text [$default_value]: " input
        eval "$var_name=\${input:-$default_value}"
    else
        read -p "$prompt_text: " input
        eval "$var_name=$input"
    fi
}

# Check running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    echo "Please run: sudo $0"
    exit 1
fi

echo -e "${BLUE}Configuration:${NC}"
prompt_input "Docker Hub Username" DOCKERHUB_USERNAME
prompt_input "Image Tag" IMAGE_TAG "latest"
prompt_input "Matrix Domain" MATRIX_DOMAIN

if [[ -z "$DOCKERHUB_USERNAME" ]] || [[ -z "$MATRIX_DOMAIN" ]]; then
    echo -e "${RED}Error: Docker Hub username and Matrix domain are required${NC}"
    exit 1
fi

FULL_IMAGE="${DOCKERHUB_USERNAME}/agent:${IMAGE_TAG}"
echo ""
echo "  Docker Hub Image: ${FULL_IMAGE}"
echo "  Matrix Domain: ${MATRIX_DOMAIN}"
echo "  Deploy Directory: ${DEPLOY_DIR}"
echo ""

# Pre-flight checks
echo -e "${YELLOW}Pre-flight checks...${NC}"

# Check Docker
if ! command -v docker &>/dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Please install Docker first"
    exit 1
fi

# Check Docker Compose
if ! command -v docker compose &>/dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed${NC}"
    echo "Please install docker-compose-plugin"
    exit 1
fi

# Check disk space
DISK_AVAILABLE=$(df -BG / | awk 'NR==2 {print $4}')
if [[ $DISK_AVAILABLE -lt 5 ]]; then
    echo -e "${RED}Error: Insufficient disk space (< 5GB)${NC}"
    exit 1
fi

# Check memory
TOTAL_MEM=$(free -g | awk 'NR==2 {print $2}')
if [[ $TOTAL_MEM -lt 2 ]]; then
    echo -e "${YELLOW}Warning: Less than 2GB RAM available${NC}"
    echo "ArmorClaw may run slowly"
fi

echo -e "${GREEN}✅ Pre-flight checks passed${NC}"
echo ""

# Create deployment directory
echo -e "${YELLOW}Step 1: Creating deployment directory...${NC}"
mkdir -p "${DEPLOY_DIR}"/{configs,logs}
echo -e "${GREEN}✅ Directory created: ${DEPLOY_DIR}${NC}"
echo ""

# Pull Docker image
echo -e "${YELLOW}Step 2: Pulling Docker image...${NC}"
docker pull "${FULL_IMAGE}"
echo -e "${GREEN}✅ Image pulled${NC}"
echo ""

# Create docker-compose.yml
echo -e "${YELLOW}Step 3: Creating docker-compose.yml...${NC}"
cat > "${DEPLOY_DIR}/docker-compose.yml" <<EOF
version: "3.8"

services:
  matrix:
    image: matrixconduit/matrix-conduit:latest
    container_name: armorclaw-matrix
    restart: unless-stopped
    
    environment:
      CONDUIT_SERVER_NAME: "${MATRIX_DOMAIN}"
      CONDUIT_ADDRESS: "0.0.0.0"
      CONDUIT_PORT: "6167"
      CONDUIT_DATABASE_BACKEND: "sqlite"
      CONDUIT_ALLOW_ENCRYPTION: "true"
      CONDUIT_ALLOW_REGISTRATION: "false"
      CONDUIT_ALLOW_FEDERATION: "true"
      CONDUIT_MAX_REQUEST_SIZE: "10485760"
      CONDUIT_LOG: "info"
    
    volumes:
      - matrix_data:/var/lib/matrix-conduit
    
    ports:
      - "6167:6167"
      - "8448:8448"
    
    networks:
      - armorclaw-net
    
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:6167/_matrix/client/versions"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
  
  caddy:
    image: caddy:2-alpine
    container_name: armorclaw-caddy
    restart: unless-stopped
    
    ports:
      - "80:80"
      - "443:443"
    
    volumes:
      - ${DEPLOY_DIR}/configs/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    
    networks:
      - armorclaw-net
    
    depends_on:
      matrix:
        condition: service_healthy
  
  agent:
    image: "${FULL_IMAGE}"
    container_name: armorclaw-agent
    restart: unless-stopped
    
    environment:
      ARMORCLAW_SECRETS_PATH: "/run/secrets"
      ARMORCLAW_SECRETS_FD: "3"
      PYTHONUNBUFFERED: "1"
      ARMORCLAW_MATRIX_HOMESERVER: "https://${MATRIX_DOMAIN}"
    
    volumes:
      - ${DEPLOY_DIR}/configs/secrets:/run/secrets:ro
      - /tmp:/tmp
    
    networks:
      - armorclaw-net
    
    depends_on:
      - caddy

networks:
  armorclaw-net:
    driver: bridge

volumes:
  matrix_data:
  caddy_data:
  caddy_config:
EOF

echo -e "${GREEN}✅ docker-compose.yml created${NC}"
echo ""

# Create Caddyfile
echo -e "${YELLOW}Step 4: Creating Caddyfile...${NC}"
cat > "${DEPLOY_DIR}/configs/Caddyfile" <<EOF
{
    email admin@${MATRIX_DOMAIN}
}

${MATRIX_DOMAIN} {
    reverse_proxy matrix:6167
}

chat.${MATRIX_DOMAIN} {
    reverse_proxy matrix:8448
}
EOF

echo -e "${GREEN}✅ Caddyfile created${NC}"
echo ""

# Deploy stack
echo -e "${YELLOW}Step 5: Deploying stack...${NC}"
cd "${DEPLOY_DIR}"
docker compose up -d

echo -e "${GREEN}✅ Stack deployed${NC}"
echo ""

# Wait for containers to start
echo -e "${YELLOW}Waiting for containers to start...${NC}"
sleep 10

# Check status
echo -e "${YELLOW}Checking container status...${NC}"
docker compose ps

echo ""

# Display logs
echo -e "${YELLOW}Recent logs:${NC}"
docker compose logs --tail=20

echo ""
echo -e "${BLUE}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Deployment Complete!                           ║${NC}"
echo -e "${BLUE}╚═════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Next steps:"
echo "  1. Configure DNS for ${MATRIX_DOMAIN}"
echo "  2. Update firewall rules (ports 80, 443, 8448, 6167)"
echo "  3. Create Matrix admin user"
echo "  4. Connect via Element X"
echo ""
echo "Commands:"
echo "  View logs:     cd ${DEPLOY_DIR} && docker compose logs -f"
echo "  Stop stack:    docker compose down"
echo "  Restart stack: docker compose restart"
echo "  Update stack:  docker compose pull && docker compose up -d"
echo ""
