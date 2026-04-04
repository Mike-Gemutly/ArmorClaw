---
name: provision
description: Use when setting up new ArmorChat or ArmorTerminal mobile devices to connect to your ArmorClaw server
---

# Provision Mobile Devices

## Overview

Generate QR codes and deep links for secure mobile app provisioning. Offers both automatic QR code display and manual entry options for connecting devices to your ArmorClaw VPS.

## When to Use

**Use provision when:**
- Setting up a new ArmorChat mobile app
- Configuring ArmorTerminal for the first time
- Need to connect mobile devices to your ArmorClaw server
- Want to provide multiple provisioning methods (QR or manual)

## Quick Reference

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `vps_ip` | string | auto-detected | Server IP or domain (reads from config if omitted) |
| `expiry` | integer | 300 | Token expiry in seconds (max: 3600) |
| `show_url` | boolean | false | Display URL only, no QR code |

## QR Code Generation

Generate QR codes with `qrencode`:

```bash
# Install qrencode (Ubuntu/Debian)
sudo apt install qrencode

# Install qrencode (macOS)
brew install qrencode

# QR code is auto-generated with UTF8 output
# Falls back to ASCII or ASCIIi if needed
```

The provisioning script tries multiple QR output formats:
- UTF8 (most compatible)
- ASCII (fallback)
- ASCIIi (inverted, final fallback)

## Deep Link Format

```
armorclaw://config?d=<base64-encoded-json>
```

**JSON payload contains:**
- `matrix_homeserver`: Matrix server URL
- `rpc_url`: Bridge RPC API URL
- `ws_url`: WebSocket URL for real-time events
- `push_gateway`: Push notification gateway URL
- `server_name`: Human-readable server name
- `setup_token`: Unique setup token (registered via bridge RPC)
- `expires_at`: Unix timestamp for expiry

## Security Notes

**DO NOT store credentials permanently:**
- Setup tokens are generated and displayed once only
- Tokens expire automatically (default 5 minutes, max 1 hour)
- Credentials are read from VPS, not stored locally
- Clear from session after display

**Bridge RPC integration:**
- Script attempts to register token with bridge via provisioning.start RPC
- Falls back to local token if bridge unreachable (may fail until bridge registers)
- Socket path: `/run/armorclaw/bridge.sock`

## Usage Examples

```bash
# Generate QR code with default 5-minute expiry
sudo ./deploy/armorclaw-provision.sh

# Generate QR code with 1-minute expiry
sudo ./deploy/armorclaw-provision.sh --expiry 60

# Show URL only (for scripting or manual entry)
sudo ./deploy/armorclaw-provision.sh --show-url
```

## Manual Entry Instructions

When QR code is unavailable or user prefers manual entry:

1. **Install ArmorChat** from Google Play
2. **Copy the deep link** displayed by the provisioning script
3. **Paste into ArmorChat** when prompted for configuration
4. **Set up biometrics** for secure keystore access
5. **Done** - your digital secretary is online

## Common Mistakes

**QR code generation fails:**
- Install qrencode: `sudo apt install qrencode`
- Script falls back to showing URL if qrencode unavailable

**Token not recognized by bridge:**
- Check bridge is running: `docker ps | grep armorclaw`
- Verify socket exists: `ls -la /run/armorclaw/bridge.sock`
- Bridge may need to register manually generated token

**Mobile app cannot connect:**
- Verify server is accessible from device (firewall rules)
- Check network connectivity (VPN, NAT, public IP)
- Ensure TLS is configured for production deployments

## Environment Detection

Script automatically detects deployment environment:

- **VPS (AWS/GCP/DigitalOcean)**: Warns about firewall rules, TLS configuration
- **Local/Hardware**: Uses local network IP, no TLS warnings
- **Production**: Recommends TLS for public deployments

## Platform Support

| Platform | QR Generation | Manual Entry |
|----------|---------------|--------------|
| **Linux** | qrencode (apt install) | Full support |
| **macOS** | qrencode (brew install) | Full support |
| **Windows (Git Bash)** | qrencode (mingw) | Full support |
| **Windows (PowerShell)** | URL display only | Copy-paste link |
| **WSL** | qrencode (apt) | Full support |

## References

- Script: `deploy/armorclaw-provision.sh`
- Config: `/etc/armorclaw/config.toml`
- Bridge RPC: `/run/armorclaw/bridge.sock`
- qrencode documentation: `man qrencode`
