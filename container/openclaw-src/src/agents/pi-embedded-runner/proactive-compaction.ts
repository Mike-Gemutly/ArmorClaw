import { shouldCompact } from "../compaction/should-compact.js";
import { compactEmbeddedPiSessionDirect } from "./compact.js";
import { isLikelyContextOverflowError } from "../pi-embedded-helpers/errors.js";
import { resolveContextWindowTokens } from "../compaction.js";
import type { OpenClawPluginApi, PluginHookAgentContext } from "../../plugins/types.js";

let lastProactiveCompactionCount = -1;
let lastSafetyNetCheckTimestamp = 0;

const SAFETY_NET_COOLDOWN_MS = 60_000;

export function registerProactiveCompaction(api: OpenClawPluginApi): void {
  api.on("agent_end", handleAgentEnd);
  api.on("before_prompt_build", handleBeforePromptBuild);
}

export function _resetProactiveCompactionState(): void {
  lastProactiveCompactionCount = -1;
  lastSafetyNetCheckTimestamp = 0;
}

async function handleAgentEnd(
  event: { messages: unknown[]; success: boolean; error?: string; sessionFile?: string },
  ctx: PluginHookAgentContext,
): Promise<void> {
  if (!event.success) return;
  if (isLikelyContextOverflowError(event.error)) return;
  if (!event.sessionFile) return;

  const contextWindowTokens = resolveContextWindowTokens();
  const msgCount = event.messages.length;
  const result = shouldCompact({
    messages: event.messages as { role: string; content: string }[],
    contextWindowTokens,
    compactionCount: msgCount,
    lastProactiveCompactionCount,
  });

  if (result.shouldCompact) {
    lastProactiveCompactionCount = msgCount;
    compactEmbeddedPiSessionDirect({
      sessionId: ctx.sessionId ?? "unknown",
      sessionFile: event.sessionFile,
      workspaceDir: ctx.workspaceDir ?? process.cwd(),
      trigger: "overflow",
    }).catch((err) => {
      console.warn("[proactive-compaction] agent_end fire-and-forget failed:", err);
    });
  }
}

async function handleBeforePromptBuild(
  event: { prompt: string; messages: unknown[]; sessionFile?: string },
  ctx: PluginHookAgentContext,
): Promise<void> {
  if (!event.sessionFile) return;

  const now = Date.now();
  if (now - lastSafetyNetCheckTimestamp < SAFETY_NET_COOLDOWN_MS) return;
  lastSafetyNetCheckTimestamp = now;

  const contextWindowTokens = resolveContextWindowTokens();
  const msgCount = event.messages.length;
  const result = shouldCompact({
    messages: event.messages as { role: string; content: string }[],
    contextWindowTokens,
    compactionCount: msgCount,
    lastProactiveCompactionCount,
  });

  if (result.shouldCompact) {
    lastProactiveCompactionCount = msgCount;
    await compactEmbeddedPiSessionDirect({
      sessionId: ctx.sessionId ?? "unknown",
      sessionFile: event.sessionFile,
      workspaceDir: ctx.workspaceDir ?? process.cwd(),
      trigger: "overflow",
    });
  }
}
