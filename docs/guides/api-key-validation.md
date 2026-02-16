# API Key Validation Guide

> **Purpose:** Pre-validation and verification of API keys before use
> **Last Updated:** 2026-02-15
> **Integration:** Keystore, Setup Wizard

---

## Overview

ArmorClaw validates API keys before use to ensure they are active, have sufficient quota, and won't fail during critical operations. This guide documents the validation architecture and implementation.

### Validation Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    API KEY VALIDATION ARCHITECTURE                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐                                                        │
│  │   User Input    │                                                        │
│  │   (API Key)     │                                                        │
│  └────────┬────────┘                                                        │
│           │                                                                  │
│           ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      VALIDATION PIPELINE                             │    │
│  │  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐            │    │
│  │  │ Format  │──▶│ API     │──▶│ Quota   │──▶│ Expiry  │            │    │
│  │  │ Check   │   │ Call    │   │ Check   │   │ Check   │            │    │
│  │  └─────────┘   └─────────┘   └─────────┘   └─────────┘            │    │
│  │       │             │             │             │                   │    │
│  │       ▼             ▼             ▼             ▼                   │    │
│  │   [Syntax]     [Lightweight   [Usage     [Expiration             │    │
│  │    Check       Models API]    %]         Date]                   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│           │                                                                  │
│           ▼                                                                  │
│  ┌─────────────────┐     ┌─────────────────┐                               │
│  │    VALID ✓      │     │   INVALID ✗     │                               │
│  │  Store in       │     │  Show error     │                               │
│  │  Keystore       │     │  with details   │                               │
│  └─────────────────┘     └─────────────────┘                               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Validation Stages

### Stage 1: Format Check (Instant)

Validates the key format before making any API calls.

| Provider | Key Prefix | Format Validation |
|----------|-----------|-------------------|
| OpenAI | `sk-` | `sk-[a-zA-Z0-9]{20,}` |
| Anthropic | `sk-ant-` | `sk-ant-api03-[a-zA-Z0-9-]{80,}` |
| OpenRouter | `sk-or-` | `sk-or-v1-[a-zA-Z0-9]{32,}` |
| Google | `AI` | `AI[a-zA-Z0-9_-]{32,}` |
| xAI | `xai-` | `xai-[a-zA-Z0-9]{20,}` |

**Implementation:**

```go
// ValidateFormat checks if the key format is correct for the provider
func ValidateFormat(provider Provider, token string) error {
    switch provider {
    case ProviderOpenAI:
        if !strings.HasPrefix(token, "sk-") {
            return fmt.Errorf("OpenAI key must start with 'sk-'")
        }
        if len(token) < 20 {
            return fmt.Errorf("OpenAI key too short")
        }
    case ProviderAnthropic:
        if !strings.HasPrefix(token, "sk-ant-") {
            return fmt.Errorf("Anthropic key must start with 'sk-ant-'")
        }
    // ... other providers
    }
    return nil
}
```

### Stage 2: API Call Validation (1-3 seconds)

Makes a lightweight API call to verify the key is active.

| Provider | Endpoint | Method | Expected Response |
|----------|----------|--------|-------------------|
| OpenAI | `/v1/models` | GET | 200 with models list |
| Anthropic | `/v1/messages` (minimal) | POST | 200 or specific error |
| OpenRouter | `/api/v1/models` | GET | 200 with models list |
| Google | `/v1beta/models` | GET | 200 with models list |
| xAI | `/v1/models` | GET | 200 with models list |

**Implementation:**

```go
// ValidateWithAPI makes a lightweight API call to verify the key
func ValidateWithAPI(ctx context.Context, provider Provider, token string) (*ValidationResult, error) {
    switch provider {
    case ProviderOpenAI:
        return validateOpenAI(ctx, token)
    case ProviderAnthropic:
        return validateAnthropic(ctx, token)
    // ... other providers
    }
}

func validateOpenAI(ctx context.Context, token string) (*ValidationResult, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("API request failed: %w", err)
    }
    defer resp.Body.Close()

    result := &ValidationResult{
        IsValid: resp.StatusCode == 200,
    }

    if resp.StatusCode == 401 {
        result.Error = "Invalid API key"
        result.ErrorCode = "AUTH-001"
    } else if resp.StatusCode == 429 {
        result.IsValid = true // Key is valid, just rate limited
        result.Warning = "Rate limited - key is valid but throttled"
    }

    // Parse response to extract quota info if available
    if resp.StatusCode == 200 {
        var models ModelsResponse
        json.NewDecoder(resp.Body).Decode(&models)
        result.AvailableModels = len(models.Data)
    }

    return result, nil
}
```

### Stage 3: Quota Check (Optional)

Checks current usage against limits.

| Provider | Method | Data Available |
|----------|--------|----------------|
| OpenAI | Response headers | `x-ratelimit-remaining` |
| Anthropic | Dashboard only | Not available via API |
| OpenRouter | `/api/v1/auth/key` | Usage and limits |
| Google | Quota API | Daily/monthly limits |
| xAI | Response headers | Rate limit info |

**Implementation:**

```go
type QuotaInfo struct {
    Used            int64   `json:"used"`             // Tokens/messages used
    Limit           int64   `json:"limit"`            // Total limit
    Remaining       int64   `json:"remaining"`        // Remaining quota
    PercentageUsed  float64 `json:"percentage_used"`  // 0-100
    ResetAt         int64   `json:"reset_at"`         // Unix timestamp
    Tier            string  `json:"tier"`             // free/tier1/tier2
}

func CheckQuota(ctx context.Context, provider Provider, token string) (*QuotaInfo, error) {
    // Provider-specific implementation
}
```

### Stage 4: Expiry Check (Optional)

Checks if the key has an expiration date.

| Provider | Expiry Detection |
|----------|-----------------|
| OpenAI | Project keys may expire |
| Anthropic | Dashboard setting |
| OpenRouter | Key creation date |
| Google | API key settings |
| xAI | Dashboard setting |

**Implementation:**

```go
type ExpiryInfo struct {
    HasExpiry    bool   `json:"has_expiry"`
    ExpiresAt    int64  `json:"expires_at"`     // Unix timestamp
    DaysUntil    int    `json:"days_until"`     // Days until expiry
    IsExpired    bool   `json:"is_expired"`
    IsExpiringSoon bool `json:"is_expiring_soon"` // < 7 days
}

func CheckExpiry(cred *Credential) *ExpiryInfo {
    info := &ExpiryInfo{}

    if cred.ExpiresAt > 0 {
        info.HasExpiry = true
        info.ExpiresAt = cred.ExpiresAt

        now := time.Now().Unix()
        info.IsExpired = now > cred.ExpiresAt

        daysRemaining := int((cred.ExpiresAt - now) / 86400)
        info.DaysUntil = daysRemaining
        info.IsExpiringSoon = daysRemaining > 0 && daysRemaining < 7
    }

    return info
}
```

---

## Validation Results

### Result Structure

```go
type ValidationResult struct {
    IsValid         bool       `json:"is_valid"`
    Error           string     `json:"error,omitempty"`
    ErrorCode       string     `json:"error_code,omitempty"`
    Warning         string     `json:"warning,omitempty"`

    // Additional info
    Provider        string     `json:"provider"`
    AvailableModels int        `json:"available_models,omitempty"`
    QuotaInfo       *QuotaInfo `json:"quota_info,omitempty"`
    ExpiryInfo      *ExpiryInfo `json:"expiry_info,omitempty"`

    // Timing
    ValidatedAt     int64      `json:"validated_at"`
    ResponseTimeMs  int64      `json:"response_time_ms"`
}
```

### Error Codes

| Code | Meaning | Resolution |
|------|---------|------------|
| `VAL-001` | Invalid key format | Check key syntax |
| `VAL-002` | Key format doesn't match provider | Verify correct provider |
| `AUTH-001` | Authentication failed | Key is invalid or revoked |
| `AUTH-002` | Key expired | Generate new key |
| `QUOTA-001` | Quota exceeded | Wait for reset or upgrade |
| `QUOTA-002` | Quota warning (>80%) | Monitor usage |
| `RATE-001` | Rate limited | Wait and retry |
| `NET-001` | Network error | Check connectivity |
| `NET-002` | Timeout | Retry validation |

---

## RPC Integration

### Validation Methods

```bash
# Validate a new key before storing
echo '{"jsonrpc":"2.0","id":1,"method":"keys.validate","params":{
  "provider": "openai",
  "token": "sk-xxx",
  "check_quota": true,
  "check_expiry": true
}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response:
{
  "result": {
    "is_valid": true,
    "provider": "openai",
    "available_models": 58,
    "quota_info": {
      "used": 15000,
      "limit": 100000,
      "percentage_used": 15.0,
      "tier": "tier1"
    },
    "validated_at": 1739664000,
    "response_time_ms": 523
  }
}

# Validate an existing stored key
echo '{"jsonrpc":"2.0","id":1,"method":"keys.check","params":{
  "key_id": "my-openai-key"
}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Batch Validation

```bash
# Validate all stored keys
echo '{"jsonrpc":"2.0","id":1,"method":"keys.validate_all"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response:
{
  "result": {
    "total": 3,
    "valid": 2,
    "invalid": 1,
    "results": [
      {"key_id": "openai-main", "is_valid": true},
      {"key_id": "anthropic-backup", "is_valid": true},
      {"key_id": "old-key", "is_valid": false, "error": "AUTH-001: Authentication failed"}
    ]
  }
}
```

---

## Setup Wizard Integration

### Validation Flow in Setup

```
┌─────────────────────────────────────────────────────────────────┐
│  SETUP WIZARD: Add API Key                                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Step 1: Enter Key                                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Provider: [OpenAI    ▼]                                │   │
│  │  API Key:  [sk-proj-••••••••••••••••••••••••••]         │   │
│  │                                                          │   │
│  │  [Validate Key]                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  Step 2: Validation Result                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  ✅ Key Validated Successfully                          │   │
│  │                                                          │   │
│  │  Provider:     OpenAI                                   │   │
│  │  Models:       58 available                             │   │
│  │  Usage:        15% (15,000 / 100,000 tokens)            │   │
│  │  Tier:         Tier 1                                   │   │
│  │                                                          │   │
│  │  Key ID:       [my-openai-key_______]                   │   │
│  │                                                          │   │
│  │  [Store Key]  [Test with sample request]                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  Step 3: Test (Optional)                                        │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  🔄 Sending test request...                             │   │
│  │                                                          │   │
│  │  Model: gpt-4o-mini                                     │   │
│  │  Prompt: "Say 'Hello from ArmorClaw!'"                  │   │
│  │                                                          │   │
│  │  Response: "Hello from ArmorClaw!"                      │   │
│  │  Latency: 523ms                                         │   │
│  │                                                          │   │
│  │  ✅ Test successful! Your key is ready to use.          │   │
│  │                                                          │   │
│  │  [Done]                                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Error Handling in Setup

```
┌─────────────────────────────────────────────────────────────────┐
│  ⚠️ Validation Failed                                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Error: Invalid API key                                         │
│  Code: AUTH-001                                                 │
│                                                                  │
│  Possible causes:                                               │
│  • The key has been revoked                                     │
│  • The key was entered incorrectly                              │
│  • The key belongs to a different provider                      │
│                                                                  │
│  Suggested actions:                                             │
│  1. Check that the key starts with "sk-"                        │
│  2. Verify the key in your OpenAI dashboard                     │
│  3. Generate a new key if needed                                │
│                                                                  │
│  [Try Again]  [Skip Validation]  [Help]                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quota Monitoring

### Dashboard Display

```
┌─────────────────────────────────────────────────────────────────┐
│  API KEY STATUS                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  OpenAI (my-openai-key)                                         │
│  ───────────────────────────────────────────────────────────   │
│  Status:     ✅ Active                                          │
│  Usage:      [████████░░░░░░░░░░░░] 45%                        │
│              45,000 / 100,000 tokens                            │
│  Reset:      3 days                                             │
│  Last used:  2 minutes ago                                      │
│                                                                  │
│  ⚠️ Warning: Usage above 80% threshold                          │
│                                                                  │
│  Anthropic (claude-key)                                         │
│  ───────────────────────────────────────────────────────────   │
│  Status:     ✅ Active                                          │
│  Usage:      [██░░░░░░░░░░░░░░░░] 12%                          │
│              No quota info available via API                    │
│  Last used:  1 hour ago                                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Quota Alerts

| Threshold | Alert Level | Action |
|-----------|-------------|--------|
| < 50% | Info | None |
| 50-80% | Warning | Log notification |
| 80-95% | Error | Matrix notification |
| > 95% | Critical | Block operations, alert admin |

---

## Implementation Status

### Current State

| Feature | Status | Notes |
|---------|--------|-------|
| Format validation | ✅ Complete | Provider-specific format checks |
| API call validation | ⚠️ Partial | OpenAI implemented, others pending |
| Quota checking | ⚠️ Partial | Response header parsing only |
| Expiry detection | ⚠️ Partial | Manual entry only |
| Setup wizard | ✅ Complete | Validates keys before storing |
| RPC methods | ⚠️ Partial | `keys.validate` planned |

### Remaining Work

1. **Complete API call validation for all providers**
   - Anthropic: Implement minimal message test
   - OpenRouter: Add `/api/v1/models` call
   - Google: Add `/v1beta/models` call
   - xAI: Add `/v1/models` call

2. **Add key expiration date detection**
   - OpenAI: Parse project key metadata
   - Implement warning system for expiring keys

3. **Add key usage quota warnings**
   - Integrate quota info into keystore
   - Add periodic quota checks
   - Implement alert thresholds

---

## Best Practices

### For Users

1. **Always validate before storing** - Catch errors early
2. **Monitor quota regularly** - Avoid unexpected blocks
3. **Set up backup keys** - Prevent single points of failure
4. **Rotate keys periodically** - Security best practice

### For Administrators

1. **Require validation in setup wizard** - Prevent invalid keys
2. **Set up quota alerts** - Proactive monitoring
3. **Audit key usage** - Detect anomalies
4. **Document key lifecycle** - Rotation and expiry policies

---

## Related Documentation

- [Getting Started Guide](getting-started.md) - Initial setup
- [Configuration Guide](configuration.md) - Key storage configuration
- [Alert Integration](alert-integration.md) - Quota alerts
- [Error Catalog](error-catalog.md) - Error codes and solutions

---

**API Key Validation Guide Last Updated:** 2026-02-15
