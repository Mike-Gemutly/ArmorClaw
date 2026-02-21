# ArmorClaw Bridge Configuration Guide

> **Last Updated:** 2026-02-06
> **Version:** 0.2.0
> **Component:** Local Bridge Configuration

---

## Overview

The ArmorClaw bridge uses a hierarchical configuration system with four layers:

1. **Built-in Defaults** - Sensible defaults for all settings
2. **Configuration File** - TOML file with customizable options
3. **Environment Variables** - Override config file settings
4. **CLI Flags** - Override all other settings

**Precedence:** CLI Flags > Environment Variables > Config File > Defaults

---

## Configuration File Locations

The bridge searches for configuration files in the following order (first found wins):

1. `~/.armorclaw/config.toml` - User-specific configuration
2. `/etc/armorclaw/config.toml` - System-wide configuration
3. `./config.toml` - Current directory configuration

### Custom Location

```bash
./build/armorclaw-bridge -config /path/to/custom.toml
```

---

## Configuration Sections

### Server Configuration

```toml
[server]
# Unix socket path for JSON-RPC communication (default: "/run/armorclaw/bridge.sock")
socket_path = "/run/armorclaw/bridge.sock"

# PID file for daemon mode (default: "/run/armorclaw/bridge.pid")
pid_file = "/run/armorclaw/bridge.pid"

# Run as background daemon (default: false)
daemonize = false
```

**Environment Variables:**
- `ARMORCLAW_SOCKET` - Socket path
- `ARMORCLAW_PID_FILE` - PID file path
- `ARMORCLAW_DAEMONIZE` - Run as daemon (true/false)

---

### Keystore Configuration

```toml
[keystore]
# Path to the encrypted keystore database
db_path = "~/.armorclaw/keystore.db"

# Optional master key (hex-encoded, NOT RECOMMENDED - use hardware derivation)
# master_key = ""

# Pre-configured provider credentials (optional)
[[keystore.providers]]
id = "openai-key-1"
provider = "openai"
token = "sk-proj-..."  # ⚠️ Stored encrypted at rest
display_name = "Production OpenAI Key"
expires_at = 1741459200  # Unix timestamp
tags = ["production", "gpt-4"]
```

**Environment Variables:**
- `ARMORCLAW_KEYSTORE_DB` - Database path
- `ARMORCLAW_MASTER_KEY` - Master key (⚠️ NOT RECOMMENDED)
- `ARMORCLAW_PROVIDER_TOKEN` - Provider token for dynamic loading

**Providers:**
- `openai` - OpenAI API
- `anthropic` - Anthropic Claude API
- `openrouter` - OpenRouter API
- `google` - Google Generative AI
- `xai` - xAI (Grok) API

---

### Matrix Configuration

```toml
[matrix]
# Enable Matrix communication (default: false)
enabled = true

# Matrix homeserver URL (required if enabled)
homeserver_url = "https://matrix.armorclaw.com"

# Bot username (for auto-login)
username = "bridge-bot"

# Bot password (NOT RECOMMENDED - use RPC login)
# password = "secret"

# Device ID for reconnection (default: "armorclaw-bridge")
device_id = "armorclaw-bridge"

# Sync interval in seconds (default: 5, minimum: 1)
sync_interval = 5

# Rooms to automatically join on login
auto_rooms = [
    "!room:matrix.armorclaw.com"
]

# Retry configuration
[matrix.retry]
max_retries = 3
retry_delay = 5  # seconds
backoff_multiplier = 2.0
```

**Environment Variables:**
- `ARMORCLAW_MATRIX_ENABLED` - Enable Matrix (true/false)
- `ARMORCLAW_MATRIX_HOMESERVER` - Homeserver URL
- `ARMORCLAW_MATRIX_USERNAME` - Bot username
- `ARMORCLAW_MATRIX_PASSWORD` - Bot password (⚠️ NOT RECOMMENDED)
- `ARMORCLAW_MATRIX_DEVICE_ID` - Device ID
- `ARMORCLAW_MATRIX_SYNC_INTERVAL` - Sync interval in seconds

---

### Logging Configuration

```toml
[logging]
# Log level: debug, info, warn, error (default: "info")
level = "info"

# Log format: json, text (default: "text")
format = "text"

# Log output: stdout, stderr, file (default: "stdout")
output = "stdout"

# Log file path (required when output is "file")
# file = "/var/log/armorclaw/bridge.log"
```

**Environment Variables:**
- `ARMORCLAW_LOG_LEVEL` - Log level
- `ARMORCLAW_LOG_FORMAT` - Log format
- `ARMORCLAW_LOG_OUTPUT` - Log output destination
- `ARMORCLAW_LOG_FILE` - Log file path

---

## Complete Example Configuration

```toml
# ~/.armorclaw/config.toml

[server]
socket_path = "/run/armorclaw/bridge.sock"
pid_file = "/run/armorclaw/bridge.pid"
daemonize = false

[keystore]
db_path = "~/.armorclaw/keystore.db"

# Pre-configure credentials (optional)
[[keystore.providers]]
id = "openai-prod"
provider = "openai"
display_name = "Production OpenAI Key"
tags = ["production", "gpt-4"]

[matrix]
enabled = true
homeserver_url = "https://matrix.armorclaw.com"
username = "bridge-bot"
device_id = "armorclaw-bridge"
sync_interval = 5

[matrix.retry]
max_retries = 3
retry_delay = 5
backoff_multiplier = 2.0

[logging]
level = "info"
format = "text"
output = "stdout"
```

---

## Configuration Commands

### Initialize

```bash
./build/armorclaw-bridge init
```

Creates `~/.armorclaw/config.toml` with default values.

### Validate

```bash
./build/armorclaw-bridge validate
```

Validates the configuration and reports any errors.

---

## Configuration Scenarios

### Development Configuration

```bash
# Enable debug logging, disable Matrix
export ARMORCLAW_LOG_LEVEL=debug
export ARMORCLAW_MATRIX_ENABLED=false

sudo ./build/armorclaw-bridge
```

### Production Configuration

```bash
# Production config with file logging
sudo ./build/armorclaw-bridge \
  -config /etc/armorclaw/production.toml \
  -log-level=warn \
  -log-output=file \
  -log-file=/var/log/armorclaw/bridge.log
```

### Docker Deployment

```bash
# docker-compose.yml
services:
  bridge:
    image: armorclaw/bridge:v1
    environment:
      - ARMORCLAW_MATRIX_ENABLED=true
      - ARMORCLAW_MATRIX_HOMESERVER=${MATRIX_HOMESERVER}
      - ARMORCLAW_LOG_LEVEL=warn
    volumes:
      - ./config:/etc/armorclaw
      - /run/armorclaw:/run/armorclaw
```

---

## Validation Rules

### Required Settings

- **server.socket_path** - Required
- **keystore.db_path** - Required
- **matrix.homeserver_url** - Required if Matrix enabled
- **matrix.sync_interval** - Must be >= 1 if Matrix enabled
- **logging.level** - Must be: debug, info, warn, error
- **logging.format** - Must be: json, text
- **logging.output** - Must be: stdout, stderr, file
- **logging.file** - Required if logging.output is "file"

### Retry Configuration

- **matrix.retry.max_retries** - Cannot be negative
- **matrix.retry.retry_delay** - Cannot be negative (seconds)
- **matrix.retry.backoff_multiplier** - Must be >= 1.0

---

## Common Issues

### Permission Denied on Socket

```bash
# Solution: Run with sudo
sudo ./build/armorclaw-bridge

# Or create directory with correct permissions
sudo mkdir -p /run/armorclaw
sudo chown $USER:$USER /run/armorclaw
```

### Matrix Login Failed

```bash
# Solution: Verify homeserver URL and use RPC login
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.login","params":{"username":"bridge-bot","password":"secret"}}' | \
  nc -U /run/armorclaw/bridge.sock
```

### Config File Not Found

```bash
# Solution: Initialize config
./build/armorclaw-bridge init

# Or use CLI flags only
./build/armorclaw-bridge -matrix-enabled -log-level debug
```

---

## Security Best Practices

1. **Never store passwords in config** - Use RPC login instead
2. **Use hardware-derived keys** - Let the bridge derive the master key from hardware
3. **Restrict file permissions** - `chmod 600 ~/.armorclaw/*`
4. **Use access tokens** - Prefer tokens over passwords when possible
5. **Run as non-root** - Use daemon mode with appropriate user

---

**Configuration Guide Last Updated:** 2026-02-06
**Compatible with Bridge Version:** 1.0.0
