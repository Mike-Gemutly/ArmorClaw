#!/usr/bin/env bash
# Cloudflare HTTPS Setup - Shared Function Library
# =============================================================================
# This library provides core utility functions for Cloudflare HTTPS setup.
# It supports both Tunnel and Proxy modes.
#
# Usage: source deploy/lib/cloudflare-functions.sh
#
# Functions:
#   - log_info()      : Print informational message
#   - log_warn()      : Print warning message
#   - log_error()     : Print error message
#   - die_on_error()  : Print error message and exit
#   - check_cloudflare_prerequisites() : Verify required tools are available
#   - install_cloudflared()            : Download and install cloudflared binary
#   - check_cloudflare_nameservers()   : Check if domain uses Cloudflare nameservers
#   - get_cloudflare_zone_id()        : Get Cloudflare zone ID for a domain
#   - get_existing_dns_record_id()    : Check if DNS record exists and get its ID
#   - create_or_update_dns_a_record() : Create or update DNS A record with proxy
#   - create_dns_a_record()           : Create DNS A records for ArmorClaw (main + matrix)
# =============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info() {
    echo -e "  ${CYAN}ℹ${NC} $*"
}

log_warn() {
    echo -e "  ${YELLOW}⚠${NC} $*"
}

log_error() {
    echo -e "  ${RED}✗${NC} ${BOLD}ERROR:${NC} $*" >&2
}

die_on_error() {
    log_error "$*"
    exit 1
}

check_cloudflare_prerequisites() {
    local missing=0

    log_info "Checking Cloudflare prerequisites..."
    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required but not installed"
        missing=$((missing + 1))
    fi
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed (for JSON processing)"
        missing=$((missing + 1))
    fi
    if ! command -v cloudflared >/dev/null 2>&1; then
        log_warn "cloudflared not found - will be installed automatically"
    else
        log_info "cloudflared is already installed: $(cloudflared --version 2>&1 | head -1)"
    fi

    if [ "$missing" -gt 0 ]; then
        return 1
    fi

    log_info "All Cloudflare prerequisites met"
    return 0
}

install_cloudflared() {
    log_info "Installing cloudflared..."
    local os arch
    case "$(uname -s)" in
        Linux*)  os="linux" ;;
        Darwin*) os="darwin" ;;
        *)       die_on_error "Unsupported OS: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)             die_on_error "Unsupported architecture: $(uname -m)" ;;
    esac

    local cloudflared_url="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-${os}-${arch}"
    local install_dir="/usr/local/bin"
    local binary_path="${install_dir}/cloudflared"
    local tmp_file
    tmp_file=$(mktemp)

    log_info "Downloading cloudflared for ${os}-${arch}..."
    if ! curl -fsSL "$cloudflared_url" -o "$tmp_file"; then
        rm -f "$tmp_file"
        die_on_error "Failed to download cloudflared from $cloudflared_url"
    fi

    chmod +x "$tmp_file"
    if [ -w "$install_dir" ]; then
        mv "$tmp_file" "$binary_path"
    elif command -v sudo >/dev/null 2>&1; then
        sudo mv "$tmp_file" "$binary_path"
    else
        rm -f "$tmp_file"
        die_on_error "Cannot write to $install_dir (need sudo)"
    fi
    if [ ! -x "$binary_path" ]; then
        die_on_error "cloudflared installation failed (binary not executable)"
    fi

    log_info "cloudflared installed successfully: $($binary_path --version 2>&1 | head -1)"
}

check_cloudflare_nameservers() {
    local domain="$1"

    if [ -z "$domain" ]; then
        log_error "Domain parameter is required"
        return 1
    fi

    log_info "Checking if $domain uses Cloudflare nameservers..."
    local nameservers
    nameservers=$(dig +short NS "$domain" 2>/dev/null)

    if [ -z "$nameservers" ]; then
        log_error "Could not retrieve nameservers for $domain"
        return 1
    fi
    local cf_ns_pattern="\.cloudflare\.com\."
    local is_cloudflare=0

    while IFS= read -r ns; do
        if echo "$ns" | grep -qiE "$cf_ns_pattern"; then
            is_cloudflare=1
            log_info "Found Cloudflare nameserver: $ns"
            break
        fi
    done <<< "$nameservers"

    if [ "$is_cloudflare" -eq 1 ]; then
        log_info "✓ Domain $domain is using Cloudflare nameservers"
        return 0
    else
        log_warn "✗ Domain $domain is NOT using Cloudflare nameservers"
        log_warn "  Current nameservers:"
        while IFS= read -r ns; do
            log_warn "    - $ns"
        done <<< "$nameservers"
        log_warn "  Proxy mode requires Cloudflare nameservers. Use Tunnel mode instead."
        return 1
    fi
}

prompt_cloudflare_mode() {
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}         Cloudflare Deployment Mode Selection${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BOLD}Detected Environment:${NC}"
    echo -e "  ${CYAN}NAT Status:${NC} ${HAS_NAT}"
    echo -e "  ${CYAN}Available Ports:${NC} ${PORTS_AVAILABLE}"
    echo ""
    echo -e "${BOLD}Recommendation:${NC}"
    echo -e "  ${GREEN}✓${NC} ${RECOMMEND}"
    echo -e "  ${YELLOW}→${NC} ${REASON}"
    echo ""
    echo -e "${BOLD}───────────────────────────────────────────────────────────${NC}"
    echo ""
    echo -e "${BOLD}Select Deployment Mode:${NC}"
    echo ""
    echo -e "  ${BOLD}1) Tunnel Mode${NC} ${CYAN}(cloudflared)${NC}"
    echo -e "     ${GREEN}Best for: NAT/CGNAT, dynamic IP, no public ports${NC}"
    echo -e "     Creates outbound tunnel to Cloudflare, bypasses NAT${NC}"
    echo ""
    echo -e "  ${BOLD}2) Proxy Mode${NC} ${CYAN}(DNS/CDN)${NC}"
    echo -e "     ${GREEN}Best for: Static public IP, port forwarding, own domain${NC}"
    echo -e "     Proxies traffic through Cloudflare DNS/CDN${NC}"
    echo ""
    echo -e "${BOLD}───────────────────────────────────────────────────────────${NC}"
    echo ""

    local choice
    while true; do
        read -p "Enter your choice [1-2]: " choice
        case "$choice" in
            1)
                export CF_MODE="tunnel"
                echo ""
                log_info "Selected: Tunnel Mode"
                return 0
                ;;
            2)
                export CF_MODE="proxy"
                echo ""
                log_info "Selected: Proxy Mode"
                return 0
                ;;
            *)
                echo ""
                log_error "Invalid choice. Please enter 1 or 2."
                ;;
        esac
    done
}

detect_network_environment() {
    log_info "Detecting network environment..."
    
    # Default values (fallback)
    local public_ip=""
    local local_ip=""
    local has_nat="unknown"
    local port_80_open="unknown"
    local port_443_open="unknown"
    
    # Get public IP with timeout
    public_ip=$(curl -s --max-time 5 --connect-timeout 3 https://api.ipify.org 2>/dev/null || echo "")
    if [ -z "$public_ip" ]; then
        log_warn "Could not detect public IP (using fallback)"
    else
        log_info "Public IP: $public_ip"
    fi
    
    # Get local IP with fallback methods
    if command -v ip >/dev/null 2>&1; then
        # Try primary interface first
        local_ip=$(ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++)if($i=="src")print $(i+1)}' | head -1)
    fi
    
    # Fallback: try hostname -I
    if [ -z "$local_ip" ] && command -v hostname >/dev/null 2>&1; then
        local_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    
    # Fallback: try ifconfig
    if [ -z "$local_ip" ] && command -v ifconfig >/dev/null 2>&1; then
        local_ip=$(ifconfig 2>/dev/null | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1)
    fi
    
    if [ -z "$local_ip" ]; then
        log_warn "Could not detect local IP (using fallback)"
    else
        log_info "Local IP: $local_ip"
    fi
    
    # Detect NAT (compare IPs if both available)
    if [ -n "$public_ip" ] && [ -n "$local_ip" ]; then
        if [ "$public_ip" = "$local_ip" ]; then
            has_nat="no"
            log_info "NAT detected: No (direct public IP)"
        else
            has_nat="yes"
            log_info "NAT detected: Yes (public IP differs from local IP)"
        fi
    else
        log_warn "Could not determine NAT status (missing IP data)"
    fi
    
    # Test port 80 connectivity (with timeout)
    if command -v timeout >/dev/null 2>&1 && command -v nc >/dev/null 2>&1; then
        if timeout 3 nc -z localhost 80 2>/dev/null; then
            port_80_open="yes"
            log_info "Port 80: Open (localhost)"
        else
            port_80_open="no"
            log_info "Port 80: Closed or unavailable (localhost)"
        fi
    elif command -v timeout >/dev/null 2>&1 && command -v bash >/dev/null 2>&1; then
        # Fallback: try using bash built-in with /dev/tcp
        if timeout 3 bash -c 'echo > /dev/tcp/127.0.0.1/80' 2>/dev/null; then
            port_80_open="yes"
            log_info "Port 80: Open (localhost)"
        else
            port_80_open="no"
            log_info "Port 80: Closed or unavailable (localhost)"
        fi
    else
        log_warn "Could not test port 80 (missing tools)"
    fi
    
    # Test port 443 connectivity (with timeout)
    if command -v timeout >/dev/null 2>&1 && command -v nc >/dev/null 2>&1; then
        if timeout 3 nc -z localhost 443 2>/dev/null; then
            port_443_open="yes"
            log_info "Port 443: Open (localhost)"
        else
            port_443_open="no"
            log_info "Port 443: Closed or unavailable (localhost)"
        fi
    elif command -v timeout >/dev/null 2>&1 && command -v bash >/dev/null 2>&1; then
        # Fallback: try using bash built-in with /dev/tcp
        if timeout 3 bash -c 'echo > /dev/tcp/127.0.0.1/443' 2>/dev/null; then
            port_443_open="yes"
            log_info "Port 443: Open (localhost)"
        else
            port_443_open="no"
            log_info "Port 443: Closed or unavailable (localhost)"
        fi
    else
        log_warn "Could not test port 443 (missing tools)"
    fi
    
    # Determine recommendation based on detected environment
    if [ "$has_nat" = "yes" ]; then
        # NAT detected - behind router/firewall
        RECOMMEND="tunnel"
        REASON="NAT detected (public IP: ${public_ip:-unknown} != local IP: ${local_ip:-unknown}). Cloudflare Tunnel bypasses NAT and doesn't require port forwarding."
    elif [ "$has_nat" = "no" ]; then
        # Direct public IP - check port availability
        if [ "$port_80_open" = "yes" ] && [ "$port_443_open" = "yes" ]; then
            RECOMMEND="proxy"
            REASON="Direct public IP detected with both HTTP (80) and HTTPS (443) ports available. Cloudflare Proxy mode is recommended for optimal performance."
        elif [ "$port_80_open" = "yes" ]; then
            RECOMMEND="proxy"
            REASON="Direct public IP detected with HTTP (80) port available. Cloudflare Proxy mode can be configured."
        elif [ "$port_443_open" = "yes" ]; then
            RECOMMEND="proxy"
            REASON="Direct public IP detected with HTTPS (443) port available. Cloudflare Proxy mode can be configured."
        else
            RECOMMEND="tunnel"
            REASON="Direct public IP detected but ports 80 and 443 are not available. Cloudflare Tunnel is recommended (no port forwarding required)."
        fi
    else
        # Unknown NAT status - safe default
        RECOMMEND="tunnel"
        REASON="Network environment detection incomplete (NAT status: ${has_nat:-unknown}). Cloudflare Tunnel is the safe default for most environments."
    fi
    
    log_info "Recommended mode: ${BOLD}${RECOMMEND}${NC}"
    log_info "Reason: $REASON"

    # Export variables for use by caller
    export RECOMMEND
    export REASON

    return 0
}

setup_cloudflare_tunnel() {
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Setting up Cloudflare Tunnel..."

        local domain
        if [ -n "${1:-}" ]; then
            domain="$1"
        else
            while true; do
                echo ""
                read -p "Enter your domain (e.g., armorclaw.example.com): " domain
                domain=$(echo "$domain" | xargs)

                if [ -z "$domain" ]; then
                    log_error "Domain cannot be empty"
                    continue
                fi

                if ! echo "$domain" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$'; then
                    log_error "Invalid domain format. Please use format like example.com or sub.example.com"
                    continue
                fi

                break
            done
        fi

        local base_domain
        local tunnel_name="armorclaw-$base_domain"
        local tunnel_id="mock-tunnel-id-$(echo "$domain" | md5sum | cut -c1-8)"

        log_info "[DRY-RUN] Using domain: $domain"
        log_info "[DRY-RUN] Would perform:"
        log_info "[DRY-RUN]   1. Install cloudflared (if needed)"
        log_info "[DRY-RUN]   2. Authenticate with Cloudflare"
        log_info "[DRY-RUN]   3. Create tunnel: $tunnel_name (ID: $tunnel_id)"
        log_info "[DRY-RUN]   4. Generate cloudflared config file"
        log_info "[DRY-RUN]   5. Create DNS records for $domain and matrix.$domain"
        log_info "[DRY-RUN]   6. Create systemd service"
        log_info "[DRY-RUN]   7. Start cloudflared service"

        local cloudflared_dir="$HOME/.cloudflared"
        local config_file="$cloudflared_dir/config.yml"
        log_info "[DRY-RUN] Would generate config file: $config_file"
        log_info "[DRY-RUN] Config would include:"
        log_info "[DRY-RUN]   - $domain -> localhost:8443"
        log_info "[DRY-RUN]   - matrix.$domain -> localhost:6167"

        export PUBLIC_URL="https://$domain"
        export MATRIX_URL="https://matrix.$domain"
        export DOMAIN="$domain"

        log_info "[DRY-RUN] Would export:"
        log_info "[DRY-RUN]   PUBLIC_URL=$PUBLIC_URL"
        log_info "[DRY-RUN]   MATRIX_URL=$MATRIX_URL"
        log_info "[DRY-RUN]   DOMAIN=$DOMAIN"

        return 0
    fi

    log_info "Setting up Cloudflare Tunnel..."

    check_existing_web_server

    if ! check_cloudflared_installed; then
        log_warn "cloudflared not installed, installing now..."
        install_cloudflared
    fi

    local domain
    while true; do
        echo ""
        read -p "Enter your domain (e.g., armorclaw.example.com): " domain
        domain=$(echo "$domain" | xargs)

        if [ -z "$domain" ]; then
            log_error "Domain cannot be empty"
            continue
        fi

        if ! echo "$domain" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$'; then
            log_error "Invalid domain format. Please use format like example.com or sub.example.com"
            continue
        fi

        break
    done

    log_info "Using domain: $domain"

    local subdomain base_domain
    if echo "$domain" | grep -q '\.'; then
        subdomain=$(echo "$domain" | cut -d. -f1)
        base_domain=$(echo "$domain" | cut -d. -f2-)
    else
        log_error "Domain must have at least one dot (e.g., example.com or sub.example.com)"
        return 1
    fi

    log_info "Subdomain: $subdomain"
    log_info "Base domain: $base_domain"

    local cloudflared_dir="$HOME/.cloudflared"
    local cert_file="$cloudflared_dir/cert.pem"

    if [ -f "$cert_file" ]; then
        log_info "Found existing Cloudflare certificate"
    else
        log_warn "No Cloudflare certificate found"
        log_info "Running 'cloudflared tunnel login' to authenticate..."

        handle_auth_timeout 300 || die_on_error "Failed to authenticate with Cloudflare"

        if [ ! -f "$cert_file" ]; then
            die_on_error "Authentication failed - certificate file not found after login"
        fi

        log_info "Authentication successful"
    fi

    local tunnel_name="armorclaw-$base_domain"
    local tunnel_info tunnel_id

    log_info "Checking for existing tunnel: $tunnel_name"

    if check_existing_tunnel "$tunnel_name"; then
        tunnel_id="$CF_TUNNEL_ID"
        log_info "Reusing existing tunnel: $tunnel_name (ID: $tunnel_id)"
    else
        log_info "Creating new tunnel: $tunnel_name"

        if ! tunnel_info=$(cloudflared tunnel create "$tunnel_name" 2>&1); then
            die_on_error "Failed to create tunnel: $tunnel_info"
        fi

        tunnel_id=$(echo "$tunnel_info" | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}')
        log_info "Tunnel created with ID: $tunnel_id"
    fi

    mkdir -p "$cloudflared_dir" || die_on_error "Failed to create $cloudflared_dir"

    local config_file="$cloudflared_dir/config.yml"

    log_info "Generating config file: $config_file"

    cat > "$config_file" <<EOF
tunnel: $tunnel_id
credentials-file: $cloudflared_dir/${tunnel_id}.json

ingress:
  - hostname: $domain
    service: http://localhost:8443
  - hostname: matrix.$domain
    service: http://localhost:6167
  - service: http_status:404
EOF

    log_info "Config file generated with ingress rules for:"
    log_info "  - $domain -> localhost:8443 (Bridge)"
    log_info "  - matrix.$domain -> localhost:6167 (Matrix)"

    log_info "Creating DNS records..."

    log_info "Creating DNS record for $domain"
    if cloudflared tunnel route dns "$tunnel_name" "$domain"; then
        log_info "DNS record created for $domain"
    else
        log_warn "Failed to create DNS record for $domain (may already exist)"
    fi

    log_info "Creating DNS record for matrix.$domain"
    if cloudflared tunnel route dns "$tunnel_name" "matrix.$domain"; then
        log_info "DNS record created for matrix.$domain"
    else
        log_warn "Failed to create DNS record for matrix.$domain (may already exist)"
    fi

    local service_file="/etc/systemd/system/cloudflared-tunnel.service"

    log_info "Creating systemd service: $service_file"

    local service_content="[Unit]
Description=Cloudflare Tunnel for ArmorClaw
After=network.target

[Service]
Type=simple
User=$USER
ExecStart=/usr/local/bin/cloudflared tunnel run --config=$config_file $tunnel_name
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target"

    if [ -w "/etc/systemd/system" ]; then
        echo "$service_content" > "$service_file"
    elif command -v sudo >/dev/null 2>&1; then
        echo "$service_content" | sudo tee "$service_file" >/dev/null || die_on_error "Failed to create systemd service file"
    else
        die_on_error "Cannot write to /etc/systemd/system (need sudo)"
    fi

    log_info "Systemd service file created"

    log_info "Reloading systemd daemon..."
    if command -v sudo >/dev/null 2>&1; then
        sudo systemctl daemon-reload || die_on_error "Failed to reload systemd daemon"
        sudo systemctl enable cloudflared-tunnel.service || die_on_error "Failed to enable cloudflared-tunnel service"
        if ! sudo systemctl start cloudflared-tunnel.service; then
            handle_service_failure "cloudflared-tunnel"
            die_on_error "Failed to start cloudflared-tunnel service"
        fi
    else
        systemctl daemon-reload || die_on_error "Failed to reload systemd daemon"
        systemctl enable cloudflared-tunnel.service || die_on_error "Failed to enable cloudflared-tunnel service"
        if ! systemctl start cloudflared-tunnel.service; then
            handle_service_failure "cloudflared-tunnel"
            die_on_error "Failed to start cloudflared-tunnel service"
        fi
    fi

    log_info "Cloudflare Tunnel service started successfully"

    export PUBLIC_URL="https://$domain"
    export MATRIX_URL="https://matrix.$domain"
    export DOMAIN="$domain"

    log_info "Environment variables exported:"
    log_info "  PUBLIC_URL=$PUBLIC_URL"
    log_info "  MATRIX_URL=$MATRIX_URL"
    log_info "  DOMAIN=$DOMAIN"

    return 0
}

configure_caddy_origin() {
    local domain="$1"
    local cert_path="$2"
    local key_path="$3"
    local caddyfile="${4:-/etc/caddy/Caddyfile}"

    if [ -z "$domain" ] || [ -z "$cert_path" ] || [ -z "$key_path" ]; then
        log_error "Missing required parameters: domain, cert_path, or key_path"
        return 1
    fi

    if [ ! -f "$cert_path" ]; then
        log_error "Certificate file not found: $cert_path"
        return 1
    fi

    if [ ! -f "$key_path" ]; then
        log_error "Private key file not found: $key_path"
        return 1
    fi

    log_info "Configuring Caddy to use origin certificates for $domain..."

    if [ -f "$caddyfile" ]; then
        local backup_file="${caddyfile}.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "Backing up existing Caddyfile to $backup_file"
        if cp "$caddyfile" "$backup_file"; then
            log_info "Backup created successfully"
        else
            log_warn "Failed to create backup, continuing anyway"
        fi
    else
        log_warn "Caddyfile not found at $caddyfile, creating new file"
    fi

    local caddy_dir
    caddy_dir=$(dirname "$caddyfile")
    if ! mkdir -p "$caddy_dir"; then
        log_error "Failed to create directory: $caddy_dir"
        return 1
    fi

    log_info "Writing Caddyfile with origin certificate configuration..."

    cat > "$caddyfile" <<EOF
# Caddyfile with Cloudflare Origin Certificate
# Auto-generated by ArmorClaw deploy script

$domain {
    reverse_proxy localhost:8443
    tls $cert_path $key_path
}

matrix.$domain {
    reverse_proxy localhost:6167
    tls $cert_path $key_path
}
EOF

    log_info "Caddyfile written successfully"
    log_info "  - $domain -> localhost:8443 (Bridge)"
    log_info "  - matrix.$domain -> localhost:6167 (Matrix)"

    if command -v caddy >/dev/null 2>&1; then
        log_info "Validating Caddyfile configuration..."
        if caddy validate --config "$caddyfile" 2>&1; then
            log_info "Caddyfile configuration is valid"
        else
            log_error "Caddyfile configuration is invalid"
            log_warn "Please check the Caddyfile and ensure certificate/key paths are correct"
            return 1
        fi
    else
        log_warn "caddy binary not found, skipping configuration validation"
        log_warn "Please install Caddy to validate the configuration"
    fi

    log_info "Caddy origin certificate configuration completed"
    return 0
}

# =============================================================================
# DNS A Record Creation Functions
# =============================================================================

# Function to get Cloudflare zone ID for a domain
# Arguments:
#   $1 - domain (e.g., example.com)
#   $2 - CF_API_TOKEN
# Returns:
#   Zone ID on success, empty string on failure
get_cloudflare_zone_id() {
    local domain="$1"
    local cf_token="$2"

    if [ -z "$domain" ] || [ -z "$cf_token" ]; then
        log_error "Domain and CF_API_TOKEN are required"
        echo ""
        return 1
    fi

    log_info "Looking up Cloudflare zone ID for $domain..."
    local response
    response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=${domain}" \
        -H "Authorization: Bearer ${cf_token}" \
        -H "Content-Type: application/json")

    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_error "Failed to get zone ID"
        log_error "Response: $(echo "$response" | jq -r '.errors[].message' 2>/dev/null || echo 'Unknown error')"
        echo ""
        return 1
    fi

    local zone_id
    zone_id=$(echo "$response" | jq -r '.result[0].id' 2>/dev/null)

    if [ -z "$zone_id" ] || [ "$zone_id" = "null" ]; then
        log_error "No zone found for domain: $domain"
        echo ""
        return 1
    fi

    log_info "Zone ID found: $zone_id"
    echo "$zone_id"
    return 0
}

# Function to check if a DNS record exists and get its ID
# Arguments:
#   $1 - zone_id
#   $2 - record_name (e.g., example.com or matrix.example.com)
#   $3 - CF_API_TOKEN
# Returns:
#   Record ID on success, empty string if not found, error on failure
get_existing_dns_record_id() {
    local zone_id="$1"
    local record_name="$2"
    local cf_token="$3"

    local response
    response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records?type=A&name=${record_name}" \
        -H "Authorization: Bearer ${cf_token}" \
        -H "Content-Type: application/json")

    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_error "Failed to query DNS records"
        log_error "Response: $(echo "$response" | jq -r '.errors[].message' 2>/dev/null || echo 'Unknown error')"
        echo ""
        return 1
    fi

    local record_id
    record_id=$(echo "$response" | jq -r '.result[0].id // ""' 2>/dev/null)

    echo "$record_id"
    return 0
}

# Function to create or update a DNS A record with proxy enabled
# Arguments:
#   $1 - zone_id
#   $2 - record_name (e.g., example.com or matrix.example.com)
#   $3 - ip_address
#   $4 - CF_API_TOKEN
# Returns:
#   0 on success, 1 on failure
create_or_update_dns_a_record() {
    local zone_id="$1"
    local record_name="$2"
    local ip_address="$3"
    local cf_token="$4"

    log_info "Processing DNS record: $record_name -> $ip_address"

    # Check if record exists
    local existing_record_id
    existing_record_id=$(get_existing_dns_record_id "$zone_id" "$record_name" "$cf_token")

    if [ $? -ne 0 ]; then
        log_error "Failed to check for existing record"
        return 1
    fi

    # Get current IP if record exists
    local current_ip=""
    if [ -n "$existing_record_id" ]; then
        local response
        response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records/${existing_record_id}" \
            -H "Authorization: Bearer ${cf_token}" \
            -H "Content-Type: application/json")

        if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
            current_ip=$(echo "$response" | jq -r '.result.content' 2>/dev/null)
        fi

        # Check if IP matches
        if [ "$current_ip" = "$ip_address" ]; then
            log_info "✓ Record exists and IP matches (proxy enabled)"
            return 0
        else
            log_warn "Record exists but IP differs: $current_ip -> $ip_address"
        fi
    fi

    # Create or update record
    local response
    if [ -n "$existing_record_id" ]; then
        # Update existing record
        log_info "Updating existing DNS record..."
        response=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records/${existing_record_id}" \
            -H "Authorization: Bearer ${cf_token}" \
            -H "Content-Type: application/json" \
            --data "{\"type\":\"A\",\"name\":\"${record_name}\",\"content\":\"${ip_address}\",\"proxied\":true}")
    else
        # Create new record
        log_info "Creating new DNS record with proxy enabled..."
        response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records" \
            -H "Authorization: Bearer ${cf_token}" \
            -H "Content-Type: application/json" \
            --data "{\"type\":\"A\",\"name\":\"${record_name}\",\"content\":\"${ip_address}\",\"proxied\":true}")
    fi

    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_error "Failed to create DNS record for $record_name"
        log_error "Response: $(echo "$response" | jq -r '.errors[].message' 2>/dev/null || echo 'Unknown error')"
        return 1
    fi

    log_info "✓ DNS A record created/updated: $record_name -> $ip_address (proxy enabled)"
    return 0
}

# Main function to create DNS A records for ArmorClaw deployment
# Creates both main domain and matrix subdomain records with Cloudflare proxy enabled
#
# Arguments:
#   $1 - domain (e.g., example.com)
#   $2 - ip_address (e.g., 203.0.113.1)
#   $3 - CF_API_TOKEN (Cloudflare API token with Zone:DNS edit permissions)
#
# Returns:
#   0 on success, 1 on failure
#
# Environment Variables Exported on Success:
#   - CF_ZONE_ID: The Cloudflare zone ID
#
# Example:
#   create_dns_a_record "example.com" "203.0.113.1" "your_api_token"
create_dns_a_record() {
    local domain="$1"
    local ip_address="$2"
    local cf_token="$3"

    if [ -z "$domain" ]; then
        log_error "Domain parameter is required"
        return 1
    fi

    if [ -z "$ip_address" ]; then
        log_error "IP address parameter is required"
        return 1
    fi

    if [ -z "$cf_token" ]; then
        log_error "CF_API_TOKEN is required"
        return 1
    fi

    log_info "Creating DNS A records for Cloudflare Proxy mode..."
    log_info "  Domain: $domain"
    log_info "  IP Address: $ip_address"

    # Get zone ID
    local zone_id
    zone_id=$(get_cloudflare_zone_id "$domain" "$cf_token")

    if [ -z "$zone_id" ]; then
        log_error "Failed to retrieve zone ID for $domain"
        return 1
    fi

    export CF_ZONE_ID="$zone_id"

    # Create main domain record
    if ! create_or_update_dns_a_record "$zone_id" "$domain" "$ip_address" "$cf_token"; then
        log_error "Failed to create/update DNS record for $domain"
        return 1
    fi

    # Create matrix subdomain record
    if ! create_or_update_dns_a_record "$zone_id" "matrix.$domain" "$ip_address" "$cf_token"; then
        log_error "Failed to create/update DNS record for matrix.$domain"
        return 1
    fi

    log_info "✓ All DNS A records created successfully with proxy enabled"
    log_info "  - $domain -> $ip_address (orange cloud: ON)"
    log_info "  - matrix.$domain -> $ip_address (orange cloud: ON)"

    return 0
}

setup_manual_domain() {
    log_info "Setting up manual domain configuration..."

    local domain
    while true; do
        echo ""
        read -p "Enter your domain (e.g., example.com or sub.example.com): " domain
        domain=$(echo "$domain" | xargs)

        if [ -z "$domain" ]; then
            log_error "Domain cannot be empty"
            continue
        fi

        if ! echo "$domain" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$'; then
            log_error "Invalid domain format. Please use format like example.com or sub.example.com"
            continue
        fi

        break
    done

    log_info "Using domain: $domain"

    # Detect public IP
    local public_ip
    log_info "Detecting public IP address..."
    public_ip=$(curl -s --max-time 5 --connect-timeout 3 https://api.ipify.org 2>/dev/null || echo "")

    if [ -z "$public_ip" ]; then
        log_warn "Could not detect public IP automatically"
        while true; do
            read -p "Enter your public IP address: " public_ip
            public_ip=$(echo "$public_ip" | xargs)

            if [ -z "$public_ip" ]; then
                log_error "Public IP cannot be empty"
                continue
            fi

            if ! echo "$public_ip" | grep -qE '^([0-9]{1,3}\.){3}[0-9]{1,3}$'; then
                log_error "Invalid IP format. Please use format like 1.2.3.4"
                continue
            fi

            break
        done
    else
        log_info "Public IP detected: $public_ip"
    fi

    # Display DNS configuration instructions
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}         Manual DNS Configuration Instructions${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BOLD}Your Public IP:${NC} ${CYAN}$public_ip${NC}"
    echo -e "${BOLD}Your Domain:${NC} ${CYAN}$domain${NC}"
    echo ""
    echo -e "${BOLD}───────────────────────────────────────────────────────────${NC}"
    echo -e "${BOLD}Step 1: Configure DNS Records${NC}"
    echo -e "${BOLD}───────────────────────────────────────────────────────────${NC}"
    echo ""
    echo -e "Add the following ${GREEN}A records${NC} to your DNS provider:"
    echo ""
    echo -e "  ${CYAN}@${NC}       IN A    ${YELLOW}$public_ip${NC}"
    echo -e "  ${CYAN}matrix${NC}  IN A    ${YELLOW}$public_ip${NC}"
    echo ""
    echo -e "Examples:"
    echo -e "  ${domain}.           IN A    $public_ip"
    echo -e "  matrix.${domain}.    IN A    $public_ip"
    echo ""
    echo -e "${BOLD}───────────────────────────────────────────────────────────${NC}"
    echo -e "${BOLD}Step 2: Configure SSL/TLS Certificates${NC}"
    echo -e "${BOLD}───────────────────────────────────────────────────────────${NC}"
    echo ""
    echo -e "${YELLOW}⚠${NC} You need to configure SSL/TLS certificates for your domain."
    echo ""
    echo -e "${BOLD}Recommended Options:${NC}"
    echo -e "  • ${GREEN}Let's Encrypt${NC} - Free, automated certificates"
    echo -e "  • ${GREEN}Caddy${NC} - Automatic HTTPS with simple config"
    echo -e "  • ${GREEN}Certbot${NC} - ACME client for Let's Encrypt"
    echo ""
    echo -e "${BOLD}Certificate Requirements:${NC}"
    echo -e "  • ${CYAN}$domain${NC} (for ArmorClaw Bridge)"
    echo -e "  • ${CYAN}matrix.$domain${NC} (for Matrix Conduit)"
    echo ""
    echo -e "${BOLD}───────────────────────────────────────────────────────────${NC}"
    echo ""
    echo -e "${YELLOW}⚠${NC} After configuring DNS, it may take ${BOLD}5-30 minutes${NC} to propagate."
    echo -e "${YELLOW}⚠${NC} You can verify DNS propagation with: ${CYAN}nslookup $domain${NC} or ${CYAN}dig $domain${NC}"
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""

    # Wait for user confirmation
    read -p "Press Enter once you have configured DNS and SSL/TLS..."

    # Export variables for downstream use
    export PUBLIC_URL="https://$domain"
    export MATRIX_URL="https://matrix.$domain"
    export DOMAIN="$domain"

    log_info "Environment variables exported:"
    log_info "  PUBLIC_URL=$PUBLIC_URL"
    log_info "  MATRIX_URL=$MATRIX_URL"
    log_info "  DOMAIN=$DOMAIN"

    return 0
}

create_cloudflare_origin_cert() {
    log_info "Creating Cloudflare origin certificate..."

    # Check for API token
    if [ -z "$CF_API_TOKEN" ]; then
        log_error "CF_API_TOKEN environment variable is required"
        log_error "Set it with: export CF_API_TOKEN=your_token"
        return 1
    fi

    # Define certificate paths
    local cert_path="/etc/ssl/certs/cloudflare-origin.pem"
    local key_path="/etc/ssl/private/cloudflare-origin.key"
    local api_url="https://api.cloudflare.com/client/v4/certificates"

    # Backup existing certificates if they exist
    local backup_needed=0
    if [ -f "$cert_path" ] || [ -f "$key_path" ]; then
        backup_needed=1
        local timestamp
        timestamp=$(date +%Y%m%d%H%M%S)
        log_warn "Existing certificates found - creating backups..."

        if [ -f "$cert_path" ]; then
            cp "$cert_path" "${cert_path}.backup-${timestamp}" || {
                log_error "Failed to backup certificate"
                return 1
            }
            log_info "Backed up: ${cert_path}.backup-${timestamp}"
        fi

        if [ -f "$key_path" ]; then
            cp "$key_path" "${key_path}.backup-${timestamp}" || {
                log_error "Failed to backup private key"
                return 1
            }
            log_info "Backed up: ${key_path}.backup-${timestamp}"
        fi
    fi

    # Ensure certificate directories exist
    log_info "Ensuring SSL directories exist..."
    if command -v sudo >/dev/null 2>&1; then
        sudo mkdir -p "$(dirname "$cert_path")" "$(dirname "$key_path")" || {
            log_error "Failed to create SSL directories"
            return 1
        }
    else
        mkdir -p "$(dirname "$cert_path")" "$(dirname "$key_path")" || {
            log_error "Failed to create SSL directories"
            return 1
        }
    fi

    # Generate 15-year validity origin certificate via Cloudflare API
    log_info "Requesting origin certificate from Cloudflare API (15-year validity)..."

    local api_response
    api_response=$(curl -s -X POST "$api_url" \
        -H "Authorization: Bearer $CF_API_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "type": "origin",
            "hostnames": ["*"],
            "validity_days": 5479,
            "requested_validity": 5479,
            "request_type": "keyless-certificate"
        }' 2>&1)

    # Check for curl errors
    if [ $? -ne 0 ]; then
        log_error "Failed to communicate with Cloudflare API"
        log_error "Response: $api_response"
        return 1
    fi

    # Parse API response
    local success
    success=$(echo "$api_response" | jq -r '.success // false' 2>/dev/null)

    if [ "$success" != "true" ]; then
        log_error "Cloudflare API returned error"
        local errors
        errors=$(echo "$api_response" | jq -r '.errors[]?.message // "Unknown error"' 2>/dev/null)
        log_error "Details: $errors"
        return 1
    fi

    # Extract certificate and key from response
    local certificate
    local private_key

    certificate=$(echo "$api_response" | jq -r '.result.certificate' 2>/dev/null)
    private_key=$(echo "$api_response" | jq -r '.result.private_key' 2>/dev/null)

    if [ -z "$certificate" ] || [ -z "$private_key" ]; then
        log_error "Failed to extract certificate or private key from API response"
        return 1
    fi

    # Save certificate
    log_info "Saving certificate to $cert_path"
    if command -v sudo >/dev/null 2>&1; then
        echo "$certificate" | sudo tee "$cert_path" >/dev/null || {
            log_error "Failed to save certificate to $cert_path"
            return 1
        }
        sudo chmod 644 "$cert_path" || {
            log_error "Failed to set permissions on certificate"
            return 1
        }
    else
        echo "$certificate" > "$cert_path" || {
            log_error "Failed to save certificate to $cert_path"
            return 1
        }
        chmod 644 "$cert_path" || {
            log_error "Failed to set permissions on certificate"
            return 1
        }
    fi

    # Save private key
    log_info "Saving private key to $key_path"
    if command -v sudo >/dev/null 2>&1; then
        echo "$private_key" | sudo tee "$key_path" >/dev/null || {
            log_error "Failed to save private key to $key_path"
            return 1
        }
        sudo chmod 600 "$key_path" || {
            log_error "Failed to set permissions on private key"
            return 1
        }
    else
        echo "$private_key" > "$key_path" || {
            log_error "Failed to save private key to $key_path"
            return 1
        }
        chmod 600 "$key_path" || {
            log_error "Failed to set permissions on private key"
            return 1
        }
    fi

    log_info "✓ Cloudflare origin certificate created successfully"
    log_info "  Certificate: $cert_path"
    log_info "  Private Key:  $key_path"

    if [ "$backup_needed" -eq 1 ]; then
        log_info "  Backups created (with timestamp)"
    fi

    # Return paths to caller
    echo "$cert_path:$key_path"
    return 0
}

# =============================================================================
# Cloudflare Proxy Setup Functions
# =============================================================================

# Main function to orchestrate Cloudflare Proxy mode setup
# This function performs the complete setup for Cloudflare Proxy mode:
#   - Verifies ports 80/443 are accessible
#   - Creates DNS A records with proxy enabled
#   - Generates Cloudflare origin certificate
#   - Configures Caddy with origin certificate
#   - Displays SSL/TLS configuration instructions
#
# Arguments:
#   None (prompts for domain and IP address if not provided)
#
# Environment Variables Required:
#   - CF_API_TOKEN: Cloudflare API token with Zone:DNS and Zone:SSL edit permissions
#
# Environment Variables Exported on Success:
#   - PUBLIC_URL: The public HTTPS URL (e.g., https://example.com)
#   - MATRIX_URL: The Matrix homeserver URL (e.g., https://matrix.example.com)
#   - DOMAIN: The base domain (e.g., example.com)
#   - CF_ZONE_ID: The Cloudflare zone ID
#
# Returns:
#   0 on success, 1 on failure
#
# Example:
#   export CF_API_TOKEN=your_api_token
#   setup_cloudflare_proxy
setup_cloudflare_proxy() {
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Setting up Cloudflare Proxy mode..."
        echo ""

        local domain
        if [ -n "${1:-}" ]; then
            domain="$1"
        else
            while true; do
                read -p "Enter your domain (e.g., example.com or sub.example.com): " domain
                domain=$(echo "$domain" | xargs)

                if [ -z "$domain" ]; then
                    log_error "Domain cannot be empty"
                    continue
                fi

                if ! echo "$domain" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$'; then
                    log_error "Invalid domain format. Please use format like example.com or sub.example.com"
                    continue
                fi

                break
            done
        fi

        local public_ip="203.0.113.1"
        local mock_zone_id="mock-zone-id-$(echo "$domain" | md5sum | cut -c1-8)"

        log_info "[DRY-RUN] Using domain: $domain"
        log_info "[DRY-RUN] Using mock public IP: $public_ip"
        log_info "[DRY-RUN] Would perform:"
        log_info "[DRY-RUN]   1. Verify domain uses Cloudflare nameservers"
        log_info "[DRY-RUN]   2. Detect public IP address"
        log_info "[DRY-RUN]   3. Verify ports 80 and 443 are accessible"
        log_info "[DRY-RUN]   4. Create DNS A records with proxy enabled (orange cloud)"
        log_info "[DRY-RUN]   5. Generate Cloudflare origin certificate"
        log_info "[DRY-RUN]   6. Configure Caddy with origin certificate"
        log_info "[DRY-RUN]   7. Display SSL/TLS configuration instructions"

        log_info "[DRY-RUN] Would create DNS records:"
        log_info "[DRY-RUN]   - $domain -> $public_ip (proxy enabled)"
        log_info "[DRY-RUN]   - matrix.$domain -> $public_ip (proxy enabled)"

        log_info "[DRY-RUN] Would export:"
        log_info "[DRY-RUN]   PUBLIC_URL=https://$domain"
        log_info "[DRY-RUN]   MATRIX_URL=https://matrix.$domain"
        log_info "[DRY-RUN]   DOMAIN=$domain"
        log_info "[DRY-RUN]   CF_ZONE_ID=$mock_zone_id"

        export PUBLIC_URL="https://$domain"
        export MATRIX_URL="https://matrix.$domain"
        export DOMAIN="$domain"
        export CF_ZONE_ID="$mock_zone_id"

        log_info "[DRY-RUN] ✓ Dry-run completed - no changes made"

        return 0
    fi

    log_info "Setting up Cloudflare Proxy mode..."
    echo ""

    if [ -z "$CF_API_TOKEN" ]; then
        log_error "CF_API_TOKEN environment variable is required"
        log_error "Set it with: export CF_API_TOKEN=your_token"
        return 1
    fi

    check_existing_web_server

    local domain
    while true; do
        read -p "Enter your domain (e.g., example.com or sub.example.com): " domain
        domain=$(echo "$domain" | xargs)

        if [ -z "$domain" ]; then
            log_error "Domain cannot be empty"
            continue
        fi

        if ! echo "$domain" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$'; then
            log_error "Invalid domain format. Please use format like example.com or sub.example.com"
            continue
        fi

        break
    done

    log_info "Using domain: $domain"

    local public_ip
    log_info "Detecting public IP address..."
    public_ip=$(curl -s --max-time 5 --connect-timeout 3 https://api.ipify.org 2>/dev/null || echo "")

    if [ -z "$public_ip" ]; then
        log_warn "Could not detect public IP automatically"
        while true; do
            read -p "Enter your public IP address: " public_ip
            public_ip=$(echo "$public_ip" | xargs)

            if [ -z "$public_ip" ]; then
                log_error "Public IP cannot be empty"
                continue
            fi

            if ! echo "$public_ip" | grep -qE '^([0-9]{1,3}\.){3}[0-9]{1,3}$'; then
                log_error "Invalid IP format. Please use format like 1.2.3.4"
                continue
            fi

            break
        done
    else
        log_info "Public IP detected: $public_ip"
    fi

    log_info "Verifying ports 80 and 443 are accessible..."

    local port_80_accessible=0
    local port_443_accessible=0

    if command -v timeout >/dev/null 2>&1 && command -v nc >/dev/null 2>&1; then
        if timeout 3 nc -z "$public_ip" 80 2>/dev/null; then
            port_80_accessible=1
            log_info "✓ Port 80 is accessible"
        else
            log_warn "✗ Port 80 is not accessible from public"
        fi
    else
        log_warn "Cannot test port 80 (missing nc command), assuming accessible"
        port_80_accessible=1
    fi

    if command -v timeout >/dev/null 2>&1 && command -v nc >/dev/null 2>&1; then
        if timeout 3 nc -z "$public_ip" 443 2>/dev/null; then
            port_443_accessible=1
            log_info "✓ Port 443 is accessible"
        else
            log_warn "✗ Port 443 is not accessible from public"
        fi
    else
        log_warn "Cannot test port 443 (missing nc command), assuming accessible"
        port_443_accessible=1
    fi

    if [ "$port_80_accessible" -eq 0 ] && [ "$port_443_accessible" -eq 0 ]; then
        log_error "Neither port 80 nor 443 is accessible from public internet"
        log_error "Cloudflare Proxy mode requires at least one of these ports to be open"
        log_error "Please configure your firewall/router to allow incoming traffic on ports 80 and 443"
        log_error "Or consider using Tunnel mode instead (bypasses port requirements)"
        return 1
    fi

    log_info "✓ Port verification completed"

    handle_ns_check_warning "$domain"

    log_info "Creating DNS A records with Cloudflare proxy enabled..."
    if ! create_dns_a_record "$domain" "$public_ip" "$CF_API_TOKEN"; then
        log_error "Failed to create DNS A records"
        return 1
    fi
    log_info "✓ DNS A records created successfully"

    wait_for_dns_propagation "$domain" "$public_ip" 600

    log_info "Generating Cloudflare origin certificate..."
    local cert_key_paths
    if ! cert_key_paths=$(create_cloudflare_origin_cert); then
        log_error "Failed to create Cloudflare origin certificate"
        return 1
    fi

    local cert_path key_path
    cert_path=$(echo "$cert_key_paths" | cut -d: -f1)
    key_path=$(echo "$cert_key_paths" | cut -d: -f2)

    log_info "✓ Origin certificate created successfully"

    log_info "Configuring Caddy with Cloudflare origin certificate..."
    if ! configure_caddy_origin "$domain" "$cert_path" "$key_path"; then
        log_error "Failed to configure Caddy"
        return 1
    fi
    log_info "✓ Caddy configured successfully"

    echo ""
    echo -e "${BOLD}═════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}         Cloudflare SSL/TLS Configuration${NC}"
    echo -e "${BOLD}═════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BOLD}IMPORTANT: Set SSL/TLS Mode to ${GREEN}Full (strict)${NC}${BOLD}${NC}"
    echo ""
    echo -e "${BOLD}Steps to configure:${NC}"
    echo ""
    echo -e "  1. Go to: ${CYAN}https://dash.cloudflare.com${NC}"
    echo -e "  2. Select your domain: ${CYAN}$domain${NC}"
    echo -e "  3. Navigate to ${BOLD}SSL/TLS${NC} → ${BOLD}Overview${NC}"
    echo -e "  4. Set SSL/TLS encryption mode to: ${GREEN}Full (strict)${NC}"
    echo ""
    echo -e "${BOLD}Why Full (strict)?${NC}"
    echo -e "  • ${YELLOW}Flexible${NC} - Cloudflare to origin: HTTP (less secure)"
    echo -e "  • ${YELLOW}Full${NC} - Cloudflare to origin: HTTPS (any cert)"
    echo -e "  • ${GREEN}Full (strict)${NC} - Cloudflare to origin: HTTPS (valid cert only)"
    echo ""
    echo -e "${BOLD}With Full (strict):${NC}"
    echo -e "  • End-to-end encryption between client and your server"
    echo -e "  • Origin certificate validates authenticity"
    echo -e "  • Maximum security for your deployment"
    echo ""
    echo -e "${YELLOW}⚠${NC} Do NOT use 'Off' or 'Flexible' - these will break your setup!"
    echo ""
    echo -e "${BOLD}═════════════════════════════════════════════════════════${NC}"
    echo ""

    export PUBLIC_URL="https://$domain"
    export MATRIX_URL="https://matrix.$domain"
    export DOMAIN="$domain"

    log_info "Environment variables exported:"
    log_info "  PUBLIC_URL=$PUBLIC_URL"
    log_info "  MATRIX_URL=$MATRIX_URL"
    log_info "  DOMAIN=$DOMAIN"
    log_info "  CF_ZONE_ID=$CF_ZONE_ID"

    echo ""
    log_info "✓ Cloudflare Proxy mode setup completed successfully"
    echo ""
    echo -e "${BOLD}Next Steps:${NC}"
    echo -e "  1. Set SSL/TLS mode to ${GREEN}Full (strict)${NC} in Cloudflare dashboard"
    echo -e "  2. Restart Caddy to apply new configuration"
    echo -e "  3. Verify your site is accessible at ${CYAN}https://$domain${NC}"
    echo ""

    return 0
}

# =============================================================================
# Edge Case Handler Functions
# =============================================================================

# Non-blocking NS check warning - warns but doesn't block setup
# Arguments:
#   $1 - domain to check
# Returns:
#   0 (always returns success - non-blocking)
handle_ns_check_warning() {
    local domain="$1"

    if [ -z "$domain" ]; then
        return 0
    fi

    log_info "Checking nameservers (non-blocking)..."
    local nameservers
    nameservers=$(dig +short NS "$domain" 2>/dev/null)

    if [ -z "$nameservers" ]; then
        log_warn "Could not retrieve nameservers for $domain"
        return 0
    fi

    local cf_ns_pattern="\.cloudflare\.com\."
    local is_cloudflare=0

    while IFS= read -r ns; do
        if echo "$ns" | grep -qiE "$cf_ns_pattern"; then
            is_cloudflare=1
            break
        fi
    done <<< "$nameservers"

    if [ "$is_cloudflare" -eq 1 ]; then
        log_info "✓ Domain $domain is using Cloudflare nameservers"
    else
        log_warn "⚠ Domain $domain is NOT using Cloudflare nameservers"
        log_warn "  Current nameservers:"
        while IFS= read -r ns; do
            log_warn "    - $ns"
        done <<< "$nameservers"
        log_warn "  Warning: Proxy mode requires Cloudflare nameservers for full functionality."
        log_warn "  You may proceed, but some features may not work as expected."
        log_warn "  Consider using Tunnel mode instead for better compatibility."
    fi

    return 0
}

# Check for existing cloudflared tunnels and reuse them
# Arguments:
#   $1 - tunnel name to check
# Returns:
#   0 if tunnel exists (with ID in CF_TUNNEL_ID), 1 if not found
check_existing_tunnel() {
    local tunnel_name="$1"

    if [ -z "$tunnel_name" ]; then
        return 1
    fi

    if ! command -v cloudflared >/dev/null 2>&1; then
        return 1
    fi

    log_info "Checking for existing tunnel: $tunnel_name"

    local tunnel_info
    if tunnel_info=$(cloudflared tunnel list 2>/dev/null | grep "$tunnel_name"); then
        local tunnel_id
        tunnel_id=$(echo "$tunnel_info" | awk '{print $1}')
        log_info "Found existing tunnel: $tunnel_name (ID: $tunnel_id)"
        export CF_TUNNEL_ID="$tunnel_id"
        return 0
    fi

    return 1
}

# Handle authentication timeout - provide manual URL fallback
# Arguments:
#   $1 - timeout in seconds (default: 300)
# Returns:
#   0 on success, 1 on failure
handle_auth_timeout() {
    local timeout="${1:-300}"
    local elapsed=0
    local interval=10

    log_info "Starting Cloudflare authentication (timeout: ${timeout}s)..."

    # Start cloudflared login in background
    cloudflared tunnel login &
    local auth_pid=$!

    log_info "Waiting for authentication..."
    log_info "  Please open the URL displayed by cloudflared in your browser"

    while [ $elapsed -lt $timeout ]; do
        if ! kill -0 $auth_pid 2>/dev/null; then
            # Process finished
            wait $auth_pid
            local exit_code=$?
            if [ $exit_code -eq 0 ]; then
                log_info "✓ Authentication successful"
                return 0
            else
                log_error "Authentication failed with exit code: $exit_code"
                return 1
            fi
        fi

        sleep $interval
        elapsed=$((elapsed + interval))

        if [ $((elapsed % 30)) -eq 0 ] && [ $elapsed -gt 0 ]; then
            log_info "Still waiting for authentication... (${elapsed}s elapsed)"
        fi
    done

    # Timeout reached
    log_warn "Authentication timed out after ${timeout}s"
    log_warn "Killing background process..."

    if kill -0 $auth_pid 2>/dev/null; then
        kill $auth_pid 2>/dev/null
        wait $auth_pid 2>/dev/null
    fi

    echo ""
    log_warn "========================================="
    log_warn "MANUAL FALLBACK INSTRUCTIONS"
    log_warn "========================================="
    log_warn "The automated authentication timed out."
    log_warn "Please complete authentication manually:"
    echo ""
    log_info "1. Run this command in a separate terminal:"
    echo -e "   ${CYAN}cloudflared tunnel login${NC}"
    echo ""
    log_info "2. Open the URL displayed in your browser"
    log_info "3. Authorize Cloudflare access"
    log_info "4. Return here and press Enter when done"
    log_warn "========================================="
    echo ""

    read -p "Press Enter once authentication is complete: "

    # Verify certificate exists
    local cert_file="$HOME/.cloudflared/cert.pem"
    if [ -f "$cert_file" ]; then
        log_info "✓ Certificate found: $cert_file"
        return 0
    else
        log_error "Certificate not found after manual authentication"
        return 1
    fi
}

# Wait for DNS propagation with timeout
# Arguments:
#   $1 - domain to check
#   $2 - expected IP address (optional)
#   $3 - timeout in seconds (default: 600)
# Returns:
#   0 if DNS propagated, 1 if timeout
wait_for_dns_propagation() {
    local domain="$1"
    local expected_ip="$2"
    local timeout="${3:-600}"
    local elapsed=0
    local interval=15
    local check_count=0

    if [ -z "$domain" ]; then
        log_error "Domain parameter is required"
        return 1
    fi

    log_info "Waiting for DNS propagation for $domain..."
    if [ -n "$expected_ip" ]; then
        log_info "  Expected IP: $expected_ip"
    fi
    log_info "  Timeout: ${timeout}s"

    while [ $elapsed -lt $timeout ]; do
        local resolved_ip
        resolved_ip=$(dig +short "$domain" @8.8.8.8 2>/dev/null | head -1)

        if [ -n "$resolved_ip" ]; then
            check_count=$((check_count + 1))

            if [ -n "$expected_ip" ]; then
                if [ "$resolved_ip" = "$expected_ip" ]; then
                    log_info "✓ DNS propagated successfully ($resolved_ip)"
                    return 0
                else
                    log_warn "DNS record found but IP mismatch:"
                    log_warn "  Expected: $expected_ip"
                    log_warn "  Found: $resolved_ip"
                fi
            else
                log_info "✓ DNS record resolved: $resolved_ip"
                return 0
            fi
        else
            log_info "DNS not yet propagated... ($elapsed/${timeout}s)"
        fi

        sleep $interval
        elapsed=$((elapsed + interval))

        if [ $((elapsed % 60)) -eq 0 ] && [ $elapsed -gt 0 ]; then
            log_info "Still waiting for DNS propagation... (${elapsed}s elapsed, ${check_count} checks performed)"
        fi
    done

    log_error "DNS propagation timed out after ${timeout}s"
    log_warn "DNS may still be propagating. You can continue, but services may not be accessible yet."
    log_warn "Check again with: dig +short $domain @8.8.8.8"
    return 1
}

# Handle tunnel/proxy service failure - show logs and recovery instructions
# Arguments:
#   $1 - service name (cloudflared or caddy)
# Returns:
#   0 (always returns - informational only)
handle_service_failure() {
    local service_name="$1"

    if [ -z "$service_name" ]; then
        log_error "Service name required"
        return 0
    fi

    log_error "========================================="
    log_error "SERVICE FAILURE: $service_name"
    log_error "========================================="

    # Show recent logs
    log_info "Recent logs (last 20 lines):"
    echo ""

    if command -v journalctl >/dev/null 2>&1 && command -v sudo >/dev/null 2>&1; then
        sudo journalctl -u "$service_name" -n 20 --no-pager 2>/dev/null || true
    elif command -v journalctl >/dev/null 2>&1; then
        journalctl -u "$service_name" -n 20 --no-pager 2>/dev/null || true
    else
        log_warn "Cannot show logs (journalctl not available)"
    fi

    echo ""
    log_warn "========================================="
    log_warn "RECOVERY INSTRUCTIONS"
    log_warn "========================================="
    log_info "1. Check service status:"
    echo -e "   ${CYAN}systemctl status $service_name${NC}"
    echo ""
    log_info "2. View full logs:"
    echo -e "   ${CYAN}journalctl -u $service_name -f${NC}"
    echo ""
    log_info "3. Restart the service:"
    echo -e "   ${CYAN}sudo systemctl restart $service_name${NC}"
    echo ""
    log_info "4. Check configuration:"
    case "$service_name" in
        cloudflared)
            echo -e "   ${CYAN}cat ~/.cloudflared/config.yml${NC}"
            ;;
        caddy)
            echo -e "   ${CYAN}cat /etc/caddy/Caddyfile${NC}"
            echo -e "   ${CYAN}caddy validate --config /etc/caddy/Caddyfile${NC}"
            ;;
    esac
    echo ""
    log_info "5. Test connectivity:"
    case "$service_name" in
        cloudflared)
            echo -e "   ${CYAN}cloudflared tunnel info <tunnel-name>${NC}"
            ;;
        caddy)
            echo -e "   ${CYAN}curl -I https://localhost${NC}"
            ;;
    esac
    log_warn "========================================="

    return 0
}

# Cleanup partial state on cancel - remove temp files, stop partial services
# Arguments:
#   $1 - cleanup type (tunnel, proxy, all)
# Returns:
#   0 on success, 1 on failure
cleanup_on_cancel() {
    local cleanup_type="${1:-all}"

    log_warn "========================================="
    log_warn "CLEANING UP PARTIAL STATE"
    log_warn "========================================="

    local cleanup_count=0

    # Clean up temp files
    log_info "Cleaning up temporary files..."

    local temp_patterns=(
        "/tmp/cloudflared-*"
        "/tmp/caddy-*"
        "/tmp/armorclaw-cloudflare-*"
    )

    for pattern in "${temp_patterns[@]}"; do
        for file in $pattern; do
            if [ -e "$file" ]; then
                rm -rf "$file" 2>/dev/null && {
                    log_info "  Removed: $file"
                    cleanup_count=$((cleanup_count + 1))
                }
            fi
        done
    done

    # Stop partial services based on cleanup type
    case "$cleanup_type" in
        tunnel|all)
            if systemctl is-active --quiet cloudflared-tunnel 2>/dev/null; then
                log_info "Stopping cloudflared-tunnel service..."
                if command -v sudo >/dev/null 2>&1; then
                    sudo systemctl stop cloudflared-tunnel 2>/dev/null && {
                        log_info "  ✓ Stopped cloudflared-tunnel"
                        cleanup_count=$((cleanup_count + 1))
                    }
                else
                    systemctl stop cloudflared-tunnel 2>/dev/null && {
                        log_info "  ✓ Stopped cloudflared-tunnel"
                        cleanup_count=$((cleanup_count + 1))
                    }
                fi
            fi
            ;;
        proxy|all)
            if systemctl is-active --quiet caddy 2>/dev/null; then
                log_info "Stopping caddy service..."
                if command -v sudo >/dev/null 2>&1; then
                    sudo systemctl stop caddy 2>/dev/null && {
                        log_info "  ✓ Stopped caddy"
                        cleanup_count=$((cleanup_count + 1))
                    }
                else
                    systemctl stop caddy 2>/dev/null && {
                        log_info "  ✓ Stopped caddy"
                        cleanup_count=$((cleanup_count + 1))
                    }
                fi
            fi
            ;;
    esac

    # Clean up systemd service files if requested
    if [ "$cleanup_type" = "all" ]; then
        log_info "Cleaning up systemd service files..."

        local service_files=(
            "/etc/systemd/system/cloudflared-tunnel.service"
        )

        for service_file in "${service_files[@]}"; do
            if [ -f "$service_file" ]; then
                if command -v sudo >/dev/null 2>&1; then
                    sudo rm -f "$service_file" 2>/dev/null && {
                        log_info "  Removed: $service_file"
                        cleanup_count=$((cleanup_count + 1))
                    }
                else
                    rm -f "$service_file" 2>/dev/null && {
                        log_info "  Removed: $service_file"
                        cleanup_count=$((cleanup_count + 1))
                    }
                fi
            fi
        done

        # Reload systemd
        log_info "Reloading systemd daemon..."
        if command -v sudo >/dev/null 2>&1; then
            sudo systemctl daemon-reload 2>/dev/null
        else
            systemctl daemon-reload 2>/dev/null
        fi
    fi

    log_warn "========================================="
    log_info "Cleanup completed: $cleanup_count items"
    log_warn "========================================="

    return 0
}

# Check if cloudflared is already installed
# Returns:
#   0 if installed (version in CF_CLOUDFLARED_VERSION), 1 if not
check_cloudflared_installed() {
    if ! command -v cloudflared >/dev/null 2>&1; then
        return 1
    fi

    local version
    version=$(cloudflared --version 2>&1 | head -1)

    if [ -n "$version" ]; then
        log_info "✓ cloudflared is already installed"
        log_info "  Version: $version"
        export CF_CLOUDFLARED_VERSION="$version"
        return 0
    fi

    return 1
}

# Check for existing Caddy/Nginx web servers and warn about conflicts
# Returns:
#   0 (always returns - informational only)
check_existing_web_server() {
    local has_caddy=0
    local has_nginx=0
    local conflicts_found=0

    log_info "Checking for existing web servers..."

    # Check for Caddy
    if command -v caddy >/dev/null 2>&1; then
        has_caddy=1
        if systemctl is-active --quiet caddy 2>/dev/null; then
            conflicts_found=$((conflicts_found + 1))
            log_warn "⚠ Caddy is running and active"
            log_warn "  Binary: $(command -v caddy)"
        else
            log_info "✓ Caddy is installed but not running"
        fi
    fi

    # Check for Nginx
    if command -v nginx >/dev/null 2>&1; then
        has_nginx=1
        if systemctl is-active --quiet nginx 2>/dev/null; then
            conflicts_found=$((conflicts_found + 1))
            log_warn "⚠ Nginx is running and active"
            log_warn "  Binary: $(command -v nginx)"
        else
            log_info "✓ Nginx is installed but not running"
        fi
    fi

    # Check for port conflicts
    if command -v ss >/dev/null 2>&1; then
        local port_80_in_use
        local port_443_in_use
        port_80_in_use=$(ss -tlnp 2>/dev/null | grep ':80 ' || echo "")
        port_443_in_use=$(ss -tlnp 2>/dev/null | grep ':443 ' || echo "")

        if [ -n "$port_80_in_use" ]; then
            log_warn "⚠ Port 80 is already in use:"
            echo "$port_80_in_use" | head -1 | sed 's/^/    /'
        fi

        if [ -n "$port_443_in_use" ]; then
            log_warn "⚠ Port 443 is already in use:"
            echo "$port_443_in_use" | head -1 | sed 's/^/    /'
        fi
    elif command -v netstat >/dev/null 2>&1; then
        local port_80_in_use
        local port_443_in_use
        port_80_in_use=$(netstat -tlnp 2>/dev/null | grep ':80 ' || echo "")
        port_443_in_use=$(netstat -tlnp 2>/dev/null | grep ':443 ' || echo "")

        if [ -n "$port_80_in_use" ]; then
            log_warn "⚠ Port 80 is already in use:"
            echo "$port_80_in_use" | head -1 | sed 's/^/    /'
        fi

        if [ -n "$port_443_in_use" ]; then
            log_warn "⚠ Port 443 is already in use:"
            echo "$port_443_in_use" | head -1 | sed 's/^/    /'
        fi
    fi

    if [ $conflicts_found -gt 0 ]; then
        echo ""
        log_warn "========================================="
        log_warn "WEB SERVER CONFLICT DETECTED"
        log_warn "========================================="
        log_warn "You have existing web servers that may conflict with:"
        log_warn "  • Caddy (for Proxy mode)"
        log_warn "  • Cloudflare Tunnel (for Tunnel mode)"
        echo ""
        log_info "Recommended actions:"
        if [ $has_nginx -eq 1 ] && [ $has_caddy -eq 0 ]; then
            log_info "  • Nginx users: Consider using existing Nginx as reverse proxy"
            log_info "  • Or stop Nginx before proceeding: sudo systemctl stop nginx"
        elif [ $has_caddy -eq 1 ] && [ $has_nginx -eq 0 ]; then
            log_info "  • Caddy users: ArmorClaw will configure your existing Caddy"
            log_info "  • Ensure Caddy is not already configured for ports 80/443"
        elif [ $has_nginx -eq 1 ] && [ $has_caddy -eq 1 ]; then
            log_warn "  • Both servers detected! Choose one or disable both"
            log_info "  • Stop one: sudo systemctl stop nginx|caddy"
        fi
        log_info "  • Tunnel mode bypasses most web server conflicts"
        log_warn "========================================="
        echo ""
    else
        log_info "✓ No web server conflicts detected"
    fi

    return 0
}
