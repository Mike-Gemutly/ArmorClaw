package com.armorclaw.shared.domain.model

import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull

class MessageTest {
    
    private val json = Json {
        ignoreUnknownKeys = true
    }
    
    @Test
    fun `should serialize message correctly`() {
        val message = Message(
            id = "msg_123",
            roomId = "!room:example.com",
            senderId = "@user:example.com",
            content = MessageContent(
                type = MessageType.TEXT,
                body = "Hello, world!"
            ),
            timestamp = Clock.System.now(),
            isOutgoing = true,
            status = MessageStatus.SENT
        )
        
        val serialized = json.encodeToString(message)
        
        assertNotNull(serialized)
        assert(serialized.contains("msg_123"))
        assert(serialized.contains("Hello, world!"))
    }
    
    @Test
    fun `should deserialize message correctly`() {
        val jsonStr = """
            {
                "id": "msg_123",
                "roomId": "!room:example.com",
                "senderId": "@user:example.com",
                "content": {
                    "type": "TEXT",
                    "body": "Hello, world!",
                    "formattedBody": null,
                    "attachments": [],
                    "mentions": []
                },
                "timestamp": "2024-01-01T00:00:00Z",
                "isOutgoing": true,
                "status": "SENT",
                "serverTimestamp": null,
                "editCount": 0,
                "replyTo": null,
                "isDeleted": false
            }
        """.trimIndent()
        
        val message = json.decodeFromString<Message>(jsonStr)
        
        assertEquals("msg_123", message.id)
        assertEquals("!room:example.com", message.roomId)
        assertEquals("@user:example.com", message.senderId)
        assertEquals(MessageType.TEXT, message.content.type)
        assertEquals("Hello, world!", message.content.body)
        assertEquals(true, message.isOutgoing)
        assertEquals(MessageStatus.SENT, message.status)
    }
    
    @Test
    fun `should create message with attachments`() {
        val message = Message(
            id = "msg_456",
            roomId = "!room:example.com",
            senderId = "@user:example.com",
            content = MessageContent(
                type = MessageType.IMAGE,
                body = "Check this image",
                attachments = listOf(
                    Attachment(
                        url = "https://example.com/image.jpg",
                        mimeType = "image/jpeg",
                        size = 1024000,
                        fileName = "image.jpg"
                    )
                )
            ),
            timestamp = Clock.System.now(),
            isOutgoing = false,
            status = MessageStatus.DELIVERED
        )
        
        assertEquals(1, message.content.attachments.size)
        assertEquals("image/jpeg", message.content.attachments[0].mimeType)
    }
}
