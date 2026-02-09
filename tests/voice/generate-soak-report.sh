#!/bin/bash
# Generate soak test report from metrics log

METRICS_FILE=$1
OUTPUT_FILE=$2

if [ -z "$METRICS_FILE" ] || [ -z "$OUTPUT_FILE" ]; then
    echo "Usage: $0 <metrics_file> <output_file>"
    exit 1
fi

if [ ! -f "$METRICS_FILE" ]; then
    echo "Metrics file not found: $METRICS_FILE"
    exit 1
fi

# Read header
HEADER=$(head -1 "$METRICS_FILE")

# Calculate statistics
TOTAL_SAMPLES=$(tail -n +2 "$METRICS_FILE" | wc -l)

if [ $TOTAL_SAMPLES -eq 0 ]; then
    echo "No samples found in metrics file"
    exit 1
fi

# Memory stats
MEMORY_VALUES=$(tail -n +2 "$METRICS_FILE" | cut -d',' -f3)
MEMORY_MIN=$(echo "$MEMORY_VALUES" | sort -n | head -1)
MEMORY_MAX=$(echo "$MEMORY_VALUES" | sort -n | tail -1)
MEMORY_AVG=$(echo "$MEMORY_VALUES" | awk '{sum+=$1; count++} END {print int(sum/count)}')
MEMORY_INITIAL=$(tail -n +2 "$METRICS_FILE" | head -1 | cut -d',' -f3)
MEMORY_FINAL=$(tail -n +2 "$METRICS_FILE" | tail -1 | cut -d',' -f3)
MEMORY_GROWTH=$((MEMORY_FINAL - MEMORY_INITIAL))
MEMORY_GROWTH_MB=$(echo "scale=2; $MEMORY_GROWTH / 1024" | bc)

# Duration stats
DURATION_VALUES=$(tail -n +2 "$METRICS_FILE" | cut -d',' -f2)
DURATION_MAX=$(echo "$DURATION_VALUES" | sort -n | tail -1)
DURATION_MIN=$(echo "$DURATION_VALUES" | sort -n | head -1)

# Session stats
SESSION_VALUES=$(tail -n +2 "$METRICS_FILE" | cut -d',' -f4)
SESSION_AVG=$(echo "$SESSION_VALUES" | awk '{sum+=$1; count++} END {print int(sum/count)}')
SESSION_MAX=$(echo "$SESSION_VALUES" | sort -n | tail -1)

# Call stats
CALLS_CREATED=$(tail -n +2 "$METRICS_FILE" | tail -1 | cut -d',' -f5)
CALLS_FAILED=$(tail -n +2 "$METRICS_FILE" | tail -1 | cut -d',' -f6)

# Generate report
cat > "$OUTPUT_FILE" << EOF
WebRTC Voice Soak Test Report
===============================
Generated: $(date)

Test Duration
-------------
Start Time: $(date -d @$(tail -n +2 "$METRICS_FILE" | head -1 | cut -d',' -f1))
End Time:   $(date -d @$(tail -n +2 "$METRICS_FILE" | tail -1 | cut -d',' -f1))
Total:      ${DURATION_MAX}s ($(echo "scale=1; $DURATION_MAX / 60" | bc) minutes)
Samples:    $TOTAL_SAMPLES

Memory Usage
-----------
Initial:   ${MEMORY_INITIAL} KB
Final:     ${MEMORY_FINAL} KB
Minimum:   ${MEMORY_MIN} KB
Maximum:   ${MEMORY_MAX} KB
Average:   ${MEMORY_AVG} KB
Growth:    ${MEMORY_GROWTH} KB (${MEMORY_GROWTH_MB} MB)

Active Sessions
--------------
Average: $SESSION_AVG
Maximum: $SESSION_MAX

Call Statistics
--------------
Created: $CALLS_CREATED
Failed:  $CALLS_FAILED
EOF

if [ $((CALLS_CREATED + CALLS_FAILED)) -gt 0 ]; then
    SUCCESS_RATE=$(echo "scale=2; $CALLS_CREATED * 100 / ($CALLS_CREATED + $CALLS_FAILED)" | bc)
    echo "Success Rate: ${SUCCESS_RATE}%" >> "$OUTPUT_FILE"
fi

# Add recommendations
cat >> "$OUTPUT_FILE" << EOF

Recommendations
--------------
EOF

# Check memory growth
if [ $MEMORY_GROWTH -gt 102400 ]; then
    echo "⚠ WARNING: Memory growth exceeds 100 MB. Investigate potential memory leak." >> "$OUTPUT_FILE"
elif [ $MEMORY_GROWTH -gt 51200 ]; then
    echo "⚠ Memory growth is > 50 MB. Monitor for leaks." >> "$OUTPUT_FILE"
else
    echo "✓ Memory usage is stable." >> "$OUTPUT_FILE"
fi

# Check failure rate
if [ $CALLS_FAILED -gt $((CALLS_CREATED / 10)) ]; then
    echo "✗ High failure rate detected. Review logs for errors." >> "$OUTPUT_FILE"
else
    echo "✓ Call failure rate is acceptable." >> "$OUTPUT_FILE"
fi

# Check session stability
if [ $SESSION_MAX -gt $((SESSION_AVG * 2)) ]; then
    echo "⚠ High variance in active sessions. Check for session leaks." >> "$OUTPUT_FILE"
else
    echo "✓ Session count is stable." >> "$OUTPUT_FILE"
fi

echo "" >> "$OUTPUT_FILE"
echo "Raw Data" >> "$OUTPUT_FILE"
echo "--------" >> "$OUTPUT_FILE"
echo "$HEADER" >> "$OUTPUT_FILE"
tail -5 "$METRICS_FILE" >> "$OUTPUT_FILE"
echo "..." >> "$OUTPUT_FILE"

echo "Report generated: $OUTPUT_FILE"
