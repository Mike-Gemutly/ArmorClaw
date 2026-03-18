package com.armorclaw.shared.domain.features

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.OperationContext
import kotlinx.coroutines.flow.Flow

/**
 * Service interface for workflow validation
 *
 * Provides workflow state validation, rule checking, and
 * execution guardrails for agent workflows.
 *
 * TODO: Implement workflow rule engine
 * TODO: Add custom validation rules
 * TODO: Integrate with agent execution
 */
interface WorkflowValidator {

    /**
     * Validate workflow execution preconditions
     * @param workflowId The workflow ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun validatePreconditions(
        workflowId: String,
        context: OperationContext? = null
    ): AppResult<ValidationResult>

    /**
     * Validate workflow execution postconditions
     * @param workflowId The workflow ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun validatePostconditions(
        workflowId: String,
        context: OperationContext? = null
    ): AppResult<ValidationResult>

    /**
     * Check if workflow can be executed
     * @param workflowId The workflow ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun canExecuteWorkflow(
        workflowId: String,
        context: OperationContext? = null
    ): AppResult<Boolean>

    /**
     * Get workflow validation rules
     * @param workflowId The workflow ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun getValidationRules(
        workflowId: String,
        context: OperationContext? = null
    ): AppResult<List<ValidationRule>>

    /**
     * Add custom validation rule
     * @param workflowId The workflow ID
     * @param rule The validation rule to add
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun addValidationRule(
        workflowId: String,
        rule: ValidationRule,
        context: OperationContext? = null
    ): AppResult<Unit>

    /**
     * Remove validation rule
     * @param workflowId The workflow ID
     * @param ruleId The rule ID to remove
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun removeValidationRule(
        workflowId: String,
        ruleId: String,
        context: OperationContext? = null
    ): AppResult<Unit>

    /**
     * Observe workflow validation status (reactive)
     * @param workflowId The workflow ID
     */
    fun observeValidationStatus(workflowId: String): Flow<ValidationStatus>
}

/**
 * Validation result
 *
 * TODO: Add detailed error messages
 * TODO: Add error severity levels
 */
@kotlinx.serialization.Serializable
data class ValidationResult(
    val isValid: Boolean,
    val errors: List<ValidationError> = emptyList(),
    val warnings: List<ValidationWarning> = emptyList()
)

/**
 * Validation error
 */
@kotlinx.serialization.Serializable
data class ValidationError(
    val code: String,
    val message: String,
    val field: String? = null
)

/**
 * Validation warning
 */
@kotlinx.serialization.Serializable
data class ValidationWarning(
    val code: String,
    val message: String,
    val field: String? = null
)

/**
 * Validation rule definition
 *
 * TODO: Add rule priority
 * TODO: Add rule condition DSL
 */
@kotlinx.serialization.Serializable
data class ValidationRule(
    val id: String,
    val name: String,
    val description: String,
    val type: ValidationRuleType,
    val condition: String
)

/**
 * Validation rule types
 */
@kotlinx.serialization.Serializable
enum class ValidationRuleType {
    PRECONDITION,
    POSTCONDITION,
    INVARIANT,
    CUSTOM
}

/**
 * Workflow validation status
 *
 * TODO: Add last validation timestamp
 * TODO: Add validation history
 */
@kotlinx.serialization.Serializable
data class ValidationStatus(
    val workflowId: String,
    val status: WorkflowState,
    val isValid: Boolean = false
)

/**
 * Workflow execution state
 */
@kotlinx.serialization.Serializable
enum class WorkflowState {
    PENDING,
    VALIDATING,
    READY,
    RUNNING,
    COMPLETED,
    FAILED,
    CANCELLED
}
