#!/usr/bin/env bash
# ArmorClaw VPS Reset — stops all services and cleans state
set -euo pipefail

COMPOSE_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$COMPOSE_DIR"

echo "=== ArmorClaw VPS Reset ==="

# Detect compose command
if docker compose version &>/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
elif docker-compose version &>/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
else
    echo "ERROR: docker compose not found"
    exit 1
fi

# Stop Matrix services
echo "[1/4] Stopping Matrix services..."
$COMPOSE_CMD -f docker-compose.yml -f docker-compose.matrix.yml -f docker-compose.bridge.yml down 2>/dev/null || \
    $COMPOSE_CMD -f docker-compose.yml -f docker-compose.matrix.yml down 2>/dev/null || true
echo "  Matrix services stopped"

# Stop Bridge
echo "[2/4] Stopping Bridge..."
if systemctl is-active --quiet armorclaw-bridge 2>/dev/null; then
    sudo systemctl stop armorclaw-bridge
    echo "  Stopped systemd service: armorclaw-bridge"
else
    echo "  No systemd service — skipping"
fi

# Clean socket
echo "[3/4] Cleaning sockets..."
sudo rm -f /run/armorclaw/bridge.sock 2>/dev/null || true
echo "  Sockets cleaned"

# Optionally remove network
echo "[4/4] Network cleanup..."
if docker network inspect armorclaw-matrix >/dev/null 2>&1; then
    docker network rm armorclaw-matrix 2>/dev/null || true
    echo "  Removed network: armorclaw-matrix"
else
    echo "  Network does not exist — skipping"
fi

echo ""
echo "=== Reset complete. Run deploy/scripts/setup.sh to restart ==="
