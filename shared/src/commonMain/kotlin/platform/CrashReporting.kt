package com.armorclaw.shared.platform

import kotlinx.coroutines.flow.StateFlow

/**
 * Platform-agnostic crash reporting interface
 */
interface CrashReporting {

    /**
     * Whether crash reporting is currently enabled
     */
    val isEnabled: StateFlow<Boolean>

    /**
     * Whether crash reporting has been initialized
     */
    val isInitialized: StateFlow<Boolean>

    /**
     * Initialize the crash reporting system
     */
    fun initialize()

    /**
     * Capture an exception for reporting
     */
    fun captureException(exception: Throwable, tags: Map<String, String>? = null)

    /**
     * Capture a message for reporting
     */
    fun captureMessage(message: String, level: Severity, tags: Map<String, String>? = null)

    /**
     * Set the user ID for crash reports
     */
    fun setUserId(userId: String)

    /**
     * Set detailed user information
     */
    fun setUserInfo(
        userId: String,
        username: String,
        email: String,
        additional: Map<String, Any> = emptyMap()
    )

    /**
     * Clear user information from crash reports
     */
    fun clearUserInfo()

    /**
     * Add a breadcrumb for tracking user flow
     */
    fun addBreadcrumb(
        message: String,
        category: String,
        type: String,
        level: Severity,
        data: Map<String, Any> = emptyMap()
    )

    /**
     * Set a single tag
     */
    fun setTag(key: String, value: String)

    /**
     * Set multiple tags
     */
    fun setTags(tags: Map<String, String>)

    /**
     * Set a context value
     */
    fun setContext(key: String, value: Any)

    /**
     * Set multiple context values
     */
    fun setContexts(contexts: Map<String, Any>)

    /**
     * Enable crash reporting
     */
    fun enable()

    /**
     * Disable crash reporting
     */
    fun disable()

    /**
     * Capture a crash report and return the report ID
     */
    fun captureCrashReport(exception: Throwable): String

    /**
     * Start performance monitoring for an operation
     */
    fun startPerformanceMonitoring(operation: String)

    /**
     * Stop performance monitoring for an operation
     */
    fun stopPerformanceMonitoring(operation: String)

    /**
     * Severity levels for crash reporting
     */
    enum class Severity {
        DEBUG,
        INFO,
        WARNING,
        ERROR,
        FATAL
    }
}
