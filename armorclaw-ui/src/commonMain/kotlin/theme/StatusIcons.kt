package theme

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Context-Aware Status Icons
 *
 * Visual indicators that adapt their appearance based on context:
 * - Security status (vault locked/unlocked)
 * - Agent activity (active/idle/error)
 * - Network status (connected/disconnected)
 * - Capability status (granted/revoked/pending)
 *
 * Phase 4 Implementation - Governor Strategy
 */

/**
 * Security Status Indicator
 *
 * Shows the current security status of the vault
 */
@Composable
fun SecurityStatusIcon(
    isSecured: Boolean,
    modifier: Modifier = Modifier
) {
    val color = if (isSecured) ArmorSuccess else ArmorWarning
    val icon = if (isSecured) Icons.Default.Lock else Icons.Default.Lock

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(8.dp),
        color = color.copy(alpha = 0.15f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(16.dp)
            )
            Text(
                text = if (isSecured) "Secured" else "Unsecured",
                style = MaterialTheme.typography.labelSmall,
                color = color
            )
        }
    }
}

/**
 * Agent Status Indicator
 *
 * Shows the current status of an agent with pulsing animation when active
 */
@Composable
fun AgentStatusIcon(
    status: AgentStatus,
    showLabel: Boolean = true,
    modifier: Modifier = Modifier
) {
    val (color, icon, label) = when (status) {
        AgentStatus.ONLINE -> Triple(ArmorSuccess, Icons.Default.Check, "Online")
        AgentStatus.BUSY -> Triple(ArmorWarning, Icons.Default.Refresh, "Busy")
        AgentStatus.OFFLINE -> Triple(ArmorTextMuted, Icons.Default.Close, "Offline")
        AgentStatus.ERROR -> Triple(ArmorError, Icons.Default.Warning, "Error")
    }

    // Pulse animation for busy status
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    val displayColor = if (status == AgentStatus.BUSY) color.copy(alpha = alpha) else color

    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        Box(
            modifier = Modifier
                .size(8.dp)
                .background(displayColor, CircleShape)
        )
        if (showLabel) {
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                color = displayColor
            )
        }
    }
}

/**
 * Agent Status Enum
 */
enum class AgentStatus {
    ONLINE,
    BUSY,
    OFFLINE,
    ERROR
}

/**
 * Capability Status Indicator
 *
 * Shows the current status of a capability
 */
@Composable
fun CapabilityStatusIcon(
    status: CapabilityStatus,
    size: StatusIconSize = StatusIconSize.MEDIUM,
    modifier: Modifier = Modifier
) {
    val (color, icon) = when (status) {
        CapabilityStatus.GRANTED -> ArmorSuccess to Icons.Default.Check
        CapabilityStatus.REVOKED -> ArmorError to Icons.Default.Close
        CapabilityStatus.PENDING -> ArmorWarning to Icons.Default.DateRange
        CapabilityStatus.EXPIRED -> ArmorTextMuted to Icons.Default.DateRange
    }

    val iconSize = when (size) {
        StatusIconSize.SMALL -> 12.dp
        StatusIconSize.MEDIUM -> 16.dp
        StatusIconSize.LARGE -> 24.dp
    }

    Icon(
        imageVector = icon,
        contentDescription = status.name,
        tint = color,
        modifier = modifier.size(iconSize)
    )
}

/**
 * Capability Status Enum
 */
enum class CapabilityStatus {
    GRANTED,
    REVOKED,
    PENDING,
    EXPIRED
}

/**
 * Status Icon Size
 */
enum class StatusIconSize {
    SMALL,
    MEDIUM,
    LARGE
}

/**
 * Network Status Indicator
 *
 * Shows connection status
 */
@Composable
fun NetworkStatusIcon(
    isConnected: Boolean,
    strength: Int = 4, // 1-4 bars
    modifier: Modifier = Modifier
) {
    val color = if (isConnected) ArmorSuccess else ArmorError

    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        // Signal bars
        Row(
            horizontalArrangement = Arrangement.spacedBy(2.dp),
            verticalAlignment = Alignment.Bottom
        ) {
            repeat(4) { index ->
                val barHeight = when (index) {
                    0 -> 4.dp
                    1 -> 8.dp
                    2 -> 12.dp
                    else -> 16.dp
                }
                val isActive = index < strength && isConnected
                Box(
                    modifier = Modifier
                        .width(3.dp)
                        .height(barHeight)
                        .background(
                            color = if (isActive) color else ArmorBorder,
                            shape = RoundedCornerShape(1.dp)
                        )
                )
            }
        }
    }
}

/**
 * Risk Level Badge
 *
 * Shows risk level with appropriate color coding
 */
@Composable
fun RiskLevelBadge(
    level: RiskLevel,
    showIcon: Boolean = true,
    modifier: Modifier = Modifier
) {
    val (color, label, icon) = when (level) {
        RiskLevel.LOW -> Triple(ArmorSuccess, "Low Risk", Icons.Default.Check)
        RiskLevel.MEDIUM -> Triple(ArmorWarning, "Medium Risk", Icons.Default.Warning)
        RiskLevel.HIGH -> Triple(Color(0xFFFF9800), "High Risk", Icons.Default.Warning)
        RiskLevel.CRITICAL -> Triple(ArmorError, "Critical", Icons.Default.Warning)
    }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = color.copy(alpha = 0.15f),
        border = androidx.compose.foundation.BorderStroke(1.dp, color.copy(alpha = 0.3f))
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            if (showIcon) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = color,
                    modifier = Modifier.size(14.dp)
                )
            }
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                color = color,
                fontWeight = FontWeight.Medium
            )
        }
    }
}

/**
 * Risk Level Enum
 */
enum class RiskLevel {
    LOW,
    MEDIUM,
    HIGH,
    CRITICAL
}

/**
 * Activity Pulse Indicator
 *
 * Pulsing dot indicating active activity
 */
@Composable
fun ActivityPulseIndicator(
    isActive: Boolean,
    color: Color = ArmorTeal,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    
    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (isActive) 1.3f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )
    
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = if (isActive) 1f else 0.5f,
        animationSpec = infiniteRepeatable(
            animation = tween(500),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Box(
        modifier = modifier
            .size(12.dp)
            .background(
                color = color.copy(alpha = alpha),
                shape = CircleShape
            )
    )
}

/**
 * Combined Status Bar
 *
 * Shows multiple status indicators in a compact bar
 */
@Composable
fun StatusBar(
    isSecured: Boolean,
    agentStatus: AgentStatus,
    isConnected: Boolean,
    activeCapabilityCount: Int,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(8.dp),
        color = ArmorNavy.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            SecurityStatusIcon(isSecured = isSecured)
            
            AgentStatusIcon(status = agentStatus, showLabel = true)
            
            NetworkStatusIcon(isConnected = isConnected)
            
            Spacer(modifier = Modifier.weight(1f))
            
            // Capability count
            if (activeCapabilityCount > 0) {
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = ArmorTeal.copy(alpha = 0.15f)
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(4.dp)
                    ) {
                        ActivityPulseIndicator(isActive = true, modifier = Modifier.size(8.dp))
                        Text(
                            text = "$activeCapabilityCount active",
                            style = MaterialTheme.typography.labelSmall,
                            color = ArmorTeal
                        )
                    }
                }
            }
        }
    }
}

/**
 * Preview
 */
@Composable
fun StatusIconsPreview() {
    ArmorClawTheme {
        Surface(
            modifier = Modifier.padding(16.dp),
            color = ArmorSurface
        ) {
            Column(
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Text("Status Icons Preview", style = MaterialTheme.typography.titleMedium, color = ArmorText)
                
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    SecurityStatusIcon(isSecured = true)
                    SecurityStatusIcon(isSecured = false)
                }
                
                Row(
                    horizontalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    AgentStatusIcon(status = AgentStatus.ONLINE)
                    AgentStatusIcon(status = AgentStatus.BUSY)
                    AgentStatusIcon(status = AgentStatus.OFFLINE)
                    AgentStatusIcon(status = AgentStatus.ERROR)
                }
                
                Row(
                    horizontalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    NetworkStatusIcon(isConnected = true, strength = 4)
                    NetworkStatusIcon(isConnected = true, strength = 2)
                    NetworkStatusIcon(isConnected = false)
                }
                
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    RiskLevelBadge(level = RiskLevel.LOW)
                    RiskLevelBadge(level = RiskLevel.MEDIUM)
                    RiskLevelBadge(level = RiskLevel.HIGH)
                    RiskLevelBadge(level = RiskLevel.CRITICAL)
                }
                
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    ActivityPulseIndicator(isActive = true)
                    ActivityPulseIndicator(isActive = false)
                }
                
                Divider(color = ArmorBorder)
                
                StatusBar(
                    isSecured = true,
                    agentStatus = AgentStatus.BUSY,
                    isConnected = true,
                    activeCapabilityCount = 3
                )
            }
        }
    }
}
