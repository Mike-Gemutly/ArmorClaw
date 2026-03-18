package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.UserSession
import com.armorclaw.shared.domain.repository.AuthRepository
import com.armorclaw.shared.domain.repository.ServerConfig
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.PasswordMasked
import com.armorclaw.shared.platform.logging.useCaseLogger

/**
 * Use case for user login
 *
 * Validates credentials and authenticates with the auth repository.
 * Uses UseCaseLogger for proper separation of concerns in logging.
 */
class LoginUseCase(
    private val authRepository: AuthRepository
) {
    private val logger = useCaseLogger("LoginUseCase", LogTag.UseCase.Login)

    suspend operator fun invoke(config: ServerConfig): Result<UserSession> {
        logger.logStart(mapOf(
            "homeserver" to config.homeserver,
            "username" to config.username,
            "password" to PasswordMasked(config.password)
        ))

        // Validate config
        if (config.homeserver.isBlank()) {
            logger.logValidationError("homeserver", "Homeserver URL is required")
            return Result.failure(ValidationException("Homeserver URL is required"))
        }

        if (!config.homeserver.startsWith("https://") && !config.homeserver.startsWith("http://")) {
            logger.logValidationError("homeserver", "Invalid homeserver URL format")
            return Result.failure(ValidationException("Invalid homeserver URL"))
        }

        if (config.username.isBlank()) {
            logger.logValidationError("username", "Username is required")
            return Result.failure(ValidationException("Username is required"))
        }

        if (config.password.isBlank()) {
            logger.logValidationError("password", "Password is required")
            return Result.failure(ValidationException("Password is required"))
        }

        logger.logBusinessRuleViolation("validation", "All validations passed")

        // Attempt login
        return authRepository.login(config).also { result ->
            result.fold(
                onSuccess = { session ->
                    logger.logSuccess("User ${session.userId} logged in")
                },
                onFailure = { error ->
                    logger.logFailure(error)
                }
            )
        }
    }
}
