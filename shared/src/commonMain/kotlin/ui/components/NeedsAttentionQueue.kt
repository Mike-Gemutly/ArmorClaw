package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.AttentionItem
import com.armorclaw.shared.domain.model.AttentionPriority
import com.armorclaw.shared.ui.theme.AppIcons

/**
 * Needs Attention Queue
 *
 * Displays a prioritized list of items requiring user intervention.
 * Critical items appear at the top with visual emphasis.
 *
 * ## Architecture
 * ```
 * NeedsAttentionQueue
 *      ├── Section header with count
 *      └── LazyColumn of AttentionItemCard[]
 *          ├── Priority indicator
 *          ├── Item details
 *          └── Quick action buttons
 * ```
 *
 * ## Usage
 * ```kotlin
 * NeedsAttentionQueue(
 *     items = attentionItems,
 *     onItemClick = { item -> viewModel.onAttentionItemClick(item) },
 *     onApprove = { item -> viewModel.approveItem(item) },
 *     onDeny = { item -> viewModel.denyItem(item) },
 *     modifier = Modifier.padding(horizontal = 16.dp)
 * )
 * ```
 */
@Composable
fun NeedsAttentionQueue(
    items: List<AttentionItem>,
    modifier: Modifier = Modifier,
    onItemClick: (AttentionItem) -> Unit = {},
    onApprove: ((AttentionItem) -> Unit)? = null,
    onDeny: ((AttentionItem) -> Unit)? = null,
    maxVisible: Int = 5
) {
    AnimatedVisibility(
        visible = items.isNotEmpty(),
        enter = fadeIn() + slideInVertically(),
        exit = fadeOut() + slideOutVertically()
    ) {
        Column(modifier = modifier) {
            // Section header
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 8.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    // Animated attention icon
                    PulsingAttentionIcon(
                        hasCritical = items.any { it.priority == AttentionPriority.CRITICAL }
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    Text(
                        text = "Needs Your Attention",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    // Count badge
                    Surface(
                        shape = RoundedCornerShape(12.dp),
                        color = when {
                            items.any { it.priority == AttentionPriority.CRITICAL } -> MaterialTheme.colorScheme.error
                            items.any { it.priority == AttentionPriority.HIGH } -> MaterialTheme.colorScheme.tertiary
                            else -> MaterialTheme.colorScheme.secondary
                        }
                    ) {
                        Text(
                            text = items.size.toString(),
                            style = MaterialTheme.typography.labelSmall,
                            color = Color.White,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                        )
                    }
                }

                if (items.size > maxVisible) {
                    TextButton(onClick = { /* Navigate to full list */ }) {
                        Text("See all")
                    }
                }
            }

            // Attention items
            LazyColumn(
                modifier = Modifier.heightIn(max = 400.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(
                    items = items.sortedByDescending { it.priority.ordinal }.take(maxVisible),
                    key = { it.id }
                ) { item ->
                    AttentionItemCard(
                        item = item,
                        onClick = { onItemClick(item) },
                        onApprove = if (onApprove != null) { { onApprove(item) } } else null,
                        onDeny = if (onDeny != null) { { onDeny(item) } } else null
                    )
                }
            }
        }
    }
}

/**
 * Individual attention item card
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AttentionItemCard(
    item: AttentionItem,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    onApprove: (() -> Unit)? = null,
    onDeny: (() -> Unit)? = null
) {
    val (icon, backgroundColor, contentColor) = getAttentionItemColors(item)

    Card(
        onClick = onClick,
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = backgroundColor,
            contentColor = contentColor
        ),
        elevation = CardDefaults.cardElevation(
            defaultElevation = if (item.priority == AttentionPriority.CRITICAL) 4.dp else 2.dp
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Priority indicator
            PriorityIndicator(
                priority = item.priority,
                modifier = Modifier.padding(end = 12.dp)
            )

            // Content
            Column(modifier = Modifier.weight(1f)) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    Icon(
                        imageVector = icon,
                        contentDescription = null,
                        tint = contentColor,
                        modifier = Modifier.size(18.dp)
                    )
                    Text(
                        text = item.title,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }

                Spacer(modifier = Modifier.height(4.dp))

                Text(
                    text = item.description,
                    style = MaterialTheme.typography.bodySmall,
                    color = contentColor.copy(alpha = 0.8f),
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )

                // Agent info
                Text(
                    text = "from ${item.agentName}",
                    style = MaterialTheme.typography.labelSmall,
                    color = contentColor.copy(alpha = 0.6f),
                    modifier = Modifier.padding(top = 4.dp)
                )
            }

            // Quick action buttons
            if (onApprove != null || onDeny != null) {
                Spacer(modifier = Modifier.width(8.dp))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    onDeny?.let { deny ->
                        IconButton(
                            onClick = deny,
                            modifier = Modifier.size(36.dp)
                        ) {
                            Icon(
                                imageVector = Icons.Default.Close,
                                contentDescription = "Deny",
                                tint = contentColor.copy(alpha = 0.7f),
                                modifier = Modifier.size(20.dp)
                            )
                        }
                    }

                    onApprove?.let { approve ->
                        FilledIconButton(
                            onClick = approve,
                            modifier = Modifier.size(36.dp),
                            colors = IconButtonDefaults.filledIconButtonColors(
                                containerColor = contentColor,
                                contentColor = backgroundColor
                            )
                        ) {
                            Icon(
                                imageVector = Icons.Default.Check,
                                contentDescription = "Approve",
                                modifier = Modifier.size(20.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}

/**
 * Priority indicator with animated dot
 */
@Composable
private fun PriorityIndicator(
    priority: AttentionPriority,
    modifier: Modifier = Modifier
) {
    val color = when (priority) {
        AttentionPriority.CRITICAL -> MaterialTheme.colorScheme.error
        AttentionPriority.HIGH -> MaterialTheme.colorScheme.tertiary
        AttentionPriority.MEDIUM -> MaterialTheme.colorScheme.secondary
        AttentionPriority.LOW -> MaterialTheme.colorScheme.outline
    }

    val infiniteTransition = rememberInfiniteTransition(label = "priority")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = if (priority == AttentionPriority.CRITICAL) 1f else 0.7f,
        animationSpec = infiniteRepeatable(
            animation = tween(if (priority == AttentionPriority.CRITICAL) 400 else 800),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    val scale by infiniteTransition.animateFloat(
        initialValue = 0.9f,
        targetValue = if (priority == AttentionPriority.CRITICAL) 1.1f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(if (priority == AttentionPriority.CRITICAL) 400 else 800),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    Box(
        modifier = modifier
            .graphicsLayer {
                scaleX = scale
                scaleY = scale
            }
            .size(10.dp)
            .clip(RoundedCornerShape(percent = 50))
            .background(color.copy(alpha = alpha))
    )
}

/**
 * Animated attention icon for header
 */
@Composable
private fun PulsingAttentionIcon(
    hasCritical: Boolean,
    modifier: Modifier = Modifier
) {
    val color = if (hasCritical) {
        MaterialTheme.colorScheme.error
    } else {
        MaterialTheme.colorScheme.tertiary
    }

    val infiniteTransition = rememberInfiniteTransition(label = "attention_icon")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.6f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(if (hasCritical) 400 else 600, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Icon(
        imageVector = if (hasCritical) Icons.Default.NotificationImportant else Icons.Default.NotificationsActive,
        contentDescription = null,
        tint = color.copy(alpha = alpha),
        modifier = modifier.size(22.dp)
    )
}

/**
 * Get colors for attention item based on type and priority
 */
@Composable
private fun getAttentionItemColors(
    item: AttentionItem
): Triple<ImageVector, Color, Color> {
    return when (item) {
        is AttentionItem.PiiRequest -> {
            val bgColor = when {
                item.piiRequest.fields.any { it.sensitivity.name == "CRITICAL" } ->
                    MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.7f)
                item.piiRequest.fields.any { it.sensitivity.name == "HIGH" } ->
                    MaterialTheme.colorScheme.tertiaryContainer
                else -> MaterialTheme.colorScheme.secondaryContainer
            }
            Triple(
                Icons.Default.Lock,
                bgColor,
                when (bgColor) {
                    MaterialTheme.colorScheme.errorContainer -> MaterialTheme.colorScheme.onErrorContainer
                    MaterialTheme.colorScheme.tertiaryContainer -> MaterialTheme.colorScheme.onTertiaryContainer
                    else -> MaterialTheme.colorScheme.onSecondaryContainer
                }
            )
        }
        is AttentionItem.CaptchaChallenge -> Triple(
            Icons.Default.Security,
            MaterialTheme.colorScheme.tertiaryContainer,
            MaterialTheme.colorScheme.onTertiaryContainer
        )
        is AttentionItem.TwoFactorAuth -> Triple(
            Icons.Default.Key,
            MaterialTheme.colorScheme.secondaryContainer,
            MaterialTheme.colorScheme.onSecondaryContainer
        )
        is AttentionItem.ApprovalRequest -> Triple(
            Icons.Default.Pending,
            MaterialTheme.colorScheme.primaryContainer,
            MaterialTheme.colorScheme.onPrimaryContainer
        )
        is AttentionItem.ErrorState -> Triple(
            AppIcons.Error,
            MaterialTheme.colorScheme.errorContainer,
            MaterialTheme.colorScheme.onErrorContainer
        )
    }
}

/**
 * Preview helper
 */
@Composable
fun NeedsAttentionQueuePreview() {
    val sampleItems = listOf<AttentionItem>(
        AttentionItem.PiiRequest(
            id = "req_1",
            agentId = "agent_checkout",
            agentName = "Checkout Bot",
            roomId = "!room1:example.com",
            timestamp = System.currentTimeMillis(),
            piiRequest = com.armorclaw.shared.domain.model.PiiAccessRequest(
                requestId = "req_1",
                agentId = "agent_checkout",
                fields = listOf(
                    com.armorclaw.shared.domain.model.PiiField(
                        name = "CVV",
                        sensitivity = com.armorclaw.shared.domain.model.SensitivityLevel.CRITICAL,
                        description = "Card verification code"
                    )
                ),
                reason = "Complete purchase on example.com",
                expiresAt = System.currentTimeMillis() + 60000
            )
        ),
        AttentionItem.CaptchaChallenge(
            id = "captcha_1",
            agentId = "agent_browse",
            agentName = "Browser Bot",
            roomId = "!room2:example.com",
            timestamp = System.currentTimeMillis(),
            siteUrl = "https://example.com"
        ),
        AttentionItem.TwoFactorAuth(
            id = "2fa_1",
            agentId = "agent_login",
            agentName = "Login Bot",
            roomId = "!room3:example.com",
            timestamp = System.currentTimeMillis(),
            service = "Banking App"
        ),
        AttentionItem.ErrorState(
            id = "error_1",
            agentId = "agent_data",
            agentName = "Data Bot",
            roomId = "!room4:example.com",
            timestamp = System.currentTimeMillis(),
            errorMessage = "Session expired. Please re-authenticate.",
            recoverable = true
        )
    )

    NeedsAttentionQueue(
        items = sampleItems,
        onItemClick = {},
        onApprove = {},
        onDeny = {}
    )
}
