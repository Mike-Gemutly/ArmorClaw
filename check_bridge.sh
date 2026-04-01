#!/bin/bash

# Configuration
# This automatically finds your Windows Host IP from WSL
WINDOWS_HOST=$(powershell.exe (Get-NetIPAddress -InterfaceAlias 'vEthernet (WSL*)').IPAddress | tr -d '\r')
BRIDGE_URL="http://${WINDOWS_HOST}:9000/api"
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}--- ArmorClaw Sentinel: Automated Health Check ---${NC}"

# 1. Test Physical Connectivity
echo -n "Checking SSH Tunnel (Port 9000)... "
if nc -z $WINDOWS_HOST 9000 2>/dev/null; then
    echo -e "${GREEN}ACTIVE${NC}"
else
    echo -e "${RED}OFFLINE${NC}"
    echo "Error: Ensure your SSH tunnel (L 9000:127.0.0.1:8081) is running."
    exit 1
fi

# 2. Check Bridge Status (JSON-RPC)
echo -n "Querying Bridge Status... "
STATUS=$(curl -s -X POST $BRIDGE_URL \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"bridge.status","id":1}')

if [[ $STATUS == *"result"* ]]; then
    echo -e "${GREEN}SUCCESS${NC}"
    echo -e "Payload: $STATUS"
else
    echo -e "${RED}FAILED${NC}"
    echo "Error: Bridge returned empty or invalid response."
    exit 1
fi

# 3. Phase 18: SkillGate PII Masking Audit
echo -n "Running SkillGate PII Audit... "
AUDIT=$(curl -s -X POST $BRIDGE_URL \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"security.audit_pii","id":2}')

if [[ $AUDIT == *"***"* ]] || [[ $AUDIT == *"redacted"* ]]; then
    echo -e "${GREEN}SECURE${NC}"
    echo -e "Audit Result: PII Masking is active."
else
    echo -e "${CYAN}WARNING${NC}"
    echo "Audit Result: No PII masking detected in raw response."
fi

echo -e "${CYAN}--------------------------------------------------${NC}"



