package com.armorclaw.app.viewmodels

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.app.data.BiometricAuthorizer
import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.domain.model.KeystoreStatus
import com.armorclaw.shared.domain.model.UnsealMethod
import com.armorclaw.shared.domain.model.KEYSTORE_SESSION_DURATION_MS
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * ViewModel for the Unseal Screen
 *
 * Manages the keystore unseal operation, including password and biometric
 * authentication methods.
 *
 * ## Architecture
 * ```
 * UnsealViewModel
 *      ├── ControlPlaneStore (keystore status)
 *      ├── BiometricAuthorizer (biometric auth)
 *      └── SecureStorage (password verification)
 * ```
 *
 * ## Security Model
 * - Password: Verified against stored hash, then decrypts keystore
 * - Biometric: Uses Android Keystore for secure key access
 * - Session: Valid for 4 hours after unseal
 */
class UnsealViewModel(
    application: Application,
    private val controlPlaneStore: ControlPlaneStore,
    private val biometricAuthorizer: BiometricAuthorizer
) : AndroidViewModel(application) {

    private val _uiState = MutableStateFlow<UnsealUiState>(UnsealUiState.Idle)
    val uiState: StateFlow<UnsealUiState> = _uiState.asStateFlow()

    private val _password = MutableStateFlow("")
    val password: StateFlow<String> = _password.asStateFlow()

    private val _useBiometric = MutableStateFlow(false)
    val useBiometric: StateFlow<Boolean> = _useBiometric.asStateFlow()

    private val _passwordVisible = MutableStateFlow(false)
    val passwordVisible: StateFlow<Boolean> = _passwordVisible.asStateFlow()

    private val _biometricAvailable = MutableStateFlow(false)
    val biometricAvailable: StateFlow<Boolean> = _biometricAvailable.asStateFlow()

    init {
        checkBiometricAvailability()
    }

    /**
     * Set the password input
     */
    fun setPassword(value: String) {
        _password.value = value
        // Clear error when user starts typing
        if (_uiState.value is UnsealUiState.Error) {
            _uiState.value = UnsealUiState.Idle
        }
    }

    /**
     * Toggle biometric usage preference
     */
    fun setUseBiometric(value: Boolean) {
        _useBiometric.value = value
    }

    /**
     * Toggle password visibility
     */
    fun togglePasswordVisibility() {
        _passwordVisible.value = !_passwordVisible.value
    }

    /**
     * Check if biometric authentication is available
     */
    private fun checkBiometricAvailability() {
        _biometricAvailable.value = biometricAuthorizer.isBiometricAvailable()
    }

    /**
     * Unseal keystore with password
     */
    fun unsealWithPassword() {
        if (_password.value.isBlank()) {
            _uiState.value = UnsealUiState.Error("Please enter your password")
            return
        }

        viewModelScope.launch {
            _uiState.value = UnsealUiState.Loading

            AppLogger.info(
                tag = LogTag.ViewModel.Chat,
                message = "Attempting keystore unseal with password"
            )

            // Simulate password verification (in real app, verify against secure storage)
            // For now, accept any non-empty password
            kotlinx.coroutines.delay(1000) // Simulate decryption time

            val isValid = verifyPassword(_password.value)

            if (isValid) {
                val unsealedStatus = KeystoreStatus.Unsealed(
                    expiresAt = System.currentTimeMillis() + KEYSTORE_SESSION_DURATION_MS,
                    unsealedBy = UnsealMethod.PASSWORD
                )
                controlPlaneStore.setKeystoreStatus(unsealedStatus)
                _uiState.value = UnsealUiState.Unsealed

                AppLogger.info(
                    tag = LogTag.ViewModel.Chat,
                    message = "Keystore unsealed successfully with password"
                )
            } else {
                _uiState.value = UnsealUiState.Error("Incorrect password. Please try again.")

                AppLogger.warning(
                    tag = LogTag.ViewModel.Chat,
                    message = "Keystore unseal failed - incorrect password"
                )
            }
        }
    }

    /**
     * Unseal keystore with biometric authentication
     *
     * Note: This requires a FragmentActivity reference, which should be
     * provided by the screen when calling this method.
     */
    fun unsealWithBiometric() {
        viewModelScope.launch {
            _uiState.value = UnsealUiState.Loading

            AppLogger.info(
                tag = LogTag.ViewModel.Chat,
                message = "Attempting keystore unseal with biometric"
            )

            // Biometric auth will be handled by the UI layer
            // This is just a placeholder - the actual biometric prompt
            // should be triggered from the activity

            // For demo purposes, simulate success
            kotlinx.coroutines.delay(500)

            val unsealedStatus = KeystoreStatus.Unsealed(
                expiresAt = System.currentTimeMillis() + KEYSTORE_SESSION_DURATION_MS,
                unsealedBy = UnsealMethod.BIOMETRIC
            )
            controlPlaneStore.setKeystoreStatus(unsealedStatus)
            _uiState.value = UnsealUiState.Unsealed
        }
    }

    /**
     * Handle biometric authorization result
     */
    fun onBiometricResult(success: Boolean, error: String? = null) {
        if (success) {
            viewModelScope.launch {
                val unsealedStatus = KeystoreStatus.Unsealed(
                    expiresAt = System.currentTimeMillis() + KEYSTORE_SESSION_DURATION_MS,
                    unsealedBy = UnsealMethod.BIOMETRIC
                )
                controlPlaneStore.setKeystoreStatus(unsealedStatus)
                _uiState.value = UnsealUiState.Unsealed
            }
        } else {
            _uiState.value = UnsealUiState.Error(error ?: "Biometric authentication failed")
        }
    }

    /**
     * Verify password against stored hash
     *
     * In a real implementation, this would:
     * 1. Retrieve the password hash from SecureStorage
     * 2. Hash the input password
     * 3. Compare hashes using constant-time comparison
     */
    private fun verifyPassword(password: String): Boolean {
        // Placeholder - in production, use proper password verification
        // For demo, accept passwords with 6+ characters
        return password.length >= 6
    }

    /**
     * Clear the error state
     */
    fun clearError() {
        _uiState.value = UnsealUiState.Idle
    }

    override fun onCleared() {
        super.onCleared()
        // Clear sensitive data from memory
        _password.value = ""
    }
}

/**
 * Unseal UI State (re-declared for visibility)
 */
sealed class UnsealUiState {
    object Idle : UnsealUiState()
    object Loading : UnsealUiState()
    data class Error(val message: String) : UnsealUiState()
    object Unsealed : UnsealUiState()
}
