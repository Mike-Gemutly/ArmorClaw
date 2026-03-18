package com.armorclaw.app.components.offline

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.WifiOff
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

/**
 * Offline indicator banner that shows when the app is offline
 * Observes SyncStatusViewModel.isOnline and displays "No connection" message
 */
@Composable
fun OfflineIndicator(
    isOnline: StateFlow<Boolean>,
    modifier: Modifier = Modifier
) {
    val onlineState by isOnline.collectAsState()
    
    if (!onlineState) {
        Surface(
            modifier = modifier.fillMaxWidth(),
            color = MaterialTheme.colorScheme.errorContainer,
            contentColor = MaterialTheme.colorScheme.onErrorContainer
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Error icon with pulse animation
                Icon(
                    imageVector = Icons.Default.WifiOff,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onErrorContainer,
                    modifier = Modifier.size(24.dp)
                )

                // Error message
                Text(
                    text = "No connection",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1
                )
            }
        }
    }
}

/**
 * Preview for OfflineIndicator in offline state
 */
@Composable
fun OfflineIndicatorPreviewOffline() {
    MaterialTheme {
        val isOffline = MutableStateFlow(false)
        OfflineIndicator(
            isOnline = isOffline
        )
    }
}

/**
 * Preview for OfflineIndicator in online state
 */
@Composable
fun OfflineIndicatorPreviewOnline() {
    MaterialTheme {
        val isOnline = MutableStateFlow(true)
        OfflineIndicator(
            isOnline = isOnline
        )
    }
}