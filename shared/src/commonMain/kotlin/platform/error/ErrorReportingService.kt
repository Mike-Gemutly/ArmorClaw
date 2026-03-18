package com.armorclaw.shared.platform.error

import com.armorclaw.shared.domain.model.ArmorClawErrorCode
import com.armorclaw.shared.domain.model.ArmorClawError
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * Centralized error reporting service for ArmorClaw
 *
 * Handles:
 * - Error collection and aggregation
 * - Error reporting to ArmorClaw servers
 * - Local error storage for debugging
 * - Error rate limiting
 * - Privacy-preserving error data
 */
class ErrorReportingService(
    private val config: ErrorReportingConfig = ErrorReportingConfig()
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val json = Json { ignoreUnknownKeys = true }

    private val _errors = MutableSharedFlow<ReportedError>(extraBufferCapacity = 100)
    val errors: SharedFlow<ReportedError> = _errors.asSharedFlow()

    // Error queue for batching
    private val errorQueue = mutableListOf<ReportedError>()
    private var lastFlushTime = System.currentTimeMillis()

    /**
     * Report an error to ArmorClaw
     *
     * This method:
     * 1. Logs the error locally
     * 2. Adds to batch queue
     * 3. Flushes if batch size exceeded
     * 4. Emits error for UI handling
     */
    fun reportError(
        error: ArmorClawError,
        context: ErrorContext = ErrorContext(),
        immediate: Boolean = false
    ) {
        val reportedError = ReportedError(
            id = generateErrorId(),
            error = error,
            context = context,
            timestamp = System.currentTimeMillis(),
            appVersion = config.appVersion,
            platform = config.platform,
            deviceId = config.deviceId
        )

        // Log locally
        AppLogger.error(
            LogTag.CrashReporting.Capture,
            "[${error.code.code}] ${error.code.userMessage}",
            context.originalError,
            mapOf(
                "errorCode" to error.code.code,
                "category" to error.code.category.name,
                "recoverable" to error.code.recoverable,
                "details" to (error.details ?: ""),
                "operation" to (context.operation ?: "unknown"),
                "layer" to (context.layer ?: "unknown")
            )
        )

        // Add breadcrumb for crash context
        AppLogger.breadcrumb(
            message = "Error: ${error.code.code}",
            category = "error",
            data = mapOf(
                "operation" to (context.operation ?: "unknown"),
                "layer" to (context.layer ?: "unknown")
            )
        )

        // Add to queue
        synchronized(errorQueue) {
            errorQueue.add(reportedError)
        }

        // Emit for UI handling
        scope.launch {
            _errors.emit(reportedError)
        }

        // Flush if needed
        if (immediate || errorQueue.size >= config.batchSize) {
            flushErrors()
        }
    }

    /**
     * Report an exception directly
     */
    fun reportException(
        exception: Throwable,
        errorCode: ArmorClawErrorCode = ArmorClawErrorCode.UNKNOWN_ERROR,
        context: ErrorContext = ErrorContext()
    ) {
        val error = ArmorClawError(
            code = errorCode,
            details = exception.message,
            timestamp = System.currentTimeMillis()
        )

        reportError(
            error = error,
            context = context.copy(originalError = exception)
        )
    }

    /**
     * Report a message-related error
     */
    fun reportMessageError(
        messageId: String,
        roomId: String,
        errorCode: ArmorClawErrorCode,
        details: String? = null,
        originalError: Throwable? = null
    ) {
        val error = ArmorClawError(
            code = errorCode,
            details = details,
            timestamp = System.currentTimeMillis()
        )

        val context = ErrorContext(
            operation = "message",
            layer = "domain",
            messageId = messageId,
            roomId = roomId,
            originalError = originalError
        )

        reportError(error, context)
    }

    /**
     * Flush all queued errors to ArmorClaw servers
     */
    fun flushErrors() {
        val now = System.currentTimeMillis()

        // Check rate limiting
        if (now - lastFlushTime < config.minFlushIntervalMs) {
            return
        }

        val errorsToSend: List<ReportedError>

        synchronized(errorQueue) {
            if (errorQueue.isEmpty()) return

            errorsToSend = errorQueue.toList()
            errorQueue.clear()
        }

        lastFlushTime = now

        // Send to server
        scope.launch {
            sendErrorsToServer(errorsToSend)
        }
    }

    /**
     * Send errors to ArmorClaw server
     */
    private suspend fun sendErrorsToServer(errors: List<ReportedError>) {
        if (!config.enabled || errors.isEmpty()) return

        try {
            val payload = ErrorBatchPayload(
                deviceId = config.deviceId,
                appVersion = config.appVersion,
                platform = config.platform,
                timestamp = System.currentTimeMillis(),
                errors = errors.map { it.toWireFormat() }
            )

            val jsonPayload = json.encodeToString(payload)

            AppLogger.debug(
                LogTag.CrashReporting.Capture,
                "Sending ${errors.size} errors to server",
                mapOf("payloadSize" to jsonPayload.length)
            )

            // In a real implementation, this would use an HTTP client
            // httpClient.post("${config.serverUrl}/api/v1/errors", jsonPayload)

            // For now, just log it
            AppLogger.info(
                LogTag.CrashReporting.Capture,
                "Errors would be sent to: ${config.serverUrl}",
                mapOf("count" to errors.size)
            )

        } catch (e: Exception) {
            AppLogger.error(
                LogTag.CrashReporting.Capture,
                "Failed to send errors to server: ${e.message}",
                e
            )
        }
    }

    private fun generateErrorId(): String {
        return "err_${System.currentTimeMillis()}_${(1000..9999).random()}"
    }

    companion object {
        @Volatile
        private var instance: ErrorReportingService? = null

        fun getInstance(config: ErrorReportingConfig = ErrorReportingConfig()): ErrorReportingService {
            return instance ?: synchronized(this) {
                instance ?: ErrorReportingService(config).also { instance = it }
            }
        }
    }
}

/**
 * Configuration for error reporting
 */
@Serializable
data class ErrorReportingConfig(
    val enabled: Boolean = true,
    val serverUrl: String = "https://api.armorclaw.app",
    val appVersion: String = "1.0.0",
    val platform: String = "android",
    val deviceId: String = "unknown",
    val batchSize: Int = 10,
    val minFlushIntervalMs: Long = 30_000, // 30 seconds
    val includeStackTraces: Boolean = true,
    val includeDeviceData: Boolean = true
)

/**
 * Context for an error
 */
@Serializable
data class ErrorContext(
    val operation: String? = null,
    val layer: String? = null,
    val messageId: String? = null,
    val roomId: String? = null,
    val userId: String? = null,
    val additionalData: Map<String, String> = emptyMap(),
    @kotlinx.serialization.Transient
    val originalError: Throwable? = null
)

/**
 * A reported error with full context
 */
@Serializable
data class ReportedError(
    val id: String,
    val error: ArmorClawError,
    val context: ErrorContext,
    val timestamp: Long,
    val appVersion: String,
    val platform: String,
    val deviceId: String
) {
    fun toWireFormat(): WireError = WireError(
        id = id,
        code = error.code.code,
        message = error.code.userMessage,
        details = error.details,
        category = error.code.category.name,
        operation = context.operation,
        layer = context.layer,
        messageId = context.messageId,
        roomId = context.roomId,
        timestamp = timestamp,
        appVersion = appVersion,
        platform = platform,
        deviceId = deviceId,
        stackTrace = context.originalError?.stackTraceToString()
    )
}

/**
 * Wire format for error reporting
 */
@Serializable
data class WireError(
    val id: String,
    val code: String,
    val message: String,
    val details: String?,
    val category: String,
    val operation: String?,
    val layer: String?,
    val messageId: String?,
    val roomId: String?,
    val timestamp: Long,
    val appVersion: String,
    val platform: String,
    val deviceId: String,
    val stackTrace: String?
)

/**
 * Batch payload for error reporting
 */
@Serializable
data class ErrorBatchPayload(
    val deviceId: String,
    val appVersion: String,
    val platform: String,
    val timestamp: Long,
    val errors: List<WireError>
)
