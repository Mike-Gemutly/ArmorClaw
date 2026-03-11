---
name: browser_fill
description: Fill an input field in the browser
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_fill

Fill an input field with text content.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| selector | string | Yes | CSS selector for the input element |
| value | string | Yes | Text value to fill |
| clear | boolean | No | Clear before filling (default: true) |

## Implementation

Maps to Chrome DevTools MCP `fill` tool.

## Approval Policy

- **auto**: Low risk - text input operation

## Risk Level

- **low**: Simple text input