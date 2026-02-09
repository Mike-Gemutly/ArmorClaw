#!/bin/bash
# ArmorClaw Container Build and Push Script
# Builds container image and pushes to registry

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
IMAGE_NAME="${IMAGE_NAME:-armorclaw/agent}"
VERSION="${VERSION:-v1.0.0}"
REGISTRY="${REGISTRY:-docker.io}"
PUSH="${PUSH:-false}"
PLATFORM="${PLATFORM:-linux/amd64}"
LOAD="${LOAD:-true}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --push)
            PUSH=true
            shift
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --no-load)
            LOAD=false
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --push           Push image to registry after build"
            echo "  --registry URL   Registry URL (default: docker.io)"
            echo "  --version VER    Version tag (default: v1.0.0)"
            echo "  --platform PLAT  Platform (default: linux/amd64)"
            echo "  --no-load        Don't load image into Docker"
            echo ""
            echo "Environment Variables:"
            echo "  IMAGE_NAME       Base image name (default: armorclaw/agent)"
            echo "  VERSION          Version tag (default: v1.0.0)"
            echo "  REGISTRY         Registry URL (default: docker.io)"
            echo "  PUSH             Set to 'true' to push after build"
            echo ""
            echo "Examples:"
            echo "  $0 --push"
            echo "  $0 --registry ghcr.io --version v1.0.0 --push"
            echo "  PUSH=true $0"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run --help for usage"
            exit 1
            ;;
    esac
done

FULL_IMAGE="$REGISTRY/$IMAGE_NAME:$VERSION"
LATEST_TAG="$REGISTRY/$IMAGE_NAME:latest"

echo -e "${BLUE}═════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}     ArmorClaw Container Build${NC}"
echo -e "${BLUE}═════════════════════════════════════════════════════════${NC}"
echo ""
echo "Configuration:"
echo "  Image:     $FULL_IMAGE"
echo "  Platform:  $PLATFORM"
echo "  Push:      $PUSH"
echo "  Load:      $LOAD"
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v docker &>/dev/null; then
    echo -e "${RED}Error: Docker not installed${NC}"
    exit 1
fi

if ! docker info &>/dev/null; then
    echo -e "${RED}Error: Docker not running${NC}"
    exit 1
fi

# Check for buildx (multi-platform support)
if ! docker buildx version &>/dev/null; then
    echo -e "${YELLOW}Warning: buildx not available, using legacy build${NC}"
    USE_BUILDX=false
else
    USE_BUILDX=true
fi

echo -e "${GREEN}✓${NC} Docker ready"
echo ""

# Build image
echo -e "${BLUE}Building container image...${NC}"

if [[ "$USE_BUILDX" == "true" ]]; then
    # Use buildx for multi-platform support
    BUILDX_NAME="armorclaw-build-$(date +%s)"
    docker buildx create --name "$BUILDX_NAME" --driver docker-container 2>/dev/null || true
    docker buildx use "$BUILDX_NAME"
    
    BUILD_ARGS="--platform $PLATFORM"
    if [[ "$LOAD" == "true" ]]; then
        BUILD_ARGS="$BUILD_ARGS --load"
    fi
    
    docker buildx build \
        $BUILD_ARGS \
        --tag "$FULL_IMAGE" \
        --tag "$LATEST_TAG" \
        --file Dockerfile \
        .
    
    docker buildx rm "$BUILDX_NAME" 2>/dev/null || true
else
    # Legacy build
    docker build \
        --tag "$FULL_IMAGE" \
        --tag "$LATEST_TAG" \
        --file Dockerfile \
        .
fi

echo -e "${GREEN}✓${NC} Image built successfully"
echo ""

# Show image info
echo "Image details:"
docker images "$IMAGE_NAME" --format "  {{.Repository}}:{{.Tag}} - {{.Size}}"
echo ""

# Push to registry
if [[ "$PUSH" == "true" ]]; then
    echo -e "${BLUE}Pushing to registry...${NC}"
    
    echo "Pushing: $FULL_IMAGE"
    docker push "$FULL_IMAGE"
    
    echo "Pushing: $LATEST_TAG"
    docker push "$LATEST_TAG"
    
    echo -e "${GREEN}✓${NC} Images pushed successfully"
    echo ""
    
    # Generate checksums
    echo "Generating SHA256 checksums..."
    docker pull "$FULL_IMAGE" >/dev/null 2>&1 || true
    IMAGE_ID=$(docker images "$FULL_IMAGE" --format "{{.ID}}")
    
    cat > "SHA256SUMS-$VERSION.txt" <<EOF
# ArmorClaw Container Image Checksums
# Version: $VERSION
# Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

# Docker Image
# Image ID: $IMAGE_ID
# Pull command: docker pull $FULL_IMAGE
EOF
    
    echo -e "${GREEN}✓${NC} Checksums saved to SHA256SUMS-$VERSION.txt"
    echo ""
else
    echo -e "${YELLOW}Note: Skipping registry push (use --push to enable)${NC}"
    echo ""
    echo "To push manually:"
    echo "  docker push $FULL_IMAGE"
    echo "  docker push $LATEST_TAG"
    echo ""
fi

# Save to tar (for offline deployment)
if [[ "$LOAD" == "true" ]]; then
    TAR_FILE="armorclaw-agent-$VERSION.tar.gz"
    echo -e "${BLUE}Creating archive for offline deployment...${NC}"
    
    docker save "$FULL_IMAGE" | gzip > "$TAR_FILE"
    
    TARBALL_SIZE=$(du -h "$TAR_FILE" | awk '{print $1}')
    echo -e "${GREEN}✓${NC} Archive created: $TAR_FILE ($TARBALL_SIZE)"
    echo ""
    
    # Generate checksum for tarball
    TARBALL_SHA=$(sha256sum "$TAR_FILE" | awk '{print $1}')
    echo "Tarball checksum:"
    echo "  $TARBALL_SHA  $TAR_FILE"
    echo ""
fi

echo -e "${GREEN}═════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}     Build Complete!${NC}"
echo -e "${GREEN}═════════════════════════════════════════════════════════${NC}"
echo ""
echo "Next steps:"
echo "  1. Test locally: docker run --rm $FULL_IMAGE id"
echo "  2. Deploy: docker-compose -f docker-compose-stack.yml up -d"
echo ""

# Exit with success
exit 0
