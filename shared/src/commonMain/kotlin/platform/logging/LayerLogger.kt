package com.armorclaw.shared.platform.logging

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.flow.onCompletion
import kotlinx.coroutines.flow.onStart

/**
 * Layer-specific logging utilities for proper separation of concerns.
 *
 * Each layer (Repository, ViewModel, UseCase) has its own logging wrapper
 * that adds appropriate context and handles errors consistently.
 *
 * This ensures:
 * 1. Errors can be traced to their source layer
 * 2. Each layer logs at appropriate boundaries
 * 3. Context is automatically added to log messages
 */

// ==================== REPOSITORY LAYER LOGGER ====================

/**
 * Logger for Repository layer operations
 *
 * Automatically adds repository context to all log messages.
 */
class RepositoryLogger(
    private val tag: LogTag,
    private val repositoryName: String
) {
    /**
     * Log repository operation start
     */
    fun logOperationStart(operation: String, params: Map<String, Any?> = emptyMap()) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Starting: $operation",
            params.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log repository operation success
     */
    fun logOperationSuccess(operation: String, result: String? = null) {
        val message = if (result != null) {
            "[$repositoryName] Success: $operation -> $result"
        } else {
            "[$repositoryName] Success: $operation"
        }
        AppLogger.debug(tag, message)
    }

    /**
     * Log repository operation error
     */
    fun logOperationError(
        operation: String,
        error: Throwable,
        params: Map<String, Any?> = emptyMap()
    ) {
        AppLogger.error(
            tag,
            "[$repositoryName] Error in $operation: ${error.message}",
            error,
            params.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log debug message
     */
    fun logDebug(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.debug(
            tag,
            "[$repositoryName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log info message
     */
    fun logInfo(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.info(
            tag,
            "[$repositoryName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log warning message
     */
    fun logWarning(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.warning(
            tag,
            "[$repositoryName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log cache hit
     */
    fun logCacheHit(key: String) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Cache hit",
            mapOf("cacheKey" to key)
        )
    }

    /**
     * Log cache miss
     */
    fun logCacheMiss(key: String) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Cache miss",
            mapOf("cacheKey" to key)
        )
    }

    /**
     * Log network request
     */
    fun logNetworkRequest(endpoint: String, method: String) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Network request",
            mapOf("endpoint" to endpoint, "method" to method)
        )
    }

    /**
     * Log network request with additional metadata
     */
    fun logNetworkRequest(endpoint: String, method: String, metadata: Map<String, Any>) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Network request",
            mapOf("endpoint" to endpoint, "method" to method) + metadata
        )
    }

    /**
     * Log network response
     */
    fun logNetworkResponse(endpoint: String, statusCode: Int, durationMs: Long) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Network response",
            mapOf(
                "endpoint" to endpoint,
                "statusCode" to statusCode,
                "durationMs" to durationMs
            )
        )
    }

    /**
     * Log network response with additional metadata
     */
    fun logNetworkResponse(endpoint: String, statusCode: Int, durationMs: Long, metadata: Map<String, Any>) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Network response",
            mapOf(
                "endpoint" to endpoint,
                "statusCode" to statusCode,
                "durationMs" to durationMs
            ) + metadata
        )
    }

    /**
     * Log database query
     */
    fun logDatabaseQuery(query: String, params: Map<String, Any?> = emptyMap()) {
        AppLogger.debug(
            tag,
            "[$repositoryName] DB query: $query",
            params.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log data transformation
     */
    fun logTransformation(from: String, to: String) {
        AppLogger.debug(
            tag,
            "[$repositoryName] Transform: $from -> $to"
        )
    }
}

// ==================== USECASE LAYER LOGGER ====================

/**
 * Logger for UseCase layer operations
 *
 * Tracks business logic execution with proper context.
 */
class UseCaseLogger(
    private val tag: LogTag,
    private val useCaseName: String
) {
    /**
     * Log use case execution start
     */
    fun logStart(params: Map<String, Any?> = emptyMap()) {
        val sanitizedParams = params.mapValues { (_, value) ->
            when (value) {
                is String -> if (value.length > 50) "${value.take(50)}..." else value
                is PasswordMasked -> "***"
                is TokenMasked -> "***"
                else -> value?.toString() ?: "null"
            }
        }
        AppLogger.info(
            tag,
            "[$useCaseName] Executing",
            sanitizedParams.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log use case success
     */
    fun logSuccess(result: String? = null) {
        val message = if (result != null) {
            "[$useCaseName] Completed: $result"
        } else {
            "[$useCaseName] Completed successfully"
        }
        AppLogger.info(tag, message)
    }

    /**
     * Log use case failure
     */
    fun logFailure(error: Throwable, params: Map<String, Any?> = emptyMap()) {
        AppLogger.error(
            tag,
            "[$useCaseName] Failed: ${error.message}",
            error,
            params.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log validation error
     */
    fun logValidationError(field: String, reason: String) {
        AppLogger.warning(
            tag,
            "[$useCaseName] Validation failed",
            mapOf("field" to field, "reason" to reason)
        )
    }

    /**
     * Log business rule violation
     */
    fun logBusinessRuleViolation(rule: String, details: String) {
        AppLogger.warning(
            tag,
            "[$useCaseName] Business rule violated: $rule",
            mapOf("details" to details)
        )
    }

    /**
     * Log execution time
     */
    fun logExecutionTime(durationMs: Long) {
        AppLogger.debug(
            tag,
            "[$useCaseName] Execution time",
            mapOf("durationMs" to durationMs)
        )
    }

    /**
     * Log debug message
     */
    fun logDebug(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.debug(
            tag,
            "[$useCaseName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log info message
     */
    fun logInfo(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.info(
            tag,
            "[$useCaseName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log warning message
     */
    fun logWarning(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.warning(
            tag,
            "[$useCaseName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }
}

/**
 * Mask for sensitive password data
 */
data class PasswordMasked(val value: String)
/**
 * Mask for sensitive token data
 */
data class TokenMasked(val value: String)

// ==================== VIEWMODEL LAYER LOGGER ====================

/**
 * Logger for ViewModel layer
 *
 * Tracks UI state changes, user actions, and UI events.
 */
class ViewModelLogger(
    private val tag: LogTag,
    private val viewModelName: String
) {
    /**
     * Log ViewModel initialization
     */
    fun logInit(params: Map<String, Any?> = emptyMap()) {
        AppLogger.info(
            tag,
            "[$viewModelName] Initialized",
            params.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log ViewModel cleanup
     */
    fun logCleanup() {
        AppLogger.info(tag, "[$viewModelName] Cleaned up")
    }

    /**
     * Log user action
     */
    fun logUserAction(action: String, params: Map<String, Any?> = emptyMap()) {
        AppLogger.debug(
            tag,
            "[$viewModelName] User action: $action",
            params.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log state change
     */
    fun logStateChange(stateName: String, value: String? = null) {
        val message = if (value != null) {
            "[$viewModelName] State: $stateName = $value"
        } else {
            "[$viewModelName] State: $stateName changed"
        }
        AppLogger.debug(tag, message)
    }

    /**
     * Log UI event emission
     */
    fun logUiEvent(event: String) {
        AppLogger.debug(
            tag,
            "[$viewModelName] UI Event: $event"
        )
    }

    /**
     * Log error in ViewModel
     */
    fun logError(
        operation: String,
        error: Throwable,
        params: Map<String, Any?> = emptyMap()
    ) {
        AppLogger.error(
            tag,
            "[$viewModelName] Error in $operation: ${error.message}",
            error,
            params.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log navigation event
     */
    fun logNavigation(destination: String) {
        AppLogger.info(
            tag,
            "[$viewModelName] Navigate to: $destination"
        )
    }

    /**
     * Log info message
     */
    fun logInfo(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.info(
            tag,
            "[$viewModelName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }

    /**
     * Log debug message
     */
    fun logDebug(message: String, data: Map<String, Any?> = emptyMap()) {
        AppLogger.debug(
            tag,
            "[$viewModelName] $message",
            data.filterValues { it != null }.mapValues { it.value!! }
        )
    }
}

// ==================== SERVICE LAYER LOGGER ====================

/**
 * Logger for Service/Background operations
 */
class ServiceLogger(
    private val tag: LogTag,
    private val serviceName: String
) {
    /**
     * Log service start
     */
    fun logStart() {
        AppLogger.info(tag, "[$serviceName] Service started")
    }

    /**
     * Log service stop
     */
    fun logStop(reason: String? = null) {
        val message = if (reason != null) {
            "[$serviceName] Service stopped: $reason"
        } else {
            "[$serviceName] Service stopped"
        }
        AppLogger.info(tag, message)
    }

    /**
     * Log task execution
     */
    fun logTaskStart(taskName: String) {
        AppLogger.debug(tag, "[$serviceName] Task started: $taskName")
    }

    /**
     * Log task completion
     */
    fun logTaskComplete(taskName: String, success: Boolean) {
        AppLogger.debug(
            tag,
            "[$serviceName] Task completed: $taskName",
            mapOf("success" to success)
        )
    }

    /**
     * Log background error
     */
    fun logError(operation: String, error: Throwable) {
        AppLogger.error(
            tag,
            "[$serviceName] Error in $operation: ${error.message}",
            error
        )
    }
}

// ==================== FLOW EXTENSIONS ====================

/**
 * Extension to add logging to Flow operations
 */
fun <T> Flow<T>.logFlow(
    logger: UseCaseLogger,
    operationName: String
): Flow<T> = this
    .onStart {
        logger.logStart(mapOf("operation" to operationName))
    }
    .onCompletion { cause ->
        if (cause == null) {
            logger.logSuccess("Flow completed")
        } else {
            logger.logFailure(cause)
        }
    }
    .catch { error ->
        logger.logFailure(error)
        throw error
    }

// ==================== RESULT WRAPPER FOR LOGGING ====================

/**
 * Wraps a suspending operation with logging
 */
suspend inline fun <T> withRepositoryLogging(
    logger: RepositoryLogger,
    operation: String,
    crossinline block: suspend () -> T
): Result<T> {
    return try {
        logger.logOperationStart(operation)
        val result = block()
        logger.logOperationSuccess(operation)
        Result.success(result)
    } catch (e: Exception) {
        logger.logOperationError(operation, e)
        Result.failure(e)
    }
}

/**
 * Wraps a suspending use case operation with logging
 */
suspend inline fun <T> withUseCaseLogging(
    logger: UseCaseLogger,
    crossinline block: suspend () -> T
): Result<T> {
    val startTime = System.currentTimeMillis()
    return try {
        logger.logStart()
        val result = block()
        logger.logSuccess()
        logger.logExecutionTime(System.currentTimeMillis() - startTime)
        Result.success(result)
    } catch (e: Exception) {
        logger.logFailure(e)
        Result.failure(e)
    }
}

/**
 * Wraps a use case operation with logging - simplified version
 */
inline fun <T> withUseCase(
    logger: UseCaseLogger,
    operation: String,
    block: () -> Result<T>
): Result<T> {
    return try {
        logger.logStart(mapOf("operation" to operation))
        val result = block()
        if (result.isSuccess) {
            logger.logSuccess()
        } else {
            result.exceptionOrNull()?.let { logger.logFailure(it) }
        }
        result
    } catch (e: Exception) {
        logger.logFailure(e)
        Result.failure(e)
    }
}
