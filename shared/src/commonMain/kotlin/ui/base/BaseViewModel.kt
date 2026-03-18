package com.armorclaw.shared.ui.base

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.viewModelLogger
import kotlinx.coroutines.CoroutineExceptionHandler
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.plus

/**
 * Base ViewModel with built-in logging and error handling.
 *
 * Provides:
 * - ViewModelLogger for proper separation of concerns
 * - Coroutine exception handling with logging
 * - UI state management
 * - Event emission
 *
 * All ViewModels should extend this class to ensure consistent logging.
 */
abstract class BaseViewModel : ViewModel() {

    private val logger = viewModelLogger(
        viewModelName = this::class.simpleName ?: "BaseViewModel",
        tag = LogTag.ViewModel.Settings
    )

    private val _uiState = MutableStateFlow<UiState>(UiState.Idle)
    val uiState: StateFlow<UiState> = _uiState.asStateFlow()

    private val _events = MutableSharedFlow<UiEvent>()
    val events: SharedFlow<UiEvent> = _events.asSharedFlow()

    private val errorHandler = CoroutineExceptionHandler { _, throwable ->
        logger.logError("unhandledException", throwable)
        viewModelScope.launch {
            _uiState.value = UiState.Error(throwable.message)
            _events.emit(UiEvent.ShowError(throwable.message ?: "Unknown error"))
        }
    }

    protected val defaultScope = viewModelScope + errorHandler

    init {
        logger.logInit()
    }

    protected fun setLoading() {
        logger.logStateChange("uiState", "Loading")
        _uiState.value = UiState.Loading
    }

    protected fun setIdle() {
        logger.logStateChange("uiState", "Idle")
        _uiState.value = UiState.Idle
    }

    protected fun setError(error: String?) {
        logger.logStateChange("uiState", "Error: $error")
        _uiState.value = UiState.Error(error)
    }

    protected suspend fun emitEvent(event: UiEvent) {
        logger.logUiEvent(event::class.simpleName ?: "UnknownEvent")
        _events.emit(event)
    }

    protected suspend fun <T> execute(
        operation: String,
        loading: Boolean = true,
        block: suspend () -> T
    ): Result<T> {
        return try {
            logger.logUserAction(operation)
            if (loading) setLoading()
            val result = block()
            logger.logStateChange("operation", "$operation success")
            Result.success(result)
        } catch (e: Exception) {
            logger.logError(operation, e)
            setError(e.message)
            Result.failure(e)
        } finally {
            if (loading) setIdle()
        }
    }

    protected fun <T> executeFlow(
        operation: String,
        loading: Boolean = true,
        block: suspend () -> T
    ) = defaultScope.launch {
        try {
            logger.logUserAction(operation)
            if (loading) setLoading()
            block()
            logger.logStateChange("operation", "$operation success")
        } catch (e: Exception) {
            logger.logError(operation, e)
            setError(e.message)
        } finally {
            if (loading) setIdle()
        }
    }

    override fun onCleared() {
        logger.logCleanup()
        super.onCleared()
    }
}

sealed class UiState {
    object Idle : UiState()
    object Loading : UiState()
    data class Error(val message: String?) : UiState()
}

sealed class UiEvent {
    data class ShowError(val message: String) : UiEvent()
    data class ShowSuccess(val message: String) : UiEvent()
    data class ShowInfo(val message: String) : UiEvent()
    data class NavigateTo(val route: String, val data: Map<String, Any?>? = null) : UiEvent()
    data class NavigateBack(val result: Map<String, Any?>? = null) : UiEvent()
    object ShowLoading : UiEvent()
    object HideLoading : UiEvent()
    
    // Unified Chat Events (Phase 4)
    data class CopyToClipboard(val text: String) : UiEvent()
    data class FocusInput(val text: String = "") : UiEvent()
    data class Custom(val type: String, val data: Map<String, Any?> = emptyMap()) : UiEvent()

    // UX feedback (Fix 4: feature suppression feedback)
    data class ShowSnackbar(val message: String) : UiEvent()
}
