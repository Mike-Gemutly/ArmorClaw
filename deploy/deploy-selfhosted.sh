#!/usr/bin/env bash
# ArmorClaw Self-Hosted Appliance Setup
# Appliance-mode deployment for single VPS or home server
#
# Usage:
#   sudo ./deploy/deploy-selfhosted.sh              # interactive
#   sudo ./deploy/deploy-selfhosted.sh --auto       # non-interactive
#   sudo ./deploy/deploy-selfhosted.sh --dry-run    # show plan only

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Defaults
AUTO_MODE=false
DRY_RUN=false
LAN_MODE=false
HOSTNAME=""
LAN_IP=""
PUBLIC_IP=""
CERT_OUTPUT_DIR="/etc/armorclaw/certs"
COMPOSE_FILE="docker-compose.selfhosted.yml"

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Self-Hosted Appliance Setup${NC}             ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}Single VPS / Home Server Deployment${NC}              ${CYAN}║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "\n${BLUE}▶${NC} ${BOLD}$1${NC}"
    echo -e "${BLUE}  ─────────────────────────────────────${NC}"
}

print_success() {
    echo -e "  ${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "  ${RED}✗${NC} ${BOLD}ERROR:${NC} $1" >&2
}

print_warning() {
    echo -e "  ${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "  ${CYAN}ℹ${NC} $1"
}

fail() {
    print_error "$1"
    exit 1
}

#=============================================================================
# 1. Parse Arguments
#=============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --auto)
                AUTO_MODE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help|-h)
                echo "Usage: sudo ./deploy/deploy-selfhosted.sh [--auto] [--dry-run]"
                echo ""
                echo "Options:"
                echo "  --auto      Non-interactive mode (use auto-detected values)"
                echo "  --dry-run   Show what would be done without executing"
                echo "  --help      Show this help message"
                exit 0
                ;;
            *)
                fail "Unknown argument: $1 (use --help for usage)"
                ;;
        esac
    done
}

#=============================================================================
# 2. Detect LAN Mode
#=============================================================================

detect_network() {
    print_step "Detecting network configuration..."

    PUBLIC_IP=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || echo "")
    LAN_IP=$(hostname -I 2>/dev/null | awk '{print $1}')

    if [[ -z "$PUBLIC_IP" ]]; then
        LAN_MODE=true
        print_info "No public IP detected — LAN mode enabled"
    else
        LAN_MODE=false
        print_info "Public IP detected: $PUBLIC_IP"
    fi

    if [[ -z "$LAN_IP" ]]; then
        LAN_IP="127.0.0.1"
        print_warning "Could not detect LAN IP — using 127.0.0.1"
    else
        print_success "LAN IP: $LAN_IP"
    fi

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] LAN_MODE=$LAN_MODE, PUBLIC_IP=${PUBLIC_IP:-none}, LAN_IP=$LAN_IP"
    fi
}

#=============================================================================
# 3. Auto-detect Hostname
#=============================================================================

detect_hostname() {
    print_step "Auto-detecting hostname..."

    HOSTNAME=$(hostname -f 2>/dev/null || hostname)

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Detected hostname: $HOSTNAME"
        return
    fi

    print_success "Hostname: $HOSTNAME"
}

#=============================================================================
# 4. Prerequisites Check
#=============================================================================

check_prerequisites() {
    print_step "Checking prerequisites..."

    local missing=false

    # Docker
    if command -v docker &>/dev/null; then
        print_success "Docker found: $(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')"
    else
        print_error "Docker is not installed"
        missing=true
    fi

    # Docker Compose
    if docker compose version &>/dev/null; then
        print_success "Docker Compose found: $(docker compose version --short 2>/dev/null)"
    else
        print_error "Docker Compose is not available (needs 'docker compose' plugin)"
        missing=true
    fi

    # openssl
    if command -v openssl &>/dev/null; then
        print_success "openssl found: $(openssl version 2>/dev/null | awk '{print $2}')"
    else
        print_error "openssl is not installed"
        missing=true
    fi

    # Avahi (non-fatal)
    if command -v avahi-daemon &>/dev/null || command -v avahi-publish &>/dev/null; then
        print_success "Avahi found (mDNS support)"
    else
        print_warning "Avahi not installed — mDNS discovery unavailable"
        print_warning "  Install with: sudo apt install avahi-daemon"
    fi

    if $missing; then
        fail "Required prerequisites missing. Install them and re-run."
    fi

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] All critical prerequisites met"
    fi
}

#=============================================================================
# 5. Create .env.selfhosted
#=============================================================================

create_env_file() {
    print_step "Creating .env.selfhosted..."

    local env_file="$PROJECT_ROOT/.env.selfhosted"

    if [[ -f "$env_file" ]]; then
        print_info ".env.selfhosted already exists — preserving existing configuration"
        return
    fi

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would create $env_file with:"
        print_info "  AI_API_KEY=\${AI_API_KEY:-}"
        print_info "  AI_PROVIDER=\${AI_PROVIDER:-openrouter}"
        print_info "  ARMORCLAW_HOSTNAME=$HOSTNAME"
        print_info "  ADMIN_EMAIL=admin@$HOSTNAME"
        return
    fi

    cat > "$env_file" << EOF
# ArmorClaw Self-Hosted Configuration
# Generated by deploy-selfhosted.sh on $(date -Iseconds 2>/dev/null || date)
AI_API_KEY=${AI_API_KEY:-}
AI_PROVIDER=${AI_PROVIDER:-openrouter}
ARMORCLAW_HOSTNAME=${HOSTNAME}
ADMIN_EMAIL=admin@${HOSTNAME}
EOF

    print_success "Created $env_file"
}

#=============================================================================
# 6. Generate Self-Signed Certificates
#=============================================================================

generate_certs() {
    print_step "Generating self-signed certificates..."

    local cert_script="$SCRIPT_DIR/scripts/generate-certs.sh"

    if [[ ! -f "$cert_script" ]]; then
        fail "Certificate generator not found: $cert_script"
    fi

    if [[ -f "$CERT_OUTPUT_DIR/server.crt" ]] && [[ -f "$CERT_OUTPUT_DIR/server.key" ]]; then
        print_info "Certificates already exist at $CERT_OUTPUT_DIR — skipping generation"
        return
    fi

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would run:"
        print_info "  bash $cert_script --output $CERT_OUTPUT_DIR --hostname $HOSTNAME --lan-ip $LAN_IP"
        return
    fi

    bash "$cert_script" --output "$CERT_OUTPUT_DIR" --hostname "$HOSTNAME" --lan-ip "$LAN_IP"
    print_success "Certificates generated at $CERT_OUTPUT_DIR"
}

#=============================================================================
# 7. Build / Pull Docker Images
#=============================================================================

build_images() {
    print_step "Building / pulling Docker images..."

    if [[ ! -f "$PROJECT_ROOT/$COMPOSE_FILE" ]]; then
        fail "Docker Compose file not found: $PROJECT_ROOT/$COMPOSE_FILE"
    fi

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would run:"
        print_info "  docker compose -f $COMPOSE_FILE pull"
        print_info "  docker compose -f $COMPOSE_FILE build"
        return
    fi

    docker compose -f "$PROJECT_ROOT/$COMPOSE_FILE" pull 2>/dev/null || true
    docker compose -f "$PROJECT_ROOT/$COMPOSE_FILE" build 2>/dev/null || true

    print_success "Docker images ready"
}

#=============================================================================
# 8. Start Docker Stack
#=============================================================================

start_stack() {
    print_step "Starting Docker stack..."

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would run:"
        print_info "  docker compose -f $COMPOSE_FILE up -d"
        return
    fi

    docker compose -f "$PROJECT_ROOT/$COMPOSE_FILE" up -d
    print_success "Docker stack started"
}

#=============================================================================
# 9. Wait for Health Checks
#=============================================================================

wait_for_health() {
    print_step "Waiting for services to start..."

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would wait 10s and run:"
        print_info "  docker compose -f $COMPOSE_FILE ps"
        return
    fi

    local waited=0
    local max_wait=30

    while [[ $waited -lt $max_wait ]]; do
        sleep 5
        waited=$((waited + 5))
        print_info "Waiting... ($waited/${max_wait}s)"
    done

    echo ""
    docker compose -f "$PROJECT_ROOT/$COMPOSE_FILE" ps
    print_success "Services status displayed above"
}

#=============================================================================
# 10. Print Discovery Info
#=============================================================================

print_discovery() {
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}                 ${BOLD}Self-Hosted Appliance Ready!${NC}                     ${GREEN}║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${BOLD}Discovery Information:${NC}"
    echo ""

    # mDNS hostname
    echo -e "  ${CYAN}mDNS Hostname:${NC}  $HOSTNAME"

    # LAN IP
    echo -e "  ${CYAN}LAN IP:${NC}          $LAN_IP"

    # Public IP (if available)
    if [[ -n "$PUBLIC_IP" ]]; then
        echo -e "  ${CYAN}Public IP:${NC}       $PUBLIC_IP"
    fi

    # LAN mode indicator
    if [[ "$LAN_MODE" == true ]]; then
        echo -e "  ${CYAN}Mode:${NC}            LAN (no public IP)"
    else
        echo -e "  ${CYAN}Mode:${NC}            Public VPS"
    fi

    # CA fingerprint
    if [[ -f "$CERT_OUTPUT_DIR/ca.crt" ]]; then
        local ca_fingerprint
        ca_fingerprint=$(openssl x509 -in "$CERT_OUTPUT_DIR/ca.crt" -noout -fingerprint -sha256 2>/dev/null | cut -d= -f2 || echo "unavailable")
        echo -e "  ${CYAN}CA Fingerprint:${NC}  $ca_fingerprint"
    fi

    # Quick connect URL
    echo ""
    echo -e "${BOLD}Quick Connect:${NC}"
    echo -e "  ${GREEN}https://$HOSTNAME${NC}"
    if [[ "$LAN_MODE" == true ]]; then
        echo -e "  ${GREEN}https://$LAN_IP${NC}"
    fi

    # Cert note
    echo ""
    echo -e "${YELLOW}⚠${NC} ${BOLD}Note:${NC} Self-signed certificates are used. Your browser will show a security"
    echo -e "  warning. This is expected — accept the certificate to proceed."
}

#=============================================================================
# 11. Print Next Steps
#=============================================================================

print_next_steps() {
    echo ""
    echo -e "${BOLD}Next Steps:${NC}"
    echo ""
    echo -e "  1. Install ArmorChat from Google Play"
    echo -e "  2. Open the app and scan the QR code from the bridge"
    echo -e "  3. Or manually enter: ${CYAN}https://$HOSTNAME${NC}"
    if [[ "$LAN_MODE" == true ]]; then
        echo -e "  4. LAN access: ${CYAN}https://$LAN_IP${NC}"
    fi
    echo -e "  5. For email setup: run ${CYAN}deploy/postfix/install.sh${NC} to configure Postfix"
    echo ""
    echo -e "${BOLD}Useful Commands:${NC}"
    echo ""
    echo -e "  ${CYAN}docker compose -f $COMPOSE_FILE logs -f${NC}    # View logs"
    echo -e "  ${CYAN}docker compose -f $COMPOSE_FILE ps${NC}          # Check status"
    echo -e "  ${CYAN}docker compose -f $COMPOSE_FILE down${NC}        # Stop stack"
    echo ""
}

#=============================================================================
# Dry-Run Summary
#=============================================================================

print_dry_run_summary() {
    echo ""
    echo -e "${YELLOW}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║${NC}                 ${BOLD}Dry Run Complete${NC}                                 ${YELLOW}║${NC}"
    echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}Planned Actions:${NC}"
    echo ""
    echo -e "  Hostname:        ${CYAN}$HOSTNAME${NC}"
    echo -e "  LAN IP:          ${CYAN}$LAN_IP${NC}"
    echo -e "  Public IP:       ${CYAN}${PUBLIC_IP:-none}${NC}"
    echo -e "  LAN Mode:        ${CYAN}$LAN_MODE${NC}"
    echo -e "  Cert output:     ${CYAN}$CERT_OUTPUT_DIR${NC}"
    echo -e "  Compose file:    ${CYAN}$COMPOSE_FILE${NC}"
    echo -e "  Env file:        ${CYAN}$PROJECT_ROOT/.env.selfhosted${NC}"
    echo ""
    echo -e "Run without ${BOLD}--dry-run${NC} to execute."
    echo ""
}

#=============================================================================
# Main
#=============================================================================

main() {
    parse_args "$@"
    print_header

    if [[ "$DRY_RUN" == true ]]; then
        print_info "Running in DRY-RUN mode — no changes will be made"
    elif [[ "$AUTO_MODE" == true ]]; then
        print_info "Running in AUTO mode — using auto-detected values"
    fi

    detect_network
    detect_hostname
    check_prerequisites
    create_env_file
    generate_certs
    build_images
    start_stack
    wait_for_health

    if [[ "$DRY_RUN" == true ]]; then
        print_dry_run_summary
    else
        print_discovery
        print_next_steps
    fi
}

main "$@"
