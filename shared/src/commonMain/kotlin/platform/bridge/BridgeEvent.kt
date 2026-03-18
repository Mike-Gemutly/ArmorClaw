package com.armorclaw.shared.platform.bridge

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/**
 * Bridge WebSocket Event Types
 *
 * Events pushed from the bridge server to clients in real-time.
 *
 * Event Categories:
 * - Messaging: message.received, message.status
 * - Rooms: room.created, room.membership
 * - Real-time: typing, receipt.read, presence
 * - Calls: call
 * - Platforms: platform.message
 * - Session: session.expired, bridge.status
 * - Recovery: recovery
 * - System: app.armorclaw.alert
 * - License/Compliance: license, budget, compliance
 * - Agents: agent.started, agent.stopped, agent.status
 * - HITL: hitl.pending, hitl.resolved
 * - Workflows: workflow.started, workflow.progress, workflow.completed
 */
sealed class BridgeEvent {
    abstract val type: String
    abstract val timestamp: Long
    abstract val sessionId: String?

    /**
     * Message received event
     */
    @Serializable
    data class MessageReceived(
        @SerialName("event_id")
        val eventId: String,
        @SerialName("room_id")
        val roomId: String,
        @SerialName("sender_id")
        val senderId: String,
        val content: BridgeEventContent,
        @SerialName("origin_server_ts")
        val originServerTs: Long,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "message.received"
    ) : BridgeEvent()

    /**
     * Message status updated event
     */
    @Serializable
    data class MessageStatusUpdated(
        @SerialName("event_id")
        val eventId: String,
        @SerialName("room_id")
        val roomId: String,
        val status: String,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "message.status"
    ) : BridgeEvent()

    /**
     * Room created event
     */
    @Serializable
    data class RoomCreated(
        @SerialName("room_id")
        val roomId: String,
        val name: String?,
        @SerialName("is_direct")
        val isDirect: Boolean,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "room.created"
    ) : BridgeEvent()

    /**
     * Room membership changed event
     */
    @Serializable
    data class RoomMembershipChanged(
        @SerialName("room_id")
        val roomId: String,
        @SerialName("user_id")
        val userId: String,
        val membership: String,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "room.membership"
    ) : BridgeEvent()

    /**
     * Typing notification event
     */
    @Serializable
    data class TypingNotification(
        @SerialName("room_id")
        val roomId: String,
        @SerialName("user_id")
        val userId: String,
        val typing: Boolean,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "typing"
    ) : BridgeEvent()

    /**
     * Read receipt event
     */
    @Serializable
    data class ReadReceipt(
        @SerialName("room_id")
        val roomId: String,
        @SerialName("user_id")
        val userId: String,
        @SerialName("event_id")
        val eventId: String,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "receipt.read"
    ) : BridgeEvent()

    /**
     * Presence update event
     */
    @Serializable
    data class PresenceUpdate(
        @SerialName("user_id")
        val userId: String,
        val presence: String,
        @SerialName("status_msg")
        val statusMsg: String?,
        @SerialName("last_active_ts")
        val lastActiveTs: Long?,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "presence"
    ) : BridgeEvent()

    /**
     * WebRTC call event
     */
    @Serializable
    data class CallEvent(
        @SerialName("call_id")
        val callId: String,
        @SerialName("room_id")
        val roomId: String,
        val action: String, // "offer", "answer", "ice_candidate", "hangup"
        val sdp: String? = null,
        @SerialName("ice_candidate")
        val iceCandidate: String? = null,
        @SerialName("sdp_mid")
        val sdpMid: String? = null,
        @SerialName("sdp_mline_index")
        val sdpMlineIndex: Int? = null,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "call"
    ) : BridgeEvent()

    /**
     * Platform message event (from external platforms like Slack, Discord)
     */
    @Serializable
    data class PlatformMessage(
        @SerialName("platform_id")
        val platformId: String,
        @SerialName("platform_type")
        val platformType: String,
        @SerialName("external_channel_id")
        val externalChannelId: String,
        @SerialName("mapped_room_id")
        val mappedRoomId: String,
        @SerialName("sender_name")
        val senderName: String,
        val content: String,
        @SerialName("external_message_id")
        val externalMessageId: String,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "platform.message"
    ) : BridgeEvent()

    /**
     * Session expired event
     */
    @Serializable
    data class SessionExpired(
        @SerialName("session_id")
        val expiredSessionId: String,
        val reason: String,
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "session.expired"
    ) : BridgeEvent()

    /**
     * Bridge status event
     */
    @Serializable
    data class BridgeStatus(
        val status: String, // "healthy", "degraded", "error"
        val message: String?,
        @SerialName("container_id")
        val containerId: String?,
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "bridge.status"
    ) : BridgeEvent()

    /**
     * Recovery event
     */
    @Serializable
    data class RecoveryEvent(
        @SerialName("recovery_id")
        val recoveryId: String,
        val action: String, // "started", "completed", "expired"
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "recovery"
    ) : BridgeEvent()

    /**
     * System Alert event (app.armorclaw.alert)
     *
     * System alerts are distinct from regular messages and displayed with
     * special UI treatment (colored cards, badges, action buttons).
     *
     * Event Type: app.armorclaw.alert
     */
    @Serializable
    data class SystemAlertEvent(
        @SerialName("alert_type")
        val alertType: SystemAlertType,
        val severity: SystemAlertSeverity,
        val title: String,
        val message: String,
        val action: String? = null,
        @SerialName("action_url")
        val actionUrl: String? = null,
        val metadata: Map<String, JsonElement>? = null,
        @SerialName("room_id")
        val roomId: String? = null,
        @SerialName("event_id")
        val eventId: String? = null,
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "app.armorclaw.alert"
    ) : BridgeEvent()

    /**
     * License event
     */
    @Serializable
    data class LicenseEvent(
        val action: String, // "expiring", "expired", "renewed", "upgraded"
        @SerialName("license_tier")
        val licenseTier: String?,
        @SerialName("expires_at")
        val expiresAt: Long?,
        @SerialName("grace_period_hours")
        val gracePeriodHours: Int?,
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "license"
    ) : BridgeEvent()

    /**
     * Budget event
     */
    @Serializable
    data class BudgetEvent(
        val action: String, // "warning", "exceeded", "reset"
        @SerialName("current_usage")
        val currentUsage: Long,
        @SerialName("limit")
        val limit: Long,
        @SerialName("percentage")
        val percentage: Int,
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "budget"
    ) : BridgeEvent()

    /**
     * Compliance event
     */
    @Serializable
    data class ComplianceEvent(
        val action: String, // "phi_detected", "quarantined", "audit_export"
        @SerialName("phi_type")
        val phiType: String?,
        @SerialName("action_taken")
        val actionTaken: String?,
        @SerialName("user_id")
        val userId: String?,
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "compliance"
    ) : BridgeEvent()

    /**
     * Agent started event
     */
    @Serializable
    data class AgentStartedEvent(
        @SerialName("agent_id")
        val agentId: String,
        val name: String,
        @SerialName("agent_type")
        val agentType: String,
        @SerialName("room_id")
        val roomId: String?,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "agent.started"
    ) : BridgeEvent()

    /**
     * Agent stopped event
     */
    @Serializable
    data class AgentStoppedEvent(
        @SerialName("agent_id")
        val agentId: String,
        val reason: String?,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "agent.stopped"
    ) : BridgeEvent()

    /**
     * Agent status changed event
     */
    @Serializable
    data class AgentStatusEvent(
        @SerialName("agent_id")
        val agentId: String,
        val status: String, // "idle", "busy", "error", "stopped"
        @SerialName("message")
        val message: String?,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "agent.status"
    ) : BridgeEvent()

    /**
     * HITL (Human-in-the-Loop) pending approval event
     */
    @Serializable
    data class HitlPendingEvent(
        @SerialName("gate_id")
        val gateId: String,
        @SerialName("workflow_id")
        val workflowId: String,
        @SerialName("agent_id")
        val agentId: String,
        @SerialName("request_type")
        val requestType: String,
        val description: String,
        @SerialName("requested_at")
        val requestedAt: Long,
        @SerialName("expires_at")
        val expiresAt: Long,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "hitl.pending"
    ) : BridgeEvent()

    /**
     * HITL approval resolved event
     */
    @Serializable
    data class HitlResolvedEvent(
        @SerialName("gate_id")
        val gateId: String,
        val resolution: String, // "approved", "rejected", "expired"
        @SerialName("resolved_by")
        val resolvedBy: String?,
        val notes: String?,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "hitl.resolved"
    ) : BridgeEvent()

    /**
     * Workflow progress event
     */
    @Serializable
    data class WorkflowProgressEvent(
        @SerialName("workflow_id")
        val workflowId: String,
        val step: Int,
        @SerialName("total_steps")
        val totalSteps: Int,
        val status: String, // "running", "completed", "failed", "waiting"
        @SerialName("step_name")
        val stepName: String?,
        val message: String?,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "workflow.progress"
    ) : BridgeEvent()

    /**
     * Workflow started event
     */
    @Serializable
    data class WorkflowStartedEvent(
        @SerialName("workflow_id")
        val workflowId: String,
        @SerialName("workflow_name")
        val workflowName: String,
        @SerialName("agent_id")
        val agentId: String?,
        @SerialName("room_id")
        val roomId: String?,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "workflow.started"
    ) : BridgeEvent()

    /**
     * Workflow completed event
     */
    @Serializable
    data class WorkflowCompletedEvent(
        @SerialName("workflow_id")
        val workflowId: String,
        val success: Boolean,
        @SerialName("result_summary")
        val resultSummary: String?,
        @SerialName("duration_ms")
        val durationMs: Long,
        override val sessionId: String? = null,
        override val timestamp: Long = System.currentTimeMillis(),
        override val type: String = "workflow.completed"
    ) : BridgeEvent()

    /**
     * Unknown event (for forward compatibility)
     */
    @Serializable
    data class UnknownEvent(
        override val type: String,
        val data: Map<String, JsonElement>,
        override val sessionId: String?,
        override val timestamp: Long = System.currentTimeMillis()
    ) : BridgeEvent()
}

/**
 * System Alert Types
 *
 * Categories of alerts sent by the ArmorClaw Bridge
 */
@Serializable
enum class SystemAlertType {
    // Budget alerts
    @SerialName("BUDGET_WARNING")
    BUDGET_WARNING,

    @SerialName("BUDGET_EXCEEDED")
    BUDGET_EXCEEDED,

    // License alerts
    @SerialName("LICENSE_EXPIRING")
    LICENSE_EXPIRING,

    @SerialName("LICENSE_EXPIRED")
    LICENSE_EXPIRED,

    @SerialName("LICENSE_INVALID")
    LICENSE_INVALID,

    // Security alerts
    @SerialName("SECURITY_EVENT")
    SECURITY_EVENT,

    @SerialName("TRUST_DEGRADED")
    TRUST_DEGRADED,

    @SerialName("VERIFICATION_REQUIRED")
    VERIFICATION_REQUIRED,

    // System alerts
    @SerialName("BRIDGE_ERROR")
    BRIDGE_ERROR,

    @SerialName("BRIDGE_RESTARTING")
    BRIDGE_RESTARTING,

    @SerialName("MAINTENANCE")
    MAINTENANCE,

    // Compliance alerts
    @SerialName("COMPLIANCE_VIOLATION")
    COMPLIANCE_VIOLATION,

    @SerialName("AUDIT_EXPORT")
    AUDIT_EXPORT;

    companion object {
        fun fromString(value: String): SystemAlertType? {
            return entries.find { it.name == value }
        }
    }
}

/**
 * System Alert Severity Levels
 */
@Serializable
enum class SystemAlertSeverity {
    @SerialName("INFO")
    INFO,

    @SerialName("WARNING")
    WARNING,

    @SerialName("ERROR")
    ERROR,

    @SerialName("CRITICAL")
    CRITICAL;

    companion object {
        fun fromString(value: String): SystemAlertSeverity? {
            return entries.find { it.name == value }
        }
    }
}

/**
 * Content of a message event
 */
@Serializable
data class BridgeEventContent(
    val type: String,
    val body: String? = null,
    val url: String? = null,
    val info: BridgeEventContentInfo? = null,
    @SerialName("m.relates_to")
    val relatesTo: BridgeEventRelatesTo? = null
)

@Serializable
data class BridgeEventContentInfo(
    val mimetype: String? = null,
    val size: Long? = null,
    val width: Int? = null,
    val height: Int? = null,
    val duration: Long? = null
)

@Serializable
data class BridgeEventRelatesTo(
    @SerialName("event_id")
    val eventId: String? = null,
    @SerialName("rel_type")
    val relType: String? = null
)

/**
 * WebSocket connection state
 */
sealed class WebSocketState {
    object Connecting : WebSocketState()
    object Connected : WebSocketState()
    object Disconnecting : WebSocketState()
    object Disconnected : WebSocketState()
    data class Error(val error: Throwable) : WebSocketState()
}
