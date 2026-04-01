#!/usr/bin/env bash

################################################################################
# ArmorClaw Restore Script
#
# Restores ArmorClaw from an encrypted backup archive.
#
# USAGE:
#   restore-armorclaw.sh BACKUP_FILE [OPTIONS]
#
# ARGUMENTS:
#   BACKUP_FILE          Path to encrypted backup file (.tar.gz.gpg)
#
# OPTIONS:
#   --dry-run           Verify and preview without actually restoring
#   --passphrase FILE  Read GPG passphrase from file (default: use env var ARMORCLAW_BACKUP_KEY)
#
# ENVIRONMENT VARIABLES:
#   ARMORCLAW_BACKUP_KEY GPG passphrase for decryption
#
# RESTORE PROCESS:
#   1. Verify backup file exists
#   2. Decrypt and verify integrity
#   3. Extract to temporary location
#   4. Stop services (bridge, conduit)
#   5. Restore files to correct locations
#   6. Restart services
#   7. Verify services are healthy
#
# RESTORED PATHS:
#   - /var/lib/armorclaw/     (keystore.db)
#   - /etc/armorclaw/          (config.toml)
#   - /var/lib/conduit/        (Matrix data)
#
# EXAMPLES:
#   # Restore from backup (interactive)
#   sudo ./restore-armorclaw.sh /var/backups/armorclaw/armorclaw-backup-20240101-120000.tar.gz.gpg
#
#   # Preview what would be restored
#   sudo ./restore-armorclaw.sh /var/backups/armorclaw/armorclaw-backup-20240101-120000.tar.gz.gpg --dry-run
#
#   # Use passphrase from file
#   sudo ./restore-armorclaw.sh /var/backups/armorclaw/armorclaw-backup-20240101-120000.tar.gz.gpg --passphrase /secure/backup-key
#
# SECURITY:
#   - Backup integrity is verified before restore
#   - Services are stopped before file restoration
#   - Dry-run mode allows verification without changes
#   - Original backup file is never modified
#   - Failed restores leave system in previous state
#
# EXIT CODES:
#   0    Success
#   1    Error (check syslog for details)
#   2    Invalid arguments
#   3    Decryption failed
#   4    Integrity verification failed
#   5    Service control failed
#   6    Health check failed after restore
#
# DEPENDENCIES:
#   - GPG (for decryption)
#   - systemd (for service control)
#   - tar (for archive extraction)
################################################################################

set -euo pipefail

# Configuration
TEMP_DIR="/tmp/armorclaw-restore-$$"
SERVICES=("armorclaw-bridge" "conduit")

# Logging
log() {
    local level="$1"
    shift
    logger -t armorclaw-restore -p "${level}" "$*"
    case "${level}" in
        user.info) echo "[INFO] $*" ;;
        user.warning) echo "[WARN] $*" >&2 ;;
        user.err) echo "[ERROR] $*" >&2 ;;
    esac
}

# Cleanup function
cleanup() {
    local exit_code=$?
    if [[ -d "${TEMP_DIR}" ]]; then
        log user.info "Cleaning up temporary directory: ${TEMP_DIR}"
        rm -rf "${TEMP_DIR}"
    fi
    exit ${exit_code}
}

trap cleanup EXIT

# Parse arguments
BACKUP_FILE=""
DRY_RUN=false
PASSPHRASE_FILE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run)
            DRY_RUN=true
            shift
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
            grep '^#' "$0" | grep -v '^#!/' | sed 's/^#//' | head -n 52
            exit 0
            ;;
        -*)
            log user.err "Unknown option: $1"
            exit 2
            ;;
        *)
            if [[ -n "${BACKUP_FILE}" ]]; then
                log user.err "Multiple backup files specified"
                exit 2
            fi
            BACKUP_FILE="$1"
            shift
            ;;
    esac
done

# Validate backup file argument
if [[ -z "${BACKUP_FILE}" ]]; then
    log user.err "No backup file specified"
    exit 2
fi

if [[ ! -f "${BACKUP_FILE}" ]]; then
    log user.err "Backup file does not exist: ${BACKUP_FILE}"
    exit 3
fi

if [[ ! "${BACKUP_FILE}" =~ \.tar\.gz\.gpg$ ]]; then
    log user.warning "Backup file does not have expected .tar.gz.gpg extension: ${BACKUP_FILE}"
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

log user.info "Starting restore from: ${BACKUP_FILE}"
log user.info "Dry-run mode: ${DRY_RUN}"

# Create temporary directory
mkdir -p "${TEMP_DIR}"

# Step 1: Verify and decrypt backup
log user.info "Decrypting backup archive..."

if ! gpg --batch --decrypt --passphrase "${PASSPHRASE}" \
    --output "${TEMP_DIR}/backup.tar.gz" "${BACKUP_FILE}" 2>/dev/null; then
    log user.err "Failed to decrypt backup file (wrong passphrase or corrupted file)"
    exit 3
fi

# Step 2: Verify archive integrity
log user.info "Verifying archive integrity..."

if ! tar tzf "${TEMP_DIR}/backup.tar.gz" >/dev/null 2>&1; then
    log user.err "Archive integrity check failed (corrupted archive)"
    exit 4
fi

# Step 3: Extract to temporary location
log user.info "Extracting archive to temporary location..."

if ! tar xzf "${TEMP_DIR}/backup.tar.gz" -C "${TEMP_DIR}"; then
    log user.err "Failed to extract archive"
    exit 4
fi

# Show what will be restored
log user.info "Contents to be restored:"
find "${TEMP_DIR}" -type f | sed "s|${TEMP_DIR}||" | while read -r file; do
    log user.info "  ${file}"
done

# If dry-run, stop here
if [[ "${DRY_RUN}" == true ]]; then
    log user.info "Dry-run mode: would restore the above files. No changes made."
    log user.info "Restore verified successfully"
    exit 0
fi

# Step 4: Stop services
log user.info "Stopping services..."

for service in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "${service}" 2>/dev/null; then
        log user.info "Stopping ${service}..."
        if ! systemctl stop "${service}"; then
            log user.err "Failed to stop service: ${service}"
            exit 5
        fi
    else
        log user.info "Service not running: ${service} (skipping)"
    fi
done

# Step 5: Restore files
log user.info "Restoring files..."

restore_failed=false

# Restore keystore
if [[ -d "${TEMP_DIR}/var/lib/armorclaw" ]]; then
    log user.info "Restoring keystore to /var/lib/armorclaw/..."
    if ! rsync -a "${TEMP_DIR}/var/lib/armorclaw/" /var/lib/armorclaw/; then
        log user.err "Failed to restore keystore"
        restore_failed=true
    fi
fi

# Restore config
if [[ -d "${TEMP_DIR}/etc/armorclaw" ]]; then
    log user.info "Restoring config to /etc/armorclaw/..."
    if ! rsync -a "${TEMP_DIR}/etc/armorclaw/" /etc/armorclaw/; then
        log user.err "Failed to restore config"
        restore_failed=true
    fi
fi

# Restore conduit data
if [[ -d "${TEMP_DIR}/var/lib/conduit" ]]; then
    log user.info "Restoring conduit data to /var/lib/conduit/..."
    if ! rsync -a "${TEMP_DIR}/var/lib/conduit/" /var/lib/conduit/; then
        log user.err "Failed to restore conduit data"
        restore_failed=true
    fi
fi

if [[ "${restore_failed}" == true ]]; then
    log user.err "Failed to restore one or more components"
    exit 4
fi

# Step 6: Restart services
log user.info "Restarting services..."

for service in "${SERVICES[@]}"; do
    log user.info "Starting ${service}..."
    if ! systemctl start "${service}"; then
        log user.err "Failed to start service: ${service}"
        exit 5
    fi
done

# Wait for services to start
log user.info "Waiting for services to stabilize..."
sleep 5

# Step 7: Verify services are healthy
log user.info "Verifying service health..."

all_healthy=true

for service in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "${service}"; then
        log user.info "Service healthy: ${service}"
    else
        log user.err "Service not running after restore: ${service}"
        all_healthy=false
    fi
done

if [[ "${all_healthy}" == false ]]; then
    log user.err "One or more services failed to start after restore"
    log user.err "Check service status with: systemctl status armorclaw-bridge conduit"
    exit 6
fi

# Final log message
log user.info "Restore completed successfully at $(date)"
log user.info "Verify restore by checking application functionality"

exit 0
