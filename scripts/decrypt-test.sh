#!/bin/bash
# Decrypt ArmorClaw test secrets
# Usage: ./scripts/decrypt-test.sh [file-to-decrypt]

set -e

ENCRYPTION_KEY="${ARMORCLAW_TEST_DECRYPT_KEY:-ARMORCLAW_TEST_DECRYPT_KEY}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🔐 ArmorClaw Test Secrets Decryptor"
echo ""

# If no argument, decrypt all known encrypted files
if [[ -z "$1" ]]; then
    echo "Decrypting all test files..."
    
    # Decrypt scrubber test
    if [[ -f "$SCRIPT_DIR/bridge/pkg/pii/scrubber_test.go.enc" ]]; then
        openssl enc -aes-256-cbc -d -pbkdf2 \
            -in "$SCRIPT_DIR/bridge/pkg/pii/scrubber_test.go.enc" \
            -out "$SCRIPT_DIR/bridge/pkg/pii/scrubber_test.go" \
            -k "$ENCRYPTION_KEY"
        echo "✅ bridge/pkg/pii/scrubber_test.go"
    else
        echo "⚠️  bridge/pkg/pii/scrubber_test.go.enc not found"
    fi
    
    echo ""
    echo "✅ All test files decrypted"
else
    # Decrypt specific file
    INPUT_FILE="$1"
    OUTPUT_FILE="${INPUT_FILE%.enc}"
    
    if [[ ! -f "$INPUT_FILE" ]]; then
        echo "❌ File not found: $INPUT_FILE"
        exit 1
    fi
    
    if [[ "$INPUT_FILE" == "$OUTPUT_FILE" ]]; then
        echo "❌ File must have .enc extension"
        exit 1
    fi
    
    echo "Decrypting: $INPUT_FILE -> $OUTPUT_FILE"
    openssl enc -aes-256-cbc -d -pbkdf2 \
        -in "$INPUT_FILE" \
        -out "$OUTPUT_FILE" \
        -k "$ENCRYPTION_KEY"
    
    echo "✅ Decrypted: $OUTPUT_FILE"
fi
