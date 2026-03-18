package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.domain.repository.MessageRepository
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Clock
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import kotlin.test.*

/**
 * Tests for LoadMessagesUseCase
 *
 * Uses Robolectric to provide Android runtime for logging.
 */
@RunWith(RobolectricTestRunner::class)
class LoadMessagesUseCaseTest {

    // ========================================================================
    // Success Tests
    // ========================================================================

    @Test
    fun `should return messages on success`() = runTest {
        // Arrange
        val mockRepo = MockMessageRepositoryForLoad()
        val useCase = LoadMessagesUseCase(mockRepo)
        val roomId = "!room:example.com"

        // Act
        val result = useCase(roomId)

        // Assert
        assertTrue(result is AppResult.Success)
        val messages = (result as AppResult.Success).data
        assertEquals(2, messages.size)
    }

    @Test
    fun `should return empty list for room with no messages`() = runTest {
        // Arrange
        val mockRepo = MockMessageRepositoryForLoad(emptyList())
        val useCase = LoadMessagesUseCase(mockRepo)
        val roomId = "!empty:example.com"

        // Act
        val result = useCase(roomId)

        // Assert
        assertTrue(result is AppResult.Success)
        val messages = (result as AppResult.Success).data
        assertTrue(messages.isEmpty())
    }

    // ========================================================================
    // Pagination Tests
    // ========================================================================

    @Test
    fun `should use default limit of 50`() = runTest {
        // Arrange
        val mockRepo = MockMessageRepositoryForLoad()
        val useCase = LoadMessagesUseCase(mockRepo)

        // Act
        val result = useCase("!room:example.com")

        // Assert
        assertTrue(result is AppResult.Success)
        assertEquals(50, mockRepo.lastLimitUsed)
    }

    @Test
    fun `should support custom limit`() = runTest {
        // Arrange
        val mockRepo = MockMessageRepositoryForLoad()
        val useCase = LoadMessagesUseCase(mockRepo)

        // Act
        val result = useCase("!room:example.com", limit = 100)

        // Assert
        assertTrue(result is AppResult.Success)
        assertEquals(100, mockRepo.lastLimitUsed)
    }

    @Test
    fun `should support offset for pagination`() = runTest {
        // Arrange
        val mockRepo = MockMessageRepositoryForLoad()
        val useCase = LoadMessagesUseCase(mockRepo)

        // Act
        val result = useCase("!room:example.com", limit = 50, offset = 50)

        // Assert
        assertTrue(result is AppResult.Success)
        assertEquals(50, mockRepo.lastOffsetUsed)
    }

    @Test
    fun `should load second page correctly`() = runTest {
        // Arrange - create a paginating mock repository
        val allMessages = (1..100).map { i ->
            createMessage(id = "msg$i", timestamp = Clock.System.now())
        }
        val mockRepo = object : MessageRepository {
            var lastLimitUsed: Int = 0
            var lastOffsetUsed: Int = 0

            override suspend fun getMessages(
                roomId: String,
                limit: Int,
                offset: Int,
                context: OperationContext?
            ): AppResult<List<Message>> {
                lastLimitUsed = limit
                lastOffsetUsed = offset
                // Simulate actual pagination
                return AppResult.Success(allMessages.drop(offset).take(limit))
            }

            override suspend fun getMessage(roomId: String, messageId: String, context: OperationContext?): AppResult<Message?> {
                return AppResult.Success(allMessages.find { it.id == messageId })
            }

            override suspend fun sendMessage(roomId: String, content: MessageContent, context: OperationContext?): AppResult<Message> {
                return AppResult.Success(createMessage())
            }

            override suspend fun editMessage(roomId: String, messageId: String, content: MessageContent, context: OperationContext?): AppResult<Message> {
                return AppResult.Success(createMessage())
            }

            override suspend fun deleteMessage(roomId: String, messageId: String, context: OperationContext?): AppResult<Unit> {
                return AppResult.Success(Unit)
            }

            override suspend fun retryMessage(messageId: String, context: OperationContext?): AppResult<Message> {
                return AppResult.Success(createMessage())
            }

            override fun observeMessages(roomId: String): Flow<List<Message>> = flowOf(allMessages)

            override fun observeMessage(roomId: String, messageId: String): Flow<Message?> = flowOf(null)

            override suspend fun clearOfflineMessages(context: OperationContext?): AppResult<Int> = AppResult.Success(0)
        }
        val useCase = LoadMessagesUseCase(mockRepo)

        // Act
        val page1 = useCase("!room:example.com", limit = 50, offset = 0)
        val page2 = useCase("!room:example.com", limit = 50, offset = 50)

        // Assert
        assertTrue(page1 is AppResult.Success)
        assertTrue(page2 is AppResult.Success)
        assertEquals(50, (page1 as AppResult.Success).data.size)
        assertEquals(50, (page2 as AppResult.Success).data.size)
        // Verify pagination parameters are passed correctly
        assertEquals(50, mockRepo.lastOffsetUsed)
    }

    // ========================================================================
    // Error Handling Tests
    // ========================================================================

    @Test
    fun `should return error when repository fails`() = runTest {
        // Arrange
        val mockRepo = MockMessageRepositoryForLoad(shouldFail = true)
        val useCase = LoadMessagesUseCase(mockRepo)

        // Act
        val result = useCase("!room:example.com")

        // Assert
        assertTrue(result is AppResult.Error)
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    private fun createMessage(
        id: String = "msg_test",
        roomId: String = "!room:example.com",
        timestamp: kotlinx.datetime.Instant = Clock.System.now()
    ): Message {
        return Message(
            id = id,
            roomId = roomId,
            senderId = "@user:example.com",
            content = MessageContent(type = MessageType.TEXT, body = "Test message"),
            timestamp = timestamp,
            isOutgoing = false,
            status = MessageStatus.SENT
        )
    }
}

/**
 * Mock MessageRepository for LoadMessagesUseCase testing
 */
class MockMessageRepositoryForLoad(
    private val messages: List<Message> = listOf(
        Message(
            id = "msg1",
            roomId = "!room:example.com",
            senderId = "@user1:example.com",
            content = MessageContent(type = MessageType.TEXT, body = "Hello"),
            timestamp = Clock.System.now(),
            isOutgoing = false,
            status = MessageStatus.SENT
        ),
        Message(
            id = "msg2",
            roomId = "!room:example.com",
            senderId = "@user2:example.com",
            content = MessageContent(type = MessageType.TEXT, body = "World"),
            timestamp = Clock.System.now(),
            isOutgoing = false,
            status = MessageStatus.SENT
        )
    ),
    private val shouldFail: Boolean = false
) : MessageRepository {

    var lastLimitUsed: Int = 0
    var lastOffsetUsed: Int = 0

    override suspend fun getMessages(
        roomId: String,
        limit: Int,
        offset: Int,
        context: OperationContext?
    ): AppResult<List<Message>> {
        lastLimitUsed = limit
        lastOffsetUsed = offset

        if (shouldFail) {
            return AppResult.Error(
                AppError(
                    code = ArmorClawErrorCode.MESSAGE_NOT_FOUND.code,
                    message = "Failed to load messages",
                    source = "MockMessageRepositoryForLoad"
                )
            )
        }
        return AppResult.Success(messages)
    }

    override suspend fun getMessage(roomId: String, messageId: String, context: OperationContext?): AppResult<Message?> {
        return AppResult.Success(messages.find { it.id == messageId })
    }

    override suspend fun sendMessage(roomId: String, content: MessageContent, context: OperationContext?): AppResult<Message> {
        return AppResult.Success(
            Message(
                id = "new_msg",
                roomId = roomId,
                senderId = "@user:example.com",
                content = content,
                timestamp = Clock.System.now(),
                isOutgoing = true,
                status = MessageStatus.SENDING
            )
        )
    }

    override suspend fun editMessage(roomId: String, messageId: String, content: MessageContent, context: OperationContext?): AppResult<Message> {
        return AppResult.Success(
            Message(
                id = messageId,
                roomId = roomId,
                senderId = "@user:example.com",
                content = content,
                timestamp = Clock.System.now(),
                isOutgoing = true,
                status = MessageStatus.SENT
            )
        )
    }

    override suspend fun deleteMessage(roomId: String, messageId: String, context: OperationContext?): AppResult<Unit> {
        return AppResult.Success(Unit)
    }

    override suspend fun retryMessage(messageId: String, context: OperationContext?): AppResult<Message> {
        return AppResult.Success(messages.first())
    }

    override fun observeMessages(roomId: String): Flow<List<Message>> = flowOf(messages)

    override fun observeMessage(roomId: String, messageId: String): Flow<Message?> = flowOf(messages.find { it.id == messageId })

    override suspend fun clearOfflineMessages(context: OperationContext?): AppResult<Int> = AppResult.Success(0)
}
