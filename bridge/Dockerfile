# ArmorClaw Bridge Dockerfile
# This Dockerfile builds the bridge with CGO enabled for SQLCipher support
# NOTE: Must use Debian (glibc) instead of Alpine (musl) because go-sqlcipher
# requires glibc-specific functions (pread64, pwrite64, off64_t)

FROM golang:1.24-bookworm AS builder

# Install build dependencies for CGO and SQLCipher
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc-dev \
    libsqlcipher-dev \
    pkg-config \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled for SQLCipher support
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o armorclaw-bridge \
    ./cmd/bridge

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    docker-cli \
    libsqlcipher0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/armorclaw-bridge /usr/local/bin/

# Copy example config
COPY config.example.toml /etc/armorclaw/config.toml.example

# Create directories with correct permissions
RUN mkdir -p /etc/armorclaw /run/armorclaw /var/lib/armorclaw && \
    chown -R 10002:10002 /etc/armorclaw /run/armorclaw /var/lib/armorclaw

# Set up non-root user
USER 10002:10002

# Environment variables
ENV ARMORCLAW_CONFIG=/etc/armorclaw/config.toml
ENV ARMORCLAW_KEYSTORE_DB=/var/lib/armorclaw/keystore.db

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD pgrep armorclaw-bridge || exit 1

# Run the bridge
ENTRYPOINT ["/usr/local/bin/armorclaw-bridge"]

# Default: run in foreground (docker handles daemonization)
CMD []
