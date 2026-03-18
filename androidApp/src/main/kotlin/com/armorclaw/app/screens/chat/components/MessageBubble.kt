package com.armorclaw.app.screens.chat.components
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.DoneAll
import androidx.compose.material.icons.filled.Error
import androidx.compose.material.icons.filled.HourglassEmpty
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.outlined.Image
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Popup
import androidx.compose.ui.window.PopupProperties
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandPurpleLight
import com.armorclaw.shared.ui.theme.BrandRed
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.OnBackground
import com.armorclaw.shared.ui.theme.Primary
import com.armorclaw.app.screens.chat.components.ReactionDisplay

data class ChatMessage(
    val id: String,
    val content: String,
    val isOutgoing: Boolean,
    val timestamp: kotlinx.datetime.Instant,
    val senderName: String,
    val senderAvatar: String? = null,
    val status: MessageStatus = MessageStatus.SENT,
    val isEncrypted: Boolean = true,
    val replyTo: ChatMessage? = null,
    val attachments: List<MessageAttachment> = emptyList(),
    val reactions: List<MessageReaction> = emptyList(),
    val isEdited: Boolean = false
)

data class MessageStatus(
    val type: StatusType
) {
    companion object {
        val SENDING = MessageStatus(StatusType.SENDING)
        val SENT = MessageStatus(StatusType.SENT)
        val DELIVERED = MessageStatus(StatusType.DELIVERED)
        val READ = MessageStatus(StatusType.READ)
        val FAILED = MessageStatus(StatusType.FAILED)
    }
}

enum class StatusType {
    SENDING,
    SENT,
    DELIVERED,
    READ,
    FAILED
}

data class MessageAttachment(
    val id: String,
    val type: AttachmentType,
    val url: String,
    val fileName: String,
    val fileSize: Long,
    val thumbnailUrl: String? = null
)

enum class AttachmentType {
    IMAGE,
    FILE,
    AUDIO,
    VIDEO,
    LOCATION
}

data class MessageReaction(
    val emoji: String,
    val count: Int,
    val hasReacted: Boolean
)

@OptIn(androidx.compose.foundation.ExperimentalFoundationApi::class)
@Composable
fun MessageBubble(
    message: ChatMessage,
    modifier: Modifier = Modifier,
    onReplyClick: (ChatMessage) -> Unit = {},
    onReactionClick: (ChatMessage) -> Unit = {},
    onEmojiSelected: (String) -> Unit = {},
    onAttachmentClick: (MessageAttachment) -> Unit = {}
) {
    var showReactionPicker by remember { mutableStateOf(false) }
    val bubbleColor = if (message.isOutgoing) {
        BrandPurple
    } else {
        BrandPurpleLight.copy(alpha = 0.5f)
    }
    
    val textColor = if (message.isOutgoing) {
        Color.White
    } else {
        OnBackground
    }
    
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = DesignTokens.Spacing.md, vertical = DesignTokens.Spacing.xs),
        horizontalArrangement = if (message.isOutgoing)
            Arrangement.End
        else
            Arrangement.Start
    ) {
        Box(modifier = Modifier.weight(1f)) {
            Surface(
                modifier = Modifier
                    .combinedClickable(
                        onClick = { onReplyClick(message) },
                        onLongClick = { showReactionPicker = true }
                    ),
                shape = RoundedCornerShape(
                        topStart = if (message.isOutgoing) 16.dp else 0.dp,
                        topEnd = if (message.isOutgoing) 0.dp else 16.dp,
                        bottomStart = 16.dp,
                        bottomEnd = 16.dp
                    ),
                color = bubbleColor,
                tonalElevation = 2.dp,
                shadowElevation = 2.dp,
                border = if (message.isEncrypted)
                        BorderStroke(1.dp, if (message.isOutgoing) Color.White.copy(alpha = 0.3f) else BrandPurple.copy(alpha = 0.2f))
                else
                        null
            ) {
                Column(
                    modifier = Modifier.padding(
                        horizontal = 16.dp,
                        vertical = 10.dp
                    )
                ) {
                // Reply preview
                if (message.replyTo != null) {
                    ReplyPreviewBubble(
                        replyTo = message.replyTo!!,
                        textColor = textColor
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                }
                
                // Content
                Text(
                    text = message.content,
                    color = textColor,
                    style = MaterialTheme.typography.bodyLarge,
                    modifier = Modifier.fillMaxWidth()
                )
                
                // Attachments
                if (message.attachments.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(8.dp))
                    message.attachments.forEach { attachment ->
                        AttachmentPreview(
                            attachment = attachment,
                            textColor = textColor,
                            onClick = { onAttachmentClick(attachment) }
                        )
                    }
                }
                
                // Reactions
                if (message.reactions.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(8.dp))
                    ReactionDisplay(
                        reactions = message.reactions,
                        onReactionClick = { onReactionClick(message) },
                        onAddReaction = { showReactionPicker = true }
                    )
                }
                
                // Footer (timestamp, status, encryption)
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 4.dp),
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = formatTimestamp(message.timestamp),
                        color = textColor.copy(alpha = 0.7f),
                        style = MaterialTheme.typography.bodySmall
                    )
                    
                    if (message.isEdited) {
                        Text(
                            text = "edited",
                            color = textColor.copy(alpha = 0.7f),
                            style = MaterialTheme.typography.bodySmall,
                            fontStyle = androidx.compose.ui.text.font.FontStyle.Italic
                        )
                    }
                    
                    Spacer(modifier = Modifier.weight(1f))
                    
                    // Encryption indicator
                    if (message.isEncrypted) {
                        Icon(
                            imageVector = Icons.Default.Lock,
                            contentDescription = "Encrypted",
                            tint = textColor.copy(alpha = 0.7f),
                            modifier = Modifier.size(10.dp)
                        )
                    }
                    
                    // Status indicator
                    if (message.isOutgoing) {
                        MessageStatusIcon(status = message.status, textColor = textColor)
                    }
                }
            }
            
if (showReactionPicker) {
            ReactionPickerOverlay(
                onReactionSelected = { emoji -> onEmojiSelected(emoji); showReactionPicker = false },
                onDismiss = { showReactionPicker = false }
            )
        }
        }
    }
}

}

@Composable
private fun ReplyPreviewBubble(
    replyTo: ChatMessage,
    textColor: Color
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(textColor.copy(alpha = 0.1f))
            .border(
                width = 1.dp,
                color = textColor.copy(alpha = 0.2f),
                shape = RoundedCornerShape(8.dp)
            )
            .padding(8.dp)
    ) {
        Column(
            modifier = Modifier.weight(1f)
        ) {
            Text(
                text = replyTo.senderName,
                color = textColor.copy(alpha = 0.7f),
                style = MaterialTheme.typography.bodySmall,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = replyTo.content,
                color = textColor.copy(alpha = 0.9f),
                style = MaterialTheme.typography.bodySmall,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis
            )
        }
    }
}

@Composable
private fun AttachmentPreview(
    attachment: MessageAttachment,
    textColor: Color,
    onClick: () -> Unit
) {
    Surface(
        onClick = onClick,
        modifier = Modifier
            .fillMaxWidth()
            .border(
                width = 1.dp,
                color = textColor.copy(alpha = 0.3f),
                shape = RoundedCornerShape(8.dp)
            ),
        shape = RoundedCornerShape(8.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Attachment icon
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .clip(CircleShape)
                    .background(textColor.copy(alpha = 0.1f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = when (attachment.type) {
                        AttachmentType.IMAGE -> Icons.Outlined.Image
                        else -> Icons.Default.AttachFile
                    },
                    contentDescription = null,
                    tint = textColor,
                    modifier = Modifier.size(20.dp)
                )
            }
            
            // Attachment info
            Column(
                modifier = Modifier.weight(1f)
            ) {
                Text(
                    text = attachment.fileName,
                    color = textColor,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    text = formatFileSize(attachment.fileSize),
                    color = textColor.copy(alpha = 0.7f),
                    style = MaterialTheme.typography.bodySmall
                )
            }
        }
    }
}

@Composable
private fun MessageReactionsRow(
    reactions: List<MessageReaction>,
    onClick: () -> Unit
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        reactions.forEach { reaction ->
            ReactionBadge(
                emoji = reaction.emoji,
                count = reaction.count,
                hasReacted = reaction.hasReacted,
                onClick = onClick
            )
        }
    }
}

@Composable
private fun ReactionBadge(
    emoji: String,
    count: Int,
    hasReacted: Boolean,
    onClick: () -> Unit
) {
    Surface(
        onClick = onClick,
        shape = CircleShape,
        color = if (hasReacted)
            BrandPurple.copy(alpha = 0.3f)
        else
            Color.White.copy(alpha = 0.2f),
        border = if (hasReacted)
            BorderStroke(1.dp, BrandPurple)
        else
            BorderStroke(1.dp, Color.White.copy(alpha = 0.3f))
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
            horizontalArrangement = Arrangement.spacedBy(2.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = emoji,
                style = MaterialTheme.typography.bodySmall
            )
            if (count > 1) {
                Text(
                    text = count.toString(),
                    style = MaterialTheme.typography.bodySmall,
                    color = if (hasReacted) BrandPurple else Color.Black
                )
            }
        }
    }
}

@Composable
private fun MessageStatusIcon(
    status: MessageStatus,
    textColor: Color
) {
    val (icon, color, description) = when (status.type) {
        StatusType.SENDING -> Triple(Icons.Default.HourglassEmpty, textColor.copy(alpha = 0.5f), "Sending message")
        StatusType.SENT -> Triple(Icons.Default.Check, textColor.copy(alpha = 0.7f), "Message sent")
        StatusType.DELIVERED -> Triple(Icons.Default.DoneAll, textColor.copy(alpha = 0.7f), "Message delivered")
        StatusType.READ -> Triple(Icons.Default.DoneAll, BrandGreen, "Message read")
        StatusType.FAILED -> Triple(Icons.Default.Error, BrandRed, "Message failed to send")
    }

    Icon(
        imageVector = icon,
        contentDescription = description,
        tint = color,
        modifier = Modifier.size(14.dp)
    )
}

private fun formatTimestamp(timestamp: kotlinx.datetime.Instant): String {
    val now = kotlinx.datetime.Clock.System.now()
    val diff = now - timestamp
    
    return when {
        diff.inWholeMinutes < 1 -> "Just now"
        diff.inWholeMinutes < 60 -> "${diff.inWholeMinutes}m ago"
        diff.inWholeHours < 24 -> "${diff.inWholeHours}h ago"
        diff.inWholeDays < 7 -> "${diff.inWholeDays}d ago"
        else -> {
            val date = timestamp.toLocalDateTime(TimeZone.currentSystemDefault()).date
            "${date.dayOfMonth} ${date.month.name.take(3)}"
        }
    }
}

private fun formatFileSize(bytes: Long): String {
    return when {
        bytes < 1024 -> "$bytes B"
        bytes < 1024 * 1024 -> "${bytes / 1024} KB"
        bytes < 1024 * 1024 * 1024 -> "${bytes / (1024 * 1024)} MB"
        else -> "${bytes / (1024 * 1024 * 1024)} GB"
    }
}
