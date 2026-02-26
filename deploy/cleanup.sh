#!/bin/bash
# =============================================================================
# ArmorClaw Cleanup Script
# Purpose: Remove all ArmorClaw files, containers, volumes, and configs
# Version: 0.1.0
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

print_header() {
    printf '%b' "${CYAN}"
    printf '╔══════════════════════════════════════════════════════╗\n'
    printf '║        %bArmorClaw Cleanup Script%b%b                   ║\n' "${BOLD}" "${NC}" "${CYAN}"
    printf '╚══════════════════════════════════════════════════════╝\n'
    printf '%b' "${NC}"
}

print_success() { printf '%b✓ %s%b\n' "${GREEN}" "$1" "${NC}"; }
print_error() { printf '%b✗ ERROR: %s%b\n' "${RED}" "$1" "${NC}" >&2; }
print_warning() { printf '%b⚠ WARNING: %s%b\n' "${YELLOW}" "$1" "${NC}"; }
print_info() { printf '%bℹ %s%b\n' "${CYAN}" "$1" "${NC}"; }

# Check for root/sudo
if [ "$EUID" -ne 0 ]; then
    print_error "This script must be run as root or with sudo"
    exit 1
fi

print_header
echo ""

# Confirmation prompt
printf '%bThis will remove ALL ArmorClaw data including:%b\n' "${YELLOW}" "${NC}"
echo "  • Docker containers and images"
echo "  • Docker volumes (keystore, configs, data)"
echo "  • System directories (/etc/armorclaw, /var/lib/armorclaw, etc.)"
echo "  • User config (~/.armorclaw)"
echo "  • Installer scripts"
echo ""
printf '%bThis action cannot be undone!%b\n\n' "${RED}" "${NC}"

read -p "Are you sure you want to continue? [y/N] " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Cleanup cancelled"
    exit 0
fi

echo ""
print_info "Starting cleanup..."
echo ""

# ========================================
# Step 1: Stop and remove Docker containers
# ========================================
print_info "Stopping Docker containers..."

# Stop armorclaw container if running
if docker ps -a --format '{{.Names}}' | grep -q '^armorclaw$'; then
    docker stop armorclaw 2>/dev/null || true
    docker rm armorclaw 2>/dev/null || true
    print_success "Removed armorclaw container"
else
    print_info "No armorclaw container found"
fi

# Stop any other armorclaw-related containers
for container in $(docker ps -a --format '{{.Names}}' | grep -E 'armorclaw|conduit|coturn|sygnal' 2>/dev/null || true); do
    docker stop "$container" 2>/dev/null || true
    docker rm "$container" 2>/dev/null || true
    print_success "Removed container: $container"
done

# ========================================
# Step 2: Remove Docker volumes
# ========================================
print_info "Removing Docker volumes..."

for volume in $(docker volume ls --format '{{.Name}}' | grep -E 'armorclaw|conduit' 2>/dev/null || true); do
    docker volume rm "$volume" 2>/dev/null || true
    print_success "Removed volume: $volume"
done

# ========================================
# Step 3: Remove Docker networks
# ========================================
print_info "Removing Docker networks..."

for network in $(docker network ls --format '{{.Name}}' | grep -E 'armorclaw' 2>/dev/null || true); do
    docker network rm "$network" 2>/dev/null || true
    print_success "Removed network: $network"
done

# ========================================
# Step 4: Remove Docker images (optional)
# ========================================
print_info "Removing Docker images..."

# Ask about images
read -p "Remove ArmorClaw Docker images too? [y/N] " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    for image in $(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E 'armorclaw|mikegemut/armorclaw' 2>/dev/null || true); do
        docker rmi "$image" 2>/dev/null || true
        print_success "Removed image: $image"
    done
else
    print_info "Keeping Docker images"
fi

# ========================================
# Step 5: Remove system directories
# ========================================
print_info "Removing system directories..."

# System config and data directories
for dir in /etc/armorclaw /var/lib/armorclaw /run/armorclaw /var/log/armorclaw; do
    if [ -d "$dir" ]; then
        rm -rf "$dir"
        print_success "Removed: $dir"
    fi
done

# ========================================
# Step 6: Remove user config
# ========================================
print_info "Removing user configuration..."

# Remove from all users' home directories
for home in /home/* /root; do
    if [ -d "$home/.armorclaw" ]; then
        rm -rf "$home/.armorclaw"
        print_success "Removed: $home/.armorclaw"
    fi
done

# ========================================
# Step 7: Remove installer scripts
# ========================================
print_info "Removing installer scripts..."

# Common installer locations
installers=(
    "/home/armorclaw/installer-v4.sh"
    "/root/installer-v4.sh"
    "/tmp/installer-v4.sh"
    "/opt/armorclaw"
)

for item in "${installers[@]}"; do
    if [ -e "$item" ]; then
        rm -rf "$item"
        print_success "Removed: $item"
    fi
done

# ========================================
# Step 8: Clean up any stray files
# ========================================
print_info "Cleaning up stray files..."

# Remove wizard temp files
rm -f /tmp/armorclaw-wizard.json 2>/dev/null || true
rm -f /tmp/armorclaw-*.json 2>/dev/null || true

# Remove any leftover socket files
rm -f /run/armorclaw/bridge.sock 2>/dev/null || true
rm -f /run/armorclaw/bridge.pid 2>/dev/null || true

print_success "Cleaned up stray files"

# ========================================
# Summary
# ========================================
echo ""
printf '%b━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%b\n' "${GREEN}" "${NC}"
printf '%b✓ Cleanup Complete%b\n' "${GREEN}${BOLD}" "${NC}"
printf '%b━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%b\n' "${GREEN}" "${NC}"
echo ""
echo "ArmorClaw has been completely removed from this system."
echo ""
echo "To reinstall:"
echo "  docker run -it --user root \\"
echo "    -v /var/run/docker.sock:/var/run/docker.sock \\"
echo "    -v armorclaw-config:/etc/armorclaw \\"
echo "    -v armorclaw-data:/var/lib/armorclaw \\"
echo "    -p 8443:8443 -p 6167:6167 -p 5000:5000 \\"
echo "    mikegemut/armorclaw:latest"
echo ""
