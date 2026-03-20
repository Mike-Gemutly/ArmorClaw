package com.armorclaw.shared.secretary

import org.junit.Before
import org.junit.After
import org.junit.Test
import org.junit.Assert.*

/**
 * Unit tests for SecretaryPolicyEngine following TDD approach.
 *
 * Test Groups:
 * A - Mode Detection (MEETING, FOCUS, SLEEP, NORMAL behavior)
 * B - Suppression Rules (urgent vs non-urgent, whitelist)
 * C - Policy Decisions (shouldSuppress, reason, determinism)
 * D - Integration (with SecretaryBriefingEngine, SecretaryViewModel)
 */
class SecretaryPolicyEngineTest {

    private lateinit var engine: SecretaryPolicyEngine

    @Before
    fun setup() {
        engine = SecretaryPolicyEngine()
    }

    @After
    fun teardown() {
        // Reset engine state between tests
    }

    // ========================================
    // Test Group A: Mode Detection
    // ========================================

    @Test
    fun meetingMode_suppressesNonUrgentCards() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(mode = SecretaryMode.MEETING)

        val decision = engine.evaluateCard(card, context)

        assertTrue("Non-urgent card should be suppressed in MEETING mode", decision.shouldSuppress)
        assertEquals("Suppression reason should mention meeting", "Meeting in progress", decision.suppressionReason)
    }

    @Test
    fun focusMode_respectsWhitelist() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(
            mode = SecretaryMode.FOCUS,
            whitelist = listOf(card.id)
        )

        val decision = engine.evaluateCard(card, context)

        assertFalse("Whitelisted card should not be suppressed in FOCUS mode", decision.shouldSuppress)
        assertNull("No suppression reason for whitelisted card", decision.suppressionReason)
    }

    @Test
    fun sleepMode_batchesNormalTraffic() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(mode = SecretaryMode.SLEEP)

        val decision = engine.evaluateCard(card, context)

        assertTrue("Normal cards should be suppressed in SLEEP mode", decision.shouldSuppress)
        assertEquals("Suppression reason should mention batching", "Batching for later review", decision.suppressionReason)
    }

    @Test
    fun normalMode_allowsAllCards() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(mode = SecretaryMode.NORMAL)

        val decision = engine.evaluateCard(card, context)

        assertFalse("All cards should be allowed in NORMAL mode", decision.shouldSuppress)
        assertNull("No suppression reason in NORMAL mode", decision.suppressionReason)
    }

    // ========================================
    // Test Group B: Suppression Rules
    // ========================================

    @Test
    fun urgentCards_bypassSuppressionInAllModes() {
        val card = createTestCard(priority = SecretaryPriority.CRITICAL)
        val modes = listOf(
            SecretaryMode.MEETING,
            SecretaryMode.FOCUS,
            SecretaryMode.SLEEP,
            SecretaryMode.NORMAL
        )

        for (mode in modes) {
            val context = PolicyContext(mode = mode)
            val decision = engine.evaluateCard(card, context)

            assertFalse("Urgent card should bypass suppression in $mode mode", decision.shouldSuppress)
            assertNull("No suppression reason for urgent card in $mode mode", decision.suppressionReason)
        }
    }

    @Test
    fun nonUrgentCards_suppressedInMeetingMode() {
        val priorities = listOf(
            SecretaryPriority.LOW,
            SecretaryPriority.NORMAL,
            SecretaryPriority.HIGH
        )

        for (priority in priorities) {
            val card = createTestCard(priority = priority)
            val context = PolicyContext(mode = SecretaryMode.MEETING)
            val decision = engine.evaluateCard(card, context)

            assertTrue("Non-urgent card ($priority) should be suppressed in MEETING mode", decision.shouldSuppress)
        }
    }

    @Test
    fun nonUrgentCards_suppressedInSleepMode() {
        val priorities = listOf(
            SecretaryPriority.LOW,
            SecretaryPriority.NORMAL,
            SecretaryPriority.HIGH
        )

        for (priority in priorities) {
            val card = createTestCard(priority = priority)
            val context = PolicyContext(mode = SecretaryMode.SLEEP)
            val decision = engine.evaluateCard(card, context)

            assertTrue("Non-urgent card ($priority) should be suppressed in SLEEP mode", decision.shouldSuppress)
        }
    }

    @Test
    fun nonWhitelistCards_suppressedInFocusMode() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(
            mode = SecretaryMode.FOCUS,
            whitelist = emptyList()
        )

        val decision = engine.evaluateCard(card, context)

        assertTrue("Non-whitelisted card should be suppressed in FOCUS mode", decision.shouldSuppress)
        assertEquals("Suppression reason should mention focus", "Focus mode active - not whitelisted", decision.suppressionReason)
    }

    // ========================================
    // Test Group C: Policy Decisions
    // ========================================

    @Test
    fun policyDecision_includesShouldSuppressBoolean() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(mode = SecretaryMode.MEETING)

        val decision = engine.evaluateCard(card, context)

        assertNotNull("Decision should have shouldSuppress field", decision.shouldSuppress)
        assertTrue("Should be suppressible in MEETING mode", decision.shouldSuppress)
    }

    @Test
    fun policyDecision_includesSuppressionReason() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(mode = SecretaryMode.MEETING)

        val decision = engine.evaluateCard(card, context)

        assertNotNull("Decision should have suppressionReason when suppressed", decision.suppressionReason)
    }

    @Test
    fun policyDecision_isDeterministic() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val context = PolicyContext(mode = SecretaryMode.MEETING)

        val decision1 = engine.evaluateCard(card, context)
        val decision2 = engine.evaluateCard(card, context)

        assertEquals("Same input should produce same output (determinism)", decision1, decision2)
    }

    // ========================================
    // Test Group D: Integration
    // ========================================

    @Test
    fun policyEngine_integratesWithBriefingEngine() {
        // Verify that policy decisions can be made based on briefing-generated cards
        val briefingCard = createBriefingCard()
        val context = PolicyContext(mode = SecretaryMode.MEETING)

        val decision = engine.evaluateCard(briefingCard, context)

        assertNotNull("Should be able to evaluate briefing-generated cards", decision)
    }

    @Test
    fun policyEngine_handlesMultipleCardTypes() {
        val cards = listOf(
            createTestCard(priority = SecretaryPriority.CRITICAL),
            createTestCard(priority = SecretaryPriority.NORMAL),
            createTestCard(priority = SecretaryPriority.LOW)
        )
        val context = PolicyContext(mode = SecretaryMode.MEETING)

        val decisions = cards.map { engine.evaluateCard(it, context) }

        assertEquals("Should have decision for each card", cards.size, decisions.size)
        assertFalse("Critical card should not be suppressed", decisions[0].shouldSuppress)
        assertTrue("Normal card should be suppressed", decisions[1].shouldSuppress)
        assertTrue("Low priority card should be suppressed", decisions[2].shouldSuppress)
    }

    @Test
    fun policyEngine_respectsModeTransitions() {
        val card = createTestCard(priority = SecretaryPriority.NORMAL)
        val normalContext = PolicyContext(mode = SecretaryMode.NORMAL)
        val meetingContext = PolicyContext(mode = SecretaryMode.MEETING)

        val normalDecision = engine.evaluateCard(card, normalContext)
        val meetingDecision = engine.evaluateCard(card, meetingContext)

        assertFalse("Should be allowed in NORMAL mode", normalDecision.shouldSuppress)
        assertTrue("Should be suppressed in MEETING mode", meetingDecision.shouldSuppress)
    }

    // ========================================
    // Test Utilities
    // ========================================

    private fun createTestCard(
        id: String = "test-card-1",
        priority: SecretaryPriority = SecretaryPriority.NORMAL
    ): ProactiveCard {
        return ProactiveCard(
            id = id,
            title = "Test Card",
            description = "Test description",
            priority = priority,
            reason = SecretaryCardReason.URGENT_KEYWORD,
            primaryAction = SecretaryAction.Local(LocalSecretaryAction.NAV_CHAT),
            dismissible = true
        )
    }

    private fun createBriefingCard(): ProactiveCard {
        return ProactiveCard(
            id = "morning-briefing-1",
            title = "Good morning",
            description = "5 unread messages. Next meeting: Team standup at 10:00 AM",
            priority = SecretaryPriority.NORMAL,
            reason = SecretaryCardReason.MORNING_BRIEFING,
            primaryAction = SecretaryAction.Local(LocalSecretaryAction.NAV_CHAT),
            dismissible = true
        )
    }
}
