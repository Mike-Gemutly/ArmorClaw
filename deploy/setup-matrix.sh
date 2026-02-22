#!/bin/bash
# ArmorClaw Matrix Setup
# Post-setup Matrix configuration script
# Version: 1.0.0
#
# Usage: sudo ./deploy/setup-matrix.sh [--enable] [--domain example.com]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Global variables
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
CONFIG_FILE="$CONFIG_DIR/config.toml"
BRIDGE_SOCK="/run/armorclaw/bridge.sock"

# Command line args
AUTO_ENABLE=false
DOMAIN=""

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    clear 2>/dev/null || true
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Matrix Setup${NC}                            ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}Post-Installation Configuration${NC}                  ${CYAN}║${NC}"
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

prompt_input() {
    local prompt="$1"
    local default="$2"
    local result

    if [[ -n "$default" ]]; then
        echo -ne "  ${CYAN}$prompt [$default]:${NC} "
    else
        echo -ne "  ${CYAN}$prompt:${NC} "
    fi

    read -r result
    echo "${result:-$default}"
}

prompt_yes_no() {
    local prompt="$1"
    local default="${2:-n}"

    echo ""
    echo -ne "  ${CYAN}$prompt [${default^^}/$(echo $default | tr 'yn' 'ny')]${NC}: "
    read -r response
    response=${response:-$default}

    case "$response" in
        [Yy]|[Yy][Ee][Ss]) return 0 ;;
        [Nn]|[Nn][Oo]) return 1 ;;
    esac

    [[ "$default" == "y" ]]
}

#=============================================================================
# Check Prerequisites
#=============================================================================

check_prerequisites() {
    print_step "Checking prerequisites..."

    # Check root
    if [[ $EUID -ne 0 ]]; then
        fail "This script must be run as root (use sudo)"
    fi
    print_success "Running as root"

    # Check if bridge is installed
    if [[ ! -f "$CONFIG_FILE" ]]; then
        fail "ArmorClaw not configured. Run setup-wizard.sh first."
    fi
    print_success "Bridge configuration found"

    # Check if Matrix is already enabled
    if grep -q 'enabled = true' "$CONFIG_FILE" 2>/dev/null | grep -A5 '\[matrix\]'; then
        print_warning "Matrix appears to be already enabled"
        if ! prompt_yes_no "Continue with reconfiguration?" "n"; then
            exit 0
        fi
    fi
}

#=============================================================================
# Matrix Configuration
#=============================================================================

configure_matrix() {
    print_step "Matrix Homeserver Configuration"

    echo ""
    cat <<'EOF'
  Matrix enables secure remote communication with your ArmorClaw agents.

  Requirements:
  • A Matrix homeserver (Conduit, Synapse, or hosted service)
  • A dedicated account for the bridge
  • (Optional) Domain with SSL for production

  Options:
  1. Use existing Matrix server
  2. Deploy local Conduit server (recommended for VPS)
  3. Use Matrix.org (free, but shared infrastructure)
EOF

    echo ""
    local choice=$(prompt_input "Select option [1/2/3]" "2")

    case "$choice" in
        1)
            configure_existing_server
            ;;
        2)
            deploy_local_conduit
            ;;
        3)
            configure_matrix_org
            ;;
        *)
            print_warning "Invalid choice, defaulting to option 2"
            deploy_local_conduit
            ;;
    esac
}

configure_existing_server() {
    print_info "Configuring existing Matrix server..."

    local homeserver_url=$(prompt_input "Homeserver URL (e.g., https://matrix.example.com)" "")
    if [[ -z "$homeserver_url" ]]; then
        fail "Homeserver URL is required"
    fi

    local username=$(prompt_input "Bridge username (e.g., armorclaw)" "armorclaw")
    local password=""
    echo -ne "  ${CYAN}Password:${NC} "
    read -s password
    echo ""

    if [[ -z "$password" ]]; then
        fail "Password is required"
    fi

    MATRIX_URL="$homeserver_url"
    MATRIX_USER="$username"
    MATRIX_PASS="$password"
}

deploy_local_conduit() {
    print_info "Setting up local Conduit server..."

    # Check for docker-compose
    if ! command -v docker-compose &>/dev/null && ! docker compose version &>/dev/null; then
        fail "Docker Compose required for local Conduit deployment"
    fi

    # Get domain
    if [[ -z "$DOMAIN" ]]; then
        DOMAIN=$(prompt_input "Domain for Matrix server (e.g., matrix.example.com)" "")
    fi

    if [[ -z "$DOMAIN" ]]; then
        print_warning "No domain specified - using localhost (not suitable for production)"
        DOMAIN="localhost"
    fi

    # Check for existing deployment
    local compose_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/matrix"

    if [[ -f "$compose_dir/docker-compose.matrix.yml" ]]; then
        print_success "Found existing Matrix deployment configuration"

        if prompt_yes_no "Start Matrix stack now?" "y"; then
            cd "$compose_dir/.."
            docker compose -f "$compose_dir/docker-compose.matrix.yml" up -d
            print_success "Matrix stack started"

            # Wait for server to be ready
            print_info "Waiting for Matrix server..."
            sleep 5

            for i in {1..30}; do
                if curl -sf http://localhost:6167/_matrix/client/versions &>/dev/null; then
                    print_success "Matrix server is ready"
                    break
                fi
                sleep 1
            done
        fi
    else
        print_warning "Matrix deployment files not found at $compose_dir"
        print_info "Please deploy Matrix stack manually first"
    fi

    MATRIX_URL="http://localhost:6167"
    MATRIX_USER="armorclaw"
    MATRIX_PASS=""
}

configure_matrix_org() {
    print_info "Configuring Matrix.org connection..."

    print_warning "Using Matrix.org means your data goes through their servers"
    print_info "For privacy, consider running your own homeserver"

    MATRIX_URL="https://matrix-client.matrix.org"
    MATRIX_USER=$(prompt_input "Your Matrix.org username" "")
    echo -ne "  ${CYAN}Password:${NC} "
    read -s MATRIX_PASS
    echo ""
}

#=============================================================================
# Zero-Trust Configuration
#=============================================================================

configure_zero_trust() {
    print_step "Zero-Trust Security Configuration"

    echo ""
    cat <<'EOF'
  Zero-trust restricts Matrix communication to trusted senders and rooms.

  Wildcards for Trusted Senders:
  • @alice:example.com       - Specific user only
  • *@admin.example.com      - All users from admin domain
  • *:example.com            - Everyone on homeserver

  Security: Empty allowlist = allow all (default for testing)
  Recommendation: Start with specific users, expand as needed
EOF

    echo ""

    if ! prompt_yes_no "Enable zero-trust sender/room filtering?" "n"; then
        TRUSTED_SENDERS=""
        TRUSTED_ROOMS=""
        REJECT_UNTRUSTED="false"
        print_info "Zero-trust disabled (allow all senders)"
        return 0
    fi

    print_info "Enter trusted Matrix user IDs (one per line, empty line to finish):"
    TRUSTED_SENDERS=""
    while IFS= read -r line; do
        [[ -z "$line" ]] && break
        TRUSTED_SENDERS="$TRUSTED_SENDERS\"$line\","
    done

    # Remove trailing comma
    TRUSTED_SENDERS="${TRUSTED_SENDERS%,}"

    echo ""
    print_info "Enter trusted room IDs (one per line, empty line to finish):"
    TRUSTED_ROOMS=""
    while IFS= read -r line; do
        [[ -z "$line" ]] && break
        TRUSTED_ROOMS="$TRUSTED_ROOMS\"$line\","
    done
    TRUSTED_ROOMS="${TRUSTED_ROOMS%,}"

    REJECT_UNTRUSTED="false"
    if prompt_yes_no "Send rejection message to untrusted senders?" "n"; then
        REJECT_UNTRUSTED="true"
    fi
}

#=============================================================================
# Update Configuration
#=============================================================================

update_config() {
    print_step "Updating configuration..."

    # Backup existing config
    cp "$CONFIG_FILE" "$CONFIG_FILE.bak"
    print_info "Backup saved to $CONFIG_FILE.bak"

    # Generate new Matrix section
    local matrix_section="
[matrix]
enabled = true
homeserver_url = \"$MATRIX_URL\"
username = \"$MATRIX_USER\"
password = \"$MATRIX_PASS\"
device_id = \"armorclaw-bridge\"
"

    # Add zero-trust if configured
    if [[ -n "$TRUSTED_SENDERS" ]] || [[ -n "$TRUSTED_ROOMS" ]]; then
        matrix_section+="
[matrix.zero_trust]
trusted_senders = [$TRUSTED_SENDERS]
trusted_rooms = [$TRUSTED_ROOMS]
reject_untrusted = $REJECT_UNTRUSTED
"
    fi

    # Remove existing [matrix] section and add new one
    if grep -q '^\[matrix\]' "$CONFIG_FILE"; then
        # Use awk to replace the matrix section
        awk -v new_section="$matrix_section" '
        /^\[matrix\]/ { in_matrix=1; next }
        /^\[/ && in_matrix { in_matrix=0 }
        !in_matrix { print }
        END { print new_section }
        ' "$CONFIG_FILE.bak" > "$CONFIG_FILE"
    else
        # Append to end
        echo "$matrix_section" >> "$CONFIG_FILE"
    fi

    print_success "Configuration updated"
}

#=============================================================================
# Notifications Configuration
#=============================================================================

configure_notifications() {
    print_step "Notification System"

    echo ""
    cat <<'EOF'
  Enable Matrix-based notifications for system events:
  • Budget alerts - Warning when approaching/exceeding limits
  • Security events - Authentication failures, access denied
  • Container events - Started, stopped, failed, restarted
  • System alerts - Startup, shutdown

  Notifications are sent to a Matrix admin room.
EOF

    echo ""

    if ! prompt_yes_no "Enable notification system?" "n"; then
        print_info "Notifications disabled"
        return 0
    fi

    local admin_room=$(prompt_input "Admin room ID for notifications" "")
    local alert_threshold=$(prompt_input "Alert threshold (percentage)" "80")

    # Update notifications section
    if grep -q '^\[notifications\]' "$CONFIG_FILE"; then
        sed -i "s/^enabled = false/enabled = true/" "$CONFIG_FILE"
        if [[ -n "$admin_room" ]]; then
            sed -i "s/^admin_room_id = \"\"/admin_room_id = \"$admin_room\"/" "$CONFIG_FILE"
        fi
        sed -i "s/^alert_threshold = .*/alert_threshold = $alert_threshold/" "$CONFIG_FILE"
    else
        cat >> "$CONFIG_FILE" <<EOF

[notifications]
enabled = true
admin_room_id = "$admin_room"
alert_threshold = $alert_threshold
EOF
    fi

    print_success "Notifications configured"
}

#=============================================================================
# Test Connection
#=============================================================================

test_connection() {
    print_step "Testing Matrix connection..."

    if [[ ! -S "$BRIDGE_SOCK" ]]; then
        print_warning "Bridge not running - skipping connection test"
        print_info "Start bridge with: systemctl start armorclaw-bridge"
        return 0
    fi

    # Try to get Matrix status via RPC
    local status
    status=$(echo '{"jsonrpc":"2.0","method":"matrix_status","id":1}' | \
        socat - UNIX-CONNECT:"$BRIDGE_SOCK" 2>/dev/null || echo '{"error":"connection failed"}')

    if echo "$status" | grep -q '"connected":true'; then
        print_success "Matrix connection successful"
    else
        print_warning "Matrix not yet connected (may need bridge restart)"
    fi
}

#=============================================================================
# Restart Bridge
#=============================================================================

restart_bridge() {
    print_step "Restarting bridge..."

    if systemctl is-active armorclaw-bridge &>/dev/null; then
        systemctl restart armorclaw-bridge
        print_success "Bridge restarted"

        # Wait for socket
        print_info "Waiting for bridge..."
        local count=0
        while [[ ! -S "$BRIDGE_SOCK" ]] && [[ $count -lt 30 ]]; do
            sleep 0.5
            ((count++))
        done

        if [[ -S "$BRIDGE_SOCK" ]]; then
            print_success "Bridge is ready"
        fi
    else
        print_info "Bridge not running - start with: systemctl start armorclaw-bridge"
    fi
}

#=============================================================================
# Completion
#=============================================================================

print_completion() {
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}                 ${BOLD}Matrix Setup Complete!${NC}                           ${GREEN}║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${BOLD}Matrix Configuration:${NC}"
    echo "  Server:   $MATRIX_URL"
    echo "  Username: $MATRIX_USER"
    echo ""

    echo -e "${BOLD}Next Steps:${NC}"
    echo ""
    echo "  1. ${CYAN}Verify connection${NC}:"
    echo "     sudo journalctl -u armorclaw-bridge -f | grep -i matrix"
    echo ""
    echo "  2. ${CYAN}Create a Matrix room${NC} and invite the bridge user"
    echo ""
    echo "  3. ${CYAN}Test messaging${NC}:"
    echo "     Send a message to the bridge user"
    echo ""
    echo "  4. ${CYAN}Configure zero-trust${NC} (if needed):"
    echo "     Edit $CONFIG_FILE"
    echo "     Add trusted senders/rooms"
    echo "     Restart: systemctl restart armorclaw-bridge"
    echo ""

    echo -e "${BOLD}Documentation:${NC}"
    echo "  Matrix Guide: docs/guides/element-x-quickstart.md"
    echo "  Zero-Trust:   docs/guides/configuration.md#zero-trust"
    echo ""
}

#=============================================================================
# Parse Arguments
#=============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --enable|-e)
                AUTO_ENABLE=true
                shift
                ;;
            --domain|-d)
                DOMAIN="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: sudo $0 [options]"
                echo ""
                echo "Options:"
                echo "  --enable, -e          Enable Matrix with minimal prompts"
                echo "  --domain, -d DOMAIN   Set Matrix server domain"
                echo "  --help, -h            Show this help message"
                exit 0
                ;;
            *)
                print_warning "Unknown option: $1"
                shift
                ;;
        esac
    done
}

#=============================================================================
# Main
#=============================================================================

main() {
    parse_args "$@"

    print_header
    check_prerequisites
    configure_matrix
    configure_zero_trust
    update_config
    configure_notifications
    test_connection
    restart_bridge
    print_completion
}

main "$@"
