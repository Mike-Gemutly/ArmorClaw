#!/bin/bash
# ArmorClaw Docker Hub Push Script
# Automates building, tagging, and pushing images to Docker Hub

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     ArmorClaw Docker Hub Push Script                     ║${NC}"
echo -e "${BLUE}╚═════════════════════════════════════════════════════════╝${NC}"
echo ""

# Configuration
DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-}"
IMAGE_NAME="${IMAGE_NAME:-agent}"
TAG="${TAG:-latest}"
BUILD_PLATFORMS="${BUILD_PLATFORMS:-linux/amd64}"

if [[ -z "$DOCKERHUB_USERNAME" ]]; then
    echo -e "${RED}Error: DOCKERHUB_USERNAME environment variable not set${NC}"
    echo ""
    echo "Usage:"
    echo "  export DOCKERHUB_USERNAME=yourusername"
    echo "  ./push-dockerhub.sh [optional:TAG]"
    echo ""
    echo "Example:"
    echo "  export DOCKERHUB_USERNAME=armorclaw"
    echo "  ./push-dockerhub.sh v1.0.0"
    exit 1
fi

FULL_IMAGE="${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${TAG}"
echo -e "${BLUE}Configuration:${NC}"
echo "  Docker Hub Username: ${DOCKERHUB_USERNAME}"
echo "  Image Name: ${IMAGE_NAME}"
echo "  Tag: ${TAG}"
echo "  Full Image: ${FULL_IMAGE}"
echo ""

# Check if Docker is installed
if ! command -v docker &>/dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

# Login to Docker Hub
echo -e "${YELLOW}Step 1: Logging into Docker Hub...${NC}"
if ! docker info | grep -q "Username"; then
    docker login
fi
echo -e "${GREEN}✅ Logged in${NC}"
echo ""

# Build Docker image
echo -e "${YELLOW}Step 2: Building Docker image...${NC}"
docker build \
  --platform ${BUILD_PLATFORMS} \
  -t "${FULL_IMAGE}" \
  -f Dockerfile \
  .

echo -e "${GREEN}✅ Build complete${NC}"
echo ""

# Display image size
IMAGE_SIZE=$(docker images "${FULL_IMAGE}" --format "{{.Size}}")
echo -e "${BLUE}Image size: ${IMAGE_SIZE}${NC}"
echo ""

# Push to Docker Hub
echo -e "${YELLOW}Step 3: Pushing to Docker Hub...${NC}"
docker push "${FULL_IMAGE}"

echo -e "${GREEN}✅ Push complete${NC}"
echo ""

# Verify push
echo -e "${YELLOW}Step 4: Verifying push...${NC}"
docker pull "${FULL_IMAGE}" >/dev/null 2>&1

if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}✅ Image successfully pushed and verified${NC}"
else
    echo -e "${RED}❌ Verification failed${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Success! Image available at:                     ║${NC}"
echo -e "${GREEN}║     ${FULL_IMAGE}                                    ║${NC}"
echo -e "${BLUE}╚═════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "To deploy on Hostinger VPS:"
echo "  1. SSH into your VPS"
echo "  2. Run: docker pull ${FULL_IMAGE}"
echo "  3. Deploy using docker-compose.yml"
echo ""
