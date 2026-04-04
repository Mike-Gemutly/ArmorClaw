# ArmorClaw

## Let Your AI Agent Handle Deployment

ArmorClaw comes with **built-in deployment skills** that let coding agents like **Claude Code**, **OpenCode**, **Cursor**, and **Crush** deploy and manage your VPS secretary platform — no manual SSH sessions, no copy-pasting commands, no configuration headaches.

### Why Use AI Agents for Deployment?

| Manual Deployment | AI Agent with Skills |
|-------------------|---------------------|
| Read 50+ page docs | Agent reads skills automatically |
| Copy-paste commands | Agent executes with confirmation |
| Debug SSH issues yourself | Agent validates and reports errors |
| Miss configuration steps | Agent follows verified checklists |
| Hours of trial-and-error | Minutes of guided deployment |

**The skills contain everything your AI agent needs:**
- Step-by-step deployment procedures
- Cross-platform commands (Linux, macOS, Windows)
- Error detection and recovery
- Security best practices built-in

### How It Works

```
You: "Deploy ArmorClaw to my VPS at 192.168.1.100 with domain myvps.example.com"
                                                                                
Agent: ┌─────────────────────────────────────────────────────────────────┐
       │ 1. Reading .skills/deploy.yaml...                              │
       │ 2. Detected platform: macOS                                    │
       │ 3. Validating SSH key at ~/.ssh/id_ed25519... ✓                │
       │ 4. Testing connection to 192.168.1.100... ✓                    │
       │                                                                │
       │ ⚡ Ready to deploy. This will:                                 │
       │    - Install Docker on your VPS                                │
       │    - Deploy ArmorClaw containers                               │
       │    - Configure HTTPS with Let's Encrypt                        │
       │                                                                │
       │ Proceed? [Yes/No]                                              │
       └─────────────────────────────────────────────────────────────────┘

You: Yes

Agent: Deploying... ✓
       Your ArmorClaw is live at https://myvps.example.com
```

### Quick Start

**1. Open your AI coding tool in this directory:**
```bash
cd /path/to/armorclaw-omo
```

**2. Tell your agent what you want:**
```
Deploy ArmorClaw to my VPS at 5.183.11.149 with domain armorclaw.example.com
```

**3. That's it.** Your agent will:
- Read the deployment skills
- Validate your environment
- Ask for confirmation before making changes
- Deploy and verify everything works
- Give you the connection URL

### Available Skills

| Skill | What It Does | Example Request |
|-------|--------------|-----------------|
| **deploy** | Full VPS deployment | "Deploy ArmorClaw to 192.168.1.100" |
| **status** | Health checks & diagnostics | "Check if my deployment is healthy" |
| **cloudflare** | HTTPS via Cloudflare | "Set up Cloudflare tunnel for my domain" |
| **provision** | Connect mobile app | "Generate QR code for ArmorChat" |

### Trust & Safety

Skills use **three automation levels** so you stay in control:

| Level | What Happens | Example |
|-------|--------------|---------|
| 🟢 **auto** | Agent executes immediately | Health checks, status queries |
| 🟡 **confirm** | Agent asks before acting | SSH connections, running installer |
| 🔵 **guide** | Agent shows you how | Creating Cloudflare tokens |

**Your agent will never:**
- Run dangerous commands without asking
- Delete data without confirmation
- Expose your API keys or credentials
- Skip security validations

### Example Conversations

#### Deploy to VPS
```
You: Deploy ArmorClaw to my VPS at 5.183.11.149 with SSH key ~/.ssh/vps_key

Agent: I'll deploy ArmorClaw to your VPS. Let me:
       1. Validate SSH connectivity
       2. Run the installer
       3. Verify services are running
       
       Ready to proceed?
```

#### Check Status
```
You: Check if my ArmorClaw deployment is healthy

Agent: Checking deployment status at 5.183.11.149...
       
       ✓ Docker daemon running
       ✓ ArmorClaw containers healthy (3/3)
       ✓ Matrix Conduit responding
       ✓ Bridge RPC accessible
       ✓ HTTPS certificate valid (expires in 89 days)
       
       All systems operational.
```

#### Setup HTTPS
```
You: My VPS is behind NAT. Set up Cloudflare tunnel for armorclaw.mydomain.com

Agent: I'll configure Cloudflare Tunnel for outbound-only access.
       
       You'll need a Cloudflare API token. Create one at:
       https://dash.cloudflare.com/profile/api-tokens
       
       Required permissions: Zone DNS Edit, Zone Cloudflare Tunnel Edit
       
       Once you have the token, say: "cloudflare token is xxxx"
```

### Deployment Modes

Skills automatically handle all deployment scenarios:

| Mode | When to Use | Agent Will Configure |
|------|-------------|---------------------|
| **Native** | Local development, testing | Unix sockets, no TLS |
| **Sentinel** | Production VPS with domain | Let's Encrypt HTTPS |
| **Cloudflare Tunnel** | VPS behind NAT/firewall | Outbound tunnel, no ports |
| **Cloudflare Proxy** | Existing Cloudflare setup | CDN, DDoS protection |

Just tell your agent your situation — it picks the right mode.

### Skills Directory Structure

```
.skills/
├── deploy.yaml          # Deployment skill definition
├── deploy/SKILL.md      # AI-friendly instructions
├── status.yaml          # Status check skill
├── status/SKILL.md      # Status documentation
├── cloudflare.yaml      # Cloudflare setup skill
├── cloudflare/SKILL.md  # Cloudflare guide
├── provision.yaml       # Mobile provisioning skill
├── provision/SKILL.md   # Provisioning guide
├── PLATFORM.md          # Cross-platform patterns
├── TEMPLATE.yaml        # Schema for new skills
└── README.md            # Skills index
```

Your AI agent reads these files to understand how to help you.

### Supported AI Tools

Skills work with any AI coding assistant that can read files:

| Tool | How to Use |
|------|------------|
| **Claude Code** | Just describe what you want in natural language |
| **OpenCode** | Same — natural language commands |
| **Cursor** | Works in chat or composer mode |
| **Crush** | Direct skill invocation |
| **Any AI with file access** | Point to `.skills/` directory |

### Why ArmorClaw Skills?

**Built by experts, executed by AI:**
- Skills encode deployment best practices
- Tested across Linux, macOS, Windows
- Security validations built into every step
- Error messages explain what went wrong

**You focus on what, agent handles how:**
- No memorizing commands
- No debugging shell scripts
- No reading deployment docs
- Just describe your goal

### Get Started Now

```bash
# 1. Clone the repo
git clone https://github.com/Gemutly/ArmorClaw.git
cd ArmorClaw

# 2. Open your AI coding tool

# 3. Say:
"Deploy ArmorClaw to my VPS at [YOUR_VPS_IP]"
```

Your AI agent will guide you through the rest.

---

## For AI Agents Reading This

If you're an AI agent, here's how to use these skills:

1. **Read skill definitions**: `.skills/*.yaml` files contain parameters, steps, and automation levels
2. **Read skill instructions**: `.skills/*/SKILL.md` files contain detailed guidance
3. **Use shell syntax**: All variables use `${variable}` format
4. **Respect automation levels**: Ask before `confirm` steps, guide users through `guide` steps
5. **Cross-platform support**: Detect OS and use appropriate commands

**Example skill invocation:**
```
User: "Deploy ArmorClaw to 192.168.1.100 with domain example.com"

Agent actions:
1. Read .skills/deploy.yaml
2. Set vps_ip=192.168.1.100, domain=example.com
3. Execute steps in order, respecting automation levels
4. Report results to user
```

---

**Full documentation**: [README.md](README.md) | **Architecture**: [doc/armorclaw.md](doc/armorclaw.md)
