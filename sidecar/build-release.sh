#!/bin/bash
set -euo pipefail

# Check prerequisites
for cmd in cmake clang cargo; do
    command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: $cmd not found. Install: apt-get install -y cmake clang"; exit 1; }
done

cd "$(dirname "$0")"
echo "Building armorclaw-sidecar (release)..."
cargo build --release --bin armorclaw-sidecar
echo "Build complete: target/release/armorclaw-sidecar"
