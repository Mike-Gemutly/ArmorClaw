---
name: browser_snapshot
description: Take a DOM snapshot of the current page
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_snapshot

Capture a complete DOM snapshot of the current page.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| include_styles | boolean | No | Include computed styles (default: false) |

## Implementation

Maps to Chrome DevTools MCP `take_snapshot` tool.

## Approval Policy

- **auto**: Low risk - read-only DOM capture

## Risk Level

- **low**: DOM snapshot is read-only but may contain sensitive data