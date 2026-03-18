package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.matrix.MatrixSyncManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant

/**
 * Repository that integrates BridgeRpcClient, BridgeWebSocketClient, and MatrixSyncManager
 *
 * This is the main integration layer between the domain layer and the bridge server.
 * It handles:
 * - Starting/stopping the bridge session
 * - Matrix login and authentication
 * - Message sending/receiving via RPC
 * - Real-time event processing via Matrix /sync (WebSocket is a stub)
 * - Room and presence synchronization
 *
 * ## Event Mode
 * If useDirectMatrixSync is true (default), real-time events come from Matrix /sync.
 * Otherwise, events come from Bridge WebSocket (when implemented).
 */
class BridgeRepository(
    private val rpcClient: BridgeRpcClient,
    private val wsClient: BridgeWebSocketClient,
    private val syncManager: MatrixSyncManager? = null,
    private val config: BridgeConfig = BridgeConfig.DEVELOPMENT
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // Connection state
    private val _isConnected = MutableStateFlow(false)
    val isConnected: StateFlow<Boolean> = _isConnected.asStateFlow()

    private val _sessionState = MutableStateFlow<BridgeSessionState>(BridgeSessionState.Disconnected)
    val sessionState: StateFlow<BridgeSessionState> = _sessionState.asStateFlow()

    // Current session info
    private var currentSession: BridgeSession? = null
    private var currentUser: User? = null
    private var currentAccessToken: String? = null

    /**
     * Start the bridge server container
     */
    suspend fun startBridge(
        userId: String,
        deviceId: String,
        context: OperationContext? = null
    ): AppResult<BridgeSession> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.startBridge(userId, deviceId, ctx)) {
            is RpcResult.Success -> {
                val session = BridgeSession(
                    sessionId = result.data.sessionId,
                    containerId = result.data.containerId,
                    status = result.data.status,
                    createdAt = Clock.System.now()
                )
                currentSession = session
                _sessionState.value = BridgeSessionState.Connected(session.sessionId)
                _isConnected.value = true

                AppLogger.info(
                    LogTag.Network.Bridge,
                    "Bridge started successfully",
                    mapOf("sessionId" to session.sessionId)
                )

                AppResult.success(session)
            }
            is RpcResult.Error -> {
                AppLogger.error(
                    LogTag.Network.Bridge,
                    "Failed to start bridge: ${result.message}",
                    null,
                    mapOf("code" to result.code)
                )
                createErrorResult(result, "startBridge")
            }
        }
    }

    /**
     * Stop the bridge server
     */
    suspend fun stopBridge(context: OperationContext? = null): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        val sessionId = currentSession?.sessionId ?: return AppResult.success(Unit)

        return when (val result = rpcClient.stopBridge(sessionId, ctx)) {
            is RpcResult.Success -> {
                // Stop WebSocket
                wsClient.disconnect("Bridge stopping")

                // Stop Matrix sync if running
                syncManager?.stopSync()

                currentSession = null
                currentAccessToken = null
                _sessionState.value = BridgeSessionState.Disconnected
                _isConnected.value = false

                AppResult.success(Unit)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "stopBridge")
            }
        }
    }

    /**
     * Login to Matrix via the bridge
     */
    suspend fun matrixLogin(
        homeserver: String,
        username: String,
        password: String,
        deviceId: String,
        context: OperationContext? = null
    ): AppResult<UserSession> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.matrixLogin(homeserver, username, password, deviceId, ctx)) {
            is RpcResult.Success -> {
                val loginResponse = result.data

                // Set access token for WebSocket
                (wsClient as? BridgeWebSocketClientImpl)?.let {
                    // Token is managed internally by RPC client
                }

                val session = UserSession(
                    userId = loginResponse.userId,
                    accessToken = loginResponse.accessToken,
                    refreshToken = loginResponse.refreshToken ?: "",
                    deviceId = loginResponse.deviceId,
                    homeserver = homeserver,
                    expiresAt = Instant.DISTANT_FUTURE // TODO: Parse from response
                )

                currentUser = User(
                    id = loginResponse.userId,
                    displayName = loginResponse.displayName ?: username,
                    email = null,
                    avatar = loginResponse.avatarUrl,
                    presence = UserPresence.ONLINE,
                    lastActive = Clock.System.now(),
                    isVerified = true
                )

                // Store access token for sync
                currentAccessToken = loginResponse.accessToken

                AppLogger.info(
                    LogTag.Network.Bridge,
                    "Matrix login successful",
                    mapOf("userId" to session.userId)
                )

                AppResult.success(session)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "matrixLogin")
            }
        }
    }

    /**
     * Connect for real-time events
     *
     * If config.useDirectMatrixSync is true, starts Matrix /sync.
     * Otherwise, connects to Bridge WebSocket.
     */
    suspend fun connectForEvents(context: OperationContext? = null): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        val accessToken = currentAccessToken
            ?: return AppResult.error(
                AppError(
                    code = ArmorClawErrorCode.SESSION_EXPIRED.code,
                    message = "No access token - login first",
                    source = "BridgeRepository"
                )
            )

        return if (config.useDirectMatrixSync && syncManager != null) {
            // Use Matrix /sync for real-time events
            AppLogger.info(LogTag.Network.Bridge, "Starting Matrix sync for real-time events")
            syncManager.startSync(accessToken)
            AppResult.success(Unit)
        } else {
            // Fallback to WebSocket (stub)
            val sessionId = currentSession?.sessionId
                ?: return AppResult.error(
                    AppError(
                        code = ArmorClawErrorCode.SESSION_EXPIRED.code,
                        message = "No active bridge session",
                        source = "BridgeRepository"
                    )
                )

            val connected = wsClient.connect(sessionId, accessToken, ctx)
            if (connected) {
                AppResult.success(Unit)
            } else {
                AppResult.error(
                    AppError(
                        code = ArmorClawErrorCode.NETWORK_CHANGED.code,
                        message = "Failed to connect WebSocket",
                        source = "BridgeRepository"
                    )
                )
            }
        }
    }

    /**
     * Connect to WebSocket for real-time events (legacy)
     *
     * @deprecated Use connectForEvents() instead
     */
    @Deprecated(
        message = "Use connectForEvents() instead",
        replaceWith = ReplaceWith("connectForEvents(context)")
    )
    suspend fun connectWebSocket(context: OperationContext? = null): AppResult<Unit> {
        return connectForEvents(context)
    }

    /**
     * Get the event flow
     *
     * If using Matrix sync, returns events from syncManager.
     * Otherwise, returns events from WebSocket.
     */
    fun getEventFlow(): Flow<BridgeEvent> = wsClient.events

    /**
     * Get Matrix sync events directly (if using direct sync)
     */
    fun getSyncEventFlow(): Flow<com.armorclaw.shared.platform.matrix.MatrixSyncEvent>? {
        return syncManager?.events
    }

    /**
     * Get connection state flow
     */
    fun getConnectionStateFlow(): Flow<WebSocketState> = wsClient.connectionState

    /**
     * Send a message via the bridge
     */
    suspend fun sendMessage(
        roomId: String,
        content: MessageContent,
        context: OperationContext? = null
    ): AppResult<Pair<String, String?>> { // Returns (eventId, txnId)
        val ctx = context ?: OperationContext.create()

        val eventType = "m.room.message"

        val msgtype = when (content.type) {
            MessageType.TEXT -> "m.text"
            MessageType.IMAGE -> "m.image"
            MessageType.VIDEO -> "m.video"
            MessageType.AUDIO -> "m.audio"
            MessageType.FILE -> "m.file"
            MessageType.NOTICE -> "m.notice"
            MessageType.EMOTE -> "m.emote"
        }

        val contentMap = mutableMapOf<String, Any?>(
            "msgtype" to msgtype,
            "body" to content.body
        )

        // Add formatted body if present
        content.formattedBody?.let { contentMap["formatted_body"] = it }

        // Add attachment info if present
        if (content.attachments.isNotEmpty()) {
            val attachment = content.attachments.first()
            contentMap["url"] = attachment.url
            contentMap["info"] = mapOf(
                "mimetype" to attachment.mimeType,
                "size" to attachment.size,
                "filename" to attachment.fileName
            )
        }

        return when (val result = rpcClient.matrixSend(roomId, eventType, contentMap, null, ctx)) {
            is RpcResult.Success -> {
                AppResult.success(Pair(result.data.eventId, result.data.txnId))
            }
            is RpcResult.Error -> {
                createErrorResult(result, "sendMessage")
            }
        }
    }

    /**
     * Create a new Matrix room
     */
    suspend fun createRoom(
        name: String? = null,
        topic: String? = null,
        isDirect: Boolean = false,
        invite: List<String>? = null,
        context: OperationContext? = null
    ): AppResult<String> { // Returns roomId
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.matrixCreateRoom(name, topic, isDirect, invite, ctx)) {
            is RpcResult.Success -> {
                val roomId = result.data.roomId
                // Subscribe to room events
                wsClient.subscribeToRoom(roomId, ctx)
                AppResult.success(roomId)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "createRoom")
            }
        }
    }

    /**
     * Join a Matrix room
     */
    suspend fun joinRoom(
        roomIdOrAlias: String,
        context: OperationContext? = null
    ): AppResult<String> { // Returns roomId
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.matrixJoinRoom(roomIdOrAlias, ctx)) {
            is RpcResult.Success -> {
                val roomId = result.data.roomId
                // Subscribe to room events
                wsClient.subscribeToRoom(roomId, ctx)
                AppResult.success(roomId)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "joinRoom")
            }
        }
    }

    /**
     * Leave a Matrix room
     */
    suspend fun leaveRoom(
        roomId: String,
        context: OperationContext? = null
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.matrixLeaveRoom(roomId, ctx)) {
            is RpcResult.Success -> {
                // Unsubscribe from room events
                wsClient.unsubscribeFromRoom(roomId, ctx)
                AppResult.success(Unit)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "leaveRoom")
            }
        }
    }

    /**
     * Invite a user to a Matrix room
     */
    suspend fun inviteUser(
        roomId: String,
        userId: String,
        context: OperationContext? = null
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.matrixInviteUser(roomId, userId, ctx)) {
            is RpcResult.Success -> {
                AppResult.success(Unit)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "inviteUser")
            }
        }
    }

    /**
     * Send typing notification
     */
    suspend fun sendTyping(
        roomId: String,
        typing: Boolean,
        context: OperationContext? = null
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.matrixSendTyping(roomId, typing, 30000, ctx)) {
            is RpcResult.Success -> {
                AppResult.success(Unit)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "sendTyping")
            }
        }
    }

    /**
     * Send read receipt
     */
    suspend fun sendReadReceipt(
        roomId: String,
        eventId: String,
        context: OperationContext? = null
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.matrixSendReadReceipt(roomId, eventId, ctx)) {
            is RpcResult.Success -> {
                AppResult.success(Unit)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "sendReadReceipt")
            }
        }
    }

    /**
     * Subscribe to room events
     */
    suspend fun subscribeToRoom(roomId: String, context: OperationContext? = null) {
        wsClient.subscribeToRoom(roomId, context)
    }

    /**
     * Unsubscribe from room events
     */
    suspend fun unsubscribeFromRoom(roomId: String, context: OperationContext? = null) {
        wsClient.unsubscribeFromRoom(roomId, context)
    }

    /**
     * Subscribe to presence updates
     */
    suspend fun subscribeToPresence(userIds: List<String>, context: OperationContext? = null) {
        wsClient.subscribeToPresence(userIds, context)
    }

    /**
     * Get bridge health status
     */
    suspend fun getHealthStatus(context: OperationContext? = null): AppResult<Map<String, Any?>> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.healthCheck(ctx)) {
            is RpcResult.Success -> {
                AppResult.success(result.data)
            }
            is RpcResult.Error -> {
                createErrorResult(result, "healthCheck")
            }
        }
    }

    /**
     * Get the current user
     */
    fun getCurrentUser(): User? = currentUser

    /**
     * Get the current session
     */
    fun getCurrentSession(): BridgeSession? = currentSession

    /**
     * Disconnect and cleanup
     */
    fun close() {
        scope.launch {
            stopBridge()
        }
        syncManager?.close()
    }

    // Private helpers

    private fun createErrorResult(rpcError: RpcResult.Error, operation: String): AppResult<Nothing> {
        val errorCode = mapRpcErrorToArmorClawError(rpcError.code)
        return AppResult.error(
            AppError(
                code = errorCode.code,
                message = rpcError.message,
                source = "BridgeRepository:$operation",
                isRecoverable = errorCode.recoverable
            )
        )
    }

    private fun mapRpcErrorToArmorClawError(rpcCode: Int): ArmorClawErrorCode {
        return when (rpcCode) {
            JsonRpcError.AUTH_FAILED, JsonRpcError.SESSION_EXPIRED -> ArmorClawErrorCode.SESSION_EXPIRED
            JsonRpcError.INVALID_PARAMS -> ArmorClawErrorCode.MESSAGE_SEND_FAILED
            JsonRpcError.METHOD_NOT_FOUND -> ArmorClawErrorCode.UNKNOWN_ERROR
            JsonRpcError.NETWORK_ERROR -> ArmorClawErrorCode.NETWORK_CHANGED
            else -> ArmorClawErrorCode.UNKNOWN_ERROR
        }
    }
}

/**
 * Bridge session state
 */
sealed class BridgeSessionState {
    object Disconnected : BridgeSessionState()
    data class Connecting(val attempt: Int = 0) : BridgeSessionState()
    data class Connected(val sessionId: String) : BridgeSessionState()
    data class Error(val message: String) : BridgeSessionState()
}

/**
 * Bridge session info
 */
data class BridgeSession(
    val sessionId: String,
    val containerId: String?,
    val status: String,
    val createdAt: Instant
)
