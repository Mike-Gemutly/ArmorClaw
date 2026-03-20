package com.armorclaw.shared.secretary

import kotlin.test.Test
import kotlin.test.assertEquals

class SecretaryTriageTest {

    private val triage = SecretaryTriage()

    @Test
    fun `A1 urgent keyword raises priority to HIGH`() {
        val input = TriageInput(
            messageContent = "This is urgent, please review",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.HIGH, result.priority)
    }

    @Test
    fun `A2 asap keyword raises priority to HIGH`() {
        val input = TriageInput(
            messageContent = "Please review ASAP",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.HIGH, result.priority)
    }

    @Test
    fun `A3 emergency keyword raises priority to CRITICAL`() {
        val input = TriageInput(
            messageContent = "Emergency - server is down",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.CRITICAL, result.priority)
    }

    @Test
    fun `A4 Multiple urgent keywords raise to CRITICAL`() {
        val input = TriageInput(
            messageContent = "Urgent - review ASAP",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.CRITICAL, result.priority)
    }

    @Test
    fun `B1 VIP sender raises priority by 1 level`() {
        val input = TriageInput(
            messageContent = "Hello, how are you?",
            isVipSender = true,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.HIGH, result.priority)
    }

    @Test
    fun `B2 VIP sender plus urgent keyword equals CRITICAL`() {
        val input = TriageInput(
            messageContent = "This is urgent, please review",
            isVipSender = true,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.CRITICAL, result.priority)
    }

    @Test
    fun `B3 Non-VIP sender does not raise priority`() {
        val input = TriageInput(
            messageContent = "Hello, how are you?",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.NORMAL, result.priority)
    }

    @Test
    fun `C1 Calendar event link raises priority to HIGH`() {
        val input = TriageInput(
            messageContent = "Meeting notes from event",
            isVipSender = false,
            isCalendarLinked = true
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.HIGH, result.priority)
    }

    @Test
    fun `C2 Calendar link plus VIP sender equals CRITICAL`() {
        val input = TriageInput(
            messageContent = "Meeting notes",
            isVipSender = true,
            isCalendarLinked = true
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.CRITICAL, result.priority)
    }

    @Test
    fun `D1 No special factors equals NORMAL priority`() {
        val input = TriageInput(
            messageContent = "Hello, how are you?",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.NORMAL, result.priority)
    }

    @Test
    fun `D2 All factors combined equals CRITICAL`() {
        val input = TriageInput(
            messageContent = "Urgent: emergency meeting",
            isVipSender = true,
            isCalendarLinked = true
        )

        val result = triage.score(input)

        assertEquals<SecretaryPriority>(SecretaryPriority.CRITICAL, result.priority)
    }

    @Test
    fun `D3 Scoring is deterministic - same input equals same output`() {
        val input = TriageInput(
            messageContent = "Please review ASAP",
            isVipSender = false,
            isCalendarLinked = false
        )

        val result1 = triage.score(input)
        val result2 = triage.score(input)

        assertEquals<SecretaryPriority>(result1.priority, result2.priority)
        assertEquals<Int>(result1.score, result2.score)
    }
}
