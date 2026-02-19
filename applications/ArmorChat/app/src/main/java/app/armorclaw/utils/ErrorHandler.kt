package app.armorclaw.utils

/**
 * Represents a structured error from the bridge API
 */
data class BridgeError(
    val code: ErrorCode,
    val title: String,
    val message: String,
    val recoverable: Boolean,
    val retryAfterMs: Long = 0
) : Exception(message)

/**
 * Error codes for bridge operations
 */
enum class ErrorCode {
    // Network errors
    NETWORK_UNAVAILABLE,
    CONNECTION_REFUSED,
    TIMEOUT,

    // Bridge errors
    BRIDGE_NOT_FOUND,
    BRIDGE_UNAVAILABLE,
    BRIDGE_VERSION_MISMATCH,

    // Authentication errors
    AUTH_REQUIRED,
    AUTH_INVALID,
    SESSION_EXPIRED,
    PERMISSION_DENIED,

    // Setup errors
    ALREADY_CLAIMED,
    INVALID_PASSPHRASE,
    DEVICE_NOT_TRUSTED,

    // Validation errors
    INVALID_INPUT,
    MISSING_FIELD,

    // Server errors
    INTERNAL_ERROR,
    RATE_LIMITED,

    // Unknown
    UNKNOWN
}

/**
 * Maps raw exceptions to structured BridgeError
 */
object ErrorHandler {

    fun mapError(error: Throwable): BridgeError {
        return when {
            // Network unavailable
            error is java.net.UnknownHostException -> BridgeError(
                code = ErrorCode.NETWORK_UNAVAILABLE,
                title = "Network Unavailable",
                message = "Unable to connect to the network. Please check your connection.",
                recoverable = true,
                retryAfterMs = 3000
            )

            // Connection refused
            error is java.net.ConnectException -> BridgeError(
                code = ErrorCode.CONNECTION_REFUSED,
                title = "Connection Failed",
                message = "Unable to reach the ArmorClaw bridge. Is it running?",
                recoverable = true,
                retryAfterMs = 5000
            )

            // Timeout
            error is java.net.SocketTimeoutException -> BridgeError(
                code = ErrorCode.TIMEOUT,
                title = "Connection Timeout",
                message = "The request took too long. Please try again.",
                recoverable = true,
                retryAfterMs = 2000
            )

            // Parse error message for known codes
            else -> parseErrorMessage(error.message ?: "")
        }
    }

    private fun parseErrorMessage(message: String): BridgeError {
        val lowerMessage = message.lowercase()

        return when {
            // Already claimed
            lowerMessage.contains("already claimed") || lowerMessage.contains("admin established") -> BridgeError(
                code = ErrorCode.ALREADY_CLAIMED,
                title = "Device Already Claimed",
                message = "This device has already been claimed by an administrator.",
                recoverable = false
            )

            // Invalid passphrase
            lowerMessage.contains("invalid passphrase") || lowerMessage.contains("invalid credentials") -> BridgeError(
                code = ErrorCode.INVALID_PASSPHRASE,
                title = "Invalid Passphrase",
                message = "The passphrase you entered is incorrect.",
                recoverable = true
            )

            // Device not trusted
            lowerMessage.contains("not trusted") || lowerMessage.contains("untrusted device") -> BridgeError(
                code = ErrorCode.DEVICE_NOT_TRUSTED,
                title = "Device Not Trusted",
                message = "This device is not trusted. Please contact an administrator.",
                recoverable = false
            )

            // Session expired
            lowerMessage.contains("session") && lowerMessage.contains("expired") -> BridgeError(
                code = ErrorCode.SESSION_EXPIRED,
                title = "Session Expired",
                message = "Your session has expired. Please log in again.",
                recoverable = false
            )

            // Auth required
            lowerMessage.contains("401") || lowerMessage.contains("unauthorized") -> BridgeError(
                code = ErrorCode.AUTH_REQUIRED,
                title = "Authentication Required",
                message = "Authentication required. Please log in.",
                recoverable = false
            )

            // Permission denied
            lowerMessage.contains("403") || lowerMessage.contains("forbidden") -> BridgeError(
                code = ErrorCode.PERMISSION_DENIED,
                title = "Permission Denied",
                message = "You do not have permission to perform this action.",
                recoverable = false
            )

            // Rate limited
            lowerMessage.contains("429") || lowerMessage.contains("rate limit") -> BridgeError(
                code = ErrorCode.RATE_LIMITED,
                title = "Too Many Requests",
                message = "Too many requests. Please wait a moment.",
                recoverable = true,
                retryAfterMs = 10000
            )

            // Server error
            lowerMessage.contains("500") || lowerMessage.contains("internal") -> BridgeError(
                code = ErrorCode.INTERNAL_ERROR,
                title = "Server Error",
                message = "An internal error occurred. Please try again.",
                recoverable = true,
                retryAfterMs = 3000
            )

            // Unknown error
            else -> BridgeError(
                code = ErrorCode.UNKNOWN,
                title = "Error",
                message = "An unexpected error occurred. Please try again.",
                recoverable = true,
                retryAfterMs = 3000
            )
        }
    }

    /**
     * Returns a user-friendly action suggestion based on error code
     */
    fun getSuggestedAction(error: BridgeError): String? {
        return when (error.code) {
            ErrorCode.NETWORK_UNAVAILABLE -> "Check your WiFi or mobile data connection"
            ErrorCode.CONNECTION_REFUSED -> "Make sure the bridge is running on your server"
            ErrorCode.TIMEOUT -> "Try again with a stronger connection"
            ErrorCode.INVALID_PASSPHRASE -> "Double-check your passphrase and try again"
            ErrorCode.DEVICE_NOT_TRUSTED -> "Ask an admin to approve your device"
            ErrorCode.RATE_LIMITED -> "Wait a few seconds before trying again"
            else -> null
        }
    }
}

/**
 * Retry helper with exponential backoff
 */
suspend fun <T> retryWithBackoff(
    maxAttempts: Int = 3,
    baseDelayMs: Long = 1000,
    maxDelayMs: Long = 30000,
    shouldRetry: (BridgeError) -> Boolean = { it.recoverable },
    block: suspend () -> T
): T {
    var lastError: BridgeError? = null

    for (attempt in 1..maxAttempts) {
        try {
            return block()
        } catch (e: Throwable) {
            val bridgeError = ErrorHandler.mapError(e)
            lastError = bridgeError

            if (attempt == maxAttempts || !shouldRetry(bridgeError)) {
                throw bridgeError
            }

            val delay = minOf(
                bridgeError.retryAfterMs.takeIf { it > 0 } ?: baseDelayMs * (1 shl (attempt - 1)),
                maxDelayMs
            )
            kotlinx.coroutines.delay(delay)
        }
    }

    throw lastError ?: BridgeError(
        code = ErrorCode.UNKNOWN,
        title = "Unknown Error",
        message = "An unexpected error occurred",
        recoverable = false
    )
}
