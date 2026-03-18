package com.armorclaw.app.viewmodels

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.platform.bridge.*
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.coroutines.Dispatchers

/**
 * ViewModel for managing the initial ArmorClaw setup flow
 *
 * Handles:
 * - Server detection and validation
 * - Bridge health checking (gates credential entry) - CRITICAL: must pass before credential screen
 * - Credential authentication
 * - Admin privilege detection
 * - Security warnings display
 * - Fallback server connections
 *
 * CTO Note (2026-02-23): Health gating is now mandatory before credential entry.
 * If bridge.health RPC fails or returns non-healthy, user sees actionable error banner
 * instead of proceeding to credential form.
 */
class SetupViewModel(
    private val setupService: SetupService,
    private val rpcClient: BridgeRpcClient
) : ViewModel() {

    // UI State
    private val _uiState = MutableStateFlow(SetupUiState())
    val uiState: StateFlow<SetupUiState> = _uiState.asStateFlow()

    // Setup progress
    val setupState: StateFlow<SetupState> = setupService.setupState

    // Security warnings
    val securityWarnings: StateFlow<List<SecurityWarning>> = setupService.securityWarnings

    // Server info
    val serverInfo: StateFlow<DetectServerInfo?> = setupService.serverInfo

    init {
        // Observe setup state changes
        viewModelScope.launch {
            setupService.setupState.collect { state ->
                _uiState.update { current ->
                    current.copy(
                        setupStep = when (state) {
                            is SetupState.Idle -> SetupStep.IDLE
                            is SetupState.Discovering -> SetupStep.DETECTING_SERVER
                            is SetupState.Discovered -> SetupStep.READY
                            is SetupState.IncompleteDiscovery -> SetupStep.DETECTING_SERVER
                            is SetupState.FallbackAttempt -> SetupStep.FALLBACK
                            is SetupState.ReadyForCredentials -> SetupStep.READY
                            is SetupState.Connecting -> SetupStep.CONNECTING
                            is SetupState.StartingBridge -> SetupStep.STARTING_BRIDGE
                            is SetupState.Authenticating -> SetupStep.AUTHENTICATING
                            is SetupState.ConnectingWebSocket -> SetupStep.CONNECTING_WEBSOCKET
                            is SetupState.ClaimingAdmin -> SetupStep.CONNECTING
                            is SetupState.ProvisioningExpired -> SetupStep.ERROR
                            is SetupState.AlreadyClaimed -> SetupStep.ERROR
                            is SetupState.CheckingPrivileges -> SetupStep.CHECKING_PRIVILEGES
                            is SetupState.Completed -> SetupStep.COMPLETED
                            is SetupState.Error -> SetupStep.ERROR
                        },
                        isConnecting = state.isConnecting,
                        isCompleted = state.isCompleted,
                        hasError = state.hasError,
                        errorMessage = if (state is SetupState.Error) state.message else null,
                        fallbackServerUrl = if (state is SetupState.FallbackAttempt) state.serverUrl else null,
                        completedInfo = if (state is SetupState.Completed) state.info else null
                    )
                }
            }
        }

        // Observe security warnings
        viewModelScope.launch {
            setupService.securityWarnings.collect { warnings ->
                _uiState.update { it.copy(warnings = warnings) }
            }
        }

        // Observe server info
        viewModelScope.launch {
            setupService.serverInfo.collect { info ->
                _uiState.update { it.copy(serverInfo = info) }
            }
        }
    }

    /**
     * Start setup with homeserver URL
     * 
     * CRITICAL: This now performs bridge health gating before allowing credential entry.
     * If bridge is unreachable, user sees actionable error banner instead of credential form.
     */
    fun startSetup(homeserver: String, bridgeUrl: String? = null) {
        viewModelScope.launch {
            _uiState.update { it.copy(
                isConnecting = true, 
                errorMessage = null,
                bridgeHealthStatus = BridgeHealthStatus.CHECKING
            )}

            val context = OperationContext.create()
            
            // Step 1: Run basic setup detection
            when (val result = setupService.startSetup(homeserver, bridgeUrl, context)) {
                is SetupResult.Success -> {
                    val serverInfo = result.info
                    
                    // Step 2: CRITICAL - Gate on bridge health before allowing credential entry
                    val healthResult = performBridgeHealthCheck(serverInfo.bridgeUrl, context)
                    
                    if (healthResult.isHealthy) {
                        _uiState.update { it.copy(
                            isConnecting = false,
                            serverInfo = serverInfo,
                            canProceed = true,
                            isNewServer = healthResult.details?.isNewServer ?: false,
                            bridgeHealthStatus = BridgeHealthStatus.HEALTHY,
                            bridgeHealthMessage = null,
                            bridgeHealthDetails = healthResult.details
                        )}
                    } else {
                        // Health check failed - show actionable error banner
                        _uiState.update { it.copy(
                            isConnecting = false,
                            serverInfo = serverInfo,
                            canProceed = false,
                            isNewServer = healthResult.details?.isNewServer ?: false,
                            bridgeHealthStatus = healthResult.status,
                            bridgeHealthMessage = healthResult.message,
                            bridgeHealthDetails = healthResult.details
                        )}
                    }
                }
                is SetupResult.Error -> {
                    _uiState.update { it.copy(
                        isConnecting = false,
                        errorMessage = result.message,
                        fallbackOptions = result.fallbackOptions,
                        bridgeHealthStatus = BridgeHealthStatus.ERROR
                    )}
                }
            }
        }
    }

    /**
     * Perform explicit bridge health check
     * Returns detailed health status with actionable suggestions
     */
    private suspend fun performBridgeHealthCheck(
        bridgeUrl: String,
        context: OperationContext
    ): BridgeHealthCheckResult {
        return withContext(Dispatchers.IO) {
            try {
                when (val result = rpcClient.healthCheck(context)) {
                    is RpcResult.Success -> {
                        val data = result.data
                        // Bridge may return "healthy": true (JSON-RPC) or "status": "ok" (HTTP).
                        // Check both to handle either contract.
                        val isHealthy = data["healthy"] as? Boolean
                            ?: (data["status"] as? String)?.equals("ok", ignoreCase = true)
                            ?: true
                        val bridgeReady = data["bridge_ready"] as? Boolean ?: true
                        val isNewServer = data["is_new_server"] as? Boolean ?: false
                        val provisioningAvailable = data["provisioning_available"] as? Boolean
                        
                        if (isHealthy && bridgeReady) {
                            BridgeHealthCheckResult(
                                isHealthy = true,
                                status = BridgeHealthStatus.HEALTHY,
                                message = null,
                                details = BridgeHealthDetails(
                                    bridgeUrl = bridgeUrl,
                                    bridgeReady = true,
                                    isNewServer = isNewServer,
                                    provisioningAvailable = provisioningAvailable
                                )
                            )
                        } else if (isHealthy && !bridgeReady) {
                            // Bridge is healthy but still initializing
                            BridgeHealthCheckResult(
                                isHealthy = false,
                                status = BridgeHealthStatus.NOT_READY,
                                message = "Bridge is starting up. Please wait a moment and retry.",
                                details = BridgeHealthDetails(
                                    bridgeUrl = bridgeUrl,
                                    isIpOnlyServer = isIpOnlyUrl(bridgeUrl),
                                    bridgeReady = false,
                                    isNewServer = isNewServer,
                                    provisioningAvailable = provisioningAvailable,
                                    suggestedActions = listOf(
                                        "Wait a few seconds for the bridge to finish initializing",
                                        "Retry the health check"
                                    )
                                )
                            )
                        } else {
                            // Bridge reachable but reported unhealthy
                            val reason = data["reason"] as? String ?: "Bridge reported unhealthy status"
                            BridgeHealthCheckResult(
                                isHealthy = false,
                                status = BridgeHealthStatus.UNHEALTHY,
                                message = "Bridge is not healthy: $reason",
                                details = BridgeHealthDetails(
                                    bridgeUrl = bridgeUrl,
                                    isIpOnlyServer = isIpOnlyUrl(bridgeUrl),
                                    bridgeReady = bridgeReady,
                                    isNewServer = isNewServer,
                                    provisioningAvailable = provisioningAvailable,
                                    suggestedActions = listOf(
                                        "Check bridge server logs for errors",
                                        "Verify all bridge services are running",
                                        "Contact server administrator"
                                    )
                                )
                            )
                        }
                    }
                    is RpcResult.Error -> {
                        // Bridge not reachable - construct actionable error message
                        val isIpOnly = isIpOnlyUrl(bridgeUrl)
                        val suggestedActions = mutableListOf<String>()
                        
                        suggestedActions.add("Check that the server is running")
                        
                        if (isIpOnly) {
                            suggestedActions.add("Verify logs for 'socket error' or 'flag redefined' messages")
                            suggestedActions.add("Ensure port 8080 is open in firewall")
                            suggestedActions.add("Confirm /run/armorclaw/bridge.sock exists after startup")
                        } else {
                            suggestedActions.add("Verify DNS resolves correctly")
                            suggestedActions.add("Check firewall allows HTTPS traffic")
                        }
                        
                        suggestedActions.add("Try again in a few moments")
                        
                        val message = if (result.message.contains("socket", ignoreCase = true) ||
                                          result.message.contains("flag redefined", ignoreCase = true)) {
                            // Specific error for known bridge-side issue
                            "Bridge not responding – check logs for socket error or flag redefined. " +
                            "This typically means the bridge server needs the v8.1.1 hotfix."
                        } else if (result.message.contains("timeout", ignoreCase = true) ||
                                   result.message.contains("connection refused", ignoreCase = true)) {
                            "Bridge unreachable at $bridgeUrl"
                        } else {
                            "Bridge health check failed: ${result.message}"
                        }
                        
                        BridgeHealthCheckResult(
                            isHealthy = false,
                            status = BridgeHealthStatus.UNREACHABLE,
                            message = message,
                            details = BridgeHealthDetails(
                                bridgeUrl = bridgeUrl,
                                isIpOnlyServer = isIpOnly,
                                suggestedActions = suggestedActions,
                                suggestedPort = if (isIpOnly) 8080 else null
                            )
                        )
                    }
                }
            } catch (e: Exception) {
                BridgeHealthCheckResult(
                    isHealthy = false,
                    status = BridgeHealthStatus.ERROR,
                    message = "Failed to check bridge health: ${e.message}",
                    details = BridgeHealthDetails(
                        bridgeUrl = bridgeUrl,
                        isIpOnlyServer = isIpOnlyUrl(bridgeUrl),
                        suggestedActions = listOf(
                            "Check your network connection",
                            "Verify the server URL is correct",
                            "Contact server administrator"
                        )
                    )
                )
            }
        }
    }

    /**
     * Check if URL points to an IP-only server (no domain)
     */
    private fun isIpOnlyUrl(url: String): Boolean {
        val host = url
            .removePrefix("https://")
            .removePrefix("http://")
            .split("/").first()
            .split(":").first()
        return isIpAddress(host)
    }

    /**
     * Delegates to shared NetworkUtils to avoid code duplication (RC-05).
     */
    private fun isIpAddress(host: String): Boolean = com.armorclaw.shared.platform.network.NetworkUtils.isIpAddress(host)

    /**
     * Retry health check only (without full setup restart)
     */
    fun retryHealthCheck() {
        val serverInfo = _uiState.value.serverInfo ?: return
        viewModelScope.launch {
            _uiState.update { it.copy(bridgeHealthStatus = BridgeHealthStatus.CHECKING) }
            
            val context = OperationContext.create()
            val healthResult = performBridgeHealthCheck(serverInfo.bridgeUrl, context)
            
            _uiState.update { it.copy(
                bridgeHealthStatus = healthResult.status,
                bridgeHealthMessage = healthResult.message,
                bridgeHealthDetails = healthResult.details,
                canProceed = healthResult.isHealthy,
                isNewServer = healthResult.details?.isNewServer ?: it.isNewServer
            )}
        }
    }

    /**
     * Data class for health check result
     */
    private data class BridgeHealthCheckResult(
        val isHealthy: Boolean,
        val status: BridgeHealthStatus,
        val message: String?,
        val details: BridgeHealthDetails?
    )

    /**
     * Connect with user credentials
     * 
     * Prerequisites: startSetup() must have completed successfully with healthy bridge
     */
    fun connectWithCredentials(username: String, password: String, deviceId: String? = null) {
        viewModelScope.launch {
            // CRITICAL: Verify bridge health is still good before attempting credentials
            val currentHealth = _uiState.value.bridgeHealthStatus
            if (currentHealth != BridgeHealthStatus.HEALTHY && currentHealth != BridgeHealthStatus.UNKNOWN) {
                _uiState.update { it.copy(
                    isConnecting = false,
                    errorMessage = "Bridge is not healthy. Please retry the health check first."
                )}
                return@launch
            }

            _uiState.update { it.copy(isConnecting = true, errorMessage = null) }

            val resolvedDeviceId = deviceId ?: generateDeviceId()
            val context = OperationContext.create()

            when (val result = setupService.connectWithCredentials(
                username = username,
                password = password,
                deviceId = resolvedDeviceId,
                context = context
            )) {
                is SetupResult.Success -> {
                    _uiState.update { it.copy(
                        isConnecting = false,
                        serverInfo = result.info,
                        isAdmin = result.info.isAdmin,
                        adminLevel = result.info.adminLevel,
                        canProceed = true
                    )}
                }
                is SetupResult.Error -> {
                    _uiState.update { it.copy(
                        isConnecting = false,
                        errorMessage = result.message,
                        fallbackOptions = result.fallbackOptions
                    )}
                }
            }
        }
    }

    /**
     * Use demo server
     */
    fun useDemoServer() {
        viewModelScope.launch {
            _uiState.update { it.copy(isConnecting = true, errorMessage = null) }

            val context = OperationContext.create()
            when (val result = setupService.useDemoServer(context)) {
                is SetupResult.Success -> {
                    _uiState.update { it.copy(
                        isConnecting = false,
                        serverInfo = result.info,
                        isDemo = true,
                        canProceed = true
                    )}
                }
                is SetupResult.Error -> {
                    _uiState.update { it.copy(
                        isConnecting = false,
                        errorMessage = result.message,
                        fallbackOptions = result.fallbackOptions
                    )}
                }
            }
        }
    }

    /**
     * Retry setup after error
     */
    fun retry() {
        val currentState = _uiState.value
        if (currentState.serverInfo != null) {
            startSetup(
                homeserver = currentState.serverInfo.homeserver,
                bridgeUrl = currentState.serverInfo.bridgeUrl
            )
        }
    }

    /**
     * Reset setup to start over
     */
    fun resetSetup() {
        setupService.resetSetup()
        _uiState.value = SetupUiState()
    }

    /**
     * Dismiss a security warning
     */
    fun dismissWarning(warningId: String) {
        setupService.dismissWarning(warningId)
    }

    /**
     * Clear error message
     */
    fun clearError() {
        _uiState.update { it.copy(errorMessage = null) }
    }

    /**
     * Handle QR code provisioning
     *
     * This is the primary setup path for QR-first onboarding.
     * Parses the QR/deep link, extracts server config, and automatically
     * provisions the user.
     *
     * Supported formats:
     * - armorclaw://config?d=<base64-encoded-json>
     * - https://armorclaw.app/config?d=<base64-encoded-json>
     * - armorclaw://invite?code=<invite-code>
     */
    fun handleQrProvision(qrData: String) {
        println("🔧 SetupViewModel.handleQrProvision() called")
        println("   QR data: ${qrData.take(100)}...")

        viewModelScope.launch {
            _uiState.update { it.copy(
                isConnecting = true,
                errorMessage = null,
                bridgeHealthStatus = BridgeHealthStatus.CHECKING
            )}

            println("🔧 State updated: isConnecting=true, setupStep=${_uiState.value.setupStep}")

            val context = OperationContext.create()

            // Use SetupService's parseSignedConfig for QR/deep link handling
            println("🔧 Calling setupService.parseSignedConfig()...")
            when (val result = setupService.parseSignedConfig(qrData, context)) {
                is SetupResult.Success -> {
                    println("✅ parseSignedConfig SUCCESS")
                    val serverInfo = result.info
                    println("   Server info: homeserver=${serverInfo.homeserver}, bridgeUrl=${serverInfo.bridgeUrl}")

                    // Store configuration in BridgeConfig
                    BridgeConfig.setRuntimeConfig(
                        BridgeConfig.fromDiscoveredServer(
                            homeserver = serverInfo.homeserver,
                            bridgeUrl = serverInfo.bridgeUrl,
                            serverName = serverInfo.displayName
                        )
                    )

                    // CRITICAL: Gate on bridge health before allowing credential entry
                    println("🔧 Calling performBridgeHealthCheck()...")
                    val healthResult = performBridgeHealthCheck(serverInfo.bridgeUrl, context)
                    println("   Health check result: isHealthy=${healthResult.isHealthy}, status=${healthResult.status}")

                    if (healthResult.isHealthy) {
                        println("✅ Bridge is healthy - showing credential form")
                        _uiState.update { it.copy(
                            isConnecting = false,
                            serverInfo = serverInfo,
                            canProceed = true,
                            isNewServer = healthResult.details?.isNewServer ?: false,
                            bridgeHealthStatus = BridgeHealthStatus.HEALTHY,
                            bridgeHealthDetails = healthResult.details,
                            setupStep = SetupStep.READY
                        )}
                        println("✅ State updated: setupStep=READY, canProceed=true")
                    } else {
                        println("❌ Bridge health failed: ${healthResult.message}")
                        _uiState.update { it.copy(
                            isConnecting = false,
                            serverInfo = serverInfo,
                            canProceed = false,
                            isNewServer = healthResult.details?.isNewServer ?: false,
                            bridgeHealthStatus = healthResult.status,
                            bridgeHealthMessage = healthResult.message,
                            bridgeHealthDetails = healthResult.details,
                            setupStep = SetupStep.ERROR
                        )}
                        println("❌ State updated: setupStep=ERROR, errorMessage=${healthResult.message}")
                    }
                }
                is SetupResult.Error -> {
                    println("❌ parseSignedConfig ERROR: ${result.message}")
                    _uiState.update { it.copy(
                        isConnecting = false,
                        errorMessage = result.message,
                        fallbackOptions = result.fallbackOptions,
                        bridgeHealthStatus = BridgeHealthStatus.ERROR,
                        setupStep = SetupStep.ERROR
                    )}
                }
            }
        }
    }

    /**
     * Handle QR provisioning with auto-login
     *
     * For QR codes that include authentication tokens,
     * this method automatically logs the user in without
     * requiring credential entry.
     */
    fun handleQrProvisionWithAuth(qrData: String, autoLogin: Boolean = false) {
        viewModelScope.launch {
            _uiState.update { it.copy(
                isConnecting = true,
                errorMessage = null,
                bridgeHealthStatus = BridgeHealthStatus.CHECKING
            )}

            val context = OperationContext.create()

            when (val result = setupService.parseSignedConfig(qrData, context)) {
                is SetupResult.Success -> {
                    val serverInfo = result.info

                    // Update config
                    BridgeConfig.setRuntimeConfig(
                        BridgeConfig.fromDiscoveredServer(
                            homeserver = serverInfo.homeserver,
                            bridgeUrl = serverInfo.bridgeUrl,
                            serverName = serverInfo.displayName
                        )
                    )

                    // CRITICAL: Gate on bridge health before any credential/login attempt
                    val healthResult = performBridgeHealthCheck(serverInfo.bridgeUrl, context)

                    if (!healthResult.isHealthy) {
                        _uiState.update { it.copy(
                            isConnecting = false,
                            serverInfo = serverInfo,
                            canProceed = false,
                            isNewServer = healthResult.details?.isNewServer ?: false,
                            bridgeHealthStatus = healthResult.status,
                            bridgeHealthMessage = healthResult.message,
                            bridgeHealthDetails = healthResult.details,
                            setupStep = SetupStep.ERROR
                        )}
                        return@launch
                    }

                    _uiState.update { it.copy(
                        bridgeHealthStatus = BridgeHealthStatus.HEALTHY,
                        isNewServer = healthResult.details?.isNewServer ?: false,
                        bridgeHealthDetails = healthResult.details
                    )}

                    val userId = serverInfo.userId
                    if (autoLogin && userId != null) {
                        // Auto-login flow - QR contained session info
                        _uiState.update { it.copy(
                            setupStep = SetupStep.CONNECTING,
                            serverInfo = serverInfo
                        )}

                        // Connect to bridge and sync
                        val deviceId = generateDeviceId()

                        when (val bridgeResult = setupService.connectWithCredentials(
                            username = userId,
                            password = "", // QR-provisioned, password not needed
                            deviceId = deviceId,
                            context = context
                        )) {
                            is SetupResult.Success -> {
                                _uiState.update { it.copy(
                                    isConnecting = false,
                                    isCompleted = true,
                                    serverInfo = bridgeResult.info,
                                    completedInfo = if (bridgeResult is SetupResult.Success && setupService.setupState.value is SetupState.Completed) {
                                        (setupService.setupState.value as SetupState.Completed).info
                                    } else null
                                )}
                            }
                            is SetupResult.Error -> {
                                // Fall back to requiring credentials
                                _uiState.update { it.copy(
                                    isConnecting = false,
                                    canProceed = true,
                                    setupStep = SetupStep.READY,
                                    serverInfo = serverInfo
                                )}
                            }
                        }
                    } else {
                        // No auto-login - user needs to enter credentials
                        _uiState.update { it.copy(
                            isConnecting = false,
                            serverInfo = serverInfo,
                            canProceed = true,
                            setupStep = SetupStep.READY
                        )}
                    }
                }
                is SetupResult.Error -> {
                    _uiState.update { it.copy(
                        isConnecting = false,
                        errorMessage = result.message,
                        fallbackOptions = result.fallbackOptions,
                        bridgeHealthStatus = BridgeHealthStatus.ERROR,
                        setupStep = SetupStep.ERROR
                    )}
                }
            }
        }
    }

    private fun generateDeviceId(): String {
        return "ARMORCLAW_${System.currentTimeMillis().toString(16).uppercase()}"
    }

    override fun onCleared() {
        super.onCleared()
        setupService.close()
    }
}

/**
 * UI State for setup screens
 */
data class SetupUiState(
    val setupStep: SetupStep = SetupStep.IDLE,
    val isConnecting: Boolean = false,
    val isCompleted: Boolean = false,
    val hasError: Boolean = false,
    val canProceed: Boolean = false,
    val errorMessage: String? = null,
    val fallbackServerUrl: String? = null,
    val serverInfo: DetectServerInfo? = null,
    val warnings: List<SecurityWarning> = emptyList(),
    val fallbackOptions: List<FallbackOption> = emptyList(),
    val isAdmin: Boolean = false,
    val adminLevel: AdminLevel = AdminLevel.NONE,
    val isDemo: Boolean = false,
    val isNewServer: Boolean = false,
    val completedInfo: SetupCompleteInfo? = null,
    // Bridge health state
    val bridgeHealthStatus: BridgeHealthStatus = BridgeHealthStatus.UNKNOWN,
    val bridgeHealthMessage: String? = null,
    val bridgeHealthDetails: BridgeHealthDetails? = null
)

/**
 * Bridge health check status
 */
enum class BridgeHealthStatus {
    UNKNOWN,        // Not checked yet
    CHECKING,       // Health check in progress
    HEALTHY,        // Bridge is reachable and healthy
    NOT_READY,      // Bridge healthy but still initializing (bridge_ready=false)
    UNREACHABLE,    // Bridge URL is not reachable
    UNHEALTHY,      // Bridge reachable but unhealthy
    ERROR           // Health check failed with error
}

/**
 * Detailed bridge health information
 */
data class BridgeHealthDetails(
    val bridgeUrl: String,
    val isIpOnlyServer: Boolean = false,
    val suggestedActions: List<String> = emptyList(),
    val suggestedPort: Int? = null,
    val bridgeReady: Boolean = true,
    val isNewServer: Boolean = false,
    val provisioningAvailable: Boolean? = null
)

enum class SetupStep {
    IDLE,
    DETECTING_SERVER,
    FALLBACK,
    READY,
    CONNECTING,
    STARTING_BRIDGE,
    AUTHENTICATING,
    CONNECTING_WEBSOCKET,
    CHECKING_PRIVILEGES,
    COMPLETED,
    ERROR;

    val progress: Float
        get() = when (this) {
            IDLE -> 0f
            DETECTING_SERVER -> 0.1f
            FALLBACK -> 0.15f
            READY -> 0.2f
            CONNECTING -> 0.3f
            STARTING_BRIDGE -> 0.4f
            AUTHENTICATING -> 0.6f
            CONNECTING_WEBSOCKET -> 0.8f
            CHECKING_PRIVILEGES -> 0.9f
            COMPLETED -> 1f
            ERROR -> 0f
        }

    val displayText: String
        get() = when (this) {
            IDLE -> "Ready to connect"
            DETECTING_SERVER -> "Detecting server..."
            FALLBACK -> "Trying backup server..."
            READY -> "Enter your credentials"
            CONNECTING -> "Connecting..."
            STARTING_BRIDGE -> "Starting secure bridge..."
            AUTHENTICATING -> "Authenticating..."
            CONNECTING_WEBSOCKET -> "Establishing real-time connection..."
            CHECKING_PRIVILEGES -> "Verifying account..."
            COMPLETED -> "Connected!"
            ERROR -> "Connection failed"
        }
}
