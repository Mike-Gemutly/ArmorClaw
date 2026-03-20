package com.armorclaw.shared.secretary

data class TriageInput(
    val mode: SecretaryMode,
    val messageContent: String,
    val isVipSender: Boolean,
    val isCalendarLinked: Boolean
)

data class TriageResult(
    val score: Int,
    val priority: SecretaryPriority
)

class SecretaryTriage {

    private val urgentKeywords = listOf("urgent", "asap", "emergency")

    fun score(input: TriageInput): TriageResult {
        val score = calculateScore(input)
        val priority = scoreToPriority(score)

        return TriageResult(
            score = score,
            priority = priority
        )
    }

    private fun calculateScore(input: TriageInput): Int {
        val baseScore = calculateBaseScore(input)

        return when (input.mode) {
            SecretaryMode.FOCUS, SecretaryMode.SLEEP -> 0
            SecretaryMode.MEETING -> maxOf(0, baseScore - 1)
            SecretaryMode.NORMAL -> baseScore
        }
    }

    private fun calculateBaseScore(input: TriageInput): Int {
        var score = 0

        score += keywordScore(input.messageContent)
        if (input.isVipSender) score += 1
        if (input.isCalendarLinked) score += 2

        return score
    }

    private fun keywordScore(messageContent: String): Int {
        val lowerContent = messageContent.lowercase()
        return if (urgentKeywords.any { keyword -> lowerContent.contains(keyword) }) {
            2
        } else {
            0
        }
    }

    private fun scoreToPriority(score: Int): SecretaryPriority {
        return when {
            score == 0 -> SecretaryPriority.LOW
            score in 1..2 -> SecretaryPriority.HIGH
            score >= 3 -> SecretaryPriority.CRITICAL
            else -> SecretaryPriority.LOW
        }
    }
}
