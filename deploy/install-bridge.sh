#!/bin/bash
# =============================================================================
# ArmorClaw Bridge Installation Script
# Purpose: Install and configure the ArmorClaw bridge on a Linux system
# Supported: Ubuntu 22.04/24.04, Debian 12
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Version and configuration
BRIDGE_VERSION="${ARMORCLAW_VERSION:-v1.0.0}"
INSTALL_DIR="/opt/armorclaw"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/armorclaw"
RUN_DIR="/run/armorclaw"
USER="armorclaw"
GROUP="armorclaw"

# =============================================================================
# Helper Functions
# =============================================================================

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

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_os() {
    if [[ ! -f /etc/os-release ]]; then
        log_error "Cannot detect OS"
        exit 1
    fi

    source /etc/os-release
    if [[ "$ID" != "ubuntu" ]] && [[ "$ID" != "debian" ]]; then
        log_warning "This script is designed for Ubuntu/Debian. Detected: $ID"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    log_success "OS detected: $PRETTY_NAME"
}

verify_checksum() {
    local file="$1"
    local expected_checksum="$2"

    if [[ ! -f "$file" ]]; then
        log_error "File not found: $file"
        return 1
    fi

    # Calculate SHA256 checksum
    local actual_checksum=$(sha256sum "$file" | awk '{print $1}')

    if [[ "$actual_checksum" == "$expected_checksum" ]]; then
        log_success "Checksum verified for $(basename "$file")"
        return 0
    else
        log_error "Checksum mismatch for $(basename "$file")"
        log_error "Expected: $expected_checksum"
        log_error "Actual:   $actual_checksum"
        return 1
    fi
}

# =============================================================================
# Installation Steps
# =============================================================================

create_user() {
    log_info "Creating ArmorClaw user..."

    if id "$USER" &>/dev/null; then
        log_warning "User $USER already exists"
    else
        useradd --system --user-group --shell /bin/false --home "$INSTALL_DIR" "$USER"
        log_success "Created user: $USER"
    fi
}

create_directories() {
    log_info "Creating directories..."

    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$RUN_DIR"
    mkdir -p "$INSTALL_DIR/containers"

    chown -R "$USER:$GROUP" "$INSTALL_DIR"
    chown -R "$USER:$GROUP" "$RUN_DIR"
    chmod 755 "$CONFIG_DIR"
    chmod 770 "$RUN_DIR"

    log_success "Directories created"
}

install_binary() {
    log_info "Installing bridge binary..."

    local binary_source=""

    # Check if binary is provided locally
    if [[ -f "./armorclaw-bridge" ]]; then
        binary_source="./armorclaw-bridge"
        log_info "Using local binary"
    elif [[ -f "./bin/bridge" ]]; then
        binary_source="./bin/bridge"
        log_info "Using local binary from bin/"
    else
        log_error "Bridge binary not found"
        log_error "Please provide armorclaw-bridge in the current directory"
        exit 1
    fi

    # Verify binary exists and is executable
    if [[ ! -x "$binary_source" ]]; then
        chmod +x "$binary_source"
    fi

    # Copy to installation directory
    cp "$binary_source" "$INSTALL_DIR/armorclaw-bridge"
    chmod 755 "$INSTALL_DIR/armorclaw-bridge"
    chown "$USER:$GROUP" "$INSTALL_DIR/armorclaw-bridge"

    # Create symlink in BIN_DIR
    ln -sf "$INSTALL_DIR/armorclaw-bridge" "$BIN_DIR/armorclaw-bridge"

    # Verify installation
    if "$INSTALL_DIR/armorclaw-bridge" --version &>/dev/null; then
        log_success "Bridge binary installed"
    else
        log_warning "Binary installed but version check failed"
    fi
}

install_container_image() {
    log_info "Checking for container image..."

    if [[ -f "./armorclaw-agent-v1.tar.gz" ]]; then
        log_info "Loading container image..."

        if command -v docker &>/dev/null; then
            docker load -i "./armorclaw-agent-v1.tar.gz" 2>/dev/null || {
                log_warning "Failed to load container image (Docker may not be running)"
            }
        else
            log_warning "Docker not found, skipping container image load"
        fi
    else
        log_info "No container image found (this is OK if using remote registry)"
    fi
}

install_config() {
    log_info "Installing configuration..."

    if [[ -f "./config.example.toml" ]]; then
        cp "./config.example.toml" "$CONFIG_DIR/config.toml"
    elif [[ -f "./bridge/config.example.toml" ]]; then
        cp "./bridge/config.example.toml" "$CONFIG_DIR/config.toml"
    else
        # Create minimal default config
        cat > "$CONFIG_DIR/config.toml" <<'EOF'
[server]
socket_path = "/run/armorclaw/bridge.sock"
pid_file = "/run/armorclaw/bridge.pid"
daemonize = false

[keystore]
db_path = "/etc/armorclaw/keystore.db"
master_key = ""

[matrix]
enabled = false
homeserver_url = "https://matrix.example.com"
username = "bridge-bot"
password = "change-me"
device_id = "armorclaw-bridge"

[logging]
level = "info"
format = "text"
output = "stdout"
EOF
    fi

    chown "$USER:$GROUP" "$CONFIG_DIR/config.toml"
    chmod 640 "$CONFIG_DIR/config.toml"

    log_success "Configuration installed at $CONFIG_DIR/config.toml"
}

install_systemd_service() {
    log_info "Installing systemd service..."

    cat > "/etc/systemd/system/armorclaw-bridge.service" <<EOF
[Unit]
Description=ArmorClaw Bridge - Secure AI Agent Container Manager
Documentation=https://github.com/armorclaw/armorclaw
After=network-online.target docker.socket
Wants=network-online.target docker.socket

[Service]
Type=forking
User=$USER
Group=$GROUP

# Configuration
Environment="ARMORCLAW_CONFIG=$CONFIG_DIR/config.toml"

# Execution
ExecStart=$INSTALL_DIR/armorclaw-bridge --daemonize --config $CONFIG_DIR/config.toml
ExecStop=$INSTALL_DIR/armorclaw-bridge --stop
ExecReload=/bin/kill -HUP \$MAINPID

# Resource limits
MemoryLimit=512M
MemoryMax=1G

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$RUN_DIR $CONFIG_DIR /var/lib/armorclaw

# Process management
PIDFile=$RUN_DIR/bridge.pid
Restart=on-failure
RestartSec=5s
TimeoutStartSec=30s
TimeoutStopSec=30s

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable armorclaw-bridge.service

    log_success "Systemd service installed and enabled"
}

create_uninstall_script() {
    log_info "Creating uninstall script..."

    cat > "$INSTALL_DIR/uninstall.sh" <<'EOF'
#!/bin/bash
# ArmorClaw Bridge Uninstall Script

set -euo pipefail

RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    exit 1
fi

echo -e "${YELLOW}This will remove ArmorClaw Bridge from your system.${NC}"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Uninstall cancelled"
    exit 0
fi

# Stop and disable service
systemctl stop armorclaw-bridge.service || true
systemctl disable armorclaw-bridge.service || true

# Remove service file
rm -f /etc/systemd/system/armorclaw-bridge.service
systemctl daemon-reload

# Remove symlink
rm -f /usr/local/bin/armorclaw-bridge

# Preserve or remove config
read -p "Remove configuration files in /etc/armorclaw? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf /etc/armorclaw
fi

# Preserve or remove keystore
read -p "Remove keystore database? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f /etc/armorclaw/keystore.db
    rm -f /etc/armorclaw/keystore.db.salt
fi

# Remove installation directory
rm -rf /opt/armorclaw

# Remove user (only if no processes remain)
if ! pgrep -u armorclaw &>/dev/null; then
    userdel armorclaw || true
fi

echo "ArmorClaw Bridge removed successfully"
EOF

    chmod +x "$INSTALL_DIR/uninstall.sh"
    chown "$USER:$GROUP" "$INSTALL_DIR/uninstall.sh"

    log_success "Uninstall script created at $INSTALL_DIR/uninstall.sh"
}

print_post_install() {
    cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║           ArmorClaw Bridge Installation Complete                    ║
╚══════════════════════════════════════════════════════════════════════╝${NC}

${BLUE}Installation Details:${NC}
  • Binary:       $INSTALL_DIR/armorclaw-bridge
  • Config:       $CONFIG_DIR/config.toml
  • Socket:       $RUN_DIR/bridge.sock
  • User:         $USER

${BLUE}Service Commands:${NC}
  • Start:        sudo systemctl start armorclaw-bridge
  • Stop:         sudo systemctl stop armorclaw-bridge
  • Restart:      sudo systemctl restart armorclaw-bridge
  • Status:       sudo systemctl status armorclaw-bridge
  • Logs:         sudo journalctl -u armorclaw-bridge -f

${BLUE}Next Steps:${NC}
  1. Edit configuration: sudo nano $CONFIG_DIR/config.toml
  2. Start the service: sudo systemctl start armorclaw-bridge
  3. Check status: sudo systemctl status armorclaw-bridge
  4. View logs: sudo journalctl -u armorclaw-bridge -n 50

${YELLOW}Configuration Required:${NC}
  • Matrix settings: Enable and configure in config.toml if needed
  • Docker: Ensure Docker is installed and running
  • Container: Pull or build armorclaw/agent:v1 container image

${YELLOW}Important:${NC}
  • The keystore database will be created on first start
  • Database is encrypted with hardware-derived master key
  • Keep backups of the database and salt file

${GREEN}To uninstall: $INSTALL_DIR/uninstall.sh${NC}

EOF
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║           ArmorClaw Bridge Installation Script                 ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo "Version: $BRIDGE_VERSION"
    echo ""

    # Prerequisites check
    check_root
    check_os

    # Confirm installation
    log_warning "This will install ArmorClaw Bridge on your system"
    echo ""
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Installation cancelled"
        exit 0
    fi

    echo ""
    log_info "Starting installation..."
    echo ""

    # Execute installation steps
    create_user
    create_directories
    install_binary
    install_container_image
    install_config
    install_systemd_service
    create_uninstall_script

    echo ""
    print_post_install
}

# Run main function
main "$@"
