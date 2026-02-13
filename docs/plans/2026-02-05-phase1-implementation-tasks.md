# Phase 1 Implementation Tasks: Standard Bridge

**Date:** 2026-02-05
**Status:** Ready for Implementation
**Timeline:** Week 1-2 (10 business days)
**Target:** < 50 MB RAM, Matrix adapter only

---

## Task Breakdown

### Week 1: Core Bridge Infrastructure

#### Task 1.1: Project Setup & Build System
**Effort:** 2 hours
**Priority:** P0

- [ ] Initialize Go module: `go mod init github.com/armorclaw/bridge`
- [ ] Set up directory structure per spec
- [ ] Create Makefile with build, test, install targets
- [ ] Set up CI/CD (GitHub Actions) for Go testing
- [ ] Add pre-commit hooks (gofmt, go vet)

**Deliverable:** Ready-to-build Go project skeleton

---

#### Task 1.2: Configuration System
**Effort:** 3 hours
**Priority:** P0

- [ ] Implement `internal/config/config.go`
- [ ] TOML parsing for bridge.toml
- [ ] Environment variable overrides
- [ ] Validation logic with helpful errors
- [ ] Default values for all settings
- [ ] Config file migration (future-proofing)

**Deliverable:** Working config loader with tests

---

#### Task 1.3: Encrypted Keystore (SQLCipher)
**Effort:** 6 hours
**Priority:** P0 (Core Security Feature)

- [ ] Implement `internal/keystore/keystore.go`
- [ ] SQLCipher integration
- [ ] Master key derivation from system properties
- [ ] CRUD operations for credentials
- [ ] Key rotation support (prepare for later)
- [ ] Unit tests for encryption/decryption

**Deliverable:** Encrypted credential storage working

---

#### Task 1.4: JSON-RPC Server
**Effort:** 4 hours
**Priority:** P0

- [ ] Implement `internal/bridge/server.go`
- [ ] Unix socket server (`/run/armorclaw/bridge.sock`)
- [ ] JSON-RPC 2.0 request/response handling
- [ ] Request routing to method handlers
- [ ] Error handling with proper error codes
- [ ] Connection multiplexing support

**Deliverable:** Functional JSON-RPC server

---

#### Task 1.5: Matrix Client Implementation
**Effort:** 5 hours
**Priority:** P0

- [ ] Implement `internal/adapter/matrix.go`
- [ ] HTTP client for Matrix API
- [ ] Login flow (username/password → access token)
- [ ] Token storage in keystore
- [ ] Message sending (`/_matrix/client/v3/rooms/{roomId}/send`)
- [ ] Sync loop (`/_matrix/client/v3/sync`)
- [ ] Event queue for incoming messages

**Deliverable:** Matrix client with send/receive working

---

### Week 2: Integration & Polish

#### Task 2.1: Core Bridge Methods
**Effort:** 3 hours
**Priority:** P0

- [ ] `bridge.send(room, message)` → Matrix
- [ ] `bridge.receive(timeout)` → Event queue
- [ ] `bridge.status()` → Connection info
- [ ] `bridge.ping()` → Health check
- [ ] Request validation and error responses

**Deliverable:** Complete JSON-RPC API for agents

---

#### Task 2.2: Main Entry Point & Daemonization
**Effort:** 2 hours
**Priority:** P0

- [ ] Implement `cmd/armorclaw-bridge/main.go`
- [ ] Signal handling (SIGTERM, SIGINT)
- [ ] Graceful shutdown
- [ ] PID file management
- [ ] Logging initialization
- [ ] Background daemon mode

**Deliverable:** Runnable bridge binary

---

#### Task 2.3: Agent-Side Python Skill
**Effort:** 3 hours
**Priority:** P0

- [ ] Implement `agent-skill/matrix_bridge.py`
- [ ] Unix socket client for JSON-RPC
- [ ] Simple API: `send()`, `receive()`, `status()`
- [ ] Error handling and reconnection
- [ ] Example agent usage
- [ ] Unit tests

**Deliverable:** Working Python skill for agents

---

#### Task 2.4: Installation Scripts
**Effort:** 4 hours
**Priority:** P1

- [ ] `scripts/install.sh` - Full system install
- [ ] `scripts/setup-keystore.sh` - Initial keystore creation
- [ ] `scripts/start.sh` - Start bridge manually
- [ ] `scripts/stop.sh` - Stop bridge gracefully
- [ ] systemd service file
- [ ] Config file template

**Deliverable:** One-command installation

---

#### Task 2.5: Docker Compose Integration
**Effort:** 2 hours
**Priority:** P1

- [ ] Create `docker-compose.yml` for Conduit + Bridge
- [ ] Bridge runs on host (not in container)
- [ ] Volume mounts for persistence
- [ ] Network configuration
- [ ] Health checks
- [ ] Environment variable configuration

**Deliverable:** Working docker-compose stack

---

#### Task 2.6: Testing Suite
**Effort:** 4 hours
**Priority:** P1

- [ ] Unit tests for keystore (90% coverage)
- [ ] Unit tests for Matrix client
- [ ] Integration tests for JSON-RPC API
- [ ] End-to-end test with mock Matrix server
- [ ] Memory leak testing
- [ ] Performance benchmarks

**Deliverable:** Passing test suite with CI

---

#### Task 2.7: Documentation
**Effort:** 3 hours
**Priority:** P1

- [ ] README.md with quick start
- [ ] API documentation (JSON-RPC methods)
- [ ] Configuration reference
- [ ] Troubleshooting guide
- [ ] Architecture diagram
- [ ] Security considerations

**Deliverable:** Complete user-facing docs

---

#### Task 2.8: Final Integration & Validation
**Effort:** 4 hours
**Priority:** P0

- [ ] Full stack integration test
- [ ] Memory usage validation (< 50 MB target)
- [ ] Concurrency testing (multiple agents)
- [ ] Matrix Conduit integration test
- [ ] Agent container communication test
- [ ] Bug fixing and polish

**Deliverable:** Production-ready Phase 1 bridge

---

## Daily Task Schedule (10 Business Days)

```
Day 1 (Mon):    Task 1.1 (Project Setup) + Task 1.2 start
Day 2 (Tue):    Task 1.2 finish + Task 1.3 start (Keystore)
Day 3 (Wed):    Task 1.3 finish (Keystore complete)
Day 4 (Thu):    Task 1.4 (JSON-RPC Server)
Day 5 (Fri):    Task 1.5 start (Matrix Client)

Day 6 (Mon):    Task 1.5 finish + Task 2.1 (Core Methods)
Day 7 (Tue):    Task 2.2 (Main + Daemon) + Task 2.3 start (Python)
Day 8 (Wed):    Task 2.3 finish + Task 2.4 (Install Scripts)
Day 9 (Thu):    Task 2.5 (Docker Compose) + Task 2.6 start (Tests)
Day 10 (Fri):   Task 2.6 finish + Task 2.7 (Docs) + Task 2.8 (Validation)
```

---

## Success Criteria for Phase 1

| Criteria | Target | How to Measure |
|----------|--------|----------------|
| **Memory usage** | < 50 MB | `ps aux \| grep bridge` |
| **Message latency** | < 100ms p95 | Timing tests |
| **Uptime** | 24 hours | Stability test |
| **Agent connectivity** | 5 concurrent | Load test |
| **Matrix integration** | Full send/receive | E2E test |
| **Test coverage** | > 80% | `go test -cover` |
| **Documentation** | Complete | README + API docs |

---

## Phase 1 Deliverables

```
bridge/
├── cmd/armorclaw-bridge/main.go          ✅ Entry point
├── internal/
│   ├── bridge/server.go                    ✅ JSON-RPC server
│   ├── keystore/keystore.go                ✅ Encrypted storage
│   ├── adapter/matrix.go                   ✅ Matrix client
│   └── config/config.go                    ✅ Config loader
├── agent-skill/matrix_bridge.py            ✅ Python skill
├── scripts/
│   ├── install.sh                          ✅ Installer
│   └── setup-keystore.sh                   ✅ Keystore setup
├── configs/
│   ├── bridge.toml                         ✅ Config template
│   └── armorclaw-bridge.service           ✅ systemd service
├── docker-compose.yml                      ✅ Stack orchestration
├── Makefile                                ✅ Build system
├── README.md                               ✅ Documentation
└── go.mod/go.sum                           ✅ Go module
```

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| **SQLCipher build issues** | Have fallback to standard SQLite + AES encryption |
| **Matrix API changes** | Pin to Conduit version, document compatibility |
| **Memory budget exceeded** | Profile early, optimize hot paths |
| **Unix socket permissions** | Test with different UIDs, document setup |
| **Go dependency conflicts** | Use go.mod, pin all versions |

---

## Handoff to Phase 2

**When Phase 1 is complete:**
1. Merge to `main` branch
2. Tag as `v1.0.0-standard`
3. Release announcement
4. Begin Phase 2 planning (Middleware Layer)

**Phase 1 completion enables:**
- Users can run ArmorClaw with Matrix integration
- Foundation for premium features
- Revenue from early adopters (optional donations/Patreon)
- Real-world testing of core architecture

---

**Status:** Ready to begin implementation
**Next action:** Initialize Go repository and start Task 1.1
