export type DetectionReason = "unicode_tricks" | "random_chars" | "repetition";

export type DetectionResult = {
  isSuspicious: boolean;
  reasons: DetectionReason[];
};

const REPETITION_THRESHOLD = 8;
const ENTROPY_THRESHOLD = 3.4;

function detectUnicodeTricks(text: string): boolean {
  const zeroWidthPattern = /[\u200B-\u200D\uFEFF]/;
  const combiningPattern = /[\u0300-\u036F\u1AB0-\u1AFF\u20D0-\u20FF]/;
  const homoglyphPattern = /[\u0400-\u04FF]/;

  return (
    zeroWidthPattern.test(text) ||
    combiningPattern.test(text) ||
    homoglyphPattern.test(text)
  );
}

function calculateShannonEntropy(text: string): number {
  const freq: Record<string, number> = {};

  for (const char of text) {
    freq[char] = (freq[char] || 0) + 1;
  }

  let entropy = 0;
  const len = text.length;

  for (const count of Object.values(freq)) {
    const p = count / len;
    entropy -= p * Math.log2(p);
  }

  return entropy;
}

function detectRandomCharacters(text: string): boolean {
  const trimmed = text.trim();

  if (trimmed.length < 4) {
    return false;
  }

  const entropy = calculateShannonEntropy(trimmed);
  const hasMultipleNonAlpha = (/[^a-zA-Z\s]/g.test(trimmed) &&
    (trimmed.match(/[^a-zA-Z\s]/g) || []).length >= trimmed.length / 2);

  return entropy > ENTROPY_THRESHOLD && hasMultipleNonAlpha;
}

function detectRepetition(text: string): boolean {
  const trimmed = text.trim();

  if (trimmed.length < REPETITION_THRESHOLD) {
    return false;
  }

  const lastChar = trimmed[trimmed.length - 1];
  let consecutiveCount = 0;

  for (let i = trimmed.length - 1; i >= 0; i--) {
    if (trimmed[i] === lastChar) {
      consecutiveCount++;
    } else {
      break;
    }
  }

  if (consecutiveCount >= REPETITION_THRESHOLD) {
    return true;
  }

  for (let i = 0; i <= trimmed.length / 2; i++) {
    const chunk = trimmed.slice(0, i);
    if (chunk.length < 2) {
      continue;
    }

    const pattern = chunk.repeat(Math.ceil(trimmed.length / chunk.length));
    if (pattern.startsWith(trimmed) && chunk.length >= 3) {
      return true;
    }
  }

  return false;
}

export function detectPromptInjection(text: string): DetectionResult {
  if (!text || text.trim().length === 0) {
    return {
      isSuspicious: false,
      reasons: [],
    };
  }

  const reasons: DetectionReason[] = [];

  if (detectUnicodeTricks(text)) {
    reasons.push("unicode_tricks");
  }

  if (detectRandomCharacters(text)) {
    reasons.push("random_chars");
  }

  if (detectRepetition(text)) {
    reasons.push("repetition");
  }

  return {
    isSuspicious: reasons.length > 0,
    reasons,
  };
}

export const __testing = {
  calculateShannonEntropy,
  REPETITION_THRESHOLD,
  ENTROPY_THRESHOLD,
};
