package com.armorclaw.shared.data.store

import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import com.armorclaw.shared.platform.matrix.MatrixSyncEvent
import com.armorclaw.shared.platform.matrix.MatrixSyncManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch

/**
 * Real-Time Event Store
 *
 * Provides real-time events to the application using Matrix /sync endpoint
 * instead of the Bridge WebSocket (which is currently a stub).
 *
 * ## Architecture
 * ```
 * MatrixSyncManager ───► RealTimeEventStore ───► ViewModels / UI
 *        │                      │
 *        │                      ├── messages: Flow<MessageEvent>
 *        │                      ├── typing: Flow<TypingEvent>
 *        │                      ├── presence: Flow<PresenceEvent>
 *        │                      ├── receipts: Flow<ReceiptEvent>
 *        │                      └── calls: Flow<CallSignalingEvent>
 *        │
 *        └── Matrix /sync (long-poll)
 * ```
 *
 * ## Usage
 * ```kotlin
 * val eventStore = RealTimeEventStore(syncManager)
 * eventStore.startListening(accessToken)
 *
 * // Subscribe to messages
 * eventStore.messages.collect { message ->
 *     // Handle new message
 * }
 *
 * // Subscribe to typing indicators
 * eventStore.typingNotifications.collect { typing ->
 *     // Update typing UI
 * }
 * ```
 */
class RealTimeEventStore(
    private val syncManager: MatrixSyncManager
) {
    private val logger = repositoryLogger("RealTimeEventStore", LogTag.Network.MatrixSync)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // Internal event buffer
    private val _allEvents = MutableSharedFlow<MatrixSyncEvent>(extraBufferCapacity = 128)

    // Filtered event flows
    val allEvents: SharedFlow<MatrixSyncEvent> = _allEvents.asSharedFlow()

    // Messages
    private val _messages = MutableSharedFlow<MessageEvent>(extraBufferCapacity = 64)
    val messages: SharedFlow<MessageEvent> = _messages.asSharedFlow()

    // Typing
    private val _typingNotifications = MutableSharedFlow<TypingNotificationEvent>(extraBufferCapacity = 32)
    val typingNotifications: SharedFlow<TypingNotificationEvent> = _typingNotifications.asSharedFlow()

    // Presence
    private val _presenceUpdates = MutableSharedFlow<PresenceUpdateEvent>(extraBufferCapacity = 32)
    val presenceUpdates: SharedFlow<PresenceUpdateEvent> = _presenceUpdates.asSharedFlow()

    // Receipts
    private val _readReceipts = MutableSharedFlow<ReadReceiptEvent>(extraBufferCapacity = 32)
    val readReceipts: SharedFlow<ReadReceiptEvent> = _readReceipts.asSharedFlow()

    // Room membership
    private val _roomMemberships = MutableSharedFlow<RoomMembershipEvent>(extraBufferCapacity = 32)
    val roomMemberships: SharedFlow<RoomMembershipEvent> = _roomMemberships.asSharedFlow()

    // Calls
    private val _callSignaling = MutableSharedFlow<CallSignalingEvent>(extraBufferCapacity = 16)
    val callSignaling: SharedFlow<CallSignalingEvent> = _callSignaling.asSharedFlow()

    // Room state changes
    private val _roomStateChanges = MutableSharedFlow<RoomStateChangeEvent>(extraBufferCapacity = 32)
    val roomStateChanges: SharedFlow<RoomStateChangeEvent> = _roomStateChanges.asSharedFlow()

    // Errors
    private val _errors = MutableSharedFlow<SyncErrorEvent>(extraBufferCapacity = 16)
    val errors: SharedFlow<SyncErrorEvent> = _errors.asSharedFlow()

    // Per-room event tracking
    private val roomSubscriptions = MutableStateFlow<Set<String>>(emptySet())

    init {
        // Subscribe to sync manager events
        scope.launch {
            syncManager.events.collect { event ->
                processEvent(event)
            }
        }
    }

    /**
     * Start listening for events
     *
     * @param accessToken The Matrix access token
     * @param initialSince Initial sync token (null for full sync)
     */
    fun startListening(
        accessToken: String,
        initialSince: String? = null
    ) {
        logger.logOperationStart("startListening", mapOf(
            "initial_since" to (initialSince ?: "null")
        ))

        syncManager.startSync(accessToken, initialSince)
    }

    /**
     * Stop listening for events
     */
    fun stopListening() {
        logger.logOperationStart("stopListening")
        syncManager.stopSync()
    }

    /**
     * Subscribe to events for a specific room
     *
     * @param roomId The room ID to subscribe to
     */
    fun subscribeToRoom(roomId: String) {
        roomSubscriptions.value = roomSubscriptions.value + roomId
        logger.logDebug("Subscribed to room", mapOf("room_id" to roomId))
    }

    /**
     * Unsubscribe from events for a specific room
     *
     * @param roomId The room ID to unsubscribe from
     */
    fun unsubscribeFromRoom(roomId: String) {
        roomSubscriptions.value = roomSubscriptions.value - roomId
        logger.logDebug("Unsubscribed from room", mapOf("room_id" to roomId))
    }

    /**
     * Get messages for a specific room
     */
    fun getMessagesForRoom(roomId: String): Flow<MessageEvent> {
        return messages.filter { it.roomId == roomId }
    }

    /**
     * Get typing notifications for a specific room
     */
    fun getTypingForRoom(roomId: String): Flow<TypingNotificationEvent> {
        return typingNotifications.filter { it.roomId == roomId }
    }

    /**
     * Get read receipts for a specific room
     */
    fun getReceiptsForRoom(roomId: String): Flow<ReadReceiptEvent> {
        return readReceipts.filter { it.roomId == roomId }
    }

    /**
     * Check if sync is running
     */
    fun isRunning(): Boolean = syncManager.isRunning()

    /**
     * Get current sync token
     */
    fun getSyncToken(): String? = syncManager.getSinceToken()

    /**
     * Cleanup resources
     */
    fun close() {
        stopListening()
        scope.cancel()
    }

    // Private implementation

    private suspend fun processEvent(event: MatrixSyncEvent) {
        // Emit to all events flow
        _allEvents.emit(event)

        when (event) {
            is MatrixSyncEvent.MessageReceived -> {
                val messageEvent = MessageEvent(
                    eventId = event.event.eventId ?: "",
                    roomId = event.roomId,
                    sender = event.event.sender ?: "",
                    eventType = event.event.type,
                    content = event.event.content,
                    timestamp = event.event.originServerTs ?: event.timestamp,
                    isEncrypted = event.event.type == "m.room.encrypted"
                )
                _messages.emit(messageEvent)
            }

            is MatrixSyncEvent.TypingNotification -> {
                val typingEvent = TypingNotificationEvent(
                    roomId = event.roomId,
                    userIds = event.userIds,
                    timestamp = event.timestamp
                )
                _typingNotifications.emit(typingEvent)
            }

            is MatrixSyncEvent.PresenceUpdate -> {
                val presenceEvent = PresenceUpdateEvent(
                    userId = event.event.sender ?: "",
                    presence = event.event.content?.get("presence")?.toString() ?: "offline",
                    statusMessage = event.event.content?.get("status_msg")?.toString(),
                    timestamp = event.timestamp
                )
                _presenceUpdates.emit(presenceEvent)
            }

            is MatrixSyncEvent.ReceiptEvent -> {
                // Parse receipt content
                val content = event.event.content
                content?.forEach { (eventId, receiptData) ->
                    @Suppress("UNCHECKED_CAST")
                    val readMap = (receiptData as? Map<*, *>)?.get("m.read") as? Map<*, *>
                    readMap?.forEach { (userId, userData) ->
                        @Suppress("UNCHECKED_CAST")
                        val ts = (userData as? Map<*, *>)?.get("ts") as? Long
                        val receiptEvent = ReadReceiptEvent(
                            roomId = event.roomId,
                            eventId = eventId,
                            userId = userId.toString(),
                            timestamp = ts ?: event.timestamp
                        )
                        _readReceipts.emit(receiptEvent)
                    }
                }
            }

            is MatrixSyncEvent.FullyReadMarker -> {
                val readMarkerEvent = ReadReceiptEvent(
                    roomId = event.roomId,
                    eventId = event.eventId ?: "",
                    userId = "",  // Own user
                    timestamp = event.timestamp,
                    isFullyRead = true
                )
                _readReceipts.emit(readMarkerEvent)
            }

            is MatrixSyncEvent.RoomMembership -> {
                val membershipEvent = RoomMembershipEvent(
                    roomId = event.roomId,
                    userId = event.userId,
                    membership = event.membership,
                    sender = event.event.sender ?: "",
                    timestamp = event.timestamp
                )
                _roomMemberships.emit(membershipEvent)
            }

            is MatrixSyncEvent.InviteReceived -> {
                val membershipEvent = RoomMembershipEvent(
                    roomId = event.roomId,
                    userId = "",  // Current user
                    membership = "invite",
                    sender = event.event.sender ?: "",
                    timestamp = event.timestamp,
                    isInvite = true
                )
                _roomMemberships.emit(membershipEvent)
            }

            is MatrixSyncEvent.CallSignaling -> {
                val callEvent = CallSignalingEvent(
                    roomId = event.roomId,
                    eventType = event.eventType,
                    callId = event.event.content?.get("call_id")?.toString() ?: "",
                    sender = event.event.sender ?: "",
                    content = event.event.content,
                    timestamp = event.timestamp
                )
                _callSignaling.emit(callEvent)
            }

            is MatrixSyncEvent.RoomNameChanged -> {
                val stateEvent = RoomStateChangeEvent(
                    roomId = event.roomId,
                    changeType = "name",
                    newValue = event.name,
                    timestamp = event.timestamp
                )
                _roomStateChanges.emit(stateEvent)
            }

            is MatrixSyncEvent.RoomTopicChanged -> {
                val stateEvent = RoomStateChangeEvent(
                    roomId = event.roomId,
                    changeType = "topic",
                    newValue = event.topic,
                    timestamp = event.timestamp
                )
                _roomStateChanges.emit(stateEvent)
            }

            is MatrixSyncEvent.RoomAvatarChanged -> {
                val stateEvent = RoomStateChangeEvent(
                    roomId = event.roomId,
                    changeType = "avatar",
                    newValue = event.avatarUrl,
                    timestamp = event.timestamp
                )
                _roomStateChanges.emit(stateEvent)
            }

            is MatrixSyncEvent.RoomEncryptionEnabled -> {
                val stateEvent = RoomStateChangeEvent(
                    roomId = event.roomId,
                    changeType = "encryption",
                    newValue = "enabled",
                    timestamp = event.timestamp
                )
                _roomStateChanges.emit(stateEvent)
            }

            is MatrixSyncEvent.SyncError -> {
                val errorEvent = SyncErrorEvent(
                    error = event.error,
                    timestamp = event.timestamp
                )
                _errors.emit(errorEvent)
            }

            // Handle other events as needed
            is MatrixSyncEvent.RoomPowerLevelsChanged -> {
                val stateEvent = RoomStateChangeEvent(
                    roomId = event.roomId,
                    changeType = "power_levels",
                    newValue = null,
                    timestamp = event.timestamp
                )
                _roomStateChanges.emit(stateEvent)
            }

            is MatrixSyncEvent.RoomTagsUpdated -> {
                val stateEvent = RoomStateChangeEvent(
                    roomId = event.roomId,
                    changeType = "tags",
                    newValue = null,
                    timestamp = event.timestamp
                )
                _roomStateChanges.emit(stateEvent)
            }

            is MatrixSyncEvent.ReactionEvent,
            is MatrixSyncEvent.RedactionEvent,
            is MatrixSyncEvent.ToDeviceMessage,
            is MatrixSyncEvent.UnknownEvent,
            // Browser automation events - handled by BrowserCommandHandler
            is MatrixSyncEvent.BrowserCommandEvent,
            is MatrixSyncEvent.BrowserResponseEvent,
            is MatrixSyncEvent.BrowserStatusEvent,
            is MatrixSyncEvent.AgentStatusEvent,
            is MatrixSyncEvent.PiiResponseEvent -> {
                // These are handled but not emitted to specific flows
                logger.logDebug("Received event: ${event::class.simpleName}")
            }
        }
    }
}

// ============================================================================
// Event Data Classes
// ============================================================================

/**
 * Message event
 */
data class MessageEvent(
    val eventId: String,
    val roomId: String,
    val sender: String,
    val eventType: String,
    val content: kotlinx.serialization.json.JsonObject?,
    val timestamp: Long,
    val isEncrypted: Boolean = false
)

/**
 * Typing notification event
 */
data class TypingNotificationEvent(
    val roomId: String,
    val userIds: List<String>,
    val timestamp: Long
)

/**
 * Presence update event
 */
data class PresenceUpdateEvent(
    val userId: String,
    val presence: String,
    val statusMessage: String?,
    val timestamp: Long
)

/**
 * Read receipt event
 */
data class ReadReceiptEvent(
    val roomId: String,
    val eventId: String,
    val userId: String,
    val timestamp: Long,
    val isFullyRead: Boolean = false
)

/**
 * Room membership event
 */
data class RoomMembershipEvent(
    val roomId: String,
    val userId: String,
    val membership: String,
    val sender: String,
    val timestamp: Long,
    val isInvite: Boolean = false
)

/**
 * Call signaling event
 */
data class CallSignalingEvent(
    val roomId: String,
    val eventType: String,
    val callId: String,
    val sender: String,
    val content: kotlinx.serialization.json.JsonObject?,
    val timestamp: Long
)

/**
 * Room state change event
 */
data class RoomStateChangeEvent(
    val roomId: String,
    val changeType: String,
    val newValue: String?,
    val timestamp: Long
)

/**
 * Sync error event
 */
data class SyncErrorEvent(
    val error: Throwable,
    val timestamp: Long
)
