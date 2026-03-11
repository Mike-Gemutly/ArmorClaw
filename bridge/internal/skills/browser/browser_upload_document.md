---
name: browser_upload_document
description: Upload a document to a web form
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: medium
approval_policy: auto
---

# browser_upload_document

Upload a document file to a web form.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| url | string | Yes | Upload page URL |
| file_input_selector | string | Yes | CSS selector for file input |
| file_path | string | Yes | Path to file to upload |
| submit_selector | string | No | CSS selector for submit button |

## Implementation

Composed workflow combining:
- navigate_page
- upload_file
- click (submit if provided)

## Approval Policy

- **auto**: Medium risk - file upload

## Risk Level

- **medium**: File upload operation