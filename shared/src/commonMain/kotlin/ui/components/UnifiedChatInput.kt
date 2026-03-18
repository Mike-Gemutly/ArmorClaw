package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.AgentType
import com.armorclaw.shared.domain.model.MessageSender

/**
 * Unified Chat Input
 *
 * Single input component for both regular messages and agent commands.
 * Automatically detects command mode (starts with !) and adjusts UI.
 *
 * ## Modes
 * - Regular: Normal message input
 * - Command: Agent command input (starts with !)
 * - Multi-line: Expanded for longer input
 *
 * ## Usage
 * ```kotlin
 * var inputText by remember { mutableStateOf(TextFieldValue()) }
 *
 * UnifiedChatInput(
 *     value = inputText,
 *     onValueChange = { inputText = it },
 *     onSend = { content ->
 *         viewModel.sendMessage(content)
 *         inputText = TextFieldValue()
 *     },
 *     activeAgent = agentState,
 *     isAgentRoom = true
 * )
 * ```
 */
@Composable
fun UnifiedChatInput(
    value: TextFieldValue,
    onValueChange: (TextFieldValue) -> Unit,
    onSend: (String) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    isAgentRoom: Boolean = false,
    activeAgent: MessageSender.AgentSender? = null,
    replyToPreview: String? = null,
    onReplyCancel: () -> Unit = {},
    onAttachClick: () -> Unit = {},
    onVoiceClick: () -> Unit = {},
    showVoiceButton: Boolean = true,
    placeholder: String = "Type a message..."
) {
    val isCommandMode = value.text.startsWith("!")
    val isMultiLine = value.text.lines().size > 1
    val hasContent = value.text.isNotBlank()

    Column(modifier = modifier) {
        // Reply preview
        AnimatedVisibility(
            visible = replyToPreview != null,
            enter = fadeIn() + expandVertically(),
            exit = fadeOut() + shrinkVertically()
        ) {
            ReplyInputPreview(
                preview = replyToPreview ?: "",
                onCancel = onReplyCancel
            )
        }

        // Command mode indicator
        AnimatedVisibility(
            visible = isCommandMode && isAgentRoom,
            enter = fadeIn() + expandVertically(),
            exit = fadeOut() + shrinkVertically()
        ) {
            CommandModeIndicator(
                agentName = activeAgent?.displayName,
                agentType = activeAgent?.agentType
            )
        }

        // Main input row
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.Bottom
        ) {
            // Attachment button
            IconButton(
                onClick = onAttachClick,
                enabled = enabled
            ) {
                Icon(
                    imageVector = Icons.Outlined.AttachFile,
                    contentDescription = "Attach file",
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            // Input field
            Surface(
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = 4.dp),
                shape = RoundedCornerShape(24.dp),
                color = if (isCommandMode && isAgentRoom) {
                    MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.3f)
                } else {
                    MaterialTheme.colorScheme.surfaceVariant
                }
            ) {
                BasicTextField(
                    value = value,
                    onValueChange = onValueChange,
                    enabled = enabled,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 12.dp),
                    textStyle = MaterialTheme.typography.bodyLarge.copy(
                        color = if (isCommandMode) {
                            MaterialTheme.colorScheme.onTertiaryContainer
                        } else {
                            MaterialTheme.colorScheme.onSurface
                        },
                        fontFamily = if (isCommandMode) {
                            FontFamily.Monospace
                        } else {
                            FontFamily.Default
                        }
                    ),
                    cursorBrush = SolidColor(MaterialTheme.colorScheme.primary),
                    keyboardOptions = KeyboardOptions(
                        capitalization = KeyboardCapitalization.Sentences,
                        imeAction = if (isMultiLine) ImeAction.Default else ImeAction.Send
                    ),
                    keyboardActions = KeyboardActions(
                        onSend = {
                            if (hasContent) {
                                onSend(value.text)
                            }
                        }
                    ),
                    maxLines = if (isMultiLine) 6 else 4,
                    decorationBox = { innerTextField ->
                        Box(modifier = Modifier.fillMaxWidth()) {
                            if (value.text.isEmpty()) {
                                Text(
                                    text = if (isAgentRoom && !isCommandMode) {
                                        "Message or type ! for commands..."
                                    } else if (isCommandMode) {
                                        "Enter command..."
                                    } else {
                                        placeholder
                                    },
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(
                                        alpha = 0.6f
                                    ),
                                    fontFamily = if (isCommandMode) {
                                        FontFamily.Monospace
                                    } else {
                                        FontFamily.Default
                                    }
                                )
                            }
                            innerTextField()
                        }
                    }
                )
            }

            // Send or Voice button
            if (hasContent) {
                FilledIconButton(
                    onClick = { onSend(value.text) },
                    enabled = enabled,
                    colors = IconButtonDefaults.filledIconButtonColors(
                        containerColor = if (isCommandMode) {
                            MaterialTheme.colorScheme.tertiary
                        } else {
                            MaterialTheme.colorScheme.primary
                        }
                    )
                ) {
                    Icon(
                        imageVector = if (isCommandMode) {
                            Icons.Default.Terminal
                        } else {
                            Icons.Default.Send
                        },
                        contentDescription = if (isCommandMode) "Execute" else "Send",
                        modifier = Modifier.size(20.dp)
                    )
                }
            } else if (showVoiceButton) {
                FilledTonalIconButton(
                    onClick = onVoiceClick,
                    enabled = enabled
                ) {
                    Icon(
                        imageVector = Icons.Default.Mic,
                        contentDescription = "Voice input",
                        modifier = Modifier.size(20.dp)
                    )
                }
            }
        }

        // Quick actions for agent rooms
        AnimatedVisibility(
            visible = isAgentRoom && !hasContent && activeAgent != null,
            enter = fadeIn() + expandVertically(),
            exit = fadeOut() + shrinkVertically()
        ) {
            AgentQuickActions(
                agentType = activeAgent!!.agentType,
                onActionClick = { action ->
                    onValueChange(TextFieldValue(action))
                }
            )
        }
    }
}

/**
 * Reply preview above input
 */
@Composable
private fun ReplyInputPreview(
    preview: String,
    onCancel: () -> Unit
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 8.dp, vertical = 4.dp),
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier.padding(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Reply,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(16.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = preview,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 1,
                modifier = Modifier.weight(1f)
            )
            IconButton(
                onClick = onCancel,
                modifier = Modifier.size(24.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "Cancel reply",
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.size(16.dp)
                )
            }
        }
    }
}

/**
 * Command mode indicator
 */
@Composable
private fun CommandModeIndicator(
    agentName: String?,
    agentType: AgentType?
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 8.dp),
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.3f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Terminal,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.tertiary,
                modifier = Modifier.size(14.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "Command mode",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.tertiary,
                fontWeight = androidx.compose.ui.text.font.FontWeight.Medium
            )
            if (agentName != null) {
                Text(
                    text = " → $agentName",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onTertiaryContainer.copy(alpha = 0.7f)
                )
            }
            Spacer(modifier = Modifier.weight(1f))
            Text(
                text = "ESC to exit",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.5f)
            )
        }
    }
}

/**
 * Quick action suggestions for agent rooms
 */
@Composable
private fun AgentQuickActions(
    agentType: AgentType,
    onActionClick: (String) -> Unit
) {
    val actions = getQuickActionsForAgent(agentType)

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 8.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        actions.forEach { action ->
            SuggestionChip(
                onClick = { onActionClick(action.command) },
                label = { Text(action.label) },
                icon = {
                    Icon(
                        imageVector = action.icon,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp)
                    )
                }
            )
        }
    }
}

/**
 * Quick action data
 */
private data class QuickAction(
    val label: String,
    val command: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector
)

/**
 * Get quick actions based on agent type
 */
private fun getQuickActionsForAgent(type: AgentType): List<QuickAction> {
    return when (type) {
        AgentType.GENERAL -> listOf(
            QuickAction("Help", "!help", Icons.Outlined.Help),
            QuickAction("Analyze", "!analyze", Icons.Outlined.Analytics),
            QuickAction("Summarize", "!summarize", Icons.Outlined.Summarize)
        )
        AgentType.ANALYSIS -> listOf(
            QuickAction("Analyze", "!analyze", Icons.Outlined.Analytics),
            QuickAction("Compare", "!compare", Icons.Outlined.Compare),
            QuickAction("Report", "!report", Icons.Outlined.Assessment)
        )
        AgentType.CODE_REVIEW -> listOf(
            QuickAction("Review", "!review", Icons.Outlined.Code),
            QuickAction("Fix", "!fix", Icons.Outlined.Build),
            QuickAction("Explain", "!explain", Icons.Outlined.Lightbulb)
        )
        AgentType.RESEARCH -> listOf(
            QuickAction("Search", "!search", Icons.Outlined.Search),
            QuickAction("Find", "!find", Icons.Outlined.TravelExplore),
            QuickAction("Sources", "!sources", Icons.Outlined.Source)
        )
        AgentType.WRITING -> listOf(
            QuickAction("Write", "!write", Icons.Outlined.Edit),
            QuickAction("Edit", "!edit", Icons.Outlined.EditNote),
            QuickAction("Improve", "!improve", Icons.Outlined.AutoFixHigh)
        )
        AgentType.TRANSLATION -> listOf(
            QuickAction("Translate", "!translate", Icons.Outlined.Translate),
            QuickAction("Detect", "!detect", Icons.Outlined.Language)
        )
        AgentType.SCHEDULING -> listOf(
            QuickAction("Schedule", "!schedule", Icons.Outlined.Event),
            QuickAction("Remind", "!remind", Icons.Outlined.Alarm),
            QuickAction("Calendar", "!calendar", Icons.Outlined.CalendarMonth)
        )
        AgentType.WORKFLOW -> listOf(
            QuickAction("Start", "!start", Icons.Outlined.PlayArrow),
            QuickAction("Status", "!status", Icons.Outlined.Info),
            QuickAction("List", "!list", Icons.Outlined.List)
        )
        AgentType.PLATFORM_BRIDGE -> listOf(
            QuickAction("Connect", "!connect", Icons.Outlined.Link),
            QuickAction("Sync", "!sync", Icons.Outlined.Sync),
            QuickAction("Status", "!status", Icons.Outlined.Info)
        )
    }
}
