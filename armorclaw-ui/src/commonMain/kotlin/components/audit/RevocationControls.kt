package components.audit

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import components.governor.RiskLevel

/**
 * One-Click Revocation Panel
 *
 * Provides instant capability revocation with a single tap.
 * Shows all active capabilities with revocation controls.
 *
 * Phase 3 Implementation - Governor Strategy
 *
 * @param activeCapabilities List of currently active capabilities
 * @param onRevoke Callback when a capability is revoked
 * @param onRevokeAll Callback when all capabilities are revoked
 * @param modifier Optional modifier
 */
@Composable
fun RevocationPanel(
    activeCapabilities: List<ActiveCapability>,
    onRevoke: (ActiveCapability) -> Unit,
    onRevokeAll: () -> Unit,
    modifier: Modifier = Modifier
) {
    var showConfirmDialog by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(Color(0xFF1A1A2E), RoundedCornerShape(12.dp))
            .border(1.dp, Color(0xFF4A4A6A), RoundedCornerShape(12.dp))
            .padding(16.dp)
    ) {
        // Header
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = null,
                    tint = Color(0xFFF44336),
                    modifier = Modifier.size(24.dp)
                )
                Text(
                    text = "Active Capabilities",
                    style = MaterialTheme.typography.titleMedium,
                    color = Color.White,
                    fontWeight = FontWeight.Bold
                )

                // Count badge
                Surface(
                    shape = CircleShape,
                    color = Color(0xFFF44336).copy(alpha = 0.2f)
                ) {
                    Text(
                        text = activeCapabilities.size.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = Color(0xFFF44336),
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                    )
                }
            }

            // Revoke all button
            if (activeCapabilities.isNotEmpty()) {
                Button(
                    onClick = { showConfirmDialog = true },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color(0xFFF44336)
                    ),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Refresh,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp)
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Revoke All", style = MaterialTheme.typography.labelMedium)
                }
            }
        }

        if (activeCapabilities.isEmpty()) {
            // Empty state
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 24.dp),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.CheckCircle,
                        contentDescription = null,
                        tint = Color(0xFF4CAF50),
                        modifier = Modifier.size(40.dp)
                    )
                    Text(
                        text = "No active capabilities",
                        style = MaterialTheme.typography.bodyMedium,
                        color = Color.White.copy(alpha = 0.6f)
                    )
                    Text(
                        text = "All agent permissions have been revoked",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color.White.copy(alpha = 0.4f)
                    )
                }
            }
        } else {
            Spacer(modifier = Modifier.height(16.dp))

            // Active capabilities list
            LazyColumn(
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(activeCapabilities, key = { it.id }) { cap ->
                    ActiveCapabilityCard(
                        capability = cap,
                        onRevoke = { onRevoke(cap) }
                    )
                }
            }
        }
    }

    // Confirm revocation dialog
    if (showConfirmDialog) {
        AlertDialog(
            onDismissRequest = { showConfirmDialog = false },
            icon = {
                Icon(
                    imageVector = Icons.Default.Warning,
                    contentDescription = null,
                    tint = Color(0xFFF44336)
                )
            },
            title = { Text("Revoke All Capabilities?") },
            text = {
                Text("This will immediately revoke all ${activeCapabilities.size} active capabilities. Agents will lose access until permissions are re-granted.")
            },
            confirmButton = {
                Button(
                    onClick = {
                        onRevokeAll()
                        showConfirmDialog = false
                    },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color(0xFFF44336)
                    )
                ) {
                    Text("Revoke All")
                }
            },
            dismissButton = {
                OutlinedButton(onClick = { showConfirmDialog = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}

/**
 * Active Capability Card
 *
 * Shows a single active capability with revocation button
 */
@Composable
private fun ActiveCapabilityCard(
    capability: ActiveCapability,
    onRevoke: () -> Unit
) {
    val riskColor = getRiskColor(capability.riskLevel)

    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFF252540)
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                // Risk indicator
                Box(
                    modifier = Modifier
                        .width(3.dp)
                        .height(40.dp)
                        .background(riskColor, RoundedCornerShape(1.dp))
                )

                // Capability info
                Column {
                    Text(
                        text = capability.displayName,
                        style = MaterialTheme.typography.bodyMedium,
                        color = Color.White,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = "Active for ${formatDuration(capability.activeDuration)}",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color.White.copy(alpha = 0.6f)
                    )
                }
            }

            // Agent info
            Column(
                horizontalAlignment = Alignment.End
            ) {
                Text(
                    text = capability.agentName,
                    style = MaterialTheme.typography.labelSmall,
                    color = Color(0xFF58A6FF)
                )
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Box(
                        modifier = Modifier
                            .size(6.dp)
                            .background(Color(0xFF3FB950), CircleShape)
                    )
                    Text(
                        text = "${capability.usageCount} uses",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color.White.copy(alpha = 0.6f)
                    )
                }
            }

            // Revoke button
            IconButton(
                onClick = onRevoke,
                modifier = Modifier
                    .size(36.dp)
                    .background(Color(0xFFF44336).copy(alpha = 0.1f), CircleShape)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "Revoke",
                    tint = Color(0xFFF44336),
                    modifier = Modifier.size(18.dp)
                )
            }
        }
    }
}

/**
 * Quick Revocation Button
 *
 * Single-click emergency revocation button
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun QuickRevocationButton(
    activeCount: Int,
    onRevokeAll: () -> Unit,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (activeCount > 0) 1.05f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    FloatingActionButton(
        onClick = onRevokeAll,
        modifier = modifier
            .size(56.dp)
            .graphicsLayer {
                scaleX = scale
                scaleY = scale
            },
        containerColor = if (activeCount > 0) Color(0xFFF44336) else Color(0xFF4A4A6A),
        contentColor = Color.White
    ) {
        BadgedBox(
            badge = {
                if (activeCount > 0) {
                    Badge(
                        containerColor = Color.White,
                        contentColor = Color(0xFFF44336)
                    ) {
                        Text(activeCount.toString())
                    }
                }
            }
        ) {
            Icon(
                imageVector = Icons.Default.Warning,
                contentDescription = "Emergency Revocation",
                modifier = Modifier.size(24.dp)
            )
        }
    }
}

/**
 * Active Capability
 */
@Immutable
data class ActiveCapability(
    val id: String,
    val name: String,
    val displayName: String,
    val agentId: String,
    val agentName: String,
    val riskLevel: RiskLevel,
    val grantedAt: Long,
    val activeDuration: Long,
    val usageCount: Int,
    val lastUsed: Long?
)

// Helper functions

private fun getRiskColor(riskLevel: RiskLevel): Color {
    return when (riskLevel) {
        RiskLevel.LOW -> Color(0xFF4CAF50)
        RiskLevel.MEDIUM -> Color(0xFFFFC107)
        RiskLevel.HIGH -> Color(0xFFFF9800)
        RiskLevel.CRITICAL -> Color(0xFFF44336)
    }
}

private fun formatDuration(millis: Long): String {
    val seconds = millis / 1000
    return when {
        seconds < 60 -> "${seconds}s"
        seconds < 3600 -> "${seconds / 60}m"
        else -> "${seconds / 3600}h ${seconds % 3600 / 60}m"
    }
}

/**
 * Preview
 */
@Composable
fun RevocationPanelPreview() {
    val sampleCapabilities = listOf(
        ActiveCapability(
            id = "c1",
            name = "calendar_write",
            displayName = "Calendar Write",
            agentId = "agent_1",
            agentName = "CalendarAgent",
            riskLevel = RiskLevel.MEDIUM,
            grantedAt = System.currentTimeMillis() - 300000,
            activeDuration = 300000,
            usageCount = 5,
            lastUsed = System.currentTimeMillis() - 60000
        ),
        ActiveCapability(
            id = "c2",
            name = "send_email",
            displayName = "Send Email",
            agentId = "agent_2",
            agentName = "EmailAgent",
            riskLevel = RiskLevel.HIGH,
            grantedAt = System.currentTimeMillis() - 600000,
            activeDuration = 600000,
            usageCount = 12,
            lastUsed = System.currentTimeMillis() - 30000
        ),
        ActiveCapability(
            id = "c3",
            name = "payment",
            displayName = "Payment Processing",
            agentId = "agent_3",
            agentName = "PaymentAgent",
            riskLevel = RiskLevel.CRITICAL,
            grantedAt = System.currentTimeMillis() - 60000,
            activeDuration = 60000,
            usageCount = 1,
            lastUsed = System.currentTimeMillis() - 10000
        )
    )

    MaterialTheme {
        Surface(color = Color(0xFF0D1117)) {
            Column(
                modifier = Modifier
                    .padding(16.dp)
                    .width(400.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                RevocationPanel(
                    activeCapabilities = sampleCapabilities,
                    onRevoke = {},
                    onRevokeAll = {}
                )

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End
                ) {
                    QuickRevocationButton(
                        activeCount = sampleCapabilities.size,
                        onRevokeAll = {}
                    )
                }
            }
        }
    }
}
