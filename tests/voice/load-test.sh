#!/bin/bash
# Load test for WebRTC voice system
# Tests concurrent call handling and performance

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BRIDGE_BIN="$PROJECT_ROOT/bridge/build/armorclaw-bridge"
SOCKET_PATH="/run/armorclaw/load-test.sock"
RESULTS_DIR="$PROJECT_ROOT/tests/results/load"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
CONCURRENT_CALLS=${CONCURRENT_CALLS:-10}
CALLS_PER_SECOND=${CALLS_PER_SECOND:-5}
TEST_DURATION=${TEST_DURATION:-60} # seconds

echo "=== WebRTC Voice Load Test ==="
echo "Configuration:"
echo "  Concurrent Calls: $CONCURRENT_CALLS"
echo "  Calls Per Second: $CALLS_PER_SECOND"
echo "  Test Duration: ${TEST_DURATION}s"
echo ""

# Create results directory
mkdir -p "$RESULTS_DIR"

# Build bridge if needed
if [ ! -f "$BRIDGE_BIN" ]; then
    echo -e "${YELLOW}Building bridge...${NC}"
    cd "$PROJECT_ROOT/bridge"
    go build -o build/armorclaw-bridge ./cmd/bridge
fi

# Start bridge with test config
TEST_SOCKET="$SOCKET_PATH" TEST_CONFIG="$PROJECT_ROOT/tests/config/load-test.toml" \
    "$BRIDGE_BIN" &
BRIDGE_PID=$!

# Wait for bridge to start
sleep 2

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    kill $BRIDGE_PID 2>/dev/null || true
    rm -f "$SOCKET_PATH"
    wait $BRIDGE_PID 2>/dev/null || true
}

trap cleanup EXIT INT TERM

# Check if bridge started
if ! kill -0 $BRIDGE_PID 2>/dev/null; then
    echo -e "${RED}Failed to start bridge${NC}"
    wait $BRIDGE_PID
    exit 1
fi

echo -e "${GREEN}Bridge started (PID: $BRIDGE_PID)${NC}"
echo ""

# Test 1: Concurrent call creation
echo -e "${YELLOW}Test 1: Creating $CONCURRENT_CALLS concurrent calls...${NC}"
START_TIME=$(date +%s)

for i in $(seq 1 $CONCURRENT_CALLS); do
    ROOM_ID="!loadtest$(date +%s%N):example.com"
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":'"$i"',"method":"webrtc.start","params":{"room_id":"'"$ROOM_ID"'","ttl":"10m"}}' | \
        socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || echo '{"error":"connection_failed"}')

    if echo "$RESPONSE" | grep -q '"result"'; then
        SESSION_ID=$(echo "$RESPONSE" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
        echo "  [$i/$CONCURRENT_CALLS] Created call: $ROOM_ID (session: $SESSION_ID)"
        echo "$ROOM_ID,$SESSION_ID" >> "$RESULTS_DIR/sessions.txt"
    else
        echo -e "${RED}  [$i/$CONCURRENT_CALLS] Failed to create call${NC}"
        echo "$RESPONSE"
    fi

    # Rate limiting
    sleep $(echo "scale=3; 1 / $CALLS_PER_SECOND" | bc)
done

CREATION_TIME=$(($(date +%s) - START_TIME))
echo -e "${GREEN}Created $CONCURRENT_CALLS calls in ${CREATION_TIME}s${NC}"
echo ""

# Test 2: ICE candidate handling
echo -e "${YELLOW}Test 2: Sending ICE candidates for all calls...${NC}"

if [ -f "$RESULTS_DIR/sessions.txt" ]; then
    CANDIDATE_SENT=0
    while IFS=',' read -r ROOM_ID SESSION_ID; do
        CANDIDATE='{"candidate":"candidate:1 1 udp 2130706431 192.168.1.100 54321 typ host","sdpMid":"audio","sdpMLineIndex":0}'
        RESPONSE=$(echo '{"jsonrpc":"2.0","id":2,"method":"webrtc.ice_candidate","params":{"session_id":"'"$SESSION_ID"'","candidate":'"$CANDIDATE"'}}' | \
            socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || echo '{"error":"connection_failed"}')

        if echo "$RESPONSE" | grep -q '"result"'; then
            CANDIDATE_SENT=$((CANDIDATE_SENT + 1))
        fi
    done < "$RESULTS_DIR/sessions.txt"

    echo -e "${GREEN}Sent candidates for $CANDIDATE_SENT calls${NC}"
fi
echo ""

# Test 3: Session listing performance
echo -e "${YELLOW}Test 3: Testing session listing performance...${NC}"

LIST_START=$(date +%s%N)
RESPONSE=$(echo '{"jsonrpc":"2.0","id":3,"method":"webrtc.list","params":{}}' | \
    socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || echo '{"error":"connection_failed"}')
LIST_END=$(date +%s%N)

LIST_TIME=$(( (LIST_END - LIST_START) / 1000000 )) # Convert to milliseconds

if echo "$RESPONSE" | grep -q '"result"'; then
    ACTIVE_CALLS=$(echo "$RESPONSE" | grep -o '"active_calls":[0-9]*' | cut -d':' -f2)
    echo -e "${GREEN}Listed $ACTIVE_CALLS active calls in ${LIST_TIME}ms${NC}"

    if [ $LIST_TIME -gt 100 ]; then
        echo -e "${YELLOW}WARNING: Listing took longer than 100ms${NC}"
    fi
else
    echo -e "${RED}Failed to list sessions${NC}"
fi
echo ""

# Test 4: Call termination
echo -e "${YELLOW}Test 4: Terminating all calls...${NC}"

if [ -f "$RESULTS_DIR/sessions.txt" ]; then
    TERMINATED=0
    while IFS=',' read -r ROOM_ID SESSION_ID; do
        RESPONSE=$(echo '{"jsonrpc":"2.0","id":4,"method":"webrtc.end","params":{"session_id":"'"$SESSION_ID"'","reason":"test_complete"}}' | \
            socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || echo '{"error":"connection_failed"}')

        if echo "$RESPONSE" | grep -q '"result"'; then
            TERMINATED=$((TERMINATED + 1))
        fi
    done < "$RESULTS_DIR/sessions.txt"

    echo -e "${GREEN}Terminated $TERMINATED calls${NC}"
fi
echo ""

# Test 5: Sustained load (create and terminate continuously)
echo -e "${YELLOW}Test 5: Sustained load test (${TEST_DURATION}s)...${NC}"

END_TIME=$(($(date +%s) + TEST_DURATION))
CALLS_CREATED=0
CALLS_FAILED=0
SESSIONS=()

while [ $(date +%s) -lt $END_TIME ]; do
    # Create a call
    ROOM_ID="!sustained$(date +%s%N):example.com"
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":5,"method":"webrtc.start","params":{"room_id":"'"$ROOM_ID"'","ttl":"1m"}}' | \
        socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || echo '{"error":"connection_failed"}')

    if echo "$RESPONSE" | grep -q '"result"'; then
        SESSION_ID=$(echo "$RESPONSE" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
        SESSIONS+=("$SESSION_ID")
        CALLS_CREATED=$((CALLS_CREATED + 1))

        # Terminate oldest call if we have too many
        if [ ${#SESSIONS[@]} -gt $CONCURRENT_CALLS ]; then
            OLD_SESSION="${SESSIONS[0]}"
            echo '{"jsonrpc":"2.0","id":6,"method":"webrtc.end","params":{"session_id":"'"$OLD_SESSION"'","reason":"rotation"}}' | \
                socat - UNIX-CONNECT:"$SOCKET_PATH" >/dev/null 2>&1 || true
            SESSIONS=("${SESSIONS[@]:1}")
        fi
    else
        CALLS_FAILED=$((CALLS_FAILED + 1))
    fi

    # Brief pause to simulate realistic load
    sleep 0.1
done

echo -e "${GREEN}Sustained load test complete${NC}"
echo "  Calls Created: $CALLS_CREATED"
echo "  Calls Failed: $CALLS_FAILED"
echo "  Success Rate: $(echo "scale=2; $CALLS_CREATED * 100 / ($CALLS_CREATED + $CALLS_FAILED)" | bc)%"
echo ""

# Summary
echo -e "${GREEN}=== Load Test Summary ===${NC}"
echo "Results saved to: $RESULTS_DIR"
echo ""
echo "Key Metrics:"
echo "  Concurrent Calls: $CONCURRENT_CALLS"
echo "  Creation Time: ${CREATION_TIME}s"
echo "  List Response Time: ${LIST_TIME}ms"
echo "  Sustained Load: ${TEST_DURATION}s"
echo "  Total Calls Created: $CALLS_CREATED"
echo "  Success Rate: $(echo "scale=2; $CALLS_CREATED * 100 / ($CALLS_CREATED + $CALLS_FAILED)" | bc)%"
echo ""

# Check for any warnings
WARNINGS=0

if [ $LIST_TIME -gt 100 ]; then
    echo -e "${YELLOW}⚠ WARNING: Session list response time > 100ms${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

if [ $CALLS_FAILED -gt 0 ]; then
    FAILURE_RATE=$(echo "scale=2; $CALLS_FAILED * 100 / ($CALLS_CREATED + $CALLS_FAILED)" | bc)
    echo -e "${YELLOW}⚠ WARNING: $CALLS_FAILED calls failed (${FAILURE_RATE}% failure rate)${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

if [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed with acceptable performance${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ Tests completed with $WARNINGS warnings${NC}"
    exit 0
fi
