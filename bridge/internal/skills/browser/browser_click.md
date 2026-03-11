---
name: browser_click
description: Click an element in the browser by selector
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_click

Click an element in the browser page by CSS or XPath selector.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| selector | string | Yes | CSS or XPath selector for the element |
| timeout | number | No | Wait timeout in ms (default: 5000) |

## Implementation

Maps to Chrome DevTools MCP `click` tool.

## Approval Policy

- **auto**: Low risk - simple click operation

## Risk Level

- **low**: Simple click operation