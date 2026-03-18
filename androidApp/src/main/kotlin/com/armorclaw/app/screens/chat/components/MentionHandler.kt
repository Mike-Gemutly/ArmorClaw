package com.armorclaw.app.screens.chat.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.BrandPurple

/**
 * Represents a mention in a message
 */
data class Mention(
    val type: MentionType,
    val id: String,
    val displayText: String,
    val startIndex: Int,
    val endIndex: Int
)

/**
 * Type of mention
 */
enum class MentionType {
    USER,       // @username
    ROOM,       // #room:server.com
    ROOM_ALIAS, // #room
    EVERYONE,   // @room or @everyone
    LINK        // URLs
}

/**
 * Handles detection and styling of mentions in text
 */
object MentionHandler {

    // Regex patterns for mentions
    private val userMentionRegex = """@([a-zA-Z0-9_-]+)(?::([a-zA-Z0-9.-]+))?""".toRegex()
    private val roomMentionRegex = """#([a-zA-Z0-9_-]+):([a-zA-Z0-9.-]+)""".toRegex()
    private val roomAliasRegex = """#([a-zA-Z0-9_-]+)""".toRegex()
    private val everyoneRegex = """@(room|everyone|all)\b""".toRegex(RegexOption.IGNORE_CASE)
    private val urlRegex = """https?://[^\s]+""".toRegex()

    /**
     * Find all mentions in text
     */
    fun findMentions(text: String): List<Mention> {
        val mentions = mutableListOf<Mention>()

        // Find user mentions
        userMentionRegex.findAll(text).forEach { match ->
            val username = if (match.groupValues[2].isNotEmpty()) {
                "@${match.groupValues[1]}:${match.groupValues[2]}"
            } else {
                "@${match.groupValues[1]}"
            }
            mentions.add(
                Mention(
                    type = MentionType.USER,
                    id = username,
                    displayText = username,
                    startIndex = match.range.first,
                    endIndex = match.range.last + 1
                )
            )
        }

        // Find room mentions
        roomMentionRegex.findAll(text).forEach { match ->
            val roomId = "#${match.groupValues[1]}:${match.groupValues[2]}"
            mentions.add(
                Mention(
                    type = MentionType.ROOM,
                    id = roomId,
                    displayText = roomId,
                    startIndex = match.range.first,
                    endIndex = match.range.last + 1
                )
            )
        }

        // Find everyone mentions
        everyoneRegex.findAll(text).forEach { match ->
            mentions.add(
                Mention(
                    type = MentionType.EVERYONE,
                    id = "everyone",
                    displayText = match.value,
                    startIndex = match.range.first,
                    endIndex = match.range.last + 1
                )
            )
        }

        // Find URLs
        urlRegex.findAll(text).forEach { match ->
            mentions.add(
                Mention(
                    type = MentionType.LINK,
                    id = match.value,
                    displayText = match.value,
                    startIndex = match.range.first,
                    endIndex = match.range.last + 1
                )
            )
        }

        return mentions.sortedBy { it.startIndex }
    }

    /**
     * Apply mention styling to text
     */
    fun applyMentionStyle(
        text: String,
        mentions: List<Mention>,
        mentionColor: Color = BrandPurple
    ): AnnotatedString {
        return buildAnnotatedString {
            append(text)

            mentions.forEach { mention ->
                addStyle(
                    style = SpanStyle(
                        color = mentionColor,
                        fontWeight = FontWeight.Bold,
                        textDecoration = if (mention.type == MentionType.LINK) {
                            TextDecoration.Underline
                        } else null
                    ),
                    start = mention.startIndex,
                    end = mention.endIndex
                )

                // Add string annotation for click handling
                addStringAnnotation(
                    tag = mention.type.name,
                    annotation = mention.id,
                    start = mention.startIndex,
                    end = mention.endIndex
                )
            }
        }
    }

    /**
     * Get mention at position (for click handling)
     */
    fun getMentionAt(
        layoutResult: TextLayoutResult,
        offset: androidx.compose.ui.geometry.Offset,
        mentions: List<Mention>
    ): Mention? {
        val position = layoutResult.getOffsetForPosition(offset)
        return mentions.find { position in it.startIndex until it.endIndex }
    }
}

/**
 * User mention chip for display
 */
@Composable
fun UserMentionChip(
    userId: String,
    displayName: String?,
    avatarUrl: String?,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = BrandPurple.copy(alpha = 0.1f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Avatar
            Box(
                modifier = Modifier
                    .size(16.dp)
                    .clip(CircleShape)
                    .background(BrandPurple),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = (displayName ?: userId).firstOrNull()?.uppercase() ?: "?",
                    style = MaterialTheme.typography.labelSmall,
                    color = Color.White,
                    fontWeight = FontWeight.Bold
                )
            }

            // Name
            Text(
                text = displayName ?: userId,
                style = MaterialTheme.typography.labelMedium,
                color = BrandPurple,
                fontWeight = FontWeight.Bold
            )
        }
    }
}

/**
 * Room mention chip for display
 */
@Composable
fun RoomMentionChip(
    roomId: String,
    roomName: String?,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.primaryContainer
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "#",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.primary,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = roomName ?: roomId,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.primary,
                fontWeight = FontWeight.Bold
            )
        }
    }
}

/**
 * User preview popup for mention hover/click
 */
@Composable
fun MentionPreviewPopup(
    userId: String,
    displayName: String?,
    avatarUrl: String?,
    status: String?,
    onDismiss: () -> Unit,
    onViewProfile: () -> Unit,
    onSendMessage: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        elevation = CardDefaults.cardElevation(defaultElevation = 4.dp)
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Avatar
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .clip(CircleShape)
                    .background(BrandPurple),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = (displayName ?: userId).firstOrNull()?.uppercase() ?: "?",
                    style = MaterialTheme.typography.titleMedium,
                    color = Color.White,
                    fontWeight = FontWeight.Bold
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Name
            Text(
                text = displayName ?: userId,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold
            )

            // Status
            status?.let {
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            Spacer(modifier = Modifier.height(12.dp))

            // Actions
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                OutlinedButton(onClick = onViewProfile) {
                    Text("View Profile")
                }
                Button(onClick = onSendMessage) {
                    Text("Message")
                }
            }
        }
    }
}
