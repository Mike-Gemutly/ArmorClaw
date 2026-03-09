#!/usr/bin/env bash
# ArmorClaw Stage-1 Full Installer
# Version: 5.0.0
# Idempotent: Yes
# Safe to re-run: Yes
# =============================================================================
#
# Note: This script is typically launched by install.sh (Stage-0 bootstrap).
# It handles the actual system discovery, sudo elevation, and configuration.
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

########################################
# Helpers
########################################

# Helper for interactive prompts (handles curl | bash and non-interactive envs)
prompt_read() {
    if [ -t 0 ] || [ -c /dev/tty ]; then
        read "$@" < /dev/tty
    fi
}

print_header() {
    clear 2>/dev/null || true
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Installer${NC}                             ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}            ${BOLD}Version: ${VERSION}${NC}                                  ${CYAN}║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "\n${BLUE}▶${NC} ${BOLD}$1${NC}"
    echo -e "${BLUE}  ─────────────────────────────────────${NC}"
}

print_success() {
    echo -e "  ${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "  ${RED}✗${NC} ${BOLD}ERROR:${NC} $1" >&2
}

print_warning() {
    echo -e "  ${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "  ${CYAN}ℹ${NC} $1"
}

print_done() {
    echo -e "  ${GREEN}✓${NC} $1"
}

show_spinner() {
    local pid=$1
    local message="$2"
    local spin='-\|/'
    local i=0
    while kill -0 "$pid" 2>/dev/null; do
        i=$(( (i+1) % 4 ))
        printf "\r  ${YELLOW}⏳${NC} $message... ${spin:$i:1}"
        sleep .2
    done
    printf "\r"
}

log() {
  print_info "$*"
}

fail() {
  print_error "$*"
  exit 1
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

########################################
# Configuration
########################################

REPO="Gemutly/ArmorClaw"
VERSION="${VERSION:-main}"
INSTALL_MODE="${INSTALL_MODE:-quick}"

BASE_URL="https://raw.githubusercontent.com/${REPO}/${VERSION}/deploy"

LOG_DIR="/var/log/armorclaw"
LOG_FILE="${LOG_DIR}/install.log"
LOCKFILE="/tmp/armorclaw-install.lock"

CONDUIT_VERSION="${CONDUIT_VERSION:-latest}"
CONDUIT_IMAGE="${CONDUIT_IMAGE:-matrixconduit/matrix-conduit:$CONDUIT_VERSION}"
INSTALLER_VERSION="2.0"

########################################
# Prerequisite Checks
########################################

check_prereqs() {
    command -v flock >/dev/null 2>&1 || {
        echo "ERROR: flock not installed" >&2
        exit 1
    }

    command -v docker >/dev/null 2>&1 || {
        # Docker might be installed later, but we need curl at least
        command -v curl >/dev/null 2>&1 || {
            echo "ERROR: curl not installed" >&2
            exit 1
        }
    }

    command -v tee >/dev/null 2>&1 || {
        echo "ERROR: tee not installed" >&2
        exit 1
    }
}

########################################
# Docker Compose Detection
########################################

detect_docker_compose() {
    if docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker-compose"
    else
        DOCKER_COMPOSE="docker compose" # Fallback/Assumption
    fi
    export DOCKER_COMPOSE
}

########################################
# Detect OS
########################################

detect_os() {
  case "$(uname -s)" in
    Linux*)   OS="linux" ;;
    Darwin*)  OS="darwin" ;;
    *) fail "Unsupported OS: $(uname -s)" ;;
  esac
  print_done "OS: $OS"
}

########################################
# Detect Architecture
########################################

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) ARCH="amd64" ;;
    arm64 | aarch64) ARCH="arm64" ;;
    *) fail "Unsupported architecture: $(uname -m)" ;;
  esac
  print_done "Arch: $ARCH"
}

########################################
# Sudo Handling
#######################################

setup_sudo() {
  if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
    print_warning "Running as root is not recommended. Consider running as a normal user."
  else
    if command_exists sudo; then
      SUDO="sudo"
      print_done "Sudo detected (elevation will be used when needed)"
    else
      fail "This installer requires root or sudo."
    fi
  fi
}

########################################
# Check Dependencies
########################################

check_dependencies() {
  if ! command_exists curl; then
    fail "curl is required but not installed."
  fi
  print_done "Dependencies checked"
}

########################################
# Docker Install (optional)
########################################

ensure_docker() {
  print_step "Checking Docker Environment"
  if command_exists docker; then
    print_done "Docker already installed"
    return
  fi

  print_info "Docker not found. Installing..."

  # Check if running interactively
  if [ -t 0 ] || [ -c /dev/tty ]; then
    echo -ne "  ${CYAN}Install Docker automatically? [Y/n]${NC}: "
    prompt_read -r ans
    ans=${ans:-Y}
  else
    ans="Y"
    print_info "Non-interactive mode: auto-installing Docker"
  fi

  if [[ "$ans" =~ ^[Yy]$ ]]; then
    curl -fsSL https://get.docker.com | sh >/var/log/armorclaw-docker-install.log 2>&1 &
    show_spinner $! "Installing Docker"
    wait $!
    
    $SUDO systemctl enable docker >/dev/null 2>&1 || true
    $SUDO systemctl start docker >/dev/null 2>&1 || true
    print_success "Docker installed successfully"
    wait_for_docker
  else
    fail "Docker is required. Install Docker manually and try again."
  fi
}

########################################
# Temp Workspace
########################################

wait_for_docker() {
  print_info "Waiting for Docker daemon..."
  for ((i=1;i<=10;i++)); do
    if docker info >/dev/null 2>&1 && docker ps >/dev/null 2>&1; then
      print_done "Docker daemon ready"
      return 0
    fi
    sleep 2
  done
  fail "Docker failed to start within 20 seconds"
}

create_workspace() {
  WORK_DIR=$(mktemp -d)
  print_info "Created temp workspace: $WORK_DIR"

  cleanup() {
    rm -rf "$WORK_DIR"
    print_info "Cleaned up temp workspace"
  }

  trap cleanup EXIT
}

########################################
# Download Script
########################################

download_script() {
  local file="$1"
  local url="${BASE_URL}/${file}"
  local dest="${WORK_DIR}/${file}"

  curl -fsSL "$url" -o "$dest" >/dev/null 2>&1 &
  show_spinner $! "Downloading ${file}"
  wait $!

  if [[ $? -ne 0 ]]; then
      fail "Failed downloading ${file}"
  fi

  chmod +x "$dest"
}

########################################
# Run Setup Script
########################################

run_setup() {
  export REPO VERSION
  export ARMORCLAW_API_KEY ARMORCLAW_ADMIN_USERNAME ARMORCLAW_ADMIN_PASSWORD
  export DOCKER_COMPOSE CONDUIT_VERSION CONDUIT_IMAGE

  case "$INSTALL_MODE" in
    quick)
      download_script setup-quick.sh
      print_step "Running Quickstart Setup"
      exec bash "$WORK_DIR/setup-quick.sh" "$@"
      ;;
    matrix)
      download_script setup-matrix.sh
      print_step "Running Matrix Setup"
      exec bash "$WORK_DIR/setup-matrix.sh" "$@"
      ;;
    *)
      fail "Unknown INSTALL_MODE: $INSTALL_MODE. Use 'quick' or 'matrix'"
      ;;
  esac
}

########################################
# Main
########################################

main() {
  check_prereqs
  detect_docker_compose

  # Start installer logic
  exec 200>"$LOCKFILE"
  flock -n 200 || {
    echo "ERROR: installer already running" >&2
    exit 1
  }
  
  # Setup logging
  mkdir -p "$LOG_DIR" 2>/dev/null || LOG_DIR="/tmp/armorclaw"
  LOG_FILE="${LOG_DIR}/install.log"
  exec > >(tee -a "$LOG_FILE") 2>&1

  print_header

  print_info "ArmorClaw installer (Stage-1) started"
  print_info "Version: 5.0.0"
  print_info "Log file: $LOG_FILE"
  print_info "Detected Docker: $(docker --version 2>/dev/null || echo unavailable)"
  print_info "Conduit image: $CONDUIT_IMAGE"

  print_step "System Discovery"
  detect_os
  detect_arch
  setup_sudo
  check_dependencies

  create_workspace

  ensure_docker

  run_setup

  print_step "Installation Result"
  print_success "Installation complete!"
}

main "$@"
