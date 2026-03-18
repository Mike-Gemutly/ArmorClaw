package com.armorclaw.shared.domain.model

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for AgentTaskStatusEvent and AgentTaskStatus
 */
class AgentTaskStatusEventTest {

    @Test
    fun `AgentTaskStatus requiresIntervention returns correct values`() {
        // Statuses requiring intervention
        assertTrue(AgentTaskStatus.AWAITING_CAPTCHA.requiresIntervention())
        assertTrue(AgentTaskStatus.AWAITING_2FA.requiresIntervention())
        assertTrue(AgentTaskStatus.AWAITING_APPROVAL.requiresIntervention())

        // Statuses not requiring intervention
        assertFalse(AgentTaskStatus.IDLE.requiresIntervention())
        assertFalse(AgentTaskStatus.BROWSING.requiresIntervention())
        assertFalse(AgentTaskStatus.FORM_FILLING.requiresIntervention())
        assertFalse(AgentTaskStatus.PROCESSING_PAYMENT.requiresIntervention())
        assertFalse(AgentTaskStatus.ERROR.requiresIntervention())
        assertFalse(AgentTaskStatus.COMPLETE.requiresIntervention())
    }

    @Test
    fun `AgentTaskStatus isActive returns correct values`() {
        // Active statuses
        assertTrue(AgentTaskStatus.BROWSING.isActive())
        assertTrue(AgentTaskStatus.FORM_FILLING.isActive())
        assertTrue(AgentTaskStatus.PROCESSING_PAYMENT.isActive())

        // Inactive statuses
        assertFalse(AgentTaskStatus.IDLE.isActive())
        assertFalse(AgentTaskStatus.AWAITING_CAPTCHA.isActive())
        assertFalse(AgentTaskStatus.AWAITING_2FA.isActive())
        assertFalse(AgentTaskStatus.AWAITING_APPROVAL.isActive())
        assertFalse(AgentTaskStatus.ERROR.isActive())
        assertFalse(AgentTaskStatus.COMPLETE.isActive())
    }

    @Test
    fun `AgentTaskStatus toDisplayString returns user-friendly strings`() {
        assertEquals("Idle", AgentTaskStatus.IDLE.toDisplayString())
        assertEquals("Browsing...", AgentTaskStatus.BROWSING.toDisplayString())
        assertEquals("Filling form...", AgentTaskStatus.FORM_FILLING.toDisplayString())
        assertEquals("Processing payment...", AgentTaskStatus.PROCESSING_PAYMENT.toDisplayString())
        assertEquals("Waiting for CAPTCHA", AgentTaskStatus.AWAITING_CAPTCHA.toDisplayString())
        assertEquals("Waiting for 2FA", AgentTaskStatus.AWAITING_2FA.toDisplayString())
        assertEquals("Waiting for approval", AgentTaskStatus.AWAITING_APPROVAL.toDisplayString())
        assertEquals("Error", AgentTaskStatus.ERROR.toDisplayString())
        assertEquals("Complete", AgentTaskStatus.COMPLETE.toDisplayString())
    }

    @Test
    fun `AgentTaskStatusEvent creates with all fields`() {
        val event = AgentTaskStatusEvent(
            agentId = "agent_001",
            status = AgentTaskStatus.FORM_FILLING,
            timestamp = 1234567890L,
            metadata = mapOf("step" to "2", "url" to "https://example.com")
        )

        assertEquals("agent_001", event.agentId)
        assertEquals(AgentTaskStatus.FORM_FILLING, event.status)
        assertEquals(1234567890L, event.timestamp)
        assertEquals("2", event.metadata?.get("step"))
        assertEquals("https://example.com", event.metadata?.get("url"))
    }

    @Test
    fun `AgentTaskStatusEvent creates with null metadata`() {
        val event = AgentTaskStatusEvent(
            agentId = "agent_001",
            status = AgentTaskStatus.BROWSING,
            timestamp = 1234567890L
        )

        assertEquals("agent_001", event.agentId)
        assertEquals(AgentTaskStatus.BROWSING, event.status)
        assertEquals(null, event.metadata)
    }
}
