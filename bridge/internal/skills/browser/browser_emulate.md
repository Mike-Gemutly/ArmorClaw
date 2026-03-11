---
name: browser_emulate
description: Emulate device characteristics
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_emulate

Emulate device characteristics (user agent, viewport, touch, etc.).

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| device | string | Yes | Device name (e.g., 'iPhone 14', 'Pixel 7') |
| viewport | object | No | Custom viewport override |
| user_agent | string | No | Custom user agent string |
| touch | boolean | No | Enable touch emulation (default: true) |

## Implementation

Maps to Chrome DevTools MCP `emulate` tool.

## Approval Policy

- **auto**: Low risk - device emulation

## Risk Level

- **low**: Device emulation configuration