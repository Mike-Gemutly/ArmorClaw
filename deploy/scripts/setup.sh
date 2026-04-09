#!/usr/bin/env bash
# ArmorClaw VPS Setup — validates, starts, and verifies all services
set -euo pipefail

COMPOSE_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$COMPOSE_DIR"

echo "=== ArmorClaw VPS Setup ==="

# 1. Preflight checks
echo "[1/6] Preflight checks..."

if ! command -v docker &>/dev/null; then
    echo "ERROR: docker not found"
    exit 1
fi

if ! docker compose version &>/dev/null 2>&1; then
    if ! docker-compose version &>/dev/null 2>&1; then
        echo "ERROR: docker compose not found"
        exit 1
    fi
    COMPOSE_CMD="docker-compose"
else
    COMPOSE_CMD="docker compose"
fi

if ! docker info &>/dev/null; then
    echo "ERROR: docker daemon not running"
    exit 1
fi

echo "  Docker: OK (${COMPOSE_CMD})"

# 2. Load .env
if [ ! -f .env ]; then
    echo "[2/6] Creating .env from template..."
    if [ -f .env.template ]; then
        cp .env.template .env
        echo "  Created .env from .env.template — review and customize!"
    else
        echo "  WARNING: No .env.template found, creating minimal .env"
        cat > .env <<'ENVEOF'
# ArmorClaw Environment
MATRIX_SERVER_NAME=matrix.armorclaw.com
TURN_SECRET=$(openssl rand -hex 32)
COTURN_EXTERNAL_IP=
AI_API_KEY=
AI_PROVIDER=openrouter
ARMORCLAW_ADMIN_TOKEN=$(openssl rand -hex 16)
ARMORCLAW_KEYSTORE_SECRET=$(openssl rand -hex 32)
ARMORCLAW_MATRIX_SECRET=$(openssl rand -hex 32)
ENVEOF
        echo "  Created .env — REVIEW AND SET SECRETS BEFORE PROCEEDING"
    fi
else
    echo "[2/6] .env exists — skipping"
fi

# Check required vars
source .env 2>/dev/null || true
MISSING=0
for VAR in TURN_SECRET AI_API_KEY; do
    if [ -z "${!VAR:-}" ]; then
        echo "  WARNING: $VAR is not set in .env"
        MISSING=$((MISSING + 1))
    fi
done
if [ "$MISSING" -gt 0 ]; then
    echo "  Fix $MISSING missing variable(s) in .env, then re-run"
    exit 1
fi
echo "  Environment: OK"

# 3. Create network if missing
echo "[3/6] Ensuring Docker network..."
if ! docker network inspect armorclaw-matrix >/dev/null 2>&1; then
    docker network create armorclaw-matrix --subnet 172.26.0.0/24
    echo "  Created network: armorclaw-matrix"
else
    echo "  Network exists: armorclaw-matrix"
fi

# 4. Start Matrix services
echo "[4/6] Starting Matrix services..."
$COMPOSE_CMD -f docker-compose.yml -f docker-compose.matrix.yml up -d
echo "  Matrix services started"

# 5. Start Bridge (systemd or native)
echo "[5/6] Starting Bridge..."
if systemctl is-active --quiet armorclaw-bridge 2>/dev/null; then
    sudo systemctl restart armorclaw-bridge
    echo "  Restarted systemd service: armorclaw-bridge"
elif [ -f /run/armorclaw/bridge.sock ]; then
    echo "  WARNING: Unix socket exists but no systemd service — manual restart needed"
else
    echo "  No bridge service detected — start it manually or via systemd"
fi

# 6. Wait for health
echo "[6/6] Verifying services..."
OK=true

# Check Matrix
for i in $(seq 1 15); do
    if curl -sf http://localhost:6167/_matrix/client/versions >/dev/null 2>&1; then
        echo "  Matrix: OK (port 6167)"
        break
    fi
    if [ "$i" -eq 15 ]; then
        echo "  Matrix: FAIL (port 6167 not responding)"
        OK=false
    fi
    sleep 1
done

# Check Bridge health (new /health endpoint on RPC port)
BRIDGE_PORT="${ARMORCLAW_BRIDGE_HOST_PORT:-8081}"
for i in $(seq 1 15); do
    if curl -sf "http://localhost:${BRIDGE_PORT}/health" >/dev/null 2>&1; then
        echo "  Bridge: OK (port ${BRIDGE_PORT})"
        break
    fi
    if [ "$i" -eq 15 ]; then
        echo "  Bridge: FAIL (port ${BRIDGE_PORT} not responding)"
        OK=false
    fi
    sleep 1
done

echo ""
if [ "$OK" = true ]; then
    echo "=== All services ready ==="
else
    echo "=== Some services failed — check logs above ==="
    exit 1
fi
