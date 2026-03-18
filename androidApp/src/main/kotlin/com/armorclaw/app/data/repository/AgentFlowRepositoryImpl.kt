package com.armorclaw.app.data.repository

import com.armorclaw.app.security.VaultRepository
import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.domain.model.AgentEvent as AgentFlowEvent
import com.armorclaw.shared.domain.model.AgentEventType
import com.armorclaw.shared.domain.model.AgentWorkflowState
import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.OMOIdentityData
import com.armorclaw.shared.domain.repository.AgentFlowRepository
import com.armorclaw.shared.domain.repository.VaultKey
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import com.armorclaw.shared.platform.logging.repositoryOperationSuspend
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject

private fun <T> AppResult<Result<T>>.unwrapKotlinResult(): Result<T> = when (this) {
    is AppResult.Success -> data
    is AppResult.Error -> Result.failure(error.toException())
    is AppResult.Loading -> Result.failure(IllegalStateException("Result is still loading"))
}

private fun <T> AppResult<T>.toKotlinResult(): Result<T> = when (this) {
    is AppResult.Success -> Result.success(data)
    is AppResult.Error -> Result.failure(error.toException())
    is AppResult.Loading -> Result.failure(IllegalStateException("Result is still loading"))
}

private fun <T> AppResult<T>.getOrThrow(): T = when (this) {
    is AppResult.Success -> data
    is AppResult.Error -> throw error.toException()
    is AppResult.Loading -> throw IllegalStateException("Result is still loading")
}

class AgentFlowRepositoryImpl(
    private val vaultRepository: VaultRepository,
    private val controlPlaneStore: ControlPlaneStore,
    private val json: Json = Json { ignoreUnknownKeys = true }
) : AgentFlowRepository {

    private val logger = repositoryLogger("AgentFlowRepository", LogTag.Data.AgentFlowRepository)

    // In-memory workflow state storage
    private val workflowStates = mutableMapOf<String, AgentWorkflowState>()
    private val workflowStateFlows = mutableMapOf<String, MutableStateFlow<AgentWorkflowState>>()

    // ========================================================================
    // OMO CRUD Operations - Credentials
    // ========================================================================

    override suspend fun storeOMOCredential(
        key: String,
        value: String,
        requiresBiometric: Boolean
    ): Result<VaultKey> = repositoryOperationSuspend(logger, "storeOMOCredential") {
        logger.logDebug("Storing OMO credential", mapOf("key" to key, "requiresBiometric" to requiresBiometric))
        vaultRepository.storeOMOCredential(key, value, requiresBiometric)
    }.unwrapKotlinResult()

    override suspend fun retrieveOMOCredential(key: String): Result<String> = repositoryOperationSuspend(
        logger,
        "retrieveOMOCredential"
    ) {
        logger.logDebug("Retrieving OMO credential", mapOf("key" to key))
        vaultRepository.retrieveOMOCredential(key)
    }.unwrapKotlinResult()

    override suspend fun deleteOMOCredential(
        key: String,
        requiresBiometric: Boolean
    ): Result<Unit> = repositoryOperationSuspend(logger, "deleteOMOCredential") {
        logger.logDebug("Deleting OMO credential", mapOf("key" to key, "requiresBiometric" to requiresBiometric))
        vaultRepository.deleteOMOCredential(key, requiresBiometric)
    }.unwrapKotlinResult()

    override suspend fun listOMOCredentials(): Result<List<VaultKey>> = repositoryOperationSuspend(
        logger,
        "listOMOCredentials"
    ) {
        logger.logDebug("Listing OMO credentials")
        vaultRepository.listOMOCredentials()
    }.unwrapKotlinResult()

    // ========================================================================
    // OMO CRUD Operations - Identity
    // ========================================================================

    override suspend fun storeOMOIdentity(
        id: String,
        name: String,
        email: String,
        phone: String
    ): Result<VaultKey> = repositoryOperationSuspend(logger, "storeOMOIdentity") {
        logger.logDebug("Storing OMO identity", mapOf("id" to id, "name" to name))
        vaultRepository.storeOMOIdentity(id, name, email, phone)
    }.unwrapKotlinResult()

    override suspend fun retrieveOMOIdentity(id: String): Result<OMOIdentityData> = repositoryOperationSuspend(
        logger,
        "retrieveOMOIdentity"
    ) {
        logger.logDebug("Retrieving OMO identity", mapOf("id" to id))
        vaultRepository.retrieveOMOIdentity(id)
    }.unwrapKotlinResult()

    override suspend fun deleteOMOIdentity(
        id: String,
        requiresBiometric: Boolean
    ): Result<Unit> = repositoryOperationSuspend(logger, "deleteOMOIdentity") {
        logger.logDebug("Deleting OMO identity", mapOf("id" to id, "requiresBiometric" to requiresBiometric))
        vaultRepository.deleteOMOIdentity(id, requiresBiometric)
    }.unwrapKotlinResult()

    override suspend fun listOMOIdentities(): Result<List<VaultKey>> = repositoryOperationSuspend(
        logger,
        "listOMOIdentities"
    ) {
        logger.logDebug("Listing OMO identities")
        vaultRepository.listOMOIdentities()
    }.unwrapKotlinResult()

    // ========================================================================
    // OMO CRUD Operations - Settings
    // ========================================================================

    override suspend fun storeOMOSetting(
        key: String,
        value: String
    ): Result<VaultKey> = repositoryOperationSuspend(logger, "storeOMOSetting") {
        logger.logDebug("Storing OMO setting", mapOf("key" to key))
        vaultRepository.storeOMOSetting(key, value)
    }.unwrapKotlinResult()

    override suspend fun retrieveOMOSetting(key: String): Result<String> = repositoryOperationSuspend(
        logger,
        "retrieveOMOSetting"
    ) {
        logger.logDebug("Retrieving OMO setting", mapOf("key" to key))
        vaultRepository.retrieveOMOSetting(key)
    }.unwrapKotlinResult()

    override suspend fun deleteOMOSetting(key: String): Result<Unit> = repositoryOperationSuspend(
        logger,
        "deleteOMOSetting"
    ) {
        logger.logDebug("Deleting OMO setting", mapOf("key" to key))
        vaultRepository.deleteOMOSetting(key)
    }.unwrapKotlinResult()

    override suspend fun listOMOSettings(): Result<List<VaultKey>> = repositoryOperationSuspend(
        logger,
        "listOMOSettings"
    ) {
        logger.logDebug("Listing OMO settings")
        vaultRepository.listOMOSettings()
    }.unwrapKotlinResult()

    // ========================================================================
    // OMO CRUD Operations - Tokens
    // ========================================================================

    override suspend fun storeOMOToken(
        key: String,
        value: String
    ): Result<VaultKey> = repositoryOperationSuspend(logger, "storeOMOToken") {
        logger.logDebug("Storing OMO token", mapOf("key" to key))
        vaultRepository.storeOMOToken(key, value)
    }.unwrapKotlinResult()

    override suspend fun retrieveOMOToken(key: String): Result<String> = repositoryOperationSuspend(
        logger,
        "retrieveOMOToken"
    ) {
        logger.logDebug("Retrieving OMO token", mapOf("key" to key))
        vaultRepository.retrieveOMOToken(key)
    }.unwrapKotlinResult()

    override suspend fun deleteOMOToken(key: String): Result<Unit> = repositoryOperationSuspend(
        logger,
        "deleteOMOToken"
    ) {
        logger.logDebug("Deleting OMO token", mapOf("key" to key))
        vaultRepository.deleteOMOToken(key)
    }.unwrapKotlinResult()

    override suspend fun listOMOTokens(): Result<List<VaultKey>> = repositoryOperationSuspend(
        logger,
        "listOMOTokens"
    ) {
        logger.logDebug("Listing OMO tokens")
        vaultRepository.listOMOTokens()
    }.unwrapKotlinResult()

    // ========================================================================
    // OMO CRUD Operations - Workspace
    // ========================================================================

    override suspend fun storeOMOWorkspace(
        key: String,
        value: String
    ): Result<VaultKey> = repositoryOperationSuspend(logger, "storeOMOWorkspace") {
        logger.logDebug("Storing OMO workspace", mapOf("key" to key))
        vaultRepository.storeOMOWorkspace(key, value)
    }.unwrapKotlinResult()

    override suspend fun retrieveOMOWorkspace(key: String): Result<String> = repositoryOperationSuspend(
        logger,
        "retrieveOMOWorkspace"
    ) {
        logger.logDebug("Retrieving OMO workspace", mapOf("key" to key))
        vaultRepository.retrieveOMOWorkspace(key)
    }.unwrapKotlinResult()

    override suspend fun deleteOMOWorkspace(key: String): Result<Unit> = repositoryOperationSuspend(
        logger,
        "deleteOMOWorkspace"
    ) {
        logger.logDebug("Deleting OMO workspace", mapOf("key" to key))
        vaultRepository.deleteOMOWorkspace(key)
    }.unwrapKotlinResult()

    override suspend fun listOMOWorkspaces(): Result<List<VaultKey>> = repositoryOperationSuspend(
        logger,
        "listOMOWorkspaces"
    ) {
        logger.logDebug("Listing OMO workspaces")
        vaultRepository.listOMOWorkspaces()
    }.unwrapKotlinResult()

    // ========================================================================
    // OMO CRUD Operations - Tasks
    // ========================================================================

    override suspend fun storeOMOTask(
        key: String,
        value: String
    ): Result<VaultKey> = repositoryOperationSuspend(logger, "storeOMOTask") {
        logger.logDebug("Storing OMO task", mapOf("key" to key))
        vaultRepository.storeOMOTask(key, value)
    }.unwrapKotlinResult()

    override suspend fun retrieveOMOTask(key: String): Result<String> = repositoryOperationSuspend(
        logger,
        "retrieveOMOTask"
    ) {
        logger.logDebug("Retrieving OMO task", mapOf("key" to key))
        vaultRepository.retrieveOMOTask(key)
    }.unwrapKotlinResult()

    override suspend fun deleteOMOTask(key: String): Result<Unit> = repositoryOperationSuspend(
        logger,
        "deleteOMOTask"
    ) {
        logger.logDebug("Deleting OMO task", mapOf("key" to key))
        vaultRepository.deleteOMOTask(key)
    }.unwrapKotlinResult()

    override suspend fun listOMOTasks(): Result<List<VaultKey>> = repositoryOperationSuspend(
        logger,
        "listOMOTasks"
    ) {
        logger.logDebug("Listing OMO tasks")
        vaultRepository.listOMOTasks()
    }.unwrapKotlinResult()

    // ========================================================================
    // Workflow State Management
    // ========================================================================

    override suspend fun getWorkflowState(workflowId: String): AgentWorkflowState? = repositoryOperationSuspend(
        logger,
        "getWorkflowState"
    ) {
        logger.logDebug("Getting workflow state", mapOf("workflowId" to workflowId))
        workflowStates[workflowId]
    }.getOrThrow()

    override suspend fun updateWorkflowState(state: AgentWorkflowState): Result<Unit> = repositoryOperationSuspend(
        logger,
        "updateWorkflowState"
    ) {
        logger.logDebug(
            "Updating workflow state",
            mapOf(
                "workflowId" to state.workflowId,
                "agentId" to state.agentId,
                "taskId" to state.taskId
            )
        )

        workflowStates[state.workflowId] = state

        val flow = workflowStateFlows.getOrPut(state.workflowId) {
            MutableStateFlow(state)
        }
        flow.value = state

        Unit
    }.toKotlinResult()

    override fun observeWorkflowState(workflowId: String): Flow<AgentWorkflowState> {
        logger.logDebug("Observing workflow state", mapOf("workflowId" to workflowId))

        return workflowStateFlows.getOrPut(workflowId) {
            val currentState = workflowStates[workflowId]
            MutableStateFlow(currentState ?: AgentWorkflowState.Initiated(
                workflowId = workflowId,
                agentId = "",
                taskId = "",
                roomId = "",
                workflowType = "unknown",
                initiatedBy = "system",
                timestamp = System.currentTimeMillis()
            ))
        }.asStateFlow()
    }

    // ========================================================================
    // Event Management
    // ========================================================================

    override suspend fun recordEvent(event: AgentFlowEvent): Result<Unit> = repositoryOperationSuspend(
        logger,
        "recordEvent"
    ) {
        logger.logDebug(
            "Recording agent event",
            mapOf(
                "eventId" to event.eventId,
                "workflowId" to event.workflowId,
                "agentId" to event.agentId,
                "type" to event.type.name
            )
        )

        val activityEvent = when (event.type) {
            AgentEventType.AGENT_THINKING -> {
                com.armorclaw.shared.domain.model.ActivityEvent.Success(
                    id = event.eventId,
                    agentId = event.agentId,
                    agentName = event.agentId,
                    roomId = event.workflowId,
                    timestamp = event.timestamp,
                    taskDescription = "Agent thinking",
                    result = event.data.toString()
                )
            }

            AgentEventType.AGENT_ERROR -> {
                val errorMessage = event.data["message"]?.toString() ?: "Unknown error"
                com.armorclaw.shared.domain.model.ActivityEvent.Error(
                    id = event.eventId,
                    agentId = event.agentId,
                    agentName = event.agentId,
                    roomId = event.workflowId,
                    timestamp = event.timestamp,
                    errorMessage = errorMessage,
                    errorType = "AGENT_ERROR",
                    recoverable = true
                )
            }

            else -> {
                com.armorclaw.shared.domain.model.ActivityEvent.Success(
                    id = event.eventId,
                    agentId = event.agentId,
                    agentName = event.agentId,
                    roomId = event.workflowId,
                    timestamp = event.timestamp,
                    taskDescription = event.type.toDisplayString(),
                    result = event.data.toString()
                )
            }
        }

        controlPlaneStore.addActivityEvent(activityEvent)
        Unit
    }.toKotlinResult()

    override suspend fun getWorkflowEvents(
        workflowId: String,
        limit: Int
    ): Result<List<AgentFlowEvent>> = repositoryOperationSuspend(logger, "getWorkflowEvents") {
        logger.logDebug(
            "Getting workflow events",
            mapOf("workflowId" to workflowId, "limit" to limit)
        )

        emptyList<AgentFlowEvent>()
    }.toKotlinResult()

    override suspend fun getAgentEvents(
        agentId: String,
        limit: Int
    ): Result<List<AgentFlowEvent>> = repositoryOperationSuspend(logger, "getAgentEvents") {
        logger.logDebug("Getting agent events", mapOf("agentId" to agentId, "limit" to limit))

        emptyList<AgentFlowEvent>()
    }.toKotlinResult()
}
