package com.armorclaw.shared.platform.matrix

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

/**
 * Matrix Sync Manager
 *
 * Handles client-side Matrix /sync long-polling since Bridge WebSocket is a stub.
 * This manager directly connects to the Matrix homeserver to receive real-time events.
 *
 * ## Architecture
 * ```
 * MatrixSyncManager
 *      │
 *      ├── HTTP GET /_matrix/client/v3/sync (long-poll)
 *      │
 *      └── Emits events via SharedFlow
 *           │
 *           ├── MatrixSyncEvent.MessageReceived
 *           ├── MatrixSyncEvent.TypingNotification
 *           ├── MatrixSyncEvent.PresenceUpdate
 *           ├── MatrixSyncEvent.ReceiptEvent
 *           ├── MatrixSyncEvent.RoomMembership
 *           └── MatrixSyncEvent.InviteReceived
 * ```
 *
 * ## Usage
 * ```kotlin
 * val syncManager = MatrixSyncManager(homeserverUrl, httpClient)
 * syncManager.startSync(accessToken)
 *
 * syncManager.events.collect { event ->
 *     when (event) {
 *         is MatrixSyncEvent.MessageReceived -> handleMessage(event)
 *         is MatrixSyncEvent.TypingNotification -> showTypingIndicator(event)
 *         // ...
 *     }
 * }
 * ```
 */
class MatrixSyncManager(
    private val homeserverUrl: String,
    private val httpClient: HttpClient,
    private val json: kotlinx.serialization.json.Json = kotlinx.serialization.json.Json {
        ignoreUnknownKeys = true
        isLenient = true
    }
) {
    private val logger = repositoryLogger("MatrixSyncManager", LogTag.Network.Matrix)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // Sync state
    private val _syncState = MutableStateFlow<SyncState>(SyncState.Idle)
    val syncState: StateFlow<SyncState> = _syncState.asStateFlow()

    // Events flow
    private val _events = MutableSharedFlow<MatrixSyncEvent>(extraBufferCapacity = 64)
    val events: SharedFlow<MatrixSyncEvent> = _events.asSharedFlow()

    // Current sync token
    private var sinceToken: String? = null
    private var syncJob: Job? = null
    private var currentAccessToken: String? = null

    /**
     * Start the sync loop
     *
     * @param accessToken The Matrix access token
     * @param initialSince Initial sync token (null for full sync)
     * @param timeout Long-poll timeout in milliseconds
     * @param filter Sync filter ID (optional)
     */
    fun startSync(
        accessToken: String,
        initialSince: String? = null,
        timeout: Long = 30000,
        filter: String? = null,
        context: OperationContext? = null
    ) {
        if (syncJob?.isActive == true) {
            logger.logWarning("Sync already running, ignoring start request")
            return
        }

        currentAccessToken = accessToken
        sinceToken = initialSince
        _syncState.value = SyncState.Connecting

        logger.logOperationStart("startSync", mapOf(
            "homeserver" to homeserverUrl,
            "initial_since" to (initialSince ?: "null"),
            "timeout" to timeout
        ))

        syncJob = scope.launch {
            while (isActive) {
                try {
                    _syncState.value = SyncState.Syncing(sinceToken)

                    val response = performSync(accessToken, timeout, filter)
                    sinceToken = response.nextBatch

                    // Process events from response
                    processSyncResponse(response)

                    _syncState.value = SyncState.Syncing(sinceToken)

                    logger.logDebug("Sync completed", mapOf("next_batch" to sinceToken))

                } catch (e: CancellationException) {
                    logger.logInfo("Sync cancelled")
                    _syncState.value = SyncState.Stopped
                    break
                } catch (e: Exception) {
                    logger.logOperationError("sync", e)
                    _syncState.value = SyncState.Error(e)

                    // Emit error event
                    _events.emit(MatrixSyncEvent.SyncError(e))

                    // Exponential backoff on error
                    delay(5000)
                }
            }
        }
    }

    /**
     * Stop the sync loop
     */
    fun stopSync() {
        logger.logOperationStart("stopSync")
        syncJob?.cancel()
        syncJob = null
        _syncState.value = SyncState.Stopped
        sinceToken = null
        currentAccessToken = null
    }

    /**
     * Perform a single sync request
     */
    private suspend fun performSync(
        accessToken: String,
        timeout: Long,
        filter: String?
    ): MatrixSyncResponseRaw {
        val urlBuilder = StringBuilder("$homeserverUrl/_matrix/client/v3/sync")
        urlBuilder.append("?access_token=$accessToken")
        urlBuilder.append("&timeout=$timeout")

        sinceToken?.let { urlBuilder.append("&since=$it") }
        filter?.let { urlBuilder.append("&filter=$it") }

        val response: HttpResponse = httpClient.get(urlBuilder.toString()) {
            header("Authorization", "Bearer $accessToken")
        }

        if (!response.status.isSuccess()) {
            throw MatrixSyncException("Sync failed: ${response.status}")
        }

        return response.body<MatrixSyncResponseRaw>()
    }

    /**
     * Process sync response and emit events
     */
    private suspend fun processSyncResponse(response: MatrixSyncResponseRaw) {
        // Process joined rooms
        response.rooms?.join?.forEach { (roomId, roomData) ->
            // Timeline events (messages)
            roomData.timeline?.events?.forEach { event ->
                processTimelineEvent(roomId, event)
            }

            // State events (room state changes)
            roomData.state?.events?.forEach { event ->
                processStateEvent(roomId, event)
            }

            // Ephemeral events (typing, receipts)
            roomData.ephemeral?.events?.forEach { event ->
                processEphemeralEvent(roomId, event)
            }

            // Account data
            roomData.accountData?.events?.forEach { event ->
                processAccountDataEvent(roomId, event)
            }
        }

        // Process invited rooms
        response.rooms?.invite?.forEach { (roomId, roomData) ->
            roomData.inviteState?.events?.forEach { event ->
                _events.emit(MatrixSyncEvent.InviteReceived(
                    roomId = roomId,
                    event = event
                ))
            }
        }

        // Process left rooms
        response.rooms?.leave?.forEach { (roomId, roomData) ->
            roomData.state?.events?.forEach { event ->
                if (event.type == "m.room.member") {
                    _events.emit(MatrixSyncEvent.RoomMembership(
                        roomId = roomId,
                        userId = event.content?.get("membership")?.toString() ?: "",
                        membership = "leave",
                        event = event
                    ))
                }
            }
        }

        // Process presence
        response.presence?.events?.forEach { event ->
            _events.emit(MatrixSyncEvent.PresenceUpdate(
                event = event
            ))
        }

        // Process to-device messages
        response.toDevice?.events?.forEach { event ->
            _events.emit(MatrixSyncEvent.ToDeviceMessage(
                event = event
            ))
        }
    }

    private suspend fun processTimelineEvent(roomId: String, event: MatrixEventRaw) {
        when (event.type) {
            "m.room.message", "m.room.encrypted" -> {
                _events.emit(MatrixSyncEvent.MessageReceived(
                    roomId = roomId,
                    event = event
                ))
            }
            "m.room.member" -> {
                val membership = event.content?.get("membership")?.toString()
                _events.emit(MatrixSyncEvent.RoomMembership(
                    roomId = roomId,
                    userId = event.stateKey ?: "",
                    membership = membership ?: "unknown",
                    event = event
                ))
            }
            "m.call.invite", "m.call.candidates", "m.call.answer", "m.call.hangup" -> {
                _events.emit(MatrixSyncEvent.CallSignaling(
                    roomId = roomId,
                    eventType = event.type,
                    event = event
                ))
            }
            "m.reaction" -> {
                _events.emit(MatrixSyncEvent.ReactionEvent(
                    roomId = roomId,
                    event = event
                ))
            }
            "m.room.redaction" -> {
                _events.emit(MatrixSyncEvent.RedactionEvent(
                    roomId = roomId,
                    event = event
                ))
            }
            // ArmorClaw browser automation events
            "com.armorclaw.browser.navigate",
            "com.armorclaw.browser.fill",
            "com.armorclaw.browser.click",
            "com.armorclaw.browser.wait",
            "com.armorclaw.browser.extract",
            "com.armorclaw.browser.screenshot" -> {
                _events.emit(MatrixSyncEvent.BrowserCommandEvent(
                    roomId = roomId,
                    eventType = event.type,
                    event = event
                ))
            }
            "com.armorclaw.browser.response" -> {
                _events.emit(MatrixSyncEvent.BrowserResponseEvent(
                    roomId = roomId,
                    event = event
                ))
            }
            "com.armorclaw.browser.status" -> {
                _events.emit(MatrixSyncEvent.BrowserStatusEvent(
                    roomId = roomId,
                    event = event
                ))
            }
            "com.armorclaw.agent.status" -> {
                _events.emit(MatrixSyncEvent.AgentStatusEvent(
                    roomId = roomId,
                    event = event
                ))
            }
            "com.armorclaw.pii.response" -> {
                _events.emit(MatrixSyncEvent.PiiResponseEvent(
                    roomId = roomId,
                    event = event
                ))
            }
            else -> {
                // Unknown event type, emit as generic
                _events.emit(MatrixSyncEvent.UnknownEvent(
                    roomId = roomId,
                    event = event
                ))
            }
        }
    }

    private suspend fun processStateEvent(roomId: String, event: MatrixEventRaw) {
        when (event.type) {
            "m.room.name" -> {
                _events.emit(MatrixSyncEvent.RoomNameChanged(
                    roomId = roomId,
                    name = event.content?.get("name")?.toString(),
                    event = event
                ))
            }
            "m.room.topic" -> {
                _events.emit(MatrixSyncEvent.RoomTopicChanged(
                    roomId = roomId,
                    topic = event.content?.get("topic")?.toString(),
                    event = event
                ))
            }
            "m.room.avatar" -> {
                _events.emit(MatrixSyncEvent.RoomAvatarChanged(
                    roomId = roomId,
                    avatarUrl = event.content?.get("url")?.toString(),
                    event = event
                ))
            }
            "m.room.encryption" -> {
                _events.emit(MatrixSyncEvent.RoomEncryptionEnabled(
                    roomId = roomId,
                    event = event
                ))
            }
            "m.room.power_levels" -> {
                _events.emit(MatrixSyncEvent.RoomPowerLevelsChanged(
                    roomId = roomId,
                    event = event
                ))
            }
        }
    }

    private suspend fun processEphemeralEvent(roomId: String, event: MatrixEventRaw) {
        when (event.type) {
            "m.typing" -> {
                val userIds = event.content?.get("user_ids")
                _events.emit(MatrixSyncEvent.TypingNotification(
                    roomId = roomId,
                    userIds = when (userIds) {
                        is List<*> -> userIds.filterIsInstance<String>()
                        else -> emptyList()
                    },
                    event = event
                ))
            }
            "m.receipt" -> {
                _events.emit(MatrixSyncEvent.ReceiptEvent(
                    roomId = roomId,
                    event = event
                ))
            }
        }
    }

    private suspend fun processAccountDataEvent(roomId: String, event: MatrixEventRaw) {
        when (event.type) {
            "m.fully_read" -> {
                _events.emit(MatrixSyncEvent.FullyReadMarker(
                    roomId = roomId,
                    eventId = event.content?.get("event_id")?.toString(),
                    event = event
                ))
            }
            "m.tag" -> {
                _events.emit(MatrixSyncEvent.RoomTagsUpdated(
                    roomId = roomId,
                    event = event
                ))
            }
        }
    }

    /**
     * Get current sync token
     */
    fun getSinceToken(): String? = sinceToken

    /**
     * Check if sync is running
     */
    fun isRunning(): Boolean = syncJob?.isActive == true

    /**
     * Cleanup resources
     */
    fun close() {
        stopSync()
        scope.cancel()
    }
}

// ============================================================================
// Sync Events
// ============================================================================

sealed class MatrixSyncEvent {
    abstract val timestamp: Long

    // Messages
    data class MessageReceived(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Typing
    data class TypingNotification(
        val roomId: String,
        val userIds: List<String>,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Presence
    data class PresenceUpdate(
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Receipts
    data class ReceiptEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class FullyReadMarker(
        val roomId: String,
        val eventId: String?,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Room membership
    data class RoomMembership(
        val roomId: String,
        val userId: String,
        val membership: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class InviteReceived(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Room state changes
    data class RoomNameChanged(
        val roomId: String,
        val name: String?,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class RoomTopicChanged(
        val roomId: String,
        val topic: String?,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class RoomAvatarChanged(
        val roomId: String,
        val avatarUrl: String?,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class RoomEncryptionEnabled(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class RoomPowerLevelsChanged(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class RoomTagsUpdated(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Reactions
    data class ReactionEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Redactions
    data class RedactionEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Calls
    data class CallSignaling(
        val roomId: String,
        val eventType: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // To-device messages
    data class ToDeviceMessage(
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Errors
    data class SyncError(
        val error: Throwable,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Browser automation events (ArmorClaw custom events)
    data class BrowserCommandEvent(
        val roomId: String,
        val eventType: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class BrowserResponseEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class BrowserStatusEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class AgentStatusEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class PiiResponseEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Unknown events
    data class UnknownEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()
}

// ============================================================================
// Raw Sync Response Models
// ============================================================================

@Serializable
data class MatrixSyncResponseRaw(
    val nextBatch: String,
    val rooms: MatrixRoomsRaw? = null,
    val presence: MatrixPresenceRaw? = null,
    val accountData: MatrixAccountDataRaw? = null,
    val toDevice: MatrixToDeviceRaw? = null,
    val deviceLists: MatrixDeviceListsRaw? = null,
    val deviceOneTimeKeysCount: Map<String, Int>? = null
)

@Serializable
data class MatrixRoomsRaw(
    val join: Map<String, MatrixJoinedRoomRaw>? = null,
    val invite: Map<String, MatrixInvitedRoomRaw>? = null,
    val leave: Map<String, MatrixLeftRoomRaw>? = null
)

@Serializable
data class MatrixJoinedRoomRaw(
    val timeline: MatrixTimelineRaw? = null,
    val state: MatrixStateRaw? = null,
    val ephemeral: MatrixEphemeralRaw? = null,
    val accountData: MatrixAccountDataRaw? = null,
    val unreadNotifications: MatrixNotificationsRaw? = null,
    val summary: MatrixRoomSummaryRaw? = null
)

@Serializable
data class MatrixTimelineRaw(
    val events: List<MatrixEventRaw>? = null,
    val limited: Boolean? = null,
    val prevBatch: String? = null
)

@Serializable
data class MatrixStateRaw(
    val events: List<MatrixEventRaw>? = null
)

@Serializable
data class MatrixEphemeralRaw(
    val events: List<MatrixEventRaw>? = null
)

@Serializable
data class MatrixAccountDataRaw(
    val events: List<MatrixEventRaw>? = null
)

@Serializable
data class MatrixNotificationsRaw(
    val highlightCount: Int? = null,
    val notificationCount: Int? = null
)

@Serializable
data class MatrixRoomSummaryRaw(
    val mJoinedMemberCount: Int? = null,
    val mInvitedMemberCount: Int? = null,
    val mHeroes: List<String>? = null
)

@Serializable
data class MatrixInvitedRoomRaw(
    val inviteState: MatrixStateRaw? = null
)

@Serializable
data class MatrixLeftRoomRaw(
    val state: MatrixStateRaw? = null,
    val timeline: MatrixTimelineRaw? = null
)

@Serializable
data class MatrixPresenceRaw(
    val events: List<MatrixEventRaw>? = null
)

@Serializable
data class MatrixToDeviceRaw(
    val events: List<MatrixEventRaw>? = null
)

@Serializable
data class MatrixDeviceListsRaw(
    val changed: List<String>? = null,
    val left: List<String>? = null
)

@Serializable
data class MatrixEventRaw(
    val eventId: String? = null,
    val type: String,
    val content: JsonObject? = null,
    val sender: String? = null,
    val originServerTs: Long? = null,
    val roomId: String? = null,
    val stateKey: String? = null,
    val unsigned: JsonObject? = null,
    val redacts: String? = null
)

// ============================================================================
// Exceptions
// ============================================================================

class MatrixSyncException(message: String) : Exception(message)
