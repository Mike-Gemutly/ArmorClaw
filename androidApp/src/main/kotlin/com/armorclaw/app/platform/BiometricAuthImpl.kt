package com.armorclaw.app.platform

import android.content.Context
import com.armorclaw.shared.platform.biometric.BiometricAuth
import com.armorclaw.shared.platform.biometric.BiometricResult

/**
 * Android implementation wrapper for BiometricAuth.
 * This delegates to the shared module's implementation.
 */
class BiometricAuthImpl(
    context: Context
) {
    val delegate = BiometricAuth.getInstance()

    init {
        BiometricAuth.setContext(context)
    }

    val isAvailable: Boolean
        get() = delegate.isAvailable()

    suspend fun authenticate(prompt: String): BiometricResult {
        val result = delegate.authenticate(prompt)
        return if (result.isSuccess) {
            BiometricResult.Success
        } else {
            val message = result.exceptionOrNull()?.message ?: "Authentication failed"
            BiometricResult.Error(message)
        }
    }

    suspend fun encrypt(data: String, keyName: String): Result<String> {
        return delegate.encrypt(data, keyName)
    }

    suspend fun decrypt(data: String, keyName: String): Result<String> {
        return delegate.decrypt(data, keyName)
    }

    suspend fun deleteKey(keyName: String): Result<Unit> {
        return delegate.deleteKey(keyName)
    }
}
