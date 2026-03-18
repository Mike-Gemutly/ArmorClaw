package com.armorclaw.app.screens.chat.components
import androidx.compose.foundation.layout.Arrangement
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.model.ThreadInfo
import com.armorclaw.shared.ui.theme.*
import kotlinx.datetime.Instant

/**
 * Full thread panel showing the thread root and all replies
 * Used as a bottom sheet or side panel in the chat screen
 */
@Composable
fun ThreadView(
    threadRoot: Message,
    replies: List<Message>,
    isLoading: Boolean = false,
    isSending: Boolean = false,
    onSendReply: (String) -> Unit = {},
    onBackClick: () -> Unit = {},
    onMessageClick: (Message) -> Unit = {},
    onReactionClick: (Message, String) -> Unit = { _, _ -> },
    modifier: Modifier = Modifier
) {
    val listState = rememberLazyListState()
    var replyText by remember { mutableStateOf("") }

    // Scroll to bottom when new replies arrive
    LaunchedEffect(replies.size) {
        if (replies.isNotEmpty()) {
            listState.animateScrollToItem(replies.size)
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
    ) {
        // Thread header
        ThreadHeader(
            threadRoot = threadRoot,
            replyCount = replies.size,
            onBackClick = onBackClick
        )

        Divider(
            color = OnBackground.copy(alpha = 0.1f),
            thickness = 1.dp
        )

        // Messages list
        Box(modifier = Modifier.weight(1f)) {
            if (isLoading && replies.isEmpty()) {
                // Loading state
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator(
                        color = BrandPurple,
                        modifier = Modifier.size(32.dp)
                    )
                }
            } else {
                LazyColumn(
                    state = listState,
                    contentPadding = PaddingValues(
                        vertical = DesignTokens.Spacing.sm,
                        horizontal = DesignTokens.Spacing.md
                    ),
                    verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.xs)
                ) {
                    // Thread root message
                    item {
                        ThreadRootMessage(
                            message = threadRoot,
                            onClick = { onMessageClick(threadRoot) },
                            onReactionClick = { emoji -> onReactionClick(threadRoot, emoji) }
                        )
                    }

                    // Thread divider
                    if (replies.isNotEmpty()) {
                        item {
                            ThreadRepliesDivider(count = replies.size)
                        }
                    }

                    // Reply messages
                    items(
                        items = replies,
                        key = { it.id }
                    ) { message ->
                        ThreadReplyMessage(
                            message = message,
                            onClick = { onMessageClick(message) },
                            onReactionClick = { emoji -> onReactionClick(message, emoji) }
                        )
                    }

                    // Empty state
                    if (replies.isEmpty() && !isLoading) {
                        item {
                            ThreadEmptyState()
                        }
                    }
                }
            }
        }

        // Reply input
        ThreadReplyInput(
            text = replyText,
            onTextChange = { replyText = it },
            onSend = {
                if (replyText.isNotBlank()) {
                    onSendReply(replyText)
                    replyText = ""
                }
            },
            isSending = isSending,
            enabled = true
        )
    }
}

@Composable
private fun ThreadHeader(
    threadRoot: Message,
    replyCount: Int,
    onBackClick: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(DesignTokens.Spacing.md),
        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Back button
        IconButton(onClick = onBackClick) {
            Icon(
                imageVector = Icons.Filled.ArrowBack,
                contentDescription = "Back",
                tint = OnBackground
            )
        }

        // Thread info
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = "Thread",
                style = MaterialTheme.typography.titleMedium,
                color = OnBackground,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = when {
                    replyCount == 0 -> "No replies yet"
                    replyCount == 1 -> "1 reply"
                    else -> "$replyCount replies"
                },
                style = MaterialTheme.typography.bodySmall,
                color = OnBackground.copy(alpha = 0.6f)
            )
        }

        // Encryption indicator
        Icon(
            imageVector = Icons.Default.Lock,
            contentDescription = "Encrypted",
            tint = BrandGreen,
            modifier = Modifier.size(20.dp)
        )
    }
}

@Composable
private fun ThreadRootMessage(
    message: Message,
    onClick: () -> Unit,
    onReactionClick: (String) -> Unit
) {
    Surface(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        color = BrandPurple.copy(alpha = 0.1f),
        border = androidx.compose.foundation.BorderStroke(1.dp, BrandPurple.copy(alpha = 0.3f))
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md)
        ) {
            // Sender info
            Row(
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Avatar placeholder
                Box(
                    modifier = Modifier
                        .size(32.dp)
                        .clip(CircleShape)
                        .background(BrandPurple),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = message.senderId.take(1).uppercase(),
                        color = Color.White,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Bold
                    )
                }

                Column {
                    Text(
                        text = message.senderId,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Bold,
                        color = OnBackground
                    )
                    Text(
                        text = formatThreadTimestamp(message.timestamp),
                        style = MaterialTheme.typography.bodySmall,
                        color = OnBackground.copy(alpha = 0.6f)
                    )
                }
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            // Message content
            Text(
                text = message.content.body,
                style = MaterialTheme.typography.bodyLarge,
                color = OnBackground
            )

            // Reactions
            if (message.reactions.isNotEmpty()) {
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.xs))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    message.reactions.forEach { reaction ->
                        Surface(
                            onClick = { onReactionClick(reaction.emoji) },
                            shape = RoundedCornerShape(12.dp),
                            color = if (reaction.includesMe)
                                BrandPurple.copy(alpha = 0.2f)
                            else
                                Color.White.copy(alpha = 0.1f)
                        ) {
                            Row(
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(text = reaction.emoji)
                                if (reaction.count > 1) {
                                    Text(
                                        text = " ${reaction.count}",
                                        style = MaterialTheme.typography.bodySmall
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ThreadRepliesDivider(count: Int) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = DesignTokens.Spacing.sm),
        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Divider(
            modifier = Modifier.weight(1f),
            color = OnBackground.copy(alpha = 0.2f)
        )
        Text(
            text = "$count ${if (count == 1) "reply" else "replies"}",
            style = MaterialTheme.typography.bodySmall,
            color = OnBackground.copy(alpha = 0.6f)
        )
        Divider(
            modifier = Modifier.weight(1f),
            color = OnBackground.copy(alpha = 0.2f)
        )
    }
}

@Composable
private fun ThreadReplyMessage(
    message: Message,
    onClick: () -> Unit,
    onReactionClick: (String) -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = DesignTokens.Spacing.lg) // Indent replies
    ) {
        // Thread line
        Box(
            modifier = Modifier
                .width(2.dp)
                .height(IntrinsicSize.Max)
                .padding(end = DesignTokens.Spacing.sm)
        ) {
            Box(
                modifier = Modifier
                    .fillMaxHeight()
                    .width(2.dp)
                    .background(BrandPurple.copy(alpha = 0.3f))
            )
        }

        // Message content
        Column(modifier = Modifier.weight(1f)) {
            // Sender and timestamp
            Row(
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.xs),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = message.senderId,
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.Bold,
                    color = OnBackground
                )
                Text(
                    text = "•",
                    style = MaterialTheme.typography.bodySmall,
                    color = OnBackground.copy(alpha = 0.4f)
                )
                Text(
                    text = formatThreadTimestamp(message.timestamp),
                    style = MaterialTheme.typography.bodySmall,
                    color = OnBackground.copy(alpha = 0.6f)
                )
            }

            Spacer(modifier = Modifier.height(2.dp))

            // Content
            Text(
                text = message.content.body,
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground
            )

            // Reactions
            if (message.reactions.isNotEmpty()) {
                Spacer(modifier = Modifier.height(4.dp))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    message.reactions.forEach { reaction ->
                        Text(
                            text = "${reaction.emoji} ${reaction.count}",
                            style = MaterialTheme.typography.bodySmall,
                            color = OnBackground.copy(alpha = 0.7f)
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ThreadEmptyState() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(DesignTokens.Spacing.xl),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(
            imageVector = Icons.Default.ChatBubbleOutline,
            contentDescription = null,
            tint = OnBackground.copy(alpha = 0.3f),
            modifier = Modifier.size(48.dp)
        )
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
        Text(
            text = "No replies yet",
            style = MaterialTheme.typography.bodyLarge,
            color = OnBackground.copy(alpha = 0.6f)
        )
        Text(
            text = "Be the first to reply!",
            style = MaterialTheme.typography.bodySmall,
            color = OnBackground.copy(alpha = 0.4f)
        )
    }
}

@Composable
private fun ThreadReplyInput(
    text: String,
    onTextChange: (String) -> Unit,
    onSend: () -> Unit,
    isSending: Boolean,
    enabled: Boolean
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        color = MaterialTheme.colorScheme.surface,
        tonalElevation = 8.dp,
        shadowElevation = 8.dp
    ) {
        Row(
            modifier = Modifier
                .padding(DesignTokens.Spacing.md)
                .imePadding(),
            horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
            verticalAlignment = Alignment.CenterVertically
        ) {
            TextField(
                value = text,
                onValueChange = onTextChange,
                modifier = Modifier.weight(1f),
                placeholder = {
                    Text(
                        text = "Reply to thread...",
                        color = OnBackground.copy(alpha = 0.4f)
                    )
                },
                enabled = enabled && !isSending,
                singleLine = false,
                maxLines = 4,
                colors = OutlinedTextFieldDefaults.colors(
                    focusedContainerColor = Color.Transparent,
                    unfocusedContainerColor = Color.Transparent,
                    cursorColor = BrandPurple,
                    focusedBorderColor = BrandPurple,
                    unfocusedBorderColor = OnBackground.copy(alpha = 0.2f)
                )
            )

            IconButton(
                onClick = onSend,
                enabled = enabled && text.isNotBlank() && !isSending
            ) {
                if (isSending) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(24.dp),
                        color = BrandPurple,
                        strokeWidth = 2.dp
                    )
                } else {
                    Icon(
                        imageVector = Icons.Filled.Send,
                        contentDescription = "Send",
                        tint = if (text.isNotBlank()) BrandPurple else OnBackground.copy(alpha = 0.3f)
                    )
                }
            }
        }
    }
}

private fun formatThreadTimestamp(timestamp: Instant): String {
    val now = kotlinx.datetime.Clock.System.now()
    val diff = now - timestamp

    return when {
        diff.inWholeMinutes < 1 -> "Just now"
        diff.inWholeMinutes < 60 -> "${diff.inWholeMinutes}m"
        diff.inWholeHours < 24 -> "${diff.inWholeHours}h"
        diff.inWholeDays < 7 -> "${diff.inWholeDays}d"
        else -> {
            val date = timestamp.toLocalDateTime(TimeZone.currentSystemDefault()).date
            "${date.dayOfMonth} ${date.month.name.take(3)}"
        }
    }
}
