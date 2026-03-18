package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.AgentTaskStatus
import com.armorclaw.shared.domain.model.AttentionPriority
import com.armorclaw.shared.domain.model.KeystoreStatus
import com.armorclaw.shared.ui.theme.AppIcons

/**
 * Mission Control Header
 *
 * Displays status summary and greeting at the top of the Mission Control Dashboard.
 * Shows vault status, active agent count, and attention queue size.
 *
 * ## Architecture
 * ```
 * MissionControlHeader
 *      ├── Greeting Section
 *      │   ├── Time-based greeting
 *      │   └── Vault status indicator
 *      └── Status Pills
 *          ├── Active agents count
 *          ├── Attention queue count
 *          └── Vault status
 * ```
 *
 * ## Usage
 * ```kotlin
 * MissionControlHeader(
 *     vaultStatus = keystoreStatus,
 *     activeAgentCount = activeAgents.size,
 *     attentionCount = attentionQueue.size,
 *     modifier = Modifier.padding(horizontal = 16.dp)
 * )
 * ```
 */
@Composable
fun MissionControlHeader(
    vaultStatus: KeystoreStatus,
    activeAgentCount: Int,
    attentionCount: Int,
    highestPriority: AttentionPriority?,
    modifier: Modifier = Modifier,
    userName: String? = null
) {
    val greeting = remember { getGreeting() }

    Column(
        modifier = modifier.fillMaxWidth()
    ) {
        // Main header row
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Greeting and vault status
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = if (userName != null) "$greeting, $userName" else greeting,
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold
                )

                Spacer(modifier = Modifier.height(4.dp))

                // Vault status row
                Row(
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    VaultStatusIndicator(
                        status = vaultStatus,
                        compact = true
                    )

                    if (activeAgentCount > 0) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = MaterialTheme.colorScheme.primaryContainer
                        ) {
                            Row(
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Icon(
                                    imageVector = Icons.Default.SmartToy,
                                    contentDescription = null,
                                    tint = MaterialTheme.colorScheme.onPrimaryContainer,
                                    modifier = Modifier.size(14.dp)
                                )
                                Spacer(modifier = Modifier.width(4.dp))
                                Text(
                                    text = "$activeAgentCount active",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onPrimaryContainer
                                )
                            }
                        }
                    }
                }
            }

            // Attention indicator
            if (attentionCount > 0) {
                AttentionBadge(
                    count = attentionCount,
                    priority = highestPriority ?: AttentionPriority.MEDIUM
                )
            }
        }

        // Status summary bar
        if (attentionCount > 0 || activeAgentCount > 0) {
            Spacer(modifier = Modifier.height(8.dp))
            StatusSummaryBar(
                vaultStatus = vaultStatus,
                activeAgentCount = activeAgentCount,
                attentionCount = attentionCount
            )
        }
    }
}

/**
 * Vault Status Indicator
 *
 * Displays the current vault (keystore) status with visual indicator.
 * Used in both Mission Control and other screens.
 */
@Composable
fun VaultStatusIndicator(
    status: KeystoreStatus,
    modifier: Modifier = Modifier,
    compact: Boolean = false
) {
    val (icon, text, color) = when (status) {
        is KeystoreStatus.Sealed -> Triple(
            Icons.Default.Lock,
            "Vault Sealed",
            MaterialTheme.colorScheme.outline
        )
        is KeystoreStatus.Unsealed -> Triple(
            Icons.Default.LockOpen,
            "Vault Unsealed",
            MaterialTheme.colorScheme.primary
        )
        is KeystoreStatus.Error -> Triple(
            AppIcons.Error,
            "Vault Error",
            MaterialTheme.colorScheme.error
        )
    }

    // Animated pulse for unsealed state
    val infiniteTransition = rememberInfiniteTransition(label = "vault_pulse")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.7f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(1000, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(if (compact) 12.dp else 8.dp),
        color = color.copy(alpha = 0.15f)
    ) {
        Row(
            modifier = Modifier.padding(
                horizontal = if (compact) 8.dp else 12.dp,
                vertical = if (compact) 4.dp else 6.dp
            ),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = if (status is KeystoreStatus.Unsealed) color.copy(alpha = alpha) else color,
                modifier = Modifier.size(if (compact) 14.dp else 18.dp)
            )
            if (!compact) {
                Spacer(modifier = Modifier.width(6.dp))
                Text(
                    text = text,
                    style = MaterialTheme.typography.labelMedium,
                    color = color,
                    fontWeight = FontWeight.Medium
                )
            }
        }
    }
}

/**
 * Attention badge with priority color
 */
@Composable
private fun AttentionBadge(
    count: Int,
    priority: AttentionPriority,
    modifier: Modifier = Modifier
) {
    val color = when (priority) {
        AttentionPriority.CRITICAL -> MaterialTheme.colorScheme.error
        AttentionPriority.HIGH -> MaterialTheme.colorScheme.tertiary
        AttentionPriority.MEDIUM -> MaterialTheme.colorScheme.secondary
        AttentionPriority.LOW -> MaterialTheme.colorScheme.outline
    }

    // Pulse animation for critical items
    val infiniteTransition = rememberInfiniteTransition(label = "attention_pulse")
    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (priority == AttentionPriority.CRITICAL) 1.1f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    Surface(
        modifier = modifier
            .clip(RoundedCornerShape(16.dp))
            .graphicsLayer { scaleX = scale; scaleY = scale },
        color = color,
        shape = RoundedCornerShape(16.dp)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.NotificationsActive,
                contentDescription = null,
                tint = Color.White,
                modifier = Modifier.size(18.dp)
            )
            Spacer(modifier = Modifier.width(6.dp))
            Text(
                text = if (count > 99) "99+" else count.toString(),
                style = MaterialTheme.typography.labelLarge,
                color = Color.White,
                fontWeight = FontWeight.Bold
            )
        }
    }
}

/**
 * Status summary bar with visual progress indicators
 */
@Composable
private fun StatusSummaryBar(
    vaultStatus: KeystoreStatus,
    activeAgentCount: Int,
    attentionCount: Int,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Vault indicator
            StatusBarItem(
                icon = when (vaultStatus) {
                    is KeystoreStatus.Sealed -> Icons.Default.Lock
                    is KeystoreStatus.Unsealed -> Icons.Default.LockOpen
                    is KeystoreStatus.Error -> AppIcons.Error
                },
                label = "Vault",
                value = when (vaultStatus) {
                    is KeystoreStatus.Sealed -> "Sealed"
                    is KeystoreStatus.Unsealed -> "Open"
                    is KeystoreStatus.Error -> "Error"
                },
                color = when (vaultStatus) {
                    is KeystoreStatus.Sealed -> MaterialTheme.colorScheme.outline
                    is KeystoreStatus.Unsealed -> MaterialTheme.colorScheme.primary
                    is KeystoreStatus.Error -> MaterialTheme.colorScheme.error
                }
            )

            Box(
                modifier = Modifier
                    .height(24.dp)
                    .width(1.dp)
                    .padding(vertical = 4.dp)
                    .background(MaterialTheme.colorScheme.outline.copy(alpha = 0.3f))
            )

            // Active agents
            if (activeAgentCount > 0) {
                StatusBarItem(
                    icon = Icons.Default.SmartToy,
                    label = "Agents",
                    value = activeAgentCount.toString(),
                    color = MaterialTheme.colorScheme.primary
                )

                Box(
                    modifier = Modifier
                        .height(24.dp)
                        .width(1.dp)
                        .padding(vertical = 4.dp)
                        .background(MaterialTheme.colorScheme.outline.copy(alpha = 0.3f))
                )
            }

            // Attention queue
            if (attentionCount > 0) {
                StatusBarItem(
                    icon = Icons.Default.PriorityHigh,
                    label = "Needs You",
                    value = attentionCount.toString(),
                    color = MaterialTheme.colorScheme.error
                )
            }
        }
    }
}

/**
 * Individual status bar item
 */
@Composable
private fun StatusBarItem(
    icon: ImageVector,
    label: String,
    value: String,
    color: Color,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = color,
            modifier = Modifier.size(18.dp)
        )
        Column {
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Text(
                text = value,
                style = MaterialTheme.typography.labelMedium,
                color = color,
                fontWeight = FontWeight.SemiBold
            )
        }
    }
}

/**
 * Get time-based greeting
 */
private fun getGreeting(): String {
    val hour = java.util.Calendar.getInstance().get(java.util.Calendar.HOUR_OF_DAY)
    return when (hour) {
        in 5..11 -> "Good morning"
        in 12..16 -> "Good afternoon"
        in 17..20 -> "Good evening"
        else -> "Hello"
    }
}

/**
 * Preview helper
 */
@Composable
fun MissionControlHeaderPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        // Empty state
        MissionControlHeader(
            vaultStatus = KeystoreStatus.Sealed(),
            activeAgentCount = 0,
            attentionCount = 0,
            highestPriority = null,
            userName = "Alex"
        )

        // Active state
        MissionControlHeader(
            vaultStatus = KeystoreStatus.Unsealed(
                expiresAt = System.currentTimeMillis() + 3600000,
                unsealedBy = com.armorclaw.shared.domain.model.UnsealMethod.BIOMETRIC
            ),
            activeAgentCount = 3,
            attentionCount = 2,
            highestPriority = AttentionPriority.HIGH,
            userName = "Alex"
        )

        // Critical attention
        MissionControlHeader(
            vaultStatus = KeystoreStatus.Unsealed(
                expiresAt = System.currentTimeMillis() + 3600000,
                unsealedBy = com.armorclaw.shared.domain.model.UnsealMethod.PASSWORD
            ),
            activeAgentCount = 1,
            attentionCount = 5,
            highestPriority = AttentionPriority.CRITICAL
        )
    }
}
