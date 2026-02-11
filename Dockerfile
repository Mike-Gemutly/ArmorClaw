# ArmorClaw v1 Container Image
# Multi-stage build for hardened OpenClaw agent runtime

# ============================================================================
# Stage 1: Builder
# ============================================================================
# Has full toolchain for compiling OpenClaw runtime and dependencies
FROM debian:bookworm-slim AS builder

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    build-essential \
    python3 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js (for OpenClaw Node-based components)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# Create Python venv for OpenClaw
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Install Python dependencies (minimal for OpenClaw)
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir \
    aiohttp \
    websockets \
    python-dotenv

# ============================================================================
# Stage 2: Runtime
# ============================================================================
# Minimal attack surface with only what's needed to run OpenClaw
FROM debian:bookworm-slim AS runtime

# Install runtime dependencies only — explicit, minimal
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Python runtime
    python3 \
    python3-venv \
    # Core utilities (allowlisted only)
    coreutils \
    ca-certificates \
    # Security hook compilation
    gcc \
    libc-dev \
    && rm -rf /var/lib/apt/lists/*

# Create dedicated non-root user (UID 10001, not 65534)
# Must do this BEFORE removing /bin/sh
RUN groupadd -g 10001 claw && \
    useradd -u 10001 -g claw -m -s /bin/false claw

# Create OpenClaw directory structure
RUN mkdir -p /opt/openclaw && \
    chown -R claw:claw /opt/openclaw

# Copy OpenClaw runtime files
COPY container/opt/openclaw/* /opt/openclaw/
COPY container/openclaw/ /opt/openclaw/

# Create lib directory and compile security hook library
RUN mkdir -p /opt/openclaw/lib && \
    gcc -shared -fPIC -o /opt/openclaw/lib/libarmorclaw_hook.so \
    /opt/openclaw/security_hook.c -ldl && \
    rm -f /opt/openclaw/security_hook.c

# Make entrypoint and health check executable
RUN chmod +x /opt/openclaw/entrypoint.py && \
    chmod +x /opt/openclaw/health.sh

# Ensure all runtime files are owned by claw user
RUN chown -R claw:claw /opt/openclaw

# Copy Python venv from builder
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy Node.js from builder (NodeSource installs to /usr/bin on Debian)
COPY --from=builder /usr/lib/node_modules /usr/lib/node_modules
COPY --from=builder /usr/bin/node /usr/bin/node
COPY --from=builder /usr/bin/npm /usr/bin/npm
COPY --from=builder /usr/bin/npx /usr/bin/npx

# ============================================================================
# SECURITY HARDENING: Multi-layered defense against runtime exploits
# ============================================================================

# Layer 1: Remove dangerous tools (MUST do while /bin/sh still works!)
# Split into separate RUN commands to avoid deleting the shell mid-execution
RUN rm -f /bin/bash /bin/dash /bin/sh /bin/mv /bin/find
RUN rm -f /bin/ps /usr/bin/top /usr/bin/lsof /usr/bin/strace
RUN rm -f /usr/bin/curl /usr/bin/wget /usr/bin/nc /usr/bin/telnet /usr/bin/ftp
RUN rm -f /usr/bin/sudo /usr/bin/su /usr/bin/passwd
RUN rm -f /usr/bin/gdb /usr/bin/ltrace
RUN apt-get autoremove -y 2>/dev/null || true

# Layer 2: Remove execute permissions from ALL remaining binaries
# This prevents Python/Node from exec'ing anything (even if tools were present)
# Using /usr/bin/find explicitly since /bin/find may have been removed
RUN /usr/bin/find /bin -type f -exec chmod a-x {} \; 2>/dev/null || true
RUN /usr/bin/find /usr/bin -type f -exec chmod a-x {} \; 2>/dev/null || true
RUN chmod +x /usr/bin/python3* /usr/bin/node /usr/bin/npm /usr/bin/npx /usr/bin/env 2>/dev/null || true

# Layer 3: LD_PRELOAD hook to intercept dangerous library calls at library level
# (Already compiled above in /opt/openclaw/lib/libarmorclaw_hook.so)

# Layer 4: Seccomp + Network isolation applied at runtime by bridge
# Bridge applies: --security-opt seccomp=profile.json --network=none

# Switch to non-root user
USER claw

# Set working directory
WORKDIR /opt/openclaw

# Add tmpfs mount points for runtime
VOLUME ["/tmp"]
VOLUME ["/run/secrets"]

# Environment variables
ENV PYTHONUNBUFFERED=1
ENV PATH="/opt/venv/bin:$PATH"
ENV PYTHONPATH="/opt"
ENV ARMORCLAW_SECRETS_PATH="/run/secrets"
ENV ARMORCLAW_SECRETS_FD="3"
# Security hook: Blocks execve(), socket(), connect() at library level
ENV LD_PRELOAD="/opt/openclaw/lib/libarmorclaw_hook.so"

# Health check - verifies agent module is importable (uses Python since /bin/sh is removed)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["python3", "-c", "from openclaw import agent; print('OK')"]

# Entrypoint (secrets verification + agent startup)
ENTRYPOINT ["/opt/openclaw/entrypoint.py"]

# Default command: start ArmorClaw agent
CMD ["python", "-c", "from openclaw import main; main()"]
