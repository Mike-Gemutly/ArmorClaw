#!/bin/sh
# ArmorClaw Matrix Provisioning Script
set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

MATRIX_DOMAIN="${MATRIX_DOMAIN:-matrix.localhost}"
MATRIX_HOMESERVER_URL="${MATRIX_HOMESERVER_URL:-http://matrix:6167}"
MATRIX_ADMIN_USER="${MATRIX_ADMIN_USER:-admin}"
MATRIX_ADMIN_PASSWORD="${MATRIX_ADMIN_PASSWORD:-admin}"
MATRIX_BRIDGE_USER="${MATRIX_BRIDGE_USER:-bridge}"
MATRIX_BRIDGE_PASSWORD="${MATRIX_BRIDGE_PASSWORD:-bridge123}"
ROOM_NAME="${ROOM_NAME:-ArmorClaw Agents}"
ROOM_ALIAS="${ROOM_ALIAS:-agents}"
STATE_FILE="/provision/.provisioned"

if [ -f "$STATE_FILE" ]; then
    cat "$STATE_FILE"
    exit 0
fi

echo "Provisioning Matrix..."
echo "Waiting for Matrix..."
apk add --no-cache curl jq >/dev/null 2>&1

MAX_WAIT=60
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    if wget -q --spider --timeout=5 "$MATRIX_HOMESERVER_URL/_matrix/client/versions" 2>/dev/null; then
        echo "Matrix ready!"
        break
    fi
    sleep 2
    WAITED=$((WAITED + 2))
done

REG=$(curl -s -X POST "$MATRIX_HOMESERVER_URL/_matrix/client/r0/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$MATRIX_ADMIN_USER\",\"password\":\"$MATRIX_ADMIN_PASSWORD\",\"auth\":{\"type\":\"m.login.dummy\"}}")

if echo "$REG" | grep -q "access_token"; then
    ADMIN_TOKEN=$(echo "$REG" | jq -r '.access_token')
else
    LOG=$(curl -s -X POST "$MATRIX_HOMESERVER_URL/_matrix/client/r0/login" \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"m.login.password\",\"user\":\"$MATRIX_ADMIN_USER\",\"password\":\"$MATRIX_ADMIN_PASSWORD\"}")
    ADMIN_TOKEN=$(echo "$LOG" | jq -r '.access_token')
fi

echo "Creating room: $ROOM_NAME"
CR=$(curl -s -X POST "$MATRIX_HOMESERVER_URL/_matrix/client/r0/createRoom" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"preset\":\"public_chat\",\"name\":\"$ROOM_NAME\",\"room_alias_name\":\"$ROOM_ALIAS\",\"visibility\":\"public\"}")

ROOM_ID=$(echo "$CR" | jq -r '.room_id')

mkdir -p /provision
cat > "$STATE_FILE" <<EOF
Provisioned: $(date)
Domain: $MATRIX_DOMAIN
Admin: $MATRIX_ADMIN_USER
Room: $ROOM_NAME ($ROOM_ID)
Alias: #$ROOM_ALIAS:$MATRIX_DOMAIN

Element X:
  Homeserver: https://$MATRIX_DOMAIN
  User: $MATRIX_ADMIN_USER
  Pass: $MATRIX_ADMIN_PASSWORD
  Room: #$ROOM_ALIAS:$MATRIX_DOMAIN
EOF

echo -e "${GREEN}Provisioning complete!${NC}"
echo ""
echo "Element X Connection:"
echo "  Homeserver: https://$MATRIX_DOMAIN"
echo "  Username: $MATRIX_ADMIN_USER"
echo "  Password: $MATRIX_ADMIN_PASSWORD"
echo "  Room: #$ROOM_ALIAS:$MATRIX_DOMAIN"
