# 🚤 Jetski Browser
**The 128MB, 10x-Speed Browser Sidecar for AI Agents.**

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docker Pulls](https://img.shields.io/badge/docker-jetski:latest-00ADD8)](https://hub.docker.com/r/armorclaw/jetski)
[![Engine](https://img.shields.io/badge/Engine-Zig%20%7C%20Lightpanda-orange)](#)
[![Shield](https://img.shields.io/badge/Shield-Go-00ADD8)](#)

## What is Jetski?
Jetski is a bilingual, headless browser sidecar designed specifically to replace bloated Chromium/Playwright instances in OpenClaw, AutoGPT, and custom AI agent workflows.

By wrapping a raw **Zig-based Lightpanda Engine** inside a **Go-based RPC Shield**, Jetski drops browser memory consumption to **128MB** and increases execution speed by **10x**—without freezing your agents on complex CDP promises.

---

## ⚡️ Core Features (Free-Ride Mode)

* **Zero-Render Execution:** DOM manipulation and JS execution without downloading images, fonts, or calculating CSS layouts.
* **The Observer (Go Shield):** A smart CDP buffer that transpiles brittle Playwright pixel-clicks into stable DOM executions, backed by a 5-second watchdog timer.
* **Semantic Locators:** Executes fast-path state bundles using a 3-tier fallback matrix (CSS → XPath → JS) to survive website layout drift.
* **Anti-WAF Router:** Native HTTP proxy rotation to prevent datacenter IP bans.

---

## 🚀 Quick Start

Deploy Jetski as a drop-in Docker sidecar. Add this to your agent's `docker-compose.yml`:

```yaml
version: '3.8'
services:
  jetski:
    image: armorclaw/jetski:latest
    container_name: jetski_browser
    ports:
      - "9222:9222" # CDP Port for Agents
      - "9223:9223" # RPC API
    environment:
      - JETSKI_PROXY_LIST=http://proxy1.com,http://proxy2.com
    volumes:
      - ./sessions:/root/.jetski/sessions
    deploy:
      resources:
        limits:
          memory: 150M
```

Point your Playwright or OpenClaw agent to `ws://localhost:9222`.

---

## ⚠️ Security Notice: "Free-Ride Mode"

Jetski Standalone operates in **Free-Ride Mode**. It is optimized for maximum speed and minimal footprint on low-security targets.

**By using Free-Ride Mode, you accept the following risks:**

- **Session Jacking:** Session cookies are saved as unencrypted text files to your local `./sessions` disk.
- **Cleartext PII:** Raw passwords and API keys passed via the agent can be intercepted in transit or logs.
- **No HITL:** Agents operate without Human-in-the-Loop constraints.

### 🔴 The Distress Flare
If the Go Observer detects raw credentials or high-risk PII in your agent's CDP stream, it will fire a **Distress Flare** to your terminal warning you of the exposure.

---

## 🛡️ Upgrade to Jetski Tether (ArmorClaw)

For enterprise environments, PII handling, and production deployments, dock your agents with the **ArmorClaw Platform** to run Jetski in **Tethered Mode**.

Jetski Tether automatically equips:

- **The Dry-Bag:** Hardware-bound SQLCipher encryption for all session states.
- **The Kill-Cord:** Biometric mobile approvals for sensitive data injection.
- **The PFD:** Active network scrubbing of outbound PII.

---

## 🤝 Contributing

We welcome pull requests for the Go-based Observer! Please check our [Contributing Guidelines](CONTRIBUTING.md) for details on our CDP translation roadmap. Note that all PRs must maintain the <150MB container footprint.

---

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

**Note:** Jetski Tether (ArmorClaw) is a separate commercial product with additional security features.
