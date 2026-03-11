---
name: browser_resize
description: Resize the browser viewport
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_resize

Resize the browser viewport to specified dimensions.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| width | number | Yes | Viewport width in pixels |
| height | number | Yes | Viewport height in pixels |
| device_scale | number | No | Device scale factor (default: 1) |

## Implementation

Maps to Chrome DevTools MCP `resize_page` tool.

## Approval Policy

- **auto**: Low risk - viewport resize

## Risk Level

- **low**: Viewport configuration