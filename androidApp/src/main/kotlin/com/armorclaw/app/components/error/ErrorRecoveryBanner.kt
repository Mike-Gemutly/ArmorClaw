package com.armorclaw.app.components.error

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Snackbar
import androidx.compose.material3.SnackbarData
import androidx.compose.material3.SnackbarDuration
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SnackbarResult
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.base.BaseViewModel
import com.armorclaw.shared.ui.base.UiEvent as BaseUiEvent
import kotlinx.coroutines.launch

/**
 * Error recovery banner that displays network errors with retry functionality.
 * Integrates with the existing BaseViewModel.UiEvent.ShowError pattern.
 *
 * @param viewModel The ViewModel that emits error events
 * @param onRetry Callback when retry button is clicked
 * @param modifier Modifier for the snackbar
 */
@Composable
fun ErrorRecoveryBanner(
    viewModel: BaseViewModel,
    onRetry: () -> Unit,
    modifier: Modifier = Modifier
) {
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()
    
    // Collect error events from the ViewModel
    LaunchedEffect(Unit) {
        viewModel.events.collect { event ->
            when (event) {
                is BaseUiEvent.ShowError -> {
                    scope.launch {
                        val result = snackbarHostState.showSnackbar(
                            message = "Network error",
                            actionLabel = "Retry",
                            duration = SnackbarDuration.Indefinite
                        )

                        if (result == SnackbarResult.ActionPerformed) {
                            onRetry()
                        }
                    }
                }
                else -> Unit
            }
        }
    }
    
    SnackbarHost(
        hostState = snackbarHostState,
        modifier = modifier
    )
}

/**
 * Custom Snackbar with improved styling for error messages
 */
@Composable
fun ErrorSnackbar(
    snackbarData: SnackbarData,
    onRetry: () -> Unit,
    modifier: Modifier = Modifier
) {
    Snackbar(
        modifier = modifier.padding(horizontal = 16.dp, vertical = 8.dp),
        action = {
            Text(
                text = "Retry",
                modifier = Modifier.clickable { onRetry() }
            )
        }
    ) {
        Text("Network error")
    }
}

/**
 * Preview for ErrorRecoveryBanner component
 */
@Composable
@Preview
fun ErrorRecoveryBannerPreview() {
    // Create a mock ViewModel for preview
    val mockViewModel = object : BaseViewModel() {
        override fun onCleared() {}
    }
    
    ErrorRecoveryBanner(
        viewModel = mockViewModel,
        onRetry = {}
    )
}
