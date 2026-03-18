package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.scale
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.data.store.AgentThinkingState

/**
 * Agent Thinking Indicator
 *
 * Displays an animated indicator when an AI agent is processing.
 * Shows a typing-like animation with optional status message.
 *
 * ## Usage
 * ```kotlin
 * val thinkingAgents by viewModel.thinkingAgents.collectAsState()
 *
 * AgentThinkingIndicator(
 *     thinkingAgents = thinkingAgents.values.toList(),
 *     expanded = true
 * )
 * ```
 *
 * ## Animation
 * - Dots animate sequentially
 * - Fade in/out effect
 * - Optional pulse effect on container
 */
@Composable
fun AgentThinkingIndicator(
    thinkingAgents: List<AgentThinkingState>,
    modifier: Modifier = Modifier,
    expanded: Boolean = true,
    showNames: Boolean = true,
    onAgentClick: ((AgentThinkingState) -> Unit)? = null
) {
    if (thinkingAgents.isEmpty()) return

    AnimatedVisibility(
        visible = expanded && thinkingAgents.isNotEmpty(),
        enter = fadeIn() + expandVertically(),
        exit = fadeOut() + shrinkVertically()
    ) {
        Column(
            modifier = modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            thinkingAgents.forEach { agent ->
                AgentThinkingRow(
                    agent = agent,
                    showName = showNames,
                    onClick = if (onAgentClick != null) {
                        { onAgentClick(agent) }
                    } else null
                )
            }
        }
    }
}

/**
 * Single agent thinking row
 */
@Composable
private fun AgentThinkingRow(
    agent: AgentThinkingState,
    showName: Boolean,
    onClick: (() -> Unit)?,
    modifier: Modifier = Modifier
) {
    val containerColor = MaterialTheme.colorScheme.secondaryContainer
    val contentColor = MaterialTheme.colorScheme.onSecondaryContainer

    Surface(
        onClick = onClick ?: {},
        enabled = onClick != null,
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = containerColor
    ) {
        Row(
            modifier = Modifier
                .padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // AI Icon with rotation animation
            RotatingIcon(
                icon = Icons.Default.AutoAwesome,
                color = contentColor,
                modifier = Modifier.size(18.dp)
            )

            Spacer(modifier = Modifier.width(10.dp))

            // Agent name and status
            Column(modifier = Modifier.weight(1f)) {
                if (showName) {
                    Text(
                        text = agent.agentName,
                        style = MaterialTheme.typography.labelMedium,
                        color = contentColor
                    )
                }
                if (agent.message != null) {
                    Text(
                        text = agent.message,
                        style = MaterialTheme.typography.bodySmall,
                        fontStyle = FontStyle.Italic,
                        color = contentColor.copy(alpha = 0.7f),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }

            // Animated dots
            AnimatedTypingDots(
                color = contentColor,
                modifier = Modifier.padding(start = 8.dp)
            )
        }
    }
}

/**
 * Rotating icon animation
 */
@Composable
private fun RotatingIcon(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    color: androidx.compose.ui.graphics.Color,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "rotation")
    val rotation by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(2000, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "rotation"
    )

    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(1000, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Icon(
        imageVector = icon,
        contentDescription = null,
        tint = color.copy(alpha = alpha),
        modifier = modifier
            .rotate(rotation)
    )
}

/**
 * Animated typing dots (three dots that animate sequentially)
 */
@Composable
private fun AnimatedTypingDots(
    color: androidx.compose.ui.graphics.Color,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(3.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        repeat(3) { index ->
            val infiniteTransition = rememberInfiniteTransition(label = "dot_$index")
            val scale by infiniteTransition.animateFloat(
                initialValue = 0.5f,
                targetValue = 1f,
                animationSpec = infiniteRepeatable(
                    animation = tween(
                        durationMillis = 400,
                        delayMillis = index * 150,
                        easing = FastOutSlowInEasing
                    ),
                    repeatMode = RepeatMode.Reverse
                ),
                label = "scale_$index"
            )

            val alpha by infiniteTransition.animateFloat(
                initialValue = 0.3f,
                targetValue = 1f,
                animationSpec = infiniteRepeatable(
                    animation = tween(
                        durationMillis = 400,
                        delayMillis = index * 150,
                        easing = LinearEasing
                    ),
                    repeatMode = RepeatMode.Reverse
                ),
                label = "alpha_$index"
            )

            Box(
                modifier = Modifier
                    .size(6.dp)
                    .scale(scale)
                    .alpha(alpha)
                    .background(
                        color = color,
                        shape = RoundedCornerShape(percent = 50)
                    )
            )
        }
    }
}

/**
 * Compact agent status chip for use in message bubbles
 */
@Composable
fun AgentStatusChip(
    agentName: String,
    isThinking: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit = {}
) {
    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = if (isThinking) {
            MaterialTheme.colorScheme.secondaryContainer
        } else {
            MaterialTheme.colorScheme.surfaceVariant
        }
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.AutoAwesome,
                contentDescription = null,
                tint = if (isThinking) {
                    MaterialTheme.colorScheme.onSecondaryContainer
                } else {
                    MaterialTheme.colorScheme.onSurfaceVariant
                },
                modifier = Modifier.size(14.dp)
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = if (isThinking) "$agentName thinking..." else agentName,
                style = MaterialTheme.typography.labelSmall,
                color = if (isThinking) {
                    MaterialTheme.colorScheme.onSecondaryContainer
                } else {
                    MaterialTheme.colorScheme.onSurfaceVariant
                },
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }
    }
}

/**
 * Inline agent thinking indicator for chat messages
 */
@Composable
fun InlineAgentThinking(
    agentName: String,
    message: String?,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            RotatingIcon(
                icon = Icons.Default.AutoAwesome,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(16.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = message ?: "$agentName is thinking...",
                style = MaterialTheme.typography.bodySmall,
                fontStyle = FontStyle.Italic,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

/**
 * Preview helpers
 */
@Composable
fun AgentThinkingIndicatorPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        AgentThinkingIndicator(
            thinkingAgents = listOf(
                AgentThinkingState(
                    agentId = "@agent_analysis:example.com",
                    agentName = "Analysis Agent",
                    message = "Processing document...",
                    timestamp = System.currentTimeMillis()
                ),
                AgentThinkingState(
                    agentId = "@agent_writer:example.com",
                    agentName = "Writer Agent",
                    message = null,
                    timestamp = System.currentTimeMillis()
                )
            ),
            expanded = true,
            showNames = true
        )
    }
}
