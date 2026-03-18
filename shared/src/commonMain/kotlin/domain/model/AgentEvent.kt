package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

/**
 * Agent Event
 *
 * Represents an event in the agent workflow lifecycle.
 * Events are stored in the vault for audit trails and can be replayed for debugging.
 *
 * ## Event Types
 * - WORKFLOW_INITIATED: Workflow started
 * - WORKFLOW_STARTED: Workflow began execution
 * - WORKFLOW_STEP_COMPLETED: A step in the workflow finished
 * - WORKFLOW_COMPLETED: Workflow finished successfully
 * - WORKFLOW_FAILED: Workflow encountered an error
 * - WORKFLOW_CANCELLED: Workflow was cancelled
 * - PII_REQUESTED: Agent requested PII access
 * - PII_APPROVED: User approved PII access
 * - PII_DENIED: User denied PII access
 * - BIOMETRIC_REQUESTED: Agent requested biometric authentication
 * - BIOMETRIC_APPROVED: Biometric authentication succeeded
 * - BIOMETRIC_DENIED: Biometric authentication failed
 * - AGENT_THINKING: Agent is processing a task
 * - AGENT_ERROR: Agent encountered an error
 *
 * ## Usage
 * ```kotlin
 * val event = AgentEvent(
 *     eventId = "evt_123",
 *     workflowId = "workflow_456",
 *     agentId = "agent_browse_001",
 *     type = AgentEventType.WORKFLOW_STARTED,
 *     data = jsonObjectOf("step" to "Navigate to checkout"),
 *     timestamp = System.currentTimeMillis()
 * )
 * ```
 */
@Serializable
data class AgentEvent(
    val eventId: String,
    val workflowId: String,
    val agentId: String,
    val type: AgentEventType,
    val data: JsonObject,
    val timestamp: Long
) {
    /**
     * Check if this event is a workflow lifecycle event
     */
    fun isWorkflowEvent(): Boolean {
        return when (type) {
            AgentEventType.WORKFLOW_INITIATED,
            AgentEventType.WORKFLOW_STARTED,
            AgentEventType.WORKFLOW_STEP_COMPLETED,
            AgentEventType.WORKFLOW_COMPLETED,
            AgentEventType.WORKFLOW_FAILED,
            AgentEventType.WORKFLOW_CANCELLED -> true
            else -> false
        }
    }

    /**
     * Check if this event is a PII-related event
     */
    fun isPiiEvent(): Boolean {
        return when (type) {
            AgentEventType.PII_REQUESTED,
            AgentEventType.PII_APPROVED,
            AgentEventType.PII_DENIED -> true
            else -> false
        }
    }

    /**
     * Check if this event is a biometric-related event
     */
    fun isBiometricEvent(): Boolean {
        return when (type) {
            AgentEventType.BIOMETRIC_REQUESTED,
            AgentEventType.BIOMETRIC_APPROVED,
            AgentEventType.BIOMETRIC_DENIED -> true
            else -> false
        }
    }
}

/**
 * Agent event types
 */
@Serializable
enum class AgentEventType {
    /** Workflow lifecycle events */
    WORKFLOW_INITIATED,
    WORKFLOW_STARTED,
    WORKFLOW_STEP_COMPLETED,
    WORKFLOW_COMPLETED,
    WORKFLOW_FAILED,
    WORKFLOW_CANCELLED,

    /** PII access events */
    PII_REQUESTED,
    PII_APPROVED,
    PII_DENIED,

    /** Biometric authentication events */
    BIOMETRIC_REQUESTED,
    BIOMETRIC_APPROVED,
    BIOMETRIC_DENIED,

    /** Agent activity events */
    AGENT_THINKING,
    AGENT_ERROR;

    /**
     * Returns a user-friendly display string
     */
    fun toDisplayString(): String {
        return name.lowercase().replace("_", " ")
    }
}

/**
 * Agent event filter for querying events
 */
@Serializable
data class AgentEventFilter(
    val workflowId: String? = null,
    val agentId: String? = null,
    val type: AgentEventType? = null,
    val fromTimestamp: Long? = null,
    val toTimestamp: Long? = null,
    val limit: Int = 100
) {
    /**
     * Check if an event matches this filter
     */
    fun matches(event: AgentEvent): Boolean {
        if (workflowId != null && event.workflowId != workflowId) return false
        if (agentId != null && event.agentId != agentId) return false
        if (type != null && event.type != type) return false
        if (fromTimestamp != null && event.timestamp < fromTimestamp) return false
        if (toTimestamp != null && event.timestamp > toTimestamp) return false
        return true
    }
}

/**
 * OMO Identity Data
 *
 * Represents user identity information stored for OMO agents
 */
@Serializable
data class OMOIdentityData(
    val id: String,
    val name: String,
    val email: String,
    val phone: String
)

