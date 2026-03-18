package com.armorclaw.shared.platform.logging

import com.armorclaw.shared.domain.model.ArmorClawErrorCode
import com.armorclaw.shared.domain.model.ErrorCategory
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlin.math.ceil
import kotlin.time.Duration
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds

/**
 * Error rate tracking and analytics for monitoring application health
 *
 * Tracks error rates by category, code, and source to detect anomalies
 * and provide alerts when thresholds are exceeded.
 *
 * Usage:
 * ```kotlin
 * // Initialize with thresholds
 * ErrorAnalytics.initialize(
 *     errorRateThreshold = 10.0,  // 10 errors per minute triggers warning
 *     criticalRateThreshold = 50.0, // 50 errors per minute triggers critical
 *     alertCallback = { alert -> handleAlert(alert) }
 * )
 *
 * // Track errors
 * ErrorAnalytics.trackError(
 *     code = "E013",
 *     source = "Repository:sendMessage",
 *     category = ErrorCategory.NETWORK
 * )
 *
 * // Get statistics
 * val stats = ErrorAnalytics.getStatistics()
 * ```
 */
object ErrorAnalytics {

    private var isInitialized = false
    private var errorRateThreshold: Double = 10.0  // errors per minute
    private var criticalRateThreshold: Double = 50.0
    private var alertCallback: ((ErrorAlert) -> Unit)? = null
    private var windowDuration: Duration = 1.minutes

    private val errorEvents = mutableListOf<ErrorEvent>()
    private val errorEventsLock = Any()

    private val _statistics = MutableStateFlow(ErrorStatistics())
    val statistics: StateFlow<ErrorStatistics> = _statistics.asStateFlow()

    private val _activeAlerts = MutableStateFlow<List<ErrorAlert>>(emptyList())
    val activeAlerts: StateFlow<List<ErrorAlert>> = _activeAlerts.asStateFlow()

    /**
     * Initialize error analytics
     */
    fun initialize(
        errorRateThreshold: Double = 10.0,
        criticalRateThreshold: Double = 50.0,
        windowDuration: Duration = 1.minutes,
        alertCallback: ((ErrorAlert) -> Unit)? = null
    ) {
        this.errorRateThreshold = errorRateThreshold
        this.criticalRateThreshold = criticalRateThreshold
        this.windowDuration = windowDuration
        this.alertCallback = alertCallback
        this.isInitialized = true
    }

    /**
     * Track an error event
     */
    fun trackError(
        code: String,
        source: String,
        category: ErrorCategory,
        message: String? = null,
        correlationId: String? = null,
        traceId: String? = null
    ) {
        val event = ErrorEvent(
            timestamp = Clock.System.now(),
            code = code,
            source = source,
            category = category,
            message = message,
            correlationId = correlationId,
            traceId = traceId
        )

        synchronized(errorEventsLock) {
            errorEvents.add(event)
            // Clean up old events outside the window
            cleanupOldEvents()
        }

        updateStatistics()
        checkThresholds()
    }

    /**
     * Track error from ArmorClawErrorCode
     */
    fun trackError(
        errorCode: ArmorClawErrorCode,
        source: String,
        message: String? = null,
        correlationId: String? = null,
        traceId: String? = null
    ) {
        trackError(
            code = errorCode.code,
            source = source,
            category = errorCode.category,
            message = message ?: errorCode.userMessage,
            correlationId = correlationId,
            traceId = traceId
        )
    }

    /**
     * Get current error statistics
     */
    fun getStatistics(): ErrorStatistics {
        return _statistics.value
    }

    /**
     * Get error rate for a specific time window
     */
    fun getErrorRate(windowMinutes: Int = 1): Double {
        val now = Clock.System.now()
        val windowStart = now - windowMinutes.minutes

        synchronized(errorEventsLock) {
            val count = errorEvents.count { it.timestamp >= windowStart }
            return count.toDouble() / windowMinutes
        }
    }

    /**
     * Get errors by category
     */
    fun getErrorsByCategory(): Map<ErrorCategory, Int> {
        synchronized(errorEventsLock) {
            return errorEvents.groupingBy { it.category }.eachCount()
        }
    }

    /**
     * Get errors by source (top N)
     */
    fun getErrorsBySource(limit: Int = 10): Map<String, Int> {
        synchronized(errorEventsLock) {
            return errorEvents
                .groupingBy { it.source }
                .eachCount()
                .entries
                .sortedByDescending { it.value }
                .take(limit)
                .associate { it.key to it.value }
        }
    }

    /**
     * Get errors by code (Top N)
     */
    fun getErrorsByCode(limit: Int = 10): Map<String, Int> {
        synchronized(errorEventsLock) {
            return errorEvents
                .groupingBy { it.code }
                .eachCount()
                .entries
                .sortedByDescending { it.value }
                .take(limit)
                .associate { it.key to it.value }
        }
    }

    /**
     * Clear all error events
     */
    fun clearEvents() {
        synchronized(errorEventsLock) {
            errorEvents.clear()
        }
        updateStatistics()
    }

    /**
     * Dismiss an active alert
     */
    fun dismissAlert(alertId: String) {
        _activeAlerts.value = _activeAlerts.value.filter { it.id != alertId }
    }

    /**
     * Clear all alerts
     */
    fun clearAlerts() {
        _activeAlerts.value = emptyList()
    }

    private fun cleanupOldEvents() {
        val cutoff = Clock.System.now() - windowDuration * 5  // Keep 5 windows of history
        errorEvents.removeAll { it.timestamp < cutoff }
    }

    private fun updateStatistics() {
        val now = Clock.System.now()
        val windowStart = now - windowDuration

        synchronized(errorEventsLock) {
            val recentEvents = errorEvents.filter { it.timestamp >= windowStart }
            val rate = recentEvents.size.toDouble() / windowDuration.inWholeSeconds * 60

            _statistics.value = ErrorStatistics(
                totalErrors = errorEvents.size,
                errorsLastWindow = recentEvents.size,
                errorRatePerMinute = rate,
                errorsByCategory = errorEvents.groupingBy { it.category }.eachCount(),
                errorsBySource = getErrorsBySource(),
                errorsByCode = getErrorsByCode(),
                windowDurationSeconds = windowDuration.inWholeSeconds,
                lastUpdated = now
            )
        }
    }

    private fun checkThresholds() {
        val stats = _statistics.value
        val now = Clock.System.now()

        // Check critical threshold
        if (stats.errorRatePerMinute >= criticalRateThreshold) {
            val alert = ErrorAlert(
                id = "critical-${now.epochSeconds}",
                severity = AlertSeverity.CRITICAL,
                message = "Critical error rate: ${String.format("%.1f", stats.errorRatePerMinute)} errors/minute",
                currentRate = stats.errorRatePerMinute,
                threshold = criticalRateThreshold,
                timestamp = now,
                affectedSources = stats.errorsBySource.keys.take(5),
                suggestedAction = "Investigate immediately. Top sources: ${stats.errorsBySource.entries.take(3).joinToString { "${it.key} (${it.value})" }}"
            )
            triggerAlert(alert)
        }
        // Check warning threshold
        else if (stats.errorRatePerMinute >= errorRateThreshold) {
            val alert = ErrorAlert(
                id = "warning-${now.epochSeconds}",
                severity = AlertSeverity.WARNING,
                message = "Elevated error rate: ${String.format("%.1f", stats.errorRatePerMinute)} errors/minute",
                currentRate = stats.errorRatePerMinute,
                threshold = errorRateThreshold,
                timestamp = now,
                affectedSources = stats.errorsBySource.keys.take(5),
                suggestedAction = "Monitor closely. Top categories: ${stats.errorsByCategory.entries.take(3).joinToString { "${it.key.name} (${it.value})" }}"
            )
            triggerAlert(alert)
        }
    }

    private fun triggerAlert(alert: ErrorAlert) {
        // Add to active alerts
        _activeAlerts.value = (_activeAlerts.value + alert)
            .distinctBy { it.id }
            .sortedByDescending { it.timestamp }
            .take(10)  // Keep max 10 active alerts

        // Notify callback
        alertCallback?.invoke(alert)

        // Also log to AppLogger
        AppLogger.warning(
            LogTag.Network.BridgeRpc,
            "Error Analytics Alert: ${alert.message}",
            mapOf(
                "alert_id" to alert.id,
                "severity" to alert.severity.name,
                "current_rate" to alert.currentRate,
                "threshold" to alert.threshold,
                "suggested_action" to alert.suggestedAction
            )
        )
    }
}

// ========================================================================
// Data Classes
// ========================================================================

/**
 * Represents a single error event
 */
data class ErrorEvent(
    val timestamp: Instant,
    val code: String,
    val source: String,
    val category: ErrorCategory,
    val message: String? = null,
    val correlationId: String? = null,
    val traceId: String? = null
)

/**
 * Statistics about error rates and distribution
 */
data class ErrorStatistics(
    val totalErrors: Int = 0,
    val errorsLastWindow: Int = 0,
    val errorRatePerMinute: Double = 0.0,
    val errorsByCategory: Map<ErrorCategory, Int> = emptyMap(),
    val errorsBySource: Map<String, Int> = emptyMap(),
    val errorsByCode: Map<String, Int> = emptyMap(),
    val windowDurationSeconds: Long = 60,
    val lastUpdated: Instant = Clock.System.now()
) {
    /**
     * Check if error rate is healthy (below warning threshold)
     */
    val isHealthy: Boolean
        get() = errorRatePerMinute < 10.0

    /**
     * Get formatted error rate
     */
    val formattedRate: String
        get() = String.format("%.2f", errorRatePerMinute)
}

/**
 * Alert triggered when error thresholds are exceeded
 */
data class ErrorAlert(
    val id: String,
    val severity: AlertSeverity,
    val message: String,
    val currentRate: Double,
    val threshold: Double,
    val timestamp: Instant,
    val affectedSources: List<String>,
    val suggestedAction: String
)

/**
 * Alert severity levels
 */
enum class AlertSeverity {
    WARNING,
    CRITICAL
}
