package com.armorclaw.app.screens.chat.components

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateIntAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChatBubbleOutline
import androidx.compose.material.icons.filled.Forum
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
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
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.OnBackground
import com.armorclaw.shared.ui.theme.Primary

/**
 * Badge showing thread reply count with optional unread indicator
 * Displays at the bottom of messages that have thread replies
 */
@Composable
fun ThreadBadge(
    replyCount: Int,
    hasUnread: Boolean = false,
    unreadCount: Int = 0,
    lastReplyPreview: String? = null,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val backgroundColor by animateColorAsState(
        targetValue = if (hasUnread)
            BrandPurple.copy(alpha = 0.15f)
        else
            Color.Transparent,
        label = "background_color"
    )

    val borderColor by animateColorAsState(
        targetValue = if (hasUnread)
            BrandPurple
        else
            BrandPurple.copy(alpha = 0.3f),
        label = "border_color"
    )

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(backgroundColor)
            .border(
                width = 1.dp,
                color = borderColor,
                shape = RoundedCornerShape(12.dp)
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Thread icon
        Icon(
            imageVector = if (replyCount > 0) Icons.Default.Forum else Icons.Default.ChatBubbleOutline,
            contentDescription = "Thread",
            tint = if (hasUnread) BrandPurple else OnBackground.copy(alpha = 0.6f),
            modifier = Modifier.size(14.dp)
        )

        // Reply count text
        Text(
            text = when {
                replyCount == 0 -> "Start thread"
                replyCount == 1 -> "1 reply"
                else -> "$replyCount replies"
            },
            color = if (hasUnread) BrandPurple else OnBackground.copy(alpha = 0.8f),
            style = MaterialTheme.typography.bodySmall,
            fontWeight = if (hasUnread) FontWeight.Bold else FontWeight.Normal
        )

        // Unread indicator
        if (hasUnread && unreadCount > 0) {
            UnreadBadge(count = unreadCount)
        }

        // Last reply preview (optional)
        if (!lastReplyPreview.isNullOrBlank() && replyCount > 0) {
            Text(
                text = "• $lastReplyPreview",
                color = OnBackground.copy(alpha = 0.5f),
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false)
            )
        }
    }
}

/**
 * Small circular badge showing unread count
 */
@Composable
private fun UnreadBadge(
    count: Int,
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier
            .size(18.dp)
            .clip(CircleShape)
            .background(BrandGreen),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = if (count > 99) "99+" else count.toString(),
            color = Color.White,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Bold,
            maxLines = 1
        )
    }
}

/**
 * Compact thread indicator for message list items
 * Shows a minimal indicator that a message has thread replies
 */
@Composable
fun ThreadIndicator(
    replyCount: Int,
    hasUnread: Boolean = false,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    if (replyCount <= 0) return

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(
                if (hasUnread) BrandPurple.copy(alpha = 0.1f)
                else Color.Transparent
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 6.dp, vertical = 2.dp),
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
            color = if (hasUnread) BrandPurple else OnBackground.copy(alpha = 0.6f),
            style = MaterialTheme.typography.bodySmall,
            fontWeight = if (hasUnread) FontWeight.Bold else FontWeight.Normal
        )
    }
}
