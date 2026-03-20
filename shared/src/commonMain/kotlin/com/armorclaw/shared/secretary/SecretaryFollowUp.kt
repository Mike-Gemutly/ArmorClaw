package com.armorclaw.shared.secretary

data class FollowUpThread(
    val threadId: String,
    val lastOutboundTimestamp: Long?,
    val lastInboundTimestamp: Long?
)

data class FollowUpContext(
    val threads: List<FollowUpThread>,
    val currentTime: Long,
    val followUpThresholdMs: Long
)

data class FollowUpItem(
    val threadId: String,
    val stalenessDurationMs: Long,
    val recommendedAction: String
)

data class FollowUpResult(
    val followUps: List<FollowUpItem>
)

class SecretaryFollowUp {

    fun detectStaleThreads(context: FollowUpContext): FollowUpResult {
        val followUps = context.threads
            .asSequence()
            .filter { it.needsFollowUp(context) }
            .map { it.toFollowUpItem(context) }
            .sortedByDescending { it.stalenessDurationMs }
            .toList()

        return FollowUpResult(followUps)
    }

    private fun FollowUpThread.needsFollowUp(context: FollowUpContext): Boolean {
        val outboundTs = lastOutboundTimestamp ?: return false
        val inboundTs = lastInboundTimestamp

        val stalenessMs = context.currentTime - outboundTs
        if (stalenessMs < context.followUpThresholdMs) return false

        if (inboundTs != null && inboundTs > outboundTs) return false

        return true
    }

    private fun FollowUpThread.toFollowUpItem(context: FollowUpContext): FollowUpItem {
        val stalenessMs = context.currentTime - (lastOutboundTimestamp ?: context.currentTime)

        return FollowUpItem(
            threadId = threadId,
            stalenessDurationMs = stalenessMs,
            recommendedAction = "Send a follow-up message"
        )
    }
}
