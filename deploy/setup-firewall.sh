#!/usr/bin/bash
# setup-firewall.sh - Configure UFW firewall for ArmorClaw
# This script sets up a deny-all incoming policy with specific exceptions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

# Check if UFW is installed
check_ufw() {
    if ! command -v ufw &> /dev/null; then
        log_info "UFW not found. Installing..."
        apt-get update -qq
        apt-get install -y ufw
        log_success "UFW installed"
    else
        log_info "UFW is already installed"
    fi
}

# Check if Tailscale is installed
check_tailscale() {
    if command -v tailscale &> /dev/null; then
        echo "true"
    else
        echo "false"
    fi
}

# Configure firewall rules
configure_firewall() {
    local ssh_port="${1:-22}"
    local tailscale_installed=$(check_tailscale)

    log_info "Configuring firewall rules..."

    # Reset UFW to default state
    log_info "Resetting UFW to default state..."
    ufw --force reset

    # Set default policies
    log_info "Setting default policies (deny incoming, allow outgoing)..."
    ufw default deny incoming
    ufw default allow outgoing

    # Allow SSH
    log_info "Allowing SSH on port ${ssh_port}..."
    ufw allow "${ssh_port}/tcp" comment 'SSH'

    # Allow Tailscale if installed
    if [ "$tailscale_installed" = "true" ]; then
        log_info "Tailscale detected - allowing UDP port 41641..."
        ufw allow 41641/udp comment 'Tailscale VPN'
    fi

    # Allow localhost
    log_info "Allowing localhost traffic..."
    ufw allow from 127.0.0.1

    # Enable UFW
    log_info "Enabling firewall..."
    echo "y" | ufw enable

    log_success "Firewall configured successfully"
}

# Display firewall status
show_status() {
    echo ""
    log_info "Current firewall status:"
    echo ""
    ufw status numbered
    echo ""
}

# Verify firewall is working
verify_firewall() {
    log_info "Verifying firewall is active..."

    if ufw status | grep -q "Status: active"; then
        log_success "Firewall is active"
        return 0
    else
        log_error "Firewall is not active"
        return 1
    fi
}

# Print security summary
print_summary() {
    local ssh_port="${1:-22}"
    local tailscale_installed=$(check_tailscale)

    echo ""
    echo "=========================================="
    echo "      Firewall Configuration Summary"
    echo "=========================================="
    echo ""
    echo "Default Policies:"
    echo "  • Incoming: DENY"
    echo "  • Outgoing: ALLOW"
    echo ""
    echo "Allowed Rules:"
    echo "  • SSH port: ${ssh_port}/tcp"
    if [ "$tailscale_installed" = "true" ]; then
        echo "  • Tailscale VPN: 41641/udp"
    fi
    echo "  • Localhost: 127.0.0.1"
    echo ""
    echo "=========================================="
    echo ""
    log_warning "IMPORTANT SECURITY NOTES:"
    echo "  • All incoming traffic is now BLOCKED by default"
    echo "  • Only SSH (port ${ssh_port}) and Tailscale (if enabled) are allowed"
    echo "  • Make sure you have SSH keys configured before closing this session!"
    echo ""
}

# Interactive mode
interactive_mode() {
    echo ""
    echo "=========================================="
    echo "      ArmorClaw Firewall Setup"
    echo "=========================================="
    echo ""

    # Prompt for SSH port
    read -p "SSH port (default: 22): " ssh_port
    ssh_port=${ssh_port:-22}

    # Verify SSH port is numeric
    if ! [[ "$ssh_port" =~ ^[0-9]+$ ]] || [ "$ssh_port" -lt 1 ] || [ "$ssh_port" -gt 65535 ]; then
        log_error "Invalid SSH port: $ssh_port"
        exit 1
    fi

    echo ""
    log_info "This will configure UFW with the following settings:"
    echo "  • Default policy: DENY incoming, ALLOW outgoing"
    echo "  • Allow SSH on port: ${ssh_port}"
    if [ "$(check_tailscale)" = "true" ]; then
        echo "  • Allow Tailscale VPN (UDP 41641)"
    fi
    echo ""

    read -p "Continue? (y/N): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        log_info "Firewall setup cancelled"
        exit 0
    fi

    # Configure firewall
    configure_firewall "$ssh_port"

    # Show status
    show_status

    # Verify
    if verify_firewall; then
        print_summary "$ssh_port"

        log_warning "Before continuing, please test your SSH connection in a NEW terminal:"
        echo "  1. Open a new terminal window"
        echo "  2. SSH into this server: ssh user@$(hostname -I | awk '{print $1}') -p ${ssh_port}"
        echo "  3. If you can connect, type 'yes' below"
        echo ""

        read -p "SSH connection test successful? (y/N): " ssh_test
        if [[ ! "$ssh_test" =~ ^[Yy]$ ]]; then
            log_warning "SSH test failed. Disabling firewall for safety..."
            ufw --force reset
            ufw default allow incoming
            ufw default allow outgoing
            echo "y" | ufw enable
            log_info "Firewall has been disabled. Please fix SSH access and try again."
            exit 1
        fi

        log_success "Firewall setup complete!"
    else
        log_error "Firewall verification failed"
        exit 1
    fi
}

# Non-interactive mode (for automation)
non_interactive_mode() {
    local ssh_port="${1:-22}"

    configure_firewall "$ssh_port"
    show_status
    verify_firewall
    print_summary "$ssh_port"
}

# Main function
main() {
    check_root
    check_ufw

    if [ "$AUTO_CONFIRM" = "1" ] || [ "$1" = "--yes" ]; then
        non_interactive_mode "${2:-22}"
    else
        interactive_mode
    fi
}

# Run main function
main "$@"
