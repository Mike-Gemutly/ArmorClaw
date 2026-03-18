package com.armorclaw.shared.data.dao

import com.armorclaw.shared.domain.model.*
import kotlinx.datetime.Instant

/**
 * Data Access Object interface for thread operations
 *
 * This interface defines the contract for thread data operations.
 * The actual implementation should use SQLDelight generated queries.
 *
 * TODO: Implement with actual SQLDelight database after successful build
 */
interface ThreadDao {

    /**
     * Get all messages in a thread, ordered by timestamp
     */
    suspend fun getThreadMessages(
        threadRootId: String,
        limit: Int = 50,
        offset: Int = 0
    ): List<Message>

    /**
     * Get the most recent reply in a thread
     */
    suspend fun getLastThreadReply(threadRootId: String): Message?

    /**
     * Get all thread roots in a room (messages that have threads)
     */
    suspend fun getThreadRoots(roomId: String, limit: Int = 50): List<Message>

    /**
     * Count replies in a thread
     */
    suspend fun getThreadReplyCount(threadRootId: String): Int

    /**
     * Insert a thread reply message
     */
    suspend fun insertThreadReply(message: Message)

    /**
     * Update room thread statistics
     */
    suspend fun updateRoomThreadStats(
        roomId: String,
        activeThreadCount: Int,
        threadUnreadCount: Int,
        lastThreadActivity: Instant?
    )
}

/**
 * In-memory implementation of ThreadDao for testing/development
 * Replace with SQLDelight implementation in production
 */
class InMemoryThreadDao : ThreadDao {

    private val threadMessages = mutableMapOf<String, MutableList<Message>>()
    private val roomThreadStats = mutableMapOf<String, RoomThreadStats>()

    private data class RoomThreadStats(
        var activeThreadCount: Int = 0,
        var threadUnreadCount: Int = 0,
        var lastThreadActivity: Instant? = null
    )

    override suspend fun getThreadMessages(
        threadRootId: String,
        limit: Int,
        offset: Int
    ): List<Message> {
        val messages = threadMessages[threadRootId] ?: return emptyList()
        return messages.drop(offset).take(limit)
    }

    override suspend fun getLastThreadReply(threadRootId: String): Message? {
        return threadMessages[threadRootId]?.lastOrNull()
    }

    override suspend fun getThreadRoots(roomId: String, limit: Int): List<Message> {
        // In a real implementation, this would query messages that have thread replies
        return threadMessages.entries
            .filter { it.value.any { msg -> msg.roomId == roomId } }
            .mapNotNull { it.value.firstOrNull() }
            .take(limit)
    }

    override suspend fun getThreadReplyCount(threadRootId: String): Int {
        return threadMessages[threadRootId]?.size ?: 0
    }

    override suspend fun insertThreadReply(message: Message) {
        val threadRootId = message.threadInfo?.threadRootId ?: return
        threadMessages.getOrPut(threadRootId) { mutableListOf() }.add(message)
    }

    override suspend fun updateRoomThreadStats(
        roomId: String,
        activeThreadCount: Int,
        threadUnreadCount: Int,
        lastThreadActivity: Instant?
    ) {
        roomThreadStats[roomId] = RoomThreadStats(
            activeThreadCount = activeThreadCount,
            threadUnreadCount = threadUnreadCount,
            lastThreadActivity = lastThreadActivity
        )
    }
}
