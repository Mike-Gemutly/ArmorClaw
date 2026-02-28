/**
 * @fileoverview Anti-detection and stealth configuration
 * Implements fingerprinting and human-like behavior
 */

import type {
  StealthConfig,
  FingerprintConfig,
  BehaviorConfig,
  EvasionConfig,
  NetworkConfig,
} from './types';

//=============================================================================
// Default Stealth Configuration
//=============================================================================

export const defaultStealthConfig: StealthConfig = {
  fingerprint: {
    seed: 'armorclaw-default-seed',
    webgl: {
      vendor: 'Google Inc. (NVIDIA)',
      renderer: 'ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Direct3D11 vs_5_0 ps_5_0)',
    },
    audioContext: {
      noise: true,
      noiseValue: 0e-4,
    },
    canvas: {
      noise: true,
    },
  },

  behavior: {
    typing: {
    minDelay: 50,
    maxDelay: 180,
    variance: 0.3,
    mistakeRate: 0.015,
    burstTyping: true,
  },
  mouse: {
    movement: 'bezier-curve',
    speedVariation: [200, 800],
    overshoot: 0.05,
  },
  scrolling: {
    smooth: true,
    speed: 'variable',
    pauses: true,
  },
  pageLoad: {
    waitBeforeAction: [500, 2000],
    scrollBeforeFill: true,
  },
  },

  evasion: {
    webdriver: { hidden: true },
    chrome: { app: true, csi: true, loadTimes: true },
    permissions: { query: true },
    plugins: { array: true },
    languages: ['en-US', 'en'],
    hardwareConcurrency: { value: 8 },
    deviceMemory: { value: 8 },
    platform: 'Win32',
  },

  network: {
    tls: {
      minVersion: 'TLS1.2',
      cipherSuites: 'modern',
    },
    http2: true,
    headers: {
      'Accept-Language': 'en-US,en;q=0.9',
      'Sec-CH-UA': '"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"',
      'Sec-CH-UA-Mobile': '?0',
      'Sec-CH-UA-Platform': '"Windows"',
    },
  },
};

//=============================================================================
// Fingerprint Generation
//=============================================================================

/**
 * Generate deterministic fingerprint from seed
 */
export function generateFingerprint(seed: string): FingerprintConfig {
  // Simple hash-based deterministic generation
  const hash = hashString(seed);

  return {
    seed,
    webgl: {
      vendor: getWebGLVendor(hash),
      renderer: getWebGLRenderer(hash),
    },
    audioContext: {
      noise: true,
      noiseValue: getNoiseValue(hash),
    },
    canvas: {
      noise: true,
    },
  };
}

function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash);
}

const WEBGL_VENDORS = [
  'Google Inc. (NVIDIA)',
  'Google Inc. (AMD)',
  'Google Inc. (Intel)',
  'Google Inc. (Microsoft)',
];

const WEBGL_RENDERERS = [
  'ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Direct3D11 vs_5_0 ps_5_0)',
  'ANGLE (AMD, AMD Radeon RX 6800 Direct3D11 vs_5_0 ps_5_0)',
  'ANGLE (Intel, Intel(R) UHD Graphics 730 Direct3D11 vs_5_0 ps_5_0)',
  'ANGLE (Microsoft, Microsoft Basic Render Driver)',
];

function getWebGLVendor(hash: number): string {
  return WEBGL_VENDORS[hash % WEBGL_VENDORS.length];
}

function getWebGLRenderer(hash: number): string {
  return WEBGL_RENDERERS[hash % WEBGL_RENDERERS.length];
}

function getNoiseValue(hash: number): number {
  // Generate noise value between 1e-5 and 1e-3
  const normalized = (hash % 100) / 10000;
  return 1e-5 + normalized * (1e-3 - 1e-5);
}

//=============================================================================
// Human-like Typing Simulation
//=============================================================================

export interface TypingOptions {
  text: string;
  minDelay?: number;
  maxDelay?: number;
  variance?: number;
  mistakeRate?: number;
}

export interface TypingResult {
  delays: number[];
  mistakes: { position: number; char: string }[];
  totalDuration: number;
}

/**
 * Generate human-like typing delays
 */
export function generateTypingDelays(options: TypingOptions): TypingResult {
  const {
    text,
    minDelay = 50,
    maxDelay = 180,
    variance = 0.3,
    mistakeRate = 0.015,
  } = options;

  const delays: number[] = [];
  const mistakes: { position: number; char: string }[] = [];
  let totalDuration = 0;

  for (let i = 0; i < text.length; i++) {
    // Base delay with variance
    const baseDelay = minDelay + Math.random() * (maxDelay - minDelay);
    const varianceFactor = 1 + (Math.random() - 0.5) * variance * 2;
    let delay = Math.round(baseDelay * varianceFactor);

    // Burst typing - sometimes type faster
    if (Math.random() < 0.3) {
      delay = Math.round(delay * 0.5);
    }

    delays.push(delay);
    totalDuration += delay;

    // Potential mistake
    if (Math.random() < mistakeRate) {
      mistakes.push({
        position: i,
        char: getNearbyChar(text[i]),
      });
    }
  }

  return { delays, mistakes, totalDuration };
}

function getNearbyChar(char: string): string {
  const keyboard: Record<string, string[]> = {
    'a': ['q', 'z', 's'],
    's': ['a', 'd', 'w'],
    'd': ['s', 'f', 'e'],
    'f': ['d', 'g', 'r'],
    // Add more mappings as needed
  };

  const lower = char.toLowerCase();
  if (keyboard[lower]) {
    const nearby = keyboard[lower];
    return nearby[Math.floor(Math.random() * nearby.length)];
  }
  return char;
}

//=============================================================================
// Mouse Movement Simulation
//=============================================================================

export interface Point {
  x: number;
  y: number;
}

/**
 * Generate bezier curve points for natural mouse movement
 */
export function generateBezierPath(
  start: Point,
  end: Point,
  steps: number = 20
): Point[] {
  const points: Point[] = [];

  // Control points for bezier curve
  const cp1: Point = {
    x: start.x + (Math.random() - 0.5) * 100,
    y: start.y + (Math.random() - 0.5) * 100,
  };
  const cp2: Point = {
    x: end.x + (Math.random() - 0.5) * 100,
    y: end.y + (Math.random() - 0.5) * 100,
  };

  for (let i = 0; i <= steps; i++) {
    const t = i / steps;

    // Cubic bezier curve
    const x =
      Math.pow(1 - t, 3) * start.x +
      3 * Math.pow(1 - t, 2) * t * cp1.x +
      3 * (1 - t) * Math.pow(t, 2) * cp2.x +
      Math.pow(t, 3) * end.x;

    const y =
      Math.pow(1 - t, 3) * start.y +
      3 * Math.pow(1 - t, 2) * t * cp1.y +
      3 * (1 - t) * Math.pow(t, 2) * cp2.y +
      Math.pow(t, 3) * end.y;

    points.push({ x: Math.round(x), y: Math.round(y) });
  }

  return points;
}

//=============================================================================
// Random Delays
//=============================================================================

/**
 * Get random delay within range
 */
export function getRandomDelay(min: number, max: number): number {
  return min + Math.random() * (max - min);
}

/**
 * Get random delay from range tuple
 */
export function getRandomDelayFromRange(range: [number, number]): number {
  return getRandomDelay(range[0], range[1]);
}

/**
 * Add jitter to a delay
 */
export function addJitter(delay: number, jitterPercent: number = 0.1): number {
  const jitter = delay * jitterPercent * (Math.random() - 0.5) * 2;
  return Math.round(delay + jitter);
}
