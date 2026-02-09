#!/bin/bash
#
# ArmorClaw Infrastructure Validation Script
# Validates that all services are running correctly
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}ArmorClaw Infrastructure Validation${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

PASSED=0
FAILED=0

# Test function
test_service() {
    local name="$1"
    local test_command="$2"

    echo -n "Testing $name... "

    if eval "$test_command" &> /dev/null; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((FAILED++))
        return 1
    fi
}

# 1. Docker is running
echo -e "${YELLOW}1. Docker Environment${NC}"
test_service "Docker daemon" "docker info &> /dev/null"
test_service "docker-compose" "command -v docker-compose"

echo ""

# 2. Containers are running
echo -e "${YELLOW}2. Container Status${NC}"
test_service "Matrix Conduit container" "docker ps | grep armorclaw-conduit"
test_service "Nginx container" "docker ps | grep armorclaw-nginx"
test_service "Coturn container" "docker ps | grep armorclaw-coturn"

echo ""

# 3. Services are responding
echo -e "${YELLOW}3. Service Endpoints${NC}"
test_service "Matrix Conduit API" "curl -f http://localhost:6167/_matrix/client/versions"
test_service "Nginx health check" "curl -f http://localhost/health"

echo ""

# 4. SSL Certificate (if enabled)
echo -e "${YELLOW}4. SSL Certificate${NC}"
if [ -f configs/nginx/ssl/fullchain.pem ]; then
    test_service "SSL certificate exists" "true"
    test_service "SSL certificate valid" "openssl x509 -checkend 86400 -in configs/nginx/ssl/fullchain.pem"
else
    echo -e "${YELLOW}⚠ SSL not configured (HTTP only)${NC}"
fi

echo ""

# 5. Well-known Matrix endpoints
echo -e "${YELLOW}5. Matrix Federation${NC}"
test_service "Matrix server endpoint" "curl -f http://localhost/.well-known/matrix/server"
test_service "Matrix client endpoint" "curl -f http://localhost/.well-known/matrix/client"

echo ""

# 6. Memory usage
echo -e "${YELLOW}6. Resource Usage${NC}"
echo "Container memory usage:"
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.MemPerc}}" | head -10

echo ""

# 7. Log check for errors
echo -e "${YELLOW}7. Error Log Check${NC}"
CONDUIT_ERRORS=$(docker-compose logs matrix-conduit 2>&1 | grep -i error | wc -l)
if [ "$CONDUIT_ERRORS" -gt 0 ]; then
    echo -e "${YELLOW}⚠ Found $CONDUIT_ERRORS error(s) in Conduit logs${NC}"
else
    echo -e "${GREEN}✓ No errors in Conduit logs${NC}"
fi

echo ""

# Summary
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Validation Summary${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ All checks passed! Infrastructure is ready.${NC}"
    echo ""
    echo "Next step: Deploy the Go Bridge"
    echo "  1. Initialize Go repository"
    echo "  2. Implement keystore (Task 1.3)"
    echo "  3. Build bridge binary"
    echo "  4. Test agent communication"
    exit 0
else
    echo ""
    echo -e "${RED}✗ Some checks failed. Review logs above.${NC}"
    exit 1
fi
