#!/bin/bash
# ArmorClaw Unified Deployment Script

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ROLLBACK_FILE="$PROJECT_ROOT/.rollback"

cd "$PROJECT_ROOT"

trap rollback ERR

rollback() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        echo -e "${RED}Deployment Failed - Rolling Back${NC}"
        if [[ -f "$ROLLBACK_FILE" ]]; then
            source "$ROLLBACK_FILE"
            if [[ -n "$DOCKER_COMPOSE_FILE" ]]; then
                docker-compose -f "$DOCKER_COMPOSE_FILE" down 2>/dev/null || true
            fi
        fi
        echo -e "${YELLOW}Rollback complete${NC}"
    fi
    rm -f "$ROLLBACK_FILE"
    exit $exit_code
}

log_step() {
    echo ""
    echo -e "${BLUE}=== $1 ===${NC}"
}

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

echo -e "${BLUE}ArmorClaw Unified Deployment${NC}"
echo ""

SKIP_CHECKS=false
SKIP_BUILD=false
COMPOSE_FILE="docker-compose-stack.yml"
PROVISION=true
RUN_SMOKE_TEST=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-checks) SKIP_CHECKS=true; shift ;;
        --skip-build) SKIP_BUILD=true; shift ;;
        --compose) COMPOSE_FILE="$2"; shift 2 ;;
        --no-provision) PROVISION=false; shift ;;
        --no-test) RUN_SMOKE_TEST=false; shift ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options: --skip-checks --skip-build --compose FILE --no-provision --no-test"
            exit 0
            ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

echo "DOCKER_COMPOSE_FILE=\"$COMPOSE_FILE\"" > "$ROLLBACK_FILE"

# Phase 1: Prerequisites
log_step "Phase 1: Prerequisites"
if [[ "$SKIP_CHECKS" == "false" && -x "$SCRIPT_DIR/check-prerequisites.sh" ]]; then
    "$SCRIPT_DIR/check-prerequisites.sh"
fi

# Phase 2: Configuration
log_step "Phase 2: Configuration"
if [[ ! -f .env ]]; then
    if [[ -f .env.example ]]; then
        cp .env.example .env
        ADMIN_PASS=$(openssl rand -base64 16 | tr -d '/+=')
        BRIDGE_PASS=$(openssl rand -base64 16 | tr -d '/+=')
        sed -i "s/MATRIX_ADMIN_PASSWORD=.*/MATRIX_ADMIN_PASSWORD=$ADMIN_PASS/" .env
        sed -i "s/MATRIX_BRIDGE_PASSWORD=.*/MATRIX_BRIDGE_PASSWORD=$BRIDGE_PASS/" .env
        log_success ".env created"
    fi
fi

# Phase 3: Container Build
log_step "Phase 3: Container"
if [[ "$SKIP_BUILD" == "false" ]]; then
    if ! docker images | grep -q "armorclaw/agent"; then
        docker build -t armorclaw/agent:v1 . || true
        log_success "Container built"
    fi
fi

# Phase 4: Start Services
log_step "Phase 4: Starting Services"
if [[ ! -f "$COMPOSE_FILE" ]]; then
    log_error "Compose file not found: $COMPOSE_FILE"
    exit 1
fi
docker-compose -f "$COMPOSE_FILE" down 2>/dev/null || true
docker-compose -f "$COMPOSE_FILE" up -d
sleep 15

SERVICES_UP=$(docker-compose -f "$COMPOSE_FILE" ps --services --filter "status=running" | wc -l)
if [[ "$SERVICES_UP" -ge 2 ]]; then
    log_success "Services started ($SERVICES_UP running)"
fi

# Phase 5: Provisioning
if [[ "$PROVISION" == "true" ]]; then
    log_step "Phase 5: Provisioning"
    docker-compose -f "$COMPOSE_FILE" run --rm provision 2>/dev/null || log_warn "Provisioning may have run already"
fi

# Phase 6: Tests
if [[ "$RUN_SMOKE_TEST" == "true" ]]; then
    log_step "Phase 6: Tests"
    curl -sf http://localhost:6167/_matrix/client/versions >/dev/null && log_success "Matrix OK" || log_error "Matrix down"
    curl -sf http://localhost/ >/dev/null && log_success "HTTP OK" || log_warn "HTTP down"
fi

rm -f "$ROLLBACK_FILE"
echo ""
echo -e "${GREEN}Deployment Complete!${NC}"
docker-compose -f "$COMPOSE_FILE" ps
exit 0
