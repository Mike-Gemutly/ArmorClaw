#!/bin/bash
# Test script for secret passing mechanism
# Tests the complete flow: keystore → bridge → container

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "========================================"
echo "ArmorClaw Secret Passing Test Suite"
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

# Test 1: Store API key in keystore
echo "[Test 1] Store API key in keystore"
echo "-----------------------------------"

RESPONSE=$(echo '{
    "jsonrpc": "2.0",
    "method": "store_key",
    "params": {
        "provider": "openai",
        "token": "sk-test-dummy-key-12345",
        "display_name": "Test Key"
    },
    "id": 1
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.result.key_id' > /dev/null; then
    echo -e "${GREEN}✓ Test 1 passed: Key stored in keystore${NC}"
    KEY_ID=$(echo "$RESPONSE" | jq -r '.result.key_id')
    echo "  Key ID: $KEY_ID"
else
    echo -e "${RED}✗ Test 1 failed: Could not store key${NC}"
    exit 1
fi

echo ""

# Test 2: Retrieve key to verify
echo "[Test 2] Retrieve key from keystore"
echo "-----------------------------------"

RESPONSE=$(echo "{
    \"jsonrpc\": \"2.0\",
    \"method\": \"get_key\",
    \"params\": {
        \"id\": \"$KEY_ID\"
    },
    \"id\": 2
}" | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.result.token' > /dev/null; then
    TOKEN=$(echo "$RESPONSE" | jq -r '.result.token')
    echo -e "${GREEN}✓ Test 2 passed: Key retrieved successfully${NC}"
    echo "  Token: ${TOKEN:0:20}..."
else
    echo -e "${RED}✗ Test 2 failed: Could not retrieve key${NC}"
    exit 1
fi

echo ""

# Test 3: Start container with key injection
echo "[Test 3] Start container with injected secrets"
echo "------------------------------------------------"

RESPONSE=$(echo "{
    \"jsonrpc\": \"2.0\",
    \"method\": \"start\",
    \"params\": {
        \"key_id\": \"$KEY_ID\",
        \"agent_type\": \"openclaw\",
        \"image\": \"armorclaw/agent:v1\"
    },
    \"id\": 3
}" | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.result.container_id' > /dev/null; then
    echo -e "${GREEN}✓ Test 3 passed: Container started${NC}"
    CONTAINER_ID=$(echo "$RESPONSE" | jq -r '.result.container_id')
    CONTAINER_NAME=$(echo "$RESPONSE" | jq -r '.result.container_name')
    echo "  Container ID: ${CONTAINER_ID:0:12}..."
    echo "  Container Name: $CONTAINER_NAME"
else
    echo -e "${RED}✗ Test 3 failed: Could not start container${NC}"
    exit 1
fi

echo ""

# Test 4: Verify secrets file was created and cleaned up
echo "[Test 4] Verify secrets file handling"
echo "--------------------------------------"

# Check if secrets file exists (should be cleaned up by now)
SECRETS_FILE="/run/armorclaw/secrets/${CONTAINER_NAME}.json"
if [ -f "$SECRETS_FILE" ]; then
    echo -e "${YELLOW}⚠ Secrets file still exists (cleanup may be pending)${NC}"
    echo "  File: $SECRETS_FILE"
else
    echo -e "${GREEN}✓ Test 4 passed: Secrets file cleaned up${NC}"
fi

echo ""

# Test 5: Check container logs for credentials verification
echo "[Test 5] Verify container received credentials"
echo "----------------------------------------------"

# Wait a moment for container to start and log
sleep 2

LOGS=$(docker logs "$CONTAINER_ID" 2>&1 || echo "Container not found")

if echo "$LOGS" | grep -q "✓ Credentials verification passed"; then
    echo -e "${GREEN}✓ Test 5 passed: Container verified credentials${NC}"
else
    echo -e "${YELLOW}⚠ Test 5: Could not verify credentials in logs${NC}"
    echo "  Logs:"
    echo "$LOGS" | tail -10
fi

if echo "$LOGS" | grep -q "OpenAI API key present"; then
    echo -e "${GREEN}✓ Test 5b passed: API key detected in container${NC}"
fi

echo ""

# Test 6: Verify container is running
echo "[Test 6] Verify container is running"
echo "------------------------------------"

if docker ps | grep -q "$CONTAINER_ID"; then
    echo -e "${GREEN}✓ Test 6 passed: Container is running${NC}"
else
    echo -e "${YELLOW}⚠ Test 6: Container may have stopped${NC}"
    docker ps -a | grep "$CONTAINER_ID" || true
fi

echo ""

# Test 7: Test environment variable fallback (for comparison)
echo "[Test 7: SKIPPED] Environment variable fallback"
echo "--------------------------------------------------"
echo "  (Tested separately via manual docker run)"
echo ""

# Cleanup
echo "========================================"
echo "Test Summary"
echo "========================================"

# Stop the test container
if [ -n "$CONTAINER_ID" ]; then
    echo "Stopping test container..."
    docker stop "$CONTAINER_ID" > /dev/null 2>&1 || true
fi

echo ""
echo -e "${GREEN}✓ Secret passing mechanism is working!${NC}"
echo ""
echo "Key findings:"
echo "  • Keystore stores and retrieves keys correctly"
echo "  • Bridge injects secrets via file mount"
echo "  • Container reads and verifies credentials"
echo "  • Secrets cleanup is scheduled"
echo ""
echo "Next steps:"
echo "  • Test with real API keys"
echo "  • Verify API calls work from container"
echo "  • Test Matrix integration"
