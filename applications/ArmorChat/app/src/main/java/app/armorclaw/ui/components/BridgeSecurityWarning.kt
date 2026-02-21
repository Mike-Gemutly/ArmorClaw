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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Bridge Security Warning - E2EE Downgrade Alert
 *
 * Resolves: Gap - Security Downgrade Warning (E2EE)
 *
 * Shows a warning when an E2EE Matrix room is bridged to a
 * non-E2EE external platform (Slack, Discord, Teams).
 *
 * This warns users that their encrypted messages will be
 * decrypted and sent in plaintext to the external platform.
 */

/**
 * Bridge security level for a room
 */
enum class BridgeSecurityLevel {
    /** Native Matrix - full E2EE */
    NATIVE_E2EE,
    /** Bridged to platform with E2EE support */
    BRIDGED_SECURE,
    /** Bridged to platform WITHOUT E2EE (security downgrade) */
    BRIDGED_INSECURE,
    /** Unknown security status */
    UNKNOWN
}

/**
 * Data class for bridge security info
 */
data class BridgeSecurityInfo(
    val securityLevel: BridgeSecurityLevel,
    val isRoomEncrypted: Boolean,
    val bridgedPlatforms: List<BridgedPlatform> = emptyList(),
    val hasInsecureBridge: Boolean = false
)

/**
 * Information about a bridged platform
 */
data class BridgedPlatform(
    val name: String,
    val displayName: String,
    val supportsE2EE: Boolean,
    val icon: String? = null
)

/**
 * Known bridge platforms with E2EE support status
 */
object BridgePlatforms {
    val SLACK = BridgedPlatform("slack", "Slack", supportsE2EE = false)
    val DISCORD = BridgedPlatform("discord", "Discord", supportsE2EE = false)
    val TEAMS = BridgedPlatform("teams", "Microsoft Teams", supportsE2EE = false)
    val WHATSAPP = BridgedPlatform("whatsapp", "WhatsApp", supportsE2EE = true)
    val SIGNAL = BridgedPlatform("signal", "Signal", supportsE2EE = true)

    val ALL = listOf(SLACK, DISCORD, TEAMS, WHATSAPP, SIGNAL)

    fun fromName(name: String): BridgedPlatform? {
        return ALL.find { it.name.equals(name, ignoreCase = true) }
    }
}

/**
 * Security warning banner for bridged rooms
 */
@Composable
fun BridgeSecurityWarningBanner(
    securityInfo: BridgeSecurityInfo,
    onDismiss: (() -> Unit)? = null,
    onLearnMore: () -> Unit,
    modifier: Modifier = Modifier
) {
    // Only show warning for insecure bridges in encrypted rooms
    if (!securityInfo.hasInsecureBridge || !securityInfo.isRoomEncrypted) {
        return
    }

    val insecurePlatforms = securityInfo.bridgedPlatforms.filter { !it.supportsE2EE }
    val platformNames = insecurePlatforms.joinToString(", ") { it.displayName }

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFFFFEBEE) // Red 50
        ),
        border = androidx.compose.foundation.BorderStroke(1.dp, Color(0xFFF44336))
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp)
        ) {
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
                        imageVector = Icons.Default.Warning,
                        contentDescription = "Security Warning",
                        tint = Color(0xFFD32F2F),
                        modifier = Modifier.size(24.dp)
                    )

                    Text(
                        text = "E2EE Bridge Warning",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = Color(0xFFB71C1C)
                    )
                }

                if (onDismiss != null) {
                    IconButton(
                        onClick = onDismiss,
                        modifier = Modifier.size(24.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Dismiss",
                            tint = Color(0xFFB71C1C).copy(alpha = 0.6f),
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "This encrypted room is bridged to $platformNames. " +
                       "Your messages will be decrypted before being sent to these platforms.",
                style = MaterialTheme.typography.bodyMedium,
                color = Color(0xFFB71C1C)
            )

            Spacer(modifier = Modifier.height(8.dp))

            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                TextButton(
                    onClick = onLearnMore,
                    colors = ButtonDefaults.textButtonColors(
                        contentColor = Color(0xFFB71C1C)
                    )
                ) {
                    Icon(
                        imageVector = Icons.Default.Info,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp)
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Learn More")
                }
            }

            // Security badge
            Surface(
                shape = RoundedCornerShape(4.dp),
                color = Color(0xFFFFCDD2)
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.LockOpen,
                        contentDescription = null,
                        tint = Color(0xFFB71C1C),
                        modifier = Modifier.size(14.dp)
                    )
                    Text(
                        text = "END-TO-END ENCRYPTION NOT PRESERVED",
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Bold,
                        color = Color(0xFFB71C1C)
                    )
                }
            }
        }
    }
}

/**
 * Compact security indicator for room list/items
 */
@Composable
fun BridgeSecurityIndicator(
    securityInfo: BridgeSecurityInfo,
    modifier: Modifier = Modifier
) {
    if (!securityInfo.hasInsecureBridge || !securityInfo.isRoomEncrypted) {
        return
    }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(4.dp),
        color = Color(0xFFFFCDD2)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(2.dp)
        ) {
            Icon(
                imageVector = Icons.Default.LockOpen,
                contentDescription = "E2EE not preserved",
                tint = Color(0xFFD32F2F),
                modifier = Modifier.size(12.dp)
            )
            Text(
                text = "BRIDGED",
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.Bold,
                color = Color(0xFFD32F2F)
            )
        }
    }
}

/**
 * Full-screen dialog explaining bridge security
 */
@Composable
fun BridgeSecurityInfoDialog(
    securityInfo: BridgeSecurityInfo,
    onDismiss: () -> Unit,
    onAccept: () -> Unit,
    modifier: Modifier = Modifier
) {
    val insecurePlatforms = securityInfo.bridgedPlatforms.filter { !it.supportsE2EE }

    AlertDialog(
        onDismissRequest = onDismiss,
        icon = {
            Icon(
                imageVector = Icons.Default.Shield,
                contentDescription = null,
                tint = Color(0xFFD32F2F),
                modifier = Modifier.size(48.dp)
            )
        },
        title = {
            Text("Bridge Security Information")
        },
        text = {
            Column(
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Text(
                    text = "This Matrix room has end-to-end encryption enabled, but is also bridged to external platforms that don't support E2EE.",
                    style = MaterialTheme.typography.bodyMedium
                )

                if (insecurePlatforms.isNotEmpty()) {
                    Text(
                        text = "Affected Platforms:",
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold
                    )

                    insecurePlatforms.forEach { platform ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            Icon(
                                imageVector = Icons.Default.Close,
                                contentDescription = "Not secure",
                                tint = Color(0xFFD32F2F),
                                modifier = Modifier.size(16.dp)
                            )
                            Text(
                                text = platform.displayName,
                                style = MaterialTheme.typography.bodyMedium
                            )
                        }
                    }
                }

                HorizontalDivider()

                Text(
                    text = "What this means:",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold
                )

                Column(
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    SecurityInfoRow(
                        icon = Icons.Default.Lock,
                        text = "Messages between Matrix users remain encrypted"
                    )
                    SecurityInfoRow(
                        icon = Icons.Default.LockOpen,
                        text = "Messages sent to bridged platforms are decrypted first",
                        isWarning = true
                    )
                    SecurityInfoRow(
                        icon = Icons.Default.Cloud,
                        text = "External platforms may store messages on their servers",
                        isWarning = true
                    )
                }

                HorizontalDivider()

                Text(
                    text = "Recommendation: Do not share highly sensitive information in this room if you need E2EE guarantees for all recipients.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        },
        confirmButton = {
            TextButton(onClick = onAccept) {
                Text("I Understand")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

@Composable
private fun SecurityInfoRow(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    text: String,
    isWarning: Boolean = false
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = if (isWarning) Color(0xFFD32F2F) else MaterialTheme.colorScheme.primary,
            modifier = Modifier.size(18.dp)
        )
        Text(
            text = text,
            style = MaterialTheme.typography.bodyMedium,
            color = if (isWarning) Color(0xFFB71C1C) else MaterialTheme.colorScheme.onSurface
        )
    }
}

/**
 * Pre-join security warning for bridged rooms
 */
@Composable
fun PreJoinBridgeSecurityWarning(
    securityInfo: BridgeSecurityInfo,
    onAcceptRisk: () -> Unit,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier
) {
    if (!securityInfo.hasInsecureBridge || !securityInfo.isRoomEncrypted) {
        return
    }

    val insecurePlatforms = securityInfo.bridgedPlatforms.filter { !it.supportsE2EE }
    val platformNames = insecurePlatforms.joinToString(", ") { it.displayName }

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFFFFEBEE)
        ),
        border = androidx.compose.foundation.BorderStroke(2.dp, Color(0xFFF44336))
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Icon(
                imageVector = Icons.Default.Warning,
                contentDescription = "Warning",
                tint = Color(0xFFD32F2F),
                modifier = Modifier.size(48.dp)
            )

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = "Security Notice",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                color = Color(0xFFB71C1C)
            )

            Spacer(modifier = Modifier.height(12.dp))

            Text(
                text = "This encrypted room is bridged to $platformNames.",
                style = MaterialTheme.typography.bodyMedium,
                color = Color(0xFFB71C1C)
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "Your encrypted messages will be decrypted and sent in plaintext to these external platforms.",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium,
                color = Color(0xFFB71C1C)
            )

            Spacer(modifier = Modifier.height(20.dp))

            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                OutlinedButton(
                    onClick = onCancel,
                    colors = ButtonDefaults.outlinedButtonColors(
                        contentColor = Color(0xFFB71C1C)
                    )
                ) {
                    Text("Cancel")
                }

                Button(
                    onClick = onAcceptRisk,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color(0xFFD32F2F)
                    )
                ) {
                    Text("Join Anyway")
                }
            }
        }
    }
}
