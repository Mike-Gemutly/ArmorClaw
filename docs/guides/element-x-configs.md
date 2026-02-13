# Sending Configs via Element X

This guide explains how to send configuration files to ArmorClaw agents using the Element X app.

> **Last Updated:** 2026-02-07
> **Phase:** Priority 2 - Config System

## Overview

ArmorClaw supports receiving configuration files through Matrix messages. This allows you to:

- Send environment files (`.env`)
- Update agent settings (`.toml`, `.yaml`)
- Inject secrets without rebuilding containers
- Configure agents on-the-fly

## Quick Start

### Prerequisites

1. ArmorClaw Bridge running with Matrix enabled
2. Element X app connected to your Matrix homeserver
3. Agent running in a container with bridge connection

### Method 1: Using the /attach_config Command

The simplest way to attach configs is via the `/attach_config` command in Element X:

```
/attach_config agent.env MODEL=gpt-4 TEMPERATURE=0.7
```

The agent will respond with:

```json
{
  "config_id": "config-agent.env-1234567890",
  "name": "agent.env",
  "path": "/run/armorclaw/configs/agent.env",
  "size": 25,
  "type": "env"
}
```

### Method 2: Sending File Attachments

For larger configurations, send a file attachment:

1. **Compose a message** in Element X
2. **Attach the file** (e.g., `config.toml`, `.env`)
3. **Send** to the agent's room

The agent will:
- Receive the file via Matrix
- Store it in `/run/armorclaw/configs/`
- Return a config_id for reference

## Configuration Types

### Environment Files (`.env`)

Simple key-value pairs:

```
/attach_config agent.env MODEL=gpt-4
/attach_config agent.env MODEL=gpt-4 TEMPERATURE=0.7 MAX_TOKENS=4096
```

Result: Stored as `/run/armorclaw/configs/agent.env`

### TOML Configuration

For complex settings, use TOML:

```
/attach_config agent.toml [agent]
model=gpt-4
temperature=0.7

[limits]
max_tokens=4096
timeout=30
```

### JSON Configuration

For structured data:

```
/attach_config settings.json {"model":"gpt-4","temperature":0.7}
```

## Advanced Usage

### Base64 Encoding

For binary or special character content, use base64:

```bash
# Encode your config
echo "SECRET=value" | base64

# Send via Element X
/attach_config secret.env U0VDUkVUPXZhbHVlCg== base64
```

### Config Metadata

Attach metadata for tracking:

```
/attach_config config.env MODEL=gpt-4
# Metadata is auto-generated with:
# - Timestamp
# - Sender
# - Config ID
```

## Config Lifecycle

### Storage Location

All configs are stored in: `/run/armorclaw/configs/`

This is a tmpfs-mounted directory, so configs are:
- ✅ Ephemeral (wiped on reboot)
- ✅ In-memory only (never written to disk)
- ✅ Isolated per container

### Config IDs

Each attachment gets a unique ID:

```
config-<name>-<timestamp>
```

Example: `config-agent.env-1736294400`

### Retrieving Configs

To list attached configs, use the bridge RPC:

```bash
echo '{"jsonrpc":"2.0","method":"list_configs","id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

## Security Considerations

### Path Traversal Protection

The bridge validates all config names:

```
❌ /attach_config ../../../etc/passwd malicious
# Error: invalid config name (path traversal not allowed)

❌ /attach_config /absolute/path.env content
# Error: invalid config name (absolute paths not allowed)

✅ /attach_config relative/path.env content
# OK: stored as /run/armorclaw/configs/relative/path.env
```

### Size Limits

Configs are limited to 1MB:

```
❌ <very large config>
# Error: config content too large (max 1 MB)
```

### Encoding Options

| Encoding | Description | Use Case |
|----------|-------------|----------|
| `raw` | Plain text (default) | Most configs |
| `base64` | Base64 encoded | Binary content, special chars |

## Examples

### Example 1: Configure GPT-4 Model

```
/attach_config model.env OPENAI_API_KEY=sk-xxx
/attach_config model.env MODEL=gpt-4
```

### Example 2: Set Temperature

```
/attach_config settings.env TEMPERATURE=0.7 MAX_TOKENS=4096
```

### Example 3: Complex TOML Config

```
/attach_config agent.toml [agent]
model=gpt-4
temperature=0.7

[logging]
level=info
format=json

[matrix]
enabled=true
room_id=!abc123:matrix.example.com
```

### Example 4: Multi-line YAML

```
/attach_config config.yaml agent:
  model: gpt-4
  temperature: 0.7

logging:
  level: info
```

## Troubleshooting

### Config Not Found

```
❌ Error: config file not found at /run/armorclaw/configs/test.env
```

**Solution:** Check that the config was attached successfully:

```
/status
# Look for "configs_attached" count
```

### Invalid Encoding

```
❌ Error: failed to decode base64 content
```

**Solution:** Verify your base64 encoding:

```bash
echo "test" | base64  # Outputs: dGVzdAo=
```

### Bridge Not Connected

```
❌ Error: Bridge not connected. Cannot attach config.
```

**Solution:** Ensure the bridge is running:

```bash
# Check bridge status
cd bridge && sudo ./build/armorclaw-bridge status

# Start bridge if needed
cd bridge && sudo ./build/armorclaw-bridge -matrix-enabled
```

## Testing

### Test attach_config RPC Method

```bash
./tests/test-attach-config.sh
```

This tests:
- ✅ Raw content attachment
- ✅ Base64-encoded content
- ✅ TOML configuration
- ✅ Path traversal protection
- ✅ Parameter validation
- ✅ File system verification

### Manual Testing

1. Start the agent container
2. Connect via Element X
3. Send test config:

```
/attach_config test.env KEY=VALUE
```

4. Verify response includes `config_id`

## Next Steps

After attaching configs:

1. **Inject into containers** - Configs are available at `/run/armorclaw/configs/`
2. **Use envsubst** - Template environment variables into configs
3. **Monitor agent** - Check `/status` for loaded configs

See also:
- [Configuration Guide](configuration.md)
- [Bridge RPC API](../reference/rpc-api.md)
- [Element X Quick Start](element-x-quickstart.md)
