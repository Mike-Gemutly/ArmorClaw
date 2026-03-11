---
name: browser_console_inspect
description: Inspect browser console logs (privileged)
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: medium
approval_policy: required
---

# browser_console_inspect

Inspect browser console logs and messages.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| level | string | No | Filter by level: log, warn, error, debug (default: all) |
| count | number | No | Number of recent messages (default: 50) |

## Implementation

Maps to Chrome DevTools console domain.

## Approval Policy

- **required**: Human-in-the-loop required

## Risk Level

- **medium**: Can expose JavaScript errors, warnings, debug info

## Security Notes

- May expose sensitive data in console logs
- May reveal application internal details