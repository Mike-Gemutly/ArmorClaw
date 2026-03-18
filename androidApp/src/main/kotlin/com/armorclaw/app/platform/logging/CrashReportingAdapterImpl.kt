package com.armorclaw.app.platform.logging

import com.armorclaw.shared.platform.logging.AnalyticsAdapter
import com.armorclaw.shared.platform.logging.CrashReportingAdapter
import com.armorclaw.shared.platform.logging.CrashSeverity
import com.armorclaw.app.platform.CrashReporter

/**
 * Android implementation of CrashReportingAdapter
 * Connects AppLogger to Sentry via CrashReporter
 */
class CrashReportingAdapterImpl(
    private val crashReporter: CrashReporter
) : CrashReportingAdapter {

    override fun captureException(throwable: Throwable, tags: Map<String, String>) {
        crashReporter.captureException(throwable, tags)
    }

    override fun captureMessage(message: String, level: CrashSeverity, tags: Map<String, String>) {
        val severity = when (level) {
            CrashSeverity.DEBUG -> com.armorclaw.shared.platform.CrashReporting.Severity.DEBUG
            CrashSeverity.INFO -> com.armorclaw.shared.platform.CrashReporting.Severity.INFO
            CrashSeverity.WARNING -> com.armorclaw.shared.platform.CrashReporting.Severity.WARNING
            CrashSeverity.ERROR -> com.armorclaw.shared.platform.CrashReporting.Severity.ERROR
            CrashSeverity.FATAL -> com.armorclaw.shared.platform.CrashReporting.Severity.FATAL
        }
        crashReporter.captureMessage(message, severity, tags)
    }

    override fun addBreadcrumb(message: String, category: String, data: Map<String, Any>) {
        val severity = com.armorclaw.shared.platform.CrashReporting.Severity.INFO
        crashReporter.addBreadcrumb(
            message = message,
            category = category,
            type = "manual",
            level = severity,
            data = data
        )
    }
}

/**
 * Android implementation of AnalyticsAdapter
 * Connects AppLogger to Analytics
 */
class AnalyticsAdapterImpl(
    private val analytics: com.armorclaw.app.platform.Analytics
) : AnalyticsAdapter {

    override fun trackError(errorName: String, errorMessage: String, properties: Map<String, Any>) {
        analytics.trackError(
            errorName = errorName,
            errorMessage = errorMessage,
            stackTrace = null,
            properties = properties
        )
    }
}
