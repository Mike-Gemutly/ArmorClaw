---
name: browser_lighthouse_audit
description: Run Lighthouse performance audit (privileged)
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: medium
approval_policy: required
---

# browser_lighthouse_audit

Run a Lighthouse performance audit on a page.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| url | string | Yes | URL to audit |
| categories | array | No | Categories: performance, accessibility, best-practices, seo (default: all) |
| format | string | No | Output format: json, html (default: json) |

## Implementation

Maps to Chrome DevTools MCP `lighthouse_audit` tool.

## Approval Policy

- **required**: Human-in-the-loop required

## Risk Level

- **medium**: Runs external audit tool, may capture sensitive page data

## Security Notes

- Audit results may contain page screenshots
- May reveal performance-related sensitive data