#!/bin/bash
echo "BASH_SOURCE[0]: ${BASH_SOURCE[0]}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "SCRIPT_DIR: $SCRIPT_DIR"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
echo "PROJECT_ROOT: $PROJECT_ROOT"
echo "Checking .env at PROJECT_ROOT/.env: $(test -f "$PROJECT_ROOT/.env" && echo "FOUND" || echo "NOT FOUND")"
