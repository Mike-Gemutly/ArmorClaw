package com.armorclaw.shared.domain.model

import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class ThreadInfoTest {

    private val json = Json {
        ignoreUnknownKeys = true
    }

    // ========== ThreadInfo Tests ==========

    @Test
    fun `ThreadInfo should serialize correctly`() {
        val threadInfo = ThreadInfo(
            threadRootId = "msg_root_123",
            isThreadReply = true,
            threadDepth = 2,
            threadParticipants = listOf("@user1:example.com", "@user2:example.com"),
            replyCount = 5,
            lastReplyAt = Clock.System.now(),
            lastReplyId = "msg_last_456",
            hasUnread = true,
            unreadCount = 3
        )

        val serialized = json.encodeToString(threadInfo)

        assertNotNull(serialized)
        assertTrue(serialized.contains("msg_root_123"))
        assertTrue(serialized.contains("5"))
    }

    @Test
    fun `ThreadInfo should deserialize correctly`() {
        val jsonStr = """
            {
                "threadRootId": "msg_root_123",
                "isThreadReply": true,
                "threadDepth": 2,
                "threadParticipants": ["@user1:example.com", "@user2:example.com"],
                "replyCount": 5,
                "lastReplyAt": "2024-01-01T00:00:00Z",
                "lastReplyId": "msg_last_456",
                "hasUnread": true,
                "unreadCount": 3
            }
        """.trimIndent()

        val threadInfo = json.decodeFromString<ThreadInfo>(jsonStr)

        assertEquals("msg_root_123", threadInfo.threadRootId)
        assertTrue(threadInfo.isThreadReply)
        assertEquals(2, threadInfo.threadDepth)
        assertEquals(2, threadInfo.threadParticipants.size)
        assertEquals(5, threadInfo.replyCount)
        assertTrue(threadInfo.hasUnread)
        assertEquals(3, threadInfo.unreadCount)
    }

    @Test
    fun `isThreadRoot should return true when threadRootId exists and is not reply`() {
        val threadRoot = ThreadInfo(
            threadRootId = "msg_root_123",
            isThreadReply = false,
            threadDepth = 0
        )
        assertTrue(threadRoot.isThreadRoot())

        val threadReply = ThreadInfo(
            threadRootId = "msg_root_123",
            isThreadReply = true,
            threadDepth = 1
        )
        assertFalse(threadReply.isThreadRoot())

        val noThread = ThreadInfo(
            threadRootId = null,
            isThreadReply = false,
            threadDepth = 0
        )
        assertFalse(noThread.isThreadRoot())
    }

    @Test
    fun `isInThread should return true when threadRootId exists`() {
        val inThread = ThreadInfo(
            threadRootId = "msg_root_123",
            isThreadReply = true
        )
        assertTrue(inThread.isInThread())

        val notInThread = ThreadInfo(
            threadRootId = null
        )
        assertFalse(notInThread.isInThread())
    }

    @Test
    fun `getSummary should return correct text based on reply count`() {
        assertEquals("Start a thread", ThreadInfo(replyCount = 0).getSummary())
        assertEquals("1 reply", ThreadInfo(replyCount = 1).getSummary())
        assertEquals("5 replies", ThreadInfo(replyCount = 5).getSummary())
        assertEquals("100 replies", ThreadInfo(replyCount = 100).getSummary())
    }

    // ========== Message with ThreadInfo Tests ==========

    @Test
    fun `Message should serialize with ThreadInfo`() {
        val message = Message(
            id = "msg_123",
            roomId = "!room:example.com",
            senderId = "@user:example.com",
            content = MessageContent(type = MessageType.TEXT, body = "Thread reply"),
            timestamp = Clock.System.now(),
            isOutgoing = true,
            status = MessageStatus.SENT,
            threadInfo = ThreadInfo(
                threadRootId = "msg_root",
                isThreadReply = true,
                threadDepth = 1
            )
        )

        val serialized = json.encodeToString(message)

        assertNotNull(serialized)
        assertTrue(serialized.contains("threadInfo"))
        assertTrue(serialized.contains("msg_root"))
    }

    @Test
    fun `Message should deserialize with ThreadInfo`() {
        val jsonStr = """
            {
                "id": "msg_123",
                "roomId": "!room:example.com",
                "senderId": "@user:example.com",
                "content": {
                    "type": "TEXT",
                    "body": "Thread reply",
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
                "isDeleted": false,
                "threadInfo": {
                    "threadRootId": "msg_root",
                    "isThreadReply": true,
                    "threadDepth": 1,
                    "threadParticipants": [],
                    "replyCount": 0,
                    "lastReplyAt": null,
                    "lastReplyId": null,
                    "hasUnread": false,
                    "unreadCount": 0
                },
                "reactions": []
            }
        """.trimIndent()

        val message = json.decodeFromString<Message>(jsonStr)

        assertNotNull(message.threadInfo)
        val threadInfo = message.threadInfo!!
        assertEquals("msg_root", threadInfo.threadRootId)
        assertTrue(threadInfo.isThreadReply)
        assertEquals(1, threadInfo.threadDepth)
    }

    // ========== Reaction Tests ==========

    @Test
    fun `Reaction should serialize correctly`() {
        val reaction = Reaction(
            emoji = "👍",
            count = 5,
            includesMe = true,
            reactedBy = listOf("@user1:example.com", "@user2:example.com")
        )

        val serialized = json.encodeToString(reaction)

        assertNotNull(serialized)
        assertTrue(serialized.contains("👍"))
        assertTrue(serialized.contains("5"))
    }

    @Test
    fun `Reaction should deserialize correctly`() {
        val jsonStr = """
            {
                "emoji": "❤️",
                "count": 3,
                "includesMe": false,
                "reactedBy": ["@user1:example.com", "@user2:example.com", "@user3:example.com"]
            }
        """.trimIndent()

        val reaction = json.decodeFromString<Reaction>(jsonStr)

        assertEquals("❤️", reaction.emoji)
        assertEquals(3, reaction.count)
        assertFalse(reaction.includesMe)
        assertEquals(3, reaction.reactedBy.size)
    }

    @Test
    fun `Message should serialize with reactions`() {
        val message = Message(
            id = "msg_123",
            roomId = "!room:example.com",
            senderId = "@user:example.com",
            content = MessageContent(type = MessageType.TEXT, body = "Hello!"),
            timestamp = Clock.System.now(),
            isOutgoing = false,
            status = MessageStatus.DELIVERED,
            reactions = listOf(
                Reaction(emoji = "👍", count = 2, includesMe = true),
                Reaction(emoji = "❤️", count = 1, includesMe = false)
            )
        )

        val serialized = json.encodeToString(message)

        assertTrue(serialized.contains("reactions"))
        assertTrue(serialized.contains("👍"))
        assertTrue(serialized.contains("❤️"))
    }
}
