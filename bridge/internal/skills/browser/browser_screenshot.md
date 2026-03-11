---
name: browser_screenshot
description: Take a screenshot of the current page
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_screenshot

Capture a screenshot of the current browser page.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| full_page | boolean | No | Capture full scrollable page (default: false) |
| format | string | No | 'png' or 'jpeg' (default: png) |
| quality | number | No | JPEG quality 0-100 (default: 80) |

## Implementation

Maps to Chrome DevTools MCP `take_screenshot` tool.

## Approval Policy

- **auto**: Low risk - read-only capture

## Risk Level

- **low**: Screenshot capture is read-only