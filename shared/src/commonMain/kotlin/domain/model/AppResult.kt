package com.armorclaw.shared.domain.model

/**
 * Represents the result of an operation with detailed error context
 *
 * This sealed class provides comprehensive error information including:
 * - Error code for categorization
 * - Error message for user display
 * - Technical details for debugging
 * - Source tag for identifying where the error occurred
 * - Timestamp for tracking
 */
sealed class AppResult<out T> {

    /**
     * Successful result with data
     */
    data class Success<T>(
        val data: T,
        val metadata: Map<String, Any> = emptyMap()
    ) : AppResult<T>()

    /**
     * Failed result with detailed error information
     */
    data class Error<T>(
        val error: AppError,
        val metadata: Map<String, Any> = emptyMap()
    ) : AppResult<T>()

    /**
     * Loading state
     */
    data class Loading<T>(
        val progress: Float = 0f,
        val message: String? = null
    ) : AppResult<T>()

    /**
     * Check if result is successful
     */
    val isSuccess: Boolean get() = this is Success

    /**
     * Check if result is an error
     */
    val isError: Boolean get() = this is Error

    /**
     * Check if result is loading
     */
    val isLoading: Boolean get() = this is Loading

    /**
     * Get data or null if not successful
     */
    fun getOrNull(): T? = (this as? Success)?.data

    /**
     * Get error or null if not an error
     */
    fun errorOrNull(): AppError? = (this as? Error)?.error

    /**
     * Map the success data to a new type
     */
    inline fun <R> map(transform: (T) -> R): AppResult<R> = when (this) {
        is Success -> Success(transform(data), metadata)
        is Error -> Error(error, metadata)
        is Loading -> Loading(progress, message)
    }

    /**
     * Execute action on success
     */
    inline fun onSuccess(action: (T) -> Unit): AppResult<T> {
        if (this is Success) action(data)
        return this
    }

    /**
     * Execute action on error
     */
    inline fun onError(action: (AppError) -> Unit): AppResult<T> {
        if (this is Error) action(error)
        return this
    }

    /**
     * Execute action on loading
     */
    inline fun onLoading(action: (Float, String?) -> Unit): AppResult<T> {
        if (this is Loading) action(progress, message)
        return this
    }

    /**
     * Get data or throw exception
     */
    fun getOrThrow(): T {
        return when (this) {
            is Success -> data
            is Error -> throw error.toException()
            is Loading -> throw IllegalStateException("Result is still loading")
        }
    }

    /**
     * Get data or default value
     */
    fun getOrDefault(default: @UnsafeVariance T): T = when (this) {
        is Success -> data
        else -> default
    }

    /**
     * Recover from error with a default value
     */
    fun recover(default: @UnsafeVariance T): AppResult<T> = when (this) {
        is Success -> this
        is Error -> Success(default, metadata)
        is Loading -> this
    }

    companion object {
        /**
         * Create a successful result
         */
        fun <T> success(data: T, metadata: Map<String, Any> = emptyMap()): AppResult<T> =
            Success(data, metadata)

        /**
         * Create an error result
         */
        fun <T> error(error: AppError, metadata: Map<String, Any> = emptyMap()): AppResult<T> =
            Error(error, metadata)

        /**
         * Create an error result from an exception
         */
        fun <T> error(
            exception: Throwable,
            code: ArmorClawErrorCode,
            source: String,
            userMessage: String? = null
        ): AppResult<T> = Error(
            AppError(
                code = code.code,
                message = userMessage ?: exception.message ?: "Unknown error",
                technicalMessage = exception.stackTraceToString(),
                source = source,
                cause = exception
            )
        )

        /**
         * Create a loading result
         */
        fun <T> loading(progress: Float = 0f, message: String? = null): AppResult<T> =
            Loading(progress, message)
    }
}

/**
 * Detailed error information
 */
data class AppError(
    /**
     * Error code for categorization
     */
    val code: String,

    /**
     * User-friendly error message
     */
    val message: String,

    /**
     * Technical message for debugging
     */
    val technicalMessage: String? = null,

    /**
     * Source tag identifying where the error occurred
     */
    val source: String,

    /**
     * Timestamp when the error occurred
     */
    val timestamp: kotlinx.datetime.Instant = kotlinx.datetime.Clock.System.now(),

    /**
     * Original exception that caused this error
     */
    val cause: Throwable? = null,

    /**
     * Additional metadata about the error
     */
    val metadata: Map<String, Any> = emptyMap(),

    /**
     * Whether this error is recoverable
     */
    val isRecoverable: Boolean = true,

    /**
     * Suggested recovery action
     */
    val recoveryAction: RecoveryAction? = null
) {
    /**
     * Convert to exception for throwing
     */
    fun toException(): AppException = AppException(this)

    /**
     * Get a log-friendly string representation
     */
    fun toLogString(): String = buildString {
        append("[$source] ")
        append(code)
        append(": ")
        append(message)
        technicalMessage?.let {
            append(" | Technical: ")
            append(it.take(200))
        }
        cause?.let {
            append(" | Cause: ")
            append(it::class.simpleName)
            append(": ")
            append(it.message)
        }
    }
}

/**
 * Suggested recovery actions for errors
 */
sealed class RecoveryAction {
    object Retry : RecoveryAction()
    object Login : RecoveryAction()
    object CheckNetwork : RecoveryAction()
    object RequestPermission : RecoveryAction()
    object RestartApp : RecoveryAction()
    object ContactSupport : RecoveryAction()
    data class Custom(val action: String, val label: String) : RecoveryAction()
}

/**
 * Exception wrapper for AppError
 */
class AppException(
    val error: AppError
) : Exception(error.message, error.cause) {
    override fun toString(): String = error.toLogString()
}

/**
 * Extension functions for easier error handling
 */
inline fun <T> runCatchingWithError(
    source: String,
    errorCode: ArmorClawErrorCode = ArmorClawErrorCode.UNKNOWN_ERROR,
    block: () -> T
): AppResult<T> {
    return try {
        AppResult.success(block())
    } catch (e: Exception) {
        AppResult.error(
            AppError(
                code = errorCode.code,
                message = e.message ?: errorCode.userMessage,
                technicalMessage = e.stackTraceToString(),
                source = source,
                cause = e
            )
        )
    }
}

suspend inline fun <T> runCatchingWithErrorSuspend(
    source: String,
    errorCode: ArmorClawErrorCode = ArmorClawErrorCode.UNKNOWN_ERROR,
    crossinline block: suspend () -> T
): AppResult<T> {
    return try {
        AppResult.success(block())
    } catch (e: Exception) {
        AppResult.error(
            AppError(
                code = errorCode.code,
                message = e.message ?: errorCode.userMessage,
                technicalMessage = e.stackTraceToString(),
                source = source,
                cause = e
            )
        )
    }
}
