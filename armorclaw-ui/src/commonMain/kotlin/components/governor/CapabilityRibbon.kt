package components.governor

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Capability Ribbon
 *
 * Horizontal scrollable ribbon showing available/active agent capabilities.
 * Provides quick visibility into what actions agents can perform.
 *
 * Phase 2 Implementation - Governor Strategy
 *
 * @param capabilities List of capabilities to display
 * @param activeCapabilities Set of capability IDs currently active
 * @param onCapabilityClick Callback when a capability is clicked
 * @param modifier Optional modifier
 */
@Composable
fun CapabilityRibbon(
    capabilities: List<Capability>,
    activeCapabilities: Set<String> = emptySet(),
    onCapabilityClick: (Capability) -> Unit = {},
    modifier: Modifier = Modifier
) {
    val scrollState = rememberScrollState()

    if (capabilities.isEmpty()) {
        // Empty state
        Surface(
            modifier = modifier,
            shape = RoundedCornerShape(12.dp),
            color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Info,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.size(16.dp)
                )
                Text(
                    text = "No capabilities available",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    } else {
        Row(
            modifier = modifier
                .fillMaxWidth()
                .horizontalScroll(scrollState),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            capabilities.forEach { capability ->
                CapabilityChip(
                    capability = capability,
                    isActive = activeCapabilities.contains(capability.id),
                    onClick = { onCapabilityClick(capability) }
                )
            }
        }
    }
}

/**
 * Capability Chip
 *
 * Individual chip for a single capability
 */
@Composable
fun CapabilityChip(
    capability: Capability,
    isActive: Boolean = false,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val riskColor = getRiskColor(capability.riskLevel)
    val backgroundColor = if (isActive) {
        riskColor.copy(alpha = 0.2f)
    } else {
        MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    }

    Surface(
        modifier = modifier
            .height(32.dp)
            .border(
                width = if (isActive) 1.dp else 0.dp,
                color = if (isActive) riskColor else Color.Transparent,
                shape = RoundedCornerShape(16.dp)
            ),
        shape = RoundedCornerShape(16.dp),
        color = backgroundColor,
        onClick = onClick
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            // Risk indicator dot
            Box(
                modifier = Modifier
                    .size(6.dp)
                    .background(riskColor, CircleShape)
            )

            // Capability name
            Text(
                text = capability.displayName,
                style = MaterialTheme.typography.labelMedium,
                fontWeight = if (isActive) FontWeight.Bold else FontWeight.Normal,
                color = if (isActive) riskColor else MaterialTheme.colorScheme.onSurface
            )

            // Approval indicator
            if (capability.requiresApproval) {
                Icon(
                    imageVector = Icons.Default.Lock,
                    contentDescription = "Requires approval",
                    modifier = Modifier.size(12.dp),
                    tint = riskColor.copy(alpha = 0.7f)
                )
            }
        }
    }
}

/**
 * Compact Capability Indicator
 *
 * Minimal indicator showing capability count and risk level
 */
@Composable
fun CapabilityIndicator(
    capabilities: List<Capability>,
    activeCount: Int = 0,
    modifier: Modifier = Modifier
) {
    val criticalCount = capabilities.count { it.riskLevel == RiskLevel.CRITICAL }
    val highRiskCount = capabilities.count { it.riskLevel == RiskLevel.HIGH }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(8.dp),
        color = when {
            criticalCount > 0 -> Color(0xFFF44336).copy(alpha = 0.1f)
            highRiskCount > 0 -> Color(0xFFFF9800).copy(alpha = 0.1f)
            else -> MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
        }
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            // Capability icon
            Icon(
                imageVector = Icons.Default.Build,
                contentDescription = null,
                modifier = Modifier.size(14.dp),
                tint = when {
                    criticalCount > 0 -> Color(0xFFF44336)
                    highRiskCount > 0 -> Color(0xFFFF9800)
                    else -> MaterialTheme.colorScheme.onSurfaceVariant
                }
            )

            // Count
            Text(
                text = if (activeCount > 0) "$activeCount/${capabilities.size}" else "${capabilities.size}",
                style = MaterialTheme.typography.labelSmall,
                color = when {
                    criticalCount > 0 -> Color(0xFFF44336)
                    highRiskCount > 0 -> Color(0xFFFF9800)
                    else -> MaterialTheme.colorScheme.onSurfaceVariant
                }
            )

            // Risk indicator
            if (criticalCount > 0) {
                Box(
                    modifier = Modifier
                        .size(6.dp)
                        .background(Color(0xFFF44336), CircleShape)
                )
            } else if (highRiskCount > 0) {
                Box(
                    modifier = Modifier
                        .size(6.dp)
                        .background(Color(0xFFFF9800), CircleShape)
                )
            }
        }
    }
}

/**
 * Capability Summary Panel
 *
 * Vertical panel showing capability categories with counts
 */
@Composable
fun CapabilitySummaryPanel(
    capabilities: List<Capability>,
    modifier: Modifier = Modifier
) {
    val categories = capabilities.groupBy { it.category }

    Column(
        modifier = modifier,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = "Capabilities",
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold
        )

        categories.forEach { (category, caps) ->
            CategoryRow(
                category = category,
                capabilities = caps
            )
        }
    }
}

@Composable
private fun CategoryRow(
    category: CapabilityCategory,
    capabilities: List<Capability>
) {
    val highestRisk = capabilities.maxByOrNull { it.riskLevel.ordinal }?.riskLevel ?: RiskLevel.LOW

    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .background(getRiskColor(highestRisk), CircleShape)
            )
            Text(
                text = formatCategory(category),
                style = MaterialTheme.typography.bodyMedium
            )
        }

        Surface(
            shape = RoundedCornerShape(12.dp),
            color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
        ) {
            Text(
                text = "${capabilities.size}",
                style = MaterialTheme.typography.labelSmall,
                modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
            )
        }
    }
}

// Helper functions

private fun getRiskColor(riskLevel: RiskLevel): Color {
    return when (riskLevel) {
        RiskLevel.LOW -> Color(0xFF4CAF50)
        RiskLevel.MEDIUM -> Color(0xFFFFC107)
        RiskLevel.HIGH -> Color(0xFFFF9800)
        RiskLevel.CRITICAL -> Color(0xFFF44336)
    }
}

private fun formatCategory(category: CapabilityCategory): String {
    return when (category) {
        CapabilityCategory.COMMUNICATION -> "Communication"
        CapabilityCategory.DATA_ACCESS -> "Data Access"
        CapabilityCategory.DATA_MODIFY -> "Data Modification"
        CapabilityCategory.EXTERNAL -> "External APIs"
        CapabilityCategory.SYSTEM -> "System"
        CapabilityCategory.WORKFLOW -> "Workflow"
    }
}

/**
 * Preview
 */
@Composable
fun CapabilityRibbonPreview() {
    val sampleCapabilities = listOf(
        Capability(
            id = "cap_1",
            name = "calendar_read",
            displayName = "Calendar Read",
            description = "Read calendar events",
            category = CapabilityCategory.DATA_ACCESS,
            riskLevel = RiskLevel.LOW,
            requiresApproval = false
        ),
        Capability(
            id = "cap_2",
            name = "calendar_write",
            displayName = "Calendar Write",
            description = "Create and modify calendar events",
            category = CapabilityCategory.DATA_MODIFY,
            riskLevel = RiskLevel.MEDIUM,
            requiresApproval = true
        ),
        Capability(
            id = "cap_3",
            name = "send_email",
            displayName = "Send Email",
            description = "Send emails on your behalf",
            category = CapabilityCategory.COMMUNICATION,
            riskLevel = RiskLevel.HIGH,
            requiresApproval = true
        ),
        Capability(
            id = "cap_4",
            name = "payment",
            displayName = "Payments",
            description = "Process payments",
            category = CapabilityCategory.EXTERNAL,
            riskLevel = RiskLevel.CRITICAL,
            requiresApproval = true
        )
    )

    MaterialTheme {
        Surface(modifier = Modifier.padding(16.dp)) {
            Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                CapabilityRibbon(
                    capabilities = sampleCapabilities,
                    activeCapabilities = setOf("cap_2", "cap_3")
                )

                Divider()

                CapabilityIndicator(
                    capabilities = sampleCapabilities,
                    activeCount = 2
                )

                Divider()

                CapabilitySummaryPanel(
                    capabilities = sampleCapabilities,
                    modifier = Modifier.fillMaxWidth()
                )
            }
        }
    }
}
