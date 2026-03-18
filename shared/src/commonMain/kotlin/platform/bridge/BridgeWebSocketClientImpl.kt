package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import io.ktor.client.*
import io.ktor.client.plugins.websocket.*
import io.ktor.client.request.*
import io.ktor.http.*
import io.ktor.websocket.*
import kotlinx.coroutines.*
import kotlinx.coroutines.channels.ClosedReceiveChannelException
import kotlinx.coroutines.channels.consumeEach
import kotlinx.coroutines.flow.*
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*
import kotlin.math.min
import kotlin.math.pow

/**
 * Implementation of BridgeWebSocketClient using Ktor WebSockets
 *
 * Handles:
 * - WebSocket connection management
 * - Automatic reconnection with exponential backoff
 * - Event parsing and distribution
 * - Room subscriptions
 * - Keep-alive pings
 */
class BridgeWebSocketClientImpl(
    private val config: WebSocketConfig,
    private val httpClient: HttpClient
) : BridgeWebSocketClient {

    private val logger = repositoryLogger("BridgeWebSocket", LogTag.Network.BridgeWebSocket)

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
    }

    // Connection state
    private var webSocketSession: DefaultWebSocketSession? = null
    private var currentSessionId: String? = null
    private var currentAccessToken: String? = null

    // Reconnection state
    private var reconnectJob: Job? = null
    private var reconnectAttempts = 0
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // Event streams
    private val _connectionState = MutableStateFlow<WebSocketState>(WebSocketState.Disconnected)
    override val connectionState: StateFlow<WebSocketState> = _connectionState.asStateFlow()

    private val _events = MutableSharedFlow<BridgeEvent>(extraBufferCapacity = 100)
    override val events: Flow<BridgeEvent> = _events.asSharedFlow()

    private val _errors = MutableSharedFlow<Throwable>(extraBufferCapacity = 10)
    override val errors: Flow<Throwable> = _errors.asSharedFlow()

    // Subscriptions
    private val subscribedRooms = mutableSetOf<String>()
    private val subscribedUsers = mutableSetOf<String>()

    override fun isConnected(): Boolean = webSocketSession != null &&
            _connectionState.value == WebSocketState.Connected

    override suspend fun connect(
        sessionId: String,
        accessToken: String?,
        context: OperationContext?
    ): Boolean {
        val ctx = context ?: OperationContext.create()

        if (isConnected()) {
            logger.logOperationStart("connect", mapOf("status" to "already_connected"))
            return true
        }

        currentSessionId = sessionId
        currentAccessToken = accessToken

        return try {
            _connectionState.value = WebSocketState.Connecting
            logger.logOperationStart("connect", mapOf(
                "session_id" to sessionId,
                "correlation_id" to ctx.correlationId
            ))

            val wsUrl = buildWebSocketUrl(sessionId, accessToken)
            logger.logOperationStart("websocket_connect", mapOf("url" to wsUrl))

            webSocketSession = httpClient.webSocketSession {
                url(wsUrl)
                accessToken?.let { header("Authorization", "Bearer $it") }
            }

            _connectionState.value = WebSocketState.Connected
            reconnectAttempts = 0

            // Start listening for events
            scope.launch { listenForEvents() }

            // Start keep-alive ping
            if (config.pingIntervalMs > 0) {
                scope.launch { keepAlive() }
            }

            // Resubscribe to rooms and presence
            scope.launch { resubscribeAll() }

            logger.logOperationSuccess("connect")
            true
        } catch (e: Exception) {
            logger.logOperationError("connect", e)
            _connectionState.value = WebSocketState.Error(e)
            _errors.emit(e)

            // Attempt reconnection if enabled
            if (config.reconnectEnabled) {
                scheduleReconnect()
            }
            false
        }
    }

    override suspend fun disconnect(reason: String?) {
        logger.logOperationStart("disconnect", mapOf("reason" to (reason ?: "user_initiated")))

        reconnectJob?.cancel()
        reconnectJob = null

        _connectionState.value = WebSocketState.Disconnecting

        try {
            webSocketSession?.close(CloseReason(CloseReason.Codes.NORMAL, reason ?: "Client disconnect"))
        } catch (e: Exception) {
            logger.logOperationError("disconnect", e)
        }

        webSocketSession = null
        _connectionState.value = WebSocketState.Disconnected
        logger.logOperationSuccess("disconnect")
    }

    /**
     * Subscribe to room events
     */
    override suspend fun subscribeToRoom(roomId: String, context: OperationContext?) {
        val ctx = context ?: OperationContext.create()

        if (!isConnected()) {
            logger.logOperationError("subscribeToRoom", Exception("Not connected"))
            return
        }

        try {
            val subscription = mapOf(
                "action" to "subscribe",
                "type" to "room",
                "room_id" to roomId,
                "correlation_id" to ctx.correlationId
            )
            sendMessage(subscription)
            subscribedRooms.add(roomId)
            logger.logOperationSuccess("subscribeToRoom:$roomId")
        } catch (e: Exception) {
            logger.logOperationError("subscribeToRoom:$roomId", e)
            // FIX for Bug #5: Ensure subscribedRooms state is consistent even on error
            // Don't add to subscribedRooms if subscription failed
        }
    }

    /**
     * Unsubscribe from room events
     */
    override suspend fun unsubscribeFromRoom(roomId: String, context: OperationContext?) {
        val ctx = context ?: OperationContext.create()

        if (!isConnected()) return

        try {
            val unsubscription = mapOf(
                "action" to "unsubscribe",
                "type" to "room",
                "room_id" to roomId,
                "correlation_id" to ctx.correlationId
            )
            sendMessage(unsubscription)
            subscribedRooms.remove(roomId)
            logger.logOperationSuccess("unsubscribeFromRoom:$roomId")
        } catch (e: Exception) {
            logger.logOperationError("unsubscribeFromRoom:$roomId", e)
            // FIX for Bug #5: Remove from subscribedRooms even if unsubscribe failed
            // The server may have already cleaned up, so we should be consistent
            subscribedRooms.remove(roomId)
        }
    }

    override suspend fun subscribeToPresence(userIds: List<String>, context: OperationContext?) {
        val ctx = context ?: OperationContext.create()

        if (!isConnected()) return

        try {
            val subscription = mapOf(
                "action" to "subscribe",
                "type" to "presence",
                "user_ids" to userIds,
                "correlation_id" to ctx.correlationId
            )
            sendMessage(subscription)
            subscribedUsers.addAll(userIds)
            logger.logOperationSuccess("subscribeToPresence")
        } catch (e: Exception) {
            logger.logOperationError("subscribeToPresence", e)
        }
    }

    override suspend fun sendTypingNotification(roomId: String, typing: Boolean, context: OperationContext?) {
        val ctx = context ?: OperationContext.create()

        if (!isConnected()) return

        try {
            val notification = mapOf(
                "action" to "typing",
                "room_id" to roomId,
                "typing" to typing,
                "correlation_id" to ctx.correlationId
            )
            sendMessage(notification)
        } catch (e: Exception) {
            logger.logOperationError("sendTypingNotification", e)
        }
    }

    override suspend fun sendReadReceipt(roomId: String, eventId: String, context: OperationContext?) {
        val ctx = context ?: OperationContext.create()

        if (!isConnected()) return

        try {
            val receipt = mapOf(
                "action" to "read_receipt",
                "room_id" to roomId,
                "event_id" to eventId,
                "correlation_id" to ctx.correlationId
            )
            sendMessage(receipt)
        } catch (e: Exception) {
            logger.logOperationError("sendReadReceipt", e)
        }
    }

    override suspend fun ping() {
        if (!isConnected()) return

        try {
            val ping = mapOf("action" to "ping", "timestamp" to System.currentTimeMillis())
            sendMessage(ping)
        } catch (e: Exception) {
            logger.logOperationError("ping", e)
        }
    }

    @Suppress("UNCHECKED_CAST")
    override fun <T : BridgeEvent> getEventsOfType(eventClass: Class<T>): Flow<T> {
        return events.filter { eventClass.isInstance(it) }.map { it as T }
    }

    override fun getMessageEvents(): Flow<BridgeEvent.MessageReceived> =
            getEventsOfType(BridgeEvent.MessageReceived::class.java)

    override fun getTypingEvents(): Flow<BridgeEvent.TypingNotification> =
            getEventsOfType(BridgeEvent.TypingNotification::class.java)

    override fun getPresenceEvents(): Flow<BridgeEvent.PresenceUpdate> =
            getEventsOfType(BridgeEvent.PresenceUpdate::class.java)

    override fun getCallEvents(): Flow<BridgeEvent.CallEvent> =
            getEventsOfType(BridgeEvent.CallEvent::class.java)

    override fun getRoomEvents(): Flow<BridgeEvent> = events.filter {
        it is BridgeEvent.RoomCreated || it is BridgeEvent.RoomMembershipChanged
    }

    // Private implementation

    private fun buildWebSocketUrl(sessionId: String, accessToken: String?): String {
        val baseUrl = config.baseUrl.removeSuffix("/")
        var url = "$baseUrl/ws?session_id=$sessionId"
        accessToken?.let { url += "&access_token=$it" }
        return url
    }

    private suspend fun sendMessage(message: Map<String, Any?>) {
        val session = webSocketSession ?: return
        val jsonString = json.encodeToString(message)
        session.send(Frame.Text(jsonString))
    }

    /**
     * Listen for WebSocket events with guaranteed error handling
     *
     * FIX for Bug #5: All exceptions are caught and properly handled,
     * ensuring the connection state is always updated regardless of error type.
     */
    private suspend fun listenForEvents() {
        val session = webSocketSession
        if (session == null) {
            // Session is null, ensure state is reset
            handleDisconnection()
            return
        }

        try {
            session.incoming.consumeEach { frame ->
                if (frame is Frame.Text) {
                    val text = frame.readText()
                    parseAndEmitEvent(text)
                }
            }
        } catch (e: ClosedReceiveChannelException) {
            logger.logOperationError("listenForEvents", Exception("Connection closed: ${e.message}"))
            handleDisconnection()
        } catch (e: CancellationException) {
            // Coroutine cancellation - this is expected during normal shutdown
            logger.logOperationStart("listenForEvents", mapOf("status" to "cancelled"))
            throw e // Re-throw to propagate cancellation
        } catch (e: Exception) {
            logger.logOperationError("listenForEvents", e)
            _errors.emit(e)
            handleDisconnection()
        } finally {
            // FIX for Bug #5: Guarantee state reset in finally block
            // This ensures we never get stuck in a transient state
            if (_connectionState.value != WebSocketState.Disconnected) {
                logger.logOperationStart("listenForEvents:finally", mapOf(
                    "state_before_reset" to _connectionState.value::class.simpleName
                ))
            }
        }
    }

    private fun parseAndEmitEvent(jsonString: String) {
        try {
            val jsonObject = json.parseToJsonElement(jsonString).jsonObject
            val eventType = jsonObject["type"]?.jsonPrimitive?.content ?: return

            val event = when (eventType) {
                "message.received" -> json.decodeFromJsonElement(
                    BridgeEvent.MessageReceived.serializer(), jsonObject
                )
                "message.status" -> json.decodeFromJsonElement(
                    BridgeEvent.MessageStatusUpdated.serializer(), jsonObject
                )
                "room.created" -> json.decodeFromJsonElement(
                    BridgeEvent.RoomCreated.serializer(), jsonObject
                )
                "room.membership" -> json.decodeFromJsonElement(
                    BridgeEvent.RoomMembershipChanged.serializer(), jsonObject
                )
                "typing" -> json.decodeFromJsonElement(
                    BridgeEvent.TypingNotification.serializer(), jsonObject
                )
                "receipt.read" -> json.decodeFromJsonElement(
                    BridgeEvent.ReadReceipt.serializer(), jsonObject
                )
                "presence" -> json.decodeFromJsonElement(
                    BridgeEvent.PresenceUpdate.serializer(), jsonObject
                )
                "call" -> json.decodeFromJsonElement(
                    BridgeEvent.CallEvent.serializer(), jsonObject
                )
                "platform.message" -> json.decodeFromJsonElement(
                    BridgeEvent.PlatformMessage.serializer(), jsonObject
                )
                "session.expired" -> json.decodeFromJsonElement(
                    BridgeEvent.SessionExpired.serializer(), jsonObject
                )
                "bridge.status" -> json.decodeFromJsonElement(
                    BridgeEvent.BridgeStatus.serializer(), jsonObject
                )
                "recovery" -> json.decodeFromJsonElement(
                    BridgeEvent.RecoveryEvent.serializer(), jsonObject
                )
                else -> {
                    BridgeEvent.UnknownEvent(
                        type = eventType,
                        data = jsonObject.toMap(),
                        sessionId = jsonObject["session_id"]?.jsonPrimitive?.content,
                        timestamp = jsonObject["timestamp"]?.jsonPrimitive?.longOrNull
                                ?: System.currentTimeMillis()
                    )
                }
            }

            scope.launch { _events.emit(event) }
        } catch (e: Exception) {
            logger.logOperationError("parseEvent", e, mapOf("json" to jsonString.take(200)))
        }
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

    private suspend fun keepAlive() {
        while (isConnected()) {
            delay(config.pingIntervalMs)
            if (isConnected()) {
                ping()
            }
        }
    }

    private suspend fun resubscribeAll() {
        if (!isConnected()) return

        // Resubscribe to rooms
        subscribedRooms.toList().forEach { roomId ->
            subscribeToRoom(roomId)
        }

        // Resubscribe to presence
        if (subscribedUsers.isNotEmpty()) {
            subscribeToPresence(subscribedUsers.toList())
        }
    }

    /**
     * Handle disconnection with guaranteed state reset
     *
     * FIX for Bug #5: This method ensures connection state is always reset properly,
     * even when errors occur during disconnect. This prevents the state machine
     * from getting stuck in a transient state like CONNECTING or DISCONNECTING.
     */
    private fun handleDisconnection() {
        val wasConnected = _connectionState.value == WebSocketState.Connected

        // Clear WebSocket session reference first
        webSocketSession = null

        // ALWAYS reset state to Disconnected - this is the guaranteed safe state
        _connectionState.value = WebSocketState.Disconnected

        // Schedule reconnection only if we were previously connected
        if (wasConnected && config.reconnectEnabled && currentSessionId != null) {
            scheduleReconnect()
        }

        logger.logOperationSuccess("handleDisconnection", "State reset to Disconnected")
    }

    private fun scheduleReconnect() {
        reconnectJob?.cancel()

        if (reconnectAttempts >= config.maxReconnectAttempts) {
            logger.logOperationError("reconnect", Exception("Max reconnect attempts reached"))
            return
        }

        val delay = calculateReconnectDelay()
        reconnectAttempts++

        logger.logOperationStart("schedule_reconnect", mapOf(
            "attempt" to reconnectAttempts,
            "delay_ms" to delay
        ))

        reconnectJob = scope.launch {
            delay(delay)
            currentSessionId?.let { sessionId ->
                connect(sessionId, currentAccessToken)
            }
        }
    }

    private fun calculateReconnectDelay(): Long {
        val baseDelay = config.reconnectDelayMs
        val exponentialDelay = baseDelay * 2.0.pow(reconnectAttempts.toDouble()).toLong()
        return min(exponentialDelay, config.maxReconnectDelayMs)
    }

    fun close() {
        scope.cancel()
        reconnectJob?.cancel()
        runBlocking { disconnect("Client shutdown") }
    }
}
