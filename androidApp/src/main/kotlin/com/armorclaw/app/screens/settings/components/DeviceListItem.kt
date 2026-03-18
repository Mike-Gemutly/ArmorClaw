package com.armorclaw.app.screens.settings.components
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.DeviceInfo
import com.armorclaw.shared.domain.model.TrustLevel
import com.armorclaw.shared.ui.theme.*

/**
 * List item displaying a device with trust status
 */
@Composable
fun DeviceListItem(
    device: DeviceInfo,
    onClick: () -> Unit = {},
    onVerifyClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surface,
        onClick = onClick
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Device icon
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .clip(CircleShape)
                    .background(
                        when {
                            device.isCurrentDevice -> BrandPurple.copy(alpha = 0.15f)
                            device.trustLevel.isTrusted() -> BrandGreen.copy(alpha = 0.15f)
                            else -> OnBackground.copy(alpha = 0.1f)
                        }
                    ),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = if (device.isCurrentDevice)
                        Icons.Default.Phone
                    else
                        Icons.Default.Devices,
                    contentDescription = null,
                    tint = when {
                        device.isCurrentDevice -> BrandPurple
                        device.trustLevel.isTrusted() -> BrandGreen
                        else -> OnBackground.copy(alpha = 0.6f)
                    },
                    modifier = Modifier.size(24.dp)
                )
            }

            // Device info
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = device.displayName ?: device.deviceId.take(8),
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium,
                        color = OnBackground,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f, fill = false)
                    )

                    if (device.isCurrentDevice) {
                        Surface(
                            shape = RoundedCornerShape(4.dp),
                            color = BrandPurple.copy(alpha = 0.15f)
                        ) {
                            Text(
                                text = "This device",
                                style = MaterialTheme.typography.bodySmall,
                                color = BrandPurple,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                }

                // Device ID
                Text(
                    text = "ID: ${device.deviceId.take(16)}...",
                    style = MaterialTheme.typography.bodySmall,
                    color = OnBackground.copy(alpha = 0.5f),
                    maxLines = 1
                )

                // Last seen
                device.lastSeenTimestamp?.let { timestamp ->
                    val lastSeen = formatLastSeen(timestamp)
                    Text(
                        text = "Last seen: $lastSeen",
                        style = MaterialTheme.typography.bodySmall,
                        color = OnBackground.copy(alpha = 0.5f)
                    )
                }
            }

            // Trust badge or verify button
            when {
                device.trustLevel == TrustLevel.UNVERIFIED && !device.isCurrentDevice -> {
                    TextButton(
                        onClick = onVerifyClick,
                        colors = ButtonDefaults.textButtonColors(
                            contentColor = BrandPurple
                        )
                    ) {
                        Text("Verify")
                    }
                }
                else -> {
                    Column(
                        horizontalAlignment = Alignment.End,
                        verticalArrangement = Arrangement.spacedBy(4.dp)
                    ) {
                        TrustBadge(
                            trustLevel = device.trustLevel,
                            size = TrustBadgeSize.SMALL
                        )
                    }
                }
            }
        }
    }
}

/**
 * Compact device list item for smaller spaces
 */
@Composable
fun DeviceListItemCompact(
    device: DeviceInfo,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Trust indicator
        TrustIndicator(trustLevel = device.trustLevel)

        // Device name
        Text(
            text = device.displayName ?: device.deviceId.take(8),
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f)
        )

        // Current device badge
        if (device.isCurrentDevice) {
            Text(
                text = "(This device)",
                style = MaterialTheme.typography.bodySmall,
                color = BrandPurple
            )
        }
    }
}

/**
 * Device section header
 */
@Composable
fun DeviceSectionHeader(
    title: String,
    count: Int,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            color = OnBackground.copy(alpha = 0.7f)
        )

        Surface(
            shape = RoundedCornerShape(12.dp),
            color = OnBackground.copy(alpha = 0.1f)
        ) {
            Text(
                text = count.toString(),
                style = MaterialTheme.typography.bodySmall,
                color = OnBackground.copy(alpha = 0.7f),
                modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
            )
        }
    }
}

/**
 * Empty state for device list
 */
@Composable
fun DeviceListEmptyState(
    onRefresh: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(
            imageVector = Icons.Default.Devices,
            contentDescription = null,
            tint = OnBackground.copy(alpha = 0.3f),
            modifier = Modifier.size(64.dp)
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "No devices found",
            style = MaterialTheme.typography.titleMedium,
            color = OnBackground.copy(alpha = 0.6f)
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Pull to refresh or check your connection",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.4f)
        )

        Spacer(modifier = Modifier.height(16.dp))

        TextButton(onClick = onRefresh) {
            Icon(
                imageVector = Icons.Default.Refresh,
                contentDescription = null,
                modifier = Modifier.size(18.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text("Refresh")
        }
    }
}

private fun formatLastSeen(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

    return when {
        diff < 60_000 -> "Just now"
        diff < 3_600_000 -> "${diff / 60_000}m ago"
        diff < 86_400_000 -> "${diff / 3_600_000}h ago"
        diff < 604_800_000 -> "${diff / 86_400_000}d ago"
        else -> {
            val sdf = java.text.SimpleDateFormat("MMM d, yyyy", java.util.Locale.getDefault())
            sdf.format(java.util.Date(timestamp))
        }
    }
}
