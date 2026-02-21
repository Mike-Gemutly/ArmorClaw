# ArmorClaw Platform Adapter Roadmap

> **Last Updated:** 2026-02-19
> **Version:** 4.4.0
> **Status:** Slack Edition - Production Ready

---

## Current Platform Support

| Platform | Status | Implementation | Notes |
|----------|--------|----------------|-------|
| **Slack** | ✅ Live | `internal/adapter/slack.go` | Full API support, message queuing, rate limiting |
| **Discord** | ··· Planned | - | Similar API patterns to Slack |
| **Teams** | ··· Planned | - | Graph API complexity |
| **WhatsApp** | ··· Planned | - | Business API restrictions |

---

## Platform Adapter Priority Order

### 1. Discord (Recommended Next) - Priority: HIGH

**Rationale:**
- API structure similar to Slack ( WebSocket-based, message events)
- Well-documented Discord Go SDK available
- Large enterprise adoption
- No business verification required

**Estimated Effort:** 2-3 weeks

**Key Implementation Tasks:**
- [ ] Create `internal/adapter/discord.go`
- [ ] Implement Discord Go SDK integration
- [ ] Add gateway connection management
- [ ] Implement message queuing (reuse existing queue)
- [ ] Add slash command support
- [ ] Implement rate limiting (Discord has strict limits)
- [ ] Add bot permissions configuration
- [ ] Write integration tests

**Configuration Schema:**
```toml
[discord]
enabled = true
bot_token = ""         # Discord bot token
application_id = ""    # Discord application ID
guild_id = ""          # Server ID to monitor
matrix_room = ""       # Destination Matrix room
```

---

### 2. Microsoft Teams - Priority: MEDIUM

**Rationale:**
- Enterprise demand (Microsoft 365 integration)
- Graph API is well-documented
- Requires Azure AD app registration

**Estimated Effort:** 3-4 weeks

**Key Challenges:**
- Graph API complexity (multiple endpoints)
- Azure AD authentication flow
- Webhook vs. Bot Framework decision
- Tenant-specific configuration

**Key Implementation Tasks:**
- [ ] Create `internal/adapter/teams.go`
- [ ] Implement Microsoft Graph SDK integration
- [ ] Add Azure AD OAuth2 flow
- [ ] Implement webhook/bot message handling
- [ ] Add Teams-specific message formatting
- [ ] Handle threaded conversations
- [ ] Write integration tests

**Configuration Schema:**
```toml
[teams]
enabled = true
tenant_id = ""         # Azure AD tenant ID
client_id = ""         # Azure AD app client ID
client_secret = ""     # Azure AD app secret
team_id = ""           # Teams team ID
channel_id = ""        # Channel to monitor
matrix_room = ""       # Destination Matrix room
```

---

### 3. WhatsApp Business - Priority: LOW

**Rationale:**
- Business API has strict requirements
- Meta verification process required
- Pricing per conversation

**Estimated Effort:** 4-6 weeks

**Key Challenges:**
- WhatsApp Business API access approval
- Webhook verification requirements
- Message templates approval process
- Per-conversation pricing
- 24-hour messaging window restrictions

**Key Implementation Tasks:**
- [ ] Create `internal/adapter/whatsapp.go`
- [ ] Implement WhatsApp Business SDK
- [ ] Add webhook verification
- [ ] Implement message template management
- [ ] Handle 24-hour messaging windows
- [ ] Add media message support
- [ ] Write integration tests

**Configuration Schema:**
```toml
[whatsapp]
enabled = true
phone_number_id = ""   # WhatsApp business phone ID
access_token = ""      # Meta access token
verify_token = ""      # Webhook verification token
business_account_id = "" # WhatsApp business account ID
matrix_room = ""       # Destination Matrix room
```

---

## Adapter Architecture

All platform adapters follow the unified interface defined in `internal/adapter/`:

```go
type PlatformAdapter interface {
    // Connection management
    Connect(ctx context.Context) error
    Disconnect() error
    IsConnected() bool

    // Messaging
    SendMessage(ctx context.Context, channelID, text string, opts ...MessageOption) (*MessageResult, error)
    GetMessages(ctx context.Context, channelID string, limit int) ([]*PlatformMessage, error)

    // Platform info
    Platform() string
    GetChannels(ctx context.Context) ([]*PlatformChannel, error)
}
```

---

## Implementation Guidelines

### Code Location
- Platform adapters: `bridge/internal/adapter/<platform>.go`
- Configuration: Add to `bridge/pkg/config/config.go`
- Tests: `bridge/internal/adapter/<platform>_test.go`

### Required Features
1. **Connection Management**
   - Reconnection with exponential backoff
   - Health monitoring
   - Graceful shutdown

2. **Message Handling**
   - Rate limiting (platform-specific)
   - Message queuing (reuse `internal/queue/`)
   - Retry logic with circuit breaker

3. **Security**
   - Token/credential validation
   - Input sanitization
   - Audit logging

4. **Testing**
   - Unit tests with mocks
   - Integration tests (optional, requires API access)

---

## Release Timeline

| Version | Platform | Target Date |
|---------|----------|-------------|
| v4.4.0 | Slack (Live) | Current |
| v4.5.0 | Discord | TBD |
| v5.0.0 | Teams | TBD |
| v5.1.0 | WhatsApp | TBD |

---

## Contributing

To implement a new platform adapter:

1. Create a feature branch: `feature/<platform>-adapter`
2. Study the existing Slack adapter as reference
3. Implement the `PlatformAdapter` interface
4. Add configuration support
5. Write comprehensive tests
6. Update documentation

---

**Roadmap Owner:** ArmorClaw Team
**Contact:** https://github.com/armorclaw/armorclaw/issues
