# ArmorClaw Skills Review

> **Review Date:** 2026-04-18
> **Supersedes:** DEPLOYMENT_SKILLS_REVIEW.md (2026-04-05)
> **Scope:** All 4 skill layers — Deployment YAML, Bridge Go, OpenClaw TypeScript, Container Python
> **Method:** Static code review (no live VPS)

## Executive Summary

Static review of all 4 skill layers found **46 findings** (3 CRITICAL, 10 HIGH, 15 MEDIUM, 18 LOW) across 15 Go files, 5 YAML files, 60 SKILL.md files, and 3 Python files. The Bridge Go layer dominates risk with zero test coverage on 6,372 lines of security-critical code.

| Layer | Lines | Findings | Test Coverage | Risk Level |
|-------|-------|----------|---------------|------------|
| Deployment YAML | ~952 | 14 | ~0% functional | 🟡 MEDIUM |
| Bridge Go | 6,372 | 25 | 0% | 🔴 CRITICAL |
| OpenClaw TS | ~1,700 | 2 | Good (~2.4:1) | 🟢 LOW |
| Container Python | 560 | 3 | 0% | 🟡 MEDIUM |

**Top 3 Risks:**

1. **S1** (CRITICAL) — `containsDangerousChars` blocks legit inputs: URLs with `&`, JSON `{}`, math `()`, HTML `<>`. Makes structured data skills unusable. (`executor.go:266-274`)
2. **S2** (CRITICAL) — Allow-by-default policy. Any unregistered skill passes security. Combined with missing policies for email.send and webdav. (`executor.go:362-363`)
3. **S3** (CRITICAL) — Hand-rolled YAML parser can't parse 52+ bundled SKILL.md files that use nested `metadata.openclaw` objects. Self-documented as "Phase 1 scaffolding". (`registry.go:352-396`)

**Previous review status:** 1 of 12 items resolved (YAML syntax errors). 11 carried forward with new tracking IDs. 34 net-new findings from deeper Bridge/Container analysis.

**Estimated remediation:** ~4,800 lines of Go tests + ~282 deployment/Python tests needed. Top 10 action items are ordered by risk.

## Methodology

- **Scope:** All 4 skill layers. Static code review only (no live VPS execution).
- **Tools:** Manual code reading, grep, `bash -n`, YAML validation, file:line cross-referencing.
- **Sources reviewed:** 15 Go files (`bridge/internal/skills/`), 5 YAML files (`.skills/`), 60 SKILL.md files (bundled + extensions), 3 Python files (`container/openclaw/skills/`), 1 RPC file (`bridge/pkg/rpc/`).
- **Previous review:** DEPLOYMENT_SKILLS_REVIEW.md (2026-04-05) cross-referenced. 12 prior action items tracked forward.
- **Test plan references:** Full test specifications stored in `.sisyphus/notepads/skills-review/` and summarized below.

---

## Layer 1: Deployment Skills (.skills/)

### Overview

Four deployment YAML skills provide VPS provisioning, deployment, status checking, and Cloudflare HTTPS setup. A fifth skill (`health.yaml`) handles periodic health checks. All files parse correctly, and referenced shell scripts pass syntax validation.

| Skill | File | Version | Steps | Lines | Purpose |
|-------|------|---------|-------|-------|---------|
| Deploy | `deploy.yaml` | 2.0.0 | 7 | 280 | Full VPS deployment with mode detection |
| Status | `status.yaml` | 1.0.0 | 9 | 210 | Container/service/SSL health checks |
| Cloudflare | `cloudflare.yaml` | 1.0.0 | 4 | 339 | Tunnel or Proxy HTTPS setup |
| Provision | `provision.yaml` | 1.0.0 | 3 | 87 | QR code generation for mobile pairing |

### Capabilities per Skill

**deploy.yaml** — 9 parameters, 7 steps. Automation ranges from `auto` (detect OS, wait for services) to `confirm` (deploy installer) to `guide` (Cloudflare API token setup). Handles Native, Sentinel, and Cloudflare deployment modes. SSH-based execution with `$ENV_VARS` prefix for secret passthrough.

**status.yaml** — 5 parameters, 9 steps. Checks Docker containers, Matrix API, Bridge RPC, SSL/TLS certs, Docker networks, and volumes. Uses `socat` for RPC socket probing. All 5 parameters are wired into step commands.

**cloudflare.yaml** — 5 parameters, 4 steps. Detects network topology, validates Cloudflare API token against live endpoint, runs `cloudflared` setup, verifies HTTPS. Uses `nc`, `jq`, `grep -qE` for network/JSON checks.

**provision.yaml** — 3 parameters, 3 steps. Generates QR code via `armorclaw-provision.sh`, displays deep link for manual entry. Steps 1 and 2 use `sudo` marked as `auto`.

### Expected Results per Step

Each step runs a bash command via SSH. Successful steps advance to the next. Failed steps halt execution and display diagnostics. The `deploy.yaml` wait loop retries up to 30 times (60s total) for service readiness. The `status.yaml` health script step is gated behind `confirm` for optional verbose output.

Step-by-step automation levels across all 4 skills:

| Skill | Step | Name | Automation | Notes |
|-------|------|------|-----------|-------|
| deploy | 1 | detect_os | auto | `[[ ]]` bash-isms, no PS branch |
| deploy | 2 | validate_environment | auto | SSH validation, clean |
| deploy | 3 | prepare_cloudflare | guide | API token guidance, gated |
| deploy | 4 | deploy_installer | confirm | `curl pipe bash`, see S4 |
| deploy | 5 | wait_for_services | auto | 30 retries x 2s = 60s |
| deploy | 6 | verify_installation | auto | `curl -sfk` (insecure flag) |
| deploy | 7 | get_connection_info | auto | Clean display |
| status | 1-8 | various checks | auto | `return 0` bug at step 7, see S5 |
| status | 9 | run_health_script | confirm | Gated, optional verbose |
| cloudflare | 1 | detect_network | auto | Uses `nc` (3x), graceful fallback |
| cloudflare | 2 | validate_prerequisites | auto | `jq`, `grep -qE`, live API check |
| cloudflare | 3 | run_setup | confirm | Unquoted `$SETUP_CMD`, see F9 |
| cloudflare | 4 | verify_https | auto | Clean, optional `openssl` |
| provision | 1 | generate_qr | auto | `sudo $CMD` (unquoted), see S16 |
| provision | 2 | display_deep_link | auto | `sudo` for display-only, see S16 |
| provision | 3 | manual_entry | guide | Clean instructional |

### Findings

| ID | Severity | File | Line | Description | Remediation |
|----|----------|------|------|-------------|-------------|
| S4 | 🟠 HIGH | `deploy.yaml` | 163, 166 | `curl -fsSL \| bash` from unversioned `main` branch. No integrity check, no tag pinning, no checksum. Supply chain attack vector. | Pin to release tag: `https://raw.githubusercontent.com/Gemutly/ArmorClaw/v4.8.0/deploy/install.sh`. Add SHA256 checksum verification. |
| S5 | 🟠 HIGH | `status.yaml` | 150 | `return 0` used outside function scope. Produces bash error and leaks control flow to subsequent commands. | Replace with `exit 0`. |
| S16 | 🟡 MEDIUM | `provision.yaml` | 42, 48 | Steps use `sudo $CMD` (unquoted variable) marked `auto`. Privilege escalation without user confirmation. | Quote variable: `sudo "$CMD"`. Change automation to `confirm`. |
| F8 | 🟡 MEDIUM | `deploy.yaml` | 150, 157, 163 | Admin password and API tokens passed via `$ENV_VARS` prefix to SSH. Visible in `ps auxww` on remote VPS. | Use `env` command or SSH `SendEnv`/`AcceptEnv` to pass secrets. |
| S22 | 🟢 LOW | `deploy.yaml` | 271, 277, 280 | Hardcoded IP `5.183.11.149` in 4 example commands. Likely a real VPS IP. | Replace with RFC 5737 documentation IPs (`192.0.2.1` or `203.0.113.1`). |
| S23 | 🟢 LOW | `deploy.yaml` | 219 | `curl -sfk` uses `-k` (insecure) to skip TLS verification on health endpoint. | Acceptable for health checks. Document the rationale. |
| F9 | 🟢 LOW | `cloudflare.yaml` | 250 | `$SETUP_CMD` unquoted. Word splitting risk if domain contains spaces. | Quote: `"$SETUP_CMD"`. |
| F10 | 🟢 LOW | `deploy.yaml`, `cloudflare.yaml`, `provision.yaml` | — | 3 unused parameters: `openrouter_api_key`, `ssh_key`, `vps_ip`. Declared but never referenced. | Remove or wire into step commands. |
| T6 | 🟢 LOW | `.skills/` | — | No unit tests for skill YAML validation. No integration tests for skill execution. Only `test-deployment-skills.sh` checks structure. | Add YAML schema tests and skill execution dry-run tests. |

### Platform Claims vs Reality

Every skill claims cross-platform support (Windows PowerShell, Git Bash, WSL) but contains zero PowerShell constructs.

| Skill | Claims | Reality | Verdict |
|-------|--------|---------|---------|
| deploy.yaml | linux, macos, windows-gitbash, windows-powershell, wsl | Bash-only: `[[ ]]`, `${var}`, `$()`. No PowerShell. | 🟠 PARTIALLY FALSE |
| status.yaml | linux, macos, windows | Bash-only: `[[ ]]` (8x), `stat -c/-f`, `socat`. | 🔴 FALSE |
| cloudflare.yaml | linux, macos, windows-gitbash, windows-powershell, wsl | Bash-only: `nc`, `jq`, `grep -qE`. No PowerShell. | 🟠 PARTIALLY FALSE |
| provision.yaml | linux, macos, windows | Uses `sudo` (2x). Not available natively on Windows. | 🔴 FALSE |

Bash-only constructs found: `[[ ]]` (12x), `${var}` (45x), `$()` (6x), `stat` (1x), `nc` (3x), `socat` (1x), `jq` (1x), `sudo` (2x). PowerShell constructs found: **zero**.

### SKILL.md Frontmatter Consistency

Four SKILL.md files accompany the deployment skills. Two have missing version fields. All four have description mismatches with their corresponding YAML files.

| Skill | YAML Version | SKILL.md Version | YAML Description | SKILL.md Description | Issue |
|-------|-------------|------------------|-----------------|---------------------|-------|
| deploy | 2.0.0 | 2.0.0 | "Deploy ArmorClaw to VPS with cross-platform support (Linux, macOS, Windows, WSL)" | "Deploy ArmorClaw to VPS with cross-platform support (Linux, macOS, Windows, PowerShell, Git Bash, WSL)" | Platform list differs |
| status | 1.0.0 | 1.0.0 | "Check ArmorClaw deployment health and status on VPS..." | "Check ArmorClaw deployment health and status" | Truncated |
| cloudflare | 1.0.0 | **MISSING** | "Set up Cloudflare HTTPS for ArmorClaw with Tunnel or Proxy mode..." | "Use when setting up Cloudflare HTTPS..." | Version absent; style differs |
| provision | 1.0.0 | **MISSING** | "Generate QR codes for secure mobile app provisioning..." | "Use when setting up new ArmorChat or ArmorTerminal mobile devices..." | Version absent; focus differs |

### Automation Level Issues

Provision steps use `sudo` with `auto` automation, meaning the tool executes privilege escalation without asking the user. Deploy correctly gates its installer step behind `confirm`. The inconsistency suggests provision was not reviewed for privilege escalation risk.

| Skill | Step | Current | Recommended | Reason |
|-------|------|---------|-------------|--------|
| provision.yaml | generate_qr | auto | confirm | Uses `sudo $CMD` (unquoted) |
| provision.yaml | display_deep_link | auto | confirm | Uses `sudo` for display-only operation |

### Previous Review Status

A prior review (DEPLOYMENT_SKILLS_REVIEW.md) produced 12 action items. Only 1 has been resolved (YAML syntax errors). The remaining 11 are carried forward with new tracking IDs in this review.

| Previous Item | Status | New ID |
|--------------|--------|--------|
| curl pipe from unversioned main | NOT FIXED | S4 |
| `return 0` outside function | NOT FIXED | S5 |
| Hardcoded IP `5.183.11.149` | NOT FIXED | S22 |
| sudo without confirmation | NOT FIXED | S16 |
| False Windows platform claims | NOT FIXED | P1-P4 |
| Unused parameters | NOT FIXED | F10 |
| SKILL.md version mismatches | NOT FIXED | P6 |
| SKILL.md description inconsistencies | NOT FIXED | P5 |
| YAML syntax errors | RESOLVED | — |

### Test Coverage Assessment

| What | Status |
|------|--------|
| YAML syntax validation | ✅ All 5 files parse correctly |
| Shell script syntax (`bash -n`) | ✅ All 4 referenced scripts pass |
| Step parameter wiring | ⚠️ 3 of 22 parameters unused |
| YAML structural schema | ❌ No schema validator |
| Step execution dry-run | ❌ No integration tests |
| Cross-platform commands | ❌ Zero PowerShell coverage |
| SKILL.md frontmatter | ⚠️ 2 missing versions, 4 description mismatches |

---

## Layer 2: Bridge Go Skills (bridge/internal/skills/)

### Overview

The Go Bridge skill layer is the core execution engine. It handles skill dispatch, security enforcement, and 8 skill implementations spanning web search, email, calendar, file I/O, and data analysis.

| File | Lines | Purpose | Tests | Status |
|------|-------|---------|-------|--------|
| `executor.go` | 579 | PETG pipeline, skill dispatch | ❌ NONE | Core security path |
| `policy.go` | 174 | Tool policy definitions, risk levels | ❌ NONE | Allow-by-default |
| `ssrf.go` | 203 | SSRF validation, private IP blocking | ❌ NONE | IPv6 gaps |
| `registry.go` | 546 | SKILL.md parsing, YAML frontmatter | ❌ NONE | Hand-rolled parser |
| `router.go` | 168 | Domain routing via keyword matching | ❌ NONE | 3 of 9 domains |
| `schema.go` | 224 | OpenAI-compatible schema generation | ❌ NONE | 4 of 15+ skills |
| `allowlist.go` | 143 | IP/CIDR allowlist for admin overrides | ❌ NONE | RPC remove broken |
| `web_search.go` | 331 | DuckDuckGo real; Google/Bing mock | ❌ NONE | Partial |
| `web_extract.go` | 506 | Web content extraction, HTML parser | ❌ NONE | Real impl |
| `email_send.go` | 480 | SMTP email sending | ❌ NONE | **MOCK only** |
| `slack_message.go` | 582 | Slack webhook messaging | ❌ NONE | **MOCK only** |
| `calendar.go` | 589 | CalDAV calendar operations | ❌ NONE | **MOCK only** |
| `webdav.go` | 469 | WebDAV protocol client | ❌ NONE | Real, no timeout |
| `file_read.go` | 420 | File reading with path traversal guard | ❌ NONE | Partial (PDF stub) |
| `data_analyze.go` | 958 | Statistical analysis of tabular data | ❌ NONE | Real impl |

**Totals: 6,372 lines across 15 files. Zero test files.**

An adjacent package, `bridge/pkg/skills/`, contains 2 production files (481 lines) with 692 lines of tests (1.8:1 ratio). The security-critical `internal/skills/` package has no such coverage.

### Executor Pipeline (PETG — 8 Steps)

File: `bridge/internal/skills/executor.go`

| Step | Name | Line | What Happens | Gap |
|------|------|------|--------------|-----|
| 1 | Registry Lookup | 70 | Looks up skill by name | No logging of failed lookups |
| 2 | SkillGate Interception | 80 | PII check if gate configured | Default config has **no gate** |
| 3 | Policy Enforcement | 93 | Checks if skill is allowed | **Allow-by-default** for all |
| 4 | Schema Validation | 103 | Validates params against schema | `extractParametersFromBody()` returns empty map; validation is no-op |
| 5 | Pre-Execution Security | 113 | SSRF check, dangerous char detection | Blocks legitimate `&`, `{}`, `()`, `<>` |
| 6 | Timeout Execution | 126 | Runs handler with deadline | Zero-value timeout = immediate expiry for SKILL.md skills |
| 7 | Post-Execution Filtering | 145 | Redacts sensitive field names | Only exact matches; `access_token` passes |
| 8 | Return Result | 155 | Returns SkillResult with metadata | No further processing |

### Capabilities per Skill

**web_search** — Queries DuckDuckGo API (real). Google and Bing engines return hardcoded mock data. No pagination, no result caching.

**web_extract** — Fetches URLs, parses HTML to extract text, links, images. Full real implementation with `goquery` HTML parser. Creates new `http.Client{}` per request (no connection pooling). No streaming size limit.

**email_send** — Returns mock success without sending. Real SMTP code is commented out (lines 287-341). Mock config includes hardcoded `"mock-password"`. No policy entry despite being highest-risk tool.

**slack_message** — Returns mock response. Real API code commented out (lines 433-512). Reads `SLACK_WEBHOOK_URL` env var for config, leaks var name in error messages.

**calendar** — In-memory `mockCalendarClient`. Events don't persist across calls, making conflict detection useless. `hasOverlap()` normalizes to day boundaries, so same-day time conflicts are always detected.

**webdav** — Full real implementation: PROPFIND, GET, PUT, DELETE. Sends Basic Auth over potentially non-TLS connections. No timeout on `http.Client{}`. No policy entry.

**file_read** — Reads text, JSON, CSV (real). PDF returns hardcoded "requires additional library" string. Rejects absolute paths. Path validation doesn't resolve symlinks.

**data_analyze** — Pure computation, no external dependencies. Full statistical analysis (mean, median, std dev, correlation, regression). 958 lines, real implementation.

### Findings

| ID | Severity | File | Line | Description | Remediation |
|----|----------|------|------|-------------|-------------|
| S1 | 🔴 CRITICAL | `executor.go` | 266-274 | `containsDangerousChars` blocks `&`, `{}`, `()`, `<>`. Breaks URLs with query params, JSON bodies, math expressions, HTML. | Replace with context-aware allowlist per parameter type. |
| S2 | 🔴 CRITICAL | `executor.go` | 362-363 | `IsAllowed()` is allow-by-default. Any skill not explicitly blocked passes. New/unknown skills auto-allowed. | Flip to deny-by-default. Require explicit policy registration. |
| S3 | 🔴 CRITICAL | `registry.go` | 352-396 | Hand-rolled YAML parser cannot handle lists, nested objects, or multiline values. 52+ SKILL.md files would fail to load. Code self-documented as "Phase 1 scaffolding". | Replace with `gopkg.in/yaml.v3` for frontmatter parsing. |
| S6 | 🟠 HIGH | `ssrf.go` | 21-29 | No IPv6 private range blocking. `fc00::/7`, `fe80::/10`, `::1/128`, `ff00::/8` not in `privateNetworks`. SSRF via IPv6 localhost. | Add IPv6 CIDRs to privateNetworks list. |
| S7 | 🟠 HIGH | `ssrf.go` | 113-129 | `extractHost()` doesn't strip userinfo. `https://attacker@169.254.169.254/` passes `attacker@169.254.169.254` as host. | Use `net/url.Parse()` instead of hand-rolled parsing. |
| S8 | 🟠 HIGH | `policy.go` | — | `email.send` has no policy despite being highest-risk tool. Combined with allow-by-default, anyone can invoke. | Add explicit policy: risk=high, AutoExecute=false, rate=5/min. |
| S9 | 🟠 HIGH | `webdav.go` | 158-469 | WebDAV makes real HTTP requests with no configurable timeout. No rate limit, no output cap. | Add timeout to http.Client. Require policy entry. |
| S10 | 🟠 HIGH | `webdav.go` | 179-181 | WebDAV sends Basic Auth over potentially non-TLS connections. SSRF only validates initial URL, not redirect targets. | Enforce HTTPS in WebDAV params validation. |
| T1 | 🟠 HIGH | `internal/skills/*.go` | — | 6,372 lines of Go across 15 files with zero tests. Covers executor, policy, SSRF, registry, router, schema, allowlist, all skills. | Add tests for executor.go, ssrf.go, policy.go, registry.go at minimum. |
| T2 | 🟠 HIGH | `pkg/rpc/methods_skills.go` | 320-370 | `handleSkillsAllowlistRemove` acknowledges removal but doesn't call `RemoveAllowedIP()`/`RemoveAllowedCIDR()`. Returns success without action. | Wire up existing AllowlistManager methods. |
| F1 | 🟠 HIGH | `executor.go` | 80 | Default executor has no PII interception. SkillGate is optional. `NewSkillExecutor()` passes empty config. | Install default SkillGate with minimum PII detection. |
| F2 | 🟠 HIGH | `calendar.go` | 388-394 | `hasOverlap()` normalizes times to midnight. Same-day conflict detection is useless. Events at 9am and 2pm flagged as conflicting. | Compare actual start/end times, not day boundaries. |
| F3 | 🟠 HIGH | `registry.go` | 232 | `extractParametersFromBody()` always returns empty map. Schema validation is a no-op. Required parameters never validated. | Implement extraction or remove dead validation path. |
| S11 | 🟡 MEDIUM | `ssrf.go` | — | No DNS rebinding protection. DNS resolved once, HTTP client follows redirects without re-validating IPs. TOCTOU attack. | Pin resolved IP for HTTP request, or validate redirects. |
| S12 | 🟡 MEDIUM | `executor.go` | 126 | `skill.Timeout` defaults to zero for SKILL.md-loaded skills. Zero Duration = immediate expiry. All SKILL.md skills would timeout instantly. | Set default timeout (30s) when skill.Timeout is zero. |
| S13 | 🟡 MEDIUM | `executor.go` | 132 | Timeout detection checks `ctx.Err()` instead of `executionCtx.Err()`. Parent cancellation misreported as timeout. | Check `executionCtx.Err()` for timeout detection. |
| S14 | 🟡 MEDIUM | `slack_message.go` | 380-383 | `getSlackConfig()` error exposes env var name: `"SLACK_WEBHOOK_URL environment variable not set"`. | Return generic error without env var name. |
| S15 | 🟡 MEDIUM | `email_send.go` | 229-242 | `getSMTPConfig()` mock config includes hardcoded `"mock-password"`. If mock leaks to production, credentials are static. | Use feature flag to prevent mock in production builds. |
| F4 | 🟡 MEDIUM | `router.go` | — | Only 3 domains with keyword maps (weather, github, web). Missing: email, slack, calendar, webdav, file, data. | Add keyword maps for all 9+ skill domains. |
| F5 | 🟡 MEDIUM | `schema.go` | 64-100 | `generateOpenAISchema()` switch handles 4 skills only. Others get empty schemas, invisible to AI function calling. | Add schema generation for all skills or generic fallback. |
| F6 | 🟡 MEDIUM | `file_read.go` | 338-339 | `validateFilePath()` rejects absolute paths. Skill limited to working directory relative reads. | Consider allowlisting specific absolute paths. |
| F7 | 🟡 MEDIUM | `pkg/rpc/methods_skills.go` | 16-78 | v6 MCP router path bypasses executor's PETG pipeline. Security depends entirely on MCP router. | Ensure MCP router implements equivalent checks. |
| T5 | 🟡 MEDIUM | `executor.go` | 277-293 | Custom `contains()` reinvents `strings.Contains()`. Untested custom logic in security-critical path. | Replace with `strings.Contains()`. |
| S19 | 🟢 LOW | `executor.go` | 296-307 | `filterSensitiveFields` only redacts exact matches. `access_token`, `bearer`, `api_key_v2` pass through. | Use case-insensitive substring matching or regex. |
| S20 | 🟢 LOW | `file_read.go` | 326-365 | `validateFilePath` doesn't resolve symlinks. Symlink pointing outside allowed directory bypasses check. | Resolve symlinks with `filepath.EvalSymlinks()` before validation. |
| S21 | 🟢 LOW | `web_extract.go` | 185-186 | New `http.Client{}` per request, no pooling. No streaming size limit (checked after full read). | Add `io.LimitReader` for streaming enforcement. |
| F11 | 🟢 LOW | `policy.go` | — | `MaxConcurrent` and `RateLimit` fields defined but never enforced. Dead fields. | Implement enforcement or remove. |

### YAML Parser Limitations

File: `bridge/internal/skills/registry.go:352-396`

The hand-rolled YAML frontmatter parser handles simple `key: value` pairs only. The code comment at line 109 acknowledges this is "Phase 1 scaffolding".

| Feature | Supported | Impact if Used |
|---------|-----------|----------------|
| Simple `key: value` pairs | ✅ | N/A |
| Quoted strings | ✅ | N/A |
| Comment lines | ✅ | N/A |
| Lists (`tags: [web, api]`) | ❌ | Silently ignored |
| Nested objects (`metadata.version`) | ❌ | Skill fails to load |
| Multiline values (`description: \|`) | ❌ | Truncated |
| Booleans (`enabled: true`) | ❌ | Stored as string |
| Numbers (`timeout: 30`) | ❌ | Stored as string |
| Flow sequences | ❌ | Raw string |
| Anchors/aliases | ❌ | Corruption |

Parse failures cause `ScanSkills()` to print a warning and skip the skill (registry.go:69). The 52+ bundled SKILL.md files use nested `metadata.openclaw` objects, which would fail silently.

### RPC Methods

File: `bridge/pkg/rpc/methods_skills.go` (571 lines)

The RPC layer exposes 14 skill-related methods. Two paths exist for skill execution:

| Path | Trigger | Security Pipeline |
|------|---------|-------------------|
| v6 MCP path | `mcpRouter != nil` | Bypasses PETG. Depends on MCP router. |
| Legacy path | `mcpRouter == nil` | Full PETG pipeline (Steps 1-8), but allow-by-default. |

**Missing RPC shortcuts:** `calendar` and `webdav` have dedicated `ExecuteXxx()` functions but no direct `skills.calendar` or `skills.webdav` RPC methods. These are only reachable through the generic `skills.execute`.

**Allowlist remove bug:** `handleSkillsAllowlistRemove` (lines 320-370) acknowledges removal but never calls `RemoveAllowedIP()` or `RemoveAllowedCIDR()`. Returns success without acting. The underlying `AllowlistManager` methods exist but are not wired.

### Mock vs Real Implementation

| Skill | Lines | Status | Mock Risk |
|-------|-------|--------|-----------|
| web_search | 331 | PARTIAL | Google/Bing return hardcoded data |
| web_extract | 506 | REAL | N/A |
| email_send | 480 | MOCK | Returns success without sending. Masks delivery failures. |
| slack_message | 582 | MOCK | Reads env var but never uses it. Leaks var name in errors. |
| calendar | 589 | MOCK | Events don't persist. Conflict detection is theater. |
| webdav | 469 | REAL | No timeout, no TLS enforcement. |
| file_read | 420 | PARTIAL | PDF returns hardcoded stub string. |
| data_analyze | 958 | REAL | Pure computation. No external deps. |

**3 fully mock, 2 partially mock, 3 fully real.**

### Test Coverage Assessment

| Directory | Files | Lines | Tests | Test:Code | Coverage |
|-----------|-------|-------|-------|-----------|----------|
| `bridge/internal/skills/` | 15 | 6,372 | 0 | 0:1 | **0%** |
| `bridge/pkg/skills/` | 2 | 481 | 692 | 1.8:1 | ~144% |
| `bridge/pkg/rpc/` | 1 | 571 | unknown | — | Partial |

The `internal/skills/` package is the security-critical core with 6,372 lines and **zero tests**. The adjacent `pkg/skills/` package (learned skills, extractor) has excellent coverage at 1.8:1 test-to-code ratio. The gap is stark: the code that enforces policy, validates SSRF, and dispatches skills has no automated verification.

## Layer 3: OpenClaw TypeScript Skills

### Overview

The OpenClaw TypeScript skill infrastructure manages skill discovery, loading, filtering, and prompt assembly for the container runtime. 16 infrastructure files in `src/agents/skills/` process 60 SKILL.md files distributed across 3 locations.

| Metric | Value |
|--------|-------|
| Infrastructure files | 16 (`src/agents/skills/`) |
| SKILL.md files | 60 (52 bundled + 6 extension + 2 bridge) |
| Test files | 31 (4,093 lines) |
| Test-to-infra ratio | ~2.4:1 |

### Skill Inventory

| Location | Count | Type | Path |
|----------|-------|------|------|
| Bundled | 52 | Core skills | `container/openclaw-src/skills/` |
| Extension | 6 | Plugin skills | `container/openclaw-src/extensions/` |
| Bridge | 2 | Go bridge skills | `bridge/internal/skills/` |
| **Total** | **60** | | |

**Extension skills:** open-prose (prose writing), lobster (lobster tool), feishu-drive, feishu-wiki, feishu-doc, feishu-perm.

### TS Infrastructure Architecture

| File | Lines | Purpose |
|------|-------|---------|
| `workspace.ts` | 779 | Core discovery, loading, snapshot building, sync |
| `types.ts` | 89 | Type definitions (SkillEntry, SkillSnapshot) |
| `frontmatter.ts` | 117 | YAML frontmatter parsing, metadata resolution |
| `config.ts` | 112 | Eligibility filtering (OS, requires, bundled allowlist) |
| `filter.ts` | 31 | Filter normalization and comparison |
| `serialize.ts` | 14 | Async serialization mutex for skill sync |
| `plugin-skills.ts` | 74 | Plugin-based skill directory resolution |
| `bundled-dir.ts` | — | Bundled skills directory path |
| `bundled-context.ts` | — | Bundled skill context loading |
| `env-overrides.ts` | — | Environment-based overrides |
| `refresh.ts` | — | Snapshot refresh via file watching |
| `tools-dir.ts` | — | Tools directory resolution |

Key patterns: Map-based merge (last-write-wins, `workspace.ts:369-388`), binary search truncation for prompt char limits (`workspace.ts:427-441`), path compaction (replaces `$HOME` with `~`), sanitized command names (lowercased, special chars stripped, 32 char max), unique command deduplication (`_2`, `_3` suffixes on collision).

### Skill Discovery Precedence

Seven-level precedence (lowest to highest, last-write-wins in Map merge):

| Level | Source | Config Control |
|-------|--------|----------------|
| 1 (lowest) | `extraDirs` | `config.skills.load.extraDirs[]` |
| 2 | `pluginSkillDirs` | Plugin manifest `skills[]` entries |
| 3 | `openclaw-bundled` | `container/openclaw-src/skills/` |
| 4 | `openclaw-managed` | `~/.config/openclaw/skills/` |
| 5 | `agents-skills-personal` | `~/.agents/skills/` |
| 6 | `agents-skills-project` | `<workspace>/.agents/skills/` |
| 7 (highest) | `openclaw-workspace` | `<workspace>/skills/` |

**Size limits:** 300 candidates/root (suspicious threshold), 200 skills/source, 150 skills/prompt, 30K prompt chars, 256KB max SKILL.md file.

**Eligibility checks** (`config.ts shouldIncludeSkill()`): `enabled` flag → bundled allowlist → OS compatibility → `metadata.always` → `requires` (bins/env/config).

### Parser Comparison: Go vs TS

| Feature | Go Parser (`registry.go:352-396`) | TS Parser (`frontmatter.ts`) |
|---------|--------------------------------------|-------------------------------|
| YAML handling | Hand-rolled line splitter | Proper YAML parser |
| Nested objects (`metadata.openclaw`) | BROKEN — raw string | Fully parsed |
| Lists (`tags: [web, api]`) | BROKEN — splits on `:` | Fully parsed |
| Multiline values (`description: \|`) | BROKEN — truncated | Supported |
| `metadata.openclaw.*` fields | Not extracted | 12 fields extracted |
| `requires` eligibility | Not evaluated | bins/env/config evaluated |
| `install` specs | Not parsed | kind/bins/os fully parsed |
| Domain/Risk/Command | Hardcoded name heuristics | N/A (TS doesn't compute) |
| Self-documentation | "Phase 1 scaffolding" (line 351) | Production-grade |

**Impact:** 52+ bundled SKILL.md files use nested `metadata.openclaw` objects. These load correctly via TS but would fail or silently corrupt via Go. See finding S3.

### Frontmatter Validation Gap

File: `container/openclaw-src/skills/skill-creator/scripts/quick_validate.py:40`

The skill-creator's validator `allowed_properties` includes `name, description, license, allowed-tools, metadata` but omits `homepage`. At least 4 bundled skills (weather, notion, obsidian, 1password) include `homepage` in their frontmatter. Running `quick_validate.py` against them would produce false rejections.

### Security Scanning

File: `src/agents/skills/skill-scanner.ts`

**LINE RULES:** dangerous-exec (child_process exec/spawn), dynamic-code-execution (eval/new Function), crypto-mining (stratum/coinhive/xmrig), suspicious-network (WebSocket non-standard ports).

**SOURCE RULES:** potential-exfiltration (readFile + fetch/post), obfuscated-code (hex sequences, large base64), env-harvesting (process.env + fetch/post).

### Findings

| ID | Severity | File | Line | Description | Remediation |
|----|----------|------|------|-------------|-------------|
| S3 | 🔴 CRITICAL | `bridge/internal/skills/registry.go` | 352-396 | Hand-rolled YAML parser cannot parse the SKILL.md files this repo ships. 52+ skills with nested `metadata.openclaw` objects fail silently. | Replace with `gopkg.in/yaml.v3`. Code comment line 109 acknowledges this. |
| T4 | 🟡 MEDIUM | `skills/skill-creator/scripts/quick_validate.py` | 40 | `allowed_properties` set missing `homepage`. Validator rejects 4+ valid bundled skills. | Add `homepage` to allowed set. |

The TS infrastructure itself is well-tested: 31 test files, 4,093 lines, ~2.4:1 test-to-code ratio. No security findings in the TS skill code.

### Test Coverage Assessment

| Area | Files | Lines | Coverage |
|------|-------|-------|----------|
| Core skills (agents/) | 21 | 2,493 | Excellent |
| Security scanning | 1 | 345 | Good |
| CLI commands | 2 | 304 | Good |
| Cron/scheduling | 1 | 395 | Good |
| Gateway/API | 2 | 101 | Moderate |
| Auto-reply commands | 1 | 131 | Good |
| Config | 1 | 47 | Moderate |
| Commands/onboard | 1 | 185 | Good |
| Discord dedupe | 1 | 26 | Basic |
| Remote skills (infra/) | 1 | 36 | Basic |
| **Total** | **31** | **4,093** | **Good** |

---

## Layer 4: Container Python Runtime Skills

### Overview

Three Python files provide SSL tunnel setup skills for the container runtime. These run inside agent containers to establish external connectivity.

| File | Lines | Purpose |
|------|-------|---------|
| `container/openclaw/skills/__init__.py` | 35 | Exports all SSL skills and handler |
| `container/openclaw/skills/ssl_skill_handler.py` | 157 | Agent instructions for SSL setup |
| `container/openclaw/skills/ssl_tunnel_setup.py` | 368 | Tunnel implementations (ngrok, cloudflare, self-signed) |
| **Total** | **560** | |

### Runtime Skills Registry

The `DEFAULT_SKILLS` dict in `ssl_skill_handler.py` registers one entry:

| Key | Module | Triggers | Priority |
|-----|--------|----------|----------|
| `ssl_setup` | `openclaw.skills.ssl_tunnel_setup` | ssl, https, external access, public url, security warning, tunnel, ngrok, cloudflare | 10 |

**Security rules enforced:** Never access user email. Never ask for passwords. Never automate browser login. Prefer no-login tunnel options. User handles authentication externally.

### SSL Implementations

| Implementation | File:Line | Method | Notes |
|----------------|-----------|--------|-------|
| NgrokTunnelSkill | `ssl_tunnel_setup.py:50-120` | ngrok quick tunnel | Free, temporary, auto-install via apt |
| CloudflareTunnelSkill | `ssl_tunnel_setup.py:130-200` | Cloudflare quick tunnel | Free, temporary URL |
| SelfSignedCertSkill | `ssl_tunnel_setup.py:210-368` | openssl self-signed cert | Local-only, no external tunnel |

### Findings

| ID | Severity | File | Line | Description | Remediation |
|----|----------|------|------|-------------|-------------|
| S17 | 🟡 MEDIUM | `ssl_tunnel_setup.py` | 64-66 | `shell=True` in subprocess calls for ngrok install. Command injection risk if URLs/params are user-controlled. | Use list form: `subprocess.run(["sudo", "apt", "install", ...])`. |
| S18 | 🟡 MEDIUM | `ssl_tunnel_setup.py` | 89, 192 | `pkill ngrok` / `pkill cloudflared` kills ALL instances system-wide, not just skill-managed ones. | Track PIDs and kill specific processes. |
| T3 | 🟡 MEDIUM | `container/openclaw/skills/` | — | Zero test files. 560 lines of Python untested. | Add pytest test suite. |

### Test Coverage Assessment

| What | Status |
|------|--------|
| Unit tests for ssl_tunnel_setup.py | ❌ None |
| Unit tests for ssl_skill_handler.py | ❌ None |
| Integration tests for tunnel lifecycle | ❌ None |
| Input validation tests | ❌ None |

**560 lines. Zero test files. 0% coverage.**

---

## Cross-cutting Concerns

### Parser Inconsistency

The Go parser (`registry.go:352-396`) and TS parser (`frontmatter.ts`) read the same SKILL.md format but produce fundamentally different results. The Go parser is hand-rolled "Phase 1 scaffolding" that handles only `key: value` pairs. The TS parser uses proper YAML.

| Impact | Detail |
|--------|--------|
| Affected files | 52+ bundled SKILL.md files with `metadata.openclaw` nested objects |
| Go behavior | Nested objects stored as raw string, silently skipped on load (`registry.go:69`) |
| TS behavior | Fully parsed with 12 metadata fields extracted |
| Root finding | S3 (CRITICAL) |

If the Bridge ever needs to read the same SKILL.md files the TS runtime uses, 52+ skills would fail to load.

### Mock vs Real Skill Coverage

| Layer | Skill | Implementation | Notes |
|-------|-------|----------------|-------|
| Bridge Go | email_send | MOCK | Returns success without sending |
| Bridge Go | slack_message | MOCK | Reads env var but never calls API |
| Bridge Go | calendar | MOCK | Events don't persist, overlap detection broken |
| Bridge Go | web_search | PARTIAL | DuckDuckGo real, Google/Bing hardcoded |
| Bridge Go | file_read | PARTIAL | PDF returns hardcoded stub |
| Bridge Go | web_extract | REAL | No connection pooling, no size limit |
| Bridge Go | webdav | REAL | No timeout, Basic Auth over non-TLS |
| Bridge Go | data_analyze | REAL | Pure computation, no external deps |
| Python Runtime | ssl_setup (all 3) | REAL | shell=True, no input validation |
| Deployment | deploy, status, cloudflare, provision | REAL (SSH) | Depends on external tool availability |

**Summary:** Bridge has 3 fully mock, 2 partially mock, 3 fully real. Container Python is real. Deployment YAMLs depend on VPS state.

### Test Coverage Summary

| Layer | Lines of Code | Test Lines | Ratio | Assessment |
|-------|---------------|------------|-------|------------|
| Deployment YAML (`.skills/`) | ~952 | 1 structural test | ~0% | ❌ No functional tests |
| Bridge Go internal (`internal/skills/`) | 6,372 | 0 | 0% | ❌ Zero tests, security-critical |
| Bridge Go pkg (`pkg/skills/`) | 481 | 692 | 1.8:1 | ✅ Excellent |
| Bridge RPC (`pkg/rpc/`) | 571 | unknown | partial | ⚠️ Partial |
| OpenClaw TS (`src/agents/skills/`) | ~1,700 | 4,093 | ~2.4:1 | ✅ Good |
| Python Runtime (`container/openclaw/skills/`) | 560 | 0 | 0% | ❌ Zero tests |

**Total untested code: ~7,884 lines** (Bridge internal + Deployment + Python Runtime).

### Previous Review Delta

A prior review (DEPLOYMENT_SKILLS_REVIEW.md, 2026-04-05) produced 12 action items. Current status:

| Status | Count |
|--------|-------|
| RESOLVED | 1 (YAML syntax errors) |
| CARRIED FORWARD | 11 |

Carried items now tracked with specific finding IDs: S4 (curl pipe), S5 (return 0), S22 (hardcoded IP), S16 (sudo without confirm), P1-P4 (false platform claims), F10 (unused params), P5/P6 (SKILL.md inconsistencies), T6/T7 (test gaps). The deeper T2/T3 analysis added 34 net-new findings with more precise tracking.

---

## Test Execution Plans

### Bridge Go Tests (P0–P2)

Full specification: `.sisyphus/notepads/skills-review/task7-bridge-test-plan.md`

| Priority | File | Lines | Findings Covered | Est. Test Lines |
|----------|------|-------|------------------|-----------------|
| P0 | `executor.go` | 579 | S1, S2, S12, S13, S19, T5, F1 | 650 |
| P0 | `ssrf.go` | 203 | S6, S7, S11 | 350 |
| P0 | `policy.go` | 174 | S8, F11 | 200 |
| P0 | `registry.go` | 546 | S3, F3 | 500 |
| P1 | `schema.go` | 224 | F5 | 250 |
| P1 | `router.go` | 168 | F4 | 200 |
| P1 | `allowlist.go` | 143 | T2 | 180 |
| P2 | `calendar.go` | 589 | F2 | 400 |
| P2 | `webdav.go` | 469 | S9, S10 | 350 |
| P2 | `email_send.go` | 480 | S14, S15 | 300 |
| P2 | `slack_message.go` | 582 | S14 | 300 |
| P2 | `file_read.go` | 420 | S20, F6 | 300 |
| P2 | `web_extract.go` | 506 | S21 | 250 |
| P2 | `web_search.go` | 331 | — | 200 |
| P2 | `data_analyze.go` | 958 | — | 420 |

**Totals:** 15 test files, ~95 test functions, ~290 test cases, ~4,800 estimated test lines. Target ratio: 0.75:1.

**Per-file summary (P0 security-critical):**

| File | Test File | # Functions | # Cases | Key Findings Covered |
|------|-----------|-------------|---------|---------------------|
| `executor.go` | `executor_test.go` | 10 | 30 | S1 (dangerousChars), S2 (allow-by-default), S12 (zero timeout), S13 (wrong ctx), S19 (redaction), T5 (custom contains), F1 (no SkillGate) |
| `ssrf.go` | `ssrf_test.go` | 8 | 20 | S6 (no IPv6), S7 (userinfo bypass), S11 (DNS rebinding) |
| `policy.go` | `policy_test.go` | 8 | 18 | S8 (missing email policy), F11 (dead fields) |
| `registry.go` | `registry_test.go` | 11 | 22 | S3 (broken YAML parser), F3 (empty param extraction) |

**Mock strategy:** `httptest.NewServer` for HTTP skills, `t.TempDir()` for filesystem tests, `t.Setenv()` for env var mocks. No external service dependencies.

**Execution order:**
```
Phase 1 (P0): executor_test → ssrf_test → policy_test → registry_test
Phase 2 (P1): schema_test → router_test → allowlist_test
Phase 3 (P2): calendar_test → webdav_test → email_test → slack_test → file_read_test → web_extract_test → web_search_test → data_analyze_test
```

### Deployment + OpenClaw + Container Tests

Full specification: `.sisyphus/notepads/skills-review/task8-deployment-openclaw-test-plan.md`

| Section | Test File | Framework | Test Cases | Findings |
|---------|-----------|-----------|------------|----------|
| Deploy YAML | `tests/test-deployment-skills-functional.sh` | bash/shunit2 | 5 | S4, S22, P1, F10 |
| Status YAML | (same) | bash/shunit2 | 4 | S5, P2 |
| Cloudflare YAML | (same) | bash/shunit2 | 4 | F9, S22, P3 |
| Provision YAML | (same) | bash/shunit2 | 4 | S16, P4 |
| Error scenarios | (same) | bash/shunit2 | 4 | — |
| Cross-platform | `tests/test-cross-platform-claims.sh` | bash/shunit2 | 2 | P1–P4 |
| Frontmatter | `container/openclaw-src/tests/test_frontmatter_conformance.py` | pytest | ~237 | T4, P6, S3 |
| Container Python | `container/openclaw/skills/test_ssl_tunnel_setup.py` | pytest | 16 | S17, S18, T3 |
| Regression | `tests/test-review-regression.sh` | bash/shunit2 | 6 | S4, S5, S16, S22, P6, T4 |
| **Total** | **4 test files** | **bash + pytest** | **~282** | **12 findings** |

**Python test detail (16 pytest cases):**

| Skill | Tests | Key Findings |
|-------|-------|-------------|
| NgrokTunnelSkill | 5 | S17 (`shell=True`), S18 (`pkill`), T3 |
| CloudflareTunnelSkill | 5 | S17, S18, T3 |
| SelfSignedCertSkill | 4 | T3, input validation |
| Subprocess safety | 2 | S17, S18 |

**Frontmatter conformance (237 pytest cases):** Validates all 60 SKILL.md files for name, description, frontmatter presence. Checks `homepage` not rejected by validator (T4). Verifies `metadata.openclaw` parseability (S3 data). Checks version consistency across 4 deployment pairs (P6).

---

## Action Items

Sorted by priority. All items reference specific file:line locations in findings tables above.

| # | ID | Priority | Effort | Layer | Description | File(s) |
|---|----|----------|--------|-------|-------------|---------|
| 1 | S3 | 🔴 CRITICAL | 1 file | Bridge Go | Replace hand-rolled YAML parser with `gopkg.in/yaml.v3` | `registry.go:352-396` |
| 2 | S2 | 🔴 CRITICAL | 1 file | Bridge Go | Flip policy to deny-by-default; require explicit registration | `executor.go:362-363` |
| 3 | S1 | 🔴 CRITICAL | 1 file | Bridge Go | Fix `containsDangerousChars` with context-aware allowlist | `executor.go:266-274` |
| 4 | S6+S7 | 🟠 HIGH | 1 file | Bridge Go | Fix SSRF: add IPv6 ranges, use `net/url.Parse()` | `ssrf.go:21-29, 113-129` |
| 5 | S4 | 🟠 HIGH | 2 lines | Deploy | Pin `curl|bash` to release tag, add SHA256 check | `deploy.yaml:163, 166` |
| 6 | S5 | 🟠 HIGH | 1 line | Deploy | Replace `return 0` with `exit 0` | `status.yaml:150` |
| 7 | T1 | 🟠 HIGH | new files | Bridge Go | Add tests for executor, ssrf, policy, registry at minimum | `internal/skills/*.go` |
| 8 | F1 | 🟠 HIGH | 1 file | Bridge Go | Install default SkillGate with PII detection | `executor.go:80` |
| 9 | F3 | 🟠 HIGH | 1 func | Bridge Go | Implement `extractParametersFromBody` (currently returns empty) | `registry.go:232` |
| 10 | S9+S10 | 🟠 HIGH | 1 file | Bridge Go | Add timeout, HTTPS enforcement, policy entry for WebDAV | `webdav.go:158-469` |

## Recommendations

**Maintenance:** Re-review after any skill layer change. Quarterly review cadence recommended. Each new SKILL.md or Go skill should trigger a focused review of the affected layer.

**CI Integration:** Add deployment YAML linting (`yamllint` or custom schema validator) to CI pipeline. Add `go test ./internal/skills/...` once tests exist. Add `pytest container/openclaw/skills/` for Python runtime. Run `quick_validate.py` against bundled skills in CI.

**Next Steps:** Implement Top 10 action items above (S3 → S2 → S1 → SSRF → deploy fixes → tests → SkillGate → params → WebDAV). Then execute both test plans. Then re-review to validate fixes and close findings.

**Supersedes:** This document replaces DEPLOYMENT_SKILLS_REVIEW.md (2026-04-05). All 12 prior action items are tracked here with updated finding IDs.
