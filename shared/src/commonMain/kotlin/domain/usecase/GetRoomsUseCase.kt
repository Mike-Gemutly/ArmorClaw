package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.Room
import com.armorclaw.shared.domain.repository.RoomRepository
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.useCaseLogger

/**
 * Use case for getting the list of rooms
 *
 * Retrieves rooms from the repository.
 * Uses UseCaseLogger for proper separation of concerns in logging.
 */
class GetRoomsUseCase(
    private val roomRepository: RoomRepository
) {
    private val logger = useCaseLogger("GetRoomsUseCase", LogTag.UseCase.GetRooms)

    suspend operator fun invoke(): AppResult<List<Room>> {
        logger.logStart()

        return roomRepository.getRooms().also { result ->
            when (result) {
                is AppResult.Success -> {
                    logger.logSuccess("${result.data.size} rooms retrieved")
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
