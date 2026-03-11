---
name: browser_trace_performance
description: Measure and analyze page performance
homepage: https://github.com/ChromeDevTools/chrome-devtools-mcp
domain: browser
risk: low
approval_policy: auto
---

# browser_trace_performance

Navigate to a page and measure its performance using Chrome DevTools tracing.

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| url | string | Yes | URL to analyze |
| duration | number | No | Trace duration in ms (default: 10000) |
| categories | array | No | Trace categories to enable |

## Implementation

Composed workflow combining:
- navigate_page
- performance_start_trace
- wait (duration)
- performance_stop_trace
- performance_analyze_insight

## Approval Policy

- **auto**: Low risk - performance measurement

## Risk Level

- **low**: Performance analysis is read-only