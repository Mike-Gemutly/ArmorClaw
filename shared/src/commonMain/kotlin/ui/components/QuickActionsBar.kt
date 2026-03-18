package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.QuickAction
import com.armorclaw.shared.ui.theme.AppIcons

/**
 * Quick Actions Bar
 *
 * Displays emergency controls for the Mission Control Dashboard.
 * Provides one-tap access to critical actions like emergency stop, pause all, and lock vault.
 *
 * ## Architecture
 * ```
 * QuickActionsBar
 *      └── Row of QuickActionButton[]
 *          ├── Emergency Stop (red, destructive)
 *          ├── Pause/Resume All (orange)
 *          └── Lock Vault (blue)
 * ```
 *
 * ## Usage
 * ```kotlin
 * QuickActionsBar(
 *     isPaused = viewModel.isPaused,
 *     isVaultLocked = keystoreStatus is KeystoreStatus.Sealed,
 *     onEmergencyStop = { viewModel.emergencyStop() },
 *     onPauseAll = { viewModel.pauseAllAgents() },
 *     onResumeAll = { viewModel.resumeAllAgents() },
 *     onLockVault = { viewModel.lockVault() },
 *     modifier = Modifier.padding(horizontal = 16.dp)
 * )
 * ```
 */
@Composable
fun QuickActionsBar(
    isPaused: Boolean,
    isVaultLocked: Boolean,
    hasActiveAgents: Boolean,
    onEmergencyStop: () -> Unit,
    onPauseAll: () -> Unit,
    onResumeAll: () -> Unit,
    onLockVault: () -> Unit,
    modifier: Modifier = Modifier,
    emergencyStopConfirmRequired: Boolean = true
) {
    var showEmergencyConfirm by remember { mutableStateOf(false) }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Emergency Stop - always visible and prominent
            EmergencyStopButton(
                onClick = {
                    if (emergencyStopConfirmRequired && hasActiveAgents) {
                        showEmergencyConfirm = true
                    } else {
                        onEmergencyStop()
                    }
                },
                enabled = hasActiveAgents,
                modifier = Modifier.weight(1f)
            )

            // Pause/Resume All
            if (isPaused) {
                QuickActionButton(
                    icon = Icons.Default.PlayArrow,
                    label = "Resume All",
                    onClick = onResumeAll,
                    color = MaterialTheme.colorScheme.tertiary,
                    contentColor = MaterialTheme.colorScheme.onTertiary,
                    enabled = hasActiveAgents,
                    modifier = Modifier.weight(1f)
                )
            } else {
                QuickActionButton(
                    icon = Icons.Default.Pause,
                    label = "Pause All",
                    onClick = onPauseAll,
                    color = MaterialTheme.colorScheme.secondary,
                    contentColor = MaterialTheme.colorScheme.onSecondary,
                    enabled = hasActiveAgents,
                    modifier = Modifier.weight(1f)
                )
            }

            // Lock Vault
            QuickActionButton(
                icon = if (isVaultLocked) Icons.Default.Lock else Icons.Default.LockOpen,
                label = if (isVaultLocked) "Vault Locked" else "Lock Vault",
                onClick = onLockVault,
                color = if (isVaultLocked) {
                    MaterialTheme.colorScheme.outline
                } else {
                    MaterialTheme.colorScheme.primary
                },
                contentColor = if (isVaultLocked) {
                    MaterialTheme.colorScheme.onSurface
                } else {
                    MaterialTheme.colorScheme.onPrimary
                },
                enabled = !isVaultLocked,
                modifier = Modifier.weight(1f)
            )
        }
    }

    // Emergency stop confirmation dialog
    if (showEmergencyConfirm) {
        EmergencyStopConfirmDialog(
            onConfirm = {
                showEmergencyConfirm = false
                onEmergencyStop()
            },
            onDismiss = { showEmergencyConfirm = false }
        )
    }
}

/**
 * Emergency Stop Button - visually distinct and prominent
 */
@Composable
private fun EmergencyStopButton(
    onClick: () -> Unit,
    enabled: Boolean,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "emergency")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.8f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Button(
        onClick = onClick,
        modifier = modifier.height(48.dp),
        enabled = enabled,
        shape = RoundedCornerShape(12.dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = MaterialTheme.colorScheme.error.copy(alpha = if (enabled) alpha else 0.5f),
            contentColor = MaterialTheme.colorScheme.onError,
            disabledContainerColor = MaterialTheme.colorScheme.error.copy(alpha = 0.3f),
            disabledContentColor = MaterialTheme.colorScheme.onError.copy(alpha = 0.5f)
        ),
        elevation = ButtonDefaults.buttonElevation(
            defaultElevation = if (enabled) 4.dp else 0.dp
        )
    ) {
        Icon(
            imageVector = Icons.Default.Stop,
            contentDescription = null,
            modifier = Modifier.size(20.dp)
        )
        Spacer(modifier = Modifier.width(6.dp))
        Text(
            text = "STOP",
            style = MaterialTheme.typography.labelLarge,
            fontWeight = FontWeight.Bold
        )
    }
}

/**
 * Generic quick action button
 */
@Composable
private fun QuickActionButton(
    icon: ImageVector,
    label: String,
    onClick: () -> Unit,
    color: androidx.compose.ui.graphics.Color,
    contentColor: androidx.compose.ui.graphics.Color,
    enabled: Boolean,
    modifier: Modifier = Modifier
) {
    FilledTonalButton(
        onClick = onClick,
        modifier = modifier.height(48.dp),
        enabled = enabled,
        shape = RoundedCornerShape(12.dp),
        colors = ButtonDefaults.filledTonalButtonColors(
            containerColor = color.copy(alpha = if (enabled) 0.2f else 0.1f),
            contentColor = contentColor,
            disabledContainerColor = color.copy(alpha = 0.1f),
            disabledContentColor = contentColor.copy(alpha = 0.5f)
        )
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.padding(horizontal = 4.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                modifier = Modifier.size(20.dp)
            )
            Spacer(modifier = Modifier.height(2.dp))
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.Medium,
                textAlign = TextAlign.Center,
                maxLines = 1
            )
        }
    }
}

/**
 * Emergency stop confirmation dialog
 */
@Composable
private fun EmergencyStopConfirmDialog(
    onConfirm: () -> Unit,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        icon = {
            Icon(
                imageVector = Icons.Default.Warning,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(32.dp)
            )
        },
        title = {
            Text(
                text = "Emergency Stop?",
                fontWeight = FontWeight.Bold
            )
        },
        text = {
            Column {
                Text(
                    text = "This will immediately stop all active agents and cancel pending tasks.",
                    style = MaterialTheme.typography.bodyMedium
                )
                Spacer(modifier = Modifier.height(12.dp))
                Text(
                    text = "Any in-progress operations (payments, form submissions) will be interrupted.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error
                )
            }
        },
        confirmButton = {
            Button(
                onClick = onConfirm,
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.error
                )
            ) {
                Text("Stop All")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

/**
 * Compact quick action bar for smaller screens
 */
@Composable
fun CompactQuickActionsBar(
    hasActiveAgents: Boolean,
    isPaused: Boolean,
    onEmergencyStop: () -> Unit,
    onPauseResume: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // Emergency stop
        FilledIconButton(
            onClick = onEmergencyStop,
            enabled = hasActiveAgents,
            modifier = Modifier.size(48.dp),
            colors = IconButtonDefaults.filledIconButtonColors(
                containerColor = MaterialTheme.colorScheme.error,
                contentColor = MaterialTheme.colorScheme.onError,
                disabledContainerColor = MaterialTheme.colorScheme.error.copy(alpha = 0.3f),
                disabledContentColor = MaterialTheme.colorScheme.onError.copy(alpha = 0.5f)
            )
        ) {
            Icon(
                imageVector = Icons.Default.Stop,
                contentDescription = "Emergency Stop",
                modifier = Modifier.size(24.dp)
            )
        }

        // Pause/Resume
        FilledIconButton(
            onClick = onPauseResume,
            enabled = hasActiveAgents,
            modifier = Modifier.size(48.dp),
            colors = IconButtonDefaults.filledIconButtonColors(
                containerColor = if (isPaused) {
                    MaterialTheme.colorScheme.tertiary
                } else {
                    MaterialTheme.colorScheme.secondary
                },
                contentColor = if (isPaused) {
                    MaterialTheme.colorScheme.onTertiary
                } else {
                    MaterialTheme.colorScheme.onSecondary
                }
            )
        ) {
            Icon(
                imageVector = if (isPaused) Icons.Default.PlayArrow else Icons.Default.Pause,
                contentDescription = if (isPaused) "Resume All" else "Pause All",
                modifier = Modifier.size(24.dp)
            )
        }
    }
}

/**
 * Preview helper
 */
@Composable
fun QuickActionsBarPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        // Active state
        QuickActionsBar(
            isPaused = false,
            isVaultLocked = false,
            hasActiveAgents = true,
            onEmergencyStop = {},
            onPauseAll = {},
            onResumeAll = {},
            onLockVault = {}
        )

        // Paused state
        QuickActionsBar(
            isPaused = true,
            isVaultLocked = false,
            hasActiveAgents = true,
            onEmergencyStop = {},
            onPauseAll = {},
            onResumeAll = {},
            onLockVault = {}
        )

        // Locked vault
        QuickActionsBar(
            isPaused = false,
            isVaultLocked = true,
            hasActiveAgents = false,
            onEmergencyStop = {},
            onPauseAll = {},
            onResumeAll = {},
            onLockVault = {}
        )

        // Compact version
        CompactQuickActionsBar(
            hasActiveAgents = true,
            isPaused = false,
            onEmergencyStop = {},
            onPauseResume = {}
        )
    }
}
