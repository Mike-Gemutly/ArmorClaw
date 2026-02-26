#!/bin/bash
# ArmorClaw Setup Log Viewer
# View the ArmorClaw setup log from inside or outside the container
# Version: 0.1.0

LOG_FILE="/var/log/armorclaw/setup.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

show_usage() {
    echo "ArmorClaw Setup Log Viewer"
    echo ""
    echo "Usage: view-setup-log [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --tail N       Show last N lines (default: 50)"
    echo "  --errors       Show only ERROR and CRITICAL lines"
    echo "  --full         Show entire log file"
    echo "  --follow       Follow log output (like tail -f)"
    echo "  --help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  # View last 50 lines (default)"
    echo "  view-setup-log"
    echo ""
    echo "  # View last 100 lines"
    echo "  view-setup-log --tail 100"
    echo ""
    echo "  # Show only errors"
    echo "  view-setup-log --errors"
    echo ""
    echo "  # Follow log in real-time"
    echo "  view-setup-log --follow"
    echo ""
    echo "From outside container:"
    echo "  docker exec armorclaw view-setup-log --errors"
    echo "  docker cp armorclaw:/var/log/armorclaw/setup.log ./setup.log"
}

# Check if log file exists
check_log_file() {
    if [ ! -f "$LOG_FILE" ]; then
        echo -e "${YELLOW}No setup log found at $LOG_FILE${NC}"
        echo ""
        echo "This could mean:"
        echo "  1. Setup has not been run yet"
        echo "  2. The log file was deleted"
        echo "  3. You're not running inside the ArmorClaw container"
        echo ""
        echo "To copy log from container:"
        echo "  docker cp armorclaw:/var/log/armorclaw/setup.log ./setup.log"
        exit 1
    fi
}

# Show log statistics
show_stats() {
    local total_lines
    local error_count
    local warning_count
    local critical_count

    total_lines=$(wc -l < "$LOG_FILE" 2>/dev/null || echo "0")
    error_count=$(grep -c '\[ERROR\]' "$LOG_FILE" 2>/dev/null || echo "0")
    warning_count=$(grep -c '\[WARN\]' "$LOG_FILE" 2>/dev/null || echo "0")
    critical_count=$(grep -c '\[CRITICAL\]' "$LOG_FILE" 2>/dev/null || echo "0")

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}Setup Log Statistics${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "  Log file:    $LOG_FILE"
    echo "  Total lines: $total_lines"
    echo ""
    echo -e "  ${RED}Errors:     $error_count${NC}"
    echo -e "  ${YELLOW}Warnings:   $warning_count${NC}"
    echo -e "  ${RED}Critical:   $critical_count${NC}"
    echo ""
}

# Main logic
case "${1:-}" in
    --help|-h)
        show_usage
        exit 0
        ;;
    --tail)
        check_log_file
        local lines="${2:-50}"
        echo -e "${CYAN}Last $lines lines of setup log:${NC}"
        echo ""
        tail -n "$lines" "$LOG_FILE"
        ;;
    --errors)
        check_log_file
        echo -e "${CYAN}Error and critical messages from setup log:${NC}"
        echo ""
        grep -E '\[ERROR\]|\[CRITICAL\]' "$LOG_FILE" 2>/dev/null || echo "(no errors found)"
        ;;
    --full)
        check_log_file
        show_stats
        echo -e "${CYAN}Full setup log:${NC}"
        echo ""
        cat "$LOG_FILE"
        ;;
    --follow|-f)
        check_log_file
        echo -e "${CYAN}Following setup log (Ctrl+C to stop):${NC}"
        echo ""
        tail -f "$LOG_FILE"
        ;;
    --stats)
        check_log_file
        show_stats
        ;;
    *)
        check_log_file
        show_stats
        echo -e "${CYAN}Last 50 lines of setup log:${NC}"
        echo -e "${CYAN}(use --tail N for more, --full for entire log)${NC}"
        echo ""
        tail -50 "$LOG_FILE"
        ;;
esac
