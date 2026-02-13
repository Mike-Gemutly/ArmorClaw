# ArmorClaw UX Analysis - Complete Flow to Element X

> **Date:** 2026-02-07
> **Scope:** Complete user journey from installation to Matrix Element X usage
> **Status:** Analysis Complete - Issues Identified

---

## Executive Summary

This analysis traces the complete user journey from installing ArmorClaw to successfully sending messages via Element X. We identified **15 critical UX issues** across 5 stages of the user journey.

**Overall UX Rating:** 6/10 (Functional but requires significant improvement)

**Primary Issues:**
1. Missing unified Element X quick start guide
2. Scattered documentation across multiple files
3. Docker Compose Stack has technical issues (CGO/SQLCipher)
4. No clear "happy path" for first-time users
5. Missing verification steps after each stage

---

## User Journey Stages

### Stage 1: Installation & Setup

**Current Options:**
1. Shell script wizard (`deploy/setup-wizard.sh`) - 10-15 min
2. CLI wizard (`armorclaw-bridge setup`) - 5-10 min
3. Manual installation - 15-20 min
4. Docker Compose Stack (`launch-element-x.sh`) - 5 min ⭐

**Issues Found:**

#### Issue #1: No Clear Recommended Path
- **Problem:** 4 different installation methods confuse first-time users
- **Impact:** Analysis paralysis, users don't know which to choose
- **Severity:** High
- **Fix:** Create decision tree based on user's goal

#### Issue #2: Docker Compose Stack Broken
- **Problem:** `docker-compose-stack.yml` builds bridge with `CGO_ENABLED=0` but keystore requires CGO for SQLCipher
- **Impact:** Docker Compose method fails at runtime
- **Severity:** Critical
- **Location:** `bridge/Dockerfile` line 8
- **Evidence:** `bridge/pkg/keystore/keystore.go` imports `github.com/mutecomm/go-sqlcipher/v4` which requires CGO

#### Issue #3: launch-element-x.sh Missing Validation
- **Problem:** Script doesn't verify prerequisites (Docker, ports, disk space)
- **Impact:** Cryptic errors mid-deployment
- **Severity:** Medium
- **Fix:** Add pre-flight checks

---

### Stage 2: Initial Configuration

**Current Flow:**
1. Setup wizard creates config file
2. User optionally edits config.toml
3. User adds API keys via `add-key` command

**Issues Found:**

#### Issue #4: Config Validation Missing
- **Problem:** No `validate` command that checks all configuration
- **Impact:** Errors only appear at runtime
- **Severity:** Medium
- **Fix:** Enhanced `validate` command with full checks

#### Issue #5: Matrix Configuration Unclear
- **Problem:** Users don't understand Matrix credentials requirements
- **Impact:** Failed Matrix connections
- **Severity:** High
- **Location:** `docs/guides/configuration.md` lines 95-127
- **Fix:** Add Matrix setup section with examples

#### Issue #6: No Interactive Matrix Setup
- **Problem:** Matrix setup requires manual config editing
- **Impact:** High friction for Matrix adoption
- **Severity:** Medium
- **Fix:** Add Matrix setup to interactive wizard

---

### Stage 3: Infrastructure Deployment (Methods 1-3 only)

**Current Flow:**
1. Deploy Matrix Conduit via `scripts/deploy-infrastructure.sh`
2. Configure DNS
3. Deploy Nginx + Coturn
4. Provision users and rooms

**Issues Found:**

#### Issue #7: Missing Infrastructure Quick Start
- **Problem:** No simple "deploy Matrix" script for local testing
- **Impact:** Users must deploy full infrastructure or use Docker Compose (which is broken)
- **Severity:** Critical
- **Fix:** Create `deploy-local-matrix.sh` for development/testing

#### Issue #8: DNS Requirement Not Clear
- **Problem:** Infrastructure guide assumes domain ownership
- **Impact:** Can't test locally without domain
- **Severity:** High
- **Fix:** Document localhost testing with /etc/hosts

---

### Stage 4: Starting the Agent

**Current Flow:**
1. Start bridge: `systemctl start armorclaw-bridge`
2. Start agent: `armorclaw-bridge start --key <key-id>`
3. Verify agent is running

**Issues Found:**

#### Issue #9: No "Start Agent" in Setup Wizard
- **Problem:** Setup wizard exits without starting first agent
- **Impact:** Users don't know if everything works
- **Severity:** High
- **Fix:** Add "Start first agent" step to wizard

#### Issue #10: Agent Verification Unclear
- **Problem:** No clear success criteria after starting agent
- **Impact:** Users don't know if agent is working
- **Severity:** Medium
- **Fix:** Add `verify-agent` command with clear output

#### Issue #11: No Container Health Check
- **Problem:** Can't easily verify agent container is healthy
- **Impact:** Silent failures
- **Severity:** Medium
- **Fix:** Add health check endpoint and verification script

---

### Stage 5: Element X Connection

**Current Flow:**
1. Download Element X app
2. Connect to homeserver
3. Join room
4. Send commands

**Issues Found:**

#### Issue #12: Missing Element X Quick Start Guide
- **Problem:** README references `element-x-deployment.md` which doesn't exist
- **Impact:** Users can't find complete Element X setup instructions
- **Severity:** Critical
- **Location:** `README.md` line 207
- **Fix:** Create `docs/guides/element-x-quickstart.md`

#### Issue #13: No Element X Screenshots
- **Problem:** Text-only instructions for app connection
- **Impact:** Users struggle with homeserver URL input, etc.
- **Severity:** Medium
- **Fix:** Add screenshots to Element X guide

#### Issue #14: Agent Commands Not Documented in Element X Context
- **Problem:** `element-x-configs.md` only covers `/attach_config`
- **Impact:** Users don't know how to interact with agent
- **Severity:** High
- **Fix:** Add complete command reference for Element X

#### Issue #15: No "Hello World" Example
- **Problem:** No simple first message example
- **Impact:** Unclear what to send first
- **Severity:** Low
- **Fix:** Add "Send your first message" example

---

## Documentation Issues

### Missing Documents
1. **Element X Quick Start** - End-to-end guide to connect via Element X
2. **Local Development Guide** - How to run everything locally for testing
3. **Production Deployment Checklist** - Pre-deployment verification

### Conflicting Information
1. **Multiple "Method 2" labels** in setup guide (fixed, but inconsistent)
2. **CGO requirement** not mentioned in infrastructure guide
3. **Matrix setup** scattered across multiple files

### Navigation Issues
1. **Dead link:** `element-x-deployment.md` referenced but doesn't exist
2. **No "Next Steps"** after each major stage
3. **Unclear progression** from setup to usage

---

## Technical Issues

### Docker Compose Stack
```yaml
# docker-compose-stack.yml line 43-46
bridge:
  build:
    context: ./bridge
    dockerfile: Dockerfile  # This Dockerfile has CGO_ENABLED=0
```

**Problem:** Bridge requires CGO for SQLCipher but Dockerfile disables it
**Impact:** Encrypted keystore doesn't work in Docker
**Fix:** Create separate Dockerfile with CGO support

### Bridge Entrypoint
```dockerfile
# bridge/Dockerfile line 26
CMD ["--daemonize"]
```

**Problem:** `--daemonize` flag doesn't exist (it's `daemon start`)
**Impact:** Container exits immediately
**Fix:** Change CMD or add proper daemon support

### Volume Permissions
```yaml
# docker-compose-stack.yml line 50
- bridge_run:/run/armorclaw
```

**Problem:** Named volume has root ownership, bridge runs as UID 10002
**Impact:** Bridge can't create socket
**Fix:** Init container or proper ownership setup

---

## Priority Fixes

### P0 - Critical (Blocks Usage)
1. Fix Docker Compose Stack CGO issue
2. Create Element X Quick Start guide
3. Fix bridge entrypoint CMD

### P1 - High (Significant Friction)
4. Add "Start first agent" to setup wizard
5. Add Matrix setup to CLI wizard
6. Create local development Matrix deployment script

### P2 - Medium (Annoying)
7. Add pre-flight checks to launch-element-x.sh
8. Add agent verification command
9. Add health check endpoint

### P3 - Low (Nice to Have)
10. Add screenshots to Element X guide
11. Add "Hello World" example
12. Create production deployment checklist

---

## Proposed "Happy Path"

### For First-Time Users (Local Testing)
1. Run `./deploy/setup-everything.sh` (new script)
2. Script installs and configures everything
3. Starts local Matrix, Bridge, and Agent
4. Displays Element X connection details
5. User opens Element X and sends `/ping`

### For Production Deployment
1. Run `./deploy/setup-wizard.sh` on server
2. Deploy infrastructure via `./deploy/infrastructure.sh`
3. Verify all services via `./deploy/verify-all.sh`
4. Connect Element X
5. Send first `/attach_config` command

---

## Next Steps

1. Create Element X Quick Start guide
2. Fix Docker Compose Stack
3. Add "Start first agent" to setup wizard
4. Create local development script
5. Add comprehensive verification commands
6. Update documentation cross-references

---

**Analysis Complete:** 2026-02-07
**Issues Found:** 15
**P0 Issues:** 3
**P1 Issues:** 3
**P2 Issues:** 3
**P3 Issues:** 3
**Documentation Gaps:** 3
