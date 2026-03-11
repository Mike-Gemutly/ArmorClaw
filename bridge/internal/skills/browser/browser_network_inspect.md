---
name: browser_network_inspect
description: Inspect network requests and responses (privileged)
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: high
approval_policy: required
---

# browser_network_inspect

Inspect network requests and responses. HIGH RISK - can expose tokens, cookies, and sensitive data.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| request_id | string | No | Specific request to inspect (default: list all) |
| filter | string | No | URL pattern to filter requests |

## Implementation

Maps to Chrome DevTools MCP `get_network_request` and `list_network_requests` tools.

## Approval Policy

- **required**: Human-in-the-loop required

## Risk Level

- **high**: Can expose auth tokens, cookies, request bodies, API keys

## Security Notes

- Can read Authorization headers
- Can read request/response bodies
- Can expose API keys in URLs
- All inspections are logged with redaction