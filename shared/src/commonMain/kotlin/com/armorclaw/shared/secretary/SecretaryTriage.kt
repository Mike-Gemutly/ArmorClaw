package com.armorclaw.shared.secretary

data class TriageInput(
    val messageContent: String,
    val isVipSender: Boolean,
    val isCalendarLinked: Boolean
)

data class TriageResult(
    val priority: SecretaryPriority,
    val score: Int
)

/**
 * SecretaryTriage determines the priority level of messages based on multiple factors:
 * - Urgent keywords in message content
 * - VIP sender status
 * - Calendar-linked thread status
 *
 * The scoring is deterministic: same input always produces same output.
 */
class SecretaryTriage {

    private val urgentKeywords = mapOf(
        "urgent" to 2,
        "asap" to 2,
        "emergency" to 3
    )

    fun score(input: TriageInput): TriageResult {
        val score = calculateScore(input)
        val priority = scoreToPriority(score)

        return TriageResult(
            priority = priority,
            score = score
        )
    }

    private fun calculateScore(input: TriageInput): Int {
        var score = 0

        score += keywordScore(input.messageContent)
        if (input.isVipSender) score += 1
        if (input.isCalendarLinked) score += 2

        return score
    }

    private fun keywordScore(messageContent: String): Int {
        val lowerContent = messageContent.lowercase()
        return urgentKeywords
            .filter { (keyword, _) -> lowerContent.contains(keyword) }
            .values
            .sum()
    }

    private fun scoreToPriority(score: Int): SecretaryPriority {
        return when {
            score >= 3 -> SecretaryPriority.CRITICAL
            score >= 1 -> SecretaryPriority.HIGH
            else -> SecretaryPriority.NORMAL
        }
    }
}
