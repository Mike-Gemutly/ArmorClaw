#!/bin/bash
# Decrypt test files with test secrets
# Usage: decrypt-test-secrets.sh

set -e

ENCRYPTION_KEY="${ARMORCLAW_TEST_DECRYPT_KEY:-ARMORCLAW_TEST_DECRYPT_KEY}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Decrypting test files..."

if [[ -f "$SCRIPT_DIR/bridge/pkg/pii/scrubber_test.go.enc" ]]; then
    openssl enc -aes-256-cbc -d -pbkdf2 -in "$SCRIPT_DIR/bridge/pkg/pii/scrubber_test.go.enc" \
        -out "$SCRIPT_DIR/bridge/pkg/pii/scrubber_test.go" \
        -k "$ENCRYPTION_KEY"
    echo "✅ Decrypted: bridge/pkg/pii/scrubber_test.go"
else
    echo "⚠️  Encrypted file not found: bridge/pkg/pii/scrubber_test.go.enc"
    exit 1
fi

echo "✅ All test files decrypted"
