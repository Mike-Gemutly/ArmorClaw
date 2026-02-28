/**
 * @fileoverview Anti-detection and stealth configuration
 * Implements fingerprinting and human-like behavior
 */
import type { StealthConfig, FingerprintConfig } from './types';
export declare const defaultStealthConfig: StealthConfig;
/**
 * Generate deterministic fingerprint from seed
 */
export declare function generateFingerprint(seed: string): FingerprintConfig;
export interface TypingOptions {
    text: string;
    minDelay?: number;
    maxDelay?: number;
    variance?: number;
    mistakeRate?: number;
}
export interface TypingResult {
    delays: number[];
    mistakes: {
        position: number;
        char: string;
    }[];
    totalDuration: number;
}
/**
 * Generate human-like typing delays
 */
export declare function generateTypingDelays(options: TypingOptions): TypingResult;
export interface Point {
    x: number;
    y: number;
}
/**
 * Generate bezier curve points for natural mouse movement
 */
export declare function generateBezierPath(start: Point, end: Point, steps?: number): Point[];
/**
 * Get random delay within range
 */
export declare function getRandomDelay(min: number, max: number): number;
/**
 * Get random delay from range tuple
 */
export declare function getRandomDelayFromRange(range: [number, number]): number;
/**
 * Add jitter to a delay
 */
export declare function addJitter(delay: number, jitterPercent?: number): number;
//# sourceMappingURL=stealth.d.ts.map