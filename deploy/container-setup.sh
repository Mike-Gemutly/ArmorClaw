#!/bin/bash
# ArmorClaw Container Setup Wizard
# Simplified setup for Docker container deployment
# Version: 0.4.1

# NOTE: Do NOT use set -e here. Transient failures (curl timeouts, docker pulls)
# must not kill the setup wizard. Critical commands check errors explicitly.

# Colors for output (only if terminal supports them)
# Detect color support - avoid ANSI codes when terminal doesn't support them
detect_color_support() {
    if [ -t 1 ] && command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null)" -ge 8 ]; then
        COLOR_SUPPORTED=true
    else
        COLOR_SUPPORTED=false
    fi
}

# Initialize color detection
detect_color_support

if [ "$COLOR_SUPPORTED" = true ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    CYAN=''
    BOLD=''
    NC=''
fi

# Paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
RUN_DIR="/run/armorclaw"
LOG_DIR="/var/log/armorclaw"
LOG_FILE="$LOG_DIR/setup.log"
CONFIG_FILE="$CONFIG_DIR/config.toml"
SETUP_FLAG="$CONFIG_DIR/.setup_complete"

# Debug mode (set ARMORCLAW_DEBUG=true to enable)
DEBUG="${ARMORCLAW_DEBUG:-false}"

# Profile defaults
DEPLOY_PROFILE="quick"
HIPAA_ENABLED="false"
QUARANTINE_ENABLED="false"
AUDIT_RETENTION_DAYS=90
COMPLIANCE_PATTERNS_PII="false"
COMPLIANCE_PATTERNS_PHI="false"
MATRIX_AVAILABLE=false

# Track if setup succeeded (for cleanup trap)
SETUP_SUCCEEDED=false

#=============================================================================
# Terminal Handling (Critical for Go wizard crash recovery)
#=============================================================================

# Full terminal reset - call this before ANY interactive input
# This is critical because the Go TUI wizard (huh) may crash and leave
# the terminal in raw mode, which breaks bash read/input handling.
reset_terminal_full() {
    # Only attempt terminal reset if we have a TTY
    if [ -t 0 ]; then
        # Step 1: Restore all terminal modes to sane defaults
        # This MUST be done first before any other terminal operations
        stty sane 2>/dev/null || true

        # Step 2: Explicitly re-enable critical modes that may be disabled
        # - echo: show typed characters
        # - icanon: enable canonical mode (line buffering)
        # - isig: enable interrupt/suspend/quit characters
        # - ixon: enable XON/XOFF flow control
        stty echo icanon isig ixon 2>/dev/null || true

        # Step 3: Flush any pending input in the buffer
        # This prevents garbage input from previous raw mode
        stty min 1 time 0 2>/dev/null || true

        # Step 4: Reset terminal rendering state using tput
        if command -v tput >/dev/null 2>&1; then
            tput sgr0 2>/dev/null || true    # Reset all attributes (bold, colors, etc.)
            tput cnorm 2>/dev/null || true   # Show cursor
            tput cr 2>/dev/null || true      # Move cursor to beginning of line
            tput ed 2>/dev/null || true      # Clear to end of screen
        fi

        # Step 5: Force terminal to process pending output
        # This ensures all reset sequences are processed before we continue
        command -v sync >/dev/null 2>&1 && sync 2>/dev/null || true

        # Step 6: Small delay to let terminal settle
        # This is critical - some terminals need time to process reset
        sleep 0.1 2>/dev/null || sleep 1 2>/dev/null || true

        # Step 7: Re-detect color support after reset
        detect_color_support
    fi
}

#=============================================================================
# Logging Functions (Phase 1: Persistent Log File System)
#=============================================================================

# Initialize log file - ensures directory exists and file is writable
init_logging() {
    mkdir -p "$LOG_DIR" 2>/dev/null || true
    touch "$LOG_FILE" 2>/dev/null || true
    chmod 644 "$LOG_FILE" 2>/dev/null || true
}

# Core logging function - writes to log file with timestamp and level
log_setup() {
    local level="$1"
    shift
    local message="$*"
    local timestamp
    timestamp=$(date -Iseconds 2>/dev/null || date '+%Y-%m-%dT%H:%M:%S%z')

    # Write to log file (always, if possible)
    if [ -w "$LOG_FILE" ] || [ -w "$LOG_DIR" ]; then
        echo "[$timestamp] [$level] $message" >> "$LOG_FILE" 2>/dev/null || true
    fi
}

# Log levels - write to both log file and stdout/stderr
log_info() {
    log_setup "INFO" "$@"
    print_info "$@"
}

log_success() {
    log_setup "INFO" "SUCCESS: $*"
    print_success "$@"
}

log_error() {
    log_setup "ERROR" "$@"
    print_error "$@"
}

log_warning() {
    log_setup "WARN" "$@"
    print_warning "$@"
}

log_debug() {
    if [ "$DEBUG" = true ]; then
        log_setup "DEBUG" "$@"
        echo -e "${BLUE}[DEBUG] $*${NC}" >&2
    fi
}

log_critical() {
    log_setup "CRITICAL" "$@"
    echo -e "${RED}[CRITICAL] $*${NC}" >&2
}

# Log command execution (for debug mode)
log_command() {
    log_debug "Executing: $*"
}

# Log section marker
log_section() {
    local section="$1"
    log_setup "INFO" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_setup "INFO" "SECTION: $section"
    log_setup "INFO" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

#=============================================================================
# Error Codes and Summary (Phase 4: Error Summary & Recovery Suggestions)
#=============================================================================

# Installation error codes (INS-XXX)
# INS-001: Docker socket not accessible
# INS-002: Huh? wizard crashed (terminal incompatibility)
# INS-003: Configuration write failed (permission denied)
# INS-004: Matrix connection failed
# INS-005: API key validation failed
# INS-006: Required tool missing
# INS-007: Conduit homeserver not ready
# INS-008: User registration failed
# INS-009: Bridge room creation failed

# Set error code and message for crash handler
set_error() {
    ERROR_CODE="$1"
    ERROR_MESSAGE="$2"
    log_error "[$ERROR_CODE] $ERROR_MESSAGE"
}

# Show error summary with recovery suggestions
show_error_summary() {
    local error_code="$1"
    local error_message="$2"

    echo ""
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}Error Code: $error_code${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${YELLOW}$error_message${NC}"
    echo ""
    echo -e "${CYAN}Suggested Fix:${NC}"

    case "$error_code" in
        INS-001)
            echo "  The Docker socket is not accessible."
            echo ""
            echo "  Solutions:"
            echo "    1. Run with --user root: docker run --user root ..."
            echo "    2. Add user to docker group: sudo usermod -aG docker \$USER"
            echo "    3. Fix socket permissions: sudo chmod 666 /var/run/docker.sock"
            ;;
        INS-002)
            echo "  The Huh? TUI wizard crashed (terminal incompatibility)."
            echo ""
            echo "  Solutions:"
            echo "    1. Try a different terminal or SSH client"
            echo "    2. Use environment variables instead:"
            echo "       -e ARMORCLAW_SERVER_NAME=your-ip"
            echo "       -e ARMORCLAW_API_KEY=sk-your-key"
            echo "       -e ARMORCLAW_PROFILE=quick"
            ;;
        INS-003)
            echo "  Configuration write failed (permission denied)."
            echo ""
            echo "  Solutions:"
            echo "    1. Check volume permissions: docker volume inspect armorclaw-data"
            echo "    2. Ensure sufficient disk space: df -h"
            echo "    3. Run with --user root if needed"
            ;;
        INS-004)
            echo "  Matrix connection failed."
            echo ""
            echo "  Solutions:"
            echo "    1. Check if Conduit is running: docker ps | grep conduit"
            echo "    2. Check Conduit logs: docker logs armorclaw-conduit"
            echo "    3. Verify network connectivity: curl http://localhost:6167/_matrix/client/versions"
            ;;
        INS-005)
            echo "  API key validation failed."
            echo ""
            echo "  Solutions:"
            echo "    1. Verify API key format (OpenAI: sk-*, Anthropic: sk-ant-*)"
            echo "    2. Check API key is at least 20 characters"
            echo "    3. Test key manually: curl https://api.openai.com/v1/models -H 'Authorization: Bearer sk-...'"
            ;;
        INS-006)
            echo "  Required tool missing."
            echo ""
            echo "  Solutions:"
            echo "    1. Install missing tools: apt-get install -y openssl jq socat curl docker.io"
            echo "    2. Ensure the container image is complete"
            ;;
        INS-007)
            echo "  Conduit homeserver did not become ready in time."
            echo ""
            echo "  Solutions:"
            echo "    1. Check Conduit logs: docker logs armorclaw-conduit"
            echo "    2. Increase wait time and retry"
            echo "    3. Run create-matrix-admin.sh manually after Conduit is ready"
            ;;
        INS-008)
            echo "  Matrix user registration failed."
            echo ""
            echo "  Solutions:"
            echo "    1. Check if Conduit is healthy"
            echo "    2. Verify registration_shared_secret in conduit.toml"
            echo "    3. Run create-matrix-admin.sh manually: /opt/armorclaw/create-matrix-admin.sh <username>"
            ;;
        INS-009)
            echo "  Bridge room creation failed."
            echo ""
            echo "  Solutions:"
            echo "    1. Create the room manually in Element X"
            echo "    2. Invite the bridge bot: @bridge:<server_name>"
            echo "    3. Send !status to verify connection"
            ;;
        *)
            echo "  Check $LOG_FILE for detailed error information"
            ;;
    esac
    echo ""
}

# Cleanup trap - only runs on failure
cleanup_on_failure() {
    local exit_code=$?
    if [ "$SETUP_SUCCEEDED" != true ] && [ $exit_code -ne 0 ]; then
        log_critical "Setup failed with exit code: $exit_code"

        echo ""
        echo -e "${YELLOW}Setup failed. Cleaning up partial state...${NC}"

        # Clean up wizard JSON if it exists
        rm -f /tmp/armorclaw-wizard.json 2>/dev/null || true

        # Reset terminal state (in case TUI left it in raw mode)
        stty sane 2>/dev/null || true
        tput cnorm 2>/dev/null || true  # Show cursor
        tput sgr0 2>/dev/null || true   # Reset colors

        # Show error summary if we have an error code
        if [ -n "${ERROR_CODE:-}" ]; then
            show_error_summary "$ERROR_CODE" "${ERROR_MESSAGE:-Unknown error}"
        fi

        # Show log location
        if [ -f "$LOG_FILE" ]; then
            echo ""
            echo -e "${CYAN}Log file saved to: $LOG_FILE${NC}"
            echo ""
            echo "To view the log:"
            echo "  docker cp armorclaw:$LOG_FILE ./setup.log"
            echo ""
            echo "Last 20 lines:"
            tail -20 "$LOG_FILE" 2>/dev/null || echo "(no log available)"
            echo ""
        fi
    fi
    exit $exit_code
}
trap cleanup_on_failure INT TERM EXIT

#=============================================================================
# Preflight Checks (Verify system readiness before setup)
#=============================================================================

# Progress tracking
TOTAL_STEPS=0
CURRENT_STEP=0

preflight_checks() {
    log_section "Preflight Checks"
    print_info "Running preflight checks..."
    echo ""

    local checks_passed=0
    local checks_failed=0

    # Check 1: Docker daemon is responsive (with detailed diagnostics)
    printf "  [1/4] Docker daemon... "

    # First check socket file exists
    if [ ! -S /var/run/docker.sock ]; then
        printf "${RED}FAILED${NC}\n"
        printf "        Docker socket not found at /var/run/docker.sock\n"
        printf "        Mount with: -v /var/run/docker.sock:/var/run/docker.sock\n"
        set_error "INS-001" "Docker socket not mounted"
        ((checks_failed++)) || true
    # Check socket is readable
    elif [ ! -r /var/run/docker.sock ]; then
        printf "${RED}FAILED${NC}\n"
        printf "        Docker socket not readable (permission denied)\n"
        printf "        Current user: $(id 2>/dev/null || echo 'unknown')\n"
        printf "        Socket perms: $(ls -la /var/run/docker.sock 2>/dev/null || echo 'unknown')\n"
        printf "        Run with --user root or fix socket permissions\n"
        set_error "INS-001" "Docker socket permission denied"
        ((checks_failed++)) || true
    # Check socket is writable
    elif [ ! -w /var/run/docker.sock ]; then
        printf "${RED}FAILED${NC}\n"
        printf "        Docker socket not writable (permission denied)\n"
        printf "        Current user: $(id 2>/dev/null || echo 'unknown')\n"
        printf "        Socket perms: $(ls -la /var/run/docker.sock 2>/dev/null || echo 'unknown')\n"
        printf "        Run with --user root or fix socket permissions\n"
        set_error "INS-001" "Docker socket not writable"
        ((checks_failed++)) || true
    # Try to communicate with Docker daemon
    else
        local docker_error
        docker_error=$(docker info 2>&1)
        if [ $? -eq 0 ]; then
            printf "${GREEN}OK${NC}\n"
            ((checks_passed++)) || true
        else
            printf "${RED}FAILED${NC}\n"
            printf "        Docker daemon not responding\n"
            printf "        Socket exists but daemon communication failed:\n"
            # Show first 3 lines of error for diagnosis
            echo "$docker_error" | head -3 | while read -r line; do
                printf "        ${YELLOW}%s${NC}\n" "$line"
            done
            printf "\n"
            printf "        Possible causes:\n"
            printf "          1. Docker daemon not running on host\n"
            printf "          2. Docker version mismatch (client vs daemon)\n"
            printf "          3. SELinux/AppArmor blocking access\n"
            printf "\n"
            printf "        Debug commands:\n"
            printf "          # On host: systemctl status docker\n"
            printf "          # On host: docker info\n"
            set_error "INS-001" "Docker daemon not responsive"
            ((checks_failed++)) || true
        fi
    fi

    # Check 2: Network connectivity (can reach Docker Hub)
    printf "  [2/4] Network connectivity... "
    if curl -s --max-time 5 https://registry-1.docker.io/v2/ >/dev/null 2>&1; then
        printf "${GREEN}OK${NC}\n"
        ((checks_passed++)) || true
    else
        printf "${YELLOW}WARNING${NC}\n"
        printf "        Cannot reach Docker registry (may affect image pulls)\n"
        # Don't fail - network might work for other registries
        ((checks_passed++)) || true
    fi

    # Check 3: DNS resolution
    printf "  [3/4] DNS resolution... "
    if host google.com >/dev/null 2>&1 || nslookup google.com >/dev/null 2>&1; then
        printf "${GREEN}OK${NC}\n"
        ((checks_passed++)) || true
    else
        printf "${YELLOW}WARNING${NC}\n"
        printf "        DNS resolution may be slow (SSL cert generation might fail)\n"
        ((checks_passed++)) || true
    fi

    # Check 4: Disk space (need at least 2GB free)
    printf "  [4/4] Disk space... "
    local free_space
    free_space=$(df -BG /var 2>/dev/null | awk 'NR==2 {gsub(/G/,"",$4); print $4}')
    if [ -n "$free_space" ] && [ "$free_space" -ge 2 ]; then
        printf "${GREEN}OK${NC} (${free_space}GB available)\n"
        ((checks_passed++)) || true
    else
        printf "${YELLOW}WARNING${NC}\n"
        printf "        Low disk space (need at least 2GB)\n"
        ((checks_passed++)) || true
    fi

    echo ""

    if [ $checks_failed -gt 0 ]; then
        print_error "Preflight checks failed ($checks_failed of 4)"
        return 1
    fi

    print_success "Preflight checks passed ($checks_passed/4)"
    echo ""
    return 0
}

#=============================================================================
# Progress Indication Functions
#=============================================================================

# Initialize progress tracking
init_progress() {
    TOTAL_STEPS="$1"
    CURRENT_STEP=0
}

# Advance to next step with progress display
next_step() {
    local description="$1"
    ((CURRENT_STEP++)) || true

    if [ "$TOTAL_STEPS" -gt 0 ]; then
        local pct=$((CURRENT_STEP * 100 / TOTAL_STEPS))
        local bar_width=30
        local filled=$((pct * bar_width / 100))
        local empty=$((bar_width - filled))

        printf "\r${CYAN}[%-${bar_width}s] %3d%%${NC} %s" \
            "$(printf '#%.0s' $(seq 1 $filled 2>/dev/null))$(printf ' %.0s' $(seq 1 $empty 2>/dev/null))" \
            "$pct" "$description"
        echo ""
    else
        print_info "[$CURRENT_STEP] $description"
    fi
}

# Show completion message
complete_progress() {
    echo ""
    print_success "All steps completed successfully!"
}

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    # Reset terminal state in case Huh? TUI left it in raw mode
    stty sane 2>/dev/null || true
    tput reset 2>/dev/null || true

    # Use printf for reliable ANSI handling
    printf '%b' "${CYAN}"
    printf '╔══════════════════════════════════════════════════════╗\n'
    printf '║        %bArmorClaw Container Setup%b%b                     ║\n' "${BOLD}" "${NC}" "${CYAN}"
    printf '║        %bVersion 0.3.6%b%b                                  ║\n' "${BOLD}" "${NC}" "${CYAN}"
    printf '╚══════════════════════════════════════════════════════╝\n'
    printf '%b' "${NC}"
}

print_step() {
    local step="$1"
    local total="$2"
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Step $step of $total: ${BOLD}$3${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ ERROR: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

prompt_input() {
    local prompt="$1"
    local default="$2"
    local result

    # CRITICAL: Drain any pending input from corrupted terminal state
    # The Go TUI wizard may have left garbage in the input buffer
    # This reads and discards anything already in the buffer
    if IFS= read -t 0.01 -r discard 2>/dev/null; then
        # Keep draining until buffer is empty
        while IFS= read -t 0.01 -r discard 2>/dev/null; do :; done
    fi

    # Build the full prompt string
    local full_prompt
    if [ -n "$default" ]; then
        full_prompt="${CYAN}${prompt} [${default}]: ${NC}"
    else
        full_prompt="${CYAN}${prompt}: ${NC}"
    fi

    # Use read -p to print prompt AND read input in one atomic operation
    # This prevents the prompt echo from being captured as input
    # The -r prevents backslash interpretation
    IFS= read -p "$full_prompt" -r result || true

    # Step 1: Strip carriage returns (handles CRLF from terminals)
    result="${result%$'\r'}"

    # Step 2: Strip ANSI escape sequences (may be present if terminal was corrupted)
    result=$(printf '%s' "$result" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' 2>/dev/null || echo "$result")

    # Step 3: Strip leading and trailing whitespace
    result="${result#"${result%%[![:space:]]*}"}"
    result="${result%"${result##*[![:space:]]}"}"

    # Step 4: If result contains colon (prompt leaked into input), extract last part
    if [[ "$result" == *":"* ]]; then
        result="${result##*:}"
        result="${result#"${result%%[![:space:]]*}"}"
    fi

    # Step 5: Return default if empty
    echo "${result:-$default}"
}

# Prompt for sensitive input (passwords, API keys) - hides input on screen
prompt_password() {
    local prompt="$1"
    local default="$2"
    local result

    # CRITICAL: Drain any pending input from corrupted terminal state
    if IFS= read -t 0.01 -r discard 2>/dev/null; then
        while IFS= read -t 0.01 -r discard 2>/dev/null; do :; done
    fi

    # Build the full prompt string
    local full_prompt
    if [ -n "$default" ]; then
        full_prompt="${CYAN}${prompt} [${default}]: ${NC}"
    else
        full_prompt="${CYAN}${prompt}: ${NC}"
    fi

    # Use read -s for silent input (hidden), -p for prompt
    # The -s flag prevents echo of typed characters
    IFS= read -s -p "$full_prompt" -r result || true

    # Print newline after hidden input (cursor is still on same line)
    echo ""

    # Step 1: Strip carriage returns
    result="${result%$'\r'}"

    # Step 2: Strip ANSI escape sequences
    result=$(printf '%s' "$result" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' 2>/dev/null || echo "$result")

    # Step 3: Strip leading and trailing whitespace
    result="${result#"${result%%[![:space:]]*}"}"
    result="${result%"${result##*[![:space:]]}"}"

    # Step 4: If result contains colon (prompt leaked), extract last part
    if [[ "$result" == *":"* ]]; then
        result="${result##*:}"
        result="${result#"${result%%[![:space:]]*}"}"
    fi

    # Step 5: Return default if empty
    echo "${result:-$default}"
}

prompt_yes_no() {
    local prompt="$1"
    local default="${2:-n}"

    while true; do
        # CRITICAL: Drain any pending input from corrupted terminal state
        if IFS= read -t 0.01 -r discard 2>/dev/null; then
            while IFS= read -t 0.01 -r discard 2>/dev/null; do :; done
        fi

        # Build the full prompt string
        local full_prompt
        if [ "$default" = "y" ]; then
            full_prompt="${CYAN}${prompt} [Y/n]: ${NC}"
        else
            full_prompt="${CYAN}${prompt} [y/N]: ${NC}"
        fi

        # Use read -p for atomic prompt + read operation
        IFS= read -p "$full_prompt" -r response || true

        # Strip carriage returns and ANSI escape sequences
        response="${response%$'\r'}"
        response=$(printf '%s' "$response" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' 2>/dev/null || echo "$response")
        response="${response#"${response%%[![:space:]]*}"}"
        response="${response%"${response##*[![:space:]]}"}"

        # If response contains colon, extract just the user input
        if [[ "$response" == *":"* ]]; then
            response="${response##*:}"
            response="${response#"${response%%[![:space:]]*}"}"
        fi

        response=${response:-$default}

        case "$response" in
            [Yy]|[Yy][Ee][Ss]) return 0 ;;
            [Nn]|[Nn][Oo]) return 1 ;;
        esac

        echo -e "${YELLOW}Please answer yes or no.${NC}"
    done
}

check_required_tools() {
    local missing=()
    for cmd in openssl jq socat curl docker; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        set_error "INS-006" "Required tools missing: ${missing[*]}"
        print_error "Required tools missing: ${missing[*]}"
        print_error "Install with: apt-get install -y ${missing[*]}"
        exit 1
    fi
    log_success "Required tools verified (openssl, jq, socat, curl, docker)"
}

validate_config_vars() {
    local errors=()

    [ -z "${MATRIX_URL:-}" ] && errors+=("MATRIX_URL is empty")
    [ -z "${SOCKET_PATH:-}" ] && errors+=("SOCKET_PATH is empty")
    [ -z "${BRIDGE_PASSWORD:-}" ] && errors+=("BRIDGE_PASSWORD is empty")
    [ -z "${API_BASE_URL:-}" ] && errors+=("API_BASE_URL is empty")

    if [ ${#errors[@]} -gt 0 ]; then
        print_error "Configuration validation failed:"
        for err in "${errors[@]}"; do
            print_error "  - $err"
        done
        print_error "Cannot write configuration. Please re-run setup."
        exit 1
    fi

    # Non-critical warnings
    [ -z "${API_KEY:-}" ] && print_warning "No API key configured — add one later via RPC store_key"

    print_success "Configuration variables validated"
}

#=============================================================================
# Wizard JSON Input Support
#=============================================================================

# Load configuration from wizard JSON output file.
# Non-secret values come from the JSON file; secrets come from env vars
# (ARMORCLAW_WIZARD_API_KEY, ARMORCLAW_WIZARD_ADMIN_PASSWORD,
# ARMORCLAW_WIZARD_BRIDGE_PASSWORD) to avoid leaving them on disk.
load_wizard_json() {
    local json_file="$1"

    if [ ! -f "$json_file" ]; then
        print_error "Wizard output file not found: $json_file"
        exit 1
    fi

    print_info "Loading configuration from wizard output..."

    # Parse non-secret config from JSON
    DEPLOY_PROFILE=$(jq -r '.profile // "quick"' "$json_file")
    API_PROVIDER=$(jq -r '.api_provider // "openai"' "$json_file")
    API_BASE_URL=$(jq -r '.api_base_url // "https://api.openai.com/v1"' "$json_file")
    ADMIN_USER=$(jq -r '.admin_user // "admin"' "$json_file")
    LOG_LEVEL=$(jq -r '.log_level // "info"' "$json_file")
    SOCKET_PATH=$(jq -r '.socket_path // "/run/armorclaw/bridge.sock"' "$json_file")
    SECURITY_TIER=$(jq -r '.security_tier // "enhanced"' "$json_file")
    HIPAA_ENABLED=$(jq -r '.hipaa_enabled // false' "$json_file")
    QUARANTINE_ENABLED=$(jq -r '.quarantine_enabled // false' "$json_file")
    AUDIT_RETENTION_DAYS=$(jq -r '.audit_retention_days // 90' "$json_file")

    # Secrets come from environment variables (set by the Go wizard process)
    API_KEY="${ARMORCLAW_WIZARD_API_KEY:-}"
    ADMIN_PASSWORD="${ARMORCLAW_WIZARD_ADMIN_PASSWORD:-}"
    BRIDGE_PASSWORD="${ARMORCLAW_WIZARD_BRIDGE_PASSWORD:-$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || head -c 32 /dev/urandom | base64 2>/dev/null | tr -d '/+=\n' | cut -c1-16)}"

    # Auto-detect server IP
    local detected_ip=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')
    SERVER_NAME="${detected_ip:-127.0.0.1}"
    MATRIX_SERVER="${SERVER_NAME}:8448"
    MATRIX_URL="http://localhost:6167"
    BRIDGE_USER="bridge"

    # Enterprise compliance patterns
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        COMPLIANCE_PATTERNS_PII="true"
        if [ "$HIPAA_ENABLED" = "true" ]; then
            COMPLIANCE_PATTERNS_PHI="true"
        fi
    fi

    NON_INTERACTIVE=true
    FROM_WIZARD=true

    print_success "Wizard configuration loaded (profile: $DEPLOY_PROFILE)"

    # Delete the JSON file immediately after reading — secrets are in env vars
    rm -f "$json_file"
    print_info "Wizard JSON file removed (secrets in memory only)"
}

#=============================================================================
# Environment Variable Support
#=============================================================================

# Check for environment variables (non-interactive mode)
check_env_vars() {
    # Check for minimal required env vars for non-interactive mode
    # ARMORCLAW_API_KEY is required for non-interactive setup
    # ARMORCLAW_SERVER_NAME alone is NOT enough for non-interactive mode
    if [ -z "${ARMORCLAW_API_KEY:-}" ]; then
        # No API key - not non-interactive mode
        NON_INTERACTIVE=false
        return
    fi

    print_info "Environment variables detected - using non-interactive mode"
    NON_INTERACTIVE=true

    # Profile support
    DEPLOY_PROFILE="${ARMORCLAW_PROFILE:-quick}"

    # Server name - auto-detect if not provided
    SERVER_NAME="${ARMORCLAW_SERVER_NAME:-$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo '127.0.0.1')}"

    # Matrix config - use internal by default
    MATRIX_SERVER="${ARMORCLAW_MATRIX_SERVER:-$SERVER_NAME:6167}"
    MATRIX_URL="${ARMORCLAW_MATRIX_URL:-http://localhost:6167}"

    # API config
    API_KEY="${ARMORCLAW_API_KEY:-}"
    API_BASE_URL="${ARMORCLAW_API_BASE_URL:-https://api.openai.com/v1}"
    # Detect provider from base URL
    case "$API_BASE_URL" in
        *anthropic*) API_PROVIDER="anthropic" ;;
        *) API_PROVIDER="openai" ;;
    esac

    # Bridge config
    BRIDGE_PASSWORD="${ARMORCLAW_BRIDGE_PASSWORD:-$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || head -c 32 /dev/urandom | base64 2>/dev/null | tr -d '/+=\n' | cut -c1-16)}"
    LOG_LEVEL="${ARMORCLAW_LOG_LEVEL:-info}"
    SOCKET_PATH="${ARMORCLAW_SOCKET_PATH:-/run/armorclaw/bridge.sock}"

    # Enterprise profile env vars
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        SECURITY_TIER="maximum"
        HIPAA_ENABLED="${ARMORCLAW_HIPAA:-false}"
        QUARANTINE_ENABLED="${ARMORCLAW_QUARANTINE:-true}"
        AUDIT_RETENTION_DAYS="${ARMORCLAW_AUDIT_RETENTION:-90}"
        COMPLIANCE_PATTERNS_PII="true"
        if [ "$HIPAA_ENABLED" = "true" ]; then
            COMPLIANCE_PATTERNS_PHI="true"
        fi
        LOG_LEVEL="${ARMORCLAW_LOG_LEVEL:-info}"
    else
        SECURITY_TIER="${ARMORCLAW_SECURITY_TIER:-enhanced}"
    fi
}

#=============================================================================
# Profile Selection
#=============================================================================

select_profile() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Deployment profile: $DEPLOY_PROFILE"
        return
    fi

    # CRITICAL: Reset terminal before any interactive input
    # The Go TUI wizard may have crashed and left terminal in raw mode
    reset_terminal_full

    # Small additional delay to ensure terminal is fully settled
    sleep 0.05 2>/dev/null || true

    echo -e "${CYAN}Choose your deployment profile:${NC}"
    echo ""
    echo "  ${BOLD}1) Quick Start${NC} (default)"
    echo "     Fewest questions, running in ~2 minutes."
    echo "     Best for: developers, testing, personal use."
    echo ""
    echo "  ${BOLD}2) Enterprise / Compliance${NC}"
    echo "     Guided setup with PII/PHI scrubbing, audit logging,"
    echo "     HIPAA support, and production-grade security."
    echo "     Best for: compliance teams, healthcare, regulated industries."
    echo ""

    while true; do
        local choice
        choice=$(prompt_input "Profile" "1")

        # Debug: show what we received (helps diagnose terminal issues)
        log_debug "Profile selection received: '$choice' (length: ${#choice})"

        # Strip any remaining escape sequences and whitespace
        choice=$(echo "$choice" | sed 's/\x1b\[[0-9;]*m//g' | tr -d '[:space:]')

        case "$choice" in
            1)
                DEPLOY_PROFILE="quick"
                print_info "Selected: Quick Start"
                break
                ;;
            2)
                DEPLOY_PROFILE="enterprise"
                print_info "Selected: Enterprise / Compliance"
                break
                ;;
            *)
                print_warning "Please enter 1 or 2. (received: '$choice')"
                ;;
        esac
    done
    echo ""
}

#=============================================================================
# Quick Start Auto-Defaults
#=============================================================================

apply_quick_defaults() {
    # Auto-detect server IP
    print_info "Detecting server IP..."
    local detected_ip=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')
    if [ -n "$detected_ip" ]; then
        print_success "Detected IP: $detected_ip"
    else
        detected_ip="127.0.0.1"
        print_warning "Could not detect IP, using $detected_ip"
    fi

    SERVER_NAME="$detected_ip"
    MATRIX_SERVER="${detected_ip}:8448"
    MATRIX_URL="http://localhost:6167"
    BRIDGE_USER="bridge"
    BRIDGE_PASSWORD=$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || head -c 32 /dev/urandom | base64 2>/dev/null | tr -d '/+=\n' | cut -c1-16)
    LOG_LEVEL="info"
    SOCKET_PATH="/run/armorclaw/bridge.sock"
    SECURITY_TIER="enhanced"
}

#=============================================================================
# Compliance Configuration (Enterprise Profile)
#=============================================================================

configure_compliance() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for compliance configuration"
        return
    fi

    local total_steps=6
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        total_steps=6
    fi

    print_step 3 $total_steps "Compliance & Security Configuration"

    echo -e "${BOLD}PII Scrubbing${NC}"
    echo "  Detects and redacts personally identifiable information"
    echo "  (SSN, credit cards, emails, phone numbers, IP addresses, API tokens)"
    echo "  in all messages before they reach the AI agent."
    echo ""
    COMPLIANCE_PATTERNS_PII="true"
    print_success "PII scrubbing: enabled (always on for Enterprise)"

    echo ""
    echo -e "${BOLD}HIPAA Compliance${NC}"
    echo "  Enables Protected Health Information (PHI) pattern detection:"
    echo "  medical record numbers, health plan IDs, lab results,"
    echo "  diagnoses (ICD codes), prescriptions, biometric data."
    echo ""

    if prompt_yes_no "Enable HIPAA compliance mode?" "n"; then
        HIPAA_ENABLED="true"
        COMPLIANCE_PATTERNS_PHI="true"
        print_success "HIPAA mode: enabled"
    else
        HIPAA_ENABLED="false"
        print_info "HIPAA mode: disabled (can enable later in config.toml)"
    fi

    echo ""
    echo -e "${BOLD}Quarantine Mode${NC}"
    echo "  Blocks messages containing critical PII/PHI findings"
    echo "  for admin review instead of silently scrubbing."
    echo ""

    if prompt_yes_no "Enable quarantine mode?" "y"; then
        QUARANTINE_ENABLED="true"
        print_success "Quarantine mode: enabled"
    else
        QUARANTINE_ENABLED="false"
        print_info "Quarantine mode: disabled"
    fi

    echo ""
    AUDIT_RETENTION_DAYS=$(prompt_input "Audit log retention (days)" "90")
    print_info "Audit retention: ${AUDIT_RETENTION_DAYS} days"

    # Enterprise always uses maximum security
    SECURITY_TIER="maximum"
    print_success "Security tier: maximum (auto-set for Enterprise)"
}

#=============================================================================
# Setup Functions
#=============================================================================

create_directories() {
    print_info "Creating directory structure..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$RUN_DIR"
    mkdir -p "/var/log/armorclaw"
    mkdir -p "$CONFIG_DIR/ssl"

    print_success "Directories created"
}

generate_self_signed_cert() {
    print_info "Generating self-signed SSL certificate..."

    local ssl_dir="$CONFIG_DIR/ssl"
    local ip_address=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')
    if [ -z "$ip_address" ]; then
        ip_address="127.0.0.1"
        print_warning "Could not detect IP for certificate, using $ip_address"
    fi

    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$ssl_dir/key.pem" \
        -out "$ssl_dir/cert.pem" \
        -subj "/CN=$ip_address" \
        -addext "subjectAltName=IP:$ip_address" 2>/dev/null

    chmod 600 "$ssl_dir/key.pem"
    chmod 644 "$ssl_dir/cert.pem"

    print_success "Self-signed certificate generated for $ip_address"
    print_warning "Note: Browsers will show security warnings. Ask the agent about ngrok or Cloudflare for trusted SSL."
}

check_docker_socket() {
    log_info "Checking Docker socket..."

    if [ ! -S /var/run/docker.sock ]; then
        set_error "INS-001" "Docker socket not found at /var/run/docker.sock"
        print_error "Docker socket not found at /var/run/docker.sock"
        print_error "This container requires Docker socket access."
        echo ""
        echo "Run with: docker run -v /var/run/docker.sock:/var/run/docker.sock ..."
        exit 1
    fi

    # Test Docker access - check if we can talk to the daemon
    local docker_error
    docker_error=$(docker info 2>&1)
    if [ $? -ne 0 ]; then
        set_error "INS-001" "Cannot connect to Docker daemon: $docker_error"
        print_error "Cannot connect to Docker daemon"
        print_error "Ensure your user has Docker permissions"
        echo ""
        echo "Error details: $docker_error"
        echo ""
        echo "Solutions:"
        echo "  1. Run container as root: --user root"
        echo "  2. Add user to docker group on host: sudo usermod -aG docker \$USER"
        echo "  3. Fix socket permissions: sudo chmod 666 /var/run/docker.sock"
        echo ""
        echo "For quickstart container, add to your docker run command:"
        echo "  --user root"
        exit 1
    fi

    log_success "Docker socket accessible"
}

configure_matrix() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for Matrix configuration"
        return
    fi

    # Quick profile: Matrix is auto-configured, skip this step
    if [ "$DEPLOY_PROFILE" = "quick" ]; then
        print_info "Matrix server: $MATRIX_SERVER (auto-configured)"
        print_success "Matrix configuration complete"
        return
    fi

    # Enterprise profile: full Matrix configuration
    print_step 1 6 "Matrix Homeserver Configuration"

    echo "Enter your Matrix server details:"
    echo "  - For domain: matrix.example.com"
    echo "  - For IP-only: YOUR_VPS_IP (e.g., 192.168.1.100)"
    echo ""

    while true; do
        MATRIX_SERVER=$(prompt_input "Matrix server (domain or IP)" "")
        if [ -n "$MATRIX_SERVER" ]; then
            break
        fi
        print_warning "Matrix server is required. Please enter a domain or IP address."
    done

    # Auto-detect if IP address
    if echo "$MATRIX_SERVER" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
        print_info "Detected IP address mode - using HTTP"
        MATRIX_URL="http://localhost:6167"
        MATRIX_SERVER="$MATRIX_SERVER:8448"
    else
        MATRIX_URL=$(prompt_input "Matrix homeserver URL" "http://localhost:6167")
    fi

    echo ""
    if prompt_yes_no "Create bridge user on Matrix?" "y"; then
        BRIDGE_USER=$(prompt_input "Bridge username" "bridge")
        # Strip @ prefix and :server suffix if user enters full Matrix ID format
        BRIDGE_USER=$(echo "$BRIDGE_USER" | sed 's/^@//' | cut -d: -f1)
        BRIDGE_PASSWORD=$(prompt_password "Bridge password" "")
        if [ -z "$BRIDGE_PASSWORD" ]; then
            # Generate random password
            BRIDGE_PASSWORD=$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || head -c 32 /dev/urandom | base64 2>/dev/null | tr -d '/+=\n' | cut -c1-16)
            print_info "Generated password: $BRIDGE_PASSWORD"
        fi
    else
        BRIDGE_USER="bridge"
        BRIDGE_PASSWORD="bridge123"
    fi

    print_success "Matrix configuration complete"
}

configure_api() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for API configuration"
    else
        local step_num=1
        local total_steps=2
        if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
            step_num=2
            total_steps=6
        fi
        print_step $step_num $total_steps "AI Provider Configuration"

        echo "Select your AI provider:"
        echo "  1) OpenAI"
        echo "  2) Anthropic (Claude)"
        echo "  3) GLM-5 (Zhipu AI)"
        echo "  4) Custom (OpenAI-compatible)"
        echo ""

        while true; do
            local choice=$(prompt_input "Provider" "1")

            case "$choice" in
                1)
                    API_BASE_URL="https://api.openai.com/v1"
                    API_PROVIDER="openai"
                    print_info "Selected: OpenAI"
                    break
                    ;;
                2)
                    API_BASE_URL="https://api.anthropic.com/v1"
                    API_PROVIDER="anthropic"
                    print_info "Selected: Anthropic (Claude)"
                    break
                    ;;
                3)
                    API_BASE_URL="https://api.z.ai/api/coding/paas/v4"
                    API_PROVIDER="openai"
                    print_info "Selected: GLM-5 (Zhipu AI)"
                    break
                    ;;
                4)
                    while true; do
                        API_BASE_URL=$(prompt_input "API base URL" "")
                        if [ -n "$API_BASE_URL" ]; then
                            break
                        fi
                        print_warning "API base URL is required for custom providers."
                    done
                    API_PROVIDER="openai"
                    print_info "Selected: Custom ($API_BASE_URL)"
                    break
                    ;;
                *)
                    print_warning "Invalid choice '$choice'. Please enter 1, 2, 3, or 4."
                    ;;
            esac
        done

        while true; do
            API_KEY=$(prompt_password "API key" "")
            if [ -z "$API_KEY" ]; then
                print_warning "API key is required."
                continue
            fi
            if [ ${#API_KEY} -lt 20 ]; then
                print_warning "API key appears too short (minimum 20 characters)."
                print_info "OpenAI keys start with 'sk-', Anthropic keys start with 'sk-ant-'"
                continue
            fi
            break
        done
    fi

    print_success "API configuration complete"
}

configure_bridge() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for bridge configuration"
        return
    fi

    # Quick profile: use defaults, skip this step
    if [ "$DEPLOY_PROFILE" = "quick" ]; then
        LOG_LEVEL="info"
        SOCKET_PATH="/run/armorclaw/bridge.sock"
        return
    fi

    # Enterprise profile: full bridge configuration
    print_step 5 6 "Bridge Configuration"

    echo "Configure the ArmorClaw bridge settings:"
    echo ""

    LOG_LEVEL=$(prompt_input "Log level (debug/info/warn)" "info")
    SOCKET_PATH=$(prompt_input "Bridge socket path" "/run/armorclaw/bridge.sock")

    echo ""
    print_info "Log level: $LOG_LEVEL"
    print_info "Socket path: $SOCKET_PATH"

    print_success "Bridge configuration complete"
}

write_config() {
    print_info "Writing configuration..."

    # Safety: Strip @ prefix and :server suffix from username if present
    # Matrix login API expects just the localpart (e.g., "bridge" not "@bridge:server.com")
    BRIDGE_USER=$(echo "${BRIDGE_USER:-bridge}" | sed 's/^@//' | cut -d: -f1)

    print_info "Creating config.toml..."

    # Budget defaults vary by profile
    local budget_daily="5.0"
    local budget_monthly="100.0"
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        budget_daily="2.0"
        budget_monthly="50.0"
    fi

    # Compliance settings
    local compliance_enabled="false"
    local compliance_streaming="true"
    local compliance_quarantine="false"
    local compliance_audit="false"
    local compliance_tier="${SECURITY_TIER:-essential}"
    local compliance_retention="${AUDIT_RETENTION_DAYS:-90}"

    if [ "$DEPLOY_PROFILE" = "enterprise" ] || [ "$SECURITY_TIER" = "maximum" ]; then
        compliance_enabled="true"
        compliance_streaming="false"
        compliance_quarantine="${QUARANTINE_ENABLED:-true}"
        compliance_audit="true"
    fi

    cat > "$CONFIG_FILE" << EOF
# ArmorClaw Bridge Configuration
# Generated by container-setup.sh on $(date -Iseconds)
# Profile: ${DEPLOY_PROFILE}

[server]
socket_path = "${SOCKET_PATH:-/run/armorclaw/bridge.sock}"
pid_file = "/run/armorclaw/bridge.pid"
daemonize = false

[keystore]
db_path = "${DATA_DIR}/keystore.db"
master_key = ""
providers = []

[matrix]
enabled = true
homeserver_url = "${MATRIX_URL}"
username = "${BRIDGE_USER:-bridge}"
password = "${BRIDGE_PASSWORD}"
device_id = "armorclaw-bridge"
sync_interval = 5
auto_rooms = []

[matrix.retry]
max_retries = 3
retry_delay = 5
backoff_multiplier = 2.0

[matrix.zero_trust]
trusted_senders = []
trusted_rooms = []
reject_untrusted = false

[budget]
daily_limit_usd = ${budget_daily}
monthly_limit_usd = ${budget_monthly}
alert_threshold = 80.0
hard_stop = true

[logging]
level = "${LOG_LEVEL:-info}"
format = "json"
output = "stdout"

[discovery]
enabled = true
instance_name = "${SERVER_NAME}"
port = 8080
tls = false
api_path = "/api"
ws_path = "/ws"

[notifications]
enabled = false
alert_threshold = 0.8

[eventbus]
websocket_enabled = false
websocket_addr = "0.0.0.0:8444"
websocket_path = "/events"
max_subscribers = 100
inactivity_timeout = "30m"

[errors]
enabled = true
store_enabled = true
notify_enabled = true
store_path = "${DATA_DIR}/errors.db"
retention_days = 30
rate_limit_window = "5m"
retention_period = "24h"

[compliance]
enabled = ${compliance_enabled}
streaming_mode = ${compliance_streaming}
quarantine_enabled = ${compliance_quarantine}
notify_on_quarantine = true
audit_enabled = ${compliance_audit}
audit_retention_days = ${compliance_retention}
tier = "${compliance_tier}"

EOF

    # Add compliance patterns section for enterprise profile
    if [ "$COMPLIANCE_PATTERNS_PII" = "true" ] || [ "$COMPLIANCE_PATTERNS_PHI" = "true" ]; then
        cat >> "$CONFIG_FILE" << EOF
[compliance.patterns]
# Standard PII patterns
ssn = true
credit_card = true
email = true
phone = true
ip_address = true
api_token = true

# HIPAA / PHI patterns
medical_record = ${COMPLIANCE_PATTERNS_PHI}
health_plan = ${COMPLIANCE_PATTERNS_PHI}
device_id = ${COMPLIANCE_PATTERNS_PHI}
biometric = ${COMPLIANCE_PATTERNS_PHI}
lab_result = ${COMPLIANCE_PATTERNS_PHI}
diagnosis = ${COMPLIANCE_PATTERNS_PHI}
prescription = ${COMPLIANCE_PATTERNS_PHI}

EOF
    fi

    # Add provisioning section
    cat >> "$CONFIG_FILE" << EOF
[provisioning]
signing_secret = "$(openssl rand -hex 32)"
default_expiry_seconds = 60
max_expiry_seconds = 300
one_time_use = true
data_dir = "${DATA_DIR}"
EOF

    chmod 600 "$CONFIG_FILE"
    print_success "Configuration written to $CONFIG_FILE"
}

initialize_keystore() {
    print_info "Initializing keystore..."

    print_info "Keystore will be initialized by bridge on first run..."

    # Create keystore directory
    mkdir -p "$DATA_DIR"

    # Store API key temporarily for later injection
    # The bridge will initialize its own SQLCipher keystore
    print_info "Storing API key for bridge injection..."

    # Create a temp file with API key for the bridge to read on startup
    # Use printf to avoid shell expansion of special chars in API keys
    printf 'api_key=%s\nbase_url=%s\nprovider=%s\n' "$API_KEY" "$API_BASE_URL" "${API_PROVIDER:-openai}" > "$DATA_DIR/.api_key_temp"
    chmod 600 "$DATA_DIR/.api_key_temp"

    print_success "Keystore prepared (bridge will initialize on startup)"
}

add_api_key_to_bridge() {
    print_info "Adding API key to bridge keystore..."

    # Wait for bridge socket
    local max_attempts=30
    local attempt=0
    while [ ! -S /run/armorclaw/bridge.sock ] && [ $attempt -lt $max_attempts ]; do
        sleep 1
        attempt=$((attempt + 1))
    done

    if [ -S /run/armorclaw/bridge.sock ]; then
        # Add API key via RPC (method: store_key, requires id + provider + token)
        local key_provider="${API_PROVIDER:-openai}"
        echo "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"store_key\",\"params\":{\"id\":\"${key_provider}-default\",\"provider\":\"${key_provider}\",\"token\":\"$API_KEY\",\"display_name\":\"Default API\"}}" | \
            socat - UNIX-CONNECT:/run/armorclaw/bridge.sock 2>/dev/null || true

        # Clean up temp file
        rm -f "$DATA_DIR/.api_key_temp"

        print_success "API key added to keystore"
    else
        print_warning "Bridge socket not available - API key stored in $DATA_DIR/.api_key_temp"
        print_warning "Add manually with: store_key RPC method"
    fi
}

start_matrix_stack() {
    print_info "Starting Matrix stack..."

    cd /opt/armorclaw

    # Check if using external Matrix server
    if [ "${ARMORCLAW_EXTERNAL_MATRIX:-false}" = "true" ]; then
        print_info "External Matrix mode enabled - skipping internal Matrix management"

        # Verify external Matrix is reachable
        local matrix_url="${ARMORCLAW_MATRIX_HOMESERVER_URL:-http://127.0.0.1:6167}"
        if curl -sf --connect-timeout 5 "${matrix_url}/_matrix/client/versions" >/dev/null 2>&1; then
            print_success "External Matrix server is reachable at $matrix_url"
            MATRIX_AVAILABLE=true
            return
        else
            print_warning "External Matrix server not reachable at $matrix_url"
            print_warning "Bridge will attempt to connect when Matrix becomes available"
            MATRIX_AVAILABLE=false
            return
        fi
    fi

    # Check if Matrix is available on localhost (external deployment via docker-compose-full.yml)
    if curl -sf --connect-timeout 3 "http://127.0.0.1:6167/_matrix/client/versions" >/dev/null 2>&1; then
        print_success "Matrix server detected on localhost:6167 - using external Matrix"
        print_info "Set ARMORCLAW_EXTERNAL_MATRIX=true to skip this check"
        MATRIX_AVAILABLE=true
        return
    fi

    if [ ! -f "docker-compose.matrix.yml" ]; then
        print_warning "docker-compose.matrix.yml not found - skipping Matrix stack"
        return
    fi

    # Check if Matrix is already running (verify health before cleanup)
    local matrix_already_running=false
    if docker ps 2>/dev/null | grep -q "armorclaw-conduit"; then
        print_info "Matrix container already running - verifying health..."
        matrix_already_running=true

        # Wait for Matrix to be healthy (up to 60 seconds)
        local health_attempt=0
        while [ $health_attempt -lt 12 ]; do
            if curl -sf --connect-timeout 3 "http://localhost:6167/_matrix/client/versions" >/dev/null 2>&1; then
                print_success "Matrix is already running and healthy"
                MATRIX_AVAILABLE=true
                return
            fi
            health_attempt=$((health_attempt + 1))
            sleep 5
        done

        # Matrix container running but not healthy - will restart
        print_warning "Matrix container running but not healthy - will restart"
    fi



    # --- Prepare config files on the HOST filesystem ---
    # When running inside a container with the Docker socket mounted,
    # bind mount paths in compose files resolve on the HOST filesystem.
    local HOST_CONFIGS="/tmp/armorclaw-configs"
    print_info "Copying config files to host ($HOST_CONFIGS)..."

    # Create config directory on HOST
    docker run --rm -v /tmp:/tmp alpine mkdir -p "$HOST_CONFIGS" 2>/dev/null || true

    # Verify directory was created
    if ! docker run --rm -v /tmp:/tmp alpine test -d "$HOST_CONFIGS" 2>/dev/null; then
        print_error "Failed to create config directory on host"
        return 1
    fi
    log_debug "Config directory created on host: $HOST_CONFIGS"

    # Export server name without port for Conduit's federation identity
    export MATRIX_SERVER_NAME="${MATRIX_SERVER%%:*}"

    # Generate a registration shared secret for creating users without open registration.
    # This is written into conduit.toml, used to register bridge + admin users,
    # then removed after registration is complete.
    export REGISTRATION_SHARED_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n')"

    # Dynamically update conduit.toml with the correct server_name and shared secret
    local CONDUIT_TEMPLATE="/opt/armorclaw/configs/conduit.toml"
    local CONDUIT_STAGING="/tmp/conduit.toml.staging"
    if [ -f "$CONDUIT_TEMPLATE" ]; then
        cp "$CONDUIT_TEMPLATE" "$CONDUIT_STAGING"
        sed -i "s|^server_name = .*|server_name = \"${MATRIX_SERVER_NAME}\"|" "$CONDUIT_STAGING"
        sed -i "s|^client = .*|client = \"https://${MATRIX_SERVER_NAME}\"|" "$CONDUIT_STAGING"
        sed -i "s|^server = .*|server = \"${MATRIX_SERVER_NAME}:6167\"|" "$CONDUIT_STAGING"
        # Append registration_shared_secret for user creation
        echo "" >> "$CONDUIT_STAGING"
        echo "# Temporary: shared secret for user registration (removed after setup)" >> "$CONDUIT_STAGING"
        echo "registration_shared_secret = \"${REGISTRATION_SHARED_SECRET}\"" >> "$CONDUIT_STAGING"
        log_debug "Generated conduit.toml with server_name=$MATRIX_SERVER_NAME"
    else
        print_error "Conduit template not found: $CONDUIT_TEMPLATE"
        return 1
    fi

    # Copy configs to host-accessible path with explicit error checking
    for f in conduit.toml turnserver.conf; do
        local src="/opt/armorclaw/configs/$f"
        # Use staged conduit.toml with dynamic values
        if [ "$f" = "conduit.toml" ] && [ -f "$CONDUIT_STAGING" ]; then
            src="$CONDUIT_STAGING"
        fi
        if [ -f "$src" ]; then
            log_debug "Copying $src to host at $HOST_CONFIGS/$f"
            if ! cat "$src" | docker run --rm -i -v /tmp:/tmp alpine sh -c "cat > $HOST_CONFIGS/$f" 2>&1; then
                print_error "Failed to copy $f to host"
                return 1
            fi
            # Verify file was written
            if ! docker run --rm -v /tmp:/tmp alpine test -f "$HOST_CONFIGS/$f" 2>/dev/null; then
                print_error "Config file not found on host after copy: $HOST_CONFIGS/$f"
                return 1
            fi
            log_debug "Verified: $HOST_CONFIGS/$f exists on host"
        else
            print_warning "Source config not found: $src"
        fi
    done
    # Note: CONDUIT_STAGING is cleaned up after Conduit container starts

    # Set ARMORCLAW_CONFIGS so compose resolves bind mounts to the host path
    export ARMORCLAW_CONFIGS="$HOST_CONFIGS"
    log_debug "ARMORCLAW_CONFIGS set to: $ARMORCLAW_CONFIGS"

    # Generate a random TURN secret for Coturn authentication
    export TURN_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n')"

    # --- Prepare Conduit database directory on HOST ---
    # Conduit runs as UID 10000 inside the container
    print_info "Creating Conduit database directory on host..."
    docker run --rm -v /var/lib:/var/lib alpine sh -c "
        mkdir -p /var/lib/conduit && \
        chown -R 10000:10000 /var/lib/conduit && \
        chmod 755 /var/lib/conduit
    " 2>/dev/null || {
        print_error "Failed to create /var/lib/conduit on host"
        return 1
    }
    log_debug "Created /var/lib/conduit with UID 10000:10000 ownership"

    # --- Copy conduit.toml to permanent HOST location ---
    # Write to /opt/armorclaw/conduit.toml (not /tmp) for persistence
    print_info "Writing Conduit config to /opt/armorclaw/conduit.toml..."
    docker run --rm -v /opt:/opt alpine sh -c "mkdir -p /opt/armorclaw" 2>/dev/null || true
    if ! cat "$CONDUIT_STAGING" | docker run --rm -i -v /opt:/opt alpine sh -c "cat > /opt/armorclaw/conduit.toml" 2>&1; then
        print_error "Failed to write conduit.toml to /opt/armorclaw/"
        return 1
    fi
    log_debug "Wrote conduit.toml to /opt/armorclaw/conduit.toml"

    # --- Start Conduit with shared network namespace ---
    # Use direct 'docker run' instead of compose because network_mode: "service:armorclaw"
    # doesn't work in compose (armorclaw is not a compose service, it's the quickstart container)
    print_info "Starting Conduit container (this may take a few minutes on first run)..."

    # Remove any existing container first
    docker rm -f armorclaw-conduit 2>/dev/null || true

    local CONDUIT_OUTPUT
    CONDUIT_OUTPUT=$(docker run -d \
        --name armorclaw-conduit \
        --restart unless-stopped \
        --network container:armorclaw \
        -v /opt/armorclaw/conduit.toml:/etc/conduit.toml:ro \
        -v /var/lib/conduit:/var/lib/conduit \
        -e CONDUIT_CONFIG=/etc/conduit.toml \
        matrixconduit/matrix-conduit:latest 2>&1)
    local CONDUIT_EXIT=$?

    if [ $CONDUIT_EXIT -ne 0 ]; then
        print_error "Failed to start Conduit container (exit $CONDUIT_EXIT):"
        echo "$CONDUIT_OUTPUT" | tail -20

        # Check for specific error types and provide actionable guidance
        if echo "$CONDUIT_OUTPUT" | grep -qi "port"; then
            print_error ""
            print_error "══════════════════════════════════════════════════════"
            print_error "PORT CONFLICT DETECTED"
            print_error "══════════════════════════════════════════════════════"
            print_error ""
            print_error "Port 6167 is already in use. This is usually caused by:"
            print_error ""
            print_error "  1. A previous Conduit container still running"
            print_error "  2. Another service using port 6167"
            print_error ""
            print_error "Quick fix:"
            print_error "  docker rm -f armorclaw-conduit 2>/dev/null"
            print_error "  docker rm -f \$(docker ps -aq --filter 'ancestor=matrixconduit/matrix-conduit') 2>/dev/null"
            print_error "══════════════════════════════════════════════════════"
        fi

        # Set Matrix as unavailable and disable in config
        MATRIX_AVAILABLE=false
        print_warning "Matrix stack unavailable - bridge will run in standalone mode"

        # Update config to disable Matrix
        if [ -f "$CONFIG_FILE" ]; then
            sed -i 's/^enabled = true/enabled = false/' "$CONFIG_FILE" 2>/dev/null || true
            sed -i 's|^homeserver_url = .*|homeserver_url = ""|' "$CONFIG_FILE" 2>/dev/null || true
            print_info "Updated config to disable Matrix integration"
        fi

        return 1
    fi
    print_success "Conduit container started"

    # Clean up staging file
    rm -f "$CONDUIT_STAGING"

    # --- Wait for Conduit to be healthy ---
    if wait_for_conduit; then
        MATRIX_AVAILABLE=true
    fi
}

wait_for_conduit() {
    log_info "Waiting for Conduit homeserver to be ready..."
    local max_attempts=24  # 24 * 5s = 120s
    local attempt=0
    local container_checked=false

    while [ $attempt -lt $max_attempts ]; do
        # First, check if container is actually running (only check once)
        if ! $container_checked; then
            local container_state
            container_state=$(docker inspect --format '{{.State.Status}}' armorclaw-conduit 2>/dev/null || echo "not found")

            if [ "$container_state" != "running" ]; then
                # Container isn't running - get detailed diagnostics
                print_error ""
                print_error "══════════════════════════════════════════════════════"
                print_error "CONDUIT CONTAINER DIAGNOSTICS"
                print_error "══════════════════════════════════════════════════════"

                # Check if container exists at all
                local container_exists
                container_exists=$(docker ps -aq --filter "name=armorclaw-conduit" --format "{{.Names}}")

                if [ -z "$container_exists" ]; then
                    print_error "❌ Container 'armorclaw-conduit' does NOT exist"
                    print_error "   The docker run command may have failed to create the container"
                    print_error ""
                    print_error "Possible causes:"
                    print_error "  1. Port conflict (port 6167 already in use)"
                    print_error "  2. Volume mount failure (config file not accessible)"
                    print_error "  3. Image pull failure (check network)"
                    print_error ""
                    print_error "Run: docker logs armorclaw-conduit"
                else
                    # Container exists but not running - check exit code
                    local exit_code
                    exit_code=$(docker inspect --format '{{.State.ExitCode}}' armorclaw-conduit 2>/dev/null || echo "N/A")

                    print_error "❌ Container exited with code: $exit_code"
                    print_error ""

                    # Get container logs
                    print_error "Container logs (last 50 lines):"
                    docker logs armorclaw-conduit --tail 50 2>&1 | sed 's/^/.*//' | head -20

                    print_error ""
                    print_error "Common Conduit startup failures:"
                    print_error "  • Config file not found at expected path"
                    print_error "  • Config file has syntax errors"
                    print_error "  • Database initialization failed"
                    print_error "  • Permission denied"
                    print_error ""

                    # Check if config file exists inside container
                    print_error "Checking config file mount..."
                    local config_check
                    config_check=$(docker exec armorclaw-conduit ls -la /etc/conduit.toml 2>&1 || echo "not found")

                    if [ "$config_check" = "not found" ]; then
                        print_error "❌ Config file /etc/conduit.toml NOT FOUND in container"
                        print_error "   Volume mount may have failed"
                        print_error ""
                        print_error "Expected mount: ${ARMORCLAW_CONFIGS:-./configs}/conduit.toml:/etc/conduit.toml"
                        print_error "Host path: ${ARMORCLAW_CONFIGS}/conduit.toml"

                        # Check if file exists on host
                        if [ -n "$ARMORCLAW_CONFIGS" ]; then
                            local host_file
                            host_file=$(docker run --rm -v /tmp:/tmp alpine ls -la "$ARMORCLAW_CONFIGS/conduit.toml" 2>&1 || echo "not found")
                            if [ "$host_file" = "not found" ]; then
                                print_error "❌ Config file DOES NOT EXIST on host at: $ARMORCLAW_CONFIGS/conduit.toml"
                                print_error "   The file copy in start_matrix_stack() may have failed silently"
                            else
                                print_error "✓ Config file exists on host at: $ARMORCLAW_CONFIGS/conduit.toml"
                                print_error "   File size: $(docker run --rm -v /tmp:/tmp alpine stat -c "$ARMORCLAW_CONFIGS/conduit.toml" 2>&1 || echo "bytes")"
                            fi
                        fi
                    fi
                fi

                print_error "══════════════════════════════════════════════════════"
                return 1
            fi

            # Container is running - don't check again
            container_checked=true
        fi

        # Check if Conduit responds to the versions endpoint
        if curl -sf --connect-timeout 3 "http://localhost:6167/_matrix/client/versions" >/dev/null 2>&1; then
            log_success "Conduit homeserver is ready"
            return 0
        fi

        attempt=$((attempt + 1))
        if [ $((attempt % 4)) -eq 0 ]; then
            log_info "Still waiting for Conduit... (${attempt}/${max_attempts})"
        else
            echo -n "."
        fi
        sleep 5
    done

    echo ""
    set_error "INS-007" "Conduit did not become ready within 120 seconds"
    print_error "Conduit did not become ready within 120 seconds"
    print_warning "Check Conduit logs: docker logs armorclaw-conduit"
    print_warning "Bridge user registration will be skipped - you may need to run create-matrix-admin.sh manually"
    return 1
}

register_matrix_user() {
    local username="$1"
    local password="$2"
    local is_admin="${3:-false}"
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    # Strip @ prefix and :server suffix if present (handle both formats)
    username=$(echo "$username" | sed 's/^@//' | cut -d: -f1)

    print_info "Registering Matrix user: $username"

    # Method 1: Try standard Matrix v3 registration API (works with Conduit)
    # This requires allow_registration = true in conduit.toml
    local REG_RESPONSE
    REG_RESPONSE=$(curl -sf --connect-timeout 10 -X POST \
        "http://localhost:6167/_matrix/client/v3/register" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\",\"auth\":{\"type\":\"m.login.dummy\"}}" 2>/dev/null)
    local REG_EXIT=$?

    if [ $REG_EXIT -eq 0 ] && [ -n "$REG_RESPONSE" ]; then
        local USER_ID
        USER_ID=$(echo "$REG_RESPONSE" | jq -r '.user_id // .access_token // empty' 2>/dev/null)
        if [ -n "$USER_ID" ]; then
            print_success "Registered Matrix user: $USER_ID"
            return 0
        fi
    fi

    # Check if user already exists
    local ERROR_MSG
    ERROR_MSG=$(echo "$REG_RESPONSE" | jq -r '.errcode // .error // empty' 2>/dev/null)
    if [ "$ERROR_MSG" = "M_USER_IN_USE" ] || echo "$REG_RESPONSE" | grep -qi "already\|exists\|in.use"; then
        print_info "User @${username}:${server_name} already exists"
        return 0
    fi

    # Method 2: Try Synapse admin API (for Synapse compatibility)
    if [ -n "$REGISTRATION_SHARED_SECRET" ]; then
        print_info "Trying Synapse admin API for registration..."

        # Step 1: Get nonce
        local NONCE_RESPONSE
        NONCE_RESPONSE=$(curl -sf --connect-timeout 10 "http://localhost:6167/_synapse/admin/v1/register" 2>/dev/null)
        if [ $? -eq 0 ] && [ -n "$NONCE_RESPONSE" ]; then
            local NONCE
            NONCE=$(echo "$NONCE_RESPONSE" | jq -r '.nonce // empty' 2>/dev/null)
            if [ -n "$NONCE" ]; then
                # Step 2: Compute HMAC
                local admin_flag="notadmin"
                if [ "$is_admin" = "true" ]; then
                    admin_flag="admin"
                fi

                local MAC
                MAC=$(printf '%s\0%s\0%s\0%s' "$NONCE" "$username" "$password" "$admin_flag" | \
                    openssl dgst -sha1 -hmac "$REGISTRATION_SHARED_SECRET" | awk '{print $NF}')

                if [ -n "$MAC" ]; then
                    # Step 3: Register with Synapse API
                    REG_RESPONSE=$(curl -sf --connect-timeout 10 -X POST \
                        "http://localhost:6167/_synapse/admin/v1/register" \
                        -H "Content-Type: application/json" \
                        -d "{\"nonce\":\"$NONCE\",\"username\":\"$username\",\"password\":\"$password\",\"admin\":$is_admin,\"mac\":\"$MAC\"}" 2>/dev/null)

                    local USER_ID
                    USER_ID=$(echo "$REG_RESPONSE" | jq -r '.user_id // empty' 2>/dev/null)
                    if [ -n "$USER_ID" ]; then
                        print_success "Registered Matrix user: $USER_ID"
                        return 0
                    fi
                fi
            fi
        fi
    fi

    # If we got here, registration failed
    print_warning "Failed to register $username"
    print_info "Response: $REG_RESPONSE"
    print_info ""
    print_info "Manual registration command:"
    print_info "  curl -X POST http://localhost:6167/_matrix/client/v3/register \\"
    print_info "    -H 'Content-Type: application/json' \\"
    print_info "    -d '{\"username\":\"$username\",\"password\":\"$password\",\"auth\":{\"type\":\"m.login.dummy\"}}'"
    return 1
}

configure_admin_user() {
    if [ "$NON_INTERACTIVE" = true ]; then
        ADMIN_USER="${ARMORCLAW_ADMIN_USER:-admin}"
        ADMIN_PASSWORD="${ARMORCLAW_ADMIN_PASSWORD:-}"
        if [ -z "$ADMIN_PASSWORD" ]; then
            ADMIN_PASSWORD=$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || head -c 32 /dev/urandom | base64 2>/dev/null | tr -d '/+=\n' | cut -c1-16)
            print_info "Generated admin password: $ADMIN_PASSWORD"
        fi
        return
    fi

    local step_num=2
    local total_steps=2
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        step_num=4
        total_steps=6
    fi

    print_step $step_num $total_steps "Admin User for Element X / ArmorChat"
    echo "This account is how YOU log in to chat with the bridge."
    echo ""

    if [ "$DEPLOY_PROFILE" = "quick" ]; then
        ADMIN_USER="admin"
        print_info "Username: admin (default)"
    else
        ADMIN_USER=$(prompt_input "Admin username" "admin")
    fi

    while true; do
        ADMIN_PASSWORD=$(prompt_password "Admin password (min 8 chars, press Enter to auto-generate)" "")
        if [ -z "$ADMIN_PASSWORD" ]; then
            # Generate a random password
            ADMIN_PASSWORD=$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || head -c 32 /dev/urandom | base64 2>/dev/null | tr -d '/+=\n' | cut -c1-16)
            if [ -z "$ADMIN_PASSWORD" ]; then
                print_warning "Auto-generation failed. Please enter a password manually."
                continue
            fi
            print_info "Generated admin password: $ADMIN_PASSWORD"
            break
        fi
        if [ ${#ADMIN_PASSWORD} -lt 8 ]; then
            print_warning "Password must be at least 8 characters."
            continue
        fi
        break
    done

    echo ""
    print_info "Admin user: @${ADMIN_USER}:${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"
}

# Wait for Matrix server to be ready (works with Conduit, Synapse, Dendrite)
wait_for_matrix() {
    local matrix_url="${1:-http://localhost:6167}"
    local max_attempts="${2:-30}"
    local attempt=0

    print_info "Waiting for Matrix server at $matrix_url..."

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf --connect-timeout 2 "${matrix_url}/_matrix/client/versions" >/dev/null 2>&1; then
            print_success "Matrix server is ready"
            return 0
        fi

        attempt=$((attempt + 1))
        if [ $((attempt % 5)) -eq 0 ]; then
            print_info "Still waiting for Matrix... (${attempt}/${max_attempts})"
        fi
        sleep 2
    done

    print_error "Matrix server not available at $matrix_url after $((max_attempts * 2)) seconds"
    print_error ""
    print_error "Possible causes:"
    print_error "  1. Matrix container not started"
    print_error "  2. Wrong Docker network (containers must share a network)"
    print_error "  3. Wrong URL (expected $matrix_url)"
    print_error ""
    print_error "Quick fix for docker-compose deployments:"
    print_error "  docker compose -f docker-compose-full.yml up -d"
    print_error "  docker compose -f docker-compose-full.yml logs matrix"
    return 1
}

register_users() {
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    print_info "Registering Matrix users..."

    # Register bridge bot user
    if register_matrix_user "${BRIDGE_USER:-bridge}" "$BRIDGE_PASSWORD" "false"; then
        print_success "Bridge user ready: @${BRIDGE_USER:-bridge}:${server_name}"
    else
        print_warning "Bridge user registration failed - bridge may not connect to Matrix"
        print_warning "Run 'create-matrix-admin.sh ${BRIDGE_USER:-bridge}' manually after setup"
    fi

    # Register admin user (for human to log in via Element X / ArmorChat)
    if register_matrix_user "$ADMIN_USER" "$ADMIN_PASSWORD" "true"; then
        print_success "Admin user ready: @${ADMIN_USER}:${server_name}"
    else
        print_warning "Admin user registration failed"
        print_warning "Run 'create-matrix-admin.sh $ADMIN_USER' manually after setup"
    fi

    # Clean up: Remove registration_shared_secret from Conduit config on host
    # This prevents anyone from registering users after setup
    print_info "Removing registration shared secret..."
    local HOST_CONFIGS="/tmp/armorclaw-configs"
    if ! echo "$(cat /dev/null)" | docker run --rm -i -v /tmp:/tmp alpine sh -c "
        if [ -f '$HOST_CONFIGS/conduit.toml' ]; then
            sed -i '/registration_shared_secret/d' '$HOST_CONFIGS/conduit.toml'
            sed -i '/Temporary.*shared secret/d' '$HOST_CONFIGS/conduit.toml'
        fi
    " 2>/dev/null; then
        print_warning "Failed to remove registration shared secret — remove manually from $HOST_CONFIGS/conduit.toml"
    fi

    # Restart Conduit to pick up the config without shared secret
    docker restart armorclaw-conduit >/dev/null 2>&1 || true
    sleep 3
}

setup_bridge_room() {
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    print_info "Setting up bridge management room..."

    # Wait for Conduit to be ready after restart
    local max_attempts=12
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf --connect-timeout 3 "http://localhost:6167/_matrix/client/versions" >/dev/null 2>&1; then
            break
        fi
        sleep 2
        attempt=$((attempt + 1))
    done

    # Log in as admin to get access token
    local LOGIN_RESPONSE
    LOGIN_RESPONSE=$(curl -sf --connect-timeout 10 -X POST "http://localhost:6167/_matrix/client/v3/login" \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"m.login.password\",\"user\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASSWORD\"}" 2>/dev/null)

    if [ -z "$LOGIN_RESPONSE" ]; then
        print_warning "Could not log in as admin — skipping room setup"
        return 1
    fi

    local ADMIN_TOKEN
    ADMIN_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token // empty' 2>/dev/null)
    if [ -z "$ADMIN_TOKEN" ]; then
        set_error "INS-009" "Admin login failed for bridge room setup"
        print_warning "Admin login failed — skipping room setup"
        return 1
    fi

    # Create the bridge management room.
    # The admin is the creator and automatically gets power level 100.
    local BRIDGE_USER_ID="@${BRIDGE_USER:-bridge}:${server_name}"
    local CREATE_RESPONSE
    CREATE_RESPONSE=$(curl -sf --connect-timeout 10 -X POST "http://localhost:6167/_matrix/client/v3/createRoom" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"preset\": \"trusted_private_chat\",
            \"name\": \"ArmorClaw Bridge\",
            \"topic\": \"Management room — send !status to check bridge health\",
            \"invite\": [\"$BRIDGE_USER_ID\"],
            \"power_level_content_override\": {
                \"users\": {
                    \"@${ADMIN_USER}:${server_name}\": 100,
                    \"$BRIDGE_USER_ID\": 50
                },
                \"events_default\": 0,
                \"state_default\": 50,
                \"ban\": 100,
                \"kick\": 100,
                \"redact\": 50,
                \"invite\": 50
            },
            \"is_direct\": false
        }" 2>/dev/null)

    local ROOM_ID
    ROOM_ID=$(echo "$CREATE_RESPONSE" | jq -r '.room_id // empty' 2>/dev/null)

    if [ -n "$ROOM_ID" ]; then
        print_success "Bridge room created: $ROOM_ID"
        print_info "Admin @${ADMIN_USER}:${server_name} has power level 100 (admin)"
        print_info "Bridge ${BRIDGE_USER_ID} has power level 50 (moderator)"

        # Update config.toml with the room so the bridge auto-joins
        if [ -f "$CONFIG_FILE" ]; then
            sed -i "s|^auto_rooms = .*|auto_rooms = [\"$ROOM_ID\"]|" "$CONFIG_FILE"
        fi

        BRIDGE_ROOM_ID="$ROOM_ID"
    else
        print_warning "Room creation failed — you can create one manually in Element X"
        BRIDGE_ROOM_ID=""
    fi

    # Log out to clean up the access token
    curl -sf --connect-timeout 5 -X POST "http://localhost:6167/_matrix/client/v3/logout" \
        -H "Authorization: Bearer $ADMIN_TOKEN" >/dev/null 2>&1 || true
}

final_summary() {
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    log_section "Setup Complete"

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}        ${BOLD}Setup Complete!${NC}                                  ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Configuration:"
    echo "  - Profile: ${DEPLOY_PROFILE}"
    echo "  - Matrix server: $MATRIX_SERVER"
    echo "  - API provider: $API_BASE_URL"
    echo "  - Security tier: ${SECURITY_TIER:-essential}"
    echo "  - Log level: ${LOG_LEVEL:-info}"
    echo ""
    echo "Files:"
    echo "  - Config: $CONFIG_FILE"
    echo "  - Keystore: $DATA_DIR/keystore.db"
    echo "  - SSL cert: $CONFIG_DIR/ssl/cert.pem"
    echo "  - Setup log: $LOG_FILE"
    echo ""

    # Enterprise compliance summary
    if [ "$DEPLOY_PROFILE" = "enterprise" ] || [ "$SECURITY_TIER" = "maximum" ]; then
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BOLD}Compliance Status:${NC}"
        echo -e "  PII scrubbing:     ${GREEN}enabled${NC} (SSN, credit card, email, phone, IP, API token)"
        if [ "$HIPAA_ENABLED" = "true" ]; then
            echo -e "  HIPAA mode:        ${GREEN}enabled${NC} (medical records, health plans, lab results, diagnoses, prescriptions)"
        else
            echo -e "  HIPAA mode:        ${YELLOW}disabled${NC}"
        fi
        if [ "$QUARANTINE_ENABLED" = "true" ]; then
            echo -e "  Quarantine:        ${GREEN}enabled${NC} (critical findings blocked for review)"
        else
            echo -e "  Quarantine:        ${YELLOW}disabled${NC} (findings scrubbed, not blocked)"
        fi
        echo -e "  Audit logging:     ${GREEN}enabled${NC} (${AUDIT_RETENTION_DAYS} day retention)"
        echo -e "  Response mode:     ${GREEN}buffered${NC} (full-text scrubbing)"
        echo -e "  Logging format:    ${GREEN}JSON${NC} (SIEM-ready)"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
    fi

    # Admin credentials (critical for Element X / ArmorChat login)
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}Admin Login (Element X / ArmorChat):${NC}"
    echo -e "  Username:   ${GREEN}@${ADMIN_USER:-admin}:${server_name}${NC}"
    echo -e "  Password:   ${GREEN}${ADMIN_PASSWORD}${NC}"
    echo -e "  Homeserver: ${GREEN}http://${server_name}:6167${NC}"
    echo ""
    if [ -f "$DATA_DIR/.admin_password" ]; then
        echo -e "  ${CYAN}Password saved to: $DATA_DIR/.admin_password${NC}"
        echo -e "  ${YELLOW}⚠ Delete this file after first login for security.${NC}"
    else
        echo -e "  ${YELLOW}⚠ Save these credentials now — the password is not stored.${NC}"
    fi
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # Show bridge room info if created
    if [ -n "$BRIDGE_ROOM_ID" ]; then
        echo -e "${BOLD}Bridge Room:${NC}"
        echo -e "  Room:       ${GREEN}ArmorClaw Bridge${NC}  ($BRIDGE_ROOM_ID)"
        echo -e "  Your role:  ${GREEN}Admin (power level 100)${NC}"
        echo -e "  Bridge:     ${GREEN}Moderator (power level 50)${NC}"
        echo ""
    fi

    # Check if IP-based setup
    if echo "$MATRIX_SERVER" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+':; then
        echo "Next steps (IP-based setup):"
        echo "  1. Ensure ports are open: 6167, 8448, 5000, 8443"
        echo "  2. Open Element X or ArmorChat"
        echo "  3. Set homeserver to: http://$server_name:6167"
        echo "  4. Log in with admin credentials above"
        if [ -n "$BRIDGE_ROOM_ID" ]; then
            echo "  5. Open 'ArmorClaw Bridge' room (auto-created)"
            echo "  6. Send '!status' to verify connection"
        else
            echo "  5. Start DM with: @${BRIDGE_USER:-bridge}:$server_name"
            echo "  6. Send '!status' to verify connection"
        fi
        echo ""
        echo "For SSL (ask the agent in ArmorChat):"
        echo "  \"Set up a cloudflare tunnel\" - Free, trusted SSL"
        echo ""
        echo "To add more apps (ArmorTerminal):"
        echo "  \"Install armorterminal\" - Terminal access to agents"
    else
        echo "Next steps:"
        echo "  1. Open Element X or ArmorChat"
        echo "  2. Set homeserver to: https://$MATRIX_SERVER"
        echo "  3. Log in with admin credentials above"
        if [ -n "$BRIDGE_ROOM_ID" ]; then
            echo "  4. Open 'ArmorClaw Bridge' room (auto-created)"
            echo "  5. Send '!status' to verify connection"
        else
            echo "  4. Start DM with: @${BRIDGE_USER:-bridge}:$server_name"
            echo "  5. Send '!status' to verify connection"
        fi
    fi
    echo ""

    # Auto-generate QR code for ArmorChat mobile app
    generate_qr_code
}

#=============================================================================
# QR Code Generation
#=============================================================================

generate_qr_code() {
    local hostname="${SERVER_NAME:-localhost}"
    local port="${BRIDGE_PORT:-8443}"

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}ArmorChat Mobile App Connection${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # Check if armorclaw-bridge command exists
    if command -v armorclaw-bridge &> /dev/null; then
        print_info "Generating QR code for ArmorChat..."
        echo ""

        # Generate QR code using the bridge CLI
        if armorclaw-bridge generate-qr --host "$hostname" --port "$port" 2>/dev/null; then
            return 0
        fi
    fi

    # Fallback: Generate QR code manually if bridge command not available
    print_info "Generating QR code..."

    # Create config JSON for deep link
    local matrix_url="http://${hostname}:6167"
    local rpc_url="http://${hostname}:${port}/api"
    local ws_url="ws://${hostname}:${port}/ws"
    local push_url="${matrix_url}/_matrix/push/v1/notify"
    local timestamp=$(date +%s)
    local expiry=$((timestamp + 86400))  # 24 hours

    # Build JSON payload
    local json_data="{\"matrix_url\":\"${matrix_url}\",\"rpc_url\":\"${rpc_url}\",\"ws_url\":\"${ws_url}\",\"push_gateway\":\"${push_url}\",\"server_name\":\"${hostname}\",\"expires_at\":${expiry}}"

    # Base64 encode (using jq if available, otherwise fallback)
    local encoded_data
    if command -v jq &> /dev/null; then
        encoded_data=$(echo -n "$json_data" | jq -sRr @uri | base64 -w 0 2>/dev/null || echo -n "$json_data" | base64 -w 0 2>/dev/null || echo -n "$json_data" | base64)
    else
        encoded_data=$(echo -n "$json_data" | base64 -w 0 2>/dev/null || echo -n "$json_data" | base64)
    fi

    # Create deep link
    local deep_link="armorclaw://config?d=${encoded_data}"
    local web_link="https://armorclaw.app/config?d=${encoded_data}"

    # Display configuration
    echo -e "${DIM}Configuration:${NC}"
    echo -e "  Server:    ${GREEN}${hostname}${NC}"
    echo -e "  Port:      ${GREEN}${port}${NC}"
    echo -e "  Matrix:    ${GREEN}${matrix_url}${NC}"
    echo -e "  RPC:       ${GREEN}${rpc_url}${NC}"
    echo -e "  WebSocket: ${GREEN}${ws_url}${NC}"
    echo -e "  Valid:     ${GREEN}24 hours${NC}"
    echo ""

    # Display deep link
    echo -e "${BOLD}Deep Link (copy to device):${NC}"
    echo -e "  ${CYAN}${deep_link}${NC}"
    echo ""

    # Display web link
    echo -e "${BOLD}Web Link (for browsers):${NC}"
    echo -e "  ${CYAN}${web_link}${NC}"
    echo ""

    # Try to display ASCII QR code if qrencode is available
    if command -v qrencode &> /dev/null; then
        echo -e "${BOLD}QR Code (scan with ArmorChat):${NC}"
        echo ""
        echo "$deep_link" | qrencode -t UTF8 2>/dev/null || {
            echo -e "${YELLOW}Note: QR encoding failed. Use the deep link above.${NC}"
        }
        echo ""
    else
        echo -e "${YELLOW}Tip: Install 'qrencode' to display ASCII QR code:${NC}"
        echo -e "  ${DIM}apt-get install qrencode${NC}"
        echo ""
        echo -e "${BOLD}Or generate QR manually:${NC}"
        echo -e "  ${DIM}echo '${deep_link}' | qrencode -t UTF8${NC}"
        echo ""
    fi

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

#=============================================================================
# Post-Setup Options
#=============================================================================

configure_security_tier() {
    # Enterprise profile: already set to maximum by configure_compliance()
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        return
    fi

    # Quick profile: use enhanced default
    if [ "$DEPLOY_PROFILE" = "quick" ]; then
        SECURITY_TIER="enhanced"
        return
    fi

    if [ "$NON_INTERACTIVE" = true ]; then
        SECURITY_TIER="${ARMORCLAW_SECURITY_TIER:-enhanced}"
        print_info "Security tier: $SECURITY_TIER"
        return
    fi

    echo -e "\n${YELLOW}Security Configuration${NC}"
    echo "Select security tier:"
    echo "  1) Essential - Basic isolation (development/testing)"
    echo "  2) Enhanced  - + Seccomp, network isolation (recommended)"
    echo "  3) Maximum   - + Audit logging, PII scrubbing (production)"
    echo ""

    local choice=$(prompt_input "Security tier" "2")

    case "$choice" in
        1) SECURITY_TIER="essential" ;;
        2) SECURITY_TIER="enhanced" ;;
        3) SECURITY_TIER="maximum" ;;
        *) SECURITY_TIER="enhanced" ;;
    esac

    print_info "Security tier set to: $SECURITY_TIER"
}

offer_post_setup_options() {
    if [ "$NON_INTERACTIVE" = true ]; then
        return
    fi

    echo ""
    echo -e "${CYAN}Post-Setup Options${NC}"
    echo ""

    # Offer to install additional apps
    if prompt_yes_no "Install ArmorTerminal for terminal access?" "n"; then
        print_info "ArmorTerminal will be available after bridge starts"
        print_info "Configure with: RPC URL = http://YOUR_IP:8443/rpc"
        INSTALL_ARMORTERMINAL=true
    fi

    # Offer hardening
    if [ "$SECURITY_TIER" != "essential" ]; then
        print_info "Security hardening ($SECURITY_TIER tier) will be applied on bridge start"
    fi
}

#=============================================================================
# Main
#=============================================================================

main() {
    # CRITICAL: Reset terminal state FIRST before any output
    # This handles the case where Go TUI wizard crashed and left terminal corrupted
    reset_terminal_full

    # Initialize logging first
    init_logging
    log_section "ArmorClaw Container Setup Starting"
    log_info "Setup version: 0.3.6"
    log_info "Debug mode: $DEBUG"

    # Enable debug tracing if requested
    if [ "$DEBUG" = true ]; then
        set -x
        log_debug "Debug tracing enabled"
    fi

    # Check for --from-wizard flag (Huh? TUI wizard output)
    FROM_WIZARD=false
    if [ "$1" = "--from-wizard" ] && [ -n "$2" ]; then
        load_wizard_json "$2"
        shift 2
    fi

    print_header

    # Run preflight checks before starting setup
    if ! preflight_checks; then
        print_error "Preflight checks failed. Fix issues above and retry."
        exit 1
    fi

    # Verify required tools are available
    check_required_tools

    # Initialize progress tracking (10 steps for full setup)
    init_progress 10

    # If not from wizard, check env vars and run interactive prompts
    if [ "$FROM_WIZARD" != true ]; then
        # Check for environment variables
        check_env_vars

        # Select deployment profile (Step 0)
        select_profile

        # Apply auto-defaults for quick profile
        if [ "$DEPLOY_PROFILE" = "quick" ] && [ "$NON_INTERACTIVE" != true ]; then
            apply_quick_defaults
        fi
    fi

    # Create directories
    next_step "Creating directories"
    create_directories

    # Generate self-signed SSL certificate (default)
    next_step "Generating SSL certificate"
    generate_self_signed_cert

    # Check Docker socket
    check_docker_socket

    # Run configuration steps (profile-aware)
    next_step "Configuring Matrix"
    configure_matrix      # Quick: auto-configured | Enterprise: Step 1/6

    next_step "Configuring API provider"
    configure_api          # Quick: Step 1/2       | Enterprise: Step 2/6

    # Enterprise-only: compliance configuration (Step 3/6)
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        next_step "Configuring compliance settings"
        configure_compliance
    fi

    # Collect admin credentials
    next_step "Setting up admin user"
    configure_admin_user   # Quick: Step 2/2       | Enterprise: Step 4/6

    # Enterprise-only: bridge config (Step 5/6)
    next_step "Configuring bridge"
    configure_bridge

    # Set security tier (auto for both profiles)
    next_step "Configuring security tier"
    configure_security_tier

    # Persist admin user info for quickstart.sh to auto-claim OWNER role
    echo "${ADMIN_USER}" > "$DATA_DIR/.admin_user"
    echo "${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}" >> "$DATA_DIR/.admin_user"
    chmod 600 "$DATA_DIR/.admin_user"

    # Save admin password to temp file for recovery (P2 improvement)
    if [ -n "${ADMIN_PASSWORD:-}" ]; then
        echo "${ADMIN_PASSWORD}" > "$DATA_DIR/.admin_password"
        chmod 600 "$DATA_DIR/.admin_password"
        log_debug "Admin password saved to $DATA_DIR/.admin_password"
    fi

    next_step "Validating and writing configuration"
    validate_config_vars
    write_config
    initialize_keystore

    # Start Matrix if available
    next_step "Starting Matrix stack"
    start_matrix_stack

    # Register bridge + admin users on Conduit (requires Matrix stack running)
    if [ "$MATRIX_AVAILABLE" = true ]; then
        # Wait for Matrix to be ready before attempting registration
        local matrix_url="${ARMORCLAW_MATRIX_HOMESERVER_URL:-http://localhost:6167}"
        if ! wait_for_matrix "$matrix_url" 30; then
            print_warning "Matrix not ready - will attempt registration anyway"
        fi

        next_step "Registering users and creating rooms"
        register_users

        # Create bridge management room with admin at power level 100
        setup_bridge_room
    else
        print_warning "Matrix homeserver not available — skipping user registration and room setup"
        print_warning "Run 'create-matrix-admin.sh' manually after Matrix is running"
    fi

    # Offer post-setup options (enterprise only)
    if [ "$DEPLOY_PROFILE" = "enterprise" ]; then
        offer_post_setup_options
    fi

    # Show progress completion
    complete_progress

    # Show summary
    final_summary

    # Mark setup as succeeded (disables cleanup trap)
    SETUP_SUCCEEDED=true
}

# Run main
main "$@"
