#!/bin/bash
# ArmorClaw Matrix Admin User Creation
# Creates admin user via Conduit's registration token API
# Version: 2.1.0
#
# Usage:
#   ./deploy/create-matrix-admin.sh [username] [password]
#   ./deploy/create-matrix-admin.sh admin MySecurePass123
#   ./deploy/create-matrix-admin.sh                         # prompts for both
#
# Security: Uses Conduit's registration_token for secure admin creation.
# Requires registration_token to be set in conduit.toml.
# The setup wizard handles this automatically during first-run setup.
#
# For post-setup use, temporarily add to conduit.toml:
#   allow_registration = true
#   registration_token = "your-secret-here"
# Then restart Conduit, run this script, and remove the lines.

# NOTE: Do NOT use set -e. We handle errors explicitly.

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
CONDUIT_URL="${CONDUIT_URL:-http://localhost:6167}"
CONTAINER_NAME="${CONDUIT_CONTAINER:-armorclaw-conduit}"
SERVER_NAME="${MATRIX_SERVER_NAME:-matrix.armorclaw.com}"
SHARED_SECRET="${REGISTRATION_SHARED_SECRET:-}"

# Parse arguments
USERNAME="${1:-}"
PASSWORD="${2:-}"

echo -e "${CYAN}ArmorClaw Matrix Admin Creation${NC}"
echo "=================================="
echo ""

# Check if Conduit is reachable
if ! curl -sf --connect-timeout 5 "${CONDUIT_URL}/_matrix/client/versions" >/dev/null 2>&1; then
    echo -e "${RED}Error: Conduit is not reachable at ${CONDUIT_URL}${NC}"
    echo ""
    echo "Make sure Conduit is running:"
    echo "  docker logs ${CONTAINER_NAME}"
    echo "  docker compose -f docker-compose.matrix.yml up -d"
    exit 1
fi

echo -e "${GREEN}✓ Conduit is reachable at ${CONDUIT_URL}${NC}"

# Get shared secret if not provided
if [ -z "$SHARED_SECRET" ]; then
    echo ""
    echo -e "${YELLOW}No REGISTRATION_SHARED_SECRET environment variable set.${NC}"
    echo ""
    echo "To use this script, you need a registration_shared_secret in conduit.toml."
    echo ""
    echo "Steps:"
    echo "  1. Generate a secret: openssl rand -hex 32"
    echo "  2. Add to conduit.toml: registration_shared_secret = \"<secret>\""
    echo "  3. Restart Conduit: docker restart ${CONTAINER_NAME}"
    echo "  4. Set the env var: export REGISTRATION_SHARED_SECRET=\"<secret>\""
    echo "  5. Run this script again"
    echo "  6. REMOVE the line from conduit.toml and restart Conduit"
    echo ""
    if [ -t 0 ] || [ -c /dev/tty ]; then
        read -p "Enter shared secret (or press Enter to exit): " SHARED_SECRET < /dev/tty
    else
        SHARED_SECRET=""
    fi
    if [ -z "$SHARED_SECRET" ]; then
        exit 1
    fi
fi

# Get username if not provided
if [ -z "$USERNAME" ]; then
    echo ""
    if [ -t 0 ] || [ -c /dev/tty ]; then
        read -p "Username [admin]: " USERNAME < /dev/tty
    else
        USERNAME="admin"
    fi
    USERNAME="${USERNAME:-admin}"
fi

# Get password if not provided
if [ -z "$PASSWORD" ]; then
    echo ""
    echo -e "${CYAN}Creating admin user: @${USERNAME}:${SERVER_NAME}${NC}"
    echo ""

    while true; do
        if [ -t 0 ] || [ -c /dev/tty ]; then
            read -s -p "Enter password (min 8 chars): " PASSWORD < /dev/tty
        else
            echo "Error: Password required in non-interactive mode"
            exit 1
        fi
        echo ""

        if [ ${#PASSWORD} -lt 8 ]; then
            echo -e "${RED}Error: Password must be at least 8 characters${NC}"
            continue
        fi

        if [ -t 0 ] || [ -c /dev/tty ]; then
            read -s -p "Confirm password: " PASSWORD_CONFIRM < /dev/tty
        fi
        echo ""

        if [ "$PASSWORD" != "$PASSWORD_CONFIRM" ]; then
            echo -e "${RED}Error: Passwords do not match${NC}"
            continue
        fi

        break
    done
fi

# Validate password length
if [ ${#PASSWORD} -lt 8 ]; then
    echo -e "${RED}Error: Password must be at least 8 characters${NC}"
    exit 1
fi

# Register user via Conduit registration token API
# Conduit uses registration_token (not HMAC like Synapse)
echo ""
echo -e "${CYAN}Registering user @${USERNAME}:${SERVER_NAME}...${NC}"

# Conduit uses registration_token directly in the request body
# No HMAC calculation needed - just pass the token
REG_RESPONSE=$(curl -s --connect-timeout 10 -X POST \
    "${CONDUIT_URL}/_matrix/client/v3/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\",\"token\":\"$SHARED_SECRET\"}" 2>/dev/null)

# Check for errors
ERROR_MSG=$(echo "$REG_RESPONSE" | jq -r '.error // empty' 2>/dev/null)
if [ -n "$ERROR_MSG" ]; then
    if echo "$ERROR_MSG" | grep -qi "exists\|already"; then
        echo -e "${YELLOW}User @${USERNAME}:${SERVER_NAME} already exists${NC}"
    else
        echo -e "${RED}Error: $ERROR_MSG${NC}"
        exit 1
    fi
else
    USER_ID=$(echo "$REG_RESPONSE" | jq -r '.user_id // empty' 2>/dev/null)
    if [ -n "$USER_ID" ]; then
        echo -e "${GREEN}✓ User created: $USER_ID${NC}"
    else
        echo -e "${YELLOW}Warning: Unexpected response: $REG_RESPONSE${NC}"
    fi
fi

# Verify login
echo ""
echo -e "${CYAN}Verifying login...${NC}"
LOGIN_RESPONSE=$(curl -s --connect-timeout 10 -X POST "${CONDUIT_URL}/_matrix/client/v3/login" \
    -H "Content-Type: application/json" \
    -d "{\"type\":\"m.login.password\",\"user\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" 2>/dev/null)

if echo "$LOGIN_RESPONSE" | grep -q '"access_token"'; then
    echo -e "${GREEN}✓ Login verified: @${USERNAME}:${SERVER_NAME}${NC}"
else
    echo -e "${YELLOW}Warning: Login verification failed${NC}"
    echo "Response: $LOGIN_RESPONSE"
fi

# Security reminder
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Security Reminder:${NC}"
echo ""
echo -e "  ${GREEN}✓${NC} User created via registration token (no open registration)"
echo -e "  ${YELLOW}⚠${NC} Remove registration_token from conduit.toml"
echo -e "  ${YELLOW}⚠${NC} Restart Conduit after removing the token"
echo ""
echo "Connect with Element X or ArmorChat:"
echo "  Homeserver: ${CONDUIT_URL}"
echo "  Username:   @${USERNAME}:${SERVER_NAME}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

