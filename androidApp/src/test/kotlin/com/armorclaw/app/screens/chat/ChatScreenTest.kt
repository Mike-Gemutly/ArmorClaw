package com.armorclaw.app.screens.chat

import org.junit.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class ChatScreenTest {
    
    @Test
    fun `should have sample messages`() {
        val messages = ChatScreenData.sampleMessages
        assertTrue(messages.isNotEmpty())
        assertEquals(3, messages.size)
    }
    
    @Test
    fun `should have mixed incoming and outgoing messages`() {
        val messages = ChatScreenData.sampleMessages
        val incoming = messages.count { !it.isOutgoing }
        val outgoing = messages.count { it.isOutgoing }
        
        assertTrue(incoming > 0)
        assertTrue(outgoing > 0)
    }
    
    @Test
    fun `should create new message`() {
        val newMessage = Message(
            id = "msg_test",
            content = "Test message",
            isOutgoing = true,
            timestamp = "Now"
        )
        
        assertEquals("msg_test", newMessage.id)
        assertEquals("Test message", newMessage.content)
        assertTrue(newMessage.isOutgoing)
    }
    
    @Test
    fun `should add message to list`() {
        var messages = ChatScreenData.sampleMessages.toMutableList()
        val initialSize = messages.size
        
        val newMessage = Message(
            id = "msg_new",
            content = "New message",
            isOutgoing = true,
            timestamp = "Now"
        )
        
        messages.add(newMessage)
        assertEquals(initialSize + 1, messages.size)
    }
}

object ChatScreenData {
    val sampleMessages = listOf(
        Message(
            id = "msg_1",
            content = "Hello! I'm your AI assistant. How can I help you today?",
            isOutgoing = false,
            timestamp = "2:30 PM"
        ),
        Message(
            id = "msg_2",
            content = "Hi! I need help analyzing some data.",
            isOutgoing = true,
            timestamp = "2:31 PM"
        ),
        Message(
            id = "msg_3",
            content = "Of course! Please share the data you'd like me to analyze.",
            isOutgoing = false,
            timestamp = "2:31 PM"
        )
    )
}

data class Message(
    val id: String,
    val content: String,
    val isOutgoing: Boolean,
    val timestamp: String
)
