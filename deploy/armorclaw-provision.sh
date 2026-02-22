#!/bin/bash
# ArmorClaw Device Provisioning
# Generate QR codes for secure ArmorChat/ArmorTerminal connection
# Version: 1.0.0
#
# Usage: sudo ./deploy/armorclaw-provision.sh [--expiry SECONDS] [--show-url]

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

    # Extract provisioning secret
    PROVISIONING_SECRET=$(grep 'signing_secret' "$CONFIG_FILE" 2>/dev/null | \
        sed 's/.*= *"\([^"]*\)".*/\1' || echo "")

    if [[ -z "$PROVISIONING_SECRET" ]]; then
        fail "Provisioning secret not configured. Run setup-wizard.sh first."
    fi

    # Get bridge info
    BRIDGE_HOSTNAME=$(hostname)
    BRIDGE_IP=$(hostname -I | awk '{print $1}')

    # Check for public IP/domain
    if grep -q 'homeserver_url.*https' "$CONFIG_FILE" 2>/dev/null; then
        # Extract domain from Matrix config
        PUBLIC_DOMAIN=$(grep 'homeserver_url' "$CONFIG_FILE" | \
            sed 's/.*https:\/\/\([^/]*\).*/\1/' | head -1)
    fi
}

#=============================================================================
# Generate Token
#=============================================================================

generate_token() {
    # Create timestamp
    local now=$(date +%s)
    local expiry=$((now + EXPIRY_SECONDS))

    # Create token payload
    # Format: base64(header).base64(payload).signature
    local header='{"alg":"HS256","typ":"JWT"}'
    local payload="{\"bridge\":\"$BRIDGE_HOSTNAME\",\"iat\":$now,\"exp\":$expiry}"

    # Base64 encode (URL-safe)
    local header_b64=$(echo -n "$header" | base64 | tr '+/' '-_' | tr -d '=')
    local payload_b64=$(echo -n "$payload" | base64 | tr '+/' '-_' | tr -d '=')

    # Create signature
    local signing_input="$header_b64.$payload_b64"
    local signature=$(echo -n "$signing_input" | \
        openssl dgst -sha256 -hmac "$PROVISIONING_SECRET" | \
        awk '{print $2}' | tr '[:lower:]' '[:upper:]')

    # Build JWT
    JWT_TOKEN="$signing_input.$signature"
    TOKEN_EXPIRY=$expiry
}

#=============================================================================
# Generate QR Code
#=============================================================================

generate_qr_code() {
    # Determine host (prefer public domain over IP)
    local host="${PUBLIC_DOMAIN:-$BRIDGE_IP}"
    local port="8080"

    # Build provisioning URL
    PROVISION_URL="armorclaw://provision?host=${host}&port=${port}&token=${JWT_TOKEN}&expires=${TOKEN_EXPIRY}"

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
    echo "  Host:    $host"
    echo "  Port:    $port"
    echo "  Expiry:  $(date -d @$TOKEN_EXPIRY 2>/dev/null || date -r $TOKEN_EXPIRY 2>/dev/null || echo "in $EXPIRY_SECONDS seconds")"
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
    echo -e "${YELLOW}Note:${NC} Token is single-use and will be invalidated after scanning"
    echo ""

    # Show URL for manual entry
    echo -e "${BOLD}Manual Entry URL:${NC}"
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
        echo "  1. Firewall allows port 8080 (or configured port)"
        echo "  2. TLS is configured for production"
        echo "  3. Public IP/domain is accessible from devices"
        echo ""
    else
        print_info "Detected: Local/Hardware deployment"
        echo ""
        echo "  Using local network IP: $BRIDGE_IP"
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
    generate_token

    if ! $SHOW_URL_ONLY; then
        detect_environment
    fi

    generate_qr_code
}

main "$@"
