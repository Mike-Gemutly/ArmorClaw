---
name: browser_login_assist
description: Assist with web form login flow
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: medium
approval_policy: auto
---

# browser_login_assist

Assist with web login form filling and submission.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| url | string | Yes | Login page URL |
| username_selector | string | Yes | CSS selector for username field |
| password_selector | string | Yes | CSS selector for password field |
| username | string | Yes | Username value |
| password | string | Yes | Password value (will be redacted in logs) |
| submit_selector | string | No | CSS selector for submit button |

## Implementation

Composed workflow combining:
- navigate_page
- fill (username)
- fill (password)
- click (submit)
- wait_for (post-login element)
- take_screenshot (on failure)

## Approval Policy

- **auto**: Medium risk - credential handling

## Risk Level

- **medium**: Involves credential input

## Security Notes

- Password is redacted in logs
- Screenshot on failure helps debugging without exposing credentials