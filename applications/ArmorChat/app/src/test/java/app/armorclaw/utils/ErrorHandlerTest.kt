package app.armorclaw.utils

import org.junit.Assert.*
import org.junit.Test

/**
 * Unit tests for ErrorHandler
 *
 * Tests:
 * - Error mapping
 * - Error code assignment
 * - Recoverability detection
 */
class ErrorHandlerTest {

    // ========================================================================
    // Network Error Mapping Tests
    // ========================================================================

    @Test
    fun `maps UnknownHostException to NETWORK_UNAVAILABLE`() {
        val error = java.net.UnknownHostException("Unable to resolve host")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to NETWORK_UNAVAILABLE", ErrorCode.NETWORK_UNAVAILABLE, mapped.code)
        assertTrue("Should be recoverable", mapped.recoverable)
        assertNotNull("Should have message", mapped.message)
    }

    @Test
    fun `maps ConnectException to CONNECTION_REFUSED`() {
        val error = java.net.ConnectException("Connection refused")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to CONNECTION_REFUSED", ErrorCode.CONNECTION_REFUSED, mapped.code)
        assertTrue("Should be recoverable", mapped.recoverable)
        assertTrue("Should have retry delay", mapped.retryAfterMs > 0)
    }

    @Test
    fun `maps SocketTimeoutException to TIMEOUT`() {
        val error = java.net.SocketTimeoutException("Read timed out")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to TIMEOUT", ErrorCode.TIMEOUT, mapped.code)
        assertTrue("Should be recoverable", mapped.recoverable)
    }

    // ========================================================================
    // Authentication Error Mapping Tests
    // ========================================================================

    @Test
    fun `maps 401 error to AUTH_REQUIRED`() {
        val error = RuntimeException("HTTP 401: Unauthorized")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to AUTH_REQUIRED", ErrorCode.AUTH_REQUIRED, mapped.code)
        assertFalse("Should not be recoverable", mapped.recoverable)
    }

    @Test
    fun `maps 403 error to PERMISSION_DENIED`() {
        val error = RuntimeException("HTTP 403: Forbidden")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to PERMISSION_DENIED", ErrorCode.PERMISSION_DENIED, mapped.code)
        assertFalse("Should not be recoverable", mapped.recoverable)
    }

    @Test
    fun `maps session expired message to SESSION_EXPIRED`() {
        val error = RuntimeException("Your session has expired")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to SESSION_EXPIRED", ErrorCode.SESSION_EXPIRED, mapped.code)
        assertFalse("Should not be recoverable", mapped.recoverable)
    }

    // ========================================================================
    // Setup Error Mapping Tests
    // ========================================================================

    @Test
    fun `maps already claimed to ALREADY_CLAIMED`() {
        val error = RuntimeException("Device already claimed")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to ALREADY_CLAIMED", ErrorCode.ALREADY_CLAIMED, mapped.code)
        assertFalse("Should not be recoverable", mapped.recoverable)
    }

    @Test
    fun `maps invalid passphrase to INVALID_PASSPHRASE`() {
        val error = RuntimeException("Invalid passphrase provided")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to INVALID_PASSPHRASE", ErrorCode.INVALID_PASSPHRASE, mapped.code)
        assertTrue("Should be recoverable", mapped.recoverable)
    }

    @Test
    fun `maps not trusted to DEVICE_NOT_TRUSTED`() {
        val error = RuntimeException("Device is not trusted")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to DEVICE_NOT_TRUSTED", ErrorCode.DEVICE_NOT_TRUSTED, mapped.code)
        assertFalse("Should not be recoverable", mapped.recoverable)
    }

    // ========================================================================
    // Server Error Mapping Tests
    // ========================================================================

    @Test
    fun `maps 429 error to RATE_LIMITED`() {
        val error = RuntimeException("HTTP 429: Rate limit exceeded")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to RATE_LIMITED", ErrorCode.RATE_LIMITED, mapped.code)
        assertTrue("Should be recoverable", mapped.recoverable)
        assertTrue("Should have retry delay", mapped.retryAfterMs >= 10000)
    }

    @Test
    fun `maps 500 error to INTERNAL_ERROR`() {
        val error = RuntimeException("HTTP 500: Internal server error")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to INTERNAL_ERROR", ErrorCode.INTERNAL_ERROR, mapped.code)
        assertTrue("Should be recoverable", mapped.recoverable)
    }

    // ========================================================================
    // Unknown Error Tests
    // ========================================================================

    @Test
    fun `maps unknown error to UNKNOWN`() {
        val error = RuntimeException("Some random error")
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to UNKNOWN", ErrorCode.UNKNOWN, mapped.code)
        assertTrue("Should be recoverable by default", mapped.recoverable)
    }

    @Test
    fun `maps null message error to UNKNOWN`() {
        val error = object : Throwable() {
            override val message: String? = null
        }
        val mapped = ErrorHandler.mapError(error)

        assertEquals("Should map to UNKNOWN", ErrorCode.UNKNOWN, mapped.code)
    }

    // ========================================================================
    // BridgeError Property Tests
    // ========================================================================

    @Test
    fun `BridgeError contains all required properties`() {
        val error = BridgeError(
            code = ErrorCode.CONNECTION_REFUSED,
            title = "Connection Failed",
            message = "Could not connect to server",
            recoverable = true,
            retryAfterMs = 5000
        )

        assertEquals("Code should match", ErrorCode.CONNECTION_REFUSED, error.code)
        assertEquals("Title should match", "Connection Failed", error.title)
        assertEquals("Message should match", "Could not connect to server", error.message)
        assertTrue("Should be recoverable", error.recoverable)
        assertEquals("Retry delay should match", 5000L, error.retryAfterMs)
    }

    @Test
    fun `BridgeError is Exception subclass`() {
        val error = BridgeError(
            code = ErrorCode.UNKNOWN,
            title = "Test",
            message = "Test error",
            recoverable = true
        )

        assertTrue("BridgeError should be Exception", error is Exception)
    }

    // ========================================================================
    // Suggested Action Tests
    // ========================================================================

    @Test
    fun `getSuggestedAction returns action for network unavailable`() {
        val error = BridgeError(
            code = ErrorCode.NETWORK_UNAVAILABLE,
            title = "Network Unavailable",
            message = "Test",
            recoverable = true
        )

        val action = ErrorHandler.getSuggestedAction(error)
        assertNotNull("Should have suggested action", action)
        assertTrue("Action should mention WiFi or data", action!!.contains("WiFi") || action.contains("data"))
    }

    @Test
    fun `getSuggestedAction returns action for connection refused`() {
        val error = BridgeError(
            code = ErrorCode.CONNECTION_REFUSED,
            title = "Connection Failed",
            message = "Test",
            recoverable = true
        )

        val action = ErrorHandler.getSuggestedAction(error)
        assertNotNull("Should have suggested action", action)
        assertTrue("Action should mention bridge", action!!.contains("bridge", ignoreCase = true))
    }

    @Test
    fun `getSuggestedAction returns null for unknown errors`() {
        val error = BridgeError(
            code = ErrorCode.UNKNOWN,
            title = "Unknown",
            message = "Test",
            recoverable = true
        )

        val action = ErrorHandler.getSuggestedAction(error)
        assertNull("Should not have suggested action for unknown", action)
    }

    // ========================================================================
    // ErrorCode Enum Tests
    // ========================================================================

    @Test
    fun `ErrorCode contains all expected values`() {
        val expectedCodes = listOf(
            ErrorCode.NETWORK_UNAVAILABLE,
            ErrorCode.CONNECTION_REFUSED,
            ErrorCode.TIMEOUT,
            ErrorCode.BRIDGE_NOT_FOUND,
            ErrorCode.BRIDGE_UNAVAILABLE,
            ErrorCode.AUTH_REQUIRED,
            ErrorCode.AUTH_INVALID,
            ErrorCode.SESSION_EXPIRED,
            ErrorCode.PERMISSION_DENIED,
            ErrorCode.ALREADY_CLAIMED,
            ErrorCode.INVALID_PASSPHRASE,
            ErrorCode.DEVICE_NOT_TRUSTED,
            ErrorCode.INVALID_INPUT,
            ErrorCode.RATE_LIMITED,
            ErrorCode.INTERNAL_ERROR,
            ErrorCode.UNKNOWN
        )

        assertEquals("Should have all expected error codes", expectedCodes.size, ErrorCode.entries.size)
    }
}
