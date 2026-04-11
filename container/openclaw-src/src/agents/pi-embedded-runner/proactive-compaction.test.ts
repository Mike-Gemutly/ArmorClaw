import { beforeEach, describe, expect, it, vi } from "vitest";

const {
  mockShouldCompact,
  mockCompactEmbeddedPiSessionDirect,
  mockIsLikelyContextOverflowError,
  mockResolveContextWindowTokens,
} = vi.hoisted(() => ({
  mockShouldCompact: vi.fn(),
  mockCompactEmbeddedPiSessionDirect: vi.fn().mockResolvedValue({ ok: true, compacted: true }),
  mockIsLikelyContextOverflowError: vi.fn(),
  mockResolveContextWindowTokens: vi.fn().mockReturnValue(200_000),
}));

vi.mock("../compaction/should-compact.js", () => ({
  shouldCompact: mockShouldCompact,
  PROACTIVE_COMPACTION_RATIO: 0.75,
}));
vi.mock("./compact.js", () => ({
  compactEmbeddedPiSessionDirect: mockCompactEmbeddedPiSessionDirect,
}));
vi.mock("../pi-embedded-helpers/errors.js", () => ({
  isLikelyContextOverflowError: mockIsLikelyContextOverflowError,
}));
vi.mock("../compaction.js", () => ({
  resolveContextWindowTokens: mockResolveContextWindowTokens,
}));

import { registerProactiveCompaction, _resetProactiveCompactionState } from "./proactive-compaction.js";
import type {
  OpenClawPluginApi,
  PluginHookAgentEndEvent,
  PluginHookBeforePromptBuildEvent,
  PluginHookAgentContext,
} from "../../plugins/types.js";

const ctx: PluginHookAgentContext = {
  agentId: "agent-1",
  sessionKey: "test-session",
  sessionId: "session-1",
  workspaceDir: "/tmp/workspace",
};

function makeMockApi(): { api: OpenClawPluginApi; handlers: Map<string, unknown> } {
  const handlers = new Map<string, unknown>();
  const api = {
    id: "test-plugin",
    name: "test",
    config: {} as any,
    runtime: {} as any,
    logger: { info: vi.fn(), warn: vi.fn(), error: vi.fn(), debug: vi.fn() },
    on: vi.fn((hookName, handler) => {
      handlers.set(hookName, handler);
    }),
  } as unknown as OpenClawPluginApi;
  return { api, handlers };
}

describe("registerProactiveCompaction", () => {
  let api: OpenClawPluginApi;
  let handlers: Map<string, unknown>;

  beforeEach(() => {
    vi.resetAllMocks();
    mockCompactEmbeddedPiSessionDirect.mockResolvedValue({ ok: true, compacted: true });
    mockResolveContextWindowTokens.mockReturnValue(200_000);
    ({ api, handlers } = makeMockApi());
    registerProactiveCompaction(api);
  });

  it("registers both agent_end and before_prompt_build hooks", () => {
    expect(api.on).toHaveBeenCalledWith("agent_end", expect.any(Function));
    expect(api.on).toHaveBeenCalledWith("before_prompt_build", expect.any(Function));
  });

  describe("agent_end handler", () => {
    function getAgentEndHandler() {
      return handlers.get("agent_end") as (
        event: PluginHookAgentEndEvent,
        ctx: PluginHookAgentContext,
      ) => Promise<void> | void;
    }

    it("skips when success is false", async () => {
      const handler = getAgentEndHandler();
      const event: PluginHookAgentEndEvent = {
        messages: [{ role: "user", content: "hello" }],
        success: false,
        error: "something went wrong",
        sessionFile: "/tmp/session.jsonl",
      };

      await handler(event, ctx);

      expect(mockShouldCompact).not.toHaveBeenCalled();
      expect(mockCompactEmbeddedPiSessionDirect).not.toHaveBeenCalled();
    });

    it("skips when error is a context overflow error", async () => {
      const handler = getAgentEndHandler();
      mockIsLikelyContextOverflowError.mockReturnValue(true);
      const event: PluginHookAgentEndEvent = {
        messages: [{ role: "user", content: "hello" }],
        success: true,
        error: "context window exceeded",
        sessionFile: "/tmp/session.jsonl",
      };

      await handler(event, ctx);

      expect(mockIsLikelyContextOverflowError).toHaveBeenCalledWith("context window exceeded");
      expect(mockShouldCompact).not.toHaveBeenCalled();
      expect(mockCompactEmbeddedPiSessionDirect).not.toHaveBeenCalled();
    });

    it("skips when no sessionFile", async () => {
      const handler = getAgentEndHandler();
      const event: PluginHookAgentEndEvent = {
        messages: [{ role: "user", content: "hello" }],
        success: true,
      };

      await handler(event, ctx);

      expect(mockShouldCompact).not.toHaveBeenCalled();
      expect(mockCompactEmbeddedPiSessionDirect).not.toHaveBeenCalled();
    });

    it("triggers compaction when threshold exceeded and success is true", async () => {
      const handler = getAgentEndHandler();
      mockShouldCompact.mockReturnValue({ shouldCompact: true, reason: "75% threshold" });
      const event: PluginHookAgentEndEvent = {
        messages: [{ role: "user", content: "hello" }],
        success: true,
        sessionFile: "/tmp/session.jsonl",
      };

      await handler(event, ctx);

      expect(mockShouldCompact).toHaveBeenCalledWith(
        expect.objectContaining({
          contextWindowTokens: 200_000,
        }),
      );
      expect(mockCompactEmbeddedPiSessionDirect).toHaveBeenCalledWith(
        expect.objectContaining({
          sessionFile: "/tmp/session.jsonl",
        }),
      );
    });

    it("does NOT trigger when below threshold", async () => {
      const handler = getAgentEndHandler();
      mockShouldCompact.mockReturnValue({ shouldCompact: false, reason: "below threshold" });
      const event: PluginHookAgentEndEvent = {
        messages: [{ role: "user", content: "hello" }],
        success: true,
        sessionFile: "/tmp/session.jsonl",
      };

      await handler(event, ctx);

      expect(mockCompactEmbeddedPiSessionDirect).not.toHaveBeenCalled();
    });
  });

  describe("before_prompt_build handler", () => {
    function getBeforePromptBuildHandler() {
      return handlers.get("before_prompt_build") as (
        event: PluginHookBeforePromptBuildEvent,
        ctx: PluginHookAgentContext,
      ) => Promise<unknown> | unknown;
    }

    it("skips when within cooldown period", async () => {
      const handler = getBeforePromptBuildHandler();
      // First call sets the timestamp
      mockShouldCompact.mockReturnValue({ shouldCompact: true, reason: "75% threshold" });
      const event: PluginHookBeforePromptBuildEvent = {
        prompt: "hello",
        messages: [{ role: "user", content: "hello" }],
        sessionFile: "/tmp/session.jsonl",
      };

      await handler(event, ctx);
      expect(mockShouldCompact).toHaveBeenCalledTimes(1);

      // Second call within cooldown — should skip
      await handler(event, ctx);
      expect(mockShouldCompact).toHaveBeenCalledTimes(1);
    });

    it("triggers compaction after cooldown when threshold exceeded", async () => {
      const handler = getBeforePromptBuildHandler();
      mockShouldCompact.mockReturnValue({ shouldCompact: true, reason: "75% threshold" });
      const event: PluginHookBeforePromptBuildEvent = {
        prompt: "hello",
        messages: [{ role: "user", content: "hello" }],
        sessionFile: "/tmp/session.jsonl",
      };

      vi.useFakeTimers({ now: Date.now() + 120_000 });

      // First call — passes cooldown because fake time is well past any previous check
      await handler(event, ctx);

      // Advance time past cooldown (60 seconds)
      vi.advanceTimersByTime(61_000);

      // Second call — should re-check
      await handler(event, ctx);
      expect(mockShouldCompact).toHaveBeenCalledTimes(2);
      expect(mockCompactEmbeddedPiSessionDirect).toHaveBeenCalledTimes(2);

      vi.useRealTimers();
    });

    it("returns undefined (does not modify prompt)", async () => {
      const handler = getBeforePromptBuildHandler();
      mockShouldCompact.mockReturnValue({ shouldCompact: false, reason: "below threshold" });
      const event: PluginHookBeforePromptBuildEvent = {
        prompt: "hello",
        messages: [{ role: "user", content: "hello" }],
        sessionFile: "/tmp/session.jsonl",
      };

      const result = await handler(event, ctx);
      expect(result).toBeUndefined();
    });
  });

  describe("anti-thrash", () => {
    it("passes compactionCount=messages.length and updates lastProactiveCompactionCount after compact", async () => {
      vi.resetAllMocks();
      _resetProactiveCompactionState();
      mockCompactEmbeddedPiSessionDirect.mockResolvedValue({ ok: true, compacted: true });
      mockResolveContextWindowTokens.mockReturnValue(200_000);
      ({ api, handlers } = makeMockApi());
      registerProactiveCompaction(api);

      const handler = handlers.get("agent_end") as (
        event: PluginHookAgentEndEvent,
        ctx: PluginHookAgentContext,
      ) => Promise<void> | void;

      const messages = Array.from({ length: 50 }, (_, i) => ({ role: "user", content: `msg-${i}` }));
      mockShouldCompact.mockReturnValue({ shouldCompact: true, reason: "75% threshold" });
      const event: PluginHookAgentEndEvent = {
        messages,
        success: true,
        sessionFile: "/tmp/session.jsonl",
      };

      await handler(event, ctx);
      expect(mockShouldCompact).toHaveBeenCalledWith(
        expect.objectContaining({
          compactionCount: 50,
          lastProactiveCompactionCount: -1,
        }),
      );
      expect(mockCompactEmbeddedPiSessionDirect).toHaveBeenCalledTimes(1);

      await handler(event, ctx);
      expect(mockShouldCompact).toHaveBeenCalledWith(
        expect.objectContaining({
          compactionCount: 50,
          lastProactiveCompactionCount: 50,
        }),
      );
    });
  });
});
