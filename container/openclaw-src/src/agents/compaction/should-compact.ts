import type { AgentMessage } from "@mariozechner/pi-agent-core";
import { estimateMessagesTokens } from "../compaction.js";

export const PROACTIVE_COMPACTION_RATIO = 0.75;
export const MIN_MESSAGES_FOR_COMPACTION = 20;

export function shouldCompact(params: {
  messages: { role: string; content: string }[];
  contextWindowTokens: number;
  compactionCount: number;
  lastProactiveCompactionCount: number;
  threshold?: number;
}): { shouldCompact: boolean; reason: string } {
  const threshold = params.threshold ?? PROACTIVE_COMPACTION_RATIO;

  if (params.messages.length < MIN_MESSAGES_FOR_COMPACTION) {
    return {
      shouldCompact: false,
      reason: `Too few messages (${params.messages.length} < ${MIN_MESSAGES_FOR_COMPACTION})`,
    };
  }

  if (params.lastProactiveCompactionCount === params.compactionCount) {
    return {
      shouldCompact: false,
      reason: `Already compacted at count ${params.compactionCount}`,
    };
  }

  const tokenEstimate = estimateMessagesTokens(params.messages as AgentMessage[]);
  const thresholdTokens = params.contextWindowTokens * threshold;

  if (tokenEstimate >= thresholdTokens) {
    return {
      shouldCompact: true,
      reason: `Token usage ${tokenEstimate} >= ${thresholdTokens.toFixed(0)} (${(threshold * 100).toFixed(0)}%)`,
    };
  }

  return {
    shouldCompact: false,
    reason: `Token usage ${tokenEstimate} < ${thresholdTokens.toFixed(0)} (${(threshold * 100).toFixed(0)}%)`,
  };
}
