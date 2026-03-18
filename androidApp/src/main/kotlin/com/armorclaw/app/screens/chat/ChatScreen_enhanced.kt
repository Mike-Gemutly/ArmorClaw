package com.armorclaw.app.screens.chat

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.Shield
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.TextButton
import androidx.compose.ui.graphics.Color
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.material3.Scaffold
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.OutlinedTextField
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import com.armorclaw.app.screens.chat.components.ChatSearchBar
import com.armorclaw.app.screens.chat.components.EncryptionStatus
import com.armorclaw.shared.ui.components.UnifiedMessageList
import com.armorclaw.shared.domain.model.UnifiedMessage
import com.armorclaw.app.screens.chat.components.EncryptionStatusIndicator
import com.armorclaw.app.screens.chat.components.MessageList
import com.armorclaw.app.screens.chat.components.MessageListState
import com.armorclaw.app.screens.chat.components.ReplyPreviewBar
import com.armorclaw.app.screens.chat.components.TypingIndicatorComponent
import com.armorclaw.app.screens.chat.components.TypingIndicator
import com.armorclaw.app.viewmodels.ChatViewModel
import com.armorclaw.shared.ui.base.UiEvent
import com.armorclaw.shared.ui.components.WorkflowProgressBanner
import com.armorclaw.shared.ui.components.AgentThinkingIndicator
import com.armorclaw.shared.ui.components.AgentTaskStatusBanner
import com.armorclaw.shared.ui.components.BlindFillCard
import com.armorclaw.shared.ui.components.CommandBar
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandRed
import com.armorclaw.shared.ui.theme.Primary
import com.armorclaw.shared.ui.components.SplitViewLayout
import com.armorclaw.shared.ui.components.ActivityLog
import com.armorclaw.shared.ui.components.AgentEvent
import com.armorclaw.shared.ui.components.AgentStepStatus
import com.armorclaw.shared.data.store.AgentTaskState

/**
 * Enhanced Chat Screen

 * ## Architecture (Post-Migration)
 * ```
 * ChatScreen
 *      ├── ChatTopBar (title, encryption, call buttons)
 *      ├── WorkflowProgressBanner (if workflow active)
 *      ├── AgentThinkingIndicator (if agent thinking)
 *      ├── ChatSearchBar (when search active)
 *      ├── MessageList
 *      ├── TypingIndicatorComponent
 *      ├── ReplyPreviewBar
 *      └── MessageInputBar
 * ```

 * ## Control Plane Integration
 * - activeWorkflow: Shows WorkflowProgressBanner when workflow is running
 * - agentThinking: Shows AgentThinkingIndicator when agent is processing
 * - isAgentRoom: Enables agent-specific UI features
 */

/**
 * Convert AgentTaskState to AgentEvent for ActivityLog display
 */
private fun mapAgentTaskToEvent(task: AgentTaskState): AgentEvent {
    return when (task) {
        is AgentTaskState.Running -> AgentEvent(
            id = task.taskId,
            stepName = "${task.agentName}: ${task.taskType}",
            status = when (task.progress) {
                0f -> AgentStepStatus.PENDING
                1f -> AgentStepStatus.COMPLETED
                else -> AgentStepStatus.RUNNING
            },
            timestamp = task.startTime,
            output = "Progress: ${task.progress * 100}%"
        )
    }
}
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreenEnhanced(
    roomId: String,
    viewModel: ChatViewModel,
    onNavigateBack: () -> Unit,
    onNavigateToRoomDetails: ((roomId: String) -> Unit)? = null,
    onNavigateToVoiceCall: ((roomId: String) -> Unit)? = null,
    onNavigateToVideoCall: ((roomId: String) -> Unit)? = null,
    onNavigateToThread: ((roomId: String, rootMessageId: String) -> Unit)? = null,
    onNavigateToImage: ((imageId: String) -> Unit)? = null,
    onNavigateToFile: ((fileId: String) -> Unit)? = null,
    onNavigateToUserProfile: ((userId: String) -> Unit)? = null
) {
    val uiState by viewModel.uiState.collectAsState()
    val unifiedMessages by viewModel.unifiedMessages.collectAsState()
    val typingIndicator by viewModel.typingIndicator.collectAsState()
    val encryptionStatus by viewModel.encryptionStatus.collectAsState()
    val isSearchActive by viewModel.isSearchActive.collectAsState()
    val searchQuery by viewModel.searchQuery.collectAsState()
    val replyTo by viewModel.replyTo.collectAsState()
    val hasMore by viewModel.hasMore.collectAsState()
    val currentUser by viewModel.currentUser.collectAsState()
    
    // Control Plane state (NEW)
    val activeWorkflow by viewModel.activeWorkflow.collectAsState()
    val agentThinking by viewModel.agentThinking.collectAsState()
    val isAgentRoom by viewModel.isAgentRoom.collectAsState()
    val agentStatus by viewModel.agentStatus.collectAsState()
    val pendingPiiRequest by viewModel.pendingPiiRequest.collectAsState()
    
    // Agent tasks for ActivityLog
    val agentTasks by viewModel.agentTasks.collectAsState()
    
    // Bridge verification (Fix 3)
    val isBridgeVerified by viewModel.isBridgeVerified.collectAsState()
    var showBridgeWarningDialog by remember { mutableStateOf(false) }

    // Show first-time unverified bridge dialog
    if (!isBridgeVerified && !showBridgeWarningDialog) {
        showBridgeWarningDialog = true
    }
    if (showBridgeWarningDialog && !isBridgeVerified) {
        AlertDialog(
            onDismissRequest = { showBridgeWarningDialog = false },
            icon = {
                Icon(
                    imageVector = Icons.Default.Shield,
                    contentDescription = null,
                    tint = Color(0xFFF57F17),
                    modifier = Modifier.size(32.dp)
                )
            },
            title = {
                Text(
                    text = "Unverified Bridge",
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Text(
                    text = "This room is bridged to an external platform whose identity " +
                           "has not been verified. Messages may be relayed through an " +
                           "untrusted server. Verify the bridge in Room Settings " +
                           "before sharing sensitive information."
                )
            },
            confirmButton = {
                Button(onClick = { showBridgeWarningDialog = false }) {
                    Text("I Understand")
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    showBridgeWarningDialog = false
                    onNavigateBack()
                }) {
                    Text("Leave Room")
                }
            }
        )
    }

    var inputMessage by remember { mutableStateOf("") }
    var isVoiceInputActive by remember { mutableStateOf(false) }
    var isRefreshing by remember { mutableStateOf(false) }
    
ArmorClawTheme {
        Scaffold(
            topBar = {
                ChatTopBar(
                    roomId = roomId,
                    encryptionStatus = encryptionStatus,
                    isBridgeVerified = isBridgeVerified,
                    isSearchActive = isSearchActive,
                    onToggleSearch = { viewModel.toggleSearch() },
                    onNavigateBack = onNavigateBack,
                    onTitleClick = {
                        onNavigateToRoomDetails?.invoke(roomId)
                    },
                    onVoiceCallClick = onNavigateToVoiceCall?.let { { it(roomId) } },
                    onVideoCallClick = onNavigateToVideoCall?.let { { it(roomId) } }
                )
            },
            snackbarHost = {
                // Connect snackbar host to ChatViewModel events
                val snackbarHostState = androidx.compose.material3.SnackbarHostState()
                androidx.compose.material3.SnackbarHost(
                    hostState = snackbarHostState,
                    modifier = Modifier
                )
                
                // Observe events and show snackbar when needed
                val event by viewModel.events.collectAsState()
                LaunchedEffect(event) {
                    event?.let { currentEvent ->
                        when (currentEvent) {
                            is UiEvent.ShowSnackbar -> {
                                snackbarHostState.showSnackbar(currentEvent.message)
                                viewModel.clearEvent()
                            }
                            else -> {}
                        }
                    }
                }
            }
        ) { paddingValues ->
            SplitViewLayout(
                chatContent = {
                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(paddingValues)
                    ) {
                        // Workflow Progress Banner (NEW)
                        if (activeWorkflow != null) {
                            WorkflowProgressBanner(
                                workflowState = activeWorkflow!!,
                                modifier = Modifier.fillMaxWidth()
                            )
                        }

                        // Agent Status Banner (Phase 2)
                        if (agentStatus != null && isAgentRoom) {
                            AgentTaskStatusBanner(
                                status = agentStatus!!.status,
                                metadata = agentStatus!!.metadata,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 12.dp, vertical = 4.dp),
                                onDismiss = {
                                    // Dismiss intervention state - this could trigger a cancel
                                    viewModel.sendMessage("!cancel")
                                }
                            )
                        }

                        // PII Access Request Card (Phase 2 - BlindFill)
                        if (pendingPiiRequest != null && isAgentRoom) {
                            BlindFillCard(
                                request = pendingPiiRequest!!,
                                onApprove = { approvedFields ->
                                    viewModel.approvePiiRequest(approvedFields)
                                },
                                onDeny = {
                                    viewModel.denyPiiRequest()
                                },
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 12.dp, vertical = 8.dp)
                            )
                        }

                        // Agent Thinking Indicator (NEW)
                        if (agentThinking != null) {
                            AgentThinkingIndicator(
                                thinkingAgents = listOf(agentThinking!!),
                                expanded = true,
                                showNames = true,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 16.dp, vertical = 4.dp)
                            )
                        }
                        
                        // Search bar (when active)
                        if (isSearchActive) {
                            ChatSearchBar(
                                query = searchQuery,
                                onQueryChange = { viewModel.onSearchQueryChange(it) },
                                onClose = { viewModel.toggleSearch() }
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                        }
                        
                        // Messages list
                        androidx.compose.foundation.layout.Box(
                            modifier = Modifier
                                .weight(1f)
                                .fillMaxWidth()
                        ) {
                            UnifiedMessageList(
                                messages = unifiedMessages,
                                currentUserId = currentUser?.id ?: "",
                                isLoading = false,
                                hasMore = hasMore,
                                onLoadMore = { viewModel.loadMoreMessages() },
                                onReply = { viewModel.replyToMessage(it) },
                                onReaction = { message, emoji -> 
                                    viewModel.toggleReaction(message, emoji)
                                }
                            )
                        }
                        
                        // Typing indicator
                        if (typingIndicator.isTyping) {
                            TypingIndicatorComponent(
                                indicator = typingIndicator,
                                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)
                            )
                        }
                        
                        // Reply preview
                        if (replyTo != null) {
                            ReplyPreviewBar(
                                replyTo = replyTo!!,
                                onCancelReply = { viewModel.cancelReply() },
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                            )
                        }
                        
// Input bar
CommandBar(
    value = TextFieldValue(inputMessage),
    onValueChange = { inputMessage = it.text },
    onSend = { 
        viewModel.sendMessage(inputMessage)
        inputMessage = ""
    },
    placeholder = "Type a message..."
)
                    }
                },
                activityLogContent = {
                    // Activity Log Content
                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(paddingValues)
                    ) {
                        // Activity Log Header
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 8.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = "Agent Activity",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.SemiBold
                            )
                            Text(
                                text = "${agentTasks.size} tasks",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                        
                        // Activity Log Content
                        if (agentTasks.isNotEmpty()) {
                            ActivityLog(
                                events = agentTasks.map { mapAgentTaskToEvent(it) },
                                modifier = Modifier.fillMaxSize(),
                                autoScroll = true
                            )
                        } else {
                            // Empty state
                            Box(
                                modifier = Modifier
                                    .fillMaxSize()
                                    .padding(16.dp),
                                contentAlignment = Alignment.Center
                            ) {
                                Text(
                                    text = "No agent activity",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                    }
                },
                isActivityLogVisible = true
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ChatTopBar(
    roomId: String,
    encryptionStatus: EncryptionStatus,
    isBridgeVerified: Boolean = true,
    isSearchActive: Boolean,
    onToggleSearch: () -> Unit,
    onNavigateBack: () -> Unit,
    onTitleClick: () -> Unit = {},
    onVoiceCallClick: (() -> Unit)? = null,
    onVideoCallClick: (() -> Unit)? = null
) {
    TopAppBar(
        title = {
            Column(
                horizontalAlignment = Alignment.Start,
                modifier = Modifier.clickable(onClick = onTitleClick)
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "ArmorClaw Agent",
                        style = MaterialTheme.typography.titleMedium
                    )

                    // Online status indicator
                    Surface(
                        shape = CircleShape,
                        color = BrandGreen,
                        modifier = Modifier.size(8.dp)
                    ) {}

                    // Encryption status
                    if (!isSearchActive) {
                        EncryptionStatusIndicator(
                            status = encryptionStatus,
                            showText = false
                        )
                    }

                    // Bridge verification indicator (Fix 3)
                    if (!isBridgeVerified && !isSearchActive) {
                        Icon(
                            imageVector = Icons.Default.Shield,
                            contentDescription = "Unverified bridge",
                            tint = Color(0xFFF57F17),
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            }
        },
        navigationIcon = {
            IconButton(onClick = onNavigateBack) {
                Icon(Icons.Default.ArrowBack, contentDescription = "Back")
            }
        },
        actions = {
            // Voice call button
            if (onVoiceCallClick != null) {
                IconButton(onClick = onVoiceCallClick) {
                    Icon(Icons.Default.Call, contentDescription = "Voice call")
                }
            }
            // Video call button
            if (onVideoCallClick != null) {
                IconButton(onClick = onVideoCallClick) {
                    Icon(Icons.Default.Videocam, contentDescription = "Video call")
                }
            }
            IconButton(onClick = onToggleSearch) {
                Icon(Icons.Default.Search, contentDescription = "Search")
            }
            IconButton(onClick = { /* More options */ }) {
                Icon(Icons.Default.MoreVert, contentDescription = "More")
            }
        },
        colors = TopAppBarDefaults.topAppBarColors(
            containerColor = Primary
        )
    )
}

@Composable
private fun MessageInputBar(
    message: String,
    onMessageChange: (String) -> Unit,
    onSend: () -> Unit,
    isVoiceInputActive: Boolean,
    onToggleVoiceInput: (Boolean) -> Unit,
    onAttachClick: () -> Unit
) {
    Surface(
        tonalElevation = 4.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Attach button
            IconButton(onClick = onAttachClick) {
                Icon(Icons.Default.AttachFile, contentDescription = "Attach")
            }
            
            // Text input
            OutlinedTextField(
                value = message,
                onValueChange = onMessageChange,
                modifier = Modifier.weight(1f),
                placeholder = { Text("Type a message...") },
                maxLines = 4,
                shape = CircleShape,
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Text,
                    imeAction = ImeAction.Send
                ),
                keyboardActions = KeyboardActions(
                    onSend = {
                        if (message.isNotBlank()) {
                            onSend()
                        }
                    }
                )
            )
            
            // Voice/ Mic button
            IconButton(onClick = { onToggleVoiceInput(!isVoiceInputActive) }) {
                Icon(
                    Icons.Default.Mic,
                    contentDescription = if (isVoiceInputActive) "Stop recording" else "Voice input",
                    tint = if (isVoiceInputActive) BrandRed else BrandPurple
                )
            }
            
            // Send button
            IconButton(
                onClick = onSend,
                enabled = message.isNotBlank()
            ) {
                Icon(
                    Icons.Default.Send,
                    contentDescription = "Send",
                    tint = if (message.isNotBlank())
                        BrandPurple
                    else
                        MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
                )
            }
        }
    }
}
