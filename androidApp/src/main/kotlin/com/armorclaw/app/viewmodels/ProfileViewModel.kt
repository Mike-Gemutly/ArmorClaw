package com.armorclaw.app.viewmodels

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserPresence
import com.armorclaw.shared.domain.repository.UserRepository
import com.armorclaw.shared.domain.usecase.LogoutUseCase
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.viewModelLogger
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * ViewModel for Profile screen
 *
 * Handles profile state management, editing, and logout.
 * State survives configuration changes.
 */
class ProfileViewModel(
    private val logoutUseCase: LogoutUseCase,
    private val userRepository: UserRepository
) : ViewModel() {

    private val logger = viewModelLogger("ProfileViewModel", LogTag.ViewModel.Profile)

    private val _uiState = MutableStateFlow(ProfileUiState())
    val uiState: StateFlow<ProfileUiState> = _uiState.asStateFlow()

    init {
        logger.logInit()
        loadProfile()
    }

    /**
     * Load user profile data
     */
    private fun loadProfile() {
        logger.logUserAction("loadProfile")
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }

            userRepository.getCurrentUser()
                .fold(
                    onSuccess = { user ->
                        if (user != null) {
                            logger.logUserAction("profile_loaded")
                            _uiState.update {
                                it.copy(
                                    isLoading = false,
                                    userName = user.displayName,
                                    userEmail = user.email ?: "",
                                    userStatus = user.presence.name.lowercase()
                                        .replaceFirstChar { it.uppercase() },
                                    userAvatar = user.avatar,
                                    error = null
                                )
                            }
                        } else {
                            _uiState.update {
                                it.copy(
                                    isLoading = false,
                                    error = "User not found"
                                )
                            }
                        }
                    },
                    onFailure = { error ->
                        logger.logError("loadProfile", error)
                        _uiState.update {
                            it.copy(
                                isLoading = false,
                                error = error.message ?: "Failed to load profile"
                            )
                        }
                    }
                )
        }
    }

    /**
     * Update profile field
     */
    fun updateName(name: String) {
        _uiState.update { it.copy(userName = name, isDirty = true) }
    }

    fun updateEmail(email: String) {
        _uiState.update { it.copy(userEmail = email, isDirty = true) }
    }

    fun updateStatus(status: String) {
        _uiState.update { it.copy(userStatus = status, isDirty = true) }
    }

    /**
     * Toggle edit mode
     */
    fun toggleEditMode() {
        val currentState = _uiState.value
        _uiState.update { it.copy(isEditing = !currentState.isEditing) }
        logger.logUserAction("toggleEditMode", mapOf("isEditing" to !currentState.isEditing))
    }

    /**
     * Save profile changes
     */
    fun saveProfile() {
        val state = _uiState.value
        logger.logUserAction("saveProfile")

        viewModelScope.launch {
            _uiState.update { it.copy(isSaving = true) }

            // Get current user and update with new values
            userRepository.getCurrentUser().fold(
                onSuccess = { currentUser ->
                    if (currentUser != null) {
                        val presence = when (state.userStatus.uppercase()) {
                            "ONLINE" -> UserPresence.ONLINE
                            "UNAVAILABLE", "AWAY", "BUSY" -> UserPresence.UNAVAILABLE
                            "OFFLINE", "INVISIBLE" -> UserPresence.OFFLINE
                            else -> UserPresence.UNKNOWN
                        }
                        val updatedUser = currentUser.copy(
                            displayName = state.userName,
                            email = state.userEmail.ifBlank { null },
                            presence = presence
                        )
                        userRepository.updateUser(updatedUser).fold(
                            onSuccess = {
                                logger.logUserAction("profile_saved")
                                _uiState.update {
                                    it.copy(
                                        isSaving = false,
                                        isEditing = false,
                                        isDirty = false,
                                        error = null
                                    )
                                }
                            },
                            onFailure = { error ->
                                logger.logError("saveProfile", error)
                                _uiState.update {
                                    it.copy(
                                        isSaving = false,
                                        error = error.message ?: "Failed to save profile"
                                    )
                                }
                            }
                        )
                    } else {
                        _uiState.update {
                            it.copy(
                                isSaving = false,
                                error = "User not found"
                            )
                        }
                    }
                },
                onFailure = { error ->
                    logger.logError("saveProfile", error)
                    _uiState.update {
                        it.copy(
                            isSaving = false,
                            error = error.message ?: "Failed to save profile"
                        )
                    }
                }
            )
        }
    }

    /**
     * Cancel editing and revert changes
     */
    fun cancelEditing() {
        logger.logUserAction("cancelEditing")
        loadProfile() // Reload to revert changes
        _uiState.update { it.copy(isEditing = false, isDirty = false) }
    }

    /**
     * Change avatar
     */
    fun changeAvatar() {
        logger.logUserAction("changeAvatar")
        // This would typically launch an image picker
        // For now, just log the action
    }

    /**
     * Log out the current user
     */
    fun logout() {
        if (_uiState.value.isLoggingOut) {
            logger.logUserAction("logout", mapOf("skipped" to "already_in_progress"))
            return
        }

        logger.logUserAction("logout")
        viewModelScope.launch {
            _uiState.update { it.copy(isLoggingOut = true) }

            val result = logoutUseCase(clearAllData = true)

            result.fold(
                onSuccess = {
                    logger.logUserAction("logout_success")
                    _uiState.update { it.copy(isLoggingOut = false, logoutSuccess = true) }
                },
                onFailure = { error ->
                    logger.logError("logout", error)
                    _uiState.update {
                        it.copy(
                            isLoggingOut = false,
                            error = error.message ?: "Failed to log out"
                        )
                    }
                }
            )
        }
    }

    /**
     * Reset error state
     */
    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }

    /**
     * Reset logout success state after navigation
     */
    fun resetLogoutState() {
        _uiState.update { it.copy(logoutSuccess = false) }
    }
}

/**
 * UI state for Profile screen
 */
data class ProfileUiState(
    val isLoading: Boolean = false,
    val isEditing: Boolean = false,
    val isSaving: Boolean = false,
    val isDirty: Boolean = false,
    val isLoggingOut: Boolean = false,
    val logoutSuccess: Boolean = false,
    val userName: String = "",
    val userEmail: String = "",
    val userStatus: String = "Available",
    val userAvatar: String? = null,
    val error: String? = null
)
