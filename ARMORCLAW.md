# ArmorClaw

## Deployment Skills for AI CLI Tools

ArmorClaw provides AI-friendly deployment skills to help you deploy and manage your VPS secretary platform using your AI CLI tools (Claude Code, OpenCode, etc.).

### Quick Reference

| Skill | Purpose | Invocation |
|-------|---------|------------|
| **deploy** | Deploy ArmorClaw to VPS | `/deploy vps_ip=... ssh_key=...` |
| **status** | Check deployment health | `/status vps_ip=...` |
| **cloudflare** | Setup HTTPS/Cloudflare | `/cloudflare domain=...` |
| **provision** | Connect mobile device | `/provision vps_ip=...` |

### Skills Directory

All deployment skills are defined in `.skills/` at the project root:
- `.skills/deploy.yaml` + `deploy/SKILL.md` - VPS deployment
- `.skills/status.yaml` + `status/SKILL.md` - Health checking
- `.skills/cloudflare.yaml` + `cloudflare/SKILL.md` - Cloudflare setup
- `.skills/provision.yaml` + `provision/SKILL.md` - Mobile provisioning

Each skill includes:
- Structured YAML with parameters, steps, automation flags
- SKILL.md with AI-friendly instructions
- Cross-platform support (Linux, macOS, Windows/PowerShell)

### Using Skills

1. **Discover available skills**: Your AI CLI tool can read `.skills/` directory
2. **Invoke via commands**: Use `/deploy`, `/status`, `/cloudflare`, `/provision`
3. **Cross-platform**: Skills automatically detect your OS and provide appropriate commands

For detailed documentation, see individual skill SKILL.md files in `.skills/{skill-name}/`.

### Deployment Modes

Skills support all ArmorClaw deployment modes:
- **Native** (default) - Local-only, Unix sockets
- **Sentinel** - Production with Let's Encrypt TLS
- **Cloudflare Tunnel** - Behind NAT/firewall
- **Cloudflare Proxy** - CDN-enabled access

Specify mode when invoking the deploy skill.

---

**See [README.md](README.md) for complete installation and usage documentation.**
