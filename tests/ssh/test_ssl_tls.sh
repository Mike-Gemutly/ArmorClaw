#!/bin/bash
# SSL/TLS Certificate Tests
# Tests certificate presence, expiry, chain, HTTPS connectivity

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Source environment - use absolute path to .env
PROJECT_DIR="/home/mink/src/armorclaw-omo"
EVIDENCE_DIR="$PROJECT_DIR/.sisyphus/evidence"

if [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    echo -e "${RED}Error: .env file not found${NC}"
    exit 2
fi

# Validate required environment variables
if [ -z "\$VPS_IP" ]; then
    echo -e "\${RED}Error: VPS_IP not set\${NC}"
    exit 2
fi

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

echo "========================================="
echo "SSL/TLS Certificate Tests"
echo "========================================="
echo "VPS IP: \$VPS_IP"
echo "VPS User: \$VPS_USER"
echo "========================================="

# Test 1: OpenSSL is available
echo -n "Test 1: OpenSSL is available... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
if command -v openssl >/dev/null 2>&1; then
    echo -e "\${GREEN}[PASS]\${NC} OpenSSL: Available"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${RED}[FAIL]\${NC} OpenSSL: Not available"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

# Test 2: Check certificate on VPS (via SSH)
echo -n "Test 2: Certificate presence on VPS... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
CERT_FOUND=false

if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "\$VPS_USER@\$VPS_IP" "
[ -f /etc/letsencrypt/live/*/fullchain.pem ] 2>/dev/null; then
    CERT_FOUND=true
    echo -e "\${GREEN}[PASS]\${NC} Certificate: Found on VPS"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} Certificate: Not found on VPS (Let's Encrypt may not be configured)"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

# Test 3: Check certificate expiry (if found)
echo -n "Test 3: Certificate expiry check... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))

if [ "\$CERT_FOUND" = true ]; then
    EXPIRY_DATE=\$(ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "\$VPS_USER@\$VPS_IP" "
openssl x509 -in /etc/letsencrypt/live/*/cert.pem -noout -dates 2>/dev/null | grep notAfter | cut -d= -f2)
    
    if [ -n "\$EXPIRY_DATE" ]; then
        # Parse expiry date and check if within 30 days
        EXPIRY_EPOCH=\$(date -d "\$EXPIRY_DATE" +%s 2>/dev/null)
        NOW_EPOCH=\$(date +%s 2>/dev/null)
        DAYS_LEFT=\$(( (\$EXPIRY_EPOCH - \$NOW_EPOCH) / 86400 ))
        
        if [ "\$DAYS_LEFT" -gt 30 ]; then
            echo -e "\${GREEN}[PASS]\${NC} Certificate Expiry: \$DAYS_LEFT days remaining"
            TESTS_PASSED=\$((TESTS_PASSED + 1))
        else
            echo -e "\${RED}[FAIL]\${NC} Certificate Expiry: Expires in \$DAYS_LEFT days (WARNING: <30 days)"
            TESTS_FAILED=\$((TESTS_FAILED + 1))
        fi
    fi
fi

# Test 4: Check certificate chain
echo -n "Test 4: Certificate chain verification... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))

if [ "\$CERT_FOUND" = true ]; then
    CHAIN_VALID=false
    
    if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "\$VPS_USER@\$VPS_IP" "
openssl verify -CAfile /etc/letsencrypt/live/*/chain.pem /etc/letsencrypt/live/*/cert.pem 2>/dev/null; then
        CHAIN_VALID=true
    fi
    
    if \$CHAIN_VALID; then
        echo -e "\${GREEN}[PASS]\${NC} Certificate Chain: Valid"
        TESTS_PASSED=\$((TESTS_PASSED + 1))
    else
        echo -e "\${YELLOW}[WARN]\${NC} Certificate Chain: Could not verify (OpenSSL verify may not be available)"
        TESTS_FAILED=\$((TESTS_FAILED + 1))
    fi
fi

# Test 5: HTTPS connectivity (if port 443/8448 bound)
echo -n "Test 5: HTTPS connectivity test... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
PUBLIC_PORT="\${PUBLIC_PORT:-8448}"

if curl -k --connect-timeout 10 --max-time  15 -s -o /dev/null -w "%{http_code}" https://\$VPS_IP:\$PUBLIC_PORT/ 2>/dev/null | grep -q "200"; then
    echo -e "\${GREEN}[PASS]\${NC} HTTPS Connectivity: Server responding on port \$PUBLIC_PORT"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} HTTPS Connectivity: Server not responding on port \$PUBLIC_PORT (may not have HTTPS)"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Total Tests: \$TESTS_TOTAL"
echo -e "\${GREEN}Passed: \$TESTS_PASSED\${NC}"
echo -e "\${RED}Failed: \$TESTS_FAILED\${NC}"
echo ""

# Save evidence
mkdir -p "\$EVIDENCE_DIR"
cat > "\$EVIDENCE_DIR/task-9-ssl-results.json" << JSONEOF
{
  "test_suite": "SSL/TLS",
  "timestamp": "\$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "vps_ip": "\$VPS_IP",
  "vps_user": "\$VPS_USER",
  "total_tests": \$TESTS_TOTAL,
  "passed": \$TESTS_PASSED,
  "failed": \$TESTS_FAILED,
  "certificate_found": \$CERT_FOUND,
  "days_to_expiry": "\${DAYS_LEFT:-N/A}"
}
JSONEOF

cat > "\$EVIDENCE_DIR/task-9-ssl-success.txt" << CONSOLEEOF
=========================================
SSL/TLS Certificate Tests Complete
=========================================

Total Tests: \$TESTS_TOTAL
Passed: \$TESTS_PASSED
Failed: \$TESTS_FAILED

Certificate Found: \$CERT_FOUND
Days to Expiry: \${DAYS_LEFT:-N/A}
=========================================
CONSOLEEOF

echo -e "\${CYAN}Evidence saved to:\${NC} \$EVIDENCE_DIR/task-9-ssl-*.txt"
echo ""
echo "========================================="
echo "SSL/TLS Certificate Tests Complete"
echo "========================================="
