package app.armorclaw.crypto

import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

/**
 * Unit tests for CryptoService
 *
 * Tests:
 * - Initialization
 * - Encryption/Decryption round-trip
 * - Different IVs produce different ciphertext
 * - Invalid key fails decryption
 * - Random generation
 */
class CryptoServiceTest {

    private lateinit var cryptoService: CryptoService

    @Before
    fun setup() {
        cryptoService = CryptoService()
    }

    // ========================================================================
    // Initialization Tests
    // ========================================================================

    @Test
    fun `initialize generates master key`() = runBlocking {
        val result = cryptoService.initialize()

        assertTrue("Initialization should succeed", result.isSuccess)
        assertTrue("Should be initialized after init", cryptoService.isInitialized())
    }

    @Test
    fun `isInitialized returns false before initialization`() {
        assertFalse("Should not be initialized before init", cryptoService.isInitialized())
    }

    // ========================================================================
    // Encryption/Decryption Tests
    // ========================================================================

    @Test
    fun `encrypt then decrypt returns original plaintext`() = runBlocking {
        cryptoService.initialize()

        val plaintext = "Hello, Matrix!".toByteArray(Charsets.UTF_8)

        val encryptResult = cryptoService.encrypt(plaintext)
        assertTrue("Encryption should succeed", encryptResult.isSuccess)

        val ciphertext = encryptResult.getOrThrow()
        assertNotEquals("Ciphertext should differ from plaintext", plaintext.toList(), ciphertext.toList())

        val decryptResult = cryptoService.decrypt(ciphertext)
        assertTrue("Decryption should succeed", decryptResult.isSuccess)

        val decrypted = decryptResult.getOrThrow()
        assertArrayEquals("Decrypted should match original", plaintext, decrypted)
    }

    @Test
    fun `encrypt with different IVs produces different ciphertext`() = runBlocking {
        cryptoService.initialize()

        val plaintext = "Same message".toByteArray(Charsets.UTF_8)

        val ciphertext1 = cryptoService.encrypt(plaintext).getOrThrow()
        val ciphertext2 = cryptoService.encrypt(plaintext).getOrThrow()

        // IV is prepended, so ciphertext should be different
        assertNotEquals(
            "Same plaintext should produce different ciphertext (due to random IV)",
            ciphertext1.toList(),
            ciphertext2.toList()
        )
    }

    @Test
    fun `decrypt with corrupted ciphertext fails`() = runBlocking {
        cryptoService.initialize()

        val plaintext = "Test message".toByteArray(Charsets.UTF_8)
        val ciphertext = cryptoService.encrypt(plaintext).getOrThrow()

        // Corrupt the ciphertext (change a byte in the middle)
        val corrupted = ciphertext.copyOf()
        corrupted[corrupted.size / 2] = (corrupted[corrupted.size / 2] + 1).toByte()

        val decryptResult = cryptoService.decrypt(corrupted)

        assertTrue("Decryption of corrupted data should fail", decryptResult.isFailure)
    }

    @Test
    fun `decrypt with truncated IV fails`() = runBlocking {
        cryptoService.initialize()

        val plaintext = "Test message".toByteArray(Charsets.UTF_8)
        val ciphertext = cryptoService.encrypt(plaintext).getOrThrow()

        // Truncate to just 5 bytes (less than IV length of 12)
        val truncated = ciphertext.sliceArray(0 until 5)

        val decryptResult = cryptoService.decrypt(truncated)

        assertTrue("Decryption of truncated data should fail", decryptResult.isFailure)
    }

    @Test
    fun `encrypt empty array succeeds`() = runBlocking {
        cryptoService.initialize()

        val plaintext = ByteArray(0)

        val encryptResult = cryptoService.encrypt(plaintext)

        assertTrue("Encryption of empty array should succeed", encryptResult.isSuccess)
    }

    @Test
    fun `encrypt large data succeeds`() = runBlocking {
        cryptoService.initialize()

        // 1MB of data
        val plaintext = ByteArray(1024 * 1024) { (it % 256).toByte() }

        val encryptResult = cryptoService.encrypt(plaintext)

        assertTrue("Encryption of large data should succeed", encryptResult.isSuccess)

        val ciphertext = encryptResult.getOrThrow()
        val decryptResult = cryptoService.decrypt(ciphertext)

        assertTrue("Decryption of large data should succeed", decryptResult.isSuccess)
        assertArrayEquals("Large data should match after round-trip", plaintext, decryptResult.getOrThrow())
    }

    // ========================================================================
    // Random Generation Tests
    // ========================================================================

    @Test
    fun `generateRandomBytes produces correct length`() {
        for (length in listOf(16, 32, 64, 128)) {
            val bytes = cryptoService.generateRandomBytes(length)
            assertEquals("Random bytes should have requested length", length, bytes.size)
        }
    }

    @Test
    fun `generateRandomBytes produces different values`() {
        val bytes1 = cryptoService.generateRandomBytes(32)
        val bytes2 = cryptoService.generateRandomBytes(32)

        assertNotEquals("Random bytes should be different", bytes1.toList(), bytes2.toList())
    }

    @Test
    fun `generateRandomString produces correct length`() {
        // Each byte becomes 2 hex chars
        for (length in listOf(16, 32, 64)) {
            val string = cryptoService.generateRandomString(length)
            assertEquals("Random string should have 2x byte length", length * 2, string.length)
        }
    }

    @Test
    fun `generateRandomString is hex encoded`() {
        val string = cryptoService.generateRandomString(16)
        assertTrue("String should be hex encoded", string.all { it in '0'..'9' || it in 'a'..'f' })
    }

    // ========================================================================
    // Key Management Tests
    // ========================================================================

    @Test
    fun `hasKey returns false for non-existent key`() {
        val nonExistentAlias = "non_existent_key_${System.currentTimeMillis()}"
        assertFalse("Non-existent key should return false", cryptoService.hasKey(nonExistentAlias))
    }

    @Test
    fun `getKeyInfo returns null for non-existent key`() {
        val nonExistentAlias = "non_existent_key_${System.currentTimeMillis()}"
        assertNull("Non-existent key info should be null", cryptoService.getKeyInfo(nonExistentAlias))
    }
}
