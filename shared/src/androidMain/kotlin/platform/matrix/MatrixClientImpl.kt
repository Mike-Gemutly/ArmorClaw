package com.armorclaw.shared.platform.matrix

import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.model.MessageContent
import com.armorclaw.shared.domain.model.MessageType
import com.armorclaw.shared.domain.model.MessageStatus
import com.armorclaw.shared.domain.model.Room
import com.armorclaw.shared.domain.model.RoomType
import com.armorclaw.shared.domain.model.Membership
import com.armorclaw.shared.domain.model.RoomMember
import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserPresence
import com.armorclaw.shared.platform.matrix.event.MatrixEvent
import com.armorclaw.shared.platform.matrix.event.ArmorClawEventType
import com.armorclaw.shared.platform.logging.LoggerDelegate
import com.armorclaw.shared.platform.logging.LogTag
import io.ktor.client.*
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import kotlinx.datetime.Clock
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject

/**
 * Matrix Client Implementation
 *
 * This implementation provides real Matrix Client-Server API integration
 * using Ktor HttpClient for HTTP calls, * MatrixSyncManager for real-time events
 * MatrixSessionStorage for secure session persistence.
 *
 * ## Architecture
 * ```
 * MatrixClientImpl
 *      │
 *      ├── MatrixApiService (HTTP API)
 *      │
 *      ├── MatrixSyncManager (real-time /sync)
 *      │
 *      └── MatrixSessionStorage (encrypted storage)
 * ```
 *
 * ## Features
 * - Real authentication via login/logout API
 * - Real-time sync via MatrixSyncManager
 * - Secure session persistence via EncryptedSharedPreferences
 * - Full room operations (create, join, leave, invite, kick)
 * - Full message operations (send, edit, redact, react)
 * - Presence, typing, read receipts
 * - Media upload/download
 * - Push notification registration
 */
class MatrixClientImpl(
    private val apiService: MatrixApiService,
    private val sessionStorage: MatrixSessionStorage,
    private val syncManager: MatrixSyncManager,
    private val json: Json,
    private val config: MatrixClientConfig = MatrixClientConfig()
) : MatrixClient {

    private val logger = LoggerDelegate(LogTag.Network.MatrixClient)
    private val clientScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // Internal state
    private var currentSession: MatrixSession? = null
    private var syncJob: Job? = null

    // Room message cache for observeMessages
    private val roomMessages = mutableMapOf<String, MutableList<Message>>()

    // Typing state per room
    private val roomTyping = mutableMapOf<String, List<String>>()

    // Presence state
    private val userPresence = mutableMapOf<String, UserPresence>()

    // ========================================================================
    // State Flows (Reactive)
    // ========================================================================

    private val _syncState = MutableStateFlow<SyncState>(SyncState.Idle)
    override val syncState: StateFlow<SyncState> = _syncState.asStateFlow()

    private val _isLoggedIn = MutableStateFlow(false)
    override val isLoggedIn: StateFlow<Boolean> = _isLoggedIn.asStateFlow()

    private val _currentUser = MutableStateFlow<User?>(null)
    override val currentUser: StateFlow<User?> = _currentUser.asStateFlow()

    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Offline)
    override val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private val _rooms = MutableStateFlow<List<Room>>(emptyList())
    override val rooms: StateFlow<List<Room>> = _rooms.asStateFlow()

    private val _events = MutableSharedFlow<MatrixEvent>(extraBufferCapacity = 64)

    // ========================================================================
    // Authentication
    // ========================================================================

    override suspend fun login(
        homeserver: String,
        username: String,
        password: String,
        deviceId: String?
    ): Result<MatrixSession> {
        logger.logInfo("Logging in to Matrix", mapOf(
            "homeserver" to homeserver,
            "username" to username
        ))

        return try {
            _connectionState.value = ConnectionState.Reconnecting

            val loginResponse = apiService.login(
                homeserver = homeserver,
                username = username,
                password = password,
                deviceId = deviceId,
                initialDeviceDisplayName = "ArmorClaw Android"
            ).getOrElse { error ->
                logger.logError("Login failed", error)
                _connectionState.value = ConnectionState.Error(error)
                return Result.failure(error)
            }

            // Create session with expiration
            val session = MatrixSession.withExpiration(
                userId = loginResponse.userId,
                deviceId = loginResponse.deviceId,
                accessToken = loginResponse.accessToken,
                refreshToken = loginResponse.refreshToken,
                homeserver = homeserver,
                displayName = username,
                avatarUrl = null,
                expiresIn = loginResponse.expiresInMs?.div(1000)
            )

            // Save session
            currentSession = session
            sessionStorage.saveSession(session)

            // Update state
            _isLoggedIn.value = true
            _currentUser.value = User(
                id = session.userId,
                displayName = session.displayName ?: username,
                avatar = session.avatarUrl,
                email = null,
                presence = UserPresence.ONLINE,
                lastActive = Clock.System.now(),
                isVerified = false
            )
            _connectionState.value = ConnectionState.Online

            logger.logInfo("Login successful", mapOf("userId" to session.userId))
            Result.success(session)

        } catch (e: Exception) {
            logger.logError("Login failed", e, mapOf("homeserver" to homeserver))
            _connectionState.value = ConnectionState.Error(e)
            Result.failure(e)
        }
    }

    override suspend fun loginWithWellKnown(
        serverName: String,
        username: String,
        password: String
    ): Result<MatrixSession> {
        logger.logInfo("Discovering homeserver via well-known", mapOf("serverName" to serverName))

        return try {
            // Discover homeserver
            val wellKnown = apiService.discoverServer(serverName).getOrElse { error ->
                logger.logError("Well-known discovery failed", error)
                // Fallback to default pattern
                return login(
                    homeserver = "https://matrix.$serverName",
                    username = username,
                    password = password
                )
            }

            val homeserverUrl = wellKnown.homeserver?.baseUrl
                ?: "https://matrix.$serverName"

            login(homeserverUrl, username, password)
        } catch (e: Exception) {
            logger.logError("Well-known login failed", e)
            Result.failure(e)
        }
    }

    override suspend fun restoreSession(session: MatrixSession): Result<Unit> {
        logger.logInfo("Restoring Matrix session", mapOf("userId" to session.userId))

        return try {
            // Check if session is expired
            if (session.isExpired()) {
                logger.logWarning("Session has expired")
                sessionStorage.clearSession()
                return Result.failure(MatrixAuthException(
                    "Session has expired",
                    "M_UNKNOWN_TOKEN",
                    softLogout = true
                ))
            }

            // Try to refresh if we have a refresh token and session is expiring soon
            if (session.refreshToken != null && session.isExpiringSoon(300)) {
                logger.logInfo("Refreshing token before expiry")
                val refreshResult = apiService.refreshToken(
                    homeserver = session.homeserver,
                    refreshToken = session.refreshToken
                )

                if (refreshResult.isSuccess) {
                    val refreshed = refreshResult.getOrThrow()
                    val newSession = session.copy(
                        accessToken = refreshed.accessToken,
                        refreshToken = refreshed.refreshToken ?: session.refreshToken,
                        expiresIn = refreshed.expiresInMs?.div(1000),
                        expiresAt = refreshed.expiresInMs?.let { System.currentTimeMillis() / 1000 + it / 1000 }
                    )
                    currentSession = newSession
                    sessionStorage.saveSession(newSession)
                }
            }

            this.currentSession = session

            _isLoggedIn.value = true
            _connectionState.value = ConnectionState.Online

            // Fetch current user profile
            val profileResult = apiService.getProfile(
                homeserver = session.homeserver,
                userId = session.userId
            )

            if (profileResult.isSuccess) {
                val profile = profileResult.getOrThrow()
                _currentUser.value = User(
                    id = session.userId,
                    displayName = profile.displayName ?: session.displayName ?: session.userId,
                    avatar = profile.avatarUrl ?: session.avatarUrl,
                    email = null,
                    presence = UserPresence.ONLINE,
                    lastActive = Clock.System.now(),
                    isVerified = false
                )
            } else {
                _currentUser.value = User(
                    id = session.userId,
                    displayName = session.displayName ?: session.userId,
                    avatar = session.avatarUrl,
                    email = null,
                    presence = UserPresence.ONLINE,
                    lastActive = Clock.System.now(),
                    isVerified = false
                )
            }

            logger.logInfo("Session restored successfully")
            Result.success(Unit)

        } catch (e: Exception) {
            logger.logError("Session restore failed", e)
            Result.failure(e)
        }
    }

    override suspend fun logout(): Result<Unit> {
        logger.logInfo("Logging out from Matrix")

        val session = currentSession ?: return Result.success(Unit)

        return try {
            stopSync()

            // Call logout API
            apiService.logout(
                homeserver = session.homeserver,
                accessToken = session.accessToken
            )

            // Clear stored session
            currentSession = null
            sessionStorage.clearSession()

            // Clear all state
            _isLoggedIn.value = false
            _currentUser.value = null
            _rooms.value = emptyList()
            roomMessages.clear()
            roomTyping.clear()
            userPresence.clear()
            _syncState.value = SyncState.Idle
            _connectionState.value = ConnectionState.Offline

            logger.logInfo("Logout successful")
            Result.success(Unit)

        } catch (e: Exception) {
            logger.logError("Logout failed", e)
            // Still clear local state even if API call fails
            currentSession = null
            _isLoggedIn.value = false
            _currentUser.value = null
            _rooms.value = emptyList()
            Result.failure(e)
        }
    }

    // ========================================================================
    // Sync Operations
    // ========================================================================

    override fun startSync() {
        if (syncJob?.isActive == true) {
            logger.logWarning("Sync already running")
            return
        }

        val session = currentSession
        if (session == null) {
            logger.logWarning("Cannot start sync: no active session")
            return
        }

        logger.logInfo("Starting Matrix sync")
        _syncState.value = SyncState.Connecting

        // Start the sync manager
        syncManager.startSync(
            accessToken = session.accessToken,
            initialSince = null,
            timeout = config.syncTimeout,
            filter = null
        )

        // Subscribe to sync events
        syncJob = clientScope.launch {
            syncManager.events.collect { event ->
                handleSyncEvent(event)
            }
        }

        // Observe sync state from manager
        clientScope.launch {
            syncManager.syncState.collect { state ->
                _syncState.value = state
            }
        }

        _connectionState.value = ConnectionState.Online
    }

    private fun handleSyncEvent(event: MatrixSyncEvent) {
        when (event) {
            is MatrixSyncEvent.MessageReceived -> {
                handleMessageReceived(event)
            }
            is MatrixSyncEvent.TypingNotification -> {
                roomTyping[event.roomId] = event.userIds
            }
            is MatrixSyncEvent.RoomMembership -> {
                handleRoomMembership(event)
            }
            is MatrixSyncEvent.RoomNameChanged -> {
                handleRoomNameChanged(event)
            }
            is MatrixSyncEvent.RoomTopicChanged -> {
                handleRoomTopicChanged(event)
            }
            is MatrixSyncEvent.RoomEncryptionEnabled -> {
                handleRoomEncryptionEnabled(event)
            }
            is MatrixSyncEvent.InviteReceived -> {
                handleInviteReceived(event)
            }
            is MatrixSyncEvent.PresenceUpdate -> {
                handlePresenceUpdate(event)
            }
            is MatrixSyncEvent.ReactionEvent -> {
                handleReactionEvent(event)
            }
            is MatrixSyncEvent.RedactionEvent -> {
                handleRedactionEvent(event)
            }
            is MatrixSyncEvent.SyncError -> {
                logger.logError("Sync error", event.error)
                _connectionState.value = ConnectionState.Error(event.error)
            }
            else -> {
                // Other events - log but don't process
                logger.logDebug("Received sync event: ${event::class.simpleName}")
            }
        }
    }

    private fun handleMessageReceived(event: MatrixSyncEvent.MessageReceived) {
        val rawEvent = event.event
        val content = rawEvent.content ?: return

        // Convert to domain Message
        val message = parseMessage(event.roomId, rawEvent)
        if (message != null) {
            // Add to room messages
            roomMessages.getOrPut(event.roomId) { mutableListOf() }
                .add(message)

            // Emit as MatrixEvent
            val matrixEvent = MatrixEvent(
                eventId = message.id,
                roomId = event.roomId,
                senderId = message.senderId,
                type = rawEvent.type,
                content = content.toString(),
                timestamp = message.timestamp.toEpochMilliseconds()
            )
            _events.tryEmit(matrixEvent)

            // Update room's last message
            updateRoomLastMessage(event.roomId, message)
        }
    }

    private fun parseMessage(roomId: String, rawEvent: MatrixEventRaw): Message? {
        val content = rawEvent.content ?: return null
        val senderId = rawEvent.sender ?: return null
        val eventId = rawEvent.eventId ?: return null

        val msgtype = content["msgtype"]?.toString() ?: return null

        val body = content["body"]?.toString() ?: ""

        val messageContent = when (msgtype) {
            "m.text" -> MessageContent(
                type = MessageType.TEXT,
                body = body,
                formattedBody = content["formatted_body"]?.toString()
            )
            "m.emote" -> MessageContent(
                type = MessageType.EMOTE,
                body = body
            )
            "m.notice" -> MessageContent(
                type = MessageType.NOTICE,
                body = body
            )
            "m.image" -> MessageContent(
                type = MessageType.IMAGE,
                body = body,
                attachments = parseAttachments(content)
            )
            "m.file" -> MessageContent(
                type = MessageType.FILE,
                body = body,
                attachments = parseAttachments(content)
            )
            "m.audio" -> MessageContent(
                type = MessageType.AUDIO,
                body = body,
                attachments = parseAttachments(content)
            )
            "m.video" -> MessageContent(
                type = MessageType.VIDEO,
                body = body,
                attachments = parseAttachments(content)
            )
            else -> MessageContent(
                type = MessageType.TEXT,
                body = body
            )
        }

        val isOutgoing = senderId == currentSession?.userId

        return Message(
            id = eventId,
            roomId = roomId,
            senderId = senderId,
            content = messageContent,
            timestamp = kotlinx.datetime.Instant.fromEpochMilliseconds(rawEvent.originServerTs ?: System.currentTimeMillis()),
            isOutgoing = isOutgoing,
            status = if (isOutgoing) MessageStatus.SYNCED else MessageStatus.SENT
        )
    }

    private fun parseAttachments(content: JsonObject): List<com.armorclaw.shared.domain.model.Attachment> {
        val attachments = mutableListOf<com.armorclaw.shared.domain.model.Attachment>()
        val url = content["url"]?.toString()
        if (url != null) {
            attachments.add(com.armorclaw.shared.domain.model.Attachment(
                url = url,
                mimeType = content["info"]?.let { (it as? JsonObject)?.get("mimetype")?.toString() } ?: "application/octet-stream",
                size = content["info"]?.let { (it as? JsonObject)?.get("size")?.toString()?.toLongOrNull() } ?: 0L,
                thumbnailUrl = content["info"]?.let { (it as? JsonObject)?.get("thumbnail_url")?.toString() },
                fileName = content["filename"]?.toString() ?: "file"
            ))
        }
        return attachments
    }

    private fun updateRoomLastMessage(roomId: String, message: Message) {
        _rooms.update { rooms ->
            rooms.map { room ->
                if (room.id == roomId) {
                    room.copy(
                        lastMessage = com.armorclaw.shared.domain.model.MessageSummary(
                            id = message.id,
                            content = message.content.body,
                            senderId = message.senderId,
                            timestamp = message.timestamp,
                            isOutgoing = message.isOutgoing
                        )
                    )
                } else {
                    room
                }
            }
        }
    }

    private fun handleRoomMembership(event: MatrixSyncEvent.RoomMembership) {
        // Update room members when membership changes
        logger.logDebug("Room membership event", mapOf(
            "roomId" to event.roomId,
            "userId" to event.userId,
            "membership" to event.membership
        ))
    }

    private fun handleRoomNameChanged(event: MatrixSyncEvent.RoomNameChanged) {
        _rooms.update { rooms ->
            rooms.map { room ->
                if (room.id == event.roomId) {
                    room.copy(name = event.name ?: room.name)
                } else {
                    room
                }
            }
        }
    }

    private fun handleRoomTopicChanged(event: MatrixSyncEvent.RoomTopicChanged) {
        _rooms.update { rooms ->
            rooms.map { room ->
                if (room.id == event.roomId) {
                    room.copy(topic = event.topic)
                } else {
                    room
                }
            }
        }
    }

    private fun handleRoomEncryptionEnabled(event: MatrixSyncEvent.RoomEncryptionEnabled) {
        logger.logInfo("Room encryption enabled", mapOf("roomId" to event.roomId))
    }

    private fun handleInviteReceived(event: MatrixSyncEvent.InviteReceived) {
        logger.logInfo("Received room invite", mapOf("roomId" to event.roomId))
        // TODO: Auto-join if configured
    }

    private fun handlePresenceUpdate(event: MatrixSyncEvent.PresenceUpdate) {
        val rawEvent = event.event
        val userId = rawEvent.sender ?: return
        val presenceStr = rawEvent.content?.get("presence")?.toString() ?: "offline"

        val presence = when (presenceStr.lowercase()) {
            "online" -> UserPresence.ONLINE
            "unavailable" -> UserPresence.UNAVAILABLE
            else -> UserPresence.OFFLINE
        }

        userPresence[userId] = presence
    }

    private fun handleReactionEvent(event: MatrixSyncEvent.ReactionEvent) {
        logger.logDebug("Reaction event received", mapOf("roomId" to event.roomId))
        // TODO: Update message reactions
    }

    private fun handleRedactionEvent(event: MatrixSyncEvent.RedactionEvent) {
        logger.logDebug("Redaction event received", mapOf("roomId" to event.roomId))
        // TODO: Mark messages as redacted
    }

    override fun stopSync() {
        logger.logInfo("Stopping Matrix sync")
        syncJob?.cancel()
        syncJob = null
        syncManager.stopSync()
        _syncState.value = SyncState.Stopped
    }

    override suspend fun syncOnce(): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("No active session")
        )

        return try {
            // The sync manager handles this - just ensure it's running
            if (!syncManager.isRunning()) {
                startSync()
            }
            Result.success(Unit)
        } catch (e: Exception) {
            logger.logError("One-time sync failed", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Room Operations
    // ========================================================================

    override suspend fun getRoom(roomId: String): Room? {
        return _rooms.value.find { it.id == roomId }
    }

    override fun observeRoom(roomId: String): Flow<Room> {
        return _rooms.map { rooms ->
            rooms.find { it.id == roomId }
        }.filterNotNull()
    }

    override suspend fun createRoom(
        name: String?,
        topic: String?,
        isDirect: Boolean,
        invite: List<String>,
        isEncrypted: Boolean
    ): Result<Room> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logInfo("Creating room", mapOf(
            "name" to (name ?: "unnamed"),
            "isDirect" to isDirect,
            "isEncrypted" to isEncrypted
        ))

        return try {
            // Build initial state for encryption if needed
            val initialState = if (isEncrypted) {
                listOf(
                    InitialStateEvent(
                        type = "m.room.encryption",
                        stateKey = "",
                        content = kotlinx.serialization.json.JsonObject(
                            mapOf("algorithm" to kotlinx.serialization.json.JsonPrimitive("m.megolm.v1.aes-sha2"))
                        )
                    )
                )
            } else null

            val request = CreateRoomRequest(
                name = name,
                topic = topic,
                preset = if (isDirect) RoomPreset.TRUSTED_PRIVATE_CHAT else RoomPreset.PRIVATE_CHAT,
                isDirect = isDirect,
                invite = invite.ifEmpty { null },
                initialState = initialState
            )

            val response = apiService.createRoom(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                request = request
            ).getOrElse { error ->
                return@createRoom Result.failure(error)
            }

            val room = Room(
                id = response.roomId,
                name = name ?: "New Room",
                avatar = null,
                type = if (isDirect) RoomType.DIRECT else RoomType.GROUP,
                membership = Membership.JOIN,
                topic = topic,
                isDirect = isDirect,
                unreadCount = 0,
                lastMessage = null,
                members = emptyList(),
                createdAt = Clock.System.now()
            )

            _rooms.update { it + room }

            logger.logInfo("Room created", mapOf("roomId" to response.roomId))
            Result.success(room)

        } catch (e: Exception) {
            logger.logError("Failed to create room", e)
            Result.failure(e)
        }
    }

    override suspend fun joinRoom(roomIdOrAlias: String): Result<Room> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logInfo("Joining room", mapOf("roomId" to roomIdOrAlias))

        return try {
            val response = apiService.joinRoom(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomIdOrAlias = roomIdOrAlias
            ).getOrElse { error ->
                return@joinRoom Result.failure(error)
            }

            // Create room object - we'll get full details from sync
            val room = Room(
                id = response.roomId,
                name = roomIdOrAlias,
                avatar = null,
                type = RoomType.GROUP,
                membership = Membership.JOIN,
                topic = null,
                isDirect = false,
                unreadCount = 0,
                lastMessage = null,
                members = emptyList(),
                createdAt = Clock.System.now()
            )

            _rooms.update { rooms ->
                if (rooms.any { it.id == response.roomId }) {
                    rooms.map { if (it.id == response.roomId) it.copy(membership = Membership.JOIN) else it }
                } else {
                    rooms + room
                }
            }

            logger.logInfo("Joined room", mapOf("roomId" to response.roomId))
            Result.success(room)

        } catch (e: Exception) {
            logger.logError("Failed to join room", e)
            Result.failure(e)
        }
    }

    override suspend fun leaveRoom(roomId: String): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logInfo("Leaving room", mapOf("roomId" to roomId))

        return try {
            apiService.leaveRoom(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId
            ).getOrElse { error ->
                return@leaveRoom Result.failure(error)
            }

            // Update local state
            _rooms.update { rooms -> rooms.filterNot { it.id == roomId } }
            roomMessages.remove(roomId)
            roomTyping.remove(roomId)

            logger.logInfo("Left room", mapOf("roomId" to roomId))
            Result.success(Unit)

        } catch (e: Exception) {
            logger.logError("Failed to leave room", e)
            Result.failure(e)
        }
    }

    override suspend fun inviteUser(roomId: String, userId: String): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logInfo("Inviting user to room", mapOf(
            "roomId" to roomId,
            "userId" to userId
        ))

        return try {
            apiService.inviteUser(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                userId = userId
            )
        } catch (e: Exception) {
            logger.logError("Failed to invite user", e)
            Result.failure(e)
        }
    }

    override suspend fun kickUser(roomId: String, userId: String, reason: String?): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logInfo("Kicking user from room", mapOf(
            "roomId" to roomId,
            "userId" to userId,
            "reason" to (reason ?: "no reason")
        ))

        return try {
            apiService.kickUser(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                userId = userId,
                reason = reason
            )
        } catch (e: Exception) {
            logger.logError("Failed to kick user", e)
            Result.failure(e)
        }
    }

    override suspend fun setRoomName(roomId: String, name: String): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            apiService.sendStateEvent(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventType = "m.room.name",
                stateKey = "",
                content = RoomNameContent(name = name)
            )

            _rooms.update { rooms ->
                rooms.map { room ->
                    if (room.id == roomId) room.copy(name = name) else room
                }
            }

            Result.success(Unit)
        } catch (e: Exception) {
            logger.logError("Failed to set room name", e)
            Result.failure(e)
        }
    }

    override suspend fun setRoomTopic(roomId: String, topic: String): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            apiService.sendStateEvent(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventType = "m.room.topic",
                stateKey = "",
                content = RoomTopicContent(topic = topic)
            )

            _rooms.update { rooms ->
                rooms.map { room ->
                    if (room.id == roomId) room.copy(topic = topic) else room
                }
            }

            Result.success(Unit)
        } catch (e: Exception) {
            logger.logError("Failed to set room topic", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Message Operations
    // ========================================================================

    override suspend fun getMessages(
        roomId: String,
        limit: Int,
        fromToken: String?
    ): Result<MessageBatch> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logDebug("Getting messages", mapOf(
            "roomId" to roomId,
            "limit" to limit
        ))

        return try {
            val response = apiService.getMessages(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                from = fromToken,
                limit = limit
            ).getOrElse { error ->
                return@getMessages Result.failure(error)
            }

            val messages = response.chunk.mapNotNull { rawEvent ->
                parseMessage(roomId, rawEvent)
            }

            // Cache messages
            roomMessages.getOrPut(roomId) { mutableListOf() }.addAll(messages)

            Result.success(MessageBatch(
                messages = messages,
                nextToken = response.end,
                prevToken = response.start
            ))

        } catch (e: Exception) {
            logger.logError("Failed to get messages", e)
            Result.failure(e)
        }
    }

    override fun observeMessages(roomId: String): Flow<List<Message>> {
        return flow {
            // Emit cached messages first
            roomMessages[roomId]?.let { emit(it.toList()) }

            // Then observe updates from sync
            // This is simplified - in a real implementation you'd
            // track individual message additions
        }
    }

    override suspend fun sendTextMessage(
        roomId: String,
        text: String,
        html: String?
    ): Result<String> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logInfo("Sending text message", mapOf(
            "roomId" to roomId,
            "textLength" to text.length
        ))

        return try {
            val content = RoomMessageContent(
                msgtype = "m.text",
                body = text,
                formattedBody = html,
                format = if (html != null) "org.matrix.custom.html" else null
            )

            val response = apiService.sendMessage(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventType = "m.room.message",
                content = content
            ).getOrElse { error ->
                return@sendTextMessage Result.failure(error)
            }

            logger.logInfo("Message sent", mapOf("eventId" to response.eventId))
            Result.success(response.eventId)

        } catch (e: Exception) {
            logger.logError("Failed to send message", e)
            Result.failure(e)
        }
    }

    override suspend fun sendEmote(roomId: String, text: String): Result<String> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val content = RoomMessageContent(
                msgtype = "m.emote",
                body = text
            )

            val response = apiService.sendMessage(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventType = "m.room.message",
                content = content
            ).getOrElse { error ->
                return@sendEmote Result.failure(error)
            }

            Result.success(response.eventId)

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun sendReply(
        roomId: String,
        replyToEventId: String,
        text: String
    ): Result<String> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val content = RoomMessageContent(
                msgtype = "m.text",
                body = text,
                relatesTo = RelationInfo(
                    inReplyTo = InReplyToInfo(eventId = replyToEventId)
                )
            )

            val response = apiService.sendMessage(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventType = "m.room.message",
                content = content
            ).getOrElse { error ->
                return@sendReply Result.failure(error)
            }

            Result.success(response.eventId)

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun editMessage(
        roomId: String,
        eventId: String,
        newText: String
    ): Result<String> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val content = RoomMessageContent(
                msgtype = "m.text",
                body = "* $newText",
                relatesTo = RelationInfo(
                    relType = "m.replace",
                    eventId = eventId
                ),
                newContent = NewContentInfo(
                    msgtype = "m.text",
                    body = newText
                )
            )

            val response = apiService.sendMessage(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventType = "m.room.message",
                content = content
            ).getOrElse { error ->
                return@editMessage Result.failure(error)
            }

            Result.success(response.eventId)

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun redactMessage(
        roomId: String,
        eventId: String,
        reason: String?
    ): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            apiService.redactEvent(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventId = eventId,
                reason = reason
            ).map { }
        } catch (e: Exception) {
            logger.logError("Failed to redact message", e)
            Result.failure(e)
        }
    }

    override suspend fun sendReaction(
        roomId: String,
        eventId: String,
        key: String
    ): Result<String> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val content = ReactionContent(
                relatesTo = ReactionRelationInfo(
                    eventId = eventId,
                    key = key
                )
            )

            val response = apiService.sendMessage(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventType = "m.reaction",
                content = content
            ).getOrElse { error ->
                return@sendReaction Result.failure(error)
            }

            Result.success(response.eventId)

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ========================================================================
    // Event Handling
    // ========================================================================

    override fun observeEvents(): Flow<MatrixEvent> {
        return _events.asSharedFlow()
    }

    override fun observeRoomEvents(roomId: String): Flow<MatrixEvent> {
        return _events.asSharedFlow().filter { it.roomId == roomId }
    }

    override fun observeArmorClawEvents(roomId: String?): Flow<MatrixEvent> {
        return _events.asSharedFlow()
            .filter { ArmorClawEventType.isArmorClawEvent(it.type) }
            .let { flow ->
                if (roomId != null) {
                    flow.filter { it.roomId == roomId }
                } else {
                    flow
                }
            }
    }

    // ========================================================================
    // Presence, Typing, Read Receipts
    // ========================================================================

    override suspend fun setPresence(presence: UserPresence, statusMessage: String?): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val presenceStr = when (presence) {
                UserPresence.ONLINE -> "online"
                UserPresence.UNAVAILABLE -> "unavailable"
                UserPresence.OFFLINE -> "offline"
                UserPresence.UNKNOWN -> "offline"
            }

            apiService.setPresence(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                userId = session.userId,
                presence = presenceStr,
                statusMessage = statusMessage
            )
        } catch (e: Exception) {
            logger.logError("Failed to set presence", e)
            Result.failure(e)
        }
    }

    override suspend fun getUserPresence(userId: String): Result<UserPresence> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val response = apiService.getPresence(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                userId = userId
            ).getOrElse { error ->
                return@getUserPresence Result.failure(error)
            }

            val presence = when (response.presence.lowercase()) {
                "online" -> UserPresence.ONLINE
                "unavailable" -> UserPresence.UNAVAILABLE
                else -> UserPresence.OFFLINE
            }

            Result.success(presence)

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override fun observePresence(): Flow<PresenceUpdate> = flow {
        // Emit from userPresence map changes
        userPresence.forEach { (userId, presence) ->
            emit(PresenceUpdate(userId = userId, presence = presence))
        }
    }

    override suspend fun sendTyping(roomId: String, typing: Boolean, timeout: Long): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            apiService.sendTyping(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                userId = session.userId,
                typing = typing,
                timeout = if (typing) timeout else null
            )
        } catch (e: Exception) {
            logger.logError("Failed to send typing notification", e)
            Result.failure(e)
        }
    }

    override fun observeTyping(roomId: String): Flow<List<String>> {
        return flow {
            roomTyping[roomId]?.let { emit(it) }
        }
    }

    override suspend fun sendReadReceipt(roomId: String, eventId: String): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            apiService.sendReadReceipt(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                roomId = roomId,
                eventId = eventId
            )
        } catch (e: Exception) {
            logger.logError("Failed to send read receipt", e)
            Result.failure(e)
        }
    }

    override suspend fun getUnreadCount(roomId: String): Result<UnreadCount> {
        // This would come from the sync response notification_count
        // For now return a placeholder
        return Result.success(UnreadCount(
            notificationCount = 0,
            highlightCount = 0,
            markedUnread = false
        ))
    }

    // ========================================================================
    // Encryption
    // ========================================================================

    override suspend fun isRoomEncrypted(roomId: String): Boolean {
        val room = _rooms.value.find { it.id == roomId } ?: return false
        // Check for encryption state event
        return false  // TODO: Track encryption status from sync
    }

    override fun getRoomEncryptionStatus(roomId: String): Flow<RoomEncryptionStatus> {
        return flowOf(RoomEncryptionStatus.Unencrypted)
    }

    override suspend fun requestVerification(
        userId: String,
        deviceId: String?
    ): Result<VerificationRequest> {
        // E2EE verification requires matrix-rust-sdk
        return Result.failure(NotImplementedError(
            "Device verification requires matrix-rust-sdk integration"
        ))
    }

    override fun observeVerificationRequests(): Flow<VerificationRequest> = emptyFlow()

    // ========================================================================
    // User Operations
    // ========================================================================

    override suspend fun getUser(userId: String): Result<User> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val profile = apiService.getProfile(
                homeserver = session.homeserver,
                userId = userId
            ).getOrElse { error ->
                return@getUser Result.failure(error)
            }

            Result.success(User(
                id = userId,
                displayName = profile.displayName ?: userId,
                avatar = profile.avatarUrl,
                email = null,
                presence = userPresence[userId] ?: UserPresence.UNKNOWN,
                lastActive = Clock.System.now(),
                isVerified = false
            ))

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getDisplayName(userId: String): Result<String?> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val profile = apiService.getProfile(
                homeserver = session.homeserver,
                userId = userId
            ).getOrElse { error ->
                return@getDisplayName Result.failure(error)
            }

            Result.success(profile.displayName)

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun setDisplayName(name: String): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            apiService.setDisplayName(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                userId = session.userId,
                displayName = name
            )

            // Update current user
            _currentUser.update { it?.copy(displayName = name) }

            Result.success(Unit)

        } catch (e: Exception) {
            logger.logError("Failed to set display name", e)
            Result.failure(e)
        }
    }

    override suspend fun getAvatarUrl(userId: String): Result<String?> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            val profile = apiService.getProfile(
                homeserver = session.homeserver,
                userId = userId
            ).getOrElse { error ->
                return@getAvatarUrl Result.failure(error)
            }

            Result.success(profile.avatarUrl)

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun setAvatar(mimeType: String, data: ByteArray): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            // First upload media
            val mxcUrl = apiService.uploadMedia(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                mimeType = mimeType,
                data = data
            ).getOrElse { error ->
                return@setAvatar Result.failure(error)
            }

            // Then set avatar URL
            apiService.setAvatarUrl(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                userId = session.userId,
                avatarUrl = mxcUrl
            )

            // Update current user
            _currentUser.update { it?.copy(avatar = mxcUrl) }

            Result.success(Unit)

        } catch (e: Exception) {
            logger.logError("Failed to set avatar", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Push Notification Operations
    // ========================================================================

    override suspend fun setPusher(
        pushKey: String,
        appId: String,
        appDisplayName: String,
        deviceDisplayName: String,
        pushGatewayUrl: String,
        profileTag: String?
    ): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Cannot set pusher: no active session")
        )

        logger.logInfo("Setting Matrix pusher", mapOf(
            "appId" to appId,
            "pushGatewayUrl" to pushGatewayUrl
        ))

        return try {
            val request = SetPusherRequest(
                pushkey = pushKey,
                kind = "http",
                appId = appId,
                appDisplayName = appDisplayName,
                deviceDisplayName = deviceDisplayName,
                profileTag = profileTag,
                lang = "en",
                data = PusherData(
                    url = pushGatewayUrl,
                    format = "event_id_only"
                ),
                append = false
            )

            apiService.setPusher(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                request = request
            )
        } catch (e: Exception) {
            logger.logError("Failed to set pusher", e)
            Result.failure(e)
        }
    }

    override suspend fun removePusher(
        pushKey: String,
        appId: String
    ): Result<Unit> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Cannot remove pusher: no active session")
        )

        logger.logInfo("Removing Matrix pusher")

        return try {
            val request = SetPusherRequest(
                pushkey = pushKey,
                kind = "",  // Empty kind = delete
                appId = appId,
                appDisplayName = "",
                deviceDisplayName = "",
                profileTag = null,
                lang = "en",
                data = PusherData(url = "", format = null)
            )

            apiService.setPusher(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                request = request
            )
        } catch (e: Exception) {
            logger.logError("Failed to remove pusher", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Media Operations
    // ========================================================================

    override suspend fun uploadMedia(mimeType: String, data: ByteArray): Result<String> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        logger.logInfo("Uploading media", mapOf(
            "mimeType" to mimeType,
            "size" to data.size
        ))

        return try {
            apiService.uploadMedia(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                mimeType = mimeType,
                data = data
            )
        } catch (e: Exception) {
            logger.logError("Failed to upload media", e)
            Result.failure(e)
        }
    }

    override suspend fun downloadMedia(mxcUrl: String): Result<ByteArray> {
        val session = currentSession ?: return Result.failure(
            IllegalStateException("Not logged in")
        )

        return try {
            apiService.downloadMedia(
                homeserver = session.homeserver,
                accessToken = session.accessToken,
                mxcUrl = mxcUrl
            )
        } catch (e: Exception) {
            logger.logError("Failed to download media", e)
            Result.failure(e)
        }
    }

    override fun getThumbnailUrl(mxcUrl: String, width: Int, height: Int): String? {
        val session = currentSession ?: return null
        return apiService.getThumbnailUrl(
            homeserver = session.homeserver,
            mxcUrl = mxcUrl,
            width = width,
            height = height
        )
    }

    // ========================================================================
    // Private Helpers
    // ========================================================================

    private fun normalizeUserId(homeserver: String, username: String): String {
        val serverName = extractServerName(homeserver)
        return if (username.startsWith("@")) {
            username
        } else {
            "@$username:$serverName"
        }
    }

    private fun extractServerName(homeserver: String): String {
        return homeserver
            .removePrefix("https://")
            .removePrefix("http://")
            .split("/").first()
            .split(":").first()
    }

    protected fun persistSession(session: MatrixSession) {
        clientScope.launch {
            sessionStorage.saveSession(session)
        }
    }

    protected fun clearSession() {
        clientScope.launch {
            sessionStorage.clearSession()
        }
    }
}
