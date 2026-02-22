# ArmorClaw Dockerfiles Documentation

> **Last Updated:** 2026-02-22
> **Purpose:** Document Docker image patterns, gotchas, and solutions

---

## Dockerfiles Overview

| File | Purpose | Base Image |
|------|---------|------------|
| `Dockerfile` | Production hardened agent container | `debian:bookworm-slim` |
| `Dockerfile.quickstart` | Self-contained deployment with bridge | Multi-stage (Go + Debian) |
| `Dockerfile.openclaw-standalone` | OpenClaw integration container | Multi-stage (Node.js) |

---

## Common Patterns

### 1. Entrypoint Flag Handling

**Problem:** Entrypoints that require Docker socket or other runtime dependencies fail when run with `--help` or `--version` flags in CI/CD tests.

**Solution:** Handle informational flags BEFORE checking for runtime dependencies.

```bash
#!/bin/bash
# CORRECT: Check flags first

# Handle --help and --version BEFORE any dependency checks
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage information..."
    exit 0
fi

if [ "$1" = "--version" ] || [ "$1" = "-v" ]; then
    echo "Version: 1.0.0"
    exit 0
fi

# NOW check for Docker socket (only needed for actual runtime)
if [ ! -S /var/run/docker.sock ]; then
    echo "ERROR: Docker socket required"
    exit 1
fi
```

**Wrong Pattern:**
```bash
#!/bin/bash
# WRONG: Check dependencies first

# This will fail CI/CD tests that run with --help
if [ ! -S /var/run/docker.sock ]; then
    echo "ERROR: Docker socket required"
    exit 1
fi

# Too late - already failed
if [ "$1" = "--help" ]; then
    echo "Usage..."
    exit 0
fi
```

### 2. CI/CD Test Image Grep Pattern

**Problem:** Test job greps for wrong image name.

**Wrong:**
```yaml
- name: Pull and test image
  run: |
    docker images | grep ${{ github.repository_owner }}/agent
```

**Correct:**
```yaml
- name: Pull and test image
  run: |
    docker images | grep ${{ env.DOCKER_IMAGE }}
```

Or use the actual image name:
```yaml
docker images | grep mikegemut/armorclaw
```

### 3. .dockerignore Exclusions

**Problem:** `.dockerignore` excludes files needed by `Dockerfile.quickstart`.

**Solution:** Comment out or remove exclusions for files needed by other Dockerfiles.

```gitignore
# Commented out to allow Dockerfile.quickstart access
# bridge/
# go.mod
# go.sum
# deploy/
# docker-compose*.yml
# .env.*
```

### 4. Docker Compose Installation

**Problem:** `docker-compose-plugin` package not available in Debian repos.

**Solution:** Download standalone binary from GitHub releases.

```dockerfile
# Wrong: apt-get install docker-compose-plugin
# Right: Download from GitHub releases
RUN curl -fsSL "https://github.com/docker/compose/releases/download/v2.24.0/docker-compose-$(uname -s)-$(uname -m)" \
    -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose
```

---

## GitHub Actions Permissions

**Problem:** CodeQL SARIF upload fails with "Resource not accessible by integration".

**Solution:** Add `security-events: write` permission.

```yaml
permissions:
  contents: read
  packages: write
  id-token: write
  security-events: write  # Required for SARIF upload
```

**Also:** Upgrade CodeQL Action from v3 to v4 (v3 deprecated December 2026).

```yaml
# Wrong
uses: github/codeql-action/upload-sarif@v3

# Right
uses: github/codeql-action/upload-sarif@v4
```

---

## Line Endings (Windows Development)

**Problem:** Shell scripts have CRLF line endings, causing failures on Linux.

**Solution:** Use `.gitattributes` to enforce LF line endings.

```gitattributes
# .gitattributes
*.sh text eol=lf
*.go text eol=lf
*.py text eol=lf
*.toml text eol=lf
Dockerfile* text eol=lf
docker-compose*.yml text eol=lf
```

---

## Quick Reference Checklist

Before committing Docker changes:

- [ ] Entrypoint handles `--help` before checking dependencies
- [ ] Entrypoint handles `--version` before checking dependencies
- [ ] CI/CD test job greps for correct image name
- [ ] `.dockerignore` doesn't exclude needed files
- [ ] Docker Compose installed from GitHub releases (not apt)
- [ ] `security-events: write` permission for SARIF upload
- [ ] CodeQL Action uses v4 (not v3)
- [ ] `.gitattributes` enforces LF for shell scripts

---

## Error Messages and Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `Docker socket not found` in CI/CD test | Entrypoint checks socket before flags | Handle `--help`/`--version` first |
| `.env.template not found` | `.dockerignore` excludes file | Comment out exclusion in `.dockerignore` |
| `docker-compose-plugin not found` | Not in Debian repos | Download binary from GitHub releases |
| `Resource not accessible by integration` | Missing `security-events: write` | Add permission to workflow |
| `CRLF will be replaced by LF` | Windows line endings | Add `.gitattributes` with `eol=lf` |
| `grep` finds no match in test | Wrong image name in grep | Use `env.DOCKER_IMAGE` or correct name |

---

## Test Commands

```bash
# Test --help locally (should not require Docker socket)
docker build -f Dockerfile.quickstart -t armorclaw-test .
docker run --rm armorclaw-test --help

# Test --version locally
docker run --rm armorclaw-test --version

# Test full startup (requires Docker socket)
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock armorclaw-test
```
