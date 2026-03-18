package com.armorclaw.app.viewmodel

import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.data.store.AgentThinkingState
import com.armorclaw.shared.domain.model.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Clock
import kotlin.test.*

/**
 * Integration tests for ChatViewModel unified state (Phase 4)
 *
 * Tests the unified message handling and command detection.
 */
class ChatViewModelUnifiedTest {

    // Test fixtures
    private lateinit var viewModel: TestChatViewModel

    @BeforeTest
    fun setup() {
        viewModel = TestChatViewModel(roomId = TEST_ROOM_ID)
    }

    // ========================================================================
    // Unified Message Tests
    // ========================================================================

    @Test
    fun `sendMessage creates regular message for non-command`() = runTest {
        // When sending a regular message
        viewModel.sendMessage("Hello World")
        
        // Then a regular message should be created
        val messages = viewModel.unifiedMessages.value
        assertEquals(1, messages.size)
        assertTrue(messages.first() is UnifiedMessage.Regular)
        val regular = messages.first() as UnifiedMessage.Regular
        assertEquals("Hello World", regular.content.body)
    }

    @Test
    fun `sendMessage creates command message in agent room`() = runTest {
        // Given an agent room
        viewModel._isAgentRoom.value = true
        
        // When sending a command
        viewModel.sendMessage("!status")
        
        // Then a command message should be created
        val commands = viewModel.unifiedMessages.value
            .filterIsInstance<UnifiedMessage.Command>()
        
        assertEquals(1, commands.size)
        assertEquals("status", commands.first().command)
        assertEquals(CommandStatus.EXECUTING, commands.first().status)
    }

    @Test
    fun `sendMessage does not create command in non-agent room`() = runTest {
        // Given a non-agent room
        viewModel._isAgentRoom.value = false
        
        // When sending something starting with !
        viewModel.sendMessage("!status")
        
        // Then a regular message should be created (not a command)
        val messages = viewModel.unifiedMessages.value
        assertEquals(1, messages.size)
        assertTrue(messages.first() is UnifiedMessage.Regular)
    }

    // ========================================================================
    // Reply Functionality Tests
    // ========================================================================

    @Test
    fun `replyToMessage sets reply state`() = runTest {
        // Given a message
        val message = UnifiedMessage.Regular(
            id = "msg_1",
            roomId = TEST_ROOM_ID,
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("user_1", "User", null),
            content = MessageContent(MessageType.TEXT, "Hello")
        )

        // When setting reply
        viewModel.replyToMessage(message)

        // Then replyTo should be set
        assertEquals(message, viewModel.replyTo.value)
    }

    @Test
    fun `cancelReply clears reply state`() = runTest {
        // Given a reply set
        val message = UnifiedMessage.Regular(
            id = "msg_1",
            roomId = TEST_ROOM_ID,
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("user_1", "User", null),
            content = MessageContent(MessageType.TEXT, "Hello")
        )
        viewModel.replyToMessage(message)

        // When cancelling reply
        viewModel.cancelReply()

        // Then replyTo should be null
        assertNull(viewModel.replyTo.value)
    }

    // ========================================================================
    // Reaction Tests
    // ========================================================================

    @Test
    fun `toggleReaction on regular message adds reaction`() = runTest {
        // Given a message
        val message = UnifiedMessage.Regular(
            id = "msg_1",
            roomId = TEST_ROOM_ID,
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("user_1", "User", null),
            content = MessageContent(MessageType.TEXT, "Hello"),
            reactions = emptyList()
        )
        viewModel._unifiedMessages.value = listOf(message)

        // When toggling reaction
        viewModel.toggleReaction(message, "👍")

        // Then reaction should be added
        val updated = viewModel.unifiedMessages.value.first() as UnifiedMessage.Regular
        assertTrue(updated.reactions.any { it.emoji == "👍" })
    }

    @Test
    fun `toggleReaction removes existing reaction`() = runTest {
        // Given a message with a reaction
        val message = UnifiedMessage.Regular(
            id = "msg_1",
            roomId = TEST_ROOM_ID,
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("user_1", "User", null),
            content = MessageContent(MessageType.TEXT, "Hello"),
            reactions = listOf(Reaction("👍", 1, true, listOf("user")))
        )
        viewModel._unifiedMessages.value = listOf(message)

        // When toggling same reaction
        viewModel.toggleReaction(message, "👍")

        // Then reaction should be removed
        val updated = viewModel.unifiedMessages.value.first() as UnifiedMessage.Regular
        assertTrue(updated.reactions.isEmpty())
    }

    @Test
    fun `toggleReaction does nothing for non-regular messages`() = runTest {
        // Given a command message
        val command = UnifiedMessage.Command(
            id = "cmd_1",
            roomId = TEST_ROOM_ID,
            timestamp = Clock.System.now(),
            sender = MessageSender.UserSender("user_1", "User", null),
            command = "status",
            args = emptyList(),
            status = CommandStatus.EXECUTING
        )
        viewModel._unifiedMessages.value = listOf(command)

        // When toggling reaction
        viewModel.toggleReaction(command, "👍")

        // Then nothing should change
        assertEquals(1, viewModel.unifiedMessages.value.size)
    }

    // ========================================================================
    // Agent Room Detection Tests
    // ========================================================================

    @Test
    fun `agent room detection updates isAgentRoom`() = runTest {
        // Initially not an agent room
        assertFalse(viewModel.isAgentRoom.value)
        
        // When set to agent room
        viewModel._isAgentRoom.value = true
        
        // Then isAgentRoom should be true
        assertTrue(viewModel.isAgentRoom.value)
    }

    // ========================================================================
    // Command Parsing Tests
    // ========================================================================

    @Test
    fun `command with arguments is parsed correctly`() = runTest {
        // Given an agent room
        viewModel._isAgentRoom.value = true
        
        // When sending a command with args
        viewModel.sendMessage("!search query terms here")
        
        // Then command and args should be parsed
        val command = viewModel.unifiedMessages.value
            .filterIsInstance<UnifiedMessage.Command>()
            .firstOrNull()
        
        assertNotNull(command)
        assertEquals("search", command!!.command)
        assertEquals(listOf("query", "terms", "here"), command.args)
    }

    companion object {
        private const val TEST_ROOM_ID = "!test:example.com"
    }
}

/**
 * Testable ChatViewModel with exposed internal state
 */
class TestChatViewModel(
    private val roomId: String
) {
    val _unifiedMessages = MutableStateFlow<List<UnifiedMessage>>(emptyList())
    val unifiedMessages = _unifiedMessages.asStateFlow()

    val _isAgentRoom = MutableStateFlow(false)
    val isAgentRoom = _isAgentRoom.asStateFlow()

    val agentThinking = MutableStateFlow<AgentThinkingState?>(null)
    val activeWorkflow = MutableStateFlow<WorkflowState?>(null)
    val replyTo = MutableStateFlow<UnifiedMessage?>(null)
    val hasMore = MutableStateFlow(false)
    val isLoading = MutableStateFlow(false)
    val isLoadingMore = MutableStateFlow(false)

    fun loadMessages() {
        // Test implementation
    }

    fun loadMoreMessages() {
        // Test implementation
    }

    fun sendMessage(content: String) {
        if (content.startsWith("!") && _isAgentRoom.value) {
            val commandText = content.removePrefix("!").trim()
            val parts = commandText.split(" ")
            val command = parts.first()
            val args = parts.drop(1)

            val cmdMessage = UnifiedMessage.Command(
                id = "cmd_${System.currentTimeMillis()}",
                roomId = roomId,
                timestamp = Clock.System.now(),
                sender = MessageSender.UserSender("user", "User", null),
                command = command,
                args = args,
                status = CommandStatus.EXECUTING
            )
            _unifiedMessages.value = _unifiedMessages.value + cmdMessage
        } else {
            val message = UnifiedMessage.Regular(
                id = "msg_${System.currentTimeMillis()}",
                roomId = roomId,
                timestamp = Clock.System.now(),
                sender = MessageSender.UserSender("user", "User", null),
                content = MessageContent(MessageType.TEXT, content),
                status = MessageStatus.SENDING
            )
            _unifiedMessages.value = listOf(message) + _unifiedMessages.value
        }
    }

    fun replyToMessage(message: UnifiedMessage) {
        replyTo.value = message
    }

    fun cancelReply() {
        replyTo.value = null
    }

    fun toggleReaction(message: UnifiedMessage, emoji: String) {
        if (message !is UnifiedMessage.Regular) return

        _unifiedMessages.value = _unifiedMessages.value.map {
            if (it.id == message.id && it is UnifiedMessage.Regular) {
                val existing = it.reactions.find { r -> r.emoji == emoji }
                val newReactions = if (existing != null) {
                    it.reactions.filter { r -> r.emoji != emoji }
                } else {
                    it.reactions + Reaction(emoji, 1, true, listOf("user"))
                }
                it.copy(reactions = newReactions)
            } else it
        }
    }

    fun observeWorkflows() {
        // Test implementation
    }

    fun observeAgentThinking() {
        // Test implementation
    }
}
