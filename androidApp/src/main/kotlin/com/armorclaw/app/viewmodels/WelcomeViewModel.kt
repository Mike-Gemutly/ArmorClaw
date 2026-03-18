package com.armorclaw.app.viewmodels

import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.ui.base.BaseViewModel
import com.armorclaw.shared.ui.base.UiEvent
import com.armorclaw.shared.ui.base.UiState
import kotlinx.coroutines.launch

class WelcomeViewModel : BaseViewModel() {
    
    init {
        checkOnboardingStatus()
    }
    
    fun onGetStarted() {
        viewModelScope.launch {
            emitEvent(UiEvent.NavigateTo(route = "security"))
        }
    }
    
    fun onSkip() {
        viewModelScope.launch {
            // Save onboarding as skipped
            emitEvent(UiEvent.NavigateTo(route = "home"))
        }
    }
    
    private fun checkOnboardingStatus() {
        viewModelScope.launch {
            // Check if onboarding has been completed
            // This would read from local storage
        }
    }
}
