#!/usr/bin/env bash
# =============================================================================
# ArmorClaw Unified Installer
# Version: 1.0.0
# Idempotent: Yes
# Safe to re-run: Yes
# =============================================================================
#
# Usage:
#   sudo ./deploy/install.sh
#   curl -fsSL https://install.armorclaw.com | bash
#
# Options:
#   --yes              Non-interactive mode
#   --domain DOMAIN    Set domain (optional)
#   --matrix           Deploy Matrix stack
#   --no-matrix        Skip Matrix (default)
#   --ai-stack         Deploy AI stack (Catwalk)
#   --no-ai-stack      Skip AI stack (default)
#   --help             Show help
#
# Environment Variables:
#   ARMORCLAW_MATRIX      true/false - Deploy Matrix stack
#   ARMORCLAW_AI_STACK    true/false - Deploy AI stack (default: false)
#   ARMORCLAW_DOMAIN      Domain name (optional)
#   ARMORCLAW_API_KEY     AI provider API key
#   ARMORCLAW_PROVIDER    AI provider (openai, anthropic, etc.)
# =============================================================================

set -euo pipefail

# =============================================================================
# Constants
# =============================================================================
readonly VERSION="1.0.0"
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default settings (SECURE DEFAULTS)
ARMORCLAW_MATRIX="${ARMORCLAW_MATRIX:-false}"
ARMORCLAW_AI_STACK="${ARMORCLAW_AI_STACK:-false}"
ARMORCLAW_DOMAIN="${ARMORCLAW_DOMAIN:-}"
ARMORCLAW_API_KEY="${ARMORCLAW_API_KEY:-}"
ARMORCLAW_PROVIDER="${ARMORCLAW_PROVIDER:-}"
NON_INTERACTIVE="${NON_INTERACTIVE:-false}"

# =============================================================================
# Error Handling
# =============================================================================
trap 'echo "[ERROR] Installer failed on line $LINENO: $BASH_COMMAND"' ERR

# =============================================================================
# Helper Functions
# =============================================================================
log_info()    { echo "[INFO] $*"; }
log_success() { echo "[SUCCESS] $*"; }
log_warn()    { echo "[WARN] $*"; }
log_error()   { echo "[ERROR] $*" >&2; exit 1; }

usage() {
    cat <<EOF
ArmorClaw Unified Installer v${VERSION}

Usage: $0 [OPTIONS]

OPTIONS:
    --yes              Non-interactive mode
    --domain DOMAIN    Set domain (optional, enables TLS)
    --matrix           Deploy Matrix stack
    --no-matrix        Skip Matrix deployment (default)
    --ai-stack         Deploy AI stack (Catwalk)
    --no-ai-stack      Skip AI stack (default)
    --help             Show this help

ENVIRONMENT VARIABLES:
    ARMORCLAW_MATRIX     true/false - Deploy Matrix stack
    ARMORCLAW_AI_STACK   true/false - Deploy AI stack (default: false)
    ARMORCLAW_DOMAIN     Domain name
    ARMORCLAW_API_KEY    AI provider API key
    ARMORCLAW_PROVIDER   AI provider (openai, anthropic, etc.)

EXAMPLES:
    # Interactive install
    sudo ./install.sh

    # Non-interactive with Matrix
    ARMORCLAW_MATRIX=true ARMORCLAW_DOMAIN=matrix.example.com sudo ./install.sh

    # Minimal install (bridge only)
    ARMORCLAW_MATRIX=false ARMORCLAW_AI_STACK=false sudo ./install.sh
EOF
    exit 0
}

# =============================================================================
# Prerequisites Check
# =============================================================================
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Must run as root (use sudo)"
    fi
}

check_docker() {
    log_info "Checking Docker installation..."
    
    if ! command -v docker >/dev/null 2>&1; then
        log_error "Docker not installed. Install with: curl -fsSL https://get.docker.com | sh"
    fi

    if ! docker info >/dev/null 2>&1; then
        log_error "Docker daemon not running. Run: sudo systemctl start docker"
    fi

    if ! docker compose version >/dev/null 2>&1; then
        log_error "Docker Compose plugin missing. Install docker-compose-plugin"
    fi
    
    log_success "Docker is ready"
}

wait_for_docker() {
    log_info "Waiting for Docker daemon..."
    
    for i in {1..15}; do
        if docker info >/dev/null 2>&1; then
            log_success "Docker daemon ready"
            return 0
        fi
        
        if [ "$i" -eq 15 ]; then
            log_error "Docker daemon failed to start"
        fi
        
        log_info "Waiting for Docker... ($i/15)"
        sleep 2
    done
}

# =============================================================================
# Idempotency Checks
# =============================================================================
check_bridge_running() {
    systemctl is-active --quiet armorclaw-bridge 2>/dev/null
}

check_matrix_running() {
    docker ps --filter "name=^armorclaw-conduit$" --format '{{.Names}}' 2>/dev/null | grep -q '^armorclaw-conduit$'
}

check_catwalk_running() {
    docker ps --filter "name=^armorclaw-catwalk$" --format '{{.Names}}' 2>/dev/null | grep -q '^armorclaw-catwalk$'
}

# =============================================================================
# Deployment Functions
# =============================================================================
deploy_bridge() {
    if check_bridge_running; then
        log_info "Bridge already running"
        return 0
    fi
    
    log_info "Deploying ArmorClaw Bridge..."
    
    if [ -f "$SCRIPT_DIR/setup-quick.sh" ]; then
        bash "$SCRIPT_DIR/setup-quick.sh"
        log_success "Bridge deployed"
    else
        log_error "setup-quick.sh not found at $SCRIPT_DIR/setup-quick.sh"
    fi
}

deploy_matrix() {
    if check_matrix_running; then
        log_info "Matrix stack already running"
        return 0
    fi
    
    log_info "Deploying Matrix stack..."
    
    if [ -f "$SCRIPT_DIR/setup-matrix.sh" ]; then
        bash "$SCRIPT_DIR/setup-matrix.sh"
        verify_matrix_health
        log_success "Matrix stack deployed"
    else
        log_error "setup-matrix.sh not found at $SCRIPT_DIR/setup-matrix.sh"
    fi
}

deploy_ai_stack() {
    if check_catwalk_running; then
        log_info "AI stack already running"
        return 0
    fi
    
    log_info "Deploying AI stack (Catwalk)..."
    
    local compose_file="$SCRIPT_DIR/ai/docker-compose.ai.yml"
    
    if [ -f "$compose_file" ]; then
        docker compose \
            --project-name armorclaw-ai \
            -f "$compose_file" \
            up -d
        
        verify_catwalk_health
        log_success "AI stack deployed"
    else
        log_warn "AI stack compose file not found: $compose_file"
        log_info "Skipping AI stack deployment"
    fi
}

# =============================================================================
# Health Verification
# =============================================================================
verify_matrix_health() {
    log_info "Verifying Matrix health..."
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:6167/_matrix/client/versions &>/dev/null; then
            log_success "Matrix stack healthy"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    
    log_warn "Matrix health check failed - service may still be starting"
}

verify_catwalk_health() {
    log_info "Verifying Catwalk health..."
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:8080/healthz &>/dev/null; then
            log_success "Catwalk healthy"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    
    log_warn "Catwalk health check failed - AI provider discovery may be limited"
}

verify_bridge_health() {
    # HTTP check
    if curl -sf http://localhost:8443/health &>/dev/null; then
        return 0
    fi
    
    # Socket fallback
    if [ -S /run/armorclaw/bridge.sock ]; then
        return 0
    fi
    
    return 1
}

wait_for_bridge() {
    log_info "Waiting for bridge to initialize..."
    
    for i in {1..20}; do
        if verify_bridge_health; then
            log_success "Bridge ready"
            return 0
        fi
        log_info "Waiting for bridge... ($i/20)"
        sleep 2
    done
    
    log_warn "Bridge health check failed"
}

# =============================================================================
# Service Management
# =============================================================================
start_bridge_service() {
    if ! systemctl is-enabled armorclaw-bridge >/dev/null 2>&1; then
        log_warn "armorclaw-bridge service not installed"
        log_info "Run setup-quick.sh first to install the bridge service"
        return 1
    fi
    
    if systemctl is-active --quiet armorclaw-bridge; then
        log_info "Bridge already running"
        return 0
    fi
    
    log_info "Starting bridge service..."
    systemctl start armorclaw-bridge
}

# =============================================================================
# AI Provider Setup
# =============================================================================
setup_ai_provider() {
    if [ -z "$ARMORCLAW_API_KEY" ]; then
        log_info "No API key provided - skip AI provider setup"
        log_info "Add keys later with: armorclaw-bridge add-key --provider <provider> --token <key>"
        return 0
    fi
    
    log_info "Setting up AI provider..."
    
    local provider="${ARMORCLAW_PROVIDER:-openai}"
    
    if command -v armorclaw-bridge &>/dev/null; then
        armorclaw-bridge add-key --provider "$provider" --token "$ARMORCLAW_API_KEY"
        log_success "AI provider configured: $provider"
    else
        log_warn "armorclaw-bridge not found in PATH"
        log_info "Add API key manually after bridge is installed"
    fi
}

# =============================================================================
# Parse Arguments
# =============================================================================
parse_args() {
    while [ $# -gt 0 ]; do
        case $1 in
            --yes)
                NON_INTERACTIVE=true
                shift
                ;;
            --domain)
                ARMORCLAW_DOMAIN="$2"
                shift 2
                ;;
            --matrix)
                ARMORCLAW_MATRIX=true
                shift
                ;;
            --no-matrix)
                ARMORCLAW_MATRIX=false
                shift
                ;;
            --ai-stack)
                ARMORCLAW_AI_STACK=true
                shift
                ;;
            --no-ai-stack)
                ARMORCLAW_AI_STACK=false
                shift
                ;;
            --help)
                usage
                ;;
            *)
                echo "Unknown option: $1"
                usage
                ;;
        esac
    done
}

# =============================================================================
# Main
# =============================================================================
main() {
    # Banner
    echo ""
    echo "===================================="
    echo " ArmorClaw Installer v$VERSION"
    echo "===================================="
    echo ""
    
    log_info "Installation started at $(date)"
    
    # Create lock directory
    mkdir -p /var/lock
    
    # Prevent concurrent runs
    exec 200>/var/lock/armorclaw-install.lock
    flock -n 200 || log_error "Installer already running. Multiple instances not allowed."
    
    # Prerequisites
    check_root
    check_docker
    wait_for_docker
    
    # Deploy Bridge
    deploy_bridge
    
    # Deploy Matrix (optional)
    if [ "$ARMORCLAW_MATRIX" = "true" ]; then
        deploy_matrix
    else
        log_info "Skipping Matrix deployment (--matrix not specified)"
    fi
    
    # Deploy AI Stack (optional)
    if [ "$ARMORCLAW_AI_STACK" = "true" ]; then
        deploy_ai_stack
    else
        log_info "Skipping AI stack (--ai-stack not specified)"
    fi
    
    # Setup AI Provider
    setup_ai_provider
    
    # Start services
    start_bridge_service || true
    
    # Wait for bridge
    wait_for_bridge
    
    # Success footer
    echo ""
    echo "===================================="
    echo " ArmorClaw installation complete"
    echo "===================================="
    echo ""
    
    log_info "Installation completed at $(date)"
    
    # Show next steps
    echo "Next steps:"
    echo ""
    echo "  1. Check bridge status: systemctl status armorclaw-bridge"
    
    if [ "$ARMORCLAW_MATRIX" = "true" ]; then
        echo "  2. Check Matrix status: docker ps | grep armorclaw"
        echo "  3. Connect to Matrix: http://localhost:6167"
    fi
    
    if [ "$ARMORCLAW_AI_STACK" = "true" ]; then
        echo "  4. Check Catwalk status: curl http://localhost:8080/healthz"
    fi
    
    echo ""
    echo "Health check: $SCRIPT_DIR/health-check.sh"
    echo ""
}

# =============================================================================
# Entry Point
# =============================================================================
parse_args "$@"
main
