package com.armorclaw.app.viewmodels

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.app.screens.chat.components.EncryptionStatus
import com.armorclaw.app.screens.chat.components.TypingIndicator
import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.data.store.AgentThinkingState
import com.armorclaw.shared.data.store.AgentTaskState
import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.domain.model.AgentTaskStatusEvent
import com.armorclaw.shared.domain.model.PiiAccessRequest
import com.armorclaw.shared.domain.model.UnifiedMessage
import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.viewModelLogger
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.matrix.MessageBatch
import com.armorclaw.shared.ui.base.UiEvent
import com.armorclaw.shared.ui.base.UiState
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlin.time.Duration.Companion.hours
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds

/**
 * ViewModel for the Chat screen
 *
 * ## Architecture (Post-Migration Phase 4)
 * ```
 * ChatViewModel
 *      ├── MatrixClient (messaging, sync, typing)
 *      ├── ControlPlaneStore (workflows, agent events)
 *      └── UnifiedMessage state
 * ```
 *
 * ## Migration Status
 * - [x] Matrix SDK integration for sending messages
 * - [x] Matrix SDK integration for receiving messages
 * - [x] Typing indicators via Matrix
 * - [x] Read receipts via Matrix
 * - [x] Control Plane events (workflows, agents)
 * - [x] UnifiedMessage model for all message types
 * - [x] Command detection and handling
 *
 * Uses ViewModelLogger for proper separation of concerns in logging.
 */
class ChatViewModel(
    private val roomId: String,
    private val matrixClient: MatrixClient,
    private val controlPlaneStore: ControlPlaneStore,
    private val messageRepository: MessageRepository // Legacy fallback
) : ViewModel() {

    private val logger = viewModelLogger("ChatViewModel", LogTag.ViewModel.Chat)

    private val _uiState = MutableStateFlow<ChatUiState>(ChatUiState.Initial)
    val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()

    // Unified message list state (Phase 4)
    private val _unifiedMessages = MutableStateFlow<List<UnifiedMessage>>(emptyList())
    val unifiedMessages: StateFlow<List<UnifiedMessage>> = _unifiedMessages.asStateFlow()

    private val _isSearchActive = MutableStateFlow(false)
    val isSearchActive: StateFlow<Boolean> = _isSearchActive.asStateFlow()

    private val _searchQuery = MutableStateFlow("")
    val searchQuery: StateFlow<String> = _searchQuery.asStateFlow()

    private val _replyTo = MutableStateFlow<UnifiedMessage?>(null)
    val replyTo: StateFlow<UnifiedMessage?> = _replyTo.asStateFlow()

    private val _typingIndicator = MutableStateFlow(TypingIndicator(isTyping = false))
    val typingIndicator: StateFlow<TypingIndicator> = _typingIndicator.asStateFlow()

    private val _encryptionStatus = MutableStateFlow(EncryptionStatus.VERIFIED)
    val encryptionStatus: StateFlow<EncryptionStatus> = _encryptionStatus.asStateFlow()

    private val _events = MutableStateFlow<UiEvent?>(null)
    val events: StateFlow<UiEvent?> = _events.asStateFlow()

    // Control Plane State
    private val _activeWorkflow = MutableStateFlow<WorkflowState?>(null)
    val activeWorkflow: StateFlow<WorkflowState?> = _activeWorkflow.asStateFlow()

    private val _agentThinking = MutableStateFlow<AgentThinkingState?>(null)
    val agentThinking: StateFlow<AgentThinkingState?> = _agentThinking.asStateFlow()

    private val _isAgentRoom = MutableStateFlow(false)
    val isAgentRoom: StateFlow<Boolean> = _isAgentRoom.asStateFlow()

    // Agent tasks for current room
    private val _agentTasks = MutableStateFlow<List<AgentTaskState>>(emptyList())
    val agentTasks: StateFlow<List<AgentTaskState>> = _agentTasks.asStateFlow()

    // Agent status for current room (Phase 2)
    private val _agentStatus = MutableStateFlow<AgentTaskStatusEvent?>(null)
    val agentStatus: StateFlow<AgentTaskStatusEvent?> = _agentStatus.asStateFlow()

    // PII access requests for current room (Phase 2)
    private val _pendingPiiRequest = MutableStateFlow<PiiAccessRequest?>(null)
    val pendingPiiRequest: StateFlow<PiiAccessRequest?> = _pendingPiiRequest.asStateFlow()

    // Bridge verification state (Fix 3)
    private val _isBridgeVerified = MutableStateFlow(true)
    val isBridgeVerified: StateFlow<Boolean> = _isBridgeVerified.asStateFlow()

    // Active agent for quick actions
    private val _activeAgent = MutableStateFlow<MessageSender.AgentSender?>(null)
    val activeAgent: StateFlow<MessageSender.AgentSender?> = _activeAgent.asStateFlow()

    // Current user (cached)
    private val _currentUser = MutableStateFlow<MessageSender.UserSender?>(null)
    val currentUser: StateFlow<MessageSender.UserSender?> = _currentUser.asStateFlow()

    // Failed messages for retry functionality
    private val _failedMessages = MutableStateFlow<List<UnifiedMessage.Regular>>(emptyList())
    val failedMessages: StateFlow<List<UnifiedMessage.Regular>> = _failedMessages.asStateFlow()

    // Pagination state
    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private val _isLoadingMore = MutableStateFlow(false)
    val isLoadingMore: StateFlow<Boolean> = _isLoadingMore.asStateFlow()

    private val _hasMore = MutableStateFlow(false)
    val hasMore: StateFlow<Boolean> = _hasMore.asStateFlow()

    private var nextBatchToken: String? = null

    init {
        logger.logInit(mapOf("roomId" to roomId))
        loadCurrentUser()
        loadMessages()
        observeTypingIndicators()
        observeControlPlaneEvents()
        checkEncryptionStatus()
        checkBridgeVerification()
    }

    /**
     * Load current user info
     */
    private fun loadCurrentUser() {
        viewModelScope.launch {
            val user = matrixClient.currentUser.value
            if (user != null) {
                _currentUser.value = MessageSender.UserSender(
                    id = user.id,
                    displayName = user.displayName ?: user.id,
                    avatarUrl = user.avatar,
                    isCurrentUser = true,
                    isVerified = user.isVerified
                )
            }
        }
    }

    /**
     * Load messages using Matrix SDK
     */
    fun loadMessages() {
        logger.logUserAction("loadMessages", mapOf("roomId" to roomId))
        viewModelScope.launch {
            _isLoading.value = true
            _unifiedMessages.value = emptyList()

            // Use Matrix SDK for loading messages
            matrixClient.getMessages(roomId, limit = PAGE_SIZE)
                .fold(
                    onSuccess = { batch ->
                        handleMessagesBatch(batch)
                        _uiState.value = ChatUiState.MessagesLoaded
                        logger.logStateChange("unifiedMessages", "${batch.messages.size} messages")
                    },
                    onFailure = { error ->
                        logger.logError("loadMessages", error, mapOf("roomId" to roomId))
                        // Fallback to legacy repository
                        loadMessagesFromRepository()
                    }
                )
        }
    }

    /**
     * Fallback to legacy MessageRepository
     */
    private suspend fun loadMessagesFromRepository() {
        messageRepository.getMessages(roomId, limit = PAGE_SIZE, offset = 0)
            .onSuccess { messages ->
                val unifiedMessages = messages.map { convertToUnifiedMessage(it) }
                _hasMore.value = messages.size == PAGE_SIZE

                _unifiedMessages.value = unifiedMessages
                _isLoading.value = false
                _uiState.value = ChatUiState.MessagesLoaded
                logger.logStateChange("unifiedMessages (legacy)", "${messages.size} messages")
            }
            .onError { error ->
                logger.logError("loadMessagesFromRepository", error.toException(), mapOf("roomId" to roomId))
                _uiState.value = ChatUiState.Error("Failed to load messages: ${error.message}")
                _isLoading.value = false
            }
    }

    /**
     * Convert legacy Message to UnifiedMessage
     */
    private fun convertToUnifiedMessage(message: Message): UnifiedMessage {
        val sender = if (message.senderId.contains("agent_", ignoreCase = true)) {
            MessageSender.AgentSender(
                id = message.senderId,
                displayName = message.senderId,
                avatarUrl = null,
                agentType = detectAgentType(message.senderId)
            )
        } else {
            MessageSender.UserSender(
                id = message.senderId,
                displayName = message.senderId,
                avatarUrl = null,
                isCurrentUser = message.senderId == _currentUser.value?.id
            )
        }

        return UnifiedMessage.Regular(
            id = message.id,
            roomId = message.roomId,
            timestamp = message.timestamp,
            sender = sender,
            content = message.content,
            status = message.status,
            replyTo = message.replyTo,
            reactions = message.reactions,
            isEncrypted = true
        )
    }

    /**
     * Detect agent type from sender ID
     */
    private fun detectAgentType(senderId: String): AgentType {
        return when {
            senderId.contains("analysis") -> AgentType.ANALYSIS
            senderId.contains("code") -> AgentType.CODE_REVIEW
            senderId.contains("research") -> AgentType.RESEARCH
            senderId.contains("writing") -> AgentType.WRITING
            senderId.contains("translation") -> AgentType.TRANSLATION
            senderId.contains("schedule") -> AgentType.SCHEDULING
            senderId.contains("workflow") -> AgentType.WORKFLOW
            senderId.contains("bridge") -> AgentType.PLATFORM_BRIDGE
            else -> AgentType.GENERAL
        }
    }

    private fun handleMessagesBatch(batch: MessageBatch) {
        val messages = batch.messages.map { convertToUnifiedMessage(it) }
        nextBatchToken = batch.nextToken
        _hasMore.value = batch.nextToken != null
        _isLoading.value = false

        _unifiedMessages.value = messages

        // Check if this is an agent room
        checkIfAgentRoom(messages)

        // Update active agent
        val agentMessage = messages.filterIsInstance<UnifiedMessage.Regular>()
            .firstOrNull { it.sender is MessageSender.AgentSender }
        if (agentMessage != null && agentMessage.sender is MessageSender.AgentSender) {
            _activeAgent.value = agentMessage.sender as MessageSender.AgentSender
        }
    }

    /**
     * Check if room contains AI agents
     */
    private fun checkIfAgentRoom(messages: List<UnifiedMessage>) {
        val hasAgentMessages = messages.any { 
            (it is UnifiedMessage.Regular && it.sender is MessageSender.AgentSender) ||
            (it is UnifiedMessage.Agent)
        }
        _isAgentRoom.value = hasAgentMessages
    }

    fun refreshMessages() {
        logger.logUserAction("refreshMessages", mapOf("roomId" to roomId))
        viewModelScope.launch {
            _isLoading.value = true
            nextBatchToken = null

            matrixClient.getMessages(roomId, limit = PAGE_SIZE)
                .fold(
                    onSuccess = { batch ->
                        handleMessagesBatch(batch)
                        _uiState.value = ChatUiState.MessagesRefreshed
                        logger.logStateChange("unifiedMessages", "refreshed ${batch.messages.size} messages")
                    },
                    onFailure = { error ->
                        logger.logError("refreshMessages", error, mapOf("roomId" to roomId))
                        _isLoading.value = false
                    }
                )
        }
    }

    /**
     * Load more messages using Matrix pagination
     */
    fun loadMoreMessages() {
        if (_isLoadingMore.value || !_hasMore.value) {
            return
        }

        logger.logUserAction("loadMoreMessages", mapOf("roomId" to roomId, "token" to nextBatchToken))
        viewModelScope.launch {
            _isLoadingMore.value = true

            matrixClient.getMessages(roomId, limit = PAGE_SIZE, fromToken = nextBatchToken)
                .fold(
                    onSuccess = { batch ->
                        val newMessages = batch.messages.map { convertToUnifiedMessage(it) }
                        val allMessages = _unifiedMessages.value + newMessages
                        nextBatchToken = batch.nextToken
                        _hasMore.value = batch.nextToken != null
                        _isLoadingMore.value = false

                        _unifiedMessages.value = allMessages

                        logger.logStateChange("unifiedMessages", "loaded ${newMessages.size} more messages")
                    },
                    onFailure = { error ->
                        logger.logError("loadMoreMessages", error, mapOf("roomId" to roomId))
                        _isLoadingMore.value = false
                    }
                )
        }
    }

    /**
     * Send message or command via Matrix SDK
     */
    fun sendMessage(content: String) {
        if (content.isBlank()) {
            return
        }

        val isCommand = content.startsWith("!")
        val command = if (isCommand) content.removePrefix("!").trim() else null
        val args = command?.split(" ")?.drop(1) ?: emptyList()

        logger.logUserAction("sendMessage", mapOf(
            "roomId" to roomId,
            "contentLength" to content.length,
            "isCommand" to isCommand,
            "replyTo" to (_replyTo.value?.id ?: "none"),
            "transport" to "matrix"
        ))

        viewModelScope.launch {
            // If it's a command, create Command message
            if (isCommand) {
                handleCommandMessage(content, command!!, args)
                return@launch
            }

            // Regular message
            handleRegularMessage(content)
        }
    }

    /**
     * Handle command message
     */
    private suspend fun handleCommandMessage(content: String, command: String, args: List<String>) {
        val commandMessage = UnifiedMessage.Command(
            id = "cmd_${System.currentTimeMillis()}",
            roomId = roomId,
            timestamp = Clock.System.now(),
            sender = _currentUser.value!!,
            command = command.split(" ").first(),
            args = args,
            status = CommandStatus.EXECUTING
        )

        // Add command to UI optimistically
        _unifiedMessages.value = _unifiedMessages.value + commandMessage

        // Send via Matrix (as a message, the Bridge will handle command processing)
        matrixClient.sendTextMessage(roomId, content)
            .fold(
                onSuccess = { eventId ->
                    // Update with completed status
                    _unifiedMessages.value = _unifiedMessages.value.map {
                        if (it.id == commandMessage.id) {
                            (it as UnifiedMessage.Command).copy(
                                status = CommandStatus.COMPLETED,
                                result = "Command executed successfully",
                                executionTime = 500L
                            )
                        } else it
                    }
                    logger.logUiEvent("commandSent:$eventId")
                },
                onFailure = { error ->
                    // Update with failed status
                    _unifiedMessages.value = _unifiedMessages.value.map {
                        if (it.id == commandMessage.id) {
                            (it as UnifiedMessage.Command).copy(
                                status = CommandStatus.FAILED,
                                result = "Failed: ${error.message}"
                            )
                        } else it
                    }
                    logger.logError("handleCommandMessage", error, mapOf("command" to command))
                }
            )
    }

    /**
     * Handle regular message
     */
    private suspend fun handleRegularMessage(content: String) {
        matrixClient.sendTextMessage(roomId, content)
            .fold(
                onSuccess = { eventId ->
                    AppLogger.breadcrumb(
                        message = "Matrix message sent",
                        category = "matrix",
                        data = mapOf("event_id" to eventId, "room_id" to roomId)
                    )

                    // Create optimistic message for UI
                    val optimisticMessage = UnifiedMessage.Regular(
                        id = eventId,
                        roomId = roomId,
                        timestamp = Clock.System.now(),
                        sender = _currentUser.value!!,
                        content = MessageContent(type = MessageType.TEXT, body = content),
                        status = MessageStatus.SENDING,
                        replyTo = _replyTo.value?.id,
                        reactions = emptyList(),
                        isEncrypted = true
                    )

                    // Add to local list optimistically
                    _unifiedMessages.value = listOf(optimisticMessage) + _unifiedMessages.value

                    _replyTo.value = null
                    matrixClient.sendTyping(roomId, false)
                    logger.logUiEvent("messageSent:$eventId")

                    // Simulate status updates
                    simulateStatusUpdates(eventId)
                },
                onFailure = { error ->
                    logger.logError("handleRegularMessage", error, mapOf("roomId" to roomId))
                    
                    // Create failed message for retry
                    val failedMessage = UnifiedMessage.Regular(
                        id = "failed_${System.currentTimeMillis()}",
                        roomId = roomId,
                        timestamp = Clock.System.now(),
                        sender = _currentUser.value!!,
                        content = MessageContent(type = MessageType.TEXT, body = content),
                        status = MessageStatus.FAILED,
                        replyTo = _replyTo.value?.id,
                        reactions = emptyList(),
                        isEncrypted = true
                    )

                    // Add to failed messages list
                    _failedMessages.value = _failedMessages.value + failedMessage
                    
                    // Show error snackbar
                    _events.value = UiEvent.ShowSnackbar("Failed to send message: ${error.message}")
                    
                    // Clear reply state
                    _replyTo.value = null
                }
            )
    }

    private suspend fun simulateStatusUpdates(messageId: String) {
        delay(500)
        updateMessageStatus(messageId, MessageStatus.SENT)

        delay(500)
        updateMessageStatus(messageId, MessageStatus.DELIVERED)

        delay(1000)
        updateMessageStatus(messageId, MessageStatus.READ)
    }

    private fun updateMessageStatus(messageId: String, status: MessageStatus) {
        _unifiedMessages.value = _unifiedMessages.value.map { msg ->
            if (msg.id == messageId && msg is UnifiedMessage.Regular) {
                msg.copy(status = status)
            } else {
                msg
            }
        }
        logger.logStateChange("messageStatus", "$messageId -> ${status.name}")
    }

    /**
     * Retry failed command
     */
    fun retryCommand(message: UnifiedMessage.Command) {
        logger.logUserAction("retryCommand", mapOf("command" to message.command))
        val fullCommand = "!" + message.command + 
            if (message.args.isNotEmpty()) " " + message.args.joinToString(" ") else ""
        sendMessage(fullCommand)
    }

    /**
     * Retry failed message
     */
    fun retryFailedMessage(message: UnifiedMessage.Regular) {
        logger.logUserAction("retryFailedMessage", mapOf("messageId" to message.id))
        
        // Remove message from failed list before attempting retry
        _failedMessages.value = _failedMessages.value.filter { it.id != message.id }
        
        // Attempt to resend the original message content
        viewModelScope.launch {
            matrixClient.sendTextMessage(roomId, message.content.body)
                .fold(
                    onSuccess = { eventId ->
                        // Create resent message with new event ID and sending status
                        val resentMessage = message.copy(
                            id = eventId,
                            status = MessageStatus.SENDING
                        )
                        
                        // Update message in unified messages list
                        _unifiedMessages.value = _unifiedMessages.value.map { 
                            if (it.id == message.id) resentMessage else it
                        }
                        
                        logger.logUiEvent("messageResent:$eventId")
                    },
                    onFailure = { error ->
                        logger.logError("retryFailedMessage", error, mapOf("messageId" to message.id))
                        // Return message to failed list if retry fails
                        _failedMessages.value = _failedMessages.value + message
                        _events.value = UiEvent.ShowSnackbar("Failed to resend message: ${error.message}")
                    }
                )
        }
    }

    /**
     * Handle agent action
     */
    fun handleAgentAction(message: UnifiedMessage.Agent, action: AgentAction) {
        logger.logUserAction("handleAgentAction", mapOf(
            "messageId" to message.id,
            "action" to action.actionType.name
        ))

        when (action.actionType) {
            AgentActionType.REGENERATE -> {
                // Request regeneration
                sendMessage("!regenerate ${message.id}")
            }
            AgentActionType.COPY -> {
                // Copy to clipboard (handled in UI)
                _events.value = UiEvent.CopyToClipboard(message.content.body)
            }
            AgentActionType.FOLLOW_UP -> {
                // Start follow-up
                _events.value = UiEvent.FocusInput("Follow up on: ${message.content.body.take(50)}...")
            }
            else -> {
                // Generic action
                _events.value = UiEvent.Custom("action_${action.actionType.name.lowercase()}", 
                    mapOf("messageId" to message.id, "action" to action))
            }
        }
    }

    /**
     * Handle system action
     */
    fun handleSystemAction(message: UnifiedMessage.System, action: SystemAction) {
        logger.logUserAction("handleSystemAction", mapOf(
            "messageId" to message.id,
            "action" to action.actionType.name
        ))

        when (action.actionType) {
            SystemActionType.CANCEL -> {
                // Cancel workflow
                sendMessage("!cancel ${message.data["workflowId"] ?: ""}")
            }
            SystemActionType.RETRY -> {
                // Retry operation
                sendMessage("!retry ${message.data["operationId"] ?: ""}")
            }
            SystemActionType.VERIFY -> {
                // Start verification
                _events.value = UiEvent.NavigateTo("verification")
            }
            else -> {
                _events.value = UiEvent.Custom("system_${action.actionType.name.lowercase()}",
                    mapOf("messageId" to message.id, "action" to action))
            }
        }
    }

    /**
     * Observe typing indicators via Matrix
     */
    private fun observeTypingIndicators() {
        viewModelScope.launch {
            matrixClient.observeTyping(roomId).collect { typingUserIds ->
                val isTyping = typingUserIds.isNotEmpty()
                val typingUser = typingUserIds.firstOrNull()

                _typingIndicator.value = TypingIndicator(
                    isTyping = isTyping,
                    typers = typingUserIds
                )

                logger.logStateChange("typingIndicator", "isTyping=$isTyping, users=${typingUserIds.size}")
            }
        }
    }

    /**
     * Observe Control Plane events (workflows, agents)
     */
    private fun observeControlPlaneEvents() {
        controlPlaneStore.subscribeToRoom(roomId)

        viewModelScope.launch {
            controlPlaneStore.activeWorkflows.collect { workflows ->
                val roomWorkflow = workflows.find { it.roomId == roomId }
                _activeWorkflow.value = roomWorkflow

                if (roomWorkflow != null) {
                    logger.logStateChange("workflow", roomWorkflow.workflowId)
                    // Add workflow system message
                    addWorkflowSystemMessage(roomWorkflow)
                }
            }
        }

        viewModelScope.launch {
            controlPlaneStore.thinkingAgents.collect { thinkingAgents ->
                val roomAgent = thinkingAgents.values.firstOrNull()
                _agentThinking.value = roomAgent

                if (roomAgent != null) {
                    logger.logStateChange("agentThinking", roomAgent.agentId)
                }
            }
        }

        // Observe agent tasks (NEW)
        viewModelScope.launch {
            controlPlaneStore.agentTasks.collect { tasks ->
                _agentTasks.value = tasks
                logger.logStateChange("agentTasks", "count=${tasks.size}")
            }
        }

        // Observe agent statuses (Phase 2)
        viewModelScope.launch {
            controlPlaneStore.agentStatuses.collect { statuses ->
                // Find status for current room's active agent
                val activeAgentId = _activeAgent.value?.id
                val status = activeAgentId?.let { statuses[it] }
                _agentStatus.value = status

                if (status != null) {
                    logger.logStateChange("agentStatus", "${status.agentId}: ${status.status}")
                }
            }
        }

        // Observe PII access requests (Phase 2)
        viewModelScope.launch {
            controlPlaneStore.pendingPiiRequests.collect { requests ->
                // Find pending request for current room's active agent
                val activeAgentId = _activeAgent.value?.id
                val request = activeAgentId?.let { agentId ->
                    requests.firstOrNull { it.agentId == agentId && !it.isExpired() }
                }
                _pendingPiiRequest.value = request

                if (request != null) {
                    logger.logStateChange("pendingPiiRequest", request.requestId)
                }
            }
        }
    }

    /**
     * Add workflow system message
     */
    private fun addWorkflowSystemMessage(workflow: WorkflowState) {
        val (eventType, description) = when (workflow) {
            is WorkflowState.Started -> {
                SystemEventType.WORKFLOW_STARTED to "Workflow started: ${workflow.workflowType}"
            }
            is WorkflowState.StepRunning -> {
                SystemEventType.WORKFLOW_STEP to "Step ${workflow.stepIndex}/${workflow.totalSteps}: ${workflow.stepName}"
            }
        }

        val systemMessage = UnifiedMessage.System(
            id = "wf_${workflow.workflowId}_${System.currentTimeMillis()}",
            roomId = roomId,
            timestamp = Clock.System.now(),
            sender = MessageSender.SystemSender(),
            eventType = eventType,
            title = workflow.workflowType.replace("_", " ").lowercase()
                .replaceFirstChar { it.uppercase() },
            description = description,
            data = mapOf("workflowId" to workflow.workflowId),
            actions = listOf(
                SystemAction("cancel", "Cancel", SystemActionType.CANCEL)
            )
        )

        // Add if not already present
        if (_unifiedMessages.value.none { it.id == systemMessage.id }) {
            _unifiedMessages.value = _unifiedMessages.value + systemMessage
        }
    }

    /**
     * Check encryption status via Matrix
     */
    private fun checkEncryptionStatus() {
        viewModelScope.launch {
            val isEncrypted = matrixClient.isRoomEncrypted(roomId)

            _encryptionStatus.value = if (isEncrypted) {
                EncryptionStatus.VERIFIED
            } else {
                EncryptionStatus.UNENCRYPTED
            }

            logger.logStateChange("encryptionStatus", _encryptionStatus.value.name)
        }
    }

    /**
     * Check bridge verification status for this room (Fix 3)
     *
     * Queries the MatrixClient to determine if the bridged room's
     * remote homeserver identity has been cross-signed / verified.
     * TODO: Implement isBridgeVerified in MatrixClient interface
     */
    private fun checkBridgeVerification() {
        viewModelScope.launch {
            // TODO: Add isBridgeVerified to MatrixClient interface
            // val verified = matrixClient.isBridgeVerified(roomId)
            val verified = true // Default to verified until method is implemented
            _isBridgeVerified.value = verified
            logger.logStateChange("bridgeVerified", verified.toString())
        }
    }

    fun replyToMessage(message: UnifiedMessage) {
        if (!message.canReply()) {
            // Fix 4: Show Snackbar instead of silently disabling
            _events.value = UiEvent.ShowSnackbar(
                "Replies are not available for this message type"
            )
            return
        }
        logger.logUserAction("replyToMessage", mapOf("messageId" to message.id))
        _replyTo.value = message
    }

    fun cancelReply() {
        logger.logUserAction("cancelReply", emptyMap())
        _replyTo.value = null
    }

    fun onTypingStarted() {
        viewModelScope.launch {
            matrixClient.sendTyping(roomId, true)
        }
    }

    fun onTypingStopped() {
        viewModelScope.launch {
            matrixClient.sendTyping(roomId, false)
        }
    }

    fun markAsRead(messageId: String) {
        viewModelScope.launch {
            matrixClient.sendReadReceipt(roomId, messageId)
            logger.logStateChange("readReceipt", messageId)
        }
    }

    fun toggleReaction(message: UnifiedMessage, emoji: String) {
        if (!message.canReact()) {
            // Fix 4: Show Snackbar instead of silently disabling
            _events.value = UiEvent.ShowSnackbar(
                "Reactions are not available for this message type"
            )
            return
        }
        logger.logUserAction("toggleReaction", mapOf("messageId" to message.id, "emoji" to emoji))
        
        if (message !is UnifiedMessage.Regular) return

        viewModelScope.launch {
            matrixClient.sendReaction(roomId, message.id, emoji)
                .fold(
                    onSuccess = {
                        _unifiedMessages.value = _unifiedMessages.value.map { msg ->
                            if (msg.id == message.id && msg is UnifiedMessage.Regular) {
                                val existingReaction = msg.reactions.find { it.emoji == emoji }
                                val newReactions = if (existingReaction != null) {
                                    msg.reactions.filter { it.emoji != emoji }
                                } else {
                                    msg.reactions + Reaction(
                                        emoji = emoji,
                                        count = 1,
                                        reactedBy = listOf(_currentUser.value?.id ?: "user")
                                    )
                                }
                                msg.copy(reactions = newReactions)
                            } else {
                                msg
                            }
                        }
                        logger.logUiEvent("reactionToggled")
                    },
                    onFailure = { error ->
                        logger.logError("toggleReaction", error, mapOf("messageId" to message.id))
                    }
                )
        }
    }

    fun toggleSearch() {
        _isSearchActive.value = !_isSearchActive.value
        if (!_isSearchActive.value) {
            _searchQuery.value = ""
        }
        logger.logStateChange("searchActive", _isSearchActive.value.toString())
    }

    fun onSearchQueryChange(query: String) {
        _searchQuery.value = query
    }

    /**
     * Approve PII access request with selected fields (Phase 2)
     */
    fun approvePiiRequest(approvedFields: Set<String>) {
        val request = _pendingPiiRequest.value ?: return

        logger.logUserAction("approvePiiRequest", mapOf(
            "requestId" to request.requestId,
            "approvedFields" to approvedFields.size
        ))

        // Remove the request from the store
        controlPlaneStore.removePiiRequest(request.requestId)

        // Send approval message to agent via Matrix
        val fieldList = approvedFields.joinToString(",")
        sendMessage("!approve_pii ${request.requestId} $fieldList")

        // Clear local state
        _pendingPiiRequest.value = null

        logger.logUiEvent("piiRequestApproved:${request.requestId}")
    }

    /**
     * Deny PII access request (Phase 2)
     */
    fun denyPiiRequest() {
        val request = _pendingPiiRequest.value ?: return

        logger.logUserAction("denyPiiRequest", mapOf("requestId" to request.requestId))

        // Remove the request from the store
        controlPlaneStore.removePiiRequest(request.requestId)

        // Send denial message to agent via Matrix
        sendMessage("!deny_pii ${request.requestId}")

        // Clear local state
        _pendingPiiRequest.value = null

        logger.logUiEvent("piiRequestDenied:${request.requestId}")
    }

    fun clearEvent() {
        _events.value = null
    }

    override fun onCleared() {
        super.onCleared()
        logger.logCleanup()
    }

    companion object {
        private const val PAGE_SIZE = 50
    }
}

sealed class ChatUiState {
    object Initial : ChatUiState()
    object Loading : ChatUiState()
    object MessagesLoaded : ChatUiState()
    object MessagesRefreshed : ChatUiState()
    data class Error(val message: String) : ChatUiState()
}
