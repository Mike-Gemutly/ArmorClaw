# ArmorClaw Architecture Review
> **Purpose:** Complete guide to ArmorClaw deployment, architecture, and components
> **Version:** 4.11.0
> **Last Updated:** 2026-03-15
> **Status:** Active Reference


## Phase 12: Skills System & Approvals (2026-03-15)

### Overview

Completed the remaining 20% of the Skills System and implemented missing Planned Features:
- **WebDAV Skill** - File operations via WebDAV protocol
- **Calendar Skill** - CalDAV-based calendar integration
- **Rolodex Database** - Contact management with encrypted storage
- **Three-Way Consent** - Matrix room-based approval workflow

### Implementation Details

| Feature | Description | Files Changed |
|---------|-------------|---------------|
| **WebDAV Skill** | Client for list/get/put/delete operations | `bridge/internal/skills/webdav.go` |
| **Calendar Skill** | CalDAV client for calendar events | `bridge/internal/skills/calendar.go` |
| **Rolodex Schema** | Contact tables with encryption | `bridge/pkg/secretary/schema.sql` |
| **Rolodex Service** | CRUD for contacts with Matrix commands | `bridge/pkg/secretary/rolodex.go` |
| **Three-Way Consent** | Matrix room approval workflow | `bridge/pkg/pii/three_way_consent.go` |

### WebDAV Skill Implementation

**File:** `bridge/internal/skills/webdav.go`

```go
// WebDAV operations using Go net/http
type WebDAVParams struct {
    URL         string `json:"url"`
    Operation   string `json:"operation"` // list, get, put, delete
    Username    string `json:"username,omitempty"`
    Password    string `json:"password,omitempty"`
    Content     []byte `json:"content,omitempty"`
    ContentType string `json:"content_type,omitempty"`
}
```

**Operations:**
- `list` - PROPFIND to list directory contents
- `get` - GET to download file contents
- `put` - PUT to upload content
- `delete` - DELETE to remove files

**Security:** SSRF validation on all URLs (blocks private networks)

### Calendar Skill (CalDAV) Implementation

**File:** `bridge/internal/skills/calendar.go`

```go
// CalDAV operations using github.com/emersion/go-webdav/caldav
type CalendarParams struct {
    ServerURL   string `json:"server_url"`
    Username    string `json:"username"`
    Password    string `json:"password"`
    Operation   string `json:"operation"` // list_calendars, create_event, get_events, delete_event
    CalendarID  string `json:"calendar_id,omitempty"`
    Event       *CalendarEvent `json:"event,omitempty"`
}
```

**Features:**
- List available calendars
- Create events with conflict detection
- Get events by date range
- Delete events

**Note:** Google Calendar API not implemented (CalDAV only per plan)

### Rolodex Database Schema

**File:** `bridge/pkg/secretary/schema.sql`

**Tables:**
- `user_contacts` - Contact entries (id, user_id, name, company, title)
- `contact_details` - Encrypted contact data (BLOB with SQLCipher)
- `contact_relationships` - User-agent relationships (doctor-patient, lawyer-client)

**Encryption:** Sensitive fields encrypted using SQLCipher with existing keystore patterns

### Three-Way Consent Matrix Integration

**File:** `bridge/pkg/pii/three_way_consent.go`

```go
type ThreeWayConsentManager struct {
    rooms       map[string]*ConsentRoom
    byRoomID    map[string]*ConsentRoom
    byToken     map[string]*ConsentRoom
    hitlManager *HITLConsentManager
    matrix      MatrixAdapter
}
```

**Flow:**
1. Doctor/Agent initiates consent request
2. Matrix room created with user + agent + bridge participants
3. User receives room invite with consent request details
4. User approves/rejects via Matrix reactions or messages
5. Approval propagates to HITL consent system

**Approval Types:**
- One-time: Single use, expires after 1 hour
- Session: Valid for entire interaction
- Standing: Pre-approved for specific forms

### Guardrails Respected

| Guardrail | Status |
|-----------|--------|
| No external WebDAV services | ✅ Self-hosted only |
| No Google Calendar API | ✅ CalDAV only |
| No favorite locations in rolodex | ✅ Not implemented |
| No automatic approvals | ✅ Manual only |
| No voice approval workflows | ✅ Not implemented |

### Commits (2026-03-15)

```
ee027bc feat(skills): register WebDAV and Calendar skills
e04c4c1 feat(consent): add Three-Way Consent Matrix integration
9da4d6f feat(secretary): add Rolodex contact management system
e7586e4 feat(skills): add Calendar skill with CalDAV support
a9bb248 feat(skills): add WebDAV skill for file operations
```

### Quick Reference Commands

```bash
# Test WebDAV skill
curl -X PROPFIND http://localhost:8080/webdav/

# Test Calendar skill (requires CalDAV server)
curl -X PROPFIND -u user:pass http://localhost:5232/

# Run secretary tests
cd bridge && go test ./pkg/secretary/... -v

# Run PII/consent tests
cd bridge && go test ./pkg/pii/... -v
```


## Phase 11: Secretary Features Completion (2026-03-15)

### Overview

Completed 3 partial implementations to achieve 100% Secretary feature parity:
- **Conditional Branching** - Template engine now filters steps based on conditions
- **Timezone-Aware Scheduling** - Jobs execute at configured timezone
- **Template CRUD RPC** - Full template management via RPC API

### Implementation Details

| Feature | Description | Files Changed |
|--------|-------------|---------------|
| **Conditional Branching** | `evaluateConditions()` filters workflow steps based on conditions | `template_engine.go`, `template_engine_test.go` |
| **Timezone Support** | IANA timezone support with `time.LoadLocation()` | `orchestrator_scheduler.go`, `orchestrator_scheduler_test.go` |
| **Template RPC** | CRUD operations via `secretary.create_template`, etc. | `rpc.go` |

### Conditional Branching Implementation

**File:** `bridge/pkg/secretary/template_engine.go`

```go
func (e *TemplateEngine) evaluateConditions(
    ctx context.Context,
    steps []WorkflowStep,
    variables map[string]string,
) ([]WorkflowStep, error) {
```

The Evaluates conditions for each step:
- Steps without conditions pass through unchanged
- Steps with failing conditions are filtered out
- Supports operators: `eq`, `neq`, `in`, `nin`, `contains`
- Resolves fields: `step.type`, `step.id`, variables

**Tests:** 10 tests covering all operators and field resolution

### Timezone-Aware Job Scheduling

**File:** `bridge/pkg/secretary/orchestrator_scheduler.go`

```go
type SchedulerConfig struct {
    // ... existing fields ...
    Timezone string  // IANA timezone (e.g., "America/New_York")
}

type Scheduler struct {
    // ... existing fields ...
    location *time.Location  // Parsed timezone for time conversion
    timezone string         // Configured IANA timezone string
}

func (s *Scheduler) Start() {
    // ... existing code ...
    
    // Load timezone
    if s.timezone != "" {
        loc, err := time.LoadLocation(s.timezone)
        if err == nil {
            s.location = loc
        } else {
            s.log.Warn("scheduler_timezone_load_failed", ...)
        }
    }
}

func (s *Scheduler) processScheduledJobs() {
    // ... existing code ...
    
    now := time.Now().In(s.location)  // Convert to configured timezone
    // ... rest of implementation
}
```

**Tests:** 6 tests covering multiple IANA timezones

### Template CRUD RPC Methods

**File:** `bridge/pkg/secretary/rpc.go`

**Methods Implemented:**

| Method | Description |
|--------|-------------|
| `secretary.create_template` | Create new task template |
| `secretary.get_template` | Retrieve template by ID |
| `secretary.list_templates` | List templates with optional filter |
| `secretary.update_template` | Update existing template |
| `secretary.delete_template` | Delete template by ID |

**Example Usage:**
```bash
curl -X POST http://localhost:8443/rpc -d '{
    "method": "secretary.create_template",
    "params": {
        "name": "Payment Approval",
        "steps": [...]
    }
}'
```

### Guardrails Respected

| Guardrail | Status |
|-----------|--------|
| No Python/OMO layer | ✅ Pure Go implementation |
| No breaking API changes | ✅ Backwards compatible |
| No new external dependencies | ✅ Uses stdlib only |

### Verification Results

| Agent | Result |
|-------|--------|
| F1 - Gap Compliance | ✅ APPROVE - All 3 gaps implemented |
| F2 - Build & Test | ✅ APPROVE - 138 tests pass/0 fail |
| F3 - Regression Check | ✅ APPROVE - 0 regressions |
| F4 - Python-Free | ✅ APPROVE - 0 Python files |

### Quick Reference Commands

```bash
# Run secretary tests
cd bridge && go test ./pkg/secretary/... -v

# Verify conditional branching
go test -v ./pkg/secretary/... -run TestCondition

# Verify timezone support
go test -v ./pkg/secretary/... -run TestScheduler

# Test template RPC methods (requires running bridge)
curl -X POST http://localhost:8443/rpc -d '{"method":"secretary.list_templates"}'
```

---

## Phase 10: Infrastructure Hardening (2026-03-14)

### Overview

Implementation of remaining infrastructure recommendations: healthchecks for all Docker services, IPv4 preference in IP detection, build error visibility, and container rollback mechanism.

### New Features

| Feature | Description | Files Changed |
|---------|-------------|---------------|
| **Bridge Healthcheck** | Socket-based health check for bridge service | docker-compose-full.yml |
| **Coturn Healthcheck** | UDP port check for TURN server | docker-compose.matrix.yml |
| **Certbot Systemd Check** | Script to verify certbot timer status | scripts/certbot-healthcheck.sh |
| **IPv4 Preference** | Prefer IPv4 in IP detection with IPv6 fallback | docker-compose.matrix.yml, scripts/deploy-infrastructure.sh |
| **Docker Build Wrapper** | Error visibility with --progress=plain | scripts/docker-build-with-errors.sh |
| **Container Rollback** | Tag-based rollback mechanism for containers | scripts/container-rollback.sh |

### Healthcheck Implementations

**Bridge Service (docker-compose-full.yml):**
```yaml
healthcheck:
  test: ["CMD-SHELL", "test -S /run/armorclaw/bridge.sock || exit 1"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 60s
```

**Coturn Service (docker-compose.matrix.yml):**
```yaml
healthcheck:
  test: ["CMD", "nc", "-z", "-u", "localhost", "3478"]
  interval: 30s
  timeout: 10s
  retries: 3
```

**Note:** Bridge uses socket check (no HTTP endpoint). Coturn uses UDP port check on 3478.

### IPv4 Preference Implementation

**Problem:** `api.ipify.org` can return IPv6 on dual-stack systems, causing issues with services expecting IPv4.

**Solution:**
```bash
# Prefer IPv4, fallback to IPv6
EXTERNAL_IP=$(curl -s -4 https://api.ipify.org 2>/dev/null || echo '0.0.0.0')
if echo "$EXTERNAL_IP" | grep -q ':'; then
  echo "Warning: Detected IPv6 address, prefer IPv4 but using IPv6 fallback" >&2
fi
```

Applied to:
- `docker-compose.matrix.yml` (coturn entrypoint)
- `scripts/deploy-infrastructure.sh` (DNS validation)

### Container Rollback Mechanism

**Tag-Based Rollback:**
```bash
# Before deploy - tag current as prev
./scripts/container-rollback.sh tag-current-as-prev

# On failure - rollback to previous
./scripts/container-rollback.sh rollback all

# Preview without executing
./scripts/container-rollback.sh rollback all --dry-run
```

**Supported Services:**
- armorclaw-bridge
- armorclaw-matrix
- armorclaw-coturn
- armorclaw-sygnal
- armorclaw-caddy

**Rollback Limitations:**
- Containers only (no configs or volumes)
- Manual trigger only (no automatic rollback)
- Independent per container (no coordination)

### Certbot Healthcheck Script

**Purpose:** Check if certbot systemd timer is active for certificate renewal.

```bash
./scripts/certbot-healthcheck.sh
# Exit 0: Timer active
# Exit 1: Timer inactive
```

**Checks:**
- `certbot.timer` (standard)
- `certbot-renew.timer` (alternative naming)

### Docker Build Error Visibility

**Usage:**
```bash
# Auto-detect docker-compose vs docker build
./scripts/docker-build-with-errors.sh

# Force specific command
./scripts/docker-build-with-errors.sh --docker-compose
./scripts/docker-build-with-errors.sh --docker-build

# Quiet mode (errors only)
./scripts/docker-build-with-errors.sh -q
```

**Features:**
- `--progress=plain` for full output
- Shows last 50 lines on failure
- Clear error banners with troubleshooting tips
- Supports docker compose, docker-compose, and docker build

### Files Modified

```
docker-compose-full.yml           | +6 (bridge healthcheck)
docker-compose.matrix.yml         | +25 (coturn healthcheck, IPv4 preference)
scripts/deploy-infrastructure.sh  | +3 (IPv4 preference)
scripts/certbot-healthcheck.sh    | +40 (new)
scripts/docker-build-with-errors.sh | +353 (new)
scripts/container-rollback.sh     | +326 (new)
```

### Commits (2026-03-14)

```
96fd460 feat(rollback): add container rollback mechanism
fc7c60b chore: update boulder.json with session IDs
835da8a feat(infra): add healthchecks for bridge/coturn, certbot check, IPv4 preference, docker build wrapper
```

### Guardrails Respected

| Guardrail | Status |
|-----------|--------|
| No bridge HTTP endpoint | ✅ Used socket check |
| No TURN test credentials | ✅ Simple port check only |
| No certificate expiry validation | ✅ Timer check only |
| No build notification/alerting | ✅ Console output only |
| No config or volume rollback | ✅ Containers only |
| No changes to existing healthchecks | ✅ Matrix/Sygnal/Nginx untouched |

### Verification Results

| Agent | Result |
|-------|--------|
| F1 - Plan Compliance (Oracle) | ✅ APPROVE - Must Have [4/4] \| Must NOT Have [6/6] |
| F2 - Infrastructure Health | ✅ APPROVE - Healthchecks [4/4] \| Configs [2/2 valid] |
| F3 - Scope Fidelity (Deep) | ✅ APPROVE - Tasks [6/6 compliant] |

### Quick Reference Commands

```bash
# Check all healthcheck statuses
docker ps --format 'table {{.Names}}\t{{.Status}}'

# Verify IPv4 detection
curl -s -4 https://api.ipify.org

# Check certbot timer
./scripts/certbot-healthcheck.sh

# Build with error visibility
./scripts/docker-build-with-errors.sh --docker-compose

# Rollback preview
./scripts/container-rollback.sh status
./scripts/container-rollback.sh rollback all --dry-run
```

---

## Phase 9: Tunnel Setup & Status Display (2026-03-13)

### Overview

Major improvements to the HTTPS tunnel setup flow for Element X mobile connectivity. Cloudflare Quick Tunnels have been experiencing rate-limiting (HTTP 429/500 errors), so ngrok was added as a fallback option.

### New Features

| Feature | Description | Files Changed |
|---------|-------------|---------------|
| **ngrok Tunnel Option** | Alternative to Cloudflare when rate-limited | deploy/setup-quick.sh |
| **Tunnel Retry Logic** | Cloudflare retries up to 2x on 500 errors | deploy/setup-quick.sh |
| **Service Status Dashboard** | Table showing Bridge/Matrix/Tunnel status | deploy/setup-quick.sh |
| **Element X Credentials Box** | Clear display of homeserver URL, username, password | deploy/setup-quick.sh |
| **Version in Header** | Shows v0.4.1 (from VERSION file) in setup banner | deploy/setup-quick.sh |
| **Manual Tunnel Instructions** | Step-by-step guide when auto-detection fails | deploy/setup-quick.sh |

### Tunnel Options

The setup now offers three tunnel choices:

```
  Options:
  1) Cloudflare Quick Tunnel (Free, instant)
  2) ngrok Tunnel (Free account needed at ngrok.com)
  3) Skip (configure manually later)
```

### Cloudflare Tunnel Improvements

**Retry Logic:**
- Retries up to 2 times on HTTP 500/Internal Server Error
- Detects specific error patterns: `500 Internal Server Error`, `failed to unmarshal`
- Progress dots while waiting (`...........`)
- Clear error messages with actionable alternatives

**Known Issues:**
- Cloudflare Quick Tunnel rate-limits IPs (HTTP 429 "Too Many Requests")
- Service occasionally returns HTML error pages instead of JSON (HTTP 500)
- Free tunnels have no uptime guarantee

### ngrok Tunnel Implementation

**How it works:**
1. Checks if ngrok is installed, installs via apt if needed
2. Verifies ngrok is configured (`ngrok config check`)
3. Starts ngrok as background process (not Docker container)
4. Polls `localhost:4040/api/tunnels` for HTTPS URL
5. Updates Matrix `server_name` to ngrok domain
6. Restarts Conduit to apply changes

**URL Detection:**
```bash
# Supports all ngrok URL formats:
# - *.ngrok-free.app (current free tier)
# - *.ngrok.app (newer format)
# - *.ngrok.io (legacy format)
ngrok_url=$(curl -s http://localhost:4040/api/tunnels | \
  grep -oE 'https://[a-zA-Z0-9.-]+\.(ngrok-free\.app|ngrok\.app|ngrok\.io)')
```

**Manual Setup (if auto-detection fails):**
```bash
# 1. Start ngrok
ngrok http 6167

# 2. Get URL from http://localhost:4040
# 3. Update Matrix config
sudo sed -i 's/^server_name = .*/server_name = "YOUR-NGROK-URL"/' /etc/conduit.toml
docker restart armorclaw-conduit
```

### Service Status Dashboard

The completion screen now shows a formatted status table:

```
╔═══════════════════════════════════════════════════════════════╗
║                    SERVICE STATUS                             ║
╚═══════════════════════════════════════════════════════════════╝

  Service              Status         Details
  ─────────────────────────────────────────────────────────────
  Bridge (ArmorClaw)    ● running      port 8443
  Matrix (Conduit)      ● running      port 6167
  Tunnel (ngrok)        ● active       https://abc-123.ngrok-free.app

  ✓ All core services running
```

**Tunnel Detection:**
- Checks for ngrok process via `pgrep -f "ngrok http"`
- Checks for Cloudflare container via `docker ps`
- Extracts tunnel URL and displays in status

### Element X Credentials Display

Clear boxed format for connection info:

```
╔═══════════════════════════════════════════════════════════════╗
║                  ELEMENT X CREDENTIALS                        ║
╚═══════════════════════════════════════════════════════════════╝

  Homeserver URL:
    https://abc-123.ngrok-free.app

  Username:  @admin:abc-123.ngrok-free.app
  Password:  <generated-password>

  📱 Connect from anywhere via ngrok Tunnel
```

### Configuration File Paths

| File | Path | Purpose |
|------|------|---------|
| Conduit config | `/etc/conduit.toml` | Matrix homeserver config |
| Bridge config | `/etc/armorclaw/config.toml` | ArmorClaw bridge config |
| Conduit data | `/var/lib/conduit` | Matrix database |

### Files Modified

```
deploy/setup-quick.sh  | +200 -80 (ngrok, status, credentials display)
VERSION                | 1 (0.4.1)
```

### Commits (2026-03-13)

```
6d6d51a fix(setup-quick): improve ngrok URL detection with better regex and manual instructions
41d5024 fix(setup-quick): run ngrok directly instead of Docker, detect both tunnel types
01808fb fix(setup-quick): add ngrok fallback and retry logic for tunnel setup
1965c5b fix(setup-quick): improve tunnel setup with retry loop, error handling, and status dashboard
7c056e6 feat(setup-quick): add version number to header banner
```

### Known Limitations

1. **Cloudflare Rate Limiting** - Free Quick Tunnels are rate-limited; wait 10-15 minutes if blocked
2. **ngrok Account Required** - Free account at ngrok.com required for ngrok tunnels
3. **ngrok Free Tier** - URLs change on restart; paid tier provides static domains
4. **Config Path** - Conduit config at `/etc/conduit.toml`, not `/etc/armorclaw/conduit.toml`

### Troubleshooting

**Cloudflare tunnel fails (HTTP 429/500):**
```bash
# Option 1: Wait and retry
# Wait 10-15 minutes, then re-run setup

# Option 2: Use ngrok
ngrok config add-authtoken YOUR_TOKEN
# Re-run setup, choose option 2
```

**ngrok URL not detected:**
```bash
# Check ngrok is running
pgrep -f ngrok

# Check ngrok API
curl http://localhost:4040/api/tunnels

# Manual setup
ngrok http 6167
# Copy URL from browser at http://localhost:4040
sudo sed -i 's/^server_name = .*/server_name = "YOUR-URL"/' /etc/conduit.toml
docker restart armorclaw-conduit
```

**Verify tunnel is working:**
```bash
curl https://YOUR-TUNNEL-URL/_matrix/client/versions
# Should return: {"versions":["v1.0",...]}
```

---

## Phase 8: Setup UX & Connectivity (2026-03-12)

### New Features

| Feature | Description | Files Changed |
|---------|-------------|---------------|
| **Cloudflare Tunnel** | Instant HTTPS for Element X connectivity without domain/SSL | deploy/setup-quick.sh |
| **Element X Credentials** | Display credentials at install completion | deploy/setup-quick.sh |
| **Go 1.26.1 Upgrade** | Updated Go version from 1.24 to 1.26.1 | deploy/container-setup.sh |
| **Go 1.22+ Minimum** | Enforce minimum Go version requirement | deploy/container-setup.sh |
| **QR Code Auto-Install** | Auto-install qrencode for terminal QR display | deploy/setup-quick.sh |
| **conduit.toml Creation** | Generate Conduit config file during setup | deploy/setup-quick.sh |
| **Homeserver URL Fix** | Correct URL format (remove stray dash) | deploy/setup-quick.sh |
| **ANSI Color Rendering** | Fix -e flag for echo commands | deploy/setup-quick.sh |

### Cloudflare Tunnel Integration

Element X requires HTTPS to connect. The setup now offers an optional Cloudflare Tunnel for instant HTTPS:

```bash
# Automatic prompt during setup
? Enable Cloudflare Tunnel for instant HTTPS? (y/n)

# If accepted:
# - Starts cloudflare/cloudflared:latest container
# - Detects tunnel URL from container logs
# - Updates Conduit server_name to match tunnel domain
# - Displays tunnel URL for Element X connection
```

**Manual fallback:**
```bash
docker run -d --name armorclaw-tunnel cloudflare/cloudflared:latest \
  tunnel --url http://host.docker.internal:6167
```

### QR Code Generation

| Component | Implementation | Status |
|-----------|---------------|--------|
| **Terminal QR** | qrencode ASCII output | ✅ Working |
| **Config QR** | `armorclaw://config?d=<base64>` | ✅ Working |
| **Provisioning** | Token-based claim flow | ✅ Working |
| **Token Expiration** | 10min setup, 24hr config | ✅ Working |

**Two QR Flows:**

1. **Config QR** (bridge/pkg/qr/public.go):
   - Direct config QR with signed payload
   - Contains: Matrix homeserver, RPC URL, WebSocket URL, Push Gateway, Server Name
   - Uses HMAC-SHA256 signing with random 32-byte key
   - Returns PNG image, deep link, web URL

2. **Provisioning QR** (bridge/pkg/provisioning/):
   - Token-based claim flow with role assignment
   - First device claims OWNER role, subsequent get NONE
   - Roles persisted to `provisioning_roles.json`
   - Requires `ARMORCLAW_PROVISIONING_SECRET` env var

**QR Generation Flow:**
1. `bridge/pkg/qr/public.go` - GenerateConfigQR creates signed config
2. `bridge/pkg/provisioning/` - Token management and claim flow
3. `bridge/pkg/http/server.go` - HTTP endpoints: `/qr/config`, `/qr/image`
4. `deploy/setup-quick.sh` - Terminal QR display via qrencode

**Known Limitations:**
- QRManager tokens are in-memory only (max 10 active)
- Dual signing keys: QR package uses random keys, provisioning uses env-configured secret

### Installer Status Summary

| Component | Version | Purpose | Status |
|-----------|---------|---------|--------|
| install.sh | 1.4.3 | Stage-0 bootstrap, GPG verification | ✅ Production |
| installer-v5.sh | 5.0.0 | Stage-1 full installer, Docker setup | ✅ Production |
| setup-quick.sh | 1.0 | Quick setup, Conduit, Cloudflare Tunnel | ✅ Production |
| container-setup.sh | 0.4.1 | Container setup wizard | ✅ Production |

### Matrix/Conduit Status

| Feature | Implementation | Status |
|---------|---------------|--------|
| Auto-install Conduit | `docker run matrixconduit/matrix-conduit` | ✅ Working |
| conduit.toml generation | Dynamic config creation | ✅ Working |
| Admin user creation | Shared-secret API | ✅ Working |
| Bridge user registration | Auto-register on Conduit | ✅ Working |
| Matrix event bus | Zero-allocation receive path | ✅ Working |
| Homeserver URL format | `http://IP:6167` or tunnel URL | ✅ Fixed |

### Quicksetup Features

| Feature | Description | Status |
|---------|-------------|--------|
| Non-interactive mode | `--non-interactive` or env vars | ✅ Working |
| API key env vars | `OPENROUTER_API_KEY`, `ZAI_API_KEY`, `OPEN_AI_KEY` | ✅ Working |
| Lockfile protection | Prevents parallel installs | ✅ Working |
| Persistent logging | `/var/log/armorclaw-setup.log` | ✅ Working |
| Docker Compose detection | `docker compose` vs `docker-compose` | ✅ Working |
| Idempotent | Safe to re-run | ✅ Working |
| Cloudflare Tunnel | Optional instant HTTPS | ✅ Working |

**Go TUI Wizard** (bridge/internal/wizard/):
| File | Purpose |
|------|---------|
| wizard.go | Main TUI using charmbracelet/huh (v0.3.6) |
| quick.go | Two-page form: Provider+Key, Admin+Deploy |
| catwalk.go | Dynamic provider discovery via localhost:8080 |
| profile.go | Profile selection (Quick vs Enterprise) |

**Environment Variables for Go Wizard:**
| Variable | Purpose |
|----------|---------|
| `ARMORCLAW_API_KEY` | API key (required for non-interactive) |
| `ARMORCLAW_PROFILE` | quick/enterprise (default: quick) |
| `ARMORCLAW_SERVER_NAME` | Server hostname/IP |
| `ARMORCLAW_ADMIN_PASSWORD` | Admin password (auto-generated if empty) |

### Files Modified

```
deploy/setup-quick.sh      | 131 + (Cloudflare Tunnel, QR display)
deploy/container-setup.sh  |  15 + (Go 1.26.1, min version)
6 files changed, 150 insertions(+), 30 deletions(-)
```

---

## Phase 7: Deployment Audit Fixes (2026-03-11)

### Critical Issues Resolved

| Issue | Fix | Files Changed |
|-------|-----|---------------|
| Network subnet overlap (172.20.0.0/16 vs /24) | Narrowed all /16 subnets to /24 | docker-compose-full.yml, docker-compose.matrix.yml, deploy/matrix/docker-compose.matrix.yml |
| Test config open registration | Disabled `allow_registration`, added `registration_shared_secret` | tests/matrix-test-server/conduit.toml |
| Missing bridge_keystore volume | Added `bridge_keystore:/var/lib/armorclaw` | docker-compose-full.yml |
| Missing health check condition | Added `condition: service_healthy` | docker-compose-full.yml |
| Conduit path inconsistency | Unified to `/var/lib/matrix-conduit` | docker-compose-full.yml |
| Config mount path inconsistency | Unified to `/etc/conduit/conduit.toml` | docker-compose-full.yml |
| Hardcoded TURN secrets in test | Replaced with empty values | tests/matrix-test-server/conduit.toml |
| Hardcoded STUN server | Made configurable via `ARMORCLAW_WEBRTC_STUN_SERVER` | bridge/pkg/config/loader.go |
| Auto-generated secrets without warning | Added startup warnings | bridge/cmd/bridge/main.go |
| Fake URLs in config.example.toml | Replaced with empty strings + security guidance | bridge/config.example.toml |

### Network Subnet Allocation (Final)

```
172.20.0.0/24 - docker-compose-full.yml (armorclaw-net)
172.21.0.0/24 - docker-compose.bridge.yml (bridge-net)
172.23.0.0/16 - deploy/browser (browser-internal)
172.24.0.0/24 - deploy/matrix (matrix-internal)
172.25.0.0/24 - deploy/matrix (matrix-public)
172.26.0.0/24 - docker-compose.matrix.yml (matrix-net)
```

**All subnets are now unique and non-overlapping.**

### New Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `ARMORCLAW_WEBRTC_STUN_SERVER` | `stun.l.google.com:19302` | Override default STUN server |
| `ARMORCLAW_WEBRTC_TURN_URL` | (none) | TURN server URL |
| `ARMORCLAW_WEBRTC_TURN_SECRET` | (none) | TURN shared secret for production |

### Path Consistency Achieved

- Conduit data: `/var/lib/matrix-conduit` (consistent across stacks)
- Conduit config: `/etc/conduit/conduit.toml` (consistent mount path)
- CONDUIT_CONFIG env: `/etc/conduit/conduit.toml` (matches mount)

### Files Modified

```
bridge/cmd/bridge/main.go               |  1 + (WebRTC warning)
bridge/config.example.toml              | 18 + (security placeholders)
bridge/pkg/config/loader.go             | 15 + (STUN/TURN env vars)
deploy/matrix/docker-compose.matrix.yml |  4 + (subnet fix)
docker-compose-full.yml                 | 17 + (volume, health, subnet, paths)
docker-compose.matrix.yml               |  2 + (subnet fix)
tests/matrix-test-server/conduit.toml   | 17 + (security hardening)
7 files changed, 51 insertions(+), 23 deletions(-)
```

---

## Phase 6: VPS Deployment (2026-03-11)

### ✅ Deployment Fixes

| Issue | Fix | Status |
|-------|-----|--------|
| GPG signature verification failing | Generated new signing key, re-signed installer | ✅ Fixed |
| sed corrupting .sig files | Excluded .sig from line-ending normalization | ✅ Fixed |
| Binary 404 errors | Disabled binary download until release created | ✅ Fixed |
| Matrix not auto-installing | Auto-installs Conduit in quickstart mode | ✅ Fixed |

### Key Changes (v0.6.0)

1. **GPG Key Rotation** - New signing key fingerprint:
   - Old: `A1482657223EAFE1C481B74A8F535F90685749E0` (expired)
   - New: `55AD64228EF6B4A342DA480A09C43CFA8AC93062`

2. **Installer Signature Fix** - The `sed -i 's/\r$//'` command was corrupting GPG signatures. Fixed by excluding `.sig` files from normalization.

3. **Binary Download Disabled** - Set `USE_BINARY=false` until GitHub release with prebuilt binaries is created.

4. **Quickstart Auto-Matrix** - `setup-quick.sh` now auto-installs Matrix Conduit instead of prompting, improving VPS deployment experience.

---

## Phase 4 & 5 Completion Status (2026-03-10)

### ✅ All Success Conditions Met

| Condition | Status | Evidence |
|-----------|--------|----------|
| Quick Start completes with minimal answers | ✅ Pass | Non-interactive mode with env vars |
| Bridge running | ✅ Pass | Process running in container |
| Matrix running | ✅ Pass | Conduit responding on port 6167 |
| SQLCipher keystore initialized | ✅ Pass | Hardware-derived key, encrypted DB |
| API key stored | ✅ Pass | Environment variable support (OPENROUTER_API_KEY, ZAI_API_KEY) |
| OWNER claimed | ✅ Pass | @admin:armorclaw.local registered |
| Provisioning QR generated | ✅ Pass | QR code generation works |
| !status works | ✅ Pass | Via matrix.status RPC |
| Normal AI chat works | ✅ Pass | Env var support for API keys |
| Agent Studio works | ✅ Pass | studio.stats returns 8 skills, 10 PII fields |
| Browser subsystem works | ✅ Pass | browser.list returns 0 jobs (healthy) |
| MCP/skills system works | ✅ Pass | 17 SKILL.md files created |

### Key Updates (v0.5.3)

1. **Environment Variable API Keys** - API keys now read from environment variables instead of stored in keystore:
   - `OPEN_AI_KEY` → openai-default (OpenAI provider)
   - `OPENROUTER_API_KEY` → openrouter-default (OpenRouter provider)
   - `ZAI_API_KEY` → xai-default (xAI provider)
   
   This keeps API keys in .zshrc and never persists them to disk.

2. **SQLCipher Linking** - Bridge binary now links against SQLCipher library with CGO:
   ```bash
   export CGO_ENABLED=1
   export CGO_CFLAGS="-I/usr/include/sqlcipher"
   export CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu -lsqlcipher"
   ```

3. **Browser Skills** - Created 21 SKILL.md files for Chrome DevTools MCP integration:
   - Safe primitives: navigate, click, fill, wait_for, screenshot, snapshot, list_pages, select_page, resize, emulate
   - Workflow skills: extract_page, login_assist, form_submit, upload_document, trace_performance
   - Guarded skills: eval_privileged, network_inspect, console_inspect, lighthouse_audit, memory_snapshot, fill_with_pii (require approval)

4. **Matrix Admin Bootstrap** - Admin user created:
   - Username: admin
   - Password: ArmorClaw2026!
   - Homeserver: armorclaw.local (accessible via localhost:6167)

### Build Command

```bash
# Build bridge with SQLCipher support
export PATH=/home/mink/go/go/bin:$PATH
export GO111MODULE=on
export CGO_ENABLED=1
export CGO_CFLAGS="-I/usr/include/sqlcipher"
export CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu -lsqlcipher"
cd /home/mink/src/ArmorClaw/bridge
go build -o build/armorclaw-bridge ./cmd/bridge
```

### Running with Environment Variables

```bash
# Source your .zshrc to get API keys
source ~/.zshrc

# Run bridge with env vars
sudo env OPENROUTER_API_KEY="$OPENROUTER_API_KEY" ZAI_API_KEY="$ZAI_API_KEY" \
  /home/mink/src/ArmorClaw/bridge/build/armorclaw-bridge
```

---

## Phase 1 Bring-Up Status (2026-03-10)

### ✅ Completed

| Component | Status | Evidence |
|-----------|--------|----------|
| Quickstart image builds | ✅ Pass | `docker build -f Dockerfile.quickstart -t armorclaw-fixed:phase1 .` |
| Bridge binary runs | ✅ Pass | Process running at PID 476 |
| Bridge socket created | ✅ Pass | `/run/armorclaw/bridge.sock` exists |
| Conduit Matrix starts | ✅ Pass | HTTP 200 on `/_matrix/client/versions` |
| Admin user registered | ✅ Pass | `@admin:152.37.165.193` created |
| QR code generated | ✅ Pass | ArmorChat provisioning QR displayed |

### Verification Commands

```bash
# Conduit responds
curl http://localhost:6167/_matrix/client/versions
# → {"versions":["r0.5.0",...]

# Bridge socket exists
docker exec armorclaw test -S /run/armorclaw/bridge.sock && echo "OK"
# → OK

# Bridge process running
docker exec armorclaw ps aux | grep armorclaw-bridge
# → /opt/armorclaw/armorclaw-bridge (PID 476)
```

### Known Issues

- RPC methods return "method not found" - bridge may need configuration or different method names
- API key injection via RPC failed (needs manual `store_key` call)
- Provisioning not available (OWNER claim via QR code)

### Files Modified

- `Dockerfile.quickstart` - Added Go 1.24 builder stage with CGO + SQLCipher
- `deploy/container-setup.sh` - Added `CONDUIT_CONFIG` env var and config mount

---

## Executive Summary

ArmorClaw v4.5.0 provides a **production-ready AI agent platform** that runs AI agents 24/7 on your VPS, controlled from your phone via Matrix (E2EE) or ArmorChat mobile app.

**Core Components:**

**Key Design Principle:** Zero persistent secrets on disk. All credentials are read from environment variables at runtime. API keys stay in your shell profile (.zshrc), never written to the encrypted keystore.
- **ArmorClaw Bridge** - Native Go binary with encrypted keystore, Matrix adapter, JSON-RPC server, Provider Registry
- **Matrix/Conduit** - Integrated homeserver for E2EE messaging (optional, auto-detected and installed)
- **Provider Registry** - Embedded registry with 12+ pre-configured AI providers
- **Agent Studio** - Agent management with skills, PII access control, and MCP approval workflow
- **Browser Automation** - Playwright-based service with anti-detection and PII protection
- **Skills Executor** - Built-in skills (data_analyze, web_extract, email_send, web_search, file_read, slack_message)
- **Memory Store** - Vector-based memory for agents
- **Catwalk Integration** - Dynamic AI provider/model discovery and runtime switching

**Deployment Options:**
- **Quick Install:** Single-command installer with auto-detection
- **Matrix Install:** Integrated Conduit setup for instant QR provisioning
- **Bridge-only:** Install bridge only (existing Matrix or external)

**Key Design Principles:**
- Zero persistent secrets on disk (SQLCipher + XChaCha20-Poly1305)
- Bridge runs as native binary (not in container) for security
- Agent containers isolated with Unix socket communication
- End-to-end encryption for all Matrix communication
- Pull-based architecture (agents request, bridge validates)

---

## Quickstart Flow Diagram (v4.5.0)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCLAW INSTALLATION FLOW (v4.5.0)           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  USER RUNS: curl -fsSL https://.../install.sh | bash                     │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 1. INSTALL.SH (Stage-0 Bootstrap)                            │   │
│  │    • Download installer-v5.sh from GitHub                        │   │
│  │    • Verify GPG signature (A1482657...)                         │   │
│  │    • Verify SHA256 checksum                                      │   │
│  │    • Download bridge binary from GitHub Releases                   │   │
│  │    • Check: curl, sha256sum, gpg, mktemp, sed                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 2. GO WIZARD (armorclaw-bridge container-setup)                     │   │
│  │    • Check env vars FIRST (tryNonInteractive)                      │   │
│  │      - If OPENROUTER_API_KEY or ZAI_API_KEY set → non-interactive  │   │
│  │      - Server name passed from host (ARMORCLAW_SERVER_NAME)        │   │
│  │    • Else: Check terminal (TTY, color support, size)               │   │
│  │    • Launch Huh? TUI wizard if terminal OK                         │   │
│  │      - Step 1 of 2: AI Provider + API Key (from env vars)          │   │
│  │      - Step 2 of 2: Admin Password + Deploy confirmation           │   │
│  │    • Output: /tmp/armorclaw-wizard.json + env vars for secrets     │   │
│  │    • Lockfile protection (flock)                                 │   │
│  │    • Persistent logging (/var/log/armorclaw/install.log)          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 3. SETUP-WIZARD.SH (Go TUI or Non-Interactive)                │   │
│  │    • Check env vars (ARMORCLAW_API_KEY triggers non-interactive)   │   │
│  │    • Detect terminal (TTY, color support, size)                   │   │
│  │    • Launch Huh? TUI wizard or use env vars                    │   │
│  │      Step 1: AI Provider selection (with registry)               │   │
│  │      Step 2: API key entry                                     │   │
│  │      Step 3: Admin credentials + deploy confirmation               │   │
│  │    • Generate randomized admin username (armor-admin-xxxxxx)        │   │
│  │    • Generate admin password (or use ARMORCLAW_ADMIN_PASSWORD)     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 4. SETUP-QUICK.SH (Infrastructure Setup)                       │   │
│  │    • Pre-flight checks (Docker, network, disk)                  │   │
│  │    • Create directories (/etc/armorclaw, /var/lib/armorclaw)     │   │
│  │    • Detect if Matrix is running                                 │   │
│  │    • Option: Offer to install Conduit if missing                  │   │
│  │    • Generate self-signed SSL certificate                         │   │
│  │    • Load Provider Registry (12+ providers embedded)              │   │
│  │    • Configure AI provider (OpenAI, Anthropic, Zhipu, etc.)     │   │
│  │    • Create admin user via shared-secret API (zero-touch)          │   │
│  │    • Write config.toml                                          │   │
│  │    • Initialize SQLCipher keystore                                │   │
│  │    • Initialize Agent Studio database (studio.db)                    │   │
│  │    • Start Matrix stack (if installing Conduit)                   │   │
│  │      docker run matrixconduit/matrix-conduit:latest               │   │
│  │    • Register bridge user on Conduit                             │   │
│  │    • Start bridge (native binary on host)                         │   │
│  │      sudo ./armorclaw-bridge                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 5. BOOTSTRAP-ADMIN (Post-Setup)                             │   │
│  │    • Wait for bridge.sock to appear                            │   │
│  │    • Inject API key via RPC (store_key method)                     │   │
│  │    • Create "ArmorClaw Bridge" room on Matrix                     │   │
│  │    • Auto-claim OWNER role for admin via provisioning.claim        │   │
│  │    • Generate QR code for ArmorChat provisioning                   │   │
│  │    • Display connection info + credentials                        │   │
│  │    • Wait for bridge process                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                          ARMORCLAW RUNNING   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Provider Registry Architecture (v4.5.0)

### Embedded Providers

ArmorClaw includes an **embedded provider registry** with 12+ pre-configured AI providers, eliminating the need for manual setup.

**Default Providers:**

| Provider | ID | Protocol | Base URL | Aliases |
|----------|-----|----------|-----------|----------|
| OpenAI | `openai` | OpenAI | https://api.openai.com/v1 | - |
| Anthropic | `anthropic` | Anthropic | https://api.anthropic.com/v1 | - |
| Google | `google` | OpenAI | https://generativelanguage.googleapis.com/v1 | - |
| xAI | `xai` | OpenAI | https://api.x.ai/v1 | - |
| OpenRouter | `openrouter` | OpenAI | https://openrouter.ai/api/v1 | - |
| **Zhipu AI** | `zhipu` | OpenAI | https://api.z.ai/api/paas/v4 | `zai`, `glm` |
| **DeepSeek** | `deepseek` | OpenAI | https://api.deepseek.com/v1 | - |
| **Moonshot AI** | `moonshot` | OpenAI | https://api.moonshot.ai/v1 | - |
| NVIDIA NIM | `nvidia` | OpenAI | https://integrate.api.nvidia.com/v1 | - |
| **Groq** | `groq` | OpenAI | https://api.groq.com/openai/v1 | - |
| Cloudflare | `cloudflare` | OpenAI | https://gateway.ai.cloudflare.com/v1 | - |
| Ollama | `ollama` | OpenAI | http://localhost:11434/v1 | - |

**Registry Features:**

- **Embedded fallback:** Registry embedded in bridge binary (no external dependency)
- **Dynamic loading:** Can load from `/etc/armorclaw/providers.json` if exists
- **Remote download:** Supports `ARMORCLAW_PROVIDERS_URL` for remote registry
- **Alias resolution:** Users can type `zai`, `zhipu`, or `glm` to use Zhipu AI
- **Protocol abstraction:** All providers expose OpenAI-compatible interface

### Catwalk Integration (Dynamic Discovery)

ArmorClaw integrates with **Catwalk** for dynamic AI provider/model discovery:

**Features:**
- **HTTP client** for querying Catwalk API
- **Dynamic provider list** in quickstart wizard (falls back to embedded if Catwalk unavailable)
- **Runtime AI switching** via Matrix commands:
  - `/ai providers` - List available providers
  - `/ai models <provider>` - List models for a provider
  - `/ai switch <provider> <model>` - Switch provider and model
  - `/ai status` - Show current configuration

**Implementation:**
- `bridge/internal/wizard/catwalk.go` - Catwalk HTTP client
- `bridge/internal/ai/runtime.go` - AI runtime with provider switching
- `bridge/pkg/matrixcmd/handler.go` - `/ai` command handler
- `bridge/Dockerfile` - Downloads Catwalk v0.28.3 at build time

---

## Bridge Architecture (v4.5.0)

### Native Binary Deployment

**Critical Design Decision:** The ArmorClaw bridge runs as a **native binary on the host**, not in a Docker container. This is intentional for security:

- **Docker socket access:** Scoped Docker API access for container management
- **Host filesystem access:** Read/write to configs, secrets, logs
- **Unix socket:** `/run/armorclaw/bridge.sock` for agent communication
- **Performance:** Zero Docker overhead for the bridge itself

**To run the bridge:**
```bash
sudo ./bridge/build/armorclaw-bridge
```

### Package Structure

```
bridge/
├── cmd/
│   ├── bridge/           # Main bridge binary (3,000+ lines)
│   ├── bootstrap-admin/   # Admin bootstrap tool
│   └── secureclaw/     # Secure bridge variant
├── pkg/                # Public packages (library code)
│   ├── adapters/         # Multi-protocol adapters (Matrix, Slack, Discord)
│   ├── admin/           # Admin RPC methods
│   ├── agent/           # Agent state machine and integration
│   ├── api/             # HTTP API layer
│   ├── appservice/      # Matrix appservice support
│   ├── audio/           # Audio processing
│   ├── audit/           # Audit logging
│   ├── auth/            # Authentication and authorization
│   ├── browser/         # Browser service client (Go)
│   ├── budget/           # Budget/FinOps controller
│   ├── config/          # Configuration management (TOML)
│   ├── crypto/          # Cryptographic primitives
│   ├── dashboard/        # Admin dashboard
│   ├── discovery/       # Service discovery
│   ├── docker/          # Docker client (scoped)
│   ├── enforcement/      # Policy enforcement
│   ├── errors/          # Error handling
│   ├── eventbus/        # Event bus implementation
│   ├── eventlog/        # Structured event logging
│   ├── ffi/             # Foreign function interfaces
│   ├── ghost/           # Ghost mode (stealth)
│   ├── health/          # Health checks
│   ├── http/            # HTTP client/server utilities
│   ├── invite/          # Invitation management
│   ├── keystore/        # SQLCipher encrypted keystore
│   ├── license/          # License validation
│   ├── logger/          # Logging infrastructure
│   ├── lockdown/        # Lockdown mode enforcement
│   ├── matrix/          # Matrix SDK integration
│   ├── matrixcmd/       # Matrix command handlers
│   ├── notification/    # Push notifications
│   ├── pii/             # PII detection and redaction
│   ├── plugin/          # Plugin system
│   ├── providers/       # Provider registry (NEW v4.5.0)
│   ├── provisioning/    # Provisioning API (QR codes)
│   ├── push/            # Push notification gateway
│   ├── qr/              # QR code generation
│   ├── queue/           # Job queue (browser)
│   ├── recovery/        # Recovery procedures
│   ├── runtime/         # Container runtime (Docker, Firecracker, Containerd)
│   ├── secrets/         # Secrets management
│   ├── securerandom/    # Secure random generation
│   ├── security/        # Security primitives
│   ├── setup/           # Setup utilities
│   ├── socket/          # Unix socket server
│   ├── sso/             # Single sign-on
│   ├── studio/          # Agent Studio (agents, skills, MCP)
│   ├── trust/           # Trust management
│   ├── ttl/             # TTL-based cache
│   ├── turn/            # TURN/STUN configuration
│   ├── voice/           # Voice/WebRTC
│   ├── webrtc/          # WebRTC implementation
│   └── websocket/       # WebSocket support
└── internal/           # Internal modules (bridge-specific)
    ├── adapter/         # Matrix adapter implementation
    ├── ai/              # AI client runtime (multi-provider)
    ├── agent/           # Agent runtime (container management)
    ├── cache/           # LRU cache implementation
    ├── capability/       # Capability detection
    ├── events/          # Matrix event bus
    ├── executor/        # Agent execution engine
    ├── memory/          # Vector-based memory store
    ├── metrics/         # Agent metrics
    ├── petg/            # Catwalk gateway integration
    ├── router/          # Request routing
    ├── skills/          # Skills executor (data_analyze, web_extract, etc.)
    ├── speculative/     # Speculative execution
    ├── trace/           # Distributed tracing
    └── wizard/          # Setup wizard (Go TUI)
```

### Key Internal Modules

**Skills Executor** (`internal/skills/`)
- **Built-in skills:**
  - `data_analyze` - Structured data analysis
  - `web_extract` - Web content extraction
  - `email_send` - Email sending
  - `web_search` - Web search queries
  - `file_read` - Secure file reading
  - `slack_message` - Slack integration
- **Policy engine:** Allowlist, denylist, rate limiting
- **SSRF protection:** Prevent server-side request forgery
- **Schema validation:** Input validation for all skills

**Memory Store** (`internal/memory/`)
- **Checkpoint system:** Agent state snapshots
- **Batch processing:** Efficient vector operations
- **Store interface:** Pluggable backends

**Agent Runtime** (`internal/agent/`)
- **State machine:** Agent lifecycle management
- **Container orchestration:** Spawn, monitor, stop agents
- **Resource limits:** CPU, memory, disk quotas

**Executor Engine** (`internal/executor/`)
- **Parallel execution:** Concurrent task execution
- **Error handling:** Retry logic with exponential backoff
- **Circuit breakers:** Prevent cascading failures

**Router** (`internal/router/`)
- **Request routing:** Matrix commands → handlers
- **Cache layer:** LRU cache for common queries
- **Middleware:** Logging, auth, rate limiting

**Capabilities** (`internal/capability/`)
- **Capability detection:** Agent feature detection
- **Policy enforcement:** What agents can/cannot do

---

## Matrix Integration (v4.5.0)

### Conduit Setup Options

**Option A: Integrated Setup (Recommended)**

The installer auto-detects if Matrix is running and offers to install Conduit:

```bash
# Quick install with Matrix
export ARMORCLAW_API_KEY=sk-your-key
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

**What happens:**
1. Installer detects no Matrix server running
2. Offers to install Conduit homeserver
3. Runs `docker run matrixconduit/matrix-conduit:latest`
4. Configures bridge to connect to local Conduit
5. Generates QR code for ArmorChat provisioning

**Option B: Bridge-Only Mode**

If you have an existing Matrix server:

```bash
# Connect to existing Matrix
export ARMORCLAW_API_KEY=sk-your-key
export MATRIX_SERVER_URL=https://your.matrix.server
export MATRIX_ACCESS_TOKEN=your-token
curl -fsSL ... | bash
```

### Matrix Event Bus Improvements

The bridge uses a **high-throughput event bus** with zero-allocation receive path:

**Key Features:**
- **Zero-allocation receive path:** Pre-allocated batch buffers
- **Instant wake-up:** Events delivered within 25ms, not polling storms
- **Slow consumer detection:** Cursor guard prevents message loss
- **Context cancellation:** Proper timeout handling prevents indefinite blocking

**RPC Methods:**
| Method | Description |
|--------|-------------|
| `matrix.status` | Returns connection health and user info |
| `matrix.login` | Dynamic login through bridge |
| `matrix.send` | Message sending via adapter |
| `matrix.receive` | Long-polling with cursor + timeout |

---

## Install Script Flow (v0.4.2)

The `install.sh` script orchestrates the entire deployment process:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     INSTALL.SH FLOW (v0.4.2)                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  USER RUNS: curl -fsSL .../install.sh | bash                                │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 1. PREFLIGHT CHECKS                                                  │   │
│  │    • Verify Docker is installed and running                         │   │
│  │    • Check for root/sudo permissions                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 2. AUTO-DETECTION (on host, before container)                        │   │
│  │    • Detect available ports (8443, 6167, 5000 or fallback)          │   │
│  │    • Detect server IP: ip route get 1                               │   │
│  │    • Collect env vars from user's shell                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 3. DOCKER RUN (passes env vars to container)                        │   │
│  │    docker run ... \                                                 │   │
│  │      -e ARMORCLAW_SERVER_NAME=<detected-ip> \                       │   │
│  │      -e OPENROUTER_API_KEY=<from-shell> \                           │   │
│  │      -e ZAI_API_KEY=<from-shell> \                                  │   │
│  │      -e ARMORCLAW_PROFILE=<if-set> \                                │   │
│  │      -e ARMORCLAW_ADMIN_PASSWORD=<if-set> \                         │   │
│  │      ...                                                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│                     CONTAINER STARTS (see Quickstart Flow)                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Why Host-Side IP Detection?

Container-side IP detection returns the **container's internal IP** (172.17.x.x), not the host's public IP. This would break ArmorChat connectivity. The install script detects the IP on the host and passes it via `ARMORCLAW_SERVER_NAME`.

### Non-Interactive Mode

```bash
# Minimal - just API keys (IP auto-detected)
# Keys are read from environment variables (OPENROUTER_API_KEY, ZAI_API_KEY, etc.)
# Add to your .zshrc:
export OPENROUTER_API_KEY=sk-or-v1-xxx
export ZAI_API_KEY=xxx

# Then run:
source ~/.zshrc
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash

# With explicit server
export OPENROUTER_API_KEY=sk-or-v1-xxx
export ARMORCLAW_SERVER_NAME=192.168.1.50
curl -fsSL ... | bash
```

---

## Agent Studio (v4.5.0)

### Overview

Agent Studio provides a **no-code interface** for creating and managing AI agents through Matrix chat commands or JSON-RPC.

### Components

**Skill Registry** (`bridge/pkg/studio/registry.go`)
- 8+ built-in skills (data_analyze, web_extract, email_send, web_search, file_read, slack_message)
- Custom skill support via JSON schema
- Skill metadata (name, description, parameters, security level)

**PII Registry** (`bridge/pkg/studio/`)
- 10+ default PII fields with sensitivity levels (low/medium/high/critical)
- Automatic regex-based redaction (credit cards, SSN, email, phone, API keys)
- BlindFill references for secure PII injection

**Resource Profiles**
- 3 tiers: low, medium, high
- Memory/CPU limits per tier
- Disk quota enforcement

**Agent Factory** (`bridge/pkg/studio/factory.go`)
- Agent definition creation
- Instance spawning with Docker
- Security hardening applied automatically

**MCP Approval Workflow** (`bridge/pkg/studio/mcp_approval.go`)
- Role-based access control for external MCP connections
- Pending approval queue
- Admin approval/rejection workflow

### Matrix Commands

```
!agent help              Show command help
!agent create <name>     Start interactive wizard
!agent list-skills       List available skills
!agent list-pii          List PII fields
!agent create <name>     Create agent definition
!agent list              List agent definitions
!agent spawn <id>        Spawn agent instance
!agent stop <instance>   Stop running instance
!agent stats             Show statistics
```

### RPC Methods

| Method | Description |
|--------|-------------|
| `studio.create_agent` | Create agent definition |
| `studio.list_skills` | List available skills |
| `studio.list_agents` | List agent definitions |
| `studio.spawn_agent` | Spawn agent instance |
| `studio.stop_agent` | Stop agent instance |
| `studio.stats` | Show studio statistics |
| `studio.list_mcps` | List available MCPs |
| `studio.request_mcp_approval` | Request MCP access |
| `studio.approve_mcp_request` | Approve MCP request (admin) |
| `studio.reject_mcp_request` | Reject MCP request (admin) |

---

## Browser Automation (v4.5.0)

### Architecture

The browser automation system consists of three main components:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     BROWSER AUTOMATION ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│  │ ArmorChat       │     │ Bridge          │     │ Browser Service │       │
│  │ (Android)       │     │ (Go)            │     │ (TypeScript)    │       │
│  │                 │     │                 │     │                 │       │
│  │ JSON-RPC        │────►│ Browser Client  │────►│ Playwright      │       │
│  │ Matrix Events   │     │ Queue Processor │     │ Stealth Mode    │       │
│  │                 │     │                 │     │                 │       │
│  └─────────────────┘     └─────────────────┘     └─────────────────┘       │
│         │                       │                       │                   │
│         │                       ▼                       │                   │
│         │              ┌─────────────────┐              │                   │
│         │              │ Job Queue       │              │                   │
│         │              │ (SQLite)        │              │                   │
│         │              └─────────────────┘              │                   │
│         │                       │                       │                   │
│         └───────────────────────┴───────────────────────┘                   │
│                     Matrix Events (status, response)                             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### Browser Service (TypeScript/Playwright)

**Location:** `browser-service/`

**Features:**
- Playwright-based headless browser automation
- Anti-detection / stealth mode
- Screenshot capture with element cropping
- Form filling with PII injection
- Cookie and session management
- Proxy rotation support

**API Endpoints:**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Service health check |
| `/navigate` | POST | Navigate to URL |
| `/fill` | POST | Fill form fields |
| `/click` | POST | Click element |
| `/extract` | POST | Extract page data |
| `/screenshot` | POST | Capture screenshot |
| `/status` | GET | Current browser state |

### Browser Client (Go)

**Location:** `bridge/pkg/browser/`

**Components:**
- `client.go` - HTTP client for browser-service API
- `processor.go` - Job queue processor with retry logic
- `browser.go` - Core browser types and interfaces

**Configuration:**
```toml
[browser]
enabled = true
service_url = "http://localhost:3001"
timeout = 30
max_retries = 3
retry_delay = 2

[browser.stealth]
enabled = true
fingerprint_seed = ""

[browser.queue]
max_workers = 3
max_depth = 100
```

### Deployment

**Docker Compose:** `deploy/browser/docker-compose.browser.yml`

```yaml
services:
  browser-service:
    build: ../../browser-service
    ports:
      - "3001:3001"
    environment:
      - NODE_ENV=production
      - STEALTH_MODE=true
    cap_add:
      - SYS_ADMIN
    security_opt:
      - seccomp:unconfined
```

---

## Docker Compose Architecture (v4.5.0)

### Topology Separation

ArmorClaw uses a **two-tier network topology** for security:

```
┌─────────────────────────────────────────────────────────────┐
│                        HOST MACHINE                        │
│                                                        │
│  ┌─────────────────┐                                    │
│  │ ArmorClaw Bridge│ ← Unix Socket /run/armorclaw/bridge.sock│
│  │ (Native Binary) │                                    │
│  └────────┬────────┘                                    │
│           │                                               │
│  ┌────────▼────────────────────────────────────────────────┐   │
│  │              matrix-net (172.20.0.0/24)              │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │   Nginx     │  │  Conduit    │  │   Coturn    │     │   │
│  │  │  (Proxy)    │  │ (Matrix HS) │  │  (TURN/STUN)│     │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │
│  └──────────────────────────────────────────────────────────┘   │
│           │                                               │
│  ┌────────▼────────────────────────────────────────────────┐   │
│  │              bridge-net (172.21.0.0/24) [internal]      │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │   Sygnal    │  │ mautrix-*   │  │   Agents    │     │   │
│  │  │ (Push GW)   │  │  (Bridges)  │  │  (Docker)   │     │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Network Security

- **matrix-net:** Exposed via Nginx (80/443) for public Matrix access
- **bridge-net:** Internal only (no external access) for services
- **Bridge on host:** Unix socket only (no network exposure)

### Docker Compose Files

**File:** `docker-compose.yml` (Meta-composition)
- Includes both `docker-compose.matrix.yml` and `docker-compose.bridge.yml`
- Provides convenient presets for full stack

**Matrix Stack:** `docker-compose.matrix.yml`
- Services: Nginx, Coturn
- Optional profile: `frontend` (Nginx)
- Network: `matrix-net` (172.20.0.0/24)

**Bridge Stack:** `docker-compose.bridge.yml`
- Services: Sygnal, mautrix-* (optional profiles)
- Network: `bridge-net` (172.21.0.0/24, internal)
- External: `matrix-net` (for Matrix API access)

### Optional Bridges

ArmorClaw supports **Matrix-to-Service bridges** via profiles:

| Bridge | Profile | Purpose |
|---------|----------|---------|
| mautrix-slack | `slack` | Slack integration |
| mautrix-discord | `discord` | Discord integration |
| mautrix-telegram | `telegram` | Telegram integration |
| mautrix-whatsapp | `whatsapp` | WhatsApp integration |

**Usage:**
```bash
docker-compose --profile slack up -d
docker-compose --profile discord up -d
```

---

## Installation Script Flow (v4.5.0)

### Bootstrap Flow

1. **Starts the bridge binary** in background
2. **Waits for socket** at `/run/armorclaw/bridge.sock`
3. **API keys are read from environment variables** (not stored in keystore)
4. **Claims OWNER role** for admin user via `provisioning.claim`
5. **Generates QR code** for ArmorChat mobile provisioning

### Stage-0: Bootstrap (install.sh)

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

**Steps:**
1. Download `installer-v5.sh`, SHA256, GPG signature, and public key
2. Normalize line endings to LF (critical for GPG verification)
3. Verify SHA256 checksum
4. Verify GPG signature (A1482657223EAFE1C481B74A8F535F90685749E0)
5. Download bridge binary from GitHub Releases
6. Execute `installer-v5.sh`

### Stage-1: Full Installer (installer-v5.sh)

**Features:**
- **Lockfile protection:** flock to prevent parallel installs
- **Docker daemon readiness:** Dual-check (info + ps) with 20s timeout
- **Persistent logging:** /var/log/armorclaw/install.log
- **Environment passthrough:** All vars passed to setup scripts
- **Docker Compose detection:** Supports both `docker compose` and `docker-compose`
- **Conduit image control:** Via CONDUIT_VERSION environment variable

**Configuration:**
```bash
INSTALL_MODE=quick|matrix  # Choose installation type
DOCKER_COMPOSE           # Auto-detected
CONDUIT_VERSION           # Default: latest
CONDUIT_IMAGE             # Default: matrixconduit/matrix-conduit:$CONDUIT_VERSION
```

### Stage-2: Setup Scripts

**Quick Setup** (`setup-quick.sh`):
- Interactive wizard or non-interactive mode
- AI provider configuration with registry
- Admin creation (zero-touch with shared-secret API)
- Bridge configuration
- Matrix detection and optional Conduit setup

**Matrix Setup** (`setup-matrix.sh`):
- Deploy Matrix/Conduit stack
- Configure Nginx reverse proxy
- Setup Coturn for voice/video
- Generate SSL certificates

---

## What You MUST Do After Setup

### 1. Save Your Credentials (CRITICAL)

The setup displays credentials **once**. Write them down:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Admin Login (Element X / ArmorChat):
  Username:   @admin:192.168.1.100
  Password:   Xy7kL9mN2pQ4rS8t
  Homeserver: http://192.168.1.100:6167

  Password saved to: /var/lib/armorclaw/.admin_password
  ⚠ Delete this file after first login for security.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Recovery if lost:**
```bash
# Password file (if still present)
docker exec armorclaw cat /var/lib/armorclaw/.admin_password

# Or reset via Matrix admin API (requires Conduit admin token)
```

### 2. Connect a Matrix Client

**Option A: Element X (Recommended)**

1. Download Element X: https://element.io/download
2. Open Element X
3. Click "Edit" next to homeserver
4. Enter: `http://YOUR_IP:6167` (or your domain)
5. Click "Sign in"
6. Enter username: `admin` (or what was configured)
7. Enter password from setup
8. You should see the "ArmorClaw Bridge" room

**Option B: ArmorChat (Mobile)**

1. Install ArmorChat on your mobile device
2. Scan the QR code displayed in terminal
3. Or manually enter:
   - Homeserver: `http://YOUR_IP:6167`
   - Username: `admin`
   - Password: (from setup)

### 3. Verify Bridge Connection

In the "ArmorClaw Bridge" room, send:

```
!status
```

You should receive a response like:

```
✓ Bridge is running
✓ Matrix connected
✓ Keystore initialized
✓ 1 API key configured
✓ Agent Studio ready
```

### 4. Create Your First Agent

Using Matrix commands:

```
!agent create "Document Processor"
```

Follow the interactive wizard to:
- Select skills (data_analyze, web_extract)
- Configure PII access (client_name, client_email)
- Set resource tier (medium)

### 5. Test AI Functionality

Send a message to the bridge:

```
Hello, can you help me with something?
```

The bridge should respond using your configured AI provider (from OPENROUTER_API_KEY or ZAI_API_KEY in your environment).

### 6. Delete Password File (Security)

After successful login:

```bash
docker exec armorclaw rm /var/lib/armorclaw/.admin_password
```

---

## Environment Variables Reference

### API Keys (Any one required for Non-Interactive Mode)

| Variable | Provider | Description |
|----------|----------|-------------|
| `OPEN_AI_KEY` | OpenAI | OpenAI API key |
| `OPENROUTER_API_KEY` | OpenRouter | OpenRouter API key (supports many providers) |
| `ZAI_API_KEY` | xAI | xAI API key |

**Note:** Set any of these in your `.zshrc` to enable non-interactive mode. The bridge will automatically use the available keys.

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ARMORCLAW_SERVER_NAME` | Auto | Server domain or IP. Auto-detected on host. |
| `ARMORCLAW_API_BASE_URL` | Provider default | Custom API endpoint (for Anthropic, Zhipu, etc.) |
| `ARMORCLAW_ADMIN_USERNAME` | (generated) | Custom admin username |
| `ARMORCLAW_ADMIN_PASSWORD` | (generated) | Admin password for Matrix |
| `ARMORCLAW_PROFILE` | `quick` | `quick` or `matrix` |
| `CONDUIT_VERSION` | `latest` | Conduit version to install |
| `CONDUIT_IMAGE` | matrixconduit/matrix-conduit:latest | Custom Conduit image |
| `DOCKER_COMPOSE` | (auto-detected) | Docker Compose command |

### Development/Debugging

| Variable | Description |
|----------|-------------|
| `ARMORCLAW_DEBUG` | Enable debug logging |
| `ARMORCLAW_DEV_MODE` | Enable log backup to `/tmp/armorclaw-logs/` |
| `ARMORCLAW_PROVIDERS_URL` | Remote provider registry URL |

---

## Common Post-Deployment Tasks

### Check System Health

```bash
# Container status
docker ps | grep armorclaw

# Bridge health via RPC
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Matrix homeserver health
curl http://localhost:6167/_matrix/client/versions

# View setup log
docker exec armorclaw cat /var/log/armorclaw/setup.log

# Agent Studio stats
echo '{"jsonrpc":"2.0","id":1,"method":"studio.stats"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Add Another API Key

API keys are now stored in environment variables, not in the keystore. To add a new provider:

1. Add the key to your `.zshrc`:
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-xxx
   ```

2. The bridge will automatically use it when available.

3. To verify keys are loaded:
   ```bash
   source ~/.zshrc
   echo $OPENROUTER_API_KEY  # Should show your key
   ```

### Generate New ArmorChat QR

The QR code is **automatically generated** at setup completion. To regenerate:

```bash
docker exec armorclaw armorclaw-bridge generate-qr --host <server-ip> --port 8443
```

### View Logs

```bash
# Setup log
docker exec armorclaw view-setup-log

# Follow setup log
docker exec armorclaw view-setup-log --follow

# Errors only
docker exec armorclaw view-setup-log --errors
```

### List Available MCPs

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_mcps"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Troubleshooting

### Setup Failed

If setup fails, you'll see:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SETUP FAILED (exit code: 1)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Rolling back partial setup...

Log file locations:
  Primary:   /var/log/armorclaw/setup.log
  Backup:    /tmp/armorclaw-logs/setup.log.backup

Last 30 lines of log:
  ...
```

**Steps to recover:**

1. Check the log: `docker cp armorclaw:/var/log/armorclaw/setup.log ./setup.log`
2. Fix the issue (usually Docker socket or network)
3. Remove the container: `docker rm -f armorclaw`
4. Re-run the docker command

### Can't Connect to Matrix

1. **Check port binding:**
   ```bash
   docker port armorclaw
   # Should show: 6167/tcp -> 0.0.0.0:6167
   ```

2. **Check firewall:**
   ```bash
   sudo ufw status
   sudo ufw allow 6167/tcp
   ```

3. **Check from inside container:**
   ```bash
   docker exec armorclaw curl http://localhost:6167/_matrix/client/versions
   ```

### Bridge Not Responding

1. **Check bridge process:**
   ```bash
   docker exec armorclaw ps aux | grep armorclaw-bridge
   ```

2. **Check socket:**
   ```bash
   docker exec armorclaw ls -la /run/armorclaw/bridge.sock
   ```

3. **Check RPC:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
    docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

### Agent Studio Issues

1. **Check studio database:**
   ```bash
   docker exec armorclaw ls -la /var/lib/armorclaw/studio.db
   ```

2. **List skills:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_skills"}' | \
    docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

3. **Check agent instances:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_instances"}' | \
    docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

---

## Architecture Reference

### Container Layout

```
armorclaw container
├── /opt/armorclaw/
│   ├── armorclaw-bridge     # Go binary
│   ├── quickstart.sh        # Entry point
│   ├── container-setup.sh   # Setup wizard
│   ├── agent/               # Agent runtime
│   └── configs/             # Config templates
├── /etc/armorclaw/
│   ├── config.toml          # Main config
│   ├── ssl/                 # SSL certificates
│   └── .setup_complete      # Setup flag
├── /var/lib/armorclaw/
│   ├── keystore.db          # SQLCipher encrypted (for other secrets)
│   ├── studio.db            # Agent Studio database
│   ├── .admin_user          # Admin info for OWNER claim
│   └── .admin_password      # Temp password file
├── /run/armorclaw/
│   └── bridge.sock          # Unix socket for RPC
└── /var/log/armorclaw/
    └── setup.log            # Setup log

# API Keys: Read from environment variables at runtime
# - OPENROUTER_API_KEY → openrouter-default
# - ZAI_API_KEY → xai-default
# - OPEN_AI_KEY → openai-default
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 4.8.0 | 2026-03-12 | **Setup UX:** Cloudflare Tunnel for instant HTTPS, Element X credentials display, Go 1.26.1 upgrade, QR auto-install, conduit.toml generation, homeserver URL fix. |
| 4.7.0 | 2026-03-11 | **Deployment Audit:** Network subnet fixes, test security hardening, STUN/TURN env vars, config placeholder cleanup. |
| 4.6.0 | 2026-03-11 | **GPG Key Rotation:** New signing key, installer signature fix, binary download disabled until release. |
| 4.5.0 | 2026-03-10 | **Provider Registry:** Embedded 12+ providers (Zhipu, Moonshot, DeepSeek, Groq). **Matrix Integration:** Auto-detect and install Conduit. **Catwalk:** Dynamic provider/model discovery with `/ai` commands. |
| 4.4.0 | 2026-03-08 | **Systemd Hardening:** Type=simple, RuntimeDirectory, deterministic startup. **Installer:** Lockfile, Docker readiness, persistent logging, GPG verification v1.4.3. |
| 4.3.0 | 2026-03-06 | **Catwalk AI:** Dynamic provider discovery, runtime switching via Matrix commands, Catwalk v0.28.3 integration. |
| 4.2.0 | 2026-03-07 | **Matrix Event Bus:** Zero-allocation receive path, slow consumer detection, context cancellation. |
| 4.1.0 | 2026-02-28 | **Browser Service:** TypeScript/Playwright automation, Browser Client (Go), Queue Processor. |
| 4.0.0 | 2026-02-28 | **Agent Studio:** Skills registry, PII registry, Resource profiles, MCP approval workflow. |

---

## Recent Reviews

### 2026-03-12 - Setup UX & Connectivity

**Focus Areas:**
- Element X connectivity (requires HTTPS)
- Cloudflare Tunnel integration
- QR code provisioning experience
- Setup wizard UX improvements

**Improvements Added:**
1. Cloudflare Tunnel for instant HTTPS (NEW)
2. Element X credentials at install completion (NEW)
3. Go 1.26.1 upgrade with 1.22+ minimum enforcement
4. Auto-install qrencode for terminal QR display
5. conduit.toml config file generation
6. Homeserver URL format correction

**Component Status:**
| Component | Status | Notes |
|-----------|--------|-------|
| install.sh | ✅ Production | v1.4.3, GPG verified |
| installer-v5.sh | ✅ Production | v5.0.0, lockfile protected |
| setup-quick.sh | ✅ Production | Cloudflare Tunnel, QR display |
| QR generation | ✅ Working | Dual flow (config + provisioning) |
| Matrix/Conduit | ✅ Working | Auto-install, config generation |
| Go TUI Wizard | ✅ Working | Non-interactive mode supported |

### 2026-03-11 - Deployment Audit

**Audited Areas:**
- Docker compose network subnets
- Conduit volume mount paths
- Matrix/Conduit configuration consistency
- Browser-service integration
- Bridge startup assumptions

**Issues Found & Fixed:**
1. Network subnet overlaps (CRITICAL) - Narrowed all /16 to /24
2. Test config open registration (HIGH) - Disabled, added shared secret
3. Missing keystore volume (HIGH) - Added bridge_keystore to full stack
4. Missing health check condition (HIGH) - Added condition: service_healthy
5. Hardcoded TURN secrets (MEDIUM) - Replaced with empty values
6. Hardcoded STUN server (MEDIUM) - Made configurable via env var
7. Auto-generated secrets without warning (MEDIUM) - Added startup warnings
8. Fake URLs in config.example.toml (MEDIUM) - Replaced with empty + guidance

### 2026-03-08 - Installer Hardening Review

**Completed Work:**

**Installer Hardening (Phase 6)**

1. **Docker Daemon Readiness Checks**
   - wait_for_docker() function in all installers
   - Dual-check: docker info && docker ps
   - 20-second timeout with 2-second intervals

2. **Installer Lockfile**
   - Uses flock for parallel install prevention
   - EXIT trap with flock -u 2>/dev/null || true

3. **Persistent Logging**
   - /var/log/armorclaw/install.log
   - Fallback to /tmp/armorclaw if /var/log unavailable

4. **Environment Variable Passthrough**
   - DOCKER_COMPOSE, CONDUIT_VERSION, CONDUIT_IMAGE exported
   - env -S bash for proper inheritance

5. **Docker Compose Detection**
   - Detects both 'docker compose' and 'docker-compose'
   - Fallback mechanism in sub-installers
