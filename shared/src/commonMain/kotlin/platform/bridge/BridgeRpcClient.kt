package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.domain.model.BrowserCommand
import com.armorclaw.shared.domain.model.BrowserJobPriority
import com.armorclaw.shared.domain.model.BrowserJobStatus
import com.armorclaw.shared.domain.model.BrowserEnqueueResponse
import com.armorclaw.shared.domain.model.BrowserJobResponse
import com.armorclaw.shared.domain.model.BrowserJobListResponse
import com.armorclaw.shared.domain.model.BrowserCancelResponse
import com.armorclaw.shared.domain.model.BrowserRetryResponse
import com.armorclaw.shared.domain.model.BrowserQueueStatsResponse
import com.armorclaw.shared.domain.model.AgentStatusResponse
import com.armorclaw.shared.domain.model.AgentStatusHistoryResponse
import com.armorclaw.shared.domain.model.UnsealChallenge
import com.armorclaw.shared.domain.model.UnsealRequest
import com.armorclaw.shared.domain.model.UnsealResult
import com.armorclaw.shared.domain.model.SessionExtensionResult
import com.armorclaw.shared.domain.model.KeystoreStatusResponse

/**
 * Interface for communicating with the ArmorClaw Bridge server
 *
 * This client implements JSON-RPC 2.0 protocol to communicate with the Go bridge server.
 * All methods support OperationContext for correlation ID tracing.
 *
 * The bridge provides:
 * - Core RPC methods (11) - bridge lifecycle, Matrix, WebRTC
 * - Recovery RPC methods (6) - account recovery flow
 * - Platform RPC methods (5) - external platform integration
 * - Agent methods (3) - AI agent management
 * - Workflow methods (3) - workflow templates and execution
 * - HITL methods (3) - human-in-the-loop approvals
 * - Budget methods (1) - budget tracking
 *
 * Total: 32+ RPC methods
 */
interface BridgeRpcClient {

    /**
     * Check if the client is connected to the bridge
     */
    fun isConnected(): Boolean

    /**
     * Get the current session ID if connected
     */
    fun getSessionId(): String?

    /**
     * Set the admin token for authenticated RPC calls (RC-01).
     *
     * After a successful provisioning.claim, the bridge returns an admin_token
     * (prefixed `aat_`). This token must be sent as `Authorization: Bearer <token>`
     * on all subsequent auth-gated RPC methods.
     *
     * @param token The admin token from provisioning.claim, or null to clear.
     */
    suspend fun setAdminToken(token: String?)

    // ========================================================================
    // Bridge Lifecycle Methods (4)
    // ========================================================================

    /**
     * Start a new bridge session
     *
     * @param userId The Matrix user ID
     * @param deviceId The device identifier
     * @param context Operation context for tracing
     * @return Bridge session information including session ID and ICE servers
     */
    suspend fun startBridge(
        userId: String,
        deviceId: String,
        context: OperationContext? = null
    ): RpcResult<BridgeStartResponse>

    /**
     * Get current bridge status
     *
     * @param context Operation context for tracing
     * @return Current bridge status including session and container info
     */
    suspend fun getBridgeStatus(
        context: OperationContext? = null
    ): RpcResult<BridgeStatusResponse>

    /**
     * Stop the current bridge session
     *
     * @param sessionId The session to stop
     * @param context Operation context for tracing
     * @return Success/failure response
     */
    suspend fun stopBridge(
        sessionId: String,
        context: OperationContext? = null
    ): RpcResult<BridgeStopResponse>

    /**
     * Health check for the bridge
     *
     * @param context Operation context for tracing
     * @return Health status
     */
    suspend fun healthCheck(
        context: OperationContext? = null
    ): RpcResult<Map<String, Any?>>

    // ========================================================================
    // Matrix Methods (4) - DEPRECATED: Use MatrixClient instead
    // ========================================================================

    /**
     * Login to Matrix homeserver via bridge
     *
     * @deprecated Use MatrixClient.login() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param homeserver The Matrix homeserver URL
     * @param username User's Matrix username or email
     * @param password User's password
     * @param deviceId Device identifier
     * @param context Operation context for tracing
     * @return Login response with access token and device ID
     */
    @Deprecated(
        message = "Use MatrixClient.login() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.login(homeserver, username, password, deviceId)",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixLogin(
        homeserver: String,
        username: String,
        password: String,
        deviceId: String,
        context: OperationContext? = null
    ): RpcResult<MatrixLoginResponse>

    /**
     * Sync with Matrix homeserver
     *
     * @deprecated Use MatrixClient.startSync() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param since Next batch token from previous sync
     * @param timeout Long poll timeout in milliseconds
     * @param filter Filter to apply
     * @param context Operation context for tracing
     * @return Sync response with rooms, messages, and presence
     */
    @Deprecated(
        message = "Use MatrixClient.startSync() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.startSync()",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixSync(
        since: String? = null,
        timeout: Long = 30000,
        filter: String? = null,
        context: OperationContext? = null
    ): RpcResult<MatrixSyncResponse>

    /**
     * Send a message to a Matrix room
     *
     * @deprecated Use MatrixClient.sendTextMessage() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param roomId The room ID
     * @param eventType The event type (e.g., "m.room.message")
     * @param content The message content
     * @param txnId Transaction ID for idempotency
     * @param context Operation context for tracing
     * @return Send response with event ID
     */
    @Deprecated(
        message = "Use MatrixClient.sendTextMessage() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.sendTextMessage(roomId, text)",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixSend(
        roomId: String,
        eventType: String,
        content: Map<String, Any?>,
        txnId: String? = null,
        context: OperationContext? = null
    ): RpcResult<MatrixSendResponse>

    /**
     * Refresh Matrix access token
     *
     * @deprecated Matrix SDK handles token refresh automatically. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param refreshToken The refresh token
     * @param context Operation context for tracing
     * @return New login response with fresh tokens
     */
    @Deprecated(
        message = "Matrix SDK handles token refresh automatically. See MATRIX_MIGRATION.md for migration guide.",
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixRefreshToken(
        refreshToken: String,
        context: OperationContext? = null
    ): RpcResult<MatrixLoginResponse>

    /**
     * Create a new Matrix room
     *
     * @deprecated Use MatrixClient.createRoom() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param name Room name (optional)
     * @param topic Room topic (optional)
     * @param isDirect Whether this is a direct message room
     * @param invite List of user IDs to invite
     * @param context Operation context for tracing
     * @return Create room response with room ID
     */
    @Deprecated(
        message = "Use MatrixClient.createRoom() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.createRoom(name, topic, isDirect, invite ?: emptyList())",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixCreateRoom(
        name: String? = null,
        topic: String? = null,
        isDirect: Boolean = false,
        invite: List<String>? = null,
        context: OperationContext? = null
    ): RpcResult<MatrixCreateRoomResponse>

    /**
     * Join a Matrix room
     *
     * @deprecated Use MatrixClient.joinRoom() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param roomIdOrAlias Room ID or alias to join
     * @param context Operation context for tracing
     * @return Join room response with room ID
     */
    @Deprecated(
        message = "Use MatrixClient.joinRoom() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.joinRoom(roomIdOrAlias)",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixJoinRoom(
        roomIdOrAlias: String,
        context: OperationContext? = null
    ): RpcResult<MatrixJoinRoomResponse>

    /**
     * Leave a Matrix room
     *
     * @deprecated Use MatrixClient.leaveRoom() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param roomId Room ID to leave
     * @param context Operation context for tracing
     * @return Success/failure
     */
    @Deprecated(
        message = "Use MatrixClient.leaveRoom() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.leaveRoom(roomId)",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixLeaveRoom(
        roomId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * Invite a user to a Matrix room
     *
     * @deprecated Use MatrixClient.inviteUser() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param roomId Room ID
     * @param userId User ID to invite
     * @param context Operation context for tracing
     * @return Success/failure
     */
    @Deprecated(
        message = "Use MatrixClient.inviteUser() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.inviteUser(roomId, userId)",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixInviteUser(
        roomId: String,
        userId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * Send typing notification
     *
     * @deprecated Use MatrixClient.sendTyping() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param roomId Room ID
     * @param typing Whether user is typing
     * @param timeout Typing timeout in milliseconds
     * @param context Operation context for tracing
     * @return Success/failure
     */
    @Deprecated(
        message = "Use MatrixClient.sendTyping() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.sendTyping(roomId, typing, timeout)",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixSendTyping(
        roomId: String,
        typing: Boolean,
        timeout: Long = 30000,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * Send read receipt
     *
     * @deprecated Use MatrixClient.sendReadReceipt() instead. This RPC method will be removed
     * in a future version. Migration guide: MATRIX_MIGRATION.md
     *
     * @param roomId Room ID
     * @param eventId Event ID to mark as read
     * @param context Operation context for tracing
     * @return Success/failure
     */
    @Deprecated(
        message = "Use MatrixClient.sendReadReceipt() instead. See MATRIX_MIGRATION.md for migration guide.",
        replaceWith = ReplaceWith(
            "matrixClient.sendReadReceipt(roomId, eventId)",
            "com.armorclaw.shared.platform.matrix.MatrixClient"
        ),
        level = DeprecationLevel.WARNING
    )
    suspend fun matrixSendReadReceipt(
        roomId: String,
        eventId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // Provisioning Methods (5) — ArmorChat ↔ ArmorClaw first-boot setup
    // ========================================================================

    /**
     * Start a new provisioning session (admin-initiated)
     *
     * Generates a QR code containing server config + setup token.
     * Used by the ArmorClaw setup wizard or by an existing admin
     * to generate onboarding QR codes.
     *
     * @param expiration How long the provisioning session is valid (e.g., "24h")
     * @param context Operation context for tracing
     * @return Provisioning session info including QR data and setup token
     */
    suspend fun provisioningStart(
        expiration: String = "24h",
        context: OperationContext? = null
    ): RpcResult<ProvisioningStartResponse>

    /**
     * Check the status of a provisioning session
     *
     * @param provisioningId The provisioning session ID
     * @param context Operation context for tracing
     * @return Current provisioning status (pending, claimed, expired, cancelled)
     */
    suspend fun provisioningStatus(
        provisioningId: String,
        context: OperationContext? = null
    ): RpcResult<ProvisioningStatusResponse>

    /**
     * Claim admin role using a setup token from QR code
     *
     * This is the critical first-boot flow: when ArmorChat scans the QR code
     * generated during ArmorClaw setup, it calls this method to register itself
     * as the admin device. The first device to claim becomes OWNER.
     *
     * @param setupToken The setup token from the QR code / deep link
     * @param deviceName Human-readable device name (e.g., "Pixel 7 Pro")
     * @param deviceType Device type identifier (e.g., "android", "desktop")
     * @param context Operation context for tracing
     * @return Claim result with admin token and assigned role
     */
    suspend fun provisioningClaim(
        setupToken: String,
        deviceName: String,
        deviceType: String = "android",
        context: OperationContext? = null
    ): RpcResult<ProvisioningClaimResponse>

    /**
     * Rotate the provisioning secret (invalidates existing QR codes)
     *
     * @param context Operation context for tracing
     * @return New provisioning info with fresh setup token
     */
    suspend fun provisioningRotate(
        context: OperationContext? = null
    ): RpcResult<ProvisioningRotateResponse>

    /**
     * Cancel an active provisioning session
     *
     * @param provisioningId The provisioning session ID to cancel
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun provisioningCancel(
        provisioningId: String,
        context: OperationContext? = null
    ): RpcResult<ProvisioningCancelResponse>

    // ========================================================================
    // WebRTC Methods (3)
    // ========================================================================

    /**
     * Create WebRTC offer for voice/video call
     *
     * @param callId The call identifier
     * @param sdpOffer The SDP offer
     * @param context Operation context for tracing
     * @return SDP answer and ICE candidates
     */
    suspend fun webrtcOffer(
        callId: String,
        sdpOffer: String,
        context: OperationContext? = null
    ): RpcResult<WebRtcSignalingResponse>

    /**
     * Process WebRTC answer
     *
     * @param callId The call identifier
     * @param sdpAnswer The SDP answer
     * @param context Operation context for tracing
     * @return ICE candidates
     */
    suspend fun webrtcAnswer(
        callId: String,
        sdpAnswer: String,
        context: OperationContext? = null
    ): RpcResult<WebRtcSignalingResponse>

    /**
     * Send ICE candidate
     *
     * @param callId The call identifier
     * @param candidate The ICE candidate
     * @param sdpMid SDP media ID
     * @param sdpMlineIndex SDP media line index
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun webrtcIceCandidate(
        callId: String,
        candidate: String,
        sdpMid: String?,
        sdpMlineIndex: Int?,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * Hang up an active WebRTC call
     *
     * @param callId The call identifier
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun webrtcHangup(
        callId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // WebRTC Methods - Bridge API Compatible (snake_case)
    // ========================================================================

    /**
     * Start a WebRTC call session
     *
     * Maps to Bridge RPC method: webrtc.start
     *
     * @param roomId The room ID for the call
     * @param callType "audio" or "video"
     * @param context Operation context for tracing
     * @return Call session information
     */
    suspend fun webrtcStart(
        roomId: String,
        callType: String,
        context: OperationContext? = null
    ): RpcResult<WebRtcCallSession>

    /**
     * Send ICE candidate (Bridge API format)
     *
     * Maps to Bridge RPC method: webrtc.ice_candidate
     *
     * @param sessionId The call session ID
     * @param candidate The ICE candidate string
     * @param sdpMid SDP media ID
     * @param sdpMLineIndex SDP media line index
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun webrtcSendIceCandidate(
        sessionId: String,
        candidate: String,
        sdpMid: String?,
        sdpMLineIndex: Int?,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * End a WebRTC call session
     *
     * Maps to Bridge RPC method: webrtc.end
     *
     * @param sessionId The call session ID to end
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun webrtcEnd(
        sessionId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * List active WebRTC call sessions
     *
     * Maps to Bridge RPC method: webrtc.list
     *
     * @param context Operation context for tracing
     * @return List of active call sessions
     */
    suspend fun webrtcList(
        context: OperationContext? = null
    ): RpcResult<List<WebRtcCallSession>>

    // ========================================================================
    // Recovery Methods (6)
    // ========================================================================

    /**
     * Generate a new recovery phrase (BIP39-style 12 words)
     *
     * RPC Method: recovery.generate_phrase
     *
     * @param context Operation context for tracing
     * @return Recovery phrase response
     */
    suspend fun recoveryGeneratePhrase(
        context: OperationContext? = null
    ): RpcResult<RecoveryPhraseResponse>

    /**
     * Store encrypted recovery phrase
     *
     * RPC Method: recovery.store_phrase
     *
     * @param phrase The recovery phrase to store
     * @param context Operation context for tracing
     * @return Success with storage confirmation
     */
    suspend fun recoveryStorePhrase(
        phrase: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * Verify recovery phrase and start recovery flow
     *
     * RPC Method: recovery.verify
     *
     * @param phrase The recovery phrase to verify
     * @param context Operation context for tracing
     * @return Verification result with recovery ID
     */
    suspend fun recoveryVerify(
        phrase: String,
        context: OperationContext? = null
    ): RpcResult<RecoveryVerifyResponse>

    /**
     * Get current recovery status
     *
     * RPC Method: recovery.status
     *
     * @param recoveryId The recovery ID from verify
     * @param context Operation context for tracing
     * @return Recovery status
     */
    suspend fun recoveryStatus(
        recoveryId: String,
        context: OperationContext? = null
    ): RpcResult<RecoveryStatusResponse>

    /**
     * Complete recovery process
     *
     * RPC Method: recovery.complete
     *
     * @param recoveryId The recovery ID
     * @param newDeviceName Name for the new device
     * @param context Operation context for tracing
     * @return Completion result with new device ID
     */
    suspend fun recoveryComplete(
        recoveryId: String,
        newDeviceName: String,
        context: OperationContext? = null
    ): RpcResult<RecoveryCompleteResponse>

    /**
     * Check if current device is still valid
     *
     * RPC Method: recovery.is_device_valid
     *
     * @param deviceId The device ID to check
     * @param context Operation context for tracing
     * @return Device validity status
     */
    suspend fun recoveryIsDeviceValid(
        deviceId: String,
        context: OperationContext? = null
    ): RpcResult<DeviceValidResponse>

    // ========================================================================
    // Platform Methods (5)
    // ========================================================================

    /**
     * Connect an external platform (Slack, Discord, Teams, WhatsApp)
     *
     * @param platformType The platform type
     * @param config Platform-specific configuration
     * @param context Operation context for tracing
     * @return Connection result with auth URL if needed
     */
    suspend fun platformConnect(
        platformType: String,
        config: Map<String, Any?>,
        context: OperationContext? = null
    ): RpcResult<PlatformConnectResponse>

    /**
     * Disconnect a platform
     *
     * @param platformId The platform connection ID
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun platformDisconnect(
        platformId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * List all connected platforms
     *
     * @param context Operation context for tracing
     * @return List of connected platforms
     */
    suspend fun platformList(
        context: OperationContext? = null
    ): RpcResult<PlatformListResponse>

    /**
     * Get status of a specific platform connection
     *
     * @param platformId The platform ID
     * @param context Operation context for tracing
     * @return Platform status
     */
    suspend fun platformStatus(
        platformId: String,
        context: OperationContext? = null
    ): RpcResult<PlatformStatusResponse>

    /**
     * Test platform connection
     *
     * @param platformId The platform ID to test
     * @param context Operation context for tracing
     * @return Test result with latency
     */
    suspend fun platformTest(
        platformId: String,
        context: OperationContext? = null
    ): RpcResult<PlatformTestResponse>

    // ========================================================================
    // Push Notification Methods (3)
    // ========================================================================

    /**
     * Register FCM/APNs push token with the bridge
     *
     * RPC Method: push.register_token
     *
     * The Bridge Server stores this token to send push notifications
     * when the user receives new messages, calls, or other events.
     *
     * @param pushToken The FCM or APNs device token
     * @param pushPlatform The platform type ("fcm" or "apns")
     * @param deviceId The device identifier for multi-device support
     * @param context Operation context for tracing
     * @return Registration result
     */
    suspend fun pushRegister(
        pushToken: String,
        pushPlatform: String,
        deviceId: String,
        context: OperationContext? = null
    ): RpcResult<PushRegisterResponse>

    /**
     * Unregister push token (e.g., on logout)
     *
     * RPC Method: push.unregister_token
     *
     * @param pushToken The token to unregister
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun pushUnregister(
        pushToken: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * Update push notification settings
     *
     * RPC Method: push.update_settings
     *
     * @param enabled Whether push notifications are enabled
     * @param quietHoursStart Quiet hours start time (e.g., "22:00")
     * @param quietHoursEnd Quiet hours end time (e.g., "08:00")
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun pushUpdateSettings(
        enabled: Boolean,
        quietHoursStart: String? = null,
        quietHoursEnd: String? = null,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // License Methods (3)
    // ========================================================================

    /**
     * Get current license status
     *
     * @param context Operation context for tracing
     * @return License status including tier, expiration, and grace period
     */
    suspend fun licenseStatus(
        context: OperationContext? = null
    ): RpcResult<LicenseStatusResponse>

    /**
     * Get available features by tier
     *
     * @param context Operation context for tracing
     * @return Feature availability matrix
     */
    suspend fun licenseFeatures(
        context: OperationContext? = null
    ): RpcResult<LicenseFeaturesResponse>

    /**
     * Check if a specific feature is available
     *
     * @param feature The feature to check
     * @param context Operation context for tracing
     * @return Feature availability
     */
    suspend fun licenseCheckFeature(
        feature: String,
        context: OperationContext? = null
    ): RpcResult<FeatureCheckResponse>

    // ========================================================================
    // Compliance Methods (2)
    // ========================================================================

    /**
     * Get current compliance mode and status
     *
     * @param context Operation context for tracing
     * @return Compliance mode details
     */
    suspend fun complianceStatus(
        context: OperationContext? = null
    ): RpcResult<ComplianceStatusResponse>

    /**
     * Get platform bridging limits
     *
     * @param context Operation context for tracing
     * @return Platform limits by tier
     */
    suspend fun platformLimits(
        context: OperationContext? = null
    ): RpcResult<PlatformLimitsResponse>

    // ========================================================================
    // Error Management Methods (2)
    // ========================================================================

    /**
     * Get recent errors from the bridge
     *
     * RPC Method: get_errors
     *
     * @param limit Maximum number of errors to return
     * @param component Filter by component (optional)
     * @param context Operation context for tracing
     * @return List of recent errors
     */
    suspend fun getErrors(
        limit: Int = 50,
        component: String? = null,
        context: OperationContext? = null
    ): RpcResult<ErrorsResponse>

    /**
     * Resolve an error
     *
     * RPC Method: resolve_error
     *
     * @param errorId The error ID to resolve
     * @param resolution Resolution notes
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun resolveError(
        errorId: String,
        resolution: String? = null,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // Raw RPC Method
    // ========================================================================

    /**
     * Execute a raw RPC call
     *
     * @param method The RPC method name
     * @param params The parameters
     * @param context Operation context for tracing
     * @return Raw result map
     */
    suspend fun <T> call(
        method: String,
        params: Map<String, Any?>? = null,
        context: OperationContext? = null
    ): RpcResult<T>

    // ========================================================================
    // Agent Management Methods (3) - NEW
    // ========================================================================

    /**
     * List all running agents
     *
     * RPC Method: agent.list
     *
     * @param context Operation context for tracing
     * @return List of running agents
     */
    suspend fun agentList(
        context: OperationContext? = null
    ): RpcResult<AgentListResponse>

    /**
     * Get status of a specific agent
     *
     * RPC Method: agent.status
     *
     * @param agentId The agent ID to check
     * @param context Operation context for tracing
     * @return Agent status details
     */
    suspend fun agentStatus(
        agentId: String,
        context: OperationContext? = null
    ): RpcResult<AgentStatusResponse>

    /**
     * Stop a running agent
     *
     * RPC Method: agent.stop
     *
     * @param agentId The agent ID to stop
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun agentStop(
        agentId: String,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // Workflow Methods (3) - NEW
    // ========================================================================

    /**
     * Get available workflow templates
     *
     * RPC Method: workflow.templates
     *
     * @param context Operation context for tracing
     * @return List of available workflow templates
     */
    suspend fun workflowTemplates(
        context: OperationContext? = null
    ): RpcResult<WorkflowTemplatesResponse>

    /**
     * Start a new workflow
     *
     * RPC Method: workflow.start
     *
     * @param templateId The workflow template ID
     * @param params Workflow parameters
     * @param roomId Optional room ID for context
     * @param context Operation context for tracing
     * @return Started workflow info
     */
    suspend fun workflowStart(
        templateId: String,
        params: Map<String, Any?> = emptyMap(),
        roomId: String? = null,
        context: OperationContext? = null
    ): RpcResult<WorkflowStartResponse>

    /**
     * Get workflow status
     *
     * RPC Method: workflow.status
     *
     * @param workflowId The workflow ID to check
     * @param context Operation context for tracing
     * @return Workflow status details
     */
    suspend fun workflowStatus(
        workflowId: String,
        context: OperationContext? = null
    ): RpcResult<WorkflowStatusResponse>

    // ========================================================================
    // HITL (Human-in-the-Loop) Methods (3) - NEW
    // ========================================================================

    /**
     * Get pending HITL approvals
     *
     * RPC Method: hitl.pending
     *
     * @param context Operation context for tracing
     * @return List of pending approvals
     */
    suspend fun hitlPending(
        context: OperationContext? = null
    ): RpcResult<HitlPendingResponse>

    /**
     * Approve a HITL request
     *
     * RPC Method: hitl.approve
     *
     * @param gateId The gate ID to approve
     * @param notes Optional approval notes
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun hitlApprove(
        gateId: String,
        notes: String? = null,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    /**
     * Reject a HITL request
     *
     * RPC Method: hitl.reject
     *
     * @param gateId The gate ID to reject
     * @param reason Optional rejection reason
     * @param context Operation context for tracing
     * @return Success/failure
     */
    suspend fun hitlReject(
        gateId: String,
        reason: String? = null,
        context: OperationContext? = null
    ): RpcResult<Boolean>

    // ========================================================================
    // Budget Methods (1) - NEW
    // ========================================================================

    /**
     * Get current budget status
     *
     * RPC Method: budget.status
     *
     * @param context Operation context for tracing
     * @return Budget usage and limits
     */
    suspend fun budgetStatus(
        context: OperationContext? = null
    ): RpcResult<BudgetStatusResponse>

    // ========================================================================
    // Browser Queue Methods (6) - NEW
    // ========================================================================

    /**
     * Enqueue a new browser automation job
     *
     * RPC Method: browser.enqueue
     *
     * @param agentId The agent ID executing the job
     * @param roomId The Matrix room ID for the job
     * @param url The starting URL for the job
     * @param commands List of browser commands to execute
     * @param priority Job priority (low, normal, high, urgent)
     * @param context Operation context for tracing
     * @return Job ID and queue position
     */
    suspend fun browserEnqueue(
        agentId: String,
        roomId: String,
        url: String,
        commands: List<BrowserCommand>,
        priority: BrowserJobPriority = BrowserJobPriority.NORMAL,
        context: OperationContext? = null
    ): RpcResult<BrowserEnqueueResponse>

    /**
     * Get a browser job by ID
     *
     * RPC Method: browser.get_job
     *
     * @param jobId The job ID to retrieve
     * @param context Operation context for tracing
     * @return Job details
     */
    suspend fun browserGetJob(
        jobId: String,
        context: OperationContext? = null
    ): RpcResult<BrowserJobResponse>

    /**
     * Cancel a browser job
     *
     * RPC Method: browser.cancel
     *
     * @param jobId The job ID to cancel
     * @param context Operation context for tracing
     * @return Cancellation result
     */
    suspend fun browserCancelJob(
        jobId: String,
        context: OperationContext? = null
    ): RpcResult<BrowserCancelResponse>

    /**
     * Retry a failed browser job
     *
     * RPC Method: browser.retry
     *
     * @param jobId The job ID to retry
     * @param context Operation context for tracing
     * @return Retry result with new job ID if created
     */
    suspend fun browserRetryJob(
        jobId: String,
        context: OperationContext? = null
    ): RpcResult<BrowserRetryResponse>

    /**
     * List browser jobs with optional filters
     *
     * RPC Method: browser.list
     *
     * @param status Filter by status (optional)
     * @param agentId Filter by agent ID (optional)
     * @param limit Maximum number of jobs to return
     * @param offset Pagination offset
     * @param context Operation context for tracing
     * @return List of jobs
     */
    suspend fun browserListJobs(
        status: BrowserJobStatus? = null,
        agentId: String? = null,
        limit: Int = 50,
        offset: Int = 0,
        context: OperationContext? = null
    ): RpcResult<BrowserJobListResponse>

    /**
     * Get browser queue statistics
     *
     * RPC Method: browser.stats
     *
     * @param context Operation context for tracing
     * @return Queue statistics
     */
    suspend fun browserQueueStats(
        context: OperationContext? = null
    ): RpcResult<BrowserQueueStatsResponse>

    // ========================================================================
    // Agent Status Methods (4 methods)
    // ========================================================================

    /**
     * Get detailed status of an agent
     *
     * RPC Method: agent.get_status
     *
     * @param agentId The agent ID to query
     * @param context Operation context for tracing
     * @return Detailed agent status
     */
    suspend fun agentGetStatus(
        agentId: String,
        context: OperationContext? = null
    ): RpcResult<AgentStatusResponse>

    /**
     * Get status history for an agent
     *
     * RPC Method: agent.status_history
     *
     * @param agentId The agent ID to query
     * @param limit Maximum number of entries to return
     * @param context Operation context for tracing
     * @return Status history
     */
    suspend fun agentStatusHistory(
        agentId: String,
        limit: Int = 50,
        context: OperationContext? = null
    ): RpcResult<AgentStatusHistoryResponse>

    // ========================================================================
    // Keystore / Zero-Trust Methods (5 methods)
    // ========================================================================

    /**
     * Get current keystore sealed status
     *
     * RPC Method: keystore.sealed
     *
     * @param context Operation context for tracing
     * @return Keystore status
     */
    suspend fun keystoreSealed(
        context: OperationContext? = null
    ): RpcResult<KeystoreStatusResponse>

    /**
     * Generate challenge for unsealing
     *
     * RPC Method: keystore.unseal_challenge
     *
     * @param context Operation context for tracing
     * @return Challenge with nonce and server public key
     */
    suspend fun keystoreUnsealChallenge(
        context: OperationContext? = null
    ): RpcResult<UnsealChallenge>

    /**
     * Respond to unseal challenge
     *
     * RPC Method: keystore.unseal_respond
     *
     * @param request The unseal request with wrapped key
     * @param context Operation context for tracing
     * @return Unseal result
     */
    suspend fun keystoreUnsealRespond(
        request: UnsealRequest,
        context: OperationContext? = null
    ): RpcResult<UnsealResult>

    /**
     * Extend the current unseal session
     *
     * RPC Method: keystore.extend_session
     *
     * @param context Operation context for tracing
     * @return Session extension result
     */
    suspend fun keystoreExtendSession(
        context: OperationContext? = null
    ): RpcResult<SessionExtensionResult>
}
