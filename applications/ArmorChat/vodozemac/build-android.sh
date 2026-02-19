#!/bin/bash
# Build script for vodozemac Android native library
#
# Usage: ./build-android.sh [release|debug]
#
# Prerequisites:
# - Rust with android targets: rustup target add aarch64-linux-android armv7-linux-androideabi
# - Android NDK (set ANDROID_NDK_HOME)
# - cargo-ndk: cargo install cargo-ndk

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BUILD_TYPE="${1:-release}"
NDK_VERSION="26.0.10792818"

# Check for Android NDK
if [ -z "$ANDROID_NDK_HOME" ]; then
    if [ -d "$HOME/Android/Sdk/ndk/$NDK_VERSION" ]; then
        export ANDROID_NDK_HOME="$HOME/Android/Sdk/ndk/$NDK_VERSION"
    elif [ -d "$HOME/Library/Android/sdk/ndk/$NDK_VERSION" ]; then
        export ANDROID_NDK_HOME="$HOME/Library/Android/sdk/ndk/$NDK_VERSION"
    else
        echo "ERROR: ANDROID_NDK_HOME not set and NDK not found in default locations"
        echo "Set ANDROID_NDK_HOME or install NDK $NDK_VERSION"
        exit 1
    fi
fi

echo "Using NDK: $ANDROID_NDK_HOME"

# Check for cargo-ndk
if ! command -v cargo-ndk &> /dev/null; then
    echo "Installing cargo-ndk..."
    cargo install cargo-ndk
fi

# Add Rust targets if not present
echo "Ensuring Rust targets are installed..."
rustup target add aarch64-linux-android 2>/dev/null || true
rustup target add armv7-linux-androideabi 2>/dev/null || true
rustup target add i686-linux-android 2>/dev/null || true
rustup target add x86_64-linux-android 2>/dev/null || true

# Create output directory
OUTPUT_DIR="$SCRIPT_DIR/../app/src/main/jniLibs"
mkdir -p "$OUTPUT_DIR/arm64-v8a"
mkdir -p "$OUTPUT_DIR/armeabi-v7a"
mkdir -p "$OUTPUT_DIR/x86"
mkdir -p "$OUTPUT_DIR/x86_64"

# Build flags
if [ "$BUILD_TYPE" = "release" ]; then
    CARGO_FLAGS="--release"
else
    CARGO_FLAGS=""
fi

echo ""
echo "=========================================="
echo "Building vodozemac for Android ($BUILD_TYPE)"
echo "=========================================="
echo ""

# Build for each architecture
build_arch() {
    local TARGET=$1
    local ABI=$2
    local MIN_SDK=$3

    echo "Building for $ABI ($TARGET)..."

    cargo ndk -t $ABI -p $MIN_SDK -- build $CARGO_FLAGS

    # Copy to jniLibs
    local LIB_NAME="libvodozemac_android"
    if [ "$BUILD_TYPE" = "release" ]; then
        cp "target/$TARGET/release/$LIB_NAME.so" "$OUTPUT_DIR/$ABI/"
    else
        cp "target/$TARGET/debug/$LIB_NAME.so" "$OUTPUT_DIR/$ABI/"
    fi

    echo "  ✓ $ABI complete"
}

# Build arm64 (most common for modern devices)
build_arch "aarch64-linux-android" "arm64-v8a" "24"

# Build armv7 (for older 32-bit devices)
build_arch "armv7-linux-androideabi" "armeabi-v7a" "24"

# Build x86_64 (for emulators)
build_arch "x86_64-linux-android" "x86_64" "24"

# Build x86 (optional, for older emulators)
# build_arch "i686-linux-android" "x86" "24"

echo ""
echo "=========================================="
echo "Build complete!"
echo "=========================================="
echo ""
echo "Libraries written to:"
ls -la "$OUTPUT_DIR"/*/
echo ""
echo "Library sizes:"
du -h "$OUTPUT_DIR"/*/*.so
