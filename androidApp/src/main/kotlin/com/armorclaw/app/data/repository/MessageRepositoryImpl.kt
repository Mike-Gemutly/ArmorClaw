package com.armorclaw.app.data.repository

import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.domain.repository.MessageRepository
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
import kotlinx.datetime.Instant

/**
 * Implementation of MessageRepository for managing messages
 *
 * This repository handles:
 * - Message CRUD operations via BridgeRepository
 * - Local caching for offline support
 * - Offline message queue
 * - Message status tracking
 * - Thread support
 *
 * Uses RepositoryLogger for consistent logging with correlation IDs.
 */
class MessageRepositoryImpl(
    private val bridgeRepository: BridgeRepository
) : MessageRepository {

    private val logger = repositoryLogger("MessageRepository", LogTag.Data.MessageRepository)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // In-memory message storage (TODO: Replace with SQLDelight database)
    private val messagesByRoom = mutableMapOf<String, MutableList<Message>>()

    // Message flow for observing changes
    private val messageUpdates = MutableSharedFlow<MessageUpdate>(replay = 0)

    // Offline message queue
    private val _offlineQueue = MutableStateFlow<List<Message>>(emptyList())
    val offlineQueue: StateFlow<List<Message>> = _offlineQueue

    // Event subscriptions
    private var eventSubscriptionJob: kotlinx.coroutines.Job? = null

    init {
        // Subscribe to bridge events
        subscribeToBridgeEvents()
    }

    /**
     * Subscribe to real-time message events from the bridge
     */
    private fun subscribeToBridgeEvents() {
        eventSubscriptionJob = scope.launch {
            bridgeRepository.getEventFlow().collect { event ->
                when (event) {
                    is BridgeEvent.MessageReceived -> handleIncomingMessage(event)
                    is BridgeEvent.MessageStatusUpdated -> handleMessageStatusUpdate(event)
                    else -> { /* Ignore other events */ }
                }
            }
        }
    }

    /**
     * Handle incoming message from WebSocket
     */
    private suspend fun handleIncomingMessage(event: BridgeEvent.MessageReceived) {
        val message = mapBridgeEventToMessage(event)
        addMessagesFromServer(event.roomId, listOf(message))

        logger.logOperationSuccess("handleIncomingMessage:${event.eventId}")
    }

    /**
     * Handle message status update from WebSocket
     */
    private suspend fun handleMessageStatusUpdate(event: BridgeEvent.MessageStatusUpdated) {
        val status = when (event.status) {
            "sent" -> MessageStatus.SENT
            "delivered" -> MessageStatus.DELIVERED
            "read" -> MessageStatus.READ
            "failed" -> MessageStatus.FAILED
            else -> MessageStatus.SENDING
        }

        updateMessageStatus(event.eventId, status)
    }

    override suspend fun getMessages(
        roomId: String,
        limit: Int,
        offset: Int,
        context: OperationContext?
    ): AppResult<List<Message>> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "getMessages",
            context = ctx.withMetadata(mapOf("roomId" to roomId)),
            errorCode = ArmorClawErrorCode.MESSAGE_NOT_FOUND
        ) {
            val roomMessages = messagesByRoom[roomId] ?: emptyList()
            val paginatedMessages = roomMessages
                .sortedByDescending { it.timestamp }
                .drop(offset)
                .take(limit)

            logger.logCacheHit("room:$roomId:offset:$offset")
            paginatedMessages
        }
    }

    override suspend fun getMessage(
        roomId: String,
        messageId: String,
        context: OperationContext?
    ): AppResult<Message?> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "getMessage",
            context = ctx.withMetadata(mapOf("roomId" to roomId, "messageId" to messageId)),
            errorCode = ArmorClawErrorCode.MESSAGE_NOT_FOUND
        ) {
            messagesByRoom[roomId]?.find { it.id == messageId }
        }
    }

    override suspend fun sendMessage(
        roomId: String,
        content: MessageContent,
        context: OperationContext?
    ): AppResult<Message> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "sendMessage",
            context = ctx.withMetadata(mapOf("roomId" to roomId, "contentType" to content.type.name)),
            errorCode = ArmorClawErrorCode.MESSAGE_SEND_FAILED
        ) {
            val tempId = generateMessageId()
            val currentUserId = bridgeRepository.getCurrentUser()?.id ?: "user"

            // Create optimistic message
            val message = Message(
                id = tempId,
                roomId = roomId,
                senderId = currentUserId,
                content = content,
                timestamp = Clock.System.now(),
                isOutgoing = true,
                status = MessageStatus.SENDING
            )

            // Add to local storage optimistically
            messagesByRoom.getOrPut(roomId) { mutableListOf() }.add(0, message)

            // Add to offline queue for retry if needed
            addToOfflineQueue(message)

            // Emit update
            messageUpdates.emit(MessageUpdate.Created(message))

            // Subscribe to room if not already subscribed
            bridgeRepository.subscribeToRoom(roomId, ctx)

            // Send via bridge
            scope.launch {
                when (val result = bridgeRepository.sendMessage(roomId, content, ctx)) {
                    is AppResult.Success -> {
                        val (eventId, _) = result.data
                        // Update message with real event ID and status
                        updateMessageAfterSend(tempId, eventId, MessageStatus.SENT)
                    }
                    is AppResult.Error -> {
                        // Update status to failed
                        updateMessageStatus(tempId, MessageStatus.FAILED)
                        logger.logOperationError("sendMessage:bridge", Exception(result.error.message))
                    }
                    is AppResult.Loading -> { /* Shouldn't happen */ }
                }
            }

            logger.logTransformation("create", "message:${message.id}")
            message
        }
    }

    override suspend fun editMessage(
        roomId: String,
        messageId: String,
        content: MessageContent,
        context: OperationContext?
    ): AppResult<Message> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "editMessage",
            context = ctx.withMetadata(mapOf("roomId" to roomId, "messageId" to messageId)),
            errorCode = ArmorClawErrorCode.MESSAGE_EDIT_FAILED
        ) {
            val roomMessages = messagesByRoom[roomId]
                ?: throw IllegalArgumentException("Room not found: $roomId")

            val messageIndex = roomMessages.indexOfFirst { it.id == messageId }
            if (messageIndex == -1) {
                throw IllegalArgumentException("Message not found: $messageId")
            }

            val existingMessage = roomMessages[messageIndex]
            val editedMessage = existingMessage.copy(
                content = content,
                editCount = existingMessage.editCount + 1
            )

            roomMessages[messageIndex] = editedMessage
            messageUpdates.emit(MessageUpdate.Edited(editedMessage))

            // TODO: Send edit via bridge when API supports it

            editedMessage
        }
    }

    override suspend fun deleteMessage(
        roomId: String,
        messageId: String,
        context: OperationContext?
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "deleteMessage",
            context = ctx.withMetadata(mapOf("roomId" to roomId, "messageId" to messageId)),
            errorCode = ArmorClawErrorCode.MESSAGE_DELETE_FAILED
        ) {
            val roomMessages = messagesByRoom[roomId]
                ?: throw IllegalArgumentException("Room not found: $roomId")

            val messageIndex = roomMessages.indexOfFirst { it.id == messageId }
            if (messageIndex == -1) {
                throw IllegalArgumentException("Message not found: $messageId")
            }

            val deletedMessage = roomMessages[messageIndex].copy(isDeleted = true)
            roomMessages[messageIndex] = deletedMessage
            messageUpdates.emit(MessageUpdate.Deleted(messageId))

            // TODO: Send redaction via bridge when API supports it

            Unit
        }
    }

    override suspend fun retryMessage(
        messageId: String,
        context: OperationContext?
    ): AppResult<Message> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "retryMessage",
            context = ctx.withMetadata(mapOf("messageId" to messageId)),
            errorCode = ArmorClawErrorCode.MESSAGE_SEND_FAILED
        ) {
            // Find message in offline queue
            val message = _offlineQueue.value.find { it.id == messageId }
                ?: throw IllegalArgumentException("Message not in offline queue: $messageId")

            // Update status to sending
            val updatedMessage = message.copy(status = MessageStatus.SENDING)
            updateMessageInQueue(message, updatedMessage)

            // Retry via bridge
            scope.launch {
                when (val result = bridgeRepository.sendMessage(message.roomId, message.content, ctx)) {
                    is AppResult.Success -> {
                        val (eventId, _) = result.data
                        updateMessageAfterSend(messageId, eventId, MessageStatus.SENT)
                        removeFromOfflineQueue(messageId)
                    }
                    is AppResult.Error -> {
                        updateMessageStatus(messageId, MessageStatus.FAILED)
                    }
                    is AppResult.Loading -> { }
                }
            }

            // Emit update
            messageUpdates.emit(MessageUpdate.StatusChanged(updatedMessage))

            updatedMessage
        }
    }

    override fun observeMessages(roomId: String): Flow<List<Message>> {
        return messageUpdates
            .asSharedFlow()
            .map { update ->
                when (update) {
                    is MessageUpdate.Created -> {
                        if (update.message.roomId == roomId) {
                            messagesByRoom[roomId] ?: emptyList()
                        } else {
                            messagesByRoom[roomId] ?: emptyList()
                        }
                    }
                    is MessageUpdate.Edited -> messagesByRoom[roomId] ?: emptyList()
                    is MessageUpdate.Deleted -> messagesByRoom[roomId] ?: emptyList()
                    is MessageUpdate.StatusChanged -> messagesByRoom[roomId] ?: emptyList()
                }
            }
    }

    override fun observeMessage(roomId: String, messageId: String): Flow<Message?> {
        return messageUpdates
            .asSharedFlow()
            .map { update ->
                when (update) {
                    is MessageUpdate.Created -> if (update.message.roomId == roomId && update.message.id == messageId) update.message else null
                    is MessageUpdate.Edited -> if (update.message.roomId == roomId && update.message.id == messageId) update.message else null
                    is MessageUpdate.Deleted -> if (update.messageId == messageId) null else null
                    is MessageUpdate.StatusChanged -> if (update.message.roomId == roomId && update.message.id == messageId) update.message else null
                }
            }
    }

    override suspend fun clearOfflineMessages(context: OperationContext?): AppResult<Int> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "clearOfflineMessages",
            context = ctx
        ) {
            val count = _offlineQueue.value.size
            _offlineQueue.value = emptyList()
            count
        }
    }

    // Public method to update message status (used by event handler)
    suspend fun updateMessageStatus(
        messageId: String,
        status: MessageStatus,
        context: OperationContext? = null
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        return repositoryOperationSuspend(
            logger = logger,
            operation = "updateMessageStatus",
            context = ctx.withMetadata(mapOf("messageId" to messageId, "status" to status.name)),
            errorCode = ArmorClawErrorCode.SYNC_CONFLICT
        ) {
            for ((roomId, messages) in messagesByRoom) {
                val messageIndex = messages.indexOfFirst { it.id == messageId }
                if (messageIndex != -1) {
                    val updatedMessage = messages[messageIndex].copy(status = status)
                    messages[messageIndex] = updatedMessage

                    // Remove from offline queue if synced
                    if (status == MessageStatus.SYNCED || status == MessageStatus.SENT) {
                        removeFromOfflineQueue(messageId)
                    }

                    messageUpdates.emit(MessageUpdate.StatusChanged(updatedMessage))
                    return@repositoryOperationSuspend Unit
                }
            }

            Unit
        }
    }

    // Public method to add messages from server sync
    suspend fun addMessagesFromServer(
        roomId: String,
        messages: List<Message>,
        context: OperationContext? = null
    ) {
        val ctx = context ?: OperationContext.create()
        logger.logOperationStart(
            "addMessagesFromServer",
            mapOf("roomId" to roomId, "count" to messages.size).withContext(ctx)
        )

        val roomMessages = messagesByRoom.getOrPut(roomId) { mutableListOf() }
        for (message in messages) {
            val existingIndex = roomMessages.indexOfFirst { it.id == message.id }
            if (existingIndex == -1) {
                roomMessages.add(message)
            } else {
                // Update existing with server version
                roomMessages[existingIndex] = message.copy(status = MessageStatus.SYNCED)
            }
        }

        // Sort by timestamp
        roomMessages.sortByDescending { it.timestamp }

        logger.logOperationSuccess("addMessagesFromServer")
    }

    // Private helper methods

    private suspend fun updateMessageAfterSend(tempId: String, eventId: String, status: MessageStatus) {
        for ((roomId, messages) in messagesByRoom) {
            val messageIndex = messages.indexOfFirst { it.id == tempId }
            if (messageIndex != -1) {
                val existing = messages[messageIndex]
                val updatedMessage = existing.copy(
                    id = eventId, // Replace temp ID with real event ID
                    status = status
                )
                messages[messageIndex] = updatedMessage
                removeFromOfflineQueue(tempId)
                messageUpdates.emit(MessageUpdate.StatusChanged(updatedMessage))
                return
            }
        }
    }

    private fun addToOfflineQueue(message: Message) {
        val currentQueue = _offlineQueue.value.toMutableList()
        currentQueue.add(message)
        _offlineQueue.value = currentQueue
        logger.logCacheMiss("offline:${message.id}")
    }

    private fun removeFromOfflineQueue(messageId: String) {
        val currentQueue = _offlineQueue.value.toMutableList()
        currentQueue.removeAll { it.id == messageId }
        _offlineQueue.value = currentQueue
    }

    private fun updateMessageInQueue(oldMessage: Message, newMessage: Message) {
        val currentQueue = _offlineQueue.value.toMutableList()
        val index = currentQueue.indexOf(oldMessage)
        if (index != -1) {
            currentQueue[index] = newMessage
            _offlineQueue.value = currentQueue
        }
    }

    private fun generateMessageId(): String {
        return "msg_${System.currentTimeMillis()}_${(1000..9999).random()}"
    }

    private fun mapBridgeEventToMessage(event: BridgeEvent.MessageReceived): Message {
        val messageType = when (event.content.type) {
            "m.text" -> MessageType.TEXT
            "m.image" -> MessageType.IMAGE
            "m.video" -> MessageType.VIDEO
            "m.audio" -> MessageType.AUDIO
            "m.file" -> MessageType.FILE
            "m.location" -> MessageType.TEXT // Location treated as text with geo URI
            "m.notice" -> MessageType.NOTICE
            "m.emote" -> MessageType.EMOTE
            else -> MessageType.TEXT
        }

        // Build attachments if present
        val contentUrl = event.content.url
        val attachments = if (contentUrl != null) {
            listOf(
                Attachment(
                    url = contentUrl,
                    mimeType = event.content.info?.mimetype ?: "application/octet-stream",
                    size = event.content.info?.size?.toLong() ?: 0L,
                    fileName = event.content.body ?: "file"
                )
            )
        } else emptyList()

        return Message(
            id = event.eventId,
            roomId = event.roomId,
            senderId = event.senderId,
            content = MessageContent(
                type = messageType,
                body = event.content.body ?: "",
                attachments = attachments
            ),
            timestamp = Instant.fromEpochMilliseconds(event.originServerTs),
            isOutgoing = false,
            status = MessageStatus.SYNCED
        )
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
 * Message update events for observing changes
 */
sealed class MessageUpdate {
    data class Created(val message: Message) : MessageUpdate()
    data class Edited(val message: Message) : MessageUpdate()
    data class Deleted(val messageId: String) : MessageUpdate()
    data class StatusChanged(val message: Message) : MessageUpdate()
}
