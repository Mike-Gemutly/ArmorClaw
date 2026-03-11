---
name: browser_fill_form
description: Fill multiple form fields at once
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: medium
approval_policy: auto
---

# browser_fill_form

Fill multiple form fields in a single operation.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| fields | object | Yes | Key-value pairs of selector to value |
| submit | boolean | No | Auto-submit form after filling (default: false) |

## Implementation

Maps to Chrome DevTools MCP `fill_form` tool. Combines multiple fill operations.

## Approval Policy

- **auto**: Medium risk - writes to multiple fields

## Risk Level

- **medium**: Multiple field input