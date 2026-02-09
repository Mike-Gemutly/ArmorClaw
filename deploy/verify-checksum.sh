#!/bin/bash
# =============================================================================
# ArmorClaw Binary Verification Script
# Purpose: Verify binary integrity using SHA256 checksums
# Usage: ./verify-checksum.sh <binary-file> <checksum-file>
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    cat <<EOF
Usage: $0 [OPTIONS] <binary-file>

Verify ArmorClaw binary integrity using SHA256 checksum.

Arguments:
  binary-file          Path to the binary to verify

Options:
  -c, --checksum FILE  Read checksum from FILE (default: SHA256SUMS)
  -s, --sum SUM        Use specific checksum value
  -g, --generate       Generate checksum file for binary
  -q, --quiet          Quiet mode (only output errors)
  -h, --help           Show this help message

Examples:
  # Verify binary against SHA256SUMS file
  $0 armorclaw-bridge

  # Verify with specific checksum
  $0 -s abc123... armorclaw-bridge

  # Generate checksum file
  $0 -g armorclaw-bridge

EOF
}

generate_checksum() {
    local binary="$1"

    if [[ ! -f "$binary" ]]; then
        log_error "Binary not found: $binary"
        exit 1
    fi

    local filename=$(basename "$binary")
    local checksum=$(sha256sum "$binary" | awk '{print $1}')

    # Write to SHA256SUMS file
    echo "$checksum  $filename" > SHA256SUMS

    log_success "Generated checksum:"
    echo ""
    echo "  $checksum  $filename"
    echo ""
    log_info "Written to: SHA256SUMS"
    log_info "Include this file with your release"
}

verify_checksum_file() {
    local binary="$1"
    local checksum_file="$2"
    local quiet="$3"

    if [[ ! -f "$binary" ]]; then
        log_error "Binary not found: $binary"
        exit 1
    fi

    if [[ ! -f "$checksum_file" ]]; then
        log_error "Checksum file not found: $checksum_file"
        exit 1
    fi

    # Run sha256sum verification
    local output
    output=$(sha256sum -c "$checksum_file" 2>&1 || true)

    # Check if our binary verified
    if echo "$output" | grep -q "$(basename "$binary")": OK; then
        [[ "$quiet" != "true" ]] && log_success "Checksum verified: $(basename "$binary")"
        exit 0
    else
        log_error "Checksum verification failed"
        echo "$output"
        exit 1
    fi
}

verify_checksum_value() {
    local binary="$1"
    local expected_sum="$2"
    local quiet="$3"

    if [[ ! -f "$binary" ]]; then
        log_error "Binary not found: $binary"
        exit 1
    fi

    # Calculate checksum
    local actual_sum=$(sha256sum "$binary" | awk '{print $1}')

    # Compare (case-insensitive for hex)
    if [[ "${actual_sum,,}" == "${expected_sum,,}" ]]; then
        [[ "$quiet" != "true" ]] && log_success "Checksum verified: $(basename "$binary")"
        exit 0
    else
        log_error "Checksum mismatch for $(basename "$binary")"
        echo ""
        echo "  Expected: $expected_sum"
        echo "  Actual:   $actual_sum"
        exit 1
    fi
}

print_binary_info() {
    local binary="$1"

    if [[ ! -f "$binary" ]]; then
        return
    fi

    echo ""
    log_info "Binary Information:"
    echo "  File:     $(basename "$binary")"
    echo "  Size:     $(numfmt --to=iec-i --suffix=B $(stat -c %s "$binary"))"
    echo "  Modified: $(stat -c %y "$binary" | cut -d'.' -f1)"
    echo ""
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    local binary=""
    local checksum_file="SHA256SUMS"
    local checksum_value=""
    local generate_mode=false
    local quiet_mode=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                usage
                exit 0
                ;;
            -c|--checksum)
                checksum_file="$2"
                shift 2
                ;;
            -s|--sum)
                checksum_value="$2"
                shift 2
                ;;
            -g|--generate)
                generate_mode=true
                shift
                ;;
            -q|--quiet)
                quiet_mode=true
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                binary="$1"
                shift
                ;;
        esac
    done

    # Validate binary argument
    if [[ -z "$binary" ]]; then
        log_error "Missing binary argument"
        usage
        exit 1
    fi

    # Generate mode
    if [[ "$generate_mode" == "true" ]]; then
        generate_checksum "$binary"
        print_binary_info "$binary"
        exit 0
    fi

    # Verify mode
    print_binary_info "$binary"

    if [[ -n "$checksum_value" ]]; then
        verify_checksum_value "$binary" "$checksum_value" "$quiet_mode"
    else
        verify_checksum_file "$binary" "$checksum_file" "$quiet_mode"
    fi
}

main "$@"
