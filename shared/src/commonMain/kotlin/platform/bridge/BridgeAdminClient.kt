package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.domain.model.AgentStatusResponse
import com.armorclaw.shared.domain.model.AgentStatusHistoryResponse
import com.armorclaw.shared.domain.model.UnsealChallenge
import com.armorclaw.shared.domain.model.UnsealRequest
import com.armorclaw.shared.domain.model.UnsealResult
import com.armorclaw.shared.domain.model.SessionExtensionResult
import com.armorclaw.shared.domain.model.KeystoreStatusResponse
import kotlinx.coroutines.flow.Flow

/**
 * Admin-only RPC Client for ArmorClaw Bridge
 *
 * This interface contains ONLY the admin functions that should NOT
 * be replaced with Matrix SDK calls. All messaging, room, and sync
 * operations should use MatrixClient instead.
 *
 * ## Migration Strategy
 * ```
 * BEFORE:
 * BridgeRpcClient (everything) → Bridge
 *
 * AFTER:
 * MatrixClient (messaging) → Matrix Homeserver
 * BridgeAdminClient (admin) → Bridge RPC
 * ```
 *
 * ## Categories
 * - Bridge Lifecycle: start/stop/status
 * - Recovery: account recovery flow
 * - Platform: external platform connections
 * - Push: FCM/APNs notifications
 * - WebRTC: voice/video signaling
 * - Agent Status: real-time agent tracking
 * - Keystore: zero-trust credential management
 */
interface BridgeAdminClient {

    // ========================================================================
    // Bridge Lifecycle (4 methods)
    // ========================================================================

    suspend fun startBridge(
        userId: String,
        deviceId: String,
        context: OperationContext? = null
    ): RpcResult<BridgeStartResponse>

    suspend fun getBridgeStatus(
        context: OperationContext? = null
    ): RpcResult<BridgeStatusResponse>

    suspend fun stopBridge(
        sessionId: String,
        context: OperationContext? = null
    ): RpcResult<BridgeStopResponse>

    suspend fun healthCheck(
        context: OperationContext? = null
    ): RpcResult<Map<String, Any?>>

    // ========================================================================
    // Recovery Methods (6 methods)
    // ========================================================================

    suspend fun recoveryGeneratePhrase(
        context: OperationContext? = null
    ): RpcResult<RecoveryPhraseResponse>

    suspend fun recoveryStorePhrase(
        phrase: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    suspend fun recoveryVerify(
        phrase: String,
        context: OperationContext? = null
    ): RpcResult<RecoveryVerifyResponse>

    suspend fun recoveryStatus(
        recoveryId: String,
        context: OperationContext? = null
    ): RpcResult<RecoveryStatusResponse>

    suspend fun recoveryComplete(
        recoveryId: String,
        newDeviceName: String,
        context: OperationContext? = null
    ): RpcResult<RecoveryCompleteResponse>

    suspend fun recoveryIsDeviceValid(
        deviceId: String,
        context: OperationContext? = null
    ): RpcResult<DeviceValidResponse>

    // ========================================================================
    // Platform Methods (5 methods)
    // ========================================================================

    suspend fun platformConnect(
        platformType: String,
        config: Map<String, Any?>,
        context: OperationContext? = null
    ): RpcResult<PlatformConnectResponse>

    suspend fun platformDisconnect(
        platformId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    suspend fun platformList(
        context: OperationContext? = null
    ): RpcResult<PlatformListResponse>

    suspend fun platformStatus(
        platformId: String,
        context: OperationContext? = null
    ): RpcResult<PlatformStatusResponse>

    suspend fun platformTest(
        platformId: String,
        context: OperationContext? = null
    ): RpcResult<PlatformTestResponse>

    // ========================================================================
    // Push Notification Methods (3 methods)
    // ========================================================================

    suspend fun pushRegister(
        pushToken: String,
        pushPlatform: String,
        deviceId: String,
        context: OperationContext? = null
    ): RpcResult<PushRegisterResponse>

    suspend fun pushUnregister(
        pushToken: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    suspend fun pushUpdateSettings(
        enabled: Boolean,
        quietHoursStart: String? = null,
        quietHoursEnd: String? = null,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // WebRTC Methods (4 methods)
    // ========================================================================

    suspend fun webrtcOffer(
        callId: String,
        sdpOffer: String,
        context: OperationContext? = null
    ): RpcResult<WebRtcSignalingResponse>

    suspend fun webrtcAnswer(
        callId: String,
        sdpAnswer: String,
        context: OperationContext? = null
    ): RpcResult<WebRtcSignalingResponse>

    suspend fun webrtcIceCandidate(
        callId: String,
        candidate: String,
        sdpMid: String?,
        sdpMlineIndex: Int?,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    suspend fun webrtcHangup(
        callId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // Agent Status Methods (4 methods + Flow)
    // ========================================================================

    /**
     * Get current status of an agent
     */
    suspend fun getAgentStatus(
        agentId: String,
        context: OperationContext? = null
    ): RpcResult<AgentStatusResponse>

    /**
     * Get status history for an agent
     */
    suspend fun getAgentStatusHistory(
        agentId: String,
        limit: Int = 50,
        context: OperationContext? = null
    ): RpcResult<AgentStatusHistoryResponse>

    /**
     * Subscribe to real-time status updates for an agent
     * Returns a Flow that emits status events via WebSocket
     */
    fun subscribeToAgentStatus(agentId: String): Flow<AgentStatusResponse>

    /**
     * Subscribe to all agent status changes
     * Returns a Flow that emits status events for all agents
     */
    fun subscribeToAllAgentStatuses(): Flow<AgentStatusResponse>

    // ========================================================================
    // Keystore / Zero-Trust Methods (5 methods + Flow)
    // ========================================================================

    /**
     * Get current keystore status
     */
    suspend fun getKeystoreStatus(
        context: OperationContext? = null
    ): RpcResult<KeystoreStatusResponse>

    /**
     * Generate challenge for unsealing
     */
    suspend fun generateUnsealChallenge(
        context: OperationContext? = null
    ): RpcResult<UnsealChallenge>

    /**
     * Respond to unseal challenge with wrapped key
     */
    suspend fun respondToUnseal(
        request: UnsealRequest,
        context: OperationContext? = null
    ): RpcResult<UnsealResult>

    /**
     * Extend the current unseal session
     */
    suspend fun extendSession(
        context: OperationContext? = null
    ): RpcResult<SessionExtensionResult>

    /**
     * Subscribe to keystore state changes
     */
    fun subscribeToKeystoreState(): Flow<KeystoreStatusResponse>
}

// Note: Response data classes (BridgeStartResponse, etc.) are defined in RpcModels.kt
// to avoid duplicate declarations. This interface uses those shared types.
