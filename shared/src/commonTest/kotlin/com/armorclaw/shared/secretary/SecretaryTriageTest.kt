package com.armorclaw.shared.secretary

import kotlin.test.Test
import kotlin.test.assertEquals

class SecretaryTriageTest {

    private val triage = SecretaryTriage()

    @Test
    fun `A1 Urgent keyword adds 2 points`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "This is urgent, please review",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(2, result.score)
    }

    @Test
    fun `A2 VIP sender adds 1 point`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Hello, how are you?",
            isVipSender = true,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(1, result.score)
    }

    @Test
    fun `A3 Calendar-linked thread adds 2 points`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Meeting notes from event",
            isVipSender = false,
            isCalendarLinked = true
        )

        val result = triage.score(input)

        assertEquals(2, result.score)
    }

    @Test
    fun `A4 Multiple urgent keywords add 2 points`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Urgent - review ASAP",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(2, result.score)
    }

    @Test
    fun `A5 MEETING mode subtracts 1 point`() {
        val input = TriageInput(
            mode = SecretaryMode.MEETING,
            messageContent = "This is urgent, please review",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(1, result.score)
    }

    @Test
    fun `A6 FOCUS mode results in 0 points (whitelist restriction)`() {
        val input = TriageInput(
            mode = SecretaryMode.FOCUS,
            messageContent = "This is urgent, please review",
            isVipSender = true,
            isCalendarLinked = true
        )

        val result = triage.score(input)

        assertEquals(0, result.score)
    }

    @Test
    fun `A7 SLEEP mode results in 0 points`() {
        val input = TriageInput(
            mode = SecretaryMode.SLEEP,
            messageContent = "This is urgent, please review",
            isVipSender = true,
            isCalendarLinked = true
        )

        val result = triage.score(input)

        assertEquals(0, result.score)
    }

    @Test
    fun `A8 NORMAL mode applies no adjustments`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "This is urgent, please review",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(2, result.score)
    }

    @Test
    fun `B1 Score 0 maps to LOW priority`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Hello, how are you?",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(0, result.score)
        assertEquals(SecretaryPriority.LOW, result.priority)
    }

    @Test
    fun `B2 Score 1-2 maps to HIGH priority`() {
        val input1 = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Hello, how are you?",
            isVipSender = true,
            isCalendarLinked = false
        )

        val result1 = triage.score(input1)
        assertEquals(1, result1.score)
        assertEquals(SecretaryPriority.HIGH, result1.priority)

        val input2 = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "This is urgent",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result2 = triage.score(input2)
        assertEquals(2, result2.score)
        assertEquals(SecretaryPriority.HIGH, result2.priority)
    }

    @Test
    fun `B3 Score 3-4 maps to CRITICAL priority`() {
        val input1 = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Urgent: emergency",
            isVipSender = true,
            isCalendarLinked = false
        )

        val result1 = triage.score(input1)
        assertEquals(3, result1.score)
        assertEquals(SecretaryPriority.CRITICAL, result1.priority)

        val input2 = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Urgent: emergency",
            isVipSender = true,
            isCalendarLinked = true
        )

        val result2 = triage.score(input2)
        assertEquals(4, result2.score)
        assertEquals(SecretaryPriority.CRITICAL, result2.priority)
    }

    @Test
    fun `C1 Same input always produces same output`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Please review ASAP",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result1 = triage.score(input)
        val result2 = triage.score(input)
        val result3 = triage.score(input)

        assertEquals(result1.score, result2.score)
        assertEquals(result2.score, result3.score)
        assertEquals(result1.priority, result2.priority)
        assertEquals(result2.priority, result3.priority)
    }

    @Test
    fun `C2 Score calculation is transparent`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Urgent: need ASAP",
            isVipSender = true,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(3, result.score)
        assertEquals(SecretaryPriority.CRITICAL, result.priority)
    }

    @Test
    fun `D1 Triage integrates with policy from SecretaryBriefingEngine`() {
        val input = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = "Urgent: need review",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals(2, result.score)
        assertEquals(SecretaryPriority.HIGH, result.priority)
    }
}
