package com.armorclaw.shared.secretary

/**
 * BriefingContext - data class for briefing context input.
 * Contains all information needed to generate a morning briefing or evening review.
 */
data class BriefingContext(
    val unreadCount: Int,
    val nextMeeting: String?,
    val pendingApprovals: Int,
    val lastMorningBriefingDate: Long?,
    val eveningReviewEnabled: Boolean = true,
    val lastEveningReviewDate: Long? = null
)

/**
 * BriefingResult - result data class for briefing generation.
 * Contains the generated briefing card content.
 */
data class BriefingResult(
    val title: String,
    val description: String,
    val primaryAction: SecretaryAction
)

/**
 * SecretaryBriefingEngine - deterministic briefing generation engine.
 *
 * Rules:
 * - Morning briefing appears only in configured time window (7am-9am)
 * - Evening review appears only in configured time window (5pm-9pm) if enabled
 * - No duplication on the same day (tracks last briefing date)
 * - Requires sufficient context (unread messages, meetings, or approvals)
 * - Generates stable summaries including relevant context
 */
class SecretaryBriefingEngine {

    companion object {
        private const val MORNING_START_HOUR = 7
        private const val MORNING_END_HOUR = 9
        private const val EVENING_START_HOUR = 17
        private const val EVENING_END_HOUR = 21
    }

    /**
     * Generate a morning briefing card if conditions are met.
     *
     * @param currentTime Current time in milliseconds since epoch
     * @param context Briefing context with unread, meetings, approvals
     * @return BriefingResult if conditions met, null otherwise
     */
    fun generateMorningBriefing(currentTime: Long, context: BriefingContext): BriefingResult? {
        val hourOfDay = getHourOfDay(currentTime)

        if (hourOfDay !in MORNING_START_HOUR until MORNING_END_HOUR) {
            return null
        }

        val currentDate = getCurrentDate(currentTime)
        if (context.lastMorningBriefingDate == currentDate) {
            return null
        }

        if (!hasSufficientContext(context)) {
            return null
        }

        return generateBriefing(
            title = "Good morning",
            context = context
        )
    }

    /**
     * Generate an evening review card if conditions are met.
     *
     * @param currentTime Current time in milliseconds since epoch
     * @param context Briefing context with unread, meetings, approvals
     * @return BriefingResult if conditions met, null otherwise
     */
    fun generateEveningReview(currentTime: Long, context: BriefingContext): BriefingResult? {
        if (!context.eveningReviewEnabled) {
            return null
        }

        val hourOfDay = getHourOfDay(currentTime)

        if (hourOfDay !in EVENING_START_HOUR until EVENING_END_HOUR) {
            return null
        }

        val currentDate = getCurrentDate(currentTime)
        if (context.lastEveningReviewDate == currentDate) {
            return null
        }

        if (!hasSufficientContext(context)) {
            return null
        }

        return generateBriefing(
            title = "Good evening",
            context = context
        )
    }

    private fun hasSufficientContext(context: BriefingContext): Boolean {
        return context.unreadCount > 0 ||
               context.nextMeeting != null ||
               context.pendingApprovals > 0
    }

    private fun generateBriefing(title: String, context: BriefingContext): BriefingResult {
        val description = buildString {
            val parts = mutableListOf<String>()

            if (context.unreadCount > 0) {
                parts.add("${context.unreadCount} ${if (context.unreadCount == 1) "unread message" else "unread messages"}")
            }

            if (context.nextMeeting != null) {
                parts.add("Next meeting: ${context.nextMeeting}")
            }

            if (context.pendingApprovals > 0) {
                parts.add("${context.pendingApprovals} ${if (context.pendingApprovals == 1) "approval" else "approvals"} pending")
            }

            append(parts.joinToString(". "))
        }

        val primaryAction = SecretaryAction.Local(LocalSecretaryAction.NAV_CHAT)

        return BriefingResult(
            title = title,
            description = description,
            primaryAction = primaryAction
        )
    }

    /**
     * Get hour of day from timestamp (0-23).
     * Simplified implementation for determinism.
     */
    private fun getHourOfDay(timestamp: Long): Int {
        val millisecondsPerDay = 24 * 60 * 60 * 1000L
        val millisecondsPerHour = 60 * 60 * 1000L
        return ((timestamp % millisecondsPerDay) / millisecondsPerHour).toInt()
    }

    /**
     * Get current date (day timestamp) from full timestamp.
     * Simplified implementation for determinism.
     */
    private fun getCurrentDate(timestamp: Long): Long {
        val millisecondsPerDay = 24 * 60 * 60 * 1000L
        return (timestamp / millisecondsPerDay) * millisecondsPerDay
    }
}
