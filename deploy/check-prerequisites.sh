#!/bin/bash
# ArmorClaw Prerequisites Check
# Validates system requirements before deployment

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASS=0
WARN=0
FAIL=0

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASS++))
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAIL++))
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARN++))
}

echo -e "${BLUE}═════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}     ArmorClaw Prerequisites Check${NC}"
echo -e "${BLUE}═════════════════════════════════════════════════════════${NC}"
echo ""

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    check_warn "Running as root (recommended for deployment)"
else
    check_fail "Not running as root (use sudo for deployment)"
    echo "  Run: sudo ./check-prerequisites.sh"
fi

echo ""
echo "=== System Requirements ==="

# OS Check
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    if [[ "$ID" == "ubuntu" ]] || [[ "$ID" == "debian" ]]; then
        check_pass "OS: $PRETTY_NAME"
    else
        check_warn "OS: $PRETTY_NAME (Ubuntu/Debian recommended)"
    fi
else
    check_fail "Cannot detect OS"
fi

# Memory Check (minimum 2GB)
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
if (( TOTAL_MEM >= 2048 )); then
    check_pass "Memory: ${TOTAL_MEM}MB (minimum 2GB)"
elif (( TOTAL_MEM >= 1024 )); then
    check_warn "Memory: ${TOTAL_MEM}MB (2GB recommended, 1GB minimum)"
else
    check_fail "Memory: ${TOTAL_MEM}MB (2GB required)"
fi

# Disk Space Check (minimum 5GB free)
DISK_FREE=$(df -m / | tail -1 | awk '{print $4}')
if (( DISK_FREE >= 5120 )); then
    check_pass "Disk Space: ${DISK_FREE}MB free (minimum 5GB)"
elif (( DISK_FREE >= 2048 )); then
    check_warn "Disk Space: ${DISK_FREE}MB free (5GB recommended)"
else
    check_fail "Disk Space: ${DISK_FREE}MB free (5GB required)"
fi

echo ""
echo "=== Software Requirements ==="

# Docker
if command -v docker &>/dev/null; then
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | tr -d ',')
    if docker info &>/dev/null; then
        check_pass "Docker: $DOCKER_VERSION (running)"
    else
        check_fail "Docker installed but not running"
        echo "  Fix: sudo systemctl start docker"
    fi
else
    check_fail "Docker not installed"
    echo "  Fix: curl -fsSL https://get.docker.com | sh"
fi

# Docker Compose
if command -v docker-compose &>/dev/null; then
    COMPOSE_VERSION=$(docker-compose --version | awk '{print $4}' | tr -d ',')
    check_pass "Docker Compose: $COMPOSE_VERSION"
else
    if docker compose version &>/dev/null; then
        COMPOSE_VERSION=$(docker compose version | awk '{print $4}' | tr -d ',')
        check_pass "Docker Compose (plugin): $COMPOSE_VERSION"
    else
        check_fail "Docker Compose not installed"
        echo "  Fix: sudo apt install docker-compose"
    fi
fi

# curl
if command -v curl &>/dev/null; then
    check_pass "curl: installed"
else
    check_fail "curl not installed"
    echo "  Fix: sudo apt install curl"
fi

# jq (for JSON parsing)
if command -v jq &>/dev/null; then
    check_pass "jq: installed"
else
    check_warn "jq not installed (optional, for advanced scripts)"
    echo "  Fix: sudo apt install jq"
fi

# socat (for socket communication)
if command -v socat &>/dev/null; then
    check_pass "socat: installed"
else
    check_warn "socat not installed (needed for bridge communication)"
    echo "  Fix: sudo apt install socat"
fi

echo ""
echo "=== Network Requirements ==="

# Port checks
PORTS=(80 443 6167)
PORTS_OK=true
for PORT in "${PORTS[@]}"; do
    if ss -tlnp 2>/dev/null | grep -q ":$PORT "; then
        check_warn "Port $PORT is already in use"
        PORTS_OK=false
    fi
done
if $PORTS_OK; then
    check_pass "Required ports available (80, 443, 6167)"
fi

# Internet connectivity
if ping -c 1 8.8.8.8 &>/dev/null; then
    check_pass "Internet connectivity: OK"
else
    check_fail "No internet connectivity"
fi

echo ""
echo "=== File Requirements ==="

# .env file
if [[ -f .env ]]; then
    check_pass ".env file exists"
else
    if [[ -f .env.example ]]; then
        check_warn ".env file missing (but .env.example exists)"
        echo "  Fix: cp .env.example .env && nano .env"
    else
        check_fail ".env and .env.example missing"
    fi
fi

# docker-compose file
if [[ -f docker-compose-stack.yml ]]; then
    check_pass "docker-compose-stack.yml exists"
elif [[ -f docker-compose.yml ]]; then
    check_pass "docker-compose.yml exists"
else
    check_fail "No docker-compose file found"
fi

# Caddyfile
if [[ -f Caddyfile ]]; then
    check_pass "Caddyfile exists"
else
    check_warn "Caddyfile missing (SSL auto-provisioning may not work)"
fi

echo ""
echo "=== Optional Enhancements ==="

# fail2ban
if systemctl is-active fail2ban &>/dev/null; then
    check_pass "fail2ban: running"
elif command -v fail2ban-server &>/dev/null; then
    check_warn "fail2ban installed but not running"
else
    echo -e "${BLUE}○${NC} fail2ban: not installed (recommended for security)"
fi

# ufw firewall
if command -v ufw &>/dev/null; then
    if ufw status | grep -q "Status: active"; then
        check_pass "UFW firewall: active"
    else
        check_warn "UFW firewall: not active"
        echo "  Fix: sudo ufw enable"
    fi
else
    echo -e "${BLUE}○${NC} UFW firewall: not installed (recommended for security)"
fi

echo ""
echo -e "${BLUE}═════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}     Prerequisites Check Complete${NC}"
echo -e "${BLUE}═════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${GREEN}Passed:${NC}   $PASS"
echo -e "  ${YELLOW}Warnings:${NC} $WARN"
echo -e "  ${RED}Failed:${NC}   $FAIL"
echo ""

if (( FAIL > 0 )); then
    echo -e "${RED}✗ Cannot proceed with deployment${NC}"
    echo "  Please fix the failed checks above."
    exit 1
elif (( WARN > 0 )); then
    echo -e "${YELLOW}⚠ Can proceed, but warnings should be addressed${NC}"
    exit 0
else
    echo -e "${GREEN}✓ All prerequisites met! Ready for deployment.${NC}"
    exit 0
fi
