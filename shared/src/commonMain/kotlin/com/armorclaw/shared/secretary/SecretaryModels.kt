package com.armorclaw.shared.secretary

import kotlinx.serialization.Serializable

/**
 * SecretaryPriority defines the urgency level of proactive cards.
 */
enum class SecretaryPriority {
    LOW,
    NORMAL,
    HIGH,
    CRITICAL
}

/**
 * Sealed class for all Secretary actions.
 * Local actions are handled entirely within the app.
 * Bridge-backed actions require external server execution.
 */
@Serializable
sealed class SecretaryAction {
    @Serializable
    data class Local(val action: LocalSecretaryAction) : SecretaryAction()
}

/**
 * Local Secretary actions that don't require Bridge RPC calls.
 */
@Serializable
enum class LocalSecretaryAction {
    NAV_CHAT,
    OPEN_MESSAGE,
    DISMISS_CARD,
    SNOOZE_CARD
}

/**
 * Reasons for creating a proactive card.
 */
enum class SecretaryCardReason {
    URGENT_KEYWORD,
    VIP_SENDER,
    MORNING_BRIEFING,
    EVENING_REVIEW
}

/**
 * ProactiveCard represents a notification card shown to the user.
 * Only contains fields needed for first-phase use case.
 */
@Serializable
data class ProactiveCard(
    val id: String,
    val title: String,
    val description: String,
    val priority: SecretaryPriority,
    val reason: SecretaryCardReason,
    val primaryAction: SecretaryAction,
    val dismissible: Boolean = true
)
