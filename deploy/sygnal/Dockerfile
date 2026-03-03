# Sygnal Dockerfile for ArmorClaw
# Matrix Push Gateway

FROM python:3.11-slim

LABEL maintainer="ArmorClaw <security@armorclaw.app>"
LABEL description="Sygnal Matrix Push Gateway for ArmorClaw"

# Install dependencies (git required for pip install from git repos)
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libffi-dev \
    libssl-dev \
    git \
    && rm -rf /var/lib/apt/lists/*

# Create sygnal user
RUN useradd -r -s /bin/false sygnal

# Install sygnal from Matrix.org (not available as matrix-sygnal on PyPI)
RUN pip install --no-cache-dir git+https://github.com/matrix-org/sygnal.git

# Create directories
RUN mkdir -p /data /etc/sygnal && \
    chown -R sygnal:sygnal /data /etc/sygnal

# Configuration is mounted at runtime via docker-compose volumes
# e.g., ./configs/sygnal.yaml:/app/sygnal.yaml:ro
RUN chown sygnal:sygnal /etc/sygnal

# Switch to sygnal user
USER sygnal

# Expose ports
EXPOSE 5000 9100

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:5000/_matrix/push/v1/notify')" || exit 1

# Run sygnal
CMD ["python", "-m", "sygnal.sygnal", "-c", "/etc/sygnal/sygnal.yaml"]
