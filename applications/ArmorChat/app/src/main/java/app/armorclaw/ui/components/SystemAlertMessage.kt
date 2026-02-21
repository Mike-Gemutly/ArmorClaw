package app.armorclaw.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import app.armorclaw.data.model.*

/**
 * System Alert Message - Distinct UI for Bridge Alerts
 *
 * Resolves: Gap 4 (Notification Pipeline "Split-Brain")
 *
 * Renders system alerts with distinct styling based on severity.
 * Ensures critical alerts are not lost in regular chat stream.
 */

/**
 * System Alert Card Component
 *
 * Renders a system alert with severity-based styling.
 */
@Composable
fun SystemAlertCard(
    alert: SystemAlertContent,
    onActionClick: ((String) -> Unit)? = null,
    onDismiss: (() -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    val colors = alert.severity.getColors()

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = colors.background
        ),
        border = androidx.compose.foundation.BorderStroke(1.dp, colors.border)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        ) {
            // Header row with icon and severity
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = alert.alertType.getIcon(),
                        contentDescription = null,
                        tint = colors.icon,
                        modifier = Modifier.size(24.dp)
                    )

                    Text(
                        text = alert.title,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = colors.text
                    )
                }

                // Dismiss button
                if (onDismiss != null) {
                    IconButton(
                        onClick = onDismiss,
                        modifier = Modifier.size(24.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Dismiss",
                            tint = colors.text.copy(alpha = 0.6f),
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Message
            Text(
                text = alert.message,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.text
            )

            // Action button
            if (alert.action != null && onActionClick != null) {
                Spacer(modifier = Modifier.height(12.dp))

                Button(
                    onClick = { alert.actionUrl?.let { onActionClick(it) } },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = colors.actionBackground,
                        contentColor = colors.actionText
                    ),
                    modifier = Modifier.height(36.dp)
                ) {
                    Text(
                        text = alert.action,
                        style = MaterialTheme.typography.labelMedium
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Icon(
                        imageVector = Icons.Default.ArrowForward,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp)
                    )
                }
            }

            // Severity badge
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp),
                horizontalArrangement = Arrangement.End
            ) {
                Surface(
                    shape = RoundedCornerShape(4.dp),
                    color = colors.badgeBackground
                ) {
                    Text(
                        text = alert.severity.displayName.uppercase(),
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Bold,
                        color = colors.badgeText,
                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                    )
                }
            }
        }
    }
}

/**
 * Compact alert banner for timeline
 */
@Composable
fun SystemAlertBanner(
    alert: SystemAlertContent,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val colors = alert.severity.getColors()

    Surface(
        modifier = modifier
            .fillMaxWidth()
            .height(IntrinsicSize.Min),
        color = colors.background,
        onClick = onClick
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = alert.alertType.getIcon(),
                contentDescription = null,
                tint = colors.icon,
                modifier = Modifier.size(20.dp)
            )

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = alert.title,
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold,
                    color = colors.text,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    text = alert.message,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.text.copy(alpha = 0.8f),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }

            // Severity indicator
            Box(
                modifier = Modifier
                    .width(4.dp)
                    .height(32.dp)
                    .background(colors.border, RoundedCornerShape(2.dp))
            )
        }
    }
}

/**
 * Alert colors based on severity
 */
data class AlertColors(
    val background: Color,
    val border: Color,
    val text: Color,
    val icon: Color,
    val actionBackground: Color,
    val actionText: Color,
    val badgeBackground: Color,
    val badgeText: Color
)

@Composable
fun AlertSeverity.getColors(): AlertColors {
    return when (this) {
        AlertSeverity.INFO -> AlertColors(
            background = Color(0xFFE3F2FD), // Blue 50
            border = Color(0xFF2196F3), // Blue 500
            text = Color(0xFF0D47A1), // Blue 900
            icon = Color(0xFF1976D2), // Blue 700
            actionBackground = Color(0xFF2196F3),
            actionText = Color.White,
            badgeBackground = Color(0xFFBBDEFB),
            badgeText = Color(0xFF0D47A1)
        )

        AlertSeverity.WARNING -> AlertColors(
            background = Color(0xFFFFF8E1), // Amber 50
            border = Color(0xFFFFC107), // Amber 500
            text = Color(0xFFFF6F00), // Amber 900
            icon = Color(0xFFFF8F00), // Amber 800
            actionBackground = Color(0xFFFFC107),
            actionText = Color(0xFF5D4037),
            badgeBackground = Color(0xFFFFECB3),
            badgeText = Color(0xFFFF6F00)
        )

        AlertSeverity.ERROR -> AlertColors(
            background = Color(0xFFFFEBEE), // Red 50
            border = Color(0xFFF44336), // Red 500
            text = Color(0xFFB71C1C), // Red 900
            icon = Color(0xFFD32F2F), // Red 700
            actionBackground = Color(0xFFF44336),
            actionText = Color.White,
            badgeBackground = Color(0xFFFFCDD2),
            badgeText = Color(0xFFB71C1C)
        )

        AlertSeverity.CRITICAL -> AlertColors(
            background = Color(0xFFD32F2F), // Red 700
            border = Color(0xFFB71C1C), // Red 900
            text = Color.White,
            icon = Color.White,
            actionBackground = Color.White,
            actionText = Color(0xFFB71C1C),
            badgeBackground = Color(0xFFEF5350),
            badgeText = Color.White
        )
    }
}

/**
 * Get icon for alert type
 */
fun AlertType.getIcon(): ImageVector {
    return when (this) {
        AlertType.BUDGET_WARNING,
        AlertType.BUDGET_EXCEEDED -> Icons.Default.AttachMoney

        AlertType.LICENSE_EXPIRING,
        AlertType.LICENSE_EXPIRED,
        AlertType.LICENSE_INVALID -> Icons.Default.CardMembership

        AlertType.SECURITY_EVENT,
        AlertType.TRUST_DEGRADED -> Icons.Default.Security

        AlertType.VERIFICATION_REQUIRED -> Icons.Default.VerifiedUser

        AlertType.BRIDGE_SECURITY_DOWNGRADE -> Icons.Default.LockOpen

        AlertType.BRIDGE_ERROR -> Icons.Default.Error

        AlertType.BRIDGE_RESTARTING -> Icons.Default.Refresh

        AlertType.MAINTENANCE -> Icons.Default.Build

        AlertType.COMPLIANCE_VIOLATION -> Icons.Default.Gavel

        AlertType.AUDIT_EXPORT -> Icons.Default.Download
    }
}

/**
 * Preview composables
 */
@Composable
fun SystemAlertPreview() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Info alert
        SystemAlertCard(
            alert = AlertFactory.verificationRequired("Chrome Browser"),
            onActionClick = {}
        )

        // Warning alert
        SystemAlertCard(
            alert = AlertFactory.budgetWarning(45.50, 100.0, 45),
            onActionClick = {}
        )

        // Error alert
        SystemAlertCard(
            alert = AlertFactory.bridgeError("Matrix Adapter", "Connection timeout"),
            onActionClick = {}
        )

        // Critical alert
        SystemAlertCard(
            alert = AlertFactory.licenseExpired(),
            onActionClick = {}
        )
    }
}
