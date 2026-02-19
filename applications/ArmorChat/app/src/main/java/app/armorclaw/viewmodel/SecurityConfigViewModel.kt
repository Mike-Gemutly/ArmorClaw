package app.armorclaw.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.armorclaw.network.BridgeApi
import app.armorclaw.repository.SetupRepository
import app.armorclaw.utils.BridgeError
import app.armorclaw.utils.ErrorHandler
import app.armorclaw.utils.retryWithBackoff
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * ViewModel for security configuration during setup
 */
class SecurityConfigViewModel(
    private val repository: SetupRepository = SetupRepository()
) : ViewModel() {

    // UI State
    private val _uiState = MutableStateFlow<SecurityConfigUiState>(SecurityConfigUiState.Loading)
    val uiState: StateFlow<SecurityConfigUiState> = _uiState.asStateFlow()

    // Categories
    private val _categories = MutableStateFlow<List<BridgeApi.DataCategory>>(emptyList())
    val categories: StateFlow<List<BridgeApi.DataCategory>> = _categories.asStateFlow()

    // Category permissions (local state before saving)
    private val _permissions = MutableStateFlow<Map<String, String>>(emptyMap())
    val permissions: StateFlow<Map<String, String>> = _permissions.asStateFlow()

    // Count of configured categories
    val configuredCount: Int
        get() = _permissions.value.count { it.value != "deny" }

    // Loading state for save operation
    private val _isSaving = MutableStateFlow(false)
    val isSaving: StateFlow<Boolean> = _isSaving.asStateFlow()

    // Current error
    private val _currentError = MutableStateFlow<BridgeError?>(null)
    val currentError: StateFlow<BridgeError?> = _currentError.asStateFlow()

    /**
     * Load security categories
     */
    fun loadCategories() {
        viewModelScope.launch {
            _uiState.value = SecurityConfigUiState.Loading
            _currentError.value = null

            try {
                val categoryList = retryWithBackoff(maxAttempts = 3) {
                    repository.getSecurityCategories().getOrThrow()
                }

                _categories.value = categoryList
                // Initialize permissions from loaded data
                _permissions.value = categoryList.associate { it.id to it.permission }
                _uiState.value = SecurityConfigUiState.Loaded
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _currentError.value = error
                _uiState.value = SecurityConfigUiState.BridgeError(error)
            }
        }
    }

    /**
     * Update permission for a category
     */
    fun setPermission(categoryId: String, permission: String) {
        _permissions.value = _permissions.value + (categoryId to permission)
    }

    /**
     * Get permission for a category
     */
    fun getPermission(categoryId: String): String {
        return _permissions.value[categoryId] ?: "deny"
    }

    /**
     * Save all category permissions
     */
    fun savePermissions(onComplete: (Boolean) -> Unit) {
        viewModelScope.launch {
            _isSaving.value = true
            _currentError.value = null

            try {
                for ((categoryId, permission) in _permissions.value) {
                    retryWithBackoff(maxAttempts = 2) {
                        repository.setCategoryPermission(categoryId, permission).getOrThrow()
                    }
                }
                onComplete(true)
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _currentError.value = error
                onComplete(false)
            } finally {
                _isSaving.value = false
            }
        }
    }

    /**
     * Complete setup and transition to operational mode
     */
    fun completeSetup(onComplete: (Boolean) -> Unit) {
        viewModelScope.launch {
            _isSaving.value = true
            _currentError.value = null

            try {
                retryWithBackoff(maxAttempts = 2) {
                    repository.completeSetup().getOrThrow()
                }
                onComplete(true)
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _currentError.value = error
                onComplete(false)
            } finally {
                _isSaving.value = false
            }
        }
    }

    /**
     * Reset to defaults
     */
    fun resetToDefaults() {
        _permissions.value = _categories.value.associate { it.id to "deny" }
    }

    /**
     * Clear current error
     */
    fun clearError() {
        _currentError.value = null
        if (_uiState.value is SecurityConfigUiState.BridgeError) {
            _uiState.value = SecurityConfigUiState.Loaded
        }
    }
}

/**
 * UI state for security configuration
 */
sealed class SecurityConfigUiState {
    object Loading : SecurityConfigUiState()
    object Loaded : SecurityConfigUiState()
    data class BridgeError(val error: BridgeError) : SecurityConfigUiState()
}
