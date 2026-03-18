package com.armorclaw.shared.platform.browser

import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.platform.bridge.BridgeRpcClient
import com.armorclaw.shared.platform.bridge.RpcResult
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import com.armorclaw.shared.platform.matrix.MatrixSyncEvent
import com.armorclaw.shared.platform.matrix.MatrixSyncManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull

/**
 * Browser Command Handler
 *
 * Processes browser automation events from Matrix and coordinates with the Bridge.
 *
 * ## Architecture
 * ```
 * MatrixSyncManager.events
 *        │
 *        ▼
 * BrowserCommandHandler
 *        │
 *        ├── BrowserCommandEvent → Enqueue job via RPC
 *        ├── BrowserResponseEvent → Update UI state
 *        ├── BrowserStatusEvent → Update browser status
 *        ├── AgentStatusEvent → Update agent status in ControlPlaneStore
 *        └── PiiResponseEvent → Handle PII approval/denial
 * ```
 *
 * ## Event Flow
 * 1. Agent sends command event (e.g., com.armorclaw.browser.navigate)
 * 2. Handler receives event via Matrix sync
 * 3. Handler enqueues job via Bridge RPC (browser.enqueue)
 * 4. Bridge executes job, sends status updates
 * 5. Handler receives status updates, updates UI state
 */
class BrowserCommandHandler(
    private val rpcClient: BridgeRpcClient,
    private val controlPlaneStore: ControlPlaneStore,
    private val json: Json = Json { ignoreUnknownKeys = true; isLenient = true }
) {
    private val logger = repositoryLogger("BrowserCommandHandler", LogTag.Network.BridgeRpc)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // Browser status state
    private val _browserStatus = MutableStateFlow<BrowserStatus?>(null)
    val browserStatus: StateFlow<BrowserStatus?> = _browserStatus.asStateFlow()

    // Active job tracking
    private val _activeJob = MutableStateFlow<BrowserJob?>(null)
    val activeJob: StateFlow<BrowserJob?> = _activeJob.asStateFlow()

    // Job history
    private val _jobHistory = MutableStateFlow<List<BrowserJob>>(emptyList())
    val jobHistory: StateFlow<List<BrowserJob>> = _jobHistory.asStateFlow()

    /**
     * Start listening to Matrix sync events
     */
    fun start(syncManager: MatrixSyncManager) {
        logger.logOperationStart("start", mapOf("message" to "Starting browser command handler"))

        scope.launch {
            syncManager.events.collect { event ->
                try {
                    handleSyncEvent(event)
                } catch (e: Exception) {
                    logger.logOperationError("handleSyncEvent", e)
                }
            }
        }
    }

    /**
     * Handle incoming Matrix sync event
     */
    private suspend fun handleSyncEvent(event: MatrixSyncEvent) {
        when (event) {
            is MatrixSyncEvent.BrowserCommandEvent -> handleBrowserCommand(event)
            is MatrixSyncEvent.BrowserResponseEvent -> handleBrowserResponse(event)
            is MatrixSyncEvent.BrowserStatusEvent -> handleBrowserStatus(event)
            is MatrixSyncEvent.AgentStatusEvent -> handleAgentStatus(event)
            is MatrixSyncEvent.PiiResponseEvent -> handlePiiResponse(event)
            else -> { /* Ignore other events */ }
        }
    }

    /**
     * Handle browser command event
     *
     * When an agent sends a browser command, enqueue it via RPC
     */
    private suspend fun handleBrowserCommand(event: MatrixSyncEvent.BrowserCommandEvent) {
        logger.logDebug("Received browser command", mapOf(
            "room_id" to event.roomId,
            "event_type" to event.eventType
        ))

        // Parse command from event content
        val content = event.event.content ?: return
        val commandType = event.eventType.removePrefix("com.armorclaw.browser.")

        // Build browser command - convert JsonObject to Map<String, String>
        val params = content.mapValues { (_, value) ->
            when {
                value.jsonPrimitive.isString -> value.jsonPrimitive.content
                else -> value.toString()
            }
        }

        val command = BrowserCommand(
            type = commandType,
            params = params
        )

        // Extract URL if navigate command
        val url = params["url"] ?: ""

        // Get agent ID from sender
        val agentId = event.event.sender ?: "unknown"

        // Enqueue job via RPC
        when (val result = rpcClient.browserEnqueue(
            agentId = agentId,
            roomId = event.roomId,
            url = url,
            commands = listOf(command),
            priority = BrowserJobPriority.NORMAL
        )) {
            is RpcResult.Success -> {
                logger.logInfo("Job enqueued", mapOf("job_id" to result.data.jobId))
                // Fetch job details
                fetchJobDetails(result.data.jobId)
            }
            is RpcResult.Error -> {
                logger.logOperationError("browserEnqueue", Exception(result.message))
            }
        }
    }

    /**
     * Handle browser response event
     *
     * Bridge sends responses when commands complete
     */
    private suspend fun handleBrowserResponse(event: MatrixSyncEvent.BrowserResponseEvent) {
        logger.logDebug("Received browser response", mapOf("room_id" to event.roomId))

        val content = event.event.content ?: return

        try {
            val statusStr = content["status"]?.jsonPrimitive?.content ?: return
            val status = if (statusStr == "success") BrowserResponseStatus.SUCCESS else BrowserResponseStatus.ERROR
            val command = content["command"]?.jsonPrimitive?.content ?: ""

            when (status) {
                BrowserResponseStatus.SUCCESS -> {
                    logger.logInfo("Browser command succeeded", mapOf("command" to command))
                }
                BrowserResponseStatus.ERROR -> {
                    val errorMsg = content["error"]?.jsonObject?.get("message")?.jsonPrimitive?.content
                    logger.logWarning("Browser command failed", mapOf(
                        "command" to command,
                        "error" to errorMsg
                    ))
                }
            }
        } catch (e: Exception) {
            logger.logOperationError("handleBrowserResponse", e)
        }
    }

    /**
     * Handle browser status event
     *
     * Real-time updates from Bridge about browser state
     */
    private suspend fun handleBrowserStatus(event: MatrixSyncEvent.BrowserStatusEvent) {
        logger.logDebug("Received browser status", mapOf("room_id" to event.roomId))

        val content = event.event.content ?: return

        try {
            val sessionId = content["session_id"]?.jsonPrimitive?.content ?: return
            val url = content["url"]?.jsonPrimitive?.content
            val title = content["title"]?.jsonPrimitive?.content
            val loading = content["loading"]?.jsonPrimitive?.booleanOrNull ?: false
            val timestamp = content["timestamp"]?.jsonPrimitive?.longOrNull ?: System.currentTimeMillis()

            _browserStatus.value = BrowserStatus(
                sessionId = sessionId,
                url = url,
                title = title,
                loading = loading,
                timestamp = timestamp
            )
        } catch (e: Exception) {
            logger.logOperationError("handleBrowserStatus", e)
        }
    }

    /**
     * Handle agent status event
     *
     * Bridge sends agent status updates during job execution
     */
    private suspend fun handleAgentStatus(event: MatrixSyncEvent.AgentStatusEvent) {
        logger.logDebug("Received agent status", mapOf("room_id" to event.roomId))

        val content = event.event.content ?: return

        try {
            val agentId = content["agent_id"]?.jsonPrimitive?.content ?: return
            val statusStr = content["status"]?.jsonPrimitive?.content ?: "idle"
            val timestamp = content["timestamp"]?.jsonPrimitive?.longOrNull ?: System.currentTimeMillis()

            // Parse metadata
            val metadata = content["metadata"]?.jsonObject?.let { metaObj ->
                metaObj.mapValues { (_, value) ->
                    value.jsonPrimitive.content
                }
            }

            // Map status string to enum
            val status = when (statusStr) {
                "idle" -> AgentTaskStatus.IDLE
                "browsing" -> AgentTaskStatus.BROWSING
                "form_filling" -> AgentTaskStatus.FORM_FILLING
                "processing_payment" -> AgentTaskStatus.PROCESSING_PAYMENT
                "awaiting_captcha" -> AgentTaskStatus.AWAITING_CAPTCHA
                "awaiting_2fa" -> AgentTaskStatus.AWAITING_2FA
                "awaiting_approval" -> AgentTaskStatus.AWAITING_APPROVAL
                "error" -> AgentTaskStatus.ERROR
                "complete" -> AgentTaskStatus.COMPLETE
                else -> AgentTaskStatus.IDLE
            }

            // Convert to domain model and update ControlPlaneStore
            val agentStatus = AgentTaskStatusEvent(
                agentId = agentId,
                status = status,
                timestamp = timestamp,
                metadata = metadata
            )
            controlPlaneStore.processStatusEvent(agentStatus)
        } catch (e: Exception) {
            logger.logOperationError("handleAgentStatus", e)
        }
    }

    /**
     * Handle PII response event
     *
     * Bridge sends PII response when user approves/denies access
     */
    private suspend fun handlePiiResponse(event: MatrixSyncEvent.PiiResponseEvent) {
        logger.logDebug("Received PII response", mapOf("room_id" to event.roomId))

        val content = event.event.content ?: return

        // PII response handling is done via ControlPlaneStore
        // The response contains approved fields that should be sent to Bridge
        // This event is primarily for confirmation logging
        val approved = content["approved"]?.jsonPrimitive?.booleanOrNull ?: false
        val jobId = content["job_id"]?.jsonPrimitive?.content

        logger.logInfo("PII response received", mapOf(
            "job_id" to jobId,
            "approved" to approved
        ))
    }

    /**
     * Fetch job details and update state
     */
    private suspend fun fetchJobDetails(jobId: String) {
        when (val result = rpcClient.browserGetJob(jobId)) {
            is RpcResult.Success -> {
                _activeJob.value = result.data.job
                // Add to history
                _jobHistory.update { history ->
                    (listOf(result.data.job) + history).take(50)
                }
            }
            is RpcResult.Error -> {
                logger.logOperationError("fetchJobDetails", Exception(result.message))
            }
        }
    }

    /**
     * Cancel the active job
     */
    suspend fun cancelActiveJob(): Boolean {
        val job = _activeJob.value ?: return false

        return when (val result = rpcClient.browserCancelJob(job.jobId)) {
            is RpcResult.Success -> {
                logger.logInfo("Job cancelled", mapOf("job_id" to job.jobId))
                _activeJob.value = null
                true
            }
            is RpcResult.Error -> {
                logger.logOperationError("cancelActiveJob", Exception(result.message))
                false
            }
        }
    }

    /**
     * Retry a failed job
     */
    suspend fun retryJob(jobId: String): String? {
        return when (val result = rpcClient.browserRetryJob(jobId)) {
            is RpcResult.Success -> {
                val newJobId = result.data.newJobId ?: jobId
                logger.logInfo("Job retried", mapOf(
                    "original_job_id" to jobId,
                    "new_job_id" to newJobId
                ))
                fetchJobDetails(newJobId)
                newJobId
            }
            is RpcResult.Error -> {
                logger.logOperationError("retryJob", Exception(result.message))
                null
            }
        }
    }

    /**
     * Get queue statistics
     */
    suspend fun getQueueStats(): BrowserQueueStatsResponse? {
        return when (val result = rpcClient.browserQueueStats()) {
            is RpcResult.Success -> result.data
            is RpcResult.Error -> {
                logger.logOperationError("getQueueStats", Exception(result.message))
                null
            }
        }
    }
}

// ============================================================================
// Supporting Models
// ============================================================================

/**
 * Browser status state
 */
data class BrowserStatus(
    val sessionId: String,
    val url: String?,
    val title: String?,
    val loading: Boolean,
    val timestamp: Long
)
