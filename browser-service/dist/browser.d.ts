/**
 * @fileoverview Playwright browser wrapper with stealth and human-like behavior
 */
import type { StealthConfig, BrowserSession, BrowserState, NavigateCommand, FillCommand, ClickCommand, WaitCommand, ExtractCommand, ScreenshotCommand, BrowserResponse } from './types';
export declare class BrowserClient {
    private browser;
    private context;
    private page;
    private config;
    private session;
    constructor(config?: Partial<StealthConfig>);
    initialize(): Promise<void>;
    close(): Promise<void>;
    navigate(command: NavigateCommand): Promise<BrowserResponse>;
    fill(command: FillCommand): Promise<BrowserResponse>;
    click(command: ClickCommand): Promise<BrowserResponse>;
    wait(command: WaitCommand): Promise<BrowserResponse>;
    extract(command: ExtractCommand): Promise<BrowserResponse>;
    screenshot(command: ScreenshotCommand): Promise<BrowserResponse>;
    getSession(): BrowserSession | null;
    getState(): BrowserState;
    private ensureReady;
    private updateState;
    private checkInterventions;
    private createInterventionResponse;
    private createErrorResponse;
    private humanDelay;
    private humanScroll;
    private humanClick;
    private getBrowserArgs;
    private getUserAgent;
    private applyStealthScripts;
}
export declare function getBrowser(): BrowserClient;
export declare function initializeBrowser(): Promise<void>;
export declare function closeBrowser(): Promise<void>;
//# sourceMappingURL=browser.d.ts.map