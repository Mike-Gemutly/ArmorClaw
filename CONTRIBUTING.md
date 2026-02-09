# Contributing to ArmorClaw

Thank you for your interest in contributing to ArmorClaw! This document provides guidelines for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.24 or later
- Docker Desktop or Docker Daemon
- Python 3.x (for agent compatibility)
- CGo-enabled compiler (for SQLCipher)

### Build

```bash
# Build the bridge
cd bridge
go build -o build/armorclaw-bridge ./cmd/bridge

# Build the container image
docker build -t armorclaw/agent:v1 .
```

### Test

```bash
# Run all tests
make test-all

# Run specific test suite
make test-hardening
./tests/test-secrets.sh
```

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use 2 spaces for indentation
- Add comments for non-obvious logic
- Write tests for new features

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Security Considerations

ArmorClaw is a security-focused project. All contributions should:

- Maintain security boundaries
- Not expose secrets to disk or logs
- Follow principle of least privilege
- Include security considerations in design

## Reporting Security Issues

For security vulnerabilities, please email **security@armorclaw.com** rather than opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
