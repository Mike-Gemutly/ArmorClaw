package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.model.MessageContent
import com.armorclaw.shared.domain.model.ThreadInfo
import kotlinx.coroutines.flow.Flow

/**
 * Repository for thread operations
 * Supports Matrix m.thread specification (MSC3440)
 */
interface ThreadRepository {

    // ========== Thread Retrieval ==========

    /**
     * Get all messages in a thread, ordered by timestamp
     * @param threadRootId The ID of the thread root message
     * @param limit Maximum number of messages to return
     * @param offset Pagination offset
     * @return Result containing list of thread messages or error
     */
    suspend fun getThreadMessages(
        threadRootId: String,
        limit: Int = 50,
        offset: Int = 0
    ): Result<List<Message>>

    /**
     * Get thread information for a specific thread
     * @param threadRootId The ID of the thread root message
     * @return Result containing ThreadInfo or null if not a thread
     */
    suspend fun getThreadInfo(threadRootId: String): Result<ThreadInfo?>

    /**
     * Get all thread roots in a room (messages that have threads)
     * @param roomId The room ID
     * @param limit Maximum number of threads to return
     * @return Result containing list of thread root messages
     */
    suspend fun getThreadRoots(
        roomId: String,
        limit: Int = 50
    ): Result<List<Message>>

    /**
     * Get the most recent reply in a thread
     * @param threadRootId The ID of the thread root message
     * @return Result containing the last reply message or null
     */
    suspend fun getLastThreadReply(threadRootId: String): Result<Message?>

    // ========== Thread Observation ==========

    /**
     * Observe all messages in a thread for real-time updates
     * @param threadRootId The ID of the thread root message
     * @return Flow emitting list of thread messages
     */
    fun observeThreadMessages(threadRootId: String): Flow<List<Message>>

    /**
     * Observe thread information for changes
     * @param threadRootId The ID of the thread root message
     * @return Flow emitting ThreadInfo updates
     */
    fun observeThreadInfo(threadRootId: String): Flow<ThreadInfo?>

    /**
     * Observe all thread roots in a room
     * @param roomId The room ID
     * @return Flow emitting list of thread root messages
     */
    fun observeThreadRoots(roomId: String): Flow<List<Message>>

    // ========== Thread Actions ==========

    /**
     * Send a reply to a thread
     * Creates a thread if one doesn't exist
     * @param threadRootId The ID of the thread root message
     * @param content The message content
     * @return Result containing the created message or error
     */
    suspend fun sendThreadReply(
        threadRootId: String,
        content: MessageContent
    ): Result<Message>

    /**
     * Create a new thread from an existing message
     * @param roomId The room ID
     * @param rootMessageId The ID of the message to start the thread from
     * @return Result containing the ThreadInfo or error
     */
    suspend fun createThread(
        roomId: String,
        rootMessageId: String
    ): Result<ThreadInfo>

    /**
     * Mark all messages in a thread as read
     * @param threadRootId The ID of the thread root message
     * @return Result indicating success or error
     */
    suspend fun markThreadAsRead(threadRootId: String): Result<Unit>

    // ========== Thread Participants ==========

    /**
     * Get all participants in a thread
     * @param threadRootId The ID of the thread root message
     * @return Result containing list of participant user IDs
     */
    suspend fun getThreadParticipants(threadRootId: String): Result<List<String>>

    /**
     * Add a participant to a thread notification list
     * @param threadRootId The ID of the thread root message
     * @param userId The user ID to add
     * @return Result indicating success or error
     */
    suspend fun addThreadParticipant(
        threadRootId: String,
        userId: String
    ): Result<Unit>

    /**
     * Remove a participant from thread notifications
     * @param threadRootId The ID of the thread root message
     * @param userId The user ID to remove
     * @return Result indicating success or error
     */
    suspend fun removeThreadParticipant(
        threadRootId: String,
        userId: String
    ): Result<Unit>

    // ========== Thread Statistics ==========

    /**
     * Get the number of unread replies in a thread
     * @param threadRootId The ID of the thread root message
     * @param userId The current user ID
     * @return Result containing unread count
     */
    suspend fun getThreadUnreadCount(
        threadRootId: String,
        userId: String
    ): Result<Int>

    /**
     * Get the total number of threads in a room
     * @param roomId The room ID
     * @return Result containing thread count
     */
    suspend fun getThreadCount(roomId: String): Result<Int>

    // ========== Matrix m.thread Support ==========

    /**
     * Sync thread relationships from server
     * Handles m.thread and m.relates_to events
     * @param roomId The room ID
     * @return Result indicating success or error
     */
    suspend fun syncThreadRelations(roomId: String): Result<Unit>

    /**
     * Check if a message is a thread root
     * @param messageId The message ID to check
     * @return Result containing boolean
     */
    suspend fun isThreadRoot(messageId: String): Result<Boolean>
}

/**
 * Thread-related operation types for offline sync
 */
enum class ThreadOperationType {
    CREATE_THREAD,
    SEND_THREAD_REPLY,
    MARK_THREAD_READ,
    ADD_THREAD_PARTICIPANT,
    REMOVE_THREAD_PARTICIPANT
}

/**
 * Thread notification settings
 */
data class ThreadNotificationSettings(
    val threadRootId: String,
    val enabled: Boolean = true,
    val notifyForReplies: Boolean = true,
    val notifyForMentions: Boolean = true
)
