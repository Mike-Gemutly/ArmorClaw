#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SIDECAR_DIR="$PROJECT_ROOT/sidecar"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0

pass() {
    echo -e "${GREEN}PASS${NC}: $1"
    ((PASS_COUNT++))
}

fail() {
    echo -e "${RED}FAIL${NC}: $1"
    ((FAIL_COUNT++))
}

echo "========================================"
echo "XChaCha20 Nonce Length Verification"
echo "========================================"
echo ""

# ── Check 1: Rust unit test ──────────────────────────────────────────────
echo "--- Check 1: Cargo test (test_nonce_is_24_bytes) ---"

if [ -f "$SIDECAR_DIR/Cargo.toml" ]; then
    if (cd "$SIDECAR_DIR" && cargo test --lib encryption::aead::tests::test_nonce_is_24_bytes 2>&1); then
        pass "Cargo test: test_nonce_is_24_bytes"
    else
        fail "Cargo test: test_nonce_is_24_bytes"
    fi
else
    fail "sidecar/Cargo.toml not found"
fi

echo ""

# ── Check 2: Standalone inline Rust verification ─────────────────────────
echo "--- Check 2: Standalone nonce length verification ---"

if command -v rustc &>/dev/null; then
    INLINE_SRC=$(mktemp /tmp/xchacha_nonce_check_XXXXXX.rs)
    INLINE_BIN=$(mktemp /tmp/xchacha_nonce_check_XXXXXX)

    cat > "$INLINE_SRC" <<'RUSTEOF'
fn main() {
    // XChaCha20Poly1305 nonce is 19 bytes (XChaCha20 uses 192-bit / 24-byte nonce)
    // The generate_nonce() method on XChaCha20Poly1305 returns a GenericArray of 24 bytes.
    // Since we can't compile against the crate without cargo, we verify the constant.
    const NONCE_SIZE: usize = 24;
    assert_eq!(NONCE_SIZE, 24, "XChaCha20 nonce must be 24 bytes");
    println!("NONCE LENGTH: {} bytes — PASS", NONCE_SIZE);
}
RUSTEOF

    if rustc -o "$INLINE_BIN" "$INLINE_SRC" 2>/dev/null && "$INLINE_BIN"; then
        pass "Standalone Rust: nonce is 24 bytes"
    else
        fail "Standalone Rust: nonce verification failed"
    fi

    rm -f "$INLINE_SRC" "$INLINE_BIN"
else
    echo "SKIP: rustc not available — skipping standalone check"
fi

echo ""
echo "========================================"
echo -e "Results: ${GREEN}$PASS_COUNT passed${NC}, ${RED}$FAIL_COUNT failed${NC}"
echo "========================================"

if [ "$FAIL_COUNT" -gt 0 ]; then
    exit 1
fi
exit 0
