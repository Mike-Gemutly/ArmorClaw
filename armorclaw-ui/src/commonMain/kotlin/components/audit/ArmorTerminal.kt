package components.audit

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
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
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import components.governor.RiskLevel
import kotlinx.coroutines.launch

/**
 * Armor Terminal
 *
 * Real-time audit log display showing all agent activity.
 * Terminal-style interface with color-coded entries.
 *
 * Phase 3 Implementation - Governor Strategy
 *
 * @param receipts List of task receipts to display
 * @param onReceiptClick Callback when a receipt is clicked
 * @param onRevoke Callback when user requests revocation
 * @param isLive Whether to show live indicator
 * @param modifier Optional modifier
 */
@Composable
fun ArmorTerminal(
    receipts: List<TaskReceipt>,
    onReceiptClick: (TaskReceipt) -> Unit = {},
    onRevoke: (TaskReceipt) -> Unit = {},
    isLive: Boolean = true,
    modifier: Modifier = Modifier
) {
    val listState = rememberLazyListState()
    val scope = rememberCoroutineScope()

    // Auto-scroll to bottom when new receipts arrive
    LaunchedEffect(receipts.size) {
        if (receipts.isNotEmpty()) {
            scope.launch {
                listState.animateScrollToItem(receipts.size - 1)
            }
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(Color(0xFF0D1117)) // Terminal background
    ) {
        // Terminal header
        TerminalHeader(
            receiptCount = receipts.size,
            isLive = isLive
        )

        Divider(color = Color(0xFF30363D), thickness = 1.dp)

        // Receipt list
        LazyColumn(
            state = listState,
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp)
        ) {
            items(receipts, key = { it.id }) { receipt ->
                ReceiptEntry(
                    receipt = receipt,
                    onClick = { onReceiptClick(receipt) },
                    onRevoke = { onRevoke(receipt) }
                )
            }

            // Empty state
            if (receipts.isEmpty()) {
                item {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(32.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = "No activity recorded",
                            style = MaterialTheme.typography.bodyMedium,
                            color = Color(0xFF8B949E),
                            fontFamily = FontFamily.Monospace
                        )
                    }
                }
            }
        }
    }
}

/**
 * Terminal Header
 */
@Composable
private fun TerminalHeader(
    receiptCount: Int,
    isLive: Boolean
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(Color(0xFF161B22))
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Terminal icon
            Icon(
                imageVector = Icons.Default.Info,
                contentDescription = null,
                tint = Color(0xFF58A6FF),
                modifier = Modifier.size(20.dp)
            )

            Text(
                text = "Armor Terminal",
                style = MaterialTheme.typography.titleSmall,
                color = Color.White,
                fontWeight = FontWeight.Bold
            )

            // Receipt count
            Surface(
                shape = RoundedCornerShape(12.dp),
                color = Color(0xFF30363D)
            ) {
                Text(
                    text = "$receiptCount entries",
                    style = MaterialTheme.typography.labelSmall,
                    color = Color(0xFF8B949E),
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                )
            }
        }

        // Live indicator
        if (isLive) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(6.dp)
            ) {
                PulsingDot(color = Color(0xFF3FB950))
                Text(
                    text = "LIVE",
                    style = MaterialTheme.typography.labelSmall,
                    color = Color(0xFF3FB950),
                    fontWeight = FontWeight.Bold
                )
            }
        }
    }
}

/**
 * Receipt Entry
 *
 * Single line in the terminal showing a task receipt
 */
@Composable
private fun ReceiptEntry(
    receipt: TaskReceipt,
    onClick: () -> Unit,
    onRevoke: () -> Unit,
    modifier: Modifier = Modifier
) {
    var expanded by remember { mutableStateOf(false) }

    val statusColor = getStatusColor(receipt.status)
    val riskColor = getRiskColor(receipt.riskLevel)
    val timestamp = formatTimestamp(receipt.timestamp)

    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(4.dp))
            .background(
                if (expanded) Color(0xFF1C2128) else Color.Transparent
            )
    ) {
        // Main entry line
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            // Timestamp
            Text(
                text = timestamp,
                style = MaterialTheme.typography.labelSmall,
                color = Color(0xFF6E7681),
                fontFamily = FontFamily.Monospace,
                modifier = Modifier.width(80.dp)
            )

            // Status indicator
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .background(statusColor, CircleShape)
            )

            // Risk level indicator
            Box(
                modifier = Modifier
                    .width(3.dp)
                    .height(16.dp)
                    .background(riskColor, RoundedCornerShape(1.dp))
            )

            // Agent name
            Text(
                text = "[${receipt.agentName}]",
                style = MaterialTheme.typography.labelSmall,
                color = Color(0xFF58A6FF),
                fontFamily = FontFamily.Monospace,
                modifier = Modifier.widthIn(max = 100.dp),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            // Action
            Text(
                text = receipt.action,
                style = MaterialTheme.typography.bodySmall,
                color = Color(0xFFE6EDF3),
                fontFamily = FontFamily.Monospace,
                modifier = Modifier.weight(1f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            // Expand/collapse button
            IconButton(
                onClick = { expanded = !expanded },
                modifier = Modifier.size(24.dp)
            ) {
                Icon(
                    imageVector = if (expanded) Icons.Default.KeyboardArrowUp else Icons.Default.KeyboardArrowDown,
                    contentDescription = if (expanded) "Collapse" else "Expand",
                    tint = Color(0xFF8B949E),
                    modifier = Modifier.size(16.dp)
                )
            }
        }

        // Expanded details
        AnimatedVisibility(visible = expanded) {
            ReceiptDetails(
                receipt = receipt,
                onRevoke = onRevoke,
                modifier = Modifier.padding(start = 100.dp, end = 8.dp, bottom = 8.dp)
            )
        }
    }
}

/**
 * Receipt Details
 *
 * Expanded view showing full receipt details
 */
@Composable
private fun ReceiptDetails(
    receipt: TaskReceipt,
    onRevoke: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(Color(0xFF21262D), RoundedCornerShape(4.dp))
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // Action type and status
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            DetailItem(label = "Type", value = receipt.actionType.name)
            DetailItem(label = "Status", value = receipt.status.name)
            DetailItem(label = "Risk", value = receipt.riskLevel.name)
        }

        // Duration
        receipt.duration?.let {
            DetailItem(label = "Duration", value = "${it}ms")
        }

        // Capabilities used
        if (receipt.capabilities.isNotEmpty()) {
            Text(
                text = "Capabilities:",
                style = MaterialTheme.typography.labelSmall,
                color = Color(0xFF8B949E)
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                receipt.capabilities.forEach { cap ->
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = Color(0xFF30363D)
                    ) {
                        Text(
                            text = cap.capabilityName,
                            style = MaterialTheme.typography.labelSmall,
                            color = Color(0xFF58A6FF),
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }
            }
        }

        // PII accessed
        if (receipt.piiAccessed.isNotEmpty()) {
            Text(
                text = "PII Accessed:",
                style = MaterialTheme.typography.labelSmall,
                color = Color(0xFF8B949E)
            )
            receipt.piiAccessed.forEach { pii ->
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Lock,
                        contentDescription = null,
                        tint = Color(0xFF3FB950),
                        modifier = Modifier.size(12.dp)
                    )
                    Text(
                        text = "${pii.displayName} (${pii.accessType.name})",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color(0xFFE6EDF3)
                    )
                }
            }
        }

        // Error message
        receipt.errorMessage?.let { error ->
            Surface(
                shape = RoundedCornerShape(4.dp),
                color = Color(0xFFF85149).copy(alpha = 0.1f)
            ) {
                Row(
                    modifier = Modifier.padding(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Warning,
                        contentDescription = null,
                        tint = Color(0xFFF85149),
                        modifier = Modifier.size(14.dp)
                    )
                    Text(
                        text = error,
                        style = MaterialTheme.typography.labelSmall,
                        color = Color(0xFFF85149)
                    )
                }
            }
        }

        // Revoke button
        if (receipt.revocable && !receipt.revoked && receipt.status == TaskStatus.COMPLETED) {
            OutlinedButton(
                onClick = onRevoke,
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = Color(0xFFF85149)
                ),
                border = androidx.compose.foundation.BorderStroke(1.dp, Color(0xFFF85149))
            ) {
                Icon(
                    imageVector = Icons.Default.Refresh,
                    contentDescription = null,
                    modifier = Modifier.size(16.dp)
                )
                Spacer(modifier = Modifier.width(4.dp))
                Text("Revoke")
            }
        }

        // Revocation info
        if (receipt.revoked) {
            Surface(
                shape = RoundedCornerShape(4.dp),
                color = Color(0xFFF85149).copy(alpha = 0.1f)
            ) {
                Row(
                    modifier = Modifier.padding(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = null,
                        tint = Color(0xFFF85149),
                        modifier = Modifier.size(14.dp)
                    )
                    Text(
                        text = "Revoked by ${receipt.revokedBy} at ${formatTimestamp(receipt.revokedAt ?: 0)}",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color(0xFFF85149)
                    )
                }
            }
        }
    }
}

@Composable
private fun DetailItem(label: String, value: String) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Text(
            text = "$label:",
            style = MaterialTheme.typography.labelSmall,
            color = Color(0xFF8B949E)
        )
        Text(
            text = value,
            style = MaterialTheme.typography.labelSmall,
            color = Color(0xFFE6EDF3)
        )
    }
}

/**
 * Pulsing dot animation
 */
@Composable
private fun PulsingDot(color: Color) {
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

    Box(
        modifier = Modifier
            .size(8.dp)
            .background(color.copy(alpha = alpha), CircleShape)
    )
}

// Helper functions

private fun getStatusColor(status: TaskStatus): Color {
    return when (status) {
        TaskStatus.PENDING -> Color(0xFFFFC107)
        TaskStatus.APPROVED -> Color(0xFF3FB950)
        TaskStatus.EXECUTING -> Color(0xFF58A6FF)
        TaskStatus.COMPLETED -> Color(0xFF3FB950)
        TaskStatus.FAILED -> Color(0xFFF85149)
        TaskStatus.CANCELLED -> Color(0xFF8B949E)
        TaskStatus.REVOKED -> Color(0xFFF85149)
    }
}

private fun getRiskColor(riskLevel: RiskLevel): Color {
    return when (riskLevel) {
        RiskLevel.LOW -> Color(0xFF3FB950)
        RiskLevel.MEDIUM -> Color(0xFFFFC107)
        RiskLevel.HIGH -> Color(0xFFFF9800)
        RiskLevel.CRITICAL -> Color(0xFFF85149)
    }
}

private fun formatTimestamp(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

    return when {
        diff < 1000 -> "now"
        diff < 60000 -> "${diff / 1000}s ago"
        diff < 3600000 -> "${diff / 60000}m ago"
        diff < 86400000 -> "${diff / 3600000}h ago"
        else -> {
            val date = java.text.SimpleDateFormat("HH:mm:ss").format(java.util.Date(timestamp))
            date
        }
    }
}

/**
 * Preview
 */
@Composable
fun ArmorTerminalPreview() {
    val sampleReceipts = listOf(
        TaskReceipt(
            id = "r1",
            taskId = "t1",
            agentId = "agent_1",
            agentName = "CalendarAgent",
            action = "Created meeting 'Weekly Sync'",
            actionType = ActionType.WRITE,
            status = TaskStatus.COMPLETED,
            timestamp = System.currentTimeMillis() - 5000,
            duration = 234,
            capabilities = listOf(
                CapabilityUsage("c1", "calendar_write", System.currentTimeMillis() - 5000, 100)
            ),
            piiAccessed = listOf(
                PiiAccess("p1", "email", "Email", PiiAccessType.READ, System.currentTimeMillis() - 5000, "Send invite")
            ),
            riskLevel = RiskLevel.MEDIUM,
            revocable = true
        ),
        TaskReceipt(
            id = "r2",
            taskId = "t2",
            agentId = "agent_2",
            agentName = "EmailAgent",
            action = "Failed to send email",
            actionType = ActionType.COMMUNICATE,
            status = TaskStatus.FAILED,
            timestamp = System.currentTimeMillis() - 30000,
            errorMessage = "SMTP connection timeout",
            capabilities = emptyList(),
            piiAccessed = emptyList(),
            riskLevel = RiskLevel.HIGH,
            revocable = false
        )
    )

    MaterialTheme {
        ArmorTerminal(
            receipts = sampleReceipts,
            isLive = true,
            modifier = Modifier
                .fillMaxSize()
                .width(400.dp)
                .height(500.dp)
        )
    }
}
