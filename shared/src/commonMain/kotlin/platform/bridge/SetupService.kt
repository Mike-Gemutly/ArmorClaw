package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.plugins.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName
import kotlinx.serialization.json.Json
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

/**
 * Service for handling initial ArmorClaw bridge setup
 *
 * This service ensures a flawless first-time setup experience with:
 * - Server validation and health checks
 * - Well-known discovery (Matrix and ArmorClaw)
 * - Admin privilege detection
 * - IP address security warnings
 * - Fallback mechanisms
 * - Progress tracking
 * - Signed configuration from QR/deep links
 *
 * ## Discovery Priority
 * 1. Signed config from QR/deep link (highest trust)
 * 2. Well-known discovery (.well-known/matrix/client)
 * 3. Manual URL derivation
 * 4. Fallback servers
 */
class SetupService(
    private val rpcClient: BridgeRpcClient,
    private val wsClient: BridgeWebSocketClient,
    private val httpClient: HttpClient = HttpClient {
        install(io.ktor.client.plugins.HttpTimeout) {
            requestTimeoutMillis = 10000
            connectTimeoutMillis = 5000
        }
    }
) {
    private val logger = repositoryLogger("SetupService", LogTag.Network.Bridge)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // JSON parser for signed config
    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
    }

    // Setup state
    private val _setupState = MutableStateFlow<SetupState>(SetupState.Idle)
    val setupState: StateFlow<SetupState> = _setupState.asStateFlow()

    // Setup configuration
    private val _config = MutableStateFlow(SetupConfig())
    val config: StateFlow<SetupConfig> = _config.asStateFlow()

    // Provisioning token from QR/deep link (used during first-boot claim)
    private var pendingSetupToken: String? = null

    // Detected server info
    private val _serverInfo = MutableStateFlow<DetectServerInfo?>(null)
    val serverInfo: StateFlow<DetectServerInfo?> = _serverInfo.asStateFlow()

    // Security warnings
    private val _securityWarnings = MutableStateFlow<List<SecurityWarning>>(emptyList())
    val securityWarnings: StateFlow<List<SecurityWarning>> = _securityWarnings.asStateFlow()

    /**
     * Start the setup process
     */
    suspend fun startSetup(
        homeserver: String,
        bridgeUrl: String?,
        context: OperationContext? = null
    ): SetupResult {
        val ctx = context ?: OperationContext.create()

        logger.logOperationStart("startSetup", mapOf(
            "homeserver" to homeserver,
            "bridge_url" to (bridgeUrl ?: "auto"),
            "correlation_id" to ctx.correlationId
        ))

        _setupState.value = SetupState.Discovering
        _securityWarnings.value = emptyList()

        return try {
            // Step 1: Detect server capabilities
            val serverInfo = detectServer(homeserver, bridgeUrl, ctx)
            _serverInfo.value = serverInfo

            // Step 2: Check security warnings
            checkSecurityWarnings(serverInfo, ctx)

            // Step 3: Update config based on detected info
            _config.value = SetupConfig(
                homeserver = homeserver,
                bridgeUrl = serverInfo.recommendedBridgeUrl,
                serverVersion = serverInfo.version,
                supportsE2EE = serverInfo.supportsE2EE,
                supportsRecovery = serverInfo.supportsRecovery,
                detectedRegion = serverInfo.region
            )

            _setupState.value = SetupState.ReadyForCredentials

            SetupResult.Success(serverInfo)
        } catch (e: Exception) {
            logger.logOperationError("startSetup", e)
            _setupState.value = SetupState.Error(e.message ?: "Unknown error")

            // Try fallback servers
            tryFallbackSetup(homeserver, ctx)
        }
    }

    /**
     * Connect with credentials
     */
    suspend fun connectWithCredentials(
        username: String,
        password: String,
        deviceId: String,
        context: OperationContext? = null
    ): SetupResult {
        val ctx = context ?: OperationContext.create()
        val config = _config.value

        if (config.homeserver.isEmpty()) {
            return SetupResult.Error("No server configured. Run startSetup first.")
        }

        _setupState.value = SetupState.Connecting

        logger.logOperationStart("connectWithCredentials", mapOf(
            "username" to username,
            "device_id" to deviceId,
            "correlation_id" to ctx.correlationId
        ))

        return try {
            // Step 1: Start bridge
            _setupState.value = SetupState.StartingBridge
            val bridgeResult = rpcClient.startBridge(
                userId = username,
                deviceId = deviceId,
                context = ctx
            )

            if (bridgeResult is RpcResult.Error) {
                throw SetupException.BridgeStartFailed(bridgeResult.message)
            }

            val bridgeResponse = (bridgeResult as RpcResult.Success).data

            // Step 2: Login to Matrix via bridge
            _setupState.value = SetupState.Authenticating
            val loginResult = rpcClient.matrixLogin(
                homeserver = config.homeserver,
                username = username,
                password = password,
                deviceId = deviceId,
                context = ctx
            )

            if (loginResult is RpcResult.Error) {
                throw SetupException.AuthenticationFailed(loginResult.message)
            }

            val loginResponse = (loginResult as RpcResult.Success).data

            // Step 3: Connect WebSocket (optional — ArmorChat uses Matrix /sync for
            // real-time events, not Bridge WebSocket. Failure here is non-fatal.)
            _setupState.value = SetupState.ConnectingWebSocket
            val sessionId = rpcClient.getSessionId()
                ?: throw SetupException.SessionNotFound("No session ID after bridge start")

            try {
                val wsConnected = wsClient.connect(sessionId, loginResponse.accessToken, ctx)
                if (!wsConnected) {
                    logger.logOperationError("connectWebSocket",
                        Exception("WebSocket connection failed — continuing without it (non-fatal)"))
                }
            } catch (wsError: Exception) {
                // Bridge WebSocket is ArmorTerminal-only per ArmorClaw architecture.
                // ArmorChat gets all real-time events from Matrix /sync directly.
                logger.logOperationError("connectWebSocket",
                    Exception("WebSocket unavailable: ${wsError.message} — continuing (non-fatal)"))
            }

            // Step 4: Attempt provisioning claim if setup token present (first-boot flow)
            val setupToken = pendingSetupToken
            var userInfo: UserInfo
            var claimedAdminToken: String? = null

            // RC-02: Check if provisioning is available before attempting claim.
            // If the bridge runs without a provisioning secret, provisioning_available
            // is false and calling provisioning.claim returns an RPC error instead of
            // a clean result, confusing the setup flow.
            val provisioningAvailable = try {
                when (val healthResult = rpcClient.healthCheck(ctx)) {
                    is RpcResult.Success -> {
                        healthResult.data["provisioning_available"] as? Boolean ?: true
                    }
                    is RpcResult.Error -> true // assume available if health check fails
                }
            } catch (_: Exception) { true }

            if (!setupToken.isNullOrBlank() && provisioningAvailable) {
                _setupState.value = SetupState.ClaimingAdmin
                logger.logOperationStart("provisioningClaim", mapOf(
                    "correlation_id" to ctx.correlationId
                ))

                val claimResult = rpcClient.provisioningClaim(
                    setupToken = setupToken,
                    deviceName = deviceId,
                    deviceType = "android",
                    context = ctx
                )

                when (claimResult) {
                    is RpcResult.Success -> {
                        val claim = claimResult.data
                        if (claim.success) {
                            // Claimed admin successfully — use server-assigned role
                            userInfo = UserInfo(
                                isAdmin = true,
                                adminLevel = claim.role ?: AdminLevel.OWNER
                            )
                            claimedAdminToken = claim.adminToken

                            // RC-01: Wire admin_token into RPC auth headers so
                            // all subsequent RPC calls include the Bearer token.
                            if (!claimedAdminToken.isNullOrBlank()) {
                                rpcClient.setAdminToken(claimedAdminToken)
                            }

                            // RC-03: Override homeserver URL if bridge returned one
                            // Critical for self-hosted deployments where the bridge's
                            // configured homeserver differs from the QR payload.
                            if (!claim.matrixHomeserver.isNullOrBlank()) {
                                _config.value = _config.value.copy(
                                    homeserver = claim.matrixHomeserver
                                )
                            }

                            logger.logOperationSuccess("provisioningClaim",
                                "role=${claim.role}, user=${claim.userId}")
                            pendingSetupToken = null // consumed
                        } else {
                            // Claim failed — token may be expired or already used
                            logger.logOperationError("provisioningClaim",
                                Exception(claim.message ?: "Claim rejected"))
                            pendingSetupToken = null

                            if (claim.message?.contains("already claimed", ignoreCase = true) == true) {
                                _setupState.value = SetupState.AlreadyClaimed(null)
                            }

                            // Fall back to server-authoritative role check
                            _setupState.value = SetupState.CheckingPrivileges
                            userInfo = getUserPrivilegesFromServer(loginResponse.userId, ctx)
                        }
                    }
                    is RpcResult.Error -> {
                        // RPC error — provisioning endpoint may not exist on older bridge
                        logger.logOperationError("provisioningClaim",
                            Exception("RPC error ${claimResult.code}: ${claimResult.message}"))
                        pendingSetupToken = null

                        // Fall back to server-authoritative role check
                        _setupState.value = SetupState.CheckingPrivileges
                        userInfo = getUserPrivilegesFromServer(loginResponse.userId, ctx)
                    }
                }
            } else {
                // No setup token or provisioning not available (RC-02) — fall back
                // to bridge.status role check path.
                if (!setupToken.isNullOrBlank() && !provisioningAvailable) {
                    logger.logOperationStart("provisioningSkipped", mapOf(
                        "reason" to "provisioning_available=false"
                    ))
                    pendingSetupToken = null
                }
                _setupState.value = SetupState.CheckingPrivileges
                userInfo = getUserPrivilegesFromServer(loginResponse.userId, ctx)
            }

            // Step 5: Complete setup
            _setupState.value = SetupState.Completed(
                SetupCompleteInfo(
                    userId = loginResponse.userId,
                    deviceId = loginResponse.deviceId,
                    sessionId = sessionId,
                    bridgeContainerId = bridgeResponse.containerId,
                    isAdmin = userInfo.isAdmin,
                    adminLevel = userInfo.adminLevel,
                    warnings = _securityWarnings.value,
                    completedAt = Clock.System.now(),
                    adminToken = claimedAdminToken
                )
            )

            logger.logOperationSuccess("connectWithCredentials", "user_id=${loginResponse.userId}, is_admin=${userInfo.isAdmin}")

            SetupResult.Success(
                DetectServerInfo(
                    homeserver = config.homeserver,
                    bridgeUrl = config.bridgeUrl ?: "",
                    version = config.serverVersion ?: "unknown",
                    userId = loginResponse.userId,
                    displayName = loginResponse.displayName,
                    isAdmin = userInfo.isAdmin,
                    adminLevel = userInfo.adminLevel,
                    supportsE2EE = config.supportsE2EE,
                    supportsRecovery = config.supportsRecovery,
                    region = config.detectedRegion ?: "us-east"
                )
            )
        } catch (e: SetupException) {
            logger.logOperationError("connectWithCredentials", e)
            _setupState.value = SetupState.Error(e.message ?: "Setup failed")
            SetupResult.Error(e.message ?: "Setup failed", e.fallbackOptions)
        } catch (e: Exception) {
            logger.logOperationError("connectWithCredentials", e)
            _setupState.value = SetupState.Error(e.message ?: "Unknown error")
            SetupResult.Error(e.message ?: "Unknown error")
        }
    }

    /**
     * Skip setup and use demo server
     */
    suspend fun useDemoServer(context: OperationContext? = null): SetupResult {
        val ctx = context ?: OperationContext.create()

        logger.logOperationStart("useDemoServer", mapOf(
            "correlation_id" to ctx.correlationId
        ))

        _setupState.value = SetupState.Discovering

        return try {
            // Demo server configuration
            val demoConfig = SetupConfig(
                homeserver = "https://demo.armorclaw.app",
                bridgeUrl = "https://bridge-demo.armorclaw.app",
                serverVersion = "1.6.2",
                supportsE2EE = true,
                supportsRecovery = true,
                detectedRegion = "us-east",
                isDemo = true
            )

            _config.value = demoConfig
            _serverInfo.value = DetectServerInfo(
                homeserver = demoConfig.homeserver,
                bridgeUrl = demoConfig.bridgeUrl!!,
                version = demoConfig.serverVersion ?: "unknown",
                supportsE2EE = true,
                supportsRecovery = true,
                region = demoConfig.detectedRegion ?: "us-east",
                isDemo = true
            )

            _setupState.value = SetupState.ReadyForCredentials

            logger.logOperationSuccess("useDemoServer")
            SetupResult.Success(_serverInfo.value!!)
        } catch (e: Exception) {
            logger.logOperationError("useDemoServer", e)
            _setupState.value = SetupState.Error(e.message ?: "Demo server unavailable")
            SetupResult.Error("Demo server unavailable: ${e.message}")
        }
    }

    /**
     * Parse signed configuration from QR code or deep link
     * This is the preferred discovery method - cryptographically verified by Bridge
     *
     * Supported formats:
     * - armorclaw://config?d=<base64-encoded-json>
     * - https://armorclaw.app/config?d=<base64-encoded-json>
     * - armorclaw://setup?token=<token>&server=<server>
     * - armorclaw://invite?code=<invite-code>
     */
    suspend fun parseSignedConfig(
        deepLinkOrQR: String,
        context: OperationContext? = null
    ): SetupResult {
        val ctx = context ?: OperationContext.create()

        logger.logOperationStart("parseSignedConfig", mapOf(
            "input_type" to when {
                deepLinkOrQR.startsWith("armorclaw://") -> "deep_link"
                deepLinkOrQR.startsWith("https://") -> "web_link"
                else -> "unknown"
            },
            "correlation_id" to ctx.correlationId
        ))

        _setupState.value = SetupState.Discovering

        return try {
            val result = when {
                deepLinkOrQR.startsWith("armorclaw://config?") -> parseConfigDeepLink(deepLinkOrQR)
                deepLinkOrQR.startsWith("https://armorclaw.app/config?") -> parseConfigDeepLink(deepLinkOrQR)
                deepLinkOrQR.startsWith("armorclaw://setup?") -> parseSetupDeepLink(deepLinkOrQR)
                deepLinkOrQR.startsWith("armorclaw://invite?") -> parseInviteDeepLink(deepLinkOrQR)
                deepLinkOrQR.startsWith("https://armorclaw.app/invite/") -> parseInviteWebLink(deepLinkOrQR)
                deepLinkOrQR.startsWith("https://armorclaw.app/setup?") -> parseSetupWebLink(deepLinkOrQR)
                else -> null
            }

            when (result) {
                is ConfigParseResult.Success -> {
                    val config = result.config

                    // Check expiration
                    if (config.expiresAt != null && config.expiresAt < Clock.System.now().epochSeconds) {
                        _setupState.value = SetupState.Error("Configuration expired")
                        return SetupResult.Error(
                            "Configuration has expired. Please scan a new QR code.",
                            listOf(FallbackOption.CONTACT_SUPPORT)
                        )
                    }

                    // Store setup token for first-boot provisioning claim
                    if (!config.setupToken.isNullOrBlank()) {
                        pendingSetupToken = config.setupToken
                    }

                    // Apply configuration
                    // RC-08: Carry over pushGateway, wsUrl, serverName from the signed
                    // QR payload so they are available to PushNotificationRepository
                    // and other consumers via SetupConfig.
                    _config.value = SetupConfig(
                        homeserver = config.matrixHomeserver,
                        bridgeUrl = config.rpcUrl,
                        serverVersion = config.version?.toString() ?: "1.0.0",
                        supportsE2EE = true,
                        supportsRecovery = true,
                        detectedRegion = config.region,
                        configSource = ConfigSource.SIGNED_URL,
                        wsUrl = config.wsUrl,
                        pushGateway = config.pushGateway,
                        serverName = config.serverName,
                        expiresAt = config.expiresAt
                    )

                    _serverInfo.value = DetectServerInfo(
                        homeserver = config.matrixHomeserver,
                        bridgeUrl = config.rpcUrl,
                        version = "1.0.0",
                        supportsE2EE = true,
                        supportsRecovery = true,
                        region = config.region ?: "us-east"
                    )

                    _setupState.value = SetupState.ReadyForCredentials

                    logger.logOperationSuccess("parseSignedConfig", "server=${config.serverName}")
                    SetupResult.Success(_serverInfo.value!!)
                }
                is ConfigParseResult.InviteCode -> {
                    // Handle invite code - extract server info and proceed
                    parseInviteAndSetup(result.code, ctx)
                }
                is ConfigParseResult.Error -> {
                    _setupState.value = SetupState.Error(result.message)
                    SetupResult.Error(result.message)
                }
                null -> {
                    _setupState.value = SetupState.Error("Invalid configuration format")
                    SetupResult.Error("Invalid configuration format. Expected armorclaw:// or https://armorclaw.app link")
                }
            }
        } catch (e: Exception) {
            logger.logOperationError("parseSignedConfig", e)
            _setupState.value = SetupState.Error(e.message ?: "Parse failed")
            SetupResult.Error("Failed to parse configuration: ${e.message}")
        }
    }

    /**
     * Smart discovery - tries multiple methods in priority order
     * 1. Signed config (if input looks like QR/deep link)
     * 2. Well-known discovery (if input is domain)
     * 3. Manual URL derivation (if input is URL)
     */
    suspend fun startSetupWithDiscovery(
        input: String,
        context: OperationContext? = null
    ): SetupResult {
        val ctx = context ?: OperationContext.create()

        // Check if it's a signed config / QR code
        if (input.startsWith("armorclaw://") || input.startsWith("https://armorclaw.app")) {
            return parseSignedConfig(input, ctx)
        }

        // Check if it's just a domain name (no protocol)
        val homeserver = if (!input.startsWith("http://") && !input.startsWith("https://")) {
            // Try well-known discovery first
            val wellKnownResult = tryWellKnownDiscovery(input, ctx)
            if (wellKnownResult is SetupResult.Success) {
                return wellKnownResult
            }
            "https://matrix.$input"
        } else {
            input
        }

        // Fall back to standard setup
        return startSetup(homeserver, null, ctx)
    }

    // Private parsing methods

    @OptIn(kotlin.io.encoding.ExperimentalEncodingApi::class)
    private fun parseConfigDeepLink(uri: String): ConfigParseResult {
        return try {
            // Extract base64 data from URL
            val data = when {
                uri.startsWith("armorclaw://config?") -> uri.substringAfter("armorclaw://config?d=")
                uri.startsWith("https://armorclaw.app/config?") -> uri.substringAfter("https://armorclaw.app/config?d=")
                else -> return ConfigParseResult.Error("Invalid config URL format")
            }

            if (data.isBlank()) {
                return ConfigParseResult.Error("Missing configuration data")
            }

            // Decode base64 (URL-safe)
            val jsonBytes = Base64.UrlSafe.decode(data)
            val jsonString = jsonBytes.decodeToString()

            // Parse JSON
            val config = json.decodeFromString<SignedServerConfig>(jsonString)

            ConfigParseResult.Success(config)
        } catch (e: Exception) {
            ConfigParseResult.Error("Failed to parse config: ${e.message}")
        }
    }

    private fun parseSetupDeepLink(uri: String): ConfigParseResult {
        return try {
            val params = uri.substringAfter("?").parseQueryParams()
            val token = params["token"] ?: return ConfigParseResult.Error("Missing token")
            val server = params["server"] ?: return ConfigParseResult.Error("Missing server")

            // Auto-derive bridge URL based on server type (IP vs domain)
            val derivedBridgeUrl = deriveBridgeUrl(server)

            ConfigParseResult.Success(
                SignedServerConfig(
                    matrixHomeserver = server,
                    rpcUrl = derivedBridgeUrl,
                    wsUrl = derivedBridgeUrl.replace("/api", "/ws").let {
                        // Handle WebSocket protocol
                        it.replace("https://", "wss://").replace("http://", "ws://")
                    },
                    pushGateway = derivedBridgeUrl.replace("/api", "/_matrix/push/v1/notify"),
                    serverName = server,
                    region = null,
                    expiresAt = null,
                    setupToken = token
                )
            )
        } catch (e: Exception) {
            ConfigParseResult.Error("Failed to parse setup link: ${e.message}")
        }
    }

    private fun parseInviteDeepLink(uri: String): ConfigParseResult {
        val params = uri.substringAfter("?").parseQueryParams()
        val code = params["code"] ?: return ConfigParseResult.Error("Missing invite code")
        return ConfigParseResult.InviteCode(code)
    }

    private fun parseInviteWebLink(uri: String): ConfigParseResult {
        val code = uri.substringAfter("/invite/").substringBefore("?")
        if (code.isBlank()) return ConfigParseResult.Error("Missing invite code")
        return ConfigParseResult.InviteCode(code)
    }

    private fun parseSetupWebLink(uri: String): ConfigParseResult {
        val params = uri.substringAfter("?").parseQueryParams()
        val token = params["token"] ?: return ConfigParseResult.Error("Missing token")
        val server = params["server"] ?: return ConfigParseResult.Error("Missing server")

        // Auto-derive bridge URL based on server type (IP vs domain)
        val derivedBridgeUrl = deriveBridgeUrl(server)

        return ConfigParseResult.Success(
            SignedServerConfig(
                matrixHomeserver = server,
                rpcUrl = derivedBridgeUrl,
                wsUrl = derivedBridgeUrl.replace("/api", "/ws").let {
                    // Handle WebSocket protocol
                    it.replace("https://", "wss://").replace("http://", "ws://")
                },
                pushGateway = derivedBridgeUrl.replace("/api", "/_matrix/push/v1/notify"),
                serverName = server,
                region = null,
                expiresAt = null,
                setupToken = token
            )
        )
    }

    private suspend fun tryWellKnownDiscovery(
        serverName: String,
        context: OperationContext
    ): SetupResult {
        logger.logOperationStart("tryWellKnownDiscovery", mapOf(
            "server_name" to serverName,
            "correlation_id" to context.correlationId
        ))

        return try {
            // Normalize server name (remove protocol if present)
            val domain = serverName
                .removePrefix("https://")
                .removePrefix("http://")
                .removeSuffix("/")
                .split("/").first()

            // Step 1: Try Matrix well-known discovery
            val wellKnownUrl = "https://$domain/.well-known/matrix/client"

            logger.logNetworkRequest(wellKnownUrl, "GET")

            val response: HttpResponse = httpClient.get(wellKnownUrl) {
                timeout {
                    requestTimeoutMillis = 5000
                    connectTimeoutMillis = 3000
                }
            }

            if (response.status != HttpStatusCode.OK) {
                logger.logOperationError("tryWellKnownDiscovery",
                    Exception("HTTP ${response.status}"))
                return SetupResult.Error("Well-known not found at $domain")
            }

            val responseBody = response.body<String>()
            val wellKnown = json.decodeFromString<MatrixWellKnown>(responseBody)

            // Extract homeserver URL from well-known
            val homeserverUrl = wellKnown.homeserver?.baseUrl
                ?: return SetupResult.Error("No homeserver in well-known")

            // Step 2: Check for ArmorClaw-specific configuration
            val bridgeUrl = wellKnown.armorclaw?.bridgeUrl
                ?: wellKnown.armorclaw?.rpcUrl
                ?: BridgeConfig.deriveBridgeUrl(homeserverUrl)

            val wsUrl = wellKnown.armorclaw?.wsUrl
            val pushGateway = wellKnown.armorclaw?.pushGateway
            val serverDisplayName = wellKnown.armorclaw?.serverName ?: domain

            logger.logNetworkResponse(wellKnownUrl, response.status.value, 0L)

            // Step 3: Create discovered configuration
            val config = SetupConfig(
                homeserver = homeserverUrl,
                bridgeUrl = bridgeUrl,
                wsUrl = wsUrl,
                pushGateway = pushGateway,
                serverName = serverDisplayName,
                serverVersion = wellKnown.armorclaw?.version,
                supportsE2EE = wellKnown.armorclaw?.supportsE2ee ?: true,
                supportsRecovery = wellKnown.armorclaw?.supportsRecovery ?: true,
                detectedRegion = wellKnown.armorclaw?.region ?: detectRegionFromUrl(domain),
                configSource = ConfigSource.WELL_KNOWN
            )

            _config.value = config
            _serverInfo.value = DetectServerInfo(
                homeserver = homeserverUrl,
                bridgeUrl = bridgeUrl,
                version = config.serverVersion ?: "unknown",
                supportsE2EE = config.supportsE2EE,
                supportsRecovery = config.supportsRecovery,
                region = config.detectedRegion ?: "us-east"
            )

            _setupState.value = SetupState.ReadyForCredentials

            logger.logOperationSuccess("tryWellKnownDiscovery",
                "homeserver=$homeserverUrl, bridge=$bridgeUrl")

            SetupResult.Success(_serverInfo.value!!)

        } catch (e: io.ktor.client.plugins.ClientRequestException) {
            logger.logOperationError("tryWellKnownDiscovery", e)
            SetupResult.Error("Server not found: ${e.message}")
        } catch (e: io.ktor.client.plugins.ServerResponseException) {
            logger.logOperationError("tryWellKnownDiscovery", e)
            SetupResult.Error("Server error: ${e.message}")
        } catch (e: io.ktor.utils.io.errors.IOException) {
            logger.logOperationError("tryWellKnownDiscovery", e)
            SetupResult.Error("Network error: ${e.message}")
        } catch (e: Exception) {
            logger.logOperationError("tryWellKnownDiscovery", e)
            SetupResult.Error("Discovery failed: ${e.message}")
        }
    }

    /**
     * Matrix well-known response format
     * See: https://matrix.org/docs/spec/client_server/r0.6.1#well-known-uri
     */
    @Serializable
    private data class MatrixWellKnown(
        @SerialName("m.homeserver")
        val homeserver: HomeserverInfo? = null,
        @SerialName("m.identity_server")
        val identityServer: IdentityServerInfo? = null,
        @SerialName("com.armorclaw")
        val armorclaw: ArmorClawWellKnown? = null
    )

    @Serializable
    private data class HomeserverInfo(
        @SerialName("base_url")
        val baseUrl: String
    )

    @Serializable
    private data class IdentityServerInfo(
        @SerialName("base_url")
        val baseUrl: String
    )

    /**
     * ArmorClaw-specific well-known configuration
     * This is a custom extension to the Matrix well-known format
     */
    @Serializable
    private data class ArmorClawWellKnown(
        @SerialName("bridge_url")
        val bridgeUrl: String? = null,
        @SerialName("rpc_url")
        val rpcUrl: String? = null,
        @SerialName("ws_url")
        val wsUrl: String? = null,
        @SerialName("push_gateway")
        val pushGateway: String? = null,
        @SerialName("server_name")
        val serverName: String? = null,
        val version: String? = null,
        val region: String? = null,
        @SerialName("supports_e2ee")
        val supportsE2ee: Boolean? = null,
        @SerialName("supports_recovery")
        val supportsRecovery: Boolean? = null
    )

    private suspend fun parseInviteAndSetup(
        inviteCode: String,
        context: OperationContext
    ): SetupResult {
        // Invite codes can contain encoded server info
        // For now, use default server
        return startSetup("https://matrix.armorclaw.app", null, context)
    }

    private fun String.parseQueryParams(): Map<String, String> {
        return split("&")
            .mapNotNull { param ->
                val parts = param.split("=", limit = 2)
                if (parts.size == 2) {
                    parts[0] to decodeUrlPart(parts[1])
                } else if (parts.size == 1 && parts[0].isNotEmpty()) {
                    parts[0] to ""
                } else {
                    null
                }
            }
            .toMap()
    }

    /**
     * URL decode a string part (cross-platform)
     * Handles common URL-encoded characters without using java.net.URLDecoder
     */
    /**
     * URL decode a string part (cross-platform)
     *
     * Uses regex-based percent decoding to correctly handle all encoded
     * characters, including %25 (literal percent sign), without ordering issues.
     */
    private fun decodeUrlPart(encoded: String): String {
        return encoded
            .replace("+", " ")
            .replace(Regex("%([0-9A-Fa-f]{2})")) { matchResult ->
                val hexValue = matchResult.groupValues[1].toInt(16)
                hexValue.toChar().toString()
            }
    }

    /**
     * Reset setup state
     */
    fun resetSetup() {
        _setupState.value = SetupState.Idle
        _serverInfo.value = null
        _securityWarnings.value = emptyList()
        _config.value = SetupConfig()
        pendingSetupToken = null
    }

    /**
     * Dismiss a security warning
     */
    fun dismissWarning(warningId: String) {
        _securityWarnings.value = _securityWarnings.value.filter { it.id != warningId }
    }

    // Private implementation

    private suspend fun detectServer(
        homeserver: String,
        bridgeUrl: String?,
        context: OperationContext
    ): DetectServerInfo {
        // Determine bridge URL from homeserver if not provided
        val resolvedBridgeUrl = bridgeUrl ?: deriveBridgeUrl(homeserver)

        // Check if this is an IP-only server
        val serverDomain = homeserver
            .removePrefix("https://")
            .removePrefix("http://")
            .removeSuffix("/")
            .split("/").first()
            .split(":").first()
        val isIpOnly = isIpAddress(serverDomain)

        // Try to get bridge health/status
        return when (val healthResult = rpcClient.healthCheck(context)) {
            is RpcResult.Success -> {
                val health = healthResult.data
                DetectServerInfo(
                    homeserver = homeserver,
                    bridgeUrl = resolvedBridgeUrl,
                    version = health["version"] as? String ?: "unknown",
                    supportsE2EE = health["supports_e2ee"] as? Boolean ?: true,
                    supportsRecovery = health["supports_recovery"] as? Boolean ?: true,
                    region = health["region"] as? String ?: detectRegionFromUrl(homeserver),
                    recommendedBridgeUrl = resolvedBridgeUrl,
                    isNewServer = health["is_new_server"] as? Boolean ?: false
                )
            }
            is RpcResult.Error -> {
                // If health check fails, use defaults with warning
                val warningMessage = if (isIpOnly) {
                    "Bridge at $resolvedBridgeUrl is not responding. " +
                    "For IP-only servers, ensure the bridge is running on port 8080. " +
                    "Check server logs for 'socket error' or 'flag redefined' messages."
                } else {
                    "Could not verify server capabilities. Some features may be limited."
                }

                _securityWarnings.value += SecurityWarning(
                    id = "server_unverified",
                    type = WarningType.SERVER_UNVERIFIED,
                    title = "Server Verification Failed",
                    message = warningMessage,
                    severity = WarningSeverity.MEDIUM,
                    canDismiss = true
                )

                DetectServerInfo(
                    homeserver = homeserver,
                    bridgeUrl = resolvedBridgeUrl,
                    version = "unknown",
                    supportsE2EE = true, // Assume supported
                    supportsRecovery = true, // Assume supported
                    region = detectRegionFromUrl(homeserver),
                    recommendedBridgeUrl = resolvedBridgeUrl
                )
            }
        }
    }

    private suspend fun checkSecurityWarnings(
        serverInfo: DetectServerInfo,
        context: OperationContext
    ) {
        val warnings = mutableListOf<SecurityWarning>()

        // Check 1: IP Address Warning - warn if connecting to non-local server
        if (!isLocalServer(serverInfo.homeserver)) {
            warnings += SecurityWarning(
                id = "external_server",
                type = WarningType.EXTERNAL_SERVER,
                title = "Connecting to External Server",
                message = "You are connecting to a server outside your local network. " +
                        "Ensure you trust this server before proceeding.",
                severity = WarningSeverity.LOW,
                canDismiss = true
            )
        }

        // Check 2: Shared IP Warning
        if (serverInfo.isSharedIp == true) {
            warnings += SecurityWarning(
                id = "shared_ip",
                type = WarningType.SHARED_IP,
                title = "Shared IP Address Detected",
                message = "This server appears to be shared with other users. " +
                        "Your connection may be visible to the server administrator.",
                severity = WarningSeverity.HIGH,
                canDismiss = true
            )
        }

        // Check 3: Unencrypted Connection Warning
        if (!serverInfo.homeserver.startsWith("https://")) {
            warnings += SecurityWarning(
                id = "unencrypted",
                type = WarningType.UNENCRYPTED_CONNECTION,
                title = "Unencrypted Connection",
                message = "This server does not use HTTPS. Your data may be intercepted.",
                severity = WarningSeverity.CRITICAL,
                canDismiss = false
            )
        }

        // Check 4: Certificate Warning (if applicable)
        if (serverInfo.certificateIssues == true) {
            warnings += SecurityWarning(
                id = "certificate",
                type = WarningType.CERTIFICATE_ISSUE,
                title = "Certificate Warning",
                message = "The server's security certificate could not be verified. " +
                        "This could indicate a security issue.",
                severity = WarningSeverity.HIGH,
                canDismiss = true
            )
        }

        _securityWarnings.value = warnings
    }

    /**
     * Get user privileges from server (Server-Authoritative Role Assignment)
     *
     * FIX for Bug #3: Admin privilege detection is now server-authoritative.
     * The server determines the user's role based on its own logic (e.g., first user,
     * server config, etc.) and returns it in the bridge status response.
     *
     * This eliminates race conditions where multiple clients connecting simultaneously
     * could all claim to be "first user" and receive admin status.
     *
     * The Bridge Server should return:
     * - For new server, first registered user: OWNER
     * - For subsequent users: NONE (or role based on server config)
     * - For invited users: Role assigned by inviter
     */
    private suspend fun getUserPrivilegesFromServer(
        userId: String,
        context: OperationContext
    ): UserInfo {
        return try {
            val statusResult = rpcClient.getBridgeStatus(context)

            if (statusResult is RpcResult.Success) {
                val status = statusResult.data

                // Server must provide the user's role explicitly
                // This is the authoritative source of truth
                val serverRole = status.userRole ?: AdminLevel.NONE
                val isAdmin = serverRole == AdminLevel.OWNER || serverRole == AdminLevel.ADMIN

                logger.logOperationSuccess(
                    "getUserPrivilegesFromServer",
                    "userId=$userId, role=$serverRole"
                )

                UserInfo(
                    isAdmin = isAdmin,
                    adminLevel = serverRole
                )
            } else {
                // If server doesn't respond, default to no privileges
                // The server is the authority - if it can't tell us, we assume no role
                logger.logOperationError(
                    "getUserPrivilegesFromServer",
                    Exception("Failed to get bridge status")
                )
                UserInfo(isAdmin = false, adminLevel = AdminLevel.NONE)
            }
        } catch (e: Exception) {
            logger.logOperationError("getUserPrivilegesFromServer", e)
            UserInfo(isAdmin = false, adminLevel = AdminLevel.NONE)
        }
    }

    /**
     * @deprecated Use getUserPrivilegesFromServer instead.
     * This method had a race condition bug where client-side determination
     * based on messageCount could be wrong if multiple users connected simultaneously.
     */
    @Deprecated(
        message = "Use getUserPrivilegesFromServer for server-authoritative role assignment",
        replaceWith = ReplaceWith("getUserPrivilegesFromServer(userId, context)")
    )
    private suspend fun checkUserPrivileges(
        userId: String,
        context: OperationContext
    ): UserInfo {
        return getUserPrivilegesFromServer(userId, context)
    }

    private suspend fun tryFallbackSetup(
        originalHomeserver: String,
        context: OperationContext
    ): SetupResult {
        logger.logOperationStart("tryFallbackSetup", mapOf(
            "original_homeserver" to originalHomeserver
        ))

        val fallbackServers = listOf(
            "https://bridge.armorclaw.app",
            "https://bridge-backup.armorclaw.app",
            "https://bridge-eu.armorclaw.app"
        )

        for (fallbackUrl in fallbackServers) {
            try {
                logger.logOperationStart("fallback_attempt", mapOf("url" to fallbackUrl))

                _setupState.value = SetupState.FallbackAttempt(fallbackUrl)
                _config.value = _config.value.copy(bridgeUrl = fallbackUrl)

                // Try health check on fallback
                when (rpcClient.healthCheck(context)) {
                    is RpcResult.Success -> {
                        _serverInfo.value = DetectServerInfo(
                            homeserver = originalHomeserver,
                            bridgeUrl = fallbackUrl,
                            version = "1.6.2",
                            supportsE2EE = true,
                            supportsRecovery = true,
                            region = detectRegionFromUrl(fallbackUrl),
                            isFallback = true
                        )

                        _securityWarnings.value += SecurityWarning(
                            id = "fallback_active",
                            type = WarningType.FALLBACK_SERVER,
                            title = "Using Backup Server",
                            message = "Primary server unavailable. Connected to backup server: $fallbackUrl",
                            severity = WarningSeverity.LOW,
                            canDismiss = true
                        )

                        _setupState.value = SetupState.ReadyForCredentials

                        logger.logOperationSuccess("fallback_attempt", "url=$fallbackUrl")
                        return SetupResult.Success(_serverInfo.value!!)
                    }
                    is RpcResult.Error -> {
                        logger.logOperationError("fallback_attempt",
                            Exception("Health check failed"), mapOf("url" to fallbackUrl))
                        continue
                    }
                }
            } catch (e: Exception) {
                logger.logOperationError("fallback_attempt", e, mapOf("url" to fallbackUrl))
                continue
            }
        }

        // All fallbacks failed
        _setupState.value = SetupState.Error("All servers unavailable")

        return SetupResult.Error(
            message = "Unable to connect to any server. Please check your internet connection.",
            fallbackOptions = listOf(
                FallbackOption.RETRY,
                FallbackOption.USE_DEMO,
                FallbackOption.CONTACT_SUPPORT
            )
        )
    }

    /**
     * Derive bridge URL from homeserver URL
     *
     * HTTPS vs HTTP Protocol Preservation:
     * - Preserves the protocol from input URL (http:// or https://)
     * - For IP-only servers: http://IP:8008 -> http://IP:8080
     * - For HTTPS IP servers: https://IP:8443 -> https://IP:8443 (preserves port for TLS)
     * - For domain servers: https://matrix.example.com -> https://bridge.example.com
     *
     * Port Assumption (MVP):
     * - For IP-only servers with http://, assumes bridge on port 8080
     * - For IP-only servers with https://, preserves the input port (TLS typically on same port)
     * - Future: Will support configurable RPC port via .well-known or signed config
     *
     * @param homeserver The Matrix homeserver URL
     * @return The derived bridge RPC URL
     */
    private fun deriveBridgeUrl(homeserver: String): String {
        return try {
            val url = homeserver.removeSuffix("/")
            
            // Determine protocol from input
            val (protocol, hostWithPort) = when {
                url.startsWith("https://") -> "https://" to url.removePrefix("https://")
                url.startsWith("http://") -> "http://" to url.removePrefix("http://")
                else -> "https://" to url // Default to HTTPS for security
            }

            val hostPart = hostWithPort.split("/").first()

            // Extract host and port
            val (host, inputPort) = if (hostPart.contains(":")) {
                val parts = hostPart.split(":")
                parts[0] to parts[1].toIntOrNull()
            } else {
                hostPart to null
            }

            // Check if it's an IP address (IPv4)
            val isIp = isIpAddress(host)

            if (isIp) {
                // For IP-only servers:
                // - HTTP: use standard bridge port 8080 (MVP assumption)
                // - HTTPS: preserve input port (TLS typically on same port)
                if (protocol == "http://") {
                    // Standard HTTP derivation: port 8080
                    "$protocol$host:8080"
                } else {
                    // HTTPS: preserve input port or use 8443 as default TLS port
                    val bridgePort = inputPort ?: 8443
                    "$protocol$host:$bridgePort"
                }
            } else {
                // For domain servers, derive bridge subdomain
                val bridgeHost = when {
                    host.contains("matrix.") -> host.replace("matrix.", "bridge.")
                    host.contains("chat.") -> host.replace("chat.", "bridge.")
                    else -> "bridge.$host"
                }
                
                // Preserve port if specified, otherwise use protocol default
                if (inputPort != null) {
                    "$protocol$bridgeHost:$inputPort"
                } else {
                    "$protocol$bridgeHost"
                }
            }
        } catch (e: Exception) {
            // Fallback to default production bridge
            "https://bridge.armorclaw.app"
        }
    }

    /**
     * Check if the given string is an IP address (IPv4)
     * Delegates to shared NetworkUtils to avoid code duplication (RC-05).
     */
    private fun isIpAddress(host: String): Boolean = com.armorclaw.shared.platform.network.NetworkUtils.isIpAddress(host)

    private fun detectRegionFromUrl(url: String): String {
        return when {
            url.contains("eu.") || url.contains("-eu") -> "eu-west"
            url.contains("asia.") || url.contains("-asia") -> "asia-east"
            url.contains("us-west") -> "us-west"
            else -> "us-east"
        }
    }

    private fun isLocalServer(url: String): Boolean {
        val localPatterns = listOf(
            "localhost",
            "127.0.0.1",
            "10.0.2.2", // Android emulator
            "192.168.",
            "10.",
            ".local",
            ".lan"
        )
        return localPatterns.any { url.contains(it) }
    }

    fun close() {
        scope.cancel()
    }
}

// State classes

/**
 * Setup state machine states
 * Follows the discovery flow:
 * Idle -> Discovering -> Discovered/WaitingForQR -> Configured
 */
sealed class SetupState {
    /** Initial state, waiting to start discovery */
    object Idle : SetupState()

    /** Actively scanning for servers via mDNS, well-known, etc. */
    object Discovering : SetupState()

    /** Found servers, waiting for user selection */
    data class Discovered(val servers: List<DiscoveredServer>) : SetupState()

    /** Found a server but needs QR scan for complete config */
    data class IncompleteDiscovery(val server: DiscoveredServer, val reason: String) : SetupState()

    /** Trying fallback servers */
    data class FallbackAttempt(val serverUrl: String) : SetupState()

    /** Ready for user to enter credentials */
    object ReadyForCredentials : SetupState()

    /** Connecting to Matrix */
    object Connecting : SetupState()

    /** Starting bridge container */
    object StartingBridge : SetupState()

    /** Authenticating with Matrix */
    object Authenticating : SetupState()

    /** Connecting to WebSocket */
    object ConnectingWebSocket : SetupState()

    /** Claiming admin role via provisioning token (first-boot) */
    object ClaimingAdmin : SetupState()

    /** Provisioning token expired — need fresh QR scan */
    data class ProvisioningExpired(val message: String) : SetupState()

    /** Another device already claimed admin */
    data class AlreadyClaimed(val claimedBy: String?) : SetupState()

    /** Checking user privileges */
    object CheckingPrivileges : SetupState()

    /** Setup completed successfully */
    data class Completed(val info: SetupCompleteInfo) : SetupState()

    /** Setup failed */
    data class Error(val message: String, val options: List<FallbackOption> = emptyList()) : SetupState()

    // Convenience properties
    val isConnecting: Boolean
        get() = this is Discovering || this is FallbackAttempt ||
                this is Connecting || this is StartingBridge ||
                this is Authenticating || this is ConnectingWebSocket ||
                this is ClaimingAdmin || this is CheckingPrivileges

    val isCompleted: Boolean
        get() = this is Completed

    val hasError: Boolean
        get() = this is Error

    val needsQRScan: Boolean
        get() = this is IncompleteDiscovery

    val isConfigured: Boolean
        get() = this is ReadyForCredentials || this is Completed
}

/**
 * Represents a discovered server
 */
data class DiscoveredServer(
    val name: String,
    val host: String,
    val port: Int,
    val homeserver: String? = null,
    val bridgeUrl: String? = null,
    val discoveryMethod: DiscoveryMethod = DiscoveryMethod.MDNS,
    val requiresQRSetup: Boolean = false
) {
    val isComplete: Boolean
        get() = !homeserver.isNullOrBlank() && !bridgeUrl.isNullOrBlank()

    val displayUrl: String
        get() = if (port == 443 || port == 0) {
            "https://$host"
        } else {
            "https://$host:$port"
        }
}

/**
 * Discovery method used to find the server
 */
enum class DiscoveryMethod {
    MDNS,           // mDNS/Bonjour discovery
    WELL_KNOWN,     // Matrix well-known discovery
    SIGNED_URL,     // Signed QR code or deep link
    MANUAL,         // Manual entry
    CACHED          // Previously cached config
}

data class SetupConfig(
    val homeserver: String = "",
    val bridgeUrl: String? = null,
    val serverVersion: String? = null,
    val supportsE2EE: Boolean = true,
    val supportsRecovery: Boolean = true,
    val detectedRegion: String? = null,
    val isDemo: Boolean = false,
    val configSource: ConfigSource = ConfigSource.DEFAULT,
    val wsUrl: String? = null,
    val pushGateway: String? = null,
    val serverName: String? = null,
    val expiresAt: Long? = null
) {
    val isDebug: Boolean
        get() = homeserver.contains("10.") ||
                homeserver.contains("192.168.") ||
                homeserver.contains("localhost") ||
                homeserver.contains("10.0.2.2")

    val isExpired: Boolean
        get() = expiresAt != null && expiresAt < Clock.System.now().epochSeconds

    fun isConfigured(): Boolean {
        return homeserver.isNotBlank() && !bridgeUrl.isNullOrBlank()
    }

    fun toDetectServerInfo(): DetectServerInfo = DetectServerInfo(
        homeserver = homeserver,
        bridgeUrl = bridgeUrl ?: "",
        version = serverVersion ?: "unknown"
    )

    companion object {
        /**
         * Create default production configuration
         */
        fun createDefault(): SetupConfig = SetupConfig(
            homeserver = "https://matrix.armorclaw.app",
            bridgeUrl = "https://bridge.armorclaw.app/api",
            wsUrl = "wss://bridge.armorclaw.app/ws",
            pushGateway = "https://bridge.armorclaw.app/_matrix/push/v1/notify",
            serverName = "ArmorClaw",
            configSource = ConfigSource.DEFAULT
        )

        /**
         * Create debug/development configuration
         */
        fun createDebug(): SetupConfig = SetupConfig(
            homeserver = "http://10.0.2.2:8008",
            bridgeUrl = "http://10.0.2.2:8080/api",
            wsUrl = "ws://10.0.2.2:8080/ws",
            pushGateway = "http://10.0.2.2:8080/_matrix/push/v1/notify",
            serverName = "Debug Server",
            configSource = ConfigSource.DEFAULT,
            isDemo = true
        )
    }
}

data class DetectServerInfo(
    val homeserver: String,
    val bridgeUrl: String,
    val version: String,
    val userId: String? = null,
    val displayName: String? = null,
    val isAdmin: Boolean = false,
    val adminLevel: AdminLevel = AdminLevel.NONE,
    val supportsE2EE: Boolean = true,
    val supportsRecovery: Boolean = true,
    val region: String = "us-east",
    val isDemo: Boolean = false,
    val isFallback: Boolean = false,
    val isSharedIp: Boolean? = null,
    val certificateIssues: Boolean? = null,
    val recommendedBridgeUrl: String? = null,
    val isNewServer: Boolean = false
)

data class SetupCompleteInfo(
    val userId: String,
    val deviceId: String,
    val sessionId: String,
    val bridgeContainerId: String?,
    val isAdmin: Boolean,
    val adminLevel: AdminLevel,
    val warnings: List<SecurityWarning>,
    val completedAt: Instant,
    /** Bridge admin token from provisioning.claim — used for authenticated RPC calls */
    val adminToken: String? = null
)

data class SecurityWarning(
    val id: String,
    val type: WarningType,
    val title: String,
    val message: String,
    val severity: WarningSeverity,
    val canDismiss: Boolean,
    val dismissed: Boolean = false
)

enum class WarningType {
    EXTERNAL_SERVER,
    SHARED_IP,
    UNENCRYPTED_CONNECTION,
    CERTIFICATE_ISSUE,
    SERVER_UNVERIFIED,
    FALLBACK_SERVER,
    ADMIN_PRIVILEGE_WARNING
}

enum class WarningSeverity {
    LOW,
    MEDIUM,
    HIGH,
    CRITICAL
}

enum class AdminLevel {
    NONE,
    MODERATOR,
    ADMIN,
    OWNER
}

enum class FallbackOption {
    RETRY,
    USE_DEMO,
    CONTACT_SUPPORT,
    CHECK_INTERNET,
    TRY_LATER
}

data class UserInfo(
    val isAdmin: Boolean,
    val adminLevel: AdminLevel
)

sealed class SetupResult {
    data class Success(val info: DetectServerInfo) : SetupResult()
    data class Error(
        val message: String,
        val fallbackOptions: List<FallbackOption> = emptyList()
    ) : SetupResult()

    val isSuccess: Boolean get() = this is Success
    val isError: Boolean get() = this is Error
}

sealed class SetupException(message: String) : Exception(message) {
    val fallbackOptions: List<FallbackOption>
        get() = when (this) {
            is BridgeStartFailed -> listOf(FallbackOption.RETRY, FallbackOption.USE_DEMO)
            is AuthenticationFailed -> listOf(FallbackOption.RETRY)
            is WebSocketFailed -> listOf(FallbackOption.RETRY, FallbackOption.TRY_LATER)
            is SessionNotFound -> listOf(FallbackOption.CONTACT_SUPPORT)
        }

    class BridgeStartFailed(message: String) : SetupException("Bridge start failed: $message")
    class AuthenticationFailed(message: String) : SetupException("Authentication failed: $message")
    class WebSocketFailed(message: String) : SetupException("WebSocket connection failed: $message")
    class SessionNotFound(message: String) : SetupException(message)
}

// ============================================================================
// Signed Configuration Models
// ============================================================================

/**
 * Source of configuration - for tracking how server was discovered
 * Priority: SIGNED_URL > WELL_KNOWN > MDNS > MANUAL > CACHED > DEFAULT
 */
enum class ConfigSource {
    /** BuildConfig defaults (lowest priority) */
    DEFAULT,

    /** User entered manually */
    MANUAL,

    /** Matrix well-known discovery */
    WELL_KNOWN,

    /** mDNS/Bonjour discovery */
    MDNS,

    /** Signed QR code or deep link from Bridge (highest priority) */
    SIGNED_URL,

    /** Previously cached configuration */
    CACHED,

    /** From invite code */
    INVITE
}

/**
 * Signed server configuration from Bridge QR/deep link
 * Matches Bridge's ConfigPayload structure
 */
@kotlinx.serialization.Serializable
data class SignedServerConfig(
    /** QR payload version for forward compatibility (bridge sends version: 1) */
    val version: Int? = null,
    @kotlinx.serialization.SerialName("matrix_homeserver")
    val matrixHomeserver: String,
    @kotlinx.serialization.SerialName("rpc_url")
    val rpcUrl: String,
    @kotlinx.serialization.SerialName("ws_url")
    val wsUrl: String? = null,
    @kotlinx.serialization.SerialName("push_gateway")
    val pushGateway: String? = null,
    @kotlinx.serialization.SerialName("server_name")
    val serverName: String,
    val region: String? = null,
    @kotlinx.serialization.SerialName("bridge_public_key")
    val bridgePublicKey: String? = null,
    @kotlinx.serialization.SerialName("expires_at")
    val expiresAt: Long? = null,
    val signature: String? = null,
    // For setup tokens
    @kotlinx.serialization.SerialName("setup_token")
    val setupToken: String? = null
)

/**
 * Result of parsing a configuration URL
 */
sealed class ConfigParseResult {
    data class Success(val config: SignedServerConfig) : ConfigParseResult()
    data class InviteCode(val code: String) : ConfigParseResult()
    data class Error(val message: String) : ConfigParseResult()
}
