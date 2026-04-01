#!/bin/bash
# E2E Test for US-7 Calendar
# Tests calendar skill functionality including:
# - Creating events via CalDAV
# - Verifying events appear
# - Conflict detection

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Configuration
BRIDGE_BIN="${SCRIPT_DIR}/../../bridge/build/armorclaw-bridge"

# Test data
CALDAV_URL="http://localhost:8080/calendars/default/"
TEST_EVENT_TITLE="E2E Test Meeting"
TEST_EVENT_START="2026-03-30T10:00:00Z"
TEST_EVENT_END="2026-03-30T11:00:00Z"
TEST_EVENT_LOCATION="Room 101"

# State tracking
CREATED_EVENT_UID=""

# ============================================================================
# Test: Calendar - Create Event
# ============================================================================

test_calendar_create_event() {
    echo ""
    echo "Test: Calendar - Create Event"

    local params
    params=$(cat <<EOF
{
    "operation": "create_event",
    "calendar_url": "$CALDAV_URL",
    "title": "$TEST_EVENT_TITLE",
    "start_time": "$TEST_EVENT_START",
    "end_time": "$TEST_EVENT_END",
    "location": "$TEST_EVENT_LOCATION",
    "description": "E2E test event"
}
EOF
)

    local result
    result=$(rpc_call "skills.execute" "{\"skill_name\":\"calendar\",\"params\":$params}")

    if [[ -z "$result" ]]; then
        log_result "calendar_create_event" "false" "No response from bridge"
        return 1
    fi

    CREATED_EVENT_UID=$(echo "$result" | grep -o '"event_uid":"[^"]*"' | cut -d'"' -f4 || echo "")

    if [[ -z "$CREATED_EVENT_UID" ]]; then
        log_result "calendar_create_event" "false" "Failed to extract event UID from response: $result"
        return 1
    fi


    if echo "$result" | grep -q '"status":"success"'; then
        log_result "calendar_create_event" "true" "Event created with UID: $CREATED_EVENT_UID"
        return 0
    else
        log_result "calendar_create_event" "false" "Event creation failed: $result"
        return 1
    fi
}

test_calendar_get_events() {
    echo ""
    echo "Test: Calendar - Get Events"

    local params
    params=$(cat <<EOF
{
    "operation": "get_events",
    "calendar_url": "$CALDAV_URL"
}
EOF
)

    local result
    result=$(rpc_call "skills.execute" "{\"skill_name\":\"calendar\",\"params\":$params}")

    if [[ -z "$result" ]]; then
        log_result "calendar_get_events" "false" "No response from bridge"
        return 1
    fi

    if echo "$result" | grep -q '"events"'; then
        log_result "calendar_get_events" "true" "Successfully retrieved events"
        return 0
    else
        log_result "calendar_get_events" "false" "No events returned: $result"
        return 1
    fi
}

test_calendar_verify_event() {
    echo ""
    echo "Test: Calendar - Verify Event Appears"

    if [[ -z "$CREATED_EVENT_UID" ]]; then
        log_result "calendar_verify_event" "false" "No event UID available (create_event must run first)"
        return 1
    fi

    local params
    params=$(cat <<EOF
{
    "operation": "get_event",
    "calendar_url": "$CALDAV_URL",
    "event_data": {
        "uid": "$CREATED_EVENT_UID"
    }
}
EOF
)

    local result
    result=$(rpc_call "skills.execute" "{\"skill_name\":\"calendar\",\"params\":$params}")

    if [[ -z "$result" ]]; then
        log_result "calendar_verify_event" "false" "No response from bridge"
        return 1
    fi


    if echo "$result" | grep -q "\"title\":\"$TEST_EVENT_TITLE\""; then
        log_result "calendar_verify_event" "true" "Event verified with title: $TEST_EVENT_TITLE"
        return 0
    else
        log_result "calendar_verify_event" "false" "Event title mismatch or not found: $result"
        return 1
    fi
}

test_calendar_conflict_detection() {
    echo ""
    echo "Test: Calendar - Conflict Detection"


    local params
    params=$(cat <<EOF
{
    "operation": "create_event",
    "calendar_url": "$CALDAV_URL",
    "title": "Conflicting Meeting",
    "start_time": "$TEST_EVENT_START",
    "end_time": "$TEST_EVENT_END",
    "description": "This should conflict with the first event"
}
EOF
)

    local result
    result=$(rpc_call "skills.execute" "{\"skill_name\":\"calendar\",\"params\":$params}")

    if [[ -z "$result" ]]; then
        log_result "calendar_conflict_detection" "false" "No response from bridge"
        return 1
    fi


    if echo "$result" | grep -q '"conflicts_detected":true'; then
        log_result "calendar_conflict_detection" "true" "Conflicts correctly detected"
        return 0
    elif echo "$result" | grep -q '"conflict_count":"[1-9]"'; then
        log_result "calendar_conflict_detection" "true" "Conflicts detected via metadata"
        return 0
    else
        log_result "calendar_conflict_detection" "false" "Conflict detection failed: $result"
        return 1
    fi
}

test_calendar_update_event() {
    echo ""
    echo "Test: Calendar - Update Event"

    if [[ -z "$CREATED_EVENT_UID" ]]; then
        log_result "calendar_update_event" "false" "No event UID available (create_event must run first)"
        return 1
    fi

    local updated_title="Updated E2E Test Meeting"
    local updated_start="2026-03-30T14:00:00Z"
    local updated_end="2026-03-30T15:00:00Z"

    local params
    params=$(cat <<EOF
{
    "operation": "update_event",
    "calendar_url": "$CALDAV_URL",
    "event_data": {
        "uid": "$CREATED_EVENT_UID"
    },
    "title": "$updated_title",
    "start_time": "$updated_start",
    "end_time": "$updated_end"
}
EOF
)

    local result
    result=$(rpc_call "skills.execute" "{\"skill_name\":\"calendar\",\"params\":$params}")

    if [[ -z "$result" ]]; then
        log_result "calendar_update_event" "false" "No response from bridge"
        return 1
    fi


    if echo "$result" | grep -q '"status":"success"'; then
        log_result "calendar_update_event" "true" "Event updated successfully"
        return 0
    else
        log_result "calendar_update_event" "false" "Event update failed: $result"
        return 1
    fi
}

test_calendar_delete_event() {
    echo ""
    echo "Test: Calendar - Delete Event"

    if [[ -z "$CREATED_EVENT_UID" ]]; then
        log_result "calendar_delete_event" "false" "No event UID available (create_event must run first)"
        return 1
    fi

    local params
    params=$(cat <<EOF
{
    "operation": "delete_event",
    "calendar_url": "$CALDAV_URL",
    "event_data": {
        "uid": "$CREATED_EVENT_UID"
    }
}
EOF
)

    local result
    result=$(rpc_call "skills.execute" "{\"skill_name\":\"calendar\",\"params\":$params}")

    if [[ -z "$result" ]]; then
        log_result "calendar_delete_event" "false" "No response from bridge"
        return 1
    fi


    if echo "$result" | grep -q '"status":"success"'; then
        log_result "calendar_delete_event" "true" "Event deleted successfully"
        return 0
    else
        log_result "calendar_delete_event" "false" "Event deletion failed: $result"
        return 1
    fi
}

test_calendar_list_calendars() {
    echo ""
    echo "Test: Calendar - List Calendars"

    local params
    params=$(cat <<EOF
{
    "operation": "list_calendars",
    "calendar_url": "$CALDAV_URL"
}
EOF
)

    local result
    result=$(rpc_call "skills.execute" "{\"skill_name\":\"calendar\",\"params\":$params}")

    if [[ -z "$result" ]]; then
        log_result "calendar_list_calendars" "false" "No response from bridge"
        return 1
    fi


    if echo "$result" | grep -q '"operation":"list_calendars"'; then
        log_result "calendar_list_calendars" "true" "Calendars listed successfully"
        return 0
    else
        log_result "calendar_list_calendars" "false" "Failed to list calendars: $result"
        return 1
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    echo "========================================"
    echo "US-7 Calendar E2E Test Suite"
    echo "========================================"
    echo ""
    echo "NOTE: This test uses the mock CalDAV client built into the calendar skill."
    echo "      No external CalDAV server (Radicale) is required."
    echo ""

    setup_test_env || exit 1

    check_dependencies || exit 1


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
        echo -e "${RED}✗ Failed to start bridge${NC}"
        exit 1
    }

    wait_for_bridge 30 || {
        echo -e "${RED}✗ Bridge socket not ready${NC}"
        stop_bridge
        exit 1
    }

    echo ""
    echo "Running calendar tests..."
    echo ""


    test_calendar_list_calendars
    test_calendar_create_event
    test_calendar_get_events
    test_calendar_verify_event
    test_calendar_conflict_detection
    test_calendar_update_event
    test_calendar_delete_event


    stop_bridge || {
        echo -e "${YELLOW}⚠ Warning: Failed to stop bridge gracefully${NC}"
    }

    test_summary
    exit $?
}

main "$@"
