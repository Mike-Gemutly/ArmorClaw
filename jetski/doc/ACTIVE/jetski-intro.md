# Jetski Codebase Reference

> LLM-readable reference for the entire Jetski project. Read this first.

## What Is Jetski?

Jetski is a **headless browser sidecar for AI agents**. It replaces Chromium/Playwright with a 128MB, 10x-faster stack built on a Zig-based engine (Lightpanda) wrapped in a Go RPC shield (the Observer). AI agents send Chrome DevTools Protocol (CDP) commands; Jetski translates brittle pixel-clicks into resilient DOM operations using 3-tier selector fallbacks.

**Three components, one system:**

| Component | Language | Purpose |
|-----------|----------|---------|
| **Jetski Core** (Observer) | Go 1.24 | CDP proxy with translation, telemetry, PII scanning |
| **Lighthouse** | Go 1.26 | REST API for storing/retrieving Nav-Charts (SQLite) |
| **Chartmaker** | TypeScript 5.3 | CLI tool for recording interactions into Nav-Charts |

---

## Repository Layout

```
jetski/
├── cmd/observer/main.go          # Jetski Core entry point
├── internal/                     # Core internal packages (Go)
│   ├── cdp/                      #   CDP proxy, translator, router, fallback
│   ├── sonar/                    #   Telemetry, circular buffer, wreckage reports
│   ├── security/                 #   PII scanner, session encryption
│   ├── network/                  #   Proxy rotation, circuit breaker
│   └── subprocess/               #   Engine lifecycle, watchdog, auto-restart
├── pkg/                          # Public packages (Go)
│   ├── config/config.go          #   YAML config with env overrides
│   └── logger/logger.go          #   Structured slog wrapper
├── lighthouse/                   # Separate Go module (Lighthouse API)
│   ├── cmd/lighthouse/main.go    #   API server entry point
│   ├── internal/
│   │   ├── api/routes.go         #     4 REST endpoints
│   │   ├── api/middleware.go      #     Auth + logging middleware
│   │   ├── db/queries.go         #     SQLite CRUD
│   │   ├── db/migrations.go      #     Schema migrations
│   │   ├── models/chart.go       #     Chart model
│   │   ├── signing/signer.go     #     HMAC-SHA256 signing
│   │   └── config/config.go      #     Config loading
│   ├── charts/                   #   5 blessed Nav-Chart JSON files
│   ├── data/lighthouse.db        #   SQLite database
│   ├── Dockerfile                #   Multi-stage build (CGO for sqlite3)
│   └── docker-compose.yml        #   Lighthouse service definition
├── jetski-chartmaker/            # TypeScript CLI (separate npm package)
│   ├── src/cli/                  #   Commander.js CLI (init, map, verify, fetch)
│   ├── src/core/browser/         #   Playwright launcher with HUD injection
│   ├── src/core/recorder/        #   Compiles recorded actions to Nav-Charts
│   ├── src/core/validator/       #   JSON Schema validation (AJV)
│   ├── src/injectables/helm/     #   Shadow DOM HUD (HTML+CSS+JS)
│   └── schemas/nav-chart.json    #   Nav-Chart JSON Schema
├── configs/config.yaml           # Main Jetski config
├── Dockerfile                    # Multi-stage build (Go + Lightpanda)
├── docker-compose.yml            # Jetski service definition
├── go.mod                        # Module: jetski-browser (Go 1.24)
└── .golangci.yml                 # Linting: govet, errcheck, staticcheck
```

---

## Jetski Core (The Observer)

**Entry**: `cmd/observer/main.go` — starts HTTP/WS server on `:9222`, spawns Lightpanda on `:9223`

### Startup Sequence
1. Parse flags (`--config`, `--port`, `--log-level`)
2. Load YAML config with env overrides
3. Start Lightpanda subprocess with watchdog
4. Create CDP proxy with method router
5. Optionally enable proxy rotation and PII scanning
6. Listen for WebSocket upgrades, proxy to engine

### Internal Packages

#### `internal/cdp/` — CDP Proxy & Translation
The core value prop. Bidirectional WebSocket proxy that **intercepts** CDP messages and translates them:

- **proxy.go** — WebSocket proxy between agent and Lightpanda engine
- **translator.go** — Converts mouse events to DOM clicks (generates selector JS)
- **router.go** — Routes CDP methods: translate, passthrough, or unsupported
- **fallback.go** — Handles unsupported methods with dummy success responses

**Method routing**:
- `Page.*`, `DOM.*`, `Network.*` → passthrough
- `Input.dispatchMouseEvent` → translate (extract selector, execute DOM click)
- `Runtime.*` → translate
- Unknown methods → fallback (dummy success)

#### `internal/sonar/` — Telemetry & Failure Analysis
Black-box flight recorder for browser failures:

- **buffer.go** — O(1) circular buffer for CDP frame history
- **telemetry.go** — Health score calculation: `H = (primary + 0.5*secondary + 0.1*fallback) / total`
  - 1.0 = Green Water, 0.5 = Choppy Seas, <0.2 = Shipwreck
- **reporter.go** — Generates `wreckage.json` snapshots on failure

#### `internal/security/` — PII Protection
- **pii_scanner.go** — Regex-based detection (SSN, credit cards, emails, passwords)
  - Non-blocking: scans in background, logs "Distress Flares" (ANSI warnings)
- **session.go** — Age encryption for session persistence (disabled in Free-Ride mode)

#### `internal/network/` — Proxy Rotation
- **proxy.go** — Round-robin proxy manager with health checks
- **circuit_breaker.go** — Three-state circuit breaker (Closed → Open → Half-Open)

#### `internal/subprocess/` — Engine Lifecycle
- **manager.go** — Process manager for Lightpanda binary
- **watchdog.go** — 5-second health checks, 3 failures → restart
- **restart.go** — Exponential backoff with jitter
- **health.go** — CDP `/json/version` endpoint polling

### Dependencies
- `gorilla/websocket` — WebSocket proxy
- `filippo.io/age` — Session encryption
- `gopkg.in/yaml.v3` — Config parsing
- **Lightpanda** — External Zig binary, downloaded in Docker build

### Configuration (`configs/config.yaml`)
```yaml
server:
  port: 9222
  host: 0.0.0.0
browser:
  engine_path: /usr/local/bin/lightpanda
  engine_port: 9223
  health_check: true
  check_interval: 5s
  watchdog:
    max_failures: 3
    auto_restart: true
security:
  pii_scanning: true
  encrypt_session: false    # Free-Ride mode
network:
  proxy_enabled: false
logging:
  level: info
```

**Environment overrides**: `JETSKI_PORT`, `JETSKI_ENGINE_PATH`, `JETSKI_PASSPHRASE`, `JETSKI_PROXY_LIST`, `JETSKI_PII_SCANNING`

---

## Lighthouse (Nav-Chart Registry)

**Entry**: `lighthouse/cmd/lighthouse/main.go` — starts REST API on `:8080`

Separate Go module (`github.com/armorclaw/lighthouse`). Think "NPM for AI browsing" — stores versioned, signed navigation charts that agents can fetch and execute.

### API Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/charts?domain=` | No | List charts for domain (max 50) |
| GET | `/charts/{domain}/{version}` | No | Get specific chart version |
| POST | `/charts` | Bearer token | Upload new chart |
| GET | `/charts/blessed?domain=` | No | Get latest ArmorClaw-blessed chart |

### Key Packages
- **api/routes.go** — 4 handlers with proper HTTP status codes
- **api/middleware.go** — Bearer token auth, request logging (`[LIGHTHOUSE]` branding)
- **db/queries.go** — SQLite CRUD with prepared statements
- **db/migrations.go** — Auto-migration on startup (charts + users tables)
- **signing/signer.go** — HMAC-SHA256: `sha256=<hex>`
- **models/chart.go** — Chart struct (domain, version, chartData, signature, blessed)

### Database Schema
```sql
CREATE TABLE charts (
    id INTEGER PRIMARY KEY,
    domain TEXT NOT NULL,
    version TEXT NOT NULL,        -- SemVer
    chart_data TEXT NOT NULL,     -- JSON payload
    signature TEXT,               -- HMAC-SHA256
    blessed BOOLEAN DEFAULT FALSE,
    downloads INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(domain, version)
);
```

### Configuration
- `LIGHTHOUSE_PORT` (default: 8080)
- `LIGHTHOUSE_SECRET_KEY` (default: "change-me-in-production")
- `LIGHTHOUSE_DATABASE_PATH` (default: ./data/lighthouse.db)

### Dependencies
- `go-chi/chi/v5` — HTTP router
- `mattn/go-sqlite3` — CGO SQLite driver

---

## Chartmaker (CLI Tool)

**Entry**: `jetski-chartmaker/src/cli/index.ts` — Commander.js CLI

Records browser interactions and compiles them into `.acsb.json` Nav-Charts.

### Commands

```
jetski-chartmaker init                    # Create project structure
jetski-chartmaker map --url <url>         # Launch browser with recording HUD
jetski-chartmaker verify <file>           # Validate + dry-run a Nav-Chart
jetski-chartmaker fetch <domain> --blessed  # Pull from Lighthouse registry
```

### The Helm (Recording HUD)
Shadow DOM-isolated overlay injected into the browser:
- Draggable panel with mode buttons (Click, Input, Assertion)
- Captures events with 100ms throttle
- Generates 3-tier selectors: `[data-automation-id]` → `[data-testid]` → `#id` → `tag.class`
- Pierces Shadow DOM up to 10 levels
- Handles cross-origin iframe detection

### Source Structure
```
src/
├── cli/commands/
│   ├── init.ts       # Create charts/, sessions/, jetski.config.json
│   ├── map.ts        # Launch Playwright + inject HUD + expose RPC
│   ├── verify.ts     # Schema validation + optional headless dry-run
│   └── fetch.ts      # HTTP GET to Lighthouse, save .acsb.json
├── core/
│   ├── browser/
│   │   ├── launcher.ts    # Playwright launch, HUD injection, RPC bridge
│   │   └── session.ts     # Session state persistence
│   ├── recorder/
│   │   └── state-compiler.ts  # Compiles RecordedAction[] → NavChart
│   └── validator/
│       └── schema.ts       # AJV validation against nav-chart.json
└── injectables/
    ├── core/event-hooks.ts  # IIFE: captures clicks/inputs, calls RPC
    └── helm/
        ├── hud.html         # HUD template
        ├── hud.css          # Navy blue maritime theme
        └── hud.js           # Draggable UI, selector generation, Shadow DOM
```

### Dependencies
- `playwright` — Browser automation
- `commander` — CLI framework
- `ajv` + `ajv-formats` — JSON Schema validation
- `zod` — Runtime type validation
- `jest` + `ts-jest` — Testing

---

## Nav-Chart Format

The `.acsb.json` files are the lingua franca between all three components:

```json
{
  "version": 1,
  "target_domain": "https://github.com",
  "metadata": {
    "generated_by": "@armorclaw/jetski-chartmaker",
    "timestamp": "2026-04-08T12:00:00Z"
  },
  "action_map": {
    "1": {
      "action_type": "navigate",
      "url": "https://github.com/login",
      "post_action_wait": { "type": "waitForSelector", "selector": { "primary_css": "body" }, "timeout": 5000 }
    },
    "2": {
      "action_type": "input",
      "selector": {
        "primary_css": "[data-automation-id='login-field']",
        "secondary_xpath": "//input[@name='login']",
        "fallback_js": "document.querySelector('input[type=text]')"
      },
      "value": "{{USERNAME}}",
      "post_action_wait": { "type": "waitForTimeout", "timeout": 500 }
    },
    "3": {
      "action_type": "click",
      "selector": {
        "primary_css": "[data-automation-id='submit-btn']",
        "secondary_xpath": "//button[@type='submit']",
        "fallback_js": "document.querySelector('button.primary')"
      },
      "post_action_wait": { "type": "waitForVisible", "selector": { "primary_css": ".dashboard" }, "timeout": 5000 }
    }
  }
}
```

**Key design decisions**:
- 3-tier selectors survive layout changes
- Post-action waits prevent race conditions
- Template variables (`{{USERNAME}}`, `{{PASSWORD}}`) for credential injection
- Frame routing metadata for iframe/Shadow DOM navigation
- Validated by `schemas/nav-chart.json` JSON Schema

---

## Data Flow

```
1. RECORD
   Developer → jetski-chartmaker map → Playwright browser + HUD
   → Record clicks/inputs → Compile to .acsb.json Nav-Chart

2. PUBLISH
   Developer → POST /charts → Lighthouse API → SQLite + HMAC signature
   → Chart is now "blessed" (or community)

3. EXECUTE
   AI Agent → fetch Nav-Chart from Lighthouse
   → Send CDP commands to Jetski Core (ws://localhost:9222)
   → Observer translates to DOM operations on Lightpanda engine
   → Sonar tracks selector health, generates wreckage on failure
```

---

## Docker Deployment

### Jetski Core
```yaml
# docker-compose.yml
services:
  jetski:
    image: armorclaw/jetski:latest
    ports: ["9222:9222", "8080:8080"]
    environment:
      - JETSKI_PROXY_LIST=
      - JETSKI_PASSPHRASE=
    volumes: ["./sessions:/root/.jetski/sessions"]
    mem_limit: 150m
    restart: unless-stopped
```

Multi-stage build: (1) Go compile, (2) fetch Lightpanda v0.2.6 with SHA256 verify, (3) Alpine runtime <150MB.

### Lighthouse
```yaml
# lighthouse/docker-compose.yml
services:
  lighthouse:
    build: .
    ports: ["8080:8080"]
    environment:
      - LIGHTHOUSE_SECRET_KEY=${LIGHTHOUSE_SECRET_KEY}
    volumes: ["./data:/app/data"]
    restart: unless-stopped
```

**Port collision**: Both default to 8080. Remap Lighthouse to `8081:8080` when co-locating.

---

## Testing

| Component | Framework | Files | Focus |
|-----------|-----------|-------|-------|
| Jetski Core | Go `testing` | 18 files | CDP proxy, circuit breaker, PII scanner, subprocess |
| Lighthouse | Go `testing` | 5 files | DB queries, signing, config, validation |
| Chartmaker | Jest + ts-jest | 6 files | Fetch command, GitHub DOM, Shadow DOM, Stripe |

---

## Key Architecture Patterns

1. **CDP Proxy** — Bidirectional WebSocket proxy with method routing (translate/passthrough/fallback)
2. **3-Tier Selector Matrix** — CSS → XPath → JS, survives layout drift
3. **Circular Buffer** — O(1) CDP frame storage for failure reconstruction
4. **Circuit Breaker** — Proxy health: Closed → Open (3 failures) → Half-Open (30s timeout)
5. **Watchdog** — 5s health checks, 3 failures → exponential backoff restart
6. **Health Score** — Weighted metric: `H = (primary + 0.5*secondary + 0.1*fallback) / total`
7. **PII Scanner** — Non-blocking regex detection with ANSI Distress Flare warnings
8. **HMAC-SHA256** — Chart signing with `sha256=<hex>` format

---

## Build & Run

```bash
# Jetski Core
go run cmd/observer/main.go --port=9222
go test ./...

# Lighthouse
cd lighthouse && go run cmd/lighthouse/main.go

# Chartmaker
cd jetski-chartmaker && npm install && npm run dev
npm test

# Docker
docker-compose up -d                              # Jetski Core
cd lighthouse && docker-compose up -d              # Lighthouse
```

---

## Security Modes

| | Free-Ride (Default) | Tethered (ArmorClaw) |
|---|---|---|
| Sessions | Unencrypted files | SQLCipher encryption |
| PII | Logged as warnings | Hardware-bound keystore |
| Credentials | Cleartext in transit | Biometric approval |
| Use case | Local development | Production deployment |

---

## Agent Infrastructure (`.opencode/agents/`)

| Agent | Purpose |
|-------|---------|
| `sentinel-ops` | Docker Compose deployment, environment, diagnostics |
| `claw-ssh` | SSH tunnel, RPC bridge access via localhost:4096 |
| `armor-test-pilot` | RPC health checks, privacy audits |
| `chat-adb` | Android app debugging, Matrix sync validation |

---

## Sprint History (`.sisyphus/plans/`)

| Sprint | Focus | Status |
|--------|-------|--------|
| A | Sonar telemetry, merge instructions | Complete |
| B | Sonar UI viewer, execution plan | Complete |
| C | Lighthouse API, 5 blessed charts, CLI fetch | Complete |
| D | Sea Trials (VPS deployment, live-fire testing) | Planned |

---

## Quick Reference: File Locations

| Need | Path |
|------|------|
| Start the observer | `cmd/observer/main.go` |
| Start Lighthouse API | `lighthouse/cmd/lighthouse/main.go` |
| CLI commands | `jetski-chartmaker/src/cli/commands/` |
| CDP translation logic | `internal/cdp/translator.go` |
| Selector fallback | `internal/cdp/fallback.go` |
| Health scoring | `internal/sonar/telemetry.go` |
| PII detection | `internal/security/pii_scanner.go` |
| API routes | `lighthouse/internal/api/routes.go` |
| Database queries | `lighthouse/internal/db/queries.go` |
| Chart signing | `lighthouse/internal/signing/signer.go` |
| Nav-Chart schema | `jetski-chartmaker/schemas/nav-chart.json` |
| HUD injection | `jetski-chartmaker/src/injectables/helm/` |
| Config (Jetski) | `configs/config.yaml` |
| Config (Lighthouse) | `lighthouse/configs/config.yaml` |
| Deployment | `AGENTS.md` |
