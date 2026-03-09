# Provider Registry - Quick Reference

## What's New?

ArmorClaw now uses a **provider registry** - you can add new AI providers by editing a JSON file. No code changes needed!

---

## Available Providers

| # | Name | Aliases |
|---|------|---------|
| 1 | OpenAI | - |
| 2 | Anthropic | - |
| 3 | Google | - |
| 4 | xAI | - |
| 5 | OpenRouter | - |
| 6 | **Zhipu AI (Z AI)** | **zai, glm** |
| 7 | DeepSeek | - |
| 8 | **Moonshot AI** | - |
| 9 | NVIDIA NIM | - |
| 10 | Groq | - |
| 11 | Cloudflare | - |
| 12 | Ollama (Local) | - |

---

## Adding a Provider

### Method 1: Edit Registry File

```bash
sudo nano /etc/armorclaw/providers.json
```

Add your provider:

```json
{
  "providers": [
    {
      "id": "your-provider",
      "name": "Your Provider",
      "protocol": "openai",
      "base_url": "https://api.your-provider.com/v1"
    }
  ]
}
```

Restart:

```bash
sudo systemctl restart armorclaw-bridge
```

### Method 2: Add via Bridge (Dynamic)

```bash
armorclaw-bridge add-key \
  --provider your-provider \
  --token YOUR_KEY
```

---

## Using Aliases

You can use alternative names for providers:

```bash
# All these work the same (resolve to Zhipu AI)
armorclaw-bridge add-key --provider zhipu --token $KEY
armorclaw-bridge add-key --provider zai --token $KEY
armorclaw-bridge add-key --provider glm --token $KEY
```

---

## Quick Commands

```bash
# Check provider registry
cat /etc/armorclaw/providers.json

# Restart bridge (after adding provider)
sudo systemctl restart armorclaw-bridge

# View bridge logs
sudo journalctl -u armorclaw-bridge -f

# Check bridge status
sudo systemctl status armorclaw-bridge
```

---

## Common Additions

### Zhipu AI

```bash
# Already included with aliases:
# - zhipu
# - zai
# - glm
```

### Moonshot AI

```json
{
  "id": "moonshot",
  "name": "Moonshot AI",
  "protocol": "openai",
  "base_url": "https://api.moonshot.ai/v1"
}
```

### Ollama (Local)

```json
{
  "id": "ollama",
  "name": "Ollama (Local)",
  "protocol": "openai",
  "base_url": "http://localhost:11434/v1"
}
```

---

## Troubleshooting

**Registry file not found:**
```bash
sudo cp /opt/armorclaw/configs/providers.json /etc/armorclaw/providers.json
sudo systemctl restart armorclaw-bridge
```

**Invalid JSON:**
```bash
sudo cp /etc/armorclaw/providers.json.backup /etc/armorclaw/providers.json
sudo systemctl restart armorclaw-bridge
```

**Provider not recognized:**
```bash
# Check syntax
python3 -m json.tool /etc/armorclaw/providers.json

# View logs
sudo journalctl -u armorclaw-bridge -n 50
```

---

## Testing

```bash
# Test provider resolution
armorclaw-bridge add-key \
  --provider zhipu \
  --id test \
  --token test \
  --dry-run 2>/dev/null || true
```

---

## Resources

- [Full Documentation](/docs/reference/provider-registry.md)
- [Migration Guide](/docs/guides/provider-registry-migration.md)
- [GitHub Repository](https://github.com/Gemutly/ArmorClaw)

---

**Last Updated:** 2026-03-09
