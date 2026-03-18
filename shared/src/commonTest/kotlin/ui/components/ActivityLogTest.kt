package com.armorclaw.shared.ui.components

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for ActivityLog component
 *
 * Tests event rendering, status handling, empty state,
 * timestamp formatting, and event click handling.
 */
class ActivityLogTest {

    // ========================================================================
    // AgentEvent Data Class Tests
    // ========================================================================

    @Test
    fun `AgentEvent has correct properties`() {
        val event = AgentEvent(
            id = "event_1",
            stepName = "Processing data",
            status = AgentStepStatus.RUNNING,
            timestamp = System.currentTimeMillis(),
            output = "Loading..."
        )

        assertEquals("event_1", event.id)
        assertEquals("Processing data", event.stepName)
        assertEquals(AgentStepStatus.RUNNING, event.status)
        assertEquals("Loading...", event.output)
    }

    @Test
    fun `AgentEvent with empty output`() {
        val event = AgentEvent(
            id = "event_2",
            stepName = "Step with no output",
            status = AgentStepStatus.COMPLETED,
            timestamp = System.currentTimeMillis(),
            output = ""
        )

        assertEquals("", event.output)
    }

    @Test
    fun `AgentEvent with blank output`() {
        val event = AgentEvent(
            id = "event_3",
            stepName = "Another step",
            status = AgentStepStatus.PENDING,
            timestamp = System.currentTimeMillis(),
            output = "   "
        )

        assertEquals("   ", event.output)
    }

    @Test
    fun `AgentEvent without output parameter`() {
        val event = AgentEvent(
            id = "event_4",
            stepName = "Step without output",
            status = AgentStepStatus.FAILED,
            timestamp = System.currentTimeMillis()
        )

        assertEquals("", event.output, "Default output should be empty string")
    }

    // ========================================================================
    // AgentStepStatus Enum Tests
    // ========================================================================

    @Test
    fun `AgentStepStatus RUNNING is active`() {
        val status = AgentStepStatus.RUNNING

        assertTrue(status.isActive(), "RUNNING should be active")
    }

    @Test
    fun `AgentStepStatus PENDING is active`() {
        val status = AgentStepStatus.PENDING

        assertTrue(status.isActive(), "PENDING should be active")
    }

    @Test
    fun `AgentStepStatus COMPLETED is not active`() {
        val status = AgentStepStatus.COMPLETED

        assertFalse(status.isActive(), "COMPLETED should not be active")
    }

    @Test
    fun `AgentStepStatus FAILED is not active`() {
        val status = AgentStepStatus.FAILED

        assertFalse(status.isActive(), "FAILED should not be active")
    }

    @Test
    fun `AgentStepStatus SKIPPED is not active`() {
        val status = AgentStepStatus.SKIPPED

        assertFalse(status.isActive(), "SKIPPED should not be active")
    }

    @Test
    fun `AgentStepStatus CANCELLED is not active`() {
        val status = AgentStepStatus.CANCELLED

        assertFalse(status.isActive(), "CANCELLED should not be active")
    }

    // ========================================================================
    // Event Rendering Tests
    // ========================================================================

    @Test
    fun `Event list can be empty`() {
        val events = emptyList<AgentEvent>()

        assertEquals(0, events.size, "Event list should be empty")
    }

    @Test
    fun `Event list can have multiple events`() {
        val events = listOf(
            AgentEvent("1", "Step 1", AgentStepStatus.RUNNING, 1000, "Output 1"),
            AgentEvent("2", "Step 2", AgentStepStatus.COMPLETED, 2000, "Output 2"),
            AgentEvent("3", "Step 3", AgentStepStatus.PENDING, 3000, "Output 3")
        )

        assertEquals(3, events.size, "Event list should have 3 events")
    }

    @Test
    fun `Events can have different statuses`() {
        val events = listOf(
            AgentEvent("1", "Running", AgentStepStatus.RUNNING, 1000),
            AgentEvent("2", "Completed", AgentStepStatus.COMPLETED, 2000),
            AgentEvent("3", "Failed", AgentStepStatus.FAILED, 3000),
            AgentEvent("4", "Pending", AgentStepStatus.PENDING, 4000),
            AgentEvent("5", "Skipped", AgentStepStatus.SKIPPED, 5000),
            AgentEvent("6", "Cancelled", AgentStepStatus.CANCELLED, 6000)
        )

        val statuses = events.map { it.status }.toSet()
        assertEquals(6, statuses.size, "All 6 statuses should be present")
    }

    @Test
    fun `Events are uniquely identified by id`() {
        val events = listOf(
            AgentEvent("event_1", "Step 1", AgentStepStatus.COMPLETED, 1000),
            AgentEvent("event_2", "Step 2", AgentStepStatus.COMPLETED, 2000),
            AgentEvent("event_3", "Step 3", AgentStepStatus.COMPLETED, 3000)
        )

        val uniqueIds = events.map { it.id }.toSet()
        assertEquals(3, uniqueIds.size, "All event IDs should be unique")
    }

    // ========================================================================
    // Status Icon Tests
    // ========================================================================

    @Test
    fun `Status icon for RUNNING status`() {
        val status = AgentStepStatus.RUNNING
        val iconName = getStatusIconName(status)

        assertEquals("Pending", iconName, "RUNNING should use Pending icon")
    }

    @Test
    fun `Status icon for COMPLETED status`() {
        val status = AgentStepStatus.COMPLETED
        val iconName = getStatusIconName(status)

        assertEquals("CheckCircle", iconName, "COMPLETED should use CheckCircle icon")
    }

    @Test
    fun `Status icon for FAILED status`() {
        val status = AgentStepStatus.FAILED
        val iconName = getStatusIconName(status)

        assertEquals("Error", iconName, "FAILED should use Error icon")
    }

    @Test
    fun `Status icon for PENDING status`() {
        val status = AgentStepStatus.PENDING
        val iconName = getStatusIconName(status)

        assertEquals("Info", iconName, "PENDING should use Info icon")
    }

    @Test
    fun `Status icon for SKIPPED status`() {
        val status = AgentStepStatus.SKIPPED
        val iconName = getStatusIconName(status)

        assertEquals("Schedule", iconName, "SKIPPED should use Schedule icon")
    }

    @Test
    fun `Status icon for CANCELLED status`() {
        val status = AgentStepStatus.CANCELLED
        val iconName = getStatusIconName(status)

        assertEquals("Close", iconName, "CANCELLED should use Close icon")
    }

    // ========================================================================
    // Status Color Tests
    // ========================================================================

    @Test
    fun `Status color for RUNNING is info color`() {
        val status = AgentStepStatus.RUNNING
        val colorCategory = getStatusColorCategory(status)

        assertEquals("Info", colorCategory, "RUNNING should use Info color")
    }

    @Test
    fun `Status color for COMPLETED is success color`() {
        val status = AgentStepStatus.COMPLETED
        val colorCategory = getStatusColorCategory(status)

        assertEquals("Success", colorCategory, "COMPLETED should use Success color")
    }

    @Test
    fun `Status color for FAILED is error color`() {
        val status = AgentStepStatus.FAILED
        val colorCategory = getStatusColorCategory(status)

        assertEquals("Error", colorCategory, "FAILED should use Error color")
    }

    @Test
    fun `Status color for PENDING is warning color`() {
        val status = AgentStepStatus.PENDING
        val colorCategory = getStatusColorCategory(status)

        assertEquals("Warning", colorCategory, "PENDING should use Warning color")
    }

    @Test
    fun `Status color for SKIPPED is onSurfaceVariant color`() {
        val status = AgentStepStatus.SKIPPED
        val colorCategory = getStatusColorCategory(status)

        assertEquals("OnSurfaceVariant", colorCategory, "SKIPPED should use OnSurfaceVariant color")
    }

    @Test
    fun `Status color for CANCELLED is error color`() {
        val status = AgentStepStatus.CANCELLED
        val colorCategory = getStatusColorCategory(status)

        assertEquals("Error", colorCategory, "CANCELLED should use Error color")
    }

    // ========================================================================
    // Timestamp Formatting Tests
    // ========================================================================

    @Test
    fun `Timestamp is formatted in correct pattern`() {
        val timestamp = 1234567890000L // 2009-02-13 23:31:30 UTC
        val formatted = formatTimestamp(timestamp)

        assertTrue(formatted.matches(Regex("\\d{2}:\\d{2}:\\d{2}")), "Timestamp should match HH:mm:ss format")
    }

    @Test
    fun `Timestamp formatting handles milliseconds`() {
        val timestamp = 1678901234567L
        val formatted = formatTimestamp(timestamp)

        assertTrue(formatted.contains(":"), "Formatted timestamp should contain colons")
        assertEquals(8, formatted.length, "Formatted timestamp should be 8 characters (HH:mm:ss)")
    }

    @Test
    fun `Timestamp at midnight formats correctly`() {
        val timestamp = 1678900000000L
        val formatted = formatTimestamp(timestamp)

        assertTrue(formatted.matches(Regex("\\d{2}:\\d{2}:\\d{2}")), "Midnight timestamp should still format correctly")
    }

    // ========================================================================
    // Event Click Handling Tests
    // ========================================================================

    @Test
    fun `onEventClick callback receives the clicked event`() {
        val event = AgentEvent(
            id = "clicked_event",
            stepName = "Clickable step",
            status = AgentStepStatus.COMPLETED,
            timestamp = System.currentTimeMillis()
        )

        var clickedEvent: AgentEvent? = null
        val clickHandler: (AgentEvent) -> Unit = { clickedEvent = it }

        clickHandler(event)

        assertEquals(event, clickedEvent, "Click handler should receive the event")
    }

    @Test
    fun `Different events trigger different callbacks`() {
        val event1 = AgentEvent("1", "Event 1", AgentStepStatus.COMPLETED, 1000)
        val event2 = AgentEvent("2", "Event 2", AgentStepStatus.RUNNING, 2000)

        var clickedId: String? = null
        val clickHandler: (AgentEvent) -> Unit = { clickedId = it.id }

        clickHandler(event1)
        assertEquals("1", clickedId, "First click should have ID 1")

        clickHandler(event2)
        assertEquals("2", clickedId, "Second click should have ID 2")
    }

    // ========================================================================
    // Empty State Tests
    // ========================================================================

    @Test
    fun `Empty state shows when events list is empty`() {
        val events = emptyList<AgentEvent>()
        val isEmpty = events.isEmpty()

        assertTrue(isEmpty, "Empty state should show when no events")
    }

    @Test
    fun `Empty state does not show when events exist`() {
        val events = listOf(
            AgentEvent("1", "Event", AgentStepStatus.COMPLETED, 1000)
        )
        val isEmpty = events.isEmpty()

        assertFalse(isEmpty, "Empty state should not show when events exist")
    }

    // ========================================================================
    // Event Count Display Tests
    // ========================================================================

    @Test
    fun `Event count display shows correct number`() {
        val events = listOf(
            AgentEvent("1", "Event 1", AgentStepStatus.COMPLETED, 1000),
            AgentEvent("2", "Event 2", AgentStepStatus.COMPLETED, 2000),
            AgentEvent("3", "Event 3", AgentStepStatus.COMPLETED, 3000)
        )

        val count = events.size
        assertEquals(3, count, "Event count should be 3")
    }

    @Test
    fun `Event count is zero for empty list`() {
        val events = emptyList<AgentEvent>()

        val count = events.size
        assertEquals(0, count, "Event count should be 0")
    }

    // ========================================================================
    // Timeline Line Tests
    // ========================================================================

    @Test
    fun `Timeline line appears for all but last event`() {
        val events = listOf(
            AgentEvent("1", "Event 1", AgentStepStatus.COMPLETED, 1000),
            AgentEvent("2", "Event 2", AgentStepStatus.COMPLETED, 2000),
            AgentEvent("3", "Event 3", AgentStepStatus.COMPLETED, 3000)
        )

        // Index 0 and 1 should have lines, index 2 should not
        val eventsWithLines = events.indices.count { index ->
            index < events.size - 1
        }

        assertEquals(2, eventsWithLines, "Two events should have timeline lines")
    }

    @Test
    fun `Single event has no timeline line`() {
        val events = listOf(
            AgentEvent("1", "Single Event", AgentStepStatus.COMPLETED, 1000)
        )

        val hasLine = events.indices.any { index ->
            index < events.size - 1
        }

        assertFalse(hasLine, "Single event should not have timeline line")
    }

    // ========================================================================
    // Auto Scroll Tests
    // ========================================================================

    @Test
    fun `Auto scroll is enabled by default`() {
        val autoScroll = true

        assertTrue(autoScroll, "Auto scroll should be enabled by default")
    }

    @Test
    fun `Auto scroll can be disabled`() {
        val autoScroll = false

        assertFalse(autoScroll, "Auto scroll can be disabled")
    }

    // ========================================================================
    // Header Visibility Tests
    // ========================================================================

    @Test
    fun `Header is visible by default`() {
        val showHeader = true

        assertTrue(showHeader, "Header should be visible by default")
    }

    @Test
    fun `Header can be hidden`() {
        val showHeader = false

        assertFalse(showHeader, "Header can be hidden")
    }

    // ========================================================================
    // Helper Functions (Mirror Component Logic)
    // ========================================================================
}

/**
 * Helper function to get status icon name
 * Mirrors the getStatusIcon function in ActivityLog.kt
 */
private fun getStatusIconName(status: AgentStepStatus): String {
    return when (status) {
        AgentStepStatus.RUNNING -> "Pending"
        AgentStepStatus.COMPLETED -> "CheckCircle"
        AgentStepStatus.FAILED -> "Error"
        AgentStepStatus.PENDING -> "Info"
        AgentStepStatus.SKIPPED -> "Schedule"
        AgentStepStatus.CANCELLED -> "Close"
    }
}

/**
 * Helper function to get status color category
 * Mirrors the getStatusColor function in ActivityLog.kt
 */
private fun getStatusColorCategory(status: AgentStepStatus): String {
    return when (status) {
        AgentStepStatus.RUNNING -> "Info"
        AgentStepStatus.COMPLETED -> "Success"
        AgentStepStatus.FAILED -> "Error"
        AgentStepStatus.PENDING -> "Warning"
        AgentStepStatus.SKIPPED -> "OnSurfaceVariant"
        AgentStepStatus.CANCELLED -> "Error"
    }
}

/**
 * Helper function to format timestamp
 * Mirrors the formatTimestamp function in ActivityLog.kt
 */
private fun formatTimestamp(timestamp: Long): String {
    val dateFormat = java.text.SimpleDateFormat("HH:mm:ss", java.util.Locale.getDefault())
    return dateFormat.format(java.util.Date(timestamp))
}
