package com.armorclaw.app.screens.home.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Governance severity levels for enterprise banners
 */
enum class GovernanceSeverity {
    INFO,
    WARNING,
    CRITICAL
}

/**
 * Governance event data for banner display
 */
data class GovernanceEvent(
    val id: String,
    val severity: GovernanceSeverity,
    val title: String,
    val message: String,
    val actionLabel: String? = null,
    val onAction: (() -> Unit)? = null
)

/**
 * Dismissable governance banner for the HomeScreen
 *
 * Displays enterprise governance events such as:
 * - License expiry warnings
 * - Compliance policy notifications
 * - Server administration alerts
 *
 * ## Dismissal Behaviour (Fix 7)
 * - INFO: dismiss permanently (until next event)
 * - WARNING: dismiss permanently
 * - CRITICAL: dismiss for 24 hours, then re-shows
 * - License Expired (id starts with "license-expired"): cannot be dismissed
 *
 * Usage:
 * ```
 * GovernanceBanner(
 *     event = GovernanceEvent(
 *         id = "license-expiry",
 *         severity = GovernanceSeverity.WARNING,
 *         title = "License Expiring",
 *         message = "Your ArmorClaw license expires in 7 days.",
 *         actionLabel = "Renew",
 *         onAction = { /* navigate to renewal */ }
 *     ),
 *     onDismiss = { /* mark dismissed */ }
 * )
 * ```
 */
@Composable
fun GovernanceBanner(
    event: GovernanceEvent,
    onDismiss: (String) -> Unit,
    isDismissed: Boolean = false,
    modifier: Modifier = Modifier
) {
    // License Expired events cannot be dismissed at all
    val isLicenseExpired = event.id.startsWith("license-expired")
    val canDismiss = !isLicenseExpired
    var visible by remember { mutableStateOf(!isDismissed) }

    AnimatedVisibility(
        visible = visible,
        enter = expandVertically(),
        exit = shrinkVertically()
    ) {
        val (containerColor, iconColor, icon) = when (event.severity) {
            GovernanceSeverity.INFO -> Triple(
                Color(0xFF1A237E).copy(alpha = 0.12f),
                Color(0xFF1565C0),
                Icons.Default.Info
            )
            GovernanceSeverity.WARNING -> Triple(
                Color(0xFFF57F17).copy(alpha = 0.12f),
                Color(0xFFF57F17),
                Icons.Default.Warning
            )
            GovernanceSeverity.CRITICAL -> Triple(
                Color(0xFFB71C1C).copy(alpha = 0.12f),
                Color(0xFFD32F2F),
                Icons.Default.Warning
            )
        }

        Card(
            modifier = modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp),
            shape = RoundedCornerShape(12.dp),
            colors = CardDefaults.cardColors(containerColor = containerColor)
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.Top
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = iconColor,
                    modifier = Modifier.size(24.dp)
                )

                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Text(
                        text = event.title,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                        color = iconColor
                    )
                    Text(
                        text = event.message,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
                    )
                    if (event.actionLabel != null && event.onAction != null) {
                        TextButton(
                            onClick = event.onAction,
                            modifier = Modifier.padding(top = 4.dp)
                        ) {
                            Text(
                                text = event.actionLabel,
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = FontWeight.Bold,
                                color = iconColor
                            )
                        }
                    }
                }

                // Only show dismiss button if this event type allows dismissal
                if (canDismiss) {
                    IconButton(
                        onClick = {
                            visible = false
                            onDismiss(event.id)
                        },
                        modifier = Modifier.size(24.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Dismiss",
                            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                            modifier = Modifier.size(16.dp)
                        )
                    }
                }
            }
        }
    }
}
