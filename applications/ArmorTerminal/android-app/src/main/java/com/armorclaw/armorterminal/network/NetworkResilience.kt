package com.armorclaw.armorterminal.network

import kotlinx.coroutines.delay
import java.util.concurrent.ThreadLocalRandom
import kotlin.math.min
import kotlin.math.pow

/**
 * Network Resilience Utilities
 *
 * Provides retry logic, backoff calculation, and network error handling.
 */
object NetworkResilience {

    // Backoff configuration
    const val INITIAL_BACKOFF_MS = 1000L
    const val MAX_BACKOFF_MS = 30000L
    const val BACKOFF_MULTIPLIER = 2.0
    const val JITTER_FACTOR = 0.3

    // Retry configuration
    const val DEFAULT_MAX_RETRIES = 3
    const val NETWORK_TIMEOUT_MS = 30000L
}

/**
 * Calculate exponential backoff with jitter
 *
 * @param attempt Current attempt number (0-indexed)
 * @param initialDelay Initial delay in milliseconds
 * @param maxDelay Maximum delay in milliseconds
 * @param multiplier Backoff multiplier
 * @param jitterFactor Jitter factor (0.0 - 1.0)
 * @return Delay in milliseconds
 */
fun calculateBackoff(
    attempt: Int,
    initialDelay: Long = NetworkResilience.INITIAL_BACKOFF_MS,
    maxDelay: Long = NetworkResilience.MAX_BACKOFF_MS,
    multiplier: Double = NetworkResilience.BACKOFF_MULTIPLIER,
    jitterFactor: Double = NetworkResilience.JITTER_FACTOR
): Long {
    val exponential = initialDelay * multiplier.pow(attempt)
    val capped = min(exponential.toLong(), maxDelay)

    val jitter = capped * jitterFactor * (ThreadLocalRandom.current().nextDouble() * 2 - 1)
    return (capped + jitter).toLong().coerceAtLeast(100)
}

/**
 * Execute a block with retry logic
 *
 * @param maxRetries Maximum number of retry attempts
 * @param initialDelay Initial delay before first retry
 * @param maxDelay Maximum delay between retries
 * @param shouldRetry Predicate to determine if a retry should be attempted
 * @param block The suspend function to execute
 * @return Result of the block, or the last error if all retries failed
 */
suspend fun <T> retryWithBackoff(
    maxRetries: Int = NetworkResilience.DEFAULT_MAX_RETRIES,
    initialDelay: Long = NetworkResilience.INITIAL_BACKOFF_MS,
    maxDelay: Long = NetworkResilience.MAX_BACKOFF_MS,
    shouldRetry: (Exception) -> Boolean = { it is NetworkException },
    block: suspend () -> T
): Result<T> {
    var lastException: Exception? = null

    repeat(maxRetries + 1) { attempt ->
        try {
            return Result.success(block())
        } catch (e: Exception) {
            lastException = e

            // Check if we should retry
            if (attempt < maxRetries && shouldRetry(e)) {
                val delayMs = calculateBackoff(attempt, initialDelay, maxDelay)
                delay(delayMs)
            } else {
                // No more retries or error is not retryable
                return Result.failure(e)
            }
        }
    }

    return Result.failure(lastException ?: Exception("Unknown error"))
}

/**
 * Execute a block with timeout and retry
 */
suspend fun <T> withTimeoutAndRetry(
    timeoutMs: Long = NetworkResilience.NETWORK_TIMEOUT_MS,
    maxRetries: Int = NetworkResilience.DEFAULT_MAX_RETRIES,
    block: suspend () -> T
): Result<T> {
    return retryWithBackoff(maxRetries) {
        kotlinx.coroutines.withTimeout(timeoutMs) {
            block()
        }
    }
}

/**
 * Network exception types
 */
sealed class NetworkException(message: String, cause: Throwable? = null) : Exception(message, cause) {
    class ConnectionFailed(cause: Throwable? = null) : NetworkException("Connection failed", cause)
    class Timeout(cause: Throwable? = null) : NetworkException("Request timed out", cause)
    class ServerError(val code: Int, message: String) : NetworkException("Server error $code: $message")
    class AuthenticationFailed(message: String = "Authentication failed") : NetworkException(message)
    class CertificateError(message: String) : NetworkException("Certificate error: $message")
    class Offline : NetworkException("Device is offline")
    class RateLimited(val retryAfter: Long?) : NetworkException("Rate limited")
}

/**
 * Check if an exception is retryable
 */
fun isRetryable(e: Exception): Boolean {
    return when (e) {
        is NetworkException.ConnectionFailed -> true
        is NetworkException.Timeout -> true
        is NetworkException.ServerError -> e.code in 500..599
        is NetworkException.RateLimited -> true
        else -> false
    }
}

/**
 * Network condition check
 */
data class NetworkCondition(
    val isOnline: Boolean,
    val isWifi: Boolean,
    val isCellular: Boolean,
    val signalStrength: Int? = null // 0-4 bars
) {
    val isGoodConnection: Boolean
        get() = isOnline && (isWifi || (isCellular && (signalStrength ?: 0) >= 2))

    val shouldUseLowBandwidth: Boolean
        get() = isCellular && (signalStrength ?: 0) < 2
}

/**
 * Network state for UI
 */
sealed class NetworkState {
    object Online : NetworkState()
    object Offline : NetworkState()
    object Reconnecting : NetworkState()
    data class Error(val message: String) : NetworkState()

    val isConnected: Boolean
        get() = this is Online

    val shouldShowOfflineBanner: Boolean
        get() = this is Offline || this is Reconnecting
}
