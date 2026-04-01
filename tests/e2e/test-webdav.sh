#!/bin/bash
# E2E Test: WebDAV Skill
# Tests WebDAV operations (list, get, put, delete) and SSRF protection

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

BRIDGE_BIN="${SCRIPT_DIR}/../../bridge/build/armorclaw-bridge"
WEBDAV_CONTAINER=""
WEBDAV_PORT="8080"
WEBDAV_HOST="localhost"
WEBDAV_URL="http://${WEBDAV_HOST}:${WEBDAV_PORT}"
WEBDAV_USER="testuser"
WEBDAV_PASS="testpass"
export WEBDAV_CONTAINER=""

start_webdav_server() {
    echo -e "${YELLOW}Starting mock WebDAV server...${NC}"

    WEBDAV_CONTAINER=$(
        docker run -d --rm \
            --name "e2e-webdav-$TEST_NS" \
            -p "${WEBDAV_PORT}:80" \
            -e AUTH_TYPE=Basic \
            -e USERNAME="${WEBDAV_USER}" \
            -e PASSWORD="${WEBDAV_PASS}" \
            bytemark/webdav:latest
    )

    if [[ -z "$WEBDAV_CONTAINER" ]]; then
        echo -e "${RED}✗ Failed to start WebDAV container${NC}"
        return 1
    fi

    echo -e "${YELLOW}Waiting for WebDAV server...${NC}"
    local count=0
    while [[ $count -lt 30 ]]; do
        if curl -s -f -o /dev/null -w "%{http_code}" "${WEBDAV_URL}/" 2>/dev/null | grep -q "200\|207"; then
            echo -e "${GREEN}✓ WebDAV server ready${NC}"
            return 0
        fi
        sleep 1
        ((count++)) || true
    done

    echo -e "${RED}✗ WebDAV server not ready${NC}"
    docker stop "$WEBDAV_CONTAINER" 2>/dev/null || true
    return 1
}

stop_webdav_server() {
    echo -e "${YELLOW}Stopping WebDAV server...${NC}"
    if [[ -n "$WEBDAV_CONTAINER" ]]; then
        docker stop "$WEBDAV_CONTAINER" 2>/dev/null || true
        WEBDAV_CONTAINER=""
        echo -e "${GREEN}✓ WebDAV server stopped${NC}"
    fi
}

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up test artifacts...${NC}"

    stop_bridge 2>/dev/null || true

    if [[ -n "$WEBDAV_CONTAINER" ]]; then
        docker stop "$WEBDAV_CONTAINER" 2>/dev/null || true
        WEBDAV_CONTAINER=""
    fi

    rm -f "$BRIDGE_SOCKET" "$BRIDGE_CONFIG" 2>/dev/null || true
    rm -rf "$KEYSTORE_DIR" "$TEST_DIR" 2>/dev/null || true

    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

webdav_rpc_call() {
    local operation="$1"
    local url="$2"
    local content="${3:-}"
    local content_type="${4:-text/plain}"

    local params='{"url":"'"$url"'","operation":"'"$operation"'","username":"'"${WEBDAV_USER}"'","password":"'"${WEBDAV_PASS}"'"}'

    if [[ "$operation" == "put" ]]; then
        local content_base64=$(echo -n "$content" | base64 -w 0)
        params='{"url":"'"$url"'","operation":"'"$operation"'","username":"'"${WEBDAV_USER}"'","password":"'"${WEBDAV_PASS}"'","content":"'"$content_base64"'","content_type":"'"$content_type"'","content_length":'"${#content}"'}'
    fi

    rpc_call "skill.execute" "{\"skill\":\"webdav\",\"params\":$params}"
}

test_webdav_list() {
    echo ""
    echo "Test: WebDAV List Operation"

    start_webdav_server || {
        log_result "webdav_list" "false" "Failed to start WebDAV server"
        return 1
    }

    curl -s -f -X PUT "${WEBDAV_URL}/test-list.txt" \
        -u "${WEBDAV_USER}:${WEBDAV_PASS}" \
        -d "Test content for list operation" || {
        log_result "webdav_list" "false" "Failed to create test file"
        stop_webdav_server
        return 1
    }

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "webdav_list" "false" "Failed to start bridge"
        stop_webdav_server
        return 1
    }

    wait_for_bridge 30 || {
        log_result "webdav_list" "false" "Socket not ready"
        stop_bridge
        stop_webdav_server
        return 1
    }

    result=$(webdav_rpc_call "list" "${WEBDAV_URL}/")

    if [[ -z "$result" ]]; then
        log_result "webdav_list" "false" "No response from bridge"
        stop_bridge
        stop_webdav_server
        return 1
    fi

    if echo "$result" | grep -q '"success":true' && echo "$result" | grep -q '"total":1'; then
        log_result "webdav_list" "true" "Successfully listed WebDAV directory"
    else
        log_result "webdav_list" "false" "List operation failed: $result"
    fi

    stop_bridge
    stop_webdav_server
}

test_webdav_get() {
    echo ""
    echo "Test: WebDAV Get Operation"

    start_webdav_server || {
        log_result "webdav_get" "false" "Failed to start WebDAV server"
        return 1
    }

    local test_content="Test content for get operation at $(date)"
    curl -s -f -X PUT "${WEBDAV_URL}/test-get.txt" \
        -u "${WEBDAV_USER}:${WEBDAV_PASS}" \
        -d "$test_content" || {
        log_result "webdav_get" "false" "Failed to create test file"
        stop_webdav_server
        return 1
    }

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "webdav_get" "false" "Failed to start bridge"
        stop_webdav_server
        return 1
    }

    wait_for_bridge 30 || {
        log_result "webdav_get" "false" "Bridge not ready"
        stop_bridge
        stop_webdav_server
        return 1
    }

    result=$(webdav_rpc_call "get" "${WEBDAV_URL}/test-get.txt")

    if [[ -z "$result" ]]; then
        log_result "webdav_get" "false" "No response from bridge"
        stop_bridge
        stop_webdav_server
        return 1
    fi

    if echo "$result" | grep -q '"success":true'; then
        log_result "webdav_get" "true" "Successfully retrieved WebDAV file"
    else
        log_result "webdav_get" "false" "Get operation failed: $result"
    fi

    stop_bridge
    stop_webdav_server
}

test_webdav_put() {
    echo ""
    echo "Test: WebDAV Put Operation"

    start_webdav_server || {
        log_result "webdav_put" "false" "Failed to start WebDAV server"
        return 1
    }

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "webdav_put" "false" "Failed to start bridge"
        stop_webdav_server
        return 1
    }

    wait_for_bridge 30 || {
        log_result "webdav_put" "false" "Bridge not ready"
        stop_bridge
        stop_webdav_server
        return 1
    }

    local test_content="Test content uploaded via WebDAV put at $(date)"
    result=$(webdav_rpc_call "put" "${WEBDAV_URL}/test-put.txt" "$test_content")

    if [[ -z "$result" ]]; then
        log_result "webdav_put" "false" "No response from bridge"
        stop_bridge
        stop_webdav_server
        return 1
    fi

    if curl -s -f "${WEBDAV_URL}/test-put.txt" -u "${WEBDAV_USER}:${WEBDAV_PASS}" | grep -q "Test content uploaded"; then
        if echo "$result" | grep -q '"success":true'; then
            log_result "webdav_put" "true" "Successfully uploaded file via WebDAV"
        else
            log_result "webdav_put" "false" "Put operation reported failure but file exists: $result"
        fi
    else
        log_result "webdav_put" "false" "Put operation failed to create file: $result"
    fi

    stop_bridge
    stop_webdav_server
}

test_webdav_delete() {
    echo ""
    echo "Test: WebDAV Delete Operation"

    start_webdav_server || {
        log_result "webdav_delete" "false" "Failed to start WebDAV server"
        return 1
    }

    curl -s -f -X PUT "${WEBDAV_URL}/test-delete.txt" \
        -u "${WEBDAV_USER}:${WEBDAV_PASS}" \
        -d "File to be deleted" || {
        log_result "webdav_delete" "false" "Failed to create test file"
        stop_webdav_server
        return 1
    }

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "webdav_delete" "false" "Failed to start bridge"
        stop_webdav_server
        return 1
    }

    wait_for_bridge 30 || {
        log_result "webdav_delete" "false" "Bridge not ready"
        stop_bridge
        stop_webdav_server
        return 1
    }

    result=$(webdav_rpc_call "delete" "${WEBDAV_URL}/test-delete.txt")

    if [[ -z "$result" ]]; then
        log_result "webdav_delete" "false" "No response from bridge"
        stop_bridge
        stop_webdav_server
        return 1
    fi

    if curl -s -f -o /dev/null -w "%{http_code}" "${WEBDAV_URL}/test-delete.txt" \
        -u "${WEBDAV_USER}:${WEBDAV_PASS}" 2>/dev/null | grep -q "404"; then
        if echo "$result" | grep -q '"success":true'; then
            log_result "webdav_delete" "true" "Successfully deleted file via WebDAV"
        else
            log_result "webdav_delete" "false" "Delete operation reported failure but file is deleted: $result"
        fi
    else
        log_result "webdav_delete" "false" "Delete operation failed to remove file: $result"
    fi

    stop_bridge
    stop_webdav_server
}

test_webdav_ssrf_protection() {
    echo ""
    echo "Test: WebDAV SSRF Protection"

    local private_urls=(
        "http://192.168.1.1:8080/"
        "http://10.0.0.1:8080/"
        "http://172.16.0.1:8080/"
        "http://169.254.169.254:8080/"
        "http://127.0.0.1:8080/"
    )

    local blocked_count=0
    local total_urls=${#private_urls[@]}

    cat > "$BRIDGE_CONFIG" << EOF
[keystore]
db_path = "$KEYSTORE_DIR/keystore.db"

[server]
socket_path = "$BRIDGE_SOCKET"

[error_system]
enabled = false
store_enabled = false
EOF

    start_bridge "$BRIDGE_CONFIG" || {
        log_result "webdav_ssrf" "false" "Failed to start bridge"
        return 1
    }

    wait_for_bridge 30 || {
        log_result "webdav_ssrf" "false" "Bridge not ready"
        stop_bridge
        return 1
    }

    for url in "${private_urls[@]}"; do
        result=$(webdav_rpc_call "list" "$url")

        if echo "$result" | grep -q -i "blocked\|denied\|private\|forbidden\|not allowed"; then
            ((blocked_count++)) || true
            echo -e "  ${GREEN}✓${NC} Blocked: $url"
        else
            echo -e "  ${RED}✗${NC} NOT blocked: $url - Result: $result"
        fi
    done

    stop_bridge

    if [[ $blocked_count -eq $total_urls ]]; then
        log_result "webdav_ssrf" "true" "All $total_urls private network URLs blocked by SSRF protection"
    else
        log_result "webdav_ssrf" "false" "Only $blocked_count/$total_urls URLs were blocked"
    fi
}

main() {
    echo "========================================"
    echo "WebDAV E2E Test Suite"
    echo "========================================"
    echo ""

    setup_test_env || exit 1

    check_dependencies || exit 1

    echo ""
    echo "Running tests..."
    echo ""

    test_webdav_list
    test_webdav_get
    test_webdav_put
    test_webdav_delete
    test_webdav_ssrf_protection

    test_summary
    exit $?
}

main "$@"
