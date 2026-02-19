#!/bin/bash
# ArmorClaw Settings Backup Utility
# Creates a portable .zip backup of agent settings
# Version: 1.0.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Default paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
BACKUP_DIR="${1:-.}"

#=============================================================================
# Helper Functions
#=============================================================================

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ ERROR: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

#=============================================================================
# Backup Function
#=============================================================================

create_backup() {
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local backup_name="armorclaw-backup-${timestamp}"
    local temp_dir
    temp_dir=$(mktemp -d)

    print_info "Creating ArmorClaw settings backup..."
    echo ""

    # Create backup structure
    mkdir -p "$temp_dir/armorclaw-backup/config"
    mkdir -p "$temp_dir/armorclaw-backup/keystore"
    mkdir -p "$temp_dir/armorclaw-backup/agent-configs"
    mkdir -p "$temp_dir/armorclaw-backup/matrix"

    local items_backed_up=0

    # Backup configuration
    if [ -f "$CONFIG_DIR/config.toml" ]; then
        cp "$CONFIG_DIR/config.toml" "$temp_dir/armorclaw-backup/config/"
        print_success "Configuration file backed up"
        ((items_backed_up++))
    else
        print_warning "Configuration file not found at $CONFIG_DIR/config.toml"
    fi

    # Backup keystore structure (not the encrypted data - that's hardware-bound)
    # We backup the schema/structure info for reference
    if [ -f "$DATA_DIR/keystore.db" ]; then
        # Create a metadata file about the keystore
        cat > "$temp_dir/armorclaw-backup/keystore/keystore-info.json" <<EOF
{
    "note": "Keystore database is hardware-bound and cannot be transferred",
    "original_path": "$DATA_DIR/keystore.db",
    "keys_must_be_re_added": true,
    "backup_date": "$(date -Iseconds)"
}
EOF
        print_warning "Keystore metadata backed up (keys cannot be transferred - hardware-bound)"
        ((items_backed_up++))
    fi

    # Backup agent configurations
    if [ -d "$DATA_DIR/agent-configs" ] && [ "$(ls -A $DATA_DIR/agent-configs 2>/dev/null)" ]; then
        cp -r "$DATA_DIR/agent-configs/"* "$temp_dir/armorclaw-backup/agent-configs/" 2>/dev/null || true
        local config_count=$(find "$temp_dir/armorclaw-backup/agent-configs" -name "*.json" | wc -l)
        print_success "Agent configurations backed up ($config_count files)"
        ((items_backed_up++))
    else
        print_warning "No agent configurations found"
    fi

    # Backup Matrix session data
    if [ -d "$DATA_DIR/matrix" ] && [ -f "$DATA_DIR/matrix/session.json" ]; then
        cp "$DATA_DIR/matrix/session.json" "$temp_dir/armorclaw-backup/matrix/" 2>/dev/null || true
        print_success "Matrix session data backed up"
        ((items_backed_up++))
    else
        print_warning "No Matrix session data found"
    fi

    # Check if we have anything to backup
    if [ $items_backed_up -eq 0 ]; then
        print_error "No settings found to backup"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Create manifest
    cat > "$temp_dir/armorclaw-backup/manifest.json" <<EOF
{
    "version": "1.0.0",
    "format": "armorclaw-backup-v1",
    "created_at": "$(date -Iseconds)",
    "hostname": "$(hostname)",
    "items": {
        "config": $([ -f "$temp_dir/armorclaw-backup/config/config.toml" ] && echo "true" || echo "false"),
        "agent_configs": $([ -d "$temp_dir/armorclaw-backup/agent-configs" ] && [ "$(ls -A $temp_dir/armorclaw-backup/agent-configs 2>/dev/null)" ] && echo "true" || echo "false"),
        "matrix_session": $([ -f "$temp_dir/armorclaw-backup/matrix/session.json" ] && echo "true" || echo "false"),
        "keystore_metadata": true
    },
    "notes": [
        "API keys are hardware-bound and cannot be transferred",
        "Re-add API keys after importing on new system",
        "Matrix session may need re-authentication on new system"
    ]
}
EOF

    # Create README
    cat > "$temp_dir/armorclaw-backup/README.txt" <<'EOF'
ArmorClaw Settings Backup
=========================

This backup contains your ArmorClaw configuration and settings.

Contents:
  - config/config.toml      : Bridge configuration
  - agent-configs/*.json    : Agent configuration profiles
  - matrix/session.json     : Matrix session data
  - keystore/keystore-info.json : Keystore metadata (keys not included)

IMPORTANT:
  - API keys are encrypted with hardware-bound keys
  - You MUST re-add API keys after importing on a new system
  - Matrix session may require re-authentication

To restore:
  1. Copy this .zip file to the new system
  2. Run: sudo ./deploy/setup-wizard.sh
  3. Choose "Import settings from backup" when prompted

Generated by: ArmorClaw backup-settings.sh
EOF

    # Create the zip file
    local zip_path="$BACKUP_DIR/${backup_name}.zip"

    if ! command -v zip &> /dev/null; then
        print_error "zip command not found. Install with: sudo apt-get install zip"
        rm -rf "$temp_dir"
        exit 1
    fi

    (cd "$temp_dir" && zip -r "$zip_path" armorclaw-backup/)

    # Cleanup
    rm -rf "$temp_dir"

    echo ""
    print_success "Backup created: $zip_path"
    echo ""
    echo -e "${BOLD}Backup Contents:${NC}"
    echo "  • Bridge configuration"
    echo "  • Agent configuration profiles"
    echo "  • Matrix session data"
    echo ""
    echo -e "${YELLOW}Note:${NC} API keys are NOT included (hardware-bound encryption)"
    echo ""
    echo -e "${BOLD}To restore this backup:${NC}"
    echo "  1. Copy to target system"
    echo "  2. Run: sudo ./deploy/setup-wizard.sh"
    echo "  3. Select 'Import settings from backup'"
    echo ""
}

#=============================================================================
# Usage
#=============================================================================

usage() {
    echo "ArmorClaw Settings Backup Utility"
    echo ""
    echo "Usage: $0 [OUTPUT_DIRECTORY]"
    echo ""
    echo "Creates a .zip backup of ArmorClaw settings that can be imported"
    echo "during setup on another system."
    echo ""
    echo "Arguments:"
    echo "  OUTPUT_DIRECTORY  Directory to save backup (default: current directory)"
    echo ""
    echo "What is backed up:"
    echo "  ✓ Bridge configuration (config.toml)"
    echo "  ✓ Agent configuration profiles"
    echo "  ✓ Matrix session data"
    echo ""
    echo "What is NOT backed up:"
    echo "  ✗ API keys (hardware-bound encryption)"
    echo ""
    echo "Examples:"
    echo "  $0                    # Save to current directory"
    echo "  $0 /home/user/backups # Save to specific directory"
    echo ""
}

#=============================================================================
# Main
#=============================================================================

# Check for help flag
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    usage
    exit 0
fi

# Check if running as root (needed to read config files)
if [ "$EUID" -ne 0 ]; then
    print_warning "Some files may not be readable without root access"
    print_info "Run with sudo for complete backup"
    echo ""
fi

# Create output directory if needed
if [ ! -d "$BACKUP_DIR" ]; then
    mkdir -p "$BACKUP_DIR"
fi

create_backup
