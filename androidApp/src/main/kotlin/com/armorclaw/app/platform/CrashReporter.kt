package com.armorclaw.app.platform
import kotlinx.datetime.Clock

import android.content.Context
import android.os.Build
import com.armorclaw.shared.platform.CrashReporting
import io.sentry.Sentry
import io.sentry.SentryLevel
import io.sentry.android.core.SentryAndroid
import io.sentry.protocol.User as SentryUser
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Crash reporting implementation using Sentry
 *
 * This class handles crash reporting, error tracking, and breadcrumbs
 * for production monitoring and debugging.
 */
class CrashReporter(
    private val context: Context,
    private val dsn: String = "",
    private val environment: String = "production",
    private val enabled: Boolean = true
) : CrashReporting {

    private val _isEnabled = MutableStateFlow(enabled)
    override val isEnabled: StateFlow<Boolean> = _isEnabled.asStateFlow()

    private val _isInitialized = MutableStateFlow(false)
    override val isInitialized: StateFlow<Boolean> = _isInitialized.asStateFlow()

    init {
        initialize()
    }

    override fun initialize() {
        if (dsn.isBlank() || !enabled) {
            _isInitialized.value = false
            return
        }

        SentryAndroid.init(context) { options ->
            options.dsn = dsn
            options.environment = environment
            options.tracesSampleRate = 0.1
            options.isAttachStacktrace = true
        }

        _isInitialized.value = true
        _isEnabled.value = enabled
    }

    override fun captureException(exception: Throwable, tags: Map<String, String>?) {
        if (!_isEnabled.value || !_isInitialized.value) return

        Sentry.captureException(exception) { scope ->
            tags?.forEach { (key, value) ->
                scope.setTag(key, value)
            }
        }
    }

    override fun captureMessage(message: String, level: CrashReporting.Severity, tags: Map<String, String>?) {
        if (!_isEnabled.value || !_isInitialized.value) return

        Sentry.captureMessage(message, level.toSentryLevel()) { scope ->
            tags?.forEach { (key, value) ->
                scope.setTag(key, value)
            }
        }
    }

    override fun setUserId(userId: String) {
        if (!_isEnabled.value || !_isInitialized.value) return

        val user = SentryUser().apply {
            this.id = userId
        }
        Sentry.setUser(user)
    }

    override fun setUserInfo(
        userId: String,
        username: String,
        email: String,
        additional: Map<String, Any>
    ) {
        if (!_isEnabled.value || !_isInitialized.value) return

        val user = SentryUser().apply {
            this.id = userId
            this.username = username
            this.email = email
        }
        Sentry.setUser(user)
    }

    override fun clearUserInfo() {
        if (!_isEnabled.value || !_isInitialized.value) return

        Sentry.setUser(null)
    }

    override fun addBreadcrumb(
        message: String,
        category: String,
        type: String,
        level: CrashReporting.Severity,
        data: Map<String, Any>
    ) {
        if (!_isEnabled.value || !_isInitialized.value) return

        val breadcrumb = io.sentry.Breadcrumb().apply {
            this.message = message
            this.category = category
            this.type = type
            this.level = level.toSentryLevel()
        }
        Sentry.addBreadcrumb(breadcrumb)
    }

    override fun setTag(key: String, value: String) {
        if (!_isEnabled.value || !_isInitialized.value) return

        Sentry.setTag(key, value)
    }

    override fun setTags(tags: Map<String, String>) {
        if (!_isEnabled.value || !_isInitialized.value) return

        tags.forEach { (key, value) ->
            Sentry.setTag(key, value)
        }
    }

    override fun setContext(key: String, value: Any) {
        if (!_isEnabled.value || !_isInitialized.value) return

        Sentry.setExtra(key, value.toString())
    }

    override fun setContexts(contexts: Map<String, Any>) {
        if (!_isEnabled.value || !_isInitialized.value) return

        contexts.forEach { (key, value) ->
            Sentry.setExtra(key, value.toString())
        }
    }

    override fun enable() {
        if (!_isInitialized.value) {
            initialize()
        }

        _isEnabled.value = true
    }

    override fun disable() {
        _isEnabled.value = false
    }

    override fun captureCrashReport(exception: Throwable): String {
        if (!_isEnabled.value || !_isInitialized.value) {
            return generateLocalCrashReport(exception)
        }

        val sentryId = Sentry.captureException(exception)

        return sentryId?.toString() ?: generateLocalCrashReport(exception)
    }

    override fun startPerformanceMonitoring(operation: String) {
        if (!_isEnabled.value || !_isInitialized.value) return

        // Sentry performance monitoring is handled automatically
    }

    override fun stopPerformanceMonitoring(operation: String) {
        if (!_isEnabled.value || !_isInitialized.value) return

        // Sentry performance monitoring is handled automatically
    }

    private fun generateLocalCrashReport(exception: Throwable): String {
        val timestamp = kotlinx.datetime.Clock.System.now()
        val stackTrace = exception.stackTraceToString()

        return """
            ArmorClaw Crash Report
            ==========================
            Timestamp: ${timestamp}
            Android Version: ${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})
            Device: ${Build.MANUFACTURER} ${Build.MODEL}

            Exception: ${exception.javaClass.simpleName}
            Message: ${exception.message}

            Stack Trace:
            $stackTrace
        """.trimIndent()
    }

    private fun CrashReporting.Severity.toSentryLevel(): SentryLevel {
        return when (this) {
            CrashReporting.Severity.DEBUG -> SentryLevel.DEBUG
            CrashReporting.Severity.INFO -> SentryLevel.INFO
            CrashReporting.Severity.WARNING -> SentryLevel.WARNING
            CrashReporting.Severity.ERROR -> SentryLevel.ERROR
            CrashReporting.Severity.FATAL -> SentryLevel.FATAL
        }
    }
}
