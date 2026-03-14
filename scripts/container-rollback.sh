#!/bin/bash
# =============================================================================
# Container Rollback Mechanism
# Purpose: Rollback containers to previous version using Docker tags
# Version: 1.0.0
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
LOG_FILE="/var/log/armorclaw/rollback.log"
DRY_RUN=false

# Service definitions (container_name: image_name)
declare -A SERVICES=(
    ["bridge"]="armorclaw/bridge"
    ["matrix"]="matrixconduit/matrix-conduit"
    ["coturn"]="coturn/coturn"
    ["sygnal"]="matrixdotorg/sygnal"
    ["caddy"]="caddy"
)

# Container name mapping
declare -A CONTAINER_NAMES=(
    ["bridge"]="armorclaw-bridge"
    ["matrix"]="armorclaw-matrix"
    ["coturn"]="armorclaw-coturn"
    ["sygnal"]="armorclaw-sygnal"
    ["caddy"]="armorclaw-caddy"
)

# =============================================================================
# Logging Functions
# =============================================================================

log() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "$1"
    if [[ "$DRY_RUN" == false ]]; then
        mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
        echo "[$timestamp] $1" >> "$LOG_FILE" 2>/dev/null || true
    fi
}

log_info() {
    log "${BLUE}[INFO]${NC} $1"
}

log_success() {
    log "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    log "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    log "${RED}[ERROR]${NC} $1"
}

# =============================================================================
# Help and Usage
# =============================================================================

show_help() {
    cat <<EOF
Container Rollback Mechanism
============================

Purpose: Rollback containers to previous version using Docker tags

Usage: $0 <command> [service] [options]

Commands:
  tag-current-as-prev [service]   Tag current images as :prev
                                   If no service specified, tags all services
  rollback [service]              Rollback to previous version
                                   If no service specified, rolls back all
  status                          Show current tag status for all services
  help, --help                    Show this help message

Options:
  --dry-run                       Preview changes without executing

Service Names:
  bridge    - ArmorClaw Bridge
  matrix    - Matrix Conduit
  coturn    - Coturn TURN Server
  sygnal    - Sygnal Push Gateway
  caddy     - Caddy Reverse Proxy
  all       - All services (default if none specified)

Examples:
  # Tag all current versions as prev before deploy
  $0 tag-current-as-prev

  # Tag specific service
  $0 tag-current-as-prev bridge

  # Rollback all services
  $0 rollback

  # Rollback specific service
  $0 rollback bridge

  # Preview rollback without executing
  $0 rollback all --dry-run

  # Check current status
  $0 status

Important Notes:
  - Only containers are rolled back, NOT configs or volumes
  - Manual trigger only - no automatic rollback on failure
  - Each service is rolled back independently

EOF
}

# =============================================================================
# Utility Functions
# =============================================================================

check_docker() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &>/dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
}

get_current_image() {
    local service="$1"
    local container="${CONTAINER_NAMES[$service]}"
    
    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        docker inspect --format='{{.Config.Image}}' "$container" 2>/dev/null || echo "unknown"
    else
        echo "not-running"
    fi
}

tag_image() {
    local image="$1"
    local from_tag="$2"
    local to_tag="$3"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "Would tag: ${image}:${from_tag} -> ${image}:${to_tag}"
    else
        if docker tag "${image}:${from_tag}" "${image}:${to_tag}" 2>/dev/null; then
            log_success "Tagged: ${image}:${from_tag} -> ${image}:${to_tag}"
        else
            log_warning "Could not tag ${image}:${from_tag} (image may not exist)"
        fi
    fi
}

# =============================================================================
# Commands
# =============================================================================

cmd_tag_current_as_prev() {
    local service="$1"
    
    log_info "Tagging current images as :prev..."
    
    if [[ -n "$service" && "$service" != "all" ]]; then
        # Single service
        if [[ -n "${SERVICES[$service]}" ]]; then
            local image="${SERVICES[$service]}"
            tag_image "$image" "current" "prev"
        else
            log_error "Unknown service: $service"
            exit 1
        fi
    else
        # All services
        for svc in "${!SERVICES[@]}"; do
            local image="${SERVICES[$svc]}"
            tag_image "$image" "current" "prev"
        done
    fi
    
    log_success "Tag current as prev complete"
}

cmd_rollback() {
    local service="$1"
    
    log_info "Rolling back to previous version..."
    
    if [[ -n "$service" && "$service" != "all" ]]; then
        # Single service
        if [[ -n "${SERVICES[$service]}" ]]; then
            local image="${SERVICES[$service]}"
            local container="${CONTAINER_NAMES[$service]}"
            
            tag_image "$image" "prev" "current"
            
            if [[ "$DRY_RUN" == false ]]; then
                log_info "Restarting container: $container"
                docker restart "$container" 2>/dev/null || log_warning "Could not restart $container"
            else
                log_info "Would restart container: $container"
            fi
        else
            log_error "Unknown service: $service"
            exit 1
        fi
    else
        # All services
        for svc in "${!SERVICES[@]}"; do
            local image="${SERVICES[$svc]}"
            local container="${CONTAINER_NAMES[$svc]}"
            
            tag_image "$image" "prev" "current"
            
            if [[ "$DRY_RUN" == false ]]; then
                log_info "Restarting container: $container"
                docker restart "$container" 2>/dev/null || log_warning "Could not restart $container"
            else
                log_info "Would restart container: $container"
            fi
        done
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "Dry-run complete - no changes made"
    else
        log_success "Rollback complete"
    fi
}

cmd_status() {
    log_info "Current service status:"
    echo ""
    printf "%-12s %-25s %s\n" "SERVICE" "CONTAINER" "CURRENT IMAGE"
    echo "------------------------------------------------------------"
    
    for svc in bridge matrix coturn sygnal caddy; do
        local container="${CONTAINER_NAMES[$svc]}"
        local current=$(get_current_image "$svc")
        printf "%-12s %-25s %s\n" "$svc" "$container" "$current"
    done
    
    echo ""
    log_info "Note: :prev and :current tags are local Docker tags"
    log_info "Use 'docker images | grep <image>' to see available tags"
}

# =============================================================================
# Main Entry Point
# =============================================================================

main() {
    # Parse arguments
    local command=""
    local service=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            help|--help|-h)
                show_help
                exit 0
                ;;
            tag-current-as-prev|rollback|status)
                command="$1"
                shift
                ;;
            bridge|matrix|coturn|sygnal|caddy|all)
                service="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                echo ""
                show_help
                exit 1
                ;;
        esac
    done
    
    # Check Docker
    check_docker
    
    # Execute command
    case "$command" in
        tag-current-as-prev)
            cmd_tag_current_as_prev "$service"
            ;;
        rollback)
            cmd_rollback "$service"
            ;;
        status)
            cmd_status
            ;;
        "")
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main
main "$@"
