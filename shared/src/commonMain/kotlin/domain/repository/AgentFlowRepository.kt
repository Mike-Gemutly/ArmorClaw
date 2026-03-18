package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.AgentEvent
import com.armorclaw.shared.domain.model.AgentWorkflowState
import com.armorclaw.shared.domain.model.OMOIdentityData
import kotlinx.coroutines.flow.Flow

/**
 * Agent Flow Repository
 *
 * Manages agent workflow state, events, and OMO PII access.
 * Provides a unified interface for agent operations that require
 * biometric-protected PII access through the Vault.
 *
 * ## Architecture
 * ```
 * AgentFlowRepository (shared interface)
 *         ↓ (implemented by)
 * AgentFlowRepositoryImpl (androidApp)
 *         ↓ (delegates to)
 * VaultRepository (androidApp)
 *         ↓ (delegates to)
 * SQLCipherProvider (database)
 * ```
 *
 * ## Security Model
 * - OMO_CREDENTIALS: API keys, tokens (OMO_CRITICAL sensitivity)
 * - OMO_IDENTITY: User identity info (OMO_LOW sensitivity)
 * - OMO_SETTINGS: Agent configuration (OMO_LOW sensitivity)
 * - OMO_TOKENS: Session tokens (OMO_HIGH sensitivity)
 * - OMO_WORKSPACE: Workspace data (OMO_MEDIUM sensitivity)
 * - OMO_TASKS: Task data (OMO_LOW sensitivity)
 *
 * ## Usage
 * ```kotlin
 * class AgentViewModel(
 *     private val agentFlowRepository: AgentFlowRepository
 * ) {
 *     fun startWorkflow() {
 *         viewModelScope.launch {
 *             // Store OMO credentials with biometric protection
 *             agentFlowRepository.storeOMOCredential(
 *                 key = "openai_api_key",
 *                 value = "sk-...",
 *                 requiresBiometric = true
 *             )
 *         }
 *     }
 * }
 * ```
 */
interface AgentFlowRepository {

    // ========================================================================
    // OMO CRUD Operations - Credentials
    // ========================================================================

    /**
     * Store an OMO credential (API keys, tokens, passwords for OMO services)
     *
     * @param key The credential key (e.g., "openai_api_key", "github_token")
     * @param value The credential value to store
     * @param requiresBiometric Whether biometric authentication is required
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOCredential(
        key: String,
        value: String,
        requiresBiometric: Boolean = false
    ): Result<VaultKey>

    /**
     * Retrieve an OMO credential
     *
     * @param key The credential key to retrieve
     * @return Result containing the credential value
     */
    suspend fun retrieveOMOCredential(key: String): Result<String>

    /**
     * Delete an OMO credential
     *
     * @param key The credential key to delete
     * @param requiresBiometric Whether biometric authentication is required
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOCredential(
        key: String,
        requiresBiometric: Boolean = false
    ): Result<Unit>

    /**
     * List all stored OMO credentials
     *
     * @return Result containing list of VaultKey entries for OMO credentials
     */
    suspend fun listOMOCredentials(): Result<List<VaultKey>>

    // ========================================================================
    // OMO CRUD Operations - Identity
    // ========================================================================

    /**
     * Store an OMO identity (User identity information for OMO agents)
     *
     * @param id Unique identifier for the identity
     * @param name Display name
     * @param email Email address
     * @param phone Phone number
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOIdentity(
        id: String,
        name: String,
        email: String,
        phone: String
    ): Result<VaultKey>

    /**
     * Retrieve an OMO identity
     *
     * @param id Unique identifier for the identity
     * @return Result containing the identity data (name, email, phone)
     */
    suspend fun retrieveOMOIdentity(id: String): Result<OMOIdentityData>

    /**
     * Delete an OMO identity
     *
     * @param id Unique identifier for the identity
     * @param requiresBiometric Whether biometric authentication is required
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOIdentity(
        id: String,
        requiresBiometric: Boolean = false
    ): Result<Unit>

    /**
     * List all stored OMO identities
     *
     * @return Result containing list of VaultKey entries for OMO identities
     */
    suspend fun listOMOIdentities(): Result<List<VaultKey>>

    // ========================================================================
    // OMO CRUD Operations - Settings
    // ========================================================================

    /**
     * Store an OMO setting (Agent configuration and settings)
     *
     * @param key The setting key (e.g., "model_temperature", "max_tokens")
     * @param value The setting value to store
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOSetting(
        key: String,
        value: String
    ): Result<VaultKey>

    /**
     * Retrieve an OMO setting
     *
     * @param key The setting key to retrieve
     * @return Result containing the setting value
     */
    suspend fun retrieveOMOSetting(key: String): Result<String>

    /**
     * Delete an OMO setting
     *
     * @param key The setting key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOSetting(key: String): Result<Unit>

    /**
     * List all stored OMO settings
     *
     * @return Result containing list of VaultKey entries for OMO settings
     */
    suspend fun listOMOSettings(): Result<List<VaultKey>>

    // ========================================================================
    // OMO CRUD Operations - Tokens
    // ========================================================================

    /**
     * Store an OMO token (Session tokens for OMO authentication)
     *
     * @param key The token key (e.g., "session_token", "refresh_token")
     * @param value The token value to store
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOToken(
        key: String,
        value: String
    ): Result<VaultKey>

    /**
     * Retrieve an OMO token
     *
     * @param key The token key to retrieve
     * @return Result containing the token value
     */
    suspend fun retrieveOMOToken(key: String): Result<String>

    /**
     * Delete an OMO token
     *
     * @param key The token key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOToken(key: String): Result<Unit>

    /**
     * List all stored OMO tokens
     *
     * @return Result containing list of VaultKey entries for OMO tokens
     */
    suspend fun listOMOTokens(): Result<List<VaultKey>>

    // ========================================================================
    // OMO CRUD Operations - Workspace
    // ========================================================================

    /**
     * Store an OMO workspace (Workspace and project data)
     *
     * @param key The workspace key (e.g., "default_workspace", "project_alpha")
     * @param value The workspace data (JSON or structured string)
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOWorkspace(
        key: String,
        value: String
    ): Result<VaultKey>

    /**
     * Retrieve an OMO workspace
     *
     * @param key The workspace key to retrieve
     * @return Result containing the workspace data
     */
    suspend fun retrieveOMOWorkspace(key: String): Result<String>

    /**
     * Delete an OMO workspace
     *
     * @param key The workspace key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOWorkspace(key: String): Result<Unit>

    /**
     * List all stored OMO workspaces
     *
     * @return Result containing list of VaultKey entries for OMO workspaces
     */
    suspend fun listOMOWorkspaces(): Result<List<VaultKey>>

    // ========================================================================
    // OMO CRUD Operations - Tasks
    // ========================================================================

    /**
     * Store an OMO task (Task data and metadata)
     *
     * @param key The task key (e.g., "task_123", "todo_list")
     * @param value The task data (JSON or structured string)
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOTask(
        key: String,
        value: String
    ): Result<VaultKey>

    /**
     * Retrieve an OMO task
     *
     * @param key The task key to retrieve
     * @return Result containing the task data
     */
    suspend fun retrieveOMOTask(key: String): Result<String>

    /**
     * Delete an OMO task
     *
     * @param key The task key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOTask(key: String): Result<Unit>

    /**
     * List all stored OMO tasks
     *
     * @return Result containing list of VaultKey entries for OMO tasks
     */
    suspend fun listOMOTasks(): Result<List<VaultKey>>

    // ========================================================================
    // Workflow State Management
    // ========================================================================

    /**
     * Get the current state of a workflow
     *
     * @param workflowId The workflow ID
     * @return The current workflow state, or null if not found
     */
    suspend fun getWorkflowState(workflowId: String): AgentWorkflowState?

    /**
     * Update the state of a workflow
     *
     * @param state The new workflow state
     * @return Result indicating success or failure
     */
    suspend fun updateWorkflowState(state: AgentWorkflowState): Result<Unit>

    /**
     * Observe workflow state changes
     *
     * @param workflowId The workflow ID
     * @return Flow of workflow state updates
     */
    fun observeWorkflowState(workflowId: String): Flow<AgentWorkflowState>

    // ========================================================================
    // Event Management
    // ========================================================================

    /**
     * Record an agent event
     *
     * @param event The event to record
     * @return Result indicating success or failure
     */
    suspend fun recordEvent(event: AgentEvent): Result<Unit>

    /**
     * Get events for a workflow
     *
     * @param workflowId The workflow ID
     * @param limit Maximum number of events to return
     * @return Result containing list of events
     */
    suspend fun getWorkflowEvents(
        workflowId: String,
        limit: Int = 100
    ): Result<List<AgentEvent>>

    /**
     * Get events for an agent
     *
     * @param agentId The agent ID
     * @param limit Maximum number of events to return
     * @return Result containing list of events
     */
    suspend fun getAgentEvents(
        agentId: String,
        limit: Int = 100
    ): Result<List<AgentEvent>>
}
