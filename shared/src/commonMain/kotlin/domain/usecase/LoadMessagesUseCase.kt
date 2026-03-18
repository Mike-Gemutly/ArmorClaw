package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.useCaseLogger

/**
 * Use case for loading messages from a room
 *
 * Retrieves messages from the repository with pagination support.
 * Uses UseCaseLogger for proper separation of concerns in logging.
 */
class LoadMessagesUseCase(
    private val messageRepository: MessageRepository
) {
    private val logger = useCaseLogger("LoadMessagesUseCase", LogTag.UseCase.LoadMessages)

    suspend operator fun invoke(
        roomId: String,
        limit: Int = 50,
        offset: Int = 0
    ): AppResult<List<Message>> {
        logger.logStart(mapOf(
            "roomId" to roomId,
            "limit" to limit,
            "offset" to offset
        ))

        return messageRepository.getMessages(roomId, limit, offset).also { result ->
            when (result) {
                is AppResult.Success -> {
                    logger.logSuccess("${result.data.size} messages loaded")
                }
                is AppResult.Error -> {
                    logger.logFailure(Exception(result.error.message))
                }
                is AppResult.Loading -> {
                    // No action needed for loading state in this use case
                }
            }
        }
    }
}
