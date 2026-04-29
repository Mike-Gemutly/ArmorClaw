#!/usr/bin/env bash
# ArmorClaw Self-Signed Certificate Generator
# Generates ECDSA P-256 CA + server certificates for self-hosted mode
#
# Usage:
#   sudo ./deploy/scripts/generate-certs.sh
#   sudo ./deploy/scripts/generate-certs.sh --output /etc/armorclaw/certs --hostname myhost.local
#   sudo ./deploy/scripts/generate-certs.sh --rotate   # regenerate server cert only (keep CA)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

#=============================================================================
# Defaults
#=============================================================================

OUTPUT_DIR="/etc/armorclaw/certs"
HOSTNAME="armorclaw.local"
LAN_IP=""
PUBLIC_IP=""
ROTATE=false

#=============================================================================
# Helper Functions
#=============================================================================

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
# Argument Parsing
#=============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --hostname)
                HOSTNAME="$2"
                shift 2
                ;;
            --lan-ip)
                LAN_IP="$2"
                shift 2
                ;;
            --public-ip)
                PUBLIC_IP="$2"
                shift 2
                ;;
            --rotate)
                ROTATE=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --output DIR      Output directory (default: /etc/armorclaw/certs)"
                echo "  --hostname NAME   Hostname for CN/SAN (default: armorclaw.local)"
                echo "  --lan-ip IP       LAN IP to include in SANs (auto-detect if omitted)"
                echo "  --public-ip IP    Public/external IP to include in SANs"
                echo "  --rotate          Regenerate server cert only (keep existing CA)"
                echo "  -h, --help        Show this help message"
                exit 0
                ;;
            *)
                fail "Unknown argument: $1"
                ;;
        esac
    done
}

#=============================================================================
# Auto-detect LAN IP
#=============================================================================

detect_lan_ip() {
    # Try hostname -I first (Linux)
    local ip
    ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    if [[ -z "$ip" ]]; then
        # Fallback: try ip route
        ip=$(ip route get 1.1.1.1 2>/dev/null | awk '{print $7; exit}')
    fi
    echo "${ip:-}"
}

#=============================================================================
# Generate CA Certificate
#=============================================================================

generate_ca() {
    print_step "Generating CA certificate..."

    if [[ -f "$OUTPUT_DIR/ca.key" ]] && [[ -f "$OUTPUT_DIR/ca.crt" ]]; then
        print_info "CA certificate already exists — skipping"
        return 0
    fi

    openssl ecparam -genkey -name prime256v1 -noout -out "$OUTPUT_DIR/ca.key"
    print_success "CA private key generated (ECDSA P-256)"

    openssl req -new -x509 \
        -key "$OUTPUT_DIR/ca.key" \
        -out "$OUTPUT_DIR/ca.crt" \
        -days 3650 \
        -subj "/O=ArmorClaw/CN=ArmorClaw CA" \
        -addext "basicConstraints=critical,CA:TRUE" \
        -addext "keyUsage=critical,digitalSignature,keyCertSign,cRLSign"

    print_success "CA certificate generated (10-year validity)"
}

#=============================================================================
# Generate Server Certificate
#=============================================================================

generate_server_cert() {
    print_step "Generating server certificate..."

    # Record old expiry if rotating
    local old_expiry=""
    if $ROTATE && [[ -f "$OUTPUT_DIR/server.crt" ]]; then
        old_expiry=$(openssl x509 -in "$OUTPUT_DIR/server.crt" -enddate -noout 2>/dev/null || echo "")
    fi

    # Build SAN extensions string for inline use
    local san_ext="basicConstraints=CA:FALSE"
    san_ext+=$'\n'"keyUsage=critical,digitalSignature,keyEncipherment"
    san_ext+=$'\n'"extendedKeyUsage=serverAuth"
    local san_list="DNS:${HOSTNAME},DNS:*.${HOSTNAME},DNS:localhost,IP:127.0.0.1"

    # Add LAN IP if available
    if [[ -n "$LAN_IP" ]]; then
        san_list+=",IP:${LAN_IP}"
    fi

    if [[ -n "$PUBLIC_IP" ]]; then
        san_list+=",IP:${PUBLIC_IP}"
    fi

    san_ext+=$'\n'"subjectAltName=${san_list}"

    # Generate server private key
    openssl ecparam -genkey -name prime256v1 -noout -out "$OUTPUT_DIR/server.key"
    print_success "Server private key generated (ECDSA P-256)"

    # Generate CSR
    openssl req -new \
        -key "$OUTPUT_DIR/server.key" \
        -out "$OUTPUT_DIR/server.csr" \
        -subj "/O=ArmorClaw/CN=${HOSTNAME}"

    # Sign with CA (1-year validity — matches ssl.go DefaultCertExpiry)
    openssl x509 -req \
        -in "$OUTPUT_DIR/server.csr" \
        -CA "$OUTPUT_DIR/ca.crt" \
        -CAkey "$OUTPUT_DIR/ca.key" \
        -CAcreateserial \
        -out "$OUTPUT_DIR/server.crt" \
        -days 365 \
        -extfile <(printf '%s' "$san_ext")

    # Clean up CSR
    rm -f "$OUTPUT_DIR/server.csr"

    print_success "Server certificate signed by CA (1-year validity)"

    # Show old vs new if rotating
    if [[ -n "$old_expiry" ]]; then
        print_info "Old certificate: ${old_expiry#notAfter=}"
        local new_expiry
        new_expiry=$(openssl x509 -in "$OUTPUT_DIR/server.crt" -enddate -noout)
        print_info "New certificate: ${new_expiry#notAfter=}"
    fi
}

#=============================================================================
# Set Permissions
#=============================================================================

set_permissions() {
    print_step "Setting file permissions..."

    chmod 644 "$OUTPUT_DIR/ca.crt" "$OUTPUT_DIR/server.crt"
    chmod 600 "$OUTPUT_DIR/ca.key" "$OUTPUT_DIR/server.key"

    print_success "Private keys: 600 (owner read/write only)"
    print_success "Certificates: 644 (world-readable)"
}

#=============================================================================
# Print Certificate Info
#=============================================================================

print_cert_info() {
    print_step "Certificate details..."

    # CA fingerprint
    local fingerprint
    fingerprint=$(openssl x509 -in "$OUTPUT_DIR/ca.crt" -fingerprint -sha256 -noout)
    print_info "CA fingerprint: ${fingerprint#SHA256 Fingerprint=}"

    # Server cert expiry
    local expiry
    expiry=$(openssl x509 -in "$OUTPUT_DIR/server.crt" -enddate -noout)
    print_info "Server cert: ${expiry}"

    # Rotation reminder
    echo ""
    echo -e "  ${YELLOW}To rotate server certificate:${NC}"
    echo -e "  ${CYAN}$0 --rotate --output $OUTPUT_DIR${NC}"
}

#=============================================================================
# Main
#=============================================================================

main() {
    parse_args "$@"

    # Auto-detect LAN IP if not provided
    if [[ -z "$LAN_IP" ]]; then
        LAN_IP=$(detect_lan_ip)
        if [[ -n "$LAN_IP" ]]; then
            print_info "Auto-detected LAN IP: $LAN_IP"
        else
            print_warning "Could not auto-detect LAN IP — server cert will not include a LAN IP SAN"
        fi
    fi

    # Check prerequisites
    if ! command -v openssl &>/dev/null; then
        fail "openssl is required but not found in PATH"
    fi

    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    print_info "Output directory: $OUTPUT_DIR"

    if $ROTATE; then
        # Rotate mode: only regenerate server cert
        if [[ ! -f "$OUTPUT_DIR/ca.key" ]]; then
            fail "Cannot rotate: CA key not found at $OUTPUT_DIR/ca.key. Run without --rotate first."
        fi
        if [[ ! -f "$OUTPUT_DIR/ca.crt" ]]; then
            fail "Cannot rotate: CA certificate not found at $OUTPUT_DIR/ca.crt. Run without --rotate first."
        fi
        print_info "Rotation mode — keeping existing CA, regenerating server certificate"
    else
        # Full mode: generate CA + server cert
        generate_ca
    fi

    generate_server_cert
    set_permissions
    print_cert_info

    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}              ${BOLD}Certificate Generation Complete${NC}                     ${GREEN}║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
}

main "$@"
