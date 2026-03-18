package com.armorclaw.shared.domain.model

/**
 * Attention Item
 *
 * Represents an item that requires user attention in the Mission Control Dashboard.
 * These are displayed in the "Needs Attention" queue for user intervention.
 *
 * ## Types
 * - PII_REQUEST: Agent requesting access to sensitive data
 * - CAPTCHA: Agent stuck on CAPTCHA challenge
 * - TWO_FA: Agent waiting for 2FA code
 * - APPROVAL: Agent waiting for explicit approval
 * - ERROR: Agent encountered an error and needs guidance
 *
 * ## Priority Levels
 * - CRITICAL: Immediate attention required (payment, security)
 * - HIGH: Urgent but not time-sensitive
 * - MEDIUM: Standard priority
 * - LOW: Can be addressed when convenient
 */
sealed class AttentionItem {
    abstract val id: String
    abstract val agentId: String
    abstract val agentName: String
    abstract val roomId: String
    abstract val priority: AttentionPriority
    abstract val timestamp: Long
    abstract val title: String
    abstract val description: String

    /**
     * PII Access Request attention item
     */
    data class PiiRequest(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val piiRequest: PiiAccessRequest
    ) : AttentionItem() {
        override val priority: AttentionPriority
            get() = when {
                piiRequest.fields.any { it.sensitivity == SensitivityLevel.CRITICAL } -> AttentionPriority.CRITICAL
                piiRequest.fields.any { it.sensitivity == SensitivityLevel.HIGH } -> AttentionPriority.HIGH
                else -> AttentionPriority.MEDIUM
            }
        override val title: String = "Data Access Request"
        override val description: String = piiRequest.reason
    }

    /**
     * CAPTCHA challenge attention item
     */
    data class CaptchaChallenge(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val siteUrl: String
    ) : AttentionItem() {
        override val priority: AttentionPriority = AttentionPriority.HIGH
        override val title: String = "CAPTCHA Required"
        override val description: String = "Solve CAPTCHA on $siteUrl"
    }

    /**
     * Two-factor authentication attention item
     */
    data class TwoFactorAuth(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val service: String
    ) : AttentionItem() {
        override val priority: AttentionPriority = AttentionPriority.HIGH
        override val title: String = "2FA Code Required"
        override val description: String = "Enter 2FA code for $service"
    }

    /**
     * Generic approval request attention item
     */
    data class ApprovalRequest(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val action: String,
        val context: String
    ) : AttentionItem() {
        override val priority: AttentionPriority = AttentionPriority.MEDIUM
        override val title: String = "Approval Required"
        override val description: String = "$action - $context"
    }

    /**
     * Error requiring user guidance
     */
    data class ErrorState(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val errorMessage: String,
        val recoverable: Boolean
    ) : AttentionItem() {
        override val priority: AttentionPriority = if (recoverable) AttentionPriority.MEDIUM else AttentionPriority.HIGH
        override val title: String = if (recoverable) "Agent Needs Guidance" else "Agent Stopped"
        override val description: String = errorMessage
    }
}

/**
 * Priority levels for attention items
 */
enum class AttentionPriority {
    CRITICAL,
    HIGH,
    MEDIUM,
    LOW;

    fun toDisplayString(): String = when (this) {
        CRITICAL -> "Critical"
        HIGH -> "High"
        MEDIUM -> "Medium"
        LOW -> "Low"
    }
}

/**
 * Quick Action types for Mission Control
 */
sealed class QuickAction {
    abstract val label: String
    abstract val icon: String
    abstract val destructive: Boolean

    data object EmergencyStop : QuickAction() {
        override val label: String = "Emergency Stop"
        override val icon: String = "stop"
        override val destructive: Boolean = true
    }

    data object PauseAll : QuickAction() {
        override val label: String = "Pause All"
        override val icon: String = "pause"
        override val destructive: Boolean = false
    }

    data object LockVault : QuickAction() {
        override val label: String = "Lock Vault"
        override val icon: String = "lock"
        override val destructive: Boolean = false
    }

    data object ResumeAll : QuickAction() {
        override val label: String = "Resume All"
        override val icon: String = "play_arrow"
        override val destructive: Boolean = false
    }
}

/**
 * Agent summary for Mission Control dashboard
 */
data class AgentSummary(
    val agentId: String,
    val agentName: String,
    val status: AgentTaskStatus,
    val currentTask: String?,
    val roomId: String,
    val roomName: String?,
    val progress: Float,
    val lastActivity: Long
)
