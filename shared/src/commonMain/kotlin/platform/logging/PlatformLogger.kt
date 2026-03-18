package com.armorclaw.shared.platform.logging

/**
 * Platform-specific logging interface
 * Implemented separately for each platform
 */
expect object PlatformLogger {
    fun debug(tag: String, message: String, throwable: Throwable?)
    fun info(tag: String, message: String)
    fun warning(tag: String, message: String, throwable: Throwable?)
    fun error(tag: String, message: String, throwable: Throwable?)
}
