# ArmorClaw SDTW Egress Proxy

This Squid proxy enables ArmorClaw's SDTW (Slack, Discord, Teams, WhatsApp) adapters to make secure HTTPS requests to external platform APIs, resolving the critical network access contradiction where containers have no network interfaces but SDTW adapters require outbound HTTPS.

## Quick Start

### 1. Start Services
```bash
cd E:/Micha/.LocalCode/ArmorClaw/deploy/squid
docker-compose up -d
```

### 2. Configure SDTW Adapters
Each SDTW adapter needs to be configured to use the appropriate proxy:

**Slack Adapter:**
```bash
# Set proxy for Slack adapter
export sdtw_slack_proxy="http://squid:3128:8080/slack"

# Verify configuration
curl http://squid:3128:8080/cache_object?url=http://squid:3128:8080/slack
```

**Discord Adapter:**
```bash
# Set proxy for Discord adapter
export sdtw_discord_proxy="http://squid:3128:8081/discord"

# Verify configuration
curl http://squid:3128:8081/cache_object?url=http://squid:3128:8081/discord
```

**Teams Adapter:**
```bash
# Set proxy for Teams adapter
export sdtw_teams_proxy="http://squid:3128:8082/teams"

# Verify configuration
curl http://squid:3128:8082/cache_object?url=http://squid:3128:8082/teams
```

**WhatsApp Adapter:**
```bash
# Set proxy for WhatsApp adapter
export sdtw_whatsapp_proxy="http://squid:3128:8083/whatsapp"

# Verify configuration
curl http://squid:3128:8083/cache_object?url=http://squid:3128:8083/whatsapp
```

### 3. Verify Connectivity
```bash
# Test Slack adapter through proxy
curl -x http://slack.com/api/chat.postMessage -H "Authorization: Bearer $OPENAI_API_KEY" -d '{"text":"test","channel":"general"}'

# Test Discord adapter through proxy
curl -x http://discord.com/api/v10/channels/12345678/messages -H "Authorization: Bot $DISCORD_BOT_TOKEN" -d '{"content":"test"}'
```

## Proxy Configuration

The Squid proxy is configured with:
- **ACLs** allowing HTTP requests from ArmorClaw containers
- **Authentication**: Basic proxy authentication
- **Cache denial** for dynamic content (prevents cache poisoning)
- **Rate limiting** to prevent abuse
- **Logging** of all proxy requests

## Troubleshooting

### Proxy Not Working
```bash
# Check Squid status
docker-compose logs squid

# Test direct connection
curl -x http://squid:3128:8080 -v

# Check ACL
grep "cache_object.*http" /e/Micha/.LocalCode/ArmorClaw/deploy/squid/squid.conf

# Restart proxy
docker-compose restart squid
```

## Security Notes

- ✅ No direct network access from container (max_fdescriptors: 256)
- ✅ ACL-restricted source networks (localnet only)
- ✅ Cache deny all (prevents attacks)
- ✅ Proxy authentication required
- ✅ Rate limiting enabled