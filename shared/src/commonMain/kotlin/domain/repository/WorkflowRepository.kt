package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.platform.matrix.event.StepStatus

/**
 * Repository for managing workflow state
 *
 * Workflows are multi-step processes initiated by users or agents.
 * This repository persists workflow state for recovery and history.
 */
interface WorkflowRepository {

    /**
     * Start a new workflow
     */
    suspend fun startWorkflow(
        workflowId: String,
        type: String,
        roomId: String,
        parameters: Map<String, String> = emptyMap()
    )

    /**
     * Update a workflow step
     */
    suspend fun updateStep(
        workflowId: String,
        stepId: String,
        status: StepStatus,
        output: String? = null,
        error: String? = null
    )

    /**
     * Complete a workflow
     */
    suspend fun completeWorkflow(
        workflowId: String,
        success: Boolean,
        result: String? = null,
        error: String? = null
    )

    /**
     * Mark a workflow as failed
     */
    suspend fun failWorkflow(
        workflowId: String,
        failedAtStep: String,
        error: String,
        recoverable: Boolean
    )

    /**
     * Get active workflows for a room
     */
    suspend fun getActiveWorkflows(roomId: String): List<WorkflowInfo>

    /**
     * Get workflow history
     */
    suspend fun getWorkflowHistory(
        roomId: String? = null,
        limit: Int = 50
    ): List<WorkflowInfo>

    /**
     * Get a specific workflow by ID
     */
    suspend fun getWorkflow(workflowId: String): WorkflowInfo?
}

/**
 * Workflow information
 */
data class WorkflowInfo(
    val id: String,
    val type: String,
    val roomId: String,
    val status: WorkflowStatus,
    val initiatedBy: String,
    val parameters: Map<String, String>,
    val steps: List<StepInfo>,
    val result: String?,
    val error: String?,
    val createdAt: Long,
    val completedAt: Long?
)

data class StepInfo(
    val id: String,
    val name: String,
    val index: Int,
    val status: StepStatus,
    val output: String?,
    val error: String?,
    val startedAt: Long?,
    val completedAt: Long?
)

enum class WorkflowStatus {
    RUNNING,
    COMPLETED,
    FAILED,
    CANCELLED
}
