---
name: browser_memory_snapshot
description: Take a memory heap snapshot (privileged)
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: high
approval_policy: required
---

# browser_memory_snapshot

Take a memory heap snapshot for debugging memory issues.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| include_objects | boolean | No | Include object references (default: false) |

## Implementation

Maps to Chrome DevTools MCP `take_memory_snapshot` tool.

## Approval Policy

- **required**: Human-in-the-loop required

## Risk Level

- **high**: Can expose sensitive object data in memory

## Security Notes

- Heap snapshots can contain sensitive data in memory
- Object references may expose internal application state
- Should be used only for debugging