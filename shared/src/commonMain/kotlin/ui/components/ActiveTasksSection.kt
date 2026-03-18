package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.data.store.AgentTaskState
import com.armorclaw.shared.domain.model.AgentTaskStatus
import com.armorclaw.shared.domain.model.AgentSummary
import com.armorclaw.shared.ui.theme.AppIcons

/**
 * Active Tasks Section
 *
 * Displays a horizontal scrolling list of active agent tasks in the Mission Control Dashboard.
 * Each task shows the agent name, task type, and progress.
 *
 * ## Architecture
 * ```
 * ActiveTasksSection
 *      ├── Section header
 *      └── LazyRow of ActiveTaskCard[]
 *          ├── Agent icon
 *          ├── Task name and room
 *          └── Progress indicator
 * ```
 *
 * ## Usage
 * ```kotlin
 * ActiveTasksSection(
 *     activeTasks = activeAgentSummaries,
 *     onTaskClick = { task -> viewModel.onTaskClick(task.agentId) },
 *     modifier = Modifier.padding(vertical = 8.dp)
 * )
 * ```
 */
@Composable
fun ActiveTasksSection(
    activeTasks: List<AgentSummary>,
    modifier: Modifier = Modifier,
    onTaskClick: (AgentSummary) -> Unit = {},
    maxVisible: Int = 5
) {
    AnimatedVisibility(
        visible = activeTasks.isNotEmpty(),
        enter = fadeIn() + slideInVertically(),
        exit = fadeOut() + slideOutVertically()
    ) {
        Column(modifier = modifier) {
            // Section header
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        imageVector = Icons.Default.SmartToy,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(20.dp)
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = "Active Agents",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold
                    )
                    if (activeTasks.size > maxVisible) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = MaterialTheme.colorScheme.primaryContainer
                        ) {
                            Text(
                                text = activeTasks.size.toString(),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                            )
                        }
                    }
                }

                if (activeTasks.size > maxVisible) {
                    TextButton(onClick = { /* Navigate to full list */ }) {
                        Text("See all")
                    }
                }
            }

            // Task cards
            LazyRow(
                contentPadding = PaddingValues(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                items(
                    items = activeTasks.take(maxVisible),
                    key = { it.agentId }
                ) { task ->
                    ActiveTaskCard(
                        task = task,
                        onClick = { onTaskClick(task) },
                        modifier = Modifier.width(280.dp)
                    )
                }
            }
        }
    }
}

/**
 * Individual task card with progress indicator
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ActiveTaskCard(
    task: AgentSummary,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val containerColor = when (task.status) {
        AgentTaskStatus.BROWSING -> MaterialTheme.colorScheme.primaryContainer
        AgentTaskStatus.FORM_FILLING -> MaterialTheme.colorScheme.tertiaryContainer
        AgentTaskStatus.PROCESSING_PAYMENT -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
        AgentTaskStatus.AWAITING_CAPTCHA,
        AgentTaskStatus.AWAITING_2FA,
        AgentTaskStatus.AWAITING_APPROVAL -> MaterialTheme.colorScheme.secondaryContainer
        AgentTaskStatus.ERROR -> MaterialTheme.colorScheme.errorContainer
        AgentTaskStatus.COMPLETE -> MaterialTheme.colorScheme.tertiaryContainer
        AgentTaskStatus.IDLE -> MaterialTheme.colorScheme.surfaceVariant
    }

    val contentColor = when (task.status) {
        AgentTaskStatus.BROWSING -> MaterialTheme.colorScheme.onPrimaryContainer
        AgentTaskStatus.FORM_FILLING -> MaterialTheme.colorScheme.onTertiaryContainer
        AgentTaskStatus.PROCESSING_PAYMENT -> MaterialTheme.colorScheme.onErrorContainer
        AgentTaskStatus.AWAITING_CAPTCHA,
        AgentTaskStatus.AWAITING_2FA,
        AgentTaskStatus.AWAITING_APPROVAL -> MaterialTheme.colorScheme.onSecondaryContainer
        AgentTaskStatus.ERROR -> MaterialTheme.colorScheme.onErrorContainer
        AgentTaskStatus.COMPLETE -> MaterialTheme.colorScheme.onTertiaryContainer
        AgentTaskStatus.IDLE -> MaterialTheme.colorScheme.onSurfaceVariant
    }

    Card(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = containerColor,
            contentColor = contentColor
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            // Header row
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Agent avatar
                Surface(
                    shape = RoundedCornerShape(10.dp),
                    color = contentColor.copy(alpha = 0.2f)
                ) {
                    Box(
                        modifier = Modifier.padding(8.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        if (task.status.isActive()) {
                            PulsingAgentIcon(
                                icon = getTaskIcon(task.status),
                                color = contentColor
                            )
                        } else {
                            Icon(
                                imageVector = getTaskIcon(task.status),
                                contentDescription = null,
                                tint = contentColor,
                                modifier = Modifier.size(20.dp)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.width(10.dp))

                // Task info
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = task.agentName,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    task.roomName?.let { room ->
                        Text(
                            text = room,
                            style = MaterialTheme.typography.bodySmall,
                            color = contentColor.copy(alpha = 0.7f),
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                }

                // Status indicator
                StatusIndicator(
                    status = task.status,
                    color = contentColor
                )
            }

            // Task description
            task.currentTask?.let { currentTask ->
                Spacer(modifier = Modifier.height(12.dp))
                Text(
                    text = currentTask,
                    style = MaterialTheme.typography.bodyMedium,
                    color = contentColor,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }

            // Progress bar
            if (task.progress > 0f && task.status.isActive()) {
                Spacer(modifier = Modifier.height(12.dp))
                LinearProgressIndicator(
                    progress = task.progress,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(4.dp)
                        .clip(RoundedCornerShape(2.dp)),
                    color = contentColor,
                    trackColor = contentColor.copy(alpha = 0.2f)
                )
            }
        }
    }
}

/**
 * Get icon for task status
 */
private fun getTaskIcon(status: AgentTaskStatus) = when (status) {
    AgentTaskStatus.IDLE -> Icons.Default.SmartToy
    AgentTaskStatus.BROWSING -> Icons.Default.Language
    AgentTaskStatus.FORM_FILLING -> Icons.Default.Edit
    AgentTaskStatus.PROCESSING_PAYMENT -> Icons.Default.Payment
    AgentTaskStatus.AWAITING_CAPTCHA -> Icons.Default.Security
    AgentTaskStatus.AWAITING_2FA -> Icons.Default.Key
    AgentTaskStatus.AWAITING_APPROVAL -> Icons.Default.Pending
    AgentTaskStatus.ERROR -> AppIcons.Error
    AgentTaskStatus.COMPLETE -> Icons.Default.CheckCircle
}

/**
 * Animated pulsing icon for active tasks
 */
@Composable
private fun PulsingAgentIcon(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    color: androidx.compose.ui.graphics.Color,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.6f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(600, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Icon(
        imageVector = icon,
        contentDescription = null,
        tint = color.copy(alpha = alpha),
        modifier = modifier.size(20.dp)
    )
}

/**
 * Small status indicator dot
 */
@Composable
private fun StatusIndicator(
    status: AgentTaskStatus,
    color: androidx.compose.ui.graphics.Color,
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
            .size(8.dp)
            .clip(RoundedCornerShape(percent = 50))
            .background(color.copy(alpha = alpha))
    )
}

/**
 * Preview helper
 */
@Composable
fun ActiveTasksSectionPreview() {
    val sampleTasks = listOf(
        AgentSummary(
            agentId = "agent_1",
            agentName = "Checkout Bot",
            status = AgentTaskStatus.BROWSING,
            currentTask = "Browsing example.com",
            roomId = "!room1:example.com",
            roomName = "Shopping",
            progress = 0.3f,
            lastActivity = System.currentTimeMillis()
        ),
        AgentSummary(
            agentId = "agent_2",
            agentName = "Form Filler",
            status = AgentTaskStatus.FORM_FILLING,
            currentTask = "Filling shipping form",
            roomId = "!room2:example.com",
            roomName = "Travel Booking",
            progress = 0.6f,
            lastActivity = System.currentTimeMillis()
        ),
        AgentSummary(
            agentId = "agent_3",
            agentName = "Payment Bot",
            status = AgentTaskStatus.PROCESSING_PAYMENT,
            currentTask = "Processing payment...",
            roomId = "!room3:example.com",
            roomName = "Bills",
            progress = 0.8f,
            lastActivity = System.currentTimeMillis()
        )
    )

    ActiveTasksSection(
        activeTasks = sampleTasks,
        onTaskClick = {}
    )
}
