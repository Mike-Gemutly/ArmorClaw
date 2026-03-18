package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.domain.model.BrowserCommand
import com.armorclaw.shared.domain.model.BrowserJobPriority
import com.armorclaw.shared.domain.model.BrowserJobStatus
import com.armorclaw.shared.domain.model.BrowserEnqueueResponse
import com.armorclaw.shared.domain.model.BrowserJobResponse
import com.armorclaw.shared.domain.model.BrowserJobListResponse
import com.armorclaw.shared.domain.model.BrowserCancelResponse
import com.armorclaw.shared.domain.model.BrowserRetryResponse
import com.armorclaw.shared.domain.model.BrowserQueueStatsResponse
import com.armorclaw.shared.domain.model.AgentStatusResponse
import com.armorclaw.shared.domain.model.AgentStatusHistoryResponse
import com.armorclaw.shared.domain.model.UnsealChallenge
import com.armorclaw.shared.domain.model.UnsealRequest
import com.armorclaw.shared.domain.model.UnsealResult
import com.armorclaw.shared.domain.model.SessionExtensionResult
import com.armorclaw.shared.domain.model.KeystoreStatusResponse
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.plugins.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.logging.*
import io.ktor.client.plugins.websocket.*
import io.ktor.client.request.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.KSerializer
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*
import kotlinx.serialization.serializer
import kotlin.math.pow
import kotlin.random.Random

/**
 * Implementation of BridgeRpcClient using Ktor HttpClient
 *
 * This client communicates with the ArmorClaw Go bridge server using JSON-RPC 2.0 protocol.
 */
class BridgeRpcClientImpl(
    private val config: BridgeConfig,
    private val httpClient: HttpClient = createDefaultHttpClient(config)
) : BridgeRpcClient {

    private val logger = repositoryLogger("BridgeRpcClient", LogTag.Network.BridgeRpc)

    private var _sessionId: String? = null
    private var _accessToken: String? = null
    private val stateMutex = Mutex()

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
    }

    companion object {
        private fun randomUuid(): String {
            val bytes = Random.nextBytes(16)
            bytes[6] = (bytes[6].toInt() and 0x0F or 0x40).toByte()
            bytes[8] = (bytes[8].toInt() and 0x3F or 0x80).toByte()
            val hex = bytes.joinToString("") { "%02x".format(it) }
            return "${hex.substring(0, 8)}-${hex.substring(8, 12)}-${hex.substring(12, 16)}-${hex.substring(16, 20)}-${hex.substring(20, 32)}"
        }

        fun createDefaultHttpClient(config: BridgeConfig): HttpClient {
            return HttpClient {
                install(ContentNegotiation) {
                    json(Json { ignoreUnknownKeys = true; isLenient = true; encodeDefaults = true })
                }
                install(WebSockets) {
                    pingInterval = 30_000
                    maxFrameSize = Long.MAX_VALUE
                }
                install(Logging) {
                    level = if (config.enableCertificatePinning) LogLevel.INFO else LogLevel.BODY
                    logger = object : Logger { override fun log(message: String) { println("[BridgeRpcClient] $message") } }
                }
                install(HttpTimeout) {
                    requestTimeoutMillis = config.timeoutMs
                    connectTimeoutMillis = config.timeoutMs / 2
                    socketTimeoutMillis = config.timeoutMs
                }
                defaultRequest {
                    url(config.baseUrl)
                    contentType(ContentType.Application.Json)
                }
            }
        }
    }

    override fun isConnected(): Boolean = _sessionId != null
    override fun getSessionId(): String? = _sessionId

    suspend fun setAccessToken(token: String?) { stateMutex.withLock { _accessToken = token } }

    override suspend fun setAdminToken(token: String?) { setAccessToken(token) }

    // ========================================================================
    // Bridge Lifecycle Methods
    // ========================================================================

    override suspend fun startBridge(userId: String, deviceId: String, context: OperationContext?): RpcResult<BridgeStartResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "user_id" to JsonPrimitive(userId),
            "device_id" to JsonPrimitive(deviceId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("bridge.start", ctx) { context ->
            callRpcTyped<BridgeStartResponse>("bridge.start", params, context)
        }.also { result ->
            if (result is RpcResult.Success) {
                stateMutex.withLock { _sessionId = result.data.sessionId }
            }
        }
    }

    override suspend fun getBridgeStatus(context: OperationContext?): RpcResult<BridgeStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("bridge.status", ctx) { context ->
            callRpcTyped<BridgeStatusResponse>("bridge.status", params, context)
        }
    }

    override suspend fun stopBridge(sessionId: String, context: OperationContext?): RpcResult<BridgeStopResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "session_id" to JsonPrimitive(sessionId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("bridge.stop", ctx) { context ->
            callRpcTyped<BridgeStopResponse>("bridge.stop", params, context)
        }.also { result ->
            if (result is RpcResult.Success) {
                stateMutex.withLock { _sessionId = null }
            }
        }
    }

    override suspend fun healthCheck(context: OperationContext?): RpcResult<Map<String, Any?>> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("bridge.health", ctx) { context ->
            when (val result = callRpcRaw("bridge.health", params, context)) {
                is RpcResult.Success -> {
                    val map = (result.data as? JsonObject)?.let { obj ->
                        obj.mapValues { extractValue(it.value) }
                    } ?: emptyMap()
                    RpcResult.success(map)
                }
                is RpcResult.Error -> result
            }
        }
    }

    // ========================================================================
    // Matrix Methods
    // ========================================================================

    override suspend fun matrixLogin(homeserver: String, username: String, password: String, deviceId: String, context: OperationContext?): RpcResult<MatrixLoginResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "homeserver" to JsonPrimitive(homeserver),
            "username" to JsonPrimitive(username),
            "password" to JsonPrimitive(password),
            "device_id" to JsonPrimitive(deviceId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.login", ctx) { context ->
            callRpcTyped<MatrixLoginResponse>("matrix.login", params, context)
        }.also { result ->
            if (result is RpcResult.Success) {
                stateMutex.withLock { _accessToken = result.data.accessToken }
            }
        }
    }

    override suspend fun matrixSync(since: String?, timeout: Long, filter: String?, context: OperationContext?): RpcResult<MatrixSyncResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>("correlation_id" to JsonPrimitive(ctx.correlationId))
        since?.let { params["since"] = JsonPrimitive(it) }
        params["timeout"] = JsonPrimitive(timeout)
        filter?.let { params["filter"] = JsonPrimitive(it) }
        return executeWithRetry("matrix.sync", ctx, maxRetries = 1) { context ->
            callRpcTyped<MatrixSyncResponse>("matrix.sync", params, context)
        }
    }

    override suspend fun matrixSend(roomId: String, eventType: String, content: Map<String, Any?>, txnId: String?, context: OperationContext?): RpcResult<MatrixSendResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "room_id" to JsonPrimitive(roomId),
            "event_type" to JsonPrimitive(eventType),
            "content" to content.toJsonElement(),
            "txn_id" to JsonPrimitive(txnId ?: randomUuid()),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.send", ctx) { context ->
            callRpcTyped<MatrixSendResponse>("matrix.send", params, context)
        }
    }

    override suspend fun matrixRefreshToken(refreshToken: String, context: OperationContext?): RpcResult<MatrixLoginResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "refresh_token" to JsonPrimitive(refreshToken),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.refresh_token", ctx) { context ->
            callRpcTyped<MatrixLoginResponse>("matrix.refresh_token", params, context)
        }.also { result ->
            if (result is RpcResult.Success) {
                stateMutex.withLock { _accessToken = result.data.accessToken }
            }
        }
    }

    override suspend fun matrixCreateRoom(
        name: String?,
        topic: String?,
        isDirect: Boolean,
        invite: List<String>?,
        context: OperationContext?
    ): RpcResult<MatrixCreateRoomResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>(
            "is_direct" to JsonPrimitive(isDirect),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        name?.let { params["name"] = JsonPrimitive(it) }
        topic?.let { params["topic"] = JsonPrimitive(it) }
        invite?.let { params["invite"] = JsonArray(it.map { id -> JsonPrimitive(id) }) }
        return executeWithRetry("matrix.create_room", ctx) { context ->
            callRpcTyped<MatrixCreateRoomResponse>("matrix.create_room", params, context)
        }
    }

    override suspend fun matrixJoinRoom(roomIdOrAlias: String, context: OperationContext?): RpcResult<MatrixJoinRoomResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "room_id_or_alias" to JsonPrimitive(roomIdOrAlias),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.join_room", ctx) { context ->
            callRpcTyped<MatrixJoinRoomResponse>("matrix.join_room", params, context)
        }
    }

    override suspend fun matrixLeaveRoom(roomId: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "room_id" to JsonPrimitive(roomId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.leave_room", ctx) { context ->
            callRpcBoolean("matrix.leave_room", params, context)
        }
    }

    override suspend fun matrixInviteUser(roomId: String, userId: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "room_id" to JsonPrimitive(roomId),
            "user_id" to JsonPrimitive(userId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.invite_user", ctx) { context ->
            callRpcBoolean("matrix.invite_user", params, context)
        }
    }

    override suspend fun matrixSendTyping(roomId: String, typing: Boolean, timeout: Long, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "room_id" to JsonPrimitive(roomId),
            "typing" to JsonPrimitive(typing),
            "timeout" to JsonPrimitive(timeout),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.send_typing", ctx) { context ->
            callRpcBoolean("matrix.send_typing", params, context)
        }
    }

    override suspend fun matrixSendReadReceipt(roomId: String, eventId: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "room_id" to JsonPrimitive(roomId),
            "event_id" to JsonPrimitive(eventId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("matrix.send_read_receipt", ctx) { context ->
            callRpcBoolean("matrix.send_read_receipt", params, context)
        }
    }

    // ========================================================================
    // WebRTC Methods
    // ========================================================================

    override suspend fun webrtcOffer(callId: String, sdpOffer: String, context: OperationContext?): RpcResult<WebRtcSignalingResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "call_id" to JsonPrimitive(callId),
            "sdp" to JsonPrimitive(sdpOffer),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("webrtc.offer", ctx) { context ->
            callRpcTyped<WebRtcSignalingResponse>("webrtc.offer", params, context)
        }
    }

    override suspend fun webrtcAnswer(callId: String, sdpAnswer: String, context: OperationContext?): RpcResult<WebRtcSignalingResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "call_id" to JsonPrimitive(callId),
            "sdp" to JsonPrimitive(sdpAnswer),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("webrtc.answer", ctx) { context ->
            callRpcTyped<WebRtcSignalingResponse>("webrtc.answer", params, context)
        }
    }

    override suspend fun webrtcIceCandidate(callId: String, candidate: String, sdpMid: String?, sdpMlineIndex: Int?, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>(
            "call_id" to JsonPrimitive(callId),
            "candidate" to JsonPrimitive(candidate),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        sdpMid?.let { params["sdp_mid"] = JsonPrimitive(it) }
        sdpMlineIndex?.let { params["sdp_mline_index"] = JsonPrimitive(it) }
        return executeWithRetry("webrtc.ice_candidate", ctx) { context ->
            callRpcBoolean("webrtc.ice_candidate", params, context)
        }
    }

    override suspend fun webrtcHangup(callId: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "call_id" to JsonPrimitive(callId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("webrtc.hangup", ctx) { context ->
            callRpcBoolean("webrtc.hangup", params, context)
        }
    }

    // ========================================================================
    // WebRTC Methods - Bridge API Compatible (snake_case)
    // ========================================================================

    override suspend fun webrtcStart(
        roomId: String,
        callType: String,
        context: OperationContext?
    ): RpcResult<WebRtcCallSession> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "room_id" to JsonPrimitive(roomId),
            "call_type" to JsonPrimitive(callType),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("webrtc.start", ctx) { context ->
            callRpcTyped<WebRtcCallSession>("webrtc.start", params, context)
        }
    }

    override suspend fun webrtcSendIceCandidate(
        sessionId: String,
        candidate: String,
        sdpMid: String?,
        sdpMLineIndex: Int?,
        context: OperationContext?
    ): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>(
            "session_id" to JsonPrimitive(sessionId),
            "candidate" to JsonPrimitive(candidate),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        sdpMid?.let { params["sdp_mid"] = JsonPrimitive(it) }
        sdpMLineIndex?.let { params["sdp_mline_index"] = JsonPrimitive(it) }
        return executeWithRetry("webrtc.ice_candidate", ctx) { context ->
            callRpcBoolean("webrtc.ice_candidate", params, context)
        }
    }

    override suspend fun webrtcEnd(sessionId: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "session_id" to JsonPrimitive(sessionId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("webrtc.end", ctx) { context ->
            callRpcBoolean("webrtc.end", params, context)
        }
    }

    override suspend fun webrtcList(context: OperationContext?): RpcResult<List<WebRtcCallSession>> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("webrtc.list", ctx) { context ->
            callRpcTyped<List<WebRtcCallSession>>("webrtc.list", params, context)
        }
    }

    // ========================================================================
    // Recovery Methods
    // ========================================================================

    override suspend fun recoveryGeneratePhrase(context: OperationContext?): RpcResult<RecoveryPhraseResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("recovery.generate_phrase", ctx) { context ->
            callRpcTyped<RecoveryPhraseResponse>("recovery.generate_phrase", params, context)
        }
    }

    override suspend fun recoveryStorePhrase(phrase: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "phrase" to JsonPrimitive(phrase),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("recovery.store_phrase", ctx) { context ->
            callRpcBoolean("recovery.store_phrase", params, context)
        }
    }

    override suspend fun recoveryVerify(phrase: String, context: OperationContext?): RpcResult<RecoveryVerifyResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "phrase" to JsonPrimitive(phrase),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("recovery.verify", ctx) { context ->
            callRpcTyped<RecoveryVerifyResponse>("recovery.verify", params, context)
        }
    }

    override suspend fun recoveryStatus(recoveryId: String, context: OperationContext?): RpcResult<RecoveryStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "recovery_id" to JsonPrimitive(recoveryId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("recovery.status", ctx) { context ->
            callRpcTyped<RecoveryStatusResponse>("recovery.status", params, context)
        }
    }

    override suspend fun recoveryComplete(recoveryId: String, newDeviceName: String, context: OperationContext?): RpcResult<RecoveryCompleteResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "recovery_id" to JsonPrimitive(recoveryId),
            "new_device_name" to JsonPrimitive(newDeviceName),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("recovery.complete", ctx) { context ->
            callRpcTyped<RecoveryCompleteResponse>("recovery.complete", params, context)
        }
    }

    override suspend fun recoveryIsDeviceValid(deviceId: String, context: OperationContext?): RpcResult<DeviceValidResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "device_id" to JsonPrimitive(deviceId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("recovery.is_device_valid", ctx) { context ->
            callRpcTyped<DeviceValidResponse>("recovery.is_device_valid", params, context)
        }
    }

    // ========================================================================
    // Platform Methods
    // ========================================================================

    override suspend fun platformConnect(platformType: String, config: Map<String, Any?>, context: OperationContext?): RpcResult<PlatformConnectResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "platform_type" to JsonPrimitive(platformType),
            "config" to config.toJsonElement(),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("platform.connect", ctx) { context ->
            callRpcTyped<PlatformConnectResponse>("platform.connect", params, context)
        }
    }

    override suspend fun platformDisconnect(platformId: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "platform_id" to JsonPrimitive(platformId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("platform.disconnect", ctx) { context ->
            callRpcBoolean("platform.disconnect", params, context)
        }
    }

    override suspend fun platformList(context: OperationContext?): RpcResult<PlatformListResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("platform.list", ctx) { context ->
            callRpcTyped<PlatformListResponse>("platform.list", params, context)
        }
    }

    override suspend fun platformStatus(platformId: String, context: OperationContext?): RpcResult<PlatformStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "platform_id" to JsonPrimitive(platformId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("platform.status", ctx) { context ->
            callRpcTyped<PlatformStatusResponse>("platform.status", params, context)
        }
    }

    override suspend fun platformTest(platformId: String, context: OperationContext?): RpcResult<PlatformTestResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "platform_id" to JsonPrimitive(platformId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("platform.test", ctx) { context ->
            callRpcTyped<PlatformTestResponse>("platform.test", params, context)
        }
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
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "push_token" to JsonPrimitive(pushToken),
            "push_platform" to JsonPrimitive(pushPlatform),
            "device_id" to JsonPrimitive(deviceId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("push.register_token", ctx) { context ->
            callRpcTyped<PushRegisterResponse>("push.register_token", params, context)
        }
    }

    override suspend fun pushUnregister(pushToken: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "push_token" to JsonPrimitive(pushToken),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("push.unregister_token", ctx) { context ->
            callRpcBoolean("push.unregister_token", params, context)
        }
    }

    override suspend fun pushUpdateSettings(
        enabled: Boolean,
        quietHoursStart: String?,
        quietHoursEnd: String?,
        context: OperationContext?
    ): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>(
            "enabled" to JsonPrimitive(enabled),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        quietHoursStart?.let { params["quiet_hours_start"] = JsonPrimitive(it) }
        quietHoursEnd?.let { params["quiet_hours_end"] = JsonPrimitive(it) }
        return executeWithRetry("push.update_settings", ctx) { context ->
            callRpcBoolean("push.update_settings", params, context)
        }
    }

    // ========================================================================
    // License Methods
    // ========================================================================

    override suspend fun licenseStatus(context: OperationContext?): RpcResult<LicenseStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("license.status", ctx) { context ->
            callRpcTyped<LicenseStatusResponse>("license.status", params, context)
        }
    }

    override suspend fun licenseFeatures(context: OperationContext?): RpcResult<LicenseFeaturesResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("license.features", ctx) { context ->
            callRpcTyped<LicenseFeaturesResponse>("license.features", params, context)
        }
    }

    override suspend fun licenseCheckFeature(feature: String, context: OperationContext?): RpcResult<FeatureCheckResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "feature" to JsonPrimitive(feature),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("license.check_feature", ctx) { context ->
            callRpcTyped<FeatureCheckResponse>("license.check_feature", params, context)
        }
    }

    // ========================================================================
    // Compliance Methods
    // ========================================================================

    override suspend fun complianceStatus(context: OperationContext?): RpcResult<ComplianceStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("compliance.status", ctx) { context ->
            callRpcTyped<ComplianceStatusResponse>("compliance.status", params, context)
        }
    }

    override suspend fun platformLimits(context: OperationContext?): RpcResult<PlatformLimitsResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("platform.limits", ctx) { context ->
            callRpcTyped<PlatformLimitsResponse>("platform.limits", params, context)
        }
    }

    // ========================================================================
    // Error Management Methods
    // ========================================================================

    override suspend fun getErrors(
        limit: Int,
        component: String?,
        context: OperationContext?
    ): RpcResult<ErrorsResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>(
            "limit" to JsonPrimitive(limit),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        component?.let { params["component"] = JsonPrimitive(it) }
        return executeWithRetry("get_errors", ctx) { context ->
            callRpcTyped<ErrorsResponse>("get_errors", params, context)
        }
    }

    override suspend fun resolveError(
        errorId: String,
        resolution: String?,
        context: OperationContext?
    ): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>(
            "error_id" to JsonPrimitive(errorId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        resolution?.let { params["resolution"] = JsonPrimitive(it) }
        return executeWithRetry("resolve_error", ctx) { context ->
            callRpcBoolean("resolve_error", params, context)
        }
    }

    // ========================================================================
    // Provisioning Methods — ArmorChat ↔ ArmorClaw first-boot setup
    // ========================================================================

    override suspend fun provisioningStart(expiration: String, context: OperationContext?): RpcResult<ProvisioningStartResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "expiration" to JsonPrimitive(expiration),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("provisioning.start", ctx) { context ->
            callRpcTyped<ProvisioningStartResponse>("provisioning.start", params, context)
        }
    }

    override suspend fun provisioningStatus(provisioningId: String, context: OperationContext?): RpcResult<ProvisioningStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "provisioning_id" to JsonPrimitive(provisioningId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("provisioning.status", ctx) { context ->
            callRpcTyped<ProvisioningStatusResponse>("provisioning.status", params, context)
        }
    }

    override suspend fun provisioningClaim(
        setupToken: String,
        deviceName: String,
        deviceType: String,
        context: OperationContext?
    ): RpcResult<ProvisioningClaimResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "setup_token" to JsonPrimitive(setupToken),
            "device_name" to JsonPrimitive(deviceName),
            "device_type" to JsonPrimitive(deviceType),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("provisioning.claim", ctx) { context ->
            callRpcTyped<ProvisioningClaimResponse>("provisioning.claim", params, context)
        }
    }

    override suspend fun provisioningRotate(context: OperationContext?): RpcResult<ProvisioningRotateResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("provisioning.rotate", ctx) { context ->
            callRpcTyped<ProvisioningRotateResponse>("provisioning.rotate", params, context)
        }
    }

    override suspend fun provisioningCancel(provisioningId: String, context: OperationContext?): RpcResult<ProvisioningCancelResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "provisioning_id" to JsonPrimitive(provisioningId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("provisioning.cancel", ctx) { context ->
            callRpcTyped<ProvisioningCancelResponse>("provisioning.cancel", params, context)
        }
    }

    // ========================================================================
    // Raw RPC Method
    // ========================================================================

    override suspend fun <T> call(method: String, params: Map<String, Any?>?, context: OperationContext?): RpcResult<T> {
        val jsonParams = params?.mapValues { it.value.toJsonElement() }
        @Suppress("UNCHECKED_CAST")
        return callRpcRaw(method, jsonParams) as RpcResult<T>
    }

    // ========================================================================
    // Private Implementation
    // ========================================================================

    private fun Any?.toJsonElement(): JsonElement = when (this) {
        null -> JsonNull
        is JsonElement -> this
        is Boolean -> JsonPrimitive(this)
        is Number -> JsonPrimitive(this)
        is String -> JsonPrimitive(this)
        is Map<*, *> -> JsonObject(entries.associate { it.key.toString() to it.value.toJsonElement() })
        is List<*> -> JsonArray(map { it.toJsonElement() })
        else -> JsonPrimitive(toString())
    }

    private fun extractValue(element: JsonElement): Any? = when (element) {
        is JsonNull -> null
        is JsonPrimitive -> when {
            element.isString -> element.content
            element.booleanOrNull != null -> element.boolean
            element.longOrNull != null -> element.long
            element.doubleOrNull != null -> element.double
            else -> element.content
        }
        is JsonObject -> element.mapValues { extractValue(it.value) }
        is JsonArray -> element.map { extractValue(it) }
    }

    private suspend fun <T> executeWithRetry(
        method: String,
        context: OperationContext,
        maxRetries: Int = config.retryCount,
        block: suspend (OperationContext) -> RpcResult<T>
    ): RpcResult<T> {
        var lastError: RpcResult.Error? = null

        repeat(maxRetries) { attempt ->
            try {
                logger.logOperationStart(method, mapOf(
                    "attempt" to (attempt + 1),
                    "max_retries" to maxRetries,
                    "correlation_id" to context.correlationId,
                    "trace_id" to (context.traceId ?: "none")
                ))

                val result = block(context)

                if (result is RpcResult.Success) {
                    logger.logOperationSuccess(method)
                    return result
                }

                val errorResult = result as? RpcResult.Error
                if (errorResult != null) {
                    lastError = errorResult
                    when (errorResult.code) {
                        JsonRpcError.AUTH_FAILED, JsonRpcError.SESSION_EXPIRED,
                        JsonRpcError.INVALID_PARAMS, JsonRpcError.METHOD_NOT_FOUND -> {
                            logger.logOperationError(method, Exception(errorResult.message), errorResult.data ?: emptyMap())
                            return result
                        }
                    }
                }

                if (attempt < maxRetries - 1) {
                    kotlinx.coroutines.delay(calculateBackoffDelay(attempt))
                }
            } catch (e: Exception) {
                val errorMetadata = mapOf(
                    "correlation_id" to context.correlationId,
                    "trace_id" to (context.traceId ?: "none"),
                    "attempt" to (attempt + 1)
                )
                logger.logOperationError(method, e, errorMetadata)
                lastError = RpcResult.Error(JsonRpcError.NETWORK_ERROR, "Network error: ${e.message}", errorMetadata)
                if (attempt < maxRetries - 1) {
                    kotlinx.coroutines.delay(calculateBackoffDelay(attempt))
                }
            }
        }

        return lastError ?: RpcResult.error(JsonRpcError.INTERNAL_ERROR, "Unknown error",
            mapOf("correlation_id" to context.correlationId))
    }

    private fun calculateBackoffDelay(attempt: Int): Long {
        val baseDelay = config.retryDelayMs
        val maxDelay = 10000L
        return (baseDelay * 2.0.pow(attempt.toDouble()).toLong()).coerceAtMost(maxDelay)
    }

    private suspend fun callRpcRaw(
        method: String,
        params: Map<String, JsonElement>?,
        context: OperationContext? = null
    ): RpcResult<JsonElement> {
        val ctx = context ?: OperationContext.create()
        val requestId = randomUuid()
        val request = JsonRpcRequest(jsonrpc = "2.0", method = method, params = params, id = requestId)

        // Log request with trace info
        logger.logNetworkRequest("/api", "POST", mapOf(
            "method" to method,
            "request_id" to requestId,
            "correlation_id" to ctx.correlationId,
            "trace_id" to (ctx.traceId ?: "none")
        ))

        return try {
            // NOTE: Bridge HTTP server uses /api endpoint (not /rpc)
            val response = httpClient.post("/api") {
                setBody(json.encodeToString(JsonRpcRequest.serializer(), request))
                _accessToken?.let { header("Authorization", "Bearer $it") }
                // Add distributed tracing headers
                header("X-Request-ID", requestId)
                header("X-Correlation-ID", ctx.correlationId)
                ctx.traceId?.let { header("X-Trace-ID", it) }
            }

            val responseText = response.body<String>()
            val jsonResponse = json.parseToJsonElement(responseText).jsonObject

            // Extract server trace headers from response (if any)
            val serverTraceId = response.headers["X-Trace-ID"]
            val serverRequestId = response.headers["X-Request-ID"]

            jsonResponse["error"]?.jsonObject?.let { errorObj ->
                val code = errorObj["code"]?.jsonPrimitive?.int ?: JsonRpcError.INTERNAL_ERROR
                val message = errorObj["message"]?.jsonPrimitive?.content ?: "Unknown error"
                val errorData = errorObj["data"]?.jsonObject

                // Include server trace info in error metadata
                val errorMetadata = buildMap {
                    put("request_id", requestId)
                    put("correlation_id", ctx.correlationId)
                    ctx.traceId?.let { put("client_trace_id", it) }
                    serverTraceId?.let { put("server_trace_id", it) }
                    serverRequestId?.let { put("server_request_id", it) }
                    errorData?.let { data ->
                        put("server_error_data", data.mapValues { it.value.toString() })
                    }
                }

                logger.logOperationError(method, Exception("$code: $message"), errorMetadata)
                return RpcResult.Error(code, message, errorMetadata)
            }

            val resultElement = jsonResponse["result"]
                ?: return RpcResult.error(JsonRpcError.INTERNAL_ERROR, "No result in response")

            // Log successful response
            logger.logNetworkResponse("/api", 200, 0, mapOf(
                "method" to method,
                "request_id" to requestId,
                "server_trace_id" to (serverTraceId ?: "none")
            ))

            RpcResult.success(resultElement)
        } catch (e: Exception) {
            val errorMetadata = mapOf(
                "request_id" to requestId,
                "correlation_id" to ctx.correlationId,
                "client_trace_id" to (ctx.traceId ?: "none"),
                "error_type" to e::class.simpleName
            )
            logger.logOperationError(method, e, errorMetadata)
            RpcResult.Error(JsonRpcError.NETWORK_ERROR, "Network error: ${e.message}", errorMetadata)
        }
    }

    private suspend fun <T> callRpcTyped(
        method: String,
        params: Map<String, JsonElement>?,
        deserializer: KSerializer<T>,
        context: OperationContext? = null
    ): RpcResult<T> {
        return when (val result = callRpcRaw(method, params, context)) {
            is RpcResult.Success -> {
                try {
                    RpcResult.success(json.decodeFromJsonElement(deserializer, result.data))
                } catch (e: Exception) {
                    RpcResult.error(JsonRpcError.INTERNAL_ERROR, "Failed to parse response: ${e.message}",
                        mapOf("parse_error" to (e.message ?: "unknown"), "method" to method))
                }
            }
            is RpcResult.Error -> result
        }
    }

    private suspend fun callRpcBoolean(
        method: String,
        params: Map<String, JsonElement>?,
        context: OperationContext? = null
    ): RpcResult<Boolean> {
        return when (val result = callRpcRaw(method, params, context)) {
            is RpcResult.Success -> RpcResult.success(result.data.jsonPrimitive.booleanOrNull ?: false)
            is RpcResult.Error -> result
        }
    }

    // Inline reified versions for convenience
    private suspend inline fun <reified T> callRpcTyped(
        method: String,
        params: Map<String, JsonElement>?,
        context: OperationContext? = null
    ): RpcResult<T> {
        return callRpcTyped(method, params, json.serializersModule.serializer<T>(), context)
    }

    // ========================================================================
    // Agent Management Methods - NEW
    // ========================================================================

    override suspend fun agentList(context: OperationContext?): RpcResult<AgentListResponse> {
        val ctx = context ?: OperationContext.create()
        return executeWithRetry("agent.list", ctx) { context ->
            callRpcTyped<AgentListResponse>("agent.list", null, context)
        }
    }

    override suspend fun agentStatus(agentId: String, context: OperationContext?): RpcResult<AgentStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("agent_id" to JsonPrimitive(agentId))
        return executeWithRetry("agent.status", ctx) { context ->
            callRpcTyped<AgentStatusResponse>("agent.status", params, context)
        }
    }

    override suspend fun agentStop(agentId: String, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("agent_id" to JsonPrimitive(agentId))
        return executeWithRetry("agent.stop", ctx) { context ->
            callRpcBoolean("agent.stop", params, context)
        }
    }

    // ========================================================================
    // Workflow Methods - NEW
    // ========================================================================

    override suspend fun workflowTemplates(context: OperationContext?): RpcResult<WorkflowTemplatesResponse> {
        val ctx = context ?: OperationContext.create()
        return executeWithRetry("workflow.templates", ctx) { context ->
            callRpcTyped<WorkflowTemplatesResponse>("workflow.templates", null, context)
        }
    }

    override suspend fun workflowStart(
        templateId: String,
        params: Map<String, Any?>,
        roomId: String?,
        context: OperationContext?
    ): RpcResult<WorkflowStartResponse> {
        val ctx = context ?: OperationContext.create()
        val rpcParams = buildMap {
            put("template_id", JsonPrimitive(templateId))
            if (roomId != null) put("room_id", JsonPrimitive(roomId))
            if (params.isNotEmpty()) {
                put("params", Json.encodeToJsonElement(params))
            }
        }
        return executeWithRetry("workflow.start", ctx) { context ->
            callRpcTyped<WorkflowStartResponse>("workflow.start", rpcParams, context)
        }
    }

    override suspend fun workflowStatus(workflowId: String, context: OperationContext?): RpcResult<WorkflowStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("workflow_id" to JsonPrimitive(workflowId))
        return executeWithRetry("workflow.status", ctx) { context ->
            callRpcTyped<WorkflowStatusResponse>("workflow.status", params, context)
        }
    }

    // ========================================================================
    // HITL Methods - NEW
    // ========================================================================

    override suspend fun hitlPending(context: OperationContext?): RpcResult<HitlPendingResponse> {
        val ctx = context ?: OperationContext.create()
        return executeWithRetry("hitl.pending", ctx) { context ->
            callRpcTyped<HitlPendingResponse>("hitl.pending", null, context)
        }
    }

    override suspend fun hitlApprove(gateId: String, notes: String?, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = buildMap {
            put("gate_id", JsonPrimitive(gateId))
            if (notes != null) put("notes", JsonPrimitive(notes))
        }
        return executeWithRetry("hitl.approve", ctx) { context ->
            callRpcBoolean("hitl.approve", params, context)
        }
    }

    override suspend fun hitlReject(gateId: String, reason: String?, context: OperationContext?): RpcResult<Boolean> {
        val ctx = context ?: OperationContext.create()
        val params = buildMap {
            put("gate_id", JsonPrimitive(gateId))
            if (reason != null) put("reason", JsonPrimitive(reason))
        }
        return executeWithRetry("hitl.reject", ctx) { context ->
            callRpcBoolean("hitl.reject", params, context)
        }
    }

    // ========================================================================
    // Budget Methods - NEW
    // ========================================================================

    override suspend fun budgetStatus(context: OperationContext?): RpcResult<BudgetStatusResponse> {
        val ctx = context ?: OperationContext.create()
        return executeWithRetry("budget.status", ctx) { context ->
            callRpcTyped<BudgetStatusResponse>("budget.status", null, context)
        }
    }

    // ========================================================================
    // Browser Queue Methods - NEW
    // ========================================================================

    override suspend fun browserEnqueue(
        agentId: String,
        roomId: String,
        url: String,
        commands: List<BrowserCommand>,
        priority: BrowserJobPriority,
        context: OperationContext?
    ): RpcResult<BrowserEnqueueResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "agent_id" to JsonPrimitive(agentId),
            "room_id" to JsonPrimitive(roomId),
            "url" to JsonPrimitive(url),
            "commands" to JsonArray(commands.map { cmd ->
                JsonObject(buildMap {
                    put("type", JsonPrimitive(cmd.type))
                    put("params", cmd.params.toJsonElement())
                })
            }),
            "priority" to JsonPrimitive(priority.name.lowercase()),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("browser.enqueue", ctx) { context ->
            callRpcTyped<BrowserEnqueueResponse>("browser.enqueue", params, context)
        }
    }

    override suspend fun browserGetJob(
        jobId: String,
        context: OperationContext?
    ): RpcResult<BrowserJobResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "job_id" to JsonPrimitive(jobId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("browser.get_job", ctx) { context ->
            callRpcTyped<BrowserJobResponse>("browser.get_job", params, context)
        }
    }

    override suspend fun browserCancelJob(
        jobId: String,
        context: OperationContext?
    ): RpcResult<BrowserCancelResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "job_id" to JsonPrimitive(jobId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("browser.cancel", ctx) { context ->
            callRpcTyped<BrowserCancelResponse>("browser.cancel", params, context)
        }
    }

    override suspend fun browserRetryJob(
        jobId: String,
        context: OperationContext?
    ): RpcResult<BrowserRetryResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "job_id" to JsonPrimitive(jobId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("browser.retry", ctx) { context ->
            callRpcTyped<BrowserRetryResponse>("browser.retry", params, context)
        }
    }

    override suspend fun browserListJobs(
        status: BrowserJobStatus?,
        agentId: String?,
        limit: Int,
        offset: Int,
        context: OperationContext?
    ): RpcResult<BrowserJobListResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mutableMapOf<String, JsonElement>(
            "limit" to JsonPrimitive(limit),
            "offset" to JsonPrimitive(offset),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        status?.let { params["status"] = JsonPrimitive(it.name.lowercase()) }
        agentId?.let { params["agent_id"] = JsonPrimitive(it) }
        return executeWithRetry("browser.list", ctx) { context ->
            callRpcTyped<BrowserJobListResponse>("browser.list", params, context)
        }
    }

    override suspend fun browserQueueStats(
        context: OperationContext?
    ): RpcResult<BrowserQueueStatsResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("browser.stats", ctx) { context ->
            callRpcTyped<BrowserQueueStatsResponse>("browser.stats", params, context)
        }
    }

    // ========================================================================
    // Agent Status Methods
    // ========================================================================

    override suspend fun agentGetStatus(
        agentId: String,
        context: OperationContext?
    ): RpcResult<AgentStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "agent_id" to JsonPrimitive(agentId),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("agent.get_status", ctx) { context ->
            callRpcTyped<AgentStatusResponse>("agent.get_status", params, context)
        }
    }

    override suspend fun agentStatusHistory(
        agentId: String,
        limit: Int,
        context: OperationContext?
    ): RpcResult<AgentStatusHistoryResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf(
            "agent_id" to JsonPrimitive(agentId),
            "limit" to JsonPrimitive(limit),
            "correlation_id" to JsonPrimitive(ctx.correlationId)
        )
        return executeWithRetry("agent.status_history", ctx) { context ->
            callRpcTyped<AgentStatusHistoryResponse>("agent.status_history", params, context)
        }
    }

    // ========================================================================
    // Keystore / Zero-Trust Methods
    // ========================================================================

    override suspend fun keystoreSealed(
        context: OperationContext?
    ): RpcResult<KeystoreStatusResponse> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("keystore.sealed", ctx) { context ->
            callRpcTyped<KeystoreStatusResponse>("keystore.sealed", params, context)
        }
    }

    override suspend fun keystoreUnsealChallenge(
        context: OperationContext?
    ): RpcResult<UnsealChallenge> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("keystore.unseal_challenge", ctx) { context ->
            callRpcTyped<UnsealChallenge>("keystore.unseal_challenge", params, context)
        }
    }

    override suspend fun keystoreUnsealRespond(
        request: UnsealRequest,
        context: OperationContext?
    ): RpcResult<UnsealResult> {
        val ctx = context ?: OperationContext.create()
        val params = buildMap {
            put("challenge_id", JsonPrimitive(request.challengeId))
            put("wrapped_key", JsonPrimitive(request.wrappedKey))
            put("unseal_method", JsonPrimitive(request.unsealMethod))
            request.clientPublicKey?.let { put("client_public_key", JsonPrimitive(it)) }
            put("correlation_id", JsonPrimitive(ctx.correlationId))
        }
        return executeWithRetry("keystore.unseal_respond", ctx) { context ->
            callRpcTyped<UnsealResult>("keystore.unseal_respond", params, context)
        }
    }

    override suspend fun keystoreExtendSession(
        context: OperationContext?
    ): RpcResult<SessionExtensionResult> {
        val ctx = context ?: OperationContext.create()
        val params = mapOf("correlation_id" to JsonPrimitive(ctx.correlationId))
        return executeWithRetry("keystore.extend_session", ctx) { context ->
            callRpcTyped<SessionExtensionResult>("keystore.extend_session", params, context)
        }
    }

    fun close() { httpClient.close() }
}
