package com.armorclaw.shared.ui.components

import com.armorclaw.shared.data.store.AgentThinkingState
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for AgentThinkingIndicator component
 *
 * Tests state handling, agent identification, and message display.
 */
class AgentThinkingIndicatorTest {

    // ========================================================================
    // AgentThinkingState Tests
    // ========================================================================

    @Test
    fun `AgentThinkingState has correct properties`() {
        val state = AgentThinkingState(
            agentId = "@agent_analysis:example.com",
            agentName = "Analysis Agent",
            message = "Processing document...",
            timestamp = System.currentTimeMillis()
        )

        assertEquals("@agent_analysis:example.com", state.agentId)
        assertEquals("Analysis Agent", state.agentName)
        assertEquals("Processing document...", state.message)
    }

    @Test
    fun `AgentThinkingState with null message`() {
        val state = AgentThinkingState(
            agentId = "@agent_writer:example.com",
            agentName = "Writer Agent",
            message = null,
            timestamp = System.currentTimeMillis()
        )

        assertEquals(null, state.message)
    }

    @Test
    fun `AgentThinkingState with empty message`() {
        val state = AgentThinkingState(
            agentId = "@agent_test:example.com",
            agentName = "Test Agent",
            message = "",
            timestamp = System.currentTimeMillis()
        )

        assertEquals("", state.message)
    }

    // ========================================================================
    // Agent Identification Tests
    // ========================================================================

    @Test
    fun `Agent ID follows Matrix user format`() {
        val matrixUserId = "@agent_analysis:matrix.example.com"
        val state = AgentThinkingState(
            agentId = matrixUserId,
            agentName = "Analysis Agent",
            message = null,
            timestamp = System.currentTimeMillis()
        )

        assertTrue(state.agentId.startsWith("@"))
        assertTrue(state.agentId.contains(":"))
    }

    @Test
    fun `Multiple agents can be distinguished`() {
        val agents = listOf(
            AgentThinkingState(
                agentId = "@agent_1:example.com",
                agentName = "Agent 1",
                message = null,
                timestamp = System.currentTimeMillis()
            ),
            AgentThinkingState(
                agentId = "@agent_2:example.com",
                agentName = "Agent 2",
                message = null,
                timestamp = System.currentTimeMillis()
            ),
            AgentThinkingState(
                agentId = "@agent_3:example.com",
                agentName = "Agent 3",
                message = null,
                timestamp = System.currentTimeMillis()
            )
        )

        val uniqueIds = agents.map { it.agentId }.toSet()
        assertEquals(3, uniqueIds.size)
    }

    @Test
    fun `Agent thinking states are unique by agentId`() {
        val agentId = "@agent_unique:example.com"
        val state1 = AgentThinkingState(
            agentId = agentId,
            agentName = "Agent",
            message = "Thinking 1",
            timestamp = System.currentTimeMillis()
        )
        val state2 = AgentThinkingState(
            agentId = agentId,
            agentName = "Agent",
            message = "Thinking 2",
            timestamp = System.currentTimeMillis()
        )

        assertEquals(state1.agentId, state2.agentId)
        // Same agent can have different messages
        assertFalse(state1.message == state2.message)
    }

    // ========================================================================
    // Agent Name Display Tests
    // ========================================================================

    @Test
    fun `Agent names can be human-readable`() {
        val state = AgentThinkingState(
            agentId = "@agent_code_review:example.com",
            agentName = "Code Review Agent",
            message = null,
            timestamp = System.currentTimeMillis()
        )

        assertEquals("Code Review Agent", state.agentName)
        assertTrue(state.agentName.contains(" "))
    }

    @Test
    fun `Agent names can be short`() {
        val state = AgentThinkingState(
            agentId = "@agent_abc:example.com",
            agentName = "Bot",
            message = null,
            timestamp = System.currentTimeMillis()
        )

        assertEquals("Bot", state.agentName)
        assertEquals(3, state.agentName.length)
    }

    @Test
    fun `Agent names can contain emoji`() {
        val state = AgentThinkingState(
            agentId = "@agent_ai:example.com",
            agentName = "🤖 AI Assistant",
            message = null,
            timestamp = System.currentTimeMillis()
        )

        assertTrue(state.agentName.contains("🤖"))
    }

    // ========================================================================
    // Thinking Message Tests
    // ========================================================================

    @Test
    fun `Thinking message describes current action`() {
        val messages = listOf(
            "Processing your request...",
            "Analyzing document...",
            "Generating response...",
            "Reviewing code...",
            "Searching knowledge base..."
        )

        messages.forEach { message ->
            val state = AgentThinkingState(
                agentId = "@agent:example.com",
                agentName = "Agent",
                message = message,
                timestamp = System.currentTimeMillis()
            )

            assertTrue(state.message!!.contains("..."))
        }
    }

    @Test
    fun `Thinking message can be long`() {
        val longMessage = "This is a very long thinking message that describes exactly what the agent is doing right now in great detail for the user to understand the process"
        val state = AgentThinkingState(
            agentId = "@agent:example.com",
            agentName = "Verbose Agent",
            message = longMessage,
            timestamp = System.currentTimeMillis()
        )

        assertEquals(longMessage, state.message)
        assertTrue(state.message!!.length > 100)
    }

    // ========================================================================
    // Timestamp Tests
    // ========================================================================

    @Test
    fun `AgentThinkingState has valid timestamp`() {
        val before = System.currentTimeMillis()
        val state = AgentThinkingState(
            agentId = "@agent:example.com",
            agentName = "Agent",
            message = null,
            timestamp = System.currentTimeMillis()
        )
        val after = System.currentTimeMillis()

        assertTrue(state.timestamp >= before)
        assertTrue(state.timestamp <= after)
    }

    @Test
    fun `Multiple thinking states can be ordered by timestamp`() {
        val states = listOf(
            AgentThinkingState(
                agentId = "@agent_1:example.com",
                agentName = "Agent 1",
                message = null,
                timestamp = 1000
            ),
            AgentThinkingState(
                agentId = "@agent_2:example.com",
                agentName = "Agent 2",
                message = null,
                timestamp = 3000
            ),
            AgentThinkingState(
                agentId = "@agent_3:example.com",
                agentName = "Agent 3",
                message = null,
                timestamp = 2000
            )
        )

        val sorted = states.sortedBy { it.timestamp }
        assertEquals(1000L, sorted[0].timestamp)
        assertEquals(2000L, sorted[1].timestamp)
        assertEquals(3000L, sorted[2].timestamp)
    }

    // ========================================================================
    // Data Class Tests
    // ========================================================================

    @Test
    fun `AgentThinkingState is a data class`() {
        val state = AgentThinkingState(
            agentId = "@agent:example.com",
            agentName = "Agent",
            message = "Thinking",
            timestamp = 0
        )

        // Data class should have equals, hashCode, toString, copy
        val copy = state.copy(message = "Still thinking")
        
        assertEquals(state.agentId, copy.agentId)
        assertEquals("Still thinking", copy.message)
    }

    @Test
    fun `AgentThinkingState equality`() {
        val state1 = AgentThinkingState(
            agentId = "@agent:example.com",
            agentName = "Agent",
            message = "Thinking",
            timestamp = 1000
        )
        val state2 = AgentThinkingState(
            agentId = "@agent:example.com",
            agentName = "Agent",
            message = "Thinking",
            timestamp = 1000
        )

        assertEquals(state1, state2)
        assertEquals(state1.hashCode(), state2.hashCode())
    }
}
