package app.armorclaw.crypto

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.security.KeyStore
import java.security.SecureRandom
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.Cipher
import javax.crypto.spec.GCMParameterSpec

/**
 * Cryptographic Service for ArmorChat
 *
 * IMPORTANT: This service provides the foundation for E2EE but the actual
 * Matrix protocol encryption (Olm/Megolm) must use libolm or vodozemac.
 *
 * Matrix E2EE Requirements:
 * - Olm: X3DH + Double Ratchet for 1:1 sessions
 * - Megolm: Group message encryption with ratchet
 * - Ed25519: Signing keys for device verification
 * - Curve25519: Key exchange for session establishment
 *
 * This implementation provides:
 * 1. AndroidKeyStore-backed key storage
 * 2. AES-256-GCM for local data encryption
 * 3. Secure random generation
 * 4. Key isolation guarantees
 */
class CryptoService {

    companion object {
        private const val TAG = "CryptoService"
        private const val ANDROID_KEYSTORE = "AndroidKeyStore"
        private const val KEY_ALIAS_PREFIX = "armorclaw_"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val TAG_LENGTH = 128
        private const val IV_LENGTH = 12

        // Key aliases
        const val KEY_ALIAS_MASTER = "${KEY_ALIAS_PREFIX}master"
        const val KEY_ALIAS_SESSION = "${KEY_ALIAS_PREFIX}session"
        const val KEY_ALIAS_MEGOLM = "${KEY_ALIAS_PREFIX}megolm"
    }

    private val keyStore: KeyStore by lazy {
        KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }
    }

    /**
     * Initialize the cryptographic service
     * Generates master key if not exists
     */
    suspend fun initialize(): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            if (!keyStore.containsAlias(KEY_ALIAS_MASTER)) {
                generateMasterKey()
                Log.i(TAG, "Master key generated successfully")
            }
            Result.success(Unit)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to initialize crypto service", e)
            Result.failure(e)
        }
    }

    /**
     * Generate a new AES-256 key in AndroidKeyStore
     */
    private fun generateMasterKey() {
        val keyGenerator = KeyGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_AES,
            ANDROID_KEYSTORE
        )

        val spec = KeyGenParameterSpec.Builder(
            KEY_ALIAS_MASTER,
            KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
        )
            .setKeySize(256)
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setRandomizedEncryptionRequired(true)
            .setUserAuthenticationRequired(false)
            .setKeyValidityStart(java.util.Date())
            // Key is valid for 10 years
            .setKeyValidityEnd(java.util.Date(System.currentTimeMillis() + 315360000000L))
            .build()

        keyGenerator.init(spec)
        keyGenerator.generateKey()
    }

    /**
     * Encrypt data using AES-256-GCM
     * Returns: Base64(IV || Ciphertext || Tag)
     */
    suspend fun encrypt(plaintext: ByteArray, keyAlias: String = KEY_ALIAS_MASTER): Result<ByteArray> =
        withContext(Dispatchers.IO) {
            try {
                val secretKey = keyStore.getKey(keyAlias, null) as SecretKey
                val cipher = Cipher.getInstance(TRANSFORMATION)
                cipher.init(Cipher.ENCRYPT_MODE, secretKey)

                val ciphertext = cipher.doFinal(plaintext)
                val iv = cipher.iv

                // Prepend IV to ciphertext
                Result.success(iv + ciphertext)
            } catch (e: Exception) {
                Log.e(TAG, "Encryption failed", e)
                Result.failure(e)
            }
        }

    /**
     * Decrypt data encrypted with AES-256-GCM
     * Input: Base64(IV || Ciphertext || Tag)
     */
    suspend fun decrypt(ciphertext: ByteArray, keyAlias: String = KEY_ALIAS_MASTER): Result<ByteArray> =
        withContext(Dispatchers.IO) {
            try {
                val secretKey = keyStore.getKey(keyAlias, null) as SecretKey

                // Extract IV (first 12 bytes)
                val iv = ciphertext.sliceArray(0 until IV_LENGTH)
                val encryptedData = ciphertext.sliceArray(IV_LENGTH until ciphertext.size)

                val cipher = Cipher.getInstance(TRANSFORMATION)
                val spec = GCMParameterSpec(TAG_LENGTH, iv)
                cipher.init(Cipher.DECRYPT_MODE, secretKey, spec)

                Result.success(cipher.doFinal(encryptedData))
            } catch (e: Exception) {
                Log.e(TAG, "Decryption failed", e)
                Result.failure(e)
            }
        }

    /**
     * Generate cryptographically secure random bytes
     */
    fun generateRandomBytes(length: Int): ByteArray {
        val secureRandom = SecureRandom()
        val bytes = ByteArray(length)
        secureRandom.nextBytes(bytes)
        return bytes
    }

    /**
     * Generate a secure random string (hex encoded)
     */
    fun generateRandomString(length: Int): String {
        return generateRandomBytes(length).joinToString("") { "%02x".format(it) }
    }

    /**
     * Check if a key exists in the keystore
     */
    fun hasKey(keyAlias: String): Boolean {
        return try {
            keyStore.containsAlias(keyAlias)
        } catch (e: Exception) {
            false
        }
    }

    /**
     * Delete a key from the keystore
     */
    fun deleteKey(keyAlias: String): Result<Unit> {
        return try {
            keyStore.deleteEntry(keyAlias)
            Result.success(Unit)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to delete key: $keyAlias", e)
            Result.failure(e)
        }
    }

    /**
     * Verify that the crypto service is properly initialized
     */
    fun isInitialized(): Boolean {
        return hasKey(KEY_ALIAS_MASTER)
    }

    /**
     * Get key information for debugging (does not expose key material)
     */
    fun getKeyInfo(keyAlias: String): KeyInfo? {
        return try {
            val entry = keyStore.getEntry(keyAlias, null) as? KeyStore.SecretKeyEntry
            if (entry != null) {
                KeyInfo(
                    alias = keyAlias,
                    exists = true,
                    algorithm = "AES",
                    keySize = 256
                )
            } else null
        } catch (e: Exception) {
            null
        }
    }

    data class KeyInfo(
        val alias: String,
        val exists: Boolean,
        val algorithm: String,
        val keySize: Int
    )
}
