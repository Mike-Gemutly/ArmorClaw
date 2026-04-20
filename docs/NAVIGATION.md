# ArmorClaw Documentation Navigation Guide

> **Purpose:** Enable LLMs to navigate from README to architecture to feature to function/variable details
> **Last Updated:** 2026-04-19
> **Version:** 2.0.0

## Navigation Path for LLMs

The ArmorClaw documentation is structured as a 4-level hierarchy for LLM navigation:

```
Level1: Entry Point
    └─> README.md (Project overview, why ArmorClaw, installation, deployment modes)

Level2: Documentation Hubs
    ├─> docs/index.md (Central navigation, feature directory, deployment options)
    └─> docs/ArmorClaw-overview.md (Docker Hub description, quick start, system requirements)

Level3: System Architecture
    ├─> doc/armorclaw.md (Complete system documentation, 3454 lines)
    ├─> doc/ArmorChat.md (Android client documentation, 5881 lines)
    └─> doc/agent-runtime.md (Agent runtime internals)

Level4: Subsystem Documentation
    ├─> doc/sidecar-pipeline.md (Document processing pipeline, Rust + Python sidecars)
    ├─> doc/communication-infra.md (Push, SSO, WebSocket, EventBus, adapters)
    ├─> doc/secretary-workflow.md (Workflow engine)
    ├─> doc/email-pipeline.md (Email pipeline)
    ├─> doc/email-android-integration.md (Email approval integration with ArmorChat)
    ├─> doc/client-applications.md (Admin Panel, ArmorTerminal, Setup Wizard, OpenClaw UI)
    ├─> doc/license-system.md (License server and state management)
    └─> doc/voice-stack.md (WebRTC voice communication)
```

## Quick Reference Table

| Level | Document | What You'll Find | Link |
|-------|----------|------------------|------|
| **Entry** | README.md | Project overview, deployment modes, architecture | [README.md](../README.md) |
| **Hub** | docs/index.md | Central navigation, feature directory, RPC API | [docs/index.md](index.md) |
| **Hub** | ArmorClaw Overview | Docker Hub description, quick start, profiles | [ArmorClaw-overview.md](ArmorClaw-overview.md) |
| **Architecture** | System Doc | Complete system documentation | [doc/armorclaw.md](../doc/armorclaw.md) |
| **Mobile** | ArmorChat Doc | Android client, ViewModels, screens, E2EE | [doc/ArmorChat.md](../doc/ArmorChat.md) |
| **Agents** | Agent Runtime | Agent runtime internals | [doc/agent-runtime.md](../doc/agent-runtime.md) |
| **Documents** | Sidecar Pipeline | Rust/Python sidecars, Go routing, YARA CDR | [doc/sidecar-pipeline.md](../doc/sidecar-pipeline.md) |
| **Comms** | Communication Infra | Push, SSO, WebSocket, EventBus, adapters | [doc/communication-infra.md](../doc/communication-infra.md) |
| **Workflows** | Secretary Workflow | Workflow engine, step execution, blockers | [doc/secretary-workflow.md](../doc/secretary-workflow.md) |
| **Email** | Email Pipeline | Email processing, PII detection, HITL | [doc/email-pipeline.md](../doc/email-pipeline.md) |
| **Email+Mobile** | Email Android | Email approval flow on ArmorChat | [doc/email-android-integration.md](../doc/email-android-integration.md) |
| **Clients** | Client Apps | Admin Panel, ArmorTerminal, Setup Wizard | [doc/client-applications.md](../doc/client-applications.md) |
| **Licensing** | License System | License server, state machine, grace periods | [doc/license-system.md](../doc/license-system.md) |
| **Voice** | Voice Stack | WebRTC voice, TURN relay | [doc/voice-stack.md](../doc/voice-stack.md) |

## By User Intent

| I want to... | Start at | Then go to |
|-------------|---------|-----------|
| **Understand why ArmorClaw exists** | [README.md](../README.md) | [ArmorClaw-overview.md](ArmorClaw-overview.md) |
| **Deploy ArmorClaw** | [README.md](../README.md) (Deployment Modes) | [docs/index.md](index.md) (Post-Setup Scripts) |
| **Understand the architecture** | [doc/armorclaw.md](../doc/armorclaw.md) | [doc/communication-infra.md](../doc/communication-infra.md) |
| **Integrate with the Bridge API** | [docs/index.md](index.md) (Feature Directory) | [doc/armorclaw.md](../doc/armorclaw.md) |
| **Work on ArmorChat Android** | [doc/ArmorChat.md](../doc/ArmorChat.md) | [doc/client-applications.md](../doc/client-applications.md) |
| **Understand document processing** | [doc/sidecar-pipeline.md](../doc/sidecar-pipeline.md) | [doc/armorclaw.md](../doc/armorclaw.md) |
| **Add an RPC method** | [docs/index.md](index.md) (JSON-RPC Server) | [doc/armorclaw.md](../doc/armorclaw.md) |
| **Configure email approvals** | [doc/email-pipeline.md](../doc/email-pipeline.md) | [doc/email-android-integration.md](../doc/email-android-integration.md) |
| **Set up browser automation** | [doc/armorclaw.md](../doc/armorclaw.md) (Jetski) | README.md (Jetski Browser Sidecar) |

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      ARMORCLAW ARCHITECTURE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   External Platforms          Bridge Components                  │
│   ┌─────────────┐            ┌─────────────────────┐           │
│   │    Slack    │───────────▶│   SDTW Adapters     │           │
│   │   Discord   │            │   (Slack/Discord/   │           │
│   │    Teams    │            │    Teams/WhatsApp)  │           │
│   │  WhatsApp   │            └──────────┬──────────┘           │
│   └─────────────┘                       │                       │
│                                         ▼                       │
│                              ┌─────────────────────┐           │
│                              │   Message Queue     │           │
│                              │   (SQLite + WAL)    │           │
│                              └──────────┬──────────┘           │
│                                         │                       │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                    BRIDGE BINARY                         │   │
│   │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌─────────┐ │   │
│   │  │ Keystore  │ │  Trust    │ │   Audit   │ │  RPC    │ │   │
│   │  │(Encrypted)│ │ Middleware│ │   Log     │ │ Server  │ │   │
│   │  └───────────┘ └───────────┘ └───────────┘ └─────────┘ │   │
│   │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌─────────┐ │   │
│   │  │  Budget   │ │  WebRTC   │ │  Errors   │ │  Health │ │   │
│   │  │  Tracker  │ │  Engine   │ │  System   │ │ Monitor │ │   │
│   │  └───────────┘ └───────────┘ └───────────┘ └─────────┘ │   │
│   │  ┌───────────┐ ┌───────────┐ ┌───────────┐             │   │
│   │  │  Ghost    │ │  License  │ │  Crypto   │             │   │
│   │  │  Manager  │ │   State   │ │   Store   │             │   │
│   │  └───────────┘ └───────────┘ └───────────┘             │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│              ┌───────────────┼───────────────┐                  │
│              ▼               ▼               ▼                  │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│   │    Matrix    │  │   Container  │  │    TURN      │         │
│   │  Homeserver  │  │   Runtime    │  │   Server     │         │
│   │  (Conduit)   │  │   (Docker)   │  │  (Coturn)    │         │
│   └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                  │
│   Sidecars                                                      │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│   │ Rust Sidecar │  │Python Sidecar│  │    Jetski    │         │
│   │ (PDF/DOCX)   │  │ (XLSX/MSG)  │  │ (CDP Proxy)  │         │
│   └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## File Structure Reference

```
ArmorClaw/
├── README.md                           # Level 1: Entry point
├── CLAUDE.md                           # Project guidance for Claude Code
├── AGENTS.md                           # Agent OS guardrails
├── LICENSE                             # MIT License
│
├── doc/                                # System documentation
│   ├── armorclaw.md                    # Main system doc (3454 lines)
│   ├── ArmorChat.md                    # Android client doc (5881 lines)
│   ├── agent-runtime.md                # Agent runtime internals
│   ├── sidecar-pipeline.md             # Document processing pipeline
│   ├── communication-infra.md          # Communication subsystems
│   ├── secretary-workflow.md           # Workflow engine
│   ├── email-pipeline.md               # Email pipeline
│   ├── email-android-integration.md    # Email-Android integration
│   ├── client-applications.md          # Client apps overview
│   ├── license-system.md               # License system
│   ├── voice-stack.md                  # Voice stack
│   ├── armor-runtime.md                # Agent runtime (legacy)
│   ├── migration/                      # Migration guides
│   └── ACTIVE/                         # Active work docs
│
├── docs/                               # Published documentation hub
│   ├── index.md                        # Central navigation
│   ├── NAVIGATION.md                   # This file
│   ├── ArmorClaw-overview.md           # Docker Hub overview
│   ├── guides/                         # How-to guides
│   ├── reference/                      # API references
│   ├── operations/                     # Operations docs
│   ├── plans/                          # Design plans
│   ├── output/                         # Generated docs
│   └── dockerfiles/                    # Docker guides
│
├── bridge/                             # Go Bridge (control plane)
│   ├── cmd/bridge/                     # Entry point
│   ├── pkg/rpc/                        # JSON-RPC server (51+ methods)
│   ├── pkg/keystore/                   # SQLCipher encrypted storage
│   ├── pkg/eventbus/                   # In-process pub/sub
│   ├── pkg/websocket/                  # Real-time event streaming
│   ├── pkg/push/                       # Push notifications
│   ├── pkg/secretary/                  # Workflow engine
│   ├── pkg/sidecar/                    # Sidecar routing
│   ├── internal/adapter/               # Matrix and Slack adapters
│   └── internal/sdtw/                  # Discord, Teams, WhatsApp adapters
│
├── jetski/                             # Jetski CDP proxy (Go)
├── browser-service/                    # Browser automation (TypeScript/Playwright)
├── sidecar-rust/                       # Rust Office Sidecar
├── sidecar-python/                     # Python MarkItDown Sidecar
├── applications/                       # Client applications
│   ├── ArmorChat/                      # Android mobile app
│   ├── admin-panel/                    # React admin dashboard
│   ├── ArmorTerminal/                  # Android provisioning app
│   └── setup-wizard/                   # React setup wizard
│
├── container/                          # Container runtime files
├── deploy/                             # Deployment scripts
└── tests/                              # Test suites
```

## Tips for LLM Navigation

1. **Always start at README.md** to understand the project context and current version
2. **Use docs/index.md** for feature directory and deployment options
3. **Read doc/armorclaw.md** for complete system architecture and all subsystem details
4. **Read doc/ArmorChat.md** for mobile client specifics (ViewModels, screens, flows)
5. **Check subsystem docs in doc/** for deep dives into specific areas
6. **Use NAVIGATION.md (this file)** as a quick reference for file locations

---

**Last Updated:** 2026-04-19
**Maintained By:** ArmorClaw Documentation Team
**Feedback:** Open an issue on [GitHub](https://github.com/Gemutly/ArmorClaw/issues)
