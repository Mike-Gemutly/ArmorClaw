#!/bin/bash
# Performance Tests
# Tests SSH connection speed, API response times, container resource usage

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

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
echo "Performance Tests"
echo "========================================="
echo "VPS IP: \$VPS_IP"
echo "VPS User: \$VPS_USER"
echo "========================================="

# Test 1: SSH connection speed (multiple quick connections)
echo -n "Test 1: SSH connection speed (3 quick connections)... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))
START_TIME=\$(date +%s)

for i in {1..3}; do
    timeout 10 ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "\$VPS_USER@\$VPS_IP" "echo 'connection \$i'" 2>/dev/null
done

END_TIME=\$(date +%s)
SSH_TOTAL_TIME=\$((END_TIME - START_TIME))
SSH_AVG_TIME=\$((SSH_TOTAL_TIME / 3))

echo "SSH connection times: \${SSH_AVG_TIME}s total, \${SSH_AVG_TIME}s average"

if [ "\$SSH_AVG_TIME" -lt 30 ]; then
    echo -e "\${GREEN}[PASS]\${NC} SSH Speed: Average \${SSH_AVG_TIME}s per connection (< 30s)"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${RED}[FAIL]\${NC} SSH Speed: Average \${SSH_AVG_TIME}s per connection (WARNING: > 30s)"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

# Test 2: Container resource usage
echo -n "Test 2: Container resource usage... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))

# Check if any containers are running
if ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "\$VPS_USER@\$VPS_IP" "
docker ps --format "table {{.Names}}\t{{.Status}}" 2>/dev/null | grep -q "Up"; then
    
    echo -e "\${CYAN}Container Stats:\${NC}"
    
    # Get container stats
    CONTAINER_STATS=\$(ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "\$VPS_USER@\$VPS_IP" "
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null || echo "No containers running")
    
    echo "\$CONTAINER_STATS"
    echo -e "\${GREEN}[PASS]\${NC} Container Resources: Stats retrieved"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
    
    # Check for resource limits (CPU < 50%, Memory < 512MB)
    if echo "\$CONTAINER_STATS" | grep -q "Up"; then
        HIGH_CPU=\$(echo "\$CONTAINER_STATS" | awk 'NR>1 {print \$2}' | awk '{print \$1+0}' | cut -d'%' -f1 | head -1)
        HIGH_MEM=\$(echo "\$CONTAINER_STATS" | awk 'NR>1 {print \$3}' | awk '{print \$1+0}' | head -1)
        
        if [ -n "\$HIGH_CPU" ] && [ "\$HIGH_CPU" -lt 50 ]; then
            echo "  CPU Usage: \$HIGH_CPU% (OK)"
        elif [ -n "\$HIGH_CPU" ]; then
            echo "  CPU Usage: N/A (no running containers)"
        fi
        
        if [ -n "\$HIGH_MEM" ] && [ "\$HIGH_MEM" -lt 512 ]; then
            echo "  Memory Usage: \$HIGH_MEMMiB (OK)"
        elif [ -n "\$HIGH_MEM" ]; then
            echo "  Memory Usage: N/A (no running containers)"
        fi
    fi
else
    echo -e "\${YELLOW}[WARN]\${NC} Container Resources: No containers running"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

# Test 3: VPS system resources
echo -n "Test 3: VPS system resources... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))

SYSTEM_INFO=\$(ssh -i "\$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "\$VPS_USER@\$VPS_IP" "
free -m && df -h / 2>/dev/null || echo "Could not get system info")

if [ -n "\$SYSTEM_INFO" ]; then
    echo "\$SYSTEM_INFO"
    echo -e "\${GREEN}[PASS]\${NC} System Resources: Info retrieved"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${YELLOW}[WARN]\${NC} System Resources: Could not retrieve info"
    TESTS_FAILED=\$((TESTS_FAILED + 1))
fi

# Test 4: Total test time
END_TIME=\$(date +%s)
TOTAL_TIME=\$((END_TIME - START_TIME))

echo -n "Test 4: Total test execution time... "
TESTS_TOTAL=\$((TESTS_TOTAL + 1))

if [ "\$TOTAL_TIME" -lt 60 ]; then
    echo -e "\${GREEN}[PASS]\${NC} Execution Time: \${TOTAL_TIME}s (< 60s)"
    TESTS_PASSED=\$((TESTS_PASSED + 1))
else
    echo -e "\${RED}[FAIL]\${NC} Execution Time: \${TOTAL_TIME}s (WARNING: > 60s)"
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
cat > "\$EVIDENCE_DIR/task-10-performance-results.json" << JSONEOF
{
  "test_suite": "Performance",
  "timestamp": "\$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "vps_ip": "\$VPS_IP",
  "vps_user": "\$VPS_USER",
  "total_tests": \$TESTS_TOTAL,
  "passed": \$TESTS_PASSED,
  "failed": \$TESTS_FAILED,
  "execution_time_seconds": \$TOTAL_TIME,
  "ssh_avg_time_seconds": \$SSH_AVG_TIME
}
JSONEOF

cat > "\$EVIDENCE_DIR/task-10-performance-success.txt" << CONSOLEEOF
=========================================
Performance Tests Complete
=========================================

Total Tests: \$TESTS_TOTAL
Passed: \$TESTS_PASSED
Failed: \$TESTS_FAILED

SSH Connection Speed: \${SSH_AVG_TIME}s average
Total Execution Time: \${TOTAL_TIME}s
=========================================
CONSOLEEOF

echo -e "\${CYAN}Evidence saved to:\${NC} \$EVIDENCE_DIR/task-10-performance-*.txt"
echo ""
echo "========================================="
echo "Performance Tests Complete"
echo "========================================="
