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
- **VS Code** with Go extension
- **Docker Desktop** for local testing
- **Postman** or similar API testing tool

### Repository Structure

```
ArmorClaw/
├── bridge/              # Go Local Bridge
│   ├── cmd/            # Main application entry points
│   ├── pkg/            # Public packages
│   │   ├── config/    # Configuration system
│   │   ├── docker/    # Docker client
│   │   ├── keystore/  # Encrypted keystore
│   │   └── rpc/       # JSON-RPC server
│   └── internal/      # Internal packages
│       └── adapter/    # Matrix adapter
├── container/          # Container runtime files
├── tests/              # Test suites
├── docs/               # Documentation (all docs here)
└── deploy/             # Deployment scripts
```

---

## Development Workflow

### 1. Fork and Clone

```bash
# Fork repository on GitHub
git clone https://github.com/YOUR_USERNAME/armorclaw.git
cd armorclaw
git remote add upstream https://github.com/armorclaw/armorclaw.git
```

### 2. Install Dependencies

```bash
# Install Go dependencies
cd bridge
go mod download

# Verify installation
go mod verify
```

### 3. Build

```bash
# Build bridge binary
cd bridge
go build -o build/armorclaw-bridge ./cmd/bridge

# Run tests
go test ./...
```

### 4. Make Changes

```bash
# Create feature branch
git checkout -b feature/amazing-feature

# Make changes
# ... edit files ...

# Test changes
go test ./...
make test-hardening
```

### 5. Commit Changes

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
if err != nil {
    return nil, fmt.Errorf("failed to initialize: %w", err)
}
```

### Testing

Write tests for all new functionality:

```go
func TestKeystoreNew(t *testing.T) {
    cfg := keystore.DefaultConfig()
    ks, err := keystore.New(cfg)
    if err != nil {
        t.Fatalf("New() failed: %v", err)
    }
    if ks == nil {
        t.Fatal("New() returned nil")
    }
}
```

---

## Running Tests

### Unit Tests

```bash
cd bridge
go test ./...
```

### Integration Tests

```bash
# Container hardening
make test-hardening

# Secrets isolation
./tests/test-secrets.sh

# Exploit mitigation
./tests/test-exploits.sh
```

### All Tests

```bash
make test-all
```

---

## Building Containers

### Build Agent Image

```bash
docker build -t armorclaw/agent:latest .
```

### Test Container Hardening

```bash
./tests/test-hardening.sh
```

---

## Documentation

### View Documentation Locally

```bash
# Serve docs with live reload
cd docs
python3 -m http.server 8000
# Open http://localhost:8000
```

### Update Documentation

- All documentation lives in `docs/`
- See `docs/index.md` for documentation hub
- Follow documentation standards in `docs/wiki/documentation-specification.md` (if available)

---

## Submitting Changes

### 1. Push to Fork

```bash
git push origin feature/amazing-feature
```

### 2. Create Pull Request

1. Visit https://github.com/armorclaw/armorclaw
2. Click "New Pull Request"
3. Provide clear description of changes
4. Link related issues
5. Request review from maintainers

### Pull Request Checklist

- [ ] Tests pass locally
- [ ] New tests added for new features
- [ ] Documentation updated
- [ ] Commits follow conventional commit format
- [ ] Code follows project style guidelines

---

## Development Resources

### Internal
- **Documentation Hub:** `docs/index.md`
- **Configuration Guide:** `docs/guides/configuration.md`
- **Troubleshooting Guide:** `docs/guides/troubleshooting.md`
- **RPC API Reference:** `docs/reference/rpc-api.md`

### External
- **Go Documentation:** https://golang.org/doc/
- **Docker Documentation:** https://docs.docker.com/
- **SQLCipher:** https://www.zetetic.net/sqlcipher/
- **Matrix Spec:** https://spec.matrix.org/

### Community
- **GitHub Issues:** https://github.com/armorclaw/armorclaw/issues
- **Matrix Room:** #armorclaw:matrix.org
- **Email:** dev@armorclaw.com

---

**Developer Guide Last Updated:** 2026-02-06
