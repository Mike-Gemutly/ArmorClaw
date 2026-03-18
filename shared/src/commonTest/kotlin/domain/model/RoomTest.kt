package com.armorclaw.shared.domain.model

import kotlinx.datetime.Clock
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class RoomTest {
    
    private val json = Json {
        ignoreUnknownKeys = true
    }
    
    @Test
    fun `should serialize room correctly`() {
        val room = Room(
            id = "!room:example.com",
            name = "Test Room",
            avatar = "mxc://example.com/avatar.jpg",
            type = RoomType.GROUP,
            membership = Membership.JOIN,
            topic = "Test topic",
            isDirect = false,
            isFavorite = true,
            isMuted = false,
            unreadCount = 5,
            createdAt = Clock.System.now()
        )
        
        val serialized = json.encodeToString(room)
        
        assertNotNull(serialized)
        assert(serialized.contains("Test Room"))
        assert(serialized.contains("GROUP"))
    }
    
    @Test
    fun `should deserialize room correctly`() {
        val jsonStr = """
            {
                "id": "!room:example.com",
                "name": "Test Room",
                "avatar": "mxc://example.com/avatar.jpg",
                "type": "GROUP",
                "membership": "JOIN",
                "topic": "Test topic",
                "isDirect": false,
                "isFavorite": true,
                "isMuted": false,
                "lastMessage": null,
                "unreadCount": 5,
                "members": [],
                "createdAt": "2024-01-01T00:00:00Z"
            }
        """.trimIndent()
        
        val room = json.decodeFromString<Room>(jsonStr)
        
        assertEquals("!room:example.com", room.id)
        assertEquals("Test Room", room.name)
        assertEquals(RoomType.GROUP, room.type)
        assertEquals(Membership.JOIN, room.membership)
        assertEquals(true, room.isFavorite)
        assertEquals(5, room.unreadCount)
    }
    
    @Test
    fun `should create direct room`() {
        val room = Room(
            id = "!direct:example.com",
            name = "John Doe",
            type = RoomType.DIRECT,
            membership = Membership.JOIN,
            isDirect = true,
            createdAt = Clock.System.now()
        )
        
        assertTrue(room.isDirect)
        assertEquals(RoomType.DIRECT, room.type)
    }
    
    @Test
    fun `should create room with members`() {
        val members = listOf(
            RoomMember(
                userId = "@alice:example.com",
                displayName = "Alice",
                membership = Membership.JOIN,
                powerLevel = 50
            ),
            RoomMember(
                userId = "@bob:example.com",
                displayName = "Bob",
                membership = Membership.JOIN,
                powerLevel = 0
            )
        )
        
        val room = Room(
            id = "!room:example.com",
            name = "Test Room",
            type = RoomType.GROUP,
            membership = Membership.JOIN,
            members = members,
            createdAt = Clock.System.now()
        )
        
        assertEquals(2, room.members.size)
        assertEquals("Alice", room.members[0].displayName)
        assertEquals(50, room.members[0].powerLevel)
    }
}
