package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.domain.model.BrowserCommand
import com.armorclaw.shared.domain.model.BrowserJob
import com.armorclaw.shared.domain.model.BrowserJobPriority
import com.armorclaw.shared.domain.model.BrowserJobStatus
import com.armorclaw.shared.domain.model.BrowserEnqueueResponse
import com.armorclaw.shared.domain.model.BrowserJobResponse
import com.armorclaw.shared.domain.model.BrowserJobListResponse
import com.armorclaw.shared.domain.model.BrowserCancelResponse
import com.armorclaw.shared.domain.model.BrowserRetryResponse
import com.armorclaw.shared.domain.model.BrowserQueueStatsResponse
import com.armorclaw.shared.domain.model.AgentStatusHistoryResponse
import com.armorclaw.shared.domain.model.AgentTaskStatus
import com.armorclaw.shared.domain.model.UnsealChallenge
import com.armorclaw.shared.domain.model.UnsealRequest
import com.armorclaw.shared.domain.model.UnsealResult
import com.armorclaw.shared.domain.model.SessionExtensionResult
import com.armorclaw.shared.domain.model.KeystoreStatusResponse
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.*
import kotlin.time.Duration
import kotlin.time.Duration.Companion.days
import kotlin.test.*

/**
 * Tests for InviteService
 */
class InviteServiceTest {

    private lateinit var inviteService: InviteService
    private val testConfig = BridgeConfig(
        baseUrl = "https://test.armorclaw.app",
        homeserverUrl = "https://matrix.test.armorclaw.app",
        timeoutMs = 30000,
        retryCount = 1
    )

    @BeforeTest
    fun setup() {
        inviteService = InviteService(
            rpcClient = MockBridgeRpcClient(),
            config = testConfig
        )
    }

    @Test
    fun testGenerateInviteLink() = runTest {
        val serverConfig = ServerInviteConfig(
            homeserver = "https://matrix.test.com",
            bridgeUrl = "https://bridge.test.com",
            serverName = "Test Server"
        )

        val result = inviteService.generateInviteLink(
            serverConfig = serverConfig,
            expiration = InviteExpiration.SEVEN_DAYS,
            maxUses = 10,
            createdBy = "admin@test.com"
        )

        assertTrue(result is InviteResult.Success)
        val success = result as InviteResult.Success
        assertNotNull(success.invite)
        assertNotNull(success.url)
        assertTrue(success.url!!.contains("invite/"))
        assertEquals(10, success.invite.maxUses)
        assertFalse(success.invite.isExpired)
    }

    @Test
    fun testParseValidInviteUrl() = runTest {
        // First generate an invite
        val serverConfig = ServerInviteConfig(
            homeserver = "https://matrix.test.com",
            bridgeUrl = "https://bridge.test.com",
            serverName = "Test Server"
        )

        val generateResult = inviteService.generateInviteLink(
            serverConfig = serverConfig,
            expiration = InviteExpiration.ONE_DAY,
            createdBy = "admin@test.com"
        ) as InviteResult.Success

        // Then parse it
        val parseResult = inviteService.parseInviteUrl(generateResult.url!!)

        assertTrue(parseResult is InviteParseResult.Valid)
        val valid = parseResult as InviteParseResult.Valid
        assertEquals("https://matrix.test.com", valid.invite.serverConfig.homeserver)
        assertEquals("Test Server", valid.invite.serverConfig.serverName)
    }

    @Test
    fun testInviteExpirationCheck() {
        val now = Clock.System.now()
        val expiredInvite = InviteLink(
            id = "test_expired",
            serverConfig = ServerInviteConfig(
                homeserver = "https://test.com",
                bridgeUrl = "https://bridge.test.com"
            ),
            createdAt = now - Duration.parse("P2D"),
            expiresAt = now - Duration.parse("P1D"),
            maxUses = null,
            currentUses = 0,
            createdBy = "admin",
            isActive = true
        )

        assertTrue(expiredInvite.isExpired)
    }

    @Test
    fun testInviteUsageLimit() {
        val invite = InviteLink(
            id = "test_usage",
            serverConfig = ServerInviteConfig(
                homeserver = "https://test.com",
                bridgeUrl = "https://bridge.test.com"
            ),
            createdAt = Clock.System.now(),
            expiresAt = Clock.System.now() + Duration.parse("P7D"),
            maxUses = 5,
            currentUses = 5,
            createdBy = "admin",
            isActive = true
        )

        assertTrue(invite.isExhausted)
        assertEquals(0, invite.remainingUses)
    }

    @Test
    fun testInviteTimeRemaining() {
        val invite = InviteLink(
            id = "test_time",
            serverConfig = ServerInviteConfig(
                homeserver = "https://test.com",
                bridgeUrl = "https://bridge.test.com"
            ),
            createdAt = Clock.System.now(),
            expiresAt = Clock.System.now() + Duration.parse("PT2H"),
            maxUses = null,
            currentUses = 0,
            createdBy = "admin",
            isActive = true
        )

        val remaining = invite.timeRemaining
        assertTrue(remaining.inWholeMinutes >= 119 && remaining.inWholeMinutes <= 120)
    }

    @Test
    fun testRevokeInvite() = runTest {
        val serverConfig = ServerInviteConfig(
            homeserver = "https://matrix.test.com",
            bridgeUrl = "https://bridge.test.com"
        )

        val generateResult = inviteService.generateInviteLink(
            serverConfig = serverConfig,
            expiration = InviteExpiration.SEVEN_DAYS,
            createdBy = "admin@test.com"
        ) as InviteResult.Success

        assertTrue(generateResult.invite.isActive)

        val revokeResult = inviteService.revokeInviteLink(generateResult.invite.id)
        assertTrue(revokeResult is InviteResult.Success)

        val revoked = (revokeResult as InviteResult.Success).invite
        assertFalse(revoked.isActive)
    }

    @Test
    fun testParseInvalidUrl() {
        val result = inviteService.parseInviteUrl("https://invalid.url/not-an-invite")
        assertTrue(result is InviteParseResult.Error)
    }

    @Test
    fun testGetInvitesByCreator() = runTest {
        val serverConfig = ServerInviteConfig(
            homeserver = "https://matrix.test.com",
            bridgeUrl = "https://bridge.test.com"
        )

        inviteService.generateInviteLink(
            serverConfig = serverConfig,
            expiration = InviteExpiration.SEVEN_DAYS,
            createdBy = "admin1@test.com"
        )

        inviteService.generateInviteLink(
            serverConfig = serverConfig,
            expiration = InviteExpiration.ONE_DAY,
            createdBy = "admin1@test.com"
        )

        inviteService.generateInviteLink(
            serverConfig = serverConfig,
            expiration = InviteExpiration.ONE_HOUR,
            createdBy = "admin2@test.com"
        )

        val admin1Invites = inviteService.getInvitesByCreator("admin1@test.com")
        assertEquals(2, admin1Invites.size)

        val admin2Invites = inviteService.getInvitesByCreator("admin2@test.com")
        assertEquals(1, admin2Invites.size)
    }
}

/**
 * Tests for SetupService
 */
class SetupServiceTest {

    private lateinit var setupService: SetupService
    private lateinit var mockRpcClient: MockBridgeRpcClient
    private lateinit var mockWsClient: MockBridgeWebSocketClient

    @BeforeTest
    fun setup() {
        mockRpcClient = MockBridgeRpcClient()
        mockWsClient = MockBridgeWebSocketClient()
        setupService = SetupService(mockRpcClient, mockWsClient)
    }

    @Test
    fun testInitialSetupState() {
        assertEquals(SetupState.Idle, setupService.setupState.value)
    }

    @Test
    fun testResetSetup() = runTest {
        // Start a setup
        setupService.startSetup("https://matrix.test.com", null)

        // Reset it
        setupService.resetSetup()

        assertEquals(SetupState.Idle, setupService.setupState.value)
        assertNull(setupService.serverInfo.value)
        assertTrue(setupService.securityWarnings.value.isEmpty())
    }

    @Test
    fun testDismissWarning() = runTest {
        val warning = SecurityWarning(
            id = "test_warning",
            type = WarningType.EXTERNAL_SERVER,
            title = "Test Warning",
            message = "This is a test",
            severity = WarningSeverity.LOW,
            canDismiss = true
        )

        // Manually add a warning to test dismissal
        setupService.dismissWarning("test_warning")
        // Warning should be dismissed without error
    }

    @Test
    fun testDeriveBridgeUrl() {
        // The deriveBridgeUrl method is private, but we can test the behavior
        // through startSetup
        runTest {
            val result = setupService.startSetup("https://matrix.example.com", null)
            // Should not throw an exception
        }
    }
}

/**
 * Tests for Security Warning Types
 */
class SecurityWarningTest {

    @Test
    fun testWarningTypes() {
        val types = WarningType.values()
        assertEquals(7, types.size)
        assertTrue(WarningType.EXTERNAL_SERVER in types)
        assertTrue(WarningType.SHARED_IP in types)
        assertTrue(WarningType.UNENCRYPTED_CONNECTION in types)
        assertTrue(WarningType.CERTIFICATE_ISSUE in types)
        assertTrue(WarningType.SERVER_UNVERIFIED in types)
        assertTrue(WarningType.FALLBACK_SERVER in types)
        assertTrue(WarningType.ADMIN_PRIVILEGE_WARNING in types)
    }

    @Test
    fun testWarningSeverities() {
        val severities = WarningSeverity.values()
        assertEquals(4, severities.size)
        assertEquals(WarningSeverity.LOW, severities[0])
        assertEquals(WarningSeverity.MEDIUM, severities[1])
        assertEquals(WarningSeverity.HIGH, severities[2])
        assertEquals(WarningSeverity.CRITICAL, severities[3])
    }

    @Test
    fun testSecurityWarningCreation() {
        val warning = SecurityWarning(
            id = "test_id",
            type = WarningType.SHARED_IP,
            title = "Shared IP Detected",
            message = "Your IP is shared with other users",
            severity = WarningSeverity.HIGH,
            canDismiss = true
        )

        assertEquals("test_id", warning.id)
        assertEquals(WarningType.SHARED_IP, warning.type)
        assertEquals(WarningSeverity.HIGH, warning.severity)
        assertTrue(warning.canDismiss)
        assertFalse(warning.dismissed)
    }
}

/**
 * Tests for Invite Expiration
 */
class InviteExpirationTest {

    @Test
    fun testExpirationDurations() {
        assertEquals(Duration.parse("PT1H"), InviteExpiration.ONE_HOUR.duration)
        assertEquals(Duration.parse("PT6H"), InviteExpiration.SIX_HOURS.duration)
        assertEquals(Duration.parse("P1D"), InviteExpiration.ONE_DAY.duration)
        assertEquals(Duration.parse("P3D"), InviteExpiration.THREE_DAYS.duration)
        assertEquals(Duration.parse("P7D"), InviteExpiration.SEVEN_DAYS.duration)
        assertEquals(Duration.parse("P14D"), InviteExpiration.FOURTEEN_DAYS.duration)
        assertEquals(Duration.parse("P30D"), InviteExpiration.THIRTY_DAYS.duration)
    }
}

/**
 * Tests for Admin Levels
 */
class AdminLevelTest {

    @Test
    fun testAdminLevels() {
        val levels = AdminLevel.values()
        assertEquals(4, levels.size)
        assertEquals(AdminLevel.NONE, levels[0])
        assertEquals(AdminLevel.MODERATOR, levels[1])
        assertEquals(AdminLevel.ADMIN, levels[2])
        assertEquals(AdminLevel.OWNER, levels[3])
    }
}

/**
 * Tests for Setup State
 */
class SetupStateTest {

    @Test
    fun testSetupStateConnecting() {
        assertTrue(SetupState.Connecting.isConnecting)
        assertTrue(SetupState.Discovering.isConnecting)
        assertTrue(SetupState.StartingBridge.isConnecting)
        assertTrue(SetupState.Authenticating.isConnecting)
        assertFalse(SetupState.Idle.isConnecting)
        assertFalse(SetupState.Completed(mockSetupCompleteInfo()).isConnecting)
        assertFalse(SetupState.Error("test").isConnecting)
    }

    @Test
    fun testSetupStateCompleted() {
        val info = mockSetupCompleteInfo()
        val completed = SetupState.Completed(info)
        assertTrue(completed.isCompleted)
        assertFalse(completed.hasError)
    }

    @Test
    fun testSetupStateError() {
        val error = SetupState.Error("Test error")
        assertTrue(error.hasError)
        assertFalse(error.isCompleted)
        assertEquals("Test error", error.message)
    }

    private fun mockSetupCompleteInfo(): SetupCompleteInfo {
        return SetupCompleteInfo(
            userId = "test@example.com",
            deviceId = "TEST123",
            sessionId = "session-123",
            bridgeContainerId = "container-123",
            isAdmin = true,
            adminLevel = AdminLevel.OWNER,
            warnings = emptyList(),
            completedAt = Clock.System.now()
        )
    }
}

// Mock implementations for testing

class MockBridgeRpcClient : BridgeRpcClient {
    override fun isConnected(): Boolean = true
    override fun getSessionId(): String? = "test-session-id"
    override suspend fun setAdminToken(token: String?) { }

    override suspend fun startBridge(userId: String, deviceId: String, context: OperationContext?) =
        RpcResult.success(BridgeStartResponse(
            sessionId = "test-session-id",
            containerId = "container-123",
            status = "running"
        ))

    override suspend fun getBridgeStatus(context: OperationContext?) =
        RpcResult.success(BridgeStatusResponse(
            sessionId = "test-session-id",
            containerId = "container-123",
            status = "running",
            messageCount = 0
        ))

    override suspend fun stopBridge(sessionId: String, context: OperationContext?) =
        RpcResult.success(BridgeStopResponse(true))

    override suspend fun healthCheck(context: OperationContext?) =
        RpcResult.success(mapOf("status" to "healthy", "version" to "1.0.0"))

    override suspend fun matrixLogin(homeserver: String, username: String, password: String, deviceId: String, context: OperationContext?) =
        RpcResult.success(MatrixLoginResponse(
            accessToken = "test-token",
            refreshToken = "test-refresh",
            deviceId = deviceId,
            userId = username
        ))

    override suspend fun matrixSync(since: String?, timeout: Long, filter: String?, context: OperationContext?) =
        RpcResult.success(MatrixSyncResponse(nextBatch = "batch-123"))

    override suspend fun matrixSend(roomId: String, eventType: String, content: Map<String, Any?>, txnId: String?, context: OperationContext?) =
        RpcResult.success(MatrixSendResponse(eventId = "\$event-123"))

    override suspend fun matrixRefreshToken(refreshToken: String, context: OperationContext?) =
        RpcResult.success(MatrixLoginResponse(
            accessToken = "new-token",
            deviceId = "device-123",
            userId = "test@example.com"
        ))

    override suspend fun matrixCreateRoom(name: String?, topic: String?, isDirect: Boolean, invite: List<String>?, context: OperationContext?) =
        RpcResult.success(MatrixCreateRoomResponse(roomId = "!room-123:example.com"))

    override suspend fun matrixJoinRoom(roomIdOrAlias: String, context: OperationContext?) =
        RpcResult.success(MatrixJoinRoomResponse(roomId = "!room-123:example.com"))

    override suspend fun matrixLeaveRoom(roomId: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun matrixInviteUser(roomId: String, userId: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun matrixSendTyping(roomId: String, typing: Boolean, timeout: Long, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun matrixSendReadReceipt(roomId: String, eventId: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun webrtcOffer(callId: String, sdpOffer: String, context: OperationContext?) =
        RpcResult.success(WebRtcSignalingResponse(sdp = "answer-sdp"))

    override suspend fun webrtcAnswer(callId: String, sdpAnswer: String, context: OperationContext?) =
        RpcResult.success(WebRtcSignalingResponse())

    override suspend fun webrtcIceCandidate(callId: String, candidate: String, sdpMid: String?, sdpMlineIndex: Int?, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun webrtcHangup(callId: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun recoveryGeneratePhrase(context: OperationContext?) =
        RpcResult.success(RecoveryPhraseResponse(phrase = "word1 word2 word3", wordCount = 12, created = System.currentTimeMillis()))

    override suspend fun recoveryStorePhrase(phrase: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun recoveryVerify(phrase: String, context: OperationContext?) =
        RpcResult.success(RecoveryVerifyResponse(valid = true, recoveryId = "recovery-123"))

    override suspend fun recoveryStatus(recoveryId: String, context: OperationContext?) =
        RpcResult.success(RecoveryStatusResponse(status = "pending"))

    override suspend fun recoveryComplete(recoveryId: String, newDeviceName: String, context: OperationContext?) =
        RpcResult.success(RecoveryCompleteResponse(success = true, newDeviceId = "device-123"))

    override suspend fun recoveryIsDeviceValid(deviceId: String, context: OperationContext?) =
        RpcResult.success(DeviceValidResponse(valid = true))

    override suspend fun platformConnect(platformType: String, config: Map<String, Any?>, context: OperationContext?) =
        RpcResult.success(PlatformConnectResponse(success = true, platformId = "platform-123"))

    override suspend fun platformDisconnect(platformId: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun platformList(context: OperationContext?) =
        RpcResult.success(PlatformListResponse(platforms = emptyList()))

    override suspend fun platformStatus(platformId: String, context: OperationContext?) =
        RpcResult.success(PlatformStatusResponse(id = "platform-123", type = "slack", status = "connected"))

    override suspend fun platformTest(platformId: String, context: OperationContext?) =
        RpcResult.success(PlatformTestResponse(success = true, latency = 50))

    override suspend fun agentList(context: OperationContext?) =
        RpcResult.success(AgentListResponse(agents = emptyList(), count = 0))

    // agentStatus returns new AgentStatusResponse from domain.model
    override suspend fun agentStatus(agentId: String, context: OperationContext?) =
        RpcResult.success(com.armorclaw.shared.domain.model.AgentStatusResponse(
            agentId = agentId,
            status = AgentTaskStatus.IDLE,
            timestamp = System.currentTimeMillis(),
            metadata = null,
            runningSince = null,
            currentTask = null
        ))

    // New agent status methods (Phase 2)
    override suspend fun agentGetStatus(agentId: String, context: OperationContext?) =
        RpcResult.success(com.armorclaw.shared.domain.model.AgentStatusResponse(
            agentId = agentId,
            status = AgentTaskStatus.IDLE,
            timestamp = System.currentTimeMillis(),
            metadata = null,
            runningSince = null,
            currentTask = null
        ))

    override suspend fun agentStatusHistory(agentId: String, limit: Int, context: OperationContext?) =
        RpcResult.success(AgentStatusHistoryResponse(
            agentId = agentId,
            history = emptyList(),
            totalCount = 0
        ))

    // Keystore methods (Phase 2)
    override suspend fun keystoreSealed(context: OperationContext?) =
        RpcResult.success(KeystoreStatusResponse(
            sealed = true,
            sealState = "sealed",
            remainingSeconds = null,
            sessionExtensionsUsed = null,
            maxExtensions = null,
            lastUnsealedAt = null,
            lastUnsealedBy = null,
            errorMessage = null
        ))

    override suspend fun keystoreUnsealChallenge(context: OperationContext?) =
        RpcResult.success(UnsealChallenge(
            challengeId = "challenge-123",
            nonce = "test-nonce",
            serverPublicKey = "test-public-key",
            expiresAt = System.currentTimeMillis() + 300000,
            keyDerivation = null
        ))

    override suspend fun keystoreUnsealRespond(request: UnsealRequest, context: OperationContext?) =
        RpcResult.success(UnsealResult(
            success = true,
            error = null,
            errorCode = null,
            sessionExpiresAt = System.currentTimeMillis() + 3600000,
            sessionDurationSeconds = 3600
        ))

    override suspend fun keystoreExtendSession(context: OperationContext?) =
        RpcResult.success(SessionExtensionResult(
            success = true,
            newExpiresAt = System.currentTimeMillis() + 7200000,
            error = null,
            maxExtensionsReached = false
        ))

    override suspend fun agentStop(agentId: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun budgetStatus(context: OperationContext?) =
        RpcResult.success(BudgetStatusResponse(
            period = "monthly",
            periodStart = System.currentTimeMillis(),
            periodEnd = System.currentTimeMillis() + 2592000000L,
            currentUsage = 1000L,
            limit = 10000L,
            percentageUsed = 10,
            currency = "USD",
            breakdown = null,
            projectedUsage = null,
            alerting = false
        ))

    override suspend fun complianceStatus(context: OperationContext?) =
        RpcResult.success(ComplianceStatusResponse(
            mode = ComplianceMode.STANDARD,
            phiScrubbing = true,
            auditLogging = true,
            tamperEvidence = true,
            quarantine = false,
            hipaaEnabled = true
        ))

    override suspend fun getErrors(limit: Int, component: String?, context: OperationContext?) =
        RpcResult.success(ErrorsResponse(
            errors = emptyList(),
            total = 0,
            hasMore = false
        ))

    // HITL (Human-in-the-loop) methods
    override suspend fun hitlPending(context: OperationContext?) =
        RpcResult.success(HitlPendingResponse(approvals = emptyList(), count = 0))

    override suspend fun hitlApprove(gateId: String, notes: String?, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun hitlReject(gateId: String, reason: String?, context: OperationContext?) =
        RpcResult.success(true)

    // Workflow methods
    override suspend fun workflowTemplates(context: OperationContext?) =
        RpcResult.success(WorkflowTemplatesResponse(templates = emptyList(), count = 0))

    override suspend fun workflowStart(templateId: String, params: Map<String, Any?>, roomId: String?, context: OperationContext?) =
        RpcResult.success(WorkflowStartResponse(
            workflowId = "wf-123",
            templateId = templateId,
            status = "started",
            estimatedDurationMs = 1000L,
            agentId = null
        ))

    override suspend fun workflowStatus(workflowId: String, context: OperationContext?) =
        RpcResult.success(WorkflowStatusResponse(
            workflowId = workflowId,
            templateId = "template-123",
            status = "running",
            currentStep = 1,
            totalSteps = 5,
            stepName = "Processing",
            startedAt = System.currentTimeMillis(),
            completedAt = null,
            durationMs = null
        ))

    // WebRTC additional methods
    override suspend fun webrtcStart(roomId: String, callType: String, context: OperationContext?) =
        RpcResult.success(WebRtcCallSession(
            sessionId = "session-123",
            roomId = roomId,
            callType = callType,
            startedAt = System.currentTimeMillis(),
            startedBy = "user-123",
            status = "pending",
            participants = null,
            sdpOffer = "offer-sdp",
            sdpAnswer = null
        ))

    override suspend fun webrtcSendIceCandidate(sessionId: String, candidate: String, sdpMid: String?, sdpMlineIndex: Int?, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun webrtcEnd(sessionId: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun webrtcList(context: OperationContext?) =
        RpcResult.success(listOf<WebRtcCallSession>())

    // Push notification methods
    override suspend fun pushRegister(pushToken: String, pushPlatform: String, deviceId: String, context: OperationContext?) =
        RpcResult.success(PushRegisterResponse(success = true, deviceId = deviceId))

    override suspend fun pushUnregister(pushToken: String, context: OperationContext?) =
        RpcResult.success(true)

    override suspend fun pushUpdateSettings(enabled: Boolean, quietHoursStart: String?, quietHoursEnd: String?, context: OperationContext?) =
        RpcResult.success(true)

    // License methods
    override suspend fun licenseStatus(context: OperationContext?) =
        RpcResult.success(LicenseStatusResponse(
            valid = true,
            tier = "pro",
            expiresAt = System.currentTimeMillis() + 31536000000L,
            gracePeriodRemaining = null,
            instanceId = "instance-123",
            maxInstances = 5,
            features = listOf("feature1", "feature2"),
            warning = null
        ))

    override suspend fun licenseFeatures(context: OperationContext?) =
        RpcResult.success(LicenseFeaturesResponse(
            tier = "pro",
            features = mapOf("feature1" to FeatureInfo(available = true), "feature2" to FeatureInfo(available = true)),
            compliance = ComplianceMode.STANDARD
        ))

    override suspend fun licenseCheckFeature(feature: String, context: OperationContext?) =
        RpcResult.success(FeatureCheckResponse(
            feature = feature,
            available = true,
            reason = null,
            limit = null,
            current = null
        ))

    // Platform limits
    override suspend fun platformLimits(context: OperationContext?) =
        RpcResult.success(PlatformLimitsResponse(platforms = emptyMap()))

    // Resolve error
    override suspend fun resolveError(errorId: String, resolution: String?, context: OperationContext?) =
        RpcResult.success(true)

    // Provisioning methods
    override suspend fun provisioningStart(expiration: String, context: OperationContext?) =
        RpcResult.success(ProvisioningStartResponse(
            provisioningId = "prov-123",
            setupToken = "setup-token",
            qrData = "qr-data",
            expiresAt = System.currentTimeMillis() + 3600000
        ))

    override suspend fun provisioningStatus(provisioningId: String, context: OperationContext?) =
        RpcResult.success(ProvisioningStatusResponse(
            provisioningId = provisioningId,
            status = "pending",
            expiresAt = System.currentTimeMillis() + 3600000
        ))

    override suspend fun provisioningClaim(setupToken: String, deviceName: String, deviceType: String, context: OperationContext?) =
        RpcResult.success(ProvisioningClaimResponse(
            success = true,
            deviceId = "device-123",
            adminToken = "admin-token",
            userId = "user-123"
        ))

    override suspend fun provisioningRotate(context: OperationContext?) =
        RpcResult.success(ProvisioningRotateResponse(
            success = true,
            newSetupToken = "new-token",
            newQrData = "new-qr-data"
        ))

    override suspend fun provisioningCancel(provisioningId: String, context: OperationContext?) =
        RpcResult.success(ProvisioningCancelResponse(success = true))

    // Browser queue methods
    override suspend fun browserEnqueue(agentId: String, roomId: String, url: String, commands: List<BrowserCommand>, priority: BrowserJobPriority, context: OperationContext?) =
        RpcResult.success(BrowserEnqueueResponse(
            jobId = "job-123",
            status = BrowserJobStatus.PENDING,
            queuePosition = 1
        ))

    override suspend fun browserGetJob(jobId: String, context: OperationContext?) =
        RpcResult.success(BrowserJobResponse(
            job = BrowserJob(
                jobId = jobId,
                agentId = "agent-123",
                roomId = "!room-123:example.com",
                url = "https://example.com",
                commands = emptyList(),
                status = BrowserJobStatus.RUNNING,
                priority = BrowserJobPriority.NORMAL,
                createdAt = System.currentTimeMillis(),
                updatedAt = System.currentTimeMillis()
            )
        ))

    override suspend fun browserCancelJob(jobId: String, context: OperationContext?) =
        RpcResult.success(BrowserCancelResponse(
            jobId = jobId,
            cancelled = true,
            status = BrowserJobStatus.CANCELLED
        ))

    override suspend fun browserRetryJob(jobId: String, context: OperationContext?) =
        RpcResult.success(BrowserRetryResponse(
            jobId = jobId,
            retried = true,
            newJobId = "job-456"
        ))

    override suspend fun browserListJobs(status: BrowserJobStatus?, agentId: String?, limit: Int, offset: Int, context: OperationContext?) =
        RpcResult.success(BrowserJobListResponse(
            jobs = emptyList(),
            total = 0
        ))

    override suspend fun browserQueueStats(context: OperationContext?) =
        RpcResult.success(BrowserQueueStatsResponse(
            pending = 0,
            running = 0,
            paused = 0,
            completed = 0,
            failed = 0,
            cancelled = 0,
            total = 0,
            activeWorkers = 0
        ))

    override suspend fun <T> call(method: String, params: Map<String, Any?>?, context: OperationContext?): RpcResult<T> {
        @Suppress("UNCHECKED_CAST")
        return RpcResult.success(null as T)
    }
}

class MockBridgeWebSocketClient : BridgeWebSocketClient {
    private val _connectionState = MutableStateFlow<WebSocketState>(WebSocketState.Disconnected)
    private val _events = MutableSharedFlow<BridgeEvent>(extraBufferCapacity = 100)

    override val connectionState: StateFlow<WebSocketState> = _connectionState.asStateFlow()
    override val events: Flow<BridgeEvent> = _events.asSharedFlow()
    override val errors: Flow<Throwable> = flowOf()

    override fun isConnected(): Boolean = _connectionState.value == WebSocketState.Connected

    override suspend fun connect(sessionId: String, accessToken: String?, context: OperationContext?): Boolean {
        _connectionState.value = WebSocketState.Connected
        return true
    }

    override suspend fun disconnect(reason: String?) {
        _connectionState.value = WebSocketState.Disconnected
    }

    override suspend fun subscribeToRoom(roomId: String, context: OperationContext?) {}
    override suspend fun unsubscribeFromRoom(roomId: String, context: OperationContext?) {}
    override suspend fun subscribeToPresence(userIds: List<String>, context: OperationContext?) {}
    override suspend fun sendTypingNotification(roomId: String, typing: Boolean, context: OperationContext?) {}
    override suspend fun sendReadReceipt(roomId: String, eventId: String, context: OperationContext?) {}
    override suspend fun ping() {}

    override fun <T : BridgeEvent> getEventsOfType(eventClass: Class<T>): Flow<T> = flowOf()
    override fun getMessageEvents(): Flow<BridgeEvent.MessageReceived> = flowOf()
    override fun getTypingEvents(): Flow<BridgeEvent.TypingNotification> = flowOf()
    override fun getPresenceEvents(): Flow<BridgeEvent.PresenceUpdate> = flowOf()
    override fun getCallEvents(): Flow<BridgeEvent.CallEvent> = flowOf()
    override fun getRoomEvents(): Flow<BridgeEvent> = flowOf()
}
