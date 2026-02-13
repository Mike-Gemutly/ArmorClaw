# ArmorClaw Security Hardening Plan – 2026 Edition

**Goal**  
Build a defensible, auditable, and cost-aware architecture for ArmorClaw that follows **secure-by-default** and **zero-trust-for-agents** principles while remaining pragmatic for a small-to-medium deployment footprint.

**Last Updated**  
2026-02-08

**Estimated Effort**  
6–9 weeks (≈25–38 dev days) depending on testing depth and environment complexity

**Status**  
Ready for kick-off | Needs final prioritization & resource commitment

## 1. Executive Summary

ArmorClaw must survive realistic attacker behaviors (compromised Matrix accounts, prompt injection, cost exhaustion, accidental leakage) without imposing unacceptable UX friction or operational overhead.

Four independent but composable security layers:

1. **Zero-Trust Request Pipeline** — explicit trust decisions + PII-aware sanitization
2. **Financial & Resource Guardrails** — predictable spend + abuse resistance
3. **Runtime Containment & Host Hardening** — reduce lateral movement & exposure window
4. **Ephemeral Secrets & Memory Hygiene** — minimize secret lifetime and blast radius

**Core philosophy change**  
Assume every Matrix message, LLM output, and tool call is **potentially malicious** until proven otherwise — even from "trusted" rooms/users.

## 2. Explicit Threat Model (2026 perspective)

### In scope – we actively defend against

- Compromised or malicious Matrix accounts (user or room takeover)
- Prompt/tool injection through LLM → unintended tool calls or data exfil
- Cost-exhaustion (DoS via long/infinite agent loops)
- Accidental / careless PII leakage into LLM providers or logs
- Container breakout or lateral movement from a single malicious execution
- Long-lived secrets in memory, logs, crash dumps or swap

### Explicit non-goals (out of scope)

- Nation-state hardware/firmware attacks
- Kernel / hypervisor / container runtime exploits (assume best-effort mitigation)
- Supply-chain compromise of base images or upstream Go/Python deps
- Full end-to-end encryption of Matrix transport

**Recommendation**  
Document this model in `SECURITY.md` and revisit **after every major LLM provider change** or **every 6 months**.

## 3. Guiding Principles

1. **Never trust, always verify** — especially for AI agents and LLM outputs
2. **Fail closed** — unknown → reject / budget stop / container kill
3. **Defense in depth** — overlapping controls (policy + budget + TTL + scrubbing)
4. **Observability is security** — every rejection / budget hit / kill must be traceable
5. **Context preservation over naive redaction** — prefer smart anonymization when possible

## 4. Four Implementation Phases (independent activation)

### Phase 1 – Zero-Trust Request Pipeline (Weeks 1–2.5)

**Must-have controls**

- **Trusted identity allow-list**  
  - Per-user Matrix MXID  
  - Per-room allow-list (more permissive but still explicit)  
  - Configurable rejection message (default: neutral 403-style reply)

- **PII / sensitive data sanitization layer** (before any LLM call)  
  - Use **Presidio**-like library or lightweight regex + context-aware model  
  - **Strategies** (configurable per field type):  
    - redact → `[REDACTED_EMAIL]`  
    - synthetic swap → `user_47f8a2@domain.invalid` (preserves format & semantics)  
    - hash → useful for deduplication without revealing value  
  - Log redaction stats (types, counts) — **never** log original value

- **Policy-based Human-in-the-Loop (HITL)**  
  - Annotate tools with risk tier (`low` / `medium` / `high` / `critical`)  
  - High/critical tools → require explicit approval (Matrix reaction / admin endpoint)  
  - Optional **sudo mode** with short TTL (15 min) + mandatory audit trail

**Failure modes**

- Untrusted sender → immediate structured rejection (logged)
- PII detected but policy=block → reject with user-facing explanation
- HITL timeout → fail closed

### Phase 2 – Financial & Abuse Guardrails (Weeks 3–4.5)

**Core controls**

- **Multi-granularity budgets**  
  - Per-session (short-lived agent run)  
  - Per-user / per-room (daily & rolling 30-day)  
  - Global hard ceiling (emergency kill switch)

- **Enforcement points**  
  - Pre-flight estimate (rough token prediction) → reject if over budget  
  - Running total tracked in Redis / SQLite (reconciled on restart)  
  - Hard stop at 100% (default), soft throttle at 80%

- **Model-aware routing & caching hints**  
  - Prefer cheaper/faster models for non-critical paths (configurable)  
  - Semantic cache awareness (avoid redundant expensive calls)

- **Alerting tiers**  
  - 70% → info  
  - 85% → warning (admin room)  
  - 95% → critical + auto-pause new sessions

**Defaults (adjustable)**

- Session max: ~$0.50  
- User daily: $4–7  
- Global monthly: $80–150 (very conservative starting point)

### Phase 3 – Runtime Containment & Host Hardening (Weeks 5–6.5)

**Container-focused controls**

- **Short container TTL**  
  - Idle timeout: **default 8 min** (configurable 2–30 min)  
  - Periodic heartbeat → kill on failure  
  - Force-terminate on budget violation or policy breach

- **Ephemeral filesystem & secrets**  
  - Secrets mounted via tmpfs / memory only  
  - Never written to disk (even temporarily)  
  - Unlink immediately after read

- **Minimal privileges**  
  - Drop all unnecessary Linux capabilities  
  - Read-only root fs where possible  
  - Non-root user by default

- **Host-level basics** (single-node assumption)  
  - UFW / nftables deny-all + explicit allow for required ports  
  - SSH → key-only, disable password & root login  
  - **VPN access (Tailscale / WireGuard / similar) is optional**  
    - Strongly recommended for production / chargeable accounts (paid Tailscale plan or self-hosted WireGuard)  
    - Provides encrypted admin access and eliminates open SSH ports  
    - **Not required** — teams can continue using direct SSH key access or cloud provider bastion / console access  
    - If Tailscale is used on a paid plan, costs are the responsibility of the infrastructure owner (not bundled into ArmorClaw)

**Skip / warn** on managed K8s / ECS / Fly.io (they provide their own controls)

### Phase 4 – Observability & Memory Defense (Weeks 7–9)

**Key guarantees**

- Secrets lifetime ≤ execution duration + short grace period
- No swap usage (disable or aggressively limit)
- Crash dumps routed to secure, short-lived location or disabled
- Runtime monitoring hooks (optional Falco / tracee / eBPF if footprint allows)

**Mandatory logging & metrics**

- Every rejection / redaction / budget hit / container kill  
- Structured format (JSON) + correlation ID (Matrix event ID when possible)  
- Export to Prometheus / Loki / OTLP (configurable sink)

## 5. Configuration Surface (minimal & opinionated)

```yaml
zero_trust:
  trusted_mxids: ["@admin:example.com", …]
  trusted_rooms: ["!roomid:example.com"]
  pii_strategy: "synthetic"       # redact | synthetic | hash | block
  hitl:
    default_policy: "low"
    high_risk_tools: ["execute_shell", "send_money"]

budget:
  session_max_usd: 0.60
  user_daily_usd: 5.00
  global_monthly_usd: 120.00
  hard_stop: true
  alert_at: 0.80

containment:
  idle_timeout_minutes: 8
  check_every_seconds: 30
  secrets_mode: "memory_only"
  # vpn: optional – not enforced by ArmorClaw
```

## 6. Success Criteria (measurable)
- 100% of untrusted senders blocked by default
- PII redaction / synthetic swap applied before ≥ 95% of LLM calls
- Budget violations trigger hard stop (no overspend > 5%)
- Idle containers reliably terminated (< 1% leak past TTL)
- Zero intentional secret writes to disk (verified via audit)
- All critical events (rejection, kill, budget hit) observable in logs/metrics

7. Rollback & Progressive Rollout
Kill switches (all disabled by default):
- zero_trust.enabled = false
- budget.hard_stop = false
- containment.idle_timeout_minutes = 0
- pii_strategy = "noop"

**Recommended rollout order**
- Observability + structured logging
- PII sanitization (synthetic preferred)
- Trusted sender checks + soft budget alerts
- Hard budget enforcement + HITL on high-risk tools
- Container TTL & host firewall

8. Next Actions
    1. Approve / adjust defaults (especially budget numbers)
    2. Create GitHub milestone per phase + one tracking issue
    3. Start Phase 1 (zero-trust + PII) — highest risk reduction per effort
    4. Draft SECURITY.md with threat model & controls overview
    5. Schedule security walkthrough after Phase 2