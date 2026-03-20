package com.armorclaw.shared.secretary

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

/**
 * Acceptance tests for SecretaryFollowUp.kt
 *
 * Test Groups:
 * A - Follow-up Detection (4 tests)
 * B - Time Threshold (3 tests)
 * C - Thread Management (3 tests)
 * D - Determinism (3 tests)
 */
class SecretaryFollowUpTest {

    private val engine = SecretaryFollowUp()

    // ========================================
    // Group A - Follow-up Detection (4 tests)
    // ========================================

    @Test
    fun `A1 Thread with outbound message older than threshold needs follow-up`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-1",
                    lastOutboundTimestamp = now - (2 * dayInMs),
                    lastInboundTimestamp = now - (3 * dayInMs)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(1, result.followUps.size)
        val followUp = result.followUps[0]
        assertEquals("thread-1", followUp.threadId)
        assertTrue(followUp.stalenessDurationMs >= dayInMs)
        assertEquals("Send a follow-up message", followUp.recommendedAction)
    }

    @Test
    fun `A2 Thread with recent outbound message doesn't need follow-up`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L
        val hourInMs = 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-2",
                    lastOutboundTimestamp = now - hourInMs,
                    lastInboundTimestamp = now - (2 * dayInMs)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(0, result.followUps.size, "Thread with recent outbound message should not need follow-up")
    }

    @Test
    fun `A3 Thread with recent reply doesn't need follow-up`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-3",
                    lastOutboundTimestamp = now - (2 * dayInMs),
                    lastInboundTimestamp = now - hourInMs
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(0, result.followUps.size, "Thread with recent reply should not need follow-up")
    }

    @Test
    fun `A4 Empty thread list returns empty follow-up list`() {
        val context = FollowUpContext(
            threads = emptyList(),
            currentTime = System.currentTimeMillis(),
            followUpThresholdMs = 24 * 60 * 60 * 1000L
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(0, result.followUps.size, "Empty thread list should return empty follow-up list")
    }

    // ========================================
    // Group B - Time Threshold (3 tests)
    // ========================================

    @Test
    fun `B1 24-hour threshold correctly identifies stale threads`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-stale-1",
                    lastOutboundTimestamp = now - (25 * 60 * 60 * 1000L),
                    lastInboundTimestamp = now - (30 * 60 * 60 * 1000L)
                ),
                FollowUpThread(
                    threadId = "thread-fresh-1",
                    lastOutboundTimestamp = now - (23 * 60 * 60 * 1000L),
                    lastInboundTimestamp = now - (25 * 60 * 60 * 1000L)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(1, result.followUps.size, "Only stale thread (25h) should be flagged")
        assertEquals("thread-stale-1", result.followUps[0].threadId)
    }

    @Test
    fun `B2 48-hour threshold correctly identifies very stale threads`() {
        val now = System.currentTimeMillis()
        val twoDaysInMs = 48 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-very-stale",
                    lastOutboundTimestamp = now - (72 * 60 * 60 * 1000L),
                    lastInboundTimestamp = now - (80 * 60 * 60 * 1000L)
                ),
                FollowUpThread(
                    threadId = "thread-mid-stale",
                    lastOutboundTimestamp = now - (36 * 60 * 60 * 1000L),
                    lastInboundTimestamp = now - (40 * 60 * 60 * 1000L)
                )
            ),
            currentTime = now,
            followUpThresholdMs = twoDaysInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(1, result.followUps.size, "Only very stale thread (72h) should be flagged")
        assertEquals("thread-very-stale", result.followUps[0].threadId)
    }

    @Test
    fun `B3 Custom threshold works correctly`() {
        val now = System.currentTimeMillis()
        val customThreshold = 6 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-7h",
                    lastOutboundTimestamp = now - (7 * 60 * 60 * 1000L),
                    lastInboundTimestamp = now - (8 * 60 * 60 * 1000L)
                ),
                FollowUpThread(
                    threadId = "thread-5h",
                    lastOutboundTimestamp = now - (5 * 60 * 60 * 1000L),
                    lastInboundTimestamp = now - (6 * 60 * 60 * 1000L)
                )
            ),
            currentTime = now,
            followUpThresholdMs = customThreshold
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(1, result.followUps.size, "Only thread older than 6h should be flagged")
        assertEquals("thread-7h", result.followUps[0].threadId)
    }

    // ========================================
    // Group C - Thread Management (3 tests)
    // ========================================

    @Test
    fun `C1 Multiple threads sorted by staleness oldest first`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-2d",
                    lastOutboundTimestamp = now - (2 * dayInMs),
                    lastInboundTimestamp = now - (3 * dayInMs)
                ),
                FollowUpThread(
                    threadId = "thread-5d",
                    lastOutboundTimestamp = now - (5 * dayInMs),
                    lastInboundTimestamp = now - (6 * dayInMs)
                ),
                FollowUpThread(
                    threadId = "thread-3d",
                    lastOutboundTimestamp = now - (3 * dayInMs),
                    lastInboundTimestamp = now - (4 * dayInMs)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(3, result.followUps.size, "All three stale threads should be flagged")
        assertEquals("thread-5d", result.followUps[0].threadId, "Should be sorted oldest first")
        assertEquals("thread-3d", result.followUps[1].threadId)
        assertEquals("thread-2d", result.followUps[2].threadId)
    }

    @Test
    fun `C2 Thread with both inbound and outbound messages handled correctly`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-both",
                    lastOutboundTimestamp = now - (2 * dayInMs), // 2 days ago
                    lastInboundTimestamp = now - (5 * dayInMs)  // 5 days ago
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(1, result.followUps.size)
        val followUp = result.followUps[0]
        assertEquals("thread-both", followUp.threadId)
        assertTrue(followUp.stalenessDurationMs >= dayInMs)
    }

    @Test
    fun `C3 Thread with only inbound messages not flagged for follow-up`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-inbound-only",
                    lastOutboundTimestamp = null,
                    lastInboundTimestamp = now - (3 * dayInMs)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(0, result.followUps.size, "Thread with only inbound messages should not be flagged")
    }

    // ========================================
    // Group D - Determinism (3 tests)
    // ========================================

    @Test
    fun `D1 Same input always produces same output`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-1",
                    lastOutboundTimestamp = now - (2 * dayInMs),
                    lastInboundTimestamp = now - (3 * dayInMs)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result1 = engine.detectStaleThreads(context)
        val result2 = engine.detectStaleThreads(context)

        assertEquals(result1.followUps.size, result2.followUps.size)
        assertEquals(result1.followUps[0].threadId, result2.followUps[0].threadId)
        assertEquals(result1.followUps[0].stalenessDurationMs, result2.followUps[0].stalenessDurationMs)
    }

    @Test
    fun `D2 Follow-up result includes thread ID and staleness duration`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-123",
                    lastOutboundTimestamp = now - (2 * dayInMs),
                    lastInboundTimestamp = now - (3 * dayInMs)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(1, result.followUps.size)
        val followUp = result.followUps[0]
        assertEquals("thread-123", followUp.threadId)
        assertNotNull(followUp.stalenessDurationMs)
        assertTrue(followUp.stalenessDurationMs > 0)
    }

    @Test
    fun `D3 Result includes recommended action`() {
        val now = System.currentTimeMillis()
        val dayInMs = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread-action",
                    lastOutboundTimestamp = now - (2 * dayInMs),
                    lastInboundTimestamp = now - (3 * dayInMs)
                )
            ),
            currentTime = now,
            followUpThresholdMs = dayInMs
        )

        val result = engine.detectStaleThreads(context)

        assertEquals(1, result.followUps.size)
        val followUp = result.followUps[0]
        assertNotNull(followUp.recommendedAction)
        assertFalse(followUp.recommendedAction.isBlank())
    }
}
