package com.armorclaw.shared.platform.bridge

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject

/**
 * JSON-RPC 2.0 Request
 */
@Serializable
data class JsonRpcRequest(
    val jsonrpc: String = "2.0",
    val method: String,
    val params: Map<String, JsonElement>? = null,
    val id: String? = null
)

/**
 * JSON-RPC 2.0 Response
 */
@Serializable
data class JsonRpcResponse(
    val jsonrpc: String = "2.0",
    val result: JsonElement? = null,
    val error: JsonRpcError? = null,
    val id: String? = null
)

/**
 * JSON-RPC 2.0 Error
 */
@Serializable
data class JsonRpcError(
    val code: Int,
    val message: String,
    val data: JsonObject? = null
) {
    companion object {
        // Standard JSON-RPC error codes
        const val PARSE_ERROR = -32700
        const val INVALID_REQUEST = -32600
        const val METHOD_NOT_FOUND = -32601
        const val INVALID_PARAMS = -32602
        const val INTERNAL_ERROR = -32603

        // ArmorClaw-specific error codes
        const val AUTH_FAILED = -32001
        const val SESSION_EXPIRED = -32002
        const val DEVICE_NOT_VERIFIED = -32003
        const val ROOM_NOT_FOUND = -32004
        const val MESSAGE_SEND_FAILED = -32005
        const val NETWORK_ERROR = -32006
        const val RATE_LIMITED = -32007
    }
}

// ============================================================================
// Core RPC Response Models
// ============================================================================

/**
 * Response from bridge.start RPC
 */
@Serializable
data class BridgeStartResponse(
    @SerialName("session_id")
    val sessionId: String,
    @SerialName("container_id")
    val containerId: String,
    val status: String,
    @SerialName("ice_servers")
    val iceServers: List<RpcIceServer>? = null
)

/**
 * Response from bridge.status RPC
 *
 * FIX for Bug #3: Added userRole field for server-authoritative role assignment.
 * The server determines the user's role and returns it in this response,
 * eliminating race conditions in client-side "first user = admin" logic.
 */
@Serializable
data class BridgeStatusResponse(
    @SerialName("session_id")
    val sessionId: String?,
    @SerialName("container_id")
    val containerId: String?,
    val status: String,
    val uptime: Long? = null,
    @SerialName("message_count")
    val messageCount: Long? = null,
    /**
     * Server-assigned user role for server-authoritative role assignment.
     * This eliminates race conditions in determining admin privileges.
     *
     * Possible values: "OWNER", "ADMIN", "MODERATOR", "NONE" (uppercase strings)
     * If not provided, defaults to AdminLevel.NONE
     */
    @SerialName("user_role")
    val userRole: AdminLevel? = null,
    /**
     * Indicates if this is a new/fresh server installation.
     * The first user to register will become the owner.
     */
    @SerialName("is_new_server")
    val isNewServer: Boolean? = null
)

/**
 * Response from bridge.stop RPC
 */
@Serializable
data class BridgeStopResponse(
    val success: Boolean,
    val message: String? = null
)

/**
 * ICE Server configuration for RPC responses
 * Note: Use IceServer from domain.model.Call for full configuration
 */
@Serializable
data class RpcIceServer(
    val urls: List<String>,
    val username: String? = null,
    val credential: String? = null
)

/**
 * Response from matrix.login RPC
 */
@Serializable
data class MatrixLoginResponse(
    @SerialName("access_token")
    val accessToken: String,
    @SerialName("refresh_token")
    val refreshToken: String? = null,
    @SerialName("device_id")
    val deviceId: String,
    @SerialName("user_id")
    val userId: String,
    @SerialName("expires_in")
    val expiresIn: Long? = null,
    @SerialName("display_name")
    val displayName: String? = null,
    @SerialName("avatar_url")
    val avatarUrl: String? = null
)

/**
 * Response from matrix.sync RPC
 */
@Serializable
data class MatrixSyncResponse(
    @SerialName("next_batch")
    val nextBatch: String,
    val rooms: MatrixRooms? = null,
    val presence: MatrixPresence? = null
)

@Serializable
data class MatrixRooms(
    val join: Map<String, MatrixJoinedRoom>? = null,
    val invite: Map<String, MatrixInvitedRoom>? = null,
    val leave: Map<String, MatrixLeftRoom>? = null
)

@Serializable
data class MatrixJoinedRoom(
    val timeline: MatrixTimeline? = null,
    val state: MatrixState? = null,
    @SerialName("unread_notifications")
    val unreadNotifications: MatrixNotifications? = null
)

@Serializable
data class MatrixTimeline(
    val events: List<MatrixEvent>? = null,
    val limited: Boolean? = null,
    val prevBatch: String? = null
)

@Serializable
data class MatrixState(
    val events: List<MatrixEvent>? = null
)

@Serializable
data class MatrixEvent(
    @SerialName("event_id")
    val eventId: String,
    val type: String,
    val content: JsonObject? = null,
    val sender: String? = null,
    @SerialName("origin_server_ts")
    val originServerTs: Long? = null,
    @SerialName("room_id")
    val roomId: String? = null
)

@Serializable
data class MatrixNotifications(
    @SerialName("highlight_count")
    val highlightCount: Int? = null,
    @SerialName("notification_count")
    val notificationCount: Int? = null
)

@Serializable
data class MatrixInvitedRoom(
    @SerialName("invite_state")
    val inviteState: MatrixState? = null
)

@Serializable
data class MatrixLeftRoom(
    val state: MatrixState? = null
)

@Serializable
data class MatrixPresence(
    val events: List<MatrixEvent>? = null
)

/**
 * Response from matrix.send RPC
 */
@Serializable
data class MatrixSendResponse(
    @SerialName("event_id")
    val eventId: String,
    @SerialName("txn_id")
    val txnId: String? = null
)

/**
 * Response from matrix.create_room RPC
 */
@Serializable
data class MatrixCreateRoomResponse(
    @SerialName("room_id")
    val roomId: String,
    @SerialName("room_alias")
    val roomAlias: String? = null
)

/**
 * Response from matrix.join_room RPC
 */
@Serializable
data class MatrixJoinRoomResponse(
    @SerialName("room_id")
    val roomId: String
)

/**
 * Response from webrtc.offer/answer RPC
 */
@Serializable
data class WebRtcSignalingResponse(
    val sdp: String? = null,
    val type: String? = null,
    @SerialName("ice_candidates")
    val iceCandidates: List<RpcIceCandidate>? = null
)

@Serializable
data class RpcIceCandidate(
    val candidate: String,
    @SerialName("sdp_mid")
    val sdpMid: String?,
    @SerialName("sdp_mline_index")
    val sdpMlineIndex: Int?
)

/**
 * WebRTC Call Session - response from webrtc.start and webrtc.list
 *
 * Matches Bridge API format for webrtc.start, webrtc.ice_candidate, webrtc.end, webrtc.list
 */
@Serializable
data class WebRtcCallSession(
    @SerialName("session_id")
    val sessionId: String,
    @SerialName("room_id")
    val roomId: String,
    @SerialName("call_type")
    val callType: String,  // "audio" or "video"
    @SerialName("started_at")
    val startedAt: Long,
    @SerialName("started_by")
    val startedBy: String,
    val status: String,  // "active", "ended", "pending"
    val participants: List<String>? = null,
    @SerialName("sdp_offer")
    val sdpOffer: String? = null,
    @SerialName("sdp_answer")
    val sdpAnswer: String? = null
)

// ============================================================================
// Recovery RPC Response Models
// ============================================================================

/**
 * Response from recovery.generate_phrase RPC
 */
@Serializable
data class RecoveryPhraseResponse(
    val phrase: String,
    @SerialName("word_count")
    val wordCount: Int = 12,
    val created: Long
)

/**
 * Response from recovery.verify RPC
 */
@Serializable
data class RecoveryVerifyResponse(
    val valid: Boolean,
    @SerialName("recovery_id")
    val recoveryId: String? = null,
    @SerialName("expires_at")
    val expiresAt: Long? = null
)

/**
 * Response from recovery.status RPC
 */
@Serializable
data class RecoveryStatusResponse(
    val status: String,
    @SerialName("started_at")
    val startedAt: Long? = null,
    @SerialName("expires_at")
    val expiresAt: Long? = null,
    @SerialName("read_only")
    val readOnly: Boolean = true
)

/**
 * Response from recovery.complete RPC
 */
@Serializable
data class RecoveryCompleteResponse(
    val success: Boolean,
    @SerialName("new_device_id")
    val newDeviceId: String? = null,
    val message: String? = null
)

/**
 * Response from recovery.is_device_valid RPC
 */
@Serializable
data class DeviceValidResponse(
    val valid: Boolean,
    val reason: String? = null
)

// ============================================================================
// Platform RPC Response Models
// ============================================================================

/**
 * Response from platform.connect RPC
 */
@Serializable
data class PlatformConnectResponse(
    val success: Boolean,
    @SerialName("platform_id")
    val platformId: String? = null,
    @SerialName("auth_url")
    val authUrl: String? = null,
    val message: String? = null
)

/**
 * Response from platform.list RPC
 */
@Serializable
data class PlatformListResponse(
    val platforms: List<ConnectedPlatform>
)

@Serializable
data class ConnectedPlatform(
    val id: String,
    val type: String,
    val name: String,
    val status: String,
    @SerialName("connected_at")
    val connectedAt: Long? = null,
    @SerialName("last_sync")
    val lastSync: Long? = null
)

/**
 * Response from platform.status RPC
 */
@Serializable
data class PlatformStatusResponse(
    val id: String,
    val type: String,
    val status: String,
    @SerialName("error_message")
    val errorMessage: String? = null,
    @SerialName("sync_in_progress")
    val syncInProgress: Boolean = false
)

/**
 * Response from platform.test RPC
 */
@Serializable
data class PlatformTestResponse(
    val success: Boolean,
    val latency: Long? = null,
    val message: String? = null
)

// ============================================================================
// Push Notification RPC Response Models
// ============================================================================

/**
 * Response from push.register RPC
 */
@Serializable
data class PushRegisterResponse(
    val success: Boolean,
    @SerialName("device_id")
    val deviceId: String? = null,
    val message: String? = null
)

// ============================================================================
// License RPC Response Models
// ============================================================================

/**
 * Response from license.status RPC
 */
@Serializable
data class LicenseStatusResponse(
    val valid: Boolean,
    val tier: String,
    @SerialName("expires_at")
    val expiresAt: Long? = null,
    @SerialName("grace_period_remaining")
    val gracePeriodRemaining: Long? = null, // in hours
    @SerialName("instance_id")
    val instanceId: String? = null,
    @SerialName("max_instances")
    val maxInstances: Int? = null,
    val features: List<String>? = null,
    val warning: String? = null
)

/**
 * Response from license.features RPC
 */
@Serializable
data class LicenseFeaturesResponse(
    val tier: String,
    val features: Map<String, FeatureInfo>,
    val compliance: ComplianceMode
)

/**
 * Feature availability info
 */
@Serializable
data class FeatureInfo(
    val available: Boolean,
    val limit: Int? = null,
    val description: String? = null
)

/**
 * Compliance mode levels
 */
@Serializable
enum class ComplianceMode {
    @SerialName("none")
    NONE,

    @SerialName("basic")
    BASIC,

    @SerialName("standard")
    STANDARD,

    @SerialName("full")
    FULL,

    @SerialName("strict")
    STRICT
}

/**
 * Response from license.check_feature RPC
 */
@Serializable
data class FeatureCheckResponse(
    val feature: String,
    val available: Boolean,
    val reason: String? = null,
    val limit: Int? = null,
    val current: Int? = null
)

// ============================================================================
// Compliance RPC Response Models
// ============================================================================

/**
 * Response from compliance.status RPC
 */
@Serializable
data class ComplianceStatusResponse(
    val mode: ComplianceMode,
    @SerialName("phi_scrubbing")
    val phiScrubbing: Boolean,
    @SerialName("audit_logging")
    val auditLogging: Boolean,
    @SerialName("tamper_evidence")
    val tamperEvidence: Boolean,
    val quarantine: Boolean,
    @SerialName("hipaa_enabled")
    val hipaaEnabled: Boolean
)

/**
 * Response from platform.limits RPC
 */
@Serializable
data class PlatformLimitsResponse(
    val platforms: Map<String, PlatformLimit>
)

/**
 * Platform bridging limit
 */
@Serializable
data class PlatformLimit(
    val enabled: Boolean,
    @SerialName("max_channels")
    val maxChannels: Int? = null,
    @SerialName("max_users")
    val maxUsers: Int? = null,
    @SerialName("current_channels")
    val currentChannels: Int? = null,
    @SerialName("current_users")
    val currentUsers: Int? = null
)

// ============================================================================
// Error Management RPC Response Models
// ============================================================================

/**
 * Response from get_errors RPC
 */
@Serializable
data class ErrorsResponse(
    val errors: List<ErrorEntry>,
    val total: Int,
    @SerialName("has_more")
    val hasMore: Boolean
)

/**
 * Error entry from the error system
 */
@Serializable
data class ErrorEntry(
    val id: String,
    val code: String,
    val component: String,
    val message: String,
    val severity: String,
    val timestamp: Long,
    val resolved: Boolean,
    @SerialName("resolved_at")
    val resolvedAt: Long? = null,
    @SerialName("resolution_notes")
    val resolutionNotes: String? = null,
    val context: Map<String, String>? = null
)

// ============================================================================
// Agent RPC Response Models - NEW
// ============================================================================

/**
 * Response from agent.list RPC
 */
@Serializable
data class AgentListResponse(
    val agents: List<AgentInfo>,
    val count: Int
)

/**
 * Information about a running agent
 */
@Serializable
data class AgentInfo(
    @SerialName("agent_id")
    val agentId: String,
    val name: String,
    val type: String, // "assistant", "workflow", "bridge", etc.
    val status: String, // "idle", "busy", "error", "stopped"
    @SerialName("room_id")
    val roomId: String?,
    @SerialName("created_at")
    val createdAt: Long,
    @SerialName("last_activity")
    val lastActivity: Long?,
    val metadata: Map<String, String>? = null
)

/**
 * Response from agent.status RPC (legacy format)
 * @deprecated Use AgentStatusResponse from domain.model for new implementations
 */
@Serializable
@Deprecated("Use AgentStatusResponse from domain.model package")
data class AgentStatusInfo(
    @SerialName("agent_id")
    val agentId: String,
    val name: String,
    val status: String,
    val uptime: Long,
    @SerialName("messages_processed")
    val messagesProcessed: Int,
    @SerialName("tasks_completed")
    val tasksCompleted: Int,
    @SerialName("errors_count")
    val errorsCount: Int,
    @SerialName("current_task")
    val currentTask: String?,
    @SerialName("last_error")
    val lastError: String?,
    val metadata: Map<String, String>? = null
)

// ============================================================================
// Workflow RPC Response Models - NEW
// ============================================================================

/**
 * Response from workflow.templates RPC
 */
@Serializable
data class WorkflowTemplatesResponse(
    val templates: List<WorkflowTemplate>,
    val count: Int
)

/**
 * Workflow template definition
 */
@Serializable
data class WorkflowTemplate(
    @SerialName("template_id")
    val templateId: String,
    val name: String,
    val description: String,
    val category: String, // "productivity", "analysis", "communication", etc.
    val parameters: List<WorkflowParameter>,
    @SerialName("estimated_duration_ms")
    val estimatedDurationMs: Long?,
    val version: String
)

/**
 * Workflow parameter definition
 */
@Serializable
data class WorkflowParameter(
    val name: String,
    val type: String, // "string", "number", "boolean", "array", "object"
    val required: Boolean,
    val default: String?,
    val description: String,
    val options: List<String>? = null // For enum-like parameters
)

/**
 * Response from workflow.start RPC
 */
@Serializable
data class WorkflowStartResponse(
    @SerialName("workflow_id")
    val workflowId: String,
    @SerialName("template_id")
    val templateId: String,
    val status: String, // "started", "queued"
    @SerialName("estimated_duration_ms")
    val estimatedDurationMs: Long?,
    @SerialName("agent_id")
    val agentId: String?
)

/**
 * Response from workflow.status RPC
 */
@Serializable
data class WorkflowStatusResponse(
    @SerialName("workflow_id")
    val workflowId: String,
    @SerialName("template_id")
    val templateId: String,
    val status: String, // "running", "completed", "failed", "waiting", "cancelled"
    @SerialName("current_step")
    val currentStep: Int,
    @SerialName("total_steps")
    val totalSteps: Int,
    @SerialName("step_name")
    val stepName: String?,
    @SerialName("started_at")
    val startedAt: Long,
    @SerialName("completed_at")
    val completedAt: Long?,
    @SerialName("duration_ms")
    val durationMs: Long?,
    val result: Map<String, String>? = null,
    val error: String? = null
)

// ============================================================================
// HITL (Human-in-the-Loop) RPC Response Models - NEW
// ============================================================================

/**
 * Response from hitl.pending RPC
 */
@Serializable
data class HitlPendingResponse(
    val approvals: List<HitlApproval>,
    val count: Int
)

/**
 * HITL approval request
 */
@Serializable
data class HitlApproval(
    @SerialName("gate_id")
    val gateId: String,
    @SerialName("workflow_id")
    val workflowId: String,
    @SerialName("agent_id")
    val agentId: String,
    @SerialName("request_type")
    val requestType: String, // "action", "data_access", "external_send", etc.
    val title: String,
    val description: String,
    @SerialName("requested_at")
    val requestedAt: Long,
    @SerialName("expires_at")
    val expiresAt: Long,
    val status: String, // "pending", "approved", "rejected", "expired"
    val priority: String, // "low", "medium", "high", "critical"
    val context: Map<String, String>? = null
)

// ============================================================================
// Budget RPC Response Models - NEW
// ============================================================================

/**
 * Response from budget.status RPC
 */
@Serializable
data class BudgetStatusResponse(
    val period: String, // "daily", "weekly", "monthly"
    @SerialName("period_start")
    val periodStart: Long,
    @SerialName("period_end")
    val periodEnd: Long,
    @SerialName("current_usage")
    val currentUsage: Long,
    val limit: Long,
    @SerialName("percentage_used")
    val percentageUsed: Int,
    val currency: String, // "USD", "EUR", "credits", etc.
    @SerialName("breakdown")
    val breakdown: BudgetBreakdown?,
    @SerialName("projected_usage")
    val projectedUsage: Long?,
    val alerting: Boolean
)

/**
 * Budget breakdown by category
 */
@Serializable
data class BudgetBreakdown(
    @SerialName("ai_tokens")
    val aiTokens: Long?,
    @SerialName("api_calls")
    val apiCalls: Long?,
    @SerialName("storage_bytes")
    val storageBytes: Long?,
    @SerialName("bandwidth_bytes")
    val bandwidthBytes: Long?,
    val other: Long?
)

// ============================================================================
// Provisioning RPC Response Models
// ============================================================================

/**
 * Response from provisioning.start RPC
 *
 * Returned when the Bridge generates a new provisioning session (typically during first-boot).
 * Contains the QR data that encodes server config + setup token for ArmorChat to scan.
 */
@Serializable
data class ProvisioningStartResponse(
    @SerialName("provisioning_id")
    val provisioningId: String,
    @SerialName("qr_data")
    val qrData: String,
    @SerialName("setup_token")
    val setupToken: String,
    @SerialName("expires_at")
    val expiresAt: Long,
    @SerialName("server_config")
    val serverConfig: ProvisioningServerConfig? = null
)

/**
 * Server configuration embedded in provisioning payload
 */
@Serializable
data class ProvisioningServerConfig(
    @SerialName("matrix_homeserver")
    val matrixHomeserver: String,
    @SerialName("rpc_url")
    val rpcUrl: String,
    @SerialName("ws_url")
    val wsUrl: String? = null,
    @SerialName("push_gateway")
    val pushGateway: String? = null,
    @SerialName("server_name")
    val serverName: String? = null
)

/**
 * Response from provisioning.status RPC
 *
 * Returns current state of the provisioning session.
 */
@Serializable
data class ProvisioningStatusResponse(
    @SerialName("provisioning_id")
    val provisioningId: String,
    val status: String, // "pending", "claimed", "expired", "cancelled"
    @SerialName("claimed_by")
    val claimedBy: String? = null,
    @SerialName("claimed_at")
    val claimedAt: Long? = null,
    @SerialName("expires_at")
    val expiresAt: Long? = null
)

/**
 * Response from provisioning.claim RPC
 *
 * Returned when ArmorChat claims admin during first-boot setup.
 * On success, the claiming device becomes the OWNER of the ArmorClaw instance.
 */
@Serializable
data class ProvisioningClaimResponse(
    val success: Boolean,
    @SerialName("admin_token")
    val adminToken: String? = null,
    @SerialName("user_id")
    val userId: String? = null,
    val role: AdminLevel? = null,
    @SerialName("device_id")
    val deviceId: String? = null,
    val message: String? = null,
    // RC-03: Bridge returns these but ArmorChat was silently dropping them
    @SerialName("matrix_homeserver")
    val matrixHomeserver: String? = null,
    @SerialName("correlation_id")
    val correlationId: String? = null
)

/**
 * Response from provisioning.rotate RPC
 *
 * Returned when the provisioning secret is rotated (invalidates old QR codes).
 */
@Serializable
data class ProvisioningRotateResponse(
    val success: Boolean,
    @SerialName("new_setup_token")
    val newSetupToken: String? = null,
    @SerialName("new_qr_data")
    val newQrData: String? = null,
    @SerialName("expires_at")
    val expiresAt: Long? = null,
    val message: String? = null
)

/**
 * Response from provisioning.cancel RPC
 */
@Serializable
data class ProvisioningCancelResponse(
    val success: Boolean,
    val message: String? = null
)

// ============================================================================
// Bridge Configuration
// ============================================================================

/**
 * Configuration for the Bridge RPC Client
 *
 * Updated to support direct Matrix access since Bridge WebSocket is a stub.
 * Use useDirectMatrixSync=true to sync via Matrix /sync endpoint directly.
 *
 * ## Configuration Sources (Priority Order)
 * 1. Runtime config (set via setRuntimeConfig())
 * 2. Environment variables (BRIDGE_URL, MATRIX_HOMESERVER)
 * 3. BuildConfig defaults (configured at build time)
 * 4. Hardcoded fallbacks
 */
data class BridgeConfig(
    val baseUrl: String,
    val homeserverUrl: String,           // Matrix homeserver for direct API access
    val wsUrl: String? = null,           // WebSocket URL (null if not available)
    val timeoutMs: Long = 30000,
    val enableCertificatePinning: Boolean = true,
    val certificatePins: List<String> = emptyList(),
    val retryCount: Int = 3,
    val retryDelayMs: Long = 1000,
    val useDirectMatrixSync: Boolean = true,  // Use Matrix /sync directly instead of Bridge
    val environment: Environment = Environment.PRODUCTION,
    val serverName: String? = null
) {
    /**
     * Environment type for configuration
     */
    enum class Environment {
        DEVELOPMENT,    // Local development
        STAGING,        // Test server
        PRODUCTION,     // Live server
        CUSTOM          // User-configured (e.g., self-hosted VPS)
    }

    /**
     * Check if this is a debug/development configuration
     */
    val isDebug: Boolean
        get() = environment == Environment.DEVELOPMENT ||
                baseUrl.contains("localhost") ||
                baseUrl.contains("10.0.2.2") ||
                baseUrl.contains("192.168.")

    /**
     * Check if using HTTPS
     */
    val isSecure: Boolean
        get() = baseUrl.startsWith("https://") && homeserverUrl.startsWith("https://")

    /**
     * Get display name for this configuration
     */
    val displayName: String
        get() = serverName ?: when (environment) {
            Environment.DEVELOPMENT -> "Development Server"
            Environment.STAGING -> "Staging Server"
            Environment.PRODUCTION -> "ArmorClaw"
            Environment.CUSTOM -> "Custom Server"
        }

    companion object {
        // Runtime configuration (can be set at app startup)
        private var _runtimeConfig: BridgeConfig? = null

        /**
         * Set runtime configuration (e.g., from user input or discovery)
         * This takes priority over all other configuration sources.
         */
        fun setRuntimeConfig(config: BridgeConfig) {
            _runtimeConfig = config
        }

        /**
         * Get the current active configuration
         * Priority: runtime > environment-specific > default
         */
        fun getCurrent(): BridgeConfig = _runtimeConfig ?: PRODUCTION

        /**
         * Clear runtime configuration (e.g., on logout)
         */
        fun clearRuntimeConfig() {
            _runtimeConfig = null
        }

        /**
         * Default configuration for production (ArmorClaw hosted)
         */
        val PRODUCTION = BridgeConfig(
            baseUrl = "https://bridge.armorclaw.app",
            homeserverUrl = "https://matrix.armorclaw.app",
            wsUrl = null,
            enableCertificatePinning = true,
            useDirectMatrixSync = true,
            environment = Environment.PRODUCTION,
            serverName = "ArmorClaw"
        )

        /**
         * Default configuration for development (Android emulator)
         */
        val DEVELOPMENT = BridgeConfig(
            baseUrl = "http://10.0.2.2:8080",
            homeserverUrl = "http://10.0.2.2:8008",
            wsUrl = null,
            enableCertificatePinning = false,
            retryCount = 1,
            useDirectMatrixSync = true,
            environment = Environment.DEVELOPMENT,
            serverName = "Development Server"
        )

        /**
         * Staging configuration for testing
         */
        val STAGING = BridgeConfig(
            baseUrl = "https://bridge-staging.armorclaw.app",
            homeserverUrl = "https://matrix-staging.armorclaw.app",
            wsUrl = null,
            enableCertificatePinning = false,
            useDirectMatrixSync = true,
            environment = Environment.STAGING,
            serverName = "Staging Server"
        )

        /**
         * Demo server configuration (read-only, public demo)
         */
        val DEMO = BridgeConfig(
            baseUrl = "https://bridge-demo.armorclaw.app",
            homeserverUrl = "https://matrix-demo.armorclaw.app",
            wsUrl = null,
            enableCertificatePinning = false,
            useDirectMatrixSync = true,
            environment = Environment.STAGING,
            serverName = "Demo Server"
        )

        /**
         * Create a custom configuration for self-hosted servers
         *
         * @param bridgeUrl The Bridge RPC server URL (e.g., "https://bridge.example.com" or "http://192.168.1.100:8080")
         * @param homeserverUrl The Matrix homeserver URL (e.g., "https://matrix.example.com" or "http://192.168.1.100:8008")
         * @param serverName Optional display name for the server
         * @return Configured BridgeConfig instance
         */
        fun createCustom(
            bridgeUrl: String,
            homeserverUrl: String,
            serverName: String? = null
        ): BridgeConfig {
            val isHttps = bridgeUrl.startsWith("https://")
            val isLocal = bridgeUrl.contains("localhost") ||
                          bridgeUrl.contains("192.168.") ||
                          bridgeUrl.contains("10.") ||
                          bridgeUrl.contains("127.0.0.1")

            return BridgeConfig(
                baseUrl = bridgeUrl,
                homeserverUrl = homeserverUrl,
                wsUrl = null,
                enableCertificatePinning = isHttps && !isLocal,
                useDirectMatrixSync = true,
                environment = Environment.CUSTOM,
                serverName = serverName
            )
        }

        /**
         * Create configuration from discovered server info
         */
        fun fromDiscoveredServer(
            homeserver: String,
            bridgeUrl: String? = null,
            serverName: String? = null
        ): BridgeConfig {
            val resolvedBridgeUrl = bridgeUrl ?: deriveBridgeUrl(homeserver)
            return createCustom(
                bridgeUrl = resolvedBridgeUrl,
                homeserverUrl = homeserver,
                serverName = serverName
            )
        }

        /**
         * Derive Bridge URL from Matrix homeserver URL
         * Uses smart derivation with common patterns
         * 
         * For IP-only servers (e.g., http://192.168.1.100:8008):
         * - Uses same IP with port 8080: http://192.168.1.100:8080
         * 
         * For domain servers (e.g., https://matrix.example.com):
         * - Uses bridge subdomain: https://bridge.example.com
         */
        fun deriveBridgeUrl(homeserver: String): String {
            val url = homeserver.removeSuffix("/").removeSuffix("/_matrix")
            val protocol = if (url.startsWith("https://")) "https://" else "http://"

            // Extract host part
            val hostPart = url
                .removePrefix("https://")
                .removePrefix("http://")
                .split("/").first()

            // Extract host and port
            val (host, port) = if (hostPart.contains(":")) {
                val parts = hostPart.split(":")
                parts[0] to parts[1].toIntOrNull()
            } else {
                hostPart to null
            }

            // Check if it's an IP address
            val isIp = isIpAddress(host)

            return when {
                // IP-only server: use port 8080 for bridge
                isIp -> "${protocol}${host}:8080"
                // Pattern: matrix.example.com -> bridge.example.com
                url.contains("://matrix.") -> url.replace("://matrix.", "://bridge.")
                // Pattern: chat.example.com -> bridge.example.com
                url.contains("://chat.") -> url.replace("://chat.", "://bridge.")
                // Pattern: synapse.example.com -> bridge.example.com
                url.contains("://synapse.") -> url.replace("://synapse.", "://bridge.")
                // Pattern: example.com:8008 -> example.com:8080 (common port pattern)
                url.contains(":8008") -> url.replace(":8008", ":8080")
                // Pattern: example.com/_matrix -> example.com (strip Matrix path)
                url.contains("/_matrix") -> url.removeSuffix("/_matrix")
                // Fallback: prepend "bridge." to domain
                else -> {
                    val domain = url.removePrefix("https://").removePrefix("http://")
                    "${protocol}bridge.$domain"
                }
            }
        }

        /**
         * Check if the given string is an IP address (IPv4)
         * Delegates to shared NetworkUtils to avoid code duplication (RC-05).
         */
        private fun isIpAddress(host: String): Boolean = com.armorclaw.shared.platform.network.NetworkUtils.isIpAddress(host)
    }
}

/**
 * Result wrapper for RPC calls
 */
sealed class RpcResult<out T> {
    data class Success<T>(val data: T) : RpcResult<T>()
    data class Error(val code: Int, val message: String, val data: Map<String, Any?>? = null) : RpcResult<Nothing>()

    val isSuccess: Boolean get() = this is Success
    val isError: Boolean get() = this is Error

    fun getOrNull(): T? = when (this) {
        is Success -> data
        is Error -> null
    }

    fun getOrThrow(): T = when (this) {
        is Success -> data
        is Error -> throw RpcException(code, message, data)
    }

    inline fun <R> map(transform: (T) -> R): RpcResult<R> = when (this) {
        is Success -> Success(transform(data))
        is Error -> this
    }

    companion object {
        fun <T> success(data: T): RpcResult<T> = Success(data)
        fun error(code: Int, message: String, data: Map<String, Any?>? = null): RpcResult<Nothing> = Error(code, message, data)
    }
}

/**
 * Result wrapper that is compatible with Kotlin's Result type
 */
sealed class Result<out T> {
    data class Success<T>(val value: T) : Result<T>()
    data class Failure(val exception: Throwable) : Result<Nothing>()

    val isSuccess: Boolean get() = this is Success
    val isFailure: Boolean get() = this is Failure

    fun getOrNull(): T? = when (this) {
        is Success -> value
        is Failure -> null
    }

    fun getOrThrow(): T = when (this) {
        is Success -> value
        is Failure -> throw exception
    }

    inline fun <R> map(transform: (T) -> R): Result<R> = when (this) {
        is Success -> Success(transform(value))
        is Failure -> this
    }

    companion object {
        fun <T> success(value: T): Result<T> = Success(value)
        fun failure(exception: Throwable): Result<Nothing> = Failure(exception)
    }
}

/**
 * Exception thrown for RPC errors
 */
class RpcException(
    val code: Int,
    override val message: String,
    val data: Map<String, Any?>? = null
) : Exception("RPC Error $code: $message")
