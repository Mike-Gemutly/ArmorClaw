package com.armorclaw.app.screens.settings.components
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.TrustLevel
import com.armorclaw.shared.ui.theme.*

/**
 * Badge displaying trust level with visual indicator
 */
@Composable
fun TrustBadge(
    trustLevel: TrustLevel,
    modifier: Modifier = Modifier,
    showLabel: Boolean = true,
    size: TrustBadgeSize = TrustBadgeSize.MEDIUM
) {
    val config = getTrustLevelConfig(trustLevel)

    val backgroundColor by animateColorAsState(
        targetValue = config.backgroundColor,
        label = "background_color"
    )

    val borderColor by animateColorAsState(
        targetValue = config.borderColor,
        label = "border_color"
    )

    val iconSize = when (size) {
        TrustBadgeSize.SMALL -> 12.dp
        TrustBadgeSize.MEDIUM -> 16.dp
        TrustBadgeSize.LARGE -> 20.dp
    }

    val textPadding = when (size) {
        TrustBadgeSize.SMALL -> 4.dp
        TrustBadgeSize.MEDIUM -> 6.dp
        TrustBadgeSize.LARGE -> 8.dp
    }

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(size.cornerRadius))
            .background(backgroundColor)
            .border(1.dp, borderColor, RoundedCornerShape(size.cornerRadius))
            .padding(horizontal = textPadding, vertical = textPadding / 2)
            .semantics { contentDescription = config.accessibilityDescription },
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = config.icon,
            contentDescription = null,
            tint = config.iconColor,
            modifier = Modifier.size(iconSize)
        )

        if (showLabel) {
            Text(
                text = config.label,
                color = config.textColor,
                style = when (size) {
                    TrustBadgeSize.SMALL -> MaterialTheme.typography.labelSmall
                    TrustBadgeSize.MEDIUM -> MaterialTheme.typography.bodyMedium
                    TrustBadgeSize.LARGE -> MaterialTheme.typography.bodyLarge
                },
                fontWeight = FontWeight.Medium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }
    }
}

/**
 * Compact trust indicator for list items
 */
@Composable
fun TrustIndicator(
    trustLevel: TrustLevel,
    modifier: Modifier = Modifier
) {
    val config = getTrustLevelConfig(trustLevel)

    Box(
        modifier = modifier
            .size(8.dp)
            .clip(CircleShape)
            .background(config.indicatorColor)
    )
}

/**
 * Trust level status card
 */
@Composable
fun TrustStatusCard(
    trustLevel: TrustLevel,
    deviceName: String? = null,
    lastVerified: String? = null,
    onActionClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val config = getTrustLevelConfig(trustLevel)

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        color = config.backgroundColor,
        border = androidx.compose.foundation.BorderStroke(1.dp, config.borderColor),
        onClick = onActionClick
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Icon
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(config.iconColor.copy(alpha = 0.15f)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = config.icon,
                        contentDescription = null,
                        tint = config.iconColor,
                        modifier = Modifier.size(24.dp)
                    )
                }

                // Content
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = config.label,
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Bold,
                        color = config.textColor
                    )

                    if (!deviceName.isNullOrBlank()) {
                        Text(
                            text = deviceName,
                            style = MaterialTheme.typography.bodySmall,
                            color = OnBackground.copy(alpha = 0.6f)
                        )
                    }
                }

                // Status indicator
                Box(
                    modifier = Modifier
                        .size(12.dp)
                        .clip(CircleShape)
                        .background(config.indicatorColor)
                )
            }

            if (!lastVerified.isNullOrBlank() && trustLevel != TrustLevel.UNVERIFIED) {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = "Last verified: $lastVerified",
                    style = MaterialTheme.typography.bodySmall,
                    color = OnBackground.copy(alpha = 0.5f)
                )
            }

            if (trustLevel == TrustLevel.UNVERIFIED) {
                Spacer(modifier = Modifier.height(8.dp))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.Warning,
                        contentDescription = null,
                        tint = StatusWarning,
                        modifier = Modifier.size(14.dp)
                    )
                    Text(
                        text = "Tap to verify this device",
                        style = MaterialTheme.typography.bodySmall,
                        color = StatusWarning,
                        fontWeight = FontWeight.Medium
                    )
                }
            }
        }
    }
}

enum class TrustBadgeSize(val cornerRadius: Dp) {
    SMALL(4.dp),
    MEDIUM(8.dp),
    LARGE(12.dp)
}

private data class TrustLevelConfig(
    val label: String,
    val icon: ImageVector,
    val backgroundColor: Color,
    val borderColor: Color,
    val iconColor: Color,
    val textColor: Color,
    val indicatorColor: Color,
    val accessibilityDescription: String
)

@Composable
private fun getTrustLevelConfig(level: TrustLevel): TrustLevelConfig {
    return when (level) {
        TrustLevel.UNVERIFIED -> TrustLevelConfig(
            label = "Unverified",
            icon = Icons.Default.Help,
            backgroundColor = Color(0xFFFFF7ED),
            borderColor = Color(0xFFF97316),
            iconColor = Color(0xFFF97316),
            textColor = Color(0xFFC2410C),
            indicatorColor = Color(0xFFEF4444),
            accessibilityDescription = "Device trust level: Unverified"
        )
        TrustLevel.CROSS_SIGNED -> TrustLevelConfig(
            label = "Cross-signed",
            icon = Icons.Default.VerifiedUser,
            backgroundColor = Color(0xFFECFDF5),
            borderColor = Color(0xFF10B981),
            iconColor = Color(0xFF10B981),
            textColor = Color(0xFF065F46),
            indicatorColor = Color(0xFF10B981),
            accessibilityDescription = "Device trust level: Cross-signed, verified"
        )
        TrustLevel.VERIFIED_IN_PERSON -> TrustLevelConfig(
            label = "Verified in person",
            icon = Icons.Default.VerifiedUser,
            backgroundColor = Color(0xFFEFF6FF),
            borderColor = Color(0xFF3B82F6),
            iconColor = Color(0xFF3B82F6),
            textColor = Color(0xFF1E40AF),
            indicatorColor = Color(0xFF3B82F6),
            accessibilityDescription = "Device trust level: Verified in person, highest trust"
        )
        TrustLevel.KNOWN_UNCOMPROMISED -> TrustLevelConfig(
            label = "Known device",
            icon = Icons.Default.CheckCircle,
            backgroundColor = Color(0xFFF0FDF4),
            borderColor = Color(0xFF22C55E),
            iconColor = Color(0xFF22C55E),
            textColor = Color(0xFF166534),
            indicatorColor = Color(0xFF22C55E),
            accessibilityDescription = "Device trust level: Known, uncompromised"
        )
        TrustLevel.COMPROMISED -> TrustLevelConfig(
            label = "Compromised",
            icon = Icons.Default.Warning,
            backgroundColor = Color(0xFFFEF2F2),
            borderColor = Color(0xFFEF4444),
            iconColor = Color(0xFFEF4444),
            textColor = Color(0xFF991B1B),
            indicatorColor = Color(0xFFEF4444),
            accessibilityDescription = "Device trust level: Compromised, security risk"
        )
    }
}
