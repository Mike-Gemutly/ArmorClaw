---
name: browser_eval_privileged
description: Execute JavaScript in browser context (privileged)
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: high
approval_policy: required
---

# browser_eval_privileged

Execute arbitrary JavaScript in the browser context. HIGH RISK - can access/modify page state and sensitive data.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| script | string | Yes | JavaScript code to execute |
| timeout | number | No | Execution timeout in ms (default: 5000) |

## Implementation

Maps to Chrome DevTools MCP `evaluate_script` tool.

## Approval Policy

- **required**: Human-in-the-loop required

## Risk Level

- **high**: Can access/modify page state, DOM, session data, cookies

## Security Notes

- Can read sensitive DOM content
- Can modify page state
- Can access cookies and localStorage
- Can execute arbitrary code in browser context
- All executions are logged with redaction