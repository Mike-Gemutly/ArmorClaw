package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow

/**
 * WebSocket client for real-time bridge events
 *
 * Manages WebSocket connection to the ArmorClaw bridge server and
 * provides real-time event streaming for messages, presence, calls, etc.
 */
interface BridgeWebSocketClient {

    /**
     * Current connection state
     */
    val connectionState: StateFlow<WebSocketState>

    /**
     * Stream of all incoming bridge events
     */
    val events: Flow<BridgeEvent>

    /**
     * Stream of connection errors
     */
    val errors: Flow<Throwable>

    /**
     * Whether the client is currently connected
     */
    fun isConnected(): Boolean

    /**
     * Connect to the bridge WebSocket endpoint
     *
     * @param sessionId The bridge session ID from RPC login
     * @param accessToken Optional access token for authentication
     * @param context Operation context for logging
     * @return true if connection initiated successfully
     */
    suspend fun connect(
        sessionId: String,
        accessToken: String? = null,
        context: OperationContext? = null
    ): Boolean

    /**
     * Disconnect from the WebSocket server
     *
     * @param reason Optional reason for disconnect
     */
    suspend fun disconnect(reason: String? = null)

    /**
     * Subscribe to events for a specific room
     *
     * @param roomId The room to subscribe to
     * @param context Operation context for logging
     */
    suspend fun subscribeToRoom(
        roomId: String,
        context: OperationContext? = null
    )

    /**
     * Unsubscribe from events for a specific room
     *
     * @param roomId The room to unsubscribe from
     * @param context Operation context for logging
     */
    suspend fun unsubscribeFromRoom(
        roomId: String,
        context: OperationContext? = null
    )

    /**
     * Subscribe to presence updates for specific users
     *
     * @param userIds List of user IDs to subscribe to
     * @param context Operation context for logging
     */
    suspend fun subscribeToPresence(
        userIds: List<String>,
        context: OperationContext? = null
    )

    /**
     * Send a typing notification
     *
     * @param roomId The room where typing is occurring
     * @param typing Whether the user is typing
     * @param context Operation context for logging
     */
    suspend fun sendTypingNotification(
        roomId: String,
        typing: Boolean,
        context: OperationContext? = null
    )

    /**
     * Send a read receipt
     *
     * @param roomId The room ID
     * @param eventId The event ID that was read
     * @param context Operation context for logging
     */
    suspend fun sendReadReceipt(
        roomId: String,
        eventId: String,
        context: OperationContext? = null
    )

    /**
     * Send a ping to keep the connection alive
     */
    suspend fun ping()

    /**
     * Get events filtered by type
     */
    fun <T : BridgeEvent> getEventsOfType(eventClass: Class<T>): Flow<T>

    /**
     * Get message events only
     */
    fun getMessageEvents(): Flow<BridgeEvent.MessageReceived>

    /**
     * Get typing events only
     */
    fun getTypingEvents(): Flow<BridgeEvent.TypingNotification>

    /**
     * Get presence events only
     */
    fun getPresenceEvents(): Flow<BridgeEvent.PresenceUpdate>

    /**
     * Get call events only
     */
    fun getCallEvents(): Flow<BridgeEvent.CallEvent>

    /**
     * Get room events (created, membership changed)
     */
    fun getRoomEvents(): Flow<BridgeEvent>
}

/**
 * WebSocket connection configuration
 */
data class WebSocketConfig(
    val baseUrl: String,
    val reconnectEnabled: Boolean = true,
    val reconnectDelayMs: Long = 1000,
    val maxReconnectDelayMs: Long = 30000,
    val maxReconnectAttempts: Int = 10,
    val pingIntervalMs: Long = 30000,
    val connectionTimeoutMs: Long = 10000
) {
    companion object {
        val DEVELOPMENT = WebSocketConfig(
            baseUrl = "ws://localhost:8080",
            reconnectEnabled = true,
            pingIntervalMs = 30000
        )

        val PRODUCTION = WebSocketConfig(
            baseUrl = "wss://bridge.armorclaw.app",
            reconnectEnabled = true,
            pingIntervalMs = 30000
        )
    }
}
