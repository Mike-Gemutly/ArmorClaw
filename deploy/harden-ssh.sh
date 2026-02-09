#!/usr/bin/bash
# harden-ssh.sh - Harden SSH configuration for ArmorClaw
# This script disables password authentication and requires SSH keys

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

# Backup SSH config
backup_config() {
    local ssh_config="/etc/ssh/sshd_config"
    local backup_dir="/var/backups/armorclaw"

    log_info "Backing up SSH configuration..."

    # Create backup directory
    mkdir -p "$backup_dir"

    # Create timestamped backup
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${backup_dir}/sshd_config_${timestamp}"

    cp "$ssh_config" "$backup_file"
    log_success "Backup created: $backup_file"

    # Create symlink to latest backup
    ln -sf "$backup_file" "${backup_dir}/sshd_config_latest"
}

# Check if user has SSH keys
check_ssh_keys() {
    local current_user="${SUDO_USER:-$USER}"

    log_info "Checking for SSH keys..."

    # Check for user's SSH keys
    if [ -n "$current_user" ] && [ "$current_user" != "root" ]; then
        local user_home="/home/$current_user"
        if [ ! -d "$user_home" ]; then
            user_home="/root"
        fi

        local ssh_dir="$user_home/.ssh"

        if [ -d "$ssh_dir" ]; then
            local key_count=$(find "$ssh_dir" -name "id_*" -not -name "*.pub" 2>/dev/null | wc -l)

            if [ "$key_count" -gt 0 ]; then
                log_success "Found $key_count SSH key(s) for user $current_user"
                return 0
            fi
        fi
    fi

    # Check root's SSH keys
    if [ -d "/root/.ssh" ]; then
        local key_count=$(find "/root/.ssh" -name "id_*" -not -name "*.pub" 2>/dev/null | wc -l)
        if [ "$key_count" -gt 0 ]; then
            log_success "Found $key_count SSH key(s) for root"
            return 0
        fi
    fi

    log_warning "No SSH keys found for current user"
    return 1
}

# Harden SSH configuration
harden_ssh() {
    local ssh_config="/etc/ssh/sshd_config"

    log_info "Hardening SSH configuration..."

    # Create backup first
    backup_config

    # Modify SSH config using sed
    # Disable root login
    sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin no/' "$ssh_config"

    # Disable password authentication
    sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' "$ssh_config"

    # Disable challenge-response authentication
    sed -i 's/^#\?ChallengeResponseAuthentication.*/ChallengeResponseAuthentication no/' "$ssh_config"

    # Enable PubkeyAuthentication (ensure it's set to yes)
    sed -i 's/^#\?PubkeyAuthentication.*/PubkeyAuthentication yes/' "$ssh_config"

    # Set stricter authentication methods
    sed -i 's/^#\?AuthenticationMethods.*/AuthenticationMethods publickey/' "$ssh_config"

    # Disable PAM for SSH (optional, adds security)
    sed -i 's/^#\?UsePAM.*/UsePAM no/' "$ssh_config"

    log_success "SSH configuration hardened"
}

# Show current SSH configuration
show_config() {
    local ssh_config="/etc/ssh/sshd_config"

    echo ""
    log_info "Current SSH security settings:"
    echo ""

    echo "PermitRootLogin: $(grep -E '^#?PermitRootLogin' "$ssh_config" | tail -1 | cut -d' ' -f2)"
    echo "PasswordAuthentication: $(grep -E '^#?PasswordAuthentication' "$ssh_config" | tail -1 | cut -d' ' -f2)"
    echo "PubkeyAuthentication: $(grep -E '^#?PubkeyAuthentication' "$ssh_config" | tail -1 | cut -d' ' -f2)"
    echo "ChallengeResponseAuthentication: $(grep -E '^#?ChallengeResponseAuthentication' "$ssh_config" | tail -1 | cut -d' ' -f2)"
    echo ""
}

# Test SSH configuration
test_config() {
    log_info "Testing SSH configuration syntax..."

    if sshd -t 2>&1 | grep -q "Could not load host key"; then
        log_warning "Host keys not found. Generating..."
        ssh-keygen -A
    fi

    local test_output=$(sshd -t 2>&1)
    if [ -n "$test_output" ]; then
        log_error "SSH configuration test failed:"
        echo "$test_output"
        return 1
    fi

    log_success "SSH configuration test passed"
    return 0
}

# Restart SSH service
restart_ssh() {
    log_info "Restarting SSH service..."

    # Detect service management system
    if command -v systemctl &> /dev/null; then
        systemctl restart sshd || systemctl restart ssh
    elif command -v service &> /dev/null; then
        service sshd restart || service ssh restart
    else
        log_error "Could not detect service management system"
        return 1
    fi

    log_success "SSH service restarted"
}

# Restore from backup
restore_backup() {
    local backup_dir="/var/backups/armorclaw"
    local backup_file="${backup_dir}/sshd_config_latest"
    local ssh_config="/etc/ssh/sshd_config"

    if [ ! -f "$backup_file" ]; then
        log_error "No backup found at $backup_file"
        return 1
    fi

    log_info "Restoring from backup: $backup_file"

    cp "$backup_file" "$ssh_config"

    log_success "Configuration restored. Restarting SSH..."
    restart_ssh

    log_success "SSH configuration has been restored to original state"
}

# Print security summary
print_summary() {
    echo ""
    echo "=========================================="
    echo "        SSH Hardening Summary"
    echo "=========================================="
    echo ""
    echo "Changes Applied:"
    echo "  ✓ Root login: DISABLED"
    echo "  ✓ Password authentication: DISABLED"
    echo "  ✓ Challenge-response: DISABLED"
    echo "  ✓ Public key authentication: ENABLED"
    echo ""
    echo "=========================================="
    echo ""
    log_warning "CRITICAL WARNINGS:"
    echo "  • Password authentication is now DISABLED"
    echo "  • You MUST have SSH keys configured to login"
    echo "  • Original config backed up to: /var/backups/armorclaw/"
    echo ""
    echo "Before closing this session:"
    echo "  1. Open a NEW terminal window"
    echo "  2. Test SSH login: ssh $USER@$(hostname -I | awk '{print $1}')"
    echo "  3. Verify you can login with your SSH keys"
    echo ""
    echo "If you cannot login:"
    echo "  • Access via console/VNC to restore backup"
    echo "  • Run: $0 --restore"
    echo ""
}

# Interactive mode
interactive_mode() {
    echo ""
    echo "=========================================="
    echo "       ArmorClaw SSH Hardening"
    echo "=========================================="
    echo ""

    # Check for existing SSH keys
    if ! check_ssh_keys; then
        echo ""
        log_warning "WARNING: No SSH keys found!"
        echo ""
        echo "SSH hardening requires SSH key authentication."
        echo "Without SSH keys, you may be locked out of your server!"
        echo ""
        echo "To generate SSH keys, run:"
        echo "  ssh-keygen -t ed25519 -a 100"
        echo ""
        echo "Then copy the key to authorized_keys:"
        echo "  cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys"
        echo ""

        read -p "Continue anyway? (NOT RECOMMENDED) (y/N): " continue_anyway
        if [[ ! "$continue_anyway" =~ ^[Yy]$ ]]; then
            log_info "SSH hardening cancelled"
            exit 0
        fi
    fi

    # Show current config
    show_config

    echo "This script will make the following changes:"
    echo "  • Disable root login (PermitRootLogin no)"
    echo "  • Disable password authentication (PasswordAuthentication no)"
    echo "  • Disable challenge-response authentication"
    echo "  • Enable public key authentication"
    echo ""

    read -p "Continue with SSH hardening? (y/N): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        log_info "SSH hardening cancelled"
        exit 0
    fi

    # Harden SSH
    if harden_ssh; then
        # Test config
        if ! test_config; then
            log_error "SSH configuration test failed. Restoring backup..."
            restore_backup
            exit 1
        fi

        # Restart SSH
        restart_ssh

        print_summary

        # Verify SSH access
        log_warning "IMPORTANT: Test your SSH access in a NEW terminal now!"
        echo ""
        read -p "SSH access test successful? (y/N): " ssh_test
        if [[ ! "$ssh_test" =~ ^[Yy]$ ]]; then
            log_warning "SSH test failed. Restoring original configuration..."
            restore_backup
            log_info "Original configuration has been restored."
            exit 1
        fi

        log_success "SSH hardening complete!"
    else
        log_error "SSH hardening failed"
        exit 1
    fi
}

# Non-interactive mode
non_interactive_mode() {
    if [ "$1" = "--restore" ]; then
        restore_backup
        exit 0
    fi

    if ! check_ssh_keys; then
        log_warning "No SSH keys found. Skipping SSH key check due to --yes flag."
    fi

    harden_ssh

    if ! test_config; then
        log_error "SSH configuration test failed. Restoring backup..."
        restore_backup
        exit 1
    fi

    restart_ssh
    show_config
    print_summary
}

# Main function
main() {
    check_root

    if [ "$AUTO_CONFIRM" = "1" ] || [ "$1" = "--yes" ]; then
        non_interactive_mode "$@"
    elif [ "$1" = "--restore" ]; then
        restore_backup
    else
        interactive_mode
    fi
}

# Run main function
main "$@"
