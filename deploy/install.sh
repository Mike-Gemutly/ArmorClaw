#!/usr/bin/env bash
# =============================================================================
# ArmorClaw Bootstrap Installer (Stage-0)
# Version: 1.4.2
# Purpose: Environment check, integrity/authenticity verification, Stage-1 exec.
# =============================================================================

set -euo pipefail

# Configuration
REPO="Gemutly/ArmorClaw"
VERSION="${VERSION:-main}"
INSTALLER="installer-v5.sh"
BASE_URL="https://raw.githubusercontent.com/$REPO/$VERSION/deploy"
SIGNING_KEY_FPR="A1482657223EAFE1C481B74A8F535F90685749E0"

# Colors for output
CYAN='\033[0;36m'
RED='\033[0;31m'
GREEN='\033[0;32m'
BOLD='\033[1m'
NC='\033[0m'

# Helper: Retry logic for network operations
retry() {
    local n=1
    local max=3
    local delay=2
    while true; do
        "$@" && return 0
        if (( n == max )); then
            return 1
        fi
        echo -e "  [armorclaw] Retrying in ${delay}s... ($n/$max)"
        sleep $delay
        ((n++))
    done
}

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Bootstrap Loader${NC} ${CYAN}║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# 1. Tool Verification
echo "[armorclaw] Verifying local environment..."
REQUIRED_TOOLS=(curl sha256sum gpg mktemp)
for tool in "${REQUIRED_TOOLS[@]}"; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo -e "${RED}ERROR:${NC} '$tool' is required but not installed."
        exit 1
    fi
done

# 2. Workspace Setup
TMP_DIR=$(mktemp -d)
INSTALL_PATH="$TMP_DIR/$INSTALLER"
CHECKSUM_PATH="$TMP_DIR/$INSTALLER.sha256"
SIG_PATH="$TMP_DIR/$INSTALLER.sig"
KEY_PATH="$TMP_DIR/armorclaw-signing-key.asc"
GNUPGHOME="$TMP_DIR/gnupg"

cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

# 3. Download Components (with TLS pinning and atomic moving)
echo "[armorclaw] Downloading components ($VERSION)..."
# Force modern TLS, HTTPS only, and strict error handling
CURL_BASE="curl --proto =https --tlsv1.2 --fail --silent --show-error --location"

retry $CURL_BASE "$BASE_URL/$INSTALLER" -o "$INSTALL_PATH.tmp" && mv "$INSTALL_PATH.tmp" "$INSTALL_PATH"
retry $CURL_BASE "$BASE_URL/$INSTALLER.sha256" -o "$CHECKSUM_PATH.tmp" && mv "$CHECKSUM_PATH.tmp" "$CHECKSUM_PATH"
retry $CURL_BASE "$BASE_URL/$INSTALLER.sig" -o "$SIG_PATH.tmp" && mv "$SIG_PATH.tmp" "$SIG_PATH"
retry $CURL_BASE "$BASE_URL/armorclaw-signing-key.asc" -o "$KEY_PATH.tmp" && mv "$KEY_PATH.tmp" "$KEY_PATH"

# 4. Verify Integrity (SHA256)
echo "[armorclaw] Verifying installer integrity..."
EXPECTED=$(cut -d ' ' -f1 "$CHECKSUM_PATH")
ACTUAL=$(sha256sum "$INSTALL_PATH" | awk '{print $1}')

if [[ "$EXPECTED" != "$ACTUAL" ]]; then
    echo -e "${RED}ERROR: SHA256 checksum mismatch!${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓ Checksum OK${NC}"

# 5. Verify Authenticity (GPG)
echo "[armorclaw] Verifying GPG signature..."
mkdir -p "$GNUPGHOME"
chmod 700 "$GNUPGHOME"

# Import key to temporary keyring
gpg --homedir "$GNUPGHOME" --batch --import "$KEY_PATH" >/dev/null 2>&1

# Verify fingerprint to prevent key replacement attacks
FPR_CHECK=$(gpg --homedir "$GNUPGHOME" --with-colons --fingerprint releases@armorclaw.ai | grep "^fpr" | cut -d: -f10)
if [[ "$FPR_CHECK" != "$SIGNING_KEY_FPR" ]]; then
    echo -e "${RED}ERROR: Unauthorized signing key detected!${NC}"
    echo "Expected: $SIGNING_KEY_FPR"
    echo "Actual:   $FPR_CHECK"
    exit 1
fi


# Verify signature
if ! gpg --homedir "$GNUPGHOME" --batch --verify "$SIG_PATH" "$INSTALL_PATH" >/dev/null 2>&1; then
    echo -e "${RED}ERROR: GPG signature verification failed!${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓ Signature Verified (ArmorClaw Release Signer)${NC}"

# 6. Execute Stage-1
chmod +x "$INSTALL_PATH"
echo ""
echo "[armorclaw] Launching Stage-1 installer..."
echo ""

# We use exec so that the Stage-1 installer takes over the PID
exec "$INSTALL_PATH" "$@"
