#!/bin/bash
# Prepare GitHub Release
# Builds all binaries and creates release assets

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

VERSION="${1:-v1.0.0}"
RELEASE_DIR="release/$VERSION"
mkdir -p "$RELEASE_DIR"

echo -e "${BLUE}Preparing Release $VERSION${NC}"
echo ""

# Build bridge binaries
echo "Building bridge binaries..."
bash scripts/build-bridge-binaries.sh

# Copy binaries to release directory
echo ""
echo "Copying release assets..."

# Copy Windows binary
if [ -f "bin/bridge.exe" ]; then
    cp bin/bridge.exe "$RELEASE_DIR/armorclaw-bridge-windows-amd64.exe"
    echo -e "${GREEN}✓${NC} Windows binary"
fi

# Copy Linux binaries
for ARCH in amd64 arm64 armv7; do
    if [ -f "build/bridge/armorclaw-bridge-linux-$ARCH" ]; then
        cp "build/bridge/armorclaw-bridge-linux-$ARCH" "$RELEASE_DIR/"
        echo -e "${GREEN}✓${NC} Linux $ARCH"
    fi
done

# Copy macOS binary (if exists)
if [ -f "build/bridge/armorclaw-bridge-darwin-amd64" ]; then
    cp "build/bridge/armorclaw-bridge-darwin-amd64" "$RELEASE_DIR/"
    echo -e "${GREEN}✓${NC} macOS amd64"
fi
if [ -f "build/bridge/armorclaw-bridge-darwin-arm64" ]; then
    cp "build/bridge/armorclaw-bridge-darwin-arm64" "$RELEASE_DIR/"
    echo -e "${GREEN}✓${NC} macOS arm64"
fi

# Create SHA256SUMS
echo ""
echo "Generating checksums..."
cd "$RELEASE_DIR"
sha256sum * > SHA256SUMS
echo -e "${GREEN}✓${NC} SHA256SUMS created"
cd - ..

# Create release manifest
cat > "$RELEASE_DIR/RELEASE_NOTES.md" <<EOF
# ArmorClaw $VERSION

## Release Assets

### Binaries
- \`armorclaw-bridge-windows-amd64.exe\` - Windows (Intel/AMD)
- \`armorclaw-bridge-linux-amd64\` - Linux (Intel/AMD)
- \`armorclaw-bridge-linux-arm64\` - Linux (ARM64)
- \`armorclaw-bridge-linux-armv7\` - Linux (ARM v7)
- \`armorclaw-bridge-darwin-amd64\` - macOS (Intel)
- \`armorclaw-bridge-darwin-arm64\` - macOS (Apple Silicon)

### Container Images
- \`armorclaw/agent:$VERSION\` - Agent container
- \`armorclaw/bridge:$VERSION\` - Bridge container

## Quick Start

### Option 1: One-Command Install
\`\`\`bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy.sh | bash
\`\`\`

### Option 2: Manual Install
\`\`\`bash
# Download binary for your platform
wget https://github.com/armorclaw/armorclaw/releases/download/$VERSION/armorclaw-bridge-linux-amd64
chmod +x armorclaw-bridge-linux-amd64
sudo mv armorclaw-bridge-linux-amd64 /usr/local/bin/armorclaw-bridge
\`\`\`

### Option 3: Docker Deploy
\`\`\`bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
./deploy/deploy-all.sh
\`\`\`

## Verification

All binaries are signed with SHA256 checksums. Verify with:
\`\`\`bash
sha256sum -c SHA256SUMS
\`\`\`

## Full Documentation

- [Quick Start Guide](https://github.com/armorclaw/armorclaw/blob/main/doc/guides/quick-start.md)
- [Deployment Guide](https://github.com/armorclaw/armorclaw/blob/main/doc/guides/deployment-guide.md)
- [Element X Setup](https://github.com/armorclaw/armorclaw/blob/main/doc/guides/element-x-deployment.md)

## Changelog

### Added
- One-command deployment script
- Cross-platform bridge binaries
- Automated provisioning with Matrix
- Element X integration

### Fixed
- Bridge containerization
- Container image distribution
- Documentation gaps

### Known Issues
- Windows deployment requires WSL2
- ARM builds may require manual Go installation
EOF

echo -e "${GREEN}✓${NC} Release notes created"

echo ""
echo -e "${GREEN}═════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}     Release $VERSION Ready!${NC}"
echo -e "${GREEN}═════════════════════════════════════════════════════════${NC}"
echo ""
echo "Assets in: $RELEASE_DIR"
echo ""
echo "Next steps:"
echo "  1. Test binaries on each platform"
echo "  2. Create GitHub Release"
echo "  3. Push container images to registry"
echo ""
echo "To create GitHub Release:"
echo "  gh release create $VERSION --title \"ArmorClaw $VERSION\" --notes-file $RELEASE_DIR/RELEASE_NOTES.md"
