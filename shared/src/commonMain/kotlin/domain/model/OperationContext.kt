package com.armorclaw.shared.domain.model

import kotlin.random.Random

/**
 * Operation context for tracing operations across layers
 *
 * Provides correlation IDs and metadata for debugging and logging.
 * This context should be passed through the entire operation chain:
 * ViewModel → UseCase → Repository
 *
 * Example usage:
 * ```kotlin
 * // In ViewModel
 * fun sendMessage(content: String) {
 *     val context = OperationContext.create(userId = currentUserId)
 *     viewModelScope.launch {
 *         sendMessageUseCase(content, context)
 *     }
 * }
 *
 * // In logs:
 * // [VM/Chat] User action: sendMessage {correlationId=abc-123}
 * // [UseCase/Message/Send] Executing {correlationId=abc-123}
 * // [Data/Repository/Message] Starting: insert {correlationId=abc-123}
 * ```
 */
data class OperationContext(
    val correlationId: String,
    val causationId: String? = null,    // ID of operation that caused this one
    val userId: String? = null,
    val sessionId: String? = null,
    val deviceId: String? = null,
    val traceId: String? = null,        // For distributed tracing across services
    val metadata: Map<String, Any?> = emptyMap()
) {
    companion object {
        /**
         * Create a new operation context with a generated correlation ID and trace ID
         */
        fun create(
            userId: String? = null,
            sessionId: String? = null,
            deviceId: String? = null,
            metadata: Map<String, Any?> = emptyMap()
        ): OperationContext {
            return OperationContext(
                correlationId = generateCorrelationId(),
                traceId = generateTraceId(),  // Auto-generate trace ID for distributed tracing
                userId = userId,
                sessionId = sessionId,
                deviceId = deviceId,
                metadata = metadata
            )
        }

        /**
         * Create a context from incoming request headers (for server-side or bridge)
         */
        fun fromHeaders(
            headers: Map<String, String>,
            userId: String? = null,
            sessionId: String? = null,
            deviceId: String? = null,
            metadata: Map<String, Any?> = emptyMap()
        ): OperationContext {
            return OperationContext(
                correlationId = headers["X-Correlation-ID"] ?: headers["X-Request-ID"] ?: generateCorrelationId(),
                traceId = headers["X-Trace-ID"] ?: generateTraceId(),
                userId = userId,
                sessionId = sessionId,
                deviceId = deviceId,
                metadata = metadata + mapOf("requestHeaders" to headers.keys)
            )
        }

        /**
         * Create a child context from a parent (for sub-operations)
         */
        fun childOf(
            parent: OperationContext,
            additionalMetadata: Map<String, Any?> = emptyMap()
        ): OperationContext {
            return OperationContext(
                correlationId = parent.correlationId,  // Same correlation ID
                causationId = generateCorrelationId(),  // New causation ID for this sub-operation
                userId = parent.userId,
                sessionId = parent.sessionId,
                deviceId = parent.deviceId,
                traceId = parent.traceId,
                metadata = parent.metadata + additionalMetadata
            )
        }

        /**
         * Create a context from an existing correlation ID (e.g., from network request)
         */
        fun fromCorrelationId(
            correlationId: String,
            userId: String? = null,
            metadata: Map<String, Any?> = emptyMap()
        ): OperationContext {
            return OperationContext(
                correlationId = correlationId,
                userId = userId,
                metadata = metadata
            )
        }

        /**
         * Generate a unique correlation ID
         * Format: UUID-like string (8-4-4-4-12 hex format)
         */
        private fun generateCorrelationId(): String {
            val bytes = Random.nextBytes(16)
            return buildString {
                bytes.take(4).forEach { append("%02x".format(it)) }
                append("-")
                bytes.drop(4).take(2).forEach { append("%02x".format(it)) }
                append("-")
                bytes.drop(6).take(2).forEach { append("%02x".format(it)) }
                append("-")
                bytes.drop(8).take(2).forEach { append("%02x".format(it)) }
                append("-")
                bytes.drop(10).take(6).forEach { append("%02x".format(it)) }
            }
        }

        /**
         * Generate a unique trace ID for distributed tracing
         * Format: Compatible with OpenTelemetry (32 hex characters)
         */
        private fun generateTraceId(): String {
            val timestamp = (System.currentTimeMillis() and 0xFFFFFFFFFFFF).toString(16).padStart(12, '0')
            val random = Random.nextBytes(10).joinToString("") { "%02x".format(it) }
            return timestamp + random
        }
    }

    /**
     * Add metadata to the context
     */
    fun withMetadata(key: String, value: Any?): OperationContext {
        return copy(metadata = metadata + (key to value))
    }

    /**
     * Add multiple metadata entries
     */
    fun withMetadata(additional: Map<String, Any?>): OperationContext {
        return copy(metadata = metadata + additional)
    }

    /**
     * Convert to log-friendly map
     */
    fun toLogMap(): Map<String, Any?> {
        val map = mutableMapOf<String, Any?>(
            "correlationId" to correlationId
        )
        causationId?.let { map["causationId"] = it }
        userId?.let { map["userId"] = it }
        sessionId?.let { map["sessionId"] = it }
        deviceId?.let { map["deviceId"] = it }
        traceId?.let { map["traceId"] = it }
        if (metadata.isNotEmpty()) {
            map["metadata"] = metadata
        }
        return map
    }

    /**
     * Short correlation ID for display (first 8 characters)
     */
    val shortCorrelationId: String
        get() = correlationId.take(8)
}

/**
 * Extension to add context to a map
 */
fun Map<String, Any?>.withContext(context: OperationContext): Map<String, Any?> {
    return this + context.toLogMap()
}
