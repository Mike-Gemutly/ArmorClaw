package components.governor

import androidx.compose.animation.animateColorAsState
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import components.vault.VaultPulseIndicator
import components.vault.VaultStatus

/**
 * Command Block Component
 *
 * Displays an action card for agent commands.
 * Replaces traditional message bubbles with actionable cards.
 *
 * Phase 2 Implementation - Governor Strategy
 *
 * @param command The command to display
 * @param onApprove Callback when user approves the command
 * @param onReject Callback when user rejects the command
 * @param onCancel Callback when user cancels an executing command
 * @param modifier Optional modifier
 */
@Composable
fun CommandBlockCard(
    command: CommandBlock,
    onApprove: () -> Unit = {},
    onReject: () -> Unit = {},
    onCancel: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val statusColor by animateColorAsState(
        targetValue = getStatusColor(command.status),
        animationSpec = tween(300),
        label = "status_color"
    )

    Card(
        modifier = modifier
            .fillMaxWidth()
            .border(
                width = 1.dp,
                color = statusColor.copy(alpha = 0.3f),
                shape = RoundedCornerShape(12.dp)
            ),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        ) {
            // Header: Agent + Status
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                // Agent info
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    // Agent avatar
                    Box(
                        modifier = Modifier
                            .size(32.dp)
                            .background(
                                color = MaterialTheme.colorScheme.primaryContainer,
                                shape = CircleShape
                            ),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = command.agentName.firstOrNull()?.uppercase() ?: "?",
                            style = MaterialTheme.typography.labelLarge,
                            color = MaterialTheme.colorScheme.onPrimaryContainer
                        )
                    }

                    Column {
                        Text(
                            text = command.agentName,
                            style = MaterialTheme.typography.labelLarge,
                            fontWeight = FontWeight.Medium
                        )
                        Text(
                            text = formatCommandType(command.commandType),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }

                // Status indicator
                CommandStatusBadge(status = command.status)
            }

            Spacer(modifier = Modifier.height(12.dp))

            // Title
            Text(
                text = command.title,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis
            )

            // Description
            if (command.description.isNotBlank()) {
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = command.description,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis
                )
            }

            // Required PII keys indicator
            if (command.requiredPiiKeys.isNotEmpty()) {
                Spacer(modifier = Modifier.height(12.dp))
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    VaultPulseIndicator(
                        isActive = true,
                        requiredKeyCount = command.requiredPiiKeys.size,
                        modifier = Modifier.size(20.dp)
                    )
                    Text(
                        text = "Requires ${command.requiredPiiKeys.size} data ${if (command.requiredPiiKeys.size == 1) "key" else "keys"}",
                        style = MaterialTheme.typography.labelMedium,
                        color = Color(0xFF00BCD4)
                    )
                }
            }

            // Required capabilities
            if (command.requiredCapabilities.isNotEmpty()) {
                Spacer(modifier = Modifier.height(12.dp))
                CapabilityChips(
                    capabilities = command.requiredCapabilities,
                    modifier = Modifier.fillMaxWidth()
                )
            }

            // Result/Error
            command.result?.let { result ->
                Spacer(modifier = Modifier.height(12.dp))
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(8.dp),
                    color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
                ) {
                    Text(
                        text = result,
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.padding(12.dp)
                    )
                }
            }

            command.error?.let { error ->
                Spacer(modifier = Modifier.height(12.dp))
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(8.dp),
                    color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.3f)
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Warning,
                            contentDescription = "Error",
                            tint = MaterialTheme.colorScheme.error,
                            modifier = Modifier.size(16.dp)
                        )
                        Text(
                            text = error,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.error
                        )
                    }
                }
            }

            // Action buttons
            if (command.status == CommandStatus.PENDING) {
                Spacer(modifier = Modifier.height(16.dp))
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    OutlinedButton(
                        onClick = onReject,
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.outlinedButtonColors(
                            contentColor = MaterialTheme.colorScheme.error
                        )
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Reject")
                    }
                    Button(
                        onClick = onApprove,
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.buttonColors(
                            containerColor = Color(0xFF00BCD4)
                        )
                    ) {
                        Icon(
                            imageVector = Icons.Default.Check,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Approve")
                    }
                }
            }

            if (command.status == CommandStatus.EXECUTING) {
                Spacer(modifier = Modifier.height(16.dp))
                OutlinedButton(
                    onClick = onCancel,
                    modifier = Modifier.fillMaxWidth(),
                    colors = ButtonDefaults.outlinedButtonColors(
                        contentColor = MaterialTheme.colorScheme.error
                    )
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp)
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Cancel")
                }
            }
        }
    }
}

/**
 * Command Status Badge
 */
@Composable
fun CommandStatusBadge(
    status: CommandStatus,
    modifier: Modifier = Modifier
) {
    val (color, label, icon) = when (status) {
        CommandStatus.PENDING -> Triple(Color(0xFFFFC107), "Pending", Icons.Default.DateRange)
        CommandStatus.APPROVED -> Triple(Color(0xFF4CAF50), "Approved", Icons.Default.Check)
        CommandStatus.EXECUTING -> Triple(Color(0xFF00BCD4), "Executing", Icons.Default.Refresh)
        CommandStatus.COMPLETED -> Triple(Color(0xFF4CAF50), "Completed", Icons.Default.CheckCircle)
        CommandStatus.FAILED -> Triple(Color(0xFFF44336), "Failed", Icons.Default.Warning)
        CommandStatus.CANCELLED -> Triple(Color(0xFF9E9E9E), "Cancelled", Icons.Default.Close)
    }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
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
                modifier = Modifier.size(14.dp),
                tint = color
            )
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                color = color
            )
        }
    }
}

/**
 * Capability Chips
 */
@Composable
private fun CapabilityChips(
    capabilities: List<Capability>,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        capabilities.take(3).forEach { capability ->
            Surface(
                shape = RoundedCornerShape(12.dp),
                color = getRiskColor(capability.riskLevel).copy(alpha = 0.1f)
            ) {
                Text(
                    text = capability.displayName,
                    style = MaterialTheme.typography.labelSmall,
                    color = getRiskColor(capability.riskLevel),
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                )
            }
        }
        if (capabilities.size > 3) {
            Surface(
                shape = RoundedCornerShape(12.dp),
                color = MaterialTheme.colorScheme.surfaceVariant
            ) {
                Text(
                    text = "+${capabilities.size - 3}",
                    style = MaterialTheme.typography.labelSmall,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                )
            }
        }
    }
}

// Helper functions

private fun getStatusColor(status: CommandStatus): Color {
    return when (status) {
        CommandStatus.PENDING -> Color(0xFFFFC107)
        CommandStatus.APPROVED -> Color(0xFF4CAF50)
        CommandStatus.EXECUTING -> Color(0xFF00BCD4)
        CommandStatus.COMPLETED -> Color(0xFF4CAF50)
        CommandStatus.FAILED -> Color(0xFFF44336)
        CommandStatus.CANCELLED -> Color(0xFF9E9E9E)
    }
}

private fun getRiskColor(riskLevel: RiskLevel): Color {
    return when (riskLevel) {
        RiskLevel.LOW -> Color(0xFF4CAF50)
        RiskLevel.MEDIUM -> Color(0xFFFFC107)
        RiskLevel.HIGH -> Color(0xFFFF9800)
        RiskLevel.CRITICAL -> Color(0xFFF44336)
    }
}

private fun formatCommandType(type: CommandType): String {
    return when (type) {
        CommandType.MESSAGE -> "Message"
        CommandType.ACTION -> "Action"
        CommandType.QUERY -> "Query"
        CommandType.SYSTEM -> "System"
        CommandType.APPROVAL_REQUIRED -> "Approval Required"
        CommandType.EXECUTING -> "Executing"
        CommandType.COMPLETED -> "Completed"
        CommandType.FAILED -> "Failed"
    }
}

/**
 * Preview
 */
@Composable
fun CommandBlockCardPreview() {
    val sampleCommand = CommandBlock(
        id = "cmd_1",
        agentId = "agent_1",
        agentName = "Calendar Agent",
        commandType = CommandType.ACTION,
        title = "Schedule Meeting",
        description = "Create a new meeting with the marketing team for Thursday at 2pm",
        status = CommandStatus.PENDING,
        requiredCapabilities = listOf(
            Capability(
                id = "cap_1",
                name = "calendar_write",
                displayName = "Calendar Write",
                description = "Create and modify calendar events",
                category = CapabilityCategory.DATA_MODIFY,
                riskLevel = RiskLevel.MEDIUM,
                requiresApproval = true
            )
        ),
        requiredPiiKeys = listOf("email", "full_name"),
        createdAt = System.currentTimeMillis(),
        isApproved = false
    )

    MaterialTheme {
        Surface(modifier = Modifier.padding(16.dp)) {
            CommandBlockCard(command = sampleCommand)
        }
    }
}
