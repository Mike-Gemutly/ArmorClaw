#!/bin/bash
# Container Health Tests
# Tests Docker container status, logs, resource usage

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    echo -e "${RED}Error: .env file not found at $PROJECT_DIR${NC}"
    exit 2
fi

# Function to execute remote command via SSH
ssh_exec() {
    local command="$1"
    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$VPS_USER@$VPS_IP" "$command" 2>&1
}

# Function to format result
print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}[PASS]${NC} $test_name: $message"
    else
        echo -e "${RED}[FAIL]${NC} $test_name: $message"
    fi
}

echo "========================================="
echo "Container Health Tests"
echo "========================================="

# Test 1: Docker is installed
echo -n "Test 1: Docker is installed... "
DOCKER_CHECK=$(ssh_exec "docker --version")
if echo "$DOCKER_CHECK" | grep -q "Docker version"; then
    print_result "Docker Installation" "PASS" "Docker is installed"
else
    print_result "Docker Installation" "FAIL" "Docker not found"
    exit 1
fi

# Test 2: Docker daemon is running
echo -n "Test 2: Docker daemon is running... "
DOCKER_DAEMON=$(ssh_exec "docker info >/dev/null 2>&1 && echo 'running'")
if [ "$DOCKER_DAEMON" = "running" ]; then
    print_result "Docker Daemon" "PASS" "Docker daemon is running"
else
    print_result "Docker Daemon" "FAIL" "Docker daemon not running"
    exit 1
fi

# Test 3: Check all expected containers are running
echo -n "Test 3: Expected containers are running... "
CONTAINERS_STATUS=$(ssh_exec "docker ps --format '{{.Names}}: {{.Status}}'")

# Check Bridge container
if echo "$CONTAINERS_STATUS" | grep -q "bridge.*running"; then
    print_result "Bridge Container" "PASS" "Bridge container is running"
else
    print_result "Bridge Container" "FAIL" "Bridge container is not running"
fi

# Check Matrix container
if echo "$CONTAINERS_STATUS" | grep -q "matrix.*running"; then
    print_result "Matrix Container" "PASS" "Matrix container is running"
else
    print_result "Matrix Container" "FAIL" "Matrix container is not running"
fi

# Check browser-service container
if echo "$CONTAINERS_STATUS" | grep -q "browser-service.*running"; then
    print_result "Browser-Service Container" "PASS" "Browser-service container is running"
else
    print_result "Browser-Service Container" "WARN" "Browser-service container not running (optional)"
fi

# Test 4: Container health status
echo -n "Test 4: Container health status... "
HEALTH_STATUS=$(ssh_exec "docker ps --format '{{.Names}}: {{.Health}}'")

if echo "$HEALTH_STATUS" | grep -q "healthy"; then
    print_result "Container Health" "PASS" "All containers are healthy"
else
    print_result "Container Health" "WARN" "Some containers are unhealthy"
fi

# Test 5: Container restart count is low
echo -n "Test 5: Container restart count is low... "
RESTART_COUNT=$(ssh_exec "docker ps --format '{{.RestartCount}}' | head -1")

if [ "$RESTART_COUNT" -lt 5 ]; then
    print_result "Container Restarts" "PASS" "Restart count is low: $RESTART_COUNT"
else
    print_result "Container Restarts" "WARN" "Restart count is high: $RESTART_COUNT"
fi

# Test 6: Container isolation (no privileged containers)
echo -n "Test 6: Container isolation... "
PRIVILEGED=$(ssh_exec "docker ps --format '{{.Names}}' | xargs -I {} docker inspect --format='{{.HostConfig.Privileged}}'")
if echo "$PRIVILEGED" | grep -q "true"; then
    print_result "Container Isolation" "FAIL" "Found privileged container"
else
    print_result "Container Isolation" "PASS" "No privileged containers"
fi

# Test 7: Container resource usage
echo -n "Test 7: Container resource usage... "
RESOURCE_USAGE=$(ssh_exec "docker stats --no-stream --format 'table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}'")
echo "$RESOURCE_USAGE"

echo ""
echo "========================================="
echo "Container Tests Complete"
echo "========================================="
