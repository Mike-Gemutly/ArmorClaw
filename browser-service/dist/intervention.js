"use strict";
/**
 * @fileoverview Intervention detection for CAPTCHA, 2FA, and unexpected states
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.INTERVENTION_SELECTORS = void 0;
exports.detectIntervention = detectIntervention;
exports.createInterventionInfo = createInterventionInfo;
exports.detectCaptchaType = detectCaptchaType;
exports.detect2FADetails = detect2FADetails;
exports.extractErrorMessage = extractErrorMessage;
//=============================================================================
// Intervention Selectors
//=============================================================================
exports.INTERVENTION_SELECTORS = {
    captcha: [
        // reCAPTCHA
        'iframe[src*="recaptcha"]',
        '.g-recaptcha',
        '.g-recaptcha-response',
        '[data-sitekey]',
        // hCaptcha
        'iframe[src*="hcaptcha"]',
        '.h-captcha',
        '#h-captcha',
        // Cloudflare
        '#cf-wrapper',
        '.challenge-platform',
        // Generic
        '#captcha',
        '[class*="captcha"]',
        '[id*="captcha"]',
        '[name*="captcha"]',
        '.captcha-container',
        '.recaptcha',
    ],
    twofa: [
        // OTP/Code inputs
        'input[placeholder*="code"]',
        'input[placeholder*="Code"]',
        'input[placeholder*="OTP"]',
        'input[placeholder*="otp"]',
        'input[placeholder*="verification"]',
        'input[placeholder*="Verification"]',
        'input[placeholder*="2FA"]',
        'input[placeholder*="two-factor"]',
        // Numeric code patterns
        'input[maxlength="6"][type="text"]',
        'input[maxlength="6"][type="number"]',
        'input[maxlength="8"][type="text"]',
        'input[pattern*="[0-9]"]',
        // SMS code patterns
        'input[placeholder*="SMS"]',
        'input[placeholder*="sms"]',
        'input[placeholder*="text"]',
        // Authenticator patterns
        '.otp-input',
        '.code-input',
        '.verification-code',
        '#verification-code',
        '[data-testid*="otp"]',
        '[data-testid*="verification"]',
    ],
    unexpected: [
        // Error messages
        '.error-message',
        '.error-message-container',
        '.alert-danger',
        '.alert-error',
        '.form-error',
        '.validation-error',
        '[class*="error"]',
        '[class*="Error"]',
        // Alerts
        '[role="alert"]',
        '.alert',
        '.notification',
        // Blocking overlays
        '.modal-backdrop',
        '.overlay',
        '.blocker',
        // Access denied
        '.access-denied',
        '.forbidden',
        '.not-allowed',
        '.blocked',
    ],
    success: [
        // Confirmation messages
        '.success-message',
        '.alert-success',
        '.confirmation',
        '.thank-you',
        // Order confirmation
        '.order-confirmation',
        '.order-complete',
        '.receipt',
        '.confirmation-number',
    ],
};
/**
 * Detect intervention on the page
 */
async function detectIntervention(page) {
    // Check each intervention type
    for (const [type, selectors] of Object.entries(exports.INTERVENTION_SELECTORS)) {
        for (const selector of selectors) {
            try {
                const locator = page.locator(selector);
                const count = await locator.count();
                if (count > 0) {
                    return {
                        detected: true,
                        type: type,
                        selectors: [selector],
                        confidence: calculateConfidence(type, selector),
                    };
                }
            }
            catch {
                // Invalid selector, continue
                continue;
            }
        }
    }
    return null;
}
/**
 * Calculate confidence score for detection
 */
function calculateConfidence(_type, selector) {
    // High confidence selectors
    const highConfidence = [
        'iframe[src*="recaptcha"]',
        'iframe[src*="hcaptcha"]',
        '.g-recaptcha',
        '.h-captcha',
        'input[maxlength="6"][type="text"]',
    ];
    // Medium confidence selectors
    const mediumConfidence = [
        '#captcha',
        '.captcha-container',
        'input[placeholder*="code"]',
        'input[placeholder*="OTP"]',
    ];
    if (highConfidence.includes(selector)) {
        return 0.9;
    }
    if (mediumConfidence.includes(selector)) {
        return 0.7;
    }
    return 0.5;
}
/**
 * Create intervention info with screenshot
 */
async function createInterventionInfo(page, type, selectors) {
    const buffer = await page.screenshot({ fullPage: false });
    const screenshot = buffer.toString('base64');
    return {
        type,
        selectors,
        screenshot,
        timestamp: Date.now(),
        url: page.url(),
    };
}
/**
 * Detect specific captcha type and details
 */
async function detectCaptchaType(page) {
    // Check reCAPTCHA v2
    const recaptchaV2Locator = page.locator('.g-recaptcha');
    if ((await recaptchaV2Locator.count()) > 0) {
        const siteKey = await recaptchaV2Locator.getAttribute('data-sitekey');
        return {
            type: 'recaptcha_v2',
            siteKey: siteKey || undefined,
            isInvisible: false,
            selector: '.g-recaptcha',
        };
    }
    // Check reCAPTCHA v3 (invisible)
    const recaptchaV3Locator = page.locator('.g-recaptcha-response');
    if ((await recaptchaV3Locator.count()) > 0) {
        return {
            type: 'recaptcha_v3',
            isInvisible: true,
            selector: '.g-recaptcha-response',
        };
    }
    // Check hCaptcha
    const hcaptchaLocator = page.locator('.h-captcha');
    if ((await hcaptchaLocator.count()) > 0) {
        const siteKey = await hcaptchaLocator.getAttribute('data-sitekey');
        return {
            type: 'hcaptcha',
            siteKey: siteKey || undefined,
            isInvisible: false,
            selector: '.h-captcha',
        };
    }
    // Check Cloudflare
    const cloudflareLocator = page.locator('#cf-wrapper');
    if ((await cloudflareLocator.count()) > 0) {
        return {
            type: 'cloudflare',
            isInvisible: true,
            selector: '#cf-wrapper',
        };
    }
    return null;
}
/**
 * Detect 2FA input details
 */
async function detect2FADetails(page) {
    // Try common 2FA input selectors
    const selectors = [
        'input[maxlength="6"][type="text"]',
        'input[maxlength="6"][type="number"]',
        'input[maxlength="8"][type="text"]',
        'input[placeholder*="code"]',
        'input[placeholder*="OTP"]',
        '.otp-input input',
    ];
    for (const selector of selectors) {
        const locator = page.locator(selector);
        if ((await locator.count()) > 0) {
            const maxLength = await locator.getAttribute('maxlength');
            const type = await locator.getAttribute('type');
            const placeholder = await locator.getAttribute('placeholder');
            return {
                inputSelector: selector,
                expectedLength: maxLength ? parseInt(maxLength, 10) : 6,
                isNumeric: type === 'number' || (placeholder?.toLowerCase().includes('code') ?? false),
                placeholder: placeholder || undefined,
            };
        }
    }
    return null;
}
/**
 * Extract error message from page
 */
async function extractErrorMessage(page) {
    const errorSelectors = [
        '.error-message',
        '.alert-danger',
        '.form-error',
        '[role="alert"]',
        '.validation-error',
    ];
    for (const selector of errorSelectors) {
        const locator = page.locator(selector);
        if ((await locator.count()) > 0) {
            const text = await locator.textContent();
            if (text && text.trim()) {
                return {
                    message: text.trim(),
                    selector,
                    isError: true,
                };
            }
        }
    }
    return null;
}
//# sourceMappingURL=intervention.js.map