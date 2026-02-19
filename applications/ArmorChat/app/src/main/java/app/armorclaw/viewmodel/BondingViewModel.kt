package app.armorclaw.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.armorclaw.network.BridgeApi
import app.armorclaw.repository.ClaimingState
import app.armorclaw.repository.SetupRepository
import app.armorclaw.utils.BridgeError
import app.armorclaw.utils.ErrorHandler
import app.armorclaw.utils.retryWithBackoff
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * ViewModel for the bonding/admin claiming flow
 */
class BondingViewModel(
    private val repository: SetupRepository = SetupRepository()
) : ViewModel() {

    // UI State
    private val _uiState = MutableStateFlow<BondingUiState>(BondingUiState.Idle)
    val uiState: StateFlow<BondingUiState> = _uiState.asStateFlow()

    // Lockdown status
    private val _lockdownStatus = MutableStateFlow<BridgeApi.LockdownStatus?>(null)
    val lockdownStatus: StateFlow<BridgeApi.LockdownStatus?> = _lockdownStatus.asStateFlow()

    // Retry attempt counter
    private val _retryAttempt = MutableStateFlow(0)
    val retryAttempt: StateFlow<Int> = _retryAttempt.asStateFlow()

    // Form state
    var displayName = MutableStateFlow("")
    var deviceName = MutableStateFlow("")
    var passphrase = MutableStateFlow("")
    var confirmPassphrase = MutableStateFlow("")

    // Validation
    val isFormValid: Boolean
        get() = displayName.value.isNotBlank() &&
                deviceName.value.isNotBlank() &&
                passphrase.value.length >= 8 &&
                passphrase.value == confirmPassphrase.value

    /**
     * Check bridge status and if we can claim
     */
    fun checkBridgeStatus() {
        viewModelScope.launch {
            _uiState.value = BondingUiState.CheckingBridge
            _retryAttempt.value = 0

            try {
                val status = retryWithBackoff(
                    maxAttempts = 3,
                    baseDelayMs = 2000,
                    onRetry = { _retryAttempt.value++ }
                ) {
                    repository.getLockdownStatus().getOrThrow()
                }

                _lockdownStatus.value = status
                if (!status.admin_established) {
                    _uiState.value = BondingUiState.ReadyToClaim
                } else {
                    _uiState.value = BondingUiState.AlreadyClaimed
                }
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _uiState.value = BondingUiState.BridgeError(error)
            }
        }
    }

    /**
     * Claim ownership
     */
    fun claimOwnership() {
        if (!isFormValid) {
            _uiState.value = BondingUiState.ValidationError(
                "Please fill in all fields correctly. Passphrase must be at least 8 characters."
            )
            return
        }

        viewModelScope.launch {
            _uiState.value = BondingUiState.Claiming

            try {
                val result = retryWithBackoff(maxAttempts = 2) {
                    repository.startClaiming(
                        displayName = displayName.value,
                        deviceName = deviceName.value,
                        passphrase = passphrase.value
                    ).getOrThrow()
                }

                when (result) {
                    is ClaimingState.Success -> {
                        _uiState.value = BondingUiState.Success(
                            adminId = result.adminId,
                            deviceId = result.deviceId,
                            sessionToken = result.sessionToken,
                            nextStep = result.nextStep
                        )
                    }
                    is ClaimingState.Error -> {
                        _uiState.value = BondingUiState.BridgeError(
                            BridgeError(
                                code = app.armorclaw.utils.ErrorCode.UNKNOWN,
                                title = "Claim Failed",
                                message = result.message,
                                recoverable = true
                            )
                        )
                    }
                    ClaimingState.Loading -> {
                        // Already in Claiming state
                    }
                }
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _uiState.value = BondingUiState.BridgeError(error)
            }
        }
    }

    /**
     * Reset to try again after error
     */
    fun reset() {
        _uiState.value = BondingUiState.ReadyToClaim
        _retryAttempt.value = 0
        displayName.value = ""
        deviceName.value = ""
        passphrase.value = ""
        confirmPassphrase.value = ""
    }

    /**
     * Clear passphrase from memory
     */
    fun clearSensitiveData() {
        passphrase.value = ""
        confirmPassphrase.value = ""
    }

    /**
     * Clear current error state
     */
    fun clearError() {
        _uiState.value = BondingUiState.ReadyToClaim
    }
}

/**
 * UI state for the bonding flow
 */
sealed class BondingUiState {
    object Idle : BondingUiState()
    object CheckingBridge : BondingUiState()
    object ReadyToClaim : BondingUiState()
    object Claiming : BondingUiState()
    object AlreadyClaimed : BondingUiState()

    data class Success(
        val adminId: String,
        val deviceId: String,
        val sessionToken: String,
        val nextStep: String
    ) : BondingUiState()

    data class ValidationError(val message: String) : BondingUiState()

    data class BridgeError(val error: BridgeError) : BondingUiState()
}
