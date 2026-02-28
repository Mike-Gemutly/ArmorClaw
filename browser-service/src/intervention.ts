/**
 * @fileoverview Intervention detection for CAPTCHA, 2FA, and unexpected states
 */

import type { Page } from 'playwright';
import type { InterventionInfo, InterventionType } from './types';

//=============================================================================
// Intervention Selectors
//=============================================================================

export const INTERVENTION_SELECTORS = {
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

//=============================================================================
// Intervention Detection
//=============================================================================

export interface DetectionResult {
  detected: boolean;
  type: InterventionType;
  selectors: string[];
  confidence: number; // 0-1
}

/**
 * Detect intervention on the page
 */
export async function detectIntervention(
  page: Page
): Promise<DetectionResult | null> {
  // Check each intervention type
  for (const [type, selectors] of Object.entries(INTERVENTION_SELECTORS)) {
    for (const selector of selectors) {
      try {
        const locator = page.locator(selector);
        const count = await locator.count();
        if (count > 0) {
          return {
            detected: true,
            type: type as InterventionType,
            selectors: [selector],
            confidence: calculateConfidence(type, selector),
          };
        }
      } catch {
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
function calculateConfidence(_type: string, selector: string): number {
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
export async function createInterventionInfo(
  page: Page,
  type: InterventionType,
  selectors: string[]
): Promise<InterventionInfo> {
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

//=============================================================================
// Captcha Type Detection
//=============================================================================

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
export async function detectCaptchaType(
  page: Page
): Promise<CaptchaDetails | null> {
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

//=============================================================================
// 2FA Detection Details
//=============================================================================

export interface TwoFADetails {
  inputSelector: string;
  expectedLength: number;
  isNumeric: boolean;
  placeholder?: string;
}

/**
 * Detect 2FA input details
 */
export async function detect2FADetails(
  page: Page
): Promise<TwoFADetails | null> {
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

//=============================================================================
// Error Message Extraction
//=============================================================================

export interface ErrorDetails {
  message: string;
  selector: string;
  isError: boolean;
}

/**
 * Extract error message from page
 */
export async function extractErrorMessage(
  page: Page
): Promise<ErrorDetails | null> {
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
