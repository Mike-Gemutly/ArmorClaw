# Provider Registry Migration Guide

## Overview

ArmorClaw v4.5.0+ introduces a **provider registry architecture** that replaces hardcoded provider lists with a centralized JSON-based system.

This guide helps you migrate from the old hardcoded provider approach to the new registry-based system.

---

## What Changed?

### Before (Hardcoded):

```
installer → hardcoded list → 5 providers only
            - openai
            - anthropic
            - openrouter
            - google
            - xai
```

### After (Registry-Based):

```
installer → provider registry (12 providers)
            - openai, anthropic, google, xai
            - openrouter
            - zhipu (Z AI)
            - deepseek
            - moonshot
            - nvidia
            - groq
            - cloudflare
            - ollama
```

---

## Quick Migration Steps

### For New Installations

**No action needed!** The new installer automatically uses the provider registry.

```bash
# Install with new architecture
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

### For Existing Installations (v4.5.0+)

Run the installer again - it will automatically use the registry:

```bash
cd /opt/armorclaw
sudo ./deploy/installer-v5.sh
```

The installer will:
1. Detect existing installation
2. Skip rebuilding bridge if already installed
3. Update configuration to use registry
4. Ensure `/etc/armorclaw/providers.json` exists

---

## Manual Registry Update

If you want to manually add a provider to your existing installation:

### Step 1: Backup Existing Config

```bash
sudo cp /etc/armorclaw/providers.json /etc/armorclaw/providers.json.backup
```

### Step 2: Update Provider List

Edit `/etc/armorclaw/providers.json`:

```json
{
  "providers": [
    {
      "id": "zhipu",
      "name": "Zhipu AI (Z AI)",
      "protocol": "openai",
      "base_url": "https://api.z.ai/api/paas/v4",
      "aliases": ["zai", "glm"]
    },
    {
      "id": "moonshot",
      "name": "Moonshot AI",
      "protocol": "openai",
      "base_url": "https://api.moonshot.ai/v1"
    },
    {
      "id": "your-new-provider",
      "name": "Your New Provider",
      "protocol": "openai",
      "base_url": "https://api.your-provider.com/v1"
    }
  ]
}
```

### Step 3: Verify File

```bash
# Check syntax
sudo python3 -m json.tool /etc/armorclaw/providers.json

# Should output: valid JSON
```

### Step 4: Restart Bridge

```bash
sudo systemctl restart armorclaw-bridge

# Check status
sudo systemctl status armorclaw-bridge
```

### Step 5: Test Provider Resolution

```bash
# Use bridge's CLI to list providers
armorclaw-bridge list-providers

# Should show all providers from registry
```

---

## Adding Zhipu AI to Existing Installation

### Option A: Via Installer (Recommended)

```bash
cd /opt/armorclaw
sudo ./deploy/installer-v5.sh
```

### Option B: Manual Edit

1. Edit `/etc/armorclaw/providers.json`:
```json
{
  "providers": [
    {
      "id": "zhipu",
      "name": "Zhipu AI (Z AI)",
      "protocol": "openai",
      "base_url": "https://api.z.ai/api/paas/v4",
      "aliases": ["zai", "glm"]
    }
  ]
}
```

2. Restart bridge:
```bash
sudo systemctl restart armorclaw-bridge
```

---

## Adding Moonshot AI to Existing Installation

1. Edit `/etc/armorclaw/providers.json`:
```json
{
  "providers": [
    {
      "id": "moonshot",
      "name": "Moonshot AI",
      "protocol": "openai",
      "base_url": "https://api.moonshot.ai/v1"
    }
  ]
}
```

2. Restart bridge:
```bash
sudo systemctl restart armorclaw-bridge
```

---

## Using Provider Aliases

Once the registry is updated, you can use aliases:

### In Bridge CLI:

```bash
# Use "zai" (alias for zhipu)
armorclaw-bridge add-key \
  --provider zai \
  --token YOUR_ZHIPU_KEY

# Bridge automatically resolves to zhipu
```

### In ArmorChat:

Select "Zhipu AI (Z AI)" from the provider menu.

---

## Troubleshooting

### Issue: Registry file doesn't exist

**Error:**
```
providers.json not found
```

**Fix:**
```bash
# Install default registry
sudo cp /opt/armorclaw/configs/providers.json /etc/armorclaw/providers.json

# Restart bridge
sudo systemctl restart armorclaw-bridge
```

### Issue: Invalid JSON in registry

**Error:**
```
failed to parse registry JSON
```

**Fix:**
```bash
# Restore backup
sudo cp /etc/armorclaw/providers.json.backup /etc/armorclaw/providers.json

# Restart bridge
sudo systemctl restart armorclaw-bridge
```

### Issue: Bridge not recognizing new providers

**Error:**
```
provider not found
```

**Fix:**
```bash
# Verify registry file exists
ls -la /etc/armorclaw/providers.json

# Check bridge logs
sudo journalctl -u armorclaw-bridge -n 50

# Restart bridge
sudo systemctl restart armorclaw-bridge
```

---

## Testing Your Migration

### 1. Verify Registry File

```bash
# Check file exists
sudo test -f /etc/armorclaw/providers.json && echo "✓ Registry exists"

# Check syntax
sudo python3 -m json.tool /etc/armorclaw/providers.json >/dev/null 2>&1 && echo "✓ Valid JSON"

# Check provider count
jq '.providers | length' /etc/armorclaw/providers.json && echo "✓ Providers loaded"
```

### 2. Test Provider Resolution

```bash
# Test Zhipu (including aliases)
armorclaw-bridge add-key \
  --provider zhipu \
  --id test-zhipu \
  --token test \
  --dry-run 2>/dev/null || true

# Test ZAI alias
armorclaw-bridge add-key \
  --provider zai \
  --id test-zai \
  --token test \
  --dry-run 2>/dev/null || true

# Test GLM alias
armorclaw-bridge add-key \
  --provider glm \
  --id test-glm \
  --token test \
  --dry-run 2>/dev/null || true
```

### 3. Check Bridge Logs

```bash
# Look for registry loading messages
sudo journalctl -u armorclaw-bridge -n 100 | grep -i provider

# Should show:
# "Loading providers from registry"
# "Found 12 providers"
```

---

## Common Use Cases

### Adding a New Custom Provider

1. Edit `/etc/armorclaw/providers.json`:
```json
{
  "providers": [
    {
      "id": "custom-llm",
      "name": "Custom LLM",
      "protocol": "openai",
      "base_url": "https://api.custom-llm.com/v1"
    }
  ]
}
```

2. Restart bridge:
```bash
sudo systemctl restart armorclaw-bridge
```

3. Use it:
```bash
armorclaw-bridge add-key \
  --provider custom-llm \
  --token YOUR_KEY
```

### Adding Multiple Aliases

```json
{
  "id": "openai",
  "name": "OpenAI",
  "protocol": "openai",
  "base_url": "https://api.openai.com/v1",
  "aliases": ["open", "openai-api", "gpt"]
}
```

Now users can type: `open`, `openai-api`, or `gpt` and it resolves to OpenAI.

---

## Rollback Procedure

If you need to rollback to the old hardcoded approach:

### 1. Restore Backup

```bash
sudo cp /etc/armorclaw/providers.json.backup /etc/armorclaw/providers.json
```

### 2. Edit Source Code

Edit `bridge/internal/wizard/quick.go`:
- Restore the old hardcoded `apiProviders` list
- Remove registry loading logic

### 3. Rebuild and Restart

```bash
cd bridge
go build -o armorclaw-bridge ./cmd/bridge

sudo cp build/armorclaw-bridge /opt/armorclaw/armorclaw-bridge
sudo systemctl restart armorclaw-bridge
```

---

## Summary

| Task | Command |
|------|---------|
| Update registry | `sudo nano /etc/armorclaw/providers.json` |
| Restart bridge | `sudo systemctl restart armorclaw-bridge` |
| View logs | `sudo journalctl -u armorclaw-bridge -f` |
| Backup registry | `sudo cp /etc/armorclaw/providers.json ~/backup/` |
| Restore registry | `sudo cp ~/backup/providers.json /etc/armorclaw/` |

---

## Additional Resources

- [Provider Registry Documentation](/docs/reference/provider-registry.md)
- [Configuration Guide](/docs/guides/configuration.md)
- [RPC API Reference](/docs/reference/rpc-api.md)
- [GitHub Repository](https://github.com/Gemutly/ArmorClaw)

---

**Version:** v4.5.0+
**Last Updated:** 2026-03-09
