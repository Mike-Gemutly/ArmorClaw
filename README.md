# ArmorClaw: The VPS Secretary Platform

[![Version](https://img.shields.io/badge/version-v4.1.3-blue)](https://github.com/armorclaw/armorclaw)
[![Status](https://img.shields.io/badge/status-production%20ready-green)](https://github.com/armorclaw/armorclaw)

**Autonomous agents working 24/7 on your VPS, controlled from your pocket.**

ArmorClaw is a Zero-Trust orchestration layer that runs AI agents (OpenClaw) on a remote VPS. These agents act as "Digital Secretaries"—browsing websites, filling forms, and managing tasks—while you stay mobile. You approve sensitive actions via your phone; the agent does the work.

---

## Key Features

*   **VPS-Based "Secretary":** Agents run headless on your server, performing desktop-class tasks (booking flights, paying bills, legal filing) while you are away from your computer.
*   **Mobile-First Control:** Use **ArmorChat** (Android) to monitor status, review screenshots, and approve PII usage via biometrics.
    *   End-to-End Encryption via Matrix SDK
    *   Push Notifications (FCM)
    *   Key Backup/Recovery (SSSS passphrase)
    *   Bridge Verification (emoji verification)
*   **No-Code Agent Studio:** Define new agents (e.g., "Contract Reviewer," "Travel Booker") via Matrix chat or Dashboard—no coding required.
*   **BlindFill™ Security:** Agents request sensitive data (SSN, Credit Cards) via references (`value_ref`). The Bridge decrypts data in memory and injects it directly into the browser. The agent never sees the raw value, and it never travels over the network.
*   **Secure Browser Automation:** Remote control of a headless browser via the `com.armorclaw.browser.*` Matrix protocol.

---

## Architecture

```
┌───────────────────────────────────────────────────────────────────────┐
│                         THE VPS (Office)                              │
│                                                                       │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐           │
│  │ ArmorClaw   │◀────▶│  OpenClaw   │◀────▶│  Playwright │           │
│  │ Bridge      │      │  (Worker)   │      │  (Browser)  │           │
│  │ (Orchestr.) │      │             │      │             │           │
│  └──────┬──────┘      └──────┬──────┘      └──────┬──────┘           │
│         │                    │                     │                   │
│         │                    │                     │                   │
│         │   BlindFill Engine │                     │                   │
│         │   (Memory-Only)    │                     │                   │
│         │                    │                     │                   │
└─────────┼────────────────────┼─────────────────────┼───────────────────┘
          │                    │                     │
          │ Secure Matrix Tunnel (E2EE)             │
          │                    │                     │
┌─────────▼────────────────────▼─────────────────────▼───────────────────┐
│                         USER (Mobile)                                 │
│                                                                       │
│   ArmorChat App                                                      │
│   ┌─────────────────────────────────────────────────────────────┐     │
│   │  "Book a flight to NYC"                                     │     │
│   │  [Approve Credit Card] 🔐                                   │     │
│   │  ⏳ Status: Filling form...                                 │     │
│   └─────────────────────────────────────────────────────────────┘     │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

---

## Getting Started

### Prerequisites
*   Docker & Docker Compose V2
*   A VPS (Recommended) or local server

### 1. Quick Start (Recommended)

```bash
docker run -it --name armorclaw \
  --restart unless-stopped \
  --user root \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-data:/etc/armorclaw \
  -v armorclaw-keystore:/var/lib/armorclaw \
  -p 8443:8443 \
  -p 6167:6167 \
  -p 5000:5000 \
  mikegemut/armorclaw:latest
```

### 2. Build from Source (Full Stack)

```bash
# Clone the repository
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Build and run the complete stack (Bridge + Matrix + Sygnal + Caddy)
docker compose -f docker-compose-full.yml up -d --build

# Or build just the quickstart image
docker build -t armorclaw/quickstart:latest -f Dockerfile.quickstart .
```

**Note:** The bridge requires Debian-based images (not Alpine) for SQLCipher compatibility.

### 3. Connect via ArmorChat
1.  Open the ArmorChat Android app.
2.  Scan the QR code displayed in the container logs.
3.  Set up your biometric key for the keystore.

### 4. Create Your First Agent (No-Code)
In the Matrix room, type:
```
!agent create name="Travel Booker" skills="web_browsing, form_filling"
```
The Bridge will provision the agent and invite you to its dedicated room.

---

## Browser Automation Protocol

ArmorClaw exposes a Matrix-native API for controlling the browser.

**Example: Secure Form Fill**
The client sends references to sensitive data, never the data itself.

```json
{
  "type": "com.armorclaw.browser.fill",
  "content": {
    "fields": [
      { "selector": "#email", "value": "user@example.com" },
      { "selector": "#credit-card", "value_ref": "payment.card_number" }
    ]
  }
}
```
*   **Bridge Action:** Receives request -> Pauses -> Emits `PII_REQUEST` to user -> User Approves -> Decrypts `payment.card_number` -> Injects into Browser.

---

## Use Case: The "Legal Team"

You can run a fleet of specialized agents on a single VPS.

| Agent | Skills | PII Access | Task |
| :--- | :--- | :--- | :--- |
| **Contracts** | `pdf_gen`, `template` | Client Name, Contract Value | Drafts service agreements. |
| **Case Work** | `evidence`, `timeline` | Client SSN, Medical Records | Organizes case evidence. |
| **Motions** | `law_search`, `brief_write` | Client Name | Drafts court motions. |

**Result:** Each agent is isolated in its own container. The "Contracts" agent cannot access "Medical Records" because the Bridge enforces PII scopes per agent.

---

## Security Model

| Layer | Protection |
| :--- | :--- |
| **Network** | All traffic over Matrix (E2EE). |
| **Memory** | Secrets decrypted only during execution, wiped immediately. |
| **Storage** | Keystore encrypted with SQLCipher + User-Held Key. |
| **Audit** | Every PII access and transaction logged immutably. |

---

## Status

*   **Core Bridge:** ✅ Stable
*   **BlindFill Engine:** ✅ Complete
*   **Agent State Machine:** ✅ Complete
*   **Browser RPC:** ✅ Complete
*   **ArmorChat Android:** ✅ Feature Complete
*   **Docker Deployment:** ✅ Hardened (v4.1.3)
*   **No-Code Agent Studio:** ✅ Complete
*   **MCP Approval Workflow:** ✅ Complete

---

## Documentation

*   **Architecture:** [docs/plans/2026-02-05-armorclaw-v1-design.md](docs/plans/2026-02-05-armorclaw-v1-design.md)
*   **Setup Guide:** [docs/guides/setup-guide.md](docs/guides/setup-guide.md)
*   **Configuration:** [docs/guides/configuration.md](docs/guides/configuration.md)
*   **RPC API Reference:** [docs/reference/rpc-api.md](docs/reference/rpc-api.md)
*   **Troubleshooting:** [docs/guides/troubleshooting.md](docs/guides/troubleshooting.md)
*   **Full Documentation Index:** [docs/index.md](docs/index.md)

## Links

*   **GitHub:** https://github.com/armorclaw/armorclaw
*   **Docker Hub:** https://hub.docker.com/r/mikegemut/armorclaw

## License

[MIT License](LICENSE)