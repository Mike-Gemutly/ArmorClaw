package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.platform.matrix.event.StepStatus
import com.armorclaw.shared.ui.theme.AppIcons

/**
 * Workflow Card
 *
 * Displays an active workflow in the HomeScreen.
 * Shows workflow type, current step, progress, and status.
 *
 * ## Usage
 * ```kotlin
 * val workflows by viewModel.activeWorkflows.collectAsState()
 *
 * LazyColumn {
 *     items(workflows) { workflow ->
 *         WorkflowCard(
 *             workflow = workflow,
 *             onClick = { navController.navigate("workflow/${workflow.workflowId}") }
 *         )
 *     }
 * }
 * ```
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WorkflowCard(
    workflow: WorkflowState,
    modifier: Modifier = Modifier,
    onClick: () -> Unit = {},
    onCancel: (() -> Unit)? = null,
    showRoomName: Boolean = false,
    roomName: String? = null
) {
    val containerColor = when (workflow) {
        is WorkflowState.Started -> MaterialTheme.colorScheme.primaryContainer
        is WorkflowState.StepRunning -> {
            when ((workflow as WorkflowState.StepRunning).status) {
                StepStatus.RUNNING -> MaterialTheme.colorScheme.primaryContainer
                StepStatus.COMPLETED -> MaterialTheme.colorScheme.tertiaryContainer
                StepStatus.FAILED -> MaterialTheme.colorScheme.errorContainer
                else -> MaterialTheme.colorScheme.surfaceVariant
            }
        }
    }

    val contentColor = when (workflow) {
        is WorkflowState.Started -> MaterialTheme.colorScheme.onPrimaryContainer
        is WorkflowState.StepRunning -> {
            when ((workflow as WorkflowState.StepRunning).status) {
                StepStatus.RUNNING -> MaterialTheme.colorScheme.onPrimaryContainer
                StepStatus.COMPLETED -> MaterialTheme.colorScheme.onTertiaryContainer
                StepStatus.FAILED -> MaterialTheme.colorScheme.onErrorContainer
                else -> MaterialTheme.colorScheme.onSurfaceVariant
            }
        }
    }

    Card(
        onClick = onClick,
        modifier = modifier.fillMaxWidth(),
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
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Workflow type icon
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = contentColor.copy(alpha = 0.15f)
                ) {
                    Icon(
                        imageVector = getWorkflowIcon(workflow.workflowType),
                        contentDescription = null,
                        tint = contentColor,
                        modifier = Modifier.padding(10.dp).size(24.dp)
                    )
                }

                Spacer(modifier = Modifier.width(12.dp))

                // Title and subtitle
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = getWorkflowTitle(workflow),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    if (showRoomName && roomName != null) {
                        Text(
                            text = roomName,
                            style = MaterialTheme.typography.bodySmall,
                            color = contentColor.copy(alpha = 0.7f),
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                }

                // Cancel button
                if (onCancel != null) {
                    IconButton(
                        onClick = onCancel,
                        modifier = Modifier.size(36.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Cancel workflow",
                            tint = contentColor.copy(alpha = 0.7f),
                            modifier = Modifier.size(20.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // Progress section
            when (workflow) {
                is WorkflowState.Started -> {
                    // Show starting state
                    Row(
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        PulsingDot(
                            color = contentColor,
                            modifier = Modifier.size(8.dp)
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "Initializing workflow...",
                            style = MaterialTheme.typography.bodyMedium,
                            color = contentColor.copy(alpha = 0.8f)
                        )
                    }
                }
                is WorkflowState.StepRunning -> {
                    val stepWorkflow = workflow as WorkflowState.StepRunning
                    WorkflowProgressSection(
                        stepName = stepWorkflow.stepName,
                        stepIndex = stepWorkflow.stepIndex,
                        totalSteps = stepWorkflow.totalSteps,
                        status = stepWorkflow.status,
                        contentColor = contentColor
                    )
                }
            }
        }
    }
}

/**
 * Progress section with step name and progress bar
 */
@Composable
private fun WorkflowProgressSection(
    stepName: String,
    stepIndex: Int,
    totalSteps: Int,
    status: StepStatus,
    contentColor: androidx.compose.ui.graphics.Color
) {
    val progress by animateFloatAsState(
        targetValue = if (totalSteps > 0) {
            (stepIndex - 1).toFloat() / totalSteps.toFloat()
        } else 0f,
        animationSpec = tween(durationMillis = 300, easing = LinearEasing),
        label = "progress"
    )

    // Step name and status
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.weight(1f)
        ) {
            when (status) {
                StepStatus.RUNNING -> PulsingDot(contentColor, Modifier.size(8.dp))
                StepStatus.COMPLETED -> Icon(
                    imageVector = Icons.Default.CheckCircle,
                    contentDescription = null,
                    tint = contentColor,
                    modifier = Modifier.size(16.dp)
                )
                StepStatus.FAILED -> Icon(
                    imageVector = AppIcons.Error,
                    contentDescription = null,
                    tint = contentColor,
                    modifier = Modifier.size(16.dp)
                )
                else -> Icon(
                    imageVector = AppIcons.Schedule,
                    contentDescription = null,
                    tint = contentColor.copy(alpha = 0.5f),
                    modifier = Modifier.size(16.dp)
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            Text(
                text = stepName,
                style = MaterialTheme.typography.bodyMedium,
                color = contentColor,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f)
            )
        }

        // Step count
        Text(
            text = "$stepIndex/$totalSteps",
            style = MaterialTheme.typography.labelMedium,
            color = contentColor.copy(alpha = 0.7f)
        )
    }

    Spacer(modifier = Modifier.height(8.dp))

    // Progress bar
    LinearProgressIndicator(
        progress = progress,
        modifier = Modifier
            .fillMaxWidth()
            .height(6.dp)
            .clip(RoundedCornerShape(3.dp)),
        color = contentColor,
        trackColor = contentColor.copy(alpha = 0.2f)
    )
}

/**
 * Pulsing dot animation
 */
@Composable
private fun PulsingDot(
    color: androidx.compose.ui.graphics.Color,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(600, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    val scale by infiniteTransition.animateFloat(
        initialValue = 0.8f,
        targetValue = 1.2f,
        animationSpec = infiniteRepeatable(
            animation = tween(600, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    Box(
        modifier = modifier
            .alpha(alpha)
            .scale(scale)
            .background(
                color = color,
                shape = RoundedCornerShape(percent = 50)
            )
    )
}

/**
 * Get workflow icon based on type
 */
private fun getWorkflowIcon(workflowType: String): androidx.compose.ui.graphics.vector.ImageVector {
    return when (workflowType.lowercase()) {
        "document_analysis", "documentanalysis" -> AppIcons.Description
        "code_review", "codereview" -> AppIcons.Code
        "data_processing", "dataprocessing" -> Icons.Default.Folder
        "report_generation", "reportgeneration" -> AppIcons.Analytics
        "meeting_summary", "meetingsummary" -> AppIcons.MeetingRoom
        "translation" -> AppIcons.Translate
        "research" -> Icons.Default.Search
        "planning" -> AppIcons.EventNote
        else -> AppIcons.AutoAwesome
    }
}

/**
 * Get workflow display title
 */
private fun getWorkflowTitle(workflow: WorkflowState): String {
    return when (workflow) {
        is WorkflowState.Started -> {
            workflow.workflowType
                .replace("_", " ")
                .split(" ")
                .joinToString(" ") { it.lowercase().replaceFirstChar { c -> c.uppercase() } }
        }
        is WorkflowState.StepRunning -> {
            workflow.workflowType
                .replace("_", " ")
                .split(" ")
                .joinToString(" ") { it.lowercase().replaceFirstChar { c -> c.uppercase() } }
        }
    }
}

/**
 * Section header for workflow list
 */
@Composable
fun WorkflowSectionHeader(
    count: Int,
    modifier: Modifier = Modifier,
    onSeeAll: (() -> Unit)? = null
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                imageVector = AppIcons.AutoAwesome,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(20.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "Active Workflows",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            if (count > 0) {
                Spacer(modifier = Modifier.width(8.dp))
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = MaterialTheme.colorScheme.primary
                ) {
                    Text(
                        text = count.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onPrimary,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                    )
                }
            }
        }

        if (onSeeAll != null) {
            TextButton(onClick = onSeeAll) {
                Text("See all")
            }
        }
    }
}

/**
 * Preview helpers
 */
@Composable
fun WorkflowCardPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        WorkflowCard(
            workflow = WorkflowState.Started(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                roomId = "!room:example.com",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            )
        )

        WorkflowCard(
            workflow = WorkflowState.StepRunning(
                workflowId = "wf_456",
                workflowType = "code_review",
                roomId = "!room:example.com",
                stepId = "step_3",
                stepName = "Analyzing code structure",
                stepIndex = 3,
                totalSteps = 5,
                status = StepStatus.RUNNING,
                timestamp = System.currentTimeMillis()
            ),
            showRoomName = true,
            roomName = "Project Alpha"
        )
    }
}
