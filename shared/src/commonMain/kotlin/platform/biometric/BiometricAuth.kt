package com.armorclaw.shared.platform.biometric

import kotlinx.coroutines.flow.Flow
import kotlin.time.Duration

expect class BiometricAuth() {
    suspend fun authenticate(prompt: String): Result<String>
    fun isAvailable(): Boolean
    fun observeAvailability(): Flow<Boolean>
    
    suspend fun encrypt(data: String, keyName: String): Result<String>
    suspend fun decrypt(data: String, keyName: String): Result<String>
    suspend fun deleteKey(keyName: String): Result<Unit>
}

data class BiometricConfig(
    val promptTitle: String = "Authentication Required",
    val promptSubtitle: String = "",
    val promptDescription: String = "",
    val negativeButtonText: String = "Cancel",
    val allowDeviceCredential: Boolean = true,
    val timeout: Duration? = null
)

sealed class BiometricResult {
    object Success : BiometricResult()
    data class Error(val message: String, val isRecoverable: Boolean = true) : BiometricResult()
    object Cancelled : BiometricResult()
    object NotAvailable : BiometricResult()
    object NotEnrolled : BiometricResult()
    object LockedOut : BiometricResult()
}
