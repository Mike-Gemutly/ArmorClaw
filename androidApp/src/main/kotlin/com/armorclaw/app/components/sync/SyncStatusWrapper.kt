package com.armorclaw.app.components.sync

import androidx.compose.foundation.layout.Column
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.armorclaw.app.components.offline.OfflineIndicator
import com.armorclaw.app.viewmodels.SyncStatusViewModel

/**
 * SyncStatusWrapper displays offline indicator when network is unavailable.
 *
 * @param viewModel The SyncStatusViewModel providing sync and connection state
 * @param onRetry Callback invoked when user taps retry (unused but kept for API compatibility)
 * @param modifier Modifier for the wrapper component
 */
@Composable
fun SyncStatusWrapper(
    viewModel: SyncStatusViewModel,
    onRetry: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier) {
        OfflineIndicator(
            isOnline = viewModel.isOnline,
            modifier = Modifier
        )
    }
}
