# Model Documentation Generator Design

**Date**: 2026-03-20
**Status**: Approved
**Related**: RPC Test Coverage (2026-03-20), Provider Registry (v4.5.0)

## Problem

Model names are scattered across:
- Static `fallbackModels` map in `bridge/internal/wizard/quick.go`
- `configs/providers.json` (providers only, no models)
- Catwalk dynamic discovery (runtime only)

Users don't know what model IDs to use with each provider.

## Solution

CLI tool that generates `docs/reference/models.md` from the same code paths the product uses.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      gen-models CLI                              │
├─────────────────────────────────────────────────────────────────┤
│  1. LoadEmbeddedProviders()  ←  configs/providers.json          │
│  2. LoadFallbackModels()     ←  quick.go fallbackModels map     │
│  3. TryCatwalkEnrichment()   ←  optional, if --catwalk-url set  │
│  4. RenderMarkdown()         ←  template                        │
│  5. WriteFile()              ←  docs/reference/models.md        │
└─────────────────────────────────────────────────────────────────┘
```

**Source Priority**:
1. Embedded registry (deterministic, always available)
2. Catwalk enrichment (optional, for richer data when available)

## Components

### 1. CLI Tool: `bridge/cmd/gen-models/main.go`

```go
type Options struct {
    OutputPath  string  // default: docs/reference/models.md
    CatwalkURL  string  // default: empty (skip Catwalk)
}

// Exit codes:
// 0 = success (docs generated)
// 1 = generator error (file write, parse error, etc.)
```

**Behavior**:
1. Load providers from embedded registry (`pkg/providers`)
2. Load models from `fallbackModels` map (share code with `quick.go`)
3. If `--catwalk-url` set and reachable, merge Catwalk data
4. Generate markdown
5. Write to output path

### 2. Output: `docs/reference/models.md`

```markdown
# AI Provider Models

> Generated from embedded provider registry.
> Last updated: 2026-03-20T15:30:00Z

## Providers

| ID | Name | Protocol | Aliases |
|----|------|----------|---------|
| openai | OpenAI | openai | - |
| zhipu | Zhipu AI (Z AI) | openai | zai, glm |

## Models by Provider

### OpenAI

| Model ID | Context |
|----------|---------|
| gpt-4o | 128000 |
| gpt-4o-mini | 128000 |

### Zhipu AI (zai, glm)

| Model ID | Context |
|----------|---------|
| glm-4 | - |
| glm-4v | - |
```

### 3. CI Workflow: `.github/workflows/docs.yml`

```yaml
name: Documentation

on:
  push:
    branches: [main]
    paths:
      - 'configs/providers.json'
      - 'bridge/internal/wizard/quick.go'
      - 'bridge/internal/wizard/catwalk.go'
  pull_request:
    paths:
      - 'configs/providers.json'
      - 'bridge/internal/wizard/quick.go'
  workflow_dispatch:

jobs:
  verify-model-docs:
    name: Verify Model Docs
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'bridge/go.mod'

      - name: Generate model docs
        run: |
          cd bridge
          go run ./cmd/gen-models --output ../docs/reference/models.md

      - name: Check for drift
        run: |
          if git diff --exit-code docs/reference/models.md; then
            echo "Model docs up to date"
          else
            echo "::warning::Model docs out of date. Run: go run ./bridge/cmd/gen-models"
            git diff docs/reference/models.md
          fi
```

**Key points**:
- Runs on changes to provider/model source files
- No Catwalk dependency in CI (deterministic)
- Warning only on drift, not failure
- Separate from dockerhub.yml

### 4. README Update

In "Supported AI Providers" section:

```markdown
### Supported AI Providers (v4.5.0+)

> **Full model list**: See [docs/reference/models.md](docs/reference/models.md) for all available models per provider.

ArmorClaw uses a Provider Registry pattern...
```

## Code Reuse

| Component | Source |
|-----------|--------|
| Provider loading | `pkg/providers.LoadEmbeddedRegistry()` |
| Catwalk client | `internal/wizard/catwalk.go` |
| Fallback models | Extract `fallbackModels` to shared location |

**Refactor needed**: Move `fallbackModels` from `quick.go` to `pkg/providers` or `internal/models` for reuse.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - docs generated |
| 1 | Error - generator failed (file I/O, parse error) |

Doc drift is NOT an error - CI handles that separately with warnings.

## Security Considerations

- No network calls by default (Catwalk optional)
- No secrets required
- Read-only access to source files
- Generated file committed to repo (reviewable)

## Testing

1. **Unit tests**: Provider/model loading, markdown rendering
2. **Integration**: Full generation with mock Catwalk
3. **CI verification**: Workflow runs on PR

## Files Changed

| File | Action |
|------|--------|
| `bridge/cmd/gen-models/main.go` | Create |
| `bridge/pkg/providers/models.go` | Create (extract fallbackModels) |
| `docs/reference/models.md` | Generate |
| `.github/workflows/docs.yml` | Create |
| `README.md` | Add link to models.md |
| `bridge/internal/wizard/quick.go` | Import shared models |
