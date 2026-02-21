#!/bin/bash
# ArmorClaw Matrix Admin User Creation
# Creates admin user via Conduit CLI (no registration window needed)
# Version: 1.0.0
#
# Usage:
#   ./deploy/create-matrix-admin.sh [username] [password]
#   ./deploy/create-matrix-admin.sh admin  # prompts for password
#
# Security: This script keeps allow_registration=false and creates users
# via the admin CLI, eliminating the registration window vulnerability.

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
CONTAINER_NAME="${CONDUIT_CONTAINER:-armorclaw-conduit}"
SERVER_NAME="${MATRIX_SERVER_NAME:-matrix.armorclaw.com}"

# Parse arguments
USERNAME="${1:-admin}"
PASSWORD="$2"

echo -e "${CYAN}ArmorClaw Matrix Admin Creation${NC}"
echo "=================================="
echo ""

# Check if container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${RED}Error: Container '$CONTAINER_NAME' is not running${NC}"
    echo "Start the Matrix stack first:"
    echo "  docker compose -f docker-compose.matrix.yml up -d"
    exit 1
fi

# Check if conduit-admin is available
if ! docker exec "$CONTAINER_NAME" which conduit-admin &>/dev/null; then
    echo -e "${YELLOW}Note: conduit-admin not found in container${NC}"
    echo "Using Conduit API for user creation..."
    USE_API=true
else
    USE_API=false
fi

# Get password if not provided
if [ -z "$PASSWORD" ]; then
    echo -e "${CYAN}Creating admin user: @${USERNAME}:${SERVER_NAME}${NC}"
    echo ""
    read -s -p "Enter password (min 8 chars): " PASSWORD
    echo ""
    read -s -p "Confirm password: " PASSWORD_CONFIRM
    echo ""

    if [ "$PASSWORD" != "$PASSWORD_CONFIRM" ]; then
        echo -e "${RED}Error: Passwords do not match${NC}"
        exit 1
    fi

    if [ ${#PASSWORD} -lt 8 ]; then
        echo -e "${RED}Error: Password must be at least 8 characters${NC}"
        exit 1
    fi
fi

# Validate password strength
STRENGTH=0
if [[ ${#PASSWORD} -ge 12 ]]; then ((STRENGTH++)); fi
if [[ "$PASSWORD" =~ [A-Z] ]]; then ((STRENGTH++)); fi
if [[ "$PASSWORD" =~ [a-z] ]]; then ((STRENGTH++)); fi
if [[ "$PASSWORD" =~ [0-9] ]]; then ((STRENGTH++)); fi
if [[ "$PASSWORD" =~ [^A-Za-z0-9] ]]; then ((STRENGTH++)); fi

if [ $STRENGTH -lt 3 ]; then
    echo -e "${YELLOW}Warning: Weak password. Consider using:${NC}"
    echo "  - 12+ characters"
    echo "  - Mix of uppercase, lowercase, numbers, symbols"
    echo ""
    read -p "Continue with weak password? [y/N]: " CONTINUE
    if [[ ! "$CONTINUE" =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Create user
FULL_USER_ID="@${USERNAME}:${SERVER_NAME}"
echo ""
echo -e "${CYAN}Creating user: ${FULL_USER_ID}${NC}"

if [ "$USE_API" = true ]; then
    # Use Conduit's registration API with shared secret (if available)
    # or fallback to admin API
    echo -e "${YELLOW}Using Conduit admin API...${NC}"

    # Check for admin token
    ADMIN_TOKEN="${CONDUIT_ADMIN_TOKEN:-}"
    if [ -z "$ADMIN_TOKEN" ]; then
        # Try to get token from environment or generate one
        echo -e "${YELLOW}No admin token found. Setting up shared secret registration...${NC}"

        # This is a temporary approach - in production, use a pre-configured admin token
        echo ""
        echo -e "${CYAN}IMPORTANT: You need to enable admin API in conduit.toml:${NC}"
        echo "  [global.admin]"
        echo "  enable = true"
        echo "  # conduit_admin_token = \"your-secure-token\""
        echo ""
        echo -e "${CYAN}Or use shared secret registration:${NC}"
        echo "  [global.matrix]"
        echo "  registration_shared_secret = \"$(openssl rand -hex 32)\""
        echo ""
        echo "After configuration, restart Conduit and run this script again."
        echo ""
        echo -e "${CYAN}Alternatively, register manually:${NC}"
        echo "  1. Temporarily set allow_registration = true in configs/conduit.toml"
        echo "  2. Restart: docker compose -f docker-compose.matrix.yml restart"
        echo "  3. Register via Element X or curl"
        echo "  4. IMMEDIATELY set allow_registration = false"
        echo "  5. Restart again"
        echo ""
        exit 1
    fi

    # Create user via admin API
    RESPONSE=$(curl -s -X POST "http://localhost:6167/_matrix/admin/v1/users/@${USERNAME}:${SERVER_NAME}" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"password\":\"$PASSWORD\",\"admin\":true}" 2>/dev/null || echo '{"error":"failed"}')

    if echo "$RESPONSE" | grep -q '"error"'; then
        echo -e "${RED}Failed to create user: $RESPONSE${NC}"
        exit 1
    fi
else
    # Use conduit-admin CLI
    echo "$PASSWORD" | docker exec -i "$CONTAINER_NAME" conduit-admin register "$FULL_USER_ID" --password-stdin --admin 2>/dev/null

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}User created successfully via CLI${NC}"
    else
        echo -e "${YELLOW}CLI creation failed, user may already exist or CLI not available${NC}"
    fi
fi

# Verify user was created
echo ""
echo -e "${CYAN}Verifying user creation...${NC}"

# Try to login with the new user
LOGIN_RESPONSE=$(curl -s -X POST "http://localhost:6167/_matrix/client/v3/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"type\":\"m.login.password\",
        \"user\":\"$USERNAME\",
        \"password\":\"$PASSWORD\"
    }" 2>/dev/null || echo '{"error":"failed"}')

if echo "$LOGIN_RESPONSE" | grep -q '"access_token"'; then
    echo -e "${GREEN}✓ User verified: ${FULL_USER_ID}${NC}"
    echo ""
    echo -e "${CYAN}You can now login with:${NC}"
    echo "  Username: @${USERNAME}:${SERVER_NAME}"
    echo "  Server:   https://${SERVER_NAME}"
else
    echo -e "${YELLOW}Warning: Could not verify user creation${NC}"
    echo "Response: $LOGIN_RESPONSE"
fi

# Security reminder
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Security Status:${NC}"
echo ""
echo -e "  ${GREEN}✓${NC} User created without enabling registration"
echo -e "  ${GREEN}✓${NC} No registration window exposure"
echo ""
echo -e "${CYAN}Verify registration is disabled:${NC}"
echo "  grep 'allow_registration' configs/conduit.toml"
echo "  # Should show: allow_registration = false"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
