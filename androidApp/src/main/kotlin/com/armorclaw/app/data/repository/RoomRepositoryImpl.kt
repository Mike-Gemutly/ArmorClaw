package com.armorclaw.app.data.repository

import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.domain.repository.RoomRepository
import com.armorclaw.shared.platform.bridge.BridgeEvent
import com.armorclaw.shared.platform.bridge.BridgeRepository
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import com.armorclaw.shared.platform.logging.repositoryOperationSuspend
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock

/**
 * Implementation of RoomRepository for managing chat rooms
 *
 * This repository handles:
 * - Room CRUD operations via BridgeRepository
 * - Room membership management
 * - Unread count tracking
 * - Room observation
 * - Real-time room updates via WebSocket
 *
 * Uses RepositoryLogger for consistent logging with correlation IDs.
 */
class RoomRepositoryImpl(
    private val bridgeRepository: BridgeRepository
) : RoomRepository {

    private val logger = repositoryLogger("RoomRepository", LogTag.Data.RoomRepository)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // In-memory room storage (TODO: Replace with SQLDelight database)
    private val rooms = mutableMapOf<String, Room>()

    // Room updates flow
    private val roomUpdates = MutableSharedFlow<RoomUpdate>(replay = 0)

    // All rooms state for observation
    private val _allRooms = MutableStateFlow<List<Room>>(emptyList())
    val allRooms: StateFlow<List<Room>> = _allRooms

    // Event subscriptions
    private var eventSubscriptionJob: kotlinx.coroutines.Job? = null

    init {
        // Subscribe to bridge events
        subscribeToBridgeEvents()
    }

    /**
     * Subscribe to real-time room events from the bridge
     */
    private fun subscribeToBridgeEvents() {
        eventSubscriptionJob = scope.launch {
            bridgeRepository.getEventFlow().collect { event ->
                when (event) {
                    is BridgeEvent.RoomCreated -> handleRoomCreated(event)
                    is BridgeEvent.RoomMembershipChanged -> handleMembershipChanged(event)
                    is BridgeEvent.MessageReceived -> handleMessageForUnread(event)
                    is BridgeEvent.ReadReceipt -> handleReadReceipt(event)
                    else -> { /* Ignore other events */ }
                }
            }
        }
    }

    /**
     * Handle room created event
     */
    private suspend fun handleRoomCreated(event: BridgeEvent.RoomCreated) {
        val room = Room(
            id = event.roomId,
            name = event.name ?: "Direct Chat",
            avatar = null,
            type = if (event.isDirect) RoomType.DIRECT else RoomType.GROUP,
            membership = Membership.JOIN,
            topic = null,
            isDirect = event.isDirect,
            isFavorite = false,
            isMuted = false,
            unreadCount = 0,
            createdAt = Clock.System.now()
        )

        rooms[event.roomId] = room
        updateAllRoomsState()
        roomUpdates.emit(RoomUpdate.Created(room))

        logger.logOperationSuccess("handleRoomCreated:${event.roomId}")
    }

    /**
     * Handle membership changed event
     */
    private suspend fun handleMembershipChanged(event: BridgeEvent.RoomMembershipChanged) {
        val membership = when (event.membership) {
            "join" -> Membership.JOIN
            "invite" -> Membership.INVITE
            "leave" -> Membership.LEAVE
            "ban" -> Membership.BAN
            else -> Membership.LEAVE
        }

        val existingRoom = rooms[event.roomId]
        if (existingRoom != null) {
            val updatedRoom = existingRoom.copy(membership = membership)
            rooms[event.roomId] = updatedRoom
            updateAllRoomsState()
            roomUpdates.emit(RoomUpdate.MembershipChanged(updatedRoom))
        }

        logger.logOperationSuccess("handleMembershipChanged:${event.roomId}")
    }

    /**
     * Handle incoming message for unread count
     */
    private suspend fun handleMessageForUnread(event: BridgeEvent.MessageReceived) {
        // Don't increment unread for our own messages
        val currentUser = bridgeRepository.getCurrentUser()
        if (currentUser != null && event.senderId == currentUser.id) {
            return
        }

        val room = rooms[event.roomId]
        if (room != null && room.membership == Membership.JOIN && !room.isMuted) {
            val updatedRoom = room.copy(unreadCount = room.unreadCount + 1)
            rooms[event.roomId] = updatedRoom
            updateAllRoomsState()
            roomUpdates.emit(RoomUpdate.Updated(updatedRoom))
        }
    }

    /**
     * Handle read receipt event
     */
    private suspend fun handleReadReceipt(event: BridgeEvent.ReadReceipt) {
        // Only process our own read receipts
        val currentUser = bridgeRepository.getCurrentUser()
        if (currentUser != null && event.userId == currentUser.id) {
            val room = rooms[event.roomId]
            if (room != null && room.unreadCount > 0) {
                val updatedRoom = room.copy(unreadCount = 0)
                rooms[event.roomId] = updatedRoom
                updateAllRoomsState()
                roomUpdates.emit(RoomUpdate.Updated(updatedRoom))
            }
        }
    }

    override suspend fun getRooms(context: OperationContext?): AppResult<List<Room>> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "getRooms",
            context = ctx
        ) {
            val roomList = rooms.values
                .filter { it.membership != Membership.LEAVE }
                .sortedByDescending { it.createdAt }

            logger.logCacheHit("all_rooms")
            roomList
        }
    }

    override suspend fun getRoom(roomId: String, context: OperationContext?): AppResult<Room?> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "getRoom",
            context = ctx.withMetadata(mapOf("roomId" to roomId)),
            errorCode = ArmorClawErrorCode.ROOM_NOT_FOUND
        ) {
            rooms[roomId]
        }
    }

    override suspend fun createRoom(
        name: String,
        isDirect: Boolean,
        context: OperationContext?
    ): AppResult<Room> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "createRoom",
            context = ctx.withMetadata(mapOf("name" to name, "isDirect" to isDirect)),
            errorCode = ArmorClawErrorCode.ROOM_CREATION_FAILED
        ) {
            // Create room via bridge
            when (val result = bridgeRepository.createRoom(name, null, isDirect, null, ctx)) {
                is AppResult.Success -> {
                    val roomId = result.data
                    val room = Room(
                        id = roomId,
                        name = name,
                        avatar = null,
                        type = if (isDirect) RoomType.DIRECT else RoomType.GROUP,
                        membership = Membership.JOIN,
                        topic = null,
                        isDirect = isDirect,
                        isFavorite = false,
                        isMuted = false,
                        unreadCount = 0,
                        createdAt = Clock.System.now()
                    )

                    rooms[roomId] = room
                    updateAllRoomsState()
                    roomUpdates.emit(RoomUpdate.Created(room))

                    logger.logTransformation("create", "room:$roomId")
                    room
                }
                is AppResult.Error -> {
                    throw Exception(result.error.message)
                }
                is AppResult.Loading -> {
                    throw Exception("Unexpected loading state")
                }
            }
        }
    }

    override suspend fun joinRoom(roomId: String, context: OperationContext?): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "joinRoom",
            context = ctx.withMetadata(mapOf("roomId" to roomId)),
            errorCode = ArmorClawErrorCode.ROOM_ACCESS_DENIED
        ) {
            // Join room via bridge
            when (val result = bridgeRepository.joinRoom(roomId, ctx)) {
                is AppResult.Success -> {
                    val joinedRoomId = result.data
                    val room = rooms[joinedRoomId]
                    if (room != null) {
                        val updatedRoom = room.copy(membership = Membership.JOIN)
                        rooms[joinedRoomId] = updatedRoom
                        updateAllRoomsState()
                        roomUpdates.emit(RoomUpdate.MembershipChanged(updatedRoom))
                    } else {
                        // Create room if it doesn't exist
                        val newRoom = Room(
                            id = joinedRoomId,
                            name = "Room $joinedRoomId",
                            avatar = null,
                            type = RoomType.GROUP,
                            membership = Membership.JOIN,
                            topic = null,
                            isDirect = false,
                            isFavorite = false,
                            isMuted = false,
                            unreadCount = 0,
                            createdAt = Clock.System.now()
                        )
                        rooms[joinedRoomId] = newRoom
                        updateAllRoomsState()
                        roomUpdates.emit(RoomUpdate.Created(newRoom))
                    }
                    Unit
                }
                is AppResult.Error -> {
                    throw Exception(result.error.message)
                }
                is AppResult.Loading -> {
                    throw Exception("Unexpected loading state")
                }
            }
        }
    }

    override suspend fun leaveRoom(roomId: String, context: OperationContext?): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "leaveRoom",
            context = ctx.withMetadata(mapOf("roomId" to roomId)),
            errorCode = ArmorClawErrorCode.NOT_MEMBER
        ) {
            // Leave room via bridge
            when (val result = bridgeRepository.leaveRoom(roomId, ctx)) {
                is AppResult.Success -> {
                    val room = rooms[roomId]
                    if (room != null) {
                        val updatedRoom = room.copy(membership = Membership.LEAVE)
                        rooms[roomId] = updatedRoom
                        updateAllRoomsState()
                        roomUpdates.emit(RoomUpdate.MembershipChanged(updatedRoom))
                    }
                    Unit
                }
                is AppResult.Error -> {
                    throw Exception(result.error.message)
                }
                is AppResult.Loading -> {
                    throw Exception("Unexpected loading state")
                }
            }
        }
    }

    override suspend fun inviteUser(
        roomId: String,
        userId: String,
        context: OperationContext?
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "inviteUser",
            context = ctx.withMetadata(mapOf("roomId" to roomId, "userId" to userId)),
            errorCode = ArmorClawErrorCode.INVITE_FAILED
        ) {
            // Invite user via bridge
            when (val result = bridgeRepository.inviteUser(roomId, userId, ctx)) {
                is AppResult.Success -> Unit
                is AppResult.Error -> {
                    throw Exception(result.error.message)
                }
                is AppResult.Loading -> {
                    throw Exception("Unexpected loading state")
                }
            }
        }
    }

    override fun observeRooms(): Flow<List<Room>> {
        return roomUpdates
            .asSharedFlow()
            .map { _ ->
                rooms.values
                    .filter { it.membership != Membership.LEAVE }
                    .sortedByDescending { it.createdAt }
            }
    }

    override fun observeRoom(roomId: String): Flow<Room?> {
        return roomUpdates
            .asSharedFlow()
            .map { update ->
                when (update) {
                    is RoomUpdate.Created -> if (update.room.id == roomId) update.room else rooms[roomId]
                    is RoomUpdate.Updated -> if (update.room.id == roomId) update.room else rooms[roomId]
                    is RoomUpdate.MembershipChanged -> if (update.room.id == roomId) update.room else rooms[roomId]
                    is RoomUpdate.Deleted -> if (update.roomId == roomId) null else rooms[roomId]
                }
            }
    }

    override suspend fun updateLastMessage(
        roomId: String,
        messageId: String,
        context: OperationContext?
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "updateLastMessage",
            context = ctx.withMetadata(mapOf("roomId" to roomId, "messageId" to messageId))
        ) {
            // TODO: Store last message ID when database is implemented
            Unit
        }
    }

    override suspend fun incrementUnreadCount(
        roomId: String,
        context: OperationContext?
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "incrementUnreadCount",
            context = ctx.withMetadata(mapOf("roomId" to roomId))
        ) {
            val room = rooms[roomId]
            if (room != null) {
                val updatedRoom = room.copy(unreadCount = room.unreadCount + 1)
                rooms[roomId] = updatedRoom
                updateAllRoomsState()
                roomUpdates.emit(RoomUpdate.Updated(updatedRoom))
            }
            Unit
        }
    }

    override suspend fun markAsRead(roomId: String, context: OperationContext?): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "markAsRead",
            context = ctx.withMetadata(mapOf("roomId" to roomId))
        ) {
            val room = rooms[roomId]
            if (room != null && room.unreadCount > 0) {
                val updatedRoom = room.copy(unreadCount = 0)
                rooms[roomId] = updatedRoom
                updateAllRoomsState()
                roomUpdates.emit(RoomUpdate.Updated(updatedRoom))

                // Send read receipt via bridge
                // Note: In a real implementation, we'd need the last message ID
                // For now, this is handled by the WebSocket read receipt handling
            }
            Unit
        }
    }

    // Public method to add rooms from server sync
    suspend fun addRoomsFromServer(
        serverRooms: List<Room>,
        context: OperationContext? = null
    ) {
        val ctx = context ?: OperationContext.create()
        logger.logOperationStart(
            "addRoomsFromServer",
            mapOf("count" to serverRooms.size).withContext(ctx)
        )

        for (room in serverRooms) {
            rooms[room.id] = room
        }
        updateAllRoomsState()

        logger.logOperationSuccess("addRoomsFromServer")
    }

    // Public method to update room settings
    suspend fun updateRoomSettings(
        roomId: String,
        isFavorite: Boolean? = null,
        isMuted: Boolean? = null,
        context: OperationContext? = null
    ): AppResult<Room> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "updateRoomSettings",
            context = ctx.withMetadata(mapOf(
                "roomId" to roomId,
                "isFavorite" to isFavorite,
                "isMuted" to isMuted
            ))
        ) {
            val room = rooms[roomId]
                ?: throw IllegalArgumentException("Room not found: $roomId")

            val updatedRoom = room.copy(
                isFavorite = isFavorite ?: room.isFavorite,
                isMuted = isMuted ?: room.isMuted
            )
            rooms[roomId] = updatedRoom
            updateAllRoomsState()
            roomUpdates.emit(RoomUpdate.Updated(updatedRoom))

            updatedRoom
        }
    }

    // Private helper methods
    private fun updateAllRoomsState() {
        _allRooms.value = rooms.values
            .filter { it.membership != Membership.LEAVE }
            .sortedByDescending { it.createdAt }
    }

    private fun generateRoomId(): String {
        return "room_${System.currentTimeMillis()}_${(1000..9999).random()}"
    }

    /**
     * Cleanup resources and cancel all coroutines.
     * Call this when the repository is no longer needed.
     */
    fun cleanup() {
        eventSubscriptionJob?.cancel()
        eventSubscriptionJob = null
        scope.cancel()
    }
}

/**
 * Room update events for observing changes
 */
sealed class RoomUpdate {
    data class Created(val room: Room) : RoomUpdate()
    data class Updated(val room: Room) : RoomUpdate()
    data class MembershipChanged(val room: Room) : RoomUpdate()
    data class Deleted(val roomId: String) : RoomUpdate()
}
