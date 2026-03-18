package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

/**
 * Agent Workflow State
 *
 * Represents the current state of an agent workflow execution.
 * Tracks workflow progress, agent status, and PII access requirements.
 *
 * ## Workflow Lifecycle
 * 1. INITIATED - Workflow created, waiting to start
 * 2. RUNNING - Workflow actively executing
 * 3. PAUSED - Workflow temporarily paused
 * 4. WAITING_FOR_PII - Waiting for user to approve PII access
 * 5. WAITING_FOR_BIOMETRIC - Waiting for biometric authentication
 * 6. COMPLETED - Workflow finished successfully
 * 7. FAILED - Workflow encountered unrecoverable error
 * 8. CANCELLED - Workflow cancelled by user
 *
 * ## Usage
 * ```kotlin
 * val state = AgentWorkflowState.Running(
 *     workflowId = "workflow_123",
 *     agentId = "agent_browse_001",
 *     taskId = "task_456",
 *     currentStep = "Navigate to checkout",
 *     stepIndex = 2,
 *     totalSteps = 5,
 *     roomId = "!room:matrix.org",
 *     timestamp = System.currentTimeMillis()
 * )
 *
 * // Check if workflow is waiting for PII
 * if (state is AgentWorkflowState.WaitingForPii) {
 *     // Show PII approval UI
 * }
 * ```
 */
@Serializable
sealed class AgentWorkflowState {
    abstract val workflowId: String
    abstract val agentId: String
    abstract val taskId: String
    abstract val roomId: String
    abstract val timestamp: Long

    /**
     * Workflow has been initiated but not yet started
     */
    @Serializable
    data class Initiated(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val workflowType: String,
        val initiatedBy: String,
        override val timestamp: Long
    ) : AgentWorkflowState()

    /**
     * Workflow is actively running
     */
    @Serializable
    data class Running(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val currentStep: String,
        val stepIndex: Int,
        val totalSteps: Int,
        val progress: Float = 0f,
        override val timestamp: Long
    ) : AgentWorkflowState()

    /**
     * Workflow has been paused by user or system
     */
    @Serializable
    data class Paused(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val currentStep: String,
        val reason: String,
        override val timestamp: Long
    ) : AgentWorkflowState()

    /**
     * Workflow is waiting for user to approve PII access
     */
    @Serializable
    data class WaitingForPii(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val piiRequest: PiiAccessRequest,
        val currentStep: String,
        override val timestamp: Long
    ) : AgentWorkflowState()

    /**
     * Workflow is waiting for biometric authentication
     */
    @Serializable
    data class WaitingForBiometric(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val reason: String,
        val currentStep: String,
        override val timestamp: Long
    ) : AgentWorkflowState()

    /**
     * Workflow completed successfully
     */
    @Serializable
    data class Completed(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val result: String,
        val duration: Long,
        override val timestamp: Long
    ) : AgentWorkflowState()

    /**
     * Workflow encountered an unrecoverable error
     */
    @Serializable
    data class Failed(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val error: String,
        val failedAtStep: String,
        val recoverable: Boolean = false,
        override val timestamp: Long
    ) : AgentWorkflowState()

    /**
     * Workflow was cancelled by user
     */
    @Serializable
    data class Cancelled(
        override val workflowId: String,
        override val agentId: String,
        override val taskId: String,
        override val roomId: String,
        val reason: String,
        override val timestamp: Long
    ) : AgentWorkflowState()
}

/**
 * Workflow status enum for easier status checks
 */
@Serializable
enum class WorkflowStatus {
    INITIATED,
    RUNNING,
    PAUSED,
    WAITING_FOR_PII,
    WAITING_FOR_BIOMETRIC,
    COMPLETED,
    FAILED,
    CANCELLED;

    /**
     * Check if workflow is in an active state
     */
    fun isActive(): Boolean {
        return this == INITIATED || this == RUNNING || this == WAITING_FOR_PII || this == WAITING_FOR_BIOMETRIC
    }

    /**
     * Check if workflow is waiting for user input
     */
    fun isWaitingForInput(): Boolean {
        return this == WAITING_FOR_PII || this == WAITING_FOR_BIOMETRIC
    }

    /**
     * Check if workflow is in a terminal state
     */
    fun isTerminal(): Boolean {
        return this == COMPLETED || this == FAILED || this == CANCELLED
    }
}
