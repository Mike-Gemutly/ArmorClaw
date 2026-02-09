# ArmorClaw Bridge Dockerfile
# This Dockerfile builds the bridge with CGO enabled for SQLCipher support

FROM golang:1.21-alpine AS builder

# Install build dependencies for CGO and SQLCipher
RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite-dev \
    pkgconfig \
    git

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
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    docker-cli \
    libgcc \
    sqlite-libs

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
