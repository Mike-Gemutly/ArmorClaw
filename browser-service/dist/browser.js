"use strict";
/**
 * @fileoverview Playwright browser wrapper with stealth and human-like behavior
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.BrowserClient = void 0;
exports.getBrowser = getBrowser;
exports.initializeBrowser = initializeBrowser;
exports.closeBrowser = closeBrowser;
const playwright_1 = require("playwright");
const stealth_1 = require("./stealth");
const intervention_1 = require("./intervention");
//=============================================================================
// Browser Client
//=============================================================================
class BrowserClient {
    browser = null;
    context = null;
    page = null;
    config;
    session = null;
    constructor(config = {}) {
        this.config = { ...stealth_1.defaultStealthConfig, ...config };
    }
    //=============================================================================
    // Lifecycle
    //=============================================================================
    async initialize() {
        if (this.browser)
            return;
        this.browser = await playwright_1.chromium.launch({
            headless: true,
            args: this.getBrowserArgs(),
        });
        this.context = await this.browser.newContext({
            userAgent: this.getUserAgent(),
            viewport: { width: 1920, height: 1080 },
            locale: 'en-US',
            timezoneId: 'America/New_York',
            // Stealth configurations
            bypassCSP: true,
            ignoreHTTPSErrors: true,
        });
        // Apply stealth scripts
        await this.applyStealthScripts();
        this.page = await this.context.newPage();
        // Initialize session
        this.session = {
            id: crypto.randomUUID(),
            createdAt: new Date(),
            lastActivity: new Date(),
            state: 'IDLE',
            cookies: {},
        };
    }
    async close() {
        if (this.page) {
            await this.page.close();
            this.page = null;
        }
        if (this.context) {
            await this.context.close();
            this.context = null;
        }
        if (this.browser) {
            await this.browser.close();
            this.browser = null;
        }
        this.session = null;
    }
    //=============================================================================
    // Navigation
    //=============================================================================
    async navigate(command) {
        const startTime = Date.now();
        try {
            this.ensureReady();
            this.updateState('LOADING');
            const waitUntil = (command.waitUntil || 'load');
            const timeout = command.timeout || 30000;
            // Random delay before navigation (human-like)
            await this.humanDelay(this.config.behavior.pageLoad.waitBeforeAction);
            await this.page.goto(command.url, {
                waitUntil,
                timeout,
            });
            // Scroll page like a human reading
            if (this.config.behavior.pageLoad.scrollBeforeFill) {
                await this.humanScroll();
            }
            // Check for interventions
            const intervention = await this.checkInterventions();
            if (intervention) {
                return this.createInterventionResponse(intervention, startTime);
            }
            this.updateState('IDLE');
            this.session.currentUrl = command.url;
            return {
                success: true,
                data: { url: command.url, title: await this.page.title() },
                duration: Date.now() - startTime,
            };
        }
        catch (error) {
            return this.createErrorResponse(error, startTime);
        }
    }
    //=============================================================================
    // Form Filling
    //=============================================================================
    async fill(command) {
        const startTime = Date.now();
        try {
            this.ensureReady();
            this.updateState('FILLING');
            const results = {};
            for (const field of command.fields) {
                const value = field.value; // In real impl, resolve value_ref via PII system
                if (!value) {
                    results[field.selector] = false;
                    continue;
                }
                // Human-like typing
                const typing = (0, stealth_1.generateTypingDelays)({
                    text: value,
                    ...this.config.behavior.typing,
                });
                const locator = this.page.locator(field.selector);
                await locator.waitFor({ timeout: 5000 });
                await locator.focus();
                await locator.clear();
                // Type with human-like delays
                await locator.pressSequentially(value, { delay: this.config.behavior.typing.minDelay });
                results[field.selector] = true;
            }
            // Check for interventions after fill
            const intervention = await this.checkInterventions();
            if (intervention) {
                return this.createInterventionResponse(intervention, startTime);
            }
            // Auto-submit if requested
            if (command.auto_submit) {
                await this.humanDelay([200, 500]);
                await this.page.keyboard.press('Enter');
            }
            this.updateState('IDLE');
            return {
                success: true,
                data: { filled: results },
                duration: Date.now() - startTime,
            };
        }
        catch (error) {
            return this.createErrorResponse(error, startTime);
        }
    }
    //=============================================================================
    // Click
    //=============================================================================
    async click(command) {
        const startTime = Date.now();
        try {
            this.ensureReady();
            this.updateState('PROCESSING');
            const element = await this.page.waitForSelector(command.selector, { timeout: 5000 });
            // Human-like mouse movement and click
            await this.humanClick(element);
            // Wait after click
            if (command.waitFor === 'navigation') {
                await this.page.waitForLoadState('load', { timeout: command.timeout || 10000 });
            }
            else if (command.waitFor?.startsWith('#') || command.waitFor?.startsWith('.')) {
                await this.page.waitForSelector(command.waitFor, { timeout: command.timeout || 5000 });
            }
            // Check for interventions
            const intervention = await this.checkInterventions();
            if (intervention) {
                return this.createInterventionResponse(intervention, startTime);
            }
            this.updateState('IDLE');
            return {
                success: true,
                data: { clicked: command.selector },
                duration: Date.now() - startTime,
            };
        }
        catch (error) {
            return this.createErrorResponse(error, startTime);
        }
    }
    //=============================================================================
    // Wait
    //=============================================================================
    async wait(command) {
        const startTime = Date.now();
        try {
            this.ensureReady();
            this.updateState('WAITING');
            const timeout = command.timeout || 5000;
            switch (command.condition) {
                case 'selector':
                    await this.page.waitForSelector(command.value, { timeout });
                    break;
                case 'timeout':
                    await this.page.waitForTimeout(parseInt(command.value, 10));
                    break;
                case 'url':
                    await this.page.waitForURL(`**/*${command.value}*`, { timeout });
                    break;
                default:
                    throw new Error(`Unknown wait condition: ${command.condition}`);
            }
            this.updateState('IDLE');
            return {
                success: true,
                duration: Date.now() - startTime,
            };
        }
        catch (error) {
            return this.createErrorResponse(error, startTime);
        }
    }
    //=============================================================================
    // Extract
    //=============================================================================
    async extract(command) {
        const startTime = Date.now();
        try {
            this.ensureReady();
            const data = {};
            for (const field of command.fields) {
                const element = await this.page.$(field.selector);
                if (element) {
                    const value = field.attribute
                        ? await element.getAttribute(field.attribute)
                        : await element.textContent();
                    data[field.name] = value;
                }
                else {
                    data[field.name] = null;
                }
            }
            return {
                success: true,
                data,
                duration: Date.now() - startTime,
            };
        }
        catch (error) {
            return this.createErrorResponse(error, startTime);
        }
    }
    //=============================================================================
    // Screenshot
    //=============================================================================
    async screenshot(command) {
        const startTime = Date.now();
        try {
            this.ensureReady();
            let buffer;
            if (command.selector) {
                const element = await this.page.$(command.selector);
                if (!element) {
                    throw new Error(`Element not found: ${command.selector}`);
                }
                buffer = await element.screenshot();
            }
            else {
                buffer = await this.page.screenshot({
                    fullPage: command.fullPage || false,
                    type: (command.format || 'png'),
                });
            }
            const base64 = buffer.toString('base64');
            return {
                success: true,
                data: { screenshot: base64 },
                duration: Date.now() - startTime,
            };
        }
        catch (error) {
            return this.createErrorResponse(error, startTime);
        }
    }
    //=============================================================================
    // Session Management
    //=============================================================================
    getSession() {
        return this.session;
    }
    getState() {
        return this.session?.state || 'IDLE';
    }
    //=============================================================================
    // Private Helpers
    //=============================================================================
    ensureReady() {
        if (!this.page || !this.browser) {
            throw new Error('Browser not initialized');
        }
    }
    updateState(state) {
        if (this.session) {
            this.session.state = state;
            this.session.lastActivity = new Date();
        }
    }
    async checkInterventions() {
        const detection = await (0, intervention_1.detectIntervention)(this.page);
        if (detection?.detected) {
            return (0, intervention_1.createInterventionInfo)(this.page, detection.type, detection.selectors);
        }
        return null;
    }
    createInterventionResponse(intervention, startTime) {
        this.updateState('WAITING'); // Wait for resolution
        return {
            success: false,
            error: {
                code: intervention.type === 'captcha' ? 'CAPTCHA_REQUIRED' :
                    intervention.type === 'twofa' ? 'TWO_FA_REQUIRED' : 'UNEXPECTED_STATE',
                message: `${intervention.type} detected`,
                screenshot: intervention.screenshot,
            },
            duration: Date.now() - startTime,
        };
    }
    createErrorResponse(error, startTime) {
        const err = error;
        this.updateState('ERROR');
        return {
            success: false,
            error: {
                code: 'BROWSER_NOT_READY',
                message: err.message,
            },
            duration: Date.now() - startTime,
        };
    }
    async humanDelay(range) {
        const delay = (0, stealth_1.getRandomDelay)(range[0], range[1]);
        await this.page.waitForTimeout(delay);
    }
    async humanScroll() {
        // Simulate reading - scroll down slowly
        const scrollSteps = 5;
        for (let i = 0; i < scrollSteps; i++) {
            // Use string-based evaluate to avoid TypeScript issues
            await this.page.evaluate(`
        window.scrollBy(0, window.innerHeight / 3);
      `);
            await this.humanDelay([300, 800]);
        }
    }
    async humanClick(element) {
        // Simulate human-like click with slight overshoot
        await element.click({ delay: (0, stealth_1.getRandomDelay)(50, 150) });
    }
    getBrowserArgs() {
        return [
            '--disable-blink-features=AutomationControlled',
            '--disable-features=IsolateOrigins,site-per-process',
            '--disable-site-isolation-trials',
            '--no-sandbox',
            '--disable-setuid-sandbox',
            '--disable-dev-shm-usage',
        ];
    }
    getUserAgent() {
        return 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36';
    }
    async applyStealthScripts() {
        if (!this.context)
            return;
        // Add stealth scripts to context
        await this.context.addInitScript({
            content: `
        // Override navigator.webdriver
        Object.defineProperty(navigator, 'webdriver', { get: () => undefined });

        // Override Chrome detection
        if (typeof window !== 'undefined') {
          (window as any).chrome = { runtime: {} };
        }

        // Override permissions
        const originalQuery = navigator.permissions.query;
        navigator.permissions.query = (parameters) => (
          parameters.name === 'notifications' ?
            Promise.resolve({ state: (typeof Notification !== 'undefined' ? Notification.permission : 'default') } as PermissionStatus) :
            originalQuery(parameters)
        );

        // Override plugins
        Object.defineProperty(navigator, 'plugins', {
          get: () => [1, 2, 3, 4, 5]
        });

        // Override languages
        Object.defineProperty(navigator, 'languages', {
          get: () => ['en-US', 'en']
        });
      `,
        });
    }
}
exports.BrowserClient = BrowserClient;
//=============================================================================
// Singleton Instance
//=============================================================================
let browserInstance = null;
function getBrowser() {
    if (!browserInstance) {
        browserInstance = new BrowserClient();
    }
    return browserInstance;
}
async function initializeBrowser() {
    const browser = getBrowser();
    await browser.initialize();
}
async function closeBrowser() {
    if (browserInstance) {
        await browserInstance.close();
        browserInstance = null;
    }
}
//# sourceMappingURL=browser.js.map