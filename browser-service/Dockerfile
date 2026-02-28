# ArmorClaw Browser Service
# Headless browser automation with Playwright + Stealth

# Build stage
FROM node:20-bookworm-slim AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    make \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy package files
COPY package*.json ./
COPY tsconfig.json ./

# Install dependencies (including playwright for browser download)
RUN npm ci

# Copy source
COPY src ./src

# Build TypeScript
RUN npm run build

# Runtime stage - use Playwright base with browsers
FROM mcr.microsoft.com/playwright:v1.42.0-jammy

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# Install additional runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    fonts-liberation \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libatspi2.0-0 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libgbm1 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libwayland-client0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxkbcommon0 \
    libxrandr2 \
    xdg-utils \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy built files and production dependencies
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./

# Ensure Playwright browsers are available and properly linked
# The base image has browsers at /ms-playwright
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

# Verify Chromium is available
RUN ls -la /ms-playwright/ && \
    ls -la /ms-playwright/chromium-*/

# Create non-root user with proper permissions
RUN groupadd --gid 10001 browser && \
    useradd --uid 10001 --gid browser --shell /bin/bash --create-home browser && \
    chown -R browser:browser /app && \
    # Allow browser user to access Playwright browsers
    chmod -R 755 /ms-playwright

# Security hardening
RUN chmod -R 755 /app && \
    chmod 700 /home/browser

# Create cache directory for browser user
RUN mkdir -p /home/browser/.cache && \
    chown -R browser:browser /home/browser/.cache

# Environment
ENV NODE_ENV=production
ENV PORT=3000
ENV LOG_LEVEL=info
ENV HOME=/home/browser

# Switch to non-root user
USER browser

# Expose port (internal only - not exposed externally in production)
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Run the service
ENTRYPOINT ["node", "dist/index.js"]
