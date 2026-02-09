#!/bin/bash
#
# ArmorClaw Infrastructure Deployment Script
# Deploys Matrix Conduit + Nginx + Coturn on Hostinger KVM2
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="${COMPOSE_PROJECT_NAME:-armorclaw}"
DOMAIN="${MATRIX_SERVER_NAME:-matrix.armorclaw.com}"
EMAIL="${SSL_EMAIL:-admin@armorclaw.com}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}ArmorClaw Infrastructure Deployment${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Install with: sudo apt install docker.io"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: docker-compose is not installed${NC}"
    echo "Install with: sudo apt install docker-compose"
    exit 1
fi

echo -e "${GREEN}✓ Docker and docker-compose found${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: Running without sudo. Some operations may fail.${NC}"
fi

# Create necessary directories
echo ""
echo -e "${YELLOW}Creating directory structure...${NC}"

mkdir -p configs/nginx/ssl
mkdir -p configs/nginx/conf.d
mkdir -p conduit_data
mkdir -p coturn_data
mkdir -p /var/www/certbot

echo -e "${GREEN}✓ Directories created${NC}"

# Load environment variables
if [ -f .env ]; then
    echo -e "${YELLOW}Loading environment variables from .env...${NC}"
    export $(cat .env | grep -v '^#' | xargs)
else
    echo -e "${YELLOW}Warning: .env file not found. Using defaults.${NC}"
    echo -e "${YELLOW}Copy .env.example to .env and configure your values.${NC}"
fi

# Validate configuration
echo ""
echo -e "${YELLOW}Validating configuration...${NC}"
echo "Domain: $DOMAIN"
echo "Email: $EMAIL"

# Generate TURN secret if not set
if [ -z "$TURN_SECRET" ]; then
    TURN_SECRET=$(openssl rand -hex 32)
    echo -e "${YELLOW}Generated TURN secret: ${TURN_SECRET}${NC}"
    echo "Add this to your .env: TURN_SECRET=$TURN_SECRET"
fi

# Validate domain resolves to this server
PUBLIC_IP=$(curl -s https://api.ipify.org)
DOMAIN_IP=$(dig +short $DOMAIN | grep -E '^[0-9.')

if [ "$DOMAIN_IP" != "$PUBLIC_IP" ]; then
    echo -e "${YELLOW}Warning: Domain $DOMAIN resolves to $DOMAIN_IP, but this server's IP is $PUBLIC_IP${NC}"
    echo "Ensure DNS A record is configured correctly."
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo -e "${GREEN}✓ DNS configuration validated${NC}"
fi

# Stop existing containers if running
echo ""
echo -e "${YELLOW}Stopping existing containers...${NC}"
docker-compose down 2>/dev/null || true

# Pull latest images
echo ""
echo -e "${YELLOW}Pulling Docker images...${NC}"
docker-compose pull

# Start services without SSL first
echo ""
echo -e "${YELLOW}Starting services (HTTP only)...${NC}"
docker-compose up -d matrix-conduit coturn nginx

# Wait for Conduit to be ready
echo ""
echo -e "${YELLOW}Waiting for Matrix Conduit to start...${NC}"
for i in {1..30}; do
    if curl -f http://localhost:6167/_matrix/client/versions &> /dev/null; then
        echo -e "${GREEN}✓ Matrix Conduit is ready${NC}"
        break
    fi
    echo "Waiting... ($i/30)"
    sleep 2
done

# Check if we should get SSL certificate
echo ""
echo -e "${YELLOW}SSL Certificate Setup${NC}"
read -p "Obtain SSL certificate from Let's Encrypt? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Install certbot if not present
    if ! command -v certbot &> /dev/null; then
        echo "Installing certbot..."
        sudo apt install certbot python3-certbot-nginx -y
    fi

    # Get certificate
    echo "Obtaining SSL certificate for $DOMAIN..."
    sudo certbot certonly --standalone \
        -d $DOMAIN \
        --email $EMAIL \
        --agree-tos \
        --non-interactive

    # Copy certificates to nginx directory
    sudo cp /etc/letsencrypt/live/$DOMAIN/fullchain.pem configs/nginx/ssl/
    sudo cp /etc/letsencrypt/live/$DOMAIN/privkey.pem configs/nginx/ssl/
    sudo cp /etc/letsencrypt/live/$DOMAIN/chain.pem configs/nginx/ssl/

    # Set permissions
    sudo chown $USER:$USER configs/nginx/ssl/*.pem
    sudo chmod 644 configs/nginx/ssl/*.pem

    echo -e "${GREEN}✓ SSL certificates installed${NC}"

    # Restart nginx with SSL
    echo ""
    echo -e "${YELLOW}Restarting with SSL enabled...${NC}"
    docker-compose restart nginx
else
    echo -e "${YELLOW}Skipping SSL. HTTP only mode.${NC}"
    echo "Enable SSL later by running: sudo certbot certonly --standalone -d $DOMAIN"
fi

# Final status check
echo ""
echo -e "${YELLOW}Testing services...${NC}"

# Test Matrix Conduit
if curl -f http://localhost:6167/_matrix/client/versions &> /dev/null; then
    echo -e "${GREEN}✓ Matrix Conduit is responding${NC}"
else
    echo -e "${RED}✗ Matrix Conduit is not responding${NC}"
fi

# Test Nginx
if docker-compose exec nginx wget -q -O /dev/null http://localhost/health; then
    echo -e "${GREEN}✓ Nginx is responding${NC}"
else
    echo -e "${RED}✗ Nginx is not responding${NC}"
fi

# Display status
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Services running:"
docker-compose ps
echo ""
echo "Access URLs:"
echo "  Matrix Client API: http://localhost:6167/_matrix/client/versions"
echo "  Health Check:      http://localhost/health"
echo ""

if [ -f configs/nginx/ssl/fullchain.pem ]; then
    echo "HTTPS enabled: https://$DOMAIN"
    echo ""
    echo "Well-known URLs:"
    echo "  https://$DOMAIN/.well-known/matrix/server"
    echo "  https://$DOMAIN/.well-known/matrix/client"
fi

echo ""
echo "Next steps:"
echo "  1. Set allow_registration = true in conduit.toml"
echo "  2. Restart: docker-compose restart matrix-conduit"
echo "  3. Create admin account via Element Web"
echo "  4. Set allow_registration = false again"
echo ""
echo "View logs:"
echo "  docker-compose logs -f"
echo ""
echo "Manage services:"
echo "  docker-compose start    # Start all services"
echo "  docker-compose stop     # Stop all services"
echo "  docker-compose restart  # Restart all services"
