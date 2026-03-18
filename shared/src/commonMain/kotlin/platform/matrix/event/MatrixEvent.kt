package com.armorclaw.shared.platform.matrix.event

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.datetime.Instant

/**
 * Matrix Event Types for ArmorClaw
 *
 * These are custom Matrix event types used for ArmorClaw-specific functionality.
 * They are sent as regular Matrix events (encrypted if the room is encrypted)
 * and processed by the ControlPlaneStore on the client.
 *
 * ## Event Type Convention
 * All ArmorClaw events use reverse-domain notation: `com.armorclaw.{category}.{action}`
 *
 * ## Event Flow
 * ```
 * Bridge/Agent → Matrix Event → Client receives via sync → ControlPlaneStore → UI
 * ```
 */

// ========================================================================
// Base Event Types
// ========================================================================

/**
 * Known Matrix event types
 */
object MatrixEventType {
    const val ROOM_MESSAGE = "m.room.message"
    const val ROOM_MEMBER = "m.room.member"
    const val ROOM_NAME = "m.room.name"
    const val ROOM_TOPIC = "m.room.topic"
    const val ROOM_ENCRYPTION = "m.room.encryption"
    const val ROOM_AVATAR = "m.room.avatar"
    const val ROOM_REDACTION = "m.room.redaction"
    const val REACTION = "m.reaction"
    const val TYPING = "m.typing"
    const val READ_RECEIPT = "m.read"
    const val PRESENCE = "m.presence"
    const val CALL_INVITE = "m.call.invite"
    const val CALL_ANSWER = "m.call.answer"
    const val CALL_HANGUP = "m.call.hangup"
    const val CALL_CANDIDATES = "m.call.candidates"
}

/**
 * ArmorClaw custom event types
 */
object ArmorClawEventType {
    // Workflow events
    const val WORKFLOW_STARTED = "com.armorclaw.workflow.started"
    const val WORKFLOW_STEP = "com.armorclaw.workflow.step"
    const val WORKFLOW_COMPLETED = "com.armorclaw.workflow.completed"
    const val WORKFLOW_FAILED = "com.armorclaw.workflow.failed"

    // Agent events
    const val AGENT_TASK_STARTED = "com.armorclaw.agent.task_started"
    const val AGENT_TASK_PROGRESS = "com.armorclaw.agent.task_progress"
    const val AGENT_TASK_COMPLETE = "com.armorclaw.agent.task_complete"
    const val AGENT_THINKING = "com.armorclaw.agent.thinking"

    // System events
    const val BUDGET_WARNING = "com.armorclaw.budget.warning"
    const val BRIDGE_CONNECTED = "com.armorclaw.bridge.connected"
    const val BRIDGE_DISCONNECTED = "com.armorclaw.bridge.disconnected"
    const val PLATFORM_MESSAGE = "com.armorclaw.platform.message"

    /**
     * Check if an event type is an ArmorClaw custom event
     */
    fun isArmorClawEvent(eventType: String): Boolean {
        return eventType.startsWith("com.armorclaw.")
    }

    /**
     * Check if an event type is a workflow event
     */
    fun isWorkflowEvent(eventType: String): Boolean {
        return eventType.startsWith("com.armorclaw.workflow.")
    }

    /**
     * Check if an event type is an agent event
     */
    fun isAgentEvent(eventType: String): Boolean {
        return eventType.startsWith("com.armorclaw.agent.")
    }
}

// ========================================================================
// Generic Matrix Event Wrapper
// ========================================================================

/**
 * Generic Matrix event
 */
@Serializable
data class MatrixEvent(
    val eventId: String,
    val roomId: String,
    val senderId: String,
    val type: String,
    val content: String, // JSON-encoded content
    val timestamp: Long,
    val transactionId: String? = null,
    val relatesTo: EventRelation? = null
)

/**
 * Event relation (for replies, threads, edits)
 */
@Serializable
data class EventRelation(
    val eventId: String,
    val type: RelationType? = null
)

@Serializable
enum class RelationType {
    @SerialName("m.in_reply_to")
    IN_REPLY_TO,

    @SerialName("m.thread")
    THREAD,

    @SerialName("m.replace")
    REPLACE,

    @SerialName("m.annotation")
    ANNOTATION
}

// ========================================================================
// Workflow Events
// ========================================================================

/**
 * Workflow started event
 *
 * Sent when a new workflow is initiated in a room.
 */
@Serializable
@SerialName("com.armorclaw.workflow.started")
data class WorkflowStartedEvent(
    val workflowId: String,
    val workflowType: String,
    val initiatedBy: String,
    val timestamp: Long,
    val parameters: Map<String, String> = emptyMap(),
    val estimatedDuration: Long? = null
)

/**
 * Workflow step event
 *
 * Sent when a workflow step starts, progresses, or completes.
 */
@Serializable
@SerialName("com.armorclaw.workflow.step")
data class WorkflowStepEvent(
    val workflowId: String,
    val stepId: String,
    val stepName: String,
    val stepIndex: Int,
    val totalSteps: Int,
    val status: StepStatus,
    val output: String? = null,
    val error: String? = null,
    val timestamp: Long
)

@Serializable
enum class StepStatus {
    @SerialName("pending")
    PENDING,

    @SerialName("running")
    RUNNING,

    @SerialName("completed")
    COMPLETED,

    @SerialName("failed")
    FAILED,

    @SerialName("skipped")
    SKIPPED
}

/**
 * Workflow completed event
 */
@Serializable
@SerialName("com.armorclaw.workflow.completed")
data class WorkflowCompletedEvent(
    val workflowId: String,
    val workflowType: String,
    val success: Boolean,
    val result: String? = null,
    val error: String? = null,
    val duration: Long,
    val timestamp: Long
)

/**
 * Workflow failed event
 */
@Serializable
@SerialName("com.armorclaw.workflow.failed")
data class WorkflowFailedEvent(
    val workflowId: String,
    val workflowType: String,
    val failedAtStep: String,
    val error: String,
    val recoverable: Boolean,
    val timestamp: Long
)

// ========================================================================
// Agent Events
// ========================================================================

/**
 * Agent task started event
 */
@Serializable
@SerialName("com.armorclaw.agent.task_started")
data class AgentTaskStartedEvent(
    val agentId: String,
    val agentName: String,
    val taskId: String,
    val taskType: String,
    val roomId: String,
    val timestamp: Long
)

/**
 * Agent task progress event
 */
@Serializable
@SerialName("com.armorclaw.agent.task_progress")
data class AgentTaskProgressEvent(
    val agentId: String,
    val taskId: String,
    val progress: Float, // 0.0 to 1.0
    val message: String,
    val timestamp: Long
)

/**
 * Agent task complete event
 */
@Serializable
@SerialName("com.armorclaw.agent.task_complete")
data class AgentTaskCompleteEvent(
    val agentId: String,
    val agentName: String,
    val taskId: String,
    val taskType: String,
    val success: Boolean,
    val result: String? = null,
    val error: String? = null,
    val duration: Long,
    val timestamp: Long
)

/**
 * Agent thinking event
 *
 * Sent when an agent is processing a request. Useful for showing
 * "typing" indicators or loading states.
 */
@Serializable
@SerialName("com.armorclaw.agent.thinking")
data class AgentThinkingEvent(
    val agentId: String,
    val agentName: String,
    val message: String? = null,
    val timestamp: Long
)

// ========================================================================
// System Events
// ========================================================================

/**
 * Budget warning event
 *
 * Sent when a user's budget usage crosses a threshold.
 * This is sent via Matrix (not RPC) so it's encrypted and
 * visible in the room timeline.
 */
@Serializable
@SerialName("com.armorclaw.budget.warning")
data class BudgetWarningEvent(
    val userId: String,
    val currentSpend: Double,
    val limit: Double,
    val percentageUsed: Double,
    val warningLevel: WarningLevel,
    val timestamp: Long
)

@Serializable
enum class WarningLevel {
    @SerialName("info")
    INFO, // 50%

    @SerialName("warning")
    WARNING, // 75%

    @SerialName("critical")
    CRITICAL, // 90%

    @SerialName("exceeded")
    EXCEEDED // 100%+
}

/**
 * Bridge connected event
 *
 * Sent when the ArmorClaw bridge connects to an external platform.
 */
@Serializable
@SerialName("com.armorclaw.bridge.connected")
data class BridgeConnectedEvent(
    val platformType: String, // slack, discord, teams, whatsapp
    val platformName: String,
    val status: String,
    val timestamp: Long
)

/**
 * Bridge disconnected event
 */
@Serializable
@SerialName("com.armorclaw.bridge.disconnected")
data class BridgeDisconnectedEvent(
    val platformType: String,
    val platformName: String,
    val reason: String?,
    val timestamp: Long
)

/**
 * Platform message event
 *
 * A message that originated from an external platform (Slack, Discord, etc.)
 * and was bridged to Matrix.
 */
@Serializable
@SerialName("com.armorclaw.platform.message")
data class PlatformMessageEvent(
    val platformType: String,
    val platformMessageId: String,
    val originalSenderId: String,
    val originalSenderName: String,
    val bridgedAt: Long,
    val edited: Boolean = false,
    val deleted: Boolean = false
)

// ========================================================================
// Event Parsing Utilities
// ========================================================================

/**
 * Parse a Matrix event content into a typed event
 */
object MatrixEventParser {
    /**
     * Check if this event is an ArmorClaw custom event
     */
    fun isArmorClawEvent(event: MatrixEvent): Boolean {
        return ArmorClawEventType.isArmorClawEvent(event.type)
    }

    /**
     * Get the category of an ArmorClaw event
     */
    fun getEventCategory(eventType: String): EventCategory? {
        return when {
            ArmorClawEventType.isWorkflowEvent(eventType) -> EventCategory.WORKFLOW
            ArmorClawEventType.isAgentEvent(eventType) -> EventCategory.AGENT
            eventType.startsWith("com.armorclaw.budget.") -> EventCategory.BUDGET
            eventType.startsWith("com.armorclaw.bridge.") -> EventCategory.BRIDGE
            eventType.startsWith("com.armorclaw.platform.") -> EventCategory.PLATFORM
            else -> null
        }
    }
}

enum class EventCategory {
    WORKFLOW,
    AGENT,
    BUDGET,
    BRIDGE,
    PLATFORM
}
