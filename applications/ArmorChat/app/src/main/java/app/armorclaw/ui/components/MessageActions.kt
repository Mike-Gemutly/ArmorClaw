package app.armorclaw.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import app.armorclaw.data.repository.BridgeCapabilities
import app.armorclaw.data.repository.BridgeProtocol
import app.armorclaw.data.repository.Feature
import app.armorclaw.data.repository.Limitation

/**
 * Message Actions - Feature-Aware Action Bar
 *
 * Resolves: G-05 (Feature Suppression)
 *
 * Conditionally shows message actions based on bridge capabilities.
 * Hides reactions, edits, and other features when not supported.
 */

/**
 * Data class for message context
 */
data class MessageContext(
    val messageId: String,
    val roomId: String,
    val senderId: String,
    val isOwnMessage: Boolean,
    val isEdited: Boolean = false,
    val isRedacted: Boolean = false,
    val hasAttachments: Boolean = false,
    val bridgeCapabilities: BridgeCapabilities = BridgeCapabilities.NATIVE_MATRIX
)

/**
 * Available message actions
 */
enum class MessageAction(
    val displayName: String,
    val icon: ImageVector,
    val requiredFeature: Feature? = null
) {
    REPLY("Reply", Icons.Default.Reply, Feature.REPLIES),
    EDIT("Edit", Icons.Default.Edit, Feature.EDITS),
    REACT("React", Icons.Default.AddReaction, Feature.REACTIONS),
    DELETE("Delete", Icons.Default.Delete, Feature.DELETION),
    COPY("Copy", Icons.Default.ContentCopy, null),
    FORWARD("Forward", Icons.Default.Forward, null),
    QUOTE("Quote", Icons.Default.FormatQuote, null),
    PIN("Pin", Icons.Default.PushPin, null),
    REPORT("Report", Icons.Default.Report, null),
    INFO("Info", Icons.Default.Info, null)
}

/**
 * Message action bar that respects bridge capabilities
 */
@Composable
fun MessageActionBar(
    messageContext: MessageContext,
    onAction: (MessageAction) -> Unit,
    modifier: Modifier = Modifier,
    expanded: Boolean = false
) {
    val capabilities = messageContext.bridgeCapabilities

    // Determine which actions are available
    val availableActions = remember(capabilities, messageContext.isOwnMessage) {
        buildList {
            // Reply is available if supported
            if (isActionAvailable(MessageAction.REPLY, capabilities)) {
                add(MessageAction.REPLY)
            }

            // React if supported
            if (isActionAvailable(MessageAction.REACT, capabilities)) {
                add(MessageAction.REACT)
            }

            // Edit only for own messages and if supported
            if (messageContext.isOwnMessage && isActionAvailable(MessageAction.EDIT, capabilities)) {
                add(MessageAction.EDIT)
            }

            // Copy always available
            add(MessageAction.COPY)

            // Forward always available
            add(MessageAction.FORWARD)

            // Delete only for own messages and if supported
            if (messageContext.isOwnMessage && isActionAvailable(MessageAction.DELETE, capabilities)) {
                add(MessageAction.DELETE)
            }

            // Pin always available (server-side)
            add(MessageAction.PIN)

            // Report always available
            add(MessageAction.REPORT)

            // Info always available
            add(MessageAction.INFO)
        }
    }

    if (expanded) {
        ExpandedMessageActions(
            actions = availableActions,
            capabilities = capabilities,
            onAction = onAction,
            modifier = modifier
        )
    } else {
        CompactMessageActions(
            actions = availableActions.take(4), // Show only first 4 in compact mode
            capabilities = capabilities,
            onAction = onAction,
            onMoreClick = { onAction(MessageAction.INFO) }, // Opens expanded view
            modifier = modifier
        )
    }
}

@Composable
private fun CompactMessageActions(
    actions: List<MessageAction>,
    capabilities: BridgeCapabilities,
    onAction: (MessageAction) -> Unit,
    onMoreClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        actions.forEach { action ->
            ActionChip(
                action = action,
                capabilities = capabilities,
                onClick = { onAction(action) }
            )
        }

        if (actions.size < MessageAction.entries.size) {
            // More options button
            IconButton(onClick = onMoreClick) {
                Icon(
                    imageVector = Icons.Default.MoreHoriz,
                    contentDescription = "More options"
                )
            }
        }
    }
}

@Composable
private fun ExpandedMessageActions(
    actions: List<MessageAction>,
    capabilities: BridgeCapabilities,
    onAction: (MessageAction) -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier,
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        // Show in rows of 4
        actions.chunked(4).forEach { rowActions ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly
            ) {
                rowActions.forEach { action ->
                    ActionChip(
                        action = action,
                        capabilities = capabilities,
                        onClick = {
                            onAction(action)
                        }
                    )
                }
            }
        }

        // Show limitations warning if any
        if (capabilities.limitations.isNotEmpty()) {
            LimitationsWarning(
                limitations = capabilities.limitations,
                protocol = capabilities.protocol
            )
        }
    }
}

@Composable
private fun ActionChip(
    action: MessageAction,
    capabilities: BridgeCapabilities,
    onClick: () -> Unit
) {
    val isSupported = action.requiredFeature?.let { capabilities.supports(it) } ?: true

    TextButton(
        onClick = onClick,
        enabled = isSupported
    ) {
        Icon(
            imageVector = action.icon,
            contentDescription = action.displayName,
            modifier = Modifier.size(18.dp)
        )
        Spacer(modifier = Modifier.width(4.dp))
        Text(action.displayName)
    }
}

/**
 * Check if an action is available given bridge capabilities
 */
private fun isActionAvailable(action: MessageAction, capabilities: BridgeCapabilities): Boolean {
    // If no required feature, action is always available
    val feature = action.requiredFeature ?: return true

    // Check if feature is supported
    return capabilities.supports(feature)
}

/**
 * Warning card showing bridge limitations
 */
@Composable
fun LimitationsWarning(
    limitations: Set<Limitation>,
    protocol: BridgeProtocol,
    modifier: Modifier = Modifier
) {
    if (limitations.isEmpty()) return

    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Info,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.width(8.dp))
            Column {
                Text(
                    text = "${protocol.displayName} Bridge Limitations",
                    style = MaterialTheme.typography.labelMedium
                )
                Text(
                    text = limitations.take(2).joinToString(", ") { it.displayName },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

/**
 * Reaction picker that respects bridge capabilities
 */
@Composable
fun CapabilityAwareReactionPicker(
    capabilities: BridgeCapabilities,
    onReactionSelected: (String) -> Unit,
    onCustomReaction: () -> Unit,
    modifier: Modifier = Modifier,
    showCustomOption: Boolean = true
) {
    // Check if reactions are supported
    if (!capabilities.supports(Feature.REACTIONS)) {
        // Show disabled state
        Card(
            modifier = modifier,
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
            )
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Block,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.outline
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = "Reactions not supported on ${capabilities.protocol.displayName}",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.outline
                )
            }
        }
        return
    }

    // Check for emoji limitations
    val hasEmojiLimitation = capabilities.hasLimitation(Limitation.NO_UNICODE_EMOJI)

    Column(modifier = modifier) {
        if (hasEmojiLimitation) {
            // Show text-based emoticons instead
            TextEmoticonPicker(
                onEmoticonSelected = onReactionSelected,
                modifier = Modifier.fillMaxWidth()
            )
        } else {
            // Standard emoji picker
            QuickEmojiPicker(
                onEmojiSelected = onReactionSelected,
                modifier = Modifier.fillMaxWidth()
            )
        }

        if (showCustomOption && !hasEmojiLimitation) {
            TextButton(
                onClick = onCustomReaction,
                modifier = Modifier.fillMaxWidth()
            ) {
                Icon(Icons.Default.Add, contentDescription = null)
                Spacer(modifier = Modifier.width(4.dp))
                Text("Custom Emoji")
            }
        }
    }
}

/**
 * Quick emoji picker with common reactions
 */
@Composable
private fun QuickEmojiPicker(
    onEmojiSelected: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    val quickEmojis = listOf(
        "\uD83D\uDC4D", // 👍
        "\uD83D\uDC4E", // 👎
        "\uD83D\uDE02", // 😂
        "\u2764\uFE0F", // ❤️
        "\uD83D\uDE00", // 😀
        "\uD83D\uDE0A", // 😊
        "\uD83D\uDE1F", // 😟
        "\uD83D\uDE2E", // 😮
        "\uD83C\uDF89", // 🎉
        "\uD83D\uDC4F"  // 👏
    )

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.SpaceEvenly
    ) {
        quickEmojis.forEach { emoji ->
            TextButton(
                onClick = { onEmojiSelected(emoji) },
                modifier = Modifier.size(48.dp)
            ) {
                Text(
                    text = emoji,
                    style = MaterialTheme.typography.headlineSmall
                )
            }
        }
    }
}

/**
 * Text emoticon picker for bridges that don't support unicode emoji
 */
@Composable
private fun TextEmoticonPicker(
    onEmoticonSelected: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    val emoticons = listOf(
        ":)" to "☺️",
        ":(" to "☹️",
        ":D" to "😀",
        ";)" to "😉",
        ":P" to "😛",
        ":O" to "😮",
        "<3" to "❤️",
        ":|" to "😐",
        ":/" to "😕",
        ":*" to "😘"
    )

    Column(modifier = modifier) {
        Text(
            text = "Text Emoticons (Emoji not supported)",
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.outline,
            modifier = Modifier.padding(horizontal = 8.dp)
        )

        Spacer(modifier = Modifier.height(8.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            emoticons.take(6).forEach { (text, _) ->
                TextButton(
                    onClick = { onEmoticonSelected(text) },
                    modifier = Modifier.size(48.dp)
                ) {
                    Text(
                        text = text,
                        style = MaterialTheme.typography.bodyLarge
                    )
                }
            }
        }
    }
}

/**
 * Message input with capability-aware features
 */
@Composable
fun CapabilityAwareMessageInput(
    capabilities: BridgeCapabilities,
    onSendMessage: (String) -> Unit,
    onSendEdit: (messageId: String, String) -> Unit,
    editingMessageId: String?,
    modifier: Modifier = Modifier
) {
    var text by remember { mutableStateOf("") }

    Column(modifier = modifier) {
        // Show edit indicator if editing
        if (editingMessageId != null && capabilities.supports(Feature.EDITS)) {
            EditIndicator(
                originalMessage = text,
                onCancel = { /* Cancel edit */ }
            )
        }

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Attachment button (if supported)
            if (capabilities.supports(Feature.FILES)) {
                IconButton(onClick = { /* Open file picker */ }) {
                    Icon(Icons.Default.AttachFile, contentDescription = "Attach file")
                }
            }

            // Text input
            OutlinedTextField(
                value = text,
                onValueChange = { text = it },
                modifier = Modifier.weight(1f),
                placeholder = {
                    Text(
                        if (capabilities.supports(Feature.MARKDOWN))
                            "Message (Markdown supported)"
                        else
                            "Message"
                    )
                },
                maxLines = 4
            )

            // Send button
            IconButton(
                onClick = {
                    if (editingMessageId != null) {
                        onSendEdit(editingMessageId, text)
                    } else {
                        onSendMessage(text)
                    }
                    text = ""
                },
                enabled = text.isNotBlank()
            ) {
                Icon(
                    imageVector = if (editingMessageId != null) Icons.Default.Check else Icons.Default.Send,
                    contentDescription = if (editingMessageId != null) "Save edit" else "Send"
                )
            }
        }

        // Show unsupported features notice
        if (!capabilities.supports(Feature.MARKDOWN)) {
            Text(
                text = "Plain text only - ${capabilities.protocol.displayName} doesn't support formatting",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.outline,
                modifier = Modifier.padding(top = 4.dp)
            )
        }
    }
}

@Composable
private fun EditIndicator(
    originalMessage: String,
    onCancel: () -> Unit
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        color = MaterialTheme.colorScheme.primaryContainer
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Icon(
                    imageVector = Icons.Default.Edit,
                    contentDescription = null,
                    modifier = Modifier.size(16.dp),
                    tint = MaterialTheme.colorScheme.onPrimaryContainer
                )
                Text(
                    text = "Editing message",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onPrimaryContainer
                )
            }

            TextButton(onClick = onCancel) {
                Text("Cancel")
            }
        }
    }
}
