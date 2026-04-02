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
    log_info "Setting up Cloudflare Tunnel..."

    if ! command -v cloudflared >/dev/null 2>&1; then
        log_warn "cloudflared not installed, installing now..."
        install_cloudflared
    else
        log_info "cloudflared already installed: $(cloudflared --version 2>&1 | head -1)"
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

        cloudflared tunnel login || die_on_error "Failed to authenticate with Cloudflare. Please run 'cloudflared tunnel login' manually."

        if [ ! -f "$cert_file" ]; then
            die_on_error "Authentication failed - certificate file not found after login"
        fi

        log_info "Authentication successful"
    fi

    local tunnel_name="armorclaw-$base_domain"
    local tunnel_info tunnel_id

    log_info "Checking for existing tunnel: $tunnel_name"

    if tunnel_info=$(cloudflared tunnel list 2>/dev/null | grep "$tunnel_name"); then
        tunnel_id=$(echo "$tunnel_info" | awk '{print $1}')
        log_info "Found existing tunnel: $tunnel_name (ID: $tunnel_id)"
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
        sudo systemctl start cloudflared-tunnel.service || die_on_error "Failed to start cloudflared-tunnel service"
    else
        systemctl daemon-reload || die_on_error "Failed to reload systemd daemon"
        systemctl enable cloudflared-tunnel.service || die_on_error "Failed to enable cloudflared-tunnel service"
        systemctl start cloudflared-tunnel.service || die_on_error "Failed to start cloudflared-tunnel service"
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
