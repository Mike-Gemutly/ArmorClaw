package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.model.MessageContent
import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.AppError
import com.armorclaw.shared.domain.model.ArmorClawErrorCode
import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.useCaseLogger

/**
 * Use case for sending a message to a room
 *
 * Handles validation, encryption, and sending of messages.
 * Uses UseCaseLogger for proper separation of concerns in logging.
 */
class SendMessageUseCase(
    private val messageRepository: MessageRepository
) {
    private val logger = useCaseLogger("SendMessageUseCase", LogTag.UseCase.SendMessage)

    /**
     * Send a message to a room
     *
     * @param roomId The room to send the message to
     * @param content The message content
     * @return AppResult containing the sent message or an error
     */
    suspend operator fun invoke(
        roomId: String,
        content: MessageContent
    ): AppResult<Message> {
        val startTime = System.currentTimeMillis()

        logger.logStart(mapOf(
            "roomId" to roomId,
            "contentType" to content.type.name,
            "bodyLength" to content.body.length
        ))

        // Validate input
        if (content.body.isBlank()) {
            logger.logValidationError("body", "Message cannot be empty")
            return createValidationError(
                ArmorClawErrorCode.MESSAGE_SEND_FAILED,
                "Message cannot be empty",
                roomId
            )
        }

        if (content.body.length > MAX_MESSAGE_LENGTH) {
            logger.logValidationError("body", "Message too long (${content.body.length} > $MAX_MESSAGE_LENGTH)")
            return createValidationError(
                ArmorClawErrorCode.MESSAGE_TOO_LONG,
                "Message too long (max $MAX_MESSAGE_LENGTH characters)",
                roomId
            )
        }

        // Send message
        return try {
            val result = messageRepository.sendMessage(roomId, content)

            val duration = System.currentTimeMillis() - startTime
            logger.logExecutionTime(duration)

            when (result) {
                is AppResult.Success -> {
                    val message = result.data
                    logger.logSuccess("Message ${message.id} sent")

                    // Add breadcrumb for crash reporting
                    AppLogger.breadcrumb(
                        message = "Message sent via UseCase",
                        category = "messaging",
                        data = mapOf("room_id" to roomId, "message_id" to message.id)
                    )

                    AppResult.success(message)
                }
                is AppResult.Error -> {
                    logger.logFailure(Exception(result.error.message), mapOf("roomId" to roomId))
                    AppResult.error(
                        AppError(
                            code = ArmorClawErrorCode.MESSAGE_SEND_FAILED.code,
                            message = result.error.message,
                            technicalMessage = result.error.technicalMessage,
                            source = "SendMessageUseCase",
                            cause = result.error.cause,
                            metadata = mapOf("roomId" to roomId)
                        )
                    )
                }
                is AppResult.Loading -> {
                    // Should not happen for send operation, return as-is
                    result
                }
            }
        } catch (e: Exception) {
            logger.logFailure(e, mapOf("roomId" to roomId))
            AppResult.error(
                AppError(
                    code = ArmorClawErrorCode.MESSAGE_SEND_FAILED.code,
                    message = "Failed to send message",
                    technicalMessage = e.stackTraceToString(),
                    source = "SendMessageUseCase",
                    cause = e,
                    metadata = mapOf("roomId" to roomId)
                )
            )
        }
    }

    private fun createValidationError(
        errorCode: ArmorClawErrorCode,
        message: String,
        roomId: String
    ): AppResult<Message> {
        return AppResult.error(
            AppError(
                code = errorCode.code,
                message = message,
                source = "SendMessageUseCase.Validation",
                isRecoverable = true,
                metadata = mapOf("roomId" to roomId)
            )
        )
    }

    companion object {
        const val MAX_MESSAGE_LENGTH = 10000
    }
}

/**
 * Legacy exception for backward compatibility
 */
class ValidationException(message: String) : Exception(message)
