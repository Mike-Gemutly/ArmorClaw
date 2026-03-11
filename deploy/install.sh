#!/usr/bin/env bash
# =============================================================================
# ArmorClaw Bootstrap Installer (Stage-0)
# Version: 1.4.3
# Purpose: Environment check, integrity/authenticity verification, Stage-1 exec.
# =============================================================================

set -euo pipefail

# Configuration
REPO="Gemutly/ArmorClaw"
VERSION="${VERSION:-main}"
INSTALLER="installer-v5.sh"
BASE_URL="https://raw.githubusercontent.com/$REPO/$VERSION/deploy"
# Official Fingerprint: 573A 62B2 39F9 8A6B 98EF 917D 03FC 7E7C CF74 8504
SIGNING_KEY_FPR="55AD64228EF6B4A342DA480A09C43CFA8AC93062"

# Colors for output
CYAN='\033[0;36m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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
        ((n++)) || true
    done
}

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║${NC}            ${BOLD}ArmorClaw Bootstrap Loader v1.4.3${NC}          ${CYAN}║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# 1. Tool Verification
echo "[armorclaw] Verifying local environment..."
REQUIRED_TOOLS=(curl sha256sum gpg mktemp sed)
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
echo "[armorclaw] Downloading components (branch: $VERSION)..."
# Force modern TLS, HTTPS only, and strict error handling
CURL_BASE="curl --proto =https --tlsv1.2 --fail --silent --show-error --location --connect-timeout 10 --max-time 60"

retry $CURL_BASE "$BASE_URL/$INSTALLER" -o "$INSTALL_PATH.tmp"
retry $CURL_BASE "$BASE_URL/$INSTALLER.sha256" -o "$CHECKSUM_PATH.tmp"
retry $CURL_BASE "$BASE_URL/$INSTALLER.sig" -o "$SIG_PATH.tmp"
retry $CURL_BASE "$BASE_URL/armorclaw-signing-key.asc" -o "$KEY_PATH.tmp"

# Normalize line endings to LF before verification (not for .sig - it's binary)
sed -i 's/\r$//' "$INSTALL_PATH.tmp" "$CHECKSUM_PATH.tmp" "$KEY_PATH.tmp"

mv "$INSTALL_PATH.tmp" "$INSTALL_PATH"
mv "$CHECKSUM_PATH.tmp" "$CHECKSUM_PATH"
mv "$SIG_PATH.tmp" "$SIG_PATH"
mv "$KEY_PATH.tmp" "$KEY_PATH"

# 4. Verify Integrity (SHA256)
echo "[armorclaw] Verifying installer integrity (SHA256)..."
# Handle checksum files that might have paths
EXPECTED=$(awk '{print $1}' "$CHECKSUM_PATH")
ACTUAL=$(sha256sum "$INSTALL_PATH" | awk '{print $1}')

if [[ "$EXPECTED" != "$ACTUAL" ]]; then
    echo -e "${RED}ERROR: SHA256 checksum mismatch!${NC}"
    echo "Expected: $EXPECTED"
    echo "Actual:   $ACTUAL"
    exit 1
fi
echo -e "  ${GREEN}✓ Checksum OK${NC}"

# 5. Verify Authenticity (GPG)
echo "[armorclaw] Verifying GPG signature..."
mkdir -p "$GNUPGHOME"
chmod 700 "$GNUPGHOME"

# Import key to temporary keyring
if ! gpg --homedir "$GNUPGHOME" --batch --import "$KEY_PATH" > "$TMP_DIR/gpg_import.log" 2>&1; then
    echo -e "${RED}ERROR: Failed to import signing key!${NC}"
    cat "$TMP_DIR/gpg_import.log"
    exit 1
fi

# Verify fingerprint to prevent key replacement attacks
# We look for the fingerprint line in the key we just imported
FPR_CHECK=$(gpg --homedir "$GNUPGHOME" --with-colons --fingerprint releases@armorclaw.ai | grep "^fpr" | head -n1 | cut -d: -f10)
if [[ "${FPR_CHECK:-}" != "$SIGNING_KEY_FPR" ]]; then
    echo -e "${RED}ERROR: Unauthorized signing key detected!${NC}"
    echo "Expected: $SIGNING_KEY_FPR"
    echo "Actual:   ${FPR_CHECK:-MISSING}"
    exit 1
fi

# Verify signature
if ! gpg --homedir "$GNUPGHOME" --batch --verify "$SIG_PATH" "$INSTALL_PATH" > "$TMP_DIR/gpg_verify.log" 2>&1; then
    echo -e "${RED}ERROR: GPG signature verification failed!${NC}"
    echo "Details from GPG:"
    cat "$TMP_DIR/gpg_verify.log"
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
