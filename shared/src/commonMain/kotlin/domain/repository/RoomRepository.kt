package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.domain.model.Room
import com.armorclaw.shared.domain.model.RoomMember
import kotlinx.coroutines.flow.Flow

/**
 * Repository interface for room operations
 *
 * All operations support optional OperationContext for correlation ID tracing.
 * If no context is provided, one will be created automatically.
 */
interface RoomRepository {
    /**
     * Get all rooms the user is a member of
     */
    suspend fun getRooms(context: OperationContext? = null): AppResult<List<Room>>

    /**
     * Get a specific room by ID
     */
    suspend fun getRoom(roomId: String, context: OperationContext? = null): AppResult<Room?>

    /**
     * Create a new room
     */
    suspend fun createRoom(
        name: String,
        isDirect: Boolean = false,
        context: OperationContext? = null
    ): AppResult<Room>

    /**
     * Join an existing room
     */
    suspend fun joinRoom(roomId: String, context: OperationContext? = null): AppResult<Unit>

    /**
     * Leave a room
     */
    suspend fun leaveRoom(roomId: String, context: OperationContext? = null): AppResult<Unit>

    /**
     * Invite a user to a room
     */
    suspend fun inviteUser(
        roomId: String,
        userId: String,
        context: OperationContext? = null
    ): AppResult<Unit>

    /**
     * Observe all rooms (reactive)
     */
    fun observeRooms(): Flow<List<Room>>

    /**
     * Observe a specific room (reactive)
     */
    fun observeRoom(roomId: String): Flow<Room?>

    /**
     * Update the last message reference for a room
     */
    suspend fun updateLastMessage(
        roomId: String,
        messageId: String,
        context: OperationContext? = null
    ): AppResult<Unit>

    /**
     * Increment the unread count for a room
     */
    suspend fun incrementUnreadCount(roomId: String, context: OperationContext? = null): AppResult<Unit>

    /**
     * Mark all messages in a room as read
     */
    suspend fun markAsRead(roomId: String, context: OperationContext? = null): AppResult<Unit>
}
