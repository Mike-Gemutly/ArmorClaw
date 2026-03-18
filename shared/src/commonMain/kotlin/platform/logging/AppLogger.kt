package com.armorclaw.shared.platform.logging

import com.armorclaw.shared.domain.model.ErrorCategory
import kotlinx.coroutines.flow.StateFlow
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*

/**
 * Centralized logging utility for ArmorClaw
 *
 * Provides structured logging with proper categorization,
 * making it easy to identify the source of errors and track
 * application behavior.
 *
 * Usage:
 * ```kotlin
 * class MyViewModel : ViewModel(), Loggable by AppLogger(LogTag.UI.CHAT) {
 *     fun doSomething() {
 *         logInfo("Starting operation")
 *         try {
 *             // ...
 *             logDebug("Operation completed", mapOf("duration" to "500ms"))
 *         } catch (e: Exception) {
 *             logError("Operation failed", e, mapOf("input" to input))
 *         }
 *     }
 * }
 * ```
 */
object AppLogger {

    private var crashReporter: CrashReportingAdapter? = null
    private var analytics: AnalyticsAdapter? = null
    private var isDebugMode: Boolean = false
    private var jsonOutputMode: JsonOutputMode = JsonOutputMode.NONE
    private var jsonOutputCallback: ((String) -> Unit)? = null

    /**
     * JSON output mode for structured logging
     */
    enum class JsonOutputMode {
        NONE,       // Standard text logging
        CONSOLE,    // Print JSON to console
        CALLBACK,   // Send JSON to callback function
        BOTH        // Both console and callback
    }

    /**
     * Initialize the logging system
     */
    fun initialize(
        crashReporter: CrashReportingAdapter? = null,
        analytics: AnalyticsAdapter? = null,
        isDebugMode: Boolean = false,
        jsonOutputMode: JsonOutputMode = JsonOutputMode.NONE,
        jsonOutputCallback: ((String) -> Unit)? = null
    ) {
        this.crashReporter = crashReporter
        this.analytics = analytics
        this.isDebugMode = isDebugMode
        this.jsonOutputMode = jsonOutputMode
        this.jsonOutputCallback = jsonOutputCallback
    }

    /**
     * Set debug mode for verbose logging
     */
    fun setDebugMode(enabled: Boolean) {
        isDebugMode = enabled
    }

    /**
     * Set JSON output mode for structured logging
     */
    fun setJsonOutputMode(mode: JsonOutputMode, callback: ((String) -> Unit)? = null) {
        jsonOutputMode = mode
        jsonOutputCallback = callback
    }

    /**
     * Create a logger for a specific tag
     */
    fun create(tag: LogTag): Loggable = LoggerImpl(tag)

    /**
     * Log a debug message
     */
    fun debug(tag: LogTag, message: String, data: Map<String, Any>? = null) {
        log(LogLevel.DEBUG, tag, message, null, data)
    }

    /**
     * Log an info message
     */
    fun info(tag: LogTag, message: String, data: Map<String, Any>? = null) {
        log(LogLevel.INFO, tag, message, null, data)
    }

    /**
     * Log a warning
     */
    fun warning(tag: LogTag, message: String, data: Map<String, Any>? = null) {
        log(LogLevel.WARNING, tag, message, null, data)
    }

    /**
     * Log an error
     */
    fun error(tag: LogTag, message: String, throwable: Throwable? = null, data: Map<String, Any>? = null) {
        log(LogLevel.ERROR, tag, message, throwable, data)
    }

    /**
     * Log a fatal error
     */
    fun fatal(tag: LogTag, message: String, throwable: Throwable? = null, data: Map<String, Any>? = null) {
        log(LogLevel.FATAL, tag, message, throwable, data)
    }

    /**
     * Log a performance metric
     */
    fun performance(tag: LogTag, operation: String, durationMs: Long, data: Map<String, Any>? = null) {
        val perfData = mapOf("duration_ms" to durationMs) + (data ?: emptyMap())
        log(LogLevel.PERFORMANCE, tag, "Performance: $operation", null, perfData)
    }

    /**
     * Log a network request
     */
    fun networkRequest(
        tag: LogTag,
        method: String,
        url: String,
        headers: Map<String, String>? = null
    ) {
        val data = mapOf(
            "method" to method,
            "url" to url,
            "headers" to (headers?.keys?.joinToString() ?: "none")
        )
        log(LogLevel.NETWORK, tag, "Request: $method $url", null, data)
    }

    /**
     * Log a network response
     */
    fun networkResponse(
        tag: LogTag,
        method: String,
        url: String,
        statusCode: Int,
        durationMs: Long
    ) {
        val data = mapOf(
            "method" to method,
            "url" to url,
            "status_code" to statusCode,
            "duration_ms" to durationMs
        )
        log(LogLevel.NETWORK, tag, "Response: $statusCode $method $url (${durationMs}ms)", null, data)
    }

    /**
     * Add a breadcrumb for crash reporting
     */
    fun breadcrumb(message: String, category: String, data: Map<String, Any>? = null) {
        crashReporter?.addBreadcrumb(message, category, data ?: emptyMap())
    }

    private fun log(
        level: LogLevel,
        tag: LogTag,
        message: String,
        throwable: Throwable?,
        data: Map<String, Any>?
    ) {
        val formattedMessage = formatMessage(level, tag, message, data)
        val timestamp = Clock.System.now()

        // Output JSON if configured
        if (jsonOutputMode != JsonOutputMode.NONE) {
            val jsonLog = formatAsJson(level, tag, message, throwable, data, timestamp)
            when (jsonOutputMode) {
                JsonOutputMode.CONSOLE -> println(jsonLog)
                JsonOutputMode.CALLBACK -> jsonOutputCallback?.invoke(jsonLog)
                JsonOutputMode.BOTH -> {
                    println(jsonLog)
                    jsonOutputCallback?.invoke(jsonLog)
                }
                JsonOutputMode.NONE -> { /* no-op */ }
            }
        }

        // Platform-specific output (always do this for backward compatibility)
        when (level) {
            LogLevel.DEBUG -> {
                if (isDebugMode) {
                    PlatformLogger.debug(tag.fullTag, formattedMessage, throwable)
                }
            }
            LogLevel.INFO -> PlatformLogger.info(tag.fullTag, formattedMessage)
            LogLevel.WARNING -> PlatformLogger.warning(tag.fullTag, formattedMessage, throwable)
            LogLevel.ERROR -> {
                PlatformLogger.error(tag.fullTag, formattedMessage, throwable)
                reportToCrashReporter(level, tag, message, throwable, data)
            }
            LogLevel.FATAL -> {
                PlatformLogger.error(tag.fullTag, formattedMessage, throwable)
                reportToCrashReporter(level, tag, message, throwable, data)
            }
            LogLevel.PERFORMANCE -> {
                if (isDebugMode) {
                    PlatformLogger.info(tag.fullTag, formattedMessage)
                }
            }
            LogLevel.NETWORK -> {
                if (isDebugMode) {
                    PlatformLogger.debug(tag.fullTag, formattedMessage, throwable)
                }
            }
        }

        // Track analytics for errors
        if (level >= LogLevel.ERROR) {
            analytics?.trackError(
                errorName = "${tag.category}_${level.levelName}",
                errorMessage = message,
                properties = buildMap {
                    put("tag", tag.fullTag)
                    put("category", tag.category)
                    put("module", tag.module)
                    throwable?.let { put("exception_type", it::class.simpleName ?: "Unknown") }
                    data?.let { putAll(it.mapKeys { "data_${it.key}" }) }
                }
            )
        }
    }

    /**
     * Format log entry as JSON for log aggregation systems
     */
    private fun formatAsJson(
        level: LogLevel,
        tag: LogTag,
        message: String,
        throwable: Throwable?,
        data: Map<String, Any>?,
        timestamp: Instant
    ): String {
        val json = Json { encodeDefaults = true }
        val logEntry = buildJsonObject {
            put("@timestamp", timestamp.toString())
            put("@version", "1")
            put("level", level.levelName)
            put("logger_name", tag.fullTag)
            put("message", message)
            put("thread_name", Thread.currentThread().name)

            // Tag structure
            putJsonObject("tag") {
                put("category", tag.category)
                put("module", tag.module)
                put("component", tag.component)
                put("full", tag.fullTag)
            }

            // Additional data
            data?.let { d ->
                putJsonObject("data") {
                    d.forEach { (key, value) ->
                        put(key, value.toString())
                    }
                }
            }

            // Exception info
            throwable?.let { t ->
                putJsonObject("exception") {
                    put("class", t::class.qualifiedName ?: t::class.simpleName ?: "Unknown")
                    put("message", t.message ?: "")
                    put("stack_trace", t.stackTraceToString())
                    t.cause?.let { cause ->
                        put("cause_class", cause::class.qualifiedName ?: cause::class.simpleName ?: "Unknown")
                        put("cause_message", cause.message ?: "")
                    }
                }
            }

            // App context
            putJsonObject("context") {
                put("app", "ArmorClaw")
                put("platform", "Android")
                put("debug_mode", isDebugMode)
            }
        }

        return json.encodeToString(logEntry)
    }

    private fun formatMessage(
        level: LogLevel,
        tag: LogTag,
        message: String,
        data: Map<String, Any>?
    ): String {
        val dataStr = data?.let {
            it.entries.joinToString(", ", "[", "]") { "${it.key}=${it.value}" }
        } ?: ""
        return "$message $dataStr".trim()
    }

    private fun reportToCrashReporter(
        level: LogLevel,
        tag: LogTag,
        message: String,
        throwable: Throwable?,
        data: Map<String, Any>?
    ) {
        crashReporter?.let { reporter ->
            val tags = mapOf(
                "log_level" to level.levelName,
                "category" to tag.category,
                "module" to tag.module
            )

            when {
                throwable != null -> reporter.captureException(throwable, tags + (data ?: emptyMap()).mapValues { it.value.toString() })
                else -> reporter.captureMessage(message, level.toCrashSeverity(), tags)
            }
        }

        // Track in ErrorAnalytics for rate monitoring
        data?.let { d ->
            val code = d["code"]?.toString() ?: "UNKNOWN"
            val source = "${tag.category}:${tag.module}"
            val category = try {
                ErrorCategory.valueOf(d["category"]?.toString() ?: "UNKNOWN")
            } catch (e: Exception) {
                ErrorCategory.UNKNOWN
            }
            val correlationId = d["correlation_id"]?.toString()
            val traceId = d["trace_id"]?.toString()

            ErrorAnalytics.trackError(
                code = code,
                source = source,
                category = category,
                message = message,
                correlationId = correlationId,
                traceId = traceId
            )
        }
    }

    /**
     * Logger implementation for a specific tag
     */
    private class LoggerImpl(private val tag: LogTag) : Loggable {
        override fun logDebug(message: String, data: Map<String, Any>?) {
            debug(tag, message, data)
        }

        override fun logInfo(message: String, data: Map<String, Any>?) {
            info(tag, message, data)
        }

        override fun logWarning(message: String, data: Map<String, Any>?) {
            warning(tag, message, data)
        }

        override fun logError(message: String, throwable: Throwable?, data: Map<String, Any>?) {
            error(tag, message, throwable, data)
        }

        override fun logFatal(message: String, throwable: Throwable?, data: Map<String, Any>?) {
            fatal(tag, message, throwable, data)
        }

        override fun logPerformance(operation: String, durationMs: Long, data: Map<String, Any>?) {
            performance(tag, operation, durationMs, data)
        }
    }
}

/**
 * Log levels
 */
enum class LogLevel(val levelName: String) {
    DEBUG("DEBUG"),
    INFO("INFO"),
    WARNING("WARNING"),
    ERROR("ERROR"),
    FATAL("FATAL"),
    PERFORMANCE("PERF"),
    NETWORK("NETWORK");

    fun toCrashSeverity(): CrashSeverity = when (this) {
        DEBUG -> CrashSeverity.DEBUG
        INFO -> CrashSeverity.INFO
        WARNING -> CrashSeverity.WARNING
        ERROR -> CrashSeverity.ERROR
        FATAL -> CrashSeverity.FATAL
        PERFORMANCE -> CrashSeverity.INFO
        NETWORK -> CrashSeverity.DEBUG
    }
}

/**
 * Crash severity levels
 */
enum class CrashSeverity {
    DEBUG, INFO, WARNING, ERROR, FATAL
}

/**
 * Interface for crash reporting integration
 */
interface CrashReportingAdapter {
    fun captureException(throwable: Throwable, tags: Map<String, String>)
    fun captureMessage(message: String, level: CrashSeverity, tags: Map<String, String>)
    fun addBreadcrumb(message: String, category: String, data: Map<String, Any>)
}

/**
 * Interface for analytics integration
 */
interface AnalyticsAdapter {
    fun trackError(errorName: String, errorMessage: String, properties: Map<String, Any>)
}

/**
 * Interface for loggable components
 */
interface Loggable {
    fun logDebug(message: String, data: Map<String, Any>? = null)
    fun logInfo(message: String, data: Map<String, Any>? = null)
    fun logWarning(message: String, data: Map<String, Any>? = null)
    fun logError(message: String, throwable: Throwable? = null, data: Map<String, Any>? = null)
    fun logFatal(message: String, throwable: Throwable? = null, data: Map<String, Any>? = null)
    fun logPerformance(operation: String, durationMs: Long, data: Map<String, Any>? = null)
}

/**
 * Delegate for Loggable interface
 */
class LoggerDelegate(private val tag: LogTag) : Loggable {
    override fun logDebug(message: String, data: Map<String, Any>?) {
        AppLogger.debug(tag, message, data)
    }

    override fun logInfo(message: String, data: Map<String, Any>?) {
        AppLogger.info(tag, message, data)
    }

    override fun logWarning(message: String, data: Map<String, Any>?) {
        AppLogger.warning(tag, message, data)
    }

    override fun logError(message: String, throwable: Throwable?, data: Map<String, Any>?) {
        AppLogger.error(tag, message, throwable, data)
    }

    override fun logFatal(message: String, throwable: Throwable?, data: Map<String, Any>?) {
        AppLogger.fatal(tag, message, throwable, data)
    }

    override fun logPerformance(operation: String, durationMs: Long, data: Map<String, Any>?) {
        AppLogger.performance(tag, operation, durationMs, data)
    }
}

/**
 * Extension function to easily create a logger for a class
 */
fun Any.logger(tag: LogTag): Loggable = AppLogger.create(tag)
