#!/usr/bin/env bash
# ArmorClaw Cloudflare HTTPS Setup - Standalone Script
# =============================================================================
# This script provides a standalone interface for setting up Cloudflare HTTPS
# for ArmorClaw deployments. It supports both Tunnel and Proxy modes.
#
# Usage: ./deploy/setup-cloudflare.sh [OPTIONS]
#
# This script sources cloudflare-functions.sh and provides a CLI interface
# for non-interactive use with flags like --domain, --mode, and --dry-run.
#
# Environment Variables:
#   CF_API_TOKEN   - Cloudflare API token (required for proxy mode)
#   CONNECT_VPS    - SSH command to connect to VPS (for tunnel mode)
#   VPS_IP         - VPS IP address (alternative to CONNECT_VPS)
#
# Example:
#   ./deploy/setup-cloudflare.sh --domain armorclaw.example.com --mode tunnel
#   ./deploy/setup-cloudflare.sh --domain example.com --mode proxy --dry-run
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "${SCRIPT_DIR}/lib/cloudflare-functions.sh" ]; then
    source "${SCRIPT_DIR}/lib/cloudflare-functions.sh"
else
    echo "ERROR: cloudflare-functions.sh not found at ${SCRIPT_DIR}/lib/cloudflare-functions.sh"
    exit 1
fi

DOMAIN=""
MODE=""
DRY_RUN="${DRY_RUN:-false}"
export DRY_RUN
HELP=false

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

print_usage() {
    cat << EOF
${BOLD}ArmorClaw Cloudflare HTTPS Setup${NC}
${BOLD}=================================${NC}

${CYAN}Usage:${NC}
  $0 [OPTIONS]

${CYAN}Options:${NC}
  ${BOLD}--domain DOMAIN${NC}        Domain name (e.g., armorclaw.example.com)
  ${BOLD}--mode tunnel|proxy${NC}     Deployment mode
                              ${YELLOW}tunnel${NC}  - Cloudflare Tunnel (cloudflared)
                              ${YELLOW}proxy${NC}   - Cloudflare DNS/CDN proxy
  ${BOLD}--dry-run${NC}               Run in dry-run mode (no changes made)
  ${BOLD}--help, -h${NC}              Show this help message

${CYAN}Environment Variables:${NC}
  ${BOLD}CF_API_TOKEN${NC}          Cloudflare API token (required for proxy mode)
                              Get from: https://dash.cloudflare.com/profile/api-tokens
                              Permissions needed: Zone - DNS, Zone - SSL
  ${BOLD}CONNECT_VPS${NC}           SSH command to connect to VPS (for tunnel mode)
                              Example: export CONNECT_VPS='ssh root@1.2.3.4'
  ${BOLD}VPS_IP${NC}                VPS IP address (alternative to CONNECT_VPS)
                              Example: export VPS_IP=1.2.3.4

${CYAN}Examples:${NC}
  # Interactive mode (prompts for domain and mode)
  $0

  # Non-interactive with tunnel mode
  $0 --domain armorclaw.example.com --mode tunnel

  # Non-interactive with proxy mode (requires CF_API_TOKEN)
  export CF_API_TOKEN=your_api_token
  $0 --domain example.com --mode proxy

  # Dry-run mode (testing without making changes)
  $0 --domain example.com --mode tunnel --dry-run

${CYAN}Mode Selection:${NC}
  ${BOLD}Tunnel Mode${NC} (${YELLOW}Recommended for NAT/CGNAT${NC})
    • No port forwarding required
    • Outbound tunnel to Cloudflare
    • Best for: NAT/CGNAT, dynamic IP, no public ports
    • Requires: cloudflared authentication (cloudflared tunnel login)

  ${BOLD}Proxy Mode${NC} (${YELLOW}Recommended for static public IP${NC})
    • Requires ports 80 and 443 to be accessible
    • Uses Cloudflare DNS/CDN proxy
    • Best for: Static public IP, port forwarding, own domain
    • Requires: CF_API_TOKEN, Cloudflare nameservers

${CYAN}After Setup:${NC}
  Tunnel mode exports:
    • PUBLIC_URL=https://your-domain.com
    • MATRIX_URL=https://matrix.your-domain.com
    • DOMAIN=your-domain.com

  Proxy mode exports:
    • PUBLIC_URL=https://your-domain.com
    • MATRIX_URL=https://matrix.your-domain.com
    • DOMAIN=your-domain.com
    • CF_ZONE_ID=your-zone-id

${CYAN}Next Steps:${NC}
  1. Set SSL/TLS mode to ${GREEN}Full (strict)${NC} in Cloudflare dashboard (proxy mode)
  2. Restart Caddy to apply configuration (proxy mode)
  3. Verify your site is accessible at https://your-domain.com

EOF
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --mode)
                MODE="$2"
                if [[ "$MODE" != "tunnel" && "$MODE" != "proxy" ]]; then
                    log_error "Invalid mode: $MODE (must be 'tunnel' or 'proxy')"
                    print_usage
                    exit 1
                fi
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                export DRY_RUN
                shift
                ;;
            --help|-h)
                HELP=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
}

validate_dry_run() {
    if [ "$DRY_RUN" = true ]; then
        log_info "Running in DRY-RUN mode - no changes will be made"
        log_info "This will simulate the setup process without creating tunnels, DNS records, or certificates"
        echo ""
    fi
}

main() {
    parse_args "$@"

    if [ "$HELP" = true ]; then
        print_usage
        exit 0
    fi

    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}         ArmorClaw Cloudflare HTTPS Setup${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""

    if ! check_cloudflare_prerequisites; then
        log_error "Missing required prerequisites"
        echo ""
        log_info "Please install the following tools:"
        echo "  • curl  - For downloading cloudflared"
        echo "  • jq    - For JSON processing (Cloudflare API)"
        echo ""
        log_info "On Ubuntu/Debian:"
        echo "  sudo apt-get update && sudo apt-get install -y curl jq"
        echo ""
        log_info "On CentOS/RHEL:"
        echo "  sudo yum install -y curl jq"
        echo ""
        exit 1
    fi

    validate_dry_run

    if [ -z "$DOMAIN" ]; then
        while true; do
            read -p "Enter your domain (e.g., armorclaw.example.com): " DOMAIN
            DOMAIN=$(echo "$DOMAIN" | xargs)

            if [ -z "$DOMAIN" ]; then
                log_error "Domain cannot be empty"
                continue
            fi

            if ! echo "$DOMAIN" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$'; then
                log_error "Invalid domain format. Please use format like example.com or sub.example.com"
                continue
            fi

            break
        done
    else
        if ! echo "$DOMAIN" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$'; then
            log_error "Invalid domain format: $DOMAIN"
            echo ""
            exit 1
        fi
    fi

    log_info "Using domain: $DOMAIN"
    echo ""

    if [ -z "$MODE" ]; then
        detect_network_environment
        echo ""
        prompt_cloudflare_mode
    else
        log_info "Using mode: $MODE"
        echo ""
    fi

    case "$MODE" in
        tunnel)
            BASE_DOMAIN=$(echo "$DOMAIN" | cut -d. -f2-)
            SUBDOMAIN=$(echo "$DOMAIN" | cut -d. -f1)
            
            if [ "$DRY_RUN" = true ]; then
                log_info "[DRY-RUN] Tunnel mode setup for $DOMAIN"
                echo ""
                log_info "Would perform the following steps:"
                echo "  1. Install cloudflared (if not already installed)"
                echo "  2. Authenticate with Cloudflare (cloudflared tunnel login)"
                echo "  3. Create or use existing tunnel: armorclaw-$BASE_DOMAIN"
                echo "  4. Generate cloudflared config file"
                echo "  5. Create DNS records via Cloudflare Tunnel API"
                echo "  6. Create systemd service for cloudflared"
                echo "  7. Start cloudflared service"
                echo ""
                log_info "Would export:"
                echo "  PUBLIC_URL=https://$DOMAIN"
                echo "  MATRIX_URL=https://matrix.$DOMAIN"
                echo "  DOMAIN=$DOMAIN"
                echo ""
                log_info "✓ Dry-run completed - no changes made"
            else
                setup_cloudflare_tunnel
            fi
            ;;
        proxy)
            if [ -z "${CF_API_TOKEN:-}" ]; then
                log_error "CF_API_TOKEN environment variable is required for proxy mode"
                echo ""
                log_info "Set it with:"
                echo "  export CF_API_TOKEN=your_api_token"
                echo ""
                log_info "Get your API token from: https://dash.cloudflare.com/profile/api-tokens"
                echo ""
                log_info "Required permissions:"
                echo "  • Zone - DNS"
                echo "  • Zone - SSL"
                echo ""
                exit 1
            fi

            if [ "$DRY_RUN" = true ]; then
                log_info "[DRY-RUN] Proxy mode setup for $DOMAIN"
                echo ""
                log_info "Would perform the following steps:"
                echo "  1. Verify domain uses Cloudflare nameservers"
                echo "  2. Detect public IP address"
                echo "  3. Verify ports 80 and 443 are accessible"
                echo "  4. Create DNS A records with proxy enabled (orange cloud)"
                echo "  5. Generate Cloudflare origin certificate"
                echo "  6. Configure Caddy with origin certificate"
                echo "  7. Display SSL/TLS configuration instructions"
                echo ""
                log_info "Would export:"
                echo "  PUBLIC_URL=https://$DOMAIN"
                echo "  MATRIX_URL=https://matrix.$DOMAIN"
                echo "  DOMAIN=$DOMAIN"
                echo "  CF_ZONE_ID=<detected-zone-id>"
                echo ""
                log_info "✓ Dry-run completed - no changes made"
            else
                setup_cloudflare_proxy
            fi
            ;;
        *)
            log_error "Invalid mode: $MODE"
            echo ""
            exit 1
            ;;
    esac

    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}✓${NC} ${BOLD}Cloudflare HTTPS Setup Completed Successfully${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""

    if [ "$DRY_RUN" = false ]; then
        log_info "Your ArmorClaw deployment is now accessible via:"
        log_info "  Bridge:   ${CYAN}https://$DOMAIN${NC}"
        log_info "  Matrix:   ${CYAN}https://matrix.$DOMAIN${NC}"
        echo ""

        if [ "$MODE" = "proxy" ]; then
            log_info "IMPORTANT: Set SSL/TLS mode to ${GREEN}Full (strict)${NC} in Cloudflare dashboard"
            echo "  1. Go to: https://dash.cloudflare.com"
            echo "  2. Select your domain: $DOMAIN"
            echo "  3. Navigate to SSL/TLS → Overview"
            echo "  4. Set SSL/TLS encryption mode to: Full (strict)"
            echo ""
            log_info "Then restart Caddy to apply the new configuration:"
            echo "  sudo systemctl restart caddy"
            echo ""
        fi

        log_info "Environment variables have been exported for use by other scripts:"
        echo "  PUBLIC_URL=$PUBLIC_URL"
        echo "  MATRIX_URL=$MATRIX_URL"
        echo "  DOMAIN=$DOMAIN"
        if [ "$MODE" = "proxy" ]; then
            echo "  CF_ZONE_ID=$CF_ZONE_ID"
        fi
        echo ""
    fi
}

main "$@"
