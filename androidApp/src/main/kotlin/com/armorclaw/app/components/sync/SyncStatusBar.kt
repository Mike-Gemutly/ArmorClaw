package com.armorclaw.app.components.sync
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.SyncState
import com.armorclaw.shared.ui.theme.*

/**
 * Visual sync status indicator showing current sync state
 * Displays in the header of main chat view or as a dedicated status bar
 *
 * States:
 * - Synced (Green): All messages encrypted and synced
 * - Syncing (Blue): Sending encrypted messages...
 * - Offline (Gray): No connection, will queue messages
 * - Conflict (Orange): Messages modified elsewhere, tap to resolve
 * - Error (Red): Connection or protocol error
 */
@Composable
fun SyncStatusBar(
    syncState: SyncState,
    lastSyncTime: Long? = null,
    queuedMessageCount: Int = 0,
    onRefreshClick: () -> Unit = {},
    onErrorClick: () -> Unit = {},
    modifier: Modifier = Modifier,
    expanded: Boolean = false
) {
    val config = getSyncStateConfig(syncState)

    val backgroundColor by animateColorAsState(
        targetValue = config.backgroundColor,
        animationSpec = tween(durationMillis = 300),
        label = "background_color"
    )

    val contentColor by animateColorAsState(
        targetValue = config.contentColor,
        animationSpec = tween(durationMillis = 300),
        label = "content_color"
    )

    Surface(
        modifier = modifier
            .fillMaxWidth()
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = {
                    when (syncState) {
                        is SyncState.Error -> onErrorClick()
                        else -> onRefreshClick()
                    }
                }
            ),
        color = backgroundColor,
        contentColor = contentColor
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Status icon with animation
            SyncStatusIcon(
                state = syncState,
                config = config,
                contentColor = contentColor
            )

            // Status text
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(2.dp)
            ) {
                Text(
                    text = config.label,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )

                // Show additional info based on state
                when (syncState) {
                    is SyncState.Success -> {
                        lastSyncTime?.let { time ->
                            Text(
                                text = "Last synced ${formatRelativeTime(time)}",
                                style = MaterialTheme.typography.bodySmall,
                                color = contentColor.copy(alpha = 0.7f),
                                maxLines = 1
                            )
                        }
                    }
                    is SyncState.Syncing -> {
                        Text(
                            text = "Please wait...",
                            style = MaterialTheme.typography.bodySmall,
                            color = contentColor.copy(alpha = 0.7f),
                            maxLines = 1
                        )
                    }
                    is SyncState.Offline -> {
                        if (queuedMessageCount > 0) {
                            Text(
                                text = "$queuedMessageCount message${if (queuedMessageCount != 1) "s" else ""} queued",
                                style = MaterialTheme.typography.bodySmall,
                                color = contentColor.copy(alpha = 0.7f),
                                maxLines = 1
                            )
                        } else {
                            Text(
                                text = "Will sync when connected",
                                style = MaterialTheme.typography.bodySmall,
                                color = contentColor.copy(alpha = 0.7f),
                                maxLines = 1
                            )
                        }
                    }
                    is SyncState.Error -> {
                        Text(
                            text = syncState.message,
                            style = MaterialTheme.typography.bodySmall,
                            color = contentColor.copy(alpha = 0.7f),
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                    else -> {
                        lastSyncTime?.let { time ->
                            Text(
                                text = "Last synced ${formatRelativeTime(time)}",
                                style = MaterialTheme.typography.bodySmall,
                                color = contentColor.copy(alpha = 0.7f),
                                maxLines = 1
                            )
                        }
                    }
                }
            }

            // Action button based on state
            when (syncState) {
                is SyncState.Syncing -> {
                    // Show cancel button during sync
                    IconButton(
                        onClick = onRefreshClick,
                        modifier = Modifier.size(32.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Cancel sync",
                            tint = contentColor,
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
                is SyncState.Error -> {
                    // Show retry button for errors
                    IconButton(
                        onClick = onErrorClick,
                        modifier = Modifier.size(32.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Refresh,
                            contentDescription = "Retry",
                            tint = contentColor,
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
                is SyncState.Offline -> {
                    // Show queued count badge
                    if (queuedMessageCount > 0) {
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = contentColor.copy(alpha = 0.2f)
                        ) {
                            Text(
                                text = queuedMessageCount.toString(),
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Medium,
                                color = contentColor,
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                            )
                        }
                    }
                }
                else -> {
                    // Show refresh button
                    IconButton(
                        onClick = onRefreshClick,
                        modifier = Modifier.size(32.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Refresh,
                            contentDescription = "Refresh",
                            tint = contentColor.copy(alpha = 0.7f),
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            }
        }
    }
}

/**
 * Animated sync status icon
 */
@Composable
private fun SyncStatusIcon(
    state: SyncState,
    config: SyncStateConfig,
    contentColor: Color
) {
    Box(
        modifier = Modifier.size(24.dp),
        contentAlignment = Alignment.Center
    ) {
        when (state) {
            is SyncState.Syncing -> {
                // Rotating sync icon
                val infiniteTransition = rememberInfiniteTransition(label = "sync_rotation")
                val rotation by infiniteTransition.animateFloat(
                    initialValue = 0f,
                    targetValue = 360f,
                    animationSpec = infiniteRepeatable(
                        animation = tween(1000, easing = LinearEasing),
                        repeatMode = RepeatMode.Restart
                    ),
                    label = "rotation"
                )

                Icon(
                    imageVector = Icons.Default.Refresh,
                    contentDescription = null,
                    tint = contentColor,
                    modifier = Modifier
                        .matchParentSize()
                        .rotate(rotation)
                )
            }
            else -> {
                // Static icon with pulse animation for errors
                val pulseScale = if (state is SyncState.Error) {
                    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
                    infiniteTransition.animateFloat(
                        initialValue = 1f,
                        targetValue = 1.1f,
                        animationSpec = infiniteRepeatable(
                            animation = tween(500, easing = FastOutSlowInEasing),
                            repeatMode = RepeatMode.Reverse
                        ),
                        label = "pulse"
                    ).value
                } else {
                    1f
                }

                Icon(
                    imageVector = config.icon,
                    contentDescription = null,
                    tint = contentColor,
                    modifier = Modifier
                        .matchParentSize()
                        .graphicsLayer {
                            scaleX = pulseScale
                            scaleY = pulseScale
                        }
                )
            }
        }
    }
}

/**
 * Compact sync indicator for use in headers/toolbars
 */
@Composable
fun SyncIndicatorCompact(
    syncState: SyncState,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val config = getSyncStateConfig(syncState)

    val indicatorColor by animateColorAsState(
        targetValue = config.indicatorColor,
        animationSpec = tween(durationMillis = 300),
        label = "indicator_color"
    )

    Box(
        modifier = modifier
            .size(12.dp)
            .clip(CircleShape)
            .background(indicatorColor)
            .semantics {
                contentDescription = config.label
                role = androidx.compose.ui.semantics.Role.Button
            }
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center
    ) {
        // Pulse animation for syncing state
        if (syncState is SyncState.Syncing) {
            val infiniteTransition = rememberInfiniteTransition(label = "pulse")
            val alpha by infiniteTransition.animateFloat(
                initialValue = 1f,
                targetValue = 0.4f,
                animationSpec = infiniteRepeatable(
                    animation = tween(500, easing = FastOutSlowInEasing),
                    repeatMode = RepeatMode.Reverse
                ),
                label = "alpha"
            )

            Box(
                modifier = Modifier
                    .matchParentSize()
                    .clip(CircleShape)
                    .background(indicatorColor.copy(alpha = alpha))
            )
        }
    }
}

/**
 * Sync status chip for use in lists/cards
 */
@Composable
fun SyncStatusChip(
    syncState: SyncState,
    modifier: Modifier = Modifier,
    onClick: () -> Unit = {}
) {
    val config = getSyncStateConfig(syncState)

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = config.backgroundColor,
        contentColor = config.contentColor,
        onClick = onClick
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(6.dp)
                    .clip(CircleShape)
                    .background(config.indicatorColor)
            )

            Text(
                text = config.shortLabel,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.Medium,
                maxLines = 1
            )
        }
    }
}

// ========== Configuration ==========

private data class SyncStateConfig(
    val label: String,
    val shortLabel: String,
    val icon: ImageVector,
    val backgroundColor: Color,
    val contentColor: Color,
    val indicatorColor: Color
)

@Composable
private fun getSyncStateConfig(state: SyncState): SyncStateConfig {
    return when (state) {
        is SyncState.Idle -> SyncStateConfig(
            label = "Ready",
            shortLabel = "Ready",
            icon = Icons.Default.Done,
            backgroundColor = BrandGreen.copy(alpha = 0.1f),
            contentColor = BrandGreenDark,
            indicatorColor = BrandGreen
        )
        is SyncState.Syncing -> SyncStateConfig(
            label = "Syncing...",
            shortLabel = "Syncing",
            icon = Icons.Default.Refresh,
            backgroundColor = BrandBlue.copy(alpha = 0.1f),
            contentColor = BrandBlueDark,
            indicatorColor = BrandBlue
        )
        is SyncState.Offline -> SyncStateConfig(
            label = "Offline",
            shortLabel = "Offline",
            icon = Icons.Default.Warning,
            backgroundColor = SyncOffline.copy(alpha = 0.1f),
            contentColor = SyncOffline,
            indicatorColor = SyncOffline
        )
        is SyncState.Error -> SyncStateConfig(
            label = if (state.isRecoverable) "Connection Error" else "Sync Failed",
            shortLabel = "Error",
            icon = Icons.Default.Warning,
            backgroundColor = BrandRed.copy(alpha = 0.1f),
            contentColor = BrandRedDark,
            indicatorColor = BrandRed
        )
        is SyncState.Success -> SyncStateConfig(
            label = "All synced",
            shortLabel = "Synced",
            icon = Icons.Default.Done,
            backgroundColor = BrandGreen.copy(alpha = 0.1f),
            contentColor = BrandGreenDark,
            indicatorColor = BrandGreen
        )
    }
}

// ========== Utility Functions ==========

private fun formatRelativeTime(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

    return when {
        diff < 60_000 -> "just now"
        diff < 3_600_000 -> {
            val minutes = diff / 60_000
            "$minutes minute${if (minutes != 1L) "s" else ""} ago"
        }
        diff < 86_400_000 -> {
            val hours = diff / 3_600_000
            "$hours hour${if (hours != 1L) "s" else ""} ago"
        }
        diff < 604_800_000 -> {
            val days = diff / 86_400_000
            "$days day${if (days != 1L) "s" else ""} ago"
        }
        else -> {
            val sdf = java.text.SimpleDateFormat("MMM d", java.util.Locale.getDefault())
            sdf.format(java.util.Date(timestamp))
        }
    }
}

// ========== Preview Composables ==========

@Composable
fun SyncStatusBarPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        SyncStatusBar(
            syncState = SyncState.Idle,
            lastSyncTime = System.currentTimeMillis() - 120_000,
            onRefreshClick = {}
        )
        SyncStatusBar(
            syncState = SyncState.Syncing,
            onRefreshClick = {}
        )
        SyncStatusBar(
            syncState = SyncState.Offline,
            queuedMessageCount = 3,
            onRefreshClick = {}
        )
        SyncStatusBar(
            syncState = SyncState.Error("Couldn't reach server", true),
            onErrorClick = {}
        )
        SyncStatusBar(
            syncState = SyncState.Success(5, 10),
            lastSyncTime = System.currentTimeMillis() - 30_000,
            onRefreshClick = {}
        )
    }
}
