package app.armorclaw.crypto

import android.util.Log

/**
 * Native JNI bindings for vodozemac (Matrix E2EE)
 *
 * This class provides the bridge between Kotlin and the Rust-based
 * vodozemac library for Matrix protocol encryption.
 *
 * IMPORTANT: This requires the native library to be loaded.
 * The library must be built for the target architecture (arm64-v8a, armeabi-v7a).
 */
object VodozemacNative {

    private const val TAG = "VodozemacNative"
    private const val LIBRARY_NAME = "vodozemac_android"

    private var isLoaded = false
    private var loadError: String? = null

    init {
        try {
            System.loadLibrary(LIBRARY_NAME)
            isLoaded = true
            Log.i(TAG, "Vodozemac native library loaded successfully")
        } catch (e: UnsatisfiedLinkError) {
            isLoaded = false
            loadError = e.message
            Log.w(TAG, "Vodozemac native library not available: ${e.message}")
            Log.w(TAG, "E2EE will not be available - falling back to compatibility mode")
        }
    }

    /**
     * Check if the native library is available
     */
    fun isAvailable(): Boolean = isLoaded

    /**
     * Get the load error if library failed to load
     */
    fun getLoadError(): String? = loadError

    // ========================================================================
    // Library Management
    // ========================================================================

    /**
     * Initialize the native library
     * @return true if initialization succeeded
     */
    @JvmStatic
    external fun initialize(): Boolean

    /**
     * Get the library version string
     */
    @JvmStatic
    external fun getVersion(): String?

    // ========================================================================
    // Key Generation
    // ========================================================================

    /**
     * Generate a Curve25519 key pair for identity/key exchange
     * @return ByteArray containing the key pair, or null on error
     */
    @JvmStatic
    external fun generateIdentityKeyPair(): ByteArray?

    /**
     * Generate an Ed25519 key pair for signing
     * @return ByteArray containing the key pair, or null on error
     */
    @JvmStatic
    external fun generateSigningKeyPair(): ByteArray?

    // ========================================================================
    // Signing
    // ========================================================================

    /**
     * Sign a message with Ed25519
     * @param privateKey The Ed25519 private key
     * @param message The message to sign
     * @return The signature bytes, or null on error
     */
    @JvmStatic
    external fun sign(privateKey: ByteArray, message: ByteArray): ByteArray?

    /**
     * Verify an Ed25519 signature
     * @param publicKey The Ed25519 public key
     * @param message The original message
     * @param signature The signature to verify
     * @return true if signature is valid
     */
    @JvmStatic
    external fun verify(publicKey: ByteArray, message: ByteArray, signature: ByteArray): Boolean

    // ========================================================================
    // Olm (1:1 Sessions)
    // ========================================================================

    /**
     * Create a new Olm account
     * @return Pointer to the account (as Long), or 0 on error
     */
    @JvmStatic
    external fun createOlmAccount(): Long

    /**
     * Get identity keys from an Olm account
     * @param accountPtr Pointer to the Olm account
     * @return JSON string with curve25519 and ed25519 keys
     */
    @JvmStatic
    external fun getIdentityKeys(accountPtr: Long): String?

    /**
     * Generate one-time keys for an Olm account
     * @param accountPtr Pointer to the Olm account
     * @param count Number of one-time keys to generate
     * @return JSON string with one-time keys
     */
    @JvmStatic
    external fun generateOneTimeKeys(accountPtr: Long, count: Int): String?

    /**
     * Create an outbound Olm session
     * @param accountPtr Pointer to the Olm account
     * @param theirIdentityKey Recipient's Curve25519 identity key
     * @param theirOneTimeKey Recipient's one-time key
     * @return Session pointer (as Long), or 0 on error
     */
    @JvmStatic
    external fun createOutboundSession(
        accountPtr: Long,
        theirIdentityKey: ByteArray,
        theirOneTimeKey: ByteArray
    ): Long

    /**
     * Encrypt a message with Olm
     * @param sessionPtr Pointer to the Olm session
     * @param plaintext The message to encrypt
     * @return Encrypted message bytes
     */
    @JvmStatic
    external fun encryptOlm(sessionPtr: Long, plaintext: ByteArray): ByteArray?

    /**
     * Decrypt a message with Olm
     * @param sessionPtr Pointer to the Olm session
     * @param ciphertext The encrypted message
     * @param messageType The message type (0 = pre-key, 1 = normal)
     * @return Decrypted message bytes
     */
    @JvmStatic
    external fun decryptOlm(sessionPtr: Long, ciphertext: ByteArray, messageType: Int): ByteArray?

    // ========================================================================
    // Megolm (Group Sessions)
    // ========================================================================

    /**
     * Create an outbound Megolm group session
     * @return Session pointer (as Long), or 0 on error
     */
    @JvmStatic
    external fun createOutboundMegolmSession(): Long

    /**
     * Get the session key for sharing with group members
     * @param sessionPtr Pointer to the Megolm session
     * @return Base64-encoded session key
     */
    @JvmStatic
    external fun getMegolmSessionKey(sessionPtr: Long): String?

    /**
     * Create an inbound Megolm session from a session key
     * @param sessionKey Base64-encoded session key
     * @return Session pointer (as Long), or 0 on error
     */
    @JvmStatic
    external fun createInboundMegolmSession(sessionKey: String): Long

    /**
     * Encrypt a message with Megolm
     * @param sessionPtr Pointer to the Megolm session
     * @param plaintext The message to encrypt
     * @return JSON string with encrypted message content
     */
    @JvmStatic
    external fun encryptMegolm(sessionPtr: Long, plaintext: ByteArray): String?

    /**
     * Decrypt a message with Megolm
     * @param sessionPtr Pointer to the Megolm session
     * @param ciphertext JSON string with encrypted message content
     * @return Decrypted message bytes
     */
    @JvmStatic
    external fun decryptMegolm(sessionPtr: Long, ciphertext: String): ByteArray?

    // ========================================================================
    // Cleanup
    // ========================================================================

    /**
     * Free an Olm account
     * @param accountPtr Pointer to the Olm account
     */
    @JvmStatic
    external fun freeOlmAccount(accountPtr: Long)

    /**
     * Free a Megolm session
     * @param sessionPtr Pointer to the Megolm session
     */
    @JvmStatic
    external fun freeMegolmSession(sessionPtr: Long)

    // ========================================================================
    // Convenience Methods
    // ========================================================================

    /**
     * Safe wrapper that checks library availability before calling
     */
    inline fun <T> withNative(block: VodozemacNative.() -> T?): T? {
        return if (isLoaded) {
            try {
                block()
            } catch (e: Exception) {
                Log.e(TAG, "Native operation failed", e)
                null
            }
        } else {
            Log.w(TAG, "Native library not available, operation skipped")
            null
        }
    }
}
