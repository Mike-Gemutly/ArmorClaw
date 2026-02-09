#!/bin/bash
# Encrypt ArmorClaw test secrets
# Usage: ./scripts/encrypt-test.sh [file-to-encrypt]

set -e

ENCRYPTION_KEY="${ARMORCLAW_TEST_DECRYPT_KEY:-ARMORCLAW_TEST_DECRYPT_KEY}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🔒 ArmorClaw Test Secrets Encryptor"
echo ""

if [[ -z "$1" ]]; then
    echo "Usage: $0 <file-to-encrypt>"
    echo ""
    echo "Example:"
    echo "  $0 bridge/pkg/pii/scrubber_test.go"
    exit 1
fi

INPUT_FILE="$1"

if [[ ! -f "$INPUT_FILE" ]]; then
    echo "❌ File not found: $INPUT_FILE"
    exit 1
fi

if [[ "$INPUT_FILE" == *.enc ]]; then
    echo "❌ File already encrypted (.enc extension)"
    exit 1
fi

OUTPUT_FILE="${INPUT_FILE}.enc"

echo "Encrypting: $INPUT_FILE -> $OUTPUT_FILE"
openssl enc -aes-256-cbc -salt -pbkdf2 \
    -in "$INPUT_FILE" \
    -out "$OUTPUT_FILE" \
    -k "$ENCRYPTION_KEY"

echo "✅ Encrypted: $OUTPUT_FILE"
echo ""
echo "⚠️  Remember to add $OUTPUT_FILE to git and remove $INPUT_FILE from git history"
