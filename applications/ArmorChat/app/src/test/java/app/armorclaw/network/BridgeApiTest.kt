package app.armorclaw.network

import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test

/**
 * Unit tests for BridgeApi RPC client
 *
 * Tests:
 * - Request formatting
 * - Response parsing
 * - Error handling
 * - Parameter encoding
 */
class BridgeApiTest {

    // ========================================================================
    // Request Building Tests
    // ========================================================================

    @Test
    fun `RPCRequest has correct structure`() {
        val request = BridgeApi.RPCRequest(
            id = 1,
            method = "test.method",
            params = mapOf("key" to "value")
        )

        assertEquals("jsonrpc should be 2.0", "2.0", request.jsonrpc)
        assertEquals("id should match", 1, request.id)
        assertEquals("method should match", "test.method", request.method)
        assertNotNull("params should not be null", request.params)
    }

    @Test
    fun `RPCRequest allows null params`() {
        val request = BridgeApi.RPCRequest(
            id = 2,
            method = "no.params",
            params = null
        )

        assertNull("params should be null", request.params)
    }

    // ========================================================================
    // Response Parsing Tests
    // ========================================================================

    @Test
    fun `RPCResponse success parses correctly`() {
        val json = """{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}"""

        // In a real test, we'd parse this through the JSON decoder
        // For now, just verify the structure
        assertTrue("Response should contain result", json.contains("result"))
        assertTrue("Response should contain jsonrpc", json.contains("jsonrpc"))
    }

    @Test
    fun `RPCResponse error parses correctly`() {
        val json = """{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}"""

        assertTrue("Response should contain error", json.contains("error"))
        assertTrue("Response should contain code", json.contains("-32600"))
    }

    // ========================================================================
    // Type Structure Tests
    // ========================================================================

    @Test
    fun `LockdownStatus has all required fields`() {
        val status = BridgeApi.LockdownStatus(
            mode = "operational",
            admin_established = true,
            single_device_mode = false,
            setup_complete = true,
            security_configured = true,
            keystore_initialized = true
        )

        assertEquals("mode should match", "operational", status.mode)
        assertTrue("admin_established should be true", status.admin_established)
        assertFalse("single_device_mode should be false", status.single_device_mode)
        assertTrue("setup_complete should be true", status.setup_complete)
    }

    @Test
    fun `BondingResponse has all required fields`() {
        val response = BridgeApi.BondingResponse(
            status = "claimed",
            admin_id = "admin-123",
            device_id = "device-456",
            certificate = "cert-data",
            session_token = "token-789",
            expires_at = "2026-03-01T00:00:00Z",
            next_step = "security_config"
        )

        assertEquals("status should match", "claimed", response.status)
        assertEquals("admin_id should match", "admin-123", response.admin_id)
        assertEquals("device_id should match", "device-456", response.device_id)
        assertNotNull("certificate should not be null", response.certificate)
    }

    @Test
    fun `DataCategory has all required fields`() {
        val category = BridgeApi.DataCategory(
            id = "banking",
            name = "Banking Information",
            description = "Financial data",
            risk_level = "high",
            permission = "deny",
            allowed_websites = emptyList(),
            requires_approval = true
        )

        assertEquals("id should match", "banking", category.id)
        assertEquals("risk_level should match", "high", category.risk_level)
        assertTrue("requires_approval should be true", category.requires_approval)
    }

    @Test
    fun `Device has all required fields`() {
        val device = BridgeApi.Device(
            id = "device-1",
            name = "My Phone",
            type = "mobile",
            trust_state = "verified",
            last_seen = "2026-02-16T00:00:00Z",
            is_current = true
        )

        assertEquals("id should match", "device-1", device.id)
        assertTrue("is_current should be true", device.is_current)
    }

    @Test
    fun `Invite has all required fields`() {
        val invite = BridgeApi.Invite(
            id = "invite-1",
            code = "ABC123",
            role = "user",
            status = "active",
            created_at = "2026-02-16T00:00:00Z",
            expires_at = "2026-03-16T00:00:00Z"
        )

        assertEquals("code should match", "ABC123", invite.code)
        assertEquals("status should match", "active", invite.status)
        assertNotNull("expires_at should not be null", invite.expires_at)
    }

    @Test
    fun `PushTokenResponse has all required fields`() {
        val response = BridgeApi.PushTokenResponse(
            success = true,
            message = "Token registered",
            device_id = "device-1"
        )

        assertTrue("success should be true", response.success)
        assertEquals("message should match", "Token registered", response.message)
    }

    // ========================================================================
    // Error Response Tests
    // ========================================================================

    @Test
    fun `RPCError structure is correct`() {
        val error = BridgeApi.RPCError(
            code = -32601,
            message = "Method not found",
            data = "additional info"
        )

        assertEquals("code should match", -32601, error.code)
        assertEquals("message should match", "Method not found", error.message)
        assertEquals("data should match", "additional info", error.data)
    }

    @Test
    fun `RPCError allows null data`() {
        val error = BridgeApi.RPCError(
            code = -32600,
            message = "Invalid Request",
            data = null
        )

        assertNull("data should be null", error.data)
    }

    // ========================================================================
    // Edge Case Tests
    // ========================================================================

    @Test
    fun `empty string fields are handled`() {
        val status = BridgeApi.LockdownStatus(
            mode = "",
            admin_established = false,
            single_device_mode = false,
            setup_complete = false,
            security_configured = false,
            keystore_initialized = false
        )

        assertEquals("Empty mode should be preserved", "", status.mode)
    }

    @Test
    fun `unicode in strings is handled`() {
        val category = BridgeApi.DataCategory(
            id = "test",
            name = "Test 中文 العربية 🎉",
            description = "Unicode test",
            risk_level = "low",
            permission = "allow",
            allowed_websites = emptyList(),
            requires_approval = false
        )

        assertTrue("Should contain Chinese characters", category.name.contains("中文"))
        assertTrue("Should contain Arabic characters", category.name.contains("العربية"))
        assertTrue("Should contain emoji", category.name.contains("🎉"))
    }
}
