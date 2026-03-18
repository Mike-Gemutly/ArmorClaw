package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.ui.theme.AppIcons
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Unified Message List
 *
 * Displays all message types (Regular, Agent, System, Command) in a single list.
 * Replaces the separate Terminal and Chat interfaces.
 *
 * ## Usage
 * ```kotlin
 * val messages by viewModel.messages.collectAsState()
 *
 * UnifiedMessageList(
 *     messages = messages,
 *     currentUserId = "@user:example.com",
 *     onReply = { viewModel.replyToMessage(it) },
 *     onReaction = { msg, emoji -> viewModel.toggleReaction(msg, emoji) },
 *     onAction = { msg, action -> viewModel.handleAction(msg, action) }
 * )
 * ```
 */
@Composable
fun UnifiedMessageList(
    messages: List<UnifiedMessage>,
    currentUserId: String,
    modifier: Modifier = Modifier,
    listState: LazyListState = rememberLazyListState(),
    isLoading: Boolean = false,
    hasMore: Boolean = false,
    onLoadMore: () -> Unit = {},
    onReply: (UnifiedMessage) -> Unit = {},
    onReaction: (UnifiedMessage, String) -> Unit = { _, _ -> },
    onAction: (UnifiedMessage, AgentAction) -> Unit = { _, _ -> },
    onSystemAction: (UnifiedMessage, SystemAction) -> Unit = { _, _ -> },
    onRetryCommand: (UnifiedMessage.Command) -> Unit = {}
) {
    LazyColumn(
        state = listState,
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        // Loading indicator at top
        if (isLoading) {
            item {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(24.dp),
                        strokeWidth = 2.dp
                    )
                }
            }
        }

        // Messages
        items(
            items = messages,
            key = { it.id }
        ) { message ->
            UnifiedMessageItem(
                message = message,
                isFromCurrentUser = message.isFromCurrentUser(currentUserId),
                onReply = { onReply(message) },
                onReaction = { emoji -> onReaction(message, emoji) },
                onAction = { action -> onAction(message, action) },
                onSystemAction = { action -> onSystemAction(message, action) },
                onRetry = if (message is UnifiedMessage.Command) {
                    { onRetryCommand(message) }
                } else null
            )
        }

        // Load more trigger
        if (hasMore && !isLoading) {
            item {
                LaunchedEffect(Unit) {
                    onLoadMore()
                }
            }
        }
    }
}

/**
 * Single message item that delegates to appropriate component
 */
@Composable
private fun UnifiedMessageItem(
    message: UnifiedMessage,
    isFromCurrentUser: Boolean,
    onReply: () -> Unit,
    onReaction: (String) -> Unit,
    onAction: (AgentAction) -> Unit,
    onSystemAction: (SystemAction) -> Unit,
    onRetry: (() -> Unit)?
) {
    when (message) {
        is UnifiedMessage.Regular -> {
            RegularMessageItem(
                message = message,
                isFromCurrentUser = isFromCurrentUser,
                onReply = onReply,
                onReaction = onReaction
            )
        }
        is UnifiedMessage.Agent -> {
            AgentMessageItem(
                message = message,
                onAction = onAction
            )
        }
        is UnifiedMessage.System -> {
            SystemMessageItem(
                message = message,
                onAction = onSystemAction
            )
        }
        is UnifiedMessage.Command -> {
            CommandMessageItem(
                message = message,
                onRetry = onRetry
            )
        }
    }
}

/**
 * Regular user message
 */
@Composable
private fun RegularMessageItem(
    message: UnifiedMessage.Regular,
    isFromCurrentUser: Boolean,
    onReply: () -> Unit,
    onReaction: (String) -> Unit
) {
    val alignment = if (isFromCurrentUser) Alignment.End else Alignment.Start
    val shape = if (isFromCurrentUser) {
        RoundedCornerShape(16.dp, 4.dp, 16.dp, 16.dp)
    } else {
        RoundedCornerShape(4.dp, 16.dp, 16.dp, 16.dp)
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 4.dp),
        horizontalAlignment = alignment
    ) {
        // Sender name (not for current user)
        if (!isFromCurrentUser) {
            Text(
                text = message.sender.displayName,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(start = 4.dp, bottom = 2.dp)
            )
        }

        // Message bubble
        Surface(
            shape = shape,
            color = if (isFromCurrentUser) {
                MaterialTheme.colorScheme.primaryContainer
            } else {
                MaterialTheme.colorScheme.surfaceVariant
            }
        ) {
            Column(
                modifier = Modifier.padding(12.dp)
            ) {
                // Reply preview
                if (message.replyTo != null) {
                    ReplyPreview(
                        replyToId = message.replyTo,
                        modifier = Modifier.padding(bottom = 8.dp)
                    )
                }

                // Message content
                Text(
                    text = message.content.body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (isFromCurrentUser) {
                        MaterialTheme.colorScheme.onPrimaryContainer
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    }
                )

                // Reactions
                if (message.reactions.isNotEmpty()) {
                    ReactionRow(
                        reactions = message.reactions,
                        onClick = onReaction,
                        modifier = Modifier.padding(top = 8.dp)
                    )
                }
            }
        }

        // Timestamp and status
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(top = 2.dp)
        ) {
            Text(
                text = formatTime(message.timestamp),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
            )

            if (isFromCurrentUser && message.status != MessageStatus.SENT) {
                Spacer(modifier = Modifier.width(4.dp))
                MessageStatusIcon(message.status)
            }
        }
    }
}

/**
 * Agent/AI message
 */
@Composable
private fun AgentMessageItem(
    message: UnifiedMessage.Agent,
    onAction: (AgentAction) -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 4.dp),
        horizontalAlignment = Alignment.Start
    ) {
        // Agent header
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(bottom = 4.dp)
        ) {
            // Agent icon
            Surface(
                shape = RoundedCornerShape(12.dp),
                color = MaterialTheme.colorScheme.secondaryContainer
            ) {
                Icon(
                    imageVector = getAgentIcon(message.agentType),
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSecondaryContainer,
                    modifier = Modifier.padding(8.dp).size(20.dp)
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            Text(
                text = message.sender.displayName,
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.onSurface
            )

            Spacer(modifier = Modifier.width(4.dp))

            // Agent type badge
            Surface(
                shape = RoundedCornerShape(4.dp),
                color = MaterialTheme.colorScheme.tertiaryContainer
            ) {
                Text(
                    text = message.agentType.name.lowercase()
                        .replaceFirstChar { it.uppercase() },
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onTertiaryContainer,
                    modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                )
            }
        }

        // Message bubble
        Surface(
            shape = RoundedCornerShape(4.dp, 16.dp, 16.dp, 16.dp),
            color = MaterialTheme.colorScheme.secondaryContainer.copy(alpha = 0.5f)
        ) {
            Column(
                modifier = Modifier.padding(12.dp)
            ) {
                // Message content
                Text(
                    text = message.content.body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSecondaryContainer
                )

                // Sources
                if (message.sources.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(8.dp))
                    SourceList(sources = message.sources)
                }

                // Action buttons
                if (message.actions.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(8.dp))
                    AgentActionRow(
                        actions = message.actions,
                        onAction = onAction
                    )
                }
            }
        }

        // Timestamp
        Text(
            text = formatTime(message.timestamp),
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f),
            modifier = Modifier.padding(top = 2.dp)
        )
    }
}

/**
 * System message (events, notifications)
 */
@Composable
private fun SystemMessageItem(
    message: UnifiedMessage.System,
    onAction: (SystemAction) -> Unit
) {
    val (icon, color) = getSystemMessageStyle(message.eventType)

    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 4.dp),
        shape = RoundedCornerShape(12.dp),
        color = color.copy(alpha = 0.1f)
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(20.dp)
            )

            Spacer(modifier = Modifier.width(12.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = message.title,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = color
                )
                if (message.description != null) {
                    Text(
                        text = message.description,
                        style = MaterialTheme.typography.bodySmall,
                        color = color.copy(alpha = 0.7f)
                    )
                }
            }

            // Quick actions
            if (message.actions.isNotEmpty()) {
                Row {
                    message.actions.take(2).forEach { action ->
                        TextButton(
                            onClick = { onAction(action) },
                            contentPadding = PaddingValues(horizontal = 8.dp)
                        ) {
                            Text(
                                text = action.label,
                                style = MaterialTheme.typography.labelSmall
                            )
                        }
                    }
                }
            }
        }
    }
}

/**
 * Command message
 */
@Composable
private fun CommandMessageItem(
    message: UnifiedMessage.Command,
    onRetry: (() -> Unit)?
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 4.dp)
    ) {
        // Command header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(8.dp))
                .background(MaterialTheme.colorScheme.surfaceVariant)
                .padding(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Terminal,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(16.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = message.command,
                style = MaterialTheme.typography.bodySmall,
                fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.weight(1f))
            CommandStatusBadge(status = message.status)
        }

        // Result (if any)
        if (message.result != null) {
            Surface(
                shape = RoundedCornerShape(8.dp),
                color = when (message.status) {
                    CommandStatus.COMPLETED -> MaterialTheme.colorScheme.tertiaryContainer
                    CommandStatus.FAILED -> MaterialTheme.colorScheme.errorContainer
                    else -> MaterialTheme.colorScheme.surfaceVariant
                },
                modifier = Modifier.padding(top = 4.dp)
            ) {
                Text(
                    text = message.result,
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                    color = when (message.status) {
                        CommandStatus.COMPLETED -> MaterialTheme.colorScheme.onTertiaryContainer
                        CommandStatus.FAILED -> MaterialTheme.colorScheme.onErrorContainer
                        else -> MaterialTheme.colorScheme.onSurfaceVariant
                    },
                    modifier = Modifier.padding(8.dp)
                )
            }
        }

        // Retry button
        if (message.status == CommandStatus.FAILED && onRetry != null) {
            TextButton(
                onClick = onRetry,
                modifier = Modifier.padding(top = 4.dp)
            ) {
                Icon(Icons.Default.Refresh, contentDescription = null)
                Spacer(modifier = Modifier.width(4.dp))
                Text("Retry")
            }
        }
    }
}

// ========================================================================
// Helper Components
// ========================================================================

@Composable
private fun ReplyPreview(
    replyToId: String,
    modifier: Modifier = Modifier
) {
    // In real implementation, would look up the replied message
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(4.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier.padding(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Reply,
                contentDescription = null,
                modifier = Modifier.size(14.dp),
                tint = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = "Reply...",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun ReactionRow(
    reactions: List<Reaction>,
    onClick: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        reactions.forEach { reaction ->
            Surface(
                onClick = { onClick(reaction.emoji) },
                shape = RoundedCornerShape(12.dp),
                color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f)
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(text = reaction.emoji)
                    if (reaction.count > 1) {
                        Spacer(modifier = Modifier.width(2.dp))
                        Text(
                            text = reaction.count.toString(),
                            style = MaterialTheme.typography.labelSmall
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun MessageStatusIcon(status: MessageStatus) {
    val (icon, color) = when (status) {
        MessageStatus.PENDING -> AppIcons.Pending to MaterialTheme.colorScheme.outline
        MessageStatus.SENDING -> AppIcons.Schedule to MaterialTheme.colorScheme.outline
        MessageStatus.SENT -> Icons.Default.Check to MaterialTheme.colorScheme.outline
        MessageStatus.DELIVERED -> AppIcons.DoneAll to MaterialTheme.colorScheme.outline
        MessageStatus.READ -> AppIcons.DoneAll to MaterialTheme.colorScheme.primary
        MessageStatus.FAILED -> AppIcons.Error to MaterialTheme.colorScheme.error
        MessageStatus.SYNCED -> AppIcons.DoneAll to MaterialTheme.colorScheme.primary
    }

    Icon(
        imageVector = icon,
        contentDescription = status.name,
        tint = color,
        modifier = Modifier.size(14.dp)
    )
}

@Composable
private fun SourceList(sources: List<SourceReference>) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        sources.take(3).forEach { source ->
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(4.dp))
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f))
                    .padding(6.dp)
            ) {
                Icon(
                    imageVector = getSourceIcon(source.type),
                    contentDescription = null,
                    modifier = Modifier.size(14.dp),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Spacer(modifier = Modifier.width(6.dp))
                Text(
                    text = source.title,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f)
                )
            }
        }
    }
}

@Composable
private fun AgentActionRow(
    actions: List<AgentAction>,
    onAction: (AgentAction) -> Unit
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        actions.take(3).forEach { action ->
            AssistChip(
                onClick = { onAction(action) },
                label = { Text(action.label) },
                leadingIcon = action.icon?.let {
                    {
                        Icon(
                            imageVector = getActionIcon(action.actionType),
                            contentDescription = null,
                            modifier = Modifier.size(14.dp)
                        )
                    }
                }
            )
        }
    }
}

@Composable
private fun CommandStatusBadge(status: CommandStatus) {
    val (color, text) = when (status) {
        CommandStatus.PENDING -> MaterialTheme.colorScheme.outline to "Pending"
        CommandStatus.EXECUTING -> MaterialTheme.colorScheme.primary to "Running"
        CommandStatus.COMPLETED -> MaterialTheme.colorScheme.tertiary to "Done"
        CommandStatus.FAILED -> MaterialTheme.colorScheme.error to "Failed"
        CommandStatus.CANCELLED -> MaterialTheme.colorScheme.outline to "Cancelled"
    }

    Surface(
        shape = RoundedCornerShape(4.dp),
        color = color.copy(alpha = 0.2f)
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            color = color,
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
        )
    }
}

// ========================================================================
// Helper Functions
// ========================================================================

@Composable
private fun getAgentIcon(type: AgentType): ImageVector {
    return when (type) {
        AgentType.GENERAL -> AppIcons.AutoAwesome
        AgentType.ANALYSIS -> AppIcons.Analytics
        AgentType.CODE_REVIEW -> AppIcons.Code
        AgentType.RESEARCH -> Icons.Default.Search
        AgentType.WRITING -> Icons.Default.Edit
        AgentType.TRANSLATION -> AppIcons.Translate
        AgentType.SCHEDULING -> AppIcons.Event
        AgentType.WORKFLOW -> AppIcons.AccountTree
        AgentType.PLATFORM_BRIDGE -> AppIcons.Link
    }
}

@Composable
private fun getSystemMessageStyle(type: SystemEventType): Pair<ImageVector, androidx.compose.ui.graphics.Color> {
    return when (type) {
        SystemEventType.WORKFLOW_STARTED -> Icons.Default.PlayArrow to MaterialTheme.colorScheme.primary
        SystemEventType.WORKFLOW_STEP -> AppIcons.Pending to MaterialTheme.colorScheme.primary
        SystemEventType.WORKFLOW_COMPLETED -> Icons.Default.CheckCircle to MaterialTheme.colorScheme.tertiary
        SystemEventType.WORKFLOW_FAILED -> AppIcons.Error to MaterialTheme.colorScheme.error
        SystemEventType.ROOM_CREATED -> Icons.Default.AddCircle to MaterialTheme.colorScheme.primary
        SystemEventType.USER_JOINED -> AppIcons.PersonAdd to MaterialTheme.colorScheme.primary
        SystemEventType.USER_LEFT -> AppIcons.PersonRemove to MaterialTheme.colorScheme.outline
        SystemEventType.USER_INVITED -> AppIcons.Mail to MaterialTheme.colorScheme.primary
        SystemEventType.ENCRYPTION_ENABLED -> Icons.Default.Lock to MaterialTheme.colorScheme.tertiary
        SystemEventType.VERIFICATION_REQUIRED -> AppIcons.VerifiedUser to MaterialTheme.colorScheme.secondary
        SystemEventType.DEVICE_ADDED -> AppIcons.Devices to MaterialTheme.colorScheme.primary
        SystemEventType.PLATFORM_CONNECTED -> AppIcons.Link to MaterialTheme.colorScheme.tertiary
        SystemEventType.PLATFORM_DISCONNECTED -> AppIcons.LinkOff to MaterialTheme.colorScheme.error
        SystemEventType.BUDGET_WARNING -> Icons.Default.Warning to MaterialTheme.colorScheme.tertiary
        SystemEventType.BUDGET_EXCEEDED -> AppIcons.Error to MaterialTheme.colorScheme.error
        SystemEventType.LICENSE_WARNING -> Icons.Default.Warning to MaterialTheme.colorScheme.tertiary
        SystemEventType.LICENSE_EXPIRED -> AppIcons.Error to MaterialTheme.colorScheme.error
        SystemEventType.CONTENT_POLICY_APPLIED -> AppIcons.VerifiedUser to MaterialTheme.colorScheme.tertiary
        SystemEventType.INFO -> Icons.Default.Info to MaterialTheme.colorScheme.primary
        SystemEventType.WARNING -> Icons.Default.Warning to MaterialTheme.colorScheme.tertiary
        SystemEventType.ERROR -> AppIcons.Error to MaterialTheme.colorScheme.error
    }
}

@Composable
private fun getSourceIcon(type: SourceType): ImageVector {
    return when (type) {
        SourceType.DOCUMENT -> AppIcons.Description
        SourceType.WEB_PAGE -> AppIcons.Language
        SourceType.CODE_FILE -> AppIcons.Code
        SourceType.MESSAGE -> AppIcons.Message
        SourceType.EXTERNAL_PLATFORM -> AppIcons.Link
    }
}

@Composable
private fun getActionIcon(type: AgentActionType): ImageVector {
    return when (type) {
        AgentActionType.COPY -> AppIcons.ContentCopy
        AgentActionType.REGENERATE -> Icons.Default.Refresh
        AgentActionType.FOLLOW_UP -> AppIcons.QuestionAnswer
        AgentActionType.APPLY -> Icons.Default.Check
        AgentActionType.SHARE -> Icons.Default.Share
        AgentActionType.DOWNLOAD -> AppIcons.Download
        AgentActionType.VIEW_SOURCE -> AppIcons.OpenInNew
        AgentActionType.EXECUTE -> Icons.Default.PlayArrow
    }
}

private fun formatTime(instant: Instant): String {
    val dateTime = instant.toLocalDateTime(TimeZone.currentSystemDefault())
    val hour = dateTime.hour.toString().padStart(2, '0')
    val minute = dateTime.minute.toString().padStart(2, '0')
    return "$hour:$minute"
}
