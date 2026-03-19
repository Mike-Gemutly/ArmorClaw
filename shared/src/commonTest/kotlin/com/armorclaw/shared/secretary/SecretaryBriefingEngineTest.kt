package com.armorclaw.shared.secretary

import org.junit.Before
import org.junit.After
import org.junit.Test
import org.junit.Assert.*

/**
 * Unit tests for SecretaryBriefingEngine following TDD approach.
 *
 * Test Groups:
 * A - Morning Briefing Rules (time windows, no duplication)
 * B - Evening Review Rules (time windows, can be disabled, no duplication)
 * C - Context Quality (insufficient context, partial context)
 * D - Summary Content (unread count, meeting, approvals, action chip)
 */
class SecretaryBriefingEngineTest {

    private lateinit var engine: SecretaryBriefingEngine

    @Before
    fun setup() {
        engine = SecretaryBriefingEngine()
    }

    @After
    fun teardown() {
        // Reset engine state between tests
    }

    @Test
    fun morningBriefing_appearsInConfiguredTimeWindow() {
        val currentTime = hourToMillis(8)
        val context = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Team standup at 10:00 AM",
            pendingApprovals = 2,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNotNull(result)
        assertEquals("Good morning", result?.title)
    }

    @Test
    fun morningBriefing_doesNotAppearOutsideTimeWindow() {
        val currentTime = hourToMillis(10)
        val context = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Team standup at 10:00 AM",
            pendingApprovals = 2,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNull(result)
    }

    @Test
    fun morningBriefing_notDuplicatedOnSameDay() {
        val today = dateToMillis(2025, 3, 19)
        val currentTime = hourToMillis(8) + today
        val context = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Team standup at 10:00 AM",
            pendingApprovals = 2,
            lastMorningBriefingDate = today
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNull(result)
    }

    @Test
    fun eveningReview_appearsInConfiguredEveningWindow() {
        val currentTime = hourToMillis(18)
        val context = BriefingContext(
            unreadCount = 3,
            nextMeeting = null,
            pendingApprovals = 0,
            lastMorningBriefingDate = null,
            lastEveningReviewDate = null,
            eveningReviewEnabled = true
        )

        val result = engine.generateEveningReview(currentTime, context)

        assertNotNull(result)
        assertEquals("Good evening", result?.title)
    }

    @Test
    fun eveningReview_canBeDisabled() {
        val currentTime = hourToMillis(18)
        val context = BriefingContext(
            unreadCount = 3,
            nextMeeting = null,
            pendingApprovals = 0,
            lastMorningBriefingDate = null,
            lastEveningReviewDate = null,
            eveningReviewEnabled = false
        )

        val result = engine.generateEveningReview(currentTime, context)

        assertNull(result)
    }

    @Test
    fun eveningReview_notDuplicatedOnSameDay() {
        val today = dateToMillis(2025, 3, 19)
        val currentTime = hourToMillis(18) + today
        val context = BriefingContext(
            unreadCount = 3,
            nextMeeting = null,
            pendingApprovals = 0,
            lastMorningBriefingDate = null,
            lastEveningReviewDate = today,
            eveningReviewEnabled = true
        )

        val result = engine.generateEveningReview(currentTime, context)

        assertNull(result)
    }

    @Test
    fun briefing_noCardWhenContextInsufficient() {
        val currentTime = hourToMillis(8)
        val context = BriefingContext(
            unreadCount = 0,
            nextMeeting = null,
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNull(result)
    }

    @Test
    fun briefing_partialContextProducesStableSummary() {
        val currentTime = hourToMillis(8)
        val context = BriefingContext(
            unreadCount = 2,
            nextMeeting = null,
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNotNull(result)
        val description = result?.description ?: ""
        assert(description.contains("2 unread") || description.contains("2 message"))
    }

    @Test
    fun briefing_includesUnreadCount() {
        val currentTime = hourToMillis(8)
        val context = BriefingContext(
            unreadCount = 7,
            nextMeeting = "Team standup at 10:00 AM",
            pendingApprovals = 2,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNotNull(result)
        val description = result?.description ?: ""
        assert(description.contains("7"))
    }

    @Test
    fun briefing_includesNextMeeting() {
        val currentTime = hourToMillis(8)
        val context = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Team standup at 10:00 AM",
            pendingApprovals = 2,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNotNull(result)
        val description = result?.description ?: ""
        assert(description.contains("meeting") || description.contains("standup"))
    }

    @Test
    fun briefing_includesPendingApprovals() {
        val currentTime = hourToMillis(8)
        val context = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Team standup at 10:00 AM",
            pendingApprovals = 3,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNotNull(result)
        val description = result?.description ?: ""
        assert(description.contains("3") || description.contains("approval"))
    }

    @Test
    fun briefing_recommendedActionChipPresent() {
        val currentTime = hourToMillis(8)
        val context = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Team standup at 10:00 AM",
            pendingApprovals = 2,
            lastMorningBriefingDate = null
        )

        val result = engine.generateMorningBriefing(currentTime, context)

        assertNotNull(result)
        assertNotNull(result?.primaryAction)
    }

    private fun hourToMillis(hour: Int): Long {
        return hour * 60 * 60 * 1000L
    }

    // Simplified date calculation for test isolation (engine should use proper time library)
    private fun dateToMillis(year: Int, month: Int, day: Int): Long {
        val daysInMonth = intArrayOf(0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31)
        var dayOfYear = day
        for (m in 1 until month) {
            dayOfYear += daysInMonth[m]
        }
        val baseYear2025 = 1735689600000L
        return baseYear2025 + (dayOfYear - 1) * 24 * 60 * 60 * 1000L
    }
}
