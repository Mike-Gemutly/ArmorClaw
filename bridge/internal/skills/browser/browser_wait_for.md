---
name: browser_wait_for
description: Wait for an element to appear or condition to be met
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_wait_for

Wait for an element to be present or a condition to be met.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| selector | string | Yes | CSS selector to wait for |
| state | string | No | 'visible', 'hidden', 'attached' (default: visible) |
| timeout | number | No | Wait timeout in ms (default: 30000) |

## Implementation

Maps to Chrome DevTools MCP `wait_for` tool.

## Approval Policy

- **auto**: Low risk - waiting operation

## Risk Level

- **low**: Read-only wait operation