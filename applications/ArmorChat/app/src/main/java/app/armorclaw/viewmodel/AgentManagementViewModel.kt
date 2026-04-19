package app.armorclaw.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.armorclaw.network.BridgeApi
import app.armorclaw.utils.BridgeError
import app.armorclaw.utils.ErrorHandler
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class AgentManagementViewModel(
    private val api: BridgeApi = BridgeApi()
) : ViewModel() {

    private val _state = MutableStateFlow(AgentManagementState())
    val state: StateFlow<AgentManagementState> = _state.asStateFlow()

    fun loadAgents() {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, errorMessage = null)
            try {
                val result = api.listAgents().getOrThrow()
                _state.value = _state.value.copy(
                    agentList = result.agents,
                    isLoading = false
                )
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _state.value = _state.value.copy(
                    isLoading = false,
                    errorMessage = error.message
                )
            }
        }
    }

    fun selectAgent(agentId: String) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, errorMessage = null)
            try {
                val agent = api.getAgent(agentId).getOrThrow()
                val instances = api.listInstances(agentId = agentId).getOrThrow()
                _state.value = _state.value.copy(
                    selectedAgent = agent,
                    instances = instances.instances,
                    isLoading = false
                )
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _state.value = _state.value.copy(
                    isLoading = false,
                    errorMessage = error.message
                )
            }
        }
    }

    fun createAgent(name: String, skills: List<String>, description: String = "") {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, errorMessage = null)
            try {
                api.createAgent(
                    name = name,
                    description = description,
                    skills = skills
                ).getOrThrow()
                loadAgents()
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _state.value = _state.value.copy(
                    isLoading = false,
                    errorMessage = error.message
                )
            }
        }
    }

    fun deleteAgent(agentId: String) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, errorMessage = null)
            try {
                api.deleteAgent(agentId).getOrThrow()
                val current = _state.value
                _state.value = current.copy(
                    agentList = current.agentList.filter { it.id != agentId },
                    selectedAgent = if (current.selectedAgent?.id == agentId) null else current.selectedAgent,
                    instances = if (current.selectedAgent?.id == agentId) emptyList() else current.instances,
                    isLoading = false
                )
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _state.value = _state.value.copy(
                    isLoading = false,
                    errorMessage = error.message
                )
            }
        }
    }

    fun refreshInstances(agentId: String = "") {
        viewModelScope.launch {
            try {
                val result = api.listInstances(agentId = agentId).getOrThrow()
                _state.value = _state.value.copy(instances = result.instances)
            } catch (e: Throwable) {
                val error = if (e is BridgeError) e else ErrorHandler.mapError(e)
                _state.value = _state.value.copy(errorMessage = error.message)
            }
        }
    }

    fun clearError() {
        _state.value = _state.value.copy(errorMessage = null)
    }

    fun clearSelection() {
        _state.value = _state.value.copy(selectedAgent = null, instances = emptyList())
    }
}

data class AgentManagementState(
    val agentList: List<BridgeApi.AgentDefinition> = emptyList(),
    val selectedAgent: BridgeApi.AgentDefinition? = null,
    val instances: List<Map<String, String>> = emptyList(),
    val isLoading: Boolean = false,
    val errorMessage: String? = null
)
