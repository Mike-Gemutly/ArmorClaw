# 🧭 Jetski Chartmaker

**Visual navigation compiler for Jetski browser automation**

> *Chart your course through the digital seas with precision and confidence.*

## What is Jetski Chartmaker?

Jetski Chartmaker is a Node.js CLI tool that allows developers to visually navigate websites and compile interactions into resilient **Nav-Charts** (`.acsb.json` files). These charts provide mathematical DOM paths for the Jetski browser's Go RPC Shield, enabling 10x faster execution with 10x less memory than traditional browser automation.

### Key Features

- **🗺️ Visual Recording**: Interactively record clicks, inputs, and assertions with The Helm HUD
- **🎯 3-Tier Selectors**: Resilient DOM selectors (CSS → XPath → JS) that survive layout drift
- **🔒 Shadow DOM Support**: Full support for Web Components and Shadow DOM piercing
- **🚀 Session Persistence**: Maintain login state across recording sessions
- **✅ Schema Validation**: JSON schema validation ensures Nav-Chart integrity

## Installation

```bash
npm install -g @armorclaw/jetski-chartmaker
```

## Quick Start

### 1. Initialize Your Project

```bash
jetski-chartmaker init
```

This creates the project structure:
```
charts/         # Nav-Charts directory
sessions/       # Browser session storage
jetski.config.json
```

### 2. Record a Nav-Chart

```bash
jetski-chartmaker map --url https://example.com
```

This launches a browser with **The Helm** HUD visible in the top-right corner. Click elements, fill forms, and set assertions as you navigate the site. The HUD records each interaction and generates a 3-tier selector matrix.

### 3. Save Your Nav-Chart

Click **💾 Save Nav-Chart** in The Helm to generate your `.acsb.json` file.

### 4. Verify Your Nav-Chart

```bash
jetski-chartmaker verify site.acsb.json --verbose
```

Run a dry-run local sea trial to validate your Nav-Chart works correctly.

## CLI Commands

### `jetski-chartmaker init`

Initialize a Jetski Chartmaker project.

**Options:**
- `--dir <path>`: Project directory (default: `.`)

### `jetski-chartmaker map`

Start an interactive mapping session.

**Options:**
- `--url <url>`: Starting URL
- `--output <file>`: Output Nav-Chart file (default: `site.acsb.json`)
- `--session <id>`: Session identifier for persistence

### `jetski-chartmaker verify`

Validate a Nav-Chart with dry-run execution.

**Arguments:**
- `<file>`: `.acsb.json` Nav-Chart file to verify

**Options:**
- `--verbose`: Detailed output
- `--headless`: Run in headless mode

## The Helm (HUD)

The Helm is a draggable, Shadow DOM-isolated HUD that provides:

- **Recording Modes**: Click, Input, Assertion
- **Live Status Updates**: Real-time feedback on recorded actions
- **Save/Clear Actions**: Manage your Nav-Chart easily
- **Isolated Styles**: No interference with page styles

## Nav-Chart Format

Nav-Charts are JSON files with the following structure:

```json
{
  "version": 1,
  "target_domain": "https://example.com",
  "metadata": {
    "generated_by": "@armorclaw/jetski-chartmaker",
    "timestamp": "2024-01-01T12:00:00.000Z",
    "session_id": "session-123"
  },
  "action_map": {
    "login": {
      "action_type": "click",
      "selector": {
        "primary_css": "[data-automation-id='login-button']",
        "secondary_xpath": "//*[@data-automation-id='login-button']",
        "fallback_js": "document.querySelector('[data-automation-id=\"login-button\"]')"
      },
      "post_action_wait": {
        "type": "waitForVisible",
        "selector": {
          "primary_css": ".dashboard"
        },
        "timeout": 5000
      }
    }
  }
}
```

## 3-Tier Selector Matrix

Jetski Chartmaker generates resilient selectors with three fallback tiers:

1. **Tier 1 (Primary)**: CSS selector using `data-automation-id` attributes
2. **Tier 2 (Secondary)**: XPath selector for robustness
3. **Tier 3 (Fallback)**: JavaScript expression for ultimate resilience

This ensures your Nav-Charts work even when page layouts change.

## Session Persistence

Browser sessions are persisted to disk, allowing you to:
- Maintain login state across recordings
- Work with MFA-protected sites
- Resume recording after browser restarts

Session data is stored in `sessions/<session-id>/`.

## Development

```bash
# Install dependencies
npm install

# Run in development mode
npm run dev

# Build
npm run build

# Run tests
npm test
```

## Technical Constraints

- **No Web Workers**: Injected scripts use vanilla JS in a single string
- **100ms Throttle**: Events are batched for performance
- **Shadow DOM Isolation**: HUD styles don't leak into the page
- **Cross-Origin Detection**: Identifies locked-down iframes early

## Contributing

We welcome contributions! Please check our [Contributing Guidelines](CONTRIBUTING.md) for details.

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Links

- [Jetski Browser](https://github.com/armorclaw/jetski)
- [ArmorClaw Platform](https://armorclaw.io)

---

**⚓ Chart your course, navigate with confidence.**
