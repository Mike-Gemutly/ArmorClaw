package app.armorclaw.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import app.armorclaw.data.model.EmailApprovalEvent
import app.armorclaw.data.model.SystemAlertContent
import app.armorclaw.network.BridgeApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class HitlViewModel(
    private val bridgeApi: BridgeApi = BridgeApi()
) : ViewModel() {

    // ── Tab state ───────────────────────────────────────────────────────

    enum class ApprovalTab(val label: String) {
        PII("PII"),
        MCP("MCP"),
        EMAIL("Email")
    }

    private val _selectedTab = MutableStateFlow(ApprovalTab.PII)
    val selectedTab: StateFlow<ApprovalTab> = _selectedTab.asStateFlow()

    fun selectTab(tab: ApprovalTab) {
        _selectedTab.value = tab
    }

    // ── PII approvals ──────────────────────────────────────────────────

    private val _pendingPii = MutableStateFlow<List<SystemAlertContent>>(emptyList())
    val pendingPii: StateFlow<List<SystemAlertContent>> = _pendingPii.asStateFlow()

    // ── MCP approvals (represented as agent definitions pending deploy) ─

    private val _pendingMcp = MutableStateFlow<List<BridgeApi.AgentDefinition>>(emptyList())
    val pendingMcp: StateFlow<List<BridgeApi.AgentDefinition>> = _pendingMcp.asStateFlow()

    // ── Email approvals ────────────────────────────────────────────────

    private val _pendingEmails = MutableStateFlow<List<EmailApprovalEvent>>(emptyList())
    val pendingEmails: StateFlow<List<EmailApprovalEvent>> = _pendingEmails.asStateFlow()

    // ── Loading / error state ──────────────────────────────────────────

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()

    // ── Data loading ───────────────────────────────────────────────────

    init {
        loadAllPending()
    }

    fun loadAllPending() {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null

            // Load pending emails via BridgeApi
            val emailResult = bridgeApi.listPendingEmails()
            emailResult.onSuccess { response ->
                _pendingEmails.value = response.approvals.map { pending ->
                    EmailApprovalEvent(
                        approvalId = pending.approval_id,
                        emailId = pending.sender,
                        to = pending.to,
                        piiFields = 0,
                        timeoutS = 300
                    )
                }
            }.onFailure { e ->
                _error.value = "Failed to load email approvals: ${e.message}"
            }

            val agentResult = bridgeApi.listAgents(activeOnly = false)
            agentResult.onSuccess { response ->
                _pendingMcp.value = response.agents
            }.onFailure { e ->
                _error.value = "Failed to load MCP approvals: ${e.message}"
            }

            // PII approvals arrive from Matrix events via setPendingPiiAlerts(), not an RPC endpoint.

            _isLoading.value = false
        }
    }

    /**
     * Set PII alerts from Matrix event stream.
     * Called by the Matrix sync layer when PII_ACCESS_REQUEST alerts arrive.
     */
    fun setPendingPiiAlerts(alerts: List<SystemAlertContent>) {
        _pendingPii.value = alerts.filter {
            it.alertType.name == "PII_ACCESS_REQUEST"
        }
    }

    // ── PII actions ────────────────────────────────────────────────────

    fun approvePii(requestId: String, approvedFields: List<String>) {
        viewModelScope.launch {
            bridgeApi.approvePiiAccess(requestId, approvedFields)
                .onSuccess {
                    _pendingPii.value = _pendingPii.value.filterNot { alert ->
                        alert.metadata?.get("request_id") == requestId
                    }
                }
                .onFailure { e ->
                    _error.value = "PII approve failed: ${e.message}"
                }
        }
    }

    fun denyPii(requestId: String, reason: String) {
        viewModelScope.launch {
            bridgeApi.rejectPiiAccess(requestId, reason)
                .onSuccess {
                    _pendingPii.value = _pendingPii.value.filterNot { alert ->
                        alert.metadata?.get("request_id") == requestId
                    }
                }
                .onFailure { e ->
                    _error.value = "PII deny failed: ${e.message}"
                }
        }
    }

    // ── MCP actions ────────────────────────────────────────────────────

    fun approveMcp(agentId: String) {
        viewModelScope.launch {
            bridgeApi.listInstances(agentId = agentId)
                .onSuccess {
                    _pendingMcp.value = _pendingMcp.value.filterNot { it.id == agentId }
                }
                .onFailure { e ->
                    _error.value = "MCP approve failed: ${e.message}"
                }
        }
    }

    fun rejectMcp(agentId: String) {
        viewModelScope.launch {
            bridgeApi.deleteAgent(agentId)
                .onSuccess {
                    _pendingMcp.value = _pendingMcp.value.filterNot { it.id == agentId }
                }
                .onFailure { e ->
                    _error.value = "MCP reject failed: ${e.message}"
                }
        }
    }

    // ── Email actions ──────────────────────────────────────────────────

    fun approveEmail(approvalId: String, userId: String = "admin") {
        viewModelScope.launch {
            bridgeApi.approveEmail(approvalId, userId)
                .onSuccess {
                    _pendingEmails.value = _pendingEmails.value.filterNot { it.approvalId == approvalId }
                }
                .onFailure { e ->
                    _error.value = "Email approve failed: ${e.message}"
                }
        }
    }

    fun denyEmail(approvalId: String, userId: String = "admin", reason: String = "User denied email send") {
        viewModelScope.launch {
            bridgeApi.denyEmail(approvalId, userId, reason)
                .onSuccess {
                    _pendingEmails.value = _pendingEmails.value.filterNot { it.approvalId == approvalId }
                }
                .onFailure { e ->
                    _error.value = "Email deny failed: ${e.message}"
                }
        }
    }

    fun clearError() {
        _error.value = null
    }

    // ── Factory ────────────────────────────────────────────────────────

    class Factory(
        private val bridgeApi: BridgeApi = BridgeApi()
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            return HitlViewModel(bridgeApi) as T
        }
    }
}
