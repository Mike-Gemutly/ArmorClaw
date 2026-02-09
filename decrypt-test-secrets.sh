#!/bin/bash
# Decrypt test files with test secrets
# Usage: decrypt-test-secrets.sh

set -e

echo "🔐 Decrypting ArmorClaw test secrets..."
./scripts/decrypt-test.sh
