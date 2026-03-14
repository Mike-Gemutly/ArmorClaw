#!/bin/bash
# =============================================================================
# Docker Build Error Visibility Script
# Purpose: Build failures show clear error messages with full context
# Supports: docker build, docker compose build, docker-compose build
# Version: 1.0.0
# =============================================================================

set -eo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

# Script information
SCRIPT_NAME="docker-build-with-errors.sh"
SCRIPT_VERSION="1.0.0"

# Default values
DOCKER_COMMAND="auto"
CONTEXT=""
DOCKERFILE="Dockerfile"
TARGET=""
BUILDKIT=1
QUIET=false

# =============================================================================
# Section 1: Color Output and Logging
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

show_banner() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}         ${BOLD}Docker Build Error Visibility${NC}                     ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}                Version ${SCRIPT_VERSION}                         ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

show_help() {
    cat <<EOF
Docker Build Error Visibility Script
=====================================

Purpose: Build failures show clear error messages with full context

Usage: $SCRIPT_NAME [OPTIONS]

Options:
  -c, --context CONTEXT     Build context directory (default: current dir)
  -f, --file DOCKERFILE     Dockerfile path (default: Dockerfile)
  -t, --target TARGET       Build target stage
  -q, --quiet               Suppress normal build output (only show errors)
  -n, --no-buildkit         Disable BuildKit
  -h, --help                Show this help message

Auto-Detection:
  --auto-detect             Auto-detect docker-compose vs docker (default)
  --docker-compose          Force docker-compose command
  --docker-compose-v2       Force docker compose (v2)
  --docker-build            Force docker build

Examples:
  # Auto-detect, use BuildKit
  $SCRIPT_NAME

  # Build with custom context
  $SCRIPT_NAME -c /path/to/project

  # Build with specific Dockerfile
  $SCRIPT_NAME -f Dockerfile.prod

  # Build with specific target
  $SCRIPT_NAME -t builder

  # Disable BuildKit
  $SCRIPT_NAME --no-buildkit

  # Quiet mode - only show errors
  $SCRIPT_NAME -q

  # Force docker-compose (legacy)
  $SCRIPT_NAME --docker-compose

  # Force docker build (newer)
  $SCRIPT_NAME --docker-build

Error Handling:
  - Shows full Docker output with --progress=plain
  - On failure: displays last 50 lines of build log
  - Clear error messages with context
  - Exit code 1 on failure, 0 on success

EOF
}

# =============================================================================
# Section 2: Argument Parsing
# =============================================================================

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                show_help
                exit 0
                ;;
            -c|--context)
                CONTEXT="$2"
                shift 2
                ;;
            -f|--file)
                DOCKERFILE="$2"
                shift 2
                ;;
            -t|--target)
                TARGET="--target $2"
                shift 2
                ;;
            -q|--quiet)
                QUIET=true
                shift
                ;;
            -n|--no-buildkit)
                BUILDKIT=0
                shift
                ;;
            --auto-detect)
                DOCKER_COMMAND="auto"
                shift
                ;;
            --docker-compose)
                DOCKER_COMMAND="docker-compose"
                shift
                ;;
            --docker-compose-v2)
                DOCKER_COMMAND="docker compose"
                shift
                ;;
            --docker-build)
                DOCKER_COMMAND="docker build"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                echo ""
                echo "Run '$SCRIPT_NAME --help' for usage information"
                exit 1
                ;;
        esac
    done
}

# =============================================================================
# Section 3: Docker Command Detection
# =============================================================================

detect_docker_command() {
    if [[ "$DOCKER_COMMAND" == "auto" ]]; then
        if command -v docker &>/dev/null && docker compose version &>/dev/null; then
            DOCKER_COMMAND="docker compose"
        elif command -v docker-compose &>/dev/null; then
            DOCKER_COMMAND="docker-compose"
        elif command -v docker &>/dev/null; then
            DOCKER_COMMAND="docker build"
        else
            log_error "No Docker command found"
            exit 1
        fi
        log_info "Using Docker command: $DOCKER_COMMAND"
    fi
}

# =============================================================================
# Section 4: Build Execution with Error Handling
# =============================================================================

run_docker_build() {
    local build_log=$(mktemp)

    log_info "Building Docker image..."
    echo "Build started at: $(date '+%Y-%m-%d %H:%M:%S')" > "$build_log"

    if [[ "$BUILDKIT" == 1 ]]; then
        log_info "BuildKit enabled"
    else
        log_warning "BuildKit disabled"
    fi

    if [[ "$QUIET" == true ]]; then
        log_info "Quiet mode enabled (errors only)"
    fi

    echo "========================================" >> "$build_log"
    echo "Build Configuration:" >> "$build_log"
    echo "  Docker command: $DOCKER_COMMAND" >> "$build_log"
    echo "  Context: ${CONTEXT:-current directory}" >> "$build_log"
    echo "  Dockerfile: $DOCKERFILE" >> "$build_log"
    echo "  Target: ${TARGET:-none}" >> "$build_log"
    echo "  BuildKit: $([ $BUILDKIT == 1 ] && echo 'enabled' || echo 'disabled')" >> "$build_log"
    echo "  Quiet: $([ $QUIET == true ] && echo 'enabled' || echo 'disabled')" >> "$build_log"
    echo "========================================" >> "$build_log"
    echo "" >> "$build_log"

    # Build command
    local build_cmd

    if [[ "$DOCKER_COMMAND" == "docker build" ]]; then
        build_cmd="docker build"
    elif [[ "$DOCKER_COMMAND" == "docker compose" ]]; then
        build_cmd="docker compose build"
    elif [[ "$DOCKER_COMMAND" == "docker-compose" ]]; then
        build_cmd="docker-compose build"
    fi

    # Construct the full command
    local full_cmd="$build_cmd"

    if [[ "$BUILDKIT" == 1 ]]; then
        full_cmd="$full_cmd --progress=plain"
    fi

    if [[ -n "$CONTEXT" ]]; then
        full_cmd="$full_cmd -f $DOCKERFILE $CONTEXT"
    else
        full_cmd="$full_cmd -f $DOCKERFILE ."
    fi

    if [[ -n "$TARGET" ]]; then
        full_cmd="$full_cmd $TARGET"
    fi

    if [[ "$QUIET" == true ]]; then
        full_cmd="$full_cmd --quiet 2>&1"
    else
        full_cmd="$full_cmd 2>&1"
    fi

    log_info "Executing: $build_cmd"

    # Execute the build and capture output
    if eval "$full_cmd" >> "$build_log" 2>&1; then
        local exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            rm -f "$build_log"
            log_success "Build completed successfully"
            return 0
        else
            rm -f "$build_log"
            return $exit_code
        fi
    else
        local exit_code=$?
        rm -f "$build_log"
        return $exit_code
    fi
}

show_build_failure() {
    local build_log=$1

    echo ""
    echo -e "${RED}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║${NC}         ${BOLD}Docker Build Failed${NC}                                  ${RED}║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    # Show last 50 lines of build log
    if [[ -f "$build_log" ]]; then
        echo -e "${YELLOW}Last 50 lines of build log:${NC}"
        echo -e "${MAGENTA}───────────────────────────────────────────────────────────────────${NC}"

        local tail_count=50
        local total_lines=$(wc -l < "$build_log" 2>/dev/null || echo "0")

        if [[ $total_lines -le $tail_count ]]; then
            tail -$total_lines "$build_log"
        else
            tail -$tail_count "$build_log"
        fi

        echo -e "${MAGENTA}───────────────────────────────────────────────────────────────────${NC}"
        echo ""

        # Show file location
        echo -e "${BLUE}[INFO]${NC} Full build log saved to: $build_log"
        echo -e "${BLUE}[INFO]${NC} To view full log, run: cat $build_log"
        echo ""
    fi

    # Show helpful troubleshooting tips
    echo -e "${YELLOW}Troubleshooting tips:${NC}"
    echo "  1. Check the Dockerfile syntax"
    echo "  2. Verify required dependencies are available"
    echo "  3. Ensure Docker daemon is running: docker info"
    echo "  4. Try: $DOCKER_COMMAND --progress=plain -f $DOCKERFILE ."
    echo ""
}

# =============================================================================
# Section 5: Main Entry Point
# =============================================================================

main() {
    # Parse arguments
    parse_arguments "$@"

    # Show banner
    show_banner

    # Detect Docker command
    detect_docker_command

    # Create temporary file for build log
    local build_log=$(mktemp)

    # Run the build
    if run_docker_build; then
        # Success - clean up and exit
        rm -f "$build_log"
        exit 0
    else
        # Failure - show error details
        local exit_code=$?
        show_build_failure "$build_log"
        rm -f "$build_log"
        exit $exit_code
    fi
}

# Run main function
main "$@"
