---
name: browser_select_page
description: Switch to a different browser page/tab
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_select_page

Switch focus to a different browser page or tab.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| page_id | string | Yes | The page ID to switch to |

## Implementation

Maps to Chrome DevTools MCP `select_page` tool.

## Approval Policy

- **auto**: Low risk - tab switching

## Risk Level

- **low**: Tab switching operation