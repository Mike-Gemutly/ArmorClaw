package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

/**
 * Agent Task Status Event
 *
 * Represents real-time status updates from AI agents executing tasks.
 * Used by the Control Plane to track agent activity and display status to users.
 *
 * Note: This is distinct from AgentStatus (online/busy/offline) which tracks
 * agent availability. AgentTaskStatus tracks the progress of specific tasks.
 *
 * ## Usage
 * ```kotlin
 * val statusEvent = AgentTaskStatusEvent(
 *     agentId = "agent_browse_001",
 *     status = AgentTaskStatus.FORM_FILLING,
 *     timestamp = System.currentTimeMillis(),
 *     metadata = mapOf("step" to "2", "url" to "https://example.com/checkout")
 * )
 * ```
 *
 * ## Status Flow
 * ```
 * IDLE → BROWSING → FORM_FILLING → PROCESSING_PAYMENT → COMPLETE
 *                 ↓
 *           AWAITING_CAPTCHA / AWAITING_2FA / AWAITING_APPROVAL
 *                 ↓
 *                    ERROR
 * ```
 */
@Serializable
data class AgentTaskStatusEvent(
    val agentId: String,
    val status: AgentTaskStatus,
    val timestamp: Long,
    val metadata: Map<String, String>? = null
)

/**
 * Agent task status enum representing the current state of an agent's task.
 *
 * Note: This is distinct from AgentStatus (in UnifiedMessage.kt) which tracks
 * agent online/offline state. This tracks task execution progress.
 */
@Serializable
enum class AgentTaskStatus {
    /** Agent is idle and not currently working on any task */
    IDLE,

    /** Agent is navigating/browsing a website */
    BROWSING,

    /** Agent is filling out a form */
    FORM_FILLING,

    /** Agent is processing a payment transaction */
    PROCESSING_PAYMENT,

    /** Agent encountered a CAPTCHA and needs user intervention */
    AWAITING_CAPTCHA,

    /** Agent encountered 2FA and needs user intervention */
    AWAITING_2FA,

    /** Agent needs user approval to proceed with sensitive action */
    AWAITING_APPROVAL,

    /** Agent encountered an error */
    ERROR,

    /** Agent completed its task successfully */
    COMPLETE;

    /**
     * Returns true if this status requires user intervention
     */
    fun requiresIntervention(): Boolean {
        return this in listOf(AWAITING_CAPTCHA, AWAITING_2FA, AWAITING_APPROVAL)
    }

    /**
     * Returns true if this status indicates active work
     */
    fun isActive(): Boolean {
        return this in listOf(BROWSING, FORM_FILLING, PROCESSING_PAYMENT)
    }

    /**
     * Returns a user-friendly display string
     */
    fun toDisplayString(): String {
        return when (this) {
            IDLE -> "Idle"
            BROWSING -> "Browsing..."
            FORM_FILLING -> "Filling form..."
            PROCESSING_PAYMENT -> "Processing payment..."
            AWAITING_CAPTCHA -> "Waiting for CAPTCHA"
            AWAITING_2FA -> "Waiting for 2FA"
            AWAITING_APPROVAL -> "Waiting for approval"
            ERROR -> "Error"
            COMPLETE -> "Complete"
        }
    }
}

/**
 * Type alias for backward compatibility
 * @deprecated Use AgentTaskStatusEvent instead
 */
typealias AgentStatusEvent = AgentTaskStatusEvent
