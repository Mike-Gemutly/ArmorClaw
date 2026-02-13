# ArmorClaw Documentation Navigation Guide

> **Purpose:** Enable LLMs to navigate from README → Architecture → Feature → Function/Variables
> **Last Updated:** 2026-02-06
> **Version:** 1.0.0

## Navigation Path for LLMs

The ArmorClaw documentation is structured as a 5-level hierarchy for LLM navigation:

```
Level1: Entry Point
    └─> README.md (Project overview, why ArmorClaw, installation)

Level2: Documentation Hub
    └─> docs/index.md (Central navigation to all documentation)

Level3: Architecture
    └─> docs/plans/2026-02-05-armorclaw-v1-design.md (Complete system architecture)

Level4: Feature Specifications
    ├─> docs/reference/bridge.md (API methods, configuration, keystore)
    ├─> docs/reference/container.md (Container hardening, security, entrypoint)
    └─> docs/reference/deployment.md (Deployment scripts, infrastructure)

Level5: Functions & Variables
    └─> docs/output/view.md (All 119 documented functions - DESTINATION)
```

## Quick Reference Table

| Level | Document | What You'll Find | Link |
|-------|----------|------------------|------|
| **Entry** | README.md | Project overview, value proposition, installation | [README.md](../README.md) |
| **Hub** | docs/index.md | Central navigation to all documentation | [docs/index.md](index.md) |
| **Architecture** | V1 Design | System architecture, security model, threat analysis | [V1 Design](plans/2026-02-05-armorclaw-v1-design.md) |
| **Feature: Bridge** | Bridge Reference | JSON-RPC API, configuration, keystore, Matrix | [Bridge Reference](reference/bridge.md) |
| **Feature: Container** | Container Reference | Hardening, entrypoint, security properties | [Container Reference](reference/container.md) |
| **Feature: Deploy** | Deployment Reference | Scripts, infrastructure, verification | [Deployment Reference](reference/deployment.md) |
| **Functions** | Architecture Review | **119 documented functions** with signatures | [Architecture Review](output/review.md) |

## By User Intent

| I want to... | Start at | Then go to |
|-------------|---------|-----------|
| **Understand why ArmorClaw exists** | [README.md](../README.md) | - |
| **Deploy ArmorClaw** | [Quick Start Guide](guides/quick-start.md) | [Deployment Guide](guides/deployment-guide.md) |
| **Understand the architecture** | [V1 Design](plans/2026-02-05-armorclaw-v1-design.md) | [Architecture Review](output/review.md) |
| **Integrate with the Bridge API** | [Bridge Reference](reference/bridge.md) | [Architecture Review](output/review.md) |
| **Understand container security** | [Container Reference](reference/container.md) | [V1 Design](plans/2026-02-05-armorclaw-v1-design.md) |
| **Find a specific function** | [Architecture Review](output/review.md#function-catalog) | - |
| **Modify container behavior** | [Container Reference](reference/container.md) | [Architecture Review](output/review.md#container-functions) |
| **Add RPC method** | [Bridge Reference](reference/bridge.md) | [Architecture Review](output/review.md#rpc-functions) |

## Function Catalog Locations

The complete function catalog (119 functions) is in [Architecture Review](output/review.md#function-catalog), organized by component:

| Component | Functions | Link |
|-----------|-----------|------|
| **Container** | 3 functions | [Container Functions](output/review.md#container-functions) |
| **Config** | 20 functions | [Config Functions](output/review.md#config-functions) |
| **Keystore** | 12 functions | [Keystore Functions](output/review.md#keystore-functions) |
| **RPC (pkg/rpc)** | 13 functions | [RPC Functions](output/review.md#rpc-functions) |
| **Docker Client** | 16 functions | [Docker Client Functions](output/review.md#docker-client-functions) |
| **Socket (pkg/socket)** | 12 functions | [Socket Functions](output/review.md#socket-functions) |
| **Matrix (pkg/matrix)** | 6 functions | [Matrix Functions](output/review.md#matrix-functions) |
| **Matrix Adapter** | 11 functions | [Matrix Adapter Functions](output/review.md#matrix-functions-1) |
| **Bridge Main** | 4 functions | [Bridge Main Functions](output/review.md#bridge-main-functions) |
| **Deployment** | 21 functions | [Deployment Functions](output/review.md#deployment-functions) |

## Key Design Decisions

| Decision | Document | Section |
|----------|----------|---------|
| **Why hardened container?** | V1 Design | [Container Hardening](plans/2026-02-05-armorclaw-v1-design.md#container-hardening) |
| **Why Unix socket?** | V1 Design | [Local Bridge Architecture](plans/2026-02-05-armorclaw-v1-design.md#local-bridge-architecture) |
| **Why FD passing for secrets?** | V1 Design | [Secrets Injection Mechanism](plans/2026-02-05-armorclaw-v1-design.md#secrets-injection-mechanism) |
| **Why SQLCipher + XChaCha20?** | Architecture Review | [Keystore Component](output/review.md#keystore-component) |
| **Why Matrix for communication?** | V1 Design | [Supported AI Providers](plans/2026-02-05-armorclaw-v1-design.md#supported-ai-providers) |

## Security Documentation

| Topic | Document | Section |
|-------|----------|---------|
| **Threat Model** | V1 Design | [Threat Model & Security Guarantees](plans/2026-02-05-armorclaw-v1-design.md#threat-model--security-guarantees) |
| **Security Architecture** | Architecture Review | [Security Architecture](output/review.md#security-architecture) |
| **Container Hardening** | Container Reference | [Hardening Measures](reference/container.md#hardening-measures) |
| **Keystore Security** | Bridge Reference | [Security Considerations](reference/bridge.md#security-considerations) |

## Getting Started by Role

| Role | Start Here | Then |
|------|-----------|------|
| **DevOps Engineer** | [Quick Start Guide](guides/quick-start.md) | [Deployment Guide](guides/deployment-guide.md) |
| **Security Researcher** | [V1 Design](plans/2026-02-05-armorclaw-v1-design.md) | [Security Architecture](output/review.md#security-architecture) |
| **Developer** | [Bridge Reference](reference/bridge.md) | [Architecture Review](output/review.md) |
| **Contributor** | [CONTRIBUTING.md](../CONTRIBUTING.md) | [Documentation Specification](wiki/documentation-specification.md) |

## File Structure Reference

```
ArmorClaw/
├── README.md                           # Level 1: Entry point
├── CLAUDE.md                           # Project guidance for Claude Code
├── LICENSE                             # MIT License
│
├── doc/                                # Documentation root
│   ├── index.md                        # Level 2: Documentation hub
│   ├── NAVIGATION.md                   # This file
│   │
│   ├── wiki/                           # Documentation standards
│   │   ├── index.md                    # Wiki index
│   │   └── documentation-specification.md  # Writing guidelines
│   │
│   ├── plans/                          # Level 3: Architecture & design
│   │   └── 2026-02-05-armorclaw-v1-design.md
│   │
│   ├── reference/                      # Level 4: Feature specifications
│   │   ├── bridge.md                   # Bridge API reference
│   │   ├── container.md                # Container reference
│   │   └── deployment.md               # Deployment reference
│   │
│   ├── guides/                         # How-to guides
│   │   ├── quick-start.md              # Get started in 15 minutes
│   │   ├── element-x-deployment.md     # Element X integration
│   │   └── deployment-guide.md         # Comprehensive deployment
│   │
│   ├── status/                         # Project status
│   │   └── 2026-02-05-status.md
│   │
│   ├── PROGRESS/                       # Milestone tracking
│   │   └── progress.md                 # Updated after each milestone
│   │
│   └── output/                         # Level 5: Architecture reviews
│       └── review.md                   # All 119 documented functions
│
├── bridge/                             # Local Bridge (Go)
├── container/                          # Container runtime files
├── deploy/                             # Deployment scripts
└── tests/                              # Test suites
```

## Tips for LLM Navigation

1. **Always start at README.md** to understand the project context
2. **Use docs/index.md** to find relevant documentation sections
3. **Read V1 Design** for architectural decisions and security model
4. **Check reference docs** for feature-specific specifications
5. **Go to review.md** for function/variable level details
6. **Use NAVIGATION.md (this file)** as a quick reference

## Documentation Standards

All ArmorClaw documentation follows the standards defined in:
- [Documentation Specification](wiki/documentation-specification.md)

This ensures consistent formatting, structure, and quality across all documentation.

---

**Last Updated:** 2026-02-06
**Maintained By:** ArmorClaw Documentation Team
**Feedback:** Open an issue on [GitHub](https://github.com/armorclaw/armorclaw/issues)
