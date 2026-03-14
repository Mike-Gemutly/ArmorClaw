#!/bin/bash
# certbot-healthcheck.sh - Check if Certbot systemd timer is active
# Returns 0 if healthy, 1 if unhealthy

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print success message
pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    exit 0
}

# Function to print failure message
fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    exit 1
}

# Function to print info message
info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Check if timer is active
info "Checking Certbot timer status..."

if systemctl is-active --quiet certbot.timer 2>/dev/null; then
    pass "Certbot timer is active"
fi

if systemctl is-active --quiet certbot-renew.timer 2>/dev/null; then
    pass "Certbot renew timer is active"
fi

# If we reach here, no timer is active
fail "Certbot timer is not active"
