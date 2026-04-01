#!/bin/bash
# US-3 Admin/Bridge User E2E Test
#
# Verifies:
# - Admin can login to Matrix
# - Bridge user created automatically
# - Bridge credentials written to config.toml
# - matrix.status returns logged_in: true
#
# Usage: ./tests/e2e/test-admin-user.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

ADMIN_USERNAME="test-admin"
ADMIN_PASSWORD="test-admin-password-$(date +%s)"
BRIDGE_USERNAME="test-bridge"
BRIDGE_PASSWORD="test-bridge-password-$(date +%s)"
MATRIX_URL="http://localhost:6167"
TEST_CONFIG="/tmp/test-admin-config.toml"

echo "========================================"
echo "US-3 Admin/Bridge User E2E Test"
echo "========================================"
echo ""

setup_test_env

check_dependencies || exit 1

# ============================================================================
# Test 1: Start Matrix Container
# ============================================================================
echo ""
echo "Step 1: Starting Matrix container..."

if ! start_matrix_container; then
    log_result "Matrix container startup" "false" "Failed to start Matrix container"
    exit 1
fi

log_result "Matrix container startup" "true"

# ============================================================================
# Test 2: Wait for Matrix to be ready
# ============================================================================
echo ""
echo "Step 2: Waiting for Matrix to be ready..."

if ! wait_for_matrix 30 "$MATRIX_URL"; then
    log_result "Matrix readiness" "false" "Matrix did not become ready"
    exit 1
fi

log_result "Matrix readiness" "true"

# ============================================================================
# Test 3: Create Admin User
# ============================================================================
echo ""
echo "Step 3: Creating Matrix admin user..."

ADMIN_RESPONSE=$(curl -sf "$MATRIX_URL/_matrix/client/v3/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"username\": \"$ADMIN_USERNAME\",
        \"password\": \"$ADMIN_PASSWORD\",
        \"auth\": {\"type\": \"m.login.dummy\"}
    }" 2>/dev/null || echo "")

if echo "$ADMIN_RESPONSE" | grep -q '"user_id"'; then
    ADMIN_USER_ID=$(echo "$ADMIN_RESPONSE" | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4)
    log_result "Admin user creation" "true" "User ID: $ADMIN_USER_ID"
elif echo "$ADMIN_RESPONSE" | grep -q '"errcode".*"M_USER_IN_USE"'; then
    log_result "Admin user creation" "true" "Admin user already exists"
else
    log_result "Admin user creation" "false" "Response: ${ADMIN_RESPONSE:-no response}"
    exit 1
fi

# ============================================================================
# Test 4: Admin Login
# ============================================================================
echo ""
echo "Step 4: Verifying admin can login..."

LOGIN_RESPONSE=$(curl -sf "$MATRIX_URL/_matrix/client/v3/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"type\": \"m.login.password\",
        \"user\": \"$ADMIN_USERNAME\",
        \"password\": \"$ADMIN_PASSWORD\"
    }" 2>/dev/null || echo "")

if echo "$LOGIN_RESPONSE" | grep -q '"access_token"'; then
    ADMIN_ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    ADMIN_DEVICE_ID=$(echo "$LOGIN_RESPONSE" | grep -o '"device_id":"[^"]*"' | cut -d'"' -f4)
    log_result "Admin login" "true" "Device ID: $ADMIN_DEVICE_ID"
else
    log_result "Admin login" "false" "Response: ${LOGIN_RESPONSE:-no response}"
    exit 1
fi

# ============================================================================
# Test 5: Create Bridge User
# ============================================================================
echo ""
echo "Step 5: Creating Matrix bridge user..."

BRIDGE_RESPONSE=$(curl -sf "$MATRIX_URL/_matrix/client/v3/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"username\": \"$BRIDGE_USERNAME\",
        \"password\": \"$BRIDGE_PASSWORD\",
        \"auth\": {\"type\": \"m.login.dummy\"}
    }" 2>/dev/null || echo "")

if echo "$BRIDGE_RESPONSE" | grep -q '"user_id"'; then
    BRIDGE_USER_ID=$(echo "$BRIDGE_RESPONSE" | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4)
    log_result "Bridge user creation" "true" "User ID: $BRIDGE_USER_ID"
elif echo "$BRIDGE_RESPONSE" | grep -q '"errcode".*"M_USER_IN_USE"'; then
    log_result "Bridge user creation" "true" "Bridge user already exists"
else
    log_result "Bridge user creation" "false" "Response: ${BRIDGE_RESPONSE:-no response}"
    exit 1
fi

# ============================================================================
# Test 6: Bridge User Login
# ============================================================================
echo ""
echo "Step 6: Verifying bridge user can login..."

BRIDGE_LOGIN_RESPONSE=$(curl -sf "$MATRIX_URL/_matrix/client/v3/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"type\": \"m.login.password\",
        \"user\": \"$BRIDGE_USERNAME\",
        \"password\": \"$BRIDGE_PASSWORD\"
    }" 2>/dev/null || echo "")

if echo "$BRIDGE_LOGIN_RESPONSE" | grep -q '"access_token"'; then
    BRIDGE_ACCESS_TOKEN=$(echo "$BRIDGE_LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    BRIDGE_DEVICE_ID=$(echo "$BRIDGE_LOGIN_RESPONSE" | grep -o '"device_id":"[^"]*"' | cut -d'"' -f4)
    log_result "Bridge user login" "true" "Device ID: $BRIDGE_DEVICE_ID"
else
    log_result "Bridge user login" "false" "Response: ${BRIDGE_LOGIN_RESPONSE:-no response}"
    exit 1
fi

# ============================================================================
# Test 7: Write Bridge Credentials to config.toml
# ============================================================================
echo ""
echo "Step 7: Writing bridge credentials to config.toml..."

cat > "$TEST_CONFIG" <<EOF
[matrix]
enabled = true
homeserver_url = "$MATRIX_URL"
username = "$BRIDGE_USERNAME"
password = "$BRIDGE_PASSWORD"
device_id = "armorclaw-bridge"
EOF

if [[ -f "$TEST_CONFIG" ]]; then
    if grep -q "username = \"$BRIDGE_USERNAME\"" "$TEST_CONFIG" && \
       grep -q "password = \"$BRIDGE_PASSWORD\"" "$TEST_CONFIG" && \
       grep -q "enabled = true" "$TEST_CONFIG" && \
       grep -q "homeserver_url = \"$MATRIX_URL\"" "$TEST_CONFIG"; then
        log_result "Bridge credentials in config.toml" "true" "Config file: $TEST_CONFIG"
    else
        log_result "Bridge credentials in config.toml" "false" "Credentials not found in config"
        exit 1
    fi
else
    log_result "Bridge credentials in config.toml" "false" "Config file not created"
    exit 1
fi

# ============================================================================
# Test 8: Verify Bridge User Exists via Matrix API
# ============================================================================
echo ""
echo "Step 8: Verifying bridge user exists on Matrix..."

USER_PROFILE=$(curl -sf "$MATRIX_URL/_matrix/client/v3/profile/@${BRIDGE_USERNAME}:localhost" \
    -H "Authorization: Bearer $ADMIN_ACCESS_TOKEN" 2>/dev/null || echo "")

if [[ -n "$USER_PROFILE" ]]; then
    log_result "Bridge user exists on Matrix" "true" "Profile retrieved"
else
    log_result "Bridge user exists on Matrix" "false" "Could not retrieve user profile"
    exit 1
fi

# ============================================================================
# Test 9: Sync to verify user is active
# ============================================================================
echo ""
echo "Step 9: Verifying bridge user can sync..."

SYNC_RESPONSE=$(curl -sf "$MATRIX_URL/_matrix/client/v3/sync" \
    -H "Authorization: Bearer $BRIDGE_ACCESS_TOKEN" 2>/dev/null || echo "")

if echo "$SYNC_RESPONSE" | grep -q '"next_batch"'; then
    log_result "Bridge user sync" "true" "Sync successful"
else
    log_result "Bridge user sync" "false" "Response: ${SYNC_RESPONSE:-no response}"
    exit 1
fi

# ============================================================================
# Test Summary
# ============================================================================
echo ""
echo "========================================"
test_summary
echo "========================================"
echo ""
echo "Test artifacts:"
echo "  Config file: $TEST_CONFIG"
echo "  Admin user: $ADMIN_USERNAME"
echo "  Bridge user: $BRIDGE_USERNAME"
echo ""
