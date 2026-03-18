package com.armorclaw.shared.platform.logging

import com.armorclaw.shared.domain.model.AppError
import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.ArmorClawErrorCode
import com.armorclaw.shared.domain.model.ErrorCategory
import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.domain.model.RecoveryAction
import com.armorclaw.shared.domain.model.withContext
import kotlinx.coroutines.CancellationException

/**
 * Error boundary patterns for consistent error handling across layers.
 *
 * These utilities ensure errors are:
 * 1. Logged at the appropriate boundary
 * 2. Wrapped with context and correlation IDs
 * 3. Propagated correctly to upper layers
 * 4. Return unified AppResult type
 */

// ==================== ERROR TYPES ====================

/**
 * Base class for domain errors with context
 */
sealed class DomainError(
    override val message: String,
    override val cause: Throwable? = null,
    val context: Map<String, Any?> = emptyMap()
) : Exception(message, cause) {

    /**
     * Network-related error
     */
    class NetworkError(
        message: String = "Network error",
        cause: Throwable? = null,
        val statusCode: Int? = null,
        context: Map<String, Any?> = emptyMap()
    ) : DomainError(message, cause, context)

    /**
     * Authentication/authorization error
     */
    class AuthError(
        message: String = "Authentication error",
        cause: Throwable? = null,
        val authType: String? = null,
        context: Map<String, Any?> = emptyMap()
    ) : DomainError(message, cause, context)

    /**
     * Data validation error
     */
    class ValidationError(
        message: String = "Validation error",
        cause: Throwable? = null,
        val field: String? = null,
        context: Map<String, Any?> = emptyMap()
    ) : DomainError(message, cause, context)

    /**
     * Data not found error
     */
    class NotFoundError(
        message: String = "Resource not found",
        cause: Throwable? = null,
        val resourceType: String? = null,
        val resourceId: String? = null,
        context: Map<String, Any?> = emptyMap()
    ) : DomainError(message, cause, context)

    /**
     * Database/storage error
     */
    class StorageError(
        message: String = "Storage error",
        cause: Throwable? = null,
        val operation: String? = null,
        context: Map<String, Any?> = emptyMap()
    ) : DomainError(message, cause, context)

    /**
     * Encryption/security error
     */
    class SecurityError(
        message: String = "Security error",
        cause: Throwable? = null,
        val securityType: String? = null,
        context: Map<String, Any?> = emptyMap()
    ) : DomainError(message, cause, context)

    /**
     * Unknown/unexpected error
     */
    class UnknownError(
        message: String = "Unknown error",
        cause: Throwable? = null,
        context: Map<String, Any?> = emptyMap()
    ) : DomainError(message, cause, context)

    /**
     * Convert to AppError for AppResult
     */
    fun toAppError(source: String): AppError {
        val errorCode = when (this) {
            is NetworkError -> when (statusCode) {
                401 -> ArmorClawErrorCode.SESSION_EXPIRED
                403 -> ArmorClawErrorCode.LOGIN_FAILED
                404 -> ArmorClawErrorCode.HOMESERVER_UNREACHABLE
                in 500..599 -> ArmorClawErrorCode.SERVER_ERROR
                else -> ArmorClawErrorCode.NETWORK_CHANGED
            }
            is AuthError -> ArmorClawErrorCode.LOGIN_FAILED
            is ValidationError -> ArmorClawErrorCode.MESSAGE_SEND_FAILED
            is NotFoundError -> ArmorClawErrorCode.MESSAGE_NOT_FOUND
            is StorageError -> ArmorClawErrorCode.SYNC_STORAGE_FULL
            is SecurityError -> ArmorClawErrorCode.ENCRYPTION_KEY_ERROR
            is UnknownError -> ArmorClawErrorCode.UNKNOWN_ERROR
        }

        val filteredContext = context.filterValues { it != null }.mapValues { it.value!! }
        return AppError(
            code = errorCode.code,
            message = message,
            technicalMessage = cause?.stackTraceToString(),
            source = source,
            cause = cause,
            metadata = filteredContext,
            isRecoverable = errorCode.recoverable,
            recoveryAction = when (errorCode.category) {
                ErrorCategory.NETWORK -> RecoveryAction.CheckNetwork
                ErrorCategory.AUTH -> RecoveryAction.Login
                else -> RecoveryAction.Retry
            }
        )
    }
}

// ==================== REPOSITORY ERROR BOUNDARY ====================

/**
 * Repository error boundary with AppResult and correlation ID support
 *
 * Catches exceptions at repository boundary, logs them with context,
 * and wraps them in AppResult.Error with full error context.
 */
inline fun <T> repositoryOperation(
    logger: RepositoryLogger,
    operation: String,
    context: OperationContext = OperationContext.create(),
    errorCode: ArmorClawErrorCode = ArmorClawErrorCode.UNKNOWN_ERROR,
    block: () -> T
): AppResult<T> {
    return try {
        logger.logOperationStart(operation, context.toLogMap())
        val result = block()
        logger.logOperationSuccess(operation)
        AppResult.success(result, context.toLogMap().filterValues { it != null }.mapValues { it.value!! })
    } catch (e: CancellationException) {
        throw e
    } catch (e: DomainError) {
        logger.logOperationError(operation, e, context.toLogMap())
        AppResult.error(e.toAppError("Repository:$operation"), context.toLogMap().filterValues { it != null }.mapValues { it.value!! })
    } catch (e: Exception) {
        logger.logOperationError(operation, e, context.toLogMap())
        val metadata = context.toLogMap().filterValues { it != null }.mapValues { it.value!! }
        AppResult.error(
            AppError(
                code = errorCode.code,
                message = "$operation failed: ${e.message}",
                technicalMessage = e.stackTraceToString(),
                source = "Repository:$operation",
                cause = e,
                metadata = metadata,
                isRecoverable = errorCode.recoverable
            ),
            metadata
        )
    }
}

/**
 * Suspend version of repository error boundary
 */
suspend fun <T> repositoryOperationSuspend(
    logger: RepositoryLogger,
    operation: String,
    context: OperationContext = OperationContext.create(),
    errorCode: ArmorClawErrorCode = ArmorClawErrorCode.UNKNOWN_ERROR,
    block: suspend () -> T
): AppResult<T> {
    return try {
        logger.logOperationStart(operation, context.toLogMap())
        val result = block()
        logger.logOperationSuccess(operation)
        val metadata = context.toLogMap().filterValues { it != null }.mapValues { it.value!! }
        AppResult.success(result, metadata)
    } catch (e: CancellationException) {
        throw e
    } catch (e: DomainError) {
        logger.logOperationError(operation, e, context.toLogMap())
        val metadata = context.toLogMap().filterValues { it != null }.mapValues { it.value!! }
        AppResult.error(e.toAppError("Repository:$operation"), metadata)
    } catch (e: Exception) {
        logger.logOperationError(operation, e, context.toLogMap())
        val metadata = context.toLogMap().filterValues { it != null }.mapValues { it.value!! }
        AppResult.error(
            AppError(
                code = errorCode.code,
                message = "$operation failed: ${e.message}",
                technicalMessage = e.stackTraceToString(),
                source = "Repository:$operation",
                cause = e,
                metadata = metadata,
                isRecoverable = errorCode.recoverable
            ),
            metadata
        )
    }
}

// ==================== USECASE ERROR BOUNDARY ====================

/**
 * UseCase error boundary with AppResult and correlation ID support
 *
 * Catches exceptions at UseCase boundary, logs them with context,
 * and ensures proper error propagation with full context.
 */
suspend fun <T> useCaseOperation(
    logger: UseCaseLogger,
    operation: String,
    context: OperationContext = OperationContext.create(),
    errorCode: ArmorClawErrorCode = ArmorClawErrorCode.UNKNOWN_ERROR,
    block: suspend () -> T
): AppResult<T> {
    val startTime = System.currentTimeMillis()
    return try {
        logger.logStart(mapOf("operation" to operation).withContext(context))
        val result = block()
        logger.logSuccess()
        logger.logExecutionTime(System.currentTimeMillis() - startTime)
        val metadata = context.toLogMap().filterValues { it != null }.mapValues { it.value!! }
        AppResult.success(result, metadata)
    } catch (e: CancellationException) {
        throw e
    } catch (e: DomainError) {
        logger.logFailure(e, mapOf("operation" to operation).withContext(context))
        val metadata = context.toLogMap().filterValues { it != null }.mapValues { it.value!! }
        AppResult.error(e.toAppError("UseCase:$operation"), metadata)
    } catch (e: Exception) {
        logger.logFailure(e, mapOf("operation" to operation).withContext(context))
        val metadata = context.toLogMap().filterValues { it != null }.mapValues { it.value!! }
        AppResult.error(
            AppError(
                code = errorCode.code,
                message = "$operation failed: ${e.message}",
                technicalMessage = e.stackTraceToString(),
                source = "UseCase:$operation",
                cause = e,
                metadata = metadata,
                isRecoverable = errorCode.recoverable
            ),
            metadata
        )
    }
}

// ==================== LEGACY COMPATIBILITY ====================

/**
 * Legacy repository error boundary (returns Kotlin Result)
 * @deprecated Use repositoryOperation instead
 */
inline fun <T> repositoryErrorBoundary(
    logger: RepositoryLogger,
    operation: String,
    block: () -> T
): Result<T> {
    return try {
        logger.logOperationStart(operation)
        val result = block()
        logger.logOperationSuccess(operation)
        Result.success(result)
    } catch (e: CancellationException) {
        throw e
    } catch (e: DomainError.NotFoundError) {
        logger.logOperationError(operation, e)
        Result.failure(e)
    } catch (e: DomainError.NetworkError) {
        logger.logOperationError(operation, e)
        Result.failure(e)
    } catch (e: Exception) {
        logger.logOperationError(operation, e)
        Result.failure(DomainError.UnknownError(
            message = "$operation failed: ${e.message}",
            cause = e,
            context = mapOf("operation" to operation)
        ))
    }
}

/**
 * Legacy suspend version of repository error boundary
 * @deprecated Use repositoryOperationSuspend instead
 */
suspend fun <T> repositoryErrorBoundarySuspend(
    logger: RepositoryLogger,
    operation: String,
    block: suspend () -> T
): Result<T> {
    return try {
        logger.logOperationStart(operation)
        val result = block()
        logger.logOperationSuccess(operation)
        Result.success(result)
    } catch (e: CancellationException) {
        throw e
    } catch (e: DomainError.NotFoundError) {
        logger.logOperationError(operation, e)
        Result.failure(e)
    } catch (e: DomainError.NetworkError) {
        logger.logOperationError(operation, e)
        Result.failure(e)
    } catch (e: Exception) {
        logger.logOperationError(operation, e)
        Result.failure(DomainError.UnknownError(
            message = "$operation failed: ${e.message}",
            cause = e,
            context = mapOf("operation" to operation)
        ))
    }
}

/**
 * Legacy UseCase error boundary (returns Kotlin Result)
 * @deprecated Use useCaseOperation instead
 */
suspend fun <T> useCaseErrorBoundary(
    logger: UseCaseLogger,
    operation: String,
    block: suspend () -> T
): Result<T> {
    val startTime = System.currentTimeMillis()
    return try {
        logger.logStart(mapOf("operation" to operation))
        val result = block()
        logger.logSuccess()
        logger.logExecutionTime(System.currentTimeMillis() - startTime)
        Result.success(result)
    } catch (e: CancellationException) {
        throw e
    } catch (e: DomainError) {
        logger.logFailure(e)
        Result.failure(e)
    } catch (e: Exception) {
        logger.logFailure(e)
        Result.failure(DomainError.UnknownError(
            message = "$operation failed: ${e.message}",
            cause = e,
            context = mapOf("operation" to operation)
        ))
    }
}

// ==================== LOGGING CONVENIENCE FUNCTIONS ====================

/**
 * Create a repository logger
 */
fun repositoryLogger(repositoryName: String, tag: LogTag = LogTag.Data.AuthRepository): RepositoryLogger {
    return RepositoryLogger(tag, repositoryName)
}

/**
 * Create a use case logger
 */
fun useCaseLogger(useCaseName: String, tag: LogTag): UseCaseLogger {
    return UseCaseLogger(tag, useCaseName)
}

/**
 * Create a ViewModel logger
 */
fun viewModelLogger(viewModelName: String, tag: LogTag): ViewModelLogger {
    return ViewModelLogger(tag, viewModelName)
}

/**
 * Create a service logger
 */
fun serviceLogger(serviceName: String, tag: LogTag): ServiceLogger {
    return ServiceLogger(tag, serviceName)
}

// ==================== EXTENSION FUNCTIONS ====================

/**
 * Log and rethrow - for cases where you want to log but still propagate
 */
fun Throwable.logAndRethrow(logger: ViewModelLogger, operation: String): Nothing {
    logger.logError(operation, this)
    throw this
}

/**
 * Log result - logs success or failure
 */
fun <T> Result<T>.logResult(logger: UseCaseLogger, operation: String): Result<T> {
    onSuccess {
        logger.logSuccess("$operation succeeded")
    }
    onFailure { error ->
        logger.logFailure(error, mapOf("operation" to operation))
    }
    return this
}

/**
 * Map error to different type while preserving logging
 */
inline fun <T, R> Result<T>.mapError(
    logger: UseCaseLogger,
    crossinline transform: (Throwable) -> Throwable
): Result<R> {
    return when {
        isSuccess -> {
            @Suppress("UNCHECKED_CAST")
            this as Result<R>
        }
        else -> {
            val error = exceptionOrNull()!!
            logger.logFailure(error)
            Result.failure(transform(error))
        }
    }
}
