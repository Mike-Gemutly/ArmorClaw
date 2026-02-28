/**
 * @fileoverview Intervention detection for CAPTCHA, 2FA, and unexpected states
 */
import type { Page } from 'playwright';
import type { InterventionInfo, InterventionType } from './types';
export declare const INTERVENTION_SELECTORS: {
    captcha: string[];
    twofa: string[];
    unexpected: string[];
    success: string[];
};
export interface DetectionResult {
    detected: boolean;
    type: InterventionType;
    selectors: string[];
    confidence: number;
}
/**
 * Detect intervention on the page
 */
export declare function detectIntervention(page: Page): Promise<DetectionResult | null>;
/**
 * Create intervention info with screenshot
 */
export declare function createInterventionInfo(page: Page, type: InterventionType, selectors: string[]): Promise<InterventionInfo>;
export type CaptchaType = 'recaptcha_v2' | 'recaptcha_v3' | 'hcaptcha' | 'cloudflare' | 'unknown';
export interface CaptchaDetails {
    type: CaptchaType;
    siteKey?: string;
    isInvisible: boolean;
    selector: string;
}
/**
 * Detect specific captcha type and details
 */
export declare function detectCaptchaType(page: Page): Promise<CaptchaDetails | null>;
export interface TwoFADetails {
    inputSelector: string;
    expectedLength: number;
    isNumeric: boolean;
    placeholder?: string;
}
/**
 * Detect 2FA input details
 */
export declare function detect2FADetails(page: Page): Promise<TwoFADetails | null>;
export interface ErrorDetails {
    message: string;
    selector: string;
    isError: boolean;
}
/**
 * Extract error message from page
 */
export declare function extractErrorMessage(page: Page): Promise<ErrorDetails | null>;
//# sourceMappingURL=intervention.d.ts.map