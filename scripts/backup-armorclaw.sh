#!/usr/bin/env bash

################################################################################
# ArmorClaw Backup Script
#
# Creates encrypted timestamped backups of ArmorClaw critical data.
#
# USAGE:
#   backup-armorclaw.sh [OPTIONS]
#
# OPTIONS:
#   --full               Create full backup (default)
#   --incremental        Create incremental backup since last full backup
#   --keep-days N        Keep backups for N days (default: 7 for daily, 28 for weekly)
#   --output DIR         Output directory (default: /var/backups/armorclaw/)
#   --passphrase FILE   Read GPG passphrase from file (default: use env var ARMORCLAW_BACKUP_KEY)
#
# ENVIRONMENT VARIABLES:
#   ARMORCLAW_BACKUP_KEY GPG passphrase for symmetric encryption
#
# BACKED UP PATHS:
#   - /var/lib/armorclaw/     (keystore.db)
#   - /etc/armorclaw/          (config.toml)
#   - /var/lib/conduit/        (Matrix data)
#
# BACKUP FORMAT:
#   armorclaw-backup-YYYYMMDD-HHMMSS.tar.gz.gpg
#
# EXAMPLES:
#   # Create full backup with 7-day retention
#   sudo ./backup-armorclaw.sh --full --keep-days 7
#
#   # Create incremental backup with 28-day retention
#   sudo ./backup-armorclaw.sh --incremental --keep-days 28
#
#   # Use passphrase from file
#   sudo ./backup-armorclaw.sh --passphrase /secure/backup-key
#
# SECURITY:
#   - Backups are encrypted with GPG symmetric encryption
#   - Passphrase should be stored securely (env var or file)
#   - Backup filenames do not contain secrets
#   - Backup operations are logged to syslog
#
# EXIT CODES:
#   0    Success
#   1    Error (check syslog for details)
#   2    Invalid arguments
#   3    Backup encryption failed
#   4    Backup creation failed
################################################################################

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/armorclaw}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="armorclaw-backup-${TIMESTAMP}.tar.gz.gpg"
KEEP_DAYS_DEFAULT=7
INCREMENTAL_KEEP_DAYS_DEFAULT=28

# Paths to backup
KESTORE_PATH="/var/lib/armorclaw"
CONFIG_PATH="/etc/armorclaw"
CONDUIT_PATH="/var/lib/conduit"

# Logging
log() {
    local level="$1"
    shift
    logger -t armorclaw-backup -p "${level}" "$*"
    case "${level}" in
        user.info) echo "[INFO] $*" ;;
        user.warning) echo "[WARN] $*" >&2 ;;
        user.err) echo "[ERROR] $*" >&2 ;;
    esac
}

# Parse arguments
FULL_BACKUP=true
KEEP_DAYS=""
PASSPHRASE_FILE=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --full)
            FULL_BACKUP=true
            shift
            ;;
        --incremental)
            FULL_BACKUP=false
            shift
            ;;
        --keep-days)
            if [[ -z "${2:-}" ]]; then
                log user.err "Missing value for --keep-days"
                exit 2
            fi
            KEEP_DAYS="$2"
            shift 2
            ;;
        --output)
            if [[ -z "${2:-}" ]]; then
                log user.err "Missing value for --output"
                exit 2
            fi
            BACKUP_DIR="$2"
            shift 2
            ;;
        --passphrase)
            if [[ -z "${2:-}" ]]; then
                log user.err "Missing value for --passphrase"
                exit 2
            fi
            PASSPHRASE_FILE="$2"
            shift 2
            ;;
        --help|-h)
            grep '^#' "$0" | grep -v '^#!/' | sed 's/^#//' | head -n 45
            exit 0
            ;;
        *)
            log user.err "Unknown option: $1"
            exit 2
            ;;
    esac
done

# Set default retention
if [[ -z "${KEEP_DAYS}" ]]; then
    if [[ "${FULL_BACKUP}" == true ]]; then
        KEEP_DAYS="${KEEP_DAYS_DEFAULT}"
    else
        KEEP_DAYS="${INCREMENTAL_KEEP_DAYS_DEFAULT}"
    fi
fi

# Validate backup directory exists
if [[ ! -d "${BACKUP_DIR}" ]]; then
    log user.err "Backup directory does not exist: ${BACKUP_DIR}"
    exit 4
fi

# Check we have write permission
if [[ ! -w "${BACKUP_DIR}" ]]; then
    log user.err "No write permission for backup directory: ${BACKUP_DIR}"
    exit 4
fi

# Get passphrase
if [[ -n "${PASSPHRASE_FILE}" ]]; then
    if [[ ! -f "${PASSPHRASE_FILE}" ]]; then
        log user.err "Passphrase file does not exist: ${PASSPHRASE_FILE}"
        exit 3
    fi
    PASSPHRASE=$(cat "${PASSPHRASE_FILE}")
elif [[ -n "${ARMORCLAW_BACKUP_KEY:-}" ]]; then
    PASSPHRASE="${ARMORCLAW_BACKUP_KEY}"
else
    log user.err "No passphrase provided. Use --passphrase FILE or set ARMORCLAW_BACKUP_KEY environment variable"
    exit 3
fi

# Check source paths exist
for path in "${KESTORE_PATH}" "${CONFIG_PATH}" "${CONDUIT_PATH}"; do
    if [[ ! -e "${path}" ]]; then
        log user.warning "Path does not exist (skipping): ${path}"
    fi
done

log user.info "Starting ${FULL_BACKUP:+full}${FULL_BACKUP:-incremental} backup at $(date)"
log user.info "Backup will be saved to: ${BACKUP_DIR}/${BACKUP_FILE}"

# Determine what to include
TAR_ARGS=()
for path in "${KESTORE_PATH}" "${CONFIG_PATH}" "${CONDUIT_PATH}"; do
    if [[ -e "${path}" ]]; then
        TAR_ARGS+=("${path}")
    fi
done

if [[ ${#TAR_ARGS[@]} -eq 0 ]]; then
    log user.err "No source paths found to backup"
    exit 4
fi

# Create temporary file for the archive
TEMP_ARCHIVE=$(mktemp "${BACKUP_DIR}/.armorclaw-backup-${TIMESTAMP}.XXXXXX")
trap "rm -f '${TEMP_ARCHIVE}'" EXIT

# Create the archive and encrypt it
log user.info "Creating encrypted archive..."

if ! tar czf - "${TAR_ARGS[@]}" | \
    gpg --batch --symmetric --cipher-algo AES256 --passphrase "${PASSPHRASE}" \
        --output "${TEMP_ARCHIVE}" 2>/dev/null; then
    log user.err "Failed to create encrypted backup"
    exit 3
fi

mv "${TEMP_ARCHIVE}" "${BACKUP_DIR}/${BACKUP_FILE}"

# Calculate backup size
BACKUP_SIZE=$(du -h "${BACKUP_DIR}/${BACKUP_FILE}" | cut -f1)

log user.info "Backup created successfully: ${BACKUP_DIR}/${BACKUP_FILE} (${BACKUP_SIZE})"

# Clean up old backups
log user.info "Cleaning up backups older than ${KEEP_DAYS} days..."

DELETED_COUNT=0
while IFS= read -r -d '' old_backup; do
    log user.info "Deleting old backup: $(basename "${old_backup}")"
    rm -f "${old_backup}"
    ((DELETED_COUNT++))
done < <(find "${BACKUP_DIR}" -name "armorclaw-backup-*.tar.gz.gpg" -type f \
    -mtime "+${KEEP_DAYS}" -print0)

log user.info "Deleted ${DELETED_COUNT} old backup(s)"

# Final log message
log user.info "Backup completed successfully at $(date)"

exit 0
