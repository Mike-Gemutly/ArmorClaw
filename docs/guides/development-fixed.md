# ArmorClaw Developer Guide

> **Last Updated:** 2026-02-06
> **Version:** 1.0.0
> **Audience:** Contributors and Developers

---

## Getting Started

### Prerequisites

#### Required
- **Go** 1.24 or later (for Local Bridge)
- **Docker** or Docker Desktop (for container runtime)
- **CGo-enabled compiler** (for SQLCipher dependency)
- **Make** (for build automation)
- **Git** (for version control)

#### Recommended
- **Python** 3.x (for OpenClaw compatibility testing)
- **VS Code** or **GoLand** (for development)
- **Shell** (bash, zsh) for running scripts

---

## Building

### Build Bridge Binary

```bash
cd bridge
go build -o build/armorclaw-bridge ./cmd/bridge
```

### Build Container Image

```bash
docker build -t armorclaw/agent:v1 .
```

### Using Make

```bash
make build    # Build bridge
make container # Build container
make all      # Build all
make clean    # Clean artifacts
```

---

## Testing

### Run Unit Tests

```bash
cd bridge
go test ./...              # All tests
go test -cover ./...        # With coverage
go test -v ./...             # Verbose output
```

### Run Integration Tests

```bash
# Run all integration tests
make test-all

# Run specific test suites
make test-hardening    # Container hardening validation
make test-secrets      # Secrets injection tests
make test-exploits     # Security exploit simulations
make test-e2e          # End-to-end integration tests

# Quick smoke test (hardening only)
make smoke
```

---

## Development Workflow

### 1. Create Feature Branch

```bash
git checkout main
git pull origin main
git checkout -b feature/your-feature-name
```

### 2. Make Changes

```bash
# Edit files
vim bridge/pkg/rpc/server.go

# Format code
go fmt ./...

# Run tests
go test ./...
```

### 3. Commit Changes

```bash
git add .
git commit -m "feat: description of changes"

# Commit types:
# feat: new feature
# fix: bug fix
# docs: documentation changes
# test: test changes
# refactor: code refactoring
# chore: build/process changes
```

---

## Code Style

### Go Code Style

Follow standard Go conventions:

```go
// Package comments
// Package keystore provides encrypted credential storage.
package keystore

// Exported functions have comments
func New(cfg Config) (*Keystore, error) {
    return &Keystore{cfg: cfg}, nil
}

// Error handling
if err \!= nil {
