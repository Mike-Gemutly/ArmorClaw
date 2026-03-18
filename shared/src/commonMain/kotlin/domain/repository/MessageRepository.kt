package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.model.MessageContent
import com.armorclaw.shared.domain.model.OperationContext
import kotlinx.coroutines.flow.Flow

/**
 * Repository interface for message operations
 *
 * All operations support optional OperationContext for correlation ID tracing.
 * If no context is provided, one will be created automatically.
 */
interface MessageRepository {
    /**
     * Get paginated messages for a room
     * @param roomId The room ID
     * @param limit Maximum number of messages to return
     * @param offset Number of messages to skip (for pagination)
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun getMessages(
        roomId: String,
        limit: Int = 50,
        offset: Int = 0,
        context: OperationContext? = null
    ): AppResult<List<Message>>

    /**
     * Get a single message
     */
    suspend fun getMessage(
        roomId: String,
        messageId: String,
        context: OperationContext? = null
    ): AppResult<Message?>

    /**
     * Send a new message
     */
    suspend fun sendMessage(
        roomId: String,
        content: MessageContent,
        context: OperationContext? = null
    ): AppResult<Message>

    /**
     * Edit an existing message
     */
    suspend fun editMessage(
        roomId: String,
        messageId: String,
        content: MessageContent,
        context: OperationContext? = null
    ): AppResult<Message>

    /**
     * Delete a message (soft delete)
     */
    suspend fun deleteMessage(
        roomId: String,
        messageId: String,
        context: OperationContext? = null
    ): AppResult<Unit>

    /**
     * Retry sending a failed message
     */
    suspend fun retryMessage(
        messageId: String,
        context: OperationContext? = null
    ): AppResult<Message>

    /**
     * Observe messages in a room (reactive)
     */
    fun observeMessages(roomId: String): Flow<List<Message>>

    /**
     * Observe a single message (reactive)
     */
    fun observeMessage(roomId: String, messageId: String): Flow<Message?>

    /**
     * Clear all offline queued messages
     */
    suspend fun clearOfflineMessages(context: OperationContext? = null): AppResult<Int>
}
