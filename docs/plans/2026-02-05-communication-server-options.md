# Communication Server Options for ArmorClaw Integration

**Date:** 2026-02-05
**Status:** Planning
**Constraint:** ≤ 400 MB RAM for communication server
**Total Budget:** ≤ 2 GB (Ubuntu + Nginx + Agent + Server)

---

## Overview

This document evaluates communication server options that integrate with ArmorClaw's containment architecture while staying within strict memory constraints.

---

## Option 1: Matrix Conduit (RECOMMENDED)

### Specifications

| Attribute | Value |
|-----------|-------|
| **Language** | Rust |
| **Idle RAM** | 32-50 MB |
| **Active RAM** | 100-200 MB (small team) |
| **Database** | Embedded SQLite |
| **Encryption** | E2EE by default (Olm/Megolm) |
| **Maturity** | Production-ready |
| **Mobile Support** | Full (Element, SchildiChat) |

### Pros
- ✅ End-to-end encryption built-in
- ✅ Federated (can connect to other Matrix servers)
- ✅ Rich client ecosystem (Element, etc.)
- ✅ Voice/Video via WebRTC
- ✅ File sharing, reactions, threads
- ✅ Well-documented API
- ✅ Room-based organization (natural for agent channels)

### Cons
- ⚠️ Slightly more complex than XMPP
- ⚠️ Federation adds attack surface (can be disabled)

### Memory Profile
```
Idle (no users):    ~32 MB
1-5 active users:   ~80-150 MB
5-10 active users:  ~150-250 MB
10+ users:          ~250-400 MB (approaches limit)
```

### Docker Compose Configuration
```yaml
services:
  matrix-conduit:
    image: matrixconduit/matrix-conduit:latest
    restart: unless-stopped
    environment:
      CONDUIT_SERVER_NAME: "matrix.armorclaw.com"
      CONDUIT_ALLOW_REGISTRATION: "false"
      CONDUIT_ALLOW_ENCRYPTION: "true"
      CONDUIT_DATABASE_PATH: /var/lib/matrix-conduit/conduit.db
      CONDUIT_MAX_REQUEST_SIZE: "10485760"  # 10MB
    volumes:
      - ./conduit_data:/var/lib/matrix-conduit/
    ports:
      - "6167:6167"   # Client API
      - "8448:8448"   # Federation (optional)
    deploy:
      resources:
        limits:
          memory: 400M
        reservations:
          memory: 100M
```

### Installation Commands
```bash
# Create directory
mkdir -p /opt/armorclaw/matrix
cd /opt/armorclaw/matrix

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
# (See configuration above)
EOF

# Create data directory
mkdir -p conduit_data

# Start service
docker-compose up -d

# Verify memory usage
docker stats matrix-conduit
```

---

## Option 2: Prosody XMPP (ALTERNATIVE)

### Specifications

| Attribute | Value |
|-----------|-------|
| **Language** | Lua |
| **Idle RAM** | 20-40 MB |
| **Active RAM** | 50-100 MB (small team) |
| **Database** | Embedded (can use SQLite) |
| **Encryption** | TLS + OMEMO (E2EE plugin) |
| **Maturity** | Very mature (20+ years) |
| **Mobile Support** | Full (Conversations, Monocles) |

### Pros
- ✅ Extremely lightweight
- ✅ Very simple to configure
- ✅ XMPP is an IETF standard
- ✅ Native to many developer tools
- ✅ Agent-friendly (JID structure)
- ✅ Low resource usage even under load

### Cons
- ⚠️ E2EE requires plugin (OMEMO)
- ⚠️ Less modern UI/UX than Matrix
- ⚠️ No built-in federation (optional)
- ⚠️ Smaller client ecosystem

### Memory Profile
```
Idle (no users):    ~20 MB
1-5 active users:   ~40-70 MB
5-10 active users:  ~70-120 MB
10+ users:          ~120-200 MB
```

### Configuration
```lua
-- /etc/prosody/conf.d/armorclaw.cfg.lua
VirtualHost "armorclaw.com"
    -- Enabled modules
    modules_enabled = {
        "roster"; "saslauth"; "tls"; "dialback";
        "disco"; "carbons"; "pep"; "private"; "vcard";
        "version"; "uptime"; "time"; "ping"; "admin_adhoc";
        "bookmarks"; "private"; "register";
    }

    -- Encryption
    c2s_require_encryption = true
    s2s_require_encryption = true
    s2s_secure_auth = true

    -- Authentication
    authentication = "internal_hashed"

    -- Limits
    max_presence_sessions = 50
    max_message_size = 10240  -- 10KB
```

---

## Option 3: Custom WebSocket Server (MINIMAL)

### Specifications

| Attribute | Value |
|-----------|-------|
| **Language** | Go |
| **Idle RAM** | 10-20 MB |
| **Active RAM** | 30-80 MB |
| **Database** | Optional (can be stateless) |
| **Encryption** | TLS + application-layer |
| **Maturity** | Custom implementation |
| **Mobile Support** | Custom client required |

### Pros
- ✅ Absolute minimal footprint
- ✅ Complete control over protocol
- ✅ Can integrate directly with Local Bridge
- ✅ Single binary deployment

### Cons
- ❌ No federation
- ❌ Custom client required (or web-only)
- ❌ Must implement encryption from scratch
- ❌ No existing ecosystem
- ❌ Higher development effort

### Memory Profile
```
Idle (no users):    ~10 MB
1-5 active users:   ~30-50 MB
5-10 active users:  ~50-80 MB
10+ users:          ~80-150 MB
```

### Go Implementation (Conceptual)
```go
// Lightweight WebSocket broker for ArmorClaw
package main

import (
    "github.com/gorilla/websocket"
    "crypto/tls"
    "net/http"
)

type Message struct {
    Type    string          `json:"type"`
    From    string          `json:"from"`
    To      string          `json:"to"`
    Payload json.RawMessage `json:"payload"`
    Time    int64           `json:"time"`
}

func main() {
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
    }

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, _ := upgrader.Upgrade(w, r, nil)
        // Handle agent messages
        broker.Register(conn)
    })

    http.ListenAndServeTLS(":8443", "cert.pem", "key.pem", nil)
}
```

---

## Comparison Table

| Feature | Matrix Conduit | Prosody XMPP | Custom WebSocket |
|---------|----------------|--------------|------------------|
| **Memory (idle)** | 32 MB | 20 MB | 10 MB |
| **Memory (active)** | 100-200 MB | 50-100 MB | 30-80 MB |
| **E2EE** | Built-in | Plugin required | Custom |
| **Setup Complexity** | Medium | Low | High (build) |
| **Client Support** | Excellent | Good | None (custom) |
| **Federation** | Yes | Optional | No |
| **Voice/Video** | Yes (WebRTC) | Yes (Jingle) | Custom |
| **Agent Integration** | Via skill | Via skill | Native |
| **Maturity** | Production | Very mature | New |
| **Mobile Apps** | Many | Several | None |

---

## Recommendation

### **Primary Recommendation: Matrix Conduit**

**Rationale:**
1. Memory footprint stays well within budget (< 200 MB for small teams)
2. End-to-end encryption is critical for agent communications
3. Rich client ecosystem (Element, SchildiChat)
4. Room-based model fits agent supervision workflows
5. Federation can be disabled if needed
6. Active development and security audits

### **Fallback: Prosody XMPP**

Use if:
- Absolute minimal memory is critical
- Simplicity of setup is preferred
- XMPP is already familiar to the team

### **Avoid: Custom WebSocket**

Unless:
- The team has Go development resources
- Only basic messaging is needed
- Long-term maintenance can be sustained

---

## Integration with ArmorClaw

### Architecture (All Options)

```
┌─────────────────────────────────────────────────────────┐
│  Hostinger KVM2 (≤ 2 GB RAM)                            │
│                                                          │
│  ┌──────────────┐  ┌──────────────────────────────┐     │
│  │ Comm Server  │  │ Nginx (SSL Termination)      │     │
│  │ (Conduit/    │◄─┤ Port 443 → 6167              │     │
│  │  Prosody)    │  └──────────────────────────────┘     │
│  └──────┬───────┘                                     │
│         │ WebSocket / XMPP stream                      │
│         ▼                                              │
│  ┌──────────────────────────────┐                      │
│  │ Local Bridge (Go)             │                      │
│  │ /run/armorclaw/bridge.sock   │                      │
│  └──────────────▲───────────────┘                      │
│                 │ JSON-RPC 2.0                         │
│  ┌──────────────┴───────────────┐                      │
│  │ ArmorClaw Container         │                      │
│  │ ┌────────────────────────┐  │                      │
│  │ │ OpenClaw Agent         │  │                      │
│  │ │ + Comm Skill           │  │                      │
│  │ │ (Matrix/XMPP/WebSocket)│  │                      │
│  │ └────────────────────────┘  │                      │
│  │ • UID 10001                 │                      │
│  │ • Secrets in RAM only       │                      │
│  └─────────────────────────────┘                      │
│                                                          │
│  ✅ Total: ≤ 2 GB                                       │
└─────────────────────────────────────────────────────────┘
```

### Agent Communication Flow

1. **User sends message** via Element client → Matrix server
2. **Matrix server pushes** to agent's room
3. **OpenClaw skill** (in container) polls/receives message
4. **Skill invokes** agent via Local Bridge
5. **Agent responds** → Bridge → Skill → Matrix server
6. **User receives** response in Element

---

## Implementation Checklist

### Pre-Deployment
- [ ] Confirm Hostinger KVM2 specifications (4 GB RAM, adequate disk)
- [ ] Reserve domain (e.g., `matrix.armorclaw.com`)
- [ ] Configure DNS A record → Hostinger IP

### System Configuration
- [ ] Install minimal Ubuntu Server (no GUI)
- [ ] Configure swap (2 GB) for memory safety
- [ ] Set up UFW firewall rules
- [ ] Configure automatic security updates

### Comm Server Deployment
- [ ] Deploy chosen server (Conduit recommended)
- [ ] Configure SSL certificates (Certbot)
- [ ] Set up Nginx reverse proxy
- [ ] Disable registration after team onboarding
- [ ] Create admin account and secure credentials

### ArmorClaw Integration
- [ ] Complete Local Bridge implementation
- [ ] Deploy ArmorClaw container
- [ ] Install communication skill in container
- [ ] Configure agent Matrix/XMPP account
- [ ] Test end-to-end messaging flow

### Verification
- [ ] Verify total RAM ≤ 2 GB under load
- [ ] Test E2EE encryption
- [ ] Verify agent cannot escape container
- [ ] Confirm secrets are memory-only
- [ ] Load test with multiple users

---

## Resource Optimization Tips

### Ubuntu Base
```bash
# Install minimal Ubuntu Server ISO
# Disable unneeded services
sudo systemctl disable snapd
sudo systemctl disable bluetooth
sudo systemctl disable cups

# Use zram for compressed swap
sudo apt install zram-config
```

### Nginx Optimization
```nginx
# /etc/nginx/nginx.conf
worker_processes 1;
worker_connections 256;
keepalive_timeout 30;
client_body_buffer_size 8k;
```

### Docker Limits
```yaml
# In docker-compose.yml
deploy:
  resources:
    limits:
      memory: 400M
    reservations:
      memory: 50M
```

---

## Monitoring Commands

```bash
# Real-time memory usage
htop

# Docker container stats
docker stats

# System-wide summary
free -h

# Process breakdown
ps aux --sort=-%mem | head -20

# Alert if approaching 2 GB
watch -n 10 'free -m | awk "NR==2{printf \"Memory: %.0f/%.0f MB (%.1f%%)\n\", \
  \$3, \$2, \$3*100/\$2}"'
```

---

**Next Step:** Confirm which communication server option to proceed with, then I'll create the detailed deployment guide.
