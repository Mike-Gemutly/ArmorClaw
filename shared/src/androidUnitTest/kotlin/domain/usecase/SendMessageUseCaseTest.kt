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
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

/**
 * Tests for SendMessageUseCase
 *
 * Uses Robolectric to provide Android runtime for logging.
 */
@RunWith(RobolectricTestRunner::class)
class SendMessageUseCaseTest {

    // ========================================================================
    // Success Tests
    // ========================================================================

    @Test
    fun `should send message successfully`() = runTest {
        // Arrange
        val content = MessageContent(
            type = MessageType.TEXT,
            body = "Hello, world!"
        )
        val roomId = "!room:example.com"
        val expectedMessage = Message(
            id = "msg_123",
            roomId = roomId,
            senderId = "@user:example.com",
            content = content,
            timestamp = Clock.System.now(),
            isOutgoing = true,
            status = MessageStatus.SENT
        )

        val mockRepository = MockMessageRepositoryForSend(expectedMessage)
        val useCase = SendMessageUseCase(mockRepository)

        // Act
        val result = useCase(roomId, content)

        // Assert
        assertTrue(result.isSuccess)
        val message = result.getOrNull()
        assertNotNull(message)
        assertEquals("msg_123", message?.id)
        assertEquals("Hello, world!", message?.content?.body)
    }

    // ========================================================================
    // Validation Tests
    // ========================================================================

    @Test
    fun `should fail when message body is empty`() = runTest {
        // Arrange
        val content = MessageContent(
            type = MessageType.TEXT,
            body = ""
        )
        val roomId = "!room:example.com"

        val mockRepository = MockMessageRepositoryForSend()
        val useCase = SendMessageUseCase(mockRepository)

        // Act
        val result = useCase(roomId, content)

        // Assert - UseCase should validate and fail
        assertFalse(result.isSuccess)
        assertTrue(result.isError)
    }

    @Test
    fun `should fail when message body is blank`() = runTest {
        // Arrange
        val content = MessageContent(
            type = MessageType.TEXT,
            body = "   "
        )
        val roomId = "!room:example.com"

        val mockRepository = MockMessageRepositoryForSend()
        val useCase = SendMessageUseCase(mockRepository)

        // Act
        val result = useCase(roomId, content)

        // Assert - UseCase should validate and fail
        assertFalse(result.isSuccess)
        assertTrue(result.isError)
    }

    // ========================================================================
    // Error Handling Tests
    // ========================================================================

    @Test
    fun `should return failure when repository fails`() = runTest {
        // Arrange
        val content = MessageContent(
            type = MessageType.TEXT,
            body = "Hello"
        )
        val roomId = "!room:example.com"

        val mockRepository = MockMessageRepositoryForSend(shouldFail = true)
        val useCase = SendMessageUseCase(mockRepository)

        // Act
        val result = useCase(roomId, content)

        // Assert
        assertTrue(result.isError)
    }
}

/**
 * Mock MessageRepository for SendMessageUseCase testing
 */
class MockMessageRepositoryForSend(
    private val messageToReturn: Message? = null,
    private val shouldFail: Boolean = false
) : MessageRepository {

    override suspend fun getMessages(
        roomId: String,
        limit: Int,
        offset: Int,
        context: OperationContext?
    ): AppResult<List<Message>> {
        return AppResult.Success(emptyList())
    }

    override suspend fun getMessage(
        roomId: String,
        messageId: String,
        context: OperationContext?
    ): AppResult<Message?> {
        return AppResult.Success(null)
    }

    override suspend fun sendMessage(
        roomId: String,
        content: MessageContent,
        context: OperationContext?
    ): AppResult<Message> {
        if (shouldFail) {
            return AppResult.Error(
                AppError(
                    code = ArmorClawErrorCode.SERVER_ERROR.code,
                    message = "Failed to send message",
                    source = "MockMessageRepositoryForSend"
                )
            )
        }
        return AppResult.Success(
            messageToReturn ?: Message(
                id = "msg_${System.currentTimeMillis()}",
                roomId = roomId,
                senderId = "@user:example.com",
                content = content,
                timestamp = Clock.System.now(),
                isOutgoing = true,
                status = MessageStatus.SENDING
            )
        )
    }

    override suspend fun editMessage(
        roomId: String,
        messageId: String,
        content: MessageContent,
        context: OperationContext?
    ): AppResult<Message> {
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

    override suspend fun deleteMessage(
        roomId: String,
        messageId: String,
        context: OperationContext?
    ): AppResult<Unit> {
        return AppResult.Success(Unit)
    }

    override suspend fun retryMessage(
        messageId: String,
        context: OperationContext?
    ): AppResult<Message> {
        return AppResult.Success(
            Message(
                id = messageId,
                roomId = "!room:example.com",
                senderId = "@user:example.com",
                content = MessageContent(type = MessageType.TEXT, body = "Retried"),
                timestamp = Clock.System.now(),
                isOutgoing = true,
                status = MessageStatus.SENT
            )
        )
    }

    override fun observeMessages(roomId: String): Flow<List<Message>> = flowOf(emptyList())

    override fun observeMessage(roomId: String, messageId: String): Flow<Message?> = flowOf(null)

    override suspend fun clearOfflineMessages(context: OperationContext?): AppResult<Int> {
        return AppResult.Success(0)
    }
}
