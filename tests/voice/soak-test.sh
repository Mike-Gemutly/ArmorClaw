#!/bin/bash
# Soak test for WebRTC voice system
# Tests long-running stability and memory usage

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BRIDGE_BIN="$PROJECT_ROOT/bridge/build/armorclaw-bridge"
SOCKET_PATH="/run/armorclaw/soak-test.sock"
RESULTS_DIR="$PROJECT_ROOT/tests/results/soak"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DURATION=${TEST_DURATION:-3600} # 1 hour default
CALL_ROTATION=${CALL_ROTATION:-5}     # Rotate 5 active calls
SAMPLE_INTERVAL=${SAMPLE_INTERVAL:-60} # Sample metrics every 60s

echo "=== WebRTC Voice Soak Test ==="
echo "Configuration:"
echo "  Test Duration: ${TEST_DURATION}s ($(echo "scale=1; $TEST_DURATION / 60" | bc) minutes)"
echo "  Active Calls: $CALL_ROTATION"
echo "  Sample Interval: ${SAMPLE_INTERVAL}s"
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
TEST_SOCKET="$SOCKET_PATH" TEST_CONFIG="$PROJECT_ROOT/tests/config/soak-test.toml" \
    "$BRIDGE_BIN" &
BRIDGE_PID=$!

# Wait for bridge to start
sleep 2

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"

    # Terminate any active sessions
    if [ -f "$RESULTS_DIR/sessions.txt" ]; then
        echo "Terminating active sessions..."
        while IFS=',' read -r ROOM_ID SESSION_ID; do
            echo '{"jsonrpc":"2.0","id":99,"method":"webrtc.end","params":{"session_id":"'"$SESSION_ID"'","reason":"cleanup"}}' | \
                socat - UNIX-CONNECT:"$SOCKET_PATH" >/dev/null 2>&1 || true
        done < "$RESULTS_DIR/sessions.txt"
    fi

    kill $BRIDGE_PID 2>/dev/null || true
    rm -f "$SOCKET_PATH"
    wait $BRIDGE_PID 2>/dev/null || true

    # Generate final report
    if [ -f "$RESULTS_DIR/metrics.log" ]; then
        echo -e "${BLUE}Generating final report...${NC}"
        "$SCRIPT_DIR/generate-soak-report.sh" "$RESULTS_DIR/metrics.log" "$RESULTS_DIR/report.txt"
        echo -e "${GREEN}Report saved to: $RESULTS_DIR/report.txt${NC}"
    fi
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

# Function to get memory usage
get_memory() {
    ps -o rss= -p $BRIDGE_PID 2>/dev/null || echo "0"
}

# Function to get number of active sessions
get_active_sessions() {
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":1,"method":"webrtc.list","params":{}}' | \
        socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || echo '{"error":"failed"}')

    if echo "$RESPONSE" | grep -q '"result"'; then
        echo "$RESPONSE" | grep -o '"active_calls":[0-9]*' | cut -d':' -f2 || echo "0"
    else
        echo "0"
    fi
}

# Function to create a call
create_call() {
    ROOM_ID="!soak$(date +%s%N):example.com"
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":2,"method":"webrtc.start","params":{"room_id":"'"$ROOM_ID"'","ttl":"1h"}}' | \
        socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || echo '{"error":"failed"}')

    if echo "$RESPONSE" | grep -q '"result"'; then
        SESSION_ID=$(echo "$RESPONSE" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
        echo "$ROOM_ID,$SESSION_ID,$(date +%s)"
        return 0
    else
        return 1
    fi
}

# Function to terminate a call
terminate_call() {
    local SESSION_ID=$1
    echo '{"jsonrpc":"2.0","id":3,"method":"webrtc.end","params":{"session_id":"'"$SESSION_ID"'","reason":"soak_test_rotation"}}' | \
        socat - UNIX-CONNECT:"$SOCKET_PATH" >/dev/null 2>&1 || true
}

# Initialize metrics header
echo "timestamp,elapsed_seconds,memory_kb,active_sessions,calls_created,calls_failed,total_samples" > "$RESULTS_DIR/metrics.log"

# Initialize counters
CALLS_CREATED=0
CALLS_FAILED=0
TOTAL_SAMPLES=0
START_TIME=$(date +%s)
INITIAL_MEMORY=$(get_memory)

echo -e "${BLUE}Starting soak test...${NC}"
echo "Initial memory: ${INITIAL_MEMORY} KB"
echo ""

# Main soak test loop
NEXT_SAMPLE_TIME=0
SESSIONS=()

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))

    # Check if test duration exceeded
    if [ $ELAPSED -ge $TEST_DURATION ]; then
        echo -e "${YELLOW}Test duration reached${NC}"
        break
    fi

    # Time to sample metrics?
    if [ $CURRENT_TIME -ge $NEXT_SAMPLE_TIME ]; then
        CURRENT_MEMORY=$(get_memory)
        ACTIVE_SESSIONS=$(get_active_sessions)
        TOTAL_SAMPLES=$((TOTAL_SAMPLES + 1))

        # Calculate memory growth
        MEMORY_GROWTH=$((CURRENT_MEMORY - INITIAL_MEMORY))
        MEMORY_GROWTH_MB=$(echo "scale=2; $MEMORY_GROWTH / 1024" | bc)

        # Log metrics
        echo "$(date +%s),$ELAPSED,$CURRENT_MEMORY,$ACTIVE_SESSIONS,$CALLS_CREATED,$CALLS_FAILED,$TOTAL_SAMPLES" >> "$RESULTS_DIR/metrics.log"

        # Display progress
        PROGRESS=$(echo "scale=1; $ELAPSED * 100 / $TEST_DURATION" | bc)
        echo -ne "\r${BLUE}[${PROGRESS}%]${NC} Elapsed: ${ELAPSED}s | Memory: ${CURRENT_MEMORY} KB (+${MEMORY_GROWTH_MB} MB) | Active: $ACTIVE_SESSIONS | Created: $CALLS_CREATED | Failed: $CALLS_FAILED"

        # Check for memory leaks (growth > 100MB)
        if [ $MEMORY_GROWTH -gt 102400 ]; then
            echo -e "\n${YELLOW}WARNING: Memory growth > 100 MB${NC}"
        fi

        # Schedule next sample
        NEXT_SAMPLE_TIME=$((CURRENT_TIME + SAMPLE_INTERVAL))
    fi

    # Maintain target number of active sessions
    while [ ${#SESSIONS[@]} -lt $CALL_ROTATION ]; do
        RESULT=$(create_call)
        if [ $? -eq 0 ]; then
            SESSIONS+=("$RESULT")
            CALLS_CREATED=$((CALLS_CREATED + 1))
        else
            CALLS_FAILED=$((CALLS_FAILED + 1))
            break
        fi
    done

    # Rotate sessions (terminate oldest, create new)
    if [ ${#SESSIONS[@]} -ge $CALL_ROTATION ] && [ $((ELAPSED % 300)) -eq 0 ]; then
        # Every 5 minutes, rotate one session
        OLD_SESSION="${SESSIONS[0]}"
        SESSION_ID=$(echo "$OLD_SESSION" | cut -d',' -f2)

        terminate_call "$SESSION_ID"

        # Remove from tracking
        SESSIONS=("${SESSIONS[@]:1}")

        # Remove from sessions file
        if [ -f "$RESULTS_DIR/sessions.txt" ]; then
            grep -v ",$SESSION_ID," "$RESULTS_DIR/sessions.txt" > "$RESULTS_DIR/sessions.tmp" || true
            mv "$RESULTS_DIR/sessions.tmp" "$RESULTS_DIR/sessions.txt"
        fi

        # Create replacement
        RESULT=$(create_call)
        if [ $? -eq 0 ]; then
            SESSIONS+=("$RESULT")
            echo "$RESULT" >> "$RESULTS_DIR/sessions.txt"
        fi
    fi

    # Add new sessions to tracking file
    for SESSION in "${SESSIONS[@]}"; do
        SESSION_ID=$(echo "$SESSION" | cut -d',' -f2)
        if ! grep -q ",$SESSION_ID," "$RESULTS_DIR/sessions.txt" 2>/dev/null; then
            echo "$SESSION" >> "$RESULTS_DIR/sessions.txt"
        fi
    done

    # Brief sleep to prevent busy-waiting
    sleep 1
done

echo ""
echo -e "${GREEN}Soak test completed${NC}"
echo ""

# Calculate final statistics
FINAL_MEMORY=$(get_memory)
MEMORY_GROWTH=$((FINAL_MEMORY - INITIAL_MEMORY))
MEMORY_GROWTH_MB=$(echo "scale=2; $MEMORY_GROWTH / 1024" | bc)

# Get min/max memory from metrics
MIN_MEMORY=$(awk -F',' 'NR>1 {print $3}' "$RESULTS_DIR/metrics.log" | sort -n | head -1)
MAX_MEMORY=$(awk -F',' 'NR>1 {print $3}' "$RESULTS_DIR/metrics.log" | sort -n | tail -1)

echo -e "${BLUE}=== Soak Test Statistics ===${NC}"
echo "Duration: $(echo "scale=1; $TEST_DURATION / 60" | bc) minutes"
echo "Samples: $TOTAL_SAMPLES"
echo ""
echo "Memory Usage:"
echo "  Initial: ${INITIAL_MEMORY} KB"
echo "  Final:   ${FINAL_MEMORY} KB"
echo "  Min:     ${MIN_MEMORY} KB"
echo "  Max:     ${MAX_MEMORY} KB"
echo "  Growth:  ${MEMORY_GROWTH_MB} MB"
echo ""
echo "Calls:"
echo "  Created: $CALLS_CREATED"
echo "  Failed:  $CALLS_FAILED"
if [ $((CALLS_CREATED + CALLS_FAILED)) -gt 0 ]; then
    SUCCESS_RATE=$(echo "scale=2; $CALLS_CREATED * 100 / ($CALLS_CREATED + $CALLS_FAILED)" | bc)
    echo "  Success Rate: ${SUCCESS_RATE}%"
fi
echo ""

# Check for issues
ISSUES=0

if [ $MEMORY_GROWTH -gt 102400 ]; then
    echo -e "${RED}✗ FAIL: Memory growth > 100 MB (potential leak)${NC}"
    ISSUES=$((ISSUES + 1))
elif [ $MEMORY_GROWTH -gt 51200 ]; then
    echo -e "${YELLOW}⚠ WARNING: Memory growth > 50 MB${NC}"
    ISSUES=$((ISSUES + 1))
fi

if [ $CALLS_FAILED -gt $((CALLS_CREATED / 10)) ]; then
    echo -e "${RED}✗ FAIL: Failure rate > 10%${NC}"
    ISSUES=$((ISSUES + 1))
fi

if [ $ISSUES -eq 0 ]; then
    echo -e "${GREEN}✓ All stability checks passed${NC}"
    exit 0
else
    echo -e "${RED}✗ Test completed with $ISSUES issues${NC}"
    exit 1
fi
