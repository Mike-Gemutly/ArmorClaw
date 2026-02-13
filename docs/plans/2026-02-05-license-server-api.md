# License Server API Specification

**Date:** 2026-02-05
**Purpose:** License validation for ArmorClaw premium features
**Version:** 1.0.0

---

## Overview

The License Server validates feature access for ArmorClaw Pro and Enterprise tiers. It provides a simple REST API that the Robust Bridge queries to determine if premium features are available.

### Design Principles

1. **Offline-first** - Clients cache licenses with grace period
2. **Privacy-respecting** - Minimal telemetry, no payload data
3. **High-availability** - Cached responses during outages
4. **Simple** - Single endpoint, minimal data exchange

---

## API Endpoints

### Base URL
```
Production: https://api.armorclaw.com/v1
Staging: https://api-staging.armorclaw.com/v1
```

---

### POST /licenses/validate

Validate a license key and check feature access.

#### Request

```http
POST /v1/licenses/validate
Content-Type: application/json
X-License-Key: [OPTIONAL: License key in header]

{
  "license_key": "SCLW-PRO-1234567890ABCDEF",  // or use X-License-Key header
  "instance_id": "550e8400-e29b-41d4-a716-446655440000",
  "feature": "slack-adapter",  // Feature to validate
  "version": "1.0.0"
}
```

#### Response (Success)

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "valid": true,
  "tier": "pro",
  "features": [
    "slack-adapter",
    "discord-adapter",
    "pii-scrubber"
  ],
  "expires_at": "2026-03-05T00:00:00Z",
  "instance_id": "550e8400-e29b-41d4-a716-446655440000",
  "grace_period_days": 3
}
```

#### Response (Invalid License)

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "valid": false,
  "error_code": "LICENSE_EXPIRED",
  "error_message": "License expired on 2026-01-05",
  "tier": null,
  "features": [],
  "grace_period_days": 3  // Still allow offline use
}
```

#### Response (Feature Not Available)

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "valid": true,
  "tier": "pro",
  "feature_valid": false,
  "error_code": "FEATURE_NOT_INCLUDED",
  "error_message": "The 'whatsapp-adapter' feature requires Enterprise tier",
  "available_features": [
    "slack-adapter",
    "discord-adapter",
    "pii-scrubber"
  ]
}
```

#### Error Responses

```http
HTTP/1.1 400 Bad Request
{"error": "Invalid request body"}

HTTP/1.1 401 Unauthorized
{"error": "Invalid license key format"}

HTTP/1.1 429 Too Many Requests
{
  "error": "Rate limit exceeded",
  "retry_after": 60
}

HTTP/1.1 500 Internal Server Error
{
  "error": "Temporary server error",
  "grace_period_active": true
}
```

---

### GET /licenses/status

Get detailed license information without validation.

#### Request

```http
GET /v1/licenses/status?license_key=SCLW-PRO-1234567890ABCDEF
Authorization: Bearer [admin_token]
```

#### Response

```http
HTTP/1.1 200 OK

{
  "license_key": "SCLW-PRO-1234567890ABCDEF",
  "tier": "pro",
  "status": "active",
  "created_at": "2026-01-05T00:00:00Z",
  "expires_at": "2026-03-05T00:00:00Z",
  "features": [
    "slack-adapter",
    "discord-adapter",
    "pii-scrubber"
  ],
  "usage": {
    "validations_this_month": 1523,
    "last_validation": "2026-02-05T10:30:00Z",
    "instances": [
      {
        "instance_id": "550e8400-e29b-41d4-a716-446655440000",
        "first_seen": "2026-01-10T00:00:00Z",
        "last_seen": "2026-02-05T10:30:00Z"
      }
    ]
  }
}
```

---

### POST /licenses/activate

Activate a new license key (first-time setup).

#### Request

```http
POST /v1/licenses/activate

{
  "license_key": "SCLW-PRO-1234567890ABCDEF",
  "email": "user@example.com",
  "instance_id": "550e8400-e29b-41d4-a716-446655440000",
  "hostname": "production-server",
  "version": "1.0.0"
}
```

#### Response

```http
HTTP/1.1 200 OK

{
  "activated": true,
  "tier": "pro",
  "features": ["slack-adapter", "discord-adapter"],
  "expires_at": "2026-03-05T00:00:00Z"
}
```

---

## Feature Catalog

### Free Tier Features (No License Required)

| Feature | Description |
|---------|-------------|
| `matrix-adapter` | Matrix Conduit integration |
| `keystore` | Encrypted credential storage |
| `basic-validation` | Message size/format validation |
| `offline-queue` | Message buffering during outages |

### Pro Tier Features ($9-29/mo)

| Feature | Description |
|---------|-------------|
| `slack-adapter` | Slack Enterprise integration |
| `discord-adapter` | Discord bot integration |
| `pii-scrubber` | PII redaction (basic) |
| `audit-log` | Basic audit logging (30-day retention) |
| `priority-support` | Email support within 24h |

### Enterprise Features (Custom)

| Feature | Description |
|---------|-------------|
| `whatsapp-adapter` | WhatsApp Business API |
| `teams-adapter` | Microsoft Teams integration |
| `pii-scrubber-hipaa` | HIPAA-compliant PII scrubbing |
| `audit-log-compliance` | Full compliance logging (90+ day) |
| `sso-integration` | SAML/OIDC single sign-on |
| `crm-salesforce` | Salesforce integration |
| `crm-hubspot` | HubSpot integration |
| `custom-adapter` | Custom protocol adapter |
| `on-premise-deploy` | On-premise deployment option |
| `dedicated-support` | Dedicated support channel |

---

## License Key Format

```
SCLW-TIER-XXXXXXXXXXXXXXXX

Components:
- SCLW: Product identifier
- TIER: FREE, PRO, ENT (Enterprise)
- XXXXXXXXXXXXXXXX: 16-character unique identifier (hex)
```

**Example:** `SCLW-PRO-A1B2C3D4E5F67890`

---

## Client Behavior Specification

### Caching Strategy

```go
type LicenseCache struct {
    sync.RWMutex
    licenses map[string]*CachedLicense
}

type CachedLicense struct {
    Valid          bool
    Tier           string
    Features       []string
    ExpiresAt      time.Time    // Server-specified expiration
    CachedAt       time.Time    // When we cached this
    GraceUntil     time.Time    // Offline grace period
}

func (c *LicenseCache) ShouldRefresh(feature string) bool {
    c.RLock()
    defer c.RUnlock()

    cached, exists := c.licenses[feature]
    if !exists {
        return true  // No cache, need to fetch
    }

    // If still valid, no refresh needed
    if time.Now().Before(cached.ExpiresAt) {
        return false
    }

    // Within grace period? Use cached value
    if time.Now().Before(cached.GraceUntil) {
        return false
    }

    // Grace period expired, must refresh
    return true
}
```

### Grace Period Handling

```
Server Response Timeline:

|─────────────────────────────────────────────────────────────────|
T0                           T1                          T2       T3
│                            │                           │        │
Cache Obtained         Expiration              Grace Period     Must
from Server             (per server)            Ends            Refresh

T0: Client validates license, caches result
T1: Cached license expires (per server's expires_at)
T1-T2: Grace period (3 days default) - offline operation OK
T2: Grace period ends, client must contact server
T3: If server unreachable, feature is DISABLED
```

### Offline Behavior

```go
func (c *Client) Validate(feature string) (bool, error) {
    cached := c.cache.Get(feature)

    // Check cache first
    if cached != nil && cached.IsValid() {
        return cached.Valid, nil
    }

    // In grace period?
    if cached != nil && time.Now().Before(cached.GraceUntil) {
        log.Printf("Using grace period for %s", feature)
        return cached.Valid, nil
    }

    // Must validate with server
    result, err := c.callServer(feature)
    if err != nil {
        // Server unreachable
        if cached != nil {
            // Extended grace for temporary outages
            return cached.Valid, fmt.Errorf("server unreachable, using cached")
        }
        return false, fmt.Errorf("no cached license and server unreachable")
    }

    // Update cache
    c.cache.Set(feature, result)
    return result.Valid, nil
}
```

---

## Security Considerations

### License Key Security

- **Transmission:** Always over HTTPS
- **Storage:** Encrypted at rest in bridge keystore
- **Validation:** Server validates signature/key integrity
- **Rotation:** Support for key rotation without reactivation

### Privacy

- **Minimal data:** Only license key, instance ID, feature requested
- **No payloads:** Message content never sent to license server
- **Opt-out telemetry:** Users can disable usage tracking
- **GDPR compliant:** Data retention policies documented

### Rate Limiting

```
Free tier:    100 requests/hour
Pro tier:     1000 requests/hour
Enterprise:   Unlimited
```

---

## Telemetry (Optional)

### POST /telemetry/heartbeat

Anonymous usage heartbeat (opt-out).

```http
POST /v1/telemetry/heartbeat

{
  "instance_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "features_used": ["matrix-adapter", "slack-adapter"],
  "memory_mb": 150,
  "active_agents": 3
}
```

**Response:** `204 No Content`

---

## Implementation Priority

### Phase 1 (Week 7-8)
- [ ] Basic validation endpoint
- [ ] License key generation system
- [ ] Database schema (PostgreSQL)
- [ ] Admin dashboard (basic)

### Phase 2 (After Launch)
- [ ] Usage analytics
- [ ] Self-service portal
- [ ] Automated billing integration
- [ ] License key rotation

### Phase 3 (Growth)
- [ ] Feature usage insights
- [ ] A/B testing for pricing
- [ ] Churn prediction
- [ ] Automated renewal reminders

---

## Database Schema (PostgreSQL)

```sql
-- Licenses table
CREATE TABLE licenses (
    id SERIAL PRIMARY KEY,
    license_key VARCHAR(255) UNIQUE NOT NULL,
    tier VARCHAR(50) NOT NULL,
    customer_email VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    metadata JSONB
);

-- Features table
CREATE TABLE features (
    id SERIAL PRIMARY KEY,
    feature_key VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tier VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- License features (many-to-many)
CREATE TABLE license_features (
    license_id INTEGER REFERENCES licenses(id),
    feature_id INTEGER REFERENCES features(id),
    granted_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (license_id, feature_id)
);

-- Instances (bridge installations)
CREATE TABLE instances (
    id SERIAL PRIMARY KEY,
    instance_id UUID UNIQUE NOT NULL,
    license_id INTEGER REFERENCES licenses(id),
    hostname VARCHAR(255),
    first_seen TIMESTAMP DEFAULT NOW(),
    last_seen TIMESTAMP DEFAULT NOW(),
    version VARCHAR(50),
    metadata JSONB
);

-- Validations log
CREATE TABLE validations (
    id SERIAL PRIMARY KEY,
    instance_id UUID REFERENCES instances(id),
    feature_key VARCHAR(100),
    validated_at TIMESTAMP DEFAULT NOW(),
    was_valid BOOLEAN,
    error_code VARCHAR(100)
);

-- Indexes
CREATE INDEX idx_licenses_key ON licenses(license_key);
CREATE INDEX idx_licenses_email ON licenses(customer_email);
CREATE INDEX idx_validations_instance ON validations(instance_id);
CREATE INDEX idx_validations_feature ON validations(feature_key);
```

---

## Admin API (Internal)

### Generate License Key

```http
POST /admin/v1/licenses
Authorization: Bearer [admin_token]

{
  "tier": "pro",
  "email": "customer@example.com",
  "duration_days": 30,
  "features": ["slack-adapter", "discord-adapter"]
}

Response:
{
  "license_key": "SCLW-PRO-A1B2C3D4E5F67890",
  "expires_at": "2026-03-07T00:00:00Z"
}
```

### Revoke License

```http
DELETE /admin/v1/licenses/SCLW-PRO-A1B2C3D4E5F67890
Authorization: Bearer [admin_token]

Response:
{
  "revoked": true,
  "revoked_at": "2026-02-05T10:30:00Z"
}
```

---

## Testing Strategy

### Unit Tests
- License key format validation
- Cache expiration logic
- Grace period calculations
- Feature matching logic

### Integration Tests
- Mock license server responses
- Offline behavior simulation
- Cache invalidation
- Concurrent validation requests

### Load Tests
- 1000 concurrent validations
- Sustained 100 QPS for 1 hour
- Failure recovery (server restart)

---

**Status:** Ready for implementation
**Priority:** P1 (Required for Pro tier)
**Dependencies:** PostgreSQL database
**Timeline:** Week 7-8 (Phase 4)
