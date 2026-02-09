#!/bin/bash
# ArmorClaw VPS Deployment Script
# Usage: Upload this script + armorclaw-deploy.tar.gz to VPS, then run this script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Global variables
TARBALL=""
DEPLOY_DIR="/opt/armorclaw"
ERRORS_OCCURRED=0

print_header() {
    echo -e "${CYAN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║     ArmorClaw VPS Deployment Script                  ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    local step="$1"
    local total="$2"
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Step $step of $total: ${BOLD}$3${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ ERROR: $1${NC}" >&2
    ERRORS_OCCURRED=1
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

prompt() {
    local prompt_text="$1"
    local default_value="$2"
    echo -ne "${CYAN}$prompt_text${NC}"
    [ -n "$default_value" ] && echo -ne " ${YELLOW}[$default_value]${NC}"
    echo -ne ": "
    read -r response
    echo "${response:-$default_value}"
}

prompt_yes_no() {
    local prompt_text="$1"
    local default="${2:-n}"
    while true; do
        if [ "$default" = "y" ]; then
            echo -ne "${CYAN}$prompt_text [Y/n]: ${NC}"
        else
            echo -ne "${CYAN}$prompt_text [y/N]: ${NC}"
        fi
        read -r response
        response=${response:-$default}
        case "$response" in
            [Yy]|[Yy][Ee][Ss]) return 0 ;;
            [Nn]|[Nn][Oo]) return 1 ;;
        esac
        echo -e "${YELLOW}Please answer yes or no.${NC}"
    done
}

check_disk_space() {
    local required_mb=500
    local available_mb=$(df -m / | tail -1 | awk '{print $4}')
    if [ "$available_mb" -lt "$required_mb" ]; then
        print_error "Insufficient disk space. Need ${required_mb}MB, only ${available_mb}MB available."
        return 1
    fi
    print_success "Disk space check passed (${available_mb}MB available)"
    return 0
}

check_memory() {
    local required_mb=1024
    local available_mb=$(free -m | awk '/^Mem:/{print $2}')
    if [ "$available_mb" -lt "$required_mb" ]; then
        print_warning "Low memory detected. ArmorClaw recommends ${required_mb}MB RAM, only ${available_mb}MB available."
        print_warning "Deployment may fail or run slowly."
        if ! prompt_yes_no "Continue anyway?" "n"; then
            return 1
        fi
    fi
    print_success "Memory check passed (${available_mb}MB available)"
    return 0
}

check_ports() {
    local ports=(80 443 8448 6167)
    local ports_in_use=()

    for port in "${ports[@]}"; do
        if ss -tuln | grep -q ":$port "; then
            ports_in_use+=("$port")
        fi
    done

    if [ ${#ports_in_use[@]} -gt 0 ]; then
        print_warning "The following ports are already in use: ${ports_in_use[*]}"
        print_warning "This may cause conflicts with ArmorClaw services."
        if ! prompt_yes_no "Continue anyway?" "n"; then
            return 1
        fi
    fi
    print_success "Port availability check passed"
    return 0
}

verify_tarball() {
    local tarball="$1"

    if ! tar -tzf "$tarball" > /dev/null 2>&1; then
        print_error "Tarball is corrupted or invalid"
        return 1
    fi

    print_success "Tarball verification passed"
    return 0
}

install_docker() {
    print_info "Installing Docker..."

    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        print_info "Installing curl..."
        apt-get update -qq
        apt-get install -y curl
    fi

    # Download and run Docker install script
    if curl -fsSL https://get.docker.com -o /tmp/get-docker.sh; then
        sh /tmp/get-docker.sh
        systemctl enable docker
        systemctl start docker

        # Verify installation
        if command -v docker &> /dev/null; then
            print_success "Docker installed: $(docker --version)"
            return 0
        else
            print_error "Docker installation failed"
            return 1
        fi
    else
        print_error "Failed to download Docker install script"
        return 1
    fi
}

get_hostname_f() {
    # Try multiple methods to get hostname
    local hostname=""
    hostname=$(hostname -f 2>/dev/null) || hostname=$(hostname) || hostname="localhost"
    echo "$hostname"
}

# Main deployment flow
main() {
    print_header

    # Step 1: Pre-flight checks
    print_step 1 7 "Pre-flight Checks"

    # Check if running as root
    if [ "$EUID" -ne 0 ]; then
        print_error "This script must be run as root (use sudo)"
        echo -e "\n${YELLOW}Run: sudo bash $0${NC}\n"
        exit 1
    fi
    print_success "Running as root"

    # Check system resources
    check_disk_space || exit 1
    check_memory || exit 1
    check_ports || exit 1

    # Step 2: Locate tarball
    print_step 2 7 "Locate Tarball"
    TARBALL=""
    if [ -f "./armorclaw-deploy.tar.gz" ]; then
        TARBALL="./armorclaw-deploy.tar.gz"
    elif [ -f "/tmp/armorclaw-deploy.tar.gz" ]; then
        TARBALL="/tmp/armorclaw-deploy.tar.gz"
    elif [ -f "$HOME/armorclaw-deploy.tar.gz" ]; then
        TARBALL="$HOME/armorclaw-deploy.tar.gz"
    else
        echo -ne "${CYAN}Enter path to armorclaw-deploy.tar.gz: ${NC}"
        read -r TARBALL
    fi

    if [ ! -f "$TARBALL" ]; then
        print_error "Tarball not found at: $TARBALL"
        echo -e "\n${YELLOW}Download ArmorClaw and create tarball:${NC}"
        echo "  tar -czf armorclaw-deploy.tar.gz --exclude='.git' --exclude='bridge/build' ."
        exit 1
    fi
    print_success "Found tarball: $TARBALL"

    # Verify tarball integrity
    verify_tarball "$TARBALL" || exit 1

    # Step 3: Extract files
    print_step 3 7 "Extract Files"
    DEPLOY_DIR=$(prompt "Deployment directory" "/opt/armorclaw")

    # Check if directory already exists
    if [ -d "$DEPLOY_DIR" ] && [ "$(ls -A $DEPLOY_DIR 2>/dev/null)" ]; then
        print_warning "Directory $DEPLOY_DIR already exists and is not empty"
        if ! prompt_yes_no "Overwrite existing files?" "n"; then
            print_info "Please specify a different directory or clean up the existing one"
            exit 1
        fi
    fi

    # Create and enter directory
    mkdir -p "$DEPLOY_DIR"
    cd "$DEPLOY_DIR"
    print_success "Created directory: $DEPLOY_DIR"

    # Extract tarball
    echo "Extracting files..."
    if tar -xzf "$TARBALL" 2>/dev/null; then
        print_success "Files extracted successfully"
    else
        print_error "Failed to extract tarball"
        exit 1
    fi

    # Verify extraction
    if [ ! -f "docker-compose-stack.yml" ]; then
        print_error "Extraction failed - docker-compose-stack.yml not found"
        exit 1
    fi

    # Fix line endings and permissions
    echo "Fixing script line endings..."
    find deploy -name "*.sh" -type f -exec sed -i 's/\r$//' {} \; 2>/dev/null || true
    chmod +x deploy/*.sh 2>/dev/null || true
    print_success "Scripts fixed and made executable"

    # Step 4: Install Docker if needed
    print_step 4 7 "Check Docker"

    if ! command -v docker &> /dev/null; then
        if prompt_yes_no "Docker is not installed. Install now?"; then
            install_docker || exit 1
        else
            print_error "Docker is required for ArmorClaw"
            exit 1
        fi
    else
        print_success "Docker already installed: $(docker --version)"
    fi

    # Check Docker Compose
    if ! docker compose version &> /dev/null; then
        print_warning "Docker Compose plugin not found"
        if prompt_yes_no "Install Docker Compose?" "y"; then
            apt-get update -qq
            apt-get install -y docker-compose-plugin
            print_success "Docker Compose installed"
        else
            print_error "Docker Compose is required"
            exit 1
        fi
    else
        print_success "Docker Compose available: $(docker compose version)"
    fi

    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        print_warning "Docker daemon is not running"
        systemctl start docker
        sleep 2
        if docker info &> /dev/null; then
            print_success "Docker daemon started"
        else
            print_error "Failed to start Docker daemon"
            exit 1
        fi
    fi

    # Step 5: Get deployment info
    print_step 5 7 "Configuration"
    echo ""
    echo "Choose deployment method:"
    echo "  1) Docker Compose (quick start) ⭐"
    echo "  2) Setup Wizard (interactive)"
    echo ""

    METHOD=$(prompt "Choose method [1-2]" "1")

    case "$METHOD" in
        1)
            echo ""
            echo -e "${BOLD}Quick Docker Compose Setup${NC}"
            echo ""

            # Get configuration
            local hostname_f=$(get_hostname_f)
            MATRIX_DOMAIN=$(prompt "Matrix domain" "$hostname_f")
            MATRIX_ADMIN_USER=$(prompt "Matrix admin user" "admin")
            MATRIX_ADMIN_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=[:space:]')
            ROOM_NAME=$(prompt "Room name" "ArmorClaw Agents")
            ROOM_ALIAS=$(prompt "Room alias" "agents")

            # Create .env file
            cat > .env <<EOF
MATRIX_DOMAIN=$MATRIX_DOMAIN
MATRIX_ADMIN_USER=$MATRIX_ADMIN_USER
MATRIX_ADMIN_PASSWORD=$MATRIX_ADMIN_PASSWORD
ROOM_NAME=$ROOM_NAME
ROOM_ALIAS=$ROOM_ALIAS
EOF

            chmod 600 .env

            echo ""
            echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
            echo -e "${GREEN}Deployment Configuration${NC}"
            echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
            echo ""
            echo -e "${CYAN}Homeserver:${NC}  $MATRIX_DOMAIN"
            echo -e "${CYAN}Username:${NC}    $MATRIX_ADMIN_USER"
            echo -e "${CYAN}Password:${NC}    ${BOLD}${YELLOW}$MATRIX_ADMIN_PASSWORD${NC}"
            echo ""
            echo -e "${RED}⚠ SAVE THIS PASSWORD - YOU WILL NEED IT TO LOGIN!${NC}"
            echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
            echo ""

            # Deploy
            if prompt_yes_no "Start deployment now" "y"; then
                print_step 6 7 "Deploying ArmorClaw"

                echo "Pulling Docker images..."
                if docker compose -f docker-compose-stack.yml pull 2>/dev/null; then
                    print_success "Images pulled"
                else
                    print_warning "Some images failed to pull (may build from source)"
                fi

                echo "Starting containers..."
                if docker compose -f docker-compose-stack.yml up -d; then
                    print_success "Containers started"

                    # Wait for services to be healthy
                    echo ""
                    print_info "Waiting for services to start (30 seconds)..."
                    sleep 30

                    # Check status
                    echo ""
                    echo -e "${BOLD}Container Status:${NC}"
                    docker compose -f docker-compose-stack.yml ps

                    echo ""
                    print_success "Deployment complete!"
                else
                    print_error "Deployment failed"
                    echo ""
                    echo "Check logs with:"
                    echo "  docker compose -f docker-compose-stack.yml logs"
                    exit 1
                fi
            else
                print_info "Deployment cancelled. To deploy manually:"
                echo "  cd $DEPLOY_DIR"
                echo "  docker compose -f docker-compose-stack.yml up -d"
            fi
            ;;

        2)
            echo ""
            echo -e "${BOLD}Setup Wizard${NC}"
            echo ""

            # Run setup wizard
            if [ -f "deploy/setup-wizard.sh" ]; then
                print_step 6 7 "Running Setup Wizard"
                echo ""

                # Check if dialog is needed for wizard
                if ! command -v dialog &> /dev/null; then
                    print_info "Installing dialog package..."
                    apt-get install -y dialog > /dev/null 2>&1
                fi

                bash deploy/setup-wizard.sh
            else
                print_error "Setup wizard not found"
                exit 1
            fi
            ;;

        *)
            print_error "Invalid choice. Please enter 1 or 2."
            exit 1
            ;;
    esac

    # Step 7: Host Hardening
    print_step 7 8 "Host Hardening"

    echo ""
    echo "ArmorClaw includes optional host hardening for production deployments."
    echo ""
    echo "This includes:"
    echo "  • Firewall configuration (UFW - deny all incoming, allow SSH)"
    echo "  • SSH hardening (disable passwords, require keys)"
    echo ""

    if prompt_yes_no "Configure firewall and SSH hardening?" "y"; then
        # Run firewall setup
        echo ""
        if [ -f "deploy/setup-firewall.sh" ]; then
            print_info "Running firewall setup..."
            AUTO_CONFIRM=1 bash deploy/setup-firewall.sh --yes 22
            print_success "Firewall configured"
        else
            print_warning "Firewall setup script not found"
        fi

        # Run SSH hardening
        echo ""
        if [ -f "deploy/harden-ssh.sh" ]; then
            print_info "Running SSH hardening..."
            # Check for SSH keys first
            if [ -d "/root/.ssh" ] && ls /root/.ssh/id_* >/dev/null 2>&1; then
                AUTO_CONFIRM=1 bash deploy/harden-ssh.sh --yes
                print_success "SSH hardened"
            else
                print_warning "No SSH keys found. Skipping SSH hardening."
                print_warning "You can run this later: sudo bash deploy/harden-ssh.sh"
            fi
        else
            print_warning "SSH hardening script not found"
        fi

        echo ""
        print_success "Host hardening complete"
    else
        print_info "Skipping host hardening"
        echo ""
        print_warning "You can run hardening later:"
        echo "  sudo bash $DEPLOY_DIR/deploy/setup-firewall.sh"
        echo "  sudo bash $DEPLOY_DIR/deploy/harden-ssh.sh"
    fi

    # Step 8: Completion
    print_step 8 8 "Deployment Complete"

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║         ArmorClaw Deployment Complete! ✓             ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""

    if [ "$METHOD" = "1" ]; then
        echo "Deployment directory: ${BOLD}$DEPLOY_DIR${NC}"
        echo ""
        echo "Next steps:"
        echo ""
        echo "  1. ${CYAN}Save the admin password shown above${NC}"
        echo "  2. ${CYAN}Wait 1-2 minutes for services to fully start${NC}"
        echo "  3. ${CYAN}Connect via Element X:${NC}"
        echo "     - Homeserver: ${YELLOW}http://$MATRIX_DOMAIN:8448${NC}"
        echo "                or: ${YELLOW}https://$MATRIX_DOMAIN${NC}"
        echo "     - Username: ${YELLOW}$MATRIX_ADMIN_USER${NC}"
        echo "     - Password: ${YELLOW}$MATRIX_ADMIN_PASSWORD${NC}"
        echo ""
        echo "Management commands:"
        echo "  cd $DEPLOY_DIR"
        echo "  docker compose ps          # Check status"
        echo "  docker compose logs -f      # View logs"
        echo "  docker compose down         # Stop services"
        echo ""
    else
        echo "Deployment directory: ${BOLD}$DEPLOY_DIR${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Check service status"
        echo "  2. Connect via Element X (see docs/guides/element-x-quickstart.md)"
        echo ""
    fi

    # Troubleshooting hint
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Troubleshooting${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "If services don't start:"
    echo "  docker compose -f $DEPLOY_DIR/docker-compose-stack.yml logs"
    echo ""
    echo "For common issues:"
    echo "  https://github.com/armorclaw/armorclaw/wiki/Troubleshooting"
    echo ""

    if [ "$ERRORS_OCCURRED" -eq 0 ]; then
        echo -e "${GREEN}Deployment completed without errors!${NC}"
    else
        echo -e "${YELLOW}Deployment completed with some warnings.${NC}"
    fi

    echo ""
}

# Run main function
main "$@"
