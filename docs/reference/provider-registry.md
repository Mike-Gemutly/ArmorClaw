# Provider Registry Architecture

## Overview

ArmorClaw now uses a **provider registry pattern** (similar to LiteLLM, OpenRouter, Vercel AI Gateway) to manage AI providers. This allows:

- ✅ **No installer updates** for new providers
- ✅ **Alias support** (zai → zhipu, glm → zhipu)
- ✅ **Dynamic provider loading** from registry
- ✅ **Custom registries** via environment variables
- ✅ **Local model support** (Ollama)

---

## Architecture

```
ArmorClaw
│
├ configs/providers.json           ← Default registry (in repo)
│
├ bridge/pkg/providers/
│   ├ registry.go                  ← Load & parse JSON
│   ├ resolver.go                  ← Alias → ID resolution
│   └ registry_test.go             ← Unit tests
│
├ internal/wizard/quick.go        ← Uses registry for Catwalk menu
│
└ /etc/armorclaw/                  ← Runtime
    └ providers.json               ← Installed registry
```

---

## Registry Format

### File: `/etc/armorclaw/providers.json`

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
    }
  ]
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Provider identifier (unique) |
| `name` | string | Yes | Display name for users |
| `protocol` | string | Yes | `openai`, `anthropic`, `custom` |
| `base_url` | string | Yes | API endpoint URL |
| `aliases` | array | No | Alternative names (e.g., ["zai", "glm"]) |

---

## Loading Priority

Providers are loaded in this order (fallback hierarchy):

```
1. ARMORCLAW_PROVIDERS_URL (ENV override)
        ↓
2. /etc/armorclaw/providers.json (local file)
        ↓
3. Embedded registry (built-in fallback)
```

---

## Using the Registry

### In Go Code

```go
import "github.com/armorclaw/bridge/pkg/providers"

// Load embedded registry
registry := providers.LoadEmbeddedRegistry()

// Resolve a provider by ID or alias
provider, found := registry.ResolveProvider("zai")
if found {
    fmt.Println(provider.Name)  // "Zhipu AI (Z AI)"
    fmt.Println(provider.BaseURL)  // "https://api.z.ai/api/paas/v4"
}

// Get all providers
allProviders := registry.GetAllProviders()
```

### In Shell/Bash

The installer uses a hash map to resolve provider selections:

```bash
declare -A PROVIDERS=(
    ["1"]="openai"
    ["2"]="anthropic"
    ["6"]="zhipu"  # zai → zhipu via alias
)

provider_key="${PROVIDERS[$choice]}"

# Bridge resolves the alias internally
armorclaw-bridge add-key \
  --provider $provider_key \
  --token YOUR_KEY
```

---

## Adding a New Provider

### Option 1: Update Embedded Registry

Edit `bridge/pkg/providers/registry.go`:

```go
var EmbeddedRegistry = Registry{
    Providers: []Provider{
        // ... existing providers
        {
            ID:      "new-provider",
            Name:    "New Provider",
            Protocol: "openai",
            BaseURL: "https://api.new-provider.com/v1",
        },
    },
}
```

### Option 2: Update Config File

Edit `/etc/armorclaw/providers.json` (after installation):

```json
{
  "providers": [
    {
      "id": "new-provider",
      "name": "New Provider",
      "protocol": "openai",
      "base_url": "https://api.new-provider.com/v1"
    }
  ]
}
```

### Option 3: Use Custom Registry

Set environment variable:

```bash
export ARMORCLAW_PROVIDERS_URL=https://company.com/providers.json
```

---

## Provider Aliases

Aliases allow users to use alternative names:

```json
{
  "id": "zhipu",
  "name": "Zhipu AI (Z AI)",
  "protocol": "openai",
  "base_url": "https://api.z.ai/api/paas/v4",
  "aliases": ["zai", "glm", "bigmodel"]
}
```

Users can now type:
- `zai` → resolves to `zhipu`
- `glm` → resolves to `zhipu`
- `zhipu` → resolves to `zhipu`

---

## Resolution Logic

The registry resolves providers in this order:

```
Input: "zai"
     ↓
Check if "zai" is a valid provider ID
     ↓ (no)
Check if "zai" is an alias for any provider
     ↓ (yes, zhipu)
Return zhipu provider details
     ↓
Use BaseURL: "https://api.z.ai/api/paas/v4"
```

---

## Supported Providers

The default registry includes 12 providers:

| # | ID | Name | Protocol | Base URL |
|---|----|------|----------|----------|
| 1 | openai | OpenAI | openai | `https://api.openai.com/v1` |
| 2 | anthropic | Anthropic | anthropic | `https://api.anthropic.com/v1` |
| 3 | google | Google | openai | `https://generativelanguage.googleapis.com/v1` |
| 4 | xai | xAI | openai | `https://api.x.ai/v1` |
| 5 | openrouter | OpenRouter | openai | `https://openrouter.ai/api/v1` |
| 6 | zhipu | Zhipu AI (Z AI) | openai | `https://api.z.ai/api/paas/v4` |
| 7 | deepseek | DeepSeek | openai | `https://api.deepseek.com/v1` |
| 8 | moonshot | Moonshot AI | openai | `https://api.moonshot.ai/v1` |
| 9 | nvidia | NVIDIA NIM | openai | `https://integrate.api.nvidia.com/v1` |
| 10 | groq | Groq | openai | `https://api.groq.com/openai/v1` |
| 11 | cloudflare | Cloudflare | openai | `https://gateway.ai.cloudflare.com/v1` |
| 12 | ollama | Ollama (Local) | openai | `http://localhost:11434/v1` |

---

## Testing

Run the provider registry tests:

```bash
cd bridge
go test ./pkg/providers/...
```

Test cases include:
- ✅ Loading embedded registry
- ✅ Resolving providers by ID
- ✅ Resolving providers by alias
- ✅ Getting all providers
- ✅ Provider count validation

---

## Future Enhancements

### Planned Features

1. **Remote Registry Loading**
   ```bash
   export ARMORCLAW_PROVIDERS_URL=https://registry.example.com/providers.json
   ```

2. **Model Discovery**
   - Fetch models from provider's `/v1/models` endpoint
   - Cache models locally

3. **Dynamic Registry Updates**
   - Hot-reload registry without restart
   - Watch for file changes

4. **Provider Health Checks**
   - Check if provider URL is accessible
   - Mark unhealthy providers

5. **Built-in Registry Sync**
   - Auto-update from upstream registry
   - Support multiple versions

---

## Migration from Hardcoded Providers

### Before (Hardcoded):

```go
var apiProviders = []apiProviderOption{
    {Name: "Zhipu AI", Key: "openai", BaseURL: "https://open.bigmodel.cn/api/paas/v4"},
    {Name: "Moonshot AI", Key: "openai", BaseURL: "https://api.moonshot.cn/v1"},
}
```

### After (Registry-Based):

```go
// Load from registry
registry := providers.LoadEmbeddedRegistry()

providerOptions := make([]apiProviderOption, 0)
for _, p := range registry.Providers {
    providerOptions = append(providerOptions, apiProviderOption{
        Name: p.Name,
        Key: p.ID,
        BaseURL: p.BaseURL,
    })
}
```

---

## Benefits

1. **No Code Changes Required**
   - Add providers by editing JSON
   - No recompilation needed

2. **Consistent URLs**
   - Correct URLs enforced
   - No manual copy-paste errors

3. **Alias Support**
   - Users can use natural names
   - Future-proof for provider rebranding

4. **Enterprise Ready**
   - Custom registry per organization
   - Security through JSON validation

---

## Security Considerations

- Registry JSON should be readable only by armorclaw user
- Base URLs should use HTTPS
- Validate all URLs before adding to registry
- Consider signing the registry file

---

## References

- [LiteLLM Documentation](https://docs.litellm.ai/)
- [OpenRouter API](https://openrouter.ai/docs)
- [Charmbracelet Catwalk](https://github.com/charmbracelet/catwalk)
