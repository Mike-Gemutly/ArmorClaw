package com.armorclaw.app.viewmodels

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.domain.usecase.LogoutUseCase
import com.armorclaw.shared.platform.bridge.*
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.viewModelLogger
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

// ============================================================================
// Server Connection ViewModel - NEW
// ============================================================================

/**
 * UI State for Server Connection screen
 */
data class ServerConnectionUiState(
    val currentHomeserver: String? = null,
    val currentBridgeUrl: String? = null,
    val connectionStatus: ConnectionStatus = ConnectionStatus.UNKNOWN,
    val serverVersion: String? = null,
    val lastConnected: Long? = null,
    val isDiscovering: Boolean = false,
    val isConnecting: Boolean = false,
    val discoveryProgress: DiscoveryProgress = DiscoveryProgress.Idle,
    val errorMessage: String? = null,
    val securityWarnings: List<SecurityWarning> = emptyList(),
    val fallbackOptions: List<FallbackOption> = emptyList(),
    val canRetry: Boolean = true,
    val isDemoMode: Boolean = false
)

/**
 * Connection status enum
 */
enum class ConnectionStatus {
    UNKNOWN,        // Never connected
    CONNECTED,      // Currently connected
    DISCONNECTED,   // Was connected, now disconnected
    ERROR,          // Connection error
    CONNECTING,     // Attempting to connect
    DISCOVERING     // Running discovery
}

/**
 * Discovery progress sealed class
 */
sealed class DiscoveryProgress {
    object Idle : DiscoveryProgress()
    data class Discovering(val step: String) : DiscoveryProgress()
    data class FoundServer(val serverInfo: DetectServerInfo) : DiscoveryProgress()
    data class Error(val message: String) : DiscoveryProgress()
}

/**
 * ViewModel for Server Connection screen
 *
 * Handles:
 * - Server discovery and re-discovery
 * - Connection status monitoring
 * - Manual server configuration
 * - Deep link/QR code activation
 */
class ServerConnectionViewModel(
    private val setupService: SetupService,
    private val rpcClient: BridgeRpcClient,
    private val appPreferences: AppPreferences
) : ViewModel() {

    private val logger = viewModelLogger("ServerConnectionViewModel", LogTag.ViewModel.Settings)

    private val _uiState = MutableStateFlow(ServerConnectionUiState())
    val uiState: StateFlow<ServerConnectionUiState> = _uiState.asStateFlow()

    init {
        logger.logInit()
        loadCurrentConfig()
    }

    /**
     * Load current server configuration from SetupService or SharedPreferences
     */
    private fun loadCurrentConfig() {
        val config = setupService.config.value
        // Use SharedPreferences as fallback if SetupService is empty
        val homeserver = if (config.homeserver.isEmpty()) {
            appPreferences.getHomeserver() ?: ""
        } else {
            config.homeserver
        }
        _uiState.update { state ->
            state.copy(
                currentHomeserver = homeserver,
                currentBridgeUrl = config.bridgeUrl,
                isDemoMode = config.isDemo
            )
        }
        checkConnectionStatus()
    }

    /**
     * Check current connection status
     */
    fun checkConnectionStatus() {
        viewModelScope.launch {
            _uiState.update { it.copy(isDiscovering = true) }

            try {
                when (val result = rpcClient.healthCheck()) {
                    is RpcResult.Success -> {
                        val health = result.data
                        val version = health["version"] as? String
                        _uiState.update { state ->
                            state.copy(
                                connectionStatus = ConnectionStatus.CONNECTED,
                                serverVersion = version,
                                isDiscovering = false,
                                errorMessage = null,
                                lastConnected = System.currentTimeMillis()
                            )
                        }
                        logger.logInfo("Health check successful", mapOf("version" to (version ?: "unknown")))
                    }
                    is RpcResult.Error -> {
                        _uiState.update { state ->
                            state.copy(
                                connectionStatus = ConnectionStatus.DISCONNECTED,
                                isDiscovering = false,
                                errorMessage = result.message
                            )
                        }
                        logger.logInfo("Health check failed: ${result.message}")
                    }
                }
            } catch (e: Exception) {
                _uiState.update { state ->
                    state.copy(
                        connectionStatus = ConnectionStatus.ERROR,
                        isDiscovering = false,
                        errorMessage = e.message ?: "Connection failed"
                    )
                }
                logger.logError("checkConnectionStatus", e)
            }
        }
    }

    /**
     * Start discovery process
     */
    fun startDiscovery(homeserver: String? = null) {
        viewModelScope.launch {
            val server = homeserver ?: _uiState.value.currentHomeserver ?: "armorclaw.app"

            _uiState.update { state ->
                state.copy(
                    isDiscovering = true,
                    connectionStatus = ConnectionStatus.DISCOVERING,
                    discoveryProgress = DiscoveryProgress.Discovering("Checking server availability..."),
                    errorMessage = null,
                    securityWarnings = emptyList()
                )
            }

            logger.logInfo("Starting discovery", mapOf("homeserver" to server))

            when (val result = setupService.startSetupWithDiscovery(server)) {
                is SetupResult.Success -> {
                    // Persist discovered server to SharedPreferences
                    appPreferences.setHomeserver(result.info.homeserver)
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.CONNECTED,
                            discoveryProgress = DiscoveryProgress.FoundServer(result.info),
                            currentHomeserver = result.info.homeserver,
                            currentBridgeUrl = result.info.bridgeUrl,
                            serverVersion = result.info.version,
                            isDemoMode = result.info.isDemo,
                            lastConnected = System.currentTimeMillis()
                        )
                    }
                    logger.logInfo("Discovery successful", mapOf("server" to result.info.homeserver))
                }
                is SetupResult.Error -> {
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.ERROR,
                            discoveryProgress = DiscoveryProgress.Error(result.message),
                            errorMessage = result.message,
                            fallbackOptions = result.fallbackOptions
                        )
                    }
                    logger.logInfo("Discovery failed", mapOf("error" to result.message))
                }
            }
        }
    }

    /**
     * Try fallback server
     */
    fun tryFallbackServer(fallbackUrl: String) {
        viewModelScope.launch {
            _uiState.update { state ->
                state.copy(
                    isDiscovering = true,
                    discoveryProgress = DiscoveryProgress.Discovering("Trying fallback: $fallbackUrl")
                )
            }

            when (val result = setupService.startSetup(
                homeserver = _uiState.value.currentHomeserver ?: "https://matrix.armorclaw.app",
                bridgeUrl = fallbackUrl
            )) {
                is SetupResult.Success -> {
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.CONNECTED,
                            discoveryProgress = DiscoveryProgress.FoundServer(result.info),
                            currentBridgeUrl = result.info.bridgeUrl,
                            errorMessage = null
                        )
                    }
                }
                is SetupResult.Error -> {
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.ERROR,
                            errorMessage = result.message
                        )
                    }
                }
            }
        }
    }

    /**
     * Use demo server
     */
    fun useDemoServer() {
        viewModelScope.launch {
            _uiState.update { state ->
                state.copy(
                    isDiscovering = true,
                    discoveryProgress = DiscoveryProgress.Discovering("Connecting to demo server...")
                )
            }

            when (val result = setupService.useDemoServer()) {
                is SetupResult.Success -> {
                    // Persist demo server to SharedPreferences
                    appPreferences.setHomeserver(result.info.homeserver)
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.CONNECTED,
                            discoveryProgress = DiscoveryProgress.FoundServer(result.info),
                            currentHomeserver = result.info.homeserver,
                            currentBridgeUrl = result.info.bridgeUrl,
                            isDemoMode = true,
                            errorMessage = null
                        )
                    }
                    logger.logInfo("Connected to demo server")
                }
                is SetupResult.Error -> {
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.ERROR,
                            errorMessage = result.message
                        )
                    }
                }
            }
        }
    }

    /**
     * Update server configuration manually
     */
    fun updateServerConfig(homeserver: String, bridgeUrl: String?) {
        viewModelScope.launch {
            _uiState.update { state ->
                state.copy(
                    isDiscovering = true,
                    discoveryProgress = DiscoveryProgress.Discovering("Validating server...")
                )
            }

            when (val result = setupService.startSetup(homeserver, bridgeUrl)) {
                is SetupResult.Success -> {
                    // Config is stored in SetupService AND persisted to SharedPreferences
                    appPreferences.setHomeserver(result.info.homeserver)
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.CONNECTED,
                            discoveryProgress = DiscoveryProgress.FoundServer(result.info),
                            currentHomeserver = result.info.homeserver,
                            currentBridgeUrl = result.info.bridgeUrl,
                            isDemoMode = false,
                            errorMessage = null
                        )
                    }
                    logger.logInfo("Server config updated", mapOf("homeserver" to result.info.homeserver))
                }
                is SetupResult.Error -> {
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.ERROR,
                            errorMessage = result.message,
                            fallbackOptions = result.fallbackOptions
                        )
                    }
                }
            }
        }
    }

    /**
     * Process deep link or QR code
     */
    fun processDeepLink(deepLink: String) {
        viewModelScope.launch {
            _uiState.update { state ->
                state.copy(
                    isDiscovering = true,
                    discoveryProgress = DiscoveryProgress.Discovering("Processing signed configuration...")
                )
            }

            when (val result = setupService.parseSignedConfig(deepLink)) {
                is SetupResult.Success -> {
                    // Config is stored in SetupService
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.CONNECTED,
                            discoveryProgress = DiscoveryProgress.FoundServer(result.info),
                            currentHomeserver = result.info.homeserver,
                            currentBridgeUrl = result.info.bridgeUrl,
                            isDemoMode = false,
                            errorMessage = null
                        )
                    }
                    logger.logInfo("Deep link processed successfully")
                }
                is SetupResult.Error -> {
                    _uiState.update { state ->
                        state.copy(
                            isDiscovering = false,
                            connectionStatus = ConnectionStatus.ERROR,
                            errorMessage = result.message
                        )
                    }
                }
            }
        }
    }

    /**
     * Dismiss security warning
     */
    fun dismissWarning(warningId: String) {
        setupService.dismissWarning(warningId)
    }

    /**
     * Clear error
     */
    fun clearError() {
        _uiState.update { it.copy(errorMessage = null) }
    }

    /**
     * Reset discovery state
     */
    fun resetDiscovery() {
        setupService.resetSetup()
        _uiState.update { ServerConnectionUiState() }
        loadCurrentConfig()
    }
}

/**
 * ViewModel for Settings screen
 *
 * Handles logout and other settings-related operations.
 * Uses ViewModelLogger for proper separation of concerns in logging.
 */
class SettingsViewModel(
    private val logoutUseCase: LogoutUseCase
) : ViewModel() {

    private val logger = viewModelLogger("SettingsViewModel", LogTag.ViewModel.Settings)

    private val _uiState = MutableStateFlow<SettingsUiState>(SettingsUiState.Idle)
    val uiState: StateFlow<SettingsUiState> = _uiState.asStateFlow()

    private val _isLoggingOut = MutableStateFlow(false)
    val isLoggingOut: StateFlow<Boolean> = _isLoggingOut.asStateFlow()

    init {
        logger.logInit()
    }

    /**
     * Log out the current user
     *
     * Clears session, tokens, and local data.
     */
    fun logout() {
        if (_isLoggingOut.value) {
            logger.logUserAction("logout", mapOf("skipped" to "already_in_progress"))
            return
        }

        logger.logUserAction("logout")
        viewModelScope.launch {
            _isLoggingOut.value = true
            logger.logStateChange("uiState", "LoggingOut")
            _uiState.value = SettingsUiState.LoggingOut

            val result = logoutUseCase(clearAllData = true)

            result.fold(
                onSuccess = {
                    logger.logStateChange("uiState", "LogoutSuccess")
                    logger.logNavigation("login")
                    _uiState.value = SettingsUiState.LogoutSuccess
                },
                onFailure = { error ->
                    logger.logError("logout", error)
                    logger.logStateChange("uiState", "LogoutError")
                    _uiState.value = SettingsUiState.LogoutError(
                        error.message ?: "Failed to log out"
                    )
                }
            )

            _isLoggingOut.value = false
        }
    }

    /**
     * Reset UI state after handling logout result
     */
    fun resetState() {
        logger.logUserAction("resetState")
        logger.logStateChange("uiState", "Idle")
        _uiState.value = SettingsUiState.Idle
    }
}

/**
 * UI state for Settings screen
 */
sealed class SettingsUiState {
    object Idle : SettingsUiState()
    object LoggingOut : SettingsUiState()
    object LogoutSuccess : SettingsUiState()
    data class LogoutError(val message: String) : SettingsUiState()
}

// ============================================================================
// Agent Management ViewModel - NEW
// ============================================================================

/**
 * ViewModel for Agent Management screen
 *
 * Handles listing, monitoring, and stopping AI agents.
 */
class AgentManagementViewModel(
    private val bridgeRpcClient: BridgeRpcClient
) : ViewModel() {

    private val logger = viewModelLogger("AgentManagementViewModel", LogTag.ViewModel.Settings)

    private val _uiState = MutableStateFlow<AgentManagementUiState>(AgentManagementUiState.Loading)
    val uiState: StateFlow<AgentManagementUiState> = _uiState.asStateFlow()

    private val _agents = MutableStateFlow<List<AgentInfo>>(emptyList())
    val agents: StateFlow<List<AgentInfo>> = _agents.asStateFlow()

    init {
        logger.logInit()
        loadAgents()
    }

    fun loadAgents() {
        logger.logUserAction("loadAgents")
        viewModelScope.launch {
            _uiState.value = AgentManagementUiState.Loading

            when (val result = bridgeRpcClient.agentList()) {
                is RpcResult.Success -> {
                    _agents.value = result.data.agents
                    _uiState.value = AgentManagementUiState.Success(result.data.count)
                    logger.logInfo("Loaded ${result.data.count} agents")
                }
                is RpcResult.Error -> {
                    _uiState.value = AgentManagementUiState.Error(
                        result.message ?: "Failed to load agents"
                    )
                    logger.logError("loadAgents", Exception(result.message))
                }
            }
        }
    }

    fun stopAgent(agentId: String) {
        logger.logUserAction("stopAgent", mapOf("agent_id" to agentId))
        viewModelScope.launch {
            _uiState.value = AgentManagementUiState.StoppingAgent(agentId)

            when (val result = bridgeRpcClient.agentStop(agentId)) {
                is RpcResult.Success -> {
                    if (result.data) {
                        _agents.value = _agents.value.filter { it.agentId != agentId }
                        _uiState.value = AgentManagementUiState.AgentStopped(agentId)
                        logger.logInfo("Agent stopped: $agentId")
                    } else {
                        _uiState.value = AgentManagementUiState.Error("Failed to stop agent")
                    }
                }
                is RpcResult.Error -> {
                    _uiState.value = AgentManagementUiState.Error(
                        result.message ?: "Failed to stop agent"
                    )
                    logger.logError("stopAgent", Exception(result.message))
                }
            }
        }
    }

    fun refreshAgentStatus(agentId: String) {
        logger.logUserAction("refreshAgentStatus", mapOf("agent_id" to agentId))
        viewModelScope.launch {
            when (val result = bridgeRpcClient.agentStatus(agentId)) {
                is RpcResult.Success -> {
                    // Update agent in list - convert enum status to string
                    _agents.value = _agents.value.map {
                        if (it.agentId == agentId) {
                            it.copy(status = result.data.status.name.lowercase())
                        } else it
                    }
                }
                is RpcResult.Error -> {
                    logger.logError("refreshAgentStatus", Exception(result.message))
                }
            }
        }
    }

    fun resetState() {
        _uiState.value = AgentManagementUiState.Success(_agents.value.size)
    }
}

sealed class AgentManagementUiState {
    object Loading : AgentManagementUiState()
    data class Success(val count: Int) : AgentManagementUiState()
    data class Error(val message: String) : AgentManagementUiState()
    data class StoppingAgent(val agentId: String) : AgentManagementUiState()
    data class AgentStopped(val agentId: String) : AgentManagementUiState()
}

// ============================================================================
// HITL (Human-in-the-Loop) ViewModel - NEW
// ============================================================================

/**
 * ViewModel for HITL Approval screen
 */
class HitlViewModel(
    private val bridgeRpcClient: BridgeRpcClient
) : ViewModel() {

    private val logger = viewModelLogger("HitlViewModel", LogTag.ViewModel.Settings)

    private val _uiState = MutableStateFlow<HitlUiState>(HitlUiState.Loading)
    val uiState: StateFlow<HitlUiState> = _uiState.asStateFlow()

    private val _pendingApprovals = MutableStateFlow<List<HitlApproval>>(emptyList())
    val pendingApprovals: StateFlow<List<HitlApproval>> = _pendingApprovals.asStateFlow()

    init {
        logger.logInit()
        loadPendingApprovals()
    }

    fun loadPendingApprovals() {
        logger.logUserAction("loadPendingApprovals")
        viewModelScope.launch {
            _uiState.value = HitlUiState.Loading

            when (val result = bridgeRpcClient.hitlPending()) {
                is RpcResult.Success -> {
                    _pendingApprovals.value = result.data.approvals
                    _uiState.value = HitlUiState.Success(result.data.count)
                    logger.logInfo("Loaded ${result.data.count} pending approvals")
                }
                is RpcResult.Error -> {
                    _uiState.value = HitlUiState.Error(
                        result.message ?: "Failed to load approvals"
                    )
                    logger.logError("loadPendingApprovals", Exception(result.message))
                }
            }
        }
    }

    fun approve(gateId: String, notes: String? = null) {
        logger.logUserAction("approve", mapOf("gate_id" to gateId))
        viewModelScope.launch {
            _uiState.value = HitlUiState.Processing(gateId)

            when (val result = bridgeRpcClient.hitlApprove(gateId, notes)) {
                is RpcResult.Success -> {
                    if (result.data) {
                        _pendingApprovals.value = _pendingApprovals.value.filter { it.gateId != gateId }
                        _uiState.value = HitlUiState.Approved(gateId)
                        logger.logInfo("Approved: $gateId")
                    } else {
                        _uiState.value = HitlUiState.Error("Failed to approve")
                    }
                }
                is RpcResult.Error -> {
                    _uiState.value = HitlUiState.Error(
                        result.message ?: "Failed to approve"
                    )
                    logger.logError("approve", Exception(result.message))
                }
            }
        }
    }

    fun reject(gateId: String, reason: String? = null) {
        logger.logUserAction("reject", mapOf("gate_id" to gateId))
        viewModelScope.launch {
            _uiState.value = HitlUiState.Processing(gateId)

            when (val result = bridgeRpcClient.hitlReject(gateId, reason)) {
                is RpcResult.Success -> {
                    if (result.data) {
                        _pendingApprovals.value = _pendingApprovals.value.filter { it.gateId != gateId }
                        _uiState.value = HitlUiState.Rejected(gateId)
                        logger.logInfo("Rejected: $gateId")
                    } else {
                        _uiState.value = HitlUiState.Error("Failed to reject")
                    }
                }
                is RpcResult.Error -> {
                    _uiState.value = HitlUiState.Error(
                        result.message ?: "Failed to reject"
                    )
                    logger.logError("reject", Exception(result.message))
                }
            }
        }
    }

    fun resetState() {
        _uiState.value = HitlUiState.Success(_pendingApprovals.value.size)
    }
}

sealed class HitlUiState {
    object Loading : HitlUiState()
    data class Success(val count: Int) : HitlUiState()
    data class Error(val message: String) : HitlUiState()
    data class Processing(val gateId: String) : HitlUiState()
    data class Approved(val gateId: String) : HitlUiState()
    data class Rejected(val gateId: String) : HitlUiState()
}

// ============================================================================
// Workflow ViewModel - NEW
// ============================================================================

/**
 * ViewModel for Workflow Management screen
 */
class WorkflowViewModel(
    private val bridgeRpcClient: BridgeRpcClient
) : ViewModel() {

    private val logger = viewModelLogger("WorkflowViewModel", LogTag.ViewModel.Settings)

    private val _uiState = MutableStateFlow<WorkflowUiState>(WorkflowUiState.Loading)
    val uiState: StateFlow<WorkflowUiState> = _uiState.asStateFlow()

    private val _templates = MutableStateFlow<List<WorkflowTemplate>>(emptyList())
    val templates: StateFlow<List<WorkflowTemplate>> = _templates.asStateFlow()

    init {
        logger.logInit()
        loadTemplates()
    }

    fun loadTemplates() {
        logger.logUserAction("loadTemplates")
        viewModelScope.launch {
            _uiState.value = WorkflowUiState.Loading

            when (val result = bridgeRpcClient.workflowTemplates()) {
                is RpcResult.Success -> {
                    _templates.value = result.data.templates
                    _uiState.value = WorkflowUiState.TemplatesLoaded(result.data.count)
                    logger.logInfo("Loaded ${result.data.count} workflow templates")
                }
                is RpcResult.Error -> {
                    _uiState.value = WorkflowUiState.Error(
                        result.message ?: "Failed to load templates"
                    )
                    logger.logError("loadTemplates", Exception(result.message))
                }
            }
        }
    }

    fun startWorkflow(
        templateId: String,
        params: Map<String, Any?> = emptyMap(),
        roomId: String? = null
    ) {
        logger.logUserAction("startWorkflow", mapOf("template_id" to templateId))
        viewModelScope.launch {
            _uiState.value = WorkflowUiState.Starting(templateId)

            when (val result = bridgeRpcClient.workflowStart(templateId, params, roomId)) {
                is RpcResult.Success -> {
                    _uiState.value = WorkflowUiState.Started(
                        workflowId = result.data.workflowId,
                        templateId = result.data.templateId
                    )
                    logger.logInfo("Started workflow: ${result.data.workflowId}")
                }
                is RpcResult.Error -> {
                    _uiState.value = WorkflowUiState.Error(
                        result.message ?: "Failed to start workflow"
                    )
                    logger.logError("startWorkflow", Exception(result.message))
                }
            }
        }
    }

    fun checkStatus(workflowId: String) {
        logger.logUserAction("checkStatus", mapOf("workflow_id" to workflowId))
        viewModelScope.launch {
            when (val result = bridgeRpcClient.workflowStatus(workflowId)) {
                is RpcResult.Success -> {
                    _uiState.value = WorkflowUiState.StatusLoaded(result.data)
                }
                is RpcResult.Error -> {
                    logger.logError("checkStatus", Exception(result.message))
                }
            }
        }
    }

    fun resetState() {
        _uiState.value = WorkflowUiState.TemplatesLoaded(_templates.value.size)
    }
}

sealed class WorkflowUiState {
    object Loading : WorkflowUiState()
    data class TemplatesLoaded(val count: Int) : WorkflowUiState()
    data class Starting(val templateId: String) : WorkflowUiState()
    data class Started(val workflowId: String, val templateId: String) : WorkflowUiState()
    data class StatusLoaded(val status: WorkflowStatusResponse) : WorkflowUiState()
    data class Error(val message: String) : WorkflowUiState()
}
