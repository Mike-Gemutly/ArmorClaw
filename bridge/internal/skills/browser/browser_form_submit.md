---
name: browser_form_submit
description: Fill and submit a web form
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: medium
approval_policy: auto
---

# browser_form_submit

Fill and submit a web form with validation.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| url | string | Yes | Form page URL |
| fields | object | Yes | Key-value pairs of field selectors to values |
| submit_selector | string | Yes | CSS selector for submit button |
| wait_for | string | No | Selector to wait for after submission |

## Implementation

Composed workflow combining:
- navigate_page
- fill_form
- upload_file (if file fields present)
- click (submit)
- handle_dialog (if present)
- wait_for (confirmation)

## Approval Policy

- **auto**: Medium risk - form submission

## Risk Level

- **medium**: Form data submission