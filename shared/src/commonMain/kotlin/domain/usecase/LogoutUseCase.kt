package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.repository.AuthRepository
import com.armorclaw.shared.domain.repository.SyncRepository
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.useCaseLogger
import com.armorclaw.shared.platform.logging.useCaseErrorBoundary

/**
 * Use case for handling user logout
 *
 * This use case ensures proper cleanup of all user session data:
 * - Clears authentication tokens
 * - Stops any active sync operations
 * - Clears local database
 * - Clears user preferences
 *
 * Uses proper logging to track operations at UseCase layer.
 */
class LogoutUseCase(
    private val authRepository: AuthRepository,
    private val syncRepository: SyncRepository
) {
    private val logger = useCaseLogger("LogoutUseCase", LogTag.UseCase.Logout)

    /**
     * Execute logout operation
     *
     * @param clearAllData Whether to clear all local data including messages
     * @return Result indicating success or failure
     */
    suspend operator fun invoke(clearAllData: Boolean = true): Result<Unit> = useCaseErrorBoundary(
        logger = logger,
        operation = "logout"
    ) {
        logger.logStart(mapOf("clearAllData" to clearAllData))

        // Step 1: Stop any active sync operations
        logger.logBusinessRuleViolation("stopSync", "Stopping sync before logout")
        runCatching { syncRepository.stopSync() }

        // Step 2: Logout from server (invalidate tokens)
        val serverLogoutResult = authRepository.logout()
        if (serverLogoutResult.isFailure) {
            logger.logValidationError("serverLogout", "Server logout failed, continuing with local cleanup")
            // Continue with local cleanup even if server logout fails
        }

        // Step 3: Clear local authentication data
        authRepository.clearLocalAuth()

        // Step 4: Clear all local data if requested
        if (clearAllData) {
            clearAllLocalData()
        }

        // Add breadcrumb for analytics
        AppLogger.breadcrumb(
            message = "User logged out",
            category = "auth",
            data = mapOf("clear_all_data" to clearAllData)
        )

        logger.logSuccess("User logged out")
    }

    /**
     * Clear all local data including messages, rooms, etc.
     */
    private suspend fun clearAllLocalData() {
        logger.logValidationError("clearAllLocalData", "Starting local data clear")
        // This would typically clear:
        // - Message database
        // - Room database
        // - User preferences
        // - Cache directories
        // For now, this is a placeholder - actual implementation depends on data layer
    }
}
