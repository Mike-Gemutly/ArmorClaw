package com.armorclaw.shared.platform

import kotlinx.coroutines.flow.StateFlow
import kotlinx.datetime.Instant

/**
 * Platform-agnostic analytics interface
 */
interface Analytics {

    /**
     * Whether analytics is currently enabled
     */
    val isEnabled: StateFlow<Boolean>

    /**
     * Whether analytics has been initialized
     */
    val isInitialized: StateFlow<Boolean>

    /**
     * Initialize the analytics system
     */
    fun initialize()

    /**
     * Track an event
     */
    fun trackEvent(
        name: String,
        properties: Map<String, Any> = emptyMap(),
        timestamp: Instant? = null
    )

    /**
     * Track a screen view
     */
    fun trackScreen(
        screenName: String,
        screenClass: String,
        properties: Map<String, Any> = emptyMap()
    )

    /**
     * Set the current user
     */
    fun setUser(
        userId: String,
        properties: Map<String, Any> = emptyMap()
    )

    /**
     * Set a user property
     */
    fun setUserProperty(name: String, value: Any)

    /**
     * Clear the current user
     */
    fun clearUser()

    /**
     * Set a group
     */
    fun setGroup(groupType: String, groupKey: String)

    /**
     * Track revenue
     */
    fun trackRevenue(
        amount: Double,
        currency: String,
        productId: String,
        properties: Map<String, Any> = emptyMap()
    )

    /**
     * Track an error
     */
    fun trackError(
        errorName: String,
        errorMessage: String,
        stackTrace: String? = null,
        properties: Map<String, Any> = emptyMap()
    )

    /**
     * Start a session
     */
    fun startSession(sessionId: String? = null)

    /**
     * End a session
     */
    fun endSession(sessionId: String? = null)

    /**
     * Enable analytics
     */
    fun enable()

    /**
     * Disable analytics
     */
    fun disable()

    /**
     * Flush pending events
     */
    fun flush()
}
