# ArmorClaw Renaming Plan

> **Created:** 2026-02-08
> **Status:** Planning - Approval Required
> **Scope:** Rename "ArmorClaw" to "ArmorClaw" across entire codebase
> **Estimated Effort:** 4-6 hours with parallel agent team

---

## Executive Summary

**Objective:** Rename all references from "ArmorClaw" to "ArmorClaw" across the entire codebase, including code, documentation, configuration files, and deployment scripts.

**Scope:** ~2,831 occurrences across 280+ files:
- "armorclaw" (lowercase): 753 occurrences in 125 files
- "ArmorClaw" (PascalCase): 172 occurrences in 41 files
- "ARMORCLAW" (uppercase): 1,906 occurrences in 114 files

**Strategy:** Parallel execution with agent team divided by component domain, with strict dependency ordering to avoid merge conflicts.

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Merge conflicts** | High | Sequential phases, clear ownership |
| **Missed occurrences** | Medium | Verification phase, grep validation |
| **Broken references** | High | Link checking, test suite |
| **Git history confusion** | Low | Use `git mv` for files, preserve history |
| **External dependencies** | Medium | Update URLs, Docker Hub, GitHub |

---

## Critical Gap Analysis (Identified & Fixed)

### 🔴 CRITICAL Gaps (Missing from Original Plan)

1. **Go Module Path** ❌ MISSING - CRITICAL
   - **File:** `bridge/go.mod`
   - **Current:** `module github.com/armorclaw/bridge`
   - **Required:** `module github.com/armorclaw/bridge`
   - **Impact:** Breaks all Go imports if not updated
   - **Assigned To:** Agent 1 (Phase 1)

2. **Docker Compose Container Names** ❌ MISSING
   - **Files:** `docker-compose-stack.yml`, `docker-compose-full.yml`, `docker-compose.yml`
   - **Current Names:** `armorclaw-matrix`, `armorclaw-caddy`, `armorclaw-bridge`, `armorclaw-provision`
   - **Required:** `armorclaw-matrix`, `armorclaw-caddy`, `armorclaw-bridge`, `armorclaw-provision`
   - **Impact:** Container naming conflicts, confusion
   - **Assigned To:** Agent 1 (Phase 1)

3. **Docker Network Names** ❌ MISSING
   - **Files:** `docker-compose-stack.yml`, `docker-compose-full.yml`, `docker-compose.yml`
   - **Current:** `armorclaw-net`
   - **Required:** `armorclaw-net`
   - **Impact:** Network isolation issues
   - **Assigned To:** Agent 1 (Phase 1)

4. **Docker Volume Paths** ❌ MISSING
   - **Files:** `docker-compose-stack.yml`, `docker-compose-full.yml`, `docker-compose.yml`
   - **Current:** `/etc/armorclaw/`, `/var/lib/armorclaw/`
   - **Required:** `/etc/armorclaw/`, `/var/lib/armorclaw/`
   - **Impact:** Data persistence issues
   - **Assigned To:** Agent 1 (Phase 1)

5. **LICENSE File Copyright** ❌ MISSING
   - **File:** `LICENSE`
   - **Current:** "Copyright (c) 2026 ArmorClaw Contributors"
   - **Required:** "Copyright (c) 2026 ArmorClaw Contributors"
   - **Impact:** Legal/copyright issues
   - **Assigned To:** Agent 2 (Phase 2 - Documentation)

### 🟠 HIGH PRIORITY Gaps

6. **ArmorClaw Evolution Evolution Branding** ⚠️ STRATEGIC DECISION NEEDED
   - **Files:** `docs/plans/2026-02-07-armorclaw-evolution-design.md`, `docs/PROGRESS/progress.md`, `docs/index.md`
   - **Current:** Multi-agent platform called "ArmorClaw Evolution"
   - **Decision Required:**
     - Option A: Rename to "ArmorClaw Evolution" (consistent branding)
     - Option B: Keep as "ArmorClaw Evolution" (distinct brand for multi-agent platform)
     - Option C: Rename to "SwarmArmorClaw" (hybrid approach)
   - **Recommendation:** Option A - Use "ArmorClaw Evolution" for consistency
   - **Assigned To:** Agent 2 (Phase 2 - Documentation)

7. **GitHub URL References** ⚠️ PARTIALLY COVERED
   - **Files:** 39 files with `github.com/armorclaw/` URLs
   - **Current:** `github.com/armorclaw/armorclaw`, `github.com/armorclaw/bridge`
   - **Required:** `github.com/armorclaw/armorclaw`, `github.com/armorclaw/bridge`
   - **Impact:** Broken links, incorrect clone URLs
   - **Assigned To:** Agent 2 (Phase 2 - Documentation)

8. **README Badge URLs** ❌ MISSING
   - **File:** `README.md`
   - **Impact:** Badge links will break after GitHub rename
   - **Assigned To:** Agent 2 (Phase 2 - Documentation)

### 🟡 MEDIUM PRIORITY Gaps

9. **Batch File** (Windows)
   - **File:** `tests/test-hardening.bat`
   - **Assigned To:** Agent 4 (Phase 4 - Testing)

10. **Python Module Imports**
    - **Files:** `container/openclaw/` directory
    - **Note:** "openclaw" is the upstream agent name, not ArmorClaw
    - **Action:** Keep "openclaw" unchanged (it's the correct upstream name)
    - **Assigned To:** N/A - No change needed

11. **Process Names in Scripts**
    - Check scripts for process name references (pgrep, pkill, etc.)
    - **Assigned To:** Agent 3 (Phase 3 - DevOps)

12. **Version Output Strings**
    - Go code version output may include "ArmorClaw"
    - **Assigned To:** Agent 1 (Phase 1)

13. **Help Text Strings**
    - CLI help text references
    - **Assigned To:** Agent 1 (Phase 1)

### 🟢 LOW PRIORITY Gaps

14. **Documentation File Names**
    - Some files may have "armorclaw" in filename (only README-ArmorClaw.md identified)
    - **Assigned To:** Agent 2 (Phase 2)

15. **Internal Comments**
    - Code comments mentioning "ArmorClaw"
    - **Assigned To:** Respective agents per phase

---

## Execution Strategy: 4 Phases (UPDATED WITH GAPS)

### Phase 1: Critical Infrastructure (Foundation)
**Order:** 1st ( blockers for all other phases)
**Team:** Agent 1 (Infrastructure Specialist)
**Files:** ~15 files, ~200 occurrences

**Responsibilities:**
1. **Go package naming** (bridge/)
   - `bridge/cmd/bridge/main.go`
   - `bridge/pkg/*/` (all packages)
   - Update internal package references

2. **Docker and container**
   - `Dockerfile`
   - `bridge/Dockerfile`
   - `docker-compose*.yml`
   - Container entrypoint scripts

3. **Core configuration**
   - `bridge/config.example.toml`
   - `.env.example`
   - Environment variable prefixes

**Deliverables:**
- All Go code compiles with new name
- Docker builds successfully
- Container images build and run

**Verification:**
```bash
cd bridge && go build -o build/armorclaw-bridge ./cmd/bridge
docker build -t armorclaw/agent:v1 .
docker-compose -f docker-compose-stack.yml config
```

---

### Phase 2: Documentation (User-Facing)
**Order:** 2nd (independent of code, but visible to users)
**Team:** Agent 2 (Documentation Specialist)
**Files:** ~80 files, ~1,500 occurrences

**Responsibilities:**

1. **Core documentation** (highest priority)
   - `README.md`
   - `CLAUDE.md`
   - `CONTRIBUTING.md`
   - `LICENSE.md`
   - `docs/index.md`
   - `docs/status/*.md`
   - `docs/PROGRESS/*.md`

2. **User guides** (alphabetical order)
   - `docs/guides/setup-guide.md`
   - `docs/guides/configuration.md`
   - `docs/guides/troubleshooting.md`
   - `docs/guides/error-catalog.md`
   - `docs/guides/element-x-*.md`
   - `docs/guides/hostinger-*.md`
   - `docs/guides/*-deployment.md` (all 11 deployment guides)

3. **Technical documentation**
   - `docs/reference/*.md`
   - `docs/plans/*.md` (architecture and design docs)
   - `docs/output/*.md` (analysis and review docs)

**Special Considerations:**
- Preserve all markdown formatting, code blocks, and links
- Update internal cross-references
- Update external links (GitHub, Docker Hub)
- Update version numbers if they contain the name

**Deliverables:**
- All markdown files updated
- No broken internal links
- All external URLs updated

**Verification:**
```bash
# Check for remaining ArmorClaw references in docs
grep -r "ArmorClaw" docs/ --exclude-dir=.git
grep -r "armorclaw" docs/ --exclude-dir=.git

# Validate markdown links (if tool available)
markdown-link-check docs/*.md
```

---

### Phase 3: Deployment & Scripts (Operational)
**Order:** 3rd (depends on Phase 1 infrastructure)
**Team:** Agent 3 (DevOps Specialist)
**Files:** ~40 files, ~400 occurrences

**Responsibilities:**

1. **Deployment scripts** (deploy/)
   - `deploy/setup-wizard.sh`
   - `deploy/vps-deploy.sh`
   - `deploy/install-bridge.sh`
   - `deploy/start-local-matrix.sh`
   - `deploy/launch-element-x.sh`
   - `deploy/deploy-all.sh`
   - `deploy/deploy-infra.sh`
   - All other deploy/*.sh scripts

2. **Infrastructure scripts** (scripts/)
   - `scripts/build-bridge-binaries.sh`
   - `scripts/prepare-release.sh`
   - `scripts/init-bridge-repo.sh`
   - `scripts/validate-infrastructure.sh`
   - All other scripts/*.sh scripts

3. **Configuration files**
   - `configs/nginx.conf`
   - `configs/conduit.toml`
   - `configs/turnserver.conf`

4. **CI/CD**
   - `.github/workflows/*.yml`
   - `Makefile`

**Special Considerations:**
- Update all socket paths: `/run/armorclaw/` → `/run/armorclaw/`
- Update all service names: `armorclaw-bridge` → `armorclaw-bridge`
- Update Docker image names: `armorclaw/agent` → `armorclaw/agent`
- Update environment variable prefixes: `ARMORCLAW_` → `ARMORCLAW_`
- Preserve script functionality and error handling

**Deliverables:**
- All shell scripts functional
- All deployment paths updated
- CI/CD pipelines updated

**Verification:**
```bash
# Test deployment script syntax
bash -n deploy/*.sh
bash -n scripts/*.sh

# Verify socket paths
grep -r "/run/armorclaw" deploy/ scripts/

# Verify environment variables
grep -r "ARMORCLAW_" deploy/ scripts/ .github/
```

---

### Phase 4: Testing & Verification (QA)
**Order:** 4th (depends on all previous phases)
**Team:** Agent 4 (QA Specialist)
**Files:** ~20 files, ~100 occurrences

**Responsibilities:**

1. **Test scripts** (tests/)
   - `tests/test-hardening.{sh,ps1,bat}`
   - `tests/test-secrets.{sh,ps1}`
   - `tests/test-exploits.{sh,ps1}`
   - `tests/test-e2e.{sh,ps1}`
   - `tests/test-attach-config.sh`
   - `tests/test-secret-passing.sh`

2. **Release artifacts**
   - `release/MANIFEST-*.txt`
   - `release/SHA256SUMS-*`
   - `deploy/SHA256SUMS*`

3. **Build system**
   - `Makefile`
   - `bridge/Makefile`
   - `.gitignore`

4. **Final verification**
   - Grep entire codebase for missed references
   - Validate all file renames completed
   - Check for hardcoded URLs
   - Verify Docker Hub references

**Deliverables:**
- All test scripts updated
- Build system functional
- Complete verification report

**Verification Commands:**
```bash
# Comprehensive grep for all variants
grep -ri "armorclaw" . --exclude-dir=.git --exclude-dir=node_modules
grep -ri "ArmorClaw" . --exclude-dir=.git --exclude-dir=node_modules
grep -ri "ARMORCLAW" . --exclude-dir=.git --exclude-dir=node_modules

# Verify no broken references in Go code
cd bridge && go mod tidy
cd bridge && go build ./...

# Verify Docker compose files
docker-compose -f docker-compose.yml config
docker-compose -f docker-compose-stack.yml config
docker-compose -f docker-compose-full.yml config
```

---

## File Renames Required

The following files/directories must be renamed using `git mv`:

### 📄 Documentation Files
- `docs/output/README-ArmorClaw.md` → `docs/output/README-ArmorClaw.md`

### 🐳 Docker Compose Service Names (Inline, Not File Renames)
- Container names in `docker-compose-stack.yml`:
  - `armorclaw-matrix` → `armorclaw-matrix`
  - `armorclaw-caddy` → `armorclaw-caddy`
  - `armorclaw-bridge` → `armorclaw-bridge`
  - `armorclaw-provision` → `armorclaw-provision`
- Network name: `armorclaw-net` → `armorclaw-net`

### 📦 Go Module Path (Not File Rename, Module Rename)
- **Current:** `module github.com/armorclaw/bridge`
- **Required:** `module github.com/armorclaw/bridge`
- **Method:** `go mod edit -module=github.com/armorclaw/bridge` then `go mod tidy`

### Scripts/Binaries
- Binary output: `armorclaw-bridge` → `armorclaw-bridge`
  (Update all references in build scripts)

### Docker Images
- `armorclaw/agent:v1` → `armorclaw/agent:v1`
  (Update all Dockerfile references)
  (Update all docker-compose files)

### Unix Sockets
- `/run/armorclaw/` → `/run/armorclaw/`
  (Update all Go code)
  (Update all scripts)
  (Update all documentation)

### Environment Variables
- `ARMORCLAW_*` → `ARMORCLAW_*`
  (Update all configuration)
  (Update all scripts)
  (Update all documentation)

---

## Naming Convention Changes

| Context | Old Name | New Name |
|---------|----------|----------|
| **Project** | ArmorClaw | ArmorClaw |
| **lowercase** | armorclaw | armorclaw |
| **uppercase** | ARMORCLAW | ARMORCLAW |
| **Binary** | armorclaw-bridge | armorclaw-bridge |
| **Docker image** | armorclaw/agent | armorclaw/agent |
| **Socket path** | /run/armorclaw/ | /run/armorclaw/ |
| **Config dir** | ~/.armorclaw/ | ~/.armorclaw/ |
| **Env prefix** | ARMORCLAW_ | ARMORCLAW_ |
| **GitHub repo** | github.com/armorclaw/ | github.com/armorclaw/ |
| **Docker Hub** | docker.io/armorclaw | docker.io/armorclaw |

---

## Detailed Agent Assignments

### Agent 1: Infrastructure Specialist (Phase 1)
**Priority:** CRITICAL - Blocks all other phases

**Files to Modify (20 files, ~250 occurrences):**

1. **🔴 CRITICAL: Go Module Path**
   - `bridge/go.mod` (1 occurrence) - **LINE 1: `module github.com/armorclaw/bridge`**
   - **Change To:** `module github.com/armorclaw/bridge`
   - **Impact Required:** `go mod tidy` after change to update dependencies

2. **Go Source Code**
   - `bridge/cmd/bridge/main.go` (91 occurrences)
   - `bridge/pkg/config/config.go` (10 occurrences)
   - `bridge/pkg/config/config_test.go` (2 occurrences)
   - `bridge/pkg/config/loader.go` (1 occurrence)
   - `bridge/pkg/rpc/server.go` (18 occurrences)
   - `bridge/pkg/logger/logger.go` (1 occurrence)
   - `bridge/pkg/logger/security.go` (1 occurrence)
   - `bridge/pkg/docker/client.go` (4 occurrences)
   - `bridge/pkg/keystore/keystore.go` (1 occurrence)
   - `bridge/pkg/budget/tracker.go` (4 occurrences)
   - `bridge/pkg/budget/tracker_test.go` (1 occurrence)
   - `bridge/pkg/logger/security_test.go` (4 occurrences)
   - `bridge/internal/adapter/matrix.go` (2 occurrences)
   - `bridge/internal/adapter/matrix_zero_trust_test.go` (1 occurrence)

3. **🔴 CRITICAL: Docker Compose Files**
   - `docker-compose-stack.yml`
     - Container names: `armorclaw-matrix`, `armorclaw-caddy`, `armorclaw-bridge`, `armorclaw-provision`
     - Network: `armorclaw-net`
     - Volume paths: `/etc/armorclaw/`, `/run/armorclaw/`, `/var/lib/armorclaw/`
     - Health check: `pgrep armorclaw-bridge`
   - `docker-compose-full.yml` (similar changes)
   - `docker-compose.yml` (similar changes)

4. **Docker Files**
   - `Dockerfile` (2 occurrences)
   - `bridge/Dockerfile` (9 occurrences)

5. **Container Files**
   - `container/opt/openclaw/entrypoint.sh` (12 occurrences)
   - `container/opt/openclaw/entrypoint.py` (50 occurrences)
   - `container/opt/openclaw/health.sh` (2 occurrences)

6. **Configuration**
   - `bridge/config.example.toml` (8 occurrences)
   - `.env.example` (1 occurrence)

**Key Changes:**
- **Go module:** `github.com/armorclaw/bridge` → `github.com/armorclaw/bridge`
- **Container names:** `armorclaw-*` → `armorclaw-*`
- **Network:** `armorclaw-net` → `armorclaw-net`
- **Volume paths:** `/etc/armorclaw/` → `/etc/armorclaw/`, `/var/lib/armorclaw/` → `/var/lib/armorclaw/`
- **Socket paths:** `/run/armorclaw/` → `/run/armorclaw/`
- **Binary name:** `armorclaw-bridge` → `armorclaw-bridge`
- **Config directory:** `~/.armorclaw/` → `~/.armorclaw/`
- **Service name:** `armorclaw` → `armorclaw`
- **Health check:** `pgrep armorclaw-bridge` → `pgrep armorclaw-bridge`

**Build Verification:**
```bash
cd bridge
# Update module path
go mod edit -module=github.com/armorclaw/bridge
go mod tidy
go build -o build/armorclaw-bridge ./cmd/bridge
./build/armorclaw-bridge --help
```

**Docker Verification:**
```bash
docker build -t armorclaw/agent:v1 .
docker run --rm armorclaw/agent:v1 --help

# Verify docker-compose files
docker-compose -f docker-compose-stack.yml config
docker-compose -f docker-compose-full.yml config
docker-compose -f docker-compose.yml config
```

---

### Agent 2: Documentation Specialist (Phase 2)
**Priority:** HIGH - User-facing, can proceed in parallel with Phase 1

**Files to Modify (~80 files, ~1,500 occurrences):**

**Priority 1: Core Docs (Update First)**
1. `README.md` (32 ArmorClaw + 6 armorclaw + **BADGE URLS**)
2. `CLAUDE.md` (13 ArmorClaw + 6 armorclaw)
3. `CONTRIBUTING.md` (3 ArmorClaw + 3 armorclaw)
4. `LICENSE.md` (1 ArmorClaw) - **🔴 UPDATE: "Copyright (c) 2026 ArmorClaw Contributors"**
5. `LICENSE` (1 ArmorClaw) - **🔴 UPDATE: Same copyright notice**
6. `docs/index.md` (48 ArmorClaw + 13 armorclaw)
7. `docs/status/2026-02-05-status.md` (23 ArmorClaw + 11 armorclaw)
8. `docs/PROGRESS/progress.md` (53 ArmorClaw + 24 armorclaw)

**Priority 2: User Guides (Alphabetical)**
8. `docs/guides/aws-fargate-deployment.md` (49 ArmorClaw + 6 armorclaw)
9. `docs/guides/azure-deployment.md` (32 ArmorClaw + 4 armorclaw)
10. `docs/guides/configuration.md` (36 ArmorClaw + 21 armorclaw)
11. `docs/guides/digitalocean-deployment.md` (6 ArmorClaw + 4 armorclaw)
12. `docs/guides/element-x-configs.md` (10 ArmorClaw + 3 armorclaw)
13. `docs/guides/element-x-quickstart.md` (11 ArmorClaw + 8 armorclaw)
14. `docs/guides/error-catalog.md` (94 ArmorClaw + 8 armorclaw)
15. `docs/guides/flyio-deployment.md` (45 ArmorClaw + 14 armorclaw)
16. `docs/guides/gcp-cloudrun-deployment.md` (89 ArmorClaw + 15 armorclaw)
17. `docs/guides/hostinger-deployment.md` (64 ArmorClaw + 12 armorclaw)
18. `docs/guides/hostinger-docker-deployment.md` (56 ArmorClaw + 19 armorclaw)
19. `docs/guides/hostinger-vps-deployment.md` (64 ArmorClaw + 23 armorclaw)
20. `docs/guides/linode-deployment.md` (17 ArmorClaw + 7 armorclaw)
21. `docs/guides/local-development.md` (10 ArmorClaw + 7 armorclaw)
22. `docs/guides/railway-deployment.md` (6 ArmorClaw + 3 armorclaw)
23. `docs/guides/render-deployment.md` (12 ArmorClaw + 3 armorclaw)
24. `docs/guides/setup-guide.md` (84 ArmorClaw + 11 armorclaw)
25. `docs/guides/troubleshooting.md` (50 ArmorClaw + 3 armorclaw)
26. `docs/guides/vultr-deployment.md` (13 ArmorClaw + 13 armorclaw)

**Priority 3: Technical Docs**
27. `docs/reference/rpc-api.md` (38 ArmorClaw)
28. `docs/plans/2026-02-05-armorclaw-v1-design.md` (93 ArmorClaw)
29. `docs/plans/2026-02-05-communication-server-options.md` (7 ArmorClaw)
30. `docs/plans/2026-02-05-local-bridge-matrix-gateway.md` (13 ArmorClaw)
31. `docs/plans/2026-02-05-minimal-bridge-spec.md` (40 ArmorClaw)
32. `docs/plans/2026-02-05-robust-bridge-spec.md` (33 ArmorClaw)
33. `docs/plans/2026-02-05-phase1-implementation-tasks.md` (5 ArmorClaw)
34. `docs/plans/2026-02-05-license-server-api.md` (2 ArmorClaw)
35. `docs/plans/2026-02-05-business-model-architecture.md` (10 ArmorClaw)
36. `docs/plans/2026-02-07-armorclaw-evolution-design.md` (17 ArmorClaw)
37. `docs/plans/2026-02-07-security-enhancements.md` (5 ArmorClaw)
38. `docs/plans/2026-02-08-android-app-plan.md` (1 ArmorClaw)

**Priority 4: Analysis & Output**
39. `docs/output/review.md` (17 ArmorClaw + 5 armorclaw)
40. `docs/output/README-ArmorClaw.md` (4 ArmorClaw + 9 armorclaw) - **RENAME FILE**
41. `docs/output/startup-config-analysis.md` (9 ArmorClaw + 7 armorclaw)
42. `docs/output/startup-ux-review.md` (35 ArmorClaw + 13 armorclaw)
43. `docs/output/critical-fixes-summary.md` (9 ArmorClaw + 4 armorclaw)
44. `docs/output/setup-flow-security-analysis.md` (6 ArmorClaw + 1 armorclaw)
45. `docs/output/ux-assessment-2026-02-07.md` (21 ArmorClaw + 47 armorclaw)
46. `docs/output/ux-analysis-element-x-flow.md` (4 ArmorClaw + 2 armorclaw)
47. `docs/output/cloudflare-workers-analysis.md` (4 ArmorClaw + 21 armorclaw)
48. `docs/output/hosting-providers-comparison.md` (29 ArmorClaw + 34 armorclaw)

**Special Instructions:**
- **🔴 GitHub URLs (39 files):** Update all `github.com/armorclaw/` → `github.com/armorclaw/`
  - Includes: `github.com/armorclaw/armorclaw`, `github.com/armorclaw/bridge`
  - Update clone URLs, import paths, API references
- **🔴 README Badge URLs:** Update badge image URLs and link URLs
- **🔴 LICENSE Files:** Update copyright notice in both `LICENSE` and `LICENSE.md`
- **⚠️ ArmorClaw Evolution Branding Decision Required:**
  - **Files:** `docs/plans/2026-02-07-armorclaw-evolution-design.md`, `docs/index.md`, `docs/PROGRESS/progress.md`
  - **Strategic Question:** Should multi-agent platform be renamed?
    - Option A (Recommended): "ArmorClaw Evolution" → "ArmorClaw Evolution" (consistent branding)
    - Option B: Keep as "ArmorClaw Evolution" (distinct brand for multi-agent)
    - Option C: "ArmorClaw Evolution" → "SwarmArmorClaw" (hybrid)
  - **Default Action:** If no decision, rename to "ArmorClaw Evolution" for consistency
- Update all Docker Hub references: `armorclaw/agent` → `armorclaw/agent`
- Update all socket paths in examples: `/run/armorclaw/` → `/run/armorclaw/`
- Update all config paths: `~/.armorclaw/` → `~/.armorclaw/`
- Update all binary names: `armorclaw-bridge` → `armorclaw-bridge`
- **🔴 Rename file:** `docs/output/README-ArmorClaw.md` → `docs/output/README-ArmorClaw.md` (use `git mv`)

**Verification:**
```bash
# Check for remaining references
grep -r "ArmorClaw" docs/ | grep -v ".git"
grep -r "armorclaw" docs/ | grep -v ".git"

# Verify all critical docs updated
grep -c "ArmorClaw" README.md CLAUDE.md docs/index.md
```

---

### Agent 3: DevOps Specialist (Phase 3)
**Priority:** HIGH - Operational scripts, depends on Phase 1

**Files to Modify (~40 files, ~400 occurrences):**

**Deployment Scripts (deploy/):**
1. `deploy/setup-wizard.sh` (57 ARMORCLAW + 13 armorclaw)
2. `deploy/vps-deploy.sh` (12 ARMORCLAW + 10 armorclaw)
3. `deploy/install-bridge.sh` (46 ARMORCLAW + 2 armorclaw)
4. `deploy/start-local-matrix.sh` (5 ARMORCLAW + 5 armorclaw)
5. `deploy/launch-element-x.sh` (3 ARMORCLAW + 3 armorclaw)
6. `deploy/deploy-all.sh` (2 ARMORCLAW + 2 armorclaw)
7. `deploy/deploy-infra.sh` (2 ARMORCLAW + 2 armorclaw)
8. `deploy/harden-ssh.sh` (3 ARMORCLAW)
9. `deploy/setup-firewall.sh` (2 ARMORCLAW)
10. `deploy/verify-bridge.sh` (19 ARMORCLAW)
11. `deploy/verify-checksum.sh` (3 ARMORCLAW)
12. `deploy/build-and-push.sh` (4 ARMORCLAW)
13. `deploy/SHA256SUMS` (1 ARMORCLAW)
14. `deploy/SHA256SUMS.example` (3 ARMORCLAW)

**Infrastructure Scripts (scripts/):**
15. `scripts/build-bridge-binaries.sh` (3 ARMORCLAW + 1 armorclaw)
16. `scripts/prepare-release.sh` (24 ARMORCLAW + 2 armorclaw)
17. `scripts/init-bridge-repo.sh` (16 ARMORCLAW + 7 armorclaw)
18. `scripts/validate-infrastructure.sh` (3 ARMORCLAW + 2 armorclaw)
19. `scripts/deploy-infrastructure.sh` (3 ARMORCLAW + 2 armorclaw)

**Configuration Files:**
20. `configs/nginx.conf` (5 ARMORCLAW)
21. `configs/conduit.toml` (3 ARMORCLAW)
22. `configs/turnserver.conf` (3 ARMORCLAW)

**CI/CD:**
23. `.github/workflows/test.yml` (8 ARMORCLAW)
24. `.github/workflows/build-release.yml` (7 ARMORCLAW)

**Build System:**
25. `Makefile` (18 ARMORCLAW)
26. `bridge/Makefile` (1 ARMORCLAW)
27. `.gitignore` (4 ARMORCLAW)

**Shell Completion:**
28. `bridge/completions/bash` (12 ARMORCLAW)
29. `bridge/completions/zsh` (12 ARMORCLAW)

**Test Scripts (PowerShell):**
30. `tests/test-hardening.ps1` (10 ARMORCLAW)
31. `tests/test-secrets.ps1` (2 ARMORCLAW)
32. `tests/test-e2e.ps1` (10 ARMORCLAW)
33. `tests/test-exploits.ps1` (1 ARMORCLAW)

**Key Changes:**
- Environment variables: `ARMORCLAW_` → `ARMORCLAW_`
- Socket paths: `/run/armorclaw/` → `/run/armorclaw/`
- Binary names: `armorclaw-bridge` → `armorclaw-bridge`
- Docker images: `armorclaw/agent` → `armorclaw/agent`
- Service names: `armorclaw-bridge` → `armorclaw-bridge`
- Config directory: `~/.armorclaw/` → `~/.armorclaw/`

**Verification:**
```bash
# Syntax check all bash scripts
for f in deploy/*.sh scripts/*.sh; do bash -n "$f"; done

# Check for remaining references
grep -r "ARMORCLAW" deploy/ scripts/ .github/ | grep -v ".git"
grep -r "armorclaw" deploy/ scripts/ .github/ | grep -v ".git"

# Verify socket paths
grep -r "/run/armorclaw" . | grep -v ".git"
```

---

### Agent 4: QA Specialist (Phase 4)
**Priority:** CRITICAL - Final verification, depends on all phases

**Files to Modify (~20 files, ~100 occurrences):**

**Test Scripts (Bash):**
1. `tests/test-hardening.sh` (10 ARMORCLAW)
2. `tests/test-secrets.sh` (2 ARMORCLAW)
3. `tests/test-exploits.sh` (1 ARMORCLAW)
4. `tests/test-e2e.sh` (10 ARMORCLAW)
5. `tests/test-attach-config.sh` (8 ARMORCLAW)
6. `tests/test-secret-passing.sh` (7 ARMORCLAW)

**🟡 Test Scripts (Windows Batch):**
7. `tests/test-hardening.bat` (1 occurrence)
   - Update batch file references
   - Test on Windows if possible

**Release Artifacts:**
7. `release/MANIFEST-v1.0.0.txt` (7 ARMORCLAW)
8. `release/SHA256SUMS-release.txt` (3 ARMORCLAW)

**Root Files:**
9. `deploy.sh` (7 ARMORCLAW)
10. `docker-compose.yml` (8 ARMORCLAW)
11. `docker-compose-full.yml` (15 ARMORCLAW)
12. `docker-compose-stack.yml` (16 ARMORCLAW)
13. `bridge/go.mod` (1 ARMORCLAW)
14. `bridge/test-client/test-rpc.go` (2 ARMORCLAW)

**Verification Responsibilities:**

1. **Complete Grep Audit**
   ```bash
   # Find ALL remaining references (should be 0 after completion)
   echo "=== Checking for ARMORCLAW ==="
   grep -r "ARMORCLAW" . --exclude-dir=.git --exclude-dir=node_modules 2>/dev/null | wc -l

   echo "=== Checking for ArmorClaw ==="
   grep -r "ArmorClaw" . --exclude-dir=.git --exclude-dir=node_modules 2>/dev/null | wc -l

   echo "=== Checking for armorclaw ==="
   grep -r "armorclaw" . --exclude-dir=.git --exclude-dir=node_modules 2>/dev/null | wc -l
   ```

2. **Build Verification**
   ```bash
   # Verify Go build
   cd bridge
   go mod tidy
   go build -o build/armorclaw-bridge ./cmd/bridge
   ./build/armorclaw-bridge version
   ```

3. **Docker Verification**
   ```bash
   # Verify Docker build
   docker build -t armorclaw/agent:v1 .
   docker images | grep armorclaw

   # Verify docker-compose files
   docker-compose -f docker-compose.yml config
   docker-compose -f docker-compose-stack.yml config
   docker-compose -f docker-compose-full.yml config
   ```

4. **Test Suite Verification**
   ```bash
   # Verify test scripts run
   make test-hardening
   make test-secrets
   ```

5. **Documentation Link Check**
   ```bash
   # Check for broken internal links
   # (requires markdown-link-check or similar tool)
   ```

**Deliverables:**
- Complete audit report with counts
- List of any remaining references (should be 0)
- Build verification results
- Test suite results
- Final approval for merge

---

## Timeline & Coordination

### Parallel Execution Strategy

**Wave 1 (Parallel - 30 min):**
- Agent 1: Phase 1 (Infrastructure) - CRITICAL PATH
- Agent 2: Phase 2 (Documentation) - Can start immediately

**Wave 2 (After Phase 1 complete - 20 min):**
- Agent 3: Phase 3 (DevOps/Scripts)
- Agent 4: Phase 4 (QA/Verification)

**Estimated Total Time:** 2-3 hours with 4 agents working in parallel

### Communication Protocol

1. **Start Signal:** All agents wait for "START" signal
2. **Phase Completion:** Each agent reports completion when done
3. **Blocking Issues:** Immediately report if blocked by another phase
4. **Verification:** Agent 4 provides final sign-off

### Merge Strategy

1. **Feature Branches:** Each agent works on own branch
   - `agent-1-infrastructure-rename`
   - `agent-2-documentation-rename`
   - `agent-3-devops-rename`
   - `agent-4-qa-verification`

2. **Merge Order:**
   ```bash
   # Merge in dependency order
   git merge agent-1-infrastructure-rename  # First
   git merge agent-2-documentation-rename   # Second (no conflicts)
   git merge agent-3-devops-rename          # Third (depends on Phase 1)
   # Agent 4 does verification, no merge needed
   ```

3. **Final Branch:** `rename-armorclaw` (integration branch)

---

## Verification Checklist

### Pre-Merge Verification (Agent 4)

- [ ] Zero occurrences of "ARMORCLAW" in code
- [ ] Zero occurrences of "ArmorClaw" in code (except in this plan)
- [ ] Zero occurrences of "armorclaw" in code (except in this plan)
- [ ] Go code compiles: `go build ./...`
- [ ] Docker image builds: `docker build -t armorclaw/agent:v1 .`
- [ ] All docker-compose files validate
- [ ] All bash scripts pass syntax check
- [ ] Test scripts reference correct names
- [ ] Documentation has no broken links
- [ ] All socket paths updated to `/run/armorclaw/`
- [ ] All env vars updated to `ARMORCLAW_`
- [ ] Binary name updated to `armorclaw-bridge`
- [ ] Docker image updated to `armorclaw/agent:v1`

### Post-Merge Verification

- [ ] Git history preserved (used `git mv` for files)
- [ ] All CI/CD pipelines pass
- [ ] Release artifacts updated
- [ ] Documentation published/updated
- [ ] GitHub organization updated (if applicable)
- [ ] Docker Hub updated (if applicable)

---

## Rollback Plan

If issues arise during renaming:

1. **Immediate Rollback:** Revert to commit before rename
   ```bash
   git revert <merge-commit>
   ```

2. **Partial Rollback:** Revert specific agent's branch
   ```bash
   git revert <agent-branch-commit>
   ```

3. **Documentation Rollback:** Can be done independently

4. **Critical Issues:** Hotfix branch with targeted fixes

---

## External Dependencies to Update

After codebase renaming is complete:

1. **GitHub Organization**
   - Create `armorclaw` organization
   - Rename repository: `armorclaw/armorclaw` → `armorclaw/armorclaw`
   - Update all repository settings
   - Set up redirects from old URLs

2. **Docker Hub**
   - Create `armorclaw` Docker Hub organization
   - Rename repository: `armorclaw/agent` → `armorclaw/agent`
   - Update all image tags
   - Set up pull mirroring

3. **Domain Names** (if applicable)
   - `armorclaw.com` → `armorclaw.com`
   - Update DNS records
   - Update SSL certificates

4. **Documentation Sites**
   - Update all published documentation URLs
   - Update API documentation references
   - Update marketing materials

5. **Third-Party Integrations**
   - Update any registered OAuth applications
   - Update any API keys with restricted domains
   - Update any webhook URLs

---

## Success Criteria

**Rename is complete when:**
1. ✅ Zero references to old name in code (verified by grep)
2. ✅ All code compiles and builds successfully
3. ✅ All tests pass with new names
4. ✅ All documentation updated and consistent
5. ✅ All deployment scripts functional
6. ✅ Docker images build and run correctly
7. ✅ CI/CD pipelines pass
8. ✅ External dependencies updated (GitHub, Docker Hub)

---

## Strategic Decisions Required

Before execution begins, the following decisions must be made:

### Decision 1: ArmorClaw Evolution Multi-Agent Platform Naming ⚠️

**Context:** The current design document refers to the multi-agent platform evolution as "ArmorClaw Evolution."

**Options:**
| Option | Name | Pros | Cons |
|--------|------|------|------|
| **A** | ArmorClaw Evolution | Consistent branding, clear evolution | Longer name |
| **B** | ArmorClaw Evolution | Distinct brand for multi-agent | Inconsistent with ArmorClaw |
| **C** | SwarmArmorClaw | Hybrid approach | Awkward name |
| **D** | ArmorClaw Multi-Agent | Clear description | Generic |

**Recommendation:** **Option A** - "ArmorClaw Evolution" for brand consistency

**Files Affected:**
- `docs/plans/2026-02-07-armorclaw-evolution-design.md` (title and content)
- `docs/index.md` (references)
- `docs/PROGRESS/progress.md` (references)

**Default Action:** If no decision made before execution, use **Option A**.

---

### Decision 2: GitHub Organization Naming

**Question:** Will GitHub organization be renamed?

**Options:**
- **Yes:** Rename `github.com/armorclaw` → `github.com/armorclaw` (requires GitHub org setup)
- **No:** Keep existing org, only rename repo (simpler)

**Impact:** Affects all GitHub URLs in documentation and code.

**Recommendation:** Plan for rename, but can execute without immediate GitHub change (use redirect).

---

### Decision 3: Docker Hub Organization Naming

**Question:** Will Docker Hub organization be renamed?

**Options:**
- **Yes:** Rename `docker.io/armorclaw` → `docker.io/armorclaw`
- **No:** Keep existing, tag new images to new org

**Recommendation:** Rename for consistency. Plan migration strategy.

---

### Decision 4: Backward Compatibility

**Question:** Should we maintain backward compatibility?

**Options:**
- **No:** Clean break, all references updated (recommended)
- **Yes:** Aliases for old names (complexity trade-off)

**Recommendation:** Clean break - update all references consistently.

---

## Next Steps

1. **🔴 DECISION REQUIRED:** ArmorClaw Evolution naming (Decision 1 above)
2. **Approval Required:** Stakeholder approval to proceed
2. **Agent Assignment:** Assign 4 agents to phases
3. **Branch Creation:** Create feature branches for each agent
4. **Execution:** Begin parallel execution with Wave 1
5. **Integration:** Merge branches in dependency order
6. **Verification:** Agent 4 provides final sign-off
7. **External Updates:** Update GitHub, Docker Hub, domains
8. **Announcement:** Communicate rename to users/stakeholders

---

**Plan Status:** ⏳ Awaiting Approval
**Estimated Completion:** 2-3 hours after approval (with 4 parallel agents)
**Risk Level:** Medium (mitigated by phased approach and verification)

---

## 🔍 Comprehensive File Inventory (All Files Requiring Changes)

### Agent 1: Infrastructure (Phase 1) - Updated
**Go Module:** `bridge/go.mod` (🔴 CRITICAL)
**Docker Compose:** `docker-compose-stack.yml`, `docker-compose-full.yml`, `docker-compose.yml`
**Go Source:** 15 files in bridge/
**Container:** 3 files in container/opt/openclaw/
**Config:** 2 files

### Agent 2: Documentation (Phase 2) - Updated
**Root Files:** `README.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `LICENSE`, `LICENSE.md` (🔴 ADDED LICENSE files)
**Guides:** 26 user guide files
**Plans:** 11 plan/strategy files (🔴 INCLUDING ArmorClaw Evolution design)
**Output:** 8 analysis/review files
**GitHub URLs:** 39 files with `github.com/armorclaw/` URLs (🔴 ADDED)
**Total:** ~85 files

### Agent 3: DevOps (Phase 3) - Updated
**Deploy Scripts:** 14 files in deploy/
**Infrastructure Scripts:** 5 files in scripts/
**Config Files:** 3 files in configs/
**CI/CD:** 2 workflow files
**Build:** 3 files (Makefile, .gitignore, etc.)
**Completion:** 2 completion scripts
**Test Scripts:** 4 PowerShell files (🔴 MOVED from Agent 4)

### Agent 4: QA (Phase 4) - Updated
**Test Scripts:** 6 bash test scripts
**Release Artifacts:** 2 files
**Root Files:** 5 files
**Windows Tests:** 1 batch file (🔴 ADDED)
**Go Module:** `bridge/go.mod` (verification)
**Test Client:** 1 file

---

## ⚠️ DO NOT CHANGE (Important Exclusions)

The following should NOT be changed during renaming:

1. **"openclaw" References** ✅ KEEP AS-IS
   - `container/openclaw/` directory (upstream agent name, correct)
   - Python imports referencing `openclaw`
   - Any references to the OpenClaw project (upstream dependency)

2. **Third-Party References** ✅ KEEP AS-IS
   - External URLs not under our control
   - Third-party documentation mentions
   - Quotes or citations mentioning ArmorClaw in external context

3. **Historical Context** ✅ KEEP AS-IS
   - Git commit messages (preserve history)
   - Changelogs mentioning "ArmorClaw" (historical record)
   - Release notes for past versions

4. **Comments Explaining Migration** ✅ KEEP AS-IS
   - Comments like "formerly ArmorClaw" are acceptable for transition period

5. **This Plan Document** ✅ KEEP AS-IS
   - This renaming plan document keeps "ArmorClaw" for clarity
   - After rename is complete, can be archived or updated

---

## 🔧 Go Module Migration Guide (Agent 1 Special Instructions)

### Step-by-Step Go Module Rename

```bash
# 1. Navigate to bridge directory
cd bridge

# 2. Backup go.mod and go.sum
cp go.mod go.mod.backup
cp go.sum go.sum.backup

# 3. Edit module path (automatic)
go mod edit -module=github.com/armorclaw/bridge

# 4. Update all imports and dependencies
go mod tidy

# 5. Verify build
go build -o build/armorclaw-bridge ./cmd/bridge

# 6. Test basic functionality
./build/armorclaw-bridge version
./build/armorclaw-bridge --help

# 7. Clean up backup if successful
rm go.mod.backup go.sum.backup
```

### Manual Verification Required

After automatic module rename, manually verify:

1. **Check `go.mod` first line:**
   ```bash
   head -1 bridge/go.mod
   # Should show: module github.com/armorclaw/bridge
   ```

2. **Verify no stale references:**
   ```bash
   grep -r "github.com/armorclaw" bridge/
   # Should return only go.mod.backup (or nothing)
   ```

3. **Verify imports updated:**
   ```bash
   grep "github.com/armorclaw/bridge" bridge/go.sum | wc -l
   # Should show multiple lines (dependencies updated)
   ```

---

## 📋 Pre-Execution Checklist

Before starting the rename:

### Decisions Made
- [ ] ArmorClaw Evolution → ArmorClaw Evolution decision confirmed (or default used)
- [ ] GitHub organization rename strategy confirmed
- [ ] Docker Hub rename strategy confirmed
- [ ] Backward compatibility: No (clean break)

### Preparation Complete
- [ ] All 4 agents assigned and available
- [ ] Feature branches created for each agent
- [ ] Git baseline commit identified
- [ ] Rollback plan communicated to all agents
- [ ] Go module backup procedure documented
- [ ] Docker Hub access confirmed (if pushing images)

### Testing Environment Ready
- [ ] Docker Desktop running and accessible
- [ ] Go 1.24+ installed and verified
- [ ] Test suite baseline results recorded
- [ ] CI/CD environment status noted

---

## 🚨 Common Pitfalls & Solutions

### Pitfall 1: Go Module Imports Not Updated
**Symptom:** Build fails with "module github.com/armorclaw/bridge: not found"
**Solution:** Run `go mod tidy` after `go mod edit -module=...`

### Pitfall 2: Docker Compose Container Name Conflicts
**Symptom:** "Container name already in use" errors
**Solution:** Stop all containers before renaming: `docker-compose down`

### Pitfall 3: Socket Path Permissions
**Symptom:** "Permission denied" on `/run/armorclaw/bridge.sock`
**Solution:** Ensure directory created with correct permissions: `sudo mkdir -p /run/armorclaw`

### Pitfall 4: Environment Variables in Active Shells
**Symptom:** Scripts use old `ARMORCLAW_` variables
**Solution:** Restart shells or `unset ARMORCLAW_*` after testing

### Pitfall 5: Docker Volume Data Loss
**Symptom:** Data loss after volume rename
**Solution:** Volume paths in docker-compose are bind mounts, not named volumes, so data persists on host

---

## 📊 Progress Tracking Template

Use this template to track progress during execution:

```
=== ArmorClaw Rename Progress ===

Wave 1 (Parallel):
  Agent 1 (Infrastructure): [ ] Not Started | [ ] In Progress | [ ] Complete
    - Go module: [ ]
    - Docker Compose: [ ]
    - Go code: [ ]
    - Container scripts: [ ]

  Agent 2 (Documentation): [ ] Not Started | [ ] In Progress | [ ] Complete
    - Core docs: [ ]
    - User guides: [ ]
    - ArmorClaw Evolution decision: [ ]
    - GitHub URLs: [ ]

Wave 2 (After Wave 1 Complete):
  Agent 3 (DevOps): [ ] Not Started | [ ] In Progress | [ ] Complete
    - Deploy scripts: [ ]
    - CI/CD: [ ]
    - Config files: [ ]

  Agent 4 (QA): [ ] Not Started | [ ] In Progress | [ ] Complete
    - Test scripts: [ ]
    - Final audit: [ ]
    - Build verification: [ ]

=== Final Verification ===
  Grep audit:
    - ARMORCLAW: [ ] 0 occurrences
    - ArmorClaw: [ ] 0 occurrences
    - armorclaw: [ ] 0 occurrences

  Build tests:
    - Go build: [ ] Pass
    - Docker build: [ ] Pass
    - Test suite: [ ] Pass

=== Approval ===
  Agent 4 Sign-off: [ ] Approved
  Ready for merge: [ ] Yes
```

---

**Plan Status:** ⏳ Awaiting Approval
**Estimated Completion:** 2-3 hours after approval (with 4 parallel agents)
**Risk Level:** Medium (mitigated by phased approach and verification)
**Gap Analysis:** ✅ Complete - All critical gaps identified and addressed

