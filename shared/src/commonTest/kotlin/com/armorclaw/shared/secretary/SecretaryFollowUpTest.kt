package com.armorclaw.shared.secretary

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class SecretaryFollowUpTest {

    @Test
    fun `A1 SecretaryMode has 4 values`() {
        val modes = SecretaryMode.entries
        assertEquals(4, modes.size)
    }

    @Test
    fun `A2 SecretaryMode is an enum`() {
        val modes = SecretaryMode.entries
        assertTrue(modes.contains(SecretaryMode.MEETING))
        assertTrue(modes.contains(SecretaryMode.FOCUS))
        assertTrue(modes.contains(SecretaryMode.SLEEP))
        assertTrue(modes.contains(SecretaryMode.NORMAL))
    }

    @Test
    fun `A3 SecretaryMode has Serializable annotation`() {
        assertTrue(SecretaryMode.MEETING::class.isAnnotationPresent(kotlinx.serialization.Serializable::class))
    }

    @Test
    fun `A1 Triage correctly calculates priority from score`() {
        val triage = SecretaryTriage()

        val resultLow = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "normal message",
            isVipSender = false,
            isCalendarLinked = false
        ))
        assertEquals(0, resultLow.score)
        assertEquals(SecretaryPriority.LOW, resultLow.priority)

        val resultHigh = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "normal message",
            isVipSender = true,
            isCalendarLinked = false
        ))
        assertEquals(1, resultHigh.score)
        assertEquals(SecretaryPriority.HIGH, resultHigh.priority)

        val resultCritical = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "urgent message",
            isVipSender = true,
            isCalendarLinked = false
        ))
        assertEquals(3, resultCritical.score)
        assertEquals(SecretaryPriority.CRITICAL, resultCritical.priority)
    }

    @Test
    fun `A2 Triage applies all factors correctly`() {
        val triage = SecretaryTriage()

        val result1 = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "urgent action needed",
            isVipSender = false,
            isCalendarLinked = false
        ))
        assertEquals(2, result1.score)

        val result2 = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "normal message",
            isVipSender = true,
            isCalendarLinked = false
        ))
        assertEquals(1, result2.score)

        val result3 = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "normal message",
            isVipSender = false,
            isCalendarLinked = true
        ))
        assertEquals(2, result3.score)

        val result4 = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "urgent action needed",
            isVipSender = true,
            isCalendarLinked = true
        ))
        assertEquals(4, result4.score)
    }

    @Test
    fun `A3 Triage determines priority levels correctly`() {
        val triage = SecretaryTriage()

        val low = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "normal message",
            isVipSender = false,
            isCalendarLinked = false
        ))
        assertEquals(SecretaryPriority.LOW, low.priority)

        val high1 = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "urgent message",
            isVipSender = false,
            isCalendarLinked = false
        ))
        assertEquals(SecretaryPriority.HIGH, high1.priority)

        val high2 = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "normal message",
            isVipSender = true,
            isCalendarLinked = false
        ))
        assertEquals(SecretaryPriority.HIGH, high2.priority)

        val critical = triage.score(TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "urgent message",
            isVipSender = true,
            isCalendarLinked = false
        ))
        assertEquals(SecretaryPriority.CRITICAL, critical.priority)
    }

    @Test
    fun `A1 Detect stale threads with 24h threshold`() {
        val followUp = SecretaryFollowUp()
        val currentTime = 1000000000000L
        val twentyFourHoursAgo = currentTime - (24 * 60 * 60 * 1000L) - 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread1",
                    lastOutboundTimestamp = twentyFourHoursAgo,
                    lastInboundTimestamp = null
                )
            ),
            currentTime = currentTime,
            followUpThresholdMs = 24 * 60 * 60 * 1000L
        )

        val result = followUp.detectStaleThreads(context)
        assertEquals(1, result.followUps.size)
        assertEquals("thread1", result.followUps[0].threadId)
    }

    @Test
    fun `A2 No follow-up when recent reply exists`() {
        val followUp = SecretaryFollowUp()
        val currentTime = 1000000000000L
        val twoDaysAgo = currentTime - (48 * 60 * 60 * 1000L)
        val oneDayAgo = currentTime - (24 * 60 * 60 * 1000L)

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread1",
                    lastOutboundTimestamp = twoDaysAgo,
                    lastInboundTimestamp = oneDayAgo
                )
            ),
            currentTime = currentTime,
            followUpThresholdMs = 24 * 60 * 60 * 1000L
        )

        val result = followUp.detectStaleThreads(context)
        assertEquals(0, result.followUps.size)
    }

    @Test
    fun `A3 Follow-up state correctly determined`() {
        val followUp = SecretaryFollowUp()
        val currentTime = 1000000000000L
        val oneDayAgo = currentTime - (24 * 60 * 60 * 1000L) - 1000L

        val thread1 = FollowUpThread(
            threadId = "thread1",
            lastOutboundTimestamp = oneDayAgo,
            lastInboundTimestamp = null
        )

        val context1 = FollowUpContext(
            threads = listOf(thread1),
            currentTime = currentTime,
            followUpThresholdMs = 24 * 60 * 60 * 1000L
        )

        val result1 = followUp.detectStaleThreads(context1)
        assertEquals(1, result1.followUps.size)
        assertEquals("thread1", result1.followUps[0].threadId)

        val recentOutbound = currentTime - (12 * 60 * 60 * 1000L)
        val thread2 = FollowUpThread(
            threadId = "thread2",
            lastOutboundTimestamp = recentOutbound,
            lastInboundTimestamp = null
        )

        val context2 = FollowUpContext(
            threads = listOf(thread2),
            currentTime = currentTime,
            followUpThresholdMs = 24 * 60 * 60 * 1000L
        )

        val result2 = followUp.detectStaleThreads(context2)
        assertEquals(0, result2.followUps.size)
    }

    @Test
    fun `C1 FollowUpContext correctly structured`() {
        val currentTime = 1000000000000L
        val threshold = 24 * 60 * 60 * 1000L

        val context = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "thread1",
                    lastOutboundTimestamp = null,
                    lastInboundTimestamp = currentTime - 1000L
                ),
                FollowUpThread(
                    threadId = "thread2",
                    lastOutboundTimestamp = currentTime - 100000L,
                    lastInboundTimestamp = null
                )
            ),
            currentTime = currentTime,
            followUpThresholdMs = threshold
        )

        assertEquals(2, context.threads.size)
        assertEquals(currentTime, context.currentTime)
        assertEquals(threshold, context.followUpThresholdMs)
    }

    @Test
    fun `C2 detectStaleThreads uses correct time comparison`() {
        val followUp = SecretaryFollowUp()
        val currentTime = 1000000000000L

        val atThreshold = currentTime - (24 * 60 * 60 * 1000L)
        val thread1 = FollowUpThread(
            threadId = "thread1",
            lastOutboundTimestamp = atThreshold,
            lastInboundTimestamp = null
        )

        val overThreshold = currentTime - (24 * 60 * 60 * 1000L) - 1
        val thread2 = FollowUpThread(
            threadId = "thread2",
            lastOutboundTimestamp = overThreshold,
            lastInboundTimestamp = null
        )

        val context = FollowUpContext(
            threads = listOf(thread1, thread2),
            currentTime = currentTime,
            followUpThresholdMs = 24 * 60 * 60 * 1000L
        )

        val result = followUp.detectStaleThreads(context)
        assertEquals(1, result.followUps.size)
        assertEquals("thread2", result.followUps[0].threadId)
    }

    @Test
    fun `C3 Integration triage and follow-up work together`() {
        val triage = SecretaryTriage()
        val followUp = SecretaryFollowUp()
        val currentTime = 1000000000000L

        val triageInput = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "urgent message",
            isVipSender = true,
            isCalendarLinked = false
        )
        val triageResult = triage.score(triageInput)
        assertEquals(SecretaryPriority.CRITICAL, triageResult.priority)

        val oldOutbound = currentTime - (48 * 60 * 60 * 1000L)
        val followUpContext = FollowUpContext(
            threads = listOf(
                FollowUpThread(
                    threadId = "urgent-thread",
                    lastOutboundTimestamp = oldOutbound,
                    lastInboundTimestamp = null
                )
            ),
            currentTime = currentTime,
            followUpThresholdMs = 24 * 60 * 60 * 1000L
        )
        val followUpResult = followUp.detectStaleThreads(followUpContext)
        assertEquals(1, followUpResult.followUps.size)
        assertEquals("urgent-thread", followUpResult.followUps[0].threadId)
    }
}
