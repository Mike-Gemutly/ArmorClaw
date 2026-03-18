package com.armorclaw.shared.platform.logging

import com.armorclaw.shared.domain.model.ErrorCategory
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Clock
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import kotlin.test.*

/**
 * Tests for ErrorAnalytics - error rate tracking and analytics
 *
 * Uses Robolectric to provide Android runtime for logging.
 */
@RunWith(RobolectricTestRunner::class)
class ErrorAnalyticsTest {

    @Before
    fun setup() {
        // Reset ErrorAnalytics before each test
        ErrorAnalytics.clearEvents()
        ErrorAnalytics.clearAlerts()
        ErrorAnalytics.initialize(
            errorRateThreshold = 10.0,
            criticalRateThreshold = 50.0,
            alertCallback = null
        )
    }

    // ========================================================================
    // Basic Error Tracking Tests
    // ========================================================================

    @Test
    fun `should track single error event`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError(
            code = "E001",
            source = "Test:method",
            category = ErrorCategory.NETWORK,
            message = "Test error"
        )

        // Assert
        val stats = ErrorAnalytics.getStatistics()
        assertEquals(1, stats.totalErrors)
    }

    @Test
    fun `should track multiple error events`() = runTest {
        // Arrange & Act
        repeat(5) {
            ErrorAnalytics.trackError(
                code = "E00$it",
                source = "Test:method$it",
                category = ErrorCategory.NETWORK
            )
        }

        // Assert
        val stats = ErrorAnalytics.getStatistics()
        assertEquals(5, stats.totalErrors)
    }

    @Test
    fun `should include correlation ID and trace ID in events`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError(
            code = "E001",
            source = "Test:method",
            category = ErrorCategory.NETWORK,
            correlationId = "corr-123",
            traceId = "trace-456"
        )

        // Assert - verify it doesn't throw and tracks the event
        val stats = ErrorAnalytics.getStatistics()
        assertEquals(1, stats.totalErrors)
    }

    // ========================================================================
    // Error Rate Calculation Tests
    // ========================================================================

    @Test
    fun `should calculate error rate correctly`() = runTest {
        // Arrange & Act - track 5 errors
        repeat(5) {
            ErrorAnalytics.trackError(
                code = "E001",
                source = "Test:method",
                category = ErrorCategory.NETWORK
            )
        }

        // Assert
        val rate = ErrorAnalytics.getErrorRate(windowMinutes = 1)
        // Rate should be approximately 5 errors per minute
        assertTrue(rate >= 4.0 && rate <= 6.0, "Expected rate ~5.0, got $rate")
    }

    @Test
    fun `should return zero rate when no errors tracked`() = runTest {
        // Arrange - no errors tracked

        // Act
        val rate = ErrorAnalytics.getErrorRate(windowMinutes = 1)

        // Assert
        assertEquals(0.0, rate)
    }

    // ========================================================================
    // Statistics Aggregation Tests
    // ========================================================================

    @Test
    fun `should aggregate errors by category`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError("E001", "Test:1", ErrorCategory.NETWORK)
        ErrorAnalytics.trackError("E002", "Test:2", ErrorCategory.NETWORK)
        ErrorAnalytics.trackError("E003", "Test:3", ErrorCategory.SYNC)
        ErrorAnalytics.trackError("E004", "Test:4", ErrorCategory.NETWORK)

        // Assert
        val byCategory = ErrorAnalytics.getErrorsByCategory()
        assertEquals(3, byCategory[ErrorCategory.NETWORK])
        assertEquals(1, byCategory[ErrorCategory.SYNC])
    }

    @Test
    fun `should aggregate errors by source`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError("E001", "Repository:save", ErrorCategory.SYNC)
        ErrorAnalytics.trackError("E002", "Repository:save", ErrorCategory.SYNC)
        ErrorAnalytics.trackError("E003", "Repository:load", ErrorCategory.SYNC)
        ErrorAnalytics.trackError("E004", "Network:fetch", ErrorCategory.NETWORK)

        // Assert
        val bySource = ErrorAnalytics.getErrorsBySource(limit = 10)
        assertEquals(2, bySource["Repository:save"])
        assertEquals(1, bySource["Repository:load"])
        assertEquals(1, bySource["Network:fetch"])
    }

    @Test
    fun `should aggregate errors by code`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError("E001", "Test:1", ErrorCategory.NETWORK)
        ErrorAnalytics.trackError("E001", "Test:2", ErrorCategory.NETWORK)
        ErrorAnalytics.trackError("E002", "Test:3", ErrorCategory.NETWORK)

        // Assert
        val byCode = ErrorAnalytics.getErrorsByCode(limit = 10)
        assertEquals(2, byCode["E001"])
        assertEquals(1, byCode["E002"])
    }

    @Test
    fun `should limit errors by source results`() = runTest {
        // Arrange & Act - track many different sources
        repeat(20) { index ->
            ErrorAnalytics.trackError(
                code = "E$index",
                source = "Source$index:method",
                category = ErrorCategory.NETWORK
            )
        }

        // Assert
        val bySource = ErrorAnalytics.getErrorsBySource(limit = 5)
        assertTrue(bySource.size <= 5)
    }

    // ========================================================================
    // Threshold Alert Tests
    // ========================================================================

    @Test
    fun `should trigger warning alert when threshold exceeded`() = runTest {
        // Arrange
        var alertTriggered: ErrorAlert? = null
        ErrorAnalytics.initialize(
            errorRateThreshold = 5.0,  // Low threshold for testing
            criticalRateThreshold = 100.0,
            alertCallback = { alert -> alertTriggered = alert }
        )

        // Act - track enough errors to exceed threshold
        repeat(10) {
            ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        }

        // Assert
        assertNotNull(alertTriggered)
        assertEquals(AlertSeverity.WARNING, alertTriggered!!.severity)
    }

    @Test
    fun `should trigger critical alert when critical threshold exceeded`() = runTest {
        // Arrange
        var alertTriggered: ErrorAlert? = null
        ErrorAnalytics.initialize(
            errorRateThreshold = 5.0,
            criticalRateThreshold = 10.0,
            alertCallback = { alert -> alertTriggered = alert }
        )

        // Act - track enough errors to exceed critical threshold
        repeat(20) {
            ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        }

        // Assert
        assertNotNull(alertTriggered)
        assertEquals(AlertSeverity.CRITICAL, alertTriggered!!.severity)
    }

    @Test
    fun `should dismiss alert by ID`() = runTest {
        // Arrange - trigger an alert
        ErrorAnalytics.initialize(
            errorRateThreshold = 2.0,
            criticalRateThreshold = 100.0
        )
        repeat(5) {
            ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        }

        // Get the alert ID from active alerts
        val alertsBefore = ErrorAnalytics.activeAlerts.first()
        assertTrue(alertsBefore.isNotEmpty())
        val alertId = alertsBefore.first().id

        // Act
        ErrorAnalytics.dismissAlert(alertId)

        // Assert
        val alertsAfter = ErrorAnalytics.activeAlerts.first()
        assertTrue(alertsAfter.none { it.id == alertId })
    }

    // ========================================================================
    // Statistics StateFlow Tests
    // ========================================================================

    @Test
    fun `should expose statistics via StateFlow`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)

        // Assert
        val stats = ErrorAnalytics.statistics.first()
        assertEquals(1, stats.totalErrors)
    }

    @Test
    fun `should expose active alerts via StateFlow`() = runTest {
        // Arrange
        ErrorAnalytics.initialize(
            errorRateThreshold = 1.0,
            criticalRateThreshold = 100.0
        )

        // Act
        ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        ErrorAnalytics.trackError("E002", "Test:method", ErrorCategory.NETWORK)

        // Assert
        val alerts = ErrorAnalytics.activeAlerts.first()
        assertTrue(alerts.isNotEmpty())
    }

    // ========================================================================
    // Clear Operations Tests
    // ========================================================================

    @Test
    fun `should clear all error events`() = runTest {
        // Arrange
        repeat(5) {
            ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        }

        // Act
        ErrorAnalytics.clearEvents()

        // Assert
        val stats = ErrorAnalytics.getStatistics()
        assertEquals(0, stats.totalErrors)
    }

    @Test
    fun `should clear all alerts`() = runTest {
        // Arrange - trigger alerts
        ErrorAnalytics.initialize(
            errorRateThreshold = 1.0,
            criticalRateThreshold = 100.0
        )
        repeat(5) {
            ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        }

        // Act
        ErrorAnalytics.clearAlerts()

        // Assert
        val alerts = ErrorAnalytics.activeAlerts.first()
        assertTrue(alerts.isEmpty())
    }

    // ========================================================================
    // ErrorStatistics Helper Properties Tests
    // ========================================================================

    @Test
    fun `should identify healthy state when below threshold`() = runTest {
        // Arrange - track few errors
        ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)

        // Act
        val stats = ErrorAnalytics.getStatistics()

        // Assert
        assertTrue(stats.isHealthy)
    }

    @Test
    fun `should format error rate correctly`() = runTest {
        // Arrange
        repeat(3) {
            ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        }

        // Act
        val stats = ErrorAnalytics.getStatistics()
        val formatted = stats.formattedRate

        // Assert
        assertTrue(formatted.matches(Regex("\\d+\\.\\d{2}")))
    }

    // ========================================================================
    // Alert Data Tests
    // ========================================================================

    @Test
    fun `should include affected sources in alert`() = runTest {
        // Arrange
        var capturedAlert: ErrorAlert? = null
        ErrorAnalytics.initialize(
            errorRateThreshold = 2.0,
            criticalRateThreshold = 100.0,
            alertCallback = { alert -> capturedAlert = alert }
        )

        // Act
        repeat(5) {
            ErrorAnalytics.trackError("E001", "Repository:save", ErrorCategory.SYNC)
        }

        // Assert
        assertNotNull(capturedAlert)
        assertTrue(capturedAlert!!.affectedSources.isNotEmpty())
        assertTrue("Repository:save" in capturedAlert!!.affectedSources)
    }

    @Test
    fun `should include suggested action in alert`() = runTest {
        // Arrange
        var capturedAlert: ErrorAlert? = null
        ErrorAnalytics.initialize(
            errorRateThreshold = 2.0,
            criticalRateThreshold = 100.0,
            alertCallback = { alert -> capturedAlert = alert }
        )

        // Act
        repeat(5) {
            ErrorAnalytics.trackError("E001", "Test:method", ErrorCategory.NETWORK)
        }

        // Assert
        assertNotNull(capturedAlert)
        assertNotNull(capturedAlert!!.suggestedAction)
        assertTrue(capturedAlert!!.suggestedAction.isNotEmpty())
    }

    // ========================================================================
    // Edge Cases Tests
    // ========================================================================

    @Test
    fun `should handle unknown error category`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError(
            code = "E001",
            source = "Test:method",
            category = ErrorCategory.UNKNOWN
        )

        // Assert
        val stats = ErrorAnalytics.getStatistics()
        assertEquals(1, stats.totalErrors)
        assertEquals(1, stats.errorsByCategory[ErrorCategory.UNKNOWN])
    }

    @Test
    fun `should handle empty message`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError(
            code = "E001",
            source = "Test:method",
            category = ErrorCategory.NETWORK,
            message = null
        )

        // Assert
        val stats = ErrorAnalytics.getStatistics()
        assertEquals(1, stats.totalErrors)
    }

    @Test
    fun `should handle special characters in error codes and sources`() = runTest {
        // Arrange & Act
        ErrorAnalytics.trackError(
            code = "E-001/Special:Code!",
            source = "com.example.package:method-name",
            category = ErrorCategory.NETWORK
        )

        // Assert
        val byCode = ErrorAnalytics.getErrorsByCode()
        assertEquals(1, byCode["E-001/Special:Code!"])
    }
}
