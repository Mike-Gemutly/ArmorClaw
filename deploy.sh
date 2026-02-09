#!/bin/bash
# ArmorClaw One-Command Deployment Script
# Auto-detects platform and installs ArmorClaw

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     ArmorClaw One-Command Deployment                        ║${NC}"
echo -e "${BLUE}╚═════════════════════════════════════════════════════════╝${NC}"
echo ""

# Detect platform
OS_TYPE=$(uname -s)
ARCH=$(uname -m)

case "$OS_TYPE" in
    Linux*)
        case "$ARCH" in
            x86_64)  PLATFORM="linux-amd64" ;;
            aarch64) PLATFORM="linux-arm64" ;;
            armv7l)  PLATFORM="linux-armv7" ;;
            *)       PLATFORM="linux-$ARCH" ;;
        esac
        ;;
    Darwin*)
        case "$ARCH" in
            x86_64)  PLATFORM="darwin-amd64" ;;
            arm64)   PLATFORM="darwin-arm64" ;;
            *)       PLATFORM="darwin-$ARCH" ;;
        esac
        ;;
    MINGW*|MSYS*|CYGWIN*)
        PLATFORM="windows-amd64"
        ;;
    *)
        echo -e "${RED}Unsupported platform: $OS_TYPE $ARCH${NC}"
        exit 1
        ;;
esac

echo "Detected: $OS_TYPE $ARCH -> $PLATFORM"

# Check if Docker is installed
if ! command -v docker &>/dev/null; then
    echo -e "${YELLOW}Docker not found. Installing...${NC}"
    if [[ "$OS_TYPE" == "Linux" ]]; then
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker $USER
        echo -e "${GREEN}Docker installed. Please log out and back in.${NC}"
        exit 0
    else
        echo -e "${RED}Please install Docker Desktop manually${NC}"
        exit 1
    fi
fi

# Download appropriate binary
BINARY_URL="https://github.com/armorclaw/armorclaw/releases/download/v1.0.0/armorclaw-bridge-${PLATFORM}"
BINARY_NAME="armorclaw-bridge"

if [[ "$PLATFORM" == "windows-amd64" ]]; then
    BINARY_NAME="${BINARY_NAME}.exe"
fi

echo "Downloading: $BINARY_URL"
curl -fsSL "$BINARY_URL" -o "$BINARY_NAME" || {
    echo -e "${RED}Failed to download. Building from source instead...${NC}"

    # Fallback: Build from source
    if [[ -d "bridge" ]]; then
        echo "Building bridge binary..."
        cd bridge
        go build -o "../$BINARY_NAME" ./cmd/bridge
        cd ..
    else
        echo -e "${RED}Source not found. Cannot build.${NC}"
        exit 1
    fi
}

chmod +x "$BINARY_NAME" 2>/dev/null || true

# Verify checksum if available
CHECKSUM_URL="${BINARY_URL}.sha256"
if curl -fsSL "$CHECKSUM_URL" -o SHA256SUMS 2>/dev/null; then
    if sha256sum -c SHA256SUMS 2>/dev/null; then
        echo -e "${GREEN}Checksum verified${NC}"
    else
        echo -e "${YELLOW}Checksum verification failed${NC}"
    fi
fi

# Install bridge
if [[ "$OS_TYPE" == "Linux" ]]; then
    echo "Installing bridge to /usr/local/bin..."
    sudo cp "$BINARY_NAME" /usr/local/bin/armorclaw-bridge
    sudo chmod +x /usr/local/bin/armorclaw-bridge
    echo -e "${GREEN}Bridge installed${NC}"
elif [[ "$OS_TYPE" == "Darwin" ]]; then
    echo "Installing bridge to /usr/local/bin..."
    sudo cp "$BINARY_NAME" /usr/local/bin/armorclaw-bridge
    sudo chmod +x /usr/local/bin/armorclaw-bridge
    echo -e "${GREEN}Bridge installed${NC}"
else
    echo "Bridge binary: $(pwd)/$BINARY_NAME"
fi

# Run deployment
if [[ -f "deploy/deploy-all.sh" ]]; then
    echo "Running full deployment..."
    exec bash deploy/deploy-all.sh "$@"
else
    echo -e "${YELLOW}Full deployment script not found${NC}"
    echo "Bridge binary installed. You can now run:"
    echo "  armorclaw-bridge --help"
fi
