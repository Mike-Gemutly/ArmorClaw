package com.armorclaw.app.screens.chat.components

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Reply
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandPurpleLight
import com.armorclaw.shared.ui.theme.OnBackground

@Composable
fun ReplyPreviewBar(
    replyTo: com.armorclaw.shared.domain.model.UnifiedMessage,
    onCancelReply: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        shape = RoundedCornerShape(8.dp),
        color = BrandPurpleLight.copy(alpha = 0.3f),
        border = BorderStroke(1.dp, BrandPurple.copy(alpha = 0.3f))
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Reply icon
            Icon(
                imageVector = Icons.Default.Reply,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(20.dp)
            )
            
            // Vertical line
            Box(
                modifier = Modifier
                    .width(2.dp)
                    .height(40.dp)
                    .background(BrandPurple.copy(alpha = 0.5f))
            )
            
            // Reply content
            Column(
                modifier = Modifier.weight(1f)
            ) {
                val senderName = when (val s = replyTo.sender) {
                    is com.armorclaw.shared.domain.model.MessageSender.UserSender -> s.displayName
                    is com.armorclaw.shared.domain.model.MessageSender.AgentSender -> s.displayName
                    is com.armorclaw.shared.domain.model.MessageSender.SystemSender -> s.displayName
                }
                Text(
                    text = "Replying to $senderName",
                    style = MaterialTheme.typography.bodySmall.copy(
                        fontWeight = FontWeight.Bold,
                        color = BrandPurple
                    )
                )

                val bodyText = when (replyTo) {
                    is com.armorclaw.shared.domain.model.UnifiedMessage.Regular -> replyTo.content.body
                    is com.armorclaw.shared.domain.model.UnifiedMessage.Agent -> replyTo.content.body
                    is com.armorclaw.shared.domain.model.UnifiedMessage.System -> replyTo.title
                    is com.armorclaw.shared.domain.model.UnifiedMessage.Command -> replyTo.command
                }
                Text(
                    text = bodyText,
                    style = MaterialTheme.typography.bodyMedium,
                    color = OnBackground.copy(alpha = 0.8f),
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }
            
            // Cancel button
            IconButton(
                onClick = onCancelReply,
                modifier = Modifier.size(32.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "Cancel reply",
                    tint = BrandPurple,
                    modifier = Modifier.size(20.dp)
                )
            }
        }
    }
}

@Composable
fun ForwardPreviewBar(
    messages: List<ChatMessage>,
    onCancelForward: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        shape = RoundedCornerShape(8.dp),
        color = BrandPurpleLight.copy(alpha = 0.3f),
        border = BorderStroke(1.dp, BrandPurple.copy(alpha = 0.3f))
    ) {
        Column(
            modifier = Modifier.padding(12.dp)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Forwarding ${messages.size} message${if (messages.size > 1) "s" else ""}",
                    style = MaterialTheme.typography.titleSmall.copy(
                        fontWeight = FontWeight.Bold,
                        color = BrandPurple
                    )
                )
                
                Spacer(modifier = Modifier.weight(1f))
                
                IconButton(
                    onClick = onCancelForward,
                    modifier = Modifier.size(24.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = "Cancel forward",
                        tint = BrandPurple,
                        modifier = Modifier.size(16.dp)
                    )
                }
            }
            
            Spacer(modifier = Modifier.height(8.dp))
            
            // Preview of messages
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                messages.take(3).forEach { message ->
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        // Avatar placeholder
                        Box(
                            modifier = Modifier
                                .size(24.dp)
                                .clip(CircleShape)
                                .background(BrandPurple.copy(alpha = 0.3f)),
                            contentAlignment = Alignment.Center
                        ) {
                            Text(
                                text = message.senderName.first().uppercaseChar().toString(),
                                style = MaterialTheme.typography.bodySmall.copy(
                                    fontWeight = FontWeight.Bold,
                                    color = BrandPurple
                                ),
                                textAlign = TextAlign.Center
                            )
                        }
                        
                        Text(
                            text = message.content,
                            style = MaterialTheme.typography.bodySmall,
                            color = OnBackground.copy(alpha = 0.7f),
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f)
                        )
                    }
                }
                
                if (messages.size > 3) {
                    Text(
                        text = "+ ${messages.size - 3} more",
                        style = MaterialTheme.typography.bodySmall,
                        color = BrandPurple
                    )
                }
            }
        }
    }
}
