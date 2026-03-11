---
name: browser_fill_with_pii
description: Fill form fields with PII data (highly privileged)
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: critical
approval_policy: required
---

# browser_fill_with_pII

Fill form fields with personally identifiable information (PII). HIGHEST RISK - requires explicit approval.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| fields | object | Yes | Key-value pairs of selector to PII field name |
| pii_source | string | No | Source: keystore, environment (default: keystore) |
| redaction_enabled | boolean | No | Enable redaction in logs (default: true) |

## Implementation

Maps to Chrome DevTools MCP `fill` tool with PII access.

## Approval Policy

- **required**: Human-in-the-loop required with explicit PII acknowledgment

## Risk Level

- **critical**: Direct PII handling

## Security Notes

- Requires explicit PII handling acknowledgment
- All operations are logged with mandatory redaction
- PII is fetched from secure keystore only
- Session data is cleared after operation