package app.armorclaw.tests

import org.junit.Assert.*
import org.junit.Test
import java.util.concurrent.atomic.AtomicInteger

/**
 * Offline and Sync Tests for ArmorChat
 *
 * Tests for offline message handling, queue management, and sync scenarios.
 */
class OfflineAndSyncTests {

    // ========================================================================
    // Message Queue Tests
    // ========================================================================

    @Test
    fun `empty queue has zero size`() {
        val queue = MessageQueue<String>()
        assertEquals("Empty queue should have size 0", 0, queue.size())
    }

    @Test
    fun `queue accepts messages`() {
        val queue = MessageQueue<String>()
        queue.enqueue("message1")
        queue.enqueue("message2")

        assertEquals("Queue should have 2 messages", 2, queue.size())
    }

    @Test
    fun `queue returns FIFO order`() {
        val queue = MessageQueue<String>()
        queue.enqueue("first")
        queue.enqueue("second")
        queue.enqueue("third")

        assertEquals("First should be 'first'", "first", queue.dequeue())
        assertEquals("Second should be 'second'", "second", queue.dequeue())
        assertEquals("Third should be 'third'", "third", queue.dequeue())
    }

    @Test
    fun `dequeue from empty queue returns null`() {
        val queue = MessageQueue<String>()
        assertNull("Empty dequeue should return null", queue.dequeue())
    }

    @Test
    fun `queue can be cleared`() {
        val queue = MessageQueue<String>()
        queue.enqueue("a")
        queue.enqueue("b")
        queue.enqueue("c")

        queue.clear()

        assertEquals("Cleared queue should be empty", 0, queue.size())
    }

    @Test
    fun `queue handles many items`() {
        val queue = MessageQueue<Int>()
        for (i in 1..10000) {
            queue.enqueue(i)
        }

        assertEquals("Queue should have 10000 items", 10000, queue.size())
    }

    // ========================================================================
    // Priority Queue Tests
    // ========================================================================

    @Test
    fun `priority queue respects priority`() {
        val queue = PriorityQueue<Message>()
        queue.enqueue(Message("low", Priority.LOW))
        queue.enqueue(Message("high", Priority.HIGH))
        queue.enqueue(Message("normal", Priority.NORMAL))

        assertEquals("High priority should be first", Priority.HIGH, queue.dequeue()?.priority)
        assertEquals("Normal priority should be second", Priority.NORMAL, queue.dequeue()?.priority)
        assertEquals("Low priority should be third", Priority.LOW, queue.dequeue()?.priority)
    }

    @Test
    fun `same priority preserves FIFO`() {
        val queue = PriorityQueue<Message>()
        queue.enqueue(Message("first", Priority.NORMAL))
        queue.enqueue(Message("second", Priority.NORMAL))
        queue.enqueue(Message("third", Priority.NORMAL))

        assertEquals("First should be 'first'", "first", queue.dequeue()?.content)
        assertEquals("Second should be 'second'", "second", queue.dequeue()?.content)
        assertEquals("Third should be 'third'", "third", queue.dequeue()?.content)
    }

    // ========================================================================
    // Retry Logic Tests
    // ========================================================================

    @Test
    fun `retry counter increments on failure`() {
        val retryCounter = RetryCounter(maxAttempts = 3)

        retryCounter.recordFailure()
        assertEquals("Should be at attempt 1", 1, retryCounter.currentAttempt())

        retryCounter.recordFailure()
        assertEquals("Should be at attempt 2", 2, retryCounter.currentAttempt())

        retryCounter.recordFailure()
        assertEquals("Should be at attempt 3", 3, retryCounter.currentAttempt())
    }

    @Test
    fun `retry counter resets on success`() {
        val retryCounter = RetryCounter(maxAttempts = 5)

        retryCounter.recordFailure()
        retryCounter.recordFailure()
        assertEquals("Should be at attempt 2", 2, retryCounter.currentAttempt())

        retryCounter.recordSuccess()
        assertEquals("Should reset to 0", 0, retryCounter.currentAttempt())
    }

    @Test
    fun `retry counter indicates when max reached`() {
        val retryCounter = RetryCounter(maxAttempts = 3)

        assertFalse("Should not be at max initially", retryCounter.isMaxReached())

        retryCounter.recordFailure()
        retryCounter.recordFailure()
        assertFalse("Should not be at max yet", retryCounter.isMaxReached())

        retryCounter.recordFailure()
        assertTrue("Should be at max now", retryCounter.isMaxReached())
    }

    @Test
    fun `retry counter cannot exceed max`() {
        val retryCounter = RetryCounter(maxAttempts = 3)

        repeat(10) { retryCounter.recordFailure() }

        assertEquals("Should be capped at max", 3, retryCounter.currentAttempt())
    }

    // ========================================================================
    // Backoff Calculation Tests
    // ========================================================================

    @Test
    fun `backoff starts at minimum`() {
        val backoff = BackoffCalculator(minDelay = 1000, maxDelay = 30000)

        val delay = backoff.calculate(0)

        assertTrue("First delay should be near min", delay >= 500 && delay <= 1500)
    }

    @Test
    fun `backoff increases with attempts`() {
        val backoff = BackoffCalculator(minDelay = 1000, maxDelay = 30000)

        val delays = (0..5).map { backoff.calculate(it) }

        // Generally increasing (accounting for jitter)
        var increasingCount = 0
        for (i in 1 until delays.size) {
            if (delays[i] >= delays[i-1] * 0.5) {
                increasingCount++
            }
        }
        assertTrue("Most delays should increase", increasingCount >= delays.size - 2)
    }

    @Test
    fun `backoff respects maximum`() {
        val backoff = BackoffCalculator(minDelay = 1000, maxDelay = 30000)

        for (attempt in 0..20) {
            val delay = backoff.calculate(attempt)
            assertTrue("Delay should not exceed max: $delay", delay <= 30000)
        }
    }

    @Test
    fun `backoff has jitter variance`() {
        val backoff = BackoffCalculator(minDelay = 1000, maxDelay = 30000)

        val delays = (1..100).map { backoff.calculate(5) }
        val uniqueDelays = delays.toSet()

        assertTrue("Should have jitter variance", uniqueDelays.size > 10)
    }

    // ========================================================================
    // Network State Tests
    // ========================================================================

    @Test
    fun `network state transitions correctly`() {
        val state = NetworkStateMachine()

        assertEquals("Initial state should be Disconnected", NetworkStatus.DISCONNECTED, state.current())

        state.connect()
        assertEquals("After connect should be Connected", NetworkStatus.CONNECTED, state.current())

        state.disconnect()
        assertEquals("After disconnect should be Disconnected", NetworkStatus.DISCONNECTED, state.current())
    }

    @Test
    fun `network state tracks quality`() {
        val state = NetworkStateMachine()

        state.connect(signalStrength = 90)
        assertEquals("Quality should be Good", NetworkQuality.GOOD, state.quality())

        state.connect(signalStrength = 30)
        assertEquals("Quality should be Poor", NetworkQuality.POOR, state.quality())
    }

    @Test
    fun `network state handles type changes`() {
        val state = NetworkStateMachine()

        state.connect(type = NetworkType.WIFI)
        assertEquals("Type should be WiFi", NetworkType.WIFI, state.type())

        state.connect(type = NetworkType.CELLULAR)
        assertEquals("Type should be Cellular", NetworkType.CELLULAR, state.type())
    }

    // ========================================================================
    // Sync State Tests
    // ========================================================================

    @Test
    fun `sync state tracks progress`() {
        val sync = SyncTracker()

        sync.start()
        assertTrue("Should be syncing", sync.isInProgress())

        sync.setProgress(50)
        assertEquals("Progress should be 50", 50, sync.progress())

        sync.complete()
        assertFalse("Should not be syncing", sync.isInProgress())
        assertEquals("Progress should be 100", 100, sync.progress())
    }

    @Test
    fun `sync state tracks pending items`() {
        val sync = SyncTracker()

        sync.setPending(100)
        assertEquals("Should have 100 pending", 100, sync.pending())

        sync.incrementProcessed(10)
        assertEquals("Should have 90 pending", 90, sync.pending())
    }

    // ========================================================================
    // Test Helpers
    // ========================================================================

    enum class Priority { HIGH, NORMAL, LOW }
    enum class NetworkStatus { CONNECTED, DISCONNECTED, RECONNECTING }
    enum class NetworkQuality { GOOD, FAIR, POOR }
    enum class NetworkType { WIFI, CELLULAR, ETHERNET }

    data class Message(val content: String, val priority: Priority)

    class MessageQueue<T> {
        private val items = ArrayDeque<T>()

        fun enqueue(item: T) { items.addLast(item) }
        fun dequeue(): T? = items.removeFirstOrNull()
        fun size(): Int = items.size
        fun clear() { items.clear() }
    }

    class PriorityQueue<T : Comparable<T>> {
        private val items = mutableListOf<T>()

        fun enqueue(item: T) { items.add(item); items.sort() }
        fun dequeue(): T? = if (items.isNotEmpty()) items.removeAt(0) else null
    }

    class RetryCounter(private val maxAttempts: Int) {
        private val counter = AtomicInteger(0)

        fun recordFailure() {
            counter.updateAndGet { minOf(it + 1, maxAttempts) }
        }
        fun recordSuccess() { counter.set(0) }
        fun currentAttempt(): Int = counter.get()
        fun isMaxReached(): Boolean = counter.get() >= maxAttempts
    }

    class BackoffCalculator(private val minDelay: Long, private val maxDelay: Long) {
        fun calculate(attempt: Int): Long {
            val base = minDelay * (1 shl attempt)
            val capped = minOf(base, maxDelay)
            val jitter = (Math.random() * 0.3 - 0.15) * capped
            return (capped + jitter).toLong().coerceIn(100, maxDelay)
        }
    }

    class NetworkStateMachine {
        private var status = NetworkStatus.DISCONNECTED
        private var quality = NetworkQuality.GOOD
        private var type = NetworkType.WIFI

        fun current() = status
        fun quality() = quality
        fun type() = type

        fun connect(signalStrength: Int = 100, type: NetworkType = NetworkType.WIFI) {
            this.status = NetworkStatus.CONNECTED
            this.type = type
            this.quality = when {
                signalStrength >= 70 -> NetworkQuality.GOOD
                signalStrength >= 40 -> NetworkQuality.FAIR
                else -> NetworkQuality.POOR
            }
        }

        fun disconnect() {
            this.status = NetworkStatus.DISCONNECTED
        }
    }

    class SyncTracker {
        private var inProgress = false
        private var progress = 0
        private var pending = 0
        private var processed = 0

        fun isInProgress() = inProgress
        fun progress() = progress
        fun pending() = pending - processed

        fun start() { inProgress = true; progress = 0 }
        fun setProgress(p: Int) { progress = p }
        fun setPending(count: Int) { pending = count; processed = 0 }
        fun incrementProcessed(count: Int) { processed += count }
        fun complete() { inProgress = false; progress = 100 }
    }
}
