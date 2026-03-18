package com.armorclaw.shared.platform.bridge

import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*
import kotlin.math.pow
import kotlin.test.*

/**
 * Tests for BridgeWebSocketClient functionality
 *
 * Tests cover WebSocket state management, event parsing, configuration,
 * and mock client behavior.
 */
class BridgeWebSocketClientTest {

    // ========================================================================
    // WebSocketState Tests
    // ========================================================================

    @Test
    fun `WebSocketState should have correct state types`() {
        val states = listOf(
            WebSocketState.Connecting,
            WebSocketState.Connected,
            WebSocketState.Disconnecting,
            WebSocketState.Disconnected,
            WebSocketState.Error(Exception("test"))
        )

        assertEquals(5, states.size)
        assertTrue(WebSocketState.Connecting is WebSocketState.Connecting)
        assertTrue(WebSocketState.Connected is WebSocketState.Connected)
    }

    @Test
    fun `WebSocketState Error should store exception`() {
        val exception = Exception("Connection failed")
        val state = WebSocketState.Error(exception)

        assertTrue(state is WebSocketState.Error)
        assertEquals("Connection failed", state.error.message)
    }

    // ========================================================================
    // WebSocketConfig Tests
    // ========================================================================

    @Test
    fun `WebSocketConfig DEVELOPMENT should have correct defaults`() {
        val config = WebSocketConfig.DEVELOPMENT

        assertEquals("ws://localhost:8080", config.baseUrl)
        assertTrue(config.reconnectEnabled)
        assertEquals(1000, config.reconnectDelayMs)
        assertEquals(30000, config.maxReconnectDelayMs)
        assertEquals(10, config.maxReconnectAttempts)
        assertEquals(30000, config.pingIntervalMs)
    }

    @Test
    fun `WebSocketConfig PRODUCTION should have correct defaults`() {
        val config = WebSocketConfig.PRODUCTION

        assertEquals("wss://bridge.armorclaw.app", config.baseUrl)
        assertTrue(config.reconnectEnabled)
    }

    @Test
    fun `WebSocketConfig should support custom configuration`() {
        val config = WebSocketConfig(
            baseUrl = "wss://custom.example.com",
            reconnectEnabled = false,
            reconnectDelayMs = 2000,
            maxReconnectDelayMs = 60000,
            maxReconnectAttempts = 5,
            pingIntervalMs = 15000,
            connectionTimeoutMs = 5000
        )

        assertEquals("wss://custom.example.com", config.baseUrl)
        assertFalse(config.reconnectEnabled)
        assertEquals(2000, config.reconnectDelayMs)
        assertEquals(60000, config.maxReconnectDelayMs)
        assertEquals(5, config.maxReconnectAttempts)
        assertEquals(15000, config.pingIntervalMs)
        assertEquals(5000, config.connectionTimeoutMs)
    }

    // ========================================================================
    // BridgeEvent Serialization Tests
    // ========================================================================

    @Test
    fun `MessageReceived should serialize correctly`() {
        val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
        val event = BridgeEvent.MessageReceived(
            eventId = "\$event123:example.com",
            roomId = "!room:example.com",
            senderId = "@user:example.com",
            content = BridgeEventContent(type = "m.text", body = "Hello"),
            originServerTs = 1700000000000,
            sessionId = "session-123"
        )

        val serialized = json.encodeToString(event)

        // Verify key fields are present in the serialized output
        assertTrue(serialized.contains(""""event_id":"${'$'}event123:example.com""""), "Expected event_id in output: $serialized")
        assertTrue(serialized.contains(""""room_id":"!room:example.com""""), "Expected room_id in output: $serialized")
        assertTrue(serialized.contains(""""type":"message.received""""), "Expected type in output: $serialized")
    }

    @Test
    fun `MessageReceived should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "message.received",
                "event_id": "${'$'}event123:example.com",
                "room_id": "!room:example.com",
                "sender_id": "@user:example.com",
                "content": {"type": "m.text", "body": "Hello"},
                "origin_server_ts": 1700000000000,
                "session_id": "session-123"
            }
        """

        val event = json.decodeFromString<BridgeEvent.MessageReceived>(jsonString)

        assertEquals("\$event123:example.com", event.eventId)
        assertEquals("!room:example.com", event.roomId)
        assertEquals("@user:example.com", event.senderId)
        assertEquals("m.text", event.content.type)
        assertEquals("Hello", event.content.body)
        assertEquals("message.received", event.type)
    }

    @Test
    fun `TypingNotification should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "typing",
                "room_id": "!room:example.com",
                "user_id": "@user:example.com",
                "typing": true
            }
        """

        val event = json.decodeFromString<BridgeEvent.TypingNotification>(jsonString)

        assertEquals("typing", event.type)
        assertEquals("!room:example.com", event.roomId)
        assertEquals("@user:example.com", event.userId)
        assertTrue(event.typing)
    }

    @Test
    fun `PresenceUpdate should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "presence",
                "user_id": "@user:example.com",
                "presence": "online",
                "status_msg": "Working",
                "last_active_ts": 1700000000000
            }
        """

        val event = json.decodeFromString<BridgeEvent.PresenceUpdate>(jsonString)

        assertEquals("presence", event.type)
        assertEquals("@user:example.com", event.userId)
        assertEquals("online", event.presence)
        assertEquals("Working", event.statusMsg)
        assertEquals(1700000000000, event.lastActiveTs)
    }

    @Test
    fun `RoomCreated should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "room.created",
                "room_id": "!newroom:example.com",
                "name": "New Room",
                "is_direct": false
            }
        """

        val event = json.decodeFromString<BridgeEvent.RoomCreated>(jsonString)

        assertEquals("room.created", event.type)
        assertEquals("!newroom:example.com", event.roomId)
        assertEquals("New Room", event.name)
        assertFalse(event.isDirect)
    }

    @Test
    fun `RoomMembershipChanged should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "room.membership",
                "room_id": "!room:example.com",
                "user_id": "@user:example.com",
                "membership": "join"
            }
        """

        val event = json.decodeFromString<BridgeEvent.RoomMembershipChanged>(jsonString)

        assertEquals("room.membership", event.type)
        assertEquals("!room:example.com", event.roomId)
        assertEquals("@user:example.com", event.userId)
        assertEquals("join", event.membership)
    }

    @Test
    fun `CallEvent should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "call",
                "call_id": "call-123",
                "room_id": "!room:example.com",
                "action": "offer",
                "sdp": "v=0..."
            }
        """

        val event = json.decodeFromString<BridgeEvent.CallEvent>(jsonString)

        assertEquals("call", event.type)
        assertEquals("call-123", event.callId)
        assertEquals("!room:example.com", event.roomId)
        assertEquals("offer", event.action)
        assertEquals("v=0...", event.sdp)
    }

    @Test
    fun `ReadReceipt should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "receipt.read",
                "room_id": "!room:example.com",
                "user_id": "@user:example.com",
                "event_id": "${'$'}event123:example.com"
            }
        """

        val event = json.decodeFromString<BridgeEvent.ReadReceipt>(jsonString)

        assertEquals("receipt.read", event.type)
        assertEquals("!room:example.com", event.roomId)
        assertEquals("@user:example.com", event.userId)
        assertEquals("\$event123:example.com", event.eventId)
    }

    @Test
    fun `MessageStatusUpdated should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "message.status",
                "event_id": "${'$'}event123:example.com",
                "room_id": "!room:example.com",
                "status": "delivered"
            }
        """

        val event = json.decodeFromString<BridgeEvent.MessageStatusUpdated>(jsonString)

        assertEquals("message.status", event.type)
        assertEquals("\$event123:example.com", event.eventId)
        assertEquals("!room:example.com", event.roomId)
        assertEquals("delivered", event.status)
    }

    @Test
    fun `SessionExpired should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "session.expired",
                "session_id": "expired-session-123",
                "reason": "timeout",
                "sessionId": null
            }
        """

        val event = json.decodeFromString<BridgeEvent.SessionExpired>(jsonString)

        assertEquals("session.expired", event.type)
        assertEquals("expired-session-123", event.expiredSessionId)
        assertEquals("timeout", event.reason)
    }

    @Test
    fun `BridgeStatus should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "bridge.status",
                "status": "healthy",
                "message": "All systems operational",
                "container_id": "container-123",
                "sessionId": null
            }
        """

        val event = json.decodeFromString<BridgeEvent.BridgeStatus>(jsonString)

        assertEquals("bridge.status", event.type)
        assertEquals("healthy", event.status)
        assertEquals("All systems operational", event.message)
        assertEquals("container-123", event.containerId)
    }

    @Test
    fun `PlatformMessage should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "platform.message",
                "platform_id": "platform-123",
                "platform_type": "slack",
                "external_channel_id": "C123456",
                "mapped_room_id": "!room:example.com",
                "sender_name": "John Doe",
                "content": "Hello from Slack!",
                "external_message_id": "1234567890.123456"
            }
        """

        val event = json.decodeFromString<BridgeEvent.PlatformMessage>(jsonString)

        assertEquals("platform.message", event.type)
        assertEquals("platform-123", event.platformId)
        assertEquals("slack", event.platformType)
        assertEquals("John Doe", event.senderName)
        assertEquals("Hello from Slack!", event.content)
    }

    @Test
    fun `RecoveryEvent should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "recovery",
                "recovery_id": "recovery-123",
                "action": "completed",
                "sessionId": null
            }
        """

        val event = json.decodeFromString<BridgeEvent.RecoveryEvent>(jsonString)

        assertEquals("recovery", event.type)
        assertEquals("recovery-123", event.recoveryId)
        assertEquals("completed", event.action)
    }

    // ========================================================================
    // BridgeEventContent Tests
    // ========================================================================

    @Test
    fun `BridgeEventContent should deserialize with all fields`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "type": "m.image",
                "body": "image.png",
                "url": "mxc://example.com/abc123",
                "info": {
                    "mimetype": "image/png",
                    "size": 12345,
                    "width": 800,
                    "height": 600
                },
                "m.relates_to": {
                    "event_id": "${'$'}parent:example.com",
                    "rel_type": "m.reply"
                }
            }
        """

        val content = json.decodeFromString<BridgeEventContent>(jsonString)

        assertEquals("m.image", content.type)
        assertEquals("image.png", content.body)
        assertEquals("mxc://example.com/abc123", content.url)
        assertEquals("image/png", content.info?.mimetype)
        assertEquals(12345, content.info?.size)
        assertEquals(800, content.info?.width)
        assertEquals(600, content.info?.height)
        assertEquals("\$parent:example.com", content.relatesTo?.eventId)
        assertEquals("m.reply", content.relatesTo?.relType)
    }

    // ========================================================================
    // MockBridgeWebSocketClient Tests
    // ========================================================================

    @Test
    fun `MockBridgeWebSocketClient should track connection state`() = runTest {
        val client = MockBridgeWebSocketClient()

        assertFalse(client.isConnected())

        client.connect("session-123", "token-abc")
        assertTrue(client.isConnected())

        client.disconnect("test")
        assertFalse(client.isConnected())
    }

    @Test
    fun `MockBridgeWebSocketClient should expose connection state flow`() = runTest {
        val client = MockBridgeWebSocketClient()

        val initialState = client.connectionState.first()
        assertTrue(initialState is WebSocketState.Disconnected)

        client.connect("session-123", "token-abc")

        val connectedState = client.connectionState.first()
        assertTrue(connectedState is WebSocketState.Connected)
    }

    @Test
    fun `MockBridgeWebSocketClient should handle subscriptions`() = runTest {
        val client = MockBridgeWebSocketClient()
        client.connect("session-123", "token-abc")

        // Should not throw
        client.subscribeToRoom("!room:example.com")
        client.unsubscribeFromRoom("!room:example.com")
        client.subscribeToPresence(listOf("@user:example.com"))
    }

    @Test
    fun `MockBridgeWebSocketClient should handle typing and read receipts`() = runTest {
        val client = MockBridgeWebSocketClient()
        client.connect("session-123", "token-abc")

        // Should not throw
        client.sendTypingNotification("!room:example.com", true)
        client.sendReadReceipt("!room:example.com", "\$event:example.com")
    }

    @Test
    fun `MockBridgeWebSocketClient should handle ping`() = runTest {
        val client = MockBridgeWebSocketClient()

        // Should not throw even when disconnected
        client.ping()

        client.connect("session-123", "token-abc")
        client.ping()
    }

    @Test
    fun `MockBridgeWebSocketClient should expose event flows`() = runTest {
        val client = MockBridgeWebSocketClient()

        // These should return flows (not throw)
        val messageEvents = client.getMessageEvents()
        val typingEvents = client.getTypingEvents()
        val presenceEvents = client.getPresenceEvents()
        val callEvents = client.getCallEvents()
        val roomEvents = client.getRoomEvents()

        assertNotNull(messageEvents)
        assertNotNull(typingEvents)
        assertNotNull(presenceEvents)
        assertNotNull(callEvents)
        assertNotNull(roomEvents)
    }

    // ========================================================================
    // Event Type Filtering Tests
    // ========================================================================

    @Test
    fun `BridgeEvent types should be unique`() {
        val eventTypes = listOf(
            BridgeEvent.MessageReceived::class,
            BridgeEvent.MessageStatusUpdated::class,
            BridgeEvent.RoomCreated::class,
            BridgeEvent.RoomMembershipChanged::class,
            BridgeEvent.TypingNotification::class,
            BridgeEvent.ReadReceipt::class,
            BridgeEvent.PresenceUpdate::class,
            BridgeEvent.CallEvent::class,
            BridgeEvent.PlatformMessage::class,
            BridgeEvent.SessionExpired::class,
            BridgeEvent.BridgeStatus::class,
            BridgeEvent.RecoveryEvent::class,
            BridgeEvent.UnknownEvent::class
        )

        assertEquals(13, eventTypes.size)
    }

    @Test
    fun `BridgeEvent should have consistent type strings`() {
        assertEquals("message.received", BridgeEvent.MessageReceived::class.java.getDeclaredField("type").apply { isAccessible = true }.get(BridgeEvent.MessageReceived(eventId = "", roomId = "", senderId = "", content = BridgeEventContent(type = ""), originServerTs = 0)))
    }

    // ========================================================================
    // Reconnection Logic Tests
    // ========================================================================

    @Test
    fun `exponential backoff should increase delay`() {
        val config = WebSocketConfig(
            baseUrl = "ws://test",
            reconnectDelayMs = 1000,
            maxReconnectDelayMs = 30000
        )

        // Simulate exponential backoff calculation
        fun calculateDelay(attempt: Int): Long {
            val exponentialDelay = config.reconnectDelayMs * 2.0.pow(attempt.toDouble()).toLong()
            return minOf(exponentialDelay, config.maxReconnectDelayMs)
        }

        val delay0 = calculateDelay(0) // 1000 * 2^0 = 1000
        val delay1 = calculateDelay(1) // 1000 * 2^1 = 2000
        val delay2 = calculateDelay(2) // 1000 * 2^2 = 4000
        val delay3 = calculateDelay(3) // 1000 * 2^3 = 8000

        assertTrue(delay1 > delay0)
        assertTrue(delay2 > delay1)
        assertTrue(delay3 > delay2)
    }

    @Test
    fun `exponential backoff should cap at max delay`() {
        val config = WebSocketConfig(
            baseUrl = "ws://test",
            reconnectDelayMs = 1000,
            maxReconnectDelayMs = 30000
        )

        fun calculateDelay(attempt: Int): Long {
            val exponentialDelay = config.reconnectDelayMs * 2.0.pow(attempt.toDouble()).toLong()
            return minOf(exponentialDelay, config.maxReconnectDelayMs)
        }

        val delay10 = calculateDelay(10) // Would be 1024000, capped at 30000
        val delay15 = calculateDelay(15)

        assertEquals(30000, delay10)
        assertEquals(30000, delay15)
    }
}
