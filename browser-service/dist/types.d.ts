/**
 * @fileoverview Shared types for browser service
 * Matches bridge/pkg/studio/browser_skill.go types
 */
export type WaitUntil = 'load' | 'domcontentloaded' | 'networkidle';
export type BrowserState = 'IDLE' | 'LOADING' | 'FILLING' | 'WAITING' | 'PROCESSING' | 'ERROR';
export type BrowserErrorCode = 'ELEMENT_NOT_FOUND' | 'NAVIGATION_FAILED' | 'TIMEOUT' | 'PII_REQUEST_DENIED' | 'INVALID_SELECTOR' | 'BROWSER_NOT_READY' | 'EXTRACTION_FAILED' | 'SCREENSHOT_FAILED' | 'SESSION_EXPIRED' | 'CAPTCHA_REQUIRED' | 'TWO_FA_REQUIRED' | 'UNEXPECTED_STATE';
export interface NavigateCommand {
    url: string;
    waitUntil?: WaitUntil;
    timeout?: number;
}
export interface FillField {
    selector: string;
    value?: string;
    value_ref?: string;
}
export interface FillCommand {
    fields: FillField[];
    auto_submit?: boolean;
    submit_delay?: number;
}
export interface ClickCommand {
    selector: string;
    waitFor?: 'none' | 'navigation' | 'selector';
    timeout?: number;
}
export interface WaitCommand {
    condition: 'selector' | 'timeout' | 'url';
    value: string;
    timeout?: number;
}
export interface ExtractField {
    name: string;
    selector: string;
    attribute?: string;
}
export interface ExtractCommand {
    fields: ExtractField[];
}
export interface ScreenshotCommand {
    fullPage?: boolean;
    selector?: string;
    format?: 'png' | 'jpeg';
}
export interface BrowserResponse<T = unknown> {
    success: boolean;
    data?: T;
    error?: BrowserError;
    duration: number;
    screenshot?: string;
}
export interface BrowserError {
    code: BrowserErrorCode;
    message: string;
    selector?: string;
    screenshot?: string;
}
export interface BrowserSession {
    id: string;
    createdAt: Date;
    lastActivity: Date;
    state: BrowserState;
    currentUrl?: string;
    cookies: Record<string, string>;
}
export type InterventionType = 'captcha' | 'twofa' | 'unexpected' | 'blocked';
export interface InterventionInfo {
    type: InterventionType;
    selectors: string[];
    screenshot: string;
    timestamp: number;
    url?: string;
}
export interface StealthConfig {
    fingerprint: FingerprintConfig;
    behavior: BehaviorConfig;
    evasion: EvasionConfig;
    network: NetworkConfig;
}
export interface FingerprintConfig {
    seed: string;
    webgl: WebGLConfig;
    audioContext: AudioConfig;
    canvas: CanvasConfig;
}
export interface WebGLConfig {
    vendor: string;
    renderer: string;
}
export interface AudioConfig {
    noise: boolean;
    noiseValue: number;
}
export interface CanvasConfig {
    noise: boolean;
}
export interface BehaviorConfig {
    typing: TypingConfig;
    mouse: MouseConfig;
    scrolling: ScrollingConfig;
    pageLoad: PageLoadConfig;
}
export interface TypingConfig {
    minDelay: number;
    maxDelay: number;
    variance: number;
    mistakeRate: number;
    burstTyping: boolean;
}
export interface MouseConfig {
    movement: 'linear' | 'bezier-curve';
    speedVariation: [number, number];
    overshoot: number;
}
export interface ScrollingConfig {
    smooth: boolean;
    speed: 'constant' | 'variable';
    pauses: boolean;
}
export interface PageLoadConfig {
    waitBeforeAction: [number, number];
    scrollBeforeFill: boolean;
}
export interface EvasionConfig {
    webdriver: {
        hidden: boolean;
    };
    chrome: {
        app: boolean;
        csi: boolean;
        loadTimes: boolean;
    };
    permissions: {
        query: boolean;
    };
    plugins: {
        array: boolean;
    };
    languages: string[];
    hardwareConcurrency: {
        value: number;
    };
    deviceMemory: {
        value: number;
    };
    platform: string;
}
export interface NetworkConfig {
    tls: {
        minVersion: string;
        cipherSuites: string;
    };
    http2: boolean;
    headers: Record<string, string>;
}
//# sourceMappingURL=types.d.ts.map