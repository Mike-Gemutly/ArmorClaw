# ArmorClaw Mobile App - Security Implementation Specification

> **Document Purpose:** Complete security architecture for certificate pinning, biometric authentication, and secure clipboard
> **Date Created:** 2026-02-10
> **Phase:** 1 (Foundation)
> **Priority:** HIGH - Critical for production security compliance

---

## 1. Certificate Pinning

### 1.1 Architecture Overview

Certificate pinning ensures the mobile app only connects to trusted Matrix homeservers by validating server certificates against known fingerprints.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Certificate Pinning Flow                         │
│                                                                      │
│  App                                Matrix Homeserver                │
│   │                                    │                            │
│   │  1. HTTPS Request                  │                            │
│   ├─────────────────────────────────> │                            │
│   │                                    │                            │
│   │  2. Certificate Chain              │                            │
│   │<──────────────────────────────────┤                            │
│   │                                    │                            │
│   │  3. Extract Leaf Certificate       │                            │
│   │  ┌─────────────────────────────┐   │                            │
│   │  │ Extract Public Key Info      │   │                            │
│   │  │ Compute SHA-256 Hash         │   │                            │
│   │  └─────────────────────────────┘   │                            │
│   │                                    │                            │
│   │  4. Compare to Pinned Hashes       │                            │
│   │  ┌─────────────────────────────┐   │                            │
│   │  │ pins.contains(certHash)     │   │                            │
│   │  │                             │   │                            │
│   │  │ Match? ✓ Continue           │   │                            │
│   │  │ No Match? ✗ Abort           │   │                            │
│   │  └─────────────────────────────┘   │                            │
│   │                                    │                            │
│   │  5. Proceed or Reject              │                            │
│   ├─────────────────────────────────> │                            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Implementation

```kotlin
/**
 * Certificate Pinning Interceptor for OkHttp
 */
class CertificatePinningInterceptor(
    private val pins: List<CertificatePin>,
    private val logger: SecurityLogger
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()

        // Skip pinning for debug builds if configured
        if (BuildConfig.DEBUG && !BuildConfig.ENABLE_PINNING_IN_DEBUG) {
            logger.log(SecurityEvent.PinningSkipped("Debug build"))
            return chain.proceed(request)
        }

        // Get certificate from the connection
        val connection = chain.connection()
            ?: throw CertificatePinningException("No connection available")

        val certificate = getLeafCertificate(connection)
        val certHash = computeCertificateHash(certificate)

        // Check if certificate hash matches any pinned hash
        val matchedPin = pins.find { it.matches(certHash) }

        if (matchedPin == null) {
            val error = CertificatePinningException(
                "Certificate not pinned!\n" +
                "Expected: ${pins.joinToString(", ") { it.hash.substring(0, 16) } }\n" +
                "Received: ${certHash.substring(0, 16)}"
            )
            logger.log(SecurityEvent.PinningFailure(request.url.toString(), certHash))
            throw error
        }

        // Check expiry
        if (matchedPin.isExpired()) {
            logger.log(SecurityEvent.PinningExpired(matchedPin.hash))
            throw CertificatePinningException("Pinned certificate has expired!")
        }

        logger.log(SecurityEvent.PinningSuccess(request.url.toString()))
        return chain.proceed(request)
    }

    private fun getLeafCertificate(connection: Connection): X509Certificate {
        val tlsHandshake = connection.handshake() as? Handshake
            ?: throw CertificatePinningException("Not a TLS connection")

        val certificates = tlsHandshake.peerCertificates
        if (certificates.isEmpty()) {
            throw CertificatePinningException("No certificates in handshake")
        }

        return certificates.first() as X509Certificate
    }

    private fun computeCertificateHash(certificate: X509Certificate): String {
        // Compute SHA-256 hash of the Subject Public Key Info (SPKI)
        val publicKeyInfo = ByteArrayOutputStream().use { output ->
            certificate.publicKey.encoded.use { input ->
                input.copyTo(output)
            }
            output.toByteArray()
        }

        val digest = MessageDigest.getInstance("SHA-256")
        val hash = digest.digest(publicKeyInfo)

        return Base64.encodeToString(hash, Base64.NO_WRAP)
    }
}

/**
 * Certificate Pin Configuration
 */
data class CertificatePin(
    val hash: String,           // Base64-encoded SHA-256 of SPKI
    val expiry: Instant?,       // Optional expiry date
    val description: String     // Human-readable description
) {
    fun matches(certHash: String): Boolean {
        return hash.equals(certHash, ignoreCase = true)
    }

    fun isExpired(): Boolean {
        return expiry?.let { Clock.System.now() > it } ?: false
    }
}

/**
 * Certificate Pinning Exception
 */
class CertificatePinningException(message: String) : SecurityException(message)

/**
 * Predefined pins for known homeservers
 */
object KnownPins {
    // Matrix.org pin (example - replace with actual pins)
    val MATRIX_ORG = CertificatePin(
        hash = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",  // Replace
        expiry = null,
        description = "matrix.org"
    )

    // Example: Self-signed cert
    val SELF_SIGNED = CertificatePin(
        hash = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=",  // Replace
        expiry = Instant.parse("2027-01-01T00:00:00Z"),
        description = "Self-signed development cert"
    )

    fun all(): List<CertificatePin> = listOf(MATRIX_ORG, SELF_SIGNED)
}
```

### 1.3 Integration with Matrix Client

```kotlin
/**
 * Create OkHttp client with certificate pinning
 */
fun createPinnedOkHttpClient(
    pins: List<CertificatePin>,
    logger: SecurityLogger
): OkHttpClient {
    return OkHttpClient.Builder()
        .addInterceptor(CertificatePinningInterceptor(pins, logger))
        // Add additional security measures
        .connectionSpecs(list_of(createSecureConnectionSpec()))
        .build()
}

/**
 * Secure connection spec
 */
fun createSecureConnectionSpec(): ConnectionSpec {
    return ConnectionSpec.Builder(ConnectionSpec.MODERN_TLS)
        .tlsVersions(TlsVersion.TLS_1_2, TlsVersion.TLS_1_3)
        .cipherSuites(
            // TLS 1.3
            CipherSuite.TLS_AES_128_GCM_SHA256,
            CipherSuite.TLS_AES_256_GCM_SHA384,
            CipherSuite.TLS_CHACHA20_POLY1305_SHA256,
            // TLS 1.2
            CipherSuite.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            CipherSuite.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
            CipherSuite.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            CipherSuite.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            CipherSuite.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
            CipherSuite.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
        )
        .build()
}
```

### 1.4 Pin Extraction Tool

```kotlin
/**
 * Utility to extract certificate pins from a server
 * Run this once to get pins to include in the app
 */
object CertificatePinExtractor {

    suspend fun extractPins(host: String, port: Int = 443): List<CertificatePin> {
        return withContext(Dispatchers.IO) {
            val sslContext = SSLContext.getInstance("TLS")
            sslContext.init(null, null, null)

            val socketFactory = sslContext.socketFactory
            val socket = socketFactory.createSocket(host, port) as SSLSocket

            socket.use {
                it.startHandshake()

                val certificates = it.session.peerCertificates
                certificates.map { cert ->
                    val x509 = cert as X509Certificate
                    val hash = computeCertificateHash(x509)
                    CertificatePin(
                        hash = hash,
                        expiry = x509.notAfter.toInstant(),
                        description = "$host:${x509.subjectDN}"
                    )
                }
            }
        }
    }

    private fun computeCertificateHash(certificate: X509Certificate): String {
        val publicKeyInfo = certificate.publicKey.encoded
        val digest = MessageDigest.getInstance("SHA-256")
        val hash = digest.digest(publicKeyInfo)
        return Base64.encodeToString(hash, Base64.NO_WRAP)
    }

    /**
     * Generate pin configuration code
     */
    fun generatePinConfig(pins: List<CertificatePin>): String {
        return buildString {
            appendLine("object ServerPins {")
            pins.forEachIndexed { index, pin ->
                appendLine("    val PIN_$index = CertificatePin(")
                appendLine("        hash = \"${pin.hash}\",")
                appendLine("        expiry = ${pin.expiry?.let { "Instant.parse(\"${it}\")" } ?: "null"},")
                appendLine("        description = \"${pin.description}\"")
                appendLine("    )")
            }
            appendLine()
            appendLine("    fun all(): List<CertificatePin> = listOf(")
            pins.forEachIndexed { index, _ ->
                if (index > 0) append(", ")
                append("PIN_$index")
            }
            appendLine(")")
            appendLine("}")
        }
    }
}
```

### 1.5 Configuration Updates

```kotlin
/**
 * Remote certificate pin updates via Matrix
 */
class RemotePinManager(
    private val matrixClient: MatrixClient,
    private val configRoomId: String,
    private val signatureVerifier: SignatureVerifier
) {
    /**
     * Fetch updated pins from Matrix room state
     */
    suspend fun fetchUpdatedPins(): List<CertificatePin> {
        val stateEvents = matrixClient.getStateEvents(
            roomId = configRoomId,
            eventType = "m.room.pinned_certificates"
        )

        val latestState = stateEvents.maxByOrNull { it.originServerTs }
            ?: throw Exception("No certificate pin configuration found")

        // Verify signature
        if (!signatureVerifier.verify(latestState)) {
            throw SecurityException("Invalid signature on certificate pins!")
        }

        // Parse pins
        val content = latestState.content as? Map<*, *>
            ?: throw Exception("Invalid pin configuration")

        val pinsArray = content["pins"] as? List<*>
            ?: throw Exception("Missing pins array")

        return pinsArray.map { pinObj ->
            val pin = pinObj as Map<*, *>
            CertificatePin(
                hash = pin["hash"] as String,
                expiry = (pin["expiry"] as? String)?.let { Instant.parse(it) },
                description = pin["description"] as String
            )
        }
    }

    /**
     * Update local pin store
     */
    suspend fun updatePins(): Result<List<CertificatePin>> {
        return try {
            val newPins = fetchUpdatedPins()
            // Store in secure preferences
            SecurePreferences.storePins(newPins)
            Result.success(newPins)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
```

---

## 2. Biometric Authentication

### 2.1 Architecture Overview

Biometric authentication protects Matrix access tokens and sensitive operations.

```
┌─────────────────────────────────────────────────────────────────────┐
│                   Biometric Authentication Flow                      │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    App Launch                                │   │
│  │                                                              │   │
│  │  1. Check if token exists in secure storage                  │   │
│  │  ┌──────────────────────────────────────────────────────┐   │   │
│  │  │ Token exists?                                         │   │   │
│  │  │   Yes → Show Biometric Prompt                         │   │   │
│  │  │   No → Show Login Screen                              │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              Biometric Prompt                                │   │
│  │  ┌─────────────────────────────────────────────────────┐   │   │
│  │  │ "Authenticate to access chat"                        │   │   │
│  │  │                                                        │   │   │
│  │  │  [Face ID / Touch ID Sensor]                          │   │   │
│  │  │                                                        │   │   │
│  │  │           [Cancel]                                    │   │   │
│  │  └─────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              Authentication Result                           │   │
│  │  ┌─────────────────────────────────────────────────────┐   │   │
│  │  │ Success → Unlock token, proceed to app               │   │   │
│  │  │ Failed → Show error, offer retry or password         │   │   │
│  │  │ Error → Handle gracefully (e.g., no biometric)       │   │   │
│  │  └─────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Implementation

```kotlin
/**
 * Biometric Authentication Manager
 */
class BiometricTokenManager(
    private val context: Context,
    private val keyStore: KeyStore,
    private val cryptoManager: CryptoManager
) {

    companion object {
        private const val KEY_NAME = "armorclaw_biometric_key"
        private const val TOKEN_PREFS = "biometric_tokens"
        private const val TOKEN_CIPHER_PREFIX = "encrypted_token_"
    }

    private val executor = ContextCompat.getMainExecutor(context)
    private val biometricPrompt = BiometricPrompt(
        context as FragmentActivity,
        executor,
        object : BiometricPrompt.AuthenticationCallback() {
            override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                // Handled by caller
            }

            override fun onAuthenticationFailed() {
                // Handled by caller
            }

            override fun onError(error: Int, errString: CharSequence) {
                // Handled by caller
            }
        }
    )

    /**
     * Check if biometric authentication is available
     */
    fun isBiometricAvailable(): BiometricAvailability {
        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setAllowedAuthenticators(
                BiometricPrompt.Authenticators.BIOMETRIC_STRONG or
                BiometricPrompt.Authenticators.DEVICE_CREDENTIAL
            )
            .build()

        val canAuthenticate = BiometricManager.from(context)
            .canAuthenticate(promptInfo.allowedAuthenticators)

        return when (canAuthenticate) {
            BiometricManager.BIOMETRIC_SUCCESS ->
                BiometricAvailability.Available
            BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED ->
                BiometricAvailability.NoEnrollment
            BiometricManager.BIOMETRIC_ERROR_NO_HARDWARE ->
                BiometricAvailability.NoHardware
            BiometricManager.BIOMETRIC_ERROR_HW_UNAVAILABLE ->
                BiometricAvailability.HwUnavailable
            else ->
                BiometricAvailability.Unknown(canAuthenticate)
        }
    }

    /**
     * Store Matrix access token with biometric protection
     */
    suspend fun storeToken(
        token: String,
        userId: String,
        prompt: String = "Authenticate to save login"
    ): Result<Unit> {
        if (isBiometricAvailable() !is BiometricAvailability.Available) {
            // Fall back to regular encrypted storage
            return storeTokenPlaintext(token, userId)
        }

        return withContext(Dispatchers.IO) {
            try {
                // Initialize cipher
                val cipher = initCipher(Cipher.ENCRYPT_MODE)

                // Encrypt token
                val encryptedToken = cipher.doFinal(token.toByteArray())

                // Store encrypted token
                val prefs = context.getSharedPreferences(TOKEN_PREFS, Context.MODE_PRIVATE)
                val edit = prefs.edit()
                edit.putString("${TOKEN_CIPHER_PREFIX}${userId}", Base64.encodeToString(encryptedToken, Base64.NO_WRAP))
                edit.putString("${TOKEN_CIPHER_PREFIX}${userId}_iv", Base64.encodeToString(cipher.iv, Base64.NO_WRAP))
                edit.apply()

                Result.success(Unit)
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Retrieve Matrix access token (requires biometric authentication)
     */
    suspend fun retrieveToken(
        userId: String,
        prompt: String = "Authenticate to access chat"
    ): Result<String> {
        val availability = isBiometricAvailable()
        if (availability !is BiometricAvailability.Available) {
            // Try to get from plaintext fallback
            return getTokenPlaintext(userId)
        }

        return suspendCancellableCoroutine { continuation ->
            val promptInfo = BiometricPrompt.PromptInfo.Builder()
                .setTitle("Authentication Required")
                .setSubtitle(prompt)
                .setNegativeButtonText("Cancel")
                .setAllowedAuthenticators(
                    BiometricPrompt.Authenticators.BIOMETRIC_STRONG or
                    BiometricPrompt.Authenticators.DEVICE_CREDENTIAL
                )
                .build()

            val cryptoObject = try {
                val cipher = initCipher(Cipher.DECRYPT_MODE, userId)
                BiometricPrompt.CryptoObject(cipher)
            } catch (e: Exception) {
                continuation.resume(Result.failure(e))
                return@suspendCancellableCoroutine
            }

            biometricPrompt.authenticate(promptInfo, cryptoObject,
                executor,
                object : BiometricPrompt.AuthenticationCallback() {
                    override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                        val cipher = result.cryptoObject?.cipher
                        if (cipher != null) {
                            try {
                                // Get encrypted token
                                val prefs = context.getSharedPreferences(TOKEN_PREFS, Context.MODE_PRIVATE)
                                val encryptedToken = prefs.getString("${TOKEN_CIPHER_PREFIX}${userId}", null)
                                val iv = prefs.getString("${TOKEN_CIPHER_PREFIX}${userId}_iv", null)

                                if (encryptedToken != null && iv != null) {
                                    val encrypted = Base64.decode(encryptedToken, Base64.NO_WRAP)
                                    val ivBytes = Base64.decode(iv, Base64.NO_WRAP)

                                    // Decrypt token
                                    cipher.iv = ivBytes
                                    val decrypted = cipher.doFinal(encrypted)
                                    continuation.resume(Result.success(String(decrypted)))
                                } else {
                                    continuation.resume(Result.failure(Exception("Token not found")))
                                }
                            } catch (e: Exception) {
                                continuation.resume(Result.failure(e))
                            }
                        } else {
                            continuation.resume(Result.failure(Exception("No crypto object")))
                        }
                    }

                    override fun onAuthenticationFailed() {
                        continuation.resume(Result.failure(SecurityException("Authentication failed")))
                    }

                    override fun onError(error: Int, errString: CharSequence) {
                        continuation.resume(Result.failure(Exception("Biometric error: $errString")))
                    }
                }
            )
        }
    }

    /**
     * Clear stored token
     */
    fun clearToken(userId: String): Result<Unit> {
        return try {
            val prefs = context.getSharedPreferences(TOKEN_PREFS, Context.MODE_PRIVATE)
            prefs.edit()
                .remove("${TOKEN_CIPHER_PREFIX}${userId}")
                .remove("${TOKEN_CIPHER_PREFIX}${userId}_iv")
                .remove("plaintext_${userId}")
                .apply()
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Initialize cipher for encryption/decryption
     */
    private fun initCipher(mode: Int, userId: String? = null): Cipher {
        val key = getOrCreateKey()

        val cipher = Cipher.getInstance(TRANSFORMATION)
        when (mode) {
            Cipher.ENCRYPT_MODE -> {
                cipher.init(mode, key)
                cipher
            }
            Cipher.DECRYPT_MODE -> {
                if (userId != null) {
                    val prefs = context.getSharedPreferences(TOKEN_PREFS, Context.MODE_PRIVATE)
                    val iv = prefs.getString("${TOKEN_CIPHER_PREFIX}${userId}_iv", null)
                    if (iv != null) {
                        val ivBytes = Base64.decode(iv, Base64.NO_WRAP)
                        val ivSpec = GCMParameterSpec(GCM_TAG_LENGTH, ivBytes)
                        cipher.init(mode, key, ivSpec)
                        cipher
                    } else {
                        throw Exception("No IV found for user $userId")
                    }
                } else {
                    throw Exception("User ID required for decryption")
                }
            }
            else -> throw IllegalArgumentException("Invalid cipher mode")
        }
    }

    /**
     * Get or create biometric key
     */
    private fun getOrCreateKey(): SecretKey {
        keyStore.load(null)

        val existingKey = keyStore.getKey(KEY_NAME, null) as? SecretKey
        if (existingKey != null) {
            return existingKey
        }

        // Generate new key
        val keyGenerator = KeyGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_AES,
            ANDROID_KEYSTORE
        )
        val keyGenSpec = KeyGenParameterSpec.Builder(
            KEY_NAME,
            KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
        )
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setUserAuthenticationRequired(true)
            .setUserAuthenticationValidityDurationSeconds(30) // 30 seconds
            .build()

        keyGenerator.init(keyGenSpec)
        return keyGenerator.generateKey()
    }

    private fun storeTokenPlaintext(token: String, userId: String): Result<Unit> {
        // Fallback for devices without biometric support
        return try {
            val prefs = context.getSharedPreferences(TOKEN_PREFS, Context.MODE_PRIVATE)
            prefs.edit().putString("plaintext_${userId}", token).apply()
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun getTokenPlaintext(userId: String): Result<String> {
        return try {
            val prefs = context.getSharedPreferences(TOKEN_PREFS, Context.MODE_PRIVATE)
            val token = prefs.getString("plaintext_${userId}", null)
            if (token != null) {
                Result.success(token)
            } else {
                Result.failure(Exception("Token not found"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    companion object {
        private const val ANDROID_KEYSTORE = "AndroidKeyStore"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val GCM_TAG_LENGTH = 128
    }
}

sealed class BiometricAvailability {
    object Available : BiometricAvailability()
    object NoEnrollment : BiometricAvailability()
    object NoHardware : BiometricAvailability()
    object HwUnavailable : BiometricAvailability()
    data class Unknown(val code: Int) : BiometricAvailability()
}
```

### 2.3 Biometric Policy Configuration

```kotlin
/**
 * Biometric authentication policy
 */
data class BiometricPolicy(
    val sessionTimeout: Duration = 5.minutes,
    val biometricPromptInterval: Duration = 1.minute,
    val requireOnAppLaunch: Boolean = true,
    val requireOnBackground: Boolean = true,
    val requireOnSensitiveActions: Boolean = true,
    val sensitiveActions: Set<String> = setOf(
        "SEND_COMMAND",
        "START_AGENT",
        "MODIFY_CONFIG",
        "DELETE_DATA"
    )
)

/**
 * Policy enforcer
 */
class BiometricPolicyEnforcer(
    private val policy: BiometricPolicy,
    private val tokenManager: BiometricTokenManager,
    private val sessionManager: SessionManager
) {

    private var lastAuthenticationTime: Instant? = null

    /**
     * Check if authentication is required for an action
     */
    fun shouldAuthenticate(action: AppAction): Boolean {
        return when (action) {
            is AppAction.Launch -> policy.requireOnAppLaunch
            is AppAction.ResumeFromBackground -> policy.requireOnBackground
            is AppAction.SensitiveAction -> {
                policy.requireOnSensitiveActions &&
                action.type in policy.sensitiveActions
            }
            else -> false
        }
    }

    /**
     * Check if current session is valid
     */
    fun isSessionValid(): Boolean {
        val lastAuth = lastAuthenticationTime ?: return false
        val elapsed = Clock.System.now() - lastAuth
        return elapsed < policy.sessionTimeout
    }

    /**
     * Perform authentication if required
     */
    suspend fun authenticateIfRequired(
        action: AppAction,
        userId: String
    ): Result<Unit> {
        if (!shouldAuthenticate(action)) {
            return Result.success(Unit)
        }

        if (isSessionValid()) {
            return Result.success(Unit)
        }

        return tokenManager.retrieveToken(
            userId = userId,
            prompt = getPromptMessage(action)
        ).map {}
    }

    /**
     * Update last authentication time
     */
    fun onAuthenticationSucceeded() {
        lastAuthenticationTime = Clock.System.now()
    }

    private fun getPromptMessage(action: AppAction): String {
        return when (action) {
            is AppAction.Launch -> "Authenticate to access ArmorClaw"
            is AppAction.ResumeFromBackground -> "Authenticate to resume session"
            is AppAction.SensitiveAction -> "Authenticate to ${action.type.lowercase().replace("_", " ")}"
            else -> "Authentication required"
        }
    }
}

sealed class AppAction {
    object Launch : AppAction()
    object ResumeFromBackground : AppAction()
    data class SensitiveAction(val type: String) : AppAction()
}
```

---

## 3. Secure Clipboard

### 3.1 Architecture Overview

Secure clipboard protects sensitive data copied from the app by auto-clearing after a short duration.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Secure Clipboard Flow                             │
│                                                                      │
│  User copies sensitive data:                                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  "Copy Token" button pressed                                 │   │
│  │          │                                                   │   │
│  │          ▼                                                   │   │
│  │  Show warning: "This will be cleared after 30 seconds"      │   │
│  │          │                                                   │   │
│  │          ▼                                                   │   │
│  │  Copy to clipboard with auto-clear timer                    │   │
│  │          │                                                   │   │
│  │          ▼                                                   │   │
│  │  Start countdown (show in UI)                               │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  Timer expires:                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Check clipboard content                                     │   │
│  │          │                                                   │   │
│  │          ▼                                                   │   │
│  │  If matches our data → Clear it                              │   │
│  │  If changed by user → Keep user's content                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 Implementation

```kotlin
/**
 * Secure Clipboard Manager
 */
class SecureClipboardManager(
    private val context: Context,
    private val coroutineScope: CoroutineScope
) {

    companion object {
        private const val DEFAULT_AUTO_CLEAR_DURATION = 30_000L // 30 seconds
        private val HASH_KEY = "clipboard_hash"
    }

    private val clipboardManager = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    private val securePrefs = context.getSharedPreferences("secure_clipboard", Context.MODE_PRIVATE)

    private val _autoClearState = MutableStateFlow<AutoClearState?>(null)
    val autoClearState: StateFlow<AutoClearState?> = _autoClearState.asStateFlow()

    private var autoClearJob: Job? = null

    /**
     * Copy sensitive data to clipboard with auto-clear
     */
    fun copySensitive(
        data: String,
        autoClearAfter: Duration = DEFAULT_AUTO_CLEAR_DURATION.milliseconds,
        warningShown: Boolean = true
    ) {
        // Show warning if configured
        if (warningShown) {
            showWarningToast(autoClearAfter)
        }

        // Store hash for later verification
        val dataHash = hashData(data)
        securePrefs.edit().putLong(HASH_KEY, dataHash).apply()

        // Copy to clipboard
        val clip = ClipData.newPlainText("ArmorClaw", data)
        clipboardManager.setPrimaryClip(clip)

        // Start auto-clear timer
        startAutoClearTimer(data, autoClearAfter)
    }

    /**
     * Copy regular data (no auto-clear)
     */
    fun copy(data: String) {
        val clip = ClipData.newPlainText("ArmorClaw", data)
        clipboardManager.setPrimaryClip(clip)
    }

    /**
     * Get current clipboard content
     */
    fun getClipboardContent(): String? {
        val clip = clipboardManager.primaryClip
        return if (clip != null && clip.itemCount > 0) {
            clip.getItemAt(0).text?.toString()
        } else {
            null
        }
    }

    /**
     * Check if clipboard contains sensitive data
     */
    fun containsSensitiveData(): Boolean {
        val currentHash = getClipboardContent()?.let { hashData(it) }
        val storedHash = securePrefs.getLong(HASH_KEY, -1)

        return currentHash == storedHash && storedHash != -1L
    }

    /**
     * Clear clipboard immediately
     */
    fun clear() {
        clipboardManager.setPrimaryClip(ClipData.newPlainText("", ""))
        securePrefs.edit().remove(HASH_KEY).apply()
        cancelAutoClearTimer()
    }

    /**
     * Start auto-clear timer
     */
    private fun startAutoClearTimer(originalData: String, duration: Duration) {
        // Cancel any existing timer
        cancelAutoClearTimer()

        val expiryTime = Clock.System.now() + duration

        autoClearJob = coroutineScope.launch {
            _autoClearState.value = AutoClearState.Active(
                expiresAt = expiryTime,
                originalData = originalData
            )

            delay(duration)

            // Check if clipboard still contains our data
            val currentContent = getClipboardContent()
            val currentHash = currentContent?.let { hashData(it) }
            val storedHash = securePrefs.getLong(HASH_KEY, -1)

            if (currentHash == storedHash) {
                // Clear the clipboard
                clipboardManager.setPrimaryClip(ClipData.newPlainText("", ""))
                securePrefs.edit().remove(HASH_KEY).apply()
                _autoClearState.value = AutoClearState.Cleared
            } else {
                // User has copied something else, don't clear
                _autoClearState.value = AutoClearState.UserOverridden
            }

            delay(2.seconds)
            _autoClearState.value = null
        }
    }

    /**
     * Cancel auto-clear timer
     */
    private fun cancelAutoClearTimer() {
        autoClearJob?.cancel()
        autoClearJob = null
        _autoClearState.value = null
    }

    /**
     * Show warning toast
     */
    private fun showWarningToast(duration: Duration) {
        val message = "Sensitive data copied. Will be cleared after ${duration.inWholeSeconds} seconds."
        Toast.makeText(context, message, Toast.LENGTH_LONG).show()
    }

    /**
     * Hash clipboard data for comparison
     */
    private fun hashData(data: String): Long {
        // Simple hash for comparison (not for security)
        return data.hashCode().toLong()
    }
}

sealed class AutoClearState {
    data class Active(
        val expiresAt: Instant,
        val originalData: String
    ) : AutoClearState()

    object Cleared : AutoClearState()
    object UserOverridden : AutoClearState()
}
```

### 3.3 UI Components

```kotlin
/**
 * Auto-clear countdown indicator
 */
@Composable
fun AutoClearIndicator(
    state: AutoClearState?,
    modifier: Modifier = Modifier
) {
    val currentTime = Clock.System.now()
    val remaining = when (state) {
        is AutoClearState.Active -> {
            val remaining = state.expiresAt - currentTime
            maxOf(0, remaining.inWholeSeconds)
        }
        else -> null
    }

    if (remaining != null) {
        Surface(
            modifier = modifier
                .padding(8.dp)
                .clip(RoundedCornerShape(8.dp)),
            color = MaterialTheme.colorScheme.errorContainer,
            contentColor = MaterialTheme.colorScheme.onErrorContainer
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Icon(
                    imageVector = Icons.Outlined.ContentCopy,
                    contentDescription = null,
                    modifier = Modifier.size(16.dp)
                )
                Text(
                    text = "Clipboard clears in ${remaining}s",
                    style = MaterialTheme.typography.bodySmall
                )
                IconButton(
                    onClick = { /* Clear manually */ },
                    modifier = Modifier.size(20.dp)
                ) {
                    Icon(
                        imageVector = Icons.Outlined.Close,
                        contentDescription = "Clear now",
                        modifier = Modifier.size(14.dp)
                    )
                }
            }
        }
    }
}

/**
 * Copy button with secure clipboard
 */
@Composable
fun SecureCopyButton(
    data: String,
    label: String = "Copy",
    modifier: Modifier = Modifier,
    clipboardManager: SecureClipboardManager
) {
    val context = LocalContext.current
    val autoClearState by clipboardManager.autoClearState.collectAsState()

    Button(
        onClick = {
            clipboardManager.copySensitive(
                data = data,
                warningShown = true
            )
            Toast.makeText(context, "Copied to clipboard", Toast.LENGTH_SHORT).show()
        },
        modifier = modifier
    ) {
        Icon(Icons.Outlined.ContentCopy, contentDescription = null)
        Spacer(Modifier.width(8.dp))
        Text(label)
    }

    // Show auto-clear indicator
    if (autoClearState != null) {
        AutoClearIndicator(
            state = autoClearState,
            modifier = Modifier.fillMaxWidth()
        )
    }
}
```

### 3.4 Integration Example

```kotlin
/**
 * Example: Token display with secure copy
 */
@Composable
fun TokenDisplayScreen(
    token: String,
    maskedToken: String,
    clipboardManager: SecureClipboardManager
) {
    var showToken by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp)
    ) {
        // Security warning
        Card(
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.errorContainer
            )
        ) {
            Text(
                text = "⚠️ Keep your token secure. Never share it with others.",
                modifier = Modifier.padding(16.dp),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onErrorContainer
            )
        }

        Spacer(Modifier.height(16.dp))

        // Token display
        Card(
            modifier = Modifier.fillMaxWidth()
        ) {
            Column(
                modifier = Modifier.padding(16.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Access Token",
                        style = MaterialTheme.typography.titleMedium
                    )

                    IconButton(onClick = { showToken = !showToken }) {
                        Icon(
                            imageVector = if (showToken) {
                                Icons.Outlined.VisibilityOff
                            } else {
                                Icons.Outlined.Visibility
                            },
                            contentDescription = if (showToken) "Hide" else "Show"
                        )
                    }
                }

                Spacer(Modifier.height(8.dp))

                Text(
                    text = if (showToken) token else maskedToken,
                    style = MaterialTheme.typography.bodyMonospace,
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState())
                )

                Spacer(Modifier.height(16.dp))

                SecureCopyButton(
                    data = token,
                    label = "Copy Token",
                    modifier = Modifier.fillMaxWidth(),
                    clipboardManager = clipboardManager
                )
            }
        }

        // Auto-clear indicator
        val autoClearState by clipboardManager.autoClearState.collectAsState()
        if (autoClearState != null) {
            AutoClearIndicator(
                state = autoClearState,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 16.dp)
            )
        }
    }
}
```

---

## 4. Security Integration & Testing

### 4.1 Security Logger

```kotlin
/**
 * Centralized security event logging
 */
interface SecurityLogger {
    fun log(event: SecurityEvent)
}

data class SecurityEvent(
    val type: SecurityEventType,
    val timestamp: Instant = Clock.System.now(),
    val details: Map<String, Any> = emptyMap()
)

sealed class SecurityEventType {
    object PinningSuccess : SecurityEventType()
    data class PinningFailure(val url: String, val certHash: String) : SecurityEventType()
    object PinningSkipped : SecurityEventType()
    data class PinningExpired(val hash: String) : SecurityEventType()
    object BiometricSuccess : SecurityEventType()
    object BiometricFailure : SecurityEventType()
    object ClipboardCleared : SecurityEventType()
    data class ClipboardOverridden(val timestamp: Instant) : SecurityEventType()
}

class CompositeSecurityLogger(
    private val loggers: List<SecurityLogger>
) : SecurityLogger {
    override fun log(event: SecurityEvent) {
        loggers.forEach { it.log(event) }
    }
}

class ConsoleSecurityLogger : SecurityLogger {
    override fun log(event: SecurityEvent) {
        Log.d("Security", "${event.type}: ${event.details}")
    }
}

class RemoteSecurityLogger(
    private val apiClient: ApiClient
) : SecurityLogger {
    override fun log(event: SecurityEvent) {
        // Send to remote monitoring service
        apiClient.postSecurityEvent(event)
    }
}
```

### 4.2 Security Checklist

```
╔═══════════════════════════════════════════════════════════════════╗
║                    Security Implementation Checklist                ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                     ║
║ Certificate Pinning                                                ║
║ ☐ Extract SPKI hash from server certificate                        ║
║ ☐ Implement pinning interceptor                                    ║
║ ☐ Add pin extraction utility for new servers                       ║
║ ☐ Implement remote pin update via Matrix                           ║
║ ☐ Test pin bypass attempts                                         ║
║ ☐ Test pin expiration handling                                     ║
║ ☐ Debug build bypass (optional)                                    ║
║                                                                     ║
║ Biometric Authentication                                            ║
║ ☐ Android implementation (BiometricPrompt)                         ║
║ ☐ iOS implementation (LocalAuthentication)                         ║
║ ☐ KeyStore/Keychain integration                                    ║
║ ☐ Token encryption/decryption                                      ║
║ ☐ Session timeout enforcement                                      ║
║ ☐ Policy enforcement for sensitive actions                         ║
║ ☐ Fallback for devices without biometric                           ║
║ ☐ Test all authentication failures                                 ║
║                                                                     ║
║ Secure Clipboard                                                   ║
║ ☐ Auto-clear timer implementation                                 ║
║ ☐ Hash-based content verification                                  ║
║ ☐ User override detection                                          ║
║ ☐ Manual clear option                                              ║
║ ☐ UI countdown indicator                                           ║
║ ☐ Test clipboard clearing after timeout                            ║
║ ☐ Test user copy override                                          ║
║                                                                     ║
║ Integration                                                        ║
║ ☐ Security event logging                                          ║
║ ☐ Remote monitoring integration                                   ║
║ ☐ Error reporting                                                 ║
║ ☐ Security audit trail                                            ║
║                                                                     ║
╚═══════════════════════════════════════════════════════════════════╝
```

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-10
**Status:** Ready for Implementation
