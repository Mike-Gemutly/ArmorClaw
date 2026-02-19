package app.armorclaw.network

import org.junit.Assert.*
import org.junit.Test
import kotlin.math.pow

/**
 * Unit tests for NetworkResilience
 *
 * Tests:
 * - Exponential backoff calculation
 * - Jitter variance
 * - Backoff bounds
 */
class NetworkResilienceTest {

    // ========================================================================
    // Backoff Calculation Tests
    // ========================================================================

    @Test
    fun `calculateBackoff returns initial delay for first attempt`() {
        val resilience = createTestResilience()
        val delay = resilience.calculateBackoff(0)

        // With jitter, delay should be around INITIAL_BACKOFF_MS
        assertTrue(
            "Delay should be close to initial delay",
            delay in 500..2000L // 1000ms ± 50% for jitter
        )
    }

    @Test
    fun `calculateBackoff increases exponentially`() {
        val resilience = createTestResilience()

        val delays = (0..5).map { resilience.calculateBackoff(it) }

        // Each delay should generally be larger than the previous
        // (accounting for jitter variance)
        var increasingCount = 0
        for (i in 1 until delays.size) {
            if (delays[i] >= delays[i - 1] * 0.5) { // Allow for jitter variance
                increasingCount++
            }
        }
        assertTrue(
            "Most delays should increase with attempt count",
            increasingCount >= delays.size - 2
        )
    }

    @Test
    fun `calculateBackoff respects maximum delay`() {
        val resilience = createTestResilience()

        // High attempt count should hit max
        for (attempt in 10..20) {
            val delay = resilience.calculateBackoff(attempt)
            assertTrue(
                "Delay should not exceed max: got $delay",
                delay <= NetworkResilience.MAX_BACKOFF_MS
            )
        }
    }

    @Test
    fun `calculateBackoff adds jitter to prevent thundering herd`() {
        val resilience = createTestResilience()

        // Generate many delays for same attempt
        val delays = (1..100).map { resilience.calculateBackoff(3) }

        // Should have variance due to jitter
        val uniqueDelays = delays.toSet()
        assertTrue(
            "Multiple calls should produce varied delays due to jitter",
            uniqueDelays.size > 10
        )
    }

    @Test
    fun `calculateBackoff minimum is respected`() {
        val resilience = createTestResilience()

        for (attempt in 0..10) {
            val delay = resilience.calculateBackoff(attempt)
            assertTrue(
                "Delay should be at least 100ms",
                delay >= 100
            )
        }
    }

    @Test
    fun `calculateBackoff never exceeds 30 seconds`() {
        val resilience = createTestResilience()

        for (attempt in 0..100) {
            val delay = resilience.calculateBackoff(attempt)
            assertTrue(
                "Delay should never exceed 30 seconds: got ${delay}ms for attempt $attempt",
                delay <= 30000
            )
        }
    }

    // ========================================================================
    // BackoffConfig Tests
    // ========================================================================

    @Test
    fun `BackoffConfig default values are sensible`() {
        val config = BackoffConfig()

        assertEquals("Initial delay should be 1 second", 1000L, config.initialDelayMs)
        assertEquals("Max delay should be 30 seconds", 30000L, config.maxDelayMs)
        assertEquals("Multiplier should be 2", 2.0, config.multiplier, 0.01)
        assertTrue("Jitter should be reasonable", config.jitterFactor in 0.0..0.5)
    }

    @Test
    fun `BackoffConfig calculateDelay follows exponential pattern`() {
        val config = BackoffConfig(
            initialDelayMs = 1000,
            maxDelayMs = 30000,
            multiplier = 2.0,
            jitterFactor = 0.0 // No jitter for predictable test
        )

        // Without jitter, delays should follow 2^n pattern
        val expectedDelays = listOf(1000L, 2000L, 4000L, 8000L, 16000L, 30000L)

        for ((attempt, expected) in expectedDelays.withIndex()) {
            val actual = config.calculateDelay(attempt)
            assertEquals(
                "Attempt $attempt should have delay ~$expected",
                expected,
                actual
            )
        }
    }

    @Test
    fun `BackoffConfig calculateDelay with jitter varies`() {
        val config = BackoffConfig(
            initialDelayMs = 1000,
            maxDelayMs = 30000,
            multiplier = 2.0,
            jitterFactor = 0.5
        )

        val delays = (1..50).map { config.calculateDelay(5) }
        val uniqueDelays = delays.toSet()

        assertTrue(
            "With jitter, delays should vary significantly",
            uniqueDelays.size > 20
        )
    }

    // ========================================================================
    // ConnectionState Tests
    // ========================================================================

    @Test
    fun `ConnectionState types are correctly structured`() {
        // Test that all state types can be created
        val states = listOf<ConnectionState>(
            ConnectionState.Disconnected,
            ConnectionState.Connected,
            ConnectionState.Failed(Exception("test")),
            ConnectionState.Retrying(1, 1000)
        )

        assertEquals("Should have all expected state types", 4, states.size)
    }

    // ========================================================================
    // Error Retryability Tests
    // ========================================================================

    @Test
    fun `isRetryable identifies network errors as retryable`() {
        val retryableErrors = listOf(
            java.net.SocketTimeoutException("timeout"),
            java.net.ConnectException("connection refused"),
            java.net.UnknownHostException("unknown host"),
            java.io.IOException("I/O error")
        )

        for (error in retryableErrors) {
            assertTrue(
                "${error::class.simpleName} should be retryable",
                error.isRetryable()
            )
        }
    }

    @Test
    fun `isRetryable identifies non-network errors as non-retryable`() {
        val nonRetryableErrors = listOf<Throwable>(
            IllegalArgumentException("bad argument"),
            NullPointerException("null"),
            RuntimeException("generic error")
        )

        for (error in nonRetryableErrors) {
            assertFalse(
                "${error::class.simpleName} should not be retryable",
                error.isRetryable()
            )
        }
    }

    // ========================================================================
    // WebSocketState Tests
    // ========================================================================

    @Test
    fun `WebSocketState isConnected property works correctly`() {
        assertTrue("Connected state should be connected", WebSocketState.Connected.isConnected)
        assertFalse("Disconnected state should not be connected", WebSocketState.Disconnected.isConnected)
        assertFalse("Connecting state should not be connected", WebSocketState.Connecting().isConnected)
    }

    @Test
    fun `WebSocketState isConnecting property works correctly`() {
        assertTrue("Connecting should be connecting", WebSocketState.Connecting().isConnecting)
        assertTrue("Reconnecting should be connecting", WebSocketState.Reconnecting(1, 1000).isConnecting)
        assertFalse("Connected should not be connecting", WebSocketState.Connected.isConnecting)
        assertFalse("Disconnected should not be connecting", WebSocketState.Disconnected.isConnecting)
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    private fun createTestResilience(): NetworkResilience {
        // Create with mock ConnectivityManager
        // In real tests, you'd use Robolectric or instrumented tests
        val mockManager = org.mockito.Mockito.mock(android.net.ConnectivityManager::class.java)
        return NetworkResilience(mockManager)
    }
}
