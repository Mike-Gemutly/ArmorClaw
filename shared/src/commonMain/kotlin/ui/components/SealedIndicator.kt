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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.KeystoreStatus
import com.armorclaw.shared.domain.model.UnsealMethod

/**
 * Sealed Indicator
 *
 * Displays the current sealed/unsealed state of the VPS keystore.
 * Shows status icon, remaining time (if unsealed), and action button.
 *
 * ## States
 * - Sealed: Shows lock icon with "Sealed" label and unlock button
 * - Unsealed: Shows unlock icon with remaining time and re-seal option
 * - Error: Shows warning icon with error message
 *
 * ## Usage
 * ```kotlin
 * val keystoreStatus by controlPlaneStore.keystoreStatus.collectAsState()
 *
 * SealedIndicator(
 *     status = keystoreStatus,
 *     onUnsealClick = { navController.navigate("unseal") },
 *     onResealClick = { viewModel.resealKeystore() }
 * )
 * ```
 */
@Composable
fun SealedIndicator(
    status: KeystoreStatus,
    onUnsealClick: () -> Unit,
    modifier: Modifier = Modifier,
    onResealClick: (() -> Unit)? = null,
    compact: Boolean = false
) {
    when (status) {
        is KeystoreStatus.Sealed -> {
            SealedStateIndicator(
                onUnsealClick = onUnsealClick,
                modifier = modifier,
                compact = compact
            )
        }
        is KeystoreStatus.Unsealed -> {
            UnsealedStateIndicator(
                status = status,
                onResealClick = onResealClick,
                modifier = modifier,
                compact = compact
            )
        }
        is KeystoreStatus.Error -> {
            ErrorStateIndicator(
                message = status.message,
                onRetryClick = onUnsealClick,
                modifier = modifier,
                compact = compact
            )
        }
    }
}

/**
 * Indicator for sealed keystore state
 */
@Composable
private fun SealedStateIndicator(
    onUnsealClick: () -> Unit,
    modifier: Modifier,
    compact: Boolean
) {
    Surface(
        onClick = onUnsealClick,
        modifier = modifier,
        shape = RoundedCornerShape(if (compact) 16.dp else 12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant
    ) {
        Row(
            modifier = Modifier.padding(
                horizontal = if (compact) 10.dp else 14.dp,
                vertical = if (compact) 6.dp else 10.dp
            ),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Lock,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(if (compact) 16.dp else 20.dp)
            )
            if (!compact) {
                Spacer(Modifier.width(8.dp))
                Text(
                    text = "Keystore Sealed",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Spacer(Modifier.width(8.dp))
                Surface(
                    color = MaterialTheme.colorScheme.primary,
                    shape = RoundedCornerShape(4.dp)
                ) {
                    Text(
                        text = "Unseal",
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimary
                    )
                }
            }
        }
    }
}

/**
 * Indicator for unsealed keystore state with remaining time
 */
@Composable
private fun UnsealedStateIndicator(
    status: KeystoreStatus.Unsealed,
    onResealClick: (() -> Unit)?,
    modifier: Modifier,
    compact: Boolean
) {
    val isExpiringSoon = status.remainingTimeMs() < (30 * 60 * 1000) // 30 minutes

    val containerColor = if (isExpiringSoon) {
        MaterialTheme.colorScheme.tertiaryContainer
    } else {
        MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f)
    }

    val contentColor = if (isExpiringSoon) {
        MaterialTheme.colorScheme.onTertiaryContainer
    } else {
        MaterialTheme.colorScheme.onPrimaryContainer
    }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(if (compact) 16.dp else 12.dp),
        color = containerColor
    ) {
        Row(
            modifier = Modifier.padding(
                horizontal = if (compact) 10.dp else 14.dp,
                vertical = if (compact) 6.dp else 10.dp
            ),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Animated indicator for expiring soon
            if (isExpiringSoon) {
                PulsingLockIcon(
                    color = contentColor,
                    modifier = Modifier.size(if (compact) 16.dp else 20.dp)
                )
            } else {
                Icon(
                    imageVector = Icons.Default.LockOpen,
                    contentDescription = null,
                    tint = contentColor,
                    modifier = Modifier.size(if (compact) 16.dp else 20.dp)
                )
            }

            if (!compact) {
                Spacer(Modifier.width(8.dp))
                Column {
                    Text(
                        text = "Unsealed",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = contentColor
                    )
                    Text(
                        text = status.remainingTimeString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = contentColor.copy(alpha = 0.7f)
                    )
                }

                // Method badge
                Spacer(Modifier.width(8.dp))
                Surface(
                    color = contentColor.copy(alpha = 0.15f),
                    shape = RoundedCornerShape(4.dp)
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 3.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = if (status.unsealedBy == UnsealMethod.BIOMETRIC) {
                                Icons.Default.Fingerprint
                            } else {
                                Icons.Default.Password
                            },
                            contentDescription = null,
                            modifier = Modifier.size(12.dp),
                            tint = contentColor
                        )
                        Spacer(Modifier.width(4.dp))
                        Text(
                            text = status.unsealedBy.toDisplayString(),
                            style = MaterialTheme.typography.labelSmall,
                            color = contentColor
                        )
                    }
                }

                // Reseal button
                if (onResealClick != null) {
                    Spacer(Modifier.weight(1f))
                    IconButton(
                        onClick = onResealClick,
                        modifier = Modifier.size(32.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Lock,
                            contentDescription = "Reseal",
                            tint = contentColor,
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            } else {
                Spacer(Modifier.width(6.dp))
                Text(
                    text = status.remainingTimeString(),
                    style = MaterialTheme.typography.labelSmall,
                    color = contentColor,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }
        }
    }
}

/**
 * Indicator for error state
 */
@Composable
private fun ErrorStateIndicator(
    message: String,
    onRetryClick: () -> Unit,
    modifier: Modifier,
    compact: Boolean
) {
    Surface(
        onClick = onRetryClick,
        modifier = modifier,
        shape = RoundedCornerShape(if (compact) 16.dp else 12.dp),
        color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier.padding(
                horizontal = if (compact) 10.dp else 14.dp,
                vertical = if (compact) 6.dp else 10.dp
            ),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Warning,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(if (compact) 16.dp else 20.dp)
            )
            if (!compact) {
                Spacer(Modifier.width(8.dp))
                Text(
                    text = "Keystore Error",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onErrorContainer
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    text = message.take(30),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onErrorContainer.copy(alpha = 0.7f),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f)
                )
                Surface(
                    color = MaterialTheme.colorScheme.error,
                    shape = RoundedCornerShape(4.dp)
                ) {
                    Text(
                        text = "Retry",
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onError
                    )
                }
            }
        }
    }
}

/**
 * Animated pulsing lock icon for expiring sessions
 */
@Composable
private fun PulsingLockIcon(
    color: Color,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.6f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Icon(
        imageVector = Icons.Default.LockOpen,
        contentDescription = null,
        tint = color.copy(alpha = alpha),
        modifier = modifier
    )
}

/**
 * Preview helper
 */
@Composable
fun SealedIndicatorPreview() {
    Column(
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        // Sealed state
        SealedIndicator(
            status = KeystoreStatus.Sealed(),
            onUnsealClick = {}
        )

        // Unsealed state
        SealedIndicator(
            status = KeystoreStatus.Unsealed(
                expiresAt = System.currentTimeMillis() + (4 * 60 * 60 * 1000),
                unsealedBy = UnsealMethod.BIOMETRIC
            ),
            onUnsealClick = {},
            onResealClick = {}
        )

        // Expiring soon
        SealedIndicator(
            status = KeystoreStatus.Unsealed(
                expiresAt = System.currentTimeMillis() + (15 * 60 * 1000),
                unsealedBy = UnsealMethod.PASSWORD
            ),
            onUnsealClick = {},
            onResealClick = {}
        )

        // Error state
        SealedIndicator(
            status = KeystoreStatus.Error("Failed to decrypt keystore"),
            onUnsealClick = {}
        )

        // Compact versions
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            SealedIndicator(
                status = KeystoreStatus.Sealed(),
                onUnsealClick = {},
                compact = true
            )
            SealedIndicator(
                status = KeystoreStatus.Unsealed(
                    expiresAt = System.currentTimeMillis() + (4 * 60 * 60 * 1000),
                    unsealedBy = UnsealMethod.BIOMETRIC
                ),
                onUnsealClick = {},
                compact = true
            )
        }
    }
}
