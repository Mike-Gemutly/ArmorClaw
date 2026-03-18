package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*
import kotlin.test.*

/**
 * Tests for BridgeRpcClient functionality
 *
 * Tests cover RpcResult handling, configuration, model serialization,
 * and mock client behavior. Integration tests with actual HTTP calls
 * would require platform-specific test setup.
 */
class BridgeRpcClientTest {

    // ========================================================================
    // RpcResult Tests
    // ========================================================================

    @Test
    fun `RpcResult Success should return true for isSuccess`() {
        val result: RpcResult<String> = RpcResult.success("test")
        assertTrue(result.isSuccess)
        assertFalse(result.isError)
    }

    @Test
    fun `RpcResult Error should return true for isError`() {
        val result: RpcResult<String> = RpcResult.error(-32601, "Method not found")
        assertTrue(result.isError)
        assertFalse(result.isSuccess)
    }

    @Test
    fun `RpcResult getOrNull should return value on success`() {
        val result: RpcResult<String> = RpcResult.success("test-value")
        assertEquals("test-value", result.getOrNull())
    }

    @Test
    fun `RpcResult getOrNull should return null on error`() {
        val result: RpcResult<String> = RpcResult.error(-32601, "Method not found")
        assertNull(result.getOrNull())
    }

    @Test
    fun `RpcResult getOrThrow should return value on success`() {
        val result: RpcResult<String> = RpcResult.success("test-value")
        assertEquals("test-value", result.getOrThrow())
    }

    @Test
    fun `RpcResult getOrThrow should throw RpcException on error`() {
        val result: RpcResult<String> = RpcResult.error(-32601, "Method not found", mapOf("detail" to "info"))
        val exception = assertFailsWith<RpcException> { result.getOrThrow() }
        assertEquals(-32601, exception.code)
        assertEquals("Method not found", exception.message)
        assertEquals(mapOf("detail" to "info"), exception.data)
    }

    @Test
    fun `RpcResult map should transform success value`() {
        val result: RpcResult<Int> = RpcResult.success(5)
        val mapped = result.map { it * 2 }
        assertTrue(mapped.isSuccess)
        assertEquals(10, mapped.getOrNull())
    }

    @Test
    fun `RpcResult map should preserve error`() {
        val result: RpcResult<Int> = RpcResult.error(-32601, "Error")
        val mapped = result.map { it * 2 }
        assertTrue(mapped.isError)
        assertEquals(-32601, (mapped as RpcResult.Error).code)
    }

    @Test
    fun `RpcResult Error should store metadata`() {
        val result: RpcResult<String> = RpcResult.error(
            code = -32001,
            message = "Auth failed",
            data = mapOf(
                "correlation_id" to "corr-123",
                "trace_id" to "trace-456",
                "attempt" to 1
            )
        )

        assertTrue(result.isError)
        val error = result as RpcResult.Error
        assertEquals(-32001, error.code)
        assertEquals("Auth failed", error.message)
        assertEquals("corr-123", error.data?.get("correlation_id"))
    }

    // ========================================================================
    // BridgeConfig Tests
    // ========================================================================

    @Test
    fun `BridgeConfig PRODUCTION should have correct defaults`() {
        val config = BridgeConfig.PRODUCTION
        assertEquals("https://bridge.armorclaw.app", config.baseUrl)
        assertEquals("https://matrix.armorclaw.app", config.homeserverUrl)
        assertNull(config.wsUrl)  // WebSocket not yet implemented
        assertTrue(config.enableCertificatePinning)
        assertTrue(config.useDirectMatrixSync)
    }

    @Test
    fun `BridgeConfig DEVELOPMENT should have correct defaults`() {
        val config = BridgeConfig.DEVELOPMENT
        assertEquals("http://10.0.2.2:8080", config.baseUrl)
        assertEquals("http://10.0.2.2:8008", config.homeserverUrl)
        assertNull(config.wsUrl)  // WebSocket not yet implemented
        assertFalse(config.enableCertificatePinning)
        assertEquals(1, config.retryCount)
        assertTrue(config.useDirectMatrixSync)
    }

    @Test
    fun `BridgeConfig should support custom configuration`() {
        val config = BridgeConfig(
            baseUrl = "https://custom.example.com",
            homeserverUrl = "https://matrix.custom.example.com",
            timeoutMs = 60000,
            retryCount = 5,
            retryDelayMs = 2000,
            enableCertificatePinning = false,
            certificatePins = listOf("sha256/ABC123")
        )

        assertEquals("https://custom.example.com", config.baseUrl)
        assertEquals(60000, config.timeoutMs)
        assertEquals(5, config.retryCount)
        assertEquals(2000, config.retryDelayMs)
        assertFalse(config.enableCertificatePinning)
        assertEquals(1, config.certificatePins.size)
    }

    // ========================================================================
    // JSON-RPC Request Serialization Tests
    // ========================================================================

    @Test
    fun `JsonRpcRequest should serialize correctly`() {
        val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
        val request = JsonRpcRequest(
            jsonrpc = "2.0",
            method = "bridge.start",
            params = mapOf("user_id" to JsonPrimitive("user123")),
            id = "req-123"
        )

        val serialized = json.encodeToString(request)

        assertTrue(serialized.contains("\"jsonrpc\":\"2.0\""))
        assertTrue(serialized.contains("\"method\":\"bridge.start\""))
        assertTrue(serialized.contains("\"id\":\"req-123\""))
        assertTrue(serialized.contains("\"user_id\":\"user123\""))
    }

    @Test
    fun `JsonRpcRequest should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """{"jsonrpc":"2.0","method":"matrix.login","id":"req-456","params":{"username":"test"}}"""

        val request = json.decodeFromString<JsonRpcRequest>(jsonString)

        assertEquals("2.0", request.jsonrpc)
        assertEquals("matrix.login", request.method)
        assertEquals("req-456", request.id)
        assertNotNull(request.params)
    }

    // ========================================================================
    // Response Model Tests
    // ========================================================================

    @Test
    fun `BridgeStartResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "session_id": "sess-123",
                "container_id": "container-456",
                "status": "running",
                "ice_servers": [{"urls": ["stun:stun.example.com"]}]
            }
        """

        val response = json.decodeFromString<BridgeStartResponse>(jsonString)

        assertEquals("sess-123", response.sessionId)
        assertEquals("container-456", response.containerId)
        assertEquals("running", response.status)
        assertEquals(1, response.iceServers?.size)
    }

    @Test
    fun `MatrixLoginResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "access_token": "token-abc",
                "refresh_token": "refresh-def",
                "device_id": "DEVICE123",
                "user_id": "@user:example.com",
                "display_name": "Test User"
            }
        """

        val response = json.decodeFromString<MatrixLoginResponse>(jsonString)

        assertEquals("token-abc", response.accessToken)
        assertEquals("refresh-def", response.refreshToken)
        assertEquals("DEVICE123", response.deviceId)
        assertEquals("@user:example.com", response.userId)
        assertEquals("Test User", response.displayName)
    }

    @Test
    fun `MatrixSyncResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "next_batch": "batch_token_123",
                "rooms": {
                    "join": {
                        "!room1:example.com": {
                            "timeline": {
                                "events": [],
                                "limited": false
                            }
                        }
                    }
                }
            }
        """

        val response = json.decodeFromString<MatrixSyncResponse>(jsonString)

        assertEquals("batch_token_123", response.nextBatch)
        assertNotNull(response.rooms)
        assertNotNull(response.rooms?.join)
        assertTrue(response.rooms?.join!!.containsKey("!room1:example.com"))
    }

    @Test
    fun `MatrixSendResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """{"event_id":"${'$'}event123:example.com","txn_id":"txn-456"}"""

        val response = json.decodeFromString<MatrixSendResponse>(jsonString)

        assertEquals("\$event123:example.com", response.eventId)
        assertEquals("txn-456", response.txnId)
    }

    @Test
    fun `RecoveryPhraseResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "phrase": "word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12",
                "word_count": 12,
                "created": 1700000000000
            }
        """

        val response = json.decodeFromString<RecoveryPhraseResponse>(jsonString)

        assertEquals(12, response.wordCount)
        assertEquals(12, response.phrase.split(" ").size)
        assertEquals(1700000000000, response.created)
    }

    @Test
    fun `RecoveryVerifyResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "valid": true,
                "recovery_id": "recovery-123",
                "expires_at": 1700000000000
            }
        """

        val response = json.decodeFromString<RecoveryVerifyResponse>(jsonString)

        assertTrue(response.valid)
        assertEquals("recovery-123", response.recoveryId)
        assertEquals(1700000000000, response.expiresAt)
    }

    @Test
    fun `PlatformConnectResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "success": true,
                "platform_id": "platform-123",
                "auth_url": "https://oauth.example.com/authorize"
            }
        """

        val response = json.decodeFromString<PlatformConnectResponse>(jsonString)

        assertTrue(response.success)
        assertEquals("platform-123", response.platformId)
        assertEquals("https://oauth.example.com/authorize", response.authUrl)
    }

    // ========================================================================
    // JsonRpcError Tests
    // ========================================================================

    @Test
    fun `JsonRpcError should have standard error codes`() {
        assertEquals(-32700, JsonRpcError.PARSE_ERROR)
        assertEquals(-32600, JsonRpcError.INVALID_REQUEST)
        assertEquals(-32601, JsonRpcError.METHOD_NOT_FOUND)
        assertEquals(-32602, JsonRpcError.INVALID_PARAMS)
        assertEquals(-32603, JsonRpcError.INTERNAL_ERROR)
    }

    @Test
    fun `JsonRpcError should have ArmorClaw-specific error codes`() {
        assertEquals(-32001, JsonRpcError.AUTH_FAILED)
        assertEquals(-32002, JsonRpcError.SESSION_EXPIRED)
        assertEquals(-32003, JsonRpcError.DEVICE_NOT_VERIFIED)
        assertEquals(-32004, JsonRpcError.ROOM_NOT_FOUND)
        assertEquals(-32005, JsonRpcError.MESSAGE_SEND_FAILED)
        assertEquals(-32006, JsonRpcError.NETWORK_ERROR)
        assertEquals(-32007, JsonRpcError.RATE_LIMITED)
    }

    @Test
    fun `JsonRpcError should serialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val error = JsonRpcError(
            code = -32001,
            message = "Authentication failed",
            data = JsonObject(mapOf("reason" to JsonPrimitive("invalid_token")))
        )

        val serialized = json.encodeToString(error)

        assertTrue(serialized.contains("\"code\":-32001"))
        assertTrue(serialized.contains("\"message\":\"Authentication failed\""))
    }

    // ========================================================================
    // MockBridgeRpcClient Tests
    // ========================================================================

    @Test
    fun `MockBridgeRpcClient should return success for startBridge`() = runTest {
        val client = MockBridgeRpcClient()

        val result = client.startBridge("user123", "device456")

        assertTrue(result.isSuccess)
        assertNotNull(result.getOrNull()?.sessionId)
        assertEquals("test-session-id", result.getOrNull()?.sessionId)
    }

    @Test
    fun `MockBridgeRpcClient should return success for matrixLogin`() = runTest {
        val client = MockBridgeRpcClient()

        val result = client.matrixLogin(
            homeserver = "https://matrix.example.com",
            username = "testuser",
            password = "testpass",
            deviceId = "DEVICE123"
        )

        assertTrue(result.isSuccess)
        assertEquals("test-token", result.getOrNull()?.accessToken)
        assertEquals("testuser", result.getOrNull()?.userId)
    }

    @Test
    fun `MockBridgeRpcClient should return success for matrixSend`() = runTest {
        val client = MockBridgeRpcClient()

        val result = client.matrixSend(
            roomId = "!room:example.com",
            eventType = "m.room.message",
            content = mapOf("msgtype" to "m.text", "body" to "Hello")
        )

        assertTrue(result.isSuccess)
        assertEquals("\$event-123", result.getOrNull()?.eventId)
    }

    @Test
    fun `MockBridgeRpcClient should return success for healthCheck`() = runTest {
        val client = MockBridgeRpcClient()

        val result = client.healthCheck()

        assertTrue(result.isSuccess)
        assertEquals("healthy", result.getOrNull()?.get("status"))
    }

    @Test
    fun `MockBridgeRpcClient should return success for recovery methods`() = runTest {
        val client = MockBridgeRpcClient()

        val generateResult = client.recoveryGeneratePhrase()
        assertTrue(generateResult.isSuccess)
        assertEquals(12, generateResult.getOrNull()?.wordCount)

        val verifyResult = client.recoveryVerify("word1 word2 word3")
        assertTrue(verifyResult.isSuccess)
        assertTrue(verifyResult.getOrNull()?.valid == true)
    }

    @Test
    fun `MockBridgeRpcClient should return success for platform methods`() = runTest {
        val client = MockBridgeRpcClient()

        val connectResult = client.platformConnect("slack", mapOf("token" to "xxx"))
        assertTrue(connectResult.isSuccess)
        assertEquals("platform-123", connectResult.getOrNull()?.platformId)

        val listResult = client.platformList()
        assertTrue(listResult.isSuccess)

        val testResult = client.platformTest("platform-123")
        assertTrue(testResult.isSuccess)
        assertTrue(testResult.getOrNull()?.success == true)
    }

    @Test
    fun `MockBridgeRpcClient should track connection state`() = runTest {
        val client = MockBridgeRpcClient()

        assertTrue(client.isConnected())
        assertEquals("test-session-id", client.getSessionId())
    }

    // ========================================================================
    // OperationContext Tests for RPC
    // ========================================================================

    @Test
    fun `OperationContext should create with auto-generated IDs`() {
        val context = OperationContext.create()

        assertNotNull(context.correlationId)
        assertNotNull(context.traceId)
        assertTrue(context.correlationId.isNotEmpty())
        assertTrue(context.traceId!!.isNotEmpty())
    }

    @Test
    fun `OperationContext should extract from headers`() {
        val headers = mapOf(
            "X-Request-ID" to "req-123",
            "X-Correlation-ID" to "corr-456",
            "X-Trace-ID" to "trace-789"
        )

        val context = OperationContext.fromHeaders(headers)

        assertEquals("corr-456", context.correlationId)
        assertEquals("trace-789", context.traceId)
    }

    @Test
    fun `OperationContext should generate IDs when headers missing`() {
        val headers = emptyMap<String, String>()

        val context = OperationContext.fromHeaders(headers)

        assertNotNull(context.correlationId)
        assertNotNull(context.traceId)
    }

    // ========================================================================
    // IceServer Model Tests
    // ========================================================================

    @Test
    fun `IceServer should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "urls": ["stun:stun.example.com:3478", "turn:turn.example.com:3478"],
                "username": "user123",
                "credential": "pass123"
            }
        """

        val server = json.decodeFromString<RpcIceServer>(jsonString)

        assertEquals(2, server.urls.size)
        assertEquals("user123", server.username)
        assertEquals("pass123", server.credential)
    }

    // ========================================================================
    // WebRTC Model Tests
    // ========================================================================

    @Test
    fun `WebRtcSignalingResponse should deserialize correctly`() {
        val json = Json { ignoreUnknownKeys = true }
        val jsonString = """
            {
                "sdp": "v=0...",
                "type": "answer",
                "ice_candidates": [
                    {"candidate": "candidate:1", "sdp_mid": "0", "sdp_mline_index": 0}
                ]
            }
        """

        val response = json.decodeFromString<WebRtcSignalingResponse>(jsonString)

        assertEquals("v=0...", response.sdp)
        assertEquals("answer", response.type)
        assertEquals(1, response.iceCandidates?.size)
    }
}
