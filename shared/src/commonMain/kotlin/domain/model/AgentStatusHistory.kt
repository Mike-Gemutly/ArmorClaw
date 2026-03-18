package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName

/**
 * Agent Status History Entry
 *
 * Represents a historical record of an agent's status change.
 * Used for auditing and debugging agent behavior.
 */
@Serializable
data class AgentStatusHistoryEntry(
    @SerialName("id")
    val id: String,

    @SerialName("agent_id")
    val agentId: String,

    @SerialName("status")
    val status: AgentTaskStatus,

    @SerialName("timestamp")
    val timestamp: Long,

    @SerialName("metadata")
    val metadata: Map<String, String>? = null,

    @SerialName("duration_ms")
    val durationMs: Long? = null
)

/**
 * Agent Status Subscription
 *
 * Represents an active subscription to agent status changes.
 */
@Serializable
data class AgentStatusSubscription(
    @SerialName("agent_id")
    val agentId: String,

    @SerialName("subscribed_at")
    val subscribedAt: Long,

    @SerialName("active")
    val active: Boolean
)

/**
 * Agent Status Response
 *
 * Response from agent.get_status RPC method.
 */
@Serializable
data class AgentStatusResponse(
    @SerialName("agent_id")
    val agentId: String,

    @SerialName("status")
    val status: AgentTaskStatus,

    @SerialName("timestamp")
    val timestamp: Long,

    @SerialName("metadata")
    val metadata: Map<String, String>? = null,

    @SerialName("running_since")
    val runningSince: Long? = null,

    @SerialName("current_task")
    val currentTask: String? = null
)

/**
 * Agent Status History Response
 *
 * Response from agent.status_history RPC method.
 */
@Serializable
data class AgentStatusHistoryResponse(
    @SerialName("agent_id")
    val agentId: String,

    @SerialName("history")
    val history: List<AgentStatusHistoryEntry>,

    @SerialName("total_count")
    val totalCount: Int
)
