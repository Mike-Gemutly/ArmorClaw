package app.armorclaw.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.armorclaw.network.BridgeApi
import app.armorclaw.ui.components.BlockerDialogState
import app.armorclaw.ui.components.BlockerInfo
import app.armorclaw.ui.components.WorkflowEvent
import app.armorclaw.ui.components.WorkflowTimelineState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import org.json.JSONArray

data class TemplateInfo(
    val id: String,
    val name: String,
    val description: String
)

data class WorkflowInfo(
    val workflowId: String,
    val status: String,
    val name: String,
    val startedAt: String
)

data class WorkflowDetail(
    val workflowId: String,
    val status: String,
    val name: String,
    val progress: Float
)

sealed class WorkflowUiState {
    object Loading : WorkflowUiState()
    data class Loaded(
        val templates: List<TemplateInfo>,
        val workflows: List<WorkflowInfo>
    ) : WorkflowUiState()
    data class Error(val message: String) : WorkflowUiState()
}

class WorkflowViewModel(
    private val bridgeApi: BridgeApi = BridgeApi()
) : ViewModel() {

    private val _uiState = MutableStateFlow<WorkflowUiState>(WorkflowUiState.Loading)
    val uiState: StateFlow<WorkflowUiState> = _uiState.asStateFlow()

    private val _selectedWorkflow = MutableStateFlow<WorkflowDetail?>(null)
    val selectedWorkflow: StateFlow<WorkflowDetail?> = _selectedWorkflow.asStateFlow()

    private val _timelineState = MutableStateFlow(
        WorkflowTimelineState(events = emptyList(), progress = 0f, isRunning = false)
    )
    val timelineState: StateFlow<WorkflowTimelineState> = _timelineState.asStateFlow()

    private val _activeBlocker = MutableStateFlow<BlockerInfo?>(null)
    val activeBlocker: StateFlow<BlockerInfo?> = _activeBlocker.asStateFlow()

    private val _blockerDialogState = MutableStateFlow(BlockerDialogState.DISMISSED)
    val blockerDialogState: StateFlow<BlockerDialogState> = _blockerDialogState.asStateFlow()

    private val _blockerError = MutableStateFlow("")
    val blockerError: StateFlow<String> = _blockerError.asStateFlow()

    private val _operationLoading = MutableStateFlow(false)
    val operationLoading: StateFlow<Boolean> = _operationLoading.asStateFlow()

    fun loadWorkflows() {
        viewModelScope.launch {
            _uiState.value = WorkflowUiState.Loading
            try {
                val templatesResult = bridgeApi.listTemplates(activeOnly = true)
                val instancesResult = bridgeApi.listInstances()

                if (templatesResult.isFailure && instancesResult.isFailure) {
                    _uiState.value = WorkflowUiState.Error(
                        templatesResult.exceptionOrNull()?.message ?: "Failed to load workflows"
                    )
                    return@launch
                }

                val templates = templatesResult.getOrNull()?.templates?.map { map ->
                    TemplateInfo(
                        id = map["id"] ?: "",
                        name = map["name"] ?: "Untitled",
                        description = map["description"] ?: ""
                    )
                } ?: emptyList()

                val workflows = instancesResult.getOrNull()?.instances?.map { map ->
                    WorkflowInfo(
                        workflowId = map["workflow_id"] ?: map["id"] ?: "",
                        status = map["status"] ?: "unknown",
                        name = map["name"] ?: "Workflow",
                        startedAt = map["started_at"] ?: map["created_at"] ?: ""
                    )
                } ?: emptyList()

                _uiState.value = WorkflowUiState.Loaded(templates, workflows)
            } catch (e: Exception) {
                _uiState.value = WorkflowUiState.Error(e.message ?: "Unknown error")
            }
        }
    }

    fun startWorkflow(templateId: String) {
        viewModelScope.launch {
            _operationLoading.value = true
            try {
                val result = bridgeApi.startWorkflow(templateId)
                if (result.isSuccess) {
                    val response = result.getOrThrow()
                    loadWorkflowDetail(response.workflow_id)
                    loadWorkflows()
                } else {
                    _blockerError.value = result.exceptionOrNull()?.message ?: "Failed to start workflow"
                }
            } catch (e: Exception) {
                _blockerError.value = e.message ?: "Unknown error"
            } finally {
                _operationLoading.value = false
            }
        }
    }

    fun selectWorkflow(workflowId: String) {
        loadWorkflowDetail(workflowId)
    }

    private fun loadWorkflowDetail(workflowId: String) {
        viewModelScope.launch {
            _operationLoading.value = true
            try {
                val result = bridgeApi.getWorkflow(workflowId)
                if (result.isSuccess) {
                    val data = result.getOrThrow()
                    val status = data["status"] ?: "unknown"
                    val name = data["name"] ?: "Workflow"
                    val progressRaw = data["progress"]?.toFloatOrNull() ?: 0f
                    val progress = (progressRaw / 100f).coerceIn(0f, 1f)

                    _selectedWorkflow.value = WorkflowDetail(
                        workflowId = workflowId,
                        status = status,
                        name = name,
                        progress = progress
                    )

                    val eventsJson = data["timeline"]
                    val events = if (!eventsJson.isNullOrEmpty()) {
                        try {
                            val jsonArray = JSONArray(eventsJson)
                            WorkflowEvent.fromTimelineEventArray(jsonArray)
                        } catch (_: Exception) {
                            emptyList()
                        }
                    } else emptyList()

                    _timelineState.value = WorkflowTimelineState(
                        events = events,
                        progress = progress,
                        isRunning = status == "running",
                        workflowName = name
                    )

                    if (status == "blocked") {
                        _activeBlocker.value = BlockerInfo(
                            blockerType = data["blocker_type"] ?: "unknown",
                            message = data["blocker_message"] ?: "Workflow requires input",
                            suggestion = data["blocker_suggestion"] ?: "",
                            field = data["blocker_field"] ?: "",
                            workflowId = workflowId,
                            stepId = data["blocker_step_id"] ?: ""
                        )
                        _blockerDialogState.value = BlockerDialogState.INPUT
                    }
                }
            } catch (_: Exception) {
                // Detail loading failure — user stays on list
            } finally {
                _operationLoading.value = false
            }
        }
    }

    fun cancelWorkflow(workflowId: String) {
        viewModelScope.launch {
            _operationLoading.value = true
            try {
                bridgeApi.cancelWorkflow(workflowId)
                clearSelection()
                loadWorkflows()
            } catch (_: Exception) {
                // List refresh will show updated state
            } finally {
                _operationLoading.value = false
            }
        }
    }

    fun resolveBlocker(workflowId: String, stepId: String, input: String, note: String) {
        viewModelScope.launch {
            _blockerDialogState.value = BlockerDialogState.LOADING
            try {
                val result = bridgeApi.resolveBlocker(workflowId, stepId, input, note)
                if (result.isSuccess) {
                    _blockerDialogState.value = BlockerDialogState.DISMISSED
                    _activeBlocker.value = null
                    _blockerError.value = ""
                    loadWorkflowDetail(workflowId)
                } else {
                    _blockerError.value = result.exceptionOrNull()?.message ?: "Failed to resolve"
                    _blockerDialogState.value = BlockerDialogState.ERROR
                }
            } catch (e: Exception) {
                _blockerError.value = e.message ?: "Unknown error"
                _blockerDialogState.value = BlockerDialogState.ERROR
            }
        }
    }

    fun dismissBlocker() {
        _blockerDialogState.value = BlockerDialogState.DISMISSED
        _activeBlocker.value = null
        _blockerError.value = ""
    }

    fun clearSelection() {
        _selectedWorkflow.value = null
        _timelineState.value = WorkflowTimelineState(
            events = emptyList(), progress = 0f, isRunning = false
        )
    }
}
