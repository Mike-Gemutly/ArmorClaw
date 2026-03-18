package com.armorclaw.app.screens.chat.components
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.OnBackground

/**
 * Inline thread preview shown below a message
 * Displays the last reply with participant count and unread indicator
 */
@Composable
fun ThreadInlinePreview(
    replyCount: Int,
    lastReplySender: String? = null,
    lastReplyContent: String? = null,
    participantCount: Int = 0,
    hasUnread: Boolean = false,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    if (replyCount <= 0) return

    Row(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .background(
                if (hasUnread) BrandPurple.copy(alpha = 0.08f)
                else OnBackground.copy(alpha = 0.03f)
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Thread icon with reply count
        Box(
            modifier = Modifier
                .size(28.dp)
                .clip(CircleShape)
                .background(
                    if (hasUnread) BrandPurple.copy(alpha = 0.2f)
                    else OnBackground.copy(alpha = 0.08f)
                ),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = Icons.Default.Chat,
                contentDescription = null,
                tint = if (hasUnread) BrandPurple else OnBackground.copy(alpha = 0.5f),
                modifier = Modifier.size(14.dp)
            )
        }

        // Preview content
        Column(modifier = Modifier.weight(1f)) {
            if (!lastReplySender.isNullOrBlank() && !lastReplyContent.isNullOrBlank()) {
                // Show last reply preview
                Text(
                    text = buildString {
                        append(lastReplySender)
                        append(": ")
                        append(lastReplyContent)
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = if (hasUnread) OnBackground else OnBackground.copy(alpha = 0.7f),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    fontWeight = if (hasUnread) FontWeight.Medium else FontWeight.Normal
                )
            } else {
                // Show reply count only
                Text(
                    text = "$replyCount ${if (replyCount == 1) "reply" else "replies"}",
                    style = MaterialTheme.typography.bodySmall,
                    color = if (hasUnread) BrandPurple else OnBackground.copy(alpha = 0.6f),
                    fontWeight = if (hasUnread) FontWeight.Medium else FontWeight.Normal
                )
            }

            // Participant count (if > 1)
            if (participantCount > 1) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.People,
                        contentDescription = null,
                        tint = OnBackground.copy(alpha = 0.4f),
                        modifier = Modifier.size(10.dp)
                    )
                    Text(
                        text = "$participantCount participants",
                        style = MaterialTheme.typography.bodySmall,
                        color = OnBackground.copy(alpha = 0.5f)
                    )
                }
            }
        }

        // Unread indicator
        if (hasUnread) {
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .clip(CircleShape)
                    .background(BrandGreen)
            )
        }

        // Expand arrow
        Icon(
            imageVector = Icons.Default.ChevronRight,
            contentDescription = "View thread",
            tint = OnBackground.copy(alpha = 0.3f),
            modifier = Modifier.size(16.dp)
        )
    }
}

/**
 * Minimal thread indicator for compact message lists
 * Shows just an icon with count
 */
@Composable
fun ThreadCompactIndicator(
    replyCount: Int,
    hasUnread: Boolean = false,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    if (replyCount <= 0) return

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(
                if (hasUnread) BrandPurple.copy(alpha = 0.1f)
                else Color.Transparent
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 8.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = Icons.Default.ChatBubbleOutline,
            contentDescription = null,
            tint = if (hasUnread) BrandPurple else OnBackground.copy(alpha = 0.5f),
            modifier = Modifier.size(12.dp)
        )

        Text(
            text = replyCount.toString(),
            style = MaterialTheme.typography.bodySmall,
            color = if (hasUnread) BrandPurple else OnBackground.copy(alpha = 0.6f),
            fontWeight = if (hasUnread) FontWeight.Bold else FontWeight.Normal
        )

        if (hasUnread) {
            Spacer(modifier = Modifier.width(2.dp))
            Box(
                modifier = Modifier
                    .size(6.dp)
                    .clip(CircleShape)
                    .background(BrandGreen)
            )
        }
    }
}

/**
 * Thread summary shown in room list items
 * Displays thread activity for a room
 */
@Composable
fun ThreadRoomSummary(
    activeThreadCount: Int,
    threadUnreadCount: Int,
    lastThreadActivity: String? = null,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    if (activeThreadCount <= 0) return

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(
                if (threadUnreadCount > 0) BrandPurple.copy(alpha = 0.1f)
                else OnBackground.copy(alpha = 0.05f)
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 8.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = Icons.Default.Forum,
            contentDescription = "Threads",
            tint = if (threadUnreadCount > 0) BrandPurple else OnBackground.copy(alpha = 0.5f),
            modifier = Modifier.size(14.dp)
        )

        Text(
            text = "$activeThreadCount active",
            style = MaterialTheme.typography.bodySmall,
            color = if (threadUnreadCount > 0) BrandPurple else OnBackground.copy(alpha = 0.6f)
        )

        if (threadUnreadCount > 0) {
            Box(
                modifier = Modifier
                    .size(16.dp)
                    .clip(CircleShape)
                    .background(BrandGreen),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = if (threadUnreadCount > 99) "99+" else threadUnreadCount.toString(),
                    style = MaterialTheme.typography.bodySmall,
                    color = Color.White,
                    fontSize = androidx.compose.ui.unit.TextUnit.Unspecified
                )
            }
        }
    }
}
