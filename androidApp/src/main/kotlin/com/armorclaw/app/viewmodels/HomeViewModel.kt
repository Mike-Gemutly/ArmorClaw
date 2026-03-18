package com.armorclaw.app.viewmodels

import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.data.store.AgentThinkingState
import com.armorclaw.shared.domain.model.AgentSummary
import com.armorclaw.shared.domain.model.AttentionItem
import com.armorclaw.shared.domain.model.AttentionPriority
import com.armorclaw.shared.domain.model.KeystoreStatus
import com.armorclaw.shared.domain.model.Room
import com.armorclaw.shared.domain.repository.RoomRepository
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.viewModelLogger
import com.armorclaw.shared.ui.base.BaseViewModel
import com.armorclaw.shared.ui.base.UiEvent
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * ViewModel for Home screen
 *
 * ## Architecture (Post-Migration)
 * ```
 * HomeViewModel
 *      ├── RoomRepository (room list)
 *      └── ControlPlaneStore (active workflows, agent status)
 * ```
 *
 * Handles room list loading, workflow display, and navigation.
 * Uses ViewModelLogger for proper separation of concerns in logging.
 *
 * ## Migration Status
 * - [x] Room list from RoomRepository
 * - [x] Active workflows from ControlPlaneStore
 * - [x] Agent thinking status
 */
class HomeViewModel(
    private val roomRepository: RoomRepository,
    private val controlPlaneStore: ControlPlaneStore
) : BaseViewModel() {

    private val logger = viewModelLogger("HomeViewModel", LogTag.ViewModel.Home)

    // Room state
    private val _rooms = MutableStateFlow<List<Room>>(emptyList())
    val rooms = _rooms.asStateFlow()

    private val _selectedRoom = MutableStateFlow<Room?>(null)
    val selectedRoom = _selectedRoom.asStateFlow()

    // Workflow state (NEW)
    private val _activeWorkflows = MutableStateFlow<List<WorkflowState>>(emptyList())
    val activeWorkflows = _activeWorkflows.asStateFlow()

    private val _thinkingAgents = MutableStateFlow<List<AgentThinkingState>>(emptyList())
    val thinkingAgents = _thinkingAgents.asStateFlow()

    // Mission Control State (VPS Secretary Mode)
    private val _needsAttentionItems = MutableStateFlow<List<AttentionItem>>(emptyList())
    val needsAttentionItems = _needsAttentionItems.asStateFlow()

    private val _activeAgentSummaries = MutableStateFlow<List<AgentSummary>>(emptyList())
    val activeAgentSummaries = _activeAgentSummaries.asStateFlow()

    private val _vaultStatus = MutableStateFlow<KeystoreStatus>(KeystoreStatus.Sealed())
    val vaultStatus = _vaultStatus.asStateFlow()

    private val _isPaused = MutableStateFlow(false)
    val isPaused = _isPaused.asStateFlow()

    private val _highestPriority = MutableStateFlow<AttentionPriority?>(null)
    val highestPriority = _highestPriority.asStateFlow()

    // UI state
    private val _showWorkflowsSection = MutableStateFlow(false)
    val showWorkflowsSection = _showWorkflowsSection.asStateFlow()

    private val _isRefreshing = MutableStateFlow(false)
    val isRefreshing = _isRefreshing.asStateFlow()

    init {
        loadRooms()
        observeRooms()
        observeWorkflows()
        observeAgents()
        observeMissionControlState()
    }

    private fun loadRooms() {
        executeFlow("loadRooms") {
            roomRepository.getRooms()
                .onSuccess { rooms ->
                    logger.logStateChange("rooms", "${rooms.size} rooms")
                    _rooms.value = rooms
                }
                .onError { error ->
                    logger.logError("loadRooms", error.toException())
                }
        }
    }

    private fun observeRooms() {
        viewModelScope.launch {
            logger.logUserAction("observeRooms")
            roomRepository.observeRooms().collect { rooms ->
                logger.logStateChange("rooms", "${rooms.size} rooms (from flow)")
                _rooms.value = rooms
            }
        }
    }

    /**
     * Observe active workflows from ControlPlaneStore
     */
    private fun observeWorkflows() {
        viewModelScope.launch {
            logger.logUserAction("observeWorkflows")
            controlPlaneStore.activeWorkflows.collect { workflows ->
                logger.logStateChange("workflows", "${workflows.size} active")
                _activeWorkflows.value = workflows
                _showWorkflowsSection.value = workflows.isNotEmpty()
            }
        }
    }

    /**
     * Observe thinking agents from ControlPlaneStore
     */
    private fun observeAgents() {
        viewModelScope.launch {
            logger.logUserAction("observeAgents")
            controlPlaneStore.thinkingAgents.collect { thinkingMap ->
                val agents = thinkingMap.values.toList()
                logger.logStateChange("thinkingAgents", "${agents.size} thinking")
                _thinkingAgents.value = agents
            }
        }
    }

    /**
     * Observe Mission Control state from ControlPlaneStore
     */
    private fun observeMissionControlState() {
        // Observe attention queue
        viewModelScope.launch {
            controlPlaneStore.needsAttentionQueue.collect { items ->
                logger.logStateChange("needsAttentionItems", "${items.size} items")
                _needsAttentionItems.value = items
                _highestPriority.value = items.minByOrNull { it.priority.ordinal }?.priority
            }
        }

        // Observe agent summaries
        viewModelScope.launch {
            controlPlaneStore.agentTasks.collect {
                _activeAgentSummaries.value = controlPlaneStore.getAgentSummaries()
            }
        }

        // Observe vault status
        viewModelScope.launch {
            controlPlaneStore.keystoreStatus.collect { status ->
                logger.logStateChange("vaultStatus", status::class.simpleName ?: "unknown")
                _vaultStatus.value = status
            }
        }

        // Observe paused state
        viewModelScope.launch {
            controlPlaneStore.isPaused.collect { isPaused ->
                logger.logStateChange("isPaused", isPaused.toString())
                _isPaused.value = isPaused
            }
        }
    }

    fun onRoomClick(roomId: String) {
        viewModelScope.launch {
            logger.logUserAction("onRoomClick", mapOf("roomId" to roomId))
            _selectedRoom.value = rooms.value.find { it.id == roomId }
            logger.logNavigation("chat/$roomId")
            emitEvent(UiEvent.NavigateTo(route = "chat", data = mapOf("roomId" to roomId)))
        }
    }

    fun onCreateRoom() {
        viewModelScope.launch {
            logger.logUserAction("onCreateRoom")
            logger.logNavigation("create_room")
            emitEvent(UiEvent.NavigateTo(route = "create_room"))
        }
    }

    /**
     * Navigate to workflow details
     */
    fun onWorkflowClick(workflowId: String) {
        viewModelScope.launch {
            logger.logUserAction("onWorkflowClick", mapOf("workflowId" to workflowId))
            logger.logNavigation("workflow/$workflowId")
            emitEvent(UiEvent.NavigateTo(route = "workflow", data = mapOf("workflowId" to workflowId)))
        }
    }

    /**
     * Cancel an active workflow
     */
    fun onCancelWorkflow(workflowId: String) {
        viewModelScope.launch {
            logger.logUserAction("onCancelWorkflow", mapOf("workflowId" to workflowId))
            // In real implementation, this would call a UseCase to cancel the workflow
            // For now, just log the action
            logger.logStateChange("workflow", "cancelled: $workflowId")
        }
    }

    fun onRefresh() {
        logger.logUserAction("onRefresh")
        _isRefreshing.value = true
        loadRooms()
        viewModelScope.launch {
            // Simulate refresh delay
            kotlinx.coroutines.delay(500)
            _isRefreshing.value = false
        }
    }

    // ========================================================================
    // Mission Control Actions (VPS Secretary Mode)
    // ========================================================================

    /**
     * Emergency stop - immediately halt all agents
     */
    fun emergencyStop() {
        viewModelScope.launch {
            logger.logUserAction("emergencyStop")
            controlPlaneStore.emergencyStop()
            emitEvent(UiEvent.ShowSnackbar("All agents stopped"))
        }
    }

    /**
     * Pause all active agents
     */
    fun pauseAllAgents() {
        viewModelScope.launch {
            logger.logUserAction("pauseAllAgents")
            controlPlaneStore.pauseAllAgents()
            emitEvent(UiEvent.ShowSnackbar("All agents paused"))
        }
    }

    /**
     * Resume all paused agents
     */
    fun resumeAllAgents() {
        viewModelScope.launch {
            logger.logUserAction("resumeAllAgents")
            controlPlaneStore.resumeAllAgents()
            emitEvent(UiEvent.ShowSnackbar("Agents resumed"))
        }
    }

    /**
     * Lock the vault (seal keystore)
     */
    fun lockVault() {
        viewModelScope.launch {
            logger.logUserAction("lockVault")
            controlPlaneStore.resealKeystore()
            emitEvent(UiEvent.ShowSnackbar("Vault locked"))
        }
    }

    /**
     * Handle attention item click
     */
    fun onAttentionItemClick(item: AttentionItem) {
        viewModelScope.launch {
            logger.logUserAction("onAttentionItemClick", mapOf("id" to item.id))
            when (item) {
                is AttentionItem.PiiRequest -> {
                    // Navigate to PII approval screen
                    emitEvent(UiEvent.NavigateTo(
                        route = "pii_approval",
                        data = mapOf("requestId" to item.piiRequest.requestId)
                    ))
                }
                is AttentionItem.CaptchaChallenge,
                is AttentionItem.TwoFactorAuth -> {
                    // Navigate to intervention screen
                    emitEvent(UiEvent.NavigateTo(
                        route = "agent_intervention",
                        data = mapOf("agentId" to item.agentId, "roomId" to item.roomId)
                    ))
                }
                is AttentionItem.ApprovalRequest -> {
                    // Navigate to approval screen
                    emitEvent(UiEvent.NavigateTo(
                        route = "approval",
                        data = mapOf("id" to item.id)
                    ))
                }
                is AttentionItem.ErrorState -> {
                    // Navigate to error resolution screen
                    emitEvent(UiEvent.NavigateTo(
                        route = "error_resolution",
                        data = mapOf("agentId" to item.agentId, "roomId" to item.roomId)
                    ))
                }
            }
        }
    }

    /**
     * Approve an attention item
     */
    fun onApproveAttentionItem(item: AttentionItem) {
        viewModelScope.launch {
            logger.logUserAction("onApproveAttentionItem", mapOf("id" to item.id))
            controlPlaneStore.removeAttentionItem(item.id)
            if (item is AttentionItem.PiiRequest) {
                // Approve the PII request
                controlPlaneStore.removePiiRequest(item.piiRequest.requestId)
            }
        }
    }

    /**
     * Deny an attention item
     */
    fun onDenyAttentionItem(item: AttentionItem) {
        viewModelScope.launch {
            logger.logUserAction("onDenyAttentionItem", mapOf("id" to item.id))
            controlPlaneStore.removeAttentionItem(item.id)
            if (item is AttentionItem.PiiRequest) {
                controlPlaneStore.removePiiRequest(item.piiRequest.requestId)
            }
        }
    }

    /**
     * Get room name for a workflow
     */
    fun getRoomNameForWorkflow(roomId: String): String? {
        return _rooms.value.find { it.id == roomId }?.name
    }
}
