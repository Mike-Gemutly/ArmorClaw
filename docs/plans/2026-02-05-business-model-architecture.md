# ArmorClaw Business Model Architecture

**Date:** 2026-02-05
**Purpose:** Define product tiers and corresponding architecture

---

## Product Tiers

### Tier 1: Community (Free)

```
┌─────────────────────────────────────────────────────────────┐
│  SELF-HOSTED (User installs on their own server)           │
│                                                             │
│  Provided:                                                  │
│  ✅ ArmorClaw container image                             │
│  ✅ Minimal Local Bridge source code                        │
│  ✅ Matrix Conduit setup guide                             │
│  ✅ Documentation                                          │
│  ✅ Community support (GitHub, Discord)                     │
│                                                             │
│  User brings:                                              │
│  • Own server (VPS, bare metal)                            │
│  • Own Matrix homeserver (or use our guide)                │
│  • Own OpenAI/Anthropic API keys                           │
│                                                             │
│  Limitations:                                               │
│  • DIY setup                                               │
│  • Community support only                                  │
│  • Basic features only                                     │
└─────────────────────────────────────────────────────────────┘
```

**Target:** Developers, hobbyists, technical teams

---

### Tier 2: Pro (Paid - $9-29/mo)

```
┌─────────────────────────────────────────────────────────────┐
│  MANAGED MATRIX INFRASTRUCTURE                              │
│                                                             │
│  Provided:                                                  │
│  ✅ Everything in Community                                │
│  ✅ Hosted Matrix homeserver (matrix.armorclaw.com)       │
│  ✅ Pre-configured Local Bridge binary                     │
│  ✅ One-line installer                                     │
│  ✅ Email support                                          │
│  ✅ Priority updates                                       │
│  ✅ Agent marketplace (basic skills)                       │
│                                                             │
│  User brings:                                              │
│  • Own server for ArmorClaw container                     │
│  • Own AI provider API keys                                │
│                                                             │
│  Extra Features:                                           │
│  • Managed Matrix backups                                  │
│  • Custom agent domains (@agent.yourdomain.com)            │
│  • Basic analytics dashboard                               │
└─────────────────────────────────────────────────────────────┘
```

**Target:** Small teams, businesses needing reliable Matrix

---

### Tier 3: Enterprise (Custom pricing)

```
┌─────────────────────────────────────────────────────────────┐
│  FULLY MANAGED                                              │
│                                                             │
│  Provided:                                                  │
│  ✅ Everything in Pro                                      │
│  ✅ Hosted ArmorClaw containers (our infra)               │
│  ✅ Secrets management (we inject keys)                    │
│  ✅ Custom agent skill development                         │
│  ✅ SLA guarantees                                         │
│  ✅ Dedicated support                                      │
│  ✅ On-premise deployment option                           │
│                                                             │
│  Extra Features:                                           │
│  • SSO integration (SAML, OIDC)                            │
│  • Audit logs & compliance reporting                       │
│  • Custom branding (white-label)                           │
│  • API access for automation                               │
│  • Advanced threat detection                               │
└─────────────────────────────────────────────────────────────┘
```

**Target:** Large orgs, regulated industries

---

## Architecture by Tier

### Community Tier Architecture

```
User's Infrastructure
┌─────────────────────────────────────────────────────────────┐
│  User's Server (AWS, DigitalOcean, etc.)                   │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │
│  │ Matrix       │  │ Local        │  │ ArmorClaw       │ │
│  │ Conduit      │  │ Bridge       │  │ Container        │ │
│  │ (Docker)     │  │ (built from  │  │ (Agent)          │ │
│  │              │  │  source)     │  │                  │ │
│  └──────────────┘  └──────────────┘  └──────────────────┘ │
│                                                             │
│  User's Element Apps → Matrix Conduit                       │
└─────────────────────────────────────────────────────────────┘
```

**Nothing hosted by ArmorClaw company.** User runs everything.

---

### Pro Tier Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     ARMORCLAW INFRASTRUCTURE                   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │           Shared Matrix Homeserver                        │  │
│  │           (matrix.armorclaw.com)                         │  │
│  │                                                            │  │
│  │  • Conduit with PostgreSQL                                │  │
│  │  • Nginx reverse proxy + SSL                              │  │
│  │  • Coturn for voice/video                                 │  │
│  │  • Nightly backups                                        │  │
│  └────────────────────┬───────────────────────────────────────┘  │
│                       │                                         │
└───────────────────────┼─────────────────────────────────────────┘
                        │ Internet (E2EE)
                        │
┌───────────────────────┴─────────────────────────────────────────┐
│                     USER'S INFRASTRUCTURE                       │
│                                                                  │
│  User's Server / Workstation                                    │
│  ┌──────────────────────┐  ┌─────────────────────────────────┐ │
│  │ Local Bridge         │  │ ArmorClaw Container            │ │
│  │ (our binary)         │  │ (Agent)                         │ │
│  │ • Connects to        │  │                                 │ │
│  │   matrix.armorclaw  │  │                                 │ │
│  └──────────────────────┘  └─────────────────────────────────┘ │
│                                                                  │
│  User's Element Apps → matrix.armorclaw.com                     │
└──────────────────────────────────────────────────────────────────┘
```

**ArmorClaw hosts:** Matrix infrastructure
**User hosts:** Local Bridge + Container

---

### Enterprise Tier Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  ARMORCLAW CLOUD (Fully Managed)               │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │           Customer's Dedicated Environment                │  │
│  │                                                            │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────┐  │  │
│  │  │ Matrix     │  │ Local      │  │ ArmorClaw         │  │  │
│  │  │ Conduit    │  │ Bridge     │  │ Containers (N)     │  │  │
│  │  │           │  │            │  │                    │  │  │
│  │  └────────────┘  └────────────┘  └────────────────────┘  │  │
│  │                                                            │  │
│  │  ┌─────────────────────────────────────────────────────┐ │  │
│  │  │           Management Dashboard (Web)                │ │  │
│  │  │  • Start/stop containers                            │ │  │
│  │  │  • Inject secrets (key management)                  │ │  │
│  │  │  • View logs & metrics                              │ │  │
│  │  │  • Manage agent skills                              │ │  │
│  │  └─────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Customer's Admins → Dashboard → Control everything              │
│  Customer's Users → Element Apps → Chat with agents              │
└──────────────────────────────────────────────────────────────────┘

OR (On-Premise Option)

┌─────────────────────────────────────────────────────────────────┐
│              CUSTOMER'S DATA CENTER (On-Premise)                │
│                                                                  │
│  Same stack, hosted on customer's infrastructure                │
│  • ArmorClaw provides deployment scripts                       │
│  • Customer pays for on-premise license + support               │
└──────────────────────────────────────────────────────────────────┘
```

---

## Minimal Bridge + Business Model: How It Works

### For Community (Free Tier)

The Minimal Bridge I designed is **perfect for this tier**:

1. User downloads bridge source code
2. Builds it themselves (`go build`)
3. Runs it alongside their container
4. **No involvement needed from ArmorClaw company**

**Value to user:** Free, self-contained, secure agent isolation
**Value to ArmorClaw:** Builds user base, converts to paid tiers

---

### For Pro Tier

The Minimal Bridge needs **slight enhancement**:

```toml
# /etc/armorclaw/bridge.toml (Pro tier config)

[matrix]
# Community: localhost
# Pro: matrix.armorclaw.com
homeserver_url = "https://matrix.armorclaw.com"

# Pro-only: Custom domain
customer_domain = "acme.corp"

[bridge]
# Pro-only: Telemetry (opt-in)
telemetry_enabled = true
telemetry_endpoint = "https://api.armorclaw.com/telemetry"
```

**Enhanced bridge for Pro:**
- Connects to managed Matrix homeserver
- Optional telemetry for product improvement
- Custom domain support (@agent.acme.corp)
- License validation

---

## Desktop/Mobile App Strategy

### Phase 1: Use Existing Matrix Clients (No Development)

```
┌─────────────────────────────────────────────────────────────┐
│  IMMEDIATE (No custom app needed)                           │
│                                                             │
│  Instructions to users:                                     │
│  1. Download Element (desktop/mobile)                       │
│  2. Set homeserver to:                                      │
│     • Community: your own Matrix                            │
│     • Pro: matrix.armorclaw.com                            │
│  3. Create account or log in                               │
│  4. Join room with your agent                              │
│                                                             │
│  ✅ Zero development cost                                   │
│  ✅ Works immediately                                      │
│  ✅ Leverages mature Matrix ecosystem                      │
└─────────────────────────────────────────────────────────────┘
```

### Phase 2: Branded "ArmorClaw Chat" App (Electron/React Native)

```
┌─────────────────────────────────────────────────────────────┐
│  ArmorClaw Chat (Desktop/Mobile)                           │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ Basically Element Web with:                             ││
│  │  • ArmorClaw branding                                  ││
│  │  • Pre-configured homeserver                            ││
│  │  • Agent discovery (list available agents)              ││
│  │  • One-click agent room creation                        ││
│  │  • Built-in help documentation                          ││
│  └─────────────────────────────────────────────────────────┘│
│                                                             │
│  Built using:                                               │
│  • matrix-js-sdk (already exists!)                          │
│  • Element Web as reference                                 │
│  • React Native for mobile                                  │
│                                                             │
│  Effort: ~4-6 weeks for MVP                                 │
└─────────────────────────────────────────────────────────────┘
```

### Phase 3: Full Management Dashboard (Web)

```
┌─────────────────────────────────────────────────────────────┐
│  ArmorClaw Dashboard (Web)                                 │
│                                                             │
│  For Pro/Enterprise tiers only                              │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐│
│  │ Agent           │  │ Secrets        │  │ Analytics     ││
│  │ Management      │  │ Injection      │  │ & Reports     ││
│  │                 │  │                │  │              ││
│  │ • Start/Stop    │  │ • Add keys     │  │ • Usage       ││
│  │ • View logs     │  │ • Rotate      │  │ • Costs       ││
│  │ • Config        │  │ • Audit trail │  │ • Uptime      ││
│  └─────────────────┘  └─────────────────┘  └──────────────┘│
│                                                             │
│  Requires: Management API (extend Local Bridge)             │
└─────────────────────────────────────────────────────────────┘
```

---

## Minimal Bridge + Custom Apps: Compatibility

**The Minimal Bridge as designed is fully compatible with custom apps:**

| Feature | Minimal Bridge | Custom App Support |
|---------|----------------|-------------------|
| **Agent chat via Matrix** | ✅ Core feature | ✅ App uses Matrix SDK |
| **Agent status** | ✅ Via Matrix presence | ✅ App displays presence |
| **Multiple agents** | ✅ One bridge, many containers | ✅ App lists all agents |
| **Secret injection** | ✅ Via FD passing | ⚠️ Needs Management API |
| **Container control** | ❌ Not in Minimal version | ⚠️ Needs Management API |

---

## Recommended Product Roadmap

### Phase 1: Foundation (Current - 2 months)

- [x] ArmorClaw container design
- [x] Minimal Bridge spec
- [ ] Minimal Bridge implementation
- [ ] Matrix Conduit deployment guide
- [ ] Basic documentation

**Deliverable:** Working Community tier

---

### Phase 2: Pro Infrastructure (Months 3-4)

- [ ] Set up matrix.armorclaw.com
- [ ] PostgreSQL-backed Conduit
- [ ] Nginx + SSL automation
- [ ] Enhanced bridge (remote Matrix support)
- [ ] One-line installer
- [ ] Subscription management

**Deliverable:** Pro tier launch

---

### Phase 3: User Experience (Months 5-6)

- [ ] Branded desktop app (Electron)
- [ ] Branded mobile app (React Native)
- [ ] Agent marketplace (basic)
- [ ] Premium agent skills

**Deliverable:** Branded experience, upsell opportunities

---

### Phase 4: Enterprise (Months 7-9)

- [ ] Management API
- [ ] Web dashboard
- [ ] SSO integration
- [ ] Audit logging
- [ ] On-premise deployment option

**Deliverable:** Enterprise tier

---

## Summary: Minimal Bridge + Business Model

| Question | Answer |
|----------|--------|
| **Does Minimal Bridge support desktop/mobile apps?** | ✅ Yes, via standard Matrix clients |
| **Can we sell branded apps?** | ✅ Yes, built on Matrix SDK |
| **Can we sell managed services?** | ✅ Yes, Pro/Enterprise tiers |
| **Does Minimal Bridge limit business model?** | ❌ No, it's the foundation |
| **What's missing for higher tiers?** | Management API (Phase 4) |

**The Minimal Bridge is the RIGHT choice for starting:**
1. Provides core security (credential isolation)
2. Enables Community tier (free, self-serve)
3. Doesn't block future tiers
4. Can be enhanced later with Management API

**Your business model works perfectly with this architecture.**
