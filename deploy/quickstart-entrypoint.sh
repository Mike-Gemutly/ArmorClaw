#!/bin/bash
# ArmorClaw Quick Start Entrypoint
# Production-grade bootstrap with shared-secret admin creation

set -e

# Paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
BOOTSTRAP_SCRIPT="/opt/armorclaw/bootstrap-admin.py"
INIT_FLAG="$DATA_DIR/.bootstrapped"
CONDUIT_CONFIG="$CONFIG_DIR/conduit.toml"

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

# Check if Conduit config exists
if [ ! -f "$CONDUIT_CONFIG" ]; then
    log_error "Conduit config not found at $CONDUIT_CONFIG"
    exit 1
fi

# Extract server name from config for display
SERVER_NAME=$(grep -E '^server_name\s*=' "$CONDUIT_CONFIG" | sed 's/^.*=\s*"\(.*\)".*/\1/' || echo "localhost")
log "Server name: $SERVER_NAME"

# Start Conduit in background
log "Starting Conduit..."
# Run Conduit as non-root user (UID 10000 is standard for Conduit)
docker run -d \
    --name armorclaw-conduit \
    --restart unless-stopped \
    --user 10000:10000 \
    -v "$CONFIG_DIR:/etc/armorclaw:ro" \
    -v /var/lib/conduit:/var/lib/conduit \
    -p 6167:6167 \
    mikegemut/conduit:latest

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

# Run the Python bootstrap script
log "Running admin bootstrap..."
if [ -f "$BOOTSTRAP_SCRIPT" ]; then
    # Set environment for bootstrap script
    export ARMORCLAW_ADMIN_PASSWORD="$ARMORCLAW_ADMIN_PASSWORD"
    export ARMORCLAW_SERVER_NAME="$SERVER_NAME"
    
    if python3 "$BOOTSTRAP_SCRIPT"; then
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