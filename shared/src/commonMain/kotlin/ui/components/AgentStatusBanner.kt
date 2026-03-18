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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.AgentTaskStatus
import com.armorclaw.shared.domain.model.AgentTaskStatusEvent
import com.armorclaw.shared.ui.theme.AppIcons

/**
 * Agent Status Banner
 *
 * Displays the current status of an AI agent in a chat screen.
 * Shows status icon, text, and optional metadata like current step.
 *
 * ## Usage
 * ```kotlin
 * val agentStatus by controlPlaneStore.agentStatuses.collectAsState()
 * val currentStatus = agentStatuses[activeAgentId]
 *
 * currentStatus?.let { status ->
 *     AgentTaskStatusBanner(
 *         status = status.status,
 *         metadata = status.metadata
 *     )
 * }
 * ```
 *
 * ## Status Colors
 * - BROWSING: Primary (blue)
 * - FORM_FILLING: Tertiary (green)
 * - PROCESSING_PAYMENT: Error (red)
 * - AWAITING_*: Secondary (orange)
 * - ERROR: Error (red)
 * - COMPLETE: Tertiary (green)
 */
@Composable
fun AgentTaskStatusBanner(
    status: AgentTaskStatus,
    metadata: Map<String, String>?,
    modifier: Modifier = Modifier,
    onDismiss: (() -> Unit)? = null
) {
    val (icon, text, color) = getStatusDisplay(status)

    AnimatedVisibility(
        visible = status != AgentTaskStatus.IDLE,
        enter = slideInVertically() + fadeIn(),
        exit = slideOutVertically() + fadeOut()
    ) {
        Surface(
            modifier = modifier.fillMaxWidth(),
            color = color.copy(alpha = 0.12f),
            shape = RoundedCornerShape(8.dp),
            tonalElevation = 1.dp
        ) {
            Row(
                modifier = Modifier
                    .padding(horizontal = 12.dp, vertical = 10.dp)
                    .fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Animated icon for active states
                if (status.isActive()) {
                    PulsingStatusIcon(
                        icon = icon,
                        color = color,
                        modifier = Modifier.size(20.dp)
                    )
                } else {
                    Icon(
                        imageVector = icon,
                        contentDescription = null,
                        tint = color,
                        modifier = Modifier.size(20.dp)
                    )
                }

                Spacer(Modifier.width(10.dp))

                // Status text
                Text(
                    text = text,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = color,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false)
                )

                // Step indicator
                if (metadata?.get("step") != null) {
                    Spacer(Modifier.width(8.dp))
                    Surface(
                        color = color.copy(alpha = 0.15f),
                        shape = RoundedCornerShape(4.dp)
                    ) {
                        Text(
                            text = "Step ${metadata["step"]}",
                            style = MaterialTheme.typography.labelSmall,
                            color = color,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }

                // URL or context
                metadata?.get("url")?.let { url ->
                    Spacer(Modifier.width(8.dp))
                    Text(
                        text = url.let {
                            if (it.length > 20) "..." + it.takeLast(17) else it
                        },
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        modifier = Modifier.weight(1f, fill = false)
                    )
                }

                // Dismiss button for intervention states
                if (status.requiresIntervention() && onDismiss != null) {
                    Spacer(Modifier.width(8.dp))
                    IconButton(
                        onClick = onDismiss,
                        modifier = Modifier.size(24.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Dismiss",
                            tint = color,
                            modifier = Modifier.size(16.dp)
                        )
                    }
                }
            }
        }
    }
}

/**
 * Get display properties for a status
 */
@Composable
private fun getStatusDisplay(status: AgentTaskStatus): Triple<ImageVector, String, Color> {
    val primary = MaterialTheme.colorScheme.primary
    val tertiary = MaterialTheme.colorScheme.tertiary
    val error = MaterialTheme.colorScheme.error
    val secondary = MaterialTheme.colorScheme.secondary
    val outline = MaterialTheme.colorScheme.outline

    return when (status) {
        AgentTaskStatus.IDLE -> Triple(
            Icons.Default.Info,
            "Idle",
            outline
        )
        AgentTaskStatus.BROWSING -> Triple(
            Icons.Default.Language,
            "Browsing...",
            primary
        )
        AgentTaskStatus.FORM_FILLING -> Triple(
            Icons.Default.Edit,
            "Filling form...",
            tertiary
        )
        AgentTaskStatus.PROCESSING_PAYMENT -> Triple(
            Icons.Default.Payment,
            "Processing payment...",
            error
        )
        AgentTaskStatus.AWAITING_CAPTCHA -> Triple(
            Icons.Default.Security,
            "Waiting for CAPTCHA",
            error
        )
        AgentTaskStatus.AWAITING_2FA -> Triple(
            Icons.Default.Key,
            "Waiting for 2FA",
            error
        )
        AgentTaskStatus.AWAITING_APPROVAL -> Triple(
            Icons.Default.Pending,
            "Waiting for approval",
            secondary
        )
        AgentTaskStatus.ERROR -> Triple(
            AppIcons.Error,
            "Error occurred",
            error
        )
        AgentTaskStatus.COMPLETE -> Triple(
            Icons.Default.CheckCircle,
            "Complete",
            tertiary
        )
    }
}

/**
 * Animated pulsing icon for active states
 */
@Composable
private fun PulsingStatusIcon(
    icon: ImageVector,
    color: Color,
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
        modifier = modifier
    )
}

/**
 * Compact status chip for use in message headers
 */
@Composable
fun AgentTaskStatusChip(
    status: AgentTaskStatus,
    modifier: Modifier = Modifier,
    onClick: () -> Unit = {}
) {
    val (_, _, color) = getStatusDisplay(status)

    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = color.copy(alpha = 0.1f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(6.dp)
                    .background(color, RoundedCornerShape(percent = 50))
            )
            Spacer(Modifier.width(6.dp))
            Text(
                text = status.toDisplayString(),
                style = MaterialTheme.typography.labelSmall,
                color = color,
                maxLines = 1
            )
        }
    }
}

/**
 * Preview helper
 */
@Composable
fun AgentTaskStatusBannerPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        AgentTaskStatusBanner(
            status = AgentTaskStatus.BROWSING,
            metadata = mapOf("url" to "https://example.com")
        )

        AgentTaskStatusBanner(
            status = AgentTaskStatus.FORM_FILLING,
            metadata = mapOf("step" to "2", "total" to "5")
        )

        AgentTaskStatusBanner(
            status = AgentTaskStatus.PROCESSING_PAYMENT,
            metadata = null
        )

        AgentTaskStatusBanner(
            status = AgentTaskStatus.AWAITING_CAPTCHA,
            metadata = null
        )

        AgentTaskStatusBanner(
            status = AgentTaskStatus.COMPLETE,
            metadata = null
        )
    }
}
