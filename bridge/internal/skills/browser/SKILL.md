---
name: browser_navigate
description: Navigate to a URL in the browser
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_navigate

Navigate to a URL in the browser session.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| url | string | Yes | The URL to navigate to |
| timeout | number | No | Navigation timeout in ms (default: 30000) |

## Implementation

Maps to Chrome DevTools MCP `navigate_page` tool.

## Approval Policy

- **auto**: Low risk - read-only navigation

## Risk Level

- **low**: Simple navigation operation