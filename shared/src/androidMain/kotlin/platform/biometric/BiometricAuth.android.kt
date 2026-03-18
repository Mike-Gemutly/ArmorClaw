package com.armorclaw.shared.platform.biometric

import android.content.Context
import android.os.CancellationSignal
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.suspendCancellableCoroutine
import java.security.KeyStore
import java.util.concurrent.Executor
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec
import kotlin.coroutines.resume

actual class BiometricAuth {

    private var context: Context? = null
    private val keyStore = KeyStore.getInstance(ANDROID_KEYSTORE)
    private val _availability = MutableStateFlow(false)

    companion object {
        private const val ANDROID_KEYSTORE = "AndroidKeyStore"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val KEY_SIZE = 256
        private const val TAG_LENGTH = 128
        private const val BIOMETRIC_PROMPT_TITLE = "ArmorClaw Authentication"
        private const val NEGATIVE_BUTTON_TEXT = "Use Password"

        @Volatile
        private var instance: BiometricAuth? = null

        fun getInstance(): BiometricAuth {
            return instance ?: synchronized(this) {
                instance ?: BiometricAuth().also { instance = it }
            }
        }

        fun setContext(context: Context) {
            getInstance().context = context.applicationContext
            getInstance().updateAvailability()
        }
    }

    init {
        keyStore.load(null)
    }

    private fun updateAvailability() {
        _availability.value = checkAvailability()
    }

    private fun checkAvailability(): Boolean {
        val ctx = context ?: return false
        val biometricManager = androidx.biometric.BiometricManager.from(ctx)
        return biometricManager.canAuthenticate(
            androidx.biometric.BiometricManager.Authenticators.BIOMETRIC_STRONG or
            androidx.biometric.BiometricManager.Authenticators.DEVICE_CREDENTIAL
        ) == androidx.biometric.BiometricManager.BIOMETRIC_SUCCESS
    }

    actual suspend fun authenticate(prompt: String): Result<String> {
        if (!isAvailable()) {
            return Result.failure(Exception("Biometric not available"))
        }

        val ctx = context ?: return Result.failure(Exception("Context not set"))
        val activity = ctx as? FragmentActivity
            ?: return Result.failure(Exception("Activity not found"))

        return try {
            val promptInfo = BiometricPrompt.PromptInfo.Builder()
                .setTitle(BIOMETRIC_PROMPT_TITLE)
                .setSubtitle(prompt)
                .setNegativeButtonText(NEGATIVE_BUTTON_TEXT)
                .setAllowedAuthenticators(
                    androidx.biometric.BiometricManager.Authenticators.BIOMETRIC_STRONG or
                    androidx.biometric.BiometricManager.Authenticators.DEVICE_CREDENTIAL
                )
                .build()

            val result = authenticateWithPrompt(activity, promptInfo)
            Result.success(result)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private suspend fun authenticateWithPrompt(
        activity: FragmentActivity,
        promptInfo: BiometricPrompt.PromptInfo
    ): String = suspendCancellableCoroutine { continuation ->
        val executor: Executor = ContextCompat.getMainExecutor(activity)

        val callback = object : BiometricPrompt.AuthenticationCallback() {
            override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                continuation.resume("success")
            }

            override fun onAuthenticationFailed() {
                // Don't resume here - user can retry
            }

            override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                if (!continuation.isCompleted) {
                    continuation.resume("error: $errString")
                }
            }
        }

        val biometricPrompt = BiometricPrompt(activity, executor, callback)

        val cancellationSignal = CancellationSignal()
        continuation.invokeOnCancellation {
            cancellationSignal.cancel()
        }

        biometricPrompt.authenticate(promptInfo)
    }

    actual fun isAvailable(): Boolean {
        return _availability.value
    }

    actual fun observeAvailability(): Flow<Boolean> = _availability.asStateFlow()

    actual suspend fun encrypt(data: String, keyName: String): Result<String> {
        return try {
            val cipher = createEncryptCipher(keyName)
            val encrypted = cipher.doFinal(data.toByteArray(Charsets.UTF_8))
            val iv = cipher.iv
            // Combine IV and encrypted data
            val combined = iv + encrypted
            Result.success(android.util.Base64.encodeToString(combined, android.util.Base64.DEFAULT))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun decrypt(data: String, keyName: String): Result<String> {
        return try {
            val combined = android.util.Base64.decode(data, android.util.Base64.DEFAULT)
            val iv = combined.copyOfRange(0, 12)
            val encrypted = combined.copyOfRange(12, combined.size)

            val cipher = createDecryptCipher(keyName, iv)
            val decrypted = cipher.doFinal(encrypted)
            Result.success(String(decrypted, Charsets.UTF_8))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun deleteKey(keyName: String): Result<Unit> {
        return try {
            if (keyStore.containsAlias(keyName)) {
                keyStore.deleteEntry(keyName)
            }
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun createEncryptCipher(keyName: String): Cipher {
        val keyGenerator = KeyGenerator.getInstance(
            android.security.keystore.KeyProperties.KEY_ALGORITHM_AES,
            ANDROID_KEYSTORE
        )

        val keyGenSpec = android.security.keystore.KeyGenParameterSpec.Builder(
            keyName,
            android.security.keystore.KeyProperties.PURPOSE_ENCRYPT or
            android.security.keystore.KeyProperties.PURPOSE_DECRYPT
        )
            .setBlockModes(android.security.keystore.KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(android.security.keystore.KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(KEY_SIZE)
            .setUserAuthenticationRequired(true)
            .build()

        keyGenerator.init(keyGenSpec)
        keyGenerator.generateKey()

        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, getSecretKey(keyName))
        return cipher
    }

    private fun createDecryptCipher(keyName: String, iv: ByteArray): Cipher {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        val key = getSecretKey(keyName)
        val spec = GCMParameterSpec(TAG_LENGTH, iv)
        cipher.init(Cipher.DECRYPT_MODE, key, spec)
        return cipher
    }

    private fun getSecretKey(keyName: String): SecretKey {
        val entry = keyStore.getEntry(keyName, null) as KeyStore.SecretKeyEntry
        return entry.secretKey
    }
}
