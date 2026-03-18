package com.armorclaw.shared.data.store

import com.armorclaw.shared.domain.repository.AgentRepository
import com.armorclaw.shared.domain.repository.WorkflowInfo
import com.armorclaw.shared.domain.repository.WorkflowRepository
import com.armorclaw.shared.domain.repository.WorkflowStatus
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.matrix.SyncState
import com.armorclaw.shared.platform.matrix.ConnectionState
import com.armorclaw.shared.platform.matrix.event.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Tests for ControlPlaneStore
 *
 * These tests verify that ArmorClaw-specific events (workflows, agents)
 * are correctly processed and state is updated.
 */
@RunWith(RobolectricTestRunner::class)
@Config(manifest = Config.NONE, sdk = [28])
class ControlPlaneStoreTest {

    private lateinit var store: ControlPlaneStore
    private lateinit var mockMatrixClient: MockMatrixClient
    private lateinit var mockWorkflowRepository: MockWorkflowRepository
    private lateinit var mockAgentRepository: MockAgentRepository
    private lateinit var testScope: CoroutineScope
    private val json = Json { ignoreUnknownKeys = true }

    @Before
    fun setup() {
        mockMatrixClient = MockMatrixClient()
        mockWorkflowRepository = MockWorkflowRepository()
        mockAgentRepository = MockAgentRepository()
        testScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    }

    private fun createStore(): ControlPlaneStore {
        return ControlPlaneStore(
            matrixClient = mockMatrixClient,
            workflowRepository = mockWorkflowRepository,
            agentRepository = mockAgentRepository,
            json = json,
            externalScope = testScope
        )
    }

    @After
    fun tearDown() {
        testScope.cancel()
    }

    // ========================================================================
    // Workflow Event Tests
    // ========================================================================

    @Test
    fun `workflow started event should update activeWorkflows`() = runBlocking {
        store = createStore()
        // Give the collector time to start
        delay(50)
        
        val event = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent:example.com",
            type = ArmorClawEventType.WORKFLOW_STARTED,
            content = json.encodeToString(WorkflowStartedEvent(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )

        mockMatrixClient.emitEvent(event)
        
        // Wait for the workflow to appear in activeWorkflows
        withTimeout(1000) {
            val workflows = store.activeWorkflows.first { it.isNotEmpty() }
            assertEquals("Should have 1 active workflow", 1, workflows.size)
            assertEquals("Workflow ID should match", "wf_123", workflows.first().workflowId)
            assertEquals("Workflow type should match", "document_analysis", workflows.first().workflowType)
        }
    }

    @Test
    fun `workflow step event should update workflow state`() = runBlocking {
        store = createStore()
        delay(50)
        
        // First emit a workflow started event
        val startEvent = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent:example.com",
            type = ArmorClawEventType.WORKFLOW_STARTED,
            content = json.encodeToString(WorkflowStartedEvent(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(startEvent)
        withTimeout(1000) {
            store.activeWorkflows.first { it.isNotEmpty() }
        }

        // Then emit a step event
        val stepEvent = MatrixEvent(
            eventId = "\$event_2",
            roomId = "!room:example.com",
            senderId = "@agent:example.com",
            type = ArmorClawEventType.WORKFLOW_STEP,
            content = json.encodeToString(WorkflowStepEvent(
                workflowId = "wf_123",
                stepId = "step_1",
                stepName = "Analyzing document",
                stepIndex = 1,
                totalSteps = 3,
                status = StepStatus.RUNNING,
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(stepEvent)
        
        withTimeout(1000) {
            val workflows = store.activeWorkflows.first { 
                it.isNotEmpty() && it.first() is WorkflowState.StepRunning 
            }
            assertEquals("Should have 1 active workflow", 1, workflows.size)
            val workflow = workflows.first() as? WorkflowState.StepRunning
            assertNotNull("Workflow should be in StepRunning state", workflow)
            assertEquals("Step name should match", "Analyzing document", workflow?.stepName)
            assertEquals("Step status should be RUNNING", StepStatus.RUNNING, workflow?.status)
        }
    }

    @Test
    fun `workflow completed event should remove from activeWorkflows`() = runBlocking {
        store = createStore()
        delay(50)
        
        // First emit a workflow started event
        val startEvent = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent:example.com",
            type = ArmorClawEventType.WORKFLOW_STARTED,
            content = json.encodeToString(WorkflowStartedEvent(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(startEvent)
        withTimeout(1000) {
            store.activeWorkflows.first { it.isNotEmpty() }
        }
        assertEquals("Should have 1 active workflow", 1, store.activeWorkflows.value.size)

        // Then emit a completed event
        val completeEvent = MatrixEvent(
            eventId = "\$event_2",
            roomId = "!room:example.com",
            senderId = "@agent:example.com",
            type = ArmorClawEventType.WORKFLOW_COMPLETED,
            content = json.encodeToString(WorkflowCompletedEvent(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                success = true,
                result = "Analysis complete",
                duration = 5000L,
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(completeEvent)
        
        withTimeout(1000) {
            store.activeWorkflows.first { it.isEmpty() }
        }
        assertEquals("Should have 0 active workflows after completion", 0, store.activeWorkflows.value.size)
    }

    // ========================================================================
    // Agent Event Tests
    // ========================================================================

    @Test
    fun `agent task started event should update agentTasks`() = runBlocking {
        store = createStore()
        delay(50)
        
        val event = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent_analysis:example.com",
            type = ArmorClawEventType.AGENT_TASK_STARTED,
            content = json.encodeToString(AgentTaskStartedEvent(
                agentId = "@agent_analysis:example.com",
                agentName = "Analysis Agent",
                taskId = "task_123",
                taskType = "document_analysis",
                roomId = "!room:example.com",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )

        mockMatrixClient.emitEvent(event)
        
        withTimeout(1000) {
            val tasks = store.agentTasks.first { it.isNotEmpty() }
            assertEquals("Should have 1 active task", 1, tasks.size)
            val task = tasks.first() as? AgentTaskState.Running
            assertNotNull("Task should be in Running state", task)
            assertEquals("Task ID should match", "task_123", task?.taskId)
            assertEquals("Agent name should match", "Analysis Agent", task?.agentName)
        }
    }

    @Test
    fun `agent thinking event should update thinkingAgents`() = runBlocking {
        store = createStore()
        delay(50)
        
        val event = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent_analysis:example.com",
            type = ArmorClawEventType.AGENT_THINKING,
            content = json.encodeToString(AgentThinkingEvent(
                agentId = "@agent_analysis:example.com",
                agentName = "Analysis Agent",
                message = "Processing document...",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )

        mockMatrixClient.emitEvent(event)
        
        withTimeout(1000) {
            val thinking = store.thinkingAgents.first { it.isNotEmpty() }
            assertTrue("Should have thinking agent", thinking.containsKey("@agent_analysis:example.com"))
            assertEquals("Agent name should match", "Analysis Agent", thinking["@agent_analysis:example.com"]?.agentName)
            assertTrue("Should be thinking", store.isAgentThinking("@agent_analysis:example.com"))
        }
    }

    @Test
    fun `agent task complete event should remove from agentTasks`() = runBlocking {
        store = createStore()
        delay(50)
        
        // First emit a task started event
        val startEvent = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent_analysis:example.com",
            type = ArmorClawEventType.AGENT_TASK_STARTED,
            content = json.encodeToString(AgentTaskStartedEvent(
                agentId = "@agent_analysis:example.com",
                agentName = "Analysis Agent",
                taskId = "task_123",
                taskType = "document_analysis",
                roomId = "!room:example.com",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(startEvent)
        withTimeout(1000) {
            store.agentTasks.first { it.isNotEmpty() }
        }
        assertEquals("Should have 1 active task", 1, store.agentTasks.value.size)

        // Then emit a complete event
        val completeEvent = MatrixEvent(
            eventId = "\$event_2",
            roomId = "!room:example.com",
            senderId = "@agent_analysis:example.com",
            type = ArmorClawEventType.AGENT_TASK_COMPLETE,
            content = json.encodeToString(AgentTaskCompleteEvent(
                agentId = "@agent_analysis:example.com",
                agentName = "Analysis Agent",
                taskId = "task_123",
                taskType = "document_analysis",
                success = true,
                result = "Analysis complete",
                duration = 5000L,
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(completeEvent)
        
        withTimeout(1000) {
            store.agentTasks.first { it.isEmpty() }
        }
        assertEquals("Should have 0 active tasks after completion", 0, store.agentTasks.value.size)
    }

    // ========================================================================
    // System Event Tests
    // ========================================================================

    @Test
    fun `budget warning event should update budgetWarnings`() = runBlocking {
        store = createStore()
        delay(50)
        
        val event = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@system:example.com",
            type = ArmorClawEventType.BUDGET_WARNING,
            content = json.encodeToString(BudgetWarningEvent(
                userId = "@user:example.com",
                currentSpend = 75.0,
                limit = 100.0,
                percentageUsed = 75.0,
                warningLevel = WarningLevel.WARNING,
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )

        mockMatrixClient.emitEvent(event)
        
        withTimeout(1000) {
            val warnings = store.budgetWarnings.first { it.isNotEmpty() }
            assertEquals("Should have 1 budget warning", 1, warnings.size)
            assertEquals("Warning level should be WARNING", WarningLevel.WARNING, warnings.first().warningLevel)
            assertEquals("Percentage used should be 75%", 75.0, warnings.first().percentageUsed, 0.01)
        }
    }

    @Test
    fun `bridge connected event should update bridgeStatus`() = runBlocking {
        store = createStore()
        delay(50)
        
        val event = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@system:example.com",
            type = ArmorClawEventType.BRIDGE_CONNECTED,
            content = json.encodeToString(BridgeConnectedEvent(
                platformType = "slack",
                platformName = "Slack Workspace",
                status = "connected",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )

        mockMatrixClient.emitEvent(event)
        
        withTimeout(1000) {
            val status = store.bridgeStatus.first { it.isNotEmpty() }
            assertTrue("Should have Slack bridge status", status.containsKey("slack"))
            val bridgeStatus = status["slack"] as? BridgeStatusState.Connected
            assertNotNull("Bridge should be connected", bridgeStatus)
            assertEquals("Platform name should match", "Slack Workspace", bridgeStatus?.platformName)
        }
    }

    // ========================================================================
    // Utility Method Tests
    // ========================================================================

    @Test
    fun `getWorkflowState should return correct workflow`() = runBlocking {
        store = createStore()
        delay(50)
        
        // Emit workflow started event
        val event = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent:example.com",
            type = ArmorClawEventType.WORKFLOW_STARTED,
            content = json.encodeToString(WorkflowStartedEvent(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(event)
        withTimeout(1000) {
            store.activeWorkflows.first { it.isNotEmpty() }
        }

        val state = store.getWorkflowState("wf_123")
        assertNotNull("Should find workflow state", state)
        assertEquals("Workflow ID should match", "wf_123", state?.workflowId)
    }

    @Test
    fun `clear should reset all state`() = runBlocking {
        store = createStore()
        delay(50)

        // Add some state
        val event = MatrixEvent(
            eventId = "\$event_1",
            roomId = "!room:example.com",
            senderId = "@agent:example.com",
            type = ArmorClawEventType.WORKFLOW_STARTED,
            content = json.encodeToString(WorkflowStartedEvent(
                workflowId = "wf_123",
                workflowType = "document_analysis",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            )),
            timestamp = System.currentTimeMillis()
        )
        mockMatrixClient.emitEvent(event)
        withTimeout(1000) {
            store.activeWorkflows.first { it.isNotEmpty() }
        }
        assertEquals("Should have 1 active workflow", 1, store.activeWorkflows.value.size)

        // Clear
        store.clear()

        assertEquals("Should have 0 active workflows after clear", 0, store.activeWorkflows.value.size)
        assertEquals("Should have 0 agent tasks after clear", 0, store.agentTasks.value.size)
        assertEquals("Should have 0 thinking agents after clear", 0, store.thinkingAgents.value.size)
        assertEquals("Should have 0 budget warnings after clear", 0, store.budgetWarnings.value.size)
        assertEquals("Should have 0 bridge statuses after clear", 0, store.bridgeStatus.value.size)
    }

    // ========================================================================
    // Phase 2 - Agent Status Tests
    // ========================================================================

    @Test
    fun `processStatusEvent should update agentStatuses`() = runBlocking {
        store = createStore()
        delay(50)

        val event = com.armorclaw.shared.domain.model.AgentTaskStatusEvent(
            agentId = "agent_browse_001",
            status = com.armorclaw.shared.domain.model.AgentTaskStatus.FORM_FILLING,
            timestamp = System.currentTimeMillis(),
            metadata = mapOf("step" to "2")
        )

        store.processStatusEvent(event)

        val status = store.getAgentStatus("agent_browse_001")
        assertNotNull("Should have status for agent", status)
        assertEquals("Status should be FORM_FILLING", com.armorclaw.shared.domain.model.AgentTaskStatus.FORM_FILLING, status?.status)
        assertEquals("Metadata should contain step", "2", status?.metadata?.get("step"))
    }

    @Test
    fun `processStatusEvent with IDLE should remove from agentStatuses`() = runBlocking {
        store = createStore()
        delay(50)

        // First add an active status
        val activeEvent = com.armorclaw.shared.domain.model.AgentTaskStatusEvent(
            agentId = "agent_001",
            status = com.armorclaw.shared.domain.model.AgentTaskStatus.BROWSING,
            timestamp = System.currentTimeMillis()
        )
        store.processStatusEvent(activeEvent)
        assertNotNull("Should have active status", store.getAgentStatus("agent_001"))

        // Then set to IDLE
        val idleEvent = com.armorclaw.shared.domain.model.AgentTaskStatusEvent(
            agentId = "agent_001",
            status = com.armorclaw.shared.domain.model.AgentTaskStatus.IDLE,
            timestamp = System.currentTimeMillis()
        )
        store.processStatusEvent(idleEvent)

        assertNull("Should not have status for IDLE agent", store.getAgentStatus("agent_001"))
    }

    @Test
    fun `getActiveAgentTaskStatuses should return only active agents`() = runBlocking {
        store = createStore()
        delay(50)

        // Add multiple statuses
        store.processStatusEvent(com.armorclaw.shared.domain.model.AgentTaskStatusEvent(
            agentId = "agent_001",
            status = com.armorclaw.shared.domain.model.AgentTaskStatus.BROWSING,
            timestamp = System.currentTimeMillis()
        ))
        store.processStatusEvent(com.armorclaw.shared.domain.model.AgentTaskStatusEvent(
            agentId = "agent_002",
            status = com.armorclaw.shared.domain.model.AgentTaskStatus.FORM_FILLING,
            timestamp = System.currentTimeMillis()
        ))

        val activeStatuses = store.getActiveAgentStatuses()
        assertEquals("Should have 2 active agents", 2, activeStatuses.size)
    }

    // ========================================================================
    // Phase 2 - PII Access Request Tests
    // ========================================================================

    @Test
    fun `addPiiRequest should update pendingPiiRequests`() = runBlocking {
        store = createStore()
        delay(50)

        val request = com.armorclaw.shared.domain.model.PiiAccessRequest(
            requestId = "req_001",
            agentId = "agent_001",
            fields = listOf(
                com.armorclaw.shared.domain.model.PiiField(
                    name = "Email",
                    sensitivity = com.armorclaw.shared.domain.model.SensitivityLevel.LOW,
                    description = "Contact info"
                )
            ),
            reason = "Test request",
            expiresAt = System.currentTimeMillis() + 60000
        )

        store.addPiiRequest(request)

        val pending = store.getAllPendingPiiRequests()
        assertEquals("Should have 1 pending request", 1, pending.size)
        assertEquals("Request ID should match", "req_001", pending.first().requestId)
    }

    @Test
    fun `removePiiRequest should remove from pendingPiiRequests`() = runBlocking {
        store = createStore()
        delay(50)

        val request = com.armorclaw.shared.domain.model.PiiAccessRequest(
            requestId = "req_001",
            agentId = "agent_001",
            fields = emptyList(),
            reason = "Test",
            expiresAt = System.currentTimeMillis() + 60000
        )

        store.addPiiRequest(request)
        assertEquals("Should have 1 pending request", 1, store.getAllPendingPiiRequests().size)

        store.removePiiRequest("req_001")
        assertEquals("Should have 0 pending requests", 0, store.getAllPendingPiiRequests().size)
    }

    @Test
    fun `getPendingPiiRequests should filter by agentId`() = runBlocking {
        store = createStore()
        delay(50)

        store.addPiiRequest(com.armorclaw.shared.domain.model.PiiAccessRequest(
            requestId = "req_001",
            agentId = "agent_001",
            fields = emptyList(),
            reason = "Test 1",
            expiresAt = System.currentTimeMillis() + 60000
        ))
        store.addPiiRequest(com.armorclaw.shared.domain.model.PiiAccessRequest(
            requestId = "req_002",
            agentId = "agent_002",
            fields = emptyList(),
            reason = "Test 2",
            expiresAt = System.currentTimeMillis() + 60000
        ))

        val agent1Requests = store.getPendingPiiRequests("agent_001")
        assertEquals("Should have 1 request for agent_001", 1, agent1Requests.size)
        assertEquals("Request ID should match", "req_001", agent1Requests.first().requestId)
    }

    // ========================================================================
    // Phase 2 - Keystore Status Tests
    // ========================================================================

    @Test
    fun `setKeystoreStatus should update keystoreStatus`() = runBlocking {
        store = createStore()
        delay(50)

        val unsealed = com.armorclaw.shared.domain.model.KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + 3600000,
            unsealedBy = com.armorclaw.shared.domain.model.UnsealMethod.PASSWORD
        )

        store.setKeystoreStatus(unsealed)

        val status = store.keystoreStatus.value
        assertTrue("Status should be Unsealed", status is com.armorclaw.shared.domain.model.KeystoreStatus.Unsealed)
        assertTrue("Keystore should be accessible", store.isKeystoreAccessible())
    }

    @Test
    fun `resealKeystore should set status to Sealed`() = runBlocking {
        store = createStore()
        delay(50)

        // First unseal
        store.setKeystoreStatus(com.armorclaw.shared.domain.model.KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + 3600000,
            unsealedBy = com.armorclaw.shared.domain.model.UnsealMethod.PASSWORD
        ))
        assertTrue("Should be accessible", store.isKeystoreAccessible())

        // Reseal
        store.resealKeystore()

        val status = store.keystoreStatus.value
        assertTrue("Status should be Sealed", status is com.armorclaw.shared.domain.model.KeystoreStatus.Sealed)
        assertFalse("Should not be accessible", store.isKeystoreAccessible())
    }

    @Test
    fun `processKeystoreEvent should handle unsealed event`() = runBlocking {
        store = createStore()
        delay(50)

        val expiresAt = System.currentTimeMillis() + 3600000
        store.processKeystoreEvent("com.armorclaw.keystore.unsealed", mapOf(
            "expiresAt" to expiresAt,
            "method" to "BIOMETRIC"
        ))

        val status = store.keystoreStatus.value as? com.armorclaw.shared.domain.model.KeystoreStatus.Unsealed
        assertNotNull("Status should be Unsealed", status)
        assertEquals("Unseal method should be BIOMETRIC", com.armorclaw.shared.domain.model.UnsealMethod.BIOMETRIC, status?.unsealedBy)
    }

    @Test
    fun `processKeystoreEvent should handle sealed event`() = runBlocking {
        store = createStore()
        delay(50)

        // First unseal
        store.setKeystoreStatus(com.armorclaw.shared.domain.model.KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + 3600000,
            unsealedBy = com.armorclaw.shared.domain.model.UnsealMethod.PASSWORD
        ))

        // Process sealed event
        store.processKeystoreEvent("com.armorclaw.keystore.sealed", emptyMap())

        assertTrue("Status should be Sealed", store.keystoreStatus.value is com.armorclaw.shared.domain.model.KeystoreStatus.Sealed)
    }

    @Test
    fun `processKeystoreEvent should handle error event`() = runBlocking {
        store = createStore()
        delay(50)

        store.processKeystoreEvent("com.armorclaw.keystore.error", mapOf(
            "message" to "Decryption failed"
        ))

        val status = store.keystoreStatus.value as? com.armorclaw.shared.domain.model.KeystoreStatus.Error
        assertNotNull("Status should be Error", status)
        assertEquals("Error message should match", "Decryption failed", status?.message)
    }

    @Test
    fun `clear should reset keystore status to Sealed`() = runBlocking {
        store = createStore()
        delay(50)

        // Unseal first
        store.setKeystoreStatus(com.armorclaw.shared.domain.model.KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + 3600000,
            unsealedBy = com.armorclaw.shared.domain.model.UnsealMethod.PASSWORD
        ))

        // Clear
        store.clear()

        assertTrue("Status should be Sealed after clear", store.keystoreStatus.value is com.armorclaw.shared.domain.model.KeystoreStatus.Sealed)
    }
}

// ========================================================================
// Mock Classes
// ========================================================================

class MockMatrixClient : MatrixClient {
    private val _events = kotlinx.coroutines.flow.MutableSharedFlow<MatrixEvent>(extraBufferCapacity = 64)

    suspend fun emitEvent(event: MatrixEvent) {
        _events.emit(event)
    }

    override val syncState = kotlinx.coroutines.flow.MutableStateFlow<SyncState>(SyncState.Idle)
    override val isLoggedIn = kotlinx.coroutines.flow.MutableStateFlow(false)
    override val currentUser = kotlinx.coroutines.flow.MutableStateFlow<com.armorclaw.shared.domain.model.User?>(null)
    override val connectionState = kotlinx.coroutines.flow.MutableStateFlow<ConnectionState>(ConnectionState.Offline)
    override val rooms = kotlinx.coroutines.flow.MutableStateFlow<List<com.armorclaw.shared.domain.model.Room>>(emptyList())

    override suspend fun login(homeserver: String, username: String, password: String, deviceId: String?) = Result.success(
        com.armorclaw.shared.platform.matrix.MatrixSession(
            userId = "@$username:example.com",
            deviceId = deviceId ?: "TEST",
            accessToken = "token",
            homeserver = homeserver
        )
    )
    override suspend fun loginWithWellKnown(serverName: String, username: String, password: String): Result<com.armorclaw.shared.platform.matrix.MatrixSession> = Result.failure(NotImplementedError())
    override suspend fun restoreSession(session: com.armorclaw.shared.platform.matrix.MatrixSession): Result<Unit> = Result.success(Unit)
    override suspend fun logout(): Result<Unit> = Result.success(Unit)
    override fun startSync() {}
    override fun stopSync() {}
    override suspend fun syncOnce(): Result<Unit> = Result.success(Unit)
    override suspend fun getRoom(roomId: String): com.armorclaw.shared.domain.model.Room? = null
    override fun observeRoom(roomId: String): kotlinx.coroutines.flow.Flow<com.armorclaw.shared.domain.model.Room> = kotlinx.coroutines.flow.emptyFlow<com.armorclaw.shared.domain.model.Room>()
    override suspend fun createRoom(name: String?, topic: String?, isDirect: Boolean, invite: List<String>, isEncrypted: Boolean): Result<com.armorclaw.shared.domain.model.Room> = Result.failure(NotImplementedError())
    override suspend fun joinRoom(roomIdOrAlias: String): Result<com.armorclaw.shared.domain.model.Room> = Result.failure(NotImplementedError())
    override suspend fun leaveRoom(roomId: String): Result<Unit> = Result.success(Unit)
    override suspend fun inviteUser(roomId: String, userId: String): Result<Unit> = Result.success(Unit)
    override suspend fun kickUser(roomId: String, userId: String, reason: String?): Result<Unit> = Result.success(Unit)
    override suspend fun setRoomName(roomId: String, name: String): Result<Unit> = Result.success(Unit)
    override suspend fun setRoomTopic(roomId: String, topic: String): Result<Unit> = Result.success(Unit)
    override suspend fun getMessages(roomId: String, limit: Int, fromToken: String?): Result<com.armorclaw.shared.platform.matrix.MessageBatch> = Result.success(
        com.armorclaw.shared.platform.matrix.MessageBatch(emptyList())
    )
    override fun observeMessages(roomId: String): kotlinx.coroutines.flow.Flow<List<com.armorclaw.shared.domain.model.Message>> = kotlinx.coroutines.flow.flowOf(emptyList())
    override suspend fun sendTextMessage(roomId: String, text: String, html: String?): Result<String> = Result.success("\$event")
    override suspend fun sendEmote(roomId: String, text: String): Result<String> = Result.success("\$event")
    override suspend fun sendReply(roomId: String, replyToEventId: String, text: String): Result<String> = Result.success("\$event")
    override suspend fun editMessage(roomId: String, eventId: String, newText: String): Result<String> = Result.success("\$event")
    override suspend fun redactMessage(roomId: String, eventId: String, reason: String?): Result<Unit> = Result.success(Unit)
    override suspend fun sendReaction(roomId: String, eventId: String, key: String): Result<String> = Result.success("\$event")
    override fun observeEvents() = _events.asSharedFlow()
    override fun observeRoomEvents(roomId: String) = _events.asSharedFlow().filter { it.roomId == roomId }
    override fun observeArmorClawEvents(roomId: String?) = _events.asSharedFlow().filter { ArmorClawEventType.isArmorClawEvent(it.type) }
    override suspend fun setPresence(presence: com.armorclaw.shared.domain.model.UserPresence, statusMessage: String?): Result<Unit> = Result.success(Unit)
    override suspend fun getUserPresence(userId: String): Result<com.armorclaw.shared.domain.model.UserPresence> = Result.success(com.armorclaw.shared.domain.model.UserPresence.ONLINE)
    override fun observePresence(): kotlinx.coroutines.flow.Flow<com.armorclaw.shared.platform.matrix.PresenceUpdate> = kotlinx.coroutines.flow.flowOf<com.armorclaw.shared.platform.matrix.PresenceUpdate>()
    override suspend fun sendTyping(roomId: String, typing: Boolean, timeout: Long): Result<Unit> = Result.success(Unit)
    override fun observeTyping(roomId: String): kotlinx.coroutines.flow.Flow<List<String>> = kotlinx.coroutines.flow.flowOf<List<String>>(emptyList())
    override suspend fun sendReadReceipt(roomId: String, eventId: String): Result<Unit> = Result.success(Unit)
    override suspend fun getUnreadCount(roomId: String): Result<com.armorclaw.shared.platform.matrix.UnreadCount> = Result.success(com.armorclaw.shared.platform.matrix.UnreadCount(0, 0, false))
    override suspend fun isRoomEncrypted(roomId: String): Boolean = false
    override fun getRoomEncryptionStatus(roomId: String): kotlinx.coroutines.flow.Flow<com.armorclaw.shared.platform.matrix.RoomEncryptionStatus> = kotlinx.coroutines.flow.flowOf<com.armorclaw.shared.platform.matrix.RoomEncryptionStatus>(com.armorclaw.shared.platform.matrix.RoomEncryptionStatus.Unencrypted)
    override suspend fun requestVerification(userId: String, deviceId: String?): Result<com.armorclaw.shared.platform.matrix.VerificationRequest> = Result.failure(NotImplementedError())
    override fun observeVerificationRequests(): kotlinx.coroutines.flow.Flow<com.armorclaw.shared.platform.matrix.VerificationRequest> = kotlinx.coroutines.flow.flowOf<com.armorclaw.shared.platform.matrix.VerificationRequest>()
    override suspend fun getUser(userId: String): Result<com.armorclaw.shared.domain.model.User> = Result.failure(NotImplementedError())
    override suspend fun getDisplayName(userId: String): Result<String?> = Result.success<String?>(null)
    override suspend fun setDisplayName(name: String): Result<Unit> = Result.success(Unit)
    override suspend fun getAvatarUrl(userId: String): Result<String?> = Result.success<String?>(null)
    override suspend fun setAvatar(mimeType: String, data: ByteArray): Result<Unit> = Result.success(Unit)
    override suspend fun uploadMedia(mimeType: String, data: ByteArray): Result<String> = Result.success("mxc://test")
    override suspend fun downloadMedia(mxcUrl: String): Result<ByteArray> = Result.failure(NotImplementedError())
    override fun getThumbnailUrl(mxcUrl: String, width: Int, height: Int): String? = null
    override suspend fun removePusher(pushKey: String, appId: String): Result<Unit> = Result.success(Unit)
    override suspend fun setPusher(pushKey: String, appId: String, appDisplayName: String, deviceDisplayName: String, pushGatewayUrl: String, profileTag: String?): Result<Unit> = Result.success(Unit)
}

class MockWorkflowRepository : WorkflowRepository {
    private val workflows = mutableMapOf<String, WorkflowInfo>()

    override suspend fun startWorkflow(workflowId: String, type: String, roomId: String, parameters: Map<String, String>) {
        workflows[workflowId] = WorkflowInfo(
            id = workflowId,
            type = type,
            roomId = roomId,
            status = WorkflowStatus.RUNNING,
            initiatedBy = "test",
            parameters = parameters,
            steps = emptyList(),
            result = null,
            error = null,
            createdAt = System.currentTimeMillis(),
            completedAt = null
        )
    }

    override suspend fun updateStep(workflowId: String, stepId: String, status: StepStatus, output: String?, error: String?) {}
    override suspend fun completeWorkflow(workflowId: String, success: Boolean, result: String?, error: String?) {
        workflows.remove(workflowId)
    }
    override suspend fun failWorkflow(workflowId: String, failedAtStep: String, error: String, recoverable: Boolean) {
        workflows.remove(workflowId)
    }
    override suspend fun getActiveWorkflows(roomId: String) = workflows.values.toList()
    override suspend fun getWorkflowHistory(roomId: String?, limit: Int) = workflows.values.toList()
    override suspend fun getWorkflow(workflowId: String) = workflows[workflowId]
}

class MockAgentRepository : AgentRepository {
    override suspend fun taskStarted(agentId: String, taskId: String, taskType: String, roomId: String) {}
    override suspend fun taskCompleted(agentId: String, taskId: String, success: Boolean, result: String?, error: String?) {}
    override suspend fun getAgent(agentId: String): com.armorclaw.shared.domain.repository.AgentInfo? = null
    override suspend fun getRoomAgents(roomId: String) = emptyList<com.armorclaw.shared.domain.repository.AgentInfo>()
    override suspend fun getAgentTasks(agentId: String, limit: Int) = emptyList<com.armorclaw.shared.domain.repository.AgentTaskInfo>()
    override fun observeAgentTasks(agentId: String) = kotlinx.coroutines.flow.flowOf(emptyList<com.armorclaw.shared.domain.repository.AgentTaskInfo>())
    override suspend fun getActiveRoomTasks(roomId: String) = emptyList<com.armorclaw.shared.domain.repository.AgentTaskInfo>()
    override suspend fun updateAgentCapabilities(agentId: String, capabilities: List<String>) {}
    override suspend fun recordUsage(agentId: String, taskId: String, tokensUsed: Int?, durationMs: Long) {}
}
