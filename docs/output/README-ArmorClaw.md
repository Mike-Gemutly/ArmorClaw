# ArmorClaw 🦞🔒 — Your AI Agent Safe Room

**ArmorClaw** is a **secure guest room** for powerful AI agents.
It lets them think, search, write, code, and help — while **strictly preventing** them from touching your personal files, stealing secrets, or escaping to the host machine.

You get the full capabilities of OpenClaw (and compatible agents) — just safely contained.

## 🏠 Who Uses ArmorClaw?

- **Professionals** — Draft emails, analyze reports, brainstorm ideas without risking company data
- **Families & Students** — Help with homework, research, creative projects — safely
- **Security-conscious teams** — Experiment with AI while supporting compliance (GDPR, HIPAA, SOC 2, etc.)
- **DevOps, marketing, researchers, compliance officers** — Automate tasks in a controlled environment
- **Anyone** who wants a powerful local AI without the usual security trade-offs

## Why You Need a Safe Room

Standard local AI setups give agents **too much freedom**:

**Standard (Risky)**
- Runs directly on your computer with full access
- Stores API keys in plaintext `.env` files on disk
- Exposes localhost ports that can be abused

**ArmorClaw (Safe)**
- AI lives in a hardened Docker container — not your laptop
- Secrets injected into memory only — vanish on shutdown
- Strict isolation: no inbound ports, no Docker socket exposure
- All visibility is **pull-based** through a signed Local Bridge

## What ArmorClaw Does (and Doesn't Do)

✅ **ArmorClaw DOES:**
- Isolate the agent from your host machine
- Inject secrets ephemerally (memory-only)
- Provide controlled, read-only visibility

❌ **ArmorClaw DOES NOT:**
- Restrict agent capabilities or censor prompts
- Validate agent outputs for safety
- Provide fine-grained usage policies (planned)
- Guarantee 100% containment of unknown exploits

We contain powerful agents — we do not tame them.

## ⚖️ Security Comparison

| Area | Standard Setup | ArmorClaw (What it means for you) |
|------|----------------|------------------------------------|
| Execution | Full host privileges | Locked container (non-root) |
| Filesystem | Can read/write your entire disk | No host access by default; optional read-only workspace |
| Secrets | Plaintext `.env` files on disk | Memory-only — no trace left behind |
| Network | Open localhost ports | Internal-only, outbound-only where allowed |
| Client Control | Direct access to agent | Read-only snapshots & validated updates via signed bridge |
| Escape Risk | High if prompt or tool is exploited | Strongly contained — even successful exploits are trapped |


**Local Bridge** — minimal signed native binary: audited command surface, prevents Docker socket abuse, schema-validated messages.

## 🚀 Get Started in ~60 Seconds

**One-Command Install:**
```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy.sh | bash
```

**Or deploy manually:**
```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
./deploy/deploy-all.sh
```

## Compliance Alignment

ArmorClaw's model supports requirements like:
- **Data Isolation**: Container boundaries prevent lateral movement
- **Secret Management**: Ephemeral injection (no persistence)
- **Audit Surface**: Narrow, logged interaction via Local Bridge

Planned: Full audit logging, RBAC policy engine, enterprise secret integrations (HashiCorp Vault, etc.)

---

[MIT License](LICENSE) • [Security Reports](SECURITY.md) • [GitHub Issues](https://github.com/armorclaw/armorclaw/issues)
