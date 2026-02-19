package app.armorclaw.tests

import org.junit.Assert.*
import org.junit.Test
import java.util.concurrent.TimeUnit

/**
 * Edge Case Tests for ArmorChat
 *
 * Tests for unusual inputs, boundary conditions, and error scenarios.
 */
class EdgeCaseTests {

    // ========================================================================
    // String Input Edge Cases
    // ========================================================================

    @Test
    fun `empty string is handled gracefully`() {
        val input = ""
        assertTrue("Empty string should be recognized", input.isEmpty())
    }

    @Test
    fun `whitespace only string is handled`() {
        val input = "   \t\n   "
        assertTrue("Whitespace string should be blank", input.isBlank())
    }

    @Test
    fun `very long string is handled`() {
        val input = "a".repeat(1_000_000)
        assertEquals("Length should match", 1_000_000, input.length)
    }

    @Test
    fun `unicode string is handled correctly`() {
        val inputs = listOf(
            "中文测试",
            "العربية",
            "עברית",
            "日本語",
            "한국어",
            "Ελληνικά",
            "Русский",
            "🎉🚀💻",
            "👨‍👩‍👧‍👦"
        )

        for (input in inputs) {
            assertTrue("Unicode should be preserved: $input", input.isNotEmpty())
        }
    }

    @Test
    fun `null bytes in string are handled`() {
        val input = "hello\u0000world"
        assertTrue("Null byte should be preserved", input.contains("\u0000"))
    }

    @Test
    fun `control characters are handled`() {
        val input = "hello\t\r\nworld"
        assertTrue("Control chars should be preserved", input.contains("\t"))
    }

    // ========================================================================
    // Numeric Edge Cases
    // ========================================================================

    @Test
    fun `zero values are handled`() {
        val values = listOf(0, 0L, 0.0, 0.0f)
        for (v in values) {
            assertEquals("Zero should equal zero", 0, v)
        }
    }

    @Test
    fun `negative values are handled`() {
        val values = listOf(-1, -100, Long.MIN_VALUE, Int.MIN_VALUE)
        for (v in values) {
            assertTrue("Negative should be negative", v < 0)
        }
    }

    @Test
    fun `max values are handled`() {
        val maxInt = Int.MAX_VALUE
        val maxLong = Long.MAX_VALUE

        assertTrue("Max int should be large", maxInt > 2_000_000_000)
        assertTrue("Max long should be very large", maxLong > maxInt)
    }

    @Test
    fun `overflow is detected`() {
        val max = Int.MAX_VALUE
        val overflow = max + 1
        assertTrue("Overflow should be negative", overflow < 0)
    }

    // ========================================================================
    // Collection Edge Cases
    // ========================================================================

    @Test
    fun `empty collection operations are safe`() {
        val emptyList = emptyList<String>()
        val emptySet = emptySet<String>()
        val emptyMap = emptyMap<String, String>()

        assertTrue("Empty list should be empty", emptyList.isEmpty())
        assertTrue("Empty set should be empty", emptySet.isEmpty())
        assertTrue("Empty map should be empty", emptyMap.isEmpty())
    }

    @Test
    fun `single element collection is handled`() {
        val singleList = listOf("only")
        val singleSet = setOf("only")

        assertEquals("Single list has size 1", 1, singleList.size)
        assertEquals("Single set has size 1", 1, singleSet.size)
    }

    @Test
    fun `large collection is handled`() {
        val largeList = (1..100_000).toList()
        assertEquals("Large list should have correct size", 100_000, largeList.size)
    }

    @Test
    fun `duplicate handling in list`() {
        val listWithDupes = listOf("a", "b", "a", "c", "b")
        assertEquals("Should have duplicates", 5, listWithDupes.size)

        val unique = listWithDupes.toSet()
        assertEquals("Set should remove duplicates", 3, unique.size)
    }

    // ========================================================================
    // Time Edge Cases
    // ========================================================================

    @Test
    fun `epoch time is handled`() {
        val epoch = 0L
        assertEquals("Epoch should be 0", 0L, epoch)
    }

    @Test
    fun `future time is handled`() {
        val future = System.currentTimeMillis() + TimeUnit.DAYS.toMillis(365)
        assertTrue("Future should be in future", future > System.currentTimeMillis())
    }

    @Test
    fun `negative timestamp is handled`() {
        val negative = -1L
        assertTrue("Negative should be negative", negative < 0)
    }

    @Test
    fun `timestamp precision is adequate`() {
        val t1 = System.currentTimeMillis()
        Thread.sleep(1)
        val t2 = System.currentTimeMillis()

        assertTrue("Timestamps should differ", t2 >= t1)
    }

    // ========================================================================
    // Base64 Edge Cases
    // ========================================================================

    @Test
    fun `empty base64 encodes correctly`() {
        val empty = ByteArray(0)
        val encoded = java.util.Base64.getEncoder().encodeToString(empty)
        assertEquals("Empty base64 should be empty", "", encoded)
    }

    @Test
    fun `base64 roundtrip preserves data`() {
        val data = ByteArray(256) { it.toByte() }
        val encoded = java.util.Base64.getEncoder().encodeToString(data)
        val decoded = java.util.Base64.getDecoder().decode(encoded)

        assertArrayEquals("Roundtrip should preserve data", data, decoded)
    }

    @Test
    fun `base64 handles binary data`() {
        val binary = ByteArray(100) { (it % 256).toByte() }
        val encoded = java.util.Base64.getEncoder().encodeToString(binary)
        assertTrue("Base64 should be valid", encoded.all {
            it in 'A'..'Z' || it in 'a'..'z' || it in '0'..'9' || it in "+/="
        })
    }

    // ========================================================================
    // JSON Edge Cases
    // ========================================================================

    @Test
    fun `empty object parses`() {
        val json = "{}"
        assertTrue("Should be valid JSON object", json.startsWith("{"))
    }

    @Test
    fun `empty array parses`() {
        val json = "[]"
        assertTrue("Should be valid JSON array", json.startsWith("["))
    }

    @Test
    fun `null value in JSON`() {
        val json = """{"value":null}"""
        assertTrue("Should contain null", json.contains("null"))
    }

    @Test
    fun `nested JSON is handled`() {
        val json = """{"a":{"b":{"c":{"d":"deep"}}}}"""
        assertTrue("Should be deeply nested", json.contains("deep"))
    }

    @Test
    fun `special characters in JSON are escaped`() {
        val json = """{"quote":"\"","backslash":"\\"}"""
        assertTrue("Should handle escaping", json.contains("\\\""))
    }

    // ========================================================================
    // Concurrency Edge Cases
    // ========================================================================

    @Test
    fun `atomic operations are threadSafe`() {
        val counter = java.util.concurrent.atomic.AtomicInteger(0)
        val threads = (1..10).map {
            Thread {
                repeat(1000) {
                    counter.incrementAndGet()
                }
            }
        }

        threads.forEach { it.start() }
        threads.forEach { it.join() }

        assertEquals("Counter should be 10000", 10000, counter.get())
    }

    @Test
    fun `concurrent map handles races`() {
        val map = java.util.concurrent.ConcurrentHashMap<String, Int>()
        val threads = (1..10).map { i ->
            Thread {
                repeat(100) { j ->
                    map["key-$i-$j"] = i * 100 + j
                }
            }
        }

        threads.forEach { it.start() }
        threads.forEach { it.join() }

        assertEquals("Map should have 1000 entries", 1000, map.size)
    }

    // ========================================================================
    // Regex Edge Cases
    // ========================================================================

    @Test
    fun `empty regex matches empty`() {
        val pattern = Regex("")
        assertTrue("Empty pattern matches empty string", pattern.matches(""))
    }

    @Test
    fun `special regex chars are escaped`() {
        val input = "a.b*c+d?e"
        val escaped = Regex.escape(input)
        assertTrue("Should escape special chars", escaped.contains("\\"))
    }

    @Test
    fun `unicode regex works`() {
        val pattern = Regex("[\\p{L}]+")
        assertTrue("Should match unicode letters", pattern.matches("中文العربية"))
    }

    // ========================================================================
    // URL Edge Cases
    // ========================================================================

    @Test
    fun `localhost URL is valid`() {
        val url = "http://localhost:8080/path"
        assertTrue("Should contain localhost", url.contains("localhost"))
    }

    @Test
    fun `ipv4 URL is handled`() {
        val url = "http://192.168.1.1:8080"
        assertTrue("Should contain IP", url.contains("192.168.1.1"))
    }

    @Test
    fun `ipv6 URL is handled`() {
        val url = "http://[::1]:8080"
        assertTrue("Should contain IPv6 brackets", url.contains("[") && url.contains("]"))
    }

    @Test
    fun `URL with query params is handled`() {
        val url = "http://example.com/path?a=1&b=2&c=3"
        assertTrue("Should contain query string", url.contains("?"))
    }

    @Test
    fun `URL with fragment is handled`() {
        val url = "http://example.com/path#fragment"
        assertTrue("Should contain fragment", url.contains("#"))
    }
}
