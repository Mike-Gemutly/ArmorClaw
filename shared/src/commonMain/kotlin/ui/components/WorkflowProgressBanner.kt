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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.platform.matrix.event.StepStatus
import com.armorclaw.shared.ui.theme.AppIcons
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Workflow Progress Banner
 *
 * Displays the current workflow progress in a chat screen.
 * Shows step name, progress bar, and status.
 *
 * ## Usage
 * ```kotlin
 * val workflowState by viewModel.activeWorkflow.collectAsState()
 *
 * when (workflowState) {
 *     is WorkflowState.StepRunning -> {
 *         WorkflowProgressBanner(workflowState as WorkflowState.StepRunning)
 *     }
 *     else -> {} // No banner
 * }
 * ```
 *
 * ## States
 * - Started: Shows "Starting workflow..."
 * - StepRunning: Shows step name and progress bar
 * - Completed: Brief success animation
 * - Failed: Error message with retry option
 */
@Composable
fun WorkflowProgressBanner(
    workflowState: WorkflowState,
    modifier: Modifier = Modifier,
    onDismiss: () -> Unit = {},
    onRetry: (() -> Unit)? = null
) {
    when (workflowState) {
        is WorkflowState.Started -> {
            WorkflowStartedBanner(
                state = workflowState,
                modifier = modifier
            )
        }
        is WorkflowState.StepRunning -> {
            WorkflowStepBanner(
                state = workflowState,
                modifier = modifier
            )
        }
    }
}

/**
 * Banner shown when workflow is starting
 */
@Composable
private fun WorkflowStartedBanner(
    state: WorkflowState.Started,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        color = MaterialTheme.colorScheme.primaryContainer,
        tonalElevation = 2.dp
    ) {
        Row(
            modifier = Modifier
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Animated loading indicator
            PulsingIndicator(
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(24.dp)
            )

            Spacer(modifier = Modifier.width(12.dp))

            Column {
                Text(
                    text = "Starting Workflow",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onPrimaryContainer
                )
                Text(
                    text = state.workflowType.replace("_", " ").lowercase()
                        .replaceFirstChar { it.uppercase() },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.7f)
                )
            }
        }
    }
}

/**
 * Banner shown during workflow step execution
 */
@Composable
private fun WorkflowStepBanner(
    state: WorkflowState.StepRunning,
    modifier: Modifier = Modifier
) {
    val progress by animateFloatAsState(
        targetValue = if (state.totalSteps > 0) {
            (state.stepIndex - 1).toFloat() / state.totalSteps.toFloat()
        } else 0f,
        animationSpec = tween(durationMillis = 300, easing = LinearEasing),
        label = "progress"
    )

    val containerColor = when (state.status) {
        StepStatus.RUNNING -> MaterialTheme.colorScheme.primaryContainer
        StepStatus.COMPLETED -> MaterialTheme.colorScheme.tertiaryContainer
        StepStatus.FAILED -> MaterialTheme.colorScheme.errorContainer
        StepStatus.PENDING -> MaterialTheme.colorScheme.surfaceVariant
        StepStatus.SKIPPED -> MaterialTheme.colorScheme.surfaceVariant
    }

    val contentColor = when (state.status) {
        StepStatus.RUNNING -> MaterialTheme.colorScheme.onPrimaryContainer
        StepStatus.COMPLETED -> MaterialTheme.colorScheme.onTertiaryContainer
        StepStatus.FAILED -> MaterialTheme.colorScheme.onErrorContainer
        StepStatus.PENDING -> MaterialTheme.colorScheme.onSurfaceVariant
        StepStatus.SKIPPED -> MaterialTheme.colorScheme.onSurfaceVariant
    }

    Surface(
        modifier = modifier.fillMaxWidth(),
        color = containerColor,
        tonalElevation = 2.dp
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Status icon
                when (state.status) {
                    StepStatus.RUNNING -> {
                        PulsingIndicator(
                            color = contentColor,
                            modifier = Modifier.size(20.dp)
                        )
                    }
                    StepStatus.COMPLETED -> {
                        Icon(
                            imageVector = Icons.Default.CheckCircle,
                            contentDescription = "Completed",
                            tint = contentColor,
                            modifier = Modifier.size(20.dp)
                        )
                    }
                    StepStatus.FAILED -> {
                        Icon(
                            imageVector = AppIcons.Error,
                            contentDescription = "Failed",
                            tint = contentColor,
                            modifier = Modifier.size(20.dp)
                        )
                    }
                    else -> {
                        Icon(
                            imageVector = AppIcons.Schedule,
                            contentDescription = "Pending",
                            tint = contentColor,
                            modifier = Modifier.size(20.dp)
                        )
                    }
                }

                Spacer(modifier = Modifier.width(12.dp))

                // Step info
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = state.stepName,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = contentColor,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    Text(
                        text = "Step ${state.stepIndex} of ${state.totalSteps}",
                        style = MaterialTheme.typography.bodySmall,
                        color = contentColor.copy(alpha = 0.7f)
                    )
                }

                // Step status badge
                Surface(
                    shape = RoundedCornerShape(4.dp),
                    color = contentColor.copy(alpha = 0.15f)
                ) {
                    Text(
                        text = state.status.name.lowercase()
                            .replaceFirstChar { it.uppercase() },
                        style = MaterialTheme.typography.labelSmall,
                        color = contentColor,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                    )
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Progress bar
            LinearProgressIndicator(
                progress = progress,
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

/**
 * Animated pulsing indicator
 */
@Composable
private fun PulsingIndicator(
    color: Color,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val scale by infiniteTransition.animateFloat(
        initialValue = 0.8f,
        targetValue = 1.2f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
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
 * Compact workflow indicator for use in message bubbles
 */
@Composable
fun WorkflowStatusChip(
    workflowState: WorkflowState,
    modifier: Modifier = Modifier,
    onClick: () -> Unit = {}
) {
    val (icon, color, text) = when (workflowState) {
        is WorkflowState.Started -> Triple(
            AppIcons.HourglassTop,
            MaterialTheme.colorScheme.primary,
            "Starting"
        )
        is WorkflowState.StepRunning -> {
            val step = workflowState as WorkflowState.StepRunning
            Triple(
                when (step.status) {
                    StepStatus.RUNNING -> AppIcons.Sync
                    StepStatus.COMPLETED -> Icons.Default.CheckCircle
                    StepStatus.FAILED -> AppIcons.Error
                    else -> AppIcons.Schedule
                },
                when (step.status) {
                    StepStatus.RUNNING -> MaterialTheme.colorScheme.primary
                    StepStatus.COMPLETED -> MaterialTheme.colorScheme.tertiary
                    StepStatus.FAILED -> MaterialTheme.colorScheme.error
                    else -> MaterialTheme.colorScheme.outline
                },
                step.stepName
            )
        }
    }

    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = color.copy(alpha = 0.1f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(14.dp)
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = text,
                style = MaterialTheme.typography.labelSmall,
                color = color,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }
    }
}

/**
 * Preview helpers
 */
@Composable
fun WorkflowProgressBannerPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        WorkflowProgressBanner(
            workflowState = WorkflowState.Started(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                roomId = "!room:example.com",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            )
        )

        WorkflowProgressBanner(
            workflowState = WorkflowState.StepRunning(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                roomId = "!room:example.com",
                stepId = "step_2",
                stepName = "Analyzing document content",
                stepIndex = 2,
                totalSteps = 5,
                status = StepStatus.RUNNING,
                timestamp = System.currentTimeMillis()
            )
        )
    }
}
