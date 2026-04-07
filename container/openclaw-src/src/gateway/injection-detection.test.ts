import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { detectPromptInjection } from "./injection-detection.js";
import { __testing as controlPlaneRateLimitTesting, checkPromptInjection } from "./control-plane-rate-limit.js";
import type { GatewayClient } from "./server-methods/types.js";

describe("prompt injection detection - unicode tricks", () => {
  it("flags messages with combining diacritics", () => {
    // Example: "H̵̭̓ ELLO" - uses combining characters to hide text
    const result = detectPromptInjection("H̵̭̓ ELLO");
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("unicode_tricks");
  });

  it("flags messages with zero-width characters", () => {
    // Zero-width characters used to hide malicious content
    const result = detectPromptInjection("hello\u200Bworld");
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("unicode_tricks");
  });

  it("flags messages with homoglyphs", () => {
    // Lookalike characters used for phishing/injection
    const result = detectPromptInjection("аdmin"); // Cyrillic 'а' not Latin 'a'
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("unicode_tricks");
  });

  it("does NOT flag normal unicode text", () => {
    const result = detectPromptInjection("Hello 世界 🌍");
    expect(result.isSuspicious).toBe(false);
    expect(result.reasons).toHaveLength(0);
  });
});

describe("prompt injection detection - random characters", () => {
  it("flags messages with high entropy random chars", () => {
    const result = detectPromptInjection("asdf1234!@#$");
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("random_chars");
  });

  it("flags messages with mixed non-linguistic patterns", () => {
    const result = detectPromptInjection("xk29!@#mz84");
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("random_chars");
  });

  it("does NOT flag normal sentences with some numbers", () => {
    const result = detectPromptInjection("My phone number is 555-1234");
    expect(result.isSuspicious).toBe(false);
    expect(result.reasons).not.toContain("random_chars");
  });

  it("does NOT flag technical strings", () => {
    const result = detectPromptInjection("Error: connection_refused");
    expect(result.isSuspicious).toBe(false);
  });
});

describe("prompt injection detection - repetition patterns", () => {
  it("flags messages with excessive character repetition", () => {
    const result = detectPromptInjection("aaaaaaaa");
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("repetition");
  });

  it("flags messages with repeated sequences", () => {
    const result = detectPromptInjection("testtesttesttest");
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("repetition");
  });

  it("does NOT flag normal text with occasional repetition", () => {
    const result = detectPromptInjection("That's great! Really great!");
    expect(result.isSuspicious).toBe(false);
    expect(result.reasons).not.toContain("repetition");
  });

  it("does NOT flag natural language", () => {
    const result = detectPromptInjection("Please check the system status");
    expect(result.isSuspicious).toBe(false);
  });
});

describe("prompt injection detection - integration", () => {
  it("flags message with multiple suspicious patterns", () => {
    const result = detectPromptInjection("H̵̭̓ ELLOaaaaaaaa");
    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("unicode_tricks");
    expect(result.reasons).toContain("repetition");
  });

  it("does NOT flag legitimate agent messages", () => {
    const legitimateMessages = [
      "Please search for the best restaurants in NYC",
      "I need to check my email for the meeting details",
      "Can you help me fill out this form?",
      "What's the weather forecast for tomorrow?",
      "Please authenticate with the provided credentials",
    ];

    for (const msg of legitimateMessages) {
      const result = detectPromptInjection(msg);
      expect(result.isSuspicious).toBe(false);
      expect(result.reasons).toHaveLength(0);
    }
  });

  it("handles empty strings gracefully", () => {
    const result = detectPromptInjection("");
    expect(result.isSuspicious).toBe(false);
    expect(result.reasons).toHaveLength(0);
  });

  it("handles whitespace-only strings", () => {
    const result = detectPromptInjection("");
    expect(result.isSuspicious).toBe(false);
    expect(result.reasons).toHaveLength(0);
  });
});

describe("prompt injection detection - rate limit integration", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-02-19T00:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function buildClient(): GatewayClient {
    return {
      connect: {
        role: "operator",
        scopes: ["operator.admin"],
        client: {
          id: "openclaw-control-ui",
          version: "1.0.0",
          platform: "darwin",
          mode: "ui",
        },
        minProtocol: 1,
        maxProtocol: 1,
      },
      connId: "conn-1",
      clientIp: "10.0.0.5",
    } as GatewayClient;
  }

  it("detects unicode tricks in rate limit context", () => {
    const client = buildClient();
    const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const result = checkPromptInjection({
      message: "H̵̭̓ ELLO",
      client,
    });

    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("unicode_tricks");
    expect(result.detectedAt).toBe(Date.now());
    expect(consoleWarnSpy).toHaveBeenCalledWith(
      expect.stringContaining("Prompt injection detected"),
    );

    consoleWarnSpy.mockRestore();
  });

  it("detects random characters in rate limit context", () => {
    const client = buildClient();
    const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const result = checkPromptInjection({
      message: "asdf1234!@#$",
      client,
    });

    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("random_chars");
    expect(consoleWarnSpy).toHaveBeenCalledWith(
      expect.stringContaining("Prompt injection detected"),
    );

    consoleWarnSpy.mockRestore();
  });

  it("detects repetition in rate limit context", () => {
    const client = buildClient();
    const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const result = checkPromptInjection({
      message: "aaaaaaaa",
      client,
    });

    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("repetition");
    expect(consoleWarnSpy).toHaveBeenCalledWith(
      expect.stringContaining("Prompt injection detected"),
    );

    consoleWarnSpy.mockRestore();
  });

  it("does NOT flag legitimate messages in rate limit context", () => {
    const client = buildClient();
    const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const result = checkPromptInjection({
      message: "Please search for the best restaurants in NYC",
      client,
    });

    expect(result.isSuspicious).toBe(false);
    expect(result.reasons).toHaveLength(0);
    expect(consoleWarnSpy).not.toHaveBeenCalled();

    consoleWarnSpy.mockRestore();
  });

  it("handles null client gracefully", () => {
    const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const result = checkPromptInjection({
      message: "H̵̭̓ ELLO",
      client: null,
    });

    expect(result.isSuspicious).toBe(true);
    expect(result.reasons).toContain("unicode_tricks");
    expect(consoleWarnSpy).toHaveBeenCalledWith(
      expect.stringContaining("unknown-device|unknown-ip"),
    );

    consoleWarnSpy.mockRestore();
  });
});
