import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock the compaction module BEFORE importing the function under test
vi.mock("../compaction", () => ({
  estimateMessagesTokens: vi.fn(),
}));

import { shouldCompact, PROACTIVE_COMPACTION_RATIO, MIN_MESSAGES_FOR_COMPACTION } from "./should-compact.js";
import { estimateMessagesTokens } from "../compaction.js";

const mockedEstimate = vi.mocked(estimateMessagesTokens);

function makeMessages(count: number): { role: string; content: string }[] {
  return Array.from({ length: count }, (_, i) => ({
    role: i % 2 === 0 ? "user" : "assistant",
    content: `Message ${i}`,
  }));
}

describe("shouldCompact", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("exports named constants", () => {
    expect(PROACTIVE_COMPACTION_RATIO).toBe(0.75);
    expect(MIN_MESSAGES_FOR_COMPACTION).toBe(20);
  });

  it("threshold met → compact (tokenEstimate >= 75% of contextWindow)", () => {
    mockedEstimate.mockReturnValue(150_000);
    const result = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
    });
    expect(result.shouldCompact).toBe(true);
    expect(result.reason).toContain("150000");
    expect(result.reason).toContain("150000");
  });

  it("threshold not met → skip (tokens below 75%)", () => {
    mockedEstimate.mockReturnValue(100_000);
    const result = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
    });
    expect(result.shouldCompact).toBe(false);
    expect(result.reason).toContain("100000");
  });

  it("minimum message guard: messages.length < 20 → false (even if tokens high)", () => {
    mockedEstimate.mockReturnValue(500_000);
    const result = shouldCompact({
      messages: makeMessages(10),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
    });
    expect(result.shouldCompact).toBe(false);
    expect(result.reason).toContain("10");
    expect(result.reason).toContain("20");
    // estimateMessagesTokens should NOT be called (short-circuit)
    expect(mockedEstimate).not.toHaveBeenCalled();
  });

  it("anti-thrash guard: lastProactiveCompactionCount === compactionCount → false", () => {
    mockedEstimate.mockReturnValue(500_000);
    const result = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 5,
      lastProactiveCompactionCount: 5,
    });
    expect(result.shouldCompact).toBe(false);
    expect(result.reason).toContain("5");
    // estimateMessagesTokens should NOT be called (short-circuit)
    expect(mockedEstimate).not.toHaveBeenCalled();
  });

  it("custom threshold: threshold=0.5 triggers at 50%", () => {
    mockedEstimate.mockReturnValue(100_000);
    const result = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
      threshold: 0.5,
    });
    expect(result.shouldCompact).toBe(true);
    expect(result.reason).toContain("50%");

    mockedEstimate.mockReturnValue(99_999);
    const result2 = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
      threshold: 0.5,
    });
    expect(result2.shouldCompact).toBe(false);
  });

  it("reason strings are meaningful for logging", () => {
    const tooFew = shouldCompact({
      messages: makeMessages(5),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
    });
    expect(tooFew.reason).toBeTruthy();
    expect(typeof tooFew.reason).toBe("string");
    expect(tooFew.reason.length).toBeGreaterThan(5);

    const thrash = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 3,
      lastProactiveCompactionCount: 3,
    });
    expect(thrash.reason).toContain("3");

    mockedEstimate.mockReturnValue(50_000);
    const below = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
    });
    expect(below.reason).toContain("50000");

    mockedEstimate.mockReturnValue(160_000);
    const above = shouldCompact({
      messages: makeMessages(30),
      contextWindowTokens: 200_000,
      compactionCount: 0,
      lastProactiveCompactionCount: -1,
    });
    expect(above.reason).toContain("160000");
  });
});
