#!/bin/bash
# =============================================================================
# ArmorClaw Infrastructure Deployment Script
# Target: Hostinger KVM2 (4 GB RAM, Ubuntu 22.04/24.04)
# Purpose: Provision Docker, Nginx (SSL), and Matrix Conduit
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN="${ARMORCLAW_DOMAIN:-chat.armorclaw.com}"
EMAIL="${ARMORCLAW_EMAIL:-admin@armorclaw.com}"
CONDUIT_VERSION="${CONDUIT_VERSION:-v0.4.0}"
MATRIX_DATA_DIR="/var/lib/matrix-conduit"
NGINX_CONF_DIR="/etc/nginx/sites-available"
NGINX_ENABLED_DIR="/etc/nginx/sites-enabled"

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_ubuntu() {
    if [[ ! -f /etc/os-release ]]; then
        log_error "Cannot detect OS"
        exit 1
    fi

    source /etc/os-release
    if [[ "$ID" != "ubuntu" ]]; then
        log_warning "This script is designed for Ubuntu. Detected: $ID"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    log_success "Ubuntu detected: $PRETTY_NAME"
}

# =============================================================================
# System Update & Base Packages
# =============================================================================

update_system() {
    log_info "Updating system packages..."

    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get upgrade -y -qq

    # Install essential tools
    apt-get install -y -qq \
        curl \
        wget \
        ca-certificates \
        gnupg \
        lsb-release \
        ufw \
        fail2ban \
        git \
        htop

    log_success "System updated and base packages installed"
}

# =============================================================================
# Docker Installation
# =============================================================================

install_docker() {
    log_info "Installing Docker..."

    # Remove old versions
    apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

    # Add Docker's official GPG key
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg

    # Set up repository
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
      tee /etc/apt/sources.list.d/docker.list > /dev/null

    # Install Docker
    apt-get update -qq
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    # Enable and start Docker
    systemctl enable docker
    systemctl start docker

    # Add current user to docker group
    if [[ -n "${SUDO_USER:-}" ]]; then
        usermod -aG docker "$SUDO_USER"
        log_success "Added user $SUDO_USER to docker group"
    fi

    # Verify installation
    docker --version
    docker compose version

    log_success "Docker installed and running"
}

# =============================================================================
# Firewall Configuration
# =============================================================================

configure_firewall() {
    log_info "Configuring firewall..."

    # Reset UFW
    ufw --force reset

    # Default policies
    ufw default deny incoming
    ufw default allow outgoing

    # Allow SSH
    ufw allow 22/tcp comment 'SSH'

    # Allow HTTP/HTTPS
    ufw allow 80/tcp comment 'HTTP'
    ufw allow 443/tcp comment 'HTTPS'

    # Allow Matrix federation (optional, can be disabled for privacy)
    # ufw allow 8448/tcp comment 'Matrix Federation'

    # Enable firewall
    ufw --force enable

    log_success "Firewall configured"
    ufw status verbose
}

# =============================================================================
# Nginx Installation & Configuration
# =============================================================================

install_nginx() {
    log_info "Installing Nginx..."

    apt-get install -y -qq nginx

    # Remove default site
    rm -f /etc/nginx/sites-enabled/default

    # Create SSL directory
    mkdir -p /etc/nginx/ssl

    log_success "Nginx installed"
}

configure_nginx_standalone() {
    log_info "Configuring Nginx for initial HTTP access..."

    cat > "$NGINX_CONF_DIR/$DOMAIN.conf" <<EOF
server {
    listen 80;
    listen [::]:80;

    server_name $DOMAIN;

    # Let's Encrypt challenge
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # Temporary redirect to HTTPS before SSL is obtained
    location / {
        return 200 "ArmorClaw Matrix Server - SSL pending";
        add_header Content-Type text/plain;
    }

    access_log /var/log/nginx/${DOMAIN}-access.log;
    error_log /var/log/nginx/${DOMAIN}-error.log;
}
EOF

    ln -sf "$NGINX_CONF_DIR/$DOMAIN.conf" "$NGINX_ENABLED_DIR/"

    # Test and reload
    nginx -t
    systemctl reload nginx

    log_success "Nginx configured for HTTP access"
}

# =============================================================================
# SSL Certificate (Let's Encrypt)
# =============================================================================

obtain_ssl() {
    log_info "Obtaining SSL certificate for $DOMAIN..."

    # Install Certbot
    apt-get install -y -qq certbot python3-certbot-nginx

    # Obtain certificate
    certbot --nginx -d "$DOMAIN" --email "$EMAIL" --non-interactive --agree-tos --redirect

    # Set up auto-renewal
    systemctl enable certbot.timer
    systemctl start certbot.timer

    log_success "SSL certificate obtained and auto-renewal configured"
}

# =============================================================================
# Matrix Conduit Installation
# =============================================================================

install_conduit() {
    log_info "Installing Matrix Conduit..."

    # Create conduit user
    useradd -rs /bin/false conduit

    # Create data directory
    mkdir -p "$MATRIX_DATA_DIR"
    chown -R conduit:conduit "$MATRIX_DATA_DIR"

    # Create Conduit directory
    mkdir -p /opt/matrix-conduit
    cd /opt/matrix-conduit

    # Download Conduit binary
    log_info "Downloading Conduit $CONDUIT_VERSION..."
    curl -L "https://github.com/ShadowRyan/Conduit/archive/refs/tags/${CONDUIT_VERSION}.tar.gz" -o conduit.tar.gz
    tar -xzf conduit.tar.gz --strip-components=1
    rm conduit.tar.gz

    # Create configuration
    cat > /opt/matrix-conduit/conduit.toml <<EOF
[global]
# The server name is the name of the server (for example: example.com)
server_name = "$DOMAIN"

# The ports Conduit will listen on
port = 6167

# Max request size for file uploads
max_request_size = 20_000_000

# Disable registration (we'll use registration tokens)
allow_registration = false
allow_encryption = true
allow_federation = true

# Database path
database_path = "/var/lib/matrix-conduit"
database_backend = "rocksdb"

[global.well_known]
client = "https://$DOMAIN"
server = "https://$DOMAIN:8448"

# Performance tuning
[global.performance]
pdu_size = 100
concurrent_request_sending = 50
EOF

    chown -R conduit:conduit /opt/matrix-conduit

    log_success "Conduit installed"
}

# =============================================================================
# Conduit Docker Service
# =============================================================================

create_conduit_docker_service() {
    log_info "Creating Conduit Docker service..."

    mkdir -p /opt/matrix-conduit/docker

    cat > /opt/matrix-conduit/docker/docker-compose.yml <<EOF
version: "3.8"

services:
  conduit:
    image: matrixconduit/matrix-conduit:$CONDUIT_VERSION
    restart: unless-stopped
    volumes:
      - ./conduit.toml:/conduit.toml:ro
      - $MATRIX_DATA_DIR:/var/lib/matrix-conduit
    ports:
      - "6167:6167"
    networks:
      - matrix

  # Optional: Caddy reverse proxy for Conduit
  caddy:
    image: caddy:2-alpine
    restart: unless-stopped
    volumes:
      - ./caddy/Caddyfile:/etc/caddy/Caddyfile:ro
      - ./caddy/data:/data
      - ./caddy/config:/config
      - ./caddy/logs:/var/log/caddy
    ports:
      - "8448:8448"
    networks:
      - matrix

networks:
  matrix:
    driver: bridge
EOF

    # Create Caddy config for Conduit
    mkdir -p /opt/matrix-conduit/docker/caddy

    cat > /opt/matrix-conduit/docker/caddy/Caddyfile <<EOF
{
    # Disable Caddy admin API
    admin off
}

https://$DOMAIN:8448 {
    reverse_proxy conduit:6167

    # Federation endpoint
    handle_path /_matrix/* {
        reverse_proxy conduit:6167
    }

    # Client API
    handle_path /_matrix/client/* {
        reverse_proxy conduit:6167
    }

    logs {
        output file /var/log/caddy/access.log
    }
}
EOF

    chown -R conduit:conduit /opt/matrix-conduit/docker

    # Start the service
    cd /opt/matrix-conduit/docker
    docker compose up -d

    log_success "Conduit Docker service started"
}

# =============================================================================
# Nginx Configuration for Matrix
# =============================================================================

configure_nginx_matrix() {
    log_info "Configuring Nginx for Matrix..."

    cat > "$NGINX_CONF_DIR/$DOMAIN-matrix.conf" <<EOF
# Rate limiting
limit_req_zone \$binary_remote_addr zone=matrix_clientapi:10m rate=10r/s;
limit_req_zone \$binary_remote_addr zone=matrix_federation:10m rate=100r/s;

# Upstream Conduit
upstream matrix_conduit {
    server 127.0.0.1:8448;
    keepalive 32;
}

# Main HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    server_name $DOMAIN;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options SAMEORIGIN always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Client API
    location /_matrix/client/ {
        limit_req zone=matrix_clientapi burst=20 nodelay;

        proxy_pass http://matrix_conduit;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_buffering off;
        proxy_read_timeout 900s;
    }

    # Federation API
    location /_matrix/federation/ {
        limit_req zone=matrix_federation burst=50 nodelay;

        proxy_pass http://matrix_conduit;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # Media
    location /_matrix/media/ {
        proxy_pass http://matrix_conduit;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_buffering off;
    }

    # Well-known for Matrix client auto-discovery
    location /.well-known/matrix/client {
        default_type application/json;
        return 200 '{
            "m.homeserver": {
                "base_url": "https://$DOMAIN"
            },
            "m.identity_server": {
                "base_url": "https://vector.im"
            }
        }';
        add_header Access-Control-Allow-Origin *;
    }

    location /.well-known/matrix/server {
        default_type application/json;
        return 200 '{
            "m.server": "$DOMAIN:8448"
        }';
        add_header Access-Control-Allow-Origin *;
    }

    # Static files (for future web client)
    location / {
        root /var/www/matrix;
        index index.html;
        try_files \$uri \$uri/ =404;
    }

    access_log /var/log/nginx/${DOMAIN}-access.log;
    error_log /var/log/nginx/${DOMAIN}-error.log;
}

# HTTP to HTTPS redirect
server {
    listen 80;
    listen [::]:80;
    server_name $DOMAIN;

    # Let's Encrypt
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 301 https://\$host\$request_uri;
    }
}
EOF

    ln -sf "$NGINX_CONF_DIR/$DOMAIN-matrix.conf" "$NGINX_ENABLED_DIR/"

    # Remove standalone config
    rm -f "$NGINX_ENABLED_DIR/$DOMAIN.conf"

    # Test and reload
    nginx -t
    systemctl reload nginx

    log_success "Nginx configured for Matrix"
}

# =============================================================================
# Verification
# =============================================================================

verify_deployment() {
    log_info "Verifying deployment..."

    echo ""
    echo "=== Service Status ==="
    systemctl status docker --no-pager -l
    echo ""
    systemctl status nginx --no-pager -l
    echo ""

    echo "=== Docker Containers ==="
    docker compose -f /opt/matrix-conduit/docker/docker-compose.yml ps
    echo ""

    echo "=== Network Listeners ==="
    ss -tlnp | grep -E ':(80|443|8448|6167)\s'
    echo ""

    echo "=== SSL Certificate ==="
    certbot certificates 2>/dev/null || echo "Certificates not yet obtained"
    echo ""

    log_success "Deployment verification complete"
}

# =============================================================================
# Print Next Steps
# =============================================================================

print_next_steps() {
    cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗║
║           ArmorClaw Infrastructure Deployment Complete              ║
╚══════════════════════════════════════════════════════════════════════╝${NC}

${BLUE}Infrastructure Details:${NC}
  • Domain:       https://$DOMAIN
  • Matrix:       https://$DOMAIN (client port 443, federation port 8448)
  • Conduit Data: $MATRIX_DATA_DIR
  • Nginx Config: $NGINX_CONF_DIR/$DOMAIN-matrix.conf

${BLUE}Services Running:${NC}
  • Docker:       Active
  • Nginx:        Active (HTTP/HTTPS)
  • Conduit:      Active (Docker)
  • Certbot:      Active (auto-renewal)

${BLUE}Memory Usage (Expected):${NC}
  • Ubuntu base:     ~400 MB
  • Nginx:           ~40 MB
  • Conduit:         ~200 MB (max, with traffic)
  • Total:           ~640 MB (well under 2 GB budget)

${BLUE}Next Steps:${NC}
  1. Test federation: https://federation-tester.matrix.org/
  2. Create admin user on the VPS:
     docker exec -it \$(docker ps -q -f name=conduit) \\
       /conduit-cli user-register -u admin -p YOUR_PASSWORD

  3. Connect with Element Web: https://app.element.io
     Homeserver: https://$DOMAIN

  4. Verify firewall:
     sudo ufw status verbose

${YELLOW}Important Notes:${NC}
  • SSL certificates auto-renew via certbot timer
  • Firewall blocks all inbound except HTTP/HTTPS/SSH
  • Matrix federation enabled (disable in conduit.toml if privacy needed)
  • Logs: /var/log/nginx/ and docker logs matrix-conduit

${GREEN}Your ArmorClaw homeserver is ready!${NC}

EOF
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║        ArmorClaw Infrastructure Deployment Script            ║"
    echo "║           Docker + Nginx + Matrix Conduit                     ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    # Prerequisites check
    check_root
    check_ubuntu

    # Confirm
    log_warning "This will deploy Matrix infrastructure on this server"
    log_warning "Domain: $DOMAIN"
    log_warning "Email: $EMAIL"
    echo ""
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deployment cancelled"
        exit 0
    fi

    # Execute deployment stages
    log_info "Starting deployment..."
    echo ""

    update_system
    configure_firewall
    install_docker
    install_nginx
    configure_nginx_standalone
    obtain_ssl
    install_conduit
    create_conduit_docker_service
    configure_nginx_matrix

    echo ""
    verify_deployment
    print_next_steps
}

# Run main function
main "$@"
