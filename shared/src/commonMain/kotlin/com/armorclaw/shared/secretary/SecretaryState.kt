package com.armorclaw.shared.secretary

import kotlinx.serialization.Serializable

/**
 * State machine for Secretary system.
 * Phase 1 implementation - keeps it lean with only core states.
 */
sealed class SecretaryState {
    /**
     * Idle state - no active operations.
     */
    object Idle : SecretaryState()

    /**
     * Observing state - waiting for Matrix events or context data.
     * @param reason Optional description of what's being observed.
     */
    data class Observing(val reason: String? = null) : SecretaryState()

    /**
     * Thinking state - analyzing events and considering next actions.
     * @param task Description of the analysis task.
     */
    data class Thinking(val task: String) : SecretaryState()

    /**
     * Proposing state - showing proactive cards for user consideration.
     * @param cards List of proactive cards to display.
     */
    data class Proposing(val cards: List<ProactiveCard>) : SecretaryState()

    /**
     * Error state - an error occurred in the Secretary system.
     * @param message Error description.
     * @param recoverable Whether the error is recoverable.
     */
    data class Error(
        val message: String,
        val recoverable: Boolean = true
    ) : SecretaryState()
}
