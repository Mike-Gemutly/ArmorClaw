package com.armorclaw.shared.domain.model

import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.platform.matrix.event.StepStatus
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for UnifiedMessage model
 *
 * Tests message types, sender identification, and extension functions.
 */
class UnifiedMessageTest {

    // ========================================================================
    // Regular Message Tests
    // ========================================================================

    @Test
    fun `Regular message has correct properties`() {
        val message = UnifiedMessage.Regular(
            id = "msg_1",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender(
                id = "@user:example.com",
                displayName = "John Doe",
                avatarUrl = null,
                isCurrentUser = true
            ),
            content = MessageContent(
                type = MessageType.TEXT,
                body = "Hello, world!"
            ),
            status = MessageStatus.SENT
        )

        assertEquals("msg_1", message.id)
        assertEquals("!room:example.com", message.roomId)
        assertEquals("Hello, world!", message.content.body)
        assertEquals(MessageType.TEXT, message.content.type)
        assertEquals(MessageStatus.SENT, message.status)
    }

    @Test
    fun `Regular message with reply`() {
        val message = UnifiedMessage.Regular(
            id = "msg_2",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender(
                id = "@user:example.com",
                displayName = "Jane Doe",
                avatarUrl = null
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Reply content"),
            replyTo = "msg_1"
        )

        assertEquals("msg_1", message.replyTo)
    }

    @Test
    fun `Regular message with reactions`() {
        val reactions = listOf(
            Reaction(emoji = "👍", count = 2, includesMe = false, reactedBy = listOf("@u1:example.com", "@u2:example.com")),
            Reaction(emoji = "❤️", count = 1, includesMe = false, reactedBy = listOf("@u1:example.com"))
        )
        val message = UnifiedMessage.Regular(
            id = "msg_3",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender(
                id = "@user:example.com",
                displayName = "User",
                avatarUrl = null
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Message"),
            reactions = reactions
        )

        assertEquals(2, message.reactions.size)
        assertEquals("👍", message.reactions[0].emoji)
        assertEquals(2, message.reactions[0].count)
    }

    // ========================================================================
    // Agent Message Tests
    // ========================================================================

    @Test
    fun `Agent message has correct properties`() {
        val message = UnifiedMessage.Agent(
            id = "agent_msg_1",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.AgentSender(
                id = "@agent_analysis:example.com",
                displayName = "Analysis Agent",
                avatarUrl = null,
                agentType = AgentType.ANALYSIS
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Analysis complete!"),
            agentType = AgentType.ANALYSIS,
            confidence = 0.95f
        )

        assertEquals("agent_msg_1", message.id)
        assertEquals(AgentType.ANALYSIS, message.agentType)
        assertEquals(0.95f, message.confidence!!)
    }

    @Test
    fun `Agent message with actions`() {
        val actions = listOf(
            AgentAction(
                id = "action_1",
                label = "Copy",
                actionType = AgentActionType.COPY
            ),
            AgentAction(
                id = "action_2",
                label = "Regenerate",
                actionType = AgentActionType.REGENERATE
            )
        )
        val message = UnifiedMessage.Agent(
            id = "agent_msg_2",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.AgentSender(
                id = "@agent:example.com",
                displayName = "Agent",
                avatarUrl = null,
                agentType = AgentType.GENERAL
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Response"),
            actions = actions
        )

        assertEquals(2, message.actions.size)
        assertEquals("Copy", message.actions[0].label)
        assertEquals(AgentActionType.COPY, message.actions[0].actionType)
    }

    @Test
    fun `Agent message with sources`() {
        val sources = listOf(
            SourceReference(
                id = "src_1",
                type = SourceType.DOCUMENT,
                title = "Document.pdf",
                url = "https://example.com/doc.pdf"
            )
        )
        val message = UnifiedMessage.Agent(
            id = "agent_msg_3",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.AgentSender(
                id = "@agent:example.com",
                displayName = "Agent",
                avatarUrl = null,
                agentType = AgentType.RESEARCH
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Based on sources..."),
            sources = sources
        )

        assertEquals(1, message.sources.size)
        assertEquals("Document.pdf", message.sources[0].title)
        assertEquals(SourceType.DOCUMENT, message.sources[0].type)
    }

    // ========================================================================
    // System Message Tests
    // ========================================================================

    @Test
    fun `System message for workflow started`() {
        val message = UnifiedMessage.System(
            id = "sys_1",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.SystemSender(),
            eventType = SystemEventType.WORKFLOW_STARTED,
            title = "Workflow Started",
            description = "Document analysis workflow has started"
        )

        assertEquals(SystemEventType.WORKFLOW_STARTED, message.eventType)
        assertEquals("Workflow Started", message.title)
    }

    @Test
    fun `System message with actions`() {
        val actions = listOf(
            SystemAction(
                id = "cancel_1",
                label = "Cancel",
                actionType = SystemActionType.CANCEL
            )
        )
        val message = UnifiedMessage.System(
            id = "sys_2",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.SystemSender(),
            eventType = SystemEventType.WORKFLOW_STEP,
            title = "Processing Step 2/5",
            actions = actions
        )

        assertEquals(1, message.actions.size)
        assertEquals("Cancel", message.actions[0].label)
        assertEquals(SystemActionType.CANCEL, message.actions[0].actionType)
    }

    @Test
    fun `System message types`() {
        val eventTypes = listOf(
            SystemEventType.WORKFLOW_STARTED,
            SystemEventType.WORKFLOW_COMPLETED,
            SystemEventType.WORKFLOW_FAILED,
            SystemEventType.USER_JOINED,
            SystemEventType.ENCRYPTION_ENABLED,
            SystemEventType.BUDGET_WARNING
        )

        eventTypes.forEach { type ->
            val message = UnifiedMessage.System(
                id = "sys_${type.name}",
                roomId = "!room:example.com",
                timestamp = Clock.System.now(),
                sender = MessageSender.SystemSender(),
                eventType = type,
                title = type.name
            )

            assertEquals(type, message.eventType)
        }
    }

    // ========================================================================
    // Command Message Tests
    // ========================================================================

    @Test
    fun `Command message has correct properties`() {
        val message = UnifiedMessage.Command(
            id = "cmd_1",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender(
                id = "@user:example.com",
                displayName = "User",
                avatarUrl = null
            ),
            command = "analyze",
            args = listOf("--deep", "--verbose"),
            status = CommandStatus.EXECUTING
        )

        assertEquals("analyze", message.command)
        assertEquals(2, message.args.size)
        assertEquals("--deep", message.args[0])
        assertEquals(CommandStatus.EXECUTING, message.status)
    }

    @Test
    fun `Command message with result`() {
        val message = UnifiedMessage.Command(
            id = "cmd_2",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender(
                id = "@user:example.com",
                displayName = "User",
                avatarUrl = null
            ),
            command = "status",
            status = CommandStatus.COMPLETED,
            result = "System is healthy",
            executionTime = 150L
        )

        assertEquals(CommandStatus.COMPLETED, message.status)
        assertEquals("System is healthy", message.result)
        assertEquals(150L, message.executionTime)
    }

    @Test
    fun `Command message status progression`() {
        val statuses = listOf(
            CommandStatus.PENDING,
            CommandStatus.EXECUTING,
            CommandStatus.COMPLETED,
            CommandStatus.FAILED,
            CommandStatus.CANCELLED
        )

        statuses.forEach { status ->
            val message = UnifiedMessage.Command(
                id = "cmd_status_${status.name}",
                roomId = "!room:example.com",
                timestamp = Clock.System.now(),
                sender = MessageSender.UserSender(
                    id = "@user:example.com",
                    displayName = "User",
                    avatarUrl = null
                ),
                command = "test",
                status = status
            )

            assertEquals(status, message.status)
        }
    }

    // ========================================================================
    // MessageSender Tests
    // ========================================================================

    @Test
    fun `UserSender properties`() {
        val sender = MessageSender.UserSender(
            id = "@user:example.com",
            displayName = "John Doe",
            avatarUrl = "mxc://example.com/avatar",
            isCurrentUser = true,
            isVerified = true
        )

        assertEquals("@user:example.com", sender.id)
        assertEquals("John Doe", sender.displayName)
        assertTrue(sender.isCurrentUser)
        assertTrue(sender.isVerified)
    }

    @Test
    fun `AgentSender properties`() {
        val sender = MessageSender.AgentSender(
            id = "@agent_analysis:example.com",
            displayName = "Analysis Agent",
            avatarUrl = null,
            agentType = AgentType.ANALYSIS,
            capabilities = listOf("analyze", "report", "compare"),
            status = AgentStatus.ONLINE
        )

        assertEquals(AgentType.ANALYSIS, sender.agentType)
        assertEquals(3, sender.capabilities.size)
        assertEquals(AgentStatus.ONLINE, sender.status)
    }

    @Test
    fun `SystemSender is singleton-like`() {
        val sender1 = MessageSender.SystemSender()
        val sender2 = MessageSender.SystemSender()

        assertEquals(sender1.id, sender2.id)
        assertEquals(sender1.displayName, sender2.displayName)
    }

    // ========================================================================
    // Extension Function Tests
    // ========================================================================

    @Test
    fun `isFromCurrentUser for regular message`() {
        val currentUserId = "@current:example.com"
        val message = UnifiedMessage.Regular(
            id = "msg_1",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender(
                id = "@current:example.com",
                displayName = "Current User",
                avatarUrl = null,
                isCurrentUser = true
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Test")
        )

        assertTrue(message.isFromCurrentUser(currentUserId))
    }

    @Test
    fun `isFromCurrentUser for other user message`() {
        val currentUserId = "@current:example.com"
        val message = UnifiedMessage.Regular(
            id = "msg_2",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender(
                id = "@other:example.com",
                displayName = "Other User",
                avatarUrl = null
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Test")
        )

        assertFalse(message.isFromCurrentUser(currentUserId))
    }

    @Test
    fun `isFromCurrentUser for agent message is always false`() {
        val currentUserId = "@user:example.com"
        val message = UnifiedMessage.Agent(
            id = "agent_1",
            roomId = "!room:example.com",
            timestamp = Clock.System.now(),
            sender = MessageSender.AgentSender(
                id = "@agent:example.com",
                displayName = "Agent",
                avatarUrl = null,
                agentType = AgentType.GENERAL
            ),
            content = MessageContent(type = MessageType.TEXT, body = "Response")
        )

        assertFalse(message.isFromCurrentUser(currentUserId))
    }

    @Test
    fun `canReact for different message types`() {
        val regular = UnifiedMessage.Regular(
            id = "1", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("u", "U", null),
            content = MessageContent(type = MessageType.TEXT, body = "")
        )
        val agent = UnifiedMessage.Agent(
            id = "2", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.AgentSender("a", "A", null, AgentType.GENERAL),
            content = MessageContent(type = MessageType.TEXT, body = "")
        )
        val system = UnifiedMessage.System(
            id = "3", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.SystemSender(),
            eventType = SystemEventType.INFO,
            title = "System"
        )
        val command = UnifiedMessage.Command(
            id = "4", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("u", "U", null),
            command = "test"
        )

        assertTrue(regular.canReact())
        assertTrue(agent.canReact())
        assertFalse(system.canReact())
        assertFalse(command.canReact())
    }

    @Test
    fun `canReply for different message types`() {
        val regular = UnifiedMessage.Regular(
            id = "1", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("u", "U", null),
            content = MessageContent(type = MessageType.TEXT, body = "")
        )
        val agent = UnifiedMessage.Agent(
            id = "2", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.AgentSender("a", "A", null, AgentType.GENERAL),
            content = MessageContent(type = MessageType.TEXT, body = "")
        )
        val system = UnifiedMessage.System(
            id = "3", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.SystemSender(),
            eventType = SystemEventType.INFO,
            title = "System"
        )

        assertTrue(regular.canReply())
        assertTrue(agent.canReply())
        assertFalse(system.canReply())
    }

    @Test
    fun `canEdit only for regular messages`() {
        val regular = UnifiedMessage.Regular(
            id = "1", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("u", "U", null),
            content = MessageContent(type = MessageType.TEXT, body = "")
        )
        val agent = UnifiedMessage.Agent(
            id = "2", roomId = "!r", timestamp = Clock.System.now(),
            sender = MessageSender.AgentSender("a", "A", null, AgentType.GENERAL),
            content = MessageContent(type = MessageType.TEXT, body = "")
        )

        assertTrue(regular.canEdit())
        assertFalse(agent.canEdit())
    }

    // ========================================================================
    // AgentType Tests
    // ========================================================================

    @Test
    fun `All agent types are covered`() {
        val types = AgentType.values()
        
        assertEquals(9, types.size)
        assertTrue(AgentType.GENERAL in types)
        assertTrue(AgentType.ANALYSIS in types)
        assertTrue(AgentType.CODE_REVIEW in types)
        assertTrue(AgentType.RESEARCH in types)
        assertTrue(AgentType.WRITING in types)
        assertTrue(AgentType.TRANSLATION in types)
        assertTrue(AgentType.SCHEDULING in types)
        assertTrue(AgentType.WORKFLOW in types)
        assertTrue(AgentType.PLATFORM_BRIDGE in types)
    }

    // ========================================================================
    // AgentStatus Tests
    // ========================================================================

    @Test
    fun `Agent status values`() {
        val statuses = AgentStatus.values()
        
        assertEquals(5, statuses.size)
        assertTrue(AgentStatus.ONLINE in statuses)
        assertTrue(AgentStatus.BUSY in statuses)
        assertTrue(AgentStatus.THINKING in statuses)
        assertTrue(AgentStatus.OFFLINE in statuses)
        assertTrue(AgentStatus.ERROR in statuses)
    }
}
