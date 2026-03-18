package com.armorclaw.shared.data.store

import com.armorclaw.shared.domain.repository.WorkflowRepository
import com.armorclaw.shared.domain.repository.AgentRepository
import com.armorclaw.shared.domain.model.AgentTaskStatusEvent
import com.armorclaw.shared.domain.model.AgentTaskStatus
import com.armorclaw.shared.domain.model.AgentSummary
import com.armorclaw.shared.domain.model.ActivityEvent
import com.armorclaw.shared.domain.model.ActivityEventType
import com.armorclaw.shared.domain.model.AttentionItem
import com.armorclaw.shared.domain.model.AttentionPriority
import com.armorclaw.shared.domain.model.PiiAccessRequest
import com.armorclaw.shared.domain.model.KeystoreStatus
import com.armorclaw.shared.domain.model.UnsealMethod
import com.armorclaw.shared.domain.model.KEYSTORE_SESSION_DURATION_MS
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.matrix.event.*
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.LoggerDelegate
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

/**
 * Control Plane Store
 *
 * Processes ArmorClaw-specific Matrix events (workflows, agent tasks, etc.)
 * and updates the application state accordingly.
 *
 * ## Architecture Position
 * ```
 * MatrixClient.sync() → MatrixEvent → ControlPlaneStore → Repository/UI
 * ```
 *
 * ## Event Flow
 * 1. MatrixClient receives events via sync
 * 2. ControlPlaneStore filters for ArmorClaw events
 * 3. Events are parsed and dispatched to appropriate handlers
 * 4. Repositories are updated
 * 5. UI observes state changes via StateFlows
 */
class ControlPlaneStore(
    private val matrixClient: MatrixClient,
    private val workflowRepository: WorkflowRepository,
    private val agentRepository: AgentRepository,
    private val json: Json = Json { ignoreUnknownKeys = true },
    externalScope: CoroutineScope? = null
) {
    private val logger = LoggerDelegate(LogTag.Domain.ControlPlane)
    private val internalJob = if (externalScope == null) SupervisorJob() else null
    private val scope = externalScope ?: CoroutineScope(internalJob!! + Dispatchers.Default)

    /**
     * Cancel all event collection and clean up resources.
     * Only cancels if the scope was created internally.
     * This should be called when the store is no longer needed.
     */
    fun cancel() {
        internalJob?.cancel()
    }

    // ========================================================================
    // State Flows
    // ========================================================================

    private val _activeWorkflows = MutableStateFlow<List<WorkflowState>>(emptyList())
    val activeWorkflows: StateFlow<List<WorkflowState>> = _activeWorkflows.asStateFlow()

    private val _agentTasks = MutableStateFlow<List<AgentTaskState>>(emptyList())
    val agentTasks: StateFlow<List<AgentTaskState>> = _agentTasks.asStateFlow()

    private val _thinkingAgents = MutableStateFlow<Map<String, AgentThinkingState>>(emptyMap())
    val thinkingAgents: StateFlow<Map<String, AgentThinkingState>> = _thinkingAgents.asStateFlow()

    private val _budgetWarnings = MutableStateFlow<List<BudgetWarningState>>(emptyList())
    val budgetWarnings: StateFlow<List<BudgetWarningState>> = _budgetWarnings.asStateFlow()

    private val _bridgeStatus = MutableStateFlow<Map<String, BridgeStatusState>>(emptyMap())
    val bridgeStatus: StateFlow<Map<String, BridgeStatusState>> = _bridgeStatus.asStateFlow()

    // Agent Status Tracking (Phase 2 - Day 5)
    private val _agentStatuses = MutableStateFlow<Map<String, AgentTaskStatusEvent>>(emptyMap())
    val agentStatuses: StateFlow<Map<String, AgentTaskStatusEvent>> = _agentStatuses.asStateFlow()

    // PII Access Requests (Phase 2 - Day 6)
    private val _pendingPiiRequests = MutableStateFlow<List<PiiAccessRequest>>(emptyList())
    val pendingPiiRequests: StateFlow<List<PiiAccessRequest>> = _pendingPiiRequests.asStateFlow()

    // Keystore Status (Phase 2 - Day 8)
    private val _keystoreStatus = MutableStateFlow<KeystoreStatus>(KeystoreStatus.Sealed())
    val keystoreStatus: StateFlow<KeystoreStatus> = _keystoreStatus.asStateFlow()

    // Mission Control - Attention Queue (VPS Secretary Mode)
    private val _needsAttentionQueue = MutableStateFlow<List<AttentionItem>>(emptyList())
    val needsAttentionQueue: StateFlow<List<AttentionItem>> = _needsAttentionQueue.asStateFlow()

    // Mission Control - Agents Paused State
    private val _isPaused = MutableStateFlow(false)
    val isPaused: StateFlow<Boolean> = _isPaused.asStateFlow()

    // Mission Control - Activity Timeline (VPS Secretary Mode)
    private val _activityTimeline = MutableStateFlow<List<ActivityEvent>>(emptyList())
    val activityTimeline: StateFlow<List<ActivityEvent>> = _activityTimeline.asStateFlow()

    // ========================================================================
    // Initialization
    // ========================================================================

    init {
        // Subscribe to ArmorClaw events
        subscribeToEvents()
    }

    /**
     * Start processing events for a specific room
     */
    fun subscribeToRoom(roomId: String) {
        logger.logInfo("Subscribing to Control Plane events for room: $roomId")
        scope.launch {
            matrixClient.observeArmorClawEvents(roomId)
                .collect { event -> processEvent(event) }
        }
    }

    /**
     * Start processing events for all rooms
     */
    private fun subscribeToEvents() {
        logger.logInfo("Subscribing to global Control Plane events")
        scope.launch {
            matrixClient.observeArmorClawEvents()
                .collect { event -> processEvent(event) }
        }
    }

    // ========================================================================
    // Event Processing
    // ========================================================================

    private suspend fun processEvent(event: MatrixEvent) {
        logger.logDebug(
            "Processing Control Plane event",
            mapOf("type" to event.type, "roomId" to event.roomId)
        )

        when (event.type) {
            // Workflow events
            ArmorClawEventType.WORKFLOW_STARTED -> {
                val data = json.decodeFromString<WorkflowStartedEvent>(event.content)
                handleWorkflowStarted(event.roomId, data)
            }

            ArmorClawEventType.WORKFLOW_STEP -> {
                val data = json.decodeFromString<WorkflowStepEvent>(event.content)
                handleWorkflowStep(event.roomId, data)
            }

            ArmorClawEventType.WORKFLOW_COMPLETED -> {
                val data = json.decodeFromString<WorkflowCompletedEvent>(event.content)
                handleWorkflowCompleted(event.roomId, data)
            }

            ArmorClawEventType.WORKFLOW_FAILED -> {
                val data = json.decodeFromString<WorkflowFailedEvent>(event.content)
                handleWorkflowFailed(event.roomId, data)
            }

            // Agent events
            ArmorClawEventType.AGENT_TASK_STARTED -> {
                val data = json.decodeFromString<AgentTaskStartedEvent>(event.content)
                handleAgentTaskStarted(data)
            }

            ArmorClawEventType.AGENT_TASK_PROGRESS -> {
                val data = json.decodeFromString<AgentTaskProgressEvent>(event.content)
                handleAgentTaskProgress(data)
            }

            ArmorClawEventType.AGENT_TASK_COMPLETE -> {
                val data = json.decodeFromString<AgentTaskCompleteEvent>(event.content)
                handleAgentTaskComplete(data)
            }

            ArmorClawEventType.AGENT_THINKING -> {
                val data = json.decodeFromString<AgentThinkingEvent>(event.content)
                handleAgentThinking(data)
            }

            // System events
            ArmorClawEventType.BUDGET_WARNING -> {
                val data = json.decodeFromString<BudgetWarningEvent>(event.content)
                handleBudgetWarning(data)
            }

            ArmorClawEventType.BRIDGE_CONNECTED -> {
                val data = json.decodeFromString<BridgeConnectedEvent>(event.content)
                handleBridgeConnected(data)
            }

            ArmorClawEventType.BRIDGE_DISCONNECTED -> {
                val data = json.decodeFromString<BridgeDisconnectedEvent>(event.content)
                handleBridgeDisconnected(data)
            }

            else -> {
                logger.logWarning("Unknown Control Plane event type: ${event.type}")
            }
        }
    }

    // ========================================================================
    // Workflow Event Handlers
    // ========================================================================

    private suspend fun handleWorkflowStarted(roomId: String, event: WorkflowStartedEvent) {
        logger.logInfo(
            "Workflow started",
            mapOf(
                "workflowId" to event.workflowId,
                "type" to event.workflowType,
                "initiatedBy" to event.initiatedBy
            )
        )

        // Add to active workflows
        val state = WorkflowState.Started(
            workflowId = event.workflowId,
            workflowType = event.workflowType,
            roomId = roomId,
            initiatedBy = event.initiatedBy,
            timestamp = event.timestamp
        )
        _activeWorkflows.update { it + state }

        // Persist to repository
        workflowRepository.startWorkflow(
            workflowId = event.workflowId,
            type = event.workflowType,
            roomId = roomId,
            parameters = event.parameters
        )
    }

    private suspend fun handleWorkflowStep(roomId: String, event: WorkflowStepEvent) {
        logger.logInfo(
            "Workflow step update",
            mapOf(
                "workflowId" to event.workflowId,
                "step" to event.stepName,
                "status" to event.status.name
            )
        )

        // Update active workflow state
        _activeWorkflows.update { workflows ->
            workflows.map { workflow ->
                if (workflow is WorkflowState.Started && workflow.workflowId == event.workflowId) {
                    WorkflowState.StepRunning(
                        workflowId = event.workflowId,
                        workflowType = workflow.workflowType,
                        roomId = roomId,
                        stepId = event.stepId,
                        stepName = event.stepName,
                        stepIndex = event.stepIndex,
                        totalSteps = event.totalSteps,
                        status = event.status,
                        timestamp = event.timestamp
                    )
                } else {
                    workflow
                }
            }
        }

        // Persist to repository
        workflowRepository.updateStep(
            workflowId = event.workflowId,
            stepId = event.stepId,
            status = event.status,
            output = event.output,
            error = event.error
        )
    }

    private suspend fun handleWorkflowCompleted(roomId: String, event: WorkflowCompletedEvent) {
        logger.logInfo(
            "Workflow completed",
            mapOf(
                "workflowId" to event.workflowId,
                "success" to event.success,
                "duration" to event.duration
            )
        )

        // Remove from active workflows, add to completed
        _activeWorkflows.update { workflows ->
            workflows.filterNot { it.workflowId == event.workflowId }
        }

        // Persist to repository
        workflowRepository.completeWorkflow(
            workflowId = event.workflowId,
            success = event.success,
            result = event.result,
            error = event.error
        )
    }

    private suspend fun handleWorkflowFailed(roomId: String, event: WorkflowFailedEvent) {
        logger.logWarning(
            "Workflow failed",
            mapOf(
                "workflowId" to event.workflowId,
                "failedAtStep" to event.failedAtStep,
                "error" to event.error
            )
        )

        // Remove from active workflows
        _activeWorkflows.update { workflows ->
            workflows.filterNot { it.workflowId == event.workflowId }
        }

        // Persist to repository
        workflowRepository.failWorkflow(
            workflowId = event.workflowId,
            failedAtStep = event.failedAtStep,
            error = event.error,
            recoverable = event.recoverable
        )
    }

    // ========================================================================
    // Agent Event Handlers
    // ========================================================================

    private suspend fun handleAgentTaskStarted(event: AgentTaskStartedEvent) {
        logger.logInfo(
            "Agent task started",
            mapOf(
                "agentId" to event.agentId,
                "taskId" to event.taskId,
                "type" to event.taskType
            )
        )

        // Add to active tasks
        val state = AgentTaskState.Running(
            agentId = event.agentId,
            agentName = event.agentName,
            taskId = event.taskId,
            taskType = event.taskType,
            roomId = event.roomId,
            startTime = event.timestamp
        )
        _agentTasks.update { it + state }

        // Clear thinking state for this agent
        _thinkingAgents.update { it - event.agentId }

        // Persist to repository
        agentRepository.taskStarted(
            agentId = event.agentId,
            taskId = event.taskId,
            taskType = event.taskType,
            roomId = event.roomId
        )
    }

    private suspend fun handleAgentTaskProgress(event: AgentTaskProgressEvent) {
        logger.logDebug(
            "Agent task progress",
            mapOf(
                "taskId" to event.taskId,
                "progress" to event.progress
            )
        )

        // Update task state
        _agentTasks.update { tasks ->
            tasks.map { task ->
                if (task.taskId == event.taskId && task is AgentTaskState.Running) {
                    task.copy(progress = event.progress)
                } else {
                    task
                }
            }
        }
    }

    private suspend fun handleAgentTaskComplete(event: AgentTaskCompleteEvent) {
        logger.logInfo(
            "Agent task complete",
            mapOf(
                "agentId" to event.agentId,
                "taskId" to event.taskId,
                "success" to event.success
            )
        )

        // Remove from active tasks
        _agentTasks.update { tasks ->
            tasks.filterNot { it.taskId == event.taskId }
        }

        // Persist to repository
        agentRepository.taskCompleted(
            agentId = event.agentId,
            taskId = event.taskId,
            success = event.success,
            result = event.result,
            error = event.error
        )
    }

    private fun handleAgentThinking(event: AgentThinkingEvent) {
        logger.logDebug(
            "Agent thinking",
            mapOf("agentId" to event.agentId, "message" to (event.message ?: ""))
        )

        // Update thinking state
        val state = AgentThinkingState(
            agentId = event.agentId,
            agentName = event.agentName,
            message = event.message,
            timestamp = event.timestamp
        )
        _thinkingAgents.update { it + (event.agentId to state) }

        // Auto-clear after 30 seconds
        scope.launch {
            kotlinx.coroutines.delay(30000)
            _thinkingAgents.update { current ->
                current[event.agentId]?.let { thinking ->
                    if (thinking.timestamp == event.timestamp) {
                        current - event.agentId
                    } else {
                        current
                    }
                } ?: current
            }
        }
    }

    // ========================================================================
    // System Event Handlers
    // ========================================================================

    private fun handleBudgetWarning(event: BudgetWarningEvent) {
        logger.logWarning(
            "Budget warning",
            mapOf(
                "userId" to event.userId,
                "percentageUsed" to event.percentageUsed,
                "level" to event.warningLevel.name
            )
        )

        // Add to warnings
        val state = BudgetWarningState(
            userId = event.userId,
            currentSpend = event.currentSpend,
            limit = event.limit,
            percentageUsed = event.percentageUsed,
            warningLevel = event.warningLevel,
            timestamp = event.timestamp
        )
        _budgetWarnings.update { warnings ->
            // Keep only the latest warning per user
            (warnings.filterNot { it.userId == event.userId } + state)
                .sortedByDescending { it.timestamp }
        }
    }

    private fun handleBridgeConnected(event: BridgeConnectedEvent) {
        logger.logInfo(
            "Bridge connected",
            mapOf(
                "platform" to event.platformType,
                "status" to event.status
            )
        )

        val state = BridgeStatusState.Connected(
            platformType = event.platformType,
            platformName = event.platformName,
            timestamp = event.timestamp
        )
        _bridgeStatus.update { it + (event.platformType to state) }
    }

    private fun handleBridgeDisconnected(event: BridgeDisconnectedEvent) {
        logger.logInfo(
            "Bridge disconnected",
            mapOf(
                "platform" to event.platformType,
                "reason" to (event.reason ?: "unknown")
            )
        )

        val state = BridgeStatusState.Disconnected(
            platformType = event.platformType,
            platformName = event.platformName,
            reason = event.reason,
            timestamp = event.timestamp
        )
        _bridgeStatus.update { it + (event.platformType to state) }
    }

    // ========================================================================
    // Public API
    // ========================================================================

    /**
     * Get workflow state by ID
     */
    fun getWorkflowState(workflowId: String): WorkflowState? {
        return _activeWorkflows.value.find { it.workflowId == workflowId }
    }

    /**
     * Get all active workflows for a room
     */
    fun getRoomWorkflows(roomId: String): List<WorkflowState> {
        return _activeWorkflows.value.filter { it.roomId == roomId }
    }

    /**
     * Get agent task state by ID
     */
    fun getTaskState(taskId: String): AgentTaskState? {
        return _agentTasks.value.find { it.taskId == taskId }
    }

    /**
     * Check if an agent is currently thinking
     */
    fun isAgentThinking(agentId: String): Boolean {
        return _thinkingAgents.value.containsKey(agentId)
    }

    /**
     * Get all thinking agents in a room
     */
    fun getThinkingAgentsInRoom(roomId: String): List<AgentThinkingState> {
        // This would require tracking which room each agent is in
        // For now, return all thinking agents
        return _thinkingAgents.value.values.toList()
    }

    /**
     * Clear all state (e.g., on logout)
     */
    fun clear() {
        _activeWorkflows.value = emptyList()
        _agentTasks.value = emptyList()
        _thinkingAgents.value = emptyMap()
        _budgetWarnings.value = emptyList()
        _bridgeStatus.value = emptyMap()
        _agentStatuses.value = emptyMap()
        _pendingPiiRequests.value = emptyList()
        _keystoreStatus.value = KeystoreStatus.Sealed()
        _needsAttentionQueue.value = emptyList()
        _isPaused.value = false
    }

    // ========================================================================
    // Agent Status API (Phase 2 - Day 5)
    // ========================================================================

    /**
     * Process an agent status event
     */
    fun processStatusEvent(event: AgentTaskStatusEvent) {
        logger.logDebug(
            "Processing agent status event",
            mapOf("agentId" to event.agentId, "status" to event.status.name)
        )

        _agentStatuses.update { map ->
            if (event.status == AgentTaskStatus.IDLE || event.status == AgentTaskStatus.COMPLETE) {
                // Remove completed/idle agents from active status
                map - event.agentId
            } else {
                map + (event.agentId to event)
            }
        }
    }

    /**
     * Get agent status by ID
     */
    fun getAgentStatus(agentId: String): AgentTaskStatusEvent? {
        return _agentStatuses.value[agentId]
    }

    /**
     * Get all active agent statuses
     */
    fun getActiveAgentStatuses(): List<AgentTaskStatusEvent> {
        return _agentStatuses.value.values.toList()
    }

    // ========================================================================
    // PII Access Request API (Phase 2 - Day 6)
    // ========================================================================

    /**
     * Add a pending PII access request
     */
    fun addPiiRequest(request: PiiAccessRequest) {
        logger.logInfo(
            "Adding PII access request",
            mapOf("requestId" to request.requestId, "agentId" to request.agentId)
        )

        _pendingPiiRequests.update { requests ->
            // Remove expired requests and add new one
            requests.filter { !it.isExpired() } + request
        }

        // Auto-expire after timeout
        scope.launch {
            val delayMs = request.expiresAt - System.currentTimeMillis()
            if (delayMs > 0) {
                kotlinx.coroutines.delay(delayMs)
            }
            removePiiRequest(request.requestId)
        }
    }

    /**
     * Remove a PII access request
     */
    fun removePiiRequest(requestId: String) {
        _pendingPiiRequests.update { requests ->
            requests.filterNot { it.requestId == requestId }
        }
    }

    /**
     * Get pending PII requests for an agent
     */
    fun getPendingPiiRequests(agentId: String): List<PiiAccessRequest> {
        return _pendingPiiRequests.value.filter { it.agentId == agentId && !it.isExpired() }
    }

    /**
     * Get all pending PII requests
     */
    fun getAllPendingPiiRequests(): List<PiiAccessRequest> {
        return _pendingPiiRequests.value.filter { !it.isExpired() }
    }

    // ========================================================================
    // Keystore Status API (Phase 2 - Day 8)
    // ========================================================================

    /**
     * Process a keystore status event
     */
    fun processKeystoreEvent(eventType: String, content: Map<String, Any>) {
        logger.logDebug(
            "Processing keystore event",
            mapOf("type" to eventType)
        )

        when (eventType) {
            "com.armorclaw.keystore.sealed" -> {
                _keystoreStatus.value = KeystoreStatus.Sealed()
            }
            "com.armorclaw.keystore.unsealed" -> {
                val expiresAt = (content["expiresAt"] as? Number)?.toLong()
                    ?: (System.currentTimeMillis() + KEYSTORE_SESSION_DURATION_MS)
                val methodStr = content["method"] as? String ?: "PASSWORD"
                val method = try {
                    UnsealMethod.valueOf(methodStr.uppercase())
                } catch (e: IllegalArgumentException) {
                    UnsealMethod.PASSWORD
                }
                _keystoreStatus.value = KeystoreStatus.Unsealed(
                    expiresAt = expiresAt,
                    unsealedBy = method
                )
            }
            "com.armorclaw.keystore.error" -> {
                val message = content["message"] as? String ?: "Unknown error"
                _keystoreStatus.value = KeystoreStatus.Error(message)
            }
        }
    }

    /**
     * Set keystore status (for local operations)
     */
    fun setKeystoreStatus(status: KeystoreStatus) {
        _keystoreStatus.value = status
    }

    /**
     * Check if keystore is accessible
     */
    fun isKeystoreAccessible(): Boolean {
        return _keystoreStatus.value.isAccessible()
    }

    /**
     * Reseal the keystore
     */
    fun resealKeystore() {
        logger.logInfo("Resealing keystore")
        _keystoreStatus.value = KeystoreStatus.Sealed()
    }

    // ========================================================================
    // Mission Control API (VPS Secretary Mode)
    // ========================================================================

    /**
     * Add an attention item to the queue
     */
    fun addAttentionItem(item: AttentionItem) {
        logger.logInfo(
            "Adding attention item",
            mapOf("id" to item.id, "type" to (item::class.simpleName ?: "Unknown"), "priority" to item.priority.name)
        )

        _needsAttentionQueue.update { items ->
            // Remove existing item with same ID, then add new one
            (items.filterNot { it.id == item.id } + item)
                .sortedByDescending { it.priority.ordinal }
        }
    }

    /**
     * Remove an attention item from the queue
     */
    fun removeAttentionItem(itemId: String) {
        logger.logInfo("Removing attention item", mapOf("id" to itemId))
        _needsAttentionQueue.update { items ->
            items.filterNot { it.id == itemId }
        }
    }

    /**
     * Get attention item by ID
     */
    fun getAttentionItem(itemId: String): AttentionItem? {
        return _needsAttentionQueue.value.find { it.id == itemId }
    }

    /**
     * Get highest priority in attention queue
     */
    fun getHighestPriority(): AttentionPriority? {
        return _needsAttentionQueue.value.minByOrNull { it.priority.ordinal }?.priority
    }

    /**
     * Pause all agents
     */
    fun pauseAllAgents() {
        logger.logInfo("Pausing all agents")
        _isPaused.value = true
    }

    /**
     * Resume all agents
     */
    fun resumeAllAgents() {
        logger.logInfo("Resuming all agents")
        _isPaused.value = false
    }

    /**
     * Emergency stop - stops all agents and clears attention queue
     */
    fun emergencyStop() {
        logger.logWarning("Emergency stop triggered")
        _isPaused.value = true
        _agentTasks.value = emptyList()
        _thinkingAgents.value = emptyMap()
        _activeWorkflows.value = emptyList()
        // Keep attention queue but update items to show they were interrupted
    }

    /**
     * Get agent summaries for Mission Control dashboard
     */
    fun getAgentSummaries(): List<AgentSummary> {
        return _agentTasks.value.mapNotNull { task ->
            when (task) {
                is AgentTaskState.Running -> {
                    val status = _agentStatuses.value[task.agentId]?.status
                        ?: com.armorclaw.shared.domain.model.AgentTaskStatus.IDLE
                    AgentSummary(
                        agentId = task.agentId,
                        agentName = task.agentName,
                        status = status,
                        currentTask = task.taskType,
                        roomId = task.roomId,
                        roomName = null, // Would need room lookup
                        progress = task.progress,
                        lastActivity = task.startTime
                    )
                }
            }
        }
    }

    /**
     * Convert PII request to attention item and add to queue
     */
    fun addPiiRequestToAttentionQueue(request: PiiAccessRequest, roomId: String = "") {
        val item = AttentionItem.PiiRequest(
            id = "pii_${request.requestId}",
            agentId = request.agentId,
            agentName = request.agentId.take(20), // Would need agent lookup for real name
            roomId = roomId,
            timestamp = System.currentTimeMillis(),
            piiRequest = request
        )
        addAttentionItem(item)
    }

    /**
     * Clear all Mission Control state
     */
    fun clearMissionControlState() {
        _needsAttentionQueue.value = emptyList()
        _isPaused.value = false
    }

    // ========================================================================
    // Activity Timeline API (VPS Secretary Mode)
    // ========================================================================

    /**
     * Add an activity event to the timeline
     */
    fun addActivityEvent(event: ActivityEvent) {
        logger.logInfo(
            "Adding activity event",
            mapOf("id" to event.id, "type" to event.eventType.name)
        )
        _activityTimeline.update { events ->
            // Keep last 100 events, add new one at end
            (events + event).takeLast(100)
        }
    }

    /**
     * Remove an activity event from the timeline
     */
    fun removeActivityEvent(eventId: String) {
        logger.logInfo("Removing activity event", mapOf("id" to eventId))
        _activityTimeline.update { events ->
            events.filterNot { it.id == eventId }
        }
    }

    /**
     * Clear the activity timeline
     */
    fun clearActivityTimeline() {
        logger.logInfo("Clearing activity timeline")
        _activityTimeline.value = emptyList()
    }

    /**
     * Get activity events for a specific agent
     */
    fun getAgentActivityEvents(agentId: String): List<ActivityEvent> {
        return _activityTimeline.value.filter { it.agentId == agentId }
    }

    /**
     * Get activity events for a specific room
     */
    fun getRoomActivityEvents(roomId: String): List<ActivityEvent> {
        return _activityTimeline.value.filter { it.roomId == roomId }
    }

    /**
     * Get attention-requiring events from timeline
     */
    fun getAttentionEvents(): List<ActivityEvent> {
        return _activityTimeline.value.filter { it.requiresAttention }
    }
}

// ========================================================================
// State Classes
// ========================================================================

sealed class WorkflowState {
    abstract val workflowId: String
    abstract val workflowType: String
    abstract val roomId: String

    data class Started(
        override val workflowId: String,
        override val workflowType: String,
        override val roomId: String,
        val initiatedBy: String,
        val timestamp: Long
    ) : WorkflowState()

    data class StepRunning(
        override val workflowId: String,
        override val workflowType: String,
        override val roomId: String,
        val stepId: String,
        val stepName: String,
        val stepIndex: Int,
        val totalSteps: Int,
        val status: StepStatus,
        val timestamp: Long
    ) : WorkflowState()
}

sealed class AgentTaskState {
    abstract val taskId: String
    abstract val agentId: String
    abstract val agentName: String

    data class Running(
        override val taskId: String,
        override val agentId: String,
        override val agentName: String,
        val taskType: String,
        val roomId: String,
        val startTime: Long,
        val progress: Float = 0f
    ) : AgentTaskState()
}

data class AgentThinkingState(
    val agentId: String,
    val agentName: String,
    val message: String?,
    val timestamp: Long
)

data class BudgetWarningState(
    val userId: String,
    val currentSpend: Double,
    val limit: Double,
    val percentageUsed: Double,
    val warningLevel: WarningLevel,
    val timestamp: Long
)

sealed class BridgeStatusState {
    abstract val platformType: String
    abstract val platformName: String

    data class Connected(
        override val platformType: String,
        override val platformName: String,
        val timestamp: Long
    ) : BridgeStatusState()

    data class Disconnected(
        override val platformType: String,
        override val platformName: String,
        val reason: String?,
        val timestamp: Long
    ) : BridgeStatusState()
}
