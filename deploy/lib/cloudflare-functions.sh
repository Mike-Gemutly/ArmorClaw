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
