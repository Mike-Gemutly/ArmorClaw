package com.armorclaw.shared.domain.repository

import kotlinx.coroutines.flow.Flow

// ========================================
// Phase 1: Cold Vault - Security Domain Models
// ========================================

/**
 * Vault Key - represents a PII field in the Cold Vault
 */
data class VaultKey(
    val id: String,
    val fieldName: String,
    val displayName: String,
    val category: VaultKeyCategory,
    val sensitivity: VaultKeySensitivity,
    val lastAccessed: Long?,
    val accessCount: Int = 0
)

enum class VaultKeyCategory {
    PERSONAL,      // Name, DOB, SSN
    FINANCIAL,     // Credit card, bank account
    CONTACT,       // Email, phone, address
    AUTHENTICATION,// Passwords, tokens
    MEDICAL,       // Health records
    OTHER,
    // OMO Categories
    OMO_CREDENTIALS, // API keys, tokens, passwords for OMO services
    OMO_IDENTITY,    // User identity information for OMO agents
    OMO_SETTINGS,    // Agent configuration and settings
    OMO_TOKENS,      // Session tokens for OMO authentication
    OMO_WORKSPACE,   // Workspace and project data
    OMO_TASKS        // Task data and metadata
}

enum class VaultKeySensitivity {
    LOW,           // Name, city
    MEDIUM,        // Email, phone
    HIGH,          // Address, DOB
    CRITICAL,      // SSN, financial, medical
    // OMO Sensitivity Levels
    OMO_LOW,       // OMO agent configuration and preferences
    OMO_MEDIUM,    // OMO workspace and project data
    OMO_HIGH,      // OMO session tokens and API keys
    OMO_CRITICAL   // OMO authentication credentials and secrets
}

/**
 * Shadow Placeholder - represents a PII placeholder in transit
 */
data class ShadowPlaceholder(
    val keyId: String,
    val placeholder: String,      // e.g., "{{VAULT:ssn:a1b2c3}}"
    val hash: String,             // SHA-256 of the actual value
    val createdAt: Long,
    val expiresAt: Long
)

// ========================================
// Agent Repository Interface
// ========================================

/**
 * Repository for managing AI agent state
 *
 * Agents are AI assistants that perform tasks in Matrix rooms.
 * This repository tracks agent status, tasks, and capabilities.
 */
interface AgentRepository {

    /**
     * Record that an agent task started
     */
    suspend fun taskStarted(
        agentId: String,
        taskId: String,
        taskType: String,
        roomId: String
    )

    /**
     * Record that an agent task completed
     */
    suspend fun taskCompleted(
        agentId: String,
        taskId: String,
        success: Boolean,
        result: String? = null,
        error: String? = null
    )

    /**
     * Get an agent by ID
     */
    suspend fun getAgent(agentId: String): AgentInfo?

    /**
     * Get all agents available in a room
     */
    suspend fun getRoomAgents(roomId: String): List<AgentInfo>

    /**
     * Get agent tasks (active or recent)
     */
    suspend fun getAgentTasks(
        agentId: String,
        limit: Int = 20
    ): List<AgentTaskInfo>

    /**
     * Observe agent tasks in real-time
     */
    fun observeAgentTasks(agentId: String): Flow<List<AgentTaskInfo>>

    /**
     * Get active tasks for a room
     */
    suspend fun getActiveRoomTasks(roomId: String): List<AgentTaskInfo>

    /**
     * Update agent capabilities
     */
    suspend fun updateAgentCapabilities(
        agentId: String,
        capabilities: List<String>
    )

    /**
     * Record agent usage (for budget tracking)
     */
    suspend fun recordUsage(
        agentId: String,
        taskId: String,
        tokensUsed: Int? = null,
        durationMs: Long
    )
}

/**
 * Agent information
 */
data class AgentInfo(
    val id: String,
    val name: String,
    val description: String?,
    val avatarUrl: String?,
    val capabilities: List<String>,
    val status: AgentStatus,
    val lastActiveAt: Long?
)

/**
 * Agent task information
 */
data class AgentTaskInfo(
    val id: String,
    val agentId: String,
    val agentName: String,
    val type: String,
    val roomId: String,
    val status: AgentTaskStatus,
    val result: String?,
    val error: String?,
    val createdAt: Long,
    val completedAt: Long?
)

enum class AgentStatus {
    ONLINE,
    BUSY,
    OFFLINE,
    ERROR
}

enum class AgentTaskStatus {
    PENDING,
    RUNNING,
    COMPLETED,
    FAILED,
    CANCELLED
}
