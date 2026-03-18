# Browser Service Implementation Guide

> **Created:** 2026-02-28
> **Status:** Implementation Specification
> **Prerequisites:** Docker, Node.js 20+, Playwright knowledge

---

## Overview

The Browser Service is a standalone Node.js application that runs Playwright browsers with anti-detection measures. The Bridge (Go) communicates with it via HTTP to execute browser automation tasks.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              ARCHITECTURE                                        │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   Bridge (Go)                     Browser Service (Node.js)                     │
│   ───────────                     ────────────────────────                      │
│                                                                                  │
│   ┌───────────────┐    HTTP      ┌─────────────────────────────────────────┐   │
│   │               │   :3000      │                                         │   │
│   │  Job Queue    │ ──────────► │  ┌─────────────────────────────────┐    │   │
│   │               │              │  │  Session Manager                │    │   │
│   │  Browser      │ ◄──────────  │  │  - Create/destroy sessions      │    │   │
│   │  Skill        │   JSON       │  │  - Track active browsers        │    │   │
│   │               │              │  └─────────────────────────────────┘    │   │
│   └───────────────┘              │                  │                       │   │
│                                  │                  ▼                       │   │
│                                  │  ┌─────────────────────────────────┐    │   │
│                                  │  │  Command Executor               │    │   │
│                                  │  │  - Navigate, Fill, Click        │    │   │
│                                  │  │  - Wait, Extract, Screenshot    │    │   │
│                                  │  └─────────────────────────────────┘    │   │
│                                  │                  │                       │   │
│                                  │                  ▼                       │   │
│                                  │  ┌─────────────────────────────────┐    │   │
│                                  │  │  Stealth Layer                  │    │   │
│                                  │  │  - Fingerprint spoofing         │    │   │
│                                  │  │  - Human-like behavior          │    │   │
│                                  │  │  - Detection evasion            │    │   │
│                                  │  └─────────────────────────────────┘    │   │
│                                  │                  │                       │   │
│                                  │                  ▼                       │   │
│                                  │  ┌─────────────────────────────────┐    │   │
│                                  │  │  Playwright Browser             │    │   │
│                                  │  │  - Chromium (headless)          │    │   │
│                                  │  │  - Persistent context           │    │   │
│                                  │  └─────────────────────────────────┘    │   │
│                                  │                                         │   │
│                                  └─────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Project Structure

```
browser-service/
├── src/
│   ├── index.ts                 # Entry point, HTTP server
│   ├── server.ts                # Express/Fastify server setup
│   ├── routes/
│   │   ├── session.ts           # Session CRUD endpoints
│   │   ├── commands.ts          # Browser command endpoints
│   │   └── health.ts            # Health check endpoint
│   ├── services/
│   │   ├── session-manager.ts   # Browser session lifecycle
│   │   ├── command-executor.ts  # Command execution logic
│   │   └── intervention.ts      # CAPTCHA/2FA detection
│   ├── stealth/
│   │   ├── index.ts             # Stealth configuration aggregator
│   │   ├── fingerprint.ts       # WebGL, canvas, audio spoofing
│   │   ├── behavior.ts          # Human-like typing, mouse, scrolling
│   │   └── evasion.ts           # Webdriver hiding, plugin spoofing
│   ├── types/
│   │   ├── session.ts           # Session types
│   │   ├── commands.ts          # Command types
│   │   └── responses.ts         # Response types
│   └── utils/
│       ├── logger.ts            # Logging utility
│       └── crypto.ts            # Session ID generation
├── Dockerfile
├── docker-compose.yml
├── package.json
├── tsconfig.json
└── .env.example
```

---

## 2. Package Dependencies

```json
{
  "name": "armorclaw-browser-service",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "tsx watch src/index.ts",
    "build": "tsc",
    "start": "node dist/index.js",
    "test": "vitest",
    "lint": "eslint src/"
  },
  "dependencies": {
    "playwright": "^1.42.0",
    "playwright-extra": "^4.3.6",
    "puppeteer-extra-plugin-stealth": "^2.11.2",
    "fastify": "^4.26.0",
    "@fastify/cors": "^9.0.0",
    "uuid": "^9.0.0",
    "dotenv": "^16.4.0",
    "pino": "^8.18.0",
    "pino-pretty": "^10.3.0"
  },
  "devDependencies": {
    "@types/node": "^20.11.0",
    "@types/uuid": "^9.0.0",
    "typescript": "^5.3.0",
    "tsx": "^4.7.0",
    "vitest": "^1.2.0",
    "eslint": "^8.56.0"
  }
}
```

---

## 3. HTTP API Specification

### Base URL
```
http://localhost:3000
```

### Endpoints

#### Session Management

##### POST /session/create
Create a new browser session with optional configuration.

**Request:**
```json
{
  "session_id": "sess_abc123",           // Optional, auto-generated if omitted
  "user_agent": "Mozilla/5.0 ...",       // Optional, defaults to realistic Chrome
  "viewport": {                          // Optional
    "width": 1920,
    "height": 1080
  },
  "cookies": [                           // Optional, restore previous session
    {
      "name": "session",
      "value": "xyz",
      "domain": ".example.com",
      "path": "/"
    }
  ],
  "proxy": {                             // Optional
    "server": "http://proxy:8080",
    "username": "user",
    "password": "pass"
  },
  "locale": "en-US",                     // Optional
  "timezone": "America/Denver"           // Optional
}
```

**Response:**
```json
{
  "session_id": "sess_abc123",
  "status": "created",
  "created_at": 1709250000000
}
```

##### GET /session/:id
Get session status.

**Response:**
```json
{
  "session_id": "sess_abc123",
  "status": "active",
  "current_url": "https://example.com/checkout",
  "created_at": 1709250000000,
  "last_activity": 1709250060000,
  "cookies_count": 12
}
```

##### POST /session/:id/close
Close and cleanup a session.

**Response:**
```json
{
  "session_id": "sess_abc123",
  "status": "closed"
}
```

##### GET /session/:id/cookies
Export session cookies (for persistence).

**Response:**
```json
{
  "cookies": [
    {
      "name": "session",
      "value": "xyz",
      "domain": ".example.com",
      "path": "/",
      "expires": 1709336400,
      "httpOnly": true,
      "secure": true
    }
  ]
}
```

---

#### Browser Commands

##### POST /session/:id/navigate
Navigate to a URL.

**Request:**
```json
{
  "url": "https://shop.example.com/checkout",
  "wait_until": "networkidle",    // "load" | "domcontentloaded" | "networkidle"
  "timeout": 30000
}
```

**Response:**
```json
{
  "status": "success",
  "url": "https://shop.example.com/checkout",
  "title": "Checkout - Shop",
  "load_time_ms": 1234
}
```

**Error Response:**
```json
{
  "status": "error",
  "error": {
    "code": "NAVIGATION_FAILED",
    "message": "Navigation timed out after 30000ms",
    "url": "https://shop.example.com/checkout"
  }
}
```

##### POST /session/:id/fill
Fill form fields.

**Request:**
```json
{
  "fields": [
    {
      "selector": "#email",
      "value": "user@example.com",
      "type": "text",
      "clear_first": true
    },
    {
      "selector": "#card-number",
      "value": "4242424242424242",
      "type": "text",
      "humanize": true           // Use human-like typing
    },
    {
      "selector": "#cvv",
      "value": "123",
      "type": "password"
    }
  ],
  "auto_submit": false,
  "submit_selector": "#submit-btn",   // Optional, click after fill
  "submit_delay": 500                 // ms delay before submit
}
```

**Response:**
```json
{
  "status": "success",
  "fields_filled": 3,
  "duration_ms": 2340
}
```

##### POST /session/:id/click
Click an element.

**Request:**
```json
{
  "selector": "button[type=submit]",
  "wait_for": "navigation",     // "none" | "navigation" | "selector"
  "wait_selector": ".success",  // If wait_for == "selector"
  "timeout": 10000,
  "humanize": true              // Move mouse naturally before click
}
```

**Response:**
```json
{
  "status": "success",
  "clicked": true,
  "new_url": "https://shop.example.com/order/123"
}
```

##### POST /session/:id/wait
Wait for a condition.

**Request:**
```json
{
  "type": "selector",           // "selector" | "timeout" | "url" | "function"
  "value": ".order-confirmed",
  "timeout": 10000
}
```

**Response:**
```json
{
  "status": "success",
  "condition_met": true,
  "wait_time_ms": 2345
}
```

##### POST /session/:id/extract
Extract data from the page.

**Request:**
```json
{
  "fields": [
    {
      "name": "order_number",
      "selector": ".order-number",
      "attribute": "textContent"
    },
    {
      "name": "total",
      "selector": ".order-total",
      "attribute": "textContent"
    },
    {
      "name": "confirmation_link",
      "selector": "a.confirm",
      "attribute": "href"
    }
  ]
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "order_number": "ORD-12345",
    "total": "$99.00",
    "confirmation_link": "https://shop.example.com/confirm/12345"
  }
}
```

##### POST /session/:id/screenshot
Take a screenshot.

**Request:**
```json
{
  "full_page": false,
  "selector": ".checkout-form",   // Optional, crop to element
  "format": "png",                // "png" | "jpeg"
  "quality": 80                   // For JPEG
}
```

**Response:**
```json
{
  "status": "success",
  "screenshot": "data:image/png;base64,iVBORw0KGgo...",
  "width": 800,
  "height": 600
}
```

---

#### Intervention Detection

##### GET /session/:id/detect-intervention
Check if the page requires user intervention (CAPTCHA, 2FA, errors).

**Response (No intervention):**
```json
{
  "intervention_required": false
}
```

**Response (CAPTCHA detected):**
```json
{
  "intervention_required": true,
  "type": "captcha",
  "subtype": "recaptcha",
  "selectors": ["iframe[src*='recaptcha']"],
  "screenshot": "data:image/png;base64,iVBORw0KGgo...",
  "hint": "Click 'I'm not a robot' and complete the challenge"
}
```

**Response (2FA detected):**
```json
{
  "intervention_required": true,
  "type": "twofa",
  "subtype": "sms",
  "selectors": ["input[placeholder*='code']"],
  "screenshot": "data:image/png;base64,iVBORw0KGgo...",
  "hint": "Enter the 6-digit code sent to your phone",
  "input_selector": "#verification-code"
}
```

**Response (Error detected):**
```json
{
  "intervention_required": true,
  "type": "error",
  "subtype": "form_error",
  "selectors": [".error-message"],
  "screenshot": "data:image/png;base64,iVBORw0KGgo...",
  "message": "Invalid card number",
  "error_selector": ".error-message"
}
```

---

#### Health Check

##### GET /health
Service health status.

**Response:**
```json
{
  "status": "healthy",
  "uptime_seconds": 86400,
  "active_sessions": 3,
  "memory_mb": 512,
  "browser_ready": true
}
```

---

## 4. Implementation Details

### 4.1 Session Manager

```typescript
// src/services/session-manager.ts

import { chromium, Browser, BrowserContext, Page } from 'playwright';
import { v4 as uuidv4 } from 'uuid';
import { applyStealth } from '../stealth';

interface Session {
  id: string;
  browser: Browser;
  context: BrowserContext;
  page: Page;
  createdAt: Date;
  lastActivity: Date;
  config: SessionConfig;
}

interface SessionConfig {
  userAgent?: string;
  viewport?: { width: number; height: number };
  cookies?: Cookie[];
  proxy?: ProxyConfig;
  locale?: string;
  timezone?: string;
}

export class SessionManager {
  private sessions: Map<string, Session> = new Map();
  private maxSessions = 10;
  private sessionTimeout = 30 * 60 * 1000; // 30 minutes

  async createSession(config: SessionConfig = {}): Promise<Session> {
    if (this.sessions.size >= this.maxSessions) {
      // Cleanup oldest inactive session
      this.cleanupOldestSession();
    }

    const sessionId = config.sessionId || uuidv4();

    // Launch browser with stealth
    const browser = await chromium.launch({
      headless: true,
      args: [
        '--disable-blink-features=AutomationControlled',
        '--disable-features=IsolateOrigins,site-per-process',
        '--no-sandbox',
        '--disable-setuid-sandbox',
      ],
      proxy: config.proxy ? {
        server: config.proxy.server,
        username: config.proxy.username,
        password: config.proxy.password,
      } : undefined,
    });

    // Create persistent context
    const context = await browser.newContext({
      userAgent: config.userAgent || this.getDefaultUserAgent(),
      viewport: config.viewport || { width: 1920, height: 1080 },
      locale: config.locale || 'en-US',
      timezoneId: config.timezone || 'America/Denver',
      // Anti-detection
      bypassCSP: true,
      ignoreHTTPSErrors: true,
    });

    // Apply stealth plugins
    await applyStealth(context);

    // Restore cookies if provided
    if (config.cookies?.length) {
      await context.addCookies(config.cookies);
    }

    const page = await context.newPage();

    const session: Session = {
      id: sessionId,
      browser,
      context,
      page,
      createdAt: new Date(),
      lastActivity: new Date(),
      config,
    };

    this.sessions.set(sessionId, session);

    // Setup activity tracking
    page.on('request', () => {
      session.lastActivity = new Date();
    });

    return session;
  }

  getSession(sessionId: string): Session | undefined {
    return this.sessions.get(sessionId);
  }

  async closeSession(sessionId: string): Promise<void> {
    const session = this.sessions.get(sessionId);
    if (!session) return;

    await session.browser.close();
    this.sessions.delete(sessionId);
  }

  private cleanupOldestSession(): void {
    let oldest: Session | null = null;

    for (const session of this.sessions.values()) {
      if (!oldest || session.lastActivity < oldest.lastActivity) {
        oldest = session;
      }
    }

    if (oldest) {
      this.closeSession(oldest.id);
    }
  }

  private getDefaultUserAgent(): string {
    // Rotate between realistic user agents
    const agents = [
      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36',
      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36',
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36',
    ];
    return agents[Math.floor(Math.random() * agents.length)];
  }

  // Cleanup inactive sessions periodically
  startCleanupInterval(): void {
    setInterval(() => {
      const now = Date.now();
      for (const [id, session] of this.sessions) {
        if (now - session.lastActivity.getTime() > this.sessionTimeout) {
          this.closeSession(id);
        }
      }
    }, 60000); // Check every minute
  }
}
```

### 4.2 Command Executor

```typescript
// src/services/command-executor.ts

import { Page } from 'playwright';
import { humanizeTyping, humanizeClick } from '../stealth/behavior';

export class CommandExecutor {
  constructor(private page: Page) {}

  async navigate(url: string, waitUntil: 'load' | 'domcontentloaded' | 'networkidle' = 'networkidle', timeout = 30000) {
    const startTime = Date.now();

    await this.page.goto(url, {
      waitUntil,
      timeout,
    });

    return {
      url: this.page.url(),
      title: await this.page.title(),
      loadTimeMs: Date.now() - startTime,
    };
  }

  async fill(fields: FillField[], options: FillOptions = {}) {
    const startTime = Date.now();

    for (const field of fields) {
      const element = await this.page.waitForSelector(field.selector, {
        timeout: options.timeout || 10000,
      });

      if (!element) {
        throw new Error(`Element not found: ${field.selector}`);
      }

      // Clear existing value if requested
      if (field.clearFirst !== false) {
        await element.fill('');
      }

      // Use humanized typing if requested
      if (field.humanize !== false) {
        await humanizeTyping(element, field.value);
      } else {
        await element.fill(field.value);
      }
    }

    // Auto-submit if requested
    if (options.autoSubmit && options.submitSelector) {
      if (options.submitDelay) {
        await this.page.waitForTimeout(options.submitDelay);
      }
      await this.click(options.submitSelector, { waitFor: 'navigation' });
    }

    return {
      fieldsFilled: fields.length,
      durationMs: Date.now() - startTime,
    };
  }

  async click(selector: string, options: ClickOptions = {}) {
    const element = await this.page.waitForSelector(selector, {
      timeout: options.timeout || 10000,
    });

    if (!element) {
      throw new Error(`Element not found: ${selector}`);
    }

    // Humanized mouse movement
    if (options.humanize !== false) {
      await humanizeClick(this.page, element);
    } else {
      await element.click();
    }

    // Wait for navigation or selector
    if (options.waitFor === 'navigation') {
      await this.page.waitForLoadState('networkidle');
    } else if (options.waitFor === 'selector' && options.waitSelector) {
      await this.page.waitForSelector(options.waitSelector);
    }

    return {
      clicked: true,
      newUrl: this.page.url(),
    };
  }

  async wait(type: string, value: string, timeout = 10000) {
    const startTime = Date.now();

    switch (type) {
      case 'selector':
        await this.page.waitForSelector(value, { timeout });
        break;
      case 'timeout':
        await this.page.waitForTimeout(parseInt(value));
        break;
      case 'url':
        await this.page.waitForURL(value, { timeout });
        break;
      case 'function':
        await this.page.waitForFunction(value, { timeout });
        break;
      default:
        throw new Error(`Unknown wait type: ${type}`);
    }

    return {
      conditionMet: true,
      waitTimeMs: Date.now() - startTime,
    };
  }

  async extract(fields: ExtractField[]) {
    const data: Record<string, string> = {};

    for (const field of fields) {
      const element = await this.page.$(field.selector);
      if (element) {
        let value: string;

        if (field.attribute === 'href' || field.attribute === 'src') {
          value = await element.getAttribute(field.attribute) || '';
        } else {
          value = await element.textContent() || '';
        }

        data[field.name] = value.trim();
      }
    }

    return { data };
  }

  async screenshot(options: ScreenshotOptions = {}) {
    let screenshot: Buffer;

    if (options.selector) {
      const element = await this.page.$(options.selector);
      if (element) {
        screenshot = await element.screenshot({
          type: options.format || 'png',
        });
      } else {
        throw new Error(`Element not found: ${options.selector}`);
      }
    } else {
      screenshot = await this.page.screenshot({
        fullPage: options.fullPage || false,
        type: options.format || 'png',
      });
    }

    return {
      screenshot: `data:image/${options.format || 'png'};base64,${screenshot.toString('base64')}`,
      width: options.fullPage ? undefined : (await this.page.viewportSize())?.width,
      height: options.fullPage ? undefined : (await this.page.viewportSize())?.height,
    };
  }
}
```

### 4.3 Stealth Configuration

```typescript
// src/stealth/index.ts

import { BrowserContext } from 'playwright';

export async function applyStealth(context: BrowserContext): Promise<void> {
  await context.addInitScript(() => {
    // Hide webdriver
    Object.defineProperty(navigator, 'webdriver', {
      get: () => undefined,
    });

    // Mock plugins
    Object.defineProperty(navigator, 'plugins', {
      get: () => [
        { name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer' },
        { name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai' },
        { name: 'Native Client', filename: 'internal-nacl-plugin' },
      ],
    });

    // Mock languages
    Object.defineProperty(navigator, 'languages', {
      get: () => ['en-US', 'en'],
    });

    // Mock platform
    Object.defineProperty(navigator, 'platform', {
      get: () => 'Win32',
    });

    // Mock hardware concurrency
    Object.defineProperty(navigator, 'hardwareConcurrency', {
      get: () => 8,
    });

    // Mock device memory
    Object.defineProperty(navigator, 'deviceMemory', {
      get: () => 8,
    });

    // Hide automation indicators
    // @ts-ignore
    window.chrome = {
      app: {
        isInstalled: false,
        InstallState: { DISABLED: 'disabled', INSTALLED: 'installed', NOT_INSTALLED: 'not_installed' },
        RunningState: { CANNOT_RUN: 'cannot_run', READY_TO_RUN: 'ready_to_run', RUNNING: 'running' },
      },
      csi: () => ({ onloadT: Date.now() }),
      loadTimes: () => ({
        commitLoadTime: Date.now() / 1000,
        connectionInfo: 'http/1.1',
        finishDocumentLoadTime: Date.now() / 1000,
        finishLoadTime: Date.now() / 1000,
        firstPaintAfterLoadTime: 0,
        firstPaintTime: Date.now() / 1000,
        navigationType: 'Other',
        npnNegotiatedProtocol: 'unknown',
        requestTime: Date.now() / 1000,
        startLoadTime: Date.now() / 1000,
        wasAlternateProtocolAvailable: false,
        wasFetchedViaSpdy: false,
        wasNpnNegotiated: false,
      }),
    };

    // Permission query spoofing
    const originalQuery = window.navigator.permissions.query;
    // @ts-ignore
    window.navigator.permissions.query = (parameters: any) => (
      parameters.name === 'notifications' ?
        Promise.resolve({ state: Notification.permission } as PermissionStatus) :
        originalQuery(parameters)
    );
  });
}
```

### 4.4 Human-like Behavior

```typescript
// src/stealth/behavior.ts

import { ElementHandle, Page } from 'playwright';

/**
 * Humanized typing with natural delays and occasional mistakes
 */
export async function humanizeTyping(
  element: ElementHandle,
  text: string,
  options: TypingOptions = {}
): Promise<void> {
  const {
    minDelay = 50,
    maxDelay = 150,
    mistakeRate = 0.02,
    correctMistakes = true,
  } = options;

  for (let i = 0; i < text.length; i++) {
    let char = text[i];

    // Simulate occasional typos
    if (Math.random() < mistakeRate) {
      // Type wrong character
      const wrongChar = getNearbyKey(char);
      await element.type(wrongChar, { delay: 0 });

      // Pause (realization)
      await sleep(100 + Math.random() * 200);

      // Correct
      if (correctMistakes) {
        await element.press('Backspace', { delay: 50 + Math.random() * 100 });
        await sleep(50 + Math.random() * 100);
      } else {
        i--; // Re-type correct char
        continue;
      }
    }

    // Type correct character with natural delay
    await element.type(char, { delay: 0 });
    await sleep(minDelay + Math.random() * (maxDelay - minDelay));

    // Occasional longer pauses (thinking)
    if (Math.random() < 0.05) {
      await sleep(300 + Math.random() * 500);
    }
  }
}

/**
 * Humanized mouse movement and click
 */
export async function humanizeClick(
  page: Page,
  element: ElementHandle
): Promise<void> {
  // Get element bounding box
  const box = await element.boundingBox();
  if (!box) throw new Error('Element not visible');

  // Calculate target point (with slight randomness)
  const targetX = box.x + box.width * (0.3 + Math.random() * 0.4);
  const targetY = box.y + box.height * (0.3 + Math.random() * 0.4);

  // Get current mouse position (or start from random position)
  const startX = Math.random() * 1920;
  const startY = Math.random() * 1080;

  // Generate bezier curve points
  const steps = 20 + Math.floor(Math.random() * 20);
  const points = generateBezierCurve(
    { x: startX, y: startY },
    { x: targetX, y: targetY },
    steps
  );

  // Move mouse along curve
  for (const point of points) {
    await page.mouse.move(point.x, point.y);
    await sleep(10 + Math.random() * 20);
  }

  // Small pause before click
  await sleep(50 + Math.random() * 100);

  // Click
  await page.mouse.click(targetX, targetY);
}

/**
 * Generate bezier curve for natural mouse movement
 */
function generateBezierCurve(
  start: Point,
  end: Point,
  steps: number
): Point[] {
  const points: Point[] = [];

  // Random control points for curve
  const cp1 = {
    x: start.x + (end.x - start.x) * (0.2 + Math.random() * 0.3) + (Math.random() - 0.5) * 200,
    y: start.y + (end.y - start.y) * (0.2 + Math.random() * 0.3) + (Math.random() - 0.5) * 200,
  };

  const cp2 = {
    x: start.x + (end.x - start.x) * (0.6 + Math.random() * 0.2) + (Math.random() - 0.5) * 200,
    y: start.y + (end.y - start.y) * (0.6 + Math.random() * 0.2) + (Math.random() - 0.5) * 200,
  };

  for (let i = 0; i <= steps; i++) {
    const t = i / steps;
    const x = cubicBezier(t, start.x, cp1.x, cp2.x, end.x);
    const y = cubicBezier(t, start.y, cp1.y, cp2.y, end.y);
    points.push({ x, y });
  }

  return points;
}

function cubicBezier(t: number, p0: number, p1: number, p2: number, p3: number): number {
  const mt = 1 - t;
  return mt * mt * mt * p0 + 3 * mt * mt * t * p1 + 3 * mt * t * t * p2 + t * t * t * p3;
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

function getNearbyKey(char: string): string {
  const keyboard = [
    ['q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p'],
    ['a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l'],
    ['z', 'x', 'c', 'v', 'b', 'n', 'm'],
  ];

  const lower = char.toLowerCase();

  for (let row = 0; row < keyboard.length; row++) {
    const col = keyboard[row].indexOf(lower);
    if (col !== -1) {
      // Pick a nearby key
      const offsets = [-1, 1, -1, 1]; // left, right, up, down
      const offset = offsets[Math.floor(Math.random() * offsets.length)];

      if (Math.random() < 0.5 && col + offset >= 0 && col + offset < keyboard[row].length) {
        return keyboard[row][col + offset];
      }
    }
  }

  return char; // Fallback to original
}

interface Point {
  x: number;
  y: number;
}

interface TypingOptions {
  minDelay?: number;
  maxDelay?: number;
  mistakeRate?: number;
  correctMistakes?: boolean;
}
```

### 4.5 Intervention Detection

```typescript
// src/services/intervention.ts

import { Page } from 'playwright';

interface InterventionResult {
  interventionRequired: boolean;
  type?: 'captcha' | 'twofa' | 'error';
  subtype?: string;
  selectors?: string[];
  screenshot?: string;
  hint?: string;
  message?: string;
  inputSelector?: string;
}

const CAPTCHA_SELECTORS = [
  'iframe[src*="recaptcha"]',
  'iframe[src*="hcaptcha"]',
  'iframe[src*="captcha"]',
  '.g-recaptcha',
  '.h-captcha',
  '#recaptcha',
  '#captcha',
  '[class*="captcha"]',
  '[id*="captcha"]',
  'div[data-sitekey]',
];

const TWOFA_SELECTORS = [
  'input[placeholder*="code"]',
  'input[placeholder*="Code"]',
  'input[placeholder*="OTP"]',
  'input[placeholder*="verification"]',
  'input[placeholder*=" Verification"]',
  'input[maxlength="6"][type="text"]',
  'input[maxlength="6"][type="number"]',
  'input[maxlength="4"][type="text"]',
  'input[name*="otp"]',
  'input[name*="code"]',
  'input[id*="otp"]',
  'input[id*="verification"]',
];

const ERROR_SELECTORS = [
  '.error-message',
  '.error',
  '.alert-danger',
  '.alert-error',
  '[role="alert"]',
  '.form-error',
  '.validation-error',
  '[class*="error"]',
];

export async function detectIntervention(page: Page): Promise<InterventionResult> {
  // Check for CAPTCHA
  for (const selector of CAPTCHA_SELECTORS) {
    const element = await page.$(selector);
    if (element) {
      const screenshot = await page.screenshot({ encoding: 'base64', fullPage: false });

      let subtype = 'unknown';
      if (selector.includes('recaptcha')) subtype = 'recaptcha';
      else if (selector.includes('hcaptcha')) subtype = 'hcaptcha';

      return {
        interventionRequired: true,
        type: 'captcha',
        subtype,
        selectors: [selector],
        screenshot: `data:image/png;base64,${screenshot}`,
        hint: "Click 'I'm not a robot' and complete the challenge",
      };
    }
  }

  // Check for 2FA
  for (const selector of TWOFA_SELECTORS) {
    const element = await page.$(selector);
    if (element && await element.isVisible()) {
      const screenshot = await page.screenshot({ encoding: 'base64', fullPage: false });

      // Try to determine 2FA type from page content
      const pageContent = await page.content();
      let subtype = 'unknown';
      if (pageContent.includes('SMS') || pageContent.includes('text message')) subtype = 'sms';
      else if (pageContent.includes('email')) subtype = 'email';
      else if (pageContent.includes('authenticator') || pageContent.includes('app')) subtype = 'totp';

      return {
        interventionRequired: true,
        type: 'twofa',
        subtype,
        selectors: [selector],
        screenshot: `data:image/png;base64,${screenshot}`,
        hint: 'Enter the verification code',
        inputSelector: selector,
      };
    }
  }

  // Check for errors
  for (const selector of ERROR_SELECTORS) {
    const element = await page.$(selector);
    if (element && await element.isVisible()) {
      const message = await element.textContent() || '';
      const screenshot = await page.screenshot({ encoding: 'base64', fullPage: false });

      return {
        interventionRequired: true,
        type: 'error',
        subtype: 'form_error',
        selectors: [selector],
        screenshot: `data:image/png;base64,${screenshot}`,
        message: message.trim(),
      };
    }
  }

  return { interventionRequired: false };
}
```

---

## 5. Docker Configuration

### Dockerfile

```dockerfile
# Dockerfile

FROM mcr.microsoft.com/playwright:v1.42.0-jammy

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm ci --only=production

# Copy source
COPY dist/ ./dist/

# Environment
ENV NODE_ENV=production
ENV PORT=3000

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/health || exit 1

# Run
CMD ["node", "dist/index.js"]
```

### docker-compose.yml

```yaml
# docker-compose.yml

version: '3.8'

services:
  browser-service:
    build: .
    container_name: armorclaw-browser
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - LOG_LEVEL=info
      - MAX_SESSIONS=10
      - SESSION_TIMEOUT=1800000  # 30 minutes
    restart: unless-stopped
    # Required for Chromium
    cap_add:
      - SYS_ADMIN
    # Shared memory for browser
    shm_size: '2gb'
    # Resource limits
    deploy:
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 1G
    # Volume for persistent data (optional)
    volumes:
      - browser-data:/app/data

volumes:
  browser-data:
```

---

## 6. Environment Configuration

```bash
# .env.example

# Server
PORT=3000
HOST=0.0.0.0
NODE_ENV=development

# Logging
LOG_LEVEL=debug  # trace | debug | info | warn | error

# Sessions
MAX_SESSIONS=10
SESSION_TIMEOUT=1800000  # 30 minutes in ms

# Browser
HEADLESS=true
DEFAULT_TIMEOUT=30000

# Proxy (optional)
# PROXY_URL=http://proxy:8080
# PROXY_USERNAME=user
# PROXY_PASSWORD=pass
```

---

## 7. Testing

### Unit Tests

```typescript
// tests/session-manager.test.ts

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { SessionManager } from '../src/services/session-manager';

describe('SessionManager', () => {
  let manager: SessionManager;

  beforeEach(() => {
    manager = new SessionManager();
  });

  afterEach(async () => {
    // Cleanup all sessions
  });

  it('should create a new session', async () => {
    const session = await manager.createSession({});

    expect(session.id).toBeDefined();
    expect(session.browser).toBeDefined();
    expect(session.page).toBeDefined();
  });

  it('should close a session', async () => {
    const session = await manager.createSession({});
    await manager.closeSession(session.id);

    expect(manager.getSession(session.id)).toBeUndefined();
  });

  it('should enforce max sessions limit', async () => {
    // Create max sessions
    for (let i = 0; i < 10; i++) {
      await manager.createSession({});
    }

    // One more should trigger cleanup
    const newSession = await manager.createSession({});
    expect(newSession).toBeDefined();
  });
});
```

### Integration Tests

```typescript
// tests/commands.test.ts

import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { SessionManager } from '../src/services/session-manager';
import { CommandExecutor } from '../src/services/command-executor';

describe('CommandExecutor', () => {
  let manager: SessionManager;
  let executor: CommandExecutor;

  beforeAll(async () => {
    manager = new SessionManager();
    const session = await manager.createSession({});
    executor = new CommandExecutor(session.page);
  });

  afterAll(async () => {
    await manager.closeAllSessions();
  });

  it('should navigate to a URL', async () => {
    const result = await executor.navigate('https://example.com');

    expect(result.url).toBe('https://example.com/');
    expect(result.title).toBeDefined();
  });

  it('should fill form fields', async () => {
    await executor.navigate('https://httpbin.org/forms/post');

    const result = await executor.fill([
      { selector: 'input[name="custname"]', value: 'Test User' },
      { selector: 'input[name="custtel"]', value: '555-1234' },
    ]);

    expect(result.fieldsFilled).toBe(2);
  });

  it('should detect CAPTCHA', async () => {
    await executor.navigate('https://www.google.com/recaptcha/api2/demo');

    const result = await detectIntervention(executor.page);

    expect(result.interventionRequired).toBe(true);
    expect(result.type).toBe('captcha');
  });
});
```

---

## 8. Deployment Checklist

- [ ] Build TypeScript: `npm run build`
- [ ] Build Docker image: `docker build -t armorclaw-browser .`
- [ ] Configure environment variables
- [ ] Set up resource limits (memory, CPU)
- [ ] Configure health checks
- [ ] Set up logging aggregation
- [ ] Configure network isolation (only Bridge can access)
- [ ] Set up monitoring and alerts
- [ ] Test anti-detection on target sites
- [ ] Load test with concurrent sessions

---

## 9. Security Considerations

1. **Network Isolation**: Browser service should only be accessible from Bridge
2. **No Credential Storage**: Values passed through, never persisted
3. **Session Cleanup**: Automatic cleanup of inactive sessions
4. **Resource Limits**: Prevent runaway memory usage
5. **Input Validation**: Sanitize all incoming requests
6. **Rate Limiting**: Prevent abuse of API endpoints

---

*Implementation guide generated on 2026-02-28*
