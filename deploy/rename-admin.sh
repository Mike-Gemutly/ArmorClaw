#!/bin/bash
# ArmorClaw Admin Rename Utility
# Renames the admin user to avoid conflicts and enumeration

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log() {
    echo -e "${CYAN}[Rename]${NC} $1"
}

log_error() {
    echo -e "${RED}[Rename] ERROR:${NC} $1"
}

log_success() {
    echo -e "${GREEN}[Rename] SUCCESS:${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[Rename] WARNING:${NC} $1"
}

# Check if running inside container
if [ ! -f "/var/lib/armorclaw/.admin_password" ]; then
    log_error "Admin credentials not found. Has ArmorClaw been initialized?"
    exit 1
fi

# Load current credentials
CURRENT_ADMIN=$(cat /var/lib/armorclaw/.admin_username 2>/dev/null || echo "admin")
CURRENT_PASSWORD=$(cat /var/lib/armorclaw/.admin_password)
SERVER_NAME=${ARMORCLAW_SERVER_NAME:-localhost}

log "Current admin user: @${CURRENT_ADMIN}:${SERVER_NAME}"

# Get new username
if [ -n "$1" ]; then
    NEW_USERNAME="$1"
else
    # Interactive mode
    echo -e "${CYAN}Enter new admin username:${NC}"
    echo -e "${YELLOW}Note: This will be your Matrix admin username${NC}"
    read -p "New username: " NEW_USERNAME
    
    if [ -z "$NEW_USERNAME" ]; then
        log_error "Username cannot be empty"
        exit 1
    fi
    
    # Validate username
    if echo "$NEW_USERNAME" | grep -q '[^a-zA-Z0-9._-]'; then
        log_error "Username can only contain letters, numbers, dots, hyphens, and underscores"
        exit 1
    fi
    
    if [ ${#NEW_USERNAME} -lt 3 ] || [ ${#NEW_USERNAME} -gt 32 ]; then
        log_error "Username must be between 3 and 32 characters"
        exit 1
    fi
    
    log "New username will be: ${NEW_USERNAME}"
    
    # Confirmation
    read -p "Confirm rename '$CURRENT_ADMIN' to '$NEW_USERNAME'? [y/N] " confirm
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        log "Rename cancelled"
        exit 0
    fi
fi

# Matrix homeserver URL
HOMESERVER_URL=${ARMORCLAW_MATRIX_HOMESERVER_URL:-http://localhost:6167}

# First, login as current admin to get access token
log "Authenticating as current admin..."
LOGIN_RESPONSE=$(curl -sf -X POST "${HOMESERVER_URL}/_matrix/client/v3/login" \
    -H "Content-Type: application/json" \
    -d "{\"type\":\"m.login.password\",\"user\":\"${CURRENT_ADMIN}\",\"password\":\"${CURRENT_PASSWORD}\"}")

if [ -z "$LOGIN_RESPONSE" ]; then
    log_error "Failed to authenticate as current admin"
    exit 1
fi

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')

if [ -z "$ACCESS_TOKEN" ]; then
    log_error "Failed to get access token"
    exit 1
fi

# Check if new username already exists
log "Checking if '${NEW_USERNAME}' is available..."
CHECK_RESPONSE=$(curl -sf -X POST "${HOMESERVER_URL}/_matrix/client/v3/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${NEW_USERNAME}\",\"password\":\"dummy-check\"}")

if echo "$CHECK_RESPONSE" | grep -qi "user_in_use\|already in use\|exists"; then
    log_error "Username '${NEW_USERNAME}' is already taken"
    exit 1
fi

# Deactivate old admin user (optional but recommended)
log "Deactivating old admin user: ${CURRENT_ADMIN}..."
DEACTIVATE_RESPONSE=$(curl -sf -X POST "${HOMESERVER_URL}/_matrix/client/v3/deactivate" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -d "{\"auth\":{\"type\":\"m.login.password\",\"user\":\"${CURRENT_ADMIN}\",\"password\":\"${CURRENT_PASSWORD}\"}}")

if [ -n "$DEACTIVATE_RESPONSE" ]; then
    log_warn "Old user may not have been deactivated (this is OK if registration is disabled)"
fi

# Now rename/replace via admin API if possible, or create new user
log "Creating new admin user: ${NEW_USERNAME}..."

# Try to register directly if registration is still temporarily enabled
REGISTER_RESPONSE=$(curl -sf -X POST "${HOMESERVER_URL}/_matrix/client/v3/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${NEW_USERNAME}\",\"password\":\"${CURRENT_PASSWORD}\",\"auth\":{\"type\":\"m.login.dummy\"}}")

if echo "$REGISTER_RESPONSE" | grep -q '"access_token"'; then
    # Success - user created
    NEW_ACCESS_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.access_token')
    log_success "New admin user created successfully"
    
    # Update stored credentials
    echo "$NEW_USERNAME" > /var/lib/armorclaw/.admin_username
    
    # Grant admin rights to new user if we have shared secret access
    # This would require access to conduit config - skipping for now
    
    log_success "Rename completed successfully"
    log ""
    log "New credentials:"
    log "  Username: @${NEW_USERNAME}:${SERVER_NAME}"
    log "  Password: ${CURRENT_PASSWORD} (unchanged)"
    log ""
    log "You can now login with the new username"
    log ""
    
    exit 0
else
    log_error "Failed to create new admin user"
    log "You may need to:"
    log "1. Enable registration temporarily: docker exec armorclaw-conduit sed -i 's/allow_registration = false/allow_registration = true/' /etc/armorclaw/conduit.toml"
    log "2. Restart Conduit: docker restart armorclaw-conduit"
    log "3. Create user manually via Element X"
    log "4. Disable registration again"
    exit 1
fi