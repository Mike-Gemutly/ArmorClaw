package app.armorclaw.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.armorclaw.network.BridgeApi
import app.armorclaw.utils.BridgeError
import app.armorclaw.utils.ErrorHandler
import app.armorclaw.utils.retryWithBackoff
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * ViewModel for the hardening wizard flow
 *
 * Guides users through security hardening steps:
 * 1. Rotate bootstrap password
 * 2. Wipe bootstrap data
 * 3. Verify device (emoji verification)
 * 4. Backup recovery keys
 * 5. (Optional) Enable biometrics
 */
class HardeningWizardViewModel(
    private val api: BridgeApi = BridgeApi()
) : ViewModel() {

    private val _uiState = MutableStateFlow<HardeningUiState>(HardeningUiState.NotStarted)
    val uiState: StateFlow<HardeningUiState> = _uiState.asStateFlow()

    private var hardeningStatus: BridgeApi.HardeningStatus? = null

    /**
     * Load current hardening status from the bridge
     */
    fun loadState() {
        viewModelScope.launch {
            _uiState.value = HardeningUiState.Loading

            try {
                val status = retryWithBackoff(maxAttempts = 3) {
                    api.getHardeningStatus().getOrThrow()
                }
                hardeningStatus = status
                _uiState.value = HardeningUiState.Loaded(status)
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _uiState.value = HardeningUiState.Error(error.message)
            }
        }
    }

    /**
     * Rotate the bootstrap password to a new value
     * After rotation, automatically reloads the state
     */
    fun rotatePassword(newPassword: String) {
        viewModelScope.launch {
            _uiState.value = HardeningUiState.Loading

            try {
                retryWithBackoff(maxAttempts = 2) {
                    api.rotateBootstrapPassword(newPassword).getOrThrow()
                }
                loadState()
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _uiState.value = HardeningUiState.Error(error.message)
            }
        }
    }

    /**
     * Acknowledge completion of a hardening step
     * After acknowledgement, automatically reloads the state
     */
    fun acknowledgeStep(step: String) {
        viewModelScope.launch {
            _uiState.value = HardeningUiState.Loading

            try {
                retryWithBackoff(maxAttempts = 2) {
                    api.acknowledgeHardeningStep(step).getOrThrow()
                }
                loadState()
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _uiState.value = HardeningUiState.Error(error.message)
            }
        }
    }

    /**
     * Get the next incomplete hardening step
     *
     * Returns COMPLETE if all steps are done
     */
    fun getCurrentStep(): HardeningStep {
        val status = hardeningStatus
            ?: return HardeningStep.ROTATE_PASSWORD

        return when {
            !status.password_rotated -> HardeningStep.ROTATE_PASSWORD
            !status.bootstrap_wiped -> HardeningStep.WIPE_BOOTSTRAP
            !status.device_verified -> HardeningStep.VERIFY_DEVICE
            !status.recovery_backed_up -> HardeningStep.BACKUP_RECOVERY
            !status.biometrics_enabled -> HardeningStep.ENABLE_BIOMETRICS
            else -> HardeningStep.COMPLETE
        }
    }

    /**
     * Check if delegation is ready
     *
     * Delegation is ready when all mandatory steps are complete:
     * - password_rotated
     * - bootstrap_wiped
     * - device_verified
     * - recovery_backed_up
     *
     * Biometrics is optional and does not affect delegation readiness
     */
    fun isDelegationReady(): Boolean {
        val status = hardeningStatus
            ?: return false

        return status.password_rotated &&
                status.bootstrap_wiped &&
                status.device_verified &&
                status.recovery_backed_up
    }
}

/**
 * Hardening wizard UI states
 */
sealed class HardeningUiState {
    object NotStarted : HardeningUiState()
    object Loading : HardeningUiState()
    data class Loaded(val status: BridgeApi.HardeningStatus) : HardeningUiState()
    data class StepCompleted(val step: String) : HardeningUiState()
    data class Error(val message: String) : HardeningUiState()
    object AllComplete : HardeningUiState()
}

/**
 * Hardening steps in the wizard flow
 */
enum class HardeningStep {
    ROTATE_PASSWORD,
    WIPE_BOOTSTRAP,
    VERIFY_DEVICE,
    BACKUP_RECOVERY,
    ENABLE_BIOMETRICS,
    COMPLETE
}
