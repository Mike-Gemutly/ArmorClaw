package com.armorclaw.shared.ui.components

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.tween
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.clickable
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Error
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Pending
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color as ComposeColor
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.text.style.TextAlign
import com.armorclaw.shared.ui.theme.AppIcons
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.StatusError
import com.armorclaw.shared.ui.theme.StatusInfo
import com.armorclaw.shared.ui.theme.StatusSuccess
import com.armorclaw.shared.ui.theme.StatusWarning
import kotlinx.coroutines.delay
import java.text.SimpleDateFormat
import java.util.*
import java.util.Date

// Data classes for ActivityLog component
data class AgentEvent(
    val id: String,
    val stepName: String,
    val status: AgentStepStatus,
    val timestamp: Long,
    val output: String = ""
)

enum class AgentStepStatus {
    RUNNING,
    COMPLETED,
    FAILED,
    PENDING,
    SKIPPED,
    CANCELLED;
    
    fun isActive(): Boolean = this == RUNNING || this == PENDING
}

/**
 * ActivityLog Component
 *
 * Displays a vertical timeline of agent activity events with status indicators.
 * Shows step name, status, timestamp, and output preview for each event.
 *
 * ## Architecture
 * ```
 * ActivityLog
 *     ├── Section header
 *     └── LazyColumn of ActivityLogItem[]
 *         ├── Status indicator (dot/line)
 *         ├── Step content
 *             ├── Step name and status icon
 *             ├── Timestamp
 *             └── Output preview
 * ```
 *
 * ## Usage
 * ```kotlin
 * val events by viewModel.agentEvents.collectAsState()
 *
 * ActivityLog(
 *     events = events,
 *     modifier = Modifier.fillMaxWidth()
 * )
 * ```
 */
@Composable
fun ActivityLog(
    events: List<AgentEvent>,
    modifier: Modifier = Modifier,
    onEventClick: (AgentEvent) -> Unit = {},
    showHeader: Boolean = true,
    autoScroll: Boolean = true
) {
    val listState = rememberLazyListState()
    
    // Auto-scroll to latest event when events change
    LaunchedEffect(events, autoScroll) {
        if (autoScroll && events.isNotEmpty()) {
            delay(100) // Small delay to ensure UI is ready
            listState.animateScrollToItem(events.size - 1)
        }
    }

    Column(modifier = modifier) {
        if (showHeader) {
            // Section header
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = DesignTokens.Spacing.lg, vertical = DesignTokens.Spacing.md),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        imageVector = Icons.Default.Info,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(20.dp)
                    )
                    Spacer(modifier = Modifier.width(DesignTokens.Spacing.md))
                    Text(
                        text = "Activity Log",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold
                    )
                }
                Text(
                    text = "${events.size} events",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }

        // Empty state
        if (events.isEmpty()) {
            EmptyState()
            return
        }

        // Timeline content
        LazyColumn(
            state = listState,
            contentPadding = PaddingValues(horizontal = DesignTokens.Spacing.lg, vertical = DesignTokens.Spacing.md),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            items(events, key = { it.id }) { event ->
                ActivityLogItem(
                    event = event,
                    events = events,
                    onClick = { onEventClick(event) }
                )
            }
        }
    }
}

/**
 * Individual timeline item
 */
@Composable
private fun ActivityLogItem(
    event: AgentEvent,
    events: List<AgentEvent>,
    onClick: () -> Unit
) {
    val localEvents = events
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(DesignTokens.Radius.md))
            .clickable(onClick = onClick)
            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f))
            .padding(DesignTokens.Spacing.md),
        verticalAlignment = Alignment.Top
    ) {
        // Status indicator column
        Column(
            modifier = Modifier.width(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Status dot
            StatusDot(
                status = event.status,
                modifier = Modifier.size(12.dp)
            )
            
            // Timeline line (for all but last item)
            if (localEvents.indexOf(event) < localEvents.size - 1) {
                Spacer(
                    modifier = Modifier
                        .fillMaxHeight()
                        .width(2.dp)
                        .background(MaterialTheme.colorScheme.outlineVariant)
                )
            }
        }

        Spacer(modifier = Modifier.width(DesignTokens.Spacing.md))

        // Content
        Column(modifier = Modifier.weight(1f)) {
            // Step name and status
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Status icon
                Icon(
                    imageVector = getStatusIcon(event.status),
                    contentDescription = "Status: ${event.status.name}",
                    tint = getStatusColor(event.status),
                    modifier = Modifier.size(16.dp)
                )
                
                Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                
                // Step name
                Text(
                    text = event.stepName,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            // Timestamp
            Text(
                text = formatTimestamp(event.timestamp),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 1
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            // Output preview
            if (event.output.isNotBlank()) {
                Text(
                    text = event.output,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }
        }
    }
}

/**
 * Status dot with animation for active states
 */
@Composable
private fun StatusDot(
    status: AgentStepStatus,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "status")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = if (status.isActive()) 1f else 0.5f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Box(
        modifier = modifier
            .clip(CircleShape)
            .background(getStatusColor(status).copy(alpha = alpha))
    )
}

/**
 * Get status icon based on status
 */
private fun getStatusIcon(status: AgentStepStatus): androidx.compose.ui.graphics.vector.ImageVector {
    return when (status) {
        AgentStepStatus.RUNNING -> Icons.Default.Pending
        AgentStepStatus.COMPLETED -> Icons.Default.CheckCircle
        AgentStepStatus.FAILED -> Icons.Default.Error
        AgentStepStatus.PENDING -> Icons.Default.Info
        AgentStepStatus.SKIPPED -> AppIcons.Schedule
        AgentStepStatus.CANCELLED -> Icons.Default.Close
        else -> Icons.Default.Info
    }
}

/**
 * Get status color based on status
 */
@Composable
private fun getStatusColor(status: AgentStepStatus): ComposeColor {
    return when (status) {
        AgentStepStatus.RUNNING -> StatusInfo
        AgentStepStatus.COMPLETED -> StatusSuccess
        AgentStepStatus.FAILED -> StatusError
        AgentStepStatus.PENDING -> StatusWarning
        AgentStepStatus.SKIPPED -> MaterialTheme.colorScheme.onSurfaceVariant
        AgentStepStatus.CANCELLED -> MaterialTheme.colorScheme.error
        else -> MaterialTheme.colorScheme.onSurfaceVariant
    }
}

/**
 * Format timestamp to readable format
 */
private fun formatTimestamp(timestamp: Long): String {
    val dateFormat = SimpleDateFormat("HH:mm:ss", Locale.getDefault())
    return dateFormat.format(Date(timestamp))
}

/**
 * Empty state for no events
 */
@Composable
private fun EmptyState() {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(DesignTokens.Spacing.xxl),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            Icon(
                imageVector = Icons.Default.Info,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(48.dp)
            )
            Text(
                text = "No activity yet",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface
            )
            Text(
                text = "Agent activity will appear here as tasks are executed",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center
            )
        }
    }
}

/**
 * Preview helper
 */
@Composable
fun ActivityLogPreview() {
    val sampleEvents = listOf(
        AgentEvent(
            id = "event_1",
            stepName = "Navigating to checkout page",
            status = AgentStepStatus.RUNNING,
            timestamp = System.currentTimeMillis() - 300000,
            output = "Loading checkout page..."
        ),
        AgentEvent(
            id = "event_2",
            stepName = "Filling shipping form",
            status = AgentStepStatus.COMPLETED,
            timestamp = System.currentTimeMillis() - 200000,
            output = "Form filled successfully"
        ),
        AgentEvent(
            id = "event_3",
            stepName = "Processing payment",
            status = AgentStepStatus.FAILED,
            timestamp = System.currentTimeMillis() - 100000,
            output = "Payment failed: Invalid card number"
        ),
        AgentEvent(
            id = "event_4",
            stepName = "Retrying payment",
            status = AgentStepStatus.PENDING,
            timestamp = System.currentTimeMillis() - 50000,
            output = "Attempting retry..."
        )
    )

    Column(verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.lg)) {
        ActivityLog(
            events = sampleEvents,
            modifier = Modifier.fillMaxWidth()
        )
        
        ActivityLog(
            events = emptyList(),
            modifier = Modifier.fillMaxWidth()
        )
    }
}

/**
 * Preview helper for dark mode
 */
@Composable
fun ActivityLogPreviewDark() {
    MaterialTheme(colorScheme = darkColorScheme()) {
        ActivityLogPreview()
    }
}