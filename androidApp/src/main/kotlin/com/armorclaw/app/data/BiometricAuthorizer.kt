package com.armorclaw.app.data

import android.content.Context
import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import com.armorclaw.app.platform.BiometricAuthImpl
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume

/**
 * Biometric Authorizer
 *
 * Handles biometric authentication for sensitive operations like
 * approving PII access requests or unsealing the keystore.
 *
 * ## Usage
 * ```kotlin
 * val authorizer = BiometricAuthorizer(context, secureStorage)
 *
 * // Authorize a field access
 * val result = authorizer.authorizeField(
 *     fieldName = "Credit Card Number",
 *     activity = this
 * )
 *
 * result.onSuccess { decryptedValue ->
 *     // Use the decrypted value
 * }
 * ```
 *
 * ## Integration Points
 * - BlindFillCard: Requires biometric for CRITICAL fields
 * - UnsealScreen: Optional biometric unlock for keystore
 */
class BiometricAuthorizer(
    private val context: Context,
    private val biometricAuth: BiometricAuthImpl
) {
    private val executor = ContextCompat.getMainExecutor(context)

    /**
     * Check if biometric authentication is available
     */
    fun isBiometricAvailable(): Boolean {
        val biometricManager = BiometricManager.from(context)
        return biometricManager.canAuthenticate(
            BiometricManager.Authenticators.BIOMETRIC_STRONG
        ) == BiometricManager.BIOMETRIC_SUCCESS
    }

    /**
     * Authorize access to a sensitive field using biometric authentication.
     *
     * @param fieldName Human-readable name of the field being accessed
     * @param activity The FragmentActivity to host the biometric prompt
     * @return Result containing the decrypted field value or an error
     */
    suspend fun authorizeField(
        fieldName: String,
        activity: FragmentActivity
    ): Result<String> = suspendCancellableCoroutine { cont ->
        AppLogger.debug(
            tag = LogTag.Domain.ControlPlane,
            message = "Starting biometric authorization for field",
            data = mapOf("fieldName" to fieldName)
        )

        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle("Authorize sensitive data")
            .setSubtitle("Allow access to: $fieldName")
            .setDescription("Biometric verification required to access this sensitive information")
            .setNegativeButtonText("Cancel")
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build()

        val biometricPrompt = BiometricPrompt(
            activity,
            executor,
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(
                    result: BiometricPrompt.AuthenticationResult
                ) {
                    AppLogger.info(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Biometric authorization succeeded",
                        data = mapOf("fieldName" to fieldName)
                    )

                    // In a real implementation, we would decrypt the field here
                    // For now, return a success result
                    cont.resume(Result.success("authorized"))
                }

                override fun onAuthenticationFailed() {
                    AppLogger.warning(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Biometric authorization failed",
                        data = mapOf("fieldName" to fieldName)
                    )
                    // Don't resume yet - user can retry
                }

                override fun onAuthenticationError(
                    errorCode: Int,
                    errString: CharSequence
                ) {
                    AppLogger.error(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Biometric authorization error",
                        data = mapOf(
                            "fieldName" to fieldName,
                            "errorCode" to errorCode,
                            "error" to errString.toString()
                        )
                    )
                    cont.resume(Result.failure(Exception(errString.toString())))
                }
            }
        )

        biometricPrompt.authenticate(promptInfo)

        cont.invokeOnCancellation {
            biometricPrompt.cancelAuthentication()
        }
    }

    /**
     * Authorize keystore unseal operation using biometric authentication.
     *
     * @param activity The FragmentActivity to host the biometric prompt
     * @return Result indicating success or failure
     */
    suspend fun authorizeUnseal(
        activity: FragmentActivity
    ): Result<Unit> = suspendCancellableCoroutine { cont ->
        AppLogger.info(
            tag = LogTag.Domain.ControlPlane,
            message = "Starting biometric authorization for keystore unseal"
        )

        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle("Unseal VPS Keystore")
            .setSubtitle("Authenticate to decrypt stored credentials")
            .setDescription("Your credentials will be available for 4 hours")
            .setNegativeButtonText("Cancel")
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build()

        val biometricPrompt = BiometricPrompt(
            activity,
            executor,
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(
                    result: BiometricPrompt.AuthenticationResult
                ) {
                    AppLogger.info(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Biometric keystore unseal succeeded"
                    )
                    cont.resume(Result.success(Unit))
                }

                override fun onAuthenticationFailed() {
                    AppLogger.warning(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Biometric keystore unseal failed"
                    )
                }

                override fun onAuthenticationError(
                    errorCode: Int,
                    errString: CharSequence
                ) {
                    AppLogger.error(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Biometric keystore unseal error",
                        data = mapOf("errorCode" to errorCode, "error" to errString.toString())
                    )
                    cont.resume(Result.failure(Exception(errString.toString())))
                }
            }
        )

        biometricPrompt.authenticate(promptInfo)

        cont.invokeOnCancellation {
            biometricPrompt.cancelAuthentication()
        }
    }

    /**
     * Authorize multiple PII fields at once (batch approval)
     *
     * @param fieldNames List of field names being accessed
     * @param activity The FragmentActivity to host the biometric prompt
     * @return Result containing the set of approved field names
     */
    suspend fun authorizeFields(
        fieldNames: Set<String>,
        activity: FragmentActivity
    ): Result<Set<String>> = suspendCancellableCoroutine { cont ->
        AppLogger.debug(
            tag = LogTag.Domain.ControlPlane,
            message = "Starting batch biometric authorization",
            data = mapOf("fieldCount" to fieldNames.size)
        )

        val fieldsDescription = if (fieldNames.size == 1) {
            fieldNames.first()
        } else {
            "${fieldNames.size} sensitive fields"
        }

        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle("Authorize Access")
            .setSubtitle("Allow access to: $fieldsDescription")
            .setDescription("Biometric verification required for sensitive data access")
            .setNegativeButtonText("Cancel")
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build()

        val biometricPrompt = BiometricPrompt(
            activity,
            executor,
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(
                    result: BiometricPrompt.AuthenticationResult
                ) {
                    AppLogger.info(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Batch biometric authorization succeeded",
                        data = mapOf("approvedFields" to fieldNames.size)
                    )
                    cont.resume(Result.success(fieldNames))
                }

                override fun onAuthenticationFailed() {
                    AppLogger.warning(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Batch biometric authorization failed"
                    )
                }

                override fun onAuthenticationError(
                    errorCode: Int,
                    errString: CharSequence
                ) {
                    AppLogger.error(
                        tag = LogTag.Domain.ControlPlane,
                        message = "Batch biometric authorization error",
                        data = mapOf("errorCode" to errorCode)
                    )
                    cont.resume(Result.failure(Exception(errString.toString())))
                }
            }
        )

        biometricPrompt.authenticate(promptInfo)

        cont.invokeOnCancellation {
            biometricPrompt.cancelAuthentication()
        }
    }
}
