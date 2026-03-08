#!/usr/bin/env bash
# =============================================================================
# ArmorClaw Production-Grade Installer
# Version: 1.0.0
# Idempotent: Yes
# Safe to re-run: Yes
# =============================================================================
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | INSTALL_MODE=matrix bash
#   curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | VERSION=v4.6.0 bash
#
# Environment Variables:
#   VERSION         - GitHub release tag (default: main)
#   INSTALL_MODE    - quick | matrix (default: quick)
#   ARMORCLAW_API_KEY - AI provider API key
#   ARMORCLAW_ADMIN_USERNAME - Custom admin username
# =============================================================================

set -euo pipefail

########################################
# Configuration
########################################

REPO="Gemutly/ArmorClaw"
VERSION="${VERSION:-main}"
INSTALL_MODE="${INSTALL_MODE:-quick}"

BASE_URL="https://raw.githubusercontent.com/${REPO}/${VERSION}/deploy"

########################################
# Helpers
########################################

log() {
  echo "[armorclaw-install] $*"
}

log_info() {
  echo "[armorclaw-install] INFO: $*"
}

log_warn() {
  echo "[armorclaw-install] WARN: $*"
}

log_error() {
  echo "[armorclaw-install] ERROR: $*" >&2
}

fail() {
  log_error "$*"
  exit 1
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
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
  log "Detected OS=$OS"
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
  log "Detected ARCH=$ARCH"
}

########################################
# Sudo Handling
########################################

setup_sudo() {
  if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
    log "Running as root"
  else
    if command_exists sudo; then
      SUDO="sudo"
      log "Using sudo for privileged operations"
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
  log "Dependency check passed"
}

########################################
# Docker Install (optional)
########################################

ensure_docker() {
  if command_exists docker; then
    log "Docker already installed"
    return
  fi

  log "Docker not found. Installing..."

  # Check if running interactively
  if [ -t 0 ]; then
    read -p "Install Docker automatically? [Y/n] " ans
    ans=${ans:-Y}
  else
    ans="Y"
    log "Non-interactive mode: auto-installing Docker"
  fi

  if [[ "$ans" =~ ^[Yy]$ ]]; then
    curl -fsSL https://get.docker.com | sh
    $SUDO systemctl enable docker || true
    $SUDO systemctl start docker || true
    log "Docker installed successfully"
  else
    fail "Docker is required. Install Docker manually and try again."
  fi
}

########################################
# Temp Workspace
########################################

create_workspace() {
  WORK_DIR=$(mktemp -d)
  log "Created temp workspace: $WORK_DIR"

  cleanup() {
    rm -rf "$WORK_DIR"
    log "Cleaned up temp workspace"
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

  log "Downloading ${file} from ${url}"

  curl -fsSL "$url" -o "$dest" || fail "Failed downloading ${file}"

  chmod +x "$dest"
}

########################################
# Run Setup Script
########################################

run_setup() {
  case "$INSTALL_MODE" in
    quick)
      download_script setup-quick.sh
      log "Running quickstart setup (bridge only)"
      "$WORK_DIR/setup-quick.sh"
      ;;
    matrix)
      download_script setup-matrix.sh
      log "Running Matrix setup (bridge + Conduit)"
      "$WORK_DIR/setup-matrix.sh"
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
  log "========================================"
  log "ArmorClaw Installer v${VERSION}"
  log "========================================"

  detect_os
  detect_arch
  setup_sudo
  check_dependencies

  create_workspace

  ensure_docker

  run_setup

  log "========================================"
  log "Installation complete!"
  log "========================================"
}

main "$@"
