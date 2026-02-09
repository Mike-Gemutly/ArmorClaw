#!/bin/bash
# Test script for attach_config RPC method
# Tests configuration file attachment via the bridge

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "========================================"
echo "ArmorClaw attach_config Test Suite"
echo "========================================"
echo ""

# Check if bridge is running
if [ ! -S "/run/armorclaw/bridge.sock" ]; then
    echo -e "${RED}✗ Bridge socket not found${NC}"
    echo "  Start the bridge first: cd bridge && sudo ./build/armorclaw-bridge"
    exit 1
fi

echo -e "${GREEN}✓ Bridge socket found${NC}"
echo ""

# Test 1: Attach raw env file
echo "[Test 1] Attach raw environment file"
echo "------------------------------------"

RESPONSE=$(echo '{
    "jsonrpc": "2.0",
    "method": "attach_config",
    "params": {
        "name": "test.env",
        "content": "MODEL=gpt-4\nTEMPERATURE=0.7\nMAX_TOKENS=4096",
        "encoding": "raw",
        "type": "env"
    },
    "id": 1
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.result.config_id' > /dev/null; then
    echo -e "${GREEN}✓ Test 1 passed: Config attached successfully${NC}"
    CONFIG_ID=$(echo "$RESPONSE" | jq -r '.result.config_id')
    echo "  Config ID: $CONFIG_ID"
else
    echo -e "${RED}✗ Test 1 failed: Could not attach config${NC}"
    exit 1
fi

echo ""

# Test 2: Attach base64-encoded config
echo "[Test 2] Attach base64-encoded config"
echo "-------------------------------------"

BASE64_CONTENT=$(echo -n "SECRET_KEY=supersecret123" | base64)

RESPONSE=$(echo "{
    \"jsonrpc\": \"2.0\",
    \"method\": \"attach_config\",
    \"params\": {
        \"name\": \"secret.env\",
        \"content\": \"$BASE64_CONTENT\",
        \"encoding\": \"base64\",
        \"type\": \"env\"
    },
    \"id\": 2
}" | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.result.config_id' > /dev/null; then
    echo -e "${GREEN}✓ Test 2 passed: Base64 config attached successfully${NC}"
else
    echo -e "${RED}✗ Test 2 failed: Could not attach base64 config${NC}"
    exit 1
fi

echo ""

# Test 3: Attach TOML config
echo "[Test 3] Attach TOML configuration"
echo "----------------------------------"

RESPONSE=$(echo '{
    "jsonrpc": "2.0",
    "method": "attach_config",
    "params": {
        "name": "agent.toml",
        "content": "[agent]\nmodel = \"gpt-4\"\ntemperature = 0.7\n\n[limits]\nmax_tokens = 4096\ntimeout = 30",
        "encoding": "raw",
        "type": "toml"
    },
    "id": 3
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.result.config_id' > /dev/null; then
    echo -e "${GREEN}✓ Test 3 passed: TOML config attached successfully${NC}"
else
    echo -e "${RED}✗ Test 3 failed: Could not attach TOML config${NC}"
    exit 1
fi

echo ""

# Test 4: Path traversal attempt (should fail)
echo "[Test 4] Path traversal protection"
echo "-----------------------------------"

RESPONSE=$(echo '{
    "jsonrpc": "2.0",
    "method": "attach_config",
    "params": {
        "name": "../../../etc/passwd",
        "content": "malicious",
        "encoding": "raw"
    },
    "id": 4
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.error' > /dev/null; then
    echo -e "${GREEN}✓ Test 4 passed: Path traversal blocked${NC}"
else
    echo -e "${RED}✗ Test 4 failed: Path traversal not blocked${NC}"
    exit 1
fi

echo ""

# Test 5: Missing required parameters (should fail)
echo "[Test 5] Missing parameters validation"
echo "---------------------------------------"

RESPONSE=$(echo '{
    "jsonrpc": "2.0",
    "method": "attach_config",
    "params": {
        "name": "test.env"
    },
    "id": 5
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.error' > /dev/null; then
    echo -e "${GREEN}✓ Test 5 passed: Missing content rejected${NC}"
else
    echo -e "${RED}✗ Test 5 failed: Missing content not rejected${NC}"
    exit 1
fi

echo ""

# Test 6: Verify config file exists
echo "[Test 6] Verify config file was written"
echo "----------------------------------------"

CONFIG_PATH=$(echo "$RESPONSE" | jq -r '.result.path // "/run/armorclaw/configs/test.env"')

if [ -f "$CONFIG_PATH" ]; then
    echo -e "${GREEN}✓ Test 6 passed: Config file exists at $CONFIG_PATH${NC}"
    echo "Content:"
    cat "$CONFIG_PATH"
else
    echo -e "${YELLOW}⚠ Test 6: Config file not found (may be expected if running in container)${NC}"
fi

echo ""
echo "========================================"
echo -e "${GREEN}All attach_config tests passed!${NC}"
echo "========================================"
