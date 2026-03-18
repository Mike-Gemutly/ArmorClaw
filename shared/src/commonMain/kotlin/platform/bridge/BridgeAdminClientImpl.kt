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
import kotlinx.coroutines.flow.flow

/**
 * Implementation of BridgeAdminClient that delegates to BridgeRpcClient
 *
 * This class provides a clean interface for admin-only operations,
 * filtering out the deprecated Matrix-related methods.
 *
 * ## Usage
 * ```kotlin
 * val adminClient = BridgeAdminClientImpl(rpcClient)
 *
 * // Use for admin operations
 * adminClient.healthCheck()
 * adminClient.platformConnect("slack", config)
 *
 * // Use MatrixClient for messaging (not here)
 * matrixClient.sendTextMessage(roomId, text)
 * ```
 */
class BridgeAdminClientImpl(
    private val rpcClient: BridgeRpcClient
) : BridgeAdminClient {

    // ========================================================================
    // Bridge Lifecycle
    // ========================================================================

    override suspend fun startBridge(
        userId: String,
        deviceId: String,
        context: OperationContext?
    ): RpcResult<BridgeStartResponse> {
        return rpcClient.startBridge(userId, deviceId, context)
    }

    override suspend fun getBridgeStatus(context: OperationContext?): RpcResult<BridgeStatusResponse> {
        return rpcClient.getBridgeStatus(context)
    }

    override suspend fun stopBridge(
        sessionId: String,
        context: OperationContext?
    ): RpcResult<BridgeStopResponse> {
        return rpcClient.stopBridge(sessionId, context)
    }

    override suspend fun healthCheck(context: OperationContext?): RpcResult<Map<String, Any?>> {
        return rpcClient.healthCheck(context)
    }

    // ========================================================================
    // Recovery Methods
    // ========================================================================

    override suspend fun recoveryGeneratePhrase(context: OperationContext?): RpcResult<RecoveryPhraseResponse> {
        return rpcClient.recoveryGeneratePhrase(context)
    }

    override suspend fun recoveryStorePhrase(
        phrase: String,
        context: OperationContext?
    ): RpcResult<Boolean> {
        return rpcClient.recoveryStorePhrase(phrase, context)
    }

    override suspend fun recoveryVerify(
        phrase: String,
        context: OperationContext?
    ): RpcResult<RecoveryVerifyResponse> {
        return rpcClient.recoveryVerify(phrase, context)
    }

    override suspend fun recoveryStatus(
        recoveryId: String,
        context: OperationContext?
    ): RpcResult<RecoveryStatusResponse> {
        return rpcClient.recoveryStatus(recoveryId, context)
    }

    override suspend fun recoveryComplete(
        recoveryId: String,
        newDeviceName: String,
        context: OperationContext?
    ): RpcResult<RecoveryCompleteResponse> {
        return rpcClient.recoveryComplete(recoveryId, newDeviceName, context)
    }

    override suspend fun recoveryIsDeviceValid(
        deviceId: String,
        context: OperationContext?
    ): RpcResult<DeviceValidResponse> {
        return rpcClient.recoveryIsDeviceValid(deviceId, context)
    }

    // ========================================================================
    // Platform Methods
    // ========================================================================

    override suspend fun platformConnect(
        platformType: String,
        config: Map<String, Any?>,
        context: OperationContext?
    ): RpcResult<PlatformConnectResponse> {
        return rpcClient.platformConnect(platformType, config, context)
    }

    override suspend fun platformDisconnect(
        platformId: String,
        context: OperationContext?
    ): RpcResult<Boolean> {
        return rpcClient.platformDisconnect(platformId, context)
    }

    override suspend fun platformList(context: OperationContext?): RpcResult<PlatformListResponse> {
        return rpcClient.platformList(context)
    }

    override suspend fun platformStatus(
        platformId: String,
        context: OperationContext?
    ): RpcResult<PlatformStatusResponse> {
        return rpcClient.platformStatus(platformId, context)
    }

    override suspend fun platformTest(
        platformId: String,
        context: OperationContext?
    ): RpcResult<PlatformTestResponse> {
        return rpcClient.platformTest(platformId, context)
    }

    // ========================================================================
    // Push Notification Methods
    // ========================================================================

    override suspend fun pushRegister(
        pushToken: String,
        pushPlatform: String,
        deviceId: String,
        context: OperationContext?
    ): RpcResult<PushRegisterResponse> {
        return rpcClient.pushRegister(pushToken, pushPlatform, deviceId, context)
    }

    override suspend fun pushUnregister(
        pushToken: String,
        context: OperationContext?
    ): RpcResult<Boolean> {
        return rpcClient.pushUnregister(pushToken, context)
    }

    override suspend fun pushUpdateSettings(
        enabled: Boolean,
        quietHoursStart: String?,
        quietHoursEnd: String?,
        context: OperationContext?
    ): RpcResult<Boolean> {
        return rpcClient.pushUpdateSettings(enabled, quietHoursStart, quietHoursEnd, context)
    }

    // ========================================================================
    // WebRTC Methods
    // ========================================================================

    override suspend fun webrtcOffer(
        callId: String,
        sdpOffer: String,
        context: OperationContext?
    ): RpcResult<WebRtcSignalingResponse> {
        return rpcClient.webrtcOffer(callId, sdpOffer, context)
    }

    override suspend fun webrtcAnswer(
        callId: String,
        sdpAnswer: String,
        context: OperationContext?
    ): RpcResult<WebRtcSignalingResponse> {
        return rpcClient.webrtcAnswer(callId, sdpAnswer, context)
    }

    override suspend fun webrtcIceCandidate(
        callId: String,
        candidate: String,
        sdpMid: String?,
        sdpMlineIndex: Int?,
        context: OperationContext?
    ): RpcResult<Boolean> {
        return rpcClient.webrtcIceCandidate(callId, candidate, sdpMid, sdpMlineIndex, context)
    }

    override suspend fun webrtcHangup(
        callId: String,
        context: OperationContext?
    ): RpcResult<Boolean> {
        return rpcClient.webrtcHangup(callId, context)
    }

    // ========================================================================
    // Agent Status Methods
    // ========================================================================

    override suspend fun getAgentStatus(
        agentId: String,
        context: OperationContext?
    ): RpcResult<AgentStatusResponse> {
        return rpcClient.agentGetStatus(agentId, context)
    }

    override suspend fun getAgentStatusHistory(
        agentId: String,
        limit: Int,
        context: OperationContext?
    ): RpcResult<AgentStatusHistoryResponse> {
        return rpcClient.agentStatusHistory(agentId, limit, context)
    }

    override fun subscribeToAgentStatus(agentId: String): Flow<AgentStatusResponse> = flow {
        // WebSocket subscription would be implemented here
        // For now, poll every 5 seconds as fallback
        while (true) {
            val result = rpcClient.agentGetStatus(agentId)
            if (result is RpcResult.Success) {
                emit(result.data)
            }
            kotlinx.coroutines.delay(5000)
        }
    }

    override fun subscribeToAllAgentStatuses(): Flow<AgentStatusResponse> = flow {
        // WebSocket subscription for all agents would be implemented here
        // This is a placeholder - actual implementation would use WebSocket
        throw UnsupportedOperationException("WebSocket subscription not yet implemented")
    }

    // ========================================================================
    // Keystore / Zero-Trust Methods
    // ========================================================================

    override suspend fun getKeystoreStatus(
        context: OperationContext?
    ): RpcResult<KeystoreStatusResponse> {
        return rpcClient.keystoreSealed(context)
    }

    override suspend fun generateUnsealChallenge(
        context: OperationContext?
    ): RpcResult<UnsealChallenge> {
        return rpcClient.keystoreUnsealChallenge(context)
    }

    override suspend fun respondToUnseal(
        request: UnsealRequest,
        context: OperationContext?
    ): RpcResult<UnsealResult> {
        return rpcClient.keystoreUnsealRespond(request, context)
    }

    override suspend fun extendSession(
        context: OperationContext?
    ): RpcResult<SessionExtensionResult> {
        return rpcClient.keystoreExtendSession(context)
    }

    override fun subscribeToKeystoreState(): Flow<KeystoreStatusResponse> = flow {
        // WebSocket subscription would be implemented here
        // For now, poll every 30 seconds as fallback
        while (true) {
            val result = rpcClient.keystoreSealed()
            if (result is RpcResult.Success) {
                emit(result.data)
            }
            kotlinx.coroutines.delay(30000)
        }
    }
}
