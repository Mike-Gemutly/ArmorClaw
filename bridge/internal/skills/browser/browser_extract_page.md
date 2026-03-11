---
name: browser_extract_page
description: Extract structured data from a web page
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_extract_page

Extract structured data from a web page using navigation, wait, and snapshot.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| url | string | Yes | URL to navigate to |
| selectors | array | Yes | CSS selectors for data extraction |
| wait_for | string | No | Selector to wait for before extraction |
| timeout | number | No | Navigation timeout (default: 30000) |

## Implementation

Composed workflow combining:
- navigate_page
- wait_for
- take_snapshot

## Approval Policy

- **auto**: Low risk - read-only data extraction

## Risk Level

- **low**: Read-only extraction workflow