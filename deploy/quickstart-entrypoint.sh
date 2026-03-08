#!/bin/bash
# ArmorClaw Quick Start Entrypoint
# Production-grade bootstrap with shared-secret admin creation

# Paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
BOOTSTRAP_SCRIPT="/opt/armorclaw/bootstrap-admin"
INIT_FLAG="$DATA_DIR/.bootstrapped"
CONDUIT_CONFIG="$CONFIG_DIR/conduit.toml"

# Docker Compose fallback
DOCKER_COMPOSE="${DOCKER_COMPOSE:-docker compose}"

# Conduit image
CONDUIT_VERSION="${CONDUIT_VERSION:-latest}"
CONDUIT_IMAGE="${CONDUIT_IMAGE:-matrixconduit/matrix-conduit:$CONDUIT_VERSION}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log() {
    echo -e "${CYAN}[QuickStart]${NC} $1"
}

log_error() {
    echo -e "${RED}[QuickStart] ERROR:${NC} $1"
}

log_success() {
    echo -e "${GREEN}[QuickStart] SUCCESS:${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[QuickStart] WARNING:${NC} $1"
}

# CI/Docker-less Environment Detection
# Check if Docker socket is available BEFORE set -e is applied
if [ ! -S /var/run/docker.sock ]; then
    log_warn "Docker socket not available at /var/run/docker.sock"
    
    # Determine if this is a CI environment or just missing Docker
    if [ "${GITHUB_ACTIONS:-false}" = "true" ] || [ "${CI:-false}" = "true" ] || [ "${ARMORCLAW_SKIP_DOCKER:-false}" = "true" ]; then
        log "CI environment detected - running in bridge-only mode"
    else
        log "Docker not available - running in bridge-only mode"
        log "For full quickstart, ensure Docker is running and socket is mounted"
    fi
    
    # Create minimal config for bridge-only mode
    mkdir -p "$CONFIG_DIR" "$DATA_DIR"
    
    # Copy bridge config if available
    if [ -f "/opt/armorclaw/configs/config.toml" ] && [ ! -f "$CONFIG_DIR/config.toml" ]; then
        cp /opt/armorclaw/configs/config.toml "$CONFIG_DIR/config.toml"
        log "Copied bridge config template"
    fi
    
    # Create minimal conduit config
    if [ -f "/opt/armorclaw/configs/conduit.toml" ] && [ ! -f "$CONDUIT_CONFIG" ]; then
        cp /opt/armorclaw/configs/conduit.toml "$CONDUIT_CONFIG"
        log "Copied Conduit config template"
    fi
    
    # Mark as bootstrapped
    touch "$INIT_FLAG"
    
    log_success "Bootstrap complete - bridge-only mode ready"
    log "To run full quickstart with Conduit, use a host with Docker access"
    
    # Exit successfully - allow CI/other tests to proceed
    exit 0
fi

set -e

# Check if already bootstrapped
if [ -f "$INIT_FLAG" ]; then
    log "Configuration already bootstrapped, starting services..."
    exec /opt/armorclaw/container-setup.sh "$@"
fi

log "Starting ArmorClaw quick start bootstrap..."

# Generate admin password if not provided
if [ -z "$ARMORCLAW_ADMIN_PASSWORD" ]; then
    ARMORCLAW_ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=' | head -c 16)
    log "Generated admin password: $ARMORCLAW_ADMIN_PASSWORD"
fi

# Use custom admin username if provided (for conflict avoidance)
if [ -n "$ARMORCLAW_ADMIN_USERNAME" ]; then
    log "Using custom admin username: $ARMORCLAW_ADMIN_USERNAME"
fi

# Ensure required directories exist
mkdir -p "$CONFIG_DIR" "$DATA_DIR"

# Migration: Handle older installs that used /var/lib/matrix-conduit
if [ -d /var/lib/matrix-conduit ] && [ ! -d /var/lib/conduit ]; then
    log "Migrating existing Matrix data from /var/lib/matrix-conduit"
    mv /var/lib/matrix-conduit /var/lib/conduit
    log_success "Migration complete"
fi

# Copy Conduit config from template if not exists
TEMPLATE_CONFIG="/opt/armorclaw/configs/conduit.toml"
if [ ! -f "$CONDUIT_CONFIG" ]; then
    if [ -f "$TEMPLATE_CONFIG" ]; then
        log "Copying Conduit config template..."
        cp "$TEMPLATE_CONFIG" "$CONDUIT_CONFIG"
    else
        log_error "Conduit config template not found at $TEMPLATE_CONFIG"
        exit 1
    fi
fi

# Detect or use provided server name
if [ -z "$ARMORCLAW_SERVER_NAME" ]; then
    # Auto-detect: try external IP, fallback to hostname, then localhost
    ARMORCLAW_SERVER_NAME=$(curl -sf --connect-timeout 3 ifconfig.me 2>/dev/null || \
                           hostname -I 2>/dev/null | awk '{print $1}' || \
                           hostname 2>/dev/null || \
                           echo "localhost")
    log "Auto-detected server name: $ARMORCLAW_SERVER_NAME"
else
    log "Using provided server name: $ARMORCLAW_SERVER_NAME"
fi

# Update server_name in config
if grep -q '^server_name\s*=' "$CONDUIT_CONFIG"; then
    sed -i "s/^server_name\s*=.*/server_name = \"$ARMORCLAW_SERVER_NAME\"/" "$CONDUIT_CONFIG"
    log "Updated server_name in config to: $ARMORCLAW_SERVER_NAME"
else
    echo "server_name = \"$ARMORCLAW_SERVER_NAME\"" >> "$CONDUIT_CONFIG"
fi

# Export for later use
SERVER_NAME="$ARMORCLAW_SERVER_NAME"
log "Server name: $SERVER_NAME"

# ============================================
# Robust Conduit Detection (Idempotent)
# Detects any Conduit container by image or port
# ============================================

CONDUIT_CONTAINER=""
CONDUIT_PORT="6167"
USE_EXISTING_CONDUIT=false

if ! docker info >/dev/null 2>&1; then
    log "Docker daemon not running"
    exit 1
fi

if ! docker ps >/dev/null 2>&1; then
    log "Docker not accessible for current user"
    exit 1
fi

# Wait for Docker to be ready
log "Waiting for Docker daemon..."
for ((i=1;i<=10;i++)); do
    if docker info >/dev/null 2>&1 && docker ps >/dev/null 2>&1; then
        log "Docker daemon ready"
        break
    fi
    sleep 2
done

# First: detect container created from Conduit image
CONDUIT_CONTAINER=$(docker ps -a \
    --filter "ancestor=matrixconduit/matrix-conduit" \
    --format "{{.Names}}" | head -n1)

# Fallback: detect container exposing port 6167
if [ -z "$CONDUIT_CONTAINER" ]; then
    while read -r NAME PORTS; do
        if echo "$PORTS" | grep -E "[:.]${CONDUIT_PORT}->" >/dev/null 2>&1; then
            IMAGE=$(docker inspect --format '{{.Config.Image}}' "$NAME" 2>/dev/null)
            if [ -n "$IMAGE" ] && echo "$IMAGE" | grep -qi "matrix-conduit"; then
                CONDUIT_CONTAINER="$NAME"
                break
            fi
        fi
    done < <(docker ps --format "{{.Names}} {{.Ports}}")
fi

if [ -n "$CONDUIT_CONTAINER" ]; then
    log "Existing Conduit: $CONDUIT_CONTAINER"
    USE_EXISTING_CONDUIT=true
    if ! docker ps --format '{{.Names}}' | grep -q "^${CONDUIT_CONTAINER}$"; then
        log "Starting existing Conduit container"
        docker start "$CONDUIT_CONTAINER" || {
            log "Failed to start existing Conduit"
            exit 1
        }
    fi
fi

# Create container if not found
if [ "$USE_EXISTING_CONDUIT" = false ]; then
    log "Creating Conduit container"
    docker run -d \
        --name armorclaw-conduit \
        --restart unless-stopped \
        --user 10000:10000 \
        -v "$CONFIG_DIR:/etc/armorclaw:ro" \
        -v /var/lib/conduit:/var/lib/conduit \
        -p 6167:6167 \
        matrixconduit/matrix-conduit:latest
fi

# Wait for Conduit to be ready
log "Waiting for Conduit to start..."
MAX_WAIT=30
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    if curl -sf http://localhost:6167/_matrix/client/versions >/dev/null 2>&1; then
        log_success "Conduit is ready"
        break
    fi
    sleep 1
    WAITED=$((WAITED + 1))
done

if [ $WAITED -eq $MAX_WAIT ]; then
    log_error "Conduit failed to start within $MAX_WAIT seconds"
    docker logs armorclaw-conduit 2>&1 | tail -20
    exit 1
fi

# Run the bootstrap binary
log "Running admin bootstrap..."
if [ -x "$BOOTSTRAP_SCRIPT" ]; then
    # Set environment for bootstrap script
    export ARMORCLAW_ADMIN_PASSWORD="$ARMORCLAW_ADMIN_PASSWORD"
    export ARMORCLAW_SERVER_NAME="$SERVER_NAME"
    
    if "$BOOTSTRAP_SCRIPT"; then
        log_success "Admin bootstrap completed"
        
        # Show admin credentials
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo -e "${GREEN}ArmorClaw Admin User Ready${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo -e "${CYAN}Credentials:${NC}"
        echo "  Username: @admin:$SERVER_NAME"
        echo "  Password: $ARMORCLAW_ADMIN_PASSWORD"
        echo ""
        echo -e "${CYAN}Connect with:${NC}"
        echo "  Element X: http://$SERVER_NAME:6167"
        echo "  ArmorChat: Scan QR below"
        echo ""
        
        # Generate QR code for mobile app
        if command -v qrencode >/dev/null 2>&1; then
            echo -e "${CYAN}QR Code for ArmorChat:${NC}"
            # Create config URL
            CONFIG_URL="armorclaw://config?server=$SERVER_NAME&port=6167&user=admin&pass=$ARMORCLAW_ADMIN_PASSWORD"
            echo "$CONFIG_URL" | qrencode -t ANSI
            echo ""
        fi
        
        # Show bridge command examples
        echo -e "${CYAN}Bridge Commands:${NC}"
        echo "  docker logs -f armorclaw-conduit                    # Matrix logs"
        echo "  docker exec -it armorclaw-conduit conduit-cli        # Conduit CLI"
        echo ""
    else
        log_error "Admin bootstrap failed"
        docker logs armorclaw-conduit 2>&1 | tail -20
        exit 1
    fi
else
    log_error "Bootstrap script not found: $BOOTSTRAP_SCRIPT"
    exit 1
fi

# Mark as bootstrapped
touch "$INIT_FLAG"

# Continue with normal setup
log "Starting main container setup..."
exec /opt/armorclaw/container-setup.sh "$@"