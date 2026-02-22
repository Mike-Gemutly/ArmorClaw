#!/bin/bash
# ArmorClaw Device Provisioning
# Generate QR codes for secure ArmorChat/ArmorTerminal connection
# Version: 1.1.0
#
# Usage: sudo ./deploy/armorclaw-provision.sh [--expiry SECONDS] [--show-url]
#
# QR Format: armorclaw://config?d=<base64-encoded-json>
# JSON contains: matrix_homeserver, rpc_url, ws_url, push_gateway, server_name, expires_at

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Default values
CONFIG_DIR="/etc/armorclaw"
CONFIG_FILE="$CONFIG_DIR/config.toml"
DEFAULT_EXPIRY=300  # 5 minutes
MAX_EXPIRY=3600     # 1 hour

# Default ports
DEFAULT_HTTP_PORT=8443
DEFAULT_MATRIX_PORT=8448
DEFAULT_PUSH_PORT=5000

# Output options
EXPIRY_SECONDS=$DEFAULT_EXPIRY
SHOW_URL_ONLY=false

#=============================================================================
# Helper Functions
#=============================================================================

print_error() {
    echo -e "${RED}✗${NC} ${BOLD}ERROR:${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

fail() {
    print_error "$1"
    exit 1
}

# URL-safe base64 encoding
base64_url_encode() {
    local input="$1"
    echo -n "$input" | base64 | tr '+/' '-_' | tr -d '='
}

#=============================================================================
# Parse Arguments
#=============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --expiry|-e)
                EXPIRY_SECONDS="$2"
                shift 2
                ;;
            --show-url|-u)
                SHOW_URL_ONLY=true
                shift
                ;;
            --help|-h)
                echo "Usage: sudo $0 [options]"
                echo ""
                echo "Generate QR codes for secure ArmorChat device provisioning."
                echo ""
                echo "Output Format: armorclaw://config?d=<base64-encoded-json>"
                echo ""
                echo "JSON Payload:"
                echo "  - matrix_homeserver: Matrix server URL"
                echo "  - rpc_url: Bridge RPC API URL"
                echo "  - ws_url: WebSocket URL for real-time events"
                echo "  - push_gateway: Push notification gateway URL"
                echo "  - server_name: Human-readable server name"
                echo "  - expires_at: Unix timestamp for expiry"
                echo ""
                echo "Options:"
                echo "  --expiry, -e SECONDS  Token expiry time (default: 300, max: 3600)"
                echo "  --show-url, -u        Only print the URL (no QR code)"
                echo "  --help, -h            Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                      # Generate QR with 5-minute expiry"
                echo "  $0 --expiry 60          # Generate QR with 1-minute expiry"
                echo "  $0 --show-url           # Print URL only (for scripting)"
                exit 0
                ;;
            *)
                print_warning "Unknown option: $1"
                shift
                ;;
        esac
    done

    # Validate expiry
    if [[ $EXPIRY_SECONDS -lt 30 ]]; then
        EXPIRY_SECONDS=30
        print_warning "Minimum expiry is 30 seconds"
    fi
    if [[ $EXPIRY_SECONDS -gt $MAX_EXPIRY ]]; then
        EXPIRY_SECONDS=$MAX_EXPIRY
        print_warning "Maximum expiry is $MAX_EXPIRY seconds"
    fi
}

#=============================================================================
# Read Configuration
#=============================================================================

read_config() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        fail "Configuration not found: $CONFIG_FILE"
    fi

    # Get bridge info
    BRIDGE_HOSTNAME=$(hostname)
    BRIDGE_IP=$(hostname -I | awk '{print $1}')

    # Extract Matrix homeserver URL from config
    MATRIX_HOMESERVER=$(grep 'homeserver_url' "$CONFIG_FILE" 2>/dev/null | \
        sed 's/.*= *"\([^"]*\)".*/\1' | head -1)

    # Extract server name (for display purposes)
    SERVER_NAME=$(grep 'server_name' "$CONFIG_FILE" 2>/dev/null | \
        sed 's/.*= *"\([^"]*\)".*/\1' || echo "$BRIDGE_HOSTNAME")

    # Determine if we have a public domain or IP-only
    if [[ -n "$MATRIX_HOMESERVER" ]] && [[ "$MATRIX_HOMESERVER" == https://* ]]; then
        # Extract domain from Matrix URL
        PUBLIC_DOMAIN=$(echo "$MATRIX_HOMESERVER" | sed 's|https://\([^/:]*\).*|\1|')
        USE_TLS=true
    else
        PUBLIC_DOMAIN=""
        USE_TLS=false
    fi

    # Determine the host to use
    if [[ -n "$PUBLIC_DOMAIN" ]]; then
        HOST="$PUBLIC_DOMAIN"
    else
        HOST="$BRIDGE_IP"
    fi

    # Build URLs based on TLS setting
    if [[ "$USE_TLS" == true ]]; then
        # Production with TLS
        MATRIX_URL="${MATRIX_HOMESERVER}"
        RPC_URL="https://${HOST}:${DEFAULT_HTTP_PORT}/api"
        WS_URL="wss://${HOST}:${DEFAULT_HTTP_PORT}/ws"
        PUSH_URL="https://${HOST}:${DEFAULT_PUSH_PORT}"
    else
        # Development/Local without TLS
        MATRIX_URL="http://${HOST}:${DEFAULT_MATRIX_PORT}"
        RPC_URL="http://${HOST}:${DEFAULT_HTTP_PORT}/api"
        WS_URL="ws://${HOST}:${DEFAULT_HTTP_PORT}/ws"
        PUSH_URL="http://${HOST}:${DEFAULT_PUSH_PORT}"
    fi

    # Override with explicit config values if present
    local config_rpc_url=$(grep 'rpc_url' "$CONFIG_FILE" 2>/dev/null | \
        sed 's/.*= *"\([^"]*\)".*/\1' | head -1)
    if [[ -n "$config_rpc_url" ]]; then
        RPC_URL="$config_rpc_url"
    fi

    local config_ws_url=$(grep 'ws_url' "$CONFIG_FILE" 2>/dev/null | \
        sed 's/.*= *"\([^"]*\)".*/\1' | head -1)
    if [[ -n "$config_ws_url" ]]; then
        WS_URL="$config_ws_url"
    fi
}

#=============================================================================
# Generate QR Config
#=============================================================================

generate_qr_config() {
    # Create timestamp
    local now=$(date +%s)
    local expiry=$((now + EXPIRY_SECONDS))

    # Build JSON config for ArmorChat
    # This format matches what ArmorChat's parseConfigDeepLink() expects
    CONFIG_JSON=$(cat <<EOF
{
  "matrix_homeserver": "${MATRIX_URL}",
  "rpc_url": "${RPC_URL}",
  "ws_url": "${WS_URL}",
  "push_gateway": "${PUSH_URL}",
  "server_name": "${SERVER_NAME:-$HOST}",
  "expires_at": ${expiry}
}
EOF
)

    # Base64 encode (URL-safe)
    CONFIG_B64=$(base64_url_encode "$CONFIG_JSON")

    # Build the deep link URL that ArmorChat expects
    # Format: armorclaw://config?d=<base64-encoded-json>
    PROVISION_URL="armorclaw://config?d=${CONFIG_B64}"

    TOKEN_EXPIRY=$expiry
}

#=============================================================================
# Generate QR Code
#=============================================================================

generate_qr_code() {
    if $SHOW_URL_ONLY; then
        echo "$PROVISION_URL"
        return 0
    fi

    # Display header
    echo ""
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Device Provisioning${NC}                     ${CYAN}║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    # Show connection info
    echo -e "${BOLD}Connection Details:${NC}"
    echo "  Server:      ${SERVER_NAME:-$HOST}"
    echo "  Matrix:      ${MATRIX_URL}"
    echo "  Bridge RPC:  ${RPC_URL}"
    echo "  WebSocket:   ${WS_URL}"
    echo "  Push:        ${PUSH_URL}"
    echo ""
    echo -e "${BOLD}Expires:${NC} $(date -d @$TOKEN_EXPIRY 2>/dev/null || date -r $TOKEN_EXPIRY 2>/dev/null || echo "in $EXPIRY_SECONDS seconds")"
    echo ""

    echo -e "${BOLD}Scan with ArmorChat or ArmorTerminal:${NC}"
    echo ""

    # Try to generate QR code
    if command -v qrencode &>/dev/null; then
        # Try UTF8 output first (most compatible)
        if qrencode -t UTF8 "$PROVISION_URL" 2>/dev/null; then
            :
        # Fall back to ASCII
        elif qrencode -t ASCII "$PROVISION_URL" 2>/dev/null; then
            :
        # Fall back to ASCIIi (inverted)
        elif qrencode -t ASCIIi "$PROVISION_URL" 2>/dev/null; then
            :
        else
            # QR generation failed, show URL
            show_url_fallback
        fi
    else
        show_url_fallback
    fi

    echo ""
    echo -e "${YELLOW}Note:${NC} This code expires in $EXPIRY_SECONDS seconds"
    echo ""

    # Show URL for manual entry
    echo -e "${BOLD}Manual Entry URL (copy to ArmorChat):${NC}"
    echo -e "  ${CYAN}$PROVISION_URL${NC}"
    echo ""
}

show_url_fallback() {
    echo -e "${YELLOW}QR code generation unavailable.${NC}"
    echo "Install qrencode for visual QR codes:"
    echo "  ${CYAN}sudo apt install qrencode${NC}"
    echo ""
    echo -e "${BOLD}Copy this URL to ArmorChat:${NC}"
    echo ""
    echo -e "  ${CYAN}$PROVISION_URL${NC}"
}

#=============================================================================
# VPS vs Local Detection
#=============================================================================

detect_environment() {
    print_info "Detecting deployment environment..."

    # Check for cloud indicators
    local is_vps=false

    # Check for cloud provider metadata
    if curl -sf --connect-timeout 1 http://169.254.169.254/latest/meta-data/ &>/dev/null; then
        is_vps=true  # AWS
        print_info "Detected: Cloud VPS (AWS)"
    elif curl -sf --connect-timeout 1 -H "Metadata-Flavor: Google" http://metadata.google.internal/ &>/dev/null; then
        is_vps=true  # GCP
        print_info "Detected: Cloud VPS (GCP)"
    elif curl -sf --connect-timeout 1 http://169.254.169.254/metadata/v1/ &>/dev/null; then
        is_vps=true  # DigitalOcean
        print_info "Detected: Cloud VPS (DigitalOcean)"
    elif [[ -f /etc/cloud/cloud.cfg ]]; then
        is_vps=true  # Generic cloud-init
        print_info "Detected: Cloud VPS (cloud-init)"
    fi

    if $is_vps; then
        echo ""
        print_warning "VPS Environment Detected"
        echo ""
        echo "  For VPS deployments, ensure:"
        echo "  1. Firewall allows ports: ${DEFAULT_HTTP_PORT}, ${DEFAULT_MATRIX_PORT}, ${DEFAULT_PUSH_PORT}"
        echo "  2. TLS is configured for production (https://)"
        echo "  3. Public IP/domain is accessible from devices"
        echo ""
        if [[ "$USE_TLS" != true ]]; then
            print_warning "TLS not detected - recommend enabling for production!"
        fi
    else
        print_info "Detected: Local/Hardware deployment"
        echo ""
        echo "  Using local network IP: $HOST"
        echo ""
    fi
}

#=============================================================================
# Main
#=============================================================================

main() {
    # Check root
    if [[ $EUID -ne 0 ]]; then
        fail "This script must be run as root (use sudo)"
    fi

    parse_args "$@"
    read_config
    generate_qr_config

    if ! $SHOW_URL_ONLY; then
        detect_environment
    fi

    generate_qr_code
}

main "$@"
